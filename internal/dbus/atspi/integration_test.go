package atspi_test

import (
	"reflect"
	"testing"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/dbus/atspi"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// findByName walks the tree below the path and returns the path of the first object
// that has the name.
func (r *rig) findByName(from godbus.ObjectPath, name string) (godbus.ObjectPath, bool) {
	r.t.Helper()
	if r.prop(from, accessible, "Name") == name {
		return from, true
	}
	for _, kid := range toRefs(r.call(from, accessible, "GetChildren")[0]) {
		if p, ok := r.findByName(kid.Path, name); ok {
			return p, true
		}
	}
	return "", false
}

// TestBridgeWithTheRealFramework puts the semantics tree that real widgets make through the
// bridge, as the engine does: the tree has the shape, the labels and the actions that the
// bridge maps, and an action that a screen reader asks for reaches the widget.
func TestBridgeWithTheRealFramework(t *testing.T) {
	ts := gluttertest.New(t)
	ts.EnsureSemantics()
	var taps int
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
		widgets.NewSemantics(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(40.0)}),
			widgets.SemanticsOpts{Container: true, Label: foundation.Some("Add"), Button: foundation.Some(true), Enabled: foundation.Some(true),
				OnTap: func() { taps++ }}),
		widgets.NewSemantics(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(40.0)}),
			widgets.SemanticsOpts{Container: true, Label: foundation.Some("Title"), Header: foundation.Some(true)}),
	})))
	ts.Pump()
	updates := ts.SemanticsUpdates()
	if len(updates) == 0 {
		t.Fatal("the framework sent no semantics update")
	}

	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.bridge.SetWindow(0, "App", true)
	for _, u := range updates {
		r.bridge.UpdateSemantics(0, u)
	}

	frame := path(0, 0)
	if got := r.call(frame, accessible, "GetRole")[0]; got != uint32(atspi.RoleFrame) {
		t.Fatalf("the root has role %v", got)
	}
	button, ok := r.findByName(frame, "Add")
	if !ok {
		t.Fatal("the button is not in the tree")
	}
	if got := r.call(button, accessible, "GetRole")[0]; got != uint32(atspi.RolePushButton) {
		t.Errorf("the button has role %v", got)
	}
	heading, ok := r.findByName(frame, "Title")
	if !ok {
		t.Fatal("the heading is not in the tree")
	}
	if got := r.call(heading, accessible, "GetRole")[0]; got != uint32(atspi.RoleHeading) {
		t.Errorf("the heading has role %v", got)
	}
	ext := reflect.ValueOf(r.call(button, component, "GetExtents", uint32(1))[0])
	if ext.Field(2).Int() != 100 || ext.Field(3).Int() != 40 {
		t.Errorf("the button is %dx%d, want 100x40", ext.Field(2).Int(), ext.Field(3).Int())
	}
	var click int32 = -1
	acts := reflect.ValueOf(r.call(button, action, "GetActions")[0])
	for i := 0; i < acts.Len(); i++ {
		if acts.Index(i).Field(0).String() == "click" {
			click = int32(i)
		}
	}
	if click < 0 {
		t.Fatal("the button has no click action")
	}

	// The screen reader presses the button: the bridge hands the action to the framework.
	r.call(button, action, "DoAction", click)
	got := r.host.performed()
	if len(got) != 1 || got[0].Type != ui.SemanticsActionTap {
		t.Fatalf("performed %+v", got)
	}
	ts.Dispatcher().DispatchSemanticsActionEvent(got[0])
	ts.Pump()
	if taps != 1 {
		t.Errorf("the button was tapped %d times, want 1", taps)
	}
}
