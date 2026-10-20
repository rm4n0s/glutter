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

// inputScaleChanged tells the pointer that the window with its focus has another
// scale: the positions of the seat are in logical pixels and the framework wants
// physical ones.
func (p *Platform) inputScaleChanged(w *window) {
	if pt := p.pointer; pt != nil && pt.focus == w {
		pt.b.Scale = w.scaleFactor()
	}
	if t := p.tablet; t != nil {
		for _, tool := range t.tools {
			if tool.focus == w {
				t.b.Scale = w.scaleFactor()
			}
		}
	}
}
