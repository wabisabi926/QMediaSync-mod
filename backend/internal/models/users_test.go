package models

import (
	"testing"

	"Q115-STRM/internal/db"

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
