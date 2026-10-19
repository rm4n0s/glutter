package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func labelled(label string) Widget {
	return NewSemantics(newBox(20, 10), SemanticsOpts{Label: foundation.Some(label)})
}

func TestSemanticsWidgetsLabels(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name   string
		w      Widget
		found  []string
		absent []string
	}{
		{"semantics", labelled("a"), []string{"a"}, nil},
		{"exclude", NewExcludeSemantics(labelled("a")), nil, []string{"a"}},
		{"exclude, not excluding", NewExcludeSemantics(labelled("a"), ExcludeSemanticsOpts{Excluding: foundation.Some(false)}), []string{"a"}, nil},
		{"two separate nodes", NewColumn([]Widget{labelled("a"), labelled("b")}), []string{"a", "b"}, nil},
		{"merge", NewMergeSemantics(NewColumn([]Widget{labelled("a"), labelled("b")})), []string{"a\nb"}, []string{"a", "b"}},
		{"block hides what is below", NewStack([]Widget{labelled("below"), NewBlockSemantics(labelled("above"))}), []string{"above"}, []string{"below"}},
		{"block, not blocking", NewStack([]Widget{labelled("below"), NewBlockSemantics(labelled("above"), BlockSemanticsOpts{Blocking: foundation.Some(false)})}),
			[]string{"above", "below"}, nil},
		{"from properties", NewSemanticsFromProperties(semantics.NewSemanticsProperties(semantics.SemanticsProperties{Label: foundation.Some("props")}), newBox(5, 5)),
			[]string{"props"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.EnsureSemantics()
			pumpLoose(ts, tt.w, ui.TextDirectionLTR)
			for _, l := range tt.found {
				if !ts.Exists(gluttertest.BySemanticsLabel(l)) {
					t.Errorf("no node with the label %q", l)
				}
			}
			for _, l := range tt.absent {
				if ts.Exists(gluttertest.BySemanticsLabel(l)) {
					t.Errorf("found a node with the label %q", l)
				}
			}
		})
	}
}

func TestSemanticsFlagsAndContainer(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.EnsureSemantics()
	pumpLoose(ts, NewSemantics(newBox(20, 10), SemanticsOpts{Container: true, Button: foundation.Some(true), Label: foundation.Some("go")}), ui.TextDirectionLTR)
	node := ts.RenderObject(gluttertest.BySemanticsLabel("go")).AsRenderObject().DebugSemantics()
	if node == nil || !node.FlagsCollection().IsButton {
		t.Fatalf("node = %v, want a button", node)
	}
}

func TestSemanticsUpdatesItsRenderObject(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.EnsureSemantics()
	pumpLoose(ts, labelled("one"), ui.TextDirectionLTR)
	pumpLoose(ts, labelled("two"), ui.TextDirectionLTR)
	if ts.Exists(gluttertest.BySemanticsLabel("one")) || !ts.Exists(gluttertest.BySemanticsLabel("two")) {
		t.Error("the label did not change")
	}
}

func TestIndexedSemantics(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewIndexedSemantics(3, newLeaf()), ui.TextDirectionLTR)
	r := ts.RenderObject(gluttertest.ByType[*IndexedSemantics]()).(*rendering.RenderIndexedSemantics)
	if r.Index() != 3 {
		t.Errorf("index = %d, want 3", r.Index())
	}
	pumpLoose(ts, NewIndexedSemantics(4, newLeaf()), ui.TextDirectionLTR)
	if r.Index() != 4 {
		t.Errorf("index = %d after the update, want 4", r.Index())
	}
}

func TestListenerReceivesPointerEvents(t *testing.T) {
	tests := []struct {
		name string
		act  func(ts *gluttertest.Tester, at ui.Offset)
		want []string
	}{
		{"down, move and up", func(ts *gluttertest.Tester, at ui.Offset) {
			g := ts.StartGesture(at)
			g.MoveBy(ui.Offset{Dx: 5})
			g.Up()
		}, []string{"down", "move", "up"}},
		{"cancel", func(ts *gluttertest.Tester, at ui.Offset) { ts.StartGesture(at).Cancel() }, []string{"down", "cancel"}},
		{"hover", func(ts *gluttertest.Tester, at ui.Offset) { ts.Hover(at) }, []string{"hover"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var log []string
			rec := func(s string) { log = append(log, s) }
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewListener(NewColoredBox(ui.NewColor(0xFF000000), newBox(50, 50)), ListenerOpts{
				OnPointerDown:   func(*gestures.PointerDownEvent) { rec("down") },
				OnPointerMove:   func(*gestures.PointerMoveEvent) { rec("move") },
				OnPointerUp:     func(*gestures.PointerUpEvent) { rec("up") },
				OnPointerCancel: func(*gestures.PointerCancelEvent) { rec("cancel") },
				OnPointerHover:  func(*gestures.PointerHoverEvent) { rec("hover") },
			}), ui.TextDirectionLTR)
			tt.act(ts, ts.Center(gluttertest.ByType[*Listener]()))
			if len(log) != len(tt.want) {
				t.Fatalf("log = %v, want %v", log, tt.want)
			}
			for i := range log {
				if log[i] != tt.want[i] {
					t.Errorf("log = %v, want %v", log, tt.want)
					break
				}
			}
		})
	}
}

func TestListenerBehavior(t *testing.T) {
	tests := []struct {
		name     string
		behavior rendering.HitTestBehavior
		child    Widget
		want     int
	}{
		{"defer to child without a hittable child", rendering.HitTestBehaviorUnspecified, newBox(50, 50), 0},
		{"opaque", rendering.HitTestBehaviorOpaque, newBox(50, 50), 1},
		{"translucent", rendering.HitTestBehaviorTranslucent, newBox(50, 50), 1},
		{"defer to a hittable child", rendering.HitTestBehaviorDeferToChild, NewColoredBox(ui.NewColor(0xFF000000), newBox(50, 50)), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downs := 0
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewListener(tt.child, ListenerOpts{Behavior: tt.behavior, OnPointerDown: func(*gestures.PointerDownEvent) { downs++ }}), ui.TextDirectionLTR)
			ts.TapAt(ui.Offset{Dx: 10, Dy: 10})
			if downs != tt.want {
				t.Errorf("downs = %d, want %d", downs, tt.want)
			}
		})
	}
}

// TestMouseRegionHover checks the hover callback, which reaches the region
// through the pointer events. The enter and exit callbacks come from the mouse
// tracker, which the bindings get with the fallback cursor (step 10.4).
func TestMouseRegionHover(t *testing.T) {
	var hovers []ui.Offset
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewMouseRegion(NewColoredBox(ui.NewColor(0xFF000000), newBox(50, 50)), MouseRegionOpts{
		OnHover: func(e *gestures.PointerHoverEvent) { hovers = append(hovers, e.Position()) },
	}), ui.TextDirectionLTR)
	for _, at := range []ui.Offset{{Dx: 100, Dy: 100}, {Dx: 10, Dy: 10}, {Dx: 20, Dy: 20}, {Dx: 200, Dy: 200}} {
		ts.Hover(at)
	}
	want := []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 20, Dy: 20}}
	if len(hovers) != len(want) || hovers[0] != want[0] || hovers[1] != want[1] {
		t.Errorf("hovers = %v, want %v", hovers, want)
	}
}

func TestMouseRegionConfiguresItsRenderObject(t *testing.T) {
	m := NewMouseRegion(nil)
	if m.Cursor() == nil {
		t.Error("the default cursor is nil")
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewMouseRegion(newBox(50, 50), MouseRegionOpts{Opaque: foundation.Some(false)}), ui.TextDirectionLTR)
	r := ts.RenderObject(gluttertest.ByType[*MouseRegion]()).(*rendering.RenderMouseRegion)
	if r.Opaque() {
		t.Error("the render object is opaque")
	}
	pumpLoose(ts, NewMouseRegion(newBox(50, 50)), ui.TextDirectionLTR)
	if !r.Opaque() {
		t.Error("the render object is not opaque after the update")
	}
}
