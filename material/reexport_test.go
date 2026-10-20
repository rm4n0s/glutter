package material_test

import (
	"strconv"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// The app of SPEC §12.12 without its Scaffold, which comes with step 13.1: it
// uses package material only.

type myApp struct{ m.StatelessWidget }

func newMyApp() *myApp { return &myApp{} }

func (a *myApp) Build(m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title: "Counter",
		Theme: m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Blue.Primary)}),
		Home:  newCounterPage("Counter"),
	})
}

type counterPage struct {
	m.StatefulWidget
	title string
}

func newCounterPage(title string, opts ...m.WidgetOpts) *counterPage {
	return &counterPage{StatefulWidget: m.NewStatefulWidgetBase(opts...), title: title}
}

func (p *counterPage) CreateState() m.State { return &counterPageState{} }

type counterPageState struct {
	m.StateOf[*counterPage]
	count int
}

func (s *counterPageState) increment() { s.SetState(func() { s.count++ }) }

func (s *counterPageState) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterial(m.NewCenter(
		m.NewColumn([]m.Widget{
			m.NewText(s.Widget().title),
			m.NewText(strconv.Itoa(s.count), m.TextOpts{Style: m.ThemeOf(bc).TextTheme.HeadlineMedium}),
			m.NewGestureDetector(m.NewText("+"), m.GestureDetectorOpts{OnTap: s.increment}),
		}, m.ColumnOpts{MainAxisAlignment: m.MainAxisAlignmentCenter}),
	))
}

func TestOneImportApp(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(newMyApp())
	for i := 1; i <= 3; i++ {
		ts.Tap(gluttertest.ByText("+"))
		ts.Pump()
		if ts.Count(gluttertest.ByText(strconv.Itoa(i))) != 1 {
			t.Fatalf("the text %d is not shown after %d taps", i, i)
		}
	}
}

func TestReexports(t *testing.T) {
	// A generic type, a generic function, a constant, a variable, an enum and a
	// function of each of the libraries that Flutter's material.dart exports.
	tween := m.NewTween(0.0, 10.0)
	if tween.Transform(0.5) != 5 {
		t.Error("a generic type of animation is not re-exported")
	}
	opt := m.Some(3)
	_ = m.Opt[int](opt)
	if opt.Value() != 3 {
		t.Errorf("Some(3) = %v", opt)
	}
	if m.Curves.Linear.Transform(0.5) != 0.5 {
		t.Error("Curves is not re-exported")
	}
	if m.MainAxisAlignmentCenter == m.MainAxisAlignmentStart {
		t.Error("the enum values are the same")
	}
	if got := (m.Offset{Dx: 1, Dy: 2}).Add(m.OffsetZero); got.Dx != 1 || got.Dy != 2 {
		t.Errorf("Offset = %v", got)
	}
}
