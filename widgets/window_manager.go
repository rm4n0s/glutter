package widgets

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// WindowEntry is a window that a WindowRegistry holds: its controller, and the
// function that builds its content.
//
// Flutter: widgets/_window.dart WindowEntry
type WindowEntry struct {
	// Controller is the controller that manages the window.
	//
	// Flutter: widgets/_window.dart WindowEntry.controller
	Controller BaseWindowController
	// Builder builds the content of the window.
	//
	// Flutter: widgets/_window.dart WindowEntry.builder
	Builder WidgetBuilder
}

// NewWindowEntry creates an entry for the controller whose content the builder
// makes.
//
// Flutter: widgets/_window.dart WindowEntry.new
func NewWindowEntry(controller BaseWindowController, builder WidgetBuilder) *WindowEntry {
	return &WindowEntry{Controller: controller, Builder: builder}
}

// WindowRegistry holds the windows that a WindowManager shows. It lets a widget
// deep in the tree make top-level windows that are logically below it: Register
// adds one, Unregister removes it. A window must be unregistered before it is
// destroyed.
//
// Flutter: widgets/_window.dart WindowRegistry
type WindowRegistry struct {
	foundation.ChangeNotifier
	windows []*WindowEntry
}

// NewWindowRegistry creates an empty registry.
//
// Flutter: widgets/_window.dart WindowRegistry.new
func NewWindowRegistry() *WindowRegistry { return &WindowRegistry{} }

// Windows are the windows that are registered, in the order they were.
//
// Flutter: widgets/_window.dart WindowRegistry.windows
func (r *WindowRegistry) Windows() []*WindowEntry { return slices.Clone(r.windows) }

// Register adds the window to the registry and tells the listeners.
//
// Flutter: widgets/_window.dart WindowRegistry.register
func (r *WindowRegistry) Register(entry *WindowEntry) {
	r.windows = append(r.windows, entry)
	r.NotifyListeners()
}

// Unregister removes the window from the registry and tells the listeners.
//
// Flutter: widgets/_window.dart WindowRegistry.unregister
func (r *WindowRegistry) Unregister(entry *WindowEntry) {
	r.windows = slices.DeleteFunc(r.windows, func(e *WindowEntry) bool { return e == entry })
	r.NotifyListeners()
}

// windowRegistryScope tells the descendants of a WindowManager which registry
// they have.
//
// Flutter: widgets/_window.dart _WindowRegistryScope
type windowRegistryScope struct {
	InheritedWidget
	registry *WindowRegistry
}

func (w *windowRegistryScope) UpdateShouldNotify(old Widget) bool {
	return w.registry != old.(*windowRegistryScope).registry
}

// WindowRegistryMaybeOf is the registry of the closest WindowManager above the
// context, or nil. The context depends on it.
//
// Flutter: widgets/_window.dart WindowRegistry.maybeOf
func WindowRegistryMaybeOf(bc BuildContext) *WindowRegistry {
	if s, ok := DependOn[*windowRegistryScope](bc); ok {
		return s.registry
	}
	return nil
}

// WindowRegistryOf is the registry of the closest WindowManager above the context.
// It panics with the tag WindowRegistryMissing when there is none.
//
// Flutter: widgets/_window.dart WindowRegistry.of
func WindowRegistryOf(bc BuildContext) *WindowRegistry {
	r := WindowRegistryMaybeOf(bc)
	if r == nil {
		panic(errors.New("WindowRegistryMissing",
			"no WindowRegistry found in the context; WindowRegistryOf needs a WindowManager widget above the context",
			"context", bc.(Element).String()))
	}
	return r
}

// WindowManagerOpts are the optional parameters of NewWindowManager.
//
// Flutter: widgets/_window.dart WindowManager.new
type WindowManagerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// WindowManager shows windows at the root of an app: its descendants register
// windows in the WindowRegistry it provides, and the manager shows each one in a
// window widget as they come and go. Place it above the app, so that dialogs are at
// the same level as the RegularWindow.
//
// Flutter: widgets/_window.dart WindowManager
type WindowManager struct {
	StatefulWidget
	child Widget
}

// NewWindowManager creates a window manager around the child.
//
// Flutter: widgets/_window.dart WindowManager.new
func NewWindowManager(child Widget, opts ...WindowManagerOpts) *WindowManager {
	o := firstOpts(opts)
	return &WindowManager{StatefulWidget: NewStatefulWidgetBase(WidgetOpts(o)), child: child}
}

// Child is the content inside of the manager.
//
// Flutter: widgets/_window.dart WindowManager.child
func (w *WindowManager) Child() Widget { return w.child }

// CreateState creates the state of the manager.
//
// Flutter: widgets/_window.dart WindowManager.createState
func (w *WindowManager) CreateState() State {
	return &windowManagerState{registry: NewWindowRegistry()}
}

type windowManagerState struct {
	StateOf[*WindowManager]
	registry *WindowRegistry
}

func (s *windowManagerState) Dispose() { s.registry.Dispose() }

// Build provides the registry and shows the registered windows next to the child.
//
// Flutter: widgets/_window.dart _WindowManagerState.build
func (s *windowManagerState) Build(BuildContext) Widget {
	return &windowRegistryScope{
		InheritedWidget: NewInheritedWidgetBase(NewListenableBuilder(s.registry, func(bc BuildContext, child Widget) Widget {
			var subViews []Widget
			for _, entry := range s.registry.Windows() {
				key := foundation.NewObjectKey(entry.Controller)
				content := entry.Builder(bc)
				switch c := entry.Controller.(type) {
				case *DialogWindowController:
					subViews = append(subViews, NewDialogWindow(c, content, DialogWindowOpts{Key: key}))
				case *RegularWindowController:
					subViews = append(subViews, NewRegularWindow(c, content, RegularWindowOpts{Key: key}))
				}
			}
			if MaybeViewOf(bc) == nil {
				// Deviation: Flutter leaves the child out here, which would leave
				// the window of the app without content; the child is a View too.
				return NewViewCollection(append([]Widget{child}, subViews...))
			}
			var anchored Widget
			if len(subViews) > 0 {
				anchored = NewViewCollection(subViews)
			}
			return NewViewAnchor(child, ViewAnchorOpts{View: anchored})
		}, s.Widget().child)),
		registry: s.registry,
	}
}
