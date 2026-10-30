package web

import (
	"context"
	"sync/atomic"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// clipboardAPI is the part of the browser's Clipboard API (navigator.clipboard) that the clipboard
// uses. Both methods start the work and return; done is called once, later, from the browser's callback
// of the promise, on any goroutine. The code in clipboard_js.go is the real one, and the tests use a fake.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardStrategy
type clipboardAPI interface {
	// ReadText reads the text of the clipboard (navigator.clipboard.readText; the browser may ask the
	// user). It reports an empty text for a clipboard without text.
	ReadText(done func(text string, err error))
	// WriteText puts the text on the clipboard (navigator.clipboard.writeText).
	WriteText(text string, done func(err error))
	// ReadHTML reads the text/html of the clipboard (navigator.clipboard.read, where the browser
	// allows it). It reports "" for a clipboard without HTML and for a browser that cannot read items.
	ReadHTML(done func(html string, err error))
	// WriteRich puts the text (when hasText) and the HTML on the clipboard as one copy, in one
	// ClipboardItem (navigator.clipboard.write).
	WriteRich(text string, hasText bool, html string, done func(err error))
}

// clipboard is the clipboard of the web platform (SPEC §23.5): the text of the system clipboard through
// the asynchronous Clipboard API. Every answer is posted to the UI goroutine from the promise's callback,
// never given before the call has returned, and the UI goroutine never waits for a promise (SPEC §23.3).
// There is no primary selection; the formats are the text and the HTML (a ClipboardItem with both): an
// image or files cannot be written, and read as empty.
//
// A paste by the shortcut of the browser does not come here: the hidden text field gets the paste event
// and sends the text to the framework itself (textinput_js.go).
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardMessageHandler
type clipboard struct {
	post func(func())
	// api is nil when the page has no navigator.clipboard (an insecure context, an old browser).
	api clipboardAPI
}

var _ platform.Clipboard = clipboard{}

// SetText writes the text to the system clipboard. The write must happen in the event handler of a user
// action (a key press), which the framework's shortcuts keep to, because the browsers ask for that.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardMessageHandler.setDataMethodCall
func (c clipboard) SetText(ctx context.Context, text string, done func(error)) {
	c.write(ctx, text, done)
}

// GetText reads the text of the system clipboard through navigator.clipboard.readText. The text is empty and
// ok false when the clipboard has no text.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardMessageHandler.getDataMethodCall
func (c clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	c.read(ctx, startReadText, func(text string, err error) {
		if done != nil {
			done(text, err == nil && text != "", err)
		}
	})
}

// SetContent writes the text of the content to the system clipboard, and its HTML next to it, as one
// ClipboardItem. The primary selection does not exist on the web (ClipboardPrimaryUnavailable), and a
// content without text or HTML cannot be written (WebClipboardFormatUnsupported), except for an empty
// one, which clears the clipboard.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardMessageHandler.setDataMethodCall
func (c clipboard) SetContent(ctx context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	switch {
	case kind == ui.ClipboardKindPrimary:
		c.answerErr(done, errors.New(tagClipboardPrimaryUnavailable, "the web has no primary selection"))
	case !content.HasText && content.HTML == "" && !content.IsEmpty():
		c.answerErr(done, errors.New(tagWebClipboardFormatUnsupported, "the web clipboard takes text and HTML only"))
	case content.HTML != "":
		c.writeRich(ctx, content, done)
	default:
		c.write(ctx, content.Text, done)
	}
}

// GetContent reads the text of the system clipboard when the format is plain text (or not given), and
// its HTML when the format is text/html (navigator.clipboard.read: a browser that does not allow it
// gives an empty content, and one that asks the user may be refused, WebClipboardReadFailed). Another
// format reads as an empty content, as a clipboard that has no such format does; the primary selection
// is ClipboardPrimaryUnavailable.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardMessageHandler.getDataMethodCall
func (c clipboard) GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	if done == nil {
		return
	}
	switch {
	case kind == ui.ClipboardKindPrimary:
		err := errors.New(tagClipboardPrimaryUnavailable, "the web has no primary selection")
		c.post(func() { done(ui.ClipboardContent{}, err) })
	case format == ui.ClipboardFormatHTML:
		c.read(ctx, startReadHTML, func(html string, err error) {
			if err != nil {
				done(ui.ClipboardContent{}, err)
				return
			}
			done(ui.ClipboardContent{HTML: html}, nil)
		})
	case format != "" && format != ui.ClipboardFormatText:
		c.post(func() { done(ui.ClipboardContent{}, nil) })
	default:
		c.read(ctx, startReadText, func(text string, err error) {
			if err != nil {
				done(ui.ClipboardContent{}, err)
				return
			}
			done(ui.ClipboardContent{Text: text, HasText: text != ""}, nil)
		})
	}
}

// answerErr posts the failure of a call to its callback.
func (c clipboard) answerErr(done func(error), err error) {
	if done != nil {
		c.post(func() { done(err) })
	}
}

// write starts the write, and posts its result to done.
func (c clipboard) write(ctx context.Context, text string, done func(error)) {
	if c.api == nil {
		c.answerErr(done, errors.New(tagWebClipboardUnavailable,
			"the page has no navigator.clipboard: it needs a secure context (https or localhost)"))
		return
	}
	if err := ctx.Err(); err != nil {
		c.answerErr(done, errors.New(tagWebClipboardWriteFailed, "navigator.clipboard.writeText failed", "detail", err.Error()))
		return
	}
	c.api.WriteText(text, func(err error) { c.answerErr(done, err) })
}

// writeRich starts the write of a ClipboardItem with the HTML and the text, and posts its result to done.
func (c clipboard) writeRich(ctx context.Context, content ui.ClipboardContent, done func(error)) {
	if c.api == nil {
		c.answerErr(done, errors.New(tagWebClipboardUnavailable,
			"the page has no navigator.clipboard: it needs a secure context (https or localhost)"))
		return
	}
	if err := ctx.Err(); err != nil {
		c.answerErr(done, errors.New(tagWebClipboardWriteFailed, "navigator.clipboard.write failed", "detail", err.Error()))
		return
	}
	c.api.WriteRich(content.Text, content.HasText, content.HTML, func(err error) { c.answerErr(done, err) })
}

// startReadText starts the read of the text.
func startReadText(api clipboardAPI, done func(string, error)) { api.ReadText(done) }

// startReadHTML starts the read of the HTML.
func startReadHTML(api clipboardAPI, done func(string, error)) { api.ReadHTML(done) }

// read starts the read that start makes, and posts its result to done once: from the promise, or when
// ctx ends first.
func (c clipboard) read(ctx context.Context, start func(api clipboardAPI, done func(string, error)), done func(text string, err error)) {
	if c.api == nil {
		err := errors.New(tagWebClipboardUnavailable,
			"the page has no navigator.clipboard: it needs a secure context (https or localhost)")
		c.post(func() { done("", err) })
		return
	}
	if err := ctx.Err(); err != nil {
		timeout := errors.NewErr(tagClipboardReadTimeout, err)
		c.post(func() { done("", timeout) })
		return
	}
	var answered atomic.Bool
	finish := func(text string, err error) {
		if answered.CompareAndSwap(false, true) {
			c.post(func() { done(text, err) })
		}
	}
	stop := context.AfterFunc(ctx, func() { finish("", errors.NewErr(tagClipboardReadTimeout, ctx.Err())) })
	start(c.api, func(text string, err error) {
		stop()
		finish(text, err)
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardPrimaryUnavailable   = "ClipboardPrimaryUnavailable"
	tagClipboardReadTimeout          = "ClipboardReadTimeout"
	tagWebClipboardFormatUnsupported = "WebClipboardFormatUnsupported"
)

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebClipboardUnavailable = "WebClipboardUnavailable"
	tagWebClipboardWriteFailed = "WebClipboardWriteFailed"
	tagWebClipboardReadFailed  = "WebClipboardReadFailed"
)
