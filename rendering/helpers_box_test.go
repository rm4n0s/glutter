package rendering_test

import (
	"github.com/rm4n0s/errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// enginePhase is EnginePhase of rendering_tester.dart: how far a frame goes.
type enginePhase int

// The phases, in order; the zero value is layout, the default of layout().
const (
	phaseLayout enginePhase = iota
	phaseCompositingBits
	phasePaint
	phaseComposite
)

// layoutOpts are the optional parameters of layout.
type layoutOpts struct {
	constraints foundation.Opt[rendering.BoxConstraints]
	alignment   painting.AlignmentGeometry
	phase       enginePhase
}

// pumpFrame runs a frame up to the phase.
func pumpFrame(h *renderingtest.Harness, phase enginePhase) {
	if phase == phaseComposite {
		h.Pump()
		return
	}
	h.Owner.FlushLayout()
	if phase >= phaseCompositingBits {
		h.Owner.FlushCompositingBits()
	}
	if phase >= phasePaint {
		h.Owner.FlushPaint()
	}
}

// layout is `layout()` of rendering_tester.dart: the box is the child of the
// render view, inside a positioned box and a constrained box when there are
// constraints, and a frame is pumped.
func layout(t *testing.T, h *renderingtest.Harness, box rendering.Box, opts ...layoutOpts) {
	t.Helper()
	var o layoutOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if c, ok := o.constraints.Get(); ok {
		box = rendering.NewRenderPositionedBox(
			rendering.NewRenderConstrainedBox(c, box),
			rendering.RenderPositionedBoxOpts{Alignment: o.alignment})
	}
	h.SetRoot(box)
	pumpFrame(h, o.phase)
}

func withConstraints(c rendering.BoxConstraints) layoutOpts {
	return layoutOpts{constraints: foundation.Some(c)}
}

func atPhase(o layoutOpts, phase enginePhase) layoutOpts {
	o.phase = phase
	return o
}

// newSizedBox is RenderSizedBox of rendering_tester.dart: a box of a fixed size,
// constrained, that is sized by its parent.
func newSizedBox(w, h float64) *tBox {
	b := newTBox(w, h)
	b.sizedByParent = true
	b.hitSelf = true
	return b
}

// testLayerReuse forces two frames and checks that a layer is created on the
// first frame and reused on the second.
func testLayerReuse[L rendering.ContainerLayerLike](t *testing.T, box rendering.Box) {
	t.Helper()
	h := newHarness(t)
	if box.AsRenderObject().Layer() != nil {
		t.Fatal("the box has a layer before layout")
	}
	layout(t, h, box, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10})), phasePaint))
	layer := box.AsRenderObject().Layer()
	if _, ok := layer.(L); !ok || layer == nil {
		t.Fatalf("layer is %T, want %s", layer, reflect.TypeOf((*L)(nil)).Elem())
	}
	box.MarkNeedsPaint()
	if foundation.DebugMode && !box.AsRenderObject().DebugNeedsPaint() {
		t.Error("the box does not need paint after MarkNeedsPaint")
	}
	pumpFrame(h, phasePaint)
	if foundation.DebugMode && box.AsRenderObject().DebugNeedsPaint() {
		t.Error("the box needs paint after the frame")
	}
	if box.AsRenderObject().Layer() != layer {
		t.Error("the layer was not reused")
	}
}

// tStack is a box that is as big as it can be and paints its children at its
// origin, in order (a bit of RenderStack).
type tStack struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
}

type tStackData struct {
	rendering.ContainerBoxParentData[rendering.Box]
}

func newTStack(children ...rendering.Box) *tStack {
	s := &tStack{}
	rendering.InitRenderBox(&s.RenderBox, s)
	rendering.InitContainerRenderObject(&s.ContainerRenderObject, s)
	s.AddAll(children)
	return s
}

func (s *tStack) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*tStackData); !ok {
		child.AsRenderObject().SetParentData(&tStackData{})
	}
}

func (s *tStack) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }
func (s *tStack) PerformLayout() {
	c := s.BoxConstraints()
	for ch := s.FirstChild(); ch != nil; ch = ch.ParentData().(*tStackData).NextSibling {
		ch.Layout(c.Loosen(), rendering.LayoutOpts{ParentUsesSize: true})
	}
	s.SetSize(c.Biggest())
}
func (s *tStack) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	for ch := s.FirstChild(); ch != nil; ch = ch.ParentData().(*tStackData).NextSibling {
		ctx.PaintChild(ch, offset.Add(rendering.BoxParentDataOf(ch).Offset))
	}
}
func (s *tStack) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for ch := s.LastChild(); ch != nil; ch = ch.ParentData().(*tStackData).PreviousSibling {
		off := rendering.BoxParentDataOf(ch).Offset
		if result.AddWithPaintOffset(&off, position, func(r *rendering.BoxHitTestResult, p ui.Offset) bool { return ch.HitTest(r, p) }) {
			return true
		}
	}
	return false
}

// callbackPainter is TestCallbackPainter of rendering_tester.dart.
type callbackPainter struct {
	rendering.CustomPainterBase
	onPaint func()
}

func newCallbackPainter(onPaint func()) *callbackPainter {
	return &callbackPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), onPaint: onPaint}
}

func (p *callbackPainter) Paint(*ui.Canvas, ui.Size)                  { p.onPaint() }
func (p *callbackPainter) ShouldRepaint(rendering.CustomPainter) bool { return true }

// debugPaintOps lays the box out in a fresh harness and records what paint and
// debugPaintSize draw, like debugPaint() of proxy_box_test.dart. It returns the
// names of the draw operations.
func debugPaintOps(t *testing.T, box rendering.Box) []string {
	t.Helper()
	h := newHarness(t)
	layout(t, h, box, layoutOpts{phase: phaseCompositingBits})
	root := rendering.NewContainerLayer()
	ctx := rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 800, 600))
	box.Paint(ctx, ui.OffsetZero)
	box.(interface {
		DebugPaintSize(*rendering.PaintingContext, ui.Offset)
	}).DebugPaintSize(ctx, ui.OffsetZero)
	ctx.StopRecordingIfNeeded()
	var names []string
	for _, l := range root.DepthFirstIterateChildren() {
		if pl, ok := l.(*rendering.PictureLayer); ok && pl.Picture() != nil {
			for _, op := range pl.Picture().DisplayList().Ops() {
				names = append(names, reflect.TypeOf(op).Name())
			}
		}
	}
	return names
}

func countOf(names []string, name string) int {
	n := 0
	for _, s := range names {
		if s == name {
			n++
		}
	}
	return n
}

var hashCodePattern = regexp.MustCompile(`#[0-9a-f]{5}`)

// ignoreHashCodes is equalsIgnoringHashCodes of flutter_test: it replaces the
// short hash codes in the text.
func ignoreHashCodes(s string) string { return hashCodePattern.ReplaceAllString(s, "#00000") }

// deepInfo is toStringDeep(minLevel: DiagnosticLevel.info).
func deepInfo(o foundation.DiagnosticableTree) string {
	return ignoreHashCodes(foundation.ToStringDeep(o, foundation.ToStringDeepOpts{MinLevel: foundation.DiagnosticLevelInfo}))
}

var alignmentTopLeft = painting.AlignmentTopLeft

// requireTag fails unless call panics with an error that has the tag.
func requireTag(t *testing.T, tag string, call func()) {
	t.Helper()
	v := panicOf(call)
	if v == nil {
		t.Fatalf("expected a panic with tag %s", tag)
	}
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(tag) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tag)
	}
}
