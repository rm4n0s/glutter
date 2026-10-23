package androidtool

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
)

// TestDexIsUpToDate compiles the Java of the embedding and compares the result with the
// checked-in classes.dex (D22). It skips without javac or the SDK.
func TestDexIsUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("runs javac and d8")
	}
	if _, err := lookPath("javac"); err != nil {
		t.Skip("javac is not installed")
	}
	home, err := FindHome(os.Getenv)
	if err != nil {
		t.Skip("no Android SDK")
	}
	tools, err := FindBuildTools(home)
	if err != nil {
		t.Skip("no build-tools")
	}
	platform, err := FindPlatform(home)
	if err != nil {
		t.Skip("no platform")
	}
	out := filepath.Join(t.TempDir(), "classes.dex")
	sdk := &SDK{Home: home, BuildTools: tools, Platform: platform}
	if err := CompileDex(context.Background(), ExecRunner{}, sdk, "java", out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, Dex) {
		t.Error("classes.dex is stale: run go run ./cmd/glutter-gen android-dex")
	}
}

func TestCompileDexErrors(t *testing.T) {
	old := lookPath
	defer func() { lookPath = old }()
	lookPath = func(string) (string, error) { return "", os.ErrNotExist }
	err := CompileDex(context.Background(), &fakeRunner{}, testSDK(), "java", filepath.Join(t.TempDir(), "x.dex"))
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("CompileDex."+TagAndroidJavaNotFound) || appErr.Metadata["install"] == "" {
		t.Errorf("no javac: %v", err)
	}
	lookPath = func(string) (string, error) { return "javac", nil }
	err = CompileDex(context.Background(), &fakeRunner{}, testSDK(), t.TempDir(), filepath.Join(t.TempDir(), "x.dex"))
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("CompileDex."+TagAndroidWriteFailed) {
		t.Errorf("no sources: %v", err)
	}
	err = CompileDex(context.Background(), &fakeRunner{fail: "javac"}, testSDK(), "java", filepath.Join(t.TempDir(), "x.dex"))
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("CompileDex->fakeRunner.Run."+TagAndroidCommandFailed) {
		t.Errorf("javac fails: %v", err)
	}
}
