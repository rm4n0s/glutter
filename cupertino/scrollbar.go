package cupertino

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	scrollbarMinLength           = 36.0
	scrollbarMinOverscrollLength = 8.0
	scrollbarTimeToFade          = 1200 * time.Millisecond
	scrollbarFadeDuration        = 250 * time.Millisecond
	scrollbarResizeDuration      = 100 * time.Millisecond
	scrollbarMainAxisMargin      = 3.0
	scrollbarCrossAxisMargin     = 3.0
)

// The default thickness and radius of a CupertinoScrollbar, and the ones while it is dragged.
const (
	// CupertinoScrollbarDefaultThickness is the thickness of the thumb: 3.
	//
	// Flutter: cupertino/scrollbar.dart CupertinoScrollbar.defaultThickness
	CupertinoScrollbarDefaultThickness = 3.0
	// CupertinoScrollbarDefaultThicknessWhileDragging is the thickness of the thumb while it is dragged: 8.
	//
	// Flutter: cupertino/scrollbar.dart CupertinoScrollbar.defaultThicknessWhileDragging
	CupertinoScrollbarDefaultThicknessWhileDragging = 8.0
)

// scrollbarColor is the color of the thumb: a translucent black, and a translucent white in dark mode.
var scrollbarColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x59000000), ui.NewColor(0x80FFFFFF))

// CupertinoScrollbarOpts are the optional parameters of NewCupertinoScrollbar.
//
// Flutter: cupertino/scrollbar.dart CupertinoScrollbar.new
type CupertinoScrollbarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller is the scroll controller of the scrollable the scrollbar is for.
	Controller widgets.ScrollController
	// ThumbVisibility is whether the thumb is always visible; the default is false.
	ThumbVisibility foundation.Opt[bool]
	// Thickness is the thickness of the thumb; the default is CupertinoScrollbarDefaultThickness.
	Thickness foundation.Opt[float64]
	// ThicknessWhileDragging is the thickness of the thumb while it is dragged; the default is
	// CupertinoScrollbarDefaultThicknessWhileDragging.
	ThicknessWhileDragging foundation.Opt[float64]
	// Radius is the radius of the corners of the thumb; the default is 1.5.
	Radius foundation.Opt[ui.Radius]
	// RadiusWhileDragging is the radius of the corners of the thumb while it is dragged; the default is 4.
	RadiusWhileDragging foundation.Opt[ui.Radius]
	// NotificationPredicate says which scroll notifications update the scrollbar; the default is
	// DefaultScrollNotificationPredicate.
	NotificationPredicate widgets.ScrollNotificationPredicate
	// ScrollbarOrientation is where the scrollbar sits.
	ScrollbarOrientation widgets.ScrollbarOrientation
	// MainAxisMargin is the margin at the ends of the thumb; the default is 3.
	MainAxisMargin foundation.Opt[float64]
}

// CupertinoScrollbar is an iOS style scrollbar: a thin thumb that fades out when the scrolling stops, grows
// while it is dragged and gives a haptic tap at the start and at the end of a drag.
//
// Flutter: cupertino/scrollbar.dart CupertinoScrollbar
type CupertinoScrollbar struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     CupertinoScrollbarOpts
}

// NewCupertinoScrollbar creates a scrollbar around the child. In debug builds it panics with the tag
// CupertinoScrollbarThicknessInfinite when a thickness is infinite.
//
// Flutter: cupertino/scrollbar.dart CupertinoScrollbar.new
func NewCupertinoScrollbar(child widgets.Widget, opts ...CupertinoScrollbarOpts) *CupertinoScrollbar {
	o := firstOf(opts)
	if foundation.DebugMode && (o.Thickness.OrElse(CupertinoScrollbarDefaultThickness) >= infinity || o.ThicknessWhileDragging.OrElse(CupertinoScrollbarDefaultThicknessWhileDragging) >= infinity) {
		panic(errors.New(tagCupertinoScrollbarThicknessInfinite, "the thickness of a CupertinoScrollbar must be finite"))
	}
	return &CupertinoScrollbar{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child, o}
}

// Child is the widget below the scrollbar.
//
// Flutter: cupertino/scrollbar.dart CupertinoScrollbar.child
func (w *CupertinoScrollbar) Child() widgets.Widget { return w.child }

// Build builds the RawScrollbar that has the Cupertino look.
//
// Flutter: cupertino/scrollbar.dart CupertinoScrollbar.build
func (w *CupertinoScrollbar) Build(widgets.BuildContext) widgets.Widget {
	o := w.o
	thickness := o.Thickness.OrElse(CupertinoScrollbarDefaultThickness)
	return widgets.NewRawScrollbar(w.child, widgets.RawScrollbarOpts{
		Controller:            o.Controller,
		ThumbVisibility:       foundation.Some(o.ThumbVisibility.OrElse(false)),
		Thickness:             foundation.Some(thickness),
		Radius:                foundation.Some(o.Radius.OrElse(ui.NewRadiusCircular(1.5))),
		FadeDuration:          foundation.Some(scrollbarFadeDuration),
		TimeToFade:            foundation.Some(scrollbarTimeToFade),
		PressDuration:         100 * time.Millisecond,
		NotificationPredicate: o.NotificationPredicate,
		ScrollbarOrientation:  o.ScrollbarOrientation,
		MainAxisMargin:        o.MainAxisMargin.OrElse(scrollbarMainAxisMargin),
		Subclass: func(base *widgets.RawScrollbarState) widgets.RawScrollbarSubclass {
			return &cupertinoScrollbarState{base: base, thicknessWhileDragging: o.ThicknessWhileDragging.OrElse(CupertinoScrollbarDefaultThicknessWhileDragging),
				radiusWhileDragging: o.RadiusWhileDragging.OrElse(ui.NewRadiusCircular(4.0)), thickness: thickness,
				radius: o.Radius.OrElse(ui.NewRadiusCircular(1.5))}
		},
	})
}

// cupertinoScrollbarState is the part of _CupertinoScrollbarState that is not the RawScrollbarState it
// extends in Flutter.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState
type cupertinoScrollbarState struct {
	base                   *widgets.RawScrollbarState
	thicknessAnimation     *animation.AnimationController
	thickness              float64
	thicknessWhileDragging float64
	radius                 ui.Radius
	radiusWhileDragging    ui.Radius
	pressStartAxisPosition float64
}

func (s *cupertinoScrollbarState) currentThickness() float64 {
	return s.thickness + s.thicknessAnimation.Value()*(s.thicknessWhileDragging-s.thickness)
}

func (s *cupertinoScrollbarState) currentRadius() ui.Radius {
	return s.radius.Lerp(s.radiusWhileDragging, s.thicknessAnimation.Value())
}

// ShowScrollbar is the one of the base.
func (s *cupertinoScrollbarState) ShowScrollbar() bool { return s.base.ShowScrollbar() }

// EnableGestures is the one of the base.
func (s *cupertinoScrollbarState) EnableGestures() bool { return s.base.EnableGestures() }

// InitState creates the controller of the thickness animation.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState.initState
func (s *cupertinoScrollbarState) InitState() {
	s.thicknessAnimation = animation.NewAnimationController(s.base, animation.AnimationControllerOpts{Duration: foundation.Some(scrollbarResizeDuration)})
	s.thicknessAnimation.AddListener(func() { s.UpdateScrollbarPainter() })
}

// BeforeDidChangeDependencies has nothing to read.
func (s *cupertinoScrollbarState) BeforeDidChangeDependencies() {}

// UpdateScrollbarPainter gives the painter the color, the thickness, the radius and the margins.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState.updateScrollbarPainter
func (s *cupertinoScrollbarState) UpdateScrollbarPainter() {
	if s.thicknessAnimation == nil {
		return
	}
	bc := s.base.Context()
	p := s.base.ScrollbarPainter()
	p.SetColor(CupertinoDynamicColorResolve(scrollbarColor, bc))
	p.SetTextDirection(widgets.DirectionalityOf(bc))
	p.SetThickness(s.currentThickness())
	p.SetMainAxisMargin(s.base.Widget().Options().MainAxisMargin)
	p.SetCrossAxisMargin(scrollbarCrossAxisMargin)
	p.SetRadius(foundation.Some(s.currentRadius()))
	p.SetPadding(widgets.MediaQueryPaddingOf(bc))
	p.SetMinLength(scrollbarMinLength)
	p.SetMinOverscrollLength(scrollbarMinOverscrollLength)
	p.SetScrollbarOrientation(s.base.Widget().Options().ScrollbarOrientation)
}

// HandleThumbPressStart remembers where the press started.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState.handleThumbPressStart
func (s *cupertinoScrollbarState) HandleThumbPressStart(localPosition ui.Offset) {
	s.base.HandleThumbPressStart(localPosition)
	direction, ok := s.base.GetScrollbarDirection().Get()
	if !ok {
		return
	}
	s.pressStartAxisPosition = localPosition.Dy
	if direction == axisHorizontal {
		s.pressStartAxisPosition = localPosition.Dx
	}
}

// HandleThumbPress grows the thumb and taps.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState.handleThumbPress
func (s *cupertinoScrollbarState) HandleThumbPress() {
	if !s.base.GetScrollbarDirection().IsSome() {
		return
	}
	s.base.HandleThumbPress()
	haptic := widgets.WidgetsBindingOf(s.base.Context()).HapticFeedback()
	s.thicknessAnimation.Forward().Then(func() { haptic.MediumImpact() })
}

// HandleThumbPressEnd shrinks the thumb and taps if the thumb moved but the finger did not fling.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState.handleThumbPressEnd
func (s *cupertinoScrollbarState) HandleThumbPressEnd(localPosition ui.Offset, velocity gestures.Velocity) {
	direction, ok := s.base.GetScrollbarDirection().Get()
	if !ok {
		return
	}
	s.thicknessAnimation.Reverse()
	s.base.HandleThumbPressEnd(localPosition, velocity)
	axisPosition, axisVelocity := localPosition.Dy, velocity.PixelsPerSecond.Dy
	if direction == axisHorizontal {
		axisPosition, axisVelocity = localPosition.Dx, velocity.PixelsPerSecond.Dx
	}
	if axisPosition != s.pressStartAxisPosition && absFloat(axisVelocity) < 10 {
		widgets.WidgetsBindingOf(s.base.Context()).HapticFeedback().MediumImpact()
	}
}

// HandleTrackTapDown scrolls a page toward the tap, except on iOS, where the track is not a control.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState.handleTrackTapDown
func (s *cupertinoScrollbarState) HandleTrackTapDown(details gestures.TapDownDetails) {
	bc := s.base.Context()
	if widgets.ScrollConfigurationOf(bc).GetPlatform(bc) != foundation.TargetPlatformIOS {
		s.base.HandleTrackTapDown(details)
	}
}

// HandleHover is the one of the base.
func (s *cupertinoScrollbarState) HandleHover(event *gestures.PointerHoverEvent) {
	s.base.HandleHover(event)
}

// HandleHoverExit is the one of the base.
func (s *cupertinoScrollbarState) HandleHoverExit(event *gestures.PointerExitEvent) {
	s.base.HandleHoverExit(event)
}

// BeforeDispose disposes the thickness animation.
//
// Flutter: cupertino/scrollbar.dart _CupertinoScrollbarState.dispose
func (s *cupertinoScrollbarState) BeforeDispose() { s.thicknessAnimation.Dispose() }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoScrollbarThicknessInfinite = "CupertinoScrollbarThicknessInfinite"
)
