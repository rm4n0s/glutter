package fake

import (
	"context"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Audio is the scripted audio service of the fake platform: a test lists devices, opens them through the app, and plays the part
// of the audio thread by pulling frames from a playback device or pushing frames into a capture device (SPEC §28.8). Nothing runs
// by itself, so a test decides when the callback runs and what the clock of the device is.
//
// Flutter: no counterpart
type Audio struct {
	mu          sync.Mutex
	devices     []platform.AudioDeviceInfo
	onChange    func()
	opened      []*AudioDevice
	unavailable bool
	openErr     error
	releases    []func()
}

// The formats that the fake devices have by default.
var (
	defaultPlayback = platform.AudioFormat{Rate: 48000, Channels: 2}
	defaultCapture  = platform.AudioFormat{Rate: 48000, Channels: 1}
)

// NewAudio makes the service with a speaker and a microphone, which are the default devices.
//
// Flutter: no counterpart
func NewAudio() *Audio {
	return &Audio{devices: []platform.AudioDeviceInfo{
		{ID: "fake-speakers", Name: "Fake speakers", Kind: platform.AudioPlayback, Default: true, Format: defaultPlayback},
		{ID: "fake-microphone", Name: "Fake microphone", Kind: platform.AudioCapture, Default: true, Format: defaultCapture},
	}}
}

// Audio returns the scripted audio service, which Services gives the engine.
//
// Flutter: no counterpart
func (p *Platform) Audio() *Audio {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.audio == nil {
		p.audio = NewAudio()
	}
	return p.audio
}

// SetUnavailable makes the service act as a system with no audio server: the lists and the opens fail with AudioUnavailable.
//
// Flutter: no counterpart
func (a *Audio) SetUnavailable(v bool) {
	a.mu.Lock()
	a.unavailable = v
	a.mu.Unlock()
}

// FailOpen makes the next opens fail with err (nil makes them work again).
//
// Flutter: no counterpart
func (a *Audio) FailOpen(err error) {
	a.mu.Lock()
	a.openErr = err
	a.mu.Unlock()
}

// Devices implements platform.Audio.
func (a *Audio) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.unavailable {
		return nil, errors.New(platform.TagAudioUnavailable, "the fake audio service is set to have no audio server")
	}
	var out []platform.AudioDeviceInfo
	for _, d := range a.devices {
		if d.Kind == kind {
			out = append(out, d)
		}
	}
	return out, nil
}

// SetChangeHandler implements platform.Audio.
func (a *Audio) SetChangeHandler(fn func()) {
	a.mu.Lock()
	a.onChange = fn
	a.mu.Unlock()
}

// changed tells the handler, outside the lock.
func (a *Audio) changed() {
	a.mu.Lock()
	fn := a.onChange
	a.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// AddDevice plugs in a device; a device with Default set becomes the default of its kind. The change handler is called before it
// returns.
//
// Flutter: no counterpart
func (a *Audio) AddDevice(info platform.AudioDeviceInfo) {
	a.mu.Lock()
	if info.Default {
		a.clearDefault(info.Kind)
	}
	a.devices = append(a.devices, info)
	a.mu.Unlock()
	a.changed()
}

// clearDefault takes the default away from the devices of a kind. The lock is held.
func (a *Audio) clearDefault(kind platform.AudioKind) {
	for i := range a.devices {
		if a.devices[i].Kind == kind {
			a.devices[i].Default = false
		}
	}
}

// RemoveDevice unplugs a device; an open device of it is lost.
//
// Flutter: no counterpart
func (a *Audio) RemoveDevice(id string) {
	a.mu.Lock()
	var gone []*AudioDevice
	for i, d := range a.devices {
		if d.ID == id {
			a.devices = append(a.devices[:i:i], a.devices[i+1:]...)
			break
		}
	}
	for _, d := range a.opened {
		if d.info.ID == id {
			gone = append(gone, d)
		}
	}
	a.mu.Unlock()
	for _, d := range gone {
		d.Lose()
	}
	a.changed()
}

// SetDefault makes a device the default of its kind.
//
// Flutter: no counterpart
func (a *Audio) SetDefault(id string) {
	a.mu.Lock()
	for i := range a.devices {
		if a.devices[i].ID == id {
			a.clearDefault(a.devices[i].Kind)
			a.devices[i].Default = true
		}
	}
	a.mu.Unlock()
	a.changed()
}

// Opened returns the devices that are open now, in the order they were opened.
//
// Flutter: no counterpart
func (a *Audio) Opened() []*AudioDevice {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]*AudioDevice(nil), a.opened...)
}

// Open implements platform.Audio.
func (a *Audio) Open(ctx context.Context, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.unavailable {
		return nil, errors.New(platform.TagAudioUnavailable, "the fake audio service is set to have no audio server")
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagAudioDeviceOpenFailed, err).SetMetadata("device", opts.ID)
	}
	if a.openErr != nil {
		return nil, a.openErr
	}
	var info platform.AudioDeviceInfo
	found := false
	for _, d := range a.devices {
		if d.Kind == kind && ((opts.ID == "" && d.Default) || (opts.ID != "" && d.ID == opts.ID)) {
			info, found = d, true
			break
		}
	}
	if !found {
		return nil, errors.New(platform.TagAudioDeviceOpenFailed, "the fake audio service has no such device", "device", opts.ID)
	}
	format := info.Format
	if opts.Format.Rate > 0 {
		format.Rate = opts.Format.Rate
	}
	if opts.Format.Channels > 0 {
		format.Channels = opts.Format.Channels
	}
	d := &AudioDevice{owner: a, info: info, kind: kind, opts: opts, format: format, handler: h, state: platform.AudioRunning}
	a.opened = append(a.opened, d)
	return d, nil
}

// AudioDevice is an open device of the fake service. A test pulls frames from a playback device and pushes frames into a capture
// device, which calls the handler on the goroutine of the test, as the audio thread of a real backend would.
//
// Flutter: no counterpart
type AudioDevice struct {
	owner  *Audio
	info   platform.AudioDeviceInfo
	kind   platform.AudioKind
	opts   platform.AudioOpenOptions
	format platform.AudioFormat

	// mu is held while the handler runs, so Close waits for the callback that is under way.
	mu      sync.Mutex
	handler platform.AudioHandler
	state   platform.AudioState
}

var _ platform.AudioDevice = (*AudioDevice)(nil)

// Info is the description of the device that was opened.
//
// Flutter: no counterpart
func (d *AudioDevice) Info() platform.AudioDeviceInfo { return d.info }

// Options are the options the device was opened with.
//
// Flutter: no counterpart
func (d *AudioDevice) Options() platform.AudioOpenOptions { return d.opts }

// Format implements platform.AudioDevice.
func (d *AudioDevice) Format() platform.AudioFormat { return d.format }

// State is the state of the device.
//
// Flutter: no counterpart
func (d *AudioDevice) State() platform.AudioState {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.state
}

// PullInto runs the callback of a playback device once for len(out)/channels frames and puts the frames in out. A device that is
// not running gives silence and the handler is not called. It does not allocate.
//
// Flutter: no counterpart
func (d *AudioDevice) PullInto(out []float32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != platform.AudioRunning || d.handler == nil {
		clear(out)
		return
	}
	d.handler.Fill(out)
}

// Pull runs the callback of a playback device for frames frames, in pieces of at most chunk frames (0 for one piece), and returns
// what came out.
//
// Flutter: no counterpart
func (d *AudioDevice) Pull(frames, chunk int) []float32 {
	if chunk <= 0 {
		chunk = frames
	}
	ch := d.format.Channels
	out := make([]float32, 0, frames*ch)
	buf := make([]float32, chunk*ch)
	for frames > 0 {
		n := min(frames, chunk)
		d.PullInto(buf[:n*ch])
		out = append(out, buf[:n*ch]...)
		frames -= n
	}
	return out
}

// Push gives the handler of a capture device the frames that the device recorded; a device that is not running drops them.
//
// Flutter: no counterpart
func (d *AudioDevice) Push(frames []float32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != platform.AudioRunning || d.handler == nil {
		return
	}
	d.handler.Capture(frames)
}

// setState changes the state and tells the handler, outside the lock.
func (d *AudioDevice) setState(s platform.AudioState) {
	d.mu.Lock()
	if d.state == platform.AudioClosed || d.state == s {
		d.mu.Unlock()
		return
	}
	d.state = s
	h := d.handler
	d.mu.Unlock()
	if h != nil {
		h.StateChanged(s)
	}
}

// Lose makes the device lost, as when it is unplugged.
//
// Flutter: no counterpart
func (d *AudioDevice) Lose() { d.setState(platform.AudioLost) }

// Suspend makes the device wait, as a browser's does until the user does something.
//
// Flutter: no counterpart
func (d *AudioDevice) Suspend() { d.setState(platform.AudioSuspended) }

// Unsuspend lets a suspended device run.
//
// Flutter: no counterpart
func (d *AudioDevice) Unsuspend() {
	d.mu.Lock()
	suspended := d.state == platform.AudioSuspended
	d.mu.Unlock()
	if suspended {
		d.setState(platform.AudioRunning)
	}
}

// Pause implements platform.AudioDevice.
func (d *AudioDevice) Pause() error {
	d.mu.Lock()
	if d.state == platform.AudioRunning {
		d.state = platform.AudioPaused
	}
	d.mu.Unlock()
	return nil
}

// Resume implements platform.AudioDevice.
func (d *AudioDevice) Resume() error {
	d.mu.Lock()
	if d.state == platform.AudioPaused {
		d.state = platform.AudioRunning
	}
	d.mu.Unlock()
	return nil
}

// Close implements platform.AudioDevice: it waits for a callback that is under way, and the handler is never called again.
func (d *AudioDevice) Close() error {
	d.mu.Lock()
	d.state = platform.AudioClosed
	d.handler = nil
	d.mu.Unlock()
	a := d.owner
	a.mu.Lock()
	for i, o := range a.opened {
		if o == d {
			a.opened = append(a.opened[:i:i], a.opened[i+1:]...)
			break
		}
	}
	a.mu.Unlock()
	return nil
}

// AddReleaseHook is called by the host's own hook list in tests of the engine; here it only keeps the functions so that a test
// can run them with RunReleaseHooks, as the engine does when the app ends.
//
// Flutter: no counterpart
func (a *Audio) AddReleaseHook(fn func()) {
	a.mu.Lock()
	a.releases = append(a.releases, fn)
	a.mu.Unlock()
}

// RunReleaseHooks runs the functions of AddReleaseHook, the way the engine does when the app ends.
//
// Flutter: no counterpart
func (a *Audio) RunReleaseHooks() {
	a.mu.Lock()
	hooks := a.releases
	a.releases = nil
	a.mu.Unlock()
	for _, fn := range hooks {
		fn()
	}
}
