//go:build !windows
// +build !windows

package database

import "syscall"

func setHideWindow(attr *syscall.SysProcAttr) {
	// 在非 Windows 系统上什么都不做
}

func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
