package web

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// DevEventsEnv is the environment variable (the query parameter of the page) with the script of
// synthetic events that debug builds replay after the first frame, so that a proof screenshot
// can show an app after a scroll or a tap. It is read only in debug builds (foundation.DebugMode).
//
// Flutter: no counterpart
const DevEventsEnv = "GLUTTER_DEV_EVENTS"

// DevEvent is one step of a dev events script.
//
// Flutter: no counterpart
type DevEvent struct {
	// Kind is "wait", "wheel", "tap", "move", "key", "input", "compose" or "paste".
	Kind string
	// Wait is how long to wait before the next step, for "wait".
	Wait time.Duration
	// X and Y are the position in CSS pixels from the top left of the view, for "wheel" and
	// "tap" and "move"; DX and DY are the deltas of a "wheel" in pixels.
	X, Y, DX, DY float64
	// Code and Key are KeyboardEvent.code and KeyboardEvent.key for a "key".
	Code, Key string
	// Text is the text of an "input" and a "paste", and Steps the successive texts of a "compose".
	Text  string
	Steps []string
}

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments".
//
//	wait:500ms        waits
//	wheel:x,y,dx,dy   a wheel event at the position, with the deltas in pixels
//	tap:x,y           a pointer down and up of the mouse's primary button at the position
//	move:x,y          the mouse moves to the position
//	key:code,key      a key down and up, such as key:KeyK,k or key:Enter,Enter
//	input:text        text typed into the hidden text field, as the browser's beforeinput and input
//	                  events do (\n is a line break); the text has no ";"
//	compose:a,b,c     an input method composing a, then b, then c, which it commits
//	paste:text        a paste event with the text, at the hidden text field
//
// A step that is not one of these, or has the wrong arguments, fails with WebDevEventsInvalid.
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
			return nil, errors.New("WebDevEventsInvalid", "a step must be name:arguments", "step", step)
		}
		ev := DevEvent{Kind: name}
		parts := strings.Split(args, ",")
		switch name {
		case "wait":
			d, err := time.ParseDuration(args)
			if err != nil || d < 0 {
				return nil, errors.New("WebDevEventsInvalid", "wait takes a duration such as 500ms", "step", step)
			}
			ev.Wait = d
		case "wheel":
			nums, ok := floats(parts, 4)
			if !ok {
				return nil, errors.New("WebDevEventsInvalid", "wheel takes x,y,dx,dy", "step", step)
			}
			ev.X, ev.Y, ev.DX, ev.DY = nums[0], nums[1], nums[2], nums[3]
		case "tap", "move":
			nums, ok := floats(parts, 2)
			if !ok {
				return nil, errors.New("WebDevEventsInvalid", name+" takes x,y", "step", step)
			}
			ev.X, ev.Y = nums[0], nums[1]
		case "key":
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return nil, errors.New("WebDevEventsInvalid", "key takes code,key", "step", step)
			}
			ev.Code, ev.Key = parts[0], parts[1]
		case "input", "paste":
			if args == "" {
				return nil, errors.New("WebDevEventsInvalid", name+" takes the text", "step", step)
			}
			ev.Text = strings.ReplaceAll(args, `\n`, "\n")
		case "compose":
			if args == "" || slices.Contains(parts, "") {
				return nil, errors.New("WebDevEventsInvalid", "compose takes the texts a,b,c", "step", step)
			}
			ev.Steps = parts
		default:
			return nil, errors.New("WebDevEventsInvalid", "the step is not wait, wheel, tap, move, key, input, compose or paste", "step", step)
		}
		events = append(events, ev)
	}
	return events, nil
}

// floats reads n numbers, and reports false when there are not n or one is not a number.
func floats(parts []string, n int) ([]float64, bool) {
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
