//go:build linux && !android

package linux

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// fakeAudio is a backend whose answers are set by the test.
type fakeAudio struct {
	name     string
	err      error
	devices  []platform.AudioDeviceInfo
	opened   int
	listed   int
	handlers []func()
}

type fakeDevice struct{ platform.AudioDevice }

func (f *fakeAudio) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	f.listed++
	return f.devices, f.err
}

func (f *fakeAudio) SetChangeHandler(fn func()) { f.handlers = append(f.handlers, fn) }

func (f *fakeAudio) Open(ctx context.Context, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	f.opened++
	if f.err != nil {
		return nil, f.err
	}
	return fakeDevice{}, nil
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// unavailableErr and failedErr are the errors that a backend makes; the tags are the platform's.
func unavailableErr() error {
	return errors.New(platform.TagAudioUnavailable, "no server")
}

func failedErr() error {
	return errors.New(platform.TagAudioDeviceOpenFailed, "no such device")
}

func TestAudioChoice(t *testing.T) {
	tests := []struct {
		name       string
		pulseErr   error
		alsaErr    error
		wantErr    string // the tag, or "" for success
		pulseCalls int
		alsaCalls  int
	}{
		{"PulseAudio is there", nil, nil, "", 1, 0},
		{"PulseAudio is not there, ALSA is", unavailableErr(), nil, "", 1, 1},
		{"neither is there", unavailableErr(), unavailableErr(), platform.TagAudioUnavailable, 1, 1},
		{"PulseAudio fails for another reason: ALSA is not asked", failedErr(), nil, platform.TagAudioDeviceOpenFailed, 1, 0},
		{"ALSA fails for another reason", unavailableErr(), failedErr(), platform.TagAudioDeviceOpenFailed, 1, 1},
	}
	for _, tt := range tests {
		for _, call := range []string{"Devices", "Open"} {
			t.Run(tt.name+"/"+call, func(t *testing.T) {
				p := &fakeAudio{name: "pulse", err: tt.pulseErr}
				a := &fakeAudio{name: "alsa", err: tt.alsaErr}
				c := newAudioChoice(quiet(), audioBackend{"pulse", p}, audioBackend{"alsa", a})
				var err error
				if call == "Devices" {
					_, err = c.Devices(platform.AudioPlayback)
				} else {
					_, err = c.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, nil)
				}
				if tt.wantErr == "" {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					appErr, ok := errors.FromError(err)
					if !ok || appErr.Tag != tt.wantErr {
						t.Fatalf("got %v, want tag %s", err, tt.wantErr)
					}
				}
				calls := func(f *fakeAudio) int {
					if call == "Devices" {
						return f.listed
					}
					return f.opened
				}
				if calls(p) != tt.pulseCalls || calls(a) != tt.alsaCalls {
					t.Errorf("PulseAudio was asked %d times and ALSA %d times, want %d and %d", calls(p), calls(a), tt.pulseCalls, tt.alsaCalls)
				}
			})
		}
	}
}

func TestAudioChoiceRoutes(t *testing.T) {
	c := newAudioChoice(quiet(), audioBackend{"pulse", &fakeAudio{err: unavailableErr()}})
	_, err := c.Devices(platform.AudioCapture)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("audioChoice.Devices.AudioUnavailable") {
		t.Errorf("Devices: %v", err)
	}
	_, err = c.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{}, nil)
	appErr, ok = errors.FromError(err)
	if !ok || !appErr.HasRoute("audioChoice.Open.AudioUnavailable") {
		t.Errorf("Open: %v", err)
	}
	// No backend at all is the same failure.
	_, err = newAudioChoice(quiet()).Devices(platform.AudioPlayback)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != platform.TagAudioUnavailable {
		t.Errorf("no backends: %v", err)
	}
}

func TestAudioChoiceDevicesAreTheBackends(t *testing.T) {
	want := []platform.AudioDeviceInfo{{ID: "default", Name: "Default", Default: true}}
	c := newAudioChoice(quiet(), audioBackend{"pulse", &fakeAudio{err: unavailableErr()}}, audioBackend{"alsa", &fakeAudio{devices: want}})
	got, err := c.Devices(platform.AudioPlayback)
	if err != nil || len(got) != 1 || got[0] != want[0] {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestAudioChoiceChangeHandler(t *testing.T) {
	p, a := &fakeAudio{}, &fakeAudio{}
	c := newAudioChoice(quiet(), audioBackend{"pulse", p}, audioBackend{"alsa", a})
	c.SetChangeHandler(func() {})
	c.SetChangeHandler(nil)
	if len(p.handlers) != 2 || len(a.handlers) != 2 || p.handlers[0] == nil || p.handlers[1] != nil {
		t.Errorf("handlers: pulse %d, alsa %d", len(p.handlers), len(a.handlers))
	}
}

func TestLinuxAudioBackendEnv(t *testing.T) {
	tests := []struct {
		value string
		want  []string
	}{
		{"", []string{"pulse", "alsa"}},
		{"auto", []string{"pulse", "alsa"}},
		{"pulse", []string{"pulse"}},
		{"ALSA", []string{"alsa"}},
		{"jack", nil},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			c := linuxAudio(func(k string) string {
				if k != AudioBackendEnv {
					t.Errorf("read %s", k)
				}
				return tt.value
			}, quiet())
			var got []string
			for _, b := range c.backends {
				got = append(got, b.name)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("backends %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("backends %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestServicesHaveAudio(t *testing.T) {
	s := NewServices(func(func()) {})
	if s.Audio() == nil {
		t.Fatal("the services have no audio")
	}
	var nilServices *Services
	if nilServices.Audio() != nil {
		t.Error("nil services have audio")
	}
}
