//go:build linux && glutter_display

package viewport3d_test

import (
	"context"
	"io"
	"log/slog"
	"math"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/model/gltf"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// The benchmarks of SPEC §27.10 on a real Vulkan device, at 1920 x 1080 with 4x multisampling. They draw the frames of the scenes of
// package scenetest and report the CPU time of a frame (the time that the UI thread's thread spends running in Draw, Finish and
// Submit, without the time it waits for the GPU or the driver's threads) and the frame rate
// (frames a second with at most two in flight, so that the GPU's time is in it). By default the device is Mesa's lavapipe, a CPU
// rasterizer: its numbers are recorded, not targets. GLUTTER_BENCH_GPU=hardware uses the machine's GPU (run it through
// .claude/no-nvidia.sh, so that the NVIDIA eGPU is never touched; the owner's AMD GPU is the one of the targets):
//
//	.claude/no-nvidia.sh -- env GLUTTER_BENCH_GPU=hardware go test -tags glutter_display -run '^$' -bench Display -benchtime 300x ./viewport3d

const (
	benchWidth  = 1920
	benchHeight = 1080
)

func benchDevice(b *testing.B) (*gpuhost.App, *gpu.Device) {
	b.Helper()
	hardware := os.Getenv("GLUTTER_BENCH_GPU") == "hardware"
	opts := vulkan.Options{AllowCPU: !hardware, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if !hardware {
		arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[runtime.GOARCH]
		icd := "/usr/share/vulkan/icd.d/lvp_icd." + arch + ".json"
		if _, err := os.Stat(icd); err != nil {
			b.Skip("Mesa's lavapipe is not installed (sudo dnf install mesa-vulkan-drivers)")
		}
		b.Setenv("VK_DRIVER_FILES", icd)
	}
	vk, err := vulkan.Open(opts)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := vk.Close(); err != nil {
			b.Error(err)
		}
	})
	b.Logf("device: %s", vk.DeviceName())
	app := gpuhost.NewBinding(b, vk)
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		b.Fatal(err)
	}
	return app, d
}

// threadCPU is the CPU time that the calling thread has used (user and system): a thread that waits for the GPU does not use any.
func threadCPU(b *testing.B) time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(1, &ru); err != nil { // RUSAGE_THREAD
		b.Fatal(err)
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

func benchDisplay(b *testing.B, scene *viewport3d.Scene, cam *viewport3d.Camera) {
	b.Helper()
	benchDisplayWith(b, func(*gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) { return scene, cam, nil })
}

// benchDisplayWith is benchDisplay for a scene that needs the device to be made (a model uploaded to it): setup returns the scene, the
// camera and a function that changes the scene before each frame (nil when nothing changes).
func benchDisplayWith(b *testing.B, setup func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func())) {
	b.Helper()
	runtime.LockOSThread() // threadCPU measures this thread
	b.Cleanup(runtime.UnlockOSThread)
	app, d := benchDevice(b)
	scene, cam, each := setup(d)
	size := gpu.Extent3D{Width: benchWidth, Height: benchHeight, DepthOrArrayLayers: 1}
	must := func(t *gpu.Texture, err error) *gpu.Texture {
		if err != nil {
			b.Fatal(err)
		}
		return t
	}
	resolve := must(d.NewTexture(gpu.TextureDesc{Label: "bench resolve", Size: size, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageTextureBinding}))
	color := must(d.NewTexture(gpu.TextureDesc{Label: "bench color", Size: size, Format: gpu.TextureFormatRGBA8Unorm, SampleCount: 4,
		Usage: gpu.TextureUsageRenderAttachment}))
	depth := must(d.NewTexture(gpu.TextureDesc{Label: "bench depth", Size: size, Format: gpu.TextureFormatDepth24PlusStencil8, SampleCount: 4,
		Usage: gpu.TextureUsageRenderAttachment}))
	r := viewport3d.NewRenderer(scene, cam)
	r.Init(d)
	b.Cleanup(func() {
		r.Dispose()
		depth.Release()
		color.Release()
		resolve.Release()
	})
	inFlight := 0
	frame := func() time.Duration {
		if each != nil {
			each()
		}
		start := threadCPU(b)
		f := gpu.ViewFrame{Device: d, Encoder: d.NewCommandEncoder(), Color: color, Resolve: resolve, Depth: depth, Width: benchWidth, Height: benchHeight,
			DevicePixelRatio: 1, Time: time.Second}
		r.Draw(&f)
		if err := d.Queue().Submit(f.Encoder.Finish()); err != nil {
			b.Fatal(err)
		}
		spent := threadCPU(b) - start
		inFlight++
		d.Queue().OnSubmittedWorkDone(func() { inFlight-- })
		return spent
	}
	wait := func(limit int) {
		for i := 0; inFlight > limit && i < 100000; i++ {
			app.RunPosted()
			if inFlight > limit {
				time.Sleep(100 * time.Microsecond)
			}
		}
		if inFlight > limit {
			b.Fatal("the GPU did not finish its frames")
		}
	}
	// The first frames make the pipelines, the buffers and the edges.
	for i := 0; i < 3; i++ {
		frame()
		wait(0)
	}
	if st := r.Stats(); st.Draws == 0 {
		b.Fatalf("the frame drew nothing: %+v", st)
	}
	b.ReportAllocs()
	var cpu time.Duration
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wait(1)
		cpu += frame()
	}
	wait(0)
	elapsed := b.Elapsed()
	b.ReportMetric(float64(cpu)/float64(b.N)/1e6, "cpu-ms/frame")
	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "fps")
}

// BenchmarkDisplayMillionTrianglesWithEdges draws a part of 1,002,528 triangles with its edges (targets, on the owner's GPU: 60 fps, 2 ms of CPU).
func BenchmarkDisplayMillionTrianglesWithEdges(b *testing.B) {
	scene, cam := scenetest.TerraceScene(benchWidth, benchHeight)
	benchDisplay(b, scene, cam)
}

// BenchmarkDisplay10000InstancedItems draws 10,000 items that share a mesh (target: 1 ms of CPU).
func BenchmarkDisplay10000InstancedItems(b *testing.B) {
	scene, cam := scenetest.InstancedScene(benchWidth, benchHeight)
	benchDisplay(b, scene, cam)
}

// BenchmarkDisplay10000SeparateItems draws 10,000 items that each have a mesh (target: 4 ms of CPU).
func BenchmarkDisplay10000SeparateItems(b *testing.B) {
	scene, cam := scenetest.SeparateScene(benchWidth, benchHeight)
	benchDisplay(b, scene, cam)
}

// BenchmarkDisplayGame draws a frame of examples/game3d (SPEC §28.6: at least 60 frames a second on the owner's machine): the level
// (PBR, a shadow) and the skinned character, which moves, and the coins, which turn, in every frame, with the camera behind. Its models
// are the files that the example embeds.
func BenchmarkDisplayGame(b *testing.B) {
	benchDisplayWith(b, func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		assets := os.DirFS("../examples/game3d/assets")
		ctx := context.Background()
		levelModel, err := gltf.Load(ctx, assets, "level.glb")
		if err != nil {
			b.Fatal(err)
		}
		charModel, err := gltf.Load(ctx, assets, "character.glb")
		if err != nil {
			b.Fatal(err)
		}
		level, err := viewport3d.Upload(ctx, d, levelModel)
		if err != nil {
			b.Fatal(err)
		}
		character, err := viewport3d.Upload(ctx, d, charModel)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() {
			level.Release()
			character.Release()
		})
		scene := viewport3d.NewScene()
		scene.Add(level.Items()...)
		scene.Add(character.Items()...)
		scene.SetStyle(viewport3d.SceneStyle{
			Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: -0.45, Y: 1, Z: 0.3}, Intensity: 1.8}},
			Shadow: viewport3d.ShadowStyle{Enabled: true, Size: 2048},
		})
		cam := viewport3d.NewCamera(viewport3d.CameraOpts{Distance: 7, FieldOfView: 0.8, Near: 0.1, Target: vecmath.Vec3{X: -7, Y: 2.8, Z: -7}})
		cam.SetViewportSize(ui.Size{Width: benchWidth, Height: benchHeight})
		cam.SetOrientation(vecmath.NewQuatAxisAngle(vecmath.Vec3{Y: 1}, 3.6).Mul(vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1}, -0.36)))
		run := viewport3d.NewAnimation(charModel.Animations[1])
		pose := viewport3d.NewPose()
		var coins []*viewport3d.Item
		for _, p := range level.Parts {
			if strings.HasPrefix(p.Node.Name, "Coin") {
				coins = append(coins, p.Item)
			}
		}
		t := 0.0
		return scene, cam, func() {
			t += 1.0 / 60
			pose.Clear()
			run.Sample(float32(math.Mod(t, float64(run.Duration()))), pose)
			character.SetPose(pose)
			character.SetTransform(vecmath.NewMat4Translation(vecmath.Vec3{X: -7, Y: 1.4, Z: -7}))
			for i, c := range coins {
				c.SetTransform(vecmath.NewMat4Translation(vecmath.Vec3{X: float64(i), Y: 1, Z: 0}).Mul(vecmath.NewMat4RotationY(t)))
			}
			cam.SetTarget(vecmath.Vec3{X: -7 + 0.1*math.Sin(t), Y: 2.8, Z: -7})
		}
	})
}
