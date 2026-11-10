package winpad

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
)

// The HID drivers of the pads that have one (SPEC §28.1, §28.2, step 31.3): a PlayStation 4 or 5 pad, a Switch Pro controller
// or an Xbox pad over Bluetooth that Raw Input lists is opened through hid.dll (Opts.OpenHID) and read by its driver, which
// gives the light, the touchpad, the motion sensors, the trigger motors and the battery that the report descriptor does not
// show. A pad whose device cannot be opened or whose driver cannot start is read through Raw Input as before. XInput serves the
// Xbox pads while it is there.

// drvPad is a pad that a driver serves.
type drvPad struct {
	dev    *gamepadhid.Device
	bridge *gamepadhid.Bridge
	handle uintptr
	id     platform.PadID
	path   string
}

// drvEvent is what a driver tells the loop from its own goroutine: the pad is lost, or the driver could not start.
type drvEvent struct {
	pad    *drvPad
	failed bool
	err    error
}

// busOf is the bus of a HID device as the drivers name it.
func busOf(path string) gamepadhid.Bus {
	if busOfPath(path) == gamepaddb.BusBluetooth {
		return gamepadhid.BusBluetooth
	}
	return gamepadhid.BusUSB
}

// driverDevices are the devices of the drivers, which the UI thread asks to rumble and the loop adds and removes.
type driverDevices struct {
	mu   sync.RWMutex
	pads map[platform.PadID]*drvPad
}

func (d *driverDevices) get(id platform.PadID) *gamepadhid.Device {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if p := d.pads[id]; p != nil {
		return p.dev
	}
	return nil
}

func (d *driverDevices) add(p *drvPad) {
	d.mu.Lock()
	if d.pads == nil {
		d.pads = map[platform.PadID]*drvPad{}
	}
	d.pads[p.id] = p
	d.mu.Unlock()
}

func (d *driverDevices) remove(id platform.PadID) {
	d.mu.Lock()
	delete(d.pads, id)
	d.mu.Unlock()
}

// driverInfo says whether a driver serves a device, and gives the description the driver starts with.
func (r *run) driverInfo(d HIDDevice) (gamepadhid.DeviceInfo, bool) {
	info := gamepadhid.DeviceInfo{Vendor: d.Vendor, Product: d.Product, Version: d.Version, Bus: busOf(d.Path), Path: d.Path}
	model := gamepadhid.ModelOf(info)
	if model == gamepadhid.ModelNone {
		return info, false
	}
	if model == gamepadhid.ModelXboxOneBluetooth && r.s.opts.XInput != nil {
		// XInput serves the Xbox pads, with its own rumble and battery.
		return info, false
	}
	return info, true
}

// upgradeDrivers is the wake-up of the drivers: each pad that Raw Input serves and that a driver can serve is let go, and the
// handler is told it is removed; the look at the devices that follows starts its driver, or reads it through Raw Input again.
func (r *run) upgradeDrivers() {
	opts := &r.s.opts
	if opts.OpenHID == nil || opts.DisableDrivers {
		return
	}
	for handle, p := range r.hid {
		if _, ok := r.driverInfo(p.dev); !ok {
			continue
		}
		delete(r.hid, handle)
		opts.HID.Release(p.dev)
		id := p.id
		r.deliver(func(h platform.PadHandler) { h.PadRemoved(id) })
	}
}

// tryDriver starts the driver of a device if it has one, and reports whether the pad is served so, at least for now: the driver
// starts on a goroutine and tells the loop if it fails, which sends the pad back to Raw Input.
func (r *run) tryDriver(d HIDDevice) bool {
	opts := &r.s.opts
	if opts.OpenHID == nil || opts.DisableDrivers || r.s.asleep.Load() || r.drvFailed[d.Handle] {
		return false
	}
	info, ok := r.driverInfo(d)
	if !ok {
		return false
	}
	tr, err := opts.OpenHID(d.Path)
	if err != nil {
		r.s.log.Debug("a gamepad's HID device cannot be opened for its driver, so it is read through Raw Input", "path", d.Path, "err", err)
		r.drvFailed[d.Handle] = true
		return false
	}
	p := &drvPad{handle: d.Handle, id: platform.PadID(r.s.nextID.Add(1)), path: d.Path}
	p.bridge = &gamepadhid.Bridge{
		ID: p.id, Info: info,
		Deliver:  r.deliver,
		OnFailed: func(err error) { r.drvTell(drvEvent{pad: p, failed: true, err: err}) },
		OnLost:   func() { r.drvTell(drvEvent{pad: p}) },
	}
	dev, ok := gamepadhid.Start(info, tr, p.bridge, gamepadhid.Options{Clock: opts.Clock, Logger: r.s.log})
	if !ok {
		if err := tr.Close(); err != nil {
			r.s.log.Debug("a HID device did not close cleanly", "path", d.Path, "err", err)
		}
		return false
	}
	p.dev = dev
	r.drv[d.Handle] = p
	r.s.drivers.add(p)
	return true
}

// drvTell tells the loop what a driver found, from the goroutine of the driver.
func (r *run) drvTell(ev drvEvent) {
	select {
	case r.drvEvents <- ev:
	case <-r.stop:
	}
}

// driverEvent handles what a driver told the loop: the pad is dropped and read through Raw Input from now on, if it still is a
// pad there.
func (r *run) driverEvent(ev drvEvent) {
	p := ev.pad
	if r.drv[p.handle] != p {
		// The pad was dropped already (its device went away).
		return
	}
	delete(r.drv, p.handle)
	r.s.drivers.remove(p.id)
	p.dev.Close()
	r.drvFailed[p.handle] = true
	if ev.failed {
		r.s.log.Info("the HID driver of a gamepad could not start, so it is read through Raw Input", "path", p.path, "err", ev.err)
	}
	r.scanHID()
}

// dropDriver ends the driver of a pad whose device is gone, and tells the handler the pad is.
func (r *run) dropDriver(p *drvPad) {
	delete(r.drv, p.handle)
	delete(r.drvFailed, p.handle)
	r.closeDriver(p)
}

// closeDriver ends a driver and makes the removal of its pad reach the handler.
func (r *run) closeDriver(p *drvPad) {
	r.s.drivers.remove(p.id)
	p.dev.Close()
	p.bridge.Remove()
}

// closeDrivers ends every driver when the service stops. The removals are not delivered: the handler has stopped.
func (r *run) closeDrivers() {
	for handle, p := range r.drv {
		delete(r.drv, handle)
		r.s.drivers.remove(p.id)
		p.dev.Close()
	}
}
