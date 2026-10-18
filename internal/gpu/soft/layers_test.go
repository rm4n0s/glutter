package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

func rectAt(x, y, w, h float64) ui.Rect { return ui.NewRectFromLTWH(x, y, w, h) }

func blended(argb uint32, mode ui.BlendMode) *ui.Paint {
	p := paint(argb)
	p.SetBlendMode(mode)
	return p
}

func TestBlendModesThroughPaint(t *testing.T) {
	// A source over a destination: the surface pixel and the shape's own pixel
	// follow the formulas, and outside the shape nothing changes.
	tests := []struct {
		name string
		mode ui.BlendMode
		want [4]uint8 // inside the overlap
	}{
		{"srcOver", ui.BlendModeSrcOver, [4]uint8{128, 0, 127, 255}},
		{"src", ui.BlendModeSrc, [4]uint8{128, 0, 0, 128}},
		{"dst", ui.BlendModeDst, [4]uint8{0, 0, 255, 255}},
		{"clear", ui.BlendModeClear, [4]uint8{0, 0, 0, 0}},
		{"srcIn", ui.BlendModeSrcIn, [4]uint8{128, 0, 0, 128}},
		{"dstOut", ui.BlendModeDstOut, [4]uint8{0, 0, 127, 127}},
		{"plus", ui.BlendModePlus, [4]uint8{128, 0, 255, 255}},
		{"multiply", ui.BlendModeMultiply, [4]uint8{0, 0, 127, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 8, 4, func(c *ui.Canvas) {
				c.DrawRect(rectAt(0, 0, 4, 4), paint(0xFF0000FF)) // an opaque blue destination
				c.DrawRect(rectAt(0, 0, 4, 4), blended(0x80FF0000, tt.mode))
			})
			got := px(img, 2, 2)
			for k := range got {
				if absInt(int(got[k])-int(tt.want[k])) > 1 {
					t.Errorf("overlap = %v, want %v", got, tt.want)
					break
				}
			}
			if px(img, 6, 2) != [4]uint8{} {
				t.Errorf("outside the shapes = %v", px(img, 6, 2))
			}
		})
	}
	// With partial coverage the result is mixed with what was there: a clear
	// over half a pixel leaves half.
	img := renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.DrawRect(rectAt(0, 0, 4, 4), paint(0xFF0000FF))
		c.DrawRect(rectAt(1.5, 0, 1, 4), blended(0xFF000000, ui.BlendModeClear))
	})
	if got := px(img, 1, 1)[3]; got < 126 || got > 129 {
		t.Errorf("half-cleared pixel alpha = %d, want 128", got)
	}
	if got := px(img, 2, 1)[3]; got < 126 || got > 129 {
		t.Errorf("half-cleared pixel alpha = %d, want 128", got)
	}
	if got := px(img, 0, 1)[3]; got != 255 {
		t.Errorf("untouched pixel alpha = %d", got)
	}
}

func TestDrawColorBlendModes(t *testing.T) {
	img := renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.DrawColor(ui.NewColor(0xFF0000FF), ui.BlendModeSrcOver)
		c.ClipRect(rectAt(0, 0, 2, 4))
		c.DrawColor(ui.NewColor(0xFFFF0000), ui.BlendModeClear)
	})
	if px(img, 1, 1) != [4]uint8{} {
		t.Errorf("cleared = %v", px(img, 1, 1))
	}
	if px(img, 3, 1) != [4]uint8{0, 0, 255, 255} {
		t.Errorf("outside the clip = %v", px(img, 3, 1))
	}
}

func TestColorFilterAndInvertOnPaint(t *testing.T) {
	filtered := paint(0xFFFF0000)
	filtered.SetColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeSrc))
	inverted := paint(0xFFFF8000)
	inverted.SetInvertColors(true)
	both := paint(0xFF000000)
	both.SetColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrc))
	both.SetInvertColors(true)
	img := renderOf(t, 12, 4, func(c *ui.Canvas) {
		c.DrawRect(rectAt(0, 0, 4, 4), filtered)
		c.DrawRect(rectAt(4, 0, 4, 4), inverted)
		c.DrawRect(rectAt(8, 0, 4, 4), both)
	})
	if got := px(img, 1, 1); got != [4]uint8{0, 255, 0, 255} {
		t.Errorf("color filter = %v", got)
	}
	if got := px(img, 5, 1); absInt(int(got[0])-0) > 1 || absInt(int(got[1])-127) > 1 || absInt(int(got[2])-255) > 1 || got[3] != 255 {
		t.Errorf("inverted = %v", got)
	}
	// The filter runs first, and the inversion after it: white, then black.
	if got := px(img, 9, 1); got != [4]uint8{0, 0, 0, 255} {
		t.Errorf("filter then invert = %v", got)
	}
}

func TestSaveLayer(t *testing.T) {
	// Two overlapping opaque rects in a half-opaque layer are uniformly half
	// opaque: the layer is composited as a whole. Drawn separately at half
	// opacity, the overlap would be darker.
	grouped := renderOf(t, 12, 4, func(c *ui.Canvas) {
		c.SaveLayer(nil, paint(0x80000000))
		c.DrawRect(rectAt(0, 0, 8, 4), paint(0xFF000000))
		c.DrawRect(rectAt(4, 0, 8, 4), paint(0xFF000000))
		c.Restore()
	})
	for _, x := range []int{1, 6, 10} {
		if got := px(grouped, x, 1)[3]; got < 127 || got > 129 {
			t.Errorf("layer alpha at %d = %d, want 128", x, got)
		}
	}
	separate := renderOf(t, 12, 4, func(c *ui.Canvas) {
		c.DrawRect(rectAt(0, 0, 8, 4), paint(0x80000000))
		c.DrawRect(rectAt(4, 0, 8, 4), paint(0x80000000))
	})
	if got := px(separate, 6, 1)[3]; got < 190 {
		t.Errorf("separate draws overlap alpha = %d, want about 192", got)
	}
}

func TestSaveLayerBoundsAndClip(t *testing.T) {
	img := renderOf(t, 12, 8, func(c *ui.Canvas) {
		bounds := rectAt(2, 2, 4, 4)
		c.SaveLayer(&bounds, nil)
		c.DrawRect(rectAt(0, 0, 12, 8), paint(0xFF00FF00)) // clipped to the layer's bounds
		c.Restore()
		c.Save()
		c.ClipRect(rectAt(8, 0, 4, 8))
		c.SaveLayer(nil, nil)
		c.DrawRect(rectAt(0, 0, 12, 8), paint(0xFFFF0000)) // clipped to the clip at the save
		c.Restore()
		c.Restore()
	})
	for _, tt := range []struct {
		x, y int
		want [4]uint8
	}{
		{3, 3, [4]uint8{0, 255, 0, 255}}, {1, 3, [4]uint8{}}, {6, 3, [4]uint8{}}, {3, 6, [4]uint8{}},
		{9, 1, [4]uint8{255, 0, 0, 255}}, {7, 1, [4]uint8{}},
	} {
		if got := px(img, tt.x, tt.y); got != tt.want {
			t.Errorf("pixel %d,%d = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestSaveLayerBlendAndFilter(t *testing.T) {
	img := renderOf(t, 12, 4, func(c *ui.Canvas) {
		c.DrawRect(rectAt(0, 0, 12, 4), paint(0xFF0000FF))
		// The layer is composited with a blend mode: srcIn keeps the blue only where the layer is.
		c.SaveLayer(nil, blended(0xFF000000, ui.BlendModeDstIn))
		c.DrawRect(rectAt(0, 0, 4, 4), paint(0xFFFFFFFF))
		c.Restore()
		c.Save()
		c.ClipRect(rectAt(8, 0, 4, 4))
		f := paint(0xFF000000)
		f.SetColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFF0000), ui.BlendModeSrcIn))
		c.SaveLayer(nil, f)
		c.DrawRect(rectAt(0, 0, 12, 4), paint(0xFFFFFFFF))
		c.Restore()
		c.Restore()
	})
	// DstIn of the layer over blue: the surface keeps blue where the layer is
	// opaque (0..4 within the layer's bounds) and clears it elsewhere.
	if got := px(img, 1, 1); got != [4]uint8{0, 0, 255, 255} {
		t.Errorf("inside the layer = %v", got)
	}
	if got := px(img, 5, 1); got != [4]uint8{} {
		t.Errorf("outside the shape in the layer = %v, want cleared", got)
	}
	if got := px(img, 9, 1); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("color filtered layer = %v", got)
	}
}

func TestUnbalancedSaveLayerIsClosed(t *testing.T) {
	img := renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.SaveLayer(nil, paint(0x80FFFFFF))
		c.DrawRect(rectAt(0, 0, 4, 4), paint(0xFFFF0000))
		// no Restore
	})
	if got := px(img, 1, 1); got[3] < 127 || got[3] > 129 {
		t.Errorf("unrestored layer alpha = %d, want 128", got[3])
	}
}

func TestAtlasColorsAndVerticesShaders(t *testing.T) {
	// srcIn with a half-transparent white color halves the sprite's alpha.
	tr := []ui.RSTransform{ui.NewRSTransform(1, 0, 0, 0)}
	rects := []ui.Rect{rectAt(0, 0, 2, 2)}
	img := renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.DrawAtlas(quads(), tr, rects, []ui.Color{ui.NewColor(0x80FFFFFF)}, ui.BlendModeSrcIn, nil, withQuality(ui.FilterQualityNone))
	})
	if got := px(img, 0, 0); got != [4]uint8{128, 0, 0, 128} {
		t.Errorf("tinted sprite = %v", got)
	}
	// Without colors, or with the default mode (src), the sprite is as it is.
	img = renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.DrawAtlas(quads(), tr, rects, []ui.Color{ui.NewColor(0x80FFFFFF)}, ui.BlendModeUnspecified, nil, withQuality(ui.FilterQualityNone))
	})
	if got := px(img, 0, 0); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("sprite with the default mode = %v", got)
	}

	// Vertices with a shader: a gradient across the triangle, or the image at the
	// texture coordinates.
	v := ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0, 0, 8))
	g := ui.NewPaint()
	g.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 8}, colorsOf(0xFF000000, 0xFFFFFFFF)))
	img = renderOf(t, 8, 8, func(c *ui.Canvas) { c.DrawVertices(v, ui.BlendModeSrcOver, g) })
	if a, b := px(img, 0, 0)[0], px(img, 5, 0)[0]; a >= b {
		t.Errorf("gradient on vertices: %d then %d", a, b)
	}
	tv := ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0, 0, 8), ui.VerticesOpts{TextureCoordinates: pts(0, 0, 4, 0, 0, 4)})
	sh := ui.NewPaint()
	sh.SetShader(ui.NewImageShader(quads(), ui.TileModeClamp, ui.TileModeClamp, nil, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityNone}))
	img = renderOf(t, 8, 8, func(c *ui.Canvas) { c.DrawVertices(tv, ui.BlendModeSrcOver, sh) })
	if got := px(img, 0, 0); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("textured vertices, top left = %v", got)
	}
	if got := px(img, 5, 0); got != [4]uint8{0, 255, 0, 255} {
		t.Errorf("textured vertices, top right = %v", got)
	}
	// Vertex colors modulated by the paint's color.
	cv := ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0, 0, 8), ui.VerticesOpts{Colors: colorsOf(0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF)})
	img = renderOf(t, 8, 8, func(c *ui.Canvas) { c.DrawVertices(cv, ui.BlendModeModulate, paint(0xFF336699)) })
	if got := px(img, 1, 1); got != [4]uint8{0x33, 0x66, 0x99, 255} {
		t.Errorf("modulated vertices = %v", got)
	}
}

func TestImageUsesPaintBlendAndFilter(t *testing.T) {
	filter := withQuality(ui.FilterQualityNone)
	filter.SetColorFilter(ui.NewColorFilterMatrix([20]float64{0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0}))
	img := renderOf(t, 4, 4, func(c *ui.Canvas) { c.DrawImage(quads(), ui.Offset{}, filter) })
	if got := px(img, 0, 0); got != [4]uint8{0, 0, 255, 255} {
		t.Errorf("red image with red and blue swapped = %v", got)
	}
	p := withQuality(ui.FilterQualityNone)
	p.SetBlendMode(ui.BlendModeDstOver)
	img = renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.DrawRect(rectAt(0, 0, 2, 4), paint(0xFFFFFFFF))
		c.DrawImage(quads(), ui.Offset{}, p)
	})
	if got := px(img, 0, 0); got != [4]uint8{255, 255, 255, 255} {
		t.Errorf("image drawn under = %v", got)
	}
	if got := px(img, 3, 0); got != [4]uint8{0, 255, 0, 255} {
		t.Errorf("image where nothing was = %v", got)
	}
}

// sceneWith builds a scene whose layers are pushed by push, over a picture that
// fills the cull rect with the color.
func sceneWith(argb uint32, push func(b *ui.SceneBuilder)) *ui.Scene {
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(rectAt(0, 0, 16, 16), paint(argb))
	b := ui.NewSceneBuilder()
	push(b)
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	return b.Build()
}

func TestSceneLayers(t *testing.T) {
	tests := []struct {
		name string
		push func(b *ui.SceneBuilder)
		at   map[image.Point][4]uint8
	}{
		{"opacity", func(b *ui.SceneBuilder) { b.PushOpacity(128) }, map[image.Point][4]uint8{{4, 4}: {128, 0, 0, 128}}},
		{"opacity 0 hides", func(b *ui.SceneBuilder) { b.PushOpacity(0) }, map[image.Point][4]uint8{{4, 4}: {}}},
		{"opacity 255 is the same", func(b *ui.SceneBuilder) { b.PushOpacity(255) }, map[image.Point][4]uint8{{4, 4}: {255, 0, 0, 255}}},
		{"opacity with an offset", func(b *ui.SceneBuilder) { b.PushOpacity(255, ui.PushOpacityOpts{Offset: ui.Offset{Dx: 4, Dy: 2}}) },
			map[image.Point][4]uint8{{2, 2}: {}, {5, 3}: {255, 0, 0, 255}, {4, 1}: {}}},
		{"nested opacity multiplies", func(b *ui.SceneBuilder) { b.PushOpacity(128); b.PushOpacity(128) },
			map[image.Point][4]uint8{{4, 4}: {64, 0, 0, 64}}},
		{"color filter", func(b *ui.SceneBuilder) {
			b.PushColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFF0000FF), ui.BlendModeSrcIn))
		}, map[image.Point][4]uint8{{4, 4}: {0, 0, 255, 255}}},
		{"offset", func(b *ui.SceneBuilder) { b.PushOffset(6, 6) }, map[image.Point][4]uint8{{3, 3}: {}, {7, 7}: {255, 0, 0, 255}}},
		{"clip rect", func(b *ui.SceneBuilder) { b.PushClipRect(rectAt(2, 2, 4, 4)) },
			map[image.Point][4]uint8{{3, 3}: {255, 0, 0, 255}, {7, 3}: {}, {1, 3}: {}}},
		{"clip rect hard", func(b *ui.SceneBuilder) {
			b.PushClipRect(rectAt(2.5, 2, 4, 4), ui.PushClipRectOpts{ClipBehavior: ui.ClipHardEdge})
		}, map[image.Point][4]uint8{{2, 3}: {255, 0, 0, 255}, {1, 3}: {}}},
		{"clip rect anti-aliased", func(b *ui.SceneBuilder) { b.PushClipRect(rectAt(2.5, 2, 4, 4)) },
			map[image.Point][4]uint8{{2, 3}: {128, 0, 0, 128}, {3, 3}: {255, 0, 0, 255}}},
		{"clip none", func(b *ui.SceneBuilder) {
			b.PushClipRect(rectAt(2, 2, 4, 4), ui.PushClipRectOpts{ClipBehavior: ui.ClipNone})
		}, map[image.Point][4]uint8{{10, 10}: {255, 0, 0, 255}}},
		{"clip with save layer", func(b *ui.SceneBuilder) {
			b.PushClipRect(rectAt(2.5, 2, 4, 4), ui.PushClipRectOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer})
		}, map[image.Point][4]uint8{{2, 3}: {128, 0, 0, 128}, {3, 3}: {255, 0, 0, 255}, {8, 3}: {}}},
		{"clip rrect", func(b *ui.SceneBuilder) { b.PushClipRRect(ui.NewRRectFromRectXY(rectAt(0, 0, 16, 16), 8, 8)) },
			map[image.Point][4]uint8{{0, 0}: {}, {8, 8}: {255, 0, 0, 255}}},
		{"clip superellipse", func(b *ui.SceneBuilder) {
			b.PushClipRSuperellipse(ui.NewRSuperellipseFromRectXY(rectAt(0, 0, 16, 16), 8, 8))
		}, map[image.Point][4]uint8{{0, 0}: {}, {8, 8}: {255, 0, 0, 255}}},
		{"clip path", func(b *ui.SceneBuilder) {
			p := ui.NewPath()
			p.AddOval(rectAt(0, 0, 16, 16))
			b.PushClipPath(p)
		}, map[image.Point][4]uint8{{0, 0}: {}, {8, 8}: {255, 0, 0, 255}}},
		{"transform", func(b *ui.SceneBuilder) {
			m := ui.NewMatrix4Identity()
			m.Scale(0.5, 0.5, 1)
			b.PushTransform(m)
		}, map[image.Point][4]uint8{{4, 4}: {255, 0, 0, 255}, {9, 4}: {}}},
		{"clip inside a transform follows it", func(b *ui.SceneBuilder) {
			b.PushOffset(8, 0)
			b.PushClipRect(rectAt(0, 0, 4, 16))
		}, map[image.Point][4]uint8{{9, 4}: {255, 0, 0, 255}, {5, 4}: {}, {13, 4}: {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := Render(sceneWith(0xFFFF0000, tt.push), 16, 16)
			if err != nil {
				t.Fatal(err)
			}
			for p, want := range tt.at {
				got := px(img, p.X, p.Y)
				for k := range got {
					if absInt(int(got[k])-int(want[k])) > 1 {
						t.Errorf("pixel %v = %v, want %v", p, got, want)
						break
					}
				}
			}
		})
	}
}

func TestSceneLayersGroupChildren(t *testing.T) {
	// An opacity layer over two overlapping children composites them as a group.
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawRect(rectAt(0, 0, 10, 8), paint(0xFF000000))
	c.DrawRect(rectAt(6, 0, 10, 8), paint(0xFF000000))
	b := ui.NewSceneBuilder()
	b.PushOpacity(128)
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := Render(b.Build(), 16, 8)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []int{2, 8, 13} {
		if got := px(img, x, 3)[3]; got < 127 || got > 129 {
			t.Errorf("alpha at %d = %d, want 128", x, got)
		}
	}
}

func TestGoldenLayers(t *testing.T) {
	t.Run("save-layer-opacity", func(t *testing.T) {
		img := renderOf(t, 64, 64, func(c *ui.Canvas) {
			white(c)
			c.DrawRect(rectAt(0, 24, 64, 16), paint(0xFF202020))
			c.SaveLayer(nil, paint(0x80FFFFFF))
			c.DrawCircle(ui.Offset{Dx: 24, Dy: 32}, 18, paint(0xFFE53935))
			c.DrawCircle(ui.Offset{Dx: 40, Dy: 32}, 18, paint(0xFF1E88E5))
			c.Restore()
		})
		goldens.Check(t, "layer-save-layer-opacity", img)
	})
	t.Run("scene-layers", func(t *testing.T) {
		rec := ui.NewPictureRecorder()
		c := ui.NewCanvas(rec)
		c.DrawCircle(ui.Offset{Dx: 20, Dy: 20}, 14, paint(0xFFE53935))
		c.DrawCircle(ui.Offset{Dx: 34, Dy: 20}, 14, paint(0xFF1E88E5))
		pic := rec.EndRecording()
		b := ui.NewSceneBuilder()
		rec2 := ui.NewPictureRecorder()
		ui.NewCanvas(rec2).DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
		b.AddPicture(ui.Offset{}, rec2.EndRecording())
		b.PushOpacity(150)
		b.AddPicture(ui.Offset{}, pic)
		b.Pop()
		b.PushOffset(0, 40)
		b.PushColorFilter(ui.NewColorFilterSaturation(0))
		b.AddPicture(ui.Offset{}, pic)
		b.Pop()
		b.Pop()
		b.PushOffset(54, 0)
		b.PushClipRRect(ui.NewRRectFromRectXY(rectAt(0, 0, 40, 40), 12, 12))
		b.AddPicture(ui.Offset{}, pic)
		scene := b.Build()
		img, err := Render(scene, 96, 80)
		if err != nil {
			t.Fatal(err)
		}
		goldens.Check(t, "layer-scene", img)
	})
	t.Run("clip-behaviors", func(t *testing.T) {
		b := ui.NewSceneBuilder()
		rec := ui.NewPictureRecorder()
		ui.NewCanvas(rec).DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
		b.AddPicture(ui.Offset{}, rec.EndRecording())
		stripes := ui.NewPictureRecorder()
		sc := ui.NewCanvas(stripes)
		for i := 0; i < 8; i++ {
			col := uint32(0xFF3060C0)
			if i%2 == 1 {
				col = 0xFFF0A020
			}
			sc.DrawRect(rectAt(float64(i)*8, 0, 8, 64), paint(col))
		}
		pic := stripes.EndRecording()
		for i, behavior := range []ui.Clip{ui.ClipNone, ui.ClipHardEdge, ui.ClipAntiAlias, ui.ClipAntiAliasWithSaveLayer} {
			b.PushOffset(float64(i)*70, 0)
			b.PushClipRRect(ui.NewRRectFromRectXY(rectAt(4.5, 4.5, 54, 54), 20, 20), ui.PushClipRRectOpts{ClipBehavior: behavior})
			b.AddPicture(ui.Offset{}, pic)
			b.Pop()
			b.Pop()
		}
		img, err := Render(b.Build(), 280, 64)
		if err != nil {
			t.Fatal(err)
		}
		goldens.Check(t, "layer-clip-behaviors", img)
	})
	t.Run("blend-mode-pairs", func(t *testing.T) {
		img := renderOf(t, 128, 64, func(c *ui.Canvas) {
			white(c)
			for i, mode := range []ui.BlendMode{ui.BlendModeMultiply, ui.BlendModeScreen, ui.BlendModeDifference, ui.BlendModeColor} {
				ox, oy := float64(i%2)*64, float64(i/2)*32
				c.Save()
				c.ClipRect(rectAt(ox, oy, 64, 32))
				c.SaveLayer(nil, nil)
				c.DrawCircle(ui.Offset{Dx: ox + 22, Dy: oy + 16}, 12, paint(0xFFE53935))
				c.DrawCircle(ui.Offset{Dx: ox + 40, Dy: oy + 16}, 12, blended(0xFF1E88E5, mode))
				c.Restore()
				c.Restore()
			}
		})
		goldens.Check(t, "layer-blend-mode-pairs", img)
	})
}

func TestLayerWithEmptyAreaDoesNothing(t *testing.T) {
	img := renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.ClipRect(rectAt(0, 0, 0, 0))
		c.SaveLayer(nil, paint(0x80FFFFFF))
		c.DrawRect(rectAt(0, 0, 4, 4), paint(0xFFFF0000))
		c.Restore()
	})
	if got := coverageSum(img); got != 0 {
		t.Errorf("drew %v", got)
	}
	if got := deviceBounds(ui.NewMatrix4Identity(), ui.NewRectFromLTWH(math.NaN(), 0, 1, 1)); !got.Empty() {
		t.Errorf("bounds of NaN = %v", got)
	}
}
