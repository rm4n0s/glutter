package ios

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the clipboard of iOS (SPEC §30.6) without a call into UIKit: what an item of the pasteboard is made of for the content
// the app copies, what a read of each format asks of the pasteboard, the paste prompt's denial, and the asynchronous reads. The
// pasteboard of UIKit (pasteboard_ios.go) is behind pasteBackend, so the logic is tested on every host.

// The pasteboard types (UTIs) of the formats.
const (
	typeText = "public.utf8-plain-text"
	typeHTML = "public.html"
	typePNG  = "public.png"
	typeURL  = "public.url"
)

// pasteValueKind says how a value is given to UIKit: as an NSString, as NSData or as an NSURL.
type pasteValueKind int

// The kinds of value of a pasteboard item.
const (
	pasteString pasteValueKind = iota
	pasteData
	pasteURL
)

// pasteValue is the value of one type in an item of the pasteboard.
type pasteValue struct {
	UTI  string
	Kind pasteValueKind
	Text string
	Data []byte
}

// pasteItem is one item of the pasteboard: a value for each of its types.
type pasteItem []pasteValue

// pasteItemsOf is the items that stand for the content: one item has the text, the HTML and the image, and each URI is an item of its
// own (as the pasteboard has the URLs of a drag). Content with no format gives no item, which empties the pasteboard.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm setClipboardData
func pasteItemsOf(c ui.ClipboardContent) []pasteItem {
	var items []pasteItem
	var main pasteItem
	if c.HasText {
		main = append(main, pasteValue{UTI: typeText, Kind: pasteString, Text: c.Text})
	}
	if c.HTML != "" {
		main = append(main, pasteValue{UTI: typeHTML, Kind: pasteString, Text: c.HTML})
	}
	if len(c.PNG) > 0 {
		main = append(main, pasteValue{UTI: typePNG, Kind: pasteData, Data: c.PNG})
	}
	if len(main) > 0 {
		items = append(items, main)
	}
	for _, uri := range c.URIs {
		items = append(items, pasteItem{{UTI: typeURL, Kind: pasteURL, Text: uri}})
	}
	return items
}

// pasteBackend is the pasteboard of the system. The reads may be called from any goroutine (a read of what another app copied waits for
// the paste prompt, which the UI thread never does); Write is for the UI thread.
type pasteBackend interface {
	// HasStrings says that the pasteboard has text; it never asks the user.
	HasStrings() bool
	// String is the text of the pasteboard. It asks the user when another app copied it (iOS 16 and later), and is not ok when there is
	// none or when the user did not allow the paste.
	String() (text string, ok bool)
	// HTML is the HTML of the pasteboard, PNG its image as PNG (an image of another type is converted), and URLs its URLs.
	HTML() (html string, ok bool)
	PNG() []byte
	URLs() []string
	// Write replaces the content of the pasteboard with the items; it is false when the pasteboard refuses.
	Write(items []pasteItem) bool
}

// pasteboard is the clipboard of the app: platform.Clipboard over UIKit's general pasteboard.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm (the Clipboard.getData and Clipboard.setData handlers)
type pasteboard struct {
	// post runs a function on the UI thread after the call that made it has returned; back is the pasteboard.
	post func(func())
	back pasteBackend
}

var _ platform.Clipboard = pasteboard{}

// SetText makes text the content of the pasteboard.
//
// Flutter: FlutterPlatformPlugin.mm Clipboard.setData
func (c pasteboard) SetText(ctx context.Context, text string, done func(err error)) {
	c.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{Text: text, HasText: true}, done)
}

// GetText reads the text of the pasteboard; it is not ok when there is none. A paste that the user did not allow is PasteDenied.
//
// Flutter: FlutterPlatformPlugin.mm Clipboard.getData
func (c pasteboard) GetText(ctx context.Context, done func(text string, ok bool, err error)) {
	if done == nil {
		return
	}
	c.GetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardFormatText, func(content ui.ClipboardContent, err error) {
		done(content.Text, content.HasText, err)
	})
}

// SetContent makes the content the content of the pasteboard. iOS has no primary selection: that kind is
// ClipboardPrimaryUnavailable. A pasteboard that refuses the items is IOSClipboardUnavailable.
//
// Flutter: FlutterPlatformPlugin.mm Clipboard.setData
func (c pasteboard) SetContent(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(err error)) {
	var err error
	switch {
	case kind == ui.ClipboardKindPrimary:
		err = errors.New(tagClipboardPrimaryUnavailable, "iOS has no primary selection")
	case !c.back.Write(pasteItemsOf(content)):
		err = errors.New(tagIOSClipboardUnavailable, "the pasteboard refused the content")
	}
	if done != nil {
		c.post(func() { done(err) })
	}
}

// GetContent reads the format from the pasteboard, on a goroutine of its own: a read of what another app copied waits for the user to
// answer the paste prompt, and the UI thread never waits for it (SPEC §30.6). done is called on the UI thread. ctx ends a wait that
// takes too long: the result is then IOSClipboardUnavailable, and the answer that comes later is dropped. The content is empty when
// the pasteboard has no such format, and a paste that the user did not allow is PasteDenied.
//
// Flutter: FlutterPlatformPlugin.mm Clipboard.getData
func (c pasteboard) GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(content ui.ClipboardContent, err error)) {
	if done == nil {
		return
	}
	if kind == ui.ClipboardKindPrimary {
		err := errors.New(tagClipboardPrimaryUnavailable, "iOS has no primary selection")
		c.post(func() { done(ui.ClipboardContent{}, err) })
		return
	}
	ch := make(chan pasteResult, 1)
	go func() {
		content, err := readPasteboard(c.back, format)
		ch <- pasteResult{content, err}
	}()
	go c.await(ctx, ch, done)
}

// pasteResult is what a read of the pasteboard gave.
type pasteResult struct {
	content ui.ClipboardContent
	err     error
}

// await waits for the read, or for ctx, and gives the outcome to done on the UI thread.
func (c pasteboard) await(ctx context.Context, ch <-chan pasteResult, done func(content ui.ClipboardContent, err error)) {
	select {
	case r := <-ch:
		c.post(func() { done(r.content, r.err) })
	case <-ctx.Done():
		err := errors.NewErr(tagIOSClipboardUnavailable, ctx.Err()).SetMetadata("reason", "the context ended before the paste prompt was answered")
		c.post(func() { done(ui.ClipboardContent{}, err) })
	}
}

// readPasteboard reads the format from the backend. Text that the pasteboard has but would not give is a paste that the user refused
// (the prompt of iOS 16 and later): it is PasteDenied, not an empty clipboard.
func readPasteboard(b pasteBackend, format string) (ui.ClipboardContent, error) {
	var out ui.ClipboardContent
	switch format {
	case ui.ClipboardFormatHTML:
		out.HTML, _ = b.HTML()
	case ui.ClipboardFormatPNG:
		out.PNG = b.PNG()
	case ui.ClipboardFormatURIList:
		out.URIs = b.URLs()
	default:
		if !b.HasStrings() {
			return out, nil
		}
		text, ok := b.String()
		if !ok {
			return out, errors.New(tagPasteDenied, "the user did not allow the paste")
		}
		out.Text, out.HasText = text, true
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardPrimaryUnavailable = "ClipboardPrimaryUnavailable"
	tagIOSClipboardUnavailable     = "IOSClipboardUnavailable"
	tagPasteDenied                 = "PasteDenied"
)
