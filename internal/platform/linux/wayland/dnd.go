package wayland

import (
	"context"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// Drag and drop through wl_data_device (SPEC §8.11 "Data exchange"): the app is a
// target for the drags of other clients (and its own), and the source of drags it
// starts. Both belong to the UI thread; the data moves through pipes read and
// written on goroutines, like the clipboard's.

// localMarkerPrefix starts the MIME type that a drag started by this app offers
// besides its data: "application/x-glutter-drag-<pid>-<id>". When the app is the
// target of such a drag it knows the drag is its own and which request made it.
const localMarkerPrefix = "application/x-glutter-drag-"

// dropReadTypes are the formats the app reads from a drop, in order of preference
// within each group: the text, and the list of URIs.
var dropTextTypes = readMimeTypes

func localMarker(id uint64) string {
	return localMarkerPrefix + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatUint(id, 10)
}

// localIDOf is the ID of the DragRequest of this process that the MIME type names,
// or 0.
func localIDOf(mime string) uint64 {
	rest, ok := strings.CutPrefix(mime, localMarkerPrefix)
	if !ok {
		return 0
	}
	pid, id, ok := strings.Cut(rest, "-")
	if !ok || pid != strconv.Itoa(os.Getpid()) {
		return 0
	}
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// wlActions turns the actions of the framework into those of wl_data_device_manager.
func wlActions(a ui.DropActions) wl.WlDataDeviceManagerDndAction {
	var out wl.WlDataDeviceManagerDndAction
	if a&ui.DropActionCopy != 0 {
		out |= wl.WlDataDeviceManagerDndActionCopy
	}
	if a&ui.DropActionMove != 0 {
		out |= wl.WlDataDeviceManagerDndActionMove
	}
	if a&ui.DropActionLink != 0 {
		out |= wl.WlDataDeviceManagerDndActionAsk // the protocol has no link action
	}
	return out
}

// dropActionsOf turns the actions of wl_data_device_manager into the framework's.
func dropActionsOf(a wl.WlDataDeviceManagerDndAction) ui.DropActions {
	var out ui.DropActions
	if a&wl.WlDataDeviceManagerDndActionCopy != 0 {
		out |= ui.DropActionCopy
	}
	if a&wl.WlDataDeviceManagerDndActionMove != 0 {
		out |= ui.DropActionMove
	}
	if a&wl.WlDataDeviceManagerDndActionAsk != 0 {
		out |= ui.DropActionLink
	}
	return out
}

// dnd is the drag and drop of the seat's data device.
type dnd struct {
	c *clipboard
	// pipe makes the pipe that a source writes a type into; tests replace it.
	pipe func() (*os.File, *os.File, error)
	// target is the drag that is over a window of the app, or nil.
	target *dndTarget
	// source is the drag the app started, or nil.
	source *dndSource
}

func newDnd(c *clipboard) *dnd { return &dnd{c: c, pipe: os.Pipe} }

// dndTarget is a drag over one of the app's windows.
type dndTarget struct {
	d      *dnd
	offer  *offer
	serial uint32
	win    *window
	// x, y are the position in the window, in logical pixels.
	x, y float64
	// action is the action the app accepted last.
	action ui.DropActions
	// dropping is true from the drop until the data is read and the app answered.
	dropping bool
	// left is true when the leave came while the drop was in progress.
	left bool
}

// onEnter starts a drag over a window.
func (d *dnd) onEnter(serial uint32, surface *wl.WlSurface, x, y float64, obj *wl.WlDataOffer) {
	o := d.c.dataOffers[obj]
	if o != nil {
		d.c.system.take(o)
	}
	w := d.c.p.windowOf(surface)
	if o == nil || w == nil || d.c.p.handler == nil {
		if o != nil {
			o.destroy()
		}
		return
	}
	if d.target != nil {
		d.target.finish()
	}
	t := &dndTarget{d: d, offer: o, serial: serial, win: w, x: x, y: y}
	d.target = t
	t.deliver(ui.DropPhaseEnter, "", nil)
}

// onMotion moves the drag.
func (d *dnd) onMotion(_ uint32, x, y float64) {
	t := d.target
	if t == nil || t.dropping {
		return
	}
	t.x, t.y = x, y
	t.deliver(ui.DropPhaseMove, "", nil)
}

// onLeave ends the drag over the window, unless a drop is being completed.
func (d *dnd) onLeave() {
	d.c.system.dropPending()
	t := d.target
	if t == nil {
		return
	}
	if t.dropping {
		t.left = true // the drop completes first
		return
	}
	t.deliver(ui.DropPhaseLeave, "", nil)
	t.finish()
}

// onDrop reads the data of the drop and gives it to the app.
func (d *dnd) onDrop() {
	t := d.target
	if t == nil || t.dropping {
		return
	}
	t.dropping = true
	if t.action == ui.DropActionNone {
		// Nothing accepted the drag: there is nothing to read, and no finish.
		t.deliver(ui.DropPhaseDrop, "", nil)
		t.finish()
		return
	}
	t.readData(func(text string, uris []string) {
		if d.target != t {
			return
		}
		action := t.deliver(ui.DropPhaseDrop, text, uris)
		if action != ui.DropActionNone && t.offer.obj != nil && t.offer.obj.ProxyVersion() >= 3 {
			t.offer.obj.Finish()
		}
		t.finish()
	})
}

// finish forgets the drag and destroys its offer.
func (t *dndTarget) finish() {
	if t.d.target == t {
		t.d.target = nil
	}
	t.offer.destroy()
}

// types are the MIME types the source offers, without the marker of the app's own
// drags.
func (t *dndTarget) types() []string {
	var out []string
	for _, m := range t.offer.mimes {
		if !strings.HasPrefix(m, localMarkerPrefix) {
			out = append(out, m)
		}
	}
	return out
}

func (t *dndTarget) localID() uint64 {
	for _, m := range t.offer.mimes {
		if id := localIDOf(m); id != 0 {
			return id
		}
	}
	return 0
}

// deliver gives the event to the app and applies its answer to the offer.
func (t *dndTarget) deliver(phase ui.DropPhase, text string, uris []string) ui.DropActions {
	p := t.d.c.p
	scale := float64(t.win.scale) / 120
	if scale <= 0 {
		scale = 1
	}
	offered := t.offer.sourceActions
	if offered == ui.DropActionNone {
		offered = ui.DropActionCopy // a source of version 2 or less copies
	}
	h := p.handler
	if h == nil {
		return ui.DropActionNone
	}
	action := h.OnDrop(ui.DropEvent{
		ViewID: t.win.id, Phase: phase, Position: ui.Offset{Dx: t.x * scale, Dy: t.y * scale},
		Types: t.types(), Actions: offered, Text: text, URIs: uris, LocalID: t.localID(),
	})
	if phase == ui.DropPhaseLeave {
		return ui.DropActionNone
	}
	t.action = action
	if phase != ui.DropPhaseDrop {
		t.apply(offered, action)
	}
	return action
}

// apply tells the source which type and action the app accepts at this place.
func (t *dndTarget) apply(offered, action ui.DropActions) {
	obj := t.offer.obj
	if obj == nil {
		return
	}
	if action == ui.DropActionNone {
		obj.Accept(t.serial, "")
		if obj.ProxyVersion() >= 3 {
			obj.SetActions(0, 0)
		}
		return
	}
	mime := t.acceptedMime()
	obj.Accept(t.serial, mime)
	if obj.ProxyVersion() >= 3 {
		obj.SetActions(wlActions(offered), wlActions(action))
	}
}

// acceptedMime is the type the app reads: files first, then text.
func (t *dndTarget) acceptedMime() string {
	if t.offer.offers(mimeURIList) {
		return mimeURIList
	}
	for _, m := range dropTextTypes {
		if t.offer.offers(m) {
			return m
		}
	}
	if len(t.offer.mimes) > 0 {
		return t.offer.mimes[0]
	}
	return ""
}

// readData reads the text and the URIs of the offer on goroutines and calls done on
// the UI thread with them.
func (t *dndTarget) readData(done func(text string, uris []string)) {
	p := t.d.c.p
	var wg sync.WaitGroup
	var text string
	var uris []string
	start := func(mimes []string, apply func([]byte)) {
		var mime string
		for _, m := range mimes {
			if t.offer.offers(m) {
				mime = m
				break
			}
		}
		if mime == "" {
			return
		}
		r, err := t.d.receivePipe(t.offer, mime)
		if err != nil {
			slog.Debug("a type of the drop could not be read", "type", mime, "error", err)
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, rerr := readClipboard(context.Background(), r)
			if rerr == nil {
				apply(data)
			}
		}()
	}
	start(dropTextTypes, func(data []byte) { text = string(data) })
	start([]string{mimeURIList}, func(data []byte) { uris = parseURIList(data) })
	go func() {
		wg.Wait()
		p.Post(func() { done(text, uris) })
	}()
}

// receivePipe asks the source to write the type into a new pipe and returns the
// reading end. It fails with DropReadFailed when the pipe cannot be made.
func (d *dnd) receivePipe(o *offer, mime string) (*os.File, error) {
	r, w, err := d.pipe()
	if err != nil {
		return nil, errors.NewErr("DropReadFailed", err)
	}
	o.receive(mime, int(w.Fd()))
	// The connection keeps its own copy of the descriptor until it has sent it.
	if cerr := w.Close(); cerr != nil {
		slog.Debug("closing the pipe of a drop failed", "error", cerr)
	}
	return r, nil
}

// dndSource is a drag the app started.
type dndSource struct {
	d       *dnd
	src     *wl.WlDataSource
	icon    *dragIcon
	request ui.DragRequest
	// action is the action the target chose last.
	action ui.DropActions
	done   bool
}

// StartDrag starts a drag from the window with the data of the request. It must be
// called while the pointer button that started the drag is down. It fails with
// DragUnsupported when the compositor has no data device and DragInProgress while
// another drag of the app goes on.
func (c *clipboard) startDrag(w *window, request ui.DragRequest) error {
	if c == nil || c.device == nil || c.mgr == nil {
		return errors.New("DragUnsupported", "the compositor has no wl_data_device_manager")
	}
	d := c.dnd
	if d.source != nil {
		return errors.New("DragInProgress", "a drag of the app is already going on")
	}
	if c.p.pointerSerial == 0 {
		return errors.New("DragNoButton", "a drag starts with a pointer button that is down")
	}
	actions := request.Actions
	if actions == ui.DropActionNone {
		actions = ui.DropActionCopy
	}
	s := &dndSource{d: d, request: request, action: ui.DropActionNone}
	src := c.mgr.CreateDataSource()
	s.src = src
	if request.Text != "" {
		for _, m := range offeredMimeTypes {
			src.Offer(m)
		}
	}
	if len(request.URIs) > 0 {
		src.Offer(mimeURIList)
	}
	src.Offer(localMarker(request.ID))
	if c.version >= 3 {
		src.SetActions(wlActions(actions))
	}
	src.SetEvents(&wl.WlDataSourceEvents{
		OnSend: func(mime string, fd int) {
			switch {
			case slices.Contains(offeredMimeTypes, mime) && request.Text != "":
				writeClipboard(fd, []byte(request.Text))
			case mime == mimeURIList && len(request.URIs) > 0:
				writeClipboard(fd, []byte(formatURIList(request.URIs)))
			default:
				closeFD(fd)
			}
		},
		OnAction:           func(a wl.WlDataDeviceManagerDndAction) { s.action = dropActionsOf(a) },
		OnDndFinished:      func() { s.end(s.action) },
		OnCancelled:        func() { s.end(ui.DropActionNone) },
		OnDndDropPerformed: func() {},
	})
	d.source = s
	var iconSurface *wl.WlSurface
	if icon, err := c.p.newDragIcon(); err == nil {
		s.icon = icon
		iconSurface = icon.surface
	}
	c.device.StartDrag(src, w.surface, iconSurface, c.p.pointerSerial)
	return nil
}

// end finishes the drag and tells the app how it ended.
func (s *dndSource) end(action ui.DropActions) {
	if s.done {
		return
	}
	s.done = true
	if s.d.source == s {
		s.d.source = nil
	}
	s.src.Destroy()
	if s.icon != nil {
		s.icon.destroy()
	}
	if h := s.d.c.p.handler; h != nil {
		h.OnDragEnd(ui.DragResult{ID: s.request.ID, Action: action})
	}
}

// close ends the drags when the platform closes.
func (d *dnd) close() {
	if d.target != nil {
		d.target.finish()
	}
	if s := d.source; s != nil {
		s.done = true
		d.source = nil
		s.src.Destroy()
		if s.icon != nil {
			s.icon.destroy()
		}
	}
}
