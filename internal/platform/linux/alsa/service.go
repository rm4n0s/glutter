//go:build linux && !android

// Package alsa is the ALSA backend of the audio service on Linux (SPEC §28.3, step 31.5), for the systems where no PulseAudio (or
// PipeWire) server answers: playback and capture of float32 frames through libasound's PCM interface, on a goroutine of its own
// for each device. It has no hotplug: the list of devices is read when it is asked for. libasound is loaded when it is first
// needed. The design follows the ALSA driver of SDL 3 (src/audio/alsa/SDL_alsa_audio.c; zlib license, third_party/sdl/LICENSE),
// translated to Go and changed: the stream is set with snd_pcm_set_params, the loop waits for the device with a timeout so that
// pausing and closing do not wait for a device that is stuck, and an underrun is recovered with snd_pcm_recover.
//
// Flutter: no counterpart
package alsa

import (
	"context"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The defaults when the app asks for none: the latency of the stream (a target of SPEC §28.6 is 30 ms at most).
const (
	defaultLatency  = 20 * time.Millisecond
	defaultRate     = 48000
	defaultChannels = 2
	maxChannels     = 8
	// waitTimeout is how long the loop waits for the device before it looks at its own state again.
	waitTimeout = 20 * time.Millisecond
	// defaultName is the PCM that ALSA's configuration points at the sound system's default device.
	defaultName = "default"
)

// Service is the ALSA backend: it implements platform.Audio.
//
// Flutter: no counterpart
type Service struct {
	load func() (*library, error)

	mu  sync.Mutex
	lib *library
}

// New makes a Service. libasound is not loaded yet.
//
// Flutter: no counterpart
func New() *Service { return &Service{load: loadLibrary} }

var _ platform.Audio = (*Service)(nil)

// getLib loads libasound on first use. It fails with AudioUnavailable when the library cannot be loaded.
func (s *Service) getLib() (*library, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lib == nil {
		lib, err := s.load()
		if err != nil {
			return nil, err
		}
		s.lib = lib
	}
	return s.lib, nil
}

// SetChangeHandler does nothing: ALSA has no notification of devices that come and go (SPEC §28.3).
//
// Flutter: no counterpart
func (s *Service) SetChangeHandler(fn func()) {}

// Devices lists what ALSA's configuration offers as PCM devices: the default one, and the system default of each card.
//
// Flutter: no counterpart
func (s *Service) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	lib, err := s.getLib()
	if err != nil {
		return nil, err
	}
	iface := dl.CString("pcm")
	defer dl.Free(iface)
	out := dl.Malloc(8)
	defer dl.Free(out)
	if lib.nameHint(-1, iface, out) < 0 {
		// A system whose configuration cannot be read has the default device and nothing else.
		return []platform.AudioDeviceInfo{{ID: defaultName, Name: "Default", Kind: kind, Default: true}}, nil
	}
	hints := *(*uintptr)(dl.Pointer(out))
	defer lib.nameFreeHint(hints)

	cName, cDesc, cIO := dl.CString("NAME"), dl.CString("DESC"), dl.CString("IOID")
	defer dl.Free(cName)
	defer dl.Free(cDesc)
	defer dl.Free(cIO)
	text := func(hint, id uintptr) string {
		p := lib.nameGetHint(hint, id)
		if p == 0 {
			return ""
		}
		defer lib.free(p)
		return dl.GoString(p)
	}

	var list []platform.AudioDeviceInfo
	for i := 0; ; i++ {
		hint := *(*uintptr)(dl.Pointer(hints + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if hint == 0 {
			break
		}
		name := text(hint, cName)
		if name != defaultName && !strings.HasPrefix(name, "sysdefault:") {
			continue
		}
		// IOID is "Input" or "Output" for a device that goes one way, and not there for one that goes both.
		switch io := text(hint, cIO); {
		case io == "Input" && kind == platform.AudioPlayback, io == "Output" && kind == platform.AudioCapture:
			continue
		}
		desc, _, _ := strings.Cut(text(hint, cDesc), "\n")
		if name == defaultName && desc == "" {
			desc = "Default"
		}
		list = append(list, platform.AudioDeviceInfo{ID: name, Name: desc, Kind: kind, Default: name == defaultName})
	}
	if len(list) == 0 {
		list = []platform.AudioDeviceInfo{{ID: defaultName, Name: "Default", Kind: kind, Default: true}}
	}
	return list, nil
}

// errnoText is libasound's text for the negative error code of a call.
func (l *library) errnoText(code int32) string { return dl.GoString(l.strerror(code)) }

// Open opens a device and starts its goroutine. It fails with AudioUnavailable when libasound cannot be loaded,
// AudioFormatUnsupported when the device takes no such format, AudioCapturePermissionDenied when the system refuses the recording
// and AudioDeviceOpenFailed for the rest.
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
	format := opts.Format
	if format.Rate == 0 {
		format.Rate = defaultRate
	}
	if format.Channels == 0 {
		format.Channels = defaultChannels
	}
	if format.Rate < 1 || format.Channels < 1 || format.Channels > maxChannels {
		return nil, errors.New(platform.TagAudioFormatUnsupported, "ALSA devices take 1 to 8 channels and a positive rate",
			"rate", format.Rate, "channels", format.Channels)
	}
	name := opts.ID
	if name == "" {
		name = defaultName
	}
	latency := defaultLatency
	if opts.BufferFrames > 0 {
		latency = time.Duration(int64(opts.BufferFrames) * int64(time.Second) / int64(format.Rate))
	}

	cName := dl.CString(name)
	defer dl.Free(cName)
	scratch := dl.Malloc(16)
	defer dl.Free(scratch)
	stream := int32(streamPlayback)
	if kind == platform.AudioCapture {
		stream = streamCapture
	}
	if rc := lib.open(scratch, cName, stream, openNonblock); rc < 0 {
		if kind == platform.AudioCapture && (rc == -errAccess || rc == -errPerm) {
			return nil, errors.New(platform.TagAudioCapturePermissionDenied, "the system does not allow the recording",
				"id", name, "reason", lib.errnoText(rc))
		}
		return nil, errors.New(platform.TagAudioDeviceOpenFailed, "the ALSA device cannot be opened", "id", name, "reason", lib.errnoText(rc))
	}
	pcm := *(*uintptr)(dl.Pointer(scratch))
	// soft_resample lets the plug layer convert what the hardware does not take, so any rate and channel count is accepted.
	if rc := lib.setParams(pcm, formatFloatLE, accessRWInter, uint32(format.Channels), uint32(format.Rate), 1, uint32(latency/time.Microsecond)); rc < 0 {
		lib.close(pcm)
		if rc == -errInval {
			return nil, errors.New(platform.TagAudioFormatUnsupported, "the ALSA device does not take the format",
				"id", name, "rate", format.Rate, "channels", format.Channels, "reason", lib.errnoText(rc))
		}
		return nil, errors.New(platform.TagAudioDeviceOpenFailed, "the ALSA device does not take its parameters", "id", name, "reason", lib.errnoText(rc))
	}
	period := int(max(int64(format.Rate)*int64(latency)/int64(time.Second)/2, 32))
	if lib.getParams(pcm, scratch, scratch+8) == 0 {
		if got := int(*(*uint64)(dl.Pointer(scratch + 8))); got > 0 {
			period = got
		}
	}

	d := newDevice(lib, pcm, kind, h, format, period)
	go d.run()
	return d, nil
}
