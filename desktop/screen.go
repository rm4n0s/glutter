package desktop

import (
	"github.com/rm4n0s/glutter/widgets"
)

// KeepScreenOn stops the screen from going idle (dimming, blanking and locking)
// while the window that the context belongs to is visible, or, with false, lets it
// go idle again. A video player or a presentation calls it with true while it plays
// and with false when it stops. The window gives the request up when it is closed.
//
// On Wayland it uses zwp_idle_inhibit_manager_v1. It fails with the tag
// IdleInhibitUnavailable when the compositor, the platform or the host cannot do
// it (the X11 backend cannot yet). It panics with ViewOfWithoutView, like
// widgets.ViewOf, when the context has no View above it.
//
// Flutter: no counterpart (Flutter apps use the wakelock_plus plugin)
func KeepScreenOn(bc widgets.BuildContext, on bool) error {
	return widgets.ViewOf(bc).KeepScreenOn(on)
}
