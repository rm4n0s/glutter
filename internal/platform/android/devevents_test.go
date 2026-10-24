package android

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
		{"tap", "tap:540,1200", []DevEvent{{Kind: "tap", X1: 540, Y1: 1200}}},
		{"swipe", "swipe:540,1800,540,600,300", []DevEvent{{Kind: "swipe", X1: 540, Y1: 1800, X2: 540, Y2: 600, Millis: 300}}},
		{"key", "key:KEYCODE_ENTER", []DevEvent{{Kind: "key", Key: "KEYCODE_ENTER"}}},
		{"several, with spaces and a trailing semicolon", " wait:1s ; tap:1.5,2 ;key:KEYCODE_VOLUME_UP;", []DevEvent{
			{Kind: "wait", Wait: time.Second}, {Kind: "tap", X1: 1.5, Y1: 2}, {Kind: "key", Key: "KEYCODE_VOLUME_UP"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevEvents(tt.script)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDevEvents(%q) = %+v, want %+v", tt.script, got, tt.want)
			}
		})
	}
}

func TestParseDevEventsErrors(t *testing.T) {
	for _, script := range []string{
		"tap", "tap:1", "tap:a,b", "tap:1,2,3", "swipe:1,2,3,4", "swipe:1,2,3,4,0", "swipe:1,2,3,4,1.5", "wait:soon", "wait:-1s",
		"key:ENTER", "key:KEYCODE_enter", "key:", "back:start", "tap:1,2;nope",
	} {
		t.Run(script, func(t *testing.T) {
			_, err := ParseDevEvents(script)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseDevEvents." + tagAndroidDevEventsInvalid) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

// The script runs once, after the first frame, in order, and the Java side gets each tap, swipe and
// key (the waits are the goroutine's own).
func TestDevEventsAreInjectedOnceInOrder(t *testing.T) {
	h := newFakeHost()
	p := newPlatform(newBridge(), h, nil)
	p.devEvents = []DevEvent{{Kind: "tap", X1: 1, Y1: 2}, {Kind: "wait", Wait: time.Millisecond}, {Kind: "key", Key: "KEYCODE_A"}}
	p.startDevEvents()
	p.startDevEvents() // a second start does nothing
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		n := len(h.injected)
		h.mu.Unlock()
		if n == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	h.mu.Lock()
	defer h.mu.Unlock()
	want := []DevEvent{{Kind: "tap", X1: 1, Y1: 2}, {Kind: "key", Key: "KEYCODE_A"}}
	if !reflect.DeepEqual(h.injected, want) {
		t.Errorf("injected %+v, want %+v", h.injected, want)
	}
}
