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

// SaveUser 保存用户
func SaveUser(user *User) error {
	if err := db.Db.Save(user).Error; err != nil {
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

var ErrInitialAdminAlreadyExists = errors.New("初始管理员已创建")

// HasAnyUser 判断用户表中是否已有用户。
func HasAnyUser() (bool, error) {
	var count int64
	if err := db.Db.Model(&User{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateInitialAdmin 创建首个管理员用户。
func CreateInitialAdmin(username string, password string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("管理员用户名不能为空")
	}
	if err := ValidateUserPassword(password); err != nil {
		return nil, err
	}

	var created User
	if err := db.Db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&User{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrInitialAdminAlreadyExists
		}
		hash, err := HashUserPassword(password)
		if err != nil {
			return err
		}
		created = User{Username: username, Password: hash}
		return tx.Create(&created).Error
	}); err != nil {
		return nil, err
	}

	return &created, nil
}

// 修改用户密码
// 传入用户 ID 和新密码，更新用户的密码
func (user *User) ChangeUsernameAndPassword(username, newPassword string) (bool, error) {
	if username == user.Username && newPassword == "" {
		return false, nil
	}
	isChange := false
	if newPassword != "" {
		if err := ValidateUserPassword(newPassword); err != nil {
			return false, err
		}
		hash, err := HashUserPassword(newPassword)
		if err != nil {
			helpers.AppLogger.Warnf("生成用户新密码失败：%v", err)
			return false, err
		}
		user.Password = hash
		isChange = true
	}
	user.Username = username
	if err := db.Db.Save(user).Error; err != nil {
		helpers.AppLogger.Errorf("修改用户名和密码失败：%v", err)
		return false, err
	}
	return isChange, nil
}

// 根据用户 ID 查询用户
func GetUserById(userId uint) (*User, error) {
	user := &User{}
	result := db.Db.First(user, userId)
	if result.Error != nil {
		helpers.AppLogger.Errorf("查询用户失败：%v", db.Db.Error)
		// 如果没有，则返回 nil
		return user, db.Db.Error
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
	if err := db.Db.Model(user).Update("password", nextHash).Error; err != nil {
		if helpers.AppLogger != nil {
			helpers.AppLogger.Warnf("保存升级后的用户密码哈希失败：%v", err)
		}
		return
	}
	user.Password = nextHash
}

// GetPwd 给密码加密
func GetPwd(pwd string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), UserPasswordBcryptCost)
	return hash, err
}
