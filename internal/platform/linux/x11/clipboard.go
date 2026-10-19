package x11

import (
	"context"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

const (
	// clipboardReadTimeout bounds a read of the selection: the owner has to answer,
	// and a stuck owner must not stop the app from pasting again.
	clipboardReadTimeout = 5 * time.Second
	// clipboardMaxBytes is the most text the app reads or sets.
	clipboardMaxBytes = 8 << 20
	// incrThreshold is the size above which the app sends the text in chunks (the
	// INCR protocol of ICCCM 2.7.2) and incrChunk the size of a chunk.
	incrThreshold = 256 << 10
	incrChunk     = 128 << 10
)

// clipboardAtoms are the atoms of the clipboard.
type clipboardAtoms struct {
	clipboard, targets, incr, property uint64
	textPlainUTF8, textPlain, text     uint64
}

// incrTransfer is a text that the app sends to a requestor in chunks.
type incrTransfer struct {
	requestor, property, target uint64
	data                        []byte
	offset                      int
}

// clipboardRequest is a read of the selection that waits for its answer.
type clipboardRequest struct {
	ctx      context.Context
	done     func(string, bool, error)
	targets  []uint64 // the formats still to try, the current one first
	incr     bool
	buf      []byte
	timer    *linux.Timer
	stopCtx  func() bool
	finished bool
}

// clipboard is the CLIPBOARD selection of X11 (ICCCM 2.6 and 2.7): the app owns
// it through a window of its own that has no size, answers the requests of
// other clients with the text, and reads the selection of another client
// through a property of that window. It belongs to the UI thread.
type clipboard struct {
	p      *Platform
	window uint64
	atoms  clipboardAtoms

	// text is what the app sends while it owns the selection.
	text  string
	owned bool

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
	return &clipboard{p: p, window: win, atoms: clipboardAtoms{
		clipboard: intern("CLIPBOARD"), targets: intern("TARGETS"), incr: intern("INCR"),
		property: intern("GLUTTER_CLIPBOARD"), textPlainUTF8: intern("text/plain;charset=utf-8"),
		textPlain: intern("text/plain"), text: intern("TEXT"),
	}}
}

func (c *clipboard) close() {
	for _, r := range append([]*clipboardRequest(nil), c.queue...) {
		c.finish(r, "", false, errors.New("ClipboardReadFailed", "the platform closed"))
	}
	c.p.x.destroyWindow(c.p.display, c.window)
	c.window = 0
}

// textTargets are the formats the app answers a request for the text in, besides
// TARGETS.
func (c *clipboard) textTargets() []uint64 {
	return []uint64{c.p.atoms.utf8String, c.atoms.textPlainUTF8, c.atoms.textPlain, c.atoms.text, xaString}
}

// SetText makes the app the owner of the CLIPBOARD selection.
func (c *clipboard) SetText(_ context.Context, text string, done func(error)) {
	var err error
	if len(text) > clipboardMaxBytes {
		err = errors.New("ClipboardTooLarge", "the text for the clipboard is too large", "limit", clipboardMaxBytes)
	} else {
		c.p.x.setSelectionOwner(c.p.display, c.atoms.clipboard, c.window, currentTime)
		if c.p.x.getSelectionOwner(c.p.display, c.atoms.clipboard) != c.window {
			err = errors.New("ClipboardWriteFailed", "the X server did not make the app the owner of the clipboard")
		} else {
			c.text, c.owned = text, true
		}
	}
	if done != nil {
		c.p.Post(func() { done(err) })
	}
}

// GetText reads the CLIPBOARD selection: the app's own text, or the text of the
// owner, converted to UTF8_STRING (or text/plain, or Latin-1 STRING).
func (c *clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	if c.owned {
		text := c.text
		c.p.Post(func() { done(text, true, nil) })
		return
	}
	if c.p.x.getSelectionOwner(c.p.display, c.atoms.clipboard) == 0 {
		c.p.Post(func() { done("", false, nil) })
		return
	}
	req := &clipboardRequest{ctx: ctx, done: done,
		targets: []uint64{c.p.atoms.utf8String, c.atoms.textPlainUTF8, xaString}}
	req.stopCtx = context.AfterFunc(ctx, func() {
		c.p.Post(func() { c.finish(req, "", false, errors.NewErr("ClipboardReadTimeout", ctx.Err())) })
	})
	timer, err := c.p.loop.AfterFunc(clipboardReadTimeout, func() {
		c.finish(req, "", false, errors.New("ClipboardReadTimeout", "the owner of the clipboard did not answer in time"))
	})
	if err != nil {
		req.stopCtx()
		c.p.Post(func() { done("", false, err) })
		return
	}
	req.timer = timer
	c.queue = append(c.queue, req)
	if len(c.queue) == 1 {
		c.convert(req)
	}
}

// convert asks the owner for the current format of the request.
func (c *clipboard) convert(req *clipboardRequest) {
	c.p.x.deleteProperty(c.p.display, c.window, c.atoms.property)
	c.p.x.convertSelection(c.p.display, c.atoms.clipboard, req.targets[0], c.atoms.property, c.window, currentTime)
	c.p.x.flush(c.p.display)
}

// finish ends a read, once, and starts the next one that waits.
func (c *clipboard) finish(req *clipboardRequest, text string, ok bool, err error) {
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
	req.done(text, ok, err)
	if wasCurrent && len(c.queue) > 0 {
		c.convert(c.queue[0])
	}
}

// onSelectionNotify is the answer of the owner to the conversion we asked for.
func (c *clipboard) onSelectionNotify(ev xSelectionEvent) {
	if len(c.queue) == 0 || ev.selection != c.atoms.clipboard {
		return
	}
	req := c.queue[0]
	if ev.property == xNone {
		// The owner cannot convert to this format: try the next one.
		req.targets = req.targets[1:]
		if len(req.targets) == 0 {
			c.finish(req, "", false, nil)
			return
		}
		c.convert(req)
		return
	}
	data, typ, ok := c.readProperty(true)
	if !ok {
		c.finish(req, "", false, errors.New("ClipboardReadFailed", "the answer of the owner could not be read"))
		return
	}
	if typ == c.atoms.incr {
		req.incr = true // the chunks come as property changes
		return
	}
	c.complete(req, data)
}

// onPropertyNotify gets a chunk of an incremental answer, or sends the next
// chunk of an incremental text.
func (c *clipboard) onPropertyNotify(ev xPropertyEvent) {
	if ev.state == propertyNewValue && ev.window == c.window && ev.atom == c.atoms.property &&
		len(c.queue) > 0 && c.queue[0].incr {
		req := c.queue[0]
		data, _, ok := c.readProperty(true)
		if !ok {
			c.finish(req, "", false, errors.New("ClipboardReadFailed", "a chunk of the answer of the owner could not be read"))
			return
		}
		if len(data) == 0 {
			c.complete(req, req.buf)
			return
		}
		if len(req.buf)+len(data) > clipboardMaxBytes {
			c.finish(req, "", false, errors.New("ClipboardTooLarge", "the text of the clipboard is too large", "limit", clipboardMaxBytes))
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

// complete turns the bytes of the answer into text and ends the read.
func (c *clipboard) complete(req *clipboardRequest, data []byte) {
	if len(data) > clipboardMaxBytes {
		c.finish(req, "", false, errors.New("ClipboardTooLarge", "the text of the clipboard is too large", "limit", clipboardMaxBytes))
		return
	}
	if req.targets[0] == xaString {
		// STRING is Latin-1: every byte is the code point.
		runes := make([]rune, len(data))
		for i, b := range data {
			runes[i] = rune(b)
		}
		c.finish(req, string(runes), true, nil)
		return
	}
	c.finish(req, string(data), true, nil)
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
// clipboard. The property of the answer is None when the request is refused.
func (c *clipboard) onSelectionRequest(ev xSelectionRequestEvent) {
	property := ev.property
	if property == xNone {
		property = ev.target // an obsolete client: ICCCM 2.2 says to use the target
	}
	var reply uint64 = xNone
	if ev.selection == c.atoms.clipboard && c.owned {
		switch {
		case ev.target == c.atoms.targets:
			targets := append([]uint64{c.atoms.targets}, c.textTargets()...)
			mem := dl.Calloc(uintptr(len(targets)), 8)
			copy(unsafe.Slice(view[uint64](mem), len(targets)), targets)
			c.p.x.changeProperty(c.p.display, ev.requestor, property, xaAtom, 32, propModeReplace, mem, int32(len(targets)))
			dl.Free(mem)
			reply = property
		case c.isTextTarget(ev.target):
			reply = c.sendText(ev, property)
		}
	}
	c.notify(ev, reply)
}

func (c *clipboard) isTextTarget(target uint64) bool {
	for _, t := range c.textTargets() {
		if t == target {
			return true
		}
	}
	return false
}

// sendText writes the text to the property of the requestor, in one piece or by
// starting an incremental transfer, and returns the property (None if refused).
func (c *clipboard) sendText(ev xSelectionRequestEvent, property uint64) uint64 {
	var data []byte
	typ := ev.target
	if ev.target == xaString {
		// Latin-1: characters that do not fit are replaced by '?'.
		for _, r := range c.text {
			if r > 0xff {
				r = '?'
			}
			data = append(data, byte(r))
		}
	} else {
		data = []byte(c.text)
	}
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
	if ev.selection == c.atoms.clipboard {
		c.owned, c.text = false, ""
	}
}
