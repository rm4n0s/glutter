package engine

import "github.com/rm4n0s/glutter/internal/platform"

// connection is a platform that is connected to its display server and can be
// closed. The backends_<goos>.go files make one for Run.
//
// Flutter: no counterpart
type connection interface {
	platform.Platform
	// Close releases the connection and everything that depends on it.
	Close() error
}
