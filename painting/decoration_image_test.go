package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// fakeImage is a DecorationImageSource that records what the decorations do with
// it; step 6.2 adds the real DecorationImage.
type fakeImage struct {
	id       int
	painters []*fakePainter
}

type fakePainter struct {
	onChanged func()
	rects     []ui.Rect
	clips     []*ui.Path
	configs   []ImageConfiguration
	disposed  int
}

func (p *fakePainter) Paint(_ *ui.Canvas, rect ui.Rect, clip *ui.Path, cfg ImageConfiguration) {
	p.rects = append(p.rects, rect)
	p.clips = append(p.clips, clip)
	p.configs = append(p.configs, cfg)
}
func (p *fakePainter) Dispose() { p.disposed++ }

func (f *fakeImage) CreatePainter(onChanged func()) DecorationImagePainter {
	p := &fakePainter{onChanged: onChanged}
	f.painters = append(f.painters, p)
	return p
}

func (f *fakeImage) Blended(a, b DecorationImageSource, t float64) DecorationImageSource {
	return &fakeImage{id: -1}
}

func (f *fakeImage) Equal(o DecorationImageSource) bool {
	x, ok := o.(*fakeImage)
	return ok && x.id == f.id
}

func TestDecorationImageLerp(t *testing.T) {
	a, b := &fakeImage{id: 1}, &fakeImage{id: 2}
	if DecorationImageLerp(nil, nil, 0.5) != nil {
		t.Error("lerp of nils should be nil")
	}
	if DecorationImageLerp(a, b, 0) != DecorationImageSource(a) || DecorationImageLerp(a, b, 1) != DecorationImageSource(b) {
		t.Error("endpoints are wrong")
	}
	if got := DecorationImageLerp(a, &fakeImage{id: 1}, 0.5); got != DecorationImageSource(a) {
		t.Error("equal images should lerp to the first")
	}
	for _, got := range []DecorationImageSource{DecorationImageLerp(a, b, 0.5), DecorationImageLerp(nil, b, 0.5), DecorationImageLerp(a, nil, 0.5)} {
		if got.(*fakeImage).id != -1 {
			t.Errorf("lerp = %v, want the blended image", got)
		}
	}
}

func TestBoxDecorationPaintsImage(t *testing.T) {
	img := &fakeImage{id: 1}
	changed := 0
	d := NewBoxDecoration(BoxDecorationOpts{Image: img, BorderRadius: NewBorderRadiusCircular(5)})
	painter := d.CreateBoxPainter(func() { changed++ })
	cfg := cfgOf(100, 50, ui.TextDirectionLTR)
	recordOps(func(c *ui.Canvas) {
		painter.Paint(c, ui.Offset{Dx: 10, Dy: 20}, cfg)
		painter.Paint(c, ui.Offset{Dx: 10, Dy: 20}, cfg)
	})
	if len(img.painters) != 1 {
		t.Fatalf("created %d painters, want 1", len(img.painters))
	}
	p := img.painters[0]
	if len(p.rects) != 2 || p.rects[0] != ui.NewRectFromLTWH(10, 20, 100, 50) || p.clips[0] == nil {
		t.Errorf("painted %v with clip %v", p.rects, p.clips)
	}
	if !p.clips[0].Contains(ui.Offset{Dx: 50, Dy: 40}) || p.clips[0].Contains(ui.Offset{Dx: 10.5, Dy: 20.5}) {
		t.Error("the clip path should be the rounded rectangle")
	}
	p.onChanged()
	if changed != 1 {
		t.Errorf("changed = %d", changed)
	}
	painter.Dispose()
	if p.disposed != 1 {
		t.Errorf("disposed = %d", p.disposed)
	}
	circle := NewBoxDecoration(BoxDecorationOpts{Image: img, Shape: BoxShapeCircle})
	cp := circle.CreateBoxPainter(func() {})
	recordOps(func(c *ui.Canvas) { cp.Paint(c, ui.Offset{}, cfg) })
	if clip := img.painters[1].clips[0]; clip == nil || clip.Contains(ui.Offset{Dx: 1, Dy: 1}) || !clip.Contains(ui.Offset{Dx: 50, Dy: 25}) {
		t.Error("the clip path of a circle should be the circle")
	}
}

func TestShapeDecorationPaintsImage(t *testing.T) {
	img := &fakeImage{id: 1}
	d := NewShapeDecoration(NewCircleBorder(), ShapeDecorationOpts{Image: img})
	log := 0
	painter := d.CreateBoxPainter(func() { log++ })
	cfg := cfgOf(100, 100, ui.TextDirectionRTL)
	recordOps(func(c *ui.Canvas) { painter.Paint(c, ui.Offset{Dx: 20, Dy: -40}, cfg) })
	p := img.painters[0]
	if len(p.rects) != 1 || p.rects[0] != ui.NewRectFromLTWH(20, -40, 100, 100) || p.configs[0].TextDirection != ui.TextDirectionRTL {
		t.Errorf("painted %v %v", p.rects, p.configs)
	}
	if p.clips[0] == nil || !p.clips[0].Contains(ui.Offset{Dx: 70, Dy: 10}) {
		t.Error("the image should be clipped to the inner path of the shape")
	}
	painter.Dispose()
	if p.disposed != 1 || log != 0 {
		t.Errorf("disposed = %d, log = %d", p.disposed, log)
	}
}

func TestDecorationsDescribeThemselves(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release and profile builds describe objects by their short names")
	}
	if got := NewBoxDecoration().String(); got != "BoxDecoration" {
		t.Errorf("empty BoxDecoration = %q", got)
	}
	d := NewBoxDecoration(BoxDecorationOpts{Color: foundation.Some(colorR), Shape: BoxShapeCircle})
	if got, want := d.String(), "BoxDecoration(color: "+colorR.String()+", shape: circle)"; got != want {
		t.Errorf("BoxDecoration = %q, want %q", got, want)
	}
	s := NewShapeDecoration(NewCircleBorder())
	if got, want := s.String(), "ShapeDecoration(shape: CircleBorder(BorderSide(width: 0.0, style: none)))"; got != want {
		t.Errorf("ShapeDecoration = %q, want %q", got, want)
	}
}
