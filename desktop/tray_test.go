package desktop_test

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	watcherDest = "org.kde.StatusNotifierWatcher"
	watcherPath = godbus.ObjectPath("/StatusNotifierWatcher")
	registerM   = "org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem"
	itemPath    = godbus.ObjectPath("/StatusNotifierItem")
	itemIface   = "org.kde.StatusNotifierItem"
	menuPath    = godbus.ObjectPath("/MenuBar")
	menuIface   = "com.canonical.dbusmenu"
)

var trayName = "org.kde.StatusNotifierItem-" + strconv.Itoa(os.Getpid()) + "-1"

// watcherOn makes the fake bus act as the StatusNotifierWatcher.
func watcherOn(b *fakebus.Bus) {
	b.Handle(watcherDest, watcherPath, registerM, func(fakebus.Call) ([]any, error) { return nil, nil })
}

func registrations(b *fakebus.Bus) []string {
	var names []string
	for _, c := range b.Calls() {
		if c.Method == registerM {
			names = append(names, c.Args[0].(string))
		}
	}
	return names
}

func trayTree(icon ui.WindowIcon, o desktop.SystemTrayOpts) widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR, desktop.NewSystemTray(icon, widgets.NewText("tray child"), o))
}

// startTray pumps a tray on a fake bus with a watcher, and waits until the icon is registered.
func startTray(t *testing.T, icon ui.WindowIcon, o desktop.SystemTrayOpts) (*gluttertest.Tester, widgets.BuildContext, *fakebus.Bus) {
	t.Helper()
	ts, bc := app(t)
	b := fakebus.New(unique)
	watcherOn(b)
	desktop.UseBus(bc, b)
	ts.PumpWidget(trayTree(icon, o))
	waitFor(t, ts, "the registration", func() bool { return len(registrations(b)) > 0 })
	return ts, bc, b
}

func props(t *testing.T, b *fakebus.Bus) map[string]godbus.Variant {
	t.Helper()
	got, err := b.Invoke(itemPath, dbus.PropertiesIface, "GetAll", itemIface)
	if err != nil {
		t.Fatal(err)
	}
	return got[0].(map[string]godbus.Variant)
}

// pixels is a size x size picture of one color.
func pixels(size int, r, g, bl, a byte) ui.WindowIconImage {
	rgba := make([]byte, size*size*4)
	for i := 0; i < len(rgba); i += 4 {
		rgba[i], rgba[i+1], rgba[i+2], rgba[i+3] = r, g, bl, a
	}
	return ui.WindowIconImage{Size: size, Scale: 1, RGBA: rgba}
}

func signalsNamed(b *fakebus.Bus, name string) int {
	n := 0
	for _, s := range b.Signals() {
		if s.Name == name {
			n++
		}
	}
	return n
}

func TestSystemTrayRegistersAndServesTheItem(t *testing.T) {
	icon := ui.WindowIcon{Name: "mail-unread", Images: []ui.WindowIconImage{pixels(2, 1, 2, 3, 4)}}
	ts, _, b := startTray(t, icon, desktop.SystemTrayOpts{
		ID: "app.id", Title: "My app", Tooltip: "Hello", TooltipBody: "body", Status: desktop.TrayStatusPassive,
		Menu: []desktop.TrayMenuEntry{desktop.NewTrayMenuItem("Open", nil)},
	})
	if got := registrations(b); !reflect.DeepEqual(got, []string{trayName}) {
		t.Fatalf("registered %v, want %s", got, trayName)
	}
	if !reflect.DeepEqual(b.Names(), []string{trayName}) {
		t.Errorf("names = %v", b.Names())
	}
	for _, e := range []struct {
		path  godbus.ObjectPath
		iface string
	}{{itemPath, itemIface}, {menuPath, menuIface}, {itemPath, dbus.PropertiesIface}, {menuPath, dbus.PropertiesIface}} {
		if !b.Exported(e.path, e.iface) {
			t.Errorf("%s %s is not exported", e.path, e.iface)
		}
	}
	ts.Element(gluttertest.ByText("tray child")) // fails the test when the child is not built

	p := props(t, b)
	want := map[string]any{
		"Category": "ApplicationStatus", "Id": "app.id", "Title": "My app", "Status": "Passive", "WindowId": uint32(0),
		"IconName": "mail-unread", "AttentionIconName": "", "OverlayIconName": "",
		"Menu": menuPath, "ItemIsMenu": true,
	}
	for k, v := range want {
		if got := p[k].Value(); !reflect.DeepEqual(got, v) {
			t.Errorf("%s = %#v, want %#v", k, got, v)
		}
	}
	// The picture is ARGB, in network byte order.
	pix := reflect.ValueOf(p["IconPixmap"].Value())
	if pix.Len() != 1 {
		t.Fatalf("IconPixmap = %v", p["IconPixmap"])
	}
	first := pix.Index(0)
	if first.Field(0).Int() != 2 || first.Field(1).Int() != 2 || !reflect.DeepEqual(first.Field(2).Bytes()[:4], []byte{4, 1, 2, 3}) {
		t.Errorf("pixmap = %v", first)
	}
	tip := reflect.ValueOf(p["ToolTip"].Value())
	if tip.Field(2).String() != "Hello" || tip.Field(3).String() != "body" {
		t.Errorf("ToolTip = %v", tip)
	}
	if got, err := b.Invoke(menuPath, dbus.PropertiesIface, "Get", menuIface, "Version"); err != nil || got[0].(godbus.Variant).Value() != uint32(3) {
		t.Errorf("menu Version = %v, %v", got, err)
	}
}

func TestSystemTrayDefaults(t *testing.T) {
	tests := []struct {
		name       string
		opts       desktop.SystemTrayOpts
		itemIsMenu bool
		menu       godbus.ObjectPath
	}{
		{"nothing", desktop.SystemTrayOpts{}, false, "/NO_DBUSMENU"},
		{"a menu opens on a click", desktop.SystemTrayOpts{Menu: []desktop.TrayMenuEntry{desktop.NewTrayMenuSeparator()}}, true, menuPath},
		{"a click activates", desktop.SystemTrayOpts{OnActivate: func() {}, Menu: []desktop.TrayMenuEntry{desktop.NewTrayMenuSeparator()}}, false, menuPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, b := startTray(t, ui.WindowIcon{Name: "x"}, tt.opts)
			p := props(t, b)
			if p["Id"].Value() != "glutter" || p["Title"].Value() != "Glutter" || p["Status"].Value() != "Active" {
				t.Errorf("Id %v, Title %v, Status %v", p["Id"], p["Title"], p["Status"])
			}
			if p["ItemIsMenu"].Value() != tt.itemIsMenu || p["Menu"].Value() != tt.menu {
				t.Errorf("ItemIsMenu %v, Menu %v", p["ItemIsMenu"], p["Menu"])
			}
		})
	}
}

func menuItems(t *testing.T, b *fakebus.Bus, parent, depth int32) (uint32, []any) {
	t.Helper()
	got, err := b.Invoke(menuPath, menuIface, "GetLayout", parent, depth, []string{})
	if err != nil {
		t.Fatal(err)
	}
	return got[0].(uint32), []any{got[1]}
}

// describe walks a layout (ia{sv}av) in the shape of its Go value.
func describe(v reflect.Value) map[string]any {
	d := map[string]any{"id": int32(v.Field(0).Int())}
	props := map[string]any{}
	for _, k := range v.Field(1).MapKeys() {
		props[k.String()] = v.Field(1).MapIndex(k).Interface().(godbus.Variant).Value()
	}
	d["props"] = props
	var kids []any
	for i := 0; i < v.Field(2).Len(); i++ {
		kids = append(kids, describe(reflect.ValueOf(v.Field(2).Index(i).Interface().(godbus.Variant).Value())))
	}
	d["children"] = kids
	return d
}

func TestSystemTrayMenuLayout(t *testing.T) {
	_, _, b := startTray(t, ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{Menu: []desktop.TrayMenuEntry{
		desktop.NewTrayMenuItem("Open_file", func() {}, desktop.TrayMenuItemOpts{Icon: "document-open"}),
		desktop.NewTrayMenuSeparator(),
		desktop.NewTrayMenuSubmenu("More", []desktop.TrayMenuEntry{
			desktop.NewTrayMenuItem("Dark", nil, desktop.TrayMenuItemOpts{Checkable: true, Checked: true}),
			desktop.NewTrayMenuItem("Left", nil, desktop.TrayMenuItemOpts{Radio: true}),
		}),
		desktop.NewTrayMenuItem("Quit", nil, desktop.TrayMenuItemOpts{Disabled: true, Hidden: true}),
	}})
	rev, layouts := menuItems(t, b, 0, -1)
	if rev == 0 {
		t.Error("revision 0")
	}
	got := describe(reflect.ValueOf(layouts[0]))
	want := map[string]any{
		"id": int32(0), "props": map[string]any{"children-display": "submenu"},
		"children": []any{
			map[string]any{"id": int32(1), "props": map[string]any{"label": "Open__file", "enabled": true, "icon-name": "document-open"}, "children": []any(nil)},
			map[string]any{"id": int32(2), "props": map[string]any{"type": "separator"}, "children": []any(nil)},
			map[string]any{"id": int32(3), "props": map[string]any{"label": "More", "enabled": true, "children-display": "submenu"}, "children": []any{
				map[string]any{"id": int32(4), "props": map[string]any{"label": "Dark", "enabled": true, "toggle-type": "checkmark", "toggle-state": int32(1)}, "children": []any(nil)},
				map[string]any{"id": int32(5), "props": map[string]any{"label": "Left", "enabled": true, "toggle-type": "radio", "toggle-state": int32(0)}, "children": []any(nil)},
			}},
			map[string]any{"id": int32(6), "props": map[string]any{"label": "Quit", "enabled": false, "visible": false}, "children": []any(nil)},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("layout\n got %#v\nwant %#v", got, want)
	}

	// Depth 0 is the item alone; depth 1 has no grandchildren; names filter the properties.
	_, l0 := menuItems(t, b, 3, 0)
	if d := describe(reflect.ValueOf(l0[0])); len(d["children"].([]any)) != 0 || d["id"] != int32(3) {
		t.Errorf("depth 0: %v", d)
	}
	g, err := b.Invoke(menuPath, menuIface, "GetLayout", int32(0), int32(1), []string{"label"})
	if err != nil {
		t.Fatal(err)
	}
	d := describe(reflect.ValueOf(g[1]))
	kids := d["children"].([]any)
	if len(kids) != 4 || len(kids[2].(map[string]any)["children"].([]any)) != 0 ||
		!reflect.DeepEqual(kids[0].(map[string]any)["props"], map[string]any{"label": "Open__file"}) {
		t.Errorf("depth 1 with names: %v", d)
	}
	if _, err := b.Invoke(menuPath, menuIface, "GetLayout", int32(99), int32(-1), []string{}); err == nil {
		t.Error("the layout of an item that is not there")
	}

	gp, err := b.Invoke(menuPath, menuIface, "GetGroupProperties", []int32{2, 99}, []string{"type"})
	if err != nil || reflect.ValueOf(gp[0]).Len() != 1 {
		t.Fatalf("GetGroupProperties = %v, %v", gp, err)
	}
	gp, err = b.Invoke(menuPath, menuIface, "GetGroupProperties", []int32{}, []string{})
	if err != nil || reflect.ValueOf(gp[0]).Len() != 7 {
		t.Errorf("all properties = %v, %v", gp, err)
	}
	pr, err := b.Invoke(menuPath, menuIface, "GetProperty", int32(1), "label")
	if err != nil || pr[0].(godbus.Variant).Value() != "Open__file" {
		t.Errorf("GetProperty = %v, %v", pr, err)
	}
	if _, err := b.Invoke(menuPath, menuIface, "GetProperty", int32(1), "nope"); err == nil {
		t.Error("a property that is not there")
	}
	if _, err := b.Invoke(menuPath, menuIface, "GetProperty", int32(77), "label"); err == nil {
		t.Error("a property of an item that is not there")
	}
	if as, err := b.Invoke(menuPath, menuIface, "AboutToShow", int32(3)); err != nil || as[0] != false {
		t.Errorf("AboutToShow = %v, %v", as, err)
	}
	if as, err := b.Invoke(menuPath, menuIface, "AboutToShowGroup", []int32{3}); err != nil || len(as) != 2 {
		t.Errorf("AboutToShowGroup = %v, %v", as, err)
	}
}

// recorder collects the calls of the callbacks of a tray, which run on the UI thread.
type recorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *recorder) add(s string) func() {
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.calls = append(r.calls, s)
	}
}

func (r *recorder) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func TestSystemTrayCallbacksRunOnTheUIThread(t *testing.T) {
	r := &recorder{}
	ts, _, b := startTray(t, ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{
		OnActivate: r.add("activate"), OnSecondaryActivate: r.add("secondary"),
		OnScroll: func(delta int, horizontal bool) {
			r.add("scroll " + strconv.Itoa(delta) + map[bool]string{true: " h", false: " v"}[horizontal])()
		},
		Menu: []desktop.TrayMenuEntry{
			desktop.NewTrayMenuItem("One", r.add("one")),
			desktop.NewTrayMenuItem("Two", r.add("two")),
			desktop.NewTrayMenuItem("None", nil),
		},
	})
	invoke := func(path godbus.ObjectPath, iface, method string, args ...any) {
		t.Helper()
		if _, err := b.Invoke(path, iface, method, args...); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	invoke(itemPath, itemIface, "Activate", int32(1), int32(2))
	invoke(itemPath, itemIface, "SecondaryActivate", int32(1), int32(2))
	invoke(itemPath, itemIface, "Scroll", int32(-3), "vertical")
	invoke(itemPath, itemIface, "Scroll", int32(2), "horizontal")
	invoke(itemPath, itemIface, "ContextMenu", int32(1), int32(2))
	invoke(menuPath, menuIface, "Event", int32(1), "clicked", godbus.MakeVariant(""), uint32(0))
	invoke(menuPath, menuIface, "Event", int32(2), "hovered", godbus.MakeVariant(""), uint32(0))
	invoke(menuPath, menuIface, "Event", int32(3), "clicked", godbus.MakeVariant(""), uint32(0))
	ev := []struct {
		ID        int32
		EventID   string
		Data      godbus.Variant
		Timestamp uint32
	}{{2, "clicked", godbus.MakeVariant(""), 0}, {55, "clicked", godbus.MakeVariant(""), 0}}
	got, err := b.Invoke(menuPath, menuIface, "EventGroup", ev)
	if err != nil || !reflect.DeepEqual(got[0], []int32{55}) {
		t.Errorf("EventGroup = %v, %v", got, err)
	}
	if _, err := b.Invoke(menuPath, menuIface, "Event", int32(55), "clicked", godbus.MakeVariant(""), uint32(0)); err == nil {
		t.Error("an event for an item that is not there")
	}

	ts.FlushPosted()
	want := []string{"activate", "secondary", "scroll -3 v", "scroll 2 h", "one", "two"}
	if got := r.all(); !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}
}

func TestSystemTrayCallbackPanicIsReported(t *testing.T) {
	ts, _, b := startTray(t, ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{OnActivate: func() { panic("boom") }})
	if _, err := b.Invoke(itemPath, itemIface, "Activate", int32(0), int32(0)); err != nil {
		t.Fatal(err)
	}
	ts.FlushPosted()
	assertTag(t, ts.TakeError(), "CallbackPanicked")
}

func TestSystemTrayUpdatesTellTheDesktop(t *testing.T) {
	r := &recorder{}
	base := desktop.SystemTrayOpts{
		Title: "T", Tooltip: "tip", OnActivate: r.add("first"),
		Menu: []desktop.TrayMenuEntry{desktop.NewTrayMenuItem("A", r.add("a1"))},
	}
	ts, _, b := startTray(t, ui.WindowIcon{Name: "one"}, base)
	count := func(name string) int { return signalsNamed(b, name) }
	if len(b.Signals()) != 0 {
		t.Fatalf("signals at the start: %+v", b.Signals())
	}

	// The same data with other functions tells nothing, and the new functions are used.
	same := base
	same.OnActivate = r.add("second")
	same.Menu = []desktop.TrayMenuEntry{desktop.NewTrayMenuItem("A", r.add("a2"))}
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "one"}, same))
	if _, err := b.Invoke(itemPath, itemIface, "Activate", int32(0), int32(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Invoke(menuPath, menuIface, "Event", int32(1), "clicked", godbus.MakeVariant(""), uint32(0)); err != nil {
		t.Fatal(err)
	}
	ts.FlushPosted()
	if got := r.all(); !reflect.DeepEqual(got, []string{"second", "a2"}) {
		t.Errorf("calls = %v", got)
	}
	if n := len(b.Signals()); n != 0 {
		t.Errorf("%d signals for a rebuild with the same data: %+v", n, b.Signals())
	}

	steps := []struct {
		name   string
		change func(o *desktop.SystemTrayOpts, icon *ui.WindowIcon)
		signal string
		check  func(p map[string]godbus.Variant) bool
	}{
		{"title", func(o *desktop.SystemTrayOpts, _ *ui.WindowIcon) { o.Title = "T2" }, itemIface + ".NewTitle",
			func(p map[string]godbus.Variant) bool { return p["Title"].Value() == "T2" }},
		{"icon", func(_ *desktop.SystemTrayOpts, i *ui.WindowIcon) { i.Name = "two" }, itemIface + ".NewIcon",
			func(p map[string]godbus.Variant) bool { return p["IconName"].Value() == "two" }},
		{"attention icon", func(o *desktop.SystemTrayOpts, _ *ui.WindowIcon) { o.AttentionIcon = ui.WindowIcon{Name: "alert"} }, itemIface + ".NewAttentionIcon",
			func(p map[string]godbus.Variant) bool { return p["AttentionIconName"].Value() == "alert" }},
		{"tooltip", func(o *desktop.SystemTrayOpts, _ *ui.WindowIcon) { o.TooltipBody = "more" }, itemIface + ".NewToolTip",
			func(p map[string]godbus.Variant) bool {
				return reflect.ValueOf(p["ToolTip"].Value()).Field(3).String() == "more"
			}},
		{"status", func(o *desktop.SystemTrayOpts, _ *ui.WindowIcon) { o.Status = desktop.TrayStatusNeedsAttention }, itemIface + ".NewStatus",
			func(p map[string]godbus.Variant) bool { return p["Status"].Value() == "NeedsAttention" }},
		{"menu", func(o *desktop.SystemTrayOpts, _ *ui.WindowIcon) {
			o.Menu = []desktop.TrayMenuEntry{desktop.NewTrayMenuItem("B", nil)}
		}, menuIface + ".LayoutUpdated", func(map[string]godbus.Variant) bool { return true }},
	}
	opts, icon := same, ui.WindowIcon{Name: "one"}
	for _, st := range steps {
		before := count(st.signal)
		st.change(&opts, &icon)
		ts.PumpWidget(trayTree(icon, opts))
		waitFor(t, ts, st.name+" signal", func() bool { return count(st.signal) > before })
		if !st.check(props(t, b)) {
			t.Errorf("%s: the properties do not have the new value", st.name)
		}
		if n := count(st.signal) - before; n != 1 {
			t.Errorf("%s: %d signals, want 1", st.name, n)
		}
	}
	// The layout signal carries the new revision and the root.
	var last godbus.Signal
	for _, s := range b.Signals() {
		if s.Name == menuIface+".LayoutUpdated" {
			last = s
		}
	}
	rev, _ := menuItems(t, b, 0, 0)
	if last.Path != menuPath || !reflect.DeepEqual(last.Body, []any{rev, int32(0)}) {
		t.Errorf("LayoutUpdated = %+v, revision %d", last, rev)
	}
	// Dropping the menu changes the properties the host reads.
	opts.Menu, opts.OnActivate = nil, nil
	ts.PumpWidget(trayTree(icon, opts))
	waitFor(t, ts, "the menu to go", func() bool { return props(t, b)["Menu"].Value() == godbus.ObjectPath("/NO_DBUSMENU") })
}

func TestSystemTrayLeavesWhenItIsRemoved(t *testing.T) {
	ts, _, b := startTray(t, ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("plain")))
	waitFor(t, ts, "the name to be released", func() bool { return len(b.Names()) == 0 })
	for _, e := range []struct {
		path  godbus.ObjectPath
		iface string
	}{{itemPath, itemIface}, {menuPath, menuIface}, {itemPath, dbus.PropertiesIface}, {menuPath, dbus.PropertiesIface}} {
		if b.Exported(e.path, e.iface) {
			t.Errorf("%s %s is still exported", e.path, e.iface)
		}
	}
	// A new tray takes its place, with the next name.
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "y"}, desktop.SystemTrayOpts{}))
	waitFor(t, ts, "the second registration", func() bool { return len(registrations(b)) == 2 })
	if got := registrations(b)[1]; got != "org.kde.StatusNotifierItem-"+strconv.Itoa(os.Getpid())+"-2" {
		t.Errorf("second name = %s", got)
	}
}

func TestSystemTrayRegistersAgainWhenTheWatcherRestarts(t *testing.T) {
	ts, _, b := startTray(t, ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{})
	other := func(name, oldOwner, newOwner string) {
		b.Emit(&godbus.Signal{Path: "/org/freedesktop/DBus", Name: "org.freedesktop.DBus.NameOwnerChanged", Body: []any{name, oldOwner, newOwner}})
	}
	other("org.example.Other", "", ":1.8") // not the watcher
	other(watcherDest, ":1.4", "")         // the watcher left
	other(watcherDest, "", ":1.9")         // and came back
	waitFor(t, ts, "the second registration", func() bool { return len(registrations(b)) == 2 })
	if got := registrations(b); got[0] != got[1] {
		t.Errorf("registered %v", got)
	}
}

func TestSystemTrayErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(bc widgets.BuildContext)
		tag   string
	}{
		{"no watcher on the bus", func(bc widgets.BuildContext) { desktop.UseBus(bc, fakebus.New(unique)) }, "TrayUnavailable"},
		{"no session bus", func(bc widgets.BuildContext) {
			desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) { return nil, errors.New("DBusConnectFailed", "no bus") })
		}, "TrayUnavailable"},
		{"the watcher refuses", func(bc widgets.BuildContext) {
			b := fakebus.New(unique)
			b.Handle(watcherDest, watcherPath, registerM, func(fakebus.Call) ([]any, error) {
				return nil, &godbus.Error{Name: "org.freedesktop.DBus.Error.AccessDenied"}
			})
			desktop.UseBus(bc, b)
		}, "TrayRegisterFailed"},
		{"the name is taken", func(bc widgets.BuildContext) {
			b := fakebus.New(unique)
			watcherOn(b)
			b.Take(trayName)
			desktop.UseBus(bc, b)
		}, "DBusNameTaken"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			tt.setup(bc)
			ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{}))
			var err error
			waitFor(t, ts, "the error", func() bool { err = ts.TakeError(); return err != nil })
			assertTag(t, err, tt.tag)
			// The widget is its child, and the error is reported once.
			ts.Element(gluttertest.ByText("tray child"))
			ts.FlushPosted()
			if again := ts.TakeError(); again != nil {
				t.Errorf("a second error: %v", again)
			}
		})
	}
}

func TestSystemTrayDuplicate(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	watcherOn(b)
	desktop.UseBus(bc, b)
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
		desktop.NewSystemTray(ui.WindowIcon{Name: "a"}, widgets.NewText("first")),
		desktop.NewSystemTray(ui.WindowIcon{Name: "b"}, widgets.NewText("second")),
	})))
	assertTag(t, ts.TakeError(), "TrayDuplicate")
	waitFor(t, ts, "the registration", func() bool { return len(registrations(b)) > 0 })
	if n := len(registrations(b)); n != 1 {
		t.Errorf("%d registrations", n)
	}
}

func TestSystemTrayInvalidIcon(t *testing.T) {
	_, _, b := startTray(t, ui.WindowIcon{Name: "x", Images: []ui.WindowIconImage{{Size: 4, RGBA: []byte{1}}, pixels(1, 9, 9, 9, 9)}}, desktop.SystemTrayOpts{})
	pix := reflect.ValueOf(props(t, b)["IconPixmap"].Value())
	if pix.Len() != 1 || pix.Index(0).Field(0).Int() != 1 {
		t.Errorf("IconPixmap = %v, want only the valid picture", pix)
	}
}

func TestTrayIconFromRGBA(t *testing.T) {
	icon, err := desktop.TrayIconFromRGBA(64, pixels(64, 255, 0, 0, 255).RGBA)
	if err != nil {
		t.Fatal(err)
	}
	var sizes []int
	for _, img := range icon.Images {
		sizes = append(sizes, img.Size)
		if len(img.RGBA) != img.Size*img.Size*4 {
			t.Errorf("size %d has %d bytes", img.Size, len(img.RGBA))
		}
		mid := (img.Size/2*img.Size + img.Size/2) * 4
		if got := img.RGBA[mid : mid+4]; !reflect.DeepEqual(got, []byte{255, 0, 0, 255}) {
			t.Errorf("size %d: middle pixel %v", img.Size, got)
		}
	}
	if !reflect.DeepEqual(sizes, []int{16, 22, 24, 32, 48}) {
		t.Errorf("sizes = %v", sizes)
	}
	if err := icon.Validate(); err != nil {
		t.Error(err)
	}
	for _, bad := range []struct {
		size int
		n    int
	}{{0, 0}, {4, 10}, {-1, 4}} {
		_, err := desktop.TrayIconFromRGBA(bad.size, make([]byte, bad.n))
		assertTag(t, err, "TrayIconInvalid")
	}
}

func TestTrayMenuItemOptions(t *testing.T) {
	defer func() {
		appErr, ok := errors.FromError(recoverErr(recover()))
		if !ok || appErr.Tag != "MultipleOptions" {
			t.Errorf("recovered %v", appErr)
		}
	}()
	desktop.NewTrayMenuItem("a", nil, desktop.TrayMenuItemOpts{}, desktop.TrayMenuItemOpts{})
}

func recoverErr(r any) error {
	err, _ := r.(error)
	return err
}
