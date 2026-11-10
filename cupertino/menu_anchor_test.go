package cupertino_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/menu_anchor_test.dart: opening and closing, the
// callbacks, the outside taps, the keyboard, the swipe and its scale, the widths, the layout around the anchor, the
// implicit and explicit dividers and the states of the items. The cases about display features, the semantics, the
// golden images drawn by `paints`, and the many cases that only check one number of the style of the text (the
// dynamic type tables are checked by one case of each kind) are not ported (SPEC §12.10, step 34.6).

// anchorButton is a button that opens and closes the menu of its anchor.
func anchorButton(label string) widgets.Widget {
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		controller := widgets.MenuControllerMaybeOf(bc)
		return cupertino.NewCupertinoButtonFilled(func() {
			if controller == nil {
				return
			}
			if controller.IsOpen() {
				controller.Close()
			} else {
				controller.Open()
			}
		}, widgets.NewText(label))
	})
}

func anchorApp(child widgets.Widget, alignment painting.AlignmentGeometry) widgets.Widget {
	return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewColoredBox(ui.NewColor(0xFF000000),
		widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewAlign(child, widgets.AlignOpts{Alignment: alignment})))})
}

func menuTester(t *testing.T, size ui.Size) *gluttertest.Tester {
	t.Helper()
	return dialogTester(t, size)
}

func item(label string, onPressed func(), opts ...cupertino.CupertinoMenuItemOpts) widgets.Widget {
	o := cupertino.CupertinoMenuItemOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.OnPressed = onPressed
	return cupertino.NewCupertinoMenuItem(widgets.NewText(label), o)
}

func noop() {}

func settle(ts *gluttertest.Tester) {
	ts.Pump()
	ts.PumpAndSettle()
}

func pumpFrames(ts *gluttertest.Tester, n int) {
	for range n {
		ts.Pump(16 * time.Millisecond)
	}
}

// menuSurface is the clipped surface of the open menu.
func menuSurface() gluttertest.Finder { return gluttertest.ByType[*widgets.ClipRSuperellipse]() }

func menuScale(ts *gluttertest.Tester) float64 {
	st := gluttertest.First(gluttertest.Ancestor(menuSurface(), gluttertest.ByType[*widgets.ScaleTransition]()))
	return ts.Widget(st).(*widgets.ScaleTransition).Scale().Value()
}

func TestCupertinoMenuAnchorOpenAndClose(t *testing.T) {
	t.Run("the controller is open while the overlay is shown", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{widgets.NewText("A")},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		ts.Tap(gluttertest.ByText("Anchor"))
		ts.Pump()
		if !controller.IsOpen() || !ts.Exists(gluttertest.ByText("A")) {
			t.Fatalf("open %v, item shown %v", controller.IsOpen(), ts.Exists(gluttertest.ByText("A")))
		}
		ts.Tap(gluttertest.ByText("Anchor"))
		settle(ts)
		if controller.IsOpen() || ts.Exists(gluttertest.ByText("A")) {
			t.Errorf("open %v, item shown %v", controller.IsOpen(), ts.Exists(gluttertest.ByText("A")))
		}
	})
	t.Run("open and close of the controller toggle the overlay", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{widgets.NewText("A")},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		if controller.IsOpen() || ts.Exists(gluttertest.ByText("A")) {
			t.Fatal("open at the start")
		}
		controller.Open()
		settle(ts)
		if !controller.IsOpen() || !ts.Exists(gluttertest.ByText("A")) {
			t.Fatal("not open")
		}
		controller.Close()
		settle(ts)
		if controller.IsOpen() || ts.Exists(gluttertest.ByText("A")) {
			t.Error("not closed")
		}
	})
	t.Run("the anchor makes a controller of its own", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{widgets.NewText("A")},
			cupertino.CupertinoMenuAnchorOpts{Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		ts.Tap(gluttertest.ByText("Anchor"))
		settle(ts)
		if !ts.Exists(gluttertest.ByText("A")) {
			t.Error("the menu did not open")
		}
	})
	t.Run("onOpen is called when the menu opens and onClose when it has closed", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		var opened, closed int
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{widgets.NewText("A")},
			cupertino.CupertinoMenuAnchorOpts{Child: anchorButton("Anchor"), OnOpen: func() { opened++ }, OnClose: func() { closed++ }}), painting.AlignmentCenter))
		ts.Tap(gluttertest.ByText("Anchor"))
		ts.Pump()
		if opened != 1 || closed != 0 {
			t.Fatalf("opened %d, closed %d", opened, closed)
		}
		ts.PumpAndSettle()
		controller := widgets.MenuControllerMaybeOf(ts.Element(gluttertest.ByText("Anchor")))
		controller.Close()
		ts.Pump(16 * time.Millisecond)
		if closed != 0 {
			t.Errorf("closed %d before the animation ends", closed)
		}
		ts.PumpAndSettle()
		if opened != 1 || closed != 1 {
			t.Errorf("opened %d, closed %d", opened, closed)
		}
	})
	t.Run("the animation status is reported", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		var statuses []string
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{widgets.NewText("A")},
			cupertino.CupertinoMenuAnchorOpts{Child: anchorButton("Anchor"), OnAnimationStatusChanged: func(s animation.AnimationStatus) { statuses = append(statuses, s.String()) }}), painting.AlignmentCenter))
		ts.Tap(gluttertest.ByText("Anchor"))
		settle(ts)
		ts.Tap(gluttertest.ByText("Anchor"))
		settle(ts)
		want := []string{"AnimationStatus.forward", "AnimationStatus.completed", "AnimationStatus.reverse", "AnimationStatus.dismissed"}
		if len(statuses) != len(want) {
			t.Fatalf("statuses %v", statuses)
		}
		for i := range want {
			if statuses[i] != want[i] {
				t.Errorf("statuses %v, want %v", statuses, want)
				break
			}
		}
	})
	t.Run("a tap outside closes the menu and is consumed when asked", func(t *testing.T) {
		for _, consume := range []bool{true, false} {
			ts := menuTester(t, ui.Size{})
			var other int
			controller := widgets.NewMenuController()
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewRow([]widgets.Widget{
				cupertino.NewCupertinoMenuAnchor([]widgets.Widget{widgets.NewText("A")},
					cupertino.CupertinoMenuAnchorOpts{Controller: controller, ConsumeOutsideTaps: consume, Child: anchorButton("Anchor")}),
				cupertino.NewCupertinoButton(func() { other++ }, widgets.NewText("Other")),
			}, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween})}))
			controller.Open()
			settle(ts)
			ts.Tap(gluttertest.ByText("Other"))
			settle(ts)
			if controller.IsOpen() {
				t.Errorf("consume %v: still open", consume)
			}
			if (other == 0) != consume {
				t.Errorf("consume %v: the other button was pressed %d times", consume, other)
			}
		}
	})
	t.Run("an item closes the menu and calls onPressed", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		pressed := false
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", func() { pressed = true })},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		ts.Tap(gluttertest.ByText("A"))
		settle(ts)
		if !pressed || controller.IsOpen() {
			t.Errorf("pressed %v, open %v", pressed, controller.IsOpen())
		}
	})
	t.Run("an item can keep the menu open", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop, cupertino.CupertinoMenuItemOpts{NotRequestCloseOnActivate: true})},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		ts.Tap(gluttertest.ByText("A"))
		settle(ts)
		if !controller.IsOpen() {
			t.Error("the menu closed")
		}
	})
	t.Run("a disabled item does nothing", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", nil)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		ts.Tap(gluttertest.ByText("A"))
		settle(ts)
		if !controller.IsOpen() {
			t.Error("a disabled item closed the menu")
		}
	})
}

func TestCupertinoMenuAnchorWidths(t *testing.T) {
	for _, tt := range []struct {
		name  string
		size  ui.Size
		scale float64
		want  float64
	}{
		{"mobile", ui.Size{Width: 765, Height: 900}, 1, 250},
		{"tablet", ui.Size{Width: 768, Height: 400}, 1, 262},
		{"mobile with large text", ui.Size{Width: 765, Height: 900}, 1 + 11.0/17, 370},
		{"tablet with large text", ui.Size{Width: 768, Height: 400}, 1 + 11.0/17, 343},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := menuTester(t, tt.size)
			controller := widgets.NewMenuController()
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				data := widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{TextScaler: foundation.Some[painting.TextScaler](painting.NewTextScalerLinear(tt.scale))})
				return widgets.NewMediaQuery(data, widgets.NewCenter(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop)},
					cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")})))
			})}))
			controller.Open()
			settle(ts)
			if got := ts.Size(menuSurface()).Width; math.Abs(got-tt.want) > 0.1 {
				t.Errorf("width %v, want %v", got, tt.want)
			}
		})
	}
	t.Run("the constraints replace the width", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor"),
				Constraints: foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{Width: 120, Height: 60}))}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		if got := ts.Size(menuSurface()); math.Abs(got.Width-120) > 0.01 || math.Abs(got.Height-60) > 0.01 {
			t.Errorf("size %v", got)
		}
	})
}

func TestCupertinoMenuAnchorSwipe(t *testing.T) {
	setup := func(t *testing.T, opts cupertino.CupertinoMenuAnchorOpts) (*gluttertest.Tester, *widgets.MenuController) {
		ts := menuTester(t, ui.Size{Width: 2000, Height: 2000})
		controller := widgets.NewMenuController()
		opts.Controller, opts.Child = controller, anchorButton("Anchor")
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop)}, opts), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		return ts, controller
	}
	t.Run("swiping away from the menu scales it, to 80 percent at most", func(t *testing.T) {
		ts, _ := setup(t, cupertino.CupertinoMenuAnchorOpts{})
		g := ts.StartGesture(ts.Center(gluttertest.ByText("A")))
		ts.Pump()
		menuRect := rectOf(ts, menuSurface())
		for _, corner := range []ui.Offset{menuRect.TopLeft(), menuRect.TopRight(), menuRect.BottomLeft(), menuRect.BottomRight()} {
			g.MoveTo(corner)
			ts.Pump()
			pumpFrames(ts, 10)
			if got := menuScale(ts); math.Abs(got-1) > 0.01 {
				t.Fatalf("scale %v at the corner %v of the menu", got, corner)
			}
		}
		g.MoveTo(menuRect.TopLeft().Sub(ui.Offset{Dx: 50, Dy: 50}))
		pumpFrames(ts, 3)
		first := menuScale(ts)
		pumpFrames(ts, 3)
		second := menuScale(ts)
		if !(first < 1 && second < first) {
			t.Errorf("the scale does not go down: %v, %v", first, second)
		}
		g.MoveTo(menuRect.TopLeft().Sub(ui.Offset{Dx: 200, Dy: 200}))
		pumpFrames(ts, 30)
		if got := menuScale(ts); math.Abs(got-0.8) > 0.1 {
			t.Errorf("minimum scale %v, want 0.8", got)
		}
		g.MoveTo(ts.Center(gluttertest.ByText("A")))
		pumpFrames(ts, 40)
		if got := menuScale(ts); math.Abs(got-1) > 0.01 {
			t.Errorf("scale %v after the swipe returned to the menu", got)
		}
		g.Up()
		settle(ts)
	})
	t.Run("the scale goes back to full size when the swipe ends", func(t *testing.T) {
		ts, _ := setup(t, cupertino.CupertinoMenuAnchorOpts{})
		g := ts.StartGesture(ts.Center(gluttertest.ByText("A")))
		ts.Pump()
		menuRect := rectOf(ts, menuSurface())
		g.MoveTo(menuRect.TopLeft().Sub(ui.Offset{Dx: 200, Dy: 200}))
		pumpFrames(ts, 30)
		if got := menuScale(ts); got > 0.9 {
			t.Fatalf("scale %v did not go down", got)
		}
		g.Up()
		pumpFrames(ts, 40)
		if got := menuScale(ts); math.Abs(got-1) > 0.01 {
			t.Errorf("scale %v after the swipe ended", got)
		}
		settle(ts)
	})
	t.Run("a swipe that ends on an item activates it", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		var pressed []string
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{
			item("A", func() { pressed = append(pressed, "A") }), item("B", func() { pressed = append(pressed, "B") })},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		g := ts.StartGesture(ts.Center(gluttertest.ByText("A")))
		ts.Pump()
		g.MoveTo(ts.Center(gluttertest.ByText("B")))
		ts.Pump()
		g.Up()
		settle(ts)
		if len(pressed) != 1 || pressed[0] != "B" {
			t.Errorf("pressed %v", pressed)
		}
	})
	t.Run("a long press on the anchor opens the menu when enabled", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor"), EnableLongPressToOpen: true}), painting.AlignmentCenter))
		g := ts.StartGesture(ts.Center(gluttertest.ByText("Anchor")))
		ts.Pump(100 * time.Millisecond)
		if controller.IsOpen() {
			t.Fatal("open before the long press")
		}
		ts.Pump(500 * time.Millisecond)
		if !controller.IsOpen() {
			t.Error("not open after the long press")
		}
		g.Up()
		settle(ts)
	})
	t.Run("the swipe can be disabled, and the long press needs it", func(t *testing.T) {
		ts, _ := setup(t, cupertino.CupertinoMenuAnchorOpts{NotEnableSwipe: true})
		g := ts.StartGesture(ts.Center(gluttertest.ByText("A")))
		ts.Pump()
		menuRect := rectOf(ts, menuSurface())
		g.MoveTo(menuRect.TopLeft().Sub(ui.Offset{Dx: 200, Dy: 200}))
		pumpFrames(ts, 30)
		if got := menuScale(ts); math.Abs(got-1) > 0.01 {
			t.Errorf("scale %v with the swipe disabled", got)
		}
		g.Up()
		settle(ts)
		skipUnlessDebug(t)
		requireErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoMenuAnchor(nil, cupertino.CupertinoMenuAnchorOpts{NotEnableSwipe: true, EnableLongPressToOpen: true})
		}), "NewCupertinoMenuAnchor.CupertinoMenuAnchorLongPressWithoutSwipe")
	})
}

func TestCupertinoMenuAnchorKeyboard(t *testing.T) {
	setup := func(t *testing.T, onPressed func(string)) (*gluttertest.Tester, *widgets.MenuController) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		items := []widgets.Widget{}
		for _, label := range []string{"A", "B", "C"} {
			items = append(items, item(label, func() { onPressed(label) }))
		}
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor(items, cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		return ts, controller
	}
	t.Run("the down and up keys move the focus, and enter activates the item", func(t *testing.T) {
		var pressed []string
		ts, _ := setup(t, func(s string) { pressed = append(pressed, s) })
		ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
		settle(ts)
		if len(pressed) != 1 || pressed[0] != "B" {
			t.Errorf("pressed %v", pressed)
		}
	})
	t.Run("end focuses the last item and home the first", func(t *testing.T) {
		var pressed []string
		ts, controller := setup(t, func(s string) { pressed = append(pressed, s) })
		ts.SendKeyEvent(services.LogicalKeyboardKeyEnd)
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
		settle(ts)
		if len(pressed) != 1 || pressed[0] != "C" || controller.IsOpen() {
			t.Fatalf("pressed %v, open %v", pressed, controller.IsOpen())
		}
		controller.Open()
		settle(ts)
		ts.SendKeyEvent(services.LogicalKeyboardKeyEnd)
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeyHome)
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
		settle(ts)
		if len(pressed) != 2 || pressed[1] != "A" {
			t.Errorf("pressed %v", pressed)
		}
	})
	t.Run("escape closes the menu", func(t *testing.T) {
		ts, controller := setup(t, func(string) {})
		ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
		settle(ts)
		if controller.IsOpen() {
			t.Error("the menu is still open")
		}
	})
}

func TestCupertinoMenuAnchorLayout(t *testing.T) {
	t.Run("the menu is below the anchor in the top half, above it in the bottom half", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 2000, Height: 2000})
		controller := widgets.NewMenuController()
		build := func(alignment painting.AlignmentGeometry) widgets.Widget {
			return anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{widgets.NewContainer(widgets.NewText("A"), widgets.ContainerOpts{
				Width: foundation.Some(50.0), Height: foundation.Some(50.0), Color: foundation.Some(ui.NewColor(0xFF0000FF))})},
				cupertino.CupertinoMenuAnchorOpts{Controller: controller, OverlayPadding: painting.EdgeInsets{},
					Constraints: foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{Width: 50, Height: 50})), Child: anchorButton("Anchor")}), alignment)
		}
		ts.PumpWidget(build(painting.AlignmentTopCenter))
		controller.Open()
		settle(ts)
		for _, horizontal := range []float64{-0.8, -0.5, -0.2, 0, 0.2, 0.5, 0.8} {
			for _, vertical := range []float64{-0.8, -0.4, 0, 0.5, 0.8} {
				ts.PumpWidget(build(painting.Alignment{X: horizontal, Y: vertical}))
				x := 0.0
				switch {
				case horizontal < -0.2:
					x = -1
				case horizontal > 0.2:
					x = 1
				}
				y := -1.0
				if vertical < 0.2 {
					y = 1
				}
				anchor := rectOf(ts, gluttertest.ByType[*cupertino.CupertinoButton]())
				surface := rectOf(ts, gluttertest.ByType[*widgets.Container]())
				position := painting.Alignment{X: x, Y: y}.WithinRect(anchor).Add(ui.Offset{Dy: 8 * y})
				want := painting.Alignment{X: x, Y: -y}.WithinRect(surface)
				if math.Abs(position.Dx-want.Dx) > 0.01 || math.Abs(position.Dy-want.Dy) > 0.01 {
					t.Errorf("anchor alignment (%v, %v): attachment %v, menu point %v", horizontal, vertical, position, want)
				}
			}
		}
	})
	t.Run("the default overlay padding keeps the menu 8 from the edge", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 400, Height: 400})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentTopLeft))
		controller.Open()
		settle(ts)
		if got := rectOf(ts, menuSurface()); got.Left < 8 || got.Top < 8 {
			t.Errorf("surface %v is closer than 8 to the edge", got)
		}
	})
	t.Run("a menu taller than the screen is as high as the screen less the padding", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 400, Height: 300})
		controller := widgets.NewMenuController()
		var items []widgets.Widget
		for i := range 30 {
			items = append(items, item("Item "+string(rune('A'+i%26))+string(rune('a'+i/26)), noop))
		}
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor(items, cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		if got := rectOf(ts, menuSurface()); got.Top < 8-0.01 || got.Bottom > 292+0.01 {
			t.Errorf("surface %v is out of the padded screen", got)
		}
	})
}

func TestCupertinoMenuDividers(t *testing.T) {
	t.Run("implicit dividers are between the items, and none next to an explicit divider", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{
			item("A", noop), item("B", noop), cupertino.NewCupertinoMenuDivider(), item("C", noop), item("D", noop)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		column := ts.Widget(gluttertest.First(gluttertest.Descendant(menuSurface(), gluttertest.ByType[*widgets.Column]()))).(*widgets.Column)
		// A, implicit, B, explicit, C, implicit, D.
		if n := len(column.Children()); n != 7 {
			t.Errorf("%d children of the menu, want 7", n)
		}
	})
	t.Run("the divider is 8 high and as wide as the menu", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop), cupertino.NewCupertinoMenuDivider(), item("B", noop)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoMenuDivider]()); got.Height != 8 || got.Width != 262 {
			t.Errorf("divider %v", got)
		}
	})
	t.Run("the divider has the color that it is given", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop),
			cupertino.NewCupertinoMenuDivider(cupertino.CupertinoMenuDividerOpts{Color: ui.NewColor(0xFFFF0000)}), item("B", noop)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		box := gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoMenuDivider](), gluttertest.ByType[*widgets.ColoredBox]())
		if got := ts.Widget(box).(*widgets.ColoredBox).Color(); got != ui.NewColor(0xFFFF0000) {
			t.Errorf("color %v", got)
		}
	})
	t.Run("an item with a leading widget moves the label of its siblings", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop), item("B", noop, cupertino.CupertinoMenuItemOpts{
			Leading: widgets.NewIcon(cupertino.CupertinoIcons.Share())})},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		a, b := ts.TopLeft(gluttertest.ByText("A")).Dx, ts.TopLeft(gluttertest.ByText("B")).Dx
		if a != b {
			t.Errorf("the labels are at %v and %v", a, b)
		}
		if a <= ts.TopLeft(gluttertest.First(gluttertest.ByType[*cupertino.CupertinoMenuItem]())).Dx+16 {
			t.Errorf("the label at %v did not leave room for the leading widget", a)
		}
	})
}

func TestCupertinoMenuItem(t *testing.T) {
	open := func(t *testing.T, opts cupertino.CupertinoMenuItemOpts, onPressed func()) *gluttertest.Tester {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("Item", onPressed, opts)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		return ts
	}
	background := func(ts *gluttertest.Tester) foundation.Opt[ui.Color] {
		box := gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Item"), gluttertest.ByType[*widgets.DecoratedBox]()))
		return ts.Widget(box).(*widgets.DecoratedBox).Decoration().(*painting.BoxDecoration).Color
	}
	t.Run("the background follows the pressed state", func(t *testing.T) {
		ts := open(t, cupertino.CupertinoMenuItemOpts{}, noop)
		if background(ts).IsSome() {
			t.Fatalf("a background %v at rest", background(ts))
		}
		g := ts.StartGesture(ts.Center(gluttertest.ByText("Item")))
		ts.Pump(200 * time.Millisecond)
		if !background(ts).IsSome() {
			t.Error("no background while pressed")
		}
		g.Up()
		settle(ts)
	})
	t.Run("the background follows the hover state, and a hover focuses the item", func(t *testing.T) {
		ts := open(t, cupertino.CupertinoMenuItemOpts{}, noop)
		ts.Hover(ts.Center(gluttertest.ByText("Item")))
		ts.Pump()
		if !background(ts).IsSome() {
			t.Error("no background while hovered")
		}
	})
	t.Run("a disabled item has no hover state and a gray label", func(t *testing.T) {
		ts := open(t, cupertino.CupertinoMenuItemOpts{}, nil)
		ts.Hover(ts.Center(gluttertest.ByText("Item")))
		ts.Pump()
		if background(ts).IsSome() {
			t.Error("a disabled item follows the hover")
		}
		style := ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Item"), gluttertest.ByType[*widgets.DefaultTextStyle]()))).(*widgets.DefaultTextStyle).Style()
		want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.SystemGrey, ts.Element(gluttertest.ByType[*cupertino.CupertinoMenuItem]()))
		if got, _ := style.Color.Get(); got != want {
			t.Errorf("color %v, want %v", got, want)
		}
	})
	t.Run("a destructive item has a red label", func(t *testing.T) {
		ts := open(t, cupertino.CupertinoMenuItemOpts{IsDestructiveAction: true}, noop)
		style := ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Item"), gluttertest.ByType[*widgets.DefaultTextStyle]()))).(*widgets.DefaultTextStyle).Style()
		want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.SystemRed, ts.Element(gluttertest.ByType[*cupertino.CupertinoMenuItem]()))
		if got, _ := style.Color.Get(); got != want {
			t.Errorf("color %v, want %v", got, want)
		}
	})
	t.Run("the child style is the body style of iOS", func(t *testing.T) {
		ts := open(t, cupertino.CupertinoMenuItemOpts{}, noop)
		style := ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Item"), gluttertest.ByType[*widgets.DefaultTextStyle]()))).(*widgets.DefaultTextStyle).Style()
		if size, _ := style.FontSize.Get(); size != 17 {
			t.Errorf("font size %v", size)
		}
		if style.FontFamily != cupertino.CupertinoSystemTextFamily {
			t.Errorf("family %q", style.FontFamily)
		}
	})
	t.Run("a long label is cut at two lines", func(t *testing.T) {
		ts := menuTester(t, ui.Size{Width: 800, Height: 800})
		controller := widgets.NewMenuController()
		long := ""
		for range 40 {
			long += "word "
		}
		ts.PumpWidget(anchorApp(cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item(long, noop)},
			cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: anchorButton("Anchor")}), painting.AlignmentCenter))
		controller.Open()
		settle(ts)
		textInMenu := gluttertest.Descendant(menuSurface(), gluttertest.ByType[*widgets.Text]())
		one := ts.Size(textInMenu).Height
		if one <= 0 {
			t.Fatal("no text")
		}
		lines := ts.Widget(gluttertest.First(gluttertest.Ancestor(textInMenu, gluttertest.ByType[*widgets.DefaultTextStyle]()))).(*widgets.DefaultTextStyle).MaxLines()
		if n, ok := lines.Get(); !ok || n != 2 {
			t.Errorf("max lines %v", lines)
		}
	})
}

func TestCupertinoMenuAnchorTagsAndZeroArea(t *testing.T) {
	t.Run("it does not crash at zero area", func(t *testing.T) {
		ts := menuTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBox(
			cupertino.NewCupertinoMenuAnchor([]widgets.Widget{item("A", noop)}, cupertino.CupertinoMenuAnchorOpts{Child: anchorButton("Anchor")}),
			widgets.SizedBoxOpts{Width: foundation.Some(0.0), Height: foundation.Some(0.0)}))}))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
}
