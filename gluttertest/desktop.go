package gluttertest

import (
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
)

// FakeNotificationAction is a button of a FakeNotification.
//
// Flutter: no counterpart
type FakeNotificationAction struct {
	// ID is the key that the app gave the action.
	ID string
	// Label is the text of the button.
	Label string
}

// FakeNotification is a notification that the app showed to the fake desktop.
//
// Flutter: no counterpart
type FakeNotification struct {
	// ID is the number that the fake desktop gave the notification.
	ID uint32
	// AppName, Icon, Summary and Body are what the app sent.
	AppName, Icon, Summary, Body string
	// Actions are the buttons.
	Actions []FakeNotificationAction
	// Urgency is the urgency hint: 0 low, 1 normal, 2 critical.
	Urgency byte
	// Closed says that the app or the test closed it.
	Closed bool
}

// FakeDesktop is a desktop for the tests of an app that uses package desktop: a
// notification service, a StatusNotifierWatcher for the system tray, and a file
// chooser portal, on an in-process bus. The code of the app runs as it does for real;
// the test looks at what it sent and answers the way the user would.
//
// Flutter: no counterpart
type FakeDesktop struct {
	t   *Tester
	bus *fakebus.Bus

	mu         sync.Mutex
	notes      []*FakeNotification
	lastID     uint32
	registered []string
	choosers   []godbus.ObjectPath
}

const (
	fakeUnique   = ":1.99"
	notifDest    = "org.freedesktop.Notifications"
	notifPath    = godbus.ObjectPath("/org/freedesktop/Notifications")
	trayItemPath = godbus.ObjectPath("/StatusNotifierItem")
	trayItemIfc  = "org.kde.StatusNotifierItem"
	trayMenuPath = godbus.ObjectPath("/MenuBar")
	trayMenuIfc  = "com.canonical.dbusmenu"
	chooserIfc   = "org.freedesktop.portal.FileChooser"
)

// fileChooserMethods are the methods of the file chooser portal that package desktop calls.
var fileChooserMethods = []string{"OpenFile", "SaveFile", "SaveFiles"}

// FakeDesktop installs a fake desktop in the app and returns it. Call it before the
// app first uses package desktop (before the SystemTray is pumped).
//
// Flutter: no counterpart
func (t *Tester) FakeDesktop() *FakeDesktop {
	f := &FakeDesktop{t: t, bus: fakebus.New(fakeUnique)}
	b := f.bus
	b.Handle(notifDest, notifPath, notifDest+".GetCapabilities", func(fakebus.Call) ([]any, error) {
		return []any{[]string{"body", "actions"}}, nil
	})
	b.Handle(notifDest, notifPath, notifDest+".Notify", f.notify)
	b.Handle(notifDest, notifPath, notifDest+".CloseNotification", func(c fakebus.Call) ([]any, error) {
		f.markClosed(c.Args[0].(uint32))
		return nil, nil
	})
	b.Handle("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher", "org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem",
		func(c fakebus.Call) ([]any, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.registered = append(f.registered, c.Args[0].(string))
			return nil, nil
		})
	for _, m := range fileChooserMethods {
		b.Handle(dbus.PortalDest, dbus.PortalPath, chooserIfc+"."+m, f.chooser)
	}
	desktop.InstallBus(t.Binding, b)
	return f
}

// DisableFileChooser takes the file chooser portal away, as on a desktop that has none:
// the calls of desktop.OpenFiles, SaveFile and SaveFiles (the dialogs for files, for
// folders and for saving) fail with FileChooserUnavailable, until the test ends. A dialog
// that is open already stays open.
//
// Flutter: no counterpart
func (f *FakeDesktop) DisableFileChooser() {
	for _, m := range fileChooserMethods {
		f.bus.Unhandle(dbus.PortalDest, dbus.PortalPath, chooserIfc+"."+m)
	}
}

func (f *FakeDesktop) notify(c fakebus.Call) ([]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := &FakeNotification{AppName: c.Args[0].(string), Icon: c.Args[2].(string), Summary: c.Args[3].(string), Body: c.Args[4].(string)}
	actions, _ := c.Args[5].([]string)
	for i := 0; i+1 < len(actions); i += 2 {
		n.Actions = append(n.Actions, FakeNotificationAction{ID: actions[i], Label: actions[i+1]})
	}
	if hints, ok := c.Args[6].(map[string]godbus.Variant); ok {
		n.Urgency, _ = hints["urgency"].Value().(byte)
	}
	if replaces, _ := c.Args[1].(uint32); replaces != 0 {
		for i, old := range f.notes {
			if old.ID == replaces {
				n.ID = replaces
				f.notes[i] = n
				return []any{n.ID}, nil
			}
		}
	}
	f.lastID++
	n.ID = f.lastID
	f.notes = append(f.notes, n)
	return []any{n.ID}, nil
}

func (f *FakeDesktop) markClosed(id uint32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, n := range f.notes {
		if n.ID == id {
			n.Closed = true
		}
	}
}

func (f *FakeDesktop) chooser(c fakebus.Call) ([]any, error) {
	opts, _ := c.Args[2].(map[string]godbus.Variant)
	token, _ := opts["handle_token"].Value().(string)
	path := dbus.PortalRequestPath(fakeUnique, token)
	f.mu.Lock()
	f.choosers = append(f.choosers, path)
	f.mu.Unlock()
	return []any{path}, nil
}

// WaitFor pumps the posted functions until cond is true, for what the app does on
// other goroutines (the calls of package desktop). It fails the test after five seconds.
//
// Flutter: no counterpart
func (f *FakeDesktop) WaitFor(what string, cond func() bool) {
	f.t.tb.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			f.t.tb.Fatalf("timed out waiting for %s", what)
			return
		}
		f.t.FlushPosted()
		time.Sleep(time.Millisecond)
	}
}

// settle runs the functions that a signal makes the app post to the UI thread. The signal
// travels through a goroutine of package desktop, so it waits a moment for the first
// function to arrive; a signal that nobody listens to posts nothing, and the wait ends.
func (f *FakeDesktop) settle() {
	f.t.adopt()
	deadline := time.Now().Add(250 * time.Millisecond)
	for f.t.host.RunPosted() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
}

// Notifications are the notifications that the app showed, oldest first; a
// notification that replaced another takes its place.
//
// Flutter: no counterpart
func (f *FakeDesktop) Notifications() []FakeNotification {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]FakeNotification, len(f.notes))
	for i, n := range f.notes {
		out[i] = *n
	}
	return out
}

// InvokeNotificationAction does what the user does when they choose an action of a
// notification, and runs the app's callback.
//
// Flutter: no counterpart
func (f *FakeDesktop) InvokeNotificationAction(id uint32, action string) {
	f.bus.Emit(&godbus.Signal{Path: notifPath, Name: notifDest + ".ActionInvoked", Body: []any{id, action}})
	f.settle()
}

// CloseNotification closes a notification the way the desktop does, with a reason
// of the specification (1 expired, 2 dismissed by the user, 3 closed by the app), and
// runs the app's callback.
//
// Flutter: no counterpart
func (f *FakeDesktop) CloseNotification(id, reason uint32) {
	f.markClosed(id)
	f.bus.Emit(&godbus.Signal{Path: notifPath, Name: notifDest + ".NotificationClosed", Body: []any{id, reason}})
	f.settle()
}

// TrayRegistered says whether a tray icon registered with the watcher.
//
// Flutter: no counterpart
func (f *FakeDesktop) TrayRegistered() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.registered) > 0
}

// TrayProperty is a property of the tray icon (StatusNotifierItem), such as "Title",
// "Status" or "IconName"; it is nil when there is none.
//
// Flutter: no counterpart
func (f *FakeDesktop) TrayProperty(name string) any {
	got, err := f.bus.Invoke(trayItemPath, dbus.PropertiesIface, "Get", trayItemIfc, name)
	if err != nil {
		return nil
	}
	return got[0].(godbus.Variant).Value()
}

// ActivateTray does what the user does when they click the tray icon.
//
// Flutter: no counterpart
func (f *FakeDesktop) ActivateTray() {
	f.t.tb.Helper()
	if _, err := f.bus.Invoke(trayItemPath, trayItemIfc, "Activate", int32(0), int32(0)); err != nil {
		f.t.tb.Fatalf("clicking the tray icon: %v", err)
	}
	f.t.FlushPosted()
}

// trayEntries reads the first level of the menu of the tray: the id and the properties
// of each entry.
func (f *FakeDesktop) trayEntries() (ids []int32, props []map[string]any) {
	f.t.tb.Helper()
	got, err := f.bus.Invoke(trayMenuPath, trayMenuIfc, "GetLayout", int32(0), int32(1), []string{})
	if err != nil {
		f.t.tb.Fatalf("reading the menu of the tray: %v", err)
		return nil, nil
	}
	root := reflect.ValueOf(got[1])
	kids := root.Field(2)
	for i := 0; i < kids.Len(); i++ {
		kid := reflect.ValueOf(kids.Index(i).Interface().(godbus.Variant).Value())
		ids = append(ids, int32(kid.Field(0).Int()))
		p := map[string]any{}
		for _, k := range kid.Field(1).MapKeys() {
			p[k.String()] = kid.Field(1).MapIndex(k).Interface().(godbus.Variant).Value()
		}
		props = append(props, p)
	}
	return ids, props
}

// TrayMenu are the labels of the first level of the menu of the tray, in order, with
// "-" for a separator.
//
// Flutter: no counterpart
func (f *FakeDesktop) TrayMenu() []string {
	f.t.tb.Helper()
	_, props := f.trayEntries()
	labels := make([]string, 0, len(props))
	for _, p := range props {
		if p["type"] == "separator" {
			labels = append(labels, "-")
			continue
		}
		label, _ := p["label"].(string)
		labels = append(labels, strings.ReplaceAll(label, "__", "_"))
	}
	return labels
}

// ClickTrayMenuItem does what the user does when they choose the item with the label
// in the menu of the tray, and runs the app's callback. It fails the test when
// there is no such item.
//
// Flutter: no counterpart
func (f *FakeDesktop) ClickTrayMenuItem(label string) {
	f.t.tb.Helper()
	ids, _ := f.trayEntries()
	for i, l := range f.TrayMenu() {
		if l != label {
			continue
		}
		if _, err := f.bus.Invoke(trayMenuPath, trayMenuIfc, "Event", ids[i], "clicked", godbus.MakeVariant(""), uint32(0)); err != nil {
			f.t.tb.Fatalf("choosing %q in the menu of the tray: %v", label, err)
		}
		f.t.FlushPosted()
		return
	}
	f.t.tb.Fatalf("the menu of the tray has no item %q; it has %q", label, f.TrayMenu())
}

// PendingFileChoosers is how many file dialogs the app has opened that the test has
// not answered.
//
// Flutter: no counterpart
func (f *FakeDesktop) PendingFileChoosers() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.choosers)
}

func (f *FakeDesktop) answerChooser(code uint32, results map[string]godbus.Variant) {
	f.t.tb.Helper()
	f.mu.Lock()
	if len(f.choosers) == 0 {
		f.mu.Unlock()
		f.t.tb.Fatalf("no file dialog is open")
		return
	}
	path := f.choosers[0]
	f.choosers = f.choosers[1:]
	f.mu.Unlock()
	f.bus.Emit(&godbus.Signal{Path: path, Name: "org.freedesktop.portal.Request.Response", Body: []any{code, results}})
	f.t.FlushPosted()
}

// AnswerFileChooser chooses the paths in the oldest file dialog that is open, the way
// the user does.
//
// Flutter: no counterpart
func (f *FakeDesktop) AnswerFileChooser(paths ...string) {
	f.t.tb.Helper()
	uris := make([]string, len(paths))
	for i, p := range paths {
		uris[i] = (&url.URL{Scheme: "file", Path: p}).String()
	}
	f.answerChooser(0, map[string]godbus.Variant{"uris": godbus.MakeVariant(uris)})
}

// CancelFileChooser closes the oldest file dialog that is open without a choice.
//
// Flutter: no counterpart
func (f *FakeDesktop) CancelFileChooser() {
	f.t.tb.Helper()
	f.answerChooser(1, map[string]godbus.Variant{})
}
