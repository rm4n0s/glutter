package offscreen_test

import (
	"bytes"
	"fmt"
	"image"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The size of the frames of the benchmarks and the tests that compare whole frames.
const (
	frameW = 1280
	frameH = 800
)

// scaffoldApp is a MaterialApp whose Scaffold has an AppBar and the body. Its theme is the
// Linux one on every platform, so that the text is set in the bundled Roboto (the typography
// of Windows asks for Segoe UI, which the tests do not have).
func scaffoldApp(body widgets.Widget) widgets.Widget {
	theme := material.NewThemeData(material.ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux)})
	return material.NewMaterialApp(material.MaterialAppOpts{Theme: theme, Home: material.NewScaffold(material.ScaffoldOpts{
		AppBar: material.NewAppBar(material.AppBarOpts{Title: widgets.NewText("Offscreen")}),
		Body:   body,
	})})
}

// plainBody is a body without text.
func plainBody() widgets.Widget {
	return widgets.NewColoredBox(ui.NewColor(0xFFE8EAF6), widgets.NewSizedBoxExpand(nil))
}

// textBody is a body of 45 lines of 14 pixel text.
func textBody() widgets.Widget {
	lines := make([]widgets.Widget, 45)
	for i := range lines {
		lines[i] = widgets.NewText(fmt.Sprintf("Line %d: the quick brown fox jumps over the lazy dog, 0123456789 — Glutter", i),
			widgets.TextOpts{Style: &painting.TextStyle{Inherit: true, FontSize: foundation.Some(14.0)}})
	}
	return widgets.NewListView(lines)
}

// pumpFrame pumps a widget in a tester of the frame size and returns the tester and
// the last scene.
func pumpFrame(tb testing.TB, w widgets.Widget) (*gluttertest.Tester, *ui.Scene) {
	tb.Helper()
	ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: frameW, Height: frameH}})
	ts.PumpWidget(w)
	ts.Pump(50 * time.Millisecond)
	scenes := ts.Scenes()
	return ts, scenes[len(scenes)-1]
}

// fieldApp is a MaterialApp whose body is a focused text field above lines of text.
func fieldApp() widgets.Widget {
	return scaffoldApp(widgets.NewColumn([]widgets.Widget{
		widgets.NewSizedBox(material.NewTextField(material.TextFieldOpts{Autofocus: true, Controller: widgets.NewTextEditingController("caret here")}),
			widgets.SizedBoxOpts{Height: foundation.Some(64.0)}),
		widgets.NewExpanded(textBody()),
	}))
}

// caretFrames returns two consecutive scenes of a focused text field that differ in the
// caret only: it blinks.
func caretFrames(tb testing.TB) (*ui.Scene, *ui.Scene) {
	tb.Helper()
	ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: frameW, Height: frameH}})
	ts.PumpWidget(fieldApp())
	ts.Pump(50 * time.Millisecond)
	ts.Pump(500 * time.Millisecond)
	a := ts.Scenes()[len(ts.Scenes())-1]
	ts.Pump(500 * time.Millisecond)
	b := ts.Scenes()[len(ts.Scenes())-1]
	return a, b
}

// samePixels fails the test when the images differ, and says where.
func samePixels(t *testing.T, what string, got, want *image.RGBA) {
	t.Helper()
	if got.Rect != want.Rect {
		t.Fatalf("%s: the images are %v and %v", what, got.Rect, want.Rect)
	}
	for y := got.Rect.Min.Y; y < got.Rect.Max.Y; y++ {
		for x := got.Rect.Min.X; x < got.Rect.Max.X; x++ {
			i := got.PixOffset(x, y)
			if !bytes.Equal(got.Pix[i:i+4], want.Pix[i:i+4]) {
				t.Fatalf("%s: pixel %d,%d is %v, want %v", what, x, y, got.Pix[i:i+4], want.Pix[i:i+4])
			}
		}
	}
}

// lastScene is the last scene the tester composited.
func lastScene(ts *gluttertest.Tester) *ui.Scene {
	scenes := ts.Scenes()
	return scenes[len(scenes)-1]
}

// area is the number of pixels the rectangles cover, when they do not overlap.
func area(rects []image.Rectangle) int {
	n := 0
	for _, r := range rects {
		n += r.Dx() * r.Dy()
	}
	return n
}
