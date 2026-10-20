// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// openTimePicker pumps an app with a button that shows the time picker.
func openTimePicker(t *testing.T, initial TimeOfDay, opts ShowTimePickerOpts, size ui.Size) (*gluttertest.Tester, *<-chan foundation.Opt[TimeOfDay]) {
	t.Helper()
	var result <-chan foundation.Opt[TimeOfDay]
	ts := gluttertest.New(t, gluttertest.Opts{Size: size})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: linuxTheme(), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("OPEN"), widgets.GestureDetectorOpts{OnTap: func() {
			result = ShowTimePicker(bc, initial, opts)
		}}))
	})})}))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	if errs := ts.Errors(); len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	return ts, &result
}

func TestShowTimePickerResults(t *testing.T) {
	tests := []struct {
		name  string
		start TimeOfDay
		opts  ShowTimePickerOpts
		act   func(t *testing.T, ts *gluttertest.Tester)
		want  foundation.Opt[TimeOfDay]
	}{
		{name: "ok keeps the initial time", start: TimeOfDay{Hour: 7, Minute: 30}, act: func(t *testing.T, ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("OK"))
		}, want: foundation.Some(TimeOfDay{Hour: 7, Minute: 30})},
		{name: "midnight is a time", start: TimeOfDay{}, act: func(t *testing.T, ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("OK"))
		}, want: foundation.Some(TimeOfDay{})},
		{name: "cancel", start: TimeOfDay{Hour: 7, Minute: 30}, act: func(t *testing.T, ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("Cancel"))
		}, want: foundation.None[TimeOfDay]()},
		{name: "custom button texts", start: TimeOfDay{Hour: 1, Minute: 2}, opts: ShowTimePickerOpts{ConfirmText: "Done"}, act: func(t *testing.T, ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("Done"))
		}, want: foundation.Some(TimeOfDay{Hour: 1, Minute: 2})},
		{name: "typed time", start: TimeOfDay{Hour: 7, Minute: 30}, opts: ShowTimePickerOpts{InitialEntryMode: TimePickerEntryModeInput}, act: func(t *testing.T, ts *gluttertest.Tester) {
			fields := gluttertest.ByType[*widgets.EditableText]()
			ts.EnterText(gluttertest.First(fields), "11")
			ts.EnterText(gluttertest.Last(fields), "45")
			ts.Tap(gluttertest.ByText("OK"))
		}, want: foundation.Some(TimeOfDay{Hour: 11, Minute: 45})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, result := openTimePicker(t, tc.start, tc.opts, ui.Size{Width: 800, Height: 800})
			tc.act(t, ts)
			ts.PumpAndSettle()
			if errs := ts.Errors(); len(errs) > 0 {
				t.Fatalf("errors: %v", errs)
			}
			got := <-*result
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTimePickerEntryModeToggle(t *testing.T) {
	var modes []TimePickerEntryMode
	ts, result := openTimePicker(t, TimeOfDay{Hour: 9, Minute: 15}, ShowTimePickerOpts{
		OnEntryModeChanged: func(m TimePickerEntryMode) { modes = append(modes, m) },
	}, ui.Size{Width: 800, Height: 800})
	if n := ts.Count(gluttertest.ByType[*widgets.EditableText]()); n != 0 {
		t.Fatalf("dial mode shows %d text fields", n)
	}
	ts.Tap(gluttertest.ByIcon(Icons.KeyboardOutlined()))
	ts.PumpAndSettle()
	if n := ts.Count(gluttertest.ByType[*widgets.EditableText]()); n != 2 {
		t.Fatalf("input mode shows %d text fields, want 2", n)
	}
	ts.Tap(gluttertest.ByIcon(Icons.AccessTime()))
	ts.PumpAndSettle()
	if n := ts.Count(gluttertest.ByType[*widgets.EditableText]()); n != 0 {
		t.Fatalf("back in dial mode shows %d text fields", n)
	}
	want := []TimePickerEntryMode{TimePickerEntryModeInput, TimePickerEntryModeDial}
	if len(modes) != 2 || modes[0] != want[0] || modes[1] != want[1] {
		t.Errorf("modes %v, want %v", modes, want)
	}
	ts.Tap(gluttertest.ByText("OK"))
	ts.PumpAndSettle()
	if got := <-*result; got != foundation.Some(TimeOfDay{Hour: 9, Minute: 15}) {
		t.Errorf("got %v", got)
	}
}

func TestTimePickerInvalidInputKeepsDialogOpen(t *testing.T) {
	ts, result := openTimePicker(t, TimeOfDay{Hour: 9, Minute: 15}, ShowTimePickerOpts{InitialEntryMode: TimePickerEntryModeInputOnly}, ui.Size{Width: 800, Height: 800})
	fields := gluttertest.ByType[*widgets.EditableText]()
	ts.EnterText(gluttertest.Last(fields), "99")
	ts.Tap(gluttertest.ByText("OK"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Enter a valid time")) {
		t.Errorf("no error text for a minute of 99")
	}
	select {
	case v := <-*result:
		t.Fatalf("the dialog closed with %v", v)
	default:
	}
	ts.EnterText(gluttertest.Last(fields), "05")
	ts.Tap(gluttertest.ByText("OK"))
	ts.PumpAndSettle()
	if got := <-*result; got != foundation.Some(TimeOfDay{Hour: 9, Minute: 5}) {
		t.Errorf("got %v", got)
	}
}

func TestTimePickerDialTap(t *testing.T) {
	// Tapping the dial at the 3 o'clock position (minute 15) in minute mode and
	// at the top (12) in hour mode changes the time.
	ts, result := openTimePicker(t, TimeOfDay{Hour: 9, Minute: 0}, ShowTimePickerOpts{}, ui.Size{Width: 800, Height: 800})
	square := gluttertest.ByElementPredicate("square custom paint", func(e widgets.Element) bool {
		if _, ok := e.Widget().(*widgets.CustomPaint); !ok {
			return false
		}
		box, ok := e.RenderObject().(rendering.Box)
		return ok && box.Size().Width == 256
	})
	if !ts.Exists(square) {
		t.Fatalf("dial not found")
	}
	center := ts.Center(square)
	sz := ts.Size(square)
	radius := math.Min(sz.Width, sz.Height) / 2
	// Hour mode: tap the 3 o'clock label (outer ring).
	ts.TapAt(ui.Offset{Dx: center.Dx + radius - 28, Dy: center.Dy})
	ts.PumpAndSettle()
	// The dial moved on to the minutes: tap the 6 o'clock position (minute 30).
	ts.TapAt(ui.Offset{Dx: center.Dx, Dy: center.Dy + radius - 28})
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("OK"))
	ts.PumpAndSettle()
	if got := <-*result; got != foundation.Some(TimeOfDay{Hour: 3, Minute: 30}) {
		t.Errorf("got %v, want 3:30 AM", got)
	}
}
