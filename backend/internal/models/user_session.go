package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"qmediasync/internal/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserSession struct {
	BaseModel
	SessionID     string `json:"session_id" gorm:"uniqueIndex;size:64;not null"` // 会话 ID
	TokenID       string `json:"-" gorm:"uniqueIndex;size:64;not null"`          // JWT jti
	UserID        uint   `json:"user_id" gorm:"index;not null"`                  // 用户 ID
	Username      string `json:"username" gorm:"size:128;not null"`              // 登录用户名快照
	CSRFTokenHash string `json:"-" gorm:"size:64;not null"`                      // CSRF Token 哈希
	UserAgent     string `json:"user_agent" gorm:"type:text"`                    // User-Agent
	IPAddress     string `json:"ip_address" gorm:"size:64"`                      // 登录 IP
	ExpiresAt     int64  `json:"expires_at" gorm:"index;not null"`               // 过期时间
	LastSeenAt    int64  `json:"last_seen_at" gorm:"index"`                      // 最后活跃时间
	RevokedAt     int64  `json:"revoked_at" gorm:"index"`                        // 撤销时间
	RevokeReason  string `json:"revoke_reason" gorm:"size:64"`                   // 撤销原因
}

func (UserSession) TableName() string {
	return "user_sessions"
}

type CreateUserSessionInput struct {
	UserID    uint
	Username  string
	UserAgent string
	IPAddress string
	ExpiresAt int64
}

func GenerateSessionSecret(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = 32
	}
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机会话密钥失败：%w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashSessionSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func CreateUserSession(input CreateUserSessionInput) (*UserSession, string, error) {
	if input.UserID == 0 {
		return nil, "", fmt.Errorf("用户 ID 不能为空")
	}
	sessionID, err := GenerateSessionSecret(32)
	if err != nil {
		return nil, "", err
	}
	tokenID, err := GenerateSessionSecret(32)
	if err != nil {
		return nil, "", err
	}
	csrfToken, err := GenerateSessionSecret(32)
	if err != nil {
		return nil, "", err
	}
	now := time.Now().Unix()
	session := &UserSession{
		SessionID:     sessionID,
		TokenID:       tokenID,
		UserID:        input.UserID,
		Username:      input.Username,
		CSRFTokenHash: HashSessionSecret(csrfToken),
		UserAgent:     input.UserAgent,
		IPAddress:     input.IPAddress,
		ExpiresAt:     input.ExpiresAt,
		LastSeenAt:    now,
	}
	if err := db.Db.Create(session).Error; err != nil {
		return nil, "", err
	}
	return session, csrfToken, nil
}

func GetActiveUserSession(sessionID string, now int64) (*UserSession, error) {
	var session UserSession
	err := db.Db.Where("session_id = ? AND revoked_at = 0 AND expires_at > ?", sessionID, now).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func ListActiveUserSessions(userID uint, currentSessionID string, now int64) ([]UserSession, error) {
	var sessions []UserSession
	query := db.Db.Where("user_id = ? AND revoked_at = 0 AND expires_at > ?", userID, now)
	if currentSessionID != "" {
		query = query.Order(clause.OrderBy{Expression: clause.Expr{
			SQL:  "CASE WHEN session_id = ? THEN 0 ELSE 1 END, last_seen_at DESC",
			Vars: []any{currentSessionID},
		}})
	} else {
		query = query.Order("last_seen_at DESC")
	}
	err := query.Find(&sessions).Error
	return sessions, err
}

// UpdateUserSessionCSRFHash 更新活动会话的 CSRF Token 哈希。
func UpdateUserSessionCSRFHash(session *UserSession) error {
	result := db.Db.Model(&UserSession{}).
		Where("id = ? AND revoked_at = 0", session.ID).
		Update("csrf_token_hash", session.CSRFTokenHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func RevokeUserSession(userID uint, sessionID string, reason string) error {
	return db.Db.Model(&UserSession{}).
		Where("user_id = ? AND session_id = ? AND revoked_at = 0", userID, sessionID).
		Updates(map[string]any{"revoked_at": time.Now().Unix(), "revoke_reason": reason}).Error
}

func RevokeOtherUserSessions(userID uint, keepSessionID string, reason string) error {
	return db.Db.Model(&UserSession{}).
		Where("user_id = ? AND session_id <> ? AND revoked_at = 0", userID, keepSessionID).
		Updates(map[string]any{"revoked_at": time.Now().Unix(), "revoke_reason": reason}).Error
}

func RevokeAllUserSessions(userID uint, reason string) error {
	return revokeAllUserSessions(db.Db, userID, reason)
}

func revokeAllUserSessions(tx *gorm.DB, userID uint, reason string) error {
	return tx.Model(&UserSession{}).
		Where("user_id = ? AND revoked_at = 0", userID).
		Updates(map[string]any{"revoked_at": time.Now().Unix(), "revoke_reason": reason}).Error
}

func TouchUserSession(sessionID string, now int64) error {
	return db.Db.Model(&UserSession{}).
		Where("session_id = ? AND revoked_at = 0", sessionID).
		Update("last_seen_at", now).Error
}

func (session *UserSession) ValidateCSRFToken(rawToken string) bool {
	return session != nil && rawToken != "" && session.CSRFTokenHash == HashSessionSecret(rawToken)
}
