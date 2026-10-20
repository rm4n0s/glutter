package atspi_test

import (
	"reflect"
	"testing"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/ui"
)

const (
	textIface      = "org.a11y.atspi.Text"
	editableIface  = "org.a11y.atspi.EditableText"
	valueIface     = "org.a11y.atspi.Value"
	selectionIface = "org.a11y.atspi.Selection"
)

// "aé😀b c" has 6 characters and 10 bytes.
const sampleText = "aé😀b c"

// textTree is a window with a text field (node 1) that has sampleText with the caret after the
// emoji, a read-only field (node 2), a slider (node 3), a progress bar (node 4) and a
// list of three items, the second selected (node 5 with children 6, 7 and 8).
func textTree() *ui.SemanticsUpdate {
	return &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{
		node(0, []int{1, 2, 3, 4, 5}, func(n *ui.SemanticsNodeUpdate) { n.Rect = ui.Rect{Right: 400, Bottom: 300} }),
		node(1, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Name"
			n.Value = sampleText
			n.Flags.IsTextField = true
			n.Flags.IsFocused = ui.TristateIsTrue
			n.TextSelectionBase, n.TextSelectionExtent = 7, 7
			n.Actions = ui.SemanticsActionSetText | ui.SemanticsActionSetSelection | ui.SemanticsActionCopy |
				ui.SemanticsActionCut | ui.SemanticsActionPaste
		}),
		node(2, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Info"
			n.Value = "read only"
			n.Flags.IsTextField = true
			n.Flags.IsReadOnly = true
			n.TextSelectionBase, n.TextSelectionExtent = -1, -1
		}),
		node(3, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Volume"
			n.Value = "40%"
			n.Flags.IsSlider = true
			n.Actions = ui.SemanticsActionIncrease | ui.SemanticsActionDecrease
		}),
		node(4, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "Progress"
			n.Value = "3"
			n.MinValue, n.MaxValue = "0", "10"
			n.Role = ui.SemanticsRoleProgressBar
		}),
		node(5, []int{6, 7, 8}, func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleList; n.Label = "Letters" }),
		node(6, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "A"
			n.Role = ui.SemanticsRoleListItem
			n.Flags.IsSelected = ui.TristateIsFalse
			n.Actions = ui.SemanticsActionTap
		}),
		node(7, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "B"
			n.Role = ui.SemanticsRoleListItem
			n.Flags.IsSelected = ui.TristateIsTrue
			n.Actions = ui.SemanticsActionTap
		}),
		node(8, nil, func(n *ui.SemanticsNodeUpdate) {
			n.Label = "C"
			n.Role = ui.SemanticsRoleListItem
			n.Flags.IsSelected = ui.TristateIsFalse
		}),
	}}
}

func newTextRig(t *testing.T) *rig {
	t.Helper()
	r := newRig(t, true)
	r.start()
	r.waitOn()
	r.bridge.SetWindow(0, "w", true)
	r.bridge.UpdateSemantics(0, textTree())
	return r
}

func TestBridgeInterfacesOfTextValueAndSelection(t *testing.T) {
	r := newTextRig(t)
	tests := []struct {
		node     int
		ifaces   []string
		exported []string
		notExp   []string
	}{
		{1, []string{accessible, component, action, textIface, editableIface}, []string{textIface, editableIface}, []string{selectionIface}},
		{2, []string{accessible, component, textIface}, []string{textIface}, []string{editableIface}},
		{3, []string{accessible, component, action, valueIface}, nil, []string{textIface, selectionIface}},
		{4, []string{accessible, component, valueIface}, nil, nil},
		{5, []string{accessible, component, selectionIface}, []string{selectionIface}, []string{textIface}},
		{6, []string{accessible, component, action}, nil, nil},
	}
	for _, tt := range tests {
		got := r.call(path(0, tt.node), accessible, "GetInterfaces")[0]
		if !reflect.DeepEqual(got, tt.ifaces) {
			t.Errorf("node %d: interfaces = %v, want %v", tt.node, got, tt.ifaces)
		}
		for _, i := range tt.exported {
			if !r.a11y.Exported(path(0, tt.node), i) {
				t.Errorf("node %d does not serve %s", tt.node, i)
			}
		}
		for _, i := range tt.notExp {
			if r.a11y.Exported(path(0, tt.node), i) {
				t.Errorf("node %d serves %s", tt.node, i)
			}
		}
	}
	// The states of the text fields.
	words := r.call(path(0, 1), accessible, "GetState")[0].([]uint32)
	for _, s := range []uint32{7, 11, 12, 26} { // editable, focusable, focused, single line
		if words[0]&(1<<s) == 0 {
			t.Errorf("state %d is not set", s)
		}
	}
	if words[1]&(1<<(38-32)) == 0 { // selectable text
		t.Error("the field has no selectable-text state")
	}
}

func TestBridgeText(t *testing.T) {
	r := newTextRig(t)
	field := path(0, 1)
	if got := r.prop(field, textIface, "CharacterCount"); got != int32(6) {
		t.Errorf("CharacterCount = %v", got)
	}
	if got := r.prop(field, textIface, "CaretOffset"); got != int32(3) {
		t.Errorf("CaretOffset = %v (bytes 7 are after 3 characters)", got)
	}
	if got := r.call(field, textIface, "GetCharacterCount")[0]; got != int32(6) {
		t.Errorf("GetCharacterCount = %v", got)
	}
	if got := r.call(field, textIface, "GetCaretOffset")[0]; got != int32(3) {
		t.Errorf("GetCaretOffset = %v", got)
	}
	texts := []struct {
		start, end int32
		want       string
	}{{0, -1, sampleText}, {1, 3, "é😀"}, {3, 100, "b c"}, {4, 2, ""}, {-5, 1, "a"}}
	for _, tt := range texts {
		if got := r.call(field, textIface, "GetText", tt.start, tt.end)[0]; got != tt.want {
			t.Errorf("GetText(%d, %d) = %q, want %q", tt.start, tt.end, got, tt.want)
		}
	}
	for offset, want := range map[int32]int32{0: 'a', 2: 0x1F600, 5: 'c', 6: -1, -1: -1} {
		if got := r.call(field, textIface, "GetCharacterAtOffset", offset)[0]; got != want {
			t.Errorf("GetCharacterAtOffset(%d) = %v, want %d", offset, got, want)
		}
	}
	// Words: "aé😀b " and "c".
	got := r.call(field, textIface, "GetStringAtOffset", int32(2), uint32(1))
	if got[0] != "aé😀b " || got[1] != int32(0) || got[2] != int32(5) {
		t.Errorf("GetStringAtOffset word = %v", got)
	}
	got = r.call(field, textIface, "GetStringAtOffset", int32(2), uint32(0))
	if got[0] != "😀" || got[1] != int32(2) || got[2] != int32(3) {
		t.Errorf("GetStringAtOffset char = %v", got)
	}
	got = r.call(field, textIface, "GetTextAtOffset", int32(5), uint32(1))
	if got[0] != "c" || got[1] != int32(5) || got[2] != int32(6) {
		t.Errorf("GetTextAtOffset = %v", got)
	}
	got = r.call(field, textIface, "GetTextBeforeOffset", int32(5), uint32(1))
	if got[0] != "aé😀b " {
		t.Errorf("GetTextBeforeOffset = %v", got)
	}
	got = r.call(field, textIface, "GetTextAfterOffset", int32(0), uint32(1))
	if got[0] != "c" {
		t.Errorf("GetTextAfterOffset = %v", got)
	}
	for _, m := range []string{"GetTextAtOffset", "GetTextBeforeOffset", "GetTextAfterOffset"} {
		if _, err := r.a11y.Invoke(field, textIface, m, int32(0), uint32(9)); err == nil {
			t.Errorf("%s with a bad boundary did not fail", m)
		}
	}
	if _, err := r.a11y.Invoke(field, textIface, "GetStringAtOffset", int32(0), uint32(9)); err == nil {
		t.Error("a bad granularity did not fail")
	}
	// No selection yet: the caret is collapsed.
	if got := r.call(field, textIface, "GetNSelections")[0]; got != int32(0) {
		t.Errorf("GetNSelections = %v", got)
	}
	if _, err := r.a11y.Invoke(field, textIface, "GetSelection", int32(0)); err == nil {
		t.Error("GetSelection without a selection did not fail")
	}
	// Things that the text has no answer for.
	if got := r.call(field, textIface, "GetOffsetAtPoint", int32(1), int32(1), uint32(1))[0]; got != int32(-1) {
		t.Errorf("GetOffsetAtPoint = %v", got)
	}
	if got := r.call(field, textIface, "GetAttributes", int32(0)); got[1] != int32(0) || got[2] != int32(6) || len(got[0].(map[string]string)) != 0 {
		t.Errorf("GetAttributes = %v", got)
	}
	if got := r.call(field, textIface, "GetDefaultAttributes")[0]; len(got.(map[string]string)) != 0 {
		t.Errorf("GetDefaultAttributes = %v", got)
	}
	ext := r.call(field, textIface, "GetRangeExtents", int32(0), int32(2), uint32(1))
	if ext[2] != int32(100) || ext[3] != int32(40) {
		t.Errorf("GetRangeExtents = %v", ext)
	}
	// A field that is not there.
	if _, err := r.a11y.Invoke(path(0, 99), textIface, "GetText", int32(0), int32(1)); err == nil {
		t.Error("an object that is not there answers")
	}
}

func TestBridgeTextSelection(t *testing.T) {
	r := newTextRig(t)
	field := path(0, 1)
	// A selection from the character 1 to the character 4 (the bytes 1 to 8; the caret is the end).
	tree := textTree()
	tree.Nodes[1].TextSelectionBase, tree.Nodes[1].TextSelectionExtent = 1, 8
	r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{tree.Nodes[1]}})
	if got := r.call(field, textIface, "GetNSelections")[0]; got != int32(1) {
		t.Fatalf("GetNSelections = %v", got)
	}
	if got := r.call(field, textIface, "GetSelection", int32(0)); got[0] != int32(1) || got[1] != int32(4) {
		t.Errorf("GetSelection = %v", got)
	}
	if got := r.prop(field, textIface, "CaretOffset"); got != int32(4) {
		t.Errorf("CaretOffset = %v", got)
	}
	if _, err := r.a11y.Invoke(field, textIface, "GetSelection", int32(1)); err == nil {
		t.Error("a second selection")
	}

	// The requests go to the framework with byte offsets.
	sel := func(base, extent int) ui.SemanticsActionEvent {
		return ui.SemanticsActionEvent{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionSetSelection,
			Arguments: map[string]int{"base": base, "extent": extent}}
	}
	var want []ui.SemanticsActionEvent
	check := func(what string) {
		t.Helper()
		if got := r.host.performed(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: performed %+v\nwant %+v", what, got, want)
		}
	}
	if got := r.call(field, textIface, "SetCaretOffset", int32(3))[0]; got != true {
		t.Error("SetCaretOffset failed")
	}
	want = append(want, sel(7, 7))
	check("SetCaretOffset")
	if got := r.call(field, textIface, "SetSelection", int32(0), int32(0), int32(6))[0]; got != true {
		t.Error("SetSelection failed")
	}
	want = append(want, sel(0, 10))
	check("SetSelection")
	if got := r.call(field, textIface, "RemoveSelection", int32(0))[0]; got != true {
		t.Error("RemoveSelection failed")
	}
	want = append(want, sel(8, 8)) // the caret stays where the selection ended: the character 4
	check("RemoveSelection")
	if got := r.call(field, textIface, "AddSelection", int32(1), int32(2))[0]; got != false {
		t.Error("AddSelection worked with a selection")
	}
	check("AddSelection with a selection")
	for _, args := range [][]any{{int32(1), int32(0), int32(1)}, {int32(0), int32(-1), int32(1)}, {int32(0), int32(0), int32(7)}} {
		if got := r.call(field, textIface, "SetSelection", args...)[0]; got != false {
			t.Errorf("SetSelection(%v) = %v", args, got)
		}
	}
	if got := r.call(field, textIface, "RemoveSelection", int32(2))[0]; got != false {
		t.Error("RemoveSelection of another selection worked")
	}
	check("the invalid ones")
	// A read-only field has no selection action.
	if got := r.call(path(0, 2), textIface, "SetCaretOffset", int32(1))[0]; got != false {
		t.Error("the read-only field took a caret")
	}
}

func TestBridgeEditableText(t *testing.T) {
	r := newTextRig(t)
	field := path(0, 1)
	setText := func(s string) ui.SemanticsActionEvent {
		return ui.SemanticsActionEvent{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionSetText, Arguments: s}
	}
	sel := func(base, extent int) ui.SemanticsActionEvent {
		return ui.SemanticsActionEvent{ViewID: 0, NodeID: 1, Type: ui.SemanticsActionSetSelection,
			Arguments: map[string]int{"base": base, "extent": extent}}
	}
	act := func(a ui.SemanticsAction) ui.SemanticsActionEvent {
		return ui.SemanticsActionEvent{ViewID: 0, NodeID: 1, Type: a}
	}
	tests := []struct {
		name   string
		method string
		args   []any
		ok     bool
		want   []ui.SemanticsActionEvent
	}{
		{"SetTextContents", "SetTextContents", []any{"new"}, true, []ui.SemanticsActionEvent{setText("new")}},
		{"InsertText", "InsertText", []any{int32(3), "XYZ", int32(2)}, true, []ui.SemanticsActionEvent{setText("aé😀XYb c")}},
		{"InsertText with a length of -1", "InsertText", []any{int32(0), "é", int32(-1)}, true, []ui.SemanticsActionEvent{setText("éaé😀b c")}},
		{"InsertText at the end", "InsertText", []any{int32(6), "!", int32(1)}, true, []ui.SemanticsActionEvent{setText("aé😀b c!")}},
		{"InsertText outside", "InsertText", []any{int32(7), "!", int32(1)}, false, nil},
		{"DeleteText", "DeleteText", []any{int32(1), int32(3)}, true, []ui.SemanticsActionEvent{setText("ab c")}},
		{"DeleteText outside", "DeleteText", []any{int32(1), int32(9)}, false, nil},
		{"DeleteText backwards", "DeleteText", []any{int32(3), int32(1)}, false, nil},
		{"CopyText", "CopyText", []any{int32(1), int32(3)}, true, []ui.SemanticsActionEvent{sel(1, 7), act(ui.SemanticsActionCopy)}},
		{"CutText", "CutText", []any{int32(0), int32(1)}, true, []ui.SemanticsActionEvent{sel(0, 1), act(ui.SemanticsActionCut)}},
		{"PasteText", "PasteText", []any{int32(3)}, true, []ui.SemanticsActionEvent{sel(7, 7), act(ui.SemanticsActionPaste)}},
		{"CopyText outside", "CopyText", []any{int32(0), int32(9)}, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := len(r.host.performed())
			if got := r.call(field, editableIface, tt.method, tt.args...)[0]; got != tt.ok {
				t.Errorf("result = %v, want %v", got, tt.ok)
			}
			if got := r.host.performed()[before:]; !reflect.DeepEqual(got, tt.want) && (len(got) != 0 || len(tt.want) != 0) {
				t.Errorf("performed %+v\nwant      %+v", got, tt.want)
			}
		})
	}
	// A read-only field is not editable.
	if r.a11y.Exported(path(0, 2), editableIface) {
		t.Error("the read-only field serves EditableText")
	}
	// A field that is not focused has no actions to set its text or its selection: it is
	// still editable, and the methods say no, until the focus comes.
	update := textTree().Nodes[1]
	update.Actions = ui.SemanticsActionFocus
	update.Flags.IsFocused = ui.TristateIsFalse
	r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{update}})
	if !r.a11y.Exported(field, editableIface) || !r.a11y.Exported(field, textIface) {
		t.Error("the field lost an interface when it lost the focus")
	}
	before := len(r.host.performed())
	for _, c := range []struct {
		iface, method string
		args          []any
	}{
		{editableIface, "SetTextContents", []any{"x"}}, {editableIface, "InsertText", []any{int32(0), "x", int32(1)}},
		{editableIface, "DeleteText", []any{int32(0), int32(1)}}, {editableIface, "CopyText", []any{int32(0), int32(1)}},
		{textIface, "SetCaretOffset", []any{int32(1)}},
	} {
		if got := r.call(field, c.iface, c.method, c.args...)[0]; got != false {
			t.Errorf("%s of a field that is not focused = %v", c.method, got)
		}
	}
	if got := len(r.host.performed()); got != before {
		t.Errorf("%d actions were performed on a field that cannot take them", got-before)
	}
	// A screen reader focuses the field first, with GrabFocus.
	if got := r.call(field, component, "GrabFocus")[0]; got != true {
		t.Error("the field did not take the focus")
	}
}

func TestBridgeTextEvents(t *testing.T) {
	r := newTextRig(t)
	field := path(0, 1)
	event := func(from int, member string) []godbus.Signal {
		var out []godbus.Signal
		for _, s := range r.a11y.Signals()[from:] {
			if s.Name == objectEvent+member && s.Path == field {
				out = append(out, s)
			}
		}
		return out
	}
	change := func(edit func(n *ui.SemanticsNodeUpdate)) int {
		before := len(r.a11y.Signals())
		n := textTree().Nodes[1]
		edit(&n)
		r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{n}})
		return before
	}
	body := func(s godbus.Signal) []any {
		return []any{s.Body[0], s.Body[1], s.Body[2], s.Body[3].(godbus.Variant).Value()}
	}

	// Typing at the end: an insert of "!" at the character 6; the caret moves with it.
	from := change(func(n *ui.SemanticsNodeUpdate) {
		n.Value = sampleText + "!"
		n.TextSelectionBase, n.TextSelectionExtent = 11, 11
	})
	if got := event(from, "TextChanged"); len(got) != 1 || !reflect.DeepEqual(body(got[0]), []any{"insert", int32(6), int32(1), "!"}) {
		t.Errorf("text-changed = %+v", got)
	}
	if got := event(from, "TextCaretMoved"); len(got) != 1 || got[0].Body[1] != int32(7) {
		t.Errorf("text-caret-moved = %+v", got)
	}

	// Replacing a word: a delete, then an insert at the same place.
	from = change(func(n *ui.SemanticsNodeUpdate) {
		n.Value = "aé😀b d!"
		n.TextSelectionBase, n.TextSelectionExtent = 11, 11
	})
	got := event(from, "TextChanged")
	if len(got) != 2 || !reflect.DeepEqual(body(got[0]), []any{"delete", int32(5), int32(1), "c"}) ||
		!reflect.DeepEqual(body(got[1]), []any{"insert", int32(5), int32(1), "d"}) {
		t.Errorf("text-changed = %+v", got)
	}
	if got := event(from, "TextCaretMoved"); len(got) != 0 {
		t.Errorf("the caret did not move: %+v", got)
	}

	// A selection: the selection changes, and so does the caret.
	from = change(func(n *ui.SemanticsNodeUpdate) {
		n.Value = "aé😀b d!"
		n.TextSelectionBase, n.TextSelectionExtent = 0, 7
	})
	if got := event(from, "TextSelectionChanged"); len(got) != 1 {
		t.Errorf("text-selection-changed = %+v", got)
	}
	if got := event(from, "TextCaretMoved"); len(got) != 1 || got[0].Body[1] != int32(3) {
		t.Errorf("text-caret-moved = %+v", got)
	}
	// Collapsing it is a selection change too.
	from = change(func(n *ui.SemanticsNodeUpdate) {
		n.Value = "aé😀b d!"
		n.TextSelectionBase, n.TextSelectionExtent = 7, 7
	})
	if got := event(from, "TextSelectionChanged"); len(got) != 1 {
		t.Errorf("text-selection-changed = %+v", got)
	}
	// Nothing changed in the text: no event.
	from = change(func(n *ui.SemanticsNodeUpdate) {
		n.Value = "aé😀b d!"
		n.TextSelectionBase, n.TextSelectionExtent = 7, 7
	})
	for _, m := range []string{"TextChanged", "TextCaretMoved", "TextSelectionChanged", "VisibleDataChanged"} {
		if got := event(from, m); len(got) != 0 {
			t.Errorf("%s without a change: %+v", m, got)
		}
	}
	// A move of the field is a change of visible data.
	from = change(func(n *ui.SemanticsNodeUpdate) {
		n.Value = "aé😀b d!"
		n.TextSelectionBase, n.TextSelectionExtent = 7, 7
		n.Transform = matrix(1, 30, 0)
	})
	if got := event(from, "VisibleDataChanged"); len(got) != 1 {
		t.Errorf("visible-data-changed = %+v", got)
	}
}

func TestBridgeValue(t *testing.T) {
	r := newTextRig(t)
	slider, progress := path(0, 3), path(0, 4)
	for _, tt := range []struct {
		path               godbus.ObjectPath
		cur, min, max, inc float64
	}{{slider, 40, 0, 100, 1}, {progress, 3, 0, 10, 0.1}} {
		for name, want := range map[string]float64{"CurrentValue": tt.cur, "MinimumValue": tt.min, "MaximumValue": tt.max, "MinimumIncrement": tt.inc} {
			if got := r.prop(tt.path, valueIface, name); got != want {
				t.Errorf("%s of %s = %v, want %v", name, tt.path, got, want)
			}
		}
	}
	if got := r.prop(slider, valueIface, "Text"); got != "40%" {
		t.Errorf("Text = %v", got)
	}
	set := func(p godbus.ObjectPath, v float64) error {
		_, err := r.a11y.Invoke(p, props, "Set", valueIface, "CurrentValue", godbus.MakeVariant(v))
		return err
	}
	if err := set(slider, 60); err != nil {
		t.Fatal(err)
	}
	if err := set(slider, 10); err != nil {
		t.Fatal(err)
	}
	if err := set(slider, 40); err != nil { // the value it has: nothing to do
		t.Fatal(err)
	}
	want := []ui.SemanticsActionEvent{
		{ViewID: 0, NodeID: 3, Type: ui.SemanticsActionIncrease},
		{ViewID: 0, NodeID: 3, Type: ui.SemanticsActionDecrease},
	}
	if got := r.host.performed(); !reflect.DeepEqual(got, want) {
		t.Errorf("performed %+v", got)
	}
	if err := set(progress, 5); err == nil {
		t.Error("a progress bar cannot be changed")
	}
	if _, err := r.a11y.Invoke(slider, props, "Set", valueIface, "CurrentValue", godbus.MakeVariant("high")); err == nil {
		t.Error("a string is not a value")
	}
	if _, err := r.a11y.Invoke(path(0, 1), props, "Set", valueIface, "CurrentValue", godbus.MakeVariant(1.0)); err == nil {
		t.Error("a text field has no value")
	}
	if _, err := r.a11y.Invoke(slider, props, "Set", valueIface, "MinimumValue", godbus.MakeVariant(1.0)); err == nil {
		t.Error("the minimum was set")
	}

	// The event when the value changes.
	before := len(r.a11y.Signals())
	update := textTree().Nodes[3]
	update.Value = "55%"
	r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{update}})
	var got []any
	for _, s := range r.a11y.Signals()[before:] {
		if s.Name == objectEvent+"PropertyChange" && s.Body[0] == "accessible-value" && s.Path == slider {
			got = append(got, s.Body[3].(godbus.Variant).Value())
		}
	}
	if !reflect.DeepEqual(got, []any{55.0}) {
		t.Errorf("accessible-value events = %v", got)
	}
	if got := r.prop(slider, valueIface, "CurrentValue"); got != 55.0 {
		t.Errorf("CurrentValue = %v", got)
	}
}

func TestBridgeSelection(t *testing.T) {
	r := newTextRig(t)
	list := path(0, 5)
	if got := r.prop(list, selectionIface, "NSelectedChildren"); got != int32(1) {
		t.Errorf("NSelectedChildren = %v", got)
	}
	if got := toRef(r.call(list, selectionIface, "GetSelectedChild", int32(0))[0]); got.Path != path(0, 7) {
		t.Errorf("GetSelectedChild = %v", got)
	}
	for i, want := range []bool{false, true, false} {
		if got := r.call(list, selectionIface, "IsChildSelected", int32(i))[0]; got != want {
			t.Errorf("IsChildSelected(%d) = %v", i, got)
		}
	}
	tap := func(id int) ui.SemanticsActionEvent {
		return ui.SemanticsActionEvent{ViewID: 0, NodeID: id, Type: ui.SemanticsActionTap}
	}
	var want []ui.SemanticsActionEvent
	check := func(what string) {
		t.Helper()
		if got := r.host.performed(); !reflect.DeepEqual(got, want) && (len(got) != 0 || len(want) != 0) {
			t.Errorf("%s: performed %+v, want %+v", what, got, want)
		}
	}
	// Selecting a child that is not selected taps it; one that is selected needs nothing.
	if got := r.call(list, selectionIface, "SelectChild", int32(0))[0]; got != true {
		t.Error("SelectChild(0) failed")
	}
	want = append(want, tap(6))
	check("SelectChild")
	if got := r.call(list, selectionIface, "SelectChild", int32(1))[0]; got != true {
		t.Error("SelectChild of the selected one failed")
	}
	check("SelectChild of the selected one")
	// The third child cannot be tapped.
	if got := r.call(list, selectionIface, "SelectChild", int32(2))[0]; got != false {
		t.Error("SelectChild(2) worked without a tap action")
	}
	if got := r.call(list, selectionIface, "DeselectChild", int32(1))[0]; got != true {
		t.Error("DeselectChild(1) failed")
	}
	want = append(want, tap(7))
	check("DeselectChild")
	if got := r.call(list, selectionIface, "DeselectChild", int32(0))[0]; got != true {
		t.Error("DeselectChild of a child that is not selected failed")
	}
	check("DeselectChild of an unselected one")
	if got := r.call(list, selectionIface, "DeselectSelectedChild", int32(0))[0]; got != true {
		t.Error("DeselectSelectedChild failed")
	}
	want = append(want, tap(7))
	check("DeselectSelectedChild")
	for _, m := range []string{"SelectAll", "ClearSelection"} {
		if got := r.call(list, selectionIface, m)[0]; got != false {
			t.Errorf("%s = %v", m, got)
		}
	}
	for _, c := range []struct {
		method string
		index  int32
	}{{"GetSelectedChild", 1}, {"SelectChild", 3}, {"DeselectChild", -1}, {"DeselectSelectedChild", 5}, {"IsChildSelected", 3}} {
		if _, err := r.a11y.Invoke(list, selectionIface, c.method, c.index); err == nil {
			t.Errorf("%s(%d) did not fail", c.method, c.index)
		}
	}

	// Another child gets selected: the list says so.
	before := len(r.a11y.Signals())
	update := textTree().Nodes[6]
	update.Flags.IsSelected = ui.TristateIsTrue
	r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{update}})
	var changed int
	for _, s := range r.a11y.Signals()[before:] {
		if s.Name == objectEvent+"SelectionChanged" && s.Path == list {
			changed++
		}
	}
	if changed != 1 {
		t.Errorf("%d selection-changed events, want 1", changed)
	}
	if got := r.prop(list, selectionIface, "NSelectedChildren"); got != int32(2) {
		t.Errorf("NSelectedChildren = %v", got)
	}
}

func TestBridgeSelectionAppearsWhenChildrenCanBeSelected(t *testing.T) {
	r := newRig(t, true)
	r.start()
	r.waitOn()
	plain := func(sel ui.Tristate) *ui.SemanticsUpdate {
		return &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{
			node(0, []int{1}, nil),
			node(1, []int{2}, nil),
			node(2, nil, func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSelected = sel }),
		}}
	}
	r.bridge.UpdateSemantics(0, plain(ui.TristateNone))
	if r.a11y.Exported(path(0, 1), selectionIface) {
		t.Fatal("a parent of nothing selectable serves Selection")
	}
	// Only the child is in this update; the parent gets the interface all the same.
	r.bridge.UpdateSemantics(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{plain(ui.TristateIsFalse).Nodes[2]}})
	if !r.a11y.Exported(path(0, 1), selectionIface) {
		t.Error("the parent does not serve Selection")
	}
	if got := r.call(path(0, 1), accessible, "GetInterfaces")[0]; !reflect.DeepEqual(got, []string{accessible, component, selectionIface}) {
		t.Errorf("interfaces = %v", got)
	}
}
