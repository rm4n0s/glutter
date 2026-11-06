package viewport3d_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	. "github.com/rm4n0s/glutter/widgets"
)

const (
	frame                = 16 * time.Millisecond
	explicitDepthStencil = int(explicit.TextureFormatDepth24PlusStencil8)
)

// newViewTester puts a viewport3d.View of 200 x 100 pixels, with no navigation, on a tester with a recording GPU device.
func newViewTester(t *testing.T, scene *viewport3d.Scene, opts ...viewport3d.ViewOpts) (*gluttertest.Tester, *gluttertest.GPURecorder) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	rec := ts.GPUDevice()
	o := viewport3d.ViewOpts{NoNavigation: true}
	if len(opts) > 0 {
		o = opts[0]
	}
	ts.PumpWidget(NewCenter(NewSizedBox(viewport3d.NewView(scene, viewport3d.NewCamera(), o), SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})))
	return ts, rec
}

// textureFormatOf is the format of a NewTexture call of the recording device.
func textureFormatOf(call string) int {
	i := strings.Index(call, "format=")
	if i < 0 {
		return -1
	}
	s := call[i+len("format="):]
	j := 0
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	n, err := strconv.Atoi(s[:j])
	if err != nil {
		return -1
	}
	return n
}

func TestAPickIsMadeInTheNextFrameAndAnsweredWhenTheGPUHasFinished(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	rig := newDrawRig(t, scene, frontCamera())
	if n := countCalls(rig.redraw(), "BeginRenderPass"); n != 1 {
		t.Fatalf("a frame with no pick made %d passes", n)
	}
	var results []viewport3d.PickResult
	var errs []error
	rig.r.Pick(ui.Offset{Dx: 100, Dy: 50}, func(r viewport3d.PickResult, err error) { results, errs = append(results, r), append(errs, err) })
	if len(results) != 0 {
		t.Fatalf("the pick was answered before a frame")
	}
	calls := rig.redraw()
	if n := countCalls(calls, "BeginRenderPass"); n != 2 {
		t.Errorf("the frame of the pick made %d passes, want its own and the pick's", n)
	}
	if n := countCalls(calls, "CopyTextureToBuffer"); n != 2 {
		t.Errorf("the pick copied %d textures, want the ids and the depth", n)
	}
	if len(results) != 0 {
		t.Errorf("the pick was answered before the GPU finished")
	}
	rig.rec.CompleteReads()
	rig.ts.Pump(frame)
	if len(results) != 1 || errs[0] != nil {
		t.Fatalf("%d answers, errors %v", len(results), errs)
	}
	if results[0].Hit {
		t.Errorf("the recording device draws nothing, but the pick hit %v", results[0])
	}
	// Never every frame: the next frame has no pick.
	if n := countCalls(rig.redraw(), "BeginRenderPass"); n != 1 {
		t.Errorf("the frame after the pick made %d passes", n)
	}
	rig.rec.CompleteReads()
	rig.ts.Pump(frame)
	if len(results) != 1 {
		t.Errorf("the pick was answered %d times", len(results))
	}
}

func TestAPickOutsideTheViewIsAnsweredWithNoHit(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	rig := newDrawRig(t, scene, frontCamera())
	answers := 0
	rig.r.Pick(ui.Offset{Dx: -50, Dy: -50}, func(r viewport3d.PickResult, err error) {
		answers++
		if err != nil || r.Hit {
			t.Errorf("result %v, error %v", r, err)
		}
	})
	rig.r.PickRect(ui.NewRectFromLTRB(300, 300, 400, 400), func(r viewport3d.PickRectResult, err error) {
		answers++
		if err != nil || len(r.Items) != 0 {
			t.Errorf("result %v, error %v", r, err)
		}
	})
	calls := rig.redraw()
	if n := countCalls(calls, "BeginRenderPass"); n != 1 {
		t.Errorf("picks outside the view made %d passes", n)
	}
	rig.ts.Pump(frame)
	if answers != 2 {
		t.Errorf("%d answers", answers)
	}
}

func TestSeveralPicksOfAFrameHaveAPassEach(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()), viewport3d.NewItem(scenetest.Polyline([3]float32{-1, 0, 0}, [3]float32{1, 0, 0}), viewport3d.NewLineMaterial()),
		viewport3d.NewItem(scenetest.Points([3]float32{0, 1, 0}), viewport3d.NewPointMaterial()))
	rig := newDrawRig(t, scene, frontCamera())
	answers := 0
	for i := 0; i < 3; i++ {
		rig.r.Pick(ui.Offset{Dx: 60 + 20*float64(i), Dy: 50}, func(r viewport3d.PickResult, err error) { answers++ })
	}
	rig.r.PickRect(ui.NewRectFromLTRB(10, 10, 60, 60), func(r viewport3d.PickRectResult, err error) { answers++ })
	if n := countCalls(rig.redraw(), "BeginRenderPass"); n != 5 {
		t.Errorf("a frame with four picks made %d passes, want 5", n)
	}
	rig.rec.CompleteReads()
	rig.ts.Pump(frame)
	if answers != 4 {
		t.Errorf("%d answers, want 4", answers)
	}
}

func TestAPickThatCannotBeMadeIsAnsweredWithPickUnavailable(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	t.Run("a renderer with no device", func(t *testing.T) {
		r := viewport3d.NewRenderer(scene, frontCamera())
		var err error
		r.Pick(ui.Offset{}, func(_ viewport3d.PickResult, e error) { err = e })
		wantTag(t, err, viewport3d.TagPickUnavailable, "Renderer.Pick.PickUnavailable")
		err = nil
		r.PickRect(ui.NewRectFromLTRB(0, 0, 1, 1), func(_ viewport3d.PickRectResult, e error) { err = e })
		wantTag(t, err, viewport3d.TagPickUnavailable, "Renderer.PickRect.PickUnavailable")
	})
	t.Run("a renderer that is disposed while the pick waits", func(t *testing.T) {
		rig := newDrawRig(t, scene, frontCamera())
		var err1, err2 error
		rig.r.Pick(ui.Offset{Dx: 10, Dy: 10}, func(_ viewport3d.PickResult, e error) { err1 = e })
		rig.r.PickRect(ui.NewRectFromLTRB(0, 0, 10, 10), func(_ viewport3d.PickRectResult, e error) { err2 = e })
		rig.r.Dispose()
		wantTag(t, err1, viewport3d.TagPickUnavailable, "Renderer.Dispose->Renderer.failPicks.PickUnavailable")
		wantTag(t, err2, viewport3d.TagPickUnavailable, "Renderer.Dispose->Renderer.failPicks.PickUnavailable")
	})
	t.Run("a controller with no view", func(t *testing.T) {
		ctl := viewport3d.NewViewController()
		var err error
		ctl.Pick(ui.Offset{}, func(_ viewport3d.PickResult, e error) { err = e })
		wantTag(t, err, viewport3d.TagPickUnavailable, "ViewController.Pick.PickUnavailable")
		err = nil
		ctl.PickRect(ui.NewRectFromLTRB(0, 0, 1, 1), func(_ viewport3d.PickRectResult, e error) { err = e })
		wantTag(t, err, viewport3d.TagPickUnavailable, "ViewController.PickRect.PickUnavailable")
	})
}

func TestAFailureOfTheDeviceIsTheAnswerOfTheWaitingPicks(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	tests := []struct {
		name   string
		warmUp bool
		picks  int
		// route is the route of the answer, empty when it is the device's own error.
		route string
	}{
		// The pipelines of the pick are the first thing the frame makes.
		{"the pipeline of the pick", false, 1, "Renderer.Draw.PickUnavailable"},
		// Twelve picks need more room in the uniform buffer than the one of the warm-up: the buffer is what fails.
		{"the buffers of the frame", true, 12, ""},
		// One pick has the room, and the pass of its own makes its textures: the first of those fails.
		{"the targets of the pick", true, 1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rig := newDrawRig(t, scene, frontCamera())
			rig.redraw()
			if tt.warmUp {
				rig.r.Pick(ui.Offset{Dx: 100, Dy: 50}, func(viewport3d.PickResult, error) {})
				rig.redraw()
				rig.rec.CompleteReads()
				rig.ts.Pump(frame)
			}
			var errs []error
			for i := 0; i < tt.picks; i++ {
				rig.r.Pick(ui.Offset{Dx: 100, Dy: 50}, func(_ viewport3d.PickResult, err error) { errs = append(errs, err) })
			}
			rig.rec.Device().Err = errFake
			rig.redraw()
			rig.rec.CompleteReads()
			rig.ts.Pump(frame)
			if len(errs) != tt.picks {
				t.Fatalf("%d answers, want %d", len(errs), tt.picks)
			}
			for _, err := range errs {
				if err == nil {
					t.Fatalf("a pick was answered with no error")
				}
				if tt.route != "" {
					wantTag(t, err, viewport3d.TagPickUnavailable, tt.route)
				}
			}
		})
	}
}

var errFake = fakeError("the device failed")

type fakeError string

func (e fakeError) Error() string { return string(e) }

func TestTheViewControllerPicksInTheViewsNextFrame(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	ctl := viewport3d.NewViewController()
	ts, rec := newViewTester(t, scene, viewport3d.ViewOpts{NoNavigation: true, Controller: ctl})
	rec.Reset()
	var got []viewport3d.PickResult
	var rects []viewport3d.PickRectResult
	ctl.Pick(ui.Offset{Dx: 100, Dy: 50}, func(r viewport3d.PickResult, err error) {
		if err != nil {
			t.Errorf("pick: %v", err)
		}
		got = append(got, r)
	})
	ctl.PickRect(ui.NewRectFromLTRB(50, 25, 150, 75), func(r viewport3d.PickRectResult, err error) {
		if err != nil {
			t.Errorf("pick rect: %v", err)
		}
		rects = append(rects, r)
	})
	if len(got)+len(rects) != 0 {
		t.Fatalf("answered before the frame")
	}
	ts.Pump(frame)
	if n := countCalls(rec.Calls(), "BeginRenderPass"); n != 3 {
		t.Errorf("the frame made %d passes, want its own and two picks", n)
	}
	rec.CompleteReads()
	ts.Pump(frame)
	if len(got) != 1 || len(rects) != 1 {
		t.Errorf("%d and %d answers, want 1 and 1", len(got), len(rects))
	}
}

func TestTheRayCastFindsTheFirstTriangleAndWhatIsCutAwayIsNotMet(t *testing.T) {
	far := scenetest.FarOrigin
	box := scenetest.Box(2, 2, 2)
	tests := []struct {
		name       string
		origin     vecmath.Vec3
		ray        func(o vecmath.Vec3) vecmath.Ray
		planes     func(o vecmath.Vec3) []vecmath.Plane
		wantHit    bool
		wantPoint  vecmath.Vec3
		wantNormal vecmath.Vec3
	}{
		{"the front face", vecmath.Vec3{}, func(o vecmath.Vec3) vecmath.Ray {
			return vecmath.Ray{Origin: o.Add(vecmath.Vec3{X: 0.2, Y: 0.3, Z: 9}), Dir: vecmath.Vec3{Z: -1}}
		}, nil, true, vecmath.Vec3{X: 0.2, Y: 0.3, Z: 1}, vecmath.Vec3Z},
		{"the top face from above", vecmath.Vec3{}, func(o vecmath.Vec3) vecmath.Ray {
			return vecmath.Ray{Origin: o.Add(vecmath.Vec3{X: -0.4, Y: 7, Z: 0.1}), Dir: vecmath.Vec3{Y: -1}}
		}, nil, true, vecmath.Vec3{X: -0.4, Y: 1, Z: 0.1}, vecmath.Vec3Y},
		{"a miss", vecmath.Vec3{}, func(o vecmath.Vec3) vecmath.Ray {
			return vecmath.Ray{Origin: o.Add(vecmath.Vec3{X: 3, Z: 9}), Dir: vecmath.Vec3{Z: -1}}
		}, nil, false, vecmath.Vec3{}, vecmath.Vec3{}},
		{"the front face is cut away: the back one is met, and faces the ray", vecmath.Vec3{}, func(o vecmath.Vec3) vecmath.Ray {
			return vecmath.Ray{Origin: o.Add(vecmath.Vec3{X: 0.2, Y: 0.3, Z: 9}), Dir: vecmath.Vec3{Z: -1}}
		}, func(o vecmath.Vec3) []vecmath.Plane {
			return []vecmath.Plane{vecmath.NewPlaneFromPointNormal(o.Add(vecmath.Vec3{Z: 0.5}), vecmath.Vec3Z)}
		}, true, vecmath.Vec3{X: 0.2, Y: 0.3, Z: -1}, vecmath.Vec3Z},
		{"the part 50 km from the origin", far, func(o vecmath.Vec3) vecmath.Ray {
			return vecmath.Ray{Origin: o.Add(vecmath.Vec3{X: 0.2, Y: 0.3, Z: 9}), Dir: vecmath.Vec3{Z: -1}}
		}, nil, true, vecmath.Vec3{X: 0.2, Y: 0.3, Z: 1}, vecmath.Vec3Z},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			it := viewport3d.NewItem(box, viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{Transform: foundation.Some(vecmath.NewMat4Translation(tt.origin))})
			scene.Add(it)
			if tt.planes != nil {
				scene.SetSectionPlanes(tt.planes(tt.origin)...)
			}
			hit, ok := scene.Raycast(tt.ray(tt.origin))
			if ok != tt.wantHit {
				t.Fatalf("hit = %v, want %v", ok, tt.wantHit)
			}
			if !ok {
				return
			}
			if hit.Item != it {
				t.Errorf("item %v", hit.Item)
			}
			if d := hit.Point.Sub(tt.origin.Add(tt.wantPoint)).Length(); d > 1e-9 {
				t.Errorf("point %v, want %v (off by %g)", hit.Point, tt.origin.Add(tt.wantPoint), d)
			}
			if d := hit.Normal.Sub(tt.wantNormal).Length(); d > 1e-9 {
				t.Errorf("normal %v, want %v", hit.Normal, tt.wantNormal)
			}
		})
	}
}

func TestTheRayCastMeetsTheNearestItemAndNotTheHiddenOnes(t *testing.T) {
	scene := viewport3d.NewScene()
	mat := viewport3d.NewPhongMaterial()
	back := viewport3d.NewItem(scenetest.Box(1, 1, 1), mat, at(0, 0, -3))
	front := viewport3d.NewItem(scenetest.Box(1, 1, 1), mat, at(0, 0, 2))
	hidden := viewport3d.NewItem(scenetest.Box(1, 1, 1), mat, viewport3d.ItemOpts{Hidden: true, Transform: foundation.Some(vecmath.NewMat4Translation(vecmath.Vec3{Z: 4}))})
	lines := viewport3d.NewItem(scenetest.Polyline([3]float32{-1, 0, 5}, [3]float32{1, 0, 5}), viewport3d.NewLineMaterial())
	scene.Add(back, front, hidden, lines)
	ray := vecmath.Ray{Origin: vecmath.Vec3{Z: 20}, Dir: vecmath.Vec3{Z: -1}}
	hit, ok := scene.Raycast(ray)
	if !ok || hit.Item != front || hit.T < 17.4 || hit.T > 17.6 {
		t.Fatalf("hit %+v, %v: want the front item at 17.5", hit, ok)
	}
	front.SetVisible(false)
	if hit, ok = scene.Raycast(ray); !ok || hit.Item != back {
		t.Errorf("with the front item hidden the ray met %v, %v", hit.Item, ok)
	}
}

func TestSelectingItemsTellsTheSceneOnceAndKeepsTheOrderOfTheScene(t *testing.T) {
	scene := viewport3d.NewScene()
	mat := viewport3d.NewPhongMaterial()
	a, b, c := viewport3d.NewItem(scenetest.Box(1, 1, 1), mat), viewport3d.NewItem(scenetest.Box(1, 1, 1), mat), viewport3d.NewItem(scenetest.Box(1, 1, 1), mat)
	stranger := viewport3d.NewItem(scenetest.Box(1, 1, 1), mat)
	scene.Add(a, b, c)
	n := listen(scene)
	tests := []struct {
		name    string
		do      func()
		want    []*viewport3d.Item
		wantNtf int
	}{
		{"select two, given out of order", func() { scene.Select(c, a) }, []*viewport3d.Item{a, c}, 1},
		{"the same selection again", func() { scene.Select(a, c) }, []*viewport3d.Item{a, c}, 0},
		{"another item replaces them", func() { scene.Select(b) }, []*viewport3d.Item{b}, 1},
		{"an item of another scene is left out", func() { scene.Select(stranger, nil) }, nil, 1},
		{"one item selected by itself", func() { a.SetSelected(true) }, []*viewport3d.Item{a}, 1},
		{"the same again", func() { a.SetSelected(true) }, []*viewport3d.Item{a}, 0},
		{"cleared", func() { scene.ClearSelection() }, nil, 1},
		{"cleared again", func() { scene.ClearSelection() }, nil, 0},
	}
	for _, tt := range tests {
		before := n.n
		tt.do()
		got := scene.Selection()
		if len(got) != len(tt.want) {
			t.Errorf("%s: selection %v, want %v", tt.name, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] || !got[i].Selected() {
				t.Errorf("%s: selection %v, want %v", tt.name, got, tt.want)
			}
		}
		if n.n-before != tt.wantNtf {
			t.Errorf("%s: %d notifications, want %d", tt.name, n.n-before, tt.wantNtf)
		}
	}
}

func TestSelectedItemsAreOtherDrawsAndHaveAnOutline(t *testing.T) {
	tests := []struct {
		name                    string
		style                   viewport3d.SceneStyle
		selected                int
		wantBatches, wantPasses int
	}{
		{"nothing selected", viewport3d.SceneStyle{}, 0, 1, 1},
		{"one of three selected", viewport3d.SceneStyle{}, 1, 2, 2},
		{"all selected", viewport3d.SceneStyle{}, 3, 1, 2},
		{"one selected, no outline", viewport3d.SceneStyle{Selection: viewport3d.SelectionStyle{NoOutline: true}}, 1, 2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.SetStyle(tt.style)
			box, mat := scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()
			var items []*viewport3d.Item
			for i := 0; i < 3; i++ {
				items = append(items, viewport3d.NewItem(box, mat, at(float64(i)*1.5-1.5, 0, 0)))
			}
			scene.Add(items...)
			for _, it := range items[:tt.selected] {
				it.SetSelected(true)
			}
			rig := newDrawRig(t, scene, frontCamera())
			st := rig.r.Stats()
			if st.Batches != tt.wantBatches {
				t.Errorf("%d batches, want %d", st.Batches, tt.wantBatches)
			}
			if n := countCalls(rig.redraw(), "BeginRenderPass"); n != tt.wantPasses {
				t.Errorf("%d passes, want %d", n, tt.wantPasses)
			}
			// The mask is made once for the size of the frame and let go when nothing is selected.
			live := rig.rec.Device().Live()
			scene.ClearSelection()
			rig.redraw()
			if after := rig.rec.Device().Live(); tt.selected > 0 && !tt.style.Selection.NoOutline && after.Textures >= live.Textures {
				t.Errorf("the mask was not let go: %d textures before, %d after", live.Textures, after.Textures)
			}
		})
	}
}

func TestTheWheelOfAViewZoomsTowardThePointOfTheSceneUnderThePointer(t *testing.T) {
	scene := viewport3d.NewScene()
	// A box whose front face is 4 units from the target: not the plane through the target, which is where a view with no scene to
	// ask would zoom.
	scene.Add(viewport3d.NewItem(scenetest.Box(4, 4, 4), viewport3d.NewPhongMaterial(), at(0, 0, -2)))
	cam := viewport3d.NewCamera()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	ts.PumpWidget(NewCenter(NewSizedBox(viewport3d.NewView(scene, cam), SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})))
	// The view is in the middle of an 800 x 600 window.
	origin := ui.Offset{Dx: 300, Dy: 250}
	local := ui.Offset{Dx: 112, Dy: 44}
	hit, ok := scene.Raycast(cam.Ray(local))
	if !ok {
		t.Fatal("the point is not over the box")
	}
	if hit.Point.Z < -0.01 || hit.Point.Z > 0.01 {
		t.Fatalf("the ray met the box at %v, want its front face at z = 0", hit.Point)
	}
	focal := cam.FocalPoint(local)
	before := cam.Distance()
	ts.ScrollAt(ui.Offset{Dx: origin.Dx + local.Dx, Dy: origin.Dy + local.Dy}, ui.Offset{Dy: -100})
	if cam.Distance() >= before {
		t.Fatalf("the distance is %v, want less than %v", cam.Distance(), before)
	}
	got, ok := cam.Project(hit.Point)
	if !ok || got.Sub(local).Distance() > 1e-4 {
		t.Errorf("the point of the box under the pointer is at %v after the zoom, want %v (the plane of the target's point was %v)", got, local, focal)
	}
}
