package dbus

import (
	"context"
	"strings"
	"unicode/utf8"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
)

// IMEKey is a key event for an input method. The input method sees the keys
// before the app, because it may consume them to compose text.
//
// Flutter: no counterpart
type IMEKey struct {
	// Keysym is the X11 keysym of the key with the modifiers that apply.
	Keysym uint32
	// Keycode is the evdev scancode of the key (the XKB keycode minus 8).
	Keycode uint32
	// State is the X11 core modifier mask: Shift 1, Lock 2, Control 4, Mod1 8,
	// and so on.
	State uint32
	// Release is whether the key went up.
	Release bool
	// Time is the time of the event in milliseconds, or 0.
	Time uint32
}

// IMEEventKind says which change an IMEEvent is.
//
// Flutter: no counterpart
type IMEEventKind int

// The kinds of IMEEvent. The zero value is not a kind.
const (
	// IMEEventCommit inserts final text.
	//
	// Flutter: no counterpart
	IMEEventCommit IMEEventKind = iota + 1
	// IMEEventPreedit shows (or, with no text, hides) the text being composed.
	//
	// Flutter: no counterpart
	IMEEventPreedit
	// IMEEventDelete deletes text around the cursor.
	//
	// Flutter: no counterpart
	IMEEventDelete
)

// IMEEvent is one change that the input method asks for.
//
// Flutter: no counterpart
type IMEEvent struct {
	// Kind says which fields mean something.
	Kind IMEEventKind
	// Text is the text to commit, or the preedit text. An IMEEventPreedit with
	// no text ends the preedit.
	Text string
	// Cursor is the position of the cursor in the preedit, as a UTF-8 byte
	// offset into Text; -1 means the input method hides the cursor.
	Cursor int
	// Offset and Count are the text to delete, in characters (runes) from the
	// cursor: the range starts Offset characters from it (negative is
	// before) and is Count characters long.
	Offset, Count int
}

// IMEContent is what the focused text field is for.
//
// Flutter: no counterpart
type IMEContent struct {
	// Purpose is a GtkInputPurpose value (0 free-form ... 8 password, 9 PIN),
	// which agrees with ui.TextInputPurpose for 0 to 9.
	Purpose uint32
	// Hidden is whether the text is hidden (a password), whatever the purpose.
	Hidden bool
}

// InputMethod is a connection to an input method framework (IBus or Fcitx5)
// with one input context, which stands for the focused text field. The
// methods may be called from one goroutine at a time. The events arrive on a
// goroutine of the implementation, through the function given to
// OpenInputMethod.
//
// Flutter: no counterpart
type InputMethod interface {
	// Name is "ibus" or "fcitx5".
	Name() string
	// FocusIn tells the input method that a text field has the focus.
	FocusIn(ctx context.Context) error
	// FocusOut tells it that the text field lost the focus.
	FocusOut(ctx context.Context) error
	// Reset drops the text being composed.
	Reset(ctx context.Context) error
	// SetCursorRect gives the rectangle of the caret in screen pixels, so the
	// candidate window can sit next to it.
	SetCursorRect(ctx context.Context, x, y, w, h int) error
	// SetContent gives the purpose of the text field.
	SetContent(ctx context.Context, content IMEContent) error
	// SetSurroundingText gives the text around the caret and the caret and
	// the selection anchor in it, in characters (runes).
	SetSurroundingText(ctx context.Context, text string, cursor, anchor int) error
	// ProcessKey asks the input method whether it consumes the key.
	ProcessKey(ctx context.Context, key IMEKey) (handled bool, err error)
	// Close destroys the input context and stops the events.
	Close() error
}

// IMEOptions tells OpenInputMethod where to look.
//
// Flutter: no counterpart
type IMEOptions struct {
	// Hint is the framework the user chose, "ibus" or "fcitx" (see
	// IMEHintFromEnv); it is tried first. Empty tries Fcitx5, then IBus.
	Hint string
	// Program is the name the input method shows for the app, such as the
	// app id.
	Program string
	// Display is the display system, "x11" or "wayland", which Fcitx5 uses to
	// tell the contexts apart.
	Display string
	// IBusBus opens the private bus of the IBus daemon. Nil, or an error, makes
	// IBus go through its portal on the session bus.
	IBusBus func(ctx context.Context) (Bus, error)
}

// IMEHintFromEnv reads which input method the environment names, from
// XMODIFIERS ("@im=fcitx"), then GTK_IM_MODULE and QT_IM_MODULE. It returns
// "fcitx", "ibus" or "".
//
// getenv looks up an environment variable; pass os.Getenv.
//
// Flutter: no counterpart
func IMEHintFromEnv(getenv func(string) string) string {
	for _, v := range []string{
		strings.TrimPrefix(getenv("XMODIFIERS"), "@im="),
		getenv("GTK_IM_MODULE"),
		getenv("QT_IM_MODULE"),
	} {
		switch strings.ToLower(v) {
		case "fcitx", "fcitx5":
			return "fcitx"
		case "ibus":
			return "ibus"
		}
	}
	return ""
}

// OpenInputMethod connects to the input method framework that is running and
// creates an input context in it. session is the session bus. The framework
// of opts.Hint is tried first. It fails with InputMethodUnavailable when none
// is running; events then never come and the app uses plain keys with the
// compose table.
//
// events is called on a goroutine of the implementation for every change the
// input method asks for, in order. It must not block.
//
// Flutter: no counterpart
func OpenInputMethod(ctx context.Context, session Bus, opts IMEOptions, events func(IMEEvent)) (InputMethod, error) {
	order := []string{"fcitx", "ibus"}
	if opts.Hint == "ibus" {
		order = []string{"ibus", "fcitx"}
	}
	var tried []string
	for _, name := range order {
		var im InputMethod
		var err error
		switch name {
		case "fcitx":
			im, err = openFcitx(ctx, session, opts, events)
		default:
			im, err = openIBus(ctx, session, opts, events)
		}
		if err == nil {
			return im, nil
		}
		if ctx.Err() != nil {
			return nil, err
		}
		tried = append(tried, name)
	}
	return nil, errors.New("InputMethodUnavailable", "neither IBus nor Fcitx5 is running", "tried", strings.Join(tried, ","))
}

// nameHasOwner reports whether a service owns the well-known name on the bus.
func nameHasOwner(ctx context.Context, bus Bus, name string) bool {
	body, err := bus.Call(ctx, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.NameHasOwner", name)
	if err != nil || len(body) != 1 {
		return false
	}
	owned, _ := body[0].(bool)
	return owned
}

// charsToBytes returns the byte offset of the character index n in text. An
// index past the end gives len(text).
func charsToBytes(text string, n int) int {
	if n < 0 {
		return 0
	}
	off := 0
	for ; n > 0 && off < len(text); n-- {
		_, size := utf8.DecodeRuneInString(text[off:])
		off += size
	}
	return off
}

// signalString returns the string at index i of the body of a signal.
func signalString(sig *godbus.Signal, i int) (string, bool) {
	if i >= len(sig.Body) {
		return "", false
	}
	s, ok := sig.Body[i].(string)
	return s, ok
}

// ownedBus is an InputMethod whose session bus is closed with it.
type ownedBus struct {
	InputMethod
	bus Bus
}

// Close closes the input context, then the bus.
func (o ownedBus) Close() error {
	err := o.InputMethod.Close()
	if cerr := o.bus.Close(); err == nil {
		err = cerr
	}
	return err
}

// OpenSessionInputMethod connects to the session bus and calls
// OpenInputMethod on it. Closing the input method closes the connection. It
// fails with DBusConnectFailed when there is no session bus, and with
// InputMethodUnavailable when no framework is running.
//
// Flutter: no counterpart
func OpenSessionInputMethod(ctx context.Context, opts IMEOptions, events func(IMEEvent)) (InputMethod, error) {
	bus, err := Connect(ctx)
	if err != nil {
		return nil, err
	}
	im, err := OpenInputMethod(ctx, bus, opts, events)
	if err != nil {
		_ = bus.Close() // the error of opening is the one to report
		return nil, err
	}
	return ownedBus{InputMethod: im, bus: bus}, nil
}
