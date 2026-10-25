package widgets

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// BaseWindowController is the base of the window controllers. A controller
// belongs to one window and its root view. When the window is destroyed, by the
// app or by the platform, the controller is no longer valid. The controllers are
// the RegularWindowController and the DialogWindowController.
//
// Deviation: the interface is sealed, as Flutter's class is; the pinned
// Flutter also has controllers for tooltips, popups and satellites, which are
// deferred (SPEC §8.7, §12.10).
//
// Flutter: widgets/_window.dart BaseWindowController
type BaseWindowController interface {
	foundation.Listenable
	// ContentSize is the size of the drawable area of the window, in logical
	// pixels. It can differ from the size that was asked for, and from the size of
	// the window when the window has a title bar and borders.
	//
	// Flutter: widgets/_window.dart BaseWindowController.contentSize
	ContentSize() ui.Size
	// Destroy destroys the window. It does nothing when the window is destroyed
	// already.
	//
	// Flutter: widgets/_window.dart BaseWindowController.destroy
	Destroy()
	// RootView is the root view of the window, unique to each window.
	//
	// Flutter: widgets/_window.dart BaseWindowController.rootView
	RootView() *ui.View

	windowBase() *windowControllerBase
}

// WindowController is what every window that an app can reach with WindowScopeOf can
// do: a regular window (the first one included) and a dialog. WindowControllerOf gives
// it for the window that hosts a context, so the title, the activation and the
// minimized state are reachable without an assertion. What only a regular window can
// do (maximize, full screen) is in MaximizableWindowController: a controller is one
// when the window can be maximized, and a type assertion asks.
//
// Deviation: Flutter's RegularWindowController and DialogWindowController declare
// these members each, and BaseWindowController has none of them; the interface is
// Glutter's, so that WindowScopeOf's result needs no assertion to the concrete type
// (SPEC §12.10).
//
// Flutter: no counterpart
type WindowController interface {
	BaseWindowController
	// Title is the title of the window.
	//
	// Flutter: widgets/_window.dart RegularWindowController.title
	Title() string
	// SetTitle asks for a new title of the window.
	//
	// Flutter: widgets/_window.dart RegularWindowController.setTitle
	SetTitle(title string)
	// Activate asks for the window to be shown and focused.
	//
	// Flutter: widgets/_window.dart RegularWindowController.activate
	Activate()
	// IsActivated is whether the window is focused.
	//
	// Flutter: widgets/_window.dart RegularWindowController.isActivated
	IsActivated() bool
	// SetMinimized asks for the window to be minimized, or shown again.
	//
	// Flutter: widgets/_window.dart RegularWindowController.setMinimized
	SetMinimized(minimized bool)
	// IsMinimized is whether the window is minimized.
	//
	// Flutter: widgets/_window.dart RegularWindowController.isMinimized
	IsMinimized() bool
}

// MaximizableWindowController is a WindowController that can be maximized and made
// fullscreen: the regular windows. A dialog is not one.
//
// Flutter: no counterpart
type MaximizableWindowController interface {
	WindowController
	// SetMaximized asks for the window to be maximized, or restored.
	//
	// Flutter: widgets/_window.dart RegularWindowController.setMaximized
	SetMaximized(maximized bool)
	// IsMaximized is whether the window is maximized.
	//
	// Flutter: widgets/_window.dart RegularWindowController.isMaximized
	IsMaximized() bool
	// SetFullscreen asks for the window to be fullscreen, or to go back.
	//
	// Flutter: widgets/_window.dart RegularWindowController.setFullscreen
	SetFullscreen(fullscreen bool)
	// IsFullscreen is whether the window is fullscreen.
	//
	// Flutter: widgets/_window.dart RegularWindowController.isFullscreen
	IsFullscreen() bool
}

var (
	_ WindowController            = (*RegularWindowController)(nil)
	_ WindowController            = (*DialogWindowController)(nil)
	_ MaximizableWindowController = (*RegularWindowController)(nil)
)

// windowSnapshot is what a WindowScope tells its dependents about a window.
type windowSnapshot struct {
	size                                                ui.Size
	title                                               string
	activated, maximized, minimized, fullscreen, hidden bool
}

// windowControllerBase is what the controllers of all the kinds of windows have
// in common: the view, the state the platform reported and the requests.
type windowControllerBase struct {
	foundation.ChangeNotifier
	owner     *WindowingOwner
	self      BaseWindowController
	view      *ui.View
	parent    BaseWindowController
	title     string
	minimized bool
	// reportedMinimized is whether the platform has ever said the window is minimized,
	// so that it is trusted when it says the window is not.
	reportedMinimized bool
	// hidden is whether the app took the window off the screen (SetHidden).
	hidden    bool
	destroyed bool
	implicit  bool
	// dialog is true for a dialog, which is never maximized or fullscreen.
	dialog bool
	// appDecorations is whether the app draws the title bar itself.
	appDecorations bool
	last           windowSnapshot
	onClose        func()   // asks the delegate whether the window may close
	onDestroy      func()   // tells the delegate that the window is gone
	children       []func() // destroy the windows that depend on this one
}

func (c *windowControllerBase) windowBase() *windowControllerBase { return c }

// RootView is the view of the window.
//
// Flutter: widgets/_window.dart BaseWindowController.rootView
func (c *windowControllerBase) RootView() *ui.View { return c.view }

// ContentSize is the size of the content of the window in logical pixels.
//
// Flutter: widgets/_window.dart BaseWindowController.contentSize
func (c *windowControllerBase) ContentSize() ui.Size {
	return c.view.PhysicalSize().Div(c.view.DevicePixelRatio())
}

// Title is the title of the window. The platform decides what is shown; this is
// the title that was asked for, or the one the platform reported.
//
// Flutter: widgets/_window.dart RegularWindowController.title
func (c *windowControllerBase) Title() string { return c.title }

// IsActivated is whether the window is focused and can receive user input.
//
// Flutter: widgets/_window.dart RegularWindowController.isActivated
func (c *windowControllerBase) IsActivated() bool { return c.view.Window().Activated }

// IsMinimized is whether the window is minimized. Where the platform says so (X11)
// it is what the window manager did; otherwise (Wayland, whose xdg-shell does not
// say) it is what the controller last asked for, until the window is activated.
//
// Flutter: widgets/_window.dart RegularWindowController.isMinimized
func (c *windowControllerBase) IsMinimized() bool { return c.minimized }

// SetSize asks for a new content size of the window, in logical pixels. The
// platform may clamp it to the constraints of the window, or ignore it.
//
// Flutter: widgets/_window.dart RegularWindowController.setSize
func (c *windowControllerBase) SetSize(size ui.Size) {
	if c.destroyed {
		return
	}
	c.owner.b.d.SetWindowSize(c.view.ViewID(), size)
}

// SetConstraints asks for new limits on the size of the window. The platform may
// resize the window to satisfy them, or ignore them.
//
// Flutter: widgets/_window.dart RegularWindowController.setConstraints
func (c *windowControllerBase) SetConstraints(constraints rendering.BoxConstraints) {
	if c.destroyed {
		return
	}
	min, max := constraintSizes(constraints)
	c.owner.b.d.SetWindowConstraints(c.view.ViewID(), min, max)
}

// SetTitle asks for a new title of the window.
//
// Flutter: widgets/_window.dart RegularWindowController.setTitle
func (c *windowControllerBase) SetTitle(title string) {
	if c.destroyed {
		return
	}
	c.title = title
	c.owner.b.d.SetWindowTitle(c.view.ViewID(), title)
	c.refresh()
}

// SetIcon sets the icon of the window; an empty icon removes it. It fails with
// WindowIconInvalid when a picture of the icon is wrong, and with the platform's
// error when it cannot use the icon. A platform that has no window icons ignores it.
//
// Flutter: no counterpart
func (c *windowControllerBase) SetIcon(icon ui.WindowIcon) error {
	if c.destroyed {
		return nil
	}
	return c.view.SetIcon(icon)
}

// SetContentType tells the compositor what the window shows (a game, a video or
// pictures), so that it can pick the refresh rate and the latency that suit it.
//
// Flutter: no counterpart
func (c *windowControllerBase) SetContentType(contentType ui.ContentType) {
	if !c.destroyed {
		c.view.SetContentType(contentType)
	}
}

// Activate asks for the window to be shown at its size and position, out of the
// minimized state, and to get the focus. The platform decides whether it gets the
// focus and the top of the stack of windows. A hidden window is shown first.
//
// Flutter: widgets/_window.dart RegularWindowController.activate
func (c *windowControllerBase) Activate() {
	if c.destroyed {
		return
	}
	if c.hidden {
		c.hidden = false
		c.owner.b.d.SetWindowHidden(c.view.ViewID(), false)
	}
	c.minimized = false
	c.view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate})
	c.refresh()
}

// SetMinimized asks for the window to be minimized, or, with false, shown again.
//
// Flutter: widgets/_window.dart RegularWindowController.setMinimized
func (c *windowControllerBase) SetMinimized(minimized bool) {
	if c.destroyed {
		return
	}
	if !minimized {
		c.Activate()
		return
	}
	c.minimized = true
	c.view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMinimize})
	c.refresh()
}

// snapshot is the state of the window that a WindowScope reports.
func (c *windowControllerBase) snapshot() windowSnapshot {
	info := c.view.Window()
	return windowSnapshot{
		size: c.ContentSize(), title: c.title,
		activated: info.Activated, maximized: info.Maximized && !c.dialog, minimized: c.minimized, fullscreen: info.Fullscreen && !c.dialog,
		hidden: c.hidden,
	}
}

// refresh takes the state the platform reported and tells the listeners when it
// changed.
func (c *windowControllerBase) refresh() {
	if c.destroyed {
		return
	}
	info := c.view.Window()
	if info.Title != "" {
		c.title = info.Title
	}
	switch {
	case info.Minimized:
		c.minimized, c.reportedMinimized = true, true
	case c.reportedMinimized:
		c.minimized, c.reportedMinimized = false, false // the platform says, and it says no
	case info.Activated:
		c.minimized = false
	}
	if s := c.snapshot(); s != c.last {
		c.last = s
		c.NotifyListeners()
	}
}

// destroy ends the window: the windows that depend on it go first, then the
// window itself, and the delegate is told. It does nothing the second time.
func (c *windowControllerBase) destroy() {
	if c.destroyed {
		return
	}
	for _, child := range slices.Clone(c.children) {
		child()
	}
	c.destroyed = true
	c.owner.forget(c.self)
	c.owner.b.d.DestroyWindow(c.view.ViewID())
	if c.onDestroy != nil {
		c.onDestroy()
	}
	c.NotifyListeners()
}

// constraintSizes turns box constraints into the smallest and largest content size
// of a window; zero is no limit.
func constraintSizes(c rendering.BoxConstraints) (min, max ui.Size) {
	min = ui.Size{Width: c.MinWidth, Height: c.MinHeight}
	if c.HasBoundedWidth() {
		max.Width = c.MaxWidth
	}
	if c.HasBoundedHeight() {
		max.Height = c.MaxHeight
	}
	return min, max
}

// RegularWindowControllerDelegate listens to the life of a regular window. Embed
// RegularWindowControllerDelegateBase and override what you need.
//
// Flutter: widgets/_window.dart RegularWindowControllerDelegate
type RegularWindowControllerDelegate interface {
	// OnWindowCloseRequested is called when the user asks to close the window. The
	// default destroys the window; an override can delay or prevent that.
	//
	// Flutter: widgets/_window.dart RegularWindowControllerDelegate.onWindowCloseRequested
	OnWindowCloseRequested(controller *RegularWindowController)
	// OnWindowDestroyed is called after the window is closed.
	//
	// Flutter: widgets/_window.dart RegularWindowControllerDelegate.onWindowDestroyed
	OnWindowDestroyed()
}

// RegularWindowControllerDelegateBase is a RegularWindowControllerDelegate that
// destroys the window on a request to close it and does nothing when it is gone.
//
// Flutter: widgets/_window.dart RegularWindowControllerDelegate
type RegularWindowControllerDelegateBase struct{}

// OnWindowCloseRequested destroys the window.
//
// Flutter: widgets/_window.dart RegularWindowControllerDelegate.onWindowCloseRequested
func (RegularWindowControllerDelegateBase) OnWindowCloseRequested(controller *RegularWindowController) {
	controller.Destroy()
}

// OnWindowDestroyed does nothing.
//
// Flutter: widgets/_window.dart RegularWindowControllerDelegate.onWindowDestroyed
func (RegularWindowControllerDelegateBase) OnWindowDestroyed() {}

// RegularWindowController is the controller of a regular window: a traditional
// top-level window that can be resized, minimized, maximized and closed. The
// window is created when the controller is; the controller does not touch the
// widget tree, a RegularWindow widget shows the content in the window. The user of
// the controller destroys the window with Destroy when it is no longer needed.
// Descendants of the RegularWindow reach the controller with WindowScopeOf.
//
// Deviation: Flutter's is an abstract class with a platform implementation
// behind a factory; this is the one implementation, made by the WindowingOwner.
// The windows that size themselves to their content (sizedToContent) are not
// ported (SPEC §12.10).
//
// Flutter: widgets/_window.dart RegularWindowController
type RegularWindowController struct {
	windowControllerBase
	delegate RegularWindowControllerDelegate
}

var _ BaseWindowController = (*RegularWindowController)(nil)

// IsMaximized is whether the window is maximized.
//
// Flutter: widgets/_window.dart RegularWindowController.isMaximized
func (c *RegularWindowController) IsMaximized() bool { return c.view.Window().Maximized }

// IsFullscreen is whether the window is fullscreen.
//
// Flutter: widgets/_window.dart RegularWindowController.isFullscreen
func (c *RegularWindowController) IsFullscreen() bool { return c.view.Window().Fullscreen }

// SetMaximized asks for the window to be maximized, or restored. It has no
// effect on a window that is fullscreen or minimized.
//
// Flutter: widgets/_window.dart RegularWindowController.setMaximized
func (c *RegularWindowController) SetMaximized(maximized bool) {
	if c.destroyed {
		return
	}
	kind := ui.WindowActionRestore
	if maximized {
		kind = ui.WindowActionMaximize
	}
	c.view.PerformWindowAction(ui.WindowAction{Kind: kind})
}

// SetFullscreen asks for the window to be fullscreen, or to go back to the state
// it had. The platform may refuse either.
//
// Deviation: Flutter's takes the display to go fullscreen on; the compositor
// chooses here (SPEC §12.10).
//
// Flutter: widgets/_window.dart RegularWindowController.setFullscreen
func (c *RegularWindowController) SetFullscreen(fullscreen bool) {
	if c.destroyed {
		return
	}
	kind := ui.WindowActionUnfullscreen
	if fullscreen {
		kind = ui.WindowActionFullscreen
	}
	c.view.PerformWindowAction(ui.WindowAction{Kind: kind})
}

// IsHidden is whether the app took the window off the screen with SetHidden. It is
// what the app asked for: the platform does not hide a window by itself.
//
// Flutter: no counterpart
func (c *RegularWindowController) IsHidden() bool { return c.hidden }

// SetHidden takes the window off the screen (true): it has no surface on the
// screen, no entry in the task bar, and gets no input and no focus. With false the
// window comes back with the size, title, icon and state it had, and asks for the
// focus. The widgets of the window and their state stay as they are: only the
// platform window is hidden. A hidden window still counts as open, so RunApp goes on
// while it exists; when every window of the app is hidden the app lifecycle state
// is AppLifecycleStateHidden, and frames and tickers stop until a window is shown.
// Activate shows a hidden window too. A platform that cannot hide a window (the
// web) ignores it.
//
// Deviation: Flutter's window controllers cannot hide a window; this is what a
// desktop app that lives in the system tray needs (SPEC §8.7, §12.10).
//
// Flutter: no counterpart
func (c *RegularWindowController) SetHidden(hidden bool) {
	if c.destroyed || c.hidden == hidden {
		return
	}
	c.hidden = hidden
	c.owner.b.d.SetWindowHidden(c.view.ViewID(), hidden)
	c.refresh()
}

// Destroy destroys the window and tells the delegate. It does nothing when the
// window is destroyed already.
//
// Flutter: widgets/_window.dart BaseWindowController.destroy
func (c *RegularWindowController) Destroy() { c.destroy() }

// DialogWindowControllerDelegate listens to the life of a dialog window. Embed
// DialogWindowControllerDelegateBase and override what you need.
//
// Flutter: widgets/_window.dart DialogWindowControllerDelegate
type DialogWindowControllerDelegate interface {
	// OnWindowCloseRequested is called when the user asks to close the window. The
	// default destroys the window; an override can delay or prevent that.
	//
	// Flutter: widgets/_window.dart DialogWindowControllerDelegate.onWindowCloseRequested
	OnWindowCloseRequested(controller *DialogWindowController)
	// OnWindowDestroyed is called after the window is closed.
	//
	// Flutter: widgets/_window.dart DialogWindowControllerDelegate.onWindowDestroyed
	OnWindowDestroyed()
}

// DialogWindowControllerDelegateBase is a DialogWindowControllerDelegate that
// destroys the window on a request to close it and does nothing when it is gone.
//
// Flutter: widgets/_window.dart DialogWindowControllerDelegate
type DialogWindowControllerDelegateBase struct{}

// OnWindowCloseRequested destroys the window.
//
// Flutter: widgets/_window.dart DialogWindowControllerDelegate.onWindowCloseRequested
func (DialogWindowControllerDelegateBase) OnWindowCloseRequested(controller *DialogWindowController) {
	controller.Destroy()
}

// OnWindowDestroyed does nothing.
//
// Flutter: widgets/_window.dart DialogWindowControllerDelegate.onWindowDestroyed
func (DialogWindowControllerDelegateBase) OnWindowDestroyed() {}

// DialogWindowController is the controller of a dialog window. A dialog with a
// parent is modal to it: it has no system menu, is not in the window switcher,
// and closes when the parent does. A dialog without a parent is modeless: it can
// be minimized but not maximized.
//
// Flutter: widgets/_window.dart DialogWindowController
type DialogWindowController struct {
	windowControllerBase
	delegate DialogWindowControllerDelegate
}

var _ BaseWindowController = (*DialogWindowController)(nil)

// Parent is the controller of the window the dialog is modal to, or nil for a
// modeless dialog.
//
// Flutter: widgets/_window.dart DialogWindowController.parent
func (c *DialogWindowController) Parent() BaseWindowController { return c.parent }

// Destroy destroys the window and tells the delegate. It does nothing when the
// window is destroyed already.
//
// Flutter: widgets/_window.dart BaseWindowController.destroy
func (c *DialogWindowController) Destroy() { c.destroy() }

// RegularWindowControllerOpts are the optional parameters of
// NewRegularWindowController.
//
// Flutter: widgets/_window.dart RegularWindowController.new
type RegularWindowControllerOpts struct {
	// Constraints are the smallest and largest size of the window. The size must
	// satisfy them. The platform enforces them in its own way and may not honor
	// them. The default is no limit.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Title is the title of the window; some platforms use the name of the app when
	// it is empty.
	Title string
	// Delegate listens to the life of the window; the default destroys the window
	// when the user closes it.
	Delegate RegularWindowControllerDelegate
	// Decorations is how the title bar is made; the default is DecorationsAuto.
	Decorations Decorations
	// Icon is the icon of the window; the default is none, and the desktop takes
	// the icon of the app.
	Icon ui.WindowIcon
	// ContentType is the hint about what the window shows.
	ContentType ui.ContentType
	// State is the state the window is created in: the platform creates it
	// maximized, minimized or fullscreen, so it never shows at its size first. The
	// zero value is a normal window. No position can be asked for (Wayland gives
	// clients none).
	//
	// Flutter: no counterpart
	State WindowState
}

// DialogWindowControllerOpts are the optional parameters of
// NewDialogWindowController.
//
// Flutter: widgets/_window.dart DialogWindowController.new
type DialogWindowControllerOpts struct {
	// Constraints are the smallest and largest size of the window; see
	// RegularWindowControllerOpts.Constraints.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Parent is the controller of the window the dialog is modal to; nil makes a
	// modeless dialog.
	Parent BaseWindowController
	// Title is the title of the window.
	Title string
	// Delegate listens to the life of the window; the default destroys the window
	// when the user closes it.
	Delegate DialogWindowControllerDelegate
	// Decorations is how the title bar is made; the default is DecorationsAuto.
	Decorations Decorations
	// Icon is the icon of the window; the default is none.
	Icon ui.WindowIcon
	// ContentType is the hint about what the window shows.
	ContentType ui.ContentType
}

// NewRegularWindowController creates a regular window of the size, in logical
// pixels, and the controller of it. The window belongs to the windowing owner of
// the binding that the context reaches. It fails with WindowingUnsupported when the
// platform cannot make windows, and with the platform's error when it cannot make
// this one. It panics with the tag WindowSizeViolatesConstraints when the size does
// not satisfy the constraints.
//
// Flutter: widgets/_window.dart RegularWindowController.new
func NewRegularWindowController(bc BuildContext, size ui.Size, opts ...RegularWindowControllerOpts) (*RegularWindowController, error) {
	return WidgetsBindingOf(bc).WindowingOwner().CreateRegularWindowController(size, opts...)
}

// NewDialogWindowController creates a dialog window of the size, in logical
// pixels, and the controller of it. See NewRegularWindowController for the errors.
//
// Flutter: widgets/_window.dart DialogWindowController.new
func NewDialogWindowController(bc BuildContext, size ui.Size, opts ...DialogWindowControllerOpts) (*DialogWindowController, error) {
	return WidgetsBindingOf(bc).WindowingOwner().CreateDialogWindowController(size, opts...)
}

// WindowingOwner creates and manages the window controllers of an app. The
// binding has one.
//
// Flutter: widgets/_window.dart WindowingOwner
type WindowingOwner struct {
	b           *WidgetsBinding
	controllers []BaseWindowController
}

func newWindowingOwner(b *WidgetsBinding) *WindowingOwner { return &WindowingOwner{b: b} }

// WindowingOwner is the owner of the windows of the app.
//
// Flutter: widgets/binding.dart WidgetsBinding.windowingOwner
func (b *WidgetsBinding) WindowingOwner() *WindowingOwner { return b.windowing }

// debugCheckConstraints panics when the size does not satisfy the constraints.
func debugCheckConstraints(size ui.Size, constraints foundation.Opt[rendering.BoxConstraints]) {
	if c, ok := constraints.Get(); ok && foundation.DebugMode && !c.IsSatisfiedBy(size) {
		panic(errors.New(tagWindowSizeViolatesConstraints, "the size of a window must satisfy its constraints",
			"size", size.String(), "constraints", c.String()))
	}
}

// newController makes the part of a controller that every kind has, and creates the
// window.
func (o *WindowingOwner) newController(self BaseWindowController, spec ui.WindowSpec, constraints foundation.Opt[rendering.BoxConstraints], title string) error {
	if c, ok := constraints.Get(); ok {
		spec.MinSize, spec.MaxSize = constraintSizes(c)
	}
	spec.Title = title
	view, err := o.b.d.CreateWindow(spec)
	if err != nil {
		return err
	}
	base := self.windowBase()
	base.owner, base.self, base.view, base.title = o, self, view, title
	base.SetErrorReporter(o.b.report)
	base.SetUIThread(o.b.thread)
	base.last = base.snapshot()
	o.controllers = append(o.controllers, self)
	return nil
}

// CreateRegularWindowController is NewRegularWindowController for the owner.
//
// Flutter: widgets/_window.dart WindowingOwner.createRegularWindowController
func (o *WindowingOwner) CreateRegularWindowController(size ui.Size, opts ...RegularWindowControllerOpts) (*RegularWindowController, error) {
	opt := firstOpts(opts)
	debugCheckConstraints(size, opt.Constraints)
	c := &RegularWindowController{delegate: opt.Delegate}
	if c.delegate == nil {
		c.delegate = RegularWindowControllerDelegateBase{}
	}
	spec := ui.WindowSpec{Kind: ui.WindowKindRegular, Size: size, AppDecorations: opt.Decorations == DecorationsApp,
		Icon: opt.Icon, ContentType: opt.ContentType, State: opt.State.startState()}
	if err := o.newController(c, spec, opt.Constraints, opt.Title); err != nil {
		return nil, err
	}
	c.appDecorations = opt.Decorations == DecorationsApp
	c.onClose = func() { c.delegate.OnWindowCloseRequested(c) }
	c.onDestroy = func() { c.delegate.OnWindowDestroyed() }
	return c, nil
}

// CreateDialogWindowController is NewDialogWindowController for the owner.
//
// Flutter: widgets/_window.dart WindowingOwner.createDialogWindowController
func (o *WindowingOwner) CreateDialogWindowController(size ui.Size, opts ...DialogWindowControllerOpts) (*DialogWindowController, error) {
	opt := firstOpts(opts)
	debugCheckConstraints(size, opt.Constraints)
	c := &DialogWindowController{delegate: opt.Delegate}
	c.dialog = true
	if c.delegate == nil {
		c.delegate = DialogWindowControllerDelegateBase{}
	}
	spec := ui.WindowSpec{Kind: ui.WindowKindDialog, Size: size, AppDecorations: opt.Decorations == DecorationsApp,
		Icon: opt.Icon, ContentType: opt.ContentType}
	var parent *windowControllerBase
	if opt.Parent != nil {
		parent = opt.Parent.windowBase()
		if parent.destroyed {
			return nil, errors.New(tagWindowParentDestroyed, "the parent of a dialog window was destroyed")
		}
		spec.Parent, spec.HasParent = parent.view.ViewID(), true
	}
	if err := o.newController(c, spec, opt.Constraints, opt.Title); err != nil {
		return nil, err
	}
	c.parent = opt.Parent
	c.appDecorations = opt.Decorations == DecorationsApp
	c.onClose = func() { c.delegate.OnWindowCloseRequested(c) }
	c.onDestroy = func() { c.delegate.OnWindowDestroyed() }
	if parent != nil {
		// A dialog closes with its parent.
		parent.children = append(parent.children, c.destroy)
	}
	return c, nil
}

// ImplicitWindowController is the controller of the first window, the one that
// RunApp makes. It is made on the first call. Closing the window by the user does
// not go through its delegate: it is a request to exit the app, as for any app that
// has only that window (AppLifecycleListener.OnExitRequested).
//
// Deviation: Flutter has no controller for the window that the embedder makes,
// so a dialog cannot name it as its parent; here it can (SPEC §12.10).
//
// Flutter: no counterpart
func (o *WindowingOwner) ImplicitWindowController() (*RegularWindowController, bool) {
	view := o.b.d.ImplicitView()
	if view == nil {
		return nil, false
	}
	for _, c := range o.controllers {
		if r, ok := c.(*RegularWindowController); ok && r.implicit {
			return r, true
		}
	}
	c := &RegularWindowController{delegate: RegularWindowControllerDelegateBase{}}
	c.owner, c.self, c.view, c.implicit = o, c, view, true
	c.title = view.Window().Title
	c.SetErrorReporter(o.b.report)
	c.SetUIThread(o.b.thread)
	c.last = c.snapshot()
	c.onDestroy = func() { c.delegate.OnWindowDestroyed() }
	o.controllers = append(o.controllers, c)
	return c, true
}

// forget drops a controller whose window is gone.
func (o *WindowingOwner) forget(c BaseWindowController) {
	o.controllers = slices.DeleteFunc(o.controllers, func(x BaseWindowController) bool { return x == c })
}

// refresh tells the controllers that the platform reported new metrics.
func (o *WindowingOwner) refresh() {
	for _, c := range slices.Clone(o.controllers) {
		c.windowBase().refresh()
	}
}

// handleCloseRequested hands the request to close a window to the delegate of its
// controller. It reports whether there is one: if so, the engine leaves the window
// open and the delegate decides. The window of the implicit controller is not
// handed over; the request to exit the app is.
func (o *WindowingOwner) handleCloseRequested(view ui.ViewID) bool {
	for _, c := range o.controllers {
		base := c.windowBase()
		if base.view.ViewID() != view || base.implicit {
			continue
		}
		if base.destroyed || base.onClose == nil {
			return true
		}
		foundation.Guard(o.b.report, foundation.CallbackPanicked, "while handling a request to close a window", nil, base.onClose)
		return true
	}
	return false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWindowParentDestroyed         = "WindowParentDestroyed"
	tagWindowSizeViolatesConstraints = "WindowSizeViolatesConstraints"
)
