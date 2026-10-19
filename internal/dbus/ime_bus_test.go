package dbus_test

import (
	"context"
	"reflect"
	"sync"
	"testing"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/internal/dbus"
)

// The tests in this file run IBus and Fcitx5 look-alikes on a private
// dbus-daemon, so the encoding of the values on the wire is the real one.

type wireText struct {
	Name        string
	Attachments map[string]godbus.Variant
	Text        string
	Attrs       godbus.Variant
}

type wireAttrs struct {
	Name        string
	Attachments map[string]godbus.Variant
	Attrs       []godbus.Variant
}

func wireIBusText(text string) godbus.Variant {
	return godbus.MakeVariant(wireText{"IBusText", map[string]godbus.Variant{}, text,
		godbus.MakeVariant(wireAttrs{"IBusAttrList", map[string]godbus.Variant{}, []godbus.Variant{}})})
}

// ibusServer is the portal, and its input context.
type ibusServer struct {
	conn *godbus.Conn

	mu      sync.Mutex
	log     []string
	caps    uint32
	text    wireText
	cursor  []uint32
	content [2]uint32
	handled bool
	last    [3]uint32
}

func (s *ibusServer) note(call string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log = append(s.log, call)
}

func (s *ibusServer) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.log...)
}

func (s *ibusServer) CreateInputContext(name string) (godbus.ObjectPath, *godbus.Error) {
	s.note("CreateInputContext " + name)
	return ibusCtxPath, nil
}

func (s *ibusServer) SetCapabilities(c uint32) *godbus.Error {
	s.mu.Lock()
	s.caps = c
	s.mu.Unlock()
	return nil
}
func (s *ibusServer) FocusIn() *godbus.Error  { s.note("FocusIn"); return nil }
func (s *ibusServer) FocusOut() *godbus.Error { s.note("FocusOut"); return nil }
func (s *ibusServer) Reset() *godbus.Error    { s.note("Reset"); return nil }
func (s *ibusServer) Destroy() *godbus.Error  { s.note("Destroy"); return nil }
func (s *ibusServer) SetCursorLocation(x, y, w, h int32) *godbus.Error {
	s.note("SetCursorLocation")
	return nil
}

func (s *ibusServer) SetSurroundingText(text godbus.Variant, cursor, anchor uint32) *godbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fields, _ := text.Value().([]any)
	if len(fields) == 4 {
		s.text = wireText{Name: fields[0].(string), Text: fields[2].(string)}
	}
	s.cursor = []uint32{cursor, anchor}
	return nil
}

func (s *ibusServer) ProcessKeyEvent(keyval, keycode, state uint32) (bool, *godbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = [3]uint32{keyval, keycode, state}
	return s.handled, nil
}

// Set is org.freedesktop.DBus.Properties.Set.
func (s *ibusServer) Set(iface, name string, v godbus.Variant) *godbus.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fields, ok := v.Value().([]any); ok && name == "ContentType" && len(fields) == 2 {
		s.content = [2]uint32{fields[0].(uint32), fields[1].(uint32)}
	}
	return nil
}

func startIBusPortal(t *testing.T, addr string) (*ibusServer, *godbus.Conn) {
	t.Helper()
	conn := dial(t, addr)
	s := &ibusServer{conn: conn}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(conn.Export(s, "/org/freedesktop/IBus", "org.freedesktop.IBus.Portal"))
	must(conn.Export(s, ibusCtxPath, ibusCtxIface))
	must(conn.Export(s, ibusCtxPath, "org.freedesktop.DBus.Properties"))
	reply, err := conn.RequestName("org.freedesktop.portal.IBus", godbus.NameFlagDoNotQueue)
	must(err)
	if reply != godbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("name reply %v", reply)
	}
	return s, conn
}

func TestIBusOnTheRealWire(t *testing.T) {
	addr := startPrivateBus(t)
	server, serverConn := startIBusPortal(t, addr)
	client := dbus.NewBus(dial(t, addr))
	t.Cleanup(func() { _ = client.Close() }) // a bus closed twice reports it; the test is over

	log := newEventLog()
	ctx := context.Background()
	im, err := dbus.OpenInputMethod(ctx, client, dbus.IMEOptions{Hint: "ibus", Program: "wire"}, log.add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	if im.Name() != "ibus" {
		t.Fatalf("Name = %s", im.Name())
	}
	server.mu.Lock()
	if server.caps != 1|8|32 {
		t.Errorf("capabilities = %d, want 41", server.caps)
	}
	server.mu.Unlock()

	if err := im.FocusIn(ctx); err != nil {
		t.Fatal(err)
	}
	if err := im.SetSurroundingText(ctx, "héllo", 2, 2); err != nil {
		t.Fatal(err)
	}
	server.mu.Lock()
	if server.text.Name != "IBusText" || server.text.Text != "héllo" || !reflect.DeepEqual(server.cursor, []uint32{2, 2}) {
		t.Errorf("surrounding text = %+v at %v", server.text, server.cursor)
	}
	server.mu.Unlock()

	if err := im.SetContent(ctx, dbus.IMEContent{Purpose: 6}); err != nil {
		t.Fatal(err)
	}
	server.mu.Lock()
	if server.content != [2]uint32{6, 0} {
		t.Errorf("content type = %v, want {6 0}", server.content)
	}
	server.mu.Unlock()

	server.mu.Lock()
	server.handled = true
	server.mu.Unlock()
	handled, err := im.ProcessKey(ctx, dbus.IMEKey{Keysym: 0x61, Keycode: 30, State: 1, Release: true})
	if err != nil || !handled {
		t.Fatalf("ProcessKey = %v, %v", handled, err)
	}
	server.mu.Lock()
	if server.last != [3]uint32{0x61, 30, 1 | 1<<30} {
		t.Errorf("key = %v", server.last)
	}
	server.mu.Unlock()

	// The signals, encoded the way IBus encodes them.
	emit := func(name string, body ...any) {
		t.Helper()
		if err := serverConn.Emit(ibusCtxPath, ibusCtxIface+"."+name, body...); err != nil {
			t.Fatal(err)
		}
	}
	emit("UpdatePreeditText", wireIBusText("にほん"), uint32(1), true, uint32(0))
	if got, want := log.next(t), (dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "にほん", Cursor: 3}); got != want {
		t.Errorf("preedit event = %+v, want %+v", got, want)
	}
	emit("CommitText", wireIBusText("日本"))
	if got, want := log.next(t), (dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "日本"}); got != want {
		t.Errorf("commit event = %+v, want %+v", got, want)
	}
	emit("HidePreeditText")
	if got, want := log.next(t), (dbus.IMEEvent{Kind: dbus.IMEEventPreedit}); got != want {
		t.Errorf("hide event = %+v, want %+v", got, want)
	}
	emit("DeleteSurroundingText", int32(-1), uint32(1))
	if got, want := log.next(t), (dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: -1, Count: 1}); got != want {
		t.Errorf("delete event = %+v, want %+v", got, want)
	}

	if err := im.Close(); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range server.calls() {
		found = found || c == "Destroy"
	}
	if !found {
		t.Errorf("calls = %v, want Destroy", server.calls())
	}
}

type fcitxArg struct{ Key, Value string }

// fcitxServer is the Fcitx5 frontend and its input context.
type fcitxServer struct {
	mu      sync.Mutex
	args    []fcitxArg
	caps    uint64
	last    []any
	handled bool
	log     []string
}

func (s *fcitxServer) CreateInputContext(args []fcitxArg) (godbus.ObjectPath, []byte, *godbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.args = args
	return fcitxCtxPath, []byte{9, 9}, nil
}

func (s *fcitxServer) SetCapability(c uint64) *godbus.Error {
	s.mu.Lock()
	s.caps = c
	s.mu.Unlock()
	return nil
}
func (s *fcitxServer) FocusIn() *godbus.Error  { return nil }
func (s *fcitxServer) FocusOut() *godbus.Error { return nil }
func (s *fcitxServer) Reset() *godbus.Error    { return nil }
func (s *fcitxServer) DestroyIC() *godbus.Error {
	s.mu.Lock()
	s.log = append(s.log, "DestroyIC")
	s.mu.Unlock()
	return nil
}
func (s *fcitxServer) SetCursorRect(x, y, w, h int32) *godbus.Error { return nil }
func (s *fcitxServer) SetSurroundingText(text string, cursor, anchor uint32) *godbus.Error {
	return nil
}

func (s *fcitxServer) ProcessKeyEvent(keyval, keycode, state uint32, release bool, time uint32) (bool, *godbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = []any{keyval, keycode, state, release, time}
	return s.handled, nil
}

func TestFcitxOnTheRealWire(t *testing.T) {
	addr := startPrivateBus(t)
	serverConn := dial(t, addr)
	server := &fcitxServer{handled: true}
	if err := serverConn.Export(server, "/org/freedesktop/portal/inputmethod", "org.fcitx.Fcitx.InputMethod1"); err != nil {
		t.Fatal(err)
	}
	if err := serverConn.Export(server, fcitxCtxPath, fcitxIface); err != nil {
		t.Fatal(err)
	}
	if _, err := serverConn.RequestName("org.fcitx.Fcitx5", godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	client := dbus.NewBus(dial(t, addr))
	t.Cleanup(func() { _ = client.Close() }) // a bus closed twice reports it; the test is over

	log := newEventLog()
	ctx := context.Background()
	im, err := dbus.OpenInputMethod(ctx, client, dbus.IMEOptions{Program: "wire", Display: "x11"}, log.add)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = im.Close() }) // closing twice is fine; the test is over
	if im.Name() != "fcitx5" {
		t.Fatalf("Name = %s", im.Name())
	}
	server.mu.Lock()
	if want := []fcitxArg{{"program", "wire"}, {"display", "x11:"}}; !reflect.DeepEqual(server.args, want) {
		t.Errorf("CreateInputContext arguments = %v, want %v", server.args, want)
	}
	if server.caps != 1<<1|1<<4|1<<6 {
		t.Errorf("capability = %b", server.caps)
	}
	server.mu.Unlock()

	handled, err := im.ProcessKey(ctx, dbus.IMEKey{Keysym: 0x61, Keycode: 30, State: 4, Time: 7})
	if err != nil || !handled {
		t.Fatalf("ProcessKey = %v, %v", handled, err)
	}
	server.mu.Lock()
	if want := []any{uint32(0x61), uint32(38), uint32(4), false, uint32(7)}; !reflect.DeepEqual(server.last, want) {
		t.Errorf("key = %v, want %v", server.last, want)
	}
	server.mu.Unlock()

	type piece struct {
		Text   string
		Format int32
	}
	emit := func(name string, body ...any) {
		t.Helper()
		if err := serverConn.Emit(fcitxCtxPath, fcitxIface+"."+name, body...); err != nil {
			t.Fatal(err)
		}
	}
	emit("UpdateFormattedPreedit", []piece{{"ni", 0}, {"hao", 8}}, int32(2))
	if got, want := log.next(t), (dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "nihao", Cursor: 2}); got != want {
		t.Errorf("preedit event = %+v, want %+v", got, want)
	}
	emit("CommitString", "你好")
	if got, want := log.next(t), (dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "你好"}); got != want {
		t.Errorf("commit event = %+v, want %+v", got, want)
	}
	emit("DeleteSurroundingText", int32(-2), uint32(2))
	if got, want := log.next(t), (dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: -2, Count: 2}); got != want {
		t.Errorf("delete event = %+v, want %+v", got, want)
	}

	if err := im.Close(); err != nil {
		t.Fatal(err)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.log) != 1 || server.log[0] != "DestroyIC" {
		t.Errorf("calls = %v, want DestroyIC", server.log)
	}
}

func TestConnectAddress(t *testing.T) {
	addr := startPrivateBus(t)
	bus, err := dbus.ConnectAddress(context.Background(), addr)
	if err != nil {
		t.Fatal(err)
	}
	if bus.UniqueName() == "" {
		t.Error("no unique name")
	}
	if err := bus.Close(); err != nil {
		t.Error(err)
	}

	_, err = dbus.ConnectAddress(context.Background(), "unix:path=/nonexistent/socket")
	assertTag(t, err, "ConnectAddress.DBusAddressDialFailed")
}
