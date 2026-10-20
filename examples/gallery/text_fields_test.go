package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// openTextFieldsPage opens the text fields page through the navigation rail.
func openTextFieldsPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("Text fields")))
	ts.PumpAndSettle()
	return ts
}

// TestTextFieldsGolden draws the top of the text fields page in the light theme.
func TestTextFieldsGolden(t *testing.T) {
	ts := openTextFieldsPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-text-fields")
}

// TestTextFieldsDarkGolden draws the page in the dark theme.
func TestTextFieldsDarkGolden(t *testing.T) {
	ts := openTextFieldsPage(t)
	ts.Tap(gluttertest.ByIcon(m.Icons.DarkMode()))
	ts.PumpAndSettle()
	ts.MatchesGolden("gallery-text-fields-dark")
}

func TestTextFieldsFormValidation(t *testing.T) {
	ts := openTextFieldsPage(t)
	if ts.Exists(gluttertest.ByText("Enter a valid email")) {
		t.Fatal("the error is shown before the form validates")
	}
	ts.Tap(gluttertest.ByText("Validate"))
	ts.PumpAndSettle()
	for _, text := range []string{"Enter a valid email", "Use at least 8 characters", "The form has errors"} {
		if !ts.Exists(gluttertest.ByText(text)) {
			t.Errorf("the text %q is not shown", text)
		}
	}
	emailField := gluttertest.Descendant(gluttertest.ByType[*m.TextFormField](), gluttertest.ByText("Email"))
	ts.EnterText(gluttertest.Ancestor(emailField, gluttertest.ByType[*m.TextFormField]()), "me@example.com")
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Enter a valid email")) {
		t.Error("the error of the email is still shown after a valid one was typed")
	}
	ts.Tap(gluttertest.ByText("Reset"))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Use at least 8 characters")) || !ts.Exists(gluttertest.ByText("Not submitted")) {
		t.Error("the form was not reset")
	}
}

// scrollPageDown scrolls the page up by 400 logical pixels.
func scrollPageDown(ts *gluttertest.Tester) {
	ts.Drag(gluttertest.First(gluttertest.ByType[*m.ListView]()), m.Offset{Dx: 0, Dy: -400})
	ts.PumpAndSettle()
}

func TestTextFieldsPasswordToggle(t *testing.T) {
	ts := openTextFieldsPage(t)
	scrollPageDown(ts)
	ts.Tap(gluttertest.ByIcon(m.Icons.Visibility()))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByIcon(m.Icons.VisibilityOff())) {
		t.Error("the button did not change")
	}
	ts.Tap(gluttertest.ByIcon(m.Icons.VisibilityOff()))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByIcon(m.Icons.Visibility())) {
		t.Error("the button did not change back")
	}
}

func TestTextFieldsDropdownMenu(t *testing.T) {
	ts := openTextFieldsPage(t)
	scrollPageDown(ts)
	scrollPageDown(ts)
	ts.Tap(gluttertest.ByType[*m.DropdownMenu[string]]())
	ts.PumpAndSettle()
	ts.Tap(gluttertest.Last(gluttertest.ByText("Green")))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Selected color: green")) {
		t.Error("the color was not selected")
	}
}

// TestTextFieldsDevVariables checks that GLUTTER_DEV_TEXT_FIELDS focuses the
// outlined field and validates the form, in a debug build.
func TestTextFieldsDevVariables(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantFocus   bool
		wantInvalid bool
	}{
		{"none", "", false, false},
		{"focus", "focus", true, false},
		{"validate", "validate", false, true},
		{"both", "focus, validate", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !foundation.DebugMode {
				t.Skip("the dev variables work only in debug builds")
			}
			t.Setenv("GLUTTER_DEV_TEXT_FIELDS", tc.value)
			ts := openTextFieldsPage(t)
			ts.Pump(time.Second)
			search := gluttertest.Ancestor(gluttertest.ByText("Search"), gluttertest.ByType[*m.TextField]())
			focused := ts.Widget(gluttertest.Descendant(search, gluttertest.ByType[*m.EditableText]())).(*m.EditableText).FocusNode().HasFocus()
			if focused != tc.wantFocus {
				t.Errorf("search field focused = %v, want %v", focused, tc.wantFocus)
			}
			if got := ts.Exists(gluttertest.ByText("Enter a valid email")); got != tc.wantInvalid {
				t.Errorf("email error shown = %v, want %v", got, tc.wantInvalid)
			}
		})
	}
}
