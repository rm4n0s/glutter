//go:build linux && !android

package evdev

import (
	"path/filepath"
	"syscall"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
)

// The HID drivers of the pads that have one (SPEC §28.1, §28.2): a PlayStation 4 or 5 pad, a Switch Pro controller or an Xbox pad
// over Bluetooth is read through its hidraw node, which gives the light, the touchpad, the motion sensors, the trigger motors and
// the battery that its event device does not. A pad whose hidraw node cannot be opened (this user may not read it) or whose
// driver cannot start is read through its event device, as any other.

// busOf is the bus of an input device as the HID drivers name it.
func busOf(bus uint16) gamepadhid.Bus {
	switch bus {
	case gamepaddb.BusUSB:
		return gamepadhid.BusUSB
	case gamepaddb.BusBluetooth:
		return gamepadhid.BusBluetooth
	}
	return gamepadhid.BusUnknown
}

// tryHID starts the HID driver of the pad of an event node if there is one. It reports whether the pad is served so, at least for
// now: the driver starts on a goroutine, and tells the loop if it fails, which sends the pad back to its event device.
func (r *run) tryHID(node string, c *caps) bool {
	if r.s.noHID || r.hidFailed[node] {
		return false
	}
	info := gamepadhid.DeviceInfo{Vendor: c.ID.Vendor, Product: c.ID.Product, Version: c.ID.Version, Bus: busOf(c.ID.Bus), Name: c.Name}
	if gamepadhid.ModelOf(info) == gamepadhid.ModelNone {
		return false
	}
	raw := hidrawOf(r.s.sysDir, node)
	if raw == "" {
		return false
	}
	info.Path = filepath.Join(r.s.hidDir, raw)
	tr, err := r.s.openHID(info.Path)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == tagHidrawPermission {
			if !r.warnedHID {
				r.warnedHID = true
				r.s.log.Info("a gamepad has a HID driver, but this user may not read its hidraw node, so it works through its event device without the light bar, the touchpad, the motion sensors and the trigger motors; a udev rule gives access (docs/GUIDE.md)",
					"device", info.Path, "err", err)
			}
		} else {
			r.s.log.Debug("a gamepad's hidraw node cannot be opened", "device", info.Path, "err", err)
		}
		r.hidFailed[node] = true
		return false
	}
	hp := &hidPad{id: platform.PadID(r.s.nextID.Add(1)), node: node, path: info.Path}
	hp.bridge = &gamepadhid.Bridge{
		ID: hp.id, Info: info,
		Deliver:  r.deliver,
		OnFailed: func(err error) { r.hidTell(hidEvent{pad: hp, failed: true, err: err}) },
		OnLost:   func() { r.hidTell(hidEvent{pad: hp}) },
	}
	dev, ok := gamepadhid.Start(info, tr, hp.bridge, gamepadhid.Options{Clock: monotonicNow, Logger: r.s.log})
	if !ok {
		if err := tr.Close(); err != nil {
			r.s.log.Debug("a hidraw node did not close cleanly", "device", info.Path, "err", err)
		}
		return false
	}
	hp.dev = dev
	r.hidpads[node] = hp
	r.s.hidMu.Lock()
	r.s.hid[hp.id] = hp
	r.s.hidMu.Unlock()
	return true
}

// hidTell tells the loop what a driver found, from the goroutine of the driver.
func (r *run) hidTell(ev hidEvent) {
	select {
	case r.hidEvents <- ev:
	case <-r.done:
		return
	}
	// The loop is told by the pipe; a pipe that is full has the loop woken already.
	_, _ = syscall.Write(r.hidW, []byte{1})
}

// hidEventsDue handles what the drivers told the loop.
func (r *run) hidEventsDue() {
	buf := make([]byte, 64)
	for {
		if n, err := syscall.Read(r.hidR, buf); err != nil || n <= 0 {
			break
		}
	}
	for {
		select {
		case ev := <-r.hidEvents:
			r.hidEvent(ev)
		default:
			return
		}
	}
}

func (r *run) hidEvent(ev hidEvent) {
	hp := ev.pad
	if r.hidpads[hp.node] != hp {
		// The pad was dropped already (its node went away).
		return
	}
	delete(r.hidpads, hp.node)
	r.s.hidMu.Lock()
	delete(r.s.hid, hp.id)
	r.s.hidMu.Unlock()
	hp.dev.Close()
	// Whatever the reason, the pad is read through its event device from now on, if it is still there.
	r.hidFailed[hp.node] = true
	if ev.failed {
		r.s.log.Info("the HID driver of a gamepad could not start, so it works through its event device", "device", hp.path, "err", ev.err)
	}
	r.consider(hp.node)
}

// dropHID ends the driver of a pad whose node is gone, and tells the handler the pad is.
func (r *run) dropHID(node string, hp *hidPad) {
	delete(r.hidpads, node)
	delete(r.seen, node)
	delete(r.hidFailed, node)
	r.closeHID(hp)
}

// closeHID ends a driver and makes the removal of its pad reach the handler.
func (r *run) closeHID(hp *hidPad) {
	r.s.hidMu.Lock()
	delete(r.s.hid, hp.id)
	r.s.hidMu.Unlock()
	hp.dev.Close()
	hp.bridge.Remove()
}
