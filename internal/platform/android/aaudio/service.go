//go:build linux

package aaudio

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The limits of a format (SPEC §28.3) and the default buffer: two bursts of the device.
const (
	minRate     = 1000
	maxRate     = 768000
	maxChannels = 8
	// startWait is how long Open waits for the stream to say that it started; a stream that is slow to start is not an error.
	startWait = 500 * time.Millisecond
	// stateWait is how long a pause waits for the stream to reach its state.
	stateWait = time.Second
)

// Permissions is what the backend needs of the Java side to record: the permission RECORD_AUDIO, which Android asks the user for
// while the app runs.
type Permissions interface {
	// RecordAudio makes sure that the app may record: it asks the user when the permission is not granted yet and waits for the answer
	// until ctx ends. It reports false when the user said no.
	RecordAudio(ctx context.Context) (granted bool, err error)
}

// Options are what New is given.
type Options struct {
	// Permissions asks for the permission to record; nil means that none is needed (tests, and a system that never asks).
	Permissions Permissions
	// Log gets what cannot be returned (nil discards it).
	Log *slog.Logger
}

// Service is the AAudio backend: it implements platform.Audio.
//
// Flutter: no counterpart
type Service struct {
	perm Permissions
	log  *slog.Logger
	load func() (*library, error)

	mu         sync.Mutex
	lib        *library
	devices    map[*device]struct{}
	background bool
	// scratch is C memory for the out parameters of the library's functions (an address that a Go variable lives at could move).
	scratch uintptr
}

var _ platform.Audio = (*Service)(nil)

// New makes a Service. libaaudio is not loaded yet.
//
// Flutter: no counterpart
func New(opts Options) *Service {
	log := opts.Log
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Service{perm: opts.Permissions, log: log, load: loadLibrary, devices: map[*device]struct{}{}}
}

// getLib loads libaaudio on first use. It fails with AudioUnavailable when the library cannot be loaded.
func (s *Service) getLib() (*library, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lib == nil {
		lib, err := s.load()
		if err != nil {
			return nil, err
		}
		s.lib = lib
		s.scratch = dl.Malloc(16)
	}
	return s.lib, nil
}

// Devices lists the default device of the kind: AAudio has no list of devices (the identifiers that the Java side's AudioManager
// gives are numbers, which Open takes as the ID of a device). It fails with AudioUnavailable when libaaudio is not there.
//
// Flutter: no counterpart
func (s *Service) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	if _, err := s.getLib(); err != nil {
		return nil, err
	}
	name := "Default output"
	if kind == platform.AudioCapture {
		name = "Default input"
	}
	return []platform.AudioDeviceInfo{{Name: name, Kind: kind, Default: true}}, nil
}

// SetChangeHandler is for the platforms that tell when the list of devices changes; AAudio's list is the default devices, and a stream
// that is on one that went away is made again on the new default one, so nothing is ever told.
//
// Flutter: no counterpart
func (s *Service) SetChangeHandler(fn func()) {}

// SetBackground says that the app is hidden (or shown again): the devices that were opened with PauseInBackground pause, and go on
// when the app is shown, unless their owner paused them.
//
// Flutter: no counterpart
func (s *Service) SetBackground(hidden bool) {
	s.mu.Lock()
	s.background = hidden
	list := make([]*device, 0, len(s.devices))
	for d := range s.devices {
		list = append(list, d)
	}
	s.mu.Unlock()
	for _, d := range list {
		d.setBackground(hidden)
	}
}

// forget drops a device that is closed.
func (s *Service) forget(d *device) {
	s.mu.Lock()
	delete(s.devices, d)
	s.mu.Unlock()
}

// Open opens a device and starts its stream. It fails with AudioUnavailable when libaaudio is not there,
// AudioCapturePermissionDenied when the user refuses the microphone, AudioFormatUnsupported when the device takes no such format, and
// AudioDeviceOpenFailed for the rest.
//
// Flutter: no counterpart
func (s *Service) Open(ctx context.Context, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagAudioDeviceOpenFailed, err).SetMetadata("id", opts.ID)
	}
	lib, err := s.getLib()
	if err != nil {
		return nil, err
	}
	if r, c := opts.Format.Rate, opts.Format.Channels; r < 0 || c < 0 || c > maxChannels || (r != 0 && (r < minRate || r > maxRate)) {
		return nil, errors.New(platform.TagAudioFormatUnsupported, "AAudio devices take 1 to 8 channels and a rate of 1000 to 768000",
			"rate", r, "channels", c)
	}
	id, ok := parseDeviceID(opts.ID)
	if !ok {
		return nil, errors.New(platform.TagAudioDeviceOpenFailed, "the ID of an AAudio device is the number of an Android audio device", "id", opts.ID)
	}
	if kind == platform.AudioCapture && s.perm != nil {
		granted, err := s.perm.RecordAudio(ctx)
		if err != nil {
			return nil, errors.NewErr(platform.TagAudioCapturePermissionDenied, err).SetMetadata("permission", "android.permission.RECORD_AUDIO")
		}
		if !granted {
			return nil, errors.New(platform.TagAudioCapturePermissionDenied, "the user did not allow the app to record",
				"permission", "android.permission.RECORD_AUDIO")
		}
	}
	d := &device{svc: s, lib: lib, kind: kind, handler: h, opts: opts, id: id}
	d.silent.Store(true)
	d.handle = register(d)
	stream, format, err := s.openStream(lib, d, opts.Format)
	if err != nil {
		unregister(d.handle)
		return nil, err
	}
	d.stream.Store(stream)
	d.format = format
	d.mu.Lock()
	s.mu.Lock()
	background := s.background && opts.PauseInBackground
	s.devices[d] = struct{}{}
	s.mu.Unlock()
	d.bgPaused = background
	d.applyLocked()
	started := d.running
	d.mu.Unlock()
	if started {
		d.waitStarted(startWait)
	}
	return d, nil
}

// openStream makes a stream for the device in the format (a zero field is the device's own) and returns it with the format it has.
// The stream is not started. It fails with AudioFormatUnsupported for a rate or a count of channels that the device does not take, and
// AudioDeviceOpenFailed for the rest.
func (s *Service) openStream(lib *library, d *device, want platform.AudioFormat) (uintptr, platform.AudioFormat, error) {
	s.mu.Lock()
	scratch := s.scratch
	s.mu.Unlock()
	if r := lib.createStreamBuilder(scratch); r != resultOK {
		return 0, platform.AudioFormat{}, resultError(lib, "AAudio_createStreamBuilder", r)
	}
	builder := *(*uintptr)(dl.Pointer(scratch))
	defer lib.builderDelete(builder)

	cbs := callbacks()
	direction := int32(directionOutput)
	if d.kind == platform.AudioCapture {
		direction = directionInput
	}
	lib.builderSetDirection(builder, direction)
	if d.id != 0 {
		lib.builderSetDeviceID(builder, d.id)
	}
	lib.builderSetSharingMode(builder, sharingModeShared)
	lib.builderSetFormat(builder, formatPCMFloat)
	lib.builderSetSampleRate(builder, int32(want.Rate))
	lib.builderSetChannelCount(builder, int32(want.Channels))
	lib.builderSetPerformanceMode(builder, performanceModeLowLatency)
	if d.kind == platform.AudioPlayback {
		if lib.builderSetUsage != nil {
			lib.builderSetUsage(builder, usageGame)
		}
		if lib.builderSetContentType != nil {
			lib.builderSetContentType(builder, contentTypeMusic)
		}
	} else if lib.builderSetInputPreset != nil {
		lib.builderSetInputPreset(builder, inputPresetGeneric)
	}
	if n := d.opts.BufferFrames; n > 0 {
		lib.builderSetBufferCapacityInFrames(builder, int32(n))
	}
	lib.builderSetDataCallback(builder, cbs.data, d.handle)
	lib.builderSetErrorCallback(builder, cbs.err, d.handle)
	if r := lib.builderOpenStream(builder, scratch); r != resultOK {
		return 0, platform.AudioFormat{}, resultError(lib, "AAudioStreamBuilder_openStream", r)
	}
	stream := *(*uintptr)(dl.Pointer(scratch))
	format := platform.AudioFormat{Rate: int(lib.streamGetSampleRate(stream)), Channels: int(lib.streamGetChannelCount(stream))}
	if f := lib.streamGetFormat(stream); f != formatPCMFloat || format.Rate <= 0 || format.Channels <= 0 || format.Channels > maxChannels {
		lib.streamClose(stream)
		return 0, platform.AudioFormat{}, errors.New(platform.TagAudioFormatUnsupported, "the AAudio stream does not give float frames in a format that the toolkit uses",
			"sample format", int(f), "rate", format.Rate, "channels", format.Channels)
	}
	// The buffer: two bursts keep the latency low and ride out a late callback; the app may ask for more (a larger buffer than the
	// stream's capacity is cut to it by AAudio).
	burst := lib.streamGetFramesPerBurst(stream)
	size := 2 * burst
	if n := int32(d.opts.BufferFrames); n > size {
		size = n
	}
	if d.kind == platform.AudioPlayback && size > 0 {
		lib.streamSetBufferSize(stream, size)
	}
	return stream, format, nil
}

// parseDeviceID reads the ID of a device: empty is the default device (0, which AAudio calls unspecified), else the number of an
// Android audio device.
func parseDeviceID(id string) (int32, bool) {
	if id == "" {
		return 0, true
	}
	var n int64
	for _, c := range id {
		if c < '0' || c > '9' || n > 1<<30 {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	return int32(n), n > 0
}

// resultError is the error of a function of AAudio that failed. It is one table: a rate, a format or an argument that the device does
// not take is AudioFormatUnsupported, and any other result is AudioDeviceOpenFailed; the metadata are "function", "result" and AAudio's
// "text". It is listed in errcheck.allow.
func resultError(lib *library, function string, result int32) error {
	switch result {
	case resultInvalidRate, resultInvalidFormat, resultOutOfRange, resultIllegalArgument:
		return errors.New(platform.TagAudioFormatUnsupported, "the AAudio device does not take the format",
			"function", function, "result", int(result), "text", lib.text(result))
	}
	return errors.New(platform.TagAudioDeviceOpenFailed, "the AAudio stream could not be opened",
		"function", function, "result", int(result), "text", lib.text(result))
}
