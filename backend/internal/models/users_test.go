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

func setupInitialAdminTestDB(t *testing.T) {
	t.Helper()
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}
}

func TestHasAnyUser(t *testing.T) {
	setupInitialAdminTestDB(t)

	hasUser, err := HasAnyUser()
	if err != nil {
		t.Fatalf("HasAnyUser() error = %v", err)
	}
	if hasUser {
		t.Fatal("空用户表 HasAnyUser() = true，期望 false")
	}

	if err := db.Db.Create(&User{Username: "admin", Password: "hashed"}).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	hasUser, err = HasAnyUser()
	if err != nil {
		t.Fatalf("HasAnyUser() after create error = %v", err)
	}
	if !hasUser {
		t.Fatal("已有用户时 HasAnyUser() = false，期望 true")
	}
}

func TestCreateInitialAdminCreatesHashedFirstUser(t *testing.T) {
	setupInitialAdminTestDB(t)

	user, err := CreateInitialAdmin(" admin ", "admin123")
	if err != nil {
		t.Fatalf("CreateInitialAdmin() error = %v", err)
	}
	if user.ID == 0 {
		t.Fatal("CreateInitialAdmin() 未返回已创建用户 ID")
	}
	if user.Username != "admin" {
		t.Fatalf("Username = %q，期望 admin", user.Username)
	}
	if user.Password == "admin123" {
		t.Fatal("用户密码不应以明文保存")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("admin123")); err != nil {
		t.Fatalf("保存的密码哈希无法校验原始密码: %v", err)
	}
}

func TestCreateInitialAdminRejectsWhenUserExists(t *testing.T) {
	setupInitialAdminTestDB(t)
	if _, err := CreateInitialAdmin("admin", "admin123"); err != nil {
		t.Fatalf("首次 CreateInitialAdmin() error = %v", err)
	}

	_, err := CreateInitialAdmin("other", "other123")
	if !errors.Is(err, ErrInitialAdminAlreadyExists) {
		t.Fatalf("第二次 CreateInitialAdmin() error = %v，期望 ErrInitialAdminAlreadyExists", err)
	}
}
