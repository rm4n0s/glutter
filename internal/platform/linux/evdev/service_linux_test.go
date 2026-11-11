//go:build linux && !android

package evdev

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/platform"
)

// formatCapabilities writes bits as a capabilities file of sysfs does.
func formatCapabilities(b bits) string {
	wordBits := strconv.IntSize
	words := (8*len(b) + wordBits - 1) / wordBits
	if words == 0 {
		return "0\n"
	}
	var parts []string
	for w := words - 1; w >= 0; w-- {
		var v uint64
		for bit := 0; bit < wordBits; bit++ {
			if b.has(w*wordBits + bit) {
				v |= 1 << bit
			}
		}
		parts = append(parts, strconv.FormatUint(v, 16))
	}
	return strings.Join(parts, " ") + "\n"
}

// recorder is the handler of the platform interface of a test, and the UI thread: functions are posted to it and run by wait.
type recorder struct {
	mu     sync.Mutex
	posts  chan func()
	added  []platform.PadInfo
	gone   []platform.PadID
	events []platform.PadEvent
	// The extras that a HID driver reports (hid_linux_test.go).
	touches   []platform.PadTouch
	sensors   []platform.PadSensor
	batteries []platform.PadBattery
}

func (r *recorder) PadAdded(info platform.PadInfo) {
	r.mu.Lock()
	r.added = append(r.added, info)
	r.mu.Unlock()
}

func (r *recorder) PadRemoved(id platform.PadID) {
	r.mu.Lock()
	r.gone = append(r.gone, id)
	r.mu.Unlock()
}

func (r *recorder) PadEvents(events []platform.PadEvent) {
	r.mu.Lock()
	r.events = append(r.events, events...)
	r.mu.Unlock()
}

func (r *recorder) post(fn func()) { r.posts <- fn }

// wait runs the posted functions until cond is true, and fails the test after a timeout.
func (r *recorder) wait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		r.mu.Lock()
		ok := cond()
		r.mu.Unlock()
		if ok {
			return
		}
		select {
		case fn := <-r.posts:
			fn()
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

// settle runs what is posted now and what arrives in the next moment.
func (r *recorder) settle() {
	for {
		select {
		case fn := <-r.posts:
			fn()
		case <-time.After(150 * time.Millisecond):
			return
		}
	}
}

// logs collects the records of a logger.
type logs struct {
	mu      sync.Mutex
	records []slog.Record
}

func (l *logs) Enabled(context.Context, slog.Level) bool { return true }
func (l *logs) Handle(_ context.Context, r slog.Record) error {
	l.mu.Lock()
	l.records = append(l.records, r)
	l.mu.Unlock()
	return nil
}
func (l *logs) WithAttrs([]slog.Attr) slog.Handler { return l }
func (l *logs) WithGroup(string) slog.Handler      { return l }

// errTags returns the tags of the errors that were logged as the "err" attribute.
func (l *logs) errTags() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var tags []string
	for _, r := range l.records {
		r.Attrs(func(a slog.Attr) bool {
			if err, ok := a.Value.Any().(error); ok && a.Key == "err" {
				if appErr, ok := errors.FromError(err); ok {
					tags = append(tags, appErr.Tag)
				}
			}
			return true
		})
	}
	return tags
}

func (l *logs) messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, r := range l.records {
		out = append(out, r.Message)
	}
	return out
}

// fixture is a service that looks at a directory of fake event devices (FIFOs) and a fake sysfs.
type fixture struct {
	t       *testing.T
	dev     string
	sys     string
	s       *Service
	h       *recorder
	log     *logs
	capsMu  sync.Mutex
	caps    map[string]*caps
	writers map[string]*os.File

	// ff is what the fake kernel was asked to do with force feedback, and ffFail makes it refuse (under ffMu).
	ffMu     sync.Mutex
	ff       []string
	ffEffect ffEffect
	ffFail   error
	// ffPlayOnly makes ffFail apply to playing an effect only, not to uploading it.
	ffPlayOnly bool
	// readOnly makes the fake kernel refuse to open a device for writing.
	readOnly bool
}

// ffLog says what the fake kernel was asked to do with force feedback.
func (f *fixture) ffLog() []string {
	f.ffMu.Lock()
	defer f.ffMu.Unlock()
	return append([]string(nil), f.ff...)
}

// modes runs a test with the service told of the devices by inotify, and with it looking at the directory on a timer.
func modes(t *testing.T, run func(t *testing.T, f *fixture)) {
	t.Run("inotify", func(t *testing.T) {
		fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
		if err != nil {
			t.Skipf("this user has no inotify instance left (fs.inotify.max_user_instances): %v", err)
		}
		_ = syscall.Close(fd)
		run(t, newFixture(t, 0))
	})
	t.Run("polling", func(t *testing.T) { run(t, newFixture(t, 40*time.Millisecond)) })
}

func newFixture(t *testing.T, pollEvery time.Duration) *fixture {
	t.Helper()
	dir := t.TempDir()
	f := &fixture{
		t: t, dev: filepath.Join(dir, "input"), sys: filepath.Join(dir, "sys"),
		h: &recorder{posts: make(chan func(), 1024)}, log: &logs{}, caps: map[string]*caps{}, writers: map[string]*os.File{},
	}
	for _, d := range []string{f.dev, f.sys} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f.s = New(Opts{Post: f.h.post, Logger: slog.New(f.log), DevDir: f.dev, SysDir: f.sys, PollEvery: pollEvery})
	f.s.k = kernel{
		readAbs: func(fd, code int) (absInfo, error) {
			c := f.capsOfFD(fd)
			if c == nil {
				return absInfo{}, syscall.ENOTTY
			}
			return c.Info[code], nil
		},
		keyState: func(fd int) (bits, bool) {
			if c := f.capsOfFD(fd); c != nil {
				return bits{0}, true
			}
			return nil, false
		},
		setClock: func(int) error { return nil },
		openWritable: func(path string) (int, error) {
			f.ffMu.Lock()
			defer f.ffMu.Unlock()
			if f.readOnly {
				return -1, syscall.EACCES
			}
			f.ff = append(f.ff, "open "+filepath.Base(path))
			return syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
		},
		uploadEffect: func(fd int, e *ffEffect) error {
			f.ffMu.Lock()
			defer f.ffMu.Unlock()
			if f.ffFail != nil && !f.ffPlayOnly {
				return f.ffFail
			}
			if e.ID < 0 {
				e.ID = 7
			}
			f.ffEffect = *e
			f.ff = append(f.ff, fmt.Sprintf("upload %d", e.ID))
			return nil
		},
		playEffect: func(fd int, id int16, value int32) error {
			f.ffMu.Lock()
			defer f.ffMu.Unlock()
			if f.ffFail != nil {
				return f.ffFail
			}
			f.ff = append(f.ff, fmt.Sprintf("play %d %d", id, value))
			return nil
		},
	}
	t.Cleanup(func() {
		f.s.Stop()
		for _, w := range f.writers {
			_ = w.Close() // the test is over
		}
	})
	return f
}

func (f *fixture) capsOfFD(fd int) *caps {
	target, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", fd))
	if err != nil {
		return nil
	}
	f.capsMu.Lock()
	defer f.capsMu.Unlock()
	return f.caps[filepath.Base(target)]
}

// writeSysfs writes the description of a device into the fake sysfs.
func (f *fixture) writeSysfs(node string, c *caps) {
	f.t.Helper()
	files := map[string]string{
		"device/id/bustype": fmt.Sprintf("%04x\n", c.ID.Bus), "device/id/vendor": fmt.Sprintf("%04x\n", c.ID.Vendor),
		"device/id/product": fmt.Sprintf("%04x\n", c.ID.Product), "device/id/version": fmt.Sprintf("%04x\n", c.ID.Version),
		"device/name": c.Name + "\n", "device/capabilities/key": formatCapabilities(c.Key), "device/capabilities/abs": formatCapabilities(c.Abs),
		"device/capabilities/rel": formatCapabilities(c.Rel), "device/capabilities/ff": formatCapabilities(c.FF),
		"device/properties": formatCapabilities(c.Prop),
	}
	for rel, content := range files {
		path := filepath.Join(f.sys, node, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			f.t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	f.capsMu.Lock()
	f.caps[node] = c
	f.capsMu.Unlock()
}

// writeNode makes the event device of the node: a FIFO that the test holds the other end of.
func (f *fixture) writeNode(node string) *os.File {
	f.t.Helper()
	path := filepath.Join(f.dev, node)
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		f.t.Fatal(err)
	}
	w, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		f.t.Fatal(err)
	}
	f.writers[node] = w
	return w
}

// encodeEvents writes events as the kernel's records.
func encodeEvents(events ...inputEvent) []byte {
	var buf []byte
	for _, e := range events {
		rec := make([]byte, eventSize)
		tv := int(unsafe.Sizeof(syscall.Timeval{}))
		if tv == 16 {
			binary.NativeEndian.PutUint64(rec[0:], uint64(e.Time/time.Second))
			binary.NativeEndian.PutUint64(rec[8:], uint64(e.Time%time.Second/time.Microsecond))
		} else {
			binary.NativeEndian.PutUint32(rec[0:], uint32(e.Time/time.Second))
			binary.NativeEndian.PutUint32(rec[4:], uint32(e.Time%time.Second/time.Microsecond))
		}
		binary.NativeEndian.PutUint16(rec[tv:], e.Type)
		binary.NativeEndian.PutUint16(rec[tv+2:], e.Code)
		binary.NativeEndian.PutUint32(rec[tv+4:], uint32(e.Value))
		buf = append(buf, rec...)
	}
	return buf
}

// send writes events to the device, as the kernel would.
func (f *fixture) send(node string, events ...inputEvent) {
	f.t.Helper()
	if _, err := f.writers[node].Write(encodeEvents(events...)); err != nil {
		f.t.Fatal(err)
	}
}

func TestParseEvents(t *testing.T) {
	f := newFixture(t, 0)
	f.writers["event0"] = nil
	events := []inputEvent{
		{Time: 3*time.Second + 250*time.Microsecond, Type: evKey, Code: btnSouth, Value: 1},
		{Time: 4 * time.Second, Type: evAbs, Code: absX, Value: -32768},
		{Time: 4 * time.Second, Type: evSyn, Code: synReport, Value: 0},
	}
	// Written through a pipe, to see the records as bytes.
	p, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	defer func() { _ = w.Close() }()
	f.writers["event0"] = w
	f.send("event0", events...)
	buf := make([]byte, 4*eventSize)
	n, err := p.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3*eventSize {
		t.Fatalf("read %d bytes, want %d", n, 3*eventSize)
	}
	got := parseEvents(buf[:n+eventSize/2-eventSize/2], nil)
	if len(got) != 3 || got[0] != events[0] || got[1] != events[1] || got[2] != events[2] {
		t.Errorf("parseEvents() = %+v, want %+v", got, events)
	}
	// A partial record at the end is dropped.
	if got := parseEvents(buf[:2*eventSize+5], nil); len(got) != 2 {
		t.Errorf("a partial record gave %d events, want 2", len(got))
	}
	f.writers["event0"] = nil
}

func TestServiceReportsThePadsThatAreThere(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		f.writeSysfs("event0", xbox360())
		f.writeNode("event0")
		// A keyboard and a mouse are described in sysfs but have no node: if the service opened them the test would see a failure.
		f.writeSysfs("event1", &caps{Name: "Keyboard", Key: bitsOf(keyEsc, keyEnter, keySpace)})
		f.writeSysfs("event2", &caps{Name: "Mouse", Key: bitsOf(btnLeft), Rel: bitsOf(relX, relY)})

		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 && len(f.h.events) > 0 })
		f.h.settle()

		info := f.h.added[0]
		if info.Name != "Xbox 360 Controller" || info.Axes != 6 || info.Buttons != 11 || info.Hats != 1 || info.Standard {
			t.Errorf("info = %+v", info)
		}
		if want := gamepaddb.NewGUID(3, 0x045e, 0x028e, 0x0110, "", "Microsoft X-Box 360 pad", 0, 0); info.GUID != want {
			t.Errorf("GUID = %x, want %x", info.GUID, want)
		}
		if info.Path != filepath.Join(f.dev, "event0") || info.Battery != -1 {
			t.Errorf("Path = %q, Battery = %d", info.Path, info.Battery)
		}
		if !strings.Contains(info.Mapping, "leftx:a0,") || !strings.Contains(info.Mapping, "dpup:h0.1,") {
			t.Errorf("Mapping = %q", info.Mapping)
		}
		// The state the pad is in at the start is reported: 11 buttons, 6 axes and the hat.
		buttons, axes := 0, 0
		for _, e := range f.h.events {
			if e.Pad != info.ID {
				t.Errorf("report %+v is for another pad", e)
			}
			switch e.Kind {
			case platform.PadButton:
				buttons++
			case platform.PadAxis:
				axes++
			}
		}
		if buttons != 11 || axes != 6 {
			t.Errorf("the first reports: %d buttons and %d axes, want 11 and 6", buttons, axes)
		}
		for _, m := range f.log.messages() {
			if strings.Contains(m, "cannot") {
				t.Errorf("the service logged %q", m)
			}
		}
		if f.s.MappingPlatform() != "Linux" {
			t.Errorf("MappingPlatform() = %q", f.s.MappingPlatform())
		}
	})
}

func TestServiceReadsTheEventsOfAPad(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		f.writeSysfs("event0", xbox360())
		f.writeNode("event0")
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
		f.h.settle()
		f.h.mu.Lock()
		f.h.events = nil
		f.h.mu.Unlock()

		f.send("event0",
			ev(10, evKey, btnSouth, 1),
			ev(10, evAbs, absX, 1000),
			ev(10, evAbs, absHat0X, 1),
			ev(10, evSyn, synReport, 0),
			ev(11, evKey, btnSouth, 0),
			ev(11, evSyn, synReport, 0),
		)
		f.h.wait(t, "the reports", func() bool { return len(f.h.events) >= 4 })
		id := f.h.added[0].ID
		want := []platform.PadEvent{
			{Pad: id, Time: 10 * time.Second, Kind: platform.PadButton, Index: 0, Value: 1},
			{Pad: id, Time: 10 * time.Second, Kind: platform.PadAxis, Index: 0, Value: 1000},
			{Pad: id, Time: 10 * time.Second, Kind: platform.PadHat, Index: 0, Value: 2},
			{Pad: id, Time: 11 * time.Second, Kind: platform.PadButton, Index: 0, Value: 0},
		}
		for i, w := range want {
			if f.h.events[i] != w {
				t.Errorf("report %d = %+v, want %+v", i, f.h.events[i], w)
			}
		}
	})
}

// TestPadThatGoesDuringTheFirstLookIsLetGo removes a pad's node right after the first look at the devices: the pad must still be
// let go (step 36.2: a service that polls listed the nodes after the first look, so a node that went in between was never
// removed, and TestHIDPadGoesWithItsNode timed out under load).
func TestPadThatGoesDuringTheFirstLookIsLetGo(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		f.writeSysfs("event0", dualShock4())
		f.writeNode("event0")
		f.s.scanned = func() {
			if err := os.Remove(filepath.Join(f.dev, "event0")); err != nil {
				t.Error(err)
			}
		}
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
		f.h.wait(t, "the removal", func() bool { return len(f.h.gone) == 1 })
		if f.h.gone[0] != f.h.added[0].ID {
			t.Errorf("removed %d, added %d", f.h.gone[0], f.h.added[0].ID)
		}
	})
}

func TestServiceHotplug(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.settle()
		if len(f.h.added) != 0 {
			t.Fatalf("pads at the start: %+v", f.h.added)
		}

		// A pad is plugged in: its attributes first, then its node, as the kernel and udev do.
		f.writeSysfs("event5", dualShock4())
		f.writeNode("event5")
		f.h.wait(t, "the new pad", func() bool { return len(f.h.added) == 1 })
		if f.h.added[0].Name != "Sony Interactive Entertainment Wireless Controller" || f.h.added[0].Path != filepath.Join(f.dev, "event5") {
			t.Errorf("info = %+v", f.h.added[0])
		}

		// And unplugged.
		if err := os.Remove(filepath.Join(f.dev, "event5")); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the removal", func() bool { return len(f.h.gone) == 1 })
		if f.h.gone[0] != f.h.added[0].ID {
			t.Errorf("removed %d, added %d", f.h.gone[0], f.h.added[0].ID)
		}

		// A pad that comes back is a new one.
		f.writeNode2("event5")
		f.h.wait(t, "the pad again", func() bool { return len(f.h.added) == 2 })
		if f.h.added[1].ID == f.h.added[0].ID {
			t.Error("the numbers of pads are reused")
		}
	})
}

// writeNode2 makes the node again, after it was removed.
func (f *fixture) writeNode2(node string) {
	f.t.Helper()
	_ = f.writers[node].Close() // the old end of the removed node
	f.writeNode(node)
}

func TestServiceLeavesDevicesThatAreNotPadsAlone(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		// The node of a keyboard is made without anything that could be opened: opening it would fail with ENXIO, as a FIFO with no
		// reader does not open for writing, and the service logs such a failure.
		f.writeSysfs("event7", &caps{Name: "Keyboard", Key: bitsOf(keyEsc, keyEnter, keySpace, 30)})
		path := filepath.Join(f.dev, "event7")
		if err := os.WriteFile(path, nil, 0o000); err != nil {
			t.Fatal(err)
		}
		f.h.settle()
		if len(f.h.added) != 0 {
			t.Errorf("a keyboard became a pad: %+v", f.h.added)
		}
		if msgs := f.log.messages(); len(msgs) != 0 {
			t.Errorf("the service logged about a keyboard: %v", msgs)
		}
	})
}

func TestServiceTellsOfAPadThatCannotBeOpened(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can open every file")
	}
	modes(t, func(t *testing.T, f *fixture) {
		f.writeSysfs("event3", xbox360())
		path := filepath.Join(f.dev, "event3")
		if err := syscall.Mkfifo(path, 0o000); err != nil {
			t.Fatal(err)
		}
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.settle()
		if len(f.h.added) != 0 {
			t.Fatalf("a pad that cannot be read was reported: %+v", f.h.added)
		}
		msgs := f.log.messages()
		if len(msgs) != 1 || !strings.Contains(msgs[0], "may not read") {
			t.Fatalf("log = %v, want one note about the permissions", msgs)
		}
		if tags := f.log.errTags(); len(tags) != 1 || tags[0] != tagEvdevPermission {
			t.Errorf("tags = %v, want %s", tags, tagEvdevPermission)
		}

		// The permissions are set a moment after the node appears: the change of the attributes is the signal to try again.
		w, err := os.OpenFile(path, os.O_RDWR, 0)
		if err == nil {
			_ = w.Close()
			t.Fatal("the FIFO opened with mode 000")
		}
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatal(err)
		}
		f.writers["event3"], err = os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the pad after the permissions changed", func() bool { return len(f.h.added) == 1 })
	})
}

func TestServiceWaitsForSysfs(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		// The node is there before its attributes are.
		f.writeNode("event4")
		f.h.settle()
		if len(f.h.added) != 0 {
			t.Fatal("a pad without attributes was reported")
		}
		f.writeSysfs("event4", xbox360())
		f.h.wait(t, "the pad when sysfs is ready", func() bool { return len(f.h.added) == 1 })
	})
}

func TestServiceLosesAPadWhenItsDeviceEnds(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		f.writeSysfs("event0", xbox360())
		w := f.writeNode("event0")
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
		// A device that is unplugged answers reads with an error (ENODEV); the FIFO answers with the end of the file once its
		// writer has gone, which the service treats the same.
		_ = w.Close()
		f.h.wait(t, "the removal", func() bool { return len(f.h.gone) == 1 })
	})
}

func TestServiceStopAndStartAgain(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		f.writeSysfs("event0", xbox360())
		f.writeNode("event0")
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		if err := f.s.Start(f.h); err != nil {
			t.Fatalf("starting twice: %v", err)
		}
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
		f.h.settle()
		f.s.Stop()
		f.s.Stop()

		// What was posted before Stop does not reach the handler after it.
		before := len(f.h.events)
		f.send("event0", ev(1, evKey, btnSouth, 1), ev(1, evSyn, synReport, 0))
		f.h.settle()
		if len(f.h.events) != before {
			t.Errorf("reports arrived after Stop: %+v", f.h.events[before:])
		}

		if err := f.s.Start(f.h); err != nil {
			t.Fatalf("starting again: %v", err)
		}
		f.h.wait(t, "the pad again", func() bool { return len(f.h.added) == 2 })
	})
}

func TestServiceErrorsHaveTheirRoutes(t *testing.T) {
	t.Run("a directory that cannot be watched", func(t *testing.T) {
		s := New(Opts{Post: func(func()) {}, DevDir: filepath.Join(t.TempDir(), "nothing")})
		err := s.Start(&recorder{})
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if appErr.Tag != tagEvdevStartFailed || !appErr.HasRoute("Service.Start->Service.newRun.EvdevStartFailed") {
			t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
		}
	})

	f := newFixture(t, 0)
	r, err := f.s.newRun(f.h)
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()

	t.Run("a node that is not there", func(t *testing.T) {
		_, _, err := r.open("event9", xbox360())
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if appErr.Tag != tagEvdevOpenFailed || !appErr.HasRoute("run.open.EvdevDeviceOpenFailed") {
			t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
		}
	})

	t.Run("a node that cannot be read", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can open every file")
		}
		path := filepath.Join(f.dev, "event8")
		if err := os.WriteFile(path, nil, 0o000); err != nil {
			t.Fatal(err)
		}
		_, _, err := r.open("event8", xbox360())
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if appErr.Tag != tagEvdevPermission || !appErr.HasRoute("run.open.EvdevDevicePermission") {
			t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
		}
	})

	t.Run("an axis that cannot be read", func(t *testing.T) {
		f.writeSysfs("event6", xbox360())
		f.writeNode("event6")
		// The fake kernel has no caps for a node it was not told of.
		f.capsMu.Lock()
		delete(f.caps, "event6")
		f.capsMu.Unlock()
		_, _, err := r.open("event6", xbox360())
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if appErr.Tag != tagEvdevIoctlFailed || !appErr.HasRoute("run.open.EvdevIoctlFailed") {
			t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
		}
	})

	t.Run("a directory that cannot be listed", func(t *testing.T) {
		g := newFixture(t, 40*time.Millisecond)
		r2, err := g.s.newRun(g.h)
		if err != nil {
			t.Fatal(err)
		}
		defer r2.close()
		if err := os.Remove(g.dev); err != nil {
			t.Fatal(err)
		}
		r2.scan()
		if tags := g.log.errTags(); !slices.Contains(tags, tagEvdevScanFailed) {
			t.Errorf("tags = %v, want %s", tags, tagEvdevScanFailed)
		}
	})

	t.Run("a device that cannot be watched", func(t *testing.T) {
		g := newFixture(t, 40*time.Millisecond)
		g.writeSysfs("event0", xbox360())
		g.writeNode("event0")
		r2, err := g.s.newRun(g.h)
		if err != nil {
			t.Fatal(err)
		}
		_ = syscall.Close(r2.epfd) // the epoll descriptor is gone: adding the device to it fails
		r2.consider("event0")
		if tags := g.log.errTags(); !slices.Contains(tags, tagEvdevWatchFailed) {
			t.Errorf("tags = %v, want %s", tags, tagEvdevWatchFailed)
		}
		r2.epfd = -1
		r2.close()
	})

	t.Run("a loop that cannot wait", func(t *testing.T) {
		g := newFixture(t, 40*time.Millisecond)
		r2, err := g.s.newRun(g.h)
		if err != nil {
			t.Fatal(err)
		}
		_ = syscall.Close(r2.epfd)
		r2.epfd = -1
		r2.loop()
		if tags := g.log.errTags(); !slices.Contains(tags, tagEvdevWaitFailed) {
			t.Errorf("tags = %v, want %s", tags, tagEvdevWaitFailed)
		}
	})
}

func TestThePadsOfTheServiceAreTheServiceOfTheMachine(t *testing.T) {
	// A service on the real directories starts and stops without a pad; on a machine with pads it finds them.
	s := New(Opts{Post: func(fn func()) { fn() }})
	h := &recorder{posts: make(chan func(), 16)}
	if err := s.Start(h); err != nil {
		t.Skipf("this machine has no /dev/input to watch: %v", err)
	}
	s.Stop()
}

func TestTriggersAndLightAreNotAvailableOnLinuxYet(t *testing.T) {
	s := New(Opts{Post: func(func()) {}})
	tests := []struct {
		name string
		err  error
		tag  string
	}{
		{"triggers", s.RumbleTriggers(1, 1, 1, time.Second), platform.TagGamepadRumbleUnavailable},
		{"light", s.SetLED(1, 1, 2, 3), platform.TagGamepadLEDUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", tt.err, tt.err)
			}
			if appErr.Tag != tt.tag {
				t.Errorf("tag = %s, want %s", appErr.Tag, tt.tag)
			}
		})
	}
}

func TestFFEffectLayout(t *testing.T) {
	want := 48
	if ptrSize == 4 {
		want = 44
	}
	if got := int(unsafe.Sizeof(ffEffect{})); got != want {
		t.Errorf("sizeof(ffEffect) = %d, want %d (the kernel's struct ff_effect)", got, want)
	}
	if got := unsafe.Offsetof(ffEffect{}.Union); got != 16 {
		t.Errorf("the union is at %d, want 16", got)
	}
	e := rumbleEffect(-1, 0x1234, 0xabcd, 1500*time.Millisecond)
	if e.Type != ffRumble || e.ID != -1 || e.ReplayLength != 1500 || e.ReplayDelay != 0 || e.Direction != 0 {
		t.Errorf("rumbleEffect() = %+v", e)
	}
	if got := binary.NativeEndian.Uint16(e.Union[0:]); got != 0x1234 {
		t.Errorf("strong magnitude = %#x", got)
	}
	if got := binary.NativeEndian.Uint16(e.Union[2:]); got != 0xabcd {
		t.Errorf("weak magnitude = %#x", got)
	}
	// The ioctl number of EVIOCSFF on 64-bit machines is 0x40304580 (linux/input.h).
	if ptrSize == 8 && eviocsff() != 0x40304580 {
		t.Errorf("EVIOCSFF = %#x, want 0x40304580", eviocsff())
	}
	for _, tt := range []struct {
		d    time.Duration
		want uint16
	}{{0, 1}, {time.Microsecond, 1}, {time.Hour, 0xffff}, {65535 * time.Millisecond, 0xffff}} {
		if got := rumbleEffect(0, 1, 1, tt.d).ReplayLength; got != tt.want {
			t.Errorf("length of %v = %d, want %d", tt.d, got, tt.want)
		}
	}
	ev := ffEvent(7, 1)
	if len(ev) != eventSize {
		t.Fatalf("len(ffEvent) = %d, want %d", len(ev), eventSize)
	}
	parsed := parseEvents(ev, nil)
	if len(parsed) != 1 || parsed[0].Type != evFF || parsed[0].Code != 7 || parsed[0].Value != 1 {
		t.Errorf("ffEvent(7, 1) parses as %+v", parsed)
	}
}

func rumblePad() *caps {
	c := xbox360()
	c.FF = bitsOf(ffRumble)
	return c
}

func TestServiceRumble(t *testing.T) {
	modes(t, func(t *testing.T, f *fixture) {
		f.writeSysfs("event0", rumblePad())
		f.writeNode("event0")
		f.writeSysfs("event1", xbox360()) // no force feedback
		f.writeNode("event1")
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "two pads", func() bool { return len(f.h.added) == 2 })
		var withFF, without platform.PadInfo
		for _, info := range f.h.added {
			if filepath.Base(info.Path) == "event0" {
				withFF = info
			} else {
				without = info
			}
		}
		if !withFF.Rumble || without.Rumble {
			t.Errorf("Rumble: %v and %v, want true and false", withFF.Rumble, without.Rumble)
		}
		if got := f.ffLog(); len(got) != 1 || got[0] != "open event0" {
			t.Errorf("only the pad that can rumble is opened for writing: %v", got)
		}

		if err := f.s.Rumble(withFF.ID, 0x8000, 0x4000, 2*time.Second); err != nil {
			t.Fatal(err)
		}
		f.ffMu.Lock()
		e := f.ffEffect
		f.ffMu.Unlock()
		if e.ReplayLength != 2000 || binary.NativeEndian.Uint16(e.Union[0:]) != 0x8000 || binary.NativeEndian.Uint16(e.Union[2:]) != 0x4000 {
			t.Errorf("uploaded %+v", e)
		}
		// The second rumble updates the effect that the kernel made, and plays it again.
		if err := f.s.Rumble(withFF.ID, 1, 2, time.Second); err != nil {
			t.Fatal(err)
		}
		if err := f.s.Rumble(withFF.ID, 0, 0, 0); err != nil {
			t.Fatal(err)
		}
		want := []string{"open event0", "upload 7", "play 7 1", "upload 7", "play 7 1", "play 7 0"}
		if got := f.ffLog(); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("force feedback calls = %v, want %v", got, want)
		}

		err := f.s.Rumble(without.ID, 1, 1, time.Second)
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if appErr.Tag != platform.TagGamepadRumbleUnavailable || !appErr.HasRoute("Service.Rumble.GamepadRumbleUnavailable") {
			t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
		}

		// A pad that is gone cannot rumble.
		_ = f.writers["event0"].Close() // the pad is unplugged
		f.h.wait(t, "the pad is removed", func() bool { return len(f.h.gone) == 1 })
		if err := f.s.Rumble(withFF.ID, 1, 1, time.Second); err == nil {
			t.Error("a pad that is gone rumbled")
		}
	})
}

func TestServiceRumbleStopWithoutPlayingDoesNothing(t *testing.T) {
	f := newFixture(t, 40*time.Millisecond)
	f.writeSysfs("event0", rumblePad())
	f.writeNode("event0")
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
	if err := f.s.Rumble(f.h.added[0].ID, 0, 0, time.Second); err != nil {
		t.Fatal(err)
	}
	if got := f.ffLog(); len(got) != 1 {
		t.Errorf("a stop before any rumble made calls: %v", got)
	}
}

func TestServiceRumbleOfAReadOnlyPad(t *testing.T) {
	f := newFixture(t, 40*time.Millisecond)
	f.readOnly = true
	f.writeSysfs("event0", rumblePad())
	f.writeNode("event0")
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
	if f.h.added[0].Rumble {
		t.Error("a pad that could not be opened for writing says it can rumble")
	}
	if err := f.s.Rumble(f.h.added[0].ID, 1, 1, time.Second); err == nil {
		t.Error("a read only pad rumbled")
	}
}

func TestServiceRumbleKernelRefuses(t *testing.T) {
	for _, tt := range []struct {
		name  string
		low   uint16
		prior bool
	}{
		{"upload", 1, false},
		{"play", 1, false},
		{"stop", 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, 40*time.Millisecond)
			f.writeSysfs("event0", rumblePad())
			f.writeNode("event0")
			if err := f.s.Start(f.h); err != nil {
				t.Fatal(err)
			}
			f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
			id := f.h.added[0].ID
			if tt.prior {
				if err := f.s.Rumble(id, 1, 1, time.Second); err != nil {
					t.Fatal(err)
				}
			}
			f.ffMu.Lock()
			f.ffFail, f.ffPlayOnly = errors.NewErr(tagEvdevRumbleFailed, syscall.EIO), tt.name == "play"
			f.ffMu.Unlock()
			err := f.s.Rumble(id, tt.low, tt.low, time.Second)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			// The refusal is made by the fake kernel, so the route of its error starts in the test; the routes of the real
			// calls are in TestKernelRumbleCallsFail. What Service.Rumble must do is return it untouched.
			if appErr.Tag != tagEvdevRumbleFailed {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
			if !errors.Is(err, syscall.EIO) {
				t.Errorf("the cause is lost: %v", err)
			}
		})
	}
}

func TestKernelRumbleCallsFail(t *testing.T) {
	for _, tt := range []struct {
		name  string
		call  func() error
		route string
	}{
		{"upload", func() error { return uploadEffect(-1, &ffEffect{}) }, "uploadEffect.EvdevRumbleFailed"},
		{"play", func() error { return playEffect(-1, 0, 1) }, "playEffect.EvdevRumbleFailed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if !errors.Is(err, syscall.EBADF) {
				t.Errorf("the cause is lost: %v", err)
			}
		})
	}
}

func TestSysfsFileThatCannotBeRead(t *testing.T) {
	_, err := dirFiles(t.TempDir()).read("device/name")
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tagEvdevSysfsReadFailed || !appErr.HasRoute("dirFiles.read.EvdevSysfsReadFailed") {
		t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
	}
}

// openFDs counts the descriptors of the test process.
func openFDs() int {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0
	}
	return len(entries)
}

// settledFDs is the smallest of a few counts of openFDs a few milliseconds apart. A service that polls opens a directory and the
// attributes of the nodes for a moment at each look, and a count made during one is one too many (step 36.2: the leak test saw
// 268 descriptors where there were 267 under load); a descriptor that leaked is in every count.
func settledFDs() int {
	least := openFDs()
	for range 9 {
		time.Sleep(2 * time.Millisecond)
		least = min(least, openFDs())
	}
	return least
}

// TestLeaksPadsPluggedAndUnplugged plugs a pad in and out many times: the descriptors, the goroutines and the heap must stay
// where they were (SPEC §15 "Memory over time").
func TestLeaksPadsPluggedAndUnplugged(t *testing.T) {
	f := newFixture(t, 5*time.Millisecond)
	f.writeSysfs("event0", xbox360())
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	plugs := 0
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("log: %v / tags %v", f.log.messages(), f.log.errTags())
		}
	})
	step := func() {
		plugs++
		f.writeNode("event0")
		f.send("event0", ev(1, evKey, btnSouth, 1), ev(1, evSyn, synReport, 0))
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == plugs })
		if err := os.Remove(filepath.Join(f.dev, "event0")); err != nil {
			t.Fatal(err)
		}
		_ = f.writers["event0"].Close() // the test's end of the removed node
		f.h.wait(t, "the removal", func() bool { return len(f.h.gone) == plugs })
		f.h.mu.Lock()
		f.h.added, f.h.gone, f.h.events = nil, nil, nil
		f.h.mu.Unlock()
		plugs = 0
	}
	leaktest.Run(t, "evdev/plug", func() func() { return step },
		leaktest.Opts{N: 40, Warmup: 5, Counters: map[string]func() int{"descriptors": settledFDs}})
}
