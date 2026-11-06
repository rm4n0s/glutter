package viewport3d_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func TestTheGridIsOneDrawInTheFramesOwnPass(t *testing.T) {
	tests := []struct {
		name  string
		style viewport3d.SceneStyle
		want  int
	}{
		{"none by default", viewport3d.SceneStyle{}, 0},
		{"a grid", viewport3d.SceneStyle{Grid: viewport3d.GridStyle{Show: true}}, 1},
		{"a grid with its options but not shown", viewport3d.SceneStyle{Grid: viewport3d.GridStyle{Spacing: 3}}, 0},
		{"a grid in a wireframe", viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeWireframe, Grid: viewport3d.GridStyle{Show: true}}, 1},
		{"a grid in a hidden line view", viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeHiddenLine, Grid: viewport3d.GridStyle{Show: true}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.SetStyle(tt.style)
			scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
			rig := newDrawRig(t, scene, frontCamera())
			if got := rig.r.Stats().GridDraws; got != tt.want {
				t.Errorf("grid draws = %d, want %d", got, tt.want)
			}
			if n := countCalls(rig.redraw(), "BeginRenderPass"); n != 1 {
				t.Errorf("the frame made %d passes: the grid is in the frame's own", n)
			}
		})
	}
}

func TestTheGridIsDrawnAfterTheOpaqueFacesAndBeforeTheTransparentOnes(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Grid: viewport3d.GridStyle{Show: true}})
	scene.Add(
		viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()),
		viewport3d.NewItem(scenetest.Sphere(0.5, 8, 6), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: gray(0x80)}), at(2, 0, 0)),
	)
	rig := newDrawRig(t, scene, frontCamera())
	var kinds []string
	for _, c := range rig.redraw() {
		switch {
		case strings.HasPrefix(c, "DrawIndexed("):
			kinds = append(kinds, "faces")
		case strings.HasPrefix(c, "Draw(") && strings.Contains(c, "vertices=3 "):
			kinds = append(kinds, "grid")
		}
	}
	if len(kinds) != 3 || kinds[0] != "faces" || kinds[1] != "grid" || kinds[2] != "faces" {
		t.Errorf("the draws are %v, want the opaque faces, the grid and the transparent faces", kinds)
	}
}

func TestTheGridGivesTheShadersItsPlane(t *testing.T) {
	far := scenetest.FarOrigin
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Grid: viewport3d.GridStyle{Show: true, Origin: far.Add(vecmath.Vec3{Y: -0.5}), Spacing: 0.25, MajorEvery: 4, LineWidth: 2}})
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{Transform: foundation.Some(vecmath.NewMat4Translation(far))}))
	cam := viewport3d.NewCamera(viewport3d.CameraOpts{Target: far, Distance: 5})
	cam.SetViewportSize(frontCamera().ViewportSize())
	rig := newDrawRig(t, scene, cam)
	rig.redraw()
	u := rig.uniformsOf()
	// The first record is the grid's, after the frame's own data (512 bytes): the plane y = -0.5 below a camera that looks at the
	// part from 5 units: its normal is up in the view and it is 5.5 - 5 = 0.5 units below the eye's height.
	const rec = 512
	n := [3]float64{floatAt(u, rec), floatAt(u, rec+4), floatAt(u, rec+8)}
	d := floatAt(u, rec+12)
	if n != [3]float64{0, 1, 0} || d < 0.499 || d > 0.501 {
		t.Errorf("the plane is %v, %v; want the normal (0, 1, 0) and the distance 0.5 (relative to the eye, 50 km from the origin)", n, d)
	}
	if spacing, major, width := floatAt(u, rec+48), floatAt(u, rec+52), floatAt(u, rec+96); spacing != 0.25 || major != 4 || width != 2 {
		t.Errorf("spacing %v, major %v, width %v", spacing, major, width)
	}
}

func TestADisposedRendererHasReleasedWhatEveryKindOfDrawMade(t *testing.T) {
	// A grid, a section with its cap and outline, a selection with its mask and a pick: what the renderer made on the device for them
	// is let go when it is disposed (a leak is a warning in the log, and a count here).
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Grid: viewport3d.GridStyle{Show: true}})
	box := viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial())
	scene.Add(box, viewport3d.NewItem(scenetest.Polyline([3]float32{-1, 0, 0}, [3]float32{1, 0, 0}), viewport3d.NewLineMaterial()),
		viewport3d.NewItem(scenetest.Points([3]float32{0, 1, 0}), viewport3d.NewPointMaterial()))
	scene.SetSectionPlanes(vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.1}, vecmath.Vec3X))
	scene.Select(box)
	rig := newDrawRigDepth(t, scene, frontCamera(), gpu.TextureFormatDepth24PlusStencil8)
	rig.r.Pick(ui.Offset{Dx: 100, Dy: 50}, func(viewport3d.PickResult, error) {})
	rig.redraw()
	rig.rec.CompleteReads()
	rig.ts.Pump(frame)
	dev := rig.frame.Device
	if made := dev.Resources(); made.ShaderModules < 6 || made.RenderPipelines < 8 || made.Buffers == 0 || made.Samplers == 0 {
		t.Fatalf("the renderer made %+v: not every kind of draw ran", made)
	}
	rig.r.Dispose()
	left := dev.Resources()
	left.Textures = 0 // the view's own targets
	if left != (gpu.ResourceCounts{}) {
		t.Errorf("after Dispose the device still has %+v", left)
	}
}
