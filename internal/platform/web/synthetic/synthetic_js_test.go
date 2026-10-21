//go:build js && wasm

package synthetic

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// TestCanvasWithoutAPage: under Node there is no page, so there is no canvas.
func TestCanvasWithoutAPage(t *testing.T) {
	_, err := Canvas()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "SyntheticCanvasNotFound" || !appErr.HasRoute("Canvas.SyntheticCanvasNotFound") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
