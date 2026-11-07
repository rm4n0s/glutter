package gamepad_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/widgets"
)

// The memory tests of the gamepads (step 31.1, SPEC §15 "Memory over time"): pads that come and go with listeners on them,
// raw pads that go through the table of mappings, pads that have no mapping, and the same mappings added over and over. The
// live heap and the goroutines must not grow, and nothing is left in the set of pads.

func leakRig(t *testing.T) (*gluttertest.Tester, *gamepad.Gamepads) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{DiscardScenes: true})
	ts.PumpWidget(widgets.NewSizedBox(nil))
	return ts, gamepad.ForBinding(ts.Binding)
}

func TestLeaksPadsComeAndGo(t *testing.T) {
	tests := []struct {
		name string
		spec gluttertest.PadSpec
		step func(ts *gluttertest.Tester, tp *gluttertest.TestPad)
	}{
		{"standard", gluttertest.PadSpec{Name: "Standard", Rumble: true}, func(ts *gluttertest.Tester, tp *gluttertest.TestPad) {
			tp.Press(gamepad.South)
			tp.SetAxis(gamepad.LeftX, 0.5)
			tp.SetAxis(gamepad.RightTrigger, 1)
			tp.Release(gamepad.South)
		}},
		{"raw with a mapping from the table", gluttertest.PadSpec{Name: "Raw", GUID: "030000004c050000e60c000011810000", Axes: 6, Buttons: 15, Hats: 1},
			func(ts *gluttertest.Tester, tp *gluttertest.TestPad) {
				tp.RawButton(0, true)
				tp.RawAxis(0, 0)
				tp.RawAxis(0, 20000)
				tp.RawHat(0, 1)
				tp.RawButton(0, false)
			}},
		{"raw with the mapping of the backend", gluttertest.PadSpec{Name: "Made", GUID: "03000000eeee00000009000001000000", Axes: 2, Buttons: 2, Mapping: "a:b0,b:b1,leftx:a0,lefty:a1,"},
			func(ts *gluttertest.Tester, tp *gluttertest.TestPad) {
				tp.RawButton(1, true)
				tp.RawButton(1, false)
			}},
		{"raw with no mapping", gluttertest.PadSpec{Name: "Nothing", GUID: "03000000ffff00000009000001000000", Axes: 2, Buttons: 2},
			func(ts *gluttertest.Tester, tp *gluttertest.TestPad) { tp.RawButton(0, true) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, pads := leakRig(t)
			notified := 0
			pads.AddListener(func() { notified++ })
			leaktest.Run(t, "gamepad/"+tt.name, func() func() {
				return func() {
					tp := ts.ConnectPad(tt.spec)
					if pad := tp.Pad(); pad != nil {
						remove := pad.AddListener(func(gamepad.Event) {})
						defer remove()
						pad.State()
					}
					tt.step(ts, tp)
					tp.Disconnect()
					if n := len(pads.Pads()); n != 0 {
						t.Fatalf("%d pads are left after one was removed", n)
					}
				}
			}, leaktest.Opts{N: 200, Warmup: 20, Counters: map[string]func() int{"pads": func() int { return len(pads.Pads()) }}})
			if tt.name != "raw with no mapping" && notified == 0 {
				t.Error("the set of pads never told its listeners")
			}
		})
	}
}

func TestLeaksMappingsAddedAgain(t *testing.T) {
	_, pads := leakRig(t)
	const lines = "03000000eeee00000009000001000000,Same,a:b0,b:b1,leftx:a0,lefty:a1,platform:Linux,\n" +
		"03000000eeee0000000a000001000000,Other,a:b2,b:b3,platform:Linux,\n"
	leaktest.Run(t, "gamepad/mappings", func() func() {
		return func() {
			if _, err := pads.AddMappings(lines); err != nil {
				t.Fatal(err)
			}
		}
	}, leaktest.Opts{N: 200, Warmup: 20})
}

// TestLeaksAThousandPadsComeAndGo connects and removes a pad 1,000 times (and 4,000 in the second run), each with a listener, a press,
// an axis and a rumble request (step 31.11, SPEC §28.6): the heap and the goroutines stay flat and no pad is left.
func TestLeaksAThousandPadsComeAndGo(t *testing.T) {
	ts, pads := leakRig(t)
	spec := gluttertest.PadSpec{Name: "Standard", Rumble: true}
	leaktest.Run(t, "gamepad/a thousand pads", func() func() {
		return func() {
			tp := ts.ConnectPad(spec)
			remove := tp.Pad().AddListener(func(gamepad.Event) {})
			tp.Press(gamepad.South)
			tp.SetAxis(gamepad.LeftX, 0.5)
			if err := tp.Pad().Rumble(0.5, 0.5, 10*time.Millisecond); err != nil {
				t.Fatal(err)
			}
			tp.Release(gamepad.South)
			remove()
			tp.Disconnect()
			ts.ClearLogs()
			if n := len(pads.Pads()); n != 0 {
				t.Fatalf("%d pads are left after one was removed", n)
			}
		}
	}, leaktest.Opts{N: 1000, Warmup: 50, Counters: map[string]func() int{"pads": func() int { return len(pads.Pads()) }}})
}
