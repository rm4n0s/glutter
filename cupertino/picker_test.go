package cupertino_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoPicker, ported from cupertino/picker_test.dart (step 34.7). The cases that read the
// canvas with a `paints` matcher are the goldens (picker_golden_test.go) and checks of the widgets that
// the picker builds; the platform channel mocks are the hooks of the fake haptic feedback and system sound.

func pickerChildren(n int, width float64) []widgets.Widget {
	var children []widgets.Widget
	for i := range n {
		children = append(children, widgets.NewSizedBox(widgets.NewText(fmt.Sprint(i)), widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(50.0)}))
	}
	return children
}

func wheelChildren(n int) []widgets.Widget {
	var children []widgets.Widget
	for i := range n {
		children = append(children, widgets.NewCenter(widgets.NewSizedBox(widgets.NewText(fmt.Sprint(i)),
			widgets.SizedBoxOpts{Width: foundation.Some(400.0), Height: foundation.Some(100.0)})))
	}
	return children
}

func squarePicker(dimension float64, picker widgets.Widget) widgets.Widget {
	return widgets.NewAlign(widgets.NewSizedBox(picker, widgets.SizedBoxOpts{Width: foundation.Some(dimension), Height: foundation.Some(dimension)}),
		widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
}

func pickerNoop(int) {}

func sizedBoxOf(text string) gluttertest.Finder {
	return gluttertest.First(gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*widgets.SizedBox]()))
}

func paragraphStyle(t *testing.T, ts *gluttertest.Tester, text string) *painting.TextStyle {
	t.Helper()
	p, ok := ts.RenderObject(gluttertest.ByText(text)).(*rendering.RenderParagraph)
	if !ok {
		t.Fatalf("%q is not drawn by a RenderParagraph", text)
	}
	span, ok := p.Text().(*painting.TextSpan)
	if !ok {
		t.Fatalf("the text of %q is %T", text, p.Text())
	}
	return span.Style()
}

func TestCupertinoPickerThemeStyling(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
	ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
		Home: squarePicker(300, cupertino.NewCupertinoPicker(50, pickerNoop, pickerChildren(3, 300)))}))
	style := paragraphStyle(t, ts, "1")
	if got := style.Color; !got.IsSome() || got.Value() != ui.NewColor(0xFF000000) {
		t.Errorf("color = %v, want black", got)
	}
	if style.FontFamily != cupertino.CupertinoSystemDisplayFamily {
		t.Errorf("fontFamily = %q, want %q", style.FontFamily, cupertino.CupertinoSystemDisplayFamily)
	}
	if style.FontSize.Value() != 21 || style.FontWeight != ui.FontWeightW400 || style.LetterSpacing.Value() != -0.6 {
		t.Errorf("style = %v, want 21 w400 with letter spacing -0.6", style)
	}
	if style.Inherit {
		t.Error("the picker's text style inherits")
	}
}

// semanticsValues finds the node of the picker, the one that has a value.
func pickerSemanticsNode(ts *gluttertest.Tester, value string) *semantics.SemanticsNode {
	owner := ts.RenderObject(gluttertest.First(gluttertest.ByType[*cupertino.CupertinoPicker]())).AsRenderObject().Owner().SemanticsOwner()
	var found *semantics.SemanticsNode
	var walk func(n *semantics.SemanticsNode)
	walk = func(n *semantics.SemanticsNode) {
		if n.Value() == value {
			found = n
		}
		n.VisitChildren(func(c *semantics.SemanticsNode) bool { walk(c); return true })
	}
	walk(owner.RootSemanticsNode())
	return found
}

func TestCupertinoPickerSemantics(t *testing.T) {
	t.Run("the node has the value and the neighbours", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.EnsureSemantics()
		controller := widgets.NewFixedExtentScrollController()
		defer controller.Dispose()
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: widgets.NewSizedBox(cupertino.NewCupertinoPicker(50, pickerNoop, pickerChildren(13, 300), cupertino.CupertinoPickerOpts{ScrollController: controller}),
				widgets.SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(300.0)})}))
		ts.Pump(time.Millisecond)
		n := pickerSemanticsNode(ts, "0")
		if n == nil {
			t.Fatal("no node has the value 0")
		}
		d := n.GetSemanticsData()
		if d.IncreasedValue() != "1" || d.DecreasedValue() != "" || !d.HasAction(ui.SemanticsActionIncrease) || d.HasAction(ui.SemanticsActionDecrease) {
			t.Errorf("item 0: increased %q decreased %q actions %v", d.IncreasedValue(), d.DecreasedValue(), d.Actions)
		}
		controller.JumpToItem(11)
		ts.PumpAndSettle()
		n = pickerSemanticsNode(ts, "11")
		if n == nil {
			t.Fatal("no node has the value 11 after the jump")
		}
		d = n.GetSemanticsData()
		if d.IncreasedValue() != "12" || d.DecreasedValue() != "10" || !d.HasAction(ui.SemanticsActionIncrease) || !d.HasAction(ui.SemanticsActionDecrease) {
			t.Errorf("item 11: increased %q decreased %q actions %v", d.IncreasedValue(), d.DecreasedValue(), d.Actions)
		}
	})
	t.Run("an excluded current item sets no value and no action", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.EnsureSemantics()
		controller := widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: 1})
		defer controller.Dispose()
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: widgets.NewSizedBox(cupertino.NewCupertinoPicker(50, pickerNoop, []widgets.Widget{
				widgets.NewText("0"), widgets.NewExcludeSemantics(widgets.NewText("1")), widgets.NewText("2")},
				cupertino.CupertinoPickerOpts{ScrollController: controller}),
				widgets.SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(300.0)})}))
		ts.Pump(time.Millisecond)
		if pickerSemanticsNode(ts, "1") != nil {
			t.Error("the excluded item gave the picker a value")
		}
		controller.JumpToItem(0)
		ts.PumpAndSettle()
		n := pickerSemanticsNode(ts, "0")
		if n == nil {
			t.Fatal("no node has the value 0")
		}
		if n.GetSemanticsData().HasAction(ui.SemanticsActionIncrease) {
			t.Error("item 0 can increase to an item with no label")
		}
		controller.JumpToItem(2)
		ts.PumpAndSettle()
		n = pickerSemanticsNode(ts, "2")
		if n == nil {
			t.Fatal("no node has the value 2")
		}
		if n.GetSemanticsData().HasAction(ui.SemanticsActionDecrease) {
			t.Error("item 2 can decrease to an item with no label")
		}
	})
}

func TestCupertinoPickerLayout(t *testing.T) {
	t.Run("the builder follows the child count", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		frame := func(count int) widgets.Widget {
			return widgets.NewDirectionality(ui.TextDirectionLTR, cupertino.NewCupertinoPickerBuilder(50, pickerNoop,
				func(bc widgets.BuildContext, i int) widgets.Widget { return widgets.NewText(fmt.Sprint(i)) },
				cupertino.CupertinoPickerOpts{ChildCount: foundation.Some(count)}))
		}
		ts.PumpWidget(frame(1))
		if !ts.RenderObject(gluttertest.ByText("0")).AsRenderObject().Attached() {
			t.Error("0 is not attached")
		}
		ts.PumpWidget(frame(2))
		for _, s := range []string{"0", "1"} {
			if !ts.RenderObject(gluttertest.ByText(s)).AsRenderObject().Attached() {
				t.Errorf("%s is not attached", s)
			}
		}
	})
	t.Run("the selected item is in the middle", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		controller := widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: 1})
		defer controller.Dispose()
		ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, squarePicker(300,
			cupertino.NewCupertinoPicker(50, pickerNoop, pickerChildren(3, 300), cupertino.CupertinoPickerOpts{ScrollController: controller}))))
		if got := ts.TopLeft(sizedBoxOf("1")); got != (ui.Offset{Dx: 0, Dy: 125}) {
			t.Errorf("item 1 at %v, want (0, 125)", got)
		}
		controller.JumpToItem(0)
		ts.Pump()
		if got := ts.TopLeft(sizedBoxOf("1")); math.Abs(got.Dx) > 0.5 || math.Abs(got.Dy-170) > 0.5 {
			t.Errorf("item 1 at %v, want (0, 170)", got)
		}
		if got := ts.TopLeft(sizedBoxOf("0")); got != (ui.Offset{Dx: 0, Dy: 125}) {
			t.Errorf("item 0 at %v, want (0, 125)", got)
		}
	})
}

func TestCupertinoPickerOverlayAndBackground(t *testing.T) {
	dyn := cupertino.NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFF123456), ui.NewColor(0xFF654321))
	for _, tt := range []struct {
		name       string
		brightness ui.Brightness
		want       ui.Color
	}{{"light", ui.BrightnessLight, ui.NewColor(0xFF123456)}, {"dark", ui.BrightnessDark, ui.NewColor(0xFF654321)}} {
		t.Run("the background resolves in "+tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
				Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
				Home: squarePicker(300, cupertino.NewCupertinoPicker(15, pickerNoop, []widgets.Widget{widgets.NewText("1"), widgets.NewText("1")},
					cupertino.CupertinoPickerOpts{BackgroundColor: dyn}))}))
			var found bool
			for _, e := range ts.Elements(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoPicker](), gluttertest.ByType[*widgets.DecoratedBox]())) {
				if d, ok := e.Widget().(*widgets.DecoratedBox).Decoration().(*painting.BoxDecoration); ok && d.Color == foundation.Some(tt.want) {
					found = true
				}
			}
			if !found {
				t.Errorf("no DecoratedBox of %v below the picker", tt.want)
			}
		})
	}
	t.Run("the default overlay is built, a nil one is not", func(t *testing.T) {
		for _, noOverlay := range []bool{false, true} {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
				Home: squarePicker(300, cupertino.NewCupertinoPicker(15, pickerNoop, []widgets.Widget{widgets.NewText("1"), widgets.NewText("1")},
					cupertino.CupertinoPickerOpts{NoSelectionOverlay: noOverlay}))}))
			n := len(ts.Elements(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay]()))
			if (n == 1) == noOverlay {
				t.Errorf("noOverlay %v: %d overlays", noOverlay, n)
			}
		}
	})
	t.Run("a given overlay is used with its background", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		bg := ui.NewColor(0x12345678)
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: squarePicker(300, cupertino.NewCupertinoPicker(15, pickerNoop, []widgets.Widget{widgets.NewText("1"), widgets.NewText("1")},
				cupertino.CupertinoPickerOpts{SelectionOverlay: cupertino.NewCupertinoPickerDefaultSelectionOverlay(cupertino.CupertinoPickerDefaultSelectionOverlayOpts{Background: bg})}))}))
		o, ok := ts.Widget(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay]()).(*cupertino.CupertinoPickerDefaultSelectionOverlay)
		if !ok || o.Background() != ui.ColorValue(bg) {
			t.Errorf("overlay = %v", o)
		}
	})
	t.Run("the default selection overlay decorates directionally", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: cupertino.NewCupertinoPicker(15, pickerNoop, []widgets.Widget{widgets.NewText("1"), widgets.NewText("1")},
				cupertino.CupertinoPickerOpts{SelectionOverlay: cupertino.NewCupertinoPickerDefaultSelectionOverlay(
					cupertino.CupertinoPickerDefaultSelectionOverlayOpts{Background: ui.NewColor(0x12345678)})})}))
		c := ts.Widget(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay](), gluttertest.ByType[*widgets.Container]()))).(*widgets.Container)
		if _, ok := c.Margin().(painting.EdgeInsetsDirectional); !ok {
			t.Errorf("margin is %T, want EdgeInsetsDirectional", c.Margin())
		}
	})
}

func iosPlatform(t *testing.T) {
	t.Helper()
	skipUnlessDebug(t)
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformIOS))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func scrollingPicker(t *testing.T, items *[]int, opts ...cupertino.CupertinoPickerOpts) (*gluttertest.Tester, *[]string) {
	t.Helper()
	// The view of flutter_test has a device pixel ratio of 3, which sets the tolerance of the spring that settles the wheel.
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 3})
	var calls []string
	ts.Binding.HapticFeedback().OnPerform = func(k ui.HapticFeedbackKind) {
		if k != ui.HapticFeedbackSelectionClick {
			t.Errorf("haptic feedback %v, want the selection click", k)
		}
		calls = append(calls, "haptic selectionClick")
	}
	ts.Binding.SystemSound().OnPlay = func(k services.SystemSoundType) {
		if k != services.SystemSoundTypeTick {
			t.Errorf("system sound %v, want the tick", k)
		}
		calls = append(calls, "sound tick")
	}
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR,
		cupertino.NewCupertinoPicker(100, func(i int) { *items = append(*items, i) }, wheelChildren(100), opts...)))
	return ts, &calls
}

func TestCupertinoPickerScroll(t *testing.T) {
	t.Run("scrolling reports the item and gives feedback when it passes the middle (iOS)", func(t *testing.T) {
		iosPlatform(t)
		var items []int
		ts, calls := scrollingPicker(t, &items)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: -90})
		if fmt.Sprint(items) != "[1]" || len(*calls) != 0 {
			t.Fatalf("items %v calls %v", items, *calls)
		}
		ts.PumpAndSettle()
		if len(*calls) != 2 || (*calls)[0] != "haptic selectionClick" || (*calls)[1] != "sound tick" {
			t.Fatalf("calls = %v", *calls)
		}
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 110})
		if fmt.Sprint(items) != "[1 0]" || len(*calls) != 4 {
			t.Errorf("items %v calls %v", items, *calls)
		}
	})
	t.Run("onScrollEnd reports only when the scroll ends", func(t *testing.T) {
		var items []int
		ts, _ := scrollingPicker(t, &items, cupertino.CupertinoPickerOpts{ChangeReportingBehavior: widgets.ChangeReportingBehaviorOnScrollEnd})
		start := ts.TopLeft(gluttertest.ByText("0"))
		g := ts.StartGesture(start)
		g.MoveBy(ui.Offset{Dy: -49})
		g.MoveBy(ui.Offset{Dy: -1})
		g.MoveBy(ui.Offset{Dy: -99})
		if len(items) != 0 {
			t.Fatalf("items = %v before the end", items)
		}
		g.MoveBy(ui.Offset{Dy: -1})
		g.Up()
		ts.PumpAndSettle()
		if fmt.Sprint(items) != "[2]" {
			t.Fatalf("items = %v, want [2]", items)
		}
		g.Down(start)
		g.MoveBy(ui.Offset{Dy: 100})
		if fmt.Sprint(items) != "[2]" {
			t.Fatalf("items = %v while dragging", items)
		}
		g.Up()
		if fmt.Sprint(items) != "[2 1]" {
			t.Errorf("items = %v, want [2 1]", items)
		}
	})
	t.Run("a tap on an item gives no feedback (iOS)", func(t *testing.T) {
		iosPlatform(t)
		var items []int
		ts, calls := scrollingPicker(t, &items)
		ts.Tap(gluttertest.ByText("2"))
		ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
		if fmt.Sprint(items) != "[1 2]" || len(*calls) != 0 {
			t.Fatalf("items %v calls %v", items, *calls)
		}
		ts.Drag(gluttertest.ByText("2"), ui.Offset{Dy: -30})
		ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
		if fmt.Sprint(items) != "[1 2]" || len(*calls) != 0 {
			t.Errorf("items %v calls %v", items, *calls)
		}
	})
	t.Run("no feedback outside iOS", func(t *testing.T) {
		for _, p := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformLinux,
			foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows} {
			t.Run(p.String(), func(t *testing.T) {
				skipUnlessDebug(t)
				foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
				t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
				var items []int
				ts, calls := scrollingPicker(t, &items)
				ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: -100})
				ts.PumpAndSettle()
				if fmt.Sprint(items) != "[1]" || len(*calls) != 0 {
					t.Errorf("items %v calls %v", items, *calls)
				}
			})
		}
	})
	for _, p := range []foundation.TargetPlatform{foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS} {
		t.Run("a drag in between items settles back "+p.String(), func(t *testing.T) {
			skipUnlessDebug(t)
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
			t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			controller := widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: 10})
			defer controller.Dispose()
			var items []int
			ts, _ := scrollingPicker(t, &items, cupertino.CupertinoPickerOpts{ScrollController: controller})
			zeroSlop := gluttertest.GestureOpts{Pointer: 1, TouchSlopY: foundation.Some(0.0)}
			ts.Drag(sizedBoxOf("10"), ui.Offset{Dy: 30}, zeroSlop)
			if got := ts.TopLeft(sizedBoxOf("10")); got != (ui.Offset{Dx: 200, Dy: 250}) {
				t.Fatalf("item 10 at %v, want (200, 250)", got)
			}
			ts.PumpAndSettle()
			if got := ts.TopLeft(sizedBoxOf("10")).Dy; math.Abs(got-250) > 0.5 || len(items) != 0 {
				t.Fatalf("item 10 at %v items %v after settling", got, items)
			}
			ts.Drag(sizedBoxOf("10"), ui.Offset{Dy: 70}, zeroSlop)
			ts.PumpAndSettle()
			if got := ts.TopLeft(sizedBoxOf("10")).Dy; math.Abs(got-340) > 0.5 || fmt.Sprint(items) != "[9]" {
				t.Errorf("item 10 at %v items %v, want 340 and [9]", got, items)
			}
		})
		t.Run("a big fling that overscrolls springs back "+p.String(), func(t *testing.T) {
			skipUnlessDebug(t)
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
			t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			controller := widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: 10})
			defer controller.Dispose()
			var items []int
			ts, _ := scrollingPicker(t, &items, cupertino.CupertinoPickerOpts{ScrollController: controller})
			ts.Fling(sizedBoxOf("10"), ui.Offset{Dy: 10000}, 1000)
			if p == foundation.TargetPlatformIOS && len(ts.Elements(sizedBoxOf("0"))) != 0 {
				t.Error("the first item is still built after the fling")
			}
			if fmt.Sprint(items) != "[8 6 4 2 0]" {
				t.Fatalf("items = %v, want [8 6 4 2 0]", items)
			}
			ts.PumpAndSettle()
			if got := ts.TopLeft(sizedBoxOf("0")).Dy; math.Abs(got-250) > 0.001 || fmt.Sprint(items) != "[8 6 4 2 0]" {
				t.Errorf("item 0 at %v items %v after the spring back", got, items)
			}
		})
	}
}

func TestCupertinoPickerDarkMode(t *testing.T) {
	for _, tt := range []struct {
		name       string
		brightness ui.Brightness
	}{{"light", ui.BrightnessLight}, {"dark", ui.BrightnessDark}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
				Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
				Home:  squarePicker(300, cupertino.NewCupertinoPicker(50, pickerNoop, pickerChildren(3, 300)))}))
			style := paragraphStyle(t, ts, "1")
			want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.Label, ts.Element(gluttertest.ByType[*cupertino.CupertinoPicker]()))
			if !style.Color.IsSome() || style.Color.Value() != want {
				t.Errorf("color = %v, want %v", style.Color, want)
			}
		})
	}
}

func TestCupertinoPickerLifecycle(t *testing.T) {
	t.Run("the scroll controller is detached when the picker is disposed", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		controller := widgets.NewFixedExtentScrollController()
		defer controller.Dispose()
		if controller.HasListeners() || len(controller.Positions()) != 0 {
			t.Fatal("a new controller has listeners or positions")
		}
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: widgets.NewAlign(widgets.NewCenter(cupertino.NewCupertinoPicker(50, pickerNoop, pickerChildren(3, 300),
				cupertino.CupertinoPickerOpts{ScrollController: controller})), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})}))
		if !controller.HasListeners() || len(controller.Positions()) != 1 {
			t.Fatalf("listeners %v positions %d while built", controller.HasListeners(), len(controller.Positions()))
		}
		ts.PumpWidget(widgets.NewSizedBoxExpand(nil))
		if controller.HasListeners() || len(controller.Positions()) != 0 {
			t.Errorf("listeners %v positions %d after dispose", controller.HasListeners(), len(controller.Positions()))
		}
	})
	t.Run("a picker removed while a tap scrolls reports no error", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: squarePicker(300, cupertino.NewCupertinoPicker(50, pickerNoop, pickerChildren(5, 300)))}))
		ts.Tap(gluttertest.ByText("2"))
		ts.Pump(50 * time.Millisecond)
		ts.PumpWidget(widgets.NewSizedBoxExpand(nil))
		ts.Pump(time.Second)
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	t.Run("a controller can be swapped", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		a, b := widgets.NewFixedExtentScrollController(), widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: 2})
		defer a.Dispose()
		defer b.Dispose()
		build := func(c *widgets.FixedExtentScrollController) widgets.Widget {
			return cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
				Home: squarePicker(300, cupertino.NewCupertinoPicker(50, pickerNoop, pickerChildren(3, 300), cupertino.CupertinoPickerOpts{ScrollController: c}))})
		}
		ts.PumpWidget(build(a))
		ts.PumpWidget(build(b))
		if a.HasListeners() || !b.HasListeners() {
			t.Errorf("a listens %v, b listens %v", a.HasListeners(), b.HasListeners())
		}
		ts.PumpWidget(build(nil))
		if b.HasListeners() {
			t.Error("b still listens after the picker made its own controller")
		}
	})
}

func TestCupertinoPickerTaps(t *testing.T) {
	t.Run("the children register taps with a small diameter ratio", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		tapped := map[int]bool{}
		var children []widgets.Widget
		for i := range 100 {
			children = append(children, widgets.NewGestureDetector(widgets.NewSizedBox(widgets.NewText(fmt.Sprint(i)), widgets.SizedBoxOpts{Width: foundation.Some(55.0), Height: foundation.Some(55.0)}),
				widgets.GestureDetectorOpts{Key: foundation.NewValueKey(i), OnTap: func() { tapped[i] = true }}))
		}
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: widgets.NewAlign(widgets.NewCenter(widgets.NewSizedBox(cupertino.NewCupertinoPicker(55, pickerNoop, children, cupertino.CupertinoPickerOpts{
				DiameterRatio: foundation.Some(0.9)}), widgets.SizedBoxOpts{Height: foundation.Some(120.0)})), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})}))
		ts.Tap(gluttertest.ByKey(foundation.NewValueKey(0)))
		ts.Tap(gluttertest.ByKey(foundation.NewValueKey(1)))
		if !tapped[0] || !tapped[1] || len(tapped) != 2 {
			t.Fatalf("tapped %v, want 0 and 1", tapped)
		}
		ts.TapAt(ts.Center(gluttertest.ByKey(foundation.NewValueKey(2))))
		if tapped[2] {
			t.Error("the third child is not painted but was hit")
		}
	})
	t.Run("a tap on a child selects it", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		selected := 0
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: cupertino.NewCupertinoPicker(10, func(i int) { selected = i }, []widgets.Widget{
				widgets.NewText("0"), widgets.NewText("1"), widgets.NewText("2"), widgets.NewText("3")})}))
		for _, i := range []int{1, 3, 0, 2} {
			ts.Tap(gluttertest.ByText(fmt.Sprint(i)))
			ts.Pump()
			ts.Pump(300*time.Millisecond + time.Microsecond)
			if selected != i {
				t.Errorf("after the tap on %d the item is %d", i, selected)
			}
		}
	})
}

func TestCupertinoPickerZeroArea(t *testing.T) {
	t.Run("the overlay", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.SetViewSize(ui.Size{})
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: widgets.NewCenter(cupertino.NewCupertinoPickerDefaultSelectionOverlay())}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay]()); got != (ui.Size{}) {
			t.Errorf("size = %v", got)
		}
	})
	t.Run("the picker", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
			Home: widgets.NewCenter(widgets.NewSizedBox(cupertino.NewCupertinoPicker(2, pickerNoop, []widgets.Widget{widgets.NewText("X"), widgets.NewText("Y")}),
				widgets.SizedBoxOpts{Width: foundation.Some(0.0), Height: foundation.Some(0.0)}))}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoPicker]()); got != (ui.Size{}) {
			t.Errorf("size = %v", got)
		}
	})
}

func TestCupertinoPickerArguments(t *testing.T) {
	skipUnlessDebug(t)
	kids := []widgets.Widget{widgets.NewText("a")}
	for _, tt := range []struct {
		name string
		make func()
		tag  string
	}{
		{"diameter ratio", func() {
			cupertino.NewCupertinoPicker(10, pickerNoop, kids, cupertino.CupertinoPickerOpts{DiameterRatio: foundation.Some(0.0)})
		}, "CupertinoPickerDiameterRatioNotPositive"},
		{"magnification", func() {
			cupertino.NewCupertinoPicker(10, pickerNoop, kids, cupertino.CupertinoPickerOpts{Magnification: foundation.Some(0.0)})
		}, "CupertinoPickerMagnificationNotPositive"},
		{"item extent", func() { cupertino.NewCupertinoPicker(0, pickerNoop, kids) }, "CupertinoPickerItemExtentNotPositive"},
		{"squeeze", func() {
			cupertino.NewCupertinoPicker(10, pickerNoop, kids, cupertino.CupertinoPickerOpts{Squeeze: foundation.Some(-1.0)})
		}, "CupertinoPickerSqueezeNotPositive"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requireErrorTag(t, recoverError(tt.make), tt.tag)
		})
	}
}
