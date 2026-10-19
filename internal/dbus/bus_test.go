package dbus_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

const privateBusConfig = `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:path=%s</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>`

// startPrivateBus starts a dbus-daemon of its own and returns its address. The
// test is skipped when the daemon is not installed.
func startPrivateBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon is not installed")
	}
	dir := t.TempDir()
	socket := filepath.Join(dir, "bus")
	config := filepath.Join(dir, "bus.conf")
	if err := os.WriteFile(config, []byte(strings.Replace(privateBusConfig, "%s", socket, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, daemon, "--config-file="+config, "--nofork", "--print-address")
	out, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait() // the daemon is killed by cancel, so Wait reports that
	})
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("reading the address of dbus-daemon: %v", err)
	}
	return strings.TrimSpace(addr)
}

func dial(t *testing.T, addr string) *godbus.Conn {
	t.Helper()
	conn, err := godbus.Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Auth(nil); err != nil {
		t.Fatal(err)
	}
	if err := conn.Hello(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() }) // a closed connection is fine at the end of a test
	return conn
}

// portalService serves the Settings portal and a portal method that answers
// through a Request object, the way xdg-desktop-portal does.
type portalService struct {
	conn *godbus.Conn
	all  map[string]map[string]godbus.Variant

	mu     sync.Mutex
	closed []godbus.ObjectPath
	// answer is whether Ask sends the Response signal.
	answer bool
}

func (p *portalService) ReadAll(namespaces []string) (map[string]map[string]godbus.Variant, *godbus.Error) {
	return p.all, nil
}

func (p *portalService) Ask(sender godbus.Sender, options map[string]godbus.Variant) (godbus.ObjectPath, *godbus.Error) {
	token, _ := options["handle_token"].Value().(string)
	path := dbus.PortalRequestPath(string(sender), token)
	if err := p.conn.Export(&requestObject{p: p, path: path}, path, "org.freedesktop.portal.Request"); err != nil {
		return "", godbus.MakeFailedError(err)
	}
	if p.answer {
		go func() {
			time.Sleep(10 * time.Millisecond)
			// The error is the test's to see: the client then times out.
			_ = p.conn.Emit(path, "org.freedesktop.portal.Request.Response", uint32(0),
				map[string]godbus.Variant{"answer": godbus.MakeVariant("yes")})
		}()
	}
	return path, nil
}

type requestObject struct {
	p    *portalService
	path godbus.ObjectPath
}

func (r *requestObject) Close() *godbus.Error {
	r.p.mu.Lock()
	defer r.p.mu.Unlock()
	r.p.closed = append(r.p.closed, r.path)
	return nil
}

func (p *portalService) closedRequests() []godbus.ObjectPath {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]godbus.ObjectPath(nil), p.closed...)
}

func startPortal(t *testing.T, addr string, all map[string]map[string]godbus.Variant, answer bool) *portalService {
	t.Helper()
	conn := dial(t, addr)
	p := &portalService{conn: conn, all: all, answer: answer}
	if err := conn.Export(p, dbus.PortalPath, "org.freedesktop.portal.Settings"); err != nil {
		t.Fatal(err)
	}
	if err := conn.Export(p, dbus.PortalPath, "org.freedesktop.portal.Test"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.RequestName(dbus.PortalDest, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConnectFailsWithoutASessionBus(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/glutter-bus")
	_, err := dbus.Connect(context.Background())
	assertRoute(t, err, "Connect.DBusConnectFailed")
}

func TestRealBusSettingsRoundTrip(t *testing.T) {
	addr := startPrivateBus(t)
	portal := startPortal(t, addr, map[string]map[string]godbus.Variant{
		"org.freedesktop.appearance":  {"color-scheme": godbus.MakeVariant(uint32(1))},
		"org.gnome.desktop.interface": {"text-scaling-factor": godbus.MakeVariant(1.5)},
	}, false)

	bus := dbus.NewBus(dial(t, addr))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := newDelivery()
	initial := dbus.WatchSettings(ctx, bus, env(nil), d.post, d.onSettings)
	if initial.PlatformBrightness != ui.BrightnessDark || initial.TextScaleFactor != 1.5 {
		t.Fatalf("initial = %+v, want dark and scale 1.5", initial)
	}

	if err := portal.conn.Emit(dbus.PortalPath, "org.freedesktop.portal.Settings.SettingChanged",
		"org.gnome.desktop.interface", "enable-animations", godbus.MakeVariant(false)); err != nil {
		t.Fatal(err)
	}
	got := d.wait(t)
	want := initial
	want.DisableAnimations = true
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after the change = %+v, want %+v", got, want)
	}
}

func TestRealBusWatchFallsBackWithoutThePortal(t *testing.T) {
	addr := startPrivateBus(t)
	bus := dbus.NewBus(dial(t, addr))
	d := newDelivery()
	got := dbus.WatchSettings(context.Background(), bus, env(map[string]string{"XCURSOR_SIZE": "36"}), d.post, d.onSettings)
	want := platform.Settings{PlatformBrightness: ui.BrightnessLight, TextScaleFactor: 1, CursorSize: 36}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
}

func TestRealBusPortalRequest(t *testing.T) {
	addr := startPrivateBus(t)
	startPortal(t, addr, nil, true)
	bus := dbus.NewBus(dial(t, addr))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := dbus.PortalRequest(ctx, bus, "org.freedesktop.portal.Test.Ask", ask)
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != dbus.PortalResponseSuccess || got.Results["answer"].Value() != "yes" {
		t.Errorf("response = %+v", got)
	}
}

func TestRealBusPortalRequestAbort(t *testing.T) {
	addr := startPrivateBus(t)
	portal := startPortal(t, addr, nil, false)
	bus := dbus.NewBus(dial(t, addr))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := dbus.PortalRequest(ctx, bus, "org.freedesktop.portal.Test.Ask", ask)
	assertRoute(t, err, "PortalRequest.PortalRequestAborted")
	closed := portal.closedRequests()
	if len(closed) != 1 || !strings.HasPrefix(string(closed[0]), "/org/freedesktop/portal/desktop/request/") {
		t.Errorf("closed requests = %v, want the one request", closed)
	}
}

func TestRealBusSubscribeFails(t *testing.T) {
	addr := startPrivateBus(t)
	bus := dbus.NewBus(dial(t, addr))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := bus.Subscribe(ctx, dbus.SignalMatch{Interface: "org.example.I", Member: "Ping"})
	assertRoute(t, err, "sessionBus.Subscribe.DBusSubscribeFailed")
}

func TestRealBusCallErrors(t *testing.T) {
	addr := startPrivateBus(t)
	bus := dbus.NewBus(dial(t, addr))
	_, err := bus.Call(context.Background(), "org.example.Missing", "/", "org.example.Missing.Method")
	assertRoute(t, err, "sessionBus.Call.DBusCallFailed")
}

func TestRealBusUniqueName(t *testing.T) {
	addr := startPrivateBus(t)
	bus := dbus.NewBus(dial(t, addr))
	if name := bus.UniqueName(); !strings.HasPrefix(name, ":") {
		t.Errorf("unique name = %q", name)
	}
}

func TestRealBusSubscribeAndClose(t *testing.T) {
	addr := startPrivateBus(t)
	bus := dbus.NewBus(dial(t, addr))
	ctx, cancel := context.WithCancel(context.Background())
	sigs, err := bus.Subscribe(ctx, dbus.SignalMatch{Interface: "org.example.I", Member: "Ping"})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case _, ok := <-sigs:
		if ok {
			t.Error("unexpected signal")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the channel was not closed after the context was done")
	}
	if err := bus.Close(); err != nil {
		t.Fatal(err)
	}
}

// failingCloser is a connection whose Close fails.
type failingCloser struct{ net.Conn }

func (failingCloser) Close() error { return io.ErrClosedPipe }

func TestRealBusCloseFails(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = server.Close() }) // the pipe is dropped at the end of the test
	conn, err := godbus.NewConn(failingCloser{client})
	if err != nil {
		t.Fatal(err)
	}
	assertRoute(t, dbus.NewBus(conn).Close(), "sessionBus.Close.DBusCloseFailed")
}

func TestSignalMatch(t *testing.T) {
	sig := &godbus.Signal{Path: "/a", Name: "org.example.I.Ping"}
	tests := []struct {
		name  string
		match dbus.SignalMatch
		want  bool
	}{
		{"empty", dbus.SignalMatch{}, true},
		{"path", dbus.SignalMatch{Path: "/a"}, true},
		{"other path", dbus.SignalMatch{Path: "/b"}, false},
		{"interface and member", dbus.SignalMatch{Interface: "org.example.I", Member: "Ping"}, true},
		{"other member", dbus.SignalMatch{Interface: "org.example.I", Member: "Pong"}, false},
		{"interface only", dbus.SignalMatch{Interface: "org.example.I"}, true},
		{"other interface", dbus.SignalMatch{Interface: "org.example.J"}, false},
		{"member only", dbus.SignalMatch{Member: "Ping"}, true},
		{"other member only", dbus.SignalMatch{Member: "Pang"}, false},
		{"sender is not compared", dbus.SignalMatch{Sender: "org.example.Svc"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.match.Matches(sig); got != tc.want {
				t.Errorf("Matches = %v, want %v", got, tc.want)
			}
		})
	}
}
