package platform

import (
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// NopHandler is a Handler that ignores every event and agrees to every close
// request. It is for tests and for the development examples that only need a
// window.
//
// Flutter: no counterpart
type NopHandler struct{}

var _ Handler = NopHandler{}

// OnPointer ignores the pointer data.
//
// Flutter: no counterpart
func (NopHandler) OnPointer([]ui.PointerData) {}

// OnKey ignores the key event.
//
// Flutter: no counterpart
func (NopHandler) OnKey(ui.KeyData) {}

// OnTextInput ignores the text-input event.
//
// Flutter: no counterpart
func (NopHandler) OnTextInput(ui.TextInputEvent) {}

// OnFrame ignores the frame request.
//
// Flutter: no counterpart
func (NopHandler) OnFrame(ui.ViewID, time.Duration) {}

// OnMetrics ignores the new metrics.
//
// Flutter: no counterpart
func (NopHandler) OnMetrics(ui.ViewID, ViewMetrics) {}

// OnLifecycle ignores the lifecycle change.
//
// Flutter: no counterpart
func (NopHandler) OnLifecycle(ui.AppLifecycleState) {}

// OnDrop ignores the drop event.
//
// Flutter: no counterpart
func (NopHandler) OnDrop(ui.ViewID, DropEvent) {}

// OnDisplaysChanged ignores the change.
//
// Flutter: no counterpart
func (NopHandler) OnDisplaysChanged() {}

// OnSettings ignores the new settings.
//
// Flutter: no counterpart
func (NopHandler) OnSettings(Settings) {}

// OnViewFocus ignores the focus change.
//
// Flutter: no counterpart
func (NopHandler) OnViewFocus(ui.ViewID, bool) {}

// OnCloseRequest agrees to close the view.
//
// Flutter: no counterpart
func (NopHandler) OnCloseRequest(ui.ViewID) bool { return true }
