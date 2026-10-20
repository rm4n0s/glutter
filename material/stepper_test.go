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
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/material/stepper_test.dart.

func threeSteps(states ...StepState) []Step {
	steps := make([]Step, 3)
	for i, name := range []string{"One", "Two", "Three"} {
		var state StepState
		if i < len(states) {
			state = states[i]
		}
		steps[i] = NewStep(widgets.NewText("Step "+name), widgets.NewSizedBox(widgets.NewText("Content "+name), widgets.SizedBoxOpts{Height: foundation.Some(100.0)}),
			StepOpts{State: state})
	}
	return steps
}

func pumpStepper(t *testing.T, theme *ThemeData, stepper widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: stepper})}))
	ts.PumpAndSettle()
	return ts
}

func TestStepperVertical(t *testing.T) {
	var tapped []int
	var continued, cancelled int
	ts := pumpStepper(t, standardTheme(true), NewStepper(threeSteps(), StepperOpts{
		OnStepTapped: func(i int) { tapped = append(tapped, i) }, OnStepContinue: func() { continued++ }, OnStepCancel: func() { cancelled++ }, CurrentStep: 1}))
	if !ts.Exists(gluttertest.ByText("Content Two")) {
		t.Fatal("the content of the current step is not shown")
	}
	ts.Tap(gluttertest.ByText("Step Three"))
	ts.Tap(gluttertest.ByText("Step One"))
	if len(tapped) != 2 || tapped[0] != 2 || tapped[1] != 0 {
		t.Errorf("tapped %v", tapped)
	}
	// Every step has controls: the ones of the current step are the second.
	ts.Tap(gluttertest.At(gluttertest.ByText("Continue"), 1))
	ts.Tap(gluttertest.At(gluttertest.ByText("Cancel"), 1))
	if continued != 1 || cancelled != 1 {
		t.Errorf("continue %d, cancel %d", continued, cancelled)
	}
}

func TestStepperUpperCaseControlsInMaterial2(t *testing.T) {
	ts := pumpStepper(t, standardTheme(false), NewStepper(threeSteps(), StepperOpts{OnStepContinue: func() {}, OnStepCancel: func() {}}))
	for _, text := range []string{"CONTINUE", "CANCEL"} {
		if !ts.Exists(gluttertest.ByText(text)) {
			t.Errorf("%q is not shown", text)
		}
	}
}

func TestStepperStates(t *testing.T) {
	ts := pumpStepper(t, standardTheme(true), NewStepper(append(threeSteps(StepStateComplete, StepStateError, StepStateDisabled), NewStep(widgets.NewText("Step Four"), widgets.NewSizedBox(nil), StepOpts{State: StepStateEditing}))))
	if !ts.Exists(gluttertest.ByIcon(Icons.Check())) || !ts.Exists(gluttertest.ByIcon(Icons.Edit())) || !ts.Exists(gluttertest.ByText("!")) || !ts.Exists(gluttertest.ByText("3")) {
		t.Error("the icons of the states are not shown")
	}
	// The title of a step in error is red, and of a disabled one is dimmed.
	colorOf := func(text string) ui.Color {
		w := gluttertest.WidgetAs[*widgets.AnimatedDefaultTextStyle](ts, gluttertest.First(gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*widgets.AnimatedDefaultTextStyle]())))
		return w.Style().Color.Value()
	}
	if colorOf("Step Two") != Colors.Red.Primary {
		t.Errorf("error title color %v", colorOf("Step Two"))
	}
	if colorOf("Step Three") != Colors.Black38 {
		t.Errorf("disabled title color %v", colorOf("Step Three"))
	}
	// A disabled step does not react to taps.
	var tapped []int
	ts = pumpStepper(t, standardTheme(true), NewStepper(threeSteps(StepStateIndexed, StepStateIndexed, StepStateDisabled), StepperOpts{OnStepTapped: func(i int) { tapped = append(tapped, i) }}))
	ts.Tap(gluttertest.ByText("Step Three"))
	if len(tapped) != 0 {
		t.Errorf("a disabled step was tapped: %v", tapped)
	}
}

func TestStepperStateChangeAnimates(t *testing.T) {
	state := StepStateIndexed
	var rebuild func()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewStatefulBuilder(
		func(bc widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
			rebuild = func() { set(func() {}) }
			return NewStepper(threeSteps(StepStateIndexed, state))
		})})}))
	ts.PumpAndSettle()
	state = StepStateError
	rebuild()
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	if !ts.Exists(gluttertest.ByType[*widgets.AnimatedCrossFade]()) {
		t.Error("the icon does not cross fade to the triangle")
	}
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("!")) {
		t.Error("no triangle after the animation")
	}
}

func TestStepperHorizontal(t *testing.T) {
	var tapped []int
	ts := pumpStepper(t, standardTheme(true), NewStepper(threeSteps(), StepperOpts{Type: StepperTypeHorizontal, CurrentStep: 1,
		OnStepTapped: func(i int) { tapped = append(tapped, i) }, OnStepContinue: func() {}, OnStepCancel: func() {}}))
	// All the headers are in one row and only the current content is visible.
	y := ts.Center(gluttertest.ByText("Step One")).Dy
	if ts.Center(gluttertest.ByText("Step Two")).Dy != y || ts.Center(gluttertest.ByText("Step Three")).Dy != y {
		t.Error("the headers are not in a row")
	}
	if !ts.Exists(gluttertest.ByText("Content Two")) || ts.Exists(gluttertest.ByText("Content One")) {
		t.Error("the content shown is not the one of the current step")
	}
	ts.Tap(gluttertest.ByText("Step Three"))
	if len(tapped) != 1 || tapped[0] != 2 {
		t.Errorf("tapped %v", tapped)
	}
	// The header is 72 high without labels, with an elevation of 2 below it.
	if got := ts.Size(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Step One"), gluttertest.ByType[*Material]()))).Height; got != 72 {
		t.Errorf("the header is %v high, want 72", got)
	}
	// Labels make the header 104 high.
	labelled := threeSteps()
	labelled[0] = NewStep(widgets.NewText("Step One"), widgets.NewText("Content One"), StepOpts{Label: widgets.NewText("Label")})
	ts = pumpStepper(t, standardTheme(true), NewStepper(labelled, StepperOpts{Type: StepperTypeHorizontal}))
	if !ts.Exists(gluttertest.ByText("Label")) {
		t.Fatal("no label")
	}
	if got := ts.Size(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Step One"), gluttertest.ByType[*Material]()))).Height; got != 104 {
		t.Errorf("the header is %v high with labels, want 104", got)
	}
}

func TestStepperCustomControlsAndIcons(t *testing.T) {
	var details []ControlsDetails
	ts := pumpStepper(t, standardTheme(true), NewStepper(threeSteps(), StepperOpts{CurrentStep: 2, OnStepContinue: func() {},
		ControlsBuilder: func(bc widgets.BuildContext, d ControlsDetails) widgets.Widget {
			details = append(details, d)
			return widgets.NewText("controls")
		},
		StepIconBuilder: func(i int, s StepState) widgets.Widget {
			if i == 0 {
				return widgets.NewText("first!")
			}
			return nil
		}}))
	if !ts.Exists(gluttertest.ByText("controls")) || ts.Exists(gluttertest.ByText("Continue")) {
		t.Error("the controls are not the custom ones")
	}
	if len(details) < 3 || details[0].CurrentStep != 2 || details[2].StepIndex != 2 || !details[2].IsActive() || details[0].IsActive() {
		t.Errorf("details %+v", details)
	}
	if !ts.Exists(gluttertest.ByText("first!")) || !ts.Exists(gluttertest.ByText("2")) {
		t.Error("the icon builder is not used for the first step only")
	}
}

func TestStepperStyleAndColors(t *testing.T) {
	steps := threeSteps()
	steps[0] = NewStep(widgets.NewText("Step One"), widgets.NewText("x"), StepOpts{IsActive: true, StepStyle: &StepStyle{Color: foundation.Some(ui.NewColor(0xFF00AA00))}})
	theme := standardTheme(true)
	ts := pumpStepper(t, theme, NewStepper(steps))
	decorations := ts.Widgets(gluttertest.ByType[*widgets.AnimatedContainer]())
	if len(decorations) != 3 {
		t.Fatalf("%d circles, want 3", len(decorations))
	}
	colorOf := func(i int) ui.Color {
		d := decorations[i].(*widgets.AnimatedContainer).Decoration().(*painting.BoxDecoration)
		return d.Color.Value()
	}
	if colorOf(0) != ui.NewColor(0xFF00AA00) {
		t.Errorf("the style color is not used: %v", colorOf(0))
	}
	if colorOf(1) != theme.ColorScheme.OnSurface().WithOpacity(0.38) {
		t.Errorf("an inactive circle is %v", colorOf(1))
	}
	// An active step has the primary color.
	steps[1] = NewStep(widgets.NewText("Step Two"), widgets.NewText("y"), StepOpts{IsActive: true})
	ts = pumpStepper(t, theme, NewStepper(steps))
	decorations = ts.Widgets(gluttertest.ByType[*widgets.AnimatedContainer]())
	if colorOf(1) != theme.ColorScheme.Primary() {
		t.Errorf("an active circle is %v", colorOf(1))
	}
}

func TestStepStyle(t *testing.T) {
	a := StepStyle{Color: foundation.Some(ui.NewColor(0xFF010101)), ConnectorThickness: foundation.Some(2.0)}
	b := StepStyle{ErrorColor: foundation.Some(ui.NewColor(0xFF020202)), ConnectorThickness: foundation.Some(4.0)}
	m := a.Merge(&b)
	if m.Color != a.Color || m.ErrorColor != b.ErrorColor || m.ConnectorThickness != foundation.Some(4.0) {
		t.Errorf("merge %+v", m)
	}
	if a.Merge(nil).Equal(a) == false || a.Equal(b) {
		t.Error("Merge(nil) or Equal")
	}
	if c := a.CopyWith(StepStyleCopy{ConnectorThickness: foundation.Some(9.0)}); c.ConnectorThickness != foundation.Some(9.0) || c.Color != a.Color {
		t.Errorf("CopyWith %+v", c)
	}
}

func TestStepperErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		f    func()
		tag  string
	}{
		{"current step", func() { NewStepper(threeSteps(), StepperOpts{CurrentStep: 3}) }, "StepperCurrentStepRange"},
		{"icon size", func() {
			NewStepper(threeSteps(), StepperOpts{StepIconHeight: foundation.Some(10.0), StepIconWidth: foundation.Some(10.0)})
		}, "StepperIconSizeRange"},
		{"icon sizes differ", func() {
			NewStepper(threeSteps(), StepperOpts{StepIconHeight: foundation.Some(30.0), StepIconWidth: foundation.Some(40.0)})
		}, "StepperIconSizeMismatch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { wantRoute(t, recoverError(tc.f), tc.tag) })
	}
	t.Run("nested", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		wantRoute(t, recoverError(func() {
			inner := NewStepper(threeSteps())
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewStepper([]Step{NewStep(widgets.NewText("a"), inner, StepOpts{IsActive: true})})})}))
			takeAndPanic(ts)
		}), "StepperNested")
	})
}
