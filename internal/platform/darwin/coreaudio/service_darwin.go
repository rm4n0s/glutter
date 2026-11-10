//go:build darwin

package coreaudio

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
)

// Permissions is what the backend needs of the system to record: the permission of the microphone, which macOS asks the user for.
type Permissions interface {
	// Microphone makes sure that the app may record: it asks the user when the answer is not given yet and waits for it until ctx
	// ends. It reports false when the user said no (or a policy says no).
	Microphone(ctx context.Context) (granted bool, err error)
}

// Options are what New is given.
type Options struct {
	// Permissions asks for the microphone; nil means that none is needed (tests).
	Permissions Permissions
	// Log gets what cannot be returned (nil discards it).
	Log *slog.Logger
	// Session makes the system ready for a stream (the AVAudioSession of iOS: its category and its activation); nil means that none is
	// needed (macOS, tests).
	Session Session
}

// Session is the audio session of a platform that has one: iOS makes an app say what its audio is for before it plays or records.
//
// Flutter: no counterpart
type Session interface {
	// Activate sets the session up for a stream of the kind and makes it active. It is called before a stream is opened, and again when an
	// interruption ended. The errors it gives are returned by Open as they are.
	Activate(kind platform.AudioKind) error
}

// Service is the Core Audio backend: it implements platform.Audio.
//
// Flutter: no counterpart
type Service struct {
	session Session
	perm    Permissions
	log     *slog.Logger
	load    func() (*library, error)

	mu      sync.Mutex
	lib     *library
	handle  uintptr
	devices map[*device]struct{}
	change  func()
	// listening says that the listeners of the system object are added.
	listening bool
}

var _ platform.Audio = (*Service)(nil)

// New makes a Service. Core Audio is not loaded yet.
//
// Flutter: no counterpart
func New(opts Options) *Service {
	log := opts.Log
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Service{perm: opts.Permissions, session: opts.Session, log: log, load: loadLibrary, devices: map[*device]struct{}{}}
}

// getLib loads Core Audio on first use and starts to listen to the system object. It fails with AudioUnavailable when Core Audio
// cannot be loaded, and with CoreAudioFailed when the listeners cannot be added.
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
	if !s.listening && s.lib.routeOnly {
		s.listening = true // the route has nothing to listen to: the devices are always the default ones
	}
	if !s.listening {
		s.handle = register(s)
		for _, sel := range []uint32{selDevices, selDefaultOutput, selDefaultInput} {
			addr := PropertyAddress{Selector: sel, Scope: scopeGlobal}
			if status := s.lib.addListener(systemObject, &addr, s.lib.listenerCB, s.handle); status != 0 {
				return nil, statusError("AudioObjectAddPropertyListener", status, false)
			}
		}
		s.listening = true
	}
	return s.lib, nil
}

// SetChangeHandler sets what is called when the list of devices or a default device changes. Core Audio calls it on a thread of its
// own.
//
// Flutter: no counterpart
func (s *Service) SetChangeHandler(fn func()) {
	s.mu.Lock()
	s.change = fn
	s.mu.Unlock()
}

// onSystemChange is the listener of the system object. The devices that were opened on the default device go to the new one when
// it is the default that changed, and the app is told.
func (s *Service) onSystemChange(selectors []uint32) {
	s.mu.Lock()
	fn := s.change
	var follow []*device
	for d := range s.devices {
		if d.opts.ID != "" {
			continue
		}
		for _, sel := range selectors {
			if (sel == selDefaultOutput && d.kind == platform.AudioPlayback) || (sel == selDefaultInput && d.kind == platform.AudioCapture) {
				follow = append(follow, d)
				break
			}
		}
	}
	s.mu.Unlock()
	for _, d := range follow {
		d.follow()
	}
	if fn != nil {
		fn()
	}
}

// forget drops a device that is closed.
func (s *Service) forget(d *device) {
	s.mu.Lock()
	delete(s.devices, d)
	s.mu.Unlock()
}

// records reads the devices of the system. A failure is CoreAudioFailed.
func (s *Service) records(lib *library) ([]DeviceRecord, error) {
	if lib.routeOnly {
		return routeRecords(), nil
	}
	ids, err := lib.objects(systemObject, PropertyAddress{Selector: selDevices, Scope: scopeGlobal})
	if err != nil {
		return nil, err
	}
	out := make([]DeviceRecord, 0, len(ids))
	for _, id := range ids {
		r := DeviceRecord{Object: id}
		r.UID, _ = lib.stringProp(id, PropertyAddress{Selector: selUID, Scope: scopeGlobal})
		r.Name, _ = lib.stringProp(id, PropertyAddress{Selector: selName, Scope: scopeGlobal})
		r.OutChannels = lib.channels(id, scopeOutput)
		r.InChannels = lib.channels(id, scopeInput)
		r.Rate, _ = lib.float64Prop(id, PropertyAddress{Selector: selNominalRate, Scope: scopeGlobal})
		if r.UID == "" {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// defaultObject is the AudioObjectID of the default device of a kind, or 0.
func (s *Service) defaultObject(lib *library, kind platform.AudioKind) uint32 {
	if lib.routeOnly {
		if kind == platform.AudioCapture {
			return routeInputObject
		}
		return routeOutputObject
	}
	sel := selDefaultOutput
	if kind == platform.AudioCapture {
		sel = selDefaultInput
	}
	v, _ := lib.uint32Prop(systemObject, PropertyAddress{Selector: sel, Scope: scopeGlobal})
	return v
}

// Devices lists the devices of a kind, with the default of the system marked. It fails with AudioUnavailable when Core Audio is not
// there.
//
// Flutter: no counterpart
func (s *Service) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	lib, err := s.getLib()
	if err != nil {
		return nil, err
	}
	recs, err := s.records(lib)
	if err != nil {
		return nil, err
	}
	return BuildDevices(kind, recs, s.defaultObject(lib, kind)), nil
}

// Open opens a device and starts its queue. It fails with AudioUnavailable when Core Audio is not there,
// AudioCapturePermissionDenied when the user refuses the microphone, AudioFormatUnsupported when the device takes no such format,
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
	if err := CheckFormat(opts.Format); err != nil {
		return nil, err
	}
	recs, err := s.records(lib)
	if err != nil {
		return nil, err
	}
	list := BuildDevices(kind, recs, s.defaultObject(lib, kind))
	var chosen *platform.AudioDeviceInfo
	for i := range list {
		if (opts.ID == "" && list[i].Default) || (opts.ID != "" && list[i].ID == opts.ID) {
			chosen = &list[i]
			break
		}
	}
	if chosen == nil && opts.ID != "" {
		return nil, errors.New(platform.TagAudioDeviceOpenFailed, "the system has no such audio device", "id", opts.ID)
	}
	if kind == platform.AudioCapture && s.perm != nil {
		granted, err := s.perm.Microphone(ctx)
		if err != nil {
			return nil, errors.NewErr(platform.TagAudioCapturePermissionDenied, err).SetMetadata("permission", "microphone")
		}
		if !granted {
			return nil, errors.New(platform.TagAudioCapturePermissionDenied, "the user did not allow the app to record", "permission", "microphone")
		}
	}
	if s.session != nil {
		if err := s.session.Activate(kind); err != nil {
			return nil, err
		}
	}
	format := opts.Format
	if format.Rate == 0 {
		format.Rate = 48000
		if chosen != nil && chosen.Format.Rate > 0 {
			format.Rate = chosen.Format.Rate
		}
	}
	if format.Channels == 0 {
		format.Channels = 2
		if chosen != nil && chosen.Format.Channels > 0 {
			format.Channels = min(chosen.Format.Channels, 2)
		}
	}
	d := &device{svc: s, lib: lib, kind: kind, h: h, opts: opts, format: format}
	if opts.ID != "" && !lib.routeOnly {
		for _, r := range recs {
			if r.UID == opts.ID {
				d.object = r.Object
			}
		}
	}
	d.handle = register(d)
	d.silent.Store(false)
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.build(); err != nil {
		unregister(d.handle)
		return nil, err
	}
	d.running = true
	if err := d.startLocked(true); err != nil {
		d.teardownLocked()
		unregister(d.handle)
		return nil, err
	}
	if d.object != 0 {
		addr := PropertyAddress{Selector: selIsAlive, Scope: scopeGlobal}
		if status := lib.addListener(d.object, &addr, lib.listenerCB, d.handle); status != 0 {
			d.teardownLocked()
			unregister(d.handle)
			return nil, statusError("AudioObjectAddPropertyListener", status, true)
		}
		d.listening = true
	}
	s.mu.Lock()
	s.devices[d] = struct{}{}
	s.mu.Unlock()
	return d, nil
}

// objects reads a property that is a list of AudioObjectIDs.
func (l *library) objects(obj uint32, addr PropertyAddress) ([]uint32, error) {
	var size uint32
	if status := l.getPropertyDataSize(obj, &addr, 0, nil, &size); status != 0 {
		return nil, statusError("AudioObjectGetPropertyDataSize", status, false)
	}
	if size == 0 {
		return nil, nil
	}
	out := make([]uint32, size/4)
	if status := l.getPropertyData(obj, &addr, 0, nil, &size, unsafe.Pointer(&out[0])); status != 0 {
		return nil, statusError("AudioObjectGetPropertyData", status, false)
	}
	return out[:size/4], nil
}

// uint32Prop reads a property that is one number.
func (l *library) uint32Prop(obj uint32, addr PropertyAddress) (uint32, bool) {
	var v uint32
	size := uint32(4)
	if l.getPropertyData(obj, &addr, 0, nil, &size, unsafe.Pointer(&v)) != 0 {
		return 0, false
	}
	return v, true
}

// float64Prop reads a property that is a double.
func (l *library) float64Prop(obj uint32, addr PropertyAddress) (float64, bool) {
	var v float64
	size := uint32(8)
	if l.getPropertyData(obj, &addr, 0, nil, &size, unsafe.Pointer(&v)) != 0 {
		return 0, false
	}
	return v, true
}

// stringProp reads a property that is a CFString, which the caller owns and which is given back here.
func (l *library) stringProp(obj uint32, addr PropertyAddress) (string, bool) {
	var ref cf.Ref
	size := uint32(8)
	if l.getPropertyData(obj, &addr, 0, nil, &size, unsafe.Pointer(&ref)) != 0 || ref == 0 {
		return "", false
	}
	s := cf.String(ref)
	cf.Release(ref)
	return s, true
}

// channels counts the channels of the streams of a device in a scope, from its stream configuration.
func (l *library) channels(obj, scope uint32) int {
	addr := PropertyAddress{Selector: selStreamConfig, Scope: scope}
	var size uint32
	if l.getPropertyDataSize(obj, &addr, 0, nil, &size) != 0 || size == 0 {
		return 0
	}
	buf := make([]byte, size)
	if l.getPropertyData(obj, &addr, 0, nil, &size, unsafe.Pointer(&buf[0])) != 0 {
		return 0
	}
	return ChannelCount(buf[:size])
}

// The made-up devices of a system that has only the route (iOS): the AudioObjectIDs and UIDs that stand for the output and the input of
// the current route. The system routes the audio, so no queue is bound to them.
const (
	routeOutputObject uint32 = 1
	routeInputObject  uint32 = 2
	// RouteOutputID and RouteInputID are the IDs of the output and the input of a system that has only the route.
	//
	// Flutter: no counterpart
	RouteOutputID = "route-output"
	// RouteInputID is the ID of the input of the route.
	//
	// Flutter: no counterpart
	RouteInputID = "route-input"
)

// routeRecords are the devices of a system that has only the route: one output and one input.
func routeRecords() []DeviceRecord {
	return []DeviceRecord{
		{Object: routeOutputObject, UID: RouteOutputID, Name: "Output", OutChannels: 2},
		{Object: routeInputObject, UID: RouteInputID, Name: "Input", InChannels: 1},
	}
}

// Interrupt tells the service that the system interrupted its audio (a call, an alarm, Siri) or ended the interruption, as
// AVAudioSession's interruption notification does. When it began, the devices that run are paused and the app is told (AudioPaused); when
// it ended and the system says that the audio should go on (resume), the session is made active again and the devices that the
// interruption paused run again (AudioRunning); the devices that the app paused itself stay paused, and so do all of them when the system
// does not say to resume. It may be called from any thread.
//
// Flutter: no counterpart
func (s *Service) Interrupt(began, resume bool) {
	s.mu.Lock()
	list := make([]*device, 0, len(s.devices))
	for d := range s.devices {
		list = append(list, d)
	}
	session := s.session
	s.mu.Unlock()
	reactivated := map[platform.AudioKind]bool{}
	for _, d := range list {
		if !began && resume && session != nil && d.wasInterrupted() && !reactivated[d.kind] {
			reactivated[d.kind] = true
			if err := session.Activate(d.kind); err != nil {
				s.log.Error("the audio session could not be made active again", "err", err)
			}
		}
		d.interrupt(began, resume)
	}
}

// RouteLost tells the service that the device that the audio went to is gone (the headphones were unplugged: AVAudioSession's
// route change with the reason "old device unavailable"): the playback devices that run are paused, as Apple's guidelines say, and
// the app resumes them when it wants.
//
// Flutter: no counterpart
func (s *Service) RouteLost() {
	s.mu.Lock()
	list := make([]*device, 0, len(s.devices))
	for d := range s.devices {
		if d.kind == platform.AudioPlayback {
			list = append(list, d)
		}
	}
	s.mu.Unlock()
	for _, d := range list {
		d.mu.Lock()
		running := d.running
		d.mu.Unlock()
		if running {
			if err := d.Pause(); err != nil {
				s.log.Error("the audio device could not be paused when its route was lost", "err", err)
			}
		}
	}
}
