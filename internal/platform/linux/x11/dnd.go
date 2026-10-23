//go:build linux && !android

package x11

import (
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// Drag and drop with XDND version 5 (https://www.freedesktop.org/wiki/Specifications/XDND/):
// every window of the app says it is XdndAware, and answers the XdndEnter,
// XdndPosition, XdndLeave and XdndDrop messages of a source; a drag the app starts
// owns the selection XdndSelection, follows the pointer through the XInput2 events
// of the button that is down, and sends those messages to the windows under it.
// Everything is on the UI thread.

const (
	xdndVersion    = 5
	xdndTimeout    = 5 * time.Second
	xdndMaxListLen = 4096

	mimeURIList  = "text/uri-list"
	mimePlainUTF = "text/plain;charset=utf-8"
	mimePlain    = "text/plain"
	mimeUTF8Str  = "UTF8_STRING"

	// localMarkerPrefix starts the type that a drag started by this app offers
	// besides its data: "application/x-glutter-drag-<pid>-<id>".
	localMarkerPrefix = "application/x-glutter-drag-"
)

// xdndTextTypes are the text types the app reads from a drop, best first, and offers.
var xdndTextTypes = []string{mimePlainUTF, mimeUTF8Str, mimePlain}

func localMarker(id uint64) string {
	return localMarkerPrefix + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatUint(id, 10)
}

// localIDOf is the ID of the DragRequest of this process that the type names, or 0.
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

// atom interns an atom by name, once.
func (p *Platform) atom(name string) uint64 {
	if a, ok := p.atomCache[name]; ok {
		return a
	}
	c := dl.CString(name)
	defer dl.Free(c)
	a := p.x.internAtom(p.display, c, 0)
	if p.atomCache == nil {
		p.atomCache = map[string]uint64{}
	}
	p.atomCache[name] = a
	return a
}

// atomName is the name of an atom, or "" for one the server does not know.
func (p *Platform) atomName(atom uint64) string {
	if atom == 0 {
		return ""
	}
	c := p.x.getAtomName(p.display, atom)
	if c == 0 {
		return ""
	}
	defer p.x.free(c)
	return dl.GoString(c)
}

// readLongs reads a property of format 32 as numbers.
func (p *Platform) readLongs(window, property, typ uint64, max int64) []uint64 {
	mem := dl.Calloc(1, 8+4+8+8+8)
	defer dl.Free(mem)
	actualType, actualFormat, nitems, bytesAfter, prop := mem, mem+8, mem+16, mem+24, mem+32
	status := p.x.getWindowProperty(p.display, window, property, 0, max, 0, typ, actualType, actualFormat, nitems, bytesAfter, prop)
	data := *view[uintptr](prop)
	if data != 0 {
		defer p.x.free(data)
	}
	if status != 0 || data == 0 || *view[int32](actualFormat) != 32 {
		return nil
	}
	n := int(*view[uint64](nitems))
	return append([]uint64(nil), unsafe.Slice(view[uint64](data), n)...)
}

// sendClient sends a client message of format 32 to a window.
func (p *Platform) sendClient(dest, messageType uint64, data [5]int64) {
	mem := dl.Calloc(1, xEventSize)
	defer dl.Free(mem)
	ev := view[xClientMessageEvent](mem)
	ev.typ = clientMessage
	ev.window = dest
	ev.messageType = messageType
	ev.format = 32
	ev.data = data
	p.x.sendEvent(p.display, dest, 0, 0, mem)
	p.x.flush(p.display)
}

// dropActionsOfAtom turns the action atom of XDND into the framework's actions.
func (p *Platform) dropActionsOfAtom(a uint64) ui.DropActions {
	switch a {
	case p.atom("XdndActionCopy"):
		return ui.DropActionCopy
	case p.atom("XdndActionMove"):
		return ui.DropActionMove
	case p.atom("XdndActionLink"), p.atom("XdndActionAsk"):
		return ui.DropActionLink
	}
	return ui.DropActionNone
}

// atomOfAction is the atom of the action to perform, or None.
func (p *Platform) atomOfAction(a ui.DropActions) uint64 {
	switch {
	case a.Has(ui.DropActionCopy):
		return p.atom("XdndActionCopy")
	case a.Has(ui.DropActionMove):
		return p.atom("XdndActionMove")
	case a.Has(ui.DropActionLink):
		return p.atom("XdndActionLink")
	}
	return xNone
}

// xdnd is the drag and drop of the platform.
type xdnd struct {
	p      *Platform
	target *xdndTarget
	source *xdndSource
}

// setXdndAware tells the other clients that the window takes drops.
func (p *Platform) setXdndAware(w *window) {
	p.setLongs(w.xid, p.atom("XdndAware"), xaAtom, []uint64{xdndVersion})
}

// ---- target ----

// xdndTarget is a drag over one of the app's windows.
type xdndTarget struct {
	d       *xdnd
	window  *window
	source  uint64
	version int
	types   []string
	// x and y are the position in the window, in pixels.
	x, y int32
	// offered are the actions the source allows (the one it asked for last).
	offered ui.DropActions
	// action is what the app accepted last.
	action ui.DropActions
	// entered is whether the app was told of the drag; the enter comes with the first
	// position.
	entered bool
	// dropping is true from the drop until the app answered.
	dropping bool
	time     uint64
	// reads are the types still to read, text and uri-list; text and uris collect them.
	reads []string
	text  string
	uris  []string
	timer *time.Timer
}

// onClientMessage handles the XDND messages that come to a window of the app. It
// reports whether it was one.
func (d *xdnd) onClientMessage(w *window, typ uint64, data [5]int64) bool {
	if d == nil {
		return false
	}
	p := d.p
	switch typ {
	case p.atom("XdndEnter"):
		d.onEnter(w, data)
	case p.atom("XdndPosition"):
		d.onPosition(w, data)
	case p.atom("XdndLeave"):
		d.onLeave(w, uint64(data[0]))
	case p.atom("XdndDrop"):
		d.onDrop(w, data)
	case p.atom("XdndStatus"):
		d.onStatus(data)
	case p.atom("XdndFinished"):
		d.onFinished(data)
	default:
		return false
	}
	return true
}

func (d *xdnd) onEnter(w *window, data [5]int64) {
	p := d.p
	if t := d.target; t != nil {
		d.abandon(t)
	}
	t := &xdndTarget{d: d, window: w, source: uint64(data[0]), version: int(data[1] >> 24 & 0xff)}
	var atoms []uint64
	if data[1]&1 != 0 {
		atoms = p.readLongs(t.source, p.atom("XdndTypeList"), xaAtom, xdndMaxListLen)
	} else {
		for _, a := range data[2:5] {
			if a != 0 {
				atoms = append(atoms, uint64(a))
			}
		}
	}
	for _, a := range atoms {
		if name := p.atomName(a); name != "" {
			t.types = append(t.types, name)
		}
	}
	d.target = t
}

func (d *xdnd) onPosition(w *window, data [5]int64) {
	p := d.p
	t := d.target
	if t == nil || t.source != uint64(data[0]) || t.dropping {
		return
	}
	t.window = w
	rootX, rootY := int32(data[2]>>16&0xffff), int32(data[2]&0xffff)
	t.x, t.y = p.windowCoords(w, rootX, rootY)
	t.time = uint64(data[3])
	t.offered = p.dropActionsOfAtom(uint64(data[4]))
	if t.offered == ui.DropActionNone {
		t.offered = ui.DropActionCopy
	}
	phase := ui.DropPhaseMove
	if !t.entered {
		t.entered = true
		phase = ui.DropPhaseEnter
	}
	t.action = t.deliver(phase, "", nil)
	t.sendStatus()
}

// windowCoords turns root coordinates into the window's.
func (p *Platform) windowCoords(w *window, rootX, rootY int32) (x, y int32) {
	mem := dl.Calloc(1, 16)
	defer dl.Free(mem)
	if p.x.translateCoordinates(p.display, p.root, w.xid, rootX, rootY, mem, mem+4, mem+8) == 0 {
		return rootX, rootY
	}
	return *view[int32](mem), *view[int32](mem + 4)
}

func (d *xdnd) onLeave(w *window, source uint64) {
	t := d.target
	if t == nil || t.source != source || t.dropping {
		return
	}
	if t.entered {
		t.deliver(ui.DropPhaseLeave, "", nil)
	}
	d.abandon(t)
}

// abandon forgets a drag.
func (d *xdnd) abandon(t *xdndTarget) {
	if t.timer != nil {
		t.timer.Stop()
	}
	if d.target == t {
		d.target = nil
	}
}

func (d *xdnd) onDrop(w *window, data [5]int64) {
	t := d.target
	if t == nil || t.source != uint64(data[0]) || t.dropping {
		return
	}
	t.dropping = true
	t.time = uint64(data[2])
	if !t.entered || t.action == ui.DropActionNone {
		// Nothing accepted the drag: the app hears of the drop, and the source of the refusal.
		t.deliver(ui.DropPhaseDrop, "", nil)
		t.finish(ui.DropActionNone)
		return
	}
	// Read the text and the URIs, one after the other, then hand the drop to the app.
	if mime := t.firstOffered(xdndTextTypes); mime != "" {
		t.reads = append(t.reads, mime)
	}
	if slices.Contains(t.types, mimeURIList) {
		t.reads = append(t.reads, mimeURIList)
	}
	t.timer = time.AfterFunc(xdndTimeout, func() {
		d.p.Post(func() {
			if d.target == t {
				slog.Debug("the source of the drop did not send its data")
				t.complete()
			}
		})
	})
	t.readNext()
}

func (t *xdndTarget) firstOffered(wanted []string) string {
	for _, m := range wanted {
		if slices.Contains(t.types, m) {
			return m
		}
	}
	return ""
}

// readNext asks the source for the next type; with none left the drop is complete.
func (t *xdndTarget) readNext() {
	if len(t.reads) == 0 {
		t.complete()
		return
	}
	p := t.d.p
	p.x.convertSelection(p.display, p.atom("XdndSelection"), p.atom(t.reads[0]), p.atom("GLUTTER_XDND_DATA"), t.window.xid, t.time)
	p.x.flush(p.display)
}

// onSelectionNotify takes the data the source wrote for a read. It reports whether
// the event was for a drop.
func (d *xdnd) onSelectionNotify(ev xSelectionEvent) bool {
	if d == nil {
		return false
	}
	p := d.p
	t := d.target
	if ev.selection != p.atom("XdndSelection") || t == nil || !t.dropping || len(t.reads) == 0 || ev.requestor != t.window.xid {
		return false
	}
	mime := t.reads[0]
	t.reads = t.reads[1:]
	if ev.property != xNone {
		if data := p.readProperty(t.window.xid, ev.property, ev.target, 1<<20); data != nil {
			p.x.deleteProperty(p.display, t.window.xid, ev.property)
			if mime == mimeURIList {
				t.uris = parseURIList(data)
			} else {
				t.text = string(data)
			}
		}
	}
	t.readNext()
	return true
}

// complete hands the drop, with the data, to the app and answers the source.
func (t *xdndTarget) complete() {
	if t.d.target != t {
		return
	}
	action := t.deliver(ui.DropPhaseDrop, t.text, t.uris)
	t.finish(action)
}

// finish ends the drag with the source (version 5 wants XdndFinished) and forgets it.
func (t *xdndTarget) finish(action ui.DropActions) {
	p := t.d.p
	if t.version >= 2 {
		accepted := int64(0)
		if action != ui.DropActionNone {
			accepted = 1
		}
		p.sendClient(t.source, p.atom("XdndFinished"), [5]int64{int64(t.window.xid), accepted, int64(p.atomOfAction(action)), 0, 0})
	}
	t.d.abandon(t)
}

// sendStatus tells the source whether the app takes the drop here, and with which
// action. The source is asked for every move (bit 1).
func (t *xdndTarget) sendStatus() {
	p := t.d.p
	flags := int64(2)
	action := int64(0)
	if t.action != ui.DropActionNone {
		flags |= 1
		action = int64(p.atomOfAction(t.action))
	}
	p.sendClient(t.source, p.atom("XdndStatus"), [5]int64{int64(t.window.xid), flags, 0, 0, action})
}

// deliver gives the event to the app and returns its answer.
func (t *xdndTarget) deliver(phase ui.DropPhase, text string, uris []string) ui.DropActions {
	p := t.d.p
	h := p.handler
	if h == nil {
		return ui.DropActionNone
	}
	var types []string
	var local uint64
	for _, m := range t.types {
		if id := localIDOf(m); id != 0 {
			local = id
		}
		if strings.HasPrefix(m, localMarkerPrefix) || strings.HasPrefix(m, "TARGETS") {
			continue
		}
		types = append(types, m)
	}
	return h.OnDrop(ui.DropEvent{
		ViewID: t.window.id, Phase: phase, Position: ui.Offset{Dx: float64(t.x), Dy: float64(t.y)},
		Types: types, Actions: t.offered, Text: text, URIs: uris, LocalID: local,
	})
}

// ---- source ----

// xdndSource is a drag the app started.
type xdndSource struct {
	d       *xdnd
	window  *window
	request ui.DragRequest
	// offered are the types of the data, as atoms and names.
	offered []uint64
	// target is the window under the pointer that takes drops (0 for none), with the
	// version it speaks.
	target  uint64
	version int
	// accepted and action are the target's last status.
	accepted bool
	action   ui.DropActions
	// sentEnter is whether the target got the XdndEnter.
	sentEnter bool
	// dropped is true once the drop was sent; the end waits for XdndFinished.
	dropped bool
	done    bool
	timer   *time.Timer
	// rootX and rootY are where the pointer was last.
	rootX, rootY int32
	// cursor is the cursor the source window had before the drag.
	cursor platform.MouseCursor
}

// startDrag starts a drag from the window. The button that started it must be down.
func (d *xdnd) startDrag(w *window, request ui.DragRequest) error {
	p := d.p
	if d.source != nil {
		return errors.New(tagDragInProgress, "a drag of the app is already going on")
	}
	if p.input == nil || p.input.buttonsDown == 0 {
		return errors.New(tagDragNoButton, "a drag starts with a pointer button that is down")
	}
	sel := p.atom("XdndSelection")
	p.x.setSelectionOwner(p.display, sel, w.xid, currentTime)
	if p.x.getSelectionOwner(p.display, sel) != w.xid {
		return errors.New(tagDragSelectionFailed, "the app could not take the XdndSelection")
	}
	s := &xdndSource{d: d, window: w, request: request, cursor: w.cursor}
	if request.Text != "" {
		for _, m := range xdndTextTypes {
			s.offered = append(s.offered, p.atom(m))
		}
		s.offered = append(s.offered, xaString)
	}
	if len(request.URIs) > 0 {
		s.offered = append(s.offered, p.atom(mimeURIList))
	}
	s.offered = append(s.offered, p.atom(localMarker(request.ID)))
	d.source = s
	// More than three types go in a property of the source window.
	if len(s.offered) > 3 {
		p.setLongs(w.xid, p.atom("XdndTypeList"), xaAtom, s.offered)
	}
	p.cursors.apply(w, platform.MouseCursorGrabbing)
	return nil
}

// preferredAction is the action to ask the target for: copy if allowed, else move,
// else link.
func (s *xdndSource) preferredAction() ui.DropActions {
	a := s.request.Actions
	switch {
	case a == ui.DropActionNone, a.Has(ui.DropActionCopy):
		return ui.DropActionCopy
	case a.Has(ui.DropActionMove):
		return ui.DropActionMove
	}
	return ui.DropActionLink
}

// onMotion follows the pointer (root coordinates).
func (s *xdndSource) onMotion(rootX, rootY int32) {
	if s.done || s.dropped {
		return
	}
	p := s.d.p
	s.rootX, s.rootY = rootX, rootY
	target, version := p.findXdndTarget(rootX, rootY)
	if target != s.target {
		s.leaveTarget()
		s.target, s.version = target, version
		s.accepted, s.action = false, ui.DropActionNone
		if target != 0 {
			s.sendEnter()
		}
	}
	if s.target != 0 {
		p.sendClient(s.target, p.atom("XdndPosition"), [5]int64{int64(s.window.xid), 0, int64(rootX)<<16 | int64(rootY), 0, int64(p.atomOfAction(s.preferredAction()))})
	}
	s.updateCursor()
}

func (s *xdndSource) sendEnter() {
	p := s.d.p
	data := [5]int64{int64(s.window.xid), int64(min(s.version, xdndVersion)) << 24}
	if len(s.offered) > 3 {
		data[1] |= 1
	} else {
		for i, a := range s.offered {
			data[2+i] = int64(a)
		}
	}
	p.sendClient(s.target, p.atom("XdndEnter"), data)
	s.sentEnter = true
}

func (s *xdndSource) leaveTarget() {
	if s.target != 0 && s.sentEnter {
		s.d.p.sendClient(s.target, s.d.p.atom("XdndLeave"), [5]int64{int64(s.window.xid)})
	}
	s.target, s.sentEnter = 0, false
}

// updateCursor shows what a drop would do.
func (s *xdndSource) updateCursor() {
	cursor := platform.MouseCursorNoDrop
	if s.target != 0 && s.accepted {
		switch {
		case s.action.Has(ui.DropActionMove):
			cursor = platform.MouseCursorMove
		case s.action.Has(ui.DropActionLink):
			cursor = platform.MouseCursorAlias
		default:
			cursor = platform.MouseCursorCopy
		}
	}
	s.d.p.cursors.apply(s.window, cursor)
}

// onRelease drops, or ends the drag when nothing takes it.
func (s *xdndSource) onRelease() {
	if s.done || s.dropped {
		return
	}
	p := s.d.p
	if s.target == 0 || !s.accepted {
		s.leaveTarget()
		s.end(ui.DropActionNone)
		return
	}
	s.dropped = true
	p.sendClient(s.target, p.atom("XdndDrop"), [5]int64{int64(s.window.xid), 0, 0, 0, 0})
	if s.version < 2 {
		s.end(s.action) // no XdndFinished from such a target
		return
	}
	s.timer = time.AfterFunc(xdndTimeout, func() {
		p.Post(func() {
			if !s.done {
				slog.Debug("the target of the drop did not finish")
				s.end(ui.DropActionNone)
			}
		})
	})
}

// onStatus is the answer of the target to a position.
func (d *xdnd) onStatus(data [5]int64) {
	s := d.source
	if s == nil || uint64(data[0]) != s.target {
		return
	}
	s.accepted = data[1]&1 != 0
	s.action = d.p.dropActionsOfAtom(uint64(data[4]))
	if !s.accepted {
		s.action = ui.DropActionNone
	}
	if !s.dropped {
		s.updateCursor()
	}
}

// onFinished is the end of a drop on the target.
func (d *xdnd) onFinished(data [5]int64) {
	s := d.source
	if s == nil || !s.dropped || uint64(data[0]) != s.target {
		return
	}
	action := ui.DropActionNone
	if data[1]&1 != 0 {
		action = d.p.dropActionsOfAtom(uint64(data[2]))
		if action == ui.DropActionNone {
			action = s.action
		}
	}
	s.end(action)
}

// end finishes the drag and tells the app how it ended.
func (s *xdndSource) end(action ui.DropActions) {
	if s.done {
		return
	}
	s.done = true
	p := s.d.p
	if s.timer != nil {
		s.timer.Stop()
	}
	if s.d.source == s {
		s.d.source = nil
	}
	if !s.window.closed {
		p.x.deleteProperty(p.display, s.window.xid, p.atom("XdndTypeList"))
		p.cursors.apply(s.window, s.cursor)
	}
	if h := p.handler; h != nil {
		h.OnDragEnd(ui.DragResult{ID: s.request.ID, Action: action})
	}
}

// onSelectionRequest answers a target that asks for the data of the drag. It reports
// whether the request was for the drag.
func (d *xdnd) onSelectionRequest(ev xSelectionRequestEvent) bool {
	if d == nil {
		return false
	}
	p := d.p
	s := d.source
	if ev.selection != p.atom("XdndSelection") {
		return false
	}
	property := ev.property
	if property == xNone {
		property = ev.target
	}
	var reply uint64 = xNone
	if s != nil && ev.owner == s.window.xid {
		switch name := p.atomName(ev.target); {
		case ev.target == p.atom("TARGETS"):
			mem := dl.Calloc(uintptr(len(s.offered)+1), 8)
			targets := append([]uint64{p.atom("TARGETS")}, s.offered...)
			copy(unsafe.Slice(view[uint64](mem), len(targets)), targets)
			p.x.changeProperty(p.display, ev.requestor, property, xaAtom, 32, propModeReplace, mem, int32(len(targets)))
			dl.Free(mem)
			reply = property
		case slices.Contains(xdndTextTypes, name) && s.request.Text != "":
			reply = p.writeDragData(ev.requestor, property, ev.target, []byte(s.request.Text))
		case ev.target == xaString && s.request.Text != "":
			reply = p.writeDragData(ev.requestor, property, ev.target, []byte(latin1(s.request.Text)))
		case name == mimeURIList && len(s.request.URIs) > 0:
			reply = p.writeDragData(ev.requestor, property, ev.target, []byte(strings.Join(s.request.URIs, "\r\n")+"\r\n"))
		}
	}
	mem := dl.Calloc(1, xEventSize)
	defer dl.Free(mem)
	*view[xSelectionEvent](mem) = xSelectionEvent{
		typ: selectionNotify, requestor: ev.requestor, selection: ev.selection,
		target: ev.target, property: reply, time: ev.time,
	}
	p.x.sendEvent(p.display, ev.requestor, 0, 0, mem)
	p.x.flush(p.display)
	return true
}

func (p *Platform) writeDragData(window, property, target uint64, data []byte) uint64 {
	var mem uintptr
	if len(data) > 0 {
		mem = dl.Malloc(uintptr(len(data)))
		copy(dl.Bytes(mem, len(data)), data)
		defer dl.Free(mem)
	}
	p.x.changeProperty(p.display, window, property, target, 8, propModeReplace, mem, int32(len(data)))
	return property
}

// findXdndTarget finds the window under the root position that takes drops: the
// deepest window below the pointer that is XdndAware. It returns 0 for none.
func (p *Platform) findXdndTarget(rootX, rootY int32) (window uint64, version int) {
	aware := p.atom("XdndAware")
	mem := dl.Calloc(1, 24)
	defer dl.Free(mem)
	cur := p.root
	for range 32 {
		if p.x.translateCoordinates(p.display, p.root, cur, rootX, rootY, mem, mem+4, mem+8) == 0 {
			break
		}
		child := *view[uint64](mem + 8)
		if child == 0 {
			break
		}
		cur = child
		if v := p.readLongs(cur, aware, xaAtom, 1); len(v) == 1 {
			window, version = cur, int(v[0])
		}
	}
	return window, version
}

// ---- the platform's side ----

// close ends the drags when the platform closes.
func (d *xdnd) close() {
	if d == nil {
		return
	}
	if t := d.target; t != nil {
		d.abandon(t)
	}
	if s := d.source; s != nil {
		s.done = true
		if s.timer != nil {
			s.timer.Stop()
		}
		d.source = nil
	}
}

// windowClosed ends the drags that involve a window that closes.
func (d *xdnd) windowClosed(w *window) {
	if d == nil {
		return
	}
	if t := d.target; t != nil && t.window == w {
		d.abandon(t)
	}
	if s := d.source; s != nil && s.window == w {
		s.leaveTarget()
		s.end(ui.DropActionNone)
	}
}

// activeDrag is the drag the app started and that goes on, or nil.
func (d *xdnd) activeDrag() *xdndSource {
	if d == nil || d.source == nil || d.source.done {
		return nil
	}
	return d.source
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragInProgress      = "DragInProgress"
	tagDragNoButton        = "DragNoButton"
	tagDragSelectionFailed = "DragSelectionFailed"
)
