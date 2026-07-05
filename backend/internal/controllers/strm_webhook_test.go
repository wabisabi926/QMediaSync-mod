package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/v115open"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupStrmWebhookControllerTest(t *testing.T) (*gin.Engine, string, *models.SyncPath) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB
	if err := db.Db.AutoMigrate(
		&models.User{},
		&models.ApiKey{},
		&models.Account{},
		&models.SyncPath{},
		&models.StrmGenerationTask{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	_, rawKey, err := models.CreateAPIKey(user.ID, "strm webhook")
	if err != nil {
		t.Fatalf("创建 API Key 失败: %v", err)
	}
	account := &models.Account{SourceType: models.SourceType115, Name: "115"}
	if err := db.Db.Create(account).Error; err != nil {
		t.Fatalf("创建账号失败: %v", err)
	}
	syncPath := &models.SyncPath{
		SourceType: models.SourceType115,
		AccountId:  account.ID,
		BaseCid:    "root",
		LocalPath:  t.TempDir(),
		RemotePath: "/remote",
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}
	router := gin.New()
	router.POST("/api/strm/webhook", StrmWebhook)
	return router, rawKey, syncPath
}

func setStrmWebhookFileDetailResolverForTesting(t *testing.T, resolver strmWebhookFileDetailResolver) {
	t.Helper()
	oldResolver := resolveStrmWebhookFileDetail
	resolveStrmWebhookFileDetail = resolver
	t.Cleanup(func() {
		resolveStrmWebhookFileDetail = oldResolver
	})
}

func setStrmWebhookFileIDDetailResolverForTesting(t *testing.T, resolver strmWebhookFileDetailByIDResolver) {
	t.Helper()
	oldResolver := resolveStrmWebhookFileDetailByID
	resolveStrmWebhookFileDetailByID = resolver
	t.Cleanup(func() {
		resolveStrmWebhookFileDetailByID = oldResolver
	})
}

func TestStrmWebhookAuthSupportsHeaderAndQueryAPIKey(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	setStrmWebhookFileIDDetailResolverForTesting(t, func(_ context.Context, _ *models.Account, fileID string) (*v115open.FileDetail, error) {
		return &v115open.FileDetail{
			FileId:       fileID,
			PickCode:     "pick-" + fileID,
			FileName:     fileID + ".mkv",
			Path:         "/remote",
			FileSizeByte: 1024,
			Utime:        "123",
		}, nil
	})
	payload := map[string]any{
		"sync_path_id": syncPath.ID,
		"file_id":      "file-1",
		"pick_code":    "pick-1",
		"file_name":    "movie.mkv",
	}

	w := performStrmWebhookRequest(t, router, rawKey, "", payload)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("header API Key 响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	w = performStrmWebhookRequest(t, router, "", rawKey, map[string]any{
		"sync_path_id": syncPath.ID,
		"file_id":      "file-2",
		"pick_code":    "pick-2",
		"file_name":    "movie2.mkv",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("query API Key 响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	w = performStrmWebhookRequest(t, router, "bad-key", "", payload)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无效 API Key 状态码 = %d，期望 401，body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookValidatesFileLocator(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	cases := []struct {
		name    string
		payload map[string]any
		want    string
	}{
		{
			name:    "缺少同步目录",
			payload: map[string]any{"file_id": "file-1"},
			want:    "sync_path_id",
		},
		{
			name:    "缺少文件定位",
			payload: map[string]any{"sync_path_id": syncPath.ID, "file_name": "movie.mkv"},
			want:    "file_id 或 path + file_name",
		},
		{
			name:    "拒绝仅提供 pick_code",
			payload: map[string]any{"sync_path_id": syncPath.ID, "pick_code": "pick-1"},
			want:    "仅提供 pick_code 无法生成 STRM",
		},
		{
			name: "拒绝 local_path",
			payload: map[string]any{
				"sync_path_id": syncPath.ID,
				"file_id":      "file-1",
				"local_path":   "/tmp/evil.strm",
			},
			want: "local_path",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w := performStrmWebhookRequest(t, router, rawKey, "", tt.payload)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), tt.want) {
				t.Fatalf("响应异常: code=%d body=%s want=%s", w.Code, w.Body.String(), tt.want)
			}
		})
	}
}

func TestStrmWebhookEnqueuesFileTaskIdempotently(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	setStrmWebhookFileIDDetailResolverForTesting(t, func(_ context.Context, _ *models.Account, fileID string) (*v115open.FileDetail, error) {
		return &v115open.FileDetail{
			FileId:       fileID,
			PickCode:     "pick-1",
			FileName:     "movie.mkv",
			Path:         "/remote",
			FileSizeByte: 1024,
			Sha1:         "sha1",
			Utime:        "123",
			Paths:        []v115open.FileDetailPath{{FileId: "parent-1", Name: "remote"}},
		}, nil
	})
	payload := map[string]any{
		"sync_path_id": syncPath.ID,
		"file_id":      "file-1",
		"pick_code":    "pick-1",
		"parent_id":    "parent-1",
		"path":         "/remote",
		"file_name":    "movie.mkv",
		"file_size":    1024,
		"sha1":         "sha1",
		"mtime":        123,
	}
	w := performStrmWebhookRequest(t, router, rawKey, "", payload)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("首次入队响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	w = performStrmWebhookRequest(t, router, rawKey, "", payload)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("重复入队响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var total int64
	if err := db.Db.Model(&models.StrmGenerationTask{}).Count(&total).Error; err != nil {
		t.Fatalf("统计 STRM 任务失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("STRM 任务数量 = %d，期望 request_hash 去重为 1", total)
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取 STRM 任务失败: %v", err)
	}
	if task.Source != models.StrmGenerationSourceWebhook ||
		task.TaskType != models.StrmGenerationTaskTypeFile ||
		task.SyncPathId != syncPath.ID ||
		task.FileId != "file-1" ||
		task.PickCode != "pick-1" {
		t.Fatalf("STRM 任务 = %+v，期望 webhook file 任务", task)
	}
}

func TestStrmWebhookOptionsDefaultFalse(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	setStrmWebhookFileIDDetailResolverForTesting(t, func(_ context.Context, _ *models.Account, fileID string) (*v115open.FileDetail, error) {
		return &v115open.FileDetail{FileId: fileID, PickCode: "pick-1", FileName: "movie.mkv", Path: "/remote"}, nil
	})

	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id": syncPath.ID,
		"action":       "file",
		"file_id":      "file-1",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取 STRM 任务失败: %v", err)
	}
	if task.DownloadMeta || task.RefreshEmby {
		t.Fatalf("默认开关 = download_meta:%v refresh_emby:%v，期望 false/false", task.DownloadMeta, task.RefreshEmby)
	}
}

func TestStrmWebhookBatchCreatesParentAndChildrenWithOptions(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	setStrmWebhookFileIDDetailResolverForTesting(t, func(_ context.Context, _ *models.Account, fileID string) (*v115open.FileDetail, error) {
		return &v115open.FileDetail{
			FileId:   fileID,
			PickCode: "pick-" + fileID,
			FileName: fileID + ".mkv",
			Path:     "/remote/show",
		}, nil
	})

	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id":  syncPath.ID,
		"action":        "batch_files",
		"download_meta": true,
		"refresh_emby":  true,
		"items": []map[string]any{
			{"file_id": "file-1"},
			{"file_id": "file-2"},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var parent models.StrmGenerationTask
	if err := db.Db.Where("task_type = ?", models.StrmGenerationTaskTypeBatchFiles).First(&parent).Error; err != nil {
		t.Fatalf("读取批量父任务失败: %v", err)
	}
	if parent.TotalItems != 2 || !parent.DownloadMeta || !parent.RefreshEmby {
		t.Fatalf("父任务 = %+v，期望 total=2 且开关为 true", parent)
	}
	var children []models.StrmGenerationTask
	if err := db.Db.Where("parent_task_id = ?", parent.ID).Find(&children).Error; err != nil {
		t.Fatalf("读取子任务失败: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("子任务数量 = %d，期望 2", len(children))
	}
	for _, child := range children {
		if !child.DownloadMeta || !child.RefreshEmby {
			t.Fatalf("子任务未继承开关: %+v", child)
		}
	}
}

func TestStrmWebhookRejectsItemLevelOptions(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id": syncPath.ID,
		"action":       "batch_files",
		"items": []map[string]any{
			{
				"file_id":       "file-1",
				"download_meta": true,
				"refresh_emby":  false,
			},
		},
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "items[] 不允许设置 download_meta 或 refresh_emby") {
		t.Fatalf("item 级开关应被拒绝: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookResolvesPathAndFileNameBeforeEnqueue(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	setStrmWebhookFileDetailResolverForTesting(t, func(_ context.Context, account *models.Account, fullPath string) (*v115open.FileDetail, error) {
		if account.ID != syncPath.AccountId {
			t.Fatalf("解析账号 ID = %d，期望 %d", account.ID, syncPath.AccountId)
		}
		if fullPath != "/remote/show/movie.mkv" {
			t.Fatalf("解析路径 = %s，期望 /remote/show/movie.mkv", fullPath)
		}
		return &v115open.FileDetail{
			FileId:       "file-path-1",
			PickCode:     "pick-path-1",
			FileName:     "movie.mkv",
			Path:         "/remote/show",
			FileSizeByte: 2048,
			Sha1:         "sha1-path",
			Utime:        "123456",
			Paths:        []v115open.FileDetailPath{{FileId: "parent-1", Name: "show"}},
		}, nil
	})

	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id": syncPath.ID,
		"path":         "/remote/show",
		"file_name":    "movie.mkv",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("path + file_name 入队响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取 STRM 任务失败: %v", err)
	}
	if task.FileId != "file-path-1" ||
		task.PickCode != "pick-path-1" ||
		task.ParentId != "parent-1" ||
		task.Path != "/remote/show" ||
		task.FileName != "movie.mkv" ||
		task.FileSize != 2048 ||
		task.Sha1 != "sha1-path" ||
		task.Mtime != 123456 {
		t.Fatalf("STRM 任务 = %+v，期望 path + file_name 已解析为完整远端详情", task)
	}
}

func TestStrmWebhookAutoMatchesMostSpecificSyncPathByFilePath(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	nestedSyncPath := &models.SyncPath{
		SourceType: models.SourceType115,
		AccountId:  syncPath.AccountId,
		BaseCid:    "nested-root",
		LocalPath:  t.TempDir(),
		RemotePath: "/remote/show",
	}
	if err := db.Db.Create(nestedSyncPath).Error; err != nil {
		t.Fatalf("创建嵌套同步目录失败: %v", err)
	}
	setStrmWebhookFileDetailResolverForTesting(t, func(_ context.Context, account *models.Account, fullPath string) (*v115open.FileDetail, error) {
		if account.ID != nestedSyncPath.AccountId {
			t.Fatalf("解析账号 ID = %d，期望 %d", account.ID, nestedSyncPath.AccountId)
		}
		if fullPath != "/remote/show/S01/movie.mkv" {
			t.Fatalf("解析路径 = %s，期望 /remote/show/S01/movie.mkv", fullPath)
		}
		return &v115open.FileDetail{
			FileId:       "file-auto",
			PickCode:     "pick-auto",
			FileName:     "movie.mkv",
			Path:         "/remote/show/S01",
			FileSizeByte: 2048,
			Utime:        "123456",
			Paths:        []v115open.FileDetailPath{{FileId: "parent-auto", Name: "S01"}},
		}, nil
	})

	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"action":    "file",
		"path":      "/remote/show/S01",
		"file_name": "movie.mkv",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("自动匹配同步目录响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取 STRM 任务失败: %v", err)
	}
	if task.SyncPathId != nestedSyncPath.ID || task.Path != "/remote/show/S01" {
		t.Fatalf("STRM 任务 = %+v，期望使用最具体同步目录 %d", task, nestedSyncPath.ID)
	}
}

func TestStrmWebhookRejectsFileIDOnlyWhenSyncPathIDMissing(t *testing.T) {
	router, rawKey, _ := setupStrmWebhookControllerTest(t)
	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"action":  "file",
		"file_id": "file-auto",
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "path + file_name") {
		t.Fatalf("未提供 sync_path_id 时仅 file_id 应被拒绝: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookRejectsAmbiguousAutoMatchedSyncPath(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	duplicateSyncPath := &models.SyncPath{
		SourceType: models.SourceType115,
		AccountId:  syncPath.AccountId,
		BaseCid:    "duplicate-root",
		LocalPath:  t.TempDir(),
		RemotePath: "/remote",
	}
	if err := db.Db.Create(duplicateSyncPath).Error; err != nil {
		t.Fatalf("创建重复远端同步目录失败: %v", err)
	}

	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"action":    "file",
		"path":      "/remote",
		"file_name": "movie.mkv",
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "多个同步目录") {
		t.Fatalf("自动匹配到多个同步目录应被拒绝: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookRejectsFileIDOutsideSyncPath(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	setStrmWebhookFileIDDetailResolverForTesting(t, func(_ context.Context, _ *models.Account, fileID string) (*v115open.FileDetail, error) {
		return &v115open.FileDetail{
			FileId:       fileID,
			PickCode:     "pick-outside",
			FileName:     "outside.mkv",
			Path:         "/outside",
			FileSizeByte: 1024,
		}, nil
	})
	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id": syncPath.ID,
		"file_id":      "file-outside",
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不在同步远端目录") {
		t.Fatalf("同步目录外 file_id 应被拒绝: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookBatchFilesReturnsItemResults(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	setStrmWebhookFileIDDetailResolverForTesting(t, func(_ context.Context, _ *models.Account, fileID string) (*v115open.FileDetail, error) {
		return &v115open.FileDetail{
			FileId:       fileID,
			PickCode:     "pick-1",
			FileName:     "movie.mkv",
			Path:         "/remote",
			FileSizeByte: 1024,
			Utime:        "123",
		}, nil
	})
	payload := map[string]any{
		"sync_path_id": syncPath.ID,
		"action":       "batch_files",
		"items": []map[string]any{
			{"file_id": "file-1", "pick_code": "pick-1", "file_name": "movie.mkv"},
			{"file_name": "missing.mkv"},
		},
	}
	w := performStrmWebhookRequest(t, router, rawKey, "", payload)
	body := w.Body.String()
	if w.Code != http.StatusOK ||
		!strings.Contains(body, `"accepted_count":1`) ||
		!strings.Contains(body, `"failed_count":1`) ||
		!strings.Contains(body, `"accepted":true`) ||
		!strings.Contains(body, `"accepted":false`) {
		t.Fatalf("批量响应异常: code=%d body=%s", w.Code, body)
	}
	var total int64
	if err := db.Db.Model(&models.StrmGenerationTask{}).Count(&total).Error; err != nil {
		t.Fatalf("统计 STRM 任务失败: %v", err)
	}
	if total != 2 {
		t.Fatalf("STRM 任务数量 = %d，期望批量父任务 + 1 个合法子任务", total)
	}
	var childCount int64
	if err := db.Db.Model(&models.StrmGenerationTask{}).
		Where("task_type = ? AND parent_task_id > 0", models.StrmGenerationTaskTypeFile).
		Count(&childCount).Error; err != nil {
		t.Fatalf("统计 STRM 子任务失败: %v", err)
	}
	if childCount != 1 {
		t.Fatalf("STRM 合法子任务数量 = %d，期望只入队 1 个合法项", childCount)
	}
}

func TestStrmWebhookRejectsAutoMatchedBatchAcrossSyncPaths(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	otherSyncPath := &models.SyncPath{
		SourceType: models.SourceType115,
		AccountId:  syncPath.AccountId,
		BaseCid:    "other-root",
		LocalPath:  t.TempDir(),
		RemotePath: "/other",
	}
	if err := db.Db.Create(otherSyncPath).Error; err != nil {
		t.Fatalf("创建其他同步目录失败: %v", err)
	}

	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"action": "batch_files",
		"items": []map[string]any{
			{"path": "/remote", "file_name": "movie-a.mkv"},
			{"path": "/other", "file_name": "movie-b.mkv"},
		},
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "多个同步目录") {
		t.Fatalf("跨同步目录批量请求应被拒绝: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookDirectoryScan(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id":   syncPath.ID,
		"action":         "directory_scan",
		"download_meta":  true,
		"refresh_emby":   true,
		"directory_id":   "dir-1",
		"directory_path": "/remote/show",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("目录扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取目录扫描任务失败: %v", err)
	}
	if task.TaskType != models.StrmGenerationTaskTypeDirectoryScan ||
		task.DirectoryId != "dir-1" ||
		task.DirectoryPath != "/remote/show" {
		t.Fatalf("目录扫描任务 = %+v，期望 directory_scan", task)
	}
	if !task.DownloadMeta || !task.RefreshEmby {
		t.Fatalf("目录扫描任务开关 = download_meta:%v refresh_emby:%v，期望 true/true", task.DownloadMeta, task.RefreshEmby)
	}

	w = performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id":   syncPath.ID,
		"action":         "directory_scan",
		"directory_path": "/other/show",
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不在同步远端目录") {
		t.Fatalf("非法目录响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookAutoMatchesSyncPathByDirectoryPath(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"action":         "directory_scan",
		"directory_path": "/remote/show",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("目录扫描自动匹配响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取目录扫描任务失败: %v", err)
	}
	if task.SyncPathId != syncPath.ID || task.DirectoryPath != "/remote/show" {
		t.Fatalf("目录扫描任务 = %+v，期望自动匹配同步目录 %d", task, syncPath.ID)
	}
}

func performStrmWebhookRequest(t *testing.T, router *gin.Engine, headerKey string, queryKey string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	path := "/api/strm/webhook"
	if queryKey != "" {
		path += "?api_key=" + queryKey
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if headerKey != "" {
		req.Header.Set(apiKeyHeaderName, headerKey)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
