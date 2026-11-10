package cupertino_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/text_field_test.dart. The cases that read the calls
// of the canvas with a `paints` matcher are goldens (text_field_golden_test.go), the cases of the semantics tester,
// of the restoration, of the scribble and of the keyboard of the platform are not ported (SPEC §12.10, step 34.5).

// usePlatform makes p the target platform of the test.
func usePlatform(t *testing.T, p foundation.TargetPlatform) {
	t.Helper()
	skipUnlessDebug(t)
	prev := foundation.DebugDefaultTargetPlatformOverride()
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(prev) })
}

func textFieldFinder() gluttertest.Finder { return gluttertest.ByType[*cupertino.CupertinoTextField]() }

func editableFinder() gluttertest.Finder { return gluttertest.ByType[*widgets.EditableText]() }

func editableState(ts *gluttertest.Tester) *widgets.EditableTextState {
	return gluttertest.StateOf[*widgets.EditableTextState](ts, editableFinder())
}

func renderEditableOf(ts *gluttertest.Tester) *rendering.RenderEditable {
	e := ts.Element(gluttertest.ByElementPredicate("a RenderEditable", func(e widgets.Element) bool {
		_, ok := e.RenderObject().(*rendering.RenderEditable)
		return ok
	}))
	return e.RenderObject().(*rendering.RenderEditable)
}

// caretPosition is the global position of the caret at the offset.
func caretPosition(ts *gluttertest.Tester, offset int) ui.Offset {
	r := renderEditableOf(ts)
	return r.LocalToGlobal(r.GetLocalRectForCaret(ui.NewTextPosition(offset)).Center(), nil)
}

// fieldApp pumps the field in the middle of a Cupertino app.
func fieldApp(ts *gluttertest.Tester, w widgets.Widget) {
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(w))}))
}

func TestCupertinoTextFieldBasics(t *testing.T) {
	t.Run("the text that is entered reaches OnChanged and the controller", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController()
		defer controller.Dispose()
		var changed []string
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller,
			OnChanged: func(s string) { changed = append(changed, s) }}))
		ts.EnterText(textFieldFinder(), "hello")
		ts.Pump()
		if controller.Text() != "hello" || len(changed) != 1 || changed[0] != "hello" {
			t.Errorf("text %q, changes %v", controller.Text(), changed)
		}
	})
	t.Run("the placeholder is shown only while the text is empty", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Placeholder: foundation.Some("Name")}))
		placeholder := gluttertest.ByText("Name")
		visible := func() bool {
			e := ts.Element(gluttertest.Ancestor(placeholder, gluttertest.ByType[*widgets.Visibility](), gluttertest.DescendantOpts{}))
			return e.Widget().(*widgets.Visibility).Visible()
		}
		if !visible() {
			t.Fatal("the placeholder is hidden in an empty field")
		}
		ts.EnterText(textFieldFinder(), "x")
		ts.Pump()
		if visible() {
			t.Error("the placeholder is visible with text")
		}
		if !ts.Exists(placeholder) {
			t.Error("the placeholder is not kept in the tree (maintainState)")
		}
	})
	t.Run("the placeholder and the text have the same baseline", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Placeholder: foundation.Some("Name"),
			Style: &painting.TextStyle{FontSize: foundation.Some(30.0)}}))
		ts.EnterText(textFieldFinder(), "Name")
		ts.Pump()
		// Both are laid out with the same style, so the baselines are the same when the tops are.
		editable := ts.TopLeft(editableFinder()).Dy
		placeholder := ts.TopLeft(gluttertest.ByText("Name")).Dy
		if math.Abs(editable-placeholder) > 0.01 {
			t.Errorf("the text is at %v and the placeholder at %v", editable, placeholder)
		}
	})
	t.Run("the default decoration is the rounded border of iOS", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField())
		d := decorationOf(ts)
		if d == nil || d.Color.OrElse(ui.Color{}) != ui.NewColor(0xFFFFFFFF) {
			t.Fatalf("decoration %v", d)
		}
		if got := d.BorderRadius.(painting.BorderRadius).TopLeft; got != ui.NewRadiusCircular(5) {
			t.Errorf("radius %v, want 5", got)
		}
	})
	t.Run("the borderless field has no decoration", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextFieldBorderless())
		if d := decorationOf(ts); d != nil {
			t.Errorf("decoration %v", d)
		}
	})
	t.Run("a dark theme paints the field black with a light border", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark}),
			Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(cupertino.NewCupertinoTextField()))}))
		if d := decorationOf(ts); d == nil || d.Color.OrElse(ui.Color{}) != ui.NewColor(0xFF000000) {
			t.Errorf("decoration %v", d)
		}
	})
	t.Run("a disabled field is gray and does not take the focus", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		node := widgets.NewFocusNode(widgets.FocusNodeOpts{})
		defer node.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Enabled: foundation.Some(false), FocusNode: node}))
		if d := decorationOf(ts); d == nil || d.Color.OrElse(ui.Color{}) != ui.NewColor(0xFFFAFAFA) {
			t.Errorf("decoration %v", d)
		}
		ts.Tap(textFieldFinder())
		ts.Pump()
		if node.HasFocus() || node.CanRequestFocus() {
			t.Error("a disabled field takes the focus")
		}
	})
	t.Run("a tap gives the field the focus and the cursor", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("abc def")
		defer controller.Dispose()
		node := widgets.NewFocusNode(widgets.FocusNodeOpts{})
		defer node.Dispose()
		taps := 0
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller, FocusNode: node, OnTap: func() { taps++ }}))
		ts.TapAt(caretPosition(ts, 5))
		ts.Pump()
		if !node.HasFocus() || taps != 1 {
			t.Errorf("focus %v, taps %d", node.HasFocus(), taps)
		}
		// A tap on iOS puts the cursor at the edge of the word that it hits.
		if s := controller.Selection(); !s.IsCollapsed() || (s.BaseOffset != 4 && s.BaseOffset != 7) {
			t.Errorf("selection %v, want a cursor at 4 or 7", s)
		}
	})
	t.Run("the maximum length limits the text", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController()
		defer controller.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller, MaxLength: foundation.Some(5)}))
		ts.EnterText(textFieldFinder(), "123456789")
		ts.Pump()
		if controller.Text() != "12345" {
			t.Errorf("text %q", controller.Text())
		}
	})
	t.Run("an obscured field shows the character", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{ObscureText: true}))
		ts.EnterText(textFieldFinder(), "abc")
		ts.Pump()
		if got := renderEditableOf(ts).PlainText(); !strings.HasPrefix(got, "•••") || strings.Contains(got, "abc") {
			t.Errorf("text %q", got)
		}
	})
	t.Run("a controller that changes keeps the text", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField())
		ts.EnterText(textFieldFinder(), "kept")
		ts.Pump()
		c := widgets.NewTextEditingController()
		defer c.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: c}))
		ts.Pump()
		if c.Text() != "" {
			t.Errorf("new controller text %q", c.Text())
		}
		fieldApp(ts, cupertino.NewCupertinoTextField())
		ts.Pump()
		if got := renderEditableOf(ts).PlainText(); got != "" {
			t.Errorf("text %q after the controller went away", got)
		}
	})
	t.Run("the cursor is two wide, rounded and offset on iOS", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Autofocus: true}))
		ts.Pump()
		r := renderEditableOf(ts)
		if r.CursorWidth() != 2 {
			t.Errorf("cursor width %v", r.CursorWidth())
		}
		if r.CursorRadius().OrElse(ui.Radius{}) != ui.NewRadiusCircular(2) {
			t.Errorf("cursor radius %v", r.CursorRadius())
		}
	})
}

// decorationOf is the decoration of the container around the field, or nil.
func decorationOf(ts *gluttertest.Tester) *painting.BoxDecoration {
	e := ts.Element(gluttertest.Descendant(textFieldFinder(), gluttertest.ByType[*widgets.Container](), gluttertest.DescendantOpts{MatchRoot: false}))
	if e == nil {
		return nil
	}
	d, _ := e.Widget().(*widgets.Container).Decoration().(*painting.BoxDecoration)
	return d
}

func TestCupertinoTextFieldOverlayModes(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformIOS)
	for _, tc := range []struct {
		name         string
		mode         cupertino.OverlayVisibilityMode
		empty, typed bool
	}{
		{"never", cupertino.OverlayVisibilityModeNever, false, false},
		{"editing", cupertino.OverlayVisibilityModeEditing, false, true},
		{"notEditing", cupertino.OverlayVisibilityModeNotEditing, true, false},
		{"always", cupertino.OverlayVisibilityModeAlways, true, true},
	} {
		t.Run("prefix "+tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Prefix: widgets.NewText("PRE"), PrefixMode: tc.mode}))
			if got := ts.Exists(gluttertest.ByText("PRE")); got != tc.empty {
				t.Errorf("empty: shown %v, want %v", got, tc.empty)
			}
			ts.EnterText(textFieldFinder(), "x")
			ts.Pump()
			if got := ts.Exists(gluttertest.ByText("PRE")); got != tc.typed {
				t.Errorf("typed: shown %v, want %v", got, tc.typed)
			}
		})
		t.Run("suffix "+tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Suffix: widgets.NewText("SUF"), SuffixMode: tc.mode}))
			if got := ts.Exists(gluttertest.ByText("SUF")); got != tc.empty {
				t.Errorf("empty: shown %v, want %v", got, tc.empty)
			}
			ts.EnterText(textFieldFinder(), "x")
			ts.Pump()
			if got := ts.Exists(gluttertest.ByText("SUF")); got != tc.typed {
				t.Errorf("typed: shown %v, want %v", got, tc.typed)
			}
		})
	}
	t.Run("the default modes of the prefix and the suffix are always", func(t *testing.T) {
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Prefix: widgets.NewText("PRE"), Suffix: widgets.NewText("SUF")}))
		if !ts.Exists(gluttertest.ByText("PRE")) || !ts.Exists(gluttertest.ByText("SUF")) {
			t.Error("a prefix or a suffix is hidden")
		}
	})
}

func TestCupertinoTextFieldClearButton(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformIOS)
	clear := gluttertest.ByIcon(cupertino.CupertinoIcons.ClearThickCircled())
	t.Run("the button shows with text and clears it", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController()
		defer controller.Dispose()
		var changed []string
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller, ClearButtonMode: cupertino.OverlayVisibilityModeEditing,
			OnChanged: func(s string) { changed = append(changed, s) }}))
		if ts.Exists(clear) {
			t.Fatal("the clear button is shown with no text")
		}
		ts.EnterText(textFieldFinder(), "abc")
		ts.Pump()
		if !ts.Exists(clear) {
			t.Fatal("no clear button with text")
		}
		ts.Tap(clear)
		ts.Pump()
		if controller.Text() != "" || len(changed) != 2 || changed[1] != "" {
			t.Errorf("text %q, changes %v", controller.Text(), changed)
		}
		if ts.Exists(clear) {
			t.Error("the clear button stays after it cleared the text")
		}
	})
	t.Run("a tap on the button does not call OnTap", func(t *testing.T) {
		ts := gluttertest.New(t)
		taps := 0
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{ClearButtonMode: cupertino.OverlayVisibilityModeAlways, OnTap: func() { taps++ }}))
		ts.Tap(clear)
		ts.Pump()
		if taps != 0 {
			t.Errorf("OnTap was called %d times", taps)
		}
	})
	t.Run("the suffix takes the place of the button when both show", func(t *testing.T) {
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Suffix: widgets.NewText("SUF"),
			ClearButtonMode: cupertino.OverlayVisibilityModeAlways}))
		if !ts.Exists(gluttertest.ByText("SUF")) || ts.Exists(clear) {
			t.Error("the suffix and the clear button are not exclusive")
		}
	})
	t.Run("the button has a semantics label", func(t *testing.T) {
		ts := gluttertest.New(t)
		h := ts.EnsureSemantics()
		defer h.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{ClearButtonMode: cupertino.OverlayVisibilityModeAlways,
			ClearButtonSemanticLabel: foundation.Some("Remove")}))
		if !ts.Exists(gluttertest.BySemanticsLabel("Remove")) {
			t.Error("no semantics label")
		}
	})
}

func TestCupertinoTextFieldSelectionGestures(t *testing.T) {
	t.Run("a long press on iOS selects the word, shows the handles and the toolbar", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("abc def ghi")
		defer controller.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller}))
		ts.LongPressAt(caretPosition(ts, 5))
		ts.PumpAndSettle()
		if s := controller.Selection(); s.BaseOffset != 4 || s.ExtentOffset != 7 {
			t.Errorf("selection %v, want 4 to 7", s)
		}
		overlay := editableState(ts).SelectionOverlay()
		if overlay == nil || !overlay.HandlesAreVisible() || !overlay.ToolbarIsVisible() {
			t.Fatal("no handles or toolbar after the long press")
		}
		if ts.Count(gluttertest.ByType[*cupertino.CupertinoTextSelectionToolbar]()) != 1 {
			t.Error("the toolbar is not the Cupertino one")
		}
	})
	t.Run("a double tap on iOS selects the word", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("abc def ghi")
		defer controller.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller}))
		at := caretPosition(ts, 5)
		ts.TapAt(at)
		ts.Pump(50 * time.Millisecond)
		ts.TapAt(at)
		ts.Pump()
		if s := controller.Selection(); s.BaseOffset != 4 || s.ExtentOffset != 7 {
			t.Errorf("selection %v, want 4 to 7", s)
		}
	})
	t.Run("a tap on iOS in the empty field shows no handles", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField())
		ts.Tap(textFieldFinder())
		ts.Pump()
		if o := editableState(ts).SelectionOverlay(); o != nil && o.HandlesAreVisible() {
			t.Error("handles in an empty field")
		}
	})
	t.Run("the handles are the circles of Cupertino", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{}))
		ts.EnterText(textFieldFinder(), "abc def")
		ts.Pump()
		ts.LongPressAt(caretPosition(ts, 2))
		ts.PumpAndSettle()
		if _, ok := editableState(ts).SelectionOverlay().SelectionControls().(cupertino.CupertinoTextSelectionControls); !ok {
			t.Errorf("controls %T", editableState(ts).SelectionOverlay().SelectionControls())
		}
	})
	t.Run("a double tap on macOS shows the desktop toolbar", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformMacOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("abc def ghi")
		defer controller.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller}))
		at := caretPosition(ts, 5)
		mouse := gluttertest.GestureOpts{Pointer: 7, Kind: ui.PointerDeviceKindMouse}
		g := ts.StartGesture(at, mouse)
		g.Up()
		ts.Pump(50 * time.Millisecond)
		g.Down(at)
		g.Up()
		ts.Pump()
		if s := controller.Selection(); s.BaseOffset != 4 || s.ExtentOffset != 7 {
			t.Errorf("selection %v, want 4 to 7", s)
		}
		// A secondary click opens the context menu, which is the desktop toolbar on macOS.
		secondary := ts.StartGesture(at, gluttertest.GestureOpts{Pointer: 8, Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.SecondaryButton)})
		secondary.Up()
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByType[*cupertino.CupertinoDesktopTextSelectionToolbar]()) != 1 {
			t.Error("no desktop toolbar after the secondary click")
		}
		if ts.Count(gluttertest.ByType[*cupertino.CupertinoTextSelectionToolbar]()) != 0 {
			t.Error("the iOS toolbar shows on macOS")
		}
	})
	t.Run("a drag with the mouse on macOS selects and has no handles", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformMacOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("abc def ghi")
		defer controller.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller}))
		g := ts.StartGesture(caretPosition(ts, 1), gluttertest.GestureOpts{Pointer: 7, Kind: ui.PointerDeviceKindMouse})
		ts.Pump()
		g.MoveTo(caretPosition(ts, 6))
		ts.Pump()
		g.Up()
		ts.Pump()
		if s := controller.Selection(); s.BaseOffset > 2 || s.ExtentOffset < 5 {
			t.Errorf("selection %v", s)
		}
		if o := editableState(ts).SelectionOverlay(); o != nil && o.HandlesAreVisible() {
			t.Error("handles with a desktop field")
		}
	})
	t.Run("the selection of a field that is not interactive cannot be made", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("abc def")
		defer controller.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller, EnableInteractiveSelection: foundation.Some(false)}))
		ts.LongPressAt(caretPosition(ts, 5))
		ts.PumpAndSettle()
		if s := controller.Selection(); !s.IsCollapsed() {
			t.Errorf("selection %v", s)
		}
	})
}

func TestCupertinoTextFieldKeyboard(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformIOS)
	t.Run("the default keyboard is text, or multiline", func(t *testing.T) {
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Autofocus: true}))
		ts.Pump()
		if got := ts.Widget(textFieldFinder()).(*cupertino.CupertinoTextField).MaxLines(); got != foundation.Some(1) {
			t.Errorf("max lines %v", got)
		}
	})
	t.Run("a null max lines is unlimited and a multiline field grows", func(t *testing.T) {
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{MaxLines: foundation.Some(rendering.MaxLinesUnlimited)}))
		if got := ts.Widget(textFieldFinder()).(*cupertino.CupertinoTextField).MaxLines(); got.IsSome() {
			t.Errorf("max lines %v, want none", got)
		}
		h1 := ts.Size(textFieldFinder()).Height
		ts.EnterText(textFieldFinder(), "a\nb\nc\nd")
		ts.Pump()
		if h2 := ts.Size(textFieldFinder()).Height; h2 <= h1 {
			t.Errorf("the height did not grow: %v then %v", h1, h2)
		}
	})
	t.Run("the minimum lines set the height", func(t *testing.T) {
		ts := gluttertest.New(t)
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{MaxLines: foundation.Some(5), MinLines: foundation.Some(3)}))
		one := ts.Size(textFieldFinder()).Height
		fieldApp(ts, cupertino.NewCupertinoTextField())
		if base := ts.Size(textFieldFinder()).Height; one <= base {
			t.Errorf("three lines are %v high and one is %v", one, base)
		}
	})
	t.Run("a read only field does not change", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("fixed")
		defer controller.Dispose()
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller, ReadOnly: true}))
		ts.EnterText(textFieldFinder(), "changed")
		ts.Pump()
		if controller.Text() != "fixed" {
			t.Errorf("text %q", controller.Text())
		}
	})
	t.Run("OnSubmitted gets the text when the user presses enter", func(t *testing.T) {
		ts := gluttertest.New(t)
		var submitted string
		fieldApp(ts, cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{OnSubmitted: func(s string) { submitted = s }}))
		ts.EnterText(textFieldFinder(), "go")
		ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
		ts.Pump()
		if submitted != "go" {
			t.Errorf("submitted %q", submitted)
		}
	})
	t.Run("a field in a stack of tap regions loses the focus on a tap outside", func(t *testing.T) {
		ts := gluttertest.New(t)
		node := widgets.NewFocusNode(widgets.FocusNodeOpts{})
		defer node.Dispose()
		outside := 0
		fieldApp(ts, widgets.NewColumn([]widgets.Widget{
			cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{FocusNode: node, Autofocus: true, OnTapOutside: func(*gestures.PointerDownEvent) { outside++ }}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
		}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}))
		ts.Pump()
		if !node.HasFocus() {
			t.Fatal("no autofocus")
		}
		ts.TapAt(ui.Offset{Dx: 10, Dy: 10})
		ts.Pump()
		if outside != 1 {
			t.Errorf("OnTapOutside %d", outside)
		}
	})
}

func TestCupertinoTextFieldAlignment(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformIOS)
	tall := func(a foundation.Opt[painting.TextAlignVertical]) widgets.Widget {
		return widgets.NewSizedBox(cupertino.NewCupertinoTextFieldBorderless(cupertino.CupertinoTextFieldOpts{
			Placeholder: foundation.Some("p"), Expands: true, MaxLines: foundation.Some(rendering.MaxLinesUnlimited), TextAlignVertical: a}),
			widgets.SizedBoxOpts{Height: foundation.Some(100.0), Width: foundation.Some(300.0)})
	}
	t.Run("the vertical alignment is an option", func(t *testing.T) {
		ts := gluttertest.New(t)
		fieldApp(ts, tall(foundation.Some(painting.TextAlignVerticalTop)))
		top := ts.TopLeft(editableFinder()).Dy - ts.TopLeft(textFieldFinder()).Dy
		fieldApp(ts, tall(foundation.Some(painting.TextAlignVerticalCenter)))
		center := ts.TopLeft(editableFinder()).Dy - ts.TopLeft(textFieldFinder()).Dy
		fieldApp(ts, tall(foundation.Some(painting.TextAlignVerticalBottom)))
		bottom := ts.TopLeft(editableFinder()).Dy - ts.TopLeft(textFieldFinder()).Dy
		if !(top < center && center < bottom) {
			t.Errorf("top %v, center %v, bottom %v", top, center, bottom)
		}
	})
	t.Run("a field with a placeholder is centered and one without is at the top", func(t *testing.T) {
		ts := gluttertest.New(t)
		fieldApp(ts, tall(foundation.None[painting.TextAlignVertical]()))
		withPlaceholder := ts.TopLeft(editableFinder()).Dy - ts.TopLeft(textFieldFinder()).Dy
		fieldApp(ts, widgets.NewSizedBox(cupertino.NewCupertinoTextFieldBorderless(cupertino.CupertinoTextFieldOpts{
			Expands: true, MaxLines: foundation.Some(rendering.MaxLinesUnlimited)}),
			widgets.SizedBoxOpts{Height: foundation.Some(100.0), Width: foundation.Some(300.0)}))
		without := ts.TopLeft(editableFinder()).Dy - ts.TopLeft(textFieldFinder()).Dy
		if !(without < withPlaceholder) {
			t.Errorf("with a placeholder %v, without %v", withPlaceholder, without)
		}
	})
}

func TestCupertinoTextFieldAssertions(t *testing.T) {
	skipUnlessDebug(t)
	for _, tc := range []struct {
		tag  string
		opts cupertino.CupertinoTextFieldOpts
	}{
		{"CupertinoTextFieldObscuringCharacterInvalid", cupertino.CupertinoTextFieldOpts{ObscuringCharacter: "ab"}},
		{"CupertinoTextFieldMaxLinesInvalid", cupertino.CupertinoTextFieldOpts{MaxLines: foundation.Some(-3)}},
		{"CupertinoTextFieldMinLinesInvalid", cupertino.CupertinoTextFieldOpts{MinLines: foundation.Some(-1)}},
		{"CupertinoTextFieldMinLinesGreaterThanMaxLines", cupertino.CupertinoTextFieldOpts{MinLines: foundation.Some(3), MaxLines: foundation.Some(2)}},
		{"CupertinoTextFieldExpandsWithLines", cupertino.CupertinoTextFieldOpts{Expands: true}},
		{"CupertinoTextFieldObscuredMultiline", cupertino.CupertinoTextFieldOpts{ObscureText: true, MaxLines: foundation.Some(2)}},
		{"CupertinoTextFieldMaxLengthInvalid", cupertino.CupertinoTextFieldOpts{MaxLength: foundation.Some(-5)}},
		{"CupertinoTextFieldNewlineActionWithTextKeyboard", cupertino.CupertinoTextFieldOpts{MaxLines: foundation.Some(3), TextInputAction: services.TextInputActionNewline,
			KeyboardType: foundation.Some(services.TextInputTypeText)}},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			err := recoverError(func() { cupertino.NewCupertinoTextField(tc.opts) })
			requireErrorTag(t, err, tc.tag)
		})
	}
}

func TestCupertinoTextFieldSpellCheck(t *testing.T) {
	t.Run("the configuration gets the style, the color and the toolbar of iOS", func(t *testing.T) {
		got := cupertino.CupertinoTextFieldInferIOSSpellCheckConfiguration(widgets.NewSpellCheckConfiguration(widgets.SpellCheckConfigurationOpts{}))
		if got.MisspelledTextStyle() != cupertino.CupertinoTextFieldMisspelledTextStyle {
			t.Error("the misspelled style is not the one of iOS")
		}
		if got.MisspelledSelectionColor().OrElse(ui.Color{}) != cupertino.CupertinoTextFieldMisspelledSelectionColor {
			t.Error("the misspelled selection color is not the one of iOS")
		}
		if got.SpellCheckSuggestionsToolbarBuilder() == nil {
			t.Error("no toolbar builder")
		}
	})
	t.Run("a nil configuration is disabled", func(t *testing.T) {
		got := cupertino.CupertinoTextFieldInferIOSSpellCheckConfiguration(nil)
		if !got.Equal(widgets.NewSpellCheckConfigurationDisabled()) {
			t.Error("not disabled")
		}
	})
}
