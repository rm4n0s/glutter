package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// multiLayoutDelegate lays out the children by id: the "a" child is placed at
// the origin, the "b" child under it. The flags break the protocol.
type multiLayoutDelegate struct {
	rendering.MultiChildLayoutDelegateBase
	skipB, layoutTwice, unknownChild, position bool
	relayout                                   bool
	layouts                                    int
}

func newMultiLayoutDelegate(relayout foundation.Listenable) *multiLayoutDelegate {
	return &multiLayoutDelegate{MultiChildLayoutDelegateBase: rendering.NewMultiChildLayoutDelegateBase(relayout), position: true}
}

func (d *multiLayoutDelegate) PerformLayout(size ui.Size) {
	d.layouts++
	loose := rendering.NewBoxConstraintsLoose(size)
	a := d.LayoutChild("a", loose)
	if d.position {
		d.PositionChild("a", ui.OffsetZero)
	}
	if d.layoutTwice {
		d.LayoutChild("a", loose)
	}
	if d.unknownChild {
		d.LayoutChild("zzz", loose)
	}
	if !d.skipB && d.HasChild("b") {
		d.LayoutChild("b", loose)
		d.PositionChild("b", ui.Offset{Dy: a.Height})
	}
}
func (d *multiLayoutDelegate) ShouldRelayout(rendering.MultiChildLayoutDelegate) bool {
	return d.relayout
}

func newLayoutBox(delegate rendering.MultiChildLayoutDelegate, ids ...any) (*rendering.RenderCustomMultiChildLayoutBox, []rendering.Box) {
	box := rendering.NewRenderCustomMultiChildLayoutBox(delegate, nil)
	var children []rendering.Box
	for i, id := range ids {
		c := newTBox(float64(10*(i+1)), 20)
		box.SetupParentData(c)
		c.ParentData().(*rendering.MultiChildLayoutParentData).ID = id
		box.Add(c)
		children = append(children, c)
	}
	return box, children
}

func TestCustomMultiChildLayout(t *testing.T) {
	delegate := newMultiLayoutDelegate(nil)
	box, children := newLayoutBox(delegate, "a", "b")
	layout(t, newHarness(t), box, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
	if box.Size() != (ui.Size{Width: 100, Height: 100}) {
		t.Errorf("size = %v, want 100x100", box.Size())
	}
	if got := offsetOf(children[0]); got != ui.OffsetZero {
		t.Errorf("a at %v", got)
	}
	if got := offsetOf(children[1]); got != off(0, 20) {
		t.Errorf("b at %v, want (0, 20)", got)
	}
	if children[1].Size() != (ui.Size{Width: 20, Height: 20}) {
		t.Errorf("b size = %v", children[1].Size())
	}
	if got := children[0].ParentData().(*rendering.MultiChildLayoutParentData).String(); got != "offset=Offset(0.0, 0.0); id=a" {
		t.Errorf("parent data = %q", got)
	}
	if got := box.GetMinIntrinsicWidth(40); got != 0 && got != 40 {
		t.Errorf("intrinsic width = %v", got)
	}
}

func TestCustomMultiChildLayoutErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name   string
		tag    string
		mutate func(d *multiLayoutDelegate)
		ids    []any
	}{
		{"child laid out twice", "MultiChildLayoutChildTwice", func(d *multiLayoutDelegate) { d.layoutTwice = true }, []any{"a", "b"}},
		{"unknown child", "MultiChildLayoutNoSuchChild", func(d *multiLayoutDelegate) { d.unknownChild = true }, []any{"a", "b"}},
		{"forgotten child", "MultiChildLayoutChildNotLaidOut", func(d *multiLayoutDelegate) { d.skipB = true }, []any{"a", "b"}},
		{"missing id", "MultiChildLayoutMissingID", func(*multiLayoutDelegate) {}, []any{"a", nil}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			delegate := newMultiLayoutDelegate(nil)
			tc.mutate(delegate)
			box, _ := newLayoutBox(delegate, tc.ids...)
			requireTag(t, tc.tag, func() {
				layout(t, newHarness(t), box, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
			})
		})
	}
	t.Run("position of an unknown child", func(t *testing.T) {
		d := newMultiLayoutDelegate(nil)
		box, _ := newLayoutBox(d, "a")
		requireTag(t, "MultiChildLayoutNoSuchChild", func() {
			d.Base().PositionChild("nope", ui.OffsetZero)
			_ = box
		})
	})
}

func TestCustomMultiChildLayoutDelegateChanges(t *testing.T) {
	notifier := foundation.NewChangeNotifier()
	first := newMultiLayoutDelegate(notifier)
	box, _ := newLayoutBox(first, "a")
	h := newHarness(t)
	layout(t, h, box, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
	if first.layouts != 1 {
		t.Fatalf("layouts = %d, want 1", first.layouts)
	}
	notifier.NotifyListeners()
	pumpFrame(h, phaseLayout)
	if first.layouts != 2 {
		t.Errorf("layouts after the relayout listenable notified = %d, want 2", first.layouts)
	}
	same := newMultiLayoutDelegate(nil)
	box.SetDelegate(same)
	pumpFrame(h, phaseLayout)
	if same.layouts != 0 {
		t.Errorf("a delegate that does not ask for relayout laid the box out %d times", same.layouts)
	}
	notifier.NotifyListeners()
	pumpFrame(h, phaseLayout)
	if first.layouts != 2 {
		t.Errorf("the old delegate is still listened to (%d layouts)", first.layouts)
	}
	again := newMultiLayoutDelegate(nil)
	again.relayout = true
	box.SetDelegate(again)
	pumpFrame(h, phaseLayout)
	if again.layouts != 1 {
		t.Errorf("a delegate that asks for relayout laid the box out %d times, want 1", again.layouts)
	}
}
