package models

import (
	"errors"
	"testing"
	"time"

	"qmediasync/internal/db"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserSessionTestDB(t *testing.T) *User {
	t.Helper()

	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&User{}, &UserSession{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}

	user := &User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return user
}

func TestUserSessionLifecycle(t *testing.T) {
	user := setupUserSessionTestDB(t)
	expiresAt := time.Now().Add(time.Hour).Unix()

	session, rawCSRF, err := CreateUserSession(CreateUserSessionInput{
		UserID:    user.ID,
		Username:  user.Username,
		UserAgent: "Mozilla/5.0",
		IPAddress: "127.0.0.1",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("CreateUserSession() error = %v", err)
	}
	if session.SessionID == "" || session.TokenID == "" || rawCSRF == "" {
		t.Fatalf("session id、token id、csrf token 不应为空: %#v csrf=%q", session, rawCSRF)
	}

	found, err := GetActiveUserSession(session.SessionID, time.Now().Unix())
	if err != nil {
		t.Fatalf("GetActiveUserSession() error = %v", err)
	}
	if found.UserID != user.ID {
		t.Fatalf("found.UserID = %d, want %d", found.UserID, user.ID)
	}
	if !found.ValidateCSRFToken(rawCSRF) {
		t.Fatalf("ValidateCSRFToken() = false, want true")
	}

	if err := RevokeUserSession(user.ID, session.SessionID, "test"); err != nil {
		t.Fatalf("RevokeUserSession() error = %v", err)
	}
	if _, err := GetActiveUserSession(session.SessionID, time.Now().Unix()); err == nil {
		t.Fatalf("撤销后的 session 不应仍然有效")
	}
}

func TestListActiveUserSessionsFiltersRevokedAndExpired(t *testing.T) {
	user := setupUserSessionTestDB(t)
	now := time.Now().Unix()

	active, _, err := CreateUserSession(CreateUserSessionInput{UserID: user.ID, Username: user.Username, ExpiresAt: now + 3600})
	if err != nil {
		t.Fatalf("创建有效 session 失败: %v", err)
	}
	revoked, _, err := CreateUserSession(CreateUserSessionInput{UserID: user.ID, Username: user.Username, ExpiresAt: now + 3600})
	if err != nil {
		t.Fatalf("创建撤销 session 失败: %v", err)
	}
	expired, _, err := CreateUserSession(CreateUserSessionInput{UserID: user.ID, Username: user.Username, ExpiresAt: now - 1})
	if err != nil {
		t.Fatalf("创建过期 session 失败: %v", err)
	}
	if err := RevokeUserSession(user.ID, revoked.SessionID, "test"); err != nil {
		t.Fatalf("撤销 session 失败: %v", err)
	}

	sessions, err := ListActiveUserSessions(user.ID, "", now)
	if err != nil {
		t.Fatalf("ListActiveUserSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != active.SessionID {
		t.Fatalf("sessions = %#v, want only %s; expired=%s", sessions, active.SessionID, expired.SessionID)
	}
}

func TestListActiveUserSessionsSortsCurrentSessionFirst(t *testing.T) {
	user := setupUserSessionTestDB(t)
	now := time.Now().Unix()
	current, _, err := CreateUserSession(CreateUserSessionInput{UserID: user.ID, Username: user.Username, ExpiresAt: now + 3600})
	if err != nil {
		t.Fatalf("创建当前 session 失败: %v", err)
	}
	other, _, err := CreateUserSession(CreateUserSessionInput{UserID: user.ID, Username: user.Username, ExpiresAt: now + 3600})
	if err != nil {
		t.Fatalf("创建其他 session 失败: %v", err)
	}
	if err := db.Db.Model(other).Update("last_seen_at", now+60).Error; err != nil {
		t.Fatalf("设置其他 session 最后活跃时间失败: %v", err)
	}

	withCurrent, err := ListActiveUserSessions(user.ID, current.SessionID, now)
	if err != nil {
		t.Fatalf("查询带当前 session 的活动会话失败: %v", err)
	}
	if len(withCurrent) != 2 || withCurrent[0].SessionID != current.SessionID {
		t.Fatalf("带当前 session 的列表 = %#v，期望当前会话 %s 置顶", withCurrent, current.SessionID)
	}

	withoutCurrent, err := ListActiveUserSessions(user.ID, "", now)
	if err != nil {
		t.Fatalf("查询不带当前 session 的活动会话失败: %v", err)
	}
	if len(withoutCurrent) != 2 || withoutCurrent[0].SessionID != other.SessionID {
		t.Fatalf("不带当前 session 的列表 = %#v，期望最后活跃的会话 %s 置顶", withoutCurrent, other.SessionID)
	}
}

func TestUpdateUserSessionCSRFHashDoesNotReactivateRevokedSession(t *testing.T) {
	user := setupUserSessionTestDB(t)
	session, _, err := CreateUserSession(CreateUserSessionInput{
		UserID:    user.ID,
		Username:  user.Username,
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	staleSession := *session
	if err := RevokeAllUserSessions(user.ID, "credential_changed"); err != nil {
		t.Fatalf("撤销会话失败: %v", err)
	}
	staleSession.CSRFTokenHash = HashSessionSecret("replacement-token")
	if err := UpdateUserSessionCSRFHash(&staleSession); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("保存陈旧会话 error = %v，期望 gorm.ErrRecordNotFound", err)
	}
	if _, err := GetActiveUserSession(session.SessionID, time.Now().Unix()); err == nil {
		t.Fatal("保存陈旧会话不应重新激活已撤销会话")
	}
}
