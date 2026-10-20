package platform

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestNopHandlerAgreesToClose(t *testing.T) {
	var h Handler = NopHandler{}
	if !h.OnCloseRequest(0) {
		t.Error("NopHandler must agree to close requests")
	}
	// The rest must not panic.
	h.OnPointer(nil)
	h.OnTextInput(ui.TextInputEvent{})
	h.OnMetrics(0, ViewMetrics{})
	h.OnSettings(Settings{})
	h.OnLifecycle(0)
	h.OnDrop(ui.DropEvent{})
	h.OnDragEnd(ui.DragResult{})
	h.OnFrame(0, 0)
}
