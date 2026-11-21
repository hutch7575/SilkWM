package main

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type finderUI struct {
	closing                time.Time
	closeAlpha             int
	host, edit, previous   uintptr
	visible                bool
	tap                    winTap
	query                  string
	apps, results          []appEntry
	selected, first        int
	opened, typed, indexed time.Time
	updates                chan []appEntry
	loading                bool
	field                  silkField
}

func (f *finderUI) start() {
	inst := called(pGetModuleHandle, 0)
	name := ptr("SilkFinder")
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), WndProc: syscall.NewCallback(finderProc), Instance: inst, ClassName: name, Cursor: called(user32.NewProc("LoadCursorW"), 0, 32512)}
	if called(pRegisterClassEx, uintptr(unsafe.Pointer(&wc))) == 0 {
		return
	}
	f.host = called(pCreateWindowEx, wsExToolWindow|0x80008, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(ptr("Silk App Finder"))), 0x80000000, 0, 0, 0, 0, 0, 0, inst, 0)
	if f.host == 0 {
		return
	}
	// The focused native edit handles text services, keyboard layouts and paste.
	// Its value is typeset at display size by the host.
	f.edit = called(pCreateWindowEx, 0, uintptr(unsafe.Pointer(ptr("EDIT"))), 0, 0x50000080, 0, 0, 1, 1, f.host, 1, inst, 0)
	if f.edit == 0 {
		called(pDestroyWindow, f.host)
		f.host = 0
		return
	}
	called(user32.NewProc("SendMessageW"), f.edit, 0xc5, 160, 0)
	called(user32.NewProc("RegisterHotKey"), app.window, 10, 0x4003, 0x20)
	if app.profile.WinFinder {
		f.tap.start()
	}
	f.refresh()
}
func (f *finderUI) stop() {
	f.tap.stop()
	called(user32.NewProc("UnregisterHotKey"), app.window, 10)
	if f.host != 0 {
		called(pDestroyWindow, f.host)
	}
}
func (f *finderUI) refresh() {
	if f.loading || time.Since(f.indexed) < 5*time.Minute {
		return
	}
	f.loading = true
	f.updates = make(chan []appEntry, 2)
	go func(ch chan []appEntry, notify uintptr) {
		links := shortcuts()
		ch <- links
		called(pPostMessage, notify, catalogueReady, 0, 0)
		rows := startApps(links)
		ch <- rows
		called(pPostMessage, notify, catalogueReady, 1, 0)
	}(f.updates, app.window)
}
func (f *finderUI) received(done bool) {
	select {
	case f.apps = <-f.updates:
	default:
		return
	}
	if done {
		f.loading = false
		f.indexed = time.Now()
	}
	f.filter()
}
func (f *finderUI) filter() {
	f.results = matches(f.apps, f.query)
	f.selected = 0
	f.first = 0
	if f.visible {
		called(invalidate, f.host, 0, 0)
	}
}
func (f *finderUI) open() {
	if f.host == 0 {
		return
	}
	if f.visible {
		f.dismiss()
		return
	}
	f.previous = called(user32.NewProc("GetForegroundWindow"))
	monitor := called(pMonitorFromWindow, f.previous, 1)
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if called(pGetMonitorInfo, monitor, uintptr(unsafe.Pointer(&info))) == 0 {
		return
	}
	r := info.Monitor
	f.visible = true
	f.closing = time.Time{}
	f.query = ""
	f.opened = time.Now()
	f.typed = f.opened
	called(user32.NewProc("SetWindowTextW"), f.edit, uintptr(unsafe.Pointer(ptr(""))))
	f.filter()
	called(setAlpha, f.host, 0, uintptr(f.alpha()), 2)
	called(pSetWindowPos, f.host, ^uintptr(0), uintptr(r.Left), uintptr(r.Top), uintptr(r.Width()), uintptr(r.Height()), 0x40)
	called(pSetForegroundWindow, f.host)
	called(user32.NewProc("SetFocus"), f.edit)
	called(pSetWindowPos, f.edit, 0, uintptr(r.Width()/2), uintptr(r.Height()/2), 1, 1, swpNoZOrder|swpNoActivate)
	f.refresh()
	f.wake()
}
func (f *finderUI) alpha() int {
	if !f.closing.IsZero() {
		return int(float64(f.closeAlpha) * (1 - progress(time.Since(f.closing), app.surfaces.duration(120))))
	}
	t := progress(time.Since(f.opened), app.surfaces.duration(150))
	return int(234 * t)
}
func (f *finderUI) wake() { called(invalidate, f.host, 0, 0); called(pSetTimer, f.host, 1, 16, 0) }
func (f *finderUI) dismiss() {
	if !f.visible || !f.closing.IsZero() {
		return
	}
	if app.surfaces.reduceMotion {
		f.close(true)
		return
	}
	f.closeAlpha = f.alpha()
	f.closing = time.Now()
	f.wake()
}
func (f *finderUI) close(restore bool) {
	if !f.visible {
		return
	}
	f.visible = false
	called(pKillTimer, f.host, 1)
	called(pShowWindow, f.host, 0)
	if restore && isWindow(f.previous) {
		called(pSetForegroundWindow, f.previous)
	}
}
func (f *finderUI) launch() {
	if f.selected < 0 || f.selected >= len(f.results) {
		return
	}
	entry := f.results[f.selected]
	f.close(true)
	target := entry.Target
	if !entry.Shortcut {
		target = "shell:AppsFolder\\" + target
	}
	r := called(shell32.NewProc("ShellExecuteW"), 0, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(target))), 0, 0, 1)
	if r <= 32 {
		alert(fmt.Sprintf("Windows could not open %s (error %d).", entry.Name, r))
	}
}
func (f *finderUI) key(m msg) bool {
	if !f.visible || m.Hwnd != f.edit || m.Message != 0x100 {
		return false
	}
	if !f.closing.IsZero() {
		return true
	}
	switch m.Wparam {
	case 13, 27, 38, 40:
	default:
		return false
	}
	// Leave the IME's composition and candidate navigation to the native edit.
	imm := syscall.NewLazyDLL("imm32.dll")
	ctx := called(imm.NewProc("ImmGetContext"), f.edit)
	if ctx != 0 {
		length := int32(called(imm.NewProc("ImmGetCompositionStringW"), ctx, 8, 0, 0))
		called(imm.NewProc("ImmReleaseContext"), f.edit, ctx)
		if length > 0 {
			return false
		}
	}
	switch m.Wparam {
	case 27:
		f.dismiss()
	case 13:
		f.launch()
	case 38, 40:
		if len(f.results) > 0 {
			step := 1
			if m.Wparam == 38 {
				step = -1
			}
			f.selected = (f.selected + step + len(f.results)) % len(f.results)
			called(invalidate, f.host, 0, 0)
		}
	default:
		return false
	}
	return true
}
func finderProc(h, msg, w, l uintptr) uintptr {
	f := &app.finder
	switch msg {
	case 0x1c:
		if w == 0 {
			f.close(false)
		}
		return 0
	case 0x111:
		if l == f.edit && w>>16 == 0x300 {
			var b [162]uint16
			called(user32.NewProc("GetWindowTextW"), f.edit, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
			f.query = syscall.UTF16ToString(b[:])
			f.typed = time.Now()
			f.filter()
			if f.visible {
				f.wake()
			}
		}
		return 0
	case wmTimer:
		if !f.closing.IsZero() {
			a := f.alpha()
			if a == 0 {
				f.close(true)
			} else {
				called(setAlpha, h, 0, uintptr(a), 2)
			}
			return 0
		}
		called(setAlpha, h, 0, uintptr(f.alpha()), 2)
		called(invalidate, h, 0, 0)
		if time.Since(f.typed) >= time.Duration(app.surfaces.duration(220))*time.Millisecond && time.Since(f.opened) >= time.Duration(app.surfaces.duration(200))*time.Millisecond {
			called(pKillTimer, h, 1)
		}
		return 0
	case wmLButtonUp:
		var r Rectangle
		called(user32.NewProc("GetClientRect"), h, uintptr(unsafe.Pointer(&r)))
		b, rows := f.geometry(r)
		x, y := int32(int16(l)), int32(int16(l>>16))
		if x >= b.Left && x < b.Right && y >= b.Top && y < b.Top+int32(rows)*b.Height() {
			idx := f.first + int((y-b.Top)/b.Height())
			if idx < len(f.results) {
				f.selected = idx
				f.launch()
				return 0
			}
		}
		f.close(true)
		return 0
	case 0x14:
		return 1
	case 0xf:
		var ps paintStruct
		dc := called(user32.NewProc("BeginPaint"), h, uintptr(unsafe.Pointer(&ps)))
		var r Rectangle
		called(user32.NewProc("GetClientRect"), h, uintptr(unsafe.Pointer(&r)))
		buffered(dc, r, f.paint)
		called(user32.NewProc("EndPaint"), h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmClose:
		f.close(true)
		return 0
	}
	return called(pDefWindowProc, h, msg, w, l)
}
func (f *finderUI) geometry(r Rectangle) (Rectangle, int) {
	sc := func(n int32) int32 { return scale(f.host, n) }
	w := min(r.Width()-sc(48), sc(490))
	cx := r.Width() / 2
	top := r.Height()/2 + sc(52)
	rh := max(int32(18), min(sc(40), r.Height()/13))
	rows := max(1, min(5, int((r.Bottom-top-sc(40))/rh)))
	return Rectangle{cx - w/2, top, cx + (w+1)/2, top + rh}, rows
}
func (f *finderUI) paint(dc uintptr, r Rectangle) {
	p := colors()
	bg := blend(p.surface, 0, .73)
	phase := progress(time.Since(f.opened), app.surfaces.duration(200))
	f.field.draw(dc, r, bg, blend(p.accent, p.surface, .27), 1-phase)
	sc := func(n int32) int32 { return scale(f.host, n) }
	cx, cy := r.Width()/2, r.Height()/2
	sz := min(sc(28), r.Height()/13)
	ribbon(dc, cx-sz/2, cy-sc(106), sz, p.accent)
	t := progress(time.Since(f.typed), app.surfaces.duration(210))
	shift := int32((1 - t) * float64(sc(15)))
	title := f.query
	size := sc(78)
	ink := p.text
	if title == "" {
		title = "Find an app"
		size = sc(42)
		ink = blend(p.text, p.accent, .22)
	}
	if len([]rune(title)) > 14 {
		size = sc(42)
	}
	titleRect := Rectangle{sc(18), cy - sc(64) + shift, r.Right - sc(18), cy + sc(36) + shift}
	label(dc, titleRect, title, "Segoe UI Light", min(size, r.Height()/7), 300, blend(ink, p.accent, (1-t)*.6), 1)
	b, rows := f.geometry(r)
	f.first = min(f.first, f.selected)
	f.first = max(f.first, f.selected-rows+1)
	if len(f.results) == 0 && strings.TrimSpace(f.query) != "" {
		s := "No matching apps"
		if f.loading {
			s = "Finding installed apps…"
		}
		label(dc, b, s, "Segoe UI", sc(14), 400, p.muted, 1)
	}
	for idx := f.first; idx < len(f.results) && idx < f.first+rows; idx++ {
		item := b
		ink := p.muted
		if idx == f.selected {
			gradient(dc, item, blend(bg, p.accent, .23), blend(bg, p.accent, .045), false)
			fill(dc, Rectangle{item.Left, item.Top + sc(9), item.Left + sc(2), item.Bottom - sc(9)}, p.accent)
			label(dc, Rectangle{item.Right - sc(55), item.Top, item.Right - sc(12), item.Bottom}, "↵", "Segoe UI", sc(16), 400, p.accent, 2)
			ink = p.text
		}
		item.Left += sc(17)
		item.Right -= sc(50)
		label(dc, item, f.results[idx].Name, "Segoe UI", sc(16), 400, ink, 0)
		b.Top += b.Height()
		b.Bottom = b.Top + item.Height()
	}
	hint := "Type to search     ·     Esc to close"
	if len(f.results) > 0 {
		hint = "↑ ↓   Select       ↵   Open       Esc   Close"
	}
	label(dc, Rectangle{0, r.Bottom - sc(42), r.Right, r.Bottom - sc(12)}, hint, "Segoe UI", sc(12), 400, p.muted, 1)
}
