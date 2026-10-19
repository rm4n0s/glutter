package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/lookup_boundary_test.dart.

var lookupBlue = ui.NewColor(0xFF0000FF)

// myStatefulContainer is MyStatefulContainer: its state is a myStatefulContainerState.
type myStatefulContainer struct {
	StatefulWidget
	child Widget
}

func newMyStatefulContainer(child Widget, key foundation.Key) *myStatefulContainer {
	return &myStatefulContainer{NewStatefulWidgetBase(WidgetOpts{Key: key}), child}
}

func (w *myStatefulContainer) CreateState() State { return &myStatefulContainerState{} }

type myStatefulContainerState struct{ StateOf[*myStatefulContainer] }

func (s *myStatefulContainerState) Build(BuildContext) Widget { return s.Widget().child }

// myOtherStatefulContainerState is MyOtherStatefulContainerState: a state that no
// widget creates.
type myOtherStatefulContainerState struct{ StateOf[*myStatefulContainer] }

func (s *myOtherStatefulContainerState) Build(BuildContext) Widget { return s.Widget().child }

// myInheritedWidget is MyInheritedWidget.
type myInheritedWidget struct {
	InheritedWidget
	value int
}

func newMyInheritedWidget(value int, child Widget, key foundation.Key) *myInheritedWidget {
	return &myInheritedWidget{NewInheritedWidgetBase(child, WidgetOpts{Key: key}), value}
}

func (w *myInheritedWidget) UpdateShouldNotify(old Widget) bool {
	return old.(*myInheritedWidget).value != w.value
}

// didChangeDependencySpy is DidChangeDependencySpy.
type didChangeDependencySpy struct {
	StatefulWidget
	onDidChangeDependencies func(bc BuildContext)
}

func newDidChangeDependencySpy(onDidChangeDependencies func(bc BuildContext), key foundation.Key) *didChangeDependencySpy {
	return &didChangeDependencySpy{NewStatefulWidgetBase(WidgetOpts{Key: key}), onDidChangeDependencies}
}

func (w *didChangeDependencySpy) CreateState() State { return &didChangeDependencySpyState{} }

type didChangeDependencySpyState struct {
	StateOf[*didChangeDependencySpy]
	didChangeDependenciesCount int
}

func (s *didChangeDependencySpyState) DidChangeDependencies() {
	s.didChangeDependenciesCount++
	s.Widget().onDidChangeDependencies(s.Context())
}

func (s *didChangeDependencySpyState) Build(BuildContext) Widget { return NewContainer(nil) }

func spyCount(ts *gluttertest.Tester) int {
	return gluttertest.StateOf[*didChangeDependencySpyState](ts, gluttertest.ByType[*didChangeDependencySpy]()).didChangeDependenciesCount
}

func lookupBuilder(fn func(bc BuildContext)) Widget {
	return NewBuilder(func(bc BuildContext) Widget {
		fn(bc)
		return NewSizedBoxExpand(nil)
	})
}

func TestLookupBoundaryDependOnInheritedWidgetOfExactType(t *testing.T) {
	t.Run("respects boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var through, stopped *myInheritedWidget
		key := foundation.NewUniqueKey()
		ts.PumpWidget(newMyInheritedWidget(2, NewLookupBoundary(lookupBuilder(func(bc BuildContext) {
			through, _ = DependOn[*myInheritedWidget](bc)
			stopped, _ = LookupBoundaryDependOnInheritedWidgetOfExactType[*myInheritedWidget](bc)
		})), key))
		if want := ts.Widget(gluttertest.ByKey(key)); through != want || stopped != nil {
			t.Errorf("through = %v, stopped = %v; want %v, nil", through, stopped, want)
		}
	})
	t.Run("ignores ancestor boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var found *myInheritedWidget
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewLookupBoundary(newMyInheritedWidget(2, lookupBuilder(func(bc BuildContext) {
			found, _ = LookupBoundaryDependOnInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}), key)))
		if want := ts.Widget(gluttertest.ByKey(key)); found != want {
			t.Errorf("found = %v, want %v", found, want)
		}
	})
	t.Run("finds widget before boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var through, stopped *myInheritedWidget
		key := foundation.NewUniqueKey()
		ts.PumpWidget(newMyInheritedWidget(2, NewLookupBoundary(newMyInheritedWidget(1, lookupBuilder(func(bc BuildContext) {
			through, _ = DependOn[*myInheritedWidget](bc)
			stopped, _ = LookupBoundaryDependOnInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}), key)), nil))
		if want := ts.Widget(gluttertest.ByKey(key)); through != want || stopped != want {
			t.Errorf("through = %v, stopped = %v; want %v for both", through, stopped, want)
		}
	})
	t.Run("creates dependency", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var found *myInheritedWidget
		tree := newDidChangeDependencySpy(func(bc BuildContext) {
			found, _ = LookupBoundaryDependOnInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}, nil)
		ts.PumpWidget(newMyInheritedWidget(1, tree, nil))
		if found.value != 1 || spyCount(ts) != 1 {
			t.Fatalf("value = %d, count = %d; want 1, 1", found.value, spyCount(ts))
		}
		ts.PumpWidget(newMyInheritedWidget(2, tree, nil))
		if found.value != 2 || spyCount(ts) != 2 {
			t.Errorf("value = %d, count = %d; want 2, 2", found.value, spyCount(ts))
		}
	})
	t.Run("causes didChangeDependencies to be called on move even if the dependency was not fulfilled due to the boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var found *myInheritedWidget
		tree := newDidChangeDependencySpy(func(bc BuildContext) {
			found, _ = LookupBoundaryDependOnInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}, NewGlobalKey[State]())
		ts.PumpWidget(newMyInheritedWidget(1, NewLookupBoundary(tree), nil))
		if found != nil || spyCount(ts) != 1 {
			t.Fatalf("found = %v, count = %d; want nil, 1", found, spyCount(ts))
		}
		// The value of the inherited widget changes, but there is no dependency due to the
		// boundary.
		ts.PumpWidget(newMyInheritedWidget(2, NewLookupBoundary(tree), nil))
		if found != nil || spyCount(ts) != 1 {
			t.Fatalf("found = %v, count = %d; want nil, 1", found, spyCount(ts))
		}
		// The widget is moved: didChangeDependencies is called, but the dependency is
		// still not found due to the boundary.
		ts.PumpWidget(NewSizedBox(newMyInheritedWidget(2, NewLookupBoundary(tree), nil)))
		if found != nil || spyCount(ts) != 2 {
			t.Fatalf("found = %v, count = %d; want nil, 2", found, spyCount(ts))
		}
		ts.PumpWidget(NewSizedBox(newMyInheritedWidget(2, NewLookupBoundary(newMyInheritedWidget(4, tree, nil)), nil)))
		if found == nil || found.value != 4 || spyCount(ts) != 3 {
			t.Errorf("found = %v, count = %d; want value 4, 3", found, spyCount(ts))
		}
	})
	t.Run("causes didChangeDependencies to be called on move even if the dependency was non-existent", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var found *myInheritedWidget
		tree := newDidChangeDependencySpy(func(bc BuildContext) {
			found, _ = LookupBoundaryDependOnInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}, NewGlobalKey[State]())
		ts.PumpWidget(tree)
		if found != nil || spyCount(ts) != 1 {
			t.Fatalf("found = %v, count = %d; want nil, 1", found, spyCount(ts))
		}
		// The widget moved, so didChangeDependencies must be called.
		ts.PumpWidget(NewSizedBox(tree))
		if found != nil || spyCount(ts) != 2 {
			t.Fatalf("found = %v, count = %d; want nil, 2", found, spyCount(ts))
		}
		ts.PumpWidget(newMyInheritedWidget(6, NewSizedBox(tree), nil))
		if found == nil || found.value != 6 || spyCount(ts) != 3 {
			t.Errorf("found = %v, count = %d; want value 6, 3", found, spyCount(ts))
		}
	})
}

func TestLookupBoundaryGetElementForInheritedWidgetOfExactType(t *testing.T) {
	t.Run("respects boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var through, stopped InheritedElementSelf
		key := foundation.NewUniqueKey()
		ts.PumpWidget(newMyInheritedWidget(2, NewLookupBoundary(lookupBuilder(func(bc BuildContext) {
			through = GetElementForInheritedWidget[*myInheritedWidget](bc)
			stopped = LookupBoundaryGetElementForInheritedWidgetOfExactType[*myInheritedWidget](bc)
		})), key))
		if want := ts.Element(gluttertest.ByKey(key)); through != want || stopped != nil {
			t.Errorf("through = %v, stopped = %v; want %v, nil", through, stopped, want)
		}
	})
	t.Run("ignores ancestor boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var found InheritedElementSelf
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewLookupBoundary(newMyInheritedWidget(2, lookupBuilder(func(bc BuildContext) {
			found = LookupBoundaryGetElementForInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}), key)))
		if want := ts.Element(gluttertest.ByKey(key)); found != want {
			t.Errorf("found = %v, want %v", found, want)
		}
	})
	t.Run("finds widget before boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var through, stopped InheritedElementSelf
		key := foundation.NewUniqueKey()
		ts.PumpWidget(newMyInheritedWidget(2, NewLookupBoundary(newMyInheritedWidget(1, lookupBuilder(func(bc BuildContext) {
			through = GetElementForInheritedWidget[*myInheritedWidget](bc)
			stopped = LookupBoundaryGetElementForInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}), key)), nil))
		if want := ts.Element(gluttertest.ByKey(key)); through != want || stopped != want {
			t.Errorf("through = %v, stopped = %v; want %v for both", through, stopped, want)
		}
	})
	t.Run("does not create a dependency", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		tree := newDidChangeDependencySpy(func(bc BuildContext) {
			LookupBoundaryGetElementForInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}, nil)
		ts.PumpWidget(newMyInheritedWidget(1, tree, nil))
		if spyCount(ts) != 1 {
			t.Fatalf("count = %d, want 1", spyCount(ts))
		}
		ts.PumpWidget(newMyInheritedWidget(2, tree, nil))
		if spyCount(ts) != 1 {
			t.Errorf("count = %d, want 1", spyCount(ts))
		}
	})
	t.Run("does not cause didChangeDependencies to be called on move when found", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		tree := newDidChangeDependencySpy(func(bc BuildContext) {
			LookupBoundaryGetElementForInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}, NewGlobalKey[State]())
		ts.PumpWidget(newMyInheritedWidget(1, NewLookupBoundary(tree), nil))
		ts.PumpWidget(newMyInheritedWidget(2, NewLookupBoundary(tree), nil))
		ts.PumpWidget(NewSizedBox(newMyInheritedWidget(2, NewLookupBoundary(tree), nil)))
		ts.PumpWidget(NewSizedBox(newMyInheritedWidget(2, NewLookupBoundary(newMyInheritedWidget(4, tree, nil)), nil)))
		if spyCount(ts) != 1 {
			t.Errorf("count = %d, want 1", spyCount(ts))
		}
	})
	t.Run("does not cause didChangeDependencies to be called on move when nothing was found", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		tree := newDidChangeDependencySpy(func(bc BuildContext) {
			LookupBoundaryGetElementForInheritedWidgetOfExactType[*myInheritedWidget](bc)
		}, NewGlobalKey[State]())
		ts.PumpWidget(tree)
		ts.PumpWidget(NewSizedBox(tree))
		ts.PumpWidget(newMyInheritedWidget(6, NewSizedBox(tree), nil))
		if spyCount(ts) != 1 {
			t.Errorf("count = %d, want 1", spyCount(ts))
		}
	})
}

// lookupFinder is one of the functions that find an ancestor through the context
// and up to the boundary: the group of the Dart tests that is run for each.
type lookupFinder struct {
	name string
	// container wraps the child in the ancestor that is looked for; key is its key.
	container func(child Widget, key foundation.Key) Widget
	// other is a widget that is another ancestor (for findRoot: the same kind).
	through, stopped, unfulfilled func(bc BuildContext) any
	// find is the thing that the key finds in the tree.
	find func(ts *gluttertest.Tester, key foundation.Key) any
}

func lookupFinders() []lookupFinder {
	return []lookupFinder{
		{
			name:      "findAncestorWidgetOfExactType",
			container: func(child Widget, key foundation.Key) Widget { return NewContainer(child, ContainerOpts{Key: key}) },
			through: func(bc BuildContext) any {
				w, _ := FindAncestorWidgetOfExactType[*Container](bc)
				return orNil(w)
			},
			stopped: func(bc BuildContext) any {
				w, _ := LookupBoundaryFindAncestorWidgetOfExactType[*Container](bc)
				return orNil(w)
			},
			unfulfilled: func(bc BuildContext) any {
				w, _ := LookupBoundaryFindAncestorWidgetOfExactType[*Wrap](bc)
				return orNil(w)
			},
			find: func(ts *gluttertest.Tester, key foundation.Key) any { return ts.Widget(gluttertest.ByKey(key)) },
		},
		{
			name:      "findAncestorStateOfType",
			container: func(child Widget, key foundation.Key) Widget { return newMyStatefulContainer(child, key) },
			through: func(bc BuildContext) any {
				s, _ := FindAncestorStateOfType[*myStatefulContainerState](bc)
				return orNil(s)
			},
			stopped: func(bc BuildContext) any {
				s, _ := LookupBoundaryFindAncestorStateOfType[*myStatefulContainerState](bc)
				return orNil(s)
			},
			unfulfilled: func(bc BuildContext) any {
				s, _ := LookupBoundaryFindAncestorStateOfType[*myOtherStatefulContainerState](bc)
				return orNil(s)
			},
			find: func(ts *gluttertest.Tester, key foundation.Key) any {
				return gluttertest.StateOf[*myStatefulContainerState](ts, gluttertest.ByKey(key))
			},
		},
		{
			name:      "findRootAncestorStateOfType",
			container: func(child Widget, key foundation.Key) Widget { return newMyStatefulContainer(child, key) },
			through: func(bc BuildContext) any {
				s, _ := FindRootAncestorStateOfType[*myStatefulContainerState](bc)
				return orNil(s)
			},
			stopped: func(bc BuildContext) any {
				s, _ := LookupBoundaryFindRootAncestorStateOfType[*myStatefulContainerState](bc)
				return orNil(s)
			},
			unfulfilled: func(bc BuildContext) any {
				s, _ := LookupBoundaryFindRootAncestorStateOfType[*myOtherStatefulContainerState](bc)
				return orNil(s)
			},
			find: func(ts *gluttertest.Tester, key foundation.Key) any {
				return gluttertest.StateOf[*myStatefulContainerState](ts, gluttertest.ByKey(key))
			},
		},
		{
			name: "findAncestorRenderObjectOfType",
			container: func(child Widget, key foundation.Key) Widget {
				return NewPadding(painting.EdgeInsetsZero, child, PaddingOpts{Key: key})
			},
			through: func(bc BuildContext) any {
				r, _ := FindAncestorRenderObjectOfType[*rendering.RenderPadding](bc)
				return orNil(r)
			},
			stopped: func(bc BuildContext) any {
				r, _ := LookupBoundaryFindAncestorRenderObjectOfType[*rendering.RenderPadding](bc)
				return orNil(r)
			},
			unfulfilled: func(bc BuildContext) any {
				r, _ := LookupBoundaryFindAncestorRenderObjectOfType[*rendering.RenderPositionedBox](bc)
				return orNil(r)
			},
			find: func(ts *gluttertest.Tester, key foundation.Key) any { return ts.RenderObject(gluttertest.ByKey(key)) },
		},
	}
}

// orNil makes the nil pointer of a failed lookup a nil interface, so that it
// compares equal to nil.
func orNil[T comparable](v T) any {
	var zero T
	if v == zero {
		return nil
	}
	return v
}

func TestLookupBoundaryFindAncestor(t *testing.T) {
	for _, f := range lookupFinders() {
		t.Run(f.name, func(t *testing.T) {
			t.Run("respects boundary", func(t *testing.T) {
				ts := gluttertest.New(t, gluttertest.Opts{})
				var through, stopped any
				key := foundation.NewUniqueKey()
				ts.PumpWidget(f.container(NewLookupBoundary(lookupBuilder(func(bc BuildContext) {
					through, stopped = f.through(bc), f.stopped(bc)
				})), key))
				if want := f.find(ts, key); through != want || stopped != nil {
					t.Errorf("through = %v, stopped = %v; want %v, nil", through, stopped, want)
				}
			})
			t.Run("finds the right widget before the boundary", func(t *testing.T) {
				ts := gluttertest.New(t, gluttertest.Opts{})
				var through, stopped any
				outerKey, innerKey := foundation.NewUniqueKey(), foundation.NewUniqueKey()
				probe := lookupBuilder(func(bc BuildContext) { through, stopped = f.through(bc), f.stopped(bc) })
				root := f.name == "findRootAncestorStateOfType"
				if root {
					// The outermost container inside the boundary is the answer of the
					// lookup that stops at the boundary.
					ts.PumpWidget(f.container(NewLookupBoundary(f.container(f.container(probe, nil), innerKey)), outerKey))
				} else {
					ts.PumpWidget(f.container(NewLookupBoundary(f.container(f.container(probe, innerKey), nil)), outerKey))
				}
				if root {
					if throughWant, stoppedWant := f.find(ts, outerKey), f.find(ts, innerKey); through != throughWant || stopped != stoppedWant {
						t.Errorf("through = %v, stopped = %v; want %v, %v", through, stopped, throughWant, stoppedWant)
					}
					return
				}
				if want := f.find(ts, innerKey); through != want || stopped != want {
					t.Errorf("through = %v, stopped = %v; want %v for both", through, stopped, want)
				}
			})
			t.Run("works if nothing is found", func(t *testing.T) {
				ts := gluttertest.New(t, gluttertest.Opts{})
				var stopped any = "unset"
				ts.PumpWidget(lookupBuilder(func(bc BuildContext) { stopped = f.stopped(bc) }))
				if stopped != nil {
					t.Errorf("stopped = %v, want nil", stopped)
				}
			})
			t.Run("does not establish a dependency", func(t *testing.T) {
				ts := gluttertest.New(t, gluttertest.Opts{})
				var through, stopped, unfulfilled any
				innerKey := foundation.NewUniqueKey()
				tree := NewLookupBoundary(f.container(newDidChangeDependencySpy(func(bc BuildContext) {
					through, stopped, unfulfilled = f.through(bc), f.stopped(bc), f.unfulfilled(bc)
				}, NewGlobalKey[State]()), innerKey))
				ts.PumpWidget(tree)
				if want := f.find(ts, innerKey); through != want || stopped != want || unfulfilled != nil {
					t.Fatalf("through = %v, stopped = %v, unfulfilled = %v; want %v, %v, nil", through, stopped, unfulfilled, want, want)
				}
				if spyCount(ts) != 1 {
					t.Fatalf("count = %d, want 1", spyCount(ts))
				}
				// Changing the structure of the tree moves the spy through its global key.
				ts.PumpWidget(NewSizedBox(tree))
				// The restructuring would have called didChangeDependencies if a dependency had
				// been established.
				if spyCount(ts) != 1 {
					t.Errorf("count = %d, want 1", spyCount(ts))
				}
			})
		})
	}
	t.Run("findAncestorWidgetOfExactType finds the boundary", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var through, stopped *LookupBoundary
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewContainer(NewLookupBoundary(lookupBuilder(func(bc BuildContext) {
			through, _ = FindAncestorWidgetOfExactType[*LookupBoundary](bc)
			stopped, _ = LookupBoundaryFindAncestorWidgetOfExactType[*LookupBoundary](bc)
		}), WidgetOpts{Key: key})))
		if want := ts.Widget(gluttertest.ByKey(key)); through != want || stopped != want {
			t.Errorf("through = %v, stopped = %v; want %v for both", through, stopped, want)
		}
	})
}

func TestLookupBoundaryVisitAncestorElements(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var through, stopped, stoppedEarly []Element
	keys := make([]foundation.Key, 5)
	for i := range keys {
		keys[i] = foundation.NewUniqueKey()
	}
	ts.PumpWidget(NewContainer(NewContainer(NewLookupBoundary(NewContainer(NewContainer(NewBuilder(func(bc BuildContext) Widget {
		bc.VisitAncestorElements(func(e Element) bool {
			through = append(through, e)
			return e.Widget().Key() != keys[0]
		})
		LookupBoundaryVisitAncestorElements(bc, func(e Element) bool {
			stopped = append(stopped, e)
			return e.Widget().Key() != keys[0]
		})
		LookupBoundaryVisitAncestorElements(bc, func(e Element) bool {
			stoppedEarly = append(stoppedEarly, e)
			return e.Widget().Key() != keys[3]
		})
		return NewSizedBox(nil)
	}), ContainerOpts{Key: keys[4]}), ContainerOpts{Key: keys[3]}), WidgetOpts{Key: keys[2]}), ContainerOpts{Key: keys[1]}), ContainerOpts{Key: keys[0]}))
	element := func(i int) Element { return ts.Element(gluttertest.ByKey(keys[i])) }
	check := func(name string, got []Element, want ...Element) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: %d elements, want %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: element %d is %v, want %v", name, i, got[i], want[i])
			}
		}
	}
	check("through the boundary", through, element(4), element(3), element(2), element(1), element(0))
	check("stopped at the boundary", stopped, element(4), element(3), element(2))
	check("stopped at the boundary, terminated early", stoppedEarly, element(4), element(3))
}

func TestLookupBoundaryVisitChildElements(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	root, child1, child2, child3 := foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()
	ts.PumpWidget(NewColumn([]Widget{
		NewLookupBoundary(NewContainer(nil), WidgetOpts{Key: child1}),
		NewContainer(NewLookupBoundary(NewContainer(nil)), ContainerOpts{Key: child2}),
		NewContainer(nil, ContainerOpts{Key: child3}),
	}, ColumnOpts{Key: root}))
	var through, stopped []Element
	context := ts.Element(gluttertest.ByKey(root))
	context.VisitChildElements(func(e Element) { through = append(through, e) })
	LookupBoundaryVisitChildElements(context, func(e Element) { stopped = append(stopped, e) })
	element := func(key foundation.Key) Element { return ts.Element(gluttertest.ByKey(key)) }
	if len(through) != 3 || through[0] != element(child1) || through[1] != element(child2) || through[2] != element(child3) {
		t.Errorf("through the boundary = %v", through)
	}
	if len(stopped) != 2 || stopped[0] != element(child2) || stopped[1] != element(child3) {
		t.Errorf("stopped at the boundary = %v", stopped)
	}
}

func TestLookupBoundaryDebugIsHidingAncestor(t *testing.T) {
	skipUnlessDebug(t) // the functions report false in release builds
	type hider struct {
		name string
		// container is the ancestor that is looked for.
		container func(child Widget) Widget
		hiding    func(bc BuildContext) bool
	}
	hiders := []hider{
		{"widget", func(child Widget) Widget {
			return NewContainer(child, ContainerOpts{Padding: painting.NewEdgeInsetsAll(10), Color: foundation.Some(lookupBlue)})
		}, LookupBoundaryDebugIsHidingAncestorWidgetOfExactType[*Container]},
		{"state", func(child Widget) Widget { return newMyStatefulContainer(child, nil) }, LookupBoundaryDebugIsHidingAncestorStateOfType[*myStatefulContainerState]},
		{"render object", func(child Widget) Widget { return NewPadding(painting.NewEdgeInsetsAll(10), child) },
			LookupBoundaryDebugIsHidingAncestorRenderObjectOfType[*rendering.RenderPadding]},
	}
	for _, h := range hiders {
		t.Run(h.name, func(t *testing.T) {
			probe := func(result *bool) Widget {
				return NewBuilder(func(bc BuildContext) Widget {
					*result = h.hiding(bc)
					return NewContainer(nil)
				})
			}
			var result bool
			tests := []struct {
				name string
				tree func() Widget
				want bool
			}{
				{"is hiding", func() Widget { return h.container(NewLookupBoundary(probe(&result))) }, true},
				{"is not hiding the entity within the boundary", func() Widget { return NewLookupBoundary(h.container(probe(&result))) }, false},
				{"is not hiding if there is no boundary", func() Widget { return h.container(probe(&result)) }, false},
				{"is not hiding if there is no boundary and no entity", func() Widget { return probe(&result) }, false},
			}
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					ts := gluttertest.New(t, gluttertest.Opts{})
					result = !tc.want
					ts.PumpWidget(tc.tree())
					if result != tc.want {
						t.Errorf("hiding = %v, want %v", result, tc.want)
					}
				})
			}
		})
	}
}
