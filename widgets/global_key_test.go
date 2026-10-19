package widgets

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// marker is StateMarker of reparent_state_test.dart: a stateful widget with a
// state that remembers a string.
type marker struct {
	StatefulWidget
	child Widget
}

func newMarker(child Widget, opts ...WidgetOpts) *marker {
	return &marker{NewStatefulWidgetBase(opts...), child}
}

func (w *marker) CreateState() State { return &markerState{} }

type markerState struct {
	StateOf[*marker]
	marker string
}

func (s *markerState) Build(BuildContext) Widget {
	if s.Widget().child != nil {
		return s.Widget().child
	}
	return newBox(1, 1)
}

// dummy is DummyWidget of global_keys_duplicated_test.dart.
type dummy struct {
	StatelessWidget
	child Widget
}

func newDummy(child Widget, opts ...WidgetOpts) *dummy {
	return &dummy{NewStatelessWidgetBase(opts...), child}
}

func (w *dummy) Build(BuildContext) Widget {
	if w.child != nil {
		return w.child
	}
	return newBox(0, 0)
}

func TestGlobalKeyControls(t *testing.T) {
	skipUnlessDebug(t)
	type testState struct{ StateOf[*marker] }
	a, b := new(int), new(int)
	t.Run("GlobalKey is equal only to itself", func(t *testing.T) {
		k1, k2 := NewGlobalKey[State](), NewGlobalKey[State]()
		if foundation.Key(k1) == foundation.Key(k2) || foundation.Key(k1) != foundation.Key(k1) {
			t.Error("identity of global keys")
		}
	})
	t.Run("GlobalKey description", func(t *testing.T) {
		got := normalizeHashes(NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "hello"}).String())
		if got != "[GlobalKey#00000 hello]" {
			t.Errorf("String = %q", got)
		}
		if got := normalizeHashes(NewGlobalKey[State]().String()); got != "[GlobalKey#00000]" {
			t.Errorf("String = %q", got)
		}
	})
	t.Run("GlobalObjectKey equality", func(t *testing.T) {
		keyA, keyA2, keyB := NewGlobalObjectKey[State](a), NewGlobalObjectKey[State](a), NewGlobalObjectKey[State](b)
		if foundation.Key(keyA) != foundation.Key(keyA2) || foundation.Key(keyA) == foundation.Key(keyB) {
			t.Error("GlobalObjectKey equality")
		}
		if foundation.Key(NewGlobalObjectKey[State](a)) == foundation.Key(NewGlobalObjectKey[*markerState](a)) {
			t.Error("keys of different state types are equal")
		}
		if keyA.Value() != any(a) {
			t.Error("Value")
		}
	})
	t.Run("GlobalObjectKey toString", func(t *testing.T) {
		tests := []struct {
			key  foundation.Key
			want string
		}{
			{NewGlobalObjectKey[State](1), "[GlobalObjectKey " + foundation.DescribeIdentity(1) + "]"},
			{NewGlobalObjectKey[*markerState](2), "[GlobalObjectKey<markerState> " + foundation.DescribeIdentity(2) + "]"},
		}
		for _, tc := range tests {
			if got := tc.key.String(); got != tc.want {
				t.Errorf("String = %q, want %q", got, tc.want)
			}
		}
		_ = testState{}
	})
	t.Run("GlobalObjectKey needs a comparable value", func(t *testing.T) {
		requireTag(t, "GlobalObjectKeyNotComparable", func() { NewGlobalObjectKey[State]([]int{1}) })
	})
}

func TestGlobalKeyCount(t *testing.T) {
	rt := newRenderTree(t)
	k1, k2 := NewGlobalKey[State](), NewGlobalKey[State]()
	initial := rt.owner.GlobalKeyCount()
	steps := []struct {
		name string
		w    Widget
		want int
	}{
		{"one key", newBox(1, 1, WidgetOpts{Key: k1}), 1},
		{"the same key in a new box", newBox(2, 1, WidgetOpts{Key: k1}), 1},
		{"two keys", newSized(5, 5, newBox(1, 1, WidgetOpts{Key: k2}), WidgetOpts{Key: k1}), 2},
		{"no keys", newBox(1, 1), 0},
	}
	for _, s := range steps {
		rt.pump(s.w)
		if got := rt.owner.GlobalKeyCount() - initial; got != s.want {
			t.Errorf("%s: %d global keys, want %d", s.name, got, s.want)
		}
	}
}

func TestGlobalKeyLookups(t *testing.T) {
	rt := newRenderTree(t)
	key := NewGlobalKey[*markerState]()
	objKey := NewGlobalObjectKey[*markerState]("k")
	if key.CurrentContext() != nil || key.CurrentWidget() != nil || key.CurrentElement() != nil {
		t.Error("a key that was never used has an element")
	}
	if _, ok := key.CurrentState(); ok {
		t.Error("a key that was never used has a state")
	}
	w := newMarker(nil, WidgetOpts{Key: key})
	rt.pump(newColumn(w, newMarker(nil, WidgetOpts{Key: objKey})))
	if key.CurrentWidget() != Widget(w) {
		t.Error("CurrentWidget is not the widget")
	}
	if ctx, ok := key.CurrentContext().(*StatefulElement); !ok || ctx.State() == nil {
		t.Errorf("CurrentContext = %v", key.CurrentContext())
	}
	if s, ok := key.CurrentState(); !ok || s.Widget() != w {
		t.Errorf("CurrentState = %v, %v", s, ok)
	}
	if objKey.CurrentElement() != nil {
		t.Error("a GlobalObjectKey does not know its owner")
	}
	if s, ok := objKey.CurrentStateIn(rt.owner); !ok || s == nil {
		t.Error("CurrentStateIn did not find the state")
	}
	if objKey.CurrentElementIn(rt.owner) == nil || objKey.CurrentElementIn(nil) != nil {
		t.Error("CurrentElementIn")
	}
	// A key on a widget that is not stateful has no state.
	boxKey := NewGlobalKey[*markerState]()
	rt.pump(newBox(1, 1, WidgetOpts{Key: boxKey}))
	if _, ok := boxKey.CurrentState(); ok {
		t.Error("a box has a state")
	}
	if key.CurrentElement() != nil {
		t.Error("a key whose widget left the tree still has an element")
	}
}

func TestGlobalKeyMovesKeepStateAndElement(t *testing.T) {
	// 'can reparent state', 'can reparent state with multichild widgets'
	left, right := NewGlobalKey[*markerState](), NewGlobalKey[*markerState]()
	rt := newRenderTree(t)
	grandchild := newMarker(nil)
	rt.pump(newColumn(
		newSized(5, 5, newMarker(nil, WidgetOpts{Key: left})),
		newSized(5, 5, newMarker(grandchild, WidgetOpts{Key: right})),
	))
	leftState, _ := left.CurrentState()
	leftState.marker = "left"
	rightState, _ := right.CurrentState()
	rightState.marker = "right"
	grandchildState := stateOf[*markerState](t, nil, rt.find(func(w Widget) bool { return w == Widget(grandchild) })[0])
	grandchildState.marker = "grandchild"
	leftElement := left.CurrentElement()

	newGrandchild := newMarker(nil)
	rt.pump(newColumn(
		newSized(5, 5, newMarker(newGrandchild, WidgetOpts{Key: right})),
		newSized(5, 5, newMarker(nil, WidgetOpts{Key: left})),
	))
	if s, _ := left.CurrentState(); s != leftState || leftState.marker != "left" {
		t.Error("the left state was lost")
	}
	if left.CurrentElement() != leftElement {
		t.Error("the left element was replaced")
	}
	if s, _ := right.CurrentState(); s != rightState || rightState.marker != "right" {
		t.Error("the right state was lost")
	}
	newGrandchildState := stateOf[*markerState](t, nil, rt.find(func(w Widget) bool { return w == Widget(newGrandchild) })[0])
	if newGrandchildState != grandchildState || newGrandchildState.marker != "grandchild" {
		t.Error("the state of the grandchild was lost")
	}

	// Only the left one stays in the tree.
	rt.pump(newCenter(newSized(5, 5, newMarker(newBox(1, 1), WidgetOpts{Key: left}))))
	if s, _ := left.CurrentState(); s != leftState || leftState.marker != "left" {
		t.Error("the left state was lost in the third tree")
	}
	if _, ok := right.CurrentState(); ok {
		t.Error("the right state is still there")
	}
	if rt.owner.GlobalKeyCount() != 1 {
		t.Errorf("%d global keys, want 1", rt.owner.GlobalKeyCount())
	}
}

func TestReparentDuringUpdateChildren(t *testing.T) {
	// 'Reparent during update children'
	key := NewGlobalKey[*markerState]()
	rt := newRenderTree(t)
	rt.pump(newColumn(newMarker(nil, WidgetOpts{Key: key}), newBox(100, 100)))
	state, _ := key.CurrentState()
	state.marker = "marked"
	for i, ws := range [][]Widget{
		{newBox(100, 100), newMarker(nil, WidgetOpts{Key: key})},
		{newMarker(nil, WidgetOpts{Key: key}), newBox(100, 100)},
	} {
		rt.pump(newColumn(ws...))
		if s, _ := key.CurrentState(); s != state || state.marker != "marked" {
			t.Fatalf("step %d: the state was lost", i)
		}
	}
}

func TestReparentToChildDuringUpdateChildren(t *testing.T) {
	// 'Reparent to child during update children'
	key := NewGlobalKey[*markerState]()
	rt := newRenderTree(t)
	rt.pump(newColumn(newBox(100, 100), newMarker(nil, WidgetOpts{Key: key}), newBox(100, 100)))
	state, _ := key.CurrentState()
	state.marker = "marked"
	steps := [][]Widget{
		{newSized(100, 100, newMarker(nil, WidgetOpts{Key: key})), newBox(100, 100)},
		{newBox(100, 100), newMarker(nil, WidgetOpts{Key: key}), newBox(100, 100)},
		{newBox(100, 100), newSized(100, 100, newMarker(nil, WidgetOpts{Key: key}))},
		{newBox(100, 100), newMarker(nil, WidgetOpts{Key: key}), newBox(100, 100)},
	}
	for i, ws := range steps {
		rt.pump(newColumn(ws...))
		if s, _ := key.CurrentState(); s != state || state.marker != "marked" {
			t.Fatalf("step %d: the state was lost", i)
		}
		if err := rt.takeError(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
}

func TestReparentingWithMultipleMoves(t *testing.T) {
	// 'Reparenting with multiple moves'
	k1, k2, k3 := NewGlobalKey[*markerState](), NewGlobalKey[*markerState](), NewGlobalKey[*markerState]()
	rt := newRenderTree(t)
	rt.pump(newColumn(
		newMarker(newMarker(newMarker(newMarker(newBox(100, 10)), WidgetOpts{Key: k3}), WidgetOpts{Key: k2}), WidgetOpts{Key: k1}),
	))
	rt.pump(newColumn(
		newMarker(newMarker(newBox(100, 10)), WidgetOpts{Key: k2}),
		newMarker(newMarker(newMarker(newBox(100, 10)), WidgetOpts{Key: k3}), WidgetOpts{Key: k1}),
	))
	if err := rt.takeError(); err != nil {
		t.Fatal(err)
	}
	if rt.owner.GlobalKeyCount() != 3 {
		t.Errorf("%d global keys, want 3", rt.owner.GlobalKeyCount())
	}
}

func TestDeactivateImpliesBuild(t *testing.T) {
	// 'Deactivate implies build'
	var log []string
	key := NewGlobalKey[State]()
	logger := newSpy(nil, WidgetOpts{Key: key})
	logger.onBuild = func(BuildContext) { log = append(log, "build") }
	logger.onDeactivate = func(BuildContext) { log = append(log, "deactivate") }
	tr := newTree(t)
	tr.pump(newWrapper(logger, WidgetOpts{Key: foundation.NewUniqueKey()}))
	if want := []string{"build"}; !slices.Equal(log, want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	tr.pump(newWrapper(logger, WidgetOpts{Key: foundation.NewUniqueKey()}))
	if want := []string{"build", "deactivate", "build"}; !slices.Equal(log, want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	log = nil
	tr.pumpFrame()
	if len(log) != 0 {
		t.Errorf("a frame with nothing dirty built %v", log)
	}
}

func TestDeactivateAndActivateWithGlobalKey(t *testing.T) {
	// The keyed half of 'Deactivate and activate are called correctly'.
	var log []string
	key := NewGlobalKey[State]()
	mk := func() *spy {
		w := newSpy(nil, WidgetOpts{Key: key})
		w.onInitState = func(BuildContext) { log = append(log, "initState") }
		w.onDidUpdateWidget = func(BuildContext) { log = append(log, "didUpdateWidget") }
		w.onDeactivate = func(BuildContext) { log = append(log, "deactivate") }
		w.onActivate = func(BuildContext) { log = append(log, "activate") }
		w.onBuild = func(BuildContext) { log = append(log, "build") }
		w.onDispose = func(BuildContext) { log = append(log, "dispose") }
		return w
	}
	tr := newTree(t)
	steps := []struct {
		name string
		do   func()
		want []string
	}{
		{"mount", func() { tr.pump(mk()) }, []string{"initState", "build"}},
		{"move below a wrapper", func() { tr.pump(newWrapper(mk())) }, []string{"deactivate", "activate", "didUpdateWidget", "build"}},
		{"remove", func() { tr.pump(newLeaf()) }, []string{"deactivate", "dispose"}},
	}
	for _, s := range steps {
		log = nil
		s.do()
		if !slices.Equal(log, s.want) {
			t.Errorf("%s: log = %v, want %v", s.name, log, s.want)
		}
	}
}

func TestGlobalKeyMovesDoNotReportErrors(t *testing.T) {
	// Cases 14 and 16 of framework_test.dart, 'moving during build', and the
	// correct cases 1 and 2.
	key := NewGlobalKey[State]()
	k := func(n int) WidgetOpts { return WidgetOpts{Key: foundation.NewValueKey(n)} }
	tests := []struct {
		name     string
		old, new []Widget
	}{
		{"moving during build - before",
			[]Widget{newBox(1, 1, WidgetOpts{Key: key}), newSized(2, 2, nil, k(0)), newSized(3, 3, nil, k(1))},
			[]Widget{newSized(2, 2, nil, k(0)), newSized(3, 3, newBox(1, 1, WidgetOpts{Key: key}), k(1))}},
		{"moving during build - after",
			[]Widget{newSized(2, 2, nil, k(0)), newSized(3, 3, nil, k(1)), newBox(1, 1, WidgetOpts{Key: key})},
			[]Widget{newSized(2, 2, nil, k(0)), newSized(3, 3, newBox(1, 1, WidgetOpts{Key: key}), k(1))}},
		{"from one parent to the next",
			[]Widget{newSized(2, 2, newBox(1, 1, WidgetOpts{Key: key}), k(1)), newSized(3, 3, nil, k(2))},
			[]Widget{newSized(2, 2, nil, k(1)), newSized(3, 3, newBox(1, 1, WidgetOpts{Key: key}), k(2))}},
		{"from the second parent to the first",
			[]Widget{newSized(2, 2, nil, k(1)), newSized(3, 3, newBox(1, 1, WidgetOpts{Key: key}), k(2))},
			[]Widget{newSized(2, 2, newBox(1, 1, WidgetOpts{Key: key}), k(1)), newSized(3, 3, nil, k(2))}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := newRenderTree(t)
			rt.pump(newColumn(tc.old...))
			before := key.CurrentElement()
			rt.pump(newColumn(tc.new...))
			if err := rt.takeError(); err != nil {
				t.Fatalf("error: %v", err)
			}
			if after := key.CurrentElement(); after != before || after.AsElement().lifecycle != lifecycleActive {
				t.Errorf("the keyed element was not moved: %v -> %v", before, after)
			}
		})
	}
}

func TestGlobalKeyDuplication(t *testing.T) {
	skipUnlessDebug(t)
	k := func(n int) WidgetOpts { return WidgetOpts{Key: foundation.NewValueKey(n)} }
	label := func(l string) *GlobalKey[State] { return NewGlobalKey[State](GlobalKeyOpts{DebugLabel: l}) }

	// 'GlobalKey duplication 1 - double appearance': two widgets of different
	// types with one key under different parents.
	t.Run("1 - double appearance", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		rt.pump(newColumn(
			newWrapper(newBox(1, 1, WidgetOpts{Key: key}), k(1)),
			newWrapper(newSized(2, 2, nil, WidgetOpts{Key: key}), k(2)),
		))
		appErr := requireErrorTag(t, rt.takeError(), "GlobalKeyUsedByMultipleParents")
		msg := normalizeHashes(appErr.Message)
		for _, want := range []string{
			"Multiple widgets used the same GlobalKey.\nThe key [GlobalKey#00000 problematic] was used by multiple widgets. The parents of those widgets were:\n",
			"- wrapper-[<1>]\n- wrapper-[<2>]\n",
			"A GlobalKey can only be specified on one widget at a time in the widget tree.",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("message does not contain %q:\n%s", want, msg)
			}
		}
	})

	t.Run("3 - the same key twice in one list", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		rt.pump(newColumn(newSized(1, 1, nil, WidgetOpts{Key: key})))
		requireTag(t, "ChildrenHaveDuplicateKeys", func() {
			rt.pump(newColumn(newBox(1, 1, WidgetOpts{Key: key}), newSized(1, 1, nil, WidgetOpts{Key: key})))
		})
	})

	t.Run("6 - the same key and type twice in one list", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		rt.pump(newColumn(newBox(1, 1, WidgetOpts{Key: key})))
		requireTag(t, "ChildrenHaveDuplicateKeys", func() {
			rt.pump(newColumn(newBox(1, 1, WidgetOpts{Key: key}), newBox(1, 1, WidgetOpts{Key: key})))
		})
	})

	t.Run("7 - appearing later", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		rt.pump(newColumn(newSized(1, 1, newBox(1, 1, WidgetOpts{Key: key}), k(1)), newSized(2, 2, nil, k(2))))
		rt.pump(newColumn(newSized(1, 1, newBox(1, 1, WidgetOpts{Key: key}), k(1)), newSized(2, 2, newBox(1, 1, WidgetOpts{Key: key}), k(2))))
		_ = requireErrorTag(t, rt.takeError(), "GlobalKeyUsedByMultipleParents")
	})

	t.Run("8 - appearing earlier", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		rt.pump(newColumn(newSized(1, 1, nil, k(1)), newSized(2, 2, newBox(1, 1, WidgetOpts{Key: key}), k(2))))
		rt.pump(newColumn(newSized(1, 1, newBox(1, 1, WidgetOpts{Key: key}), k(1)), newSized(2, 2, newBox(1, 1, WidgetOpts{Key: key}), k(2))))
		_ = requireErrorTag(t, rt.takeError(), "GlobalKeyUsedByMultipleParents")
	})

	t.Run("11 - double sibling appearance", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		requireTag(t, "ChildrenHaveDuplicateKeys", func() {
			rt.pump(newColumn(newBox(1, 1, WidgetOpts{Key: key}), newBox(1, 1, WidgetOpts{Key: key})))
		})
	})

	t.Run("15 - duplicating during build - before", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		rt.pump(newColumn(newBox(1, 1, WidgetOpts{Key: key}), newSized(2, 2, nil, k(0)), newSized(3, 3, nil, k(1))))
		rt.pump(newColumn(newBox(1, 1, WidgetOpts{Key: key}), newSized(2, 2, nil, k(0)), newSized(3, 3, newBox(1, 1, WidgetOpts{Key: key}), k(1))))
		_ = requireErrorTag(t, rt.takeError(), "GlobalKeyUsedByMultipleParents")
	})

	t.Run("17 - duplicating during build - after", func(t *testing.T) {
		key := label("problematic")
		rt := newRenderTree(t)
		rt.pump(newColumn(newSized(2, 2, nil, k(0)), newSized(3, 3, nil, k(1)), newBox(1, 1, WidgetOpts{Key: key})))
		rt.pump(newColumn(newSized(2, 2, nil, k(0)), newSized(3, 3, newBox(1, 1, WidgetOpts{Key: key}), k(1)), newBox(1, 1, WidgetOpts{Key: key})))
		if len(rt.errs) != 1 {
			t.Fatalf("%d errors reported, want 1: %v", len(rt.errs), rt.errs)
		}
	})

	// Cases of global_keys_duplicated_test.dart.
	obj := NewGlobalObjectKey[State](0)
	t.Run("children of one node", func(t *testing.T) {
		rt := newRenderTree(t)
		v := panicOf(func() { rt.pump(newColumn(newDummy(nil, WidgetOpts{Key: obj}), newDummy(nil, WidgetOpts{Key: obj}))) })
		appErr := requireErrorTagFromPanic(t, v, "ChildrenHaveDuplicateKeys")
		if !strings.HasPrefix(appErr.Message, "Duplicate keys found.\n") || !strings.Contains(appErr.Message, "column") ||
			!strings.Contains(appErr.Message, obj.String()) {
			t.Errorf("message = %q", appErr.Message)
		}
	})

	t.Run("children of two nodes - A", func(t *testing.T) {
		rt := newRenderTree(t)
		// The widgets are constants in Flutter: the same object twice, which is
		// why the two parents have the same description.
		same := newDummy(newDummy(nil, WidgetOpts{Key: obj}))
		rt.pump(newColumn(same, same))
		appErr := requireErrorTag(t, rt.takeError(), "GlobalKeyUsedByMultipleParents")
		for _, want := range []string{"Multiple widgets used the same GlobalKey.\n", "different widgets that both had the following description", "dummy", obj.String()} {
			if !strings.Contains(normalizeHashes(appErr.Message), normalizeHashes(want)) {
				t.Errorf("message does not contain %q:\n%s", want, appErr.Message)
			}
		}
		if !strings.HasSuffix(appErr.Message, "\nA GlobalKey can only be specified on one widget at a time in the widget tree.") {
			t.Errorf("message = %q", appErr.Message)
		}
	})

	t.Run("children of two different nodes - B", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newColumn(newDummy(newDummy(nil, WidgetOpts{Key: obj})), newDummy(newDummy(nil, WidgetOpts{Key: obj}), WidgetOpts{Key: foundation.NewKey("x")})))
		appErr := requireErrorTag(t, rt.takeError(), "GlobalKeyUsedByMultipleParents")
		msg := normalizeHashes(appErr.Message)
		if strings.Contains(msg, "different widgets that both had the following description") {
			t.Errorf("message names one description for two parents:\n%s", msg)
		}
		for _, want := range []string{"- dummy\n", "- dummy-[<'x'>]\n", obj.String()} {
			if !strings.Contains(msg, normalizeHashes(want)) {
				t.Errorf("message does not contain %q:\n%s", want, msg)
			}
		}
	})

	t.Run("children of two nodes - C", func(t *testing.T) {
		var nestedSetState func(func())
		flag := false
		rt := newRenderTree(t)
		rt.pump(newColumn(
			newDummy(newDummy(nil, WidgetOpts{Key: obj})),
			newDummy(newBuilder(func(_ BuildContext, set func(func())) Widget {
				nestedSetState = set
				if flag {
					return newDummy(nil, WidgetOpts{Key: obj})
				}
				return newDummy(nil)
			})),
		))
		nestedSetState(func() { flag = true })
		rt.pumpFrame()
		appErr := requireErrorTag(t, rt.takeError(), "GlobalKeyDuplicateDetected")
		msg := normalizeHashes(strings.ReplaceAll(appErr.Message, "\n", " "))
		for _, want := range []string{
			"Duplicate GlobalKey detected in widget tree.",
			"The following GlobalKey was specified multiple times",
			"This was determined by noticing that after the widget with the above global key was moved out of its previous parent, that previous parent never updated during this frame, meaning that it either did not update at all or updated before the widget was moved, in either case implying that it still thinks that it should have a child with that global key.",
			obj.String(), "dummy",
		} {
			if !strings.Contains(msg, normalizeHashes(want)) {
				t.Errorf("message does not contain %q:\n%s", want, msg)
			}
		}
	})
}

// swapKey is SwapKeyWidget of framework_test.dart: a stateful widget that builds
// a leaf box and can change the key of it.
type swapKey struct {
	StatefulWidget
	childKey foundation.Key
}

func (w *swapKey) CreateState() State { return &swapKeyState{key: w.childKey} }

type swapKeyState struct {
	StateOf[*swapKey]
	key foundation.Key
}

func (s *swapKeyState) Swap(k foundation.Key) { s.SetState(func() { s.key = k }) }
func (s *swapKeyState) Build(BuildContext) Widget {
	return newBox(1, 1, WidgetOpts{Key: s.key})
}

func TestGlobalKeyDuplicationInSubtreeBuild(t *testing.T) {
	skipUnlessDebug(t)
	k := func(n int) WidgetOpts { return WidgetOpts{Key: foundation.NewValueKey(n)} }
	t.Run("18 - same type", func(t *testing.T) {
		key := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "problematic"})
		rt := newRenderTree(t)
		swap := &swapKey{NewStatefulWidgetBase(), foundation.NewValueKey(0)}
		rt.pump(newColumn(swap, newBox(1, 1, k(1)), newBox(1, 1, WidgetOpts{Key: key})))
		stateOf[*swapKeyState](t, nil, renderElementOfType[*swapKey](t, rt)).Swap(key)
		rt.pumpFrame()
		appErr := requireErrorTag(t, rt.takeError(), "GlobalKeyDuplicateDetected")
		msg := normalizeHashes(appErr.Message)
		for _, want := range []string{
			"Duplicate GlobalKey detected in widget tree.\nThe following GlobalKey was specified multiple times in the widget tree.",
			"- [GlobalKey#00000 problematic]\n",
			"The specific parent that did not update after having one or more children forcibly removed due to GlobalKey reparenting is:\n- column(",
			"A GlobalKey can only be specified on one widget at a time in the widget tree.",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("message does not contain %q:\n%s", want, msg)
			}
		}
	})
	t.Run("19 - different types", func(t *testing.T) {
		key := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "problematic"})
		rt := newRenderTree(t)
		swap := &swapKey{NewStatefulWidgetBase(), foundation.NewValueKey(0)}
		rt.pump(newColumn(swap, newBox(1, 1, k(1)), newSized(2, 2, nil, WidgetOpts{Key: key})))
		stateOf[*swapKeyState](t, nil, renderElementOfType[*swapKey](t, rt)).Swap(key)
		rt.pumpFrame()
		appErr := requireErrorTag(t, rt.takeError(), "GlobalKeyUsedByMultipleWidgets")
		msg := normalizeHashes(appErr.Message)
		for _, want := range []string{
			"Multiple widgets used the same GlobalKey.\nThe key [GlobalKey#00000 problematic] was used by 2 widgets:\n",
			"- sized-[GlobalKey#00000 problematic]",
			"- box-[GlobalKey#00000 problematic]",
			"A GlobalKey can only be specified on one widget at a time in the widget tree.",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("message does not contain %q:\n%s", want, msg)
			}
		}
	})
}

func TestGlobalKeyUsedTwiceInChildList(t *testing.T) {
	skipUnlessDebug(t)
	key := NewGlobalKey[State]()
	tr := newTree(t)
	// The component-only harness has no duplicate key check for its multi
	// element, so the retaking code finds the first child as the owner.
	requireTag(t, "GlobalKeyUsedTwiceInChildList", func() {
		tr.pump(newMulti(newLeaf(WidgetOpts{Key: key}), newLeaf(WidgetOpts{Key: key})))
	})
}

func TestFinalizeTreeReportsToTheOwner(t *testing.T) {
	skipUnlessDebug(t)
	// Without OnError the errors found in FinalizeTree panic.
	owner := NewBuildOwner()
	key := NewGlobalKey[State]()
	root := &rootElement{}
	InitElement(&root.ElementBase, root, &rootWidget{child: newMulti(newLeaf(WidgetOpts{Key: key}))})
	root.AssignOwner(owner)
	owner.BuildScope(root, func() { root.Mount(nil, nil) })
	owner.FinalizeTree()
	owner.BuildScope(root, func() {
		root.Update(&rootWidget{child: newMulti(newSized2(key), newLeaf(WidgetOpts{Key: key}))})
	})
	requireTag(t, "GlobalKeyUsedByMultipleWidgets", owner.FinalizeTree)

	// A panic that is not an error is wrapped.
	owner = NewBuildOwner()
	var got error
	owner.OnError = func(err error) { got = err }
	owner.reportPanic("boom")
	_ = requireErrorTag(t, got, "UnexpectedPanic")
}

// newSized2 is a leaf-less stand-in of another type that has the key.
func newSized2(key foundation.Key) Widget { return &otherLeaf{NewWidgetBase(WidgetOpts{Key: key})} }
