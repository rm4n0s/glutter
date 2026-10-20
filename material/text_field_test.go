// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func str(s string) foundation.Opt[string] { return foundation.Some(s) }

// pumpField builds the field in a Material app: a centered box 300 wide.
func pumpField(t *testing.T, field widgets.Widget, opts ...gluttertest.Opts) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, opts...)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
		Body: widgets.NewCenter(widgets.NewSizedBox(field, widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))})}))
	return ts
}

func findField() gluttertest.Finder { return gluttertest.ByType[*TextField]() }

// rightClick presses the secondary button of the mouse at the position.
func rightClick(ts *gluttertest.Tester, at ui.Offset) {
	ts.TapAt(at, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.SecondaryButton)})
	ts.Pump()
}

func TestTextFieldTypingAndCallbacks(t *testing.T) {
	controller := widgets.NewTextEditingController()
	t.Cleanup(controller.Dispose)
	var changes []string
	ts := pumpField(t, NewTextField(TextFieldOpts{
		Controller: controller, Decoration: &InputDecoration{LabelText: str("Name")},
		OnChanged: func(s string) { changes = append(changes, s) },
	}))
	if !ts.Exists(gluttertest.ByText("Name")) {
		t.Fatal("no label")
	}
	ts.Tap(findField())
	ts.Pump()
	ts.EnterText(findField(), "hello")
	if controller.Text() != "hello" {
		t.Errorf("text = %q", controller.Text())
	}
	if len(changes) == 0 || changes[len(changes)-1] != "hello" {
		t.Errorf("onChanged calls = %q", changes)
	}
	if !ts.Exists(gluttertest.ByText("Name")) {
		t.Error("the label is gone after typing")
	}
}

func TestTextFieldFocusStates(t *testing.T) {
	states := widgets.NewWidgetStatesController()
	t.Cleanup(states.Dispose)
	ts := pumpField(t, NewTextField(TextFieldOpts{StatesController: states}))
	tests := []struct {
		name  string
		do    func()
		state widgets.WidgetState
		want  bool
	}{
		{"initially not focused", func() {}, widgets.WidgetStateFocused, false},
		{"tap focuses", func() { ts.Tap(findField()); ts.Pump() }, widgets.WidgetStateFocused, true},
		{"hover", func() { ts.Hover(ts.Center(findField())); ts.Pump() }, widgets.WidgetStateHovered, true},
		{"leaving the field ends the hover", func() { ts.Hover(ui.Offset{Dx: 1, Dy: 1}); ts.Pump() }, widgets.WidgetStateHovered, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.do()
			if got := states.Value().Has(tc.state); got != tc.want {
				t.Errorf("%v = %v, want %v (states %v)", tc.state, got, tc.want, states.Value())
			}
		})
	}
}

func TestTextFieldCounter(t *testing.T) {
	tests := []struct {
		name        string
		opts        TextFieldOpts
		text        string
		wantCounter string
		wantError   bool
		wantText    string
	}{
		{"counts the characters", TextFieldOpts{MaxLength: foundation.Some(10)}, "abc", "3/10", false, "abc"},
		{"truncates over the limit", TextFieldOpts{MaxLength: foundation.Some(5), MaxLengthEnforcement: services.MaxLengthEnforcementEnforced}, "abcdefgh", "5/5", false, "abcde"},
		{"error over the limit", TextFieldOpts{MaxLength: foundation.Some(5), MaxLengthEnforcement: services.MaxLengthEnforcementNone}, "abcdefgh", "8/5", true, "abcdefgh"},
		{"counts without limit", TextFieldOpts{MaxLength: foundation.Some(TextFieldNoMaxLength)}, "abc", "3", false, "abc"},
		{"counts graphemes", TextFieldOpts{MaxLength: foundation.Some(10)}, "a👨‍👩‍👧b", "3/10", false, "a👨‍👩‍👧b"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controller := widgets.NewTextEditingController()
			t.Cleanup(controller.Dispose)
			tc.opts.Controller = controller
			ts := pumpField(t, NewTextField(tc.opts))
			ts.EnterText(findField(), tc.text)
			if controller.Text() != tc.wantText {
				t.Errorf("text = %q, want %q", controller.Text(), tc.wantText)
			}
			if !ts.Exists(gluttertest.ByText(tc.wantCounter)) {
				t.Fatalf("no counter %q", tc.wantCounter)
			}
			scheme := ThemeOf(ts.Element(findField())).ColorScheme
			style := ts.Widget(gluttertest.Descendant(gluttertest.ByText(tc.wantCounter), gluttertest.ByType[*widgets.Text](), gluttertest.DescendantOpts{MatchRoot: true})).(*widgets.Text).Style()
			isError := style != nil && style.Color == foundation.Some(scheme.Error())
			if isError != tc.wantError {
				t.Errorf("counter in the error color = %v, want %v", isError, tc.wantError)
			}
		})
	}
}

func TestTextFieldBuildCounter(t *testing.T) {
	tests := []struct {
		name    string
		builder InputCounterWidgetBuilder
		want    string
		none    bool
	}{
		{"custom counter", func(bc widgets.BuildContext, n int, max foundation.Opt[int], focused bool) widgets.Widget {
			return widgets.NewText("n=" + string(rune('0'+n)))
		}, "n=2", false},
		{"nil counter", func(widgets.BuildContext, int, foundation.Opt[int], bool) widgets.Widget { return nil }, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controller := widgets.NewTextEditingController()
			t.Cleanup(controller.Dispose)
			ts := pumpField(t, NewTextField(TextFieldOpts{Controller: controller, BuildCounter: tc.builder}))
			ts.EnterText(findField(), "ab")
			if tc.none {
				if ts.Exists(gluttertest.ByTextContaining("n=")) || ts.Exists(gluttertest.ByTextContaining("2/")) {
					t.Error("a counter was built")
				}
				return
			}
			if !ts.Exists(gluttertest.ByText(tc.want)) {
				t.Errorf("no counter %q", tc.want)
			}
		})
	}
}

func TestTextFieldDisabled(t *testing.T) {
	ts := pumpField(t, NewTextField(TextFieldOpts{Enabled: foundation.Some(false), Decoration: &InputDecoration{LabelText: str("Name")}}))
	ts.Tap(findField())
	ts.Pump()
	editable := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText)
	if !editable.ReadOnly() {
		t.Error("a disabled field is not read-only")
	}
	if editable.FocusNode().HasFocus() {
		t.Error("a disabled field took the focus")
	}
	scheme := ThemeOf(ts.Element(findField())).ColorScheme
	want := scheme.OnSurface().WithOpacity(0.38)
	if got := editable.Style().Color; got != foundation.Some(want) {
		t.Errorf("text color = %v, want %v", got, want)
	}
}

func TestTextFieldSelectionControlsAndCursorColor(t *testing.T) {
	tests := []struct {
		name       string
		opts       TextFieldOpts
		wantCursor func(s *ColorScheme) ui.Color
	}{
		{"primary by default", TextFieldOpts{}, func(s *ColorScheme) ui.Color { return s.Primary() }},
		{"given color", TextFieldOpts{CursorColor: foundation.Some(ui.NewColor(0xFF00FF00))}, func(*ColorScheme) ui.Color { return ui.NewColor(0xFF00FF00) }},
		{"error color when the field has an error", TextFieldOpts{Decoration: &InputDecoration{ErrorText: str("bad")}}, func(s *ColorScheme) ui.Color { return s.Error() }},
		{"given error color", TextFieldOpts{Decoration: &InputDecoration{ErrorText: str("bad")}, CursorErrorColor: foundation.Some(ui.NewColor(0xFF0000FF))},
			func(*ColorScheme) ui.Color { return ui.NewColor(0xFF0000FF) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpField(t, NewTextField(tc.opts))
			scheme := ThemeOf(ts.Element(findField())).ColorScheme
			r := ts.RenderObject(gluttertest.Descendant(findField(), gluttertest.ByElementPredicate("render editable", func(e widgets.Element) bool {
				_, ok := e.AsElement().RenderObject().(*rendering.RenderEditable)
				return ok
			}))).(*rendering.RenderEditable)
			got := r.CursorColor().OrElse(ui.Color{})
			// The caret blinks: its alpha is the opacity of the blink.
			if got, want := got.WithOpacity(1), tc.wantCursor(scheme).WithOpacity(1); got != want {
				t.Errorf("cursor color = %v, want %v", got, want)
			}
		})
	}
}

func TestTextFieldRightClickShowsTheContextMenu(t *testing.T) {
	controller := widgets.NewTextEditingController("hello world")
	t.Cleanup(controller.Dispose)
	ts := pumpField(t, NewTextField(TextFieldOpts{Controller: controller}))
	rightClick(ts, ts.Center(findField()))
	for _, label := range []string{"Select all"} {
		if !ts.Exists(gluttertest.ByText(label)) {
			t.Errorf("the context menu has no %q button", label)
		}
	}
	ts.Tap(gluttertest.ByText("Select all"))
	ts.Pump()
	if sel := controller.Selection(); sel.Start() != 0 || sel.End() != len("hello world") {
		t.Errorf("selection = %v after Select all", sel)
	}
}

func TestTextFieldContextMenuHasCutAndCopyWithASelection(t *testing.T) {
	controller := widgets.NewTextEditingController("hello world")
	t.Cleanup(controller.Dispose)
	ts := pumpField(t, NewTextField(TextFieldOpts{Controller: controller}))
	ts.Tap(findField())
	ts.Pump()
	controller.SetSelection(services.NewTextSelection(0, 5))
	ts.Pump()
	rightClick(ts, ts.Center(findField()))
	for _, label := range []string{"Cut", "Copy"} {
		if !ts.Exists(gluttertest.ByText(label)) {
			t.Errorf("the context menu has no %q button", label)
		}
	}
	ts.Tap(gluttertest.ByText("Copy"))
	ts.Pump()
	if text, _ := ts.ClipboardText(); text != "hello" {
		t.Errorf("clipboard = %q, want hello", text)
	}
}

func TestTextFieldNoContextMenu(t *testing.T) {
	controller := widgets.NewTextEditingController("hello")
	t.Cleanup(controller.Dispose)
	ts := pumpField(t, NewTextField(TextFieldOpts{Controller: controller, NoContextMenu: true}))
	rightClick(ts, ts.Center(findField()))
	if ts.Exists(gluttertest.ByText("Select all")) {
		t.Error("a context menu was built")
	}
}

func TestTextFieldCustomContextMenu(t *testing.T) {
	ts := pumpField(t, NewTextField(TextFieldOpts{ContextMenuBuilder: func(bc widgets.BuildContext, state *widgets.EditableTextState) widgets.Widget {
		return widgets.NewText("custom menu")
	}}))
	rightClick(ts, ts.Center(findField()))
	if !ts.Exists(gluttertest.ByText("custom menu")) {
		t.Error("the custom menu is not shown")
	}
}

func TestTextFieldOnTap(t *testing.T) {
	taps := 0
	ts := pumpField(t, NewTextField(TextFieldOpts{OnTap: func() { taps++ }}))
	ts.Tap(findField())
	ts.Pump()
	if taps != 1 {
		t.Errorf("taps = %d, want 1", taps)
	}
}

func TestTextFieldIgnorePointers(t *testing.T) {
	taps := 0
	ts := pumpField(t, NewTextField(TextFieldOpts{OnTap: func() { taps++ }, IgnorePointers: foundation.Some(true)}))
	ts.Tap(findField())
	ts.Pump()
	if taps != 0 {
		t.Errorf("taps = %d, want 0", taps)
	}
}

func TestTextFieldDecorationAndStyleDefaults(t *testing.T) {
	tests := []struct {
		name      string
		opts      TextFieldOpts
		wantEmpty bool
		wantSize  float64
	}{
		{"default decoration is Material 3 sized", TextFieldOpts{}, false, 0},
		{"no decoration shows the editable alone", TextFieldOpts{NoDecoration: true}, true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpField(t, NewTextField(tc.opts))
			if got := !ts.Exists(gluttertest.ByType[*InputDecorator]()); got != tc.wantEmpty {
				t.Errorf("no decorator = %v, want %v", got, tc.wantEmpty)
			}
		})
	}
}

func TestTextFieldMultilineGrowsWithMinLines(t *testing.T) {
	controller := widgets.NewTextEditingController()
	t.Cleanup(controller.Dispose)
	ts := pumpField(t, NewTextField(TextFieldOpts{Controller: controller, MinLines: foundation.Some(2), MaxLines: foundation.Some(4)}))
	h1 := ts.RenderObject(findField()).(rendering.Box).Size().Height
	ts.EnterText(findField(), "1\n2\n3\n4\n5\n6")
	h2 := ts.RenderObject(findField()).(rendering.Box).Size().Height
	if !(h2 > h1) {
		t.Errorf("height with six lines %v is not more than with none %v", h2, h1)
	}
	ts.EnterText(findField(), "1\n2\n3\n4\n5\n6\n7\n8")
	if h3 := ts.RenderObject(findField()).(rendering.Box).Size().Height; h3 != h2 {
		t.Errorf("height with eight lines %v differs from the height with six %v: maxLines is 4", h3, h2)
	}
}

func TestTextFieldObscureText(t *testing.T) {
	ts := pumpField(t, NewTextField(TextFieldOpts{ObscureText: true, ObscuringCharacter: "*"}))
	editable := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText)
	if !editable.ObscureText() || editable.ObscuringCharacter() != "*" {
		t.Errorf("obscure = %v, character = %q", editable.ObscureText(), editable.ObscuringCharacter())
	}
}

func TestTextFieldReadOnlyShowsNoCursor(t *testing.T) {
	ts := pumpField(t, NewTextField(TextFieldOpts{ReadOnly: true}))
	editable := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText)
	if editable.ShowCursor() || !editable.ReadOnly() {
		t.Errorf("showCursor = %v, readOnly = %v", editable.ShowCursor(), editable.ReadOnly())
	}
}

func TestTextFieldMouseCursor(t *testing.T) {
	tests := []struct {
		name string
		opts TextFieldOpts
		want services.MouseCursor
	}{
		{"text cursor by default", TextFieldOpts{}, services.SystemMouseCursors.Text},
		{"basic when disabled", TextFieldOpts{Enabled: foundation.Some(false)}, services.SystemMouseCursors.Basic},
		{"given cursor", TextFieldOpts{MouseCursor: services.SystemMouseCursors.Click}, services.SystemMouseCursors.Click},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpField(t, NewTextField(tc.opts))
			region := ts.Widget(gluttertest.Ancestor(gluttertest.ByType[*widgets.EditableText](), gluttertest.ByType[*widgets.MouseRegion]())).(*widgets.MouseRegion)
			if got := region.Cursor(); got != tc.want {
				t.Errorf("cursor = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextFieldRebuildKeepsTheTextWhenTheControllerIsLocal(t *testing.T) {
	build := func(f widgets.Widget) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: f})})
	}
	ts := gluttertest.New(t)
	ts.PumpWidget(build(NewTextField()))
	ts.EnterText(findField(), "kept")
	ts.PumpWidget(build(NewTextField(TextFieldOpts{Decoration: &InputDecoration{LabelText: str("x")}})))
	editable := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText)
	if editable.Controller().Text() != "kept" {
		t.Errorf("text = %q after rebuild", editable.Controller().Text())
	}
}

func TestTextFieldSwitchingToAGivenControllerAndBack(t *testing.T) {
	given := widgets.NewTextEditingController("given")
	t.Cleanup(given.Dispose)
	build := func(c *widgets.TextEditingController) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTextField(TextFieldOpts{Controller: c})})})
	}
	ts := gluttertest.New(t)
	ts.PumpWidget(build(nil))
	ts.EnterText(findField(), "local")
	ts.PumpWidget(build(given))
	if got := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).Controller(); got != given {
		t.Error("the field does not use the given controller")
	}
	ts.PumpWidget(build(nil))
	if got := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).Controller().Text(); got != "given" {
		t.Errorf("text after dropping the controller = %q, want the text of the old controller", got)
	}
}

func TestTextFieldDiagnostics(t *testing.T) {
	skipUnlessDebug(t)
	controller := widgets.NewTextEditingController()
	t.Cleanup(controller.Dispose)
	w := NewTextField(TextFieldOpts{Controller: controller, MaxLength: foundation.Some(8), ObscureText: true, CursorColor: foundation.Some(ui.NewColor(0xFF112233))})
	got := foundation.DiagnosticableString(w)
	for _, want := range []string{"controller", "obscureText: true", "maxLength: 8", "cursorColor"} {
		if !containsStr(got, want) {
			t.Errorf("description lacks %q:\n%s", want, got)
		}
	}
}

func TestTextFieldAccessors(t *testing.T) {
	w := NewTextField(TextFieldOpts{MaxLines: foundation.Some(rendering.MaxLinesUnlimited), MinLines: foundation.Some(2), ReadOnly: true})
	if w.MaxLines().IsSome() || w.MinLines() != foundation.Some(2) || !w.ReadOnly() || w.ObscureText() || w.Decoration() == nil {
		t.Errorf("maxLines %v, minLines %v, readOnly %v, obscure %v", w.MaxLines(), w.MinLines(), w.ReadOnly(), w.ObscureText())
	}
	if NewTextField(TextFieldOpts{NoDecoration: true}).Decoration() != nil {
		t.Error("NoDecoration still has a decoration")
	}
	if got := NewTextField().MaxLines(); got != foundation.Some(1) {
		t.Errorf("default maxLines = %v", got)
	}
}

func TestTextFieldErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"obscuring character", func() { NewTextField(TextFieldOpts{ObscuringCharacter: "ab"}) }, "TextFieldObscuringCharacterInvalid"},
		{"max lines", func() { NewTextField(TextFieldOpts{MaxLines: foundation.Some(-3)}) }, "TextFieldMaxLinesInvalid"},
		{"min lines", func() { NewTextField(TextFieldOpts{MinLines: foundation.Some(0)}) }, "TextFieldMinLinesInvalid"},
		{"min lines above max lines", func() {
			NewTextField(TextFieldOpts{MinLines: foundation.Some(3), MaxLines: foundation.Some(2)})
		}, "TextFieldMinLinesGreaterThanMaxLines"},
		{"expands with lines", func() {
			NewTextField(TextFieldOpts{Expands: true, MaxLines: foundation.Some(2)})
		}, "TextFieldExpandsWithLines"},
		{"obscured multiline", func() {
			NewTextField(TextFieldOpts{ObscureText: true, MaxLines: foundation.Some(2)})
		}, "TextFieldObscuredMultiline"},
		{"max length", func() { NewTextField(TextFieldOpts{MaxLength: foundation.Some(0)}) }, "TextFieldMaxLengthInvalid"},
		{"newline action with the text keyboard", func() {
			NewTextField(TextFieldOpts{TextInputAction: services.TextInputActionNewline, MaxLines: foundation.Some(3), KeyboardType: foundation.Some(services.TextInputTypeText)})
		}, "TextFieldNewlineActionWithTextKeyboard"},
		{"style that does not inherit", func() {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTextField(TextFieldOpts{
				Style: painting.NewTextStyle(painting.TextStyleOpts{Inherit: foundation.Some(false)})})})}))
			takeAndPanic(ts)
		}, "TextFieldStyleNotInheriting"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.run), tc.wantRoute)
		})
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

var _ = time.Second
