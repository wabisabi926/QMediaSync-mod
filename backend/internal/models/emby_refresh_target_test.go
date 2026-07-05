package models

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"qmediasync/internal/db"
)

func createEmbyRefreshSyncFile(t *testing.T, syncPathID uint, path, fileName, pickCode string) *SyncFile {
	t.Helper()
	syncFile := &SyncFile{
		SyncPathId:    syncPathID,
		AccountId:     1,
		SourceType:    SourceType115,
		FileId:        "file-" + pickCode,
		ParentId:      "parent-" + pickCode,
		PickCode:      pickCode,
		Path:          path,
		FileName:      fileName,
		LocalFilePath: "/strm" + path + "/" + fileName[:len(fileName)-4] + ".strm",
		IsVideo:       true,
	}
	if err := db.Db.Create(syncFile).Error; err != nil {
		t.Fatalf("创建 SyncFile 失败: %v", err)
	}
	return syncFile
}

func createEmbyRefreshItem(t *testing.T, itemID string, itemType string, seasonID string, seriesID string) {
	t.Helper()
	if err := db.Db.Create(&EmbyMediaItem{
		ItemId:     itemID,
		ItemIdInt:  helpersStringToInt64ForTest(itemID),
		Name:       "item-" + itemID,
		Type:       itemType,
		SeasonId:   seasonID,
		SeriesId:   seriesID,
		LibraryId:  "lib-tv",
		Path:       "/strm/remote/item-" + itemID + ".strm",
		PickCode:   "pick-" + itemID,
		LastSeenAt: nowUnix(),
	}).Error; err != nil {
		t.Fatalf("创建 EmbyMediaItem 失败: %v", err)
	}
}

func helpersStringToInt64ForTest(value string) int64 {
	var id int64
	for _, r := range value {
		if r < '0' || r > '9' {
			continue
		}
		id = id*10 + int64(r-'0')
	}
	return id
}

func TestResolveEmbyRefreshTargetUsesExistingMovieItem(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	syncFile := createEmbyRefreshSyncFile(t, 10, "/remote/movie", "movie.mkv", "movie")
	createEmbyRefreshItem(t, "101", "Movie", "", "")
	if err := db.Db.Create(&EmbyMediaSyncFile{SyncPathId: 10, EmbyItemId: 101, SyncFileId: syncFile.ID, PickCode: syncFile.PickCode}).Error; err != nil {
		t.Fatalf("创建关联失败: %v", err)
	}

	target, err := ResolveEmbyRefreshTarget(syncFile)
	if err != nil {
		t.Fatalf("解析刷新目标失败: %v", err)
	}
	if target.TargetType != EmbyRefreshTargetTypeItem || target.ItemID != "101" || target.Recursive {
		t.Fatalf("刷新目标 = %+v，期望 Movie item 101 非递归刷新", target)
	}
}

func TestResolveEmbyRefreshTargetUsesExistingEpisodeItem(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	syncFile := createEmbyRefreshSyncFile(t, 10, "/remote/show/Season 01", "episode.mkv", "episode")
	createEmbyRefreshItem(t, "201", "Episode", "301", "401")
	if err := db.Db.Create(&EmbyMediaSyncFile{SyncPathId: 10, EmbyItemId: 201, SyncFileId: syncFile.ID, PickCode: syncFile.PickCode}).Error; err != nil {
		t.Fatalf("创建关联失败: %v", err)
	}

	target, err := ResolveEmbyRefreshTarget(syncFile)
	if err != nil {
		t.Fatalf("解析刷新目标失败: %v", err)
	}
	if target.TargetType != EmbyRefreshTargetTypeItem || target.ItemID != "201" || target.Recursive {
		t.Fatalf("刷新目标 = %+v，期望 Episode item 201 非递归刷新", target)
	}
}

func TestResolveEmbyRefreshTargetUsesSeasonThenSeriesForNewEpisode(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	existing := createEmbyRefreshSyncFile(t, 10, "/remote/show/Season 01", "episode01.mkv", "ep1")
	createEmbyRefreshItem(t, "201", "Episode", "301", "401")
	if err := db.Db.Create(&EmbyMediaSyncFile{SyncPathId: 10, EmbyItemId: 201, SyncFileId: existing.ID, PickCode: existing.PickCode}).Error; err != nil {
		t.Fatalf("创建关联失败: %v", err)
	}
	newEpisode := createEmbyRefreshSyncFile(t, 10, "/remote/show/Season 01", "episode02.mkv", "ep2")

	target, err := ResolveEmbyRefreshTarget(newEpisode)
	if err != nil {
		t.Fatalf("解析刷新目标失败: %v", err)
	}
	if target.ItemID != "301" || target.ItemType != "Season" || !target.Recursive {
		t.Fatalf("刷新目标 = %+v，期望优先刷新 Season 301", target)
	}

	if err := db.Db.Model(&EmbyMediaItem{}).Where("item_id = ?", "201").Update("season_id", "").Error; err != nil {
		t.Fatalf("清空 season_id 失败: %v", err)
	}
	target, err = ResolveEmbyRefreshTarget(newEpisode)
	if err != nil {
		t.Fatalf("解析刷新目标失败: %v", err)
	}
	if target.ItemID != "401" || target.ItemType != "Series" || !target.Recursive {
		t.Fatalf("刷新目标 = %+v，期望回退刷新 Series 401", target)
	}
}

func TestResolveEmbyRefreshTargetFallsBackToPathLookupWhenLocalRelationIsStale(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	syncFile := createEmbyRefreshSyncFile(t, 10, "/remote/movie", "movie.mkv", "stale")
	if err := db.Db.Create(&EmbyMediaSyncFile{SyncPathId: 10, EmbyItemId: 999, SyncFileId: syncFile.ID, PickCode: syncFile.PickCode}).Error; err != nil {
		t.Fatalf("创建陈旧关联失败: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/emby/Items" || r.URL.Query().Get("Path") != syncFile.LocalFilePath {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"Items":[{"Id":"501","Name":"path movie","Type":"Movie","Path":"`+syncFile.LocalFilePath+`"}],"TotalRecordCount":1}`)
	}))
	defer server.Close()
	GlobalEmbyConfig.EmbyUrl = server.URL

	target, err := ResolveEmbyRefreshTarget(syncFile)
	if err != nil {
		t.Fatalf("解析刷新目标失败: %v", err)
	}
	if target.ItemID != "501" || target.ItemType != "Movie" || target.Recursive {
		t.Fatalf("刷新目标 = %+v，期望路径兜底 Movie 501", target)
	}
}

func TestResolveEmbyRefreshTargetFallsBackToPathLookupWhenCachedItemIDIsMissingInEmby(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	syncFile := createEmbyRefreshSyncFile(t, 10, "/remote/movie", "movie.mkv", "stale-id")
	createEmbyRefreshItem(t, "101", "Movie", "", "")
	if err := db.Db.Model(&EmbyMediaItem{}).Where("item_id = ?", "101").Update("last_seen_at", 0).Error; err != nil {
		t.Fatalf("标记本地 Emby item 缓存未确认失败: %v", err)
	}
	if err := db.Db.Create(&EmbyMediaSyncFile{SyncPathId: 10, EmbyItemId: 101, SyncFileId: syncFile.ID, PickCode: syncFile.PickCode}).Error; err != nil {
		t.Fatalf("创建关联失败: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/emby/Items" && r.URL.Query().Get("Ids") == "101":
			fmt.Fprint(w, `{"Items":[],"TotalRecordCount":0}`)
		case r.URL.Path == "/emby/Items" && r.URL.Query().Get("Path") == syncFile.LocalFilePath:
			fmt.Fprint(w, `{"Items":[{"Id":"501","Name":"path movie","Type":"Movie","Path":"`+syncFile.LocalFilePath+`"}],"TotalRecordCount":1}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	GlobalEmbyConfig.EmbyUrl = server.URL

	target, err := ResolveEmbyRefreshTarget(syncFile)
	if err != nil {
		t.Fatalf("解析刷新目标失败: %v", err)
	}
	if target.ItemID != "501" || target.ItemType != "Movie" || target.Recursive {
		t.Fatalf("刷新目标 = %+v，期望 stale item id 走路径兜底 Movie 501", target)
	}
}

func TestRequestEmbyRefreshBySyncFileFallsBackToLibrary(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	syncFile := createEmbyRefreshSyncFile(t, 10, "/remote/movie", "movie.mkv", "fallback")
	syncFile.LocalFilePath = ""

	if err := RequestEmbyRefreshBySyncFile(syncFile); err != nil {
		t.Fatalf("提交刷新失败: %v", err)
	}
	var task EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-movie").First(&task).Error; err != nil {
		t.Fatalf("读取媒体库刷新任务失败: %v", err)
	}
	if task.TargetType != "" && task.TargetType != string(EmbyRefreshTargetTypeLibrary) {
		t.Fatalf("target_type = %s，期望空或 library 兼容值", task.TargetType)
	}
}

func TestRequestEmbyRefreshBySyncFileDedupesSameItem(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	syncFile := createEmbyRefreshSyncFile(t, 10, "/remote/movie", "movie.mkv", "dedupe")
	createEmbyRefreshItem(t, "101", "Movie", "", "")
	if err := db.Db.Create(&EmbyMediaSyncFile{SyncPathId: 10, EmbyItemId: 101, SyncFileId: syncFile.ID, PickCode: syncFile.PickCode}).Error; err != nil {
		t.Fatalf("创建关联失败: %v", err)
	}

	if err := RequestEmbyRefreshBySyncFile(syncFile); err != nil {
		t.Fatalf("首次提交刷新失败: %v", err)
	}
	if err := RequestEmbyRefreshBySyncFile(syncFile); err != nil {
		t.Fatalf("重复提交刷新失败: %v", err)
	}

	var tasks []EmbyLibraryRefreshTask
	if err := db.Db.Find(&tasks).Error; err != nil {
		t.Fatalf("查询刷新任务失败: %v", err)
	}
	if len(tasks) != 1 || tasks[0].TargetType != string(EmbyRefreshTargetTypeItem) {
		t.Fatalf("刷新任务 = %+v，期望单个 item 任务", tasks)
	}
	itemIDs := tasks[0].GetItemIds()
	if len(itemIDs) != 1 || itemIDs[0] != "101" || tasks[0].LibraryId != "item:101" {
		t.Fatalf("item_ids=%v library_id=%s，期望 item 101 去抖任务", itemIDs, tasks[0].LibraryId)
	}
}

func TestRequestEmbyRefreshTargetsDedupesItemsAndLibraries(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})

	targets := []EmbyRefreshTarget{
		{
			TargetType:          EmbyRefreshTargetTypeItem,
			ItemID:              "101",
			ItemName:            "电影 1",
			FallbackLibraryId:   "lib-movie",
			FallbackLibraryName: "电影",
		},
		{
			TargetType:          EmbyRefreshTargetTypeItem,
			ItemID:              "101",
			ItemName:            "电影 1",
			FallbackLibraryId:   "lib-movie",
			FallbackLibraryName: "电影",
		},
		{
			TargetType:          EmbyRefreshTargetTypeLibrary,
			FallbackLibraryId:   "lib-movie",
			FallbackLibraryName: "电影",
		},
	}

	if err := RequestEmbyRefreshTargets(10, targets); err != nil {
		t.Fatalf("批量提交刷新失败: %v", err)
	}
	var tasks []EmbyLibraryRefreshTask
	if err := db.Db.Find(&tasks).Error; err != nil {
		t.Fatalf("查询刷新任务失败: %v", err)
	}
	if len(tasks) != 1 || tasks[0].TargetType != EmbyLibraryRefreshTargetTypeLibrary || tasks[0].LibraryId != "lib-movie" {
		t.Fatalf("刷新任务 = %+v，期望只提交 lib-movie 媒体库刷新", tasks)
	}
}

func TestRequestEmbyRefreshTargetsKeepsItemsAcrossDifferentLibraries(t *testing.T) {
	setupEmbyRefreshTestDB(t)

	targets := []EmbyRefreshTarget{
		{
			TargetType:        EmbyRefreshTargetTypeItem,
			ItemID:            "101",
			ItemName:          "电影",
			FallbackLibraryId: "lib-movie",
		},
		{
			TargetType:        EmbyRefreshTargetTypeItem,
			ItemID:            "301",
			ItemName:          "第一季",
			ItemType:          "Season",
			Recursive:         true,
			FallbackLibraryId: "lib-tv",
		},
	}
	if err := RequestEmbyRefreshTargets(10, targets); err != nil {
		t.Fatalf("批量提交刷新失败: %v", err)
	}
	var tasks []EmbyLibraryRefreshTask
	if err := db.Db.Order("library_id ASC").Find(&tasks).Error; err != nil {
		t.Fatalf("查询刷新任务失败: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("刷新任务数量 = %d，期望 2", len(tasks))
	}
}

func TestRequestEmbyRefreshTargetsSetsDebounceTime(t *testing.T) {
	setupEmbyRefreshTestDB(t)

	if err := RequestEmbyRefreshTargets(10, []EmbyRefreshTarget{
		{
			TargetType:        EmbyRefreshTargetTypeItem,
			ItemID:            "101",
			ItemName:          "电影",
			FallbackLibraryId: "lib-movie",
		},
	}); err != nil {
		t.Fatalf("批量提交刷新失败: %v", err)
	}
	var task EmbyLibraryRefreshTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("查询刷新任务失败: %v", err)
	}
	want := task.LastEventAt + DefaultEmbyRefreshDebounceSeconds
	if task.RefreshAfterAt != want {
		t.Fatalf("refresh_after_at = %d，期望 %d", task.RefreshAfterAt, want)
	}
}

func TestEmbyRefreshTargetWaitsForSameSyncPathDownloads(t *testing.T) {
	setupEmbyRefreshTestDB(t)

	now := nowUnix()
	task := newPendingEmbyLibraryRefreshTask("lib-tv", "剧集", []uint{10}, now-DefaultEmbyRefreshDebounceSeconds-1)
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建刷新任务失败: %v", err)
	}
	for _, status := range []DownloadStatus{DownloadStatusPending, DownloadStatusDownloading} {
		db.Db.Exec("DELETE FROM db_download_tasks")
		if err := db.Db.Create(&DbDownloadTask{SyncPathId: 10, Status: status}).Error; err != nil {
			t.Fatalf("创建下载任务失败: %v", err)
		}
		ready, reason, err := IsEmbyLibraryRefreshTaskReady(task, now)
		if err != nil {
			t.Fatalf("检查刷新任务失败: %v", err)
		}
		if ready || reason != "download_running" {
			t.Fatalf("下载状态 %s 时 ready=%v reason=%s，期望等待同 sync_path_id 下载完成", status, ready, reason)
		}
	}
}
