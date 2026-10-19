package platform

// The types below are placeholders for things that later steps define. They
// live here, in the internal package, so the public ui package gets no stub
// types. The step that defines the real type replaces its placeholder.

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

// Services groups the optional desktop capabilities. Each accessor returns nil
// when the capability is absent.
//
// TODO(steps 15.1-15.4): the accessors (tray, notifications, file chooser, open URI,
// accessibility bridge).
//
// Flutter: no counterpart
type Services interface{}
