//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func openAppWindow(url string)                  { fmt.Println("Hellrise:", url) }
func setDPIAware()                              {}
func framelessLoop()                            {}
func windowAction(a string)                     {}
func isFrameless() bool                         { return false }
func setInner(h, dpr float64)                   {}
func startTrayThread()                          {}
func appExit()                                  { os.Exit(0) }
func relaunchLater(exe string)                  {}
func notify(title, msg string)                  {}
func setAutostart(on bool)                      {}
func openURL(u string)                          {}
func openFolder(p string)                       {}
func hiddenCmd(n string, a ...string) *exec.Cmd { return exec.Command(n, a...) }

func startOverlay()         {}
func reloadOverlayParagon() {}

func openGuideWindow(u string) {}

func browserDump(u string) (string, error) { return "", fmt.Errorf("solo Windows") }
