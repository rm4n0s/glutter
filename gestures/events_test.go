// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/gestures/events_test.dart. The tests of
// computeHitSlop, computePanSlop and computeScaleSlop come with step 5.6, which
// ports their constants and DeviceGestureSettings.

func TestNthButton(t *testing.T) {
	if got := NthMouseButton(2); got != SecondaryMouseButton {
		t.Errorf("NthMouseButton(2) = %d, want %d", got, SecondaryMouseButton)
	}
	if got := NthStylusButton(2); got != SecondaryStylusButton {
		t.Errorf("NthStylusButton(2) = %d, want %d", got, SecondaryStylusButton)
	}
}

func TestSmallestButton(t *testing.T) {
	for _, c := range []struct{ in, want int }{{0x0, 0x0}, {0x1, 0x1}, {0x200, 0x200}, {0x220, 0x20}} {
		if got := SmallestButton(c.in); got != c.want {
			t.Errorf("SmallestButton(%#x) = %#x, want %#x", c.in, got, c.want)
		}
	}
}

func TestIsSingleButton(t *testing.T) {
	for _, c := range []struct {
		in   int
		want bool
	}{{0x0, false}, {0x1, true}, {0x200, true}, {0x220, false}} {
		if got := IsSingleButton(c.in); got != c.want {
			t.Errorf("IsSingleButton(%#x) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFromMouseEvent(t *testing.T) {
	common := PointerEventOpts{
		TimeStamp:   24 * time.Hour,
		Kind:        ui.PointerDeviceKindUnknown,
		Device:      10,
		Position:    ui.Offset{Dx: 101, Dy: 202},
		Buttons:     someInt(7),
		Obscured:    true,
		PressureMax: someFloat(2.1),
		PressureMin: someFloat(1.1),
		DistanceMax: 110,
		Size:        11,
		RadiusMajor: 11,
		RadiusMinor: 9,
		RadiusMin:   1.1,
		RadiusMax:   22,
		Orientation: 1.1,
		Tilt:        1.1,
		Synthesized: true,
	}
	hoverOpts := common
	hoverOpts.Distance = 11
	hover := NewPointerHoverEvent(hoverOpts)
	move := NewPointerMoveEvent(common)

	tests := []struct {
		name   string
		from   PointerEvent
		make   func(PointerEvent) PointerEvent
		empty  PointerEvent
		isDown bool // whether the new event has the source's down
	}{
		{"enter from hover", hover, func(e PointerEvent) PointerEvent { return NewPointerEnterEventFromMouseEvent(e) }, NewPointerEnterEvent(), false},
		{"exit from hover", hover, func(e PointerEvent) PointerEvent { return NewPointerExitEventFromMouseEvent(e) }, NewPointerExitEvent(), false},
		{"enter from move", move, func(e PointerEvent) PointerEvent { return NewPointerEnterEventFromMouseEvent(e) }, NewPointerEnterEvent(), true},
		{"exit from move", move, func(e PointerEvent) PointerEvent { return NewPointerExitEventFromMouseEvent(e) }, NewPointerExitEvent(), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.make(tc.from)
			check := func(name string, got, want any) {
				t.Helper()
				if got != want {
					t.Errorf("%s = %v, want %v", name, got, want)
				}
			}
			src := tc.from
			check("timeStamp", got.TimeStamp(), src.TimeStamp())
			check("pointer", got.Pointer(), tc.empty.Pointer())
			check("kind", got.Kind(), src.Kind())
			check("device", got.Device(), src.Device())
			check("position", got.Position(), src.Position())
			check("buttons", got.Buttons(), src.Buttons())
			check("down", got.Down(), src.Down())
			if tc.isDown != src.Down() {
				t.Fatalf("the test source is down: %v, want %v", src.Down(), tc.isDown)
			}
			check("obscured", got.Obscured(), src.Obscured())
			check("pressure", got.Pressure(), tc.empty.Pressure())
			check("pressureMin", got.PressureMin(), src.PressureMin())
			check("pressureMax", got.PressureMax(), src.PressureMax())
			check("distance", got.Distance(), src.Distance())
			check("distanceMax", got.DistanceMax(), src.DistanceMax())
			check("size", got.Size(), src.Size())
			check("radiusMajor", got.RadiusMajor(), src.RadiusMajor())
			check("radiusMinor", got.RadiusMinor(), src.RadiusMinor())
			check("radiusMin", got.RadiusMin(), src.RadiusMin())
			check("radiusMax", got.RadiusMax(), src.RadiusMax())
			check("orientation", got.Orientation(), src.Orientation())
			check("tilt", got.Tilt(), src.Tilt())
			check("synthesized", got.Synthesized(), src.Synthesized())
		})
	}
}

func TestEventDefaults(t *testing.T) {
	tests := []struct {
		name    string
		event   PointerEvent
		buttons int
		down    bool
	}{
		{"down", NewPointerDownEvent(), PrimaryButton, true},
		{"move", NewPointerMoveEvent(), PrimaryButton, true},
		{"up", NewPointerUpEvent(), 0, false},
		{"hover", NewPointerHoverEvent(), 0, false},
		{"cancel", NewPointerCancelEvent(), 0, false},
		{"added", NewPointerAddedEvent(), 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.event.Buttons(); got != tc.buttons {
				t.Errorf("buttons = %d, want %d", got, tc.buttons)
			}
			if got := tc.event.Down(); got != tc.down {
				t.Errorf("down = %v, want %v", got, tc.down)
			}
			if got := tc.event.Kind(); got != ui.PointerDeviceKindTouch {
				t.Errorf("kind = %v, want touch", got)
			}
		})
	}
	pressure := map[string]float64{
		"down": 1, "move": 1, "up": 0, "hover": 0, "cancel": 0, "added": 0, "removed": 0, "enter": 0, "exit": 0,
		"scroll": 1, "panZoomUpdate": 1,
	}
	events := map[string]PointerEvent{
		"down": NewPointerDownEvent(), "move": NewPointerMoveEvent(), "up": NewPointerUpEvent(),
		"hover": NewPointerHoverEvent(), "cancel": NewPointerCancelEvent(), "added": NewPointerAddedEvent(),
		"removed": NewPointerRemovedEvent(), "enter": NewPointerEnterEvent(), "exit": NewPointerExitEvent(),
		"scroll": NewPointerScrollEvent(), "panZoomUpdate": NewPointerPanZoomUpdateEvent(),
	}
	for name, want := range pressure {
		if got := events[name].Pressure(); got != want {
			t.Errorf("%s pressure = %v, want %v", name, got, want)
		}
	}
	// A device may report the pressure of an up event.
	if got := NewPointerUpEvent(PointerEventOpts{Pressure: someFloat(0.5)}).Pressure(); got != 0.5 {
		t.Errorf("up pressure = %v, want 0.5", got)
	}
	// The kind of signals and pan/zoom events.
	if got := NewPointerScrollEvent().Kind(); got != ui.PointerDeviceKindMouse {
		t.Errorf("scroll kind = %v, want mouse", got)
	}
	if got := NewPointerPanZoomStartEvent(PointerEventOpts{Kind: ui.PointerDeviceKindMouse}).Kind(); got != ui.PointerDeviceKindTrackpad {
		t.Errorf("pan/zoom kind = %v, want trackpad", got)
	}
	// A constructor ignores what its Flutter counterpart does not take.
	if got := NewPointerAddedEvent(PointerEventOpts{Buttons: someInt(4), Delta: ui.Offset{Dx: 1}}); got.Buttons() != 0 || got.Delta() != (ui.Offset{}) {
		t.Errorf("added took buttons %d and delta %v, want neither", got.Buttons(), got.Delta())
	}
}

func TestRemovePerspectiveTransform(t *testing.T) {
	original := ui.NewMatrix4Identity()
	if changed := RemovePerspectiveTransform(original); !changed.Equals(original) {
		t.Errorf("the identity changed to %v", changed.Storage())
	}
	original = ui.NewMatrix4Identity()
	original.Scale(3, 3, 3)
	changed := RemovePerspectiveTransform(original)
	if changed.Equals(original) {
		t.Error("a scale by 3 did not change")
	}
	original.SetEntry(0, 2, 0)
	original.SetEntry(1, 2, 0)
	original.SetEntry(2, 2, 1)
	original.SetEntry(3, 2, 0)
	original.SetEntry(2, 0, 0)
	original.SetEntry(2, 1, 0)
	original.SetEntry(2, 3, 0)
	if !changed.Equals(original) {
		t.Errorf("got %v, want %v", changed.Storage(), original.Storage())
	}
}

func TestTransformPosition(t *testing.T) {
	position := ui.Offset{Dx: 20, Dy: 30}
	tests := []struct {
		name      string
		transform *ui.Matrix4
		want      ui.Offset
	}{
		{"nil", nil, position},
		{"identity", ui.NewMatrix4Identity(), position},
		{"translation", ui.NewMatrix4TranslationValues(10, 20, 0), ui.Offset{Dx: 30, Dy: 50}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TransformPosition(tc.transform, position); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTransformDeltaViaPositions(t *testing.T) {
	scale2 := ui.NewMatrix4Identity()
	scale2.Scale(2, 2, 1)
	end, delta := ui.Offset{Dx: 20, Dy: 30}, ui.Offset{Dx: 5, Dy: 5}
	tests := []struct {
		name      string
		transform *ui.Matrix4
		opts      []TransformDeltaViaPositionsOpts
		want      ui.Offset
	}{
		{"scale", scale2, nil, ui.Offset{Dx: 10, Dy: 10}},
		{"scale, end given", scale2, []TransformDeltaViaPositionsOpts{{TransformedEndPosition: foundation.Some(ui.Offset{Dx: 40, Dy: 60})}}, ui.Offset{Dx: 10, Dy: 10}},
		{"nil transform", nil, []TransformDeltaViaPositionsOpts{{TransformedEndPosition: foundation.Some(ui.Offset{Dx: 40, Dy: 60})}}, ui.Offset{Dx: 5, Dy: 5}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TransformDeltaViaPositions(end, delta, tc.transform, tc.opts...); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTransformingEvents(t *testing.T) {
	transform := ui.NewMatrix4Identity()
	transform.Scale(2, 2, 1)
	transform = transform.Multiply(ui.NewMatrix4TranslationValues(10, 20, 0))
	localPosition := ui.Offset{Dx: 60, Dy: 100}
	localDelta := ui.Offset{Dx: 10, Dy: 10}
	mouse := PointerEventOpts{
		TimeStamp:   2 * time.Second,
		Kind:        ui.PointerDeviceKindMouse,
		Device:      1,
		Position:    ui.Offset{Dx: 20, Dy: 30},
		Obscured:    true,
		PressureMin: someFloat(10),
		PressureMax: someFloat(60),
		Distance:    12,
		DistanceMax: 24,
		RadiusMin:   10,
		RadiusMax:   50,
		Orientation: 2,
		Tilt:        4,
	}
	withContact := mouse
	withContact.Pointer = 45
	withContact.Buttons = someInt(4)
	withContact.Size = 10
	withContact.RadiusMajor = 33
	withContact.RadiusMinor = 44
	withDelta := withContact
	withDelta.Delta = ui.Offset{Dx: 5, Dy: 5}
	withDelta.Synthesized = true
	pressed := withContact
	pressed.Pressure = someFloat(34)
	moving := withDelta
	moving.Pressure = someFloat(34)
	moving.PlatformData = 10
	plain := PointerEventOpts{TimeStamp: 2 * time.Second, Device: 1, Position: ui.Offset{Dx: 20, Dy: 30}}

	tests := []struct {
		name          string
		event         PointerEvent
		localDelta    ui.Offset
		wantDeltaSame bool
	}{
		{"added", NewPointerAddedEvent(mouse), ui.Offset{}, true},
		{"cancel", NewPointerCancelEvent(withContact), ui.Offset{}, true},
		{"down", NewPointerDownEvent(pressed), ui.Offset{}, true},
		{"enter", NewPointerEnterEvent(withDelta), localDelta, false},
		{"exit", NewPointerExitEvent(withDelta), localDelta, false},
		{"hover", NewPointerHoverEvent(withDelta), localDelta, false},
		{"move", NewPointerMoveEvent(moving), localDelta, false},
		{"removed", NewPointerRemovedEvent(mouse), ui.Offset{}, true},
		{"scroll", NewPointerScrollEvent(plain), ui.Offset{}, true},
		{"pan/zoom start", NewPointerPanZoomStartEvent(plain), ui.Offset{}, true},
		{"pan/zoom update", NewPointerPanZoomUpdateEvent(plain), ui.Offset{}, true},
		{"pan/zoom end", NewPointerPanZoomEndEvent(plain), ui.Offset{}, true},
		{"up", NewPointerUpEvent(pressed), ui.Offset{}, true},
		{"scroll inertia cancel", NewPointerScrollInertiaCancelEvent(plain), ui.Offset{}, true},
		{"scale", NewPointerScaleEvent(plain), ui.Offset{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.event
			if o.Position() != o.LocalPosition() || o.Delta() != o.LocalDelta() {
				t.Errorf("an original has local values %v, %v for %v, %v", o.LocalPosition(), o.LocalDelta(), o.Position(), o.Delta())
			}
			if o.Original() != nil || o.Transform() != nil {
				t.Errorf("an original has original %v and transform %v", o.Original(), o.Transform())
			}
			tr := o.Transformed(transform)
			if tr.Original() != o {
				t.Errorf("Original() = %v, want the original event", tr.Original())
			}
			if tr.Transform() != transform {
				t.Errorf("Transform() = %v, want the matrix", tr.Transform())
			}
			wantLocalDelta := tc.localDelta
			if tc.wantDeltaSame {
				wantLocalDelta = o.LocalDelta()
			}
			if tr.LocalDelta() != wantLocalDelta {
				t.Errorf("LocalDelta() = %v, want %v", tr.LocalDelta(), wantLocalDelta)
			}
			if tr.LocalPosition() != localPosition {
				t.Errorf("LocalPosition() = %v, want %v", tr.LocalPosition(), localPosition)
			}
			same := map[string][2]any{
				"buttons": {tr.Buttons(), o.Buttons()}, "delta": {tr.Delta(), o.Delta()},
				"device": {tr.Device(), o.Device()}, "distance": {tr.Distance(), o.Distance()},
				"distanceMax": {tr.DistanceMax(), o.DistanceMax()}, "distanceMin": {tr.DistanceMin(), o.DistanceMin()},
				"down": {tr.Down(), o.Down()}, "kind": {tr.Kind(), o.Kind()}, "obscured": {tr.Obscured(), o.Obscured()},
				"orientation": {tr.Orientation(), o.Orientation()}, "platformData": {tr.PlatformData(), o.PlatformData()},
				"pointer": {tr.Pointer(), o.Pointer()}, "position": {tr.Position(), o.Position()},
				"pressure": {tr.Pressure(), o.Pressure()}, "pressureMax": {tr.PressureMax(), o.PressureMax()},
				"pressureMin": {tr.PressureMin(), o.PressureMin()}, "radiusMajor": {tr.RadiusMajor(), o.RadiusMajor()},
				"radiusMax": {tr.RadiusMax(), o.RadiusMax()}, "radiusMin": {tr.RadiusMin(), o.RadiusMin()},
				"radiusMinor": {tr.RadiusMinor(), o.RadiusMinor()}, "size": {tr.Size(), o.Size()},
				"synthesized": {tr.Synthesized(), o.Synthesized()}, "tilt": {tr.Tilt(), o.Tilt()},
				"timeStamp": {tr.TimeStamp(), o.TimeStamp()},
			}
			for name, v := range same {
				if v[0] != v[1] {
					t.Errorf("%s = %v after Transformed, want %v", name, v[0], v[1])
				}
			}
			// Transforming a transformed event starts from the original; nil gives
			// the original back.
			other := ui.NewMatrix4TranslationValues(1, 1, 0)
			if again := tr.Transformed(other); again.Original() != o || again.Transform() != other {
				t.Errorf("a second Transformed has original %v and transform %v", again.Original(), again.Transform())
			}
			if back := tr.Transformed(nil); back != o {
				t.Errorf("Transformed(nil) = %v, want the original", back)
			}
			if o.Transformed(nil) != o {
				t.Error("Transformed(nil) of an original is not the event itself")
			}
		})
	}
}

func TestTransformedPanZoomUpdate(t *testing.T) {
	transform := ui.NewMatrix4Identity()
	transform.Scale(2, 2, 1)
	e := NewPointerPanZoomUpdateEvent(PointerEventOpts{
		Position: ui.Offset{Dx: 5, Dy: 5}, Pan: ui.Offset{Dx: 10, Dy: 20}, PanDelta: ui.Offset{Dx: 1, Dy: 2},
		Scale: someFloat(1.5), Rotation: 0.5,
	})
	tr := e.Transformed(transform).(*PointerPanZoomUpdateEvent)
	if tr.Pan() != e.Pan() || tr.PanDelta() != e.PanDelta() || tr.Scale() != 1.5 || tr.Rotation() != 0.5 {
		t.Errorf("the pan, scale or rotation changed: %v %v %v %v", tr.Pan(), tr.PanDelta(), tr.Scale(), tr.Rotation())
	}
	if want := (ui.Offset{Dx: 20, Dy: 40}); tr.LocalPan() != want {
		t.Errorf("LocalPan() = %v, want %v", tr.LocalPan(), want)
	}
	if want := (ui.Offset{Dx: 2, Dy: 4}); tr.LocalPanDelta() != want {
		t.Errorf("LocalPanDelta() = %v, want %v", tr.LocalPanDelta(), want)
	}
	if e.LocalPan() != e.Pan() || e.LocalPanDelta() != e.PanDelta() {
		t.Error("an original has local values that differ")
	}
}

func TestCopyWith(t *testing.T) {
	base := PointerEventOpts{
		TimeStamp:   24 * time.Hour,
		Kind:        ui.PointerDeviceKindUnknown,
		Device:      10,
		Position:    ui.Offset{Dx: 101, Dy: 202},
		Obscured:    true,
		PressureMax: someFloat(2.1),
		PressureMin: someFloat(1.1),
		DistanceMax: 110,
		RadiusMin:   1.1,
		RadiusMax:   22,
		Orientation: 1.1,
		Tilt:        1.1,
	}
	added := base
	added.Distance = 11
	hover := base
	hover.Distance = 11
	hover.Buttons = someInt(7)
	hover.Size, hover.RadiusMajor, hover.RadiusMinor, hover.Synthesized = 11, 11, 9, true
	down := base
	down.Pointer = 1
	down.Buttons = someInt(7)
	down.Size, down.RadiusMajor, down.RadiusMinor = 11, 11, 9
	move := down
	move.Delta = ui.Offset{Dx: 1, Dy: 2}
	move.Synthesized = true
	up := down
	removed := PointerEventOpts{
		TimeStamp: base.TimeStamp, Kind: base.Kind, Device: 10, Position: base.Position, Obscured: true,
		PressureMax: base.PressureMax, PressureMin: base.PressureMin, DistanceMax: 110, RadiusMin: 1.1, RadiusMax: 22,
	}

	position := ui.Offset{}
	timeStamp := 48 * time.Hour
	tests := []struct {
		name  string
		event PointerEvent
		empty PointerEvent
		// The values the copy takes from the event, by field; the others take the
		// default of an empty event.
		keeps []string
	}{
		{"added", NewPointerAddedEvent(added), NewPointerAddedEvent(),
			[]string{"kind", "device", "obscured", "pressureMin", "pressureMax", "distance", "distanceMax", "radiusMin", "radiusMax", "orientation", "tilt"}},
		{"hover", NewPointerHoverEvent(hover), NewPointerHoverEvent(),
			[]string{"kind", "device", "buttons", "obscured", "pressureMin", "pressureMax", "distance", "distanceMax", "size", "radiusMajor", "radiusMinor", "radiusMin", "radiusMax", "orientation", "tilt", "synthesized"}},
		{"down", NewPointerDownEvent(down), NewPointerDownEvent(),
			[]string{"pointer", "kind", "device", "buttons", "obscured", "pressureMin", "pressureMax", "distanceMax", "size", "radiusMajor", "radiusMinor", "radiusMin", "radiusMax", "orientation", "tilt"}},
		{"move", NewPointerMoveEvent(move), NewPointerMoveEvent(),
			[]string{"pointer", "kind", "device", "delta", "buttons", "obscured", "pressureMin", "pressureMax", "distanceMax", "size", "radiusMajor", "radiusMinor", "radiusMin", "radiusMax", "orientation", "tilt", "synthesized"}},
		{"up", NewPointerUpEvent(up), NewPointerUpEvent(),
			[]string{"pointer", "kind", "device", "buttons", "obscured", "pressureMin", "pressureMax", "distanceMax", "size", "radiusMajor", "radiusMinor", "radiusMin", "radiusMax", "orientation", "tilt"}},
		{"removed", NewPointerRemovedEvent(removed), NewPointerRemovedEvent(),
			[]string{"kind", "device", "obscured", "pressureMin", "pressureMax", "distanceMax", "radiusMin", "radiusMax"}},
	}
	field := func(e PointerEvent, name string) any {
		switch name {
		case "pointer":
			return e.Pointer()
		case "kind":
			return e.Kind()
		case "device":
			return e.Device()
		case "delta":
			return e.Delta()
		case "buttons":
			return e.Buttons()
		case "obscured":
			return e.Obscured()
		case "pressure":
			return e.Pressure()
		case "pressureMin":
			return e.PressureMin()
		case "pressureMax":
			return e.PressureMax()
		case "distance":
			return e.Distance()
		case "distanceMax":
			return e.DistanceMax()
		case "size":
			return e.Size()
		case "radiusMajor":
			return e.RadiusMajor()
		case "radiusMinor":
			return e.RadiusMinor()
		case "radiusMin":
			return e.RadiusMin()
		case "radiusMax":
			return e.RadiusMax()
		case "orientation":
			return e.Orientation()
		case "tilt":
			return e.Tilt()
		case "synthesized":
			return e.Synthesized()
		case "down":
			return e.Down()
		}
		t.Fatalf("unknown field %s", name)
		return nil
	}
	all := []string{"pointer", "kind", "device", "delta", "buttons", "obscured", "pressure", "pressureMin", "pressureMax", "distance", "distanceMax", "size", "radiusMajor", "radiusMinor", "radiusMin", "radiusMax", "orientation", "tilt", "synthesized", "down"}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.event.CopyWith(PointerEventCopyOpts{Position: foundation.Some(position), TimeStamp: foundation.Some(timeStamp)})
			if got.TimeStamp() != timeStamp || got.Position() != position {
				t.Errorf("time stamp %v and position %v, want %v and %v", got.TimeStamp(), got.Position(), timeStamp, position)
			}
			keeps := map[string]bool{}
			for _, k := range tc.keeps {
				keeps[k] = true
			}
			for _, name := range all {
				want := field(tc.empty, name)
				if keeps[name] {
					want = field(tc.event, name)
				}
				if name == "pressure" {
					// Down, move and up copy the pressure, but the events here have the default.
					want = field(tc.empty, name)
				}
				if g := field(got, name); g != want {
					t.Errorf("%s = %v, want %v", name, g, want)
				}
			}
		})
	}

	t.Run("replacements", func(t *testing.T) {
		got := NewPointerDownEvent(down).CopyWith(PointerEventCopyOpts{Buttons: someInt(2), Pressure: someFloat(0.25), Pointer: someInt(9)})
		if got.Buttons() != 2 || got.Pressure() != 0.25 || got.Pointer() != 9 {
			t.Errorf("buttons %d, pressure %v, pointer %d", got.Buttons(), got.Pressure(), got.Pointer())
		}
	})
	t.Run("keeps the transform", func(t *testing.T) {
		transform := ui.NewMatrix4TranslationValues(10, 20, 0)
		orig := NewPointerMoveEvent(move)
		got := orig.Transformed(transform).CopyWith(PointerEventCopyOpts{Position: foundation.Some(ui.Offset{Dx: 1, Dy: 1})})
		if got.Transform() != transform || got.LocalPosition() != (ui.Offset{Dx: 11, Dy: 21}) {
			t.Errorf("transform %v, local position %v", got.Transform(), got.LocalPosition())
		}
	})
	t.Run("scroll keeps the delta and the response", func(t *testing.T) {
		var allowed bool
		e := NewPointerScrollEvent(PointerEventOpts{ScrollDelta: ui.Offset{Dx: 3, Dy: 4}, OnRespond: func(a bool) { allowed = a }})
		c := e.CopyWith(PointerEventCopyOpts{Device: someInt(2)}).(*PointerScrollEvent)
		c.Respond(true)
		if c.ScrollDelta() != e.ScrollDelta() || c.Device() != 2 || !allowed {
			t.Errorf("delta %v, device %d, responded %v", c.ScrollDelta(), c.Device(), allowed)
		}
	})
}
