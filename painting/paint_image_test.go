package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func recordPaintImage(img *ui.Image, rect ui.Rect, o PaintImageOpts) []ui.DisplayOp {
	return recordOps(func(c *ui.Canvas) { PaintImage(c, rect, img, o) })
}

func lrtb(l, t, r, b float64) ui.Rect { return ui.NewRectFromLTRB(l, t, r, b) }

func TestPaintImageFit(t *testing.T) {
	img := testImage(100, 50)
	outer := lrtb(0, 0, 200, 200)
	tests := []struct {
		name    string
		opts    PaintImageOpts
		wantSrc ui.Rect
		wantDst ui.Rect
	}{
		{"fill", PaintImageOpts{Fit: BoxFitFill}, lrtb(0, 0, 100, 50), lrtb(0, 0, 200, 200)},
		{"contain", PaintImageOpts{Fit: BoxFitContain}, lrtb(0, 0, 100, 50), lrtb(0, 50, 200, 150)},
		{"cover", PaintImageOpts{Fit: BoxFitCover}, lrtb(25, 0, 75, 50), lrtb(0, 0, 200, 200)},
		{"fitWidth", PaintImageOpts{Fit: BoxFitFitWidth}, lrtb(0, 0, 100, 50), lrtb(0, 50, 200, 150)},
		{"fitHeight", PaintImageOpts{Fit: BoxFitFitHeight}, lrtb(25, 0, 75, 50), lrtb(0, 0, 200, 200)},
		{"none", PaintImageOpts{Fit: BoxFitNone}, lrtb(0, 0, 100, 50), lrtb(50, 75, 150, 125)},
		{"scaleDown is the default and does not enlarge", PaintImageOpts{}, lrtb(0, 0, 100, 50), lrtb(50, 75, 150, 125)},
		{"top left", PaintImageOpts{Fit: BoxFitContain, Alignment: foundation.Some(AlignmentTopLeft)}, lrtb(0, 0, 100, 50), lrtb(0, 0, 200, 100)},
		{"bottom right", PaintImageOpts{Fit: BoxFitContain, Alignment: foundation.Some(AlignmentBottomRight)}, lrtb(0, 0, 100, 50), lrtb(0, 100, 200, 200)},
		{"scale 2", PaintImageOpts{Scale: foundation.Some(2.0)}, lrtb(0, 0, 100, 50), lrtb(75, 87.5, 125, 112.5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := recordPaintImage(img, outer, tt.opts)
			if len(ops) != 1 {
				t.Fatalf("ops = %v", ops)
			}
			op, ok := ops[0].(ui.OpDrawImageRect)
			if !ok {
				t.Fatalf("op = %T", ops[0])
			}
			if !rectNear(op.Src, tt.wantSrc) || !rectNear(op.Dst, tt.wantDst) {
				t.Errorf("src %v dst %v, want src %v dst %v", op.Src, op.Dst, tt.wantSrc, tt.wantDst)
			}
		})
	}
}

func TestPaintImagePaint(t *testing.T) {
	img := testImage(10, 10)
	filter := ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeSrcIn)
	ops := recordPaintImage(img, lrtb(0, 0, 10, 10), PaintImageOpts{
		Opacity: foundation.Some(0.5), ColorFilter: filter, FilterQuality: ui.FilterQualityHigh, InvertColors: true,
		IsAntiAlias: true, BlendMode: ui.BlendModePlus,
	})
	op := ops[0].(ui.OpDrawImageRect)
	p := op.Paint
	if !approx(float64(p.Color().A), 0.5, 0.01) || p.ColorFilter() == nil || p.FilterQuality() != ui.FilterQualityHigh || !p.InvertColors() ||
		!p.IsAntiAlias() || p.BlendMode() != ui.BlendModePlus {
		t.Errorf("paint = %v", &p)
	}
	// Defaults: medium quality, no anti-aliasing, srcOver, opaque.
	def := recordPaintImage(img, lrtb(0, 0, 10, 10), PaintImageOpts{})[0].(ui.OpDrawImageRect).Paint
	if def.FilterQuality() != ui.FilterQualityMedium || def.IsAntiAlias() || def.BlendMode() != ui.BlendModeSrcOver || !approx(float64(def.Color().A), 1, 0.001) {
		t.Errorf("default paint = %v", &def)
	}
	// An opacity out of range is clamped.
	over := recordPaintImage(img, lrtb(0, 0, 10, 10), PaintImageOpts{Opacity: foundation.Some(3.0)})[0].(ui.OpDrawImageRect).Paint
	if !approx(float64(over.Color().A), 1, 0.001) {
		t.Errorf("opacity 3 → alpha %v", over.Color().A)
	}
	if ops := recordPaintImage(img, ui.Rect{}, PaintImageOpts{}); len(ops) != 0 {
		t.Errorf("an empty rect paints nothing, got %v", ops)
	}
}

func TestPaintImageRepeat(t *testing.T) {
	img := testImage(10, 10)
	rect := lrtb(0, 0, 25, 15)
	tests := []struct {
		name   string
		repeat ImageRepeat
		want   int
	}{
		{"noRepeat", ImageRepeatNoRepeat, 1},
		{"repeat", ImageRepeatRepeat, 6},
		{"repeatX", ImageRepeatRepeatX, 3},
		{"repeatY", ImageRepeatRepeatY, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := recordPaintImage(img, rect, PaintImageOpts{Repeat: tt.repeat, Fit: BoxFitNone, Alignment: foundation.Some(AlignmentTopLeft)})
			n := 0
			for _, op := range ops {
				if _, ok := op.(ui.OpDrawImageRect); ok {
					n++
				}
			}
			if n != tt.want {
				t.Errorf("tiles = %d, want %d (ops %v)", n, tt.want, ops)
			}
			clipped := false
			for _, op := range ops {
				if _, ok := op.(ui.OpClipRect); ok {
					clipped = true
				}
			}
			if clipped != (tt.repeat != ImageRepeatNoRepeat) {
				t.Errorf("clipped = %v", clipped)
			}
		})
	}
	// A repeat that has nothing left to repeat is turned off.
	ops := recordPaintImage(img, lrtb(0, 0, 10, 10), PaintImageOpts{Repeat: ImageRepeatRepeat, Fit: BoxFitFill})
	if len(ops) != 1 {
		t.Errorf("a full cover needs no tiles, got %v", ops)
	}
}

func TestGenerateImageTileRects(t *testing.T) {
	tiles := generateImageTileRects(lrtb(0, 0, 25, 15), lrtb(0, 0, 10, 10), ImageRepeatRepeat)
	// x: floor(0/10)=0 .. ceil((25-10)/10)=2; y: 0 .. ceil((15-10)/10)=1.
	if len(tiles) != 6 {
		t.Fatalf("tiles = %v", tiles)
	}
	want := []ui.Rect{lrtb(0, 0, 10, 10), lrtb(0, 10, 10, 20), lrtb(10, 0, 20, 10), lrtb(10, 10, 20, 20), lrtb(20, 0, 30, 10), lrtb(20, 10, 30, 20)}
	for i := range want {
		if tiles[i] != want[i] {
			t.Errorf("tile %d = %v, want %v", i, tiles[i], want[i])
		}
	}
}

func TestPaintImageFlip(t *testing.T) {
	img := testImage(10, 10)
	ops := recordPaintImage(img, lrtb(0, 0, 100, 100), PaintImageOpts{Fit: BoxFitNone, FlipHorizontally: true, Alignment: foundation.Some(AlignmentTopLeft)})
	var save, restore, transforms int
	for _, op := range ops {
		switch op.(type) {
		case ui.OpSave:
			save++
		case ui.OpRestore:
			restore++
		case ui.OpTransform:
			transforms++
		}
	}
	if save != 1 || restore != 1 || transforms != 3 {
		t.Errorf("save %d restore %d transforms %d in %v", save, restore, transforms, ops)
	}
	// With a flipped image the alignment is mirrored: a left alignment puts the
	// destination at the right edge, before the mirror transform moves it back.
	draw := ops[len(ops)-2].(ui.OpDrawImageRect)
	if !approx(draw.Dst.Left, 90, 1e-6) {
		t.Errorf("dst = %v", draw.Dst)
	}
}

func TestPaintImageCenterSlice(t *testing.T) {
	img := testImage(30, 30)
	slice := lrtb(10, 10, 20, 20)
	ops := recordPaintImage(img, lrtb(0, 0, 90, 60), PaintImageOpts{CenterSlice: foundation.Some(slice)})
	var nine []ui.OpDrawImageNine
	for _, op := range ops {
		if n, ok := op.(ui.OpDrawImageNine); ok {
			nine = append(nine, n)
		}
	}
	if len(nine) != 1 {
		t.Fatalf("ops = %v", ops)
	}
	if nine[0].Center != slice || !rectNear(nine[0].Dst, lrtb(0, 0, 90, 60)) {
		t.Errorf("center %v dst %v", nine[0].Center, nine[0].Dst)
	}
	// With a scale the center slice and the destination are scaled, and the canvas is scaled back.
	scaled := recordPaintImage(img, lrtb(0, 0, 90, 60), PaintImageOpts{CenterSlice: foundation.Some(slice), Scale: foundation.Some(2.0)})
	for _, op := range scaled {
		if n, ok := op.(ui.OpDrawImageNine); ok {
			if !rectNear(n.Center, lrtb(20, 20, 40, 40)) {
				t.Errorf("scaled center = %v", n.Center)
			}
		}
	}
	// Repeated nine-patches draw a tile each.
	rep := recordPaintImage(img, lrtb(0, 0, 90, 60), PaintImageOpts{CenterSlice: foundation.Some(slice), Fit: BoxFitContain, Repeat: ImageRepeatRepeat})
	if len(rep) == 0 {
		t.Error("no ops for a repeated nine-patch")
	}
}
