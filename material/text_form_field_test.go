// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/widgets"
)

func requiredValidator(v foundation.Opt[string]) foundation.Opt[string] {
	if v.OrElse("") == "" {
		return str("required")
	}
	return foundation.None[string]()
}

// pumpForm builds the form with the field in a Material app.
func pumpForm(t *testing.T, key *widgets.GlobalKey[*widgets.FormState], field widgets.Widget, opts ...widgets.FormOpts) *gluttertest.Tester {
	t.Helper()
	o := widgets.FormOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Key = key
	ts := gluttertest.New(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
		Body: widgets.NewCenter(widgets.NewSizedBox(widgets.NewForm(field, o), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))})}))
	return ts
}

func formState(key *widgets.GlobalKey[*widgets.FormState]) *widgets.FormState {
	s, _ := key.CurrentState()
	return s
}

func findFormField() gluttertest.Finder { return gluttertest.ByType[*TextFormField]() }

func TestTextFormFieldValidation(t *testing.T) {
	key := widgets.NewGlobalKey[*widgets.FormState]()
	ts := pumpForm(t, key, NewTextFormField(TextFormFieldOpts{
		TextFieldOpts: TextFieldOpts{Decoration: &InputDecoration{LabelText: str("Name")}},
		Validator:     requiredValidator,
	}))
	if ts.Exists(gluttertest.ByText("required")) {
		t.Fatal("the error is shown before the form validates")
	}
	if formState(key).Validate() {
		t.Error("an empty required field is valid")
	}
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("required")) {
		t.Fatal("the error is not shown")
	}
	ts.EnterText(findFormField(), "x")
	if !formState(key).Validate() {
		t.Error("a field with text is not valid")
	}
	ts.Pump()
	if ts.Exists(gluttertest.ByText("required")) {
		t.Error("the error is still shown")
	}
}

func TestTextFormFieldAutovalidate(t *testing.T) {
	tests := []struct {
		name string
		mode widgets.AutovalidateMode
		want bool
	}{
		{"disabled never validates", widgets.AutovalidateModeDisabled, false},
		{"on user interaction validates after typing", widgets.AutovalidateModeOnUserInteraction, true},
		{"always validates after typing", widgets.AutovalidateModeAlways, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := widgets.NewGlobalKey[*widgets.FormState]()
			ts := pumpForm(t, key, NewTextFormField(TextFormFieldOpts{
				Validator: func(v foundation.Opt[string]) foundation.Opt[string] {
					if len(v.OrElse("")) < 3 {
						return str("too short")
					}
					return foundation.None[string]()
				},
				AutovalidateMode: tc.mode,
			}))
			ts.EnterText(findFormField(), "ab")
			ts.Pump()
			if got := ts.Exists(gluttertest.ByText("too short")); got != tc.want {
				t.Errorf("error shown = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextFormFieldSaveAndReset(t *testing.T) {
	key := widgets.NewGlobalKey[*widgets.FormState]()
	var saved []string
	var changed []string
	ts := pumpForm(t, key, NewTextFormField(TextFormFieldOpts{
		InitialValue: str("start"),
		OnSaved:      func(v foundation.Opt[string]) { saved = append(saved, v.OrElse("<none>")) },
		TextFieldOpts: TextFieldOpts{
			OnChanged: func(s string) { changed = append(changed, s) },
		},
	}))
	editable := func() *widgets.TextEditingController {
		return ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).Controller()
	}
	if editable().Text() != "start" {
		t.Fatalf("initial text = %q", editable().Text())
	}
	ts.EnterText(findFormField(), "changed")
	formState(key).Save()
	if len(saved) != 1 || saved[0] != "changed" {
		t.Errorf("saved = %q, want [changed]", saved)
	}
	changed = nil
	formState(key).Reset()
	ts.Pump()
	if editable().Text() != "start" {
		t.Errorf("text after reset = %q, want start", editable().Text())
	}
	if len(changed) == 0 || changed[len(changed)-1] != "start" {
		t.Errorf("onChanged calls on reset = %q, want the initial text last", changed)
	}
	formState(key).Save()
	if saved[len(saved)-1] != "start" {
		t.Errorf("saved after reset = %q", saved)
	}
}

func TestTextFormFieldFollowsTheController(t *testing.T) {
	key := widgets.NewGlobalKey[*widgets.FormState]()
	controller := widgets.NewTextEditingController("one")
	t.Cleanup(controller.Dispose)
	var saved string
	ts := pumpForm(t, key, NewTextFormField(TextFormFieldOpts{
		TextFieldOpts: TextFieldOpts{Controller: controller},
		OnSaved:       func(v foundation.Opt[string]) { saved = v.OrElse("") },
	}))
	controller.SetText("two")
	ts.Pump()
	formState(key).Save()
	if saved != "two" {
		t.Errorf("saved = %q, want the text that the controller got", saved)
	}
}

func TestTextFormFieldForceErrorTextAndErrorBuilder(t *testing.T) {
	tests := []struct {
		name  string
		opts  TextFormFieldOpts
		wants string
	}{
		{"forced error text", TextFormFieldOpts{ForceErrorText: str("forced")}, "forced"},
		{"error builder", TextFormFieldOpts{ForceErrorText: str("forced"), ErrorBuilder: func(bc widgets.BuildContext, text string) widgets.Widget {
			return widgets.NewText("built: " + text)
		}}, "built: forced"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpForm(t, widgets.NewGlobalKey[*widgets.FormState](), NewTextFormField(tc.opts))
			if !ts.Exists(gluttertest.ByText(tc.wants)) {
				t.Errorf("no %q", tc.wants)
			}
		})
	}
}

func TestTextFormFieldDisabledDoesNotValidateOnItsOwn(t *testing.T) {
	key := widgets.NewGlobalKey[*widgets.FormState]()
	ts := pumpForm(t, key, NewTextFormField(TextFormFieldOpts{
		TextFieldOpts: TextFieldOpts{Enabled: foundation.Some(false)},
		Validator:     requiredValidator,
	}), widgets.FormOpts{AutovalidateMode: widgets.AutovalidateModeAlways})
	ts.Pump()
	if ts.Exists(gluttertest.ByText("required")) {
		t.Error("a disabled field validated on its own")
	}
}

func TestTextFormFieldAccessors(t *testing.T) {
	c := widgets.NewTextEditingController("given")
	t.Cleanup(c.Dispose)
	if got := NewTextFormField(TextFormFieldOpts{TextFieldOpts: TextFieldOpts{Controller: c}}); got.Controller() != c || got.InitialValue() != "given" {
		t.Errorf("controller %v, initialValue %q", got.Controller(), got.InitialValue())
	}
	if got := NewTextFormField(TextFormFieldOpts{InitialValue: str("init")}); got.InitialValue() != "init" {
		t.Errorf("initialValue %q", got.InitialValue())
	}
	if got := NewTextFormField(); got.InitialValue() != "" {
		t.Errorf("default initialValue %q", got.InitialValue())
	}
}

func TestTextFormFieldErrors(t *testing.T) {
	skipUnlessDebug(t)
	c := widgets.NewTextEditingController()
	t.Cleanup(c.Dispose)
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"initial value and controller", func() {
			NewTextFormField(TextFormFieldOpts{InitialValue: str("x"), TextFieldOpts: TextFieldOpts{Controller: c}})
		}, "TextFormFieldInitialValueWithController"},
		{"error builder and error text", func() {
			NewTextFormField(TextFormFieldOpts{ErrorBuilder: func(widgets.BuildContext, string) widgets.Widget { return nil },
				TextFieldOpts: TextFieldOpts{Decoration: &InputDecoration{ErrorText: str("e")}}})
		}, "TextFormFieldErrorBuilderWithErrorText"},
		{"the checks of the text field", func() {
			NewTextFormField(TextFormFieldOpts{TextFieldOpts: TextFieldOpts{MaxLength: foundation.Some(-4)}})
		}, "TextFieldMaxLengthInvalid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.run), tc.wantRoute)
		})
	}
}
