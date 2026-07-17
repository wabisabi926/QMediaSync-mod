package syncconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"qmediasync/internal/db"
	"qmediasync/internal/models"
	"qmediasync/internal/synccron"

	"gorm.io/gorm"
)

const (
	ErrorCodeInvalidRequest       = "INVALID_REQUEST"
	ErrorCodeAccountSourceInvalid = "ACCOUNT_SOURCE_INVALID"
	ErrorCodeSyncPathNotFound     = "SYNC_PATH_NOT_FOUND"
	ErrorCodeRuleOwnership        = "DIRECTORY_UPLOAD_RULE_OWNERSHIP"
	ErrorCodeRuleBoundary         = "DIRECTORY_UPLOAD_RULE_BOUNDARY"
	ErrorCodeRuleConflict         = "DIRECTORY_UPLOAD_RULE_CONFLICT"
	ErrorCodeIdempotencyConflict  = "IDEMPOTENCY_CONFLICT"
	ErrorCodeDatabaseSave         = "DATABASE_SAVE_FAILED"
)

// FieldError 描述可定位到基础字段或规则 client_id 的校验错误。
type FieldError struct {
	ClientID string `json:"client_id,omitempty"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

// SaveError 是同步目录聚合保存业务错误。
type SaveError struct {
	Code        string
	Message     string
	FieldErrors []FieldError
	Cause       error
}

func (err *SaveError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

func (err *SaveError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// SyncPathInput 描述同步目录基础配置。
type SyncPathInput struct {
	SourceType   models.SourceType
	AccountID    uint
	BaseCid      string
	LocalPath    string
	RemotePath   string
	EnableCron   bool
	CustomConfig bool
	Setting      models.SettingStrm
}

// SaveSyncPathCommand 保存同步目录聚合。
type SaveSyncPathCommand struct {
	ID             uint
	IdempotencyKey string
	SyncPath       SyncPathInput
}

// SaveSyncPathResult 是同步目录聚合保存结果。
type SaveSyncPathResult struct {
	SyncPath *models.SyncPath `json:"sync_path"`
	Warnings []string         `json:"warnings"`
}

// ServiceOptions 配置同步目录应用服务及事务后副作用。
type ServiceOptions struct {
	DB                             *gorm.DB
	CreateLocalDirectory           func(string) error
	ReloadSyncCron                 func()
	ReloadDirectoryUpload          func()
	ReloadSyncCronWithError        func() error
	ReloadDirectoryUploadWithError func() error
	RunTransaction                 func(context.Context, *gorm.DB, func(*gorm.DB) error) error
}

// Service 保存同步目录配置聚合。
type Service struct {
	db                    *gorm.DB
	createLocalDirectory  func(string) error
	reloadSyncCron        func() error
	reloadDirectoryUpload func() error
	runTransaction        func(context.Context, *gorm.DB, func(*gorm.DB) error) error
}

// NewService 创建同步目录配置应用服务。
func NewService(options ServiceOptions) *Service {
	handle := options.DB
	if handle == nil {
		handle = db.Db
	}
	createLocalDirectory := options.CreateLocalDirectory
	if createLocalDirectory == nil {
		createLocalDirectory = func(string) error { return nil }
	}
	reloadSyncCron := options.ReloadSyncCronWithError
	if reloadSyncCron == nil && options.ReloadSyncCron != nil {
		reloadSyncCron = func() error {
			options.ReloadSyncCron()
			return nil
		}
	}
	if reloadSyncCron == nil {
		reloadSyncCron = func() error { return nil }
	}
	reloadDirectoryUpload := options.ReloadDirectoryUploadWithError
	if reloadDirectoryUpload == nil && options.ReloadDirectoryUpload != nil {
		reloadDirectoryUpload = func() error {
			options.ReloadDirectoryUpload()
			return nil
		}
	}
	if reloadDirectoryUpload == nil {
		reloadDirectoryUpload = func() error { return nil }
	}
	runTransaction := options.RunTransaction
	if runTransaction == nil {
		runTransaction = func(ctx context.Context, handle *gorm.DB, fn func(*gorm.DB) error) error {
			return handle.WithContext(ctx).Transaction(fn)
		}
	}
	return &Service{
		db:                    handle,
		createLocalDirectory:  createLocalDirectory,
		reloadSyncCron:        reloadSyncCron,
		reloadDirectoryUpload: reloadDirectoryUpload,
		runTransaction:        runTransaction,
	}
}

// NewDefaultService 创建使用生产副作用的同步目录配置应用服务。
func NewDefaultService() *Service {
	return NewService(ServiceOptions{
		DB:                             db.Db,
		CreateLocalDirectory:           func(path string) error { return os.MkdirAll(path, 0o777) },
		ReloadSyncCronWithError:        synccron.InitSyncCronWithError,
		ReloadDirectoryUploadWithError: func() error { return nil },
	})
}

// Save 原子保存 SyncPath，并在 commit 后执行运行态副作用。
func (service *Service) Save(ctx context.Context, command SaveSyncPathCommand) (*SaveSyncPathResult, error) {
	if service == nil || service.db == nil {
		return nil, newSaveError(ErrorCodeDatabaseSave, "数据库连接为空", nil)
	}
	if err := validateCommand(command); err != nil {
		return nil, err
	}
	idempotencyKey := strings.TrimSpace(command.IdempotencyKey)
	idempotencyKeyHash := hashIdempotencyKey(idempotencyKey)
	var result *SaveSyncPathResult
	replayed := false
	err := service.runTransaction(ctx, service.db, func(tx *gorm.DB) error {
		if command.ID == 0 && idempotencyKey != "" {
			existing, found, err := findIdempotencyRecordWithDB(tx, idempotencyKeyHash)
			if err != nil {
				return err
			}
			if found {
				if existing.Status != "completed" || existing.SyncPathId == 0 {
					return newSaveError(ErrorCodeIdempotencyConflict, "相同幂等键的创建请求正在处理", nil)
				}
				result, err = loadAggregateWithDB(tx, existing.SyncPathId)
				replayed = err == nil
				return err
			}
			if err := tx.Create(&models.SyncPathIdempotencyRecord{KeyHash: idempotencyKeyHash, Status: "pending"}).Error; err != nil {
				return newSaveError(ErrorCodeIdempotencyConflict, "幂等键冲突", err)
			}
		}
		if replayed {
			return nil
		}
		writeInput := models.SyncPathWriteInput{
			SourceType:   command.SyncPath.SourceType,
			AccountID:    command.SyncPath.AccountID,
			BaseCid:      command.SyncPath.BaseCid,
			LocalPath:    command.SyncPath.LocalPath,
			RemotePath:   command.SyncPath.RemotePath,
			EnableCron:   command.SyncPath.EnableCron,
			CustomConfig: command.SyncPath.CustomConfig,
			Setting:      command.SyncPath.Setting,
		}
		var syncPath *models.SyncPath
		var err error
		if command.ID != 0 {
			syncPath = &models.SyncPath{}
			if err = tx.First(syncPath, command.ID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				return newSaveError(ErrorCodeSyncPathNotFound, "同步目录不存在", err)
			}
			if err != nil {
				return err
			}
			if err := validateImmutableSyncPathFields(syncPath, command.SyncPath); err != nil {
				return err
			}
		}
		if err := validateAccountWithDB(tx, command.SyncPath); err != nil {
			return err
		}
		if command.ID == 0 {
			syncPath, err = models.CreateSyncPathWithDB(tx, writeInput)
		} else {
			err = models.UpdateSyncPathWithDB(tx, syncPath, writeInput)
		}
		if err != nil {
			return newSaveError(ErrorCodeDatabaseSave, "保存同步目录失败", err)
		}
		result = &SaveSyncPathResult{
			SyncPath: syncPath,
			Warnings: []string{},
		}
		if command.ID == 0 && idempotencyKey != "" {
			if err := tx.Model(&models.SyncPathIdempotencyRecord{}).
				Where("key_hash = ?", idempotencyKeyHash).
				Updates(map[string]any{"sync_path_id": syncPath.ID, "status": "completed"}).Error; err != nil {
				return newSaveError(ErrorCodeDatabaseSave, "保存幂等结果失败", err)
			}
		}
		return nil
	})
	if err != nil {
		var saveErr *SaveError
		if errors.As(err, &saveErr) {
			return nil, saveErr
		}
		return nil, newSaveError(ErrorCodeDatabaseSave, "保存同步目录失败", err)
	}
	if result == nil || replayed {
		return result, nil
	}
	if err := service.createLocalDirectory(result.SyncPath.GetFullLocalPath()); err != nil {
		result.Warnings = append(result.Warnings, "同步目录已保存，但创建本地目录失败")
	}
	if err := service.reloadSyncCron(); err != nil {
		result.Warnings = append(result.Warnings, "同步目录已保存，但重载定时同步任务失败")
	}
	return result, nil
}

func validateCommand(command SaveSyncPathCommand) error {
	if command.SyncPath.SourceType == "" || strings.TrimSpace(command.SyncPath.BaseCid) == "" ||
		strings.TrimSpace(command.SyncPath.LocalPath) == "" || strings.TrimSpace(command.SyncPath.RemotePath) == "" {
		return newSaveError(ErrorCodeInvalidRequest, "请求格式错误", nil)
	}
	return nil
}

func validateImmutableSyncPathFields(existing *models.SyncPath, input SyncPathInput) error {
	if existing.SourceType != input.SourceType {
		return &SaveError{
			Code:        ErrorCodeInvalidRequest,
			Message:     "同步来源不能修改",
			FieldErrors: []FieldError{{Field: "source_type", Message: "同步来源不能修改"}},
		}
	}
	if existing.AccountId != input.AccountID {
		return &SaveError{
			Code:        ErrorCodeInvalidRequest,
			Message:     "同步账号不能修改",
			FieldErrors: []FieldError{{Field: "account_id", Message: "同步账号不能修改"}},
		}
	}
	return nil
}

func hashIdempotencyKey(key string) string {
	if key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func validateAccountWithDB(tx *gorm.DB, input SyncPathInput) error {
	if input.SourceType == models.SourceTypeLocal {
		return nil
	}
	var account models.Account
	if err := tx.First(&account, input.AccountID).Error; err != nil {
		return &SaveError{
			Code:        ErrorCodeAccountSourceInvalid,
			Message:     "账号不存在",
			FieldErrors: []FieldError{{Field: "account_id", Message: "账号不存在"}},
			Cause:       err,
		}
	}
	if account.SourceType != input.SourceType {
		return &SaveError{
			Code:        ErrorCodeAccountSourceInvalid,
			Message:     "账号类型与同步源类型不一致",
			FieldErrors: []FieldError{{Field: "account_id", Message: "账号类型与同步源类型不一致"}},
		}
	}
	return nil
}

func findIdempotencyRecordWithDB(tx *gorm.DB, keyHash string) (*models.SyncPathIdempotencyRecord, bool, error) {
	var record models.SyncPathIdempotencyRecord
	err := tx.Where("key_hash = ?", strings.TrimSpace(keyHash)).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	return &record, err == nil, err
}

func loadAggregateWithDB(tx *gorm.DB, syncPathID uint) (*SaveSyncPathResult, error) {
	var syncPath models.SyncPath
	if err := tx.First(&syncPath, syncPathID).Error; err != nil {
		return nil, err
	}
	return &SaveSyncPathResult{
		SyncPath: &syncPath,
		Warnings: []string{},
	}, nil
}

func newSaveError(code string, message string, cause error) *SaveError {
	return &SaveError{Code: code, Message: message, Cause: cause}
}

func (result SaveSyncPathResult) String() string {
	if result.SyncPath == nil {
		return ""
	}
	return fmt.Sprintf("sync_path:%d", result.SyncPath.ID)
}
