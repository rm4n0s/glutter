package pads

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// TestReportsOfAPadAllocateNothing is the allocation target of SPEC §28.6 for the path that a report takes: the backend's events go
// through the mapping layer to the pad's state, to its listeners and to the key events, and none of it allocates.
func TestReportsOfAPadAllocateNothing(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes what a call allocates")
	}
	for _, withKeys := range []bool{false, true} {
		name := "without key events"
		if withKeys {
			name = "with key events"
		}
		t.Run(name, func(t *testing.T) {
			r := newRig(t, withKeys)
			id := r.connect(standard("Pad"))
			pad := r.m.Pads()[0]
			got := 0
			pad.AddListener(func(Event) { got++ })
			batch := []platform.PadEvent{
				{Pad: id, Kind: platform.PadButton, Index: int(gamepaddb.South), Value: 1},
				{Pad: id, Kind: platform.PadAxis, Index: int(gamepaddb.LeftX), Value: 12000},
				{Pad: id, Kind: platform.PadButton, Index: int(gamepaddb.DPadDown), Value: 1},
				{Pad: id, Kind: platform.PadAxis, Index: int(gamepaddb.LeftTrigger), Value: 30000},
			}
			flip := func() {
				for i := range batch {
					switch batch[i].Kind {
					case platform.PadButton:
						batch[i].Value = 1 - batch[i].Value
					default:
						batch[i].Value = -batch[i].Value
					}
				}
				r.m.PadEvents(batch)
			}
			flip()
			r.keys = r.keys[:0]
			if allocs := testing.AllocsPerRun(200, func() {
				flip()
				r.keys = r.keys[:0]
			}); allocs != 0 {
				t.Errorf("a batch of reports allocates %v times, want 0", allocs)
			}
			if got == 0 {
				t.Error("the listener heard nothing")
			}
		})
	}
}
