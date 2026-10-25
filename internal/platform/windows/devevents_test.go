package windows

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestParseDevEvents(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   []DevEvent
	}{
		{"empty", "", nil},
		{"only separators", " ; ;", nil},
		{"a wait", "wait:500ms", []DevEvent{{Kind: "wait", Wait: 500 * time.Millisecond}}},
		{"several waits with blanks", " wait:1s ; wait:20ms;", []DevEvent{{Kind: "wait", Wait: time.Second}, {Kind: "wait", Wait: 20 * time.Millisecond}}},
		{"zero is allowed", "wait:0s", []DevEvent{{Kind: "wait"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevEvents(tt.script)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("event %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseDevEventsFailures(t *testing.T) {
	tests := []struct {
		name, script, step string
	}{
		{"no colon", "wait", "wait"},
		{"a bad duration", "wait:soon", "wait:soon"},
		{"a negative duration", "wait:-1s", "wait:-1s"},
		{"a missing duration", "wait:", "wait:"},
		{"a step that comes later", "tap:1,2", "tap:1,2"},
		{"an unknown step", "dance:now", "dance:now"},
		{"the error is in the second step", "wait:1ms;jump:3", "jump:3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDevEvents(tt.script)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseDevEvents.WindowsDevEventsInvalid") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Tag != TagWindowsDevEventsInvalid || appErr.Metadata["step"] != tt.step {
				t.Errorf("tag %s, step %v; want %s, %q", appErr.Tag, appErr.Metadata["step"], TagWindowsDevEventsInvalid, tt.step)
			}
		})
	}
}
