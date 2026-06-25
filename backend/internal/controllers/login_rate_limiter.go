package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"Q115-STRM/internal/helpers"

	"github.com/gin-gonic/gin"
)

const (
	loginFailureLimit  = 5
	loginFailureWindow = 15 * time.Minute
	loginLockDuration  = 15 * time.Minute
)

type loginRateLimitEntry struct {
	Count       int
	FirstSeen   time.Time
	LockedUntil time.Time
	LastReason  string
}

type LoginRateLimiter struct {
	mu           sync.Mutex
	limit        int
	window       time.Duration
	lockDuration time.Duration
	entries      map[string]*loginRateLimitEntry
}

func NewLoginRateLimiter(limit int, window time.Duration, lockDuration time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		limit:        limit,
		window:       window,
		lockDuration: lockDuration,
		entries:      make(map[string]*loginRateLimitEntry),
	}
}

var defaultLoginRateLimiter = NewLoginRateLimiter(loginFailureLimit, loginFailureWindow, loginLockDuration)

func loginRateLimitKey(ip string, username string) string {
	return strings.TrimSpace(ip) + "\x00" + strings.ToLower(strings.TrimSpace(username))
}

func (l *LoginRateLimiter) Allow(ip string, username string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	key := loginRateLimitKey(ip, username)
	entry, ok := l.entries[key]
	if !ok {
		return true, 0
	}
	if !entry.LockedUntil.IsZero() && now.Before(entry.LockedUntil) {
		return false, time.Until(entry.LockedUntil).Round(time.Second)
	}
	if now.Sub(entry.FirstSeen) > l.window {
		delete(l.entries, key)
		return true, 0
	}
	return true, 0
}

func (l *LoginRateLimiter) RecordFailure(ip string, username string, reason string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	key := loginRateLimitKey(ip, username)
	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.FirstSeen) > l.window {
		entry = &loginRateLimitEntry{FirstSeen: now}
		l.entries[key] = entry
	}
	entry.Count++
	entry.LastReason = reason
	if entry.Count >= l.limit {
		entry.LockedUntil = now.Add(l.lockDuration)
	}
	if helpers.AppLogger != nil {
		helpers.AppLogger.Warnf("登录失败：ip=%s username=%s reason=%s count=%d locked_until=%s", ip, username, reason, entry.Count, entry.LockedUntil.Format(time.RFC3339))
	}
}

func (l *LoginRateLimiter) Reset(ip string, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, loginRateLimitKey(ip, username))
}

func writeLoginRateLimited(c *gin.Context, wait time.Duration) {
	c.JSON(http.StatusTooManyRequests, APIResponse[any]{
		Code:    BadRequest,
		Message: fmt.Sprintf("请求过于频繁，请 %d 秒后再试", int(wait.Seconds())),
		Data:    nil,
	})
}
