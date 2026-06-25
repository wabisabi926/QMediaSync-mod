package helpers

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

var Version = "0.0.1"
var ReleaseDate = "2025-11-07"

type DbEngine string

const (
	DbEngineSqlite   DbEngine = "sqlite"
	DbEnginePostgres DbEngine = "postgres"
	DbEngineUnset    DbEngine = ""
)

type PostgresType string

const (
	PostgresTypeEmbedded PostgresType = "embedded"
	PostgresTypeExternal PostgresType = "external"
)

type ConfigLog struct {
	File       string `yaml:"file"`
	V115       string `yaml:"v115"`
	OpenList   string `yaml:"openList"`
	TMDB       string `yaml:"tmdb"`
	BaiduPan   string `yaml:"baiduPan"`
	Web        string `yaml:"web"`
	SyncLogDir string `yaml:"syncLogDir"` // 同步任务的日志目录，每个同步任务会生成一个日志文件，文件名为任务 ID
}

type PostgresConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	SSL          bool   `yaml:"ssl"`
	MaxOpenConns int    `yaml:"maxOpenConns"`
	MaxIdleConns int    `yaml:"maxIdleConns"`
}

type ConfigDb struct {
	Engine         DbEngine       `yaml:"engine"`         // 使用的数据库引擎，可选值：sqlite, postgres
	SqliteFile     string         `yaml:"sqliteFile"`     // SQLite 数据库文件路径
	PostgresType   PostgresType   `yaml:"postgresType"`   // PostgreSQL 数据库类型，可选值：embedded, external
	PostgresConfig PostgresConfig `yaml:"postgresConfig"` // PostgreSQL 数据库配置
}

type ConfigStrm struct {
	VideoExt     []string `yaml:"videoExt"`
	MinVideoSize int64    `yaml:"minVideoSize"` // 最小视频大小，单位字节
	MetaExt      []string `yaml:"metaExt"`
	Cron         string   `yaml:"cron"` // 定时任务表达式
}

type Config struct {
	Log           ConfigLog  `yaml:"log"`
	Db            ConfigDb   `yaml:"db"`
	CacheSize     int        `yaml:"cacheSize"` // 数据库缓存大小，单位字节
	JwtSecret     string     `yaml:"jwtSecret"`
	HttpHost      string     `yaml:"httpHost"`  // HTTP 主机地址
	HttpsHost     string     `yaml:"httpsHost"` // HTTPS 主机地址
	Strm          ConfigStrm `yaml:"strm"`
	AuthServer    string     `yaml:"authServer"`
	NewAuthServer string     `yaml:"newAuthServer"`
	BaiDuPanAppId string     `yaml:"baiDuPanAppId"`
	AdminUsername string     `yaml:"adminUsername"`
	AdminPassword string     `yaml:"adminPassword"`
}

var GlobalConfig Config
var RootDir string
var ConfigDir string

var DataDir string
var SharePathes string
var AccessiblePathes string
var IsFnOS bool
var IsRelease bool
var Guid string
var FANART_API_KEY = "" // 生效值：Fanart 客户端读取，由 ScrapeSettings.ApplyKeyOverrides 按“UI > 默认”刷新
var HTTP_PROXY = ""     // 生效的通用刮削代理：由 ScrapeSettings.ApplyKeyOverrides 按代理开关刷新；Fanart 等直读 helpers 的客户端使用（空=直连）
var DEFAULT_TMDB_ACCESS_TOKEN = ""
var DEFAULT_TMDB_API_KEY = ""
var DEFAULT_SC_API_KEY = ""
var OAuthRelayEncryptionKey = "" // 生效值：网盘 OAuth 中转共享 AES 密钥，环境变量优先于 ldflags 注入
var DEFAULT_FANART_API_KEY = ""  // 默认基线：环境变量 > ldflags

const (
	ConfigFileName       = "config.yaml"
	legacyConfigFileName = "config.yml"
	DefaultJWTSecret     = "Q115-STRM-JWT-TOKEN-250706"
)

func ConfigFilePath() string {
	return filepath.Join(ConfigDir, ConfigFileName)
}

func ExistingConfigFilePath() string {
	configPath := ConfigFilePath()
	if PathExists(configPath) {
		return configPath
	}
	legacyConfigPath := filepath.Join(ConfigDir, legacyConfigFileName)
	if PathExists(legacyConfigPath) {
		return legacyConfigPath
	}
	return configPath
}

func HasConfigFile() bool {
	return PathExists(ConfigFilePath()) || PathExists(filepath.Join(ConfigDir, legacyConfigFileName))
}

func InitConfig() error {
	configPath := ExistingConfigFilePath()
	// 从配置文件加载
	if err := loadYaml(configPath, &GlobalConfig); err != nil {
		return err
	}
	// 给 STRM 填充默认值
	if len(GlobalConfig.Strm.VideoExt) == 0 {
		GlobalConfig.Strm.VideoExt = []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".3gp", ".ts"}
	}
	if len(GlobalConfig.Strm.MetaExt) == 0 {
		GlobalConfig.Strm.MetaExt = []string{".jpg", ".jpeg", ".png", ".webp", ".nfo", ".srt", ".ass", ".svg", ".sup", ".lrc"}
	}
	// if GlobalConfig.Strm.MinVideoSize == 0 {
	GlobalConfig.Strm.MinVideoSize = 100 // 100 MB
	// }
	// if GlobalConfig.Strm.Cron == "" {
	GlobalConfig.Strm.Cron = "30 * * * *" // 每小时 30 分执行
	// }
	if GlobalConfig.AuthServer == "" {
		GlobalConfig.AuthServer = "https://api.mqfamily.top"
	}
	if GlobalConfig.NewAuthServer == "" {
		GlobalConfig.NewAuthServer = "https://oauth.qmediasync.cn"
	}
	if err := EnsureJWTSecret(); err != nil {
		return err
	}
	return nil
}

func generateRandomJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成 JWT 密钥失败：%w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func EnsureJWTSecret() error {
	if GlobalConfig.JwtSecret != "" && GlobalConfig.JwtSecret != DefaultJWTSecret {
		return nil
	}
	secret, err := generateRandomJWTSecret()
	if err != nil {
		return err
	}
	GlobalConfig.JwtSecret = secret
	if err := SaveConfig(&GlobalConfig); err != nil {
		return fmt.Errorf("保存自动生成的 JWT 密钥失败：%w", err)
	}
	if AppLogger != nil {
		AppLogger.Warnf("检测到 jwtSecret 为空或为公开默认值，已自动生成强随机密钥并写回配置")
	}
	return nil
}

func LoadEnvFromFile(envPath string) error {
	f, err := os.Open(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("环境变量配置文件不存在：%s\n", envPath)
			return nil
		}
		return err
	}
	defer f.Close()
	fmt.Printf("已加载环境变量配置文件：%s\n", envPath)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" {
			continue
		}
		value := line[idx+1:]
		os.Setenv(key, value)
		// fmt.Printf("Loaded env: %s=%s\n", key, value)
	}

	return scanner.Err()
}

func loadYaml(configPath string, cfg interface{}) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败：%w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("解析配置文件失败：%w", err)
	}

	return nil
}

func MakeOldConfig() error {
	yamlConfig := MakeDefaultConfig()
	host := os.Getenv("DB_HOST")
	if host != "" {
		yamlConfig.Db.PostgresConfig.Host = host
	}
	port := os.Getenv("DB_PORT")
	if port != "" {
		yamlConfig.Db.PostgresConfig.Port = StringToInt(port)
	}
	user := os.Getenv("DB_USER")
	if user != "" {
		yamlConfig.Db.PostgresConfig.User = user
	}
	password := os.Getenv("DB_PASSWORD")
	if password != "" {
		yamlConfig.Db.PostgresConfig.Password = password
	}
	database := os.Getenv("DB_NAME")
	if database != "" {
		yamlConfig.Db.PostgresConfig.Database = database
	}

	return SaveConfig(yamlConfig)
}

func SaveConfig(config *Config) error {
	configPath := ConfigFilePath()
	configData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return err
	}
	return nil
}

func MakeDefaultConfig() *Config {
	return &Config{
		Log: ConfigLog{
			File:       "logs/app.log",
			V115:       "logs/115.log",
			OpenList:   "logs/openList.log",
			TMDB:       "logs/tmdb.log",
			BaiduPan:   "logs/baidupan.log",
			Web:        "logs/web.log",
			SyncLogDir: "logs/sync",
		},
		Db: ConfigDb{
			Engine:       DbEnginePostgres,
			SqliteFile:   "qmediasync.db",
			PostgresType: PostgresTypeEmbedded,
			PostgresConfig: PostgresConfig{
				Host:         "localhost",
				Port:         5432,
				User:         "qms",
				Password:     "qms123456",
				Database:     "qms",
				MaxOpenConns: 25,
				MaxIdleConns: 25,
			},
		},
		CacheSize:     20971520,
		JwtSecret:     DefaultJWTSecret,
		HttpHost:      ":12333",
		HttpsHost:     ":12332",
		AuthServer:    "https://api.mqfamily.top",
		NewAuthServer: "https://oauth.qmediasync.cn",
		BaiDuPanAppId: "QMediaSync",
		AdminUsername: "admin",
		AdminPassword: "admin123",
		Strm: ConfigStrm{
			VideoExt:     []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".3gp", ".ts"},
			MetaExt:      []string{".jpg", ".jpeg", ".png", ".webp", ".nfo", ".srt", ".ass", ".svg", ".sup", ".lrc"},
			MinVideoSize: 100,          // 100 MB
			Cron:         "30 * * * *", // 每小时 30 分执行
		},
	}
}
