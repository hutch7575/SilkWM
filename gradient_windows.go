package main

import (
	"syscall"
	"unsafe"
)

type gradientVertex struct {
	X, Y       int32
	R, G, B, A uint16
}

var gradientFill = syscall.NewLazyDLL("msimg32.dll").NewProc("GradientFill")

func gradient(dc uintptr, r Rectangle, a, b uintptr, vertical bool) {
	vertex := func(x, y int32, c uintptr) gradientVertex {
		return gradientVertex{x, y, uint16(c&255) * 257, uint16(c>>8&255) * 257, uint16(c>>16&255) * 257, 0xffff}
	}
	v := [2]gradientVertex{vertex(r.Left, r.Top, a), vertex(r.Right, r.Bottom, b)}
	indices := [2]uint32{0, 1}
	mode := uintptr(0)
	if vertical {
		mode = 1
	}
	called(gradientFill, dc, uintptr(unsafe.Pointer(&v[0])), 2, uintptr(unsafe.Pointer(&indices[0])), 1, mode)
}

// A small cached colour field supplies the continuous folds at any resolution.
type silkField struct {
	pixels       []uint32
	accent, base uintptr
}

func (f *silkField) draw(dc uintptr, r Rectangle, base, accent uintptr, offset float64) {
	const w, h = 400, 240
	if len(f.pixels) == 0 || f.accent != accent || f.base != base {
		f.pixels = make([]uint32, w*h)
		f.accent = accent
		f.base = base
		for x := 0; x < w; x++ {
			u := float64(x) / w
			for y := 0; y < h; y++ {
				c := blend(base, accent, foldLight(u, float64(y)/h))
				f.pixels[y*w+x] = uint32(c&255)<<16 | uint32(c&0xff00) | uint32(c>>16)
			}
		}
	}
	// Pan within overscan during entry; no background work while the finder is idle.
	dx := int32(float64(r.Width()) * .035)
	dy := int32(float64(r.Height()) * .035)
	bmi := bitmapInfo{Header: bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: w, Height: -h, Planes: 1, BitCount: 32}}
	called(gdi32.NewProc("SetStretchBltMode"), dc, 4)
	called(gdi32.NewProc("StretchDIBits"), dc, uintptr(r.Left-dx+int32(float64(dx)*offset)), uintptr(r.Top-dy+int32(float64(dy)*offset)), uintptr(r.Width()+2*dx), uintptr(r.Height()+2*dy), 0, 0, w, h, uintptr(unsafe.Pointer(&f.pixels[0])), uintptr(unsafe.Pointer(&bmi)), 0, 0x00cc0020)
}
