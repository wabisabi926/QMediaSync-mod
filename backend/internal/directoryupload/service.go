package directoryupload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/v115open"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RemoteDirectory 是目录监控上传目标目录。
type RemoteDirectory struct {
	ID   string
	Path string
}

// RemoteFile 是目录监控上传前检查到的远端文件。
type RemoteFile struct {
	ID       string
	PickCode string
	SHA1     string
	Size     int64
	Mtime    int64
}

// RemoteClient 封装目录监控创建上传任务前需要的远端能力。
type RemoteClient interface {
	EnsureDir(ctx context.Context, rule *models.DirectoryUploadRule, relativeDir string) (RemoteDirectory, error)
	FindFile(ctx context.Context, parentID string, fileName string) (*RemoteFile, error)
	DeleteFile(ctx context.Context, parentID string, fileID string) error
}

var errStableFileNoRetry = errors.New("稳定文件不再重试")

// HandleStableFileOptions 是处理稳定文件的内部选项。
type HandleStableFileOptions struct {
	Force bool
}

type handleStableFileOptions = HandleStableFileOptions

// ServiceOptions 是目录监控上传服务依赖。
type ServiceOptions struct {
	Now                       func() time.Time
	RemoteClient              RemoteClient
	WatcherFactory            WatcherFactory
	PollInterval              time.Duration
	StabilityCheckInterval    time.Duration
	ProcessedCleanupInterval  time.Duration
	ProcessedMissingSourceTTL time.Duration
}

type processedFile struct {
	expiresAt time.Time
}

type processedSourceState struct {
	scopeHash         string
	sourceKey         string
	sourceFingerprint string
	fileSize          int64
	mtimeNs           int64
}

// Service 负责把稳定后的本地文件转换为上传队列任务。
type Service struct {
	now                    func() time.Time
	queue                  *StabilityQueue
	remoteClient           RemoteClient
	watcherFactory         WatcherFactory
	pollInterval           time.Duration
	stabilityCheckInterval time.Duration
	cleanupInterval        time.Duration
	missingSourceTTL       time.Duration

	mutex          sync.Mutex
	processed      map[string]processedFile
	recentlyQueued map[string]processedFile
	runtimes       []*RuleRuntime
	cleanupCancel  context.CancelFunc
	cleanupDone    chan struct{}
}

// NewService 创建目录监控上传服务。
func NewService(options ServiceOptions) *Service {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		now:                    now,
		queue:                  NewStabilityQueue(StabilityQueueOptions{Now: now}),
		remoteClient:           options.RemoteClient,
		watcherFactory:         options.WatcherFactory,
		pollInterval:           options.PollInterval,
		stabilityCheckInterval: options.StabilityCheckInterval,
		cleanupInterval:        options.ProcessedCleanupInterval,
		missingSourceTTL:       options.ProcessedMissingSourceTTL,
		processed:              make(map[string]processedFile),
		recentlyQueued:         make(map[string]processedFile),
	}
}

// SetRemoteClient 设置远端客户端，主要供测试替换。
func (service *Service) SetRemoteClient(client RemoteClient) {
	if service == nil {
		return
	}
	service.mutex.Lock()
	defer service.mutex.Unlock()
	service.remoteClient = client
}

// PendingPaths 返回指定规则当前待稳定文件路径。
func (service *Service) PendingPaths(ruleID uint) []string {
	if service == nil || service.queue == nil {
		return []string{}
	}
	return service.queue.PendingPaths(ruleID)
}

// TrackPath 将文件加入稳定性检查队列。
func (service *Service) TrackPath(ruleID uint, path string) {
	if service == nil || service.queue == nil {
		return
	}
	service.queue.Track(ruleID, path)
}

func (service *Service) trackCandidatePath(ctx context.Context, rule *models.DirectoryUploadRule, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if rule == nil {
		return false, errors.New("目录监控规则为空")
	}
	syncPath := models.GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return false, fmt.Errorf("同步目录不存在：%d", rule.SyncPathId)
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	rel, err := safeRelativePath(rule.MonitorPath, path)
	if err != nil {
		return false, err
	}
	if !rule.Recursive && isNestedRelativePath(rel) {
		return false, nil
	}
	if shouldIgnorePath(rel, info.Name(), false, parseIgnorePatterns(rule.IgnorePatternsStr)) {
		return false, nil
	}
	if !shouldUploadByRule(rule, syncPath, info.Name()) {
		return false, nil
	}
	sourceFingerprint := models.BuildDirectoryUploadSourceFingerprint(info.Size(), info.ModTime().UnixNano())
	if service.trackRecentlyQueued(rule, rel, sourceFingerprint) {
		return false, nil
	}
	service.queue.Track(rule.ID, path)
	return true, nil
}

// CheckStableFiles 执行指定规则的一轮稳定性检查。
func (service *Service) CheckStableFiles(rule *models.DirectoryUploadRule) ([]StableFile, error) {
	if service == nil || service.queue == nil {
		return []StableFile{}, nil
	}
	return service.queue.Check(rule)
}

// ScanRule 扫描目录上传规则，把候选视频文件加入稳定性队列。
func (service *Service) ScanRule(ctx context.Context, rule *models.DirectoryUploadRule) (int, error) {
	root := ""
	if rule != nil {
		root = rule.MonitorPath
	}
	return service.scanRoot(ctx, rule, root)
}

// ScanSubtree 扫描监控目录下的新子目录，把候选文件加入稳定性队列。
func (service *Service) ScanSubtree(ctx context.Context, rule *models.DirectoryUploadRule, root string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if rule == nil {
		return 0, errors.New("目录监控规则为空")
	}
	if err := ensurePathInMonitor(rule.MonitorPath, root); err != nil {
		return 0, err
	}
	accepted, err := service.scanRoot(ctx, rule, root)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return accepted, nil
	}
	return accepted, err
}

func (service *Service) scanRoot(ctx context.Context, rule *models.DirectoryUploadRule, root string) (int, error) {
	if service == nil {
		service = NewService(ServiceOptions{})
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if rule == nil {
		return 0, errors.New("目录监控规则为空")
	}
	syncPath := models.GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return 0, fmt.Errorf("同步目录不存在：%d", rule.SyncPathId)
	}
	monitorPath := filepath.Clean(rule.MonitorPath)
	root = filepath.Clean(root)
	if err := ensurePathInMonitor(monitorPath, root); err != nil {
		return 0, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return 0, fmt.Errorf("读取监控目录失败：%w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("监控路径不是目录：%s", root)
	}

	accepted := 0
	ignorePatterns := parseIgnorePatterns(rule.IgnorePatternsStr)
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := relativePathInMonitor(monitorPath, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if rel != "." {
				if shouldIgnorePath(rel, entry.Name(), true, ignorePatterns) {
					return filepath.SkipDir
				}
				if !rule.Recursive {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !rule.Recursive && isNestedRelativePath(rel) {
			return nil
		}
		if shouldIgnorePath(rel, entry.Name(), false, ignorePatterns) || !shouldUploadByRule(rule, syncPath, entry.Name()) {
			return nil
		}
		service.queue.Track(rule.ID, path)
		accepted++
		return nil
	})
	if walkErr != nil {
		return accepted, walkErr
	}
	return accepted, nil
}

// HandleStableFile 为已稳定的本地文件创建上传任务。
func (service *Service) HandleStableFile(ctx context.Context, rule *models.DirectoryUploadRule, filePath string) error {
	return service.handleStableFile(ctx, rule, filePath, handleStableFileOptions{})
}

func (service *Service) handleStableFile(ctx context.Context, rule *models.DirectoryUploadRule, filePath string, options handleStableFileOptions) error {
	if service == nil {
		service = NewService(ServiceOptions{})
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if rule == nil {
		return errors.New("目录监控规则为空")
	}
	filePath = filepath.Clean(filePath)
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("读取稳定文件失败：%w", err)
	}
	if info.IsDir() {
		return nil
	}
	syncPath := models.GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return fmt.Errorf("%w：同步目录不存在：%d", errStableFileNoRetry, rule.SyncPathId)
	}
	if !shouldUploadByRule(rule, syncPath, info.Name()) {
		return nil
	}
	rel, err := safeRelativePath(rule.MonitorPath, filePath)
	if err != nil {
		return err
	}
	skipProcessed, sourceState, err := service.shouldSkipProcessedSource(rule, rel, filePath, info, options)
	if err != nil {
		return err
	}
	if skipProcessed {
		return nil
	}
	if !models.CheckCanUploadByLocalPath(models.UploadSourceDirectoryMonitor, filePath) {
		return nil
	}

	relativeDir := filepath.ToSlash(filepath.Dir(rel))
	if relativeDir == "." {
		relativeDir = ""
	}
	remoteClient, err := service.remote(ctx, rule)
	if err != nil {
		return err
	}
	remoteDir, err := remoteClient.EnsureDir(ctx, rule, relativeDir)
	if err != nil {
		return fmt.Errorf("确认远端目录失败：%w", err)
	}
	fileName := info.Name()
	remoteFilePath := cleanRemoteFilePath(rule.RemoteRootPath, rel)
	task := &models.DbUploadTask{
		Source:              models.UploadSourceDirectoryMonitor,
		AccountId:           rule.AccountId,
		SyncPathId:          rule.SyncPathId,
		SourceType:          models.SourceType115,
		LocalFullPath:       filePath,
		RelativePath:        filepath.ToSlash(rel),
		SourceFingerprint:   sourceState.sourceFingerprint,
		RemoteFileId:        remoteFilePath,
		RemotePathId:        remoteDir.ID,
		FileName:            fileName,
		Status:              models.UploadStatusPending,
		FileSize:            sourceState.fileSize,
		LocalMtime:          info.ModTime().Unix(),
		LocalMtimeNs:        sourceState.mtimeNs,
		UploadResult:        models.UploadResultUnknown,
		SourceCleanupStatus: cleanupInitialStatus(rule),
	}

	remoteFile, err := remoteClient.FindFile(ctx, remoteDir.ID, fileName)
	if err != nil {
		return fmt.Errorf("检查远端同名文件失败：%w", err)
	}
	if remoteFile != nil && remoteFile.ID != "" {
		if isSameRemoteFile(remoteFile, filePath, info.Size()) {
			task.Status = models.UploadStatusCompleted
			task.UploadResult = models.UploadResultRemoteExists
			task.UploadedBytes = info.Size()
			task.CompletedRemoteFileId = remoteFile.ID
			task.CompletedPickCode = remoteFile.PickCode
			task.EndTime = service.now().Unix()
			created, err := service.createDirectoryUploadTaskWithProcessedClaim(task, rule, rel, filePath, sourceState, options)
			if err != nil {
				return err
			}
			if !created {
				return nil
			}
			if err := task.EnqueueStrmGenerationAfterUploadAndMarkDirectoryProcessed(); err != nil {
				return fmt.Errorf("创建 STRM 生成任务失败：%w", err)
			}
			service.markProcessed(rule, rel, sourceState.sourceFingerprint)
			return nil
		}

		switch rule.OverwriteMode {
		case "", models.DirectoryUploadOverwriteSkipSame:
			if err := service.upsertDirectoryUploadProcessed(rule, rel, filePath, sourceState, models.DirectoryUploadProcessedResultSkippedExisting, 0); err != nil {
				return fmt.Errorf("记录跳过已有远端文件失败：%w", err)
			}
			service.markProcessed(rule, rel, sourceState.sourceFingerprint)
			return nil
		case models.DirectoryUploadOverwriteFailConflict:
			if err := service.upsertDirectoryUploadProcessed(rule, rel, filePath, sourceState, models.DirectoryUploadProcessedResultFailed, 0); err != nil {
				return fmt.Errorf("记录远端同名冲突失败：%w", err)
			}
			return fmt.Errorf("%w：远端已存在同名文件且大小或 SHA1 不一致：%s", errStableFileNoRetry, remoteFilePath)
		case models.DirectoryUploadOverwriteReplaceConflict:
			if err := remoteClient.DeleteFile(ctx, remoteDir.ID, remoteFile.ID); err != nil {
				return fmt.Errorf("删除远端同名文件失败：%w", err)
			}
		default:
			return fmt.Errorf("%w：不支持的同名文件处理方式：%s", errStableFileNoRetry, rule.OverwriteMode)
		}
	}

	created, err := service.createDirectoryUploadTaskWithProcessedClaim(task, rule, rel, filePath, sourceState, options)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	return nil
}

func (service *Service) shouldSkipProcessedSource(rule *models.DirectoryUploadRule, rel string, filePath string, info os.FileInfo, options handleStableFileOptions) (bool, processedSourceState, error) {
	state := buildProcessedSourceState(rule, rel, info)
	if options.Force {
		return false, state, nil
	}
	if service.isProcessed(rule, rel, state.sourceFingerprint) {
		return true, state, nil
	}

	record, err := models.FindDirectoryUploadProcessedBySourceKey(state.sourceKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, state, nil
		}
		return false, state, fmt.Errorf("查询目录监控源文件处理记录失败：%w", err)
	}
	if record.SourceFingerprint != state.sourceFingerprint {
		return false, state, nil
	}

	now := service.now().Unix()
	switch {
	case models.IsDirectoryUploadProcessedTerminal(record.Result):
		if err := updateDirectoryUploadProcessedLastSeen(record.SourceKey, now); err != nil {
			return false, state, fmt.Errorf("更新目录监控源文件最后发现时间失败：%w", err)
		}
		service.markProcessed(rule, rel, state.sourceFingerprint)
		return true, state, nil
	case record.Result == models.DirectoryUploadProcessedResultQueued:
		active, err := hasActiveDirectoryUploadTask(record.UploadTaskId, filePath)
		if err != nil {
			return false, state, fmt.Errorf("检查目录监控源文件上传任务状态失败：%w", err)
		}
		if active {
			if err := updateDirectoryUploadProcessedLastSeen(record.SourceKey, now); err != nil {
				return false, state, fmt.Errorf("更新目录监控源文件最后发现时间失败：%w", err)
			}
			return true, state, nil
		}
	}

	return false, state, nil
}

func buildProcessedSourceState(rule *models.DirectoryUploadRule, rel string, info os.FileInfo) processedSourceState {
	mtimeNs := info.ModTime().UnixNano()
	scopeHash := models.BuildDirectoryUploadScopeHash(rule)
	return processedSourceState{
		scopeHash:         scopeHash,
		sourceKey:         models.BuildDirectoryUploadSourceKey(scopeHash, rel),
		sourceFingerprint: models.BuildDirectoryUploadSourceFingerprint(info.Size(), mtimeNs),
		fileSize:          info.Size(),
		mtimeNs:           mtimeNs,
	}
}

func updateDirectoryUploadProcessedLastSeen(sourceKey string, lastSeenAt int64) error {
	return db.Db.Model(&models.DirectoryUploadProcessedFile{}).
		Where("source_key = ?", sourceKey).
		Updates(map[string]any{
			"last_seen_at": lastSeenAt,
		}).Error
}

func hasActiveDirectoryUploadTask(uploadTaskID uint, filePath string) (bool, error) {
	return hasActiveDirectoryUploadTaskWithDB(nil, uploadTaskID, filePath)
}

func hasActiveDirectoryUploadTaskWithDB(tx *gorm.DB, uploadTaskID uint, filePath string) (bool, error) {
	handle := tx
	if handle == nil {
		handle = db.Db
	}
	query := handle.Model(&models.DbUploadTask{}).
		Where("status IN ?", []models.UploadStatus{models.UploadStatusPending, models.UploadStatusUploading})
	if uploadTaskID > 0 {
		query = query.Where("id = ?", uploadTaskID)
	} else {
		query = query.Where("source = ? AND local_full_path = ?", models.UploadSourceDirectoryMonitor, filePath)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (service *Service) createDirectoryUploadTaskWithProcessedClaim(task *models.DbUploadTask, rule *models.DirectoryUploadRule, rel string, filePath string, state processedSourceState, options ...handleStableFileOptions) (bool, error) {
	var option handleStableFileOptions
	if len(options) > 0 {
		option = options[0]
	}
	var created bool
	err := db.Db.Transaction(func(tx *gorm.DB) error {
		claimed, err := service.claimDirectoryUploadProcessedWithDB(tx, rule, rel, filePath, state, option)
		if err != nil || !claimed {
			return err
		}
		if err := models.SaveDirectoryMonitorUploadTaskWithDB(tx, task); err != nil {
			return err
		}
		if err := service.upsertDirectoryUploadProcessedWithDB(tx, rule, rel, filePath, state, models.DirectoryUploadProcessedResultQueued, task.ID); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("创建目录监控上传任务失败：%w", err)
	}
	if created {
		models.PublishUploadTaskCreated(task)
	}
	return created, nil
}

func (service *Service) claimDirectoryUploadProcessedWithDB(tx *gorm.DB, rule *models.DirectoryUploadRule, rel string, filePath string, state processedSourceState, options handleStableFileOptions) (bool, error) {
	claim := service.newDirectoryUploadProcessedRecord(rule, rel, filePath, state, models.DirectoryUploadProcessedResultQueued, 0)
	insert := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_key"}},
		DoNothing: true,
	}).Create(claim)
	if insert.Error != nil {
		return false, insert.Error
	}
	if insert.RowsAffected > 0 {
		return true, nil
	}

	var existing models.DirectoryUploadProcessedFile
	if err := tx.Where("source_key = ?", state.sourceKey).First(&existing).Error; err != nil {
		return false, err
	}
	if existing.SourceFingerprint != state.sourceFingerprint ||
		existing.Result == models.DirectoryUploadProcessedResultFailed {
		claimed, err := service.updateDirectoryUploadProcessedClaimWithDB(tx, &existing, claim)
		if err != nil || claimed {
			return claimed, err
		}
		return false, nil
	}
	if existing.Result == models.DirectoryUploadProcessedResultQueued {
		active, err := hasActiveDirectoryUploadTaskWithDB(tx, existing.UploadTaskId, filePath)
		if err != nil || active {
			return false, err
		}
		return service.updateDirectoryUploadProcessedClaimWithDB(tx, &existing, claim)
	}
	if options.Force {
		active, err := hasActiveDirectoryUploadTaskWithDB(tx, existing.UploadTaskId, filePath)
		if err != nil || active {
			return false, err
		}
		return service.updateDirectoryUploadProcessedClaimWithDB(tx, &existing, claim)
	}
	return false, nil
}

func (service *Service) updateDirectoryUploadProcessedClaimWithDB(tx *gorm.DB, existing *models.DirectoryUploadProcessedFile, claim *models.DirectoryUploadProcessedFile) (bool, error) {
	result := tx.Model(&models.DirectoryUploadProcessedFile{}).
		Where("source_key = ? AND source_fingerprint = ? AND result = ? AND upload_task_id = ?",
			existing.SourceKey,
			existing.SourceFingerprint,
			existing.Result,
			existing.UploadTaskId,
		).
		Updates(map[string]any{
			"rule_id":            claim.RuleId,
			"sync_path_id":       claim.SyncPathId,
			"account_id":         claim.AccountId,
			"scope_hash":         claim.ScopeHash,
			"relative_path":      claim.RelativePath,
			"local_full_path":    claim.LocalFullPath,
			"source_fingerprint": claim.SourceFingerprint,
			"file_size":          claim.FileSize,
			"local_mtime_ns":     claim.LocalMtimeNs,
			"result":             claim.Result,
			"upload_task_id":     claim.UploadTaskId,
			"processed_at":       claim.ProcessedAt,
			"last_seen_at":       claim.LastSeenAt,
			"updated_at":         claim.UpdatedAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (service *Service) upsertDirectoryUploadProcessed(rule *models.DirectoryUploadRule, rel string, filePath string, state processedSourceState, result models.DirectoryUploadProcessedResult, uploadTaskID uint) error {
	return service.upsertDirectoryUploadProcessedWithDB(db.Db, rule, rel, filePath, state, result, uploadTaskID)
}

func (service *Service) upsertDirectoryUploadProcessedWithDB(tx *gorm.DB, rule *models.DirectoryUploadRule, rel string, filePath string, state processedSourceState, result models.DirectoryUploadProcessedResult, uploadTaskID uint) error {
	record := service.newDirectoryUploadProcessedRecord(rule, rel, filePath, state, result, uploadTaskID)
	return models.UpsertDirectoryUploadProcessedFileWithDB(tx, record)
}

func (service *Service) newDirectoryUploadProcessedRecord(rule *models.DirectoryUploadRule, rel string, filePath string, state processedSourceState, result models.DirectoryUploadProcessedResult, uploadTaskID uint) *models.DirectoryUploadProcessedFile {
	now := service.now().Unix()
	return &models.DirectoryUploadProcessedFile{
		RuleId:            rule.ID,
		SyncPathId:        rule.SyncPathId,
		AccountId:         rule.AccountId,
		ScopeHash:         state.scopeHash,
		SourceKey:         state.sourceKey,
		RelativePath:      filepath.ToSlash(rel),
		LocalFullPath:     filePath,
		SourceFingerprint: state.sourceFingerprint,
		FileSize:          state.fileSize,
		LocalMtimeNs:      state.mtimeNs,
		Result:            result,
		UploadTaskId:      uploadTaskID,
		ProcessedAt:       now,
		LastSeenAt:        now,
	}
}

func (service *Service) remote(ctx context.Context, rule *models.DirectoryUploadRule) (RemoteClient, error) {
	service.mutex.Lock()
	client := service.remoteClient
	service.mutex.Unlock()
	if client != nil {
		return client, nil
	}
	return newOpen115RemoteClient(ctx, rule)
}

func shouldUploadByRule(rule *models.DirectoryUploadRule, syncPath *models.SyncPath, name string) bool {
	if syncPath == nil {
		return false
	}
	if syncPath.IsValidVideoExt(name) {
		return true
	}
	return rule != nil && rule.UploadMetadata && syncPath.IsValidMetaExt(name)
}

func (service *Service) isProcessed(rule *models.DirectoryUploadRule, rel string, signature string) bool {
	key := processedKey(rule.ID, rel, signature)
	now := service.now()
	service.mutex.Lock()
	defer service.mutex.Unlock()
	item, ok := service.processed[key]
	if !ok {
		return false
	}
	if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
		delete(service.processed, key)
		return false
	}
	return true
}

func (service *Service) markProcessed(rule *models.DirectoryUploadRule, rel string, signature string) {
	key := processedKey(rule.ID, rel, signature)
	ttl := time.Duration(rule.ProcessedCacheTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	service.mutex.Lock()
	defer service.mutex.Unlock()
	service.processed[key] = processedFile{expiresAt: service.now().Add(ttl)}
}

func (service *Service) trackRecentlyQueued(rule *models.DirectoryUploadRule, rel string, signature string) bool {
	if service == nil || rule == nil {
		return false
	}
	key := processedKey(rule.ID, rel, signature)
	ttl := time.Duration(rule.ProcessedCacheTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	now := time.Now()
	if service.now != nil {
		now = service.now()
	}
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.recentlyQueued == nil {
		service.recentlyQueued = make(map[string]processedFile)
	}
	cleanupExpiredProcessedFileMap(service.recentlyQueued, now)
	if item, ok := service.recentlyQueued[key]; ok {
		if item.expiresAt.IsZero() || now.Before(item.expiresAt) || now.Equal(item.expiresAt) {
			return true
		}
		delete(service.recentlyQueued, key)
	}
	service.recentlyQueued[key] = processedFile{expiresAt: now.Add(ttl)}
	return false
}

func (service *Service) cleanupExpiredMemoryCaches(now time.Time) {
	if service == nil {
		return
	}
	service.mutex.Lock()
	defer service.mutex.Unlock()
	cleanupExpiredProcessedFileMap(service.processed, now)
	cleanupExpiredProcessedFileMap(service.recentlyQueued, now)
}

func cleanupExpiredProcessedFileMap(items map[string]processedFile, now time.Time) {
	for key, file := range items {
		if !file.expiresAt.IsZero() && now.After(file.expiresAt) {
			delete(items, key)
		}
	}
}

func processedKey(ruleID uint, rel string, sourceFingerprint string) string {
	return fmt.Sprintf("%d:%s:%s", ruleID, filepath.ToSlash(strings.ReplaceAll(rel, "\\", "/")), sourceFingerprint)
}

func cleanupInitialStatus(rule *models.DirectoryUploadRule) models.UploadSourceCleanupStatus {
	if rule != nil && rule.DeleteSourceAfterSuccess {
		return models.UploadSourceCleanupStatusPending
	}
	return models.UploadSourceCleanupStatusNone
}

func cleanRemoteFilePath(rootPath string, rel string) string {
	rootPath = strings.ReplaceAll(strings.TrimSpace(rootPath), "\\", "/")
	if rootPath == "" {
		rootPath = "/"
	}
	if !strings.HasPrefix(rootPath, "/") {
		rootPath = "/" + rootPath
	}
	parts := []string{rootPath}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return pathpkg.Clean(pathpkg.Join(parts...))
}

func safeRelativePath(basePath string, path string) (string, error) {
	basePath = filepath.Clean(basePath)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		return "", fmt.Errorf("计算相对路径失败：%w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("文件路径越界：%s", path)
	}
	return filepath.ToSlash(rel), nil
}

func ensurePathInMonitor(monitorPath string, path string) error {
	_, err := relativePathInMonitor(monitorPath, path)
	return err
}

func relativePathInMonitor(monitorPath string, path string) (string, error) {
	monitorPath = filepath.Clean(monitorPath)
	path = filepath.Clean(path)
	if path == monitorPath {
		return ".", nil
	}
	return safeRelativePath(monitorPath, path)
}

func isNestedRelativePath(rel string) bool {
	rel = pathpkg.Clean(filepath.ToSlash(rel))
	return rel != "." && pathpkg.Dir(rel) != "."
}

func parseIgnorePatterns(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var patterns []string
	if err := json.Unmarshal([]byte(raw), &patterns); err == nil {
		return compactPatterns(patterns)
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	return compactPatterns(fields)
}

func compactPatterns(patterns []string) []string {
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			result = append(result, pattern)
		}
	}
	return result
}

func shouldIgnorePath(rel string, name string, isDir bool, patterns []string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	lowerName := strings.ToLower(name)
	for _, suffix := range []string{".part", ".tmp", ".download"} {
		if strings.HasSuffix(lowerName, suffix) {
			return true
		}
	}
	rel = filepath.ToSlash(rel)
	for _, pattern := range patterns {
		if matched, _ := pathpkg.Match(pattern, rel); matched {
			return true
		}
		if matched, _ := pathpkg.Match(pattern, name); matched {
			return true
		}
		if isDir && strings.TrimSuffix(pattern, "/") == rel {
			return true
		}
	}
	return false
}

func isSameRemoteFile(remoteFile *RemoteFile, localPath string, size int64) bool {
	if remoteFile == nil || remoteFile.ID == "" {
		return false
	}
	if remoteFile.Size != size {
		return false
	}
	if strings.TrimSpace(remoteFile.SHA1) == "" {
		return false
	}
	localSHA1, err := helpers.FileSHA1(localPath)
	if err != nil {
		return false
	}
	return strings.EqualFold(remoteFile.SHA1, localSHA1)
}

type open115RemoteClient struct {
	client *v115open.OpenClient
}

func newOpen115RemoteClient(_ context.Context, rule *models.DirectoryUploadRule) (RemoteClient, error) {
	if rule == nil {
		return nil, errors.New("目录监控规则为空")
	}
	account, err := models.GetAccountById(rule.AccountId)
	if err != nil {
		return nil, fmt.Errorf("获取 115 账号失败：%w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("账号不存在：%d", rule.AccountId)
	}
	client := account.Get115Client()
	if client == nil {
		return nil, fmt.Errorf("账号 %d 未初始化 115 客户端", rule.AccountId)
	}
	return &open115RemoteClient{client: client}, nil
}

func (client *open115RemoteClient) EnsureDir(ctx context.Context, rule *models.DirectoryUploadRule, relativeDir string) (RemoteDirectory, error) {
	if client == nil || client.client == nil {
		return RemoteDirectory{}, errors.New("115 远端客户端为空")
	}
	if rule == nil {
		return RemoteDirectory{}, errors.New("目录监控规则为空")
	}
	parentID := rule.RemoteRootId
	if parentID == "" {
		return RemoteDirectory{}, errors.New("远端根目录 ID 为空")
	}
	rootPath := strings.TrimSpace(rule.RemoteRootPath)
	if rootPath == "" {
		rootPath = "/"
	}
	relativeDir = strings.Trim(filepath.ToSlash(relativeDir), "/")
	if relativeDir == "" || relativeDir == "." {
		return RemoteDirectory{ID: parentID, Path: cleanRemoteFilePath(rootPath, "")}, nil
	}

	currentPath := strings.TrimSuffix(cleanRemoteFilePath(rootPath, ""), "/")
	if currentPath == "." {
		currentPath = "/"
	}
	for _, segment := range strings.Split(relativeDir, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." {
			continue
		}
		nextPath := cleanRemoteFilePath(currentPath, segment)
		detail, err := client.client.GetFsDetailByPath(ctx, nextPath)
		if err == nil && detail != nil && detail.FileId != "" {
			parentID = detail.FileId
			currentPath = nextPath
			continue
		}
		createdID, err := client.client.MkDir(ctx, parentID, segment)
		if err != nil {
			return RemoteDirectory{}, fmt.Errorf("创建远端目录 %s 失败：%w", nextPath, err)
		}
		parentID = createdID
		currentPath = nextPath
	}
	return RemoteDirectory{ID: parentID, Path: currentPath}, nil
}

func (client *open115RemoteClient) FindFile(ctx context.Context, parentID string, fileName string) (*RemoteFile, error) {
	if client == nil || client.client == nil {
		return nil, errors.New("115 远端客户端为空")
	}
	if parentID == "" || fileName == "" {
		return nil, nil
	}
	const pageSize = 1150
	for offset := 0; ; offset += pageSize {
		resp, err := client.client.GetFsList(ctx, parentID, true, false, false, offset, pageSize)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil, nil
		}
		for _, item := range resp.Data {
			if item.FileCategory != v115open.TypeFile || item.FileName != fileName {
				continue
			}
			mtime := item.Ptime
			if mtime == 0 {
				mtime = item.Utime
			}
			return &RemoteFile{
				ID:       item.FileId,
				PickCode: item.PickCode,
				SHA1:     item.Sha1,
				Size:     item.FileSize,
				Mtime:    mtime,
			}, nil
		}
		if len(resp.Data) < pageSize {
			return nil, nil
		}
	}
}

func (client *open115RemoteClient) DeleteFile(ctx context.Context, parentID string, fileID string) error {
	if client == nil || client.client == nil {
		return errors.New("115 远端客户端为空")
	}
	if parentID == "" || fileID == "" {
		return errors.New("远端文件信息为空")
	}
	success, err := client.client.Del(ctx, []string{fileID}, parentID)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("115 删除接口返回失败")
	}
	return nil
}
