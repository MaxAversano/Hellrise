//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// ---------- Icona nell'area di notifica (vicino all'orologio) con menu Apri / Esci ----------

var (
	pShellNotifyIcon = syscall.NewLazyDLL("shell32.dll").NewProc("Shell_NotifyIconW")
	pLoadImage       = u32.NewProc("LoadImageW")
	pCreatePopupMenu = u32.NewProc("CreatePopupMenu")
	pAppendMenu      = u32.NewProc("AppendMenuW")
	pTrackPopupMenu  = u32.NewProc("TrackPopupMenu")
	pDestroyMenu     = u32.NewProc("DestroyMenu")
	pGetCursorPos    = u32.NewProc("GetCursorPos")
	pSetForeground   = u32.NewProc("SetForegroundWindow")
	pPostMessage     = u32.NewProc("PostMessageW")
)

type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

const (
	wmTray      = 0x8000 + 1 // WM_APP + 1
	nimAdd      = 0
	nimDelete   = 2
	nifMessage  = 1
	nifIcon     = 2
	nifTip      = 4
	menuOpen    = 1
	menuOverlay = 2
	menuQuit    = 3
)

var (
	trayHwnd uintptr
	trayData notifyIconData
)

func trayQuit() {
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&trayData)))
	os.Exit(0)
}

func trayMenu(hwnd uintptr) {
	m, _, _ := pCreatePopupMenu.Call()
	add := func(id int, text string) {
		t, _ := syscall.UTF16PtrFromString(text)
		pAppendMenu.Call(m, 0, uintptr(id), uintptr(unsafe.Pointer(t)))
	}
	add(menuOpen, "Apri Hellrise")
	pAppendMenu.Call(m, 0x800, 0, 0) // separatore
	add(menuQuit, "Esci da Hellrise")
	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForeground.Call(hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(m, 0x100|0x80|0x2, uintptr(pt.x), uintptr(pt.y), 0, hwnd, 0)
	pPostMessage.Call(hwnd, 0, 0, 0)
	pDestroyMenu.Call(m)
	switch cmd {
	case menuOpen:
		go openAppWindow(appURL)
	case menuQuit:
		trayQuit()
	}
}

func trayProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if msg == wmTray {
		switch lparam & 0xFFFF {
		case 0x0203, 0x0202: // doppio clic / clic sinistro
			go openAppWindow(appURL)
		case 0x0205: // clic destro
			trayMenu(hwnd)
		}
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wparam, lparam)
	return r
}

// va chiamata dal thread dell'overlay (stesso ciclo di messaggi)
func startTray(hinst uintptr) {
	cls, _ := syscall.UTF16PtrFromString("HellriseTray")
	wc := wndClassEx{lpfnWndProc: syscall.NewCallback(trayProc), hInstance: hinst, lpszClassName: cls}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	trayHwnd, _, _ = pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(cls)), 0, 0, 0, 0, 0, 0, 0, hinst, 0)
	if trayHwnd == 0 {
		return
	}
	cx, _, _ := pGetSystemMetrics.Call(49) // SM_CXSMICON
	icon, _, _ := pLoadImage.Call(hinst, 1, 1, cx, cx, 0)
	trayData = notifyIconData{hWnd: trayHwnd, uID: 1, uFlags: nifMessage | nifIcon | nifTip, uCallbackMessage: wmTray, hIcon: icon}
	trayData.cbSize = uint32(unsafe.Sizeof(trayData))
	tip, _ := syscall.UTF16FromString("Hellrise · clic destro per uscire")
	copy(trayData.szTip[:], tip)
	pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&trayData)))
}

// notifica di Windows (fumetto dall'icona vicino all'orologio)
func notify(title, msg string) {
	if trayData.hWnd == 0 || !getSettings().Notifications {
		return
	}
	d := trayData
	d.uFlags = 0x10 // NIF_INFO
	tt, _ := syscall.UTF16FromString(title)
	mm, _ := syscall.UTF16FromString(msg)
	copy(d.szInfoTitle[:63], tt)
	copy(d.szInfo[:255], mm)
	d.dwInfoFlags = 0x4 | 0x20 // NIIF_USER | NIIF_LARGE_ICON: icona dell'app
	d.hBalloonIcon = trayData.hIcon
	pShellNotifyIcon.Call(1, uintptr(unsafe.Pointer(&d))) // NIM_MODIFY
}

// avvio con Windows: chiave Run dell'utente (nessun permesso da amministratore)
func setAutostart(on bool) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	var c *exec.Cmd
	if on {
		c = exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Hellrise", "/t", "REG_SZ", "/d", `"`+exe+`" --nascosto`, "/f")
	} else {
		c = exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Hellrise", "/f")
	}
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	c.Run()
}
