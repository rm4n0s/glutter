package web

import (
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
)

// The audio of the web (SPEC §28.3, step 31.6): an AudioWorklet that plays frames the Go program puts in a ring, and one that records.
// The calls into the browser are in audio_js.go; this file has what needs no browser, so that a test runs it anywhere: the names of
// the files and the processors, the lengths of time, the way a browser's error becomes a tag, and the list of devices.

const (
	// AudioWorkletFile is the name of the script that glutter-web writes next to index.html and the page loads with
	// audioWorklet.addModule.
	//
	// Flutter: no counterpart
	AudioWorkletFile = "glutter-audio-worklet.js"
	// AudioPlaybackProcessor is the name that the worklet registers its playback processor under.
	//
	// Flutter: no counterpart
	AudioPlaybackProcessor = "glutter-playback"
	// AudioCaptureProcessor is the name that the worklet registers its capture processor under.
	//
	// Flutter: no counterpart
	AudioCaptureProcessor = "glutter-capture"
	// AudioDefaultBuffer is the length of the buffer of a device when the app asks for none (SPEC §22: the web's target is 120 ms).
	//
	// Flutter: no counterpart
	AudioDefaultBuffer = 100 * time.Millisecond
	// AudioPumpInterval is how often the Go program tops up the buffers of the playbacks and takes what the captures have.
	//
	// Flutter: no counterpart
	AudioPumpInterval = 10 * time.Millisecond
	// AudioCaptureBlock is the length of the blocks that a capture without shared memory posts.
	//
	// Flutter: no counterpart
	AudioCaptureBlock = 10 * time.Millisecond
	// AudioMaxChunk is the most frames that one call of a playback's handler is asked for.
	//
	// Flutter: no counterpart
	AudioMaxChunk = 4096
)

// MediaErrorTag is the tag of the audio error that a browser's DOMException (the name of the rejection of getUserMedia, or of the
// exception of a constructor) stands for: a refusal of the user is AudioCapturePermissionDenied, constraints that the browser cannot
// meet are AudioFormatUnsupported, and any other failure is AudioDeviceOpenFailed.
//
// Flutter: no counterpart
func MediaErrorTag(name string) string {
	switch name {
	case "NotAllowedError", "SecurityError", "PermissionDeniedError":
		return platform.TagAudioCapturePermissionDenied
	case "OverconstrainedError", "ConstraintNotSatisfiedError", "NotSupportedError", "IndexSizeError":
		return platform.TagAudioFormatUnsupported
	}
	return platform.TagAudioDeviceOpenFailed
}

// MediaDevice is one entry of the list that navigator.mediaDevices.enumerateDevices gives.
//
// Flutter: no counterpart
type MediaDevice struct {
	// Kind is "audioinput", "audiooutput" or a kind of video.
	Kind string
	// ID is the deviceId.
	ID string
	// Label is the name of the device, which the browser gives only after the user allowed a recording ("" before).
	Label string
}

// AudioDevicesOf turns the list of the browser into the devices of a kind. The default device is the one with the ID "default" (Chrome
// lists it first), else the first. A device with no label is named by its number.
//
// Flutter: no counterpart
func AudioDevicesOf(list []MediaDevice, kind platform.AudioKind) []platform.AudioDeviceInfo {
	want := "audiooutput"
	generic := "Audio output "
	if kind == platform.AudioCapture {
		want, generic = "audioinput", "Microphone "
	}
	var out []platform.AudioDeviceInfo
	for _, d := range list {
		if d.Kind != want {
			continue
		}
		name := d.Label
		if name == "" {
			name = generic + strconv.Itoa(len(out)+1)
		}
		out = append(out, platform.AudioDeviceInfo{ID: d.ID, Name: name, Kind: kind, Default: d.ID == "default"})
	}
	if len(out) > 0 {
		any := false
		for _, d := range out {
			any = any || d.Default
		}
		if !any {
			out[0].Default = true
		}
	}
	return out
}

// DefaultAudioDevice is the device that Devices lists before the browser has listed its own, and when it lists none (the labels and
// the devices of the microphones are hidden until the user has allowed one, and the browsers have no list of outputs).
//
// Flutter: no counterpart
func DefaultAudioDevice(kind platform.AudioKind) platform.AudioDeviceInfo {
	if kind == platform.AudioCapture {
		return platform.AudioDeviceInfo{ID: "default", Name: "Default microphone", Kind: kind, Default: true}
	}
	return platform.AudioDeviceInfo{ID: "default", Name: "Default output", Kind: kind, Default: true}
}

// PlaybackChannels is the number of channels of a playback: the app's (2 when it asks for none), cut to what the output has.
//
// Flutter: no counterpart
func PlaybackChannels(asked, max int) int {
	if asked < 1 {
		asked = 2
	}
	if max >= 1 && asked > max {
		return max
	}
	return asked
}

// AudioTarget is how many frames the Go program keeps in the ring of a device: BufferFrames of the options, else AudioDefaultBuffer
// at the rate.
//
// Flutter: no counterpart
func AudioTarget(bufferFrames, rate int) int {
	if bufferFrames > 0 {
		return bufferFrames
	}
	return int(int64(rate) * int64(AudioDefaultBuffer) / int64(time.Second))
}
