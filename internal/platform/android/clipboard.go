package android

import (
	"context"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// clipboard is the clipboard of Android: the text of the ClipboardManager (SPEC §24.6). A write is
// a command to the activity; a read is a command and an event that brings the answer, which the UI
// thread gives to the function of the call. The formats are the text and the HTML (one ClipData made
// by ClipData.newHtmlText); another format reads as empty and cannot be written, and there is no
// primary selection. Every answer is given later on the UI thread, never
// before the call has returned.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.getClipboardData
type clipboard struct {
	p *Platform
	// reads are the reads that wait for the activity, by number; next is the number of the next.
	reads map[int]func(text string, ok bool, err error)
	next  int
}

var _ platform.Clipboard = (*clipboard)(nil)

// SetText puts the text on the clipboard.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.setClipboardData
func (c *clipboard) SetText(_ context.Context, text string, done func(error)) {
	c.p.host.Command(command{Op: opClipboardSet, S: text})
	c.answerErr(done, nil)
}

// GetText reads the text of the clipboard; the text is empty and ok false when it has none. A read
// that ctx ends before the activity answers fails with ClipboardReadTimeout, once.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.getClipboardData
func (c *clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	if done == nil {
		return
	}
	c.read(ctx, opClipboardGet, done)
}

// SetContent puts the text of the content on the clipboard, and its HTML next to it as one clip
// (ClipData.newHtmlText). Deviation: a clip with HTML always has a plain text, so a content with HTML
// and no text gets the text that Html.fromHtml makes of the HTML. The primary selection does not exist
// (ClipboardPrimaryUnavailable), and a content without text or HTML cannot be written
// (AndroidClipboardFormatUnsupported), except for an empty one, which clears the clipboard.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.setClipboardData
func (c *clipboard) SetContent(ctx context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	switch {
	case kind == ui.ClipboardKindPrimary:
		c.answerErr(done, errors.New(tagClipboardPrimaryUnavailable, "Android has no primary selection"))
	case !content.HasText && content.HTML == "" && !content.IsEmpty():
		c.answerErr(done, errors.New(tagAndroidClipboardFormatUnsupported, "the clipboard of Android takes text and HTML only"))
	case content.HTML != "":
		html := strings.ToValidUTF8(content.HTML, "\uFFFD")
		text := strings.ToValidUTF8(content.Text, "\uFFFD")
		hasText := int64(0)
		if content.HasText {
			hasText = 1
		}
		c.p.host.Command(command{Op: opClipboardSetHTML, A: int64(utf16Len(html)), B: hasText, S: html + text})
		c.answerErr(done, nil)
	default:
		c.SetText(ctx, content.Text, done)
	}
}

// GetContent reads the text of the clipboard when the format is plain text (or not given), and its
// HTML when the format is text/html. Another format reads as an empty content; the primary selection
// is ClipboardPrimaryUnavailable.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.getClipboardData
func (c *clipboard) GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	if done == nil {
		return
	}
	switch {
	case kind == ui.ClipboardKindPrimary:
		err := errors.New(tagClipboardPrimaryUnavailable, "Android has no primary selection")
		c.p.Post(func() { done(ui.ClipboardContent{}, err) })
	case format == ui.ClipboardFormatHTML:
		c.read(ctx, opClipboardGetHTML, func(html string, _ bool, err error) {
			if err != nil {
				done(ui.ClipboardContent{}, err)
				return
			}
			done(ui.ClipboardContent{HTML: html}, nil)
		})
	case format != "" && format != ui.ClipboardFormatText:
		c.p.Post(func() { done(ui.ClipboardContent{}, nil) })
	default:
		c.read(ctx, opClipboardGet, func(text string, _ bool, err error) {
			if err != nil {
				done(ui.ClipboardContent{}, err)
				return
			}
			done(ui.ClipboardContent{Text: text, HasText: text != ""}, nil)
		})
	}
}

// answerErr gives the result of a call to its function, later.
func (c *clipboard) answerErr(done func(error), err error) {
	if done != nil {
		c.p.Post(func() { done(err) })
	}
}

// read asks the activity for the text (op is opClipboardGet) or the HTML (opClipboardGetHTML) and waits
// for its answer, or for the end of ctx.
func (c *clipboard) read(ctx context.Context, op int, done func(text string, ok bool, err error)) {
	if ctx.Err() != nil {
		err := errors.NewErr(tagClipboardReadTimeout, ctx.Err())
		c.p.Post(func() { done("", false, err) })
		return
	}
	if c.reads == nil {
		c.reads = map[int]func(string, bool, error){}
	}
	c.next++
	id := c.next
	// The end of the context is told to the UI thread, which gives the answer, if the activity has not.
	stop := context.AfterFunc(ctx, func() {
		c.p.Post(func() {
			if fn, ok := c.reads[id]; ok {
				delete(c.reads, id)
				fn("", false, errors.NewErr(tagClipboardReadTimeout, ctx.Err()))
			}
		})
	})
	c.reads[id] = func(text string, ok bool, err error) {
		stop()
		done(text, ok, err)
	}
	c.p.host.Command(command{Op: op, A: int64(id)})
}

// answer is the activity's answer to a read (Native.clipboardText), on the UI thread: a read that
// the context ended is not answered again.
func (c *clipboard) answer(id int, ok bool, text string) {
	fn, found := c.reads[id]
	if !found {
		return
	}
	delete(c.reads, id)
	fn(text, ok && text != "", nil)
}

// Clipboard returns the clipboard of Android.
//
// Flutter: services/clipboard.dart Clipboard
func (p *Platform) Clipboard() platform.Clipboard {
	if p.clipboard == nil {
		p.clipboard = &clipboard{p: p}
	}
	return p.clipboard
}

// clipboardText is the activity's answer to a read of the clipboard (Native.clipboardText).
func (p *Platform) clipboardText(id int, ok bool, text string) {
	if p.clipboard != nil {
		p.clipboard.answer(id, ok, text)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardPrimaryUnavailable       = "ClipboardPrimaryUnavailable"
	tagClipboardReadTimeout              = "ClipboardReadTimeout"
	tagAndroidClipboardFormatUnsupported = "AndroidClipboardFormatUnsupported"
)
