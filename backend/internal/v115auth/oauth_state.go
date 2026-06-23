package v115auth

import (
	"sync"
	"time"
)

const OAuthStateTTLSeconds int64 = 600

type OAuthState struct {
	State       string
	AccountID   uint
	Provider    AuthProvider
	RedirectURL string
	CreatedAt   int64
	ExpiresAt   int64
}

var oauthStates = struct {
	sync.Mutex
	items map[string]OAuthState
}{
	items: map[string]OAuthState{},
}

func SaveOAuthState(state OAuthState) {
	now := time.Now().Unix()
	if state.CreatedAt == 0 {
		state.CreatedAt = now
	}
	if state.ExpiresAt == 0 {
		state.ExpiresAt = state.CreatedAt + OAuthStateTTLSeconds
	}
	oauthStates.Lock()
	oauthStates.items[state.State] = state
	oauthStates.Unlock()
}

func GetOAuthState(state string, provider AuthProvider) (OAuthState, bool) {
	oauthStates.Lock()
	defer oauthStates.Unlock()
	item, ok := oauthStates.items[state]
	if !ok {
		return OAuthState{}, false
	}
	if item.Provider != provider || item.ExpiresAt <= time.Now().Unix() {
		if item.ExpiresAt <= time.Now().Unix() {
			delete(oauthStates.items, state)
		}
		return OAuthState{}, false
	}
	return item, true
}

func DeleteOAuthState(state string) {
	oauthStates.Lock()
	delete(oauthStates.items, state)
	oauthStates.Unlock()
}

func CleanupExpiredOAuthStates(now int64) {
	oauthStates.Lock()
	for state, item := range oauthStates.items {
		if item.ExpiresAt <= now {
			delete(oauthStates.items, state)
		}
	}
	oauthStates.Unlock()
}

func ResetOAuthStatesForTest() {
	oauthStates.Lock()
	oauthStates.items = map[string]OAuthState{}
	oauthStates.Unlock()
}
