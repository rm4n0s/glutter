package platform

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// plainHandler answers OnCloseRequest only.
type plainHandler struct {
	NopHandler
	views []ui.ViewID
}

func (h *plainHandler) OnCloseRequest(view ui.ViewID) bool {
	h.views = append(h.views, view)
	return false
}

// reasonHandler also answers OnCloseRequestFor.
type reasonHandler struct {
	plainHandler
	reasons []ui.AppExitReason
}

func (h *reasonHandler) OnCloseRequestFor(view ui.ViewID, reason ui.AppExitReason) bool {
	h.reasons = append(h.reasons, reason)
	return true
}

func TestCloseRequestUsesTheReasonWhenTheHandlerTakesIt(t *testing.T) {
	p := &plainHandler{}
	if CloseRequest(p, 3, ui.AppExitReasonSessionEnd) {
		t.Error("CloseRequest = true, want the answer of OnCloseRequest (false)")
	}
	if len(p.views) != 1 || p.views[0] != 3 {
		t.Errorf("OnCloseRequest views = %v, want [3]", p.views)
	}

	r := &reasonHandler{}
	if !CloseRequest(r, 4, ui.AppExitReasonQuit) {
		t.Error("CloseRequest = false, want the answer of OnCloseRequestFor (true)")
	}
	if len(r.reasons) != 1 || r.reasons[0] != ui.AppExitReasonQuit {
		t.Errorf("OnCloseRequestFor reasons = %v, want [AppExitReasonQuit]", r.reasons)
	}
	if len(r.views) != 0 {
		t.Errorf("OnCloseRequest was called too (%v)", r.views)
	}
}
