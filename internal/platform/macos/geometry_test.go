package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestFlipRectOnTwoDisplays(t *testing.T) {
	// The primary display is 1440 x 900 (the origin of Cocoa), the second is 2560 x 1440 to its right
	// and higher: its bottom left is at (1440, 200), so its top is at 1640 in Cocoa and -740 in ui.
	const primary = 900.0
	tests := []struct {
		name string
		in   cocoaRect
		want ui.Rect
	}{
		{"the primary display", cocoaRect{0, 0, 1440, 900}, ui.NewRectFromLTWH(0, 0, 1440, 900)},
		{"the second display, higher", cocoaRect{1440, 200, 2560, 1440}, ui.NewRectFromLTWH(1440, -740, 2560, 1440)},
		{"a window at the bottom left", cocoaRect{100, 0, 400, 300}, ui.NewRectFromLTWH(100, 600, 400, 300)},
		{"a window below the primary display", cocoaRect{0, -400, 800, 300}, ui.NewRectFromLTWH(0, 1000, 800, 300)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flipRect(tt.in, primary)
			if got != tt.want {
				t.Errorf("flipRect = %v, want %v", got, tt.want)
			}
			if back := unflipRect(got, primary); back != tt.in {
				t.Errorf("unflipRect of the result = %v, want %v", back, tt.in)
			}
		})
	}
	if got := flipPoint(10, 880, primary); got != (ui.Offset{Dx: 10, Dy: 20}) {
		t.Errorf("flipPoint = %v", got)
	}
}

func TestPhysicalAndLogicalSize(t *testing.T) {
	tests := []struct {
		name   string
		points ui.Size
		scale  float64
		want   ui.Size
	}{
		{"scale 1", ui.Size{Width: 800, Height: 600}, 1, ui.Size{Width: 800, Height: 600}},
		{"scale 2", ui.Size{Width: 800, Height: 600}, 2, ui.Size{Width: 1600, Height: 1200}},
		{"scale 1.5 rounds", ui.Size{Width: 801, Height: 601}, 1.5, ui.Size{Width: 1202, Height: 902}},
		{"scale 0 is 1", ui.Size{Width: 10, Height: 20}, 0, ui.Size{Width: 10, Height: 20}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := physicalSize(tt.points, tt.scale); got != tt.want {
				t.Errorf("physicalSize = %v, want %v", got, tt.want)
			}
		})
	}
	if got := logicalSize(ui.Size{Width: 1600, Height: 1200}, 2); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("logicalSize = %v", got)
	}
}
