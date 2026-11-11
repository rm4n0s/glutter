package web

// IsNativePasteKey says whether the key event is the paste shortcut of the browser while the hidden text
// field has the focus: Control+V (also with Shift, which pastes plain text) and Shift+Insert, and on macOS
// and iOS Command+V (also with Shift), where Control+V pastes nothing. The event is not given to the
// framework, so that nothing handles the shortcut and the browser pastes into the hidden text field and
// sends its paste event, which needs no permission from the user (SPEC §23.5). The framework's own paste
// (Command+V in its macOS bindings) would read the clipboard with navigator.clipboard, which Safari answers
// with a "Paste" button that the user has to press.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.handleClipboardEvent
func IsNativePasteKey(e KeyEvent, os OperatingSystem) bool {
	if !e.Down || e.Alt {
		return false
	}
	isV := e.Code == "KeyV" || e.Key == "v" || e.Key == "V"
	if os == OSMacOS || os == OSIOS {
		return e.Meta && !e.Ctrl && isV
	}
	if e.Meta {
		return false
	}
	switch {
	case e.Ctrl:
		return isV
	case e.Shift:
		return e.Code == "Insert" || e.Key == "Insert"
	}
	return false
}
