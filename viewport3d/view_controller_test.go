package viewport3d_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/viewport3d"
	. "github.com/rm4n0s/glutter/widgets"
)

// A view that moves in the tree (a layout that changes from a row to a column when the window gets narrow) is a new state, made before
// the old one is disposed. The controller that both were given must keep the new view: it failed with "the controller has no view with a
// GPU" for every pick after the move (found by examples/cad).
func TestAControllerKeepsItsViewWhenTheViewMovesInTheTree(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	camera := viewport3d.NewCamera()
	ctl := viewport3d.NewViewController()
	view := func() Widget {
		return NewSizedBox(viewport3d.NewView(scene, camera, viewport3d.ViewOpts{NoNavigation: true, Controller: ctl}),
			SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})
	}
	ltr := func(w Widget) Widget { return NewDirectionality(ui.TextDirectionLTR, w) }
	layouts := map[string]func() Widget{
		"a row":    func() Widget { return ltr(NewCenter(NewRow([]Widget{view()}))) },
		"a column": func() Widget { return ltr(NewCenter(NewColumn([]Widget{view()}))) },
		"a stack":  func() Widget { return ltr(NewCenter(NewStack([]Widget{view()}))) },
	}
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	for _, name := range []string{"a row", "a column", "a stack", "a row"} {
		ts.PumpWidget(layouts[name]())
		ts.Pump(frame)
		var failed error
		answered := false
		ctl.Pick(ui.Offset{Dx: 100, Dy: 50}, func(r viewport3d.PickResult, err error) {
			answered, failed = true, err
		})
		if answered {
			t.Errorf("in %s: the pick was answered before the frame, with %v", name, failed)
		}
		ts.Pump(frame)
	}
}
