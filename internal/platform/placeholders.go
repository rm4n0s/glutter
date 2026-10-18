package platform

// The types below are placeholders for things that later steps define. They
// live here, in the internal package, so the public ui package gets no stub
// types. The step that defines the real type replaces its placeholder.

// MouseCursor is the cursor shown over a window.
//
// TODO(step 1.8): the cursor kinds (basic, click, text, grab, grabbing,
// resize*, wait, forbidden, none).
//
// Flutter: services/mouse_cursor.dart MouseCursor
type MouseCursor int

// TextInputState is what the IME needs to know about the focused text field.
//
// TODO(step 11.2): enabled flag, cursor rectangle, surrounding text, content
// type.
//
// Flutter: services/text_input.dart TextInputConfiguration
type TextInputState struct {
	// Enabled is whether a text field has focus.
	Enabled bool
}

// TextInputEvent is an event from the IME: preedit, commit or a deletion.
//
// TODO(step 11.2): the event fields.
//
// Flutter: services/text_input.dart TextInputClient
type TextInputEvent struct{}

// DragData is the payload of a drag-and-drop operation started by the app.
//
// TODO(step 14.6): the data formats.
//
// Flutter: no counterpart
type DragData struct{}

// DropEvent is a drag-and-drop event from another application.
//
// TODO(step 14.6): enter, move, leave and drop, with the dropped data.
//
// Flutter: no counterpart
type DropEvent struct{}

// Clipboard is the system clipboard.
//
// TODO(step 10.4): text access; TODO(step 11.6): primary selection and rich
// formats.
//
// Flutter: services/clipboard.dart Clipboard
type Clipboard interface{}

// Services groups the optional desktop capabilities. Each accessor returns nil
// when the capability is absent.
//
// TODO(steps 15.1-15.4): the accessors (tray, notifications, file chooser, open URI,
// accessibility bridge).
//
// Flutter: no counterpart
type Services interface{}
