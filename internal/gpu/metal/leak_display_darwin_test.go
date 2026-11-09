//go:build darwin && !ios && glutter_display

package metal

import (
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// TestLeaksMetalResources makes and destroys buffers, textures and a frame with a copy and a read over and over (SPEC §15, "Memory
// over time", step 33.15) and checks that nothing that the process holds grows: the Go heap and goroutines (leaktest), the
// handles of the Objective-C registry, the blocks of C memory, the C callbacks of the process (they are never freed, so a
// scenario that makes one for each step would wear them out), and the bytes that the Metal device says it has allocated, which
// is where an object that the autorelease pool or a missing Release keeps alive shows. LEAKTEST_SCALE makes it longer.
func TestLeaksMetalResources(t *testing.T) {
	d := openTestDevice(t, Options{})
	allocated := func() int { return int(objc.Send[uint64](d.dev, objcx.Sel("currentAllocatedSize"))) }
	leaktest.Run(t, "metal buffers and textures", func() func() {
		return func() {
			objcx.WithPool(func() {
				a, err := d.NewBuffer(64<<10, explicit.BufferCopySource|explicit.BufferCopyDestination)
				if err != nil {
					t.Fatal(err)
				}
				b, err := d.NewBuffer(64<<10, explicit.BufferCopySource|explicit.BufferCopyDestination)
				if err != nil {
					t.Fatal(err)
				}
				tex, err := d.NewTexture(explicit.TextureDesc{Width: 128, Height: 128, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled | explicit.TextureCopySource})
				if err != nil {
					t.Fatal(err)
				}
				d.WriteBuffer(a, 0, make([]byte, 64<<10))
				enc, err := d.Begin()
				if err != nil {
					t.Fatal(err)
				}
				enc.CopyBuffer(a, 0, b, 0, 64<<10)
				enc.ReadBuffer(b, 0, 64<<10, func([]byte, error) {})
				if err := enc.Submit(); err != nil {
					t.Fatal(err)
				}
				tex.Destroy()
				a.Destroy()
				b.Destroy()
			})
		}
	}, leaktest.Opts{N: 100, Counters: map[string]func() int{
		"Objective-C handles":    objcx.HandleCount,
		"blocks of C memory":     func() int { return int(dl.LiveBlocks()) },
		"C callbacks":            dl.CallbackCount,
		"bytes the device holds": allocated,
	}, CounterSlack: 0})
}
