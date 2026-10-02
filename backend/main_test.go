package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"

	"github.com/gin-gonic/gin"
	"go.yaml.in/yaml/v3"
)

func TestSyncPathAggregateWriteRoutesReplaceLegacyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldLogger := helpers.AppLogger
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(&bytes.Buffer{}, "", 0)}
	t.Cleanup(func() { helpers.AppLogger = oldLogger })
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "web_statics"), 0o755); err != nil {
		t.Fatalf("创建测试静态目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "web_statics", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("创建测试 index.html 失败: %v", err)
	}
	t.Chdir(root)
	router := gin.New()
	setRouter(router)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{"POST /api/sync/paths", "PUT /api/sync/paths/:id"} {
		if _, ok := routes[expected]; !ok {
			t.Fatalf("缺少新同步目录写路由 %s", expected)
		}
	}
	for _, removed := range []string{
		"POST /api/sync/path-add",
		"POST /api/sync/path-update",
	} {
		if _, ok := routes[removed]; ok {
			t.Fatalf("旧写路由仍存在：%s", removed)
		}
	}
	for _, retained := range []string{
		"GET /api/sync/path/:id",
		"GET /api/directory-upload/rules",
		"POST /api/directory-upload/sync-paths/:sync_path_id/scan",
		"GET /api/directory-upload/runtime-status",
	} {
		if _, ok := routes[retained]; !ok {
			t.Fatalf("应保留的查询或运行接口缺失：%s", retained)
		}
	}
}

func TestLegacySyncWriteHandlersRemovedFromControllerSources(t *testing.T) {
	files := []struct {
		path    string
		removed []string
	}{
		{
			path: "internal/controllers/sync.go",
			removed: []string{
				"func AddSyncPath(",
				"func UpdateSyncPath(",
				"@Router /sync/path-add [post]",
				"@Router /sync/path-update [post]",
			},
		},
		{
			path: "internal/controllers/directory_upload.go",
			removed: []string{
				"func SaveDirectoryUploadSyncPathRules(",
			},
		},
	}
	for _, file := range files {
		source, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", file.path, err)
		}
		for _, removed := range file.removed {
			if strings.Contains(string(source), removed) {
				t.Fatalf("%s 仍包含旧写接口源码标记 %q", file.path, removed)
			}
		}
	}
}

func TestInitialDatabaseConfigSelection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		engine    helpers.DbEngine
		mode      helpers.PostgresType
		wantError bool
	}{
		{name: "SQLite", engine: helpers.DbEngineSqlite},
		{name: "PostgreSQL", engine: helpers.DbEnginePostgres},
		{name: "legacy external client", engine: helpers.DbEnginePostgres, mode: helpers.PostgresTypeExternal},
		{name: "embedded client", engine: helpers.DbEnginePostgres, mode: helpers.PostgresTypeEmbedded, wantError: true},
		{name: "unknown mode", engine: helpers.DbEnginePostgres, mode: "unknown", wantError: true},
		{name: "unknown engine", engine: "mysql", wantError: true},
		{name: "missing engine", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldDir := helpers.ConfigDir
			helpers.ConfigDir = t.TempDir()
			t.Cleanup(func() { helpers.ConfigDir = oldDir })
			req := databaseConfigRequest{
				Engine: tc.engine, PostgresType: tc.mode,
				Host: "db.example", Port: 5433, User: "postgres", Password: "test-password",
				Database: "qmediasync", SSL: true,
			}
			config, err := req.toConfig()
			if (err != nil) != tc.wantError {
				t.Fatalf("toConfig() error = %v, wantError %v", err, tc.wantError)
			}
			if tc.wantError {
				return
			}
			if err := helpers.SaveConfig(config); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(helpers.ConfigFilePath())
			if err != nil {
				t.Fatal(err)
			}
			var saved helpers.Config
			if err := yaml.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.Db.Engine != tc.engine || bytes.Contains(data, []byte("postgresType:")) {
				t.Fatalf("saved database engine or legacy mode is wrong: %+v", saved.Db)
			}
			if tc.engine == helpers.DbEnginePostgres {
				pg := saved.Db.PostgresConfig
				if pg.Host != req.Host || pg.Port != req.Port || pg.User != req.User ||
					pg.Password != req.Password || pg.Database != req.Database || !pg.SSL {
					t.Fatal("saved PostgreSQL connection settings do not match the selection")
				}
			}
		})
	}
}

func TestInitEnvRejectsLegacyDatabaseState(t *testing.T) {
	for _, tc := range []struct {
		name       string
		config     string
		legacyPath string
		wantError  string
	}{
		{
			name: "old database without config", legacyPath: "postgres/data/PG_VERSION",
			wantError: "已移除内嵌数据库和自动迁移",
		},
		{
			name: "pending migration", legacyPath: "backups/migrate.zip",
			config: "db:\n  engine: sqlite\n  sqliteFile: existing.db\n", wantError: "已不提供自动迁移",
		},
		{
			name: "pending migration without config", legacyPath: "backups/migrate.zip",
			wantError: "已不提供自动迁移",
		},
		{
			name: "embedded configuration", config: "db:\n  engine: postgres\n  postgresType: embedded\n",
			wantError: "已移除内嵌 PostgreSQL",
		},
		{
			name: "unknown engine", config: "db:\n  engine: unknown\n", wantError: "不支持的数据库引擎",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldRoot, oldDir, oldConfig := helpers.RootDir, helpers.ConfigDir, helpers.GlobalConfig
			oldLock, oldDB, oldFirstRun := instanceLock, db.Db, helpers.IsFirstRun
			oldOutput, oldTimeZone := log.Writer(), time.Local
			t.Cleanup(func() {
				if instanceLock != nil && instanceLock != oldLock {
					instanceLock.Close()
				}
				helpers.RootDir, helpers.ConfigDir, helpers.GlobalConfig = oldRoot, oldDir, oldConfig
				instanceLock, db.Db, helpers.IsFirstRun = oldLock, oldDB, oldFirstRun
				log.SetOutput(oldOutput)
				time.Local = oldTimeZone
			})
			t.Setenv("TRIM_PKGETC", "")
			t.Setenv("TRIM_DATA_SHARE_PATHS", "")
			t.Setenv("LOCALAPPDATA", t.TempDir())
			helpers.RootDir = t.TempDir()
			dir := filepath.Join(helpers.RootDir, "config")
			files := map[string]string{"existing.db": "existing database contents"}
			if tc.config != "" {
				files["config.yaml"] = tc.config
			}
			if tc.legacyPath != "" {
				files[tc.legacyPath] = "legacy data"
			}
			for path, content := range files {
				path = filepath.Join(dir, path)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var output bytes.Buffer
			log.SetOutput(&output)
			if initEnv() || db.Db != oldDB {
				t.Fatal("startup opened a database despite unsupported legacy state")
			}
			if !strings.Contains(output.String(), tc.wantError) {
				t.Fatalf("missing error %q in startup log: %s", tc.wantError, output.String())
			}
			for path, before := range files {
				after, err := os.ReadFile(filepath.Join(dir, path))
				if err != nil || string(after) != before {
					t.Fatalf("startup changed existing file %s", path)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "encryption.key")); !os.IsNotExist(err) {
				t.Fatal("rejected startup generated an encryption key")
			}
			if tc.config == "" && helpers.HasConfigFile() {
				t.Fatal("rejected startup generated a new configuration")
			}
		})
	}
}
