package viewport3d

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/widgets"
)

// AnchoredOpts are the optional parameters of NewAnchored.
//
// Flutter: no counterpart
type AnchoredOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is the point of the child that is put on the anchor: Alignment{X: -1, Y: -1} is its top left corner, {X: 0, Y: 0}
	// (the default) its center and {X: 0, Y: 1} the middle of its bottom edge.
	Alignment painting.Alignment
	// Offset moves the child from the anchor, in logical pixels.
	Offset ui.Offset
	// HideWhenCovered hides the child while another item of the scene is between it and the camera, by the depth of the picture
	// (one pick, read back, in each frame that draws the scene, for each such child; none while there are no such children). The
	// answer comes with the frame after the one that drew.
	HideWhenCovered bool
}

// Anchored is a widget of a view that follows a point of the world (SPEC §27.8, "Overlays"): the label of a part, a dimension, a
// marker. Give it to ViewOpts.Children. At each layout the view puts the child at the point's place in the picture (Camera.Project),
// so the child moves with the camera, and it is hidden, with no painting, no hit testing and no semantics, while the point is
// behind the camera or the child is outside the view, and, with HideWhenCovered, while something is in front of the point. A child
// that is not an Anchored fills the view. The children are over the picture, in the order they are given, and the pointer reaches
// them before the navigation, so a label can be tapped and a tap elsewhere orbits.
//
// Flutter: no counterpart
type Anchored struct {
	widgets.SingleChildRenderObjectWidget
	point vecmath.Vec3
	o     AnchoredOpts
}

// NewAnchored creates a widget that follows a point of the world, in float64.
//
// Flutter: no counterpart
func NewAnchored(point vecmath.Vec3, child widgets.Widget, opts ...AnchoredOpts) *Anchored {
	o := firstOpts(opts)
	return &Anchored{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child, widgets.WidgetOpts{Key: o.Key}), point: point, o: o}
}

// Point is the point of the world that the child follows.
//
// Flutter: no counterpart
func (a *Anchored) Point() vecmath.Vec3 { return a.point }

// CreateRenderObject creates the render object.
//
// Flutter: no counterpart
func (a *Anchored) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderAnchor{}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	r.configure(a)
	return r
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: no counterpart
func (a *Anchored) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	ro.(*renderAnchor).configure(a)
}

// renderAnchor is the render object of an Anchored: a box with its child's size that the view's layer puts at a point.
type renderAnchor struct {
	rendering.RenderProxyBox
	point       vecmath.Vec3
	alignment   painting.Alignment
	offset      ui.Offset
	hideCovered bool
	// hidden is set by the layer: the point is behind the camera or the child is out of the view. covered is set by the answer of
	// the pick of the depth.
	hidden, covered bool
}

// configure takes the widget's configuration, and asks for a layout of the layer when it changed.
func (r *renderAnchor) configure(a *Anchored) {
	changed := r.point != a.point || r.alignment != a.o.Alignment || r.offset != a.o.Offset
	r.point, r.alignment, r.offset = a.point, a.o.Alignment, a.o.Offset
	if r.hideCovered != a.o.HideWhenCovered {
		r.hideCovered = a.o.HideWhenCovered
		r.covered = false
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
	if changed {
		if p := r.Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

// shown says whether the child is painted, hit tested and in the semantics tree.
func (r *renderAnchor) shown() bool { return !r.hidden && (!r.hideCovered || !r.covered) }

// setHidden is called by the layer.
func (r *renderAnchor) setHidden(hidden bool) {
	if r.hidden != hidden {
		r.hidden = hidden
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// setCovered is called with the answer of the pick of the depth.
func (r *renderAnchor) setCovered(covered bool) {
	if r.covered != covered && r.Attached() {
		r.covered = covered
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// HitTest is false while the child is hidden.
func (r *renderAnchor) HitTest(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return r.shown() && r.RenderProxyBox.HitTest(result, position)
}

// Paint paints nothing while the child is hidden.
func (r *renderAnchor) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if r.shown() {
		r.RenderProxyBox.Paint(context, offset)
	}
}

// PaintsChild is false while the child is hidden.
func (r *renderAnchor) PaintsChild(rendering.RenderObject) bool { return r.shown() }

// VisitChildrenForSemantics visits nothing while the child is hidden.
func (r *renderAnchor) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	if r.shown() {
		r.RenderProxyBox.VisitChildrenForSemantics(visitor)
	}
}

// anchorHost is what a view's state keeps of the layer of its anchored children: the layer, to ask it for the picks of the depth.
type anchorHost struct {
	layer *renderAnchored
}

// anchoredLayer is the widget of the view's children: the view first, then the anchored children over it.
type anchoredLayer struct {
	widgets.MultiChildRenderObjectWidget
	camera *Camera
	host   *anchorHost
}

func newAnchoredLayer(camera *Camera, host *anchorHost, children []widgets.Widget) *anchoredLayer {
	return &anchoredLayer{MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(children), camera: camera, host: host}
}

// CreateRenderObject creates the render object.
func (w *anchoredLayer) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderAnchored{camera: w.camera, host: w.host}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.clip = rendering.NewLayerHandle[*rendering.ClipRectLayer](nil)
	w.host.layer = r
	return r
}

// UpdateRenderObject follows a new camera.
func (w *anchoredLayer) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderAnchored)
	r.setCamera(w.camera)
	r.host = w.host
	w.host.layer = r
}

// DidUnmountRenderObject lets the host go of the layer.
func (w *anchoredLayer) DidUnmountRenderObject(ro rendering.RenderObject) {
	if w.host.layer == ro {
		w.host.layer = nil
	}
}

// renderAnchored lays out its children: the first fills it and the anchored ones are put at the projected points of their anchors.
type renderAnchored struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	camera *Camera
	host   *anchorHost
	handle foundation.ListenerHandle
	clip   *rendering.LayerHandle[*rendering.ClipRectLayer]
}

// SetupParentData gives the child a parent data with an offset.
func (r *renderAnchored) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*rendering.ContainerBoxParentData[rendering.Box]); !ok {
		child.AsRenderObject().SetParentData(&rendering.ContainerBoxParentData[rendering.Box]{})
	}
}

// setCamera follows another camera.
func (r *renderAnchored) setCamera(c *Camera) {
	if c == r.camera {
		return
	}
	if r.Attached() {
		r.camera.RemoveListener(r.handle)
		r.handle = c.AddListener(r.MarkNeedsLayout)
	}
	r.camera = c
	r.MarkNeedsLayout()
}

// Attach follows the camera: a move of it is a layout, not a build.
func (r *renderAnchored) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.handle = r.camera.AddListener(r.MarkNeedsLayout)
}

// Detach stops following the camera.
func (r *renderAnchored) Detach() {
	r.camera.RemoveListener(r.handle)
	r.handle = foundation.ListenerHandle{}
	r.RenderBox.Detach()
}

// Dispose lets the clip layer go.
func (r *renderAnchored) Dispose() {
	r.clip.SetLayer(nil)
	r.RenderBox.Dispose()
}

func (r *renderAnchored) ComputeMinIntrinsicWidth(float64) float64  { return 0 }
func (r *renderAnchored) ComputeMaxIntrinsicWidth(float64) float64  { return 0 }
func (r *renderAnchored) ComputeMinIntrinsicHeight(float64) float64 { return 0 }
func (r *renderAnchored) ComputeMaxIntrinsicHeight(float64) float64 { return 0 }

// ComputeDryLayout is the biggest size the constraints allow.
func (r *renderAnchored) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }

// PerformLayout takes the biggest size, lays the children that are not anchored out to fill it, and puts each anchored child at its
// point.
func (r *renderAnchored) PerformLayout() {
	constraints := r.BoxConstraints()
	size := constraints.Constrain(constraints.Biggest())
	if math.IsInf(size.Width, 0) || math.IsInf(size.Height, 0) {
		size = constraints.Smallest()
	}
	r.SetSize(size)
	if size.Width > 0 && size.Height > 0 && r.camera.ViewportSize() != size {
		r.camera.SetViewportSize(size)
	}
	tight := rendering.NewBoxConstraintsTight(size)
	loose := tight.Loosen()
	view := ui.Rect{Right: size.Width, Bottom: size.Height}
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		data := rendering.BoxParentDataOf(child)
		anchor, ok := child.(*renderAnchor)
		if !ok {
			child.Layout(tight, rendering.LayoutOpts{ParentUsesSize: false})
			data.Offset = ui.Offset{}
			continue
		}
		child.Layout(loose, rendering.LayoutOpts{ParentUsesSize: true})
		at, visible := r.camera.Project(anchor.point)
		if !visible {
			data.Offset = ui.Offset{Dx: -child.Size().Width - 1, Dy: -child.Size().Height - 1}
			anchor.setHidden(true)
			continue
		}
		data.Offset = at.Add(anchor.offset).Sub(anchor.alignment.AlongSize(child.Size()))
		anchor.setHidden(!view.Overlaps(data.Offset.And(child.Size())))
	}
}

// Paint paints the children in order, clipped to the box.
func (r *renderAnchored) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	r.clip.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()),
		func(c *rendering.PaintingContext, o ui.Offset) {
			rendering.DefaultPaint(&r.ContainerRenderObject, c, o)
		},
		rendering.PushClipRectOpts{ClipBehavior: ui.ClipHardEdge, OldLayer: r.clip.Layer()}))
}

// HitTestChildren hit tests the children from the front: the last one first; a hidden anchored child refuses.
func (r *renderAnchored) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return rendering.DefaultHitTestChildren(&r.ContainerRenderObject, result, position)
}

// DescribeApproximatePaintClip is the box.
func (r *renderAnchored) DescribeApproximatePaintClip(rendering.RenderObject) foundation.Opt[ui.Rect] {
	return foundation.Some(ui.OffsetZero.And(r.Size()))
}

// anchorProbe is an anchored child that wants the depth: where it is, and the function that takes the answer.
type anchorProbe struct {
	at    ui.Offset
	point vecmath.Vec3
	set   func(covered bool)
}

// probes returns the anchored children that hide when covered and are not hidden already, with where they are.
func (r *renderAnchored) probes() []anchorProbe {
	var out []anchorProbe
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		a, ok := child.(*renderAnchor)
		if !ok || !a.hideCovered || a.hidden {
			continue
		}
		at, visible := r.camera.Project(a.point)
		if !visible {
			continue
		}
		out = append(out, anchorProbe{at: at, point: a.point, set: a.setCovered})
	}
	return out
}

// askDepth asks the renderer, for the frame that is about to be drawn, whether something is in front of each anchor: a pick of one
// pixel's radius at the point, which is covered when the nearest surface there is nearer to the eye than the point.
func (r *renderAnchored) askDepth(pick func(at ui.Offset, done func(PickResult, error), opts ...PickOpts)) {
	for _, p := range r.probes() {
		p := p
		eye := r.camera.Eye()
		want := p.point.Sub(eye).Length()
		pick(p.at, func(res PickResult, err error) {
			if err != nil {
				return
			}
			p.set(res.Hit && res.Point.Sub(eye).Length() < want*(1-coverTolerance))
		}, PickOpts{Radius: 1})
	}
}

// coverTolerance is how much nearer, relative to its distance, the surface must be for the anchor to be covered: a point on a
// surface is not covered by it.
const coverTolerance = 2e-3
