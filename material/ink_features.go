package material

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// InteractiveInkFeature is an ink feature that responds to a gesture: a splash
// that grows when a tap is confirmed and fades when it is cancelled.
//
// Flutter: material/ink_well.dart InteractiveInkFeature
type InteractiveInkFeature interface {
	InkFeature
	// Confirm is called when the user input that triggered this feature's
	// appearance was confirmed (for example, a tap was recognized).
	//
	// Flutter: material/ink_well.dart InteractiveInkFeature.confirm
	Confirm()
	// Cancel is called when the user input that triggered this feature's
	// appearance was canceled.
	//
	// Flutter: material/ink_well.dart InteractiveInkFeature.cancel
	Cancel()
	// AsInteractiveInkFeature returns the base of the feature.
	AsInteractiveInkFeature() *InteractiveInkFeatureBase
}

// InteractiveInkFeatureBase is the embedded base of an InteractiveInkFeature:
// the color and the custom border of the feature.
//
// Flutter: material/ink_well.dart InteractiveInkFeature
type InteractiveInkFeatureBase struct {
	InkFeatureBase
	color        ui.Color
	customBorder painting.ShapeBorder
}

// InitInteractiveInkFeature initializes the base of the feature self.
//
// Flutter: material/ink_well.dart InteractiveInkFeature.new
func InitInteractiveInkFeature(f *InteractiveInkFeatureBase, self InteractiveInkFeature, controller MaterialInkController,
	referenceBox rendering.Box, color ui.Color, customBorder painting.ShapeBorder, onRemoved func()) {
	InitInkFeature(&f.InkFeatureBase, self, controller, referenceBox, InkFeatureOpts{OnRemoved: onRemoved})
	f.color = color
	f.customBorder = customBorder
}

// AsInteractiveInkFeature returns the base.
func (f *InteractiveInkFeatureBase) AsInteractiveInkFeature() *InteractiveInkFeatureBase { return f }

// Color is the primary color used to paint the feature.
//
// Flutter: material/ink_well.dart InteractiveInkFeature.color
func (f *InteractiveInkFeatureBase) Color() ui.Color { return f.color }

// SetColor changes the color and repaints when it changed.
//
// Flutter: material/ink_well.dart InteractiveInkFeature.color
func (f *InteractiveInkFeatureBase) SetColor(value ui.Color) {
	if value == f.color {
		return
	}
	f.color = value
	f.controller.MarkNeedsPaint()
}

// CustomBorder is the custom shape of the feature's clip, or nil.
//
// Flutter: material/ink_well.dart InteractiveInkFeature.customBorder
func (f *InteractiveInkFeatureBase) CustomBorder() painting.ShapeBorder { return f.customBorder }

// SetCustomBorder changes the custom border and repaints when it changed.
//
// Flutter: material/ink_well.dart InteractiveInkFeature.customBorder
func (f *InteractiveInkFeatureBase) SetCustomBorder(value painting.ShapeBorder) {
	if foundation.ValuesEqual(value, f.customBorder) {
		return
	}
	f.customBorder = value
	f.controller.MarkNeedsPaint()
}

// InkCircle describes a circle that PaintInkCircle paints.
//
// Flutter: material/ink_well.dart InteractiveInkFeature.paintInkCircle
type InkCircle struct {
	// Transform takes the coordinate system of the reference box to the one of the canvas.
	Transform *ui.Matrix4
	// Paint is the paint of the circle.
	Paint *ui.Paint
	// Center is the center of the circle.
	Center ui.Offset
	// Radius is the radius of the circle.
	Radius float64
	// TextDirection is the direction for a custom border that depends on it.
	TextDirection ui.TextDirection
	// CustomBorder clips the circle to the outer path of the shape.
	CustomBorder painting.ShapeBorder
	// BorderRadius clips the circle to a rounded rectangle when it is not zero.
	BorderRadius painting.BorderRadius
	// ClipCallback gives the rectangle to clip to; nil for no clip.
	ClipCallback RectCallback
}

// PaintInkCircle paints a circle centered on the center, clipped as the
// InkCircle says, with the transform applied.
//
// Flutter: material/ink_well.dart InteractiveInkFeature.paintInkCircle
func PaintInkCircle(canvas *ui.Canvas, c InkCircle) {
	originOffset, isTranslation := c.Transform.AsTranslation()
	canvas.Save()
	if !isTranslation {
		canvas.Transform(c.Transform)
	} else {
		canvas.Translate(originOffset.Dx, originOffset.Dy)
	}
	if c.ClipCallback != nil {
		rect := c.ClipCallback()
		switch {
		case c.CustomBorder != nil:
			canvas.ClipPath(c.CustomBorder.GetOuterPath(rect, c.TextDirection))
		case c.BorderRadius != painting.BorderRadiusZero:
			canvas.ClipRRect(c.BorderRadius.ToRRect(rect))
		default:
			canvas.ClipRect(rect)
		}
	}
	canvas.DrawCircle(c.Center, c.Radius, c.Paint)
	canvas.Restore()
}

// InkFeatureCreateOpts are the parameters of InteractiveInkFeatureFactory.Create.
//
// Flutter: material/ink_well.dart InteractiveInkFeatureFactory.create
type InkFeatureCreateOpts struct {
	// Controller is the controller of the material that the feature paints on.
	Controller MaterialInkController
	// ReferenceBox is the render box whose coordinate system the feature uses.
	ReferenceBox rendering.Box
	// Position is the position of the feature, in the coordinates of the reference box.
	Position ui.Offset
	// Color is the color of the feature.
	Color ui.Color
	// TextDirection is the text direction of the context.
	TextDirection ui.TextDirection
	// ContainedInkWell says whether the feature is clipped to the reference box (or to the RectCallback).
	ContainedInkWell bool
	// RectCallback gives the rectangle that the feature is clipped to; it requires ContainedInkWell.
	RectCallback RectCallback
	// BorderRadius clips the feature to rounded corners.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// CustomBorder clips the feature to a shape.
	CustomBorder painting.ShapeBorder
	// Radius is the radius of the feature; the default depends on the feature.
	Radius foundation.Opt[float64]
	// OnRemoved is called when the feature is disposed.
	OnRemoved func()
}

// InteractiveInkFeatureFactory defines the interface for InteractiveInkFeature
// subclasses to create splashes: the theme and the InkWell choose the factory.
//
// Flutter: material/ink_well.dart InteractiveInkFeatureFactory
type InteractiveInkFeatureFactory interface {
	// Create makes the feature.
	//
	// Flutter: material/ink_well.dart InteractiveInkFeatureFactory.create
	Create(o InkFeatureCreateOpts) InteractiveInkFeature
}

// inkClipCallback is Flutter's _getClipCallback of ink_splash.dart and
// ink_ripple.dart.
func inkClipCallback(referenceBox rendering.Box, containedInkWell bool, rectCallback RectCallback) RectCallback {
	if rectCallback != nil {
		return rectCallback
	}
	if containedInkWell {
		return func() ui.Rect { return ui.OffsetZero.And(referenceBox.Size()) }
	}
	return nil
}

func zeroBorderRadius(r foundation.Opt[painting.BorderRadius]) painting.BorderRadius {
	return r.OrElse(painting.BorderRadiusZero)
}

const defaultHighlightFadeDuration = 200 * time.Millisecond

// InkHighlight is a visual emphasis on a part of a Material receiving user
// interaction: a box that fades in when the user presses, hovers or focuses.
//
// Flutter: material/ink_highlight.dart InkHighlight
type InkHighlight struct {
	InteractiveInkFeatureBase
	shape           painting.BoxShape
	radius          foundation.Opt[float64]
	borderRadius    painting.BorderRadius
	rectCallback    RectCallback
	textDirection   ui.TextDirection
	alpha           animation.Animation[int]
	alphaController *animation.AnimationController
	active          bool
}

// InkHighlightOpts are the optional parameters of NewInkHighlight.
//
// Flutter: material/ink_highlight.dart InkHighlight.new
type InkHighlightOpts struct {
	// Shape is the shape of the highlight; the default is a rectangle.
	Shape painting.BoxShape
	// Radius is the radius of a circle highlight; the default is Material's default splash radius.
	Radius foundation.Opt[float64]
	// BorderRadius is the corner radii of a rectangle highlight.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// CustomBorder clips the highlight to a shape.
	CustomBorder painting.ShapeBorder
	// RectCallback gives the rectangle of the highlight; the default is the reference box.
	RectCallback RectCallback
	// OnRemoved is called when the highlight is disposed.
	OnRemoved func()
	// FadeDuration is how long the highlight takes to fade; the default is 200ms.
	FadeDuration time.Duration
}

// NewInkHighlight begins a highlight, that fades in, on the material of the
// controller and adds it to it.
//
// Flutter: material/ink_highlight.dart InkHighlight.new
func NewInkHighlight(controller MaterialInkController, referenceBox rendering.Box, color ui.Color, textDirection ui.TextDirection, opts ...InkHighlightOpts) *InkHighlight {
	o := firstOpts(opts)
	if o.Shape == painting.BoxShapeUnspecified {
		o.Shape = painting.BoxShapeRectangle
	}
	if o.FadeDuration == 0 {
		o.FadeDuration = defaultHighlightFadeDuration
	}
	h := &InkHighlight{shape: o.Shape, radius: o.Radius, borderRadius: zeroBorderRadius(o.BorderRadius), textDirection: textDirection,
		rectCallback: o.RectCallback, active: true}
	InitInteractiveInkFeature(&h.InteractiveInkFeatureBase, h, controller, referenceBox, color, o.CustomBorder, o.OnRemoved)
	h.alphaController = animation.NewAnimationController(controller.Vsync(), animation.AnimationControllerOpts{Duration: foundation.Some(o.FadeDuration)})
	h.alphaController.AddListener(controller.MarkNeedsPaint)
	h.alphaController.AddStatusListener(h.handleAlphaStatusChanged)
	h.alphaController.Forward()
	h.alpha = animation.Drive[int](h.alphaController, animation.NewIntTween(0, color.Alpha()))
	controller.AddInkFeature(h)
	return h
}

// Active is whether the highlight is shown (it is not while it fades out).
//
// Flutter: material/ink_highlight.dart InkHighlight.active
func (h *InkHighlight) Active() bool { return h.active }

// Activate makes the highlight fade in again.
//
// Flutter: material/ink_highlight.dart InkHighlight.activate
func (h *InkHighlight) Activate() {
	h.active = true
	h.alphaController.Forward()
}

// Deactivate makes the highlight fade out, and then disposes it.
//
// Flutter: material/ink_highlight.dart InkHighlight.deactivate
func (h *InkHighlight) Deactivate() {
	h.active = false
	h.alphaController.Reverse()
}

func (h *InkHighlight) handleAlphaStatusChanged(status animation.AnimationStatus) {
	if status.IsDismissed() && !h.active {
		h.Dispose()
	}
}

// Confirm does nothing: a highlight is not confirmed.
func (h *InkHighlight) Confirm() {}

// Cancel does nothing: a highlight is not cancelled.
func (h *InkHighlight) Cancel() {}

// Dispose releases the animation and removes the highlight.
//
// Flutter: material/ink_highlight.dart InkHighlight.dispose
func (h *InkHighlight) Dispose() {
	h.alphaController.Dispose()
	h.InkFeatureBase.Dispose()
}

func (h *InkHighlight) paintHighlight(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint) {
	canvas.Save()
	if h.customBorder != nil {
		canvas.ClipPath(h.customBorder.GetOuterPath(rect, h.textDirection))
	}
	switch h.shape {
	case painting.BoxShapeCircle:
		canvas.DrawCircle(rect.Center(), h.radius.OrElse(MaterialDefaultSplashRadius), paint)
	default:
		if h.borderRadius != painting.BorderRadiusZero {
			canvas.DrawRRect(h.borderRadius.ToRRect(rect), paint)
		} else {
			canvas.DrawRect(rect, paint)
		}
	}
	canvas.Restore()
}

// PaintFeature paints the highlight.
func (h *InkHighlight) PaintFeature(canvas *ui.Canvas, transform *ui.Matrix4) {
	paint := ui.NewPaint()
	paint.SetColor(h.color.WithAlpha(h.alpha.Value()))
	originOffset, isTranslation := transform.AsTranslation()
	rect := ui.OffsetZero.And(h.referenceBox.Size())
	if h.rectCallback != nil {
		rect = h.rectCallback()
	}
	if !isTranslation {
		canvas.Save()
		canvas.Transform(transform)
		h.paintHighlight(canvas, rect, paint)
		canvas.Restore()
	} else {
		h.paintHighlight(canvas, rect.Shift(originOffset), paint)
	}
}

const (
	unconfirmedSplashDuration = 1 * time.Second
	splashFadeDuration        = 200 * time.Millisecond
	splashInitialSize         = 0.0 // logical pixels
	splashConfirmedVelocity   = 1.0 // logical pixels per millisecond
)

// inkSplashTargetRadius is Flutter's _getTargetRadius of ink_splash.dart.
func inkSplashTargetRadius(referenceBox rendering.Box, containedInkWell bool, rectCallback RectCallback, position ui.Offset) float64 {
	if containedInkWell {
		size := referenceBox.Size()
		if rectCallback != nil {
			size = rectCallback().Size()
		}
		return splashRadiusForPositionInSize(size, position)
	}
	return MaterialDefaultSplashRadius
}

func splashRadiusForPositionInSize(bounds ui.Size, position ui.Offset) float64 {
	d1 := position.Sub(bounds.TopLeft(ui.OffsetZero)).Distance()
	d2 := position.Sub(bounds.TopRight(ui.OffsetZero)).Distance()
	d3 := position.Sub(bounds.BottomLeft(ui.OffsetZero)).Distance()
	d4 := position.Sub(bounds.BottomRight(ui.OffsetZero)).Distance()
	return math.Ceil(math.Max(math.Max(d1, d2), math.Max(d3, d4)))
}

type inkSplashFactory struct{}

func (inkSplashFactory) Create(o InkFeatureCreateOpts) InteractiveInkFeature {
	return NewInkSplash(o.Controller, o.ReferenceBox, o.TextDirection, o.Color, InkSplashOpts{
		Position: foundation.Some(o.Position), ContainedInkWell: o.ContainedInkWell, RectCallback: o.RectCallback,
		BorderRadius: o.BorderRadius, CustomBorder: o.CustomBorder, Radius: o.Radius, OnRemoved: o.OnRemoved})
}

// InkSplashFactory creates InkSplash features: the splash of Material 2. It is
// read-only.
//
// Flutter: material/ink_splash.dart InkSplash.splashFactory
var InkSplashFactory InteractiveInkFeatureFactory = inkSplashFactory{}

// InkSplash is a visual reaction on a piece of Material to user input: a circular
// ink feature whose origin starts at the input touch point and whose radius
// expands from zero.
//
// Flutter: material/ink_splash.dart InkSplash
type InkSplash struct {
	InteractiveInkFeatureBase
	position              ui.Offset
	hasPosition           bool
	borderRadius          painting.BorderRadius
	targetRadius          float64
	clipCallback          RectCallback
	repositionToReference bool
	textDirection         ui.TextDirection
	radius                animation.Animation[float64]
	radiusController      *animation.AnimationController
	alpha                 animation.Animation[int]
	alphaController       *animation.AnimationController
}

// InkSplashOpts are the optional parameters of NewInkSplash.
//
// Flutter: material/ink_splash.dart InkSplash.new
type InkSplashOpts struct {
	// Position is the position of the touch, in the coordinates of the reference box.
	Position foundation.Opt[ui.Offset]
	// ContainedInkWell says whether the splash is clipped to the reference box (or to the RectCallback).
	ContainedInkWell bool
	// RectCallback gives the rectangle that the splash is clipped to; it requires ContainedInkWell.
	RectCallback RectCallback
	// BorderRadius clips the splash to rounded corners.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// CustomBorder clips the splash to a shape.
	CustomBorder painting.ShapeBorder
	// Radius is the final radius of the splash.
	Radius foundation.Opt[float64]
	// OnRemoved is called when the splash is disposed.
	OnRemoved func()
}

// NewInkSplash begins a splash, centered at the position, on the material of the
// controller and adds it to it. The position is required unless the radius is
// given and the splash is not contained.
//
// Flutter: material/ink_splash.dart InkSplash.new
func NewInkSplash(controller MaterialInkController, referenceBox rendering.Box, textDirection ui.TextDirection, color ui.Color, opts ...InkSplashOpts) *InkSplash {
	o := firstOpts(opts)
	s := &InkSplash{position: o.Position.OrElse(ui.OffsetZero), hasPosition: o.Position.IsSome(), borderRadius: zeroBorderRadius(o.BorderRadius),
		clipCallback: inkClipCallback(referenceBox, o.ContainedInkWell, o.RectCallback), repositionToReference: !o.ContainedInkWell, textDirection: textDirection}
	if r, ok := o.Radius.Get(); ok {
		s.targetRadius = r
	} else {
		s.targetRadius = inkSplashTargetRadius(referenceBox, o.ContainedInkWell, o.RectCallback, s.position)
	}
	InitInteractiveInkFeature(&s.InteractiveInkFeatureBase, s, controller, referenceBox, color, o.CustomBorder, o.OnRemoved)
	s.radiusController = animation.NewAnimationController(controller.Vsync(), animation.AnimationControllerOpts{Duration: foundation.Some(unconfirmedSplashDuration)})
	s.radiusController.AddListener(controller.MarkNeedsPaint)
	s.radiusController.Forward()
	s.radius = animation.Drive[float64](s.radiusController, animation.NewTween(splashInitialSize, s.targetRadius))
	s.alphaController = animation.NewAnimationController(controller.Vsync(), animation.AnimationControllerOpts{Duration: foundation.Some(splashFadeDuration)})
	s.alphaController.AddListener(controller.MarkNeedsPaint)
	s.alphaController.AddStatusListener(s.handleAlphaStatusChanged)
	s.alpha = animation.Drive[int](s.alphaController, animation.NewIntTween(color.Alpha(), 0))
	controller.AddInkFeature(s)
	return s
}

// Confirm makes the splash expand to its full size and fade out.
//
// Flutter: material/ink_splash.dart InkSplash.confirm
func (s *InkSplash) Confirm() {
	duration := int(math.Floor(s.targetRadius / splashConfirmedVelocity))
	s.radiusController.Duration = foundation.Some(time.Duration(duration) * time.Millisecond)
	s.radiusController.Forward()
	s.alphaController.Forward()
}

// Cancel makes the splash fade out.
//
// Flutter: material/ink_splash.dart InkSplash.cancel
func (s *InkSplash) Cancel() {
	if s.alphaController != nil {
		s.alphaController.Forward()
	}
}

func (s *InkSplash) handleAlphaStatusChanged(status animation.AnimationStatus) {
	if status.IsCompleted() {
		s.Dispose()
	}
}

// Dispose releases the animations and removes the splash.
//
// Flutter: material/ink_splash.dart InkSplash.dispose
func (s *InkSplash) Dispose() {
	s.radiusController.Dispose()
	s.alphaController.Dispose()
	s.alphaController = nil
	s.InkFeatureBase.Dispose()
}

// PaintFeature paints the splash.
func (s *InkSplash) PaintFeature(canvas *ui.Canvas, transform *ui.Matrix4) {
	paint := ui.NewPaint()
	paint.SetColor(s.color.WithAlpha(s.alpha.Value()))
	center := s.position
	if s.repositionToReference {
		center = s.position.Lerp(s.referenceBox.Size().Center(ui.OffsetZero), s.radiusController.Value())
	}
	PaintInkCircle(canvas, InkCircle{Transform: transform, Paint: paint, Center: center, TextDirection: s.textDirection, Radius: s.radius.Value(),
		CustomBorder: s.customBorder, BorderRadius: s.borderRadius, ClipCallback: s.clipCallback})
}

const (
	unconfirmedRippleDuration = 1 * time.Second
	rippleFadeInDuration      = 75 * time.Millisecond
	rippleRadiusDuration      = 225 * time.Millisecond
	rippleFadeOutDuration     = 375 * time.Millisecond
	rippleCancelDuration      = 75 * time.Millisecond
	// The fade out begins 225ms after the fadeOutController starts. See Confirm.
	rippleFadeOutIntervalStart = 225.0 / 375.0
)

func inkRippleTargetRadius(referenceBox rendering.Box, rectCallback RectCallback) float64 {
	size := referenceBox.Size()
	if rectCallback != nil {
		size = rectCallback().Size()
	}
	d1 := size.BottomRight(ui.OffsetZero).Distance()
	d2 := size.TopRight(ui.OffsetZero).Sub(size.BottomLeft(ui.OffsetZero)).Distance()
	return math.Max(d1, d2) / 2.0
}

type inkRippleFactory struct{}

func (inkRippleFactory) Create(o InkFeatureCreateOpts) InteractiveInkFeature {
	return NewInkRipple(o.Controller, o.ReferenceBox, o.Position, o.Color, o.TextDirection, InkRippleOpts{
		ContainedInkWell: o.ContainedInkWell, RectCallback: o.RectCallback, BorderRadius: o.BorderRadius,
		CustomBorder: o.CustomBorder, Radius: o.Radius, OnRemoved: o.OnRemoved})
}

// InkRippleFactory creates InkRipple features: the splash of Material 3 on every
// platform but Android. It is read-only.
//
// Flutter: material/ink_ripple.dart InkRipple.splashFactory
var InkRippleFactory InteractiveInkFeatureFactory = inkRippleFactory{}

// InkRipple is a visual reaction on a piece of Material to user input: a circular
// ink feature whose origin starts at the input touch point and whose radius
// expands from 60% of the final radius. The splash origin animates to the center
// of its reference box.
//
// Flutter: material/ink_ripple.dart InkRipple
type InkRipple struct {
	InteractiveInkFeatureBase
	position          ui.Offset
	borderRadius      painting.BorderRadius
	targetRadius      float64
	clipCallback      RectCallback
	textDirection     ui.TextDirection
	radius            animation.Animation[float64]
	radiusController  *animation.AnimationController
	fadeIn            animation.Animation[int]
	fadeInController  *animation.AnimationController
	fadeOut           animation.Animation[int]
	fadeOutController *animation.AnimationController
}

// InkRippleOpts are the optional parameters of NewInkRipple.
//
// Flutter: material/ink_ripple.dart InkRipple.new
type InkRippleOpts struct {
	// ContainedInkWell says whether the ripple is clipped to the reference box (or to the RectCallback).
	ContainedInkWell bool
	// RectCallback gives the rectangle that the ripple is clipped to; it requires ContainedInkWell.
	RectCallback RectCallback
	// BorderRadius clips the ripple to rounded corners.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// CustomBorder clips the ripple to a shape.
	CustomBorder painting.ShapeBorder
	// Radius is the target radius of the ripple.
	Radius foundation.Opt[float64]
	// OnRemoved is called when the ripple is disposed.
	OnRemoved func()
}

var (
	rippleEaseCurveTween       = animation.NewCurveTween(animation.Curves.Ease)
	rippleFadeOutIntervalTween = animation.NewCurveTween(animation.NewInterval(rippleFadeOutIntervalStart, 1.0))
)

// NewInkRipple begins a ripple, centered at the position relative to the
// reference box, on the material of the controller and adds it to it.
//
// Flutter: material/ink_ripple.dart InkRipple.new
func NewInkRipple(controller MaterialInkController, referenceBox rendering.Box, position ui.Offset, color ui.Color, textDirection ui.TextDirection, opts ...InkRippleOpts) *InkRipple {
	o := firstOpts(opts)
	r := &InkRipple{position: position, borderRadius: zeroBorderRadius(o.BorderRadius), textDirection: textDirection,
		clipCallback: inkClipCallback(referenceBox, o.ContainedInkWell, o.RectCallback)}
	if v, ok := o.Radius.Get(); ok {
		r.targetRadius = v
	} else {
		r.targetRadius = inkRippleTargetRadius(referenceBox, o.RectCallback)
	}
	InitInteractiveInkFeature(&r.InteractiveInkFeatureBase, r, controller, referenceBox, color, o.CustomBorder, o.OnRemoved)
	vsync := controller.Vsync()
	// Immediately begin fading-in the initial splash.
	r.fadeInController = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{Duration: foundation.Some(rippleFadeInDuration)})
	r.fadeInController.AddListener(controller.MarkNeedsPaint)
	r.fadeInController.Forward()
	r.fadeIn = animation.Drive[int](r.fadeInController, animation.NewIntTween(0, color.Alpha()))
	// Controls the splash radius and its center. Starts upon confirm.
	r.radiusController = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{Duration: foundation.Some(unconfirmedRippleDuration)})
	r.radiusController.AddListener(controller.MarkNeedsPaint)
	r.radiusController.Forward()
	// Initial splash diameter is 60% of the target diameter, final diameter is
	// 10dps larger than the target diameter.
	r.radius = animation.Drive[float64](r.radiusController, animation.NewTween(r.targetRadius*0.30, r.targetRadius+5.0).Chain(rippleEaseCurveTween))
	// Controls the splash radius and its center. Starts upon confirm however its
	// Interval delays changes until the radius expansion has completed.
	r.fadeOutController = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{Duration: foundation.Some(rippleFadeOutDuration)})
	r.fadeOutController.AddListener(controller.MarkNeedsPaint)
	r.fadeOutController.AddStatusListener(r.handleAlphaStatusChanged)
	r.fadeOut = animation.Drive[int](r.fadeOutController, animation.NewIntTween(color.Alpha(), 0).Chain(rippleFadeOutIntervalTween))
	controller.AddInkFeature(r)
	return r
}

// Confirm makes the ripple expand to its full size and fade out.
//
// Flutter: material/ink_ripple.dart InkRipple.confirm
func (r *InkRipple) Confirm() {
	r.radiusController.Duration = foundation.Some(rippleRadiusDuration)
	r.radiusController.Forward()
	// This confirm may have been preceded by a cancel.
	r.fadeInController.Forward()
	r.fadeOutController.AnimateTo(1.0, animation.AnimateOpts{Duration: foundation.Some(rippleFadeOutDuration)})
}

// Cancel makes the ripple fade out quickly.
//
// Flutter: material/ink_ripple.dart InkRipple.cancel
func (r *InkRipple) Cancel() {
	r.fadeInController.Stop()
	// Watch out: setting fadeOutController's value to 1.0 will trigger a call to
	// handleAlphaStatusChanged which will dispose fadeOutController.
	fadeOutValue := 1.0 - r.fadeInController.Value()
	r.fadeOutController.SetValue(fadeOutValue)
	if fadeOutValue < 1.0 {
		r.fadeOutController.AnimateTo(1.0, animation.AnimateOpts{Duration: foundation.Some(rippleCancelDuration)})
	}
}

func (r *InkRipple) handleAlphaStatusChanged(status animation.AnimationStatus) {
	if status.IsCompleted() {
		r.Dispose()
	}
}

// Dispose releases the animations and removes the ripple.
//
// Flutter: material/ink_ripple.dart InkRipple.dispose
func (r *InkRipple) Dispose() {
	r.radiusController.Dispose()
	r.fadeInController.Dispose()
	r.fadeOutController.Dispose()
	r.InkFeatureBase.Dispose()
}

// PaintFeature paints the ripple.
func (r *InkRipple) PaintFeature(canvas *ui.Canvas, transform *ui.Matrix4) {
	alpha := r.fadeOut.Value()
	if r.fadeInController.IsAnimating() {
		alpha = r.fadeIn.Value()
	}
	paint := ui.NewPaint()
	paint.SetColor(r.color.WithAlpha(alpha))
	var rectCenter ui.Offset
	if r.clipCallback != nil {
		rectCenter = r.clipCallback().Center()
	} else {
		rectCenter = r.referenceBox.Size().Center(ui.OffsetZero)
	}
	// Splash moves to the center of the reference box.
	center := r.position.Lerp(rectCenter, animation.Curves.Ease.Transform(r.radiusController.Value()))
	PaintInkCircle(canvas, InkCircle{Transform: transform, Paint: paint, Center: center, TextDirection: r.textDirection, Radius: r.radius.Value(),
		CustomBorder: r.customBorder, BorderRadius: r.borderRadius, ClipCallback: r.clipCallback})
}
