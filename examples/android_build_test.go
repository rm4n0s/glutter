//go:build !js

package examples

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
)

// TestEveryAndroidExampleBuilds checks what step 26.5 asks: every example except multiwindow and tray builds as
// an APK for android/arm64 with the NDK (SPEC §19, §24). It is skipped by -short and when the machine has no
// Android SDK and NDK, because the first build compiles the framework with cgo for Android.
func TestEveryAndroidExampleBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("builds every example for android/arm64; skipped by -short")
	}
	sdk, err := androidtool.FindSDK(os.Getenv)
	if err != nil {
		t.Skipf("there is no Android SDK and NDK to build with: %v", err)
	}
	// The examples that are not for the web are the ones that are not for Android: a phone has no second window and
	// no tray.
	names := webExamples(t)
	if len(names) < 9 {
		t.Fatalf("found %d examples for Android, want at least 9: %v", len(names), names)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			res, err := androidtool.Build(ctx, sdk, androidtool.ExecRunner{}, androidtool.BuildOptions{
				Package:   "./examples/" + name,
				Dir:       "..",
				Out:       filepath.Join(t.TempDir(), name+".apk"),
				ABIs:      []string{"arm64"},
				ConfigDir: filepath.Join("..", ".android-test", "debug-key"),
			})
			if err != nil {
				t.Fatalf("glutter-android build ./examples/%s: %v", name, err)
			}
			if res == nil {
				t.Fatal("the build gave no result")
			}
		})
	}
}
