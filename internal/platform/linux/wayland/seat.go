package wayland

import "github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"

// updateSeat creates the input devices that the seat has, and drops the ones it
// lost: a keyboard, a pointer and touch can come and go while the app runs.
func (p *Platform) updateSeat(caps wl.WlSeatCapability) {
	if has := caps&wl.WlSeatCapabilityKeyboard != 0; has && p.keyboard == nil {
		p.keyboard = newKeyboard(p)
	} else if !has && p.keyboard != nil {
		p.keyboard.close()
		p.keyboard = nil
	}
	if has := caps&wl.WlSeatCapabilityPointer != 0; has && p.pointer == nil {
		p.pointer = newPointer(p)
	} else if !has && p.pointer != nil {
		p.pointer.close()
		p.pointer = nil
	}
	if has := caps&wl.WlSeatCapabilityTouch != 0; has && p.touch == nil {
		p.touch = newTouch(p)
	} else if !has && p.touch != nil {
		p.touch.close()
		p.touch = nil
	}
}

// closeSeatDevices drops every input device.
func (p *Platform) closeSeatDevices() {
	p.updateSeat(0)
}
