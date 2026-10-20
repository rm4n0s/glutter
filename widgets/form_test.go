package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/form_test.dart. The
// fields are plain FormFields whose builder shows the value and the error.

// stringField builds a FormField[string] that records its state in *out.
func stringField(out **FormFieldState[string], opts FormFieldOpts[string]) *FormField[string] {
	return NewFormField(func(f *FormFieldState[string]) Widget {
		*out = f
		text := f.Value().OrElse("")
		if f.HasError() {
			text += "|" + f.ErrorText().Value()
		}
		return ltrText(text)
	}, opts)
}

func requiredValidator(value foundation.Opt[string]) foundation.Opt[string] {
	if value.OrElse("") == "" {
		return foundation.Some("required")
	}
	return foundation.None[string]()
}

func TestFormFieldSaveAndResetCallHandlers(t *testing.T) {
	ts := gluttertest.New(t)
	var saved []foundation.Opt[string]
	var resets int
	var field *FormFieldState[string]
	formKey := NewGlobalKey[*FormState]()
	pumpContainer(ts, NewForm(stringField(&field, FormFieldOpts[string]{
		InitialValue: foundation.Some("initial"),
		OnSaved:      func(v foundation.Opt[string]) { saved = append(saved, v) },
		OnReset:      func() { resets++ },
	}), FormOpts{Key: formKey}))
	form, _ := formKey.CurrentState()
	form.Save()
	if len(saved) != 1 || saved[0].Value() != "initial" {
		t.Fatalf("saved = %v, want [initial]", saved)
	}
	field.DidChange(foundation.Some("changed"))
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("changed")) {
		t.Fatal("the field did not show the changed value")
	}
	form.Reset()
	ts.Pump()
	if resets != 1 || field.Value().Value() != "initial" || !ts.Exists(gluttertest.ByText("initial")) {
		t.Fatalf("after reset: resets=%d value=%v", resets, field.Value())
	}
}

func TestFormValidateReturnsFalseAndShowsTheErrors(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		want    bool
	}{
		{"invalid", "", false},
		{"valid", "x", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var field *FormFieldState[string]
			formKey := NewGlobalKey[*FormState]()
			pumpContainer(ts, NewForm(stringField(&field, FormFieldOpts[string]{
				InitialValue: foundation.Some(tc.initial), Validator: requiredValidator,
			}), FormOpts{Key: formKey}))
			form, _ := formKey.CurrentState()
			if got := form.Validate(); got != tc.want {
				t.Fatalf("Validate = %v, want %v", got, tc.want)
			}
			ts.Pump()
			if got := ts.Exists(gluttertest.ByText("|required")); got == tc.want {
				t.Fatalf("error shown = %v, want %v", got, !tc.want)
			}
			form.ClearError()
			ts.Pump()
			if field.HasError() {
				t.Fatal("ClearError left the error")
			}
		})
	}
}

func TestFormValidateGranularlyReturnsTheInvalidFields(t *testing.T) {
	ts := gluttertest.New(t)
	var a, b, c *FormFieldState[string]
	formKey := NewGlobalKey[*FormState]()
	pumpContainer(ts, NewForm(NewColumn([]Widget{
		stringField(&a, FormFieldOpts[string]{Validator: requiredValidator}),
		stringField(&b, FormFieldOpts[string]{InitialValue: foundation.Some("ok"), Validator: requiredValidator}),
		stringField(&c, FormFieldOpts[string]{Validator: requiredValidator}),
	}), FormOpts{Key: formKey}))
	form, _ := formKey.CurrentState()
	invalid := form.ValidateGranularly()
	if len(invalid) != 2 || invalid[0] != FormFieldStateBase(a) || invalid[1] != FormFieldStateBase(c) {
		t.Fatalf("invalid fields = %v, want [a c]", invalid)
	}
}

func TestFormOnChangedIsCalledWhenAFieldChanges(t *testing.T) {
	ts := gluttertest.New(t)
	var field *FormFieldState[string]
	var changed int
	pumpContainer(ts, NewForm(stringField(&field, FormFieldOpts[string]{}), FormOpts{OnChanged: func() { changed++ }}))
	field.DidChange(foundation.Some("a"))
	field.DidChange(foundation.Some("b"))
	if changed != 2 {
		t.Fatalf("OnChanged called %d times, want 2", changed)
	}
}

func TestFormAutovalidateModes(t *testing.T) {
	tests := []struct {
		name       string
		mode       AutovalidateMode
		wantBefore bool
		wantAfter  bool
	}{
		{"disabled", AutovalidateModeDisabled, false, false},
		{"always", AutovalidateModeAlways, true, true},
		{"onUserInteraction", AutovalidateModeOnUserInteraction, false, true},
		{"onUserInteractionIfError", AutovalidateModeOnUserInteractionIfError, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var field *FormFieldState[string]
			pumpContainer(ts, stringField(&field, FormFieldOpts[string]{Validator: requiredValidator, AutovalidateMode: tc.mode}))
			ts.Pump()
			if got := field.HasError(); got != tc.wantBefore {
				t.Fatalf("HasError before interaction = %v, want %v", got, tc.wantBefore)
			}
			field.DidChange(foundation.Some(""))
			ts.Pump()
			if got := field.HasError(); got != tc.wantAfter {
				t.Fatalf("HasError after interaction = %v, want %v", got, tc.wantAfter)
			}
		})
	}
}

func TestFormForceErrorTextSkipsTheValidator(t *testing.T) {
	ts := gluttertest.New(t)
	var field *FormFieldState[string]
	pumpContainer(ts, stringField(&field, FormFieldOpts[string]{
		InitialValue: foundation.Some("fine"), ForceErrorText: foundation.Some("forced"), Validator: requiredValidator,
	}))
	if !ts.Exists(gluttertest.ByText("fine|forced")) {
		t.Fatal("the forced error was not shown")
	}
	if field.Validate() {
		t.Fatal("Validate returned true with a forced error")
	}
	if field.ErrorText().Value() != "forced" {
		t.Fatalf("error = %q, want forced", field.ErrorText().Value())
	}
}

func TestFormFieldsAreUnregisteredWhenRemoved(t *testing.T) {
	ts := gluttertest.New(t)
	var a, b *FormFieldState[string]
	formKey := NewGlobalKey[*FormState]()
	pumpContainer(ts, NewForm(NewColumn([]Widget{
		stringField(&a, FormFieldOpts[string]{Key: foundation.NewValueKey("a")}),
		stringField(&b, FormFieldOpts[string]{Key: foundation.NewValueKey("b")}),
	}), FormOpts{Key: formKey}))
	form, _ := formKey.CurrentState()
	if n := len(form.Fields()); n != 2 {
		t.Fatalf("fields = %d, want 2", n)
	}
	pumpContainer(ts, NewForm(NewColumn([]Widget{
		stringField(&a, FormFieldOpts[string]{Key: foundation.NewValueKey("a")}),
	}), FormOpts{Key: formKey}))
	if n := len(form.Fields()); n != 1 {
		t.Fatalf("fields = %d after removal, want 1", n)
	}
}

func TestFormOfWithoutAFormPanics(t *testing.T) {
	ts := gluttertest.New(t)
	var bc BuildContext
	pumpContainer(ts, NewBuilder(func(c BuildContext) Widget { bc = c; return ltrText("x") }))
	requireTag(t, "FormNotFound", func() { FormOf(bc) })
	if FormMaybeOf(bc) != nil {
		t.Fatal("FormMaybeOf found a form")
	}
}

func TestFormFieldOnDidChangeIsCalledWithTheNewValue(t *testing.T) {
	ts := gluttertest.New(t)
	var got []foundation.Opt[string]
	var field *FormFieldState[string]
	pumpContainer(ts, stringField(&field, FormFieldOpts[string]{OnDidChange: func(v foundation.Opt[string]) { got = append(got, v) }}))
	field.DidChange(foundation.Some("one"))
	field.DidChange(foundation.None[string]())
	ts.Pump()
	if len(got) != 2 || got[0] != foundation.Some("one") || got[1].IsSome() {
		t.Errorf("OnDidChange calls = %v, want [one, none]", got)
	}
}
