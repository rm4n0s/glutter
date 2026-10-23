package widgets

import (
	"math"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// PanAxis is the axes along which an InteractiveViewer pans.
//
// Flutter: widgets/interactive_viewer.dart PanAxis
type PanAxis int

const (
	// PanAxisUnspecified is the zero value: the default of the widget (free).
	PanAxisUnspecified PanAxis = iota
	// PanAxisHorizontal allows panning only in the horizontal axis.
	//
	// Flutter: widgets/interactive_viewer.dart PanAxis.horizontal
	PanAxisHorizontal
	// PanAxisVertical allows panning only in the vertical axis.
	//
	// Flutter: widgets/interactive_viewer.dart PanAxis.vertical
	PanAxisVertical
	// PanAxisAligned allows panning in one axis at a time, the one the gesture
	// starts in.
	//
	// Flutter: widgets/interactive_viewer.dart PanAxis.aligned
	PanAxisAligned
	// PanAxisFree allows panning in any direction.
	//
	// Flutter: widgets/interactive_viewer.dart PanAxis.free
	PanAxisFree
)

// InteractiveViewerWidgetBuilder builds the child of an InteractiveViewer for the
// part of the scene that is visible: the viewport as a quad in scene coordinates.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewerWidgetBuilder
type InteractiveViewerWidgetBuilder func(bc BuildContext, viewport ui.Quad) Widget

// interactiveViewerDrag is the friction coefficient at the end of an interaction.
const interactiveViewerDrag = 0.0000135

// InteractiveViewerOpts are the optional parameters of NewInteractiveViewer and
// NewInteractiveViewerBuilder.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.new
type InteractiveViewerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ClipBehavior is how the child is clipped; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// PanAxis is the axes along which the child pans; the default is PanAxisFree.
	PanAxis PanAxis
	// BoundaryMargin is the margin around the child that the viewport can pan
	// into; the default is none.
	BoundaryMargin painting.EdgeInsets
	// Constrained is whether the child gets the constraints of the viewer (true)
	// or unbounded ones; the default is true. It is false for the builder.
	Constrained foundation.Opt[bool]
	// MaxScale is the maximum scale; the default is 2.5.
	MaxScale foundation.Opt[float64]
	// MinScale is the minimum scale; the default is 0.8.
	MinScale foundation.Opt[float64]
	// InteractionEndFrictionCoefficient is the drag of the inertia after an
	// interaction; the default is 0.0000135.
	InteractionEndFrictionCoefficient foundation.Opt[float64]
	// OnInteractionEnd is called when the user ends an interaction.
	OnInteractionEnd gestures.GestureScaleEndCallback
	// OnInteractionStart is called when the user starts an interaction.
	OnInteractionStart gestures.GestureScaleStartCallback
	// OnInteractionUpdate is called when the user updates an interaction.
	OnInteractionUpdate gestures.GestureScaleUpdateCallback
	// PanEnabled is whether panning is enabled; the default is true.
	PanEnabled foundation.Opt[bool]
	// ScaleEnabled is whether scaling is enabled; the default is true.
	ScaleEnabled foundation.Opt[bool]
	// ScaleFactor is how many pixels of mouse scroll make a scale of e; the
	// default is 200.
	ScaleFactor foundation.Opt[float64]
	// TransformationController is the controller of the transformation; the
	// default is one the viewer makes.
	TransformationController *TransformationController
	// Alignment is the alignment of the origin of the transformation.
	Alignment painting.AlignmentGeometry
	// TrackpadScrollCausesScale is whether a trackpad scroll scales (true) or pans
	// (false, the default).
	TrackpadScrollCausesScale bool
}

// InteractiveViewer is a widget that enables pan and zoom interactions with its
// child.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer
type InteractiveViewer struct {
	StatefulWidget
	child                     Widget
	builder                   InteractiveViewerWidgetBuilder
	constrained               bool
	clipBehavior              ui.Clip
	panAxis                   PanAxis
	boundaryMargin            painting.EdgeInsets
	maxScale, minScale        float64
	interactionEndFriction    float64
	onInteractionEnd          gestures.GestureScaleEndCallback
	onInteractionStart        gestures.GestureScaleStartCallback
	onInteractionUpdate       gestures.GestureScaleUpdateCallback
	panEnabled, scaleEnabled  bool
	scaleFactor               float64
	transformationController  *TransformationController
	alignment                 painting.AlignmentGeometry
	trackpadScrollCausesScale bool
}

func newInteractiveViewer(child Widget, builder InteractiveViewerWidgetBuilder, constrained bool, o InteractiveViewerOpts) *InteractiveViewer {
	minScale, maxScale := o.MinScale.OrElse(0.8), o.MaxScale.OrElse(2.5)
	friction := o.InteractionEndFrictionCoefficient.OrElse(interactiveViewerDrag)
	if foundation.DebugMode {
		finite := func(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }
		m := o.BoundaryMargin
		switch {
		case !(minScale > 0) || !finite(minScale):
			panic(errors.New(tagInteractiveViewerScaleInvalid, "the minScale must be greater than zero and finite", "minScale", minScale))
		case !(friction > 0):
			panic(errors.New(tagInteractiveViewerFrictionInvalid, "the interactionEndFrictionCoefficient must be greater than zero", "friction", friction))
		case !(maxScale > 0) || math.IsNaN(maxScale):
			panic(errors.New(tagInteractiveViewerScaleInvalid, "the maxScale must be greater than zero", "maxScale", maxScale))
		case maxScale < minScale:
			panic(errors.New(tagInteractiveViewerScaleInvalid, "the maxScale must not be less than the minScale", "minScale", minScale, "maxScale", maxScale))
		case !((math.IsInf(m.Left+m.Right, 0) && math.IsInf(m.Top+m.Bottom, 0)) ||
			(finite(m.Top) && finite(m.Right) && finite(m.Bottom) && finite(m.Left))):
			panic(errors.New(tagInteractiveViewerBoundaryMarginInvalid, "the boundaryMargin must be finite in all directions or infinite in both axes"))
		}
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.PanAxis == PanAxisUnspecified {
		o.PanAxis = PanAxisFree
	}
	return &InteractiveViewer{
		StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, builder: builder, constrained: constrained,
		clipBehavior: o.ClipBehavior, panAxis: o.PanAxis, boundaryMargin: o.BoundaryMargin, maxScale: maxScale, minScale: minScale,
		interactionEndFriction: friction, onInteractionEnd: o.OnInteractionEnd, onInteractionStart: o.OnInteractionStart,
		onInteractionUpdate: o.OnInteractionUpdate, panEnabled: o.PanEnabled.OrElse(true), scaleEnabled: o.ScaleEnabled.OrElse(true),
		scaleFactor: o.ScaleFactor.OrElse(gestures.DefaultMouseScrollToScaleFactor), transformationController: o.TransformationController,
		alignment: o.Alignment, trackpadScrollCausesScale: o.TrackpadScrollCausesScale,
	}
}

// NewInteractiveViewer creates a viewer of the child that pans and zooms.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.new
func NewInteractiveViewer(child Widget, opts ...InteractiveViewerOpts) *InteractiveViewer {
	o := firstOpts(opts)
	return newInteractiveViewer(child, nil, o.Constrained.OrElse(true), o)
}

// NewInteractiveViewerBuilder creates a viewer whose child the builder builds for
// the visible part of the scene.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.builder
func NewInteractiveViewerBuilder(builder InteractiveViewerWidgetBuilder, opts ...InteractiveViewerOpts) *InteractiveViewer {
	o := firstOpts(opts)
	if !o.ScaleFactor.IsSome() {
		o.ScaleFactor = foundation.Some(200.0)
	}
	return newInteractiveViewer(nil, builder, false, o)
}

// Child is the widget that is panned and zoomed, or nil for the builder.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.child
func (w *InteractiveViewer) Child() Widget { return w.child }

// MinScale is the minimum scale.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.minScale
func (w *InteractiveViewer) MinScale() float64 { return w.minScale }

// MaxScale is the maximum scale.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.maxScale
func (w *InteractiveViewer) MaxScale() float64 { return w.maxScale }

// TransformationController is the controller of the transformation, or nil.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.transformationController
func (w *InteractiveViewer) TransformationController() *TransformationController {
	return w.transformationController
}

// CreateState creates the state.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.createState
func (w *InteractiveViewer) CreateState() State {
	return &interactiveViewerState{childKey: NewGlobalKey[State](), parentKey: NewGlobalKey[State]()}
}

func v64(v ui.Vector3) (x, y, z float64) { return float64(v.X), float64(v.Y), float64(v.Z) }

func mkV3(x, y, z float64) ui.Vector3 { return ui.Vector3{X: float32(x), Y: float32(y), Z: float32(z)} }

// InteractiveViewerGetNearestPointOnLine is the point on the line segment between
// l1 and l2 that is nearest to the point.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.getNearestPointOnLine
func InteractiveViewerGetNearestPointOnLine(point, l1, l2 ui.Vector3) ui.Vector3 {
	px, py, _ := v64(point)
	ax, ay, az := v64(l1)
	bx, by, bz := v64(l2)
	lengthSquared := (bx-ax)*(bx-ax) + (by-ay)*(by-ay)
	// In this case, l1 == l2.
	if lengthSquared == 0 {
		return l1
	}
	// Calculate how far down the line segment the closest point is and return
	// the point.
	l1Px, l1Py := px-ax, py-ay
	l1L2x, l1L2y, l1L2z := bx-ax, by-ay, bz-az
	// The z component of point is not used: Dart's dot does use it, but the
	// points are on the plane z = 0.
	_, _, pz := v64(point)
	fraction := ui.ClampDouble((l1Px*l1L2x+l1Py*l1L2y+(pz-az)*l1L2z)/lengthSquared, 0, 1)
	return mkV3(ax+l1L2x*fraction, ay+l1L2y*fraction, az+l1L2z*fraction)
}

// InteractiveViewerGetAxisAlignedBoundingBox is the smallest axis aligned
// rectangle, as a quad, that contains the quad.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.getAxisAlignedBoundingBox
func InteractiveViewerGetAxisAlignedBoundingBox(quad ui.Quad) ui.Quad {
	minX := math.Min(float64(quad.Point0.X), math.Min(float64(quad.Point1.X), math.Min(float64(quad.Point2.X), float64(quad.Point3.X))))
	minY := math.Min(float64(quad.Point0.Y), math.Min(float64(quad.Point1.Y), math.Min(float64(quad.Point2.Y), float64(quad.Point3.Y))))
	maxX := math.Max(float64(quad.Point0.X), math.Max(float64(quad.Point1.X), math.Max(float64(quad.Point2.X), float64(quad.Point3.X))))
	maxY := math.Max(float64(quad.Point0.Y), math.Max(float64(quad.Point1.Y), math.Max(float64(quad.Point2.Y), float64(quad.Point3.Y))))
	return ui.NewQuadPoints(mkV3(minX, minY, 0), mkV3(maxX, minY, 0), mkV3(maxX, maxY, 0), mkV3(minX, maxY, 0))
}

// InteractiveViewerPointIsInside is whether the point is inside the quad, which
// must be a rectangle.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.pointIsInside
func InteractiveViewerPointIsInside(point ui.Vector3, quad ui.Quad) bool {
	px, py, pz := v64(point)
	ax, ay, az := v64(quad.Point0)
	bx, by, bz := v64(quad.Point1)
	dx, dy, dz := v64(quad.Point3)
	aMx, aMy, aMz := px-ax, py-ay, pz-az
	aBx, aBy, aBz := bx-ax, by-ay, bz-az
	aDx, aDy, aDz := dx-ax, dy-ay, dz-az
	aMAB := aMx*aBx + aMy*aBy + aMz*aBz
	aBAB := aBx*aBx + aBy*aBy + aBz*aBz
	aMAD := aMx*aDx + aMy*aDy + aMz*aDz
	aDAD := aDx*aDx + aDy*aDy + aDz*aDz
	return 0 <= aMAB && aMAB <= aBAB && 0 <= aMAD && aMAD <= aDAD
}

// InteractiveViewerGetNearestPointInside is the point inside the quad, which must
// be a rectangle, that is nearest to the point.
//
// Flutter: widgets/interactive_viewer.dart InteractiveViewer.getNearestPointInside
func InteractiveViewerGetNearestPointInside(point ui.Vector3, quad ui.Quad) ui.Vector3 {
	// If the point is inside the axis aligned bounding box, then it's ok where it is.
	if InteractiveViewerPointIsInside(point, quad) {
		return point
	}
	// Otherwise, return the nearest point on the quad.
	closestPoints := []ui.Vector3{
		InteractiveViewerGetNearestPointOnLine(point, quad.Point0, quad.Point1),
		InteractiveViewerGetNearestPointOnLine(point, quad.Point1, quad.Point2),
		InteractiveViewerGetNearestPointOnLine(point, quad.Point2, quad.Point3),
		InteractiveViewerGetNearestPointOnLine(point, quad.Point3, quad.Point0),
	}
	minDistance := math.Inf(1)
	var closestOverall ui.Vector3
	for _, closePoint := range closestPoints {
		distance := math.Sqrt(math.Pow(float64(point.X)-float64(closePoint.X), 2) + math.Pow(float64(point.Y)-float64(closePoint.Y), 2))
		if distance < minDistance {
			minDistance = distance
			closestOverall = closePoint
		}
	}
	return closestOverall
}

type interactiveGestureType int

const (
	gesturePan interactiveGestureType = iota + 1
	gestureScale
	gestureRotate
)

// interactiveViewerState is the state of an InteractiveViewer.
//
// Flutter: widgets/interactive_viewer.dart _InteractiveViewerState
type interactiveViewerState struct {
	StateOf[*InteractiveViewer]
	TickerProvider
	transformer          *TransformationController
	childKey, parentKey  *GlobalKey[State]
	animation            animation.Animation[ui.Offset]
	animationHandle      foundation.ListenerHandle
	scaleAnimation       animation.Animation[float64]
	scaleAnimationHandle foundation.ListenerHandle
	scaleAnimationFocal  ui.Offset
	controller           *animation.AnimationController
	scaleController      *animation.AnimationController
	currentAxis          foundation.Opt[painting.Axis]
	referenceFocalPoint  foundation.Opt[ui.Offset]
	scaleStart           foundation.Opt[float64]
	rotationStart        foundation.Opt[float64]
	currentRotation      float64
	gestureType          interactiveGestureType
	transformerListener  foundation.ListenerHandle
}

const interactiveRotateEnabled = false

func (s *interactiveViewerState) boundaryRect() ui.Rect {
	childRenderBox := s.childKey.CurrentContext().FindRenderObject().(rendering.Box)
	childSize := childRenderBox.Size()
	boundaryRect := s.Widget().boundaryMargin.InflateRect(ui.OffsetZero.And(childSize))
	if foundation.DebugMode {
		if boundaryRect.IsEmpty() {
			panic(errors.New(tagInteractiveViewerChildEmpty, "InteractiveViewer's child must have nonzero dimensions"))
		}
		inf := math.IsInf
		if !boundaryRect.IsFinite() && !(inf(boundaryRect.Left, 0) && inf(boundaryRect.Top, 0) && inf(boundaryRect.Right, 0) && inf(boundaryRect.Bottom, 0)) {
			panic(errors.New(tagInteractiveViewerBoundaryInvalid, "boundaryRect must either be infinite in all directions or finite in all directions"))
		}
	}
	return boundaryRect
}

func (s *interactiveViewerState) viewport() ui.Rect {
	parentRenderBox := s.parentKey.CurrentContext().FindRenderObject().(rendering.Box)
	return ui.OffsetZero.And(parentRenderBox.Size())
}

// matrixTranslate returns a copy of the matrix translated by the translation,
// kept within the boundaries.
func (s *interactiveViewerState) matrixTranslate(matrix *ui.Matrix4, translation ui.Offset) *ui.Matrix4 {
	if translation == (ui.Offset{}) {
		return matrix.Clone()
	}
	alignedTranslation := translation
	if s.currentAxis.IsSome() {
		switch s.Widget().panAxis {
		case PanAxisHorizontal:
			alignedTranslation = alignAxis(translation, painting.AxisHorizontal)
		case PanAxisVertical:
			alignedTranslation = alignAxis(translation, painting.AxisVertical)
		case PanAxisAligned:
			alignedTranslation = alignAxis(translation, s.currentAxis.Value())
		}
	}
	nextMatrix := matrix.Clone()
	nextMatrix.Translate(float32(alignedTranslation.Dx), float32(alignedTranslation.Dy), 0)
	// Transform the viewport to determine where its four corners will be after
	// the child has been transformed.
	nextViewport := transformViewport(nextMatrix, s.viewport())
	// If the boundaries are infinite, then no need to check if the translation
	// fits within them.
	boundaryRect := s.boundaryRect()
	if boundaryRect.IsInfinite() {
		return nextMatrix
	}
	// Expand the boundaries with rotation. This prevents the problem where a
	// mismatch in orientation between the viewport and boundaries effectively
	// limits translation. With this approach, all points that are visible with
	// no rotation are visible after rotation.
	boundariesAabbQuad := axisAlignedBoundingBoxWithRotation(boundaryRect, s.currentRotation)
	// If the given translation fits completely within the boundaries, allow it.
	offendingDistance := exceedsBy(boundariesAabbQuad, nextViewport)
	if offendingDistance == (ui.Offset{}) {
		return nextMatrix
	}
	// Desired translation goes out of bounds, so translate to the nearest
	// in-bounds point instead.
	nextTotalTranslation := matrixTranslation(nextMatrix)
	currentScale := matrix.GetMaxScaleOnAxis()
	correctedTotalTranslation := ui.Offset{
		Dx: nextTotalTranslation.Dx - offendingDistance.Dx*currentScale,
		Dy: nextTotalTranslation.Dy - offendingDistance.Dy*currentScale,
	}
	// TODO(justinmc): This needs to be understood better. The correct matrix
	// translation may actually be a combination of the original matrix.
	correctedMatrix := matrix.Clone()
	correctedMatrix.SetTranslationRaw(float32(correctedTotalTranslation.Dx), float32(correctedTotalTranslation.Dy), 0)
	// Double check that the corrected translation fits within the viewport
	// after translation and the translation can be applied.
	correctedViewport := transformViewport(correctedMatrix, s.viewport())
	offendingCorrectedDistance := exceedsBy(boundariesAabbQuad, correctedViewport)
	if offendingCorrectedDistance == (ui.Offset{}) {
		return correctedMatrix
	}
	// If the corrected translation doesn't fit in either direction, don't allow
	// any translation at all. This happens when the viewport is larger than the
	// entire boundary.
	if offendingCorrectedDistance.Dx != 0 && offendingCorrectedDistance.Dy != 0 {
		return matrix.Clone()
	}
	// Otherwise, allow translation in only the direction that fits. This
	// happens when the viewport is larger than the boundary in one direction.
	unidirectional := ui.Offset{}
	if offendingCorrectedDistance.Dx == 0 {
		unidirectional.Dx = correctedTotalTranslation.Dx
	}
	if offendingCorrectedDistance.Dy == 0 {
		unidirectional.Dy = correctedTotalTranslation.Dy
	}
	result := matrix.Clone()
	result.SetTranslationRaw(float32(unidirectional.Dx), float32(unidirectional.Dy), 0)
	return result
}

// matrixScale returns a copy of the matrix scaled by the scale, kept within the
// bounds of the scale.
func (s *interactiveViewerState) matrixScale(matrix *ui.Matrix4, scale float64) *ui.Matrix4 {
	if scale == 1 {
		return matrix.Clone()
	}
	if foundation.DebugMode && scale == 0 {
		panic(errors.New(tagInteractiveViewerScaleZero, "scale must not be zero"))
	}
	// Don't allow a zoom that makes the viewport smaller than the boundary
	// rectangle.
	currentScale := s.transformer.Value().GetMaxScaleOnAxis()
	boundaryRect := s.boundaryRect()
	viewport := s.viewport()
	totalScale := math.Max(currentScale*scale, math.Max(viewport.Width()/boundaryRect.Width(), viewport.Height()/boundaryRect.Height()))
	clampedTotalScale := ui.ClampDouble(totalScale, s.Widget().minScale, s.Widget().maxScale)
	clampedScale := clampedTotalScale / currentScale
	result := matrix.Clone()
	result.Scale(float32(clampedScale), float32(clampedScale), float32(clampedScale))
	return result
}

// matrixRotate returns a copy of the matrix rotated by the rotation around the
// focal point.
func (s *interactiveViewerState) matrixRotate(matrix *ui.Matrix4, rotation float64, focalPoint ui.Offset) *ui.Matrix4 {
	if rotation == 0 {
		return matrix.Clone()
	}
	focalPointScene := s.transformer.ToScene(focalPoint)
	result := matrix.Clone()
	result.Translate(float32(focalPointScene.Dx), float32(focalPointScene.Dy), 0)
	result.RotateZ(-rotation)
	result.Translate(float32(-focalPointScene.Dx), float32(-focalPointScene.Dy), 0)
	return result
}

func (s *interactiveViewerState) gestureIsSupported(gestureType interactiveGestureType) bool {
	switch gestureType {
	case gestureRotate:
		return interactiveRotateEnabled
	case gestureScale:
		return s.Widget().scaleEnabled
	}
	return s.Widget().panEnabled
}

// getGestureType is the gesture that the update is mostly doing: the one with the
// greatest change.
func (s *interactiveViewerState) getGestureType(details gestures.ScaleUpdateDetails) interactiveGestureType {
	scale := details.Scale
	if !s.Widget().scaleEnabled {
		scale = 1
	}
	rotation := details.Rotation
	if !interactiveRotateEnabled {
		rotation = 0
	}
	switch {
	case math.Abs(scale-1) > math.Abs(rotation):
		return gestureScale
	case rotation != 0:
		return gestureRotate
	}
	return gesturePan
}

func (s *interactiveViewerState) onScaleStart(details gestures.ScaleStartDetails) {
	w := s.Widget()
	if w.onInteractionStart != nil {
		w.onInteractionStart(details)
	}
	if s.controller.IsAnimating() {
		s.controller.Stop()
		s.controller.Reset()
		s.removeInertiaListener()
	}
	if s.scaleController.IsAnimating() {
		s.scaleController.Stop()
		s.scaleController.Reset()
		s.removeScaleListener()
	}
	s.gestureType = 0
	s.currentAxis = foundation.None[painting.Axis]()
	s.scaleStart = foundation.Some(s.transformer.Value().GetMaxScaleOnAxis())
	s.referenceFocalPoint = foundation.Some(s.transformer.ToScene(details.LocalFocalPoint))
	s.rotationStart = foundation.Some(s.currentRotation)
}

func (s *interactiveViewerState) removeInertiaListener() {
	if s.animation != nil {
		s.animation.RemoveListener(s.animationHandle)
		s.animation = nil
	}
}

func (s *interactiveViewerState) removeScaleListener() {
	if s.scaleAnimation != nil {
		s.scaleAnimation.RemoveListener(s.scaleAnimationHandle)
		s.scaleAnimation = nil
	}
}

func (s *interactiveViewerState) onScaleUpdate(details gestures.ScaleUpdateDetails) {
	w := s.Widget()
	update := func() {
		if w.onInteractionUpdate != nil {
			w.onInteractionUpdate(details)
		}
	}
	scale := s.transformer.Value().GetMaxScaleOnAxis()
	s.scaleAnimationFocal = details.LocalFocalPoint
	focalPointScene := s.transformer.ToScene(details.LocalFocalPoint)
	switch s.gestureType {
	case gesturePan:
		// When a gesture first starts, it sometimes has no change in scale and
		// rotation despite being a two-finger gesture. Here the gesture is
		// allowed to be reinterpreted as its correct type after originally
		// being marked as a pan.
		s.gestureType = s.getGestureType(details)
	case 0:
		s.gestureType = s.getGestureType(details)
	}
	if !s.gestureIsSupported(s.gestureType) {
		update()
		return
	}
	switch s.gestureType {
	case gestureScale:
		desiredScale := s.scaleStart.Value() * details.Scale
		scaleChange := desiredScale / scale
		s.transformer.SetValue(s.matrixScale(s.transformer.Value(), scaleChange))
		// While scaling, translate such that the user's two fingers stay on the
		// same places in the scene. That means that the focal point of the scale
		// should be on the same place in the scene before and after the scale.
		focalPointSceneScaled := s.transformer.ToScene(details.LocalFocalPoint)
		s.transformer.SetValue(s.matrixTranslate(s.transformer.Value(), focalPointSceneScaled.Sub(s.referenceFocalPoint.Value())))
		// details.localFocalPoint should now be at the same location as the
		// original _referenceFocalPoint point. If it's not, that's because the
		// translate came in contact with a boundary. In that case, update
		// _referenceFocalPoint so subsequent updates happen in relation to the new
		// effective focal point.
		focalPointSceneCheck := s.transformer.ToScene(details.LocalFocalPoint)
		if roundOffset(s.referenceFocalPoint.Value()) != roundOffset(focalPointSceneCheck) {
			s.referenceFocalPoint = foundation.Some(focalPointSceneCheck)
		}
	case gestureRotate:
		if details.Rotation == 0 {
			update()
			return
		}
		desiredRotation := s.rotationStart.Value() + details.Rotation
		s.transformer.SetValue(s.matrixRotate(s.transformer.Value(), s.currentRotation-desiredRotation, details.LocalFocalPoint))
		s.currentRotation = desiredRotation
	case gesturePan:
		if details.Scale != 1 {
			update()
			return
		}
		// Translate so that the same point in the scene is underneath the focal
		// point before and after the movement.
		if s.currentAxis.IsNone() {
			s.currentAxis = panAxisOf(s.referenceFocalPoint.Value(), focalPointScene)
		}
		translationChange := focalPointScene.Sub(s.referenceFocalPoint.Value())
		s.transformer.SetValue(s.matrixTranslate(s.transformer.Value(), translationChange))
		s.referenceFocalPoint = foundation.Some(s.transformer.ToScene(details.LocalFocalPoint))
	}
	update()
}

func (s *interactiveViewerState) onScaleEnd(details gestures.ScaleEndDetails) {
	w := s.Widget()
	if w.onInteractionEnd != nil {
		w.onInteractionEnd(details)
	}
	s.scaleStart = foundation.None[float64]()
	s.rotationStart = foundation.None[float64]()
	s.referenceFocalPoint = foundation.None[ui.Offset]()
	s.removeInertiaListener()
	s.removeScaleListener()
	s.controller.Reset()
	s.scaleController.Reset()
	// If the interaction did not take place on a controller (a gesture that was
	// not supported), the axis is cleared.
	if s.gestureType != 0 && !s.gestureIsSupported(s.gestureType) {
		s.currentAxis = foundation.None[painting.Axis]()
		return
	}
	if s.gestureType == 0 && !s.gestureIsSupported(gesturePan) {
		s.currentAxis = foundation.None[painting.Axis]()
		return
	}
	switch s.gestureType {
	case gesturePan:
		if details.Velocity.PixelsPerSecond.Distance() < gestures.MinFlingVelocity {
			s.currentAxis = foundation.None[painting.Axis]()
			return
		}
		translationVector := s.transformer.Value().GetTranslation()
		translation := ui.Offset{Dx: float64(translationVector.X), Dy: float64(translationVector.Y)}
		frictionSimulationX := physics.NewFrictionSimulation(w.interactionEndFriction, translation.Dx, details.Velocity.PixelsPerSecond.Dx)
		frictionSimulationY := physics.NewFrictionSimulation(w.interactionEndFriction, translation.Dy, details.Velocity.PixelsPerSecond.Dy)
		tFinal := interactiveFinalTime(details.Velocity.PixelsPerSecond.Distance(), w.interactionEndFriction, 10)
		end := ui.Offset{Dx: frictionSimulationX.FinalX(), Dy: frictionSimulationY.FinalX()}
		s.animation = animation.NewTweenFunc(translation, end, func(a, b ui.Offset, t float64) ui.Offset { return a.Lerp(b, t) }).
			Chain(animation.NewCurveTween(animation.Curves.Decelerate)).Animate(s.controller)
		s.controller.Duration = foundation.Some(time.Duration(math.Round(tFinal*1000)) * time.Millisecond)
		s.animationHandle = s.animation.AddListener(s.handleInertiaAnimation)
		s.controller.Forward()
	case gestureScale:
		if math.Abs(details.ScaleVelocity) < 0.1 {
			s.currentAxis = foundation.None[painting.Axis]()
			return
		}
		scale := s.transformer.Value().GetMaxScaleOnAxis()
		frictionSimulation := physics.NewFrictionSimulation(w.interactionEndFriction*w.scaleFactor, scale, details.ScaleVelocity/10)
		tFinal := interactiveFinalTime(math.Abs(details.ScaleVelocity), w.interactionEndFriction, 0.1)
		s.scaleAnimation = animation.NewTween(scale, frictionSimulation.X(tFinal)).Chain(animation.NewCurveTween(animation.Curves.Decelerate)).Animate(s.scaleController)
		s.scaleController.Duration = foundation.Some(time.Duration(math.Round(tFinal*1000)) * time.Millisecond)
		s.scaleAnimationHandle = s.scaleAnimation.AddListener(s.handleScaleAnimation)
		s.scaleController.Forward()
	}
}

// receivedPointerSignal handles a mouse wheel scroll or a pinch of a trackpad.
func (s *interactiveViewerState) receivedPointerSignal(event gestures.PointerSignalEvent) {
	w := s.Widget()
	local, global := event.LocalPosition(), event.Position()
	var scaleChange float64
	switch e := event.(type) {
	case *gestures.PointerScrollEvent:
		scrollDelta := e.ScrollDelta()
		if e.Kind() == ui.PointerDeviceKindTrackpad && !w.trackpadScrollCausesScale {
			// Trackpad scroll, treated as a pan.
			if w.onInteractionStart != nil {
				w.onInteractionStart(gestures.NewScaleStartDetails(gestures.ScaleStartOpts{FocalPoint: global, LocalFocalPoint: foundation.Some(local)}))
			}
			localDelta := gestures.TransformDeltaViaPositions(global.Add(scrollDelta), scrollDelta, e.Transform())
			if !s.gestureIsSupported(gesturePan) {
				if w.onInteractionUpdate != nil {
					w.onInteractionUpdate(gestures.NewScaleUpdateDetails(gestures.ScaleUpdateOpts{
						FocalPoint: global.Sub(scrollDelta), LocalFocalPoint: foundation.Some(local.Sub(scrollDelta)), FocalPointDelta: localDelta.Neg()}))
				}
				if w.onInteractionEnd != nil {
					w.onInteractionEnd(gestures.ScaleEndDetails{})
				}
				return
			}
			focalPointScene := s.transformer.ToScene(local)
			newFocalPointScene := s.transformer.ToScene(local.Sub(localDelta))
			s.transformer.SetValue(s.matrixTranslate(s.transformer.Value(), newFocalPointScene.Sub(focalPointScene)))
			if w.onInteractionUpdate != nil {
				w.onInteractionUpdate(gestures.NewScaleUpdateDetails(gestures.ScaleUpdateOpts{
					FocalPoint: global.Sub(scrollDelta), LocalFocalPoint: foundation.Some(local.Sub(localDelta)), FocalPointDelta: localDelta.Neg()}))
			}
			if w.onInteractionEnd != nil {
				w.onInteractionEnd(gestures.ScaleEndDetails{})
			}
			return
		}
		// Ignore left and right scroll.
		if scrollDelta.Dy == 0 {
			return
		}
		// Handle mouse scroll.
		scaleChange = math.Exp(-scrollDelta.Dy / w.scaleFactor)
	case *gestures.PointerScaleEvent:
		scaleChange = e.Scale()
	default:
		return
	}
	if w.onInteractionStart != nil {
		w.onInteractionStart(gestures.NewScaleStartDetails(gestures.ScaleStartOpts{FocalPoint: global, LocalFocalPoint: foundation.Some(local)}))
	}
	if !s.gestureIsSupported(gestureScale) {
		if w.onInteractionUpdate != nil {
			w.onInteractionUpdate(gestures.NewScaleUpdateDetails(gestures.ScaleUpdateOpts{
				FocalPoint: global, LocalFocalPoint: foundation.Some(local), Scale: foundation.Some(scaleChange)}))
		}
		if w.onInteractionEnd != nil {
			w.onInteractionEnd(gestures.ScaleEndDetails{})
		}
		return
	}
	focalPointScene := s.transformer.ToScene(local)
	s.transformer.SetValue(s.matrixScale(s.transformer.Value(), scaleChange))
	// After scaling, translate such that the event's position is at the same
	// scene point before and after the scale.
	focalPointSceneScaled := s.transformer.ToScene(local)
	s.transformer.SetValue(s.matrixTranslate(s.transformer.Value(), focalPointSceneScaled.Sub(focalPointScene)))
	if w.onInteractionUpdate != nil {
		w.onInteractionUpdate(gestures.NewScaleUpdateDetails(gestures.ScaleUpdateOpts{
			FocalPoint: global, LocalFocalPoint: foundation.Some(local), Scale: foundation.Some(scaleChange)}))
	}
	if w.onInteractionEnd != nil {
		w.onInteractionEnd(gestures.ScaleEndDetails{})
	}
}

// handleInertiaAnimation handles a tick of the inertia of a pan.
func (s *interactiveViewerState) handleInertiaAnimation() {
	if !s.controller.IsAnimating() {
		s.currentAxis = foundation.None[painting.Axis]()
		s.removeInertiaListener()
		s.controller.Reset()
		return
	}
	translationVector := s.transformer.Value().GetTranslation()
	translation := ui.Offset{Dx: float64(translationVector.X), Dy: float64(translationVector.Y)}
	s.transformer.SetValue(s.matrixTranslate(s.transformer.Value(), s.transformer.ToScene(s.animation.Value()).Sub(s.transformer.ToScene(translation))))
}

// handleScaleAnimation handles a tick of the inertia of a scale.
func (s *interactiveViewerState) handleScaleAnimation() {
	if !s.scaleController.IsAnimating() {
		s.currentAxis = foundation.None[painting.Axis]()
		s.removeScaleListener()
		s.scaleController.Reset()
		return
	}
	desiredScale := s.scaleAnimation.Value()
	scaleChange := desiredScale / s.transformer.Value().GetMaxScaleOnAxis()
	referenceFocalPoint := s.transformer.ToScene(s.scaleAnimationFocal)
	s.transformer.SetValue(s.matrixScale(s.transformer.Value(), scaleChange))
	// While scaling, translate such that the user's two fingers stay on the same
	// places in the scene.
	focalPointSceneScaled := s.transformer.ToScene(s.scaleAnimationFocal)
	s.transformer.SetValue(s.matrixTranslate(s.transformer.Value(), focalPointSceneScaled.Sub(referenceFocalPoint)))
}

func (s *interactiveViewerState) handleTransformation() {
	// A change to the TransformationController's value is a change to the state.
	s.SetState(func() {})
}

func (s *interactiveViewerState) InitState() {
	s.transformer = s.Widget().transformationController
	if s.transformer == nil {
		s.transformer = NewTransformationController(nil)
	}
	s.controller = animation.NewAnimationController(s)
	s.scaleController = animation.NewAnimationController(s)
	s.transformerListener = s.transformer.AddListener(s.handleTransformation)
}

func (s *interactiveViewerState) DidUpdateWidget(oldWidget *InteractiveViewer) {
	newController := s.Widget().transformationController
	if newController == oldWidget.transformationController {
		return
	}
	s.transformer.RemoveListener(s.transformerListener)
	if oldWidget.transformationController == nil {
		s.transformer.Dispose()
	}
	s.transformer = newController
	if s.transformer == nil {
		s.transformer = NewTransformationController(nil)
	}
	s.transformerListener = s.transformer.AddListener(s.handleTransformation)
}

func (s *interactiveViewerState) Dispose() {
	s.controller.Dispose()
	s.scaleController.Dispose()
	s.transformer.RemoveListener(s.transformerListener)
	if s.Widget().transformationController == nil {
		s.transformer.Dispose()
	}
}

func (s *interactiveViewerState) Build(BuildContext) Widget {
	w := s.Widget()
	var child Widget
	if w.child != nil {
		child = &interactiveViewerBuilt{
			StatelessWidget: NewStatelessWidgetBase(), child: w.child, childKey: s.childKey, clipBehavior: w.clipBehavior,
			constrained: w.constrained, matrix: s.transformer.Value(), alignment: w.alignment}
	} else {
		if foundation.DebugMode && (w.builder == nil || w.constrained) {
			panic(errors.New(tagInteractiveViewerBuilderInvalid, "a builder viewer needs a builder and can not be constrained"))
		}
		child = NewLayoutBuilder(func(bc BuildContext, constraints rendering.BoxConstraints) Widget {
			matrix := s.transformer.Value()
			return &interactiveViewerBuilt{
				StatelessWidget: NewStatelessWidgetBase(), child: w.builder(bc, transformViewport(matrix, ui.OffsetZero.And(constraints.Biggest()))),
				childKey: s.childKey, clipBehavior: w.clipBehavior, constrained: w.constrained, matrix: matrix, alignment: w.alignment}
		})
	}
	return NewListener(NewGestureDetector(child, GestureDetectorOpts{
		Behavior:   rendering.HitTestBehaviorOpaque, // Necessary when panning off screen.
		OnScaleEnd: s.onScaleEnd, OnScaleStart: s.onScaleStart, OnScaleUpdate: s.onScaleUpdate,
		TrackpadScrollCausesScale:   w.trackpadScrollCausesScale,
		TrackpadScrollToScaleFactor: foundation.Some(ui.Offset{Dx: 0, Dy: -1 / w.scaleFactor}),
	}), ListenerOpts{Key: s.parentKey, OnPointerSignal: s.receivedPointerSignal})
}

// interactiveViewerBuilt is the transformed, clipped child of an
// InteractiveViewer.
//
// Flutter: widgets/interactive_viewer.dart _InteractiveViewerBuilt
type interactiveViewerBuilt struct {
	StatelessWidget
	child        Widget
	childKey     *GlobalKey[State]
	clipBehavior ui.Clip
	constrained  bool
	matrix       *ui.Matrix4
	alignment    painting.AlignmentGeometry
}

func (w *interactiveViewerBuilt) Build(BuildContext) Widget {
	var child Widget = NewTransform(w.matrix, NewKeyedSubtree(w.child, WidgetOpts{Key: w.childKey}), TransformOpts{Alignment: w.alignment})
	if !w.constrained {
		child = NewOverflowBox(child, OverflowBoxOpts{
			Alignment: painting.AlignmentTopLeft, MinWidth: foundation.Some(0.0), MinHeight: foundation.Some(0.0),
			MaxWidth: foundation.Some(math.Inf(1)), MaxHeight: foundation.Some(math.Inf(1))})
	}
	return NewClipRect(child, ClipRectOpts{ClipBehavior: w.clipBehavior})
}

// TransformationController is the controller of the transformation of an
// InteractiveViewer: it holds the matrix.
//
// Flutter: widgets/interactive_viewer.dart TransformationController
type TransformationController struct {
	foundation.ValueNotifier[*ui.Matrix4]
}

// NewTransformationController creates a controller with the matrix, or the
// identity if it is nil.
//
// Flutter: widgets/interactive_viewer.dart TransformationController.new
func NewTransformationController(value *ui.Matrix4) *TransformationController {
	if value == nil {
		value = ui.NewMatrix4Identity()
	}
	return &TransformationController{*foundation.NewValueNotifier(value)}
}

// ToScene transforms a point in the coordinate system of the viewport to the one
// of the scene (the child).
//
// Flutter: widgets/interactive_viewer.dart TransformationController.toScene
func (c *TransformationController) ToScene(viewportPoint ui.Offset) ui.Offset {
	inverse := c.Value().Inverted()
	untransformed := inverse.Transform3(mkV3(viewportPoint.Dx, viewportPoint.Dy, 0))
	return ui.Offset{Dx: float64(untransformed.X), Dy: float64(untransformed.Y)}
}

// interactiveFinalTime is the time at which the velocity has decayed to the
// motionless threshold.
//
// Flutter: widgets/interactive_viewer.dart _getFinalTime
func interactiveFinalTime(velocity, drag, effectivelyMotionless float64) float64 {
	return math.Log(effectivelyMotionless/velocity) / math.Log(drag/100)
}

func matrixTranslation(matrix *ui.Matrix4) ui.Offset {
	t := matrix.GetTranslation()
	return ui.Offset{Dx: float64(t.X), Dy: float64(t.Y)}
}

// transformViewport is the viewport, as a quad, in the coordinates of the scene
// that the matrix maps it to.
//
// Flutter: widgets/interactive_viewer.dart _transformViewport
func transformViewport(matrix *ui.Matrix4, viewport ui.Rect) ui.Quad {
	inverse := matrix.Clone()
	inverse.Invert()
	return ui.NewQuadPoints(
		inverse.Transform3(mkV3(viewport.TopLeft().Dx, viewport.TopLeft().Dy, 0)),
		inverse.Transform3(mkV3(viewport.TopRight().Dx, viewport.TopRight().Dy, 0)),
		inverse.Transform3(mkV3(viewport.BottomRight().Dx, viewport.BottomRight().Dy, 0)),
		inverse.Transform3(mkV3(viewport.BottomLeft().Dx, viewport.BottomLeft().Dy, 0)))
}

func axisAlignedBoundingBoxWithRotation(rect ui.Rect, rotation float64) ui.Quad {
	rotationMatrix := ui.NewMatrix4Identity()
	rotationMatrix.Translate(float32(rect.Width()/2), float32(rect.Height()/2), 0)
	rotationMatrix.RotateZ(rotation)
	rotationMatrix.Translate(float32(-rect.Width()/2), float32(-rect.Height()/2), 0)
	boundariesRotated := ui.NewQuadPoints(
		rotationMatrix.Transform3(mkV3(rect.Left, rect.Top, 0)),
		rotationMatrix.Transform3(mkV3(rect.Right, rect.Top, 0)),
		rotationMatrix.Transform3(mkV3(rect.Right, rect.Bottom, 0)),
		rotationMatrix.Transform3(mkV3(rect.Left, rect.Bottom, 0)))
	return InteractiveViewerGetAxisAlignedBoundingBox(boundariesRotated)
}

// exceedsBy is how far the viewport is outside of the boundary, along each axis;
// zero if it is inside.
//
// Flutter: widgets/interactive_viewer.dart _exceedsBy
func exceedsBy(boundary, viewport ui.Quad) ui.Offset {
	largestExcess := ui.Offset{}
	for _, point := range []ui.Vector3{viewport.Point0, viewport.Point1, viewport.Point2, viewport.Point3} {
		pointInside := InteractiveViewerGetNearestPointInside(point, boundary)
		excess := ui.Offset{Dx: float64(pointInside.X) - float64(point.X), Dy: float64(pointInside.Y) - float64(point.Y)}
		if math.Abs(excess.Dx) > math.Abs(largestExcess.Dx) {
			largestExcess.Dx = excess.Dx
		}
		if math.Abs(excess.Dy) > math.Abs(largestExcess.Dy) {
			largestExcess.Dy = excess.Dy
		}
	}
	return roundOffset(largestExcess)
}

// roundOffset rounds the offset to remove floating point noise.
//
// Deviation: Dart rounds to 9 decimals; the matrices of Glutter keep float32
// values (SPEC §7), whose noise is larger, so the offset is rounded to 5
// decimals (SPEC §12.10).
//
// Flutter: widgets/interactive_viewer.dart _round
func roundOffset(offset ui.Offset) ui.Offset {
	round := func(v float64) float64 {
		r, err := strconv.ParseFloat(strconv.FormatFloat(v, 'f', 5, 64), 64)
		if err != nil {
			return v
		}
		return r + 0 // avoids negative zero
	}
	return ui.Offset{Dx: round(offset.Dx), Dy: round(offset.Dy)}
}

func alignAxis(offset ui.Offset, axis painting.Axis) ui.Offset {
	if axis == painting.AxisHorizontal {
		return ui.Offset{Dx: offset.Dx, Dy: 0}
	}
	return ui.Offset{Dx: 0, Dy: offset.Dy}
}

// panAxisOf is the axis along which the two points mostly differ, or none if they
// are the same.
//
// Flutter: widgets/interactive_viewer.dart _getPanAxis
func panAxisOf(point1, point2 ui.Offset) foundation.Opt[painting.Axis] {
	if point1 == point2 {
		return foundation.None[painting.Axis]()
	}
	x, y := point2.Dx-point1.Dx, point2.Dy-point1.Dy
	if math.Abs(x) > math.Abs(y) {
		return foundation.Some(painting.AxisHorizontal)
	}
	return foundation.Some(painting.AxisVertical)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagInteractiveViewerBoundaryInvalid       = "InteractiveViewerBoundaryInvalid"
	tagInteractiveViewerBoundaryMarginInvalid = "InteractiveViewerBoundaryMarginInvalid"
	tagInteractiveViewerBuilderInvalid        = "InteractiveViewerBuilderInvalid"
	tagInteractiveViewerChildEmpty            = "InteractiveViewerChildEmpty"
	tagInteractiveViewerFrictionInvalid       = "InteractiveViewerFrictionInvalid"
	tagInteractiveViewerScaleInvalid          = "InteractiveViewerScaleInvalid"
	tagInteractiveViewerScaleZero             = "InteractiveViewerScaleZero"
)
