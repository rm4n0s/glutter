//go:build darwin && !ios

package macos

import "testing"

func TestScriptPendingFollowsTheScript(t *testing.T) {
	tests := []struct {
		name   string
		events []DevEvent
		done   bool
		want   bool
	}{
		{"no script", nil, false, false},
		{"script not run", []DevEvent{{Kind: "tap"}}, false, true},
		{"script ended", []DevEvent{{Kind: "tap"}}, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Platform{devEvents: tc.events, devDone: tc.done}
			if got := p.ScriptPending(); got != tc.want {
				t.Errorf("ScriptPending() = %v, want %v", got, tc.want)
			}
		})
	}
}
