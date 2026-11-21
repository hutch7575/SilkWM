package main

import (
	"time"
	"unsafe"
)

type palette struct{ surface, text, accent, muted, edge, selected uintptr }

func colors() palette {
	p := app.profile
	base, _ := colorRef(p.Surface)
	fg, _ := colorRef(p.Text)
	accent, _ := colorRef(p.Transition)
	if p.Accent != "auto" {
		accent, _ = colorRef(p.Accent)
	} else {
		accent = app.surfaces.wallpaper.colorAt(accent)
	}
	bg := blend(base, accent, .07)
	accent = readable(accent, bg)
	fg = readable(fg, bg)
	return palette{bg, fg, accent, readable(blend(fg, bg, .4), bg), blend(bg, fg, .14), blend(bg, accent, .16)}
}

var getDPI = user32.NewProc("GetDpiForWindow")

func scale(h uintptr, n int32) int32 {
	dpi := uintptr(96)
	if h != 0 && getDPI.Find() == nil {
		if d := called(getDPI, h); d != 0 {
			dpi = d
		}
	}
	return (n*int32(dpi) + 48) / 96
}
func solid(c uintptr) uintptr { return called(gdi32.NewProc("CreateSolidBrush"), c) }
func fill(dc uintptr, r Rectangle, c uintptr) {
	b := solid(c)
	called(user32.NewProc("FillRect"), dc, uintptr(unsafe.Pointer(&r)), b)
	called(pDeleteObject, b)
}
func rounded(dc uintptr, r Rectangle, diam int32, c uintptr) {
	brush := solid(c)
	sel := gdi32.NewProc("SelectObject")
	oldBrush := called(sel, dc, brush)
	oldPen := called(sel, dc, called(gdi32.NewProc("GetStockObject"), 8))
	called(gdi32.NewProc("RoundRect"), dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right+1), uintptr(r.Bottom+1), uintptr(diam), uintptr(diam))
	called(sel, dc, oldPen)
	called(sel, dc, oldBrush)
	called(pDeleteObject, brush)
}
func label(dc uintptr, r Rectangle, s, family string, height int32, weight int, ink, align uintptr) {
	if r.Width() <= 0 || r.Height() <= 0 {
		return
	}
	face := called(gdi32.NewProc("CreateFontW"), uintptr(-height), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(ptr(family))))
	old := called(gdi32.NewProc("SelectObject"), dc, face)
	called(gdi32.NewProc("SetTextColor"), dc, ink)
	called(gdi32.NewProc("SetBkMode"), dc, 1)
	called(user32.NewProc("DrawTextW"), dc, uintptr(unsafe.Pointer(ptr(s))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), align|0x8824)
	called(gdi32.NewProc("SelectObject"), dc, old)
	called(pDeleteObject, face)
}

// Two tapering curves share a diagonal seam to make the Silk ribbon.
func ribbon(dc uintptr, x, y, size int32, c uintptr) {
	transform := func(a, b int32) point { return point{x + a*size/100, y + b*size/100} }
	curves := [][]int32{
		{86, 7, 74, 26, 51, 30, 34, 43, 21, 54, 27, 60, 43, 66, 16, 64, 5, 55, 11, 40, 21, 23, 57, 17, 86, 7},
		{16, 93, 28, 74, 51, 70, 68, 57, 81, 46, 75, 40, 59, 34, 86, 36, 97, 45, 91, 60, 81, 77, 45, 83, 16, 93}}
	for i, curve := range curves {
		ink := c
		if i == 1 {
			ink = blend(c, 0xffffff, .23)
		}
		b := solid(ink)
		old := called(gdi32.NewProc("SelectObject"), dc, b)
		called(gdi32.NewProc("BeginPath"), dc)
		start := transform(curve[0], curve[1])
		called(gdi32.NewProc("MoveToEx"), dc, uintptr(start.X), uintptr(start.Y), 0)
		pts := make([]point, 0, 12)
		for j := 2; j < len(curve); j += 2 {
			pts = append(pts, transform(curve[j], curve[j+1]))
		}
		called(gdi32.NewProc("PolyBezierTo"), dc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
		called(gdi32.NewProc("CloseFigure"), dc)
		called(gdi32.NewProc("EndPath"), dc)
		called(gdi32.NewProc("FillPath"), dc)
		called(gdi32.NewProc("SelectObject"), dc, old)
		called(pDeleteObject, b)
	}
}
func paintBufferedBar(dc uintptr, r Rectangle) {
	if r.Width() <= 0 || r.Height() <= 0 {
		return
	}
	mem := called(gdi32.NewProc("CreateCompatibleDC"), dc)
	bitmap := called(gdi32.NewProc("CreateCompatibleBitmap"), dc, uintptr(r.Width()), uintptr(r.Height()))
	if mem == 0 || bitmap == 0 {
		paintBar(dc, r)
		if mem != 0 {
			called(gdi32.NewProc("DeleteDC"), mem)
		}
		if bitmap != 0 {
			called(pDeleteObject, bitmap)
		}
		return
	}
	old := called(gdi32.NewProc("SelectObject"), mem, bitmap)
	paintBar(mem, r)
	called(gdi32.NewProc("BitBlt"), dc, 0, 0, uintptr(r.Width()), uintptr(r.Height()), mem, 0, 0, 0x00CC0020)
	called(gdi32.NewProc("SelectObject"), mem, old)
	called(pDeleteObject, bitmap)
	called(gdi32.NewProc("DeleteDC"), mem)
}
func paintBar(dc uintptr, r Rectangle) {
	p := colors()
	fill(dc, r, p.surface)
	fill(dc, Rectangle{0, r.Bottom - 1, r.Right, r.Bottom}, p.edge)
	sc := func(n int32) int32 { return scale(app.surfaces.bar, n) }
	cy := r.Height() / 2
	x := sc(16)
	now := time.Now()
	font := min(sc(14), max(1, r.Height()-2))
	face := "Segoe UI"
	widgets := app.profile.Widgets
	// Place a trailing date/clock group on the centre line; keep custom ordering.
	split := len(widgets)
	for split > 0 && (widgets[split-1] == "clock" || widgets[split-1] == "date") {
		split--
	}
	width := func(k string) int32 {
		switch k {
		case "silk":
			return sc(73)
		case "status":
			return sc(87)
		case "date":
			return sc(92)
		default:
			return sc(49)
		}
	}
	var centreWidth int32
	for _, k := range widgets[split:] {
		centreWidth += width(k)
	}
	for i, k := range widgets {
		if i == split {
			x = max(x, (r.Width()-centreWidth)/2)
		}
		w := width(k)
		if x+w > r.Right-sc(46) {
			break
		}
		rect := Rectangle{x, 0, x + w, r.Bottom}
		switch k {
		case "silk":
			sz := min(sc(22), r.Height())
			ribbon(dc, x, cy-sz/2, sz, p.accent)
			rect.Left += sc(27)
			label(dc, rect, "Silk", face, min(font+sc(2), r.Height()), 600, p.text, 0)
		case "status":
			txt := "Tiling"
			color := p.accent
			if app.paused {
				txt = "Paused"
				color = p.muted
			}
			h := min(sc(23), r.Height())
			chip := Rectangle{x, cy - h/2, x + sc(71), cy + (h+1)/2}
			rounded(dc, chip, sc(12), blend(p.surface, color, .12))
			dot := sc(5)
			rounded(dc, Rectangle{x + sc(9), cy - dot/2, x + sc(9) + dot, cy + (dot+1)/2}, dot, color)
			rect.Left += sc(21)
			label(dc, rect, txt, face, min(font, sc(12)), 500, readable(color, blend(p.surface, color, .12)), 0)
		case "clock":
			label(dc, rect, now.Format("15:04"), face, font, 600, p.text, 0)
		case "date":
			label(dc, rect, now.Format("Mon, 02 Jan"), face, font, 400, p.muted, 0)
		}
		x += w
	}
	label(dc, Rectangle{r.Right - sc(51), 0, r.Right - sc(10), r.Bottom}, "\ue713", "Segoe MDL2 Assets", min(sc(16), r.Height()), 400, p.muted, 1)
}
func paintTransition(dc uintptr, r Rectangle, restarting bool) {
	p := colors()
	base, _ := colorRef(app.profile.Transition)
	// Preserve the profile's blue reveal with a subtle tint from the current wallpaper.
	color := blend(base, p.accent, .18)
	fill(dc, r, blend(color, 0, .58))
	size := scale(app.surfaces.bar, 60)
	cx, cy := r.Width()/2, r.Height()/2
	ribbon(dc, cx-size/2, cy-size, size, blend(color, 0xffffff, .8))
	name := "Silk"
	if restarting {
		name = "Restarting…"
	}
	label(dc, Rectangle{0, cy + size/6, r.Right, cy + size}, name, "Segoe UI", size/2, 500, 0xffffff, 1)
}

type menuRow struct {
	id          int
	icon, title string
	group       bool
}

func menuRows(paused bool) []menuRow {
	state, icon := "Pause tiling", "\ue769"
	if paused {
		state, icon = "Resume tiling", "\ue768"
	}
	return []menuRow{
		{menuPause, icon, state, false}, {menuArrange, "\ue8a9", "Arrange windows", false},
		{103, "\ue771", "Edit profile…", true}, {104, "\ue72c", "Reload profile", false},
		{107, "\ue91b", "Use Silk wallpaper", false}, {105, "\ue7b3", "Preview transition", false},
		{106, "\ue777", "Restart Windows…", true}, {menuExit, "\ue8bb", "Exit Silk", false}}
}

type menuItemInfo struct {
	Size, Mask, Type, State, ID       uint32
	Submenu, Checked, Unchecked, Data uintptr
	Text                              *uint16
	Length                            uint32
	Bitmap                            uintptr
}
type menuInfo struct {
	Size, Mask, Style, MaxHeight uint32
	Background                   uintptr
	Help                         uint32
	Data                         uintptr
}
type measureItem struct {
	Type, ID, Item, Width, Height uint32
	Data                          uintptr
}
type drawItem struct {
	Type, ID, Item, Action, State uint32
	Window, DC                    uintptr
	Rect                          Rectangle
	Data                          uintptr
}

func rowFor(id uintptr) (menuRow, bool) {
	for _, row := range menuRows(app.paused) {
		if uintptr(row.id) == id {
			return row, true
		}
	}
	return menuRow{}, false
}
func measureMenu(lp uintptr) uintptr {
	if lp == 0 {
		return 0
	}
	m := (*measureItem)(unsafe.Pointer(lp))
	row, ok := rowFor(m.Data)
	if !ok || m.Type != 1 {
		return 0
	}
	h := int32(35)
	if row.group {
		h += 9
	}
	m.Width = uint32(scale(app.surfaces.bar, 215))
	m.Height = uint32(scale(app.surfaces.bar, h))
	return 1
}
func paintMenu(lp uintptr) uintptr {
	if lp == 0 {
		return 0
	}
	d := (*drawItem)(unsafe.Pointer(lp))
	row, ok := rowFor(d.Data)
	if !ok || d.Type != 1 {
		return 0
	}
	paintMenuRow(d.DC, d.Rect, row, d.State&1 != 0)
	return 1
}
func paintMenuRow(dc uintptr, r Rectangle, row menuRow, selected bool) {
	p := colors()
	sc := func(n int32) int32 { return scale(app.surfaces.bar, n) }
	fill(dc, r, p.surface)
	if row.group {
		y := r.Top + sc(4)
		fill(dc, Rectangle{r.Left + sc(12), y, r.Right - sc(12), y + 1}, p.edge)
		r.Top += sc(9)
	}
	bg := p.surface
	if selected {
		bg = p.selected
		rounded(dc, Rectangle{r.Left + sc(4), r.Top + sc(2), r.Right - sc(4), r.Bottom - sc(2)}, sc(10), bg)
	}
	color := p.muted
	if selected {
		color = p.accent
	}
	label(dc, Rectangle{r.Left + sc(10), r.Top, r.Left + sc(39), r.Bottom}, row.icon, "Segoe MDL2 Assets", sc(15), 400, readable(color, bg), 1)
	r.Left += sc(44)
	label(dc, r, row.title, "Segoe UI", sc(14), 400, readable(p.text, bg), 0)
}
