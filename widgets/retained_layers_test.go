package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// countingPainter draws a rectangle and counts how many times it was asked to.
type countingPainter struct {
	rendering.CustomPainterBase
	paints int
	color  uint32
}

func (p *countingPainter) Paint(c *ui.Canvas, size ui.Size) {
	p.paints++
	paint := ui.NewPaint()
	paint.SetColor(ui.NewColor(p.color + uint32(p.paints)))
	c.DrawRect(ui.NewRectFromLTWH(0, 0, size.Width, size.Height), paint)
}

func (p *countingPainter) ShouldRepaint(rendering.CustomPainter) bool { return true }

// transformLayers collects the layers a renderer may keep as textures that are not the
// root, with something drawn under them.
func transformLayers(l ui.Layer, out map[ui.Layer]bool) {
	if _, ok := l.(*ui.TransformLayer); ok {
		out[l] = true
	}
	for _, c := range l.Children() {
		transformLayers(c, out)
	}
}

// TestRepaintBoundaryLayerIsRetainedUntilItsSubtreeChanges checks the framework side of
// the raster cache's rule (SPEC §9.1): the layer of a RepaintBoundary whose subtree is
// not repainted is the same layer object in every scene (that is what the GPU renderer
// counts: three scenes in a row), and the one beside it, which repaints each frame, is
// a new layer each time.
func TestRepaintBoundaryLayerIsRetainedUntilItsSubtreeChanges(t *testing.T) {
	still := &countingPainter{color: 0xFF100000}
	repaint := foundation.NewChangeNotifier()
	changing := &countingPainter{CustomPainterBase: rendering.NewCustomPainterBase(repaint), color: 0xFF001000}

	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	ts.PumpWidget(widgets.NewColumn([]widgets.Widget{
		widgets.NewRepaintBoundary(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: still, Size: ui.Size{Width: 100, Height: 40}})),
		widgets.NewRepaintBoundary(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: changing, Size: ui.Size{Width: 100, Height: 40}})),
	}))
	var runs []map[ui.Layer]bool
	record := func() {
		t.Helper()
		scenes := ts.Scenes()
		m := map[ui.Layer]bool{}
		transformLayers(scenes[len(scenes)-1].Root(), m)
		runs = append(runs, m)
	}
	record()
	for i := 0; i < 3; i++ {
		repaint.NotifyListeners()
		ts.Pump()
		record()
	}
	if still.paints != 1 || changing.paints != 4 {
		t.Fatalf("paints: still %d (want 1), changing %d (want 4)", still.paints, changing.paints)
	}
	// Some layer is in all four scenes: the still boundary's. The changing one's is
	// new in every scene, so exactly one layer is shared by consecutive scenes.
	shared := 0
	for l := range runs[0] {
		if runs[1][l] && runs[2][l] && runs[3][l] {
			shared++
		}
	}
	if shared != 1 {
		t.Errorf("%d layers are in four consecutive scenes, want exactly the still boundary's", shared)
	}
	for i := 1; i < len(runs); i++ {
		fresh := 0
		for l := range runs[i] {
			if !runs[i-1][l] {
				fresh++
			}
		}
		if fresh == 0 {
			t.Errorf("scene %d has no new layer: the repainted boundary should make one", i)
		}
	}
}
