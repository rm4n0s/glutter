// Package audio plays and records sound (SPEC §28.3): a System lists the devices, opens a playback or a capture Device, and the
// streams of a device are mixed into it. Each stream is converted to the format of its device (the channels, the rate and the
// speed) by the code of SDL 3's audio streams, which is ported to Go in internal/audiocvt, and all the streams of a device are
// mixed in float32 and clipped. Reading sound files is in package audio/wav; other codecs are not in Glutter.
//
// Samples are interleaved: one sample for each channel of a frame, a number of frames a second (Format.Rate). A stream takes
// float32 samples from -1 to 1, or signed 16-bit and 32-bit integers, which are converted when they are put. The channels are in
// SDL's order: mono (FC); stereo (FL FR); 2.1 (FL FR LFE); quad (FL FR BL BR); 4.1 (FL FR LFE BL BR); 5.1 (FL FR FC LFE BL BR);
// 6.1 (FL FR FC LFE BC SL SR); 7.1 (FL FR FC LFE BL BR SL SR).
//
// The backend of a device runs on a thread of its own, so Stream.Put and Stream.Get may be called from any goroutine, and the
// function of a callback stream runs on that thread and must not touch the framework. Everything else is for the UI thread, as
// the rest of the framework (SPEC §13). Flutter has no audio of its own; this package is Glutter's.
//
// Flutter: no counterpart
package audio

import (
	"time"

	"github.com/rm4n0s/glutter/internal/audiocvt"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Format is the format of audio: the sample rate in frames a second and the number of channels, from 1 to 8.
//
// Flutter: no counterpart
type Format struct {
	Rate     int
	Channels int
}

// The limits of a Format.
const (
	// MinRate is the lowest sample rate of a stream.
	//
	// Flutter: no counterpart
	MinRate = 1000
	// MaxRate is the highest sample rate of a stream.
	//
	// Flutter: no counterpart
	MaxRate = 768000
	// MaxChannels is the most channels of a stream (7.1).
	//
	// Flutter: no counterpart
	MaxChannels = audiocvt.MaxChannels
)

// Valid says whether the rate and the channels are within the limits.
//
// Flutter: no counterpart
func (f Format) Valid() bool {
	return f.Rate >= MinRate && f.Rate <= MaxRate && f.Channels >= 1 && f.Channels <= MaxChannels
}

// Frames returns how many frames a duration is.
//
// Flutter: no counterpart
func (f Format) Frames(d time.Duration) int { return int(d.Seconds() * float64(f.Rate)) }

// Duration returns how long frames frames take.
//
// Flutter: no counterpart
func (f Format) Duration(frames int) time.Duration {
	if f.Rate <= 0 {
		return 0
	}
	return time.Duration(float64(frames) / float64(f.Rate) * float64(time.Second))
}

func (f Format) platform() platform.AudioFormat {
	return platform.AudioFormat{Rate: f.Rate, Channels: f.Channels}
}

func formatOf(f platform.AudioFormat) Format { return Format{Rate: f.Rate, Channels: f.Channels} }

// DeviceKind says whether a device plays sound or records it.
//
// Flutter: no counterpart
type DeviceKind uint8

const (
	// Playback is a device that plays sound.
	//
	// Flutter: no counterpart
	Playback DeviceKind = iota
	// Capture is a device that records sound.
	//
	// Flutter: no counterpart
	Capture
)

// String names the kind.
//
// Flutter: no counterpart
func (k DeviceKind) String() string {
	if k == Capture {
		return "capture"
	}
	return "playback"
}

// DeviceInfo describes a device that the system has.
//
// Flutter: no counterpart
type DeviceInfo struct {
	// ID names the device for DeviceOpts.ID.
	ID string
	// Name is what to show to the user.
	Name string
	Kind DeviceKind
	// Default says that the system uses this device when none is asked for.
	Default bool
	// Format is the format that the device likes best; zero when the backend does not say.
	Format Format
}

// DeviceState is the state of a device.
//
// Flutter: no counterpart
type DeviceState uint8

const (
	// DeviceStateRunning is a device that plays or records.
	//
	// Flutter: no counterpart
	DeviceStateRunning DeviceState = iota
	// DeviceStatePaused is a device that was paused with Device.Pause, or by the platform because the app is hidden.
	//
	// Flutter: no counterpart
	DeviceStatePaused
	// DeviceStateSuspended is a device that waits for the user: on the web a page may not make sound before the user has
	// pressed a key or clicked. The device runs by itself afterwards, and the listeners of the device are told.
	//
	// Flutter: no counterpart
	DeviceStateSuspended
	// DeviceStateLost is a device that is gone (unplugged, or the audio server ended) and that was not replaced.
	//
	// Flutter: no counterpart
	DeviceStateLost
	// DeviceStateClosed is a device that was closed.
	//
	// Flutter: no counterpart
	DeviceStateClosed
)

// The tags of the errors of this package (SPEC §28.7).
const (
	// TagUnavailable is the tag of the error of an audio service that is not there: the platform has none, or no audio server runs.
	//
	// Flutter: no counterpart
	TagUnavailable = platform.TagAudioUnavailable
	// TagDeviceOpenFailed is the tag of the error of a device that cannot be opened.
	//
	// Flutter: no counterpart
	TagDeviceOpenFailed = platform.TagAudioDeviceOpenFailed
	// TagDeviceLost is the tag of the error of a call on a device that is gone.
	//
	// Flutter: no counterpart
	TagDeviceLost = platform.TagAudioDeviceLost
	// TagFormatUnsupported is the tag of the error of a format that a device cannot work in, and of samples that are not a whole
	// number of frames or not of a type that a stream takes.
	//
	// Flutter: no counterpart
	TagFormatUnsupported = platform.TagAudioFormatUnsupported
	// TagCapturePermissionDenied is the tag of the error of a recording that the user did not allow.
	//
	// Flutter: no counterpart
	TagCapturePermissionDenied = platform.TagAudioCapturePermissionDenied
	// TagStreamClosed is the tag of the error of a call on a stream that was closed, or whose device was.
	//
	// Flutter: no counterpart
	TagStreamClosed = "AudioStreamClosed"
	// TagWrongKind is the tag of the error of Put on the stream of a capture device, or Get on the stream of a playback device.
	//
	// Flutter: no counterpart
	TagWrongKind = "AudioStreamWrongKind"
)
