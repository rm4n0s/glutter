package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func resizeTo(w, h float64) fake.MetricsEvent {
	return fake.MetricsEvent{View: 0, Metrics: platform.ViewMetrics{PhysicalSize: ui.Size{Width: w, Height: h}, DevicePixelRatio: 1}}
}

// TestScrollableTakenOutAfterItsMetricsChanged runs the framework in the real
// loop: the first layout of a Scrollable posts DidUpdateScrollMetrics, and the
// width of the window drops under the size where the app takes the Scrollable
// out of the tree before the posted function runs. gluttertest runs what is
// posted inside Pump, in the same frame, so it cannot show it (step 27.1).
func TestScrollableTakenOutAfterItsMetricsChanged(t *testing.T) {
	tests := []struct {
		name   string
		widths []float64 // the width of the window at each frame
		// scrollable is whether the app still shows the Scrollable at the end.
		scrollable bool
	}{
		{"the width drops under the breakpoint", []float64{800, 200, 200}, false},
		{"the width drops and comes back", []float64{800, 200, 800}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := fake.New()
			p.Inject(resizeTo(tt.widths[0], 600))
			for i, w := range tt.widths {
				if i > 0 {
					p.Inject(resizeTo(w, 600))
				}
				p.Inject(fake.FrameEvent{View: 0, FrameTime: time.Duration(i+1) * time.Millisecond})
			}

			var errs []error
			var stop func()
			frames := 0
			shown := false
			cfg := engine.Config{
				Getenv: func(string) string { return "" },
				Hooks:  engine.Hooks{Window: func(_ platform.Window, s func()) { stop = s }},
				Start: func(d *ui.PlatformDispatcher) error {
					b := widgets.NewWidgetsBinding(context.Background(), d, widgets.WidgetsBindingOpts{
						ErrorReporter: func(err error) { errs = append(errs, err) },
					})
					app := widgets.NewLayoutBuilder(func(_ widgets.BuildContext, c rendering.BoxConstraints) widgets.Widget {
						shown = c.MaxWidth >= 400
						if !shown {
							return widgets.NewSizedBox(nil)
						}
						return widgets.NewScrollable(func(_ widgets.BuildContext, offset rendering.ViewportOffset) widgets.Widget {
							return widgets.NewViewport(offset, []widgets.Widget{
								widgets.NewSliverToBoxAdapter(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(5000.0)})),
							})
						})
					})
					b.AttachRootWidget(b.WrapWithDefaultView(widgets.NewDirectionality(ui.TextDirectionLTR, app)))
					prev := d.OnDrawFrame
					d.OnDrawFrame = func() {
						prev()
						frames++
						if frames == len(tt.widths) {
							// Posted after the function that the first frame posted.
							d.Post(func() { stop() })
						}
					}
					return nil
				},
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			type outcome struct {
				err      error
				panicked any
			}
			done := make(chan outcome, 1)
			go func() {
				var out outcome
				defer func() {
					out.panicked = recover()
					done <- out
				}()
				out.err = engine.RunOnFake(ctx, cfg, p)
			}()
			select {
			case out := <-done:
				if out.panicked != nil {
					t.Fatalf("the loop panicked: %v", out.panicked)
				}
				if out.err != nil {
					t.Fatal(out.err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Run did not return")
			}
			if frames != len(tt.widths) {
				t.Errorf("%d frames ran, want %d", frames, len(tt.widths))
			}
			if shown != tt.scrollable {
				t.Errorf("the Scrollable is shown at the end = %v, want %v", shown, tt.scrollable)
			}
			for _, err := range errs {
				t.Errorf("the framework reported: %v", err)
			}
		})
	}
}
