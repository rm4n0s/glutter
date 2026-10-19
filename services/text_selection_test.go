package services_test

import (
	"testing"

	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func TestTextSelection(t *testing.T) {
	tests := []struct {
		name             string
		sel              services.TextSelection
		start, end       int
		valid, collapsed bool
		wantRange        ui.TextRange
	}{
		{"forward", services.NewTextSelection(2, 5), 2, 5, true, false, ui.TextRange{Start: 2, End: 5}},
		{"backward", services.NewTextSelection(5, 2), 2, 5, true, false, ui.TextRange{Start: 2, End: 5}},
		{"collapsed", services.NewTextSelectionCollapsed(3), 3, 3, true, true, ui.TextRange{Start: 3, End: 3}},
		{"invalid", services.NewTextSelection(-1, -1), -1, -1, false, true, ui.TextRange{Start: -1, End: -1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.sel.Start() != tc.start || tc.sel.End() != tc.end || tc.sel.IsValid() != tc.valid ||
				tc.sel.IsCollapsed() != tc.collapsed || tc.sel.Range() != tc.wantRange {
				t.Errorf("selection = %+v", tc.sel)
			}
		})
	}
}
