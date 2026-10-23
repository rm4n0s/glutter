//go:build linux && !android

package x11

import (
	"context"
	"strings"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

const (
	// clipboardReadTimeout bounds a read of the selection: the owner has to answer,
	// and a stuck owner must not stop the app from pasting again.
	clipboardReadTimeout = 5 * time.Second
	// clipboardMaxBytes is the most data (text, an image, a list of URIs) the app
	// reads or sets.
	clipboardMaxBytes = 64 << 20
	// incrThreshold is the size above which the app sends the text in chunks (the
	// INCR protocol of ICCCM 2.7.2) and incrChunk the size of a chunk.
	incrThreshold = 256 << 10
	incrChunk     = 128 << 10
)

// clipboardAtoms are the atoms of the clipboard.
type clipboardAtoms struct {
	clipboard, primary, targets, incr, property uint64
	textPlainUTF8, textPlain, text              uint64
	imagePNG, uriList                           uint64
}

// xSelection is one of the selections of X11 and what the app sends while it
// owns it.
type xSelection struct {
	atom    uint64
	content ui.ClipboardContent
	owned   bool
}

// incrTransfer is a text that the app sends to a requestor in chunks.
type incrTransfer struct {
	requestor, property, target uint64
	data                        []byte
	offset                      int
}

// clipboardRequest is a read of the selection that waits for its answer.
type clipboardRequest struct {
	ctx       context.Context
	done      func(ui.ClipboardContent, error)
	selection uint64   // the selection that is read
	format    string   // the format of the content that is read
	targets   []uint64 // the formats still to try, the current one first
	incr      bool
	buf       []byte
	timer     *linux.Timer
	stopCtx   func() bool
	finished  bool
}

// clipboard is the CLIPBOARD selection of X11 (ICCCM 2.6 and 2.7): the app owns
// it through a window of its own that has no size, answers the requests of
// other clients with the text, and reads the selection of another client
// through a property of that window. It belongs to the UI thread.
type clipboard struct {
	p      *Platform
	window uint64
	atoms  clipboardAtoms

	// system is the CLIPBOARD selection and primary the PRIMARY one.
	system, primary xSelection

	transfers []*incrTransfer
	queue     []*clipboardRequest // the reads that wait; the first is in progress
}

var _ platform.Clipboard = (*clipboard)(nil)

// newClipboard creates the window that owns the selection. It returns nil, with
// a log, when the server refuses.
func newClipboard(p *Platform) *clipboard {
	attrs := dl.Calloc(1, 112)
	defer dl.Free(attrs)
	view[xSetWindowAttributes](attrs).eventMask = propertyChangeMask
	win := p.x.createWindow(p.display, p.root, 0, 0, 1, 1, 0, 0, inputOnly, 0, cwEventMask, attrs)
	if win == 0 {
		return nil
	}
	intern := func(name string) uint64 {
		c := dl.CString(name)
		defer dl.Free(c)
		return p.x.internAtom(p.display, c, 0)
	}
	atoms := clipboardAtoms{
		clipboard: intern("CLIPBOARD"), primary: xaPrimary, targets: intern("TARGETS"), incr: intern("INCR"),
		property: intern("GLUTTER_CLIPBOARD"), textPlainUTF8: intern("text/plain;charset=utf-8"),
		textPlain: intern("text/plain"), text: intern("TEXT"), imagePNG: intern("image/png"), uriList: intern("text/uri-list"),
	}
	return &clipboard{p: p, window: win, atoms: atoms, system: xSelection{atom: atoms.clipboard}, primary: xSelection{atom: atoms.primary}}
}

func (c *clipboard) close() {
	for _, r := range append([]*clipboardRequest(nil), c.queue...) {
		c.finish(r, ui.ClipboardContent{}, errors.New(tagClipboardReadFailed, "the platform closed"))
	}
	c.p.x.destroyWindow(c.p.display, c.window)
	c.window = 0
}

// textTargets are the formats the app answers a request for the text in, besides
// TARGETS.
func (c *clipboard) textTargets() []uint64 {
	return []uint64{c.p.atoms.utf8String, c.atoms.textPlainUTF8, c.atoms.textPlain, c.atoms.text, xaString}
}

// selectionOf is the selection of the kind.
func (c *clipboard) selectionOf(kind ui.ClipboardKind) *xSelection {
	if kind == ui.ClipboardKindPrimary {
		return &c.primary
	}
	return &c.system
}

// selectionByAtom is the selection with the atom, or nil.
func (c *clipboard) selectionByAtom(atom uint64) *xSelection {
	switch atom {
	case c.atoms.clipboard:
		return &c.system
	case c.atoms.primary:
		return &c.primary
	}
	return nil
}

// SetText makes the app the owner of the CLIPBOARD selection.
func (c *clipboard) SetText(ctx context.Context, text string, done func(error)) {
	c.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{Text: text, HasText: true}, done)
}

// GetText reads the CLIPBOARD selection: the app's own text, or the text of the
// owner, converted to UTF8_STRING (or text/plain, or Latin-1 STRING).
func (c *clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	c.GetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardFormatText, func(content ui.ClipboardContent, err error) {
		done(content.Text, content.HasText, err)
	})
}

// contentSize is the number of bytes the content takes.
func contentSize(content ui.ClipboardContent) int {
	n := len(content.Text) + len(content.PNG)
	for _, u := range content.URIs {
		n += len(u) + 2
	}
	return n
}

// SetContent makes the app the owner of the selection of the kind.
func (c *clipboard) SetContent(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	var err error
	sel := c.selectionOf(kind)
	if contentSize(content) > clipboardMaxBytes {
		err = errors.New(tagClipboardTooLarge, "the content for the clipboard is too large", "limit", clipboardMaxBytes)
	} else {
		c.p.x.setSelectionOwner(c.p.display, sel.atom, c.window, currentTime)
		if c.p.x.getSelectionOwner(c.p.display, sel.atom) != c.window {
			err = errors.New(tagClipboardWriteFailed, "the X server did not make the app the owner of the clipboard")
		} else {
			sel.content, sel.owned = content, true
		}
	}
	if done != nil {
		c.p.Post(func() { done(err) })
	}
}

// targetsOf are the formats of X11 that the app reads for a format of the
// content, the one to try first first.
func (c *clipboard) targetsOf(format string) []uint64 {
	switch format {
	case ui.ClipboardFormatText:
		return []uint64{c.p.atoms.utf8String, c.atoms.textPlainUTF8, xaString}
	case ui.ClipboardFormatPNG:
		return []uint64{c.atoms.imagePNG}
	case ui.ClipboardFormatURIList:
		return []uint64{c.atoms.uriList}
	}
	return nil
}

// GetContent reads the selection of the kind in the format: what the app owns, or
// what the owner converts to it.
func (c *clipboard) GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	sel := c.selectionOf(kind)
	targets := c.targetsOf(format)
	if sel.owned {
		out := contentIn(sel.content, format)
		c.p.Post(func() { done(out, nil) })
		return
	}
	if len(targets) == 0 || c.p.x.getSelectionOwner(c.p.display, sel.atom) == 0 {
		c.p.Post(func() { done(ui.ClipboardContent{}, nil) })
		return
	}
	req := &clipboardRequest{ctx: ctx, done: done, selection: sel.atom, format: format, targets: targets}
	req.stopCtx = context.AfterFunc(ctx, func() {
		c.p.Post(func() { c.finish(req, ui.ClipboardContent{}, errors.NewErr(tagClipboardReadTimeout, ctx.Err())) })
	})
	timer, err := c.p.loop.AfterFunc(clipboardReadTimeout, func() {
		c.finish(req, ui.ClipboardContent{}, errors.New(tagClipboardReadTimeout, "the owner of the clipboard did not answer in time"))
	})
	if err != nil {
		req.stopCtx()
		c.p.Post(func() { done(ui.ClipboardContent{}, err) })
		return
	}
	req.timer = timer
	c.queue = append(c.queue, req)
	if len(c.queue) == 1 {
		c.convert(req)
	}
}

// contentIn is the part of the content in the format.
func contentIn(content ui.ClipboardContent, format string) ui.ClipboardContent {
	switch format {
	case ui.ClipboardFormatText:
		return ui.ClipboardContent{Text: content.Text, HasText: content.HasText}
	case ui.ClipboardFormatPNG:
		return ui.ClipboardContent{PNG: content.PNG}
	case ui.ClipboardFormatURIList:
		return ui.ClipboardContent{URIs: content.URIs}
	}
	return ui.ClipboardContent{}
}

// parseURIList reads a text/uri-list (RFC 2483): the lines that are not empty or
// comments.
func parseURIList(data []byte) []string {
	var uris []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		uris = append(uris, line)
	}
	return uris
}

// formatURIList is the text/uri-list form: one URI per line, each ended by CRLF.
func formatURIList(uris []string) string {
	var b strings.Builder
	for _, u := range uris {
		b.WriteString(u)
		b.WriteString("\r\n")
	}
	return b.String()
}

// convert asks the owner for the current format of the request.
func (c *clipboard) convert(req *clipboardRequest) {
	c.p.x.deleteProperty(c.p.display, c.window, c.atoms.property)
	c.p.x.convertSelection(c.p.display, req.selection, req.targets[0], c.atoms.property, c.window, currentTime)
	c.p.x.flush(c.p.display)
}

// finish ends a read, once, and starts the next one that waits.
func (c *clipboard) finish(req *clipboardRequest, content ui.ClipboardContent, err error) {
	if req.finished {
		return
	}
	req.finished = true
	if req.timer != nil {
		req.timer.Stop()
	}
	if req.stopCtx != nil {
		req.stopCtx()
	}
	wasCurrent := len(c.queue) > 0 && c.queue[0] == req
	for i, r := range c.queue {
		if r == req {
			c.queue = append(c.queue[:i], c.queue[i+1:]...)
			break
		}
	}
	req.done(content, err)
	if wasCurrent && len(c.queue) > 0 {
		c.convert(c.queue[0])
	}
}

// onSelectionNotify is the answer of the owner to the conversion we asked for.
func (c *clipboard) onSelectionNotify(ev xSelectionEvent) {
	if len(c.queue) == 0 || ev.selection != c.queue[0].selection {
		return
	}
	req := c.queue[0]
	if ev.property == xNone {
		// The owner cannot convert to this format: try the next one.
		req.targets = req.targets[1:]
		if len(req.targets) == 0 {
			c.finish(req, ui.ClipboardContent{}, nil)
			return
		}
		c.convert(req)
		return
	}
	data, typ, ok := c.readProperty(true)
	if !ok {
		c.finish(req, ui.ClipboardContent{}, errors.New(tagClipboardReadFailed, "the answer of the owner could not be read"))
		return
	}
	if typ == c.atoms.incr {
		req.incr = true // the chunks come as property changes
		return
	}
	c.complete(req, data)
}

// onPropertyNotify gets a chunk of an incremental answer, or sends the next
// chunk of an incremental transfer.
func (c *clipboard) onPropertyNotify(ev xPropertyEvent) {
	if ev.state == propertyNewValue && ev.window == c.window && ev.atom == c.atoms.property &&
		len(c.queue) > 0 && c.queue[0].incr {
		req := c.queue[0]
		data, _, ok := c.readProperty(true)
		if !ok {
			c.finish(req, ui.ClipboardContent{}, errors.New(tagClipboardReadFailed, "a chunk of the answer of the owner could not be read"))
			return
		}
		if len(data) == 0 {
			c.complete(req, req.buf)
			return
		}
		if len(req.buf)+len(data) > clipboardMaxBytes {
			c.finish(req, ui.ClipboardContent{}, errors.New(tagClipboardTooLarge, "the content of the clipboard is too large", "limit", clipboardMaxBytes))
			return
		}
		req.buf = append(req.buf, data...)
		return
	}
	if ev.state == propertyDelete {
		for _, t := range c.transfers {
			if t.requestor == ev.window && t.property == ev.atom {
				c.sendChunk(t)
				return
			}
		}
	}
}

// complete turns the bytes of the answer into the content and ends the read.
func (c *clipboard) complete(req *clipboardRequest, data []byte) {
	if len(data) > clipboardMaxBytes {
		c.finish(req, ui.ClipboardContent{}, errors.New(tagClipboardTooLarge, "the content of the clipboard is too large", "limit", clipboardMaxBytes))
		return
	}
	switch req.format {
	case ui.ClipboardFormatPNG:
		c.finish(req, ui.ClipboardContent{PNG: data}, nil)
	case ui.ClipboardFormatURIList:
		c.finish(req, ui.ClipboardContent{URIs: parseURIList(data)}, nil)
	default:
		if req.targets[0] == xaString {
			// STRING is Latin-1: every byte is the code point.
			runes := make([]rune, len(data))
			for i, b := range data {
				runes[i] = rune(b)
			}
			c.finish(req, ui.ClipboardContent{Text: string(runes), HasText: true}, nil)
			return
		}
		c.finish(req, ui.ClipboardContent{Text: string(data), HasText: true}, nil)
	}
}

// readProperty reads, and deletes, the property of our window that the owner
// wrote. The data is copied out of the memory of Xlib.
func (c *clipboard) readProperty(del bool) (data []byte, typ uint64, ok bool) {
	mem := dl.Calloc(4, 8)
	defer dl.Free(mem)
	actualType, actualFormat, nitems, bytesAfter := mem, mem+8, mem+16, mem+24
	var prop uintptr
	propPtr := dl.Calloc(1, 8)
	defer dl.Free(propPtr)
	d := int32(0)
	if del {
		d = 1
	}
	status := c.p.x.getWindowProperty(c.p.display, c.window, c.atoms.property, 0, clipboardMaxBytes/4+1, d, anyPropertyType,
		actualType, actualFormat, nitems, bytesAfter, propPtr)
	if status != 0 {
		return nil, 0, false
	}
	prop = *view[uintptr](propPtr)
	typ = *view[uint64](actualType)
	format := *view[int32](actualFormat)
	count := *view[uint64](nitems)
	more := *view[uint64](bytesAfter)
	if prop != 0 {
		defer c.p.x.free(prop)
	}
	if more > 0 {
		return nil, typ, false // larger than the limit
	}
	if typ == c.atoms.incr {
		return nil, typ, true
	}
	if format != 8 && count > 0 {
		return nil, typ, false
	}
	if prop != 0 && count > 0 {
		data = append([]byte(nil), dl.Bytes(prop, int(count))...)
	}
	return data, typ, true
}

// onSelectionRequest answers a client that asks the app, the owner, for the
// clipboard or the primary selection. The property of the answer is None when the
// request is refused.
func (c *clipboard) onSelectionRequest(ev xSelectionRequestEvent) {
	property := ev.property
	if property == xNone {
		property = ev.target // an obsolete client: ICCCM 2.2 says to use the target
	}
	var reply uint64 = xNone
	if sel := c.selectionByAtom(ev.selection); sel != nil && sel.owned {
		if ev.target == c.atoms.targets {
			targets := []uint64{c.atoms.targets}
			if sel.content.HasText {
				targets = append(targets, c.textTargets()...)
			}
			if len(sel.content.PNG) > 0 {
				targets = append(targets, c.atoms.imagePNG)
			}
			if len(sel.content.URIs) > 0 {
				targets = append(targets, c.atoms.uriList)
			}
			mem := dl.Calloc(uintptr(len(targets)), 8)
			copy(unsafe.Slice(view[uint64](mem), len(targets)), targets)
			c.p.x.changeProperty(c.p.display, ev.requestor, property, xaAtom, 32, propModeReplace, mem, int32(len(targets)))
			dl.Free(mem)
			reply = property
		} else if data, ok := c.dataFor(sel.content, ev.target); ok {
			reply = c.sendData(ev, property, data)
		}
	}
	c.notify(ev, reply)
}

// dataFor is what the app writes for a requested target; ok is false when the
// content has nothing for it.
func (c *clipboard) dataFor(content ui.ClipboardContent, target uint64) (data []byte, ok bool) {
	switch {
	case c.isTextTarget(target):
		if !content.HasText {
			return nil, false
		}
		if target == xaString {
			// Latin-1: characters that do not fit are replaced by '?'.
			for _, r := range content.Text {
				if r > 0xff {
					r = '?'
				}
				data = append(data, byte(r))
			}
			return data, true
		}
		return []byte(content.Text), true
	case target == c.atoms.imagePNG:
		return content.PNG, len(content.PNG) > 0
	case target == c.atoms.uriList:
		return []byte(formatURIList(content.URIs)), len(content.URIs) > 0
	}
	return nil, false
}

func (c *clipboard) isTextTarget(target uint64) bool {
	for _, t := range c.textTargets() {
		if t == target {
			return true
		}
	}
	return false
}

// sendData writes the data to the property of the requestor, in one piece or by
// starting an incremental transfer, and returns the property.
func (c *clipboard) sendData(ev xSelectionRequestEvent, property uint64, data []byte) uint64 {
	typ := ev.target
	if len(data) > incrThreshold {
		// The requestor gets the size, and then the chunks as it deletes the property.
		c.p.x.selectInput(c.p.display, ev.requestor, propertyChangeMask)
		size := dl.Calloc(1, 8)
		*view[int64](size) = int64(len(data))
		c.p.x.changeProperty(c.p.display, ev.requestor, property, c.atoms.incr, 32, propModeReplace, size, 1)
		dl.Free(size)
		c.transfers = append(c.transfers, &incrTransfer{requestor: ev.requestor, property: property, target: typ, data: data})
		return property
	}
	var mem uintptr
	if len(data) > 0 {
		mem = dl.Malloc(uintptr(len(data)))
		copy(dl.Bytes(mem, len(data)), data)
		defer dl.Free(mem)
	}
	c.p.x.changeProperty(c.p.display, ev.requestor, property, typ, 8, propModeReplace, mem, int32(len(data)))
	return property
}

// sendChunk writes the next chunk of an incremental text, or the empty chunk that
// ends it.
func (c *clipboard) sendChunk(t *incrTransfer) {
	end := min(t.offset+incrChunk, len(t.data))
	chunk := t.data[t.offset:end]
	t.offset = end
	var mem uintptr
	if len(chunk) > 0 {
		mem = dl.Malloc(uintptr(len(chunk)))
		copy(dl.Bytes(mem, len(chunk)), chunk)
	}
	c.p.x.changeProperty(c.p.display, t.requestor, t.property, t.target, 8, propModeReplace, mem, int32(len(chunk)))
	if mem != 0 {
		dl.Free(mem)
	}
	c.p.x.flush(c.p.display)
	if len(chunk) == 0 {
		for i, o := range c.transfers {
			if o == t {
				c.transfers = append(c.transfers[:i], c.transfers[i+1:]...)
				break
			}
		}
	}
}

// notify sends the SelectionNotify event that ends a request.
func (c *clipboard) notify(req xSelectionRequestEvent, property uint64) {
	mem := dl.Calloc(1, xEventSize)
	defer dl.Free(mem)
	*view[xSelectionEvent](mem) = xSelectionEvent{
		typ: selectionNotify, requestor: req.requestor, selection: req.selection,
		target: req.target, property: property, time: req.time,
	}
	c.p.x.sendEvent(c.p.display, req.requestor, 0, 0, mem)
	c.p.x.flush(c.p.display)
}

// onSelectionClear is called when another client took the selection.
func (c *clipboard) onSelectionClear(ev xSelectionClearEvent) {
	if sel := c.selectionByAtom(ev.selection); sel != nil {
		sel.owned, sel.content = false, ui.ClipboardContent{}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardReadFailed  = "ClipboardReadFailed"
	tagClipboardReadTimeout = "ClipboardReadTimeout"
	tagClipboardTooLarge    = "ClipboardTooLarge"
	tagClipboardWriteFailed = "ClipboardWriteFailed"
)
