package desktop_test

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/dbus"
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

// startPrivateBus starts a dbus-daemon of its own and returns its address; the test is
// skipped without the daemon.
func startPrivateBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon is not installed")
	}
	dir := t.TempDir()
	socket, config := filepath.Join(dir, "bus"), filepath.Join(dir, "bus.conf")
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
		_ = cmd.Wait() // killed by cancel, so Wait reports that
	})
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("reading the address of dbus-daemon: %v", err)
	}
	return strings.TrimSpace(addr)
}

func dialBus(t *testing.T, addr string) *godbus.Conn {
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
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// watcher is a StatusNotifierWatcher on a real bus.
type watcher struct {
	mu    sync.Mutex
	items []string
}

func (w *watcher) RegisterStatusNotifierItem(sender godbus.Sender, service string) *godbus.Error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.items = append(w.items, service+" from "+string(sender))
	return nil
}

func (w *watcher) registered() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.items...)
}

// TestSystemTrayOnARealBus checks that what the tray exports is what a host decodes:
// the types of the properties, the layout and the pixmaps go through godbus.
func TestSystemTrayOnARealBus(t *testing.T) {
	addr := startPrivateBus(t)
	host := dialBus(t, addr)
	w := &watcher{}
	if err := host.Export(w, watcherPath, watcherDest); err != nil {
		t.Fatal(err)
	}
	if reply, err := host.RequestName(watcherDest, godbus.NameFlagDoNotQueue); err != nil || reply != godbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("watcher name: %v, %v", reply, err)
	}

	ts, bc := app(t)
	desktop.UseBus(bc, dbus.NewBus(dialBus(t, addr)))
	pressed := make(chan struct{}, 1)
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "mail", Images: []ui.WindowIconImage{pixels(2, 1, 2, 3, 4)}}, desktop.SystemTrayOpts{
		Tooltip: "tip",
		Menu: []desktop.TrayMenuEntry{
			desktop.NewTrayMenuItem("Open", func() { pressed <- struct{}{} }),
			desktop.NewTrayMenuSubmenu("More", []desktop.TrayMenuEntry{desktop.NewTrayMenuSeparator()}),
		},
	}))
	waitFor(t, ts, "the registration", func() bool { return len(w.registered()) == 1 })
	service := strings.Fields(w.registered()[0])[0]
	if service != trayName {
		t.Fatalf("registered %q, want %s", service, trayName)
	}

	// Read the properties as a host does: with the types of the protocol.
	obj := host.Object(service, itemPath)
	var all map[string]godbus.Variant
	if err := obj.Call(dbus.PropertiesIface+".GetAll", 0, itemIface).Store(&all); err != nil {
		t.Fatal(err)
	}
	var pixmaps []struct {
		W, H int32
		Data []byte
	}
	if err := all["IconPixmap"].Store(&pixmaps); err != nil || len(pixmaps) != 1 || !reflect.DeepEqual(pixmaps[0].Data[:4], []byte{4, 1, 2, 3}) {
		t.Errorf("IconPixmap = %+v, %v", pixmaps, err)
	}
	var tip struct {
		Name string
		Icon []struct {
			W, H int32
			Data []byte
		}
		Title, Description string
	}
	if err := all["ToolTip"].Store(&tip); err != nil || tip.Title != "tip" {
		t.Errorf("ToolTip = %+v, %v", tip, err)
	}
	if all["Menu"].Value() != menuPath || all["IconName"].Value() != "mail" || all["Category"].Value() != "ApplicationStatus" {
		t.Errorf("properties = %v", all)
	}

	// The layout.
	var layout struct {
		ID       int32
		Props    map[string]godbus.Variant
		Children []godbus.Variant
	}
	var rev uint32
	if err := host.Object(service, menuPath).Call(menuIface+".GetLayout", 0, int32(0), int32(-1), []string{}).Store(&rev, &layout); err != nil {
		t.Fatal(err)
	}
	if layout.ID != 0 || len(layout.Children) != 2 {
		t.Fatalf("layout = %+v", layout)
	}
	var first struct {
		ID       int32
		Props    map[string]godbus.Variant
		Children []godbus.Variant
	}
	if err := layout.Children[0].Store(&first); err != nil || first.Props["label"].Value() != "Open" {
		t.Errorf("first child = %+v, %v", first, err)
	}

	// A click goes to the callback on the UI thread.
	if err := host.Object(service, menuPath).Call(menuIface+".Event", 0, int32(1), "clicked", godbus.MakeVariant(""), uint32(0)).Err; err != nil {
		t.Fatal(err)
	}
	waitFor(t, ts, "the click", func() bool { return len(pressed) == 1 })

	// An update reaches the host as a signal.
	if err := host.AddMatchSignal(godbus.WithMatchInterface(itemIface), godbus.WithMatchMember("NewTitle")); err != nil {
		t.Fatal(err)
	}
	sigs := make(chan *godbus.Signal, 8)
	host.Signal(sigs)
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "mail"}, desktop.SystemTrayOpts{Title: "Changed"}))
	deadline := time.After(5 * time.Second)
	for got := false; !got; {
		ts.FlushPosted()
		select {
		case s := <-sigs:
			got = s.Name == itemIface+".NewTitle"
		case <-deadline:
			t.Fatal("no NewTitle signal")
		case <-time.After(time.Millisecond):
		}
	}
}

// realNotifyServer is a notification service on a real bus.
type realNotifyServer struct {
	mu    sync.Mutex
	calls int
}

func (n *realNotifyServer) Notify(string, uint32, string, string, string, []string, map[string]godbus.Variant, int32) (uint32, *godbus.Error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls++
	return uint32(10 + n.calls), nil
}

func (n *realNotifyServer) GetCapabilities() ([]string, *godbus.Error) { return []string{"body"}, nil }

// TestTheSessionBusStaysOpen opens the session bus the way an app does (from
// DBUS_SESSION_BUS_ADDRESS, with the default opener) and uses it for several calls:
// godbus closes a connection when the context it was made with ends, so a connection
// made with the context of the first call would be dead by the second.
func TestTheSessionBusStaysOpen(t *testing.T) {
	addr := startPrivateBus(t)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", addr)
	service := &realNotifyServer{}
	host := dialBus(t, addr)
	if err := host.Export(service, "/org/freedesktop/Notifications", "org.freedesktop.Notifications"); err != nil {
		t.Fatal(err)
	}
	if reply, err := host.RequestName("org.freedesktop.Notifications", godbus.NameFlagDoNotQueue); err != nil || reply != godbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("notification service name: %v, %v", reply, err)
	}

	ts, bc := app(t)
	for i := 1; i <= 3; i++ {
		var id desktop.NotificationID
		err := inGoroutine(t, ts, func() (err error) {
			id, err = desktop.ShowNotification(context.Background(), bc, "hello")
			return err
		})
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if id != desktop.NotificationID(10+i) {
			t.Errorf("call %d: id = %d", i, id)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestMissingServicesOnARealBus checks that the errors of a real bus for a service that is
// not there are taken as that: godbus gives them as values, not pointers. The tray is
// unavailable, the notifications are, and OpenURI falls back to xdg-open.
func TestMissingServicesOnARealBus(t *testing.T) {
	addr := startPrivateBus(t)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", addr)

	ts, bc := app(t)
	progs := &programs{}
	desktop.UseProgram(bc, progs.run)
	err := inGoroutine(t, ts, func() error {
		_, err := desktop.ShowNotification(context.Background(), bc, "hello")
		return err
	})
	assertTag(t, err, "NotificationsUnavailable")
	if err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "https://example.org") }); err != nil {
		t.Fatal(err)
	}
	if runs := progs.all(); len(runs) != 1 || runs[0].name != "xdg-open" {
		t.Errorf("runs = %+v, want xdg-open", runs)
	}
	err = inGoroutine(t, ts, func() error {
		_, err := desktop.OpenFiles(context.Background(), bc)
		return err
	})
	assertTag(t, err, "FileChooserUnavailable")

	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{}))
	var trayErr error
	waitFor(t, ts, "the tray error", func() bool { trayErr = ts.TakeError(); return trayErr != nil })
	assertTag(t, trayErr, "TrayUnavailable")
}
