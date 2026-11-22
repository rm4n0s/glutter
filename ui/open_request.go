package ui

import "slices"

// OpenRequest is what the system asks the app to open (SPEC §8.7 "Open requests"): the URLs of a scheme the app
// handles (a click on a magnet link) and the files of a type it opens (a double click on a document, a file
// dropped on its icon). macOS sends them to the running app as Apple events, never as arguments of the program,
// also the ones that started it; the other platforms give them as arguments (os.Args) and send no request.
//
// Flutter: no counterpart (a deep link reaches Flutter as a route, widgets/binding.dart
// WidgetsBindingObserver.didPushRouteInformation; a file the app is asked to open does not reach it)
type OpenRequest struct {
	// URLs are the URLs to open, as the system gave them ("magnet:?xt=..."), without the file URLs.
	URLs []string
	// Files are the absolute paths of the files to open.
	Files []string
}

// IsEmpty reports whether the request names nothing to open.
//
// Flutter: no counterpart
func (r OpenRequest) IsEmpty() bool { return len(r.URLs) == 0 && len(r.Files) == 0 }

// openRequests is the state of the open requests of a dispatcher.
type openRequests struct {
	listeners []*openRequestListener
	// pending are the requests that came while nobody listened: the ones that started the app come before its
	// first widget is built.
	pending []OpenRequest
}

type openRequestListener struct{ fn func(OpenRequest) }

// AddOpenRequestListener calls fn for every request of the system to open URLs or files (OpenRequest), on the UI
// thread, and returns the function that removes it. The requests that came while there was no listener, such as
// the ones that started the app, are kept and given to the first listener when it is added, in their order, so
// an app that listens from the InitState of its first widget misses none.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) AddOpenRequestListener(fn func(request OpenRequest)) (remove func()) {
	l := &openRequestListener{fn: fn}
	d.open.listeners = append(d.open.listeners, l)
	pending := d.open.pending
	d.open.pending = nil
	for _, r := range pending {
		fn(r)
	}
	return func() {
		if i := slices.Index(d.open.listeners, l); i >= 0 {
			d.open.listeners = slices.Delete(d.open.listeners, i, i+1)
		}
	}
}

// DispatchOpenRequest passes a request of the system to open URLs or files to the listeners of
// AddOpenRequestListener, or keeps it for the first one when there is none. An empty request is dropped. Called
// by the engine.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) DispatchOpenRequest(request OpenRequest) {
	if request.IsEmpty() {
		return
	}
	request = OpenRequest{URLs: slices.Clone(request.URLs), Files: slices.Clone(request.Files)}
	if len(d.open.listeners) == 0 {
		d.open.pending = append(d.open.pending, request)
		return
	}
	for _, l := range slices.Clone(d.open.listeners) {
		if slices.Contains(d.open.listeners, l) { // not one that an earlier listener removed
			l.fn(OpenRequest{URLs: slices.Clone(request.URLs), Files: slices.Clone(request.Files)})
		}
	}
}
