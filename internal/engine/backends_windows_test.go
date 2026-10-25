//go:build windows

package engine

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestRunHasNoBackendOnWindowsYet(t *testing.T) {
	err := Run(context.Background(), Config{Logger: quiet()})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "BackendUnavailable" || !appErr.HasRoute("Run->connectDisplay.BackendUnavailable") {
		t.Errorf("got %s via %s, want BackendUnavailable via Run->connectDisplay", appErr.Tag, appErr.Route())
	}
}
