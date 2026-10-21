package web

import (
	"reflect"
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
		{"wait", "wait:500ms", []DevEvent{{Kind: "wait", Wait: 500 * time.Millisecond}}},
		{"wheel", "wheel:240,320,0,300", []DevEvent{{Kind: "wheel", X: 240, Y: 320, DY: 300}}},
		{"tap", "tap: 10.5 , 20", []DevEvent{{Kind: "tap", X: 10.5, Y: 20}}},
		{"move", "move:3,4", []DevEvent{{Kind: "move", X: 3, Y: 4}}},
		{"key", "key:KeyK,k", []DevEvent{{Kind: "key", Code: "KeyK", Key: "k"}}},
		{"input", "input:Hello, world", []DevEvent{{Kind: "input", Text: "Hello, world"}}},
		{"input with a line break", `input:one\ntwo`, []DevEvent{{Kind: "input", Text: "one\ntwo"}}},
		{"paste", "paste:copied text", []DevEvent{{Kind: "paste", Text: "copied text"}}},
		{"compose", "compose:n,ni,你", []DevEvent{{Kind: "compose", Steps: []string{"n", "ni", "你"}}}},
		{"several, with a trailing separator", "wait:1s;wheel:1,2,3,4;;key:Enter,Enter;", []DevEvent{
			{Kind: "wait", Wait: time.Second}, {Kind: "wheel", X: 1, Y: 2, DX: 3, DY: 4}, {Kind: "key", Code: "Enter", Key: "Enter"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevEvents(tt.script)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseDevEventsFailures(t *testing.T) {
	for _, script := range []string{"wait", "wait:soon", "wait:-1s", "wheel:1,2,3", "wheel:1,2,3,x", "tap:1", "key:Enter", "key:,k", "jump:1,2", "input:", "paste:", "compose:", "compose:a,,b"} {
		t.Run(script, func(t *testing.T) {
			_, err := ParseDevEvents(script)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != "WebDevEventsInvalid" || !appErr.HasRoute("ParseDevEvents.WebDevEventsInvalid") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestIsFirefox(t *testing.T) {
	tests := []struct {
		ua   string
		want bool
	}{
		{"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0", true},
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsFirefox(tt.ua); got != tt.want {
			t.Errorf("IsFirefox(%q) = %v, want %v", tt.ua, got, tt.want)
		}
	}
}
