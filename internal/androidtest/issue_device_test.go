//go:build glutter_android

package androidtest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
	"github.com/rm4n0s/glutter/internal/issues"
)

// runIssue runs the reproduction of a reported issue (internal/issues) in the parity app, with the
// renderer, on the emulator, and returns the files the app wrote: report.txt, frame.png, device.txt
// and validation.txt. The run fails when the app ends without finishing, which is what a frame
// that fails does: the log of the app says why.
func runIssue(ctx context.Context, t *testing.T, e *Emulator, issue, renderer string, extras map[string]string, opts androidtool.BuildOptions) map[string][]byte {
	t.Helper()
	appID := installAppWith(ctx, t, e, "./examples/dev/android-parity", opts)
	all := map[string]string{"GLUTTER_RENDERER": renderer, issues.Env: issue}
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
	files, err := e.PullAppDir(ctx, appID, "files/issues/"+issue+"/"+renderer)
	if err != nil {
		log, _ := e.Log(ctx, appID)
		t.Fatalf("%s\n%s", Describe(err), log)
	}
	if _, ok := files["DONE"]; !ok {
		log, _ := e.Log(ctx, appID)
		t.Fatalf("the run of the issue %s did not finish (files: %d)\n%s", issue, len(files), log)
	}
	return files
}

// TestAndroidDrawsAFrameThatFillsTheAtlas draws, with the emulator's Vulkan (with the validation
// layer when it is installed) and its OpenGL ES, the frame that stopped Fishball (step 46.1):
// shadows whose masks need more tiles at once than the eight pages the coverage atlas keeps. It is
// the reproduction that every platform runs (internal/issues): the frame is drawn with nothing left
// out; a small frame after it frees the pages over the eight; and the first frame again, on pages
// made again, has the pixels it had. Before the step the first frame failed with AtlasFull, and
// the app ended without its report.
func TestAndroidDrawsAFrameThatFillsTheAtlas(t *testing.T) {
	tests := []struct {
		renderer string
		device   string // what device.txt says of the renderer
	}{
		{"vulkan", "vulkan:"},
		{"gles", ""},
	}
	e := Shared(t, "glutter-37")
	for _, tt := range tests {
		t.Run(tt.renderer, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
			defer cancel()
			var (
				opts   androidtool.BuildOptions
				extras map[string]string
			)
			validated := false
			if tt.renderer == "vulkan" {
				if dir, err := ValidationLayerDir(); err == nil {
					opts, extras, validated = androidtool.BuildOptions{ValidationLayerDir: dir}, map[string]string{"GLUTTER_VULKAN_VALIDATION": "1"}, true
				} else {
					t.Logf("drawn without the validation layer: %s", Describe(err))
				}
			}
			files := runIssue(ctx, t, e, issues.AtlasFullName, tt.renderer, extras, opts)
			device := strings.TrimSpace(string(files["device.txt"]))
			t.Logf("%s on %q", tt.renderer, device)
			if !strings.Contains(device, tt.device) {
				t.Errorf("the run did not use %s: %q", tt.renderer, device)
			}
			if validated {
				if msg := strings.TrimSpace(string(files["validation.txt"])); msg != "" {
					t.Errorf("the validation layer reported errors:\n%s", msg)
				}
			}
			if dir := os.Getenv("GLUTTER_TEST_DUMP"); dir != "" {
				if err := os.WriteFile(filepath.Join(dir, "android-"+tt.renderer+"-many-shadows.png"), files["frame.png"], 0o644); err != nil {
					t.Errorf("dump: %v", err)
				}
			}
			rep, err := issues.ParseAtlasFull(strings.Split(string(files["report.txt"]), "\n"))
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range rep.Lines() {
				t.Log(line)
			}
			for _, problem := range rep.Problems() {
				t.Error(problem)
			}
		})
	}
}
