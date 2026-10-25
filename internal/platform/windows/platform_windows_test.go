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

// A platform whose OLE objects could not be made (p.ole is nil) starts no drag: the drag stays in the
// window, as on Linux without a data device (SPEC §25.10).
func TestStartDragWithoutOLEIsUnsupported(t *testing.T) {
	err := (&window{p: &Platform{}}).StartDrag(ui.DragRequest{})
	assertRoute(t, err, "DragUnsupported", "window.StartDrag.DragUnsupported")
}

func TestTheCallbacksOfTheWindowsStayUnderTheLimit(t *testing.T) {
	_ = wndProcCallback()
	if n := win32.CallbackCount(); n > 1500 {
		t.Errorf("%d callbacks", n)
	}
}
