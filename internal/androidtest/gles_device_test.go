//go:build glutter_android

package androidtest

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/proof"
)

// runParity runs the parity app with the renderer on the emulator and returns the images it drew
// of every proof scene, by scene name, with the notes it wrote.
func runParity(ctx context.Context, t *testing.T, e *Emulator, renderer string, extras map[string]string, opts androidtool.BuildOptions) map[string][]byte {
	t.Helper()
	appID := installAppWith(ctx, t, e, "./examples/dev/android-parity", opts)
	all := map[string]string{"GLUTTER_RENDERER": renderer}
	for k, v := range extras {
		all[k] = v
	}
	if err := e.Launch(ctx, appID, all); err != nil {
		t.Fatal(Describe(err))
	}
	if err := e.WaitExit(ctx, appID, 4*time.Minute); err != nil {
		log, _ := e.Log(ctx, appID)
		t.Fatalf("%s\n%s", Describe(err), log)
	}
	files, err := e.PullAppDir(ctx, appID, "files/parity/"+renderer)
	if err != nil {
		t.Fatal(Describe(err))
	}
	ref, err := e.PullAppDir(ctx, appID, "files/parity/soft")
	if err != nil {
		t.Fatal(Describe(err))
	}
	for name, b := range ref {
		files["soft/"+name] = b
	}
	if _, ok := files["DONE"]; !ok {
		log, _ := e.Log(ctx, appID)
		t.Fatalf("the parity run did not finish (files: %d)\n%s", len(files), log)
	}
	return files
}

// checkParity compares the images of the GPU renderer with the software renderer's, both drawn on the
// device (the host's fonts are not the device's), scene by scene, within the tolerance of the GPU
// renderers (gputest.ProofTolerances), and checks that nothing was skipped.
func checkParity(t *testing.T, files map[string][]byte) {
	t.Helper()
	for _, name := range proof.Names() {
		name := name
		t.Run(name, func(t *testing.T) {
			tol, ok := gputest.ProofTolerances[name]
			if !ok {
				t.Fatalf("the proof scene %q has no tolerance in gputest.ProofTolerances", name)
			}
			wantData, ok := files["soft/"+name+".png"]
			if !ok {
				t.Fatalf("the device drew no software image of %s", name)
			}
			want := decodePNG(t, wantData)
			data, ok := files[name+".png"]
			if !ok {
				t.Fatalf("the device drew no image of %s", name)
			}
			got := decodePNG(t, data)
			if dir := os.Getenv("GLUTTER_ANDROIDTEST_DUMP"); dir != "" {
				_ = os.WriteFile(filepath.Join(dir, name+"-device.png"), data, 0o644)
				_ = os.WriteFile(filepath.Join(dir, name+"-soft.png"), wantData, 0o644)
			}
			if got.Rect.Size() != want.Rect.Size() {
				t.Fatalf("the image is %v, want %v", got.Rect, want.Rect)
			}
			if name == "gradients" {
				tol.Over96 = gradientSeamOver96
			}
			tol.Check(t, name, gputest.DiffRect(got, want, got.Bounds()))
			if note := strings.TrimSpace(string(files[name+".txt"])); note != "skipped 0" {
				t.Errorf("the renderer %s", note)
			}
		})
	}
}

// gradientSeamOver96 is how many pixels of the gradients scene may differ by more than 96 levels on
// the device: the repeat seams of a diagonal gradient run through pixel centers, where t is exactly
// a whole number and each GPU decides for itself which end of the ramp the pixel gets (about half the
// pixels of the two seams, 80, on the emulator's GPU; llvmpipe, the Linux reference, has none).
const gradientSeamOver96 = 200

// TestGLESParityWithSoft draws every proof scene with gles on the emulator and compares with soft.
func TestGLESParityWithSoft(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	checkParity(t, runParity(ctx, t, e, "gles", nil, androidtool.BuildOptions{}))
}

// TestGLESIsUsedAndKeepsItsTexturesOnHomeAndBack checks that GLUTTER_RENDERER given as an extra is used, and that Home and back gives the existing context a new surface, whose first frame
// makes nothing again (no new tiles rasterized, no new uploads).
func TestGLESIsUsedAndKeepsItsTexturesOnHomeAndBack(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	appID := installApp(ctx, t, e, "./examples/counter")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })

	// gles asked for as an extra is used (auto chooses Vulkan on the emulator since step 25.5).
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "gles"}); err != nil {
		t.Fatal(Describe(err))
	}
	log, err := e.WaitLog(ctx, appID, "renderer chosen", time.Minute)
	if err != nil {
		t.Fatal(Describe(err))
	}
	if !strings.Contains(log, `renderer="gles: `) {
		t.Errorf("the renderer is not gles:\n%s", log)
	}

	// Home, and when Android has taken the surface away (it does so some time after the activity
	// stopped), back in the same process.
	pid, err := e.ADB.Pid(ctx, appID)
	if err != nil || pid == 0 {
		t.Fatalf("the app has no process before Home: %v", err)
	}
	shell(ctx, t, e, "input", "keyevent", "KEYCODE_HOME")
	if _, err := e.WaitLog(ctx, appID, "surface released", 30*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	shell(ctx, t, e, "am", "start", "-n", appID+"/"+androidtool.ActivityClass)
	if again, _ := e.ADB.Pid(ctx, appID); again != pid {
		t.Fatalf("the process changed: %d, then %d", pid, again)
	}
	log, err = e.WaitLog(ctx, appID, "first frame on the new surface", 30*time.Second)
	if err != nil {
		full, _ := e.FullLog(ctx)
		t.Fatalf("%s\n%s", Describe(err), full[max(0, len(full)-6000):])
	}
	if !strings.Contains(log, "OpenGL ES surface attached to the existing context") {
		t.Errorf("the new surface was not attached to the existing context:\n%s", log)
	}
	if strings.Count(log, "OpenGL ES renderer ready") != 1 {
		t.Errorf("the context was made again:\n%s", log)
	}
	var line string
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, "first frame on the new surface") {
			line = l
		}
	}
	if !strings.Contains(line, "tiles_rasterized=0") || !strings.Contains(line, "images_uploaded=0") {
		t.Errorf("the first frame on the new surface made things again: %s", line)
	}

	// And it looks as before (GPU text differs a little from the software golden).
	time.Sleep(2 * time.Second)
	b, err := e.ADB.Screencap(ctx)
	if err != nil {
		t.Fatal(Describe(err))
	}
	got := decodePNG(t, b)
	mod, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := goldens.ReadPNG(filepath.Join(mod, "examples", "counter", "testdata", "goldens", "counter-android.png"))
	if err != nil {
		t.Fatal(err)
	}
	band := image.Rect(0, 1000, 1080, 1400)
	gputest.Tolerance{Over8Permille: 40, Over32Permille: 15, Over96: 400}.Check(t, "the middle of the counter",
		gputest.DiffRect(got, want, band))
}
