package windows

import (
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
	// Kind is the name of the step. Only "wait" exists until step 28.3 adds the mouse and 28.4 the
	// keyboard.
	Kind string
	// Wait is how long to wait, for "wait".
	Wait time.Duration
}

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments".
//
//	wait:500ms   waits
//
// The mouse steps (move, tap, wheel) come with step 28.3, and key, text and compose with 28.4. A
// step that is not known, or has the wrong arguments, fails with WindowsDevEventsInvalid.
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
		default:
			return nil, errors.New(TagWindowsDevEventsInvalid, "the step is not wait (move, tap, wheel, key, text and compose come with later steps)", "step", step)
		}
	}
	return events, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagWindowsDevEventsInvalid is the tag of the error for a GLUTTER_DEV_EVENTS script that is
	// not valid; the metadata is "step".
	TagWindowsDevEventsInvalid = "WindowsDevEventsInvalid"
)
