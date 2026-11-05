// Package gotest adapts *testing.T to devicetest.T. It imports testing, so only tests import it.
//
// Flutter: no counterpart
package gotest

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
)

// Wrap returns the devicetest.T of a test.
//
// Flutter: no counterpart
func Wrap(t *testing.T) devicetest.T { return wrapped{t} }

type wrapped struct{ *testing.T }

// Run implements devicetest.T.
func (w wrapped) Run(name string, fn func(t devicetest.T)) {
	w.T.Run(name, func(t *testing.T) { fn(wrapped{t}) })
}
