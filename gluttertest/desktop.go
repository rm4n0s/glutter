package gluttertest

import (
	"bytes"
	"net/url"
	"reflect"
	"slices"
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
// The fake replaces the desktop of every platform, whatever the target platform of the
// test (foundation.DefaultTargetPlatform) and whatever the host: package desktop then
// talks to the fake's bus and never to the desktop of macOS, Windows, the web, Android or
// iOS. Its bus speaks Linux's protocols (D-Bus, StatusNotifierItem), but the calls of the
// app are the same on every platform, so a test of an app's tray need not run as Linux.
//
// Flutter: no counterpart
type FakeDesktop struct {
	t   *Tester
	bus *fakebus.Bus

	mu         sync.Mutex
	notes      []*FakeNotification
	lastID     uint32
	registered []string
	// choosers are the file dialogs that the app opened, oldest first; pending are the
	// request paths of those the test has not answered, with the index of each in choosers.
	choosers []FakeFileChooser
	pending  []pendingChooser
}

// pendingChooser is a file dialog that waits for the test's answer.
type pendingChooser struct {
	path  godbus.ObjectPath
	index int
}

// FakeFileChooser is what the app asked of a file dialog (desktop.OpenFiles, SaveFile or
// SaveFiles), as the fake desktop's file chooser portal read it from the call.
//
// Flutter: no counterpart
type FakeFileChooser struct {
	// Method is the method of the portal that the app called: "OpenFile" (desktop.OpenFiles),
	// "SaveFile" or "SaveFiles".
	Method string
	// Title is the title of the dialog; empty for the portal's default.
	Title string
	// AcceptLabel is the text of the button that accepts the dialog; empty for the default.
	AcceptLabel string
	// Modal is false when the app asked for a dialog that leaves its window usable
	// (NotModal).
	Modal bool
	// Multiple and Directory are the options of OpenFiles: more than one choice, and
	// folders instead of files.
	Multiple, Directory bool
	// Filters are the kinds of file that the dialog offers, in order; CurrentFilter is the
	// name of the one chosen when it opens, empty without filters.
	Filters       []desktop.FileFilter
	CurrentFilter string
	// CurrentFolder is the folder the dialog opens in; empty when the app gave none.
	CurrentFolder string
	// CurrentName and CurrentFile are the name and the path of the file that SaveFile
	// starts with.
	CurrentName, CurrentFile string
	// Names are the names of the files of SaveFiles, in order.
	Names []string
	// Answered says that the test answered the dialog (AnswerFileChooser or
	// CancelFileChooser).
	Answered bool
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

// FakeDesktop installs a fake desktop in the app and returns it, in place of the
// desktop of every platform (desktop.InstallBus). Call it before the app first uses
// package desktop (before the SystemTray is pumped).
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
	fc := decodeFileChooser(c)
	f.mu.Lock()
	f.choosers = append(f.choosers, fc)
	f.pending = append(f.pending, pendingChooser{path: path, index: len(f.choosers) - 1})
	f.mu.Unlock()
	return []any{path}, nil
}

// decodeFileChooser reads the arguments of a call of the file chooser portal:
// (parent_window, title, options).
func decodeFileChooser(c fakebus.Call) FakeFileChooser {
	fc := FakeFileChooser{Method: strings.TrimPrefix(c.Method, chooserIfc+"."), Modal: true}
	if len(c.Args) < 3 {
		return fc
	}
	fc.Title, _ = c.Args[1].(string)
	opts, _ := c.Args[2].(map[string]godbus.Variant)
	for key, v := range opts {
		switch key {
		case "accept_label":
			fc.AcceptLabel, _ = v.Value().(string)
		case "modal":
			if modal, ok := v.Value().(bool); ok {
				fc.Modal = modal
			}
		case "multiple":
			fc.Multiple, _ = v.Value().(bool)
		case "directory":
			fc.Directory, _ = v.Value().(bool)
		case "current_folder":
			fc.CurrentFolder = portalPath(v.Value())
		case "current_name":
			fc.CurrentName, _ = v.Value().(string)
		case "current_file":
			fc.CurrentFile = portalPath(v.Value())
		case "files":
			if files, ok := v.Value().([][]byte); ok {
				for _, name := range files {
					fc.Names = append(fc.Names, portalPath(name))
				}
			}
		case "filters":
			rv := reflect.ValueOf(v.Value())
			if rv.Kind() == reflect.Slice {
				for i := 0; i < rv.Len(); i++ {
					fc.Filters = append(fc.Filters, portalFilter(rv.Index(i)))
				}
			}
		case "current_filter":
			fc.CurrentFilter = portalFilter(reflect.ValueOf(v.Value())).Name
		}
	}
	return fc
}

// portalPath is a path or a name the way the portals take it, bytes with a closing NUL
// (D-Bus type ay), as a string.
func portalPath(v any) string {
	b, _ := v.([]byte)
	return string(bytes.TrimRight(b, "\x00"))
}

// portalFilter reads a filter of the portal, (sa(us)): its name and its rules, a glob
// (0) or a MIME type (1) each. A bus gives a structure as a slice of its fields and a call
// on the fake bus as the struct that was sent, so both are read by position.
func portalFilter(v reflect.Value) desktop.FileFilter {
	var out desktop.FileFilter
	fields := structFields(v)
	if len(fields) < 2 {
		return out
	}
	out.Name, _ = fields[0].Interface().(string)
	rules := fields[1]
	for rules.Kind() == reflect.Interface {
		rules = rules.Elem()
	}
	if rules.Kind() != reflect.Slice {
		return out
	}
	for i := 0; i < rules.Len(); i++ {
		rule := structFields(rules.Index(i))
		if len(rule) < 2 {
			continue
		}
		kind, _ := rule[0].Interface().(uint32)
		pattern, _ := rule[1].Interface().(string)
		if kind == 1 {
			out.MIMETypes = append(out.MIMETypes, pattern)
		} else {
			out.Patterns = append(out.Patterns, pattern)
		}
	}
	return out
}

// structFields are the fields of a D-Bus structure, given as a struct or as a slice of
// its values.
func structFields(v reflect.Value) []reflect.Value {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	var out []reflect.Value
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			out = append(out, v.Field(i))
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			f := v.Index(i)
			for f.Kind() == reflect.Interface && !f.IsNil() {
				f = f.Elem()
			}
			out = append(out, f)
		}
	}
	return out
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
	return len(f.pending)
}

// FileChoosers are the file dialogs that the app has opened, oldest first, with what it
// asked of each (the folder, the filters, the title, ...); the dialogs that the test
// answered stay in the list, with Answered set.
//
// Flutter: no counterpart
func (f *FakeDesktop) FileChoosers() []FakeFileChooser {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]FakeFileChooser, len(f.choosers))
	for i, fc := range f.choosers {
		out[i] = fc
		out[i].Filters = slices.Clone(fc.Filters)
		out[i].Names = slices.Clone(fc.Names)
	}
	return out
}

func (f *FakeDesktop) answerChooser(code uint32, results map[string]godbus.Variant) {
	f.t.tb.Helper()
	f.mu.Lock()
	if len(f.pending) == 0 {
		f.mu.Unlock()
		f.t.tb.Fatalf("no file dialog is open")
		return
	}
	p := f.pending[0]
	f.pending = f.pending[1:]
	f.choosers[p.index].Answered = true
	path := p.path
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
