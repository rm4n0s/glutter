package platform

import "context"

// Clipboard is the system clipboard of the desktop. Its methods are for the UI
// thread. Reading the clipboard of another application takes time (the other
// application has to write the data), so the results come back through a
// function that the platform calls later on the UI thread; it is never called
// before the method has returned.
//
// TODO(step 11.6): the primary selection and rich formats.
//
// Flutter: services/clipboard.dart Clipboard
type Clipboard interface {
	// SetText makes text the content of the clipboard; the app owns it until
	// another application takes it. done, when it is not nil, is called with the
	// error, if any.
	SetText(ctx context.Context, text string, done func(err error))
	// GetText reads the text of the clipboard. The text is empty and ok false when
	// the clipboard has no text. ctx ends a read that is taking too long.
	GetText(ctx context.Context, done func(text string, ok bool, err error))
}
