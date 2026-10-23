//go:build !js

package androidtool

import (
	"context"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestExecRunnerFailureRoute(t *testing.T) {
	_, err := ExecRunner{}.Run(context.Background(), Command{Name: "false"})
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("ExecRunner.Run."+TagAndroidCommandFailed) {
		t.Fatalf("got %v", err)
	}
	if appErr.Metadata["command"] != "false" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
	out, err := ExecRunner{}.Run(context.Background(), Command{Name: "echo", Args: []string{"hi"}})
	if err != nil || strings.TrimSpace(out) != "hi" {
		t.Errorf("echo: %q %v", out, err)
	}
}
