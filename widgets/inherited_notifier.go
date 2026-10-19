package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
)

// InheritedNotifier is an inherited widget whose dependents are rebuilt when a
// Listenable (such as a ChangeNotifier) notifies, not only when the widget is
// replaced. Embed it and use DependOn to look the outer widget up.
//
// Flutter: widgets/inherited_notifier.dart InheritedNotifier
type InheritedNotifier struct {
	InheritedWidget
	notifier foundation.Listenable
}

// NewInheritedNotifierBase creates the embedded base. The notifier may be nil.
//
// Flutter: widgets/inherited_notifier.dart InheritedNotifier.new
func NewInheritedNotifierBase(notifier foundation.Listenable, child Widget, opts ...WidgetOpts) InheritedNotifier {
	return InheritedNotifier{NewInheritedWidgetBase(child, opts...), notifier}
}

// Notifier is the object the dependents listen to, or nil.
//
// Flutter: widgets/inherited_notifier.dart InheritedNotifier.notifier
func (n *InheritedNotifier) Notifier() foundation.Listenable { return n.notifier }

// UpdateShouldNotify is whether the notifier is another one.
//
// Flutter: widgets/inherited_notifier.dart InheritedNotifier.updateShouldNotify
func (n *InheritedNotifier) UpdateShouldNotify(oldWidget Widget) bool {
	return oldWidget.(notifierWidget).Notifier() != n.notifier
}

func (*InheritedNotifier) isInheritedNotifier() {}

// notifierWidget is any widget with the InheritedNotifier base.
type notifierWidget interface {
	InheritedWidgetInterface
	Notifier() foundation.Listenable
	isInheritedNotifier()
}

// InheritedNotifierElement is the element of an InheritedNotifier: it listens to
// the notifier and rebuilds its dependents when it notifies.
//
// Flutter: widgets/inherited_notifier.dart _InheritedNotifierElement
type InheritedNotifierElement struct {
	InheritedElement
	handle foundation.ListenerHandle
	listen bool
	dirty  bool
}

// NewInheritedNotifierElement creates the element of an inherited notifier.
//
// Flutter: widgets/inherited_notifier.dart _InheritedNotifierElement.new
func NewInheritedNotifierElement(w notifierWidget) *InheritedNotifierElement {
	e := &InheritedNotifierElement{}
	InitInheritedElement(&e.InheritedElement, e, w)
	e.addListener(w.Notifier())
	return e
}

func (e *InheritedNotifierElement) addListener(n foundation.Listenable) {
	if n != nil {
		e.handle, e.listen = n.AddListener(e.handleUpdate), true
	}
}

func (e *InheritedNotifierElement) removeListener(n foundation.Listenable) {
	if n != nil && e.listen {
		n.RemoveListener(e.handle)
		e.listen = false
	}
}

// Update moves the listener to the new notifier.
//
// Flutter: widgets/inherited_notifier.dart _InheritedNotifierElement.update
func (e *InheritedNotifierElement) Update(newWidget Widget) {
	oldNotifier := e.widget.(notifierWidget).Notifier()
	newNotifier := newWidget.(notifierWidget).Notifier()
	if oldNotifier != newNotifier {
		e.removeListener(oldNotifier)
		e.addListener(newNotifier)
	}
	e.InheritedElement.Update(newWidget)
}

// Build tells the dependents first if the notifier notified.
//
// Flutter: widgets/inherited_notifier.dart _InheritedNotifierElement.build
func (e *InheritedNotifierElement) Build() Widget {
	if e.dirty {
		e.NotifyClients(e.widget)
	}
	return e.InheritedElement.Build()
}

func (e *InheritedNotifierElement) handleUpdate() {
	e.dirty = true
	e.MarkNeedsBuild()
}

// NotifyClients tells the dependents and clears the flag.
//
// Flutter: widgets/inherited_notifier.dart _InheritedNotifierElement.notifyClients
func (e *InheritedNotifierElement) NotifyClients(oldWidget Widget) {
	e.InheritedElement.NotifyClients(oldWidget)
	e.dirty = false
}

// Unmount stops listening.
//
// Flutter: widgets/inherited_notifier.dart _InheritedNotifierElement.unmount
func (e *InheritedNotifierElement) Unmount() {
	e.removeListener(e.widget.(notifierWidget).Notifier())
	e.InheritedElement.Unmount()
}
