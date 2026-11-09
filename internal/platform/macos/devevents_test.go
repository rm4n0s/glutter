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
		{"move", "move:10,20.5", []DevEvent{{Kind: "move", X: 10, Y: 20.5}}, ""},
		{"tap", "tap: 1 , 2", []DevEvent{{Kind: "tap", X: 1, Y: 2}}, ""},
		{"wheel", "wheel:5,6,0,-120", []DevEvent{{Kind: "wheel", X: 5, Y: 6, DX: 0, DY: -120}}, ""},
		{"move with one number", "move:1", nil, TagMacOSDevEventsInvalid},
		{"tap with a word", "tap:a,b", nil, TagMacOSDevEventsInvalid},
		{"wheel with three numbers", "wheel:1,2,3", nil, TagMacOSDevEventsInvalid},
		{"a number that is not finite", "move:NaN,1", nil, TagMacOSDevEventsInvalid},
		{"a step that comes later", "key:A", nil, TagMacOSDevEventsInvalid},
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

func TestParsePanMakesTheStepsOfAGesture(t *testing.T) {
	got, err := ParseDevEvents("pan:10,20,0,-160,64ms")
	if err != nil {
		t.Fatal(err)
	}
	// 64 ms is four intervals of 16 ms: begin, four (wait, step) pairs, a wait and the end.
	if len(got) != 1+4*2+2 {
		t.Fatalf("%d steps: %+v", len(got), got)
	}
	if got[0].Kind != "panbegin" || got[0].X != 10 || got[0].Y != 20 {
		t.Errorf("first = %+v", got[0])
	}
	var dy float64
	var waits time.Duration
	for _, e := range got {
		switch e.Kind {
		case "panstep":
			dy += e.DY
		case "wait":
			waits += e.Wait
		}
	}
	if dy != -160 {
		t.Errorf("the steps scroll %v in all, want -160", dy)
	}
	if waits != 80*time.Millisecond { // four before the steps and one before the end
		t.Errorf("the waits are %v, want 80ms", waits)
	}
	if got[len(got)-1].Kind != "panend" {
		t.Errorf("last = %+v", got[len(got)-1])
	}
	def, err := ParseDevEvents("pan:1,2,3,4")
	if err != nil || len(def) < 3 {
		t.Errorf("default duration: %v, %d steps", err, len(def))
	}
	for _, bad := range []string{"pan:1,2,3", "pan:1,2,3,4,soon", "pan:1,2,3,4,0s", "pan:1,2,3,4,-5ms", "pan:a,b,c,d"} {
		if _, err := ParseDevEvents(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}
