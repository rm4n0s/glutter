//go:build windows

package windows

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/ole"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the glue of drag and drop (SPEC §25.10): each window is registered as a drop target with
// OLE, and a drag that the app starts runs DoDragDrop in a posted function. The rules are dnd.go's.

// oleRuntime is what the COM objects of the platform need: its log and its guard, so that a panic of the
// handler inside a method that Windows called does not unwind through Windows' frames.
func (p *Platform) oleRuntime() ole.Runtime {
	return ole.Runtime{Log: p.log, Guard: p.guard}
}

// registerDrop makes the window a drop target (RegisterDragDrop). A failure is logged: the window
// takes no drops, and the rest works.
func (w *window) registerDrop() {
	p := w.p
	if p.ole == nil {
		return
	}
	d := &dropWindow{
		view:    w.id,
		handler: func() platform.Handler { return p.handler },
		toClient: func(pt win32.Point) (ui.Offset, bool) {
			c, ok := win32.ScreenToClient(w.hwnd, pt)
			return ui.Offset{Dx: float64(c.X), Dy: float64(c.Y)}, ok
		},
		nameOf: win32.GetClipboardFormatName,
		pid:    ole.PID(), now: time.Now, budget: dropReadBudget, log: p.log,
	}
	target, err := p.ole.NewDropTarget(d, p.oleRuntime())
	if err != nil {
		p.log.Warn("the window cannot take drops", "err", err)
		return
	}
	if err := ole.RegisterDragDrop(w.hwnd, target); err != nil {
		p.log.Warn("the window cannot take drops", "err", err)
		return
	}
	w.dropOK = true
}

// revokeDrop takes the drop target away from the window; OLE frees the object. It runs on WM_DESTROY,
// while the window still exists.
func (w *window) revokeDrop() {
	if !w.dropOK {
		return
	}
	w.dropOK = false
	if err := win32.RevokeDragDrop(w.hwnd); err != nil {
		w.p.log.Debug("the drop target of a window was not revoked", "err", err)
	}
}

// drags is the runner of the drags the app starts, made by the first one.
func (p *Platform) drags() *dragRunner {
	if p.drag == nil {
		p.drag = &dragRunner{
			pid: ole.PID(), log: p.log,
			buttons:  func() uint32 { return ole.ButtonKeys(p.pointers.mouseButtons()) },
			register: win32.RegisterClipboardFormat,
			post:     p.Post,
			loop:     p.dragLoop,
			released: p.dragReleased,
			end: func(result ui.DragResult) {
				if h := p.handler; h != nil {
					h.OnDragEnd(result)
				}
			},
		}
	}
	return p.drag
}

// dragLoop runs the modal loop of a drag from the window of the view: the objects are made, DoDragDrop
// runs with the timer of the window going (the loop of OLE dispatches the posted functions and the frames
// through it, SPEC §25.3), and the objects are released.
func (p *Platform) dragLoop(view ui.ViewID, data *ole.DataSet, machine *ole.SourceMachine, allowed uint32) (uint32, bool, error) {
	rt := p.oleRuntime()
	dataObj, err := p.ole.NewDataObject(data, rt)
	if err != nil {
		return 0, false, err
	}
	defer dataObj.Release()
	source, err := p.ole.NewDropSource(machine, rt)
	if err != nil {
		return 0, false, err
	}
	defer source.Release()
	if w := p.windowByView(view); w != nil && !w.closed && !w.modalTimer {
		w.startModalTimer() // a timer that runs already (a menu around this loop) is left to its owner
		defer w.stopModalTimer()
	}
	return win32.DoDragDrop(dataObj, source, allowed)
}

// dragReleased lifts the mouse of the window the drag started from: the button went up inside the loop
// of OLE, which had the capture, so the window never heard of it. The pointer is released with a
// synthesized up (SPEC §25.10).
func (p *Platform) dragReleased(view ui.ViewID) {
	w := p.windowByView(view)
	if w == nil || w.mouse == nil {
		return
	}
	w.mouse.captureChanged(false)
	w.flushPointers()
	if win32.GetCapture() == w.hwnd {
		win32.ReleaseCapture()
	}
}
