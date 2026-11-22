//go:build linux && !android

package x11

import "testing"

func TestStartupTimeIsTheTimeOfTheStartupID(t *testing.T) {
	tests := []struct {
		id   string
		want uint32
	}{
		{"theatro-2-1234-host_TIME6000", 6000},
		{"gnome-shell/Theatro/5432-0-host_TIME98765", 98765},
		{"a_TIME1_TIME42", 42},           // the last _TIME counts
		{"no-time-in-it", 0},             // a launcher may leave it out
		{"", 0},                          // no token
		{"x_TIME", 0},                    // no number
		{"x_TIME12abc", 0},               // not a number
		{"x_TIME4294967296", 0},          // more than 32 bits
		{"x_TIME4294967295", 4294967295}, // the largest time
	}
	for _, tt := range tests {
		if got := startupTime(tt.id); got != tt.want {
			t.Errorf("startupTime(%q) = %d, want %d", tt.id, got, tt.want)
		}
	}
}
