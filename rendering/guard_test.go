package rendering

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// TestGuardAllocatesNothingWithoutAPanic checks the recovery point of every layout, resize and paint
// (step 52.1): it makes no allocation when nothing panics, with or without a reporter. Before the
// step it built the context of the report ("during PerformLayout()") on each call. The context of a
// report is checked by TestErrorContainment.
func TestGuardAllocatesNothingWithoutAPanic(t *testing.T) {
	tests := []struct {
		name   string
		owner  bool
		report foundation.ErrorReporter
	}{
		{"no owner", false, nil},
		{"an owner without a reporter", true, nil},
		{"an owner with a reporter", true, func(error) {}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &RenderObjectBase{}
			if tt.owner {
				o := NewPipelineOwner()
				o.debug.ReportError = tt.report
				b.owner = o
			}
			ran := 0
			fn := func() { ran++ }
			for _, method := range []string{"PerformLayout", "PerformResize", "paint"} {
				if n := testing.AllocsPerRun(100, func() { b.guard(foundation.LayoutPanicked, method, fn) }); n != 0 {
					t.Errorf("guard of %s: %v allocations, want 0", method, n)
				}
			}
			if ran == 0 {
				t.Error("guard did not run the function")
			}
		})
	}
}
