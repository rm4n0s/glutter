package androidtest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

func TestAVDConfig(t *testing.T) {
	cfg := AVDs["glutter-37"].avdConfig()
	for _, want := range []string{
		"image.sysdir.1=system-images/android-37.2/google_apis_ps16k/x86_64/\n",
		"tag.id=google_apis_ps16k\n", "target=android-37.2\n", "hw.keyboard=no\n", "hw.sdCard=no\n", "showDeviceFrame=no\n",
		"hw.lcd.width=1080\n", "hw.lcd.height=2400\n", "AvdId=glutter-37\n"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config.ini has no %q", want)
		}
	}
	if a := AVDs["glutter-26"]; !strings.Contains(a.avdConfig(), "image.sysdir.1=system-images/android-26/google_apis/x86_64/\n") || a.tag() != "google_apis" {
		t.Errorf("glutter-26: %s", a.avdConfig())
	}
}

func TestEnsureAVD(t *testing.T) {
	sdk, root := t.TempDir(), t.TempDir()
	a := AVDs["glutter-37"]
	// The image is not installed.
	err := ensureAVD(sdk, root, a)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("ensureAVD."+TagAndroidTestImageMissing) || !strings.Contains(appErr.Metadata["install"].(string), "sdkmanager") {
		t.Fatalf("missing image: %v", err)
	}
	img := filepath.Join(sdk, a.sysdir())
	if err := os.MkdirAll(img, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(img, "system.img"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureAVD(sdk, root, a); err != nil {
		t.Fatal(err)
	}
	ini, err := os.ReadFile(filepath.Join(root, "avd", "glutter-37.ini"))
	if err != nil || !strings.Contains(string(ini), "path="+filepath.Join(root, "avd", "glutter-37.avd")) {
		t.Errorf("ini: %v %s", err, ini)
	}
	if _, err := os.Stat(filepath.Join(root, "avd", "glutter-37.avd", "config.ini")); err != nil {
		t.Error(err)
	}
	// The root is a file.
	blocker := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	err = ensureAVD(sdk, blocker, a)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("ensureAVD."+TagAndroidTestAVDFailed) {
		t.Errorf("unwritable root: %v", err)
	}
}

func TestStartErrors(t *testing.T) {
	t.Run("unknown virtual device", func(t *testing.T) {
		_, err := Start(context.Background(), Options{AVD: "nope", Root: t.TempDir()})
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Start."+TagAndroidTestBootFailed) {
			t.Errorf("got %v", err)
		}
	})
	t.Run("no SDK", func(t *testing.T) {
		t.Setenv("ANDROID_HOME", "")
		t.Setenv("ANDROID_SDK_ROOT", "")
		_, err := Start(context.Background(), Options{Root: t.TempDir()})
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Start->FindHome."+androidtool.TagAndroidSDKNotFound) {
			t.Errorf("got %v", err)
		}
	})
	t.Run("no emulator", func(t *testing.T) {
		t.Setenv("ANDROID_HOME", t.TempDir())
		_, err := Start(context.Background(), Options{Root: t.TempDir()})
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("Start."+TagAndroidTestEmulatorMissing) || appErr.Metadata["install"] == "" {
			t.Errorf("got %v", err)
		}
	})
	t.Run("no image", func(t *testing.T) {
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, "emulator"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, "emulator", "emulator"), nil, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("ANDROID_HOME", home)
		_, err := Start(context.Background(), Options{Root: t.TempDir()})
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Start->ensureAVD."+TagAndroidTestImageMissing) {
			t.Errorf("got %v", err)
		}
	})
}

func TestWaitBootReportsAnEmulatorThatExited(t *testing.T) {
	log := filepath.Join(t.TempDir(), "emu.log")
	if err := os.WriteFile(log, []byte("line1\nERROR | Could not start renderer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := &Emulator{AVD: AVDs["glutter-37"], done: make(chan struct{}), logPath: log}
	close(e.done)
	err := e.waitBoot(context.Background(), time.Second)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("Emulator.waitBoot."+TagAndroidTestBootFailed) || !strings.Contains(appErr.Metadata["log"].(string), "Could not start renderer") {
		t.Errorf("got %v", err)
	}
	// It never boots in time.
	e.done = make(chan struct{})
	e.ADB = androidtool.ADB{Run: scripted{out: "0\n"}}
	err = e.waitBoot(context.Background(), 10*time.Millisecond)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Emulator.waitBoot."+TagAndroidTestBootFailed) || !strings.Contains(appErr.Message, "in time") {
		t.Errorf("got %v", err)
	}
	// The context ends.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = e.waitBoot(ctx, time.Minute)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Emulator.waitBoot."+TagAndroidTestBootFailed) {
		t.Errorf("got %v", err)
	}
	e.ADB = androidtool.ADB{Run: scripted{out: "1\n"}}
	if err := e.waitBoot(context.Background(), time.Minute); err != nil {
		t.Errorf("booted: %v", err)
	}
}

// scripted is a Runner that prints the same output for every command.
type scripted struct{ out string }

func (s scripted) Run(context.Context, androidtool.Command) (string, error) { return s.out, nil }

func TestWaitTimeouts(t *testing.T) {
	e := &Emulator{ADB: androidtool.ADB{Run: scripted{out: "1234\n"}}}
	err := e.WaitExit(context.Background(), "org.x.y", 20*time.Millisecond)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Emulator.WaitExit."+TagAndroidTestTimeout) {
		t.Errorf("WaitExit: %v", err)
	}
	e = &Emulator{ADB: androidtool.ADB{Run: scripted{out: ""}}}
	if _, err := e.WaitPid(context.Background(), "org.x.y", 20*time.Millisecond); err == nil {
		t.Error("WaitPid found a process")
	} else if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Emulator.WaitPid."+TagAndroidTestTimeout) {
		t.Errorf("WaitPid: %v", err)
	}
	_, err = e.WaitLog(context.Background(), "org.x.y", "never", 20*time.Millisecond)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Emulator.WaitLog."+TagAndroidTestTimeout) {
		t.Errorf("WaitLog: %v", err)
	}
	if err := e.WaitExit(context.Background(), "org.x.y", time.Second); err != nil {
		t.Errorf("WaitExit with no process: %v", err)
	}
}

func TestHarnessEnv(t *testing.T) {
	t.Setenv("DISPLAY", ":0")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("ANDROID_SERIAL", "x")
	env := strings.Join(harnessEnv("/r", "/sdk", 5555), "\n")
	for _, want := range []string{"HOME=/r/home", "ANDROID_USER_HOME=/r/user", "ANDROID_AVD_HOME=/r/avd", "ANDROID_SDK_ROOT=/sdk", "ANDROID_ADB_SERVER_PORT=5555"} {
		if !strings.Contains(env, want) {
			t.Errorf("no %s in\n%s", want, env)
		}
	}
	for _, bad := range []string{"DISPLAY=", "ANDROID_SERIAL="} {
		if strings.Contains(env, bad+":0") || strings.Contains(env, bad+"wayland") || strings.Contains(env, "\n"+bad+"x") {
			t.Errorf("the environment has %s", bad)
		}
	}
}

func TestPorts(t *testing.T) {
	p, err := freePort()
	if err != nil || p <= 0 {
		t.Fatalf("freePort %d %v", p, err)
	}
	c, err := freeConsolePort()
	if err != nil {
		t.Skip("every console port is in use on this machine")
	}
	if c < 5554 || c > 5584 || c%2 != 0 {
		t.Errorf("console port %d", c)
	}
}
