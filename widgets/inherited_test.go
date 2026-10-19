package widgets

import (
	"slices"
	"strconv"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// testInherited is TestInherited of inherited_test.dart.
type testInherited struct {
	InheritedWidget
	shouldNotify bool
}

func newTestInherited(shouldNotify bool, child Widget, opts ...WidgetOpts) *testInherited {
	return &testInherited{NewInheritedWidgetBase(child, opts...), shouldNotify}
}

func (w *testInherited) UpdateShouldNotify(Widget) bool { return w.shouldNotify }

// valueInherited is ValueInherited of inherited_test.dart.
type valueInherited struct {
	InheritedWidget
	value int
}

func newValueInherited(value int, child Widget, opts ...WidgetOpts) *valueInherited {
	return &valueInherited{NewInheritedWidgetBase(child, opts...), value}
}

func (w *valueInherited) UpdateShouldNotify(old Widget) bool {
	return w.value != old.(*valueInherited).value
}

// changeNotifierInherited is ChangeNotifierInherited of inherited_test.dart.
type changeNotifierInherited struct{ InheritedNotifier }

func newChangeNotifierInherited(n foundation.Listenable, child Widget, opts ...WidgetOpts) *changeNotifierInherited {
	return &changeNotifierInherited{NewInheritedNotifierBase(n, child, opts...)}
}

// statelessBuilder is Builder: a stateless widget with a build function.
type statelessBuilder struct {
	StatelessWidget
	build func(bc BuildContext) Widget
}

func (w *statelessBuilder) Build(bc BuildContext) Widget { return w.build(bc) }

// builderOf makes a Builder.
func builderOf(f func(bc BuildContext) Widget, opts ...WidgetOpts) *statelessBuilder {
	return &statelessBuilder{NewStatelessWidgetBase(opts...), f}
}

func TestInheritedNotifiesDependents(t *testing.T) {
	var log []*testInherited
	b := builderOf(func(bc BuildContext) Widget {
		w, _ := DependOn[*testInherited](bc)
		log = append(log, w)
		return newLeaf()
	})
	tr := newTree(t)
	first := newTestInherited(true, b)
	tr.pump(first)
	if !slices.Equal(log, []*testInherited{first}) {
		t.Fatalf("log = %v", log)
	}
	tr.pump(newTestInherited(false, b))
	if len(log) != 1 {
		t.Errorf("a widget that says it does not notify rebuilt its dependents: %v", log)
	}
	third := newTestInherited(true, b)
	tr.pump(third)
	if !slices.Equal(log, []*testInherited{first, third}) {
		t.Errorf("log = %v, want the first and the third", log)
	}
}

func TestUpdateInheritedWhenReparentingState(t *testing.T) {
	key := NewGlobalKey[State]()
	var log []*testInherited
	build := func() *testInherited {
		return newTestInherited(true, newWrapper(builderOf(func(bc BuildContext) Widget {
			w, _ := DependOn[*testInherited](bc)
			log = append(log, w)
			return newLeaf()
		}), WidgetOpts{Key: key}), WidgetOpts{Key: foundation.NewUniqueKey()})
	}
	tr := newTree(t)
	first := build()
	tr.pump(first)
	second := build()
	tr.pump(second)
	if !slices.Equal(log, []*testInherited{first, second}) {
		t.Errorf("log = %v, want the first and the second", log)
	}
}

func TestUpdateInheritedWhenRemovingNode(t *testing.T) {
	var log []string
	report := func(name string) *statelessBuilder {
		return builderOf(func(bc BuildContext) Widget {
			v, _ := DependOn[*valueInherited](bc)
			log = append(log, name+": "+strconv.Itoa(v.value))
			return newLeaf()
		})
	}
	check := func(step string, want ...string) {
		t.Helper()
		if !slices.Equal(log, want) && (len(log) != 0 || len(want) != 0) {
			t.Errorf("%s: log = %v, want %v", step, log, want)
		}
		log = nil
	}
	tr := newTree(t)
	tr.pump(newValueInherited(1, newFlip(
		newValueInherited(2, newValueInherited(3, report("a"))),
		newValueInherited(2, report("b")),
	)))
	check("mount", "a: 3")
	tr.pumpFrame()
	check("pump", []string{}...)
	flipStateOf(t, tr).Flip()
	tr.pumpFrame()
	check("flip", "b: 2")
	flipStateOf(t, tr).Flip()
	tr.pumpFrame()
	check("flip back", "a: 3")
}

func TestUpdateInheritedWhenRemovingNodeAndChildHasGlobalKey(t *testing.T) {
	// The three variants: a new builder in each branch, a constant child, and the
	// minimised case with a global key on the builder itself.
	var log []int
	mkBuilder := func(opts ...WidgetOpts) *statelessBuilder {
		return builderOf(func(bc BuildContext) Widget {
			v, _ := DependOn[*valueInherited](bc)
			log = append(log, v.value)
			return newLeaf()
		}, opts...)
	}
	tests := []struct {
		name string
		tree func() Widget
	}{
		{"new builders", func() Widget {
			key := NewGlobalKey[State]()
			return newValueInherited(1, newFlip(
				newValueInherited(2, newValueInherited(3, newWrapper(mkBuilder(), WidgetOpts{Key: key}))),
				newValueInherited(2, newWrapper(mkBuilder(), WidgetOpts{Key: key})),
			))
		}},
		{"constant child", func() Widget {
			key := NewGlobalKey[State]()
			child := mkBuilder()
			return newValueInherited(1, newFlip(
				newValueInherited(2, newValueInherited(3, newWrapper(child, WidgetOpts{Key: key}))),
				newValueInherited(2, newWrapper(child, WidgetOpts{Key: key})),
			))
		}},
		{"minimised", func() Widget {
			child := mkBuilder(WidgetOpts{Key: NewGlobalKey[State]()})
			return newValueInherited(2, newFlip(newValueInherited(3, child), child))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log = nil
			tr := newTree(t)
			tr.pump(tc.tree())
			// The first value is the one of the left branch.
			first := log[0]
			if len(log) != 1 {
				t.Fatalf("log after mount = %v", log)
			}
			log = nil
			tr.pumpFrame()
			if len(log) != 0 {
				t.Errorf("a frame with nothing dirty built %v", log)
			}
			flipStateOf(t, tr).Flip()
			tr.pumpFrame()
			if len(log) != 1 || log[0] != 2 {
				t.Errorf("after the flip: log = %v, want [2]", log)
			}
			log = nil
			flipStateOf(t, tr).Flip()
			tr.pumpFrame()
			if len(log) != 1 || log[0] != first {
				t.Errorf("after the flip back: log = %v, want [%d]", log, first)
			}
		})
	}
}

func TestInheritedWidgetNotifiesDescendantsThatFailedToFindAMatch(t *testing.T) {
	var value = -1
	inner := newWrapper(builderOf(func(bc BuildContext) Widget {
		if w, ok := DependOn[*valueInherited](bc); ok {
			value = w.value
		} else {
			value = -99
		}
		return newLeaf()
	}), WidgetOpts{Key: NewGlobalKey[State]()})
	tr := newTree(t)
	tr.pump(inner)
	if value != -99 {
		t.Fatalf("value = %d, want no inherited widget", value)
	}
	value = -2
	tr.pump(newValueInherited(3, inner))
	if value != 3 {
		t.Errorf("value = %d, want 3", value)
	}
}

func TestInheritedWidgetDoesNotNotifyDescendantsWithoutDependencies(t *testing.T) {
	var builds int
	inner := newWrapper(builderOf(func(BuildContext) Widget { builds++; return newLeaf() }), WidgetOpts{Key: NewGlobalKey[State]()})
	tr := newTree(t)
	tr.pump(inner)
	tr.pump(newValueInherited(3, inner))
	if builds != 1 {
		t.Errorf("built %d times, want 1", builds)
	}
}

func TestInheritedWidgetNotifiesDescendantsWithOtherDependencies(t *testing.T) {
	var builds int
	inner := newWrapper(newTestInherited(false, builderOf(func(bc BuildContext) Widget {
		DependOn[*testInherited](bc)
		builds++
		return newLeaf()
	})), WidgetOpts{Key: NewGlobalKey[State]()})
	tr := newTree(t)
	tr.pump(inner)
	tr.pump(newValueInherited(3, inner))
	if builds != 2 {
		t.Errorf("built %d times, want 2", builds)
	}
}

func TestGetInheritedDoesNotCreateADependency(t *testing.T) {
	var builds int
	key := NewGlobalKey[State]()
	notifier := foundation.NewChangeNotifier()
	b := builderOf(func(bc BuildContext) Widget {
		got, ok := GetInherited[*changeNotifierInherited](bc)
		if !ok || Widget(got) != key.CurrentWidget() {
			t.Errorf("GetInherited = %v, %v; want the widget of the key", got, ok)
		}
		builds++
		return newLeaf()
	})
	inner := newChangeNotifierInherited(notifier, b, WidgetOpts{Key: key})
	tr := newTree(t)
	tr.pump(inner)
	notifier.NotifyListeners()
	tr.pump(inner)
	if builds != 1 {
		t.Errorf("built %d times, want 1", builds)
	}
}

func TestGetElementForInheritedWidget(t *testing.T) {
	tr := newTree(t)
	var found, missing InheritedElementSelf
	tr.pump(newTestInherited(true, builderOf(func(bc BuildContext) Widget {
		found = GetElementForInheritedWidget[*testInherited](bc)
		missing = GetElementForInheritedWidget[*valueInherited](bc)
		return newLeaf()
	})))
	if found == nil || missing != nil {
		t.Errorf("found = %v, missing = %v", found, missing)
	}
}

type expectFail struct {
	StatefulWidget
	onError func(error)
}

func (w *expectFail) CreateState() State { return &expectFailState{} }

type expectFailState struct{ StateOf[*expectFail] }

func (s *expectFailState) InitState() {
	if err := tryError(func() { DependOn[*testInherited](s.Context()) }); err != nil {
		s.Widget().onError(err)
	}
}
func (s *expectFailState) Build(BuildContext) Widget { return newLeaf() }

func TestDependencyInInitStateIsRejected(t *testing.T) {
	skipUnlessDebug(t)
	var got error
	tr := newTree(t)
	tr.pump(newTestInherited(true, &expectFail{NewStatefulWidgetBase(), func(err error) { got = err }}))
	_ = requireErrorTag(t, got, "DependOnInheritedBeforeInitState")
}

func TestInheritedNotifier(t *testing.T) {
	var builds int
	notifier := foundation.NewChangeNotifier()
	b := builderOf(func(bc BuildContext) Widget {
		DependOn[*changeNotifierInherited](bc)
		builds++
		return newLeaf()
	})
	inner := newChangeNotifierInherited(notifier, b)
	tr := newTree(t)
	steps := []struct {
		name string
		do   func()
		want int
	}{
		{"mount", func() { tr.pump(inner) }, 1},
		{"the same widget", func() { tr.pump(inner) }, 1},
		{"a frame", func() { tr.pumpFrame() }, 1},
		{"the notifier notifies", func() { notifier.NotifyListeners(); tr.pumpFrame() }, 2},
		{"the same widget again", func() { tr.pump(inner) }, 2},
		{"no notifier", func() { tr.pump(newChangeNotifierInherited(nil, b)) }, 3},
	}
	for _, s := range steps {
		s.do()
		if builds != s.want {
			t.Fatalf("%s: %d builds, want %d", s.name, builds, s.want)
		}
	}
	// After the swap to no notifier the old one is not listened to any more.
	notifier.NotifyListeners()
	tr.pumpFrame()
	if builds != 3 {
		t.Errorf("a notifier that was removed rebuilt the dependents: %d builds", builds)
	}
	if notifier.HasListeners() {
		t.Error("the notifier still has the listener")
	}
}

func TestInheritedNotifierRemovesListenerOnUnmount(t *testing.T) {
	notifier := foundation.NewChangeNotifier()
	tr := newTree(t)
	tr.pump(newChangeNotifierInherited(notifier, newLeaf()))
	if !notifier.HasListeners() {
		t.Fatal("the notifier has no listener")
	}
	tr.pump(newLeaf())
	if notifier.HasListeners() {
		t.Error("the listener stayed after the unmount")
	}
	// A swap to another notifier moves the listener.
	other := foundation.NewChangeNotifier()
	tr.pump(newChangeNotifierInherited(notifier, newLeaf()))
	tr.pump(newChangeNotifierInherited(other, newLeaf()))
	if notifier.HasListeners() || !other.HasListeners() {
		t.Error("the listener did not move to the new notifier")
	}
}

// themedBox is ThemedCard: a render object widget that reads an inherited widget
// when it creates and when it updates its render object.
type themedBox struct{ LeafRenderObjectWidget }

func (w *themedBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderConstrainedBox(themedConstraints(bc), nil)
}

func (w *themedBox) UpdateRenderObject(bc BuildContext, ro rendering.RenderObject) {
	ro.(*rendering.RenderConstrainedBox).SetAdditionalConstraints(themedConstraints(bc))
}

func themedConstraints(bc BuildContext) rendering.BoxConstraints {
	v, _ := DependOn[*valueInherited](bc)
	return rendering.NewBoxConstraintsTight(ui.Size{Width: float64(v.value), Height: 7})
}

func TestInheritedWidgetsCanTriggerRenderObjectUpdates(t *testing.T) {
	rt := newRenderTree(t)
	var setState func(func())
	value := 10
	rt.pump(newCenter(newBuilder(func(_ BuildContext, set func(func())) Widget {
		setState = set
		return newValueInherited(value, &themedBox{NewLeafRenderObjectWidgetBase()})
	})))
	el := renderElementOfType[*themedBox](t, rt)
	for _, v := range []int{10, 30, 55, 12} {
		value = v
		setState(func() {})
		rt.pumpFrame()
		if size, _ := el.Size(); size != (ui.Size{Width: float64(v), Height: 7}) {
			t.Errorf("value %d: size = %v", v, size)
		}
	}
}

func TestNotifyClientsOutsideBuild(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newTestInherited(true, newLeaf()))
	e := elementOfType[*testInherited](t, tr).(*InheritedElement)
	requireTag(t, "NotifyClientsOutsideBuild", func() { e.NotifyClients(e.widget) })
}
