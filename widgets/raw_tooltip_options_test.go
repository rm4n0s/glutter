package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// The options of RawTooltip that Glutter adds (step 27.6): the hover delay read when the
// mouse enters, OnShown and OnHidden, the controller and DismissOnKey.

// pumpTooltipWith pumps a 100x100 tooltip target in the middle of the view with the options.
func pumpTooltipWith(ts *gluttertest.Tester, o RawTooltipOpts) gluttertest.Finder {
	ts.Hover(ui.Offset{Dx: 1, Dy: 1})
	ts.Pump()
	ts.Hover(ui.Offset{})
	o.PositionDelegate = below24
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewCenter(NewRawTooltip(foundation.Some(tooltipText),
		func(BuildContext, animation.Animation[float64]) Widget { return ltrText(tooltipText) },
		NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}), o))}))
	return gluttertest.ByType[*RawTooltip]()
}

func TestRawTooltipHoverDelayFuncIsReadWhenTheMouseEnters(t *testing.T) {
	tests := []struct {
		name       string
		delays     []time.Duration // the value of the function at each enter
		hoverDelay time.Duration
		// shownAfter is, for each enter, whether the tooltip shows after 10 ms.
		shownAfter []bool
	}{
		{"a long delay", []time.Duration{500 * time.Millisecond}, 0, []bool{false}},
		{"no delay", []time.Duration{0}, 500 * time.Millisecond, []bool{true}},
		{"the function wins over HoverDelay", []time.Duration{500 * time.Millisecond}, 0, []bool{false}},
		{"read at each enter", []time.Duration{500 * time.Millisecond, 0, 500 * time.Millisecond}, 0, []bool{false, true, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			enters := 0
			tooltip := pumpTooltipWith(ts, RawTooltipOpts{
				HoverDelay: tt.hoverDelay,
				HoverDelayFunc: func() time.Duration {
					d := tt.delays[enters]
					enters++
					return d
				},
			})
			for i, want := range tt.shownAfter {
				ts.Hover(ts.Center(tooltip))
				ts.Pump()
				ts.Pump(10 * time.Millisecond)
				if got := tooltipShown(ts); got != want {
					t.Errorf("enter %d: shown after 10 ms = %v, want %v", i, got, want)
				}
				if want {
					// It is hidden again before the next enter.
					ts.Hover(ui.Offset{})
					ts.Pump(time.Second)
					ts.PumpAndSettle()
				} else {
					ts.Hover(ui.Offset{})
					ts.Pump()
					ts.PumpAndSettle()
				}
				if tooltipShown(ts) {
					t.Fatalf("enter %d: the tooltip stayed after the mouse left", i)
				}
			}
			if enters != len(tt.delays) {
				t.Errorf("the function was read %d times, want %d (once for each enter)", enters, len(tt.delays))
			}
		})
	}
}

func TestRawTooltipCallbacksForEveryWayOfHiding(t *testing.T) {
	type way struct {
		name string
		opts RawTooltipOpts
		// hide hides the tooltip.
		hide func(t *testing.T, ts *gluttertest.Tester, tooltip gluttertest.Finder, c *RawTooltipController)
	}
	ways := []way{
		{"the mouse leaves", RawTooltipOpts{}, func(t *testing.T, ts *gluttertest.Tester, _ gluttertest.Finder, _ *RawTooltipController) {
			ts.Hover(ui.Offset{})
			ts.Pump()
		}},
		{"a tap elsewhere", RawTooltipOpts{}, func(t *testing.T, ts *gluttertest.Tester, _ gluttertest.Finder, _ *RawTooltipController) {
			ts.TapAt(ui.Offset{Dx: 5, Dy: 5})
			ts.Pump()
		}},
		{"RawTooltipDismissAll", RawTooltipOpts{}, func(t *testing.T, ts *gluttertest.Tester, tooltip gluttertest.Finder, _ *RawTooltipController) {
			RawTooltipDismissAll(ts.Element(tooltip))
			ts.Pump()
		}},
		{"the controller", RawTooltipOpts{}, func(t *testing.T, ts *gluttertest.Tester, _ gluttertest.Finder, c *RawTooltipController) {
			c.Hide()
			ts.Pump()
		}},
		{"a key", RawTooltipOpts{DismissOnKey: true}, func(t *testing.T, ts *gluttertest.Tester, _ gluttertest.Finder, _ *RawTooltipController) {
			ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyA)
			ts.Pump()
		}},
		{"the widget leaves the tree", RawTooltipOpts{}, func(t *testing.T, ts *gluttertest.Tester, _ gluttertest.Finder, _ *RawTooltipController) {
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewCenter(NewSizedBox(nil))}))
		}},
	}
	for _, w := range ways {
		t.Run(w.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var shown, hidden int
			c := NewRawTooltipController()
			o := w.opts
			o.Controller, o.OnShown, o.OnHidden = c, func() { shown++ }, func() { hidden++ }
			tooltip := pumpTooltipWith(ts, o)
			if shown != 0 || hidden != 0 {
				t.Fatalf("callbacks before the tooltip showed: %d, %d", shown, hidden)
			}
			ts.Hover(ts.Center(tooltip))
			ts.Pump()
			ts.Pump(0)
			if !tooltipShown(ts) || shown != 1 || hidden != 0 || !c.IsShowing() {
				t.Fatalf("shown = %v, OnShown %d, OnHidden %d, IsShowing %v", tooltipShown(ts), shown, hidden, c.IsShowing())
			}
			w.hide(t, ts, tooltip, c)
			ts.PumpAndSettle()
			if tooltipShown(ts) && w.name != "the widget leaves the tree" {
				t.Errorf("the tooltip is still shown")
			}
			if shown != 1 || hidden != 1 {
				t.Errorf("OnShown %d times, OnHidden %d times, want 1 and 1", shown, hidden)
			}
		})
	}
}

func TestRawTooltipController(t *testing.T) {
	t.Run("a mounted tooltip", func(t *testing.T) {
		ts := gluttertest.New(t)
		c := NewRawTooltipController()
		if c.IsShowing() {
			t.Error("a controller of no tooltip says it shows")
		}
		c.Show() // nothing is attached: nothing happens
		c.Hide()
		pumpTooltipWith(ts, RawTooltipOpts{Controller: c})
		if c.IsShowing() || tooltipShown(ts) {
			t.Fatal("the tooltip shows before Show")
		}
		c.Show()
		ts.Pump()
		ts.Pump(0)
		if !c.IsShowing() || !tooltipShown(ts) {
			t.Fatalf("Show did not show the tooltip: IsShowing %v", c.IsShowing())
		}
		ts.Pump(24 * time.Hour) // nothing hides it
		if !c.IsShowing() {
			t.Error("the tooltip went away on its own")
		}
		c.Hide()
		ts.PumpAndSettle()
		if c.IsShowing() || tooltipShown(ts) {
			t.Error("Hide did not hide the tooltip")
		}
	})
	t.Run("an unmounted tooltip", func(t *testing.T) {
		ts := gluttertest.New(t)
		c := NewRawTooltipController()
		pumpTooltipWith(ts, RawTooltipOpts{Controller: c})
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(nil))}))
		c.Show()
		c.Hide()
		ts.Pump()
		if c.IsShowing() || tooltipShown(ts) {
			t.Error("a controller of a tooltip that left the tree did something")
		}
	})
	t.Run("the controller is replaced", func(t *testing.T) {
		ts := gluttertest.New(t)
		first, second := NewRawTooltipController(), NewRawTooltipController()
		pumpTooltipWith(ts, RawTooltipOpts{Controller: first})
		pumpTooltipWith(ts, RawTooltipOpts{Controller: second})
		first.Show()
		ts.Pump()
		if tooltipShown(ts) || first.IsShowing() {
			t.Error("the old controller still works the tooltip")
		}
		second.Show()
		ts.Pump()
		ts.Pump(0)
		if !tooltipShown(ts) || !second.IsShowing() {
			t.Error("the new controller does not")
		}
	})
}

func TestRawTooltipDismissOnKey(t *testing.T) {
	tests := []struct {
		name         string
		dismissOnKey bool
		wantShown    bool
	}{
		{"hides on a key down", true, false},
		{"without the option the key leaves it", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			tooltip := pumpTooltipWith(ts, RawTooltipOpts{DismissOnKey: tt.dismissOnKey})
			ts.Hover(ts.Center(tooltip))
			ts.Pump()
			ts.Pump(0)
			if !tooltipShown(ts) {
				t.Fatal("hover did not show the tooltip")
			}
			// The key is not handled by the tooltip: it goes on to the focused widget.
			if ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyA) {
				t.Error("the key was handled")
			}
			ts.PumpAndSettle()
			if got := tooltipShown(ts); got != tt.wantShown {
				t.Errorf("shown after the key = %v, want %v", got, tt.wantShown)
			}
		})
	}
	t.Run("a key outside a showing tooltip does nothing", func(t *testing.T) {
		ts := gluttertest.New(t)
		var hidden int
		pumpTooltipWith(ts, RawTooltipOpts{DismissOnKey: true, OnHidden: func() { hidden++ }})
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyA)
		ts.PumpAndSettle()
		if hidden != 0 {
			t.Errorf("OnHidden was called %d times for a tooltip that never showed", hidden)
		}
	})
	t.Run("the key reaches the focused widget", func(t *testing.T) {
		ts := gluttertest.New(t)
		var got []services.LogicalKeyboardKey
		node := NewFocusNode(FocusNodeOpts{})
		defer node.Dispose()
		node.SetOnKeyEvent(func(_ *FocusNode, e services.KeyEvent) KeyEventResult {
			if _, ok := e.(*services.KeyDownEvent); ok {
				got = append(got, e.LogicalKey())
			}
			return KeyEventResultHandled
		})
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewFocus(NewCenter(NewRawTooltip(foundation.Some(tooltipText),
			func(BuildContext, animation.Animation[float64]) Widget { return ltrText(tooltipText) },
			NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
			RawTooltipOpts{PositionDelegate: below24, DismissOnKey: true})), FocusOpts{FocusNode: node, Autofocus: true})}))
		ts.Pump()
		ts.Hover(ts.Center(gluttertest.ByType[*RawTooltip]()))
		ts.Pump()
		ts.Pump(0)
		if !tooltipShown(ts) {
			t.Fatal("hover did not show the tooltip")
		}
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyA)
		ts.PumpAndSettle()
		if tooltipShown(ts) || len(got) != 1 || got[0] != services.LogicalKeyboardKeyKeyA {
			t.Errorf("shown = %v, the focused widget got %v", tooltipShown(ts), got)
		}
	})
}
