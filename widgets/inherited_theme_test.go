package widgets

import "testing"

// themeA and themeB are inherited themes that wrap by making a copy of
// themselves around the child.
type themeA struct {
	InheritedTheme
	name string
}

func newThemeA(name string, child Widget) *themeA {
	return &themeA{NewInheritedThemeBase(child), name}
}

func (w *themeA) UpdateShouldNotify(old Widget) bool       { return old.(*themeA).name != w.name }
func (w *themeA) Wrap(_ BuildContext, child Widget) Widget { return newThemeA(w.name, child) }

type themeB struct {
	InheritedTheme
	name string
}

func newThemeB(name string, child Widget) *themeB {
	return &themeB{NewInheritedThemeBase(child), name}
}

func (w *themeB) UpdateShouldNotify(old Widget) bool       { return old.(*themeB).name != w.name }
func (w *themeB) Wrap(_ BuildContext, child Widget) Widget { return newThemeB(w.name, child) }

func TestCaptureThemes(t *testing.T) {
	var inner, outerCtx BuildContext
	var captured Widget
	var capturedTo Widget
	tr := newTree(t)
	tr.pump(newThemeA("outer", newThemeB("b", builderOf(func(bc BuildContext) Widget {
		outerCtx = bc
		return newThemeA("inner", builderOf(func(bc BuildContext) Widget {
			inner = bc
			return newLeaf()
		}))
	}))))
	// Everything between the inner context and the root: the inner theme A
	// shadows the outer one.
	captured = CaptureAllThemes(inner, newLeaf(), nil)
	tr2 := newTree(t)
	var names []string
	tr2.pump(captured)
	for _, e := range tr2.find(func(w Widget) bool { return true }) {
		switch w := e.AsElement().widget.(type) {
		case *themeA:
			names = append(names, "A:"+w.name)
		case *themeB:
			names = append(names, "B:"+w.name)
		}
	}
	// The order is the order the themes were found going up, wrapped from the
	// closest to the farthest: the farthest ends up outermost.
	if len(names) != 2 || names[0] != "B:b" || names[1] != "A:inner" {
		t.Errorf("captured themes = %v", names)
	}

	// With a "to" context the capture stops there.
	capturedTo = CaptureAllThemes(inner, newLeaf(), outerCtx)
	tr3 := newTree(t)
	names = nil
	tr3.pump(capturedTo)
	for _, e := range tr3.find(func(w Widget) bool { return true }) {
		if w, ok := e.AsElement().widget.(*themeA); ok {
			names = append(names, w.name)
		}
	}
	if len(names) != 1 || names[0] != "inner" {
		t.Errorf("themes captured up to the outer context = %v", names)
	}

	// Nothing to capture from a context to itself.
	if got := CaptureThemes(inner, inner); len(got.themes) != 0 {
		t.Errorf("captured %d themes from a context to itself", len(got.themes))
	}
}

func TestCaptureThemesToNotAnAncestor(t *testing.T) {
	skipUnlessDebug(t)
	var a, b BuildContext
	tr := newTree(t)
	tr.pump(newMulti(
		builderOf(func(bc BuildContext) Widget { a = bc; return newLeaf() }),
		builderOf(func(bc BuildContext) Widget { b = bc; return newLeaf() }),
	))
	requireTag(t, "CaptureToNotAncestor", func() { CaptureThemes(a, b) })
}
