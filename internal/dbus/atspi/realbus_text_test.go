package atspi_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/atspi"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/ui"
)

// TestBridgeTextOnARealBus checks the types of Text, EditableText, Value and Selection on a
// real bus, and the events of text and value, as an assistive technology decodes them.
func TestBridgeTextOnARealBus(t *testing.T) {
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
	session.Handle("org.a11y.Bus", "/org/a11y/bus", props+".Get", func(fakebus.Call) ([]any, error) { return []any{godbus.MakeVariant(true)}, nil })
	host := &testHost{}
	bridge := atspi.New(atspi.Options{Post: func(f func()) { f() },
		OpenSession: func(context.Context) (dbus.Bus, error) { return session, nil }})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); bridge.Close() })
	bridge.Start(ctx, host)
	eventually(t, "the bridge to turn on", func() bool { on, ok := host.lastEnabled(); return ok && on })
	bridge.SetWindow(0, "w", true)
	bridge.UpdateSemantics(0, textTree())

	client := dialBus(t, addr)
	if err := client.AddMatchSignal(godbus.WithMatchInterface("org.a11y.atspi.Event.Object")); err != nil {
		t.Fatal(err)
	}
	events := make(chan *godbus.Signal, 64)
	client.Signal(events)
	reg.mu.Lock()
	appName := reg.embeds[0][0].(string)
	reg.mu.Unlock()
	obj := func(p godbus.ObjectPath) godbus.BusObject { return client.Object(appName, p) }

	field := obj(path(0, 1))
	var text string
	if err := field.Call(textIface+".GetText", 0, int32(1), int32(3)).Store(&text); err != nil || text != "é😀" {
		t.Errorf("GetText = %q, %v", text, err)
	}
	var piece string
	var start, end int32
	if err := field.Call(textIface+".GetStringAtOffset", 0, int32(2), uint32(1)).Store(&piece, &start, &end); err != nil || piece != "aé😀b " || start != 0 || end != 5 {
		t.Errorf("GetStringAtOffset = %q %d %d, %v", piece, start, end, err)
	}
	if err := field.Call(textIface+".GetTextAtOffset", 0, int32(5), uint32(1)).Store(&piece, &start, &end); err != nil || piece != "c" {
		t.Errorf("GetTextAtOffset = %q, %v", piece, err)
	}
	var x, y, w, h int32
	if err := field.Call(textIface+".GetCharacterExtents", 0, int32(0), uint32(1)).Store(&x, &y, &w, &h); err != nil || w != 100 || h != 40 {
		t.Errorf("GetCharacterExtents = %d %d %d %d, %v", x, y, w, h, err)
	}
	var ranges []struct {
		Start, End int32
		Content    string
		Data       godbus.Variant
	}
	if err := field.Call(textIface+".GetBoundedRanges", 0, int32(0), int32(0), int32(10), int32(10), uint32(1), uint32(0), uint32(0)).Store(&ranges); err != nil || len(ranges) != 0 {
		t.Errorf("GetBoundedRanges = %v, %v", ranges, err)
	}
	var attrs map[string]string
	if err := field.Call(textIface+".GetAttributes", 0, int32(0)).Store(&attrs, &start, &end); err != nil || end != 6 {
		t.Errorf("GetAttributes = %v %d %d, %v", attrs, start, end, err)
	}
	var count int32
	if err := field.Call(textIface+".GetNSelections", 0).Store(&count); err != nil || count != 0 {
		t.Errorf("GetNSelections = %d, %v", count, err)
	}
	var ok bool
	if err := field.Call(textIface+".SetSelection", 0, int32(0), int32(1), int32(3)).Store(&ok); err != nil || !ok {
		t.Errorf("SetSelection = %v, %v", ok, err)
	}
	if err := field.Call(editableIface+".InsertText", 0, int32(6), "!", int32(1)).Store(&ok); err != nil || !ok {
		t.Errorf("InsertText = %v, %v", ok, err)
	}
	want := []ui.SemanticsActionEvent{
		{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionSetSelection, Arguments: map[string]int{"base": 1, "extent": 7}},
		{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionSetText, Arguments: sampleText + "!"},
	}
	if got := host.performed(); !reflect.DeepEqual(got, want) {
		t.Errorf("performed %+v\nwant      %+v", got, want)
	}
	var caret godbus.Variant
	if err := field.Call(props+".Get", 0, textIface, "CaretOffset").Store(&caret); err != nil || caret.Value() != int32(3) {
		t.Errorf("CaretOffset = %v, %v", caret, err)
	}

	slider := obj(path(0, 3))
	var cur godbus.Variant
	if err := slider.Call(props+".Get", 0, valueIface, "CurrentValue").Store(&cur); err != nil || cur.Value() != 40.0 {
		t.Errorf("CurrentValue = %v, %v", cur, err)
	}
	if err := slider.Call(props+".Set", 0, valueIface, "CurrentValue", godbus.MakeVariant(41.0)).Err; err != nil {
		t.Errorf("Set CurrentValue: %v", err)
	}
	list := obj(path(0, 5))
	var selected struct {
		Name string
		Path godbus.ObjectPath
	}
	if err := list.Call(selectionIface+".GetSelectedChild", 0, int32(0)).Store(&selected); err != nil || selected.Path != path(0, 7) {
		t.Errorf("GetSelectedChild = %+v, %v", selected, err)
	}
	var n godbus.Variant
	if err := list.Call(props+".Get", 0, selectionIface, "NSelectedChildren").Store(&n); err != nil || n.Value() != int32(1) {
		t.Errorf("NSelectedChildren = %v, %v", n, err)
	}

	// The events of text and of the value.
	update := textTree()
	update.Nodes[1].Value = sampleText + "!"
	update.Nodes[1].TextSelectionBase, update.Nodes[1].TextSelectionExtent = 11, 11
	update.Nodes[3].Value = "70%"
	bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{update.Nodes[1], update.Nodes[3]}})
	seen := map[string]bool{}
	deadline := time.After(5 * time.Second)
	for !seen["TextChanged"] || !seen["TextCaretMoved"] || !seen["PropertyChange"] {
		select {
		case sig := <-events:
			switch sig.Name {
			case objectEvent + "TextChanged":
				if sig.Body[0] != "insert" || sig.Body[1] != int32(6) || sig.Body[2] != int32(1) || sig.Body[3].(godbus.Variant).Value() != "!" {
					t.Errorf("text-changed body = %#v", sig.Body)
				}
				seen["TextChanged"] = true
			case objectEvent + "TextCaretMoved":
				if sig.Body[1] != int32(7) {
					t.Errorf("text-caret-moved body = %#v", sig.Body)
				}
				seen["TextCaretMoved"] = true
			case objectEvent + "PropertyChange":
				if sig.Body[0] == "accessible-value" && sig.Body[3].(godbus.Variant).Value() == 70.0 {
					seen["PropertyChange"] = true
				}
			}
		case <-deadline:
			t.Fatalf("events seen: %v", seen)
		}
	}
}
