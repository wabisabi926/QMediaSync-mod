package models

import (
	"errors"
	"testing"

	"qmediasync/internal/db"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestCheckLoginRehashesLowCostPassword(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("生成低成本密码失败: %v", err)
	}
	user := &User{Username: "admin", Password: string(hash)}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	if _, err := CheckLogin("admin", "admin123"); err != nil {
		t.Fatalf("CheckLogin() error = %v", err)
	}

	var updated User
	if err := db.Db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(updated.Password))
	if err != nil {
		t.Fatalf("读取 bcrypt cost 失败: %v", err)
	}
	if cost < UserPasswordBcryptCost {
		t.Fatalf("bcrypt cost = %d, want >= %d", cost, UserPasswordBcryptCost)
	}
}



func TestUserTableAllowsOnlyOneUser(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}
	if err := db.Db.Create(&User{Username: "admin", Password: "hashed"}).Error; err != nil {
		t.Fatalf("创建首个用户失败: %v", err)
	}

	if err := db.Db.Create(&User{Username: "other", Password: "hashed"}).Error; err == nil {
		t.Fatal("创建第二个用户 error = nil，期望被唯一约束拒绝")
	}
}

func TestSaveUserTwoFactorDoesNotOverwriteChangedPasswordFromStaleUser(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&User{}, &UserSession{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	passwordHash, err := HashUserPassword("old-password")
	if err != nil {
		t.Fatalf("生成旧密码哈希失败: %v", err)
	}
	user := &User{Username: "admin", Password: passwordHash}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	var staleUser User
	if err := db.Db.First(&staleUser, user.ID).Error; err != nil {
		t.Fatalf("读取陈旧用户失败: %v", err)
	}
	if changed, err := user.ChangeUsernameAndPasswordAndRevokeSessions("admin", "new-password"); err != nil || !changed {
		t.Fatalf("修改密码 changed=%t err=%v，期望成功", changed, err)
	}
	staleUser.TwoFactorPendingSecret = "pending-secret"
	if err := SaveUserTwoFactor(&staleUser); err != nil {
		t.Fatalf("保存两步验证草稿失败: %v", err)
	}
	if _, err := CheckLogin("admin", "old-password"); err == nil {
		t.Fatal("陈旧用户保存不应覆盖新密码")
	}
	if _, err := CheckLogin("admin", "new-password"); err != nil {
		t.Fatalf("新密码应保持可用: %v", err)
	}
}
