package viewport3d_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// pipelinesOf lists the shaders that the frame's SetPipeline calls name, in order, with the repeats of one removed.
func pipelinesOf(calls []string) []string {
	var out []string
	for _, c := range calls {
		if rest, ok := strings.CutPrefix(c, "SetPipeline(shader="); ok {
			name := strings.TrimSuffix(rest, ")")
			if len(out) == 0 || out[len(out)-1] != name {
				out = append(out, name)
			}
		}
	}
	return out
}

func countPrefix(calls []string, prefix string) int {
	n := 0
	for _, c := range calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func sun() []viewport3d.DirectionalLight {
	return []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: 1, Y: 1, Z: 1}}}
}

// TestPBRItemsAreDrawnIntoTheFloatTargetAndToneMapped checks the passes of a frame: the shadow map, the PBR items, the tone mapping
// over the view's target, and the pass of the other items, which loads what the others drew.
func TestPBRItemsAreDrawnIntoTheFloatTargetAndToneMapped(t *testing.T) {
	pbr := viewport3d.NewPBRMaterial()
	phong := viewport3d.NewPhongMaterial()
	sphere, box := scenetest.Sphere(0.5, 16, 8), scenetest.Box(1, 1, 1)
	tests := []struct {
		name  string
		style viewport3d.SceneStyle
		items func() []*viewport3d.Item
		// pipelines are the shaders of the frame in order, passes the number of render passes and load whether the last pass loads.
		pipelines []string
		passes    int
		load      bool
	}{
		{"no PBR item: one pass that clears", viewport3d.SceneStyle{}, func() []*viewport3d.Item { return []*viewport3d.Item{viewport3d.NewItem(box, phong)} },
			[]string{"mesh"}, 1, false},
		{"PBR items: the float target, the tone mapping, then the others", viewport3d.SceneStyle{}, func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(sphere, pbr), viewport3d.NewItem(box, phong, at(1, 0, 0))}
		}, []string{"pbr", "tonemap", "mesh"}, 3, true},
		{"only PBR items: the last pass is empty and loads", viewport3d.SceneStyle{}, func() []*viewport3d.Item { return []*viewport3d.Item{viewport3d.NewItem(sphere, pbr)} },
			[]string{"pbr", "tonemap"}, 3, true},
		{"a shadow, cast by both kinds", viewport3d.SceneStyle{Lights: sun(), Shadow: viewport3d.ShadowStyle{Enabled: true, Size: 256}}, func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(sphere, pbr), viewport3d.NewItem(box, phong, at(1, 0, 0))}
		}, []string{"shadow", "pbr", "tonemap", "mesh"}, 4, true},
		{"a shadow with no PBR item to receive it is not drawn", viewport3d.SceneStyle{Lights: sun(), Shadow: viewport3d.ShadowStyle{Enabled: true}}, func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(box, phong)}
		}, []string{"mesh"}, 1, false},
		{"a shadow of a light that does not exist is not drawn", viewport3d.SceneStyle{Lights: sun(), Shadow: viewport3d.ShadowStyle{Enabled: true, Light: 3}}, func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(sphere, pbr)}
		}, []string{"pbr", "tonemap"}, 3, true},
		{"hidden line draws a PBR item as a flat fill", viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeHiddenLine}, func() []*viewport3d.Item {
			return []*viewport3d.Item{viewport3d.NewItem(sphere, pbr)}
		}, []string{"mesh"}, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.SetStyle(tt.style)
			scene.Add(tt.items()...)
			rig := newDrawRig(t, scene, frontCamera())
			calls := rig.redraw()
			got := pipelinesOf(calls)
			if strings.Join(got, ",") != strings.Join(tt.pipelines, ",") {
				t.Errorf("the pipelines of the frame are %v, want %v", got, tt.pipelines)
			}
			if n := countPrefix(calls, "BeginRenderPass("); n != tt.passes {
				t.Errorf("%d render passes, want %d", n, tt.passes)
			}
			last := ""
			for _, c := range calls {
				if strings.HasPrefix(c, "BeginRenderPass(") {
					last = c
				}
			}
			if loads := strings.Contains(last, "color0={1 load=1"); loads != tt.load {
				t.Errorf("the last pass loads the color: %v, want %v (%s)", loads, tt.load, last)
			}
		})
	}
}

// TestPBRItemsThatShareAMeshAndAMaterialAreOneDraw checks that the batching of the PBR items is the batching of the others: the
// opaque ones that share the mesh and the material are one draw, and a transparent one is a draw of its own, after them.
func TestPBRItemsThatShareAMeshAndAMaterialAreOneDraw(t *testing.T) {
	mesh := scenetest.Sphere(0.3, 12, 6)
	mat := viewport3d.NewPBRMaterial()
	other := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Metallic: foundation.Some(1.0)})
	glass := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: ui.NewColor(0x80FF8040), AlphaMode: viewport3d.AlphaModeBlend})
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(mesh, mat), viewport3d.NewItem(mesh, mat, at(0.5, 0, 0)), viewport3d.NewItem(mesh, mat, at(-0.5, 0, 0)),
		viewport3d.NewItem(mesh, other, at(0, 0.5, 0)), viewport3d.NewItem(mesh, glass, at(0, 0, 0.5)))
	rig := newDrawRig(t, scene, frontCamera())
	calls := rig.redraw()
	var instances []int
	for _, c := range calls {
		if strings.HasPrefix(c, "DrawIndexed(") {
			instances = append(instances, intAfter(c, "instances="))
		}
	}
	// The three that share, the other material, the glass.
	if len(instances) != 3 || instances[0] != 3 || instances[1] != 1 || instances[2] != 1 {
		t.Errorf("the draws have %v instances, want [3 1 1]", instances)
	}
	if st := rig.r.Stats(); st.Items != 5 || st.Batches != 3 || st.Instances != 5 {
		t.Errorf("stats %+v", st)
	}
}

// TestChangingAPBRMaterialDrawsAgain checks that the scene's views are drawn again when a PBR material, a skin or the weights of an
// item change, and not when the change is none.
func TestChangingAPBRMaterialDrawsAgain(t *testing.T) {
	skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity(), vecmath.Mat4Identity()})
	mat := viewport3d.NewPBRMaterial()
	bar := viewport3d.NewItem(scenetest.SkinnedBar(), mat)
	bar.SetSkin(skin)
	ball := viewport3d.NewItem(scenetest.MorphSphere(0.3, 8, 4), mat)
	scene := viewport3d.NewScene()
	scene.Add(bar, ball)
	changes := 0
	scene.AddListener(func() { changes++ })
	tests := []struct {
		name string
		do   func()
		want int
	}{
		{"a new roughness", func() { mat.SetRoughness(0.9) }, 1},
		{"the same roughness", func() { mat.SetRoughness(0.9) }, 0},
		{"a joint", func() { skin.SetJointWorld(1, vecmath.NewMat4Translation(vecmath.Vec3{Y: 1})) }, 1},
		{"the same joint", func() { skin.SetJointWorld(1, vecmath.NewMat4Translation(vecmath.Vec3{Y: 1})) }, 0},
		{"all the joints", func() { skin.SetJoints([]vecmath.Mat4{vecmath.Mat4Identity(), vecmath.Mat4Identity()}) }, 1},
		{"morph weights", func() { ball.SetMorphWeights([]float32{0.5}) }, 1},
		{"the same morph weights", func() { ball.SetMorphWeights([]float32{0.5}) }, 0},
		{"a skin taken off", func() { bar.SetSkin(nil) }, 1},
		{"a joint of a skin that no item has", func() { skin.SetJointWorld(0, vecmath.NewMat4Translation(vecmath.Vec3{X: 3})) }, 0},
	}
	for _, tt := range tests {
		changes = 0
		tt.do()
		if changes != tt.want {
			t.Errorf("%s: the scene changed %d times, want %d", tt.name, changes, tt.want)
		}
	}
}
