package main

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
)

// TestLayoutGolden draws the root widget of the example at the size of its
// window and compares it with the golden image.
func TestLayoutGolden(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 960, Height: 480}})
	ts.PumpWidget(newLayoutApp())
	ts.MatchesGolden("layout")
}

// TestLayoutFits checks that nothing overflows: the framework reports an error
// for a RenderFlex that does.
func TestLayoutFits(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 960, Height: 480}})
	ts.PumpWidget(newLayoutApp())
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("the example reported errors: %v", errs)
	}
}

// TestDebugFromEnv checks that GLUTTER_DEV_DEBUG_PAINT=1 turns on the painting of sizes.
func TestDebugFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{{"", false}, {"0", false}, {"1", true}}
	for _, tt := range tests {
		got := debugFromEnv(func(string) string { return tt.value }).Rendering.PaintSizeEnabled
		if got != tt.want {
			t.Errorf("GLUTTER_DEV_DEBUG_PAINT=%q: PaintSizeEnabled = %v, want %v", tt.value, got, tt.want)
		}
	}
}

// TestDebugPaintGolden draws the example with the sizes painted.
func TestDebugPaintGolden(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the debug flags have an effect in debug builds only")
	}
	debug := debugFromEnv(func(string) string { return "1" })
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 960, Height: 480}, Debug: debug})
	ts.PumpWidget(newLayoutApp())
	ts.MatchesGolden("layout-debug-paint")
}
