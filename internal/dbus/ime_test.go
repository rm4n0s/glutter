package dbus_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	rmerrors "github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
)

const (
	ibusCtxPath  = godbus.ObjectPath("/org/freedesktop/IBus/InputContext_1")
	fcitxCtxPath = godbus.ObjectPath("/org/freedesktop/portal/inputmethod/ctx1")
	ibusCtxIface = "org.freedesktop.IBus.InputContext"
	fcitxIface   = "org.fcitx.Fcitx.InputContext1"
)

// nameOwners makes the fake bus answer NameHasOwner for the names.
func nameOwners(b *fakebus.Bus, names ...string) {
	owned := map[string]bool{}
	for _, n := range names {
		owned[n] = true
	}
	b.Handle("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.NameHasOwner", func(c fakebus.Call) ([]any, error) {
		return []any{owned[c.Args[0].(string)]}, nil
	})
}

// ok answers a call with no values.
func ok(fakebus.Call) ([]any, error) { return nil, nil }

// ibusService makes the fake bus an IBus portal whose input context handles the
// calls with the handlers given (all other calls get an empty answer).
func ibusService(b *fakebus.Bus, dest, iface string, handlers map[string]fakebus.Handler) {
	b.Handle(dest, "/org/freedesktop/IBus", iface+".CreateInputContext", func(fakebus.Call) ([]any, error) {
		return []any{ibusCtxPath}, nil
	})
	for _, m := range []string{"SetCapabilities", "FocusIn", "FocusOut", "Reset", "SetCursorLocation", "SetSurroundingText", "Destroy"} {
		b.Handle(dest, ibusCtxPath, ibusCtxIface+"."+m, ok)
	}
	b.Handle(dest, ibusCtxPath, "org.freedesktop.DBus.Properties.Set", ok)
	for m, h := range handlers {
		b.Handle(dest, ibusCtxPath, ibusCtxIface+"."+m, h)
	}
}

func fcitxService(b *fakebus.Bus, handlers map[string]fakebus.Handler) {
	b.Handle("org.fcitx.Fcitx5", "/org/freedesktop/portal/inputmethod", "org.fcitx.Fcitx.InputMethod1.CreateInputContext",
		func(fakebus.Call) ([]any, error) { return []any{fcitxCtxPath, []byte{1, 2}}, nil })
	for _, m := range []string{"SetCapability", "FocusIn", "FocusOut", "Reset", "SetCursorRect", "SetSurroundingText", "DestroyIC"} {
		b.Handle("org.fcitx.Fcitx5", fcitxCtxPath, fcitxIface+"."+m, ok)
	}
	for m, h := range handlers {
		b.Handle("org.fcitx.Fcitx5", fcitxCtxPath, fcitxIface+"."+m, h)
	}
}

// eventLog collects the events of an input method.
type eventLog struct{ ch chan dbus.IMEEvent }

func newEventLog() *eventLog { return &eventLog{ch: make(chan dbus.IMEEvent, 16)} }

func (l *eventLog) add(e dbus.IMEEvent) { l.ch <- e }

func (l *eventLog) next(t *testing.T) dbus.IMEEvent {
	t.Helper()
	select {
	case e := <-l.ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
		return dbus.IMEEvent{}
	}
}

func (l *eventLog) none(t *testing.T) {
	t.Helper()
	select {
	case e := <-l.ch:
		t.Fatalf("unexpected event %+v", e)
	case <-time.After(50 * time.Millisecond):
	}
}

// waitSubscribers waits for the subscription of the input method to end.
func waitSubscribers(t *testing.T, b *fakebus.Bus, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for b.Subscribers() != n {
		if time.Now().After(deadline) {
			t.Fatalf("%d subscribers, want %d", b.Subscribers(), n)
		}
		time.Sleep(time.Millisecond)
	}
}

func callsTo(b *fakebus.Bus, method string) []fakebus.Call {
	var out []fakebus.Call
	for _, c := range b.Calls() {
		if c.Method == method {
			out = append(out, c)
		}
	}
	return out
}

func assertTag(t *testing.T, err error, route string) {
	t.Helper()
	appErr, isApp := rmerrors.FromError(err)
	if !isApp {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestIMEHintFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"nothing", nil, ""},
		{"xmodifiers fcitx", map[string]string{"XMODIFIERS": "@im=fcitx"}, "fcitx"},
		{"xmodifiers ibus", map[string]string{"XMODIFIERS": "@im=ibus"}, "ibus"},
		{"gtk module", map[string]string{"GTK_IM_MODULE": "fcitx5"}, "fcitx"},
		{"qt module", map[string]string{"QT_IM_MODULE": "IBus"}, "ibus"},
		{"xmodifiers wins", map[string]string{"XMODIFIERS": "@im=ibus", "GTK_IM_MODULE": "fcitx"}, "ibus"},
		{"unknown", map[string]string{"XMODIFIERS": "@im=other", "GTK_IM_MODULE": "xim"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dbus.IMEHintFromEnv(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("IMEHintFromEnv = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenInputMethodChoosesTheFrameworkThatRuns(t *testing.T) {
	tests := []struct {
		name  string
		owned []string
		hint  string
		want  string
	}{
		{"fcitx only", []string{"org.fcitx.Fcitx5"}, "", "fcitx5"},
		{"ibus portal only", []string{"org.freedesktop.portal.IBus"}, "", "ibus"},
		{"both prefer fcitx", []string{"org.fcitx.Fcitx5", "org.freedesktop.portal.IBus"}, "", "fcitx5"},
		{"both with the ibus hint", []string{"org.fcitx.Fcitx5", "org.freedesktop.portal.IBus"}, "ibus", "ibus"},
		{"ibus hint but only fcitx", []string{"org.fcitx.Fcitx5"}, "ibus", "fcitx5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := fakebus.New(":1.1")
			nameOwners(bus, tt.owned...)
			fcitxService(bus, nil)
			ibusService(bus, "org.freedesktop.portal.IBus", "org.freedesktop.IBus.Portal", nil)
			im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{Hint: tt.hint}, newEventLog().add)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
			if im.Name() != tt.want {
				t.Errorf("Name = %q, want %q", im.Name(), tt.want)
			}
		})
	}
}

func TestOpenInputMethodNeitherRuns(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus)
	_, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{}, newEventLog().add)
	assertTag(t, err, "OpenInputMethod.InputMethodUnavailable")
}

func TestOpenInputMethodCancelled(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.fcitx.Fcitx5")
	bus.Handle("org.fcitx.Fcitx5", "/org/freedesktop/portal/inputmethod", "org.fcitx.Fcitx.InputMethod1.CreateInputContext",
		func(fakebus.Call) ([]any, error) { return nil, context.Canceled })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := dbus.OpenInputMethod(ctx, bus, dbus.IMEOptions{}, newEventLog().add)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the cancellation", err)
	}
}

func TestIBusUsesThePrivateBusWhenThereIsOne(t *testing.T) {
	session := fakebus.New(":1.1")
	nameOwners(session) // the portal is not there
	direct := fakebus.New(":2.1")
	ibusService(direct, "org.freedesktop.IBus", "org.freedesktop.IBus", nil)
	opts := dbus.IMEOptions{Hint: "ibus", IBusBus: func(context.Context) (dbus.Bus, error) { return direct, nil }}
	im, err := dbus.OpenInputMethod(context.Background(), session, opts, newEventLog().add)
	if err != nil {
		t.Fatal(err)
	}
	if im.Name() != "ibus" {
		t.Errorf("Name = %q", im.Name())
	}
	if err := im.FocusIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(callsTo(direct, ibusCtxIface+".FocusIn")); n != 1 {
		t.Errorf("%d FocusIn calls on the private bus, want 1", n)
	}
	if err := im.Close(); err != nil {
		t.Fatal(err)
	}
	if err := im.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestIBusFallsBackToThePortalWhenThePrivateBusFails(t *testing.T) {
	session := fakebus.New(":1.1")
	nameOwners(session, "org.freedesktop.portal.IBus")
	ibusService(session, "org.freedesktop.portal.IBus", "org.freedesktop.IBus.Portal", nil)
	opts := dbus.IMEOptions{Hint: "ibus", IBusBus: func(context.Context) (dbus.Bus, error) {
		return nil, errors.New("no address")
	}}
	im, err := dbus.OpenInputMethod(context.Background(), session, opts, newEventLog().add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	if n := len(callsTo(session, "org.freedesktop.IBus.Portal.CreateInputContext")); n != 1 {
		t.Errorf("%d CreateInputContext calls on the portal, want 1", n)
	}
}

func TestIBusCalls(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.freedesktop.portal.IBus")
	var handled bool
	ibusService(bus, "org.freedesktop.portal.IBus", "org.freedesktop.IBus.Portal", map[string]fakebus.Handler{
		"ProcessKeyEvent": func(fakebus.Call) ([]any, error) { return []any{handled}, nil },
	})
	im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{Hint: "ibus", Program: "app"}, newEventLog().add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	ctx := context.Background()

	if got := callsTo(bus, "org.freedesktop.IBus.Portal.CreateInputContext")[0].Args; !reflect.DeepEqual(got, []any{"app"}) {
		t.Errorf("CreateInputContext args = %#v", got)
	}
	// The app draws the preedit, wants focus events and sends surrounding text.
	if got := callsTo(bus, ibusCtxIface+".SetCapabilities")[0].Args; !reflect.DeepEqual(got, []any{uint32(1 | 8 | 32)}) {
		t.Errorf("SetCapabilities args = %#v", got)
	}

	if err := im.SetCursorRect(ctx, 10, 20, 2, 16); err != nil {
		t.Fatal(err)
	}
	if got := callsTo(bus, ibusCtxIface+".SetCursorLocation")[0].Args; !reflect.DeepEqual(got, []any{int32(10), int32(20), int32(2), int32(16)}) {
		t.Errorf("SetCursorLocation args = %#v", got)
	}

	if err := im.SetSurroundingText(ctx, "héllo", 2, 2); err != nil {
		t.Fatal(err)
	}
	args := callsTo(bus, ibusCtxIface+".SetSurroundingText")[0].Args
	text := args[0].(godbus.Variant)
	if text.Signature().String() != "(sa{sv}sv)" {
		t.Errorf("surrounding text signature = %s, want (sa{sv}sv)", text.Signature())
	}
	if args[1] != uint32(2) || args[2] != uint32(2) {
		t.Errorf("cursor and anchor = %v %v", args[1], args[2])
	}

	if err := im.SetContent(ctx, dbus.IMEContent{Purpose: 5}); err != nil {
		t.Fatal(err)
	}
	if err := im.SetContent(ctx, dbus.IMEContent{Hidden: true}); err != nil {
		t.Fatal(err)
	}
	sets := callsTo(bus, "org.freedesktop.DBus.Properties.Set")
	if len(sets) != 2 {
		t.Fatalf("%d property sets", len(sets))
	}
	for i, wantPurpose := range []uint32{5, 8} {
		if sets[i].Args[0] != ibusCtxIface || sets[i].Args[1] != "ContentType" {
			t.Errorf("set %d args = %#v", i, sets[i].Args)
		}
		v := sets[i].Args[2].(godbus.Variant)
		if v.Signature().String() != "(uu)" {
			t.Errorf("ContentType signature = %s, want (uu)", v.Signature())
		}
		if f := reflect.ValueOf(v.Value()).Field(0).Uint(); uint32(f) != wantPurpose {
			t.Errorf("set %d purpose = %d, want %d", i, f, wantPurpose)
		}
	}

	for _, tt := range []struct {
		name    string
		key     dbus.IMEKey
		handled bool
		state   uint32
	}{
		{"press not consumed", dbus.IMEKey{Keysym: 'a', Keycode: 30, State: 1}, false, 1},
		{"press consumed", dbus.IMEKey{Keysym: 'a', Keycode: 30}, true, 0},
		{"release carries the release bit", dbus.IMEKey{Keysym: 'a', Keycode: 30, State: 4, Release: true}, true, 4 | 1<<30},
	} {
		handled = tt.handled
		got, err := im.ProcessKey(ctx, tt.key)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got != tt.handled {
			t.Errorf("%s: handled = %v", tt.name, got)
		}
		calls := callsTo(bus, ibusCtxIface+".ProcessKeyEvent")
		last := calls[len(calls)-1].Args
		if want := []any{tt.key.Keysym, tt.key.Keycode, tt.state}; !reflect.DeepEqual(last, want) {
			t.Errorf("%s: args = %#v, want %#v", tt.name, last, want)
		}
	}

	for name, call := range map[string]func() error{
		"FocusIn":  func() error { return im.FocusIn(ctx) },
		"FocusOut": func() error { return im.FocusOut(ctx) },
		"Reset":    func() error { return im.Reset(ctx) },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
		if n := len(callsTo(bus, ibusCtxIface+"."+name)); n != 1 {
			t.Errorf("%s was called %d times, want 1", name, n)
		}
	}
}

func TestIBusProcessKeyBadReply(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.freedesktop.portal.IBus")
	ibusService(bus, "org.freedesktop.portal.IBus", "org.freedesktop.IBus.Portal", map[string]fakebus.Handler{
		"ProcessKeyEvent": ok,
	})
	im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{Hint: "ibus"}, newEventLog().add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	_, err = im.ProcessKey(context.Background(), dbus.IMEKey{})
	assertTag(t, err, "InputMethodBadReply")
}

func TestIBusCreateInputContextBadReply(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.freedesktop.portal.IBus")
	bus.Handle("org.freedesktop.portal.IBus", "/org/freedesktop/IBus", "org.freedesktop.IBus.Portal.CreateInputContext", ok)
	_, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{Hint: "ibus"}, newEventLog().add)
	assertTag(t, err, "InputMethodUnavailable")
}

// ibusText builds the value of an IBusText as godbus decodes it from a signal.
func ibusTextValue(text string) godbus.Variant {
	return godbus.MakeVariant([]any{
		"IBusText", map[string]godbus.Variant{}, text,
		godbus.MakeVariant([]any{"IBusAttrList", map[string]godbus.Variant{}, []godbus.Variant{}}),
	})
}

func TestIBusSignalsBecomeEvents(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.freedesktop.portal.IBus")
	ibusService(bus, "org.freedesktop.portal.IBus", "org.freedesktop.IBus.Portal", nil)
	log := newEventLog()
	im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{Hint: "ibus"}, log.add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over

	emit := func(name string, body ...any) {
		bus.Emit(&godbus.Signal{Path: ibusCtxPath, Name: ibusCtxIface + "." + name, Body: body})
	}
	tests := []struct {
		name string
		emit func()
		want dbus.IMEEvent
	}{
		{"commit", func() { emit("CommitText", ibusTextValue("日本")) }, dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "日本"}},
		{"preedit cursor in characters becomes bytes", func() {
			emit("UpdatePreeditText", ibusTextValue("にほん"), uint32(2), true)
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "にほん", Cursor: 6}},
		{"preedit with mode", func() {
			emit("UpdatePreeditTextWithMode", ibusTextValue("ab"), uint32(1), true, uint32(0))
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "ab", Cursor: 1}},
		{"preedit not visible ends it", func() {
			emit("UpdatePreeditText", ibusTextValue("ab"), uint32(1), false)
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit}},
		{"hide preedit", func() { emit("HidePreeditText") }, dbus.IMEEvent{Kind: dbus.IMEEventPreedit}},
		{"delete before", func() { emit("DeleteSurroundingText", int32(-2), uint32(2)) },
			dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: -2, Count: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.emit()
			if got := log.next(t); got != tt.want {
				t.Errorf("event = %+v, want %+v", got, tt.want)
			}
		})
	}

	// Signals with no event, or with a body that is not understood, are dropped.
	emit("ForwardKeyEvent", uint32('a'), uint32(30), uint32(0))
	emit("ShowPreeditText")
	emit("CommitText", "not an IBusText")
	emit("UpdatePreeditText", ibusTextValue("x"))
	emit("DeleteSurroundingText", "x", "y")
	bus.Emit(&godbus.Signal{Path: "/other", Name: ibusCtxIface + ".CommitText", Body: []any{ibusTextValue("x")}})
	log.none(t)
}

func TestIBusCloseDestroysTheContextAndStopsTheEvents(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.freedesktop.portal.IBus")
	ibusService(bus, "org.freedesktop.portal.IBus", "org.freedesktop.IBus.Portal", nil)
	log := newEventLog()
	im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{Hint: "ibus"}, log.add)
	if err != nil {
		t.Fatal(err)
	}
	waitSubscribers(t, bus, 1)
	if err := im.Close(); err != nil {
		t.Fatal(err)
	}
	if n := len(callsTo(bus, ibusCtxIface+".Destroy")); n != 1 {
		t.Errorf("%d Destroy calls, want 1", n)
	}
	waitSubscribers(t, bus, 0)
}

func TestFcitxCalls(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.fcitx.Fcitx5")
	var handled bool
	fcitxService(bus, map[string]fakebus.Handler{
		"ProcessKeyEvent": func(fakebus.Call) ([]any, error) { return []any{handled}, nil },
	})
	im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{Program: "app", Display: "wayland"}, newEventLog().add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	ctx := context.Background()

	create := callsTo(bus, "org.fcitx.Fcitx.InputMethod1.CreateInputContext")[0].Args
	if want := []any{[]struct{ Key, Value string }(nil)}; len(create) != len(want) {
		t.Fatalf("CreateInputContext args = %#v", create)
	}
	got := reflect.ValueOf(create[0])
	if got.Len() != 2 || got.Index(0).Field(0).String() != "program" || got.Index(0).Field(1).String() != "app" ||
		got.Index(1).Field(0).String() != "display" || got.Index(1).Field(1).String() != "wayland:" {
		t.Errorf("CreateInputContext arguments = %#v", create[0])
	}

	capability := func() uint64 {
		calls := callsTo(bus, fcitxIface+".SetCapability")
		return calls[len(calls)-1].Args[0].(uint64)
	}
	const base = 1<<1 | 1<<4 | 1<<6 // preedit, formatted preedit, surrounding text
	if got := capability(); got != base {
		t.Errorf("capability = %b, want %b", got, uint64(base))
	}
	for _, tt := range []struct {
		content dbus.IMEContent
		extra   uint64
	}{
		{dbus.IMEContent{Purpose: 2}, 1 << 8},
		{dbus.IMEContent{Purpose: 3}, 1 << 14},
		{dbus.IMEContent{Purpose: 4}, 1 << 13},
		{dbus.IMEContent{Purpose: 5}, 1 << 12},
		{dbus.IMEContent{Purpose: 6}, 1 << 7},
		{dbus.IMEContent{Purpose: 8}, 1 << 3},
		{dbus.IMEContent{Purpose: 9}, 1 << 3},
		{dbus.IMEContent{Hidden: true}, 1 << 3},
		{dbus.IMEContent{Purpose: 1}, 0},
	} {
		if err := im.SetContent(ctx, tt.content); err != nil {
			t.Fatal(err)
		}
		if got := capability(); got != base|tt.extra {
			t.Errorf("capability for %+v = %b, want %b", tt.content, got, base|tt.extra)
		}
	}

	if err := im.SetCursorRect(ctx, 1, 2, 3, 4); err != nil {
		t.Fatal(err)
	}
	if got := callsTo(bus, fcitxIface+".SetCursorRect")[0].Args; !reflect.DeepEqual(got, []any{int32(1), int32(2), int32(3), int32(4)}) {
		t.Errorf("SetCursorRect args = %#v", got)
	}
	if err := im.SetSurroundingText(ctx, "abc", 1, 1); err != nil {
		t.Fatal(err)
	}
	if got := callsTo(bus, fcitxIface+".SetSurroundingText")[0].Args; !reflect.DeepEqual(got, []any{"abc", uint32(1), uint32(1)}) {
		t.Errorf("SetSurroundingText args = %#v", got)
	}

	handled = true
	ok, err := im.ProcessKey(ctx, dbus.IMEKey{Keysym: 'a', Keycode: 30, State: 4, Release: true, Time: 99})
	if err != nil || !ok {
		t.Fatalf("ProcessKey = %v, %v", ok, err)
	}
	// Fcitx5 takes the X keycode: the evdev code plus 8.
	last := callsTo(bus, fcitxIface+".ProcessKeyEvent")[0].Args
	if want := []any{uint32('a'), uint32(38), uint32(4), true, uint32(99)}; !reflect.DeepEqual(last, want) {
		t.Errorf("ProcessKeyEvent args = %#v, want %#v", last, want)
	}

	for name, call := range map[string]func() error{
		"FocusIn":  func() error { return im.FocusIn(ctx) },
		"FocusOut": func() error { return im.FocusOut(ctx) },
		"Reset":    func() error { return im.Reset(ctx) },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
		if n := len(callsTo(bus, fcitxIface+"."+name)); n != 1 {
			t.Errorf("%s was called %d times, want 1", name, n)
		}
	}
}

func TestFcitxPortalName(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.freedesktop.portal.Fcitx")
	bus.Handle("org.freedesktop.portal.Fcitx", "/org/freedesktop/portal/inputmethod", "org.fcitx.Fcitx.InputMethod1.CreateInputContext",
		func(fakebus.Call) ([]any, error) { return []any{fcitxCtxPath, []byte{}}, nil })
	bus.HandleAnyPath("org.freedesktop.portal.Fcitx", fcitxIface+".SetCapability", ok)
	bus.HandleAnyPath("org.freedesktop.portal.Fcitx", fcitxIface+".DestroyIC", ok)
	im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{}, newEventLog().add)
	if err != nil {
		t.Fatal(err)
	}
	if err := im.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFcitxBadReplyAndKeyReply(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.fcitx.Fcitx5")
	bus.Handle("org.fcitx.Fcitx5", "/org/freedesktop/portal/inputmethod", "org.fcitx.Fcitx.InputMethod1.CreateInputContext", ok)
	_, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{}, newEventLog().add)
	assertTag(t, err, "InputMethodUnavailable")

	bus2 := fakebus.New(":1.1")
	nameOwners(bus2, "org.fcitx.Fcitx5")
	fcitxService(bus2, map[string]fakebus.Handler{"ProcessKeyEvent": ok})
	im, err := dbus.OpenInputMethod(context.Background(), bus2, dbus.IMEOptions{}, newEventLog().add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	_, err = im.ProcessKey(context.Background(), dbus.IMEKey{})
	assertTag(t, err, "InputMethodBadReply")
}

func TestFcitxSignalsBecomeEvents(t *testing.T) {
	bus := fakebus.New(":1.1")
	nameOwners(bus, "org.fcitx.Fcitx5")
	fcitxService(bus, nil)
	log := newEventLog()
	im, err := dbus.OpenInputMethod(context.Background(), bus, dbus.IMEOptions{}, log.add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	emit := func(name string, body ...any) {
		bus.Emit(&godbus.Signal{Path: fcitxCtxPath, Name: fcitxIface + "." + name, Body: body})
	}
	tests := []struct {
		name string
		emit func()
		want dbus.IMEEvent
	}{
		{"commit", func() { emit("CommitString", "你好") }, dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "你好"}},
		{"preedit pieces are joined, the cursor is in bytes", func() {
			emit("UpdateFormattedPreedit", [][]any{{"ni", int32(0)}, {"hao", int32(8)}}, int32(2))
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "nihao", Cursor: 2}},
		{"preedit as generic values", func() {
			emit("UpdateFormattedPreedit", []any{[]any{"ab", int32(0)}}, int32(0))
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "ab", Cursor: 0}},
		{"no cursor", func() {
			emit("UpdateFormattedPreedit", [][]any{{"ab", int32(0)}}, int32(-1))
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "ab", Cursor: -1}},
		{"cursor past the end is no cursor", func() {
			emit("UpdateFormattedPreedit", [][]any{{"ab", int32(0)}}, int32(9))
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "ab", Cursor: -1}},
		{"empty preedit ends it", func() {
			emit("UpdateFormattedPreedit", [][]any{}, int32(0))
		}, dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Cursor: 0}},
		{"delete", func() { emit("DeleteSurroundingText", int32(-1), uint32(1)) },
			dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: -1, Count: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.emit()
			if got := log.next(t); got != tt.want {
				t.Errorf("event = %+v, want %+v", got, tt.want)
			}
		})
	}
	emit("ForwardKey", uint32('a'), uint32(0), false)
	emit("CurrentIM", "a", "b", "c")
	emit("CommitString", 42)
	emit("UpdateFormattedPreedit", "not a list", int32(0))
	emit("UpdateFormattedPreedit", [][]any{})
	emit("DeleteSurroundingText", "x", "y")
	log.none(t)
}

func TestIBusAddress(t *testing.T) {
	files := map[string]string{
		"/etc/machine-id":                        "abc123\n",
		"/cfg/ibus/bus/abc123-unix-0":            "# comment\nIBUS_ADDRESS=unix:path=/x/y,guid=g0\nIBUS_DAEMON_PID=1\n",
		"/cfg/ibus/bus/abc123-unix-wayland-1":    "IBUS_ADDRESS=unix:path=/w,guid=g1\n",
		"/cfg/ibus/bus/abc123-unix-wayland-0":    "IBUS_DAEMON_PID=2\n",
		"/cfg/ibus/bus/abc123-remote-3":          "IBUS_ADDRESS=unix:path=/r,guid=g3\n",
		"/home/u/.config/ibus/bus/abc123-unix-0": "IBUS_ADDRESS=unix:path=/h,guid=g4\n",
		"/var/lib/dbus/machine-id":               "",
		"/cfg2/ibus/bus/abc123-unix-0":           "IBUS_ADDRESS=unix:path=/c2,guid=g5\n",
	}
	read := func(name string) ([]byte, error) {
		s, found := files[name]
		if !found {
			return nil, errors.New("not found: " + name)
		}
		return []byte(s), nil
	}
	tests := []struct {
		name    string
		env     map[string]string
		display string
		want    string
		tag     string
	}{
		{"environment variable wins", map[string]string{"IBUS_ADDRESS": "unix:path=/env"}, "x11", "unix:path=/env", ""},
		{"x11 file", map[string]string{"XDG_CONFIG_HOME": "/cfg", "DISPLAY": ":0.0"}, "x11", "unix:path=/x/y,guid=g0", ""},
		{"x11 without a screen", map[string]string{"XDG_CONFIG_HOME": "/cfg", "DISPLAY": ":0"}, "x11", "unix:path=/x/y,guid=g0", ""},
		{"x11 remote host", map[string]string{"XDG_CONFIG_HOME": "/cfg", "DISPLAY": "remote:3.0"}, "x11", "unix:path=/r,guid=g3", ""},
		{"x11 default display", map[string]string{"XDG_CONFIG_HOME": "/cfg"}, "x11", "unix:path=/x/y,guid=g0", ""},
		{"wayland file", map[string]string{"XDG_CONFIG_HOME": "/cfg", "WAYLAND_DISPLAY": "wayland-1"}, "wayland", "unix:path=/w,guid=g1", ""},
		{"home config", map[string]string{"HOME": "/home/u", "DISPLAY": ":0"}, "x11", "unix:path=/h,guid=g4", ""},
		{"no address line", map[string]string{"XDG_CONFIG_HOME": "/cfg"}, "wayland", "", "IBusAddressNotFound"},
		{"no file", map[string]string{"XDG_CONFIG_HOME": "/cfg", "DISPLAY": ":9"}, "x11", "", "IBusAddressNotFound"},
		{"no home", map[string]string{}, "x11", "", "IBusAddressNotFound"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dbus.IBusAddress(func(k string) string { return tt.env[k] }, read, tt.display)
			if tt.tag != "" {
				assertTag(t, err, "IBusAddress."+tt.tag)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("address = %q, want %q", got, tt.want)
			}
		})
	}

	// Without a machine id the file cannot be named.
	delete(files, "/etc/machine-id")
	_, err := dbus.IBusAddress(func(k string) string { return map[string]string{"XDG_CONFIG_HOME": "/cfg"}[k] }, read, "x11")
	assertTag(t, err, "IBusAddress.IBusAddressNotFound")
}
