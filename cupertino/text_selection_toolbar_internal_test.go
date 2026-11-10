package cupertino

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/text_selection_toolbar_test.dart. They are in package
// cupertino because they find the chevrons by the type of their painter and the visible part of the toolbar by its
// private type, as the Dart test does with runtimeType. The cases that read the canvas with a `paints` matcher are
// the goldens of text_selection_golden_test.go, and 'can create and use a custom toolbar' is not ported: the
// deprecated buildToolbar of the selection controls does not exist here (SPEC §12.10, step 34.5).

const (
	toolbarTestArrowScreenPadding = 26.0
	toolbarTestContentDistance    = 8.0
)

var toolbarTestArrowSize = ui.Size{Width: 14, Height: 7}

func toolbarTestApp(home widgets.Widget) widgets.Widget {
	roboto := &painting.TextStyle{FontFamily: "Roboto", FontSize: foundation.Some(17.0), Color: foundation.Some(ui.NewColor(0xFF000000))}
	return NewCupertinoApp(CupertinoAppOpts{
		Theme:                      NewCupertinoThemeData(CupertinoThemeDataOpts{TextTheme: NewCupertinoTextThemeData(CupertinoTextThemeDataOpts{TextStyle: roboto})}),
		DebugShowCheckedModeBanner: foundation.Some(false),
		Home:                       home,
	})
}

type toolbarTestBox struct{ widgets.StatelessWidget }

func newToolbarTestBox() *toolbarTestBox {
	return &toolbarTestBox{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{})}
}

func (*toolbarTestBox) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(44.0)})
}

func toolbarTestBoxes(n int) []widgets.Widget {
	out := make([]widgets.Widget, n)
	for i := range out {
		out[i] = newToolbarTestBox()
	}
	return out
}

func chevron(left bool) gluttertest.Finder {
	return gluttertest.ByPredicate("a chevron", func(w widgets.Widget) bool {
		c, ok := w.(*widgets.CustomPaint)
		if !ok {
			return false
		}
		p, ok := c.Painter().(*cupertinoChevronPainter)
		return ok && p.isLeft == left
	})
}

func visibleToolbar() gluttertest.Finder {
	return gluttertest.ByType[*cupertinoTextSelectionToolbarContent]()
}

func TestCupertinoTextSelectionToolbarChevronsAndPages(t *testing.T) {
	boxes := gluttertest.ByType[*toolbarTestBox]()
	t.Run("the chevrons point to the correct side", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(toolbarTestApp(widgets.NewCenter(NewCupertinoTextSelectionToolbar(ui.Offset{Dx: 50, Dy: 100}, ui.Offset{Dx: 50, Dy: 200}, toolbarTestBoxes(15)))))
		if ts.Exists(chevron(true)) || ts.Count(chevron(false)) != 1 {
			t.Fatal("the first page has a back button or no next button")
		}
		ts.TapAt(ts.Center(chevron(false)))
		ts.PumpAndSettle()
		if ts.Count(chevron(true)) != 1 || ts.Count(chevron(false)) != 1 {
			t.Fatal("the second page does not have both buttons")
		}
		ts.TapAt(ts.Center(chevron(false)))
		ts.PumpAndSettle()
		if ts.Count(chevron(true)) != 1 || ts.Exists(chevron(false)) {
			t.Error("the last page does not have only the back button")
		}
	})
	t.Run("the chevron has the geometry of the arrow of iOS", func(t *testing.T) {
		for _, left := range []bool{false, true} {
			recorder := ui.NewPictureRecorder()
			(&cupertinoChevronPainter{color: ui.NewColor(0xFF000000), isLeft: left}).Paint(ui.NewCanvas(recorder), ui.Size{Width: 10, Height: 10})
			var lines [][2]ui.Offset
			for _, op := range recorder.EndRecording().DisplayList().Ops() {
				if l, ok := op.(ui.OpDrawLine); ok {
					lines = append(lines, [2]ui.Offset{l.P1, l.P2})
				}
			}
			want := [][2]ui.Offset{{{Dx: 2.5}, {Dx: 7.5, Dy: 5}}, {{Dx: 7.5, Dy: 5}, {Dx: 2.5, Dy: 10}}}
			if left {
				want = [][2]ui.Offset{{{Dx: 7.5}, {Dx: 2.5, Dy: 5}}, {{Dx: 2.5, Dy: 5}, {Dx: 7.5, Dy: 10}}}
			}
			if len(lines) != 2 || lines[0] != want[0] || lines[1] != want[1] {
				t.Errorf("left %v: lines %v, want %v", left, lines, want)
			}
		}
	})
	t.Run("the children are paginated when they overflow", func(t *testing.T) {
		ts := gluttertest.New(t)
		children := toolbarTestBoxes(7)
		var setState widgets.StateSetter
		ts.PumpWidget(toolbarTestApp(widgets.NewCenter(widgets.NewStatefulBuilder(func(_ widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
			setState = set
			return NewCupertinoTextSelectionToolbar(ui.Offset{Dx: 50, Dy: 100}, ui.Offset{Dx: 50, Dy: 200}, children)
		}))))
		check := func(step string, wantBoxes int, next, back bool) {
			t.Helper()
			if got := ts.Count(boxes); got != wantBoxes {
				t.Errorf("%s: %d boxes, want %d", step, got, wantBoxes)
			}
			if got := ts.Exists(chevron(false)); got != next {
				t.Errorf("%s: next button %v, want %v", step, got, next)
			}
			if got := ts.Exists(chevron(true)); got != back {
				t.Errorf("%s: back button %v, want %v", step, got, back)
			}
		}
		check("all fit", 7, false, false)
		setState(func() { children = append(children, newToolbarTestBox()) })
		ts.PumpAndSettle()
		check("one more", 7, true, false)
		ts.TapAt(ts.Center(chevron(false)))
		ts.PumpAndSettle()
		check("the next page", 1, false, true)
		ts.TapAt(ts.Center(chevron(true)))
		ts.PumpAndSettle()
		check("back to the first", 7, true, false)
		setState(func() { children = append(children, toolbarTestBoxes(6)...) })
		ts.PumpAndSettle()
		check("three pages", 7, true, false)
		ts.TapAt(ts.Center(chevron(false)))
		ts.PumpAndSettle()
		check("the second of three", 6, true, true)
		ts.TapAt(ts.Center(chevron(false)))
		ts.PumpAndSettle()
		check("the third of three", 1, false, true)
		ts.TapAt(ts.Center(chevron(true)))
		ts.PumpAndSettle()
		check("back to the second", 6, true, true)
		ts.TapAt(ts.Center(chevron(true)))
		ts.PumpAndSettle()
		check("back to the first of three", 7, true, false)
	})
	t.Run("the children do not paginate if they fit with zero margin", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.SetDevicePixelRatio(2)
		spacer, divider := 1.0/2, 1.0/2
		width := 7*100.0 + 6*(divider+2*spacer) + 2*8.0
		ts.PumpWidget(toolbarTestApp(widgets.NewCenter(widgets.NewSizedBox(
			NewCupertinoTextSelectionToolbar(ui.Offset{Dx: 50, Dy: 100}, ui.Offset{Dx: 50, Dy: 200}, toolbarTestBoxes(7)),
			widgets.SizedBoxOpts{Width: foundation.Some(width)}))))
		if ts.Count(boxes) != 7 || ts.Exists(chevron(false)) || ts.Exists(chevron(true)) {
			t.Errorf("%d boxes, next %v, back %v", ts.Count(boxes), ts.Exists(chevron(false)), ts.Exists(chevron(true)))
		}
	})
	t.Run("large buttons get their own page", func(t *testing.T) {
		ts := gluttertest.New(t)
		var children []widgets.Widget
		for _, w := range []float64{100, 300, 100, 100} {
			k := foundation.NewUniqueKey()
			children = append(children, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: k, Width: foundation.Some(w)}))
		}
		ts.PumpWidget(toolbarTestApp(widgets.NewCenter(widgets.NewSizedBox(
			NewCupertinoTextSelectionToolbar(ui.Offset{Dx: 50, Dy: 100}, ui.Offset{Dx: 50, Dy: 200}, children),
			widgets.SizedBoxOpts{Width: foundation.Some(420.0)}))))
		widthsShown := func() []float64 {
			var out []float64
			for _, w := range ts.Widgets(gluttertest.ByPredicate("a sized box", func(w widgets.Widget) bool {
				s, ok := w.(*widgets.SizedBox)
				return ok && s.Key() != nil && s.Width().IsSome()
			})) {
				out = append(out, w.(*widgets.SizedBox).Width().Value())
			}
			return out
		}
		equal := func(a, b []float64) bool {
			if len(a) != len(b) {
				return false
			}
			for i := range a {
				if a[i] != b[i] {
					return false
				}
			}
			return true
		}
		if got := widthsShown(); !equal(got, []float64{100}) {
			t.Fatalf("first page %v", got)
		}
		ts.TapAt(ts.Center(chevron(false)))
		ts.PumpAndSettle()
		if got := widthsShown(); !equal(got, []float64{300}) {
			t.Fatalf("second page %v", got)
		}
		ts.TapAt(ts.Center(chevron(false)))
		ts.PumpAndSettle()
		if got := widthsShown(); !equal(got, []float64{100, 100}) {
			t.Errorf("third page %v", got)
		}
	})
}

func TestCupertinoTextSelectionToolbarPosition(t *testing.T) {
	const height, anchorBelowY, paddingAbove = 50.0, 500.0, 12.0
	setup := func(t *testing.T) (*gluttertest.Tester, func(float64)) {
		ts := gluttertest.New(t)
		anchorAboveY := 0.0
		var setState widgets.StateSetter
		ts.PumpWidget(toolbarTestApp(widgets.NewCenter(widgets.NewStatefulBuilder(func(bc widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
			setState = set
			data := widgets.MediaQueryOf(bc)
			box := func(c uint32) widgets.Widget {
				return widgets.NewColoredBox(ui.NewColor(c), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(height)}))
			}
			padding := data.ViewPadding
			padding.Top = paddingAbove
			return widgets.NewMediaQuery(data.CopyWith(widgets.MediaQueryDataCopyOpts{Padding: foundation.Some(padding)}),
				NewCupertinoTextSelectionToolbar(ui.Offset{Dx: 50, Dy: anchorAboveY}, ui.Offset{Dx: 50, Dy: anchorBelowY},
					[]widgets.Widget{box(0xFFFF0000), box(0xFF00FF00), box(0xFF0000FF)}))
		}))))
		return ts, func(y float64) {
			setState(func() { anchorAboveY = y })
			ts.Pump()
		}
	}
	t.Run("the toolbar is below the anchor when it does not fit above, and above when it does", func(t *testing.T) {
		ts, move := setup(t)
		if y := ts.TopLeft(visibleToolbar()).Dy; y != anchorBelowY+toolbarTestContentDistance {
			t.Errorf("toolbar at %v, want %v", y, anchorBelowY+toolbarTestContentDistance)
		}
		move(70) // barely does not fit
		if y := ts.TopLeft(visibleToolbar()).Dy; y != anchorBelowY+toolbarTestContentDistance {
			t.Errorf("toolbar at %v when it barely does not fit, want %v", y, anchorBelowY+toolbarTestContentDistance)
		}
		move(80)
		want := 80 - height + toolbarTestArrowSize.Height - toolbarTestContentDistance
		if y := ts.TopLeft(visibleToolbar()).Dy; y != want {
			t.Errorf("toolbar at %v, want %v", y, want)
		}
	})
	t.Run("the layout delegate gets the anchor below minus the padding", func(t *testing.T) {
		ts, _ := setup(t)
		layout := ts.Widget(gluttertest.ByType[*widgets.CustomSingleChildLayout]()).(*widgets.CustomSingleChildLayout)
		d, ok := layout.Delegate().(*widgets.TextSelectionToolbarLayoutDelegate)
		if !ok {
			t.Fatalf("delegate %T", layout.Delegate())
		}
		if d.AnchorBelow().Dy != anchorBelowY-paddingAbove {
			t.Errorf("anchor below %v, want %v", d.AnchorBelow().Dy, anchorBelowY-paddingAbove)
		}
	})
}

func TestCupertinoTextSelectionToolbarButtonColorFollowsTheBrightness(t *testing.T) {
	brightnesses := []ui.Brightness{ui.BrightnessUnspecified, ui.BrightnessLight, ui.BrightnessDark}
	for _, themeB := range brightnesses {
		for _, mediaB := range brightnesses {
			t.Run("", func(t *testing.T) {
				ts := gluttertest.New(t)
				ts.PumpWidget(NewCupertinoApp(CupertinoAppOpts{
					Theme:                      NewCupertinoThemeData(CupertinoThemeDataOpts{Brightness: themeB}),
					DebugShowCheckedModeBanner: foundation.Some(false),
					Home: widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
						data := widgets.MediaQueryOf(bc)
						if mediaB != ui.BrightnessUnspecified {
							data = data.CopyWith(widgets.MediaQueryDataCopyOpts{PlatformBrightness: foundation.Some(mediaB)})
						}
						return widgets.NewMediaQuery(data, NewCupertinoTextSelectionToolbar(ui.Offset{Dx: 100}, ui.Offset{Dx: 100},
							[]widgets.Widget{NewCupertinoTextSelectionToolbarButtonText("Button", CupertinoTextSelectionToolbarButtonOpts{OnPressed: func() {}})}))
					})),
				}))
				text := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*CupertinoButton](), gluttertest.ByType[*widgets.Text]())).(*widgets.Text)
				effective := themeB
				if effective == ui.BrightnessUnspecified {
					effective = mediaB
				}
				if effective == ui.BrightnessUnspecified {
					effective = ui.BrightnessLight
				}
				want := ui.NewColor(0xFF000000)
				if effective == ui.BrightnessDark {
					want = ui.NewColor(0xFFFFFFFF)
				}
				if got := text.Style().Color.Value(); got != want {
					t.Errorf("theme %v, media %v: color %v, want %v", themeB, mediaB, got, want)
				}
			})
		}
	}
}

func TestCupertinoTextSelectionToolbarNoChildren(t *testing.T) {
	requireRoute(t, func() { NewCupertinoTextSelectionToolbar(ui.Offset{}, ui.Offset{}, nil) },
		"NewCupertinoTextSelectionToolbar."+tagCupertinoTextSelectionToolbarNoChildren)
}
