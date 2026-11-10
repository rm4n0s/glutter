package cupertino

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
	"time"
)

const (
	refreshActivityIndicatorRadius = 14.0
	refreshActivityIndicatorMargin = 16.0
)

// cupertinoSliverRefresh is the sliver that makes room for the indicator of a CupertinoSliverRefreshControl while the
// list is overscrolled or the refresh is going on.
//
// Flutter: cupertino/refresh.dart _CupertinoSliverRefresh
type cupertinoSliverRefresh struct {
	widgets.SingleChildRenderObjectWidget
	refreshIndicatorLayoutExtent float64
	hasLayoutExtent              bool
}

func newCupertinoSliverRefresh(extent float64, hasLayoutExtent bool, child widgets.Widget) *cupertinoSliverRefresh {
	return &cupertinoSliverRefresh{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child),
		refreshIndicatorLayoutExtent: extent, hasLayoutExtent: hasLayoutExtent}
}

func (w *cupertinoSliverRefresh) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderCupertinoSliverRefresh{refreshIndicatorExtent: w.refreshIndicatorLayoutExtent, hasLayoutExtent: w.hasLayoutExtent}
	rendering.InitRenderSliverSingleBoxAdapter(&r.RenderSliverSingleBoxAdapter, r, nil)
	return r
}

func (w *cupertinoSliverRefresh) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderCupertinoSliverRefresh)
	r.setRefreshIndicatorLayoutExtent(w.refreshIndicatorLayoutExtent)
	r.setHasLayoutExtent(w.hasLayoutExtent)
}

// renderCupertinoSliverRefresh is the render object of cupertinoSliverRefresh: a sliver whose child is as tall as the
// overscroll plus the extent that the refresh holds.
//
// Flutter: cupertino/refresh.dart _RenderCupertinoSliverRefresh
type renderCupertinoSliverRefresh struct {
	rendering.RenderSliverSingleBoxAdapter
	refreshIndicatorExtent float64
	hasLayoutExtent        bool
	// layoutExtentOffsetCompensation is the extent that the scroll offset was last corrected for.
	layoutExtentOffsetCompensation float64
}

func (r *renderCupertinoSliverRefresh) setRefreshIndicatorLayoutExtent(value float64) {
	if value == r.refreshIndicatorExtent {
		return
	}
	r.refreshIndicatorExtent = value
	r.MarkNeedsLayout()
}

func (r *renderCupertinoSliverRefresh) setHasLayoutExtent(value bool) {
	if value == r.hasLayoutExtent {
		return
	}
	r.hasLayoutExtent = value
	r.MarkNeedsLayout()
}

func (r *renderCupertinoSliverRefresh) PerformLayout() {
	c := r.SliverConstraints()
	if foundation.DebugMode && (c.AxisDirection != painting.AxisDirectionDown || c.GrowthDirection != rendering.GrowthDirectionForward) {
		panic(errors.New(tagCupertinoSliverRefreshDirection, "a CupertinoSliverRefreshControl scrolls down from its start", "axisDirection", c.AxisDirection.String()))
	}
	layoutExtent := 0.0
	if r.hasLayoutExtent {
		layoutExtent = r.refreshIndicatorExtent
	}
	// If the refresh is holding an extent, the first time that it is seen the scroll offset is moved by it, so that the list does not jump.
	if layoutExtent != r.layoutExtentOffsetCompensation {
		r.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(layoutExtent - r.layoutExtentOffsetCompensation)}))
		r.layoutExtentOffsetCompensation = layoutExtent
		return
	}
	active := c.Overlap < 0 || layoutExtent > 0
	overscrolledExtent := 0.0
	if c.Overlap < 0 {
		overscrolledExtent = math.Abs(c.Overlap)
	}
	child := r.Child()
	// The child is as tall as the overscroll and the held extent, however long it is laid out.
	child.Layout(c.AsBoxConstraints(rendering.AsBoxConstraintsOpts{MaxExtent: foundation.Some(layoutExtent + overscrolledExtent)}), rendering.LayoutOpts{ParentUsesSize: true})
	if !active {
		r.SetGeometry(rendering.SliverGeometryZero)
		return
	}
	childHeight := child.Size().Height
	r.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{
		ScrollExtent:   layoutExtent,
		PaintOrigin:    -overscrolledExtent - c.ScrollOffset,
		PaintExtent:    math.Max(math.Max(childHeight, layoutExtent)-c.ScrollOffset, 0),
		MaxPaintExtent: math.Max(math.Max(childHeight, layoutExtent)-c.ScrollOffset, 0),
		LayoutExtent:   foundation.Some(math.Max(layoutExtent-c.ScrollOffset, 0)),
	}))
}

func (r *renderCupertinoSliverRefresh) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	c := r.SliverConstraints()
	if child := r.Child(); c.Overlap < 0 || c.ScrollOffset+child.Size().Height > 0 {
		context.PaintChild(child, offset)
	}
}

// ApplyPaintTransform does nothing: the child is painted at the sliver's origin.
func (r *renderCupertinoSliverRefresh) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
}

// RefreshIndicatorMode is the state of the indicator of a CupertinoSliverRefreshControl.
//
// Flutter: cupertino/refresh.dart RefreshIndicatorMode
type RefreshIndicatorMode int

const (
	// RefreshIndicatorModeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	RefreshIndicatorModeUnspecified RefreshIndicatorMode = iota
	// RefreshIndicatorModeInactive is when the list is not being pulled and no refresh is going on.
	//
	// Flutter: cupertino/refresh.dart RefreshIndicatorMode.inactive
	RefreshIndicatorModeInactive
	// RefreshIndicatorModeDrag is while the list is pulled down but not far enough to refresh.
	//
	// Flutter: cupertino/refresh.dart RefreshIndicatorMode.drag
	RefreshIndicatorModeDrag
	// RefreshIndicatorModeArmed is when the list was pulled far enough and the refresh starts as it is let go.
	//
	// Flutter: cupertino/refresh.dart RefreshIndicatorMode.armed
	RefreshIndicatorModeArmed
	// RefreshIndicatorModeRefresh is while the refresh is going on.
	//
	// Flutter: cupertino/refresh.dart RefreshIndicatorMode.refresh
	RefreshIndicatorModeRefresh
	// RefreshIndicatorModeDone is when the refresh is done and the indicator goes away.
	//
	// Flutter: cupertino/refresh.dart RefreshIndicatorMode.done
	RefreshIndicatorModeDone
)

// String is the name of the mode, like Dart's enum toString.
//
// Flutter: cupertino/refresh.dart RefreshIndicatorMode
func (m RefreshIndicatorMode) String() string {
	switch m {
	case RefreshIndicatorModeInactive:
		return "RefreshIndicatorMode.inactive"
	case RefreshIndicatorModeDrag:
		return "RefreshIndicatorMode.drag"
	case RefreshIndicatorModeArmed:
		return "RefreshIndicatorMode.armed"
	case RefreshIndicatorModeRefresh:
		return "RefreshIndicatorMode.refresh"
	case RefreshIndicatorModeDone:
		return "RefreshIndicatorMode.done"
	}
	return "RefreshIndicatorMode.unspecified"
}

// RefreshControlIndicatorBuilder builds the indicator of a CupertinoSliverRefreshControl for its state, the extent that
// the list is pulled by, the distance that arms the refresh and the extent that the indicator keeps while it refreshes.
//
// Flutter: cupertino/refresh.dart RefreshControlIndicatorBuilder
type RefreshControlIndicatorBuilder func(bc widgets.BuildContext, refreshState RefreshIndicatorMode, pulledExtent, refreshTriggerPullDistance, refreshIndicatorExtent float64) widgets.Widget

// RefreshCallback is called when the user pulls far enough to refresh. The refresh is over when the channel it
// returns is closed (or sent to); a nil channel is a refresh that is already over.
//
// Flutter: cupertino/refresh.dart RefreshCallback
type RefreshCallback func() <-chan struct{}

const (
	refreshDefaultTriggerPullDistance = 100.0
	refreshDefaultIndicatorExtent     = 60.0
	// refreshInactiveResetOverscrollFraction is the part of the trigger distance under which a finished refresh becomes inactive.
	refreshInactiveResetOverscrollFraction = 0.1
)

// CupertinoSliverRefreshControlOpts are the optional parameters of NewCupertinoSliverRefreshControl.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl.new
type CupertinoSliverRefreshControlOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// RefreshTriggerPullDistance is how far the list must be pulled to arm the refresh; the default is 100.
	RefreshTriggerPullDistance foundation.Opt[float64]
	// RefreshIndicatorExtent is the extent that the indicator keeps while it refreshes; the default is 60.
	RefreshIndicatorExtent foundation.Opt[float64]
	// Builder builds the indicator; the default is CupertinoSliverRefreshControlBuildRefreshIndicator.
	Builder RefreshControlIndicatorBuilder
	// NoBuilder shows no indicator at all (Flutter's builder: null).
	NoBuilder bool
	// OnRefresh is called when the refresh starts. With none the control only shows the pull.
	OnRefresh RefreshCallback
}

// CupertinoSliverRefreshControl is the pull to refresh of iOS: a sliver that grows as the list is overscrolled, shows an
// activity indicator and calls OnRefresh when the list is let go from far enough.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl
type CupertinoSliverRefreshControl struct {
	widgets.StatefulWidget
	o CupertinoSliverRefreshControlOpts
}

// NewCupertinoSliverRefreshControl creates the control. In debug builds it panics with the tag
// CupertinoSliverRefreshControlTriggerNotPositive, CupertinoSliverRefreshControlExtentNegative or
// CupertinoSliverRefreshControlExtentAboveTrigger when its distances break the rule of that name.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl.new
func NewCupertinoSliverRefreshControl(opts ...CupertinoSliverRefreshControlOpts) *CupertinoSliverRefreshControl {
	o := firstOf(opts)
	if !o.RefreshTriggerPullDistance.IsSome() {
		o.RefreshTriggerPullDistance = foundation.Some(refreshDefaultTriggerPullDistance)
	}
	if !o.RefreshIndicatorExtent.IsSome() {
		o.RefreshIndicatorExtent = foundation.Some(refreshDefaultIndicatorExtent)
	}
	if o.Builder == nil && !o.NoBuilder {
		o.Builder = CupertinoSliverRefreshControlBuildRefreshIndicator
	}
	if foundation.DebugMode {
		trigger, extent := o.RefreshTriggerPullDistance.Value(), o.RefreshIndicatorExtent.Value()
		switch {
		case !(trigger > 0):
			panic(errors.New(tagCupertinoSliverRefreshControlTriggerNotPositive, "refreshTriggerPullDistance must be greater than 0", "refreshTriggerPullDistance", trigger))
		case extent < 0:
			panic(errors.New(tagCupertinoSliverRefreshControlExtentNegative, "refreshIndicatorExtent must not be negative", "refreshIndicatorExtent", extent))
		case trigger < extent:
			panic(errors.New(tagCupertinoSliverRefreshControlExtentAboveTrigger, "The refresh indicator cannot take more space in its final state than the amount initially created by overscrolling.",
				"refreshTriggerPullDistance", trigger, "refreshIndicatorExtent", extent))
		}
	}
	return &CupertinoSliverRefreshControl{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// RefreshTriggerPullDistance is how far the list must be pulled to arm the refresh.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl.refreshTriggerPullDistance
func (w *CupertinoSliverRefreshControl) RefreshTriggerPullDistance() float64 {
	return w.o.RefreshTriggerPullDistance.Value()
}

// RefreshIndicatorExtent is the extent that the indicator keeps while it refreshes.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl.refreshIndicatorExtent
func (w *CupertinoSliverRefreshControl) RefreshIndicatorExtent() float64 {
	return w.o.RefreshIndicatorExtent.Value()
}

// CupertinoSliverRefreshControlState is the state of the control of the context, for tests.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl.state
func CupertinoSliverRefreshControlState(bc widgets.BuildContext) RefreshIndicatorMode {
	if s, ok := widgets.FindAncestorStateOfType[*refreshControlState](bc); ok {
		return s.refreshState
	}
	return RefreshIndicatorModeUnspecified
}

// CupertinoSliverRefreshControlBuildRefreshIndicator is the default indicator: an activity indicator that is revealed as
// the list is pulled, spins while it refreshes and shrinks when it is done.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl.buildRefreshIndicator
func CupertinoSliverRefreshControlBuildRefreshIndicator(bc widgets.BuildContext, refreshState RefreshIndicatorMode, pulledExtent, refreshTriggerPullDistance, refreshIndicatorExtent float64) widgets.Widget {
	percentageComplete := ui.ClampDouble(pulledExtent/refreshTriggerPullDistance, 0, 1)
	return widgets.NewCenter(widgets.NewStack([]widgets.Widget{
		widgets.NewPositioned(buildIndicatorForRefreshState(refreshState, refreshActivityIndicatorRadius, percentageComplete),
			widgets.PositionedOpts{Top: foundation.Some(refreshActivityIndicatorMargin), Left: foundation.Some(0.0), Right: foundation.Some(0.0)}),
	}, widgets.StackOpts{ClipBehavior: ui.ClipNone}))
}

func buildIndicatorForRefreshState(refreshState RefreshIndicatorMode, radius, percentageComplete float64) widgets.Widget {
	switch refreshState {
	case RefreshIndicatorModeDrag:
		opacityCurve := animation.NewInterval(0, 0.35, animation.Curves.EaseInOut)
		return widgets.NewOpacity(opacityCurve.Transform(percentageComplete), NewCupertinoActivityIndicatorPartiallyRevealed(
			CupertinoActivityIndicatorPartiallyRevealedOpts{Radius: foundation.Some(radius), Progress: foundation.Some(percentageComplete)}))
	case RefreshIndicatorModeArmed, RefreshIndicatorModeRefresh:
		return NewCupertinoActivityIndicator(CupertinoActivityIndicatorOpts{Radius: foundation.Some(radius)})
	case RefreshIndicatorModeDone:
		return NewCupertinoActivityIndicator(CupertinoActivityIndicatorOpts{Radius: foundation.Some(radius * percentageComplete)})
	}
	return widgets.NewSizedBoxShrink(nil)
}

// CreateState creates the state.
//
// Flutter: cupertino/refresh.dart CupertinoSliverRefreshControl.createState
func (w *CupertinoSliverRefreshControl) CreateState() widgets.State { return &refreshControlState{} }

type refreshControlState struct {
	widgets.StateOf[*CupertinoSliverRefreshControl]
	refreshState             RefreshIndicatorMode
	refreshTask              <-chan struct{}
	refreshing               bool
	latestIndicatorBoxExtent float64
	hasSliverLayoutExtent    bool
}

func (s *refreshControlState) InitState() { s.refreshState = RefreshIndicatorModeInactive }

// transitionNextState finds the state that follows the current one from the extent of the indicator.
//
// Flutter: cupertino/refresh.dart _CupertinoSliverRefreshControlState.transitionNextState
func (s *refreshControlState) transitionNextState() RefreshIndicatorMode {
	w := s.Widget()
	var next RefreshIndicatorMode
	goToDone := func() {
		next = RefreshIndicatorModeDone
		if widgets.WidgetsBindingOf(s.Context()).Scheduler().SchedulerPhase() == scheduler.SchedulerPhaseIdle {
			s.SetState(func() { s.hasSliverLayoutExtent = false })
		} else {
			widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
				s.SetState(func() { s.hasSliverLayoutExtent = false })
			})
		}
	}
	// The cases of the switch in Dart continue into the next one.
	state := s.refreshState
	if state == RefreshIndicatorModeInactive {
		if s.latestIndicatorBoxExtent <= 0 {
			return RefreshIndicatorModeInactive
		}
		next = RefreshIndicatorModeDrag
		state = RefreshIndicatorModeDrag
	}
	if state == RefreshIndicatorModeDrag {
		switch {
		case s.latestIndicatorBoxExtent == 0:
			return RefreshIndicatorModeInactive
		case s.latestIndicatorBoxExtent < w.o.RefreshTriggerPullDistance.Value():
			return RefreshIndicatorModeDrag
		}
		if w.o.OnRefresh != nil {
			widgets.WidgetsBindingOf(s.Context()).HapticFeedback().MediumImpact()
			widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
				s.startRefresh()
				s.SetState(func() { s.hasSliverLayoutExtent = true })
			})
		}
		return RefreshIndicatorModeArmed
	}
	if state == RefreshIndicatorModeArmed {
		if s.refreshState == RefreshIndicatorModeArmed && !s.refreshing {
			goToDone()
			state = RefreshIndicatorModeDone
		} else {
			if s.latestIndicatorBoxExtent > w.o.RefreshIndicatorExtent.Value() {
				return RefreshIndicatorModeArmed
			}
			next = RefreshIndicatorModeRefresh
			state = RefreshIndicatorModeRefresh
		}
	}
	if state == RefreshIndicatorModeRefresh {
		if s.refreshing {
			return RefreshIndicatorModeRefresh
		}
		goToDone()
		state = RefreshIndicatorModeDone
	}
	if state == RefreshIndicatorModeDone {
		if s.latestIndicatorBoxExtent > w.o.RefreshTriggerPullDistance.Value()*refreshInactiveResetOverscrollFraction {
			return RefreshIndicatorModeDone
		}
		next = RefreshIndicatorModeInactive
	}
	return next
}

// startRefresh calls OnRefresh and waits for the end of the refresh, which may come on any goroutine.
func (s *refreshControlState) startRefresh() {
	s.refreshing = true
	task := s.Widget().o.OnRefresh()
	s.refreshTask = task
	finish := func() {
		if !s.Mounted() {
			return
		}
		s.SetState(func() { s.refreshing = false; s.refreshTask = nil })
		s.refreshState = s.transitionNextState()
	}
	if task == nil {
		finish()
		return
	}
	binding := widgets.WidgetsBindingOf(s.Context())
	go func() {
		<-task
		binding.Post(finish)
	}()
}

func (s *refreshControlState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return newCupertinoSliverRefresh(w.o.RefreshIndicatorExtent.Value(), s.hasSliverLayoutExtent,
		widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
			s.latestIndicatorBoxExtent = constraints.MaxHeight
			s.refreshState = s.transitionNextState()
			if w.o.Builder != nil && s.latestIndicatorBoxExtent > 0 {
				return w.o.Builder(bc, s.refreshState, s.latestIndicatorBoxExtent, w.o.RefreshTriggerPullDistance.Value(), w.o.RefreshIndicatorExtent.Value())
			}
			return widgets.NewLimitedBox(widgets.NewSizedBoxExpand(nil), widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0), MaxHeight: foundation.Some(0.0)})
		}))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSliverRefreshDirection                 = "CupertinoSliverRefreshDirection"
	tagCupertinoSliverRefreshControlTriggerNotPositive = "CupertinoSliverRefreshControlTriggerNotPositive"
	tagCupertinoSliverRefreshControlExtentNegative     = "CupertinoSliverRefreshControlExtentNegative"
	tagCupertinoSliverRefreshControlExtentAboveTrigger = "CupertinoSliverRefreshControlExtentAboveTrigger"
)
