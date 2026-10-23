//go:build linux && !android

package wayland

import (
	"cmp"
	"slices"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// output is one wl_output the compositor offers, with what it says about itself:
// the mode and the scale, the position and the physical size, the name, and, when
// the compositor has zxdg_output_manager_v1, the logical position and size (SPEC
// §8.2). A monitor that is plugged in later appears in the registry and is bound
// at once; one that is unplugged goes with wl_registry.global_remove.
type output struct {
	global  uint32 // the registry name, which is also the ID of the display
	version uint32
	wl      *wl.WlOutput
	xdg     *wl.ZxdgOutputV1

	name, description, make, model string

	x, y, mmWidth, mmHeight int32
	transform               wl.WlOutputTransform

	// The current mode, in pixels and millihertz.
	modeWidth, modeHeight, refresh int32
	scale                          int32

	// The logical position and size of zxdg_output_v1; hasLogical says that the
	// compositor sent them.
	logX, logY, logWidth, logHeight int32
	hasLogical                      bool

	// done says that the compositor sent all it had to say once.
	done bool
}

// newOutput binds a wl_output global and listens to it.
func (s *session) newOutput(g Global, max uint32) *output {
	o := &output{global: g.Name, version: bindVersion(g, max), scale: 1}
	o.wl = wl.BindWlOutput(s.registry, g.Name, o.version)
	s.outputScale[o.wl] = 1
	o.wl.SetEvents(&wl.WlOutputEvents{
		OnGeometry: func(x, y, mmW, mmH int32, _ wl.WlOutputSubpixel, make, model string, transform wl.WlOutputTransform) {
			o.x, o.y, o.mmWidth, o.mmHeight, o.make, o.model, o.transform = x, y, mmW, mmH, make, model, transform
		},
		OnMode: func(flags wl.WlOutputMode, width, height, refresh int32) {
			if flags&wl.WlOutputModeCurrent != 0 {
				o.modeWidth, o.modeHeight, o.refresh = width, height, refresh
			}
		},
		OnScale: func(factor int32) {
			o.scale = factor
			s.outputScale[o.wl] = factor
			if s.onOutputScale != nil {
				s.onOutputScale()
			}
			s.notifyOutputs()
		},
		OnName:        func(name string) { o.name = name },
		OnDescription: func(description string) { o.description = description },
		OnDone: func() {
			o.done = true
			s.notifyOutputs()
		},
	})
	s.outputs = append(s.outputs, o)
	if s.xdgOutputs != nil {
		s.attachXdgOutput(o)
	}
	return o
}

// attachXdgOutput asks for the logical geometry of the output.
func (s *session) attachXdgOutput(o *output) {
	if o.xdg != nil {
		return
	}
	o.xdg = s.xdgOutputs.GetXdgOutput(o.wl)
	o.xdg.SetEvents(&wl.ZxdgOutputV1Events{
		OnLogicalPosition: func(x, y int32) { o.logX, o.logY, o.hasLogical = x, y, true },
		OnLogicalSize:     func(w, h int32) { o.logWidth, o.logHeight, o.hasLogical = w, h, true },
		OnName: func(name string) {
			if o.name == "" {
				o.name = name
			}
		},
		OnDescription: func(description string) {
			if o.description == "" {
				o.description = description
			}
		},
		// Before version 3 the compositor says that it is done with this event;
		// from version 3 it does with wl_output.done.
		OnDone: func() {
			if o.xdg != nil && o.xdg.ProxyVersion() < 3 {
				o.done = true
				s.notifyOutputs()
			}
		},
	})
}

// removeOutput forgets the output of a registry name that went away, and reports
// whether there was one.
func (s *session) removeOutput(name uint32) bool {
	for i, o := range s.outputs {
		if o.global != name {
			continue
		}
		s.outputs = slices.Delete(s.outputs, i, i+1)
		delete(s.outputScale, o.wl)
		if o.xdg != nil {
			o.xdg.Destroy()
		}
		if o.version >= 3 {
			o.wl.Release()
		}
		if s.onOutputScale != nil {
			s.onOutputScale()
		}
		s.notifyOutputs()
		return true
	}
	return false
}

func (s *session) notifyOutputs() {
	if s.onOutputsChanged != nil {
		s.onOutputsChanged()
	}
}

// outputOf finds the output of a wl_output.
func (s *session) outputOf(w *wl.WlOutput) *output {
	for _, o := range s.outputs {
		if o.wl == w {
			return o
		}
	}
	return nil
}

// rotated says that the output shows its mode turned by a quarter, so that its width
// is the height of the mode.
func (o *output) rotated() bool {
	switch o.transform {
	case wl.WlOutputTransform90, wl.WlOutputTransform270, 5, 7: // 5 and 7 are flipped 90 and 270
		return true
	}
	return false
}

// pixelSize is the size of the output in pixels, as the user sees it.
func (o *output) pixelSize() (w, h int32) {
	if o.rotated() {
		return o.modeHeight, o.modeWidth
	}
	return o.modeWidth, o.modeHeight
}

// display describes the output for the engine: its place and size in logical
// pixels, its scale, and its refresh rate.
func (o *output) display() platform.Display {
	pw, ph := o.pixelSize()
	scale := float64(max(o.scale, 1))
	d := platform.Display{
		ID: int(o.global), Name: o.label(), DevicePixelRatio: scale,
		RefreshRate: float64(o.refresh) / 1000,
	}
	switch {
	case o.hasLogical && o.logWidth > 0 && o.logHeight > 0:
		d.Bounds = ui.NewRectFromLTWH(float64(o.logX), float64(o.logY), float64(o.logWidth), float64(o.logHeight))
		if pw > 0 {
			// A fractional scale shows up as the ratio of the mode to the logical size.
			d.DevicePixelRatio = float64(pw) / float64(o.logWidth)
		}
	default:
		d.Bounds = ui.NewRectFromLTWH(float64(o.x), float64(o.y), float64(pw)/scale, float64(ph)/scale)
	}
	return d
}

// label is the name of the output: the one it announced (wl_output v4 or
// zxdg_output_v1), else its description, else its make and model.
func (o *output) label() string {
	switch {
	case o.name != "":
		return o.name
	case o.description != "":
		return o.description
	case o.make != "" || o.model != "":
		return o.make + " " + o.model
	}
	return ""
}

// Displays lists the outputs that are connected, in the order they were announced.
// The primary one is the one at the origin of the logical space, or the first.
//
// Flutter: no counterpart
func (p *Platform) Displays() []platform.Display {
	var out []platform.Display
	for _, o := range p.sess.outputs {
		if o.done || o.hasMode() {
			out = append(out, o.display())
		}
	}
	primary := slices.IndexFunc(out, func(d platform.Display) bool { return d.Bounds.Left == 0 && d.Bounds.Top == 0 })
	if len(out) > 0 {
		out[max(primary, 0)].Primary = true
	}
	return out
}

func (o *output) hasMode() bool { return o.modeWidth > 0 && o.modeHeight > 0 }

// displayOf is the output the window is on: of those the surface has entered, the
// one with the largest scale (the one that decides how the window is drawn), the
// latest if there are several, or the only one there is when the compositor never
// said.
func (w *window) displayOf() (id int, ok bool) {
	var best *output
	for _, e := range w.entered {
		o := w.p.sess.outputOf(e)
		if o == nil {
			continue
		}
		if best == nil || cmp.Compare(o.scale, best.scale) >= 0 {
			best = o
		}
	}
	if best == nil {
		return 0, false
	}
	return int(best.global), true
}
