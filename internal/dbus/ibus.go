package dbus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
)

const (
	ibusDirectDest     = "org.freedesktop.IBus"
	ibusDirectIface    = "org.freedesktop.IBus"
	ibusPortalDest     = "org.freedesktop.portal.IBus"
	ibusPortalIface    = "org.freedesktop.IBus.Portal"
	ibusPath           = godbus.ObjectPath("/org/freedesktop/IBus")
	ibusContextIface   = "org.freedesktop.IBus.InputContext"
	ibusPropertiesSet  = "org.freedesktop.DBus.Properties.Set"
	ibusCapPreedit     = 1 << 0
	ibusCapFocus       = 1 << 3
	ibusCapSurrounding = 1 << 5
	// ibusReleaseMask is the bit of the state of a key event that says the key
	// went up.
	ibusReleaseMask = 1 << 30
)

// ibusText is an IBusText, the way IBus serializes a string: the type name, an
// empty dictionary of attachments, the text, and an IBusAttrList variant.
type ibusText struct {
	Name        string
	Attachments map[string]godbus.Variant
	Text        string
	Attrs       godbus.Variant
}

// ibusAttrList is an IBusAttrList without attributes.
type ibusAttrList struct {
	Name        string
	Attachments map[string]godbus.Variant
	Attrs       []godbus.Variant
}

func newIBusText(text string) godbus.Variant {
	return godbus.MakeVariant(ibusText{
		Name: "IBusText", Attachments: map[string]godbus.Variant{}, Text: text,
		Attrs: godbus.MakeVariant(ibusAttrList{Name: "IBusAttrList", Attachments: map[string]godbus.Variant{}, Attrs: []godbus.Variant{}}),
	})
}

// ibusTextOf reads the string of an IBusText that came in a signal. godbus
// gives the struct as a slice of its values.
func ibusTextOf(v any) (string, bool) {
	v = unwrapVariant(v)
	fields, ok := v.([]any)
	if !ok || len(fields) < 3 {
		return "", false
	}
	text, ok := fields[2].(string)
	return text, ok
}

// ibusContext is the InputMethod of an IBus input context.
type ibusContext struct {
	bus   Bus
	dest  string
	path  godbus.ObjectPath
	owned Bus // a private bus to close with the context, or nil

	cancel context.CancelFunc
	done   chan struct{}

	closeOnce sync.Once
}

var _ InputMethod = (*ibusContext)(nil)

// openIBus creates an input context in IBus. It goes to the private bus of the
// IBus daemon when opts.IBusBus gives one, and to the IBus portal on the
// session bus otherwise. It fails with IBusNotRunning when neither answers.
func openIBus(ctx context.Context, session Bus, opts IMEOptions, events func(IMEEvent)) (InputMethod, error) {
	var (
		bus          = session
		owned        Bus
		dest, iface  = ibusPortalDest, ibusPortalIface
		haveEndpoint bool
	)
	if opts.IBusBus != nil {
		if direct, err := opts.IBusBus(ctx); err == nil {
			bus, owned = direct, direct
			dest, iface = ibusDirectDest, ibusDirectIface
			haveEndpoint = true
		}
	}
	if !haveEndpoint && !nameHasOwner(ctx, session, ibusPortalDest) {
		return nil, errors.New(tagIBusNotRunning, "IBus has no private bus and no portal on the session bus")
	}
	closeOwned := func() {
		if owned != nil {
			_ = owned.Close() // nothing more to do for a bus that fails to close
		}
	}

	program := opts.Program
	if program == "" {
		program = "glutter"
	}
	body, err := bus.Call(ctx, dest, ibusPath, iface+".CreateInputContext", program)
	if err != nil {
		closeOwned()
		return nil, err
	}
	var path godbus.ObjectPath
	if len(body) == 1 {
		path, _ = body[0].(godbus.ObjectPath)
	}
	if path == "" {
		closeOwned()
		return nil, errors.New(tagInputMethodBadReply, "IBus answered CreateInputContext without a path", "framework", "ibus")
	}

	c := &ibusContext{bus: bus, dest: dest, path: path, owned: owned, done: make(chan struct{})}
	subCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	sigs, err := bus.Subscribe(subCtx, SignalMatch{Path: path, Interface: ibusContextIface})
	if err != nil {
		cancel()
		_, _ = bus.Call(ctx, dest, path, ibusContextIface+".Destroy") // best effort: the context is not used
		closeOwned()
		return nil, err
	}
	go c.run(sigs, events)

	caps := uint32(ibusCapPreedit | ibusCapFocus | ibusCapSurrounding)
	if _, err := c.call(ctx, "SetCapabilities", caps); err != nil {
		_ = c.Close() // the capabilities are needed; the error below says why
		return nil, err
	}
	return c, nil
}

func (c *ibusContext) Name() string { return "ibus" }

func (c *ibusContext) call(ctx context.Context, method string, args ...any) ([]any, error) {
	return c.bus.Call(ctx, c.dest, c.path, ibusContextIface+"."+method, args...)
}

func (c *ibusContext) FocusIn(ctx context.Context) error {
	_, err := c.call(ctx, "FocusIn")
	return err
}

func (c *ibusContext) FocusOut(ctx context.Context) error {
	_, err := c.call(ctx, "FocusOut")
	return err
}

func (c *ibusContext) Reset(ctx context.Context) error {
	_, err := c.call(ctx, "Reset")
	return err
}

func (c *ibusContext) SetCursorRect(ctx context.Context, x, y, w, h int) error {
	_, err := c.call(ctx, "SetCursorLocation", int32(x), int32(y), int32(w), int32(h))
	return err
}

// SetContent sets the ContentType property: the purpose and the hints.
func (c *ibusContext) SetContent(ctx context.Context, content IMEContent) error {
	purpose := content.Purpose
	if content.Hidden && purpose != 8 && purpose != 9 {
		purpose = 8
	}
	type contentType struct{ Purpose, Hints uint32 }
	_, err := c.bus.Call(ctx, c.dest, c.path, ibusPropertiesSet, ibusContextIface, "ContentType",
		godbus.MakeVariant(contentType{Purpose: purpose}))
	return err
}

func (c *ibusContext) SetSurroundingText(ctx context.Context, text string, cursor, anchor int) error {
	_, err := c.call(ctx, "SetSurroundingText", newIBusText(text), uint32(cursor), uint32(anchor))
	return err
}

func (c *ibusContext) ProcessKey(ctx context.Context, key IMEKey) (bool, error) {
	state := key.State
	if key.Release {
		state |= ibusReleaseMask
	}
	body, err := c.call(ctx, "ProcessKeyEvent", key.Keysym, key.Keycode, state)
	if err != nil {
		return false, err
	}
	if len(body) != 1 {
		return false, errors.New(tagInputMethodBadReply, "IBus answered ProcessKeyEvent without a boolean", "framework", "ibus")
	}
	handled, _ := body[0].(bool)
	return handled, nil
}

func (c *ibusContext) Close() error {
	var err error
	c.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), portalCloseTimeout)
		defer cancel()
		_, err = c.call(ctx, "Destroy")
		c.cancel()
		<-c.done
		if c.owned != nil {
			if cerr := c.owned.Close(); err == nil {
				err = cerr
			}
		}
	})
	return err
}

// run turns the signals of the input context into events, until the
// subscription ends.
func (c *ibusContext) run(sigs <-chan *godbus.Signal, events func(IMEEvent)) {
	defer close(c.done)
	for sig := range sigs {
		if ev, ok := ibusEvent(sig); ok {
			events(ev)
		}
	}
}

// ibusEvent maps a signal of the input context to an event. The signals that
// have no event (ForwardKeyEvent, the lookup table and the properties) are
// ignored.
func ibusEvent(sig *godbus.Signal) (IMEEvent, bool) {
	switch strings.TrimPrefix(sig.Name, ibusContextIface+".") {
	case "CommitText":
		if len(sig.Body) < 1 {
			return IMEEvent{}, false
		}
		text, ok := ibusTextOf(sig.Body[0])
		if !ok {
			return IMEEvent{}, false
		}
		return IMEEvent{Kind: IMEEventCommit, Text: text}, true
	case "UpdatePreeditText", "UpdatePreeditTextWithMode":
		// (text, cursor_pos in characters, visible[, mode])
		if len(sig.Body) < 3 {
			return IMEEvent{}, false
		}
		text, ok := ibusTextOf(sig.Body[0])
		cursor, _ := sig.Body[1].(uint32)
		visible, _ := sig.Body[2].(bool)
		if !ok {
			return IMEEvent{}, false
		}
		if !visible {
			text = ""
		}
		return IMEEvent{Kind: IMEEventPreedit, Text: text, Cursor: charsToBytes(text, int(cursor))}, true
	case "HidePreeditText":
		return IMEEvent{Kind: IMEEventPreedit}, true
	case "DeleteSurroundingText":
		if len(sig.Body) < 2 {
			return IMEEvent{}, false
		}
		offset, ok1 := sig.Body[0].(int32)
		count, ok2 := sig.Body[1].(uint32)
		if !ok1 || !ok2 {
			return IMEEvent{}, false
		}
		return IMEEvent{Kind: IMEEventDelete, Offset: int(offset), Count: int(count)}, true
	}
	return IMEEvent{}, false
}

// IBusAddress finds the address of the private bus of the IBus daemon: the
// variable IBUS_ADDRESS, or the line IBUS_ADDRESS of the file the daemon writes
// to <config>/ibus/bus/<machine-id>-<host>-<display>. display is "wayland"
// (the file is named after WAYLAND_DISPLAY, "unix-wayland-0") or "x11" (named
// after DISPLAY, "unix-0"). It fails with IBusAddressNotFound when there is
// none.
//
// getenv looks up an environment variable; readFile reads a file; pass
// os.Getenv and os.ReadFile.
//
// Flutter: no counterpart
func IBusAddress(getenv func(string) string, readFile func(string) ([]byte, error), display string) (string, error) {
	if addr := getenv("IBUS_ADDRESS"); addr != "" {
		return addr, nil
	}
	config := getenv("XDG_CONFIG_HOME")
	if config == "" {
		home := getenv("HOME")
		if home == "" {
			return "", errors.New(tagIBusAddressNotFound, "neither IBUS_ADDRESS, XDG_CONFIG_HOME nor HOME is set")
		}
		config = filepath.Join(home, ".config")
	}
	machine := ""
	for _, p := range []string{"/var/lib/dbus/machine-id", "/etc/machine-id"} {
		if b, err := readFile(p); err == nil {
			if machine = strings.TrimSpace(string(b)); machine != "" {
				break
			}
		}
	}
	if machine == "" {
		return "", errors.New(tagIBusAddressNotFound, "the machine id is unknown, so the IBus socket file cannot be named")
	}
	host, number := "unix", "0"
	if display == "wayland" {
		number = getenv("WAYLAND_DISPLAY")
		if number == "" {
			number = "wayland-0"
		}
	} else if d := getenv("DISPLAY"); d != "" {
		// ":0.0" or "host:1.0": the host is "unix" for a local display, and the
		// number is the part between ":" and ".".
		if i := strings.LastIndex(d, ":"); i >= 0 {
			if h := d[:i]; h != "" {
				host = h
			}
			d = d[i+1:]
		}
		if i := strings.Index(d, "."); i >= 0 {
			d = d[:i]
		}
		if d != "" {
			number = d
		}
	}
	file := filepath.Join(config, "ibus", "bus", machine+"-"+host+"-"+number)
	b, err := readFile(file)
	if err != nil {
		return "", errors.NewErr(tagIBusAddressNotFound, err).SetMetadata("file", file)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if addr, ok := strings.CutPrefix(strings.TrimSpace(line), "IBUS_ADDRESS="); ok && addr != "" {
			return addr, nil
		}
	}
	return "", errors.New(tagIBusAddressNotFound, "the IBus socket file has no IBUS_ADDRESS line", "file", file)
}

// IBusBusFromEnvironment returns a function that opens the private bus of the
// IBus daemon found by IBusAddress for the display system ("x11" or
// "wayland"). Use it as IMEOptions.IBusBus.
//
// Flutter: no counterpart
func IBusBusFromEnvironment(display string) func(ctx context.Context) (Bus, error) {
	return func(ctx context.Context) (Bus, error) {
		addr, err := IBusAddress(os.Getenv, os.ReadFile, display)
		if err != nil {
			return nil, err
		}
		return ConnectAddress(ctx, addr)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIBusAddressNotFound = "IBusAddressNotFound"
	tagIBusNotRunning      = "IBusNotRunning"
)
