//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	u32 = syscall.NewLazyDLL("user32.dll")
	g32 = syscall.NewLazyDLL("gdi32.dll")
	k32 = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassEx  = u32.NewProc("RegisterClassExW")
	pCreateWindowEx   = u32.NewProc("CreateWindowExW")
	pDefWindowProc    = u32.NewProc("DefWindowProcW")
	pGetMessage       = u32.NewProc("GetMessageW")
	pTranslateMessage = u32.NewProc("TranslateMessage")
	pDispatchMessage  = u32.NewProc("DispatchMessageW")
	pSetWindowPos     = u32.NewProc("SetWindowPos")
	pSendMessage      = u32.NewProc("SendMessageW")
	pShowWindow       = u32.NewProc("ShowWindow")
	pGetSystemMetrics = u32.NewProc("GetSystemMetrics")
	pGetModuleHandle  = k32.NewProc("GetModuleHandleW")
	pCreateSolidBrush = g32.NewProc("CreateSolidBrush")
	pSetCapture       = u32.NewProc("SetCapture")
	pReleaseCapture   = u32.NewProc("ReleaseCapture")
	pGetWindowLongPtr = u32.NewProc("GetWindowLongPtrW")
	pSetWindowLongPtr = u32.NewProc("SetWindowLongPtrW")
	pLoadCursor       = u32.NewProc("LoadCursorW")
	pSetCursor        = u32.NewProc("SetCursor")
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type point struct{ x, y int32 }
type msgT struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
	priv    uint32
}
type rect struct{ l, t, r, b int32 }

func screenSize() (int32, int32) {
	w, _, _ := pGetSystemMetrics.Call(0)
	h, _, _ := pGetSystemMetrics.Call(1)
	return int32(w), int32(h)
}

// thread con il ciclo dei messaggi per l'icona vicino all'orologio
func startTrayThread() {
	runtime.LockOSThread()
	hinst, _, _ := pGetModuleHandle.Call(0)
	startTray(hinst)
	var m msgT
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}
