package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

type dummyTarget struct{}

func (dummyTarget) HandleEvent(gestures.PointerEvent, gestures.HitTestEntry) {}

func TestBoxHitTestResultWrap(t *testing.T) {
	e1, e2, e3 := gestures.NewHitTestEntry(dummyTarget{}), gestures.NewHitTestEntry(dummyTarget{}), gestures.NewHitTestEntry(dummyTarget{})
	transform := ui.NewMatrix4TranslationValues(40, 150, 0)
	wrapped := gestures.NewHitTestResult()
	wrapped.PushTransform(transform)
	wrapped.Add(e1)
	if len(wrapped.Path()) != 1 || !e1.Transform().Equals(transform) {
		t.Fatalf("path = %v, transform = %v", wrapped.Path(), e1.Transform())
	}
	wrapping := rendering.NewBoxHitTestResultWrap(wrapped)
	if len(wrapping.Path()) != 1 {
		t.Fatalf("wrapping path = %v", wrapping.Path())
	}
	wrapping.Add(e2)
	if len(wrapped.Path()) != 2 || !e2.Transform().Equals(transform) {
		t.Errorf("after add through the wrapper: path = %v, transform = %v", wrapped.Path(), e2.Transform())
	}
	wrapped.Add(e3)
	if len(wrapping.Path()) != 3 || !e3.Transform().Equals(transform) {
		t.Errorf("after add through the original: path = %v, transform = %v", wrapping.Path(), e3.Transform())
	}
}

func hitRecorder(positions *[]ui.Offset, hit bool) rendering.BoxHitTest {
	return func(r *rendering.BoxHitTestResult, p ui.Offset) bool {
		if r == nil {
			panic("nil result")
		}
		*positions = append(*positions, p)
		return hit
	}
}

func TestBoxHitTestResultAddWith(t *testing.T) {
	pos := ui.Offset{Dx: 3, Dy: 4}
	tr := ui.NewMatrix4TranslationValues(20, 30, 0)
	off := func(x, y float64) *ui.Offset { return &ui.Offset{Dx: x, Dy: y} }
	tests := []struct {
		name    string
		call    func(r *rendering.BoxHitTestResult, hit rendering.BoxHitTest) bool
		hit     bool
		want    []ui.Offset
		wantHit bool
	}{
		{"paintTransform nil", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintTransform(nil, ui.OffsetZero, h)
		}, true, []ui.Offset{{}}, true},
		{"paintTransform translate", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintTransform(tr, ui.OffsetZero, h)
		}, true, []ui.Offset{{Dx: -20, Dy: -30}}, true},
		{"paintTransform nil miss", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintTransform(nil, pos, h)
		}, false, []ui.Offset{pos}, false},
		{"paintTransform identity", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintTransform(ui.NewMatrix4Identity(), pos, h)
		}, true, []ui.Offset{pos}, true},
		{"paintTransform offset", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintTransform(tr, pos, h)
		}, true, []ui.Offset{{Dx: -17, Dy: -26}}, true},
		{"paintTransform singular", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintTransform(ui.NewMatrix4ForceToPoint(pos), pos, h)
		}, true, nil, false},
		{"paintOffset nil", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintOffset(nil, ui.OffsetZero, h)
		}, true, []ui.Offset{{}}, true},
		{"paintOffset", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintOffset(off(55, 32), ui.OffsetZero, h)
		}, true, []ui.Offset{{Dx: -55, Dy: -32}}, true},
		{"paintOffset nil miss", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintOffset(nil, pos, h)
		}, false, []ui.Offset{pos}, false},
		{"paintOffset zero", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintOffset(off(0, 0), pos, h)
		}, true, []ui.Offset{pos}, true},
		{"paintOffset position", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithPaintOffset(off(20, 30), pos, h)
		}, true, []ui.Offset{{Dx: -17, Dy: -26}}, true},
		{"rawTransform nil", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithRawTransform(nil, ui.OffsetZero, h)
		}, true, []ui.Offset{{}}, true},
		{"rawTransform translate", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithRawTransform(tr, ui.OffsetZero, h)
		}, true, []ui.Offset{{Dx: 20, Dy: 30}}, true},
		{"rawTransform nil miss", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithRawTransform(nil, pos, h)
		}, false, []ui.Offset{pos}, false},
		{"rawTransform identity", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithRawTransform(ui.NewMatrix4Identity(), pos, h)
		}, true, []ui.Offset{pos}, true},
		{"rawTransform offset", func(r *rendering.BoxHitTestResult, h rendering.BoxHitTest) bool {
			return r.AddWithRawTransform(tr, pos, h)
		}, true, []ui.Offset{{Dx: 23, Dy: 34}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := rendering.NewBoxHitTestResult()
			var got []ui.Offset
			hit := tc.call(r, hitRecorder(&got, tc.hit))
			if hit != tc.wantHit {
				t.Errorf("hit = %v, want %v", hit, tc.wantHit)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("positions = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("position %d = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestBoxHitTestResultOutOfBand(t *testing.T) {
	ran := false
	hit := func(*rendering.BoxHitTestResult) bool { ran = true; return true }
	tests := []struct {
		name string
		o    rendering.OutOfBandOpts
	}{
		{"paintOffset", rendering.OutOfBandOpts{PaintOffset: &ui.Offset{Dx: 20, Dy: 30}}},
		{"paintTransform", rendering.OutOfBandOpts{PaintTransform: ui.NewMatrix4TranslationValues(20, 30, 0)}},
		{"rawTransform", rendering.OutOfBandOpts{RawTransform: ui.NewMatrix4TranslationValues(20, 30, 0)}},
		{"singular rawTransform", rendering.OutOfBandOpts{RawTransform: ui.NewMatrix4ForceToPoint(ui.OffsetZero)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ran = false
			if !rendering.NewBoxHitTestResult().AddWithOutOfBandPosition(tc.o, hit) || !ran {
				t.Error("the hit test did not run")
			}
		})
	}
}

func TestBoxBaselineOffset(t *testing.T) {
	b := func(v float64) rendering.BaselineOffset { return rendering.NewBaselineOffset(foundation.Some(v)) }
	tests := []struct {
		name string
		got  rendering.BaselineOffset
		want rendering.BaselineOffset
	}{
		{"min none none", rendering.NoBaseline.MinOf(rendering.NoBaseline), rendering.NoBaseline},
		{"min none 1", rendering.NoBaseline.MinOf(b(1)), b(1)},
		{"min 1 none", b(1).MinOf(rendering.NoBaseline), b(1)},
		{"min 2 1", b(2).MinOf(b(1)), b(1)},
		{"min 1 2", b(1).MinOf(b(2)), b(1)},
		{"add none", rendering.NoBaseline.Add(2), rendering.NoBaseline},
		{"add", b(1).Add(2), b(3)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestRenderBoxLayoutAndSize(t *testing.T) {
	h := newHarness(t)
	box := newTBox(100, 50)
	layoutBox(t, h, box)
	// The view is 800x600 and tight: the root is forced to that size.
	if got := box.Size(); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("Size() = %v, want 800x600", got)
	}
	if !box.HasSize() {
		t.Error("HasSize() is false after layout")
	}
	if got := box.BoxConstraints(); got != rendering.NewBoxConstraintsTight(ui.Size{Width: 800, Height: 600}) {
		t.Errorf("BoxConstraints() = %v", got)
	}
	if got, want := box.PaintBounds(), ui.NewRectFromLTWH(0, 0, 800, 600); got != want {
		t.Errorf("PaintBounds() = %v, want %v", got, want)
	}
	if got := box.SemanticBounds(); got != box.PaintBounds() {
		t.Errorf("SemanticBounds() = %v", got)
	}
}

func TestRenderBoxFlexAndPadding(t *testing.T) {
	h := newHarness(t)
	a, b, c := newTBox(100, 100), newTBox(50, 80), newTBox(30, 20)
	row := newTRow(a, b, c)
	pad := newTPadding(10, row)
	h.SetRoot(newTLoosen(pad))
	h.Pump()
	tests := []struct {
		name string
		box  rendering.Box
		want ui.Size
	}{
		{"a", a, ui.Size{Width: 100, Height: 100}},
		{"b", b, ui.Size{Width: 50, Height: 80}},
		{"row", row, ui.Size{Width: 180, Height: 100}},
		{"pad", pad, ui.Size{Width: 200, Height: 120}},
	}
	for _, tc := range tests {
		if got := tc.box.Size(); got != tc.want {
			t.Errorf("%s size = %v, want %v", tc.name, got, tc.want)
		}
	}
	if off := b.ParentData().(*tRowData).Offset; off != (ui.Offset{Dx: 100}) {
		t.Errorf("b offset = %v, want (100, 0)", off)
	}
	if got := c.LocalToGlobal(ui.OffsetZero, nil); got != (ui.Offset{Dx: 10 + 150, Dy: 10}) {
		t.Errorf("c.LocalToGlobal = %v, want (160, 10)", got)
	}
	if got := c.GlobalToLocal(ui.Offset{Dx: 165, Dy: 15}, nil); got != (ui.Offset{Dx: 5, Dy: 5}) {
		t.Errorf("c.GlobalToLocal = %v, want (5, 5)", got)
	}
	if got := c.LocalToGlobal(ui.OffsetZero, row); got != (ui.Offset{Dx: 150}) {
		t.Errorf("c.LocalToGlobal(ancestor row) = %v, want (150, 0)", got)
	}
	// The dry layout of the row agrees with its layout.
	if got := row.GetDryLayout(rendering.NewBoxConstraintsLoose(ui.Size{Width: 780, Height: 580})); got != (ui.Size{Width: 180, Height: 100}) {
		t.Errorf("GetDryLayout = %v", got)
	}
}

func TestRenderBoxHitTesting(t *testing.T) {
	h := newHarness(t)
	a, b := newTBox(100, 100), newTBox(50, 80)
	a.hitSelf, b.hitSelf = true, true
	row := newTRow(a, b)
	pad := newTPadding(10, row)
	h.SetRoot(newTLoosen(pad))
	h.Pump()
	tests := []struct {
		name string
		pos  ui.Offset
		want []rendering.Box // front first
		pts  []ui.Offset
	}{
		{"on b", ui.Offset{Dx: 125, Dy: 30}, []rendering.Box{b, row, pad}, []ui.Offset{{Dx: 15, Dy: 20}}},
		{"on a", ui.Offset{Dx: 30, Dy: 30}, []rendering.Box{a, row, pad}, nil},
		{"in the padding", ui.Offset{Dx: 5, Dy: 5}, nil, nil},
		{"outside the view", ui.Offset{Dx: 900, Dy: 5}, nil, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := gestures.NewHitTestResult()
			h.Binding.HitTestInView(result, tc.pos, ui.ImplicitViewID)
			var got []rendering.Box
			for _, e := range result.Path() {
				if be, ok := e.(*rendering.BoxHitTestEntry); ok {
					got = append(got, be.Target().(rendering.Box))
				}
			}
			// The pad is hit because the proxy root covers the view; the row
			// and boxes only when the position is inside them.
			if tc.want == nil {
				if len(got) != 0 {
					t.Errorf("hit %v, want nothing", got)
				}
				return
			}
			if len(got) < len(tc.want) {
				t.Fatalf("hit %d boxes, want at least %d", len(got), len(tc.want))
			}
			for i, w := range tc.want {
				if got[i] != w {
					t.Errorf("box %d = %v, want %v", i, foundation.DescribeIdentity(got[i]), foundation.DescribeIdentity(w))
				}
			}
			for i, p := range tc.pts {
				if lp := result.Path()[i].(*rendering.BoxHitTestEntry).LocalPosition; lp != p {
					t.Errorf("local position %d = %v, want %v", i, lp, p)
				}
			}
			// The last entries are the render view and the gesture binding.
			path := result.Path()
			if _, ok := path[len(path)-2].Target().(*rendering.RenderView); !ok {
				t.Errorf("the entry before the binding is %T, want the RenderView", path[len(path)-2].Target())
			}
		})
	}
}

func TestRenderBoxIntrinsicsAreCached(t *testing.T) {
	calls := 0
	box := newCountingBox(&calls)
	for i := 0; i < 3; i++ {
		if got := box.GetMinIntrinsicWidth(10); got != 1 {
			t.Fatalf("GetMinIntrinsicWidth(10) = %v, want 1", got)
		}
	}
	if got := box.GetMinIntrinsicWidth(20); got != 2 {
		t.Errorf("GetMinIntrinsicWidth(20) = %v, want 2", got)
	}
	if got := box.GetMaxIntrinsicWidth(10); got != 3 {
		t.Errorf("GetMaxIntrinsicWidth(10) = %v, want 3", got)
	}
	if got := box.GetMinIntrinsicHeight(10); got != 4 {
		t.Errorf("GetMinIntrinsicHeight(10) = %v, want 4", got)
	}
	if got := box.GetMaxIntrinsicHeight(10); got != 5 {
		t.Errorf("GetMaxIntrinsicHeight(10) = %v, want 5", got)
	}
	if calls != 5 {
		t.Errorf("computed %d times, want 5", calls)
	}
	// The same values come from the cache, in any order.
	if box.GetMaxIntrinsicHeight(10) != 5 || box.GetMinIntrinsicWidth(20) != 2 || box.GetMinIntrinsicWidth(10) != 1 {
		t.Error("the cached values changed")
	}
	// Marking the box dirty clears the cache.
	box.MarkNeedsLayout()
	if got := box.GetMinIntrinsicWidth(10); got != 6 {
		t.Errorf("after MarkNeedsLayout GetMinIntrinsicWidth(10) = %v, want 6", got)
	}
}

type countingBox struct {
	rendering.RenderBox
	calls *int
}

func newCountingBox(calls *int) *countingBox {
	b := &countingBox{calls: calls}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}
func (b *countingBox) next() float64                             { *b.calls++; return float64(*b.calls) }
func (b *countingBox) ComputeMinIntrinsicWidth(float64) float64  { return b.next() }
func (b *countingBox) ComputeMaxIntrinsicWidth(float64) float64  { return b.next() }
func (b *countingBox) ComputeMinIntrinsicHeight(float64) float64 { return b.next() }
func (b *countingBox) ComputeMaxIntrinsicHeight(float64) float64 { return b.next() }

// baseBox is a box with a baseline at half its height; calls counts the real
// computations.
type baselineBox struct {
	rendering.RenderBox
	calls int
	dry   bool
}

func newBaselineBox(dry bool) *baselineBox {
	b := &baselineBox{dry: dry}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}
func (b *baselineBox) PerformLayout()                                      { b.SetSize(b.BoxConstraints().Biggest()) }
func (b *baselineBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }
func (b *baselineBox) ComputeDistanceToActualBaseline(ui.TextBaseline) foundation.Opt[float64] {
	b.calls++
	return foundation.Some(b.Size().Height / 2)
}
func (b *baselineBox) ComputeDryBaseline(c rendering.BoxConstraints, _ ui.TextBaseline) foundation.Opt[float64] {
	if !b.dry {
		return b.RenderBox.ComputeDryBaseline(c, ui.TextBaselineAlphabetic)
	}
	return foundation.Some(c.Biggest().Height / 2)
}

// baselineParent reads the baseline of its child in PerformLayout and offsets
// the child by it.
type baselineParent struct {
	rendering.RenderBox
	rendering.RenderObjectWithChild[rendering.Box]
}

func newBaselineParent(child rendering.Box) *baselineParent {
	p := &baselineParent{}
	rendering.InitRenderBox(&p.RenderBox, p)
	rendering.InitRenderObjectWithChild(&p.RenderObjectWithChild, p)
	p.SetChild(child)
	return p
}
func (p *baselineParent) PerformLayout() {
	c := p.BoxConstraints()
	p.Child().Layout(c, rendering.LayoutOpts{ParentUsesSize: true})
	b := p.Child().GetDistanceToBaseline(ui.TextBaselineAlphabetic, rendering.BaselineOpts{OnlyReal: true})
	p.Child().ParentData().(*rendering.BoxParentData).Offset = ui.Offset{Dy: -b.OrElse(0)}
	p.SetSize(c.Biggest())
}
func (p *baselineParent) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	ctx.PaintChild(p.Child(), offset.Add(p.Child().ParentData().(*rendering.BoxParentData).Offset))
}

func TestRenderBoxBaselinesAreCached(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	box := newBaselineBox(false)
	parent := newBaselineParent(box)
	h.SetRoot(parent)
	h.Pump()
	if off := box.ParentData().(*rendering.BoxParentData).Offset; off.Dy != -300 {
		t.Fatalf("offset = %v, want -300", off)
	}
	if box.calls != 1 {
		t.Fatalf("baseline computed %d times, want 1", box.calls)
	}
	// A layout with the same constraints reads the cache.
	parent.MarkNeedsLayout()
	h.Pump()
	if box.calls != 1 {
		t.Errorf("baseline computed %d times after a parent relayout, want 1", box.calls)
	}
	// Marking the child dirty clears its cache and relays out the parent too.
	box.MarkNeedsLayout()
	if !parent.DebugNeedsLayout() {
		t.Error("the parent does not need layout after the child with a cache was marked")
	}
	h.Pump()
	if box.calls != 2 {
		t.Errorf("baseline computed %d times after the child was marked, want 2", box.calls)
	}
}

func TestRenderBoxDryBaseline(t *testing.T) {
	box := newBaselineBox(true)
	c := rendering.NewBoxConstraintsTight(ui.Size{Width: 40, Height: 60})
	for i := 0; i < 2; i++ {
		if got := box.GetDryBaseline(c, ui.TextBaselineAlphabetic); got != foundation.Some(30.0) {
			t.Fatalf("GetDryBaseline = %v, want 30", got)
		}
	}
	if got := rendering.ChildLayoutHelper.GetDryBaseline(box, c, ui.TextBaselineAlphabetic); got != foundation.Some(30.0) {
		t.Errorf("ChildLayoutHelper.GetDryBaseline = %v", got)
	}
	if got := rendering.ChildLayoutHelper.DryLayoutChild(box, c); got != (ui.Size{Width: 40, Height: 60}) {
		t.Errorf("ChildLayoutHelper.DryLayoutChild = %v", got)
	}
}

func TestChildLayoutHelperLayoutChild(t *testing.T) {
	h := newHarness(t)
	child := newTBox(30, 40)
	parent := newTProxy(child)
	h.SetRoot(parent)
	h.Pump()
	c := rendering.NewBoxConstraintsLoose(ui.Size{Width: 100, Height: 100})
	if got := rendering.ChildLayoutHelper.LayoutChild(child, c); got != (ui.Size{Width: 30, Height: 40}) {
		t.Errorf("LayoutChild = %v", got)
	}
	if !child.BoxConstraints().IsSatisfiedBy(child.Size()) || child.BoxConstraints() != c {
		t.Errorf("the child was laid out with %v", child.BoxConstraints())
	}
}

func TestRenderBoxChangingSizeRelayoutsParent(t *testing.T) {
	// dynamic_intrinsics_test: a parent that reads the size of a child that
	// grows is laid out again.
	h := newHarness(t)
	inner := newFixedSize()
	parentSize := newParentSize(inner)
	root := newIntrinsicSize(parentSize)
	h.SetRoot(newTLoosen(root))
	h.Pump()
	if root.Size() != inner.Size() {
		t.Fatalf("root size %v != inner size %v", root.Size(), inner.Size())
	}
	inner.grow()
	h.Pump()
	if root.Size() != inner.Size() || inner.Size().Width != 200 {
		t.Errorf("after growing: root %v, inner %v", root.Size(), inner.Size())
	}
}

func TestRenderBoxParentReturnsCorrectIntrinsics(t *testing.T) {
	h := newHarness(t)
	inner := newFixedSize()
	parentSize := newParentSize(inner)
	root := newIntrinsicSize(parentSize)
	h.SetRoot(newTLoosen(root))
	h.Pump()
	check := func(dimension float64) {
		t.Helper()
		if parentSize.GetMinIntrinsicWidth(math.Inf(1)) != dimension || parentSize.GetMaxIntrinsicWidth(math.Inf(1)) != dimension ||
			parentSize.GetMinIntrinsicHeight(math.Inf(1)) != dimension || parentSize.GetMaxIntrinsicHeight(math.Inf(1)) != dimension {
			t.Errorf("intrinsics are not %v", dimension)
		}
	}
	check(100)
	inner.grow()
	h.Pump()
	check(200)
}

// fixedSize, parentSize and intrinsicSize are the render objects of
// dynamic_intrinsics_test.dart.
type fixedSize struct {
	rendering.RenderBox
	dimension float64
}

func newFixedSize() *fixedSize {
	f := &fixedSize{dimension: 100}
	rendering.InitRenderBox(&f.RenderBox, f)
	return f
}
func (f *fixedSize) grow()                                     { f.dimension *= 2; f.MarkNeedsLayout() }
func (f *fixedSize) ComputeMinIntrinsicWidth(float64) float64  { return f.dimension }
func (f *fixedSize) ComputeMaxIntrinsicWidth(float64) float64  { return f.dimension }
func (f *fixedSize) ComputeMinIntrinsicHeight(float64) float64 { return f.dimension }
func (f *fixedSize) ComputeMaxIntrinsicHeight(float64) float64 { return f.dimension }
func (f *fixedSize) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(ui.Size{Width: f.dimension, Height: f.dimension})
}
func (f *fixedSize) PerformLayout() { f.SetSize(ui.Size{Width: f.dimension, Height: f.dimension}) }

type parentSize struct {
	tProxy
}

func newParentSize(child rendering.Box) *parentSize {
	p := &parentSize{}
	rendering.InitRenderBox(&p.RenderBox, p)
	rendering.InitRenderObjectWithChild(&p.RenderObjectWithChild, p)
	p.SetChild(child)
	return p
}
func (p *parentSize) SizedByParent() bool                                 { return true }
func (p *parentSize) PerformResize()                                      { p.SetSize(p.BoxConstraints().Biggest()) }
func (p *parentSize) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }
func (p *parentSize) PerformLayout()                                      { p.Child().Layout(p.BoxConstraints()) }

type intrinsicSize struct {
	tProxy
}

func newIntrinsicSize(child rendering.Box) *intrinsicSize {
	p := &intrinsicSize{}
	rendering.InitRenderBox(&p.RenderBox, p)
	rendering.InitRenderObjectWithChild(&p.RenderObjectWithChild, p)
	p.SetChild(child)
	return p
}
func (p *intrinsicSize) PerformLayout() {
	p.Child().Layout(p.BoxConstraints())
	p.SetSize(p.BoxConstraints().Constrain(ui.Size{
		Width:  p.Child().GetMinIntrinsicWidth(math.Inf(1)),
		Height: p.Child().GetMinIntrinsicHeight(math.Inf(1)),
	}))
}
func (p *intrinsicSize) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(ui.Size{
		Width:  p.Child().GetMinIntrinsicWidth(math.Inf(1)),
		Height: p.Child().GetMinIntrinsicHeight(math.Inf(1)),
	})
}
