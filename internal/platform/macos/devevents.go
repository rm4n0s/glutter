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
	// Kind is the name of the step: "wait", "move", "tap", "wheel", or one of the pieces that "pan" is
	// made of: "panbegin", "panstep" and "panend" (key, text and compose come with steps 33.4 and 33.5).
	Kind string
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
		case "pan":
			pan, err := parsePan(args, step)
			if err != nil {
				return nil, err
			}
			events = append(events, pan...)
		default:
			return nil, errors.New(TagMacOSDevEventsInvalid, "the step is not wait, move, tap, wheel or pan (key, text and compose come with the keyboard steps)", "step", step)
		}
	}
	return events, nil
}

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
