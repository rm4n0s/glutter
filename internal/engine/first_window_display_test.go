//go:build linux && !android && glutter_display

package engine_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests run the whole framework on the engine in a headless compositor
// (scripts/headless-wayland.sh) or X server (scripts/headless-x11.sh): two windows, the
// first one destroyed while the second goes on, then the second one, which ends Run
// (step 27.5). Each run is a process of its own, as the renderer-choice tests are
// (Mesa's EGL does not like a second connection in one process).

const firstWindowChildEnv = "GLUTTER_FIRST_WINDOW_CHILD"

// TestDisplayFirstWindowChild is the app, started by runFirstWindowApp; it prints what
// happened. It does nothing in a normal run.
func TestDisplayFirstWindowChild(t *testing.T) {
	backend := os.Getenv(firstWindowChildEnv)
	if backend == "" {
		t.Skip("run by runFirstWindowApp")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var (
		first, second *widgets.RegularWindowController
		frames        int
		firstGone     = -1
		secondFrames  int
		secondGone    bool
		disposed      bool
	)
	cfg := engine.Config{
		Backend: backend,
		Window:  platform.WindowConfig{Title: "first", AppID: "dev.glutter.firstwindow"},
		Start: func(d *ui.PlatformDispatcher) error {
			d.SetTextEngine(renderingtest.BundledTextEngine())
			b := widgets.NewWidgetsBinding(ctx, d, widgets.WidgetsBindingOpts{
				ErrorReporter: func(err error) { fmt.Println("ERROR: " + err.Error()) },
			})
			owner := b.WindowingOwner()
			first, _ = owner.ImplicitWindowController()
			var err error
			second, err = owner.CreateRegularWindowController(ui.Size{Width: 300, Height: 200}, widgets.RegularWindowControllerOpts{Title: "second"})
			if err != nil {
				return err
			}
			b.AttachRootWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewViewCollection([]widgets.Widget{
				widgets.NewView(d.ImplicitView(), &probeWidget{StatefulWidget: widgets.NewStatefulWidgetBase(), disposed: &disposed}),
				widgets.NewRegularWindow(second, widgets.NewText("second")),
			})))
			prev := d.OnDrawFrame
			d.OnDrawFrame = func() {
				prev()
				frames++
				if firstGone >= 0 && !secondGone {
					secondFrames++
				}
				switch {
				case frames == 10:
					firstGone = frames
					b.Post(first.Destroy)
				case firstGone >= 0 && secondFrames == 10 && !secondGone:
					fmt.Printf("SECOND-DREW %d frames after the first was destroyed, states of the first disposed: %v\n", secondFrames, disposed)
					secondGone = true
					second.Destroy()
				}
			}
			// Something always changes: a frame is asked for every few milliseconds.
			go func() {
				for ctx.Err() == nil {
					time.Sleep(10 * time.Millisecond)
					b.Post(b.Scheduler().ScheduleFrame)
				}
			}()
			return nil
		},
	}
	if err := engine.Run(ctx, cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}
	fmt.Println("RUN-ENDED")
}

// runFirstWindowApp runs the child with the backend and returns what it printed.
func runFirstWindowApp(t *testing.T, backend string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run", "^TestDisplayFirstWindowChild$", "-test.count=1", "-test.v")
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), firstWindowChildEnv+"="+backend)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the app failed: %v\n%s", err, out)
	}
	return string(out)
}

func TestDisplayTheFirstWindowCanBeDestroyedWhileOthersGoOn(t *testing.T) {
	for _, backend := range []string{"wayland", "x11"} {
		t.Run(backend, func(t *testing.T) {
			if backend == "wayland" && os.Getenv("WAYLAND_DISPLAY") == "" {
				t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
			}
			if backend == "x11" && (os.Getenv("DISPLAY") == "" || os.Getenv("WAYLAND_DISPLAY") != "") {
				t.Skip("there is no headless X server (run under scripts/headless-x11.sh)")
			}
			out := runFirstWindowApp(t, backend)
			if strings.Contains(out, "ERROR:") {
				t.Errorf("the framework reported an error:\n%s", out)
			}
			drew := strings.Index(out, "SECOND-DREW 10 frames after the first was destroyed, states of the first disposed: true")
			ended := strings.Index(out, "RUN-ENDED")
			if drew < 0 || ended < drew {
				t.Errorf("the second window did not draw after the first was destroyed, or Run did not end after the second:\n%s", out)
			}
		})
	}
}

// probeWidget is a stateful widget that tells when its state is disposed.
type probeWidget struct {
	widgets.StatefulWidget
	disposed *bool
}

func (p *probeWidget) CreateState() widgets.State { return &probeWidgetState{} }

type probeWidgetState struct{ widgets.StateOf[*probeWidget] }

func (s *probeWidgetState) Dispose() { *s.Widget().disposed = true }

func (s *probeWidgetState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewText("first")
}
