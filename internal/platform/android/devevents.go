package android

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// DevEventsEnv is the environment variable (an intent extra) with the script of synthetic events
// that debug builds replay after the first frame, so that a proof screenshot or a device test can
// show an app after a tap or a swipe where `adb shell input` cannot reach (SPEC §24.13). It is read
// only in debug builds (foundation.DebugMode).
//
// Flutter: no counterpart
const DevEventsEnv = "GLUTTER_DEV_EVENTS"

// DevEvent is one step of a dev events script.
//
// Flutter: no counterpart
type DevEvent struct {
	// Kind is "wait", "tap", "swipe" or "key".
	Kind string
	// Wait is how long to wait, for "wait"; and a swipe takes its Millis.
	Wait time.Duration
	// X1, Y1 and X2, Y2 are the points of a "tap" (the first) and a "swipe", in pixels of the view.
	X1, Y1, X2, Y2 float64
	// Millis is how long a swipe takes.
	Millis int
	// Key is the name of a key: KEYCODE_ENTER.
	Key string
}

var keyName = regexp.MustCompile(`^KEYCODE_[A-Z0-9_]+$`)

// ParseDevEvents reads a dev events script: steps separated by ";", each "name:arguments".
//
//	wait:500ms          waits
//	tap:x,y             a touch down and up at the position
//	swipe:x1,y1,x2,y2,ms  a finger from the first point to the second in that many milliseconds
//	key:KEYCODE_ENTER   a key down and up
//
// A step that is not one of these, or has the wrong arguments, fails with AndroidDevEventsInvalid.
// The later steps of the script (text, compose, back, a11y) come with the steps that add them.
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
			return nil, errors.New(tagAndroidDevEventsInvalid, "a step must be name:arguments", "step", step)
		}
		ev := DevEvent{Kind: name}
		parts := strings.Split(args, ",")
		switch name {
		case "wait":
			d, err := time.ParseDuration(args)
			if err != nil || d < 0 {
				return nil, errors.New(tagAndroidDevEventsInvalid, "wait takes a duration such as 500ms", "step", step)
			}
			ev.Wait = d
		case "tap":
			nums, ok := floats(parts, 2)
			if !ok {
				return nil, errors.New(tagAndroidDevEventsInvalid, "tap takes x,y", "step", step)
			}
			ev.X1, ev.Y1 = nums[0], nums[1]
		case "swipe":
			nums, ok := floats(parts, 5)
			if !ok || nums[4] <= 0 || nums[4] != float64(int(nums[4])) {
				return nil, errors.New(tagAndroidDevEventsInvalid, "swipe takes x1,y1,x2,y2,ms", "step", step)
			}
			ev.X1, ev.Y1, ev.X2, ev.Y2, ev.Millis = nums[0], nums[1], nums[2], nums[3], int(nums[4])
		case "key":
			if !keyName.MatchString(args) {
				return nil, errors.New(tagAndroidDevEventsInvalid, "key takes a name such as KEYCODE_ENTER", "step", step)
			}
			ev.Key = args
		default:
			return nil, errors.New(tagAndroidDevEventsInvalid, "the step is not wait, tap, swipe or key", "step", step)
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

// startDevEvents replays the script, once, from a goroutine of its own, after the first frame: the
// Java side injects each tap, swipe and key through the view's own dispatch, the path of the user's
// input. The goroutine waits as long as a step takes.
func (p *Platform) startDevEvents() {
	if len(p.devEvents) == 0 || p.devStarted {
		return
	}
	p.devStarted = true
	events := p.devEvents
	go func() {
		for _, e := range events {
			switch e.Kind {
			case "wait":
				time.Sleep(e.Wait)
			case "tap":
				p.host.Inject(e)
				time.Sleep(150 * time.Millisecond)
			case "swipe":
				p.host.Inject(e)
				time.Sleep(time.Duration(e.Millis)*time.Millisecond + 150*time.Millisecond)
			case "key":
				p.host.Inject(e)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAndroidDevEventsInvalid = "AndroidDevEventsInvalid"
)
