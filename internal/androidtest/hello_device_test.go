//go:build glutter_android

package androidtest

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
)

func TestMain(m *testing.M) { os.Exit(Run(m)) }

// TestAndroidHello runs examples/dev/android-hello on the API 37 devices (step 25.2): it logs its
// line, sees an intent extra, has HOME in its files directory, exits with code 3 and its
// activity is gone; and when main returns, the activity finishes and the process ends.
func TestAndroidHello(t *testing.T) {
	for _, avd := range []string{"glutter-37", "glutter-37-16k"} {
		t.Run(avd, func(t *testing.T) {
			e := Shared(t, avd)
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			apk, appID, err := e.BuildAPK(ctx, "./examples/dev/android-hello", t.TempDir(), androidtool.BuildOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if err := e.Install(ctx, apk); err != nil {
				t.Fatal(err)
			}

			t.Run("os.Exit", func(t *testing.T) {
				extras := map[string]string{"GLUTTER_TEST_EXTRA": "abc def", "GLUTTER_DEV_EXIT": "3"}
				if err := e.Launch(ctx, appID, extras); err != nil {
					t.Fatal(err)
				}
				log, err := e.WaitLog(ctx, appID, "exiting", time.Minute)
				if err != nil {
					t.Fatalf("%v\nfull log:\n%s", err, mustFullLog(ctx, e))
				}
				// The priority of a logcat line comes from the slog level of the line (SPEC §24.9).
				for _, want := range []string{"I/" + appID, "W/" + appID, "E/" + appID, "hello from Glutter", `var="GLUTTER_TEST_EXTRA=abc def"`, "HOME=/data/", "/files", "code=3"} {
					if !strings.Contains(log, want) {
						t.Errorf("the log has no %q:\n%s", want, log)
					}
				}
				if err := e.WaitExit(ctx, appID, 20*time.Second); err != nil {
					t.Fatal(err)
				}
				waitActivityGone(ctx, t, e, appID)
			})

			t.Run("main returns", func(t *testing.T) {
				if err := e.Launch(ctx, appID, nil); err != nil {
					t.Fatal(err)
				}
				if _, err := e.WaitLog(ctx, appID, "main returns", time.Minute); err != nil {
					t.Fatalf("%v\nfull log:\n%s", err, mustFullLog(ctx, e))
				}
				if err := e.WaitExit(ctx, appID, 20*time.Second); err != nil {
					t.Fatal(err)
				}
				waitActivityGone(ctx, t, e, appID)
			})
		})
	}
}

// waitActivityGone waits up to 10 seconds for the system to drop the activity of an app whose
// process ended, and checks that the process does not come back.
func waitActivityGone(ctx context.Context, t *testing.T, e *Emulator, appID string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		has, err := e.HasActivity(ctx, appID)
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the activity of %s is still there 10 seconds after its process ended", appID)
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	if pid, _ := e.ADB.Pid(ctx, appID); pid != 0 {
		t.Errorf("the app runs again (pid %d) after it ended", pid)
	}
}

func mustFullLog(ctx context.Context, e *Emulator) string {
	s, err := e.FullLog(ctx)
	if err != nil {
		return err.Error()
	}
	if len(s) > 6000 {
		s = s[len(s)-6000:]
	}
	return s
}

// TestLibrariesAre16KBAligned builds android-hello for both ABIs and checks with llvm-readelf that
// the LOAD segments of libapp.so are aligned to 16 KB, and with zipalign that the APK is (SPEC
// §24.1).
func TestLibrariesAre16KBAligned(t *testing.T) {
	sdk, err := androidtool.FindSDK(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	apk := filepath.Join(dir, "hello.apk")
	_, err = androidtool.Build(context.Background(), sdk, androidtool.ExecRunner{}, androidtool.BuildOptions{
		Package: "./examples/dev/android-hello", Dir: mod, Out: apk, ConfigDir: filepath.Join(mod, ".android-test", "debug-key")})
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(sdk.Tool("zipalign"), "-c", "-P", "16", "4", apk).CombinedOutput(); err != nil {
		t.Errorf("zipalign -c -P 16: %v\n%s", err, out)
	}
	zr, err := zip.OpenReader(apk)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	readelf := filepath.Join(sdk.NDK, "toolchains", "llvm", "prebuilt", "linux-x86_64", "bin", "llvm-readelf")
	found := 0
	for _, f := range zr.File {
		if !strings.HasSuffix(f.Name, "/libapp.so") {
			continue
		}
		found++
		if f.Method != zip.Store {
			t.Errorf("%s is compressed", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		so := filepath.Join(dir, strings.ReplaceAll(f.Name, "/", "_"))
		out, err := os.Create(so)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			t.Fatal(err)
		}
		rc.Close()
		out.Close()
		res, err := exec.Command(readelf, "-l", so).CombinedOutput()
		if err != nil {
			t.Fatalf("llvm-readelf: %v\n%s", err, res)
		}
		loads := 0
		for _, line := range strings.Split(string(res), "\n") {
			if fields := strings.Fields(line); len(fields) > 0 && fields[0] == "LOAD" {
				loads++
				if align := fields[len(fields)-1]; align != "0x4000" && align != "0x10000" {
					t.Errorf("%s: a LOAD segment is aligned to %s, want 0x4000: %s", f.Name, align, line)
				}
			}
		}
		if loads == 0 {
			t.Errorf("%s: no LOAD segment in\n%s", f.Name, res)
		}
	}
	if found != 2 {
		t.Errorf("the APK has %d libapp.so, want 2 (arm64-v8a and x86_64)", found)
	}
}
