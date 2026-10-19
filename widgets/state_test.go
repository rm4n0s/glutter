package widgets

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// probe is ProbeWidget of stateful_component_test.dart: it calls SetState at
// every time that is legal.
type probe struct {
	StatefulWidget
	buildCount *int
}

func newProbe(count *int, opts ...WidgetOpts) *probe {
	return &probe{NewStatefulWidgetBase(opts...), count}
}

func (w *probe) CreateState() State { return &probeState{} }

type probeState struct{ StateOf[*probe] }

func (s *probeState) InitState()                 { s.SetState(nil) }
func (s *probeState) DidUpdateWidget(old *probe) { s.SetState(nil) }
func (s *probeState) Build(BuildContext) Widget {
	s.SetState(nil)
	*s.Widget().buildCount++
	return newLeaf()
}

// flip is FlipWidget of test_widgets.dart: it shows its left child or its right
// child.
type flip struct {
	StatefulWidget
	left, right Widget
}

func newFlip(left, right Widget, opts ...WidgetOpts) *flip {
	return &flip{NewStatefulWidgetBase(opts...), left, right}
}

func (w *flip) CreateState() State { return &flipState{showLeft: true} }

type flipState struct {
	StateOf[*flip]
	showLeft bool
}

func (s *flipState) Flip() { s.SetState(func() { s.showLeft = !s.showLeft }) }
func (s *flipState) Build(BuildContext) Widget {
	if s.showLeft {
		return s.Widget().left
	}
	return s.Widget().right
}

func flipStateOf(t *testing.T, tr *tree) *flipState {
	t.Helper()
	return stateOf[*flipState](t, tr, elementOfType[*flip](t, tr))
}

func TestLegalTimesForSetState(t *testing.T) {
	var count int
	tr := newTree(t)
	steps := []struct {
		name string
		do   func()
		want int
	}{
		{"mount", func() { tr.pump(newProbe(&count, WidgetOpts{Key: foundation.NewKey("a")})) }, 1},
		{"replace by key", func() { tr.pump(newProbe(&count, WidgetOpts{Key: foundation.NewKey("b")})) }, 2},
		{"hidden in a flip", func() { tr.pump(newFlip(newLeaf(), newProbe(&count, WidgetOpts{Key: foundation.NewKey("c")}))) }, 2},
		{"flip to the probe", func() { flipStateOf(t, tr).Flip(); tr.pumpFrame() }, 3},
		{"flip away", func() { flipStateOf(t, tr).Flip(); tr.pumpFrame() }, 3},
		{"remove", func() { tr.pump(newLeaf()) }, 3},
	}
	for _, s := range steps {
		s.do()
		if count != s.want {
			t.Fatalf("after %q: build count = %d, want %d", s.name, count, s.want)
		}
	}
}

// badWidget (BadWidget of stateful_component_test.dart) marks the state of its
// parent dirty while it builds.
type badChild struct {
	StatelessWidget
	parent *badParentState
}

func (w *badChild) Build(BuildContext) Widget {
	w.parent.SetState(nil)
	return newLeaf()
}

type badParent struct{ StatefulWidget }

func (w *badParent) CreateState() State { return &badParentState{} }

type badParentState struct{ StateOf[*badParent] }

func (s *badParentState) Build(BuildContext) Widget {
	return &badChild{NewStatelessWidgetBase(), s}
}

func TestSetStateRules(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("parent state during build", func(t *testing.T) {
		tr := newTree(t)
		requireTag(t, "MarkNeedsBuildDuringBuild", func() { tr.pump(&badParent{NewStatefulWidgetBase()}) })
	})

	t.Run("state during dispose", func(t *testing.T) {
		tr := newTree(t)
		var st *spyState
		w := newSpy(nil)
		w.onInitState = func(bc BuildContext) { st = bc.(*StatefulElement).State().(*spyState) }
		w.onDispose = func(BuildContext) { st.SetState(nil) }
		tr.pump(w)
		requireTag(t, "MarkNeedsBuildOnDefunctElement", func() { tr.pump(newLeaf()) })
	})

	t.Run("after dispose", func(t *testing.T) {
		tr := newTree(t)
		var setState func(func())
		tr.pump(newBuilder(func(_ BuildContext, set func(func())) Widget { setState = set; return newLeaf() }))
		setState(nil) // control: setState before dispose is fine
		tr.pump(newLeaf())
		v := panicOf(func() { setState(nil) })
		err, ok := v.(error)
		if !ok {
			t.Fatalf("expected an error panic, got %v", v)
		}
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("SetStateAfterDispose") {
			t.Fatalf("unexpected failure: %v", v)
		}
		// The message explains the asynchronous causes and the Mounted check.
		for _, want := range []string{"asynchronous", "Mounted"} {
			if !strings.Contains(appErr.Message, want) {
				t.Errorf("message does not mention %q: %s", want, appErr.Message)
			}
		}
	})

	t.Run("before mount", func(t *testing.T) {
		s := &spyState{}
		requireTag(t, "SetStateBeforeMount", func() { s.SetState(nil) })
	})

	t.Run("context after unmount", func(t *testing.T) {
		tr := newTree(t)
		tr.pump(newSpy(nil))
		s := stateOf[*spyState](t, tr, elementOfType[*spy](t, tr))
		tr.pump(newLeaf())
		if s.Mounted() {
			t.Error("a disposed state is still mounted")
		}
		requireTag(t, "StateContextUnmounted", func() { s.Context() })
	})
}

func TestStateToString(t *testing.T) {
	skipUnlessDebug(t)
	if s := (&spyState{}).String(); !strings.Contains(s, "no widget") {
		t.Errorf("String of a new state = %q, want it to contain 'no widget'", s)
	}
	tr := newTree(t)
	tr.pump(newSpy(nil))
	got := normalizeHashes(stateOf[*spyState](t, tr, elementOfType[*spy](t, tr)).String())
	if got != "spyState#00000" {
		t.Errorf("String of a mounted state = %q", got)
	}
}

func TestUnmountedElementReleasesWidgetAndState(t *testing.T) {
	tr := newTree(t)
	tr.pump(newSpy(nil))
	el := elementOfType[*spy](t, tr)
	if el.Widget() == nil || el.(*StatefulElement).State() == nil {
		t.Fatal("a mounted element has no widget or state")
	}
	tr.pump(newLeaf())
	requireTag(t, "ElementWidgetUnmounted", func() { el.Widget() })
	requireTag(t, "ElementStateUnmounted", func() { el.(*StatefulElement).State() })
	if el.Mounted() {
		t.Error("an unmounted element is mounted")
	}
}

func TestStateLifecycleHooks(t *testing.T) {
	var log []string
	mk := func(key foundation.Key, child Widget) *spy {
		w := newSpy(child, WidgetOpts{Key: key})
		w.onInitState = func(BuildContext) { log = append(log, "initState") }
		w.onDidChangeDependencies = func(BuildContext) { log = append(log, "didChangeDependencies") }
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
		{"mount", func() { tr.pump(mk(nil, nil)) }, []string{"initState", "didChangeDependencies", "build"}},
		{"update with a new widget", func() { tr.pump(mk(nil, nil)) }, []string{"didUpdateWidget", "build"}},
		{"move under a container: replaced", func() { tr.pump(newWrapper(mk(nil, nil))) },
			[]string{"deactivate", "initState", "didChangeDependencies", "build", "dispose"}},
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

func TestStateLifecycleStates(t *testing.T) {
	skipUnlessDebug(t)
	var seen []StateLifecycle
	tr := newTree(t)
	w := newSpy(nil)
	var st *spyState
	capture := func(BuildContext) { seen = append(seen, st.DebugLifecycleState()) }
	w.onInitState = func(bc BuildContext) { st = bc.(*StatefulElement).State().(*spyState); capture(bc) }
	w.onDidChangeDependencies = capture
	w.onBuild = capture
	w.onDispose = capture
	tr.pump(w)
	tr.pump(newLeaf())
	want := []StateLifecycle{StateLifecycleCreated, StateLifecycleInitialized, StateLifecycleReady, StateLifecycleReady}
	if !slices.Equal(seen, want) {
		t.Errorf("lifecycle states = %v, want %v", seen, want)
	}
	if st.DebugLifecycleState() != StateLifecycleDefunct {
		t.Errorf("after dispose the state is %v", st.DebugLifecycleState())
	}
	if StateLifecycleReady.String() != "ready" || StateLifecycleReady.Name() != "ready" {
		t.Error("lifecycle names")
	}
}

func TestLifetimeIsCancelledAtDispose(t *testing.T) {
	tr := newTree(t)
	tr.pump(newSpy(nil))
	s := stateOf[*spyState](t, tr, elementOfType[*spy](t, tr))
	ctx := s.Lifetime()
	if ctx.Err() != nil {
		t.Fatal("the lifetime of a mounted state is done")
	}
	tr.pump(newLeaf())
	if ctx.Err() == nil {
		t.Error("the lifetime of a disposed state is not done")
	}
	if s.Lifetime().Err() == nil {
		t.Error("the lifetime asked for after dispose is not done")
	}
}

// reactivate does what a GlobalKey does in step 7.2: take an element out of
// the inactive list and give it a new parent.
func reactivate(t *testing.T, tr *tree, e Element, newParent Element, w Widget) {
	t.Helper()
	tr.owner.inactive.remove(e)
	e.AsElement().activateWithParent(newParent, nil)
	newParent.UpdateChild(e, w, nil)
}

func TestDeactivateAndActivate(t *testing.T) {
	var log []string
	mk := func() *spy {
		w := newSpy(nil)
		w.onInitState = func(BuildContext) { log = append(log, "initState") }
		w.onDidUpdateWidget = func(BuildContext) { log = append(log, "didUpdateWidget") }
		w.onDeactivate = func(BuildContext) { log = append(log, "deactivate") }
		w.onActivate = func(BuildContext) { log = append(log, "activate") }
		w.onBuild = func(BuildContext) { log = append(log, "build") }
		w.onDispose = func(BuildContext) { log = append(log, "dispose") }
		return w
	}
	check := func(step string, want ...string) {
		t.Helper()
		if !slices.Equal(log, want) {
			t.Errorf("%s: log = %v, want %v", step, log, want)
		}
		log = nil
	}
	tr := newTree(t)
	first, second := newMulti(mk()), newMulti()
	tr.pump(newMulti(first, second))
	check("mount", "initState", "build")

	// Move the state from the first parent to the second without losing it.
	from := tr.find(func(w Widget) bool { return w == first })[0].(*multiElement)
	to := tr.find(func(w Widget) bool { return w == second })[0].(*multiElement)
	el := from.children[0]
	from.ForgetChild(el)
	from.DeactivateChild(el)
	check("deactivate", "deactivate")
	reactivate(t, tr, el, to, mk())
	to.children = []Element{el}
	tr.pumpFrame()
	check("reactivate", "activate", "didUpdateWidget", "build")
	if got := el.AsElement().Parent(); got != Element(to) {
		t.Errorf("parent after reactivation = %v", got)
	}
	if !el.AsElement().DebugIsActiveForTest() {
		t.Error("the element is not active again")
	}
}

// stateMaker is a stateful widget whose CreateState is given by the test.
type stateMaker struct {
	StatefulWidget
	create func() State
}

func (w *stateMaker) CreateState() State { return w.create() }

type makerState struct{ StateOf[*stateMaker] }

func (*makerState) Build(BuildContext) Widget { return newLeaf() }

func TestCreateStateContract(t *testing.T) {
	reused := &makerState{}
	tests := []struct {
		name   string
		create func() State
		tag    string
	}{
		{"nil", func() State { return nil }, "CreateStateReturnedNil"},
		{"state of another widget type", func() State { return &builderState{} }, "CreateStateWrongType"},
		{"state that is already in use", func() State { return reused }, "CreateStateReturnedUsedState"},
	}
	// The first element takes the state; asking again for the same state is the
	// contract violation.
	NewStatefulElement(&stateMaker{StatefulWidget: NewStatefulWidgetBase(), create: func() State { return reused }}).Widget()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &stateMaker{StatefulWidget: NewStatefulWidgetBase(), create: tc.create}
			requireTag(t, tc.tag, func() { NewStatefulElement(w) })
		})
	}
}

func TestDidChangeDependenciesOutsideBuild(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newWrapper(newLeaf()))
	e := elementOfType[*wrapper](t, tr)
	requireTag(t, "DidChangeDependenciesOutsideBuild", e.DidChangeDependencies)

	// During a build it marks the element dirty (and a state hears about it).
	var calls int
	w := newSpy(nil)
	w.onDidChangeDependencies = func(BuildContext) { calls++ }
	tr.pump(w)
	calls = 0
	el := elementOfType[*spy](t, tr)
	tr.owner.BuildScope(tr.root, func() { el.DidChangeDependencies() })
	if calls != 1 {
		t.Errorf("DidChangeDependencies hook called %d times, want 1", calls)
	}
}
