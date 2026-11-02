package main

// Rectangle uses physical screen pixels. All edges are right/bottom exclusive.
type Rectangle struct{ Left, Top, Right, Bottom int32 }

func (r Rectangle) Width() int32  { return r.Right - r.Left }
func (r Rectangle) Height() int32 { return r.Bottom - r.Top }

// calculateTiles implements a deterministic master-and-stack layout:
// one window fills the work area; for two or more, the first gets ~60% on
// the left and the others form an equally spaced stack on the right.
func calculateTiles(work Rectangle, count int, gap int32) []Rectangle {
	if count <= 0 || work.Width() < 80 || work.Height() < 80 {
		return nil
	}
	if gap < 0 {
		gap = 0
	}
	if gap > work.Width()/10 {
		gap = work.Width() / 10
	}
	if gap > work.Height()/10 {
		gap = work.Height() / 10
	}

	left, top := work.Left+gap, work.Top+gap
	right, bottom := work.Right-gap, work.Bottom-gap
	if right <= left || bottom <= top {
		return nil
	}
	if count == 1 {
		return []Rectangle{{left, top, right, bottom}}
	}

	innerWidth := right - left - gap
	if innerWidth < 2 {
		return nil
	}
	masterWidth := (innerWidth * 3) / 5
	if masterWidth < 1 {
		masterWidth = 1
	}
	stackLeft := left + masterWidth + gap
	result := []Rectangle{{left, top, left + masterWidth, bottom}}

	stackCount := count - 1
	availableHeight := bottom - top
	// Prevent too many windows from creating negative-size rectangles.
	stackGap := gap
	if int32(stackCount)*stackGap > availableHeight/2 {
		stackGap = availableHeight / (int32(stackCount) * 2)
	}
	if stackGap < 0 {
		stackGap = 0
	}
	tileHeight := (availableHeight - int32(stackCount-1)*stackGap) / int32(stackCount)
	for i := 0; i < stackCount; i++ {
		y := top + int32(i)*(tileHeight+stackGap)
		yEnd := y + tileHeight
		if i == stackCount-1 {
			yEnd = bottom
		}
		if yEnd <= y {
			return nil
		}
		result = append(result, Rectangle{stackLeft, y, right, yEnd})
	}
	return result
}
