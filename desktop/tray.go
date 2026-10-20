package desktop

import (
	"context"
	"image"
	"os"
	"reflect"
	"strconv"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
	xdraw "golang.org/x/image/draw"
)

const (
	watcherDest  = "org.kde.StatusNotifierWatcher"
	watcherPath  = godbus.ObjectPath("/StatusNotifierWatcher")
	watcherIface = "org.kde.StatusNotifierWatcher"

	itemPath  = godbus.ObjectPath("/StatusNotifierItem")
	itemIface = "org.kde.StatusNotifierItem"
	menuPath  = godbus.ObjectPath("/MenuBar")
	menuIface = "com.canonical.dbusmenu"
	// noMenuPath is what the protocol says in Menu for an item without a menu.
	noMenuPath = godbus.ObjectPath("/NO_DBUSMENU")
)

// TrayStatus is how the desktop shows a system tray icon.
//
// Flutter: no counterpart
type TrayStatus int

// The values of TrayStatus.
const (
	// TrayStatusUnspecified is TrayStatusActive.
	//
	// Flutter: no counterpart
	TrayStatusUnspecified TrayStatus = iota
	// TrayStatusActive is an icon that is shown.
	//
	// Flutter: no counterpart
	TrayStatusActive
	// TrayStatusPassive is an icon the desktop may hide, because nothing happens.
	//
	// Flutter: no counterpart
	TrayStatusPassive
	// TrayStatusNeedsAttention shows the attention icon, and desktops draw the eye to it.
	//
	// Flutter: no counterpart
	TrayStatusNeedsAttention
)

func (s TrayStatus) protocol() string {
	switch s {
	case TrayStatusPassive:
		return "Passive"
	case TrayStatusNeedsAttention:
		return "NeedsAttention"
	}
	return "Active"
}

// TraySizes are the sizes in pixels of the pictures that TrayIconFromRGBA makes.
//
// Flutter: no counterpart
var TraySizes = [...]int{16, 22, 24, 32, 48}

// TrayIconFromRGBA makes the icon of a system tray from one square picture: the
// picture is scaled to each size of TraySizes (16, 22, 24, 32 and 48 pixels), so the
// desktop picks the one that fits its panel. rgba holds size*size*4 bytes, the rows
// from the top, each pixel as red, green, blue and alpha, not premultiplied. It fails
// with TrayIconInvalid when rgba has another length.
//
// Flutter: no counterpart
func TrayIconFromRGBA(size int, rgba []byte) (ui.WindowIcon, error) {
	if size <= 0 || len(rgba) != size*size*4 {
		return ui.WindowIcon{}, errors.New("TrayIconInvalid", "a tray icon picture must have size*size*4 bytes and a positive size",
			"size", size, "bytes", len(rgba))
	}
	src := &image.NRGBA{Pix: rgba, Stride: size * 4, Rect: image.Rect(0, 0, size, size)}
	var icon ui.WindowIcon
	for _, s := range TraySizes {
		dst := image.NewNRGBA(image.Rect(0, 0, s, s))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
		icon.Images = append(icon.Images, ui.WindowIconImage{Size: s, Scale: 1, RGBA: dst.Pix})
	}
	return icon, nil
}

// SystemTrayOpts are the optional parameters of NewSystemTray.
//
// Flutter: no counterpart
type SystemTrayOpts struct {
	// Key is the widget's key.
	Key foundation.Key
	// ID names the icon to the desktop (the Id of the StatusNotifierItem). The default is
	// the application ID of the window, or "glutter".
	ID string
	// Title is the name of the app in the places that list the icons; the default is
	// the application ID of the window, or "Glutter".
	Title string
	// Tooltip is the title of the tip that shows when the pointer rests on the icon.
	Tooltip string
	// TooltipBody is the text under the title of the tip.
	TooltipBody string
	// Status is how the icon is shown.
	Status TrayStatus
	// AttentionIcon is the icon that is shown with TrayStatusNeedsAttention.
	AttentionIcon ui.WindowIcon
	// OnActivate is called, on the UI thread, when the user clicks the icon (the primary
	// action). Without it, and with a Menu, a click opens the menu.
	OnActivate func()
	// OnSecondaryActivate is called when the user clicks the icon with the middle button.
	OnSecondaryActivate func()
	// OnScroll is called when the user scrolls over the icon: delta is the amount, and
	// horizontal says the direction.
	OnScroll func(delta int, horizontal bool)
	// Menu is the menu that opens from the icon.
	Menu []TrayMenuEntry
}

// SystemTray puts an icon in the system tray (the notification area of the panel)
// while it is in the widget tree, and keeps the icon, the tip, the status and the
// menu up to date when it is rebuilt with other values. It builds its child. It uses
// the StatusNotifierItem protocol with a dbusmenu menu (SPEC §8.9); KDE, XFCE, Cinnamon
// and others show it, and GNOME shows it only with the AppIndicator extension. When no
// desktop shows icons, the error TrayUnavailable is reported once through the error
// reporter of the binding and the widget is just its child. An app has one tray at a
// time; a second one reports TrayDuplicate.
//
// Flutter: no counterpart
type SystemTray struct {
	widgets.StatefulWidget
	icon  ui.WindowIcon
	child widgets.Widget
	o     SystemTrayOpts
}

// NewSystemTray creates the widget. The icon is a name of the icon theme, pictures
// (TrayIconFromRGBA), or both; child is built in place of the widget and may be nil.
//
// Flutter: no counterpart
func NewSystemTray(icon ui.WindowIcon, child widgets.Widget, opts ...SystemTrayOpts) *SystemTray {
	o := firstOpts(opts)
	return &SystemTray{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), icon: icon, child: child, o: o}
}

// Icon is the icon of the tray.
//
// Flutter: no counterpart
func (w *SystemTray) Icon() ui.WindowIcon { return w.icon }

// Child is the widget that is built in place of the tray.
//
// Flutter: no counterpart
func (w *SystemTray) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: no counterpart
func (w *SystemTray) CreateState() widgets.State { return &systemTrayState{} }

type systemTrayState struct {
	widgets.StateOf[*SystemTray]
	appID string
	tray  *tray
	// tried is whether the tray was asked for: a tray that failed is not asked again.
	tried bool
}

func (s *systemTrayState) DidChangeDependencies() {
	if s.tried {
		return
	}
	s.tried = true
	if v := widgets.MaybeViewOf(s.Context()); v != nil {
		s.appID = v.PlatformDispatcher().AppID()
	}
	b := widgets.WidgetsBindingOf(s.Context())
	t, err := newTray(servicesOf(b), s.model())
	if err != nil {
		if rep := b.ErrorReporter(); rep != nil {
			rep(err)
		}
		return
	}
	s.tray = t
	s.Go(t.run)
}

func (s *systemTrayState) DidUpdateWidget(*SystemTray) {
	if s.tray != nil {
		s.tray.update(s.model())
	}
}

func (s *systemTrayState) Dispose() {
	if s.tray != nil {
		s.tray.release()
	}
}

func (s *systemTrayState) Build(widgets.BuildContext) widgets.Widget {
	if c := s.Widget().child; c != nil {
		return c
	}
	return widgets.NewSizedBoxShrink(nil)
}

// model is what the widget asks the tray to be.
func (s *systemTrayState) model() *trayModel {
	w := s.Widget()
	id, title := w.o.ID, w.o.Title
	if id == "" {
		id = s.appID
	}
	if id == "" {
		id = "glutter"
	}
	if title == "" {
		title = s.appID
	}
	if title == "" {
		title = "Glutter"
	}
	m := &trayModel{
		data: trayData{
			ID: id, Title: title, Status: w.o.Status.protocol(),
			Tooltip: w.o.Tooltip, TooltipBody: w.o.TooltipBody,
			Menu: noMenuPath,
		},
		onActivate: w.o.OnActivate, onSecondary: w.o.OnSecondaryActivate, onScroll: w.o.OnScroll,
	}
	m.data.IconName, m.data.Icon = pixmapsOf(w.icon)
	m.data.AttentionName, m.data.Attention = pixmapsOf(w.o.AttentionIcon)
	if len(w.o.Menu) > 0 {
		m.menu = buildMenu(w.o.Menu)
		m.data.Menu = menuPath
		m.data.ItemIsMenu = w.o.OnActivate == nil
	}
	return m
}

// trayPixmap is the type (iiay) of a picture of the protocol: the width, the height
// and the pixels in the order alpha, red, green and blue (ARGB32 in network byte
// order), not premultiplied.
type trayPixmap struct {
	Width, Height int32
	Data          []byte
}

// trayToolTip is the type (sa(iiay)ss) of the tip.
type trayToolTip struct {
	IconName    string
	Icon        []trayPixmap
	Title       string
	Description string
}

// pixmapsOf makes the pictures of the protocol from an icon; the pictures that
// are wrong are left out.
func pixmapsOf(icon ui.WindowIcon) (string, []trayPixmap) {
	pix := []trayPixmap{}
	for _, img := range icon.Images {
		if img.Size <= 0 || len(img.RGBA) != img.Size*img.Size*4 {
			continue
		}
		data := make([]byte, len(img.RGBA))
		for i := 0; i < len(data); i += 4 {
			data[i], data[i+1], data[i+2], data[i+3] = img.RGBA[i+3], img.RGBA[i], img.RGBA[i+1], img.RGBA[i+2]
		}
		pix = append(pix, trayPixmap{Width: int32(img.Size), Height: int32(img.Size), Data: data})
	}
	return icon.Name, pix
}

// trayData is what the desktop can see of the tray, apart from the menu items; two
// of them are equal when nothing has to be told.
type trayData struct {
	ID, Title, Status       string
	Tooltip, TooltipBody    string
	IconName, AttentionName string
	Icon, Attention         []trayPixmap
	ItemIsMenu              bool
	Menu                    godbus.ObjectPath
}

type trayModel struct {
	data trayData
	menu *menuTree

	onActivate, onSecondary func()
	onScroll                func(int, bool)
}

// what changed between two models, and so what has to be told.
type trayChange uint8

const (
	changeTitle trayChange = 1 << iota
	changeIcon
	changeAttention
	changeToolTip
	changeStatus
	changeMenu
	changeProps // a property without a signal of its own
)

func (m *trayModel) menuEntries() *menuTree {
	if m.menu == nil {
		return buildMenu(nil)
	}
	return m.menu
}

// diff says what differs from old.
func (m *trayModel) diff(old *trayModel) trayChange {
	var c trayChange
	a, b := &old.data, &m.data
	if a.Title != b.Title {
		c |= changeTitle
	}
	if a.IconName != b.IconName || !reflect.DeepEqual(a.Icon, b.Icon) {
		c |= changeIcon
	}
	if a.AttentionName != b.AttentionName || !reflect.DeepEqual(a.Attention, b.Attention) {
		c |= changeAttention
	}
	if a.Tooltip != b.Tooltip || a.TooltipBody != b.TooltipBody {
		c |= changeToolTip
	}
	if a.Status != b.Status {
		c |= changeStatus
	}
	if a.ItemIsMenu != b.ItemIsMenu || a.Menu != b.Menu || a.ID != b.ID {
		c |= changeProps
	}
	if !reflect.DeepEqual(old.menuEntries().root, m.menuEntries().root) {
		c |= changeMenu
	}
	return c
}

// trayState is what the services know of the trays.
type trayState struct {
	mu     sync.Mutex
	active bool
	count  int
	// done is closed when the worker of the last tray has cleaned up.
	done chan struct{}
}

// tray is the icon of the app: it serves the StatusNotifierItem and the menu on the
// bus, from a worker goroutine, and takes the changes that the widget hands over.
type tray struct {
	s    *services
	name string
	prev <-chan struct{}
	done chan struct{}
	wake chan struct{}

	props     *dbus.Properties
	menuProps *dbus.Properties

	mu       sync.Mutex
	cur      *trayModel
	revision uint32
	dirty    trayChange
}

// newTray reserves the tray of the app. It fails with TrayDuplicate when there is one.
func newTray(s *services, m *trayModel) (*tray, error) {
	s.tray.mu.Lock()
	defer s.tray.mu.Unlock()
	if s.tray.active {
		return nil, errors.New("TrayDuplicate", "the app already has a system tray; an app has one at a time")
	}
	s.tray.active = true
	s.tray.count++
	t := &tray{
		s: s, name: "org.kde.StatusNotifierItem-" + strconv.Itoa(os.Getpid()) + "-" + strconv.Itoa(s.tray.count),
		prev: s.tray.done, done: make(chan struct{}), wake: make(chan struct{}, 1),
		cur: m, revision: 1,
	}
	s.tray.done = t.done
	return t, nil
}

// update hands the tray a new model; what changed is told on the bus by the worker.
func (t *tray) update(m *trayModel) {
	t.mu.Lock()
	c := m.diff(t.cur)
	if c&changeMenu != 0 {
		t.revision++
	}
	t.cur = m
	t.dirty |= c
	t.mu.Unlock()
	select {
	case t.wake <- struct{}{}:
	default:
	}
}

// release lets a new tray be made; it is called when the widget goes away, and the
// worker, whose context ends with the state, takes the icon off the bus.
func (t *tray) release() {
	t.s.tray.mu.Lock()
	t.s.tray.active = false
	t.s.tray.mu.Unlock()
}

// fail reports that the tray cannot be shown, once, on the UI thread.
func (t *tray) fail(err error) {
	t.s.binding.Post(func() {
		if rep := t.s.binding.ErrorReporter(); rep != nil {
			rep(err)
		}
	})
}

// run is the worker of the tray: it makes the icon appear and keeps the bus up to
// date until ctx ends.
func (t *tray) run(ctx context.Context) {
	defer close(t.done)
	if t.prev != nil {
		select {
		case <-t.prev:
		case <-ctx.Done():
			return
		}
	}
	bus, err := t.s.connect(ctx)
	if err != nil {
		if ctx.Err() == nil {
			t.fail(errors.NewErr("TrayUnavailable", err))
		}
		return
	}
	if err := t.serve(ctx, bus); err != nil && ctx.Err() == nil {
		t.fail(err)
	}
}

// serve exports the objects, takes the name, registers with the watcher and then
// follows the changes until ctx ends.
func (t *tray) serve(ctx context.Context, bus dbus.Bus) error {
	owner, err := bus.Subscribe(ctx, dbus.SignalMatch{Sender: "org.freedesktop.DBus", Path: "/org/freedesktop/DBus",
		Interface: "org.freedesktop.DBus", Member: "NameOwnerChanged"})
	if err != nil {
		return err
	}
	t.props = dbus.NewProperties(bus, itemPath)
	t.menuProps = dbus.NewProperties(bus, menuPath)
	t.mu.Lock()
	cur := t.cur
	t.mu.Unlock()
	t.storeAll(cur)
	t.menuProps.Store(menuIface, "Version", uint32(3))
	t.menuProps.Store(menuIface, "TextDirection", "ltr")
	t.menuProps.Store(menuIface, "Status", "normal")
	t.menuProps.Store(menuIface, "IconThemePath", []string{})

	defer func() {
		t.props.Unexport()
		t.menuProps.Unexport()
		bus.Unexport(itemPath, itemIface)
		bus.Unexport(menuPath, menuIface)
		_ = bus.ReleaseName(t.name)
	}()
	for _, e := range []struct {
		recv  any
		path  godbus.ObjectPath
		iface string
	}{{&trayItem{t}, itemPath, itemIface}, {&trayMenu{t}, menuPath, menuIface}} {
		if err := bus.Export(e.recv, e.path, e.iface); err != nil {
			return err
		}
	}
	if err := t.props.Export(); err != nil {
		return err
	}
	if err := t.menuProps.Export(); err != nil {
		return err
	}
	if err := bus.RequestName(t.name); err != nil {
		return err
	}
	if err := t.register(ctx, bus); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.wake:
			t.flush(bus)
		case sig, ok := <-owner:
			if !ok {
				return nil
			}
			// The watcher started again (the panel restarted): it has forgotten the icon.
			if len(sig.Body) == 3 && sig.Body[0] == watcherDest && sig.Body[2] != "" {
				if err := t.register(ctx, bus); err != nil {
					return err
				}
			}
		}
	}
}

// register tells the watcher about the icon. A desktop without a watcher is
// TrayUnavailable.
func (t *tray) register(ctx context.Context, bus dbus.Bus) error {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if _, err := bus.Call(ctx, watcherDest, watcherPath, watcherIface+".RegisterStatusNotifierItem", t.name); err != nil {
		if portalMissing(err) {
			return errors.NewErr("TrayUnavailable", err)
		}
		return errors.NewErr("TrayRegisterFailed", err)
	}
	return nil
}

// storeAll puts every property of the model in the store.
func (t *tray) storeAll(m *trayModel) {
	d := &m.data
	p := func(name string, v any) { t.props.Store(itemIface, name, v) }
	p("Category", "ApplicationStatus")
	p("Id", d.ID)
	p("Title", d.Title)
	p("Status", d.Status)
	p("WindowId", uint32(0))
	p("IconThemePath", "")
	p("Menu", d.Menu)
	p("ItemIsMenu", d.ItemIsMenu)
	p("IconName", d.IconName)
	p("IconPixmap", d.Icon)
	p("OverlayIconName", "")
	p("OverlayIconPixmap", []trayPixmap{})
	p("AttentionIconName", d.AttentionName)
	p("AttentionIconPixmap", d.Attention)
	p("AttentionMovieName", "")
	p("ToolTip", trayToolTip{Icon: []trayPixmap{}, Title: d.Tooltip, Description: d.TooltipBody})
}

// flush tells the desktop what changed since the last time.
func (t *tray) flush(bus dbus.Bus) {
	t.mu.Lock()
	m, c, rev := t.cur, t.dirty, t.revision
	t.dirty = 0
	t.mu.Unlock()
	if c == 0 {
		return
	}
	d := &m.data
	emit := func(name string, values ...any) {
		// A signal that cannot be sent is lost like one that nobody hears: the next
		// property read has the right value.
		_ = bus.EmitSignal(itemPath, itemIface+"."+name, values...)
	}
	set := func(name string, v any) { _ = t.props.Update(itemIface, name, v) }
	if c&changeTitle != 0 {
		set("Title", d.Title)
		emit("NewTitle")
	}
	if c&changeIcon != 0 {
		set("IconName", d.IconName)
		set("IconPixmap", d.Icon)
		emit("NewIcon")
	}
	if c&changeAttention != 0 {
		set("AttentionIconName", d.AttentionName)
		set("AttentionIconPixmap", d.Attention)
		emit("NewAttentionIcon")
	}
	if c&changeToolTip != 0 {
		set("ToolTip", trayToolTip{Icon: []trayPixmap{}, Title: d.Tooltip, Description: d.TooltipBody})
		emit("NewToolTip")
	}
	if c&changeStatus != 0 {
		set("Status", d.Status)
		emit("NewStatus", d.Status)
	}
	if c&changeProps != 0 {
		set("Id", d.ID)
		set("Menu", d.Menu)
		set("ItemIsMenu", d.ItemIsMenu)
	}
	if c&changeMenu != 0 {
		_ = bus.EmitSignal(menuPath, menuIface+".LayoutUpdated", rev, int32(0))
	}
}

// current is the model now.
func (t *tray) current() (*trayModel, uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cur, t.revision
}

// call runs fn on the UI thread, as a callback of the app.
func (t *tray) call(what string, fn func()) {
	if fn == nil {
		return
	}
	b := t.s.binding
	b.Post(func() {
		foundation.Guard(b.ErrorReporter(), foundation.CallbackPanicked, "while handling "+what+" of the system tray", nil, fn)
	})
}

// trayItem is the org.kde.StatusNotifierItem object that the desktop calls.
type trayItem struct{ t *tray }

// Activate is called when the user clicks the icon.
func (i *trayItem) Activate(x, y int32) *godbus.Error {
	m, _ := i.t.current()
	i.t.call("a click", m.onActivate)
	return nil
}

// SecondaryActivate is called when the user clicks the icon with the middle button.
func (i *trayItem) SecondaryActivate(x, y int32) *godbus.Error {
	m, _ := i.t.current()
	i.t.call("a middle click", m.onSecondary)
	return nil
}

// ContextMenu is called by hosts that draw the menu themselves from dbusmenu; there is
// nothing to do, the menu is on the bus.
func (i *trayItem) ContextMenu(x, y int32) *godbus.Error { return nil }

// Scroll is called when the user scrolls over the icon.
func (i *trayItem) Scroll(delta int32, orientation string) *godbus.Error {
	m, _ := i.t.current()
	if m.onScroll != nil {
		on, horizontal := m.onScroll, orientation == "horizontal"
		i.t.call("a scroll", func() { on(int(delta), horizontal) })
	}
	return nil
}

// menuEvent is the type (isvu) of an event of EventGroup.
type menuEvent struct {
	ID        int32
	EventID   string
	Data      godbus.Variant
	Timestamp uint32
}

// menuProperties is the type (ia{sv}) of GetGroupProperties.
type menuProperties struct {
	ID    int32
	Props map[string]godbus.Variant
}

// trayMenu is the com.canonical.dbusmenu object that the desktop calls.
type trayMenu struct{ t *tray }

func unknownItem(id int32) *godbus.Error {
	return &godbus.Error{Name: "org.freedesktop.DBus.Error.InvalidArgs", Body: []any{"no menu item " + strconv.Itoa(int(id))}}
}

// GetLayout is the layout of the menu below parent, depth levels deep (-1 is all).
func (m *trayMenu) GetLayout(parent, depth int32, names []string) (uint32, menuLayout, *godbus.Error) {
	cur, rev := m.t.current()
	n, ok := cur.menuEntries().nodes[parent]
	if !ok {
		return 0, menuLayout{}, unknownItem(parent)
	}
	return rev, n.layout(depth, names), nil
}

// GetGroupProperties are the properties of the items.
func (m *trayMenu) GetGroupProperties(ids []int32, names []string) ([]menuProperties, *godbus.Error) {
	cur, _ := m.t.current()
	tree := cur.menuEntries()
	out := []menuProperties{}
	if len(ids) == 0 {
		for id := int32(0); int(id) < len(tree.nodes); id++ {
			if n, ok := tree.nodes[id]; ok {
				out = append(out, menuProperties{id, filterProps(n.Props, names)})
			}
		}
		return out, nil
	}
	for _, id := range ids {
		if n, ok := tree.nodes[id]; ok {
			out = append(out, menuProperties{id, filterProps(n.Props, names)})
		}
	}
	return out, nil
}

// GetProperty is one property of an item.
func (m *trayMenu) GetProperty(id int32, name string) (godbus.Variant, *godbus.Error) {
	cur, _ := m.t.current()
	n, ok := cur.menuEntries().nodes[id]
	if !ok {
		return godbus.Variant{}, unknownItem(id)
	}
	v, ok := n.Props[name]
	if !ok {
		return godbus.Variant{}, &godbus.Error{Name: "org.freedesktop.DBus.Error.InvalidArgs", Body: []any{"no property " + name}}
	}
	return v, nil
}

// Event is called when the user does something to an item; "clicked" calls the item's function.
func (m *trayMenu) Event(id int32, eventID string, _ godbus.Variant, _ uint32) *godbus.Error {
	cur, _ := m.t.current()
	tree := cur.menuEntries()
	if _, ok := tree.nodes[id]; !ok {
		return unknownItem(id)
	}
	if eventID == "clicked" {
		m.t.call("a menu item", tree.pressed[id])
	}
	return nil
}

// EventGroup is Event for several items; the IDs that are not items are returned.
func (m *trayMenu) EventGroup(events []menuEvent) ([]int32, *godbus.Error) {
	cur, _ := m.t.current()
	tree := cur.menuEntries()
	missing := []int32{}
	for _, e := range events {
		if _, ok := tree.nodes[e.ID]; !ok {
			missing = append(missing, e.ID)
			continue
		}
		if e.EventID == "clicked" {
			m.t.call("a menu item", tree.pressed[e.ID])
		}
	}
	if len(missing) == len(events) && len(events) > 0 {
		return nil, unknownItem(missing[0])
	}
	return missing, nil
}

// AboutToShow is called before a submenu opens; the menu is always up to date.
func (m *trayMenu) AboutToShow(id int32) (bool, *godbus.Error) { return false, nil }

// AboutToShowGroup is AboutToShow for several items.
func (m *trayMenu) AboutToShowGroup(ids []int32) ([]int32, []int32, *godbus.Error) {
	return []int32{}, []int32{}, nil
}
