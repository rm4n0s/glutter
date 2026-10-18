package gles

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

func solidPaint(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

func linePaint(argb uint32, width float64, cap ui.StrokeCap, join ui.StrokeJoin) *ui.Paint {
	p := solidPaint(argb)
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	p.SetStrokeCap(cap)
	p.SetStrokeJoin(join)
	return p
}

// build flattens a drawing and builds the instances for a surface of 200x100
// pixels.
func build(t *testing.T, skip bool, fn func(c *ui.Canvas)) (*frameBuilder, error) {
	t.Helper()
	return buildWith(t, nil, skip, fn)
}

// buildWith is build with the caches of paths, clip masks and images.
func buildWith(t *testing.T, res *resources, skip bool, fn func(c *ui.Canvas)) (*frameBuilder, error) {
	t.Helper()
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	list, err := gpu.Flatten(sb.Build(), gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 200, 100)})
	if err != nil {
		t.Fatal(err)
	}
	b := &frameBuilder{w: 200, h: 100, skip: skip, res: res}
	for _, c := range list.Commands {
		if err := b.command(c); err != nil {
			return b, err
		}
	}
	return b, nil
}

func mustBuild(t *testing.T, fn func(c *ui.Canvas)) *frameBuilder {
	t.Helper()
	b, err := build(t, false, fn)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func instanceAt(b *frameBuilder, i int) []float32 {
	return b.frame.data[pipeShape][i*shapeFloats : (i+1)*shapeFloats]
}

func TestBuildFillRect(t *testing.T) {
	b := mustBuild(t, func(c *ui.Canvas) {
		c.Translate(10, 20)
		c.DrawRect(ui.NewRectFromLTWH(5, 5, 40, 30), solidPaint(0x80FF8000))
	})
	if b.frame.items(pipeShape) != 1 {
		t.Fatalf("%d instances", b.frame.items(pipeShape))
	}
	in := instanceAt(b, 0)
	// The rectangle is at 15,25 to 55,55; the quad has a pixel more on each side.
	if in[0] != 14 || in[1] != 24 || in[2] != 56 || in[3] != 56 {
		t.Errorf("quad = %v", in[0:4])
	}
	if in[10] != kindBox {
		t.Errorf("kind = %v, want the exact box of an axis-aligned rectangle", in[10])
	}
	if in[20] != 15 || in[21] != 25 || in[22] != 55 || in[23] != 55 {
		t.Errorf("box = %v", in[20:24])
	}
	// 50% alpha, premultiplied.
	want := [4]float32{0.5, 0.5 * 0.50196, 0, 0.50196}
	for i, w := range want {
		if d := in[12+i] - w; d > 0.01 || d < -0.01 {
			t.Errorf("color = %v, want about %v", in[12:16], want)
			break
		}
	}
	if in[16] != 0 || in[17] != 0 || in[18] != 200 || in[19] != 100 {
		t.Errorf("clip = %v", in[16:20])
	}
	if int(in[11])&flagAntialias == 0 {
		t.Errorf("flags = %v, want anti-aliased", in[11])
	}
}

func TestBuildShapeKinds(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 60, 40)
	tests := []struct {
		name string
		draw func(c *ui.Canvas)
		kind float32
		hasB bool
	}{
		{"rrect", func(c *ui.Canvas) { c.DrawRRect(ui.NewRRectFromRectXY(rect, 8, 8), solidPaint(0xFF000000)) }, kindRRect, false},
		{"oval", func(c *ui.Canvas) { c.DrawOval(rect, solidPaint(0xFF000000)) }, kindRRect, false},
		{"circle", func(c *ui.Canvas) { c.DrawCircle(ui.Offset{Dx: 30, Dy: 30}, 10, solidPaint(0xFF000000)) }, kindRRect, false},
		{"rotated rect", func(c *ui.Canvas) { c.Rotate(0.3); c.DrawRect(rect, solidPaint(0xFF000000)) }, kindRRect, false},
		{"drrect", func(c *ui.Canvas) {
			c.DrawDRRect(ui.NewRRectFromRectXY(rect, 8, 8), ui.NewRRectFromRectXY(rect.Deflate(6), 4, 4), solidPaint(0xFF000000))
		}, kindRRect, true},
		{"drrect without a hole", func(c *ui.Canvas) {
			c.DrawDRRect(ui.NewRRectFromRectXY(rect, 8, 8), ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 20, 0, 0), 0, 0), solidPaint(0xFF000000))
		}, kindRRect, false},
		{"superellipse", func(c *ui.Canvas) {
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(rect, 8, 8), solidPaint(0xFF000000))
		}, kindSuper, false},
		{"line", func(c *ui.Canvas) {
			c.DrawLine(ui.Offset{Dx: 5, Dy: 5}, ui.Offset{Dx: 50, Dy: 40}, linePaint(0xFF000000, 4, ui.StrokeCapRound, ui.StrokeJoinMiter))
		}, kindSegment, false},
		{"stroked rect", func(c *ui.Canvas) { c.DrawRect(rect, linePaint(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter)) }, kindBox, true},
		{"stroked rect with round joins", func(c *ui.Canvas) { c.DrawRect(rect, linePaint(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinRound)) }, kindOutline, false},
		{"stroked rrect", func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectXY(rect, 8, 8), linePaint(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}, kindOutline, false},
		{"stroked rotated rect", func(c *ui.Canvas) {
			c.Rotate(0.3)
			c.DrawRect(rect, linePaint(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}, kindRRect, true},
		{"stroke wider than the rect", func(c *ui.Canvas) {
			c.DrawRect(ui.NewRectFromLTWH(10, 10, 6, 6), linePaint(0xFF000000, 12, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}, kindBox, false},
		{"color", func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFF112233), ui.BlendModeSrcOver) }, kindFill, false},
		{"paint", func(c *ui.Canvas) { c.DrawPaint(solidPaint(0x80112233)) }, kindFill, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := mustBuild(t, tt.draw)
			if b.frame.items(pipeShape) != 1 {
				t.Fatalf("%d instances, want 1", b.frame.items(pipeShape))
			}
			in := instanceAt(b, 0)
			if in[10] != tt.kind {
				t.Errorf("kind = %v, want %v", in[10], tt.kind)
			}
			if got := int(in[11])&flagHasB != 0; got != tt.hasB {
				t.Errorf("has B = %v, want %v", got, tt.hasB)
			}
		})
	}
}

func TestBuildStrokeWidths(t *testing.T) {
	rect := ui.NewRectFromLTWH(20, 20, 60, 40)
	tests := []struct {
		name  string
		width float64
		scale float64
		hw    float32 // half width in the shape's units
		alpha float32
	}{
		{"wide", 6, 1, 3, 1},
		{"one pixel", 1, 1, 0.5, 1},
		{"hairline", 0, 1, 0.5, 1},
		{"thin: one pixel wide with less alpha", 0.25, 1, 0.5, 0.25},
		{"hairline under a scale", 0, 4, 0.125, 1},
		{"a width of a pixel and a half under a scale", 0.5, 3, 3.0 / 12, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := mustBuild(t, func(c *ui.Canvas) {
				c.Scale(tt.scale, tt.scale)
				c.DrawRRect(ui.NewRRectFromRectXY(rect, 8, 8), linePaint(0xFF000000, tt.width, ui.StrokeCapButt, ui.StrokeJoinMiter))
			})
			in := instanceAt(b, 0)
			if in[44] != tt.hw {
				t.Errorf("half width = %v, want %v", in[44], tt.hw)
			}
			if in[15] != tt.alpha {
				t.Errorf("alpha = %v, want %v", in[15], tt.alpha)
			}
		})
	}
}

func TestBuildLinesAndPoints(t *testing.T) {
	count := func(mode ui.PointMode, cap ui.StrokeCap, pts ...ui.Offset) (n int, last []float32) {
		b := mustBuild(t, func(c *ui.Canvas) {
			c.DrawPoints(mode, pts, linePaint(0xFF000000, 8, cap, ui.StrokeJoinMiter))
		})
		n = b.frame.items(pipeShape)
		if n > 0 {
			last = instanceAt(b, n-1)
		}
		return
	}
	p := func(x, y float64) ui.Offset { return ui.Offset{Dx: x, Dy: y} }
	if n, last := count(ui.PointModePoints, ui.StrokeCapRound, p(10, 10), p(20, 20), p(30, 30)); n != 3 || last[45] != 1 {
		t.Errorf("round points: %d instances, cap %v", n, last)
	}
	// A point with a butt cap is a square, as Skia draws it.
	if n, last := count(ui.PointModePoints, ui.StrokeCapButt, p(10, 10)); n != 1 || last[45] != 2 {
		t.Errorf("butt point: %d instances", n)
	}
	if n, _ := count(ui.PointModeLines, ui.StrokeCapRound, p(0, 0), p(10, 10), p(20, 20), p(30, 30), p(40, 40)); n != 2 {
		t.Errorf("lines of five points: %d instances, want 2 (a last point alone draws nothing)", n)
	}
	if n, _ := count(ui.PointModePoints, ui.StrokeCapRound); n != 0 {
		t.Errorf("no points: %d instances", n)
	}
	// A line of no length with a butt cap draws nothing; with a cap it is a dot.
	b := mustBuild(t, func(c *ui.Canvas) {
		c.DrawLine(p(5, 5), p(5, 5), linePaint(0xFF000000, 8, ui.StrokeCapButt, ui.StrokeJoinMiter))
	})
	if b.frame.items(pipeShape) != 0 {
		t.Errorf("a butt-capped dot drew %d instances", b.frame.items(pipeShape))
	}
	b = mustBuild(t, func(c *ui.Canvas) {
		c.DrawLine(p(5, 5), p(5, 5), linePaint(0xFF000000, 8, ui.StrokeCapSquare, ui.StrokeJoinMiter))
	})
	if b.frame.items(pipeShape) != 1 {
		t.Errorf("a square-capped dot drew %d instances", b.frame.items(pipeShape))
	}
}

func TestBuildBatching(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 20, 20)
	over, dstOver, plus := solidPaint(0xFF000000), solidPaint(0xFF000000), solidPaint(0xFF000000)
	dstOver.SetBlendMode(ui.BlendModeDstOver)
	plus.SetBlendMode(ui.BlendModePlus)
	b := mustBuild(t, func(c *ui.Canvas) {
		c.DrawRect(rect, over)
		c.DrawCircle(ui.Offset{Dx: 50, Dy: 50}, 5, over) // same blend and clip: one batch
		c.DrawRect(rect, dstOver)                        // new blend
		c.DrawRect(rect, over)
		c.Save()
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 100, 50)) // new scissor
		c.DrawRect(rect, over)
		c.DrawRect(rect, over)
		c.Restore()
		c.DrawRect(rect, plus)
	})
	var counts []int
	for _, bt := range b.frame.calls {
		counts = append(counts, bt.count)
	}
	want := []int{2, 1, 1, 2, 1}
	if len(counts) != len(want) {
		t.Fatalf("batch sizes = %v, want %v", counts, want)
	}
	for i := range want {
		if counts[i] != want[i] {
			t.Fatalf("batch sizes = %v, want %v", counts, want)
		}
	}
	if b.frame.calls[3].scissor != [4]int{0, 0, 100, 50} || b.frame.calls[0].scissor != [4]int{0, 0, 200, 100} {
		t.Errorf("scissors = %v, %v", b.frame.calls[0].scissor, b.frame.calls[3].scissor)
	}
	// Batches are consecutive ranges of the instance data.
	next := 0
	for _, bt := range b.frame.calls {
		if bt.first != next {
			t.Errorf("batch starts at %d, want %d", bt.first, next)
		}
		next += bt.count
	}
	if next != b.frame.items(pipeShape) {
		t.Errorf("batches hold %d instances of %d", next, b.frame.items(pipeShape))
	}
	if b.frame.calls[2].blend != blendFuncs[ui.BlendModeSrcOver] || b.frame.calls[1].blend != blendFuncs[ui.BlendModeDstOver] {
		t.Error("the blend of a batch is the blend of its paint")
	}
}

func TestBuildClipsAndCulling(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 20, 20)
	// Drawing outside the clip makes no instance; a shape partly inside is cut to the clip's scissor.
	b := mustBuild(t, func(c *ui.Canvas) {
		c.ClipRect(ui.NewRectFromLTWH(50.5, 50.5, 20, 20))
		c.DrawRect(rect, solidPaint(0xFF000000)) // outside
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 60, 60), solidPaint(0xFF000000))
	})
	if b.frame.items(pipeShape) != 1 {
		t.Fatalf("%d instances, want 1", b.frame.items(pipeShape))
	}
	in := instanceAt(b, 0)
	if in[0] != 50 || in[1] != 50 || in[2] != 61 || in[3] != 61 {
		t.Errorf("quad = %v, want it cut to the clip's pixels 50..71 and the shape's 60 (+1)", in[0:4])
	}
	if in[16] != 50.5 || in[18] != 70.5 {
		t.Errorf("clip rect = %v, want the fractional clip for the shader", in[16:20])
	}
	if got := b.frame.calls[0].scissor; got != [4]int{50, 50, 21, 21} {
		t.Errorf("scissor = %v, want the clip's outer pixels", got)
	}
	// A clip with nothing in it draws nothing, and is not an unsupported feature.
	b = mustBuild(t, func(c *ui.Canvas) {
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
		c.ClipRect(ui.NewRectFromLTWH(50, 50, 10, 10))
		c.DrawRect(rect, solidPaint(0xFF000000))
	})
	if b.frame.items(pipeShape) != 0 || b.stats.Skipped != 0 {
		t.Errorf("%d instances, %d skipped", b.frame.items(pipeShape), b.stats.Skipped)
	}
	// Shapes that are off the surface or have no size draw nothing.
	b = mustBuild(t, func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(500, 500, 10, 10), solidPaint(0xFF000000))
		c.DrawRect(ui.NewRectFromLTWH(5, 5, 0, 10), solidPaint(0xFF000000))
		c.DrawCircle(ui.Offset{Dx: 5, Dy: 5}, 0, solidPaint(0xFF000000))
		c.Scale(0, 1)
		c.DrawRect(rect, solidPaint(0xFF000000))
	})
	if b.frame.items(pipeShape) != 0 {
		t.Errorf("%d instances, want 0", b.frame.items(pipeShape))
	}
}

func TestBuildDrawsNothingForDst(t *testing.T) {
	p := solidPaint(0xFF000000)
	p.SetBlendMode(ui.BlendModeDst)
	b := mustBuild(t, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), p) })
	if b.frame.items(pipeShape) != 0 || b.stats.Skipped != 0 || b.stats.Drawn != 1 {
		t.Errorf("instances %d, stats %+v", b.frame.items(pipeShape), b.stats)
	}
}

// manyColorsPaint is a paint with a gradient of more colors than a shader can hold:
// the one paint that the renderer cannot draw.
func manyColorsPaint() *ui.Paint {
	p := solidPaint(0xFF000000)
	colors := make([]ui.Color, maxStops+1)
	stops := make([]float64, len(colors))
	for i := range colors {
		colors[i], stops[i] = ui.NewColor(0xFF000000+uint32(i)), float64(i)/float64(len(colors)-1)
	}
	p.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, colors, ui.GradientLinearOpts{ColorStops: stops}))
	return p
}

// unknownBlendPaint is a paint with a blend mode that no renderer knows, which cannot
// be drawn, and unknownBlendFeature the feature it is reported as.
func unknownBlendPaint() *ui.Paint {
	p := solidPaint(0x80000000)
	p.SetBlendMode(ui.BlendMode(200))
	return p
}

var unknownBlendFeature = "blend mode " + ui.BlendMode(200).String()

func TestBuildUnsupported(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 60, 40)
	manyColors := manyColorsPaint()
	tests := []struct {
		name    string
		feature string
		draw    func(c *ui.Canvas)
	}{
		{"path", "operation ui.OpDrawPath", func(c *ui.Canvas) {
			p := ui.NewPath()
			p.AddRect(rect)
			c.DrawPath(p, solidPaint(0xFF000000))
		}},
		{"arc", "operation ui.OpDrawArc", func(c *ui.Canvas) { c.DrawArc(rect, 0, 1, true, solidPaint(0xFF000000)) }},
		{"gradient with too many colors", "gradient with more than 32 colors", func(c *ui.Canvas) { c.DrawRect(rect, manyColors) }},
		{"polygon points", "polygon points", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePolygon, []ui.Offset{{}, {Dx: 5, Dy: 5}, {Dx: 9}}, linePaint(0xFF000000, 2, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"bevel join", "bevel stroke join", func(c *ui.Canvas) { c.DrawRect(rect, linePaint(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinBevel)) }},
		{"miter limit", "stroke miter limit below the corner's miter", func(c *ui.Canvas) {
			p := linePaint(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter)
			p.SetStrokeMiterLimit(1.2)
			c.DrawRect(rect, p)
		}},
		{"stroked drrect", "stroked DRRect", func(c *ui.Canvas) {
			c.DrawDRRect(ui.NewRRectFromRectXY(rect, 8, 8), ui.NewRRectFromRectXY(rect.Deflate(6), 4, 4), linePaint(0xFF000000, 2, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"stroked superellipse", "stroked superellipse", func(c *ui.Canvas) {
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(rect, 8, 8), linePaint(0xFF000000, 2, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"perspective", "perspective transform", func(c *ui.Canvas) {
			m := ui.NewMatrix4Identity()
			m.SetEntry(3, 0, 0.001)
			c.Transform(m)
			c.DrawRect(rect, solidPaint(0xFF000000))
		}},
		{"clip shape", "clip shape", func(c *ui.Canvas) {
			c.ClipRRect(ui.NewRRectFromRectXY(rect, 8, 8))
			c.DrawRect(rect, solidPaint(0xFF000000))
		}},
		{"image", "operation ui.OpDrawImageRect", func(c *ui.Canvas) {
			img := ui.NewImageFromRGBA(make([]byte, 16), 2, 2)
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 2, 2), rect, solidPaint(0xFF000000))
		}},
		{"layer with a paint that cannot be drawn", unknownBlendFeature, func(c *ui.Canvas) {
			c.SaveLayer(nil, unknownBlendPaint())
			c.DrawRect(rect, solidPaint(0xFF000000))
			c.Restore()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Strict: GLNotImplemented, naming the feature, before anything is built.
			_, err := build(t, false, tt.draw)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != "GLNotImplemented" || appErr.Metadata["feature"] != tt.feature || !appErr.HasRoute("frameBuilder.unsupported.GLNotImplemented") {
				t.Fatalf("error = %v, want GLNotImplemented for %q", err, tt.feature)
			}
			// Skipping: the feature is listed once, and the command is counted as skipped.
			b, err := build(t, true, func(c *ui.Canvas) {
				tt.draw(c)
				tt.draw(c)
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(b.stats.Unsupported) != 1 || b.stats.Unsupported[0] != tt.feature {
				t.Errorf("unsupported = %v, want [%s]", b.stats.Unsupported, tt.feature)
			}
			if b.stats.Skipped == 0 || b.stats.Drawn != 0 || b.frame.items(pipeShape) != 0 {
				t.Errorf("stats %+v, %d instances", b.stats, b.frame.items(pipeShape))
			}
		})
	}
}

func TestBuildSkipsWholeLayers(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 10, 20, 20)
	b, err := build(t, true, func(c *ui.Canvas) {
		c.DrawRect(rect, solidPaint(0xFF000000))
		c.SaveLayer(nil, unknownBlendPaint())
		c.DrawRect(rect, solidPaint(0xFF000000)) // inside the layer
		c.SaveLayer(nil, solidPaint(0x80000000))
		c.DrawRect(rect, solidPaint(0xFF000000)) // inside the nested layer
		c.Restore()
		c.DrawRect(rect, solidPaint(0xFF000000)) // still inside the first
		c.Restore()
		c.DrawRect(rect, solidPaint(0xFF000000)) // after the layers
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.stats.Drawn != 2 || b.stats.Skipped != 3 || b.stats.Commands != 5 || b.frame.items(pipeShape) != 2 {
		t.Errorf("stats %+v, %d instances", b.stats, b.frame.items(pipeShape))
	}
}

func TestPartsOf(t *testing.T) {
	tests := []struct {
		name  string
		rect  ui.Rect
		radii [4]ui.Radius
		want  rrectParts
		ok    bool
	}{
		{"plain", ui.NewRectFromLTWH(10, 20, 40, 60), [4]ui.Radius{}, rrectParts{cx: 30, cy: 50, hx: 20, hy: 30}, true},
		{"edges swapped", ui.Rect{Left: 50, Top: 80, Right: 10, Bottom: 20}, [4]ui.Radius{}, rrectParts{cx: 30, cy: 50, hx: 20, hy: 30}, true},
		{"a radius with a zero side is square", ui.NewRectFromLTWH(0, 0, 40, 40), [4]ui.Radius{{X: 5, Y: 0}, {X: 0, Y: 5}, {X: -1, Y: 5}, {X: 5, Y: 5}},
			rrectParts{cx: 20, cy: 20, hx: 20, hy: 20, r: [4]ui.Radius{{}, {}, {}, {X: 5, Y: 5}}}, true},
		{"radii scaled to fit", ui.NewRectFromLTWH(0, 0, 40, 100), [4]ui.Radius{{X: 40, Y: 40}, {X: 40, Y: 40}, {X: 40, Y: 40}, {X: 40, Y: 40}},
			rrectParts{cx: 20, cy: 50, hx: 20, hy: 50, r: [4]ui.Radius{{X: 20, Y: 20}, {X: 20, Y: 20}, {X: 20, Y: 20}, {X: 20, Y: 20}}}, true},
		{"scaled by the tightest side", ui.NewRectFromLTWH(0, 0, 100, 30), [4]ui.Radius{{X: 60, Y: 30}, {X: 60, Y: 10}, {X: 10, Y: 10}, {X: 10, Y: 10}},
			rrectParts{cx: 50, cy: 15, hx: 50, hy: 15, r: [4]ui.Radius{{X: 45, Y: 22.5}, {X: 45, Y: 7.5}, {X: 7.5, Y: 7.5}, {X: 7.5, Y: 7.5}}}, true},
		{"infinite", ui.Rect{Left: 0, Top: 0, Right: math.Inf(1), Bottom: 1}, [4]ui.Radius{}, rrectParts{}, false},
		{"NaN", ui.Rect{Left: math.NaN(), Top: 0, Right: 1, Bottom: 1}, [4]ui.Radius{}, rrectParts{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := partsOf(tt.rect, tt.radii)
			if ok != tt.ok {
				t.Fatalf("ok = %v", ok)
			}
			if !ok {
				return
			}
			near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
			if !near(got.cx, tt.want.cx) || !near(got.cy, tt.want.cy) || !near(got.hx, tt.want.hx) || !near(got.hy, tt.want.hy) {
				t.Errorf("parts = %+v, want %+v", got, tt.want)
			}
			for i := range got.r {
				if !near(got.r[i].X, tt.want.r[i].X) || !near(got.r[i].Y, tt.want.r[i].Y) {
					t.Errorf("radii = %v, want %v", got.r, tt.want.r)
					break
				}
			}
		})
	}
}

func TestAffine(t *testing.T) {
	tests := []struct {
		name       string
		m          affine2D
		invertible bool
		similarity bool
	}{
		{"identity", affine2D{a: 1, d: 1}, true, true},
		{"uniform scale", affine2D{a: 2, d: 2, e: 5}, true, true},
		{"rotation", affine2D{a: math.Cos(0.5), b: math.Sin(0.5), c: -math.Sin(0.5), d: math.Cos(0.5)}, true, true},
		{"flip", affine2D{a: -1, d: 1}, true, true},
		{"stretch", affine2D{a: 2, d: 1}, true, false},
		{"skew", affine2D{a: 1, c: 0.5, d: 1}, true, false},
		{"singular", affine2D{a: 1}, false, false},
		{"NaN", affine2D{a: math.NaN(), d: 1}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, ok := tt.m.invert()
			if ok != tt.invertible {
				t.Fatalf("invertible = %v", ok)
			}
			if ok {
				// The inverse undoes the transform.
				x, y := 3.0, -7.0
				u, v := tt.m.a*x+tt.m.c*y+tt.m.e, tt.m.b*x+tt.m.d*y+tt.m.f
				bx, by := inv.a*u+inv.c*v+inv.e, inv.b*u+inv.d*v+inv.f
				if math.Abs(bx-x) > 1e-9 || math.Abs(by-y) > 1e-9 {
					t.Errorf("inverse maps back to %v, %v", bx, by)
				}
				if got := tt.m.similarity(); got != tt.similarity {
					t.Errorf("similarity = %v", got)
				}
			}
		})
	}
}
