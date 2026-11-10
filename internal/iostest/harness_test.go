package iostest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// The routes of the failures of the harness that need no Simulator.

func TestWaitForFailsWhenTheTimeIsOut(t *testing.T) {
	l := &Launched{lines: make(chan string, 4), done: make(chan struct{}), output: []string{"a line"}}
	_, err := l.WaitFor("never", 20*time.Millisecond)
	requireRoute(t, err, "Launched.WaitFor."+TagIOSSimulatorFailed)
	if appErr, _ := errors.FromError(err); appErr.Metadata["output"] != "a line" {
		t.Errorf("the output is not in the error: %v", appErr.Metadata)
	}
}

func TestWaitForFailsWhenTheAppEnds(t *testing.T) {
	l := &Launched{lines: make(chan string, 4), done: make(chan struct{})}
	close(l.done)
	_, err := l.WaitFor("never", time.Minute)
	requireRoute(t, err, "Launched.WaitFor."+TagIOSSimulatorFailed)
}

func TestWaitForFindsTheLinesThatCameBefore(t *testing.T) {
	l := &Launched{lines: make(chan string, 4), done: make(chan struct{}), output: []string{"one", "two three"}}
	if line, err := l.WaitFor("three", time.Second); err != nil || line != "two three" {
		t.Errorf("WaitFor = %q, %v", line, err)
	}
	l.lines <- "late"
	l.output = append(l.output, "late")
	if line, err := l.WaitFor("late", time.Second); err != nil || line != "late" {
		t.Errorf("WaitFor of a line that came after = %q, %v", line, err)
	}
}

func TestADeviceOfAnUnknownFamilyFails(t *testing.T) {
	h := &Harness{sim: iostool.Simulator{}, booted: map[string]string{}}
	_, err := h.Device(t.Context(), "watch")
	requireRoute(t, err, "Device->Simulator.Device."+iostool.TagIOSSimctlFailed)
}

func TestModuleRootFindsTheGoMod(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("root = %s has no go.mod: %v", root, err)
	}
}

func requireRoute(t *testing.T, err error, route string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error, want the route %s", route)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("the error is a %T: %v", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("route = %s, want it to contain %s", appErr.Route(), route)
	}
}
