package models

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	ws "qmediasync/internal/websocket"

	"gorm.io/gorm"
)

// UploadStatus 上传状态
type UploadStatus int

const (
	UploadStatusPending   UploadStatus = iota // 等待中
	UploadStatusUploading                     // 上传中
	UploadStatusCompleted                     // 已完成
	UploadStatusFailed                        // 失败
	UploadStatusCancelled                     // 已取消
	UploadStatusAll       = -1                // 所有状态
)

type UploadSource string

const (
	UploadSourceStrm             UploadSource = "strm_sync"
	UploadSourceScrape           UploadSource = "scrape_organize"
	UploadSourceDirectoryMonitor UploadSource = "directory_monitor"
)

// UploadResult 是上传任务的最终结果类型。
type UploadResult string

const (
	UploadResultUnknown               UploadResult = "unknown"
	UploadResultRapidUpload           UploadResult = "rapid_upload"
	UploadResultMultipartUploaded     UploadResult = "multipart_uploaded"
	UploadResultRemoteExists          UploadResult = "remote_exists"
	UploadResultSkippedAfterRapidWait UploadResult = "skipped_after_rapid_wait"
)

// UploadSourceCleanupStatus 是目录监控上传后的源文件清理状态。
type UploadSourceCleanupStatus string

const (
	UploadSourceCleanupStatusNone      UploadSourceCleanupStatus = "none"
	UploadSourceCleanupStatusPending   UploadSourceCleanupStatus = "pending"
	UploadSourceCleanupStatusCompleted UploadSourceCleanupStatus = "completed"
	UploadSourceCleanupStatusFailed    UploadSourceCleanupStatus = "failed"
)

type DbUploadTask struct {
	BaseModel
	Source                UploadSource              `json:"source"` // 上传来源存储值，展示文案由前端映射
	AccountId             uint                      `json:"account_id"`
	SyncFileId            uint                      `json:"sync_file_id"`                                      // 同步文件 ID
	ScrapeMediaFileId     uint                      `json:"scrape_media_file_id"`                              // 刮削文件 ID
	SyncPathId            uint                      `json:"sync_path_id" gorm:"index"`                         // 同步目录 ID
	SourceType            SourceType                `json:"source_type"`                                       // 任务来源类型
	LocalFullPath         string                    `json:"local_full_path" gorm:"index:idx_local_full_path"`  // 本地完整文件路径，包含文件名
	RelativePath          string                    `json:"relative_path" gorm:"type:text;size:1024"`          // 目录监控源文件相对路径
	SourceFingerprint     string                    `json:"source_fingerprint" gorm:"size:128;index"`          // 目录监控源文件签名
	RemoteFileId          string                    `json:"remote_file_id" gorm:"index:idx_remote_file_id"`    // 远程文件 ID，包含完整路径
	RemotePathId          string                    `json:"remote_path_id"`                                    // 父目录 CID，如果是 115 则是文件夹 ID，如果是 OpenList 则是父文件夹路径
	FileName              string                    `json:"file_name"`                                         // 要上传的文件名
	Status                UploadStatus              `json:"status" gorm:"index:idx_status_new"`                // 任务状态
	FileSize              int64                     `json:"file_size"`                                         // 文件大小
	LocalMtime            int64                     `json:"local_mtime" gorm:"default:0"`                      // 本地源文件修改时间，用于源文件清理前校验
	LocalMtimeNs          int64                     `json:"local_mtime_ns" gorm:"default:0"`                   // 本地源文件纳秒级修改时间
	UploadedBytes         int64                     `json:"uploaded_bytes" gorm:"default:0"`                   // 已上传字节数
	UploadResult          UploadResult              `json:"upload_result" gorm:"size:32;default:unknown"`      // 上传结果
	ResumeState           UploadResumeState         `json:"resume_state" gorm:"size:32;default:none"`          // 断点续传状态
	RapidWaitAttempts     int                       `json:"rapid_wait_attempts" gorm:"default:0"`              // 秒传等待已尝试次数
	RapidWaitUntil        int64                     `json:"rapid_wait_until" gorm:"default:0"`                 // 秒传等待截止时间
	CompletedRemoteFileId string                    `json:"completed_remote_file_id" gorm:"size:128"`          // 完成后的远端文件 ID
	CompletedPickCode     string                    `json:"completed_pick_code" gorm:"size:128"`               // 完成后的 pickcode
	Error                 string                    `json:"error"`                                             // 错误信息
	StartTime             int64                     `json:"start_time"`                                        // 开始时间
	EndTime               int64                     `json:"end_time"`                                          // 结束时间
	RetryCount            int                       `json:"retry_count" gorm:"default:0"`                      // 已重试次数
	LastRetryTime         int64                     `json:"last_retry_time" gorm:"default:0"`                  // 最近重试时间
	SourceCleanupStatus   UploadSourceCleanupStatus `json:"source_cleanup_status" gorm:"size:32;default:none"` // 源文件清理状态
	SourceCleanupError    string                    `json:"source_cleanup_error" gorm:"type:text"`             // 源文件清理错误
	SourceDeletedAt       int64                     `json:"source_deleted_at" gorm:"default:0"`                // 源文件删除时间
	IsSeasonOrTvshowFile  bool                      `json:"is_season_or_tvshow_file"`                          // 是否是剧集或电视剧文件
	SyncFile              *SyncFile                 `json:"-" gorm:"-"`                                        // 同步文件
	ScrapeMediaFile       *ScrapeMediaFile          `json:"-" gorm:"-"`                                        // 刮削文件
	Account               *Account                  `json:"-" gorm:"-"`                                        // 账户
	UploadPhase           string                    `json:"upload_phase" gorm:"-"`                             // 上传阶段，仅用于队列展示
	UploadSpeedBytes      int64                     `json:"upload_speed_bytes" gorm:"-"`                       // 上传速度，仅用于队列展示
	ProgressPercent       float64                   `json:"progress_percent" gorm:"-"`                         // 上传进度，仅用于队列展示
	TotalParts            int                       `json:"total_parts" gorm:"-"`                              // 总分片数，仅用于队列展示
	UploadedParts         int                       `json:"uploaded_parts" gorm:"-"`                           // 已上传分片数，仅用于队列展示
}

// String 返回状态的字符串表示
func (s UploadStatus) String() string {
	switch s {
	case UploadStatusPending:
		return "pending"
	case UploadStatusUploading:
		return "uploading"
	case UploadStatusCompleted:
		return "completed"
	case UploadStatusFailed:
		return "failed"
	case UploadStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// CanRetry 判断上传任务是否还能重试
func (task *DbUploadTask) CanRetry(maxRetry int) bool {
	return task != nil && task.Status == UploadStatusFailed && task.RetryCount < maxRetry
}

var uploadQueueProgressBroadcast = struct {
	sync.Mutex
	lastAt map[uint]time.Time
}{
	lastAt: make(map[uint]time.Time),
}

func publishUploadQueueChanged(task *DbUploadTask, reason string) {
	payload := ws.QueueChangedPayload{Reason: reason}
	if task != nil {
		if reason == "progress" && shouldThrottleUploadProgressBroadcast(task.ID) {
			return
		}
		task.applyUploadQueueDisplayFields(nil)
		payload.TaskID = task.ID
		payload.Status = int(task.Status)
		payload.Source = string(task.Source)
		payload.UploadedBytes = task.UploadedBytes
		payload.FileSize = task.FileSize
		payload.ProgressPercent = task.ProgressPercent
		payload.UploadSpeedBytes = task.UploadSpeedBytes
		payload.UploadPhase = task.UploadPhase
		payload.UploadResult = string(task.UploadResult)
		payload.ResumeState = string(task.ResumeState)
		payload.RapidWaitUntil = task.RapidWaitUntil
		payload.TotalParts = task.TotalParts
		payload.UploadedParts = task.UploadedParts
	}
	if reason == "progress" {
		ws.TryBroadcastEvent(ws.EventUploadQueueChanged, payload)
		return
	}
	ws.BroadcastQueueChanged(ws.EventUploadQueueChanged, payload)
}

func shouldThrottleUploadProgressBroadcast(taskID uint) bool {
	if taskID == 0 {
		return false
	}
	now := time.Now()
	uploadQueueProgressBroadcast.Lock()
	defer uploadQueueProgressBroadcast.Unlock()
	lastAt := uploadQueueProgressBroadcast.lastAt[taskID]
	if !lastAt.IsZero() && now.Sub(lastAt) < time.Second {
		return true
	}
	uploadQueueProgressBroadcast.lastAt[taskID] = now
	return false
}

// PrepareUploadRetry 将上传失败任务重新放回等待中
func (task *DbUploadTask) PrepareUploadRetry(maxRetry int) {
	if !task.CanRetry(maxRetry) {
		return
	}
	task.Status = UploadStatusPending
	task.Error = ""
	task.RetryCount++
	task.LastRetryTime = time.Now().Unix()
}

func (task *DbUploadTask) Complete() {
	// 标记为已完成
	task.Status = UploadStatusCompleted
	task.EndTime = time.Now().Unix()
	err := db.Db.Save(task).Error
	if err != nil {
		helpers.AppLogger.Warnf("[上传] 标记为已完成失败：%s", err.Error())
		return
	}
	publishUploadQueueChanged(task, "status_changed")
}

func (task *DbUploadTask) Fail(err error) {
	// 标记为失败
	task.Status = UploadStatusFailed
	task.EndTime = time.Now().Unix()
	task.Error = err.Error()
	err = db.Db.Save(task).Error
	if err != nil {
		helpers.AppLogger.Warnf("[上传] 标记为失败失败：%s", err.Error())
		return
	}
	publishUploadQueueChanged(task, "status_changed")
}

func (task *DbUploadTask) Cancel() {
	// 标记为已取消
	task.Status = UploadStatusCancelled
	task.EndTime = time.Now().Unix()
	err := db.Db.Save(task).Error
	if err != nil {
		helpers.AppLogger.Warnf("[上传] 标记为已取消失败：%s", err.Error())
		return
	}
	publishUploadQueueChanged(task, "status_changed")
}

func (task *DbUploadTask) Uploading() {
	task.Status = UploadStatusUploading
	task.StartTime = time.Now().Unix()
	err := db.Db.Save(task).Error
	if err != nil {
		helpers.AppLogger.Warnf("[上传] 标记为上传中失败：%s", err.Error())
		return
	}
	publishUploadQueueChanged(task, "status_changed")
}

func (task *DbUploadTask) GetAccount() *Account {
	if task.Account != nil {
		return task.Account
	}
	// 通过 AccountId 查询账户，然后判断是什么来源
	account, err := GetAccountById(task.AccountId)
	if err != nil {
		task.Fail(err)
		return nil
	}
	task.Account = account
	return account
}

// 执行上传
func (task *DbUploadTask) Upload() {
	if !helpers.PathExists(task.LocalFullPath) {
		task.Fail(fmt.Errorf("本地文件 %s 不存在", task.LocalFullPath))
		return
	}
	switch task.SourceType {
	case SourceType115:
		if !task.Upload115File() {
			return
		}
	case SourceTypeOpenList:
		if !task.UploadOpenListFile() {
			return
		}
	case SourceTypeLocal:
		if !task.UploadLocalFile() {
			return
		}
	case SourceTypeBaiduPan:
		if !task.UploadBaiduPanFile() {
			return
		}
	default:
		task.Fail(fmt.Errorf("未知的上传来源类型 %s", task.SourceType))
		return
	}
	// 标记为已完成
	task.Complete()
	if err := task.markDirectoryUploadProcessedAfterUploadComplete(); err != nil {
		helpers.AppLogger.Warnf("[上传] 标记目录监控源文件上传完成失败：%s", err.Error())
	}
	if err := task.enqueueStrmGenerationAfterUploadAndMarkDirectoryProcessed(); err != nil {
		helpers.AppLogger.Warnf("[上传] 创建 STRM 生成任务失败：%s", err.Error())
	}
	// 如果是刮削类型，需要进行后续通知
	if task.Source == UploadSourceScrape {
		// 通知刮削整理完成
		scrapeMediaFile := GetScrapeMediaFileById(task.ScrapeMediaFileId)
		if scrapeMediaFile == nil {
			helpers.AppLogger.Errorf("刮削文件 %d 不存在", task.ScrapeMediaFileId)
			return
		}
		scrapeMediaFile.RemoveTmpFiles(task)
	}
}

func (task *DbUploadTask) Upload115File() bool {
	// 检查账户是否存在
	account := task.GetAccount()
	if account == nil {
		task.Fail(fmt.Errorf("账户 %d 不存在", task.AccountId))
		return false
	}
	// 上传文件
	client := account.Get115Client()
	if client == nil {
		task.Fail(fmt.Errorf("账户 %s 115 客户端不存在", account.Name))
		return false
	}
	task.Uploading()
	result, err := currentUpload115Runner.Upload(context.Background(), task, client)
	if err != nil {
		task.Fail(fmt.Errorf("调用 115 上传 API 失败：%v", err))
		return false
	}
	task.applyUpload115TaskResult(result)
	if task.Source == UploadSourceStrm {
		if result.CompletedMtime > 0 {
			mtime := time.Unix(result.CompletedMtime, 0)
			if err = os.Chtimes(task.LocalFullPath, mtime, mtime); err != nil {
				task.Fail(fmt.Errorf("更新本地文件 %s 修改时间失败：%v", task.LocalFullPath, err))
				return false
			}
		}
	}
	return true
}

// 百度网盘上传文件
func (task *DbUploadTask) UploadBaiduPanFile() bool {
	// 检查账户是否存在
	account := task.GetAccount()
	if account == nil {
		task.Fail(fmt.Errorf("账户 %d 不存在", task.AccountId))
		return false
	}
	// 上传文件
	client := account.GetBaiDuPanClient()
	if client == nil {
		task.Fail(fmt.Errorf("账户 %s 百度网盘客户端不存在", account.Name))
		return false
	}
	task.Uploading()
	// 调用上传方法
	resp, err := client.Upload(context.Background(), task.LocalFullPath, task.RemoteFileId)
	if err != nil {
		task.Fail(fmt.Errorf("百度网盘上传文件 %s 失败：%v", task.FileName, err))
		return false
	}
	if task.Source == UploadSourceStrm {
		t := time.Unix(int64(*resp.Mtime), 0)
		// 更新本地文件的修改时间
		err = os.Chtimes(task.LocalFullPath, t, t)
		if err != nil {
			task.Fail(fmt.Errorf("更新本地文件 %s 修改时间失败：%v", task.LocalFullPath, err))
			return false
		}
	}
	return true
}

func (task *DbUploadTask) UploadOpenListFile() bool {
	// 检查账户是否存在
	account := task.GetAccount()
	if account == nil {
		task.Fail(fmt.Errorf("账户 %d 不存在", task.AccountId))
		return false
	}
	// 上传文件
	client := account.GetOpenListClient()
	if client == nil {
		task.Fail(fmt.Errorf("账户 %s OpenList 客户端不存在", account.Name))
		return false
	}
	task.Uploading()
	_, err := client.Upload(task.LocalFullPath, task.RemoteFileId)
	if err != nil {
		task.Fail(fmt.Errorf("OpenList 上传文件 %s 失败：%v", task.FileName, err))
		return false
	}
	if task.Source == UploadSourceStrm {
		// 查询文件详情
		detail, err := client.FileDetail(task.RemoteFileId)
		if err != nil {
			task.Fail(fmt.Errorf("OpenList 查询文件详情 %s 失败：%s", task.RemoteFileId, err.Error()))
			return false
		}
		// 将 ISO 8601 格式的日期字符串转换为时间戳
		t, err := time.Parse(time.RFC3339, detail.Modified)
		if err != nil {
			helpers.AppLogger.Warnf("解析时间格式失败：%v, 时间字符串：%s", err, detail.Modified)
			return true
		}
		// 更新本地文件的修改时间
		err = os.Chtimes(task.LocalFullPath, t, t)
		if err != nil {
			task.Fail(fmt.Errorf("更新本地文件 %s 修改时间失败：%v", task.LocalFullPath, err))
			return false
		}
	}
	return true
}

func (task *DbUploadTask) UploadLocalFile() bool {
	task.Uploading()
	err := helpers.CopyFile(task.LocalFullPath, task.RemoteFileId)
	if err != nil {
		task.Fail(fmt.Errorf("本地文件 %s 复制到 %s 失败：%v", task.LocalFullPath, task.RemoteFileId, err))
		return false
	}
	return true
}

func CheckCanUploadByLocalPath(source UploadSource, localPath string) bool {
	var total int64
	err := db.Db.Model(&DbUploadTask{}).
		Where("source = ? AND local_full_path = ? AND status IN ?", source, localPath, []UploadStatus{UploadStatusPending, UploadStatusUploading}).
		Count(&total).Error
	if err != nil {
		helpers.AppLogger.Warnf("检查本地路径上传任务失败：source=%s, local_path=%s, err=%v", source, localPath, err)
		return true
	}
	return total == 0
}

// 检查任务是否已经存在，通过 Source + RemoteFileId
func CheckUploadTaskExist(source UploadSource, remoteFileId string) *DbUploadTask {
	var task *DbUploadTask
	err := db.Db.Model(&DbUploadTask{}).
		Where("source = ? AND remote_file_id = ?", source, remoteFileId).
		First(&task).Error
	if err != nil {
		return nil
	}
	return task
}

// AddDirectoryMonitorUploadTask 添加目录监控产生的上传任务。
func AddDirectoryMonitorUploadTask(task *DbUploadTask) error {
	if err := SaveDirectoryMonitorUploadTaskWithDB(db.Db, task); err != nil {
		return err
	}
	PublishUploadTaskCreated(task)
	return nil
}

// SaveDirectoryMonitorUploadTaskWithDB 在指定事务中保存目录监控上传任务。
func SaveDirectoryMonitorUploadTaskWithDB(tx *gorm.DB, task *DbUploadTask) error {
	if tx == nil {
		return errors.New("数据库连接为空")
	}
	if task == nil {
		return errors.New("上传任务为空")
	}
	task.Source = UploadSourceDirectoryMonitor
	if task.Status == 0 {
		task.Status = UploadStatusPending
	}
	if task.UploadResult == "" {
		task.UploadResult = UploadResultUnknown
	}
	if task.SourceCleanupStatus == "" {
		task.SourceCleanupStatus = UploadSourceCleanupStatusNone
	}
	return tx.Save(task).Error
}

// PublishUploadTaskCreated 发布上传任务创建事件。
func PublishUploadTaskCreated(task *DbUploadTask) {
	publishUploadQueueChanged(task, "created")
}

// 添加 STRM 同步产生的上传任务
func AddUploadTaskFromSyncFile(file *SyncFile) error {
	// 先检查是否存在
	if task := CheckUploadTaskExist(UploadSourceStrm, file.FileId); task != nil {
		if task.Status == UploadStatusPending {
			return errors.New("任务已存在，状态为待上传")
		}
		if task.Status == UploadStatusUploading {
			return errors.New("任务已存在，状态为上传中")
		}
	}
	// if file.SyncPath == nil {
	// 	file.SyncPath = GetSyncPathById(file.SyncPathId)
	// }
	remoteFileId := file.FileId
	// if file.SourceType == SourceType115 {
	// 	remoteFileId = filepath.Join(file.Path, file.FileName)
	// }
	// 插入新纪录
	task := &DbUploadTask{
		AccountId:     file.AccountId,
		SourceType:    file.SourceType,
		SyncFileId:    file.ID,
		SyncPathId:    file.SyncPathId,
		RemoteFileId:  remoteFileId,
		FileName:      file.FileName,
		RemotePathId:  file.ParentId,
		LocalFullPath: file.LocalFilePath,
		Source:        UploadSourceStrm,
		Status:        UploadStatusPending,
		FileSize:      file.FileSize,
	}
	err := db.Db.Save(task).Error
	if err != nil {
		helpers.AppLogger.Errorf("添加上传任务 %s => %s 失败：%s", file.LocalFilePath, remoteFileId, err.Error())
		return err
	}
	helpers.AppLogger.Infof("添加上传任务 %s => %s 成功", file.LocalFilePath, remoteFileId)
	publishUploadQueueChanged(task, "created")
	return nil
}

// 添加刮削整理产生的上传任务
func AddUploadTaskFromMediaFile(mediaFile *ScrapeMediaFile, scrapePath *ScrapePath, fileName, localFullPath, remoteFileId, remotePathId string, isSeasonOrTvshowFile bool) error {
	stat, err := os.Stat(localFullPath)
	if err != nil {
		helpers.AppLogger.Errorf("要上传的文件 %s 无法获取到文件信息，错误：%vs", localFullPath, err.Error())
		return err
	}
	size := stat.Size()
	// 先检查是否存在
	if task := CheckUploadTaskExist(UploadSourceScrape, remoteFileId); task != nil {
		if task.Status == UploadStatusPending {
			return errors.New("任务已存在，状态为待上传")
		}
		if task.Status == UploadStatusUploading {
			return errors.New("任务已存在，状态为上传中")
		}
	}
	// 插入新纪录
	task := &DbUploadTask{
		AccountId:            scrapePath.AccountId,
		ScrapeMediaFileId:    mediaFile.ID,
		SourceType:           scrapePath.SourceType,
		RemoteFileId:         remoteFileId,
		FileName:             fileName,
		RemotePathId:         remotePathId,
		LocalFullPath:        localFullPath,
		Source:               UploadSourceScrape,
		Status:               UploadStatusPending,
		FileSize:             size,
		IsSeasonOrTvshowFile: isSeasonOrTvshowFile,
	}
	derr := db.Db.Save(task).Error
	if derr == nil {
		publishUploadQueueChanged(task, "created")
	}
	return derr
}

func GetPendingUploadTasks(limit int) []*DbUploadTask {
	var tasks []*DbUploadTask
	db.Db.Model(&DbUploadTask{}).
		Where("status = ?", UploadStatusPending).
		Limit(limit).
		Order("id ASC").
		Find(&tasks)
	return tasks
}

func GetUploadingCount() int64 {
	var count int64
	db.Db.Model(&DbUploadTask{}).
		Where("status = ?", UploadStatusUploading).
		Count(&count)
	return count
}

// 查询上传队列任务列表
func GetUploadTaskList(status UploadStatus, page, pageSize int) ([]*DbUploadTask, int64) {
	var tasks []*DbUploadTask
	var total int64
	tx := db.Db.Model(&DbUploadTask{})
	if status >= 0 {
		tx.Where("status = ?", status)
	}
	tx.Count(&total).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Order("id DESC").
		Find(&tasks)
	hydrateUploadTaskQueueFields(tasks)
	return tasks, total
}

func ClearPendingUploadTasks() error {
	err := db.Db.Model(&DbUploadTask{}).
		Where("status = ?", UploadStatusPending).
		Delete(&DbUploadTask{}).Error
	if err != nil {
		helpers.AppLogger.Errorf("清除待上传任务失败：%v", err)
		return err
	}
	publishUploadQueueChanged(nil, "clear_pending")
	return err
}

func ClearExpireUploadTasks() error {
	err := db.Db.Model(&DbUploadTask{}).
		Where("created_at < ?", time.Now().AddDate(0, 0, -3).Unix()).
		Delete(&DbUploadTask{}).Error
	if err != nil {
		helpers.AppLogger.Errorf("清除 3 天前的上传任务失败：%v", err)
		return err
	} else {
		helpers.AppLogger.Infof("已清除 3 天前的上传任务")
	}
	return err
}

func ClearUploadSuccessAndFailed() error {
	err := db.Db.Model(&DbUploadTask{}).
		Where("status IN (?, ?)", UploadStatusCompleted, UploadStatusFailed).
		Delete(&DbUploadTask{}).Error
	if err != nil {
		helpers.AppLogger.Errorf("清除上传成功和失败任务失败：%v", err)
		return err
	} else {
		helpers.AppLogger.Infof("清除上传成功和失败任务成功")
	}
	publishUploadQueueChanged(nil, "clear_success_failed")
	return err
}

func UpdateUploadingToPending() error {
	err := db.Db.Model(&DbUploadTask{}).
		Where("status = ?", UploadStatusUploading).
		Update("status", UploadStatusPending).Error
	if err != nil {
		helpers.AppLogger.Errorf("更新上传中的任务为待上传失败：%v", err)
		return err
	} else {
		helpers.AppLogger.Infof("更新上传中的任务为待上传成功")
	}
	return err
}

func RetryFailedUploadTasks(maxRetry int) error {
	updateData := map[string]interface{}{
		"status":          UploadStatusPending,
		"error":           "",
		"retry_count":     gorm.Expr("retry_count + 1"),
		"last_retry_time": time.Now().Unix(),
	}
	err := db.Db.Model(&DbUploadTask{}).
		Where("status = ? AND retry_count < ?", UploadStatusFailed, maxRetry).
		Updates(updateData).Error
	if err != nil {
		helpers.AppLogger.Errorf("重试失败的上传任务失败：%v", err)
		return err
	}
	helpers.AppLogger.Infof("重试失败的上传任务成功")
	publishUploadQueueChanged(nil, "retry_failed")
	return nil
}

func GetUnFinishUploadTaskCountByScrapeMediaId(scrapeMediaFileId uint) int64 {
	var count int64
	db.Db.Model(&DbUploadTask{}).
		Where("scrape_media_file_id = ? AND status IN (?, ?)", scrapeMediaFileId, UploadStatusPending, UploadStatusUploading).
		Count(&count)
	return count
}
