package desktop

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	notificationsDest = "org.freedesktop.Notifications"
	notificationsPath = godbus.ObjectPath("/org/freedesktop/Notifications")
	notificationsIfc  = "org.freedesktop.Notifications"

	portalNotificationIfc = "org.freedesktop.portal.Notification"
)

// NotificationID names a notification that is shown: ShowNotification gives it, and
// CloseNotification and the replacement of a notification take it.
//
// Flutter: no counterpart
type NotificationID uint32

// NotificationUrgency is how important a notification is. The zero value is
// NotificationUrgencyUnspecified, which is Normal.
//
// Flutter: no counterpart
type NotificationUrgency int

// The values of NotificationUrgency.
const (
	// NotificationUrgencyUnspecified is Normal.
	//
	// Flutter: no counterpart
	NotificationUrgencyUnspecified NotificationUrgency = iota
	// NotificationUrgencyLow is for notifications that can wait, such as a download that
	// started.
	//
	// Flutter: no counterpart
	NotificationUrgencyLow
	// NotificationUrgencyNormal is the usual one.
	//
	// Flutter: no counterpart
	NotificationUrgencyNormal
	// NotificationUrgencyCritical is for what needs the user at once; desktops keep it
	// on screen until it is dealt with.
	//
	// Flutter: no counterpart
	NotificationUrgencyCritical
)

// NotificationCloseReason is why a notification went away.
//
// Flutter: no counterpart
type NotificationCloseReason int

// The values of NotificationCloseReason are those of the Desktop Notifications
// specification.
const (
	// NotificationClosedUnknown is a reason that the app does not know.
	//
	// Flutter: no counterpart
	NotificationClosedUnknown NotificationCloseReason = iota
	// NotificationClosedExpired is a notification whose time was up.
	//
	// Flutter: no counterpart
	NotificationClosedExpired
	// NotificationClosedDismissed is a notification that the user dismissed.
	//
	// Flutter: no counterpart
	NotificationClosedDismissed
	// NotificationClosedByApp is a notification that CloseNotification closed.
	//
	// Flutter: no counterpart
	NotificationClosedByApp
)

// NotificationAction is a button of a notification.
//
// Flutter: no counterpart
type NotificationAction struct {
	// ID is what OnAction gets when the user chooses the action. The ID "default" is the
	// action of a click on the notification itself, which a desktop may show no button
	// for.
	ID string
	// Label is the text of the button.
	Label string
}

// NotificationOpts are the optional parameters of ShowNotification.
//
// Flutter: no counterpart
type NotificationOpts struct {
	// Body is the text under the summary. It is plain text: where the desktop reads
	// markup in a body, the characters &, < and > are escaped, unless BodyMarkup is set.
	Body string
	// BodyMarkup says that Body has the markup of the specification (<b>, <i>, <u>,
	// <a href>), which is passed as it is.
	BodyMarkup bool
	// Icon is the name of an icon of the theme, or the path of an image file.
	Icon string
	// Urgency is the importance of the notification.
	Urgency NotificationUrgency
	// Category is a kind of notification from the specification, such as "email.arrived"
	// or "transfer.complete", which desktops use to choose the sound and the place.
	Category string
	// ReplacesID replaces a notification that is shown (the ID that ShowNotification gave)
	// instead of adding another one; the zero value adds one.
	ReplacesID NotificationID
	// Timeout is how long the notification stays; zero is the desktop's default.
	Timeout time.Duration
	// Persistent keeps the notification until the user deals with it, whatever Timeout
	// says.
	Persistent bool
	// AppName is the name of the app that the notification shows; the default is the
	// application ID of the app, or "Glutter".
	AppName string
	// Actions are the buttons of the notification.
	Actions []NotificationAction
	// OnAction is called, on the UI thread, with the ID of the action that the user chose.
	OnAction func(actionID string)
	// OnClosed is called, on the UI thread, when the notification goes away. Inside a
	// Flatpak sandbox the portal does not tell, and it is not called.
	OnClosed func(reason NotificationCloseReason)
}

// notifyHandlers are the callbacks of one notification.
type notifyHandlers struct {
	onAction func(string)
	onClosed func(NotificationCloseReason)
}

// notifyState is the state of the notifications of an app: the signals of the
// desktop are followed from the first notification on, and each is delivered to the
// callbacks of its notification.
type notifyState struct {
	mu       sync.Mutex
	started  bool
	cancel   context.CancelFunc
	handlers map[uint32]notifyHandlers
	// caps are the capabilities that the server said it has, once it said.
	caps      []string
	capsKnown bool
	// portalIDs are the names the portal has for the notifications of a sandboxed app
	// (the portal takes names, the API numbers), and next the number of the next one.
	portalIDs map[uint32]string
	next      uint32
}

// ShowNotification shows a notification with the summary (its title) through the
// desktop's notification service (org.freedesktop.Notifications), or, inside a
// Flatpak sandbox (/.flatpak-info), through the Notification portal. It returns the
// ID of the notification, for CloseNotification and for replacing it. The actions
// that the user chooses and the end of the notification come back to
// NotificationOpts.OnAction and OnClosed on the UI thread. Without the service it
// fails with NotificationsUnavailable.
//
// It must be called from a goroutine and not on the UI thread (it fails with
// DesktopCallOnUIThread there); ctx ends the wait for the service.
//
// Flutter: no counterpart (Flutter apps use the flutter_local_notifications plugin)
func ShowNotification(ctx context.Context, bc widgets.BuildContext, summary string, opts ...NotificationOpts) (NotificationID, error) {
	o := firstOpts(opts)
	if summary == "" {
		return 0, errors.New("NotificationSummaryEmpty", "a notification needs a summary")
	}
	for _, a := range o.Actions {
		if a.ID == "" || a.Label == "" {
			return 0, errors.New("NotificationActionInvalid", "an action of a notification needs an ID and a label", "id", a.ID, "label", a.Label)
		}
	}
	c, err := begin(ctx, bc)
	if err != nil {
		return 0, err
	}
	if c.s.flatpak() {
		return c.showWithPortal(ctx, summary, o)
	}
	return c.show(ctx, summary, o)
}

// CloseNotification closes a notification that is shown: its OnClosed gets
// NotificationClosedByApp. Closing one that is gone is not an error. The rules of the
// call are those of ShowNotification.
//
// Flutter: no counterpart
func CloseNotification(ctx context.Context, bc widgets.BuildContext, id NotificationID) error {
	c, err := begin(ctx, bc)
	if err != nil {
		return err
	}
	bus, err := c.s.connect(ctx)
	if err != nil {
		return errors.NewErr("NotificationsUnavailable", err)
	}
	if c.s.flatpak() {
		name, ok := c.s.notify.portalName(uint32(id))
		if !ok {
			return nil
		}
		_, err = bus.Call(ctx, dbus.PortalDest, dbus.PortalPath, portalNotificationIfc+".RemoveNotification", name)
		c.s.notify.forget(uint32(id))
	} else {
		_, err = bus.Call(ctx, notificationsDest, notificationsPath, notificationsIfc+".CloseNotification", uint32(id))
	}
	if err != nil {
		if portalMissing(err) {
			return errors.NewErr("NotificationsUnavailable", err)
		}
		return err
	}
	return nil
}

func (c *call) appName(o NotificationOpts) string {
	switch {
	case o.AppName != "":
		return o.AppName
	case c.appID != "":
		return c.appID
	}
	return "Glutter"
}

// show sends the notification to org.freedesktop.Notifications.
func (c *call) show(ctx context.Context, summary string, o NotificationOpts) (NotificationID, error) {
	bus, err := c.s.connect(ctx)
	if err != nil {
		return 0, errors.NewErr("NotificationsUnavailable", err)
	}
	if err := c.s.followNotifications(bus); err != nil {
		return 0, err
	}
	body := o.Body
	if !o.BodyMarkup && c.s.serverHas(ctx, bus, "body-markup") {
		body = escapeMarkup(body)
	}
	actions := make([]string, 0, 2*len(o.Actions))
	for _, a := range o.Actions {
		actions = append(actions, a.ID, a.Label)
	}
	hints := map[string]godbus.Variant{"urgency": godbus.MakeVariant(urgencyByte(o.Urgency))}
	if c.appID != "" {
		hints["desktop-entry"] = godbus.MakeVariant(c.appID)
	}
	if o.Category != "" {
		hints["category"] = godbus.MakeVariant(o.Category)
	}
	timeout := int32(-1) // the server's default
	switch {
	case o.Persistent:
		timeout = 0
	case o.Timeout > 0:
		timeout = int32(o.Timeout.Milliseconds())
	}
	reply, err := bus.Call(ctx, notificationsDest, notificationsPath, notificationsIfc+".Notify",
		c.appName(o), uint32(o.ReplacesID), o.Icon, summary, body, actions, hints, timeout)
	if err != nil {
		if portalMissing(err) {
			return 0, errors.NewErr("NotificationsUnavailable", err)
		}
		return 0, err
	}
	if len(reply) != 1 {
		return 0, errors.New("NotificationReplyInvalid", "the reply of Notify does not have one value", "values", len(reply))
	}
	id, ok := reply[0].(uint32)
	if !ok {
		return 0, errors.New("NotificationReplyInvalid", "the reply of Notify is not a uint32", "reply", reply[0])
	}
	c.s.notify.set(id, notifyHandlers{o.OnAction, o.OnClosed})
	return NotificationID(id), nil
}

// showWithPortal sends the notification to the Notification portal of a sandbox.
func (c *call) showWithPortal(ctx context.Context, summary string, o NotificationOpts) (NotificationID, error) {
	bus, err := c.s.connect(ctx)
	if err != nil {
		return 0, errors.NewErr("NotificationsUnavailable", err)
	}
	if err := c.s.followNotifications(bus); err != nil {
		return 0, err
	}
	id, name := c.s.notify.portalID(uint32(o.ReplacesID))
	n := map[string]godbus.Variant{"title": godbus.MakeVariant(summary), "priority": godbus.MakeVariant(portalPriority(o.Urgency))}
	if o.Body != "" {
		n["body"] = godbus.MakeVariant(o.Body)
	}
	if o.Icon != "" {
		n["icon"] = godbus.MakeVariant(struct {
			Kind string
			Data godbus.Variant
		}{"themed", godbus.MakeVariant([]string{o.Icon})})
	}
	var buttons []map[string]godbus.Variant
	for _, a := range o.Actions {
		if a.ID == "default" {
			n["default-action"] = godbus.MakeVariant("default")
			continue
		}
		buttons = append(buttons, map[string]godbus.Variant{"label": godbus.MakeVariant(a.Label), "action": godbus.MakeVariant(a.ID)})
	}
	if len(buttons) > 0 {
		n["buttons"] = godbus.MakeVariant(buttons)
	}
	if _, err := bus.Call(ctx, dbus.PortalDest, dbus.PortalPath, portalNotificationIfc+".AddNotification", name, n); err != nil {
		if portalMissing(err) {
			return 0, errors.NewErr("NotificationsUnavailable", err)
		}
		return 0, err
	}
	c.s.notify.set(id, notifyHandlers{o.OnAction, o.OnClosed})
	return NotificationID(id), nil
}

func urgencyByte(u NotificationUrgency) byte {
	switch u {
	case NotificationUrgencyLow:
		return 0
	case NotificationUrgencyCritical:
		return 2
	}
	return 1
}

func portalPriority(u NotificationUrgency) string {
	switch u {
	case NotificationUrgencyLow:
		return "low"
	case NotificationUrgencyCritical:
		return "urgent"
	}
	return "normal"
}

var markupEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escapeMarkup(s string) string { return markupEscaper.Replace(s) }

// serverHas asks the server for its capabilities, once, and says whether it has one.
// A server that does not answer is taken to have none.
func (s *services) serverHas(ctx context.Context, bus dbus.Bus, capability string) bool {
	s.notify.mu.Lock()
	known, caps := s.notify.capsKnown, s.notify.caps
	s.notify.mu.Unlock()
	if !known {
		reply, err := bus.Call(ctx, notificationsDest, notificationsPath, notificationsIfc+".GetCapabilities")
		if err == nil && len(reply) == 1 {
			caps, _ = reply[0].([]string)
		}
		s.notify.mu.Lock()
		s.notify.caps, s.notify.capsKnown = caps, err == nil
		s.notify.mu.Unlock()
	}
	return slices.Contains(caps, capability)
}

func (n *notifyState) set(id uint32, h notifyHandlers) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.handlers == nil {
		n.handlers = map[uint32]notifyHandlers{}
	}
	n.handlers[id] = h
}

func (n *notifyState) take(id uint32) (notifyHandlers, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	h, ok := n.handlers[id]
	delete(n.handlers, id)
	delete(n.portalIDs, id)
	return h, ok
}

func (n *notifyState) get(id uint32) (notifyHandlers, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	h, ok := n.handlers[id]
	return h, ok
}

// forget drops the callbacks of a notification that is closed.
func (n *notifyState) forget(id uint32) { n.take(id) }

// portalID gives the number and the name that the portal has for a notification: the
// one that is replaced, or a new one.
func (n *notifyState) portalID(replaces uint32) (uint32, string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if name, ok := n.portalIDs[replaces]; ok && replaces != 0 {
		return replaces, name
	}
	n.next++
	if n.portalIDs == nil {
		n.portalIDs = map[uint32]string{}
	}
	name := "glutter-" + strconv.FormatUint(uint64(n.next), 10)
	n.portalIDs[n.next] = name
	return n.next, name
}

func (n *notifyState) portalName(id uint32) (string, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	name, ok := n.portalIDs[id]
	return name, ok
}

func (n *notifyState) idOfPortalName(name string) (uint32, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for id, v := range n.portalIDs {
		if v == name {
			return id, true
		}
	}
	return 0, false
}

// followNotifications starts following the signals of the notification service and of
// the portal, once for the app: they run until the process ends.
func (s *services) followNotifications(bus dbus.Bus) error {
	s.notify.mu.Lock()
	defer s.notify.mu.Unlock()
	if s.notify.started {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	for _, match := range []dbus.SignalMatch{
		{Path: notificationsPath, Interface: notificationsIfc},
		{Path: dbus.PortalPath, Interface: portalNotificationIfc, Member: "ActionInvoked"},
	} {
		sigs, err := bus.Subscribe(ctx, match)
		if err != nil {
			cancel()
			return errors.NewErr("NotificationsUnavailable", err)
		}
		go s.deliverNotificationSignals(sigs)
	}
	s.notify.started, s.notify.cancel = true, cancel
	return nil
}

// deliverNotificationSignals gives each signal to the callbacks of its notification,
// on the UI thread.
func (s *services) deliverNotificationSignals(sigs <-chan *godbus.Signal) {
	for sig := range sigs {
		switch sig.Name {
		case notificationsIfc + ".ActionInvoked":
			if len(sig.Body) < 2 {
				continue
			}
			id, ok1 := sig.Body[0].(uint32)
			key, ok2 := sig.Body[1].(string)
			if ok1 && ok2 {
				s.callOnUIThread(id, func(h notifyHandlers) {
					if h.onAction != nil {
						h.onAction(key)
					}
				}, false)
			}
		case portalNotificationIfc + ".ActionInvoked":
			if len(sig.Body) < 2 {
				continue
			}
			name, ok1 := sig.Body[0].(string)
			key, ok2 := sig.Body[1].(string)
			if !ok1 || !ok2 {
				continue
			}
			if id, found := s.notify.idOfPortalName(name); found {
				s.callOnUIThread(id, func(h notifyHandlers) {
					if h.onAction != nil {
						h.onAction(key)
					}
				}, false)
			}
		case notificationsIfc + ".NotificationClosed":
			if len(sig.Body) < 2 {
				continue
			}
			id, ok1 := sig.Body[0].(uint32)
			code, ok2 := sig.Body[1].(uint32)
			if ok1 && ok2 {
				reason := closeReason(code)
				s.callOnUIThread(id, func(h notifyHandlers) {
					if h.onClosed != nil {
						h.onClosed(reason)
					}
				}, true)
			}
		}
	}
}

func closeReason(code uint32) NotificationCloseReason {
	switch code {
	case 1:
		return NotificationClosedExpired
	case 2:
		return NotificationClosedDismissed
	case 3:
		return NotificationClosedByApp
	}
	return NotificationClosedUnknown
}

// callOnUIThread runs fn with the callbacks of a notification on the UI thread; a
// notification that is closed has its callbacks dropped. A callback that panics is
// reported.
func (s *services) callOnUIThread(id uint32, fn func(notifyHandlers), closing bool) {
	var h notifyHandlers
	var ok bool
	if closing {
		h, ok = s.notify.take(id)
	} else {
		h, ok = s.notify.get(id)
	}
	if !ok {
		return
	}
	s.binding.Post(func() {
		foundation.Guard(s.binding.ErrorReporter(), foundation.CallbackPanicked, "while handling a notification",
			func() []any { return []any{"notification", id} }, func() { fn(h) })
	})
}
