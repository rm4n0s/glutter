package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// observedClip paints the (laid out) box into a fresh context, like
// TestClipPaintingContext.paintChild of rendering_tester.dart, and returns the
// clip behavior that it clipped with: ClipNone if it did not clip.
func observedClip(t *testing.T, h *renderingtest.Harness, box rendering.Box) ui.Clip {
	t.Helper()
	root := rendering.NewContainerLayer()
	ctx := rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 800, 600))
	box.Paint(ctx, ui.OffsetZero)
	ctx.StopRecordingIfNeeded()
	clip := ui.ClipNone
	for _, l := range root.DepthFirstIterateChildren() {
		switch layer := l.(type) {
		case *rendering.ClipRectLayer:
			clip = layer.ClipBehavior()
		case *rendering.PictureLayer:
			if layer.Picture() == nil {
				continue
			}
			for _, op := range layer.Picture().DisplayList().Ops() {
				switch op := op.(type) {
				case ui.OpClipRect:
					if clip == ui.ClipNone {
						clip = ui.ClipHardEdge
						if op.AntiAlias {
							clip = ui.ClipAntiAlias
						}
					}
				case ui.OpSaveLayer:
					if clip == ui.ClipAntiAlias {
						clip = ui.ClipAntiAliasWithSaveLayer
					}
				}
			}
		}
	}
	return clip
}

// clipCases are the clip behaviors a render object is tested with, and what the
// null of Dart (ClipUnspecified) stands for.
func clipCases(def ui.Clip) []struct{ clip, want ui.Clip } {
	return []struct{ clip, want ui.Clip }{
		{ui.ClipUnspecified, def},
		{ui.ClipNone, ui.ClipNone},
		{ui.ClipHardEdge, ui.ClipHardEdge},
		{ui.ClipAntiAlias, ui.ClipAntiAlias},
		{ui.ClipAntiAliasWithSaveLayer, ui.ClipAntiAliasWithSaveLayer},
	}
}

// recordOps paints the (laid out) box into a fresh context and returns the
// operations of its display lists, in order.
func recordOps(box rendering.Box) []any {
	root := rendering.NewContainerLayer()
	ctx := rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 800, 600))
	box.Paint(ctx, ui.OffsetZero)
	ctx.StopRecordingIfNeeded()
	var ops []any
	for _, l := range root.DepthFirstIterateChildren() {
		if pl, ok := l.(*rendering.PictureLayer); ok && pl.Picture() != nil {
			for _, op := range pl.Picture().DisplayList().Ops() {
				ops = append(ops, op)
			}
		}
	}
	return ops
}

// countOps counts the operations of the type.
func countOps[T any](ops []any) int {
	n := 0
	for _, op := range ops {
		if _, ok := op.(T); ok {
			n++
		}
	}
	return n
}

// displayOps returns the operations of the picture.
func displayOps(p *ui.Picture) []any {
	var ops []any
	for _, op := range p.DisplayList().Ops() {
		ops = append(ops, op)
	}
	return ops
}
