//go:build windows

package wasapi

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// mtaThread is an OS thread in the multithreaded COM apartment that runs the functions it is given, one at a time. Every call into
// the device enumerator is made on it, so that a COM object is only ever used by the thread that made it.
type mtaThread struct {
	calls chan func()
	quit  chan struct{}
	done  chan struct{}
}

// newMTAThread starts the thread and waits until its apartment is made. It fails with the error of CoInitializeEx.
func newMTAThread() (*mtaThread, error) {
	t := &mtaThread{calls: make(chan func()), quit: make(chan struct{}), done: make(chan struct{})}
	ready := make(chan error, 1)
	go t.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return t, nil
}

// run is the goroutine of the thread: it is locked to the thread for good (a thread that ends with its apartment needs no unlock).
func (t *mtaThread) run(ready chan<- error) {
	runtime.LockOSThread()
	defer close(t.done)
	if err := win32.CoInitializeEx(win32.COInitMultithreaded); err != nil {
		ready <- err
		return
	}
	defer win32.CoUninitialize()
	ready <- nil
	for {
		select {
		case fn := <-t.calls:
			fn()
		case <-t.quit:
			return
		}
	}
}

// do runs fn on the thread and returns when it is done; false when the thread has ended.
func (t *mtaThread) do(fn func()) bool {
	finished := make(chan struct{})
	select {
	case t.calls <- func() { fn(); close(finished) }:
		<-finished
		return true
	case <-t.done:
		return false
	}
}

// stop ends the thread and waits for it.
func (t *mtaThread) stop() {
	select {
	case <-t.quit:
	default:
		close(t.quit)
	}
	<-t.done
}

// Service is the WASAPI backend: it implements platform.Audio.
//
// Flutter: no counterpart
type Service struct {
	log *slog.Logger

	mu      sync.Mutex
	mgmt    *mtaThread
	closed  bool
	handler func()
	devices map[*device]struct{}

	// enum and notif belong to the thread mgmt.
	enum  win32.Object
	notif win32.Object
}

var _ platform.Audio = (*Service)(nil)

// New makes a Service. Nothing of Windows is touched until the app first lists or opens a device; log gets what cannot be
// returned (nil discards it).
//
// Flutter: no counterpart
func New(log *slog.Logger) *Service {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Service{log: log, devices: map[*device]struct{}{}}
}

// thread returns the thread of the enumerator, making it on first use. It fails with AudioUnavailable when Windows has no device
// enumerator, and with the error of COM when the apartment cannot be made.
func (s *Service) thread() (*mtaThread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errors.New(platform.TagAudioUnavailable, "the audio service was closed")
	}
	if s.mgmt != nil {
		return s.mgmt, nil
	}
	t, err := newMTAThread()
	if err != nil {
		return nil, err
	}
	var initErr error
	t.do(func() { initErr = s.start() })
	if initErr != nil {
		t.stop()
		return nil, initErr
	}
	s.mgmt = t
	return t, nil
}

// start runs on the new thread: it makes the enumerator and registers the notification client. A client that cannot be registered
// is logged: the devices are then listed and opened as always, and a change of the default device is not followed.
func (s *Service) start() error {
	enum, hr := win32.CoCreateInstanceHR(&win32.CLSIDMMDeviceEnumerator, &win32.IIDMMDeviceEnumerator, win32.ClsctxAll)
	if hr.Failed() {
		return errors.New(platform.TagAudioUnavailable, "Windows has no audio device enumerator", "hresult", hr.String())
	}
	s.enum = enum
	env, err := sharedNotifyEnv()
	if err != nil {
		s.log.Warn("audio: the notifications of the devices are not available", "err", err)
		return nil
	}
	obj, err := env.host.NewObject(&notifier{svc: s}, env.typ)
	if err != nil {
		s.log.Warn("audio: the notification client could not be made", "err", err)
		return nil
	}
	if hr := call(s.enum, win32.IMMDeviceEnumeratorRegisterEndpointNotificationCallback, obj.Ptr()); hr.Failed() {
		s.log.Warn("audio: the notification client was refused", "hresult", hr.String())
		obj.Release()
		return nil
	}
	s.notif = obj
	return nil
}

// stopEnumerator runs on the thread of the enumerator before it ends.
func (s *Service) stopEnumerator() {
	if !s.notif.IsNil() {
		if hr := call(s.enum, win32.IMMDeviceEnumeratorUnregisterEndpointNotificationCallback, s.notif.Ptr()); hr.Failed() {
			s.log.Debug("audio: the notification client could not be unregistered", "hresult", hr.String())
		}
		s.notif.Release()
		s.notif = win32.Object{}
	}
	release(s.enum)
	s.enum = win32.Object{}
}

// Close closes the devices that are still open and ends the thread of the enumerator. The service cannot be used afterwards.
//
// Flutter: no counterpart
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	t := s.mgmt
	s.mgmt = nil
	open := make([]*device, 0, len(s.devices))
	for d := range s.devices {
		open = append(open, d)
	}
	s.mu.Unlock()
	for _, d := range open {
		if err := d.Close(); err != nil {
			s.log.Debug("audio: a device could not be closed", "err", err)
		}
	}
	if t != nil {
		t.do(s.stopEnumerator)
		t.stop()
	}
}

// SetChangeHandler sets what is called when the list of devices or a default device changes (from a thread of Windows).
//
// Flutter: no counterpart
func (s *Service) SetChangeHandler(fn func()) {
	s.mu.Lock()
	s.handler = fn
	s.mu.Unlock()
}

// changed calls the change handler.
func (s *Service) changed() {
	s.mu.Lock()
	fn := s.handler
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// openDevices are the devices that are open, for the notifications.
func (s *Service) openDevices() []*device {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := make([]*device, 0, len(s.devices))
	for d := range s.devices {
		list = append(list, d)
	}
	return list
}

// defaultChanged tells the devices that follow the default device of a flow that it changed, and the app that the list changed.
func (s *Service) defaultChanged(flow uint32, id string) {
	for _, d := range s.openDevices() {
		if d.pinned == "" && flowOf(d.kind) == flow {
			d.post(func() { d.followDefault(id) })
		}
	}
	s.changed()
}

// endpointChanged tells the devices that are open on an endpoint that its state changed (it may be gone), and the app that the list
// changed.
func (s *Service) endpointChanged(id string) {
	for _, d := range s.openDevices() {
		if d.pinned == id {
			d.post(d.probe)
		}
	}
	s.changed()
}

// forget drops a device that is closed or that failed to open.
func (s *Service) forget(d *device) {
	s.mu.Lock()
	delete(s.devices, d)
	s.mu.Unlock()
}

// Devices lists the active endpoints of a kind, with the default device marked. It fails with AudioUnavailable when Windows has no
// audio service.
//
// Flutter: no counterpart
func (s *Service) Devices(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	t, err := s.thread()
	if err != nil {
		return nil, err
	}
	var list []platform.AudioDeviceInfo
	var listErr error
	if !t.do(func() { list, listErr = s.enumerate(kind) }) {
		return nil, errors.New(platform.TagAudioUnavailable, "the audio service was closed")
	}
	return list, listErr
}

// enumerate runs on the thread of the enumerator.
func (s *Service) enumerate(kind platform.AudioKind) ([]platform.AudioDeviceInfo, error) {
	flow := flowOf(kind)
	defaultID := s.defaultEndpointID(flow)
	var coll uintptr
	if hr := call(s.enum, win32.IMMDeviceEnumeratorEnumAudioEndpoints, uintptr(flow), win32.DeviceStateActive, uintptr(unsafe.Pointer(&coll))); hr.Failed() {
		return nil, errors.New(platform.TagAudioUnavailable, "Windows could not list the audio devices",
			"method", "IMMDeviceEnumerator::EnumAudioEndpoints", "hresult", hr.String())
	}
	collection := win32.ObjectFromPtr(coll)
	defer collection.Release()
	var count uint32
	if hr := call(collection, win32.IMMDeviceCollectionGetCount, uintptr(unsafe.Pointer(&count))); hr.Failed() {
		return nil, errors.New(platform.TagAudioUnavailable, "Windows could not count the audio devices",
			"method", "IMMDeviceCollection::GetCount", "hresult", hr.String())
	}
	list := make([]platform.AudioDeviceInfo, 0, count)
	for i := range count {
		var dev uintptr
		if hr := call(collection, win32.IMMDeviceCollectionItem, uintptr(i), uintptr(unsafe.Pointer(&dev))); hr.Failed() {
			s.log.Debug("audio: a device of the list could not be read", "index", i, "hresult", hr.String())
			continue
		}
		info := s.describe(win32.ObjectFromPtr(dev), kind)
		info.Default = info.ID != "" && info.ID == defaultID
		if info.ID != "" {
			list = append(list, info)
		}
	}
	return list, nil
}

// describe reads the ID, the name and the mix format of an endpoint, and releases it.
func (s *Service) describe(dev win32.Object, kind platform.AudioKind) platform.AudioDeviceInfo {
	defer dev.Release()
	info := platform.AudioDeviceInfo{Kind: kind}
	info.ID = endpointIDOf(dev)
	info.Name = friendlyName(dev)
	if info.Name == "" {
		info.Name = info.ID
	}
	var ac uintptr
	if hr := call(dev, win32.IMMDeviceActivate, uintptr(unsafe.Pointer(&win32.IIDAudioClient)), win32.ClsctxAll, 0, uintptr(unsafe.Pointer(&ac))); hr.Succeeded() {
		client := win32.ObjectFromPtr(ac)
		defer client.Release()
		info.Format, _ = mixFormat(client)
	}
	return info
}

// defaultEndpointID is the ID of the default endpoint of a flow (the console role), "" when there is none. It runs on the thread of
// the enumerator.
func (s *Service) defaultEndpointID(flow uint32) string {
	var dev uintptr
	if hr := call(s.enum, win32.IMMDeviceEnumeratorGetDefaultAudioEndpoint, uintptr(flow), win32.EConsole, uintptr(unsafe.Pointer(&dev))); hr.Failed() {
		return ""
	}
	d := win32.ObjectFromPtr(dev)
	defer d.Release()
	return endpointIDOf(d)
}

// endpointIDOf is the ID of an endpoint, "" when it cannot be read.
func endpointIDOf(dev win32.Object) string {
	var id uintptr
	if hr := call(dev, win32.IMMDeviceGetID, uintptr(unsafe.Pointer(&id))); hr.Failed() {
		return ""
	}
	defer win32.CoTaskMemFree(id)
	return win32.WideString(id)
}

// friendlyName is the name of an endpoint that a user sees, "" when the property is not there.
func friendlyName(dev win32.Object) string {
	var store uintptr
	if hr := call(dev, win32.IMMDeviceOpenPropertyStore, win32.STGMRead, uintptr(unsafe.Pointer(&store))); hr.Failed() {
		return ""
	}
	ps := win32.ObjectFromPtr(store)
	defer ps.Release()
	var pv win32.PropVariant
	if hr := call(ps, win32.IPropertyStoreGetValue, uintptr(unsafe.Pointer(&win32.PKeyDeviceFriendlyName)), uintptr(unsafe.Pointer(&pv))); hr.Failed() {
		return ""
	}
	defer win32.PropVariantClear(&pv)
	if pv.VT != win32.VTLPWStr {
		return ""
	}
	return win32.WideString(pv.Val)
}

// waveHead is the first words of a WAVEFORMATEX, which is all the audio code reads of the engine's mix format (it may be a plain
// WAVEFORMATEX of 18 bytes, so the extensible structure is not read whole).
type waveHead struct {
	Tag, Channels uint16
	Rate          uint32
}

// mixFormat is the engine's mix format of an audio client (GetMixFormat), as a rate and a number of channels.
func mixFormat(client win32.Object) (platform.AudioFormat, win32.HRESULT) {
	var wf uintptr
	if hr := call(client, win32.IAudioClientGetMixFormat, uintptr(unsafe.Pointer(&wf))); hr.Failed() {
		return platform.AudioFormat{}, hr
	}
	defer win32.CoTaskMemFree(wf)
	head := win32.View[waveHead](wf)
	return platform.AudioFormat{Rate: int(head.Rate), Channels: int(head.Channels)}, win32.SOK
}

// Open opens a device and starts the thread that serves it; it returns when the device is ready. It fails with AudioUnavailable
// when Windows has no audio service, AudioFormatUnsupported when the device takes no such format, AudioCapturePermissionDenied when
// the user's settings do not allow the microphone, and AudioDeviceOpenFailed for the rest.
//
// Flutter: no counterpart
func (s *Service) Open(ctx context.Context, kind platform.AudioKind, opts platform.AudioOpenOptions, h platform.AudioHandler) (platform.AudioDevice, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagAudioDeviceOpenFailed, err).SetMetadata("id", opts.ID)
	}
	if r, c := opts.Format.Rate, opts.Format.Channels; r < 0 || c < 0 || c > maxChannels || (r != 0 && (r < minRate || r > maxRate)) {
		return nil, errors.New(platform.TagAudioFormatUnsupported, "WASAPI devices take 1 to 8 channels and a rate of 1000 to 768000",
			"rate", r, "channels", c)
	}
	if _, err := s.thread(); err != nil {
		return nil, err
	}
	d := newDevice(s, kind, opts, h)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errors.New(platform.TagAudioUnavailable, "the audio service was closed")
	}
	s.devices[d] = struct{}{}
	s.mu.Unlock()
	go d.run()
	select {
	case err := <-d.ready:
		if err != nil {
			s.forget(d)
			return nil, err
		}
		return d, nil
	case <-ctx.Done():
		go func() {
			if err := <-d.ready; err != nil {
				s.log.Debug("audio: a device that nobody waits for did not open", "err", err)
			}
			if err := d.Close(); err != nil {
				s.log.Debug("audio: a device that nobody waits for could not be closed", "err", err)
			}
		}()
		return nil, errors.NewErr(platform.TagAudioDeviceOpenFailed, ctx.Err()).SetMetadata("id", opts.ID)
	}
}
