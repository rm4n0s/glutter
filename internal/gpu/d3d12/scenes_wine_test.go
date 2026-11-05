//go:build windows && glutter_wine

package d3d12

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest/gotest"
)

// TestScenes runs the scenes that every backend runs (SPEC §27.12) on Direct3D 12 under Wine (vkd3d
// over lavapipe): each format is written and read back, depth, stencil, multisampling, texture kinds,
// four attachments, the clip space and the vertex formats, with nothing skipped and no error from the
// debug layer when it is installed.
func TestScenes(t *testing.T) {
	d, rec := openTestDevice(t, Options{Debug: true})
	devicetest.Run(gotest.Wrap(t), &devicetest.Env{Dev: d, Pixels: true})
	if errs := d.DebugErrors(); len(errs) != 0 {
		t.Errorf("the debug layer reported %d errors: %q", len(errs), errs)
	}
	_ = rec
}
