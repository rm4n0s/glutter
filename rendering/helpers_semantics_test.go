package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// semNode is the render objects of the semantics tests of Flutter in one:
// RenderTestParent, RenderTestLayoutSemanticsBoundary and TestRender. With a
// child it is as big as it can be; without one it has the target size.
type semNode struct {
	rendering.RenderBox
	rendering.RenderObjectWithChild[rendering.Box]

	targetSize                        ui.Size
	usesSize                          bool
	boundary, explicit, blocking      bool
	label                             string
	tap, longPress, scrollLeft        bool
	scrollRight, scrollUp, scrollDown bool
	describeCalls                     int
}

func newSemNode(child rendering.Box) *semNode {
	n := &semNode{targetSize: ui.Size{Width: 100, Height: 100}}
	rendering.InitRenderBox(&n.RenderBox, n)
	rendering.InitRenderObjectWithChild(&n.RenderObjectWithChild, n)
	if child != nil {
		n.SetChild(child)
	}
	return n
}

func (n *semNode) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*rendering.BoxParentData); !ok {
		child.AsRenderObject().SetParentData(&rendering.BoxParentData{})
	}
}

func (n *semNode) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	if n.Child() == nil {
		return c.Constrain(n.targetSize)
	}
	return c.Biggest()
}

func (n *semNode) PerformLayout() {
	c := n.BoxConstraints()
	if ch := n.Child(); ch != nil {
		ch.Layout(c.Loosen(), rendering.LayoutOpts{ParentUsesSize: n.usesSize})
		n.SetSize(c.Biggest())
		return
	}
	n.SetSize(c.Constrain(n.targetSize))
}

func (n *semNode) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	if ch := n.Child(); ch != nil {
		ctx.PaintChild(ch, offset)
	}
}

func (n *semNode) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	if ch := n.Child(); ch != nil {
		return ch.HitTest(result, position)
	}
	return false
}

func (n *semNode) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	n.RenderBox.DescribeSemanticsConfiguration(config)
	n.describeCalls++
	config.SetIsSemanticBoundary(n.boundary)
	config.ExplicitChildNodes = n.explicit
	config.IsBlockingSemanticsOfPreviouslyPaintedNodes = n.blocking
	if n.label != "" {
		config.SetLabel(n.label)
		config.SetTextDirection(ui.TextDirectionLTR)
	}
	noop := func() {}
	if n.tap {
		config.SetOnTap(noop)
	}
	if n.longPress {
		config.SetOnLongPress(noop)
	}
	if n.scrollLeft {
		config.SetOnScrollLeft(noop)
	}
	if n.scrollRight {
		config.SetOnScrollRight(noop)
	}
	if n.scrollUp {
		config.SetOnScrollUp(noop)
	}
	if n.scrollDown {
		config.SetOnScrollDown(noop)
	}
}

// semMulti is RenderTestMultiChildParent and
// RenderTestLastChildSemanticsMultiChildParent.
type semMulti struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	explicit, lastChildOnly bool
}

type semMultiData struct {
	rendering.ContainerBoxParentData[rendering.Box]
}

func newSemMulti(children ...rendering.Box) *semMulti {
	m := &semMulti{}
	rendering.InitRenderBox(&m.RenderBox, m)
	rendering.InitContainerRenderObject(&m.ContainerRenderObject, m)
	m.AddAll(children)
	return m
}

func (m *semMulti) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*semMultiData); !ok {
		child.AsRenderObject().SetParentData(&semMultiData{})
	}
}

func (m *semMulti) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }

func (m *semMulti) PerformLayout() {
	c := m.BoxConstraints()
	for ch := m.FirstChild(); ch != nil; ch = ch.ParentData().(*semMultiData).NextSibling {
		ch.Layout(c.Loosen())
	}
	m.SetSize(c.Biggest())
}

func (m *semMulti) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	for ch := m.FirstChild(); ch != nil; ch = ch.ParentData().(*semMultiData).NextSibling {
		ctx.PaintChild(ch, offset.Add(rendering.BoxParentDataOf(ch).Offset))
	}
}

func (m *semMulti) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return false
}

func (m *semMulti) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	m.RenderBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
	config.ExplicitChildNodes = m.explicit
	config.SetLabel("Test Parent")
	config.SetTextDirection(ui.TextDirectionLTR)
}

func (m *semMulti) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	if m.lastChildOnly {
		if last := m.LastChild(); last != nil {
			visitor(last)
		}
		return
	}
	m.RenderBox.VisitChildrenForSemantics(visitor)
}

// withSemantics makes the harness produce semantics for the test.
func withSemantics(t *testing.T) (*renderingtest.Harness, *semantics.SemanticsOwner) {
	t.Helper()
	skipUnlessDebug(t) // DebugSemantics exists in debug builds only
	h := newHarness(t)
	handle := h.Binding.SemanticsBinding().EnsureSemantics()
	t.Cleanup(handle.Dispose)
	return h, h.Owner.SemanticsOwner()
}

func countSemanticsChildren(object rendering.RenderObject) int {
	count := 0
	object.VisitChildrenForSemantics(func(rendering.RenderObject) { count++ })
	return count
}

func debugRect(t *testing.T, o rendering.RenderObject) ui.Rect {
	t.Helper()
	node := o.AsRenderObject().DebugSemantics()
	if node == nil {
		t.Fatalf("%T has no semantics node", o)
	}
	return node.Rect()
}

var _ = foundation.DebugMode
