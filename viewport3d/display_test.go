//go:build linux && glutter_display

package viewport3d_test

import (
	"image"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/vecmath"
)

// These tests draw the scenes of package scenetest on Vulkan with Mesa's lavapipe and the validation layer on, and compare the
// pictures with the goldens (SPEC §27.12). They need no display. A golden is made with -update-goldens and looked at.

// openLavapipe opens lavapipe with validation, and checks at the end of the test that the layer reported nothing.
func openLavapipe(t *testing.T) *vulkan.Device {
	t.Helper()
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[runtime.GOARCH]
	icd := "/usr/share/vulkan/icd.d/lvp_icd." + arch + ".json"
	if _, err := os.Stat(icd); err != nil {
		t.Skip("Mesa's lavapipe is not installed (sudo dnf install mesa-vulkan-drivers)")
	}
	if _, err := os.Stat("/usr/share/vulkan/explicit_layer.d/VkLayer_khronos_validation.json"); err != nil {
		t.Skip("the validation layer is not installed (sudo dnf install vulkan-validation-layers)")
	}
	t.Setenv("VK_DRIVER_FILES", icd)
	d, err := vulkan.Open(vulkan.Options{AllowCPU: true, Validation: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if errs := d.ValidationErrors(); len(errs) > 0 {
			t.Errorf("the validation layer reported %d errors:\n%s", len(errs), strings.Join(errs, "\n"))
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

// lavapipeApp is an app whose GPU device is lavapipe.
func lavapipeApp(t *testing.T) (*gpuhost.App, *gpu.Device) {
	t.Helper()
	app := gpuhost.NewBinding(t, openLavapipe(t))
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		t.Fatal(err)
	}
	return app, d
}

// deviceApp is the app of the picks tests: lavapipe here, Direct3D 12 under Wine (wine_test.go).
func deviceApp(t *testing.T) (*gpuhost.App, *gpu.Device) { return lavapipeApp(t) }

// keepGoing makes the Fatalf of the golden check an Errorf, so that one case that fails does not stop the others (the cases
// run on the test's goroutine, which package gpu requires: it is the UI thread's).
type keepGoing struct{ *testing.T }

func (k keepGoing) Fatalf(format string, args ...any) { k.Helper(); k.Errorf(format, args...) }

func TestScenesAgainstTheirGoldens(t *testing.T) {
	app, d := lavapipeApp(t)
	for _, c := range scenetest.Cases() {
		img, err := scenetest.Render(d, waitFor(app), c)
		if err != nil {
			t.Errorf("%s: %v", c.Name, err)
			continue
		}
		goldens.Check(keepGoing{t}, c.Name, img)
	}
}

func TestAPartFarFromTheOriginIsDrawnAsIfItWereAtTheOrigin(t *testing.T) {
	// The part is 50 km from the origin and moves by 1 mm: float32 positions there are 4 mm apart, so the picture would jump
	// or stand still. The matrices are relative to the eye (D30), so the picture is the one of the part at the origin.
	app, d := lavapipeApp(t)
	render := func(c scenetest.Case) *image.RGBA {
		img, err := scenetest.Render(d, waitFor(app), c)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	far0, far1 := render(scenetest.PartAt("far-a", scenetest.FarOrigin, 0)), render(scenetest.PartAt("far-b", scenetest.FarOrigin, 0.001))
	near1 := render(scenetest.PartAt("near-b", vecmath.Vec3{}, 0.001))
	stats := func(a, b *image.RGBA) (differing, biggest int) {
		for i := range a.Pix {
			diff := int(a.Pix[i]) - int(b.Pix[i])
			if diff < 0 {
				diff = -diff
			}
			if diff > 0 {
				differing++
			}
			biggest = max(biggest, diff)
		}
		return
	}
	// The same part at the origin: the same picture, but for the rounding of the float32 matrices.
	if n, big := stats(far1, near1); big > 2 {
		t.Errorf("the part 50 km away differs from the one at the origin: %d channels, by up to %d", n, big)
	}
	// And 1 mm is seen: the two far pictures differ, at the edges only, by no more than a sample of the multisampling.
	n, big := stats(far0, far1)
	if n == 0 || big > 64 {
		t.Errorf("1 mm of the far part changed %d channels, by up to %d; want some change at the edges, by one sample at most", n, big)
	}
}

func TestAGridFarFromTheOriginIsDrawnAsIfItWereAtTheOrigin(t *testing.T) {
	// The grid's plane and the part are 50 km from the origin: the lines are worked out from the eye's coordinates in the plane in
	// float64 (reduced to the major period), so the picture is the one at the origin (D30).
	app, d := lavapipeApp(t)
	render := func(c scenetest.Case) *image.RGBA {
		img, err := scenetest.Render(d, waitFor(app), c)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	near, far := render(scenetest.GridPartAt("grid-near", vecmath.Vec3{})), render(scenetest.GridPartAt("grid-far", scenetest.FarOrigin))
	differing, biggest := 0, 0
	for i := range near.Pix {
		diff := int(near.Pix[i]) - int(far.Pix[i])
		if diff < 0 {
			diff = -diff
		}
		if diff > 2 {
			differing++
		}
		biggest = max(biggest, diff)
	}
	if differing > 0 {
		t.Errorf("the grid 50 km away differs from the one at the origin: %d channels by more than 2, up to %d", differing, biggest)
	}
}
