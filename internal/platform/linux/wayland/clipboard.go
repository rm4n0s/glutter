package wayland

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
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

// The other formats. The app offers each when the content has it.
const (
	mimePNG     = "image/png"
	mimeURIList = "text/uri-list"
)

// readMimesOf are the formats of the selection that the app reads for a format
// of the content, in order of preference.
func readMimesOf(format string) []string {
	switch format {
	case ui.ClipboardFormatText:
		return readMimeTypes
	case ui.ClipboardFormatPNG:
		return []string{mimePNG}
	case ui.ClipboardFormatURIList:
		return []string{mimeURIList}
	}
	return nil
}

// offeredMimesOf are the formats the app offers for the content.
func offeredMimesOf(c ui.ClipboardContent) []string {
	var mimes []string
	if c.HasText {
		mimes = append(mimes, offeredMimeTypes...)
	}
	if len(c.PNG) > 0 {
		mimes = append(mimes, mimePNG)
	}
	if len(c.URIs) > 0 {
		mimes = append(mimes, mimeURIList)
	}
	return mimes
}

// dataOf is what the app writes to the pipe of a reader of the mime type; ok is
// false when the content has nothing for it.
func dataOf(c ui.ClipboardContent, mime string) (data []byte, ok bool) {
	switch {
	case slices.Contains(offeredMimeTypes, mime):
		return []byte(c.Text), c.HasText
	case mime == mimePNG:
		return c.PNG, len(c.PNG) > 0
	case mime == mimeURIList:
		return []byte(formatURIList(c.URIs)), len(c.URIs) > 0
	}
	return nil, false
}

// formatURIList is the text/uri-list form (RFC 2483): one URI per line, each
// ended by CRLF.
func formatURIList(uris []string) string {
	var b strings.Builder
	for _, u := range uris {
		b.WriteString(u)
		b.WriteString("\r\n")
	}
	return b.String()
}

// parseURIList reads a text/uri-list: the lines that are not empty or comments.
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

// contentOf makes the content of the bytes of a format.
func contentOf(format string, data []byte) ui.ClipboardContent {
	switch format {
	case ui.ClipboardFormatText:
		return ui.ClipboardContent{Text: string(data), HasText: true}
	case ui.ClipboardFormatPNG:
		return ui.ClipboardContent{PNG: data}
	case ui.ClipboardFormatURIList:
		return ui.ClipboardContent{URIs: parseURIList(data)}
	}
	return ui.ClipboardContent{}
}

// offer is a data offer (wl_data_offer or zwp_primary_selection_offer_v1) with
// the formats it offers.
type offer struct {
	mimes   []string
	receive func(mime string, fd int)
	destroy func()
}

func (o *offer) offers(mime string) bool { return slices.Contains(o.mimes, mime) }

// source is the selection source the app owns a selection with.
type source struct {
	destroy func()
}

// selection is one selection of the seat, the clipboard or the primary
// selection: it follows the offer that the compositor announces, and owns the
// selection with a source whose data it writes to the pipes it is given. The two
// selections differ only in the Wayland objects, which the functions create.
type selection struct {
	c *clipboard

	// pending are the offers the compositor introduced that no selection event has
	// claimed; current is the current selection, or nil.
	pending []*offer
	current *offer

	// newSource creates a source with the formats, and the function that makes it
	// the selection (with the serial of an input event).
	newSource func(mimes []string, send func(mime string, fd int), cancelled func()) (set func(serial uint32), destroy func())
	// closeDevice releases the data device.
	closeDevice func()

	src     *source
	content ui.ClipboardContent
	owned   bool
}

func (s *selection) onOffer(o *offer) { s.pending = append(s.pending, o) }

// onSelection follows the selection: the offer is the new content of the
// selection, or nil when it is empty. The offer that is replaced is destroyed.
func (s *selection) onSelection(o *offer) {
	var next *offer
	for i, p := range s.pending {
		if p == o {
			next = p
			s.pending = slices.Delete(s.pending, i, i+1)
			break
		}
	}
	if s.current != nil {
		s.current.destroy()
	}
	s.current = next
}

// dropPending destroys the offers of a drag that went over the window.
func (s *selection) dropPending() {
	for _, o := range s.pending {
		o.destroy()
	}
	s.pending = nil
}

func (s *selection) close() {
	if s.src != nil {
		s.src.destroy()
		s.src = nil
	}
	s.dropPending()
	if s.current != nil {
		s.current.destroy()
		s.current = nil
	}
	if s.closeDevice != nil {
		s.closeDevice()
	}
}

// set makes the app the owner of the selection, with the content.
func (s *selection) set(content ui.ClipboardContent, done func(error)) {
	var src *source
	install, destroy := s.newSource(offeredMimesOf(content),
		func(mime string, fd int) {
			if data, ok := dataOf(content, mime); ok {
				writeClipboard(fd, data)
				return
			}
			closeFD(fd)
		},
		func() { s.sourceCancelled(src) })
	src = &source{destroy: destroy}
	install(s.c.p.inputSerial)
	old := s.src
	s.src, s.content, s.owned = src, content, true
	if old != nil {
		old.destroy()
	}
	if done != nil {
		s.c.reply(func() { done(nil) })
	}
}

// sourceCancelled is called when another client took the selection (or the
// compositor refused ours).
func (s *selection) sourceCancelled(src *source) {
	src.destroy()
	if s.src == src {
		s.src, s.owned, s.content = nil, false, ui.ClipboardContent{}
	}
}

// get reads the selection in the format. What the app owns comes back without a
// round trip.
func (s *selection) get(ctx context.Context, format string, done func(ui.ClipboardContent, error)) {
	c := s.c
	if s.owned {
		out := contentIn(s.content, format)
		c.reply(func() { done(out, nil) })
		return
	}
	var mime string
	if s.current != nil {
		for _, m := range readMimesOf(format) {
			if s.current.offers(m) {
				mime = m
				break
			}
		}
	}
	if mime == "" {
		c.reply(func() { done(ui.ClipboardContent{}, nil) })
		return
	}
	r, w, err := os.Pipe()
	if err != nil {
		err = errors.NewErr("ClipboardReadFailed", err)
		c.reply(func() { done(ui.ClipboardContent{}, err) })
		return
	}
	s.current.receive(mime, int(w.Fd()))
	// The connection keeps its own copy of the descriptor until it has sent it.
	if cerr := w.Close(); cerr != nil {
		_ = r.Close()
		err = errors.NewErr("ClipboardReadFailed", cerr)
		c.reply(func() { done(ui.ClipboardContent{}, err) })
		return
	}
	go func() {
		data, rerr := readClipboard(ctx, r)
		c.p.Post(func() {
			if rerr != nil {
				done(ui.ClipboardContent{}, rerr)
				return
			}
			done(contentOf(format, data), nil)
		})
	}()
}

// contentIn is the part of the content in the format.
func contentIn(c ui.ClipboardContent, format string) ui.ClipboardContent {
	switch format {
	case ui.ClipboardFormatText:
		return ui.ClipboardContent{Text: c.Text, HasText: c.HasText}
	case ui.ClipboardFormatPNG:
		return ui.ClipboardContent{PNG: c.PNG}
	case ui.ClipboardFormatURIList:
		return ui.ClipboardContent{URIs: c.URIs}
	}
	return ui.ClipboardContent{}
}

// clipboard is the clipboard and the primary selection through wl_data_device
// and zwp_primary_selection_device_v1 (SPEC §8.11 "Data exchange"). It belongs
// to the UI thread; only the pipe reads and writes are on other goroutines.
type clipboard struct {
	p *Platform

	mgr     *wl.WlDataDeviceManager
	device  *wl.WlDataDevice
	version uint32 // the version of the manager that was bound

	pmgr   *wl.ZwpPrimarySelectionDeviceManagerV1
	pdev   *wl.ZwpPrimarySelectionDeviceV1
	system *selection
	// primary is nil when the compositor has no primary selection manager.
	primary *selection

	// dataOffers and primaryOffers find the offer of a Wayland object that a
	// selection event names.
	dataOffers    map[*wl.WlDataOffer]*offer
	primaryOffers map[*wl.ZwpPrimarySelectionOfferV1]*offer
}

var _ platform.Clipboard = (*clipboard)(nil)

// newClipboard creates the data device of the seat. It returns nil when the
// compositor has no wl_data_device_manager.
func newClipboard(p *Platform) *clipboard {
	g, ok := p.sess.optionalPresent[wl.WlDataDeviceManagerInterface.Name]
	if !ok {
		return nil
	}
	c := &clipboard{p: p, version: bindVersion(g, 3), dataOffers: map[*wl.WlDataOffer]*offer{}, primaryOffers: map[*wl.ZwpPrimarySelectionOfferV1]*offer{}}
	c.mgr = wl.BindWlDataDeviceManager(p.sess.registry, g.Name, c.version)
	c.device = c.mgr.GetDataDevice(p.sess.seat)
	c.system = &selection{c: c}
	c.system.newSource = func(mimes []string, send func(string, int), cancelled func()) (func(uint32), func()) {
		src := c.mgr.CreateDataSource()
		for _, mime := range mimes {
			src.Offer(mime)
		}
		src.SetEvents(&wl.WlDataSourceEvents{
			OnSend:      func(mime string, fd int) { send(mime, fd) },
			OnCancelled: cancelled,
		})
		return func(serial uint32) { c.device.SetSelection(src, serial) }, src.Destroy
	}
	c.system.closeDevice = func() {
		// wl_data_device.release is since version 2. The manager has a destructor
		// only since version 4, which is more than this backend binds; it goes with
		// the connection.
		if c.device != nil && c.version >= 2 {
			c.device.Release()
		}
		c.device, c.mgr = nil, nil
	}
	c.device.SetEvents(&wl.WlDataDeviceEvents{
		OnDataOffer: func(obj *wl.WlDataOffer) {
			o := &offer{receive: obj.Receive, destroy: func() { delete(c.dataOffers, obj); obj.Destroy() }}
			obj.SetEvents(&wl.WlDataOfferEvents{OnOffer: func(mime string) { o.mimes = append(o.mimes, mime) }})
			c.dataOffers[obj] = o
			c.system.onOffer(o)
		},
		OnSelection: func(obj *wl.WlDataOffer) { c.system.onSelection(c.dataOffers[obj]) },
		OnLeave:     c.system.dropPending,
	})
	if pg, ok := p.sess.optionalPresent[wl.ZwpPrimarySelectionDeviceManagerV1Interface.Name]; ok {
		c.initPrimary(pg)
	}
	return c
}

// initPrimary binds the primary selection manager and creates the device of the
// seat.
func (c *clipboard) initPrimary(g Global) {
	c.pmgr = wl.BindZwpPrimarySelectionDeviceManagerV1(c.p.sess.registry, g.Name, bindVersion(g, 1))
	c.pdev = c.pmgr.GetDevice(c.p.sess.seat)
	c.primary = &selection{c: c}
	c.primary.newSource = func(mimes []string, send func(string, int), cancelled func()) (func(uint32), func()) {
		src := c.pmgr.CreateSource()
		for _, mime := range mimes {
			src.Offer(mime)
		}
		src.SetEvents(&wl.ZwpPrimarySelectionSourceV1Events{
			OnSend:      func(mime string, fd int) { send(mime, fd) },
			OnCancelled: cancelled,
		})
		return func(serial uint32) { c.pdev.SetSelection(src, serial) }, src.Destroy
	}
	c.primary.closeDevice = func() {
		if c.pdev != nil {
			c.pdev.Destroy()
		}
		if c.pmgr != nil {
			c.pmgr.Destroy()
		}
		c.pdev, c.pmgr = nil, nil
	}
	c.pdev.SetEvents(&wl.ZwpPrimarySelectionDeviceV1Events{
		OnDataOffer: func(obj *wl.ZwpPrimarySelectionOfferV1) {
			o := &offer{receive: obj.Receive, destroy: func() { delete(c.primaryOffers, obj); obj.Destroy() }}
			obj.SetEvents(&wl.ZwpPrimarySelectionOfferV1Events{OnOffer: func(mime string) { o.mimes = append(o.mimes, mime) }})
			c.primaryOffers[obj] = o
			c.primary.onOffer(o)
		},
		OnSelection: func(obj *wl.ZwpPrimarySelectionOfferV1) { c.primary.onSelection(c.primaryOffers[obj]) },
	})
}

// close destroys the objects.
func (c *clipboard) close() {
	c.system.close()
	if c.primary != nil {
		c.primary.close()
	}
}

// reply calls done on the UI thread, after the method that asked has returned.
func (c *clipboard) reply(fn func()) { c.p.Post(fn) }

func (c *clipboard) selectionOf(kind ui.ClipboardKind) *selection {
	if kind == ui.ClipboardKindPrimary {
		return c.primary
	}
	return c.system
}

// SetText makes the app the owner of the clipboard, with the text.
func (c *clipboard) SetText(ctx context.Context, text string, done func(error)) {
	c.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{Text: text, HasText: true}, done)
}

// GetText reads the text of the clipboard.
func (c *clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	c.GetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardFormatText, func(content ui.ClipboardContent, err error) {
		done(content.Text, content.HasText, err)
	})
}

// SetContent makes the app the owner of the selection of the kind, with the
// content. A compositor without the primary selection fails the primary one with
// ClipboardPrimaryUnavailable.
func (c *clipboard) SetContent(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	sel := c.selectionOf(kind)
	if sel == nil {
		err := errPrimaryUnavailable()
		if done != nil {
			c.reply(func() { done(err) })
		}
		return
	}
	sel.set(content, done)
}

// GetContent reads the selection of the kind in the format.
func (c *clipboard) GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	sel := c.selectionOf(kind)
	if sel == nil {
		err := errPrimaryUnavailable()
		c.reply(func() { done(ui.ClipboardContent{}, err) })
		return
	}
	sel.get(ctx, format, done)
}

func errPrimaryUnavailable() error {
	return errors.New("ClipboardPrimaryUnavailable", "the compositor has no primary selection")
}

// writeClipboard writes the data to the pipe of a reader and closes it, on a
// goroutine: the reader may be slow, and the UI thread must not wait for it.
func writeClipboard(fd int, data []byte) {
	f := os.NewFile(uintptr(fd), "clipboard-send")
	go func() {
		_ = f.SetWriteDeadline(time.Now().Add(clipboardReadTimeout))
		if _, err := f.Write(data); err != nil {
			// The reader quit early or was too slow; nobody is left to tell.
			slog.Debug("the reader of the clipboard did not take all the data", "error", err)
		}
		if err := f.Close(); err != nil {
			slog.Debug("closing the pipe of the clipboard failed", "error", err)
		}
	}()
}

// closeFD closes the pipe of a reader for a format the app has no data for.
func closeFD(fd int) {
	if err := os.NewFile(uintptr(fd), "clipboard-send").Close(); err != nil {
		slog.Debug("closing the pipe of the clipboard failed", "error", err)
	}
}

// readClipboard reads the whole pipe, within the timeout and the limit, and
// closes it. Nothing here touches the connection.
func readClipboard(ctx context.Context, r *os.File) ([]byte, error) {
	defer func() { _ = r.Close() }()
	ctx, cancel := context.WithTimeout(ctx, clipboardReadTimeout)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = r.SetReadDeadline(time.Now()) })
	defer stop()
	data, err := io.ReadAll(io.LimitReader(r, clipboardMaxBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.NewErr("ClipboardReadTimeout", ctx.Err())
		}
		return nil, errors.NewErr("ClipboardReadFailed", err)
	}
	if len(data) > clipboardMaxBytes {
		return nil, errors.New("ClipboardTooLarge", "the data of the clipboard is too large", "limit", clipboardMaxBytes)
	}
	return data, nil
}
