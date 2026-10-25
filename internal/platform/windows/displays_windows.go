//go:build windows

package windows

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// Displays lists the monitors (SPEC §25.4): EnumDisplayMonitors and GetMonitorInfoW give the
// rectangle, the primary flag and the device name; the scale is the monitor's effective DPI over
// 96; the refresh rate is what EnumDisplaySettingsEx says for the mode in use (0 when it says 0
// or 1); the name is the monitor's own from EnumDisplayDevices, else the device's; the ID is the
// HMONITOR. The size is in logical pixels, and the position is the one of the virtual screen as
// Windows gives it, which the framework does not use.
//
// Flutter: shell/platform/windows/display_manager.cc DisplayManagerWin32::UpdateDisplays
func (p *Platform) Displays() []platform.Display {
	var out []platform.Display
	err := win32.EnumDisplayMonitors(func(monitor uintptr, _ win32.Rect) bool {
		mi, err := win32.GetMonitorInfo(monitor)
		if err != nil {
			p.log.Warn("a monitor could not be read", "err", err)
			return true
		}
		scale := scaleOfDPI(win32.GetDpiForMonitor(monitor))
		device := win32.UTF16ToString(mi.Device[:])
		name := device
		if dd, ok := win32.EnumDisplayDevices(device, 0); ok {
			if s := win32.UTF16ToString(dd.DeviceString[:]); s != "" {
				name = s
			}
		}
		var dm win32.DevMode
		rate := 0.0
		if win32.EnumDisplaySettingsEx(device, win32.EnumCurrentSettings, &dm) && dm.DisplayFrequency > 1 {
			rate = float64(dm.DisplayFrequency)
		}
		r := mi.Monitor
		out = append(out, platform.Display{
			ID: int(monitor), Name: name,
			Bounds: ui.Rect{
				Left: float64(r.Left), Top: float64(r.Top),
				Right: float64(r.Left) + float64(r.Right-r.Left)/scale, Bottom: float64(r.Top) + float64(r.Bottom-r.Top)/scale,
			},
			DevicePixelRatio: scale, RefreshRate: rate, Primary: mi.Flags&win32.MonitorInfoFPrimary != 0,
		})
		return true
	})
	if err != nil {
		p.log.Warn("the monitors could not be listed", "err", err)
	}
	return out
}

// primaryRefreshRate is the refresh rate of the primary monitor, 60 when the system does not
// say; it paces the frames when DwmFlush cannot.
func (p *Platform) primaryRefreshRate() float64 {
	for _, d := range p.Displays() {
		if d.Primary && d.RefreshRate > 1 {
			return d.RefreshRate
		}
	}
	return defaultRefreshRate
}

// displaysChanged tells the handler that the monitors changed (WM_DISPLAYCHANGE, WM_DPICHANGED),
// and follows the refresh rate of the primary one.
func (p *Platform) displaysChanged() {
	rate := p.primaryRefreshRate()
	if rate != p.refreshRate && p.vsync != nil {
		p.refreshRate = rate
		p.vsync.setRefreshRate(rate)
	}
	if p.handler != nil {
		p.handler.OnDisplaysChanged()
	}
}
