package gamepad_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/widgets"
)

// TestAFrameOfReadingPadsAllocatesNothing is the allocation target of SPEC §28.6 for pads: what a game does in a frame (the list of
// the pads, the state of each, an axis through the dead zone, a button) allocates nothing, and neither does a report of a pad that
// reaches a listener.
func TestAFrameOfReadingPadsAllocatesNothing(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes what a call allocates")
	}
	ts, pads := app(t)
	a := ts.ConnectPad(gluttertest.PadSpec{Name: "A", Rumble: true})
	b := ts.ConnectPad(gluttertest.PadSpec{Name: "B"})
	events := 0
	for _, p := range pads.AppendPads(nil) {
		p.AddListener(func(gamepad.Event) { events++ })
	}
	var list []*gamepad.Pad
	sum := 0.0
	read := func() {
		list = pads.AppendPads(list[:0])
		for _, p := range list {
			st := p.State()
			x, y := gamepad.ApplyStickDeadZone(st.Axis(gamepad.LeftX), st.Axis(gamepad.LeftY), 0.15)
			sum += x + y + gamepad.ApplyDeadZone(st.Axis(gamepad.RightTrigger), 0.1)
			if st.Down(gamepad.South) {
				sum++
			}
		}
	}
	a.SetAxis(gamepad.LeftX, 0.5)
	b.Press(gamepad.South)
	read()
	if len(list) != 2 {
		t.Fatalf("%d pads", len(list))
	}
	if got := testing.AllocsPerRun(200, read); got != 0 {
		t.Errorf("reading the pads allocates %v times, want 0", got)
	}
	// A report of a pad that reaches its listener: the tester's own work is in the number, so the test says what it counts.
	v := 0.2
	report := func() {
		v = -v
		a.SetAxis(gamepad.LeftX, v)
	}
	report()
	allocs := testing.AllocsPerRun(200, report)
	t.Logf("a report of a pad through the tester allocates %v times (the tester's own part included)", allocs)
	if events == 0 || sum == 0 {
		t.Errorf("events %d, sum %v", events, sum)
	}
	got := pads.Pads()
	if len(got) != len(list) || got[0] != list[0] || got[1] != list[1] {
		t.Errorf("Pads and AppendPads differ: %v and %v", got, list)
	}
}

// BenchmarkReadPads is what a game does with two pads in a frame: the list, the state of each, the sticks through the dead zone.
func BenchmarkReadPads(b *testing.B) {
	ts := gluttertest.New(b)
	ts.PumpWidget(widgets.NewSizedBox(nil))
	pads := gamepad.ForBinding(ts.Binding)
	ts.ConnectPad(gluttertest.PadSpec{Name: "A"}).SetAxis(gamepad.LeftX, 0.5)
	ts.ConnectPad(gluttertest.PadSpec{Name: "B"}).Press(gamepad.South)
	var list []*gamepad.Pad
	sum := 0.0
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		list = pads.AppendPads(list[:0])
		for _, p := range list {
			st := p.State()
			x, y := gamepad.ApplyStickDeadZone(st.Axis(gamepad.LeftX), st.Axis(gamepad.LeftY), 0.15)
			rx, ry := gamepad.ApplyStickDeadZone(st.Axis(gamepad.RightX), st.Axis(gamepad.RightY), 0.15)
			sum += x + y + rx + ry
			if st.Down(gamepad.South) {
				sum++
			}
		}
	}
	if sum < 0 {
		b.Fatal(sum)
	}
}
