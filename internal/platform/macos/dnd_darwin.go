//go:build darwin && !ios

package macos

import (
	"os"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// droppedTypes are the pasteboard types that the view registers for: the ones a drop is read from.
var droppedTypes = []string{pbFileURL, pbURL, pbText, pbPlainText, pbOldString}

// The glue of drag and drop (SPEC §29.9): the view is an NSDraggingDestination, and StartDrag begins an
// NSDraggingSession whose source is a class that Go implements. The rules are in dnd.go.

// dropState is what a window knows of the drag that is over it.
type dropState struct {
	// entered says that the framework heard of the drag.
	entered bool
	// types and local describe the data on offer: they are read at the enter.
	types []string
	local uint64
}

// registerDrops makes the view a destination for what a drop reads.
func (w *window) registerDrops() {
	types := make([]objc.ID, len(droppedTypes))
	for i, t := range droppedTypes {
		types[i] = objcx.NewString(t)
	}
	arr := objcx.NewArray(types...)
	w.view.Send(sel("registerForDraggedTypes:"), arr)
	objcx.Release(arr)
	for _, t := range types {
		objcx.Release(t)
	}
}

// pasteboardItems reads the items of a pasteboard: the types of each, and the strings of the types that a drop
// reads.
func pasteboardItems(pb objc.ID) []dropItem {
	items := send[objc.ID](pb, "pasteboardItems")
	n := objcx.ArrayCount(items)
	out := make([]dropItem, 0, n)
	for i := range n {
		item := objcx.ArrayAt(items, i)
		it := dropItem{Strings: map[string]string{}}
		types := send[objc.ID](item, "types")
		for j := range objcx.ArrayCount(types) {
			it.Types = append(it.Types, objcx.String(objcx.ArrayAt(types, j)))
		}
		for _, t := range droppedTypes {
			if !containsString(it.Types, t) {
				continue
			}
			ns := objcx.NewString(t)
			if s := objcx.String(item.Send(sel("stringForType:"), ns)); s != "" {
				it.Strings[t] = s
			}
			objcx.Release(ns)
		}
		out = append(out, it)
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// dropEvent makes the event for the framework with the state of the drag that info (an NSDraggingInfo) describes.
func (w *window) dropEvent(phase ui.DropPhase, info objc.ID) ui.DropEvent {
	mask := send[uint64](info, "draggingSourceOperationMask")
	loc := w.p.dragLocation(info)
	pt := objc.Send[nsPoint](w.view, sel("convertPoint:fromView:"), loc, objc.ID(0))
	h := send[nsRect](w.view, "bounds").Size.Height
	return ui.DropEvent{
		ViewID: w.id, Phase: phase,
		Position: ui.Offset{Dx: pt.X * w.scale, Dy: (h - pt.Y) * w.scale},
		Types:    w.drop.types, Actions: dropActionsOf(mask), LocalID: w.drop.local,
	}
}

// deliverDrop gives the event to the handler and returns the operation for the action it accepted.
func (w *window) deliverDrop(ev ui.DropEvent, info objc.ID) uint64 {
	h := w.p.handler
	if h == nil || w.closed {
		return dragOpNone
	}
	return dragOperationFor(h.OnDrop(ev), send[uint64](info, "draggingSourceOperationMask"))
}

// dragEntered is a drag coming into the view: its types are read and the framework hears of it. A window with
// a modal window over it takes no drag.
func (w *window) dragEntered(info objc.ID) uint64 {
	if w.blocked() {
		return dragOpNone
	}
	w.drop.types, w.drop.local = describeDrag(pasteboardItems(send[objc.ID](info, "draggingPasteboard")), os.Getpid())
	w.drop.entered = true
	return w.deliverDrop(w.dropEvent(ui.DropPhaseEnter, info), info)
}

// dragUpdated is the drag moving over the view, or the keys changing the operation.
func (w *window) dragUpdated(info objc.ID) uint64 {
	if !w.drop.entered {
		return dragOpNone
	}
	return w.deliverDrop(w.dropEvent(ui.DropPhaseMove, info), info)
}

// dragExited is the drag leaving the view without a drop.
func (w *window) dragExited() {
	if !w.drop.entered {
		return
	}
	w.drop.entered = false
	if h := w.p.handler; h != nil && !w.closed {
		h.OnDrop(ui.DropEvent{ViewID: w.id, Phase: ui.DropPhaseLeave, Types: w.drop.types, LocalID: w.drop.local})
	}
	w.drop = dropState{}
}

// dragPerform is the drop: the data is read now, and the framework hears of the drop with it. It is true when the
// framework accepted an action that the source allows.
func (w *window) dragPerform(info objc.ID) bool {
	if w.blocked() || w.closed {
		return false
	}
	data := readDrop(pasteboardItems(send[objc.ID](info, "draggingPasteboard")), os.Getpid())
	w.drop = dropState{types: data.Types, local: data.Local}
	ev := w.dropEvent(ui.DropPhaseDrop, info)
	ev.Text, ev.URIs = data.Text, data.URIs
	op := w.deliverDrop(ev, info)
	w.drop = dropState{}
	return op != dragOpNone
}

// dropMethods are the methods of NSDraggingDestination that the view implements.
func dropMethods() []objcx.Method {
	dragOp := func(name string, fn func(w *window, info objc.ID) uint64) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL, info objc.ID) (op uint64) {
			if w := windowOf(self); w != nil && !w.closed {
				w.p.guard(func() { op = fn(w, info) })
			}
			return op
		}}
	}
	return []objcx.Method{
		dragOp("draggingEntered:", (*window).dragEntered),
		dragOp("draggingUpdated:", (*window).dragUpdated),
		{Name: "draggingExited:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
			if w := windowOf(self); w != nil && !w.closed {
				w.p.guard(w.dragExited)
			}
		}},
		{Name: "prepareForDragOperation:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) bool {
			w := windowOf(self)
			return w != nil && !w.closed && !w.blocked()
		}},
		{Name: "performDragOperation:", Fn: func(self objc.ID, _ objc.SEL, info objc.ID) (ok bool) {
			if w := windowOf(self); w != nil && !w.closed {
				w.p.guard(func() { ok = w.dragPerform(info) })
			}
			return ok
		}},
		{Name: "wantsPeriodicDraggingUpdates", Fn: func(objc.ID, objc.SEL) bool { return false }},
	}
}

// ---- the drags the app starts ----

// dragSession is the drag that the app started: its id, the window it started from and the source object
// that the session needs until it ends.
type dragSession struct {
	p      *Platform
	id     uint64
	view   ui.ViewID
	source objc.ID
	handle objcx.Handle
}

// dragSourceMethods are the methods of NSDraggingSource that the source implements.
func dragSourceMethods() []objcx.Method {
	return []objcx.Method{
		// The same operations inside this app and outside it; the modifier keys change the operation.
		{Name: "draggingSession:sourceOperationMaskForDraggingContext:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID, _ int64) uint64 {
			if s, ok := objcx.LookupAs[*dragSession](objcx.HandleOf(self)); ok {
				return s.p.dragMask
			}
			return dragOpCopy
		}},
		// The end of the drag: the point is two doubles, which the callback takes as the registers they travel in.
		{Name: "draggingSession:endedAtPoint:operation:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID, _, _ float64, op uint64) {
			if s, ok := objcx.LookupAs[*dragSession](objcx.HandleOf(self)); ok {
				s.p.guard(func() { s.p.endDrag(s, op) })
			}
		}},
		{Name: "ignoreModifierKeysForDraggingSession:", Fn: func(objc.ID, objc.SEL, objc.ID) bool { return false }},
	}
}

// StartDrag begins a drag from the window with the mouse button that is down: the items go on the dragging
// pasteboard and AppKit runs the drag, which ends with the callback of the source (SPEC §29.9). It fails with
// DragInProgress when a drag of the app is going on, with DragNoButton when the event that is being handled is
// not a button that is down, and with DragEmpty when the request has no text and no URI.
//
// Flutter: no counterpart
func (w *window) StartDrag(req ui.DragRequest) error {
	p := w.p
	if p.drag != nil {
		return errors.New(tagDragInProgress, "a drag of the app is already going on")
	}
	ev := p.currentEvent()
	if ev == 0 || !isButtonEvent(send[uint64](ev, "type")) {
		return errors.New(tagDragNoButton, "a drag starts with a pointer button that is down")
	}
	entries := dragEntries(req, os.Getpid())
	if len(entries) == 0 {
		return errors.New(tagDragEmpty, "the drag has no text and no URI to offer", "id", req.ID)
	}
	s := &dragSession{p: p, id: req.ID, view: w.id}
	s.handle = objcx.NewHandle(s)
	s.source = p.s.dragSource.New(s.handle)
	p.drag = s
	p.dragMask = dragMaskOf(req.Actions)
	objcx.WithPool(func() {
		items := w.draggingItems(entries, ev)
		p.beginDrag(w, items, ev, s.source)
	})
	// The drag swallows the release of the button, so the pointer that the framework knows as down is cancelled
	// once the handler that asked has returned.
	p.Post(func() {
		if w.closed {
			return
		}
		p.guard(func() {
			p.ptr.cancelView(w.id, p.frameTime(p.s.mediaTime()))
			p.flushPointers()
		})
	})
	return nil
}

// isButtonEvent says that the event type is a press or a drag of a mouse button.
func isButtonEvent(t uint64) bool {
	switch t {
	case eventTypeLeftMouseDown, eventTypeLeftMouseDragged, 3, 7, 25, 27: // right and other buttons
		return true
	}
	return false
}

// pasteboardItemFor makes the NSPasteboardItem of an entry.
func pasteboardItemFor(e pasteboardEntry) objc.ID {
	item := class("NSPasteboardItem").Send(sel("alloc")).Send(sel("init"))
	for _, t := range e.Types {
		ty, s := objcx.NewString(t), objcx.NewString(e.Strings[t])
		item.Send(sel("setString:forType:"), s, ty)
		objcx.Release(ty)
		objcx.Release(s)
	}
	return item
}

// draggingItems makes the NSDraggingItems of the entries, each with the icon of what it is and a frame at the
// place of the event, a little off from the one before.
func (w *window) draggingItems(entries []pasteboardEntry, ev objc.ID) objc.ID {
	loc := objc.Send[nsPoint](w.view, sel("convertPoint:fromView:"), send[nsPoint](ev, "locationInWindow"), objc.ID(0))
	workspace := class("NSWorkspace").Send(sel("sharedWorkspace"))
	items := make([]objc.ID, 0, len(entries))
	for i, e := range entries {
		pb := pasteboardItemFor(e)
		di := class("NSDraggingItem").Send(sel("alloc")).Send(sel("initWithPasteboardWriter:"), pb)
		objcx.Release(pb)
		icon := dragIcon(workspace, e)
		off := float64(4 * i)
		frame := nsRect{Origin: nsPoint{X: loc.X - 16 + off, Y: loc.Y - 16 - off}, Size: nsSize{Width: 32, Height: 32}}
		di.Send(sel("setDraggingFrame:contents:"), frame, icon)
		items = append(items, di)
	}
	arr := objcx.Autorelease(objcx.NewArray(items...))
	for _, di := range items {
		objcx.Release(di)
	}
	return arr
}

// dragIcon is the icon of an entry: the icon of the file for a file URL, else the icon of a file type (text, or an
// internet location for a URL).
func dragIcon(workspace objc.ID, e pasteboardEntry) objc.ID {
	if uri := e.Strings[pbFileURL]; uri != "" {
		s := objcx.NewString(uri)
		defer objcx.Release(s)
		url := class("NSURL").Send(sel("URLWithString:"), s)
		if url != 0 {
			if path := send[objc.ID](url, "path"); path != 0 {
				return workspace.Send(sel("iconForFile:"), path)
			}
		}
	}
	kind := "txt"
	if e.Strings[pbURL] != "" {
		kind = "webloc"
	}
	k := objcx.NewString(kind)
	defer objcx.Release(k)
	return workspace.Send(sel("iconForFileType:"), k)
}

// beginDragSession is the real beginDrag: the view begins a dragging session with the items.
func beginDragSession(w *window, items, ev, source objc.ID) {
	w.view.Send(sel("beginDraggingSessionWithItems:event:source:"), items, ev, source)
}

// endDrag is the end of the drag that the app started: the handler is told how it ended, and the session's
// source is released.
func (p *Platform) endDrag(s *dragSession, op uint64) {
	if p.drag != s {
		return
	}
	p.drag = nil
	if h := p.handler; h != nil {
		h.OnDragEnd(ui.DragResult{ID: s.id, Action: dragResultAction(op)})
	}
	objcx.Release(s.source)
	objcx.DropHandle(s.handle)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragInProgress = "DragInProgress"
	tagDragNoButton   = "DragNoButton"
	tagDragEmpty      = "DragEmpty"
)

// dragLocation is the place of a drag in the coordinates of the window (draggingLocation).
func (p *Platform) dragLocation(info objc.ID) nsPoint {
	if p.testDragLocation != nil {
		return p.testDragLocation(info)
	}
	return send[nsPoint](info, "draggingLocation")
}
