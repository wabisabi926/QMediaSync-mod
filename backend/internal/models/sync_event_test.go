package models

import "testing"

func TestSyncLogRelativePath(t *testing.T) {
	got := SyncLogRelativePath(42)
	if got != "libs/sync_42.log" {
		t.Fatalf("SyncLogRelativePath = %s，期望 libs/sync_42.log", got)
	}
}

func TestSyncTaskEventPayloadUsesRealSyncID(t *testing.T) {
	sync := &Sync{
		BaseModel:         BaseModel{ID: 42, CreatedAt: 100, UpdatedAt: 110},
		SyncPathId:        7,
		Status:            SyncStatusInProgress,
		SubStatus:         SyncSubStatusProcessNetFileList,
		Total:             8,
		NewStrm:           3,
		NewMeta:           2,
		NewUpload:         1,
		NetFileStartAt:    120,
		NetFileFinishAt:   140,
		LocalFileStartAt:  141,
		LocalFileFinishAt: 160,
		LocalPath:         "/media",
		RemotePath:        "/cloud",
	}

	payload := sync.SyncTaskEventPayload()
	if payload.SyncID != 42 {
		t.Fatalf("sync_id = %d，期望 42", payload.SyncID)
	}
	if payload.SyncPathID != 7 {
		t.Fatalf("sync_path_id = %d，期望 7", payload.SyncPathID)
	}
	if payload.LogPath != "libs/sync_42.log" {
		t.Fatalf("log_path = %s，期望 libs/sync_42.log", payload.LogPath)
	}
	if payload.NetFileStartAt != 120 {
		t.Fatalf("net_file_start_at = %d，期望 120", payload.NetFileStartAt)
	}
	if payload.NetFileFinishAt != 140 {
		t.Fatalf("net_file_finish_at = %d，期望 140", payload.NetFileFinishAt)
	}
	if payload.LocalFileStartAt != 141 {
		t.Fatalf("local_file_start_at = %d，期望 141", payload.LocalFileStartAt)
	}
	if payload.LocalFileFinishAt != 160 {
		t.Fatalf("local_file_finish_at = %d，期望 160", payload.LocalFileFinishAt)
	}
}
