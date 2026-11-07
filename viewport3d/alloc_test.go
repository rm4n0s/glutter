package viewport3d_test

import (
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// steadyFrame returns a function that runs one whole frame of the rig, the way gpu.View does: an encoder, the renderer's Draw, the
// finish, the submit and the completion of what the GPU did. The device is quiet, so that what the fake makes of the calls (its log
// of them) is not counted: what is left is package viewport3d and package gpu.
func steadyFrame(t testing.TB, rig *drawRig) func() {
	t.Helper()
	rig.rec.Device().Quiet = true
	f := rig.frame
	frame := func() {
		f.Encoder = f.Device.NewCommandEncoder()
		rig.r.Draw(&f)
		if err := f.Device.Queue().Submit(f.Encoder.Finish()); err != nil {
			t.Fatal(err)
		}
		rig.rec.CompleteReads()
	}
	// The first frames make the buffers, the pipelines and the scratch.
	for i := 0; i < 4; i++ {
		frame()
	}
	return frame
}

// TestASteadyFrameAllocatesNothing is the allocation test of SPEC §27.10: a frame in which nothing is added, removed or changed does not
// allocate, in every display mode and with each kind of draw: package gpu records its commands in lists that it keeps, and the renderer
// keeps its scratch.
func TestASteadyFrameAllocatesNothing(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes what a call allocates")
	}
	if runtime.GOARCH == "wasm" && foundation.DebugMode {
		// A debug build checks that each call is on the UI thread, and on wasm there is no assembly to read the goroutine's number
		// (internal/goid), so the check walks the stack, which allocates. The release build (-tags glutter_release) allocates nothing.
		t.Skip("a debug build on wasm allocates in the check of the UI thread")
	}
	grid := func(n int) []*viewport3d.Item {
		mesh, mat := scenetest.Box(0.1, 0.1, 0.1), viewport3d.NewPhongMaterial()
		items := make([]*viewport3d.Item, n)
		for i := range items {
			items[i] = viewport3d.NewItem(mesh, mat, at(float64(i%20)*0.2-2, float64(i/20)*0.2-2, 0))
		}
		return items
	}
	steel := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: ui.NewColor(0xFFB8C4D0)})
	tests := []struct {
		name  string
		build func() ([]*viewport3d.Item, viewport3d.SceneStyle)
		after func(s *viewport3d.Scene, items []*viewport3d.Item)
		depth gpu.TextureFormat
	}{
		{"20 items that share a batch", func() ([]*viewport3d.Item, viewport3d.SceneStyle) { return grid(20), viewport3d.SceneStyle{} }, nil, gpu.TextureFormatDepth32Float},
		{"1,600 items that share a batch", func() ([]*viewport3d.Item, viewport3d.SceneStyle) { return grid(1600), viewport3d.SceneStyle{} }, nil, gpu.TextureFormatDepth32Float},
		{"the part, shaded with edges", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			return scenetest.Part(), viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges}
		}, nil, gpu.TextureFormatDepth32Float},
		{"the part, wireframe", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			return scenetest.Part(), viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeWireframe}
		}, nil, gpu.TextureFormatDepth32Float},
		{"the part, hidden line", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			return scenetest.Part(), viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeHiddenLine}
		}, nil, gpu.TextureFormatDepth32Float},
		{"the part with a grid", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			return scenetest.Part(), viewport3d.SceneStyle{Grid: viewport3d.GridStyle{Show: true, Origin: vecmath.Vec3{Y: -0.5}, Spacing: 0.5, MajorEvery: 4}}
		}, nil, gpu.TextureFormatDepth32Float},
		{"the part cut by a plane, with its cap", func() ([]*viewport3d.Item, viewport3d.SceneStyle) { return scenetest.Part(), viewport3d.SceneStyle{} },
			func(s *viewport3d.Scene, _ []*viewport3d.Item) {
				s.SetSectionPlanes(vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.1}, vecmath.Vec3{X: 1, Z: 1}.Normalize()))
			}, gpu.TextureFormatDepth24PlusStencil8},
		{"two items selected", func() ([]*viewport3d.Item, viewport3d.SceneStyle) { return scenetest.Part(), viewport3d.SceneStyle{} },
			func(s *viewport3d.Scene, items []*viewport3d.Item) { s.Select(items[0], items[2]) }, gpu.TextureFormatDepth32Float},
		{"lines and points", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			return []*viewport3d.Item{
				viewport3d.NewItem(scenetest.Polyline([3]float32{-1, 0, 0}, [3]float32{0, 1, 0}, [3]float32{1, 0, 0}),
					viewport3d.NewLineMaterial(viewport3d.LineOpts{Width: 2, Stipple: []float64{6, 4}})),
				viewport3d.NewItem(scenetest.Points([3]float32{0, 0, 0}, [3]float32{0.5, 0.5, 0}), viewport3d.NewPointMaterial(viewport3d.PointOpts{Size: 6})),
			}, viewport3d.SceneStyle{}
		}, nil, gpu.TextureFormatDepth32Float},
		{"PBR items, shaded with edges", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			mesh, metal := scenetest.Sphere(0.4, 24, 12), viewport3d.NewPBRMaterial(viewport3d.PBROpts{Metallic: foundation.Some(1.0)})
			return []*viewport3d.Item{
				viewport3d.NewItem(mesh, viewport3d.NewPBRMaterial()), viewport3d.NewItem(mesh, metal, at(1, 0, 0)), viewport3d.NewItem(mesh, metal, at(-1, 0, 0)),
			}, viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges}
		}, nil, gpu.TextureFormatDepth32Float},
		{"PBR items with a shadow, and a Phong item that casts one", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			return []*viewport3d.Item{
				viewport3d.NewItem(scenetest.Box(2, 0.1, 2), viewport3d.NewPBRMaterial(), at(0, -0.6, 0)),
				viewport3d.NewItem(scenetest.Box(0.5, 0.5, 0.5), viewport3d.NewPBRMaterial()),
				viewport3d.NewItem(scenetest.Sphere(0.3, 16, 8), steel, at(0.8, 0, 0)),
			}, viewport3d.SceneStyle{Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: 1, Y: 1, Z: 0.3}}}, Shadow: viewport3d.ShadowStyle{Enabled: true}}
		}, nil, gpu.TextureFormatDepth32Float},
		{"a skinned item, one with morph targets, a transparent one, and the sections", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity(), vecmath.NewMat4Translation(vecmath.Vec3{X: -1})})
			skin.SetJointWorld(1, vecmath.NewMat4Translation(vecmath.Vec3{X: 1}).Mul(vecmath.NewMat4RotationZ(0.6)))
			bar := viewport3d.NewItem(scenetest.SkinnedBar(), viewport3d.NewPBRMaterial())
			bar.SetSkin(skin)
			ball := viewport3d.NewItem(scenetest.MorphSphere(0.4, 16, 8), viewport3d.NewPBRMaterial(), at(0, 1, 0))
			ball.SetMorphWeights([]float32{0.5})
			glass := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: ui.NewColor(0x80FF8040), AlphaMode: viewport3d.AlphaModeBlend})
			return []*viewport3d.Item{bar, ball, viewport3d.NewItem(scenetest.Box(1, 1, 1), glass, at(0.5, 0.2, 0.5))}, viewport3d.SceneStyle{}
		}, func(s *viewport3d.Scene, _ []*viewport3d.Item) {
			s.SetSectionPlanes(vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.1}, vecmath.Vec3{X: 1, Z: 1}.Normalize()))
		}, gpu.TextureFormatDepth24PlusStencil8},
		{"transparent items", func() ([]*viewport3d.Item, viewport3d.SceneStyle) {
			glass := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: ui.NewColor(0x80FF8040)})
			return []*viewport3d.Item{
				viewport3d.NewItem(scenetest.Box(1, 1, 1), glass), viewport3d.NewItem(scenetest.Box(1, 1, 1), glass, at(0.4, 0.2, 0.5)),
				viewport3d.NewItem(scenetest.Box(1, 1, 1), steel, at(-1, 0, 0)),
			}, viewport3d.SceneStyle{}
		}, nil, gpu.TextureFormatDepth32Float},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, style := tt.build()
			scene := viewport3d.NewScene()
			scene.Add(items...)
			scene.SetStyle(style)
			if tt.after != nil {
				tt.after(scene, items)
			}
			rig := newDrawRigDepth(t, scene, frontCamera(), tt.depth)
			frame := steadyFrame(t, rig)
			if got := testing.AllocsPerRun(50, frame); got != 0 {
				t.Errorf("a steady frame allocates %v times, want 0", got)
			}
			if st := rig.r.Stats(); st.Draws == 0 {
				t.Errorf("the frame drew nothing: %+v", st)
			}
		})
	}
}

// TestFramesThatDifferShareTheRecycledStorage checks what the lists that package gpu keeps change for the
// frames after a change: the encoder, its passes and the replay of the frame are used again, so a frame with another number of passes
// (a pick makes small ones) and one after a change of the scene must draw what it should.
func TestFramesThatDifferShareTheRecycledStorage(t *testing.T) {
	scene := viewport3d.NewScene()
	items := scenetest.Part()
	scene.Add(items...)
	rig := newDrawRig(t, scene, frontCamera())
	f := rig.frame
	draw := func() int {
		f.Encoder = f.Device.NewCommandEncoder()
		rig.r.Draw(&f)
		if err := f.Device.Queue().Submit(f.Encoder.Finish()); err != nil {
			t.Fatal(err)
		}
		rig.rec.CompleteReads()
		return rig.r.Stats().Draws
	}
	base := draw()
	if base == 0 {
		t.Fatal("nothing drawn")
	}
	picked := 0
	for i := 0; i < 3; i++ {
		rig.r.Pick(ui.Offset{Dx: 100, Dy: 50}, func(r viewport3d.PickResult, err error) { picked++ })
		rig.redraw()
		rig.rec.CompleteReads()
		rig.ts.Pump(frame)
	}
	if picked != 3 {
		t.Errorf("%d of 3 picks were answered", picked)
	}
	scene.Remove(items[0])
	if got := draw(); got >= base {
		t.Errorf("after an item was removed the frame drew %d, before %d", got, base)
	}
	scene.Add(items[0])
	if got := draw(); got != base {
		t.Errorf("after it was added again the frame drew %d, want %d", got, base)
	}
}
