package x11

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// The monitors of the X screen, and the scale of each (SPEC §8.3, §8.11). XRandR
// says where the monitors are, how big they are on the desk and how fast they
// refresh. The scale is the one the desktop chose, Xft.dpi in RESOURCE_MANAGER
// divided by 96, for every monitor. Without it, a monitor of more than 144 dpi, by
// its size in millimetres, gets the scale that makes it about 96 dpi, in steps of a
// quarter; any other gets 1.

const (
	baseDPI         = 96.0
	hiDPIThreshold  = 144.0
	maxScale        = 4.0
	rrScreenChange  = 0        // RRScreenChangeNotify, relative to the event base of RandR
	rrScreenMask    = 1 << 0   // RRScreenChangeNotifyMask
	rrCrtcMask      = 1 << 1   // RRCrtcChangeNotifyMask
	rrOutputMask    = 1 << 2   // RROutputChangeNotifyMask
	rrNotify        = 1        // RRNotify, relative to the event base of RandR: a CRTC or an output changed
	rrMonitorActive = int32(1) // get_active of XRRGetMonitors: only the monitors that show something
	mmPerInch       = 25.4
)

// monitor is one rectangle of the screen, in pixels.
type monitor struct {
	// id identifies the monitor: the atom of its name.
	id      int
	name    string
	primary bool

	x, y, width, height int32
	mmWidth, mmHeight   int32
}

// parseXftDPI finds Xft.dpi in the text of RESOURCE_MANAGER (lines of "name:\tvalue").
func parseXftDPI(resources string) (dpi float64, ok bool) {
	for _, line := range strings.Split(resources, "\n") {
		name, value, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(name) != "Xft.dpi" {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

// scaleFor is the scale of a monitor: xftDPI is the Xft.dpi setting, 0 when there is
// none.
func scaleFor(xftDPI float64, m monitor) float64 {
	dpi := xftDPI
	if dpi <= 0 {
		if m.mmWidth <= 0 || m.width <= 0 {
			return 1
		}
		dpi = float64(m.width) * mmPerInch / float64(m.mmWidth)
		if dpi < hiDPIThreshold {
			return 1
		}
	}
	scale := math.Round(dpi/baseDPI*4) / 4 // steps of a quarter
	return min(max(scale, 1), maxScale)
}

// refreshMonitors reads the monitors and Xft.dpi again.
func (p *Platform) refreshMonitors() {
	p.xftDPI, _ = parseXftDPI(p.resourceManager())
	if p.xftDPI == 0 && p.xs != nil {
		// The desktop scales by a whole factor and says so over XSETTINGS.
		if scale := xsettingsScale(p.xs.settings); scale > 0 {
			p.xftDPI = baseDPI * scale
		}
	}
	p.monitors = p.monitors[:0]
	r, err := loadXrandrOnce()
	if err != nil || p.rrEventBase == 0 {
		return
	}
	count := dl.Calloc(1, 4)
	defer dl.Free(count)
	list := r.getMonitors(p.display, p.root, rrMonitorActive, count)
	if list == 0 {
		return
	}
	defer r.freeMonitors(list)
	n := int(*view[int32](count))
	for i := range n {
		info := view[xrrMonitorInfo](list + uintptr(i)*unsafe.Sizeof(xrrMonitorInfo{}))
		m := monitor{
			id: int(info.name), primary: info.primary != 0,
			x: info.x, y: info.y, width: info.width, height: info.height,
			mmWidth: info.mmWidth, mmHeight: info.mmHeight,
		}
		if name := p.x.getAtomName(p.display, info.name); name != 0 {
			m.name = dl.GoString(name)
			p.x.free(name)
		}
		p.monitors = append(p.monitors, m)
	}
	// A server without a primary monitor: the first one is.
	if len(p.monitors) > 0 && !slices.ContainsFunc(p.monitors, func(m monitor) bool { return m.primary }) {
		p.monitors[0].primary = true
	}
}

// resourceManager reads RESOURCE_MANAGER of the root window, the X resources that a
// settings daemon or xrdb put there. (XResourceManagerString returns the text of
// when the connection opened, so it does not see a change.)
func (p *Platform) resourceManager() string {
	return string(p.readProperty(p.root, p.atoms.resourceManager, xaString, 64*1024))
}

// selectMonitorEvents asks for the events that change the monitors or the scale:
// RandR's screen changes, and changes of RESOURCE_MANAGER on the root window.
func (p *Platform) selectMonitorEvents() {
	// StructureNotify brings the MANAGER messages that tell of a new settings daemon.
	p.x.selectInput(p.display, p.root, propertyChangeMask|structureNotifyMask)
	r, err := loadXrandrOnce()
	if err != nil {
		return
	}
	base := dl.Calloc(1, 8)
	defer dl.Free(base)
	if r.queryExtension(p.display, base, base+4) == 0 {
		return
	}
	p.rrEventBase = *view[int32](base)
	// A monitor that is plugged in or taken away changes an output and a CRTC, and
	// usually the size of the screen, which is the event that every client gets.
	r.selectInput(p.display, p.root, rrScreenMask|rrCrtcMask|rrOutputMask)
}

// monitorAt is the monitor that has the point, the primary one when none has.
func (p *Platform) monitorAt(x, y int32) (monitor, bool) {
	for _, m := range p.monitors {
		if x >= m.x && x < m.x+m.width && y >= m.y && y < m.y+m.height {
			return m, true
		}
	}
	for _, m := range p.monitors {
		if m.primary {
			return m, true
		}
	}
	if len(p.monitors) > 0 {
		return p.monitors[0], true
	}
	return monitor{}, false
}

// scaleAt is the scale of the monitor at a point of the screen.
func (p *Platform) scaleAt(x, y int32) (scale float64, id int, ok bool) {
	m, found := p.monitorAt(x, y)
	if !found {
		return scaleFor(p.xftDPI, monitor{}), 0, false
	}
	return scaleFor(p.xftDPI, m), m.id, true
}

// Displays lists the monitors, the primary one marked. A monitor is placed and sized
// in logical pixels, by its own scale.
//
// Flutter: no counterpart
func (p *Platform) Displays() []platform.Display {
	rate := float64(p.xrandrRate())
	if rate < 1 || rate > 1000 {
		rate = defaultRefreshRate
	}
	out := make([]platform.Display, 0, len(p.monitors))
	for _, m := range p.monitors {
		scale := scaleFor(p.xftDPI, m)
		out = append(out, platform.Display{
			ID: m.id, Name: m.name, Primary: m.primary, DevicePixelRatio: scale, RefreshRate: rate,
			Bounds: ui.NewRectFromLTWH(float64(m.x)/scale, float64(m.y)/scale, float64(m.width)/scale, float64(m.height)/scale),
		})
	}
	return out
}

// monitorsChanged follows a change of the screen or of Xft.dpi: the windows get the
// scale of the monitor they are on now, and the handler the new displays.
func (p *Platform) monitorsChanged() {
	before, dpi := slices.Clone(p.monitors), p.xftDPI
	p.refreshMonitors()
	for _, w := range p.windows {
		w.updateScale()
	}
	// One plugged in monitor makes several events (the output, the CRTC, the
	// screen); the app is told once, when the monitors or the scale are not the same.
	if p.handler != nil && (!slices.Equal(before, p.monitors) || dpi != p.xftDPI) {
		p.handler.OnDisplaysChanged()
	}
}

// px converts logical pixels of the window to device pixels.
func (w *window) px(v float64) int32 {
	if w.scale <= 0 {
		return int32(math.Round(v))
	}
	return int32(math.Round(v * w.scale))
}

// updateScale finds the monitor of the window and its scale. A new scale is
// reported at once, with a new frame; the window keeps its size in pixels, so its
// size in logical pixels changes.
func (w *window) updateScale() {
	if w.closed {
		return
	}
	x, y := w.rootOrigin()
	cx, cy := int32(x)+w.width/2, int32(y)+w.height/2
	scale, id, ok := w.p.scaleAt(cx, cy)
	if scale == w.scale && id == w.display && ok == w.hasDisplay {
		return
	}
	changed := scale != w.scale
	w.scale, w.display, w.hasDisplay = scale, id, ok
	if changed {
		w.p.cursors.apply(w, w.cursor) // the cursor has the size of the new scale
	}
	if w.mapped {
		w.reportMetrics()
		if changed {
			w.renderNow()
		}
	}
}
