//go:build darwin && !ios

package iokit

import (
	"sync/atomic"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
)

// device is an IOHIDDevice that the manager found. The manager retains it, so that a device that is unplugged cannot be used
// after it is freed (SDL's fix of bug 4961); Release lets it go.
type device struct {
	m    *manager
	ref  cf.Ref
	info DeviceInfo
	// elems are the elements of the device by cookie; they belong to the device and live as long as it is retained.
	elems map[uint32]cf.Ref
	raw   []RawElement
	// gone is set on the UI thread when the manager says the device is unplugged, and read by the goroutine of a HID driver.
	gone     atomic.Bool
	released bool
}

// newDevice describes the device and lists its elements.
func newDevice(m *manager, ref cf.Ref) *device {
	cf.Retain(ref)
	d := &device{m: m, ref: ref, elems: map[uint32]cf.Ref{}}
	f := m.f
	num := func(key cf.Ref) uint32 {
		v := f.deviceProperty(ref, key)
		if v == 0 {
			return 0
		}
		var out int32
		if !f.numberValue(v, kCFNumberSInt32Type, unsafe.Pointer(&out)) {
			return 0
		}
		return uint32(out)
	}
	str := func(key cf.Ref) string {
		if v := f.deviceProperty(ref, key); v != 0 {
			return cf.String(v)
		}
		return ""
	}
	d.info = DeviceInfo{
		Vendor: uint16(num(f.propVendor)), Product: uint16(num(f.propProduct)), Version: uint16(num(f.propVer)),
		VendorName: str(f.propMaker), ProductName: str(f.propName), Transport: str(f.propTransport),
		UsagePage: num(f.propUsagePage), Usage: num(f.propUsage),
	}
	if loc := num(f.propLoc); loc != 0 {
		d.info.Path = "0x" + hex8(loc)
	}
	if arr := f.deviceElements(ref, 0, 0); arr != 0 {
		d.addElements(arr)
		cf.Release(arr)
	}
	return d
}

// hex8 writes a number in eight hexadecimal digits.
func hex8(v uint32) string {
	const digits = "0123456789abcdef"
	var b [8]byte
	for i := 7; i >= 0; i-- {
		b[i] = digits[v&0xF]
		v >>= 4
	}
	return string(b[:])
}

// addElements lists the elements of an array, and those of the collections in it.
func (d *device) addElements(arr cf.Ref) {
	f := d.m.f
	for i := int64(0); i < f.arrayCount(arr); i++ {
		e := f.arrayAt(arr, i)
		if e == 0 {
			continue
		}
		t := ElementType(f.elementType(e))
		if t == ElementCollection {
			if children := f.elementChildren(e); children != 0 {
				d.addElements(children)
			}
			continue
		}
		c := f.elementCookie(e)
		d.elems[c] = e
		d.raw = append(d.raw, RawElement{
			Cookie: c, Type: t, Page: f.elementUsagePage(e), Usage: f.elementUsage(e),
			Min: int32(f.elementLogicalMin(e)), Max: int32(f.elementLogicalMax(e)),
		})
	}
}

// Info implements Device.
func (d *device) Info() DeviceInfo { return d.info }

// Elements implements Device.
func (d *device) Elements() []RawElement { return d.raw }

// Value implements Device.
func (d *device) Value(cookie uint32) (int32, bool) {
	e := d.elems[cookie]
	if e == 0 || d.released || d.gone.Load() {
		return 0, false
	}
	var v cf.Ref
	if d.m.f.deviceValue(d.ref, e, &v) != 0 || v == 0 {
		return 0, false
	}
	return int32(d.m.f.valueInteger(v)), true
}

// Release implements Device.
func (d *device) Release() {
	if d.released {
		return
	}
	d.released = true
	cf.Release(d.ref)
}

// OpenHID implements Device: the reports of the device go to a transport, which a HID driver reads.
func (d *device) OpenHID() (gamepadhid.Transport, error) {
	return openTransport(d)
}

var _ Device = (*device)(nil)
