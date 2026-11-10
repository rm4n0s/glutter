package ios

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

func TestStartDragIsUnsupported(t *testing.T) {
	err := unsupportedWindow{}.StartDrag(ui.DragRequest{})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("the error is a %T: %v", err, err)
	}
	if !appErr.HasRoute("unsupportedWindow.StartDrag." + tagIOSDragUnsupported) {
		t.Errorf("route = %s", appErr.Route())
	}
}
