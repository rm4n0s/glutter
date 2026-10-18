package platform

import "testing"

func TestNopHandlerAgreesToClose(t *testing.T) {
	var h Handler = NopHandler{}
	if !h.OnCloseRequest(0) {
		t.Error("NopHandler must agree to close requests")
	}
	// The rest must not panic.
	h.OnPointer(nil)
	h.OnTextInput(TextInputEvent{})
	h.OnMetrics(0, ViewMetrics{})
	h.OnSettings(Settings{})
	h.OnLifecycle(0)
	h.OnDrop(0, DropEvent{})
	h.OnFrame(0, 0)
}
