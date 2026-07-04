package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	pathpkg "path"
	"strings"

	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/v115open"

	"github.com/gin-gonic/gin"
)

type strmWebhookFileDetailResolver func(context.Context, *models.Account, string) (*v115open.FileDetail, error)
type strmWebhookFileDetailByIDResolver func(context.Context, *models.Account, string) (*v115open.FileDetail, error)

var resolveStrmWebhookFileDetail strmWebhookFileDetailResolver = defaultStrmWebhookFileDetailResolver
var resolveStrmWebhookFileDetailByID strmWebhookFileDetailByIDResolver = defaultStrmWebhookFileDetailByIDResolver

const (
	strmWebhookActionFile          = "file"
	strmWebhookActionBatchFiles    = "batch_files"
	strmWebhookActionDirectoryScan = "directory_scan"
)

type strmWebhookRequest struct {
	SyncPathID uint   `json:"sync_path_id"`
	Action     string `json:"action"`
	LocalPath  string `json:"local_path"`

	strmWebhookFileItem

	Items         []strmWebhookFileItem `json:"items"`
	DirectoryID   string                `json:"directory_id"`
	DirectoryPath string                `json:"directory_path"`
}

type strmWebhookFileItem struct {
	FileID    string `json:"file_id"`
	PickCode  string `json:"pick_code"`
	ParentID  string `json:"parent_id"`
	Path      string `json:"path"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	Sha1      string `json:"sha1"`
	Mtime     int64  `json:"mtime"`
	LocalPath string `json:"local_path"`
}

type strmWebhookResponse struct {
	RequestID     string                  `json:"request_id"`
	TaskIDs       []uint                  `json:"task_ids"`
	AcceptedCount int                     `json:"accepted_count"`
	FailedCount   int                     `json:"failed_count"`
	Results       []strmWebhookItemResult `json:"results"`
}

type strmWebhookItemResult struct {
	Index    int    `json:"index"`
	Accepted bool   `json:"accepted"`
	TaskID   uint   `json:"task_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

// StrmWebhook 接收外部请求创建 STRM 生成任务。
func StrmWebhook(c *gin.Context) {
	apiKey, err := authenticateStrmWebhookAPIKey(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "API Key 无效", Data: nil})
		return
	}
	_ = apiKey.UpdateLastUsedAt()

	var req strmWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("请求参数错误：%v", err), Data: nil})
		return
	}
	resp, err := handleStrmWebhookRequest(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[strmWebhookResponse]{Code: Success, Message: "STRM 生成任务已接收", Data: resp})
}

func authenticateStrmWebhookAPIKey(c *gin.Context) (*models.ApiKey, error) {
	raw := c.GetHeader(apiKeyHeaderName)
	if raw == "" {
		raw = c.Query("api_key")
	}
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("API Key 为空")
	}
	return models.ValidateAPIKey(raw)
}

func handleStrmWebhookRequest(ctx context.Context, req strmWebhookRequest) (strmWebhookResponse, error) {
	if strings.TrimSpace(req.LocalPath) != "" {
		return strmWebhookResponse{}, errors.New("不允许通过 local_path 指定本地写入路径")
	}
	syncPath, err := loadStrmWebhookSyncPath(req.SyncPathID)
	if err != nil {
		return strmWebhookResponse{}, err
	}
	action := inferStrmWebhookAction(req)
	resp := strmWebhookResponse{RequestID: "strm_" + helpers.RandStr(16)}
	switch action {
	case strmWebhookActionFile:
		result := enqueueStrmWebhookFile(ctx, syncPath, 0, req.strmWebhookFileItem)
		if !result.Accepted {
			return strmWebhookResponse{}, errors.New(result.Error)
		}
		resp.Results = append(resp.Results, result)
	case strmWebhookActionBatchFiles:
		if len(req.Items) == 0 {
			return strmWebhookResponse{}, errors.New("items 不能为空")
		}
		for index, item := range req.Items {
			resp.Results = append(resp.Results, enqueueStrmWebhookFile(ctx, syncPath, index, item))
		}
	case strmWebhookActionDirectoryScan:
		result := enqueueStrmWebhookDirectory(syncPath, req)
		if !result.Accepted {
			return strmWebhookResponse{}, errors.New(result.Error)
		}
		resp.Results = append(resp.Results, result)
	default:
		return strmWebhookResponse{}, fmt.Errorf("不支持的 action：%s", action)
	}
	for _, result := range resp.Results {
		if result.Accepted {
			resp.AcceptedCount++
			resp.TaskIDs = append(resp.TaskIDs, result.TaskID)
		} else {
			resp.FailedCount++
		}
	}
	return resp, nil
}

func loadStrmWebhookSyncPath(syncPathID uint) (*models.SyncPath, error) {
	if syncPathID == 0 {
		return nil, errors.New("sync_path_id 不能为空")
	}
	syncPath := models.GetSyncPathById(syncPathID)
	if syncPath == nil {
		return nil, fmt.Errorf("同步目录不存在：%d", syncPathID)
	}
	return syncPath, nil
}

func inferStrmWebhookAction(req strmWebhookRequest) string {
	action := strings.TrimSpace(req.Action)
	if action != "" {
		return action
	}
	if len(req.Items) > 0 {
		return strmWebhookActionBatchFiles
	}
	if req.DirectoryID != "" || req.DirectoryPath != "" {
		return strmWebhookActionDirectoryScan
	}
	return strmWebhookActionFile
}

func enqueueStrmWebhookFile(ctx context.Context, syncPath *models.SyncPath, index int, item strmWebhookFileItem) strmWebhookItemResult {
	if strings.TrimSpace(item.LocalPath) != "" {
		return strmWebhookItemResult{Index: index, Accepted: false, Error: "不允许通过 local_path 指定本地写入路径"}
	}
	item = normalizeStrmWebhookFileItem(item)
	if err := validateStrmWebhookFileItem(syncPath, item); err != nil {
		return strmWebhookItemResult{Index: index, Accepted: false, Error: err.Error()}
	}
	if err := resolveStrmWebhookFileItem(ctx, syncPath, &item); err != nil {
		return strmWebhookItemResult{Index: index, Accepted: false, Error: err.Error()}
	}
	task, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:      models.StrmGenerationSourceWebhook,
		TaskType:    models.StrmGenerationTaskTypeFile,
		SyncPathId:  syncPath.ID,
		AccountId:   syncPath.AccountId,
		FileId:      item.FileID,
		ParentId:    item.ParentID,
		PickCode:    item.PickCode,
		Path:        item.Path,
		FileName:    item.FileName,
		FileSize:    item.FileSize,
		Sha1:        item.Sha1,
		Mtime:       item.Mtime,
		RequestHash: strmWebhookFileRequestHash(syncPath.ID, item),
	})
	if err != nil {
		return strmWebhookItemResult{Index: index, Accepted: false, Error: err.Error()}
	}
	return strmWebhookItemResult{Index: index, Accepted: true, TaskID: task.ID}
}

func defaultStrmWebhookFileDetailResolver(ctx context.Context, account *models.Account, fullPath string) (*v115open.FileDetail, error) {
	if account == nil {
		return nil, errors.New("账号为空")
	}
	client := account.Get115Client()
	if client == nil {
		return nil, errors.New("115 客户端为空")
	}
	return client.GetFsDetailByPath(ctx, fullPath)
}

func defaultStrmWebhookFileDetailByIDResolver(ctx context.Context, account *models.Account, fileID string) (*v115open.FileDetail, error) {
	if account == nil {
		return nil, errors.New("账号为空")
	}
	client := account.Get115Client()
	if client == nil {
		return nil, errors.New("115 客户端为空")
	}
	return client.GetFsDetailByCid(ctx, fileID)
}

func resolveStrmWebhookFileItem(ctx context.Context, syncPath *models.SyncPath, item *strmWebhookFileItem) error {
	if item == nil {
		return nil
	}
	account, err := models.GetAccountById(syncPath.AccountId)
	if err != nil {
		return fmt.Errorf("查询同步账号失败: %w", err)
	}
	var detail *v115open.FileDetail
	var requestedFullPath string
	if item.FileID != "" {
		detail, err = resolveStrmWebhookFileDetailByID(ctx, account, item.FileID)
		if err != nil {
			return fmt.Errorf("解析远端文件详情失败: %w", err)
		}
	} else {
		requestedFullPath = pathpkg.Join(item.Path, item.FileName)
		detail, err = resolveStrmWebhookFileDetail(ctx, account, requestedFullPath)
		if err != nil {
			return fmt.Errorf("解析远端文件详情失败: %w", err)
		}
	}
	if detail == nil || strings.TrimSpace(detail.FileId) == "" {
		return errors.New("解析远端文件详情失败：返回空文件 ID")
	}
	applyStrmWebhookFileDetail(item, detail)
	if requestedFullPath != "" {
		if item.FileName == "" {
			item.FileName = pathpkg.Base(requestedFullPath)
		}
		if item.Path == "" {
			item.Path = normalizeRemotePath(pathpkg.Dir(requestedFullPath))
		}
	}
	if item.FileName == "" {
		return errors.New("解析远端文件详情失败：缺少文件名")
	}
	if item.Path == "" {
		return errors.New("解析远端文件详情失败：缺少远端路径")
	}
	if !remotePathWithin(item.Path, syncPath.RemotePath) {
		return fmt.Errorf("远端路径 %s 不在同步远端目录 %s 下", item.Path, normalizeRemotePath(syncPath.RemotePath))
	}
	return nil
}

func applyStrmWebhookFileDetail(item *strmWebhookFileItem, detail *v115open.FileDetail) {
	item.FileID = strings.TrimSpace(detail.FileId)
	item.PickCode = strings.TrimSpace(detail.PickCode)
	item.FileName = strings.TrimSpace(detail.FileName)
	item.Path = normalizeRemotePath(detail.Path)
	item.ParentID = strmWebhookDetailParentID(detail)
	if detail.FileSizeByte > 0 {
		item.FileSize = detail.FileSizeByte
	} else {
		item.FileSize = helpers.StringToInt64(detail.FileSize)
	}
	item.Sha1 = strings.TrimSpace(detail.Sha1)
	item.Mtime = helpers.StringToInt64(detail.Utime)
	if item.Mtime == 0 {
		item.Mtime = helpers.StringToInt64(detail.Ptime)
	}
}

func strmWebhookDetailParentID(detail *v115open.FileDetail) string {
	if detail == nil || len(detail.Paths) == 0 {
		return ""
	}
	return strings.TrimSpace(detail.Paths[len(detail.Paths)-1].FileId)
}

func normalizeStrmWebhookFileItem(item strmWebhookFileItem) strmWebhookFileItem {
	item.FileID = strings.TrimSpace(item.FileID)
	item.PickCode = strings.TrimSpace(item.PickCode)
	item.ParentID = strings.TrimSpace(item.ParentID)
	item.Path = normalizeRemotePath(item.Path)
	item.FileName = strings.TrimSpace(item.FileName)
	item.Sha1 = strings.TrimSpace(item.Sha1)
	item.LocalPath = strings.TrimSpace(item.LocalPath)
	return item
}

func validateStrmWebhookFileItem(syncPath *models.SyncPath, item strmWebhookFileItem) error {
	hasPathName := item.Path != "" && item.FileName != ""
	if item.FileID == "" && !hasPathName {
		if item.PickCode != "" {
			return errors.New("仅提供 pick_code 无法生成 STRM，请提供 file_id 或 path + file_name")
		}
		return errors.New("file_id 或 path + file_name 至少需要提供一组")
	}
	if item.Path != "" && !remotePathWithin(item.Path, syncPath.RemotePath) {
		return fmt.Errorf("远端路径 %s 不在同步远端目录 %s 下", item.Path, normalizeRemotePath(syncPath.RemotePath))
	}
	return nil
}

func enqueueStrmWebhookDirectory(syncPath *models.SyncPath, req strmWebhookRequest) strmWebhookItemResult {
	directoryID := strings.TrimSpace(req.DirectoryID)
	directoryPath := normalizeRemotePath(req.DirectoryPath)
	if directoryID == "" && directoryPath == "" {
		return strmWebhookItemResult{Index: 0, Accepted: false, Error: "directory_id 或 directory_path 至少需要提供一个"}
	}
	if directoryPath != "" && !remotePathWithin(directoryPath, syncPath.RemotePath) {
		return strmWebhookItemResult{Index: 0, Accepted: false, Error: fmt.Sprintf("远端目录 %s 不在同步远端目录 %s 下", directoryPath, normalizeRemotePath(syncPath.RemotePath))}
	}
	task, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:        models.StrmGenerationSourceWebhook,
		TaskType:      models.StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:    syncPath.ID,
		AccountId:     syncPath.AccountId,
		DirectoryId:   directoryID,
		DirectoryPath: directoryPath,
		RequestHash:   strmWebhookDirectoryRequestHash(syncPath.ID, directoryID, directoryPath),
	})
	if err != nil {
		return strmWebhookItemResult{Index: 0, Accepted: false, Error: err.Error()}
	}
	return strmWebhookItemResult{Index: 0, Accepted: true, TaskID: task.ID}
}

func strmWebhookFileRequestHash(syncPathID uint, item strmWebhookFileItem) string {
	return fmt.Sprintf("webhook:file:%d:%s:%s:%s:%s", syncPathID, item.FileID, item.PickCode, item.Path, item.FileName)
}

func strmWebhookDirectoryRequestHash(syncPathID uint, directoryID string, directoryPath string) string {
	return fmt.Sprintf("webhook:directory:%d:%s:%s", syncPathID, directoryID, directoryPath)
}

func normalizeRemotePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return pathpkg.Clean(value)
}

func remotePathWithin(remotePath string, basePath string) bool {
	remotePath = normalizeRemotePath(remotePath)
	basePath = normalizeRemotePath(basePath)
	if remotePath == "" || basePath == "" {
		return false
	}
	if basePath == "/" {
		return strings.HasPrefix(remotePath, "/")
	}
	return remotePath == basePath || strings.HasPrefix(remotePath, basePath+"/")
}
