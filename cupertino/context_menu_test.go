package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/context_menu_test.dart and context_menu_action_test.dart:
// the opening by a long press, the builder, the closing by a tap, a drag and a fling, the layout of the open menu in a
// portrait and a landscape view, the overlay that goes away with the menu or when a route is pushed, and the actions.
// The goldens of the open menu are in dialog_golden_test.go; the semantics cases are not ported (SPEC §12.10, step 34.6).

const (
	menuOpenScale = 1.15
)

func menuChild() widgets.Widget {
	return widgets.NewContainer(nil, widgets.ContainerOpts{Width: foundation.Some(300.0), Height: foundation.Some(100.0), Color: foundation.Some(ui.Color(cupertino.CupertinoColors.ActiveOrange.EffectiveColor()))})
}

func menuActions(n int) []widgets.Widget {
	var actions []widgets.Widget
	for i := range n {
		actions = append(actions, cupertino.NewCupertinoContextMenuAction(widgets.NewText("Action "+string(rune('0'+i)))))
	}
	return actions
}

func menuApp(alignment painting.Alignment, child widgets.Widget, actions ...widgets.Widget) widgets.Widget {
	if len(actions) == 0 {
		actions = []widgets.Widget{cupertino.NewCupertinoContextMenuAction(widgets.NewText("CupertinoContextMenuAction"))}
	}
	return testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(
		widgets.NewAlign(cupertino.NewCupertinoContextMenu(actions, child), widgets.AlignOpts{Alignment: alignment}))})
}

func rectOf(ts *gluttertest.Tester, f gluttertest.Finder) ui.Rect {
	tl, sz := ts.TopLeft(f), ts.Size(f)
	return ui.NewRectFromLTWH(tl.Dx, tl.Dy, sz.Width, sz.Height)
}

func decoy() gluttertest.Finder  { return gluttertest.ByType[*cupertino.ContextMenuDecoyChild]() }
func static() gluttertest.Finder { return gluttertest.ByType[*cupertino.ContextMenuRouteStatic]() }

// openMenu presses on the child until the menu opens, and lets go.
func openMenu(ts *gluttertest.Tester, child widgets.Widget) {
	g := ts.StartGesture(ts.Center(gluttertest.ByWidget(child)))
	ts.PumpAndSettle()
	g.Up()
	ts.PumpAndSettle()
}

func TestCupertinoContextMenuOpening(t *testing.T) {
	t.Run("an unopened menu draws the child where it would be without it", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(child))}))
		want := rectOf(ts, gluttertest.ByWidget(child))
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		if got := rectOf(ts, gluttertest.ByWidget(child)); got != want {
			t.Errorf("rect %v, want %v", got, want)
		}
	})
	t.Run("a press and hold opens the menu", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		childRect := rectOf(ts, gluttertest.ByWidget(child))
		if ts.Exists(decoy()) {
			t.Fatal("a decoy before the press")
		}
		g := ts.StartGesture(childRect.Center())
		ts.Pump()
		inDecoy := gluttertest.Descendant(decoy(), gluttertest.ByWidget(child))
		if got := rectOf(ts, inDecoy); got != childRect {
			t.Errorf("the decoy %v is not on the child %v", got, childRect)
		}
		ts.Pump(400 * time.Millisecond)
		if got := rectOf(ts, inDecoy); got == childRect {
			t.Error("the decoy did not start to animate")
		}
		ts.Pump(800 * time.Millisecond)
		if got := rectOf(ts, inDecoy); got.Width() != childRect.Width()*menuOpenScale {
			t.Errorf("decoy width %v, want %v", got.Width(), childRect.Width()*menuOpenScale)
		}
		ts.PumpAndSettle()
		g.Up()
		ts.PumpAndSettle()
		if !ts.Exists(static()) {
			t.Error("the menu is not open")
		}
	})
	t.Run("a short press does not open the menu and the child comes back", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		g := ts.StartGesture(ts.Center(gluttertest.ByWidget(child)))
		ts.Pump(100 * time.Millisecond)
		g.Up()
		ts.PumpAndSettle()
		if ts.Exists(static()) || ts.Exists(decoy()) {
			t.Error("a menu or a decoy after a short press")
		}
		if !ts.Exists(gluttertest.ByWidget(child)) {
			t.Error("the child is gone")
		}
	})
	t.Run("a builder can change the animation", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(
			cupertino.NewCupertinoContextMenuBuilder(menuActions(1), func(bc widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
				return widgets.NewContainer(nil, widgets.ContainerOpts{Width: foundation.Some(300.0), Height: foundation.Some(100.0),
					Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFFF9500)),
						BorderRadius: painting.NewBorderRadiusCircular(25.0 * a.Value())})})
			})))}))
		inTickerMode := gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*widgets.TickerMode](), gluttertest.ByType[*widgets.Container]()))
		g := ts.StartGesture(ts.Center(inTickerMode))
		ts.Pump()
		radius := func() painting.BorderRadiusGeometry {
			c := ts.Widget(gluttertest.First(gluttertest.Descendant(decoy(), gluttertest.ByType[*widgets.Container]()))).(*widgets.Container)
			return c.Decoration().(*painting.BoxDecoration).BorderRadius
		}
		first := radius()
		ts.Pump(500 * time.Millisecond)
		if later := radius(); later == first {
			t.Errorf("the border radius did not change: %v", later)
		}
		ts.PumpAndSettle()
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("the haptic feedback is given when the menu opens, if enabled", func(t *testing.T) {
		for _, enabled := range []bool{false, true} {
			ts := dialogTester(t, ui.Size{})
			var kinds []ui.HapticFeedbackKind
			ts.Binding.HapticFeedback().OnPerform = func(k ui.HapticFeedbackKind) { kinds = append(kinds, k) }
			child := menuChild()
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(
				cupertino.NewCupertinoContextMenu(menuActions(1), child, cupertino.CupertinoContextMenuOpts{EnableHapticFeedback: enabled})))}))
			openMenu(ts, child)
			if got := len(kinds) > 0; got != enabled {
				t.Errorf("enabled %v: feedback %v", enabled, kinds)
			}
		}
	})
}

func TestCupertinoContextMenuOpen(t *testing.T) {
	staticChild := func(child widgets.Widget) gluttertest.Finder {
		return gluttertest.Descendant(static(), gluttertest.ByWidget(child))
	}
	t.Run("a tap on the background closes the menu", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		openMenu(ts, child)
		if !ts.Exists(static()) {
			t.Fatal("the menu is not open")
		}
		ts.TapAt(ui.Offset{Dx: 1, Dy: 1})
		ts.PumpAndSettle()
		if ts.Exists(static()) {
			t.Error("the menu is still open")
		}
	})
	t.Run("a drag down closes the menu only when it goes far enough", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		openMenu(ts, child)
		center := ts.Center(staticChild(child))
		g := ts.StartGesture(center)
		g.MoveByAt(ui.Offset{Dy: 100}, 100*time.Millisecond)
		ts.Pump()
		g.Up()
		ts.Pump()
		if ts.Center(staticChild(child)).Dy <= center.Dy {
			t.Error("the child did not follow the drag")
		}
		ts.PumpAndSettle()
		if got := ts.Center(staticChild(child)); got != center || !ts.Exists(static()) {
			t.Errorf("the child did not bounce back: %v (want %v)", got, center)
		}
		g = ts.StartGesture(center)
		g.MoveByAt(ui.Offset{Dy: 200}, 100*time.Millisecond)
		ts.Pump()
		g.Up()
		ts.PumpAndSettle()
		if ts.Exists(static()) {
			t.Error("the menu is still open after a long drag")
		}
	})
	t.Run("a fling down closes the menu and a fling up does not", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		openMenu(ts, child)
		ts.Fling(staticChild(child), ui.Offset{Dy: -100}, 1000)
		ts.PumpAndSettle()
		if !ts.Exists(staticChild(child)) {
			t.Fatal("a fling up closed the menu")
		}
		ts.Fling(staticChild(child), ui.Offset{Dy: 100}, 1000)
		ts.PumpAndSettle()
		if ts.Exists(static()) {
			t.Error("the menu is still open after a fling down")
		}
	})
	t.Run("the backdrop is the filter of the route", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		if ts.Exists(gluttertest.ByType[*widgets.BackdropFilter]()) {
			t.Fatal("a backdrop before the menu opens")
		}
		openMenu(ts, child)
		if n := ts.Count(gluttertest.ByType[*widgets.BackdropFilter]()); n != 1 {
			t.Errorf("%d backdrops", n)
		}
	})
	t.Run("the preview has the open border radius", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		openMenu(ts, child)
		clip := gluttertest.Descendant(gluttertest.Descendant(static(), gluttertest.ByType[*widgets.FittedBox]()), gluttertest.ByType[*widgets.ClipRSuperellipse]())
		if n := ts.Count(clip); n != 1 {
			t.Fatalf("%d previews", n)
		}
		got := ts.Widget(clip).(*widgets.ClipRSuperellipse).BorderRadius()
		if want := painting.NewBorderRadiusCircular(cupertino.CupertinoContextMenuKOpenBorderRadius); got != painting.BorderRadiusGeometry(want) {
			t.Errorf("border radius %v, want %v", got, want)
		}
	})
	t.Run("the actions are 250 wide", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child, menuActions(3)...))
		openMenu(ts, child)
		for _, a := range ts.Widgets(gluttertest.ByType[*cupertino.CupertinoContextMenuAction]()) {
			if w := ts.Size(gluttertest.ByWidget(a)).Width; w != 250 {
				t.Errorf("action width %v", w)
			}
		}
	})
	t.Run("the sheet is at the side of the child that is nearer to the edge, in a portrait view", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 600, Height: 800})
		child := menuChild()
		action := gluttertest.ByType[*cupertino.CupertinoContextMenuAction]()
		var left [3]float64
		for i, alignment := range []painting.Alignment{painting.AlignmentCenter, painting.AlignmentCenterLeft, painting.AlignmentCenterRight} {
			ts.PumpWidget(menuApp(alignment, child))
			ts.PumpAndSettle()
			openMenu(ts, child)
			if n := ts.Count(action); n != 1 {
				t.Fatalf("alignment %v: %d actions", alignment, n)
			}
			left[i] = ts.TopLeft(action).Dx
			ts.TapAt(ui.Offset{Dx: 1, Dy: 1})
			ts.PumpAndSettle()
		}
		if !(left[1] < left[0] && left[0] < left[2]) {
			t.Errorf("x of the action at the left, the center and the right: %v", left)
		}
	})
	t.Run("the sheet of a child at the center is at the left, in a landscape view", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		child := menuChild()
		action := gluttertest.ByType[*cupertino.CupertinoContextMenuAction]()
		var x [3]float64
		for i, alignment := range []painting.Alignment{painting.AlignmentCenter, painting.AlignmentCenterLeft, painting.AlignmentCenterRight} {
			ts.PumpWidget(menuApp(alignment, child))
			ts.PumpAndSettle()
			openMenu(ts, child)
			x[i] = ts.TopLeft(action).Dx
			ts.TapAt(ui.Offset{Dx: 1, Dy: 1})
			ts.PumpAndSettle()
		}
		if x[0] != x[1] || !(x[2] < x[1]) {
			t.Errorf("x of the action at the center, the left and the right: %v", x)
		}
	})
	t.Run("the sheet is inside the screen", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 600, Height: 800})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentBottomRight, child, menuActions(12)...))
		openMenu(ts, child)
		sheet := rectOf(ts, gluttertest.First(gluttertest.Ancestor(gluttertest.ByType[*cupertino.CupertinoContextMenuAction](), gluttertest.ByType[*widgets.SingleChildScrollView]())))
		if sheet.Left < 0 || sheet.Right > 600 || sheet.Top < 0 || sheet.Bottom > 800 {
			t.Errorf("the sheet %v is out of the screen", sheet)
		}
	})
}

func TestCupertinoContextMenuOverlay(t *testing.T) {
	t.Run("pushing a route removes the decoy", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		var bc widgets.BuildContext
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
			bc = c
			return widgets.NewCenter(cupertino.NewCupertinoContextMenu(menuActions(1), child))
		}))}))
		g := ts.StartGesture(ts.Center(gluttertest.ByWidget(child)))
		ts.Pump(100 * time.Millisecond)
		if !ts.Exists(decoy()) {
			t.Fatal("no decoy while pressed")
		}
		widgets.NavigatorOf(bc).Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Next") }))
		ts.PumpAndSettle()
		if ts.Exists(decoy()) {
			t.Error("the decoy is still there")
		}
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("removing the menu from the tree removes the decoy", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenter, child))
		g := ts.StartGesture(ts.Center(gluttertest.ByWidget(child)))
		ts.Pump(100 * time.Millisecond)
		if !ts.Exists(decoy()) {
			t.Fatal("no decoy while pressed")
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(widgets.NewText("Gone")))}))
		ts.Pump()
		if ts.Exists(decoy()) {
			t.Error("the decoy is still there")
		}
		g.Up()
		ts.PumpAndSettle()
	})
}

func TestCupertinoContextMenuScale(t *testing.T) {
	t.Run("a child near the edge grows less, to the minimum scale factor", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 600, Height: 800})
		child := menuChild()
		ts.PumpWidget(menuApp(painting.AlignmentCenterLeft, child))
		childRect := rectOf(ts, gluttertest.ByWidget(child))
		g := ts.StartGesture(childRect.Center())
		ts.Pump()
		ts.Pump(1500 * time.Millisecond)
		got := rectOf(ts, gluttertest.Descendant(decoy(), gluttertest.ByWidget(child))).Width() / childRect.Width()
		if got < 1.02 || got > menuOpenScale {
			t.Errorf("scale %v is out of [1.02, 1.15]", got)
		}
		ts.PumpAndSettle()
		g.Up()
		ts.PumpAndSettle()
	})
}

func TestCupertinoContextMenuAction(t *testing.T) {
	actionApp := func(opts cupertino.CupertinoContextMenuActionOpts, theme *cupertino.CupertinoThemeData) widgets.Widget {
		return testApp(cupertino.CupertinoAppOpts{Theme: theme, Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(
			cupertino.NewCupertinoContextMenuAction(widgets.NewText("Action"), opts)))})
	}
	styleOf := func(ts *gluttertest.Tester) *widgets.DefaultTextStyle {
		return ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Action"), gluttertest.ByType[*widgets.DefaultTextStyle]()))).(*widgets.DefaultTextStyle)
	}
	t.Run("a tap calls onPressed", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pressed := false
		ts.PumpWidget(actionApp(cupertino.CupertinoContextMenuActionOpts{OnPressed: func() { pressed = true }}, nil))
		ts.Tap(gluttertest.ByText("Action"))
		if !pressed {
			t.Error("not pressed")
		}
	})
	t.Run("the background is darker while the action is pressed", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(actionApp(cupertino.CupertinoContextMenuActionOpts{OnPressed: func() {}}, nil))
		color := func() ui.Color {
			return ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Action"), gluttertest.ByType[*widgets.ColoredBox]()))).(*widgets.ColoredBox).Color()
		}
		before := color()
		g := ts.StartGesture(ts.Center(gluttertest.ByText("Action")))
		ts.Pump()
		if pressed := color(); pressed == before || pressed != ui.NewColor(0xFFDDDDDD) {
			t.Errorf("color %v, before %v", pressed, before)
		}
		g.Up()
		ts.Pump()
		if color() != before {
			t.Error("the color did not come back")
		}
	})
	t.Run("the colors of the text and the icon are right", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		icon := widgets.NewIcon(cupertino.CupertinoIcons.Share())
		_ = icon
		ts.PumpWidget(actionApp(cupertino.CupertinoContextMenuActionOpts{TrailingIcon: foundation.Some(cupertino.CupertinoIcons.Share())}, nil))
		style := styleOf(ts).Style()
		if c, _ := style.Color.Get(); c != ui.NewColor(0xFF000000) {
			t.Errorf("text color %v", c)
		}
		if got, _ := ts.Widget(gluttertest.ByType[*widgets.Icon]()).(*widgets.Icon).Color().Get(); got != ui.NewColor(0xFF000000) {
			t.Errorf("icon color %v", got)
		}
		ts.PumpWidget(actionApp(cupertino.CupertinoContextMenuActionOpts{IsDestructiveAction: true,
			TrailingIcon: foundation.Some(cupertino.CupertinoIcons.Share())}, nil))
		red := cupertino.CupertinoColors.DestructiveRed.EffectiveColor()
		if c, _ := styleOf(ts).Style().Color.Get(); c != red {
			t.Errorf("destructive text color %v, want %v", c, red)
		}
		if got, _ := ts.Widget(gluttertest.ByType[*widgets.Icon]()).(*widgets.Icon).Color().Get(); got != red {
			t.Errorf("destructive icon color %v", got)
		}
	})
	t.Run("the default action is bold and takes the label color of the brightness", func(t *testing.T) {
		for _, tt := range []struct {
			theme *cupertino.CupertinoThemeData
			color ui.Color
		}{{testTheme(), ui.NewColor(0xFF000000)}, {darkTheme(), ui.NewColor(0xFFFFFFFF)}} {
			ts := dialogTester(t, ui.Size{})
			ts.PumpWidget(actionApp(cupertino.CupertinoContextMenuActionOpts{IsDefaultAction: true}, tt.theme))
			style := styleOf(ts).Style()
			if c, _ := style.Color.Get(); c != tt.color || style.FontWeight != ui.FontWeightW600 {
				t.Errorf("color %v weight %v", c, style.FontWeight)
			}
		}
	})
	t.Run("it does not crash at zero area", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBox(
			cupertino.NewCupertinoContextMenuAction(widgets.NewText("X")), widgets.SizedBoxOpts{Width: foundation.Some(0.0), Height: foundation.Some(0.0)}))}))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
}

func TestCupertinoContextMenuNoActionsRoute(t *testing.T) {
	skipUnlessDebug(t)
	requireErrorTag(t, recoverError(func() { cupertino.NewCupertinoContextMenu(nil, menuChild()) }), "NewCupertinoContextMenu.CupertinoContextMenuNoActions")
	requireErrorTag(t, recoverError(func() { cupertino.NewCupertinoContextMenuBuilder(nil, nil) }), "NewCupertinoContextMenuBuilder.CupertinoContextMenuNoActions")
}
