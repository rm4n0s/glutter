package viewport3d_test

import (
	"context"
	"image/color"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	. "github.com/rm4n0s/glutter/widgets"
)

// The memory tests of the 3D view (step 30.13, SPEC §15 "Memory over time"): a model opened and closed 100 times, a view resized
// 1,000 times and a device lost over and over. The live heap, the goroutines, the resources that package gpu counts and what the
// recording device holds must not grow, and when a scenario ends nothing it made is left on the device.

// leakCounters are the numbers that must not grow in a scenario with a GPU device.
func leakCounters(ts *gluttertest.Tester, rec *gluttertest.GPURecorder) map[string]func() int {
	live := func(f func(fakedevice.Live) int) func() int { return func() int { return f(rec.Device().Live()) } }
	return map[string]func() int{
		"gpu resources": func() int {
			d, err := gpu.DeviceOfBinding(ts.Binding)
			if err != nil {
				return 0
			}
			return d.Resources().Total()
		},
		"buffers":    live(func(l fakedevice.Live) int { return l.Buffers }),
		"textures":   live(func(l fakedevice.Live) int { return l.Textures }),
		"samplers":   live(func(l fakedevice.Live) int { return l.Samplers }),
		"pipelines":  live(func(l fakedevice.Live) int { return l.Pipelines }),
		"registered": func() int { return rec.GPUTextures().Len() },
	}
}

// leakModel is a model with every kind of thing Upload makes a GPU resource of: meshes of triangles with and without the attributes,
// lines, points, a texture, and a mesh used by two nodes.
func leakModel(t *testing.T) *model.Model {
	t.Helper()
	tex := &model.Image{Name: "px.png", Type: "image/png", Data: pngBytes(t, 8, 8, color.NRGBA{R: 200, G: 100, B: 50, A: 255})}
	red := &model.Material{Name: "red", BaseColor: [4]float32{1, 0, 0, 1}, Shininess: 40}
	glass := &model.Material{Name: "glass", BaseColor: [4]float32{0, 0, 1, 0.5}, AlphaMode: model.AlphaBlend, BaseColorImage: tex, DoubleSided: true}
	a, b := triangleMesh("a"), triangleMesh("b")
	a.Material, b.Material = red, glass
	lines := &model.Mesh{Primitive: model.PrimitiveLines, Positions: []float32{0, 0, 0, 1, 1, 1}, Indices: []uint32{0, 1}, Material: red}
	points := &model.Mesh{Primitive: model.PrimitivePoints, Positions: []float32{0, 0, 0}, Indices: []uint32{0}}
	colored := triangleMesh("colored")
	colored.Colors = []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255}
	root, child := model.NewNode("root"), model.NewNode("child")
	child.Transform = vecmath.NewMat4Translation(vecmath.Vec3{Y: 5})
	child.Meshes = []*model.Mesh{a, b, lines, points, colored}
	root.Meshes = []*model.Mesh{a}
	root.Children = []*model.Node{child}
	return &model.Model{Nodes: []*model.Node{root}}
}

// TestLeaksModel opens a model into a scene, draws it, closes it and draws again, 100 times (the "open and close" of
// examples/cad): its meshes, materials and textures are made on the device and released, and nothing stays.
func TestLeaksModel(t *testing.T) {
	scene := viewport3d.NewScene()
	rig := newDrawRigOpts(t, scene, frontCamera(), gpu.TextureFormatDepth32Float, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := rig.rec
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	dev := rig.frame.Device
	m := leakModel(t)
	openClose := func() {
		u, err := viewport3d.Upload(context.Background(), dev, m, viewport3d.UploadOpts{FirstPickID: 1})
		if err != nil {
			t.Fatal(err)
		}
		scene.Add(u.Items()...)
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
	// The first time makes the renderer's own pipelines for the kinds of surface that the model has, which stay.
	openClose()
	base := dev.Resources().Total()
	baseLive := rec.Device().Live()
	leaktest.Run(t, "viewport3d/model", func() func() { return openClose },
		leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(rig.ts, rec)})
	if got := dev.Resources().Total(); got != base {
		t.Errorf("a model opened and closed 400 times left %d GPU resources alive (%d before)", got, base)
	}
	if got := rec.Device().Live(); got != baseLive {
		t.Errorf("the device holds %+v after the model was closed 400 times, %+v before", got, baseLive)
	}
}

// TestLeaksView resizes a 3D view 1,000 times, with a part in it and a pick now and then: each size makes new targets, and the
// renderer's buffers and pipelines stay.
func TestLeaksView(t *testing.T) {
	tests := []struct {
		name string
		step func(ts *gluttertest.Tester, scene *viewport3d.Scene, cam *viewport3d.Camera, ctl *viewport3d.ViewController, i int)
	}{
		{"resizes", func(ts *gluttertest.Tester, scene *viewport3d.Scene, cam *viewport3d.Camera, ctl *viewport3d.ViewController, i int) {
		}},
		{"resizes with picks", func(ts *gluttertest.Tester, scene *viewport3d.Scene, cam *viewport3d.Camera, ctl *viewport3d.ViewController, i int) {
			if i%3 == 0 {
				ctl.Pick(ui.Offset{Dx: 40, Dy: 30}, func(viewport3d.PickResult, error) {})
			}
		}},
		{"resizes with the pixel ratio and the camera moving", func(ts *gluttertest.Tester, scene *viewport3d.Scene, cam *viewport3d.Camera, ctl *viewport3d.ViewController, i int) {
			ts.SetDevicePixelRatio(float64(1 + i%3))
			cam.Orbit(0.01, 0.005)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.Add(scenetest.Part()...)
			cam := scenetest.Isometric(scene, 200, 100)
			ctl := viewport3d.NewViewController()
			ts := gluttertest.New(t, gluttertest.Opts{GPU: true, DiscardScenes: true})
			rec := ts.GPUDevice()
			rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
			leaktest.Run(t, "viewport3d/view/"+tt.name, func() func() {
				i := 0
				return func() {
					tt.step(ts, scene, cam, ctl, i)
					ts.PumpWidget(NewCenter(NewSizedBox(viewport3d.NewView(scene, cam, viewport3d.ViewOpts{Controller: ctl}),
						SizedBoxOpts{Width: foundation.Some(float64(60 + i%37)), Height: foundation.Some(float64(40 + i%23))})))
					ts.Pump(16 * time.Millisecond)
					rec.CompleteReads()
					ts.Pump(16 * time.Millisecond)
					ts.ClearLogs()
					rec.ClearLogs()
					i++
				}
			}, leaktest.Opts{N: 250, Warmup: 20, Counters: leakCounters(ts, rec)})
		})
	}
}

// TestLeaksDeviceLoss loses the device under a 3D view with a model in it, over and over: the renderer lets go of what it held on
// the lost device and makes it again on the new one, and the meshes of the scene are uploaded to each.
func TestLeaksDeviceLoss(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(scenetest.Part()...)
	cam := scenetest.Isometric(scene, 200, 100)
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := ts.GPUDevice()
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	view := viewport3d.NewView(scene, cam)
	ts.PumpWidget(NewCenter(NewSizedBox(view, SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})))
	leaktest.Run(t, "viewport3d/device-loss", func() func() {
		return func() {
			rec.Lose()
			cam.Orbit(0.01, 0) // the view draws on the new device
			ts.Pump(16 * time.Millisecond)
			ts.Pump(16 * time.Millisecond)
			rec.CompleteReads()
			ts.ClearLogs()
			rec.ClearLogs()
		}
	}, leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(ts, rec)})
	if st := rec.Device().Live(); st.Buffers == 0 {
		t.Errorf("the view has no buffers on the device after the losses: %+v", st)
	}
}
