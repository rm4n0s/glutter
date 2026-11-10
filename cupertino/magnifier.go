package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// CupertinoMagnifierAboveFocalPoint is the vertical distance that the magnifier is above the focal point.
	//
	// Flutter: cupertino/magnifier.dart CupertinoMagnifier.kMagnifierAboveFocalPoint
	CupertinoMagnifierAboveFocalPoint = -26.0

	magnifierDragAnimationDuration  = 45 * time.Millisecond
	magnifierInOutAnimationDuration = 150 * time.Millisecond
)

// CupertinoMagnifierDefaultSize is the default size of the magnifier. It is read-only.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier.kDefaultSize
var CupertinoMagnifierDefaultSize = ui.Size{Width: 80, Height: 47.5}

// CupertinoTextMagnifierOpts are the optional parameters of NewCupertinoTextMagnifier.
//
// Flutter: cupertino/magnifier.dart CupertinoTextMagnifier.new
type CupertinoTextMagnifierOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AnimationCurve is the curve of the animations of the position and of the fade in and out; the default is
	// Curves.EaseOut.
	AnimationCurve animation.Curve
	// DragResistance is how much the magnifier resists a drag below the line: it moves 1/DragResistance of the
	// distance; the default is 10.
	DragResistance foundation.Opt[float64]
	// HideBelowThreshold is how far below the line the gesture can go before the magnifier hides; the default is 48.
	HideBelowThreshold foundation.Opt[float64]
	// HorizontalScreenEdgePadding is the padding between the magnifier and the sides of the screen; the default is
	// 10.
	HorizontalScreenEdgePadding foundation.Opt[float64]
}

// CupertinoTextMagnifier is the magnifier of iOS for text: it follows the gesture on the line, resists a drag
// below it, and hides when the gesture is too far below.
//
// Flutter: cupertino/magnifier.dart CupertinoTextMagnifier
type CupertinoTextMagnifier struct {
	widgets.StatefulWidget
	controller    *widgets.MagnifierController
	magnifierInfo *foundation.ValueNotifier[widgets.MagnifierInfo]
	curve         animation.Curve
	dragResist    float64
	hideBelow     float64
	edgePadding   float64
}

// NewCupertinoTextMagnifier creates the magnifier. The controller is the one that shows it, and the magnifierInfo
// must be updated with the values of the gesture.
//
// Flutter: cupertino/magnifier.dart CupertinoTextMagnifier.new
func NewCupertinoTextMagnifier(controller *widgets.MagnifierController, magnifierInfo *foundation.ValueNotifier[widgets.MagnifierInfo], opts ...CupertinoTextMagnifierOpts) *CupertinoTextMagnifier {
	o := firstOf(opts)
	curve := o.AnimationCurve
	if curve == nil {
		curve = animation.Curves.EaseOut
	}
	return &CupertinoTextMagnifier{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), controller: controller, magnifierInfo: magnifierInfo,
		curve: curve, dragResist: o.DragResistance.OrElse(10.0), hideBelow: o.HideBelowThreshold.OrElse(48.0), edgePadding: o.HorizontalScreenEdgePadding.OrElse(10.0)}
}

// Controller is the controller that shows and hides the magnifier.
//
// Flutter: cupertino/magnifier.dart CupertinoTextMagnifier.controller
func (w *CupertinoTextMagnifier) Controller() *widgets.MagnifierController { return w.controller }

// MagnifierInfo is the status of the gesture of the user.
//
// Flutter: cupertino/magnifier.dart CupertinoTextMagnifier.magnifierInfo
func (w *CupertinoTextMagnifier) MagnifierInfo() *foundation.ValueNotifier[widgets.MagnifierInfo] {
	return w.magnifierInfo
}

// CreateState creates the state.
//
// Flutter: cupertino/magnifier.dart CupertinoTextMagnifier.createState
func (w *CupertinoTextMagnifier) CreateState() widgets.State { return &cupertinoTextMagnifierState{} }

type cupertinoTextMagnifierState struct {
	widgets.StateOf[*CupertinoTextMagnifier]
	widgets.SingleTickerProvider
	currentAdjustedMagnifierPosition ui.Offset
	verticalFocalPointAdjustment     float64
	ioAnimationController            *animation.AnimationController
	ioAnimation                      animation.Animation[float64]
	ioCurvedAnimation                *animation.CurvedAnimation
	ioHandle, infoHandle             foundation.ListenerHandle
}

func (s *cupertinoTextMagnifierState) InitState() {
	w := s.Widget()
	s.ioAnimationController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Value: foundation.Some(0.0), Duration: foundation.Some(magnifierInOutAnimationDuration)})
	s.ioHandle = s.ioAnimationController.AddListener(func() { s.SetState(func() {}) })
	w.controller.SetAnimationController(s.ioAnimationController)
	s.infoHandle = w.magnifierInfo.AddListener(s.determineMagnifierPositionAndFocalPoint)
	s.ioCurvedAnimation = animation.NewCurvedAnimation(s.ioAnimationController, w.curve)
	s.ioAnimation = animation.NewTween(0.0, 1.0).Animate(s.ioCurvedAnimation)
}

func (s *cupertinoTextMagnifierState) Dispose() {
	w := s.Widget()
	w.controller.SetAnimationController(nil)
	s.ioAnimationController.RemoveListener(s.ioHandle)
	s.ioAnimationController.Dispose()
	s.ioCurvedAnimation.Dispose()
	w.magnifierInfo.RemoveListener(s.infoHandle)
}

func (s *cupertinoTextMagnifierState) DidUpdateWidget(old *CupertinoTextMagnifier) {
	if old.magnifierInfo != s.Widget().magnifierInfo {
		old.magnifierInfo.RemoveListener(s.infoHandle)
		s.infoHandle = s.Widget().magnifierInfo.AddListener(s.determineMagnifierPositionAndFocalPoint)
	}
}

func (s *cupertinoTextMagnifierState) DidChangeDependencies() {
	s.determineMagnifierPositionAndFocalPoint()
}

func (s *cupertinoTextMagnifierState) determineMagnifierPositionAndFocalPoint() {
	w := s.Widget()
	textEditingContext := w.magnifierInfo.Value()
	verticalCenterOfCurrentLine := textEditingContext.CaretRect.Center().Dy

	// The magnifier hides when the gesture is too far below the line.
	if verticalCenterOfCurrentLine-textEditingContext.GlobalGesturePosition.Dy < -w.hideBelow {
		if w.controller.Shown() {
			w.controller.Hide(widgets.MagnifierControllerHideOpts{KeepInOverlay: true})
		}
		return
	}
	if !w.controller.Shown() {
		s.ioAnimationController.Forward()
	}

	// Adjust the magnifier position such that it stays at the line when the drag goes below it, with resistance.
	verticalPositionOfLens := math.Max(verticalCenterOfCurrentLine,
		verticalCenterOfCurrentLine-(verticalCenterOfCurrentLine-textEditingContext.GlobalGesturePosition.Dy)/w.dragResist)

	rawMagnifierPosition := ui.Offset{
		Dx: textEditingContext.GlobalGesturePosition.Dx - CupertinoMagnifierDefaultSize.Width/2,
		Dy: verticalPositionOfLens - (CupertinoMagnifierDefaultSize.Height - CupertinoMagnifierAboveFocalPoint),
	}
	screenRect := ui.OffsetZero.And(widgets.MediaQuerySizeOf(s.Context()))
	adjustedMagnifierPosition := widgets.MagnifierControllerShiftWithinBounds(
		rawMagnifierPosition.And(CupertinoMagnifierDefaultSize),
		ui.NewRectFromLTRB(
			screenRect.Left+w.edgePadding,
			screenRect.Top-(CupertinoMagnifierDefaultSize.Height+CupertinoMagnifierAboveFocalPoint),
			screenRect.Right-w.edgePadding,
			screenRect.Bottom+(CupertinoMagnifierDefaultSize.Height+CupertinoMagnifierAboveFocalPoint),
		)).TopLeft()
	s.SetState(func() {
		s.currentAdjustedMagnifierPosition = adjustedMagnifierPosition
		s.verticalFocalPointAdjustment = verticalCenterOfCurrentLine - verticalPositionOfLens
	})
}

func (s *cupertinoTextMagnifierState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return widgets.NewAnimatedPositioned(magnifierDragAnimationDuration,
		NewCupertinoMagnifier(CupertinoMagnifierOpts{
			InOutAnimation:             s.ioAnimation,
			AdditionalFocalPointOffset: ui.Offset{Dy: s.verticalFocalPointAdjustment},
			BorderSide:                 foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(CupertinoDynamicColorResolve(CupertinoThemeOf(bc).PrimaryColor(), bc)), Width: foundation.Some(2.0)})),
		}),
		widgets.AnimatedPositionedOpts{Curve: w.curve, Left: foundation.Some(s.currentAdjustedMagnifierPosition.Dx), Top: foundation.Some(s.currentAdjustedMagnifierPosition.Dy)})
}

// CupertinoMagnifierOpts are the optional parameters of NewCupertinoMagnifier.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier.new
type CupertinoMagnifierOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Size is the size of the magnifier; the default is CupertinoMagnifierDefaultSize.
	Size foundation.Opt[ui.Size]
	// BorderRadius is the radius of the corners of the magnifier; the default is an ellipse of 60 by 50.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// AdditionalFocalPointOffset is added to the offset of the focal point.
	AdditionalFocalPointOffset ui.Offset
	// Shadows are the shadows of the magnifier; nil is the default one, and an empty slice is none.
	Shadows []painting.BoxShadow
	// ClipBehavior is whether and how the magnifier clips its shadows; the default is ClipNone.
	ClipBehavior ui.Clip
	// BorderSide is the border of the magnifier; the default is a blue line 2 wide.
	BorderSide foundation.Opt[painting.BorderSide]
	// InOutAnimation is the animation that fades and moves the magnifier in and out; none shows it at once.
	InOutAnimation animation.Animation[float64]
	// MagnificationScale is how much the magnifier zooms in; the default is 1, and it must be greater than zero.
	MagnificationScale foundation.Opt[float64]
}

// CupertinoMagnifier is a magnifying glass in the style of iOS.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier
type CupertinoMagnifier struct {
	widgets.StatelessWidget
	o CupertinoMagnifierOpts
}

// NewCupertinoMagnifier creates a magnifier. In debug builds it panics with the tag
// CupertinoMagnifierScaleInvalid when the magnification scale is not greater than zero.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier.new
func NewCupertinoMagnifier(opts ...CupertinoMagnifierOpts) *CupertinoMagnifier {
	o := firstOf(opts)
	if foundation.DebugMode && !(o.MagnificationScale.OrElse(1.0) > 0) {
		panic(errors.New(tagCupertinoMagnifierScaleInvalid, "The magnification scale should be greater than zero.", "magnificationScale", o.MagnificationScale.OrElse(1.0)))
	}
	return &CupertinoMagnifier{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Size is the size of the magnifier.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier.size
func (w *CupertinoMagnifier) Size() ui.Size { return w.o.Size.OrElse(CupertinoMagnifierDefaultSize) }

// MagnificationScale is how much the magnifier zooms in.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier.magnificationScale
func (w *CupertinoMagnifier) MagnificationScale() float64 { return w.o.MagnificationScale.OrElse(1.0) }

// BorderSide is the border of the magnifier.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier.borderSide
func (w *CupertinoMagnifier) BorderSide() painting.BorderSide {
	return w.o.BorderSide.OrElse(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(ui.NewColorFromARGB(255, 0, 124, 255)), Width: foundation.Some(2.0)}))
}

// Build builds the magnifier: a RawMagnifier that is moved by the in and out animation.
//
// Flutter: cupertino/magnifier.dart CupertinoMagnifier.build
func (w *CupertinoMagnifier) Build(widgets.BuildContext) widgets.Widget {
	o := w.o
	t := 1.0
	if o.InOutAnimation != nil {
		t = o.InOutAnimation.Value()
	}
	focalPointOffset := ui.Offset{Dy: CupertinoMagnifierDefaultSize.Height/2 - CupertinoMagnifierAboveFocalPoint}.Add(o.AdditionalFocalPointOffset)
	borderRadius := o.BorderRadius.OrElse(painting.NewBorderRadiusAll(ui.NewRadiusElliptical(60, 50)))
	shadows := o.Shadows
	if shadows == nil {
		shadows = []painting.BoxShadow{{Shadow: ui.NewShadow(ui.ShadowOpts{Color: ptrColor(ui.NewColorFromARGB(25, 0, 0, 0)), BlurRadius: 11}), SpreadRadius: 0.2, BlurStyle: ui.BlurStyleOuter}}
	}
	clip := o.ClipBehavior
	if clip == ui.ClipUnspecified {
		clip = ui.ClipNone
	}
	borderSide := w.BorderSide()
	return widgets.NewTransformTranslate(*ui.OffsetLerp(&ui.Offset{Dy: -CupertinoMagnifierAboveFocalPoint}, &ui.Offset{}, t),
		widgets.NewRawMagnifier(w.Size(), widgets.RawMagnifierOpts{
			FocalPointOffset: focalPointOffset,
			Decoration: widgets.NewMagnifierDecoration(widgets.MagnifierDecorationOpts{
				Opacity: foundation.Some(t),
				Shape:   painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: borderRadius, Side: borderSide}),
				Shadows: shadows,
			}),
			ClipBehavior:       clip,
			MagnificationScale: foundation.Some(w.MagnificationScale()),
		}))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoMagnifierScaleInvalid = "CupertinoMagnifierScaleInvalid"
