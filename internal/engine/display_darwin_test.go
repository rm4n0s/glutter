//go:build darwin && !ios && glutter_display

package engine_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/mainthread"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
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

// syncBuffer is a log destination that the engine and the test both use.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestDisplayRunDrawsWithSoftAndEndsWhenCancelled(t *testing.T) {
	runDrawsUntilCancelled(t, "soft", "soft")
}

// TestDisplayAutoChoosesMetal runs the engine with the renderer on auto: on a Mac that has a Metal device the
// first candidate works, and the log names the device (SPEC §29.5).
func TestDisplayAutoChoosesMetal(t *testing.T) {
	log := runDrawsUntilCancelled(t, "", "metal: ")
	if !strings.Contains(log, "renderer chosen") {
		t.Errorf("the log does not say which renderer was chosen:\n%s", log)
	}
}

// TestDisplayMetalByName runs the engine with the renderer asked for by name.
func TestDisplayMetalByName(t *testing.T) {
	runDrawsUntilCancelled(t, "metal", "metal: ")
}

// runDrawsUntilCancelled runs the engine with a window and the renderer, until a frame was drawn and a few more,
// and checks that the log names the renderer that was chosen (want is a part of it). It returns the log.
func runDrawsUntilCancelled(t *testing.T, renderer, want string) string {
	t.Helper()
	logs := &syncBuffer{}
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
			Getenv:   macEnv,
			Renderer: renderer,
			Logger:   slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelInfo})),
			Window:   platform.WindowConfig{Title: "engine on macOS", Size: ui.Size{Width: 200, Height: 100}},
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
	if !strings.Contains(logs.String(), want) {
		t.Errorf("the log does not name the renderer (%q):\n%s", want, logs.String())
	}
	return logs.String()
}

func TestDisplayRendererNames(t *testing.T) {
	tests := []struct {
		name     string
		renderer string
		wantTag  string
	}{
		{"gles is never on macOS", "gles", "RendererUnavailable"},
		{"vulkan is never on macOS", "vulkan", "RendererUnavailable"},
		{"d3d12 is not on macOS", "d3d12", "RendererUnavailable"},
		{"webgpu is not on macOS", "webgpu", "RendererUnavailable"},
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

// cursorNames are the NSCursor class methods that the test looks for in the current cursor of the app.
var cursorNames = []string{"arrowCursor", "IBeamCursor", "pointingHandCursor"}

// currentCursorName says which of cursorNames the current cursor of the app is, or "".
func currentCursorName() string {
	cur := objc.ID(objcx.MustClass("NSCursor")).Send(objcx.Sel("currentCursor"))
	for _, n := range cursorNames {
		if objc.ID(objcx.MustClass("NSCursor")).Send(objcx.Sel(n)) == cur {
			return n
		}
	}
	return ""
}

// TestDisplayCursorOverATextFieldAndAButton runs a Material app on macOS, moves the pointer with the dev events
// (real NSEvents) over a text field, then over a tappable area that asks for the click cursor, then off both, and watches the cursor of the app:
// the I-beam, the pointing hand, the arrow (SPEC §29.6, §29.13). The cursor is read from the goroutine of the
// test, which only looks at a pointer.
func TestDisplayCursorOverATextFieldAndAButton(t *testing.T) {
	t.Setenv("GLUTTER_MACOS_ACCESSORY", "1")
	t.Setenv("GLUTTER_DISPLAYLINK", "cv")
	// The column of the app is 56 high (the field), 24 (a gap) and 40 (the button) and is centered in a
	// window of 400 x 300 points: the field is at y 118, the button at y 190.
	t.Setenv("GLUTTER_DEV_EVENTS", "wait:500ms;move:200,118;wait:500ms;move:200,190;wait:500ms;move:20,20;wait:500ms")
	app := material.NewMaterialApp(material.MaterialAppOpts{Home: material.NewScaffold(material.ScaffoldOpts{
		Body: material.NewCenter(material.NewColumn([]widgets.Widget{
			material.NewSizedBox(material.NewTextField(), widgets.SizedBoxOpts{Width: foundation.Some(200.0)}),
			material.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(24.0)}),
			// Material's buttons and InkWells show the arrow on a desktop, as in Flutter, so the area asks for the
			// pointing hand itself.
			material.NewSizedBox(material.NewInkWell(material.NewText("Go"), material.InkWellOpts{OnTap: func() {}, MouseCursor: services.SystemMouseCursors.Click}), widgets.SizedBoxOpts{Height: foundation.Some(40.0)}),
		}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})),
	})})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var seen []string
	var mu sync.Mutex
	go func() {
		defer cancel()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
			name := currentCursorName()
			mu.Lock()
			if name != "" && (len(seen) == 0 || seen[len(seen)-1] != name) {
				seen = append(seen, name)
			}
			done := len(seen) >= 3 && seen[len(seen)-1] == "arrowCursor" && slices.Contains(seen, "pointingHandCursor")
			mu.Unlock()
			if done {
				return
			}
		}
	}()
	var err error
	mainthread.Do(func() {
		err = material.RunAppContext(ctx, app, widgets.RunAppOpts{Window: widgets.WindowConfig{Title: "cursors", Size: ui.Size{Width: 400, Height: 300}}})
	})
	if err != nil {
		t.Fatalf("RunApp: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	iBeam, hand := slices.Index(seen, "IBeamCursor"), slices.Index(seen, "pointingHandCursor")
	if iBeam < 0 || hand < 0 || iBeam > hand {
		t.Errorf("the cursors the app showed, in order: %v; want the I-beam over the field and then the pointing hand over the button", seen)
	}
	if len(seen) == 0 || seen[len(seen)-1] != "arrowCursor" {
		t.Errorf("the cursors the app showed, in order: %v; want the arrow at the end", seen)
	}
}
