package fakedevice

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest/gotest"
)

// The scenes of every backend run on the fake device too, which draws nothing: that proves the
// descriptors they make are valid for the checks of the explicit package.
func TestScenesMakeValidDescriptors(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	devicetest.Run(gotest.Wrap(t), &devicetest.Env{Dev: d, Settle: d.CompleteReads})
}
