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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func formFieldItems(values ...string) []*DropdownMenuItem[string] {
	var out []*DropdownMenuItem[string]
	for _, v := range values {
		out = append(out, NewDropdownMenuItem(widgets.NewText("item "+v), DropdownMenuItemOpts[string]{Value: foundation.Some(v)}))
	}
	return out
}

func pumpDropdownFormField(t *testing.T, key *widgets.GlobalKey[*widgets.FormState], field widgets.Widget, opts ...widgets.FormOpts) *gluttertest.Tester {
	t.Helper()
	o := widgets.FormOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Key = key
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 600, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
		Body: widgets.NewAlign(widgets.NewSizedBox(widgets.NewForm(field, o), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))})}))
	return ts
}

func TestDropdownButtonFormFieldSelectsAndValidates(t *testing.T) {
	key := widgets.NewGlobalKey[*widgets.FormState]()
	var changed, saved []string
	ts := pumpDropdownFormField(t, key, NewDropdownButtonFormField(formFieldItems("a", "b", "c"),
		func(v foundation.Opt[string]) { changed = append(changed, v.OrElse("<none>")) },
		DropdownButtonFormFieldOpts[string]{
			Decoration: &InputDecoration{LabelText: str("Letter")},
			Validator: func(v foundation.Opt[string]) foundation.Opt[string] {
				if v.IsNone() {
					return str("pick one")
				}
				return foundation.None[string]()
			},
			OnSaved: func(v foundation.Opt[string]) { saved = append(saved, v.OrElse("<none>")) },
		}))
	if !ts.Exists(gluttertest.ByText("Letter")) {
		t.Fatal("no label")
	}
	if formState(key).Validate() {
		t.Fatal("a field with no value is valid")
	}
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("pick one")) {
		t.Fatal("the error is not shown")
	}
	ts.Tap(gluttertest.ByType[*DropdownButton[string]]())
	ts.PumpAndSettle()
	ts.Tap(gluttertest.Last(gluttertest.ByText("item b")))
	ts.PumpAndSettle()
	if len(changed) != 1 || changed[0] != "b" {
		t.Errorf("onChanged calls = %q, want [b]", changed)
	}
	if !formState(key).Validate() {
		t.Error("the field is not valid with a value")
	}
	ts.Pump()
	if ts.Exists(gluttertest.ByText("pick one")) {
		t.Error("the error is still shown")
	}
	formState(key).Save()
	if len(saved) != 1 || saved[0] != "b" {
		t.Errorf("saved = %q, want [b]", saved)
	}
	changed = nil
	formState(key).Reset()
	ts.Pump()
	if len(changed) != 1 || changed[0] != "<none>" {
		t.Errorf("onChanged calls on reset = %q, want [<none>]", changed)
	}
}

func TestDropdownButtonFormFieldInitialValueAndHint(t *testing.T) {
	tests := []struct {
		name string
		opts DropdownButtonFormFieldOpts[string]
		want string
	}{
		{"initial value", DropdownButtonFormFieldOpts[string]{InitialValue: str("c")}, "item c"},
		{"hint", DropdownButtonFormFieldOpts[string]{Hint: widgets.NewText("choose")}, "choose"},
		{"hint from the decoration", DropdownButtonFormFieldOpts[string]{Decoration: &InputDecoration{HintText: str("from decoration")}}, "from decoration"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpDropdownFormField(t, widgets.NewGlobalKey[*widgets.FormState](),
				NewDropdownButtonFormField(formFieldItems("a", "b", "c"), func(foundation.Opt[string]) {}, tc.opts))
			if !ts.Exists(gluttertest.ByText(tc.want)) {
				t.Errorf("no %q", tc.want)
			}
		})
	}
}

func TestDropdownButtonFormFieldDisabledWithoutOnChanged(t *testing.T) {
	ts := pumpDropdownFormField(t, widgets.NewGlobalKey[*widgets.FormState](),
		NewDropdownButtonFormField(formFieldItems("a", "b"), nil, DropdownButtonFormFieldOpts[string]{InitialValue: str("a")}))
	ts.Tap(gluttertest.ByType[*DropdownButton[string]]())
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("item b")) {
		t.Error("a disabled field opened its menu")
	}
}

func TestDropdownButtonFormFieldErrorBuilderAndForcedError(t *testing.T) {
	ts := pumpDropdownFormField(t, widgets.NewGlobalKey[*widgets.FormState](),
		NewDropdownButtonFormField(formFieldItems("a"), func(foundation.Opt[string]) {}, DropdownButtonFormFieldOpts[string]{
			ForceErrorText: str("nope"),
			ErrorBuilder: func(bc widgets.BuildContext, text string) widgets.Widget {
				return widgets.NewText("built " + text)
			},
		}))
	if !ts.Exists(gluttertest.ByText("built nope")) {
		t.Error("the error widget is not shown")
	}
}

func TestDropdownButtonFormFieldInputDecoratorStates(t *testing.T) {
	ts := pumpDropdownFormField(t, widgets.NewGlobalKey[*widgets.FormState](),
		NewDropdownButtonFormField(formFieldItems("a"), func(foundation.Opt[string]) {}))
	dec := ts.Widget(gluttertest.ByType[*InputDecorator]()).(*InputDecorator)
	if !dec.IsEmpty() || dec.IsFocused() {
		t.Errorf("isEmpty %v, isFocused %v: an empty field with no hint is empty and not focused", dec.IsEmpty(), dec.IsFocused())
	}
	ts.Hover(ts.Center(gluttertest.ByType[*DropdownButton[string]]()))
	ts.Pump()
}

func TestDropdownButtonFormFieldErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"value not in the items", func() {
			NewDropdownButtonFormField(formFieldItems("a", "b"), nil, DropdownButtonFormFieldOpts[string]{InitialValue: str("z")})
		}, "DropdownButtonFormFieldValueNotUnique"},
		{"error builder and error text", func() {
			NewDropdownButtonFormField(formFieldItems("a"), nil, DropdownButtonFormFieldOpts[string]{
				ErrorBuilder: func(widgets.BuildContext, string) widgets.Widget { return nil },
				Decoration:   &InputDecoration{ErrorText: str("e")}})
		}, "DropdownButtonFormFieldErrorBuilderWithErrorText"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.run), tc.wantRoute)
		})
	}
}
