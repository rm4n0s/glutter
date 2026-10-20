package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TestSemanticsOfAWidgetInsertedUnderTheApp checks that a widget that appears
// while the semantics are on, under the routes of a MaterialApp, gets a node: the
// geometry of a new child that the flush did not reach is computed when the node
// of its parent is built (SPEC §12.10, "Semantics of new children").
func TestSemanticsOfAWidgetInsertedUnderTheApp(t *testing.T) {
	skipUnlessDebug(t) // the semantics finders need a debug build
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	show := false
	var rebuild func()
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewStatefulBuilder(
		func(bc widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
			rebuild = func() { set(func() {}) }
			children := []widgets.Widget{widgets.NewText("hello")}
			if show {
				children = append(children, widgets.NewText("inserted"))
			}
			return widgets.NewStack(children)
		})})}))
	handle := ts.EnsureSemantics()
	defer handle.Dispose()
	ts.Pump()
	show = true
	rebuild()
	ts.Pump()
	takeNone(t, ts)
	if !ts.Exists(gluttertest.BySemanticsLabel("inserted")) {
		t.Error("the inserted text has no semantics node")
	}
}

func takeNone(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	for _, e := range ts.TakeErrors() {
		t.Errorf("the framework reported %v", e)
	}
}
