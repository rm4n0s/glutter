package viewport3d_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/viewport3d"
)

// The CPU benchmarks of SPEC §27.10. A frame is built on a quiet recording device, which keeps no log: what is timed is the work of
// the renderer and of package gpu on the UI thread (the culling, the sorting, the batches, the matrices, the commands and their
// replay), which does not depend on the GPU. The time of the GPU itself is measured on a real device (display_bench_test.go). Run
// them with go test -run '^$' -bench Frame ./viewport3d.

// timedFrames runs b.N frames of the rig and reports the CPU time of a frame in ms/frame: the time of the renderer's Draw, of
// Finish and of Submit, without the completion of the fake device's reads (which the GPU does). before runs untimed ahead of each frame.
func timedFrames(b *testing.B, rig *drawRig, before func()) {
	b.Helper()
	steadyFrame(b, rig)
	// The view is the size of the rig, not the 1920 x 1080 of the camera that the scene made: fit the camera to it, so that every item
	// is in view and none is culled (the worst frame).
	rig.cam.FitTo(rig.scene.Bounds())
	steadyFrame(b, rig)
	if st := rig.r.Stats(); st.Draws == 0 || st.Culled != 0 {
		b.Fatalf("the frame drew nothing, or culled items: %+v", st)
	}
	f := rig.frame
	b.ReportAllocs()
	var total time.Duration
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if before != nil {
			before()
		}
		start := time.Now()
		f.Encoder = f.Device.NewCommandEncoder()
		rig.r.Draw(&f)
		if err := f.Device.Queue().Submit(f.Encoder.Finish()); err != nil {
			b.Fatal(err)
		}
		total += time.Since(start)
		rig.rec.CompleteReads()
	}
	b.ReportMetric(float64(total)/float64(b.N)/1e6, "cpu-ms/frame")
}

func benchFrame(b *testing.B, scene *viewport3d.Scene, cam *viewport3d.Camera) {
	b.Helper()
	timedFrames(b, newDrawRig(b, scene, cam), nil)
}

// BenchmarkFrameMillionTrianglesWithEdges is a steady frame of a part of 1,002,528 triangles with its edges (target: 2 ms of CPU).
func BenchmarkFrameMillionTrianglesWithEdges(b *testing.B) {
	scene, cam := scenetest.TerraceScene(1920, 1080)
	benchFrame(b, scene, cam)
}

// BenchmarkFrame10000InstancedItems is a steady frame of 10,000 items that share a mesh and a material (target: 1 ms of CPU).
func BenchmarkFrame10000InstancedItems(b *testing.B) {
	scene, cam := scenetest.InstancedScene(1920, 1080)
	benchFrame(b, scene, cam)
}

// BenchmarkFrame10000SeparateItems is a steady frame of 10,000 items that each have a mesh of their own (target: 4 ms of CPU).
func BenchmarkFrame10000SeparateItems(b *testing.B) {
	scene, cam := scenetest.SeparateScene(1920, 1080)
	benchFrame(b, scene, cam)
}

// BenchmarkFrameAfterTheCameraMoved is a frame of the same 10,000 instanced items in which the camera has moved, which is the frame of
// a navigation (the culling and the matrices are made again).
func BenchmarkFrameAfterTheCameraMoved(b *testing.B) {
	scene, cam := scenetest.InstancedScene(1920, 1080)
	timedFrames(b, newDrawRig(b, scene, cam), func() { cam.Orbit(0.001, 0) })
}
