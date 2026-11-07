package audio

import (
	"context"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/widgets"
)

// DeviceOpts are the options of OpenPlayback and OpenCapture.
//
// Flutter: no counterpart
type DeviceOpts struct {
	// ID is the device to open (DeviceInfo.ID); empty is the default device, which the device follows when the system's default
	// changes.
	ID string
	// Format is the format the app would like the device to work in; a zero field leaves it to the backend. The device may work
	// in another format (Device.Format says); the streams are converted either way.
	Format Format
	// BufferFrames is the size of the device's buffer in frames; 0 lets the backend choose a latency within the targets of SPEC §28.6.
	BufferFrames int
	// PauseInBackground makes the device pause while the app is hidden. The default is true on Android and false elsewhere.
	PauseInBackground foundation.Opt[bool]
}

// System is the audio of the app: the devices of the system, and the devices that the app opened. Get it with Of. It is made when it
// is first asked for, and closed with the app.
//
// Flutter: no counterpart
type System struct {
	plat   platform.Audio
	post   func(func())
	thread *foundation.UIThread
	report foundation.ErrorReporter

	notifier foundation.ChangeNotifier

	mu      sync.Mutex
	devices map[*Device]struct{}
	closed  bool
}

type systemKey struct{}

// Of returns the audio of the app (the one of the binding). An app on a platform that has no audio service gets a System whose
// calls fail with AudioUnavailable.
//
// Flutter: no counterpart
func Of(bc widgets.BuildContext) *System { return ForBinding(widgets.WidgetsBindingOf(bc)) }

// ForBinding returns the audio of the binding, for code that has no BuildContext.
//
// Flutter: no counterpart
func ForBinding(b *widgets.WidgetsBinding) *System {
	return b.Extension(systemKey{}, func() any {
		var host platform.AudioHost
		var plat platform.Audio
		if b.PlatformDispatcher().HostAs(&host) {
			plat = host.Audio()
		}
		s := newSystem(plat, b.Post, b.UIThread(), b.ErrorReporter())
		if host != nil && plat != nil {
			host.AddReleaseHook(s.Close)
		}
		return s
	}).(*System)
}

func newSystem(plat platform.Audio, post func(func()), thread *foundation.UIThread, report foundation.ErrorReporter) *System {
	s := &System{plat: plat, post: post, thread: thread, report: report, devices: map[*Device]struct{}{}}
	s.notifier.SetUIThread(thread)
	s.notifier.SetErrorReporter(report)
	if plat != nil {
		plat.SetChangeHandler(func() { post(func() { s.notifier.NotifyListeners() }) })
	}
	return s
}

// Available says whether the platform has an audio service.
//
// Flutter: no counterpart
func (s *System) Available() bool { return s.plat != nil }

// Devices lists the devices of a kind that the system has now; it is empty when the platform has no audio service or no server runs.
//
// Flutter: no counterpart
func (s *System) Devices(kind DeviceKind) []DeviceInfo {
	if s.plat == nil {
		return nil
	}
	list, err := s.plat.Devices(platform.AudioKind(kind))
	if err != nil {
		return nil
	}
	out := make([]DeviceInfo, len(list))
	for i, d := range list {
		out[i] = DeviceInfo{ID: d.ID, Name: d.Name, Kind: DeviceKind(d.Kind), Default: d.Default, Format: formatOf(d.Format)}
	}
	return out
}

// AddListener adds a function that is called, on the UI thread, when the list of devices or the default device changed. It returns
// the function that removes it.
//
// Flutter: no counterpart
func (s *System) AddListener(fn func()) (remove func()) {
	h := s.notifier.AddListener(fn)
	return func() { s.notifier.RemoveListener(h) }
}

// OpenPlayback opens a playback device: the default one, which it follows, or DeviceOpts.ID. It fails with AudioUnavailable when
// the platform has no audio service or no server runs, AudioDeviceOpenFailed when the device cannot be opened (also when ctx ends
// first) and AudioFormatUnsupported when no format works. At most one DeviceOpts may be given.
//
// Flutter: no counterpart
func (s *System) OpenPlayback(ctx context.Context, opts ...DeviceOpts) (*Device, error) {
	return s.open(ctx, Playback, opts)
}

// OpenCapture opens a capture device, as OpenPlayback does a playback one. It also fails with AudioCapturePermissionDenied when
// the user does not allow the recording. Where the system asks the user (Android, the web) the call waits for the answer, until ctx
// ends: call it from a goroutine that is not the UI one.
//
// Flutter: no counterpart
func (s *System) OpenCapture(ctx context.Context, opts ...DeviceOpts) (*Device, error) {
	return s.open(ctx, Capture, opts)
}

func (s *System) open(ctx context.Context, kind DeviceKind, opts []DeviceOpts) (*Device, error) {
	if len(opts) > 1 {
		panic(errors.New(tagMultipleOptions, "OpenPlayback and OpenCapture take at most one DeviceOpts", "count", len(opts)))
	}
	var o DeviceOpts
	if len(opts) == 1 {
		o = opts[0]
	}
	if s.plat == nil {
		return nil, errors.New(TagUnavailable, "the platform has no audio service")
	}
	if (o.Format.Rate != 0 && (o.Format.Rate < MinRate || o.Format.Rate > MaxRate)) ||
		(o.Format.Channels != 0 && (o.Format.Channels < 1 || o.Format.Channels > MaxChannels)) || o.BufferFrames < 0 {
		return nil, errors.New(TagFormatUnsupported, "the format of a device has a rate from 1000 to 768000 and 1 to 8 channels", "rate", o.Format.Rate, "channels", o.Format.Channels)
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return nil, errors.New(TagUnavailable, "the audio of the app is closed")
	}
	d := &Device{sys: s, kind: kind}
	d.notifier.SetUIThread(s.thread)
	d.notifier.SetErrorReporter(s.report)
	d.gain.Store(math32Bits1)
	pause := o.PauseInBackground.OrElse(foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid)
	plat, err := s.plat.Open(ctx, platform.AudioKind(kind), platform.AudioOpenOptions{ID: o.ID, Format: o.Format.platform(), BufferFrames: o.BufferFrames, PauseInBackground: pause}, d)
	if err != nil {
		return nil, err
	}
	d.plat = plat
	d.format = formatOf(plat.Format())
	s.mu.Lock()
	s.devices[d] = struct{}{}
	s.mu.Unlock()
	return d, nil
}

// forget takes a closed device out of the list that Close closes.
func (s *System) forget(d *Device) {
	s.mu.Lock()
	delete(s.devices, d)
	s.mu.Unlock()
}

// Close closes every device that the app opened and stops listening to the system. The engine calls it when the app ends; after
// it, the opens fail with AudioUnavailable. It may be called more than once.
//
// Flutter: no counterpart
func (s *System) Close() {
	s.mu.Lock()
	s.closed = true
	devices := make([]*Device, 0, len(s.devices))
	for d := range s.devices {
		devices = append(devices, d)
	}
	s.mu.Unlock()
	for _, d := range devices {
		_ = d.Close() //nolint:errcheck // the app is ending: there is nobody to tell
	}
	if s.plat != nil {
		s.plat.SetChangeHandler(nil)
	}
}

// The tag of the error of a function that takes at most one options value.
const tagMultipleOptions = "MultipleOptions"
