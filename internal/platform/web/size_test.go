package web

import (
	"testing"
	"time"
)

func TestSanitizeDPR(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"one", 1, 1},
		{"fractional", 1.25, 1.25},
		{"retina", 2, 2},
		{"zero", 0, 1},
		{"negative", -2, 1},
		{"nan", nan(), 1},
		{"infinite", inf(), 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeDPR(tc.in); got != tc.want {
				t.Errorf("SanitizeDPR(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolutionQuery(t *testing.T) {
	tests := []struct {
		dpr  float64
		want string
	}{
		{1, "(resolution: 1dppx)"},
		{1.5, "(resolution: 1.5dppx)"},
		{2, "(resolution: 2dppx)"},
		{1.100000023841858, "(resolution: 1.100000023841858dppx)"},
		{0, "(resolution: 1dppx)"},
	}
	for _, tc := range tests {
		if got := ResolutionQuery(tc.dpr); got != tc.want {
			t.Errorf("ResolutionQuery(%v) = %q, want %q", tc.dpr, got, tc.want)
		}
	}
}

func TestPhysicalSize(t *testing.T) {
	tests := []struct {
		name         string
		g            Geometry
		wantW, wantH int
	}{
		{"css at 1x", Geometry{CSSWidth: 480, CSSHeight: 640, DPR: 1}, 480, 640},
		{"css at 2x", Geometry{CSSWidth: 480, CSSHeight: 640, DPR: 2}, 960, 1280},
		{"css rounds", Geometry{CSSWidth: 333.3, CSSHeight: 100.5, DPR: 1.5}, 500, 151},
		{"exact device box wins", Geometry{CSSWidth: 333.3, CSSHeight: 100.5, DeviceWidth: 499, DeviceHeight: 150, DPR: 1.5}, 499, 150},
		{"half a device box is not used", Geometry{CSSWidth: 100, CSSHeight: 50, DeviceWidth: 200, DPR: 2}, 200, 100},
		{"no ratio means one", Geometry{CSSWidth: 100, CSSHeight: 50}, 100, 50},
		{"not a number", Geometry{CSSWidth: nan(), CSSHeight: -3, DPR: 1}, 0, 0},
		{"empty", Geometry{}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, h := tc.g.PhysicalSize()
			if w != tc.wantW || h != tc.wantH {
				t.Errorf("PhysicalSize() = %dx%d, want %dx%d", w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestViewMetrics(t *testing.T) {
	m := Geometry{CSSWidth: 480, CSSHeight: 640, DPR: 1.5}.ViewMetrics()
	if m.PhysicalSize.Width != 720 || m.PhysicalSize.Height != 960 || m.DevicePixelRatio != 1.5 {
		t.Errorf("ViewMetrics() = %+v, want 720x960 at 1.5", m)
	}
	if m.ViewPadding != (m.ViewInsets) || m.ViewPadding.Left != 0 || m.ViewPadding.Bottom != 0 {
		t.Errorf("the browser has no system areas, got padding %+v insets %+v", m.ViewPadding, m.ViewInsets)
	}
}

func TestFrameTime(t *testing.T) {
	tests := []struct {
		name string
		ms   float64
		want time.Duration
	}{
		{"zero", 0, 0},
		{"a frame", 16.5, 16*time.Millisecond + 500*time.Microsecond},
		{"seconds", 12345.678, 12345*time.Millisecond + 678*time.Microsecond},
		{"negative", -1, 0},
		{"nan", nan(), 0},
		{"infinite", inf(), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FrameTime(tc.ms)
			if diff := got - tc.want; diff < -time.Microsecond || diff > time.Microsecond {
				t.Errorf("FrameTime(%v) = %v, want %v", tc.ms, got, tc.want)
			}
		})
	}
}

func TestDisplayOf(t *testing.T) {
	d := DisplayOf(1920, 1080, 2)
	if !d.Primary || d.RefreshRate != 60 || d.DevicePixelRatio != 2 || d.Bounds.Width() != 1920 || d.Bounds.Height() != 1080 {
		t.Errorf("DisplayOf() = %+v", d)
	}
	if d := DisplayOf(800, 600, 0); d.DevicePixelRatio != 1 {
		t.Errorf("a missing ratio is 1, got %v", d.DevicePixelRatio)
	}
}
