package windows

import (
	"slices"
	"testing"
)

func TestModalGate(t *testing.T) {
	type op struct {
		block bool
	}
	tests := []struct {
		name string
		ops  []op
		want []bool // the calls of set, in order
	}{
		{"one dialog", []op{{true}, {false}}, []bool{false, true}},
		{"two dialogs: the window is enabled when the last closes", []op{{true}, {true}, {false}, {false}}, []bool{false, true}},
		{"two dialogs, one closes, the window stays disabled", []op{{true}, {true}, {false}}, []bool{false}},
		{"a close with no dialog does nothing", []op{{false}}, nil},
		{"the window can be blocked again", []op{{true}, {false}, {true}, {false}}, []bool{false, true, false, true}},
		{"one close too many", []op{{true}, {false}, {false}}, []bool{false, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g modalGate
			var calls []bool
			set := func(enabled bool) { calls = append(calls, enabled) }
			for _, o := range tt.ops {
				if o.block {
					g.block(set)
				} else {
					g.unblock(set)
				}
			}
			if !slices.Equal(calls, tt.want) {
				t.Errorf("set calls = %v, want %v", calls, tt.want)
			}
		})
	}
	var g modalGate
	g.block(func(bool) {})
	if !g.blocked() {
		t.Error("blocked() is false with a dialog open")
	}
	g.unblock(func(bool) {})
	if g.blocked() {
		t.Error("blocked() is true with no dialog")
	}
}
