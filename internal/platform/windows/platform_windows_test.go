//go:build windows

package windows

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

func assertRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func TestStartDragIsUnsupportedUntilStep28_12(t *testing.T) {
	err := (&window{}).StartDrag(ui.DragRequest{})
	assertRoute(t, err, "DragUnsupported", "window.StartDrag.DragUnsupported")
}

func TestTheCallbacksOfTheWindowsStayUnderTheLimit(t *testing.T) {
	_ = wndProcCallback()
	if n := win32.CallbackCount(); n > 1500 {
		t.Errorf("%d callbacks", n)
	}
}
