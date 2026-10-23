package dbus

import (
	"context"
	"strings"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
)

const (
	fcitxDest         = "org.fcitx.Fcitx5"
	fcitxPortalDest   = "org.freedesktop.portal.Fcitx"
	fcitxPath         = godbus.ObjectPath("/org/freedesktop/portal/inputmethod")
	fcitxMethodIface  = "org.fcitx.Fcitx.InputMethod1"
	fcitxContextIface = "org.fcitx.Fcitx.InputContext1"

	// The capability flags of Fcitx5 (CapabilityFlag in fcitx/inputcontext.h).
	fcitxCapPreedit          = 1 << 1
	fcitxCapPassword         = 1 << 3
	fcitxCapFormattedPreedit = 1 << 4
	fcitxCapSurroundingText  = 1 << 6
	fcitxCapEmail            = 1 << 7
	fcitxCapDigit            = 1 << 8
	fcitxCapUrl              = 1 << 12
	fcitxCapDialable         = 1 << 13
	fcitxCapNumber           = 1 << 14

	// evdevToX is the offset between an evdev scancode and the X keycode that
	// Fcitx5 takes.
	evdevToX = 8
)

// fcitxContext is the InputMethod of a Fcitx5 input context.
type fcitxContext struct {
	bus  Bus
	dest string
	path godbus.ObjectPath

	cancel context.CancelFunc
	done   chan struct{}

	closeOnce sync.Once
}

var _ InputMethod = (*fcitxContext)(nil)

// openFcitx creates an input context in the D-Bus frontend of Fcitx5 on the
// session bus. It fails with Fcitx5NotRunning when no service of Fcitx5 owns a
// name there.
func openFcitx(ctx context.Context, bus Bus, opts IMEOptions, events func(IMEEvent)) (InputMethod, error) {
	dest := fcitxDest
	if !nameHasOwner(ctx, bus, dest) {
		dest = fcitxPortalDest
		if !nameHasOwner(ctx, bus, dest) {
			return nil, errors.New(tagFcitx5NotRunning, "no Fcitx5 service is on the session bus")
		}
	}
	program := opts.Program
	if program == "" {
		program = "glutter"
	}
	type pair struct{ Key, Value string }
	args := []pair{{"program", program}}
	if opts.Display != "" {
		args = append(args, pair{"display", opts.Display + ":"})
	}
	body, err := bus.Call(ctx, dest, fcitxPath, fcitxMethodIface+".CreateInputContext", args)
	if err != nil {
		return nil, err
	}
	var path godbus.ObjectPath
	if len(body) >= 1 {
		path, _ = body[0].(godbus.ObjectPath)
	}
	if path == "" {
		return nil, errors.New(tagInputMethodBadReply, "Fcitx5 answered CreateInputContext without a path", "framework", "fcitx5")
	}

	c := &fcitxContext{bus: bus, dest: dest, path: path, done: make(chan struct{})}
	subCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	sigs, err := bus.Subscribe(subCtx, SignalMatch{Path: path, Interface: fcitxContextIface})
	if err != nil {
		cancel()
		_, _ = bus.Call(ctx, dest, path, fcitxContextIface+".DestroyIC") // best effort: the context is not used
		return nil, err
	}
	go c.run(sigs, events)

	if err := c.setCapability(ctx, IMEContent{}); err != nil {
		_ = c.Close() // the capabilities are needed; the error below says why
		return nil, err
	}
	return c, nil
}

func (c *fcitxContext) Name() string { return "fcitx5" }

func (c *fcitxContext) call(ctx context.Context, method string, args ...any) ([]any, error) {
	return c.bus.Call(ctx, c.dest, c.path, fcitxContextIface+"."+method, args...)
}

// setCapability sends the flags: the app draws the preedit (formatted), reads
// and writes the surrounding text, and says what the field is for.
func (c *fcitxContext) setCapability(ctx context.Context, content IMEContent) error {
	flags := uint64(fcitxCapPreedit | fcitxCapFormattedPreedit | fcitxCapSurroundingText)
	switch content.Purpose {
	case 2: // digits
		flags |= fcitxCapDigit
	case 3: // number
		flags |= fcitxCapNumber
	case 4: // phone
		flags |= fcitxCapDialable
	case 5: // URL
		flags |= fcitxCapUrl
	case 6: // email
		flags |= fcitxCapEmail
	case 8, 9: // password, PIN
		flags |= fcitxCapPassword
	}
	if content.Hidden {
		flags |= fcitxCapPassword
	}
	_, err := c.call(ctx, "SetCapability", flags)
	return err
}

func (c *fcitxContext) FocusIn(ctx context.Context) error {
	_, err := c.call(ctx, "FocusIn")
	return err
}

func (c *fcitxContext) FocusOut(ctx context.Context) error {
	_, err := c.call(ctx, "FocusOut")
	return err
}

func (c *fcitxContext) Reset(ctx context.Context) error {
	_, err := c.call(ctx, "Reset")
	return err
}

func (c *fcitxContext) SetCursorRect(ctx context.Context, x, y, w, h int) error {
	_, err := c.call(ctx, "SetCursorRect", int32(x), int32(y), int32(w), int32(h))
	return err
}

func (c *fcitxContext) SetContent(ctx context.Context, content IMEContent) error {
	return c.setCapability(ctx, content)
}

func (c *fcitxContext) SetSurroundingText(ctx context.Context, text string, cursor, anchor int) error {
	_, err := c.call(ctx, "SetSurroundingText", text, uint32(cursor), uint32(anchor))
	return err
}

func (c *fcitxContext) ProcessKey(ctx context.Context, key IMEKey) (bool, error) {
	body, err := c.call(ctx, "ProcessKeyEvent", key.Keysym, key.Keycode+evdevToX, key.State, key.Release, key.Time)
	if err != nil {
		return false, err
	}
	if len(body) != 1 {
		return false, errors.New(tagInputMethodBadReply, "Fcitx5 answered ProcessKeyEvent without a boolean", "framework", "fcitx5")
	}
	handled, _ := body[0].(bool)
	return handled, nil
}

func (c *fcitxContext) Close() error {
	var err error
	c.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), portalCloseTimeout)
		defer cancel()
		_, err = c.call(ctx, "DestroyIC")
		c.cancel()
		<-c.done
	})
	return err
}

func (c *fcitxContext) run(sigs <-chan *godbus.Signal, events func(IMEEvent)) {
	defer close(c.done)
	for sig := range sigs {
		if ev, ok := fcitxEvent(sig); ok {
			events(ev)
		}
	}
}

// fcitxEvent maps a signal of the input context to an event. ForwardKey and
// CurrentIM have no event.
func fcitxEvent(sig *godbus.Signal) (IMEEvent, bool) {
	switch strings.TrimPrefix(sig.Name, fcitxContextIface+".") {
	case "CommitString":
		text, ok := signalString(sig, 0)
		if !ok {
			return IMEEvent{}, false
		}
		return IMEEvent{Kind: IMEEventCommit, Text: text}, true
	case "UpdateFormattedPreedit":
		// (a(si) text with format flags, i cursor in bytes, -1 for none)
		if len(sig.Body) < 2 {
			return IMEEvent{}, false
		}
		var text strings.Builder
		switch parts := sig.Body[0].(type) {
		case [][]any:
			for _, p := range parts {
				if len(p) > 0 {
					s, _ := p[0].(string)
					text.WriteString(s)
				}
			}
		case []any:
			for _, p := range parts {
				if fields, ok := p.([]any); ok && len(fields) > 0 {
					s, _ := fields[0].(string)
					text.WriteString(s)
				}
			}
		default:
			return IMEEvent{}, false
		}
		cursor, _ := sig.Body[1].(int32)
		if int(cursor) > text.Len() || cursor < 0 {
			cursor = -1
		}
		return IMEEvent{Kind: IMEEventPreedit, Text: text.String(), Cursor: int(cursor)}, true
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFcitx5NotRunning    = "Fcitx5NotRunning"
	tagInputMethodBadReply = "InputMethodBadReply"
)
