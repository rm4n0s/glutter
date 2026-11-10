package cupertino

import (
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// menuSwipeTarget marks an item of a menu that a swipe can enter, leave and complete.
//
// Flutter: cupertino/menu_anchor.dart _SwipeTarget
type menuSwipeTarget struct {
	widgets.StatelessWidget
	onEnter, onExit, onCompletion func()
	child                         widgets.Widget
}

func newMenuSwipeTarget(onEnter, onExit, onCompletion func(), child widgets.Widget) *menuSwipeTarget {
	return &menuSwipeTarget{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), onEnter: onEnter, onExit: onExit, onCompletion: onCompletion, child: child}
}

func (w *menuSwipeTarget) isOpaque() bool { return true }

func (w *menuSwipeTarget) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewMetaData(w.child, widgets.MetaDataOpts{MetaData: w})
}

// menuSwipeScope gives the swipe region to the surfaces below it.
//
// Flutter: cupertino/menu_anchor.dart _SwipeScope
type menuSwipeScope struct {
	widgets.InheritedWidget
	state *menuSwipeRegionState
}

func (w *menuSwipeScope) UpdateShouldNotify(old widgets.Widget) bool {
	return old.(*menuSwipeScope).state != w.state
}

// menuSwipeRegion is the region where a swipe that started on a swipe surface goes on: it reports the distance of
// the pointer to the nearest surface.
//
// Flutter: cupertino/menu_anchor.dart _SwipeRegion
type menuSwipeRegion struct {
	widgets.StatefulWidget
	enabled           bool
	onDistanceChanged func(distance float64)
	child             widgets.Widget
}

func newMenuSwipeRegion(enabled bool, onDistanceChanged func(float64), child widgets.Widget) *menuSwipeRegion {
	return &menuSwipeRegion{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}), enabled: enabled, onDistanceChanged: onDistanceChanged, child: child}
}

func menuSwipeRegionOf(bc widgets.BuildContext) *menuSwipeRegionState {
	if scope, ok := widgets.DependOn[*menuSwipeScope](bc); ok {
		return scope.state
	}
	return nil
}

func (w *menuSwipeRegion) CreateState() widgets.State { return &menuSwipeRegionState{} }

type menuSwipeRegionState struct {
	widgets.StateOf[*menuSwipeRegion]
	surfaces   []*renderMenuSwipeSurface
	recognizer interface{ Dispose() }
	position   foundation.Opt[ui.Offset]
}

func (s *menuSwipeRegionState) isSwiping() bool { return s.position.IsSome() }

func (s *menuSwipeRegionState) DidUpdateWidget(old *menuSwipeRegion) {
	if s.Widget().enabled != old.enabled && !s.Widget().enabled {
		s.disposeRecognizer()
		s.position = foundation.None[ui.Offset]()
		s.Widget().onDistanceChanged(0)
	}
}

func (s *menuSwipeRegionState) disposeRecognizer() {
	if s.recognizer != nil {
		s.recognizer.Dispose()
		s.recognizer = nil
	}
}

func (s *menuSwipeRegionState) Dispose() { s.disposeRecognizer() }

func (s *menuSwipeRegionState) attachSurface(surface *renderMenuSwipeSurface) {
	if !slices.Contains(s.surfaces, surface) {
		s.surfaces = append(s.surfaces, surface)
	}
}

func (s *menuSwipeRegionState) detachSurface(surface *renderMenuSwipeSurface) {
	s.surfaces = slices.DeleteFunc(s.surfaces, func(o *renderMenuSwipeSurface) bool { return o == surface })
}

func (s *menuSwipeRegionState) gestureSettings() foundation.Opt[gestures.DeviceGestureSettings] {
	if settings, ok := widgets.MediaQueryMaybeGestureSettingsOf(s.Context()); ok {
		return foundation.Some(settings)
	}
	return foundation.None[gestures.DeviceGestureSettings]()
}

func (s *menuSwipeRegionState) beginSwipe(event *gestures.PointerDownEvent, delay time.Duration, onStart func()) {
	if s.isSwiping() || !s.Widget().enabled {
		return
	}
	s.disposeRecognizer()
	handleStart := func(position ui.Offset) gestures.Drag {
		if onStart != nil {
			onStart()
		}
		return s.createSwipeHandle(position)
	}
	binding := widgets.WidgetsBindingOf(s.Context()).Gestures()
	filter := func(buttons int) bool { return buttons == gestures.PrimaryButton }
	if delay == 0 {
		r := gestures.NewImmediateMultiDragGestureRecognizer(binding, gestures.MultiDragGestureRecognizerOpts{AllowedButtonsFilter: filter})
		r.OnStart = handleStart
		r.GestureSettings = s.gestureSettings()
		s.recognizer = r
		r.AddPointer(event)
		return
	}
	r := gestures.NewDelayedMultiDragGestureRecognizer(binding, gestures.DelayedMultiDragGestureRecognizerOpts{
		MultiDragGestureRecognizerOpts: gestures.MultiDragGestureRecognizerOpts{AllowedButtonsFilter: filter}, Delay: foundation.Some(delay)})
	r.OnStart = handleStart
	r.GestureSettings = s.gestureSettings()
	s.recognizer = r
	r.AddPointer(event)
}

func (s *menuSwipeRegionState) createSwipeHandle(position ui.Offset) gestures.Drag {
	s.position = foundation.Some(position)
	return newMenuSwipeHandle(widgets.WidgetsBindingOf(s.Context()), widgets.ViewOf(s.Context()).ViewID(), position,
		s.handleSwipeUpdate, func(gestures.DragEndDetails) { s.completeSwipe() }, s.completeSwipe)
}

func (s *menuSwipeRegionState) handleSwipeUpdate(d gestures.DragUpdateDetails) {
	s.position = foundation.Some(s.position.Value().Add(d.Delta))
	minimumSquaredDistance := math.MaxFloat64
	for _, surface := range s.surfaces {
		squared := menuSquaredDistanceToRect(s.position.Value(), surface.computeRect())
		if math.Floor(squared) == 0 {
			s.Widget().onDistanceChanged(0)
			return
		}
		minimumSquaredDistance = math.Min(squared, minimumSquaredDistance)
	}
	distance := 0.0
	if minimumSquaredDistance != 0 {
		distance = math.Sqrt(minimumSquaredDistance)
	}
	s.Widget().onDistanceChanged(distance)
}

func (s *menuSwipeRegionState) completeSwipe() {
	s.disposeRecognizer()
	s.position = foundation.None[ui.Offset]()
	if s.Mounted() {
		s.Widget().onDistanceChanged(0)
	}
}

func (s *menuSwipeRegionState) Build(widgets.BuildContext) widgets.Widget {
	return &menuSwipeScope{InheritedWidget: widgets.NewInheritedWidgetBase(s.Widget().child), state: s}
}

// menuSwipeSurface is a part of the screen that a swipe started on it can leave and come back to: the anchor
// (to open the menu by a long press) or the menu.
//
// Flutter: cupertino/menu_anchor.dart _SwipeSurface
type menuSwipeSurface struct {
	widgets.SingleChildRenderObjectWidget
	delay   time.Duration
	onStart func()
}

func newMenuSwipeSurface(child widgets.Widget, delay time.Duration, onStart func()) *menuSwipeSurface {
	return &menuSwipeSurface{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), delay: delay, onStart: onStart}
}

func (w *menuSwipeSurface) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	return newRenderMenuSwipeSurface(menuSwipeRegionOf(bc), w.delay, w.onStart)
}

func (w *menuSwipeSurface) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderMenuSwipeSurface)
	r.setRegion(menuSwipeRegionOf(bc))
	r.delay = w.delay
	r.onStart = w.onStart
}

// renderMenuSwipeSurface starts the swipe of its region at a pointer down.
//
// Flutter: cupertino/menu_anchor.dart _RenderSwipeSurface
type renderMenuSwipeSurface struct {
	rendering.RenderProxyBoxWithHitTestBehavior
	region  *menuSwipeRegionState
	delay   time.Duration
	onStart func()
}

func newRenderMenuSwipeSurface(region *menuSwipeRegionState, delay time.Duration, onStart func()) *renderMenuSwipeSurface {
	r := &renderMenuSwipeSurface{region: region, delay: delay, onStart: onStart}
	rendering.InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, rendering.HitTestBehaviorOpaque, nil)
	region.attachSurface(r)
	return r
}

func (r *renderMenuSwipeSurface) setRegion(value *menuSwipeRegionState) {
	if r.region != value {
		r.region.detachSurface(r)
		r.region = value
		r.region.attachSurface(r)
	}
}

func (r *renderMenuSwipeSurface) computeRect() ui.Rect {
	offset := r.LocalToGlobal(ui.Offset{}, nil)
	return ui.NewRectFromLTWH(offset.Dx, offset.Dy, r.Size().Width, r.Size().Height)
}

func (r *renderMenuSwipeSurface) Detach() {
	r.region.detachSurface(r)
	r.RenderProxyBoxWithHitTestBehavior.Detach()
}

func (r *renderMenuSwipeSurface) Dispose() {
	r.region.detachSurface(r)
	r.RenderProxyBoxWithHitTestBehavior.Dispose()
}

func (r *renderMenuSwipeSurface) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	r.DebugHandleEvent(event, entry)
	if down, ok := event.(*gestures.PointerDownEvent); ok {
		r.region.beginSwipe(down, r.delay, r.onStart)
	}
}

// menuSwipeHandle is the drag of a swipe: it enters and leaves the swipe targets under the pointer.
//
// Flutter: cupertino/menu_anchor.dart _SwipeHandle
type menuSwipeHandle struct {
	binding         *widgets.WidgetsBinding
	viewID          ui.ViewID
	enteredTargets  []*menuSwipeTarget
	onSwipeUpdate   func(gestures.DragUpdateDetails)
	onSwipeEnd      func(gestures.DragEndDetails)
	onSwipeCanceled func()
	position        ui.Offset
}

func newMenuSwipeHandle(binding *widgets.WidgetsBinding, viewID ui.ViewID, initial ui.Offset, onUpdate func(gestures.DragUpdateDetails),
	onEnd func(gestures.DragEndDetails), onCancel func()) *menuSwipeHandle {
	h := &menuSwipeHandle{binding: binding, viewID: viewID, position: initial, onSwipeUpdate: onUpdate, onSwipeEnd: onEnd, onSwipeCanceled: onCancel}
	h.updateSwipe()
	return h
}

func (h *menuSwipeHandle) Update(details gestures.DragUpdateDetails) {
	old := h.position
	h.position = h.position.Add(details.Delta)
	if h.position != old {
		h.updateSwipe()
		h.onSwipeUpdate(details)
	}
}

func (h *menuSwipeHandle) End(details gestures.DragEndDetails) {
	h.leaveAllEntered(true)
	h.onSwipeEnd(details)
}

func (h *menuSwipeHandle) Cancel() {
	h.leaveAllEntered(false)
	h.onSwipeCanceled()
}

func (h *menuSwipeHandle) updateSwipe() {
	result := gestures.NewHitTestResult()
	h.binding.Renderer().HitTestInView(result, h.position, h.viewID)
	var targets []*menuSwipeTarget
	for _, entry := range result.Path() {
		if md, ok := entry.Target().(*rendering.RenderMetaData); ok {
			if t, ok := md.MetaData.(*menuSwipeTarget); ok {
				targets = append(targets, t)
			}
		}
	}
	h.enteredTargets = slices.DeleteFunc(h.enteredTargets, func(t *menuSwipeTarget) bool {
		if !slices.Contains(targets, t) {
			if t.onExit != nil {
				t.onExit()
			}
			return true
		}
		return false
	})
	hitTargets := map[*menuSwipeTarget]bool{}
	var hitOrder, newlyEntered []*menuSwipeTarget
	hitExistingTarget := false
	for _, t := range targets {
		if slices.Contains(h.enteredTargets, t) {
			if !hitTargets[t] {
				hitOrder = append(hitOrder, t)
			}
			hitTargets[t] = true
			hitExistingTarget = true
			continue
		}
		if !hitExistingTarget {
			if !hitTargets[t] {
				hitOrder = append(hitOrder, t)
			}
			hitTargets[t] = true
			newlyEntered = append(newlyEntered, t)
		}
		if t.isOpaque() {
			break
		}
	}
	for _, t := range slices.Backward(h.enteredTargets) {
		if !hitTargets[t] && t.onExit != nil {
			t.onExit()
		}
	}
	for _, t := range slices.Backward(newlyEntered) {
		if t.onEnter != nil {
			t.onEnter()
		}
	}
	h.enteredTargets = hitOrder
}

func (h *menuSwipeHandle) leaveAllEntered(pointerUp bool) {
	for _, t := range h.enteredTargets {
		if t.onExit != nil {
			t.onExit()
		}
		if pointerUp && t.onCompletion != nil {
			t.onCompletion()
		}
	}
	h.enteredTargets = nil
}
