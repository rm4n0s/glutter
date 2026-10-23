package dbus_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
)

// echo is an object a test exports.
type echo struct{}

func (echo) Echo(s string) (string, *godbus.Error) { return s + s, nil }
func (echo) Fail() *godbus.Error {
	return &godbus.Error{Name: "dev.glutter.Error.Nope", Body: []any{"no"}}
}

func tagOf(t *testing.T, err error) string {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	return appErr.Tag
}

func TestBusServesObjectsSignalsAndNames(t *testing.T) {
	addr := startPrivateBus(t)
	server, client := dbus.NewBus(dial(t, addr)), dbus.NewBus(dial(t, addr))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	const name, path, iface = "dev.glutter.Test", godbus.ObjectPath("/Echo"), "dev.glutter.Echo"

	if err := server.Export(echo{}, path, iface); err != nil {
		t.Fatal(err)
	}
	if err := server.RequestName(name); err != nil {
		t.Fatal(err)
	}
	reply, err := client.Call(ctx, name, path, iface+".Echo", "ab")
	if err != nil || !reflect.DeepEqual(reply, []any{"abab"}) {
		t.Fatalf("reply %v, err %v", reply, err)
	}
	if _, err := client.Call(ctx, name, path, iface+".Fail"); err == nil || tagOf(t, err) != "DBusCallFailed" {
		t.Errorf("a method error did not fail the call: %v", err)
	}

	// The name is owned by the server only.
	if tag := tagOf(t, client.RequestName(name)); tag != "DBusNameTaken" {
		t.Errorf("tag = %s, want DBusNameTaken", tag)
	}

	sigs, err := client.Subscribe(ctx, dbus.SignalMatch{Path: path, Interface: iface, Member: "Ping"})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.EmitSignal(path, iface+".Ping", int32(7)); err != nil {
		t.Fatal(err)
	}
	select {
	case sig := <-sigs:
		if sig.Name != iface+".Ping" || !reflect.DeepEqual(sig.Body, []any{int32(7)}) {
			t.Errorf("signal %+v", sig)
		}
	case <-ctx.Done():
		t.Fatal("no signal")
	}

	server.Unexport(path, iface)
	if _, err := client.Call(ctx, name, path, iface+".Echo", "x"); err == nil {
		t.Error("an unexported object still answers")
	}
	if err := server.ReleaseName(name); err != nil {
		t.Fatal(err)
	}
	if err := client.RequestName(name); err != nil {
		t.Errorf("a released name is not free: %v", err)
	}
}

func TestBusExportOfABadPathFails(t *testing.T) {
	bus := dbus.NewBus(dial(t, startPrivateBus(t)))
	err := bus.Export(echo{}, "not a path", "dev.glutter.Echo")
	if tag := tagOf(t, err); tag != "DBusExportFailed" {
		t.Errorf("tag = %s", tag)
	}
}

func TestPropertiesOnARealBus(t *testing.T) {
	addr := startPrivateBus(t)
	server, client := dbus.NewBus(dial(t, addr)), dbus.NewBus(dial(t, addr))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	const name, path, iface = "dev.glutter.Props", godbus.ObjectPath("/P"), "dev.glutter.P"

	p := dbus.NewProperties(server, path)
	p.Store(iface, "Count", int32(1))
	if err := p.Export(); err != nil {
		t.Fatal(err)
	}
	if err := server.RequestName(name); err != nil {
		t.Fatal(err)
	}
	reply, err := client.Call(ctx, name, path, dbus.PropertiesIface+".GetAll", iface)
	if err != nil {
		t.Fatal(err)
	}
	all, _ := reply[0].(map[string]godbus.Variant)
	if all["Count"].Value() != int32(1) {
		t.Errorf("GetAll = %v", reply)
	}
	sigs, err := client.Subscribe(ctx, dbus.SignalMatch{Path: path, Interface: dbus.PropertiesIface, Member: "PropertiesChanged"})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Update(iface, "Count", int32(2)); err != nil {
		t.Fatal(err)
	}
	select {
	case sig := <-sigs:
		changed, _ := sig.Body[1].(map[string]godbus.Variant)
		if sig.Body[0] != iface || changed["Count"].Value() != int32(2) {
			t.Errorf("signal body %v", sig.Body)
		}
	case <-ctx.Done():
		t.Fatal("no PropertiesChanged")
	}
	if _, err := client.Call(ctx, name, path, dbus.PropertiesIface+".Set", iface, "Count", godbus.MakeVariant(int32(5))); err == nil {
		t.Error("a property was set from the bus")
	}
}

func TestPropertiesWithTheFakeBus(t *testing.T) {
	const path, iface = godbus.ObjectPath("/P"), "dev.glutter.P"
	b := fakebus.New(":1.1")
	p := dbus.NewProperties(b, path)

	if !p.Store(iface, "A", "x") {
		t.Error("a new value is not a change")
	}
	if p.Store(iface, "A", "x") {
		t.Error("the same value is a change")
	}
	if err := p.Update(iface, "A", "y"); err != nil {
		t.Fatal(err)
	}
	if len(b.Signals()) != 0 {
		t.Error("a signal was sent before the object was exported")
	}
	if err := p.Export(); err != nil {
		t.Fatal(err)
	}
	if !b.Exported(path, dbus.PropertiesIface) {
		t.Fatal("not exported")
	}
	tests := []struct {
		name   string
		method string
		args   []any
		want   []any
		errIs  string
	}{
		{"Get", "Get", []any{iface, "A"}, []any{godbus.MakeVariant("y")}, ""},
		{"Get of what is not there", "Get", []any{iface, "B"}, nil, "org.freedesktop.DBus.Error.UnknownProperty"},
		{"GetAll", "GetAll", []any{iface}, []any{map[string]godbus.Variant{"A": godbus.MakeVariant("y")}}, ""},
		{"GetAll of an unknown interface", "GetAll", []any{"other"}, []any{map[string]godbus.Variant{}}, ""},
		{"Set", "Set", []any{iface, "A", godbus.MakeVariant("z")}, nil, "org.freedesktop.DBus.Error.PropertyReadOnly"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.Invoke(path, dbus.PropertiesIface, tt.method, tt.args...)
			if tt.errIs != "" {
				if e, ok := err.(*godbus.Error); !ok || e.Name != tt.errIs {
					t.Fatalf("err = %v, want %s", err, tt.errIs)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, %v; want %#v", got, err, tt.want)
			}
		})
	}

	if err := p.Update(iface, "A", "w"); err != nil {
		t.Fatal(err)
	}
	if err := p.Update(iface, "A", "w"); err != nil {
		t.Fatal(err)
	}
	sigs := b.Signals()
	if len(sigs) != 1 || sigs[0].Name != dbus.PropertiesIface+".PropertiesChanged" || sigs[0].Path != path {
		t.Fatalf("signals = %+v, want one PropertiesChanged", sigs)
	}
	if want := []any{iface, map[string]godbus.Variant{"A": godbus.MakeVariant("w")}, []string{}}; !reflect.DeepEqual(sigs[0].Body, want) {
		t.Errorf("body = %#v", sigs[0].Body)
	}
	if v, ok := p.Value(iface, "A"); !ok || v != "w" {
		t.Errorf("Value = %v, %v", v, ok)
	}
	p.Unexport()
	if b.Exported(path, dbus.PropertiesIface) {
		t.Error("still exported")
	}
}

func TestFakeBusInvoke(t *testing.T) {
	b := fakebus.New(":1.9")
	if err := b.Export(echo{}, "/E", "dev.glutter.Echo"); err != nil {
		t.Fatal(err)
	}
	got, err := b.Invoke("/E", "dev.glutter.Echo", "Echo", "ab")
	if err != nil || !reflect.DeepEqual(got, []any{"abab"}) {
		t.Errorf("got %v, %v", got, err)
	}
	if _, err := b.Invoke("/E", "dev.glutter.Echo", "Fail"); err == nil {
		t.Error("the error of the method is lost")
	}
	if _, err := b.Invoke("/E", "dev.glutter.Echo", "Nope"); err == nil {
		t.Error("an unknown method answers")
	}
	if _, err := b.Invoke("/E", "dev.glutter.Echo", "Echo", 5); err == nil {
		t.Error("a wrong argument type is accepted")
	}
	if _, err := b.Invoke("/X", "dev.glutter.Echo", "Echo", "a"); err == nil {
		t.Error("an unknown object answers")
	}
	b.Take("n")
	if tag := tagOf(t, b.RequestName("n")); tag != "DBusNameTaken" {
		t.Errorf("tag = %s", tag)
	}
	if err := b.RequestName("m"); err != nil || !reflect.DeepEqual(b.Names(), []string{"m"}) {
		t.Errorf("names = %v, err %v", b.Names(), err)
	}
}

// TestConnectOutlivesItsContext checks that the connection from Connect is not closed
// when the context given to it ends, which callers do as soon as Connect returns
// (godbus ties a connection to the context it is made with).
func TestConnectOutlivesItsContext(t *testing.T) {
	addr := startPrivateBus(t)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", addr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	bus, err := dbus.Connect(ctx)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bus.Close() })
	time.Sleep(20 * time.Millisecond)
	reply, err := bus.Call(context.Background(), "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.GetId")
	if err != nil || len(reply) != 1 {
		t.Fatalf("a call on the connection after its context ended: %v, %v", reply, err)
	}
}

func TestConnectWithACancelledContext(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", startPrivateBus(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// The connection may win the race against the context; either way no error but
	// DBusConnectFailed comes out, and a connection that arrives late is closed.
	bus, err := dbus.Connect(ctx)
	if err != nil {
		if tag := tagOf(t, err); tag != "DBusConnectFailed" {
			t.Errorf("tag = %s", tag)
		}
		// The context ended first, or the dial failed because it ended.
		appErr, _ := errors.FromError(err)
		if !appErr.HasRoute("dialWithoutContext.DBusConnectFailed") && !appErr.HasRoute("Connect.func1.DBusConnectFailed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
		return
	}
	_ = bus.Close()
}
