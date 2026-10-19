package painting

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func drawOps(ops []ui.DisplayOp) []ui.OpDrawImageRect {
	var out []ui.OpDrawImageRect
	for _, op := range ops {
		if d, ok := op.(ui.OpDrawImageRect); ok {
			out = append(out, d)
		}
	}
	return out
}

func TestDecorationImagePainter(t *testing.T) {
	pt := newPaintingTester(t)
	data := pngBytes(t, 10, 10, red)
	cfg := ImageConfiguration{Binding: pt.PaintingBinding}
	changed := 0
	di := NewDecorationImage(NewMemoryImage(data), DecorationImageOpts{Fit: BoxFitFill})
	painter := di.CreatePainter(func() { changed++ })
	rect := lrtb(0, 0, 40, 20)

	// Before the image arrives nothing is drawn.
	ops := recordOps(func(c *ui.Canvas) { painter.Paint(c, rect, nil, cfg) })
	if len(drawOps(ops)) != 0 {
		t.Fatalf("drew %v before the image loaded", ops)
	}
	pt.waitFor(t, "onChanged", func() bool { return changed == 1 })
	ops = recordOps(func(c *ui.Canvas) { painter.Paint(c, rect, nil, cfg) })
	draws := drawOps(ops)
	if len(draws) != 1 || draws[0].Dst != rect {
		t.Fatalf("ops = %v", ops)
	}
	if changed != 1 {
		t.Errorf("onChanged = %d, want once", changed)
	}
	// A clip path is applied around the image.
	clip := ui.NewPath()
	clip.AddRect(lrtb(0, 0, 5, 5))
	ops = recordOps(func(c *ui.Canvas) { painter.Paint(c, rect, clip, cfg) })
	if _, ok := ops[0].(ui.OpSave); !ok {
		t.Errorf("first op = %T, want a save", ops[0])
	}
	if _, ok := ops[1].(ui.OpClipPath); !ok {
		t.Errorf("second op = %T, want a clip", ops[1])
	}
	if _, ok := ops[len(ops)-1].(ui.OpRestore); !ok {
		t.Errorf("last op = %T, want a restore", ops[len(ops)-1])
	}
	painter.Dispose()
	if pt.ImageCache().LiveImageCount() != 0 {
		t.Errorf("live images after dispose = %d", pt.ImageCache().LiveImageCount())
	}
}

func TestDecorationImagePainterMatchTextDirection(t *testing.T) {
	pt := newPaintingTester(t)
	di := NewDecorationImage(NewMemoryImage(pngBytes(t, 10, 10, red)), DecorationImageOpts{MatchTextDirection: true, Fit: BoxFitNone, Alignment: AlignmentCenterLeft})
	tests := []struct {
		name    string
		dir     ui.TextDirection
		flipped bool
	}{
		{"ltr", ui.TextDirectionLTR, false},
		{"rtl", ui.TextDirectionRTL, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := di.CreatePainter(func() {})
			defer p.Dispose()
			cfg := ImageConfiguration{Binding: pt.PaintingBinding, TextDirection: tt.dir}
			rect := lrtb(0, 0, 100, 100)
			recordOps(func(c *ui.Canvas) { p.Paint(c, rect, nil, cfg) })
			pt.waitFor(t, "the image", func() bool {
				return len(drawOps(recordOps(func(c *ui.Canvas) { p.Paint(c, rect, nil, cfg) }))) == 1
			})
			ops := recordOps(func(c *ui.Canvas) { p.Paint(c, rect, nil, cfg) })
			transforms := 0
			for _, op := range ops {
				if _, ok := op.(ui.OpTransform); ok {
					transforms++
				}
			}
			if flipped := transforms == 3; flipped != tt.flipped {
				t.Errorf("flipped = %v, want %v (ops %v)", flipped, tt.flipped, ops)
			}
		})
	}
}

func TestDecorationImagePainterNewStreamReplacesOld(t *testing.T) {
	pt := newPaintingTester(t)
	first := NewMemoryImage(pngBytes(t, 4, 4, red))
	di := NewDecorationImage(first)
	p := di.CreatePainter(func() {}).(*decorationImagePainter)
	cfg := ImageConfiguration{Binding: pt.PaintingBinding}
	recordOps(func(c *ui.Canvas) { p.Paint(c, lrtb(0, 0, 4, 4), nil, cfg) })
	pt.waitFor(t, "the image", func() bool { return p.image != nil })
	stream := p.imageStream
	recordOps(func(c *ui.Canvas) { p.Paint(c, lrtb(0, 0, 4, 4), nil, cfg) })
	if p.imageStream.Key() != stream.Key() {
		t.Error("the same image keeps its stream")
	}
	p.Dispose()
}

func TestDecorationImageString(t *testing.T) {
	tests := []struct {
		name string
		di   *DecorationImage
		want string
	}{
		{"defaults", NewDecorationImage(NewFileImage("a.png")),
			`DecorationImage(FileImage("a.png", scale: 1.0), Alignment.center, scale 1.0, opacity 1.0, FilterQuality.medium)`},
		{"all", NewDecorationImage(NewFileImage("a.png"), DecorationImageOpts{
			Fit: BoxFitCover, Alignment: AlignmentTopLeft, CenterSlice: foundation.Some(lrtb(1, 2, 3, 4)), Repeat: ImageRepeatRepeatX,
			MatchTextDirection: true, Scale: foundation.Some(2.0), Opacity: foundation.Some(0.5), FilterQuality: ui.FilterQualityLow,
			InvertColors: true, IsAntiAlias: true}),
			`DecorationImage(FileImage("a.png", scale: 1.0), BoxFit.cover, Alignment.topLeft, centerSlice: Rect.fromLTRB(1.0, 2.0, 3.0, 4.0), ` +
				`ImageRepeat.repeatX, match text direction, scale 2.0, opacity 0.5, FilterQuality.low, invert colors, use anti-aliasing)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.di.String(); got != tt.want {
				t.Errorf("String =\n %s\nwant\n %s", got, tt.want)
			}
		})
	}
}

func TestDecorationImageEqualAndLerp(t *testing.T) {
	p := NewFileImage("a.png")
	a := NewDecorationImage(p)
	same := NewDecorationImage(p)
	other := NewDecorationImage(p, DecorationImageOpts{Fit: BoxFitCover})
	if !a.Equal(same) || a.Equal(other) || a.Equal(nil) {
		t.Error("Equal compares the fields")
	}
	if got := DecorationImageLerp(a, other, 0); got != DecorationImageSource(a) {
		t.Error("lerp at 0 is a")
	}
	if got := DecorationImageLerp(a, other, 1); got != DecorationImageSource(other) {
		t.Error("lerp at 1 is b")
	}
	mid := DecorationImageLerp(a, other, 0.5)
	if _, ok := mid.(*blendedDecorationImage); !ok {
		t.Fatalf("lerp = %T", mid)
	}
	if !mid.Equal(DecorationImageLerp(a, other, 0.5)) || mid.Equal(DecorationImageLerp(a, other, 0.6)) || mid.Equal(a) {
		t.Error("blended images are equal when a, b and t are")
	}
	if DecorationImageLerp(nil, nil, 0.5) != nil {
		t.Error("lerp of nothing is nothing")
	}
}

func TestBlendedDecorationImagePainter(t *testing.T) {
	pt := newPaintingTester(t)
	cfg := ImageConfiguration{Binding: pt.PaintingBinding}
	a := NewDecorationImage(NewMemoryImage(pngBytes(t, 4, 4, red)), DecorationImageOpts{Fit: BoxFitFill})
	b := NewDecorationImage(NewMemoryImage(pngBytes(t, 6, 6, red)), DecorationImageOpts{Fit: BoxFitFill})
	tests := []struct {
		name       string
		from, to   DecorationImageSource
		wantDraws  int
		wantAlphas []float64
		wantModes  []ui.BlendMode
	}{
		{"both", a, b, 2, []float64{0.75, 0.25}, []ui.BlendMode{ui.BlendModeSrcOver, ui.BlendModePlus}},
		{"from nothing", nil, b, 1, []float64{0.25}, []ui.BlendMode{ui.BlendModeSrcOver}},
		{"to nothing", a, nil, 1, []float64{0.75}, []ui.BlendMode{ui.BlendModeSrcOver}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blended := DecorationImageLerp(tt.from, tt.to, 0.25)
			p := blended.CreatePainter(func() {})
			defer p.Dispose()
			rect := lrtb(0, 0, 8, 8)
			var draws []ui.OpDrawImageRect
			pt.waitFor(t, "the images", func() bool {
				draws = drawOps(recordOps(func(c *ui.Canvas) { p.Paint(c, rect, nil, cfg) }))
				return len(draws) == tt.wantDraws
			})
			for i, d := range draws {
				if !approx(float64(d.Paint.Color().A), tt.wantAlphas[i], 0.01) || d.Paint.BlendMode() != tt.wantModes[i] {
					t.Errorf("draw %d: alpha %v mode %v, want %v %v", i, d.Paint.Color().A, d.Paint.BlendMode(), tt.wantAlphas[i], tt.wantModes[i])
				}
			}
			ops := recordOps(func(c *ui.Canvas) { p.Paint(c, rect, nil, cfg) })
			if _, ok := ops[0].(ui.OpSaveLayer); !ok {
				t.Errorf("a blend paints in a layer, first op %T", ops[0])
			}
		})
	}
}

func TestImageRepeatNames(t *testing.T) {
	for r, want := range map[ImageRepeat]string{ImageRepeatRepeat: "ImageRepeat.repeat", ImageRepeatRepeatX: "ImageRepeat.repeatX",
		ImageRepeatRepeatY: "ImageRepeat.repeatY", ImageRepeatNoRepeat: "ImageRepeat.noRepeat", ImageRepeatUnspecified: "ImageRepeat.noRepeat"} {
		if got := r.String(); got != want {
			t.Errorf("%d: %s, want %s", r, got, want)
		}
	}
	if !strings.Contains(NewDecorationImage(NewFileImage("x")).String(), "FilterQuality.medium") {
		t.Error("the default quality is medium")
	}
}
