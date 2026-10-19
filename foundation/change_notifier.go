package foundation

import (
	"slices"

	"github.com/rm4n0s/errors"
)

// VoidCallback is a function that takes no arguments and returns nothing.
//
// Flutter: dart:ui VoidCallback
type VoidCallback func()

// ListenerHandle identifies one registration made with Listenable.AddListener.
// Pass it to RemoveListener to undo that registration. The zero value
// identifies nothing, and removing it does nothing.
//
// Deviation: Go functions cannot be compared, so AddListener returns a handle
// where Flutter's removeListener takes the listener itself (SPEC §12.10).
//
// Flutter: no counterpart
type ListenerHandle struct{ id uint64 }

// Listenable is an object that maintains a list of listeners. The listeners
// are typically used to notify clients that the object has been updated.
//
// Flutter: foundation/change_notifier.dart Listenable
type Listenable interface {
	// AddListener registers a function to be called when the object changes
	// and returns the handle that removes it again.
	//
	// Flutter: foundation/change_notifier.dart Listenable.addListener
	AddListener(listener VoidCallback) ListenerHandle
	// RemoveListener removes a registration made with AddListener. Removing a
	// handle that is not registered (or twice) does nothing.
	//
	// Flutter: foundation/change_notifier.dart Listenable.removeListener
	RemoveListener(handle ListenerHandle)
}

// ValueListenable is an interface for subclasses of Listenable that expose a
// current value.
//
// Flutter: foundation/change_notifier.dart ValueListenable
type ValueListenable[T any] interface {
	Listenable
	// Value is the current value of the object. When it changes, the
	// listeners are called.
	//
	// Flutter: foundation/change_notifier.dart ValueListenable.value
	Value() T
}

// listenerEntry is one registered listener. A nil fn is a listener removed
// while a notification was running.
type listenerEntry struct {
	id uint64
	fn VoidCallback
}

// ChangeNotifier is a class that can be extended or embedded to provide a
// change notification API using AddListener and RemoveListener.
//
// Embed it in a struct and call NotifyListeners when the struct changes. The
// zero value is ready to use. A ChangeNotifier must not be copied after first
// use. Only the UI thread may use it (SPEC §13).
//
// Listeners may add and remove listeners, including themselves, while they
// run: a listener added during a notification is not called in that
// notification, and one removed during it is not called afterwards.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier
type ChangeNotifier struct {
	listeners []listenerEntry
	nextID    uint64
	// notifyDepth counts the running NotifyListeners calls (they can nest).
	notifyDepth int
	// removedWhileNotifying counts the entries whose fn was set to nil during
	// a notification.
	removedWhileNotifying int
	disposed              bool
	reporter              ErrorReporter
	uiThread              *UIThread
}

// SetErrorReporter sets the function that receives the panics of listeners. The
// owner of the notifier (the binding, a controller) sets it. Without one, a
// listener that panics does not stop the others, and the first panic goes on
// when the last one has been called.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.notifyListeners
func (n *ChangeNotifier) SetErrorReporter(r ErrorReporter) { n.reporter = r }

// SetUIThread makes debug builds check that NotifyListeners is called from the UI
// thread, with the tag WrongThread (SPEC §13).
//
// Flutter: no counterpart
func (n *ChangeNotifier) SetUIThread(t *UIThread) { n.uiThread = t }

// NewChangeNotifier creates a ChangeNotifier. The zero value works too; this
// constructor is for a notifier used on its own.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.new
func NewChangeNotifier() *ChangeNotifier { return &ChangeNotifier{} }

// DebugAssertNotDisposed panics with the tag ChangeNotifierUsedAfterDispose
// in debug builds when Dispose has been called, and returns true. Other
// builds do nothing.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.debugAssertNotDisposed
func (n *ChangeNotifier) DebugAssertNotDisposed() bool {
	if DebugMode && n.disposed {
		panic(errors.New("ChangeNotifierUsedAfterDispose",
			"A ChangeNotifier was used after being disposed. Once you have called Dispose() on a ChangeNotifier, it can no longer be used."))
	}
	return true
}

// HasListeners reports whether any listeners are currently registered. A
// listener removed during a notification still counts until the notification
// is over.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.hasListeners
func (n *ChangeNotifier) HasListeners() bool { return len(n.listeners) > 0 }

// AddListener registers a function to be called when the object changes. It
// can be called again with the same function: it is then called once per
// registration.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.addListener
func (n *ChangeNotifier) AddListener(listener VoidCallback) ListenerHandle {
	n.DebugAssertNotDisposed()
	n.nextID++
	n.listeners = append(n.listeners, listenerEntry{id: n.nextID, fn: listener})
	return ListenerHandle{id: n.nextID}
}

// RemoveListener removes a registration made with AddListener. It is allowed
// on a disposed object, so that listeners can clean up after their owner.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.removeListener
func (n *ChangeNotifier) RemoveListener(handle ListenerHandle) {
	if handle.id == 0 {
		return
	}
	for i, e := range n.listeners {
		if e.id != handle.id || e.fn == nil {
			continue
		}
		if n.notifyDepth > 0 {
			// The list keeps its length during a notification; the entry
			// goes when all notifications are over.
			n.listeners[i].fn = nil
			n.removedWhileNotifying++
		} else {
			n.listeners = slices.Delete(n.listeners, i, i+1)
			if len(n.listeners) == 0 {
				n.listeners = nil
			}
		}
		return
	}
}

// Dispose discards any resources used by the object. After this is called,
// the object is not in a usable state (AddListener and NotifyListeners panic
// in debug builds). It panics with the tag ChangeNotifierDisposedWhileNotifying
// in debug builds when called from a listener.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.dispose
func (n *ChangeNotifier) Dispose() {
	n.DebugAssertNotDisposed()
	if DebugMode && n.notifyDepth != 0 {
		panic(errors.New("ChangeNotifierDisposedWhileNotifying",
			"Dispose() was called during the call to NotifyListeners(). This is likely to cause errors since it modifies the list of listeners while the list is being used."))
	}
	n.disposed = true
	n.listeners = nil
}

// NotifyListeners calls all the registered listeners, in the order they were
// added.
//
// Listeners added during the call are not called by it, and a listener
// removed during the call is not called after its removal.
//
// A listener that panics does not stop the others. The panic is reported to the
// error reporter, if the notifier has one (SetErrorReporter), as a
// CallbackPanicked error; without a reporter the first panic goes on after the
// last listener was called. In debug builds, a notifier that has a UI thread
// (SetUIThread) panics with the tag WrongThread when it is notified from another
// goroutine.
//
// Flutter: foundation/change_notifier.dart ChangeNotifier.notifyListeners
func (n *ChangeNotifier) NotifyListeners() {
	n.DebugAssertNotDisposed()
	n.uiThread.AssertCurrent("ChangeNotifier.NotifyListeners")
	if len(n.listeners) == 0 {
		return
	}
	n.notifyDepth++
	// The deferred call keeps the bookkeeping right if a listener panics.
	defer n.endNotify()
	var first any
	end := len(n.listeners)
	for i := 0; i < end; i++ {
		fn := n.listeners[i].fn
		if fn == nil {
			continue
		}
		func() {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				if n.reporter == nil {
					if first == nil {
						first = r
					}
					return
				}
				n.reporter(PanicError(CallbackPanicked, "while dispatching notifications for "+DescribeIdentity(n), r))
			}()
			fn()
		}()
	}
	if first != nil {
		panic(first)
	}
}

// endNotify ends one NotifyListeners call, and drops the listeners that were
// removed during the notifications when the last one is over.
func (n *ChangeNotifier) endNotify() {
	n.notifyDepth--
	if n.notifyDepth != 0 || n.removedWhileNotifying == 0 {
		return
	}
	n.listeners = slices.DeleteFunc(n.listeners, func(e listenerEntry) bool { return e.fn == nil })
	if len(n.listeners) == 0 {
		n.listeners = nil
	}
	n.removedWhileNotifying = 0
}

// mergedListenable is the Listenable that MergeListenables returns.
type mergedListenable struct {
	children []Listenable
	nextID   uint64
	handles  map[uint64][]childHandle
}

type childHandle struct {
	listenable Listenable
	handle     ListenerHandle
}

// MergeListenables creates a Listenable that triggers when any of the given
// Listenables themselves trigger. A nil entry is skipped.
//
// Flutter: foundation/change_notifier.dart Listenable.merge
func MergeListenables(listenables ...Listenable) Listenable {
	return &mergedListenable{children: slices.Clone(listenables), handles: map[uint64][]childHandle{}}
}

// AddListener registers the listener with every child.
func (m *mergedListenable) AddListener(listener VoidCallback) ListenerHandle {
	m.nextID++
	hs := make([]childHandle, 0, len(m.children))
	for _, c := range m.children {
		if isNil(c) {
			continue
		}
		hs = append(hs, childHandle{c, c.AddListener(listener)})
	}
	m.handles[m.nextID] = hs
	return ListenerHandle{id: m.nextID}
}

// RemoveListener removes the registration from every child.
func (m *mergedListenable) RemoveListener(handle ListenerHandle) {
	for _, h := range m.handles[handle.id] {
		h.listenable.RemoveListener(h.handle)
	}
	delete(m.handles, handle.id)
}

// String describes the merged object, such as "Listenable.merge([a, b])".
func (m *mergedListenable) String() string {
	parts := make([]string, len(m.children))
	for i, c := range m.children {
		parts[i] = describeValue(c)
	}
	return "Listenable.merge([" + joinComma(parts) + "])"
}

// ValueNotifier is a ChangeNotifier that holds a single value. When the value
// is replaced with something that is not equal to the old value, it notifies
// its listeners. Equality is Go's == for comparable values and identity for
// slices, maps and functions, as Dart's == is for lists and functions.
//
// Flutter: foundation/change_notifier.dart ValueNotifier
type ValueNotifier[T any] struct {
	ChangeNotifier
	value T
}

// NewValueNotifier creates a ValueNotifier that starts with value.
//
// Flutter: foundation/change_notifier.dart ValueNotifier.new
func NewValueNotifier[T any](value T) *ValueNotifier[T] { return &ValueNotifier[T]{value: value} }

// Value is the current value stored in the notifier.
//
// Flutter: foundation/change_notifier.dart ValueNotifier.value
func (v *ValueNotifier[T]) Value() T { return v.value }

// SetValue replaces the value. When it is not equal to the old one, the
// listeners are notified.
//
// Flutter: foundation/change_notifier.dart ValueNotifier.value=
func (v *ValueNotifier[T]) SetValue(newValue T) {
	if equalValues(v.value, newValue) {
		return
	}
	v.value = newValue
	v.NotifyListeners()
}

// String describes the notifier, such as "ValueNotifier<int>#1a2b3(5)".
//
// Flutter: foundation/change_notifier.dart ValueNotifier.toString
func (v *ValueNotifier[T]) String() string {
	return DescribeIdentity(v) + "(" + describeValue(v.value) + ")"
}
