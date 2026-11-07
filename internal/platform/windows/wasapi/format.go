// Package wasapi is the WASAPI backend of the audio service on Windows (SPEC §28.3, step 31.6): playback and capture of float32
// frames through the shared mode of the Windows audio engine, driven by events, with the list of devices and the default device
// from the device enumerator and its notification client (a COM object that Go makes, SPEC §25.2). Every COM call is made on a
// thread of its own that is in the multithreaded apartment: one thread for the enumerator, and one for each open device. The design
// follows the WASAPI driver of SDL 3 (src/audio/wasapi/SDL_wasapi.c; zlib license,
// third_party/sdl/LICENSE), translated to Go and changed: each device has a thread
// of its own that does everything for it (SDL proxies to a management thread), the stream is made again on the new default device
// when the default changes, and IAudioClient3 is asked for the period the app wants.
//
// This file and the files without an operating system in their name are plain Go (the choice of the format, the channel masks, the
// arithmetic of the buffers) and are tested on any host; the calls into Windows are in the files that end in _windows.go.
//
// Flutter: no counterpart
package wasapi

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The limits of a format, as in the other backends (SPEC §28.3).
const (
	maxChannels = 8
	maxRate     = 768000
	minRate     = 1000
)

// channelMasks are the speakers of SDL's layouts for 1 to 8 channels (mono, stereo, 2.1, quad, 4.1, 5.1, 6.1, 7.1). A frame of a
// WASAPI stream has its channels in the order of the bits that are set in the mask, and SDL's order is that order for each layout
// (front left and right, center, low frequency, back left and right, back center, side left and right), so no channel needs to be
// moved: SDL's driver relies on the same fact.
var channelMasks = [maxChannels + 1]uint32{
	0,
	win32.SpeakerFrontCenter,
	win32.SpeakerFrontLeft | win32.SpeakerFrontRight,
	win32.SpeakerFrontLeft | win32.SpeakerFrontRight | win32.SpeakerLowFrequency,
	win32.SpeakerFrontLeft | win32.SpeakerFrontRight | win32.SpeakerBackLeft | win32.SpeakerBackRight,
	win32.SpeakerFrontLeft | win32.SpeakerFrontRight | win32.SpeakerLowFrequency | win32.SpeakerBackLeft | win32.SpeakerBackRight,
	win32.SpeakerFrontLeft | win32.SpeakerFrontRight | win32.SpeakerFrontCenter | win32.SpeakerLowFrequency | win32.SpeakerBackLeft | win32.SpeakerBackRight,
	win32.SpeakerFrontLeft | win32.SpeakerFrontRight | win32.SpeakerFrontCenter | win32.SpeakerLowFrequency | win32.SpeakerBackCenter | win32.SpeakerSideLeft | win32.SpeakerSideRight,
	win32.SpeakerFrontLeft | win32.SpeakerFrontRight | win32.SpeakerFrontCenter | win32.SpeakerLowFrequency | win32.SpeakerBackLeft | win32.SpeakerBackRight | win32.SpeakerSideLeft | win32.SpeakerSideRight,
}

// channelMask is the channel mask of SDL's layout of n channels, 0 for a number that is not 1 to 8.
func channelMask(n int) uint32 {
	if n < 1 || n > maxChannels {
		return 0
	}
	return channelMasks[n]
}

// wireFormat is the extensible wave format of float32 frames of the given rate and channels, the format that every stream asks
// the audio engine for.
func wireFormat(f platform.AudioFormat) win32.WaveFormatExtensible {
	blockAlign := uint16(f.Channels * 4)
	return win32.WaveFormatExtensible{
		FormatTag:          win32.WaveFormatTagExtensible,
		Channels:           uint16(f.Channels),
		SamplesPerSec:      uint32(f.Rate),
		AvgBytesPerSec:     uint32(f.Rate) * uint32(blockAlign),
		BlockAlign:         blockAlign,
		BitsPerSample:      32,
		CbSize:             22,
		ValidBitsPerSample: 32,
		ChannelMask:        channelMask(f.Channels),
		SubFormat:          win32.KSDataFormatSubTypeIEEEFloat,
	}
}

// chooseFormat is the format that a stream asks the engine for: the app's rate and channels, and the engine's mix format for a
// field that the app left at 0. A field outside the limits is the engine's too (the caller reports a format that the device
// refuses).
func chooseFormat(want, mix platform.AudioFormat) platform.AudioFormat {
	f := want
	if f.Rate < minRate || f.Rate > maxRate {
		f.Rate = mix.Rate
	}
	if f.Channels < 1 || f.Channels > maxChannels {
		f.Channels = mix.Channels
	}
	if f.Rate < minRate || f.Rate > maxRate {
		f.Rate = 48000
	}
	if f.Channels < 1 || f.Channels > maxChannels {
		f.Channels = 2
	}
	return f
}

// mixFormatOf is the format of the engine's mix format, which GetMixFormat returned: the first words of a WAVEFORMATEX, read by
// the caller.
func mixFormatOf(channels uint16, rate uint32) platform.AudioFormat {
	return platform.AudioFormat{Rate: int(rate), Channels: int(channels)}
}

// periodFrames is the period that IAudioClient3 is asked for: the frames the app wants (the engine's smallest period when it wants
// none), rounded to a multiple of the fundamental period, which is all that InitializeSharedAudioStream takes, and kept between the
// smallest and the largest. A fundamental of 0 (an engine that says nothing) leaves the number as it is.
func periodFrames(want, fundamental, lowest, highest uint32) uint32 {
	n := want
	if n == 0 {
		n = lowest
	}
	if fundamental > 0 {
		n = (n + fundamental/2) / fundamental * fundamental
	}
	if highest > 0 && n > highest {
		n = highest
		if fundamental > 0 {
			n -= n % fundamental
		}
	}
	if n < lowest {
		n = lowest
	}
	return n
}

// framesDuration is the length of a number of frames at a rate.
func framesDuration(frames, rate int) time.Duration {
	if rate <= 0 {
		return 0
	}
	return time.Duration(int64(frames) * int64(time.Second) / int64(rate))
}

// flowOf is the data flow (eRender or eCapture) of a kind.
func flowOf(kind platform.AudioKind) uint32 {
	if kind == platform.AudioCapture {
		return win32.ECapture
	}
	return win32.ERender
}

// failure says how a failed HRESULT of the audio client is told to the app.
type failure uint8

const (
	// failOpen is a device that cannot be opened (the generic failure).
	failOpen failure = iota
	// failFormat is a format that the device does not take.
	failFormat
	// failPermission is a microphone that the user's settings do not allow.
	failPermission
	// failLost is a device that went away.
	failLost
)

// classify is the way a failed HRESULT of the audio client is told: the unsupported format, the denied access to a microphone, and
// a client that was invalidated (the device was unplugged, disabled or reconfigured, the service stopped) are told apart, and
// everything else is a device that cannot be opened.
func classify(hr win32.HRESULT) failure {
	switch hr {
	case win32.AudclntEUnsupportedFormat:
		return failFormat
	case win32.EAccessDenied:
		return failPermission
	case win32.AudclntEDeviceInvalidated, win32.AudclntEResourcesInvalidated, win32.AudclntEServiceNotRunning:
		return failLost
	}
	return failOpen
}
