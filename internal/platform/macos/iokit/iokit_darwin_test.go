//go:build darwin && !ios

package iokit

import (
	"bytes"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// fakeIO is a hidFuncs whose functions record what they are asked and answer from tables. The references that the package
// itself retains and releases (the manager, the device, the arrays it makes) are real Core Foundation numbers, so that those
// calls are valid; the rest are numbers that only the fake knows.
type fakeIO struct {
	*hidFuncs
	t     *testing.T
	calls []string
	fake  cf.Ref // the next number of a reference that only the fake knows

	createFails, arrayFails bool
	openCode                int32

	props  map[cf.Ref]any // the properties of the device by key: a uint32 or a string
	elems  map[cf.Ref]*fakeElement
	arrays map[cf.Ref][]cf.Ref
	top    []cf.Ref // the elements that the device lists
	values map[cf.Ref]int64
	vals   map[cf.Ref]int64 // the value references that deviceValue gave

	setCode, getCode int32
	setReports       [][]byte
	setTypes         []uint32
	getReply         []byte
	reportCB         uintptr
	reportCtx        uintptr
	posted           []func()
}

type fakeElement struct {
	typ, page, usage, cookie uint32
	min, max                 int64
	children                 []cf.Ref
	device                   cf.Ref
}

func newFakeIO(t *testing.T) *fakeIO {
	t.Helper()
	f := &fakeIO{hidFuncs: &hidFuncs{}, t: t, fake: 1 << 40, props: map[cf.Ref]any{}, elems: map[cf.Ref]*fakeElement{},
		arrays: map[cf.Ref][]cf.Ref{}, values: map[cf.Ref]int64{}, vals: map[cf.Ref]int64{}}
	h := f.hidFuncs
	note := func(s string) { f.calls = append(f.calls, s) }
	realRef := func() cf.Ref { return cf.NewNumber(7) }
	str := func(s string) cf.Ref { r := cf.NewString(s); t.Cleanup(func() { cf.Release(r) }); return r }
	h.managerCreate = func(uintptr, uint32) cf.Ref {
		note("create")
		if f.createFails {
			return 0
		}
		return realRef()
	}
	h.managerSetMatching = func(cf.Ref, cf.Ref) { note("matching") }
	h.managerOnDevice = func(_ cf.Ref, cb, _ uintptr) { note("onDevice") }
	h.managerOnRemoval = func(cf.Ref, uintptr, uintptr) { note("onRemoval") }
	h.managerOnValue = func(cf.Ref, uintptr, uintptr) { note("onValue") }
	h.managerSchedule = func(cf.Ref, cf.Ref, cf.Ref) { note("schedule") }
	h.managerUnschedule = func(cf.Ref, cf.Ref, cf.Ref) { note("unschedule") }
	h.managerOpen = func(cf.Ref, uint32) int32 { note("open"); return f.openCode }
	h.managerClose = func(cf.Ref, uint32) int32 { note("close"); return 0 }
	h.arrayCreate = func(_ uintptr, _ unsafe.Pointer, _ int64, _ uintptr) cf.Ref {
		note("arrayCreate")
		if f.arrayFails {
			return 0
		}
		return realRef()
	}
	h.arrayCount = func(a cf.Ref) int64 { return int64(len(f.arrays[a])) }
	h.arrayAt = func(a cf.Ref, i int64) cf.Ref { return f.arrays[a][i] }
	h.numberValue = func(n cf.Ref, _ int64, out unsafe.Pointer) bool {
		v, ok := f.props[n].(uint32)
		*(*int32)(out) = int32(v)
		return ok
	}
	h.deviceProperty = func(_ cf.Ref, key cf.Ref) cf.Ref {
		switch v := f.props[key].(type) {
		case uint32:
			return key // numberValue reads the number from the table, by the key
		case string:
			return str(v)
		}
		return 0
	}
	h.deviceElements = func(cf.Ref, cf.Ref, uint32) cf.Ref {
		a := realRef()
		f.arrays[a] = f.top
		return a
	}
	h.deviceValue = func(_ cf.Ref, e cf.Ref, out *cf.Ref) int32 {
		v, ok := f.values[e]
		if !ok {
			return -1
		}
		f.fake++
		f.vals[f.fake] = v
		*out = f.fake
		return 0
	}
	h.valueInteger = func(v cf.Ref) int64 { return f.vals[v] }
	h.valueElement = func(v cf.Ref) cf.Ref { return cf.Ref(f.vals[v+1]) }
	h.elementCookie = func(e cf.Ref) uint32 { return f.elems[e].cookie }
	h.elementType = func(e cf.Ref) uint32 { return f.elems[e].typ }
	h.elementUsagePage = func(e cf.Ref) uint32 { return f.elems[e].page }
	h.elementUsage = func(e cf.Ref) uint32 { return f.elems[e].usage }
	h.elementLogicalMin = func(e cf.Ref) int64 { return f.elems[e].min }
	h.elementLogicalMax = func(e cf.Ref) int64 { return f.elems[e].max }
	h.elementDevice = func(e cf.Ref) cf.Ref { return f.elems[e].device }
	h.elementChildren = func(e cf.Ref) cf.Ref {
		c := f.elems[e].children
		if len(c) == 0 {
			return 0
		}
		f.fake++
		f.arrays[f.fake] = c
		return f.fake
	}
	h.deviceSchedule = func(cf.Ref, cf.Ref, cf.Ref) { note("deviceSchedule") }
	h.deviceUnschedule = func(cf.Ref, cf.Ref, cf.Ref) { note("deviceUnschedule") }
	h.deviceOnReport = func(_ cf.Ref, _ uintptr, _ int64, cb, ctx uintptr) {
		note("onReport")
		f.reportCB, f.reportCtx = cb, ctx
	}
	h.deviceSetReport = func(_ cf.Ref, typ uint32, id int64, p unsafe.Pointer, n int64) int32 {
		f.setReports = append(f.setReports, append([]byte{byte(id)}, unsafe.Slice((*byte)(p), n)...))
		f.setTypes = append(f.setTypes, typ)
		return f.setCode
	}
	h.deviceGetReport = func(_ cf.Ref, _ uint32, _ int64, p unsafe.Pointer, n *int64) int32 {
		if f.getCode != 0 {
			return f.getCode
		}
		*n = int64(copy(unsafe.Slice((*byte)(p), *n), f.getReply))
		return 0
	}
	h.mainRunLoop = func() cf.Ref { return 0 }
	h.commonMode = func() cf.Ref { return 0 }
	h.arrayCallbacks = 0
	// The keys of a matching dictionary are real strings: Core Foundation hashes them.
	h.keyUsagePage, h.keyUsage = str("DeviceUsagePage"), str("DeviceUsage")
	for i, p := range []*cf.Ref{&h.propUsagePage, &h.propUsage, &h.propVendor, &h.propProduct, &h.propVer, &h.propMaker, &h.propName, &h.propTransport, &h.propLoc} {
		*p = cf.Ref(1<<41 + i)
	}
	return f
}

// manager makes a manager over the fake, whose post queues the functions.
func (f *fakeIO) manager() *manager {
	return newManager(f.hidFuncs, func(fn func()) { f.posted = append(f.posted, fn) }, ManagerOptions{})
}

// addElement adds an element to the fake, in the list of the device or in a collection, and returns its reference.
func (f *fakeIO) addElement(e fakeElement) cf.Ref {
	f.fake++
	f.elems[f.fake] = &e
	return f.fake
}

// pad makes the elements of a game pad in a collection, and a button outside it.
func (f *fakeIO) pad(dev cf.Ref) (x, hat, button cf.Ref) {
	x = f.addElement(fakeElement{typ: uint32(ElementInputMisc), page: pageGenericDesktop, usage: usageX, cookie: 1, min: 0, max: 255, device: dev})
	hat = f.addElement(fakeElement{typ: uint32(ElementInputMisc), page: pageGenericDesktop, usage: usageHatswitch, cookie: 2, min: 0, max: 7, device: dev})
	coll := f.addElement(fakeElement{typ: uint32(ElementCollection), children: []cf.Ref{x, hat}, device: dev})
	button = f.addElement(fakeElement{typ: uint32(ElementInputButton), page: pageButton, usage: 1, cookie: 3, max: 1, device: dev})
	f.top = []cf.Ref{coll, button}
	f.values[x], f.values[button] = 200, 1
	return x, hat, button
}

func (f *fakeIO) has(call string) bool {
	for _, c := range f.calls {
		if c == call {
			return true
		}
	}
	return false
}

func TestLoadFuncsBindsIOKit(t *testing.T) {
	f, err := loadFuncs()
	if err != nil {
		t.Fatalf("loadFuncs: %v", err)
	}
	if f.managerCreate == nil || f.arrayCallbacks == 0 || f.keyUsage == 0 {
		t.Error("the table is not complete")
	}
}

func TestManagerStartAndStop(t *testing.T) {
	f := newFakeIO(t)
	m := f.manager()
	if err := m.Start(Callbacks{}); err != nil {
		t.Fatal(err)
	}
	want := []string{"create", "arrayCreate", "matching", "onDevice", "onRemoval", "onValue", "schedule", "open"}
	if len(f.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", f.calls, want)
	}
	for i := range want {
		if f.calls[i] != want[i] {
			t.Errorf("call %d = %s, want %s", i, f.calls[i], want[i])
		}
	}
	f.calls = nil
	m.Stop()
	m.Stop()
	want = []string{"onDevice", "onRemoval", "onValue", "unschedule", "close"}
	if len(f.calls) != len(want) {
		t.Fatalf("stop calls = %v, want %v", f.calls, want)
	}
	// It starts again.
	f.calls = nil
	if err := m.Start(Callbacks{}); err != nil {
		t.Fatal(err)
	}
	m.Stop()
	if !f.has("create") {
		t.Errorf("calls = %v", f.calls)
	}
}

func TestManagerStartFailures(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakeIO)
		route string
	}{
		{"no manager", func(f *fakeIO) { f.createFails = true }, "manager.Start.HIDManagerFailed"},
		{"no array", func(f *fakeIO) { f.arrayFails = true }, "manager.setMatching.HIDManagerFailed"},
		{"cannot open", func(f *fakeIO) { f.openCode = -536870174 }, "manager.Start.HIDManagerOpenFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeIO(t)
			tt.setup(f)
			m := f.manager()
			err := m.Start(Callbacks{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route = %s, want %s", appErr.Route(), tt.route)
			}
			if m.started || m.ref != 0 {
				t.Error("a manager that did not start holds its reference")
			}
			// A manager that failed to start can be stopped.
			m.Stop()
		})
	}
}

func TestUnavailableManager(t *testing.T) {
	want := errors.New(tagIOKitUnavailable, "no IOKit")
	m := &unavailable{err: want}
	if err := m.Start(Callbacks{}); err != want {
		t.Errorf("Start = %v", err)
	}
	m.Stop()
}

func TestManagerReportsDevicesAndValues(t *testing.T) {
	f := newFakeIO(t)
	h := f.hidFuncs
	dev := cf.NewNumber(99) // a real reference: the device retains it
	t.Cleanup(func() { cf.Release(dev) })
	f.props[h.propUsagePage], f.props[h.propUsage] = uint32(pageGenericDesktop), uint32(usageGamePad)
	f.props[h.propVendor], f.props[h.propProduct], f.props[h.propVer] = uint32(0x046d), uint32(0xc216), uint32(0x0305)
	f.props[h.propMaker], f.props[h.propName], f.props[h.propTransport] = "Logitech", "Dual Action", "USB"
	f.props[h.propLoc] = uint32(0x14100000)
	x, hat, button := f.pad(dev)

	var added, removed []Device
	type input struct {
		cookie uint32
		value  int32
	}
	var inputs []input
	m := f.manager()
	err := m.Start(Callbacks{
		Added:   func(d Device) { added = append(added, d) },
		Removed: func(d Device) { removed = append(removed, d) },
		Input:   func(_ Device, c uint32, v int32) { inputs = append(inputs, input{c, v}) },
	})
	if err != nil {
		t.Fatal(err)
	}
	m.deviceAdded(dev)
	m.deviceAdded(dev) // the same device again is not a second device
	if len(added) != 1 {
		t.Fatalf("added = %d", len(added))
	}
	d := added[0]
	want := DeviceInfo{Vendor: 0x046d, Product: 0xc216, Version: 0x0305, VendorName: "Logitech", ProductName: "Dual Action", Transport: "USB",
		UsagePage: pageGenericDesktop, Usage: usageGamePad, Path: "0x14100000"}
	if d.Info() != want {
		t.Errorf("info = %+v, want %+v", d.Info(), want)
	}
	// The elements of the collection are listed, and the collection is not.
	raw := d.Elements()
	if len(raw) != 3 || raw[0].Cookie != 1 || raw[1].Cookie != 2 || raw[2].Cookie != 3 || raw[0].Max != 255 || raw[2].Page != pageButton {
		t.Errorf("elements = %+v", raw)
	}
	if v, ok := d.Value(1); !ok || v != 200 {
		t.Errorf("Value(1) = %d, %v", v, ok)
	}
	if v, ok := d.Value(3); !ok || v != 1 {
		t.Errorf("Value(3) = %d, %v", v, ok)
	}
	if _, ok := d.Value(2); ok {
		t.Error("a value that the system has none of")
	}
	if _, ok := d.Value(77); ok {
		t.Error("a value of an element that is not there")
	}

	// A change of a value reaches the callback with the cookie of its element.
	value := func(elem cf.Ref, v int64) cf.Ref {
		f.fake += 2
		f.vals[f.fake] = v
		f.vals[f.fake+1] = int64(elem)
		return f.fake
	}
	m.valueChanged(value(button, 0))
	m.valueChanged(value(hat, 3))
	m.valueChanged(value(x, -4))
	if len(inputs) != 3 || inputs[0] != (input{3, 0}) || inputs[1] != (input{2, 3}) || inputs[2] != (input{1, -4}) {
		t.Errorf("inputs = %+v", inputs)
	}

	// A value of a device that is not known is dropped.
	f.elems[x].device = 12345
	m.valueChanged(value(x, 1))
	if len(inputs) != 3 {
		t.Errorf("inputs = %+v", inputs)
	}

	m.deviceRemoved(dev)
	m.deviceRemoved(dev)
	if len(removed) != 1 || removed[0] != d {
		t.Errorf("removed = %v", removed)
	}
	if _, ok := d.Value(1); ok {
		t.Error("a value of a device that has gone")
	}
	d.Release()
	d.Release()
	m.Stop()
	// A late callback of the manager reaches nothing.
	m.deviceAdded(dev)
	if len(added) != 1 {
		t.Errorf("added = %d", len(added))
	}
}

func TestManagerKeepsPanicsOfItsCallbacksOut(t *testing.T) {
	f := newFakeIO(t)
	dev := cf.NewNumber(99)
	t.Cleanup(func() { cf.Release(dev) })
	m := newManager(f.hidFuncs, func(func()) {}, ManagerOptions{Guard: func(fn func()) {
		defer func() { recover() }()
		fn()
	}})
	if err := m.Start(Callbacks{Added: func(Device) { panic("the handler broke") }}); err != nil {
		t.Fatal(err)
	}
	m.deviceAdded(dev) // does not panic
	m.Stop()
}

// transportOf makes a device on the fake and the transport of its reports.
func transportOf(t *testing.T, f *fakeIO) (*transport, *device) {
	t.Helper()
	dev := cf.NewNumber(99)
	t.Cleanup(func() { cf.Release(dev) })
	m := f.manager()
	if err := m.Start(Callbacks{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Stop)
	d := newDevice(m, dev)
	d.info.ProductName = "Pad"
	m.devices[dev] = d
	tr, err := openTransport(d)
	if err != nil {
		t.Fatal(err)
	}
	return tr.(*transport), d
}

func TestTransportReadsReports(t *testing.T) {
	f := newFakeIO(t)
	tr, _ := transportOf(t, f)
	if !f.has("onReport") || !f.has("deviceSchedule") || f.reportCB == 0 || f.reportCtx != tr.ctx {
		t.Fatalf("calls = %v", f.calls)
	}
	buf := make([]byte, 64)
	// Nothing comes: the time runs out and that is no error.
	if n, err := tr.Read(buf, 5*time.Millisecond); n != 0 || err != nil {
		t.Errorf("Read = %d, %v", n, err)
	}
	// A report of a device that has report IDs has its ID in front already; one that has not gets it.
	mem := dl.Malloc(8)
	defer dl.Free(mem)
	copy(dl.Bytes(mem, 8), []byte{1, 2, 3, 4, 5, 6, 7, 8})
	tr.report(1, mem, 8)
	if n, err := tr.Read(buf, time.Second); err != nil || !bytes.Equal(buf[:n], []byte{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Errorf("Read = %x, %v", buf[:n], err)
	}
	tr.report(5, mem, 3)
	if n, err := tr.Read(buf, time.Second); err != nil || !bytes.Equal(buf[:n], []byte{5, 1, 2, 3}) {
		t.Errorf("Read = %x, %v", buf[:n], err)
	}
	tr.report(0, mem, 2)
	if n, err := tr.Read(buf, time.Second); err != nil || !bytes.Equal(buf[:n], []byte{1, 2}) {
		t.Errorf("Read = %x, %v", buf[:n], err)
	}
	// Empty reports are dropped, and a full queue drops the rest.
	tr.report(1, mem, 0)
	tr.report(1, 0, 4)
	for range 100 {
		tr.report(1, mem, 4)
	}
	if len(tr.in) != cap(tr.in) {
		t.Errorf("queued %d reports", len(tr.in))
	}
}

func TestTransportWritesAndAsksForReports(t *testing.T) {
	f := newFakeIO(t)
	tr, _ := transportOf(t, f)
	// An output report with an ID goes with its ID; with the ID 0 it goes without it.
	if n, err := tr.Write([]byte{5, 1, 2, 3}); err != nil || n != 4 {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if n, err := tr.Write([]byte{0, 9, 8}); err != nil || n != 3 {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if n, err := tr.SendFeature([]byte{2, 7}); err != nil || n != 2 {
		t.Fatalf("SendFeature = %d, %v", n, err)
	}
	if n, err := tr.Write(nil); err != nil || n != 0 {
		t.Errorf("Write(nil) = %d, %v", n, err)
	}
	// IOKit takes the ID apart from the data: the fake gives it back in front of what it got.
	want := [][]byte{{5, 5, 1, 2, 3}, {0, 9, 8}, {2, 2, 7}}
	if len(f.setReports) != 3 {
		t.Fatalf("reports = %x", f.setReports)
	}
	for i := range want {
		if !bytes.Equal(f.setReports[i], want[i]) {
			t.Errorf("report %d = %x, want %x", i, f.setReports[i], want[i])
		}
	}
	if f.setTypes[0] != reportTypeOutput || f.setTypes[2] != reportTypeFeature {
		t.Errorf("types = %v", f.setTypes)
	}
	// A feature report comes with its ID in front.
	f.getReply = []byte{0x11, 0x22, 0x33}
	buf := []byte{0x02, 0, 0, 0, 0}
	if n, err := tr.GetFeature(buf); err != nil || n != 3 || !bytes.Equal(buf[:3], []byte{0x11, 0x22, 0x33}) {
		t.Errorf("GetFeature = %d, %v, %x", n, err, buf)
	}
	// A report without an ID gets one more byte in what is returned (the ID, 0).
	buf = []byte{0x00, 0, 0, 0}
	if n, err := tr.GetFeature(buf); err != nil || n != 4 || !bytes.Equal(buf[1:], []byte{0x11, 0x22, 0x33}) {
		t.Errorf("GetFeature = %d, %v, %x", n, err, buf)
	}
}

func TestTransportErrors(t *testing.T) {
	f := newFakeIO(t)
	tr, d := transportOf(t, f)
	f.setCode, f.getCode = -1, -2
	routes := []struct {
		name  string
		err   error
		route string
	}{
		{"write", func() error { _, err := tr.Write([]byte{1, 2}); return err }(), "transport.Write->transport.set.HIDReportFailed"},
		{"feature", func() error { _, err := tr.SendFeature([]byte{1, 2}); return err }(), "transport.SendFeature->transport.set.HIDReportFailed"},
		{"get", func() error { _, err := tr.GetFeature([]byte{1, 0}); return err }(), "transport.GetFeature.HIDReportFailed"},
		{"get without room", func() error { _, err := tr.GetFeature(nil); return err }(), "transport.GetFeature.HIDReportFailed"},
		{"get with only an ID of 0", func() error { _, err := tr.GetFeature([]byte{0}); return err }(), "transport.GetFeature.HIDReportFailed"},
	}
	for _, tt := range routes {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.err)
			if !ok {
				t.Fatalf("error = %T %v", tt.err, tt.err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route = %s, want %s", appErr.Route(), tt.route)
			}
		})
	}

	// A device that has gone ends the reads.
	d.gone.Store(true)
	_, err := tr.Read(make([]byte, 8), time.Millisecond)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("transport.Read.HIDDeviceClosed") {
		t.Errorf("Read on a device that is gone = %v", err)
	}
	if _, err := openTransport(d); err == nil {
		t.Error("a transport of a device that is gone")
	} else if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("openTransport.HIDDeviceClosed") {
		t.Errorf("openTransport = %v", err)
	}
	d.gone.Store(false)

	// A transport that was closed refuses everything and ends the reads that wait.
	waiting := make(chan error, 1)
	go func() { _, err := tr.Read(make([]byte, 8), 5*time.Second); waiting <- err }()
	time.Sleep(10 * time.Millisecond)
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-waiting:
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("transport.Read.HIDDeviceClosed") {
			t.Errorf("Read after Close = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a read that waits was not ended by Close")
	}
	if _, err := tr.Write([]byte{1, 2}); err == nil {
		t.Error("Write after Close")
	}
	if _, err := tr.GetFeature([]byte{1, 0}); err == nil {
		t.Error("GetFeature after Close")
	}
	// The callback is taken away on the UI thread, where it was added.
	if f.has("deviceUnschedule") || len(f.posted) != 1 {
		t.Fatalf("calls = %v, posted %d", f.calls, len(f.posted))
	}
	f.posted[0]()
	if !f.has("deviceUnschedule") || f.reportCB != 0 || lookup(tr.ctx) != nil {
		t.Errorf("calls = %v, callback = %d", f.calls, f.reportCB)
	}
}

// TestServiceOverTheRealManagerStartsAndStopsWithoutPads starts the whole backend on the real manager, which finds no pad on a
// machine without one, and stops it, twice. It asks for pads only: a test that asked for a mouse or a keyboard would make macOS
// ask the user for the permission of Input Monitoring.
func TestServiceOverTheRealManagerStartsAndStopsWithoutPads(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f, err := loadFuncs()
	if err != nil {
		t.Fatal(err)
	}
	m := newManager(f, func(func()) {}, ManagerOptions{})
	m.runLoop = cf.CurrentRunLoop
	s := New(Options{Manager: m, Post: func(func()) {}})
	h := &recorder{}
	for range 2 {
		if err := s.Start(h); err != nil {
			t.Skipf("the manager cannot be opened here: %v", err)
		}
		cf.RunInMode(cf.DefaultMode(), 100*time.Millisecond, false)
		s.Stop()
	}
	for _, p := range h.added {
		t.Logf("found a pad: %s", p.Name)
	}
}
