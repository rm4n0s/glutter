//go:build darwin && !ios

package iokit

import (
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// ffFuncs is the table of the calls into ForceFeedback.framework (and the two that find a device's IOService and make the sine
// effect's UUID). A test puts its own functions in loadFF's place.
type ffFuncs struct {
	service         func(device cf.Ref) uint32 // IOHIDDeviceGetService
	isForceFeedback func(service uint32) int32 // FFIsForceFeedback
	createDevice    func(service uint32, out *uintptr) int32
	releaseDevice   func(dev uintptr) int32
	command         func(dev uintptr, cmd uint32) int32
	createEffect    func(dev uintptr, typ cf.Ref, e unsafe.Pointer, out *uintptr) int32
	releaseEffect   func(dev, effect uintptr) int32
	setParameters   func(effect uintptr, e unsafe.Pointer, flags uint32) int32
	start           func(effect uintptr, iterations, flags uint32) int32
	stop            func(effect uintptr) int32
	sineUUID        cf.Ref
}

// loadFF binds the framework. A machine without it (or without a symbol) has no force feedback, and nothing else is lost.
var loadFF = sync.OnceValues(func() (*ffFuncs, error) {
	io, err := dl.OpenFramework("IOKit")
	if err != nil {
		return nil, errors.NewErr(tagIOKitUnavailable, err)
	}
	ff, err := dl.OpenFramework("ForceFeedback")
	if err != nil {
		return nil, errors.NewErr(tagIOKitUnavailable, err)
	}
	cfl, err := dl.OpenFramework("CoreFoundation")
	if err != nil {
		return nil, errors.NewErr(tagIOKitUnavailable, err)
	}
	f := new(ffFuncs)
	var uuidFromString func(alloc uintptr, s cf.Ref) cf.Ref
	for _, b := range []struct {
		lib    *dl.Lib
		fptr   any
		symbol string
	}{
		{io, &f.service, "IOHIDDeviceGetService"},
		{ff, &f.isForceFeedback, "FFIsForceFeedback"}, {ff, &f.createDevice, "FFCreateDevice"},
		{ff, &f.releaseDevice, "FFReleaseDevice"}, {ff, &f.command, "FFDeviceSendForceFeedbackCommand"},
		{ff, &f.createEffect, "FFDeviceCreateEffect"}, {ff, &f.releaseEffect, "FFDeviceReleaseEffect"},
		{ff, &f.setParameters, "FFEffectSetParameters"}, {ff, &f.start, "FFEffectStart"}, {ff, &f.stop, "FFEffectStop"},
		{cfl, &uuidFromString, "CFUUIDCreateFromString"},
	} {
		if err := b.lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, errors.NewErr(tagIOKitUnavailable, err).SetMetadata("symbol", b.symbol)
		}
	}
	s := cf.NewString(ffSineEffectUUID)
	f.sineUUID = uuidFromString(0, s)
	cf.Release(s)
	if f.sineUUID == 0 {
		return nil, errors.New(tagIOKitUnavailable, "the UUID of the sine effect cannot be made")
	}
	return f, nil
})

// ffLoader finds the calls; tests replace it.
var ffLoader = loadFF

var _ ForceFeedbacker = (*device)(nil)

// HasForceFeedback implements ForceFeedbacker.
func (d *device) HasForceFeedback() bool {
	f, err := ffLoader()
	if err != nil || d.released || d.gone.Load() {
		return false
	}
	service := f.service(d.ref)
	return service != 0 && f.isForceFeedback(service) == ffOK
}

// OpenForceFeedback implements ForceFeedbacker.
func (d *device) OpenForceFeedback() (ForceFeedback, error) {
	f, err := ffLoader()
	if err != nil {
		return nil, err
	}
	if d.released || d.gone.Load() {
		return nil, errors.New(tagForceFeedbackGone, "the device is gone")
	}
	var dev uintptr
	if hr := f.createDevice(f.service(d.ref), &dev); hr != ffOK || dev == 0 {
		return nil, errors.New(tagForceFeedbackDevice, "the force feedback device cannot be made from its service", "result", int(hr))
	}
	return newRumbler(&realFF{f: f, dev: dev})
}

// realFF is the ffAPI over the framework's calls for one device.
type realFF struct {
	f   *ffFuncs
	dev uintptr
}

func (r *realFF) Command(cmd uint32) int32 { return r.f.command(r.dev, cmd) }

func (r *realFF) CreateEffect(e *ffEffect) (uintptr, int32) {
	var handle uintptr
	hr := r.f.createEffect(r.dev, r.f.sineUUID, unsafe.Pointer(e), &handle)
	return handle, hr
}

func (r *realFF) SetParameters(handle uintptr, e *ffEffect, flags uint32) int32 {
	return r.f.setParameters(handle, unsafe.Pointer(e), flags)
}

func (r *realFF) Start(handle uintptr) int32 { return r.f.start(handle, 1, 0) }
func (r *realFF) Stop(handle uintptr) int32  { return r.f.stop(handle) }
func (r *realFF) ReleaseEffect(handle uintptr) {
	_ = r.f.releaseEffect(r.dev, handle) // the effect is let go with its pad: nothing is left to recover
}
func (r *realFF) ReleaseDevice() { _ = r.f.releaseDevice(r.dev) } // the same
