package platform

import (
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// SyncInputHandler is implemented by a Handler that can handle input at once and say what the
// framework made of it, for a platform whose events need the answer before the event handler
// returns: the web calls preventDefault on a key that the framework handled and on a wheel that
// a scrollable took (SPEC §23.3). The input that arrived earlier and is still waiting for its
// frame is handled first, so the order stays as it arrived.
//
// Flutter: no counterpart (the synchronous invokeOnKeyData and invokeOnPointerDataPacket of
// the web engine)
type SyncInputHandler interface {
	// OnKeySync handles the key event now and reports whether the framework handled it.
	OnKeySync(data ui.KeyData) bool
	// OnPointerSync handles the pointer data now. The OnRespond functions of the data
	// have been called, for the data that is answered, when it returns.
	OnPointerSync(data []ui.PointerData)
	// OnTextInputSync handles the text-input event now. A platform whose text field holds the
	// text (the web's) needs the framework's answer, the new state of the field, before the next
	// event of the field is read, so that it does not take its own echo for a change.
	OnTextInputSync(event ui.TextInputEvent)
}

// ContextMenuController is implemented by a Window that has a context menu of its own to show
// or to hide: the browser's, on the web (SPEC §23.4).
//
// Flutter: no counterpart
type ContextMenuController interface {
	// SetBrowserContextMenuEnabled says whether the browser's context menu is shown when the
	// user asks for it (a right click); it is enabled until the app disables it.
	SetBrowserContextMenuEnabled(enabled bool)
}

// SurfaceRenewer is implemented by a Window whose native surface can be replaced by a new one: the
// web's canvas keeps the first kind of context it was asked for, so a renderer that follows a
// failed one needs a canvas that has had none (SPEC §9.6). The window keeps its size, its input
// and its place; the new surface is blank until the next frame.
//
// Flutter: no counterpart
type SurfaceRenewer interface {
	// RenewSurface replaces the native surface.
	RenewSurface()
}

// SurfaceHandler is implemented by a Handler that is told when the native surface of a view goes
// away while the view stays: an Android activity gets a new Surface when the user comes back to
// the app (SPEC §24.4). The renderer lets go of what it made for the old surface; the next frame
// attaches the new one, which the platform asks for once there is one.
//
// Flutter: shell/platform/android/android_surface.h AndroidSurface (the surface is made again)
type SurfaceHandler interface {
	// OnSurfaceLost says that the native surface lost is no longer valid. It is called on the UI
	// thread, before the platform lets go of the native window, and the renderer must not use the
	// handle after it returns.
	OnSurfaceLost(view ui.ViewID, lost gpu.SurfaceHandle)
}
