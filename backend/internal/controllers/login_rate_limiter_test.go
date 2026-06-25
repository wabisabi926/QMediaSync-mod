package controllers

import (
	"testing"
	"time"
)

func TestLoginRateLimiterLocksByIPAndUsername(t *testing.T) {
	limiter := NewLoginRateLimiter(3, time.Minute, time.Minute)
	ip := "127.0.0.1"
	username := "admin"

	for i := 0; i < 3; i++ {
		if allowed, _ := limiter.Allow(ip, username); !allowed {
			t.Fatalf("第 %d 次失败前不应被锁定", i+1)
		}
		limiter.RecordFailure(ip, username, "password_mismatch")
	}
	if allowed, wait := limiter.Allow(ip, username); allowed || wait <= 0 {
		t.Fatalf("达到阈值后应被锁定，allowed=%v wait=%v", allowed, wait)
	}

	if allowed, _ := limiter.Allow(ip, "other"); !allowed {
		t.Fatalf("不同用户名不应共享锁定")
	}
	if allowed, _ := limiter.Allow("127.0.0.2", username); !allowed {
		t.Fatalf("不同 IP 不应共享锁定")
	}

	limiter.Reset(ip, username)
	if allowed, _ := limiter.Allow(ip, username); !allowed {
		t.Fatalf("Reset 后应允许登录")
	}
}
