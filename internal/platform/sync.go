package platform

import "github.com/rm4n0s/glutter/ui"

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
