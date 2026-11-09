package macos

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// This file is the plain-Go part of the notifications (SPEC §29.9): the numbering and the replacing of
// notifications, the categories that carry their actions, the timeouts, and what a response means to the app. The
// calls to UserNotifications are behind notifyBackend, which notify_darwin.go implements and the tests replace.

// notifyRequest is what is handed to the notification center: the content, and the category that has the buttons.
type notifyRequest struct {
	// Identifier names the notification to the center; a request with the identifier of one that is shown replaces it.
	Identifier  string
	Title, Body string
	// Quiet makes no sound (UrgencyLow); Critical shows at the time-sensitive level (UrgencyCritical).
	Quiet, Critical bool
	// Category is the category that has the actions, "" for a notification without buttons that wants no dismiss
	// event either.
	Category string
	// IconPath is the file of the attachment, "" for none.
	IconPath string
}

// notifyCategory is a set of actions that notifications share, made from the buttons of a notification.
type notifyCategory struct {
	ID      string
	Actions []mobile.Action
}

// notifyBackend is what the notifications ask of the system.
type notifyBackend interface {
	// Ready says whether the app can show notifications: it needs a bundle with an identifier. The reason says why not.
	Ready() (ok bool, reason string)
	// Authorize asks the user for the permission, the first time, and returns whether it is granted. The errors of
	// the backend are NotificationsUnavailable already.
	Authorize(ctx context.Context) (granted bool, err error)
	// SetCategories gives the center the whole set of categories.
	SetCategories(cats []notifyCategory) error
	// Add shows the request, or replaces the one with its identifier.
	Add(ctx context.Context, req notifyRequest) error
	// Remove takes the notification with the identifier off the screen and out of the center.
	Remove(identifier string)
}

// liveNote is a notification that is shown.
type liveNote struct {
	id   uint32
	spec mobile.NotificationSpec
	stop func() bool
	// gen counts the replacements of the notification, so that a timer of the one that was replaced does nothing.
	gen int
}

// notifier is the numbering and the life of the notifications of the app. Its methods may be called from any
// goroutine.
type notifier struct {
	b     notifyBackend
	after func(d time.Duration, fn func()) (stop func() bool)

	mu    sync.Mutex
	next  uint32
	live  map[uint32]*liveNote
	cats  map[string]notifyCategory
	order []string
	// asked says the permission was asked for in this run; granted is the answer.
	asked, granted bool
}

func newNotifier(b notifyBackend) *notifier {
	return &notifier{
		b: b, live: map[uint32]*liveNote{}, cats: map[string]notifyCategory{},
		after: func(d time.Duration, fn func()) func() bool { return time.AfterFunc(d, fn).Stop },
	}
}

// identifierOf is the identifier that the center knows the notification by.
func identifierOf(id uint32) string { return "glutter." + strconv.FormatUint(uint64(id), 10) }

// idOfIdentifier is the inverse: 0 when the identifier is not one of this app's.
func idOfIdentifier(identifier string) uint32 {
	rest, ok := cutPrefix(identifier, "glutter.")
	if !ok {
		return 0
	}
	n, err := strconv.ParseUint(rest, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(n)
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return s, false
}

// categoryOf is the category for the actions: its identifier is made from the actions, so that the same buttons
// are the same category. The action "default" is the click on the notification and is not a button. A notification
// with no buttons has a category all the same, with no action, which makes the center report its dismissal.
func categoryOf(actions []mobile.Action) notifyCategory {
	var buttons []mobile.Action
	h := sha256.New()
	for _, a := range actions {
		if a.ID == mobile.DefaultAction {
			continue
		}
		buttons = append(buttons, a)
		h.Write([]byte(a.ID + "\x00" + a.Label + "\x00"))
	}
	return notifyCategory{ID: "glutter.cat." + hex.EncodeToString(h.Sum(nil))[:16], Actions: buttons}
}

// requestOf makes the request of a notification.
func requestOf(id uint32, spec mobile.NotificationSpec, cat notifyCategory) notifyRequest {
	return notifyRequest{
		Identifier: identifierOf(id), Title: spec.Title, Body: spec.Body,
		Quiet: spec.Urgency == mobile.UrgencyLow, Critical: spec.Urgency == mobile.UrgencyCritical,
		Category: cat.ID, IconPath: spec.Icon,
	}
}

// show shows the notification: it is the number that is returned, the same number when the notification replaces
// one that is shown. It fails with NotificationsUnavailable when the app has no bundle, when the user refused the
// permission, and when the center refuses the request.
func (n *notifier) show(ctx context.Context, spec mobile.NotificationSpec) (uint32, error) {
	if ok, reason := n.b.Ready(); !ok {
		return 0, errors.New(tagNotificationsUnavailable, "the app cannot show notifications", "reason", reason)
	}
	if err := n.authorize(ctx); err != nil {
		return 0, err
	}
	cat := categoryOf(spec.Actions)
	n.mu.Lock()
	id := spec.ReplacesID
	old := n.live[id]
	if old == nil {
		n.next++
		id = n.next
	}
	cats, newCat := n.addCategory(cat)
	n.mu.Unlock()
	if newCat {
		if err := n.b.SetCategories(cats); err != nil {
			return 0, err
		}
	}
	if err := n.b.Add(ctx, requestOf(id, spec, cat)); err != nil {
		return 0, err
	}
	n.mu.Lock()
	ln := n.live[id]
	if ln == nil {
		ln = &liveNote{id: id}
		n.live[id] = ln
	} else if ln.stop != nil {
		ln.stop()
		ln.stop = nil
	}
	ln.spec = spec
	ln.gen++
	if spec.Timeout > 0 {
		gen := ln.gen
		ln.stop = n.after(spec.Timeout, func() { n.expire(id, gen) })
	}
	n.mu.Unlock()
	return id, nil
}

// addCategory adds the category to the set; it returns the whole set and whether the category is new. The caller
// holds the mutex.
func (n *notifier) addCategory(cat notifyCategory) (all []notifyCategory, isNew bool) {
	if _, ok := n.cats[cat.ID]; !ok {
		n.cats[cat.ID] = cat
		n.order = append(n.order, cat.ID)
		isNew = true
	}
	for _, id := range n.order {
		all = append(all, n.cats[id])
	}
	return all, isNew
}

// authorize asks for the permission once in a run: a refusal is remembered, as the user decides in the settings of
// the system after it.
func (n *notifier) authorize(ctx context.Context) error {
	n.mu.Lock()
	asked, granted := n.asked, n.granted
	n.mu.Unlock()
	if asked {
		if !granted {
			return errors.New(tagNotificationsUnavailable, "the user did not allow notifications; the setting is in System Settings", "reason", "denied")
		}
		return nil
	}
	ok, err := n.b.Authorize(ctx)
	if err != nil {
		return err
	}
	n.mu.Lock()
	n.asked, n.granted = true, ok
	n.mu.Unlock()
	if !ok {
		return errors.New(tagNotificationsUnavailable, "the user did not allow notifications; the setting is in System Settings", "reason", "denied")
	}
	return nil
}

// expire is the end of the timeout of a notification: it is taken off the screen and the app hears of it.
func (n *notifier) expire(id uint32, gen int) {
	n.mu.Lock()
	ln := n.live[id]
	if ln == nil || ln.gen != gen {
		n.mu.Unlock()
		return
	}
	delete(n.live, id)
	spec := ln.spec
	n.mu.Unlock()
	n.b.Remove(identifierOf(id))
	if spec.OnClose != nil {
		spec.OnClose(mobile.CloseExpired)
	}
}

// close is the close that the app asks for. A notification that is gone is not an error.
func (n *notifier) close(id uint32) {
	n.mu.Lock()
	ln := n.live[id]
	if ln == nil {
		n.mu.Unlock()
		return
	}
	delete(n.live, id)
	if ln.stop != nil {
		ln.stop()
	}
	spec := ln.spec
	n.mu.Unlock()
	n.b.Remove(identifierOf(id))
	if spec.OnClose != nil {
		spec.OnClose(mobile.CloseByApp)
	}
}

// response is what the user did with the notification: actionID is the action that was chosen, mobile.DefaultAction
// for a click on the notification itself, and "" for a dismissal. It runs the callbacks of the app.
func (n *notifier) response(identifier, actionID string) {
	id := idOfIdentifier(identifier)
	n.mu.Lock()
	ln := n.live[id]
	if ln == nil {
		n.mu.Unlock()
		return
	}
	delete(n.live, id)
	if ln.stop != nil {
		ln.stop()
	}
	spec := ln.spec
	n.mu.Unlock()
	if actionID != "" && spec.OnAction != nil && n.hasAction(spec, actionID) {
		spec.OnAction(actionID)
	}
	if actionID != "" {
		n.b.Remove(identifierOf(id)) // a click takes the notification off the center
	}
	if spec.OnClose != nil {
		spec.OnClose(mobile.CloseDismissed)
	}
}

// hasAction says that the notification has the action: the click is "default" only when the app listed it.
func (n *notifier) hasAction(spec mobile.NotificationSpec, id string) bool {
	for _, a := range spec.Actions {
		if a.ID == id {
			return true
		}
	}
	return false
}

// shutdown stops the timers; the notifications that are shown stay in the center, as the user's.
func (n *notifier) shutdown() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ln := range n.live {
		if ln.stop != nil {
			ln.stop()
		}
	}
	n.live = map[uint32]*liveNote{}
}

// The tag of the error that this file makes (rm4n0s/errors, R5); package desktop matches it by value.
const tagNotificationsUnavailable = "NotificationsUnavailable"
