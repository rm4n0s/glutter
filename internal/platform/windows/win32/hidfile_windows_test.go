//go:build windows

package win32

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// A HID device cannot be had under Wine on a headless host, so the file layer is tried on a named pipe: its client end is a
// handle with FILE_FLAG_OVERLAPPED like the one of a HID device, and the server end plays the device. The calls of hid.dll
// (the feature reports and the lengths) are tried on handles that are not devices, which must fail with their tags.

var (
	kernelProcs = syscall.NewLazyDLL("kernel32.dll")
	createPipe  = kernelProcs.NewProc("CreateNamedPipeW")
	connectPipe = kernelProcs.NewProc("ConnectNamedPipe")
	pipeRead    = kernelProcs.NewProc("ReadFile")
	pipeWrite   = kernelProcs.NewProc("WriteFile")
	pipeClose   = kernelProcs.NewProc("CloseHandle")
)

// pipePair makes a named pipe and returns the client end as a HidFile with the given report lengths, and the server end, which
// has no overlapped flag so that the test reads and writes it without waiting.
type pipePair struct {
	file   *HidFile
	server uintptr
}

func newPipePair(t *testing.T, in, out, feature int) *pipePair {
	t.Helper()
	name := fmt.Sprintf(`\\.\pipe\glutter-hidfile-%d-%d`, os.Getpid(), time.Now().UnixNano())
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	const pipeAccessDuplex, pipeTypeByte, pipeWait = 3, 0, 0
	server, _, errno := createPipe.Call(uintptr(unsafe.Pointer(p)), pipeAccessDuplex, pipeTypeByte|pipeWait, 1, 4096, 4096, 0, 0)
	if server == ^uintptr(0) {
		t.Fatalf("CreateNamedPipe: %v", errno)
	}
	h, err := CreateFileFlags(name, GenericRead|GenericWrite, 0, OpenExisting, FileFlagOverlapped)
	if err != nil {
		t.Fatal(err)
	}
	// The client has connected, which a server that has no waiting call sees as connected already.
	_, _, _ = connectPipe.Call(server, 0)
	f, err := newHidFile(h, name, in, out, feature)
	if err != nil {
		t.Fatal(err)
	}
	pp := &pipePair{file: f, server: server}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
		pp.closeServer()
	})
	return pp
}

func (p *pipePair) closeServer() {
	if p.server != 0 {
		_, _, _ = pipeClose.Call(p.server)
		p.server = 0
	}
}

func (p *pipePair) serverWrite(t *testing.T, data []byte) {
	t.Helper()
	var n uint32
	r, _, errno := pipeWrite.Call(p.server, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(unsafe.Pointer(&n)), 0)
	if r == 0 || int(n) != len(data) {
		t.Fatalf("the server wrote %d of %d bytes: %v", n, len(data), errno)
	}
}

func (p *pipePair) serverRead(t *testing.T, size int) []byte {
	t.Helper()
	buf := make([]byte, size)
	var n uint32
	r, _, errno := pipeRead.Call(p.server, uintptr(unsafe.Pointer(&buf[0])), uintptr(size), uintptr(unsafe.Pointer(&n)), 0)
	if r == 0 {
		t.Fatalf("the server read failed: %v", errno)
	}
	return buf[:n]
}

func TestHidFileReadWaitsForItsTime(t *testing.T) {
	p := newPipePair(t, 64, 8, 0)
	buf := make([]byte, 64)
	start := time.Now()
	if n, err := p.file.Read(buf, 30*time.Millisecond); n != 0 || err != nil {
		t.Fatalf("a read with no report gave %d, %v", n, err)
	}
	if time.Since(start) < 25*time.Millisecond {
		t.Error("the read did not wait for its time")
	}
	// The read that timed out is still under way: what the device sends now reaches the next call.
	want := []byte{0x01, 0x80, 0x80, 0x00, 0x07}
	p.serverWrite(t, want)
	n, err := p.file.Read(buf, time.Second)
	if err != nil || !bytes.Equal(buf[:n], want) {
		t.Errorf("read = %x, %v", buf[:n], err)
	}
	// A second report goes through the same way, and a zero timeout only looks.
	p.serverWrite(t, []byte{0x02, 0x09})
	time.Sleep(20 * time.Millisecond)
	if n, err = p.file.Read(buf, 0); err != nil || !bytes.Equal(buf[:n], []byte{0x02, 0x09}) {
		t.Errorf("second read = %x, %v", buf[:n], err)
	}
	if n, err = p.file.Read(buf, 0); n != 0 || err != nil {
		t.Errorf("a read with nothing there gave %d, %v", n, err)
	}
}

func TestHidFileWritePadsToTheOutputReport(t *testing.T) {
	p := newPipePair(t, 64, 8, 0)
	n, err := p.file.Write([]byte{0x05, 1, 2})
	if err != nil || n != 3 {
		t.Fatalf("write = %d, %v", n, err)
	}
	if got := p.serverRead(t, 64); !bytes.Equal(got, []byte{0x05, 1, 2, 0, 0, 0, 0, 0}) {
		t.Errorf("the device got %x", got)
	}
	// A report as long as the output report goes as it is, and a longer one as well.
	long := []byte{0x11, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if _, err := p.file.Write(long); err != nil {
		t.Fatal(err)
	}
	if got := p.serverRead(t, 64); !bytes.Equal(got, long) {
		t.Errorf("the device got %x", got)
	}
}

func TestHidFileLengthAndLimits(t *testing.T) {
	p := newPipePair(t, 78, 78, 41)
	if in, out, feat := p.file.Lengths(); in != 78 || out != 78 || feat != 41 {
		t.Errorf("lengths = %d %d %d", in, out, feat)
	}
	// The buffers are as long as the reports that the device has: a longer report is refused, not cut.
	_, err := p.file.Write(make([]byte, 200))
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagHidFileWriteFailed || !appErr.HasRoute("HidFile.Write.HidFileWriteFailed") {
		t.Errorf("err = %v", err)
	}
}

func TestHidFileCloseLetsAWaitingReadGo(t *testing.T) {
	p := newPipePair(t, 64, 64, 0)
	// The first read starts the call and times out; the second waits while the file is closed.
	buf := make([]byte, 64)
	if n, err := p.file.Read(buf, time.Millisecond); n != 0 || err != nil {
		t.Fatalf("read = %d, %v", n, err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := p.file.Read(buf, 10*time.Second)
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	if err := p.file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case err := <-done:
		t.Logf("the read ended with: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("a read was still waiting after Close")
	}
	if err := p.file.Close(); err != nil {
		t.Errorf("the second Close failed: %v", err)
	}
}

func TestHidFileErrorsHaveTheirRoutes(t *testing.T) {
	tests := []struct {
		name  string
		tag   string
		route string
		run   func(t *testing.T) error
	}{
		{"read after the device went", TagHidFileReadFailed, "HidFile.Read.HidFileReadFailed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 0)
			p.closeServer()
			buf := make([]byte, 64)
			var err error
			for i := 0; i < 20 && err == nil; i++ {
				_, err = p.file.Read(buf, 50*time.Millisecond)
			}
			return err
		}},
		{"read after Close", TagHidFileClosed, "HidFile.Read.HidFileClosed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 0)
			if err := p.file.Close(); err != nil {
				t.Fatal(err)
			}
			_, err := p.file.Read(make([]byte, 8), time.Millisecond)
			return err
		}},
		{"write after Close", TagHidFileClosed, "HidFile.Write.HidFileClosed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 0)
			if err := p.file.Close(); err != nil {
				t.Fatal(err)
			}
			_, err := p.file.Write([]byte{1})
			return err
		}},
		{"write after the device went", TagHidFileWriteFailed, "HidFile.Write.HidFileWriteFailed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 0)
			p.closeServer()
			_, err := p.file.Write([]byte{1, 2})
			return err
		}},
		{"feature report sent to a handle that is not a device", TagHidFileFeatureFailed, "HidFile.SendFeature.HidFileFeatureFailed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 8)
			_, err := p.file.SendFeature([]byte{2, 0, 0})
			return err
		}},
		{"feature report asked of a handle that is not a device", TagHidFileFeatureFailed, "HidFile.GetFeature.HidFileFeatureFailed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 8)
			_, err := p.file.GetFeature([]byte{2, 0, 0})
			return err
		}},
		{"empty feature report sent", TagHidFileFeatureFailed, "HidFile.SendFeature.HidFileFeatureFailed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 8)
			_, err := p.file.SendFeature(nil)
			return err
		}},
		{"empty feature report asked", TagHidFileFeatureFailed, "HidFile.GetFeature.HidFileFeatureFailed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 8)
			_, err := p.file.GetFeature(nil)
			return err
		}},
		{"feature report after Close", TagHidFileClosed, "HidFile.SendFeature.HidFileClosed", func(t *testing.T) error {
			p := newPipePair(t, 64, 64, 8)
			if err := p.file.Close(); err != nil {
				t.Fatal(err)
			}
			_, err := p.file.SendFeature([]byte{2, 0})
			return err
		}},
		{"open of a file that is not a device", TagHidFileFailed, "OpenHidFile->hidCapsOf.HidFileFailed", func(t *testing.T) error {
			path := filepath.Join(t.TempDir(), "plain.txt")
			if err := os.WriteFile(path, []byte("not a device"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := OpenHidFile(path)
			return err
		}},
		{"open of a path that is not there", TagWin32CallFailed, "OpenHidFile->CreateFileFlags->Win32Error.Win32CallFailed", func(t *testing.T) error {
			_, err := OpenHidFile(filepath.Join(t.TempDir(), "nothing"))
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
}
