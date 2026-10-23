package painting

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

func TestPaintingBindingReportError(t *testing.T) {
	failure := errors.New(tagImageLoadFailed, "no image")
	tests := []struct {
		name         string
		withDetails  bool
		withReporter bool
		wantDetails  int
		wantErrors   int
	}{
		{"details reporter wins", true, true, 1, 0},
		{"framework reporter", false, true, 0, 1},
		{"nobody listens", false, false, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var details []ImageErrorDetails
			var errs []error
			var o PaintingBindingOpts
			if tc.withDetails {
				o.ErrorReporter = func(d ImageErrorDetails) { details = append(details, d) }
			}
			if tc.withReporter {
				o.Reporter = func(err error) { errs = append(errs, err) }
			}
			b := NewPaintingBinding(context.Background(), scheduler.NewSchedulerBinding(ui.NewPlatformDispatcher(&fakeHost{})), o)
			b.ReportError(ImageErrorDetails{Err: failure, Context: "while testing"})
			if len(details) != tc.wantDetails || len(errs) != tc.wantErrors {
				t.Fatalf("details = %d, errors = %d; want %d, %d", len(details), len(errs), tc.wantDetails, tc.wantErrors)
			}
			if tc.wantErrors == 1 {
				appErr, ok := errors.FromError(errs[0])
				if !ok {
					t.Fatalf("expected *errors.Error, got %T", errs[0])
				}
				if !appErr.HasRoute("ImageLoadFailed") {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			}
		})
	}
}

// TestImageListenerPanicsAreReported checks the recovery point of the image
// stream listeners (SPEC §14.2): a panic in a listener is reported with the tag
// CallbackPanicked, and the other listeners still get the image.
func TestImageListenerPanicsAreReported(t *testing.T) {
	var errs []error
	pt := newPaintingTester(t, PaintingBindingOpts{Reporter: func(err error) { errs = append(errs, err) }})
	c := NewImageStreamCompleter(pt.PaintingBinding)
	got := 0
	c.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { panic("listener boom") }))
	c.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { got++ }))
	c.SetImage(NewImageInfo(testImage(2, 2)))
	if got != 1 {
		t.Errorf("the second listener was called %d times, want 1", got)
	}
	if len(errs) != 1 {
		t.Fatalf("%d errors reported, want 1: %v", len(errs), errs)
	}
	appErr, ok := errors.FromError(errs[0])
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", errs[0])
	}
	if !appErr.HasRoute("CallbackPanicked") || appErr.Metadata["context"] != "by an image listener" {
		t.Errorf("unexpected failure path: %s, context %v", appErr.Route(), appErr.Metadata["context"])
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagImageLoadFailed = "ImageLoadFailed"
)
