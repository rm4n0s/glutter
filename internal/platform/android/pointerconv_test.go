package android

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// pcWant is what a test expects of one converted pointer data. Every field is compared, so a test
// states the delta, the pointer identifier and the like of every data it expects.
type pcWant struct {
	change       ui.PointerChange
	signal       ui.PointerSignalKind
	device, pid  int
	synth        bool
	x, y, dx, dy float64
	buttons      int
	view         ui.ViewID
	scrollX      float64
	scrollY      float64
	panX, panY   float64
	panDX, panDY float64
	scale        float64
	rotation     float64
}

func pcw(change ui.PointerChange, device, pid int, synth bool, x, y, dx, dy float64, buttons int) pcWant {
	return pcWant{change: change, signal: ui.PointerSignalKindNone, device: device, pid: pid, synth: synth, x: x, y: y, dx: dx, dy: dy, buttons: buttons}
}

func (w pcWant) onView(v ui.ViewID) pcWant { w.view = v; return w }

func (w pcWant) withSignal(k ui.PointerSignalKind, sx, sy float64) pcWant {
	w.signal, w.scrollX, w.scrollY = k, sx, sy
	return w
}

func (w pcWant) withPan(px, py, pdx, pdy, scale float64) pcWant {
	w.panX, w.panY, w.panDX, w.panDY, w.scale = px, py, pdx, pdy, scale
	return w
}

// pcTouch makes the data of CreateSimulatedPointerData: a touch with the given change, device,
// position and buttons.
func pcTouch(change ui.PointerChange, device int, x, y float64, buttons int) ui.PointerData {
	return ui.PointerData{
		Change: change, Kind: ui.PointerDeviceKindTouch, SignalKind: ui.PointerSignalKindNone,
		Device: device, PhysicalX: x, PhysicalY: y, Buttons: buttons,
	}
}

func pcOnView(d ui.PointerData, v ui.ViewID) ui.PointerData { d.ViewID = v; return d }

// pcMouse makes the data of CreateSimulatedMousePointerData.
func pcMouse(change ui.PointerChange, signal ui.PointerSignalKind, device int, x, y, sx, sy float64, buttons int) ui.PointerData {
	return ui.PointerData{
		Change: change, Kind: ui.PointerDeviceKindMouse, SignalKind: signal,
		Device: device, PhysicalX: x, PhysicalY: y, Buttons: buttons, ScrollDeltaX: sx, ScrollDeltaY: sy,
	}
}

// pcTrackpad makes the data of CreateSimulatedTrackpadGestureData.
func pcTrackpad(change ui.PointerChange, device int, x, y, panX, panY, scale, rotation float64) ui.PointerData {
	return ui.PointerData{
		Change: change, Kind: ui.PointerDeviceKindMouse, SignalKind: ui.PointerSignalKindNone,
		Device: device, PhysicalX: x, PhysicalY: y, PanX: panX, PanY: panY, Scale: scale, Rotation: rotation,
	}
}

func pcCheck(t *testing.T, got []ui.PointerData, want []pcWant) {
	t.Helper()
	if len(got) != len(want) {
		for i, g := range got {
			t.Logf("got[%d] = change %v device %d pid %d synth %v at (%v, %v)", i, g.Change, g.Device, g.PointerIdentifier, g.Synthesized, g.PhysicalX, g.PhysicalY)
		}
		t.Fatalf("got %d data, want %d", len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		have := pcWant{
			change: g.Change, signal: g.SignalKind, device: g.Device, pid: g.PointerIdentifier, synth: g.Synthesized,
			x: g.PhysicalX, y: g.PhysicalY, dx: g.PhysicalDeltaX, dy: g.PhysicalDeltaY, buttons: g.Buttons, view: g.ViewID,
			scrollX: g.ScrollDeltaX, scrollY: g.ScrollDeltaY,
			panX: g.PanX, panY: g.PanY, panDX: g.PanDeltaX, panDY: g.PanDeltaY, scale: g.Scale, rotation: g.Rotation,
		}
		if have != w {
			t.Errorf("data %d:\n got  %+v\n want %+v", i, have, w)
		}
	}
}

const (
	pcAdd    = ui.PointerChangeAdd
	pcHover  = ui.PointerChangeHover
	pcDown   = ui.PointerChangeDown
	pcMove   = ui.PointerChangeMove
	pcUp     = ui.PointerChangeUp
	pcRemove = ui.PointerChangeRemove
	pcCancel = ui.PointerChangeCancel
)

// TestPointerConverter ports the tests of pointer_data_packet_converter_unittests.cc that run one
// packet through a converter, with every field of the result checked.
func TestPointerConverter(t *testing.T) {
	tests := []struct {
		name string
		// views are the views that exist; nil means the implicit view only.
		views []ui.ViewID
		in    []ui.PointerData
		want  []pcWant
	}{
		{
			name: "CanConvertPointerDataPacket",
			in: []ui.PointerData{
				pcTouch(pcAdd, 0, 0, 0, 0),
				pcTouch(pcHover, 0, 3, 0, 0),
				pcTouch(pcDown, 0, 3, 0, 1),
				pcTouch(pcMove, 0, 3, 4, 1),
				pcTouch(pcUp, 0, 3, 4, 0),
				pcTouch(pcRemove, 0, 3, 4, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0),
				pcw(pcHover, 0, 0, false, 3, 0, 3, 0, 0),
				pcw(pcDown, 0, 1, false, 3, 0, 0, 0, 1),
				pcw(pcMove, 0, 1, false, 3, 4, 0, 4, 1),
				pcw(pcUp, 0, 1, false, 3, 4, 0, 0, 0),
				pcw(pcRemove, 0, 0, false, 3, 4, 0, 0, 0),
			},
		},
		{
			name: "CanSynthesizeDownAndUp",
			in: []ui.PointerData{
				pcTouch(pcAdd, 0, 0, 0, 0),
				pcTouch(pcDown, 0, 3, 0, 1),
				pcTouch(pcUp, 0, 3, 4, 0),
				pcTouch(pcRemove, 0, 3, 4, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0),
				// A hover is synthesized.
				pcw(pcHover, 0, 0, true, 3, 0, 3, 0, 0),
				pcw(pcDown, 0, 1, false, 3, 0, 0, 0, 1),
				// A move is synthesized, with the buttons of the down.
				pcw(pcMove, 0, 1, true, 3, 4, 0, 4, 1),
				pcw(pcUp, 0, 1, false, 3, 4, 0, 0, 0),
				pcw(pcRemove, 0, 0, false, 3, 4, 0, 0, 0),
			},
		},
		{
			name: "CanUpdatePointerIdentifier",
			in: []ui.PointerData{
				pcTouch(pcAdd, 0, 0, 0, 0),
				pcTouch(pcDown, 0, 0, 0, 1),
				pcTouch(pcUp, 0, 0, 0, 0),
				pcTouch(pcDown, 0, 0, 0, 1),
				pcTouch(pcMove, 0, 3, 0, 1),
				pcTouch(pcUp, 0, 3, 0, 0),
				pcTouch(pcRemove, 0, 3, 0, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0),
				pcw(pcDown, 0, 1, false, 0, 0, 0, 0, 1),
				pcw(pcUp, 0, 1, false, 0, 0, 0, 0, 0),
				// The pointer count increases to 2.
				pcw(pcDown, 0, 2, false, 0, 0, 0, 0, 1),
				pcw(pcMove, 0, 2, false, 3, 0, 3, 0, 1),
				pcw(pcUp, 0, 2, false, 3, 0, 0, 0, 0),
				pcw(pcRemove, 0, 0, false, 3, 0, 0, 0, 0),
			},
		},
		{
			name: "AlwaysForwardMoveEvent",
			in: []ui.PointerData{
				pcTouch(pcAdd, 0, 0, 0, 0),
				pcTouch(pcDown, 0, 0, 0, 1),
				// A move without a location change.
				pcTouch(pcMove, 0, 0, 0, 1),
				pcTouch(pcUp, 0, 0, 0, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0),
				pcw(pcDown, 0, 1, false, 0, 0, 0, 0, 1),
				// The move is not filtered out.
				pcw(pcMove, 0, 1, false, 0, 0, 0, 0, 1),
				pcw(pcUp, 0, 1, false, 0, 0, 0, 0, 0),
			},
		},
		{
			name: "CanWorkWithDifferentDevices",
			in: []ui.PointerData{
				pcTouch(pcAdd, 0, 0, 0, 0),
				pcTouch(pcDown, 0, 0, 0, 1),
				pcTouch(pcAdd, 1, 0, 0, 0),
				pcTouch(pcDown, 1, 0, 0, 1),
				pcTouch(pcUp, 0, 0, 0, 0),
				pcTouch(pcDown, 0, 0, 0, 1),
				pcTouch(pcMove, 1, 0, 4, 1),
				pcTouch(pcMove, 0, 3, 0, 1),
				pcTouch(pcUp, 1, 0, 4, 0),
				pcTouch(pcUp, 0, 3, 0, 0),
				pcTouch(pcRemove, 0, 3, 0, 0),
				pcTouch(pcRemove, 1, 0, 4, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0),
				pcw(pcDown, 0, 1, false, 0, 0, 0, 0, 1),
				pcw(pcAdd, 1, 0, false, 0, 0, 0, 0, 0),
				pcw(pcDown, 1, 2, false, 0, 0, 0, 0, 1),
				pcw(pcUp, 0, 1, false, 0, 0, 0, 0, 0),
				pcw(pcDown, 0, 3, false, 0, 0, 0, 0, 1),
				pcw(pcMove, 1, 2, false, 0, 4, 0, 4, 1),
				pcw(pcMove, 0, 3, false, 3, 0, 3, 0, 1),
				pcw(pcUp, 1, 2, false, 0, 4, 0, 0, 0),
				pcw(pcUp, 0, 3, false, 3, 0, 0, 0, 0),
				pcw(pcRemove, 0, 0, false, 3, 0, 0, 0, 0),
				pcw(pcRemove, 1, 0, false, 0, 4, 0, 0, 0),
			},
		},
		{
			name: "CanSynthesizeAdd",
			in: []ui.PointerData{
				pcTouch(pcDown, 0, 330, 450, 1),
				pcTouch(pcUp, 0, 0, 0, 0),
			},
			want: []pcWant{
				// An add is synthesized, at the place of the down and with no buttons.
				pcw(pcAdd, 0, 0, true, 330, 450, 0, 0, 0),
				pcw(pcDown, 0, 1, false, 330, 450, 0, 0, 1),
				// A move is synthesized to the place of the up.
				pcw(pcMove, 0, 1, true, 0, 0, -330, -450, 1),
				pcw(pcUp, 0, 1, false, 0, 0, 0, 0, 0),
			},
		},
		{
			name:  "CanSynthesizeRemove",
			views: []ui.ViewID{100, 200},
			in: []ui.PointerData{
				pcOnView(pcTouch(pcAdd, 0, 0, 0, 0), 100),
				pcOnView(pcTouch(pcDown, 0, 3, 4, 1), 100),
				pcOnView(pcTouch(pcAdd, 0, 0, 0, 0), 200),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0).onView(100),
				// A hover is synthesized.
				pcw(pcHover, 0, 0, true, 3, 4, 3, 4, 0).onView(100),
				pcw(pcDown, 0, 1, false, 3, 4, 0, 0, 1).onView(100),
				// A cancel is synthesized, where the pointer is and with its buttons...
				pcw(pcCancel, 0, 1, true, 3, 4, 0, 0, 1).onView(100),
				// ...and then a remove, on the view the pointer was added to.
				pcw(pcRemove, 0, 0, true, 3, 4, 0, 0, 1).onView(100),
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0).onView(200),
			},
		},
		{
			name:  "CanAvoidDoubleRemoveAfterSynthesizedRemove",
			views: []ui.ViewID{100, 200},
			in: []ui.PointerData{
				pcOnView(pcTouch(pcAdd, 0, 0, 0, 0), 100),
				pcOnView(pcTouch(pcAdd, 0, 0, 0, 0), 200),
				// A double remove: the pointer belongs to view 200 now.
				pcOnView(pcTouch(pcRemove, 0, 0, 0, 0), 100),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0).onView(100),
				pcw(pcRemove, 0, 0, true, 0, 0, 0, 0, 0).onView(100),
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0).onView(200),
			},
		},
		{
			name: "CanHandleThreeFingerGesture",
			in: []ui.PointerData{
				// First finger down, second finger down.
				pcTouch(pcDown, 0, 0, 0, 1),
				pcTouch(pcDown, 1, 33, 44, 1),
				// Three cancels, the last for a pointer that was never added.
				pcTouch(pcCancel, 1, 33, 44, 0),
				pcTouch(pcCancel, 0, 0, 0, 0),
				pcTouch(pcCancel, 2, 40, 50, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, true, 0, 0, 0, 0, 0),
				pcw(pcDown, 0, 1, false, 0, 0, 0, 0, 1),
				pcw(pcAdd, 1, 0, true, 33, 44, 0, 0, 0),
				pcw(pcDown, 1, 2, false, 33, 44, 0, 0, 1),
				pcw(pcCancel, 1, 2, false, 33, 44, 0, 0, 0),
				pcw(pcCancel, 0, 1, false, 0, 0, 0, 0, 0),
				// The third cancel is dropped.
			},
		},
		{
			name: "CanConvertTrackpadGesture",
			in: []ui.PointerData{
				pcTrackpad(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 0, 1, 0),
				pcTrackpad(ui.PointerChangePanZoomUpdate, 0, 0, 0, 3, 4, 1, 0),
				pcTrackpad(ui.PointerChangePanZoomEnd, 0, 0, 0, 0, 0, 1, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, true, 0, 0, 0, 0, 0).withPan(0, 0, 0, 0, 1),
				pcw(ui.PointerChangePanZoomStart, 0, 1, false, 0, 0, 0, 0, 0).withPan(0, 0, 0, 0, 1),
				pcw(ui.PointerChangePanZoomUpdate, 0, 1, false, 0, 0, 0, 0, 0).withPan(3, 4, 3, 4, 1),
				pcw(ui.PointerChangePanZoomEnd, 0, 1, false, 0, 0, 0, 0, 0).withPan(0, 0, 0, 0, 1),
			},
		},
		{
			name: "PanZoomStartAtAnotherPlaceSynthesizesAHover",
			in: []ui.PointerData{
				pcTouch(pcAdd, 0, 0, 0, 0),
				pcTrackpad(ui.PointerChangePanZoomStart, 0, 5, 6, 0, 0, 1, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0),
				pcw(pcHover, 0, 0, true, 5, 6, 5, 6, 0).withPan(0, 0, 0, 0, 1),
				pcw(ui.PointerChangePanZoomStart, 0, 1, false, 5, 6, 0, 0, 0).withPan(0, 0, 0, 0, 1),
			},
		},
		{
			name: "PanZoomEndAtAnotherPlaceSynthesizesAnUpdate",
			in: []ui.PointerData{
				pcTrackpad(ui.PointerChangePanZoomStart, 0, 0, 0, 0, 0, 1, 0),
				pcTrackpad(ui.PointerChangePanZoomUpdate, 0, 0, 0, 3, 4, 2, 0.5),
				pcTrackpad(ui.PointerChangePanZoomEnd, 0, 7, 8, 0, 0, 1, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, true, 0, 0, 0, 0, 0).withPan(0, 0, 0, 0, 1),
				pcw(ui.PointerChangePanZoomStart, 0, 1, false, 0, 0, 0, 0, 0).withPan(0, 0, 0, 0, 1),
				func() pcWant {
					w := pcw(ui.PointerChangePanZoomUpdate, 0, 1, false, 0, 0, 0, 0, 0).withPan(3, 4, 3, 4, 2)
					w.rotation = 0.5
					return w
				}(),
				// The synthesized update keeps the pan, the scale and the rotation, and moves.
				func() pcWant {
					w := pcw(ui.PointerChangePanZoomUpdate, 0, 1, true, 7, 8, 7, 8, 0).withPan(3, 4, 0, 0, 2)
					w.rotation = 0.5
					return w
				}(),
				pcw(ui.PointerChangePanZoomEnd, 0, 1, false, 7, 8, 0, 0, 0).withPan(0, 0, 0, 0, 1),
			},
		},
		{
			name:  "CanConvertViewId",
			views: []ui.ViewID{100, 200},
			in: []ui.PointerData{
				pcOnView(pcTouch(pcAdd, 0, 0, 0, 0), 100),
				pcOnView(pcTouch(pcHover, 0, 1, 0, 0), 200),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0).onView(100),
				pcw(pcHover, 0, 0, false, 1, 0, 1, 0, 0).onView(200),
			},
		},
		{
			name:  "AnInvalidViewIsIgnored",
			views: []ui.ViewID{100},
			in: []ui.PointerData{
				pcOnView(pcTouch(pcAdd, 0, 0, 0, 0), 100),
				pcOnView(pcTouch(pcHover, 0, 1, 0, 0), 300),
				pcOnView(pcTouch(pcHover, 0, 2, 0, 0), 100),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0).onView(100),
				pcw(pcHover, 0, 0, false, 2, 0, 2, 0, 0).onView(100),
			},
		},
		{
			name: "AHoverWithoutALocationChangeIsDropped",
			in: []ui.PointerData{
				pcTouch(pcHover, 0, 4, 4, 0),
				pcTouch(pcHover, 0, 4, 4, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, true, 4, 4, 0, 0, 0),
			},
		},
		{
			name: "AMoveOrUpOfAnUnknownDeviceIsDropped",
			in: []ui.PointerData{
				pcTouch(pcMove, 7, 1, 1, 1),
				pcTouch(pcUp, 7, 1, 1, 0),
				pcTouch(pcRemove, 7, 1, 1, 0),
			},
			want: nil,
		},
		{
			name: "ARemoveOfAPointerThatIsDownCancelsAndHoversFirst",
			in: []ui.PointerData{
				pcTouch(pcDown, 0, 1, 1, 1),
				pcTouch(pcRemove, 0, 5, 5, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, true, 1, 1, 0, 0, 0),
				pcw(pcDown, 0, 1, false, 1, 1, 0, 0, 1),
				pcw(pcCancel, 0, 1, true, 5, 5, 0, 0, 0),
				pcw(pcHover, 0, 0, true, 5, 5, 4, 4, 0),
				pcw(pcRemove, 0, 0, false, 5, 5, 0, 0, 0),
			},
		},
		{
			name: "ACancelAtAnotherPlaceMovesFirst",
			in: []ui.PointerData{
				pcTouch(pcDown, 0, 1, 1, 1),
				pcTouch(pcCancel, 0, 2, 3, 0),
			},
			want: []pcWant{
				pcw(pcAdd, 0, 0, true, 1, 1, 0, 0, 0),
				pcw(pcDown, 0, 1, false, 1, 1, 0, 0, 1),
				pcw(pcMove, 0, 1, true, 2, 3, 1, 2, 0),
				pcw(pcCancel, 0, 1, false, 2, 3, 0, 0, 0),
			},
		},
		{
			name: "AnUnknownSignalIsIgnored",
			in: []ui.PointerData{
				pcMouse(pcHover, ui.PointerSignalKindUnknown, 0, 1, 1, 0, 0, 0),
			},
			want: nil,
		},
		{
			name: "AChangeThatHasNoCaseIsForwarded",
			in: []ui.PointerData{
				pcTouch(ui.PointerChangeUnspecified, 0, 1, 1, 0),
			},
			want: []pcWant{
				{change: ui.PointerChangeUnspecified, signal: ui.PointerSignalKindNone, x: 1, y: 1},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newPointerConverter()
			if tc.views != nil {
				views := map[ui.ViewID]bool{}
				for _, v := range tc.views {
					views[v] = true
				}
				c.viewExists = func(id ui.ViewID) bool { return views[id] }
			}
			pcCheck(t, c.Convert(tc.in), tc.want)
		})
	}
}

// TestPointerConverterPacketsKeepTheState checks that the state is kept between packets: the
// converter of the platform sees one packet per event.
func TestPointerConverterPacketsKeepTheState(t *testing.T) {
	c := newPointerConverter()
	var got []ui.PointerData
	got = append(got, c.Convert([]ui.PointerData{pcTouch(pcDown, 0, 0, 0, 1)})...)
	got = append(got, c.Convert([]ui.PointerData{pcTouch(pcMove, 0, 2, 2, 1)})...)
	got = append(got, c.Convert([]ui.PointerData{pcTouch(pcUp, 0, 2, 2, 0)})...)
	got = append(got, c.Convert([]ui.PointerData{pcTouch(pcRemove, 0, 2, 2, 0)})...)
	got = append(got, c.Convert([]ui.PointerData{pcTouch(pcDown, 0, 9, 9, 1)})...)
	pcCheck(t, got, []pcWant{
		pcw(pcAdd, 0, 0, true, 0, 0, 0, 0, 0),
		pcw(pcDown, 0, 1, false, 0, 0, 0, 0, 1),
		pcw(pcMove, 0, 1, false, 2, 2, 2, 2, 1),
		pcw(pcUp, 0, 1, false, 2, 2, 0, 0, 0),
		pcw(pcRemove, 0, 0, false, 2, 2, 0, 0, 0),
		// The device was removed: the next contact is added again, and has the next identifier.
		pcw(pcAdd, 0, 0, true, 9, 9, 0, 0, 0),
		pcw(pcDown, 0, 2, false, 9, 9, 0, 0, 1),
	})
}

// TestPointerConverterSignals ports CanConvertPointerSignals for each kind of signal.
func TestPointerConverterSignals(t *testing.T) {
	kinds := []ui.PointerSignalKind{
		ui.PointerSignalKindScroll,
		ui.PointerSignalKindScrollInertiaCancel,
		ui.PointerSignalKindScale,
	}
	for _, kind := range kinds {
		t.Run(kind.String(), func(t *testing.T) {
			none := ui.PointerSignalKindNone
			in := []ui.PointerData{
				pcMouse(pcAdd, none, 0, 0, 0, 0, 0, 0),
				pcMouse(pcAdd, none, 1, 0, 0, 0, 0, 0),
				pcMouse(pcDown, none, 1, 0, 0, 0, 0, 1),
				pcMouse(pcHover, kind, 0, 34, 34, 30, 0, 0),
				pcMouse(pcHover, kind, 1, 49, 49, 50, 0, 0),
				pcMouse(pcHover, kind, 2, 10, 20, 30, 40, 0),
			}
			// The signal data are the ones of the input, but for the deltas of the synthesized data.
			want := []pcWant{
				pcw(pcAdd, 0, 0, false, 0, 0, 0, 0, 0),
				pcw(pcAdd, 1, 0, false, 0, 0, 0, 0, 0),
				pcw(pcDown, 1, 1, false, 0, 0, 0, 0, 1),
				// A hover is synthesized to the position for device 0.
				func() pcWant {
					w := pcw(pcHover, 0, 0, true, 34, 34, 34, 34, 0)
					w.scrollX = 30
					return w
				}(),
				pcw(pcHover, 0, 0, false, 34, 34, 0, 0, 0).withSignal(kind, 30, 0),
				// A move is synthesized to the position for device 1, with the buttons of the down.
				func() pcWant {
					w := pcw(pcMove, 1, 0, true, 49, 49, 49, 49, 1)
					w.scrollX = 50
					return w
				}(),
				pcw(pcHover, 1, 0, false, 49, 49, 0, 0, 0).withSignal(kind, 50, 0),
				// An add is synthesized for device 2.
				func() pcWant {
					w := pcw(pcAdd, 2, 0, true, 10, 20, 0, 0, 0)
					w.scrollX, w.scrollY = 30, 40
					return w
				}(),
				pcw(pcHover, 2, 0, false, 10, 20, 0, 0, 0).withSignal(kind, 30, 40),
			}
			pcCheck(t, newPointerConverter().Convert(in), want)
		})
	}
}

// TestPointerConverterIsNotGlobal checks that two converters do not share the identifiers.
func TestPointerConverterIsNotGlobal(t *testing.T) {
	a, b := newPointerConverter(), newPointerConverter()
	a.Convert([]ui.PointerData{pcTouch(pcDown, 0, 0, 0, 1)})
	got := b.Convert([]ui.PointerData{pcTouch(pcDown, 0, 0, 0, 1)})
	if len(got) != 2 || got[1].PointerIdentifier != 1 {
		t.Fatalf("second converter gave %v, want its own first identifier 1", got)
	}
}
