package main

import "testing"

func TestLayoutSingle(t *testing.T) {
	got := calculateTiles(Rectangle{0, 0, 1920, 1080}, 1, 12)
	if len(got) != 1 || got[0] != (Rectangle{12, 12, 1908, 1068}) {
		t.Fatalf("unexpected: %+v", got)
	}
}
func TestLayoutStack(t *testing.T) {
	work := Rectangle{0, 0, 1920, 1080}
	for n := 2; n <= 10; n++ {
		tiles := calculateTiles(work, n, 12)
		if len(tiles) != n {
			t.Fatalf("count %d returned %d", n, len(tiles))
		}
		for i, r := range tiles {
			if r.Width() <= 0 || r.Height() <= 0 || r.Left < work.Left || r.Right > work.Right || r.Bottom > work.Bottom || r.Top < work.Top {
				t.Fatalf("invalid rect n=%d i=%d: %+v", n, i, r)
			}
		}
		for i := 2; i < n; i++ {
			if tiles[i].Top < tiles[i-1].Bottom {
				t.Fatalf("overlap %d %d", i-1, i)
			}
		}
	}
}
func TestLayoutEmptyAndSmall(t *testing.T) {
	if calculateTiles(Rectangle{0, 0, 1920, 1080}, 0, 12) != nil {
		t.Fatal("empty layout should be nil")
	}
	if calculateTiles(Rectangle{0, 0, 50, 50}, 3, 12) != nil {
		t.Fatal("small area should be nil")
	}
}