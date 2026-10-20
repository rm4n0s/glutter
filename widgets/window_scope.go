package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// windowAspect is a part of the state of a window that a dependent of a
// WindowScope can depend on.
//
// Flutter: widgets/_window.dart _WindowControllerAspect
type windowAspect int

const (
	windowAspectContentSize windowAspect = iota
	windowAspectTitle
	windowAspectActivated
	windowAspectMaximized
	windowAspectMinimized
	windowAspectFullscreen
)

// WindowScope gives the descendants access to the controller of the window they
// are rendered in, and lets them depend on one part of its state: the content
// size, the title, or whether it is activated, maximized, minimized or
// fullscreen. The RegularWindow and DialogWindow widgets place it, and RunApp does
// for the first window.
//
// Deviation: Flutter's notifies its dependents only when the controller object
// changes, which never happens, so they would never update; this one keeps the
// state of the window it was made with and notifies when that changes (SPEC
// §12.10).
//
// Flutter: widgets/_window.dart WindowScope
type WindowScope struct {
	InheritedModel
	controller BaseWindowController
	state      windowSnapshot
}

// NewWindowScope creates a scope for the controller around the child.
//
// Flutter: widgets/_window.dart WindowScope.new
func NewWindowScope(controller BaseWindowController, child Widget, opts ...WidgetOpts) *WindowScope {
	return &WindowScope{
		InheritedModel: NewInheritedModelBase(child, opts...),
		controller:     controller,
		state:          controller.windowBase().snapshot(),
	}
}

// Controller is the controller of the window.
//
// Flutter: widgets/_window.dart WindowScope.controller
func (w *WindowScope) Controller() BaseWindowController { return w.controller }

// UpdateShouldNotify is whether the controller or the state of the window changed.
//
// Flutter: widgets/_window.dart WindowScope.updateShouldNotify
func (w *WindowScope) UpdateShouldNotify(old Widget) bool {
	o := old.(*WindowScope)
	return w.controller != o.controller || w.state != o.state
}

// UpdateShouldNotifyDependent is whether a part of the state that a dependent asked
// for changed.
//
// Flutter: widgets/_window.dart WindowScope.updateShouldNotifyDependent
func (w *WindowScope) UpdateShouldNotifyDependent(old Widget, dependencies *AspectSet) bool {
	o := old.(*WindowScope)
	for _, dep := range dependencies.Items() {
		aspect, ok := dep.(windowAspect)
		if !ok {
			continue
		}
		changed := false
		switch aspect {
		case windowAspectContentSize:
			changed = w.state.size != o.state.size
		case windowAspectTitle:
			changed = w.state.title != o.state.title
		case windowAspectActivated:
			changed = w.state.activated != o.state.activated
		case windowAspectMaximized:
			changed = w.state.maximized != o.state.maximized
		case windowAspectMinimized:
			changed = w.state.minimized != o.state.minimized
		case windowAspectFullscreen:
			changed = w.state.fullscreen != o.state.fullscreen
		}
		if changed {
			return true
		}
	}
	return false
}

func windowScopeOf(bc BuildContext, aspect any) *WindowScope {
	s, ok := DependOnModel[*WindowScope](bc, aspect)
	if !ok {
		return nil
	}
	return s
}

// WindowScopeOf is the controller of the window that hosts the context. It panics
// with the tag WindowScopeMissing when there is none: the context is not below a
// RegularWindow, a DialogWindow, or the first window of the app.
//
// Flutter: widgets/_window.dart WindowScope.of
func WindowScopeOf(bc BuildContext) BaseWindowController {
	return mustWindowScope(bc, nil).controller
}

// WindowScopeMaybeOf is WindowScopeOf that returns nil when there is no scope.
//
// Flutter: widgets/_window.dart WindowScope.maybeOf
func WindowScopeMaybeOf(bc BuildContext) BaseWindowController {
	if s := windowScopeOf(bc, nil); s != nil {
		return s.controller
	}
	return nil
}

func mustWindowScope(bc BuildContext, aspect any) *WindowScope {
	s := windowScopeOf(bc, aspect)
	if s == nil {
		panic(errors.New("WindowScopeMissing",
			"no WindowScope found in the context; WindowScopeOf needs a RegularWindow or DialogWindow widget above the context, or the first window of the app",
			"context", bc.(Element).String()))
	}
	return s
}

// WindowScopeContentSizeOf is the content size of the window that hosts the
// context, in logical pixels; the context depends on it only.
//
// Flutter: widgets/_window.dart WindowScope.contentSizeOf
func WindowScopeContentSizeOf(bc BuildContext) ui.Size {
	return mustWindowScope(bc, windowAspectContentSize).state.size
}

// WindowScopeMaybeContentSizeOf is WindowScopeContentSizeOf that returns false when
// there is no scope.
//
// Flutter: widgets/_window.dart WindowScope.maybeContentSizeOf
func WindowScopeMaybeContentSizeOf(bc BuildContext) (ui.Size, bool) {
	if s := windowScopeOf(bc, windowAspectContentSize); s != nil {
		return s.state.size, true
	}
	return ui.Size{}, false
}

// WindowScopeTitleOf is the title of the window that hosts the context.
//
// Flutter: widgets/_window.dart WindowScope.titleOf
func WindowScopeTitleOf(bc BuildContext) string {
	return mustWindowScope(bc, windowAspectTitle).state.title
}

// WindowScopeMaybeTitleOf is WindowScopeTitleOf that returns false when there is
// no scope.
//
// Flutter: widgets/_window.dart WindowScope.maybeTitleOf
func WindowScopeMaybeTitleOf(bc BuildContext) (string, bool) {
	if s := windowScopeOf(bc, windowAspectTitle); s != nil {
		return s.state.title, true
	}
	return "", false
}

// WindowScopeIsActivatedOf is whether the window that hosts the context is
// activated.
//
// Flutter: widgets/_window.dart WindowScope.isActivatedOf
func WindowScopeIsActivatedOf(bc BuildContext) bool {
	return mustWindowScope(bc, windowAspectActivated).state.activated
}

// WindowScopeMaybeIsActivatedOf is WindowScopeIsActivatedOf that returns false
// as its second result when there is no scope.
//
// Flutter: widgets/_window.dart WindowScope.maybeIsActivatedOf
func WindowScopeMaybeIsActivatedOf(bc BuildContext) (activated, ok bool) {
	if s := windowScopeOf(bc, windowAspectActivated); s != nil {
		return s.state.activated, true
	}
	return false, false
}

// WindowScopeIsMinimizedOf is whether the window that hosts the context is
// minimized.
//
// Flutter: widgets/_window.dart WindowScope.isMinimizedOf
func WindowScopeIsMinimizedOf(bc BuildContext) bool {
	return mustWindowScope(bc, windowAspectMinimized).state.minimized
}

// WindowScopeMaybeIsMinimizedOf is WindowScopeIsMinimizedOf that returns false as
// its second result when there is no scope.
//
// Flutter: widgets/_window.dart WindowScope.maybeIsMinimizedOf
func WindowScopeMaybeIsMinimizedOf(bc BuildContext) (minimized, ok bool) {
	if s := windowScopeOf(bc, windowAspectMinimized); s != nil {
		return s.state.minimized, true
	}
	return false, false
}

// WindowScopeIsMaximizedOf is whether the window that hosts the context is
// maximized.
//
// Flutter: widgets/_window.dart WindowScope.isMaximizedOf
func WindowScopeIsMaximizedOf(bc BuildContext) bool {
	return mustWindowScope(bc, windowAspectMaximized).state.maximized
}

// WindowScopeMaybeIsMaximizedOf is WindowScopeIsMaximizedOf that returns false as
// its second result when there is no scope.
//
// Flutter: widgets/_window.dart WindowScope.maybeIsMaximizedOf
func WindowScopeMaybeIsMaximizedOf(bc BuildContext) (maximized, ok bool) {
	if s := windowScopeOf(bc, windowAspectMaximized); s != nil {
		return s.state.maximized, true
	}
	return false, false
}

// WindowScopeIsFullscreenOf is whether the window that hosts the context is
// fullscreen.
//
// Flutter: widgets/_window.dart WindowScope.isFullscreenOf
func WindowScopeIsFullscreenOf(bc BuildContext) bool {
	return mustWindowScope(bc, windowAspectFullscreen).state.fullscreen
}

// WindowScopeMaybeIsFullscreenOf is WindowScopeIsFullscreenOf that returns false
// as its second result when there is no scope.
//
// Flutter: widgets/_window.dart WindowScope.maybeIsFullscreenOf
func WindowScopeMaybeIsFullscreenOf(bc BuildContext) (fullscreen, ok bool) {
	if s := windowScopeOf(bc, windowAspectFullscreen); s != nil {
		return s.state.fullscreen, true
	}
	return false, false
}

// windowContent is the widget of a window: it rebuilds when the controller
// changes, and shows the child in the root view of the controller below a
// WindowScope.
func windowContent(controller BaseWindowController, child Widget, appDecorations bool) Widget {
	return NewListenableBuilder(controller, func(_ BuildContext, child Widget) Widget {
		content := child
		if !appDecorations {
			// Shows a title bar when the window has none (SPEC §8.2).
			content = NewWindowDecoration(content)
		}
		return NewWindowScope(controller, NewView(controller.RootView(), content))
	}, child)
}

// RegularWindow shows its child in the regular window of a controller. When the
// widget leaves the tree the window stays, until the controller destroys it.
// Descendants reach the controller with WindowScopeOf.
//
// Flutter: widgets/_window.dart RegularWindow
type RegularWindow struct {
	StatelessWidget
	controller *RegularWindowController
	child      Widget
}

// RegularWindowOpts are the optional parameters of NewRegularWindow.
//
// Flutter: widgets/_window.dart RegularWindow.new
type RegularWindowOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// NewRegularWindow creates the widget that shows the child in the window of the
// controller.
//
// Flutter: widgets/_window.dart RegularWindow.new
func NewRegularWindow(controller *RegularWindowController, child Widget, opts ...RegularWindowOpts) *RegularWindow {
	o := firstOpts(opts)
	return &RegularWindow{StatelessWidget: NewStatelessWidgetBase(WidgetOpts(o)), controller: controller, child: child}
}

// Controller is the controller of the window.
//
// Flutter: widgets/_window.dart RegularWindow.controller
func (w *RegularWindow) Controller() *RegularWindowController { return w.controller }

// Child is the content of the window.
//
// Flutter: widgets/_window.dart RegularWindow.child
func (w *RegularWindow) Child() Widget { return w.child }

// Build shows the child in the view of the window.
//
// Flutter: widgets/_window.dart RegularWindow.build
func (w *RegularWindow) Build(BuildContext) Widget {
	return windowContent(w.controller, w.child, w.controller.appDecorations)
}

// DialogWindow shows its child in the dialog window of a controller. When the
// widget leaves the tree the window stays, until the controller destroys it.
//
// Flutter: widgets/_window.dart DialogWindow
type DialogWindow struct {
	StatelessWidget
	controller *DialogWindowController
	child      Widget
}

// DialogWindowOpts are the optional parameters of NewDialogWindow.
//
// Flutter: widgets/_window.dart DialogWindow.new
type DialogWindowOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// NewDialogWindow creates the widget that shows the child in the dialog window of
// the controller.
//
// Flutter: widgets/_window.dart DialogWindow.new
func NewDialogWindow(controller *DialogWindowController, child Widget, opts ...DialogWindowOpts) *DialogWindow {
	o := firstOpts(opts)
	return &DialogWindow{StatelessWidget: NewStatelessWidgetBase(WidgetOpts(o)), controller: controller, child: child}
}

// Controller is the controller of the window.
//
// Flutter: widgets/_window.dart DialogWindow.controller
func (w *DialogWindow) Controller() *DialogWindowController { return w.controller }

// Child is the content of the window.
//
// Flutter: widgets/_window.dart DialogWindow.child
func (w *DialogWindow) Child() Widget { return w.child }

// Build shows the child in the view of the window.
//
// Flutter: widgets/_window.dart DialogWindow.build
func (w *DialogWindow) Build(BuildContext) Widget {
	return windowContent(w.controller, w.child, w.controller.appDecorations)
}
