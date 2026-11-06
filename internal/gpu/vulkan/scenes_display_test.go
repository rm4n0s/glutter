//go:build linux && glutter_display

package vulkan

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest/gotest"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// TestScenes runs the scenes that every backend runs (SPEC §27.12) on lavapipe with the
// validation layer on: each format is written and read back, depth, stencil, multisampling, texture
// kinds, four attachments, the clip space and the vertex formats, and (step 30.2) compute, storage
// resources, indirect calls, the copies, several encoders and a thousand frames of readback. The
// memory of the device counts the C heap and the blocks of device memory.
func TestScenes(t *testing.T) {
	d := openLavapipe(t)
	memory := func() int64 {
		n, _ := dl.CHeapInUse()
		for _, blocks := range d.alloc.blocks {
			for _, b := range blocks {
				n += int64(b.size)
			}
		}
		return n
	}
	devicetest.Run(gotest.Wrap(t), &devicetest.Env{Dev: d, Pixels: true, Memory: memory})
}
