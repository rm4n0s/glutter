package viewport3d_test

import (
	"context"
	"os"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/model/gltf"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// TestLeaksTheGameLevel loads the level and the character of examples/game3d (glTF files that the example embeds) 100 times, as a
// game that starts a level again does (step 31.11, SPEC §15 "Memory over time"): the files are read, uploaded, drawn with a shadow with
// the character's skeleton in four poses, and released. The live heap, the goroutines and the resources of package gpu must not grow,
// and nothing is left on the device.
func TestLeaksTheGameLevel(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: -0.45, Y: 1, Z: 0.3}}}, Shadow: viewport3d.ShadowStyle{Enabled: true, Size: 512}})
	rig := newDrawRigOpts(t, scene, frontCamera(), gpu.TextureFormatDepth32Float, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := rig.rec
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	dev := rig.frame.Device
	assets := os.DirFS("../examples/game3d/assets")
	pose := viewport3d.NewPose()
	loadLevel := func() {
		ctx := context.Background()
		levelModel, err := gltf.Load(ctx, assets, "level.glb")
		if err != nil {
			t.Fatal(err)
		}
		charModel, err := gltf.Load(ctx, assets, "character.glb")
		if err != nil {
			t.Fatal(err)
		}
		level, err := viewport3d.Upload(ctx, dev, levelModel)
		if err != nil {
			t.Fatal(err)
		}
		character, err := viewport3d.Upload(ctx, dev, charModel)
		if err != nil {
			t.Fatal(err)
		}
		scene.Add(level.Items()...)
		scene.Add(character.Items()...)
		for i, an := range charModel.Animations {
			pose.Clear()
			viewport3d.NewAnimation(an).Sample(an.Duration()*float32(i)/4, pose)
			character.SetPose(pose)
			character.SetTransform(vecmath.NewMat4Translation(vecmath.Vec3{X: float64(i)}))
			rig.redraw()
			rec.CompleteReads()
		}
		if st := rig.r.Stats(); st.Draws == 0 {
			t.Fatalf("the level drew nothing: %+v", st)
		}
		scene.Remove(level.Items()...)
		scene.Remove(character.Items()...)
		level.Release()
		character.Release()
		rig.redraw()
		rec.CompleteReads()
		rig.ts.ClearLogs()
		rec.ClearLogs()
	}
	// The first time makes what the renderer keeps: the pipelines, the environment, the float target and the shadow map.
	loadLevel()
	base := dev.Resources().Total()
	baseLive := rec.Device().Live()
	leaktest.Run(t, "viewport3d/game level", func() func() { return loadLevel },
		leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(rig.ts, rec)})
	if got := dev.Resources().Total(); got != base {
		t.Errorf("the level loaded 500 times left %d GPU resources alive (%d before)", got, base)
	}
	if got := rec.Device().Live(); got != baseLive {
		t.Errorf("the device holds %+v after the level was released 500 times, %+v before", got, baseLive)
	}
}
