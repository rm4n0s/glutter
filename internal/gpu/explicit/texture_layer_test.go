package explicit_test

import (
	"bytes"
	"encoding/binary"
	"log/slog"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/ui"
)

// textureRig is a renderer over a fake device with a registry, and a log.
type textureRig struct {
	r   *explicit.Renderer
	dev *fakedevice.Device
	reg *explicit.TextureRegistry
	log *bytes.Buffer
}

func newTextureRig(t *testing.T, opts ...explicit.RendererOpts) *textureRig {
	t.Helper()
	var o explicit.RendererOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	rig := &textureRig{reg: explicit.NewTextureRegistry(), log: &bytes.Buffer{}}
	o.Textures = rig.reg
	o.Log = slog.New(slog.NewTextHandler(rig.log, nil))
	r, dev, _ := newRenderer(t, o)
	rig.r, rig.dev = r, dev
	return rig
}

// register makes a texture of the format and registers it.
func (g *textureRig) register(t *testing.T, w, h int, format explicit.TextureFormat, levels int) int64 {
	t.Helper()
	tex, err := g.dev.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: format, MipLevels: levels, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	return g.reg.Register(g.dev, tex)
}

func textureScene(f func(b *ui.SceneBuilder)) *ui.Scene {
	b := ui.NewSceneBuilder()
	f(b)
	return b.Build()
}

// imageInstances returns the instances of the image pipeline that the last frame wrote, as floats.
func imageInstances(dev *fakedevice.Device) [][]float32 {
	var out [][]float32
	for _, w := range dev.Writes {
		if w.Usage&explicit.BufferVertex == 0 || len(w.Data)%(4*gpu.ImageFloats) != 0 || len(w.Data) == 0 {
			continue
		}
		for off := 0; off < len(w.Data); off += 4 * gpu.ImageFloats {
			in := make([]float32, gpu.ImageFloats)
			for i := range in {
				in[i] = math.Float32frombits(binary.LittleEndian.Uint32(w.Data[off+4*i:]))
			}
			out = append(out, in)
		}
	}
	return out
}

func TestATextureLayerIsDrawnAsAnImage(t *testing.T) {
	tests := []struct {
		name     string
		format   explicit.TextureFormat
		levels   int
		offset   ui.Offset
		w, h     float64
		quality  ui.FilterQuality
		linear   float32
		level, q float32
	}{
		{"an 8 bit texture", explicit.TextureFormatRGBA8Unorm, 1, ui.Offset{Dx: 10, Dy: 20}, 100, 50, ui.FilterQualityLow, 0, 0, 1},
		{"a BGRA texture with no filter", explicit.TextureFormatBGRA8Unorm, 1, ui.Offset{Dx: 0, Dy: 0}, 64, 32, ui.FilterQualityNone, 0, 0, 0},
		{"a half float texture is linear", explicit.TextureFormatRGBA16Float, 1, ui.Offset{Dx: 5, Dy: 5}, 32, 16, ui.FilterQualityLow, 1, 0, 1},
		{"medium quality scaled down picks a mip level", explicit.TextureFormatRGBA8Unorm, 7, ui.Offset{Dx: 0, Dy: 0}, 16, 8, ui.FilterQualityMedium, 0, 2, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTextureRig(t)
			id := g.register(t, 64, 32, tt.format, tt.levels)
			scene := textureScene(func(b *ui.SceneBuilder) {
				b.AddTexture(id, ui.AddTextureOpts{Offset: tt.offset, Width: tt.w, Height: tt.h, FilterQuality: tt.quality})
			})
			if _, err := g.r.Draw(window, 200, 100, scene, false); err != nil {
				t.Fatal(err)
			}
			if got := pipelinesOfScenePass(g.dev); !reflect.DeepEqual(got, []string{"image"}) {
				t.Fatalf("pipelines %v, want the image pipeline", got)
			}
			if st := g.r.LastFrame(); st.Instances != 1 {
				t.Fatalf("stats %+v, want one instance", st)
			}
			ins := imageInstances(g.dev)
			if len(ins) != 1 {
				t.Fatalf("%d image instances written", len(ins))
			}
			in := ins[0]
			// The box is the rectangle drawn, in pixels (the flag says so).
			box := [4]float32{float32(tt.offset.Dx), float32(tt.offset.Dy), float32(tt.offset.Dx + tt.w), float32(tt.offset.Dy + tt.h)}
			if got := [4]float32(in[20:24]); got != box {
				t.Errorf("box = %v, want %v", got, box)
			}
			if in[11] != tt.linear {
				t.Errorf("linear flag = %v, want %v", in[11], tt.linear)
			}
			if in[24] != tt.level || in[25] != tt.q {
				t.Errorf("level, quality = %v, %v; want %v, %v", in[24], in[25], tt.level, tt.q)
			}
			if g.log.Len() != 0 {
				t.Errorf("log: %s", g.log)
			}
		})
	}
}

func TestATextureLayerThatCannotBeDrawnDrawsNothingAndIsLoggedOnce(t *testing.T) {
	tests := []struct {
		name string
		make func(t *testing.T, g *textureRig) int64
		want string
	}{
		{"an id that is not registered", func(*testing.T, *textureRig) int64 { return 99 }, "not registered"},
		{"an unregistered texture", func(t *testing.T, g *textureRig) int64 {
			id := g.register(t, 8, 8, explicit.TextureFormatRGBA8Unorm, 1)
			g.reg.Unregister(id)
			return id
		}, "not registered"},
		{"a format the shader cannot read", func(t *testing.T, g *textureRig) int64 {
			return g.register(t, 8, 8, explicit.TextureFormatR8Unorm, 1)
		}, "format"},
		{"a texture of another device", func(t *testing.T, g *textureRig) int64 {
			other := fakedevice.New()
			tex, _ := other.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
			return g.reg.Register(other, tex)
		}, "not registered"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTextureRig(t)
			id := tt.make(t, g)
			scene := func() *ui.Scene {
				return textureScene(func(b *ui.SceneBuilder) { b.AddTexture(id, ui.AddTextureOpts{Width: 50, Height: 50}) })
			}
			for range 3 {
				if _, err := g.r.Draw(window, 100, 100, scene(), false); err != nil {
					t.Fatal(err)
				}
			}
			if got := pipelinesOfScenePass(g.dev); len(got) != 0 {
				t.Errorf("pipelines %v, want none", got)
			}
			if n := strings.Count(g.log.String(), "a texture layer draws nothing"); n != 1 {
				t.Errorf("logged %d times, want once:\n%s", n, g.log)
			}
			if !strings.Contains(g.log.String(), tt.want) {
				t.Errorf("the log lacks %q:\n%s", tt.want, g.log)
			}
		})
	}
}

func TestATextureLayerUnderAClipAndATransform(t *testing.T) {
	g := newTextureRig(t)
	id := g.register(t, 64, 64, explicit.TextureFormatRGBA8Unorm, 1)
	scene := textureScene(func(b *ui.SceneBuilder) {
		b.PushOffset(30, 40)
		b.PushClipRect(ui.NewRectFromLTWH(0, 0, 20, 20))
		b.AddTexture(id, ui.AddTextureOpts{Width: 64, Height: 64})
		b.Pop()
		b.Pop()
	})
	if _, err := g.r.Draw(window, 200, 200, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := g.r.LastFrame(); st.Instances != 1 {
		t.Fatalf("stats %+v", st)
	}
	ins := imageInstances(g.dev)
	if len(ins) != 1 {
		t.Fatalf("%d image instances", len(ins))
	}
	// The clip is the 20 by 20 square at (30, 40): left, top, right, bottom.
	if got := [4]float32(ins[0][16:20]); got != [4]float32{30, 40, 50, 60} {
		t.Errorf("clip = %v, want (30, 40, 50, 60)", got)
	}
}

func TestATextureOfALostDeviceIsNotDrawnByTheNextOne(t *testing.T) {
	g := newTextureRig(t)
	id := g.register(t, 8, 8, explicit.TextureFormatRGBA8Unorm, 1)
	if g.reg.Len() != 1 {
		t.Fatal("the texture is not registered")
	}
	scene := textureScene(func(b *ui.SceneBuilder) { b.AddTexture(id, ui.AddTextureOpts{Width: 8, Height: 8}) })
	if _, err := g.r.Draw(window, 20, 20, scene, false); err != nil {
		t.Fatal(err)
	}
	g.r.Close()
	if g.reg.Len() != 0 {
		t.Errorf("the registry still has %d textures of the closed device", g.reg.Len())
	}
	if _, ok := g.reg.Lookup(g.dev, id); ok {
		t.Errorf("the texture of the closed device is found")
	}
}

// TestALayerWithATextureIsNotKeptByTheRasterCache: a retained layer is drawn from its pixels after three frames, unless a
// texture layer is under it, since the texture changes without the scene changing (SPEC §27.4).
func TestALayerWithATextureIsNotKeptByTheRasterCache(t *testing.T) {
	tests := []struct {
		name        string
		texture     bool
		wantCaching bool
	}{
		{"a picture", false, true},
		{"a texture", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTextureRig(t, explicit.RendererOpts{RasterCacheAfter: 3})
			id := g.register(t, 8, 8, explicit.TextureFormatRGBA8Unorm, 1)
			pic := pictureOfRect(ui.NewRectFromLTWH(0, 0, 30, 30))
			first := ui.NewSceneBuilder()
			handle := first.PushOffset(5, 5)
			if tt.texture {
				first.AddTexture(id, ui.AddTextureOpts{Width: 30, Height: 30})
			} else {
				first.AddPicture(ui.Offset{}, pic)
			}
			first.Pop()
			scenes := []*ui.Scene{first.Build()}
			for range 5 {
				b := ui.NewSceneBuilder()
				b.AddRetained(handle)
				scenes = append(scenes, b.Build())
			}
			made, drawn := 0, 0
			for _, s := range scenes {
				if _, err := g.r.Draw(window, 100, 100, s, false); err != nil {
					t.Fatal(err)
				}
				made += g.r.LastFrame().CachedLayersMade
				drawn += g.r.LastFrame().CachedLayersDrawn
			}
			if cached := made+drawn > 0; cached != tt.wantCaching {
				t.Errorf("cached layers made %d, drawn %d; want caching %v", made, drawn, tt.wantCaching)
			}
		})
	}
}

func pictureOfRect(r ui.Rect) *ui.Picture {
	rec := ui.NewPictureRecorder()
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0xFF336699))
	ui.NewCanvas(rec).DrawRect(r, p)
	return rec.EndRecording()
}
