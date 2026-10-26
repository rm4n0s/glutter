package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are for ErrorBoundary, which has no Flutter counterpart (SPEC
// §14.2, §12.10).

// boundaryLog records what the widgets of a test did.
type boundaryLog struct {
	events []string
	// paints is how many times a painter of the tree boundaryPainted.
	paints int
}

func (l *boundaryLog) add(event string) { l.events = append(l.events, event) }

func (l *boundaryLog) count(event string) int {
	n := 0
	for _, e := range l.events {
		if e == event {
			n++
		}
	}
	return n
}

// boundaryPaintCounter is a painter that counts its paints.
type boundaryPaintCounter struct {
	rendering.CustomPainterBase
	log *boundaryLog
}

func (p *boundaryPaintCounter) Paint(*ui.Canvas, ui.Size)                  { p.log.paints++ }
func (p *boundaryPaintCounter) ShouldRepaint(rendering.CustomPainter) bool { return false }

// boundaryPainted is a box that counts its paints in the log.
func boundaryPainted(log *boundaryLog) Widget {
	return NewCustomPaint(nil, CustomPaintOpts{
		Painter: &boundaryPaintCounter{CustomPainterBase: rendering.NewCustomPainterBase(nil), log: log},
		Size:    ui.Size{Width: 10, Height: 10},
	})
}

// boundaryProbe is a stateful widget that logs its life and fails on request.
type boundaryProbe struct {
	StatefulWidget
	name        string
	log         *boundaryLog
	child       Widget
	initPanic   bool
	buildPanic  bool
	updatePanic bool
}

func (w *boundaryProbe) CreateState() State { return &boundaryProbeState{} }

type boundaryProbeState struct{ StateOf[*boundaryProbe] }

func (s *boundaryProbeState) InitState() {
	w := s.Widget()
	w.log.add("init " + w.name)
	if w.initPanic {
		panic("init " + w.name + " failed")
	}
}

func (s *boundaryProbeState) DidUpdateWidget(*boundaryProbe) {
	w := s.Widget()
	w.log.add("update " + w.name)
	if w.updatePanic {
		panic("update " + w.name + " failed")
	}
}

func (s *boundaryProbeState) Dispose() { s.Widget().log.add("dispose " + s.Widget().name) }

func (s *boundaryProbeState) Build(BuildContext) Widget {
	w := s.Widget()
	w.log.add("build " + w.name)
	if w.buildPanic {
		panic("build " + w.name + " failed")
	}
	if w.child != nil {
		return w.child
	}
	return boundaryPainted(w.log)
}

func newBoundaryProbe(log *boundaryLog, name string, child Widget) *boundaryProbe {
	return &boundaryProbe{StatefulWidget: NewStatefulWidgetBase(), name: name, log: log, child: child}
}

var boundaryFallbackKey = foundation.NewValueKey("fallback")

// boundaryFallbackBox is what the fallbacks of the tests build.
func boundaryFallbackBox(log *boundaryLog) func(BuildContext, error) Widget {
	return func(BuildContext, error) Widget {
		log.add("fallback")
		return NewSizedBox(nil, SizedBoxOpts{Key: boundaryFallbackKey, Width: foundation.Some(5.0), Height: foundation.Some(5.0)})
	}
}

func boundaryErrorWidgets(ts *gluttertest.Tester) int {
	return ts.Count(gluttertest.ByType[*ErrorWidget]())
}

// boundaryTakeAll takes the errors of the owner's reporter.
func boundaryTakeAll(ts *gluttertest.Tester) []error { return ts.TakeErrors() }

func TestErrorBoundaryReplacesTheSubtreeInTheSamePump(t *testing.T) {
	tests := []struct {
		name string
		// child is the child of the boundary; it fails while it is built.
		child func(log *boundaryLog) Widget
		// disposed are the probes of the subtree that are disposed.
		disposed []string
	}{
		{"a panic in the Build of the child", func(log *boundaryLog) Widget {
			return NewColumn([]Widget{boundaryPainted(log), NewBuilder(func(BuildContext) Widget { panic("child failed") })})
		}, nil},
		{"a panic in the Build of the state of a grandchild", func(log *boundaryLog) Widget {
			return newBoundaryProbe(log, "outer", NewColumn([]Widget{boundaryPainted(log),
				NewBuilder(func(BuildContext) Widget {
					return &boundaryProbe{StatefulWidget: NewStatefulWidgetBase(), name: "bad", log: log, buildPanic: true}
				})}))
		}, []string{"outer"}},
		{"a panic in a deeper InitState", func(log *boundaryLog) Widget {
			return newBoundaryProbe(log, "outer", NewColumn([]Widget{boundaryPainted(log),
				NewBuilder(func(BuildContext) Widget {
					return NewBuilder(func(BuildContext) Widget {
						return &boundaryProbe{StatefulWidget: NewStatefulWidgetBase(), name: "bad", log: log, initPanic: true}
					})
				})}))
		}, []string{"outer"}},
		{"a panic in a LayoutBuilder", func(log *boundaryLog) Widget {
			return newBoundaryProbe(log, "outer", NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
				panic("layout builder failed")
			}))
		}, []string{"outer"}},
		{"a panic in an item of ListView.builder", func(log *boundaryLog) Widget {
			return newBoundaryProbe(log, "outer", NewListViewBuilder(func(_ BuildContext, i int) Widget {
				if i == 1 {
					panic("item failed")
				}
				return newBoundaryProbe(log, "item0", boundaryPainted(log))
			}, ListViewOpts{ItemCount: foundation.Some(3)}))
		}, []string{"outer", "item0"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			log := &boundaryLog{}
			var boundaryErrs []error
			var ownerErrsSeen []int
			boundary := NewErrorBoundary(boundaryFallbackBox(log), tc.child(log), ErrorBoundaryOpts{OnError: func(err error) {
				boundaryErrs = append(boundaryErrs, err)
				ownerErrsSeen = append(ownerErrsSeen, len(ts.Errors()))
			}})
			ts.PumpWidget(ltr(NewCenter(boundary)))

			if !ts.Exists(gluttertest.ByKey(boundaryFallbackKey)) {
				t.Fatal("the fallback is not in the tree after the pump")
			}
			if n := boundaryErrorWidgets(ts); n != 0 {
				t.Errorf("%d ErrorWidgets after the pump, want 0: the failed subtree must be gone", n)
			}
			if log.paints != 0 {
				t.Errorf("the failed subtree boundaryPainted %d times, want 0: it must never reach a boundaryPainted frame", log.paints)
			}
			if len(boundaryErrs) != 1 || ownerErrsSeen[0] != 0 {
				t.Fatalf("boundary got %d errors, seen %v owner errors before it; want 1 and [0]", len(boundaryErrs), ownerErrsSeen)
			}
			errs := boundaryTakeAll(ts)
			if len(errs) != 1 || errs[0] != boundaryErrs[0] {
				t.Errorf("the owner got %v, want the error that the boundary got: %v", errs, boundaryErrs)
			}
			for _, name := range tc.disposed {
				if log.count("dispose "+name) != 1 {
					t.Errorf("state %s disposed %d times, want 1; events %v", name, log.count("dispose "+name), log.events)
				}
			}
			if log.count("fallback") != 1 {
				t.Errorf("the fallback was built %d times, want 1", log.count("fallback"))
			}
		})
	}
}

func TestErrorBoundaryReplacesAChildThatFailsToUpdate(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	var got []error
	opts := ErrorBoundaryOpts{OnError: func(err error) { got = append(got, err) }}
	ts.PumpWidget(ltr(NewErrorBoundary(boundaryFallbackBox(log), newBoundaryProbe(log, "child", nil), opts)))
	if len(got) != 0 || ts.Exists(gluttertest.ByKey(boundaryFallbackKey)) {
		t.Fatal("the child failed before it was told to")
	}
	failing := newBoundaryProbe(log, "child", nil)
	failing.updatePanic = true
	ts.PumpWidget(ltr(NewErrorBoundary(boundaryFallbackBox(log), failing, opts)))
	if len(got) != 1 || !ts.Exists(gluttertest.ByKey(boundaryFallbackKey)) {
		t.Fatalf("boundary errors = %v, fallback shown = %v; want 1 and true", got, ts.Exists(gluttertest.ByKey(boundaryFallbackKey)))
	}
	if n := boundaryErrorWidgets(ts); n != 0 {
		t.Errorf("%d ErrorWidgets, want 0", n)
	}
	// The child that failed half way is disposed, too, though nothing refers to it.
	if log.count("dispose child") != 1 {
		t.Errorf("the child was disposed %d times, want 1; events %v", log.count("dispose child"), log.events)
	}
	boundaryTakeAll(ts)
}

func TestErrorBoundaryStaysFailedUntilItsKeyChanges(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	var got []error
	boundary := func(key foundation.Key, child Widget) Widget {
		return ltr(NewErrorBoundary(boundaryFallbackBox(log), child, ErrorBoundaryOpts{Key: key, OnError: func(err error) { got = append(got, err) }}))
	}
	ts.PumpWidget(boundary(nil, NewBuilder(func(BuildContext) Widget { panic("first child failed") })))
	boundaryTakeAll(ts)
	if !ts.Exists(gluttertest.ByKey(boundaryFallbackKey)) {
		t.Fatal("no fallback")
	}

	// A new child from the parent: the fallback again; the child is not tried.
	ts.PumpWidget(boundary(nil, newBoundaryProbe(log, "second", nil)))
	if !ts.Exists(gluttertest.ByKey(boundaryFallbackKey)) || log.count("init second") != 0 {
		t.Errorf("the failed boundary tried its new child: events %v", log.events)
	}
	if len(got) != 1 {
		t.Errorf("OnError was called %d times, want 1", len(got))
	}

	// A new key: a new boundary, and the child builds.
	ts.PumpWidget(boundary(foundation.NewValueKey("again"), newBoundaryProbe(log, "third", nil)))
	if ts.Exists(gluttertest.ByKey(boundaryFallbackKey)) || log.count("init third") != 1 {
		t.Errorf("a new key did not try the child again: events %v", log.events)
	}
	if len(got) != 1 {
		t.Errorf("OnError was called %d times, want still 1", len(got))
	}
}

func TestErrorBoundaryFurtherErrorsAreOnlyReported(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	var got []error
	child := NewColumn([]Widget{
		NewBuilder(func(BuildContext) Widget { panic("one") }),
		NewBuilder(func(BuildContext) Widget { panic("two") }),
	})
	ts.PumpWidget(ltr(NewErrorBoundary(boundaryFallbackBox(log), child, ErrorBoundaryOpts{OnError: func(err error) { got = append(got, err) }})))
	errs := boundaryTakeAll(ts)
	if len(errs) != 2 || len(got) != 2 {
		t.Fatalf("owner got %d errors and the boundary %d, want 2 and 2", len(errs), len(got))
	}
	if log.count("fallback") != 1 || boundaryErrorWidgets(ts) != 0 {
		t.Errorf("the fallback was built %d times with %d ErrorWidgets, want 1 and 0", log.count("fallback"), boundaryErrorWidgets(ts))
	}
}

func TestErrorBoundaryFallbackGetsTheFirstError(t *testing.T) {
	ts := gluttertest.New(t)
	var fallbackErr error
	child := NewColumn([]Widget{
		NewBuilder(func(BuildContext) Widget { panic("one") }),
		NewBuilder(func(BuildContext) Widget { panic("two") }),
	})
	ts.PumpWidget(ltr(NewErrorBoundary(func(_ BuildContext, err error) Widget {
		fallbackErr = err
		return NewSizedBox(nil)
	}, child)))
	errs := boundaryTakeAll(ts)
	if len(errs) != 2 || fallbackErr != errs[0] {
		t.Errorf("fallback error = %v, want the first of %v", fallbackErr, errs)
	}
}

func TestErrorBoundaryNested(t *testing.T) {
	failingChild := func() Widget { return NewBuilder(func(BuildContext) Widget { panic("child failed") }) }
	tests := []struct {
		name string
		// build makes the outer boundary with the inner one in it.
		build func(log *boundaryLog, outer, inner func(error)) Widget
		// outerErrs and innerErrs are how many errors each boundary gets.
		outerErrs, innerErrs int
		// outerFallback is whether the outer boundary shows its fallback.
		outerFallback bool
		// innerTag is the tag that the outer boundary gets, when it gets one.
		outerTag string
		// ownerErrs is how many errors the owner gets.
		ownerErrs int
	}{
		{"the inner one takes the error", func(log *boundaryLog, outer, inner func(error)) Widget {
			innerB := NewErrorBoundary(boundaryFallbackBox(log), failingChild(), ErrorBoundaryOpts{OnError: inner})
			return NewErrorBoundary(func(BuildContext, error) Widget { log.add("outer fallback"); return NewSizedBox(nil) },
				NewColumn([]Widget{boundaryPainted(log), innerB}), ErrorBoundaryOpts{OnError: outer})
		}, 0, 1, false, "", 1},
		{"a panic in the inner fallback goes to the outer one", func(log *boundaryLog, outer, inner func(error)) Widget {
			innerB := NewErrorBoundary(func(BuildContext, error) Widget { panic("inner fallback failed") }, failingChild(),
				ErrorBoundaryOpts{OnError: inner})
			return NewErrorBoundary(func(BuildContext, error) Widget { log.add("outer fallback"); return NewSizedBox(nil) },
				NewColumn([]Widget{boundaryPainted(log), innerB}), ErrorBoundaryOpts{OnError: outer})
		}, 1, 1, true, foundation.BuildPanicked, 2},
		{"a panic in what the inner fallback builds goes to the outer one", func(log *boundaryLog, outer, inner func(error)) Widget {
			innerB := NewErrorBoundary(func(BuildContext, error) Widget {
				return NewBuilder(func(BuildContext) Widget { panic("inner fallback child failed") })
			}, failingChild(), ErrorBoundaryOpts{OnError: inner})
			return NewErrorBoundary(func(BuildContext, error) Widget { log.add("outer fallback"); return NewSizedBox(nil) },
				NewColumn([]Widget{boundaryPainted(log), innerB}), ErrorBoundaryOpts{OnError: outer})
		}, 1, 1, true, foundation.BuildPanicked, 2},
		{"a panic in the OnError of the inner one goes to the outer one", func(log *boundaryLog, outer, inner func(error)) Widget {
			innerB := NewErrorBoundary(boundaryFallbackBox(log), failingChild(), ErrorBoundaryOpts{OnError: func(err error) {
				inner(err)
				panic("inner OnError failed")
			}})
			return NewErrorBoundary(func(BuildContext, error) Widget { log.add("outer fallback"); return NewSizedBox(nil) },
				NewColumn([]Widget{boundaryPainted(log), innerB}), ErrorBoundaryOpts{OnError: outer})
		}, 1, 1, true, foundation.CallbackPanicked, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			log := &boundaryLog{}
			var outerGot, innerGot []error
			ts.PumpWidget(ltr(tc.build(log, func(err error) { outerGot = append(outerGot, err) }, func(err error) { innerGot = append(innerGot, err) })))
			if len(outerGot) != tc.outerErrs || len(innerGot) != tc.innerErrs {
				t.Fatalf("outer got %d errors, inner %d; want %d and %d", len(outerGot), len(innerGot), tc.outerErrs, tc.innerErrs)
			}
			if got := log.count("outer fallback") == 1; got != tc.outerFallback {
				t.Errorf("outer fallback shown = %v, want %v; events %v", got, tc.outerFallback, log.events)
			}
			if tc.outerFallback {
				if log.paints != 0 {
					t.Errorf("the subtree of the outer boundary boundaryPainted %d times, want 0", log.paints)
				}
				_ = requireErrorTag(t, outerGot[0], tc.outerTag)
			} else if log.paints == 0 {
				t.Error("the sibling of the inner boundary was not boundaryPainted")
			}
			if n := boundaryErrorWidgets(ts); n != 0 {
				t.Errorf("%d ErrorWidgets, want 0", n)
			}
			if errs := boundaryTakeAll(ts); len(errs) != tc.ownerErrs {
				t.Errorf("the owner got %d errors, want %d: %v", len(errs), tc.ownerErrs, errs)
			}
		})
	}
}

func TestErrorBoundaryWithoutAFallbackShowsTheErrorWidget(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	child := NewColumn([]Widget{boundaryPainted(log), NewBuilder(func(BuildContext) Widget { panic("failed") })})
	ts.PumpWidget(ltr(NewErrorBoundary(nil, child)))
	if n := boundaryErrorWidgets(ts); n != 1 {
		t.Errorf("%d ErrorWidgets, want 1", n)
	}
	if log.paints != 0 {
		t.Errorf("the failed subtree boundaryPainted %d times, want 0", log.paints)
	}
	if ts.Exists(gluttertest.ByType[*Column]()) {
		t.Error("the failed child is still in the tree")
	}
	boundaryTakeAll(ts)
}

func TestErrorBoundaryWithoutAFailureShowsTheChild(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	ts.PumpWidget(ltr(NewErrorBoundary(boundaryFallbackBox(log), newBoundaryProbe(log, "child", nil))))
	if log.count("init child") != 1 || log.count("fallback") != 0 || log.paints == 0 {
		t.Errorf("events %v, paints %d", log.events, log.paints)
	}
}

func TestWithoutAnErrorBoundaryTheErrorWidgetTakesThePlace(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	ts.PumpWidget(ltr(NewColumn([]Widget{boundaryPainted(log), NewExpanded(NewBuilder(func(BuildContext) Widget { panic("failed") }))})))
	if n := boundaryErrorWidgets(ts); n != 1 {
		t.Errorf("%d ErrorWidgets, want 1", n)
	}
	if log.paints == 0 {
		t.Error("the sibling of the widget that failed was not boundaryPainted")
	}
	if errs := boundaryTakeAll(ts); len(errs) != 1 {
		t.Errorf("the owner got %d errors, want 1: %v", len(errs), errs)
	}
}

func TestErrorBoundaryDoesNotTakeErrorsOutsideItsSubtree(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	var got []error
	ts.PumpWidget(ltr(NewColumn([]Widget{
		NewErrorBoundary(boundaryFallbackBox(log), newBoundaryProbe(log, "inside", nil), ErrorBoundaryOpts{OnError: func(err error) { got = append(got, err) }}),
		NewBuilder(func(BuildContext) Widget { panic("outside") }),
	})))
	if len(got) != 0 || log.count("fallback") != 0 {
		t.Errorf("the boundary took an error of a sibling: %v", got)
	}
	if n := boundaryErrorWidgets(ts); n != 1 {
		t.Errorf("%d ErrorWidgets, want 1", n)
	}
	boundaryTakeAll(ts)
}

// dependent is a stateful widget whose DidChangeDependencies panics when the
// text direction changes: BuildScope recovers that one, not the Build.
type dependent struct {
	StatefulWidget
	log *boundaryLog
}

func (w *dependent) CreateState() State { return &dependentState{} }

type dependentState struct {
	StateOf[*dependent]
	seen bool
}

func (s *dependentState) DidChangeDependencies() {
	if s.seen {
		panic("the direction changed")
	}
	s.seen = true
}

func (s *dependentState) Dispose() { s.Widget().log.add("dispose dependent") }

func (s *dependentState) Build(bc BuildContext) Widget {
	DirectionalityOf(bc)
	return boundaryPainted(s.Widget().log)
}

func TestErrorBoundaryTakesAFailedLifecycleHookOfARebuild(t *testing.T) {
	ts := gluttertest.New(t)
	log := &boundaryLog{}
	var got []error
	tree := func(direction ui.TextDirection) Widget {
		return NewDirectionality(direction, NewErrorBoundary(boundaryFallbackBox(log),
			&dependent{StatefulWidget: NewStatefulWidgetBase(), log: log},
			ErrorBoundaryOpts{Key: foundation.NewValueKey("boundary"), OnError: func(err error) { got = append(got, err) }}))
	}
	ts.PumpWidget(tree(ui.TextDirectionLTR))
	if len(got) != 0 {
		t.Fatalf("the child failed before the direction changed: %v", got)
	}
	ts.PumpWidget(tree(ui.TextDirectionRTL))
	if len(got) != 1 || !ts.Exists(gluttertest.ByKey(boundaryFallbackKey)) {
		t.Fatalf("boundary errors = %v, fallback shown = %v; want 1 and true", got, ts.Exists(gluttertest.ByKey(boundaryFallbackKey)))
	}
	if log.count("dispose dependent") != 1 {
		t.Errorf("the state was disposed %d times, want 1", log.count("dispose dependent"))
	}
	boundaryTakeAll(ts)
}
