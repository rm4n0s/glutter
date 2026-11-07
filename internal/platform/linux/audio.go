//go:build linux && !android

package linux

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/alsa"
	"github.com/rm4n0s/glutter/internal/platform/linux/pulse"
)

// AudioBackendEnv names the environment variable that limits the audio backend: "pulse" (PulseAudio, and PipeWire's PulseAudio
// protocol) or "alsa". Unset, the backends are tried in that order, PulseAudio first (SPEC §28.3).
const AudioBackendEnv = "GLUTTER_AUDIO_BACKEND"

// audioBackend is one way to reach the sound system.
type audioBackend struct {
	name string
	svc  platform.Audio
}

// audioChoice is the audio service of Linux: it asks the backends in order, and a backend that is not there (its library cannot
// be loaded, or no server answers) passes the call on to the next one. When none is there the call fails with AudioUnavailable.
// The backends load their libraries and connect when they are used, so an app that makes no sound pays nothing.
type audioChoice struct {
	backends []audioBackend
	logger   *slog.Logger

	mu     sync.Mutex
	change func()
}

// newAudioChoice makes the service of the backends, which are tried in the order given.
func newAudioChoice(logger *slog.Logger, backends ...audioBackend) *audioChoice {
	return &audioChoice{backends: backends, logger: logger}
}

// linuxAudio makes the audio service of the system: PulseAudio and ALSA, or the one that GLUTTER_AUDIO_BACKEND names.
func linuxAudio(getenv func(string) string, logger *slog.Logger) *audioChoice {
	all := []audioBackend{{"pulse", pulse.New(pulse.Options{})}, {"alsa", alsa.New()}}
	switch want := strings.ToLower(getenv(AudioBackendEnv)); want {
	case "", "auto":
		return newAudioChoice(logger, all...)
	default:
		for _, b := range all {
			if b.name == want {
				return newAudioChoice(logger, b)
			}
		}
		logger.Warn("audio: the backend in "+AudioBackendEnv+" is not known, so none is used", "value", want)
		return newAudioChoice(logger)
	}
}

var _ platform.Audio = (*audioChoice)(nil)

// unavailable says that err is a backend that is not there, which passes the call on.
func unavailable(err error) bool {
	appErr, ok := errors.FromError(err)
	return ok && appErr.Tag == platform.TagAudioUnavailable
}

// SetChangeHandler gives the handler to every backend.
func (c *audioChoice) SetChangeHandler(fn func()) {
	c.mu.Lock()
	c.change = fn
	c.mu.Unlock()
	for _, b := range c.backends {
		b.svc.SetChangeHandler(fn)
	}
}

// Devices lists the devices of the first backend that is there. It fails with AudioUnavailable when none is.
func (c *audioChoice) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	for _, b := range c.backends {
		list, err := b.svc.Devices(kind)
		if err == nil {
			return list, nil
		}
		if !unavailable(err) {
			return nil, err
		}
		c.logger.Debug("audio: backend is not there", "backend", b.name, "err", err)
	}
	return nil, errors.New(platform.TagAudioUnavailable, "there is no PulseAudio server and no ALSA", "backends", len(c.backends))
}

// Open opens a device on the first backend that is there. It fails with AudioUnavailable when none is; the other errors are the
// backend's.
func (c *audioChoice) Open(ctx context.Context, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	for _, b := range c.backends {
		dev, err := b.svc.Open(ctx, kind, opts, h)
		if err == nil {
			return dev, nil
		}
		if !unavailable(err) {
			return nil, err
		}
		c.logger.Debug("audio: backend is not there", "backend", b.name, "err", err)
	}
	return nil, errors.New(platform.TagAudioUnavailable, "there is no PulseAudio server and no ALSA", "backends", len(c.backends))
}

// audioFromEnv is linuxAudio for the process: it reads the environment.
func audioFromEnv() *audioChoice { return linuxAudio(os.Getenv, slog.Default()) }
