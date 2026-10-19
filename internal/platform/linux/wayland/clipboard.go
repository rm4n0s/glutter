package wayland

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// The text formats of the clipboard. The app offers the first three, and reads
// the first of readMimeTypes that the owner of the selection offers. The legacy
// "STRING" and "TEXT" are left out of both: their encoding is not known.
var (
	offeredMimeTypes = []string{"text/plain;charset=utf-8", "text/plain", "UTF8_STRING"}
	readMimeTypes    = []string{"text/plain;charset=utf-8", "UTF8_STRING", "text/plain"}
)

const (
	// clipboardReadTimeout bounds a read of the selection: the owner has to write
	// the data, and a stuck owner must not stop the app from pasting again.
	clipboardReadTimeout = 5 * time.Second
	// clipboardMaxBytes is the most text the app reads from the selection.
	clipboardMaxBytes = 64 << 20
)

// dataOffer is a wl_data_offer with the formats it offers.
type dataOffer struct {
	obj   *wl.WlDataOffer
	mimes []string
}

func (o *dataOffer) offers(mime string) bool { return slices.Contains(o.mimes, mime) }

// clipboard is the system clipboard through wl_data_device (SPEC §8.11 "Data
// exchange"): it follows the selection that the compositor announces, reads it
// through a pipe on a goroutine, and owns the selection with a wl_data_source
// whose data it writes to the pipes it is given. It belongs to the UI thread;
// only the pipe reads and writes are on other goroutines.
type clipboard struct {
	p       *Platform
	mgr     *wl.WlDataDeviceManager
	device  *wl.WlDataDevice
	version uint32 // the version of the manager that was bound

	// pending are the offers the compositor introduced that no selection event has
	// claimed; selection is the current selection, or nil.
	pending   []*dataOffer
	selection *dataOffer

	// source is the data source the app owns the selection with, and text the text
	// it sends; owned is whether the compositor still has it as the selection.
	source *wl.WlDataSource
	text   string
	owned  bool
}

var _ platform.Clipboard = (*clipboard)(nil)

// newClipboard creates the data device of the seat. It returns nil when the
// compositor has no wl_data_device_manager.
func newClipboard(p *Platform) *clipboard {
	g, ok := p.sess.optionalPresent[wl.WlDataDeviceManagerInterface.Name]
	if !ok {
		return nil
	}
	c := &clipboard{p: p, version: bindVersion(g, 3)}
	c.mgr = wl.BindWlDataDeviceManager(p.sess.registry, g.Name, c.version)
	c.device = c.mgr.GetDataDevice(p.sess.seat)
	c.device.SetEvents(&wl.WlDataDeviceEvents{
		OnDataOffer: c.onDataOffer,
		OnSelection: c.onSelection,
		OnLeave:     c.onLeave,
	})
	return c
}

func (c *clipboard) onDataOffer(obj *wl.WlDataOffer) {
	offer := &dataOffer{obj: obj}
	obj.SetEvents(&wl.WlDataOfferEvents{OnOffer: func(mime string) { offer.mimes = append(offer.mimes, mime) }})
	c.pending = append(c.pending, offer)
}

// onSelection follows the selection: the offer is the new content of the
// clipboard, or nil when it is empty. The offer that is replaced is destroyed.
func (c *clipboard) onSelection(obj *wl.WlDataOffer) {
	var next *dataOffer
	for i, o := range c.pending {
		if o.obj == obj {
			next = o
			c.pending = slices.Delete(c.pending, i, i+1)
			break
		}
	}
	if c.selection != nil {
		c.selection.obj.Destroy()
	}
	c.selection = next
}

// onLeave ends a drag that went over the window: its offers are dropped (drag
// and drop itself is step 14.6).
func (c *clipboard) onLeave() {
	for _, o := range c.pending {
		o.obj.Destroy()
	}
	c.pending = nil
}

// close destroys the objects.
func (c *clipboard) close() {
	if c.source != nil {
		c.source.Destroy()
		c.source = nil
	}
	for _, o := range c.pending {
		o.obj.Destroy()
	}
	c.pending = nil
	if c.selection != nil {
		c.selection.obj.Destroy()
		c.selection = nil
	}
	// wl_data_device.release is since version 2. The manager has a destructor only
	// since version 4, which is more than this backend binds; it goes with the
	// connection.
	if c.device != nil && c.version >= 2 {
		c.device.Release()
	}
	c.device, c.mgr = nil, nil
}

// reply calls done on the UI thread, after the method that asked has returned.
func (c *clipboard) reply(fn func()) { c.p.Post(fn) }

// SetText makes the app the owner of the selection, with the text.
func (c *clipboard) SetText(_ context.Context, text string, done func(error)) {
	src := c.mgr.CreateDataSource()
	for _, mime := range offeredMimeTypes {
		src.Offer(mime)
	}
	src.SetEvents(&wl.WlDataSourceEvents{
		OnSend:      func(_ string, fd int) { writeClipboard(fd, text) },
		OnCancelled: func() { c.sourceCancelled(src) },
	})
	c.device.SetSelection(src, c.p.inputSerial)
	old := c.source
	c.source, c.text, c.owned = src, text, true
	if old != nil {
		old.Destroy()
	}
	if done != nil {
		c.reply(func() { done(nil) })
	}
}

// sourceCancelled is called when another client took the selection (or the
// compositor refused ours).
func (c *clipboard) sourceCancelled(src *wl.WlDataSource) {
	src.Destroy()
	if c.source == src {
		c.source, c.owned, c.text = nil, false, ""
	}
}

// writeClipboard writes the text to the pipe of a reader and closes it, on a
// goroutine: the reader may be slow, and the UI thread must not wait for it.
func writeClipboard(fd int, text string) {
	f := os.NewFile(uintptr(fd), "clipboard-send")
	go func() {
		_ = f.SetWriteDeadline(time.Now().Add(clipboardReadTimeout))
		if _, err := io.WriteString(f, text); err != nil {
			// The reader quit early or was too slow; nobody is left to tell.
			slog.Debug("the reader of the clipboard did not take all the text", "error", err)
		}
		if err := f.Close(); err != nil {
			slog.Debug("closing the pipe of the clipboard failed", "error", err)
		}
	}()
}

// GetText reads the selection. The text of the selection that the app owns comes
// back without a round trip.
func (c *clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	if c.owned {
		text := c.text
		c.reply(func() { done(text, true, nil) })
		return
	}
	var mime string
	if c.selection != nil {
		for _, m := range readMimeTypes {
			if c.selection.offers(m) {
				mime = m
				break
			}
		}
	}
	if mime == "" {
		c.reply(func() { done("", false, nil) })
		return
	}
	r, w, err := os.Pipe()
	if err != nil {
		err = errors.NewErr("ClipboardReadFailed", err)
		c.reply(func() { done("", false, err) })
		return
	}
	c.selection.obj.Receive(mime, int(w.Fd()))
	// The connection keeps its own copy of the descriptor until it has sent it.
	if cerr := w.Close(); cerr != nil {
		_ = r.Close()
		err = errors.NewErr("ClipboardReadFailed", cerr)
		c.reply(func() { done("", false, err) })
		return
	}
	go func() {
		text, rerr := readClipboard(ctx, r)
		c.p.Post(func() {
			if rerr != nil {
				done("", false, rerr)
				return
			}
			done(text, true, nil)
		})
	}()
}

// readClipboard reads the whole pipe, within the timeout and the limit, and
// closes it. Nothing here touches the connection.
func readClipboard(ctx context.Context, r *os.File) (string, error) {
	defer func() { _ = r.Close() }()
	ctx, cancel := context.WithTimeout(ctx, clipboardReadTimeout)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = r.SetReadDeadline(time.Now()) })
	defer stop()
	data, err := io.ReadAll(io.LimitReader(r, clipboardMaxBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return "", errors.NewErr("ClipboardReadTimeout", ctx.Err())
		}
		return "", errors.NewErr("ClipboardReadFailed", err)
	}
	if len(data) > clipboardMaxBytes {
		return "", errors.New("ClipboardTooLarge", "the text of the clipboard is too large", "limit", clipboardMaxBytes)
	}
	return string(data), nil
}
