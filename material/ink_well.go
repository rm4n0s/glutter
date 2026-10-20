package material

import (
	"github.com/rm4n0s/errors"
	"reflect"
	"slices"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// parentInkResponseState is Flutter's _ParentInkResponseState: the state of an
// InkResponse that nested InkResponses tell when they are pressed, so that the
// parent does not splash too.
//
// Flutter: material/ink_well.dart _ParentInkResponseState
type parentInkResponseState interface {
	markChildInkResponsePressed(childState parentInkResponseState, value bool)
}

// parentInkResponseProvider is Flutter's _ParentInkResponseProvider.
//
// Flutter: material/ink_well.dart _ParentInkResponseProvider
type parentInkResponseProvider struct {
	widgets.InheritedWidget
	state parentInkResponseState
}

func (w *parentInkResponseProvider) UpdateShouldNotify(old widgets.Widget) bool {
	return w.state != old.(*parentInkResponseProvider).state
}

// InkResponseOpts are the optional parameters of NewInkResponse.
//
// Flutter: material/ink_well.dart InkResponse.new
type InkResponseOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnTap is called when the user taps this part of the material.
	OnTap gestures.GestureTapCallback
	// OnTapDown is called when the user taps down on this part of the material.
	OnTapDown gestures.GestureTapDownCallback
	// OnTapUp is called when the user releases a tap that was started on this part of the material.
	OnTapUp gestures.GestureTapUpCallback
	// OnTapCancel is called when the user cancels a tap that was started on this part of the material.
	OnTapCancel gestures.GestureTapCancelCallback
	// OnDoubleTap is called when the user double taps this part of the material.
	OnDoubleTap gestures.GestureTapCallback
	// OnLongPress is called when the user long-presses on this part of the material.
	OnLongPress gestures.GestureLongPressCallback
	// OnLongPressUp is called when the user lifts the pointer after a long press.
	OnLongPressUp gestures.GestureLongPressUpCallback
	// OnSecondaryTap is called when the user taps this part of the material with a secondary button.
	OnSecondaryTap gestures.GestureTapCallback
	// OnSecondaryTapUp is called when the user releases a secondary button tap.
	OnSecondaryTapUp gestures.GestureTapUpCallback
	// OnSecondaryTapDown is called when the user taps down with a secondary button.
	OnSecondaryTapDown gestures.GestureTapDownCallback
	// OnSecondaryTapCancel is called when the user cancels a secondary button tap.
	OnSecondaryTapCancel gestures.GestureTapCancelCallback
	// OnHighlightChanged is called when this part of the material either becomes
	// highlighted or stops being highlighted.
	OnHighlightChanged func(value bool)
	// OnHover is called when a pointer enters or exits the ink response area.
	OnHover func(value bool)
	// MouseCursor is the cursor for a mouse pointer when it enters or is
	// hovering over the widget; the default is WidgetStateMouseCursorAdaptiveClickable.
	MouseCursor services.MouseCursor
	// ContainedInkWell says whether this ink response should be clipped its
	// bounds.
	ContainedInkWell bool
	// HighlightShape is the shape (e.g., circle, rectangle) to use for the
	// highlight drawn around this part of the material; the default is a circle.
	HighlightShape painting.BoxShape
	// Radius is the radius of the ink splash.
	Radius foundation.Opt[float64]
	// BorderRadius is the border radius of the containing rectangle, used when
	// the highlight shape is a rectangle and the ink is contained.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// CustomBorder is the custom clip border; it takes precedence over the
	// highlight shape and the border radius.
	CustomBorder painting.ShapeBorder
	// FocusColor is the focus color; the default is the focus color of the theme.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the hover color; the default is the hover color of the theme.
	HoverColor foundation.Opt[ui.Color]
	// HighlightColor is the highlight color; the default is the highlight color
	// of the theme.
	HighlightColor foundation.Opt[ui.Color]
	// OverlayColor defines the ink response focus, hover, and splash colors; it
	// takes precedence over the colors above.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashColor is the splash color; the default is the splash color of the theme.
	SplashColor foundation.Opt[ui.Color]
	// SplashFactory defines the appearance of the splash; the default is the one
	// of the theme.
	SplashFactory InteractiveInkFeatureFactory
	// EnableFeedback is whether detected gestures should provide acoustic and/or
	// haptic feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// ExcludeFromSemantics is whether to exclude the gestures from the semantics tree.
	ExcludeFromSemantics bool
	// OnFocusChange is called when the focus changes.
	OnFocusChange func(value bool)
	// Autofocus is whether this widget will focus itself if nothing else has focus.
	Autofocus bool
	// FocusNode is the focus node used by the widget.
	FocusNode *widgets.FocusNode
	// CanRequestFocus is whether this widget can request the focus; the default is true.
	CanRequestFocus foundation.Opt[bool]
	// StatesController is the controller of the widget states; the default is one the widget makes.
	StatesController *widgets.WidgetStatesController
	// HoverDuration is the duration of the fade of the hover and focus highlights.
	HoverDuration foundation.Opt[time.Duration]
	// GetRectCallback gives the rectangle of the splash and of the highlight for
	// the reference box; the default is none, that is the reference box.
	//
	// Deviation: Dart overrides InkResponse.getRectCallback in a subclass.
	GetRectCallback func(referenceBox rendering.Box) RectCallback
}

// InkResponse is an area of a Material that responds to touch. It displays
// splashes and highlights that are clipped to the shape (or not clipped, when
// the area is not contained).
//
// Flutter: material/ink_well.dart InkResponse
type InkResponse struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     InkResponseOpts
}

// NewInkResponse creates an area of a Material that responds to touch.
//
// Flutter: material/ink_well.dart InkResponse.new
func NewInkResponse(child widgets.Widget, opts ...InkResponseOpts) *InkResponse {
	o := firstOpts(opts)
	if o.HighlightShape == painting.BoxShapeUnspecified {
		o.HighlightShape = painting.BoxShapeCircle
	}
	return &InkResponse{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child, o}
}

// Child is the widget below this widget in the tree.
//
// Flutter: material/ink_well.dart InkResponse.child
func (w *InkResponse) Child() widgets.Widget { return w.child }

// Opts returns the options of the response.
func (w *InkResponse) Opts() InkResponseOpts { return w.o }

// Build makes the stateful widget that does the work, below the parent state.
//
// Flutter: material/ink_well.dart InkResponse.build
func (w *InkResponse) Build(bc widgets.BuildContext) widgets.Widget {
	var parent parentInkResponseState
	if p, ok := widgets.DependOn[*parentInkResponseProvider](bc); ok {
		parent = p.state
	}
	return &inkResponseStateWidget{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}), child: w.child, o: w.o, parentState: parent}
}

// InkWellOpts are the optional parameters of NewInkWell: those of InkResponse
// but the highlight shape and the contained flag.
//
// Flutter: material/ink_well.dart InkWell.new
type InkWellOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnTap is called when the user taps this part of the material.
	OnTap gestures.GestureTapCallback
	// OnDoubleTap is called when the user double taps this part of the material.
	OnDoubleTap gestures.GestureTapCallback
	// OnLongPress is called when the user long-presses on this part of the material.
	OnLongPress gestures.GestureLongPressCallback
	// OnLongPressUp is called when the user lifts the pointer after a long press.
	OnLongPressUp gestures.GestureLongPressUpCallback
	// OnTapDown is called when the user taps down on this part of the material.
	OnTapDown gestures.GestureTapDownCallback
	// OnTapUp is called when the user releases a tap that was started on this part of the material.
	OnTapUp gestures.GestureTapUpCallback
	// OnTapCancel is called when the user cancels a tap that was started on this part of the material.
	OnTapCancel gestures.GestureTapCancelCallback
	// OnSecondaryTap is called when the user taps this part of the material with a secondary button.
	OnSecondaryTap gestures.GestureTapCallback
	// OnSecondaryTapUp is called when the user releases a secondary button tap.
	OnSecondaryTapUp gestures.GestureTapUpCallback
	// OnSecondaryTapDown is called when the user taps down with a secondary button.
	OnSecondaryTapDown gestures.GestureTapDownCallback
	// OnSecondaryTapCancel is called when the user cancels a secondary button tap.
	OnSecondaryTapCancel gestures.GestureTapCancelCallback
	// OnHighlightChanged is called when this part of the material either becomes
	// highlighted or stops being highlighted.
	OnHighlightChanged func(value bool)
	// OnHover is called when a pointer enters or exits the ink response area.
	OnHover func(value bool)
	// MouseCursor is the cursor for a mouse pointer when it enters or is
	// hovering over the widget.
	MouseCursor services.MouseCursor
	// FocusColor is the focus color; the default is the focus color of the theme.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the hover color; the default is the hover color of the theme.
	HoverColor foundation.Opt[ui.Color]
	// HighlightColor is the highlight color; the default is the highlight color
	// of the theme.
	HighlightColor foundation.Opt[ui.Color]
	// OverlayColor defines the ink response focus, hover, and splash colors.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashColor is the splash color; the default is the splash color of the theme.
	SplashColor foundation.Opt[ui.Color]
	// SplashFactory defines the appearance of the splash.
	SplashFactory InteractiveInkFeatureFactory
	// Radius is the radius of the ink splash.
	Radius foundation.Opt[float64]
	// BorderRadius is the clipping radius of the containing rect.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// CustomBorder is the custom clip border.
	CustomBorder painting.ShapeBorder
	// EnableFeedback is whether detected gestures should provide acoustic and/or
	// haptic feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// ExcludeFromSemantics is whether to exclude the gestures from the semantics tree.
	ExcludeFromSemantics bool
	// FocusNode is the focus node used by the widget.
	FocusNode *widgets.FocusNode
	// CanRequestFocus is whether this widget can request the focus; the default is true.
	CanRequestFocus foundation.Opt[bool]
	// OnFocusChange is called when the focus changes.
	OnFocusChange func(value bool)
	// Autofocus is whether this widget will focus itself if nothing else has focus.
	Autofocus bool
	// StatesController is the controller of the widget states.
	StatesController *widgets.WidgetStatesController
	// HoverDuration is the duration of the fade of the hover and focus highlights.
	HoverDuration foundation.Opt[time.Duration]
}

// InkWell is a rectangular area of a Material that responds to touch. It is an
// InkResponse that is contained and has a rectangle highlight.
//
// Flutter: material/ink_well.dart InkWell
type InkWell struct {
	*InkResponse
}

// NewInkWell creates an ink well.
//
// Flutter: material/ink_well.dart InkWell.new
func NewInkWell(child widgets.Widget, opts ...InkWellOpts) *InkWell {
	o := firstOpts(opts)
	return &InkWell{NewInkResponse(child, InkResponseOpts{
		Key: o.Key, OnTap: o.OnTap, OnDoubleTap: o.OnDoubleTap, OnLongPress: o.OnLongPress, OnLongPressUp: o.OnLongPressUp,
		OnTapDown: o.OnTapDown, OnTapUp: o.OnTapUp, OnTapCancel: o.OnTapCancel, OnSecondaryTap: o.OnSecondaryTap,
		OnSecondaryTapUp: o.OnSecondaryTapUp, OnSecondaryTapDown: o.OnSecondaryTapDown, OnSecondaryTapCancel: o.OnSecondaryTapCancel,
		OnHighlightChanged: o.OnHighlightChanged, OnHover: o.OnHover, MouseCursor: o.MouseCursor, FocusColor: o.FocusColor,
		HoverColor: o.HoverColor, HighlightColor: o.HighlightColor, OverlayColor: o.OverlayColor, SplashColor: o.SplashColor,
		SplashFactory: o.SplashFactory, Radius: o.Radius, BorderRadius: o.BorderRadius, CustomBorder: o.CustomBorder,
		EnableFeedback: o.EnableFeedback, ExcludeFromSemantics: o.ExcludeFromSemantics, FocusNode: o.FocusNode,
		CanRequestFocus: o.CanRequestFocus, OnFocusChange: o.OnFocusChange, Autofocus: o.Autofocus,
		StatesController: o.StatesController, HoverDuration: o.HoverDuration,
		ContainedInkWell: true, HighlightShape: painting.BoxShapeRectangle,
	})}
}

// inkResponseStateWidget is Flutter's _InkResponseStateWidget.
//
// Flutter: material/ink_well.dart _InkResponseStateWidget
type inkResponseStateWidget struct {
	widgets.StatefulWidget
	child       widgets.Widget
	o           InkResponseOpts
	parentState parentInkResponseState
}

func (w *inkResponseStateWidget) enableFeedback() bool { return w.o.EnableFeedback.OrElse(true) }

func (w *inkResponseStateWidget) CreateState() widgets.State {
	return &inkResponseState{highlights: map[highlightType]*InkHighlight{}}
}

func (w *inkResponseStateWidget) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	o := w.o
	var gestureNames []string
	for _, g := range []struct {
		name string
		set  bool
	}{
		{"tap", o.OnTap != nil}, {"double tap", o.OnDoubleTap != nil}, {"long press", o.OnLongPress != nil},
		{"long press up", o.OnLongPressUp != nil}, {"tap down", o.OnTapDown != nil}, {"tap up", o.OnTapUp != nil},
		{"tap cancel", o.OnTapCancel != nil}, {"secondary tap", o.OnSecondaryTap != nil}, {"secondary tap up", o.OnSecondaryTapUp != nil},
		{"secondary tap down", o.OnSecondaryTapDown != nil}, {"secondary tap cancel", o.OnSecondaryTapCancel != nil},
	} {
		if g.set {
			gestureNames = append(gestureNames, g.name)
		}
	}
	p.Add(foundation.NewIterableProperty("gestures", gestureNames, foundation.IterablePropertyOpts{IfEmpty: "<none>"}))
	p.Add(foundation.NewDiagnosticsProperty("mouseCursor", o.MouseCursor))
	p.Add(foundation.NewDiagnosticsProperty("containedInkWell", o.ContainedInkWell, foundation.DiagnosticsPropertyOpts{Level: foundation.DiagnosticLevelFine}))
	description := o.HighlightShape.String()
	if o.ContainedInkWell {
		description = "clipped to " + description
	}
	p.Add(foundation.NewDiagnosticsProperty("highlightShape", o.HighlightShape, foundation.DiagnosticsPropertyOpts{Description: description, ShowName: foundation.Some(false)}))
}

type highlightType int

const (
	highlightPressed highlightType = iota
	highlightHover
	highlightFocus
)

// inkResponseState is Flutter's _InkResponseState.
//
// Flutter: material/ink_well.dart _InkResponseState
type inkResponseState struct {
	widgets.StateOf[*inkResponseStateWidget]
	widgets.AutomaticKeepAliveClient

	splashes                 []InteractiveInkFeature
	hasSplashes              bool
	currentSplash            InteractiveInkFeature
	hovering                 bool
	highlights               map[highlightType]*InkHighlight
	actionMap                widgets.ActionMap
	internalStatesController *widgets.WidgetStatesController
	statesListener           foundation.ListenerHandle
	activeChildren           []parentInkResponseState
	activationTimer          *gestures.Timer
	hasFocus                 bool
	focusHandle              widgets.FocusHandlerHandle
}

const inkActivationDuration = 100 * time.Millisecond

func (s *inkResponseState) highlightsExist() bool {
	for _, h := range s.highlights {
		if h != nil {
			return true
		}
	}
	return false
}

func (s *inkResponseState) markChildInkResponsePressed(childState parentInkResponseState, value bool) {
	lastAnyPressed := s.anyChildInkResponsePressed()
	if value {
		s.activeChildren = append(s.activeChildren, childState)
	} else if i := slices.Index(s.activeChildren, childState); i >= 0 {
		s.activeChildren = slices.Delete(s.activeChildren, i, i+1)
	}
	nowAnyPressed := s.anyChildInkResponsePressed()
	if nowAnyPressed != lastAnyPressed && s.Widget().parentState != nil {
		s.Widget().parentState.markChildInkResponsePressed(s, nowAnyPressed)
	}
}

func (s *inkResponseState) anyChildInkResponsePressed() bool { return len(s.activeChildren) > 0 }

func (s *inkResponseState) activateOnIntent(widgets.Intent) any {
	if s.activationTimer != nil {
		s.activationTimer.Cancel()
		s.activationTimer = nil
	}
	s.startNewSplashFromContext()
	if s.currentSplash != nil {
		s.currentSplash.Confirm()
	}
	s.currentSplash = nil
	w := s.Widget()
	if w.o.OnTap != nil {
		if w.enableFeedback() {
			widgets.FeedbackForTap(s.Context())
		}
		w.o.OnTap()
	}
	// Delay the call to updateHighlight to simulate a pressed delay and give
	// WidgetStatesController listeners a chance to react.
	s.activationTimer = widgets.WidgetsBindingOf(s.Context()).Gestures().NewTimer(inkActivationDuration, func() {
		s.updateHighlight(highlightPressed, false, true)
	})
	return nil
}

func (s *inkResponseState) simulateTap() {
	s.startNewSplashFromContext()
	s.handleTap()
}

func (s *inkResponseState) simulateLongPress() {
	s.startNewSplashFromContext()
	s.handleLongPress()
}

func (s *inkResponseState) handleStatesControllerChange() {
	// Force a rebuild to resolve widget.overlayColor, widget.mouseCursor.
	s.SetState(func() {})
}

func (s *inkResponseState) statesController() *widgets.WidgetStatesController {
	if c := s.Widget().o.StatesController; c != nil {
		return c
	}
	return s.internalStatesController
}

func (s *inkResponseState) initStatesController() {
	if s.Widget().o.StatesController == nil {
		s.internalStatesController = widgets.NewWidgetStatesController()
	}
	s.statesController().Update(widgets.WidgetStateDisabled, !s.enabled())
	s.statesListener = s.statesController().AddListener(s.handleStatesControllerChange)
}

func (s *inkResponseState) InitState() {
	s.actionMap = widgets.ActionMap{
		reflect.TypeFor[widgets.ActivateIntent]():       widgets.NewCallbackAction(func(i widgets.ActivateIntent) any { return s.activateOnIntent(i) }),
		reflect.TypeFor[widgets.ButtonActivateIntent](): widgets.NewCallbackAction(func(i widgets.ButtonActivateIntent) any { return s.activateOnIntent(i) }),
	}
	s.initStatesController()
	s.focusHandle = s.Context().Owner().FocusManager().AddHighlightModeListener(s.handleFocusHighlightModeChange)
}

func (s *inkResponseState) DidUpdateWidget(old *inkResponseStateWidget) {
	w := s.Widget()
	if w.o.StatesController != old.o.StatesController {
		if old.o.StatesController != nil {
			old.o.StatesController.RemoveListener(s.statesListener)
		} else if s.internalStatesController != nil {
			s.internalStatesController.RemoveListener(s.statesListener)
		}
		if w.o.StatesController != nil && s.internalStatesController != nil {
			s.internalStatesController.Dispose()
			s.internalStatesController = nil
		}
		s.initStatesController()
	}
	if w.o.Radius != old.o.Radius || w.o.HighlightShape != old.o.HighlightShape || w.o.BorderRadius != old.o.BorderRadius {
		if hover := s.highlights[highlightHover]; hover != nil {
			hover.Dispose()
			s.updateHighlight(highlightHover, s.hovering, false)
		}
		if focus := s.highlights[highlightFocus]; focus != nil {
			focus.Dispose()
			// Do not call updateFocusHighlights() here because it is called below.
		}
	}
	if !foundation.ValuesEqual(w.o.CustomBorder, old.o.CustomBorder) {
		s.updateHighlightsAndSplashes()
	}
	if s.enabled() != isInkWidgetEnabled(old) {
		s.statesController().Update(widgets.WidgetStateDisabled, !s.enabled())
		if !s.enabled() {
			s.statesController().Update(widgets.WidgetStatePressed, false)
			// Remove the existing hover highlight immediately when enabled is
			// false. Do not rely on updateHighlight or InkHighlight.deactivate
			// to not break the expected lifecycle which is updating hovering when
			// the mouse exits.
			if hover := s.highlights[highlightHover]; hover != nil {
				hover.Dispose()
			}
		}
		// Don't call widget.onHover because many widgets, including the button
		// widgets, apply setState to an ancestor context from onHover.
		s.updateHighlight(highlightHover, s.hovering, false)
	}
	s.updateFocusHighlights()
}

func (s *inkResponseState) Dispose() {
	s.Context().Owner().FocusManager().RemoveHighlightModeListener(s.focusHandle)
	s.statesController().RemoveListener(s.statesListener)
	if s.internalStatesController != nil {
		s.internalStatesController.Dispose()
	}
	if s.activationTimer != nil {
		s.activationTimer.Cancel()
		s.activationTimer = nil
	}
}

// WantKeepAlive keeps the state alive while it has highlights or splashes.
func (s *inkResponseState) WantKeepAlive() bool {
	return s.highlightsExist() || len(s.splashes) > 0
}

func (s *inkResponseState) fadeDurationForType(t highlightType) time.Duration {
	if t == highlightPressed {
		return 200 * time.Millisecond
	}
	return s.Widget().o.HoverDuration.OrElse(50 * time.Millisecond)
}

func (s *inkResponseState) overlayColorFor(states widgets.WidgetStates) (ui.Color, bool) {
	if p := s.Widget().o.OverlayColor; p != nil {
		return p.Resolve(states).Get()
	}
	return ui.Color{}, false
}

func (s *inkResponseState) updateHighlight(t highlightType, value, callOnHover bool) {
	w := s.Widget()
	highlight := s.highlights[t]
	handleInkRemoval := func() {
		s.highlights[t] = nil
		s.UpdateKeepAlive()
	}
	switch t {
	case highlightPressed:
		s.statesController().Update(widgets.WidgetStatePressed, value)
	case highlightHover:
		if callOnHover {
			s.statesController().Update(widgets.WidgetStateHovered, value)
		}
	}
	if t == highlightPressed && w.parentState != nil {
		w.parentState.markChildInkResponsePressed(s, value)
	}
	if value == (highlight != nil && highlight.Active()) {
		return
	}
	if value {
		if highlight == nil {
			theme := ThemeOf(s.Context())
			resolved, ok := s.overlayColorFor(s.statesController().Value())
			if !ok {
				switch t {
				case highlightPressed:
					resolved = w.o.HighlightColor.OrElse(theme.HighlightColor)
				case highlightFocus:
					resolved = w.o.FocusColor.OrElse(theme.FocusColor)
				case highlightHover:
					resolved = w.o.HoverColor.OrElse(theme.HoverColor)
				}
			}
			referenceBox := s.Context().FindRenderObject().(rendering.Box)
			color := resolved
			if !s.enabled() {
				color = resolved.WithAlpha(0)
			}
			var rect RectCallback
			if w.o.GetRectCallback != nil {
				rect = w.o.GetRectCallback(referenceBox)
			}
			s.highlights[t] = NewInkHighlight(MaterialOf(s.Context()), referenceBox, color, directionOf(s.Context()), InkHighlightOpts{
				Shape: w.o.HighlightShape, Radius: w.o.Radius, BorderRadius: w.o.BorderRadius, CustomBorder: w.o.CustomBorder,
				RectCallback: rect, OnRemoved: handleInkRemoval, FadeDuration: s.fadeDurationForType(t)})
			s.UpdateKeepAlive()
		} else {
			highlight.Activate()
		}
	} else {
		highlight.Deactivate()
	}
	switch t {
	case highlightPressed:
		if w.o.OnHighlightChanged != nil {
			w.o.OnHighlightChanged(value)
		}
	case highlightHover:
		if callOnHover && w.o.OnHover != nil {
			w.o.OnHover(value)
		}
	}
}

// directionOf is Directionality.of for the ink: in debug builds a missing
// Directionality panics.
func directionOf(bc widgets.BuildContext) ui.TextDirection { return widgets.DirectionalityOf(bc) }

func (s *inkResponseState) updateHighlightsAndSplashes() {
	w := s.Widget()
	for _, h := range s.highlights {
		if h != nil {
			h.SetCustomBorder(w.o.CustomBorder)
		}
	}
	if s.currentSplash != nil {
		s.currentSplash.AsInteractiveInkFeature().SetCustomBorder(w.o.CustomBorder)
	}
	for _, f := range s.splashes {
		f.AsInteractiveInkFeature().SetCustomBorder(w.o.CustomBorder)
	}
}

func (s *inkResponseState) createSplash(globalPosition ui.Offset) InteractiveInkFeature {
	w := s.Widget()
	inkController := MaterialOf(s.Context())
	referenceBox := s.Context().FindRenderObject().(rendering.Box)
	position := referenceBox.GlobalToLocal(globalPosition, nil)
	theme := ThemeOf(s.Context())
	color, ok := s.overlayColorFor(s.statesController().Value())
	if !ok {
		color = w.o.SplashColor.OrElse(theme.SplashColor)
	}
	var rectCallback RectCallback
	if w.o.ContainedInkWell && w.o.GetRectCallback != nil {
		rectCallback = w.o.GetRectCallback(referenceBox)
	}
	var splash InteractiveInkFeature
	onRemoved := func() {
		if s.hasSplashes {
			if i := slices.Index(s.splashes, splash); i >= 0 {
				s.splashes = slices.Delete(s.splashes, i, i+1)
			}
			if s.currentSplash == splash {
				s.currentSplash = nil
			}
			s.UpdateKeepAlive()
		} // else we're probably in deactivate()
	}
	factory := w.o.SplashFactory
	if factory == nil {
		factory = theme.SplashFactory
	}
	splash = factory.Create(InkFeatureCreateOpts{
		Controller: inkController, ReferenceBox: referenceBox, Position: position, Color: color, ContainedInkWell: w.o.ContainedInkWell,
		RectCallback: rectCallback, Radius: w.o.Radius, BorderRadius: w.o.BorderRadius, CustomBorder: w.o.CustomBorder,
		OnRemoved: onRemoved, TextDirection: directionOf(s.Context())})
	return splash
}

func (s *inkResponseState) handleFocusHighlightModeChange(widgets.FocusHighlightMode) {
	if !s.Mounted() {
		return
	}
	s.SetState(func() { s.updateFocusHighlights() })
}

func (s *inkResponseState) shouldShowFocus() bool {
	if mode, ok := widgets.MediaQueryMaybeNavigationModeOf(s.Context()); ok && mode == widgets.NavigationModeDirectional {
		return s.hasFocus
	}
	return s.enabled() && s.hasFocus
}

func (s *inkResponseState) updateFocusHighlights() {
	showFocus := false
	if s.Context().Owner().FocusManager().HighlightMode() == widgets.FocusHighlightModeTraditional {
		showFocus = s.shouldShowFocus()
	}
	s.updateHighlight(highlightFocus, showFocus, true)
}

func (s *inkResponseState) handleFocusUpdate(hasFocus bool) {
	s.hasFocus = hasFocus
	// Set here rather than in updateHighlight because this widget's states
	// include focused if the InkWell has the focus, rather than if it is showing
	// the focus per FocusManager.highlightMode.
	s.statesController().Update(widgets.WidgetStateFocused, hasFocus)
	s.updateFocusHighlights()
	if cb := s.Widget().o.OnFocusChange; cb != nil {
		cb(hasFocus)
	}
}

func (s *inkResponseState) handleAnyTapDown(details gestures.TapDownDetails) {
	if s.anyChildInkResponsePressed() {
		return
	}
	s.startNewSplash(&details, false)
}

func (s *inkResponseState) handleTapDown(details gestures.TapDownDetails) {
	s.handleAnyTapDown(details)
	if cb := s.Widget().o.OnTapDown; cb != nil {
		cb(details)
	}
}

func (s *inkResponseState) handleTapUp(details gestures.TapUpDetails) {
	if cb := s.Widget().o.OnTapUp; cb != nil {
		cb(details)
	}
}

func (s *inkResponseState) handleSecondaryTapDown(details gestures.TapDownDetails) {
	s.handleAnyTapDown(details)
	if cb := s.Widget().o.OnSecondaryTapDown; cb != nil {
		cb(details)
	}
}

func (s *inkResponseState) handleSecondaryTapUp(details gestures.TapUpDetails) {
	if cb := s.Widget().o.OnSecondaryTapUp; cb != nil {
		cb(details)
	}
}

func (s *inkResponseState) startNewSplashFromContext() { s.startNewSplash(nil, true) }

func (s *inkResponseState) startNewSplash(details *gestures.TapDownDetails, fromContext bool) {
	var globalPosition ui.Offset
	if fromContext {
		referenceBox := s.Context().FindRenderObject().(rendering.Box)
		if foundation.DebugMode && !referenceBox.HasSize() {
			panic(errorInkResponseNoSize())
		}
		globalPosition = referenceBox.LocalToGlobal(referenceBox.PaintBounds().Center(), nil)
	} else {
		globalPosition = details.GlobalPosition
	}
	s.statesController().Update(widgets.WidgetStatePressed, true) // ... before creating the splash
	splash := s.createSplash(globalPosition)
	s.hasSplashes = true
	s.splashes = append(s.splashes, splash)
	if s.currentSplash != nil {
		s.currentSplash.Cancel()
	}
	s.currentSplash = splash
	s.UpdateKeepAlive()
	s.updateHighlight(highlightPressed, true, true)
}

func (s *inkResponseState) confirmCurrent() {
	if s.currentSplash != nil {
		s.currentSplash.Confirm()
	}
	s.currentSplash = nil
}

func (s *inkResponseState) cancelCurrent() {
	if s.currentSplash != nil {
		s.currentSplash.Cancel()
	}
	s.currentSplash = nil
}

func (s *inkResponseState) handleTap() {
	s.confirmCurrent()
	s.updateHighlight(highlightPressed, false, true)
	w := s.Widget()
	if w.o.OnTap != nil {
		if w.enableFeedback() {
			widgets.FeedbackForTap(s.Context())
		}
		w.o.OnTap()
	}
}

func (s *inkResponseState) handleTapCancel() {
	s.cancelCurrent()
	if cb := s.Widget().o.OnTapCancel; cb != nil {
		cb()
	}
	s.updateHighlight(highlightPressed, false, true)
}

func (s *inkResponseState) handleDoubleTap() {
	s.confirmCurrent()
	s.updateHighlight(highlightPressed, false, true)
	if cb := s.Widget().o.OnDoubleTap; cb != nil {
		cb()
	}
}

func (s *inkResponseState) handleLongPress() {
	s.confirmCurrent()
	w := s.Widget()
	if w.o.OnLongPress != nil {
		if w.enableFeedback() {
			widgets.FeedbackForLongPress(s.Context())
		}
		w.o.OnLongPress()
	}
}

func (s *inkResponseState) handleLongPressUp() {
	s.confirmCurrent()
	if cb := s.Widget().o.OnLongPressUp; cb != nil {
		cb()
	}
}

func (s *inkResponseState) handleSecondaryTap() {
	s.confirmCurrent()
	s.updateHighlight(highlightPressed, false, true)
	if cb := s.Widget().o.OnSecondaryTap; cb != nil {
		cb()
	}
}

func (s *inkResponseState) handleSecondaryTapCancel() {
	s.cancelCurrent()
	if cb := s.Widget().o.OnSecondaryTapCancel; cb != nil {
		cb()
	}
	s.updateHighlight(highlightPressed, false, true)
}

func (s *inkResponseState) Deactivate() {
	if s.hasSplashes {
		splashes := s.splashes
		s.splashes = nil
		s.hasSplashes = false
		for _, splash := range splashes {
			splash.AsInkFeature().Dispose()
		}
		s.currentSplash = nil
	}
	for t, h := range s.highlights {
		if h != nil {
			h.Dispose()
		}
		s.highlights[t] = nil
	}
	if w := s.Widget(); w.parentState != nil {
		w.parentState.markChildInkResponsePressed(s, false)
	}
}

func isInkWidgetEnabled(w *inkResponseStateWidget) bool {
	return primaryInkButtonEnabled(w) || secondaryInkButtonEnabled(w)
}

func primaryInkButtonEnabled(w *inkResponseStateWidget) bool {
	return w.o.OnTap != nil || w.o.OnDoubleTap != nil || w.o.OnLongPress != nil || w.o.OnLongPressUp != nil ||
		w.o.OnTapUp != nil || w.o.OnTapDown != nil
}

func secondaryInkButtonEnabled(w *inkResponseStateWidget) bool {
	return w.o.OnSecondaryTap != nil || w.o.OnSecondaryTapUp != nil || w.o.OnSecondaryTapDown != nil
}

func (s *inkResponseState) enabled() bool { return isInkWidgetEnabled(s.Widget()) }

func (s *inkResponseState) handleMouseEnter(*gestures.PointerEnterEvent) {
	s.hovering = true
	if s.enabled() {
		s.handleHoverChange()
	}
}

func (s *inkResponseState) handleMouseExit(*gestures.PointerExitEvent) {
	s.hovering = false
	// If the exit occurs after we've been disabled, we still want to take down
	// the highlights and run widget.onHover.
	s.handleHoverChange()
}

func (s *inkResponseState) handleHoverChange() { s.updateHighlight(highlightHover, s.hovering, true) }

func (s *inkResponseState) canRequestFocus() bool {
	if mode, ok := widgets.MediaQueryMaybeNavigationModeOf(s.Context()); ok && mode == widgets.NavigationModeDirectional {
		return true
	}
	return s.enabled() && s.Widget().o.CanRequestFocus.OrElse(true)
}

func (s *inkResponseState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	widgets.DebugCheckHasDirectionality(bc)
	s.KeepAliveBuild() // See AutomaticKeepAliveClientMixin.
	w := s.Widget()
	theme := ThemeOf(bc)
	// Each highlightable state is resolved separately to get the corresponding
	// color. For this resolution to be correct, the non-highlightable states
	// should be preserved.
	nonHighlightable := s.statesController().Value().
		Without(widgets.WidgetStateFocused).Without(widgets.WidgetStateHovered).Without(widgets.WidgetStatePressed)
	colorForType := func(t highlightType) ui.Color {
		switch t {
		case highlightFocus:
			if c, ok := s.overlayColorFor(nonHighlightable.With(widgets.WidgetStateFocused)); ok {
				return c
			}
			return w.o.FocusColor.OrElse(theme.FocusColor)
		case highlightHover:
			if c, ok := s.overlayColorFor(nonHighlightable.With(widgets.WidgetStateHovered)); ok {
				return c
			}
			return w.o.HoverColor.OrElse(theme.HoverColor)
		}
		// The pressed state triggers a ripple (ink splash), per the current
		// Material Design spec. A separate highlight is no longer used.
		if c, ok := s.overlayColorFor(nonHighlightable.With(widgets.WidgetStatePressed)); ok {
			return c
		}
		return w.o.HighlightColor.OrElse(theme.HighlightColor)
	}
	for t, h := range s.highlights {
		if h != nil {
			h.SetColor(colorForType(t))
		}
	}
	if s.currentSplash != nil {
		color, ok := s.overlayColorFor(s.statesController().Value())
		if !ok {
			color = w.o.SplashColor.OrElse(ThemeOf(bc).SplashColor)
		}
		s.currentSplash.AsInteractiveInkFeature().SetColor(color)
	}
	var mouseCursor services.MouseCursor = widgets.WidgetStateMouseCursorAdaptiveClickable
	if w.o.MouseCursor != nil {
		mouseCursor = w.o.MouseCursor
	}
	effectiveMouseCursor := widgets.WidgetStateResolveAs[services.MouseCursor](mouseCursor, s.statesController().Value())

	primaryEnabled := primaryInkButtonEnabled(w)
	secondaryEnabled := secondaryInkButtonEnabled(w)
	gd := widgets.GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, ExcludeFromSemantics: true}
	if primaryEnabled {
		gd.OnTapDown, gd.OnTapUp, gd.OnTap, gd.OnTapCancel = s.handleTapDown, s.handleTapUp, s.handleTap, s.handleTapCancel
	}
	if w.o.OnDoubleTap != nil {
		gd.OnDoubleTap = s.handleDoubleTap
	}
	if w.o.OnLongPress != nil {
		gd.OnLongPress = s.handleLongPress
	}
	if w.o.OnLongPressUp != nil {
		gd.OnLongPressUp = s.handleLongPressUp
	}
	if secondaryEnabled {
		gd.OnSecondaryTapDown, gd.OnSecondaryTapUp = s.handleSecondaryTapDown, s.handleSecondaryTapUp
		gd.OnSecondaryTap, gd.OnSecondaryTapCancel = s.handleSecondaryTap, s.handleSecondaryTapCancel
	}
	sem := widgets.SemanticsOpts{}
	if !w.o.ExcludeFromSemantics && w.o.OnTap != nil {
		sem.OnTap = s.simulateTap
	}
	if !w.o.ExcludeFromSemantics && w.o.OnLongPress != nil {
		sem.OnLongPress = s.simulateLongPress
	}
	return &parentInkResponseProvider{
		InheritedWidget: widgets.NewInheritedWidgetBase(
			widgets.NewActions(s.actionMap,
				widgets.NewFocus(
					widgets.NewMouseRegion(
						widgets.NewDefaultSelectionStyleMerge(
							widgets.NewSemantics(widgets.NewGestureDetector(w.child, gd), sem),
							widgets.DefaultSelectionStyleOpts{MouseCursor: effectiveMouseCursor}),
						widgets.MouseRegionOpts{Cursor: effectiveMouseCursor, OnEnter: s.handleMouseEnter, OnExit: s.handleMouseExit}),
					widgets.FocusOpts{FocusNode: w.o.FocusNode, CanRequestFocus: foundation.Some(s.canRequestFocus()),
						OnFocusChange: s.handleFocusUpdate, Autofocus: w.o.Autofocus}))),
		state: s,
	}
}

func errorInkResponseNoSize() error {
	return errors.New("InkResponseNoSize", "InkResponse must be done with layout before starting a splash.")
}
