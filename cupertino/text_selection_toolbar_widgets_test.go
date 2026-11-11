package cupertino_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/{text_selection_toolbar_button,
// desktop_text_selection_toolbar,desktop_text_selection_toolbar_button,adaptive_text_selection_toolbar}_test.dart.
// The cases of the toolbar of iOS are in text_selection_toolbar_internal_test.go.

func homeApp(ts *gluttertest.Tester, w widgets.Widget) {
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(w)}))
}

func TestCupertinoTextSelectionToolbarButton(t *testing.T) {
	button := gluttertest.ByType[*cupertino.CupertinoTextSelectionToolbarButton]()
	shapeColorOf := func(ts *gluttertest.Tester) ui.Color {
		box := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoButton](), gluttertest.ByType[*widgets.DecoratedBox]())).(*widgets.DecoratedBox)
		switch d := box.Decoration().(type) {
		case *painting.ShapeDecoration:
			return d.Color.OrElse(ui.Color{})
		case *painting.BoxDecoration:
			return d.Color.OrElse(ui.Color{})
		}
		t.Fatalf("decoration %T", box.Decoration())
		return ui.Color{}
	}
	t.Run("can press", func(t *testing.T) {
		ts := gluttertest.New(t)
		pressed := false
		homeApp(ts, cupertino.NewCupertinoTextSelectionToolbarButton(widgets.NewText("Tap me"), cupertino.CupertinoTextSelectionToolbarButtonOpts{OnPressed: func() { pressed = true }}))
		if pressed {
			t.Fatal("pressed before the tap")
		}
		ts.Tap(button)
		if !pressed {
			t.Error("not pressed")
		}
	})
	t.Run("the background darkens when pressed", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, cupertino.NewCupertinoTextSelectionToolbarButton(widgets.NewText("Tap me"), cupertino.CupertinoTextSelectionToolbarButtonOpts{OnPressed: func() {}}))
		if got := shapeColorOf(ts); got.Alpha() != 0 {
			t.Fatalf("original color %v, want transparent", got)
		}
		g := ts.StartGesture(ts.Center(button))
		ts.PumpAndSettle()
		if got := shapeColorOf(ts); got != ui.NewColor(0x10000000) {
			t.Errorf("pressed color %v, want 0x10000000", got)
		}
		g.Up()
		ts.PumpAndSettle()
		if got := shapeColorOf(ts); got.Alpha() != 0 {
			t.Errorf("released color %v, want transparent", got)
		}
	})
	t.Run("a nil OnPressed disables the button", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, cupertino.NewCupertinoTextSelectionToolbarButton(widgets.NewText("Tap me")))
		if ts.Widget(gluttertest.ByType[*cupertino.CupertinoButton]()).(*cupertino.CupertinoButton).Enabled() {
			t.Error("the button is enabled")
		}
	})
	t.Run("a button item shows its label or the localized one of its type", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, widgets.NewColumn([]widgets.Widget{
			cupertino.NewCupertinoTextSelectionToolbarButtonButtonItem(widgets.ContextMenuButtonItem{Type: widgets.ContextMenuButtonTypeCut, OnPressed: func() {}}),
			cupertino.NewCupertinoTextSelectionToolbarButtonButtonItem(widgets.ContextMenuButtonItem{Type: widgets.ContextMenuButtonTypeSelectAll, OnPressed: func() {}}),
			cupertino.NewCupertinoTextSelectionToolbarButtonButtonItem(widgets.ContextMenuButtonItem{Label: foundation.Some("Mine"), OnPressed: func() {}}),
		}))
		for _, label := range []string{"Cut", "Select All", "Mine"} {
			if !ts.Exists(gluttertest.ByText(label)) {
				t.Errorf("no %q", label)
			}
		}
	})
	t.Run("a disabled button has the inactive gray text", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, cupertino.NewCupertinoTextSelectionToolbarButtonText("Off"))
		got := ts.Widget(gluttertest.ByText("Off")).(*widgets.Text).Style().Color.Value()
		if want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.InactiveGray, ts.Element(gluttertest.ByText("Off"))); got != want {
			t.Errorf("color %v, want %v", got, want)
		}
	})
	t.Run("a button of zero area does not crash", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, widgets.NewSizedBoxShrink(cupertino.NewCupertinoTextSelectionToolbarButton(widgets.NewText("X"))))
		if got := ts.Size(button); got != (ui.Size{}) {
			t.Errorf("size %v", got)
		}
	})
}

func TestCupertinoDesktopTextSelectionToolbarButton(t *testing.T) {
	button := gluttertest.ByType[*cupertino.CupertinoDesktopTextSelectionToolbarButton]()
	t.Run("can press", func(t *testing.T) {
		ts := gluttertest.New(t)
		pressed := false
		homeApp(ts, cupertino.NewCupertinoDesktopTextSelectionToolbarButton(func() { pressed = true }, widgets.NewText("Tap me")))
		ts.Tap(button)
		if !pressed {
			t.Error("not pressed")
		}
	})
	t.Run("the text keeps its contrast with the background on hover", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, cupertino.NewCupertinoDesktopTextSelectionToolbarButtonText(func() {}, "Tap me"))
		colorOf := func() ui.Color { return ts.Widget(gluttertest.ByText("Tap me")).(*widgets.Text).Style().Color.Value() }
		if got := colorOf(); got != cupertino.CupertinoColors.Black {
			t.Fatalf("color %v, want black", got)
		}
		ts.Hover(ts.Center(button))
		ts.PumpAndSettle()
		bc := ts.Element(button)
		want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoThemeOf(bc).PrimaryContrastingColor(), bc)
		if got := colorOf(); got != want {
			t.Errorf("hovered color %v, want %v", got, want)
		}
	})
	t.Run("the pressed opacity is 0.7", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, cupertino.NewCupertinoDesktopTextSelectionToolbarButton(func() {}, widgets.NewText("Tap me")))
		opacity := func() float64 {
			return ts.Widget(gluttertest.Descendant(button, gluttertest.ByType[*widgets.FadeTransition]())).(*widgets.FadeTransition).Opacity().Value()
		}
		if got := opacity(); got != 1.0 {
			t.Fatalf("opacity %v", got)
		}
		g := ts.StartGesture(ts.Center(button))
		ts.PumpAndSettle()
		if got := opacity(); math.Abs(got-0.7) > 1e-9 {
			t.Errorf("pressed opacity %v, want 0.7", got)
		}
		g.Up()
		ts.PumpAndSettle()
		if got := opacity(); got != 1.0 {
			t.Errorf("released opacity %v", got)
		}
	})
	t.Run("a nil OnPressed disables the button", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, cupertino.NewCupertinoDesktopTextSelectionToolbarButton(nil, widgets.NewText("Tap me")))
		if ts.Widget(gluttertest.ByType[*cupertino.CupertinoButton]()).(*cupertino.CupertinoButton).Enabled() {
			t.Error("the button is enabled")
		}
	})
	t.Run("a button of zero area does not crash", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, widgets.NewSizedBoxShrink(cupertino.NewCupertinoDesktopTextSelectionToolbarButton(nil, widgets.NewText("X"))))
		if got := ts.Size(button); got != (ui.Size{}) {
			t.Errorf("size %v", got)
		}
	})
}

func TestCupertinoDesktopTextSelectionToolbar(t *testing.T) {
	toolbar := gluttertest.ByType[*cupertino.CupertinoDesktopTextSelectionToolbar]()
	oneButton := func(anchor ui.Offset) widgets.Widget {
		return cupertino.NewCupertinoDesktopTextSelectionToolbar(anchor, []widgets.Widget{
			cupertino.NewCupertinoDesktopTextSelectionToolbarButton(func() {}, widgets.NewText("Tap me"))})
	}
	boxes := func(ts *gluttertest.Tester) []*widgets.DecoratedBox {
		var out []*widgets.DecoratedBox
		for _, w := range ts.Widgets(gluttertest.Descendant(toolbar, gluttertest.ByType[*widgets.DecoratedBox]())) {
			out = append(out, w.(*widgets.DecoratedBox))
		}
		return out
	}
	t.Run("it has the backdrop filter of the blur and the saturation", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, oneButton(ui.Offset{}))
		f := ts.Widget(gluttertest.Descendant(toolbar, gluttertest.ByType[*widgets.BackdropFilter]())).(*widgets.BackdropFilter).Filter()
		if want := ui.NewImageFilterCompose(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 1}), ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 1})); f == nil || typeName(f) != typeName(want) {
			t.Errorf("filter %T, want %T", f, want)
		}
	})
	t.Run("it has a shadow", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, oneButton(ui.Offset{}))
		d, ok := boxes(ts)[0].Decoration().(*painting.ShapeDecoration)
		if !ok || d.Shadows == nil {
			t.Errorf("decoration %v", boxes(ts)[0].Decoration())
		}
	})
	t.Run("it is translucent", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, oneButton(ui.Offset{}))
		d, ok := boxes(ts)[1].Decoration().(*painting.ShapeDecoration)
		if !ok || d.Color.OrElse(ui.NewColor(0xFFFFFFFF)).Opacity() >= 1 {
			t.Errorf("decoration %v", boxes(ts)[1].Decoration())
		}
	})
	t.Run("it positions itself at the anchor", func(t *testing.T) {
		ts := gluttertest.New(t)
		anchor := ui.Offset{Dx: 30, Dy: 40}
		homeApp(ts, oneButton(anchor))
		got := ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoDesktopTextSelectionToolbarButton]())
		if got.Dx <= anchor.Dx || got.Dy <= anchor.Dy {
			t.Errorf("button at %v, want it past %v (the padding of the toolbar)", got, anchor)
		}
	})
	t.Run("a toolbar of zero area does not crash", func(t *testing.T) {
		ts := gluttertest.New(t)
		homeApp(ts, widgets.NewSizedBoxShrink(cupertino.NewCupertinoDesktopTextSelectionToolbar(ui.Offset{Dx: 10, Dy: 10}, []widgets.Widget{widgets.NewText("X")})))
		if got := ts.Size(toolbar); got != (ui.Size{}) {
			t.Errorf("size %v", got)
		}
	})
	t.Run("no children is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		err := recoverError(func() { cupertino.NewCupertinoDesktopTextSelectionToolbar(ui.Offset{}, nil) })
		checkErrorTag(t, err, "CupertinoDesktopTextSelectionToolbarNoChildren")
	})
}

func typeName(v any) string { return fmt.Sprintf("%T", v) }
