package atspi_test

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
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/atspi"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
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

// registry is the AT-SPI2 registry on a real bus: it takes the app's Embed.
type registry struct {
	mu     sync.Mutex
	embeds [][]any
}

func (r *registry) Embed(plug struct {
	Name string
	Path godbus.ObjectPath
}) (struct {
	Name string
	Path godbus.ObjectPath
}, *godbus.Error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.embeds = append(r.embeds, []any{plug.Name, plug.Path})
	return struct {
		Name string
		Path godbus.ObjectPath
	}{"org.a11y.atspi.Registry", rootPath}, nil
}

// TestBridgeOnARealBus checks the types that an assistive technology decodes: the
// replies of the methods, the properties and the signature of the events go through
// godbus, which the fake bus does not check.
func TestBridgeOnARealBus(t *testing.T) {
	addr := startPrivateBus(t)
	registryConn := dialBus(t, addr)
	reg := &registry{}
	if err := registryConn.Export(reg, rootPath, "org.a11y.atspi.Socket"); err != nil {
		t.Fatal(err)
	}
	if reply, err := registryConn.RequestName("org.a11y.atspi.Registry", godbus.NameFlagDoNotQueue); err != nil || reply != godbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("registry name: %v, %v", reply, err)
	}

	session := fakebus.New(":1.10")
	session.Handle("org.a11y.Bus", "/org/a11y/bus", "org.a11y.Bus.GetAddress", func(fakebus.Call) ([]any, error) { return []any{addr}, nil })
	session.Handle("org.a11y.Bus", "/org/a11y/bus", props+".Get", func(fakebus.Call) ([]any, error) {
		return []any{godbus.MakeVariant(true)}, nil
	})
	host := &testHost{}
	bridge := atspi.New(atspi.Options{
		AppName:     "Real",
		Post:        func(f func()) { f() },
		OpenSession: func(context.Context) (dbus.Bus, error) { return session, nil },
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); bridge.Close() })
	bridge.Start(ctx, host)
	eventually(t, "the bridge to turn on", func() bool { on, ok := host.lastEnabled(); return ok && on })

	reg.mu.Lock()
	embeds := len(reg.embeds)
	reg.mu.Unlock()
	if embeds != 1 {
		t.Fatalf("%d Embed calls", embeds)
	}
	bridge.SetWindow(0, "Real window", true)
	bridge.UpdateSemantics(0, sampleTree())

	// An assistive technology: its own connection, listening for events.
	client := dialBus(t, addr)
	if err := client.AddMatchSignal(godbus.WithMatchInterface("org.a11y.atspi.Event.Object")); err != nil {
		t.Fatal(err)
	}
	events := make(chan *godbus.Signal, 32)
	client.Signal(events)

	var appName string
	reg.mu.Lock()
	appName = reg.embeds[0][0].(string)
	reg.mu.Unlock()
	obj := func(path godbus.ObjectPath) godbus.BusObject { return client.Object(appName, path) }

	var children []struct {
		Name string
		Path godbus.ObjectPath
	}
	if err := obj(rootPath).Call(accessible+".GetChildren", 0).Store(&children); err != nil || len(children) != 1 {
		t.Fatalf("GetChildren = %v, %v", children, err)
	}
	if children[0].Name != appName || children[0].Path != path(0, 0) {
		t.Errorf("child = %+v", children[0])
	}
	var frameKids []struct {
		Name string
		Path godbus.ObjectPath
	}
	if err := obj(path(0, 0)).Call(accessible+".GetChildren", 0).Store(&frameKids); err != nil || len(frameKids) != 3 {
		t.Fatalf("GetChildren of the frame = %v, %v", frameKids, err)
	}
	button := obj(path(0, 1))
	var role uint32
	var states []uint32
	var roleName string
	var attrs map[string]string
	var ifaces []string
	if err := button.Call(accessible+".GetRole", 0).Store(&role); err != nil || role != uint32(atspi.RolePushButton) {
		t.Errorf("GetRole = %d, %v", role, err)
	}
	if err := button.Call(accessible+".GetRoleName", 0).Store(&roleName); err != nil || roleName != "push button" {
		t.Errorf("GetRoleName = %q, %v", roleName, err)
	}
	if err := button.Call(accessible+".GetState", 0).Store(&states); err != nil || len(states) != 2 || states[0]&(1<<atspi.StateEnabled) == 0 {
		t.Errorf("GetState = %v, %v", states, err)
	}
	if err := button.Call(accessible+".GetAttributes", 0).Store(&attrs); err != nil || attrs["id"] != "save" {
		t.Errorf("GetAttributes = %v, %v", attrs, err)
	}
	if err := button.Call(accessible+".GetInterfaces", 0).Store(&ifaces); err != nil || !reflect.DeepEqual(ifaces, []string{accessible, component, action}) {
		t.Errorf("GetInterfaces = %v, %v", ifaces, err)
	}
	var relations []struct {
		Type    uint32
		Targets []struct {
			Name string
			Path godbus.ObjectPath
		}
	}
	if err := button.Call(accessible+".GetRelationSet", 0).Store(&relations); err != nil || len(relations) != 0 {
		t.Errorf("GetRelationSet = %v, %v", relations, err)
	}
	var all map[string]godbus.Variant
	if err := button.Call(props+".GetAll", 0, accessible).Store(&all); err != nil {
		t.Fatal(err)
	}
	if all["Name"].Value() != "Save" || all["ChildCount"].Value() != int32(0) || all["HelpText"].Value() != "Ctrl+S" {
		t.Errorf("properties = %v", all)
	}
	var parent struct {
		Name string
		Path godbus.ObjectPath
	}
	if err := all["Parent"].Store(&parent); err != nil || parent.Path != path(0, 0) {
		t.Errorf("Parent = %+v, %v", parent, err)
	}
	var nActions godbus.Variant
	if err := button.Call(props+".Get", 0, action, "NActions").Store(&nActions); err != nil || nActions.Value() != int32(1) {
		t.Errorf("NActions = %v, %v", nActions, err)
	}
	var extents struct{ X, Y, W, H int32 }
	if err := button.Call(component+".GetExtents", 0, uint32(1)).Store(&extents); err != nil || extents != (struct{ X, Y, W, H int32 }{20, 40, 200, 80}) {
		t.Errorf("GetExtents = %+v, %v", extents, err)
	}
	var hit struct {
		Name string
		Path godbus.ObjectPath
	}
	if err := obj(path(0, 0)).Call(component+".GetAccessibleAtPoint", 0, int32(100), int32(60), uint32(1)).Store(&hit); err != nil || hit.Path != path(0, 1) {
		t.Errorf("GetAccessibleAtPoint = %+v, %v", hit, err)
	}
	var actions []struct{ Name, Description, KeyBinding string }
	if err := button.Call(action+".GetActions", 0).Store(&actions); err != nil || len(actions) != 1 || actions[0].Name != "click" {
		t.Errorf("GetActions = %+v, %v", actions, err)
	}
	var done bool
	if err := button.Call(action+".DoAction", 0, int32(0)).Store(&done); err != nil || !done {
		t.Errorf("DoAction = %v, %v", done, err)
	}
	if got := host.performed(); len(got) != 1 || got[0].Type != ui.SemanticsActionTap || got[0].NodeID != 1 {
		t.Errorf("performed %+v", got)
	}
	if err := button.Call(action+".DoAction", 0, int32(7)).Err; err == nil {
		t.Error("an action that is not there")
	}
	var appRole uint32
	if err := obj(rootPath).Call(accessible+".GetRole", 0).Store(&appRole); err != nil || appRole != uint32(atspi.RoleApplication) {
		t.Errorf("the application has role %d, %v", appRole, err)
	}
	var toolkit godbus.Variant
	if err := obj(rootPath).Call(props+".Get", 0, application, "ToolkitName").Store(&toolkit); err != nil || toolkit.Value() != "Glutter" {
		t.Errorf("ToolkitName = %v, %v", toolkit, err)
	}

	// An event, as the assistive technology receives it: (siiva{sv}) from the node's path.
	bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{node(1, nil, func(n *ui.SemanticsNodeUpdate) {
		n.Label = "Save"
		n.Flags.IsButton = true
		n.Flags.IsFocused = ui.TristateIsTrue
		n.Actions = ui.SemanticsActionTap | ui.SemanticsActionFocus
		n.Transform = matrix(1, 10, 20)
	})}})
	deadline := time.After(5 * time.Second)
	for focused := false; !focused; {
		select {
		case sig := <-events:
			if sig.Name == "org.a11y.atspi.Event.Object.StateChanged" && sig.Path == path(0, 1) && sig.Body[0] == "focused" {
				focused = true
				if len(sig.Body) != 5 || sig.Body[1] != int32(1) || sig.Body[2] != int32(0) {
					t.Errorf("event body = %#v", sig.Body)
				}
				if _, ok := sig.Body[3].(godbus.Variant); !ok {
					t.Errorf("the event has no variant: %#v", sig.Body[3])
				}
			}
		case <-deadline:
			t.Fatal("no state-changed:focused event")
		}
	}
}
