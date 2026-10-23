//go:build linux

package linux

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Backend names a windowing system.
//
// Flutter: no counterpart
type Backend string

// The backends of Linux.
const (
	// BackendWayland is the Wayland backend.
	//
	// Flutter: no counterpart
	BackendWayland Backend = "wayland"
	// BackendX11 is the X11 backend.
	//
	// Flutter: no counterpart
	BackendX11 Backend = "x11"
)

// BackendEnv names the environment variable that chooses the backend: "wayland"
// or "x11" (SPEC §6.3 rule 5).
const BackendEnv = "GLUTTER_BACKEND"

// Connection is a platform that is connected to its display server and can be
// closed.
//
// Flutter: no counterpart
type Connection interface {
	platform.Platform
	// Close releases the connection and everything that depends on it.
	Close() error
}

// Selection says how to choose a backend. The two connect functions are the
// backends themselves (the engine passes wayland.New and x11.New; package linux
// cannot import them), so the choice can be tested with fakes.
//
// Flutter: no counterpart
type Selection struct {
	// Backend, when not empty, is the backend to use and overrides the
	// environment variable (RunAppOpts.Backend, SPEC §13).
	Backend string
	// Getenv reads the environment; os.Getenv when nil.
	Getenv func(string) string
	// Wayland connects to the Wayland compositor. It must fail, not hang, when
	// the compositor or a library is missing.
	Wayland func() (Connection, error)
	// X11 connects to the X server.
	X11 func() (Connection, error)
	// Logger gets the record of the choice; slog.Default() when nil.
	Logger *slog.Logger
}

// Select chooses and connects a backend (SPEC §6.3 rule 5):
//
//  1. The backend named by Selection.Backend or $GLUTTER_BACKEND, if any. It is
//     the only one tried, and its failure is returned unchanged.
//  2. Wayland, when $WAYLAND_DISPLAY is set and the connection and its libraries
//     work. A failure is logged and the next one is tried.
//  3. X11, when $DISPLAY is set and the connection works.
//  4. Otherwise NoDisplay, which carries what was tried and why it failed.
//
// The log says which backend was chosen and why. A value that is not "wayland" or
// "x11" fails with BackendUnknown.
//
// Flutter: no counterpart
func Select(s Selection) (Connection, Backend, error) {
	getenv := s.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	log := s.Logger
	if log == nil {
		log = slog.Default()
	}

	name, source := s.Backend, "the Backend option"
	if name == "" {
		name, source = getenv(BackendEnv), BackendEnv
	}
	switch Backend(name) {
	case BackendWayland:
		return connect(log, BackendWayland, s.Wayland, source+"="+name)
	case BackendX11:
		return connect(log, BackendX11, s.X11, source+"="+name)
	case "":
	default:
		return nil, "", errors.New(tagBackendUnknown, "the backend must be wayland or x11",
			"value", name, "from", source)
	}

	var tried []string
	if display := getenv("WAYLAND_DISPLAY"); display != "" {
		conn, b, err := connect(log, BackendWayland, s.Wayland, "WAYLAND_DISPLAY="+display)
		if err == nil {
			return conn, b, nil
		}
		log.Warn("Wayland is not usable", "WAYLAND_DISPLAY", display, "err", err)
		tried = append(tried, "wayland: "+describe(err))
	}
	if display := getenv("DISPLAY"); display != "" {
		reason := "DISPLAY=" + display
		if len(tried) > 0 {
			reason += ", after Wayland failed"
		}
		conn, b, err := connect(log, BackendX11, s.X11, reason)
		if err == nil {
			return conn, b, nil
		}
		log.Warn("X11 is not usable", "DISPLAY", display, "err", err)
		tried = append(tried, "x11: "+describe(err))
	}
	return nil, "", errors.New(tagNoDisplay, "no Wayland compositor or X server could be used",
		"tried", tried, "WAYLAND_DISPLAY", getenv("WAYLAND_DISPLAY"), "DISPLAY", getenv("DISPLAY"))
}

// connect runs a backend's connect function and logs the choice. A backend that
// the caller does not provide fails with BackendUnavailable.
func connect(log *slog.Logger, b Backend, fn func() (Connection, error), why string) (Connection, Backend, error) {
	if fn == nil {
		return nil, "", errors.New(tagBackendUnavailable, "this build has no such backend", "backend", string(b))
	}
	conn, err := fn()
	if err != nil {
		return nil, "", err
	}
	log.Info("backend chosen", "backend", string(b), "because", why)
	return conn, b, nil
}

// describe summarizes an error for the list of what was tried: its tag and
// message when it has them.
func describe(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		return fmt.Sprintf("%s: %s", appErr.Tag, appErr.Message)
	}
	return err.Error()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendUnavailable = "BackendUnavailable"
	tagBackendUnknown     = "BackendUnknown"
	tagNoDisplay          = "NoDisplay"
)
