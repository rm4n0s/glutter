package ios

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

func TestSnapshotOfMapsTheButtonsAndAxes(t *testing.T) {
	tests := []struct {
		name   string
		state  gcState
		button gamepaddb.Button
	}{
		{"A is south", gcState{A: true}, gamepaddb.South},
		{"B is east", gcState{B: true}, gamepaddb.East},
		{"X is west", gcState{X: true}, gamepaddb.West},
		{"Y is north", gcState{Y: true}, gamepaddb.North},
		{"the menu button is start", gcState{Menu: true}, gamepaddb.Start},
		{"the options button is back", gcState{Options: true}, gamepaddb.Back},
		{"the home button is the guide", gcState{Home: true}, gamepaddb.Guide},
		{"the left shoulder", gcState{LeftShoulder: true}, gamepaddb.LeftShoulder},
		{"the right shoulder", gcState{RightShoulder: true}, gamepaddb.RightShoulder},
		{"the left stick's press", gcState{LeftThumbstickButton: true}, gamepaddb.LeftStick},
		{"the right stick's press", gcState{RightThumbstickButton: true}, gamepaddb.RightStick},
		{"d-pad up", gcState{DPadUp: true}, gamepaddb.DPadUp},
		{"d-pad down", gcState{DPadDown: true}, gamepaddb.DPadDown},
		{"d-pad left", gcState{DPadLeft: true}, gamepaddb.DPadLeft},
		{"d-pad right", gcState{DPadRight: true}, gamepaddb.DPadRight},
		{"a paddle", gcState{Paddle3: true}, gamepaddb.Paddle3},
		{"the touchpad", gcState{Touchpad: true}, gamepaddb.Touchpad},
		{"the share button", gcState{Share: true}, gamepaddb.Misc1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := snapshotOf(tc.state)
			for b := gamepaddb.Button(0); b < gamepaddb.ButtonCount; b++ {
				if snap.Buttons[b] != (b == tc.button) {
					t.Errorf("button %d is %v", b, snap.Buttons[b])
				}
			}
		})
	}
	// The axes: up is the layout's negative y, the triggers are 0 to 32767.
	snap := snapshotOf(gcState{LeftX: 1, LeftY: 1, RightX: -1, RightY: -1, LeftTrigger: 1, RightTrigger: 0.5})
	if snap.Axes[gamepaddb.LeftX] != gamepaddb.AxisMax || snap.Axes[gamepaddb.LeftY] != gamepaddb.AxisMin ||
		snap.Axes[gamepaddb.RightX] != gamepaddb.AxisMin || snap.Axes[gamepaddb.RightY] != gamepaddb.AxisMax {
		t.Errorf("the sticks: %v", snap.Axes)
	}
	if snap.Axes[gamepaddb.LeftTrigger] != gamepaddb.AxisMax || snap.Axes[gamepaddb.RightTrigger] != 16383 {
		t.Errorf("the triggers: %d, %d", snap.Axes[gamepaddb.LeftTrigger], snap.Axes[gamepaddb.RightTrigger])
	}
	// At rest everything is zero.
	if rest := snapshotOf(gcState{}); rest.Axes != [gamepaddb.AxisCount]int16{} {
		t.Errorf("at rest: %v", rest.Axes)
	}
}

func TestButtonMaskOf(t *testing.T) {
	has := func(mask uint32, b gamepaddb.Button) bool { return mask&(1<<b) != 0 }
	base := buttonMaskOf(gcFeatures{})
	for _, b := range []gamepaddb.Button{gamepaddb.South, gamepaddb.East, gamepaddb.West, gamepaddb.North, gamepaddb.LeftShoulder,
		gamepaddb.RightShoulder, gamepaddb.DPadUp, gamepaddb.DPadDown, gamepaddb.DPadLeft, gamepaddb.DPadRight, gamepaddb.Start} {
		if !has(base, b) {
			t.Errorf("a controller of the extended profile lacks button %d", b)
		}
	}
	for _, b := range []gamepaddb.Button{gamepaddb.Back, gamepaddb.Guide, gamepaddb.LeftStick, gamepaddb.RightStick, gamepaddb.Paddle1, gamepaddb.Touchpad, gamepaddb.Misc1} {
		if has(base, b) {
			t.Errorf("a plain controller has button %d", b)
		}
	}
	full := buttonMaskOf(gcFeatures{Options: true, Home: true, ThumbstickButtons: true, Paddles: true, Touchpad: true, Share: true})
	for _, b := range []gamepaddb.Button{gamepaddb.Back, gamepaddb.Guide, gamepaddb.LeftStick, gamepaddb.RightStick, gamepaddb.Paddle1,
		gamepaddb.Paddle2, gamepaddb.Paddle3, gamepaddb.Paddle4, gamepaddb.Touchpad, gamepaddb.Misc1} {
		if !has(full, b) {
			t.Errorf("a full controller lacks button %d", b)
		}
	}
}

// padRecorder is a PadHandler that records what it hears.
type padRecorder struct {
	added   []platform.PadInfo
	removed []platform.PadID
	events  [][]platform.PadEvent
}

func (r *padRecorder) PadAdded(i platform.PadInfo)     { r.added = append(r.added, i) }
func (r *padRecorder) PadRemoved(id platform.PadID)    { r.removed = append(r.removed, id) }
func (r *padRecorder) PadEvents(e []platform.PadEvent) { r.events = append(r.events, e) }

func TestPadSetReportsConnectionsAndEvents(t *testing.T) {
	s := newPadSet()
	// A controller that is there before Start is reported at Start, and not before.
	id, fresh := s.add(10, padSpec{Name: "Xbox Wireless Controller", Features: gcFeatures{Options: true, Home: true}, Rumble: true, LED: false, Battery: 0.5})
	if !fresh || id != 1 {
		t.Fatalf("add: %d, %v", id, fresh)
	}
	h := &padRecorder{}
	s.start(h)
	if len(h.added) != 1 {
		t.Fatalf("added = %d", len(h.added))
	}
	info := h.added[0]
	if info.ID != 1 || info.Name != "Xbox Wireless Controller" || !info.Standard || !info.Rumble || info.LED || info.Battery != 50 ||
		info.Axes != int(gamepaddb.AxisCount) || info.Buttons != int(gamepaddb.ButtonCount) || info.GUID == ([16]byte{}) {
		t.Errorf("info = %+v", info)
	}
	if info.ButtonMask&(1<<gamepaddb.Back) == 0 || info.ButtonMask&(1<<gamepaddb.LeftStick) != 0 {
		t.Errorf("the mask of the buttons: %b", info.ButtonMask)
	}
	// A second add of the same controller changes nothing.
	if id2, fresh := s.add(10, padSpec{Name: "x"}); fresh || id2 != 1 || len(h.added) != 1 {
		t.Errorf("a repeat: %d, %v, %d", id2, fresh, len(h.added))
	}
	// A controller that connects later is reported at once.
	if id, _ := s.add(20, padSpec{Name: "DualSense", Battery: -1}); id != 2 || len(h.added) != 2 || h.added[1].Battery != -1 {
		t.Errorf("the second pad: %d, %+v", id, h.added)
	}

	// Events: only what changed.
	s.update(10, gcState{A: true, LeftX: 1}, time.Second)
	if len(h.events) != 1 || len(h.events[0]) != 2 {
		t.Fatalf("events = %+v", h.events)
	}
	if e := h.events[0][0]; e.Pad != 1 || e.Kind != platform.PadButton || e.Index != int(gamepaddb.South) || e.Value != 1 || e.Time != time.Second {
		t.Errorf("the button event: %+v", e)
	}
	if e := h.events[0][1]; e.Kind != platform.PadAxis || e.Index != int(gamepaddb.LeftX) || e.Value != gamepaddb.AxisMax {
		t.Errorf("the axis event: %+v", e)
	}
	s.update(10, gcState{A: true, LeftX: 1}, 2*time.Second)
	if len(h.events) != 1 {
		t.Errorf("an unchanged state gave events: %+v", h.events)
	}
	s.update(10, gcState{}, 3*time.Second)
	if len(h.events) != 2 || len(h.events[1]) != 2 || h.events[1][0].Value != 0 {
		t.Errorf("the release: %+v", h.events)
	}
	// A controller that is not known gives nothing.
	s.update(99, gcState{A: true}, 4*time.Second)
	if len(h.events) != 2 {
		t.Errorf("an unknown controller gave events")
	}

	// A controller that goes is reported once.
	s.remove(10)
	s.remove(10)
	if len(h.removed) != 1 || h.removed[0] != 1 {
		t.Errorf("removed = %v", h.removed)
	}
	if _, ok := s.id(10); ok {
		t.Error("the removed pad is still known")
	}
	if key, ok := s.keyOf(2); !ok || key != 20 {
		t.Errorf("keyOf(2) = %d, %v", key, ok)
	}
}

func TestPadSetStopAndStartAgain(t *testing.T) {
	s := newPadSet()
	s.add(1, padSpec{Name: "A", Battery: -1})
	h1 := &padRecorder{}
	s.start(h1)
	s.stop()
	// Nothing reaches the handler after Stop, even for a controller that connects or goes.
	s.add(2, padSpec{Name: "B", Battery: -1})
	s.update(1, gcState{A: true}, time.Second)
	s.remove(1)
	if len(h1.added) != 1 || len(h1.events) != 0 || len(h1.removed) != 0 {
		t.Errorf("after Stop: added %d, events %d, removed %d", len(h1.added), len(h1.events), len(h1.removed))
	}
	// The next Start reports the controllers that are connected.
	h2 := &padRecorder{}
	s.start(h2)
	if len(h2.added) != 1 || h2.added[0].Name != "B" {
		t.Errorf("after the second Start: %+v", h2.added)
	}
}

func TestRumbleParams(t *testing.T) {
	tests := []struct {
		low, high            uint16
		intensity, sharpness float32
	}{
		{0, 0, 0, 0},
		{65535, 0, 1, 0},
		{0, 65535, 1, 1},
		{65535, 65535, 1, 0.5},
		{32768, 0, 32768.0 / 65535, 0},
		{16384, 49152, 49152.0 / 65535, 0.75},
	}
	for _, tc := range tests {
		i, s := rumbleParams(tc.low, tc.high)
		if i != tc.intensity || s != tc.sharpness {
			t.Errorf("rumbleParams(%d, %d) = %v, %v; want %v, %v", tc.low, tc.high, i, s, tc.intensity, tc.sharpness)
		}
	}
}
