package rendering_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// faultyBox is a box whose layout, resize and paint are scripted by the test.
type faultyBox struct {
	rendering.RenderBox
	sizedByParent bool
	onLayout      func(b *faultyBox)
	onResize      func(b *faultyBox)
	onPaint       func(b *faultyBox)
}

func newFaulty() *faultyBox {
	b := &faultyBox{}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *faultyBox) SizedByParent() bool { return b.sizedByParent }
func (b *faultyBox) PerformLayout() {
	if b.onLayout != nil {
		b.onLayout(b)
	} else if !b.sizedByParent {
		b.SetSize(b.BoxConstraints().Smallest())
	}
}
func (b *faultyBox) PerformResize() {
	if b.onResize != nil {
		b.onResize(b)
		return
	}
	b.RenderBox.PerformResize()
}
func (b *faultyBox) Paint(*rendering.PaintingContext, ui.Offset) {
	if b.onPaint != nil {
		b.onPaint(b)
	}
}

func collectingHarness(t *testing.T) *renderingtest.Harness {
	t.Helper()
	return renderingtest.New(t, renderingtest.Opts{Size: ui.Size{Width: 800, Height: 600}, CollectPanics: true})
}

// TestRenderPanicsAreReported checks the recovery points of SPEC §14.2 in the
// render tree: a panic in PerformResize, PerformLayout or Paint is reported with
// its tag, the siblings of the render object are still laid out and painted, and
// the frame finishes.
func TestRenderPanicsAreReported(t *testing.T) {
	tests := []struct {
		name  string
		setup func(bad *faultyBox)
		tag   string
		ctx   string
	}{
		{"PerformLayout", func(b *faultyBox) {
			b.onLayout = func(b *faultyBox) { b.SetSize(ui.Size{Width: 5, Height: 5}); panic("layout boom") }
		}, foundation.LayoutPanicked, "during PerformLayout()"},
		{"PerformResize", func(b *faultyBox) {
			b.sizedByParent = true
			b.onResize = func(b *faultyBox) { b.SetSize(ui.Size{Width: 5, Height: 5}); panic("resize boom") }
			b.onLayout = func(*faultyBox) {}
		}, foundation.LayoutPanicked, "during PerformResize()"},
		{"Paint", func(b *faultyBox) {
			b.onPaint = func(*faultyBox) { panic("paint boom") }
		},
			foundation.PaintPanicked, "during paint()"},
		{"tagged panic keeps its tag", func(b *faultyBox) {
			b.onLayout = func(b *faultyBox) {
				b.SetSize(ui.Size{Width: 5, Height: 5})
				panic(errors.New(tagMyOwnFailure, "tagged"))
			}
		}, "MyOwnFailure", "during PerformLayout()"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bad := newFaulty()
			tc.setup(bad)
			sibling := newSizedBox(30, 20)
			flex := rendering.NewRenderFlex([]rendering.Box{bad, sibling},
				rendering.RenderFlexOpts{Direction: painting.AxisHorizontal, TextDirection: ui.TextDirectionLTR})
			h := collectingHarness(t)
			layout(t, h, flex, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 200, Height: 100})), phasePaint))
			if len(h.Errors) == 0 {
				t.Fatal("no error was reported")
			}
			appErr, ok := errors.FromError(h.Errors[0])
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", h.Errors[0])
			}
			if !appErr.HasRoute(tc.tag) || appErr.Tag != tc.tag {
				t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tc.tag)
			}
			if appErr.Metadata["context"] != tc.ctx {
				t.Errorf("context = %v, want %q", appErr.Metadata["context"], tc.ctx)
			}
			if _, has := appErr.Metadata["stack"]; !has || appErr.Metadata["object"] == nil {
				t.Errorf("metadata = %v, want stack and object", appErr.Metadata)
			}
			if !foundation.IsRecoveredPanic(h.Errors[0]) {
				t.Error("IsRecoveredPanic is false for a recovered panic")
			}
			if flex.Size() != (ui.Size{Width: 200, Height: 100}) {
				t.Errorf("flex size = %v: the frame did not go on after the panic", flex.Size())
			}
			if sibling.Size() != (ui.Size{Width: 30, Height: 20}) {
				t.Errorf("sibling size = %v, want 30x20", sibling.Size())
			}
		})
	}
}

// TestRenderPanicsPropagateWithoutReporter checks that a panic is not swallowed
// when nobody can receive the report.
func TestRenderPanicsPropagateWithoutReporter(t *testing.T) {
	bad := newFaulty()
	bad.onLayout = func(*faultyBox) { panic(errors.New(tagNobodyListens, "x")) }
	requireTag(t, "NobodyListens", func() { layout(t, newHarness(t), bad) })
}

// TestMarkNeedsLayoutFromAnotherGoroutine checks the WrongThread route of the
// render tree (SPEC §13): debug builds report a layout request from a goroutine
// that is not the UI thread.
func TestMarkNeedsLayoutFromAnotherGoroutine(t *testing.T) {
	skipUnlessDebug(t)
	dbg := rendering.NewDebugOptions()
	dbg.UIThread = foundation.NewUIThread()
	h := renderingtest.New(t, renderingtest.Opts{Debug: dbg})
	box := newSizedBox(10, 10)
	layout(t, h, box, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 50, Height: 50})))
	got := make(chan any)
	go func() { got <- panicOf(box.MarkNeedsLayout) }()
	v := <-got
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("WrongThread") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	box.MarkNeedsLayout() // the UI thread is fine
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMyOwnFailure  = "MyOwnFailure"
	tagNobodyListens = "NobodyListens"
)
