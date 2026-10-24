package engine

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

var (
	_ platform.BackHandler = (*handler)(nil)
	_ ui.BackHost          = (*handler)(nil)
)

// SetFrameworkHandlesBack tells the window of the view whether the framework wants the back gesture
// (ui.BackHost); a window that cannot choose (the desktop's) ignores it.
func (h *handler) SetFrameworkHandlesBack(handles bool) {
	w := h.windowOf(ui.ImplicitViewID)
	if w == nil {
		return
	}
	if b, ok := w.Window.(ui.BackHost); ok {
		b.SetFrameworkHandlesBack(handles)
	}
}

// OnPopRoute reports that the system asks the app to go back (platform.BackHandler).
func (h *handler) OnPopRoute() { h.disp.PopRoute() }

// OnStartBackGesture reports the start of a predictive back gesture and returns whether the
// framework took it (platform.BackHandler).
func (h *handler) OnStartBackGesture(e ui.BackGestureEvent) bool { return h.disp.StartBackGesture(e) }

// OnUpdateBackGestureProgress reports the progress of a predictive back gesture
// (platform.BackHandler).
func (h *handler) OnUpdateBackGestureProgress(e ui.BackGestureEvent) {
	h.disp.UpdateBackGestureProgress(e)
}

// OnCommitBackGesture reports that a predictive back gesture was completed (platform.BackHandler).
func (h *handler) OnCommitBackGesture() { h.disp.CommitBackGesture() }

// OnCancelBackGesture reports that a predictive back gesture was cancelled (platform.BackHandler).
func (h *handler) OnCancelBackGesture() { h.disp.CancelBackGesture() }

// OnMemoryPressure reports that the system is low on memory (platform.BackHandler).
func (h *handler) OnMemoryPressure() { h.disp.NotifyMemoryPressure() }
