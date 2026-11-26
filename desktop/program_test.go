package desktop

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// TestRunProgramCancellation: the program that CommandContext kills when its context ends is an error of
// ProgramFailed with the cancellation as its cause; a program that fails with a live context is not.
func TestRunProgramCancellation(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("no sleep program")
	}
	if _, err := exec.LookPath("false"); err != nil {
		t.Skip("no false program")
	}
	t.Run("killed with its context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(50*time.Millisecond, cancel)
		err := runProgram(ctx, nil, "sleep", "30")
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagProgramFailed || !appErr.HasRoute("runProgram."+tagProgramFailed) {
			t.Fatalf("got %v, want ProgramFailed made by runProgram", err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("errors.Is(err, context.Canceled) is false for %v", err)
		}
		if appErr.Metadata["program"] != "sleep" || appErr.Metadata["reason"] == nil {
			t.Errorf("metadata = %v, want the program and the reason", appErr.Metadata)
		}
	})
	t.Run("failing with a live context", func(t *testing.T) {
		err := runProgram(context.Background(), nil, "false")
		if _, ok := errors.FromError(err); !ok || errors.Is(err, context.Canceled) {
			t.Errorf("got %v, want an error that is no cancellation", err)
		}
	})
}
