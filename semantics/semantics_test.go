package semantics_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// Tests ported from flutter/test/semantics/semantics_test.dart.

var (
	tag1 = semantics.NewSemanticsTag("Tag One")
	tag2 = semantics.NewSemanticsTag("Tag Two")
	tag3 = semantics.NewSemanticsTag("Tag Three")
)

func TestNodeTagging(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	node := semantics.NewSemanticsNode()
	if node.IsTagged(tag1) || node.IsTagged(tag2) {
		t.Fatal("a new node is tagged")
	}
	node.Tags = map[semantics.SemanticsTag]struct{}{tag1: {}}
	if !node.IsTagged(tag1) || node.IsTagged(tag2) {
		t.Error("tags = {tag1}")
	}
	node.Tags[tag2] = struct{}{}
	if !node.IsTagged(tag1) || !node.IsTagged(tag2) {
		t.Error("tags = {tag1, tag2}")
	}
}

func TestGetSemanticsDataIncludesTags(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	tags := map[semantics.SemanticsTag]struct{}{tag1: {}, tag2: {}}
	node := nodeWithRect(rect(0, 0, 10, 10))
	node.Tags = tags
	if got := node.GetSemanticsData().Tags; len(got) != 2 {
		t.Errorf("tags = %v, want 2", got)
	}
	tags[tag3] = struct{}{}
	child := nodeWithRect(rect(5, 5, 10, 10))
	child.Tags = tags
	node.UpdateWith(boundaryConfig(true), []*semantics.SemanticsNode{child})
	if got := node.GetSemanticsData().Tags; len(got) != 3 {
		t.Errorf("merged tags = %v, want 3", got)
	}
}

func TestConfigurationStringsAndAttributedStrings(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	attributed := func(s string) semantics.AttributedString {
		return semantics.NewAttributedString(s, ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 1)))
	}
	const spell = ` [SpellOutStringAttribute(TextRange(start: 0, end: 1))]`
	config := semantics.NewSemanticsConfiguration()
	type view struct {
		plain      func() string
		attributed func() semantics.AttributedString
	}
	label := view{config.Label, config.AttributedLabel}
	value := view{config.Value, config.AttributedValue}
	hint := view{config.Hint, config.AttributedHint}
	tests := []struct {
		name   string
		apply  func()
		view   view
		str    string
		spells bool
	}{
		{"label1", func() { config.SetLabel("label1") }, label, `label: "label1"`, false},
		{"label2", func() { config.SetAttributedLabel(attributed("label2")) }, label, `label: "label2"` + spell, true},
		{"label3", func() { config.SetLabel("label3") }, label, `label: "label3"`, false},
		{"value1", func() { config.SetValue("value1") }, value, `label: "label3", value: "value1"`, false},
		{"value2", func() { config.SetAttributedValue(attributed("value2")) }, value, `label: "label3", value: "value2"` + spell, true},
		{"value3", func() { config.SetValue("value3") }, value, `label: "label3", value: "value3"`, false},
		{"hint1", func() { config.SetHint("hint1") }, hint, `label: "label3", value: "value3", hint: "hint1"`, false},
		{"hint2", func() { config.SetAttributedHint(attributed("hint2")) }, hint, `label: "label3", value: "value3", hint: "hint2"` + spell, true},
		{"hint3", func() { config.SetHint("hint3") }, hint, `label: "label3", value: "value3", hint: "hint3"`, false},
	}
	for i, tc := range tests {
		tc.apply()
		if got, want := tc.view.plain(), tc.view.attributed().String; got != want {
			t.Errorf("%s: plain %q != attributed %q", tc.name, got, want)
		}
		if got := len(tc.view.attributed().Attributes); (got == 1) != tc.spells || got > 1 {
			t.Errorf("%s: %d attributes", tc.name, got)
		}
		node := semantics.NewSemanticsNode()
		node.UpdateWith(config, nil)
		want := fmt.Sprintf("SemanticsNode#%d(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, %s)", i+1, tc.str)
		if got := node.String(); got != want {
			t.Errorf("%s: node = %s\nwant %s", tc.name, got, want)
		}
	}
}

func TestIsMergedIntoParent(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	root := nodeWithRect(rect(0, 0, 10, 10))
	node1 := nodeWithRect(rect(1, 0, 10, 10))
	node11 := nodeWithRect(rect(2, 0, 10, 10))
	node12 := nodeWithRect(rect(3, 0, 10, 10))
	noMerge, merge := boundaryConfig(false), boundaryConfig(true)
	type state struct{ n1, n1all, n11, n12, root, rootAll bool }
	got := func() state {
		return state{node1.IsMergedIntoParent(), node1.MergeAllDescendantsIntoThisNode(), node11.IsMergedIntoParent(),
			node12.IsMergedIntoParent(), root.IsMergedIntoParent(), root.MergeAllDescendantsIntoThisNode()}
	}
	node1.UpdateWith(noMerge, []*semantics.SemanticsNode{node11, node12})
	if g := got(); g != (state{}) {
		t.Errorf("after node1 update = %+v", g)
	}
	root.UpdateWith(merge, []*semantics.SemanticsNode{node1})
	if g, w := got(), (state{n1: true, n11: true, n12: true, rootAll: true}); g != w {
		t.Errorf("after root merge = %+v, want %+v", g, w)
	}
	node1.UpdateWith(merge, []*semantics.SemanticsNode{node11, node12})
	if g, w := got(), (state{n1: true, n1all: true, n11: true, n12: true, rootAll: true}); g != w {
		t.Errorf("after node1 merge = %+v, want %+v", g, w)
	}
	root.UpdateWith(noMerge, []*semantics.SemanticsNode{node1})
	if g, w := got(), (state{n1all: true, n11: true, n12: true}); g != w {
		t.Errorf("after root no merge = %+v, want %+v", g, w)
	}
}

func TestSendSemanticsUpdateVerifiesNoInvisibleNodes(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	invisible, visible := rect(0, 0, 0, 10), rect(0, 0, 10, 10)
	owner := newOwner()
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(invisible)
	child := semantics.NewSemanticsNode()

	// An invisible root is fine.
	owner.SendSemanticsUpdate()

	// An invisible child is fine if it is merged into an ancestor.
	root.SetRect(visible)
	child.SetRect(invisible)
	root.UpdateWith(boundaryConfig(true), []*semantics.SemanticsNode{child})
	owner.SendSemanticsUpdate()

	// All visible is fine.
	child.SetRect(visible)
	root.UpdateWith(boundaryConfig(false), []*semantics.SemanticsNode{child})
	owner.SendSemanticsUpdate()

	const tail = "An invisible SemanticsNode is one whose rect is not on screen hence not reachable for users, and its semantic information is not merged into a visible parent.\n" +
		"An invisible SemanticsNode makes the accessibility experience confusing, as it does not provide any visual indication when the user selects it via accessibility technologies.\n" +
		"Consider removing the above invisible SemanticsNodes if they were added by your RenderObject.assembleSemanticsNode implementation, or filing a bug on GitHub:\n" +
		"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml"

	// An invisible root with children is bad.
	root.SetRect(invisible)
	child.SetRect(invisible)
	root.UpdateWith(boundaryConfig(true), []*semantics.SemanticsNode{child})
	err := requireTag(t, "SemanticsInvisibleNodes", owner.SendSemanticsUpdate)
	want := "Invisible SemanticsNodes should not be added to the tree.\n" +
		"The following invisible SemanticsNodes were added to the tree:\n" +
		"SemanticsNode#0(dirty, merge boundary ⛔️, Rect.fromLTRB(0.0, 0.0, 0.0, 10.0), invisible)\n" +
		"which was added as the root SemanticsNode\n" + tail
	if err.Message != want {
		t.Errorf("message:\n%s\nwant:\n%s", err.Message, want)
	}

	// Invisible children are bad.
	root.SetRect(visible)
	child.SetRect(invisible)
	root.UpdateWith(boundaryConfig(false), []*semantics.SemanticsNode{child})
	err = requireTag(t, "SemanticsInvisibleNodes", owner.SendSemanticsUpdate)
	want = "Invisible SemanticsNodes should not be added to the tree.\n" +
		"The following invisible SemanticsNodes were added to the tree:\n" +
		"SemanticsNode#1(dirty, Rect.fromLTRB(0.0, 0.0, 0.0, 10.0), invisible)\n" +
		"which was added as a child of:\n" +
		"  SemanticsNode#0(dirty, Rect.fromLTRB(0.0, 0.0, 10.0, 10.0))\n" + tail
	if err.Message != want {
		t.Errorf("message:\n%s\nwant:\n%s", err.Message, want)
	}
}

func TestMutatingTheChildListErrors(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	node := nodeWithRect(rect(0, 0, 10, 10))
	config := boundaryConfig(true)
	// Room for a second element, so that appending keeps the list.
	children := make([]*semantics.SemanticsNode, 1, 4)
	children[0] = nodeWithRect(rect(5, 5, 10, 10))
	node.UpdateWith(config, children)

	children = append(children, nodeWithRect(rect(42, 42, 52, 52)))
	err := requireTag(t, "SemanticsChildrenMutated", func() { node.UpdateWith(config, children) })
	want := "Failed to replace child semantics nodes because the list of `SemanticsNode`s was mutated.\n" +
		"Instead of mutating the existing list, create a new list containing the desired `SemanticsNode`s.\n" +
		"Error details:\n" +
		"The list's length has changed from 1 to 2."
	if err.Message != want {
		t.Errorf("message:\n%s\nwant:\n%s", err.Message, want)
	}

	modified := []*semantics.SemanticsNode{nodeWithRect(rect(5, 5, 10, 10)), nodeWithRect(rect(10, 10, 20, 20))}
	node.UpdateWith(config, modified)
	previous0, previous1 := modified[0].String(), modified[1].String()
	modified[0] = nodeWithRect(rect(0, 0, 20, 20))
	modified[1] = nodeWithRect(rect(40, 14, 60, 60))
	err = requireTag(t, "SemanticsChildrenMutated", func() { node.UpdateWith(config, modified) })
	want = "Failed to replace child semantics nodes because the list of `SemanticsNode`s was mutated.\n" +
		"Instead of mutating the existing list, create a new list containing the desired `SemanticsNode`s.\n" +
		"Error details:\n" +
		"Child node at position 0 was replaced:\n" +
		"Previous child: " + previous0 + "\n" +
		"New child: " + modified[0].String() + "\n\n" +
		"Child node at position 1 was replaced:\n" +
		"Previous child: " + previous1 + "\n" +
		"New child: " + modified[1].String()
	if err.Message != want {
		t.Errorf("message:\n%s\nwant:\n%s", err.Message, want)
	}
	if !strings.Contains(previous0, "merged up ⬆️, Rect.fromLTRB(5.0, 5.0, 10.0, 10.0)") {
		t.Errorf("previous child = %s", previous0)
	}
}

func TestUpdateWithMarksNodeDirtyWhenRoleChanges(t *testing.T) {
	skipUnlessDebug(t)
	node := semantics.NewSemanticsNode()
	if node.Role() != ui.SemanticsRoleNone {
		t.Errorf("role = %v", node.Role())
	}
	if d, ok := node.DebugIsDirty().Get(); !ok || d {
		t.Errorf("a new node should not be dirty: %v", node.DebugIsDirty())
	}
	config := semantics.NewSemanticsConfiguration()
	config.SetRole(ui.SemanticsRoleTab)
	node.UpdateWith(config, nil)
	if node.Role() != ui.SemanticsRoleTab {
		t.Errorf("role = %v", node.Role())
	}
	if d, _ := node.DebugIsDirty().Get(); !d {
		t.Error("the node should be dirty")
	}
}

func TestUpdateWithMarksNodeDirtyWhenCustomActionsChange(t *testing.T) {
	skipUnlessDebug(t)
	owner := newOwner()
	node := semantics.NewSemanticsNodeRoot(owner)
	node.SetRect(rect(0, 0, 10, 10))
	configWith := func(label string) *semantics.SemanticsConfiguration {
		c := semantics.NewSemanticsConfiguration()
		c.SetIsSemanticBoundary(true)
		c.SetCustomSemanticsActions(map[semantics.CustomSemanticsAction]semantics.VoidCallback{semantics.NewCustomSemanticsAction(label): noop})
		return c
	}
	dirty := func() bool { d, _ := node.DebugIsDirty().Get(); return d }
	node.UpdateWith(configWith("Action 1"), nil)
	if !dirty() {
		t.Error("the node should be dirty after the first update")
	}
	owner.SendSemanticsUpdate()
	if dirty() {
		t.Error("the node should be clean after the update was sent")
	}
	node.UpdateWith(configWith("Action 2"), nil)
	if !dirty() {
		t.Error("the node should be dirty after the custom actions changed")
	}
}

func TestToStringDeepWithoutTransform(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	child1 := nodeWithRect(rect(0, 0, 5, 5))
	child2 := nodeWithRect(rect(5, 0, 10, 5))
	root := nodeWithRect(rect(0, 0, 10, 5))
	root.UpdateWith(nil, []*semantics.SemanticsNode{child1, child2})
	if root.Transform() != nil || child1.Transform() != nil || child2.Transform() != nil {
		t.Error("transforms should be nil")
	}
	want := "SemanticsNode#3\n" +
		" │ STALE\n" +
		" │ owner: null\n" +
		" │ Rect.fromLTRB(0.0, 0.0, 10.0, 5.0)\n" +
		" │\n" +
		" ├─SemanticsNode#1\n" +
		" │   STALE\n" +
		" │   owner: null\n" +
		" │   Rect.fromLTRB(0.0, 0.0, 5.0, 5.0)\n" +
		" │\n" +
		" └─SemanticsNode#2\n" +
		"     STALE\n" +
		"     owner: null\n" +
		"     Rect.fromLTRB(5.0, 0.0, 10.0, 5.0)\n"
	if got := root.ToStringDeep(); got != want {
		t.Errorf("ToStringDeep:\n%s\nwant:\n%s", got, want)
	}
}

type customSortKey struct{ *semantics.OrdinalSortKey }

func TestOrdinalSortKeyIncompatibleTypesPanic(t *testing.T) {
	skipUnlessDebug(t)
	other := customSortKey{semantics.NewOrdinalSortKey(0)}
	_ = requireTag(t, "SemanticsSortKeyTypeMismatch", func() {
		semantics.SemanticsSortKeyCompare(semantics.NewOrdinalSortKey(0), other)
	})
}

func TestOrdinalSortKeyComparison(t *testing.T) {
	k := func(order float64, name ...string) semantics.SemanticsSortKey {
		return semantics.NewOrdinalSortKey(order, name...)
	}
	tests := []struct {
		name string
		a, b semantics.SemanticsSortKey
		want int
	}{
		{"0,0", k(0), k(0), 0},
		{"0,1", k(0), k(1), -1},
		{"1,0", k(1), k(0), 1},
		{"1,1", k(1), k(1), 0},
		{"a0,a0", k(0, "a"), k(0, "a"), 0},
		{"a0,a1", k(0, "a"), k(1, "a"), -1},
		{"a1,a0", k(1, "a"), k(0, "a"), 1},
		{"a1,a1", k(1, "a"), k(1, "a"), 0},
		// Different names: the unnamed key is first, then by name.
		{"0,bar0", k(0), k(0, "bar"), -1},
		{"0,bar1", k(0), k(1, "bar"), -1},
		{"1,bar0", k(1), k(0, "bar"), -1},
		{"1,bar1", k(1), k(1, "bar"), -1},
		{"foo0,0", k(0, "foo"), k(0), 1},
		{"foo0,1", k(0, "foo"), k(1), 1},
		{"foo1,0", k(1, "foo"), k(0), 1},
		{"foo1,1", k(1, "foo"), k(1), 1},
		{"foo0,bar0", k(0, "foo"), k(0, "bar"), 1},
		{"foo0,bar1", k(0, "foo"), k(1, "bar"), 1},
		{"foo1,bar0", k(1, "foo"), k(0, "bar"), 1},
		{"foo1,bar1", k(1, "foo"), k(1, "bar"), 1},
		{"bar0,foo0", k(0, "bar"), k(0, "foo"), -1},
		{"bar0,foo1", k(0, "bar"), k(1, "foo"), -1},
		{"bar1,foo0", k(1, "bar"), k(0, "foo"), -1},
		{"bar1,foo1", k(1, "bar"), k(1, "foo"), -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := semantics.SemanticsSortKeyCompare(tc.a, tc.b); got != tc.want {
				t.Errorf("compare = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestToStringDeepRespectsChildOrder(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	child1 := nodeWithRect(rect(15, 0, 20, 5))
	child2 := nodeWithRect(rect(10, 0, 15, 5))
	root := nodeWithRect(rect(0, 0, 20, 5))
	root.UpdateWith(nil, []*semantics.SemanticsNode{child1, child2})
	simple := "SemanticsNode#3\n" +
		" │ STALE\n │ owner: null\n │ Rect.fromLTRB(0.0, 0.0, 20.0, 5.0)\n │\n" +
		" ├─SemanticsNode#1\n │   STALE\n │   owner: null\n │   Rect.fromLTRB(15.0, 0.0, 20.0, 5.0)\n │\n" +
		" └─SemanticsNode#2\n     STALE\n     owner: null\n     Rect.fromLTRB(10.0, 0.0, 15.0, 5.0)\n"
	if got := root.ToStringDeep(); got != simple {
		t.Errorf("traversal order:\n%s\nwant:\n%s", got, simple)
	}
	if got := root.ToStringDeep(semantics.SemanticsToStringDeepOpts{ChildOrder: semantics.DebugSemanticsDumpOrderInverseHitTest}); got != simple {
		t.Errorf("inverse hit test order:\n%s\nwant:\n%s", got, simple)
	}

	child3 := nodeWithRect(rect(0, 0, 10, 5))
	child3.UpdateWith(nil, []*semantics.SemanticsNode{nodeWithRect(rect(5, 0, 10, 5)), nodeWithRect(rect(0, 0, 5, 5))})
	rootComplex := nodeWithRect(rect(0, 0, 25, 5))
	rootComplex.UpdateWith(nil, []*semantics.SemanticsNode{child1, child2, child3})
	complexWant := "SemanticsNode#7\n" +
		" │ STALE\n │ owner: null\n │ Rect.fromLTRB(0.0, 0.0, 25.0, 5.0)\n │\n" +
		" ├─SemanticsNode#1\n │   STALE\n │   owner: null\n │   Rect.fromLTRB(15.0, 0.0, 20.0, 5.0)\n │\n" +
		" ├─SemanticsNode#2\n │   STALE\n │   owner: null\n │   Rect.fromLTRB(10.0, 0.0, 15.0, 5.0)\n │\n" +
		" └─SemanticsNode#4\n" +
		"   │ STALE\n   │ owner: null\n   │ Rect.fromLTRB(0.0, 0.0, 10.0, 5.0)\n   │\n" +
		"   ├─SemanticsNode#5\n   │   STALE\n   │   owner: null\n   │   Rect.fromLTRB(5.0, 0.0, 10.0, 5.0)\n   │\n" +
		"   └─SemanticsNode#6\n       STALE\n       owner: null\n       Rect.fromLTRB(0.0, 0.0, 5.0, 5.0)\n"
	if got := rootComplex.ToStringDeep(); got != complexWant {
		t.Errorf("complex traversal order:\n%s\nwant:\n%s", got, complexWant)
	}
	if got := rootComplex.ToStringDeep(semantics.SemanticsToStringDeepOpts{ChildOrder: semantics.DebugSemanticsDumpOrderInverseHitTest}); got != complexWant {
		t.Errorf("complex inverse hit test order:\n%s\nwant:\n%s", got, complexWant)
	}
}

func TestDebugProperties(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	minimal := semantics.NewSemanticsNode()
	if got, want := minimal.ToStringDeep(), "SemanticsNode#1\n   Rect.fromLTRB(0.0, 0.0, 0.0, 0.0)\n   invisible\n"; got != want {
		t.Errorf("minimal:\n%s\nwant:\n%s", got, want)
	}
	hidden := foundation.ToStringDeepOpts{MinLevel: foundation.DiagnosticLevelHidden}
	wantHidden := "SemanticsNode#1\n" +
		"   owner: null\n" +
		"   isMergedIntoParent: false\n" +
		"   mergeAllDescendantsIntoThisNode: false\n" +
		"   Rect.fromLTRB(0.0, 0.0, 0.0, 0.0)\n" +
		"   tags: null\n" +
		"   actions: []\n" +
		"   customActions: []\n" +
		"   flags: []\n" +
		"   invisible\n" +
		"   isHidden: false\n" +
		"   identifier: \"\"\n" +
		"   traversalParentIdentifier: null\n" +
		"   traversalChildIdentifier: null\n" +
		"   label: \"\"\n" +
		"   value: \"\"\n" +
		"   increasedValue: \"\"\n" +
		"   decreasedValue: \"\"\n" +
		"   hint: \"\"\n" +
		"   tooltip: \"\"\n" +
		"   textDirection: null\n" +
		"   sortKey: null\n" +
		"   platformViewId: null\n" +
		"   maxValueLength: null\n" +
		"   currentValueLength: null\n" +
		"   scrollChildren: null\n" +
		"   scrollIndex: null\n" +
		"   scrollExtentMin: null\n" +
		"   scrollPosition: null\n" +
		"   scrollExtentMax: null\n" +
		"   indexInParent: null\n" +
		"   headingLevel: 0\n" +
		"   minValue: null\n" +
		"   maxValue: null\n"
	if got := minimal.ToStringDeep(semantics.SemanticsToStringDeepOpts{ToStringDeepOpts: hidden}); got != wantHidden {
		t.Errorf("minimal hidden:\n%s\nwant:\n%s", got, wantHidden)
	}

	config := semantics.NewSemanticsConfiguration()
	config.SetIsSemanticBoundary(true)
	config.SetIsMergingSemanticsOfDescendants(true)
	config.SetOnScrollUp(noop)
	config.SetOnLongPress(noop)
	config.SetOnShowOnScreen(noop)
	config.SetIsChecked(foundation.Some(false))
	config.SetIsSelected(true)
	config.SetIsButton(true)
	config.SetLabel("Use all the properties")
	config.SetTextDirection(ui.TextDirectionRTL)
	config.SetSortKey(semantics.NewOrdinalSortKey(1))
	all := nodeWithRect(ui.NewRectFromLTWH(50, 10, 20, 30))
	all.SetTransform(ui.NewMatrix4TranslationValues(10, 10, 0))
	all.UpdateWith(config, nil)
	got := all.ToStringDeep()
	for _, want := range []string{
		"SemanticsNode#2\n   STALE\n   owner: null\n   merge boundary ⛔️\n   Rect.fromLTRB(60.0, 20.0, 80.0, 50.0)\n",
		"   actions: longPress, scrollUp, showOnScreen\n",
		"   flags: hasCheckedState, isSelected, isButton, hasSelectedState\n",
		"   label: \"Use all the properties\"\n   textDirection: rtl\n   sortKey: OrdinalSortKey#",
		"(order: 1.0)\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("all properties lack %q:\n%s", want, got)
		}
	}
	wantData := "SemanticsData(" +
		"Rect.fromLTRB(50.0, 10.0, 70.0, 40.0), " +
		"[1.0,0.0,0.0,10.0; 0.0,1.0,0.0,10.0; 0.0,0.0,1.0,0.0; 0.0,0.0,0.0,1.0], " +
		"actions: [longPress, scrollUp, showOnScreen], " +
		"flags: [hasCheckedState, isSelected, isButton, hasSelectedState], " +
		"label: \"Use all the properties\", textDirection: rtl)"
	if got := all.GetSemanticsData().String(); got != wantData {
		t.Errorf("data = %s\nwant %s", got, wantData)
	}

	scaled := nodeWithRect(ui.NewRectFromLTWH(50, 10, 20, 30))
	scaled.SetTransform(ui.NewMatrix4Diagonal3Values(10, 10, 1))
	if got, want := scaled.ToStringDeep(), "SemanticsNode#3\n   STALE\n   owner: null\n   Rect.fromLTRB(50.0, 10.0, 70.0, 40.0) scaled by 10.0x\n"; got != want {
		t.Errorf("scaled:\n%s\nwant:\n%s", got, want)
	}
	wantScaled := "SemanticsData(Rect.fromLTRB(50.0, 10.0, 70.0, 40.0), [10.0,0.0,0.0,0.0; 0.0,10.0,0.0,0.0; 0.0,0.0,1.0,0.0; 0.0,0.0,0.0,1.0])"
	if got := scaled.GetSemanticsData().String(); got != wantScaled {
		t.Errorf("scaled data = %s\nwant %s", got, wantScaled)
	}
}

func TestBlockedActionsDebugProperties(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	config := semantics.NewSemanticsConfiguration()
	config.IsBlockingUserActions = true
	config.SetOnScrollUp(noop)
	config.SetOnLongPress(noop)
	config.SetOnShowOnScreen(noop)
	config.SetOnDidGainAccessibilityFocus(noop)
	blocked := nodeWithRect(ui.NewRectFromLTWH(50, 10, 20, 30))
	blocked.SetTransform(ui.NewMatrix4TranslationValues(10, 10, 0))
	blocked.UpdateWith(config, nil)
	want := "SemanticsNode#1\n" +
		"   STALE\n" +
		"   owner: null\n" +
		"   Rect.fromLTRB(60.0, 20.0, 80.0, 50.0)\n" +
		"   actions: didGainAccessibilityFocus, longPress🚫️, scrollUp🚫️,\n" +
		"     showOnScreen🚫️\n"
	if got := blocked.ToStringDeep(); got != want {
		t.Errorf("blocked:\n%s\nwant:\n%s", got, want)
	}
}

func TestValidationResultDebugProperties(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	config := semantics.NewSemanticsConfiguration()
	config.SetValidationResult(ui.SemanticsValidationResultValid)
	node := semantics.NewSemanticsNode()
	node.UpdateWith(config, nil)
	want := "SemanticsNode#1\n   STALE\n   owner: null\n   Rect.fromLTRB(0.0, 0.0, 0.0, 0.0)\n   invisible\n   validationResult: valid\n"
	if got := node.ToStringDeep(); got != want {
		t.Errorf("validation result:\n%s\nwant:\n%s", got, want)
	}
}

func TestCustomActionsDebugProperties(t *testing.T) {
	skipUnlessDebug(t)
	semantics.DebugResetSemanticsIDCounter()
	semantics.ResetCustomSemanticsActionsForTests()
	config := semantics.NewSemanticsConfiguration()
	config.SetCustomSemanticsActions(map[semantics.CustomSemanticsAction]semantics.VoidCallback{
		semantics.NewCustomSemanticsAction("action1"): noop,
		semantics.NewCustomSemanticsAction("action2"): noop,
		semantics.NewCustomSemanticsAction("action3"): noop,
	})
	node := semantics.NewSemanticsNode()
	node.UpdateWith(config, nil)
	hidden := semantics.SemanticsToStringDeepOpts{ToStringDeepOpts: foundation.ToStringDeepOpts{MinLevel: foundation.DiagnosticLevelHidden}}
	got := node.ToStringDeep(hidden)
	for _, want := range []string{"   actions: customAction\n", "   customActions: action1, action2, action3\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("custom actions dump lacks %q:\n%s", want, got)
		}
	}
}

func TestAttributedStringConcat(t *testing.T) {
	string1 := semantics.NewAttributedString("string1", ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 4)))
	string2 := semantics.NewAttributedString("string2", ui.NewLocaleStringAttribute(ui.NewTextRange(0, 4), ui.NewLocale("es", "MX")))
	result := string1.Add(string2)
	if result.String != "string1string2" || len(result.Attributes) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if r := result.Attributes[0].Range(); r != ui.NewTextRange(0, 4) {
		t.Errorf("first range = %v", r)
	}
	if _, ok := result.Attributes[0].(ui.SpellOutStringAttribute); !ok {
		t.Errorf("first attribute is %T", result.Attributes[0])
	}
	want := "AttributedString('string1string2', attributes: [SpellOutStringAttribute(TextRange(start: 0, end: 4)), LocaleStringAttribute(TextRange(start: 7, end: 11), es-MX)])"
	if got := fmt.Sprintf("%#v", result); got != want {
		t.Errorf("result = %s\nwant %s", got, want)
	}
}

func TestSemanticsIDDoesNotRepeat(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	owner := newOwner()
	const expectID = 1400
	var toRemove *semantics.SemanticsNode
	for i := 0; i < 1<<16-1; i++ {
		node := semantics.NewSemanticsNode()
		node.Attach(owner)
		if node.ID() == expectID {
			toRemove = node
		}
	}
	toRemove.Detach()
	newNode := semantics.NewSemanticsNode()
	newNode.Attach(owner)
	// The ID is reused.
	if newNode.ID() != expectID {
		t.Errorf("id = %d, want %d", newNode.ID(), expectID)
	}
}

func TestPerformActionAtHitTestsMergedNode(t *testing.T) {
	tapped := false
	owner := newOwner()
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rect(0, 0, 10, 10))
	merged := nodeWithRect(rect(0, 0, 10, 10))
	mergeConfig := boundaryConfig(true)
	mergeConfig.SetOnTap(func() { tapped = true })
	rootConfig := semantics.NewSemanticsConfiguration()
	rootConfig.SetIsSemanticBoundary(true)
	merged.UpdateWith(mergeConfig, nil)
	root.UpdateWith(rootConfig, []*semantics.SemanticsNode{merged})
	owner.PerformActionAt(ui.Offset{Dx: 5, Dy: 5}, ui.SemanticsActionTap, nil)
	if !tapped {
		t.Error("the tap did not reach the merged node")
	}
}

func TestTagsShowUpInDebugProperties(t *testing.T) {
	skipUnlessDebug(t)
	node := semantics.NewSemanticsNode()
	node.Tags = map[semantics.SemanticsTag]struct{}{semantics.NewSemanticsTag("RenderViewport.twoPane"): {}}
	if got := node.ToStringDeep(); !strings.Contains(got, "\n   tags: RenderViewport.twoPane\n") {
		t.Errorf("dump lacks the tag:\n%s", got)
	}
}

func TestConfigurationGettersAndSetters(t *testing.T) {
	config := semantics.NewSemanticsConfiguration()
	action := semantics.NewCustomSemanticsAction("test")
	if config.IsSemanticBoundary() || config.IsButton() || config.IsLink() || config.IsMergingSemanticsOfDescendants() ||
		config.IsSelected() || config.IsBlockingSemanticsOfPreviouslyPaintedNodes || config.IsTextField() {
		t.Error("a new configuration has flags set")
	}
	if config.IsEnabled().IsSome() || config.IsChecked().IsSome() || config.IsFocused().IsSome() {
		t.Error("a new configuration has tristate flags set")
	}
	if config.OnShowOnScreen() != nil || config.OnScrollDown() != nil || config.OnScrollUp() != nil || config.OnScrollLeft() != nil ||
		config.OnScrollRight() != nil || config.OnScrollToOffset() != nil || config.OnLongPress() != nil || config.OnDecrease() != nil ||
		config.OnIncrease() != nil || config.OnMoveCursorForwardByCharacter() != nil || config.OnMoveCursorBackwardByCharacter() != nil ||
		config.OnTap() != nil || config.CustomSemanticsActions()[action] != nil {
		t.Error("a new configuration has handlers")
	}

	config.SetIsSemanticBoundary(true)
	config.SetIsButton(true)
	config.SetIsLink(true)
	config.SetIsMergingSemanticsOfDescendants(true)
	config.SetIsEnabled(foundation.Some(true))
	config.SetIsChecked(foundation.Some(true))
	config.SetIsSelected(true)
	config.IsBlockingSemanticsOfPreviouslyPaintedNodes = true
	config.SetIsFocused(foundation.Some(true))
	config.SetIsTextField(true)

	calls := map[string]int{}
	counter := func(name string) func() { return func() { calls[name]++ } }
	config.SetOnShowOnScreen(counter("showOnScreen"))
	config.SetOnScrollDown(counter("scrollDown"))
	config.SetOnScrollUp(counter("scrollUp"))
	config.SetOnScrollLeft(counter("scrollLeft"))
	config.SetOnScrollRight(counter("scrollRight"))
	config.SetOnScrollToOffset(func(ui.Offset) { calls["scrollToOffset"]++ })
	config.SetOnLongPress(counter("longPress"))
	config.SetOnDecrease(counter("decrease"))
	config.SetOnIncrease(counter("increase"))
	config.SetOnMoveCursorForwardByCharacter(func(bool) { calls["forward"]++ })
	config.SetOnMoveCursorBackwardByCharacter(func(bool) { calls["backward"]++ })
	config.SetOnTap(counter("tap"))
	customActions := config.CustomSemanticsActions()
	customActions[action] = counter("custom")

	if !config.IsSemanticBoundary() || !config.IsButton() || !config.IsLink() || !config.IsMergingSemanticsOfDescendants() ||
		config.IsEnabled() != foundation.Some(true) || config.IsChecked() != foundation.Some(true) || !config.IsSelected() ||
		!config.IsBlockingSemanticsOfPreviouslyPaintedNodes || config.IsFocused() != foundation.Some(true) || !config.IsTextField() {
		t.Error("the flags were not set")
	}
	config.OnShowOnScreen()()
	config.OnScrollDown()()
	config.OnScrollUp()()
	config.OnScrollLeft()()
	config.OnScrollRight()()
	config.OnScrollToOffset()(ui.Offset{})
	config.OnLongPress()()
	config.OnDecrease()()
	config.OnIncrease()()
	config.OnMoveCursorForwardByCharacter()(false)
	config.OnMoveCursorBackwardByCharacter()(false)
	config.OnTap()()
	config.CustomSemanticsActions()[action]()
	for _, name := range []string{"showOnScreen", "scrollDown", "scrollUp", "scrollLeft", "scrollRight", "scrollToOffset", "longPress",
		"decrease", "increase", "forward", "backward", "tap", "custom"} {
		if calls[name] != 1 {
			t.Errorf("handler %s was called %d times, want 1", name, calls[name])
		}
	}
}

func TestConfigurationCopyPreservesHitTestBehavior(t *testing.T) {
	config := semantics.NewSemanticsConfiguration()
	config.SetIsSemanticBoundary(true)
	config.SetLabel("test")
	config.SetHitTestBehavior(ui.SemanticsHitTestBehaviorOpaque)
	cp := config.Copy()
	if cp.HitTestBehavior() != ui.SemanticsHitTestBehaviorOpaque || !cp.IsSemanticBoundary() || cp.Label() != "test" {
		t.Errorf("copy = %v %v %q", cp.HitTestBehavior(), cp.IsSemanticBoundary(), cp.Label())
	}
	for _, b := range []ui.SemanticsHitTestBehavior{ui.SemanticsHitTestBehaviorDefer, ui.SemanticsHitTestBehaviorOpaque, ui.SemanticsHitTestBehaviorTransparent} {
		c := semantics.NewSemanticsConfiguration()
		c.SetHitTestBehavior(b)
		if got := c.Copy().HitTestBehavior(); got != b {
			t.Errorf("copy of %v = %v", b, got)
		}
	}
}

func TestIndexInParentAppearsInStringOutput(t *testing.T) {
	skipUnlessDebug(t)
	node := semantics.NewSemanticsNode()
	node.IndexInParent = foundation.Some(10)
	if got := node.String(); !strings.Contains(got, "indexInParent: 10") {
		t.Errorf("node = %s", got)
	}
}

func TestSemanticsLabelBuilder(t *testing.T) {
	build := func(opts semantics.SemanticsLabelBuilderOpts, parts ...string) string {
		b := semantics.NewSemanticsLabelBuilder(opts)
		for _, p := range parts {
			b.AddPart(p)
		}
		return b.Build()
	}
	ltr := semantics.SemanticsLabelBuilderOpts{TextDirection: ui.TextDirectionLTR}
	_ = ltr
	tests := []struct {
		name  string
		opts  semantics.SemanticsLabelBuilderOpts
		parts []string
		want  string
	}{
		{"default separator", semantics.SemanticsLabelBuilderOpts{}, []string{"Hello", "world"}, "Hello world"},
		{"custom separator", semantics.SemanticsLabelBuilderOpts{Separator: foundation.Some(", ")}, []string{"One", "Two", "Three"}, "One, Two, Three"},
		{"empty separator", semantics.SemanticsLabelBuilderOpts{Separator: foundation.Some("")}, []string{"Hello", "World"}, "HelloWorld"},
		{"ignores empty parts", semantics.SemanticsLabelBuilderOpts{}, []string{"Hello", "", "world"}, "Hello world"},
		{"single part", semantics.SemanticsLabelBuilderOpts{}, []string{"Single"}, "Single"},
		{"empty builder", semantics.SemanticsLabelBuilderOpts{}, nil, ""},
		{"pipe separator", semantics.SemanticsLabelBuilderOpts{Separator: foundation.Some(" | ")}, []string{"first", "second", "third"}, "first | second | third"},
		{"unicode", semantics.SemanticsLabelBuilderOpts{}, []string{"Emoji: 😀🎉", "Math: ∑∆π", "Currency: €£¥"}, "Emoji: 😀🎉 Math: ∑∆π Currency: €£¥"},
		{"long labels", semantics.SemanticsLabelBuilderOpts{}, []string{strings.Repeat("A", 1000), "short"}, strings.Repeat("A", 1000) + " short"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := build(tc.opts, tc.parts...); got != tc.want {
				t.Errorf("label = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("length and clear", func(t *testing.T) {
		b := semantics.NewSemanticsLabelBuilder()
		if !b.IsEmpty() || b.Length() != 0 {
			t.Fatal("a new builder is not empty")
		}
		b.AddPart("First")
		b.AddPart("batch")
		if b.Build() != "First batch" || b.Length() != 2 {
			t.Errorf("batch 1: %q %d", b.Build(), b.Length())
		}
		b.AddPart("Second")
		b.AddPart("batch")
		if b.Build() != "First batch Second batch" || b.Length() != 4 {
			t.Errorf("batch 2: %q %d", b.Build(), b.Length())
		}
		b.Clear()
		if !b.IsEmpty() || b.Build() != "" {
			t.Errorf("after clear: %q", b.Build())
		}
		for i := 0; i < 100; i++ {
			b.AddPart(fmt.Sprintf("part%d", i))
		}
		if got := b.Build(); !strings.HasPrefix(got, "part0 part1") || !strings.HasSuffix(got, "part98 part99") || b.Length() != 100 {
			t.Errorf("many parts: %q", got)
		}
	})

	t.Run("text direction", func(t *testing.T) {
		const arabic, hebrew = "مرحبا", "שלום"
		dir := func(o semantics.SemanticsLabelBuilderOpts, parts ...directed) string {
			b := semantics.NewSemanticsLabelBuilder(o)
			for _, p := range parts {
				b.AddPart(p.text, p.dir)
			}
			return b.Build()
		}
		l, r := ui.TextDirectionLTR, ui.TextDirectionRTL
		tests := []struct {
			name  string
			opts  semantics.SemanticsLabelBuilderOpts
			parts []directed
			want  string
		}{
			{"no overall direction", semantics.SemanticsLabelBuilderOpts{}, []directed{{"Hello", l}, {arabic, r}}, "Hello " + arabic},
			{"ltr overall", semantics.SemanticsLabelBuilderOpts{TextDirection: l}, []directed{{"Hello", l}, {arabic, r}, {"world", l}},
				"Hello \u202B" + arabic + "\u202C world"},
			{"rtl overall", semantics.SemanticsLabelBuilderOpts{TextDirection: r}, []directed{{arabic, r}, {"Hello", l}},
				arabic + " \u202AHello\u202C"},
			{"same direction", semantics.SemanticsLabelBuilderOpts{TextDirection: l}, []directed{{"Hello", l}, {"world", l}}, "Hello world"},
			{"falls back to overall", semantics.SemanticsLabelBuilderOpts{TextDirection: l}, []directed{{"Hello", 0}, {"world", 0}}, "Hello world"},
			{"multilingual", semantics.SemanticsLabelBuilderOpts{TextDirection: l, Separator: foundation.Some(", ")},
				[]directed{{"Welcome", l}, {arabic, r}, {hebrew, r}, {"to our app", l}},
				"Welcome, \u202B" + arabic + "\u202C, \u202B" + hebrew + "\u202C, to our app"},
		}
		for _, tc := range tests {
			if got := dir(tc.opts, tc.parts...); got != tc.want {
				t.Errorf("%s: label = %q, want %q", tc.name, got, tc.want)
			}
		}
	})
}

type directed struct {
	text string
	dir  ui.TextDirection
}
