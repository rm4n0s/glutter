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
		{"a step that comes later", "text:abc", nil, TagMacOSDevEventsInvalid},
		{"a key", "key:KeyA", []DevEvent{{Kind: "key", Key: "KeyA"}}, ""},
		{"a key with a modifier", "key:KeyC,meta", []DevEvent{{Kind: "key", Key: "KeyC", Mods: "meta"}}, ""},
		{"modifiers are sorted and cleaned", "key: ArrowLeft , Meta + shift", []DevEvent{{Kind: "key", Key: "ArrowLeft", Mods: "shift+meta"}}, ""},
		{"all the modifiers", "key:F5,alt+control+meta+shift", []DevEvent{{Kind: "key", Key: "F5", Mods: "shift+control+alt+meta"}}, ""},
		{"a key that is not known", "key:KeyAA", nil, TagMacOSDevEventsInvalid},
		{"a lower case letter is not a name", "key:a", nil, TagMacOSDevEventsInvalid},
		{"a modifier that is not known", "key:KeyA,hyper", nil, TagMacOSDevEventsInvalid},
		{"no key", "key:", nil, TagMacOSDevEventsInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevEvents(tt.script)
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantTag {
					t.Fatalf("err = %v, want tag %s", err, tt.wantTag)
				}
				// The route is ParseDevEvents.<tag>, or goes through the helper that reads the step.
				if !appErr.HasRoute("ParseDevEvents") || !appErr.HasRoute("."+tt.wantTag) {
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

func TestDevKeyTable(t *testing.T) {
	tests := []struct {
		name        string
		code        uint16
		char, shift string
	}{
		{"KeyA", 0x00, "a", "A"},
		{"KeyZ", 0x06, "z", "Z"},
		{"KeyM", 0x2e, "m", "M"},
		{"Digit1", 0x12, "1", "!"},
		{"Digit0", 0x1d, "0", ")"},
		{"Enter", 0x24, "\r", "\r"},
		{"Space", 0x31, " ", " "},
		{"ArrowLeft", 0x7b, "\uf702", "\uf702"},
		{"F1", 0x7a, "\uf704", "\uf704"},
		{"F12", 0x6f, "\uf70f", "\uf70f"},
		{"PageDown", 0x79, "\uf72d", "\uf72d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ok := devKeyByName(tt.name)
			if !ok || k.code != tt.code || k.char != tt.char || k.shift != tt.shift {
				t.Errorf("devKeyByName(%q) = %+v, %v; want code %#x, %q, %q", tt.name, k, ok, tt.code, tt.char, tt.shift)
			}
		})
	}
	// Every key has its own key code, and every key code is one that the table of physical keys knows.
	seen := map[uint16]string{}
	for name, k := range devKeyTable {
		if other, dup := seen[k.code]; dup {
			t.Errorf("%s and %s have the key code %#x", name, other, k.code)
		}
		seen[k.code] = name
		if _, ok := keyCodeToPhysicalKey[k.code]; !ok {
			t.Errorf("%s: the key code %#x is not in the table of physical keys", name, k.code)
		}
	}
}
