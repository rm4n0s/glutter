//go:build windows

package win32

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// HidFile is a HID device opened for the HID drivers of the gamepads (step 31.3, SPEC §28.2): reads and writes that do not wait
// (overlapped I/O) with a time for the read, and the feature reports of hid.dll. It is the transport of Windows under
// internal/gamepadhid, which package windows makes of it.
//
// The OVERLAPPED structures and the buffers that Windows keeps a pointer to while a call is under way live in memory from
// HeapAlloc (SPEC §6.3 rule 11). A read that is waiting when the time runs out stays under way, and the next Read waits for it
// (the way hidapi does it), so no report is lost between two calls.
//
// Flutter: no counterpart
type HidFile struct {
	path                   string
	inLen, outLen, featLen int
	// bufLen is the length of each of the two buffers.
	bufLen int

	// guard keeps the handle: a call holds it for read while it uses the handle, and Close takes it to close the handle.
	guard   sync.RWMutex
	handle  uintptr
	closing atomic.Bool
	once    sync.Once
	err     error

	heap  uintptr
	block uintptr // the block of C memory: two OVERLAPPED, then the buffers
	// rd and wr are the OVERLAPPED of the read and of the write, rbuf and wbuf their buffers.
	rd, wr     uintptr
	rbuf, wbuf uintptr
	revent     uintptr
	wevent     uintptr

	readMu      sync.Mutex
	readPending bool
	writeMu     sync.Mutex
}

// hidWriteTimeout is how long a write may take before it is cancelled (hidapi waits a second).
const hidWriteTimeout = time.Second

// OpenHidFile opens the HID device at path (a path of Raw Input or of the device interface) for reading and writing, and asks it
// how long its reports are. It fails with the tag of CreateFileW when the device cannot be opened, with HidFileFailed when
// hid.dll does not give the lengths.
//
// Flutter: no counterpart
func OpenHidFile(path string) (*HidFile, error) {
	h, err := CreateFileFlags(path, GenericRead|GenericWrite, FileShareRead|FileShareWrite, OpenExisting, FileFlagOverlapped)
	if err != nil {
		return nil, err
	}
	caps, err := hidCapsOf(h)
	if err != nil {
		return nil, closeAfterFailure(err, h)
	}
	f, err := newHidFile(h, path, int(caps.InputReportByteLength), int(caps.OutputReportByteLength), int(caps.FeatureReportByteLength))
	if err != nil {
		return nil, closeAfterFailure(err, h)
	}
	return f, nil
}

// closeAfterFailure closes a handle of an open that failed with err and returns err; a close that fails too is noted in the
// metadata of err.
func closeAfterFailure(err error, h uintptr) error {
	if cerr := CloseHandle(h); cerr != nil {
		if appErr, ok := errors.FromError(err); ok {
			_ = appErr.SetMetadata("close", cerr.Error()) // it changes the error that is returned
		}
	}
	return err
}

// hidCapsOf asks hid.dll for the lengths of the reports of an open device (HidD_GetPreparsedData, HidP_GetCaps).
func hidCapsOf(h uintptr) (HidpCaps, error) {
	var caps HidpCaps
	var pre uintptr
	if !boolCall(hidDLL, "HidD_GetPreparsedData", h, uintptr(unsafe.Pointer(&pre))) || pre == 0 {
		return caps, errors.New(TagHidFileFailed, "the report descriptor of the device cannot be read", "call", "HidD_GetPreparsedData")
	}
	defer func() { _ = boolCall(hidDLL, "HidD_FreePreparsedData", pre) }() // a failed free leaves nothing to do
	r, _, err := hidDLL.call("HidP_GetCaps", pre, uintptr(unsafe.Pointer(&caps)))
	if err != nil {
		return caps, err
	}
	if uint32(r) != HidPStatusSuccess {
		return caps, errors.New(TagHidParserFailed, "HidP_GetCaps failed", "call", "HidP_GetCaps", "status", uint32(r))
	}
	return caps, nil
}

// newHidFile makes the file of a handle that was opened with FILE_FLAG_OVERLAPPED. Tests make one of a named pipe. The report
// lengths include the report ID.
func newHidFile(h uintptr, path string, inLen, outLen, featLen int) (*HidFile, error) {
	// The buffers are as long as the longest report that the device has, at least 64 bytes.
	size := max(inLen, outLen, 64)
	heap, err := ProcessHeap()
	if err != nil {
		return nil, err
	}
	ovSize := unsafe.Sizeof(Overlapped{})
	block, err := HeapAlloc(heap, HeapZeroMemory, 2*ovSize+2*uintptr(size))
	if err != nil {
		return nil, err
	}
	revent, err := CreateEvent(true, false)
	if err != nil {
		_ = HeapFree(heap, 0, block) //nolint:errcheck // the open fails with the first error
		return nil, err
	}
	wevent, err := CreateEvent(true, false)
	if err != nil {
		_ = CloseHandle(revent)      //nolint:errcheck // the open fails with the first error
		_ = HeapFree(heap, 0, block) //nolint:errcheck // the same
		return nil, err
	}
	f := &HidFile{path: path, inLen: inLen, outLen: outLen, featLen: featLen, bufLen: size, handle: h, heap: heap, block: block, revent: revent, wevent: wevent}
	f.rd, f.wr = block, block+ovSize
	f.rbuf = block + 2*ovSize
	f.wbuf = f.rbuf + uintptr(size)
	View[Overlapped](f.rd).Event = revent
	View[Overlapped](f.wr).Event = wevent
	return f, nil
}

// Path is the path the file was opened by.
//
// Flutter: no counterpart
func (f *HidFile) Path() string { return f.path }

// Lengths are the lengths of the input, output and feature reports that the device has, report ID included (0 when the device
// has none of that kind).
//
// Flutter: no counterpart
func (f *HidFile) Lengths() (input, output, feature int) { return f.inLen, f.outLen, f.featLen }

// Read waits up to timeout for an input report and copies it into buf; it returns 0 and no error when the time ran out. The
// report starts with its report ID. It fails with HidFileReadFailed when the device is gone, and HidFileClosed after Close.
//
// Flutter: no counterpart
func (f *HidFile) Read(buf []byte, timeout time.Duration) (int, error) {
	f.guard.RLock()
	defer f.guard.RUnlock()
	if f.handle == 0 {
		return 0, errors.New(TagHidFileClosed, "the HID device is closed", "path", f.path)
	}
	f.readMu.Lock()
	defer f.readMu.Unlock()
	ov := View[Overlapped](f.rd)
	if !f.readPending {
		if err := ResetEvent(f.revent); err != nil {
			return 0, err
		}
		ov.Internal, ov.InternalHigh, ov.Offset, ov.OffsetHigh = 0, 0, 0, 0
		length := f.inLen
		if length <= 0 {
			length = 64
		}
		r, errno, err := kern32.call("ReadFile", f.handle, f.rbuf, uintptr(length), 0, f.rd)
		if err != nil {
			return 0, err
		}
		if r == 0 && errno != ErrorIoPending {
			return 0, errors.New(TagHidFileReadFailed, "ReadFile failed", "path", f.path, "code", int(errno))
		}
		// Whether the read finished at once or not, its event says when it is done.
		f.readPending = true
	}
	res, err := WaitForSingleObject(f.revent, timeout)
	if err != nil {
		return 0, err
	}
	if res == WaitTimeout {
		return 0, nil
	}
	f.readPending = false
	var n uint32
	r, errno, err := kern32.call("GetOverlappedResult", f.handle, f.rd, uintptr(unsafe.Pointer(&n)), 0)
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, errors.New(TagHidFileReadFailed, "the read did not finish", "path", f.path, "code", int(errno))
	}
	return copy(buf, Bytes(f.rbuf, int(n))), nil
}

// Write sends an output report (its first byte is the report ID) and waits until the device has it. A report shorter than the
// output report of the device is padded with zeros, as the system wants. It fails with HidFileWriteFailed, also when the device
// does not take the report in a second, and with HidFileClosed after Close.
//
// Flutter: no counterpart
func (f *HidFile) Write(data []byte) (int, error) {
	f.guard.RLock()
	defer f.guard.RUnlock()
	if f.handle == 0 {
		return 0, errors.New(TagHidFileClosed, "the HID device is closed", "path", f.path)
	}
	f.writeMu.Lock()
	defer f.writeMu.Unlock()
	length := max(len(data), f.outLen)
	if length > f.bufLen {
		return 0, errors.New(TagHidFileWriteFailed, "the report is longer than the buffers of the device", "path", f.path, "length", len(data))
	}
	b := Bytes(f.wbuf, length)
	clear(b)
	copy(b, data)
	if err := ResetEvent(f.wevent); err != nil {
		return 0, err
	}
	ov := View[Overlapped](f.wr)
	ov.Internal, ov.InternalHigh, ov.Offset, ov.OffsetHigh = 0, 0, 0, 0
	r, errno, err := kern32.call("WriteFile", f.handle, f.wbuf, uintptr(length), 0, f.wr)
	if err != nil {
		return 0, err
	}
	if r == 0 && errno != ErrorIoPending {
		return 0, errors.New(TagHidFileWriteFailed, "WriteFile failed", "path", f.path, "code", int(errno), "length", len(data))
	}
	res, err := WaitForSingleObject(f.wevent, hidWriteTimeout)
	if err != nil {
		return 0, err
	}
	if res == WaitTimeout {
		// The call is cancelled and waited for, so that Windows lets go of the buffer.
		_, _, _ = kern32.call("CancelIoEx", f.handle, f.wr)
		_, _, _ = kern32.call("GetOverlappedResult", f.handle, f.wr, uintptr(unsafe.Pointer(new(uint32))), 1)
		return 0, errors.New(TagHidFileWriteFailed, "the device did not take the report in time", "path", f.path, "length", len(data))
	}
	var n uint32
	r, errno, err = kern32.call("GetOverlappedResult", f.handle, f.wr, uintptr(unsafe.Pointer(&n)), 0)
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, errors.New(TagHidFileWriteFailed, "the write did not finish", "path", f.path, "code", int(errno), "length", len(data))
	}
	return min(int(n), len(data)), nil
}

// SendFeature sends a feature report (data[0] is its ID), padded with zeros to the length that the device has (HidD_SetFeature).
// It fails with HidFileFeatureFailed.
//
// Flutter: no counterpart
func (f *HidFile) SendFeature(data []byte) (int, error) {
	f.guard.RLock()
	defer f.guard.RUnlock()
	if f.handle == 0 {
		return 0, errors.New(TagHidFileClosed, "the HID device is closed", "path", f.path)
	}
	if len(data) == 0 {
		return 0, errors.New(TagHidFileFeatureFailed, "a feature report has at least its ID", "path", f.path)
	}
	buf := make([]byte, max(len(data), f.featLen))
	copy(buf, data)
	r, errno, err := hidDLL.call("HidD_SetFeature", f.handle, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, errors.New(TagHidFileFeatureFailed, "HidD_SetFeature failed", "path", f.path, "report", int(data[0]), "code", int(errno))
	}
	return len(data), nil
}

// GetFeature asks the device for the feature report whose ID is data[0] and puts it in data with its ID first (HidD_GetFeature).
// It returns the number of bytes, at most the length of the report. It fails with HidFileFeatureFailed.
//
// Flutter: no counterpart
func (f *HidFile) GetFeature(data []byte) (int, error) {
	f.guard.RLock()
	defer f.guard.RUnlock()
	if f.handle == 0 {
		return 0, errors.New(TagHidFileClosed, "the HID device is closed", "path", f.path)
	}
	if len(data) == 0 {
		return 0, errors.New(TagHidFileFeatureFailed, "a feature report has at least its ID", "path", f.path)
	}
	buf := make([]byte, max(len(data), f.featLen))
	buf[0] = data[0]
	r, errno, err := hidDLL.call("HidD_GetFeature", f.handle, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if err != nil {
		return 0, err
	}
	if r == 0 {
		return 0, errors.New(TagHidFileFeatureFailed, "HidD_GetFeature failed", "path", f.path, "report", int(data[0]), "code", int(errno))
	}
	n := copy(data, buf)
	if f.featLen > 0 {
		n = min(n, f.featLen)
	}
	return n, nil
}

// Close cancels what is under way, waits for it and closes the device. It may be called more than once and from any goroutine.
//
// Flutter: no counterpart
func (f *HidFile) Close() error {
	f.once.Do(func() { f.err = f.close() })
	return f.err
}

func (f *HidFile) close() error {
	f.closing.Store(true)
	// A read that waits is let go at once: the call it waits for ends as cancelled.
	_, _, _ = kern32.call("CancelIoEx", f.handle, 0)
	f.guard.Lock()
	defer f.guard.Unlock()
	pending := f.readPending
	handle := f.handle
	f.handle = 0
	var first error
	// Windows keeps the buffer of a read until the call has ended: it is waited for before the memory goes, and the memory is
	// kept when the call does not end.
	keep := false
	if pending {
		if res, err := WaitForSingleObject(f.revent, time.Second); err != nil || res == WaitTimeout {
			keep = true
			first = errors.New(TagHidFileCloseFailed, "the read under way did not end, so its memory is kept", "path", f.path)
		}
	}
	if err := CloseHandle(handle); err != nil && first == nil {
		first = err
	}
	if err := CloseHandle(f.revent); err != nil && first == nil {
		first = err
	}
	if err := CloseHandle(f.wevent); err != nil && first == nil {
		first = err
	}
	if !keep {
		if err := HeapFree(f.heap, 0, f.block); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// The tags of the errors of the HID files (rm4n0s/errors, R5); the metadata are "path" and, for a failed call, "code".
const (
	// TagHidFileFailed is the tag of the error of a HID device that cannot be set up for the drivers.
	TagHidFileFailed = "HidFileFailed"
	// TagHidFileCloseFailed is the tag of the error of a close that left a read under way, whose memory is therefore kept.
	TagHidFileCloseFailed = "HidFileCloseFailed"
	// TagHidFileClosed is the tag of the error of a call on a HID device that was closed.
	TagHidFileClosed = "HidFileClosed"
	// TagHidFileReadFailed is the tag of the error of a read that failed: the device is gone.
	TagHidFileReadFailed = "HidFileReadFailed"
	// TagHidFileWriteFailed is the tag of the error of a write that failed or did not finish in time.
	TagHidFileWriteFailed = "HidFileWriteFailed"
	// TagHidFileFeatureFailed is the tag of the error of a feature report that could not be sent or read.
	TagHidFileFeatureFailed = "HidFileFeatureFailed"
)
