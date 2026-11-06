//go:build windows && glutter_wine

package d3d12

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// The Direct3D 12 counterpart of vulkan's texture_display_test.go (step 30.5): a registered texture shown under a
// rotated, clipped, half transparent layer matches the software renderer's image of the same pixels.

func TestWineATextureLayerMatchesTheImage(t *testing.T) {
	for _, tt := range gputest.TextureCases() {
		t.Run(tt.Name, func(t *testing.T) {
			reg := explicit.NewTextureRegistry()
			g := newRig(t, Options{}, func(o *explicit.RendererOpts) { o.Textures = reg })
			// The first frame opens the device.
			if _, err := g.r.Draw(gpu.SurfaceHandle{}, gputest.TextureWindow, gputest.TextureWindow, ui.NewSceneBuilder().Build(), false); err != nil {
				t.Fatalf("Draw: %+v", err)
			}
			dev := explicit.Device(offscreenDevice{g.dev()})
			format := map[gputest.TextureKind]explicit.TextureFormat{gputest.TextureRGBA8: explicit.TextureFormatRGBA8Unorm,
				gputest.TextureBGRA8: explicit.TextureFormatBGRA8Unorm, gputest.TextureRGBA16F: explicit.TextureFormatRGBA16Float}[tt.Kind]
			tex, err := dev.NewTexture(explicit.TextureDesc{Width: gputest.TextureSize, Height: gputest.TextureSize, Format: format, MipLevels: 1, Usage: explicit.TextureSampled})
			if err != nil {
				t.Fatal(err)
			}
			dev.WriteTexture(tex, 0, image.Rect(0, 0, gputest.TextureSize, gputest.TextureSize), tt.Texels, gputest.TextureSize*tt.Kind.TexelBytes())
			id := reg.Register(dev, tex)
			d := g.draw(gputest.TextureScene(id, tt.Quality), gputest.TextureWindow, gputest.TextureWindow, false)
			if d.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", d.stats.Skipped, d.stats.Unsupported)
			}
			want, err := soft.Render(gputest.ImageScene(tt.Reference, tt.Quality), gputest.TextureWindow, gputest.TextureWindow)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "d3d12-texture-"+slug(tt.Name)+"-gpu", d.got)
			dumpImage(t, "d3d12-texture-"+slug(tt.Name)+"-soft", want)
			tt.Tolerance.Check(t, tt.Name, gputest.DiffRect(d.got, want, d.got.Bounds()))
		})
	}
}
