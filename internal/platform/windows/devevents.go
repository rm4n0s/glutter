package windows

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// DevEventsEnv is the environment variable with the script of synthetic events that debug builds
// replay after the first frame, so that a proof screenshot or a Wine test can show an app after a
// click or a key (SPEC §25.6). It is read only in debug builds (foundation.DebugMode).
//
// Flutter: no counterpart
const DevEventsEnv = "GLUTTER_DEV_EVENTS"

// DevEvent is one step of a dev events script.
//
// Flutter: no counterpart
type DevEvent struct {
	// Kind is the name of the step: "wait", "move", "tap", "wheel", "key", "keydown" or "keyup".
	Kind string
	// Wait is how long to wait, for "wait".
	Wait time.Duration
	// X and Y are the place in the first window, in logical pixels from its top left, for "move",
	// "tap" and "wheel".
	X, Y float64
	// DX and DY are the scroll of a "wheel" in logical pixels; positive scrolls down and to the
	// right, as the web's deltas do.
	DX, DY float64
	// Keys are the virtual-key codes of a "key", "keydown" or "keyup": "key" presses them in order
	// and releases them in the opposite order.
	Keys []uint32
}

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments".
//
//	wait:500ms        waits
//	move:x,y          moves the mouse to the place (logical pixels of the first window)
//	tap:x,y           moves the mouse there and clicks the left button
//	wheel:x,y,dx,dy   turns the wheel at the place; dx and dy are the scroll in logical pixels
//	key:A             presses and releases a key; key:SHIFT+LEFT presses Shift, then Left, and releases both
//	keydown:SHIFT     presses a key and keeps it down (until keyup:SHIFT)
//	keyup:SHIFT       releases a key
//
// A key is a name of Flutter's table of virtual keys (the name of the VK_ constant without the prefix,
// in any case, with or without "VK_": RETURN, BACK, LEFT, PRIOR, F1, SHIFT, CONTROL, MENU...), a
// letter or a digit, or one of the aliases ENTER, BACKSPACE, CTRL, ALT, ESC, DEL, PAGEUP and PAGEDOWN.
// text and compose come with step 28.5. A step that is not known, or has the wrong arguments, fails
// with WindowsDevEventsInvalid.
//
// Flutter: no counterpart
func ParseDevEvents(script string) ([]DevEvent, error) {
	var events []DevEvent
	for _, step := range strings.Split(script, ";") {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		name, args, ok := strings.Cut(step, ":")
		if !ok {
			return nil, errors.New(TagWindowsDevEventsInvalid, "a step must be name:arguments", "step", step)
		}
		switch name {
		case "wait":
			d, err := time.ParseDuration(args)
			if err != nil || d < 0 {
				return nil, errors.New(TagWindowsDevEventsInvalid, "wait takes a duration such as 500ms", "step", step)
			}
			events = append(events, DevEvent{Kind: name, Wait: d})
		case "move", "tap":
			xy, ok := parseFloats(args, 2)
			if !ok {
				return nil, errors.New(TagWindowsDevEventsInvalid, "move and tap take x,y", "step", step)
			}
			events = append(events, DevEvent{Kind: name, X: xy[0], Y: xy[1]})
		case "wheel":
			v, ok := parseFloats(args, 4)
			if !ok {
				return nil, errors.New(TagWindowsDevEventsInvalid, "wheel takes x,y,dx,dy", "step", step)
			}
			events = append(events, DevEvent{Kind: name, X: v[0], Y: v[1], DX: v[2], DY: v[3]})
		case "key", "keydown", "keyup":
			keys, ok := parseKeys(args)
			if !ok || (name != "key" && len(keys) != 1) {
				return nil, errors.New(TagWindowsDevEventsInvalid, "key takes key names joined by +, such as SHIFT+LEFT; keydown and keyup take one name", "step", step)
			}
			events = append(events, DevEvent{Kind: name, Keys: keys})
		default:
			return nil, errors.New(TagWindowsDevEventsInvalid, "the step is not wait, move, tap, wheel, key, keydown or keyup (text and compose come with step 28.5)", "step", step)
		}
	}
	return events, nil
}

// keyAliases are the names of keys that the scripts accept besides Flutter's names of virtual keys.
var keyAliases = map[string]string{
	"ENTER": "RETURN", "BACKSPACE": "BACK", "CTRL": "CONTROL", "ALT": "MENU", "ESC": "ESCAPE",
	"DEL": "DELETE", "PAGEUP": "PRIOR", "PAGEDOWN": "NEXT", "WIN": "LWIN", "CAPSLOCK": "CAPITAL",
}

// parseKeys reads key names joined by "+".
func parseKeys(args string) ([]uint32, bool) {
	var keys []uint32
	for _, name := range strings.Split(args, "+") {
		vk, ok := virtualKeyByName(name)
		if !ok {
			return nil, false
		}
		keys = append(keys, vk)
	}
	return keys, true
}

// virtualKeyByName finds the virtual key of a name: a letter or a digit is its character, else the
// table of Flutter's names (virtualKeyNames) after the aliases.
func virtualKeyByName(name string) (uint32, bool) {
	name = strings.ToUpper(strings.TrimSpace(name))
	name = strings.TrimPrefix(name, "VK_")
	if len(name) == 1 && (name[0] >= 'A' && name[0] <= 'Z' || name[0] >= '0' && name[0] <= '9') {
		return uint32(name[0]), true
	}
	if a, ok := keyAliases[name]; ok {
		name = a
	}
	if name == "MENU" {
		// Flutter's table names only the two sides of Alt (LMENU and RMENU).
		return win32.VKMenu, true
	}
	vk, ok := virtualKeyNames[name]
	return vk, ok
}

// isExtendedKey reports whether a virtual key is an extended key of the keyboard: Windows tells it
// from the key of the numpad with the same scan code by the extended flag, which a key sent with
// SendInput must carry.
func isExtendedKey(vk uint32) bool {
	switch vk {
	case win32.VKRControl, win32.VKRMenu, win32.VKInsert, win32.VKDelete, win32.VKHome, win32.VKEnd,
		win32.VKPrior, win32.VKNext, win32.VKLeft, win32.VKUp, win32.VKRight, win32.VKDown,
		win32.VKNumLock, win32.VKDivide, win32.VKLWin, win32.VKRWin, win32.VKApps, 0x2C /* VK_SNAPSHOT */ :
		return true
	}
	return false
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagWindowsDevEventsInvalid is the tag of the error for a GLUTTER_DEV_EVENTS script that is
	// not valid; the metadata is "step".
	TagWindowsDevEventsInvalid = "WindowsDevEventsInvalid"
)
