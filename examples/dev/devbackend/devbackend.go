// Package devbackend picks the platform backend for the development examples,
// until the engine does it (step 1.10). It reads GLUTTER_DEV_BACKEND: "wayland"
// or "x11". Without it, Wayland is used when WAYLAND_DISPLAY is set, else X11
// when DISPLAY is set, else Wayland (which then reports that there is no
// compositor).
//
// Flutter: no counterpart
package devbackend

import (
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/internal/platform/linux/x11"
)

// BackendEnv names the environment variable that chooses the backend.
const BackendEnv = "GLUTTER_DEV_BACKEND"

// Backend is a platform that can be closed.
//
// Flutter: no counterpart
type Backend interface {
	platform.Platform
	// Close releases the connection.
	Close() error
}

// Name returns the backend that New will use, given getenv (os.Getenv in an
// app): "wayland" or "x11".
//
// Flutter: no counterpart
func Name(getenv func(string) string) (string, error) {
	switch v := getenv(BackendEnv); v {
	case "wayland", "x11":
		return v, nil
	case "":
		if getenv("WAYLAND_DISPLAY") == "" && getenv("DISPLAY") != "" {
			return "x11", nil
		}
		return "wayland", nil
	default:
		return "", errors.New("DevBackendUnknown", "GLUTTER_DEV_BACKEND must be wayland or x11", "value", v)
	}
}

// New connects to the display server. shm makes a Wayland window draw through a
// wl_shm buffer instead of EGL; the X11 backend has no such mode and fails with
// DevBackendUnsupported.
//
// Flutter: no counterpart
func New(shm bool) (Backend, error) {
	name, err := Name(os.Getenv)
	if err != nil {
		return nil, err
	}
	if name == "x11" {
		if shm {
			return nil, errors.New("DevBackendUnsupported", "the shared-memory mode is for Wayland only")
		}
		return x11.New("")
	}
	return wayland.New("", wayland.Options{ShmFill: shm})
}
