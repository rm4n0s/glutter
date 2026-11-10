package ios

import (
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// DevEventsEnv is the environment variable with the script of synthetic touches and keys that debug builds replay after the first
// frame, so that a proof screenshot or a Simulator test can show an app after a tap, a drag or a key: the headless Simulator has no way
// to send a touch. The events go through the same conversion as the real ones (SPEC §30.4). It is read only in debug builds
// (foundation.DebugMode).
//
// Flutter: no counterpart
const DevEventsEnv = "GLUTTER_DEV_EVENTS"

// ChromeLogLine is the message of the line that the dev events step "chrome" logs.
//
// Flutter: no counterpart
const ChromeLogLine = "ios chrome"

// A11yLogLines are the messages of the lines that the dev events steps "a11y" and "a11y-activate" log.
//
// Flutter: no counterpart
const (
	A11yElementsLogLine = "ios a11y elements"
	A11yElementLogLine  = "ios a11y element"
	A11yActivateLogLine = "ios a11y activate"
)

// TagIOSDevEventsInvalid is the tag of the error of a GLUTTER_DEV_EVENTS script that cannot be read.
//
// Flutter: no counterpart
const TagIOSDevEventsInvalid = "IOSDevEventsInvalid"

// DevEvent is one step of a dev events script.
//
// Flutter: no counterpart
type DevEvent struct {
	// Kind is the name of the step: "wait", "tap", "drag", "key", "text", "compose", "composing", "keyboard", "keyboardhide", "chrome", "a11y" or "a11y-activate".
	Kind string
	// Wait is how long to wait, for "wait", and how long a "drag" takes.
	Wait time.Duration
	// Text is the text of a "text" step, and Steps are the pieces of a "compose" or "composing" step, each the marked text of an update of
	// the input method.
	Text  string
	Steps []string
	// Height is the height in points of the keyboard of a "keyboard" step.
	Height float64
	// X, Y are the place of a "tap" and the start of a "drag", in points from the top left of the view; X2, Y2 are the end of a "drag".
	X, Y, X2, Y2 float64
	// Key is the physical key of a "key" step, in Flutter's names (KeyA, Digit1, Enter, ArrowLeft), and Mods the modifiers that are down while
	// it is pressed, a "+"-joined subset of shift, control, alt and meta.
	Key, Mods string
}

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments".
//
//	wait:500ms            waits
//	tap:x,y               touches the place (points from the top left of the view) and lifts the finger
//	drag:x,y,x2,y2[,d]    puts a finger at the first place, moves it to the second in steps of 16 ms over the duration d (200ms when it is
//	                      left out) and lifts it
//	text:Hello            types the text as a keyboard does that composes nothing: one insertText: for each character (the blanks are
//	                      typed; the control characters type nothing; no ";")
//	compose:a,b,c         an input method composes a, then b, then c (setMarkedText: each), commits c (insertText:) and ends the composition
//	composing:a,b,c       the same, but the composition stays (a later compose: goes on with it)
//	keyboard:h            the soft keyboard comes up, h points high (UIKit's frame notification, with a keyboard's duration)
//	keyboardhide          the soft keyboard goes down
//	a11y                  logs the accessibility elements of the first window as UIKit has them: the line "ios a11y elements" with their
//	                      number and a line "ios a11y element" for each, with its label, value, hint, traits and frame
//	a11y-activate:label   activates the element whose label is the text, as VoiceOver's double tap does (accessibilityActivate), and logs the
//	                      line "ios a11y activate" with the answer
//	chrome                logs what UIKit has of the system UI of the window (the status bar's style and visibility, the home indicator,
//	                      the supported orientations), as the line "ios chrome"
//	key:Name[,mods]       presses and releases a key of a hardware keyboard with the modifiers (shift, control, alt, meta, joined by "+")
//	                      pressed before it and released after it: key:KeyC,meta is Command+C. The names are Flutter's physical keys:
//	                      KeyA to KeyZ, Digit0 to Digit9, Enter, Tab, Space, Backspace, Delete, Escape, ArrowLeft, ArrowRight, ArrowUp,
//	                      ArrowDown, Home, End, PageUp, PageDown, and F1 to F12
//
// The text steps need a text field with the focus (the input view is the first responder). A step that is not known, or has the wrong
// arguments, fails with IOSDevEventsInvalid.
//
// Flutter: no counterpart
func ParseDevEvents(script string) ([]DevEvent, error) {
	var events []DevEvent
	for _, raw := range strings.Split(script, ";") {
		step := strings.TrimSpace(raw)
		if step == "" {
			continue
		}
		if step == "keyboardhide" || step == "chrome" || step == "a11y" {
			events = append(events, DevEvent{Kind: step})
			continue
		}
		name, args, ok := strings.Cut(step, ":")
		if !ok {
			return nil, errors.New(TagIOSDevEventsInvalid, "a step must be name:arguments", "step", step)
		}
		switch name {
		case "a11y-activate":
			events = append(events, DevEvent{Kind: name, Text: args})
		case "wait":
			d, err := time.ParseDuration(args)
			if err != nil || d < 0 {
				return nil, errors.New(TagIOSDevEventsInvalid, "wait takes a duration such as 500ms", "step", step)
			}
			events = append(events, DevEvent{Kind: name, Wait: d})
		case "tap":
			v, ok := parseFloats(args, 2)
			if !ok {
				return nil, errors.New(TagIOSDevEventsInvalid, "tap takes x,y", "step", step)
			}
			events = append(events, DevEvent{Kind: name, X: v[0], Y: v[1]})
		case "drag":
			parts := strings.Split(args, ",")
			d := 200 * time.Millisecond
			if len(parts) == 5 {
				var err error
				if d, err = time.ParseDuration(strings.TrimSpace(parts[4])); err != nil || d <= 0 {
					return nil, errors.New(TagIOSDevEventsInvalid, "the duration of a drag must be positive, such as 200ms", "step", step)
				}
				parts = parts[:4]
			}
			v, ok := parseFloats(strings.Join(parts, ","), 4)
			if !ok {
				return nil, errors.New(TagIOSDevEventsInvalid, "drag takes x,y,x2,y2 and an optional duration", "step", step)
			}
			events = append(events, DevEvent{Kind: name, X: v[0], Y: v[1], X2: v[2], Y2: v[3], Wait: d})
		case "key":
			key, mods, _ := strings.Cut(args, ",")
			key, mods = strings.TrimSpace(key), strings.TrimSpace(mods)
			if _, ok := devKeyCodes[key]; !ok {
				return nil, errors.New(TagIOSDevEventsInvalid, "the key is not one of Flutter's physical keys that a script can press", "step", step)
			}
			for _, m := range strings.Split(mods, "+") {
				if _, ok := devModifiers[m]; m != "" && !ok {
					return nil, errors.New(TagIOSDevEventsInvalid, "a modifier must be shift, control, alt or meta", "step", step)
				}
			}
			events = append(events, DevEvent{Kind: name, Key: key, Mods: mods})
		case "text":
			// The blanks of the text are the text: only the step's own leading blanks go.
			_, args, _ = strings.Cut(strings.TrimLeft(raw, " \t\r\n"), ":")
			if args == "" {
				return nil, errors.New(TagIOSDevEventsInvalid, "text takes the text to type", "step", step)
			}
			events = append(events, DevEvent{Kind: name, Text: args})
		case "compose", "composing":
			if strings.TrimSpace(args) == "" {
				return nil, errors.New(TagIOSDevEventsInvalid, "compose takes the marked texts, separated by commas", "step", step)
			}
			events = append(events, DevEvent{Kind: name, Steps: strings.Split(args, ",")})
		case "keyboard":
			h, err := strconv.ParseFloat(strings.TrimSpace(args), 64)
			if err != nil || h <= 0 {
				return nil, errors.New(TagIOSDevEventsInvalid, "keyboard takes the height in points", "step", step)
			}
			events = append(events, DevEvent{Kind: name, Height: h})
		default:
			return nil, errors.New(TagIOSDevEventsInvalid, "the step is not one of wait, tap, drag, key, text, compose, composing, keyboard and keyboardhide", "step", step)
		}
	}
	return events, nil
}

func parseFloats(s string, n int) ([]float64, bool) {
	parts := strings.Split(s, ",")
	if len(parts) != n {
		return nil, false
	}
	out := make([]float64, n)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

// devModifiers are the key codes of the modifiers of a script, and their flags, in the order they are pressed.
var devModifiers = map[string]struct {
	code uint32
	flag uint64
}{
	"shift": {keyLeftShift, flagShiftAny}, "control": {keyLeftControl, flagControlAny}, "alt": {keyLeftAlt, flagAltAny}, "meta": {keyLeftGUI, flagMetaAny},
}

// devModifierOrder is the order in which the modifiers of a step are pressed.
var devModifierOrder = []string{"shift", "control", "alt", "meta"}

// devKeyCodes are the key codes (UIKeyboardHIDUsage) of the keys a script presses, by Flutter's physical key names.
var devKeyCodes = func() map[string]uint32 {
	m := map[string]uint32{
		"Enter": 0x28, "Escape": 0x29, "Backspace": 0x2a, "Tab": 0x2b, "Space": 0x2c, "Delete": 0x4c,
		"Home": 0x4a, "PageUp": 0x4b, "End": 0x4d, "PageDown": 0x4e,
		"ArrowRight": 0x4f, "ArrowLeft": 0x50, "ArrowDown": 0x51, "ArrowUp": 0x52,
	}
	for i := 0; i < 26; i++ {
		m["Key"+string(rune('A'+i))] = 0x04 + uint32(i)
	}
	for i := 1; i <= 9; i++ {
		m["Digit"+strconv.Itoa(i)] = 0x1d + uint32(i)
	}
	m["Digit0"] = 0x27
	for i := 1; i <= 12; i++ {
		m["F"+strconv.Itoa(i)] = 0x39 + uint32(i)
	}
	return m
}()

// devKeyCharacters is the text that a key makes: the letter (upper case with Shift), the digit, a blank, and the control characters.
func devKeyCharacters(key string, shift bool) string {
	switch {
	case strings.HasPrefix(key, "Key") && len(key) == 4:
		if shift {
			return key[3:]
		}
		return strings.ToLower(key[3:])
	case strings.HasPrefix(key, "Digit"):
		return key[5:]
	}
	switch key {
	case "Space":
		return " "
	case "Enter":
		return "\r"
	case "Tab":
		return "\t"
	}
	return ""
}
