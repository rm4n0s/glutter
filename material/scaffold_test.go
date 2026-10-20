package material_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests check the Material 3 geometry and colors of the Scaffold, AppBar
// and FloatingActionButton that the counter example uses. The ported tests of
// Flutter are in the other files.

// bodyBox is the body of the scaffolds below: a box that expands, so its size is
// the size the scaffold gives its body.
func bodyBox() widgets.Widget {
	return widgets.NewColoredBox(ui.NewColor(0xFF00FF00), widgets.NewSizedBoxExpand(nil))
}

func pumpScaffold(t *testing.T, dir ui.TextDirection, mq widgets.MediaQueryData, s *Scaffold) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(widgets.NewDirectionality(dir, NewTheme(NewThemeData(), widgets.NewMediaQuery(mq, s))))
	return ts
}

func TestScaffoldLayout(t *testing.T) {
	const w, h = 400.0, 600.0
	tests := []struct {
		name       string
		dir        ui.TextDirection
		mq         widgets.MediaQueryData
		o          ScaffoldOpts
		wantBar    ui.Size
		wantBody   ui.Rect
		wantButton ui.Offset
	}{
		{
			name:       "no insets",
			mq:         widgets.NewMediaQueryData(),
			wantBar:    ui.Size{Width: w, Height: 56},
			wantBody:   ui.NewRectFromLTWH(0, 56, w, h-56),
			wantButton: ui.Offset{Dx: w - 16 - 56, Dy: h - 16 - 56},
		},
		{
			name: "status bar",
			mq: widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{
				Padding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 0), ViewPadding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 0)}),
			wantBar:    ui.Size{Width: w, Height: 56 + 24},
			wantBody:   ui.NewRectFromLTWH(0, 80, w, h-80),
			wantButton: ui.Offset{Dx: w - 16 - 56, Dy: h - 16 - 56},
		},
		{
			name: "status bar with a scaffold that is not primary",
			mq: widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{
				Padding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 0), ViewPadding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 0)}),
			o:          ScaffoldOpts{NoPrimary: true},
			wantBar:    ui.Size{Width: w, Height: 56},
			wantBody:   ui.NewRectFromLTWH(0, 56, w, h-56),
			wantButton: ui.Offset{Dx: w - 16 - 56, Dy: h - 16 - 56},
		},
		{
			name:       "keyboard",
			mq:         widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{ViewInsets: painting.NewEdgeInsetsFromLTRB(0, 0, 0, 200)}),
			wantBar:    ui.Size{Width: w, Height: 56},
			wantBody:   ui.NewRectFromLTWH(0, 56, w, h-56-200),
			wantButton: ui.Offset{Dx: w - 16 - 56, Dy: h - 200 - 16 - 56},
		},
		{
			name:       "keyboard and a scaffold that does not resize",
			mq:         widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{ViewInsets: painting.NewEdgeInsetsFromLTRB(0, 0, 0, 200)}),
			o:          ScaffoldOpts{ResizeToAvoidBottomInset: foundation.Some(false)},
			wantBar:    ui.Size{Width: w, Height: 56},
			wantBody:   ui.NewRectFromLTWH(0, 56, w, h-56),
			wantButton: ui.Offset{Dx: w - 16 - 56, Dy: h - 16 - 56},
		},
		{
			// As in Flutter: only the keyboard shrinks the body; the body pads itself,
			// and the button keeps out of the view padding: 34 + 16 above the bottom.
			name: "home indicator",
			mq: widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{
				Padding: painting.NewEdgeInsetsFromLTRB(0, 0, 0, 34), ViewPadding: painting.NewEdgeInsetsFromLTRB(0, 0, 0, 34)}),
			wantBar:    ui.Size{Width: w, Height: 56},
			wantBody:   ui.NewRectFromLTWH(0, 56, w, h-56),
			wantButton: ui.Offset{Dx: w - 16 - 56, Dy: h - 56 - 34 - 16},
		},
		{
			name:       "right to left",
			dir:        ui.TextDirectionRTL,
			mq:         widgets.NewMediaQueryData(),
			wantBar:    ui.Size{Width: w, Height: 56},
			wantBody:   ui.NewRectFromLTWH(0, 56, w, h-56),
			wantButton: ui.Offset{Dx: 16, Dy: h - 16 - 56},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.o
			o.AppBar = NewAppBar(AppBarOpts{Title: widgets.NewText("Title"), NoPrimary: o.NoPrimary})
			o.Body = bodyBox()
			o.FloatingActionButton = NewFloatingActionButton(func() {}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)}))
			dir := tc.dir
			if dir == ui.TextDirectionUnspecified {
				dir = ui.TextDirectionLTR
			}
			ts := pumpScaffold(t, dir, tc.mq, NewScaffold(o))

			if got := ts.Size(gluttertest.ByType[*AppBar]()); got != tc.wantBar {
				t.Errorf("app bar size = %v, want %v", got, tc.wantBar)
			}
			body := gluttertest.ByType[*widgets.ColoredBox]()
			bodyRect := ui.NewRectFromLTWH(ts.TopLeft(body).Dx, ts.TopLeft(body).Dy, ts.Size(body).Width, ts.Size(body).Height)
			if bodyRect != tc.wantBody {
				t.Errorf("body = %v, want %v", bodyRect, tc.wantBody)
			}
			fab := gluttertest.ByType[*FloatingActionButton]()
			if got := ts.Size(fab); got != (ui.Size{Width: 56, Height: 56}) {
				t.Errorf("button size = %v, want 56 x 56", got)
			}
			if got := ts.TopLeft(fab); got != tc.wantButton {
				t.Errorf("button at %v, want %v", got, tc.wantButton)
			}
		})
	}
}

// TestScaffoldBodyPadding checks that the body does not pad itself for the
// status bar that the app bar already covers, and that the app bar does.
func TestScaffoldBodyPadding(t *testing.T) {
	var bodyPadding painting.EdgeInsets
	body := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		bodyPadding = widgets.MediaQueryPaddingOf(bc)
		return widgets.NewSizedBoxExpand(nil)
	})
	mq := widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{
		Padding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 34), ViewPadding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 34)})
	pumpScaffold(t, ui.TextDirectionLTR, mq, NewScaffold(ScaffoldOpts{AppBar: NewAppBar(), Body: body}))
	if bodyPadding.Top != 0 || bodyPadding.Bottom != 34 {
		t.Errorf("padding of the body = %v, want 0 on the top and 34 at the bottom", bodyPadding)
	}
}

func TestAppBarLayout(t *testing.T) {
	const w = 400.0
	title := func() widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(80.0), Height: foundation.Some(20.0)})
	}
	box := func(size float64) widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(size), Height: foundation.Some(size)})
	}
	tests := []struct {
		name      string
		o         AppBarOpts
		wantTitle ui.Rect
	}{
		{"title at the start", AppBarOpts{Title: title()}, ui.NewRectFromLTWH(16, 18, 80, 20)},
		{"centered title", AppBarOpts{Title: title(), CenterTitle: foundation.Some(true)}, ui.NewRectFromLTWH((w-80)/2, 18, 80, 20)},
		{"title spacing", AppBarOpts{Title: title(), TitleSpacing: foundation.Some(4.0)}, ui.NewRectFromLTWH(4, 18, 80, 20)},
		{"leading widget", AppBarOpts{Title: title(), Leading: box(24)}, ui.NewRectFromLTWH(56+16, 18, 80, 20)},
		{"tall toolbar", AppBarOpts{Title: title(), ToolbarHeight: foundation.Some(100.0)}, ui.NewRectFromLTWH(16, 40, 80, 20)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(),
				NewScaffold(ScaffoldOpts{AppBar: NewAppBar(tc.o), Body: bodyBox()}))
			sized := gluttertest.ByPredicate("the title box", func(wd widgets.Widget) bool {
				s, ok := wd.(*widgets.SizedBox)
				return ok && s.Width().OrElse(0) == 80
			})
			got := ui.NewRectFromLTWH(ts.TopLeft(sized).Dx, ts.TopLeft(sized).Dy, ts.Size(sized).Width, ts.Size(sized).Height)
			if got != tc.wantTitle {
				t.Errorf("title = %v, want %v", got, tc.wantTitle)
			}
		})
	}
}

func TestAppBarActions(t *testing.T) {
	action := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(48.0), Height: foundation.Some(48.0)})
	ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(),
		NewScaffold(ScaffoldOpts{AppBar: NewAppBar(AppBarOpts{Title: widgets.NewText("T"), Actions: []widgets.Widget{action}}), Body: bodyBox()}))
	f := gluttertest.ByWidget(action)
	if tl := ts.TopLeft(f); tl.Dx != 400-48 || tl.Dy != 4 {
		t.Errorf("action at %v, want (352, 4)", tl)
	}
	bar := NewAppBar()
	if got := bar.PreferredSize(); !math.IsInf(got.Width, 1) || got.Height != 56 {
		t.Errorf("preferred size = %v, want infinite x 56", got)
	}
	if got := NewAppBar(AppBarOpts{ToolbarHeight: foundation.Some(80.0)}).PreferredSize().Height; got != 80 {
		t.Errorf("preferred height = %v, want 80", got)
	}
}

func TestMaterial3Colors(t *testing.T) {
	theme := NewThemeData()
	cs := theme.ColorScheme
	ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{
		AppBar:               NewAppBar(AppBarOpts{Title: widgets.NewText("T")}),
		FloatingActionButton: NewFloatingActionButton(func() {}, nil),
	}))
	want := func(name string, x, y int, c ui.Color) {
		t.Helper()
		got := pixel(ts, x, y)
		if got.R != uint8(c.Red()) || got.G != uint8(c.Green()) || got.B != uint8(c.Blue()) {
			t.Errorf("%s at (%d, %d) = %v, want %v", name, x, y, got, c)
		}
	}
	want("scaffold", 200, 300, theme.ScaffoldBackgroundColor)
	want("app bar", 300, 10, cs.Surface())
	want("button", 400-16-56+28, 600-16-56+10, cs.PrimaryContainer())
}

func TestFloatingActionButton(t *testing.T) {
	taps := 0
	pump := func(onPressed foundation.VoidCallback) (*gluttertest.Tester, gluttertest.Finder) {
		ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{
			FloatingActionButton: NewFloatingActionButton(onPressed, widgets.NewSizedBox(nil), FloatingActionButtonOpts{Tooltip: "Add"}),
		}))
		return ts, gluttertest.ByType[*FloatingActionButton]()
	}
	ts, fab := pump(func() { taps++ })
	ts.Tap(fab)
	ts.Pump()
	if taps != 1 {
		t.Errorf("taps = %d, want 1", taps)
	}
	if w := gluttertest.WidgetAs[*FloatingActionButton](ts, fab); w.Tooltip() != "Add" || w.OnPressed() == nil || w.Child() == nil {
		t.Errorf("accessors: tooltip %q", w.Tooltip())
	}
	ts, fab = pump(nil)
	ts.Tap(fab)
	ts.Pump()
	if taps != 1 {
		t.Errorf("a disabled button was tapped: taps = %d", taps)
	}
}
