package platform

import (
	"context"

	"github.com/rm4n0s/glutter/ui"
)

// Clipboard is the system clipboard of the desktop. Its methods are for the UI
// thread. Reading the clipboard of another application takes time (the other
// application has to write the data), so the results come back through a
// function that the platform calls later on the UI thread; it is never called
// before the method has returned.
//
// SetContent and GetContent reach the primary selection and the formats beyond
// text; the platform that has no primary selection fails them for it with
// ClipboardPrimaryUnavailable.
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
	// SetContent makes the content the content of the selection of the kind: the
	// system clipboard or the primary selection. done is as for SetText.
	SetContent(ctx context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(err error))
	// GetContent reads the selection of the kind in the format (one of the
	// ui.ClipboardFormat constants). The content is empty when the selection has
	// no such format. ctx ends a read that is taking too long.
	GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(content ui.ClipboardContent, err error))
}
