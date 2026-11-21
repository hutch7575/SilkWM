package main

import (
	"testing"
	"time"
)

func TestProfile(t *testing.T) {
	p, e := parseProfile([]byte(`{"gap":0,"barHeight":0,"motionMs":0,"widgets":[]}`))
	if e != nil || p.Gap != 0 || p.MasterPercent != 60 || p.BarHeight != 0 || p.MotionMS != 0 || len(p.Widgets) != 0 {
		t.Fatal(p, e)
	}
	for _, s := range []string{`{"masterPercent":110}`, `{"widgets":["unknown"]}`, `{"surface":"#zz1111"}`, `{"fadMs":1}`, "{}{}", "null"} {
		if _, e := parseProfile([]byte(s)); e == nil {
			t.Fatal(s)
		}
	}
}
func TestMotionAndRatio(t *testing.T) {
	tiles := masterTiles(Rectangle{-1000, 0, 0, 800}, 2, 0, 70)
	if tiles[0].Right != -300 || tiles[1].Left != -300 {
		t.Fatal(tiles)
	}
	a, b := tiles[0], tiles[1]
	mid := interpolate(a, b, progress(50*time.Millisecond, 100))
	if mid.Left <= a.Left || mid.Left >= b.Left || interpolate(a, b, progress(time.Second, 100)) != b || progress(0, 0) != 1 {
		t.Fatal(mid)
	}
}
