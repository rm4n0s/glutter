package padstd

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

func TestTrackerReportsChangesOnly(t *testing.T) {
	tr := NewTracker(7)
	var s Snapshot
	if got := tr.Update(time.Second, &s, nil); len(got) != 0 {
		t.Fatalf("a pad at rest reported %+v", got)
	}
	s.Buttons[gamepaddb.South] = true
	s.Buttons[gamepaddb.DPadLeft] = true
	s.Axes[gamepaddb.LeftY] = -1000
	s.Axes[gamepaddb.RightTrigger] = 32767
	got := tr.Update(2*time.Second, &s, nil)
	want := []platform.PadEvent{
		{Pad: 7, Time: 2 * time.Second, Kind: platform.PadButton, Index: int(gamepaddb.South), Value: 1},
		{Pad: 7, Time: 2 * time.Second, Kind: platform.PadButton, Index: int(gamepaddb.DPadLeft), Value: 1},
		{Pad: 7, Time: 2 * time.Second, Kind: platform.PadAxis, Index: int(gamepaddb.LeftY), Value: -1000},
		{Pad: 7, Time: 2 * time.Second, Kind: platform.PadAxis, Index: int(gamepaddb.RightTrigger), Value: 32767},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if got := tr.Update(3*time.Second, &s, nil); len(got) != 0 {
		t.Errorf("nothing changed but %+v was reported", got)
	}
	s.Buttons[gamepaddb.South] = false
	got = tr.Update(4*time.Second, &s, got[:0])
	if len(got) != 1 || got[0].Value != 0 || got[0].Index != int(gamepaddb.South) {
		t.Errorf("the release = %+v", got)
	}
}

func TestScaling(t *testing.T) {
	tests := []struct {
		name string
		got  int16
		want int16
	}{
		{"trigger 0", Trigger8(0), 0},
		{"trigger 255", Trigger8(255), 32767},
		{"trigger 128", Trigger8(128), 16447},
		{"unit below", Unit(-1), 0},
		{"unit nan", Unit(nanValue()), 0},
		{"unit half", Unit(0.5), 16383},
		{"unit above", Unit(2), 32767},
		{"signed min", Signed(-1), -32768},
		{"signed beyond", Signed(-5), -32768},
		{"signed max", Signed(1), 32767},
		{"signed zero", Signed(0), 0},
		{"signed half down", Signed(-0.5), -16384},
		{"flip", Flip(100), -100},
		{"flip min", Flip(-32768), 32767},
		{"flip max", Flip(32767), -32767},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("= %d, want %d", tt.got, tt.want)
			}
		})
	}
}

func nanValue() float64 {
	zero := 0.0
	return zero / zero
}
