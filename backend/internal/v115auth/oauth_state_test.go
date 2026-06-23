package v115auth

import (
	"testing"
	"time"
)

func TestOAuthStateStore(t *testing.T) {
	ResetOAuthStatesForTest()
	now := time.Now().Unix()
	SaveOAuthState(OAuthState{
		State:       "state-1",
		AccountID:   10,
		Provider:    ProviderMoviePilot,
		RedirectURL: "http://127.0.0.1/callback",
		CreatedAt:   now,
		ExpiresAt:   now + 600,
	})

	state, ok := GetOAuthState("state-1", ProviderMoviePilot)
	if !ok {
		t.Fatal("应能读取未过期且 provider 匹配的 state")
	}
	if state.AccountID != 10 {
		t.Fatalf("account_id = %d，期望 10", state.AccountID)
	}
	if _, ok := GetOAuthState("state-1", ProviderOpenList); ok {
		t.Fatal("provider 不匹配的 state 不应可读")
	}

	DeleteOAuthState("state-1")
	if _, ok := GetOAuthState("state-1", ProviderMoviePilot); ok {
		t.Fatal("删除后的 state 不应可读")
	}
}

func TestOAuthStateExpires(t *testing.T) {
	ResetOAuthStatesForTest()
	now := time.Now().Unix()
	SaveOAuthState(OAuthState{
		State:     "expired",
		AccountID: 11,
		Provider:  ProviderCloudDrive,
		CreatedAt: now - 700,
		ExpiresAt: now - 100,
	})

	if _, ok := GetOAuthState("expired", ProviderCloudDrive); ok {
		t.Fatal("过期 state 不应可读")
	}
}
