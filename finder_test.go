package main

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestFinderOrder(t *testing.T) {
	entries := []appEntry{{"Studio Code", "1", true}, {"Code", "2", true}, {"Barcode Reader", "3", true}, {"Visual Studio Code", "4", false}}
	cases := map[string][]string{"code": {"2", "1", "4", "3"}, "vis code": {"4"}, "  ": {}, "☃": {}}
	for q, want := range cases {
		found := matches(entries, q)
		if len(found) != len(want) {
			t.Fatalf("%q: %+v", q, found)
		}
		for i, id := range want {
			if found[i].Target != id {
				t.Fatalf("%q: %+v", q, found)
			}
		}
	}
}
func TestSignInArt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signin.jpg")
	if e := signInArt(path, rgb(60, 120, 230)); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	cfg, e := jpeg.DecodeConfig(f)
	if e != nil || cfg.Width != 1920 || cfg.Height != 1080 {
		t.Fatal(cfg, e)
	}
}
