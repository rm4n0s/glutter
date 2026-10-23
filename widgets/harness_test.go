package widgets

import (
	"regexp"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// This file is the minimal harness the widgets tests of step 7.1 run on, until
// gluttertest exists (step 7.5): a root element with an owner, and a few test
// widgets (Flutter's test_widgets.dart and the helper classes of
// framework_test.dart).

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
}

func panicOf(call func()) (v any) {
	defer func() { v = recover() }()
	call()
	return nil
}

// requireTag runs call and fails unless it panics with an *errors.Error whose
// route ends in tag.
func requireTag(t *testing.T, tag string, call func()) {
	t.Helper()
	v := panicOf(call)
	if v == nil {
		t.Fatalf("expected a panic with tag %s", tag)
	}
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(tag) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tag)
	}
}

// rootWidget holds the one child of the tree under test.
type rootWidget struct {
	WidgetBase
	child Widget
}

// rootElement is the root of the tree under test; it stands for Flutter's
// RenderObjectToWidgetElement.
type rootElement struct {
	ElementBase
	child Element
}

func (e *rootElement) VisitChildren(v ElementVisitor) {
	if e.child != nil {
		v(e.child)
	}
}

func (e *rootElement) ForgetChild(child Element) { e.child = nil }

func (e *rootElement) Mount(parent Element, slot any) {
	e.ElementBase.Mount(parent, slot)
	e.child = e.UpdateChild(e.child, e.widget.(*rootWidget).child, nil)
}

func (e *rootElement) Update(w Widget) {
	e.ElementBase.Update(w)
	e.child = e.UpdateChild(e.child, w.(*rootWidget).child, nil)
}

// tree is a widget tree with its build owner.
type tree struct {
	t         *testing.T
	owner     *BuildOwner
	root      *rootElement
	scheduled int
	// errs are the errors the owner reported.
	errs []error
	// collectPanics makes the tree collect the panics that the framework
	// recovers in errs. By default it panics again with them, so that a test that
	// does not expect one fails where it happens.
	collectPanics bool
}

func newTree(t *testing.T) *tree {
	t.Helper()
	tr := &tree{t: t}
	tr.owner = NewBuildOwner(BuildOwnerOpts{
		OnBuildScheduled: func() { tr.scheduled++ },
		OnError: func(err error) {
			if !tr.collectPanics && foundation.IsRecoveredPanic(err) {
				panic(err)
			}
			tr.errs = append(tr.errs, err)
		},
	})
	return tr
}

// pump replaces the widget of the tree and builds it, as the first part of
// tester.pumpWidget does.
func (tr *tree) pump(w Widget) {
	tr.t.Helper()
	rw := &rootWidget{child: w}
	if tr.root == nil {
		tr.root = &rootElement{}
		InitElement(&tr.root.ElementBase, tr.root, rw)
		tr.root.AssignOwner(tr.owner)
		tr.owner.BuildScope(tr.root, func() { tr.root.Mount(nil, nil) })
	} else {
		tr.owner.BuildScope(tr.root, func() { tr.root.Update(rw) })
	}
	tr.owner.FinalizeTree()
}

// pumpFrame builds the dirty elements, as tester.pump does.
func (tr *tree) pumpFrame() {
	tr.t.Helper()
	tr.owner.BuildScope(tr.root, nil)
	tr.owner.FinalizeTree()
}

// find returns the elements below the root, in depth-first order, whose widget
// satisfies pred.
func (tr *tree) find(pred func(Widget) bool) []Element {
	var found []Element
	var visit func(Element)
	visit = func(e Element) {
		if pred(e.AsElement().widget) {
			found = append(found, e)
		}
		e.VisitChildren(visit)
	}
	tr.root.VisitChildren(visit)
	return found
}

func findByType[W Widget](tr *tree) []Element {
	return tr.find(func(w Widget) bool { _, ok := w.(W); return ok })
}

func elementOfType[W Widget](t *testing.T, tr *tree) Element {
	t.Helper()
	es := findByType[W](tr)
	if len(es) != 1 {
		t.Fatalf("found %d elements for %T, want 1", len(es), *new(W))
	}
	return es[0]
}

func stateOf[S State](t *testing.T, tr *tree, e Element) S {
	t.Helper()
	return e.(*StatefulElement).State().(S)
}

// leaf is a widget with no children and no build: Flutter's Container() with
// no child, where the test does not care what it is.
type leaf struct{ WidgetBase }

func newLeaf(opts ...WidgetOpts) *leaf { return &leaf{NewWidgetBase(opts...)} }

type leafElement struct{ ElementBase }

func (w *leaf) CreateElement() Element {
	e := &leafElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

func (e *leafElement) Mount(p Element, s any) {
	e.ElementBase.Mount(p, s)
	e.dirty = false
}

// wrapper is a StatelessWidget that builds its child (Wrapper of
// stateful_component_test.dart).
type wrapper struct {
	StatelessWidget
	child Widget
}

func newWrapper(child Widget, opts ...WidgetOpts) *wrapper {
	return &wrapper{NewStatelessWidgetBase(opts...), child}
}

func (w *wrapper) Build(BuildContext) Widget { return w.child }

// multi is a widget with a list of children (_DummyMultiChildWidget).
type multi struct {
	WidgetBase
	children []Widget
}

func newMulti(children ...Widget) *multi { return &multi{children: children} }

type multiElement struct {
	ElementBase
	children  []Element
	forgotten map[Element]struct{}
}

func (w *multi) CreateElement() Element {
	e := &multiElement{forgotten: map[Element]struct{}{}}
	InitElement(&e.ElementBase, e, w)
	return e
}

func (e *multiElement) VisitChildren(v ElementVisitor) {
	for _, c := range e.children {
		if _, ok := e.forgotten[c]; !ok {
			v(c)
		}
	}
}

func (e *multiElement) ForgetChild(child Element) {
	e.forgotten[child] = struct{}{}
	e.ElementBase.ForgetChild(child)
}

func (e *multiElement) Mount(parent Element, slot any) {
	e.ElementBase.Mount(parent, slot)
	var previous Element
	e.children = make([]Element, len(e.widget.(*multi).children))
	for i, w := range e.widget.(*multi).children {
		previous = e.InflateWidget(w, IndexedSlot{Index: i, Value: previous})
		e.children[i] = previous
	}
	e.dirty = false
}

func (e *multiElement) Update(w Widget) {
	e.ElementBase.Update(w)
	e.children = e.UpdateChildren(e.children, w.(*multi).children, UpdateChildrenOpts{ForgottenChildren: e.forgotten})
	clear(e.forgotten)
}

// builder is a StatefulBuilder (widgets/basic.dart StatefulBuilder).
type builder struct {
	StatefulWidget
	build func(bc BuildContext, setState func(func())) Widget
}

func newBuilder(build func(bc BuildContext, setState func(func())) Widget, opts ...WidgetOpts) *builder {
	return &builder{NewStatefulWidgetBase(opts...), build}
}

func (w *builder) CreateState() State { return &builderState{} }

type builderState struct{ StateOf[*builder] }

func (s *builderState) Build(bc BuildContext) Widget { return s.Widget().build(bc, s.SetState) }

// spy is StatefulWidgetSpy of framework_test.dart: a stateful widget that
// reports its lifecycle.
type spy struct {
	StatefulWidget
	child                                                                        Widget
	onInitState, onDidUpdateWidget, onDeactivate, onActivate, onBuild, onDispose func(bc BuildContext)
	onDidChangeDependencies                                                      func(bc BuildContext)
}

func newSpy(child Widget, opts ...WidgetOpts) *spy {
	return &spy{StatefulWidget: NewStatefulWidgetBase(opts...), child: child}
}

func (w *spy) CreateState() State { return &spyState{} }

type spyState struct{ StateOf[*spy] }

func (s *spyState) call(f func(BuildContext)) {
	if f != nil {
		f(s.Context())
	}
}
func (s *spyState) InitState()               { s.call(s.Widget().onInitState) }
func (s *spyState) DidChangeDependencies()   { s.call(s.Widget().onDidChangeDependencies) }
func (s *spyState) DidUpdateWidget(old *spy) { s.call(s.Widget().onDidUpdateWidget) }
func (s *spyState) Deactivate()              { s.call(s.Widget().onDeactivate) }
func (s *spyState) Activate()                { s.call(s.Widget().onActivate) }
func (s *spyState) Dispose()                 { s.call(s.Widget().onDispose) }
func (s *spyState) Build(BuildContext) Widget {
	s.call(s.Widget().onBuild)
	if s.Widget().child == nil {
		return newLeaf()
	}
	return s.Widget().child
}

var hashRe = regexp.MustCompile(`#[0-9a-f]{5}`)

// normalizeHashes replaces the short hashes of a description with #00000, as
// Flutter's equalsIgnoringHashCodes does.
func normalizeHashes(s string) string { return hashRe.ReplaceAllString(s, "#00000") }

// DebugIsActiveForTest is whether the element is active, in every build mode.
func (b *ElementBase) DebugIsActiveForTest() bool { return b.lifecycle == lifecycleActive }

// takeError returns the one error that was reported since the last call, or nil;
// it fails the test when there were more.
func (tr *tree) takeError() error {
	tr.t.Helper()
	return takeError(tr.t, &tr.errs)
}

func takeError(t *testing.T, errs *[]error) error {
	t.Helper()
	switch len(*errs) {
	case 0:
		return nil
	case 1:
		err := (*errs)[0]
		*errs = nil
		return err
	}
	t.Fatalf("%d errors were reported, want at most 1: %v", len(*errs), *errs)
	return nil
}

// requireErrorTag fails unless err is an *errors.Error whose route ends in tag.
func requireErrorTag(t *testing.T, err error, tag string) *errors.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error with tag %s, got none", tag)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(tag) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tag)
	}
	return appErr
}

// requireErrorTagFromPanic is requireTag for a value that was already recovered.
func requireErrorTagFromPanic(t *testing.T, v any, tag string) *errors.Error {
	t.Helper()
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	return requireErrorTag(t, err, tag)
}

// tryError runs fn and returns the error it panicked with, or nil.
func tryError(fn func()) error {
	v := panicOf(fn)
	if v == nil {
		return nil
	}
	if err, ok := v.(error); ok {
		return err
	}
	return errors.New(tagUnexpectedPanic, "not an error", "value", v)
}
