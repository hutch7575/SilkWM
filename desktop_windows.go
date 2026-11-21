package main

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"syscall"
	"time"
	"unsafe"
)

type move struct {
	from, to Rectangle
	start    time.Time
}

func (c *controller) animate() {
	now := time.Now()
	for h, m := range c.moves {
		if !isWindow(h) || called(pIsIconic, h) != 0 {
			delete(c.moves, h)
			continue
		}
		t := progress(now.Sub(m.start), c.surfaces.duration(c.profile.MotionMS))
		r := interpolate(m.from, m.to, t)
		called(pSetWindowPos, h, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Width()), uintptr(r.Height()), swpNoZOrder|swpNoActivate|swpNoOwnerZOrder|0x4000)
		if t >= 1 {
			delete(c.moves, h)
		}
	}
	if len(c.moves) == 0 {
		called(pKillTimer, c.window, 2)
	}
}

type appBarData struct {
	Size           uint32
	Window         uintptr
	Callback, Edge uint32
	Rect           Rectangle
	Param          uintptr
}
type paintStruct struct {
	DC              uintptr
	Erase           int32
	Rect            Rectangle
	Restore, Update int32
	Reserved        [32]byte
}
type desktopUI struct {
	field                           silkField
	reduceMotion                    bool
	wallpaper                       wallpaperWatcher
	reserved                        Rectangle
	bar, overlay                    uintptr
	registered, placing, restarting bool
	startTime                       time.Time
}

var (
	barMessage = shell32.NewProc("SHAppBarMessage")
	setAlpha   = user32.NewProc("SetLayeredWindowAttributes")
	invalidate = user32.NewProc("InvalidateRect")
	wts        = syscall.NewLazyDLL("wtsapi32.dll")
)

const wmBar = wmApp + 5

func alert(msg string) {
	called(pMessageBox, 0, uintptr(unsafe.Pointer(ptr(msg))), uintptr(unsafe.Pointer(ptr("Silk"))), 0x30)
}
func (u *desktopUI) start(c *controller) {
	inst := called(pGetModuleHandle, 0)
	name := ptr("SilkSurface")
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), WndProc: syscall.NewCallback(surfaceProc), Instance: inst, ClassName: name, Cursor: called(user32.NewProc("LoadCursorW"), 0, 32512)}
	if called(pRegisterClassEx, uintptr(unsafe.Pointer(&wc))) == 0 {
		alert("Could not initialise the Silk desktop surfaces.")
		return
	}
	makeWindow := func(ex uintptr, title string) uintptr {
		return called(pCreateWindowEx, ex|wsExToolWindow|wsExNoActivate, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(ptr(title))), 0x80000000, 0, 0, 0, 0, 0, 0, inst, 0)
	}
	u.bar = makeWindow(8, "Silk bar")
	u.overlay = makeWindow(8|0x80000|0x20, "Silk transition")
	called(wts.NewProc("WTSRegisterSessionNotification"), c.window, 0)
	u.updateMotion()
	u.wallpaper.refresh(true)
	called(pSetTimer, c.window, 4, 1000, 0)
	u.resetBar()
	u.fade(false)
}
func (u *desktopUI) ab(msg uintptr, d *appBarData) uintptr {
	d.Size = uint32(unsafe.Sizeof(*d))
	d.Window = u.bar
	return called(barMessage, msg, uintptr(unsafe.Pointer(d)))
}
func (u *desktopUI) resetBar() {
	u.reserved = Rectangle{}
	if u.bar == 0 {
		return
	}
	if u.registered {
		u.registered = false
		u.ab(1, &appBarData{})
	}
	called(pKillTimer, u.bar, 1)
	if app.profile.BarHeight == 0 {
		called(pShowWindow, u.bar, 0)
		return
	}
	u.registered = u.ab(0, &appBarData{Callback: wmBar}) != 0
	if !u.registered {
		called(pShowWindow, u.bar, 0)
		return
	}
	u.placeBar()
	called(pShowWindow, u.bar, 4)
	called(pSetTimer, u.bar, 1, 1000, 0)
	called(invalidate, u.bar, 0, 0)
}
func (u *desktopUI) placeBar() {
	if !u.registered || u.placing {
		return
	}
	u.placing = true
	defer func() { u.placing = false }()
	monitor := called(user32.NewProc("MonitorFromPoint"), 0, 1)
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if called(pGetMonitorInfo, monitor, uintptr(unsafe.Pointer(&mi))) == 0 {
		return
	}
	d := appBarData{Edge: 1, Rect: mi.Monitor}
	height := scale(u.bar, app.profile.BarHeight)
	d.Rect.Bottom = d.Rect.Top + height
	u.ab(2, &d)
	d.Rect.Bottom = d.Rect.Top + height
	if d.Rect == u.reserved {
		return
	}
	u.reserved = d.Rect
	u.ab(3, &d)
	r := d.Rect
	called(pSetWindowPos, u.bar, ^uintptr(0), uintptr(r.Left), uintptr(r.Top), uintptr(r.Width()), uintptr(r.Height()), swpNoActivate)
}
func (u *desktopUI) stop(c *controller) {
	called(wts.NewProc("WTSUnRegisterSessionNotification"), c.window)
	if u.registered {
		u.registered = false
		u.ab(1, &appBarData{})
	}
	for _, h := range []uintptr{u.overlay, u.bar} {
		if h != 0 {
			called(pDestroyWindow, h)
		}
	}
}
func surfaceProc(h, msg, w, l uintptr) uintptr {
	u := &app.surfaces
	switch msg {
	case 0x02e0: // WM_DPICHANGED
		if h == u.bar {
			u.placeBar()
		}
		return 0
	case wmBar:
		if w == 1 {
			u.placeBar()
		}
		if w == 2 {
			z := ^uintptr(0)
			if l != 0 {
				z = 1
			}
			called(pSetWindowPos, h, z, 0, 0, 0, 0, 0x13)
		}
		return 0
	case wmTimer:
		called(invalidate, h, 0, 0)
		return 0
	case wmLButtonUp, wmRButtonUp:
		if h == u.bar {
			app.showMenu()
		}
		return 0
	case 0x14:
		return 1
	case 0xf:
		var p paintStruct
		dc := called(user32.NewProc("BeginPaint"), h, uintptr(unsafe.Pointer(&p)))
		var r Rectangle
		called(user32.NewProc("GetClientRect"), h, uintptr(unsafe.Pointer(&r)))
		if h == u.bar {
			paintBufferedBar(dc, r)
		} else if h == u.overlay {
			buffered(dc, r, func(dc uintptr, r Rectangle) { paintTransition(dc, r, u.restarting) })
		}
		called(user32.NewProc("EndPaint"), h, uintptr(unsafe.Pointer(&p)))
		return 0
	}
	return called(pDefWindowProc, h, msg, w, l)
}
func (u *desktopUI) fade(restart bool) {
	if u.restarting {
		return
	}
	if u.duration(app.profile.FadeMS) == 0 || u.overlay == 0 {
		if restart {
			restartWindows()
		}
		return
	}
	u.startTime = time.Now()
	u.restarting = restart
	metric := user32.NewProc("GetSystemMetrics")
	x, y, w, h := called(metric, 76), called(metric, 77), called(metric, 78), called(metric, 79)
	alpha := uintptr(255)
	if restart {
		alpha = 0
	}
	called(setAlpha, u.overlay, 0, alpha, 2)
	called(pSetWindowPos, u.overlay, ^uintptr(0), x, y, w, h, 0x50)
	called(invalidate, u.overlay, 0, 0)
	called(pSetTimer, app.window, 3, 16, 0)
}
func (u *desktopUI) frame() {
	t := progress(time.Since(u.startTime), u.duration(app.profile.FadeMS))
	opacity := 1 - t
	if u.restarting {
		opacity = t
	}
	called(setAlpha, u.overlay, 0, uintptr(math.Round(opacity*255)), 2)
	called(invalidate, u.overlay, 0, 0)
	if t >= 1 {
		called(pKillTimer, app.window, 3)
		if u.restarting {
			restartWindows()
			u.restarting = false
		}
		called(pShowWindow, u.overlay, 0)
	}
}
func restartWindows() {
	cmd := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "shutdown.exe"), "/r", "/t", "0")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if e := cmd.Run(); e != nil {
		alert(e.Error())
	}
}

func (u *desktopUI) updateMotion() {
	var enabled int32
	if called(systemParameters, 0x1042, 0, uintptr(unsafe.Pointer(&enabled)), 0) != 0 {
		u.reduceMotion = enabled == 0
	}
}
func (u *desktopUI) duration(ms int) int {
	if u.reduceMotion {
		return 0
	}
	return ms
}
