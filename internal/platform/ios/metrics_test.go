package ios

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestPhysicalSize(t *testing.T) {
	for _, tc := range []struct {
		name   string
		points ui.Size
		scale  float64
		want   ui.Size
	}{
		{"iPhone 17 portrait", ui.Size{Width: 402, Height: 874}, 3, ui.Size{Width: 1206, Height: 2622}},
		{"iPad at 2x", ui.Size{Width: 820, Height: 1180}, 2, ui.Size{Width: 1640, Height: 2360}},
		{"fractional scale rounds", ui.Size{Width: 375.3, Height: 667}, 2.61, ui.Size{Width: 980, Height: 1741}},
		{"no scale is 1", ui.Size{Width: 10, Height: 20}, 0, ui.Size{Width: 10, Height: 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := physicalSize(tc.points, tc.scale); got != tc.want {
				t.Errorf("physicalSize(%v, %v) = %v, want %v", tc.points, tc.scale, got, tc.want)
			}
		})
	}
}

func TestViewPadding(t *testing.T) {
	for _, tc := range []struct {
		name  string
		in    edgeInsets
		scale float64
		want  ui.ViewPadding
	}{
		{"portrait with the island and the indicator", edgeInsets{Top: 62, Bottom: 34}, 3, ui.ViewPadding{Top: 186, Bottom: 102}},
		{"landscape has side insets", edgeInsets{Top: 0, Left: 62, Bottom: 21, Right: 62}, 3, ui.ViewPadding{Left: 186, Bottom: 63, Right: 186}},
		{"a negative inset is none", edgeInsets{Top: -4, Bottom: 2}, 2, ui.ViewPadding{Bottom: 4}},
		{"no scale is 1", edgeInsets{Top: 20}, 0, ui.ViewPadding{Top: 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := viewPadding(tc.in, tc.scale); got != tc.want {
				t.Errorf("viewPadding(%v, %v) = %v, want %v", tc.in, tc.scale, got, tc.want)
			}
		})
	}
}
