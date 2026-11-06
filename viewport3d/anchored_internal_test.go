package viewport3d

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/widgets"
)

// layerWith puts a view with the children on a tester with the recording GPU, and returns the layer of the children.
func layerWith(t *testing.T, cam *Camera, children ...widgets.Widget) (*gluttertest.Tester, *renderAnchored) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	view := NewView(NewScene(), cam, ViewOpts{Children: children})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewCenter(widgets.NewSizedBox(view,
		widgets.SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)}))))
	layer, ok := ts.RenderObject(gluttertest.ByType[*anchoredLayer]()).(*renderAnchored)
	if !ok {
		t.Fatal("the view has no layer of anchored children")
	}
	return ts, layer
}

func anchorOf(t *testing.T, ts *gluttertest.Tester, key foundation.Key) *renderAnchor {
	t.Helper()
	a, ok := ts.RenderObject(gluttertest.ByKey(key)).(*renderAnchor)
	if !ok {
		t.Fatal("the child is not an anchor")
	}
	return a
}

func TestAnAnchorIsCoveredWhenTheNearestSurfaceIsNearerThanThePoint(t *testing.T) {
	cam := NewCamera() // the eye is at (0, 0, 10)
	key := foundation.NewUniqueKey()
	child := NewAnchored(vecmath.Vec3{}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}),
		AnchoredOpts{Key: key, HideWhenCovered: true})
	ts, layer := layerWith(t, cam, child)
	anchor := anchorOf(t, ts, key)
	tests := []struct {
		name string
		res  PickResult
		err  error
		// covered is what the anchor was before, and want what it is after.
		before, want bool
	}{
		{"nothing is there", PickResult{}, nil, false, false},
		{"a surface in front", PickResult{Hit: true, Point: vecmath.Vec3{Z: 6}}, nil, false, true},
		{"a surface at the point (the point is on it)", PickResult{Hit: true, Point: vecmath.Vec3{Z: 0.0001}}, nil, false, false},
		{"a surface behind", PickResult{Hit: true, Point: vecmath.Vec3{Z: -4}}, nil, true, false},
		{"a surface just in front, within the tolerance of the point being on it", PickResult{Hit: true, Point: vecmath.Vec3{Z: 10 - 10*(1-coverTolerance/2)}}, nil, false, false},
		{"a failed pick leaves it as it is", PickResult{}, errors.New(tagTestPickFailed, "the GPU failed"), true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anchor.setCovered(tt.before)
			var asked []ui.Offset
			layer.askDepth(func(at ui.Offset, done func(PickResult, error), opts ...PickOpts) {
				asked = append(asked, at)
				if len(opts) != 1 || opts[0].Radius != 1 {
					t.Errorf("the pick has the options %v, want a radius of one pixel", opts)
				}
				done(tt.res, tt.err)
			})
			if len(asked) != 1 || asked[0] != (ui.Offset{Dx: 100, Dy: 50}) {
				t.Fatalf("asked at %v, want once at the center of the view", asked)
			}
			if anchor.covered != tt.want {
				t.Errorf("covered = %v, want %v", anchor.covered, tt.want)
			}
		})
	}
}

func TestOnlyTheAnchorsThatHideWhenCoveredAndAreInViewAreAskedAbout(t *testing.T) {
	cam := NewCamera()
	k := func(s string) foundation.Key { return foundation.NewKey(s) }
	box := func() widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)})
	}
	ts, layer := layerWith(t, cam,
		NewAnchored(vecmath.Vec3{X: 1}, box(), AnchoredOpts{Key: k("hides"), HideWhenCovered: true}),
		NewAnchored(vecmath.Vec3{X: -1}, box(), AnchoredOpts{Key: k("shows")}),
		NewAnchored(vecmath.Vec3{Z: 30}, box(), AnchoredOpts{Key: k("behind the camera"), HideWhenCovered: true}),
		NewAnchored(vecmath.Vec3{X: 400}, box(), AnchoredOpts{Key: k("out of the view"), HideWhenCovered: true}),
	)
	_ = ts
	var asked []ui.Offset
	layer.askDepth(func(at ui.Offset, _ func(PickResult, error), _ ...PickOpts) { asked = append(asked, at) })
	want, _ := cam.Project(vecmath.Vec3{X: 1})
	if len(asked) != 1 || asked[0] != want {
		t.Errorf("asked at %v, want once at %v (the one that hides, in view)", asked, want)
	}
}

func TestACoveredAnchorIsHiddenAndAnswersAgainWhenTheViewIsNot(t *testing.T) {
	cam := NewCamera()
	key := foundation.NewUniqueKey()
	child := NewAnchored(vecmath.Vec3{}, widgets.NewSizedBox(widgets.NewColoredBox(ui.NewColor(0xFFFF00FF), nil), widgets.SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}),
		AnchoredOpts{Key: key, HideWhenCovered: true})
	ts, layer := layerWith(t, cam, child)
	anchor := anchorOf(t, ts, key)
	if !anchor.shown() {
		t.Fatalf("the anchor is not shown at first")
	}
	front := func(at ui.Offset, done func(PickResult, error), _ ...PickOpts) {
		done(PickResult{Hit: true, Point: vecmath.Vec3{Z: 5}}, nil)
	}
	layer.askDepth(front)
	ts.Pump(16 * time.Millisecond)
	if anchor.shown() {
		t.Errorf("the anchor is shown while it is covered")
	}
	// A covered anchor is still asked about, so that it comes back when the cover goes away.
	asked := 0
	layer.askDepth(func(at ui.Offset, done func(PickResult, error), _ ...PickOpts) { asked++; done(PickResult{}, nil) })
	if asked != 1 || !anchor.shown() {
		t.Errorf("asked %d times and shown = %v: the anchor should be asked about again and come back", asked, anchor.shown())
	}
	// A widget that stops hiding when covered is shown at once.
	anchor.setCovered(true)
	anchor.configure(NewAnchored(vecmath.Vec3{}, nil))
	if !anchor.shown() {
		t.Errorf("the anchor that no longer hides when covered is hidden")
	}
}

// tagTestPickFailed is the tag of the error that the fake pick gives.
const tagTestPickFailed = "TestPickFailed"
