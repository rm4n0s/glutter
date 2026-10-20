// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// FloatingActionButtonMargin is the margin between a floating action button
// and the edges of the scaffold.
//
// Flutter: material/floating_action_button_location.dart kFloatingActionButtonMargin
const FloatingActionButtonMargin = 16.0

// FloatingActionButtonSegue is the amount of time the floating action button
// takes to transition in or out: half of the time of a move to another location.
//
// Flutter: material/floating_action_button_location.dart kFloatingActionButtonSegue
const FloatingActionButtonSegue = 200 * time.Millisecond

// FloatingActionButtonTurnInterval is the fraction of a full turn that the
// floating action button turns when it enters or leaves.
//
// Flutter: material/floating_action_button_location.dart kFloatingActionButtonTurnInterval
const FloatingActionButtonTurnInterval = 0.125

// MiniButtonOffsetAdjustment is how much the mini locations move the button
// towards the edge of the screen, so that its center is as far from the edge as
// the center of a regular button.
//
// Flutter: material/floating_action_button_location.dart kMiniButtonOffsetAdjustment
const MiniButtonOffsetAdjustment = 4.0

// FloatingActionButtonLocation places the floating action button in a scaffold:
// where the button goes within the scaffold, at the end of the layout.
//
// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation
type FloatingActionButtonLocation interface {
	// GetOffset is the position of the top left of the button for the geometry.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.getOffset
	GetOffset(geometry ScaffoldPrelayoutGeometry) ui.Offset
}

// StandardFabLocation is a location that gives the horizontal and the vertical
// offsets separately, adjusted for the mini buttons. The offsets of the
// locations of Flutter are the FabXxxOffsetX and FabXxxOffsetY functions.
//
// Flutter: material/floating_action_button_location.dart StandardFabLocation
type StandardFabLocation interface {
	FloatingActionButtonLocation
	// GetOffsetX is the horizontal offset of the button.
	//
	// Flutter: material/floating_action_button_location.dart StandardFabLocation.getOffsetX
	GetOffsetX(geometry ScaffoldPrelayoutGeometry, adjustment float64) float64
	// GetOffsetY is the vertical offset of the button.
	//
	// Flutter: material/floating_action_button_location.dart StandardFabLocation.getOffsetY
	GetOffsetY(geometry ScaffoldPrelayoutGeometry, adjustment float64) float64
	// IsMini says whether the location is for mini buttons: the adjustment moves
	// them towards the edge.
	//
	// Flutter: material/floating_action_button_location.dart StandardFabLocation.isMini
	IsMini() bool
}

// StandardFabLocationOffset is the offset of a standard location for the
// geometry: its horizontal and vertical offsets with the adjustment of a mini
// location.
//
// Flutter: material/floating_action_button_location.dart StandardFabLocation.getOffset
func StandardFabLocationOffset(l StandardFabLocation, geometry ScaffoldPrelayoutGeometry) ui.Offset {
	adjustment := 0.0
	if l.IsMini() {
		adjustment = MiniButtonOffsetAdjustment
	}
	return ui.Offset{Dx: l.GetOffsetX(geometry, adjustment), Dy: l.GetOffsetY(geometry, adjustment)}
}

func fabLeftOffsetX(g ScaffoldPrelayoutGeometry, adjustment float64) float64 {
	return FloatingActionButtonMargin + g.MinInsets.Left - adjustment
}

func fabRightOffsetX(g ScaffoldPrelayoutGeometry, adjustment float64) float64 {
	return g.ScaffoldSize.Width - FloatingActionButtonMargin - g.MinInsets.Right - g.FloatingActionButtonSize.Width + adjustment
}

// FabTopOffsetY is the vertical offset of the locations at the top: below the
// top of the content, centered on its edge when there is an app bar.
//
// Flutter: material/floating_action_button_location.dart FabTopOffsetY
func FabTopOffsetY(g ScaffoldPrelayoutGeometry, _ float64) float64 {
	if g.ContentTop > g.MinViewPadding.Top {
		fabHalfHeight := g.FloatingActionButtonSize.Height / 2.0
		return g.ContentTop - fabHalfHeight
	}
	// Otherwise, ensure we are placed within the bounds of a safe area.
	return g.MinViewPadding.Top
}

// FabFloatOffsetY is the vertical offset of the locations that float above the
// bottom of the content, over the snack bar and the bottom sheet.
//
// Flutter: material/floating_action_button_location.dart FabFloatOffsetY
func FabFloatOffsetY(g ScaffoldPrelayoutGeometry, adjustment float64) float64 {
	contentBottom := g.ContentBottom
	bottomContentHeight := g.ScaffoldSize.Height - contentBottom
	bottomSheetHeight := g.BottomSheetSize.Height
	fabHeight := g.FloatingActionButtonSize.Height
	snackBarHeight := g.SnackBarSize.Height
	safeMargin := math.Max(FloatingActionButtonMargin, g.MinViewPadding.Bottom-bottomContentHeight+FloatingActionButtonMargin)

	fabY := contentBottom - fabHeight - safeMargin
	if snackBarHeight > 0.0 {
		fabY = math.Min(fabY, contentBottom-snackBarHeight-fabHeight-FloatingActionButtonMargin)
	}
	if bottomSheetHeight > 0.0 {
		fabY = math.Min(fabY, contentBottom-bottomSheetHeight-fabHeight/2.0)
	}
	return fabY + adjustment
}

// FabDockedOffsetY is the vertical offset of the locations docked on the top
// edge of the bottom bar.
//
// Flutter: material/floating_action_button_location.dart FabDockedOffsetY
func FabDockedOffsetY(g ScaffoldPrelayoutGeometry, _ float64) float64 {
	contentBottom := g.ContentBottom
	contentMargin := g.ScaffoldSize.Height - contentBottom
	bottomViewPadding := g.MinViewPadding.Bottom
	bottomSheetHeight := g.BottomSheetSize.Height
	fabHeight := g.FloatingActionButtonSize.Height
	snackBarHeight := g.SnackBarSize.Height
	bottomMinInset := g.MinInsets.Bottom

	var safeMargin float64
	switch {
	case contentMargin > bottomMinInset+fabHeight/2.0:
		// If contentMargin is higher than bottomMinInset enough to display the FAB
		// without clipping, don't provide a margin.
		safeMargin = 0.0
	case bottomMinInset == 0.0:
		// If bottomMinInset is zero(the software keyboard is not on the screen)
		// provide bottomViewPadding as margin.
		safeMargin = bottomViewPadding
	default:
		// Provide a margin that would shift the FAB enough so that it stays away
		// from the keyboard.
		safeMargin = fabHeight/2.0 + FloatingActionButtonMargin
	}

	// This is to compute the docked location for the fab.
	fabY := contentBottom - fabHeight/2.0 - safeMargin
	// The FAB should sit with a margin between it and the snack bar.
	if snackBarHeight > 0.0 {
		fabY = math.Min(fabY, contentBottom-snackBarHeight-fabHeight-FloatingActionButtonMargin)
	}
	// The FAB should sit with its center in front of the top of the bottom sheet.
	if bottomSheetHeight > 0.0 {
		fabY = math.Min(fabY, contentBottom-bottomSheetHeight-fabHeight/2.0)
	}
	maxFabY := g.ScaffoldSize.Height - fabHeight - safeMargin
	return math.Min(maxFabY, fabY)
}

// FabContainedOffsetY is the vertical offset of the locations contained in the
// bottom bar: centered in the space below the content.
//
// Flutter: material/floating_action_button_location.dart FabContainedOffsetY
func FabContainedOffsetY(g ScaffoldPrelayoutGeometry, _ float64) float64 {
	contentBottom := g.ContentBottom
	contentMargin := g.ScaffoldSize.Height - contentBottom
	bottomViewPadding := g.MinViewPadding.Bottom
	fabHeight := g.FloatingActionButtonSize.Height

	var safeMargin float64
	if contentMargin > bottomViewPadding+fabHeight {
		// If contentMargin is higher than bottomViewPadding enough to display the
		// FAB without clipping, don't provide a margin.
		safeMargin = 0.0
	} else {
		safeMargin = bottomViewPadding
	}

	// This is to compute the position for the contained location for the fab.
	contentBottomToFabTop := (contentMargin - bottomViewPadding - fabHeight) / 2.0
	fabY := contentBottom + contentBottomToFabTop
	maxFabY := g.ScaffoldSize.Height - fabHeight - safeMargin
	return math.Min(maxFabY, fabY)
}

// FabStartOffsetX is the horizontal offset of the locations at the start.
//
// Flutter: material/floating_action_button_location.dart FabStartOffsetX
func FabStartOffsetX(g ScaffoldPrelayoutGeometry, adjustment float64) float64 {
	if g.TextDirection == ui.TextDirectionRTL {
		return fabRightOffsetX(g, adjustment)
	}
	return fabLeftOffsetX(g, adjustment)
}

// FabCenterOffsetX is the horizontal offset of the locations at the center.
//
// Flutter: material/floating_action_button_location.dart FabCenterOffsetX
func FabCenterOffsetX(g ScaffoldPrelayoutGeometry, _ float64) float64 {
	return (g.ScaffoldSize.Width - g.FloatingActionButtonSize.Width) / 2.0
}

// FabEndOffsetX is the horizontal offset of the locations at the end.
//
// Flutter: material/floating_action_button_location.dart FabEndOffsetX
func FabEndOffsetX(g ScaffoldPrelayoutGeometry, adjustment float64) float64 {
	if g.TextDirection == ui.TextDirectionRTL {
		return fabLeftOffsetX(g, adjustment)
	}
	return fabRightOffsetX(g, adjustment)
}

// fabLocation is a standard location made of an horizontal and a vertical offset.
type fabLocation struct {
	name string
	mini bool
	x, y func(ScaffoldPrelayoutGeometry, float64) float64
}

func (l *fabLocation) GetOffsetX(g ScaffoldPrelayoutGeometry, adjustment float64) float64 {
	return l.x(g, adjustment)
}

func (l *fabLocation) GetOffsetY(g ScaffoldPrelayoutGeometry, adjustment float64) float64 {
	return l.y(g, adjustment)
}

func (l *fabLocation) IsMini() bool { return l.mini }

func (l *fabLocation) GetOffset(g ScaffoldPrelayoutGeometry) ui.Offset {
	return StandardFabLocationOffset(l, g)
}

func (l *fabLocation) String() string { return "FloatingActionButtonLocation." + l.name }

// The locations of a floating action button. The ones with Top are at the top of
// the screen, with Float they float above the bottom widgets, with Docked they
// are docked on the top edge of a bottom bar and the contained one is within it.
// The Mini ones are for the small buttons: they are moved 4 pixels towards the
// edge.
var (
	// FloatingActionButtonLocationStartTop is at the start, at the top.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.startTop
	FloatingActionButtonLocationStartTop FloatingActionButtonLocation = &fabLocation{name: "startTop", mini: false, x: FabStartOffsetX, y: FabTopOffsetY}
	// FloatingActionButtonLocationMiniStartTop is at the start, at the top, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniStartTop
	FloatingActionButtonLocationMiniStartTop FloatingActionButtonLocation = &fabLocation{name: "miniStartTop", mini: true, x: FabStartOffsetX, y: FabTopOffsetY}
	// FloatingActionButtonLocationCenterTop is at the center, at the top.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.centerTop
	FloatingActionButtonLocationCenterTop FloatingActionButtonLocation = &fabLocation{name: "centerTop", mini: false, x: FabCenterOffsetX, y: FabTopOffsetY}
	// FloatingActionButtonLocationMiniCenterTop is at the center, at the top, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniCenterTop
	FloatingActionButtonLocationMiniCenterTop FloatingActionButtonLocation = &fabLocation{name: "miniCenterTop", mini: true, x: FabCenterOffsetX, y: FabTopOffsetY}
	// FloatingActionButtonLocationEndTop is at the end, at the top.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.endTop
	FloatingActionButtonLocationEndTop FloatingActionButtonLocation = &fabLocation{name: "endTop", mini: false, x: FabEndOffsetX, y: FabTopOffsetY}
	// FloatingActionButtonLocationMiniEndTop is at the end, at the top, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniEndTop
	FloatingActionButtonLocationMiniEndTop FloatingActionButtonLocation = &fabLocation{name: "miniEndTop", mini: true, x: FabEndOffsetX, y: FabTopOffsetY}
	// FloatingActionButtonLocationStartFloat is at the start, floating above the bottom widgets.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.startFloat
	FloatingActionButtonLocationStartFloat FloatingActionButtonLocation = &fabLocation{name: "startFloat", mini: false, x: FabStartOffsetX, y: FabFloatOffsetY}
	// FloatingActionButtonLocationMiniStartFloat is at the start, floating above the bottom widgets, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniStartFloat
	FloatingActionButtonLocationMiniStartFloat FloatingActionButtonLocation = &fabLocation{name: "miniStartFloat", mini: true, x: FabStartOffsetX, y: FabFloatOffsetY}
	// FloatingActionButtonLocationCenterFloat is at the center, floating above the bottom widgets.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.centerFloat
	FloatingActionButtonLocationCenterFloat FloatingActionButtonLocation = &fabLocation{name: "centerFloat", mini: false, x: FabCenterOffsetX, y: FabFloatOffsetY}
	// FloatingActionButtonLocationMiniCenterFloat is at the center, floating above the bottom widgets, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniCenterFloat
	FloatingActionButtonLocationMiniCenterFloat FloatingActionButtonLocation = &fabLocation{name: "miniCenterFloat", mini: true, x: FabCenterOffsetX, y: FabFloatOffsetY}
	// FloatingActionButtonLocationEndFloat is at the end, floating above the bottom widgets.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.endFloat
	FloatingActionButtonLocationEndFloat FloatingActionButtonLocation = &fabLocation{name: "endFloat", mini: false, x: FabEndOffsetX, y: FabFloatOffsetY}
	// FloatingActionButtonLocationMiniEndFloat is at the end, floating above the bottom widgets, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniEndFloat
	FloatingActionButtonLocationMiniEndFloat FloatingActionButtonLocation = &fabLocation{name: "miniEndFloat", mini: true, x: FabEndOffsetX, y: FabFloatOffsetY}
	// FloatingActionButtonLocationStartDocked is at the start, docked on the bottom bar.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.startDocked
	FloatingActionButtonLocationStartDocked FloatingActionButtonLocation = &fabLocation{name: "startDocked", mini: false, x: FabStartOffsetX, y: FabDockedOffsetY}
	// FloatingActionButtonLocationMiniStartDocked is at the start, docked on the bottom bar, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniStartDocked
	FloatingActionButtonLocationMiniStartDocked FloatingActionButtonLocation = &fabLocation{name: "miniStartDocked", mini: true, x: FabStartOffsetX, y: FabDockedOffsetY}
	// FloatingActionButtonLocationCenterDocked is at the center, docked on the bottom bar.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.centerDocked
	FloatingActionButtonLocationCenterDocked FloatingActionButtonLocation = &fabLocation{name: "centerDocked", mini: false, x: FabCenterOffsetX, y: FabDockedOffsetY}
	// FloatingActionButtonLocationMiniCenterDocked is at the center, docked on the bottom bar, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniCenterDocked
	FloatingActionButtonLocationMiniCenterDocked FloatingActionButtonLocation = &fabLocation{name: "miniCenterDocked", mini: true, x: FabCenterOffsetX, y: FabDockedOffsetY}
	// FloatingActionButtonLocationEndDocked is at the end, docked on the bottom bar.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.endDocked
	FloatingActionButtonLocationEndDocked FloatingActionButtonLocation = &fabLocation{name: "endDocked", mini: false, x: FabEndOffsetX, y: FabDockedOffsetY}
	// FloatingActionButtonLocationMiniEndDocked is at the end, docked on the bottom bar, for mini buttons.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.miniEndDocked
	FloatingActionButtonLocationMiniEndDocked FloatingActionButtonLocation = &fabLocation{name: "miniEndDocked", mini: true, x: FabEndOffsetX, y: FabDockedOffsetY}
	// FloatingActionButtonLocationEndContained is at the end, contained in the bottom bar.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.endContained
	FloatingActionButtonLocationEndContained FloatingActionButtonLocation = &fabLocation{name: "endContained", x: FabEndOffsetX, y: FabContainedOffsetY}
)

// fabLocationIsTop says whether the location is one of those at the top, where a
// snack bar does not sit above the button.
func fabLocationIsTop(l FloatingActionButtonLocation) bool {
	switch l {
	case FloatingActionButtonLocationStartTop, FloatingActionButtonLocationCenterTop, FloatingActionButtonLocationEndTop,
		FloatingActionButtonLocationMiniStartTop, FloatingActionButtonLocationMiniCenterTop, FloatingActionButtonLocationMiniEndTop:
		return true
	}
	return false
}

// fabTransitionSnapshotLocation is the location in the middle of a move to
// another location: the animator mixes the two offsets.
//
// Flutter: material/scaffold.dart _TransitionSnapshotFabLocation
type fabTransitionSnapshotLocation struct {
	begin, end FloatingActionButtonLocation
	animator   FloatingActionButtonAnimator
	progress   float64
}

func (l *fabTransitionSnapshotLocation) GetOffset(g ScaffoldPrelayoutGeometry) ui.Offset {
	return l.animator.GetOffset(l.begin.GetOffset(g), l.end.GetOffset(g), l.progress)
}

func (l *fabTransitionSnapshotLocation) String() string {
	return fmt.Sprintf("_TransitionSnapshotFabLocation(begin: %v, end: %v, progress: %v)", l.begin, l.end, l.progress)
}

// FloatingActionButtonAnimator animates the floating action button when it
// moves to another location, or when it enters or leaves: the offset of the
// move, and the scale and the rotation of the button.
//
// Flutter: material/floating_action_button_location.dart FloatingActionButtonAnimator
type FloatingActionButtonAnimator interface {
	// GetOffset is the offset of the button at the progress of the move.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonAnimator.getOffset
	GetOffset(begin, end ui.Offset, progress float64) ui.Offset
	// GetScaleAnimation is the scale of the button for the move animation.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonAnimator.getScaleAnimation
	GetScaleAnimation(parent animation.Animation[float64]) animation.Animation[float64]
	// GetRotationAnimation is the rotation, in turns, of the button for the move animation.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonAnimator.getRotationAnimation
	GetRotationAnimation(parent animation.Animation[float64]) animation.Animation[float64]
	// GetAnimationRestart is the value to restart a move animation from, when a
	// move starts in the middle of another one.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonAnimator.getAnimationRestart
	GetAnimationRestart(previousValue float64) float64
}

type scalingFabMotionAnimator struct{}

// FloatingActionButtonAnimatorScaling scales the button down at the old location
// and up at the new one, with a turn, when it moves. It is the default.
//
// Flutter: material/floating_action_button_location.dart FloatingActionButtonAnimator.scaling
var FloatingActionButtonAnimatorScaling FloatingActionButtonAnimator = scalingFabMotionAnimator{}

func (scalingFabMotionAnimator) GetOffset(begin, end ui.Offset, progress float64) ui.Offset {
	if progress < 0.5 {
		return begin
	}
	return end
}

// fabAnimationSwap is an animation of the first until the parent reaches the
// threshold, and of the next after.
//
// Flutter: material/floating_action_button_location.dart _AnimationSwap
func fabAnimationSwap(first, next, parent animation.Animation[float64], swapThreshold float64) animation.Animation[float64] {
	return animation.NewCompoundAnimation("_AnimationSwap", first, next, func(f, n float64) float64 {
		if parent.Value() < swapThreshold {
			return f
		}
		return n
	})
}

func (scalingFabMotionAnimator) GetScaleAnimation(parent animation.Animation[float64]) animation.Animation[float64] {
	curve := animation.NewInterval(0.5, 1.0, animation.Curves.Ease)
	return fabAnimationSwap(
		animation.NewReverseAnimation(animation.Drive[float64](parent, animation.NewCurveTween(curve.Flipped()))),
		animation.Drive[float64](parent, animation.NewCurveTween(curve)),
		parent, 0.5)
}

func (scalingFabMotionAnimator) GetRotationAnimation(parent animation.Animation[float64]) animation.Animation[float64] {
	rotationTween := animation.NewTween(1.0-FloatingActionButtonTurnInterval*2.0, 1.0)
	thresholdCenterTween := animation.NewCurveTween(animation.NewThreshold(0.5))
	return fabAnimationSwap(
		animation.Drive[float64](parent, rotationTween),
		animation.NewReverseAnimation(animation.Drive[float64](parent, thresholdCenterTween)),
		parent, 0.5)
}

func (scalingFabMotionAnimator) GetAnimationRestart(previousValue float64) float64 {
	return math.Min(1.0-previousValue, previousValue)
}

func (scalingFabMotionAnimator) String() string { return "FloatingActionButtonAnimator.scaling" }

type noAnimationFabMotionAnimator struct{}

// FloatingActionButtonAnimatorNoAnimation moves the button to the new location
// at once, without animation.
//
// Flutter: material/floating_action_button_location.dart FloatingActionButtonAnimator.noAnimation
var FloatingActionButtonAnimatorNoAnimation FloatingActionButtonAnimator = noAnimationFabMotionAnimator{}

func (noAnimationFabMotionAnimator) GetOffset(_, end ui.Offset, _ float64) ui.Offset { return end }

func (noAnimationFabMotionAnimator) GetRotationAnimation(animation.Animation[float64]) animation.Animation[float64] {
	return animation.NewAlwaysStoppedAnimation(1.0)
}

func (noAnimationFabMotionAnimator) GetScaleAnimation(animation.Animation[float64]) animation.Animation[float64] {
	return animation.NewAlwaysStoppedAnimation(1.0)
}

func (noAnimationFabMotionAnimator) GetAnimationRestart(float64) float64 { return 0.0 }

func (noAnimationFabMotionAnimator) String() string {
	return "FloatingActionButtonAnimator.noAnimation"
}

// floatingActionButtonTransition animates the floating action button of a
// scaffold when it changes or moves: it scales and turns the old button out and
// the new one in.
//
// Flutter: material/scaffold.dart _FloatingActionButtonTransition
type floatingActionButtonTransition struct {
	widgets.StatefulWidget
	child             widgets.Widget
	fabMoveAnimation  animation.Animation[float64]
	fabMotionAnimator FloatingActionButtonAnimator
	geometryNotifier  *scaffoldGeometryNotifier
	currentController *animation.AnimationController
}

func newFloatingActionButtonTransition(child widgets.Widget, fabMoveAnimation animation.Animation[float64], fabMotionAnimator FloatingActionButtonAnimator,
	geometryNotifier *scaffoldGeometryNotifier, currentController *animation.AnimationController) *floatingActionButtonTransition {
	return &floatingActionButtonTransition{StatefulWidget: widgets.NewStatefulWidgetBase(), child: child, fabMoveAnimation: fabMoveAnimation,
		fabMotionAnimator: fabMotionAnimator, geometryNotifier: geometryNotifier, currentController: currentController}
}

func (w *floatingActionButtonTransition) CreateState() widgets.State {
	return &floatingActionButtonTransitionState{}
}

type floatingActionButtonTransitionState struct {
	widgets.StateOf[*floatingActionButtonTransition]
	widgets.TickerProvider
	previousController            *animation.AnimationController
	previousExitScaleAnimation    *animation.CurvedAnimation
	previousExitRotationCurved    *animation.CurvedAnimation
	currentEntranceScaleAnimation *animation.CurvedAnimation
	previousScaleAnimation        animation.Animation[float64]
	previousRotationAnimation     *animation.TrainHoppingAnimation
	currentScaleAnimation         animation.Animation[float64]
	extendedCurrentScaleAnimation animation.Animation[float64]
	currentRotationAnimation      *animation.TrainHoppingAnimation
	previousChild                 widgets.Widget
	previousScaleHandle           foundation.ListenerHandle
	currentScaleHandle            foundation.ListenerHandle
	hasHandles                    bool
}

func (s *floatingActionButtonTransitionState) InitState() {
	s.previousController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(FloatingActionButtonSegue)})
	s.previousController.AddStatusListener(s.handlePreviousAnimationStatusChanged)
	s.updateAnimations()
	w := s.Widget()
	if w.child != nil {
		w.currentController.SetValue(1.0)
		s.updateGeometryScale(1.0)
	} else {
		s.updateGeometryScale(0.0)
	}
}

func (s *floatingActionButtonTransitionState) Dispose() {
	s.previousController.Dispose()
	if s.previousExitScaleAnimation != nil {
		s.previousExitScaleAnimation.Dispose()
	}
	if s.previousExitRotationCurved != nil {
		s.previousExitRotationCurved.Dispose()
	}
	if s.currentEntranceScaleAnimation != nil {
		s.currentEntranceScaleAnimation.Dispose()
	}
	s.disposeAnimations()
}

func (s *floatingActionButtonTransitionState) DidUpdateWidget(old *floatingActionButtonTransition) {
	w := s.Widget()
	if old.fabMotionAnimator != w.fabMotionAnimator || old.fabMoveAnimation != w.fabMoveAnimation {
		// Get the right scale and rotation animations to use for the new animator.
		s.disposeAnimations()
		s.updateAnimations()
	}
	oldChildIsNull, newChildIsNull := old.child == nil, w.child == nil
	if oldChildIsNull == newChildIsNull && (oldChildIsNull || old.child.Key() == w.child.Key()) {
		return
	}
	if s.previousController.Status().IsDismissed() {
		currentValue := w.currentController.Value()
		if currentValue == 0.0 || old.child == nil {
			// The simplest case: we're either starting with no previous child or
			// have just completed the exit of the previous child. Start entering.
			s.previousChild = nil
			if w.child != nil {
				w.currentController.Forward()
			}
		} else {
			s.previousChild = old.child
			s.previousController.SetValue(currentValue)
			s.previousController.Reverse()
			w.currentController.SetValue(0.0)
		}
	}
}

func (s *floatingActionButtonTransitionState) disposeAnimations() {
	if s.hasHandles {
		s.currentScaleAnimation.RemoveListener(s.currentScaleHandle)
		s.previousScaleAnimation.RemoveListener(s.previousScaleHandle)
		s.hasHandles = false
	}
	s.previousRotationAnimation.Dispose()
	s.currentRotationAnimation.Dispose()
}

func (s *floatingActionButtonTransitionState) updateAnimations() {
	w := s.Widget()
	if s.previousExitScaleAnimation != nil {
		s.previousExitScaleAnimation.Dispose()
	}
	s.previousExitScaleAnimation = animation.NewCurvedAnimation(s.previousController, animation.Curves.EaseIn)
	if s.previousExitRotationCurved != nil {
		s.previousExitRotationCurved.Dispose()
	}
	s.previousExitRotationCurved = animation.NewCurvedAnimation(s.previousController, animation.Curves.EaseIn)
	previousExitRotationAnimation := animation.Drive[float64](s.previousExitRotationCurved, animation.NewTween(1.0, 1.0))

	if s.currentEntranceScaleAnimation != nil {
		s.currentEntranceScaleAnimation.Dispose()
	}
	s.currentEntranceScaleAnimation = animation.NewCurvedAnimation(w.currentController, animation.Curves.EaseIn)
	entranceTurnTween := animation.NewTween(1.0-FloatingActionButtonTurnInterval, 1.0).Chain(animation.NewCurveTween(animation.Curves.EaseIn))
	currentEntranceRotationAnimation := animation.Drive[float64](w.currentController, entranceTurnTween)

	moveScaleAnimation := w.fabMotionAnimator.GetScaleAnimation(w.fabMoveAnimation)
	moveRotationAnimation := w.fabMotionAnimator.GetRotationAnimation(w.fabMoveAnimation)

	if w.fabMotionAnimator == FloatingActionButtonAnimatorNoAnimation {
		s.previousScaleAnimation = moveScaleAnimation
		s.currentScaleAnimation = moveScaleAnimation
		s.previousRotationAnimation = animation.NewTrainHoppingAnimation(moveRotationAnimation, nil, nil)
		s.currentRotationAnimation = animation.NewTrainHoppingAnimation(moveRotationAnimation, nil, nil)
	} else {
		s.previousScaleAnimation = animation.NewAnimationMin[float64](moveScaleAnimation, s.previousExitScaleAnimation)
		s.currentScaleAnimation = animation.NewAnimationMin[float64](moveScaleAnimation, s.currentEntranceScaleAnimation)
		s.previousRotationAnimation = animation.NewTrainHoppingAnimation(previousExitRotationAnimation, moveRotationAnimation, nil)
		s.currentRotationAnimation = animation.NewTrainHoppingAnimation(currentEntranceRotationAnimation, moveRotationAnimation, nil)
	}
	s.extendedCurrentScaleAnimation = animation.Drive[float64](s.currentScaleAnimation, animation.NewCurveTween(animation.NewInterval(0.0, 0.1)))
	s.currentScaleHandle = s.currentScaleAnimation.AddListener(s.onProgressChanged)
	s.previousScaleHandle = s.previousScaleAnimation.AddListener(s.onProgressChanged)
	s.hasHandles = true
}

func (s *floatingActionButtonTransitionState) handlePreviousAnimationStatusChanged(status animation.AnimationStatus) {
	s.SetState(func() {
		if w := s.Widget(); w.child != nil && status.IsDismissed() {
			w.currentController.Forward()
		}
	})
}

func isExtendedFloatingActionButton(w widgets.Widget) bool {
	fab, ok := w.(*FloatingActionButton)
	return ok && fab.IsExtended()
}

func (s *floatingActionButtonTransitionState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var children []widgets.Widget
	if !s.previousController.Status().IsDismissed() {
		if isExtendedFloatingActionButton(s.previousChild) {
			children = append(children, widgets.NewFadeTransition(s.previousScaleAnimation, s.previousChild))
		} else {
			children = append(children, widgets.NewScaleTransition(s.previousScaleAnimation,
				widgets.NewRotationTransition(s.previousRotationAnimation, s.previousChild)))
		}
	}
	if isExtendedFloatingActionButton(w.child) {
		children = append(children, widgets.NewScaleTransition(s.extendedCurrentScaleAnimation,
			widgets.NewFadeTransition(s.currentScaleAnimation, w.child)))
	} else {
		children = append(children, widgets.NewScaleTransition(s.currentScaleAnimation,
			widgets.NewRotationTransition(s.currentRotationAnimation, w.child)))
	}
	return widgets.NewStack(children, widgets.StackOpts{Alignment: painting.AlignmentCenterRight})
}

func (s *floatingActionButtonTransitionState) onProgressChanged() {
	s.updateGeometryScale(math.Max(s.previousScaleAnimation.Value(), s.currentScaleAnimation.Value()))
}

func (s *floatingActionButtonTransitionState) updateGeometryScale(scale float64) {
	s.Widget().geometryNotifier.updateScale(scale)
}
