//go:build linux && glutter_display

package vulkan

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest/gotest"
)

// TestScenes runs the scenes that every backend runs (SPEC §27.12) on lavapipe with the
// validation layer on: each format is written and read back, depth, stencil, multisampling, texture
// kinds, four attachments, the clip space and the vertex formats.
func TestScenes(t *testing.T) {
	d := openLavapipe(t)
	devicetest.Run(gotest.Wrap(t), &devicetest.Env{Dev: d, Pixels: true})
}
