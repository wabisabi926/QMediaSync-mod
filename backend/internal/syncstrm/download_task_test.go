package syncstrm

import (
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"qmediasync/internal/db"
	"qmediasync/internal/models"
)

func TestAddMetaDownloadTaskCountsNewMeta(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&models.DbDownloadTask{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}

	s := &SyncStrm{}
	file := &models.SyncFile{
		SyncPathId:    10,
		SourceType:    models.SourceType115,
		PickCode:      "pick-meta",
		FileName:      "movie.nfo",
		LocalFilePath: "/media/movie/movie.nfo",
		SyncPath:      &models.SyncPath{},
	}

	if err := s.addMetaDownloadTask(file); err != nil {
		t.Fatalf("添加下载任务失败: %v", err)
	}
	if got := atomic.LoadInt64(&s.NewMeta); got != 1 {
		t.Fatalf("NewMeta = %d，期望 1", got)
	}

	var task models.DbDownloadTask
	if err := db.Db.Where("remote_file_id = ?", "pick-meta").First(&task).Error; err != nil {
		t.Fatalf("查询下载任务失败: %v", err)
	}
	if task.SyncPathId != 10 {
		t.Fatalf("下载任务 sync_path_id = %d，期望 10", task.SyncPathId)
	}
}
