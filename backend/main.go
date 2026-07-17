package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"qmediasync/emby302/config"
	emby302https "qmediasync/emby302/util/https"
	"qmediasync/emby302/util/logs/colors"
	"qmediasync/emby302/web"
	"qmediasync/internal/backup"
	"qmediasync/internal/controllers"
	"qmediasync/internal/db"
	"qmediasync/internal/db/database"
	"qmediasync/internal/github"
	"qmediasync/internal/helpers"
	"qmediasync/internal/migrate"
	"qmediasync/internal/models"
	"qmediasync/internal/synccron"
	"qmediasync/internal/syncstrm"
	"qmediasync/internal/v115open"
	"qmediasync/internal/websocket"

	"github.com/gin-gonic/gin"
)

var Version string = "v0.15.11"
var PublishDate string = "2025-08-08"
var OAuthRelayEncryptionKey = ""

var AppName string = "QMediaSync"
var QMSApp *App

func parseBuildUnixTime(value string) int64 {
	if value == "" {
		return 0
	}

	if timestamp, err := helpers.ParseRFC3339Unix(value); err == nil {
		return timestamp
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Unix()
		}
	}

	return 0
}

type App struct {
	isRelease   bool
	dbManager   *database.EmbeddedManager
	httpServer  *http.Server
	httpsServer *http.Server
	version     string
	publishDate string
}

func (app *App) Start() {
	// 启动外网 302 服务
	startEmby302()
	if helpers.IsRelease {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(controllers.Cors())
	setRouter(r)
	app.StartHttpServer(r)
	app.StartHttpsServer(r)
	if runtime.GOOS == "windows" {
		// 监听 Ctrl+C 信号
		go func() {
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit
			log.Println("收到 Ctrl+C 信号")
			helpers.ExitChan <- struct{}{}
		}()
		<-helpers.ExitChan
		log.Println("收到停止信号")
		app.Stop()
		close(helpers.ExitChan)
		log.Println("应用程序正常退出")
		return
	} else {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("收到停止信号")
		// 停止应用
		app.Stop()
		log.Println("应用程序正常退出")
	}
}

func (app *App) Stop() {
	app.shutdownHTTPServers()
	// 关闭同步任务执行队列
	synccron.PauseAllNewSyncQueues()
	// 关闭上传下载队列
	models.GlobalDownloadQueue.Stop()
	models.GlobalUploadQueue.Stop()
	// 关闭 STRM 生成 worker
	syncstrm.StopStrmGenerationWorker()
	// 关闭定时任务（包含备份定时任务）
	synccron.GlobalCron.Stop()
	// 关闭数据库
	if app.dbManager != nil {
		app.dbManager.Stop()
	}
	helpers.CloseLogger() // 关闭日志
}

// shutdownHTTPServers 停止接收新请求并等待现有 handler 退出。
func (app *App) shutdownHTTPServers() {
	if app.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.httpServer.Shutdown(ctx); err != nil {
			log.Println("HTTP Server Shutdown:", err)
		}
	}
	if app.httpsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.httpsServer.Shutdown(ctx); err != nil {
			log.Println("HTTPS Server Shutdown:", err)
		}
	}
}

func (app *App) StartHttpsServer(r *gin.Engine) {
	certFile := filepath.Join(helpers.RootDir, "config", "server.crt")
	keyFile := filepath.Join(helpers.RootDir, "config", "server.key")
	if !helpers.PathExists(certFile) || !helpers.PathExists(keyFile) {
		return
	}
	go func() {
		// 在 12332 端口上启动 HTTPS 服务
		sslHost := ""
		// 启动 Web Server
		if !helpers.IsRelease {
			sslHost = "localhost:12332"
		} else {
			sslHost = helpers.GlobalConfig.HttpsHost
		}
		app.httpsServer = &http.Server{
			Addr:    sslHost,
			Handler: r,
		}
		// 没有证书则回退到普通 HTTP
		weberr := app.httpsServer.ListenAndServeTLS(certFile, keyFile)
		if weberr != nil {
			fmt.Println("ListenAndServe error:", weberr)
		}
	}()
}

func (app *App) StartHttpServer(r *gin.Engine) {
	host := helpers.GlobalConfig.HttpHost
	// 同时在 12333 端口上启动 HTTP 服务
	app.httpServer = &http.Server{
		Addr:    host,
		Handler: r,
	}
	go func() {
		weberr := app.httpServer.ListenAndServe()
		if weberr != nil {
			fmt.Println("ListenAndServe error:", weberr)
		}
	}()
}

func (app *App) StartDatabase(migrateMode bool) error {
	// 根据配置启动数据库连接
	if helpers.GlobalConfig.Db.Engine == helpers.DbEngineSqlite {
		// 如果是 SQLite，直接初始化 SQLite 连接
		sqliteFile := filepath.Join(helpers.ConfigDir, helpers.GlobalConfig.Db.SqliteFile)
		helpers.AppLogger.Infof("SQLite 数据库文件路径：%s", sqliteFile)
		db.Db = db.InitSqlite3(sqliteFile)
		models.Migrate()
		if err := models.ResetStaleEmbySyncRunOnStartup(); err != nil {
			return err
		}
		return nil
	}

	// 初始化数据库配置
	dbConfig := &database.Config{
		Mode:         helpers.GlobalConfig.Db.PostgresType,
		Host:         helpers.GlobalConfig.Db.PostgresConfig.Host,
		Port:         helpers.GlobalConfig.Db.PostgresConfig.Port,
		User:         helpers.GlobalConfig.Db.PostgresConfig.User,
		Password:     helpers.GlobalConfig.Db.PostgresConfig.Password,
		DBName:       helpers.GlobalConfig.Db.PostgresConfig.Database,
		SSLMode:      "disable",
		LogDir:       filepath.Join(helpers.ConfigDir, "postgres", "log"),
		DataDir:      filepath.Join(helpers.ConfigDir, "postgres", "data"),
		BinaryPath:   db.GetPostgresBinaryPath(helpers.DataDir),
		MaxOpenConns: helpers.GlobalConfig.Db.PostgresConfig.MaxOpenConns,
		MaxIdleConns: helpers.GlobalConfig.Db.PostgresConfig.MaxIdleConns,
	}
	if helpers.GlobalConfig.Db.PostgresConfig.SSL {
		dbConfig.SSLMode = "require"
	}
	if dbConfig.Mode == helpers.PostgresTypeEmbedded {
		// 如果使用内置数据库，则需要启动和初始化数据库
		app.dbManager = database.NewEmbeddedManager(dbConfig)
		// 启动数据库
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := app.dbManager.Start(ctx); err != nil {
			return err
		}
		db.InitPostgres(app.dbManager.GetDB())

		// 如果是迁移模式，启动迁移服务
		if migrateMode {
			helpers.AppLogger.Info("检测到使用内嵌 PostgreSQL，启动迁移服务…")
			migrateServer := migrate.NewMigrateServer(app.dbManager, dbConfig)
			if err := migrateServer.Start(); err != nil {
				helpers.AppLogger.Errorf("启动迁移服务失败：%v", err)
				return err
			}
		}
	} else {
		// 初始化 PostgreSQL 数据库连接
		if err := db.ConnectPostgres(dbConfig); err != nil {
			return err
		}
	}
	models.Migrate()
	if err := models.ResetStaleEmbySyncRunOnStartup(); err != nil {
		return err
	}
	return nil
}

func configureInitialAdminSetup() error {
	hasUser, err := models.HasAnyUser()
	if err != nil {
		return err
	}
	if !hasUser {
		helpers.AppLogger.Warnf("检测到系统尚未创建管理员，请访问 Web 页面创建首个管理员")
	}
	return nil
}

func newApp() {
	if QMSApp != nil {
		log.Println("App 已经初始化，不能再次初始化")
		return
	}
	// 初始化 App
	QMSApp = &App{
		isRelease:   helpers.IsRelease,
		version:     Version,
		publishDate: PublishDate,
	}
}

func initTimeZone() {
	cstZone := time.FixedZone("CST", 8*3600)
	time.Local = cstZone
}

func checkRelease() {
	if helpers.IsRunningInDocker() {
		helpers.IsRelease = true
	}
	arg1 := strings.ToLower(os.Args[0])
	name := strings.ToLower(filepath.Base(arg1))
	helpers.IsRelease = strings.Index(name, "qmediasync") == 0 && !strings.Contains(arg1, "go-build")
}

func getRootDir() string {
	var exPath string = "/app" // 默认使用 Docker 路径
	checkRelease()
	if os.Getenv("TRIM_APPDEST") != "" {
		helpers.RootDir = os.Getenv("TRIM_APPDEST")
		return helpers.RootDir
	}
	if helpers.IsRelease {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath = filepath.Dir(ex)
	} else {
		exPath, _ = os.Getwd()
	}
	helpers.RootDir = exPath // 获取当前工作目录
	return exPath
}

// 获取用户数据目录
func getDataAndConfigDir() {
	var appData string
	var dataDir string
	var configDir string
	needMk := false
	if runtime.GOOS == "windows" {
		// 使用 AppData 目录，用户有完全控制权限
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = os.Getenv("APPDATA")
		}
		dataDir = filepath.Join(helpers.RootDir, "postgres")      // 数据库目录
		oldConfigDir := filepath.Join(appData, AppName, "config") // 配置目录
		configDir = filepath.Join(helpers.RootDir, "config")      // 配置目录
		err := os.MkdirAll(dataDir, 0755)
		if err != nil {
			fmt.Printf("创建数据目录失败：%v\n", err)
			panic("创建数据目录失败")
		}
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			fmt.Printf("创建配置目录失败：%v\n", err)
			panic("创建配置目录失败")
		}
		helpers.DataDir = dataDir
		helpers.ConfigDir = configDir
		if helpers.PathExists(oldConfigDir) {
			// 迁移旧配置
			err := helpers.MoveDir(oldConfigDir, configDir)
			if err != nil {
				fmt.Printf("迁移旧配置目录失败：%v\n", err)
				panic("迁移旧配置目录失败")
			}
			// 删除旧目录
			err = os.RemoveAll(oldConfigDir)
			if err != nil {
				fmt.Printf("删除旧配置目录失败：%v\n", err)
				panic("删除旧配置目录失败")
			}
		}
	} else {
		if os.Getenv("TRIM_PKGETC") == "" {
			appData = helpers.RootDir
			configDir = filepath.Join(appData, "config") // 配置目录
			dataDir = filepath.Join(appData, "postgres") // 数据库目录
			needMk = true
			helpers.DataDir = dataDir
			helpers.ConfigDir = configDir
		} else {
			oldConfigDir := os.Getenv("TRIM_PKGETC")
			configDir = os.Getenv("TRIM_DATA_SHARE_PATHS")
			if configDir == "" {
				configDir = oldConfigDir
				needMk = false
			} else {
				configDir = filepath.Join(configDir, "config")
				needMk = true
				// 检查是否需要迁移文件
				// oldConfigDir 必须存在且不为空
				if helpers.PathExists(oldConfigDir) && oldConfigDir != configDir {
					// 检查 oldConfigDir 是否为空目录
					if !helpers.IsDirEmpty(oldConfigDir) {
						err := os.MkdirAll(configDir, 0755)
						if err != nil {
							log.Printf("创建配置目录失败：%v\n", err)
							panic("创建配置目录失败")
						}
						// 迁移旧配置
						err = helpers.MoveDir(oldConfigDir, configDir)
						if err != nil {
							log.Printf("迁移旧配置目录失败：%v\n", err)
							panic("迁移旧配置目录失败")
						}
						needMk = false
					}
				}
			}
			dataDir = filepath.Join(configDir, "postgres") // 数据库目录
			helpers.DataDir = dataDir
			helpers.ConfigDir = configDir
		}
	}
	if needMk {
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			log.Printf("创建配置目录失败：%v\n", err)
			panic("创建配置目录失败")
		}
	}
}

//go:embed emby302.yaml
//go:embed assets/db_config.html
//go:embed assets/migrate.html
//go:embed web_statics
var embedFiles embed.FS

func init() {
	migrate.SetMigrateFiles(embedFiles)
}

func startEmby302() {
	dataRoot := helpers.ConfigDir
	data, err := embedFiles.ReadFile("emby302.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := config.ReadFromFile(data); err != nil {
		log.Fatal(err)
	}
	if models.GlobalEmbyConfig == nil || models.GlobalEmbyConfig.EmbyUrl == "" {
		helpers.AppLogger.Warnf("Emby 302 未配置 Emby 地址，跳过启动 Emby 302 服务")
		return
	}
	emby302https.ConfigureClient(emby302https.ClientOptions{
		InsecureSkipVerify: helpers.GlobalConfig.Emby302.InsecureSkipVerify,
	})
	if helpers.GlobalConfig.Emby302.InsecureSkipVerify {
		helpers.AppLogger.RequiredWarnf("Emby 302 已开启 insecure_skip_verify，出站 HTTPS 请求将跳过证书校验，存在中间人攻击风险，仅建议在受控内网自签名证书场景临时使用")
	}
	config.C.Emby.Host = models.GlobalEmbyConfig.EmbyUrl
	config.C.Emby.EpisodesUnplayPrior = false // 关闭剧集排序
	certFile := filepath.Join(dataRoot, "server.crt")
	keyFile := filepath.Join(dataRoot, "server.key")
	if helpers.PathExists(certFile) && helpers.PathExists(keyFile) {
		config.C.Ssl.Enable = true
		config.C.Ssl.SinglePort = false
		config.C.Ssl.Crt = "server.crt"
		config.C.Ssl.Key = "server.key"
	}
	config.BasePath = dataRoot
	config.C.Emby.LocalMediaRoot = "/"
	config.C.VideoPreview.Enable = true
	config.C.VideoPreview.Containers = []string{"strm"}
	go func() {
		if err := web.Listen(); err != nil {
			log.Fatal(colors.ToRed(err.Error()))
		}
	}()

}

func initLogger() {
	logPath := filepath.Join(helpers.ConfigDir, "logs")
	os.MkdirAll(logPath, 0755) // 如果没有 logs 目录则创建
	syncLogPath := filepath.Join(helpers.ConfigDir, helpers.SyncLogDir())
	os.MkdirAll(syncLogPath, 0755) // 如果没有同步任务日志目录则创建
	logConfig := helpers.LogConfigSnapshot()
	helpers.AppLogger = helpers.NewLogger(logConfig.App, true, true)
	helpers.V115Log = helpers.NewLogger(logConfig.V115, false, true)
	helpers.OpenListLog = helpers.NewLogger(logConfig.OpenList, false, true)
	helpers.BaiduPanLog = helpers.NewLogger(logConfig.BaiduPan, false, true)
	helpers.WarnUnsafeSensitiveLogIfEnabled()
}

func initOthers() {
	helpers.InitEventBus() // 初始化事件总线
	models.LoadSettings()  // 从数据库加载设置
	// 初始化 GitHub 访问管理器
	github.InitManager(models.SettingsGlobal.HttpProxy)
	helpers.AppLogger.Infof("已加载配置，准备初始化 115 请求队列，线程数：%d", models.SettingsGlobal.FileDetailThreads)
	qps := models.SettingsGlobal.FileDetailThreads
	if qps <= 0 {
		qps = 2
	}
	v115open.SetGlobalExecutorConfig(qps, qps*60, qps*3600)
	models.InitDQ()                      // 初始化下载队列
	models.InitUQ()                      // 初始化上传队列
	models.InitNotificationManager()     // 初始化通知管理器
	controllers.StartListenTelegramBot() // 初始化 Telegram Bot 监听
	models.GetEmbyConfig()               // 加载 Emby 配置
	helpers.SubscribeSync(helpers.V115TokenInValidEvent, models.HandleV115TokenInvalid)
	helpers.SubscribeSync(helpers.SaveOpenListTokenEvent, models.HandleOpenListTokenSaveSync)
	models.FailAllRunningSyncTasks()   // 将所有运行中的同步任务设置为失败状态
	synccron.RefreshOAuthAccessToken() // 启动时刷新一次 115 的访问凭证，避免过期 Token 导致同步失败

	// 设置 115 请求队列的统计保存回调函数
	v115open.SetGlobalExecutorStatSaver(func(requestTime int64, url, method string, duration int64, isThrottled bool) {
		stat := &models.RequestStat{
			RequestTime: requestTime,
			URL:         url,
			Method:      method,
			Duration:    duration,
			IsThrottled: isThrottled,
			AccountID:   0, // 可以后续扩展传入账号 ID
		}
		if err := models.CreateRequestStat(stat); err != nil {
			helpers.V115Log.Errorf("写入请求统计失败：%v", err)
		}
	})

	// 启动同步任务队列管理器
	synccron.InitNewSyncQueueManager()
	models.InitEmbyLibraryRefreshCoordinator()
	syncstrm.InitStrmGenerationWorker()
	// 初始化 WebSocket 事件中心
	wsHub := websocket.NewEventHub()
	websocket.GlobalEventHub = wsHub
	go wsHub.Run()
	synccron.InitCron()      // 初始化定时任务（包含备份定时任务）
	synccron.InitSyncCron()  // 初始化同步目录的定时任务
	synccron.InitTokenCron() // 初始化定时刷新 115 的访问凭证
	// 初始化备份服务
	models.InitBackupService()
	// 上传中的任务改为待上传
	models.UpdateUploadingToPending()
	// 下载中的任务改为待下载
	models.UpdateDownloadingToPending()
	helpers.Subscribe(helpers.BackupCronEevent, func(event helpers.Event) {
		backup.Backup("定时", "定时备份")
	})
}

// 设置路由
func setRouter(r *gin.Engine) {
	data, err := embedFiles.ReadFile("web_statics/index.html")
	if err != nil {
		log.Fatalf("读取 web_statics/index.html 失败：%v", err)
	}
	tmpl := template.Must(template.New("index.html").Parse(string(data)))
	r.SetHTMLTemplate(tmpl)

	webStaticsFS, err := fs.Sub(embedFiles, "web_statics/assets")
	if err != nil {
		log.Fatalf("创建 web_statics/assets 子文件系统失败：%v", err)
	}
	r.StaticFS("/assets", http.FS(webStaticsFS))
	r.GET("/favicon.ico", func(c *gin.Context) {
		data, _ := embedFiles.ReadFile("web_statics/favicon.ico")
		if data != nil {
			c.Data(200, "image/x-icon", data)
		} else {
			c.Status(404)
		}
	})
	r.GET("/qms-icon.png", func(c *gin.Context) {
		data, _ := embedFiles.ReadFile("web_statics/qms-icon.png")
		if data != nil {
			c.Data(200, "image/png", data)
		} else {
			c.Status(404)
		}
	})
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{})
	})
	r.POST("/emby/webhook", controllers.Webhook)
	r.POST("/api/login", controllers.LoginAction)
	r.POST("/api/strm/webhook", controllers.StrmWebhook)
	r.GET("/api/setup/status", controllers.SetupStatusAction)
	r.POST("/api/setup/admin", controllers.CreateInitialAdminAction)
	r.GET("/api/session", controllers.SessionAction)
	r.GET("/115/url/*filename", controllers.Get115UrlByPickCode)           // 查询 115 直链，按 PickCode 查询，支持 ISO，路径最后一部分为 .扩展名格式
	r.GET("/115/newurl", controllers.Get115UrlByPickCode)                  // 查询 115 直链，按 PickCode 查询
	r.GET("/baidupan/url/*filename", controllers.GetBaiduPanUrlByPickCode) // 查询百度网盘直链，按 fs_id 查询，支持 ISO，路径最后一部分为 .扩展名格式

	r.GET("/openlist/url", controllers.GetOpenListFileUrl) // 查询 OpenList 直链

	r.GET("/proxy-115", controllers.Proxy115)                      // 115 CDN 反代路由
	r.POST("/api/update-fn-access-path", controllers.UpdateFNPath) // 更新飞牛访问路径

	api := r.Group("/api")
	api.Use(controllers.JWTAuthMiddleware())
	{
		api.GET("/logs/ws", controllers.LogWebSocket)                 // WebSocket 日志查看
		api.GET("/events/ws", controllers.EventWebSocket)             // WebSocket 事件推送
		api.GET("/sync/tasks/:id/stream", controllers.SyncTaskStream) // 同步任务详情实时流
		api.GET("/logs/old", controllers.GetOldLogs)                  // 通过 HTTP 获取旧日志
		api.GET("/logs/download", controllers.DownloadLogFile)        // 下载日志文件
		api.GET("/path/is-fn-os", controllers.IsFnOS)                 // 查询是否是飞牛环境

		api.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, map[string]interface{}{
				"version":    Version,
				"build_time": parseBuildUnixTime(PublishDate),
				"date":       PublishDate,
				"isWindows":  runtime.GOOS == "windows",
				"isRelease":  helpers.IsRelease,
			})
		})
		api.POST("/database/delete-all-table", controllers.DeleteAllTabble) // 删除所有表
		api.POST("/database/repair", controllers.RepairDB)                  // 修复数据库表结构和主键序列
		api.POST("/auth/115-qrcode-open", controllers.GetLoginQrCodeOpen)   // 获取 115 开放平台登录二维码
		api.POST("/auth/115-qrcode-status", controllers.GetQrCodeStatus)    // 查询 115 二维码扫码状态
		api.GET("/115/status", controllers.Get115Status)                    // 查询 115 状态
		api.GET("/115/appids", controllers.GetV115AppIDSources)             // 查询 115 开放平台 APP ID 目录
		api.GET("/115/oauth-url", controllers.GetOAuthUrl)                  // 获取 115 OAuth 登录地址
		api.POST("115/oauth-confirm", controllers.ConfirmOAuthCode)         // 确认 115 OAuth 登录
		api.GET("/115/oauth-status", controllers.GetOAuthStatus)            // 查询 115 OAuth 授权状态
		api.GET("/115/queue/stats", controllers.GetQueueStats)              // 获取 115 开放平台请求队列统计数据
		api.POST("/115/queue/rate-limit", controllers.SetQueueRateLimit)    // 设置 115 开放平台请求队列速率限制
		api.GET("/115/stats/daily", controllers.GetRequestStatsByDay)       // 获取 115 请求统计（按天）
		api.GET("/115/stats/hourly", controllers.GetRequestStatsByHour)     // 获取 115 请求统计（按小时）
		api.POST("/115/stats/clean", controllers.CleanOldRequestStats)      // 清理旧的请求统计数据
		// 百度网盘相关路由
		api.GET("/baidupan/oauth-url", controllers.GetBaiDuPanOAuthUrl)           // 获取百度网盘 OAuth 登录地址
		api.POST("/baidupan/oauth-confirm", controllers.ConfirmBaiDuPanOAuthCode) // 确认百度网盘 OAuth 登录
		api.GET("/baidupan/status", controllers.GetBaiDuPanStatus)                // 查询百度网盘状态

		api.GET("/user/info", controllers.GetUserInfo)
		api.POST("/logout", controllers.LogoutAction)
		api.GET("/user/sessions", controllers.ListUserSessions)
		api.DELETE("/user/sessions/:session_id", controllers.RevokeUserSessionAction)
		api.POST("/user/sessions/revoke-others", controllers.RevokeOtherUserSessionsAction)
		api.GET("/user/two-factor/status", controllers.GetTwoFactorStatus)
		api.POST("/user/two-factor/setup", controllers.SetupTwoFactor)
		api.POST("/user/two-factor/enable", controllers.EnableTwoFactor)
		api.POST("/user/two-factor/disable", controllers.DisableTwoFactor)
		api.POST("/user/change", controllers.ChangePassword)

		api.POST("/setting/http-proxy", controllers.UpdateHttpProxy)                               // 更改 HTTP 代理
		api.GET("/setting/http-proxy", controllers.GetHttpProxy)                                   // 获取 HTTP 代理
		api.POST("/setting/test-http-proxy", controllers.TestHttpProxy)                            // 测试 HTTP 代理
		api.GET("/setting/log", controllers.GetLogSetting)                                         // 获取日志设置
		api.POST("/setting/log", controllers.UpdateLogSetting)                                     // 更新日志设置
		api.GET("/setting/notification/channels", controllers.GetNotificationChannels)             // 获取所有通知渠道
		api.POST("/setting/notification/channels/telegram", controllers.CreateTelegramChannel)     // 创建 Telegram 渠道
		api.GET("/setting/notification/channels/telegram/:id", controllers.GetTelegramChannel)     // 查询 Telegram 渠道
		api.PUT("/setting/notification/channels/telegram", controllers.UpdateTelegramChannel)      // 更新 Telegram 渠道
		api.POST("/setting/notification/channels/meow", controllers.CreateMeoWChannel)             // 创建 MeoW 渠道
		api.GET("/setting/notification/channels/meow/:id", controllers.GetMeoWChannel)             // 查询 MeoW 渠道
		api.PUT("/setting/notification/channels/meow", controllers.UpdateMeoWChannel)              // 更新 MeoW 渠道
		api.POST("/setting/notification/channels/bark", controllers.CreateBarkChannel)             // 创建 Bark 渠道
		api.GET("/setting/notification/channels/bark/:id", controllers.GetBarkChannel)             // 查询 Bark 渠道
		api.PUT("/setting/notification/channels/bark", controllers.UpdateBarkChannel)              // 更新 Bark 渠道
		api.POST("/setting/notification/channels/serverchan", controllers.CreateServerChanChannel) // 创建 Server 酱渠道
		api.GET("/setting/notification/channels/serverchan/:id", controllers.GetServerChanChannel) // 查询 Server 酱渠道
		api.PUT("/setting/notification/channels/serverchan", controllers.UpdateServerChanChannel)  // 更新 Server 酱渠道
		api.POST("/setting/notification/channels/webhook", controllers.CreateCustomWebhookChannel) // 创建自定义 Webhook 渠道
		api.GET("/setting/notification/channels/webhook/:id", controllers.GetCustomWebhookChannel) // 查询自定义 Webhook 渠道
		api.PUT("/setting/notification/channels/webhook", controllers.UpdateCustomWebhookChannel)  // 更新自定义 Webhook 渠道
		api.POST("/setting/notification/channels/status", controllers.UpdateChannelStatus)         // 启用/禁用渠道
		api.DELETE("/setting/notification/channels/:id", controllers.DeleteChannel)                // 删除渠道
		api.GET("/setting/notification/rules", controllers.GetNotificationRules)                   // 获取通知规则
		api.PUT("/setting/notification/rules", controllers.UpdateNotificationRule)                 // 更新通知规则
		api.POST("/setting/notification/channels/test", controllers.TestChannelConnection)         // 测试通知渠道连接
		api.GET("/setting/strm-config", controllers.GetStrmConfig)                                 // 获取 STRM 配置
		api.POST("/setting/strm-config", controllers.UpdateStrmConfig)                             // 更新 STRM 配置
		api.GET("/setting/cron", controllers.GetCronNextTime)                                      // 获取 Cron 表达式的下 5 次执行时间
		api.POST("/cron/validate", controllers.ValidateCron)                                       // 验证 Cron 表达式并返回描述
		api.POST("/setting/emby/parse", controllers.ParseEmby)                                     // 解析 Emby 媒体信息
		api.GET("/setting/emby-config", controllers.GetEmbyConfig)                                 // 获取新的 Emby 配置
		api.POST("/setting/emby-config", controllers.UpdateEmbyConfig)                             // 更新新的 Emby 配置
		api.POST("/setting/threads", controllers.UpdateThreads)                                    // 更新线程数
		api.GET("/setting/threads", controllers.GetThreads)                                        // 获取线程数

		api.POST("/emby/sync/start", controllers.StartEmbySync)     // 手动启动 Emby 同步
		api.GET("/emby/sync/status", controllers.GetEmbySyncStatus) // 获取 Emby 同步状态
		api.GET("/emby/libraries", controllers.GetEmbyLibraries)    // 获取 Emby 媒体库列表

		api.POST("/sync/start", controllers.StartSync)                   // 启动同步
		api.GET("/sync/records", controllers.GetSyncRecords)             // 同步列表
		api.GET("/sync/task", controllers.GetSyncTask)                   // 获取同步任务详情
		api.GET("/sync/path-list", controllers.GetSyncPathList)          // 获取同步路径列表
		api.POST("/sync/paths", controllers.CreateSyncPathAggregate)     // 原子创建同步路径
		api.PUT("/sync/paths/:id", controllers.UpdateSyncPathAggregate)  // 原子更新同步路径
		api.POST("/sync/path-delete", controllers.DeleteSyncPath)        // 删除同步路径
		api.POST("/sync/path/stop", controllers.StopSyncByPath)          // 停止同步路径的同步任务
		api.POST("/sync/path/start", controllers.StartSyncByPath)        // 启动同步路径的同步任务
		api.POST("/sync/path/full-start", controllers.FullStart115Sync)  // 启动 115 的全量同步任务
		api.POST("/sync/delete-records", controllers.DelSyncRecords)     // 批量删除同步记录
		api.POST("/sync/path/toggle-cron", controllers.ToggleSyncByPath) // 关闭或开启同步目录的定时同步
		api.GET("/sync/path/:id", controllers.GetSyncPathById)           // 获取同步路径详情
		api.POST("/sync/manual", controllers.ManualSync)                 // 手动同步

		api.GET("/account/list", controllers.GetAccountList)             // 获取开放平台账号列表
		api.POST("/account/add", controllers.CreateTmpAccount)           // 创建开放平台账号
		api.POST("/account/update", controllers.UpdateAccountInfo)       // 更新开放平台账号资料
		api.POST("/account/delete", controllers.DeleteAccount)           // 删除开放平台账号
		api.POST("/account/openlist", controllers.CreateOpenListAccount) // 创建 OpenList 账号

		// API Key 管理接口
		api.POST("/api-keys", controllers.CreateAPIKey)                 // 创建 API Key
		api.GET("/api-keys", controllers.ListAPIKeys)                   // 获取 API Key 列表
		api.PUT("/api-keys/:id/status", controllers.UpdateAPIKeyStatus) // 更新 API Key 状态
		api.DELETE("/api-keys/:id", controllers.DeleteAPIKey)           // 删除 API Key

		// 备份与恢复相关路由
		api.GET("/backup/list", controllers.GetBackupList)               // 获取备份列表
		api.GET("/backup/records/:id", controllers.GetBackupRecord)      // 获取备份记录详情
		api.POST("/backup/create", controllers.CreateBackup)             // 创建手动备份
		api.DELETE("/backup/records/:id", controllers.DeleteBackup)      // 删除备份记录
		api.POST("/backup/restore", controllers.RestoreFromBackup)       // 从备份恢复
		api.POST("/backup/upload-restore", controllers.UploadAndRestore) // 上传文件并恢复
		api.GET("/backup/download/:id", controllers.DownloadBackup)      // 下载备份文件
		api.GET("/backup/config", controllers.GetBackupConfig)           // 获取备份配置
		api.PUT("/backup/config", controllers.UpdateBackupConfig)        // 更新备份配置
		api.GET("/backup/status", controllers.GetBackupStatus)           // 获取备份状态

	}
}

// firstNonEmpty 返回参数中第一个非空字符串，用于按优先级取环境变量/默认值。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func initEnv() bool {
	log.Printf("当前版本号：%s，发布日期：%s\n", Version, PublishDate)
	// 将版本写入 helper
	helpers.Version = Version
	helpers.ReleaseDate = PublishDate
	// 加载环境变量配置
	helpers.LoadEnvFromFile(filepath.Join(helpers.RootDir, "config", ".env"))
	helpers.OAuthRelayEncryptionKey = firstNonEmpty(os.Getenv("OAUTH_RELAY_ENCRYPTION_KEY"), OAuthRelayEncryptionKey)
	initTimeZone()        // 设置东 8 区
	getDataAndConfigDir() // 获取数据库数据目录和配置文件目录
	log.Printf("当前工作目录：%s\n", helpers.RootDir)
	log.Printf("当前数据目录：%s\n", helpers.DataDir)
	log.Printf("当前配置文件目录：%s\n", helpers.ConfigDir)
	if err := helpers.InitEncryptionKey(); err != nil {
		log.Printf("初始化本机加密密钥失败：%v\n", err)
		return false
	}
	ipv4, _ := helpers.GetLocalIP()
	log.Printf("本机 IPv4 地址：%s\n", ipv4)
	// 检查配置文件是否存在
	configPath := helpers.ExistingConfigFilePath()
	helpers.IsFirstRun = !helpers.HasConfigFile()
	// 如果不存在，启动一个简易 Web 服务来配置数据库连接信息
	if helpers.IsFirstRun {
		// 检查是否有旧的数据库配置和记录，有的话生成配置文件，跳过配置流程
		oldPostgresDataDir := filepath.Join(helpers.ConfigDir, "postgres")
		if helpers.PathExists(oldPostgresDataDir) {
			log.Printf("发现旧的数据库数据目录：%s", oldPostgresDataDir)
			// 生成新的配置文件
			if err := helpers.MakeOldConfig(); err != nil {
				log.Printf("生成新的配置文件失败：%v", err)
				return false
			}
			configPath = helpers.ConfigFilePath()
			log.Printf("已生成配置文件：%s", configPath)
			helpers.IsFirstRun = false
		} else {
			log.Printf("配置文件不存在，启动简单配置服务：%s", helpers.ConfigFilePath())
			StartConfigWebServer()
			return false
		}
	}
	configPath = helpers.ExistingConfigFilePath()
	log.Printf("配置文件存在，加载配置文件：%s", configPath)
	// 如果存在，则加载配置文件，进行其他的初始化工作
	err := helpers.InitConfig()
	if err != nil {
		log.Printf("初始化配置文件失败：%v", err)
		return false
	}
	initLogger()
	// 创建 App
	newApp()
	helpers.AppLogger.Infof("当前版本号：%s，发布日期：%s", Version, PublishDate)

	// 检查是否需要自动恢复
	if migrate.ShouldRestore() {
		helpers.AppLogger.Info("检测到迁移备份文件存在且使用外部 PostgreSQL，开始自动恢复…")
		// 先启动外部数据库连接
		if err := QMSApp.StartDatabase(false); err != nil {
			log.Printf("数据库启动失败：%v", err)
			return false
		}
		// 执行恢复
		backupPath := migrate.GetMigrateBackupPath()
		if err := performMigrateRestore(backupPath); err != nil {
			helpers.AppLogger.Errorf("恢复数据失败：%v", err)
			return false
		}
		// 恢复成功，删除备份文件
		os.Remove(backupPath)
		helpers.AppLogger.Info("数据恢复完成，已删除迁移备份文件")
	} else {
		// 检查是否需要启动迁移服务
		needMigrate := migrate.ShouldMigrate()
		// needMigrate := false
		if err := QMSApp.StartDatabase(needMigrate); err != nil {
			helpers.AppLogger.Errorf("数据库启动失败：%v", err)
			return false
		}
		// 如果启动了迁移服务，则直接返回 false（迁移服务会自己处理退出）
		if needMigrate {
			return false
		}
	}

	if err := configureInitialAdminSetup(); err != nil {
		helpers.AppLogger.Errorf("初始化管理员创建状态失败：%v", err)
		return false
	}

	db.InitCache() // 初始化内存缓存
	initOthers()
	return true
}

func parseParams() {
	// 定义 GUID 参数
	flag.StringVar(&helpers.Guid, "guid", "", "GUID 参数")
	flag.BoolVar(&helpers.IsFnOS, "fnos", false, "是否是飞牛环境")
	// 解析命令行参数
	flag.Parse()
	// 使用参数
	if helpers.IsFnOS {
		log.Printf("当前环境为飞牛环境\n")
	}
	if helpers.Guid == "" || helpers.Guid == "0" {
		// 检查是否有 GUID 环境变量，有的话直接使用
		guidEnv := os.Getenv("GUID")
		if guidEnv != "" {
			log.Printf("使用环境变量 GUID：%s 执行操作\n", guidEnv)
			helpers.Guid = guidEnv
		} else {
			log.Printf("使用 root 执行操作\n")
			helpers.Guid = "0"
		}
	}
}

// @title QMediaSync API
// @version 1.0
// @description 媒体同步和刮削系统 API
// @host localhost:8115
// @BasePath /
// @securityDefinitions.apikey JwtAuth
// @in header
// @name Authorization
// @securityDefinitions.apikey ApiKeyAuth
// @in query
// @name api_key
func main() {
	parseParams()
	getRootDir()
	if !initEnv() {
		panic("初始化环境失败")
	}
	if runtime.GOOS == "windows" {
		if helpers.IsRelease {
			go QMSApp.Start()
			helpers.StartApp(func() {
				QMSApp.Stop()
			})
		} else {
			QMSApp.Start()
		}
	} else {
		QMSApp.Start()
	}
}

func isInRestrictedDirectory() (bool, string) {
	if runtime.GOOS != "windows" {
		return false, ""
	}

	exePath, err := os.Executable()
	if err != nil {
		return false, ""
	}
	exeDir := filepath.Dir(exePath)

	driveLetter := strings.ToUpper(string(exeDir[0]))
	log.Printf("应用程序路径：%s，盘符：%s", exePath, driveLetter)
	if driveLetter == "C" {
		return true, "应用程序位于 C 盘，建议将应用程序移动到其他盘符（如 D 盘、E 盘等）以避免权限问题"
	}

	restrictedPaths := []string{
		"Program Files",
		"Program Files (x86)",
		"ProgramData",
		"Windows",
	}

	for _, restrictedPath := range restrictedPaths {
		log.Printf("检查目录：%s，是否包含受限路径：%s", exeDir, restrictedPath)
		if strings.Contains(exeDir, restrictedPath) {
			return true, fmt.Sprintf("应用程序位于受限目录 %s 中，建议将应用程序移动到普通用户目录或其他非系统目录", restrictedPath)
		}
	}

	return false, ""
}

func performMigrateRestore(backupPath string) error {
	helpers.AppLogger.Infof("开始从迁移备份恢复：%s", backupPath)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在：%s", backupPath)
	}

	if err := backup.Restore(backupPath); err != nil {
		return fmt.Errorf("恢复失败：%v", err)
	}

	helpers.AppLogger.Info("迁移恢复完成")
	return nil
}

func StartConfigWebServer() {
	if helpers.IsRelease {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	// 使用 embed.FS 加载模板
	data, err := embedFiles.ReadFile("assets/db_config.html")
	if err != nil {
		log.Fatal(err)
	}
	// 创建内存中的 HTML 文件
	tmpl := template.Must(template.New("db_config.html").Parse(string(data)))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", func(c *gin.Context) {
		isRestricted, warningMsg := isInRestrictedDirectory()
		c.HTML(200, "db_config.html", gin.H{
			"title":        "数据库配置",
			"isRestricted": isRestricted,
			"warningMsg":   warningMsg,
			"isWindows":    runtime.GOOS == "windows",
		})
	})

	r.POST("/api/config/test-db", func(c *gin.Context) {
		var req struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
			Database string `json:"database"`
			SSL      bool   `json:"ssl"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": err.Error()})
			return
		}

		sslMode := "disable"
		if req.SSL {
			sslMode = "require"
		}
		connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
			req.Host, req.Port, req.User, req.Password, sslMode)
		sqlDB, err := sql.Open("postgres", connStr)
		if err != nil {
			c.JSON(200, gin.H{"success": false, "error": "连接失败：" + err.Error()})
			return
		}
		defer sqlDB.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(200, gin.H{"success": false, "error": "连接失败：" + err.Error()})
			return
		}

		var dbExists bool
		err = sqlDB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", req.Database).Scan(&dbExists)
		if err != nil {
			c.JSON(200, gin.H{"success": false, "error": "检查数据库失败：" + err.Error()})
			return
		}

		if !dbExists {
			c.JSON(200, gin.H{"success": true, "message": "数据库连接成功", "dbExists": false})
			return
		}

		connStrDb := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			req.Host, req.Port, req.User, req.Password, req.Database, sslMode)
		sqlDBDb, err := sql.Open("postgres", connStrDb)
		if err != nil {
			c.JSON(200, gin.H{"success": true, "message": "数据库连接成功", "dbExists": true, "hasOtherTables": false})
			return
		}
		defer sqlDBDb.Close()

		var tableCount int
		err = sqlDBDb.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name NOT LIKE 'gorr_%'").Scan(&tableCount)
		if err != nil {
			c.JSON(200, gin.H{"success": true, "message": "数据库连接成功", "dbExists": true, "hasOtherTables": false})
			return
		}

		c.JSON(200, gin.H{"success": true, "message": "数据库连接成功", "dbExists": true, "hasOtherTables": tableCount > 0})
	})

	r.POST("/api/config/save", func(c *gin.Context) {
		var req struct {
			Engine       string `json:"engine"`
			PostgresType string `json:"postgresType"`
			Host         string `json:"host"`
			Port         int    `json:"port"`
			User         string `json:"user"`
			Password     string `json:"password"`
			Database     string `json:"database"`
			SSL          bool   `json:"ssl"`
			DropDatabase bool   `json:"dropDatabase"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		yamlConfig := helpers.MakeDefaultConfig()
		if req.Engine == string(helpers.DbEnginePostgres) {
			yamlConfig.Db.PostgresType = helpers.PostgresType(req.PostgresType)
			if req.PostgresType == string(helpers.PostgresTypeExternal) {
				quotedDatabase, err := database.QuotePostgresIdentifier(req.Database)
				if err != nil {
					c.JSON(200, gin.H{"error": "数据库名不合法：" + err.Error()})
					return
				}
				if req.DropDatabase {
					sslMode := "disable"
					if req.SSL {
						sslMode = "require"
					}
					connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
						req.Host, req.Port, req.User, req.Password, sslMode)
					sqlDB, err := sql.Open("postgres", connStr)
					if err == nil {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						sqlDB.ExecContext(ctx, "DROP DATABASE IF EXISTS "+quotedDatabase)
						sqlDB.Close()
					}
				}
				yamlConfig.Db.PostgresConfig = helpers.PostgresConfig{
					Host:         req.Host,
					Port:         req.Port,
					User:         req.User,
					Password:     req.Password,
					Database:     req.Database,
					SSL:          req.SSL,
					MaxOpenConns: 25,
					MaxIdleConns: 25,
				}
			} else {
				yamlConfig.Db.PostgresConfig = helpers.PostgresConfig{
					Host:         "localhost",
					Port:         5432,
					User:         "qms",
					Password:     "qms123456",
					Database:     "qms",
					MaxOpenConns: 25,
					MaxIdleConns: 25,
				}
			}
		} else {
			yamlConfig.Db.Engine = helpers.DbEngineSqlite
		}

		if err := helpers.SaveConfig(yamlConfig); err != nil {
			c.JSON(500, gin.H{"error": "保存配置失败：" + err.Error()})
			return
		}

		c.JSON(200, gin.H{"success": true, "message": "配置已保存，配置服务已退出。重启后请查看启动日志中的初始化码，并在 Web 页面创建首个管理员。"})
		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
	})

	fmt.Printf("配置服务已启动，请在浏览器中访问：http://ip:12333\n")
	go func() {
		// 第一次启动建议多等一会儿，因为数据库初始化需要时间
		time.Sleep(2 * time.Second)
		helpers.OpenBrowser("http://127.0.0.1:12333")
	}()
	if err := r.Run(":12333"); err != nil {
		log.Fatalf("启动配置服务失败：%v", err)
	}
}
