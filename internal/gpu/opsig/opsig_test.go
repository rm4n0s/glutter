package opsig

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func paintOf(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

func withPaint(set func(p *ui.Paint)) ui.Paint {
	p := paintOf(0xFF336699)
	set(p)
	return *p
}

func pathOf(points ...float64) *ui.Path {
	p := ui.NewPath()
	p.MoveTo(points[0], points[1])
	for i := 2; i < len(points); i += 2 {
		p.LineTo(points[i], points[i+1])
	}
	p.Close()
	return p
}

func imageOf(seed byte) *ui.Image {
	pix := make([]byte, 4*4*4)
	for i := range pix {
		pix[i] = seed + byte(i)
	}
	return ui.NewImageFromRGBA(pix, 4, 4)
}

func rrectOf(radius float64) ui.RRect {
	return ui.NewRRectFromRectXY(ui.NewRectFromLTWH(1, 2, 30, 20), radius, radius)
}

func gradientOf(to ui.Offset, colors []ui.Color, opts ...ui.GradientLinearOpts) *ui.Gradient {
	return ui.NewGradientLinear(ui.Offset{Dx: 1, Dy: 2}, to, colors, opts...)
}

func pictureOf(fn func(c *ui.Canvas)) *ui.Picture {
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	return rec.EndRecording()
}

// mustSig is the signature of an operation that has one.
func mustSig(t *testing.T, op ui.DisplayOp) uint64 {
	t.Helper()
	sig, ok := Op(op)
	if !ok {
		t.Fatalf("%T has no signature", op)
	}
	return sig
}

// The images, the Vertices and the picture of the operations below: the same object each
// time an operation is made, since an image and a Vertices are taken by their identity.
var (
	testImage    = imageOf(1)
	otherImage   = imageOf(2)
	testVertices = ui.NewVertices(ui.VertexModeTriangles, []ui.Offset{{Dx: 0, Dy: 0}, {Dx: 10, Dy: 0}, {Dx: 0, Dy: 10}})
	moreVertices = ui.NewVertices(ui.VertexModeTriangles, []ui.Offset{{Dx: 0, Dy: 0}, {Dx: 10, Dy: 0}, {Dx: 0, Dy: 10}})
)

// opVariants lists, for each kind of operation, a function that makes the operation from
// nothing (new paths, paints, gradients and filters each time) and the functions that make
// it with one thing changed.
func opVariants() []struct {
	name     string
	base     func() ui.DisplayOp
	variants []func() ui.DisplayOp
} {
	red, blue := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)
	rect := ui.NewRectFromLTWH(1, 2, 30, 20)
	plain := func() ui.Paint { return *paintOf(0xFF336699) }
	identity := ui.NewMatrix4Identity()
	shifted := ui.NewMatrix4TranslationValues(3, 0, 0)
	return []struct {
		name     string
		base     func() ui.DisplayOp
		variants []func() ui.DisplayOp
	}{
		{"save", func() ui.DisplayOp { return ui.OpSave{} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpRestore{} },
		}},
		{"save layer", func() ui.DisplayOp { return ui.OpSaveLayer{Bounds: rect, HasBounds: true, Paint: plain()} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpSaveLayer{Bounds: rect, Paint: plain()} },
			func() ui.DisplayOp {
				return ui.OpSaveLayer{Bounds: rect.Shift(ui.Offset{Dx: 1}), HasBounds: true, Paint: plain()}
			},
			func() ui.DisplayOp { return ui.OpSaveLayer{Bounds: rect, HasBounds: true, Paint: *paintOf(0x80336699)} },
		}},
		{"transform", func() ui.DisplayOp { return ui.OpTransform{Matrix: *identity} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpTransform{Matrix: *shifted} },
		}},
		{"clip rect", func() ui.DisplayOp { return ui.OpClipRect{Rect: rect, Op: ui.ClipOpIntersect, AntiAlias: true} }, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpClipRect{Rect: rect.Inflate(1), Op: ui.ClipOpIntersect, AntiAlias: true}
			},
			func() ui.DisplayOp { return ui.OpClipRect{Rect: rect, Op: ui.ClipOpDifference, AntiAlias: true} },
			func() ui.DisplayOp { return ui.OpClipRect{Rect: rect, Op: ui.ClipOpIntersect} },
		}},
		{"clip rrect", func() ui.DisplayOp { return ui.OpClipRRect{RRect: rrectOf(4), AntiAlias: true} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpClipRRect{RRect: rrectOf(5), AntiAlias: true} },
			func() ui.DisplayOp { return ui.OpClipRRect{RRect: rrectOf(4)} },
			func() ui.DisplayOp {
				return ui.OpClipRSuperellipse{RSuperellipse: ui.RSuperellipse(rrectOf(4)), AntiAlias: true}
			},
		}},
		{"clip path", func() ui.DisplayOp { return ui.OpClipPath{Path: pathOf(0, 0, 10, 0, 5, 9), AntiAlias: true} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpClipPath{Path: pathOf(0, 0, 10, 0, 5, 8), AntiAlias: true} },
			func() ui.DisplayOp { return ui.OpClipPath{Path: pathOf(0, 0, 10, 0, 5, 9)} },
			func() ui.DisplayOp { return ui.OpClipPath{AntiAlias: true} },
		}},
		{"color", func() ui.DisplayOp { return ui.OpDrawColor{Color: red, BlendMode: ui.BlendModeSrcOver} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpDrawColor{Color: blue, BlendMode: ui.BlendModeSrcOver} },
			func() ui.DisplayOp { return ui.OpDrawColor{Color: red, BlendMode: ui.BlendModeMultiply} },
		}},
		{"paint", func() ui.DisplayOp { return ui.OpDrawPaint{Paint: plain()} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpDrawPaint{Paint: *paintOf(0xFF336698)} },
		}},
		{"line", func() ui.DisplayOp {
			return ui.OpDrawLine{P1: ui.Offset{Dx: 1}, P2: ui.Offset{Dx: 9, Dy: 9}, Paint: plain()}
		}, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpDrawLine{P1: ui.Offset{Dx: 2}, P2: ui.Offset{Dx: 9, Dy: 9}, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawLine{P1: ui.Offset{Dx: 1}, P2: ui.Offset{Dx: 9, Dy: 8}, Paint: plain()}
			},
		}},
		{"rect", func() ui.DisplayOp { return ui.OpDrawRect{Rect: rect, Paint: plain()} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpDrawRect{Rect: rect.Shift(ui.Offset{Dy: 1}), Paint: plain()} },
			func() ui.DisplayOp { return ui.OpDrawOval{Rect: rect, Paint: plain()} },
			func() ui.DisplayOp { return ui.OpDrawRect{Rect: rect, Paint: *paintOf(0xFF000000)} },
		}},
		{"rrect", func() ui.DisplayOp { return ui.OpDrawRRect{RRect: rrectOf(4), Paint: plain()} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpDrawRRect{RRect: rrectOf(3), Paint: plain()} },
			func() ui.DisplayOp {
				return ui.OpDrawRSuperellipse{RSuperellipse: ui.RSuperellipse(rrectOf(4)), Paint: plain()}
			},
			func() ui.DisplayOp { return ui.OpDrawDRRect{Outer: rrectOf(4), Inner: rrectOf(2), Paint: plain()} },
			func() ui.DisplayOp { return ui.OpDrawDRRect{Outer: rrectOf(4), Inner: rrectOf(1), Paint: plain()} },
		}},
		{"circle", func() ui.DisplayOp {
			return ui.OpDrawCircle{Center: ui.Offset{Dx: 5, Dy: 5}, Radius: 4, Paint: plain()}
		}, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpDrawCircle{Center: ui.Offset{Dx: 5, Dy: 6}, Radius: 4, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawCircle{Center: ui.Offset{Dx: 5, Dy: 5}, Radius: 5, Paint: plain()}
			},
		}},
		{"arc", func() ui.DisplayOp { return ui.OpDrawArc{Rect: rect, StartAngle: 1, SweepAngle: 2, Paint: plain()} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpDrawArc{Rect: rect, StartAngle: 1.5, SweepAngle: 2, Paint: plain()} },
			func() ui.DisplayOp { return ui.OpDrawArc{Rect: rect, StartAngle: 1, SweepAngle: 2.5, Paint: plain()} },
			func() ui.DisplayOp {
				return ui.OpDrawArc{Rect: rect, StartAngle: 1, SweepAngle: 2, UseCenter: true, Paint: plain()}
			},
		}},
		{"path", func() ui.DisplayOp { return ui.OpDrawPath{Path: pathOf(0, 0, 10, 0, 5, 9), Paint: plain()} }, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpDrawPath{Path: pathOf(0, 0, 10, 0, 5, 9, 2, 2), Paint: plain()} },
			func() ui.DisplayOp {
				p := pathOf(0, 0, 10, 0, 5, 9)
				p.SetFillType(ui.PathFillTypeEvenOdd)
				return ui.OpDrawPath{Path: p, Paint: plain()}
			},
		}},
		{"image", func() ui.DisplayOp { return ui.OpDrawImage{Image: testImage, Offset: ui.Offset{Dx: 3}, Paint: plain()} }, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpDrawImage{Image: otherImage, Offset: ui.Offset{Dx: 3}, Paint: plain()}
			},
			func() ui.DisplayOp { return ui.OpDrawImage{Image: testImage, Offset: ui.Offset{Dx: 4}, Paint: plain()} },
			func() ui.DisplayOp { return ui.OpDrawImage{Offset: ui.Offset{Dx: 3}, Paint: plain()} },
			func() ui.DisplayOp { return ui.OpDrawImageRect{Image: testImage, Src: rect, Dst: rect, Paint: plain()} },
			func() ui.DisplayOp {
				return ui.OpDrawImageRect{Image: testImage, Src: rect.Inflate(1), Dst: rect, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawImageRect{Image: testImage, Src: rect, Dst: rect.Inflate(1), Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawImageNine{Image: testImage, Center: rect, Dst: rect, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawImageNine{Image: testImage, Center: rect.Inflate(1), Dst: rect, Paint: plain()}
			},
		}},
		{"points", func() ui.DisplayOp {
			return ui.OpDrawPoints{Mode: ui.PointModePoints, Points: []ui.Offset{{Dx: 1}, {Dx: 2}}, Paint: plain()}
		}, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpDrawPoints{Mode: ui.PointModeLines, Points: []ui.Offset{{Dx: 1}, {Dx: 2}}, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawPoints{Mode: ui.PointModePoints, Points: []ui.Offset{{Dx: 1}, {Dx: 3}}, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawPoints{Mode: ui.PointModePoints, Points: []ui.Offset{{Dx: 1}, {Dx: 2}, {Dx: 2}}, Paint: plain()}
			},
		}},
		{"vertices", func() ui.DisplayOp {
			return ui.OpDrawVertices{Vertices: testVertices, BlendMode: ui.BlendModeModulate, Paint: plain()}
		}, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpDrawVertices{Vertices: moreVertices, BlendMode: ui.BlendModeModulate, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawVertices{Vertices: testVertices, BlendMode: ui.BlendModeSrcOver, Paint: plain()}
			},
			func() ui.DisplayOp { return ui.OpDrawVertices{BlendMode: ui.BlendModeModulate, Paint: plain()} },
		}},
		{"atlas", func() ui.DisplayOp {
			return ui.OpDrawAtlas{Atlas: testImage, Transforms: []ui.RSTransform{{Scos: 1}}, Rects: []ui.Rect{rect}, Colors: []ui.Color{red},
				BlendMode: ui.BlendModeModulate, Paint: plain()}
		}, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpDrawAtlas{Atlas: testImage, Transforms: []ui.RSTransform{{Scos: 1, Tx: 2}}, Rects: []ui.Rect{rect}, Colors: []ui.Color{red},
					BlendMode: ui.BlendModeModulate, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawAtlas{Atlas: testImage, Transforms: []ui.RSTransform{{Scos: 1}}, Rects: []ui.Rect{rect.Inflate(1)}, Colors: []ui.Color{red},
					BlendMode: ui.BlendModeModulate, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawAtlas{Atlas: testImage, Transforms: []ui.RSTransform{{Scos: 1}}, Rects: []ui.Rect{rect}, Colors: []ui.Color{blue},
					BlendMode: ui.BlendModeModulate, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawAtlas{Atlas: testImage, Transforms: []ui.RSTransform{{Scos: 1}}, Rects: []ui.Rect{rect}, Colors: []ui.Color{red},
					BlendMode: ui.BlendModeModulate, CullRect: rect, HasCullRect: true, Paint: plain()}
			},
			func() ui.DisplayOp {
				return ui.OpDrawAtlas{Atlas: otherImage, Transforms: []ui.RSTransform{{Scos: 1}}, Rects: []ui.Rect{rect}, Colors: []ui.Color{red},
					BlendMode: ui.BlendModeModulate, Paint: plain()}
			},
		}},
		{"shadow", func() ui.DisplayOp {
			return ui.OpDrawShadow{Path: pathOf(0, 0, 10, 0, 5, 9), Color: red, Elevation: 4}
		}, []func() ui.DisplayOp{
			func() ui.DisplayOp { return ui.OpDrawShadow{Path: pathOf(0, 0, 10, 0, 5, 8), Color: red, Elevation: 4} },
			func() ui.DisplayOp {
				return ui.OpDrawShadow{Path: pathOf(0, 0, 10, 0, 5, 9), Color: blue, Elevation: 4}
			},
			func() ui.DisplayOp { return ui.OpDrawShadow{Path: pathOf(0, 0, 10, 0, 5, 9), Color: red, Elevation: 5} },
			func() ui.DisplayOp {
				return ui.OpDrawShadow{Path: pathOf(0, 0, 10, 0, 5, 9), Color: red, Elevation: 4, TransparentOccluder: true}
			},
		}},
		{"picture", func() ui.DisplayOp {
			return ui.OpDrawPicture{Picture: pictureOf(func(c *ui.Canvas) { c.DrawRect(rect, paintOf(0xFF102030)) })}
		}, []func() ui.DisplayOp{
			func() ui.DisplayOp {
				return ui.OpDrawPicture{Picture: pictureOf(func(c *ui.Canvas) { c.DrawRect(rect, paintOf(0xFF102031)) })}
			},
			func() ui.DisplayOp {
				return ui.OpDrawPicture{Picture: pictureOf(func(c *ui.Canvas) {
					c.DrawRect(rect, paintOf(0xFF102030))
					c.DrawRect(rect, paintOf(0xFF102030))
				})}
			},
			func() ui.DisplayOp { return ui.OpDrawPicture{} },
		}},
	}
}

// TestSignatureIsOfTheContents makes each operation twice, from new objects, and checks
// that the two have one signature, and that every variant of it has another.
func TestSignatureIsOfTheContents(t *testing.T) {
	seen := map[uint64]string{}
	for _, tt := range opVariants() {
		t.Run(tt.name, func(t *testing.T) {
			base := mustSig(t, tt.base())
			if again := mustSig(t, tt.base()); again != base {
				t.Errorf("the operation made again has the signature %x, want %x", again, base)
			}
			if other, ok := seen[base]; ok {
				t.Errorf("the signature is that of %s", other)
			}
			seen[base] = tt.name
			for i, variant := range tt.variants {
				name := tt.name + " variant " + string(rune('a'+i))
				sig := mustSig(t, variant())
				if again := mustSig(t, variant()); again != sig {
					t.Errorf("%s made again has the signature %x, want %x", name, again, sig)
				}
				if other, ok := seen[sig]; ok {
					t.Errorf("%s has the signature of %s", name, other)
				}
				seen[sig] = name
			}
		})
	}
}

// TestSignatureHoldsEveryPartOfAPaint changes one thing of a paint at a time: each gives
// another signature, and the same paint made again gives the same.
func TestSignatureHoldsEveryPartOfAPaint(t *testing.T) {
	red, blue, green := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF), ui.NewColor(0xFF00FF00)
	matrix := ui.NewMatrix4TranslationValues(2, 3, 0)
	saturation := [20]float64{0.5, 0, 0, 0, 0, 0, 0.5, 0, 0, 0, 0, 0, 0.5, 0, 0, 0, 0, 0, 1, 0}
	paints := []struct {
		name string
		set  func(p *ui.Paint)
	}{
		{"the base", func(*ui.Paint) {}},
		{"no anti-aliasing", func(p *ui.Paint) { p.SetIsAntiAlias(false) }},
		{"a color", func(p *ui.Paint) { p.SetColor(ui.NewColor(0xFF336698)) }},
		{"an alpha", func(p *ui.Paint) { p.SetColor(ui.NewColor(0xFE336699)) }},
		{"a blend mode", func(p *ui.Paint) { p.SetBlendMode(ui.BlendModeMultiply) }},
		{"stroked", func(p *ui.Paint) { p.SetStyle(ui.PaintingStyleStroke) }},
		{"a stroke width", func(p *ui.Paint) { p.SetStrokeWidth(3) }},
		{"a cap", func(p *ui.Paint) { p.SetStrokeCap(ui.StrokeCapRound) }},
		{"a join", func(p *ui.Paint) { p.SetStrokeJoin(ui.StrokeJoinBevel) }},
		{"a miter limit", func(p *ui.Paint) { p.SetStrokeMiterLimit(7) }},
		{"a filter quality", func(p *ui.Paint) { p.SetFilterQuality(ui.FilterQualityHigh) }},
		{"inverted colors", func(p *ui.Paint) { p.SetInvertColors(true) }},
		{"a blur", func(p *ui.Paint) { p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 3)) }},
		{"a wider blur", func(p *ui.Paint) { p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 4)) }},
		{"a blur of another style", func(p *ui.Paint) { p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleOuter, 3)) }},
		{"a color filter", func(p *ui.Paint) { p.SetColorFilter(ui.NewColorFilterMode(red, ui.BlendModeSrcIn)) }},
		{"a color filter of another color", func(p *ui.Paint) { p.SetColorFilter(ui.NewColorFilterMode(blue, ui.BlendModeSrcIn)) }},
		{"a color filter of another mode", func(p *ui.Paint) { p.SetColorFilter(ui.NewColorFilterMode(red, ui.BlendModeSrcOut)) }},
		{"a matrix color filter", func(p *ui.Paint) { p.SetColorFilter(ui.NewColorFilterMatrix(saturation)) }},
		{"a gamma color filter", func(p *ui.Paint) { p.SetColorFilter(ui.NewColorFilterLinearToSrgbGamma()) }},
		{"a gradient", func(p *ui.Paint) { p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 9}, []ui.Color{red, blue})) }},
		{"a gradient to another point", func(p *ui.Paint) { p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 8}, []ui.Color{red, blue})) }},
		{"a gradient of other colors", func(p *ui.Paint) { p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 9}, []ui.Color{red, green})) }},
		{"a gradient of more colors", func(p *ui.Paint) {
			p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 9}, []ui.Color{red, blue, green}, ui.GradientLinearOpts{ColorStops: []float64{0, 0.5, 1}}))
		}},
		{"a gradient with stops", func(p *ui.Paint) {
			p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 9}, []ui.Color{red, blue}, ui.GradientLinearOpts{ColorStops: []float64{0.2, 0.8}}))
		}},
		{"a gradient with other stops", func(p *ui.Paint) {
			p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 9}, []ui.Color{red, blue}, ui.GradientLinearOpts{ColorStops: []float64{0.3, 0.8}}))
		}},
		{"a gradient that repeats", func(p *ui.Paint) {
			p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 9}, []ui.Color{red, blue}, ui.GradientLinearOpts{TileMode: ui.TileModeRepeated}))
		}},
		{"a gradient with a matrix", func(p *ui.Paint) {
			p.SetShader(gradientOf(ui.Offset{Dx: 9, Dy: 9}, []ui.Color{red, blue}, ui.GradientLinearOpts{Matrix4: matrix}))
		}},
		{"a radial gradient", func(p *ui.Paint) {
			p.SetShader(ui.NewGradientRadial(ui.Offset{Dx: 1, Dy: 2}, 9, []ui.Color{red, blue}))
		}},
		{"a radial gradient of another radius", func(p *ui.Paint) {
			p.SetShader(ui.NewGradientRadial(ui.Offset{Dx: 1, Dy: 2}, 8, []ui.Color{red, blue}))
		}},
		{"a sweep gradient", func(p *ui.Paint) {
			p.SetShader(ui.NewGradientSweep(ui.Offset{Dx: 1, Dy: 2}, []ui.Color{red, blue}))
		}},
		{"an image shader", func(p *ui.Paint) {
			p.SetShader(ui.NewImageShader(testImage, ui.TileModeClamp, ui.TileModeClamp, matrix))
		}},
		{"an image shader of another image", func(p *ui.Paint) {
			p.SetShader(ui.NewImageShader(otherImage, ui.TileModeClamp, ui.TileModeClamp, matrix))
		}},
		{"an image shader that repeats", func(p *ui.Paint) {
			p.SetShader(ui.NewImageShader(testImage, ui.TileModeRepeated, ui.TileModeClamp, matrix))
		}},
		{"an image shader that repeats down", func(p *ui.Paint) {
			p.SetShader(ui.NewImageShader(testImage, ui.TileModeClamp, ui.TileModeRepeated, matrix))
		}},
		{"an image shader with another matrix", func(p *ui.Paint) {
			p.SetShader(ui.NewImageShader(testImage, ui.TileModeClamp, ui.TileModeClamp, ui.NewMatrix4Identity()))
		}},
		{"an image shader of another quality", func(p *ui.Paint) {
			p.SetShader(ui.NewImageShader(testImage, ui.TileModeClamp, ui.TileModeClamp, matrix, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityHigh}))
		}},
		{"a blur filter", func(p *ui.Paint) {
			p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 3}))
		}},
		{"a blur filter of another sigma", func(p *ui.Paint) {
			p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 4}))
		}},
		{"a blur filter that mirrors", func(p *ui.Paint) {
			p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 3, TileMode: ui.TileModeMirror}))
		}},
		{"a dilate filter", func(p *ui.Paint) {
			p.SetImageFilter(ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 3}))
		}},
		{"an erode filter", func(p *ui.Paint) {
			p.SetImageFilter(ui.NewImageFilterErode(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 3}))
		}},
		{"a matrix filter", func(p *ui.Paint) { p.SetImageFilter(ui.NewImageFilterMatrix(matrix)) }},
		{"a matrix filter of another matrix", func(p *ui.Paint) { p.SetImageFilter(ui.NewImageFilterMatrix(ui.NewMatrix4Identity())) }},
		{"a composed filter", func(p *ui.Paint) {
			p.SetImageFilter(ui.NewImageFilterCompose(ui.NewImageFilterMatrix(matrix), ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 3})))
		}},
		{"a composed filter the other way", func(p *ui.Paint) {
			p.SetImageFilter(ui.NewImageFilterCompose(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 3}), ui.NewImageFilterMatrix(matrix)))
		}},
		{"a color filter as an image filter", func(p *ui.Paint) { p.SetImageFilter(ui.NewColorFilterMode(red, ui.BlendModeSrcIn)) }},
	}
	rect := ui.NewRectFromLTWH(1, 2, 30, 20)
	seen := map[uint64]string{}
	for _, tt := range paints {
		t.Run(tt.name, func(t *testing.T) {
			sig := mustSig(t, ui.OpDrawRect{Rect: rect, Paint: withPaint(tt.set)})
			if again := mustSig(t, ui.OpDrawRect{Rect: rect, Paint: withPaint(tt.set)}); again != sig {
				t.Errorf("the paint made again has the signature %x, want %x", again, sig)
			}
			if other, ok := seen[sig]; ok {
				t.Errorf("the signature is that of %s", other)
			}
			seen[sig] = tt.name
		})
	}
}

// TestADefaultLeftUnsetIsTheDefault checks that a paint is taken by what it draws with,
// not by which of its fields were set.
func TestADefaultLeftUnsetIsTheDefault(t *testing.T) {
	rect := ui.NewRectFromLTWH(1, 2, 30, 20)
	unset := ui.NewPaint()
	set := ui.NewPaint()
	set.SetColor(ui.NewColor(0xFF000000))
	set.SetBlendMode(ui.BlendModeSrcOver)
	set.SetStyle(ui.PaintingStyleFill)
	set.SetStrokeMiterLimit(4)
	set.SetIsAntiAlias(true)
	if a, b := mustSig(t, ui.OpDrawRect{Rect: rect, Paint: *unset}), mustSig(t, ui.OpDrawRect{Rect: rect, Paint: *set}); a != b {
		t.Errorf("signatures = %x and %x, want the same", a, b)
	}
}

// TestAnImageIsItsPixels checks that every handle on the pixels of an image has the
// signature of the image (the offscreen detacher draws clones), and an image of other
// pixels another.
func TestAnImageIsItsPixels(t *testing.T) {
	img := imageOf(7)
	same := ui.NewImageFromRGBA(append([]byte(nil), img.Pixels()...), 4, 4)
	op := func(i *ui.Image) ui.DisplayOp { return ui.OpDrawImage{Image: i} }
	base := mustSig(t, op(img))
	tests := []struct {
		name  string
		image *ui.Image
		same  bool
	}{
		{"the image", img, true},
		{"a clone", img.Clone(), true},
		{"a clone of a clone", img.Clone().Clone(), true},
		{"a copy of the pixels", same, false},
		{"no image", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustSig(t, op(tt.image)); (got == base) != tt.same {
				t.Errorf("signature = %x, the image's is %x; same = %v, want %v", got, base, got == base, tt.same)
			}
		})
	}
}

// fakeLayout is a paragraph layout that emits the draws it is given.
type fakeLayout struct {
	ui.ParagraphLayout
	draws []ui.ParagraphDraw
}

func (f *fakeLayout) PaintParagraph(emit func(ui.ParagraphDraw)) {
	for _, d := range f.draws {
		emit(d)
	}
}

// signedLayout is a fake layout that knows its signature.
type signedLayout struct {
	fakeLayout
	sig uint64
	ok  bool
}

func (s *signedLayout) PaintSignature() (uint64, bool) { return s.sig, s.ok }

// mute is a layout that cannot draw itself.
type mute struct{ ui.ParagraphLayout }

func TestParagraphSignature(t *testing.T) {
	glyph := func(dx float64) ui.ParagraphDraw {
		return ui.ParagraphDraw{Op: ui.OpDrawPath{Path: pathOf(0, 0, 4, 0, 2, 5), Paint: *paintOf(0xFF000000)}, Offset: ui.Offset{Dx: dx, Dy: 10}, Snap: true}
	}
	text := func(draws ...ui.ParagraphDraw) *ui.Paragraph { return ui.NewParagraph(&fakeLayout{draws: draws}) }
	scaled := glyph(8)
	scaled.Matrix = ui.NewMatrix4Diagonal3Values(2, 2, 1)
	unsnapped := glyph(8)
	unsnapped.Snap = false
	base, ok := Paragraph(text(glyph(0), glyph(8)))
	if !ok {
		t.Fatal("the paragraph has no signature")
	}
	tests := []struct {
		name   string
		para   *ui.Paragraph
		same   bool
		wantOK bool
	}{
		{"the same draws from new objects", text(glyph(0), glyph(8)), true, true},
		{"a glyph moved", text(glyph(0), glyph(9)), false, true},
		{"a glyph more", text(glyph(0), glyph(8), glyph(16)), false, true},
		{"a glyph less", text(glyph(0)), false, true},
		{"the glyphs in the other order", text(glyph(8), glyph(0)), false, true},
		{"a glyph under a matrix", text(glyph(0), scaled), false, true},
		{"a glyph that does not snap", text(glyph(0), unsnapped), false, true},
		{"no draws", text(), false, true},
		{"no paragraph", nil, false, true},
		{"a layout that knows its signature", ui.NewParagraph(&signedLayout{sig: base, ok: true}), true, true},
		{"a layout that knows another", ui.NewParagraph(&signedLayout{sig: base + 1, ok: true}), false, true},
		{"a layout that says it has none", ui.NewParagraph(&signedLayout{sig: base}), true, false},
		{"a layout that cannot draw", ui.NewParagraph(&mute{}), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Paragraph(tt.para)
			if ok != tt.wantOK || (got == base) != tt.same {
				t.Errorf("Paragraph = %x, %v; the base is %x; want same = %v, ok = %v", got, ok, base, tt.same, tt.wantOK)
			}
			// The operation that draws it holds the offset too.
			at, opOK := Op(ui.OpDrawParagraph{Paragraph: tt.para, Offset: ui.Offset{Dx: 1}})
			moved, _ := Op(ui.OpDrawParagraph{Paragraph: tt.para, Offset: ui.Offset{Dx: 2}})
			if opOK != tt.wantOK || at == moved {
				t.Errorf("the operation: ok = %v, want %v; signatures at two offsets %x and %x", opOK, tt.wantOK, at, moved)
			}
		})
	}
}

func TestPictureSignature(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	record := func(argb uint32, extra bool) *ui.Picture {
		return pictureOf(func(c *ui.Canvas) {
			c.Save()
			c.Translate(3, 4)
			c.ClipRect(rect)
			c.DrawRect(rect, paintOf(argb))
			if extra {
				c.DrawPicture(pictureOf(func(c *ui.Canvas) { c.DrawCircle(ui.Offset{}, 3, paintOf(argb)) }))
			}
			c.Restore()
		})
	}
	base, ok := Picture(record(0xFF112233, true))
	if !ok {
		t.Fatal("the picture has no signature")
	}
	tests := []struct {
		name    string
		picture *ui.Picture
		same    bool
		wantOK  bool
	}{
		{"recorded again", record(0xFF112233, true), true, true},
		{"another color", record(0xFF112234, true), false, true},
		{"without the picture in it", record(0xFF112233, false), false, true},
		{"nothing recorded", pictureOf(func(*ui.Canvas) {}), false, true},
		{"no picture", nil, false, true},
		{"a paragraph that cannot draw", pictureOf(func(c *ui.Canvas) { c.DrawParagraph(ui.NewParagraph(&mute{}), ui.Offset{}) }), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Picture(tt.picture)
			if ok != tt.wantOK || (got == base) != tt.same {
				t.Errorf("Picture = %x, %v; the base is %x; want same = %v, ok = %v", got, ok, base, tt.same, tt.wantOK)
			}
		})
	}
}

func TestDrawsOfNothing(t *testing.T) {
	var d Draws
	if _, ok := d.Sum(); !ok {
		t.Error("no draws should have a signature")
	}
	d.Add(ui.ParagraphDraw{Op: ui.OpDrawParagraph{Paragraph: ui.NewParagraph(&mute{})}})
	if _, ok := d.Sum(); ok {
		t.Error("a draw without a signature should leave the draws without one")
	}
}
