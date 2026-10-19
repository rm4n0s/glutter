package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// requireRoute fails the test unless err is an *errors.Error with the route.
func requireRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), route)
	}
}

// panicOf returns the value that call panics with, or nil.
func panicOf(call func()) (v any) {
	defer func() { v = recover() }()
	call()
	return nil
}

// tParentData is the parent data of tObject (TestRenderObjectParentData).
type tParentData struct {
	rendering.ParentDataBase
	rendering.ContainerParentData[rendering.RenderObject]
}

// tObject is TestRenderObject of object_test.dart: a render object with a list
// of children, a settable repaint boundary flag and a paint transform.
type tObject struct {
	rendering.RenderObjectBase
	rendering.ContainerRenderObject[rendering.RenderObject]
	repaintBoundary  bool
	allowPaintBounds bool
	paintTransform   *ui.Matrix4
}

func newTObject(allowPaintBounds bool) *tObject {
	o := &tObject{allowPaintBounds: allowPaintBounds, paintTransform: ui.NewMatrix4Identity()}
	rendering.InitRenderObject(&o.RenderObjectBase, o)
	rendering.InitContainerRenderObject(&o.ContainerRenderObject, o)
	return o
}

func (o *tObject) IsRepaintBoundary() bool { return o.repaintBoundary }
func (o *tObject) PaintBounds() ui.Rect {
	if !o.allowPaintBounds {
		panic("PaintBounds must not be called")
	}
	return ui.Rect{}
}
func (o *tObject) SemanticBounds() ui.Rect { return ui.NewRectFromLTWH(0, 0, 10, 20) }
func (o *tObject) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
	o.RenderObjectBase.ApplyPaintTransform(child, transform)
	transform.MultiplyBy(o.paintTransform)
}
func (o *tObject) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*tParentData); !ok {
		child.AsRenderObject().SetParentData(&tParentData{})
	}
}
func (o *tObject) PerformLayout() {}
func (o *tObject) PerformResize() {}

// tBox is a leaf render box that wants a size.
type tBox struct {
	rendering.RenderBox
	pref          ui.Size
	color         ui.Color
	hasColor      bool
	sizedByParent bool
	hitSelf       bool
	layouts       int
	paints        int
	boundary      bool
}

func newTBox(w, h float64) *tBox {
	b := &tBox{pref: ui.Size{Width: w, Height: h}}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *tBox) SizedByParent() bool     { return b.sizedByParent }
func (b *tBox) IsRepaintBoundary() bool { return b.boundary }
func (b *tBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(b.pref)
}
func (b *tBox) PerformLayout() {
	b.layouts++
	if !b.sizedByParent {
		b.SetSize(b.BoxConstraints().Constrain(b.pref))
	}
}
func (b *tBox) ComputeMinIntrinsicWidth(float64) float64  { return b.pref.Width }
func (b *tBox) ComputeMaxIntrinsicWidth(float64) float64  { return b.pref.Width }
func (b *tBox) ComputeMinIntrinsicHeight(float64) float64 { return b.pref.Height }
func (b *tBox) ComputeMaxIntrinsicHeight(float64) float64 { return b.pref.Height }
func (b *tBox) HitTestSelf(ui.Offset) bool                { return b.hitSelf }
func (b *tBox) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	b.paints++
	if b.hasColor {
		p := ui.NewPaint()
		p.SetColor(b.color)
		ctx.Canvas().DrawRect(offset.And(b.Size()), p)
	}
}

// tProxy is a render box with one child, as big as the child (RenderProxyBox).
type tProxy struct {
	rendering.RenderBox
	rendering.RenderObjectWithChild[rendering.Box]
	boundary bool
	layouts  int
}

func newTProxy(child rendering.Box) *tProxy {
	p := &tProxy{}
	rendering.InitRenderBox(&p.RenderBox, p)
	rendering.InitRenderObjectWithChild(&p.RenderObjectWithChild, p)
	if child != nil {
		p.SetChild(child)
	}
	return p
}

func (p *tProxy) IsRepaintBoundary() bool { return p.boundary }
func (p *tProxy) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	if ch := p.Child(); ch != nil {
		return ch.GetDryLayout(c)
	}
	return c.Smallest()
}
func (p *tProxy) PerformLayout() {
	p.layouts++
	if ch := p.Child(); ch != nil {
		ch.Layout(p.BoxConstraints(), rendering.LayoutOpts{ParentUsesSize: true})
		p.SetSize(ch.Size())
	} else {
		p.SetSize(p.BoxConstraints().Smallest())
	}
}
func (p *tProxy) ComputeMinIntrinsicWidth(h float64) float64 {
	if ch := p.Child(); ch != nil {
		return ch.GetMinIntrinsicWidth(h)
	}
	return 0
}
func (p *tProxy) ComputeMaxIntrinsicWidth(h float64) float64 {
	if ch := p.Child(); ch != nil {
		return ch.GetMaxIntrinsicWidth(h)
	}
	return 0
}
func (p *tProxy) ComputeMinIntrinsicHeight(w float64) float64 {
	if ch := p.Child(); ch != nil {
		return ch.GetMinIntrinsicHeight(w)
	}
	return 0
}
func (p *tProxy) ComputeMaxIntrinsicHeight(w float64) float64 {
	if ch := p.Child(); ch != nil {
		return ch.GetMaxIntrinsicHeight(w)
	}
	return 0
}
func (p *tProxy) ComputeDistanceToActualBaseline(b ui.TextBaseline) foundation.Opt[float64] {
	if ch := p.Child(); ch != nil {
		return ch.GetDistanceToActualBaseline(b)
	}
	return p.RenderBox.ComputeDistanceToActualBaseline(b)
}
func (p *tProxy) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	if ch := p.Child(); ch != nil {
		return ch.HitTest(result, position)
	}
	return false
}
func (p *tProxy) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	if ch := p.Child(); ch != nil {
		ctx.PaintChild(ch, offset)
	}
}

// tPadding is a render box that pads its child (RenderPadding).
type tPadding struct {
	rendering.RenderBox
	rendering.RenderObjectWithChild[rendering.Box]
	pad float64
}

func newTPadding(pad float64, child rendering.Box) *tPadding {
	p := &tPadding{pad: pad}
	rendering.InitRenderBox(&p.RenderBox, p)
	rendering.InitRenderObjectWithChild(&p.RenderObjectWithChild, p)
	if child != nil {
		p.SetChild(child)
	}
	return p
}

func (p *tPadding) inset() ui.Size { return ui.Size{Width: 2 * p.pad, Height: 2 * p.pad} }
func (p *tPadding) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	d := ui.Size{Width: 2 * p.pad, Height: 2 * p.pad}
	if ch := p.Child(); ch != nil {
		inner := c.Deflate(edgeAll(p.pad))
		return c.Constrain(ch.GetDryLayout(inner).Add(ui.Offset{Dx: d.Width, Dy: d.Height}))
	}
	return c.Constrain(d)
}
func (p *tPadding) PerformLayout() {
	c := p.BoxConstraints()
	ch := p.Child()
	if ch == nil {
		p.SetSize(c.Constrain(p.inset()))
		return
	}
	ch.Layout(c.Deflate(edgeAll(p.pad)), rendering.LayoutOpts{ParentUsesSize: true})
	ch.ParentData().(*rendering.BoxParentData).Offset = ui.Offset{Dx: p.pad, Dy: p.pad}
	p.SetSize(c.Constrain(ch.Size().Add(ui.Offset{Dx: 2 * p.pad, Dy: 2 * p.pad})))
}
func (p *tPadding) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	if ch := p.Child(); ch != nil {
		ctx.PaintChild(ch, offset.Add(ch.ParentData().(*rendering.BoxParentData).Offset))
	}
}
func (p *tPadding) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	ch := p.Child()
	if ch == nil {
		return false
	}
	off := ch.ParentData().(*rendering.BoxParentData).Offset
	return result.AddWithPaintOffset(&off, position, func(r *rendering.BoxHitTestResult, t ui.Offset) bool {
		return ch.HitTest(r, t)
	})
}

// tRowData is the parent data of the children of tRow.
type tRowData struct {
	rendering.ContainerBoxParentData[rendering.Box]
}

// tRow lays its children out in a row (a bit of RenderFlex).
type tRow struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
}

func newTRow(children ...rendering.Box) *tRow {
	r := &tRow{}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

func (r *tRow) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*tRowData); !ok {
		child.AsRenderObject().SetParentData(&tRowData{})
	}
}
func (r *tRow) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	var w, h float64
	for ch := r.FirstChild(); ch != nil; ch = r.ChildAfter(ch) {
		s := ch.GetDryLayout(c.Loosen())
		w += s.Width
		h = math.Max(h, s.Height)
	}
	return c.Constrain(ui.Size{Width: w, Height: h})
}
func (r *tRow) PerformLayout() {
	c := r.BoxConstraints()
	var x, h float64
	for ch := r.FirstChild(); ch != nil; ch = r.ChildAfter(ch) {
		ch.Layout(c.Loosen(), rendering.LayoutOpts{ParentUsesSize: true})
		ch.ParentData().(*tRowData).Offset = ui.Offset{Dx: x}
		x += ch.Size().Width
		h = math.Max(h, ch.Size().Height)
	}
	r.SetSize(c.Constrain(ui.Size{Width: x, Height: h}))
}
func (r *tRow) ComputeDistanceToActualBaseline(b ui.TextBaseline) foundation.Opt[float64] {
	return rendering.DefaultComputeDistanceToFirstActualBaseline(&r.ContainerRenderObject, b)
}
func (r *tRow) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return rendering.DefaultHitTestChildren(&r.ContainerRenderObject, result, position)
}
func (r *tRow) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	rendering.DefaultPaint(&r.ContainerRenderObject, ctx, offset)
}

// tLayerBox is _TestCustomLayerBox of object_test.dart: it paints through a
// function that pushes a layer, and keeps the layers it was given.
type tLayerBox struct {
	rendering.RenderBox
	painter func(painter rendering.PaintingContextCallback, ctx *rendering.PaintingContext, offset ui.Offset, old rendering.ContainerLayerLike) rendering.ContainerLayerLike
	painted []rendering.ContainerLayerLike
}

func newTLayerBox(f func(rendering.PaintingContextCallback, *rendering.PaintingContext, ui.Offset, rendering.ContainerLayerLike) rendering.ContainerLayerLike) *tLayerBox {
	b := &tLayerBox{painter: f}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *tLayerBox) PerformLayout() { b.SetSize(b.BoxConstraints().Smallest()) }
func (b *tLayerBox) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	layer := b.painter(func(*rendering.PaintingContext, ui.Offset) {}, ctx, offset, b.Layer())
	b.painted = append(b.painted, layer)
	b.SetLayer(layer)
}

// newHarness is a harness with a small view.
func newHarness(t *testing.T) *renderingtest.Harness {
	t.Helper()
	return renderingtest.New(t, renderingtest.Opts{Size: ui.Size{Width: 800, Height: 600}})
}

// tightRoot lays out the box in a view of 800x600 and pumps a frame.
func layoutBox(t *testing.T, h *renderingtest.Harness, box rendering.Box) {
	t.Helper()
	h.SetRoot(box)
	h.Pump()
}

func edgeAll(v float64) painting.EdgeInsets { return painting.NewEdgeInsetsAll(v) }

// tLoosen is a bit of RenderPositionedBox: it gives its child loose
// constraints, puts it at the origin and is as big as it can be.
type tLoosen struct {
	rendering.RenderBox
	rendering.RenderObjectWithChild[rendering.Box]
}

func newTLoosen(child rendering.Box) *tLoosen {
	p := &tLoosen{}
	rendering.InitRenderBox(&p.RenderBox, p)
	rendering.InitRenderObjectWithChild(&p.RenderObjectWithChild, p)
	p.SetChild(child)
	return p
}

func (p *tLoosen) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }
func (p *tLoosen) PerformLayout() {
	c := p.BoxConstraints()
	p.Child().Layout(c.Loosen(), rendering.LayoutOpts{ParentUsesSize: true})
	p.SetSize(c.Biggest())
}
func (p *tLoosen) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	ctx.PaintChild(p.Child(), offset.Add(p.Child().ParentData().(*rendering.BoxParentData).Offset))
}
func (p *tLoosen) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return p.Child().HitTest(result, position)
}

// skipUnlessDebug skips tests that read state the framework keeps only in debug
// builds.
func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only state of the framework")
	}
}
