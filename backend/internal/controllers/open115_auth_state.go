package controllers

import (
	"fmt"
	"sync"
	"time"

	"qmediasync/internal/v115open"
)

type open115AuthState struct {
	AccountId  uint
	CodeData   *v115open.QrCodeDataReturn
	CreatedAt  int64
	LastStatus v115open.QrCodeScanStatus
	TokenSaved bool
}

var open115AuthStates = struct {
	sync.RWMutex
	items map[string]*open115AuthState
}{items: map[string]*open115AuthState{}}

func open115AuthStateKey(accountId uint, uid string) string {
	return fmt.Sprintf("%d:%s", accountId, uid)
}

func saveOpen115AuthState(accountId uint, data *v115open.QrCodeDataReturn) {
	open115AuthStates.Lock()
	defer open115AuthStates.Unlock()
	open115AuthStates.items[open115AuthStateKey(accountId, data.Uid)] = &open115AuthState{
		AccountId:  accountId,
		CodeData:   data,
		CreatedAt:  time.Now().Unix(),
		LastStatus: v115open.QrCodeScanStatusNotScanned,
	}
}

func getOpen115AuthState(accountId uint, uid string) (*open115AuthState, bool) {
	open115AuthStates.RLock()
	defer open115AuthStates.RUnlock()
	state, ok := open115AuthStates.items[open115AuthStateKey(accountId, uid)]
	if !ok || time.Now().Unix()-state.CreatedAt > 300 {
		return nil, false
	}
	return state, true
}

func deleteOpen115AuthState(accountId uint, uid string) {
	open115AuthStates.Lock()
	defer open115AuthStates.Unlock()
	delete(open115AuthStates.items, open115AuthStateKey(accountId, uid))
}

func markOpen115AuthTokenSaving(accountId uint, uid string) bool {
	open115AuthStates.Lock()
	defer open115AuthStates.Unlock()
	state, ok := open115AuthStates.items[open115AuthStateKey(accountId, uid)]
	if !ok || state.TokenSaved {
		return false
	}
	state.TokenSaved = true
	return true
}

func resetOpen115AuthTokenSaving(accountId uint, uid string) {
	open115AuthStates.Lock()
	defer open115AuthStates.Unlock()
	if state, ok := open115AuthStates.items[open115AuthStateKey(accountId, uid)]; ok {
		state.TokenSaved = false
	}
}

func setOpen115AuthLastStatus(accountId uint, uid string, status v115open.QrCodeScanStatus) {
	open115AuthStates.Lock()
	defer open115AuthStates.Unlock()
	if state, ok := open115AuthStates.items[open115AuthStateKey(accountId, uid)]; ok {
		state.LastStatus = status
	}
}
