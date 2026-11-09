//go:build darwin && !ios

package iokit

import (
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// This file is the call into IOKit's HID manager (SDL 3's src/joystick/darwin/SDL_iokitjoystick.c is where the rules came from;
// this is a new Go version of it, and changed). The manager is scheduled on the main run loop, which is the UI thread of the
// app, so its callbacks run where the Service wants them. Every call goes through a hidFuncs table, so that tests put their own
// functions in it and try the paths that a pad would take.

// hidFuncs is the table of the IOKit and Core Foundation functions of this package.
type hidFuncs struct {
	managerCreate           func(alloc uintptr, options uint32) cf.Ref
	managerSetMatching      func(m, array cf.Ref)
	managerOnDevice         func(m cf.Ref, cb, ctx uintptr)
	managerOnRemoval        func(m cf.Ref, cb, ctx uintptr)
	managerOnValue          func(m cf.Ref, cb, ctx uintptr)
	managerSchedule         func(m, rl, mode cf.Ref)
	managerUnschedule       func(m, rl, mode cf.Ref)
	managerOpen             func(m cf.Ref, options uint32) int32
	managerClose            func(m cf.Ref, options uint32) int32
	deviceProperty          func(d, key cf.Ref) cf.Ref
	deviceElements          func(d, matching cf.Ref, options uint32) cf.Ref
	deviceValue             func(d, elem cf.Ref, value *cf.Ref) int32
	deviceSchedule          func(d, rl, mode cf.Ref)
	deviceUnschedule        func(d, rl, mode cf.Ref)
	deviceOnReport          func(d cf.Ref, report uintptr, length int64, cb, ctx uintptr)
	deviceSetReport         func(d cf.Ref, typ uint32, id int64, report unsafe.Pointer, length int64) int32
	deviceGetReport         func(d cf.Ref, typ uint32, id int64, report unsafe.Pointer, length *int64) int32
	elementCookie           func(e cf.Ref) uint32
	elementType             func(e cf.Ref) uint32
	elementUsagePage        func(e cf.Ref) uint32
	elementUsage            func(e cf.Ref) uint32
	elementLogicalMin       func(e cf.Ref) int64
	elementLogicalMax       func(e cf.Ref) int64
	elementChildren         func(e cf.Ref) cf.Ref
	elementDevice           func(e cf.Ref) cf.Ref
	valueInteger            func(v cf.Ref) int64
	valueElement            func(v cf.Ref) cf.Ref
	arrayCreate             func(alloc uintptr, values unsafe.Pointer, n int64, callbacks uintptr) cf.Ref
	arrayCount              func(a cf.Ref) int64
	arrayAt                 func(a cf.Ref, i int64) cf.Ref
	numberValue             func(n cf.Ref, typ int64, out unsafe.Pointer) bool
	arrayCallbacks          uintptr
	keyUsagePage, keyUsage  cf.Ref // the keys of a matching dictionary
	propUsagePage           cf.Ref // the properties of a device
	propUsage, propVendor   cf.Ref
	propProduct, propVer    cf.Ref
	propMaker, propName     cf.Ref
	propTransport, propLoc  cf.Ref
	mainRunLoop, commonMode func() cf.Ref
}

// kCFNumberSInt32Type is Core Foundation's 32-bit signed integer number type.
const kCFNumberSInt32Type = 3

// The report types of IOHIDDeviceSetReport and IOHIDDeviceGetReport.
const (
	reportTypeOutput  = 1
	reportTypeFeature = 2
)

// loadFuncs binds IOKit and Core Foundation. A failure is the tag IOKitUnavailable.
var loadFuncs = sync.OnceValues(func() (*hidFuncs, error) {
	io, err := dl.OpenFramework("IOKit")
	if err != nil {
		return nil, errors.NewErr(tagIOKitUnavailable, err)
	}
	cfl, err := dl.OpenFramework("CoreFoundation")
	if err != nil {
		return nil, errors.NewErr(tagIOKitUnavailable, err)
	}
	f := new(hidFuncs)
	for _, b := range []struct {
		lib    *dl.Lib
		fptr   any
		symbol string
	}{
		{io, &f.managerCreate, "IOHIDManagerCreate"}, {io, &f.managerSetMatching, "IOHIDManagerSetDeviceMatchingMultiple"},
		{io, &f.managerOnDevice, "IOHIDManagerRegisterDeviceMatchingCallback"},
		{io, &f.managerOnRemoval, "IOHIDManagerRegisterDeviceRemovalCallback"},
		{io, &f.managerOnValue, "IOHIDManagerRegisterInputValueCallback"},
		{io, &f.managerSchedule, "IOHIDManagerScheduleWithRunLoop"}, {io, &f.managerUnschedule, "IOHIDManagerUnscheduleFromRunLoop"},
		{io, &f.managerOpen, "IOHIDManagerOpen"}, {io, &f.managerClose, "IOHIDManagerClose"},
		{io, &f.deviceProperty, "IOHIDDeviceGetProperty"}, {io, &f.deviceElements, "IOHIDDeviceCopyMatchingElements"},
		{io, &f.deviceValue, "IOHIDDeviceGetValue"},
		{io, &f.deviceSchedule, "IOHIDDeviceScheduleWithRunLoop"}, {io, &f.deviceUnschedule, "IOHIDDeviceUnscheduleFromRunLoop"},
		{io, &f.deviceOnReport, "IOHIDDeviceRegisterInputReportCallback"},
		{io, &f.deviceSetReport, "IOHIDDeviceSetReport"}, {io, &f.deviceGetReport, "IOHIDDeviceGetReport"},
		{io, &f.elementCookie, "IOHIDElementGetCookie"}, {io, &f.elementType, "IOHIDElementGetType"},
		{io, &f.elementUsagePage, "IOHIDElementGetUsagePage"}, {io, &f.elementUsage, "IOHIDElementGetUsage"},
		{io, &f.elementLogicalMin, "IOHIDElementGetLogicalMin"}, {io, &f.elementLogicalMax, "IOHIDElementGetLogicalMax"},
		{io, &f.elementChildren, "IOHIDElementGetChildren"}, {io, &f.elementDevice, "IOHIDElementGetDevice"},
		{io, &f.valueInteger, "IOHIDValueGetIntegerValue"}, {io, &f.valueElement, "IOHIDValueGetElement"},
		{cfl, &f.arrayCreate, "CFArrayCreate"}, {cfl, &f.arrayCount, "CFArrayGetCount"},
		{cfl, &f.arrayAt, "CFArrayGetValueAtIndex"}, {cfl, &f.numberValue, "CFNumberGetValue"},
	} {
		if err := b.lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, errors.NewErr(tagIOKitUnavailable, err).SetMetadata("symbol", b.symbol)
		}
	}
	addr, err := cfl.Symbol("kCFTypeArrayCallBacks")
	if err != nil {
		return nil, errors.NewErr(tagIOKitUnavailable, err)
	}
	f.arrayCallbacks = addr
	// The keys are made once for the process: a CFString constant lives as long as the process does.
	for _, k := range []struct {
		dst *cf.Ref
		s   string
	}{
		{&f.keyUsagePage, "DeviceUsagePage"}, {&f.keyUsage, "DeviceUsage"},
		{&f.propUsagePage, "PrimaryUsagePage"}, {&f.propUsage, "PrimaryUsage"}, {&f.propVendor, "VendorID"},
		{&f.propProduct, "ProductID"}, {&f.propVer, "VersionNumber"}, {&f.propMaker, "Manufacturer"}, {&f.propName, "Product"},
		{&f.propTransport, "Transport"}, {&f.propLoc, "LocationID"},
	} {
		*k.dst = cf.NewString(k.s)
	}
	f.mainRunLoop, f.commonMode = cf.MainRunLoop, cf.CommonModes
	return f, nil
})

// handlers maps the context that IOKit hands back to what it stands for, because a callback made once for the process must find
// its object (SPEC §8.1 rule 2). It is process-wide state with its reason: purego callbacks are never freed. An entry is deleted
// when its manager, device or transport ends, so the map holds only what is alive.
var handlers struct {
	mu   sync.RWMutex
	next uintptr
	m    map[uintptr]any
}

// register stores what a context stands for and returns the context.
func register(v any) uintptr {
	handlers.mu.Lock()
	defer handlers.mu.Unlock()
	if handlers.m == nil {
		handlers.m = map[uintptr]any{}
	}
	handlers.next++
	handlers.m[handlers.next] = v
	return handlers.next
}

// lookup returns what a context stands for, or nil.
func lookup(ctx uintptr) any {
	handlers.mu.RLock()
	defer handlers.mu.RUnlock()
	return handlers.m[ctx]
}

// forget drops a context.
func forget(ctx uintptr) {
	handlers.mu.Lock()
	delete(handlers.m, ctx)
	handlers.mu.Unlock()
}

// The callbacks of IOKit, made once, on first use (counted by package dl). Each finds its object by its context.
var (
	deviceCallback = sync.OnceValue(func() uintptr {
		return dl.Callback(func(ctx uintptr, _ int32, _ uintptr, dev cf.Ref) {
			if m, ok := lookup(ctx).(*manager); ok {
				m.deviceAdded(dev)
			}
		})
	})
	removalCallback = sync.OnceValue(func() uintptr {
		return dl.Callback(func(ctx uintptr, _ int32, _ uintptr, dev cf.Ref) {
			if m, ok := lookup(ctx).(*manager); ok {
				m.deviceRemoved(dev)
			}
		})
	})
	valueCallback = sync.OnceValue(func() uintptr {
		return dl.Callback(func(ctx uintptr, _ int32, _ uintptr, value cf.Ref) {
			if m, ok := lookup(ctx).(*manager); ok {
				m.valueChanged(value)
			}
		})
	})
)

// ManagerOptions are the options of NewManager.
//
// Flutter: no counterpart
type ManagerOptions struct {
	// Guard runs fn and keeps a panic of it from crossing back into IOKit, which would end the process (the platform's own
	// guard: it reports the panic to the app). Nil calls fn as it is.
	Guard func(fn func())
}

// manager is the Manager of the HID devices of the system, on IOKit's HID manager.
type manager struct {
	f     *hidFuncs
	post  func(func())
	guard func(fn func())
	// matching is what the manager asks for, and runLoop where it is scheduled; the tests change them.
	matching [][2]uint32
	runLoop  func() cf.Ref

	ref     cf.Ref
	ctx     uintptr
	cb      Callbacks
	devices map[cf.Ref]*device
	started bool
}

// NewManager makes the Manager of the system's HID devices. It is for the Service of the macOS platform: post runs a function on
// the UI thread from any goroutine, for the HID transports.
//
// Flutter: no counterpart
func NewManager(post func(func()), o ManagerOptions) Manager {
	f, err := loadFuncs()
	if err != nil {
		return &unavailable{err: err}
	}
	return newManager(f, post, o)
}

func newManager(f *hidFuncs, post func(func()), o ManagerOptions) *manager {
	g := o.Guard
	if g == nil {
		g = func(fn func()) { fn() }
	}
	return &manager{f: f, post: post, guard: g, matching: MatchingPages(), runLoop: f.mainRunLoop, devices: map[cf.Ref]*device{}}
}

// unavailable is the Manager of a system whose IOKit cannot be bound: every Start fails with the same error.
type unavailable struct{ err error }

func (u *unavailable) Start(Callbacks) error { return u.err }
func (u *unavailable) Stop()                 {}

// Start implements Manager. It must be called on the thread that wants the callbacks (the main thread).
func (m *manager) Start(cb Callbacks) error {
	if m.started {
		m.Stop()
	}
	f := m.f
	m.ref = f.managerCreate(0, 0)
	if m.ref == 0 {
		return errors.New(tagHIDManagerFailed, "IOHIDManagerCreate gave no manager")
	}
	m.cb = cb
	m.ctx = register(m)
	if err := m.setMatching(); err != nil {
		m.ctx = 0
		return err
	}
	f.managerOnDevice(m.ref, deviceCallback(), m.ctx)
	f.managerOnRemoval(m.ref, removalCallback(), m.ctx)
	f.managerOnValue(m.ref, valueCallback(), m.ctx)
	f.managerSchedule(m.ref, m.runLoop(), f.commonMode())
	if code := f.managerOpen(m.ref, 0); code != 0 {
		f.managerUnschedule(m.ref, m.runLoop(), f.commonMode())
		forget(m.ctx)
		cf.Release(m.ref)
		m.ref, m.ctx = 0, 0
		return errors.New(tagHIDManagerOpenFailed, "IOHIDManagerOpen failed", "code", int64(code))
	}
	m.started = true
	return nil
}

// setMatching tells the manager which devices to report. A failure gives the manager back.
func (m *manager) setMatching() error {
	f := m.f
	var dicts []cf.Ref
	for _, p := range m.matching {
		page, usage := cf.NewNumber(int64(p[0])), cf.NewNumber(int64(p[1]))
		dicts = append(dicts, cf.NewDictionary([]cf.Ref{f.keyUsagePage, f.keyUsage}, []cf.Ref{page, usage}))
		cf.Release(page)
		cf.Release(usage)
	}
	arr := f.arrayCreate(0, unsafe.Pointer(unsafe.SliceData(dicts)), int64(len(dicts)), f.arrayCallbacks)
	for _, d := range dicts {
		cf.Release(d)
	}
	if arr == 0 {
		forget(m.ctx)
		cf.Release(m.ref)
		m.ref = 0
		return errors.New(tagHIDManagerFailed, "CFArrayCreate gave no array of matching dictionaries")
	}
	f.managerSetMatching(m.ref, arr)
	cf.Release(arr)
	return nil
}

// Stop implements Manager.
func (m *manager) Stop() {
	if !m.started {
		return
	}
	m.started = false
	f := m.f
	f.managerOnDevice(m.ref, 0, 0)
	f.managerOnRemoval(m.ref, 0, 0)
	f.managerOnValue(m.ref, 0, 0)
	f.managerUnschedule(m.ref, m.runLoop(), f.commonMode())
	f.managerClose(m.ref, 0)
	forget(m.ctx)
	cf.Release(m.ref)
	m.ref, m.ctx = 0, 0
	m.devices = map[cf.Ref]*device{}
	m.cb = Callbacks{}
}

// deviceAdded is the callback of a device that matches.
func (m *manager) deviceAdded(ref cf.Ref) {
	m.guard(func() {
		if !m.started || m.devices[ref] != nil {
			return
		}
		d := newDevice(m, ref)
		m.devices[ref] = d
		if m.cb.Added != nil {
			m.cb.Added(d)
		}
	})
}

// deviceRemoved is the callback of a device that has gone.
func (m *manager) deviceRemoved(ref cf.Ref) {
	m.guard(func() {
		d := m.devices[ref]
		if !m.started || d == nil {
			return
		}
		delete(m.devices, ref)
		d.gone.Store(true)
		if m.cb.Removed != nil {
			m.cb.Removed(d)
		}
	})
}

// valueChanged is the callback of an element that has a new value.
func (m *manager) valueChanged(value cf.Ref) {
	m.guard(func() {
		if !m.started || m.cb.Input == nil {
			return
		}
		f := m.f
		elem := f.valueElement(value)
		if elem == 0 {
			return
		}
		d := m.devices[f.elementDevice(elem)]
		if d == nil {
			return
		}
		m.cb.Input(d, f.elementCookie(elem), int32(f.valueInteger(value)))
	})
}

// The tags of the errors of this file.
const (
	tagIOKitUnavailable     = "IOKitUnavailable"
	tagHIDManagerFailed     = "HIDManagerFailed"
	tagHIDManagerOpenFailed = "HIDManagerOpenFailed"
)
