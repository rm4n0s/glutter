package cupertino_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/action_sheet_test.dart: the taps and the slides, the
// cancel button, the sizes of the sections, the sizes of the sheet and the zero area. The sizes that depend on the
// metrics of Flutter's test font (the heights of the texts) are checked against the metrics of the test font here;
// the cases that read the canvas are the goldens (dialog_golden_test.go) and the semantics cases are not ported
// (SPEC §12.10, step 34.6).

// launchSheet is a Cupertino app with a button, "Go", that shows the sheet in a modal popup.
func launchSheet(sheet func(bc widgets.BuildContext) widgets.Widget, opts ...cupertino.CupertinoAppOpts) widgets.Widget {
	o := cupertino.CupertinoAppOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Home = widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoButton(func() { cupertino.ShowCupertinoModalPopup(bc, sheet) }, widgets.NewText("Go"))
	}))
	return testApp(o)
}

func sheetAction(label string, onPressed func(), opts ...cupertino.CupertinoActionSheetActionOpts) widgets.Widget {
	return cupertino.NewCupertinoActionSheetAction(onPressed, widgets.NewText(label), opts...)
}

func showSheet(t *testing.T, size ui.Size, sheet func(bc widgets.BuildContext) widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := dialogTester(t, size)
	ts.PumpWidget(launchSheet(sheet))
	ts.Tap(goButton())
	ts.PumpAndSettle()
	return ts
}

func TestCupertinoActionSheetTaps(t *testing.T) {
	t.Run("a tap on an action calls onPressed", func(t *testing.T) {
		pressed := false
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
				Actions: []widgets.Widget{sheetAction("One", func() { pressed = true; pop(bc) }), sheetAction("Two", func() {})}})
		})
		if pressed {
			t.Fatal("pressed before the tap")
		}
		ts.Tap(gluttertest.ByText("One"))
		ts.PumpAndSettle()
		if !pressed || ts.Exists(gluttertest.ByText("One")) {
			t.Errorf("pressed %v, the sheet is still there %v", pressed, ts.Exists(gluttertest.ByText("One")))
		}
	})
	t.Run("a tap on the cancel button calls onPressed", func(t *testing.T) {
		pressed := false
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
				Actions:      []widgets.Widget{sheetAction("One", func() {})},
				CancelButton: sheetAction("Cancel", func() { pressed = true; pop(bc) })})
		})
		ts.Tap(gluttertest.ByText("Cancel"))
		ts.PumpAndSettle()
		if !pressed || ts.Exists(gluttertest.ByText("Cancel")) {
			t.Errorf("pressed %v", pressed)
		}
	})
	t.Run("a tap on the padding of an action calls onPressed", func(t *testing.T) {
		pressed := false
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
				Actions: []widgets.Widget{sheetAction("One", func() { pressed = true })}})
		})
		at := ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoActionSheetAction]()).Add(ui.Offset{Dx: 3, Dy: 3})
		ts.TapAt(at)
		ts.Pump()
		if !pressed {
			t.Error("the tap on the padding did not press")
		}
	})
	t.Run("a slide from an action to another presses the other", func(t *testing.T) {
		var pressed []string
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
				Actions: []widgets.Widget{sheetAction("One", func() { pressed = append(pressed, "One") }),
					sheetAction("Two", func() { pressed = append(pressed, "Two") })}})
		})
		g := ts.StartGesture(ts.Center(gluttertest.ByText("Two")))
		ts.Pump()
		g.MoveTo(ts.Center(gluttertest.ByText("One")))
		ts.Pump()
		g.Up()
		ts.Pump()
		if len(pressed) != 1 || pressed[0] != "One" {
			t.Errorf("pressed %v", pressed)
		}
	})
	t.Run("a slide from an action to the cancel button presses it", func(t *testing.T) {
		var pressed []string
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
				Actions:      []widgets.Widget{sheetAction("One", func() { pressed = append(pressed, "One") })},
				CancelButton: sheetAction("Cancel", func() { pressed = append(pressed, "Cancel") })})
		})
		g := ts.StartGesture(ts.Center(gluttertest.ByText("One")))
		ts.Pump()
		g.MoveTo(ts.Center(gluttertest.ByText("Cancel")))
		ts.Pump()
		g.Up()
		ts.Pump()
		if len(pressed) != 1 || pressed[0] != "Cancel" {
			t.Errorf("pressed %v", pressed)
		}
	})
	t.Run("a slide that ends outside the buttons presses nothing", func(t *testing.T) {
		pressed := false
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
				Title:   widgets.NewText("Title"),
				Actions: []widgets.Widget{sheetAction("One", func() { pressed = true })}})
		})
		g := ts.StartGesture(ts.Center(gluttertest.ByText("One")))
		ts.Pump()
		g.MoveTo(ts.Center(gluttertest.ByText("Title")))
		ts.Pump()
		g.Up()
		ts.Pump()
		if pressed {
			t.Error("pressed")
		}
	})
	t.Run("an action without onPressed is not pressed by a tap", func(t *testing.T) {
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
				Actions: []widgets.Widget{cupertino.NewCupertinoActionSheetAction(nil, widgets.NewText("Off"))}})
		})
		ts.Tap(gluttertest.ByText("Off"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Off")) {
			t.Error("the sheet is gone")
		}
	})
}

func TestCupertinoActionSheetActionStyles(t *testing.T) {
	styleOf := func(ts *gluttertest.Tester, label string) *widgets.DefaultTextStyle {
		return ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText(label), gluttertest.ByType[*widgets.DefaultTextStyle](), gluttertest.DescendantOpts{}))).(*widgets.DefaultTextStyle)
	}
	ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{
			Actions: []widgets.Widget{sheetAction("Plain", func() {}),
				sheetAction("Default", func() {}, cupertino.CupertinoActionSheetActionOpts{IsDefaultAction: true}),
				sheetAction("Destructive", func() {}, cupertino.CupertinoActionSheetActionOpts{IsDestructiveAction: true})}})
	})
	plain, def, destructive := styleOf(ts, "Plain").Style(), styleOf(ts, "Default").Style(), styleOf(ts, "Destructive").Style()
	if plain.FontWeight != ui.FontWeightW400 || def.FontWeight != ui.FontWeightW600 {
		t.Errorf("weights: plain %v, default %v", plain.FontWeight, def.FontWeight)
	}
	if plain.Color == destructive.Color {
		t.Error("the destructive action has the color of the plain one")
	}
	if got, ok := destructive.Color.Get(); !ok || got != resolvedDestructiveRed(ts) {
		t.Errorf("destructive color %v", got)
	}
}

func resolvedDestructiveRed(ts *gluttertest.Tester) ui.Color {
	return cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.DestructiveRed, ts.Element(gluttertest.ByType[*cupertino.CupertinoActionSheet]()))
}

func TestCupertinoActionSheetSizes(t *testing.T) {
	actionsSection := func(ts *gluttertest.Tester, label string) ui.Size {
		return ts.Size(gluttertest.Ancestor(gluttertest.ByText(label), gluttertest.ByType[*widgets.SingleChildScrollView](), gluttertest.DescendantOpts{}))
	}
	long := func() widgets.Widget {
		text := ""
		for range 200 {
			text += "Very long content"
		}
		return widgets.NewText(text)
	}
	t.Run("one action with a very long message has the height of the button", func(t *testing.T) {
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{Title: widgets.NewText("The title"), Message: long(),
				Actions: []widgets.Widget{sheetAction("One", func() {})}})
		})
		if got := actionsSection(ts, "One").Height; math.Abs(got-57.17) > 0.01 {
			t.Errorf("height %v, want 57.17", got)
		}
	})
	t.Run("two actions with a very long message have the minimum height", func(t *testing.T) {
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{Title: widgets.NewText("The title"), Message: long(),
				Actions: []widgets.Widget{sheetAction("One", func() {}), sheetAction("Two", func() {})}})
		})
		if got := actionsSection(ts, "One").Height; math.Abs(got-84.0) > 0.01 {
			t.Errorf("height %v, want 84", got)
		}
	})
	t.Run("two actions with a cancel button and a very long message have the minimum height", func(t *testing.T) {
		ts := showSheet(t, ui.Size{}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{Title: widgets.NewText("The title"), Message: long(),
				Actions:      []widgets.Widget{sheetAction("One", func() {}), sheetAction("Two", func() {})},
				CancelButton: sheetAction("Cancel", func() {})})
		})
		if got := actionsSection(ts, "One").Height; math.Abs(got-84.0) > 0.01 {
			t.Errorf("height %v, want 84", got)
		}
	})
	t.Run("the sheet is as wide as the shorter side of the view, in a landscape view", func(t *testing.T) {
		ts := showSheet(t, ui.Size{Width: 800, Height: 600}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{CancelButton: sheetAction("Cancel", func() {})})
		})
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoActionSheet]()).Width; got != 600 {
			t.Errorf("width %v, want 600", got)
		}
	})
	t.Run("the sheet is as wide as the view, in a portrait view", func(t *testing.T) {
		ts := showSheet(t, ui.Size{Width: 400, Height: 800}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{CancelButton: sheetAction("Cancel", func() {})})
		})
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoActionSheet]()).Width; got != 400 {
			t.Errorf("width %v, want 400", got)
		}
	})
	t.Run("the sheet of a cancel button only is the height of the button and the paddings", func(t *testing.T) {
		ts := showSheet(t, ui.Size{Width: 400, Height: 800}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{CancelButton: sheetAction("Cancel", func() {})})
		})
		button := ts.Size(gluttertest.ByType[*cupertino.CupertinoActionSheetAction]())
		sheet := ts.Size(gluttertest.ByType[*cupertino.CupertinoActionSheet]())
		// The top padding (status bar or the edge padding), the button, and no space above the cancel button.
		if sheet.Height < button.Height || button.Width != 400-16 {
			t.Errorf("sheet %v, button %v", sheet, button)
		}
	})
	t.Run("the actions of a tall sheet scroll", func(t *testing.T) {
		var actions []widgets.Widget
		for i := range 30 {
			actions = append(actions, sheetAction(string(rune('A'+i%26))+string(rune('a'+i/26)), func() {}))
		}
		ts := showSheet(t, ui.Size{Width: 400, Height: 600}, func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{Actions: actions, CancelButton: sheetAction("Cancel", func() {})})
		})
		sheet := ts.Size(gluttertest.ByType[*cupertino.CupertinoActionSheet]())
		if sheet.Height > 600 {
			t.Errorf("sheet %v is higher than the view", sheet)
		}
		before := ts.TopLeft(gluttertest.ByText("Aa")).Dy
		ts.DragFrom(ts.Center(gluttertest.ByText("Aa")), ui.Offset{Dy: -100})
		ts.PumpAndSettle()
		if after := ts.TopLeft(gluttertest.ByText("Aa")).Dy; after >= before {
			t.Errorf("the actions did not scroll: %v -> %v", before, after)
		}
	})
}

func TestCupertinoActionSheetZeroArea(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 400, Height: 800})
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBox(
		cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{Actions: []widgets.Widget{sheetAction("One", func() {})}}),
		widgets.SizedBoxOpts{Width: foundation.Some(0.0), Height: foundation.Some(0.0)}))}))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestCupertinoActionSheetEmptyRoute(t *testing.T) {
	skipUnlessDebug(t)
	err := recoverError(func() { cupertino.NewCupertinoActionSheet() })
	requireErrorTag(t, err, "NewCupertinoActionSheet.CupertinoActionSheetEmpty")
	if err := recoverError(func() {
		cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{Actions: []widgets.Widget{}})
	}); err != nil {
		t.Errorf("an empty non-nil actions slice is a sheet with actions: %v", err)
	}
}
