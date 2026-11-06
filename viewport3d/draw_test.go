package viewport3d_test

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	. "github.com/rm4n0s/glutter/widgets"
)

// drawRig is a tester with a recording GPU device and a view of 200 x 100 pixels that draws a scene with a Renderer, the way an
// app that makes its own gpu.View does.
type drawRig struct {
	t     testing.TB
	ts    *gluttertest.Tester
	rec   *gluttertest.GPURecorder
	r     *viewport3d.Renderer
	scene *viewport3d.Scene
	cam   *viewport3d.Camera
	ctl   *gpu.ViewController
	frame gpu.ViewFrame
}

func newDrawRig(t testing.TB, scene *viewport3d.Scene, cam *viewport3d.Camera) *drawRig {
	t.Helper()
	return newDrawRigDepth(t, scene, cam, gpu.TextureFormatDepth32Float)
}

// newDrawRigDepth is newDrawRig with the depth format of the view: the caps of the section planes need a stencil.
func newDrawRigDepth(t testing.TB, scene *viewport3d.Scene, cam *viewport3d.Camera, depth gpu.TextureFormat) *drawRig {
	t.Helper()
	return newDrawRigOpts(t, scene, cam, depth, gluttertest.Opts{GPU: true})
}

// newDrawRigOpts is newDrawRigDepth with the options of the tester: a memory test does not keep the scenes.
func newDrawRigOpts(t testing.TB, scene *viewport3d.Scene, cam *viewport3d.Camera, depth gpu.TextureFormat, opts gluttertest.Opts) *drawRig {
	t.Helper()
	rig := &drawRig{t: t, ts: gluttertest.New(t, opts), scene: scene, cam: cam, ctl: gpu.NewViewController()}
	rig.rec = rig.ts.GPUDevice()
	rig.r = viewport3d.NewRenderer(scene, cam)
	view := gpu.NewView(func(f *gpu.ViewFrame) { rig.frame = *f; rig.r.Draw(f) }, gpu.ViewOpts{
		DepthFormat: depth, SampleCount: 4, Controller: rig.ctl,
		OnInit: rig.r.Init, OnDispose: rig.r.Dispose,
	})
	rig.ts.PumpWidget(NewCenter(NewSizedBox(view, SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})))
	return rig
}

// redraw draws the scene once more and returns the calls of that frame.
func (d *drawRig) redraw() []string {
	d.t.Helper()
	d.rec.Reset()
	d.ctl.MarkNeedsPaint()
	d.ts.Pump(16 * time.Millisecond)
	return d.rec.Calls()
}

// drawn is a draw call of the frame and the model-view matrix of its first instance.
type drawn struct {
	instances int
	mv        [16]float32
}

// draws returns the triangle draws of the frame in order (the draws that use the instance buffer: the last vertex buffer they
// set has an offset in it), with the matrix of the first instance of each, read from the last write of the frame (the
// instances).
func (d *drawRig) draws(calls []string) []drawn {
	d.t.Helper()
	writes := d.rec.Device().Writes
	if len(writes) == 0 {
		return nil
	}
	inst := writes[len(writes)-1].Data
	var out []drawn
	offset := 0
	for _, c := range calls {
		switch {
		case strings.HasPrefix(c, "SetVertexBuffer("):
			offset = intAfter(c, "offset=")
		case strings.HasPrefix(c, "DrawIndexed("), strings.HasPrefix(c, "Draw("):
			if strings.Contains(c, "vertices=6 ") {
				continue // a quad of an edge, a line or a point
			}
			var m [16]float32
			for i := range m {
				m[i] = math.Float32frombits(binary.LittleEndian.Uint32(inst[offset+4*i:]))
			}
			out = append(out, drawn{instances: intAfter(c, "instances="), mv: m})
		}
	}
	return out
}

func intAfter(s, key string) int {
	i := strings.Index(s, key)
	if i < 0 {
		return -1
	}
	s = s[i+len(key):]
	j := 0
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	n, _ := strconv.Atoi(s[:j])
	return n
}

func at(x, y, z float64) viewport3d.ItemOpts {
	return viewport3d.ItemOpts{Transform: foundation.Some(vecmath.NewMat4Translation(vecmath.Vec3{X: x, Y: y, Z: z}))}
}

func frontCamera() *viewport3d.Camera {
	cam := viewport3d.NewCamera()
	cam.SetViewportSize(ui.Size{Width: 200, Height: 100})
	return cam
}

func gray(a uint32) ui.Color { return ui.NewColor(a<<24 | 0x808080) }

func TestItemsThatShareAMeshAndAMaterialAreOneDraw(t *testing.T) {
	box, ball := scenetest.Box(1, 1, 1), scenetest.Sphere(0.5, 8, 6)
	steel, red := viewport3d.NewPhongMaterial(), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: ui.NewColor(0xFFFF0000)})
	tests := []struct {
		name                                  string
		items                                 func() []*viewport3d.Item
		wantItems, wantHidden, wantCulled     int
		wantDraws, wantInstances, wantBatches int
	}{
		{"three boxes", func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(box, steel, at(-1, 0, 0)), viewport3d.NewItem(box, steel), viewport3d.NewItem(box, steel, at(1, 0, 0))}
		}, 3, 0, 0, 1, 3, 1},
		{"added in turns", func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(box, steel, at(-1, 0, 0)), viewport3d.NewItem(ball, steel), viewport3d.NewItem(box, steel, at(1, 0, 0)), viewport3d.NewItem(ball, steel, at(0, 1, 0))}
		}, 4, 0, 0, 2, 4, 2},
		{"two materials", func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(box, steel), viewport3d.NewItem(box, red, at(1, 0, 0)), viewport3d.NewItem(box, steel, at(2, 0, 0))}
		}, 3, 0, 0, 2, 3, 2},
		{"a hidden item and one behind the camera", func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(box, steel), viewport3d.NewItem(box, steel, viewport3d.ItemOpts{Hidden: true}), viewport3d.NewItem(box, steel, at(0, 0, 50))}
		}, 1, 1, 1, 1, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.Add(tt.items()...)
			rig := newDrawRig(t, scene, frontCamera())
			st := rig.r.Stats()
			if st.Items != tt.wantItems || st.Hidden != tt.wantHidden || st.Culled != tt.wantCulled {
				t.Errorf("items %d, hidden %d, culled %d; want %d, %d, %d", st.Items, st.Hidden, st.Culled, tt.wantItems, tt.wantHidden, tt.wantCulled)
			}
			if st.Draws != tt.wantDraws || st.Batches != tt.wantBatches || st.Instances != tt.wantInstances {
				t.Errorf("draws %d, batches %d, instances %d; want %d, %d, %d", st.Draws, st.Batches, st.Instances, tt.wantDraws, tt.wantBatches, tt.wantInstances)
			}
		})
	}
}

func TestTransparentItemsAreDrawnBackToFront(t *testing.T) {
	box := scenetest.Box(1, 1, 1)
	glass := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: gray(0x80)})
	scene := viewport3d.NewScene()
	// The camera is 10 units from the origin; the items are 10, 15, 5 and 20 units away, added in that order.
	for _, z := range []float64{0, -5, 5, -10} {
		scene.Add(viewport3d.NewItem(box, glass, at(0, 0, z)))
	}
	rig := newDrawRig(t, scene, frontCamera())
	var got []float64
	for _, d := range rig.draws(rig.redraw()) {
		if d.instances != 1 {
			t.Fatalf("a transparent item was batched: %d instances", d.instances)
		}
		got = append(got, -float64(d.mv[14]))
	}
	want := []float64{20, 15, 10, 5}
	if len(got) != len(want) {
		t.Fatalf("%d draws, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-4 {
			t.Fatalf("the draws are %v away from the eye, want %v", got, want)
		}
	}
}

func TestOpaqueItemsAreDrawnBeforeTheTransparentOnes(t *testing.T) {
	box := scenetest.Box(1, 1, 1)
	glass := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: gray(0x80)})
	solid := viewport3d.NewPhongMaterial()
	scene := viewport3d.NewScene()
	// The glass is the nearer and was added first.
	scene.Add(viewport3d.NewItem(box, glass, at(0, 0, 3)), viewport3d.NewItem(box, solid, at(0, 0, -3)))
	rig := newDrawRig(t, scene, frontCamera())
	draws := rig.draws(rig.redraw())
	if len(draws) != 2 || math.Abs(float64(draws[0].mv[14])+13) > 1e-4 || math.Abs(float64(draws[1].mv[14])+7) > 1e-4 {
		t.Errorf("draws = %+v, want the solid box (13 away) and then the glass box (7 away)", draws)
	}
}

func TestTheModelViewOfAFarItemIsRelativeToTheEye(t *testing.T) {
	// The part is 50 km from the origin and the camera looks at it from 4 units: the matrix has the offset, in float32, and
	// not the position (SPEC §27.8, D30).
	box := scenetest.Box(1, 1, 1)
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(box, viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{Transform: foundation.Some(vecmath.NewMat4Translation(scenetest.FarOrigin))}))
	cam := viewport3d.NewCamera(viewport3d.CameraOpts{Target: scenetest.FarOrigin.Add(vecmath.Vec3{X: 0.001}), Distance: 4})
	cam.SetViewportSize(ui.Size{Width: 200, Height: 100})
	rig := newDrawRig(t, scene, cam)
	draws := rig.draws(rig.redraw())
	if len(draws) != 1 {
		t.Fatalf("%d draws", len(draws))
	}
	m := draws[0].mv
	// The camera looks down -z from 4 units; the item is 1 mm to the left of the target and 4 units in front of the eye.
	if math.Abs(float64(m[12])+0.001) > 1e-7 || math.Abs(float64(m[13])) > 1e-7 || math.Abs(float64(m[14])+4) > 1e-6 {
		t.Errorf("the translation is (%v, %v, %v), want (-0.001, 0, -4)", m[12], m[13], m[14])
	}
}

func TestTheDisplayModesDrawWhatTheyShould(t *testing.T) {
	tests := []struct {
		mode                       viewport3d.DisplayMode
		wantBatches, wantEdgeDraws int
	}{
		{viewport3d.DisplayModeShaded, 2, 0},
		{viewport3d.DisplayModeShadedWithEdges, 2, 1}, // the smooth sphere has no feature edges
		{viewport3d.DisplayModeWireframe, 0, 2},
		{viewport3d.DisplayModeHiddenLine, 2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.SetStyle(viewport3d.SceneStyle{DisplayMode: tt.mode})
			scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()), viewport3d.NewItem(scenetest.Sphere(0.5, 12, 8), viewport3d.NewPhongMaterial(), at(2, 0, 0)))
			rig := newDrawRig(t, scene, frontCamera())
			st := rig.r.Stats()
			if st.Batches != tt.wantBatches || st.EdgeDraws != tt.wantEdgeDraws {
				t.Errorf("batches %d, edge draws %d; want %d, %d", st.Batches, st.EdgeDraws, tt.wantBatches, tt.wantEdgeDraws)
			}
		})
	}
}

func TestAChangeOfTheSceneOrTheCameraDrawsOnce(t *testing.T) {
	scene := viewport3d.NewScene()
	item := viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial())
	scene.Add(item)
	cam := frontCamera()
	rig := newDrawRig(t, scene, cam)
	draws := func() int { return countCalls(rig.rec.Calls(), "BeginRenderPass") }
	rig.rec.Reset()
	rig.ts.Pump(16 * time.Millisecond)
	if draws() != 0 {
		t.Fatalf("an idle view drew")
	}
	// The renderer, not the view, was made here: only the controller draws. The view of package viewport3d listens for itself
	// (TestTheViewDrawsWhenTheSceneOrTheCameraChanges).
	rig.redraw()
	if draws() != 1 {
		t.Errorf("one request made %d passes", draws())
	}
}

func countCalls(calls []string, op string) int {
	n := 0
	for _, c := range calls {
		if strings.HasPrefix(c, op+"(") {
			n++
		}
	}
	return n
}

func TestTheViewDrawsWhenTheSceneOrTheCameraChanges(t *testing.T) {
	scene := viewport3d.NewScene()
	item := viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial())
	scene.Add(item)
	cam := viewport3d.NewCamera()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	rec := ts.GPUDevice()
	ts.PumpWidget(NewCenter(NewSizedBox(viewport3d.NewView(scene, cam, viewport3d.ViewOpts{NoNavigation: true}), SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})))
	passes := func() int { return countCalls(rec.Calls(), "BeginRenderPass") }
	if passes() != 1 {
		t.Fatalf("the first frame made %d passes", passes())
	}
	if cam.ViewportSize() != (ui.Size{Width: 200, Height: 100}) {
		t.Errorf("the camera's viewport is %v", cam.ViewportSize())
	}
	if b := cam.Bounds(); b.Max.X < 0.49 || b.Max.X > 0.51 {
		t.Errorf("the camera's bounds are %v, not the scene's", b)
	}
	tests := []struct {
		name   string
		change func()
	}{
		{"a camera move", func() { cam.Orbit(0.1, 0) }},
		{"a transform", func() { item.SetTransform(vecmath.NewMat4Translation(vecmath.Vec3{X: 1})) }},
		{"a material color", func() { item.Material().(*viewport3d.PhongMaterial).SetColor(ui.NewColor(0xFF00FF00)) }},
		{"a style", func() { scene.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeWireframe}) }},
		{"a new item", func() {
			scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial(), at(5, 0, 0)))
		}},
	}
	for _, tt := range tests {
		before := passes()
		tt.change()
		ts.Pump(16 * time.Millisecond)
		if passes() != before+1 {
			t.Errorf("%s: %d passes, want %d", tt.name, passes(), before+1)
		}
		ts.Pump(16 * time.Millisecond)
		if passes() != before+1 {
			t.Errorf("%s: the view drew again without a change", tt.name)
		}
	}
	// The bounds follow the scene: the item that moved and the new one.
	if b := cam.Bounds(); b.Max.X < 5 {
		t.Errorf("the bounds %v did not grow with the scene", b)
	}
}

func TestRemovingTheViewReleasesTheRenderersResources(t *testing.T) {
	scene := viewport3d.NewScene()
	mesh := scenetest.Box(1, 1, 1)
	scene.Add(viewport3d.NewItem(mesh, viewport3d.NewPhongMaterial()))
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	rec := ts.GPUDevice()
	cam := viewport3d.NewCamera()
	ts.PumpWidget(NewSizedBox(viewport3d.NewView(scene, cam), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}))
	if live := rec.Device().Live(); live.Buffers == 0 || live.Pipelines == 0 {
		t.Fatalf("the view made nothing: %+v", live)
	}
	ts.PumpWidget(NewSizedBox(nil))
	if live := rec.Device().Live(); live != (fakedevice.Live{}) {
		t.Errorf("after the view left the tree the device has %+v", live)
	}
	// The mesh and the scene are still good: a new view draws them.
	ts.PumpWidget(NewSizedBox(viewport3d.NewView(scene, cam), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}))
	if live := rec.Device().Live(); live.Buffers == 0 {
		t.Errorf("the mesh was not made again: %+v", live)
	}
}

func TestNewViewChecksItsArguments(t *testing.T) {
	tests := []struct {
		name  string
		build func()
	}{
		{"no scene", func() { viewport3d.NewView(nil, viewport3d.NewCamera()) }},
		{"no camera", func() { viewport3d.NewView(viewport3d.NewScene(), nil) }},
		{"two samples", func() {
			viewport3d.NewView(viewport3d.NewScene(), viewport3d.NewCamera(), viewport3d.ViewOpts{Samples: 2})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skipUnlessDebug(t)
			wantTag(t, panicOf(tt.build), viewport3d.TagViewInvalid, "NewView.Viewport3DViewInvalid")
		})
	}
}

// skipUnlessDebug skips a test of a check that runs in debug builds only.
func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the checks of the arguments run in debug builds only")
	}
}

// panicOf runs fn and returns the error it panicked with, or nil.
func panicOf(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	fn()
	return nil
}

// wantTag checks that err is a module error with the tag and, when route is not empty, a route that has it.
func wantTag(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error with tag %s, got %T: %v", tag, err, err)
	}
	if appErr.Tag != tag {
		t.Fatalf("tag = %s, want %s (route %s)", appErr.Tag, tag, appErr.Route())
	}
	if route != "" && !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

func TestReleasingAMeshOrAMaterialFreesItsGPUResources(t *testing.T) {
	mesh := scenetest.Box(1, 1, 1)
	mat := viewport3d.NewPhongMaterial()
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges})
	scene.Add(viewport3d.NewItem(mesh, mat))
	rig := newDrawRig(t, scene, frontCamera())
	live := func() fakedevice.Live { return rig.rec.Device().Live() }
	before := live()
	mesh.Release()
	after := live()
	// The mesh has its vertex, index and edge buffers, and the edges' segments.
	if after.Buffers >= before.Buffers {
		t.Fatalf("releasing the mesh freed nothing: %+v then %+v", before, after)
	}
	mat.Release()
	// The scene draws again with what it still has: the mesh is uploaded once more.
	rig.redraw()
	if got := live(); got.Buffers != before.Buffers {
		t.Errorf("after a new frame the device has %d buffers, want the %d of before", got.Buffers, before.Buffers)
	}
	if st := rig.r.Stats(); st.Draws == 0 {
		t.Errorf("the scene drew nothing after its mesh was released: %+v", st)
	}
}

func TestAViewThatGetsANewSceneDrawsIt(t *testing.T) {
	box := scenetest.Box(1, 1, 1)
	first, second := viewport3d.NewScene(), viewport3d.NewScene()
	mat := viewport3d.NewPhongMaterial()
	first.Add(viewport3d.NewItem(box, mat))
	second.Add(viewport3d.NewItem(box, mat), viewport3d.NewItem(box, mat, at(2, 0, 0)))
	cam := frontCamera()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	rec := ts.GPUDevice()
	shown := func(s *viewport3d.Scene) Widget {
		return NewCenter(NewSizedBox(viewport3d.NewView(s, cam, viewport3d.ViewOpts{NoNavigation: true}), SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)}))
	}
	ts.PumpWidget(shown(first))
	rec.Reset()
	ts.PumpWidget(shown(second))
	calls := rec.Calls()
	if countCalls(calls, "BeginRenderPass") != 1 {
		t.Fatalf("the new scene made %d passes: %v", countCalls(calls, "BeginRenderPass"), calls)
	}
	var instances int
	for _, c := range calls {
		if strings.HasPrefix(c, "DrawIndexed(") {
			instances = intAfter(c, "instances=")
		}
	}
	if instances != 2 {
		t.Errorf("the new scene's two boxes were drawn as %d instances", instances)
	}
	// The first scene is no longer listened to.
	rec.Reset()
	first.SetStyle(viewport3d.SceneStyle{EdgeWidth: 2})
	ts.Pump(16 * time.Millisecond)
	if n := countCalls(rec.Calls(), "BeginRenderPass"); n != 0 {
		t.Errorf("a change of the old scene made %d passes", n)
	}
}

func TestTheViewDrawsAgainOnTheNewDeviceAfterALoss(t *testing.T) {
	// The renderer is given the new device and must keep it: the disposal of the lost device's state comes before the new init
	// (gpu.View), not after it, or the view would be empty for good (step 30.13).
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	rec := ts.GPUDevice()
	ts.PumpWidget(NewCenter(NewSizedBox(viewport3d.NewView(scene, viewport3d.NewCamera(), viewport3d.ViewOpts{NoNavigation: true}),
		SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})))
	for i := 0; i < 3; i++ {
		rec.Lose()
		ts.Pump(16 * time.Millisecond)
		ts.Pump(16 * time.Millisecond)
		if n := countCalls(rec.Calls(), "BeginRenderPass"); n != 1 {
			t.Fatalf("after loss %d the view drew %d passes on the new device, want 1", i+1, n)
		}
		if live := rec.Device().Live(); live.Buffers == 0 || live.Pipelines == 0 {
			t.Errorf("after loss %d the new device holds %+v", i+1, live)
		}
	}
}
