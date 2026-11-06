package gluttertest

import (
	"context"

	"github.com/rm4n0s/glutter/audio"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
)

// TestAudio is the scripted audio service of the test platform (SPEC §28.8): it has a speaker and a microphone, which are the
// default devices, and a test adds and removes devices and plays the part of the audio thread: it pulls frames from the playback
// device that the app opened, or pushes frames into a capture device, and the app's streams run in the test's goroutine as they
// would on the backend's thread. Nothing plays by itself, so the test decides when and for how long.
//
// Flutter: no counterpart
type TestAudio struct {
	t *Tester
	a *fake.Audio
}

// Audio returns the scripted audio service of the tester.
//
// Flutter: no counterpart
func (t *Tester) Audio() *TestAudio { return &TestAudio{t: t, a: t.audiosvc} }

// AddDevice plugs in a device, as the system reports it; a device with Default set becomes the default of its kind. The listeners
// of audio.System have run when it returns.
//
// Flutter: no counterpart
func (a *TestAudio) AddDevice(info audio.DeviceInfo) {
	a.t.adopt()
	a.a.AddDevice(platform.AudioDeviceInfo{ID: info.ID, Name: info.Name, Kind: platform.AudioKind(info.Kind), Default: info.Default,
		Format: platform.AudioFormat{Rate: info.Format.Rate, Channels: info.Format.Channels}})
	a.t.host.RunPosted()
}

// RemoveDevice unplugs a device: an open device of it is lost. The listeners have run when it returns.
//
// Flutter: no counterpart
func (a *TestAudio) RemoveDevice(id string) {
	a.t.adopt()
	a.a.RemoveDevice(id)
	a.t.host.RunPosted()
}

// SetDefault makes a device the default of its kind. The listeners have run when it returns.
//
// Flutter: no counterpart
func (a *TestAudio) SetDefault(id string) {
	a.t.adopt()
	a.a.SetDefault(id)
	a.t.host.RunPosted()
}

// SetUnavailable makes the platform act as one whose audio server does not run: the lists are empty and the opens fail with
// AudioUnavailable.
//
// Flutter: no counterpart
func (a *TestAudio) SetUnavailable(v bool) { a.a.SetUnavailable(v) }

// FailOpen makes the next opens fail with err; nil makes them work again.
//
// Flutter: no counterpart
func (a *TestAudio) FailOpen(err error) { a.a.FailOpen(err) }

// Devices returns the devices that the app has open now, in the order it opened them.
//
// Flutter: no counterpart
func (a *TestAudio) Devices() []*TestAudioDevice {
	var out []*TestAudioDevice
	for _, d := range a.a.Opened() {
		out = append(out, &TestAudioDevice{t: a.t, d: d})
	}
	return out
}

// Playback returns the first playback device that the app has open, and fails the test when there is none.
//
// Flutter: no counterpart
func (a *TestAudio) Playback() *TestAudioDevice {
	a.t.tb.Helper()
	return a.first(platform.AudioPlayback)
}

// Capture returns the first capture device that the app has open, and fails the test when there is none.
//
// Flutter: no counterpart
func (a *TestAudio) Capture() *TestAudioDevice {
	a.t.tb.Helper()
	return a.first(platform.AudioCapture)
}

func (a *TestAudio) first(kind platform.AudioKind) *TestAudioDevice {
	a.t.tb.Helper()
	for _, d := range a.a.Opened() {
		if d.Info().Kind == kind {
			return &TestAudioDevice{t: a.t, d: d}
		}
	}
	a.t.tb.Fatalf("the app has no %v audio device open", audio.DeviceKind(kind))
	return nil
}

// Open is for tests of the platform side: it opens a device the way a backend's client does, without the audio package.
// Most tests open devices through audio.System instead.
//
// Flutter: no counterpart
func (a *TestAudio) Open(ctx context.Context, kind audio.DeviceKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	return a.a.Open(ctx, platform.AudioKind(kind), opts, h)
}

// TestAudioDevice is a device that the app opened, seen from the side of the backend.
//
// Flutter: no counterpart
type TestAudioDevice struct {
	t *Tester
	d *fake.AudioDevice
}

// ID is the ID of the system's device that the app opened.
//
// Flutter: no counterpart
func (d *TestAudioDevice) ID() string { return d.d.Info().ID }

// Format is the format that the device works in.
//
// Flutter: no counterpart
func (d *TestAudioDevice) Format() audio.Format {
	f := d.d.Format()
	return audio.Format{Rate: f.Rate, Channels: f.Channels}
}

// BufferFrames and PauseInBackground are what the app asked for when it opened the device.
//
// Flutter: no counterpart
func (d *TestAudioDevice) BufferFrames() int { return d.d.Options().BufferFrames }

// PauseInBackground says whether the app asked for the device to pause when it is hidden.
//
// Flutter: no counterpart
func (d *TestAudioDevice) PauseInBackground() bool { return d.d.Options().PauseInBackground }

// Pull runs the audio thread's callback of a playback device for frames frames, in pieces of at most chunk frames (0 for one
// piece), and returns what it played, interleaved. A device that is paused, suspended or lost plays silence.
//
// Flutter: no counterpart
func (d *TestAudioDevice) Pull(frames, chunk int) []float32 { return d.d.Pull(frames, chunk) }

// PullInto does the same for len(out)/channels frames into out, without allocating.
//
// Flutter: no counterpart
func (d *TestAudioDevice) PullInto(out []float32) { d.d.PullInto(out) }

// Push gives the streams of a capture device the frames that it recorded, interleaved in the device's format.
//
// Flutter: no counterpart
func (d *TestAudioDevice) Push(frames []float32) { d.d.Push(frames) }

// Lose makes the device lost, as when it is unplugged. The listeners have run when it returns.
//
// Flutter: no counterpart
func (d *TestAudioDevice) Lose() {
	d.t.adopt()
	d.d.Lose()
	d.t.host.RunPosted()
}

// Suspend makes the device wait for the user, as a browser's does; Unsuspend lets it run. The listeners have run when they return.
//
// Flutter: no counterpart
func (d *TestAudioDevice) Suspend() {
	d.t.adopt()
	d.d.Suspend()
	d.t.host.RunPosted()
}

// Unsuspend lets a suspended device run.
//
// Flutter: no counterpart
func (d *TestAudioDevice) Unsuspend() {
	d.t.adopt()
	d.d.Unsuspend()
	d.t.host.RunPosted()
}
