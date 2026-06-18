//go:build !windows
// +build !windows

package helpers

import (
	"os/exec"
	"runtime"
)

var ExitChan chan struct{} = make(chan struct{})
var IsFirstRun bool = false // 默认为 false

func StartApp(stopFunc func()) {
}

func StopApp() {}

func StartNewProcess(exePath, updateDir string) bool {
	return true
}

func IsProcessAlive(pid int) (bool, error) {
	return true, nil
}

func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}
