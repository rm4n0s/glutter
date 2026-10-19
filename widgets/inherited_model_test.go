package widgets

import (
	"fmt"
	"maps"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// abcModel is ABCModel of inherited_model_test.dart.
type abcModel struct {
	InheritedModel
	a, b, c foundation.Opt[int]
	// aspects are the aspects the model supports; nil supports all of them.
	aspects map[string]struct{}
}

func newABCModel(a, b, c foundation.Opt[int], aspects map[string]struct{}, child Widget) *abcModel {
	return &abcModel{NewInheritedModelBase(child), a, b, c, aspects}
}

func (m *abcModel) IsSupportedAspect(aspect any) bool {
	if m.aspects == nil {
		return true
	}
	_, ok := m.aspects[aspect.(string)]
	return ok
}

func sameSet(a, b map[string]struct{}) bool {
	return (a == nil) == (b == nil) && maps.Equal(a, b)
}

func (m *abcModel) UpdateShouldNotify(old Widget) bool {
	o := old.(*abcModel)
	return !sameSet(m.aspects, o.aspects) || m.a != o.a || m.b != o.b || m.c != o.c
}

func (m *abcModel) UpdateShouldNotifyDependent(old Widget, deps *AspectSet) bool {
	o := old.(*abcModel)
	return !sameSet(m.aspects, o.aspects) ||
		(m.a != o.a && deps.Contains("a")) ||
		(m.b != o.b && deps.Contains("b")) ||
		(m.c != o.c && deps.Contains("c"))
}

func optStr(o foundation.Opt[int]) string {
	if v, ok := o.Get(); ok {
		return fmt.Sprint(v)
	}
	return "null"
}

// showABCField is ShowABCField: it depends on one aspect of the model and writes
// down what it shows.
type showABCField struct {
	StatefulWidget
	field string
	out   map[string]string
}

func (w *showABCField) CreateState() State { return &showABCFieldState{} }

type showABCFieldState struct {
	StateOf[*showABCField]
	buildCount int
}

func (s *showABCFieldState) Build(bc BuildContext) Widget {
	m, _ := DependOnModel[*abcModel](bc, s.Widget().field)
	var v foundation.Opt[int]
	switch s.Widget().field {
	case "a":
		v = m.a
	case "b":
		v = m.b
	default:
		v = m.c
	}
	s.Widget().out[s.Widget().field] = fmt.Sprintf("%s: %s [%d]", s.Widget().field, optStr(v), s.buildCount)
	s.buildCount++
	return newLeaf()
}

func some1(v int) foundation.Opt[int] { return foundation.Some(v) }

func TestInheritedModelBasics(t *testing.T) {
	a, b, c := 0, 1, 2
	out := map[string]string{}
	showA := &showABCField{NewStatefulWidgetBase(), "a", out}
	showB := &showABCField{NewStatefulWidgetBase(), "b", out}
	showC := &showABCField{NewStatefulWidgetBase(), "c", out}
	showABC := builderOf(func(bc BuildContext) Widget {
		m, _ := DependOnModel[*abcModel](bc, nil)
		out["abc"] = fmt.Sprintf("a: %s b: %s c: %s", optStr(m.a), optStr(m.b), optStr(m.c))
		return newLeaf()
	})
	var setState func(func())
	tr := newTree(t)
	tr.pump(newBuilder(func(_ BuildContext, set func(func())) Widget {
		setState = set
		return newABCModel(some1(a), some1(b), some1(c), nil, newMulti(showA, showB, showC, showABC))
	}))
	check := func(step string, want map[string]string) {
		t.Helper()
		if !maps.Equal(out, want) {
			t.Errorf("%s:\n got %v\nwant %v", step, out, want)
		}
	}
	check("mount", map[string]string{"a": "a: 0 [0]", "b": "b: 1 [0]", "c": "c: 2 [0]", "abc": "a: 0 b: 1 c: 2"})
	// Only the widgets that depend on the aspect that changed are rebuilt.
	steps := []struct {
		name string
		do   func()
		want map[string]string
	}{
		{"increment a", func() { a++ }, map[string]string{"a": "a: 1 [1]", "b": "b: 1 [0]", "c": "c: 2 [0]", "abc": "a: 1 b: 1 c: 2"}},
		{"increment a again", func() { a++ }, map[string]string{"a": "a: 2 [2]", "b": "b: 1 [0]", "c": "c: 2 [0]", "abc": "a: 2 b: 1 c: 2"}},
		{"increment b", func() { b++ }, map[string]string{"a": "a: 2 [2]", "b": "b: 2 [1]", "c": "c: 2 [0]", "abc": "a: 2 b: 2 c: 2"}},
		{"increment c", func() { c++ }, map[string]string{"a": "a: 2 [2]", "b": "b: 2 [1]", "c": "c: 3 [1]", "abc": "a: 2 b: 2 c: 3"}},
	}
	for _, s := range steps {
		s.do()
		setState(func() {})
		tr.pumpFrame()
		check(s.name, s.want)
	}
}

func TestLookingUpMissingInheritedModel(t *testing.T) {
	var found *abcModel
	var ok bool
	tr := newTree(t)
	tr.pump(builderOf(func(bc BuildContext) Widget {
		found, ok = DependOnModel[*abcModel](bc, nil)
		return newLeaf()
	}))
	if ok || found != nil {
		t.Errorf("DependOnModel = %v, %v; want nothing", found, ok)
	}
	tr = newTree(t)
	tr.pump(builderOf(func(bc BuildContext) Widget {
		found, ok = DependOnModel[*abcModel](bc, "a")
		return newLeaf()
	}))
	if ok || found != nil {
		t.Errorf("DependOnModel with an aspect = %v, %v; want nothing", found, ok)
	}
}

// shadowPage builds the page of 'Inner InheritedModel shadows the outer one':
// two models, the inner one supporting the aspects given by inner().
func shadowPage(t *testing.T, a, b, c *int, inner func() map[string]struct{}, out map[string]string) (*tree, func()) {
	t.Helper()
	showA := &showABCField{NewStatefulWidgetBase(), "a", out}
	showB := &showABCField{NewStatefulWidgetBase(), "b", out}
	showC := &showABCField{NewStatefulWidgetBase(), "c", out}
	showABC := builderOf(func(bc BuildContext) Widget {
		m, _ := DependOnModel[*abcModel](bc, nil)
		out["abc"] = fmt.Sprintf("a: %s b: %s c: %s", optStr(m.a), optStr(m.b), optStr(m.c))
		return newLeaf()
	})
	var setState func(func())
	tr := newTree(t)
	tr.pump(newBuilder(func(_ BuildContext, set func(func())) Widget {
		setState = set
		return newABCModel(some1(*a), some1(*b), some1(*c), nil,
			newABCModel(some1(100+*a), some1(100+*b), foundation.None[int](), inner(),
				newMulti(showA, showB, showC, showABC)))
	}))
	return tr, func() { setState(func() {}); tr.pumpFrame() }
}

func TestInnerInheritedModelShadowsTheOuterOne(t *testing.T) {
	a, b, c := 0, 1, 2
	out := map[string]string{}
	tr, rebuild := shadowPage(t, &a, &b, &c, func() map[string]struct{} { return map[string]struct{}{"a": {}} }, out)
	_ = tr
	check := func(step string, want map[string]string) {
		t.Helper()
		if !maps.Equal(out, want) {
			t.Errorf("%s:\n got %v\nwant %v", step, out, want)
		}
	}
	check("mount", map[string]string{"a": "a: 100 [0]", "b": "b: 1 [0]", "c": "c: 2 [0]", "abc": "a: 100 b: 101 c: null"})
	steps := []struct {
		name string
		do   func()
		want map[string]string
	}{
		{"increment a", func() { a++ }, map[string]string{"a": "a: 101 [1]", "b": "b: 1 [0]", "c": "c: 2 [0]", "abc": "a: 101 b: 101 c: null"}},
		{"increment a again", func() { a++ }, map[string]string{"a": "a: 102 [2]", "b": "b: 1 [0]", "c": "c: 2 [0]", "abc": "a: 102 b: 101 c: null"}},
		{"increment b", func() { b++ }, map[string]string{"a": "a: 102 [2]", "b": "b: 2 [1]", "c": "c: 2 [0]", "abc": "a: 102 b: 102 c: null"}},
		{"increment c", func() { c++ }, map[string]string{"a": "a: 102 [2]", "b": "b: 2 [1]", "c": "c: 3 [1]", "abc": "a: 102 b: 102 c: null"}},
	}
	for _, s := range steps {
		s.do()
		rebuild()
		check(s.name, s.want)
	}
}

func TestInheritedModelInnerModelSupportedAspectChange(t *testing.T) {
	a, b, c := 0, 1, 2
	aspects := map[string]struct{}{"a": {}}
	out := map[string]string{}
	_, rebuild := shadowPage(t, &a, &b, &c, func() map[string]struct{} { return aspects }, out)
	check := func(step string, want map[string]string) {
		t.Helper()
		if !maps.Equal(out, want) {
			t.Errorf("%s:\n got %v\nwant %v", step, out, want)
		}
	}
	check("mount", map[string]string{"a": "a: 100 [0]", "b": "b: 1 [0]", "c": "c: 2 [0]", "abc": "a: 100 b: 101 c: null"})

	aspects = map[string]struct{}{"a": {}, "b": {}}
	rebuild()
	check("b is supported by the inner model", map[string]string{"a": "a: 100 [1]", "b": "b: 101 [1]", "c": "c: 2 [1]", "abc": "a: 100 b: 101 c: null"})
	a++
	rebuild()
	check("increment a", map[string]string{"a": "a: 101 [2]", "b": "b: 101 [1]", "c": "c: 2 [1]", "abc": "a: 101 b: 101 c: null"})
	b++
	rebuild()
	check("increment b", map[string]string{"a": "a: 101 [2]", "b": "b: 102 [2]", "c": "c: 2 [1]", "abc": "a: 101 b: 102 c: null"})
	c++
	rebuild()
	check("increment c", map[string]string{"a": "a: 101 [2]", "b": "b: 102 [2]", "c": "c: 3 [2]", "abc": "a: 101 b: 102 c: null"})

	aspects = map[string]struct{}{"a": {}, "b": {}, "c": {}}
	rebuild()
	check("all three are supported", map[string]string{"a": "a: 101 [3]", "b": "b: 102 [3]", "c": "c: null [3]", "abc": "a: 101 b: 102 c: null"})

	aspects = map[string]struct{}{}
	rebuild()
	check("none is supported", map[string]string{"a": "a: 1 [4]", "b": "b: 2 [4]", "c": "c: 3 [4]", "abc": "a: 101 b: 102 c: null"})

	aspects = nil
	rebuild()
	check("all are supported", map[string]string{"a": "a: 101 [5]", "b": "b: 102 [5]", "c": "c: null [5]", "abc": "a: 101 b: 102 c: null"})
}

func TestAspectSet(t *testing.T) {
	s := &AspectSet{}
	s.set.add("x")
	s.set.add(2)
	if !s.Contains("x") || s.Contains("y") || s.Len() != 2 {
		t.Errorf("AspectSet = %v", s.Items())
	}
	items := s.Items()
	items[0] = "changed"
	if !s.Contains("x") {
		t.Error("Items returned the inside of the set")
	}
}
