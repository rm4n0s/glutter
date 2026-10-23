// Command android-hello is a development example (step 25.2): it shows that a Glutter program
// starts on Android. It logs a line, the GLUTTER_ variables it got and the directories of the
// app, and then ends, so that the activity finishes. Nothing is drawn.
//
// Run it on a device or the emulator with: go run ./cmd/glutter-android run ./examples/dev/android-hello
//
// Environment (the intent extras of `glutter-android run -env K=V`): every GLUTTER_ variable is
// logged. In debug builds GLUTTER_DEV_EXIT=<n> ends the program with os.Exit(n) instead of
// returning from main, and GLUTTER_DEV_STAY=1 keeps it running until the activity is closed.
package main

import (
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	_ "github.com/rm4n0s/glutter/internal/platform/android" // the entry points of the Java embedding
)

func main() {
	slog.Info("android-hello: hello from Glutter", "target", foundation.DefaultTargetPlatform().String())
	slog.Warn("android-hello: a warning line")
	slog.Error("android-hello: an error line")
	var vars []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GLUTTER_") {
			vars = append(vars, kv)
		}
	}
	sort.Strings(vars)
	for _, kv := range vars {
		slog.Info("android-hello: variable", "var", kv)
	}
	cfg, cfgErr := os.UserConfigDir()
	cache, cacheErr := os.UserCacheDir()
	slog.Info("android-hello: directories", "HOME", os.Getenv("HOME"), "TMPDIR", os.TempDir(),
		"config", cfg, "configErr", cfgErr, "cache", cache, "cacheErr", cacheErr,
		"XDG_DATA_HOME", os.Getenv("XDG_DATA_HOME"), "pid", os.Getpid())
	if foundation.DebugMode {
		if os.Getenv("GLUTTER_DEV_STAY") == "1" {
			select {}
		}
		if v := os.Getenv("GLUTTER_DEV_EXIT"); v != "" {
			code, err := strconv.Atoi(v)
			if err != nil {
				slog.Error("android-hello: GLUTTER_DEV_EXIT is not a number", "value", v)
				code = 2
			}
			slog.Info("android-hello: exiting", "code", code)
			time.Sleep(200 * time.Millisecond) // let the log lines reach logcat
			os.Exit(code)
		}
	}
	slog.Info("android-hello: main returns")
	time.Sleep(200 * time.Millisecond)
}
