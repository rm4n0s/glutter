package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// ScrollNotificationCallback is called with the scroll notifications of the
// scrollables below a ScrollNotificationObserver.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationCallback
type ScrollNotificationCallback func(notification ScrollNotification)

// scrollNotificationObserverScope is the inherited widget that gives the
// descendants of a ScrollNotificationObserver its state.
//
// Flutter: widgets/scroll_notification_observer.dart _ScrollNotificationObserverScope
type scrollNotificationObserverScope struct {
	InheritedWidget
	state *ScrollNotificationObserverState
}

func (w *scrollNotificationObserverScope) UpdateShouldNotify(old Widget) bool {
	return w.state != old.(*scrollNotificationObserverScope).state
}

// ScrollNotificationObserver notifies its listeners of the scroll notifications
// of the scrollables below it, without the listeners having to be above the
// scrollables in the tree.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserver
type ScrollNotificationObserver struct {
	StatefulWidget
	child Widget
}

// NewScrollNotificationObserver creates the widget.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserver.new
func NewScrollNotificationObserver(child Widget, opts ...WidgetOpts) *ScrollNotificationObserver {
	return &ScrollNotificationObserver{NewStatefulWidgetBase(opts...), child}
}

// Child is the widget below this one.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserver.child
func (w *ScrollNotificationObserver) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserver.createState
func (w *ScrollNotificationObserver) CreateState() State { return &ScrollNotificationObserverState{} }

// ScrollNotificationObserverMaybeOf is the state of the closest observer above the
// context, or nil. The context depends on it.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserver.maybeOf
func ScrollNotificationObserverMaybeOf(bc BuildContext) *ScrollNotificationObserverState {
	if w, ok := DependOn[*scrollNotificationObserverScope](bc); ok {
		return w.state
	}
	return nil
}

// ScrollNotificationObserverOf is the state of the closest observer above the
// context. In debug builds it panics with the tag ScrollNotificationObserverMissing
// when there is none.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserver.of
func ScrollNotificationObserverOf(bc BuildContext) *ScrollNotificationObserverState {
	s := ScrollNotificationObserverMaybeOf(bc)
	if s == nil && foundation.DebugMode {
		panic(errors.New("ScrollNotificationObserverMissing",
			"ScrollNotificationObserverOf() was called with a context that does not contain a ScrollNotificationObserver widget; no ScrollNotificationObserver widget ancestor could be found starting from the context that was passed",
			"context", bc.(Element).String()))
	}
	return s
}

// ScrollNotificationListenerHandle identifies a listener added to a
// ScrollNotificationObserverState.
//
// Flutter: no counterpart (Dart removes a listener by its function)
type ScrollNotificationListenerHandle struct{ id uint64 }

type observerEntry struct {
	handle   ScrollNotificationListenerHandle
	listener ScrollNotificationCallback
}

// ScrollNotificationObserverState is the state of a ScrollNotificationObserver:
// it keeps the listeners.
//
// Deviation: AddListener returns a handle that RemoveListener takes, where Dart
// removes the listener by its function (SPEC §12.10).
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserverState
type ScrollNotificationObserverState struct {
	StateOf[*ScrollNotificationObserver]
	listeners []*observerEntry
	nextID    uint64
	disposed  bool
}

func (s *ScrollNotificationObserverState) assertNotDisposed() {
	if foundation.DebugMode && s.disposed {
		panic(errors.New("ScrollNotificationObserverDisposed",
			"a ScrollNotificationObserverState was used after being disposed; once you have called Dispose() on it, it can no longer be used"))
	}
}

// AddListener registers the listener; the handle removes it.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserverState.addListener
func (s *ScrollNotificationObserverState) AddListener(listener ScrollNotificationCallback) ScrollNotificationListenerHandle {
	s.assertNotDisposed()
	s.nextID++
	e := &observerEntry{listener: listener}
	e.handle = ScrollNotificationListenerHandle{s.nextID}
	s.listeners = append(s.listeners, e)
	return e.handle
}

// RemoveListener removes the listener that the handle was made for.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserverState.removeListener
func (s *ScrollNotificationObserverState) RemoveListener(handle ScrollNotificationListenerHandle) {
	s.assertNotDisposed()
	for i, e := range s.listeners {
		if e.handle == handle {
			s.listeners = append(s.listeners[:i:i], s.listeners[i+1:]...)
			return
		}
	}
}

func (s *ScrollNotificationObserverState) notifyListeners(notification ScrollNotification) {
	s.assertNotDisposed()
	if len(s.listeners) == 0 {
		return
	}
	local := append([]*observerEntry(nil), s.listeners...)
	for _, e := range local {
		// A listener that was removed by an earlier one is not called.
		still := false
		for _, cur := range s.listeners {
			if cur == e {
				still = true
				break
			}
		}
		if !still {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					WidgetsBindingOf(s.Context()).ErrorReporter()(errors.NewErrf("ScrollNotificationObserverListenerFailed",
						"%v while dispatching notifications for ScrollNotificationObserverState", r))
				}
			}()
			e.listener(notification)
		}()
	}
}

// Build listens to the scroll notifications below and gives the state to them.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserverState.build
func (s *ScrollNotificationObserverState) Build(BuildContext) Widget {
	return NewNotificationListener[*ScrollMetricsNotification](
		NewNotificationListener[ScrollNotification](
			&scrollNotificationObserverScope{NewInheritedWidgetBase(s.Widget().child), s},
			func(notification ScrollNotification) bool {
				s.notifyListeners(notification)
				return false
			}),
		func(notification *ScrollMetricsNotification) bool {
			// A ScrollMetricsNotification allows listeners to be notified for an
			// initial state, as well as if the content dimensions change without
			// scrolling.
			s.notifyListeners(notification.AsScrollUpdate())
			return false
		})
}

// Dispose drops the listeners.
//
// Flutter: widgets/scroll_notification_observer.dart ScrollNotificationObserverState.dispose
func (s *ScrollNotificationObserverState) Dispose() {
	s.assertNotDisposed()
	s.listeners = nil
	s.disposed = true
}
