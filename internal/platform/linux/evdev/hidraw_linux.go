//go:build linux && !android

package evdev

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
)

// The transport of the HID drivers (internal/gamepadhid) on Linux: a /dev/hidrawN node, which gives the reports of a HID device as
// the device sends them and takes the reports that are written to it, and the feature reports through ioctl (linux/hidraw.h).
// The node of a pad is found in sysfs, below the HID device that the input device belongs to.

// The ioctl requests of linux/hidraw.h: the feature reports have the length of the report in the request.
func hidiocsfeature(n int) uintptr { return ioc(iocRead|iocWrite, 'H', 0x06, uintptr(n)) }
func hidiocgfeature(n int) uintptr { return ioc(iocRead|iocWrite, 'H', 0x07, uintptr(n)) }

// hidrawNode matches the name of a hidraw node.
var hidrawNode = regexp.MustCompile(`^hidraw[0-9]+$`)

// hidrawOf finds the hidraw node of an event device: sysfs has it below the HID device that the input device is a child of
// (/sys/class/input/eventN/device/device/hidraw/hidrawM). It returns the name of the node, or "" when the device is not on HID
// (a pad of the kernel's own drivers that is not HID, a virtual one).
func hidrawOf(sysDir, node string) string {
	entries, err := os.ReadDir(filepath.Join(sysDir, node, "device", "device", "hidraw"))
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if hidrawNode.MatchString(e.Name()) {
			return e.Name()
		}
	}
	return ""
}

// hidrawKernel is what the transport asks of the kernel, for the tests to replace.
type hidrawKernel struct {
	ioctl func(fd int, req uintptr, arg unsafe.Pointer) (int, error)
}

func realHidrawKernel() hidrawKernel {
	return hidrawKernel{ioctl: func(fd int, req uintptr, arg unsafe.Pointer) (int, error) {
		n, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg))
		if e != 0 {
			return 0, e
		}
		return int(n), nil
	}}
}

// hidrawTransport is a gamepadhid.Transport over a hidraw node. The descriptor is guarded by a lock: a call holds it for the
// system call, and Close takes it to close the descriptor, so a number that was closed is never used.
type hidrawTransport struct {
	k    hidrawKernel
	path string

	mu   sync.RWMutex
	fd   int
	epfd int
}

var _ gamepadhid.Transport = (*hidrawTransport)(nil)

// openHidraw opens the node for reading and writing and puts its descriptor in an epoll instance of its own, so that a read can
// wait for a report for a time.
func openHidraw(path string, k hidrawKernel) (*hidrawTransport, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		tag := tagHidrawOpenFailed
		if err == syscall.EACCES || err == syscall.EPERM {
			tag = tagHidrawPermission
		}
		return nil, errors.NewErr(tag, err).SetMetadata("device", path)
	}
	epfd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		_ = syscall.Close(fd) // the open failed
		return nil, errors.NewErr(tagHidrawOpenFailed, err).SetMetadata("device", path, "step", "epoll_create1")
	}
	ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fd)}
	if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, fd, &ev); err != nil {
		_ = syscall.Close(epfd) // the open failed
		_ = syscall.Close(fd)   // the same
		return nil, errors.NewErr(tagHidrawOpenFailed, err).SetMetadata("device", path, "step", "epoll_ctl")
	}
	return &hidrawTransport{k: k, path: path, fd: fd, epfd: epfd}, nil
}

// Read implements gamepadhid.Transport.
func (t *hidrawTransport) Read(buf []byte, timeout time.Duration) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.fd < 0 {
		return 0, errors.New(tagHidrawClosed, "the hidraw node is closed", "device", t.path)
	}
	ms := int(timeout / time.Millisecond)
	if timeout > 0 && ms == 0 {
		ms = 1
	}
	var events [1]syscall.EpollEvent
	for {
		n, err := syscall.EpollWait(t.epfd, events[:], ms)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return 0, errors.NewErr(tagHidrawReadFailed, err).SetMetadata("device", t.path, "step", "epoll_wait")
		}
		if n == 0 {
			return 0, nil
		}
		break
	}
	for {
		n, err := syscall.Read(t.fd, buf)
		switch {
		case err == syscall.EINTR:
			continue
		case err == syscall.EAGAIN:
			return 0, nil
		case err != nil:
			// ENODEV: the device is gone.
			return 0, errors.NewErr(tagHidrawReadFailed, err).SetMetadata("device", t.path)
		case n == 0 && events[0].Events&(syscall.EPOLLHUP|syscall.EPOLLERR) != 0:
			return 0, errors.New(tagHidrawReadFailed, "the hidraw node hung up", "device", t.path)
		}
		return n, nil
	}
}

// Write implements gamepadhid.Transport.
func (t *hidrawTransport) Write(data []byte) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.fd < 0 {
		return 0, errors.New(tagHidrawClosed, "the hidraw node is closed", "device", t.path)
	}
	for {
		n, err := syscall.Write(t.fd, data)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return 0, errors.NewErr(tagHidrawWriteFailed, err).SetMetadata("device", t.path, "length", len(data))
		}
		return n, nil
	}
}

// SendFeature implements gamepadhid.Transport.
func (t *hidrawTransport) SendFeature(data []byte) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.fd < 0 {
		return 0, errors.New(tagHidrawClosed, "the hidraw node is closed", "device", t.path)
	}
	if len(data) == 0 {
		return 0, errors.New(tagHidrawFeatureFailed, "a feature report has at least its ID", "device", t.path)
	}
	n, err := t.k.ioctl(t.fd, hidiocsfeature(len(data)), unsafe.Pointer(&data[0]))
	if err != nil {
		return 0, errors.NewErr(tagHidrawFeatureFailed, err).SetMetadata("device", t.path, "request", "HIDIOCSFEATURE", "report", int(data[0]))
	}
	return n, nil
}

// GetFeature implements gamepadhid.Transport.
func (t *hidrawTransport) GetFeature(data []byte) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.fd < 0 {
		return 0, errors.New(tagHidrawClosed, "the hidraw node is closed", "device", t.path)
	}
	if len(data) == 0 {
		return 0, errors.New(tagHidrawFeatureFailed, "a feature report has at least its ID", "device", t.path)
	}
	n, err := t.k.ioctl(t.fd, hidiocgfeature(len(data)), unsafe.Pointer(&data[0]))
	if err != nil {
		return 0, errors.NewErr(tagHidrawFeatureFailed, err).SetMetadata("device", t.path, "request", "HIDIOCGFEATURE", "report", int(data[0]))
	}
	return n, nil
}

// Close implements gamepadhid.Transport. It waits for the calls that are under way.
func (t *hidrawTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.fd < 0 {
		return nil
	}
	err := syscall.Close(t.fd)
	if e := syscall.Close(t.epfd); err == nil {
		err = e
	}
	t.fd, t.epfd = -1, -1
	if err != nil {
		return errors.NewErr(tagHidrawCloseFailed, err).SetMetadata("device", t.path)
	}
	return nil
}

// The tags of the errors that the transport makes (rm4n0s/errors, R5).
const (
	tagHidrawOpenFailed    = "HidrawOpenFailed"
	tagHidrawPermission    = "HidrawPermission"
	tagHidrawClosed        = "HidrawClosed"
	tagHidrawReadFailed    = "HidrawReadFailed"
	tagHidrawWriteFailed   = "HidrawWriteFailed"
	tagHidrawFeatureFailed = "HidrawFeatureFailed"
	tagHidrawCloseFailed   = "HidrawCloseFailed"
)
