package ios

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestParseDevEvents(t *testing.T) {
	got, err := ParseDevEvents("wait:100ms; tap:10,20 ;drag:1,2,3,4; drag:5,6,7,8,64ms;key:KeyC,meta;key:Enter;key:ArrowLeft,shift+alt")
	if err != nil {
		t.Fatal(err)
	}
	want := []DevEvent{
		{Kind: "wait", Wait: 100 * time.Millisecond},
		{Kind: "tap", X: 10, Y: 20},
		{Kind: "drag", X: 1, Y: 2, X2: 3, Y2: 4, Wait: 200 * time.Millisecond},
		{Kind: "drag", X: 5, Y: 6, X2: 7, Y2: 8, Wait: 64 * time.Millisecond},
		{Kind: "key", Key: "KeyC", Mods: "meta"},
		{Kind: "key", Key: "Enter"},
		{Kind: "key", Key: "ArrowLeft", Mods: "shift+alt"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v\nwant     %+v", got, want)
	}
}

func TestParseTextDevEvents(t *testing.T) {
	got, err := ParseDevEvents("text: a 😀;compose:に,にほ,日本;composing:x;keyboard:300;keyboardhide")
	if err != nil {
		t.Fatal(err)
	}
	want := []DevEvent{
		{Kind: "text", Text: " a 😀"},
		{Kind: "compose", Steps: []string{"に", "にほ", "日本"}},
		{Kind: "composing", Steps: []string{"x"}},
		{Kind: "keyboard", Height: 300},
		{Kind: "keyboardhide"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v\nwant     %+v", got, want)
	}
}

func TestParseDevEventsErrors(t *testing.T) {
	for _, script := range []string{
		"tap", "wait:soon", "wait:-1s", "tap:1", "tap:a,b", "drag:1,2,3", "drag:1,2,3,4,0s", "drag:1,2,3,4,x", "key:Nope", "key:KeyA,hyper", "jump:1,2", "text:", "compose:", "keyboard:", "keyboard:0", "keyboard:tall",
	} {
		_, err := ParseDevEvents(script)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != TagIOSDevEventsInvalid || !appErr.HasRoute("ParseDevEvents."+TagIOSDevEventsInvalid) {
			t.Errorf("%q: err = %v", script, err)
		}
	}
}

func TestDevKeyCodes(t *testing.T) {
	for name, want := range map[string]uint32{"KeyA": 0x04, "KeyZ": 0x1d, "Digit1": 0x1e, "Digit9": 0x26, "Digit0": 0x27, "F1": 0x3a, "F12": 0x45, "Enter": 0x28} {
		if got := devKeyCodes[name]; got != want {
			t.Errorf("%s = 0x%x, want 0x%x", name, got, want)
		}
	}
	// The codes agree with the physical key table: KeyA is the USB usage 0x70004.
	if physicalKeyForCode(devKeyCodes["KeyA"]) != 0x70004 || physicalKeyForCode(devKeyCodes["ArrowUp"]) != 0x70052 {
		t.Error("the key codes of a script are not the ones of the physical key table")
	}
	if devKeyCharacters("KeyA", true) != "A" || devKeyCharacters("KeyA", false) != "a" || devKeyCharacters("Digit5", false) != "5" || devKeyCharacters("Space", false) != " " || devKeyCharacters("F1", false) != "" {
		t.Error("the characters of a key are wrong")
	}
}
