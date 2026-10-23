package atspi_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/atspi"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/ui"
)

const (
	rootPath    = godbus.ObjectPath("/org/a11y/atspi/accessible/root")
	accessible  = "org.a11y.atspi.Accessible"
	component   = "org.a11y.atspi.Component"
	action      = "org.a11y.atspi.Action"
	application = "org.a11y.atspi.Application"
	props       = "org.freedesktop.DBus.Properties"
)

// ref is the (so) type as the tests read it from the replies.
type ref struct {
	Name string
	Path godbus.ObjectPath
}

type testHost struct {
	mu      sync.Mutex
	enabled []bool
	actions []ui.SemanticsActionEvent
}

func (h *testHost) ScreenReaderChanged(on bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = append(h.enabled, on)
}

func (h *testHost) PerformAction(ev ui.SemanticsActionEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actions = append(h.actions, ev)
}

func (h *testHost) lastEnabled() (bool, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.enabled) == 0 {
		return false, false
	}
	return h.enabled[len(h.enabled)-1], true
}

func (h *testHost) performed() []ui.SemanticsActionEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]ui.SemanticsActionEvent(nil), h.actions...)
}

// keepOpen is a fake bus that the bridge can close and open again: the tests go on
// reading what it recorded.
type keepOpen struct{ *fakebus.Bus }

func (keepOpen) Close() error { return nil }

// rig is a bridge on two fake buses: the session bus with the accessibility service, and
// the accessibility bus.
type rig struct {
	t       *testing.T
	session *fakebus.Bus
	a11y    *fakebus.Bus
	host    *testHost
	bridge  *atspi.Bridge

	mu      sync.Mutex
	address string
	embeds  []ref
	opened  int
}

func newRig(t *testing.T, screenReader bool) *rig {
	t.Helper()
	r := &rig{t: t, session: fakebus.New(":1.10"), a11y: fakebus.New(":1.20"), host: &testHost{}}
	r.session.Handle("org.a11y.Bus", "/org/a11y/bus", "org.a11y.Bus.GetAddress", func(fakebus.Call) ([]any, error) {
		return []any{"unix:path=/fake/a11y"}, nil
	})
	flag := screenReader
	r.session.Handle("org.a11y.Bus", "/org/a11y/bus", props+".Get", func(c fakebus.Call) ([]any, error) {
		if c.Args[0] != "org.a11y.Status" || c.Args[1] != "ScreenReaderEnabled" {
			t.Errorf("Get(%v)", c.Args)
		}
		return []any{godbus.MakeVariant(flag)}, nil
	})
	r.a11y.Handle("org.a11y.atspi.Registry", rootPath, "org.a11y.atspi.Socket.Embed", func(c fakebus.Call) ([]any, error) {
		fields := reflect.ValueOf(c.Args[0])
		r.mu.Lock()
		r.embeds = append(r.embeds, ref{fields.Field(0).String(), godbus.ObjectPath(fields.Field(1).String())})
		r.mu.Unlock()
		return []any{[]any{"org.a11y.atspi.Registry", rootPath}}, nil
	})
	r.bridge = atspi.New(atspi.Options{
		AppName:     "Test app",
		Post:        func(f func()) { f() },
		OpenSession: func(context.Context) (dbus.Bus, error) { return r.session, nil },
		OpenA11y: func(_ context.Context, address string) (dbus.Bus, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.address = address
			r.opened++
			return keepOpen{r.a11y}, nil
		},
		Getenv: func(k string) string {
			if k == "LANG" {
				return "el_GR.UTF-8"
			}
			return ""
		},
	})
	return r
}

func (r *rig) start() {
	ctx, cancel := context.WithCancel(context.Background())
	r.t.Cleanup(func() {
		cancel()
		r.bridge.Close()
	})
	r.bridge.Start(ctx, r.host)
}

// setScreenReader changes the flag on the session bus and tells the bridge, the way
// the accessibility service does.
func (r *rig) setScreenReader(on bool) {
	r.session.Emit(&godbus.Signal{Path: "/org/a11y/bus", Name: props + ".PropertiesChanged",
		Body: []any{"org.a11y.Status", map[string]godbus.Variant{"ScreenReaderEnabled": godbus.MakeVariant(on)}, []string{}}})
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func (r *rig) waitOn() {
	r.t.Helper()
	eventually(r.t, "the bridge to turn on", func() bool { on, ok := r.host.lastEnabled(); return ok && on })
}

// call invokes a method of the objects of the app, as an assistive technology would.
func (r *rig) call(path godbus.ObjectPath, iface, method string, args ...any) []any {
	r.t.Helper()
	got, err := r.a11y.Invoke(path, iface, method, args...)
	if err != nil {
		r.t.Fatalf("%s.%s on %s: %v", iface, method, path, err)
	}
	return got
}

// toRef reads a (so) value that the bridge returned.
func toRef(v any) ref {
	f := reflect.ValueOf(v)
	return ref{f.Field(0).String(), godbus.ObjectPath(f.Field(1).String())}
}

func toRefs(v any) []ref {
	s := reflect.ValueOf(v)
	out := make([]ref, s.Len())
	for i := range out {
		out[i] = toRef(s.Index(i).Interface())
	}
	return out
}

func (r *rig) prop(path godbus.ObjectPath, iface, name string) any {
	r.t.Helper()
	got := r.call(path, props, "Get", iface, name)
	return got[0].(godbus.Variant).Value()
}

// path is the path of the node of the view.
func path(view, id int) godbus.ObjectPath {
	return godbus.ObjectPath("/org/a11y/atspi/accessible/" + itoa(view) + "_" + itoa(id))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// matrix is a 4x4 transform that moves by (dx, dy) and scales by s.
func matrix(s, dx, dy float32) [16]float32 {
	return [16]float32{s, 0, 0, 0, 0, s, 0, 0, 0, 0, 1, 0, dx, dy, 0, 1}
}

func node(id int, children []int, edit func(n *ui.SemanticsNodeUpdate)) ui.SemanticsNodeUpdate {
	n := ui.SemanticsNodeUpdate{
		ID: id, Rect: ui.Rect{Left: 0, Top: 0, Right: 100, Bottom: 40}, Transform: matrix(1, 0, 0),
		ChildrenInTraversalOrder: children, ChildrenInHitTestOrder: children, TraversalParent: -1,
		TextSelectionBase: -1, TextSelectionExtent: -1, PlatformViewID: -1, MaxValueLength: -1, CurrentValueLength: -1,
	}
	if edit != nil {
		edit(&n)
	}
	return n
}

// sampleTree is a window with a "Save" button, a checked check box and a heading:
//
//	0 (root, scale 2)
//	├── 1 button "Save", tap, at (10, 20) size 100x40
//	├── 2 check box "Dark mode", checked, at (10, 80)
//	└── 3 heading "Settings" level 2
func sampleTree() *ui.SemanticsUpdate {
	return &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{
		node(0, []int{1, 2, 3}, func(n *ui.SemanticsNodeUpdate) {
			n.Rect = ui.Rect{Right: 400, Bottom: 300}
			n.Transform = matrix(2, 0, 0)
		}),
		node(1, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label, n.Hint, n.Tooltip = "Save", "Saves the file", "Ctrl+S"
			n.Flags.IsButton = true
			n.Flags.IsEnabled = ui.TristateIsTrue
			n.Actions = ui.SemanticsActionTap | ui.SemanticsActionFocus
			n.Transform = matrix(1, 10, 20)
			n.Identifier = "save"
		}),
		node(2, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Dark mode"
			n.Flags.IsButton = true
			n.Flags.IsChecked = ui.CheckedStateIsTrue
			n.Actions = ui.SemanticsActionTap
			n.Transform = matrix(1, 10, 80)
		}),
		node(3, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Settings"
			n.Flags.IsHeader = true
			n.HeadingLevel = 2
			n.Transform = matrix(1, 10, 140)
		}),
	}}
}

func TestBridgeStaysOffWithoutAScreenReader(t *testing.T) {
	r := newRig(t, false)
	r.start()
	eventually(t, "the flag to be read", func() bool {
		for _, c := range r.session.Calls() {
			if c.Method == props+".Get" {
				return true
			}
		}
		return false
	})
	r.bridge.SetWindow(0, "Window", true)
	r.bridge.UpdateSemantics(0, sampleTree())
	time.Sleep(20 * time.Millisecond)
	r.mu.Lock()
	opened := r.opened
	r.mu.Unlock()
	if opened != 0 || len(r.a11y.Calls()) != 0 || len(r.a11y.Signals()) != 0 {
		t.Errorf("opened %d, calls %v, signals %v", opened, r.a11y.Calls(), r.a11y.Signals())
	}
	if _, told := r.host.lastEnabled(); told {
		t.Error("the host was told about a screen reader")
	}
}

func TestBridgeConnectsAndRegisters(t *testing.T) {
	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.mu.Lock()
	address, embeds := r.address, append([]ref(nil), r.embeds...)
	r.mu.Unlock()
	if address != "unix:path=/fake/a11y" {
		t.Errorf("address = %q", address)
	}
	if want := []ref{{":1.20", rootPath}}; !reflect.DeepEqual(embeds, want) {
		t.Errorf("embeds = %v, want %v", embeds, want)
	}
	for _, iface := range []string{accessible, application, props} {
		if !r.a11y.Exported(rootPath, iface) {
			t.Errorf("the root does not serve %s", iface)
		}
	}
	if got := r.prop(rootPath, application, "ToolkitName"); got != "Glutter" {
		t.Errorf("ToolkitName = %v", got)
	}
	if got := r.prop(rootPath, accessible, "Name"); got != "Test app" {
		t.Errorf("Name = %v", got)
	}
	if got := r.prop(rootPath, accessible, "Locale"); got != "el_GR.UTF-8" {
		t.Errorf("Locale = %v", got)
	}
	// The registry's answer is the parent of the application.
	parent := reflect.ValueOf(r.prop(rootPath, accessible, "Parent"))
	if parent.Field(0).String() != "org.a11y.atspi.Registry" || godbus.ObjectPath(parent.Field(1).String()) != rootPath {
		t.Errorf("Parent = %v", parent)
	}
	if got := r.call(rootPath, accessible, "GetRoleName"); got[0] != "application" {
		t.Errorf("GetRoleName = %v", got)
	}
	if got := r.call(rootPath, accessible, "GetRole"); got[0] != uint32(atspi.RoleApplication) {
		t.Errorf("GetRole = %v", got)
	}
	// The registry sets the ID of the application.
	r.call(rootPath, props, "Set", application, "Id", godbus.MakeVariant(int32(7)))
	if got := r.prop(rootPath, application, "Id"); got != int32(7) {
		t.Errorf("Id = %v", got)
	}
	if _, err := r.a11y.Invoke(rootPath, props, "Set", accessible, "Name", godbus.MakeVariant("x")); err == nil {
		t.Error("the name was set")
	}
	if got := r.call(rootPath, accessible, "GetInterfaces"); !reflect.DeepEqual(got[0], []string{accessible, application}) {
		t.Errorf("GetInterfaces = %v", got)
	}
}

func TestBridgeMirrorsTheTree(t *testing.T) {
	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.bridge.SetWindow(0, "My window", true)
	r.bridge.UpdateSemantics(0, sampleTree())

	// The application has the frame as its only child.
	kids := toRefs(r.call(rootPath, accessible, "GetChildren")[0])
	if want := []ref{{":1.20", path(0, 0)}}; !reflect.DeepEqual(kids, want) {
		t.Fatalf("children of the application = %v", kids)
	}
	if got := r.prop(rootPath, accessible, "ChildCount"); got != int32(1) {
		t.Errorf("ChildCount = %v", got)
	}
	frame := path(0, 0)
	if got := r.call(frame, accessible, "GetRole")[0]; got != uint32(atspi.RoleFrame) {
		t.Errorf("the frame has role %v", got)
	}
	if got := r.prop(frame, accessible, "Name"); got != "My window" {
		t.Errorf("the frame is named %v", got)
	}
	states := r.call(frame, accessible, "GetState")[0].([]uint32)
	if states[0]&(1<<atspi.StateActive) == 0 {
		t.Errorf("the frame is not active: %v", states)
	}
	parent := reflect.ValueOf(r.prop(frame, accessible, "Parent"))
	if godbus.ObjectPath(parent.Field(1).String()) != rootPath {
		t.Errorf("the parent of the frame = %v", parent)
	}
	if got := r.call(frame, accessible, "GetIndexInParent")[0]; got != int32(0) {
		t.Errorf("index of the frame = %v", got)
	}
	if got := toRef(r.call(rootPath, accessible, "GetApplication")[0]); !reflect.DeepEqual(got, ref{":1.20", rootPath}) {
		t.Errorf("GetApplication = %v", got)
	}

	children := toRefs(r.call(frame, accessible, "GetChildren")[0])
	wantKids := []ref{{":1.20", path(0, 1)}, {":1.20", path(0, 2)}, {":1.20", path(0, 3)}}
	if !reflect.DeepEqual(children, wantKids) {
		t.Fatalf("children of the frame = %v", children)
	}
	if got := toRef(r.call(frame, accessible, "GetChildAtIndex", int32(1))[0]); !reflect.DeepEqual(got, wantKids[1]) {
		t.Errorf("GetChildAtIndex(1) = %v", got)
	}
	if _, err := r.a11y.Invoke(frame, accessible, "GetChildAtIndex", int32(3)); err == nil {
		t.Error("a child that is not there")
	}

	tests := []struct {
		name       string
		node       int
		role       atspi.Role
		roleName   string
		accName    string
		desc       string
		states     []atspi.State
		notStates  []atspi.State
		interfaces []string
		attrs      map[string]string
	}{
		{"button", 1, atspi.RolePushButton, "push button", "Save", "Saves the file",
			[]atspi.State{atspi.StateEnabled, atspi.StateSensitive, atspi.StateShowing, atspi.StateVisible, atspi.StateFocusable, atspi.StateHasTooltip},
			[]atspi.State{atspi.StateChecked, atspi.StateFocused},
			[]string{accessible, component, action}, map[string]string{"toolkit": "Glutter", "id": "save"}},
		{"check box", 2, atspi.RoleCheckBox, "check box", "Dark mode", "",
			[]atspi.State{atspi.StateCheckable, atspi.StateChecked, atspi.StateEnabled},
			[]atspi.State{atspi.StateFocusable},
			[]string{accessible, component, action}, map[string]string{"toolkit": "Glutter"}},
		{"heading", 3, atspi.RoleHeading, "heading", "Settings", "",
			[]atspi.State{atspi.StateShowing}, []atspi.State{atspi.StateCheckable},
			[]string{accessible, component}, map[string]string{"toolkit": "Glutter", "level": "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := path(0, tt.node)
			if got := r.call(p, accessible, "GetRole")[0]; got != uint32(tt.role) {
				t.Errorf("role = %v, want %d", got, tt.role)
			}
			if got := r.call(p, accessible, "GetRoleName")[0]; got != tt.roleName {
				t.Errorf("role name = %v", got)
			}
			if got := r.call(p, accessible, "GetLocalizedRoleName")[0]; got != tt.roleName {
				t.Errorf("localized role name = %v", got)
			}
			if got := r.prop(p, accessible, "Name"); got != tt.accName {
				t.Errorf("name = %v", got)
			}
			if got := r.prop(p, accessible, "Description"); got != tt.desc {
				t.Errorf("description = %v", got)
			}
			words := r.call(p, accessible, "GetState")[0].([]uint32)
			set := atspi.StateSet(uint64(words[0]) | uint64(words[1])<<32)
			for _, s := range tt.states {
				if !set.Has(s) {
					t.Errorf("state %s is not set", s.Name())
				}
			}
			for _, s := range tt.notStates {
				if set.Has(s) {
					t.Errorf("state %s is set", s.Name())
				}
			}
			if got := r.call(p, accessible, "GetInterfaces")[0]; !reflect.DeepEqual(got, tt.interfaces) {
				t.Errorf("interfaces = %v", got)
			}
			if got := r.call(p, accessible, "GetAttributes")[0]; !reflect.DeepEqual(got, tt.attrs) {
				t.Errorf("attributes = %v", got)
			}
			if got := r.call(p, accessible, "GetRelationSet")[0]; reflect.ValueOf(got).Len() != 0 {
				t.Errorf("relations = %v", got)
			}
			if got := r.call(p, accessible, "GetIndexInParent")[0]; got != int32(tt.node-1) {
				t.Errorf("index = %v", got)
			}
		})
	}
	if got := r.prop(path(0, 1), accessible, "HelpText"); got != "Ctrl+S" {
		t.Errorf("HelpText = %v", got)
	}
	if got := r.prop(path(0, 1), accessible, "AccessibleId"); got != "save" {
		t.Errorf("AccessibleId = %v", got)
	}
}

func TestBridgeComponent(t *testing.T) {
	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.bridge.SetWindow(0, "w", true)
	r.bridge.UpdateSemantics(0, sampleTree())
	button, box := path(0, 1), path(0, 2)

	// The button is moved by (10, 20) inside the root, which scales by 2: (20, 40) 200x80.
	ext := reflect.ValueOf(r.call(button, component, "GetExtents", uint32(1))[0])
	if ext.Field(0).Int() != 20 || ext.Field(1).Int() != 40 || ext.Field(2).Int() != 200 || ext.Field(3).Int() != 80 {
		t.Errorf("extents = %v", ext)
	}
	if got := r.call(button, component, "GetPosition", uint32(1)); got[0] != int32(20) || got[1] != int32(40) {
		t.Errorf("position = %v", got)
	}
	if got := r.call(button, component, "GetSize"); got[0] != int32(200) || got[1] != int32(80) {
		t.Errorf("size = %v", got)
	}
	if got := r.call(button, component, "Contains", int32(100), int32(60), uint32(1))[0]; got != true {
		t.Errorf("Contains inside = %v", got)
	}
	if got := r.call(button, component, "Contains", int32(10), int32(60), uint32(1))[0]; got != false {
		t.Errorf("Contains outside = %v", got)
	}
	// The deepest node at a point of the frame.
	if got := toRef(r.call(path(0, 0), component, "GetAccessibleAtPoint", int32(100), int32(60), uint32(1))[0]); !reflect.DeepEqual(got, ref{":1.20", button}) {
		t.Errorf("at the button = %v", got)
	}
	if got := toRef(r.call(path(0, 0), component, "GetAccessibleAtPoint", int32(5), int32(5), uint32(1))[0]); !reflect.DeepEqual(got, ref{":1.20", path(0, 0)}) {
		t.Errorf("at the frame = %v", got)
	}
	if got := toRef(r.call(path(0, 0), component, "GetAccessibleAtPoint", int32(5000), int32(5), uint32(1))[0]); got.Path != "/org/a11y/atspi/null" {
		t.Errorf("outside = %v", got)
	}
	if got := r.call(button, component, "GetLayer")[0]; got != uint32(3) {
		t.Errorf("layer = %v", got)
	}
	if got := r.call(button, component, "GetAlpha")[0]; got != float64(1) {
		t.Errorf("alpha = %v", got)
	}
	for _, m := range []string{"SetPosition", "SetSize"} {
		args := []any{int32(1), int32(1)}
		if m == "SetPosition" {
			args = append(args, uint32(0))
		}
		if got := r.call(button, component, m, args...)[0]; got != false {
			t.Errorf("%s = %v", m, got)
		}
	}

	// Focus: only the button can take it.
	if got := r.call(button, component, "GrabFocus")[0]; got != true {
		t.Error("the button did not take the focus")
	}
	if got := r.call(box, component, "GrabFocus")[0]; got != false {
		t.Error("the check box took the focus")
	}
	want := []ui.SemanticsActionEvent{{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionFocus}}
	if got := r.host.performed(); !reflect.DeepEqual(got, want) {
		t.Errorf("performed %v, want %v", got, want)
	}
}

func TestBridgeActions(t *testing.T) {
	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.bridge.SetWindow(0, "w", true)
	tree := sampleTree()
	tree.Nodes[1].Actions |= ui.SemanticsActionLongPress | ui.SemanticsActionCustomAction
	tree.Nodes[1].AdditionalActions = []int{5, 9}
	tree.CustomActions = []ui.SemanticsCustomActionUpdate{{ID: 5, Label: "Rename"}}
	r.bridge.UpdateSemantics(0, tree)
	button := path(0, 1)

	if got := r.prop(button, action, "NActions"); got != int32(4) {
		t.Fatalf("NActions = %v", got)
	}
	acts := reflect.ValueOf(r.call(button, action, "GetActions")[0])
	var names []string
	for i := 0; i < acts.Len(); i++ {
		names = append(names, acts.Index(i).Field(0).String())
	}
	if want := []string{"click", "longpress", "Rename", "9"}; !reflect.DeepEqual(names, want) {
		t.Errorf("actions = %v, want %v", names, want)
	}
	if got := r.call(button, action, "GetName", int32(0))[0]; got != "click" {
		t.Errorf("GetName = %v", got)
	}
	if got := r.call(button, action, "GetLocalizedName", int32(2))[0]; got != "Rename" {
		t.Errorf("GetLocalizedName = %v", got)
	}
	if got := r.call(button, action, "GetDescription", int32(0))[0]; got != "Clicks the item" {
		t.Errorf("GetDescription = %v", got)
	}
	if got := r.call(button, action, "GetKeyBinding", int32(0))[0]; got != "" {
		t.Errorf("GetKeyBinding = %v", got)
	}
	for _, index := range []int32{0, 1, 2, 3} {
		if got := r.call(button, action, "DoAction", index)[0]; got != true {
			t.Errorf("DoAction(%d) = %v", index, got)
		}
	}
	want := []ui.SemanticsActionEvent{
		{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionTap},
		{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionLongPress},
		{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionCustomAction, Arguments: 5},
		{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionCustomAction, Arguments: 9},
	}
	if got := r.host.performed(); !reflect.DeepEqual(got, want) {
		t.Errorf("performed %+v\nwant      %+v", got, want)
	}
	for _, method := range []string{"GetName", "GetDescription", "GetKeyBinding", "DoAction", "GetLocalizedName"} {
		for _, index := range []int32{-1, 4} {
			if _, err := r.a11y.Invoke(button, action, method, index); err == nil {
				t.Errorf("%s(%d) did not fail", method, index)
			}
		}
	}
	if _, err := r.a11y.Invoke(path(0, 99), action, "GetActions"); err == nil {
		t.Error("an object that is not there answers")
	}
}

func signals(a *fakebus.Bus, member string) []godbus.Signal {
	var out []godbus.Signal
	for _, s := range a.Signals() {
		if s.Name == member {
			out = append(out, s)
		}
	}
	return out
}

const (
	objectEvent = "org.a11y.atspi.Event.Object."
	windowEvent = "org.a11y.atspi.Event.Window."
)

func TestBridgeEvents(t *testing.T) {
	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.bridge.SetWindow(0, "w", false)
	r.bridge.UpdateSemantics(0, sampleTree())

	// The frame appeared: the application announces its child.
	added := signals(r.a11y, objectEvent+"ChildrenChanged")
	if len(added) != 1 || added[0].Path != rootPath || added[0].Body[0] != "add" || added[0].Body[1] != int32(0) {
		t.Fatalf("children-changed = %+v", added)
	}
	body := added[0].Body
	if len(body) != 5 || !reflect.DeepEqual(toRef(body[3].(godbus.Variant).Value()), ref{":1.20", path(0, 0)}) {
		t.Errorf("event body = %#v", body)
	}
	before := len(r.a11y.Signals())

	// The button gets the focus, is renamed and gets disabled; the check box is unchecked.
	update := &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{
		node(1, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Save as"
			n.Hint = "Saves under a new name"
			n.Flags.IsButton = true
			n.Flags.IsFocused = ui.TristateIsTrue
			n.Flags.IsEnabled = ui.TristateIsFalse
			n.Actions = ui.SemanticsActionTap | ui.SemanticsActionFocus
			n.Transform = matrix(1, 10, 20)
		}),
		node(2, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Dark mode"
			n.Flags.IsButton = true
			n.Flags.IsChecked = ui.CheckedStateIsFalse
			n.Actions = ui.SemanticsActionTap
		}),
	}}
	r.bridge.UpdateSemantics(0, update)
	type stateEvent struct {
		path  godbus.ObjectPath
		name  string
		value int32
	}
	var states []stateEvent
	for _, s := range r.a11y.Signals()[before:] {
		if s.Name == objectEvent+"StateChanged" {
			states = append(states, stateEvent{s.Path, s.Body[0].(string), s.Body[1].(int32)})
		}
	}
	wantStates := []stateEvent{
		{path(0, 1), "enabled", 0}, {path(0, 1), "focused", 1}, {path(0, 1), "has-tooltip", 0}, {path(0, 1), "sensitive", 0},
		{path(0, 2), "checked", 0},
	}
	if !reflect.DeepEqual(states, wantStates) {
		t.Errorf("state events = %v\nwant         %v", states, wantStates)
	}
	var names []string
	for _, s := range signals(r.a11y, objectEvent+"PropertyChange") {
		names = append(names, string(s.Path)+" "+s.Body[0].(string)+"="+s.Body[3].(godbus.Variant).Value().(string))
	}
	wantNames := []string{string(path(0, 1)) + " accessible-name=Save as", string(path(0, 1)) + " accessible-description=Saves under a new name"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("property events = %v", names)
	}
	if got := r.prop(path(0, 1), accessible, "Name"); got != "Save as" {
		t.Errorf("the name is still %v", got)
	}

	// A child is removed and a new one added.
	before = len(r.a11y.Signals())
	r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{
		node(0, []int{1, 3, 4}, func(n *ui.SemanticsNodeUpdate) {
			n.Rect = ui.Rect{Right: 400, Bottom: 300}
			n.Transform = matrix(2, 0, 0)
		}),
		node(4, nil, func(n *ui.SemanticsNodeUpdate) { n.Label = "New" }),
	}})
	var kids []string
	for _, s := range r.a11y.Signals()[before:] {
		if s.Name == objectEvent+"ChildrenChanged" {
			kids = append(kids, s.Body[0].(string)+" "+itoa(int(s.Body[1].(int32)))+" "+string(toRef(s.Body[3].(godbus.Variant).Value()).Path))
		}
	}
	wantKids := []string{"remove 1 " + string(path(0, 2)), "add 2 " + string(path(0, 4))}
	if !reflect.DeepEqual(kids, wantKids) {
		t.Errorf("child events = %v, want %v", kids, wantKids)
	}
	if r.a11y.Exported(path(0, 2), accessible) {
		t.Error("the removed node is still exported")
	}
	if !r.a11y.Exported(path(0, 4), accessible) {
		t.Error("the new node is not exported")
	}

	// The window gets the focus.
	before = len(r.a11y.Signals())
	r.bridge.SetWindow(0, "w", true)
	if got := signals(r.a11y, windowEvent+"Activate"); len(got) != 1 || got[0].Path != path(0, 0) {
		t.Errorf("window events = %+v", got)
	}
	var activated bool
	for _, s := range r.a11y.Signals()[before:] {
		if s.Name == objectEvent+"StateChanged" && s.Body[0] == "active" && s.Body[1] == int32(1) && s.Path == path(0, 0) {
			activated = true
		}
	}
	if !activated {
		t.Error("no state-changed:active event")
	}
	r.bridge.SetWindow(0, "w", false)
	if got := signals(r.a11y, windowEvent+"Deactivate"); len(got) != 1 {
		t.Errorf("deactivate events = %+v", got)
	}
	// A new title renames the frame.
	r.bridge.SetWindow(0, "Renamed", false)
	if got := r.prop(path(0, 0), accessible, "Name"); got != "Renamed" {
		t.Errorf("the frame is named %v", got)
	}
}

func TestBridgeRemovesAView(t *testing.T) {
	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.bridge.UpdateSemantics(0, sampleTree())
	r.bridge.UpdateSemantics(3, sampleTree())
	kids := toRefs(r.call(rootPath, accessible, "GetChildren")[0])
	if len(kids) != 2 || kids[0].Path != path(0, 0) || kids[1].Path != path(3, 0) {
		t.Fatalf("frames = %v", kids)
	}
	r.bridge.RemoveView(0)
	kids = toRefs(r.call(rootPath, accessible, "GetChildren")[0])
	if len(kids) != 1 || kids[0].Path != path(3, 0) {
		t.Errorf("frames after the removal = %v", kids)
	}
	for id := 0; id <= 3; id++ {
		if r.a11y.Exported(path(0, id), accessible) {
			t.Errorf("node %d of the removed view is still exported", id)
		}
	}
	removed := false
	for _, s := range signals(r.a11y, objectEvent+"ChildrenChanged") {
		if s.Body[0] == "remove" && s.Path == rootPath {
			removed = true
		}
	}
	if !removed {
		t.Error("no children-changed:remove for the frame")
	}
}

func TestBridgeFollowsTheScreenReader(t *testing.T) {
	r := newRig(t, false)
	r.start()
	eventually(t, "the subscription", func() bool { return r.session.Subscribers() > 0 })
	r.bridge.SetWindow(0, "w", true)
	r.bridge.UpdateSemantics(0, sampleTree()) // the tree that was made before the screen reader came

	r.setScreenReader(true)
	r.waitOn()
	// The nodes that the model had are served at once.
	if got := r.call(path(0, 1), accessible, "GetRoleName")[0]; got != "push button" {
		t.Errorf("role name = %v", got)
	}
	if len(toRefs(r.call(rootPath, accessible, "GetChildren")[0])) != 1 {
		t.Error("the frame is not a child of the application")
	}

	r.setScreenReader(false)
	eventually(t, "the bridge to turn off", func() bool { on, ok := r.host.lastEnabled(); return ok && !on })
	eventually(t, "the objects to go", func() bool { return !r.a11y.Exported(rootPath, accessible) })
	if r.a11y.Exported(path(0, 1), accessible) {
		t.Error("a node is still exported")
	}
	// An update with the screen reader off only changes the model.
	r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{node(0, nil, nil)}})

	r.setScreenReader(true)
	eventually(t, "the bridge to turn on again", func() bool {
		r.host.mu.Lock()
		defer r.host.mu.Unlock()
		return len(r.host.enabled) == 3
	})
	if !r.a11y.Exported(rootPath, accessible) || r.a11y.Exported(path(0, 1), accessible) {
		t.Error("the objects are not those of the model now")
	}
	r.mu.Lock()
	opened := r.opened
	r.mu.Unlock()
	if opened != 2 {
		t.Errorf("the accessibility bus was opened %d times, want 2", opened)
	}
}

func TestBridgeStaysOffWhenTheServiceIsMissing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(r *rig)
	}{
		{"no accessibility service on the session bus", func(r *rig) { r.session = fakebus.New(":1.10") }},
		{"no registry on the accessibility bus", func(r *rig) {
			r.a11y = fakebus.New(":1.20") // Embed is not handled
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, true)
			tt.setup(r)
			r.bridge = atspi.New(atspi.Options{
				Post:        func(f func()) { f() },
				OpenSession: func(context.Context) (dbus.Bus, error) { return r.session, nil },
				OpenA11y:    func(context.Context, string) (dbus.Bus, error) { return keepOpen{r.a11y}, nil },
			})
			if tt.name != "no accessibility service on the session bus" {
				r.session.Handle("org.a11y.Bus", "/org/a11y/bus", props+".Get", func(fakebus.Call) ([]any, error) { return []any{godbus.MakeVariant(true)}, nil })
				r.session.Handle("org.a11y.Bus", "/org/a11y/bus", "org.a11y.Bus.GetAddress", func(fakebus.Call) ([]any, error) { return []any{"unix:path=/x"}, nil })
			}
			r.start()
			time.Sleep(50 * time.Millisecond)
			if _, told := r.host.lastEnabled(); told {
				t.Error("the host was told about a screen reader")
			}
			// The app keeps working.
			r.bridge.SetWindow(0, "w", true)
			r.bridge.UpdateSemantics(0, sampleTree())
		})
	}
}

func TestBridgeStaysOffWithoutASessionBus(t *testing.T) {
	r := newRig(t, true)
	r.bridge = atspi.New(atspi.Options{
		Post:        func(f func()) { f() },
		OpenSession: func(context.Context) (dbus.Bus, error) { return nil, errors.New(tagDBusConnectFailed, "no bus") },
	})
	r.start()
	time.Sleep(20 * time.Millisecond)
	r.bridge.UpdateSemantics(0, sampleTree())
	if _, told := r.host.lastEnabled(); told {
		t.Error("the host was told about a screen reader")
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDBusConnectFailed = "DBusConnectFailed"
)
