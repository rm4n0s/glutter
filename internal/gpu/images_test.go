package gpu

import (
	"image"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestMipLevels(t *testing.T) {
	for _, tt := range []struct{ w, h, want int }{{1, 1, 1}, {2, 1, 2}, {3, 3, 2}, {4, 4, 3}, {64, 64, 7}, {100, 7, 7}, {8192, 1, 14}} {
		if got := MipLevels(tt.w, tt.h); got != tt.want {
			t.Errorf("MipLevels(%d, %d) = %d, want %d", tt.w, tt.h, got, tt.want)
		}
	}
}

func TestNinePieces(t *testing.T) {
	tests := []struct {
		name         string
		origin, size float64
		want         [4]float64
	}{
		{"the middle takes the rest", 10, 100, [4]float64{10, 18, 102, 110}},
		{"squeezed: the corners shrink and the middle vanishes", 0, 8, [4]float64{0, 4, 4, 8}},
		{"exactly the corners", 0, 16, [4]float64{0, 8, 8, 16}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NinePieces(tt.origin, tt.size, [4]float64{0, 8, 16, 24}); got != tt.want {
				t.Errorf("NinePieces = %v, want %v", got, tt.want)
			}
		})
	}
}

// imageEncode runs fn over an encoder of a surface of 100 x 100 pixels and returns the
// instances it made.
func imageEncode(t ui.FilterQuality, levels int, fn func(e *ImageEncoder)) []ImageInstance {
	var out []ImageInstance
	e := &ImageEncoder{
		T: Affine2D{A: 1, D: 1}, Outer: image.Rect(0, 0, 100, 100), Clip: [4]float32{0, 0, 100, 100},
		Paint: ShapePaint{Color: [4]float32{1, 1, 1, 1}, Antialias: true}, Quality: t, Levels: levels,
		Emit: func(in *ImageInstance, _ image.Rectangle) { out = append(out, *in) },
	}
	fn(e)
	return out
}

func TestImageEncoderRect(t *testing.T) {
	src, dst := ui.NewRectFromLTWH(0, 0, 16, 16), ui.NewRectFromLTWH(10, 20, 32, 32)
	got := imageEncode(ui.FilterQualityLow, 5, func(e *ImageEncoder) { e.Rect(16, 16, src, dst) })
	if len(got) != 1 {
		t.Fatalf("%d instances", len(got))
	}
	in := got[0]
	// The quad has a pixel more on each side, and the inverse maps pixels to texels at half size.
	if in[0] != 9 || in[1] != 19 || in[2] != 43 || in[3] != 53 {
		t.Errorf("quad %v", in[0:4])
	}
	if in[4] != 0.5 || in[7] != 0.5 || in[8] != -5 || in[9] != -10 {
		t.Errorf("inverse %v", in[4:10])
	}
	// An axis-aligned draw is an exact box in pixels, with the quality and level in the instance.
	if int(in[10])&ImageFlagBox == 0 || int(in[10])&ImageFlagAntialias == 0 {
		t.Errorf("flags %v", in[10])
	}
	if in[20] != 10 || in[21] != 20 || in[22] != 42 || in[23] != 52 {
		t.Errorf("box %v", in[20:24])
	}
	if in[24] != 0 || in[25] != 1 || in[26] != -1 {
		t.Errorf("level %v, quality %v, blend %v", in[24], in[25], in[26])
	}
}

func TestImageEncoderLevelsAndQuality(t *testing.T) {
	src := ui.NewRectFromLTWH(0, 0, 64, 64)
	tests := []struct {
		name    string
		quality ui.FilterQuality
		dst     float64
		level   float32
		q       float32
	}{
		{"nearest never uses a level", ui.FilterQualityNone, 8, 0, 0},
		{"low never uses a level", ui.FilterQualityLow, 8, 0, 1},
		{"medium picks the level of the scale", ui.FilterQualityMedium, 16, 2, 2},
		{"medium at an eighth", ui.FilterQualityMedium, 8, 3, 2},
		{"high picks one too", ui.FilterQualityHigh, 16, 2, 3},
		{"medium magnified uses the image", ui.FilterQualityMedium, 128, 0, 2},
		{"the level stops at the last", ui.FilterQualityMedium, 1, 3, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := imageEncode(tt.quality, 4, func(e *ImageEncoder) { e.Rect(64, 64, src, ui.NewRectFromLTWH(0, 0, tt.dst, tt.dst)) })
			if len(got) != 1 || got[0][24] != tt.level || got[0][25] != tt.q {
				t.Errorf("instances %v, want level %v quality %v", got, tt.level, tt.q)
			}
		})
	}
}

func TestImageEncoderCutsAndSkips(t *testing.T) {
	r := func(l, t, w, h float64) ui.Rect { return ui.NewRectFromLTWH(l, t, w, h) }
	tests := []struct {
		name string
		draw func(e *ImageEncoder)
		n    int
		box  []float32 // the box of the first instance, if any
	}{
		{"an empty source", func(e *ImageEncoder) { e.Rect(16, 16, r(0, 0, 0, 5), r(0, 0, 10, 10)) }, 0, nil},
		{"an empty destination", func(e *ImageEncoder) { e.Rect(16, 16, r(0, 0, 5, 5), r(0, 0, 0, 10)) }, 0, nil},
		{"a source outside the image", func(e *ImageEncoder) { e.Rect(16, 16, r(40, 40, 5, 5), r(0, 0, 10, 10)) }, 0, nil},
		{"a source half outside is cut with its destination", func(e *ImageEncoder) { e.Rect(16, 16, r(8, 0, 16, 16), r(0, 0, 32, 32)) }, 1, []float32{0, 0, 16, 32}},
		{"a destination off the surface", func(e *ImageEncoder) { e.Rect(16, 16, r(0, 0, 16, 16), r(500, 500, 10, 10)) }, 0, nil},
		{"a nine-patch is nine pieces", func(e *ImageEncoder) { e.Nine(24, 24, r(8, 8, 8, 8), r(0, 0, 60, 60)) }, 9, nil},
		{"a squeezed nine-patch has no middle", func(e *ImageEncoder) { e.Nine(24, 24, r(8, 8, 8, 8), r(10, 10, 8, 8)) }, 4, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := imageEncode(ui.FilterQualityLow, 3, tt.draw)
			if len(got) != tt.n {
				t.Fatalf("%d instances, want %d", len(got), tt.n)
			}
			if tt.box != nil && !reflect.DeepEqual(got[0][20:24], tt.box) {
				t.Errorf("box %v, want %v", got[0][20:24], tt.box)
			}
		})
	}
}

func TestImageEncoderAtlas(t *testing.T) {
	xf := []ui.RSTransform{ui.NewRSTransformFromComponents(0, 1, 0, 0, 5, 5), ui.NewRSTransformFromComponents(0.5, 0.5, 0, 0, 50, 5), ui.NewRSTransformFromComponents(0, 1, 0, 0, 500, 5)}
	rects := []ui.Rect{ui.NewRectFromLTWH(0, 0, 8, 8), ui.NewRectFromLTWH(8, 8, 8, 8), ui.NewRectFromLTWH(0, 0, 8, 8)}
	op := ui.OpDrawAtlas{Transforms: xf, Rects: rects}
	got := imageEncode(ui.FilterQualityLow, 3, func(e *ImageEncoder) { e.Atlas(16, 16, op, 0) })
	if len(got) != 2 { // the third sprite is off the surface
		t.Fatalf("%d instances, want 2", len(got))
	}
	if got[0][26] != -1 {
		t.Errorf("a sprite without colors has blend %v", got[0][26])
	}
	op.Colors = []ui.Color{ui.NewColor(0x80FF0000), ui.NewColor(0x8000FF00), ui.NewColor(0xFF0000FF)}
	got = imageEncode(ui.FilterQualityLow, 3, func(e *ImageEncoder) { e.Atlas(16, 16, op, 5) })
	if got[0][26] != 5 || got[0][28] <= 0 || got[0][31] <= 0 {
		t.Errorf("a tinted sprite: blend %v, tint %v", got[0][26], got[0][28:32])
	}
}

func TestVertexTriangles(t *testing.T) {
	pos := []ui.Offset{{Dx: 0, Dy: 0}, {Dx: 10, Dy: 0}, {Dx: 10, Dy: 10}, {Dx: 0, Dy: 10}}
	colors := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF), ui.NewColor(0xFFFFFFFF)}
	m := ui.NewMatrix4Identity()
	m.Translate(5, 6, 0)
	tests := []struct {
		name     string
		mode     ui.VertexMode
		indices  []int
		vertices int
		first    []float32 // the first three positions, flattened
	}{
		{"triangles ignore a remainder", ui.VertexModeTriangles, nil, 3, []float32{5, 6, 15, 6, 15, 16}},
		{"a strip", ui.VertexModeTriangleStrip, nil, 6, []float32{5, 6, 15, 6, 15, 16}},
		{"a fan", ui.VertexModeTriangleFan, nil, 6, []float32{5, 6, 15, 6, 15, 16}},
		{"indices pick the vertices", ui.VertexModeTriangles, []int{3, 2, 1}, 3, []float32{5, 16, 15, 16, 15, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ui.NewVertices(tt.mode, pos, ui.VerticesOpts{Colors: colors, Indices: tt.indices})
			floats, bounds, ok := VertexTriangles(v, m)
			if !ok || len(floats) != tt.vertices*VertexFloats {
				t.Fatalf("ok %v, %d floats, want %d vertices", ok, len(floats), tt.vertices)
			}
			got := []float32{floats[0], floats[1], floats[VertexFloats], floats[VertexFloats+1], floats[2*VertexFloats], floats[2*VertexFloats+1]}
			if !reflect.DeepEqual(got, tt.first) {
				t.Errorf("first positions %v, want %v", got, tt.first)
			}
			if !bounds.In(image.Rect(5, 6, 15, 16)) || bounds.Empty() {
				t.Errorf("bounds %v", bounds)
			}
		})
	}
	// The colors and the texture coordinates follow the position.
	v := ui.NewVertices(ui.VertexModeTriangles, pos[:3], ui.VerticesOpts{Colors: colors[:3], TextureCoordinates: []ui.Offset{{Dx: 0.5, Dy: 0.25}, {}, {}}})
	floats, _, _ := VertexTriangles(v, ui.NewMatrix4Identity())
	if floats[2] != 1 || floats[3] != 0 || floats[5] != 1 || floats[6] != 0.5 || floats[7] != 0.25 {
		t.Errorf("first vertex %v", floats[:VertexFloats])
	}
	// Nothing to draw.
	if _, _, ok := VertexTriangles(ui.NewVertices(ui.VertexModeTriangles, pos[:2]), m); ok {
		t.Error("two vertices made triangles")
	}
}
