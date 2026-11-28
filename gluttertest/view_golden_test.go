package gluttertest

import (
	"image"
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The colors of the swatches of the main window and of the other one.
var (
	mainSwatch   = ui.NewColor(0xFF6750A4)
	windowSwatch = ui.NewColor(0xFFFFB74D)
)

// mainAndWindow pumps an app as the issue's app is: a WindowManager around the
// main window's content, and a regular window of 400x300 registered in it, which
// each frame draws after the implicit view. It returns the window's controller.
func mainAndWindow(t *testing.T, ts *Tester) *widgets.RegularWindowController {
	t.Helper()
	var bc widgets.BuildContext
	ts.PumpWidget(widgets.NewWindowManager(widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
		bc = c
		return newRow(newSwatch(mainSwatch, 100, 60))
	})))
	c, err := widgets.NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, widgets.RegularWindowControllerOpts{Title: "Report"})
	if err != nil {
		t.Fatal(err)
	}
	widgets.WindowRegistryOf(bc).Register(widgets.NewWindowEntry(c, func(widgets.BuildContext) widgets.Widget {
		return newRow(newSwatch(windowSwatch, 60, 100))
	}))
	ts.Pump()
	return c
}

// rgba is the color as the software renderer writes it.
func rgba(c ui.Color) color.RGBA {
	return color.RGBA{R: uint8(c.Red()), G: uint8(c.Green()), B: uint8(c.Blue()), A: uint8(c.Alpha())}
}

// TestTesterTakesTheGoldenOfOneWindow is the issue "A golden of one window while others are open" (step 62.1):
// Screenshot and MatchesGolden rendered the last scene that was composited, of any view, so while a window of a
// WindowManager was open they showed that window (at the size of the main one), never the main window. Now they
// render the implicit view's last scene, and ScreenshotView and MatchesViewGolden render another view's.
func TestTesterTakesTheGoldenOfOneWindow(t *testing.T) {
	tests := []struct {
		name  string
		shot  func(ts *Tester, window ui.ViewID) *image.RGBA
		size  image.Point
		color ui.Color
	}{
		{"Screenshot is of the main window", func(ts *Tester, _ ui.ViewID) *image.RGBA { return ts.Screenshot() },
			image.Pt(800, 600), mainSwatch},
		{"ScreenshotView of the implicit view", func(ts *Tester, _ ui.ViewID) *image.RGBA { return ts.ScreenshotView(ui.ImplicitViewID) },
			image.Pt(800, 600), mainSwatch},
		{"ScreenshotView of the other window", func(ts *Tester, window ui.ViewID) *image.RGBA { return ts.ScreenshotView(window) },
			image.Pt(400, 300), windowSwatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t)
			c := mainAndWindow(t, ts)
			window := c.RootView().ViewID()
			scenes := ts.Scenes()
			if len(scenes) == 0 || len(ts.ViewScenes(window)) == 0 || scenes[len(scenes)-1] != ts.ViewScenes(window)[len(ts.ViewScenes(window))-1] {
				t.Fatal("the other window did not draw last: the test does not show the issue")
			}
			img := tt.shot(ts, window)
			if got := img.Bounds().Size(); got != tt.size {
				t.Errorf("size = %v, want %v", got, tt.size)
			}
			if got, want := img.RGBAAt(5, 5), rgba(tt.color); got != want {
				t.Errorf("the swatch's pixel = %v, want %v", got, want)
			}
		})
	}
}

// TestTesterMatchesTheGoldenOfOneWindow compares each window with a golden of its own while both are open.
func TestTesterMatchesTheGoldenOfOneWindow(t *testing.T) {
	ts := New(t)
	c := mainAndWindow(t, ts)
	ts.MatchesGolden("one_window_main")
	ts.MatchesViewGolden(c.RootView().ViewID(), "one_window_report")
}

func TestScreenshotViewErrors(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(t *testing.T, ts *Tester)
		route string
	}{
		{"a view that was never made", func(_ *testing.T, ts *Tester) {
			ts.PumpWidget(newRow())
			ts.ScreenshotView(42)
		}, "Tester.ScreenshotView->Tester.screenshot.ScreenshotViewUnknown"},
		{"the view of a destroyed window", func(t *testing.T, ts *Tester) {
			c := mainAndWindow(t, ts)
			c.Destroy()
			ts.Pump()
			ts.ScreenshotView(c.RootView().ViewID())
		}, "Tester.ScreenshotView->Tester.screenshot.ScreenshotViewUnknown"},
		{"a window that drew no frame yet", func(t *testing.T, ts *Tester) {
			var bc widgets.BuildContext
			ts.PumpWidget(widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget { bc = c; return newRow() }))
			c, err := widgets.NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
			if err != nil {
				t.Fatal(err)
			}
			ts.ScreenshotView(c.RootView().ViewID())
		}, "Tester.ScreenshotView->Tester.screenshot.ScreenshotNoFrame"},
		{"MatchesViewGolden of a view that was never made", func(_ *testing.T, ts *Tester) {
			ts.PumpWidget(newRow())
			ts.MatchesViewGolden(42, "never_written")
		}, "Tester.MatchesViewGolden->Tester.ScreenshotView->Tester.screenshot.ScreenshotViewUnknown"},
		{"Screenshot without a frame", func(_ *testing.T, ts *Tester) { ts.Screenshot() },
			"Tester.Screenshot->Tester.screenshot.ScreenshotNoFrame"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := failure(t, func(ts *Tester) { tt.fn(t, ts) })
			requireRoute(t, appErr, tt.route)
		})
	}
}
