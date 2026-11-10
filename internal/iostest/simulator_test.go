//go:build glutter_ios

package iostest_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostest/hellolines"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// TestHelloAppStartsInTheSimulator builds the tiny app for the Simulator, installs it on an iPhone, launches it and reads the lines
// that it prints when UIKit has launched it and its scene has connected; nothing stays booted afterwards (SPEC §30.11).
func TestHelloAppStartsInTheSimulator(t *testing.T) {
	var h *iostest.Harness
	// Cleanups run last in first out: this one runs after the harness has shut the device down.
	t.Cleanup(func() {
		if h == nil {
			return
		}
		left, err := h.Leftovers(context.Background())
		if err != nil {
			t.Errorf("listing the booted devices: %v", err)
		}
		if len(left) != 0 {
			t.Errorf("devices are still booted: %v", left)
		}
	})
	h = iostest.New(t)
	root, err := iostest.ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := iostool.Build(ctx, iostool.BuildOptions{Package: "./internal/iostest/hello", Dir: root, Out: filepath.Join(t.TempDir(), "Hello.app"), Simulator: true, ID: "dev.glutter.hello"})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	dev, err := h.Device(ctx, "iphone")
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	if err := h.Simctl.Install(ctx, dev.UDID, res.Layout.App); err != nil {
		t.Fatalf("install: %v", err)
	}
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, hellolines.EnvName+"=arrived")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	line, err := app.WaitFor(hellolines.Launched, time.Minute)
	if err != nil {
		t.Fatalf("the launch line: %v", err)
	}
	if want := hellolines.Launched + " arrived"; line != want {
		t.Errorf("launch line = %q, want %q (the environment of the launch reaches the app)", line, want)
	}
	if _, err := app.WaitFor(hellolines.SceneConnected, time.Minute); err != nil {
		t.Errorf("the scene line: %v", err)
	}
}

// TestMainOnAnotherThreadFails runs the app with Main called from a goroutine: UIKit works on the main thread only, and Main says so with
// IOSNotMainThread, whose route names Main (the route test of that tag, which only iOS can run).
func TestMainOnAnotherThreadFails(t *testing.T) {
	h := iostest.New(t)
	root, err := iostest.ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := iostool.Build(ctx, iostool.BuildOptions{Package: "./internal/iostest/hello", Dir: root, Out: filepath.Join(t.TempDir(), "Hello.app"), Simulator: true, ID: "dev.glutter.hello"})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	dev, err := h.Device(ctx, "iphone")
	if err != nil {
		t.Fatalf("device: %v", err)
	}
	if err := h.Simctl.Install(ctx, dev.UDID, res.Layout.App); err != nil {
		t.Fatalf("install: %v", err)
	}
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, hellolines.ModeEnv+"=goroutine")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	line, err := app.WaitFor(hellolines.NotMainThread, time.Minute)
	if err != nil {
		t.Fatalf("the line: %v", err)
	}
	if want := hellolines.NotMainThread + " tag=IOSNotMainThread route=true"; line != want {
		t.Errorf("line = %q, want %q", line, want)
	}
}
