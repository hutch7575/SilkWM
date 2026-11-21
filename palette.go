package main

import (
	"errors"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// GDI colours are packed as 0x00BBGGRR.
func rgb(r, g, b int) uintptr { return uintptr(r | g<<8 | b<<16) }
func blend(a, b uintptr, t float64) uintptr {
	var out uintptr
	for shift := uint(0); shift < 24; shift += 8 {
		x, y := float64((a>>shift)&255), float64((b>>shift)&255)
		out |= uintptr(math.Round(x+(y-x)*t)) << shift
	}
	return out
}
func luminance(c uintptr) float64 {
	linear := func(x uintptr) float64 {
		v := float64(x&255) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return .2126*linear(c) + .7152*linear(c>>8) + .0722*linear(c>>16)
}
func contrast(a, b uintptr) float64 {
	x, y := luminance(a), luminance(b)
	return (max(x, y) + .05) / (min(x, y) + .05)
}
func readable(fg, bg uintptr) uintptr {
	if contrast(fg, bg) >= 4.5 {
		return fg
	}
	end := uintptr(0xffffff)
	if contrast(0, bg) > contrast(end, bg) {
		end = 0
	}
	for t := 0.1; t < 1; t += 0.1 {
		c := blend(fg, end, t)
		if contrast(c, bg) >= 4.5 {
			return c
		}
	}
	return end
}

// Count hue families, weighting chromatic midtones above shadows and white highlights.
func wallpaperColor(img image.Image) (uintptr, bool) {
	type bucket struct{ weight, r, g, b float64 }
	var hues [24]bucket
	var neutral bucket
	rect := img.Bounds()
	for y := rect.Min.Y; y < rect.Max.Y; y += max(1, rect.Dy()/72) {
		for x := rect.Min.X; x < rect.Max.X; x += max(1, rect.Dx()/96) {
			pr, pg, pb, a := img.At(x, y).RGBA()
			if a < 0x8000 {
				continue
			}
			r, g, b := float64(pr)/float64(a), float64(pg)/float64(a), float64(pb)/float64(a)
			hi, lo := max(r, g, b), min(r, g, b)
			span := hi - lo
			bin := &neutral
			weight := 1.0
			if span > .08 && hi > .075 {
				var hue float64
				switch hi {
				case r:
					hue = (g - b) / span
				case g:
					hue = 2 + (b-r)/span
				default:
					hue = 4 + (r-g)/span
				}
				index := (int(math.Floor(hue*4+.5)) + 24) % 24
				bin = &hues[index]
				weight = span * (.3 + hi)
			}
			bin.weight += weight
			bin.r += r * weight
			bin.g += g * weight
			bin.b += b * weight
		}
	}
	best := bucket{}
	for _, v := range hues {
		if v.weight > best.weight {
			best = v
		}
	}
	if best.weight < max(1.0, neutral.weight*.002) {
		best = neutral
	}
	if best.weight == 0 {
		return 0, false
	}
	return rgb(int(255*best.r/best.weight), int(255*best.g/best.weight), int(255*best.b/best.weight)), true
}
func sampleFile(path string) (uintptr, error) {
	f, e := os.Open(path)
	if e != nil {
		return 0, e
	}
	defer f.Close()
	c, _, e := image.DecodeConfig(f)
	if e != nil {
		return 0, e
	}
	if c.Width <= 0 || c.Height <= 0 || int64(c.Width)*int64(c.Height) > 48000000 {
		return 0, errors.New("unsupported wallpaper dimensions")
	}
	if _, e = f.Seek(0, 0); e != nil {
		return 0, e
	}
	img, _, e := image.Decode(f)
	if e != nil {
		return 0, e
	}
	col, ok := wallpaperColor(img)
	if !ok {
		return 0, errors.New("empty wallpaper")
	}
	return col, nil
}
