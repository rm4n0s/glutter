package macos

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// DevEventsEnv is the environment variable with the script of synthetic events that debug builds replay
// after the first frame, so that a proof screenshot or a test can show an app after a click or a key
// (SPEC §29.6). It is read only in debug builds (foundation.DebugMode).
//
// Flutter: no counterpart
const DevEventsEnv = "GLUTTER_DEV_EVENTS"

// DevEvent is one step of a dev events script.
//
// Flutter: no counterpart
type DevEvent struct {
	// Kind is the name of the step: "wait", "move", "tap", "wheel", "key", or one of the pieces that "pan" is
	// made of: "panbegin", "panstep" and "panend" (text and compose come with step 33.5).
	Kind string
	// Key is the physical key of a "key" step, in Flutter's names (KeyA, Digit1, Enter, ArrowLeft), and Mods
	// the modifiers that are down while it is pressed: a "+"-joined subset of shift, control, alt and meta, in
	// that order.
	Key, Mods string
	// Wait is how long to wait, for "wait".
	Wait time.Duration
	// X and Y are the place in the first window, in logical pixels from its top left.
	X, Y float64
	// DX and DY are the scroll of a "wheel" in logical pixels (positive scrolls down and to the right, as
	// the web's deltas do), or the part of the scroll that a "panstep" does, in points of the trackpad.
	DX, DY float64
}

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments".
//
//	wait:500ms        waits
//	move:x,y          moves the mouse to the place (logical pixels of the first window, from its top left)
//	tap:x,y           moves the mouse there and clicks the left button
//	wheel:x,y,dx,dy   turns a notched wheel at the place; dx and dy are the scroll in logical pixels
//	                  (40 make a line; positive scrolls down and to the right)
//	key:Name[,mods]   presses and releases a key, with the modifiers (shift, control, alt, meta, joined by
//	                  "+") pressed before it and released after it: key:KeyC,meta is Command+C. The names
//	                  are Flutter's physical keys: KeyA to KeyZ, Digit0 to Digit9, Enter, Tab, Space,
//	                  Backspace, Delete, Escape, ArrowLeft, ArrowRight, ArrowUp, ArrowDown, Home, End,
//	                  PageUp, PageDown, and F1 to F12
//	pan:x,y,dx,dy[,d] a two-finger scroll of the trackpad at the place, with its phases: dx and dy are the
//	                  total scroll in points (positive moves the content down and to the right, as the
//	                  trackpad does), spread over the duration d (100ms when it is left out) in steps of
//	                  16 ms, followed by the end of the gesture
//
// A step that is not known, or has the wrong arguments, fails with MacOSDevEventsInvalid.
//
// Flutter: no counterpart
func ParseDevEvents(script string) ([]DevEvent, error) {
	var events []DevEvent
	for _, raw := range strings.Split(script, ";") {
		step := strings.TrimSpace(raw)
		if step == "" {
			continue
		}
		name, args, ok := strings.Cut(step, ":")
		if !ok {
			return nil, errors.New(TagMacOSDevEventsInvalid, "a step must be name:arguments", "step", step)
		}
		switch name {
		case "wait":
			d, err := time.ParseDuration(args)
			if err != nil || d < 0 {
				return nil, errors.New(TagMacOSDevEventsInvalid, "wait takes a duration such as 500ms", "step", step)
			}
			events = append(events, DevEvent{Kind: name, Wait: d})
		case "move", "tap":
			xy, ok := parseFloats(args, 2)
			if !ok {
				return nil, errors.New(TagMacOSDevEventsInvalid, "move and tap take x,y", "step", step)
			}
			events = append(events, DevEvent{Kind: name, X: xy[0], Y: xy[1]})
		case "wheel":
			v, ok := parseFloats(args, 4)
			if !ok {
				return nil, errors.New(TagMacOSDevEventsInvalid, "wheel takes x,y,dx,dy", "step", step)
			}
			events = append(events, DevEvent{Kind: name, X: v[0], Y: v[1], DX: v[2], DY: v[3]})
		case "key":
			ev, err := parseKeyStep(args, step)
			if err != nil {
				return nil, err
			}
			events = append(events, ev)
		case "pan":
			pan, err := parsePan(args, step)
			if err != nil {
				return nil, err
			}
			events = append(events, pan...)
		default:
			return nil, errors.New(TagMacOSDevEventsInvalid, "the step is not wait, move, tap, wheel, pan or key (text and compose come with step 33.5)", "step", step)
		}
	}
	return events, nil
}

// devModifiers are the modifiers of a key step, in the order they are pressed.
var devModifiers = []string{"shift", "control", "alt", "meta"}

// parseKeyStep reads "Name[,mods]".
func parseKeyStep(args, step string) (DevEvent, error) {
	name, mods, _ := strings.Cut(args, ",")
	name = strings.TrimSpace(name)
	if _, ok := devKeyByName(name); !ok {
		return DevEvent{}, errors.New(TagMacOSDevEventsInvalid, "key takes the name of a physical key, such as KeyA or ArrowLeft", "step", step)
	}
	have := map[string]bool{}
	for _, m := range strings.Split(mods, "+") {
		m = strings.ToLower(strings.TrimSpace(m))
		switch m {
		case "":
		case "shift", "control", "alt", "meta":
			have[m] = true
		default:
			return DevEvent{}, errors.New(TagMacOSDevEventsInvalid, "the modifiers of key are shift, control, alt and meta, joined by +", "step", step)
		}
	}
	var list []string
	for _, m := range devModifiers {
		if have[m] {
			list = append(list, m)
		}
	}
	return DevEvent{Kind: "key", Key: name, Mods: strings.Join(list, "+")}, nil
}

// devKey is a key of a script: its key code, and the characters that it makes without and with Shift (the
// ones of a US layout).
type devKey struct {
	code        uint16
	char, shift string
}

// The function keys' characters in NSEvent are in the private use area (NSUpArrowFunctionKey and the others).
func fnChar(r rune) string { return string(r) }

var devKeyTable = func() map[string]devKey {
	letters := [...]uint16{0x00, 0x0b, 0x08, 0x02, 0x0e, 0x03, 0x05, 0x04, 0x22, 0x26, 0x28, 0x25, 0x2e, 0x2d, 0x1f, 0x23, 0x0c, 0x0f, 0x01, 0x11, 0x20, 0x09, 0x0d, 0x07, 0x10, 0x06}
	digits := [...]uint16{0x1d, 0x12, 0x13, 0x14, 0x15, 0x17, 0x16, 0x1a, 0x1c, 0x19}
	shifted := [...]string{")", "!", "@", "#", "$", "%", "^", "&", "*", "("}
	m := map[string]devKey{}
	for i, code := range letters {
		l := string(rune('a' + i))
		m["Key"+strings.ToUpper(l)] = devKey{code, l, strings.ToUpper(l)}
	}
	for i, code := range digits {
		m["Digit"+strconv.Itoa(i)] = devKey{code, strconv.Itoa(i), shifted[i]}
	}
	for name, k := range map[string]devKey{
		"Enter": {0x24, "\r", "\r"}, "Tab": {0x30, "\t", "\x19"}, "Space": {0x31, " ", " "},
		"Backspace": {0x33, "\x7f", "\x7f"}, "Escape": {0x35, "\x1b", "\x1b"},
		"Delete":  {0x75, fnChar(0xf728), fnChar(0xf728)},
		"ArrowUp": {0x7e, fnChar(0xf700), fnChar(0xf700)}, "ArrowDown": {0x7d, fnChar(0xf701), fnChar(0xf701)},
		"ArrowLeft": {0x7b, fnChar(0xf702), fnChar(0xf702)}, "ArrowRight": {0x7c, fnChar(0xf703), fnChar(0xf703)},
		"Home": {0x73, fnChar(0xf729), fnChar(0xf729)}, "End": {0x77, fnChar(0xf72b), fnChar(0xf72b)},
		"PageUp": {0x74, fnChar(0xf72c), fnChar(0xf72c)}, "PageDown": {0x79, fnChar(0xf72d), fnChar(0xf72d)},
	} {
		m[name] = k
	}
	fn := [...]uint16{0x7a, 0x78, 0x63, 0x76, 0x60, 0x61, 0x62, 0x64, 0x65, 0x6d, 0x67, 0x6f}
	for i, code := range fn {
		m["F"+strconv.Itoa(i+1)] = devKey{code, fnChar(rune(0xf704 + i)), fnChar(rune(0xf704 + i))}
	}
	return m
}()

// devKeyByName finds a key of a script by its Flutter name.
func devKeyByName(name string) (devKey, bool) {
	k, ok := devKeyTable[name]
	return k, ok
}

// devModifier is a modifier key of a script: its key code, its device-independent flag and the
// device-dependent flag that tells the left key.
type devModifier struct {
	code uint16
	flag uint64
	side uint64
}

var devModifierKeys = map[string]devModifier{
	"shift":   {0x38, 1 << 17, flagShiftLeft},
	"control": {0x3b, 1 << 18, flagControlLeft},
	"alt":     {0x3a, 1 << 19, flagAltLeft},
	"meta":    {0x37, 1 << 20, flagMetaLeft},
}

// devBaseFlags is what the flags of every real event have: the non-coalesced mask.
const devBaseFlags uint64 = 0x100

// panStepInterval is the time between the updates of a pan, about one frame at 60 Hz.
const panStepInterval = 16 * time.Millisecond

// parsePan makes the steps of a pan: the start of the gesture, an update and a wait for each interval
// of the duration, and the end.
func parsePan(args, step string) ([]DevEvent, error) {
	parts := strings.Split(args, ",")
	d := 100 * time.Millisecond
	if len(parts) == 5 {
		var err error
		if d, err = time.ParseDuration(strings.TrimSpace(parts[4])); err != nil || d <= 0 {
			return nil, errors.New(TagMacOSDevEventsInvalid, "the duration of pan must be positive, such as 100ms", "step", step)
		}
		parts = parts[:4]
	}
	v, ok := parseFloats(strings.Join(parts, ","), 4)
	if !ok {
		return nil, errors.New(TagMacOSDevEventsInvalid, "pan takes x,y,dx,dy and optionally a duration", "step", step)
	}
	n := int(d / panStepInterval)
	if n < 1 {
		n = 1
	}
	events := []DevEvent{{Kind: "panbegin", X: v[0], Y: v[1]}}
	// The trackpad's deltas are whole numbers of points: each step is the difference of the rounded
	// running totals, so that the steps add up to the total whatever n is.
	part := func(total float64, i int) float64 { return math.Round(total * float64(i) / float64(n)) }
	for i := 0; i < n; i++ {
		events = append(events, DevEvent{Kind: "wait", Wait: d / time.Duration(n)},
			DevEvent{Kind: "panstep", X: v[0], Y: v[1], DX: part(v[2], i+1) - part(v[2], i), DY: part(v[3], i+1) - part(v[3], i)})
	}
	return append(events, DevEvent{Kind: "wait", Wait: d / time.Duration(n)}, DevEvent{Kind: "panend", X: v[0], Y: v[1]}), nil
}

// parseFloats reads n numbers separated by commas.
func parseFloats(args string, n int) ([]float64, bool) {
	parts := strings.Split(args, ",")
	if len(parts) != n {
		return nil, false
	}
	out := make([]float64, n)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}
