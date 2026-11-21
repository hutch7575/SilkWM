package main

import (
	"image"
	"image/color"
	"testing"
)

func TestWallpaperAccent(t *testing.T) {
	for _, c := range []color.RGBA{{190, 50, 15, 255}, {28, 70, 180, 255}, {128, 128, 128, 255}} {
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		for i := range 100 {
			for j := range 100 {
				pixel := color.RGBA{4, 4, 4, 255}
				if i > 60 {
					pixel = c
				}
				img.SetRGBA(i, j, pixel)
			}
		}
		col, ok := wallpaperColor(img)
		if !ok {
			t.Fatal("no sample")
		}
		if c.R != c.B && (int(col&255) > int(col>>16)) != (c.R > c.B) {
			t.Fatalf("wrong hue for %v: %x", c, col)
		}
		for _, bg := range []uintptr{0, 0x24201b, 0xffffff, 0x7e7e7e} {
			if contrast(readable(col, bg), bg) < 4.5 {
				t.Fatal("low contrast")
			}
		}
	}
	fixed, e := parseProfile([]byte(`{"accent":"#FA7411"}`))
	if e != nil || fixed.Accent != "#FA7411" {
		t.Fatal(e)
	}
	if _, e := parseProfile([]byte(`{"accent":"orange"}`)); e == nil {
		t.Fatal("accepted invalid accent")
	}
}
