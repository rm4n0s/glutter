//go:build glutter_android

package androidtest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// exampleNames are the directories of examples/ that hold a main package, except those that are not for a
// phone (SPEC §19): a phone has no second window and no tray.
func exampleNames(t *testing.T) []string {
	t.Helper()
	mod, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(mod, "examples"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "dev" || e.Name() == "multiwindow" || e.Name() == "tray" {
			continue
		}
		if _, err := os.Stat(filepath.Join(mod, "examples", e.Name(), "main.go")); err == nil {
			out = append(out, e.Name())
		}
	}
	return out
}

// TestEveryExampleDrawsItsFirstFrame installs each example (step 26.5) and starts it with the renderer of its
// own choice (Vulkan or OpenGL ES, as the app would on a phone): the app presents its first frame and the log
// has no panic.
func TestEveryExampleDrawsItsFirstFrame(t *testing.T) {
	names := exampleNames(t)
	if len(names) < 9 {
		t.Fatalf("found %d examples, want at least 9: %v", len(names), names)
	}
	for _, avd := range []string{"glutter-37", "glutter-26"} {
		t.Run(avd, func(t *testing.T) {
			e := Shared(t, avd)
			for _, name := range names {
				t.Run(name, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
					defer cancel()
					appID := installApp(ctx, t, e, "./examples/"+name)
					t.Cleanup(func() {
						_, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID)
						_, _ = e.ADB.Output(context.Background(), "uninstall", appID)
					})
					if err := e.Launch(ctx, appID, nil); err != nil {
						t.Fatal(Describe(err))
					}
					log, err := e.WaitLog(ctx, appID, "first frame presented", 90*time.Second)
					if err != nil {
						full, _ := e.FullLog(context.Background())
						t.Fatal(Describe(err), "\n", grep(full, appID, "AndroidRuntime", "GlutterJava"))
					}
					if strings.Contains(log, "panic") || strings.Contains(log, "fatal error") {
						t.Errorf("the log of the app has a panic:\n%s", log)
					}
				})
			}
		})
	}
}
