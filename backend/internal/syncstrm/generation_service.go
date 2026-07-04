package syncstrm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/v115open"

	"gorm.io/gorm"
)

var errOldStrmCleanupFailed = errors.New("旧 STRM 清理失败")

const (
	strmGenerationWorkerInterval = 5 * time.Second
	strmGenerationWorkerBatch    = 5
)

var strmGenerationWorkerOnce sync.Once

// StrmGenerationInput 是单文件 STRM 生成输入。
type StrmGenerationInput struct {
	Task *models.StrmGenerationTask
}

// StrmGenerationResult 描述单文件 STRM 生成结果。
type StrmGenerationResult struct {
	SyncFile *models.SyncFile
	Changed  bool
}

// StrmGenerationService 复用现有 SyncStrm 能力完成单文件 STRM 后处理。
type StrmGenerationService struct {
	buildSyncer        func(*models.SyncPath, *models.Account) (*SyncStrm, error)
	compareStrm        func(*SyncStrm, *SyncFileCache) int
	processStrmFile    func(*SyncStrm, *SyncFileCache) error
	requestEmbyRefresh func(uint) error
	detailByFileID     func(context.Context, *SyncStrm, string) (*SyncFileCache, error)
}

// NewStrmGenerationService 创建 STRM 生成服务。
func NewStrmGenerationService() *StrmGenerationService {
	service := &StrmGenerationService{}
	service.buildSyncer = func(syncPath *models.SyncPath, _ *models.Account) (*SyncStrm, error) {
		syncer := NewSyncStrmFromSyncPath(syncPath)
		if syncer == nil {
			return nil, errors.New("初始化 STRM 同步器失败")
		}
		return syncer, nil
	}
	service.compareStrm = func(syncer *SyncStrm, file *SyncFileCache) int {
		return syncer.CompareStrm(file)
	}
	service.processStrmFile = func(syncer *SyncStrm, file *SyncFileCache) error {
		return syncer.ProcessStrmFile(file)
	}
	service.requestEmbyRefresh = models.RequestEmbyLibraryRefreshBySyncPathId
	service.detailByFileID = func(ctx context.Context, syncer *SyncStrm, fileID string) (*SyncFileCache, error) {
		if syncer == nil || syncer.SyncDriver == nil {
			return nil, errors.New("STRM 同步器未初始化远端驱动")
		}
		return syncer.SyncDriver.DetailByFileId(ctx, fileID)
	}
	return service
}

// Generate 为单个远端文件生成或确认 STRM。
func (service *StrmGenerationService) Generate(ctx context.Context, input StrmGenerationInput) (*StrmGenerationResult, error) {
	if service == nil {
		service = NewStrmGenerationService()
	}
	task := input.Task
	if task == nil {
		return nil, errors.New("STRM 生成任务为空")
	}
	if task.SyncPathId == 0 {
		return nil, errors.New("STRM 生成任务缺少同步目录")
	}
	if task.TaskType != "" && task.TaskType != models.StrmGenerationTaskTypeFile {
		return nil, fmt.Errorf("暂不支持的 STRM 生成任务类型：%s", task.TaskType)
	}

	syncPath := models.GetSyncPathById(task.SyncPathId)
	if syncPath == nil {
		return nil, fmt.Errorf("同步目录不存在：%d", task.SyncPathId)
	}
	account, err := loadStrmGenerationAccount(syncPath, task.AccountId)
	if err != nil {
		return nil, err
	}
	syncer, err := service.buildSyncer(syncPath, account)
	if err != nil {
		return nil, err
	}
	if syncer == nil {
		return nil, errors.New("STRM 同步器为空")
	}
	if syncer.Cancel != nil {
		defer syncer.Cancel()
	}

	file, err := service.buildFileCache(ctx, syncer, syncPath, account, task)
	if err != nil {
		return nil, err
	}
	existing, err := findExistingGeneratedSyncFile(task.SyncPathId, file.GetFileId(), file.PickCode)
	if err != nil {
		return nil, err
	}

	syncFile := file.GetSyncFile(syncer, account.BaseUrl)
	if existing != nil {
		syncFile.BaseModel = existing.BaseModel
	}
	oldLocalPath := ""
	if existing != nil {
		oldLocalPath = existing.LocalFilePath
	}
	newLocalPath := syncFile.LocalFilePath

	changed := file.IsVideo && service.compareStrm(syncer, file) != 1
	if changed {
		if err := service.processStrmFile(syncer, file); err != nil {
			return nil, fmt.Errorf("生成 STRM 文件失败：%w", err)
		}
		if err := cleanupOldGeneratedStrm(oldLocalPath, newLocalPath); err != nil {
			return nil, err
		}
	}

	if err := db.Db.Save(syncFile).Error; err != nil {
		return nil, fmt.Errorf("保存 SyncFile 失败：%w", err)
	}
	if changed {
		if err := service.requestEmbyRefresh(task.SyncPathId); err != nil {
			return nil, fmt.Errorf("提交 Emby 刷新任务失败：%w", err)
		}
	}
	return &StrmGenerationResult{SyncFile: syncFile, Changed: changed}, nil
}

func (service *StrmGenerationService) buildFileCache(ctx context.Context, syncer *SyncStrm, syncPath *models.SyncPath, account *models.Account, task *models.StrmGenerationTask) (*SyncFileCache, error) {
	file := &SyncFileCache{
		FileId:     task.FileId,
		ParentId:   task.ParentId,
		FileType:   v115open.TypeFile,
		FileName:   task.FileName,
		Path:       task.Path,
		FileSize:   task.FileSize,
		MTime:      task.Mtime,
		PickCode:   task.PickCode,
		Sha1:       task.Sha1,
		SourceType: syncPath.SourceType,
	}
	if file.SourceType == "" && account != nil {
		file.SourceType = account.SourceType
	}
	if fileNeedsRemoteDetail(file) && task.FileId != "" {
		detail, err := service.detailByFileID(ctx, syncer, task.FileId)
		if err != nil {
			return nil, fmt.Errorf("补齐远端文件详情失败：%w", err)
		}
		mergeFileCache(file, detail)
	}
	if file.Path == "" {
		file.Path = syncPath.RemotePath
	}
	if file.SourceType == "" {
		file.SourceType = models.SourceType115
	}
	file.IsVideo = syncer.IsValidVideoExt(file.FileName)
	file.IsMeta = syncer.IsValidMetaExt(file.FileName)
	if file.FileName == "" {
		return nil, errors.New("STRM 生成任务缺少文件名")
	}
	if file.GetFileId() == "" && file.PickCode == "" {
		return nil, errors.New("STRM 生成任务缺少远端文件标识")
	}
	return file, nil
}

func fileNeedsRemoteDetail(file *SyncFileCache) bool {
	if file == nil {
		return false
	}
	return file.FileName == "" || file.Path == "" || file.ParentId == "" || file.PickCode == ""
}

func mergeFileCache(target *SyncFileCache, detail *SyncFileCache) {
	if target == nil || detail == nil {
		return
	}
	if detail.FileId != "" {
		target.FileId = detail.FileId
	}
	if detail.ParentId != "" {
		target.ParentId = detail.ParentId
	}
	if detail.FileType != "" {
		target.FileType = detail.FileType
	}
	if detail.FileName != "" {
		target.FileName = detail.FileName
	}
	if detail.Path != "" {
		target.Path = detail.Path
	}
	if detail.FileSize > 0 {
		target.FileSize = detail.FileSize
	}
	if detail.MTime > 0 {
		target.MTime = detail.MTime
	}
	if detail.PickCode != "" {
		target.PickCode = detail.PickCode
	}
	if detail.Sha1 != "" {
		target.Sha1 = detail.Sha1
	}
	if detail.ThumbUrl != "" {
		target.ThumbUrl = detail.ThumbUrl
	}
	if detail.OpenlistSign != "" {
		target.OpenlistSign = detail.OpenlistSign
	}
	if detail.SourceType != "" {
		target.SourceType = detail.SourceType
	}
	if len(detail.Paths) > 0 {
		target.Paths = detail.Paths
	}
}

func loadStrmGenerationAccount(syncPath *models.SyncPath, taskAccountID uint) (*models.Account, error) {
	accountID := taskAccountID
	if accountID == 0 && syncPath != nil {
		accountID = syncPath.AccountId
	}
	if accountID == 0 {
		return &models.Account{SourceType: models.SourceTypeLocal}, nil
	}
	account, err := models.GetAccountById(accountID)
	if err != nil {
		return nil, fmt.Errorf("获取网盘账号失败：%w", err)
	}
	return account, nil
}

func findExistingGeneratedSyncFile(syncPathID uint, fileID string, pickCode string) (*models.SyncFile, error) {
	if fileID == "" && pickCode == "" {
		return nil, nil
	}
	var existing models.SyncFile
	query := db.Db.Where("sync_path_id = ?", syncPathID)
	switch {
	case fileID != "" && pickCode != "":
		query = query.Where("(file_id = ? OR pick_code = ?)", fileID, pickCode)
	case fileID != "":
		query = query.Where("file_id = ?", fileID)
	default:
		query = query.Where("pick_code = ?", pickCode)
	}
	err := query.Order("id ASC").First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func cleanupOldGeneratedStrm(oldLocalPath string, newLocalPath string) error {
	if oldLocalPath == "" || newLocalPath == "" {
		return nil
	}
	oldLocalPath = filepath.Clean(oldLocalPath)
	newLocalPath = filepath.Clean(newLocalPath)
	if oldLocalPath == newLocalPath {
		return nil
	}
	if err := os.Remove(oldLocalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %v", errOldStrmCleanupFailed, err)
	}
	return nil
}

// InitStrmGenerationWorker 初始化 STRM 生成队列 worker。
func InitStrmGenerationWorker() {
	if err := models.ResetRunningStrmGenerationTasks(); err != nil {
		helpers.AppLogger.Errorf("恢复 STRM 生成任务失败：%v", err)
	}
	strmGenerationWorkerOnce.Do(func() {
		go runStrmGenerationWorker(context.Background(), NewStrmGenerationService())
	})
}

func runStrmGenerationWorker(ctx context.Context, service *StrmGenerationService) {
	ticker := time.NewTicker(strmGenerationWorkerInterval)
	defer ticker.Stop()
	for {
		if _, err := ProcessPendingStrmGenerationTasks(ctx, service, strmGenerationWorkerBatch); err != nil && !errors.Is(err, context.Canceled) {
			helpers.AppLogger.Errorf("处理 STRM 生成队列失败：%v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ProcessPendingStrmGenerationTasks 处理一批待生成 STRM 任务。
func ProcessPendingStrmGenerationTasks(ctx context.Context, service *StrmGenerationService, limit int) (int, error) {
	if service == nil {
		service = NewStrmGenerationService()
	}
	tasks, err := models.GetPendingStrmGenerationTasks(limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return processed, ctx.Err()
		default:
		}
		if err := task.MarkRunning(); err != nil {
			continue
		}
		processed++
		if _, err := service.Generate(ctx, StrmGenerationInput{Task: task}); err != nil {
			if markErr := task.MarkFailed(err.Error()); markErr != nil {
				return processed, markErr
			}
			if task.ParentTaskId > 0 {
				_ = models.IncrementStrmGenerationDirectoryStats(task.ParentTaskId, 0, 1)
			}
			continue
		}
		if err := task.MarkCompleted(); err != nil {
			return processed, err
		}
		if task.ParentTaskId > 0 {
			_ = models.IncrementStrmGenerationDirectoryStats(task.ParentTaskId, 1, 0)
		}
	}
	return processed, nil
}
