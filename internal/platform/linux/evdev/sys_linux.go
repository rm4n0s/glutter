//go:build linux && !android

package evdev

import (
	"encoding/binary"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// The tag of the error of a file of sysfs that cannot be read (rm4n0s/errors, R5).
const tagEvdevSysfsReadFailed = "EvdevSysfsReadFailed"

// The ioctl requests of linux/input.h. The number of a request is the direction, the size of the argument, a type letter and a
// number; the layout below is that of every architecture Go runs on except MIPS, PowerPC and SPARC, where the direction bits
// are different (they have no gamepads that Glutter runs on).
const (
	iocWrite = 1
	iocRead  = 2
)

func ioc(dir, typ, nr, size uintptr) uintptr { return dir<<30 | size<<16 | typ<<8 | nr }

func eviocgkey(n int) uintptr { return ioc(iocRead, 'E', 0x18, uintptr(n)) }
func eviocgabs(abs int) uintptr {
	return ioc(iocRead, 'E', 0x40+uintptr(abs), unsafe.Sizeof(absInfo{}))
}
func eviocsclockid() uintptr { return ioc(iocWrite, 'E', 0xa0, unsafe.Sizeof(int32(0))) }

const clockMonotonic = 1

// ioctl calls the ioctl with a pointer to the argument.
func ioctl(fd int, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

// readAbs reads the range and the value of an axis.
func readAbs(fd, code int) (absInfo, error) {
	var info absInfo
	err := ioctl(fd, eviocgabs(code), unsafe.Pointer(&info))
	return info, err
}

// setMonotonicClock makes the kernel stamp the events of the device with the monotonic clock, which is the clock of the other
// events of the framework.
func setMonotonicClock(fd int) error {
	clock := int32(clockMonotonic)
	return ioctl(fd, eviocsclockid(), unsafe.Pointer(&clock))
}

// monotonicNow reads the monotonic clock, the one that the events of a device are stamped with.
func monotonicNow() time.Duration {
	var ts syscall.Timespec
	if _, _, e := syscall.Syscall(syscall.SYS_CLOCK_GETTIME, clockMonotonic, uintptr(unsafe.Pointer(&ts)), 0); e != 0 {
		return 0
	}
	return time.Duration(ts.Sec)*time.Second + time.Duration(ts.Nsec)
}

// eventSize is the size of the kernel's struct input_event: a timeval, then the type and code (16 bits each) and the value.
var eventSize = int(unsafe.Sizeof(syscall.Timeval{})) + 8

// ffEvent makes the input_event that plays an effect (value 1) or stops it (value 0): the type EV_FF, the id of the effect as the
// code, and the time left at zero, which the kernel fills in.
func ffEvent(id int16, value int32) []byte {
	b := make([]byte, eventSize)
	off := eventSize - 8
	binary.NativeEndian.PutUint16(b[off:], evFF)
	binary.NativeEndian.PutUint16(b[off+2:], uint16(id))
	binary.NativeEndian.PutUint32(b[off+4:], uint32(value))
	return b
}

func eviocsff() uintptr { return ioc(iocWrite, 'E', 0x80, unsafe.Sizeof(ffEffect{})) }

// uploadEffect gives the effect to the device; the kernel writes the id it chose into the effect.
func uploadEffect(fd int, e *ffEffect) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), eviocsff(), uintptr(unsafe.Pointer(e))); errno != 0 {
		return errors.NewErr(tagEvdevRumbleFailed, errno).SetMetadata("step", "upload")
	}
	return nil
}

// playEffect starts (value 1) or stops (value 0) an uploaded effect.
func playEffect(fd int, id int16, value int32) error {
	if _, err := syscall.Write(fd, ffEvent(id, value)); err != nil {
		return errors.NewErr(tagEvdevRumbleFailed, err).SetMetadata("step", "play", "value", int(value))
	}
	return nil
}

// openWritable opens the event device for reading and writing, which force feedback needs.
func openWritable(path string) (int, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return -1, errors.NewErr(tagEvdevOpenFailed, err).SetMetadata("device", path)
	}
	return fd, nil
}

// parseEvents reads the events in a buffer that was read from a device. A partial event at the end is dropped.
func parseEvents(buf []byte, out []inputEvent) []inputEvent {
	tv := int(unsafe.Sizeof(syscall.Timeval{}))
	for len(buf) >= eventSize {
		var sec, usec int64
		if tv == 16 {
			sec = int64(binary.NativeEndian.Uint64(buf[0:]))
			usec = int64(binary.NativeEndian.Uint64(buf[8:]))
		} else {
			sec = int64(int32(binary.NativeEndian.Uint32(buf[0:])))
			usec = int64(int32(binary.NativeEndian.Uint32(buf[4:])))
		}
		out = append(out, inputEvent{
			Time:  time.Duration(sec)*time.Second + time.Duration(usec)*time.Microsecond,
			Type:  binary.NativeEndian.Uint16(buf[tv:]),
			Code:  binary.NativeEndian.Uint16(buf[tv+2:]),
			Value: int32(binary.NativeEndian.Uint32(buf[tv+4:])),
		})
		buf = buf[eventSize:]
	}
	return out
}

// dirFiles reads the files of a device's directory of sysfs.
type dirFiles string

func (d dirFiles) read(rel string) (string, error) {
	b, err := os.ReadFile(string(d) + "/" + rel)
	if err != nil {
		return "", errors.NewErr(tagEvdevSysfsReadFailed, err).SetMetadata("file", string(d)+"/"+rel)
	}
	return string(b), nil
}

// driverName finds the name of the kernel driver of the device behind an input device, when sysfs says: the link
// device/device/driver of a USB pad points at the driver of its interface.
func driverName(dir string) string {
	for _, rel := range []string{"/device/device/driver", "/device/driver"} {
		if target, err := os.Readlink(dir + rel); err == nil {
			return target[strings.LastIndexByte(target, '/')+1:]
		}
	}
	return ""
}

// kernel is what the service asks of an open event device besides reading its events. The real one makes the ioctls; a test
// replaces it to drive the service with pipes in place of devices.
type kernel struct {
	// readAbs reads the range and the value of an axis.
	readAbs func(fd, code int) (absInfo, error)
	// keyState reads which keys are down.
	keyState func(fd int) (bits, bool)
	// setClock makes the device stamp its events with the monotonic clock.
	setClock func(fd int) error
	// openWritable opens a device for reading and writing, for a pad that can rumble; nil opens it as any other.
	openWritable func(path string) (int, error)
	// uploadEffect gives a force feedback effect to the device and sets its id; playEffect starts or stops it.
	uploadEffect func(fd int, e *ffEffect) error
	playEffect   func(fd int, id int16, value int32) error
}

// realKernel is the kernel of the machine.
func realKernel() kernel {
	return kernel{
		readAbs: readAbs,
		keyState: func(fd int) (bits, bool) {
			buf := make(bits, (keyMax+8)/8)
			if err := ioctl(fd, eviocgkey(len(buf)), unsafe.Pointer(&buf[0])); err != nil {
				return nil, false
			}
			return buf, true
		},
		setClock:     setMonotonicClock,
		openWritable: openWritable,
		uploadEffect: uploadEffect,
		playEffect:   playEffect,
	}
}

// deviceSource reads the state of an open device.
type deviceSource struct {
	k  kernel
	fd int
}

func (s deviceSource) absValue(code int) (int32, bool) {
	info, err := s.k.readAbs(s.fd, code)
	return info.Value, err == nil
}

func (s deviceSource) keyState() (bits, bool) { return s.k.keyState(s.fd) }
