package models

import (
	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	Username string `gorm:"unique;not null" json:"username"`
	Password string `gorm:"not null"`
}

// 表名
func (User) TableName() string {
	return "users"
}

// 修改用户密码
// 传入用户ID和新密码，更新用户的密码
func (user *User) ChangeUsernameAndPassword(username, newPassword string) (bool, error) {
	if username == user.Username && newPassword == "" {
		return false, nil
	}
	isChange := false
	if newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.MinCost)
		if err != nil {
			helpers.AppLogger.Warnf("生成用户新密码失败: %v", err)
			return false, err
		}
		user.Password = string(hash)
		isChange = true
	}
	user.Username = username
	if err := db.Db.Save(user).Error; err != nil {
		helpers.AppLogger.Errorf("修改用户名和密码失败: %v", err)
		return false, err
	}
	return isChange, nil
}

// 根据用户ID查询用户
func GetUserById(userId uint) (*User, error) {
	user := &User{}
	result := db.Db.First(user, userId)
	if result.Error != nil {
		helpers.AppLogger.Errorf("查询用户失败: %v", db.Db.Error)
		// 如果没有，则返回Nil
		return user, db.Db.Error
	}
	return user, nil
}

// 根据用户名查询用户
// 如果没有找到，则返回一个空的User对象和nil错误
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	result := db.Db.First(user, "username = ?", username)
	if result.Error != nil {
		helpers.AppLogger.Errorf("根据用户名查询用户失败: %v", result.Error)
		// 如果没有，则返回Nil
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
	// helpers.AppLogger.Infof("检查用户%s的密码: %s => %s", username, password, user.Password)
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}
	return user, nil
}

// GetPwd 给密码加密
func GetPwd(pwd string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	return hash, err
}
