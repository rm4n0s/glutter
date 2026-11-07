package viewport3d_test

import (
	"context"
	"image/color"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	. "github.com/rm4n0s/glutter/widgets"
)

// The memory tests of the PBR material (step 31.9): what it makes on the device (the environment, the shadow map, the float target,
// the buffers of the skins and the morph targets) is let go of when the model closes, the size changes, the environment changes or the
// device is lost.

// leakPBRModel is a model whose one mesh is skinned and has a morph target, with a PBR material that has every texture.
func leakPBRModel(t *testing.T) *model.Model {
	t.Helper()
	img := &model.Image{Name: "px.png", Type: "image/png", Data: pngBytes(t, 8, 8, color.NRGBA{R: 200, G: 100, B: 50, A: 255})}
	ref := func() *model.TextureRef { return &model.TextureRef{Texture: &model.Texture{Image: img}} }
	mat := &model.Material{PBR: true, BaseColor: [4]float32{1, 1, 1, 1}, Metallic: 1, Roughness: 0.5, NormalScale: 1, OcclusionStrength: 1, AlphaCutoff: 0.5,
		BaseColorTexture: ref(), MetallicRoughnessTexture: ref(), NormalTexture: ref(), OcclusionTexture: ref(), EmissiveTexture: ref()}
	mesh := triangleMesh("skinned")
	mesh.Material = mat
	mesh.Joints = []uint16{0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0}
	mesh.Weights = []float32{0.5, 0.5, 0, 0, 0.5, 0.5, 0, 0, 0.5, 0.5, 0, 0}
	mesh.MorphTargets = []model.MorphTarget{{Positions: []float32{0, 0, 1, 0, 0, 1, 0, 0, 1}}}
	root, j1 := model.NewNode("root"), model.NewNode("joint")
	root.Children = []*model.Node{j1}
	root.Meshes = []*model.Mesh{mesh}
	root.Skin = &model.Skin{Joints: []*model.Node{root, j1}, InverseBindMatrices: []vecmath.Mat4{vecmath.Mat4Identity(), vecmath.Mat4Identity()}}
	return &model.Model{Nodes: []*model.Node{root}, Textures: nil}
}

// TestLeaksPBRModel opens a skinned PBR model with morph targets and textures, draws it with a shadow and closes it, 100 times.
func TestLeaksPBRModel(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: 1, Y: 1, Z: 1}}}, Shadow: viewport3d.ShadowStyle{Enabled: true, Size: 256}})
	rig := newDrawRigOpts(t, scene, frontCamera(), gpu.TextureFormatDepth32Float, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := rig.rec
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	dev := rig.frame.Device
	m := leakPBRModel(t)
	openClose := func() {
		u, err := viewport3d.Upload(context.Background(), dev, m, viewport3d.UploadOpts{})
		if err != nil {
			t.Fatal(err)
		}
		scene.Add(u.Items()...)
		pose := viewport3d.NewPose()
		u.SetPose(pose)
		rig.redraw()
		rec.CompleteReads()
		if st := rig.r.Stats(); st.Draws == 0 {
			t.Fatalf("the model drew nothing: %+v", st)
		}
		scene.Remove(u.Items()...)
		u.Release()
		rig.redraw()
		rec.CompleteReads()
		rig.ts.ClearLogs()
		rec.ClearLogs()
	}
	// The first time makes what the renderer keeps: the pipelines, the environment, the float target and the shadow map.
	openClose()
	base := dev.Resources().Total()
	baseLive := rec.Device().Live()
	leaktest.Run(t, "viewport3d/pbr-model", func() func() { return openClose },
		leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(rig.ts, rec)})
	if got := dev.Resources().Total(); got != base {
		t.Errorf("a PBR model opened and closed 400 times left %d GPU resources alive (%d before)", got, base)
	}
	if got := rec.Device().Live(); got != baseLive {
		t.Errorf("the device holds %+v after the model was closed 400 times, %+v before", got, baseLive)
	}
}

// TestLeaksPBREnvironment changes the environment of a scene with a PBR item, over and over, between the default one and cubes of the
// app's: each change makes the cubes of the new one and lets go of the old.
func TestLeaksPBREnvironment(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Sphere(0.5, 16, 8), viewport3d.NewPBRMaterial()))
	rig := newDrawRigOpts(t, scene, frontCamera(), gpu.TextureFormatDepth32Float, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := rig.rec
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	dev := rig.frame.Device
	cube, err := dev.NewTexture(gpu.TextureDesc{Label: "sky", Kind: gpu.TextureKindCube, Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 6}, MipLevelCount: 4,
		Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		t.Fatal(err)
	}
	defer cube.Release()
	envs := []*viewport3d.Environment{nil, viewport3d.NewEnvironment(cube)}
	step := func() func() {
		i := 0
		return func() {
			scene.SetStyle(viewport3d.SceneStyle{Environment: envs[i%2]})
			i++
			rig.redraw()
			rec.CompleteReads()
			rig.redraw()
			rec.CompleteReads()
			rig.ts.ClearLogs()
			rec.ClearLogs()
		}
	}
	step()()
	step()()
	leaktest.Run(t, "viewport3d/pbr-environment", step, leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(rig.ts, rec)})
}

// TestLeaksPBRView resizes a 3D view with PBR items, a shadow and tone mapping 250 times: each size makes a new float target, and
// the shadow map, the environment and the pipelines stay.
func TestLeaksPBRView(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: 1, Y: 1, Z: 1}}}, Shadow: viewport3d.ShadowStyle{Enabled: true, Size: 256},
		ToneMapping: viewport3d.ToneMappingACES})
	scene.Add(viewport3d.NewItem(scenetest.Sphere(0.5, 16, 8), viewport3d.NewPBRMaterial()), viewport3d.NewItem(scenetest.Box(2, 0.1, 2), viewport3d.NewPBRMaterial(), at(0, -0.6, 0)))
	cam := scenetest.Isometric(scene, 200, 100)
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := ts.GPUDevice()
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	leaktest.Run(t, "viewport3d/pbr-view", func() func() {
		i := 0
		return func() {
			ts.SetDevicePixelRatio(float64(1 + i%2))
			ts.PumpWidget(NewCenter(NewSizedBox(viewport3d.NewView(scene, cam), SizedBoxOpts{Width: foundation.Some(float64(60 + i%37)), Height: foundation.Some(float64(40 + i%23))})))
			ts.Pump(16 * time.Millisecond)
			rec.CompleteReads()
			ts.Pump(16 * time.Millisecond)
			ts.ClearLogs()
			rec.ClearLogs()
			i++
		}
	}, leaktest.Opts{N: 250, Warmup: 20, Counters: leakCounters(ts, rec)})
}

// TestLeaksPBRDeviceLoss loses the device under a view with PBR items, a shadow and a skin, over and over: the renderer lets go of
// what it held on the lost device (the environment, the float target, the shadow map, the storage buffers) and makes it again.
func TestLeaksPBRDeviceLoss(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: 1, Y: 1, Z: 1}}}, Shadow: viewport3d.ShadowStyle{Enabled: true, Size: 256}})
	skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity(), vecmath.NewMat4Translation(vecmath.Vec3{X: -1})})
	bar := viewport3d.NewItem(scenetest.SkinnedBar(), viewport3d.NewPBRMaterial())
	bar.SetSkin(skin)
	ball := viewport3d.NewItem(scenetest.MorphSphere(0.4, 16, 8), viewport3d.NewPBRMaterial(), at(0, 1, 0))
	ball.SetMorphWeights([]float32{0.5})
	scene.Add(bar, ball)
	cam := scenetest.Isometric(scene, 200, 100)
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := ts.GPUDevice()
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	ts.PumpWidget(NewCenter(NewSizedBox(viewport3d.NewView(scene, cam), SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})))
	leaktest.Run(t, "viewport3d/pbr-device-loss", func() func() {
		return func() {
			rec.Lose()
			cam.Orbit(0.01, 0)
			ts.Pump(16 * time.Millisecond)
			ts.Pump(16 * time.Millisecond)
			rec.CompleteReads()
			ts.ClearLogs()
			rec.ClearLogs()
		}
	}, leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(ts, rec)})
}
