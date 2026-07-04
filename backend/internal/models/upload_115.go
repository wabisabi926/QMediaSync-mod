package models

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/v115open"

	"gorm.io/gorm"
)

const (
	uploadPhasePending            = "pending"
	uploadPhaseCheckingRemote     = "checking_remote"
	uploadPhaseRapidWaiting       = "rapid_waiting"
	uploadPhaseMultipartUploading = "multipart_uploading"
	uploadPhaseCompleting         = "completing"
	uploadPhaseCompleted          = "completed"
	uploadPhaseRapidUploaded      = "rapid_uploaded"
	uploadPhaseRemoteExists       = "remote_exists"
	uploadPhaseSkipped            = "skipped"
	uploadPhaseFailed             = "failed"
	uploadPhaseCancelled          = "cancelled"
)

type upload115Runner interface {
	Upload(context.Context, *DbUploadTask, *v115open.OpenClient) (upload115TaskResult, error)
}

type open115UploadRunner struct{}

type testingCleanup interface {
	Cleanup(func())
}

var currentUpload115Runner upload115Runner = open115UploadRunner{}

func setUpload115RunnerForTesting(t testingCleanup, runner upload115Runner) {
	oldRunner := currentUpload115Runner
	currentUpload115Runner = runner
	t.Cleanup(func() {
		currentUpload115Runner = oldRunner
	})
}

type upload115LocalFileInfo struct {
	FileName       string
	FileSize       int64
	LocalMtime     int64
	LocalSignature string
	FileSha1       string
	Preid          string
}

type upload115TaskResult struct {
	UploadResult          UploadResult
	ResumeState           UploadResumeState
	UploadedBytes         int64
	TotalParts            int
	UploadedParts         int
	CompletedRemoteFileId string
	CompletedPickCode     string
	CompletedParentId     string
	CompletedSha1         string
	CompletedSize         int64
	CompletedMtime        int64
}

func (runner open115UploadRunner) Upload(ctx context.Context, task *DbUploadTask, client *v115open.OpenClient) (upload115TaskResult, error) {
	if task == nil {
		return upload115TaskResult{}, errors.New("上传任务为空")
	}
	if client == nil {
		return upload115TaskResult{}, errors.New("115 客户端为空")
	}
	info, err := buildUpload115LocalFileInfo(task.LocalFullPath, task.FileName)
	if err != nil {
		return upload115TaskResult{}, err
	}
	task.FileName = info.FileName
	task.FileSize = info.FileSize
	task.publish115UploadPhase(nil, uploadPhaseCheckingRemote)

	if result, ok := runner.findExistingRemoteFile(ctx, task, client, info); ok {
		return result, nil
	}

	parentDetail, err := client.GetFsDetailByCid(ctx, task.RemotePathId)
	if err != nil {
		return upload115TaskResult{}, fmt.Errorf("115 检查父目录 %s 失败：%w", task.RemotePathId, err)
	}
	if parentDetail.FileId == "" {
		return upload115TaskResult{}, fmt.Errorf("115 检查父目录 %s 失败：返回空文件 ID", task.RemotePathId)
	}

	session, err := task.prepare115UploadSession(info)
	if err != nil {
		return upload115TaskResult{}, err
	}

	if session.ResumeState == UploadResumeStateResumedSession && session.UploadId != "" && session.PickCode != "" {
		resumeResult, resumeErr := client.UploadResume(ctx, session.PickCode, info.FileSize, task.RemotePathId, info.FileSha1)
		if resumeErr == nil {
			apply115ResumeResultToSession(session, resumeResult)
			if err := session.Save(); err != nil {
				return upload115TaskResult{}, fmt.Errorf("保存 115 续传调度信息失败：%w", err)
			}
			return runner.uploadMultipart(ctx, task, client, session, info)
		}
		helpers.V115Log.Warnf("115 续传调度失败，将重新初始化上传任务 %d：%v", task.ID, resumeErr)
		task.mark115SessionRestarted(session, resumeErr)
	}

	return runner.uploadByInit(ctx, task, client, session, info)
}

func (runner open115UploadRunner) findExistingRemoteFile(
	ctx context.Context,
	task *DbUploadTask,
	client *v115open.OpenClient,
	info upload115LocalFileInfo,
) (upload115TaskResult, bool) {
	if strings.TrimSpace(task.RemoteFileId) == "" {
		return upload115TaskResult{}, false
	}
	detail, err := client.GetFsDetailByPath(ctx, task.RemoteFileId)
	if err != nil || detail == nil || detail.FileId == "" {
		return upload115TaskResult{}, false
	}
	sameSHA1 := strings.EqualFold(detail.Sha1, info.FileSha1)
	sameSize := detail.FileSizeByte == info.FileSize
	if !sameSHA1 || !sameSize {
		return upload115TaskResult{}, false
	}
	mtime := helpers.StringToInt64(detail.Ptime)
	if mtime == 0 {
		mtime = helpers.StringToInt64(detail.Utime)
	}
	return upload115TaskResult{
		UploadResult:          UploadResultRemoteExists,
		ResumeState:           UploadResumeStateNone,
		UploadedBytes:         info.FileSize,
		CompletedRemoteFileId: detail.FileId,
		CompletedPickCode:     detail.PickCode,
		CompletedParentId:     task.RemotePathId,
		CompletedSha1:         detail.Sha1,
		CompletedSize:         detail.FileSizeByte,
		CompletedMtime:        mtime,
	}, true
}

func (runner open115UploadRunner) uploadByInit(
	ctx context.Context,
	task *DbUploadTask,
	client *v115open.OpenClient,
	session *UploadSession,
	info upload115LocalFileInfo,
) (upload115TaskResult, error) {
	request := v115open.UploadInitRequest{
		FileName:     info.FileName,
		FileSize:     info.FileSize,
		ParentFileId: task.RemotePathId,
		FileSha1:     info.FileSha1,
		Preid:        info.Preid,
		TopUpload:    "0",
	}
	initResult, err := client.UploadInit(ctx, request)
	if err != nil {
		return upload115TaskResult{}, fmt.Errorf("上传初始化失败：%w", err)
	}
	if initResult.Status == v115open.UploadInitStatusNeedSign {
		signVal, err := v115open.SignValueForRange(task.LocalFullPath, initResult.SignCheck)
		if err != nil {
			return upload115TaskResult{}, err
		}
		request.SignKey = initResult.SignKey
		request.SignVal = signVal
		session.SignKey = initResult.SignKey
		session.SignValSha1 = signVal
		initResult, err = client.UploadInit(ctx, request)
		if err != nil {
			return upload115TaskResult{}, fmt.Errorf("上传二次认证失败：%w", err)
		}
	}
	apply115InitResultToSession(session, initResult)
	if err := session.Save(); err != nil {
		return upload115TaskResult{}, fmt.Errorf("保存 115 初始化调度信息失败：%w", err)
	}

	policy := uploadRapidWaitPolicyFromSettings()
	if initResult.Status == v115open.UploadInitStatusNeedUpload && policy.Enabled {
		session.Status = UploadSessionStatusRapidWaiting
		session.RapidWaitUntil = time.Now().Add(policy.Timeout).Unix()
		if err := session.Save(); err != nil {
			return upload115TaskResult{}, fmt.Errorf("保存秒传等待状态失败：%w", err)
		}
		task.publish115UploadPhase(session, uploadPhaseRapidWaiting)
		outcome, err := v115open.WaitForRapidUpload(ctx, initResult, policy, info.FileSize, func(ctx context.Context) (*v115open.UploadInitResult, error) {
			return client.UploadInit(ctx, request)
		}, nil)
		if err != nil {
			return upload115TaskResult{}, err
		}
		session.RapidWaitAttempts += outcome.Attempts
		task.RapidWaitAttempts = session.RapidWaitAttempts
		initResult = outcome.Result
		apply115InitResultToSession(session, initResult)
		if err := session.Save(); err != nil {
			return upload115TaskResult{}, fmt.Errorf("保存秒传等待结果失败：%w", err)
		}
		if outcome.TimedOut && outcome.SkipUpload {
			return upload115TaskResult{
				UploadResult: UploadResultSkippedAfterRapidWait,
				ResumeState:  session.ResumeState,
			}, nil
		}
	}

	switch initResult.Status {
	case v115open.UploadInitStatusRapidUploaded:
		complete := UploadSessionCompleteResult{
			FileId:   initResult.FileId,
			PickCode: initResult.PickCode,
			ParentId: task.RemotePathId,
			Sha1:     info.FileSha1,
			Size:     info.FileSize,
		}
		if err := session.MarkCompleted(complete); err != nil {
			return upload115TaskResult{}, fmt.Errorf("保存秒传完成状态失败：%w", err)
		}
		return upload115TaskResult{
			UploadResult:          UploadResultRapidUpload,
			ResumeState:           session.ResumeState,
			UploadedBytes:         info.FileSize,
			CompletedRemoteFileId: complete.FileId,
			CompletedPickCode:     complete.PickCode,
			CompletedParentId:     complete.ParentId,
			CompletedSha1:         complete.Sha1,
			CompletedSize:         complete.Size,
		}, nil
	case v115open.UploadInitStatusNeedUpload:
		return runner.uploadMultipart(ctx, task, client, session, info)
	case v115open.UploadInitStatusSignFailed:
		return upload115TaskResult{}, errors.New("签名验证后失败")
	case v115open.UploadInitStatusSignRejected:
		return upload115TaskResult{}, errors.New("签名认证失败")
	default:
		return upload115TaskResult{}, fmt.Errorf("未知的 115 上传初始化状态：%d", initResult.Status)
	}
}

func (runner open115UploadRunner) uploadMultipart(
	ctx context.Context,
	task *DbUploadTask,
	client *v115open.OpenClient,
	session *UploadSession,
	info upload115LocalFileInfo,
) (upload115TaskResult, error) {
	session.Status = UploadSessionStatusMultipart
	if err := session.Save(); err != nil {
		return upload115TaskResult{}, fmt.Errorf("保存 multipart 状态失败：%w", err)
	}
	task.publish115UploadPhase(session, uploadPhaseMultipartUploading)

	ossResult, err := client.UploadMultipartWithResult(ctx, v115open.UploadMultipartInput{
		Bucket:      session.Bucket,
		Object:      session.Object,
		Callback:    session.Callback,
		CallbackVar: session.CallbackVar,
		FilePath:    task.LocalFullPath,
		FileSize:    info.FileSize,
		FileSha1:    info.FileSha1,
		UploadId:    session.UploadId,
		PartSize:    session.PartSize,
		OnProgress: func(progress v115open.OSSMultipartProgress) {
			task.save115UploadProgress(session, progress)
		},
	})
	if err != nil {
		session.Status = UploadSessionStatusFailed
		session.LastError = err.Error()
		_ = session.Save()
		return upload115TaskResult{}, err
	}

	session.Status = UploadSessionStatusCompleting
	session.UploadId = ossResult.UploadId
	session.PartSize = ossResult.PartSize
	session.TotalParts = ossResult.TotalParts
	session.UploadedBytes = ossResult.UploadedBytes
	session.UploadedParts = ossResult.UploadedParts
	if err := session.Save(); err != nil {
		return upload115TaskResult{}, fmt.Errorf("保存 multipart 完成前状态失败：%w", err)
	}
	task.publish115UploadPhase(session, uploadPhaseCompleting)

	completeResult, err := v115open.ParseCompleteCallbackResult(ossResult.CallbackResult)
	if err != nil {
		_ = session.MarkCompleteCallbackFailed(err)
		return upload115TaskResult{}, err
	}
	if err := session.MarkCompleted(UploadSessionCompleteResult{
		FileId:   completeResult.FileId,
		PickCode: completeResult.PickCode,
		ParentId: completeResult.ParentId,
		Sha1:     completeResult.Sha1,
		Size:     completeResult.Size,
		Mtime:    completeResult.Mtime,
	}); err != nil {
		return upload115TaskResult{}, fmt.Errorf("保存上传完成状态失败：%w", err)
	}
	return upload115TaskResult{
		UploadResult:          UploadResultMultipartUploaded,
		ResumeState:           session.ResumeState,
		UploadedBytes:         ossResult.UploadedBytes,
		TotalParts:            ossResult.TotalParts,
		UploadedParts:         ossResult.UploadedParts,
		CompletedRemoteFileId: completeResult.FileId,
		CompletedPickCode:     completeResult.PickCode,
		CompletedParentId:     completeResult.ParentId,
		CompletedSha1:         completeResult.Sha1,
		CompletedSize:         completeResult.Size,
		CompletedMtime:        completeResult.Mtime,
	}, nil
}

func buildUpload115LocalFileInfo(filePath string, preferredFileName string) (upload115LocalFileInfo, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return upload115LocalFileInfo{}, fmt.Errorf("获取文件信息失败：%w", err)
	}
	fileName := preferredFileName
	if strings.TrimSpace(fileName) == "" {
		fileName = fileInfo.Name()
	}
	fileSha1, err := helpers.FileSHA1(filePath)
	if err != nil {
		return upload115LocalFileInfo{}, fmt.Errorf("计算文件 SHA1 失败：%w", err)
	}
	preid, err := helpers.FileSHA1Partial(filePath, 0, 128*1024-1)
	if err != nil {
		return upload115LocalFileInfo{}, fmt.Errorf("计算文件前 128 KiB SHA1 失败：%w", err)
	}
	localMtime := fileInfo.ModTime().Unix()
	localSignature := fmt.Sprintf("%d:%d:%s:%s", fileInfo.Size(), localMtime, fileSha1, preid)
	return upload115LocalFileInfo{
		FileName:       fileName,
		FileSize:       fileInfo.Size(),
		LocalMtime:     localMtime,
		LocalSignature: localSignature,
		FileSha1:       fileSha1,
		Preid:          preid,
	}, nil
}

func (task *DbUploadTask) prepare115UploadSession(info upload115LocalFileInfo) (*UploadSession, error) {
	if task == nil {
		return nil, errors.New("上传任务为空")
	}
	signature := UploadSessionLocalSignature{
		FileSize:       info.FileSize,
		LocalMtime:     info.LocalMtime,
		FileSha1:       info.FileSha1,
		LocalSignature: info.LocalSignature,
	}
	session, err := GetUploadSessionByUploadTaskId(task.ID)
	if err == nil {
		if validateErr := session.ValidateLocalFile(signature); validateErr != nil {
			session.Status = UploadSessionStatusAborted
			session.LastError = fmt.Sprintf("本地文件已变化，不能复用断点续传 session：%v", validateErr)
			if saveErr := session.Save(); saveErr != nil {
				return nil, fmt.Errorf("保存废弃上传会话失败：%w", saveErr)
			}
			task.ResumeState = UploadResumeStateSessionExpiredRestarted
			_ = db.Db.Model(task).Updates(map[string]any{
				"resume_state": task.ResumeState,
			}).Error
			return nil, fmt.Errorf("本地文件已变化，不能复用断点续传 session：%w", validateErr)
		}
		session.ResumeState = UploadResumeStateResumedSession
		session.LastResumeAt = time.Now().Unix()
		if err := session.Save(); err != nil {
			return nil, fmt.Errorf("保存续传会话状态失败：%w", err)
		}
		task.ResumeState = session.ResumeState
		task.UploadedBytes = session.UploadedBytes
		task.RapidWaitAttempts = session.RapidWaitAttempts
		task.RapidWaitUntil = session.RapidWaitUntil
		_ = db.Db.Model(task).Updates(map[string]any{
			"resume_state":        task.ResumeState,
			"uploaded_bytes":      task.UploadedBytes,
			"rapid_wait_attempts": task.RapidWaitAttempts,
			"rapid_wait_until":    task.RapidWaitUntil,
		}).Error
		return session, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	session = &UploadSession{
		UploadTaskId:   task.ID,
		AccountId:      task.AccountId,
		LocalFullPath:  task.LocalFullPath,
		FileName:       info.FileName,
		FileSize:       info.FileSize,
		LocalMtime:     info.LocalMtime,
		LocalSignature: info.LocalSignature,
		FileSha1:       info.FileSha1,
		Preid:          info.Preid,
		ParentFileId:   task.RemotePathId,
		Target:         fmt.Sprintf("U_1_%s", task.RemotePathId),
		Status:         UploadSessionStatusInit,
		ResumeState:    UploadResumeStateNewSession,
		LastInitAt:     time.Now().Unix(),
	}
	if err := session.Save(); err != nil {
		return nil, fmt.Errorf("保存新上传会话失败：%w", err)
	}
	task.ResumeState = session.ResumeState
	task.UploadedBytes = 0
	_ = db.Db.Model(task).Updates(map[string]any{
		"resume_state":   task.ResumeState,
		"uploaded_bytes": task.UploadedBytes,
		"file_size":      info.FileSize,
		"file_name":      info.FileName,
	}).Error
	return session, nil
}

func apply115InitResultToSession(session *UploadSession, result *v115open.UploadInitResult) {
	if session == nil || result == nil {
		return
	}
	session.PickCode = result.PickCode
	session.FileId = result.FileId
	session.Target = result.Target
	session.Bucket = result.Bucket
	session.Object = result.Object
	session.SignKey = result.SignKey
	session.Callback = result.Callback.Callback
	session.CallbackVar = result.Callback.CallbackVar
	session.LastInitAt = time.Now().Unix()
}

func apply115ResumeResultToSession(session *UploadSession, result *v115open.UploadResumeResult) {
	if session == nil || result == nil {
		return
	}
	session.PickCode = result.PickCode
	session.Target = result.Target
	session.Bucket = result.Bucket
	session.Object = result.Object
	session.Callback = result.Callback.Callback
	session.CallbackVar = result.Callback.CallbackVar
	session.LastResumeAt = time.Now().Unix()
}

func (task *DbUploadTask) mark115SessionRestarted(session *UploadSession, cause error) {
	if session == nil {
		return
	}
	session.ResumeState = UploadResumeStateSessionExpiredRestarted
	session.Status = UploadSessionStatusInit
	session.UploadId = ""
	session.UploadedBytes = 0
	session.UploadedParts = 0
	session.LastError = cause.Error()
	if err := session.Save(); err != nil {
		helpers.AppLogger.Warnf("[上传] 保存重新初始化 session 状态失败：%s", err.Error())
	}
	task.ResumeState = UploadResumeStateSessionExpiredRestarted
	task.UploadedBytes = 0
	if err := db.Db.Model(task).Updates(map[string]any{
		"resume_state":   task.ResumeState,
		"uploaded_bytes": task.UploadedBytes,
	}).Error; err != nil {
		helpers.AppLogger.Warnf("[上传] 保存重新初始化任务状态失败：%s", err.Error())
	}
}

func (task *DbUploadTask) save115UploadProgress(session *UploadSession, progress v115open.OSSMultipartProgress) {
	if task == nil || session == nil {
		return
	}
	now := time.Now().Unix()
	oldBytes := session.UploadedBytes
	oldAt := session.LastProgressAt
	session.UploadId = progress.UploadId
	session.PartSize = progress.PartSize
	session.TotalParts = progress.TotalParts
	session.UploadedBytes = progress.UploadedBytes
	session.UploadedParts = progress.UploadedParts
	session.LastPartNumber = progress.LastPartNumber
	session.LastPartEtag = progress.LastPartEtag
	session.LastProgressAt = now
	session.Status = UploadSessionStatusMultipart
	if err := session.Save(); err != nil {
		helpers.AppLogger.Warnf("[上传] 保存 115 上传进度失败：%s", err.Error())
		return
	}

	var speed int64
	if oldAt > 0 && now > oldAt && progress.UploadedBytes > oldBytes {
		speed = (progress.UploadedBytes - oldBytes) / (now - oldAt)
	}
	task.UploadedBytes = progress.UploadedBytes
	task.ResumeState = session.ResumeState
	task.UploadSpeedBytes = speed
	task.UploadPhase = uploadPhaseMultipartUploading
	task.TotalParts = progress.TotalParts
	task.UploadedParts = progress.UploadedParts
	task.RapidWaitAttempts = session.RapidWaitAttempts
	task.RapidWaitUntil = session.RapidWaitUntil
	if err := db.Db.Model(task).Updates(map[string]any{
		"uploaded_bytes":      task.UploadedBytes,
		"resume_state":        task.ResumeState,
		"rapid_wait_attempts": task.RapidWaitAttempts,
		"rapid_wait_until":    task.RapidWaitUntil,
	}).Error; err != nil {
		helpers.AppLogger.Warnf("[上传] 保存上传任务进度失败：%s", err.Error())
	}
	publishUploadQueueChanged(task, "progress")
}

func (task *DbUploadTask) publish115UploadPhase(session *UploadSession, phase string) {
	if task == nil {
		return
	}
	task.UploadPhase = phase
	if session != nil {
		task.ResumeState = session.ResumeState
		task.UploadedBytes = session.UploadedBytes
		task.RapidWaitAttempts = session.RapidWaitAttempts
		task.RapidWaitUntil = session.RapidWaitUntil
		task.TotalParts = session.TotalParts
		task.UploadedParts = session.UploadedParts
	}
	_ = db.Db.Model(task).Updates(map[string]any{
		"resume_state":        task.ResumeState,
		"uploaded_bytes":      task.UploadedBytes,
		"rapid_wait_attempts": task.RapidWaitAttempts,
		"rapid_wait_until":    task.RapidWaitUntil,
		"file_size":           task.FileSize,
		"file_name":           task.FileName,
	}).Error
	publishUploadQueueChanged(task, "progress")
}

func (task *DbUploadTask) applyUpload115TaskResult(result upload115TaskResult) {
	if task == nil {
		return
	}
	if result.UploadResult == "" {
		result.UploadResult = UploadResultUnknown
	}
	task.UploadResult = result.UploadResult
	if result.ResumeState != "" {
		task.ResumeState = result.ResumeState
	}
	if result.UploadedBytes > 0 {
		task.UploadedBytes = result.UploadedBytes
	}
	task.TotalParts = result.TotalParts
	task.UploadedParts = result.UploadedParts
	task.CompletedRemoteFileId = result.CompletedRemoteFileId
	task.CompletedPickCode = result.CompletedPickCode
	if result.CompletedSize > 0 {
		task.FileSize = result.CompletedSize
	}
	task.applyUploadQueueDisplayFields(nil)
}

func hydrateUploadTaskQueueFields(tasks []*DbUploadTask) {
	if len(tasks) == 0 {
		return
	}
	taskIDs := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		if task != nil {
			taskIDs = append(taskIDs, task.ID)
		}
	}
	if len(taskIDs) == 0 {
		return
	}
	var sessions []UploadSession
	if err := db.Db.Where("upload_task_id IN ?", taskIDs).
		Order("upload_task_id ASC, id DESC").
		Find(&sessions).Error; err != nil {
		helpers.AppLogger.Warnf("[上传] 查询上传会话展示字段失败：%s", err.Error())
	}
	sessionByTaskID := make(map[uint]*UploadSession, len(sessions))
	for i := range sessions {
		session := &sessions[i]
		if _, ok := sessionByTaskID[session.UploadTaskId]; !ok {
			sessionByTaskID[session.UploadTaskId] = session
		}
	}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		task.applyUploadQueueDisplayFields(sessionByTaskID[task.ID])
	}
}

func (task *DbUploadTask) applyUploadQueueDisplayFields(session *UploadSession) {
	if task == nil {
		return
	}
	if session != nil {
		if session.UploadedBytes > task.UploadedBytes {
			task.UploadedBytes = session.UploadedBytes
		}
		if session.TotalParts > 0 {
			task.TotalParts = session.TotalParts
		}
		if session.UploadedParts > 0 {
			task.UploadedParts = session.UploadedParts
		}
		if session.ResumeState != "" {
			task.ResumeState = session.ResumeState
		}
		if task.RapidWaitAttempts == 0 {
			task.RapidWaitAttempts = session.RapidWaitAttempts
		}
		if task.RapidWaitUntil == 0 {
			task.RapidWaitUntil = session.RapidWaitUntil
		}
		if task.UploadPhase == "" {
			task.UploadPhase = uploadPhaseFromSession(session)
		}
	}
	if task.UploadPhase == "" {
		task.UploadPhase = uploadPhaseFromTask(task)
	}
	if task.FileSize > 0 && task.UploadedBytes > 0 {
		percent := float64(task.UploadedBytes) / float64(task.FileSize) * 100
		if percent > 100 {
			percent = 100
		}
		task.ProgressPercent = math.Round(percent*100) / 100
	}
}

func uploadPhaseFromSession(session *UploadSession) string {
	if session == nil {
		return ""
	}
	switch session.Status {
	case UploadSessionStatusRapidWaiting:
		return uploadPhaseRapidWaiting
	case UploadSessionStatusMultipart:
		return uploadPhaseMultipartUploading
	case UploadSessionStatusCompleting:
		return uploadPhaseCompleting
	case UploadSessionStatusCompleted:
		return uploadPhaseCompleted
	case UploadSessionStatusFailed:
		return uploadPhaseFailed
	case UploadSessionStatusAborted:
		return uploadPhaseFailed
	default:
		return ""
	}
}

func uploadPhaseFromTask(task *DbUploadTask) string {
	if task == nil {
		return ""
	}
	switch task.Status {
	case UploadStatusPending:
		return uploadPhasePending
	case UploadStatusUploading:
		return "uploading"
	case UploadStatusCompleted:
		switch task.UploadResult {
		case UploadResultRapidUpload:
			return uploadPhaseRapidUploaded
		case UploadResultRemoteExists:
			return uploadPhaseRemoteExists
		case UploadResultSkippedAfterRapidWait:
			return uploadPhaseSkipped
		default:
			return uploadPhaseCompleted
		}
	case UploadStatusFailed:
		return uploadPhaseFailed
	case UploadStatusCancelled:
		return uploadPhaseCancelled
	default:
		return ""
	}
}

func uploadRapidWaitPolicyFromSettings() v115open.RapidUploadWaitPolicy {
	if SettingsGlobal == nil {
		return v115open.RapidUploadWaitPolicy{}
	}
	intervalSeconds := SettingsGlobal.UploadRapidWaitIntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	return v115open.RapidUploadWaitPolicy{
		Enabled:    SettingsGlobal.UploadRapidWaitEnabled == 1,
		Timeout:    time.Duration(SettingsGlobal.UploadRapidWaitTimeoutSeconds) * time.Second,
		Interval:   time.Duration(intervalSeconds) * time.Second,
		MinSize:    SettingsGlobal.UploadRapidWaitMinSize,
		ForceSize:  SettingsGlobal.UploadRapidWaitForceSize,
		SkipUpload: SettingsGlobal.UploadRapidWaitSkipUpload == 1,
	}
}

// EnqueueStrmGenerationAfterUpload 根据上传完成结果创建 STRM 生成任务。
func (task *DbUploadTask) EnqueueStrmGenerationAfterUpload() error {
	return task.enqueueStrmGenerationAfterUpload()
}

func (task *DbUploadTask) enqueueStrmGenerationAfterUpload() error {
	if task == nil {
		return nil
	}
	if task.Source != UploadSourceDirectoryMonitor && task.Source != UploadSourceStrm {
		return nil
	}
	if task.UploadResult == UploadResultSkippedAfterRapidWait {
		return nil
	}
	if task.CompletedRemoteFileId == "" && task.CompletedPickCode == "" {
		return nil
	}

	syncPathID := task.SyncPathId
	accountID := task.AccountId
	parentID := task.RemotePathId
	path := task.RemoteFileId
	fileName := task.FileName
	fileSize := task.FileSize
	var sha1 string
	var mtime int64
	if session, err := GetUploadSessionByUploadTaskId(task.ID); err == nil && session != nil {
		if parentID == "" {
			parentID = session.CompletedParentId
		}
		if fileSize == 0 {
			fileSize = session.CompletedSize
		}
		sha1 = session.CompletedSha1
		mtime = session.CompletedMtime
	}
	if task.SyncFileId > 0 {
		if syncFile := GetSyncFileById(task.SyncFileId); syncFile != nil {
			if syncPathID == 0 {
				syncPathID = syncFile.SyncPathId
			}
			if accountID == 0 {
				accountID = syncFile.AccountId
			}
			if parentID == "" {
				parentID = syncFile.ParentId
			}
			if path == "" {
				path = syncFile.Path
			}
			if fileName == "" {
				fileName = syncFile.FileName
			}
			if fileSize == 0 {
				fileSize = syncFile.FileSize
			}
			if sha1 == "" {
				sha1 = syncFile.Sha1
			}
			if mtime == 0 {
				mtime = syncFile.MTime
			}
		}
	}
	if syncPathID == 0 {
		return nil
	}

	source := StrmGenerationSourceUploadCompleted
	if task.UploadResult == UploadResultRemoteExists {
		source = StrmGenerationSourceRemoteExists
	}
	requestHash := fmt.Sprintf("%s:%d:%s:%s", source, syncPathID, task.CompletedRemoteFileId, task.CompletedPickCode)
	_, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:       source,
		TaskType:     StrmGenerationTaskTypeFile,
		UploadTaskId: task.ID,
		SyncPathId:   syncPathID,
		AccountId:    accountID,
		FileId:       task.CompletedRemoteFileId,
		ParentId:     parentID,
		PickCode:     task.CompletedPickCode,
		Path:         path,
		FileName:     fileName,
		FileSize:     fileSize,
		Sha1:         sha1,
		Mtime:        mtime,
		RequestHash:  requestHash,
	})
	return err
}
