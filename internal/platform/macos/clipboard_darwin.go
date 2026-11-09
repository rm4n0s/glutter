//go:build darwin && !ios

package macos

import (
	"context"
	"net/url"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// clipboard is the pasteboard of macOS (SPEC §29.7): NSPasteboard's general pasteboard for the system
// clipboard, with text, an image as PNG, file URLs and HTML. macOS has no primary selection. The calls are
// on the UI thread and do not wait for another app (the pasteboard serves its data in the process); the
// results go to done after the call has returned, as the interface says.
//
// Flutter: engine/src/flutter/shell/platform/darwin/macos/framework/Source/FlutterPlatformPlugin.mm (the Clipboard.getData and Clipboard.setData handlers)
type clipboard struct {
	p *Platform
	// pb is the pasteboard to use instead of the general one (the tests use one of their own, so that they
	// never touch the user's clipboard).
	pb objc.ID
	// named says that pb is a pasteboard of a name (DevPasteboardEnv), which the platform releases when it ends.
	named bool
	// text is the text that was read last and the change count of the pasteboard when it was: a read of a
	// pasteboard that did not change is not made again.
	text      string
	textOK    bool
	textCount int64
	textKnown bool
}

// The pasteboard types (UTIs) of the formats.
const (
	typeText    = "public.utf8-plain-text"
	typeHTML    = "public.html"
	typePNG     = "public.png"
	typeTIFF    = "public.tiff"
	typeFileURL = "public.file-url"
)

// bitmapImageFileTypePNG is NSBitmapImageFileTypePNG.
const bitmapImageFileTypePNG uint64 = 4

func (c *clipboard) board() objc.ID {
	if c.pb != 0 {
		return c.pb
	}
	return class("NSPasteboard").Send(sel("generalPasteboard"))
}

// post gives a result to its callback on the UI thread, after the call that made it has returned.
func (c *clipboard) post(f func()) {
	if c.p != nil {
		c.p.Post(f)
		return
	}
	f()
}

// SetText makes text the content of the system clipboard.
//
// Flutter: FlutterPlatformPlugin.mm Clipboard.setData
func (c *clipboard) SetText(ctx context.Context, text string, done func(err error)) {
	c.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{Text: text, HasText: true}, done)
}

// GetText reads the text of the system clipboard; it is not ok when there is none. The text is read again
// only when the pasteboard has changed.
//
// Flutter: FlutterPlatformPlugin.mm Clipboard.getData
func (c *clipboard) GetText(_ context.Context, done func(text string, ok bool, err error)) {
	if done == nil {
		return
	}
	var text string
	var ok bool
	objcx.WithPool(func() {
		pb := c.board()
		count := send[int64](pb, "changeCount")
		if !c.textKnown || count != c.textCount {
			c.text, c.textOK = c.readString(pb, typeText)
			c.textCount, c.textKnown = count, true
		}
		text, ok = c.text, c.textOK
	})
	c.post(func() { done(text, ok, nil) })
}

// SetContent clears the pasteboard and writes the content: one item that has the text, the HTML and the
// image, and one item for each file URL. An empty content only clears it. The primary selection does not
// exist on macOS: ClipboardPrimaryUnavailable.
func (c *clipboard) SetContent(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(err error)) {
	if kind == ui.ClipboardKindPrimary {
		c.answerErr(done, errors.New(tagClipboardPrimaryUnavailable, "macOS has no primary selection"))
		return
	}
	var err error
	objcx.WithPool(func() { err = c.write(content) })
	if done != nil {
		c.post(func() { done(err) })
	}
}

// write is SetContent on the UI thread.
func (c *clipboard) write(content ui.ClipboardContent) error {
	pb := c.board()
	pb.Send(sel("clearContents"))
	c.textKnown = false
	if content.IsEmpty() {
		return nil
	}
	var items []objc.ID
	add := func(item objc.ID) { items = append(items, item) }
	newItem := func() objc.ID { return class("NSPasteboardItem").Send(sel("alloc")).Send(sel("init")) }
	setString := func(item objc.ID, s, typ string) {
		ns, t := objcx.NewString(s), objcx.NewString(typ)
		item.Send(sel("setString:forType:"), ns, t)
		objcx.Release(ns)
		objcx.Release(t)
	}
	if content.HasText || content.HTML != "" || len(content.PNG) > 0 {
		item := newItem()
		if content.HasText {
			setString(item, content.Text, typeText)
		}
		if content.HTML != "" {
			setString(item, content.HTML, typeHTML)
		}
		if len(content.PNG) > 0 {
			data := newData(content.PNG)
			t := objcx.NewString(typePNG)
			item.Send(sel("setData:forType:"), data, t)
			objcx.Release(t)
			objcx.Release(data)
		}
		add(item)
	}
	for _, uri := range content.URIs {
		item := newItem()
		setString(item, uri, typeFileURL)
		add(item)
	}
	arr := objcx.NewArray(items...)
	ok := send[bool](pb, "writeObjects:", arr)
	objcx.Release(arr)
	for _, it := range items {
		objcx.Release(it)
	}
	if !ok {
		return errors.New(tagClipboardWriteFailed, "NSPasteboard refused the content")
	}
	return nil
}

// GetContent reads the format from the pasteboard: text and HTML as they are, an image as PNG (an image that
// an app copied as TIFF is converted), and the file URLs of the items. The content is empty when the
// pasteboard has no such format; the primary selection is ClipboardPrimaryUnavailable.
func (c *clipboard) GetContent(_ context.Context, kind ui.ClipboardKind, format string, done func(content ui.ClipboardContent, err error)) {
	if done == nil {
		return
	}
	if kind == ui.ClipboardKindPrimary {
		err := errors.New(tagClipboardPrimaryUnavailable, "macOS has no primary selection")
		c.post(func() { done(ui.ClipboardContent{}, err) })
		return
	}
	var out ui.ClipboardContent
	objcx.WithPool(func() {
		pb := c.board()
		switch format {
		case ui.ClipboardFormatHTML:
			out.HTML, _ = c.readString(pb, typeHTML)
		case ui.ClipboardFormatPNG:
			out.PNG = c.readPNG(pb)
		case ui.ClipboardFormatURIList:
			out.URIs = c.readURIs(pb)
		default:
			out.Text, out.HasText = c.readString(pb, typeText)
		}
	})
	c.post(func() { done(out, nil) })
}

// answerErr gives a failure to the callback after the call has returned.
func (c *clipboard) answerErr(done func(error), err error) {
	if done != nil {
		c.post(func() { done(err) })
	}
}

// readString is the string of the pasteboard in the type, and whether there is one.
func (c *clipboard) readString(pb objc.ID, typ string) (string, bool) {
	t := objcx.NewString(typ)
	defer objcx.Release(t)
	s := send[objc.ID](pb, "stringForType:", t)
	if s == 0 {
		return "", false
	}
	return objcx.String(s), true
}

// readPNG is the image of the pasteboard as PNG: the PNG data if there is, else the TIFF data converted.
func (c *clipboard) readPNG(pb objc.ID) []byte {
	t := objcx.NewString(typePNG)
	data := send[objc.ID](pb, "dataForType:", t)
	objcx.Release(t)
	if data != 0 {
		return dataBytes(data)
	}
	t = objcx.NewString(typeTIFF)
	tiff := send[objc.ID](pb, "dataForType:", t)
	objcx.Release(t)
	if tiff == 0 {
		return nil
	}
	rep := class("NSBitmapImageRep").Send(sel("alloc")).Send(sel("initWithData:"), tiff)
	if rep == 0 {
		return nil
	}
	defer objcx.Release(rep)
	props := class("NSDictionary").Send(sel("dictionary"))
	png := send[objc.ID](rep, "representationUsingType:properties:", bitmapImageFileTypePNG, props)
	if png == 0 {
		return nil
	}
	return dataBytes(png)
}

// readURIs are the file URLs of the items of the pasteboard.
func (c *clipboard) readURIs(pb objc.ID) []string {
	items := send[objc.ID](pb, "pasteboardItems")
	if items == 0 {
		return nil
	}
	t := objcx.NewString(typeFileURL)
	defer objcx.Release(t)
	var uris []string
	for i := 0; i < objcx.ArrayCount(items); i++ {
		s := send[objc.ID](objcx.ArrayAt(items, i), "stringForType:", t)
		if s == 0 {
			continue
		}
		if u, err := url.Parse(objcx.String(s)); err == nil && u.Scheme == "file" {
			uris = append(uris, u.String())
		}
	}
	return uris
}

// newData makes an NSData of the bytes (+1: release it).
func newData(b []byte) objc.ID {
	return class("NSData").Send(sel("alloc")).Send(sel("initWithBytes:length:"), unsafe.Pointer(unsafe.SliceData(b)), uint64(len(b)))
}

// dataBytes copies the bytes of an NSData.
func dataBytes(data objc.ID) []byte {
	n := int(send[uint64](data, "length"))
	if n == 0 {
		return nil
	}
	b := make([]byte, n)
	data.Send(sel("getBytes:length:"), unsafe.Pointer(unsafe.SliceData(b)), uint64(n))
	return b
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardPrimaryUnavailable = "ClipboardPrimaryUnavailable"
	tagClipboardWriteFailed        = "ClipboardWriteFailed"
)
