package main

import (
	"bufio"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
)

// The desktop field and sign-in still share the same folded-light geometry.
func foldLight(u, v float64) float64 {
	ridge := .83 - .34*u + .105*math.Sin(6.3*u-.45)
	d := v - ridge
	side := math.Pow(math.Abs(u-.5)*2, 1.1)
	upper := v + .13*u - .085
	return min(.94, (.11+.37*side)*math.Exp(-d*d/.014)+(.09+.28*side)*math.Exp(-math.Abs(d)*105)+.12*math.Exp(-upper*upper/.023))
}
func signInArt(path string, accent uintptr) error {
	const w, h = 1920, 1080
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	base := rgb(5, 7, 15)
	for x := 0; x < w; x++ {
		u := float64(x) / w
		for y := 0; y < h; y++ {
			v := float64(y) / h
			l := foldLight(u, v)
			// Leave the central native account and credential controls on a quiet field.
			c := blend(base, accent, min(1.0, l*1.32))
			img.SetRGBA(x, y, color.RGBA{uint8(c), uint8(c >> 8), uint8(c >> 16), 255})
		}
	}
	f, e := os.CreateTemp(filepath.Dir(path), "silk-art-*.jpg")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	b := bufio.NewWriter(f)
	e = jpeg.Encode(b, img, &jpeg.Options{Quality: 94})
	if e == nil {
		e = b.Flush()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
