package svg

import (
	"image/color"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// red20x10 is a red rectangle in a document of 20 by 10 units.
const red20x10 = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10" viewBox="0 0 20 10"><rect width="20" height="10" fill="#f00"/></svg>`

// newTester makes a tester with a view of 100 by 100 logical pixels.
func newTester(t *testing.T, o ...gluttertest.Opts) *gluttertest.Tester {
	t.Helper()
	if len(o) == 0 {
		o = []gluttertest.Opts{{Size: ui.Size{Width: 100, Height: 100}}}
	}
	return gluttertest.New(t, o[0])
}

// pumpTopLeft shows the widget at the top left of the view, with loose constraints.
func pumpTopLeft(ts *gluttertest.Tester, w widgets.Widget, dir ...ui.TextDirection) {
	d := ui.TextDirectionLTR
	if len(dir) > 0 {
		d = dir[0]
	}
	ts.PumpWidget(widgets.NewDirectionality(d, widgets.NewAlign(w, widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})))
}

// waitFor pumps frames until the condition holds.
func waitFor(t *testing.T, ts *gluttertest.Tester, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		ts.Pump()
		time.Sleep(time.Millisecond)
	}
}

// loaded is true once the picture widget has its document (or its error).
func loaded(ts *gluttertest.Tester) func() bool {
	return func() bool {
		s := gluttertest.StateOf[*svgPictureState](ts, gluttertest.ByType[*SvgPicture]())
		return s.entry != nil || s.err != nil
	}
}

// pixel is the color of a pixel of the last frame.
func pixel(ts *gluttertest.Tester, x, y int) color.RGBA {
	return ts.Screenshot().RGBAAt(x, y)
}

var (
	redC  = color.RGBA{255, 0, 0, 255}
	blueC = color.RGBA{0, 0, 255, 255}
)

func some[T any](v T) foundation.Opt[T] { return foundation.Some(v) }
