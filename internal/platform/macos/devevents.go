package macos

import (
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
	// Kind is the name of the step: "wait" (the others come with the input of steps 33.3 to 33.5).
	Kind string
	// Wait is how long to wait, for "wait".
	Wait time.Duration
}

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments". Only
// wait:500ms is known yet; a step that is not known, or has the wrong arguments, fails with
// MacOSDevEventsInvalid.
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
		default:
			return nil, errors.New(TagMacOSDevEventsInvalid, "the step is not wait (move, tap, wheel, key, text and compose come with the input steps)", "step", step)
		}
	}
	return events, nil
}
