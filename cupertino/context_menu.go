package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	contextMenuOpenScale                = 1.15
	contextMenuMinScaleFactor           = 1.02
	contextMenuPreviewBorderRadiusRatio = 12.0
	contextMenuModalPopupTransition     = 335 * time.Millisecond
	contextMenuPreviewLongPressTimeout  = 800 * time.Millisecond
	// contextMenuAnimationDuration is the length of the whole opening, in milliseconds: the long press and the
	// popup that follows it.
	contextMenuAnimationDuration = 800 + 335
)

// CupertinoContextMenuKOpenBorderRadius is the radius of the corners of the child when the menu is open.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.kOpenBorderRadius
const CupertinoContextMenuKOpenBorderRadius = contextMenuPreviewBorderRadiusRatio

// CupertinoContextMenuKEndBoxShadow is the shadow of the child when the menu is open.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.kEndBoxShadow
func CupertinoContextMenuKEndBoxShadow() []painting.BoxShadow {
	return []painting.BoxShadow{{
		Shadow: ui.NewShadow(ui.ShadowOpts{Color: ptrColor(ui.NewColor(0x40000000)), BlurRadius: 10.0}), SpreadRadius: 0.5}}
}

// CupertinoContextMenuAnimationOpensAt is the point of the opening animation at which the menu opens: the share of
// the long press in the whole animation.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.animationOpensAt
var CupertinoContextMenuAnimationOpensAt = float64(contextMenuPreviewLongPressTimeout.Milliseconds()) / contextMenuAnimationDuration

// CupertinoContextMenuKBackgroundColor is the color of the sheet of the actions.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.kBackgroundColor
var CupertinoContextMenuKBackgroundColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFF1F1F1), ui.NewColor(0xFF212122))

var contextMenuBorderColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFA9A9AF), ui.NewColor(0xFF57585A))

// CupertinoContextMenuBuilder builds the child of a context menu for the progress of its opening: the animation
// goes from 0 to 1 while the child grows, and the menu is open at CupertinoContextMenuAnimationOpensAt.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenuBuilder
type CupertinoContextMenuBuilder func(bc widgets.BuildContext, anim animation.Animation[float64]) widgets.Widget

// CupertinoContextMenuOpts are the optional parameters of the constructors of CupertinoContextMenu.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.new
type CupertinoContextMenuOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// EnableHapticFeedback makes the device vibrate when the menu opens.
	EnableHapticFeedback bool
}

// CupertinoContextMenu is the menu that opens when its child is long pressed, as iOS shows it: the child grows,
// the screen is blurred and the actions appear beside the child.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu
type CupertinoContextMenu struct {
	widgets.StatefulWidget
	builder CupertinoContextMenuBuilder
	child   widgets.Widget
	actions []widgets.Widget
	o       CupertinoContextMenuOpts
}

// NewCupertinoContextMenu creates a context menu of the actions that grows the child while it is pressed. In debug
// builds it panics with the tag CupertinoContextMenuNoActions when there are no actions.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.new
func NewCupertinoContextMenu(actions []widgets.Widget, child widgets.Widget, opts ...CupertinoContextMenuOpts) *CupertinoContextMenu {
	if foundation.DebugMode && len(actions) == 0 {
		panic(errors.New(tagCupertinoContextMenuNoActions, "a CupertinoContextMenu needs at least one action"))
	}
	o := firstOf(opts)
	return &CupertinoContextMenu{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), actions: actions, child: child, o: o,
		builder: func(widgets.BuildContext, animation.Animation[float64]) widgets.Widget { return child }}
}

// NewCupertinoContextMenuBuilder creates a context menu of the actions whose child the builder makes for the
// progress of the opening. In debug builds it panics with the tag CupertinoContextMenuNoActions when there are no
// actions.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.builder
func NewCupertinoContextMenuBuilder(actions []widgets.Widget, builder CupertinoContextMenuBuilder, opts ...CupertinoContextMenuOpts) *CupertinoContextMenu {
	if foundation.DebugMode && len(actions) == 0 {
		panic(errors.New(tagCupertinoContextMenuNoActions, "a CupertinoContextMenu needs at least one action"))
	}
	o := firstOf(opts)
	return &CupertinoContextMenu{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), actions: actions, builder: builder, o: o}
}

// Builder is the function that builds the child.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.builder
func (w *CupertinoContextMenu) Builder() CupertinoContextMenuBuilder { return w.builder }

// Child is the child, if the menu was made with one.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.child
func (w *CupertinoContextMenu) Child() widgets.Widget { return w.child }

// Actions are the actions of the menu.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.actions
func (w *CupertinoContextMenu) Actions() []widgets.Widget { return w.actions }

// EnableHapticFeedback is whether the device vibrates when the menu opens.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.enableHapticFeedback
func (w *CupertinoContextMenu) EnableHapticFeedback() bool { return w.o.EnableHapticFeedback }

// CreateState creates the state.
//
// Flutter: cupertino/context_menu.dart CupertinoContextMenu.createState
func (w *CupertinoContextMenu) CreateState() widgets.State { return &cupertinoContextMenuState{} }

type contextMenuLocation int

const (
	contextMenuLocationCenter contextMenuLocation = iota
	contextMenuLocationLeft
	contextMenuLocationRight
)

// contextMenuRect is the rectangle of the render box of the context in global coordinates.
//
// Flutter: cupertino/context_menu.dart _getRect
func contextMenuRect(bc widgets.BuildContext) ui.Rect {
	box := bc.FindRenderObject().(rendering.Box)
	bounds := box.PaintBounds()
	return ui.NewRectFromPoints(box.LocalToGlobal(bounds.TopLeft(), nil), box.LocalToGlobal(bounds.BottomRight(), nil))
}

type cupertinoContextMenuState struct {
	widgets.StateOf[*CupertinoContextMenu]
	widgets.TickerProvider
	childKey            *widgets.GlobalKey[widgets.State]
	childHidden         bool
	openController      *animation.AnimationController
	decoyChildEndRect   foundation.Opt[ui.Rect]
	scaleFactor         float64
	lastOverlayEntry    *widgets.OverlayEntry
	route               *contextMenuRoute
	midpoint            float64
	tapRecognizer       *gestures.TapGestureRecognizer
	openListener        foundation.ListenerHandle
	listeningToOpen     bool
	routeStatusListener animation.StatusListenerHandle
}

func (s *cupertinoContextMenuState) InitState() {
	s.childKey = widgets.NewGlobalKey[widgets.State]()
	s.midpoint = CupertinoContextMenuAnimationOpensAt / 2
	s.openController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(contextMenuPreviewLongPressTimeout), UpperBound: foundation.Some(CupertinoContextMenuAnimationOpensAt)})
	s.openController.AddStatusListener(s.onDecoyAnimationStatusChange)
	s.tapRecognizer = gestures.NewTapGestureRecognizer(widgets.WidgetsBindingOf(s.Context()).Gestures(), gestures.TapGestureRecognizerOpts{})
	s.tapRecognizer.OnTapCancel = s.onTapCompleted
	s.tapRecognizer.OnTapDown = s.onTapDown
	s.tapRecognizer.OnTapUp = func(gestures.TapUpDetails) { s.onTapCompleted() }
	s.tapRecognizer.OnTap = s.onTapCompleted
}

func (s *cupertinoContextMenuState) addOpenListener() {
	s.removeOpenListener()
	s.openListener = s.openController.AddListener(s.listenerCallback)
	s.listeningToOpen = true
}

func (s *cupertinoContextMenuState) removeOpenListener() {
	if s.listeningToOpen {
		s.openController.RemoveListener(s.openListener)
		s.listeningToOpen = false
	}
}

func (s *cupertinoContextMenuState) listenerCallback() {
	if s.openController.Status() != animation.AnimationStatusReverse && s.openController.Value() >= s.midpoint {
		if s.Widget().o.EnableHapticFeedback {
			widgets.WidgetsBindingOf(s.Context()).HapticFeedback().HeavyImpact()
		}
		s.tapRecognizer.Resolve(gestures.GestureAccepted)
		s.removeOpenListener()
	}
}

func (s *cupertinoContextMenuState) contextMenuLocation() contextMenuLocation {
	childRect := contextMenuRect(s.childKey.CurrentContext())
	center := widgets.MediaQueryWidthOf(s.Context()) / 2
	centerDividesChild := childRect.Left < center && childRect.Right > center
	distanceFromCenter := math.Abs(center - childRect.Center().Dx)
	if centerDividesChild && distanceFromCenter <= childRect.Width()/4 {
		return contextMenuLocationCenter
	}
	if childRect.Center().Dx > center {
		return contextMenuLocationRight
	}
	return contextMenuLocationLeft
}

// contextMenuScaleFactor is how much the child can grow before it leaves the screen, between the minimum and the
// open scale.
//
// Flutter: cupertino/context_menu.dart _CupertinoContextMenuState._getScaleFactor
func contextMenuScaleFactor(childRect ui.Rect, padding painting.EdgeInsets, size ui.Size) float64 {
	leftMaxScale := 2 * (childRect.Center().Dx - padding.Left) / childRect.Width()
	topMaxScale := 2 * (childRect.Center().Dy - padding.Top) / childRect.Height()
	rightMaxScale := 2 * (size.Width - padding.Right - childRect.Center().Dx) / childRect.Width()
	bottomMaxScale := 2 * (size.Height - padding.Bottom - childRect.Center().Dy) / childRect.Height()
	minWidth := math.Min(leftMaxScale, rightMaxScale)
	minHeight := math.Min(topMaxScale, bottomMaxScale)
	return ui.ClampDouble(math.Min(minWidth, minHeight), contextMenuMinScaleFactor, contextMenuOpenScale)
}

func contextMenuDefaultPreviewBuilder(anim animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return widgets.NewFittedBox(
		widgets.NewClipRSuperellipse(child, widgets.ClipRSuperellipseOpts{
			BorderRadius: painting.NewBorderRadiusCircular(contextMenuPreviewBorderRadiusRatio * anim.Value())}),
		widgets.FittedBoxOpts{Fit: painting.BoxFitCover})
}

func (s *cupertinoContextMenuState) openContextMenu() {
	s.SetState(func() { s.childHidden = true })
	w := s.Widget()
	bc := s.Context()
	r := &contextMenuRoute{
		actions: w.actions, barrierLabel: CupertinoLocalizationsOf(bc).MenuDismissLabel(),
		location: s.contextMenuLocation(), previousChildRect: s.decoyChildEndRect.Value(), scaleFactor: s.scaleFactor,
		postFrame: func(f func()) {
			widgets.WidgetsBindingOf(bc).Scheduler().AddPostFrameCallback(func(time.Duration) { f() })
		},
		builder: func(bc widgets.BuildContext, anim animation.Animation[float64]) widgets.Widget {
			if w.child == nil {
				local := animation.NewTween(CupertinoContextMenuAnimationOpensAt, 1.0).Animate(anim)
				return w.builder(bc, local)
			}
			return contextMenuDefaultPreviewBuilder(anim, w.child)
		},
	}
	r.init()
	s.route = r
	widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: true}).Push(r)
	s.routeStatusListener = r.Animation().AddStatusListener(s.routeAnimationStatusListener)
}

func (s *cupertinoContextMenuState) removeContextMenuDecoy() {
	widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if s.Mounted() {
			s.closeContextMenu()
			s.openController.Reset()
		}
	})
}

func (s *cupertinoContextMenuState) closeContextMenu() {
	if s.lastOverlayEntry != nil {
		s.lastOverlayEntry.Remove()
		s.lastOverlayEntry.Dispose()
		s.lastOverlayEntry = nil
	}
}

func (s *cupertinoContextMenuState) onDecoyAnimationStatusChange(status animation.AnimationStatus) {
	switch status {
	case animation.AnimationStatusDismissed:
		if s.route == nil && s.Mounted() {
			s.SetState(func() { s.childHidden = false })
		}
		s.closeContextMenu()
	case animation.AnimationStatusCompleted:
		s.openContextMenu()
		s.removeContextMenuDecoy()
	case animation.AnimationStatusForward, animation.AnimationStatusReverse:
		if !widgets.ModalRouteOf(s.Context()).IsCurrent() {
			s.removeContextMenuDecoy()
		}
	}
}

func (s *cupertinoContextMenuState) routeAnimationStatusListener(status animation.AnimationStatus) {
	if !status.IsDismissed() {
		return
	}
	if s.Mounted() {
		s.SetState(func() { s.childHidden = false })
	}
	s.route.Animation().RemoveStatusListener(s.routeStatusListener)
	s.route = nil
}

func (s *cupertinoContextMenuState) onTapCompleted() {
	s.removeOpenListener()
	if s.openController.IsAnimating() && s.openController.Value() < s.midpoint {
		s.openController.Reverse()
	}
}

func (s *cupertinoContextMenuState) onTapDown(gestures.TapDownDetails) {
	s.addOpenListener()
	s.SetState(func() { s.childHidden = true })
	bc := s.Context()
	childRect := contextMenuRect(s.childKey.CurrentContext())
	s.scaleFactor = contextMenuScaleFactor(childRect, widgets.MediaQueryPaddingOf(bc), widgets.MediaQuerySizeOf(bc))
	endRect := ui.NewRectFromCenter(childRect.Center(), childRect.Width()*s.scaleFactor, childRect.Height()*s.scaleFactor)
	s.decoyChildEndRect = foundation.Some(endRect)
	w := s.Widget()
	s.lastOverlayEntry = widgets.NewOverlayEntry(func(widgets.BuildContext) widgets.Widget {
		return &contextMenuDecoyChild{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}),
			beginRect: childRect, controller: s.openController, endRect: endRect, builder: w.builder, child: w.child}
	})
	widgets.OverlayOf(bc, widgets.OverlayOfOpts{RootOverlay: true, DebugRequiredFor: w}).Insert(s.lastOverlayEntry)
	s.openController.Forward()
}

// Build builds the child, hidden while the menu is open.
//
// Flutter: cupertino/context_menu.dart _CupertinoContextMenuState.build
func (s *cupertinoContextMenuState) Build(bc widgets.BuildContext) widgets.Widget {
	var cursor services.MouseCursor = services.MouseCursorDefer
	if foundation.IsWeb {
		cursor = services.SystemMouseCursors.Click
	}
	return widgets.NewMouseRegion(widgets.NewListener(
		widgets.NewTickerMode(!s.childHidden, widgets.NewVisibilityMaintain(s.Widget().builder(bc, s.openController),
			widgets.VisibilityOpts{Key: s.childKey, Visible: foundation.Some(!s.childHidden)})),
		widgets.ListenerOpts{OnPointerDown: func(e *gestures.PointerDownEvent) { s.tapRecognizer.AddPointer(e) }}),
		widgets.MouseRegionOpts{Cursor: cursor})
}

func (s *cupertinoContextMenuState) Dispose() {
	// Deviation: the status listener goes first, because disposing the recognizer cancels the tap, which reverses the
	// controller, and its status listener must not run on the state that is going away.
	s.openController.ClearStatusListeners()
	s.closeContextMenu()
	s.tapRecognizer.Dispose()
	s.removeOpenListener()
	s.openController.Dispose()
}

// contextMenuDecoyChild is the copy of the child that grows in the overlay while the child is pressed.
//
// Flutter: cupertino/context_menu.dart _DecoyChild
type contextMenuDecoyChild struct {
	widgets.StatefulWidget
	beginRect  ui.Rect
	controller *animation.AnimationController
	endRect    ui.Rect
	child      widgets.Widget
	builder    CupertinoContextMenuBuilder
}

func (w *contextMenuDecoyChild) CreateState() widgets.State { return &contextMenuDecoyChildState{} }

type contextMenuDecoyChildState struct {
	widgets.StateOf[*contextMenuDecoyChild]
	rect                        animation.Animation[foundation.Opt[ui.Rect]]
	boxDecoration               animation.Animation[painting.Decoration]
	boxDecorationCurvedAnimator *animation.CurvedAnimation
}

func (s *contextMenuDecoyChildState) InitState() {
	w := s.Widget()
	const (
		beginPause          = 1.0
		openAnimationLength = 5.0
		totalOpenAnimation  = beginPause + openAnimationLength
	)
	endPause := (totalOpenAnimation*contextMenuAnimationDuration)/float64(contextMenuPreviewLongPressTimeout.Milliseconds()) - totalOpenAnimation
	rectTween := func(begin, end ui.Rect, curve animation.Curve) animation.Animatable[foundation.Opt[ui.Rect]] {
		return animation.NewRectTween(foundation.Some(begin), foundation.Some(end)).Chain(animation.NewCurveTween(curve))
	}
	s.rect = animation.NewTweenSequence([]animation.TweenSequenceItem[foundation.Opt[ui.Rect]]{
		{Tween: rectTween(w.beginRect, w.beginRect, animation.Curves.Linear), Weight: beginPause},
		{Tween: rectTween(w.beginRect, w.endRect, animation.Curves.EaseOutSine), Weight: openAnimationLength},
		{Tween: rectTween(w.endRect, w.endRect, animation.Curves.Linear), Weight: endPause},
	}).Animate(w.controller)
	s.boxDecorationCurvedAnimator = animation.NewCurvedAnimation(w.controller, animation.NewInterval(0.0, CupertinoContextMenuAnimationOpensAt))
	s.boxDecoration = widgets.NewDecorationTween(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{BoxShadow: []painting.BoxShadow{}}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{BoxShadow: CupertinoContextMenuKEndBoxShadow()})).Animate(s.boxDecorationCurvedAnimator)
}

func (s *contextMenuDecoyChildState) Dispose() { s.boxDecorationCurvedAnimator.Dispose() }

func (s *contextMenuDecoyChildState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	build := func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
		return widgets.NewPositionedFromRect(s.rect.Value().Value(), widgets.NewContainer(w.child, widgets.ContainerOpts{Decoration: s.boxDecoration.Value()}))
	}
	if w.child == nil {
		build = func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
			return widgets.NewPositionedFromRect(s.rect.Value().Value(), w.builder(bc, w.controller))
		}
	}
	return widgets.NewStack([]widgets.Widget{widgets.NewAnimatedBuilder(w.controller, build, nil)})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoContextMenuNoActions = "CupertinoContextMenuNoActions"
