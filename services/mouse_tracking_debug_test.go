//go:build !glutter_release && !glutter_profile

package services

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
)

// Diagnostics output exists in debug builds only.

func TestMouseTrackerAnnotationDiagnostics(t *testing.T) {
	a := NewMouseTrackerAnnotation(MouseTrackerAnnotationOpts{OnExit: func(*gestures.PointerExitEvent) {}, Cursor: MouseCursorUncontrolled})
	got := foundation.ToDiagnosticsNode(a).ToStringDeep()
	for _, want := range []string{"callbacks: exit", "cursor: uncontrolled"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q does not contain %q", got, want)
		}
	}
	if none := foundation.ToDiagnosticsNode(NewMouseTrackerAnnotation()).ToStringDeep(); !strings.Contains(none, "<none>") || strings.Contains(none, "cursor") {
		t.Errorf("default annotation prints %q", none)
	}
}
