package windows

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestCenterOn(t *testing.T) {
	work := win32.Rect{Left: 0, Top: 0, Right: 1000, Bottom: 700}
	tests := []struct {
		name         string
		owner        win32.Rect
		w, h         int32
		wantX, wantY int32
	}{
		{"centered", win32.Rect{Left: 100, Top: 100, Right: 500, Bottom: 400}, 200, 100, 200, 200},
		{"an odd difference rounds down", win32.Rect{Left: 0, Top: 0, Right: 101, Bottom: 51}, 50, 20, 25, 15},
		{"pushed back in at the right and the bottom", win32.Rect{Left: 800, Top: 600, Right: 1000, Bottom: 700}, 300, 200, 700, 500},
		{"pushed back in at the left and the top", win32.Rect{Left: -100, Top: -50, Right: 100, Bottom: 50}, 100, 100, 0, 0},
		{"larger than the work area keeps its corner", win32.Rect{Left: 0, Top: 0, Right: 100, Bottom: 100}, 1200, 900, 0, 0},
		{"a dialog larger than its owner", win32.Rect{Left: 400, Top: 300, Right: 500, Bottom: 350}, 300, 200, 300, 225},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if x, y := centerOn(tt.owner, tt.w, tt.h, work); x != tt.wantX || y != tt.wantY {
				t.Errorf("centerOn = %d,%d, want %d,%d", x, y, tt.wantX, tt.wantY)
			}
		})
	}
	// A work area that does not start at the origin (a monitor to the left).
	left := win32.Rect{Left: -1920, Top: 0, Right: 0, Bottom: 1040}
	if x, y := centerOn(win32.Rect{Left: -1900, Top: 10, Right: -1700, Bottom: 110}, 400, 300, left); x != -1920 || y != 0 {
		t.Errorf("on the left monitor: %d,%d", x, y)
	}
}

func TestCascadeFrom(t *testing.T) {
	work := win32.Rect{Left: 0, Top: 0, Right: 1000, Bottom: 700}
	tests := []struct {
		name         string
		prev         win32.Rect
		w, h         int32
		wantX, wantY int32
	}{
		{"one step down and to the right", win32.Rect{Left: 100, Top: 50, Right: 500, Bottom: 450}, 400, 400, 132, 82},
		{"no room to the right: the cascade starts again", win32.Rect{Left: 600, Top: 0, Right: 1000, Bottom: 300}, 400, 300, 0, 0},
		{"no room below: the cascade starts again", win32.Rect{Left: 0, Top: 400, Right: 400, Bottom: 700}, 400, 300, 0, 0},
		{"fits exactly", win32.Rect{Left: 568, Top: 368, Right: 968, Bottom: 668}, 400, 300, 600, 400},
		{"the previous window is off the work area", win32.Rect{Left: -500, Top: -500, Right: -100, Bottom: -100}, 100, 100, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if x, y := cascadeFrom(tt.prev, 32, tt.w, tt.h, work); x != tt.wantX || y != tt.wantY {
				t.Errorf("cascadeFrom = %d,%d, want %d,%d", x, y, tt.wantX, tt.wantY)
			}
		})
	}
}
