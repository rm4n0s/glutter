package platform

// The types below are placeholders for things that later steps define. They
// live here, in the internal package, so the public ui package gets no stub
// types. The step that defines the real type replaces its placeholder.

// Services groups the optional desktop capabilities. Each accessor returns nil
// when the capability is absent.
//
// TODO(steps 15.1-15.4): the accessors (tray, notifications, file chooser, open URI,
// accessibility bridge).
//
// Flutter: no counterpart
type Services interface{}
