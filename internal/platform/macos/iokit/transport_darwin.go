//go:build darwin && !ios

package iokit

import (
	"sync"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// reportBufferSize is the size of the buffer that receives an input report: the longest report of the pads that have a HID
// driver is a few hundred bytes (a Switch Pro controller's full state is 362 over USB); 512 leaves room.
const reportBufferSize = 512

// reportCallback is the callback of an input report, made once, on first use. It finds its transport by its context.
var reportCallback = sync.OnceValue(func() uintptr {
	return dl.Callback(func(ctx uintptr, _ int32, _ uintptr, _ int32, id uint32, report uintptr, length int64) {
		if t, ok := lookup(ctx).(*transport); ok {
			t.report(id, report, length)
		}
	})
})

// transport is the link of a HID driver to a device (gamepadhid.Transport): the reports come to the run loop of the main
// thread, which hands them to the goroutine of the driver, and the output and feature reports go out from that goroutine.
type transport struct {
	d   *device
	ctx uintptr
	buf uintptr // the buffer that IOKit writes input reports into, in C memory

	in     chan []byte
	closed chan struct{}
	once   sync.Once
}

var _ gamepadhid.Transport = (*transport)(nil)

// openTransport starts the reports of a device. It runs on the UI thread, where the callbacks of the reports will run.
func openTransport(d *device) (gamepadhid.Transport, error) {
	f := d.m.f
	if d.gone.Load() || d.released {
		return nil, errors.New(tagHIDDeviceClosed, "the device is gone", "product", d.info.ProductName)
	}
	cf.Retain(d.ref)
	t := &transport{d: d, in: make(chan []byte, 64), closed: make(chan struct{})}
	t.buf = dl.Malloc(reportBufferSize)
	t.ctx = register(t)
	f.deviceOnReport(d.ref, t.buf, reportBufferSize, reportCallback(), t.ctx)
	f.deviceSchedule(d.ref, f.mainRunLoop(), f.commonMode())
	return t, nil
}

// report is the callback of an input report: it copies the report, with its ID as the first byte as the drivers expect, and
// queues it. A queue that is full drops the report (the driver is behind, and the next report says more).
func (t *transport) report(id uint32, report uintptr, length int64) {
	if length <= 0 || report == 0 {
		return
	}
	data := dl.Bytes(report, int(length))
	out := make([]byte, 0, len(data)+1)
	if id != 0 && data[0] != byte(id) {
		out = append(out, byte(id))
	}
	out = append(out, data...)
	select {
	case t.in <- out:
	default:
	}
}

// Read implements gamepadhid.Transport.
func (t *transport) Read(buf []byte, timeout time.Duration) (int, error) {
	select {
	case r := <-t.in:
		return copy(buf, r), nil
	case <-t.closed:
		return 0, errors.New(tagHIDDeviceClosed, "the transport was closed", "product", t.d.info.ProductName)
	default:
	}
	if t.d.gone.Load() {
		return 0, errors.New(tagHIDDeviceClosed, "the device is gone", "product", t.d.info.ProductName)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case r := <-t.in:
		return copy(buf, r), nil
	case <-t.closed:
		return 0, errors.New(tagHIDDeviceClosed, "the transport was closed", "product", t.d.info.ProductName)
	case <-timer.C:
		return 0, nil
	}
}

// Write implements gamepadhid.Transport: an output report. A report whose ID is 0 is sent without it.
func (t *transport) Write(data []byte) (int, error) {
	if err := t.set(reportTypeOutput, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// SendFeature implements gamepadhid.Transport.
func (t *transport) SendFeature(data []byte) (int, error) {
	if err := t.set(reportTypeFeature, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// set sends a report of a type. IOKit takes the ID apart from the data, and a report that has the ID 0 has none on the wire.
func (t *transport) set(typ uint32, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	id, rest := int64(data[0]), data
	if id == 0 {
		rest = data[1:]
	}
	if len(rest) == 0 {
		return nil
	}
	if t.isClosed() {
		return errors.New(tagHIDDeviceClosed, "the transport was closed", "product", t.d.info.ProductName)
	}
	if code := t.d.m.f.deviceSetReport(t.d.ref, typ, id, unsafe.Pointer(&rest[0]), int64(len(rest))); code != 0 {
		return errors.New(tagHIDReportFailed, "IOHIDDeviceSetReport failed", "code", int64(code), "type", int64(typ), "report", id)
	}
	return nil
}

// GetFeature implements gamepadhid.Transport.
func (t *transport) GetFeature(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, errors.New(tagHIDReportFailed, "no room for a feature report")
	}
	if t.isClosed() {
		return 0, errors.New(tagHIDDeviceClosed, "the transport was closed", "product", t.d.info.ProductName)
	}
	id, buf := int64(data[0]), data
	if id == 0 {
		buf = data[1:]
	}
	if len(buf) == 0 {
		return 0, errors.New(tagHIDReportFailed, "no room for a feature report")
	}
	n := int64(len(buf))
	if code := t.d.m.f.deviceGetReport(t.d.ref, reportTypeFeature, id, unsafe.Pointer(&buf[0]), &n); code != 0 {
		return 0, errors.New(tagHIDReportFailed, "IOHIDDeviceGetReport failed", "code", int64(code), "report", id)
	}
	if id == 0 {
		n++
	}
	return int(n), nil
}

func (t *transport) isClosed() bool {
	select {
	case <-t.closed:
		return true
	default:
		return false
	}
}

// Close implements gamepadhid.Transport. The callback of the reports is taken away on the UI thread, where it was added, and
// the buffer and the device are given back there.
func (t *transport) Close() error {
	t.once.Do(func() {
		close(t.closed)
		t.d.m.post(func() {
			f := t.d.m.f
			f.deviceOnReport(t.d.ref, t.buf, reportBufferSize, 0, 0)
			f.deviceUnschedule(t.d.ref, f.mainRunLoop(), f.commonMode())
			forget(t.ctx)
			dl.Free(t.buf)
			cf.Release(t.d.ref)
		})
	})
	return nil
}

// The tags of the errors of this file.
const (
	tagHIDDeviceClosed = "HIDDeviceClosed"
	tagHIDReportFailed = "HIDReportFailed"
)
