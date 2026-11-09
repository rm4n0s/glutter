//go:build darwin && !ios && glutter_display

package engine_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/mainthread"
	"github.com/rm4n0s/glutter/ui"
)

// These tests run engine.Run on the main thread of the test binary with a real window of an accessory app
// (SPEC §29.13): AppKit works only on the main thread, so TestMain gives it to mainthread.

func TestMain(m *testing.M) { mainthread.Main(m) }

// macEnv is the environment of the app of a test: an accessory app whose frames follow CVDisplayLink,
// which ticks whether or not anybody looks at the screen.
func macEnv(k string) string {
	switch k {
	case "GLUTTER_MACOS_ACCESSORY":
		return "1"
	case "GLUTTER_DISPLAYLINK":
		return "cv"
	}
	return os.Getenv(k)
}

func TestDisplayRunDrawsWithSoftAndEndsWhenCancelled(t *testing.T) {
	scene := gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 50, 30), gputest.FillPaint(0xFF3366CC))
	})
	var once sync.Once
	drew := make(chan struct{})
	var metrics []platform.ViewMetrics
	var mu sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-drew:
			time.Sleep(200 * time.Millisecond) // a few more frames
		case <-time.After(20 * time.Second):
			t.Error("no frame in 20 s")
		}
		cancel()
	}()
	var err error
	mainthread.Do(func() {
		err = engine.Run(ctx, engine.Config{
			Getenv: macEnv,
			Window: platform.WindowConfig{Title: "engine on macOS", Size: ui.Size{Width: 200, Height: 100}},
			Scene: func(ui.Size, float64) *ui.Scene {
				once.Do(func() { close(drew) })
				return scene
			},
			Hooks: engine.Hooks{Metrics: func(_ ui.ViewID, m platform.ViewMetrics) {
				mu.Lock()
				metrics = append(metrics, m)
				mu.Unlock()
			}},
		})
	})
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			t.Fatalf("Run: %v (route %s, metadata %v)", err, appErr.Route(), appErr.Metadata)
		}
		t.Fatalf("Run: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(metrics) == 0 {
		t.Fatal("the engine heard no metrics")
	}
	if got := metrics[0].PhysicalSize; got.Width < 200 || got.Height < 100 {
		t.Errorf("physical size = %v, want at least 200 x 100", got)
	}
}

func TestDisplayRendererNames(t *testing.T) {
	tests := []struct {
		name     string
		renderer string
		wantTag  string
	}{
		{"metal is not there yet", "metal", "RendererUnavailable"},
		{"gles is never on macOS", "gles", "RendererUnavailable"},
		{"vulkan is never on macOS", "vulkan", "RendererUnavailable"},
		{"a name that is no renderer", "bogus", "RendererUnknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			mainthread.Do(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err = engine.Run(ctx, engine.Config{Getenv: macEnv, Renderer: tt.renderer,
					Scene: func(ui.Size, float64) *ui.Scene { return gputest.CanvasScene(func(*ui.Canvas) {}) }})
			})
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.wantTag {
				t.Errorf("err = %v, want %s", err, tt.wantTag)
			}
		})
	}
}
