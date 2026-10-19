package main

import (
	"testing"

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
