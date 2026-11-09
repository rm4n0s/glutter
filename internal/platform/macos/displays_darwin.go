//go:build darwin && !ios

package macos

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// Displays lists NSScreen.screens (SPEC §29.4): the frames are flipped to the top-left origin of the
// primary display, in points; the scale is the backing scale factor and the refresh rate is
// maximumFramesPerSecond. The first screen is the primary one, the one with the menu bar.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterEngine.mm (the displays of the engine)
func (p *Platform) Displays() []platform.Display {
	var out []platform.Display
	objcx.WithPool(func() {
		screens := class("NSScreen").Send(sel("screens"))
		n := objcx.ArrayCount(screens)
		if n == 0 {
			return
		}
		primary := send[nsRect](objcx.ArrayAt(screens, 0), "frame")
		for i := 0; i < n; i++ {
			s := objcx.ArrayAt(screens, i)
			fr := send[nsRect](s, "frame")
			d := platform.Display{
				ID:               screenID(s),
				Bounds:           flipRect(cocoaRect{X: fr.Origin.X, Y: fr.Origin.Y, W: fr.Size.Width, H: fr.Size.Height}, primary.Size.Height),
				DevicePixelRatio: send[float64](s, "backingScaleFactor"),
				RefreshRate:      float64(send[int64](s, "maximumFramesPerSecond")),
				Primary:          i == 0,
			}
			if name := send[objc.ID](s, "localizedName"); name != 0 {
				d.Name = objcx.String(name)
			}
			out = append(out, d)
		}
	})
	return out
}

// screenID is the display ID of a screen (CGDirectDisplayID), the value of "NSScreenNumber" in its device
// description: it is what ViewMetrics.Display and Display.ID use.
func screenID(screen objc.ID) int {
	desc := send[objc.ID](screen, "deviceDescription")
	key := objcx.NewString("NSScreenNumber")
	defer objcx.Release(key)
	num := desc.Send(sel("objectForKey:"), key)
	if num == 0 {
		return 0
	}
	return int(send[uint64](num, "unsignedIntValue"))
}

// displaysChanged is NSApplicationDidChangeScreenParametersNotification: a display was connected or
// disconnected, or changed its size, scale or refresh rate.
func (p *Platform) displaysChanged() {
	if p.handler != nil {
		p.handler.OnDisplaysChanged()
	}
}
