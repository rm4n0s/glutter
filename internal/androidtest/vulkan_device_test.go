//go:build glutter_android

package androidtest

import (
	"context"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
)

// withValidation is the build option that packs the validation layer, and the extra that turns it
// on. A device test of Vulkan fails, with the way to get the layer, when it is not in
// .android-test/vvl (SPEC §24.13).
func withValidation(t *testing.T) (androidtool.BuildOptions, map[string]string) {
	t.Helper()
	dir, err := ValidationLayerDir()
	if err != nil {
		t.Fatal(Describe(err))
	}
	return androidtool.BuildOptions{ValidationLayerDir: dir}, map[string]string{"GLUTTER_VULKAN_VALIDATION": "1"}
}

// setRotation turns the emulator (0 portrait, 1 landscape, 2 upside down, 3 the other landscape) and
// turns it back to portrait when the test ends.
func setRotation(ctx context.Context, t *testing.T, e *Emulator, rotation int) {
	t.Helper()
	shell(ctx, t, e, "settings", "put", "system", "accelerometer_rotation", "0")
	t.Cleanup(func() {
		_, _ = e.ADB.Output(context.Background(), "shell", "settings", "put", "system", "user_rotation", "0")
	})
	shell(ctx, t, e, "settings", "put", "system", "user_rotation", fmt.Sprint(rotation))
	time.Sleep(2 * time.Second)
}

// checkNoValidationErrors fails when the validation layer said anything of the run.
func checkNoValidationErrors(t *testing.T, files map[string][]byte) {
	t.Helper()
	if msg := strings.TrimSpace(string(files["validation.txt"])); msg != "" {
		t.Errorf("the validation layer reported errors:\n%s", msg)
	}
	if dev := string(files["device.txt"]); !strings.Contains(dev, "vulkan:") {
		t.Errorf("the parity run did not use Vulkan: %q", dev)
	}
}

// TestVulkanParityWithSoftAndValidation draws every proof scene through the explicit renderer on the
// emulator's Vulkan device, with the validation layer on, and compares with the software renderer's
// images drawn on the device (step 25.5).
func TestVulkanParityWithSoftAndValidation(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	opts, extras := withValidation(t)
	files := runParity(ctx, t, e, "vulkan", extras, opts)
	checkNoValidationErrors(t, files)
	checkParity(t, files)
}

// TestVulkanParityInLandscapeWithValidation is the same with the emulator turned: the swapchain then
// has a pre-transform, its images are turned, and the validation layer checks the whole sequence.
func TestVulkanParityInLandscapeWithValidation(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	opts, extras := withValidation(t)
	setRotation(ctx, t, e, 1)
	files := runParity(ctx, t, e, "vulkan", extras, opts)
	checkNoValidationErrors(t, files)
	checkParity(t, files)
}

// TestVulkanDrawsUprightInEveryOrientation turns the emulator through the four rotations while the
// counter runs with Vulkan and with soft: when the swapchain has a pre-transform (the log says which)
// the frame is drawn rotated to it, and either way the screen shows the same upright counter: the
// Vulkan screen equals the software one (the part below the status bar, whose clock differs).
func TestVulkanDrawsUprightInEveryOrientation(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/counter")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	shot := func(renderer string) (*image.RGBA, string) {
		t.Helper()
		_, _ = e.ADB.Output(ctx, "shell", "am", "force-stop", appID)
		if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": renderer}); err != nil {
			t.Fatal(Describe(err))
		}
		log, err := e.WaitLog(ctx, appID, "renderer chosen", time.Minute)
		if err != nil {
			t.Fatal(Describe(err))
		}
		time.Sleep(3 * time.Second)
		b, err := e.ADB.Screencap(ctx)
		if err != nil {
			t.Fatal(Describe(err))
		}
		return decodePNG(t, b), log
	}
	preRotated := false
	defer func() {
		if !preRotated {
			t.Error("no orientation made a swapchain with a pre-transform: the pre-rotation was not exercised")
		}
	}()
	for rotation := 0; rotation < 4; rotation++ {
		t.Run(fmt.Sprintf("rotation %d", rotation), func(t *testing.T) {
			setRotation(ctx, t, e, rotation)
			soft, _ := shot("soft")
			vk, log := shot("vulkan")
			// The system reports the transform of a surface when it likes (for a half turn, which
			// changes no size, often not at all: the compositor then turns the frame, correctly);
			// what the swapchain was made with is in the log.
			for turns := 1; turns < 4; turns++ {
				if strings.Contains(log, fmt.Sprintf("pre_rotation=%d", turns)) {
					preRotated = true
				}
			}
			for _, l := range strings.Split(log, "\n") {
				if strings.Contains(l, "Vulkan swapchain") {
					t.Log(strings.TrimSpace(l))
				}
			}
			if vk.Rect != soft.Rect {
				t.Fatalf("the Vulkan screen is %v, the software one %v", vk.Rect, soft.Rect)
			}
			wantSize := image.Pt(1080, 2400)
			if rotation%2 == 1 {
				wantSize = image.Pt(2400, 1080)
			}
			if vk.Rect.Size() != wantSize {
				t.Errorf("the screen is %v, want %v", vk.Rect.Size(), wantSize)
			}
			band := image.Rect(0, 160, vk.Rect.Dx(), vk.Rect.Dy()-120)
			gputest.Tolerance{Over8Permille: 1}.Check(t, "below the status bar", gputest.DiffRect(vk, soft, band))
		})
	}
}

// TestAutoChoosesVulkanOnTheEmulator checks `auto` on the emulator, which has Android 17 and Vulkan
// 1.3 on its (virtual) GPU, and the log that says what was chosen and on what: the device, the
// Vulkan version, the swapchain (step 25.5).
func TestAutoChoosesVulkanOnTheEmulator(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/counter")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	if err := e.Launch(ctx, appID, nil); err != nil {
		t.Fatal(Describe(err))
	}
	log, err := e.WaitLog(ctx, appID, "renderer chosen", time.Minute)
	if err != nil {
		t.Fatal(Describe(err))
	}
	for _, want := range []string{`INFO Vulkan device device="vulkan: `, "api=1.", "INFO Vulkan swapchain", "present_mode=", `INFO renderer chosen renderer="vulkan: `} {
		if !strings.Contains(log, want) {
			t.Errorf("the log lacks %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "renderer not usable") {
		t.Errorf("auto tried a renderer that was not usable:\n%s", log)
	}
}

// TestWithoutVulkanNamedFailsAndAutoUsesGLES starts an emulator without Vulkan (-feature -Vulkan):
// GLUTTER_RENDERER=vulkan fails with the tag of the Vulkan error and no fallback, and auto says why
// Vulkan was not used and draws with gles.
func TestWithoutVulkanNamedFailsAndAutoUsesGLES(t *testing.T) {
	StopShared() // one emulator at a time
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	e, err := Start(ctx, Options{AVD: "glutter-37", Args: []string{"-feature", "-Vulkan"}})
	if err != nil {
		t.Fatalf("starting glutter-37 without Vulkan: %v", Describe(err))
	}
	t.Cleanup(e.Stop) // registered first, so it runs after the cleanup below that still talks to the emulator
	appID := installApp(ctx, t, e, "./examples/counter")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })

	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "vulkan"}); err != nil {
		t.Fatal(Describe(err))
	}
	if err := e.WaitExit(ctx, appID, time.Minute); err != nil {
		log, _ := e.Log(ctx, appID)
		t.Fatalf("the app that asked for Vulkan still runs: %s\n%s", Describe(err), log)
	}
	// The error that ended Run is the last thing the app logs.
	log, err := e.WaitLog(ctx, appID, "ERROR glutter:", 30*time.Second)
	if err != nil {
		full, _ := e.FullLog(ctx)
		var about []string
		for _, l := range strings.Split(full, "\n") {
			if strings.Contains(l, "glutter") || strings.Contains(l, "AndroidRuntime") || strings.Contains(l, "ActivityManager") || strings.Contains(l, "ActivityTaskManager") || strings.Contains(l, "libc ") {
				if !strings.Contains(l, "adbd") && !strings.Contains(l, "logdr") {
					about = append(about, l)
				}
			}
		}
		t.Fatalf("%s\n%s", Describe(err), strings.Join(about, "\n"))
	}
	if !strings.Contains(log, "err.Tag=Vulkan") {
		t.Errorf("the log does not say that Vulkan failed with the tag of its error:\n%s", log)
	}
	if strings.Contains(log, "renderer chosen") {
		t.Errorf("a renderer was chosen although Vulkan was asked for by name:\n%s", log)
	}

	if err := e.Launch(ctx, appID, nil); err != nil {
		t.Fatal(Describe(err))
	}
	log, err = e.WaitLog(ctx, appID, "renderer chosen", time.Minute)
	if err != nil {
		t.Fatal(Describe(err))
	}
	for _, want := range []string{"renderer not usable", "renderer=vulkan", "tag=Vulkan", `renderer chosen renderer="gles: `} {
		if !strings.Contains(log, want) {
			t.Errorf("the log of auto lacks %q:\n%s", want, log)
		}
	}
}
