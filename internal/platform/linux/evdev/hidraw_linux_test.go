//go:build linux && !android

package evdev

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

func newFIFO(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "hidraw0")
	if err := syscall.Mkfifo(p, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// pipeTransport is a transport over the two ends of a pipe, which gives it the errors that a hidraw node gives when its device
// is gone: the read end hangs up when the write end is closed, and the write end cannot be read.
func pipeTransport(t *testing.T, k hidrawKernel) (tr *hidrawTransport, readEnd, writeEnd int) {
	t.Helper()
	var fds [2]int
	if err := syscall.Pipe2(fds[:], syscall.O_NONBLOCK|syscall.O_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	epfd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		t.Fatal(err)
	}
	ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fds[0])}
	if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, fds[0], &ev); err != nil {
		t.Fatal(err)
	}
	return &hidrawTransport{k: k, path: "/dev/hidraw9", fd: fds[0], epfd: epfd}, fds[0], fds[1]
}

func TestHidrawRequestNumbers(t *testing.T) {
	// HIDIOCSFEATURE(9) and HIDIOCGFEATURE(9) as linux/hidraw.h makes them.
	for _, tc := range []struct {
		name      string
		got, want uintptr
	}{
		{"set", hidiocsfeature(9), 0xC0094806},
		{"get", hidiocgfeature(9), 0xC0094807},
		{"get 64", hidiocgfeature(64), 0xC0404807},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: %#x, want %#x", tc.name, tc.got, tc.want)
		}
	}
}

func TestHidrawReadsAndWrites(t *testing.T) {
	path := newFIFO(t)
	tr, err := openHidraw(path, realHidrawKernel())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tr.Close() }() // closed by the test already, or never open
	buf := make([]byte, 64)
	start := time.Now()
	if n, err := tr.Read(buf, 20*time.Millisecond); n != 0 || err != nil {
		t.Errorf("a read with no report gave %d, %v", n, err)
	}
	if time.Since(start) < 15*time.Millisecond {
		t.Error("the read did not wait for its time")
	}
	// What is written to a FIFO that is open for reading and writing can be read back.
	want := []byte{0x05, 1, 2, 3}
	if n, err := tr.Write(want); n != len(want) || err != nil {
		t.Fatalf("write = %d, %v", n, err)
	}
	n, err := tr.Read(buf, time.Second)
	if err != nil || !bytes.Equal(buf[:n], want) {
		t.Errorf("read = %x, %v", buf[:n], err)
	}
	// A timeout of less than a millisecond still waits a little.
	if n, err := tr.Read(buf, time.Microsecond); n != 0 || err != nil {
		t.Errorf("short read = %d, %v", n, err)
	}
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tr.Close(); err != nil {
		t.Errorf("the second close failed: %v", err)
	}
}

func TestHidrawFeatureReportsUseTheIoctls(t *testing.T) {
	var gotReq []uintptr
	var gotData [][]byte
	k := hidrawKernel{ioctl: func(fd int, req uintptr, arg unsafe.Pointer) (int, error) {
		gotReq = append(gotReq, req)
		b := (*[64]byte)(arg)
		gotData = append(gotData, append([]byte(nil), b[:4]...))
		b[1] = 0x77 // what the pad answers
		return 4, nil
	}}
	path := newFIFO(t)
	tr, err := openHidraw(path, k)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tr.Close() }() // closed by the test already, or never open
	if n, err := tr.SendFeature([]byte{0x02, 1, 2, 3}); n != 4 || err != nil {
		t.Fatalf("send = %d, %v", n, err)
	}
	buf := []byte{0x05, 0, 0, 0}
	if n, err := tr.GetFeature(buf); n != 4 || err != nil || buf[1] != 0x77 {
		t.Fatalf("get = %d, %v, %x", n, err, buf)
	}
	if gotReq[0] != hidiocsfeature(4) || gotReq[1] != hidiocgfeature(4) || gotData[0][0] != 0x02 || gotData[1][0] != 0x05 {
		t.Errorf("requests = %#x, data = %x", gotReq, gotData)
	}
}

func TestHidrawErrors(t *testing.T) {
	failing := hidrawKernel{ioctl: func(int, uintptr, unsafe.Pointer) (int, error) { return 0, syscall.EPIPE }}
	for _, tc := range []struct {
		name  string
		route string
		tag   string
		run   func(t *testing.T) error
	}{
		{"open of a node that is not there", "openHidraw.HidrawOpenFailed", tagHidrawOpenFailed, func(t *testing.T) error {
			_, err := openHidraw(filepath.Join(t.TempDir(), "hidraw0"), realHidrawKernel())
			return err
		}},
		{"open of a node that the user may not open", "openHidraw.HidrawPermission", tagHidrawPermission, func(t *testing.T) error {
			if os.Geteuid() == 0 {
				t.Skip("root opens everything")
			}
			p := filepath.Join(t.TempDir(), "hidraw0")
			if err := os.WriteFile(p, nil, 0); err != nil {
				t.Fatal(err)
			}
			_, err := openHidraw(p, realHidrawKernel())
			return err
		}},
		{"open of a file that epoll does not take", "openHidraw.HidrawOpenFailed", tagHidrawOpenFailed, func(t *testing.T) error {
			p := filepath.Join(t.TempDir(), "hidraw0")
			if err := os.WriteFile(p, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := openHidraw(p, realHidrawKernel())
			return err
		}},
		{"read of a node that hung up", "hidrawTransport.Read.HidrawReadFailed", tagHidrawReadFailed, func(t *testing.T) error {
			tr, _, w := pipeTransport(t, realHidrawKernel())
			defer func() { _ = tr.Close() }() // closed by the test already, or never open
			_ = syscall.Close(w)
			_, err := tr.Read(make([]byte, 8), time.Second)
			return err
		}},
		{"read after the close", "hidrawTransport.Read.HidrawClosed", tagHidrawClosed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, realHidrawKernel())
			_ = tr.Close()
			_, err := tr.Read(make([]byte, 8), time.Millisecond)
			return err
		}},
		{"write that fails", "hidrawTransport.Write.HidrawWriteFailed", tagHidrawWriteFailed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, realHidrawKernel())
			defer func() { _ = tr.Close() }() // closed by the test already, or never open
			_, err := tr.Write([]byte{1, 2})  // the read end of a pipe cannot be written
			return err
		}},
		{"write after the close", "hidrawTransport.Write.HidrawClosed", tagHidrawClosed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, realHidrawKernel())
			_ = tr.Close()
			_, err := tr.Write([]byte{1})
			return err
		}},
		{"feature report sent that fails", "hidrawTransport.SendFeature.HidrawFeatureFailed", tagHidrawFeatureFailed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, failing)
			defer func() { _ = tr.Close() }() // closed by the test already, or never open
			_, err := tr.SendFeature([]byte{2, 0})
			return err
		}},
		{"empty feature report sent", "hidrawTransport.SendFeature.HidrawFeatureFailed", tagHidrawFeatureFailed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, failing)
			defer func() { _ = tr.Close() }() // closed by the test already, or never open
			_, err := tr.SendFeature(nil)
			return err
		}},
		{"feature report got that fails", "hidrawTransport.GetFeature.HidrawFeatureFailed", tagHidrawFeatureFailed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, failing)
			defer func() { _ = tr.Close() }() // closed by the test already, or never open
			_, err := tr.GetFeature([]byte{2, 0})
			return err
		}},
		{"empty feature report got", "hidrawTransport.GetFeature.HidrawFeatureFailed", tagHidrawFeatureFailed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, failing)
			defer func() { _ = tr.Close() }() // closed by the test already, or never open
			_, err := tr.GetFeature(nil)
			return err
		}},
		{"feature report after the close", "hidrawTransport.SendFeature.HidrawClosed", tagHidrawClosed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, failing)
			_ = tr.Close()
			_, err := tr.SendFeature([]byte{2, 0})
			return err
		}},
		{"get feature report after the close", "hidrawTransport.GetFeature.HidrawClosed", tagHidrawClosed, func(t *testing.T) error {
			tr, _, _ := pipeTransport(t, failing)
			_ = tr.Close()
			_, err := tr.GetFeature([]byte{2, 0})
			return err
		}},
		{"close that fails", "hidrawTransport.Close.HidrawCloseFailed", tagHidrawCloseFailed, func(t *testing.T) error {
			tr, r, w := pipeTransport(t, failing)
			_ = syscall.Close(w)
			_ = syscall.Close(r) // the descriptor of the transport is closed already
			return tr.Close()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(t)
			if err == nil {
				t.Fatal("no error")
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if appErr.Tag != tc.tag || !appErr.HasRoute(tc.route) {
				t.Errorf("tag %s, route %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestHidrawOfFindsTheNode(t *testing.T) {
	sys := t.TempDir()
	if got := hidrawOf(sys, "event0"); got != "" {
		t.Errorf("a device with no hidraw directory gave %q", got)
	}
	dir := filepath.Join(sys, "event0", "device", "device", "hidraw")
	for _, n := range []string{"notanode", "hidraw12"} {
		if err := os.MkdirAll(filepath.Join(dir, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := hidrawOf(sys, "event0"); got != "hidraw12" {
		t.Errorf("hidrawOf = %q", got)
	}
}
