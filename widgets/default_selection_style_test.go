package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests cover DefaultSelectionStyle (widgets/default_selection_style.dart)
// and its use by Text.

func TestDefaultSelectionStyleOfFallsBackWithoutAnAncestor(t *testing.T) {
	ts := gluttertest.New(t)
	var got *DefaultSelectionStyle
	pumpContainer(ts, NewBuilder(func(bc BuildContext) Widget { got = DefaultSelectionStyleOf(bc); return NewSizedBox(nil) }))
	if got.CursorColor().IsSome() || got.SelectionColor().IsSome() || got.MouseCursor() != nil {
		t.Fatalf("the fallback has values: %v %v %v", got.CursorColor(), got.SelectionColor(), got.MouseCursor())
	}
}

func TestDefaultSelectionStyleMergeKeepsTheParentValues(t *testing.T) {
	ts := gluttertest.New(t)
	var got *DefaultSelectionStyle
	parentCursor, childSelection := ui.NewColor(0xFF112233), ui.NewColor(0xFF445566)
	pumpContainer(ts, NewDefaultSelectionStyle(
		NewDefaultSelectionStyleMerge(
			NewBuilder(func(bc BuildContext) Widget { got = DefaultSelectionStyleOf(bc); return NewSizedBox(nil) }),
			DefaultSelectionStyleOpts{SelectionColor: foundation.Some(childSelection)}),
		DefaultSelectionStyleOpts{CursorColor: foundation.Some(parentCursor), SelectionColor: foundation.Some(ui.NewColor(0xFF000000)),
			MouseCursor: services.SystemMouseCursors.Click}))
	if c, _ := got.CursorColor().Get(); c != parentCursor {
		t.Errorf("cursor color = %v, want the parent's %v", c, parentCursor)
	}
	if c, _ := got.SelectionColor().Get(); c != childSelection {
		t.Errorf("selection color = %v, want the merged %v", c, childSelection)
	}
	if got.MouseCursor() != services.SystemMouseCursors.Click {
		t.Errorf("mouse cursor = %v, want the parent's", got.MouseCursor())
	}
}

func TestDefaultSelectionStyleNotifiesDependentsWhenItChanges(t *testing.T) {
	ts := gluttertest.New(t)
	builds := 0
	child := NewBuilder(func(bc BuildContext) Widget { builds++; DefaultSelectionStyleOf(bc); return NewSizedBox(nil) })
	opts := func(c ui.Color) DefaultSelectionStyleOpts {
		return DefaultSelectionStyleOpts{SelectionColor: foundation.Some(c)}
	}
	pumpContainer(ts, NewDefaultSelectionStyle(child, opts(ui.NewColor(0xFF000000))))
	pumpContainer(ts, NewDefaultSelectionStyle(child, opts(ui.NewColor(0xFF000000))))
	if builds != 1 {
		t.Fatalf("builds = %d after an equal style, want 1", builds)
	}
	pumpContainer(ts, NewDefaultSelectionStyle(child, opts(ui.NewColor(0xFFFFFFFF))))
	if builds != 2 {
		t.Fatalf("builds = %d after a different style, want 2", builds)
	}
}

func TestTextSelectionColor(t *testing.T) {
	explicit, inherited := ui.NewColor(0xFF010203), ui.NewColor(0xFF0A0B0C)
	tests := []struct {
		name string
		home Widget
		want ui.Color
	}{
		{"default", NewSelectableRegion(NewEmptyTextSelectionControls(), ltrText("x")), DefaultSelectionColor},
		{"inherited", NewDefaultSelectionStyle(NewSelectableRegion(NewEmptyTextSelectionControls(), ltrText("x")),
			DefaultSelectionStyleOpts{SelectionColor: foundation.Some(inherited)}), inherited},
		{"explicit", NewDefaultSelectionStyle(NewSelectableRegion(NewEmptyTextSelectionControls(),
			NewText("x", TextOpts{TextDirection: ui.TextDirectionLTR, SelectionColor: foundation.Some(explicit)})),
			DefaultSelectionStyleOpts{SelectionColor: foundation.Some(inherited)}), explicit},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpContainer(ts, tc.home)
			got, _ := paragraphOf(ts, "x").SelectionColor().Get()
			if got != tc.want {
				t.Fatalf("selection color = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextInASelectableRegionShowsTheTextCursor(t *testing.T) {
	ts := gluttertest.New(t)
	pumpContainer(ts, NewSelectableRegion(NewEmptyTextSelectionControls(), NewCenter(ltrText("x"))))
	region := ts.Widget(gluttertest.Descendant(gluttertest.ByText("x"), gluttertest.ByType[*MouseRegion]())).(*MouseRegion)
	if region.Cursor() != services.SystemMouseCursors.Text {
		t.Fatalf("cursor = %v, want the text cursor", region.Cursor())
	}
}
