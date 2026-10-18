package ui

// ViewID identifies a View: one top-level surface that the engine renders
// into, such as a window. The first view is the implicit view and has ID 0.
//
// Flutter: ui/window.dart FlutterView.viewId
type ViewID int64

// ImplicitViewID is the ID of the implicit view, the first view that exists
// when the app starts.
//
// Flutter: ui/window.dart PlatformDispatcher.implicitView
const ImplicitViewID ViewID = 0
