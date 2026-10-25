package windows

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
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
	// Kind is the name of the step: "wait", "move", "tap" or "wheel" (step 28.4 adds the keyboard).
	Kind string
	// Wait is how long to wait, for "wait".
	Wait time.Duration
	// X and Y are the place in the first window, in logical pixels from its top left, for "move",
	// "tap" and "wheel".
	X, Y float64
	// DX and DY are the scroll of a "wheel" in logical pixels; positive scrolls down and to the
	// right, as the web's deltas do.
	DX, DY float64
}

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments".
//
//	wait:500ms        waits
//	move:x,y          moves the mouse to the place (logical pixels of the first window)
//	tap:x,y           moves the mouse there and clicks the left button
//	wheel:x,y,dx,dy   turns the wheel at the place; dx and dy are the scroll in logical pixels
//
// key, text and compose come with step 28.4. A step that is not known, or has the wrong
// arguments, fails with WindowsDevEventsInvalid.
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
		default:
			return nil, errors.New(TagWindowsDevEventsInvalid, "the step is not wait, move, tap or wheel (key, text and compose come with step 28.4)", "step", step)
		}
	}
	return events, nil
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
