package models

import (
	"context"
	"errors"
	"testing"

	"qmediasync/internal/v115open"
)

func TestBuild115RapidUploadCompleteResultFillsRemoteMtime(t *testing.T) {
	oldResolver := get115FileDetailByCid
	get115FileDetailByCid = func(_ context.Context, _ *v115open.OpenClient, fileID string) (*v115open.FileDetail, error) {
		if fileID != "file-rapid" {
			t.Fatalf("查询文件 ID = %s，期望 file-rapid", fileID)
		}
		return &v115open.FileDetail{
			FileId:       "file-rapid",
			PickCode:     "pick-detail",
			Sha1:         "sha1-detail",
			FileSizeByte: 2048,
			Ptime:        "123456",
		}, nil
	}
	t.Cleanup(func() {
		get115FileDetailByCid = oldResolver
	})

	got, err := build115RapidUploadCompleteResult(
		context.Background(),
		nil,
		&DbUploadTask{RemotePathId: "parent-1"},
		upload115LocalFileInfo{FileSha1: "sha1-local", FileSize: 1024},
		&v115open.UploadInitResult{Status: v115open.UploadInitStatusRapidUploaded, FileId: "file-rapid", PickCode: "pick-init"},
	)
	if err != nil {
		t.Fatalf("构造秒传完成结果失败: %v", err)
	}
	if got.FileId != "file-rapid" ||
		got.PickCode != "pick-detail" ||
		got.ParentId != "parent-1" ||
		got.Sha1 != "sha1-detail" ||
		got.Size != 2048 ||
		got.Mtime != 123456 {
		t.Fatalf("秒传完成结果 = %+v，期望补齐远端详情和 mtime", got)
	}
}

func TestBuild115RapidUploadCompleteResultFallsBackForDirectoryMonitor(t *testing.T) {
	oldResolver := get115FileDetailByCid
	get115FileDetailByCid = func(context.Context, *v115open.OpenClient, string) (*v115open.FileDetail, error) {
		return nil, errors.New("temporary detail error")
	}
	t.Cleanup(func() {
		get115FileDetailByCid = oldResolver
	})

	got, err := build115RapidUploadCompleteResult(
		context.Background(),
		nil,
		&DbUploadTask{Source: UploadSourceDirectoryMonitor, RemotePathId: "parent-1"},
		upload115LocalFileInfo{FileSha1: "sha1-local", FileSize: 1024},
		&v115open.UploadInitResult{Status: v115open.UploadInitStatusRapidUploaded, FileId: "file-rapid", PickCode: "pick-init"},
	)
	if err != nil {
		t.Fatalf("目录监控秒传详情查询失败时应使用 init 返回兜底: %v", err)
	}
	if got.FileId != "file-rapid" ||
		got.PickCode != "pick-init" ||
		got.ParentId != "parent-1" ||
		got.Sha1 != "sha1-local" ||
		got.Size != 1024 ||
		got.Mtime != 0 {
		t.Fatalf("兜底秒传完成结果 = %+v", got)
	}
}

func TestBuild115RapidUploadCompleteResultRequiresDetailForStrmSync(t *testing.T) {
	oldResolver := get115FileDetailByCid
	get115FileDetailByCid = func(context.Context, *v115open.OpenClient, string) (*v115open.FileDetail, error) {
		return nil, errors.New("temporary detail error")
	}
	t.Cleanup(func() {
		get115FileDetailByCid = oldResolver
	})

	_, err := build115RapidUploadCompleteResult(
		context.Background(),
		nil,
		&DbUploadTask{Source: UploadSourceStrm, RemotePathId: "parent-1"},
		upload115LocalFileInfo{FileSha1: "sha1-local", FileSize: 1024},
		&v115open.UploadInitResult{Status: v115open.UploadInitStatusRapidUploaded, FileId: "file-rapid", PickCode: "pick-init"},
	)
	if err == nil {
		t.Fatal("STRM 同步秒传详情查询失败时应返回错误，避免本地 mtime 不同步")
	}
}
