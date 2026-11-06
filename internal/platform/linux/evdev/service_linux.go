//go:build linux && !android

package evdev

import (
	"encoding/binary"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Opts are the options of a Service.
//
// Flutter: no counterpart
type Opts struct {
	// Post runs a function on the UI thread; it is safe to call from any goroutine.
	Post func(func())
	// Logger receives the notes of the backend; nil logs nothing.
	Logger *slog.Logger
	// DevDir is the directory of the event devices ("/dev/input" when empty) and SysDir that of their description in sysfs
	// ("/sys/class/input" when empty); tests point them at other places.
	DevDir, SysDir string
	// PollEvery makes the service look at the directory of the devices that often, instead of asking inotify to tell it. The
	// service does so by itself (every 2 seconds) when inotify is not there for it, which happens when the user has used up the
	// instances that the system allows; tests use it to try that way.
	PollEvery time.Duration
}

// Service is the gamepad backend of Linux (platform.Gamepads). It finds the pads among the event devices, reads them on a
// goroutine of its own and hands the reports over to the UI thread with Post.
//
// Flutter: no counterpart
type Service struct {
	post           func(func())
	log            *slog.Logger
	devDir, sysDir string
	pollEvery      time.Duration
	k              kernel
	nextID         atomic.Int64

	mu  sync.Mutex
	cur *run
}

// New makes the service. It does nothing until Start.
//
// Flutter: no counterpart
func New(opts Opts) *Service {
	s := &Service{post: opts.Post, log: opts.Logger, devDir: opts.DevDir, sysDir: opts.SysDir, pollEvery: opts.PollEvery, k: realKernel()}
	if s.devDir == "" {
		s.devDir = "/dev/input"
	}
	if s.sysDir == "" {
		s.sysDir = "/sys/class/input"
	}
	if s.log == nil {
		s.log = slog.New(slog.DiscardHandler)
	}
	return s
}

var _ platform.Gamepads = (*Service)(nil)

// MappingPlatform implements platform.Gamepads.
func (s *Service) MappingPlatform() string { return "Linux" }

// Start implements platform.Gamepads: it looks at the event devices that are there, watches /dev/input for the ones that come
// and go, and reads the pads on a goroutine of its own.
func (s *Service) Start(h platform.PadHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		return nil
	}
	r, err := s.newRun(h)
	if err != nil {
		return err
	}
	s.cur = r
	go r.loop()
	return nil
}

// Stop implements platform.Gamepads.
func (s *Service) Stop() {
	s.mu.Lock()
	r := s.cur
	s.cur = nil
	s.mu.Unlock()
	if r != nil {
		r.stop()
	}
}

// Rumble implements platform.Gamepads: force feedback is added in a later step (SPEC §28.2).
func (s *Service) Rumble(id platform.PadID, low, high uint16, d time.Duration) error {
	return errors.New(platform.TagGamepadRumbleUnavailable, "rumble is not available on Linux yet", "pad", int64(id))
}

// RumbleTriggers implements platform.Gamepads.
func (s *Service) RumbleTriggers(id platform.PadID, left, right uint16, d time.Duration) error {
	return errors.New(platform.TagGamepadRumbleUnavailable, "the triggers of a pad cannot rumble on Linux", "pad", int64(id))
}

// SetLED implements platform.Gamepads.
func (s *Service) SetLED(id platform.PadID, r, g, b uint8) error {
	return errors.New(platform.TagGamepadLEDUnavailable, "the light of a pad cannot be set on Linux yet", "pad", int64(id))
}

// run is one Start of the service.
type run struct {
	s       *Service
	h       platform.PadHandler
	epfd    int
	inofd   int
	stopR   int
	stopW   int
	done    chan struct{}
	stopped atomic.Bool

	// The rest belongs to the goroutine of loop.
	devices map[int]*padDevice // by file descriptor
	byPath  map[string]*padDevice
	// ignored are the event devices that are not pads, so that they are not looked at again.
	ignored map[string]bool
	// retries are the devices that could not be looked at yet (sysfs or the permissions were not ready), with when to try again.
	retries   map[string]*retry
	warnedAcl bool
	// every is how often the directory is looked at on a timer, 0 when inotify tells; nextPoll is when it is next, and seen the
	// nodes it had then.
	every    time.Duration
	nextPoll time.Time
	seen     map[string]bool
	// denied are the nodes that could not be opened for want of permission: with inotify the change of their attributes
	// says when to try again, and while polling every look does.
	denied map[string]bool
}

type retry struct {
	at    time.Time
	tries int
}

// padDevice is an open event device that is a pad.
type padDevice struct {
	id     platform.PadID
	path   string
	fd     int
	dec    *decoder
	events []inputEvent
}

// pollFallback is how often the directory is looked at when inotify is not there.
const pollFallback = 2 * time.Second

var eventNode = regexp.MustCompile(`^event[0-9]+$`)

func (s *Service) newRun(h platform.PadHandler) (*run, error) {
	r := &run{
		s: s, h: h, done: make(chan struct{}),
		devices: map[int]*padDevice{}, byPath: map[string]*padDevice{}, ignored: map[string]bool{}, retries: map[string]*retry{},
		seen: map[string]bool{}, denied: map[string]bool{},
		epfd: -1, inofd: -1, stopR: -1, stopW: -1,
	}
	var err error
	if r.epfd, err = syscall.EpollCreate1(syscall.EPOLL_CLOEXEC); err != nil {
		r.closeAll()
		return nil, errors.NewErr(tagEvdevStartFailed, err).SetMetadata("step", "epoll_create1")
	}
	r.every = s.pollEvery
	if r.every > 0 {
		// Asked for.
	} else if r.inofd, err = syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK); err != nil {
		// The user may have used up the inotify instances the system allows: the directory is looked at on a timer instead.
		s.log.Info("inotify is not available, so the gamepads are looked for every two seconds", "err", errors.NewErr(tagEvdevStartFailed, err).SetMetadata("step", "inotify_init1"))
		r.inofd, r.every = -1, pollFallback
	}
	var p [2]int
	if err = syscall.Pipe2(p[:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
		r.closeAll()
		return nil, errors.NewErr(tagEvdevStartFailed, err).SetMetadata("step", "pipe2")
	}
	r.stopR, r.stopW = p[0], p[1]
	if r.every > 0 {
		if _, err := os.Stat(s.devDir); err != nil {
			r.closeAll()
			return nil, errors.NewErr(tagEvdevStartFailed, err).SetMetadata("step", "stat", "dir", s.devDir)
		}
	} else {
		// Watch for the devices that come and go. The permissions of a new node are set a moment after it appears, so ATTRIB
		// matters too. A directory that cannot be watched (a container with no /dev/input) ends the start: there are no pads.
		const mask = syscall.IN_CREATE | syscall.IN_DELETE | syscall.IN_MOVED_TO | syscall.IN_MOVED_FROM | syscall.IN_ATTRIB
		if _, err = syscall.InotifyAddWatch(r.inofd, s.devDir, mask); err != nil {
			if err == syscall.ENOSPC {
				s.log.Info("the system's inotify watches are used up, so the gamepads are looked for every two seconds")
				_ = syscall.Close(r.inofd) // the watch was refused: there is nothing to keep
				r.inofd, r.every = -1, pollFallback
			} else {
				r.closeAll()
				return nil, errors.NewErr(tagEvdevStartFailed, err).SetMetadata("step", "inotify_add_watch", "dir", s.devDir)
			}
		}
	}
	fds := []int{r.stopR}
	if r.inofd >= 0 {
		fds = append(fds, r.inofd)
	}
	for _, fd := range fds {
		ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fd)}
		if err = syscall.EpollCtl(r.epfd, syscall.EPOLL_CTL_ADD, fd, &ev); err != nil {
			r.closeAll()
			return nil, errors.NewErr(tagEvdevStartFailed, err).SetMetadata("step", "epoll_ctl")
		}
	}
	return r, nil
}

// close closes the descriptors that the goroutine of the loop owns: the write end of the pipe that stops it is Stop's.
func (r *run) close() {
	for _, fd := range []*int{&r.epfd, &r.inofd, &r.stopR} {
		if *fd >= 0 {
			_ = syscall.Close(*fd) // the end of the run: a descriptor that fails to close is not recoverable
			*fd = -1
		}
	}
}

// closeAll closes every descriptor, for a run that never started its goroutine.
func (r *run) closeAll() {
	r.close()
	if r.stopW >= 0 {
		_ = syscall.Close(r.stopW) // the same
		r.stopW = -1
	}
}

// stop ends the goroutine and waits for it. The reports that are posted already are dropped when they run.
func (r *run) stop() {
	r.stopped.Store(true)
	_, _ = syscall.Write(r.stopW, []byte{1}) // if the pipe is gone the loop is gone, and done is closed
	<-r.done
	_ = syscall.Close(r.stopW) // the end of the run
}

// deliver runs fn on the UI thread unless the service has been stopped by then.
func (r *run) deliver(fn func(platform.PadHandler)) {
	r.s.post(func() {
		if !r.stopped.Load() {
			fn(r.h)
		}
	})
}

func (r *run) loop() {
	defer close(r.done)
	defer func() {
		for _, d := range r.devices {
			_ = syscall.Close(d.fd) // the end of the run
		}
		r.close()
	}()
	r.scan()
	if r.every > 0 {
		r.seen = r.listing()
	}
	r.nextPoll = time.Now().Add(r.every)
	events := make([]syscall.EpollEvent, 16)
	buf := make([]byte, 64*eventSize)
	var batch []platform.PadEvent
	for {
		n, err := syscall.EpollWait(r.epfd, events, r.timeout())
		if err != nil && err != syscall.EINTR {
			r.s.log.Error("the gamepad loop ends", "err", errors.NewErr(tagEvdevWaitFailed, err))
			return
		}
		for i := 0; i < n; i++ {
			fd := int(events[i].Fd)
			switch fd {
			case r.stopR:
				return
			case r.inofd:
				r.watchEvents()
			default:
				if d := r.devices[fd]; d != nil {
					batch = r.readDevice(d, buf, batch)
				}
			}
		}
		r.retryDue()
		if r.every > 0 && !time.Now().Before(r.nextPoll) {
			r.nextPoll = time.Now().Add(r.every)
			r.rescan()
		}
		if len(batch) > 0 {
			out := batch
			batch = nil
			r.deliver(func(h platform.PadHandler) { h.PadEvents(out) })
		}
	}
}

// timeout is how long the loop may wait for something to happen: until the next retry or the next look at the directory, or for
// ever.
func (r *run) timeout() int {
	wait := -1
	now := time.Now()
	soonest := func(at time.Time) {
		d := int(at.Sub(now) / time.Millisecond)
		if d < 1 {
			d = 1
		}
		if wait < 0 || d < wait {
			wait = d
		}
	}
	for _, t := range r.retries {
		soonest(t.at)
	}
	if r.every > 0 {
		soonest(r.nextPoll)
	}
	return wait
}

// scan looks at the event devices that are there.
func (r *run) scan() {
	entries, err := os.ReadDir(r.s.devDir)
	if err != nil {
		r.s.log.Info("the input devices cannot be listed, so there are no gamepads", "err", errors.NewErr(tagEvdevScanFailed, err).SetMetadata("dir", r.s.devDir))
		return
	}
	for _, e := range entries {
		if eventNode.MatchString(e.Name()) {
			r.consider(e.Name())
		}
	}
}

// listing returns the event nodes that are in the directory now.
func (r *run) listing() map[string]bool {
	out := map[string]bool{}
	entries, _ := os.ReadDir(r.s.devDir)
	for _, e := range entries {
		if eventNode.MatchString(e.Name()) {
			out[e.Name()] = true
		}
	}
	return out
}

// rescan is the look at the directory of a service that is not told by inotify: the nodes that are new are considered, the
// ones that are gone are let go, and the ones that could not be opened for want of permission are tried again.
func (r *run) rescan() {
	entries, err := os.ReadDir(r.s.devDir)
	if err != nil {
		return
	}
	now := map[string]bool{}
	for _, e := range entries {
		if eventNode.MatchString(e.Name()) {
			now[e.Name()] = true
		}
	}
	for node := range r.seen {
		if !now[node] {
			delete(r.ignored, node)
			delete(r.retries, node)
			delete(r.denied, node)
			r.remove(node)
		}
	}
	for node := range now {
		if !r.seen[node] || r.denied[node] {
			r.consider(node)
		}
	}
	r.seen = now
}

// watchEvents reads the events of inotify: a node that appears is looked at, one that goes is let go.
func (r *run) watchEvents() {
	buf := make([]byte, 4096)
	for {
		n, err := syscall.Read(r.inofd, buf)
		if err != nil || n <= 0 {
			return
		}
		for off := 0; off+syscall.SizeofInotifyEvent <= n; {
			mask := binary.NativeEndian.Uint32(buf[off+4:])
			nameLen := int(binary.NativeEndian.Uint32(buf[off+12:]))
			name := buf[off+syscall.SizeofInotifyEvent : off+syscall.SizeofInotifyEvent+nameLen]
			off += syscall.SizeofInotifyEvent + nameLen
			for i, c := range name {
				if c == 0 {
					name = name[:i]
					break
				}
			}
			node := string(name)
			if !eventNode.MatchString(node) {
				continue
			}
			switch {
			case mask&(syscall.IN_DELETE|syscall.IN_MOVED_FROM) != 0:
				delete(r.ignored, node)
				delete(r.retries, node)
				r.remove(node)
			case mask&(syscall.IN_CREATE|syscall.IN_MOVED_TO|syscall.IN_ATTRIB) != 0:
				if mask&syscall.IN_ATTRIB != 0 {
					// The permissions changed: a device that could not be opened may be opened now.
					delete(r.retries, node)
				}
				r.consider(node)
			}
		}
	}
}

// retryDue looks again at the devices whose time has come.
func (r *run) retryDue() {
	now := time.Now()
	for node, t := range r.retries {
		if !t.at.After(now) {
			r.consider(node)
		}
	}
}

// consider looks at an event device and, when it is a pad, opens it and tells the handler.
func (r *run) consider(node string) {
	if r.ignored[node] || r.byPath[node] != nil {
		return
	}
	dir := filepath.Join(r.s.sysDir, node)
	c, ok := probe(dirFiles(dir))
	if !ok {
		// A device that has just appeared may not have its attributes yet.
		r.later(node)
		return
	}
	if !c.isJoystick() {
		r.ignored[node] = true
		delete(r.retries, node)
		return
	}
	c.Driver = driverName(dir)
	d, info, err := r.open(node, &c)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == tagEvdevPermission {
			if !r.warnedAcl {
				r.warnedAcl = true
				r.s.log.Info("a gamepad is there but this user may not read it; on a desktop the login session gives access, otherwise add the user to the input group or install a udev rule (docs/GUIDE.md)", "device", filepath.Join(r.s.devDir, node), "err", err)
			}
			// It is retried when the permissions change (IN_ATTRIB), not by the clock.
			delete(r.retries, node)
			r.denied[node] = true
			return
		}
		r.s.log.Warn("a gamepad cannot be opened", "err", err)
		r.later(node)
		return
	}
	if d == nil {
		r.ignored[node] = true
		delete(r.retries, node)
		return
	}
	delete(r.retries, node)
	delete(r.denied, node)
	r.devices[d.fd] = d
	r.byPath[node] = d
	ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(d.fd)}
	if err := syscall.EpollCtl(r.epfd, syscall.EPOLL_CTL_ADD, d.fd, &ev); err != nil {
		r.s.log.Warn("a gamepad cannot be watched", "err", errors.NewErr(tagEvdevWatchFailed, err).SetMetadata("device", d.path))
		r.drop(d)
		return
	}
	r.deliver(func(h platform.PadHandler) { h.PadAdded(info) })
	// Report the state the pad is in now: the layer above finds the rest values of the axes from it.
	d.dec.pollAll(monotonicNow(), deviceSource{r.s.k, d.fd})
	if out := d.dec.take(); len(out) > 0 {
		r.deliver(func(h platform.PadHandler) { h.PadEvents(out) })
	}
}

// later schedules another look at a device that is not ready, a few times.
func (r *run) later(node string) {
	t := r.retries[node]
	if t == nil {
		t = &retry{}
		r.retries[node] = t
	}
	t.tries++
	if t.tries > 20 {
		delete(r.retries, node)
		r.s.log.Debug("giving up on an input device", "device", node)
		return
	}
	t.at = time.Now().Add(100 * time.Millisecond)
}

// open opens the event device of a pad, reads its axes and makes its description. It returns a nil device for a pad that is
// to be ignored.
func (r *run) open(node string, c *caps) (*padDevice, platform.PadInfo, error) {
	path := filepath.Join(r.s.devDir, node)
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		tag := tagEvdevOpenFailed
		if err == syscall.EACCES || err == syscall.EPERM {
			tag = tagEvdevPermission
		}
		return nil, platform.PadInfo{}, errors.NewErr(tag, err).SetMetadata("device", path)
	}
	for code := 0; code < absCnt; code++ {
		if !c.Abs.has(code) {
			continue
		}
		info, err := r.s.k.readAbs(fd, code)
		if err != nil {
			_ = syscall.Close(fd) // the device is given up
			return nil, platform.PadInfo{}, errors.NewErr(tagEvdevIoctlFailed, err).SetMetadata("device", path, "request", "EVIOCGABS", "axis", code)
		}
		c.Info[code] = info
	}
	if err := r.s.k.setClock(fd); err != nil {
		// Without it the events are stamped with the wall clock: the pad still works.
		r.s.log.Debug("a gamepad keeps the wall clock", "device", path, "err", errors.NewErr(tagEvdevIoctlFailed, err).SetMetadata("device", path, "request", "EVIOCSCLOCKID"))
	}
	fixupForMapping(c)
	if ignoreDevice(c) {
		_ = syscall.Close(fd) // the device is ignored
		return nil, platform.PadInfo{}, nil
	}
	l := newLayout(c)
	name := gamepaddb.DeviceName(c.ID.Vendor, c.ID.Product, "", c.Name)
	id := platform.PadID(r.s.nextID.Add(1))
	info := platform.PadInfo{
		ID:      id,
		Name:    name,
		GUID:    gamepaddb.NewGUID(c.ID.Bus, c.ID.Vendor, c.ID.Product, c.ID.Version, "", c.Name, 0, 0),
		Path:    path,
		Axes:    l.axes,
		Buttons: l.buttons,
		Hats:    l.hats,
		Mapping: mappingOf(c, l),
		Battery: -1,
	}
	return &padDevice{id: id, path: path, fd: fd, dec: newDecoder(id, l)}, info, nil
}

// readDevice reads the events that are waiting on a device and decodes them into batch.
func (r *run) readDevice(d *padDevice, buf []byte, batch []platform.PadEvent) []platform.PadEvent {
	for {
		n, err := syscall.Read(d.fd, buf)
		if err == syscall.EAGAIN || err == syscall.EINTR {
			break
		}
		if err != nil || n <= 0 {
			// ENODEV: the device is gone; the removal from inotify follows, and this does it first.
			r.drop(d)
			return batch
		}
		d.events = parseEvents(buf[:n], d.events[:0])
		now := monotonicNow()
		for _, ev := range d.events {
			d.dec.handle(ev, now, deviceSource{r.s.k, d.fd})
		}
		batch = append(batch, d.dec.take()...)
	}
	return batch
}

// remove lets go of the device of a node.
func (r *run) remove(node string) {
	if d := r.byPath[node]; d != nil {
		r.drop(d)
	}
}

// drop closes a device and tells the handler it is gone.
func (r *run) drop(d *padDevice) {
	if r.devices[d.fd] != d {
		return
	}
	delete(r.devices, d.fd)
	delete(r.byPath, filepath.Base(d.path))
	// A node that is looked at on a timer is new again if it is there at the next look: a pad that was unplugged and plugged in
	// again between two looks has the number it had.
	delete(r.seen, filepath.Base(d.path))
	_ = syscall.EpollCtl(r.epfd, syscall.EPOLL_CTL_DEL, d.fd, nil) // closing the descriptor removes it anyway
	_ = syscall.Close(d.fd)                                        // the device is gone or given up
	id := d.id
	r.deliver(func(h platform.PadHandler) { h.PadRemoved(id) })
}

// The tags of the errors that this package makes (rm4n0s/errors, R5).
const (
	tagEvdevStartFailed = "EvdevStartFailed"
	tagEvdevWaitFailed  = "EvdevWaitFailed"
	tagEvdevScanFailed  = "EvdevScanFailed"
	tagEvdevOpenFailed  = "EvdevDeviceOpenFailed"
	tagEvdevPermission  = "EvdevDevicePermission"
	tagEvdevIoctlFailed = "EvdevIoctlFailed"
	tagEvdevWatchFailed = "EvdevWatchFailed"
)
