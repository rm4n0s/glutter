//go:build linux && !android

package evdev

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform"
)

// These tests make virtual pads with /dev/uinput, which the kernel shows as real event devices, and run the service on the
// real /dev/input (SPEC §28.8). They need write access to /dev/uinput, and they skip without it; a udev rule gives it:
//
//	echo 'KERNEL=="uinput", TAG+="uaccess"' | sudo tee /etc/udev/rules.d/70-glutter-uinput.rules
//	sudo udevadm control --reload && sudo udevadm trigger /dev/uinput
//
// The new device must also be readable by the user: on a desktop the login session gives that for joysticks.

const uinputPath = "/dev/uinput"

func uio(dir, nr, size uintptr) uintptr { return ioc(dir, 'U', nr, size) }

// absSpec is the range of an axis of a virtual pad.
type absSpec struct{ min, max, fuzz, flat int32 }

// virtualPad is a pad that the kernel shows as an event device.
type virtualPad struct {
	t *testing.T
	f *os.File
}

// newVirtualPad makes the pad; the last arguments are the force feedback effects it can play (FF_RUMBLE), none for a pad
// without force feedback.
func newVirtualPad(t *testing.T, name string, vendor, product, version uint16, keys []int, abs map[int]absSpec, ff ...int) *virtualPad {
	t.Helper()
	mode := os.O_WRONLY
	if len(ff) > 0 {
		// The kernel asks the program for each effect that is uploaded, played or erased: they are read from the descriptor.
		mode = os.O_RDWR
	}
	f, err := os.OpenFile(uinputPath, mode|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Skipf("no write access to %s (%v): see the udev rule in the comment of this file", uinputPath, err)
	}
	p := &virtualPad{t: t, f: f}
	t.Cleanup(p.Close)
	set := func(req uintptr, v int) {
		t.Helper()
		arg := int32(v)
		if err := ioctl(int(f.Fd()), req, unsafe.Pointer(&arg)); err != nil {
			t.Fatalf("uinput ioctl 0x%x(%d): %v", req, v, err)
		}
	}
	setEv, setKey, setAbs := uio(iocWrite, 100, 4), uio(iocWrite, 101, 4), uio(iocWrite, 103, 4)
	set(setEv, evKey)
	set(setEv, evAbs)
	set(setEv, evSyn)
	for _, k := range keys {
		set(setKey, k)
	}
	if len(ff) > 0 {
		set(setEv, evFF)
		for _, e := range ff {
			set(uio(iocWrite, 107, 4), e) // UI_SET_FFBIT
		}
	}
	// struct uinput_user_dev: the name (80 bytes), the id, ff_effects_max, then absmax, absmin, absfuzz and absflat (64 each).
	const devSize = 80 + 8 + 4 + 4*64*4
	dev := make([]byte, devSize)
	copy(dev, name)
	binary.NativeEndian.PutUint16(dev[80:], 3) // BUS_USB
	binary.NativeEndian.PutUint16(dev[82:], vendor)
	binary.NativeEndian.PutUint16(dev[84:], product)
	binary.NativeEndian.PutUint16(dev[86:], version)
	if len(ff) > 0 {
		binary.NativeEndian.PutUint32(dev[88:], 16) // ff_effects_max
	}
	for code, a := range abs {
		set(setAbs, code)
		binary.NativeEndian.PutUint32(dev[92+4*code:], uint32(a.max))
		binary.NativeEndian.PutUint32(dev[92+256+4*code:], uint32(a.min))
		binary.NativeEndian.PutUint32(dev[92+512+4*code:], uint32(a.fuzz))
		binary.NativeEndian.PutUint32(dev[92+768+4*code:], uint32(a.flat))
	}
	if _, err := f.Write(dev); err != nil {
		t.Fatalf("uinput write: %v", err)
	}
	if err := ioctl(int(f.Fd()), uio(0, 1, 0), nil); err != nil {
		t.Fatalf("UI_DEV_CREATE: %v", err)
	}
	return p
}

// Send makes the pad report events; a SYN_REPORT ends them.
func (p *virtualPad) Send(events ...inputEvent) {
	p.t.Helper()
	events = append(events, inputEvent{Type: evSyn, Code: synReport})
	if _, err := p.f.Write(encodeEvents(events...)); err != nil {
		p.t.Fatalf("uinput event: %v", err)
	}
}

// Close removes the pad.
func (p *virtualPad) Close() {
	if p.f == nil {
		return
	}
	_ = ioctl(int(p.f.Fd()), uio(0, 2, 0), nil) // the device may be gone already
	_ = p.f.Close()
	p.f = nil
}

func xboxLikeKeys() []int {
	return []int{btnSouth, btnEast, btnNorth, btnWest, btnTL, btnTR, btnSelect, btnStart, btnMode, btnThumbL, btnThumbR}
}

func xboxLikeAxes() map[int]absSpec {
	s := absSpec{-32768, 32767, 16, 128}
	t := absSpec{0, 255, 0, 0}
	h := absSpec{-1, 1, 0, 0}
	return map[int]absSpec{absX: s, absY: s, absRX: s, absRY: s, absZ: t, absRZ: t, absHat0X: h, absHat0Y: h}
}

// realService starts a service on the real directories and waits for the pad with the name.
func realService(t *testing.T, name string) (*Service, *recorder, *logs, platform.PadInfo) {
	t.Helper()
	h := &recorder{posts: make(chan func(), 1024)}
	l := &logs{}
	s := New(Opts{Post: h.post, Logger: slog.New(l)})
	if err := s.Start(h); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	return s, h, l, platform.PadInfo{}
}

func findPad(t *testing.T, h *recorder, l *logs, name string) platform.PadInfo {
	t.Helper()
	var info platform.PadInfo
	deadline := time.After(10 * time.Second)
	for {
		h.mu.Lock()
		for _, a := range h.added {
			if strings.Contains(a.Name, name) {
				info = a
			}
		}
		h.mu.Unlock()
		if info.ID != 0 {
			return info
		}
		for _, m := range l.messages() {
			if strings.Contains(m, "may not read") {
				t.Skipf("the virtual pad cannot be read by this user (%s)", m)
			}
		}
		select {
		case fn := <-h.posts:
			fn()
		case <-deadline:
			t.Fatalf("the virtual pad %q did not show up; log: %v", name, l.messages())
		}
	}
}

func TestVirtualXboxPad(t *testing.T) {
	pad := newVirtualPad(t, "Glutter Test Xbox Pad", 0x045e, 0x028e, 0x0114, xboxLikeKeys(), xboxLikeAxes())
	_, h, l, _ := realService(t, "")
	info := findPad(t, h, l, "Xbox")
	if info.Axes != 6 || info.Buttons != 11 || info.Hats != 1 || info.Standard || !strings.HasPrefix(info.Path, "/dev/input/event") {
		t.Errorf("info = %+v", info)
	}
	if !strings.Contains(info.Mapping, "a:b0,") || !strings.Contains(info.Mapping, "lefttrigger:a2,") || !strings.Contains(info.Mapping, "dpup:h0.1,") {
		t.Errorf("Mapping = %q", info.Mapping)
	}

	h.settle()
	h.mu.Lock()
	h.events = nil
	h.mu.Unlock()
	start := monotonicNow()
	pad.Send(
		inputEvent{Type: evKey, Code: btnEast, Value: 1},
		inputEvent{Type: evAbs, Code: absX, Value: 20000},
		inputEvent{Type: evAbs, Code: absHat0Y, Value: -1},
	)
	h.wait(t, "the reports of the virtual pad", func() bool { return len(h.events) >= 3 })
	want := map[platform.PadEvent]bool{
		{Pad: info.ID, Kind: platform.PadButton, Index: 1, Value: 1}:   false,
		{Pad: info.ID, Kind: platform.PadAxis, Index: 0, Value: 20000}: false,
		{Pad: info.ID, Kind: platform.PadHat, Index: 0, Value: 1}:      false,
	}
	for _, e := range h.events {
		stamp := e.Time
		if stamp < start || stamp > monotonicNow() {
			t.Errorf("report %+v is not stamped with the monotonic clock (now %v)", e, monotonicNow())
		}
		e.Time = 0
		if _, ok := want[e]; ok {
			want[e] = true
		}
	}
	for e, seen := range want {
		if !seen {
			t.Errorf("no report %+v in %+v", e, h.events)
		}
	}

	pad.Close()
	h.wait(t, "the removal of the virtual pad", func() bool { return len(h.gone) > 0 && h.gone[len(h.gone)-1] == info.ID })
}

func TestVirtualGenericJoystick(t *testing.T) {
	// A joystick that is not a gamepad by the specification: it is a pad (it has no mapping in the table, so the layer above
	// does not make it a gamepad) without a mapping text from the backend.
	pad := newVirtualPad(t, "Glutter Test Flight Stick", 0x1234, 0x5678, 1, []int{btnTrigger, btnTrigger + 1},
		map[int]absSpec{absX: {-512, 511, 3, 15}, absY: {-512, 511, 3, 15}, absThrot: {0, 255, 0, 0}})
	_, h, l, _ := realService(t, "")
	info := findPad(t, h, l, "Flight Stick")
	if info.Axes != 3 || info.Buttons != 2 || info.Hats != 0 || info.Mapping != "" {
		t.Errorf("info = %+v", info)
	}
	pad.Close()
	h.wait(t, "the removal", func() bool { return len(h.gone) > 0 })
}

// ffUpload is the kernel's struct uinput_ff_upload and ffErase its struct uinput_ff_erase.
type ffUpload struct {
	RequestID uint32
	Retval    int32
	Effect    ffEffect
	Old       ffEffect
}

type ffErase struct {
	RequestID uint32
	Retval    int32
	EffectID  uint32
}

// ffServer answers what the kernel asks of a virtual pad with force feedback: it takes each uploaded effect, erases, and notes
// the effects that are played. The kernel waits for the answers (an upload by the service blocks until it is given, and so does
// the erasing of the effects when the service lets go of the pad).
type ffServer struct {
	mu       sync.Mutex
	uploaded []ffEffect
	played   []string
	stop     chan struct{}
	done     chan struct{}
}

const (
	evUinput     = 0x0101
	uiFFUpload   = 1
	uiFFErase    = 2
	uiBeginUp    = 200
	uiEndUp      = 201
	uiBeginErase = 202
	uiEndErase   = 203
)

func serveFF(t *testing.T, p *virtualPad) *ffServer {
	t.Helper()
	s := &ffServer{stop: make(chan struct{}), done: make(chan struct{})}
	fd := int(p.f.Fd())
	go func() {
		defer close(s.done)
		buf := make([]byte, 16*eventSize)
		for {
			select {
			case <-s.stop:
				return
			default:
			}
			n, err := syscall.Read(fd, buf)
			if err != nil || n <= 0 {
				time.Sleep(2 * time.Millisecond)
				continue
			}
			for _, ev := range parseEvents(buf[:n], nil) {
				switch {
				case ev.Type == evUinput && ev.Code == uiFFUpload:
					up := ffUpload{RequestID: uint32(ev.Value)}
					if err := ioctl(fd, ioc(3, 'U', uiBeginUp, unsafe.Sizeof(up)), unsafe.Pointer(&up)); err != nil {
						continue
					}
					s.mu.Lock()
					s.uploaded = append(s.uploaded, up.Effect)
					s.mu.Unlock()
					up.Retval = 0
					_ = ioctl(fd, ioc(iocWrite, 'U', uiEndUp, unsafe.Sizeof(up)), unsafe.Pointer(&up)) // the kernel times out if it fails
				case ev.Type == evUinput && ev.Code == uiFFErase:
					er := ffErase{RequestID: uint32(ev.Value)}
					if err := ioctl(fd, ioc(3, 'U', uiBeginErase, unsafe.Sizeof(er)), unsafe.Pointer(&er)); err != nil {
						continue
					}
					er.Retval = 0
					_ = ioctl(fd, ioc(iocWrite, 'U', uiEndErase, unsafe.Sizeof(er)), unsafe.Pointer(&er)) // the same
				case ev.Type == evFF:
					s.mu.Lock()
					s.played = append(s.played, fmt.Sprintf("%d:%d", ev.Code, ev.Value))
					s.mu.Unlock()
				}
			}
		}
	}()
	t.Cleanup(func() {
		close(s.stop)
		<-s.done
	})
	return s
}

func TestVirtualPadRumble(t *testing.T) {
	pad := newVirtualPad(t, "Glutter Test Rumble Pad", 0x045e, 0x028e, 0x0114, xboxLikeKeys(), xboxLikeAxes(), ffRumble)
	ff := serveFF(t, pad)
	s, h, l, _ := realService(t, "")
	info := findPad(t, h, l, "Xbox")
	if !info.Rumble {
		t.Fatalf("the pad says it cannot rumble: %+v", info)
	}
	if err := s.Rumble(info.ID, 0xc000, 0x2000, 750*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		ff.mu.Lock()
		n, played := len(ff.uploaded), append([]string(nil), ff.played...)
		ff.mu.Unlock()
		if n == 1 && len(played) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("uploads %d, played %v", n, played)
		}
		time.Sleep(5 * time.Millisecond)
	}
	ff.mu.Lock()
	e := ff.uploaded[0]
	ff.mu.Unlock()
	if e.Type != ffRumble || e.ReplayLength != 750 || binary.NativeEndian.Uint16(e.Union[0:]) != 0xc000 || binary.NativeEndian.Uint16(e.Union[2:]) != 0x2000 {
		t.Errorf("uploaded %+v", e)
	}
	if err := s.Rumble(info.ID, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	pad.Close()
}
