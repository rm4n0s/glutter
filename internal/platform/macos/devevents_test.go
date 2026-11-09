package macos

import (
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestParseDevEvents(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		want    []DevEvent
		wantTag string
	}{
		{"empty", "", nil, ""},
		{"a wait", "wait:500ms", []DevEvent{{Kind: "wait", Wait: 500 * time.Millisecond}}, ""},
		{"blanks and empty steps", " wait:1s ; ;wait:2ms", []DevEvent{{Kind: "wait", Wait: time.Second}, {Kind: "wait", Wait: 2 * time.Millisecond}}, ""},
		{"no colon", "wait", nil, TagMacOSDevEventsInvalid},
		{"a bad duration", "wait:soon", nil, TagMacOSDevEventsInvalid},
		{"a negative duration", "wait:-1s", nil, TagMacOSDevEventsInvalid},
		{"a step that comes later", "tap:1,2", nil, TagMacOSDevEventsInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevEvents(tt.script)
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantTag {
					t.Fatalf("err = %v, want tag %s", err, tt.wantTag)
				}
				if !appErr.HasRoute("ParseDevEvents." + tt.wantTag) {
					t.Errorf("route = %s", appErr.Route())
				}
				return
			}
			if err != nil || !slices.Equal(got, tt.want) {
				t.Errorf("got %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}
