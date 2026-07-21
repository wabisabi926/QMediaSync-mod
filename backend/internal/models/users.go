package models

import (
	"errors"
	"fmt"
	"strings"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	SingletonKey           uint8  `gorm:"uniqueIndex;not null;default:1" json:"-"`
	Username               string `gorm:"unique;not null" json:"username"`
	Password               string `gorm:"not null"`
	TwoFactorEnabled       bool   `gorm:"default:false" json:"two_factor_enabled"` // 是否启用两步验证
	TwoFactorSecret        string `gorm:"type:text" json:"-"`                      // 加密后的 TOTP 密钥
	TwoFactorPendingSecret string `gorm:"type:text" json:"-"`                      // 启用确认前的加密 TOTP 密钥
}

// 表名
func (User) TableName() string {
	return "users"
}

func (user *User) BeforeCreate(tx *gorm.DB) error {
	user.SingletonKey = 1
	return nil
}

// IsTwoFactorEnabled 判断用户是否启用两步验证
func (user *User) IsTwoFactorEnabled() bool {
	return user != nil && user.TwoFactorEnabled && user.TwoFactorSecret != ""
}

// EnableTwoFactor 启用两步验证
func (user *User) EnableTwoFactor(encryptedSecret string) {
	user.TwoFactorEnabled = true
	user.TwoFactorSecret = encryptedSecret
	user.TwoFactorPendingSecret = ""
}

// DisableTwoFactor 关闭两步验证
func (user *User) DisableTwoFactor() {
	user.TwoFactorEnabled = false
	user.TwoFactorSecret = ""
	user.TwoFactorPendingSecret = ""
}

// SaveUserTwoFactor 保存用户的两步验证状态。
func SaveUserTwoFactor(user *User) error {
	if err := db.Db.Model(&User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"two_factor_enabled":        user.TwoFactorEnabled,
		"two_factor_secret":         user.TwoFactorSecret,
		"two_factor_pending_secret": user.TwoFactorPendingSecret,
	}).Error; err != nil {
		helpers.AppLogger.Errorf("保存用户失败：%v", err)
		return err
	}
	return nil
}

func HashUserPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), UserPasswordBcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

var (
	ErrNewPasswordMatchesCurrent = errors.New("新密码不能与当前密码相同")
	ErrUserCredentialsUnchanged  = errors.New("用户名和密码未发生变化")
)

// ChangeUsernameAndPasswordAndRevokeSessions 修改用户凭据并撤销全部浏览器会话。
func (user *User) ChangeUsernameAndPasswordAndRevokeSessions(username, newPassword string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == user.Username && newPassword == "" {
		return false, ErrUserCredentialsUnchanged
	}
	if err := ValidateUserUsername(username); err != nil {
		return false, err
	}
	updates := map[string]any{
		"username": username,
	}
	if newPassword != "" {
		if err := ValidateUserPassword(newPassword); err != nil {
			return false, err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newPassword)); err == nil {
			return false, ErrNewPasswordMatchesCurrent
		} else if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, fmt.Errorf("比较新旧密码: %w", err)
		}
		hash, err := HashUserPassword(newPassword)
		if err != nil {
			helpers.AppLogger.Warnf("生成用户新密码失败：%v", err)
			return false, err
		}
		updates["password"] = hash
	}
	if err := db.Db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&User{}).Where("id = ?", user.ID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := revokeAllUserSessions(tx, user.ID, "credential_changed"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false, err
	}
	user.Username = username
	if password, ok := updates["password"].(string); ok {
		user.Password = password
	}
	return true, nil
}

// 根据用户 ID 查询用户
func GetUserById(userId uint) (*User, error) {
	user := &User{}
	result := db.Db.First(user, userId)
	if result.Error != nil {
		helpers.AppLogger.Errorf("查询用户失败：%v", result.Error)
		// 返回查询错误，由调用方区分记录不存在和内部故障。
		return user, result.Error
	}
	return user, nil
}

// 根据用户名查询用户
// 如果没有找到，则返回一个空的 User 对象和 nil 错误
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	result := db.Db.First(user, "username = ?", username)
	if result.Error != nil {
		helpers.AppLogger.Errorf("根据用户名查询用户失败：%v", result.Error)
		// 如果没有，则返回 nil
		return user, result.Error
	}
	return user, nil
}

// 检查用户名和密码是否匹配
// 如果匹配，则返回用户对象，否则返回错误
func CheckLogin(username, password string) (*User, error) {
	user, err := GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if user.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	// helpers.AppLogger.Infof("检查用户 %s 的密码：%s => %s", username, password, user.Password)
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}
	user.ensurePasswordHashCost(password)
	return user, nil
}

func (user *User) ensurePasswordHashCost(password string) {
	cost, err := bcrypt.Cost([]byte(user.Password))
	if err != nil || cost >= UserPasswordBcryptCost {
		return
	}
	nextHash, err := HashUserPassword(password)
	if err != nil {
		if helpers.AppLogger != nil {
			helpers.AppLogger.Warnf("升级用户密码哈希成本失败：%v", err)
		}
		return
	}
	result := db.Db.Model(&User{}).
		Where("id = ? AND password = ?", user.ID, user.Password).
		Update("password", nextHash)
	if result.Error != nil {
		if helpers.AppLogger != nil {
			helpers.AppLogger.Warnf("保存升级后的用户密码哈希失败：%v", result.Error)
		}
		return
	}
	if result.RowsAffected == 0 {
		return
	}
	user.Password = nextHash
}

// GetPwd 给密码加密
func GetPwd(pwd string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), UserPasswordBcryptCost)
	return hash, err
}
