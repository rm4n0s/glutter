package windows

import (
	"slices"
	"strings"
	"testing"
)

func TestConsolePlan(t *testing.T) {
	tests := []struct {
		name   string
		valid  bool
		parent bool
		want   consoleAction
		asked  bool // whether the parent's console was asked for
	}{
		{"valid handles keep the output", true, true, consoleKeep, false},
		{"no handles and a parent console", false, true, consoleAttach, true},
		{"no handles and no console", false, false, consoleDebugger, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asked := false
			got := consolePlan(tt.valid, func() bool { asked = true; return tt.parent })
			if got != tt.want || asked != tt.asked {
				t.Errorf("consolePlan = %v (asked %v), want %v (asked %v)", got, asked, tt.want, tt.asked)
			}
		})
	}
}

func TestValidHandle(t *testing.T) {
	tests := []struct {
		h    uintptr
		want bool
	}{{0, false}, {^uintptr(0), false}, {1, true}, {0x1c4, true}}
	for _, tt := range tests {
		if got := validHandle(tt.h); got != tt.want {
			t.Errorf("validHandle(%#x) = %v, want %v", tt.h, got, tt.want)
		}
	}
}

func TestForwardLines(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
	}{
		{"lines", "one\ntwo\n", []string{"one", "two"}},
		{"a last line with no break", "one\ntwo", []string{"one", "two"}},
		{"carriage returns", "one\r\ntwo\r\n", []string{"one", "two"}},
		{"empty lines", "\n\nx\n", []string{"", "", "x"}},
		{"nothing", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			forwardLines(strings.NewReader(tt.in), func(l string) { got = append(got, l) })
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
	// A line longer than the buffer ends the lines, and the rest is drained: the writer never blocks.
	var got []string
	r := strings.NewReader("ok\n" + strings.Repeat("x", 2<<20) + "\nlost\n")
	forwardLines(r, func(l string) { got = append(got, l) })
	if !slices.Equal(got, []string{"ok"}) || r.Len() != 0 {
		t.Errorf("got %d lines, %d bytes left unread", len(got), r.Len())
	}
}
