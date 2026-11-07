//go:build linux

// Package aaudio is the AAudio backend of the audio service on Android (SPEC §28.3, step 31.6): playback and capture of float32 frames
// through libaaudio.so (Android 8, API 26), called through purego, with a data callback that is made once, the low-latency
// performance mode, a stream that is made again when the device it was on is disconnected, and a device that pauses while the app
// is hidden. libaaudio is loaded when the app first lists or opens a device. The design follows the AAudio driver of SDL 3
// (src/audio/aaudio/SDL_aaudio.c; zlib license, third_party/sdl/LICENSE), translated to Go and changed: the stream is in float32 with
// the sample rate and channels the app asks for (AAudio converts), the error callback hands the work of a disconnect to a goroutine
// (a stream cannot be closed from its own callbacks), and `RECORD_AUDIO` is asked through the Java side of the embedding.
//
// The package has only the tag of the Linux kernel (Android has it too), so that its tests run on the desktop with a scripted library,
// as the tests of the PulseAudio backend do; it is used on Android only.
//
// Flutter: no counterpart
package aaudio

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// LibraryEnv names the environment variable that overrides the path of libaaudio (SPEC §8.1).
const LibraryEnv = "GLUTTER_AAUDIO_LIBRARY"

// The values of the NDK's <aaudio/AAudio.h>.
const (
	unspecified = 0 // AAUDIO_UNSPECIFIED

	directionOutput = 0 // AAUDIO_DIRECTION_OUTPUT
	directionInput  = 1 // AAUDIO_DIRECTION_INPUT

	formatPCMFloat = 2 // AAUDIO_FORMAT_PCM_FLOAT

	sharingModeShared = 1 // AAUDIO_SHARING_MODE_SHARED

	performanceModeLowLatency = 12 // AAUDIO_PERFORMANCE_MODE_LOW_LATENCY

	usageGame = 14 // AAUDIO_USAGE_GAME

	contentTypeMusic = 2 // AAUDIO_CONTENT_TYPE_MUSIC

	inputPresetGeneric = 1 // AAUDIO_INPUT_PRESET_GENERIC

	callbackContinue = 0 // AAUDIO_CALLBACK_RESULT_CONTINUE
	callbackStop     = 1 // AAUDIO_CALLBACK_RESULT_STOP

	streamStateStarting     = 3  // AAUDIO_STREAM_STATE_STARTING
	streamStateStarted      = 4  // AAUDIO_STREAM_STATE_STARTED
	streamStatePausing      = 5  // AAUDIO_STREAM_STATE_PAUSING
	streamStatePaused       = 6  // AAUDIO_STREAM_STATE_PAUSED
	streamStateStopping     = 9  // AAUDIO_STREAM_STATE_STOPPING
	streamStateStopped      = 10 // AAUDIO_STREAM_STATE_STOPPED
	streamStateDisconnected = 13 // AAUDIO_STREAM_STATE_DISCONNECTED

	clockMonotonic = 1 // CLOCK_MONOTONIC
)

// The results of AAudio that the backend tells apart (aaudio_result_t).
const (
	resultOK              = 0
	resultDisconnected    = -899 // AAUDIO_ERROR_DISCONNECTED
	resultIllegalArgument = -898 // AAUDIO_ERROR_ILLEGAL_ARGUMENT
	resultInternal        = -896 // AAUDIO_ERROR_INTERNAL
	resultInvalidState    = -895 // AAUDIO_ERROR_INVALID_STATE
	resultUnavailable     = -889 // AAUDIO_ERROR_UNAVAILABLE
	resultNoFreeHandles   = -888 // AAUDIO_ERROR_NO_FREE_HANDLES
	resultNoMemory        = -887 // AAUDIO_ERROR_NO_MEMORY
	resultTimeout         = -885 // AAUDIO_ERROR_TIMEOUT
	resultInvalidFormat   = -883 // AAUDIO_ERROR_INVALID_FORMAT
	resultOutOfRange      = -882 // AAUDIO_ERROR_OUT_OF_RANGE
	resultNoService       = -881 // AAUDIO_ERROR_NO_SERVICE
	resultInvalidRate     = -880 // AAUDIO_ERROR_INVALID_RATE
)

// library holds the bound functions of libaaudio. The builder and the stream are uintptr (AAudioStreamBuilder * and AAudioStream *);
// the callbacks and the user data are uintptr too. The setters of the usage, the content type and the input preset came with
// Android 9 (API 28) and are nil on Android 8.
type library struct {
	createStreamBuilder func(out uintptr) int32

	builderDelete                    func(b uintptr) int32
	builderSetDeviceID               func(b uintptr, id int32)
	builderSetDirection              func(b uintptr, d int32)
	builderSetSharingMode            func(b uintptr, m int32)
	builderSetSampleRate             func(b uintptr, rate int32)
	builderSetChannelCount           func(b uintptr, n int32)
	builderSetFormat                 func(b uintptr, f int32)
	builderSetPerformanceMode        func(b uintptr, m int32)
	builderSetBufferCapacityInFrames func(b uintptr, n int32)
	builderSetFramesPerDataCallback  func(b uintptr, n int32)
	builderSetDataCallback           func(b, cb, user uintptr)
	builderSetErrorCallback          func(b, cb, user uintptr)
	builderSetUsage                  func(b uintptr, u int32)
	builderSetContentType            func(b uintptr, c int32)
	builderSetInputPreset            func(b uintptr, p int32)
	builderOpenStream                func(b, out uintptr) int32

	streamClose              func(s uintptr) int32
	streamRequestStart       func(s uintptr) int32
	streamRequestPause       func(s uintptr) int32
	streamRequestFlush       func(s uintptr) int32
	streamRequestStop        func(s uintptr) int32
	streamGetState           func(s uintptr) int32
	streamWaitForStateChange func(s uintptr, input int32, next uintptr, timeoutNanos int64) int32
	streamGetSampleRate      func(s uintptr) int32
	streamGetChannelCount    func(s uintptr) int32
	streamGetFormat          func(s uintptr) int32
	streamGetFramesPerBurst  func(s uintptr) int32
	streamGetBufferSize      func(s uintptr) int32
	streamSetBufferSize      func(s uintptr, n int32) int32
	streamGetXRunCount       func(s uintptr) int32
	streamGetTimestamp       func(s uintptr, clock int32, pos, nanos uintptr) int32
	convertResultToText      func(r int32) uintptr
}

// loadLibrary opens libaaudio.so and binds the functions. It fails with AudioUnavailable when the library cannot be loaded (Android
// 7 and older, or a desktop) or lacks a function of API 26.
func loadLibrary() (*library, error) {
	lib, err := dl.OpenOverride(LibraryEnv, "libaaudio.so")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(platform.TagAudioUnavailable, "could not load libaaudio", "names", md["names"], "dlerror", md["dlerror"])
	}
	l := new(library)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.createStreamBuilder, "AAudio_createStreamBuilder"},
		{&l.builderDelete, "AAudioStreamBuilder_delete"},
		{&l.builderSetDeviceID, "AAudioStreamBuilder_setDeviceId"},
		{&l.builderSetDirection, "AAudioStreamBuilder_setDirection"},
		{&l.builderSetSharingMode, "AAudioStreamBuilder_setSharingMode"},
		{&l.builderSetSampleRate, "AAudioStreamBuilder_setSampleRate"},
		{&l.builderSetChannelCount, "AAudioStreamBuilder_setChannelCount"},
		{&l.builderSetFormat, "AAudioStreamBuilder_setFormat"},
		{&l.builderSetPerformanceMode, "AAudioStreamBuilder_setPerformanceMode"},
		{&l.builderSetBufferCapacityInFrames, "AAudioStreamBuilder_setBufferCapacityInFrames"},
		{&l.builderSetFramesPerDataCallback, "AAudioStreamBuilder_setFramesPerDataCallback"},
		{&l.builderSetDataCallback, "AAudioStreamBuilder_setDataCallback"},
		{&l.builderSetErrorCallback, "AAudioStreamBuilder_setErrorCallback"},
		{&l.builderOpenStream, "AAudioStreamBuilder_openStream"},
		{&l.streamClose, "AAudioStream_close"},
		{&l.streamRequestStart, "AAudioStream_requestStart"},
		{&l.streamRequestPause, "AAudioStream_requestPause"},
		{&l.streamRequestFlush, "AAudioStream_requestFlush"},
		{&l.streamRequestStop, "AAudioStream_requestStop"},
		{&l.streamGetState, "AAudioStream_getState"},
		{&l.streamWaitForStateChange, "AAudioStream_waitForStateChange"},
		{&l.streamGetSampleRate, "AAudioStream_getSampleRate"},
		{&l.streamGetChannelCount, "AAudioStream_getChannelCount"},
		{&l.streamGetFormat, "AAudioStream_getFormat"},
		{&l.streamGetFramesPerBurst, "AAudioStream_getFramesPerBurst"},
		{&l.streamGetBufferSize, "AAudioStream_getBufferSizeInFrames"},
		{&l.streamSetBufferSize, "AAudioStream_setBufferSizeInFrames"},
		{&l.streamGetXRunCount, "AAudioStream_getXRunCount"},
		{&l.streamGetTimestamp, "AAudioStream_getTimestamp"},
		{&l.convertResultToText, "AAudio_convertResultToText"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			md := map[string]any{}
			if appErr, ok := errors.FromError(err); ok {
				md = appErr.Metadata
			}
			return nil, errors.New(platform.TagAudioUnavailable, "libaaudio lacks a function", "symbol", md["symbol"])
		}
	}
	// API 28 and later.
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.builderSetUsage, "AAudioStreamBuilder_setUsage"},
		{&l.builderSetContentType, "AAudioStreamBuilder_setContentType"},
		{&l.builderSetInputPreset, "AAudioStreamBuilder_setInputPreset"},
	} {
		if lib.Has(b.symbol) {
			if err := lib.Bind(b.fptr, b.symbol); err != nil {
				return nil, errors.New(platform.TagAudioUnavailable, "libaaudio lacks a function", "symbol", b.symbol)
			}
		}
	}
	return l, nil
}

// text is AAudio's own text for a result.
func (l *library) text(result int32) string {
	if l.convertResultToText == nil {
		return ""
	}
	return dl.GoString(l.convertResultToText(result))
}
