//go:build linux && !android

package engine

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
)

// newSelectHarness is newHarness for a test of the choice between Wayland and X11:
// the engine chooses the backend, and the X11 backend is the fake platform.
func newSelectHarness(t *testing.T, env map[string]string) *harness {
	t.Helper()
	h := newHarness(t, env)
	h.cfg.connectors = map[string]func() (connection, error){
		"x11": func() (connection, error) { return h.c, nil },
		"wayland": func() (connection, error) {
			return nil, errors.New(tagWaylandConnectFailed, "no compositor")
		},
	}
	return h
}

func TestRunSelectsTheBackend(t *testing.T) {
	t.Run("no display", func(t *testing.T) {
		h := newSelectHarness(t, nil)
		assertRoute(t, Run(context.Background(), h.cfg), "NoDisplay", "Run->connectDisplay->Select.NoDisplay")
	})
	t.Run("an unknown backend", func(t *testing.T) {
		h := newSelectHarness(t, map[string]string{"DISPLAY": ":0"})
		h.cfg.Backend = "quartz"
		assertRoute(t, Run(context.Background(), h.cfg), "BackendUnknown", "Run->connectDisplay->Select.BackendUnknown")
	})
	t.Run("the backend's own error comes back unchanged", func(t *testing.T) {
		h := newSelectHarness(t, map[string]string{"WAYLAND_DISPLAY": "wayland-0"})
		h.cfg.Backend = "wayland"
		assertRoute(t, Run(context.Background(), h.cfg), "WaylandConnectFailed", "Run->connectDisplay->Select->connect->")
	})
}

func TestShmFillRefusedByTheX11Connector(t *testing.T) {
	cfg := Config{ShmFill: true}
	_, err := cfg.x11Connector()()
	assertRoute(t, err, "ShmFillUnsupported", "ShmFillUnsupported")
}

func TestRealConnectorsReportTheirFailures(t *testing.T) {
	// No server is listening: the real connectors fail with their own tags, as a
	// session with a broken DISPLAY or WAYLAND_DISPLAY would.
	t.Setenv("DISPLAY", ":9999")
	t.Setenv("WAYLAND_DISPLAY", "glutter-no-such-socket")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	cfg := Config{Logger: quiet()}
	_, err := cfg.x11Connector()()
	assertRoute(t, err, "X11OpenDisplayFailed", "New.X11OpenDisplayFailed")
	_, err = cfg.waylandConnector()()
	if appErr, ok := errors.FromError(err); !ok || (appErr.Tag != tagWaylandConnectFailed && appErr.Tag != "LibraryLoadFailed") {
		t.Errorf("Wayland error = %v, want WaylandConnectFailed (or LibraryLoadFailed without libwayland)", err)
	}
	// With both broken, Run says there is no display.
	err = Run(context.Background(), Config{Logger: quiet()})
	assertRoute(t, err, "NoDisplay", "Run->connectDisplay->Select.NoDisplay")
	appErr, _ := errors.FromError(err)
	if tried, _ := appErr.Metadata["tried"].([]string); len(tried) != 2 {
		t.Errorf("tried %v, want Wayland and X11", appErr.Metadata["tried"])
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWaylandConnectFailed = "WaylandConnectFailed"
)
