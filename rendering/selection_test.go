package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestSelectionUtils(t *testing.T) {
	rect := ui.NewRectFromLTWH(100, 100, 200, 500)
	points := []struct {
		name   string
		point  ui.Offset
		result rendering.SelectionResult
		ltr    ui.Offset
		rtl    ui.Offset
	}{
		{"outside top left", off(50, 50), rendering.SelectionResultPrevious, rect.TopLeft(), rect.TopRight()},
		{"outside left", off(50, 200), rendering.SelectionResultPrevious, rect.TopLeft(), rect.TopRight()},
		{"outside bottom left", off(50, 700), rendering.SelectionResultNext, rect.BottomRight(), rect.BottomLeft()},
		{"outside top", off(200, 50), rendering.SelectionResultPrevious, rect.TopLeft(), rect.TopRight()},
		{"outside top right", off(350, 50), rendering.SelectionResultPrevious, rect.TopLeft(), rect.TopRight()},
		{"outside right", off(350, 200), rendering.SelectionResultNext, rect.BottomRight(), rect.BottomLeft()},
		{"outside bottom right", off(350, 700), rendering.SelectionResultNext, rect.BottomRight(), rect.BottomLeft()},
		{"outside bottom", off(200, 700), rendering.SelectionResultNext, rect.BottomRight(), rect.BottomLeft()},
		{"center", off(150, 300), rendering.SelectionResultEnd, off(150, 300), off(150, 300)},
	}
	for _, tc := range points {
		t.Run(tc.name, func(t *testing.T) {
			if got := rendering.SelectionUtilsGetResultBasedOnRect(rect, tc.point); got != tc.result {
				t.Errorf("result = %v, want %v", got, tc.result)
			}
			if got := rendering.SelectionUtilsAdjustDragOffset(rect, tc.point, ui.TextDirectionLTR); got != tc.ltr {
				t.Errorf("ltr drag offset = %v, want %v", got, tc.ltr)
			}
			if got := rendering.SelectionUtilsAdjustDragOffset(rect, tc.point, ui.TextDirectionRTL); got != tc.rtl {
				t.Errorf("rtl drag offset = %v, want %v", got, tc.rtl)
			}
		})
	}
}

func TestSelectionGeometryEqual(t *testing.T) {
	a := rendering.SelectionGeometry{Status: rendering.SelectionStatusCollapsed, HasContent: true}
	tests := []struct {
		name  string
		other rendering.SelectionGeometry
		want  bool
	}{
		{"same", a, true},
		{"other status", rendering.SelectionGeometry{Status: rendering.SelectionStatusUncollapsed, HasContent: true}, false},
		{"other content", rendering.SelectionGeometry{Status: rendering.SelectionStatusCollapsed}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := a.Equal(tc.other); got != tc.want {
				t.Errorf("Equal = %v, want %v", got, tc.want)
			}
		})
	}
	if !a.HasSelection() || (rendering.SelectionGeometry{Status: rendering.SelectionStatusNone}).HasSelection() {
		t.Error("HasSelection is wrong")
	}
}

func TestSelectionEnumNames(t *testing.T) {
	tests := []struct{ got, want string }{
		{rendering.SelectionResultPrevious.String(), "SelectionResult.previous"},
		{rendering.SelectionEventTypeSelectAll.String(), "SelectionEventType.selectAll"},
		{rendering.SelectionStatusUncollapsed.String(), "SelectionStatus.uncollapsed"},
		{rendering.TextGranularityWord.String(), "TextGranularity.word"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}
