package services

import (
	"context"

	"github.com/rm4n0s/errors"
)

// ClipboardKTextPlain is the format of plain text, for Clipboard.GetData.
//
// Flutter: services/clipboard.dart Clipboard.kTextPlain
const ClipboardKTextPlain = "text/plain"

// ClipboardData is data on the clipboard. Plain text is the only kind.
//
// Flutter: services/clipboard.dart ClipboardData
type ClipboardData struct {
	// Text is the text.
	Text string
}

// NewClipboardData makes the data of a text.
//
// Flutter: services/clipboard.dart ClipboardData.ClipboardData
func NewClipboardData(text string) ClipboardData { return ClipboardData{Text: text} }

// ClipboardBackend is where a Clipboard gets and puts text: the platform
// dispatcher (ui.PlatformDispatcher) is one. The functions it takes are called
// later on the UI thread, never before the method has returned.
//
// Flutter: no counterpart (the flutter/platform channel)
type ClipboardBackend interface {
	// SetClipboardText puts text on the clipboard.
	SetClipboardText(ctx context.Context, text string, done func(err error))
	// GetClipboardText reads the text of the clipboard; ok is false when there is
	// none.
	GetClipboardText(ctx context.Context, done func(text string, ok bool, err error))
}

// Clipboard is the system clipboard. Reading it can take time (another
// application has to hand the data over), so each method takes a function that
// is called later, on the UI thread, with the result; ctx ends a read that takes
// too long. The binding has one.
//
// Deviation: Flutter's Clipboard has static methods that return a Future, and
// that call the platform through a channel; here the methods take a callback and
// go through the engine (SPEC §12.10).
//
// Flutter: services/clipboard.dart Clipboard
type Clipboard struct {
	backend ClipboardBackend
}

// NewClipboard makes a clipboard on a backend.
//
// Flutter: no counterpart
func NewClipboard(backend ClipboardBackend) *Clipboard { return &Clipboard{backend: backend} }

// SetData stores the data on the clipboard. done, when it is not nil, is called
// with the error if any. A clipboard without a backend calls it at once with
// ClipboardUnavailable.
//
// Flutter: services/clipboard.dart Clipboard.setData
func (c *Clipboard) SetData(ctx context.Context, data ClipboardData, done func(err error)) {
	if c == nil || c.backend == nil {
		if done != nil {
			done(errClipboardUnavailable())
		}
		return
	}
	c.backend.SetClipboardText(ctx, data.Text, done)
}

// GetData retrieves the data of the format from the clipboard. Only
// ClipboardKTextPlain is supported: for another format, data is nil at once. The
// data is also nil when the clipboard has no text. A clipboard without a backend
// calls done at once with ClipboardUnavailable.
//
// Flutter: services/clipboard.dart Clipboard.getData
func (c *Clipboard) GetData(ctx context.Context, format string, done func(data *ClipboardData, err error)) {
	if c == nil || c.backend == nil {
		done(nil, errClipboardUnavailable())
		return
	}
	if format != ClipboardKTextPlain {
		done(nil, nil)
		return
	}
	c.backend.GetClipboardText(ctx, func(text string, ok bool, err error) {
		if err != nil || !ok {
			done(nil, err)
			return
		}
		done(&ClipboardData{Text: text}, nil)
	})
}

// HasStrings reports whether the clipboard has text that can be pasted; empty
// text does not count.
//
// Flutter: services/clipboard.dart Clipboard.hasStrings
func (c *Clipboard) HasStrings(ctx context.Context, done func(has bool, err error)) {
	c.GetData(ctx, ClipboardKTextPlain, func(data *ClipboardData, err error) {
		done(data != nil && data.Text != "", err)
	})
}

func errClipboardUnavailable() error {
	return errors.New("ClipboardUnavailable", "the clipboard has no backend")
}
