//go:build windows

package main

import (
	"bytes"
	_ "embed"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

//go:embed assets/icon.png
var iconPNG []byte

const (
	wmDestroy        = 0x0002
	wmClose          = 0x0010
	wmDisplayChange  = 0x007E
	wmSettingChange  = 0x001A
	wmTimer          = 0x0113
	wmContextMenu    = 0x007B
	wmLButtonUp      = 0x0202
	wmRButtonUp      = 0x0205
	wmApp            = 0x8000
	wmTray           = wmApp + 1
	wmReflow         = wmApp + 2
	trayID           = 1
	timerReflow      = 1
	menuPause        = 100
	menuArrange      = 101
	menuExit         = 102
	nifMessage       = 0x1
	nifIcon          = 0x2
	nifTip           = 0x4
	nimAdd           = 0
	nimDelete        = 2
	nimSetVersion    = 4
	notifyIconV4     = 4
	mfString         = 0x0000
	mfSeparator      = 0x0800
	mfChecked        = 0x0008
	tpmRightButton   = 0x0002
	tpmReturnCmd     = 0x0100
	swpNoZOrder      = 0x0004
	swpNoActivate    = 0x0010
	swpNoOwnerZOrder = 0x0200
	swRestore        = 9
	gwOwner          = 4
	gwlStyle         = ^uintptr(15) // -16
	gwlExStyle       = ^uintptr(19) // -20
	wsCaption        = 0x00C00000
	wsExToolWindow   = 0x00000080
	wsExAppWindow    = 0x00040000
	wsExNoActivate   = 0x08000000
	monitorNearest   = 2
	dwmwaCloaked     = 14
	winEventOutOfCtx = 0x0000
	winEventSkipOwn  = 0x0002
	eventForeground  = 0x0003
	eventMinStart    = 0x0016
	eventMinEnd      = 0x0017
	eventCreate      = 0x8000
	eventDestroy     = 0x8001
	eventShow        = 0x8002
	eventHide        = 0x8003
	imageBitmap      = 0
	dibRGBColors     = 0
	biRGB            = 0
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	pRegisterClassEx               = user32.NewProc("RegisterClassExW")
	pCreateWindowEx                = user32.NewProc("CreateWindowExW")
	pDefWindowProc                 = user32.NewProc("DefWindowProcW")
	pDestroyWindow                 = user32.NewProc("DestroyWindow")
	pGetMessage                    = user32.NewProc("GetMessageW")
	pTranslateMessage              = user32.NewProc("TranslateMessage")
	pDispatchMessage               = user32.NewProc("DispatchMessageW")
	pPostQuitMessage               = user32.NewProc("PostQuitMessage")
	pPostMessage                   = user32.NewProc("PostMessageW")
	pSetTimer                      = user32.NewProc("SetTimer")
	pKillTimer                     = user32.NewProc("KillTimer")
	pEnumWindows                   = user32.NewProc("EnumWindows")
	pIsWindow                      = user32.NewProc("IsWindow")
	pIsWindowVisible               = user32.NewProc("IsWindowVisible")
	pIsIconic                      = user32.NewProc("IsIconic")
	pIsZoomed                      = user32.NewProc("IsZoomed")
	pShowWindow                    = user32.NewProc("ShowWindow")
	pGetWindowRect                 = user32.NewProc("GetWindowRect")
	pGetWindowTextLength           = user32.NewProc("GetWindowTextLengthW")
	pGetWindow                     = user32.NewProc("GetWindow")
	pGetWindowLong                 = user32.NewProc("GetWindowLongPtrW")
	pGetClassName                  = user32.NewProc("GetClassNameW")
	pGetShellWindow                = user32.NewProc("GetShellWindow")
	pMonitorFromWindow             = user32.NewProc("MonitorFromWindow")
	pGetMonitorInfo                = user32.NewProc("GetMonitorInfoW")
	pSetWindowPos                  = user32.NewProc("SetWindowPos")
	pGetWindowPlacement            = user32.NewProc("GetWindowPlacement")
	pSetWindowPlacement            = user32.NewProc("SetWindowPlacement")
	pSetWinEventHook               = user32.NewProc("SetWinEventHook")
	pUnhookWinEvent                = user32.NewProc("UnhookWinEvent")
	pCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	pAppendMenu                    = user32.NewProc("AppendMenuW")
	pTrackPopupMenu                = user32.NewProc("TrackPopupMenu")
	pDestroyMenu                   = user32.NewProc("DestroyMenu")
	pGetCursorPos                  = user32.NewProc("GetCursorPos")
	pSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	pSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	pMessageBox                    = user32.NewProc("MessageBoxW")
	pCreateIconIndirect            = user32.NewProc("CreateIconIndirect")
	pDestroyIcon                   = user32.NewProc("DestroyIcon")
	pRegisterWindowMessage         = user32.NewProc("RegisterWindowMessageW")
	pCreateMutex                   = kernel32.NewProc("CreateMutexW")
	pGetModuleHandle               = kernel32.NewProc("GetModuleHandleW")
	pCloseHandle                   = kernel32.NewProc("CloseHandle")
	pShellNotifyIcon               = shell32.NewProc("Shell_NotifyIconW")
	pCreateDIBSection              = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap                  = gdi32.NewProc("CreateBitmap")
	pDeleteObject                  = gdi32.NewProc("DeleteObject")
	pDwmGetWindowAttribute         = dwmapi.NewProc("DwmGetWindowAttribute")

	app *controller
)

type point struct{ X, Y int32 }
type monitorInfo struct {
	Size          uint32
	Monitor, Work Rectangle
	Flags         uint32
}
type windowPlacement struct {
	Length, Flags, ShowCmd   uint32
	MinPosition, MaxPosition point
	NormalPosition           Rectangle
}
type msg struct {
	Hwnd           uintptr
	Message        uint32
	_              uint32
	Wparam, Lparam uintptr
	Time           uint32
	Pt             point
	Private        uint32
}
type wndClass struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	SmallIcon                          uintptr
}
type notifyIcon struct {
	Size                uint32
	Hwnd                uintptr
	ID, Flags, Callback uint32
	Icon                uintptr
	Tip                 [128]uint16
	State, StateMask    uint32
	Info                [256]uint16
	Version             uint32
	InfoTitle           [64]uint16
	InfoFlags           uint32
	GUID                [16]byte
	BalloonIcon         uintptr
}
type bitmapInfoHeader struct {
	Size                   uint32
	Width, Height          int32
	Planes, BitCount       uint16
	Compression, ImageSize uint32
	XPels, YPels           int32
	ClrUsed, ClrImportant  uint32
}
type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}
type iconInfo struct {
	IsIcon             int32
	HotspotX, HotspotY uint32
	Mask, Color        uintptr
}

type controller struct {
	finder         finderUI
	window         uintptr
	icon           uintptr
	mutex          uintptr
	tray           notifyIcon
	hooks          []uintptr
	paused         bool
	reflowing      bool
	original       map[uintptr]windowPlacement
	order          map[uintptr]uint64
	nextOrder      uint64
	profile        Profile
	groups         map[uintptr][]windowItem
	shell          uintptr
	moves          map[uintptr]move
	surfaces       desktopUI
	taskbarMessage uint32
}

func ptr(s string) *uint16                                { return syscall.StringToUTF16Ptr(s) }
func called(p *syscall.LazyProc, args ...uintptr) uintptr { r, _, _ := p.Call(args...); return r }
func isWindow(h uintptr) bool                             { return called(pIsWindow, h) != 0 }

func main() {
	runtime.LockOSThread()
	if len(os.Args) == 3 && os.Args[1] == "--signin-art" {
		p, e := readProfile()
		if e != nil {
			os.Exit(1)
		}
		accent, _ := colorRef(p.Transition)
		if p.Accent != "auto" {
			accent, _ = colorRef(p.Accent)
		} else if c, e := sampleFile(activeWallpaper()); e == nil {
			accent = c
		}
		if signInArt(os.Args[2], readable(accent, rgb(8, 10, 20))) != nil {
			os.Exit(1)
		}
		return
	}
	ole := syscall.NewLazyDLL("ole32.dll")
	if hr := called(ole.NewProc("OleInitialize"), 0); int32(hr) >= 0 {
		defer called(ole.NewProc("OleUninitialize"))
	}
	// Keep coordinates in physical pixels on mixed-DPI, multi-monitor setups.
	if called(pSetProcessDpiAwarenessContext, ^uintptr(3)) == 0 {
		called(pSetProcessDPIAware)
	}
	mutex, _, err := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(ptr("Local\\SilkWM-Prototype-SingleInstance"))))
	if mutex == 0 {
		return
	}
	if err == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		called(pMessageBox, 0, uintptr(unsafe.Pointer(ptr("Silk is already running in the notification area."))), uintptr(unsafe.Pointer(ptr("Silk"))), 0x40)
		called(pCloseHandle, mutex)
		return
	}
	app = &controller{mutex: mutex, original: make(map[uintptr]windowPlacement), order: make(map[uintptr]uint64)}
	app.profile, err = readProfile()
	if err != nil {
		alert(err.Error())
		app.profile = defaults()
	}
	app.moves = make(map[uintptr]move)
	app.run()
}

func (c *controller) run() {
	inst := called(pGetModuleHandle, 0)
	name := ptr("SilkPrototypeHiddenWindow")
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), WndProc: syscall.NewCallback(windowProc), Instance: inst, ClassName: name}
	if called(pRegisterClassEx, uintptr(unsafe.Pointer(&wc))) == 0 {
		return
	}
	c.window = called(pCreateWindowEx, 0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(ptr("Silk"))), 0, 0, 0, 0, 0, 0, 0, inst, 0)
	if c.window == 0 {
		return
	}
	c.icon = makeIcon()
	c.taskbarMessage = uint32(called(pRegisterWindowMessage, uintptr(unsafe.Pointer(ptr("TaskbarCreated")))))
	c.addTray()
	c.hookEvents()
	c.surfaces.start(c)
	c.finder.start()
	c.schedule()
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if c.finder.key(m) {
			continue
		}
		called(pTranslateMessage, uintptr(unsafe.Pointer(&m)))
		called(pDispatchMessage, uintptr(unsafe.Pointer(&m)))
	}
	for _, hook := range c.hooks {
		called(pUnhookWinEvent, hook)
	}
	c.finder.stop()
	c.surfaces.stop(c)
	c.restoreAll()
	called(pShellNotifyIcon, nimDelete, uintptr(unsafe.Pointer(&c.tray)))
	if c.icon != 0 {
		called(pDestroyIcon, c.icon)
	}
	if c.window != 0 {
		called(pDestroyWindow, c.window)
	}
	called(pCloseHandle, c.mutex)
}

func (c *controller) addTray() {
	c.tray = notifyIcon{Size: uint32(unsafe.Sizeof(notifyIcon{})), Hwnd: c.window, ID: trayID, Flags: nifMessage | nifIcon | nifTip, Callback: wmTray, Icon: c.icon}
	copy(c.tray.Tip[:], syscall.StringToUTF16("Silk — automatic tiling"))
	called(pShellNotifyIcon, nimAdd, uintptr(unsafe.Pointer(&c.tray)))
	c.tray.Version = notifyIconV4
	called(pShellNotifyIcon, nimSetVersion, uintptr(unsafe.Pointer(&c.tray)))
}

func (c *controller) hookEvents() {
	cb := syscall.NewCallback(eventProc)
	for _, span := range [][2]uintptr{{eventCreate, eventHide}, {eventMinStart, eventMinEnd}, {0x000B, 0x000B}, {eventForeground, eventForeground}} {
		h := called(pSetWinEventHook, span[0], span[1], 0, cb, 0, 0, winEventOutOfCtx|winEventSkipOwn)
		if h != 0 {
			c.hooks = append(c.hooks, h)
		}
	}
}

func eventProc(_ uintptr, event uintptr, hwnd uintptr, object uintptr, child uintptr, _ uintptr, _ uintptr) uintptr {
	if app != nil && app.window != 0 && hwnd != 0 && (object == 0 || event == eventForeground) && (child == 0 || event == eventForeground) {
		if event == eventDestroy {
			delete(app.moves, hwnd)
			delete(app.order, hwnd)
			delete(app.original, hwnd)
		}
		called(pPostMessage, app.window, wmReflow, 0, 0)
	}
	return 0
}

func windowProc(hwnd uintptr, message uintptr, wparam uintptr, lparam uintptr) uintptr {
	if app == nil {
		return called(pDefWindowProc, hwnd, message, wparam, lparam)
	}
	if uint32(message) == app.taskbarMessage && app.taskbarMessage != 0 {
		app.addTray()
		app.surfaces.resetBar()
		return 0
	}
	switch message {
	case catalogueReady:
		app.finder.received(wparam != 0)
		return 0
	case finderToggle:
		app.finder.open()
		return 0
	case 0x312:
		if wparam == 10 {
			app.finder.open()
		}
		return 0
	case 0x02b1: // WM_WTSSESSION_CHANGE
		if wparam == 8 {
			app.surfaces.fade(false)
		}
		return 0
	case 0x2c:
		return measureMenu(lparam)
	case 0x2b:
		return paintMenu(lparam)
	case wmReflow:
		app.schedule()
		return 0
	case wmTimer:
		if wparam == 5 {
			app.surfaces.wallpaper.frame()
			return 0
		}
		if wparam == 4 {
			app.surfaces.wallpaper.refresh(false)
			return 0
		}
		if wparam == 2 {
			app.animate()
			return 0
		}
		if wparam == 3 {
			app.surfaces.frame()
			return 0
		}
		if wparam == timerReflow {
			called(pKillTimer, hwnd, timerReflow)
			if !app.paused {
				app.retile()
			}
		}
		return 0
	case wmDisplayChange, wmSettingChange:
		app.surfaces.updateMotion()
		app.surfaces.wallpaper.refresh(true)
		app.surfaces.placeBar()
		app.schedule()
		return 0
	case wmTray:
		// Shell_NotifyIcon version 4 uses LOWORD(lParam) for event type.
		event := uint32(lparam & 0xffff)
		if event == wmContextMenu || event == wmLButtonUp || event == wmRButtonUp || event == 0x0400 {
			app.showMenu()
		}
		return 0
	case wmClose:
		called(pPostQuitMessage, 0)
		return 0
	case wmDestroy:
		return 0
	}
	return called(pDefWindowProc, hwnd, message, wparam, lparam)
}

func (c *controller) schedule() {
	if c.paused || c.window == 0 {
		return
	}
	called(pKillTimer, c.window, timerReflow)
	called(pSetTimer, c.window, timerReflow, 180, 0)
}

func (c *controller) showMenu() {
	menu := called(pCreatePopupMenu)
	if menu == 0 {
		return
	}
	items := menuRows(c.paused)
	for _, row := range items {
		called(pAppendMenu, menu, 0x100, uintptr(row.id), uintptr(row.id))
		// Supply names to accessibility clients as well as painting each row.
		name := ptr(row.title)
		entry := menuItemInfo{Size: uint32(unsafe.Sizeof(menuItemInfo{})), Mask: 0x40, Text: name}
		called(user32.NewProc("SetMenuItemInfoW"), menu, uintptr(row.id), 0, uintptr(unsafe.Pointer(&entry)))
	}
	pal := colors()
	brush := solid(pal.surface)
	mi := menuInfo{Size: uint32(unsafe.Sizeof(menuInfo{})), Mask: 2, Background: brush}
	called(user32.NewProc("SetMenuInfo"), menu, uintptr(unsafe.Pointer(&mi)))
	var pt point
	called(pGetCursorPos, uintptr(unsafe.Pointer(&pt)))
	called(pSetForegroundWindow, c.window)
	choice := called(pTrackPopupMenu, menu, tpmRightButton|tpmReturnCmd|0x1000, uintptr(pt.X), uintptr(pt.Y), 0, c.window, 0)
	called(pDestroyMenu, menu)
	called(pDeleteObject, brush)
	called(pPostMessage, c.window, 0, 0, 0)
	switch choice {
	case menuPause:
		if c.paused {
			c.paused = false
			c.schedule()
		} else {
			c.paused = true
			called(pKillTimer, c.window, timerReflow)
			c.restoreAll()
		}
	case menuArrange:
		if c.paused {
			c.paused = false
		}
		c.schedule()
	case 103:
		path, e := profileFile()
		if e == nil {
			e = exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe"), path).Start()
		}
		if e != nil {
			alert(e.Error())
		}
	case 104:
		p, e := readProfile()
		if e != nil {
			alert(e.Error())
			break
		}
		c.profile = p
		if p.WinFinder && c.finder.host != 0 {
			c.finder.tap.start()
		} else {
			c.finder.tap.stop()
		}
		c.surfaces.resetBar()
		c.schedule()
	case 108:
		c.finder.open()
	case 105:
		c.surfaces.fade(false)
	case 106:
		if called(pMessageBox, c.window, uintptr(unsafe.Pointer(ptr("Restart Windows now? Save your work first."))), uintptr(unsafe.Pointer(ptr("Silk"))), 0x124) == 6 {
			c.surfaces.fade(true)
		}
	case 107:
		if e := setSilkWallpaper(); e != nil {
			alert(e.Error())
		}
		c.surfaces.wallpaper.refresh(true)
	case menuExit:
		called(pPostMessage, c.window, wmClose, 0, 0)
	}
}

type windowItem struct {
	hwnd    uintptr
	monitor uintptr
	order   uint64
}

func (c *controller) candidates() map[uintptr][]windowItem {
	c.groups = make(map[uintptr][]windowItem)
	c.shell = called(pGetShellWindow)
	called(pEnumWindows, enumCallback, 0)
	return c.groups
}

var enumCallback = syscall.NewCallback(collectWindow)

func collectWindow(h, _ uintptr) uintptr {
	c := app

	if h == c.window || h == c.shell || called(pIsWindowVisible, h) == 0 || called(pIsIconic, h) != 0 || called(pGetWindowTextLength, h) == 0 {
		return 1
	}
	ex := called(pGetWindowLong, h, gwlExStyle)
	if ex&(wsExToolWindow|wsExNoActivate) != 0 {
		return 1
	}
	if called(pGetWindow, h, gwOwner) != 0 && ex&wsExAppWindow == 0 {
		return 1
	}
	var cloaked uint32
	if called(pDwmGetWindowAttribute, h, dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked)) == 0 && cloaked != 0 {
		return 1
	}
	var name [128]uint16
	called(pGetClassName, h, uintptr(unsafe.Pointer(&name[0])), 128)
	switch syscall.UTF16ToString(name[:]) {
	case "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "NotifyIconOverflowWindow":
		return 1
	}
	monitor := called(pMonitorFromWindow, h, monitorNearest)
	if monitor == 0 {
		return 1
	}
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if called(pGetMonitorInfo, monitor, uintptr(unsafe.Pointer(&info))) == 0 {
		return 1
	}
	var rect Rectangle
	if called(pGetWindowRect, h, uintptr(unsafe.Pointer(&rect))) == 0 {
		return 1
	}
	if rect.Width() < 90 || rect.Height() < 50 {
		return 1
	}
	// Do not touch borderless fullscreen windows (including games).
	style := called(pGetWindowLong, h, gwlStyle)
	if style&wsCaption == 0 && rect.Width() >= info.Monitor.Width()-2 && rect.Height() >= info.Monitor.Height()-2 {
		return 1
	}
	order, ok := c.order[h]
	if !ok {
		c.nextOrder++
		order = c.nextOrder
		c.order[h] = order
	}
	c.groups[monitor] = append(c.groups[monitor], windowItem{h, monitor, order})
	return 1
}

func (c *controller) retile() {
	if c.paused || c.reflowing {
		return
	}
	c.reflowing = true
	defer func() { c.reflowing = false }()
	next := make(map[uintptr]move)
	for monitor, items := range c.candidates() {
		info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		if called(pGetMonitorInfo, monitor, uintptr(unsafe.Pointer(&info))) == 0 {
			continue
		}
		sort.Slice(items, func(i, j int) bool { return items[i].order < items[j].order })
		tiles := masterTiles(info.Work, len(items), c.profile.Gap, c.profile.MasterPercent)
		if len(tiles) != len(items) {
			continue
		}
		for i, item := range items {
			h := item.hwnd
			if !isWindow(h) {
				continue
			}
			dst := tiles[i]
			var actual Rectangle
			if called(pGetWindowRect, h, uintptr(unsafe.Pointer(&actual))) == 0 {
				continue
			}
			if actual == dst && called(pIsZoomed, h) == 0 {
				continue
			}
			if _, ok := c.original[h]; !ok {
				placement := windowPlacement{Length: uint32(unsafe.Sizeof(windowPlacement{}))}
				if called(pGetWindowPlacement, h, uintptr(unsafe.Pointer(&placement))) != 0 {
					c.original[h] = placement
				}
			}
			if called(pIsZoomed, h) != 0 {
				called(pShowWindow, h, swRestore)
			}
			if old, ok := c.moves[h]; ok && old.to == dst {
				next[h] = old
			} else {
				next[h] = move{actual, dst, time.Now()}
			}
		}
	}
	c.moves = next
	c.animate()
	if len(c.moves) > 0 {
		called(pSetTimer, c.window, 2, 16, 0)
	}
}

func (c *controller) restoreAll() {
	clear(c.moves)
	called(pKillTimer, c.window, 2)
	for h, p := range c.original {
		if isWindow(h) {
			called(pSetWindowPlacement, h, uintptr(unsafe.Pointer(&p)))
		}
	}
	clear(c.original)
	clear(c.order)
}

// Build a native HICON from an embedded transparent PNG, so the tray icon
// works without external files or a separate installer.
func makeIcon() uintptr {
	decoded, err := png.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		return 0
	}
	const size = 128
	bmi := bitmapInfo{Header: bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: size, Height: -size, Planes: 1, BitCount: 32, Compression: biRGB}}
	var bits uintptr
	color := called(pCreateDIBSection, 0, uintptr(unsafe.Pointer(&bmi)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if color == 0 || bits == 0 {
		return 0
	}
	pixels := unsafe.Slice((*byte)(unsafe.Pointer(bits)), size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r, g, b, a := decoded.At(x, y).RGBA()
			o := (y*size + x) * 4
			pixels[o] = byte(b >> 8)
			pixels[o+1] = byte(g >> 8)
			pixels[o+2] = byte(r >> 8)
			pixels[o+3] = byte(a >> 8)
		}
	}
	maskData := make([]byte, size*size/8) // Alpha channel controls transparency.
	mask := called(pCreateBitmap, size, size, 1, 1, uintptr(unsafe.Pointer(&maskData[0])))
	if mask == 0 {
		called(pDeleteObject, color)
		return 0
	}
	info := iconInfo{IsIcon: 1, Mask: mask, Color: color}
	icon := called(pCreateIconIndirect, uintptr(unsafe.Pointer(&info)))
	called(pDeleteObject, color)
	called(pDeleteObject, mask)
	return icon
}
