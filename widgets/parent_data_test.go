package widgets

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// stackW is Stack, positioned is Positioned and subclassPositioned is
// SubclassPositioned of parent_data_test.dart.
type stackW struct{ MultiChildRenderObjectWidget }

func newStack(children ...Widget) *stackW {
	return &stackW{NewMultiChildRenderObjectWidgetBase(children)}
}

func (w *stackW) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderStack(nil, rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR})
}

type rowW struct{ MultiChildRenderObjectWidget }

func newRow(children ...Widget) *rowW { return &rowW{NewMultiChildRenderObjectWidgetBase(children)} }

func (w *rowW) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderFlex(nil, rendering.RenderFlexOpts{Direction: painting.AxisHorizontal, TextDirection: ui.TextDirectionLTR})
}

type positioned struct {
	ParentDataWidgetOf[*rendering.StackParentData]
	left, top, right, bottom foundation.Opt[float64]
}

func newPositioned(child Widget, left, top, right, bottom foundation.Opt[float64]) *positioned {
	return &positioned{NewParentDataWidgetBase[*rendering.StackParentData](child), left, top, right, bottom}
}

func (w *positioned) ApplyParentData(ro rendering.RenderObject) {
	pd := ro.ParentData().(*rendering.StackParentData)
	pd.Left, pd.Top, pd.Right, pd.Bottom = w.left, w.top, w.right, w.bottom
	if p := ro.AsRenderObject().Parent(); p != nil {
		p.MarkNeedsLayout()
	}
}

func (w *positioned) DebugTypicalAncestorWidgetClass() string { return "Stack" }

func (w *positioned) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDoubleProperty("left", w.left, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewDoubleProperty("top", w.top, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewDoubleProperty("right", w.right, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewDoubleProperty("bottom", w.bottom, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// customPositioned is CustomPositionedWidget: another widget that writes the
// same parent data type.
type customPositioned struct {
	ParentDataWidgetOf[*rendering.StackParentData]
	bottom float64
}

func (w *customPositioned) ApplyParentData(ro rendering.RenderObject) {
	ro.ParentData().(*rendering.StackParentData).Bottom = foundation.Some(w.bottom)
}

func (w *customPositioned) DebugTypicalAncestorWidgetClass() string { return "Stack" }

// expanded is Expanded: it writes flex parent data.
type expanded struct {
	ParentDataWidgetOf[*rendering.FlexParentData]
}

func (w *expanded) ApplyParentData(ro rendering.RenderObject) {
	ro.ParentData().(*rendering.FlexParentData).Flex = foundation.Some(1)
}

func (w *expanded) DebugTypicalAncestorWidgetClass() string { return "Flex" }

func (w *expanded) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewIntProperty("flex", foundation.Some(1)))
}

type parentDataValues struct{ top, right, bottom, left foundation.Opt[float64] }

func checkStack(t *testing.T, rt *renderTree, want []parentDataValues) {
	t.Helper()
	stack := renderElementOfType[*stackW](t, rt).RenderObject().(*rendering.RenderStack)
	children := flexChildrenOf(stack)
	if len(children) != len(want) {
		t.Fatalf("%d children, want %d", len(children), len(want))
	}
	for i, c := range children {
		pd, ok := c.ParentData().(*rendering.StackParentData)
		if !ok {
			t.Fatalf("child %d has parent data %T", i, c.ParentData())
		}
		got := parentDataValues{pd.Top, pd.Right, pd.Bottom, pd.Left}
		if got != want[i] {
			t.Errorf("child %d: parent data = %v, want %v", i, got, want[i])
		}
	}
}

func flexChildrenOf(o rendering.RenderObject) []rendering.RenderObject {
	var out []rendering.RenderObject
	o.VisitChildren(func(c rendering.RenderObject) { out = append(out, c) })
	return out
}

func some(v float64) foundation.Opt[float64] { return foundation.Some(v) }

var none = foundation.None[float64]()

func TestParentDataWidgetControl(t *testing.T) {
	rt := newRenderTree(t)
	b := func() Widget { return newBox(1, 1) }
	steps := []struct {
		name string
		w    Widget
		want []parentDataValues
	}{
		{"one positioned", newStack(b(), newPositioned(b(), some(10), some(10), none, none), b()),
			[]parentDataValues{{}, {top: some(10), left: some(10)}, {}}},
		{"two positioned", newStack(newPositioned(b(), none, none, some(7), some(5)), newPositioned(b(), some(10), some(10), none, none), b()),
			[]parentDataValues{{bottom: some(5), right: some(7)}, {top: some(10), left: some(10)}, {}}},
		{"changed values", newStack(newPositioned(b(), none, none, some(8), some(6)), newPositioned(b(), some(10), none, some(10), none), b()),
			[]parentDataValues{{bottom: some(6), right: some(8)}, {left: some(10), right: some(10)}, {}}},
		{"positioned removed", newStack(b(), newPositioned(b(), some(10), some(10), none, none), b()),
			[]parentDataValues{{}, {top: some(10), left: some(10)}, {}}},
		{"empty", newStack(), nil},
	}
	for _, s := range steps {
		rt.pump(s.w)
		checkStack(t, rt, s.want)
		if err := rt.takeError(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
	}
}

func TestParentDataWidgetWithConstantChildren(t *testing.T) {
	// The children are the same widgets (constants in Flutter): the parent data is
	// still applied when only the parent data widget changes.
	rt := newRenderTree(t)
	a, c := newBox(1, 1), newBox(2, 2)
	rt.pump(newStack(newPositioned(a, none, none, some(7), some(5)), c))
	checkStack(t, rt, []parentDataValues{{bottom: some(5), right: some(7)}, {}})
	rt.pump(newStack(newPositioned(a, none, none, some(8), some(6)), c))
	checkStack(t, rt, []parentDataValues{{bottom: some(6), right: some(8)}, {}})
}

func TestParentDataOverwriteWithCompetingWidgets(t *testing.T) {
	skipUnlessDebug(t)
	rt := newRenderTree(t)
	rt.pump(newStack(&customPositioned{NewParentDataWidgetBase[*rendering.StackParentData](
		newPositioned(newBox(1, 1), some(7), some(6), none, none)), 8}))
	appErr := requireErrorTag(t, rt.takeError(), "ParentDataWidgetsCompeting")
	for _, want := range []string{
		"Incorrect use of ParentDataWidget.\nCompeting ParentDataWidgets are providing parent data to the same RenderObject:\n",
		"- positioned(left: 7.0, top: 6.0), which writes ParentData of type StackParentData, (typically placed directly inside a Stack widget)\n",
		"- customPositioned, which writes ParentData of type StackParentData, (typically placed directly inside a Stack widget)\n",
		"A RenderObject can receive parent data from multiple ParentDataWidgets, but the Type of ParentData must be unique to prevent one overwriting another.\n",
		"Otherwise, separating aspects of ParentData to prevent conflicts can be done using mixins, mixing them all in on the full ParentData Object, such as KeepAlive does with KeepAliveParentDataMixin.\n",
		"The ownership chain for the RenderObject that received the parent data was:\n  box ← positioned ← customPositioned ← stackW ← ",
	} {
		if !strings.Contains(appErr.Message, want) {
			t.Errorf("message does not contain %q:\n%s", want, appErr.Message)
		}
	}
}

func TestParentDataWidgetConflictingData(t *testing.T) {
	skipUnlessDebug(t)
	rt := newRenderTree(t)
	rt.pump(newStack(newPositioned(newPositioned(newBox(1, 1), some(7), some(6), none, none), none, some(5), none, some(8))))
	appErr := requireErrorTag(t, rt.takeError(), "ParentDataWidgetsCompeting")
	for _, want := range []string{
		"- positioned(left: 7.0, top: 6.0), which writes ParentData of type StackParentData",
		"- positioned(top: 5.0, bottom: 8.0), which writes ParentData of type StackParentData",
	} {
		if !strings.Contains(appErr.Message, want) {
			t.Errorf("message does not contain %q:\n%s", want, appErr.Message)
		}
	}
	// A tree with an empty stack is fine again.
	rt.pump(newStack())
	checkStack(t, rt, nil)
}

func TestParentDataInvalidAncestor(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("positioned in a row", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newWrapper(newRow(newPositioned(newBox(1, 1), some(7), some(6), none, none))))
		appErr := requireErrorTag(t, rt.takeError(), "ParentDataWidgetIncorrectUse")
		for _, want := range []string{
			"Incorrect use of ParentDataWidget.\nThe ParentDataWidget positioned(left: 7.0, top: 6.0) wants to apply ParentData of type StackParentData to a RenderObject, which has been set up to accept ParentData of incompatible type FlexParentData.\n",
			"Usually, this means that the positioned widget has the wrong ancestor RenderObjectWidget. Typically, positioned widgets are placed directly inside Stack widgets.\n",
			"The offending positioned is currently placed inside a rowW widget.\n",
			"The ownership chain for the RenderObject that received the incompatible parent data was:\n  box ← positioned ← rowW ← wrapper ← ",
		} {
			if !strings.Contains(appErr.Message, want) {
				t.Errorf("message does not contain %q:\n%s", want, appErr.Message)
			}
		}
		rt.pump(newStack())
		if err := rt.takeError(); err != nil {
			t.Errorf("a valid tree after the error reports %v", err)
		}
	})
	t.Run("expanded in a stack in a row", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newRow(newStack(&expanded{NewParentDataWidgetBase[*rendering.FlexParentData](newBox(1, 1))})))
		appErr := requireErrorTag(t, rt.takeError(), "ParentDataWidgetIncorrectUse")
		for _, want := range []string{
			"The ParentDataWidget expanded(flex: 1) wants to apply ParentData of type FlexParentData to a RenderObject, which has been set up to accept ParentData of incompatible type StackParentData.\n",
			"Typically, expanded widgets are placed directly inside Flex widgets.\n",
			"The offending expanded is currently placed inside a stackW widget.\n",
			"  box ← expanded ← stackW ← rowW ← ",
		} {
			if !strings.Contains(appErr.Message, want) {
				t.Errorf("message does not contain %q:\n%s", want, appErr.Message)
			}
		}
	})
	t.Run("a render view with plain parent data", func(t *testing.T) {
		rt := newRenderTree(t)
		// Positioned directly under the root: the render view gives its child plain
		// parent data.
		rt.pump(newPositioned(newBox(1, 1), some(1), none, none, none))
		appErr := requireErrorTag(t, rt.takeError(), "ParentDataWidgetIncorrectUse")
		if !strings.Contains(appErr.Message, "incompatible type ParentDataBase") {
			t.Errorf("message = %s", appErr.Message)
		}
	})
}

func TestParentDataInteractsWithGlobalKeys(t *testing.T) {
	key := NewGlobalKey[State]()
	rt := newRenderTree(t)
	steps := []Widget{
		newStack(newPositioned(newBox(1, 1, WidgetOpts{Key: key}), some(10), some(10), none, none)),
		newStack(newPositioned(newSized(2, 2, newBox(1, 1, WidgetOpts{Key: key})), some(10), some(10), none, none)),
		newStack(newPositioned(newBox(1, 1, WidgetOpts{Key: key}), some(10), some(10), none, none)),
	}
	for i, w := range steps {
		rt.pump(w)
		checkStack(t, rt, []parentDataValues{{top: some(10), left: some(10)}})
		if err := rt.takeError(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
}

// testPD / oneAncestor / anotherAncestor are DummyParentData, OneAncestorWidget
// and AnotherAncestorWidget: a parent data widget used under two different
// render object widgets.
type dummyParentData struct {
	rendering.ParentDataBase
	str foundation.Opt[string]
}

type oneRender struct{ rendering.RenderProxyBox }

func (r *oneRender) SetupParentData(child rendering.RenderObject) { setupDummyParentData(child) }

type anotherRender struct{ rendering.RenderProxyBox }

func (r *anotherRender) SetupParentData(child rendering.RenderObject) { setupDummyParentData(child) }

func setupDummyParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*dummyParentData); !ok {
		child.AsRenderObject().SetParentData(&dummyParentData{})
	}
}

type oneAncestor struct{ SingleChildRenderObjectWidget }

func newOneAncestor(child Widget) *oneAncestor {
	return &oneAncestor{NewSingleChildRenderObjectWidgetBase(child)}
}

func (w *oneAncestor) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &oneRender{}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

type anotherAncestor struct{ SingleChildRenderObjectWidget }

func newAnotherAncestor(child Widget) *anotherAncestor {
	return &anotherAncestor{NewSingleChildRenderObjectWidgetBase(child)}
}

func (w *anotherAncestor) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &anotherRender{}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

type testParentDataWidget struct {
	ParentDataWidgetOf[*dummyParentData]
	str string
}

func (w *testParentDataWidget) ApplyParentData(ro rendering.RenderObject) {
	ro.ParentData().(*dummyParentData).str = foundation.Some(w.str)
}

func (w *testParentDataWidget) DebugTypicalAncestorWidgetClass() string { return "oneAncestor" }

func TestParentDataWidgetWithDifferentAncestorRenderObjectWidgets(t *testing.T) {
	rt := newRenderTree(t)
	str := func() foundation.Opt[string] {
		return renderElementOfType[*box](t, rt).RenderObject().ParentData().(*dummyParentData).str
	}
	tpd := func(s string, child Widget) *testParentDataWidget {
		return &testParentDataWidget{NewParentDataWidgetBase[*dummyParentData](child), s}
	}
	rt.pump(newOneAncestor(newBox(1, 1)))
	if str().IsSome() {
		t.Errorf("without the widget the data is %v", str())
	}
	rt.pump(newOneAncestor(tpd("Foo", newBox(1, 1))))
	if got, _ := str().Get(); got != "Foo" {
		t.Errorf("data = %q, want Foo", got)
	}
	rt.pump(newAnotherAncestor(tpd("Bar", newBox(1, 1))))
	if got, _ := str().Get(); got != "Bar" {
		t.Errorf("data = %q, want Bar", got)
	}
	if err := rt.takeError(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyWidgetOutOfTurn(t *testing.T) {
	skipUnlessDebug(t)
	rt := newRenderTree(t)
	child := newBox(1, 1)
	rt.pump(newStack(newPositioned(child, some(1), none, none, none)))
	el := renderElementOfType[*positioned](t, rt).(*ParentDataElement)
	// The widget does not say it can be applied out of turn.
	requireTag(t, "ParentDataOutOfTurnNotAllowed", func() {
		el.ApplyWidgetOutOfTurn(newPositioned(child, some(2), none, none, none))
	})
	ok := &outOfTurn{positioned: *newPositioned(child, some(2), none, none, none)}
	el.ApplyWidgetOutOfTurn(ok)
	checkStack(t, rt, []parentDataValues{{left: some(2)}})
	other := &outOfTurn{positioned: *newPositioned(newBox(5, 5), some(3), none, none, none)}
	requireTag(t, "ParentDataOutOfTurnChildChanged", func() { el.ApplyWidgetOutOfTurn(other) })
}

type outOfTurn struct{ positioned }

func (*outOfTurn) DebugCanApplyOutOfTurn() bool { return true }
