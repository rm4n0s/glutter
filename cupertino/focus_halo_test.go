package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/cupertino_focus_halo_test.dart: the halo draws
// the outline of the focus when a widget inside it has the focus, and none otherwise.

func TestCupertinoFocusHalo(t *testing.T) {
	expected := func(hasFocus bool) painting.BorderSide {
		if !hasFocus {
			return painting.BorderSideNone
		}
		return painting.NewBorderSide(painting.BorderSideOpts{
			Color: foundation.Some(painting.NewHSLColorFromColor(cupertino.CupertinoColors.ActiveBlue.Color.WithOpacity(cupertino.CupertinoFocusColorOpacity)).
				WithLightness(cupertino.CupertinoFocusColorBrightness).WithSaturation(cupertino.CupertinoFocusColorSaturation).ToColor()),
			Width: foundation.Some(3.5)})
	}
	radius := painting.NewBorderRadiusAll(ui.NewRadiusCircular(10))
	for _, tt := range []struct {
		name string
		make func(child widgets.Widget) widgets.Widget
	}{
		{"rect", func(c widgets.Widget) widgets.Widget { return cupertino.NewCupertinoFocusHaloWithRect(c) }},
		{"rrect", func(c widgets.Widget) widgets.Widget { return cupertino.NewCupertinoFocusHaloWithRRect(radius, c) }},
		{"superellipse", func(c widgets.Widget) widgets.Widget {
			return cupertino.NewCupertinoFocusHaloWithRoundedSuperellipse(radius, c)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			a := widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "a"})
			b := widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "b"})
			t.Cleanup(a.Dispose)
			t.Cleanup(b.Dispose)
			focusable := func(n *widgets.FocusNode) widgets.Widget {
				return widgets.NewFocus(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(50.0)}), widgets.FocusOpts{FocusNode: n})
			}
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewColumn([]widgets.Widget{
				tt.make(focusable(a)), tt.make(focusable(b)),
			})}))
			halos := func() []widgets.Widget { return ts.Widgets(gluttertest.ByType[*widgets.DecoratedBox]()) }
			sides := func() []painting.BorderSide {
				var out []painting.BorderSide
				for _, w := range halos() {
					if sd, ok := w.(*widgets.DecoratedBox).Decoration().(*painting.ShapeDecoration); ok {
						if ob, ok := sd.Shape.(painting.OutlinedBorder); ok {
							out = append(out, ob.Side())
						}
					}
				}
				return out
			}
			for _, side := range sides() {
				if side != expected(false) {
					t.Fatalf("unfocused side = %v", side)
				}
			}
			a.RequestFocus()
			ts.PumpAndSettle()
			got := sides()
			if len(got) != 2 || got[0] != expected(true) || got[1] != expected(false) {
				t.Errorf("sides with the first focused = %v", got)
			}
			b.RequestFocus()
			ts.PumpAndSettle()
			got = sides()
			if len(got) != 2 || got[0] != expected(false) || got[1] != expected(true) {
				t.Errorf("sides with the second focused = %v", got)
			}
		})
	}
}
