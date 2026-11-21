package main

import (
	"math"
	"time"
)

func progress(elapsed time.Duration, ms int) float64 {
	if ms <= 0 {
		return 1
	}
	t := min(1.0, max(0.0, float64(elapsed)/float64(time.Duration(ms)*time.Millisecond)))
	return 1 - (1-t)*(1-t)*(1-t)
}
func interpolate(a, b Rectangle, t float64) Rectangle {
	mix := func(x, y int32) int32 { return int32(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return Rectangle{mix(a.Left, b.Left), mix(a.Top, b.Top), mix(a.Right, b.Right), mix(a.Bottom, b.Bottom)}
}
