package emby

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSyncEmbyItemByID使用ItemsIds单条Upsert且关联幂等(t *testing.T) {
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	models.GlobalEmbyConfig = nil
	SetEmbySyncRunning(false)

	if err := db.Db.AutoMigrate(&models.EmbyConfig{}, &models.EmbyMediaItem{}, &models.EmbyMediaSyncFile{}, &models.EmbyLibrarySyncPath{}, &models.SyncFile{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	if err := db.Db.Create(&models.SyncFile{PickCode: "pc-1", SyncPathId: 12}).Error; err != nil {
		t.Fatalf("创建 SyncFile 失败: %v", err)
	}

	requestedIDs := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/emby/Items" {
			http.NotFound(w, r)
			return
		}
		requestedIDs = append(requestedIDs, r.URL.Query().Get("Ids"))
		fmt.Fprint(w, `{"TotalRecordCount":1,"Items":[{"Id":"122145","Name":"阿雅与魔女","Type":"Movie","ParentId":"lib-a","DateCreated":"2026-06-29T00:00:00Z","DateModified":"2026-06-29T00:10:00Z","MediaSources":[{"Path":"http://qms.local/stream?pickcode=pc-1"}]}]}`)
	}))
	defer server.Close()

	if err := db.Db.Create(&models.EmbyConfig{EmbyUrl: server.URL, EmbyApiKey: "test-key", SyncEnabled: 1, SyncCron: "0 * * * *"}).Error; err != nil {
		t.Fatalf("创建 EmbyConfig 失败: %v", err)
	}

	for i := 0; i < 2; i++ {
		changed, err := SyncEmbyItemByID("122145")
		if err != nil {
			t.Fatalf("SyncEmbyItemByID() error = %v", err)
		}
		if !changed {
			t.Fatal("SyncEmbyItemByID() changed = false, want true")
		}
	}

	if len(requestedIDs) != 2 || requestedIDs[0] != "122145" || requestedIDs[1] != "122145" {
		t.Fatalf("requested Ids = %v, want [122145 122145]", requestedIDs)
	}
	var item models.EmbyMediaItem
	if err := db.Db.Where("item_id = ?", "122145").First(&item).Error; err != nil {
		t.Fatalf("查询 EmbyMediaItem 失败: %v", err)
	}
	if item.Name != "阿雅与魔女" || item.LibraryId != "lib-a" || item.PickCode != "pc-1" {
		t.Fatalf("item = %+v, want name/library/pickcode updated", item)
	}
	var relationCount int64
	if err := db.Db.Model(&models.EmbyMediaSyncFile{}).Where("emby_item_id = ?", uint(122145)).Count(&relationCount).Error; err != nil {
		t.Fatalf("统计 EmbyMediaSyncFile 失败: %v", err)
	}
	if relationCount != 1 {
		t.Fatalf("relationCount = %d, want 1", relationCount)
	}
}

func TestSyncEmbyItemByID空结果不触发全量同步(t *testing.T) {
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	models.GlobalEmbyConfig = nil
	SetEmbySyncRunning(false)

	if err := db.Db.AutoMigrate(&models.EmbyConfig{}, &models.EmbyMediaItem{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"TotalRecordCount":0,"Items":[]}`)
	}))
	defer server.Close()
	if err := db.Db.Create(&models.EmbyConfig{EmbyUrl: server.URL, EmbyApiKey: "test-key", SyncEnabled: 1, SyncCron: "0 * * * *"}).Error; err != nil {
		t.Fatalf("创建 EmbyConfig 失败: %v", err)
	}

	changed, err := SyncEmbyItemByID("missing")
	if err != nil {
		t.Fatalf("SyncEmbyItemByID() error = %v", err)
	}
	if changed {
		t.Fatal("SyncEmbyItemByID() changed = true, want false")
	}
	var total int64
	if err := db.Db.Model(&models.EmbyMediaItem{}).Count(&total).Error; err != nil {
		t.Fatalf("统计 EmbyMediaItem 失败: %v", err)
	}
	if total != 0 {
		t.Fatalf("total items = %d, want 0", total)
	}
}
