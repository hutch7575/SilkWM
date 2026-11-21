package main

import (
	"golang.org/x/image/draw"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

// Render the same Win32 paint routines used by the bar and popup into a DIB.
func TestDesktopPaint(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	src, e := os.Open("assets/midnight-silk.jpg")
	if e != nil {
		t.Fatal(e)
	}
	wall, e := jpeg.Decode(src)
	src.Close()
	if e != nil {
		t.Fatal(e)
	}
	oldApp := app
	defer func() { app = oldApp }()
	app = &controller{profile: defaults()}
	col, ok := wallpaperColor(wall)
	app.surfaces.wallpaper.last = wallSample{color: col, valid: ok}
	const w, h = 1440, 810
	preview := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(preview, preview.Bounds(), wall, wall.Bounds(), draw.Src, nil)
	dc := called(gdi32.NewProc("CreateCompatibleDC"), 0)
	defer called(gdi32.NewProc("DeleteDC"), dc)
	bmi := bitmapInfo{Header: bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: w, Height: -h, Planes: 1, BitCount: 32}}
	var bits unsafe.Pointer
	bm := called(pCreateDIBSection, dc, uintptr(unsafe.Pointer(&bmi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bm == 0 || bits == nil {
		t.Fatal("DIB allocation failed")
	}
	defer called(pDeleteObject, bm)
	selectObj := gdi32.NewProc("SelectObject")
	old := called(selectObj, dc, bm)
	defer called(selectObj, dc, old)
	px := unsafe.Slice((*byte)(bits), w*h*4)
	for i := 0; i < len(px); i += 4 {
		px[i], px[i+1], px[i+2] = preview.Pix[i+2], preview.Pix[i+1], preview.Pix[i]
	}
	paintBar(dc, Rectangle{0, 0, w, 40})
	y := int32(49)
	for _, row := range menuRows(false) {
		rh := int32(35)
		if row.group {
			rh += 9
		}
		paintMenuRow(dc, Rectangle{12, y, 251, y + rh}, row, row.id == 107)
		y += rh
	}
	called(gdi32.NewProc("GdiFlush"))
	for i := 0; i < len(px); i += 4 {
		preview.Pix[i], preview.Pix[i+1], preview.Pix[i+2] = px[i+2], px[i+1], px[i]
	}
	ink := map[uint32]bool{}
	for y := 8; y < 32; y++ {
		for x := 16; x < 74; x++ {
			r, g, b, _ := preview.At(x, y).RGBA()
			ink[(r>>8)<<16|(g>>8)<<8|b>>8] = true
		}
	}
	if len(ink) < 4 {
		t.Fatal("bar branding did not render")
	}
	if path := os.Getenv("SILK_RENDER_PREVIEW"); path != "" {
		f, e := os.Create(path)
		if e != nil {
			t.Fatal(e)
		}
		e = jpeg.Encode(f, preview, &jpeg.Options{Quality: 93})
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
		// Render a typed query through the same painter as the live overlay.
		fnd := &app.finder
		fnd.query = "v"
		fnd.opened = time.Now().Add(-time.Second)
		fnd.typed = fnd.opened
		fnd.results = matches([]appEntry{{"Visual Studio Code", "1", false}, {"VLC media player", "2", false}, {"Voice Recorder", "3", false}}, fnd.query)
		fnd.paint(dc, Rectangle{0, 0, w, h})
		called(gdi32.NewProc("GdiFlush"))
		for i := 0; i < len(px); i += 4 {
			preview.Pix[i], preview.Pix[i+1], preview.Pix[i+2] = px[i+2], px[i+1], px[i]
		}
		out, e := os.Create(filepath.Join(filepath.Dir(path), "Silk-finder.jpg"))
		if e != nil {
			t.Fatal(e)
		}
		e = jpeg.Encode(out, preview, &jpeg.Options{Quality: 94})
		out.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
}
