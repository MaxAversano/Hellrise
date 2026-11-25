//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func appExit() { trayQuit() }

func relaunchLater(exe string) {
	c := exec.Command("cmd.exe")
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000,
		CmdLine: `cmd.exe /c ping -n 3 127.0.0.1 >nul & start "" "` + exe + `"`}
	c.Start()
}
