package android

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestDecodeInsets(t *testing.T) {
	tests := []struct {
		name   string
		values []int32
		want   insets
		ok     bool
	}{
		{"the three sets", []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, insets{
			padding: edgeInsets{1, 2, 3, 4}, keyboard: edgeInsets{5, 6, 7, 8}, gesture: edgeInsets{9, 10, 11, 12},
		}, true},
		{"zero", make([]int32, 12), insets{}, true},
		{"too few", []int32{1, 2, 3}, insets{}, false},
		{"too many", make([]int32, 13), insets{}, false},
		{"negative", []int32{0, 0, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0}, insets{}, false},
	}
	for _, tt := range tests {
		got, ok := decodeInsets(tt.values)
		if ok != tt.ok || got != tt.want {
			t.Errorf("%s: decodeInsets = %+v, %v; want %+v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

// TestInsetsAreInTheMetrics checks that the insets of the Java side become the view's padding,
// the keyboard's insets and the gesture insets, and that a change of them is a change of the
// metrics (also while the size is the same).
func TestInsetsAreInTheMetrics(t *testing.T) {
	h := newHarness(t)
	h.b.surfaceCreated(0x1000)
	h.b.surfaceChanged(1080, 2400, 2.625, 60, 1080, 2400)
	h.run()
	waitFor(t, "the first metrics", func() bool { return h.h.count("metrics") == 1 })
	if m := h.h.lastMetrics(); m.ViewPadding != ui.ViewPaddingZero || m.ViewInsets != ui.ViewPaddingZero {
		t.Errorf("the first metrics have insets: %+v", m)
	}
	h.b.insetsChanged(insets{padding: edgeInsets{0, 63, 0, 126}, keyboard: edgeInsets{0, 0, 0, 800}, gesture: edgeInsets{40, 0, 40, 126}})
	waitFor(t, "the metrics with insets", func() bool { return h.h.count("metrics") == 2 })
	m := h.h.lastMetrics()
	if m.ViewPadding != (ui.ViewPadding{Top: 63, Bottom: 126}) || m.ViewInsets != (ui.ViewPadding{Bottom: 800}) ||
		m.SystemGestureInsets != (ui.ViewPadding{Left: 40, Right: 40, Bottom: 126}) {
		t.Errorf("the metrics are %+v", m)
	}
	// The same insets again change nothing.
	h.b.insetsChanged(insets{padding: edgeInsets{0, 63, 0, 126}, keyboard: edgeInsets{0, 0, 0, 800}, gesture: edgeInsets{40, 0, 40, 126}})
	h.sync(func() {})
	if n := h.h.count("metrics"); n != 2 {
		t.Errorf("the same insets were reported again (%d metrics)", n)
	}
	// A new surface of the same size keeps the insets.
	h.b.surfaceChanged(1080, 2400, 2.625, 60, 1080, 2400)
	h.sync(func() {})
	if m := h.h.lastMetrics(); m.ViewPadding.Top != 63 {
		t.Errorf("a surface change dropped the insets: %+v", m)
	}
}

func TestDisplayChangesAreReported(t *testing.T) {
	h := newHarness(t)
	h.b.surfaceCreated(0x1000)
	h.b.surfaceChanged(1080, 2400, 2.625, 60, 1080, 2400)
	h.run()
	waitFor(t, "the first metrics", func() bool { return h.h.count("metrics") == 1 })
	h.b.displayChanged(2.625, 120, 1080, 2400)
	waitFor(t, "the displays change", func() bool { return h.h.count("displays") == 1 })
	if got := h.p.Displays()[0].RefreshRate; got != 120 {
		t.Errorf("the refresh rate is %v after the change, want 120", got)
	}
	h.b.displayChanged(2.625, 120, 1080, 2400)
	h.sync(func() {})
	if n := h.h.count("displays"); n != 1 {
		t.Errorf("the same display was reported again (%d)", n)
	}
}
