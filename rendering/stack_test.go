package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func stackData(b rendering.Box) *rendering.StackParentData {
	return b.ParentData().(*rendering.StackParentData)
}

func tightFor(w, h foundation.Opt[float64]) rendering.BoxConstraints {
	return rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: w, Height: h})
}

func TestStackParentDataBasic(t *testing.T) {
	d := &rendering.StackParentData{}
	stackSize := ui.Size{Width: 800, Height: 600}
	if d.IsPositioned() {
		t.Fatal("a new parent data is positioned")
	}
	none := foundation.None[float64]()
	steps := []struct {
		name  string
		apply func()
		want  rendering.BoxConstraints
	}{
		{"negative width", func() { d.Width = some(-100) }, tightFor(some(0), none)},
		{"width", func() { d.Width = some(100) }, tightFor(some(100), none)},
		{"left and right", func() { d.Left, d.Right = some(0), some(0) }, tightFor(some(800), none)},
		{"negative height", func() { d.Height = some(-100) }, tightFor(some(800), some(0))},
		{"height", func() { d.Height = some(100) }, tightFor(some(800), some(100))},
		{"top and bottom", func() { d.Top, d.Bottom = some(0), some(0) }, tightFor(some(800), some(600))},
		{"big bottom", func() { d.Bottom = some(1000) }, tightFor(some(800), some(0))},
	}
	for _, s := range steps {
		s.apply()
		if !d.IsPositioned() {
			t.Errorf("%s: not positioned", s.name)
		}
		if got := d.PositionedChildConstraints(stackSize); got != s.want {
			t.Errorf("%s: constraints = %v, want %v", s.name, got, s.want)
		}
	}
}

func TestStackParentDataStringAndRect(t *testing.T) {
	d := &rendering.StackParentData{}
	if got, want := d.String(), "not positioned; offset=Offset(0.0, 0.0)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	d.SetRect(rendering.RelativeRect{Left: 1, Top: 2, Right: 3, Bottom: 4})
	d.Width = some(5)
	if got, want := d.String(), "top=2.0; right=3.0; bottom=4.0; left=1.0; width=5.0; offset=Offset(0.0, 0.0)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got := d.Rect(); got != (rendering.RelativeRect{Left: 1, Top: 2, Right: 3, Bottom: 4}) {
		t.Errorf("Rect() = %v", got)
	}
}

func TestStackLayoutWithAllEdgesZero(t *testing.T) {
	size := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 100}), nil)
	red := rendering.NewRenderProxyBox(size)
	green := rendering.NewRenderProxyBox(nil)
	stack := rendering.NewRenderStack([]rendering.Box{red, green}, rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR})
	d := stackData(green)
	d.Top, d.Right, d.Bottom, d.Left = some(0), some(0), some(0), some(0)
	layout(t, newHarness(t), stack, withConstraints(rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}))
	for name, got := range map[string]ui.Size{"stack": stack.Size(), "red": red.Size(), "green": green.Size()} {
		if got != (ui.Size{Width: 100, Height: 100}) {
			t.Errorf("%s size = %v, want 100x100", name, got)
		}
	}
}

func TestStackLayoutWithNoChildren(t *testing.T) {
	stack := rendering.NewRenderStack(nil, rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR})
	layout(t, newHarness(t), stack, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 100})))
	if stack.Size() != (ui.Size{Width: 100, Height: 100}) {
		t.Errorf("size = %v, want 100x100", stack.Size())
	}
}

func TestStackInFlexWithNoChildren(t *testing.T) {
	flex := rendering.NewRenderFlex([]rendering.Box{rendering.NewRenderStack(nil, rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR})},
		rendering.RenderFlexOpts{TextDirection: ui.TextDirectionLTR, MainAxisAlignment: rendering.MainAxisAlignmentSpaceEvenly})
	h := newHarness(t)
	layout(t, h, flex, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 100})))
	if len(h.Errors) != 0 {
		t.Errorf("errors = %v", h.Errors)
	}
}

func TestStackHasCorrectClipBehavior(t *testing.T) {
	for _, tc := range clipCases(ui.ClipHardEdge) {
		t.Run(tc.clip.String(), func(t *testing.T) {
			child := newSizedBox(200, 200)
			stack := rendering.NewRenderStack([]rendering.Box{child}, rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR, ClipBehavior: tc.clip})
			// The child is positioned so that the stack considers it overflowing.
			d := stackData(child)
			d.Left, d.Right = some(0), some(0)
			h := newHarness(t)
			layout(t, h, stack, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseComposite))
			if len(h.Errors) != 0 {
				t.Errorf("errors = %v", h.Errors)
			}
			if got := observedClip(t, h, stack); got != tc.want {
				t.Errorf("clip = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStackLayoutAlignmentAndPositions(t *testing.T) {
	tests := []struct {
		name      string
		alignment painting.AlignmentGeometry
		td        ui.TextDirection
		positions func(d *rendering.StackParentData)
		want      ui.Offset
	}{
		{"topStart ltr", nil, ui.TextDirectionLTR, func(*rendering.StackParentData) {}, off(0, 0)},
		{"topStart rtl", nil, ui.TextDirectionRTL, func(*rendering.StackParentData) {}, off(200, 0)},
		{"center", painting.AlignmentCenter, ui.TextDirectionLTR, func(*rendering.StackParentData) {}, off(100, 100)},
		{"left and top", painting.AlignmentCenter, ui.TextDirectionLTR, func(d *rendering.StackParentData) { d.Left, d.Top = some(10), some(20) }, off(10, 20)},
		{"right and bottom", painting.AlignmentCenter, ui.TextDirectionLTR, func(d *rendering.StackParentData) { d.Right, d.Bottom = some(10), some(20) }, off(190, 180)},
		{"width only is aligned", painting.AlignmentBottomRight, ui.TextDirectionLTR, func(d *rendering.StackParentData) { d.Width = some(50) }, off(250, 200)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			big := newSizedBox(300, 300)
			child := newSizedBox(100, 100)
			stack := rendering.NewRenderStack([]rendering.Box{big, child}, rendering.RenderStackOpts{Alignment: tc.alignment, TextDirection: tc.td})
			tc.positions(stackData(child))
			layout(t, newHarness(t), stack, withConstraints(rendering.BoxConstraints{MaxWidth: 300, MaxHeight: 300}))
			if got := offsetOf(child); got != tc.want {
				t.Errorf("child offset = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStackFit(t *testing.T) {
	tests := []struct {
		fit  rendering.StackFit
		want ui.Size
	}{
		{rendering.StackFitUnspecified, ui.Size{Width: 100, Height: 100}},
		{rendering.StackFitLoose, ui.Size{Width: 100, Height: 100}},
		{rendering.StackFitExpand, ui.Size{Width: 300, Height: 200}},
		{rendering.StackFitPassthrough, ui.Size{Width: 100, Height: 100}},
	}
	for _, tc := range tests {
		t.Run(tc.fit.String(), func(t *testing.T) {
			stack := rendering.NewRenderStack([]rendering.Box{newSizedBox(100, 100)}, rendering.RenderStackOpts{Fit: tc.fit, TextDirection: ui.TextDirectionLTR})
			layout(t, newHarness(t), stack, withConstraints(rendering.BoxConstraints{MaxWidth: 300, MaxHeight: 200}))
			if stack.Size() != tc.want {
				t.Errorf("stack size = %v, want %v", stack.Size(), tc.want)
			}
		})
	}
}

func TestStackIntrinsicsIgnorePositionedChildren(t *testing.T) {
	positioned := newSizedBox(500, 500)
	stack := rendering.NewRenderStack([]rendering.Box{newTBox(40, 30), newTBox(10, 70), positioned})
	stackData(positioned).Left = some(0)
	if got := stack.GetMinIntrinsicWidth(inf); got != 40 {
		t.Errorf("min intrinsic width = %v, want 40", got)
	}
	if got := stack.GetMaxIntrinsicHeight(inf); got != 70 {
		t.Errorf("max intrinsic height = %v, want 70", got)
	}
}

func TestStackDryLayoutAndBaseline(t *testing.T) {
	skipUnlessDebug(t)
	c1 := newBaselineGridBox(1, atTen)
	c2 := newBaselineGridBox(3, noBaseline)
	stack := rendering.NewRenderStack([]rendering.Box{c1, c2}, rendering.RenderStackOpts{Alignment: painting.AlignmentTopLeft})
	cons := rendering.NewBoxConstraintsLoose(ui.Size{Width: 200, Height: 100})
	if got := stack.GetDryLayout(cons); got != (ui.Size{Width: 30, Height: 10}) {
		t.Errorf("dry layout = %v", got)
	}
	if got := stack.GetDryBaseline(cons, ui.TextBaselineAlphabetic); got != foundation.Some(10.0) {
		t.Errorf("dry baseline = %v, want 10", got)
	}
}

func TestStackUnboundedConstraintsPanics(t *testing.T) {
	skipUnlessDebug(t)
	child := newTBox(10, 10)
	stack := rendering.NewRenderStack([]rendering.Box{child}, rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR})
	stackData(child).Left = some(0)
	parent := rendering.NewRenderConstrainedOverflowBox(stack, rendering.RenderConstrainedOverflowBoxOpts{
		MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(0), MaxHeight: some(inf)})
	requireTag(t, "StackUnboundedConstraints", func() { layout(t, newHarness(t), parent) })
}

func TestRenderIndexedStack(t *testing.T) {
	mk := func() []rendering.Box {
		return []rendering.Box{newSizedBox(100, 100), newSizedBox(100, 100), newSizedBox(100, 100)}
	}
	t.Run("paints and hit tests the displayed child only", func(t *testing.T) {
		children := mk()
		stack := rendering.NewRenderIndexedStack(children, rendering.RenderIndexedStackOpts{
			RenderStackOpts: rendering.RenderStackOpts{TextDirection: ui.TextDirectionLTR}, Index: foundation.Some(1)})
		h := newHarness(t)
		layout(t, h, stack, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 100})), phasePaint))
		if hidden := children[0].(*tBox); hidden.paints != 0 || children[1].(*tBox).paints != 1 {
			t.Errorf("paints: hidden %d, displayed %d, want 0 and 1", hidden.paints, children[1].(*tBox).paints)
		}
		res := rendering.NewBoxHitTestResult()
		if !stack.HitTest(res, ui.Offset{Dx: 10, Dy: 10}) {
			t.Fatal("hit test missed")
		}
		var hit []any
		for _, e := range res.Path() {
			hit = append(hit, e.Target())
		}
		if len(hit) == 0 || hit[0] != any(children[1]) {
			t.Errorf("hit targets = %v, want the displayed child first", hit)
		}
		stack.SetIndex(foundation.None[int]())
		if stack.HitTestChildren(rendering.NewBoxHitTestResult(), ui.Offset{Dx: 10, Dy: 10}) {
			t.Error("a null index was hit")
		}
	})
	t.Run("debugDescribeChildren marks invisible children as offstage", func(t *testing.T) {
		skipUnlessDebug(t)
		tests := []struct {
			name  string
			index foundation.Opt[int]
			want  [3]foundation.DiagnosticsTreeStyle
		}{
			{"index 2", foundation.Some(2), [3]foundation.DiagnosticsTreeStyle{foundation.DiagnosticsTreeStyleOffstage, foundation.DiagnosticsTreeStyleOffstage, foundation.DiagnosticsTreeStyleSparse}},
			{"null index", foundation.None[int](), [3]foundation.DiagnosticsTreeStyle{foundation.DiagnosticsTreeStyleOffstage, foundation.DiagnosticsTreeStyleOffstage, foundation.DiagnosticsTreeStyleOffstage}},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				stack := rendering.NewRenderIndexedStack(mk(), rendering.RenderIndexedStackOpts{Index: tc.index})
				stack.SetIndex(tc.index)
				nodes := stack.DebugDescribeChildren()
				if len(nodes) != 3 {
					t.Fatalf("%d nodes", len(nodes))
				}
				for i, n := range nodes {
					if name := n.Name().Value(); name != "child "+string(rune('1'+i)) {
						t.Errorf("node %d name = %q", i, name)
					}
					style := n.Style()
					if style == foundation.DiagnosticsTreeStyleUnspecified {
						style = foundation.DiagnosticsTreeStyleSparse
					}
					if style != tc.want[i] {
						t.Errorf("node %d style = %v, want %v", i, style, tc.want[i])
					}
				}
			})
		}
	})
	t.Run("default index is 0", func(t *testing.T) {
		stack := rendering.NewRenderIndexedStack(mk())
		if stack.Index() != foundation.Some(0) {
			t.Errorf("index = %v, want 0", stack.Index())
		}
	})
}

func TestLayoutPositionedStackChildNeedsPositionedChild(t *testing.T) {
	skipUnlessDebug(t)
	child := newTBox(10, 10)
	rendering.NewRenderStack([]rendering.Box{child})
	requireTag(t, "StackChildNotPositioned", func() {
		rendering.LayoutPositionedStackChild(child, stackData(child), ui.Size{Width: 100, Height: 100}, painting.AlignmentCenter)
	})
}
