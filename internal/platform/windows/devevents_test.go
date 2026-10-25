package windows

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
		{"only separators", " ; ;", nil},
		{"a wait", "wait:500ms", []DevEvent{{Kind: "wait", Wait: 500 * time.Millisecond}}},
		{"several waits with blanks", " wait:1s ; wait:20ms;", []DevEvent{{Kind: "wait", Wait: time.Second}, {Kind: "wait", Wait: 20 * time.Millisecond}}},
		{"zero is allowed", "wait:0s", []DevEvent{{Kind: "wait"}}},
		{"a move", "move:240,400", []DevEvent{{Kind: "move", X: 240, Y: 400}}},
		{"a tap with blanks and decimals", "tap: 12.5 , 7", []DevEvent{{Kind: "tap", X: 12.5, Y: 7}}},
		{"a wheel", "wheel:240,400,0,300", []DevEvent{{Kind: "wheel", X: 240, Y: 400, DY: 300}}},
		{"a wheel to the left and up", "wheel:1,2,-30,-40", []DevEvent{{Kind: "wheel", X: 1, Y: 2, DX: -30, DY: -40}}},
		{"a letter", "key:a", []DevEvent{{Kind: "key", Keys: []uint32{'A'}}}},
		{"a digit", "key:7", []DevEvent{{Kind: "key", Keys: []uint32{'7'}}}},
		{"a name with the prefix", "key:VK_LEFT", []DevEvent{{Kind: "key", Keys: []uint32{0x25}}}},
		{"a name in lower case", "key:return", []DevEvent{{Kind: "key", Keys: []uint32{0x0d}}}},
		{"an alias", "key:ctrl+a", []DevEvent{{Kind: "key", Keys: []uint32{0x11, 'A'}}}},
		{"Alt, by name and by alias", "key:menu;key:ALT", []DevEvent{{Kind: "key", Keys: []uint32{0x12}}, {Kind: "key", Keys: []uint32{0x12}}}},
		{"the sides of Alt", "key:LMENU+RMENU", []DevEvent{{Kind: "key", Keys: []uint32{0xa4, 0xa5}}}},
		{"a combination", "key:SHIFT+LEFT", []DevEvent{{Kind: "key", Keys: []uint32{0x10, 0x25}}}},
		{"a key held and released", "keydown:SHIFT;key:RIGHT;keyup:SHIFT", []DevEvent{
			{Kind: "keydown", Keys: []uint32{0x10}}, {Kind: "key", Keys: []uint32{0x27}}, {Kind: "keyup", Keys: []uint32{0x10}}}},
		{"page keys and a function key", "key:PAGEDOWN;key:pageup;key:F10", []DevEvent{
			{Kind: "key", Keys: []uint32{0x22}}, {Kind: "key", Keys: []uint32{0x21}}, {Kind: "key", Keys: []uint32{0x79}}}},
		{"all the steps in a script", "wait:1ms;move:1,2;tap:3,4;wheel:5,6,7,8;wait:2ms", []DevEvent{
			{Kind: "wait", Wait: time.Millisecond}, {Kind: "move", X: 1, Y: 2}, {Kind: "tap", X: 3, Y: 4},
			{Kind: "wheel", X: 5, Y: 6, DX: 7, DY: 8}, {Kind: "wait", Wait: 2 * time.Millisecond}}},
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
				if !reflect.DeepEqual(got[i], tt.want[i]) {
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
		{"a step of step 28.5", "text:abc", "text:abc"},
		{"a key that is not known", "key:VK_NOPE", "key:VK_NOPE"},
		{"a key with nothing", "key:", "key:"},
		{"a combination with an empty key", "key:SHIFT+", "key:SHIFT+"},
		{"keydown with two keys", "keydown:SHIFT+A", "keydown:SHIFT+A"},
		{"keyup with a name that is not a key", "keyup:?", "keyup:?"},
		{"a move with one number", "move:1", "move:1"},
		{"a move with three numbers", "move:1,2,3", "move:1,2,3"},
		{"a tap that is not a number", "tap:a,b", "tap:a,b"},
		{"a tap with an empty number", "tap:1,", "tap:1,"},
		{"a wheel with two numbers", "wheel:1,2", "wheel:1,2"},
		{"a wheel with an infinity", "wheel:1,2,3,Inf", "wheel:1,2,3,Inf"},
		{"a move with NaN", "move:NaN,1", "move:NaN,1"},
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
