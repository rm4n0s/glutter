package web

// IsNativePasteKey says whether the key event is the paste shortcut of the browser while the hidden text
// field has the focus: Control+V (also with Shift, which pastes plain text) and Shift+Insert, except on
// macOS and iOS, where the browser's shortcut is Command+V and Control+V pastes nothing. The event is not
// given to the framework, so that nothing handles the shortcut and the browser pastes into the hidden text
// field and sends its paste event, which needs no permission from the user (SPEC §23.5).
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.handleClipboardEvent
func IsNativePasteKey(e KeyEvent, os OperatingSystem) bool {
	if !e.Down || e.Alt || e.Meta || os == OSMacOS || os == OSIOS {
		return false
	}
	switch {
	case e.Ctrl:
		return e.Code == "KeyV" || e.Key == "v" || e.Key == "V"
	case e.Shift:
		return e.Code == "Insert" || e.Key == "Insert"
	}
	return false
}
