package cupertino_test

import (
	"math"
	"testing"

	"golang.org/x/image/font/gofont/gomono"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/dialog_test.dart: the taps and the slides, the styles
// of the actions, the sizes of the sections, the scrolling, the insets, the surface and the zero area. The cases that
// read the canvas with a `paints` matcher are the goldens (dialog_golden_test.go), the semantics cases are not
// ported, and the cases about the restoration and the dialog route are in route_popups_test.go (SPEC §12.10,
// step 34.6).

// launchApp is a Cupertino app with a button, "Go", that shows the dialog that the builder makes.
func launchApp(builder widgets.WidgetBuilder, opts ...cupertino.CupertinoAppOpts) widgets.Widget {
	o := cupertino.CupertinoAppOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Home = widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoButton(func() { cupertino.ShowCupertinoDialog(bc, builder) }, widgets.NewText("Go"))
	}))
	return testApp(o)
}

// dialogTester is a tester with a font for the system families of the text of the dialogs, which have no glyph in
// the test engine (the Material tests and the toolbars do the same with sans-serif).
func dialogTester(t *testing.T, size ui.Size) *gluttertest.Tester {
	t.Helper()
	return gluttertest.New(t, gluttertest.Opts{Size: size, Fonts: []gluttertest.Font{
		{Family: cupertino.CupertinoSystemTextFamily, Data: gomono.TTF}, {Family: cupertino.CupertinoSystemDisplayFamily, Data: gomono.TTF}}})
}

func goButton() gluttertest.Finder { return gluttertest.ByText("Go") }

func pop(bc widgets.BuildContext) { widgets.NavigatorOf(bc).Pop() }

func TestCupertinoAlertDialogTaps(t *testing.T) {
	t.Run("a tap on a button calls onPressed", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		deleted := false
		ts.PumpWidget(launchApp(func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{
				Title: widgets.NewText("The title"), Content: widgets.NewText("The content"),
				Actions: []widgets.Widget{
					cupertino.NewCupertinoDialogAction(widgets.NewText("Cancel")),
					cupertino.NewCupertinoDialogAction(widgets.NewText("Delete"), cupertino.CupertinoDialogActionOpts{IsDestructiveAction: true,
						OnPressed: func() { deleted = true; pop(bc) }}),
				}})
		}))
		ts.Tap(goButton())
		ts.Pump()
		if deleted {
			t.Fatal("deleted before the tap")
		}
		ts.Tap(gluttertest.ByText("Delete"))
		ts.Pump()
		if !deleted || ts.Exists(gluttertest.ByText("Delete")) {
			t.Errorf("deleted %v, the dialog is still there %v", deleted, ts.Exists(gluttertest.ByText("Delete")))
		}
	})
	t.Run("a tap on the padding of a button calls onPressed", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pressed := false
		ts.PumpWidget(launchApp(func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("Title"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"), cupertino.CupertinoDialogActionOpts{OnPressed: func() { pressed = true }})}})
		}))
		ts.Tap(goButton())
		ts.PumpAndSettle()
		at := ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoDialogAction]()).Add(ui.Offset{Dx: 3, Dy: 3})
		ts.TapAt(at)
		if !pressed {
			t.Error("the tap at the padding did not call onPressed")
		}
	})
	t.Run("a tap on a disabled button does nothing", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(launchApp(func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("Title"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"))}})
		}))
		ts.Tap(goButton())
		ts.PumpAndSettle()
		ts.Tap(gluttertest.ByText("OK"))
		ts.Pump()
		if !ts.Exists(gluttertest.ByText("OK")) {
			t.Error("a disabled action closed the dialog")
		}
	})
	t.Run("a tap on a button can be slid to other buttons", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pressed := 0
		ts.PumpWidget(launchApp(func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Actions: []widgets.Widget{
				cupertino.NewCupertinoDialogAction(widgets.NewText("One"), cupertino.CupertinoDialogActionOpts{OnPressed: func() { pressed = 1; pop(bc) }}),
				cupertino.NewCupertinoDialogAction(widgets.NewText("Two"), cupertino.CupertinoDialogActionOpts{OnPressed: func() { pressed = 2; pop(bc) }}),
			}})
		}))
		ts.Tap(goButton())
		ts.PumpAndSettle()
		g := ts.StartGesture(ts.Center(gluttertest.ByText("Two")))
		ts.PumpAndSettle()
		g.MoveTo(ts.Center(gluttertest.ByText("One")))
		ts.PumpAndSettle()
		g.Up()
		if pressed != 1 {
			t.Errorf("pressed %d, want 1", pressed)
		}
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByText("One")) {
			t.Error("the dialog is still there")
		}
	})
	t.Run("a tap on the content can be slid to the buttons", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pressed := 0
		ts.PumpWidget(launchApp(func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Content: widgets.NewText("The content"), Actions: []widgets.Widget{
				cupertino.NewCupertinoDialogAction(widgets.NewText("One"), cupertino.CupertinoDialogActionOpts{OnPressed: func() { pressed = 1; pop(bc) }}),
			}})
		}))
		ts.Tap(goButton())
		ts.PumpAndSettle()
		g := ts.StartGesture(ts.Center(gluttertest.ByText("The content")))
		ts.PumpAndSettle()
		g.MoveTo(ts.Center(gluttertest.ByText("One")))
		ts.PumpAndSettle()
		g.Up()
		if pressed != 1 {
			t.Errorf("pressed %d, want 1", pressed)
		}
	})
	t.Run("a tap on the barrier cannot be slid to the buttons", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pressed := 0
		ts.PumpWidget(launchApp(func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Actions: []widgets.Widget{
				cupertino.NewCupertinoDialogAction(widgets.NewText("One"), cupertino.CupertinoDialogActionOpts{OnPressed: func() { pressed = 1; pop(bc) }}),
			}})
		}))
		ts.Tap(goButton())
		ts.PumpAndSettle()
		g := ts.StartGesture(ui.Offset{Dx: 10, Dy: 10})
		ts.PumpAndSettle()
		g.MoveTo(ts.Center(gluttertest.ByText("One")))
		ts.PumpAndSettle()
		g.Up()
		if pressed != 0 {
			t.Errorf("pressed %d, want none", pressed)
		}
	})
	t.Run("a pressed action shows its highlight as soon as the pointer is down", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(launchApp(func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Actions: []widgets.Widget{
				cupertino.NewCupertinoDialogAction(widgets.NewText("One"), cupertino.CupertinoDialogActionOpts{OnPressed: func() {}}),
			}})
		}))
		ts.Tap(goButton())
		ts.PumpAndSettle()
		colorOf := func() ui.Color {
			c := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoAlertDialog](), gluttertest.ByType[*widgets.Container]())).(*widgets.Container)
			return c.Decoration().(*painting.BoxDecoration).Color.OrElse(ui.Color{})
		}
		idle := colorOf()
		g := ts.StartGesture(ts.Center(gluttertest.ByText("One")))
		ts.Pump()
		if pressed := colorOf(); pressed == idle {
			t.Error("the action did not change its color at the pointer down")
		}
		g.Up()
	})
	t.Run("a pressed disabled action does not highlight", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(launchApp(func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Actions: []widgets.Widget{
				cupertino.NewCupertinoDialogAction(widgets.NewText("Off")),
			}})
		}))
		ts.Tap(goButton())
		ts.PumpAndSettle()
		colorOf := func() ui.Color {
			c := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoAlertDialog](), gluttertest.ByType[*widgets.Container]())).(*widgets.Container)
			return c.Decoration().(*painting.BoxDecoration).Color.OrElse(ui.Color{})
		}
		idle := colorOf()
		g := ts.StartGesture(ts.Center(gluttertest.ByText("Off")))
		ts.Pump()
		if colorOf() != idle {
			t.Error("a disabled action highlights")
		}
		g.Up()
	})
}

func TestCupertinoDialogActionStyles(t *testing.T) {
	styleOf := func(t *testing.T, opts cupertino.CupertinoDialogActionOpts) *painting.TextStyle {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoDialogAction(widgets.NewText("Action"), opts))}))
		return ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Action"), gluttertest.ByType[*widgets.DefaultTextStyle](), gluttertest.DescendantOpts{}))).(*widgets.DefaultTextStyle).Style()
	}
	noop := func() {}
	t.Run("an enabled action has the primary color", func(t *testing.T) {
		s := styleOf(t, cupertino.CupertinoDialogActionOpts{OnPressed: noop})
		if s.Color.Value().Alpha() != 0xFF || s.FontWeight != ui.FontWeightW400 {
			t.Errorf("style %v", s)
		}
	})
	t.Run("a disabled action is half transparent", func(t *testing.T) {
		if s := styleOf(t, cupertino.CupertinoDialogActionOpts{}); s.Color.Value().Opacity() > 0.51 {
			t.Errorf("disabled color %v", s.Color.Value())
		}
	})
	t.Run("the default action is bold", func(t *testing.T) {
		if s := styleOf(t, cupertino.CupertinoDialogActionOpts{OnPressed: noop, IsDefaultAction: true}); s.FontWeight != ui.FontWeightW600 {
			t.Errorf("weight %v", s.FontWeight)
		}
	})
	t.Run("the destructive action is red", func(t *testing.T) {
		s := styleOf(t, cupertino.CupertinoDialogActionOpts{OnPressed: noop, IsDestructiveAction: true})
		if got := s.Color.Value(); got.Red() < 200 || got.Green() > 100 {
			t.Errorf("destructive color %v", got)
		}
	})
	t.Run("the default and destructive action is bold and red", func(t *testing.T) {
		s := styleOf(t, cupertino.CupertinoDialogActionOpts{OnPressed: noop, IsDefaultAction: true, IsDestructiveAction: true})
		if s.FontWeight != ui.FontWeightW600 || s.Color.Value().Red() < 200 {
			t.Errorf("style %v", s)
		}
	})
	t.Run("a text style is merged", func(t *testing.T) {
		s := styleOf(t, cupertino.CupertinoDialogActionOpts{OnPressed: noop, TextStyle: &painting.TextStyle{FontSize: foundation.Some(30.0)}})
		if s.FontSize.Value() != 30 {
			t.Errorf("size %v", s.FontSize)
		}
	})
}

func TestCupertinoAlertDialogSections(t *testing.T) {
	buttonOf := func(ts *gluttertest.Tester, label string) ui.Size {
		return ts.Size(gluttertest.Ancestor(gluttertest.ByText(label), gluttertest.ByType[*cupertino.CupertinoDialogAction](), gluttertest.DescendantOpts{}))
	}
	actionsSection := func(ts *gluttertest.Tester, label string) ui.Size {
		return ts.Size(gluttertest.Ancestor(gluttertest.ByText(label), gluttertest.ByType[*widgets.SingleChildScrollView](), gluttertest.DescendantOpts{}))
	}
	show := func(t *testing.T, dialog func() widgets.Widget) *gluttertest.Tester {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(launchApp(func(widgets.BuildContext) widgets.Widget { return dialog() }))
		ts.Tap(goButton())
		ts.Pump()
		return ts
	}
	t.Run("the actions section of one button is the height of the button", func(t *testing.T) {
		ts := show(t, func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("The Title"), Content: widgets.NewText("The message"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"))}})
		})
		if b, a := buttonOf(ts, "OK"), actionsSection(ts, "OK"); b != a {
			t.Errorf("button %v, section %v", b, a)
		}
	})
	t.Run("the actions section of two side by side buttons is the height of the tallest", func(t *testing.T) {
		ts := show(t, func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("The Title"), Content: widgets.NewText("The message"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK")),
					cupertino.NewCupertinoDialogAction(widgets.NewText("Cancel"), cupertino.CupertinoDialogActionOpts{IsDestructiveAction: true})}})
		})
		ok, cancel, section := buttonOf(ts, "OK"), buttonOf(ts, "Cancel"), actionsSection(ts, "OK")
		if ok.Width != cancel.Width || math.Abs(section.Width-(ok.Width+cancel.Width+0.3)) > 1e-9 || section.Height != math.Max(ok.Height, cancel.Height) {
			t.Errorf("ok %v, cancel %v, section %v", ok, cancel, section)
		}
	})
	t.Run("two buttons that do not fit side by side are in a column", func(t *testing.T) {
		long := "A very long text that does not fit in half of the dialog"
		ts := show(t, func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("Title"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK")), cupertino.NewCupertinoDialogAction(widgets.NewText(long))}})
		})
		ok, section := buttonOf(ts, "OK"), actionsSection(ts, "OK")
		if ok.Width != section.Width || section.Height < 2*ok.Height {
			t.Errorf("ok %v, section %v", ok, section)
		}
	})
	t.Run("three buttons are in a column", func(t *testing.T) {
		ts := show(t, func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("The Title"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("One")), cupertino.NewCupertinoDialogAction(widgets.NewText("Two")),
					cupertino.NewCupertinoDialogAction(widgets.NewText("Three"))}})
		})
		one, section := buttonOf(ts, "One"), actionsSection(ts, "One")
		if math.Abs(section.Height-(3*one.Height+2*0.3)) > 1e-6 || one.Width != section.Width {
			t.Errorf("one %v, section %v", one, section)
		}
	})
	t.Run("the title section is empty and the button section is not", func(t *testing.T) {
		ts := show(t, func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"))}})
		})
		if got := buttonOf(ts, "OK"); got.Width != 270 || got.Height < 45 {
			t.Errorf("button %v", got)
		}
	})
	t.Run("the button section is empty and the title section is not", func(t *testing.T) {
		ts := show(t, func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("The Title"), Content: widgets.NewText("The message")})
		})
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoAlertDialog]()); got.Width <= 0 {
			t.Errorf("dialog %v", got)
		}
		if !ts.Exists(gluttertest.ByText("The Title")) {
			t.Error("no title")
		}
	})
	t.Run("the dialog is 270 wide, and 310 with a big text scale", func(t *testing.T) {
		ts := show(t, func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("T"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"))}})
		})
		if got := buttonOf(ts, "OK").Width; got != 270 {
			t.Errorf("width %v", got)
		}
		ts2 := dialogTester(t, ui.Size{})
		ts2.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			data := widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{TextScaler: foundation.Some[painting.TextScaler](painting.NewTextScalerLinear(2))})
			return widgets.NewMediaQuery(data, widgets.NewCenter(cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("T"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"))}})))
		})}))
		if got := buttonOf(ts2, "OK").Width; got != 310 {
			t.Errorf("width with a big text scale %v", got)
		}
	})
	t.Run("the dialog respects small constraints", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 400, Height: 200})
		ts.PumpWidget(launchApp(func(widgets.BuildContext) widgets.Widget {
			var lines []widgets.Widget
			for range 40 {
				lines = append(lines, widgets.NewText("line"))
			}
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("The Title"),
				Content: widgets.NewColumn(lines), Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"))}})
		}))
		ts.Tap(goButton())
		ts.Pump()
		if h := ts.Size(gluttertest.ByType[*cupertino.CupertinoAlertDialog]()).Height; h > 200 {
			t.Errorf("the dialog is %v high in a view of 200", h)
		}
		if !ts.Exists(gluttertest.ByText("OK")) {
			t.Error("the actions were pushed away")
		}
	})
	t.Run("the dialog is moved by the view insets", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			data := widgets.MediaQueryOf(bc)
			return widgets.NewMediaQuery(data.CopyWith(widgets.MediaQueryDataCopyOpts{ViewInsets: foundation.Some(painting.EdgeInsets{Bottom: 300})}), cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("T"),
				Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"))}}))
		})}))
		ts.PumpAndSettle()
		if bottom := ts.BottomRight(gluttertest.ByType[*cupertino.CupertinoPopupSurface]()).Dy; bottom > 600-300 {
			t.Errorf("the dialog ends at %v, under the inset keyboard", bottom)
		}
	})
}

func TestCupertinoDialogZeroArea(t *testing.T) {
	for _, tc := range []struct {
		name string
		w    func() widgets.Widget
		f    gluttertest.Finder
	}{
		{"action", func() widgets.Widget { return cupertino.NewCupertinoDialogAction(widgets.NewText("X")) }, gluttertest.ByType[*cupertino.CupertinoDialogAction]()},
		{"action sheet action", func() widgets.Widget {
			return cupertino.NewCupertinoActionSheetAction(func() {}, widgets.NewText("X"))
		}, gluttertest.ByType[*cupertino.CupertinoActionSheetAction]()},
		{"popup surface", func() widgets.Widget { return cupertino.NewCupertinoPopupSurface(widgets.NewText("X")) }, gluttertest.ByType[*cupertino.CupertinoPopupSurface]()},
		{"alert dialog", func() widgets.Widget {
			return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("X")})
		}, gluttertest.ByType[*cupertino.CupertinoAlertDialog]()},
	} {
		t.Run(tc.name+" does not crash at zero area", func(t *testing.T) {
			ts := dialogTester(t, ui.Size{})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(tc.w()))}))
			if got := ts.Size(tc.f); got != (ui.Size{}) && tc.name != "alert dialog" {
				t.Errorf("size %v", got)
			}
		})
	}
}

func TestCupertinoPopupSurface(t *testing.T) {
	t.Run("the blur sigma has a default, and a negative one is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		if got := cupertino.NewCupertinoPopupSurface(nil).BlurSigma(); got != 30 {
			t.Errorf("blur sigma %v", got)
		}
		err := recoverError(func() {
			cupertino.NewCupertinoPopupSurface(nil, cupertino.CupertinoPopupSurfaceOpts{BlurSigma: foundation.Some(-1.0)})
		})
		checkErrorTag(t, err, "CupertinoPopupSurfaceBlurSigmaNegative")
	})
	t.Run("the surface has the backdrop filter and its color", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoPopupSurface(widgets.NewText("X")))}))
		if !ts.Exists(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoPopupSurface](), gluttertest.ByType[*widgets.BackdropFilter]())) {
			t.Error("no backdrop filter")
		}
		if !ts.Exists(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoPopupSurface](), gluttertest.ByType[*widgets.ColoredBox]())) {
			t.Error("no color")
		}
	})
	t.Run("a surface that is not painted has no color box", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoPopupSurface(widgets.NewText("X"),
			cupertino.CupertinoPopupSurfaceOpts{NotSurfacePainted: true}))}))
		if ts.Exists(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoPopupSurface](), gluttertest.ByType[*widgets.ColoredBox]())) {
			t.Error("a color box")
		}
	})
}
