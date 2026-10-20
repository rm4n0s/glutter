package widgets

// Notification is a message that bubbles up the tree from the widget that
// dispatches it (see DispatchNotification) to the NotificationListener widgets
// above it. It can be a value of any type.
//
// Flutter: widgets/notification_listener.dart Notification
type Notification any

// NotifiableElement is an element that listens to the notifications dispatched
// below it. NotificationListener makes one; implement it on a custom element to
// receive notifications too.
//
// Flutter: widgets/framework.dart NotifiableElementMixin
type NotifiableElement interface {
	Element
	// OnNotification is called when a notification reaches the element, and
	// returns whether it is handled, which stops it from bubbling further.
	//
	// Flutter: widgets/framework.dart NotifiableElementMixin.onNotification
	OnNotification(notification Notification) bool
}

// DispatchNotification starts bubbling the notification at the context: the
// closest NotifiableElement ancestor that handles it stops it.
//
// Flutter: widgets/notification_listener.dart Notification.dispatch
func DispatchNotification(target BuildContext, notification Notification) {
	if target == nil {
		return
	}
	target.VisitAncestorElements(func(e Element) bool {
		if n, ok := e.(NotifiableElement); ok && n.OnNotification(notification) {
			return false
		}
		return true
	})
}

// NotificationListenerCallback is called when a notification of the listener's
// type reaches it; it returns whether the notification is handled, which stops
// it from bubbling.
//
// Flutter: widgets/notification_listener.dart NotificationListenerCallback
type NotificationListenerCallback[T any] func(notification T) bool

// NotificationListener is a widget that listens for notifications of type T
// bubbling up the tree.
//
// Flutter: widgets/notification_listener.dart NotificationListener
type NotificationListener[T any] struct {
	ProxyWidget
	onNotification NotificationListenerCallback[T]
}

// NewNotificationListener creates a widget that calls onNotification, if it is
// not nil, for the notifications of type T that reach it.
//
// Flutter: widgets/notification_listener.dart NotificationListener.new
func NewNotificationListener[T any](onNotification NotificationListenerCallback[T], child Widget, opts ...WidgetOpts) *NotificationListener[T] {
	return &NotificationListener[T]{NewProxyWidgetBase(child, opts...), onNotification}
}

// OnNotification is called when a notification of type T bubbles up.
//
// Flutter: widgets/notification_listener.dart NotificationListener.onNotification
func (w *NotificationListener[T]) OnNotification() NotificationListenerCallback[T] {
	return w.onNotification
}

// CreateElement creates the element.
//
// Flutter: widgets/notification_listener.dart NotificationListener.createElement
func (w *NotificationListener[T]) CreateElement() Element {
	e := &notificationElement[T]{}
	InitProxyElement(&e.ProxyElement, e, w)
	return e
}

type notificationElement[T any] struct{ ProxyElement }

func (e *notificationElement[T]) OnNotification(notification Notification) bool {
	listener := e.widget.(*NotificationListener[T])
	if n, ok := notification.(T); ok && listener.onNotification != nil {
		return listener.onNotification(n)
	}
	return false
}

// NotifyClients does nothing: the notification tree has no clients to notify.
//
// Flutter: widgets/notification_listener.dart _NotificationElement.notifyClients
func (e *notificationElement[T]) NotifyClients(Widget) {}

// LayoutChangedNotification indicates that the layout of a subtree changed.
//
// Flutter: widgets/notification_listener.dart LayoutChangedNotification
type LayoutChangedNotification struct{}
