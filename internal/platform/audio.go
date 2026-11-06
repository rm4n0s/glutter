package platform

import "context"

// AudioKind says whether an audio device plays sound or records it.
//
// Flutter: no counterpart
type AudioKind uint8

const (
	// AudioPlayback is a device that plays sound: a speaker, headphones.
	//
	// Flutter: no counterpart
	AudioPlayback AudioKind = iota
	// AudioCapture is a device that records sound: a microphone.
	//
	// Flutter: no counterpart
	AudioCapture
)

// AudioFormat is the format of the samples that a device passes: interleaved float32 frames of Channels channels, Rate frames a
// second. The channels are in SDL's order (SPEC §28.3).
//
// Flutter: no counterpart
type AudioFormat struct {
	Rate, Channels int
}

// AudioDeviceInfo describes a device that the system has.
//
// Flutter: no counterpart
type AudioDeviceInfo struct {
	// ID names the device for Open; it is the system's own name for it and means nothing to the app.
	ID string
	// Name is what to show to the user.
	Name string
	Kind AudioKind
	// Default says that this is the device that the system uses when none is asked for.
	Default bool
	// Format is the format that the device likes best (zero when the backend does not know).
	Format AudioFormat
}

// AudioState is the state of an open device.
//
// Flutter: no counterpart
type AudioState uint8

const (
	// AudioRunning is a device that plays or records.
	//
	// Flutter: no counterpart
	AudioRunning AudioState = iota
	// AudioPaused is a device that its owner paused (or that the platform paused because the app is hidden).
	//
	// Flutter: no counterpart
	AudioPaused
	// AudioSuspended is a device that waits for something that is not up to the app (the browser's rule that sound starts with a
	// user gesture): it starts to run by itself.
	//
	// Flutter: no counterpart
	AudioSuspended
	// AudioLost is a device that is gone (unplugged, or the system's server ended) and was not replaced.
	//
	// Flutter: no counterpart
	AudioLost
	// AudioClosed is a device that was closed.
	//
	// Flutter: no counterpart
	AudioClosed
)

// AudioHandler is what a device calls, on the thread of its own (the audio thread: a locked goroutine or the thread of the
// system's library), never on the UI thread. It must not wait for the UI thread, and it must not allocate or take locks that a
// slow thread holds, or the sound breaks up.
//
// Flutter: no counterpart
type AudioHandler interface {
	// Fill is called by a playback device when it needs frames: out holds a whole number of frames of the device's format, and
	// is to be filled completely (with silence when there is no sound). The slice is only valid during the call.
	Fill(out []float32)
	// Capture is called by a capture device with the frames it recorded, a whole number of frames of the device's format. The
	// slice is only valid during the call.
	Capture(in []float32)
	// StateChanged says that the state of the device changed without the owner asking for it (a device that is lost, a suspended
	// device that starts). It may be called from any thread.
	StateChanged(s AudioState)
}

// AudioOpenOptions are what an app asks of a device when it opens it.
//
// Flutter: no counterpart
type AudioOpenOptions struct {
	// ID is the device to open; empty means the default device of the kind, which the backend follows when the system's default
	// changes.
	ID string
	// Format is the format the app would like; the device may give another, as Format of the AudioDevice says. A zero field
	// leaves the choice to the backend.
	Format AudioFormat
	// BufferFrames is the size of the buffer in frames the app would like; 0 leaves it to the backend (a latency within the
	// targets of SPEC §28.6).
	BufferFrames int
	// PauseInBackground says that the device pauses while the app is hidden (Android).
	PauseInBackground bool
}

// AudioDevice is a device that is open. The handler it was opened with is called until Close returns.
//
// Flutter: no counterpart
type AudioDevice interface {
	// Format is the format of the frames that the handler gets and gives. It does not change while the device is open.
	Format() AudioFormat
	// Pause stops the device from calling the handler; Resume starts it again. Both may be called on a device that is paused or
	// running.
	Pause() error
	Resume() error
	// Close ends the device. When it returns the handler is not running and is never called again.
	Close() error
}

// Audio is the audio service of a platform (SPEC §28.3).
//
// Flutter: no counterpart
type Audio interface {
	// Devices lists the devices of a kind. It fails with AudioUnavailable when the system has no audio server.
	Devices(kind AudioKind) ([]AudioDeviceInfo, error)
	// SetChangeHandler sets what is called when the list of devices or the default device changes; nil clears it. The function may
	// be called from any thread.
	SetChangeHandler(fn func())
	// Open opens a device. It fails with AudioDeviceOpenFailed when the device cannot be opened, AudioFormatUnsupported when it
	// cannot work in any format, AudioCapturePermissionDenied when the user refused the microphone, and AudioUnavailable when the
	// system has no audio server.
	Open(ctx context.Context, kind AudioKind, opts AudioOpenOptions, h AudioHandler) (AudioDevice, error)
}

// AudioHost is implemented by the host of the dispatcher (the engine) and found through ui.PlatformDispatcher.HostAs: it gives
// access to the audio service of the platform, or nil when the platform has none.
//
// Flutter: no counterpart
type AudioHost interface {
	// Audio returns the audio service, or nil.
	Audio() Audio
	// AddReleaseHook registers a function that the host calls when the app ends, so that a device that is still open is closed
	// before the process leaves. The function runs on the UI thread.
	AddReleaseHook(fn func())
}

// The tags of the errors an Audio service returns.
const (
	// TagAudioUnavailable is the tag of the error of an audio service that is not there: the platform has none, or the system has
	// no audio server running.
	//
	// Flutter: no counterpart
	TagAudioUnavailable = "AudioUnavailable"
	// TagAudioDeviceOpenFailed is the tag of the error of a device that cannot be opened.
	//
	// Flutter: no counterpart
	TagAudioDeviceOpenFailed = "AudioDeviceOpenFailed"
	// TagAudioDeviceLost is the tag of the error of a device that is gone.
	//
	// Flutter: no counterpart
	TagAudioDeviceLost = "AudioDeviceLost"
	// TagAudioFormatUnsupported is the tag of the error of a format, or samples, that the device or the stream cannot take.
	//
	// Flutter: no counterpart
	TagAudioFormatUnsupported = "AudioFormatUnsupported"
	// TagAudioCapturePermissionDenied is the tag of the error of a recording that the user did not allow.
	//
	// Flutter: no counterpart
	TagAudioCapturePermissionDenied = "AudioCapturePermissionDenied"
)
