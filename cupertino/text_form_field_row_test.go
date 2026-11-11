package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoTextFormFieldRow, ported from cupertino/text_form_field_row_test.dart (step 34.8). The cases that
// only look at the parameters that the row gives to its CupertinoTextField (textAlign, scrollPhysics, cursor attributes and
// the like) are covered by text_field_test.go, which has no getter for them either; the one on the context menu
// of a read-only field needs the platform's toolbar, which the tests of the toolbars cover.

func rowEditable(ts *gluttertest.Tester) *widgets.EditableText {
	return ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText)
}

func pumpTextRow(t *testing.T, row widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := dialogTester(t, ui.Size{})
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(row)}))
	return ts
}

func pumpTextRowInForm(t *testing.T, row widgets.Widget) (*gluttertest.Tester, *widgets.GlobalKey[*widgets.FormState]) {
	t.Helper()
	key := widgets.NewGlobalKey[*widgets.FormState]()
	return pumpTextRow(t, widgets.NewForm(row, widgets.FormOpts{Key: key})), key
}

func str(s string) foundation.Opt[string] { return foundation.Some(s) }

func TestCupertinoTextFormFieldRow(t *testing.T) {
	t.Run("onFieldSubmitted is called", func(t *testing.T) {
		called := false
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{OnFieldSubmitted: func(string) { called = true }}))
		ts.ShowKeyboard(gluttertest.ByType[*cupertino.CupertinoTextField]()).PerformAction(services.TextInputActionDone)
		ts.Pump()
		if !called {
			t.Error("onFieldSubmitted was not called")
		}
	})
	t.Run("onChanged is called", func(t *testing.T) {
		var value string
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{OnChanged: func(v string) { value = v }}))
		ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "Soup")
		ts.Pump()
		if value != "Soup" {
			t.Errorf("onChanged got %q", value)
		}
	})
	for _, enabled := range []foundation.Opt[bool]{foundation.None[bool](), foundation.Some(true)} {
		t.Run("autovalidateMode always validates at the start and on every change", func(t *testing.T) {
			calls := 0
			ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{
				Enabled: enabled, AutovalidateMode: widgets.AutovalidateModeAlways,
				Validator: func(foundation.Opt[string]) foundation.Opt[string] { calls++; return foundation.None[string]() }}))
			if calls != 1 {
				t.Fatalf("the validator was called %d times at the start", calls)
			}
			ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "a")
			ts.Pump()
			if calls != 2 {
				t.Errorf("the validator was called %d times after a change", calls)
			}
		})
	}
	t.Run("a field that is not enabled is not validated by itself", func(t *testing.T) {
		calls := 0
		pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{
			Enabled: foundation.Some(false), AutovalidateMode: widgets.AutovalidateModeAlways,
			Validator: func(foundation.Opt[string]) foundation.Opt[string] { calls++; return foundation.None[string]() }}))
		if calls != 0 {
			t.Errorf("the validator was called %d times", calls)
		}
	})
	t.Run("onTap is called upon a tap", func(t *testing.T) {
		tapped := false
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{OnTap: func() { tapped = true }}))
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoTextField]())
		ts.Pump()
		if !tapped {
			t.Error("onTap was not called")
		}
	})
	t.Run("a read-only field takes no keyboard", func(t *testing.T) {
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{InitialValue: str("readonly"), ReadOnly: true}))
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoTextField]())
		ts.Pump()
		if !rowEditable(ts).ReadOnly() {
			t.Error("the text field is not read-only")
		}
		if rowEditable(ts).ShowCursor() {
			t.Error("a read-only field shows its cursor")
		}
	})
	t.Run("reset puts the initial value back", func(t *testing.T) {
		ts, key := pumpTextRowInForm(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{InitialValue: str("initialValue")}))
		ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextFormFieldRow](), "changedValue")
		state, _ := key.CurrentState()
		state.Reset()
		ts.Pump()
		if got := rowEditable(ts).Controller().Text(); got != "initialValue" {
			t.Errorf("the text is %q", got)
		}
	})
	t.Run("a change of the controller changes the value of the field", func(t *testing.T) {
		c := widgets.NewTextEditingController("")
		defer c.Dispose()
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{
			Controller: c, AutovalidateMode: widgets.AutovalidateModeOnUserInteraction,
			Validator: func(foundation.Opt[string]) foundation.Opt[string] { return str("Error") }}))
		if ts.Exists(gluttertest.ByType[*widgets.Text]()) {
			t.Fatal("an error is shown before the user touched the field")
		}
		c.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "Value"}))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByType[*widgets.Text]()) != 1 || ts.Widget(gluttertest.ByType[*widgets.Text]()).(*widgets.Text).Data() != str("Error") {
			t.Error("the error is not shown")
		}
	})
	t.Run("the value that onChanged gets is the one of the form field", func(t *testing.T) {
		called := false
		ts, _ := pumpTextRowInForm(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{OnChanged: func(v string) {
			called = true
			if v != "Soup" {
				t.Errorf("onChanged got %q", v)
			}
		}}))
		ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "Soup")
		if !called {
			t.Error("onChanged was not called")
		}
	})
	t.Run("autovalidateMode always shows the error from the start", func(t *testing.T) {
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{
			InitialValue: str("Value"), AutovalidateMode: widgets.AutovalidateModeAlways,
			Validator: func(foundation.Opt[string]) foundation.Opt[string] { return str("Error") }}))
		if ts.Count(gluttertest.ByType[*widgets.Text]()) != 1 || ts.Widget(gluttertest.ByType[*widgets.Text]()).(*widgets.Text).Data() != str("Error") {
			t.Error("the error is not shown")
		}
	})
	t.Run("shows the error of a form that validates", func(t *testing.T) {
		ts, key := pumpTextRowInForm(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{
			Validator: func(v foundation.Opt[string]) foundation.Opt[string] {
				if v.OrElse("") == "" {
					return str("required")
				}
				return foundation.None[string]()
			}}))
		state, _ := key.CurrentState()
		if state.Validate() {
			t.Error("an empty required field is valid")
		}
		ts.Pump()
		if !ts.Exists(gluttertest.ByText("required")) {
			t.Error("the error is not shown")
		}
		ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "x")
		if !state.Validate() {
			t.Error("a field with text is not valid")
		}
		ts.Pump()
		if ts.Exists(gluttertest.ByText("required")) {
			t.Error("the error is still shown")
		}
	})
	t.Run("shows the prefix", func(t *testing.T) {
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{Prefix: widgets.NewText("Prefix")}))
		if !ts.Exists(gluttertest.ByText("Prefix")) {
			t.Error("the prefix is missing")
		}
	})
	t.Run("saves the value", func(t *testing.T) {
		var saved []string
		ts, key := pumpTextRowInForm(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{
			OnSaved: func(v foundation.Opt[string]) { saved = append(saved, v.OrElse("<none>")) }}))
		ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "kept")
		state, _ := key.CurrentState()
		state.Save()
		if len(saved) != 1 || saved[0] != "kept" {
			t.Errorf("saved %q", saved)
		}
	})
	t.Run("onChanged is called when the form is reset", func(t *testing.T) {
		value := "initialValue"
		ts, key := pumpTextRowInForm(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{
			InitialValue: str(value), OnChanged: func(v string) { value = v }}))
		ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "changedValue")
		if value != "changedValue" {
			t.Fatalf("value is %q", value)
		}
		state, _ := key.CurrentState()
		state.Reset()
		ts.Pump()
		if value != "initialValue" || rowEditable(ts).Controller().Text() != "initialValue" {
			t.Errorf("after the reset value is %q and the text is %q", value, rowEditable(ts).Controller().Text())
		}
	})
	t.Run("the controller of the row is the one of the text", func(t *testing.T) {
		c := widgets.NewTextEditingController("from the controller")
		defer c.Dispose()
		ts := pumpTextRow(t, cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{Controller: c}))
		if rowEditable(ts).Controller() != c || c.Text() != "from the controller" {
			t.Error("the text field does not use the controller")
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := pumpTextRow(t, widgets.NewSizedBoxShrink(cupertino.NewCupertinoTextFormFieldRow()))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoTextFormFieldRow]()); got != (ui.Size{}) {
			t.Errorf("the size is %v", got)
		}
	})
	t.Run("an initial value and a controller are an error", func(t *testing.T) {
		skipUnlessDebug(t)
		c := widgets.NewTextEditingController("")
		defer c.Dispose()
		err := recoverError(func() {
			cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{Controller: c, InitialValue: str("x")})
		})
		checkErrorTag(t, err, "NewCupertinoTextFormFieldRow.CupertinoTextFormFieldRowInitialValueWithController")
	})
}
