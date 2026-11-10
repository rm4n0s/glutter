package ios

import (
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// This file is the plain-Go part of the audio session of iOS (SPEC §30.9): the category that the streams of the app need, the rule that
// an app that records keeps the play-and-record category, and the activation. The calls to AVAudioSession are in audiosession_ios.go,
// behind sessionAPI, so that the rules are tested on every host.
//
// The design follows SDL 3's src/audio/coreaudio/SDL_coreaudio.m (the iOS session parts), which is where the categories, the
// activation and the handling of the interruptions came from; the code here is new and is changed.

// The categories of AVAudioSession that the app uses (the strings are the values of the constants).
const (
	categoryPlayback      = "AVAudioSessionCategoryPlayback"
	categoryPlayAndRecord = "AVAudioSessionCategoryPlayAndRecord"
)

// The options of a category (AVAudioSessionCategoryOptions): play and record sends the sound to the speaker and lets a Bluetooth headset
// be the input, and playback lets what another app plays go on under the sound of this one.
const (
	optionMixWithOthers      uint64 = 1 << 0
	optionAllowBluetooth     uint64 = 1 << 2
	optionDefaultToSpeaker   uint64 = 1 << 3
	optionAllowBluetoothA2DP uint64 = 1 << 5
)

// sessionAPI is what the session asks of AVAudioSession.
type sessionAPI interface {
	// SetCategory sets the category and its options; it gives the reason of a failure.
	SetCategory(category string, options uint64) (reason string, ok bool)
	// SetActive makes the session active or not; it gives the reason of a failure.
	SetActive(active bool) (reason string, ok bool)
}

// sessionPolicy says which category the streams that have been opened need.
type sessionPolicy struct {
	mu sync.Mutex
	// recording says that a stream recorded: the category stays play and record for the rest of the run, since switching back would cut
	// a stream that still runs.
	recording bool
	// category is the category that was set last, "" for none.
	category string
	api      sessionAPI
}

// categoryOf is the category and the options for the streams: playback for an app that only plays, play and record (with the speaker and
// Bluetooth) once a stream has recorded.
//
// Flutter: no counterpart
func categoryOf(recording bool) (category string, options uint64) {
	if recording {
		return categoryPlayAndRecord, optionDefaultToSpeaker | optionAllowBluetooth | optionAllowBluetoothA2DP
	}
	return categoryPlayback, 0
}

// Activate sets the session up for a stream of the kind and makes it active: the category is set when it is not the one the session has,
// and the session is activated each time (it is cheap, and an interruption that ended needs it). It fails with AudioSessionFailed, with
// the reason that AVAudioSession gave.
//
// Flutter: no counterpart
func (p *sessionPolicy) Activate(kind platform.AudioKind) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kind == platform.AudioCapture {
		p.recording = true
	}
	category, options := categoryOf(p.recording)
	if category != p.category {
		if reason, ok := p.api.SetCategory(category, options); !ok {
			return errors.New(tagAudioSessionFailed, "the audio session refused the category", "category", category, "reason", reason)
		}
		p.category = category
	}
	if reason, ok := p.api.SetActive(true); !ok {
		return errors.New(tagAudioSessionFailed, "the audio session could not be made active", "reason", reason)
	}
	return nil
}

// The values of AVAudioSession's interruption and route change notifications that the app looks at.
const (
	interruptionBegan                = 1
	interruptionOptionResume  uint64 = 1
	routeOldDeviceUnavailable        = 2
)

// interruptionOf reads an interruption notification: began, and for an ended one whether the system says that the audio should go on.
//
// Flutter: no counterpart
func interruptionOf(typ, options uint64) (began, resume bool) {
	if typ == interruptionBegan {
		return true, false
	}
	return false, options&interruptionOptionResume != 0
}

// routeLostOf says that the reason of a route change is the loss of the device that the audio went to (the headphones were unplugged).
//
// Flutter: no counterpart
func routeLostOf(reason uint64) bool { return reason == routeOldDeviceUnavailable }

// The tag of the error that this file makes (rm4n0s/errors, R5).
const tagAudioSessionFailed = "AudioSessionFailed"
