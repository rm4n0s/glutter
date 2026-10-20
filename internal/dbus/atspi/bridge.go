package atspi

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/ui"
)

const (
	a11yBusDest   = "org.a11y.Bus"
	a11yBusPath   = godbus.ObjectPath("/org/a11y/bus")
	a11yStatusIfc = "org.a11y.Status"

	registryDest = "org.a11y.atspi.Registry"

	eventObjectIfc = "org.a11y.atspi.Event.Object"
	eventWindowIfc = "org.a11y.atspi.Event.Window"

	// connectTimeout is how long the connections may take.
	connectTimeout = 5 * time.Second
)

// Host is what the bridge needs of the app. The bridge calls it on the UI thread
// (through Options.Post).
type Host interface {
	// ScreenReaderChanged tells that a screen reader started or stopped: the framework
	// then produces the semantics tree, or stops.
	ScreenReaderChanged(enabled bool)
	// PerformAction asks the framework to perform an action of a node, the request of
	// the assistive technology.
	PerformAction(event ui.SemanticsActionEvent)
}

// Options are the parameters of New.
type Options struct {
	// AppName is the name of the application, as the screen reader says it.
	AppName string
	// Version is the version of the toolkit; it defaults to "1.0".
	Version string
	// Post runs a function on the UI thread, from any goroutine. It is required.
	Post func(func())
	// OpenSession connects to the session bus; nil is dbus.Connect.
	OpenSession func(ctx context.Context) (dbus.Bus, error)
	// OpenA11y connects to the accessibility bus at the address that the session bus
	// gives; nil is dbus.ConnectAddress.
	OpenA11y func(ctx context.Context, address string) (dbus.Bus, error)
	// WindowOrigin is where the window is on the screen, for the coordinates of the
	// screen. Wayland does not tell, and the default is the origin.
	WindowOrigin func(view ui.ViewID) (x, y int)
	// Getenv reads the environment (for the locale); nil is os.Getenv.
	Getenv func(string) string
	// Logger gets the debug messages; nil is slog.Default().
	Logger *slog.Logger
}

// Bridge mirrors the semantics trees of the views as AT-SPI2 objects while a screen
// reader is on. Its methods are for the UI thread, except Close.
//
// Flutter: no counterpart
type Bridge struct {
	opts Options

	mu           sync.Mutex
	model        *Model
	host         Host
	a11y         dbus.Bus
	exported     map[NodeRef]map[string]bool
	registryRoot objRef
	appID        int32

	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a bridge that does nothing until Start.
//
// Flutter: no counterpart
func New(opts Options) *Bridge {
	if opts.AppName == "" {
		opts.AppName = toolkitName
	}
	if opts.Version == "" {
		opts.Version = "1.0"
	}
	if opts.OpenSession == nil {
		opts.OpenSession = dbus.Connect
	}
	if opts.OpenA11y == nil {
		opts.OpenA11y = dbus.ConnectAddress
	}
	if opts.Getenv == nil {
		opts.Getenv = os.Getenv
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Bridge{opts: opts, model: NewModel(), exported: map[NodeRef]map[string]bool{}}
}

func (b *Bridge) busName() string {
	if b.a11y == nil {
		return ""
	}
	return b.a11y.UniqueName()
}

func (b *Bridge) locale() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := b.opts.Getenv(k); v != "" {
			return v
		}
	}
	return "C"
}

// perform asks the host to perform the action on the UI thread.
func (b *Bridge) perform(ev ui.SemanticsActionEvent) {
	host := b.host
	if host == nil {
		return
	}
	b.opts.Post(func() { host.PerformAction(ev) })
}

// Start follows the screen reader flag of the session bus in the background, until ctx
// ends or Close is called. A desktop without the accessibility service is not an
// error: the bridge stays off (and says so in the debug log).
func (b *Bridge) Start(ctx context.Context, host Host) {
	b.mu.Lock()
	if b.cancel != nil {
		b.mu.Unlock()
		return
	}
	b.host = host
	ctx, b.cancel = context.WithCancel(ctx)
	b.done = make(chan struct{})
	b.mu.Unlock()
	go b.run(ctx)
}

// Close stops the bridge and takes the objects off the bus.
func (b *Bridge) Close() {
	b.mu.Lock()
	cancel, done := b.cancel, b.done
	b.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

// run watches the screen reader flag.
func (b *Bridge) run(ctx context.Context) {
	defer close(b.done)
	log := b.opts.Logger
	openCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	session, err := b.opts.OpenSession(openCtx)
	cancel()
	if err != nil {
		log.Debug("accessibility is off: no session bus", "error", err)
		return
	}
	defer func() { _ = session.Close() }()
	// Subscribe before reading, so that a change in between is not lost.
	sigs, err := session.Subscribe(ctx, dbus.SignalMatch{Sender: a11yBusDest, Path: a11yBusPath,
		Interface: ifaceProperties, Member: "PropertiesChanged"})
	if err != nil {
		log.Debug("accessibility is off: cannot follow the screen reader flag", "error", err)
		return
	}
	enabled := false
	if on, err := readScreenReader(ctx, session); err != nil {
		log.Debug("accessibility is off: the screen reader flag is not there", "error", err)
	} else {
		enabled = on
	}
	b.setEnabled(ctx, session, enabled)
	defer b.setEnabled(context.Background(), session, false)
	for {
		select {
		case <-ctx.Done():
			return
		case sig, ok := <-sigs:
			if !ok {
				return
			}
			if on, ok := screenReaderOf(sig); ok {
				b.setEnabled(ctx, session, on)
			}
		}
	}
}

// readScreenReader reads org.a11y.Status.ScreenReaderEnabled.
func readScreenReader(ctx context.Context, session dbus.Bus) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	reply, err := session.Call(ctx, a11yBusDest, a11yBusPath, ifaceProperties+".Get", a11yStatusIfc, "ScreenReaderEnabled")
	if err != nil {
		return false, err
	}
	if len(reply) != 1 {
		return false, errors.New("ScreenReaderFlagInvalid", "the reply of Get does not have one value", "values", len(reply))
	}
	v := reply[0]
	if variant, ok := v.(godbus.Variant); ok {
		v = variant.Value()
	}
	on, ok := v.(bool)
	if !ok {
		return false, errors.New("ScreenReaderFlagInvalid", "ScreenReaderEnabled is not a bool", "value", v)
	}
	return on, nil
}

// screenReaderOf reads the new value of ScreenReaderEnabled from a PropertiesChanged signal.
func screenReaderOf(sig *godbus.Signal) (on, ok bool) {
	if len(sig.Body) < 2 {
		return false, false
	}
	if iface, _ := sig.Body[0].(string); iface != a11yStatusIfc {
		return false, false
	}
	changed, _ := sig.Body[1].(map[string]godbus.Variant)
	v, found := changed["ScreenReaderEnabled"]
	if !found {
		return false, false
	}
	on, ok = v.Value().(bool)
	return on, ok
}

// setEnabled turns the bridge on or off. It is on when the connection to the
// accessibility bus works; the host is told after that, so that the semantics tree
// that it makes finds the bridge ready.
func (b *Bridge) setEnabled(ctx context.Context, session dbus.Bus, on bool) {
	b.mu.Lock()
	connected := b.a11y != nil
	b.mu.Unlock()
	switch {
	case on && !connected:
		if err := b.connect(ctx, session); err != nil {
			b.opts.Logger.Debug("accessibility is off: cannot connect to the accessibility bus", "error", err)
			return
		}
		b.opts.Post(func() { b.host.ScreenReaderChanged(true) })
	case !on && connected:
		b.opts.Post(func() { b.host.ScreenReaderChanged(false) })
		b.disconnect()
	}
}

// connect opens the accessibility bus, puts the root and the nodes on it and registers
// the app with the registry.
func (b *Bridge) connect(ctx context.Context, session dbus.Bus) error {
	callCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	reply, err := session.Call(callCtx, a11yBusDest, a11yBusPath, a11yBusDest+".GetAddress")
	if err != nil {
		return err
	}
	address, ok := firstString(reply)
	if !ok || address == "" {
		return errors.New("A11yAddressInvalid", "org.a11y.Bus.GetAddress did not give an address", "reply", reply)
	}
	// The connection lives as long as ctx, that is, as the bridge (dbus.ConnectAddress ties
	// the connection to its context).
	a11y, err := b.opts.OpenA11y(ctx, address)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.a11y = a11y
	b.registryRoot = objRef{Name: registryDest, Path: rootPath}
	err = b.exportLocked(appRef)
	if err == nil {
		err = b.exportAllLocked()
	}
	b.mu.Unlock()
	if err != nil {
		b.disconnect()
		return err
	}
	// The registry asks the app's objects while it answers, so the app has to be
	// served before Embed.
	embedded, err := a11y.Call(callCtx, registryDest, rootPath, "org.a11y.atspi.Socket.Embed",
		objRef{Name: a11y.UniqueName(), Path: rootPath})
	if err != nil {
		b.disconnect()
		return err
	}
	if len(embedded) == 1 {
		if root, ok := embedded[0].(objRef); ok {
			b.mu.Lock()
			b.registryRoot = root
			b.mu.Unlock()
		} else if fields, ok := embedded[0].([]any); ok && len(fields) == 2 {
			name, _ := fields[0].(string)
			path, _ := fields[1].(godbus.ObjectPath)
			b.mu.Lock()
			b.registryRoot = objRef{Name: name, Path: path}
			b.mu.Unlock()
		}
	}
	return nil
}

func firstString(reply []any) (string, bool) {
	if len(reply) != 1 {
		return "", false
	}
	s, ok := reply[0].(string)
	return s, ok
}

// disconnect takes the objects off the bus and closes it.
func (b *Bridge) disconnect() {
	b.mu.Lock()
	a11y := b.a11y
	if a11y != nil {
		for ref := range b.exported {
			b.unexportLocked(ref)
		}
	}
	b.a11y = nil
	b.mu.Unlock()
	if a11y != nil {
		if err := a11y.Close(); err != nil {
			b.opts.Logger.Debug("closing the accessibility bus failed", "error", err)
		}
	}
}

// wantedExports are the interfaces that the object of the node is served under, with the
// receivers. Action is there for every node, with no actions for most (GetInterfaces
// tells the truth); Value has no methods, only the properties that Properties serves.
func (b *Bridge) wantedExports(ref NodeRef) map[string]any {
	o := &object{b: b, ref: ref}
	if ref.IsApp() {
		return map[string]any{ifaceAccessible: o, ifaceApplication: o, ifaceProperties: propertiesObject{o}}
	}
	want := map[string]any{ifaceAccessible: o, ifaceComponent: o, ifaceAction: o, ifaceProperties: propertiesObject{o}}
	n := b.model.Node(ref)
	if n == nil {
		return want
	}
	if hasText(n) {
		want[ifaceText] = textObject{o}
	}
	if hasEditableText(n) {
		want[ifaceEditableText] = editableObject{o}
	}
	if hasSelection(b.model, n) {
		want[ifaceSelection] = selectionObject{o}
	}
	return want
}

// exportLocked puts the object of the node (or of the application) on the bus, under the
// interfaces that it has now: a text field that becomes read-only loses EditableText.
func (b *Bridge) exportLocked(ref NodeRef) error {
	if b.a11y == nil {
		return nil
	}
	have := b.exported[ref]
	if have == nil {
		have = map[string]bool{}
		b.exported[ref] = have
	}
	path := nodePath(ref)
	want := b.wantedExports(ref)
	for iface, recv := range want {
		if have[iface] {
			// The receiver only holds the bridge and the ref, so the exported one is current.
			continue
		}
		if err := b.a11y.Export(recv, path, iface); err != nil {
			return err
		}
		have[iface] = true
	}
	for iface := range have {
		if _, ok := want[iface]; !ok {
			b.a11y.Unexport(path, iface)
			delete(have, iface)
		}
	}
	return nil
}

func (b *Bridge) unexportLocked(ref NodeRef) {
	if b.a11y == nil {
		return
	}
	path := nodePath(ref)
	for iface := range b.exported[ref] {
		b.a11y.Unexport(path, iface)
	}
	delete(b.exported, ref)
}

// exportAllLocked exports every node of the model.
func (b *Bridge) exportAllLocked() error {
	for _, view := range b.model.Views() {
		vt := b.model.views[view]
		for id := range vt.nodes {
			if err := b.exportLocked(NodeRef{view, id}); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateSemantics takes the changes of the semantics tree of a view.
func (b *Bridge) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dispatchLocked(b.model.Apply(view, update))
}

// SetWindow tells the title of the window of the view, which names its frame, and
// whether it has the focus.
func (b *Bridge) SetWindow(view ui.ViewID, title string, active bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dispatchLocked(b.model.SetWindow(view, title, active))
}

// RemoveView forgets the tree of a view that closed.
func (b *Bridge) RemoveView(view ui.ViewID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dispatchLocked(b.model.RemoveView(view))
}

// dispatchLocked puts the changes on the bus: it exports and unexports objects and
// sends the events. The bus may be off, and then the model is all that changes.
func (b *Bridge) dispatchLocked(changes []Change) {
	if b.a11y == nil {
		return
	}
	for _, c := range changes {
		switch c.Kind {
		case ChangeNodeAdded, ChangeNodeUpdated:
			if err := b.exportLocked(c.Ref); err != nil {
				b.opts.Logger.Debug("exporting an accessible object failed", "error", err)
			}
		case ChangeNodeRemoved:
			b.unexportLocked(c.Ref)
		case ChangeChildAdded:
			b.emitObject(c.Parent, "ChildrenChanged", "add", int32(c.Index), 0, godbus.MakeVariant(objRef{b.busName(), nodePath(c.Child)}))
		case ChangeChildRemoved:
			b.emitObject(c.Parent, "ChildrenChanged", "remove", int32(c.Index), 0, godbus.MakeVariant(objRef{b.busName(), nodePath(c.Child)}))
		case ChangeState:
			on := int32(0)
			if c.On {
				on = 1
			}
			b.emitObject(c.Ref, "StateChanged", c.State.Name(), on, 0, godbus.MakeVariant(int32(0)))
		case ChangeProperty:
			b.emitObject(c.Ref, "PropertyChange", c.Property, 0, 0, godbus.MakeVariant(c.Value))
		case ChangeText:
			b.emitObject(c.Ref, "TextChanged", c.Property, int32(c.Index), int32(c.Length), godbus.MakeVariant(c.Value))
		case ChangeCaret:
			b.emitObject(c.Ref, "TextCaretMoved", "", int32(c.Index), 0, godbus.MakeVariant(int32(0)))
		case ChangeTextSelection:
			b.emitObject(c.Ref, "TextSelectionChanged", "", 0, 0, godbus.MakeVariant(int32(0)))
		case ChangeValue:
			b.emitObject(c.Ref, "PropertyChange", "accessible-value", 0, 0, godbus.MakeVariant(c.Number))
		case ChangeVisibleData:
			b.emitObject(c.Ref, "VisibleDataChanged", "", 0, 0, godbus.MakeVariant(int32(0)))
		case ChangeSelection:
			b.emitObject(c.Ref, "SelectionChanged", "", 0, 0, godbus.MakeVariant(int32(0)))
		case ChangeWindow:
			member := "Deactivate"
			if c.On {
				member = "Activate"
			}
			b.emit(c.Ref, eventWindowIfc, member, "", 0, 0, godbus.MakeVariant(int32(0)))
		}
	}
}

// emitObject sends an event of the interface org.a11y.atspi.Event.Object.
func (b *Bridge) emitObject(ref NodeRef, member, detail string, d1, d2 int32, value godbus.Variant) {
	b.emit(ref, eventObjectIfc, member, detail, d1, d2, value)
}

// emit sends an AT-SPI2 event from the object: the signature is (siiva{sv}).
func (b *Bridge) emit(ref NodeRef, iface, member, detail string, d1, d2 int32, value godbus.Variant) {
	if err := b.a11y.EmitSignal(nodePath(ref), iface+"."+member, detail, d1, d2, value, map[string]godbus.Variant{}); err != nil {
		b.opts.Logger.Debug("sending an accessibility event failed", "event", member, "error", err)
	}
}
