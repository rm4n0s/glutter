package widgets

import (
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// DesktopTextSelectionToolbarLayoutDelegate positions the text selection toolbar
// on desktop platforms: its top left corner is at the anchor, moved back as far
// as it takes for the toolbar to fit in the box.
//
// Flutter: widgets/desktop_text_selection_toolbar_layout_delegate.dart DesktopTextSelectionToolbarLayoutDelegate
type DesktopTextSelectionToolbarLayoutDelegate struct {
	rendering.SingleChildLayoutDelegateBase
	anchor ui.Offset
}

// NewDesktopTextSelectionToolbarLayoutDelegate creates the delegate of a
// toolbar that is anchored to a point, given in the coordinates of the box that
// the delegate lays out.
//
// Flutter: widgets/desktop_text_selection_toolbar_layout_delegate.dart DesktopTextSelectionToolbarLayoutDelegate.new
func NewDesktopTextSelectionToolbarLayoutDelegate(anchor ui.Offset) *DesktopTextSelectionToolbarLayoutDelegate {
	return &DesktopTextSelectionToolbarLayoutDelegate{SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil), anchor: anchor}
}

// Anchor is the point that the toolbar is placed at.
//
// Flutter: widgets/desktop_text_selection_toolbar_layout_delegate.dart DesktopTextSelectionToolbarLayoutDelegate.anchor
func (d *DesktopTextSelectionToolbarLayoutDelegate) Anchor() ui.Offset { return d.anchor }

// GetConstraintsForChild loosens the constraints.
//
// Flutter: widgets/desktop_text_selection_toolbar_layout_delegate.dart DesktopTextSelectionToolbarLayoutDelegate.getConstraintsForChild
func (d *DesktopTextSelectionToolbarLayoutDelegate) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return constraints.Loosen()
}

// GetPositionForChild puts the child at the anchor, or as close to it as keeps
// it inside the box.
//
// Flutter: widgets/desktop_text_selection_toolbar_layout_delegate.dart DesktopTextSelectionToolbarLayoutDelegate.getPositionForChild
func (d *DesktopTextSelectionToolbarLayoutDelegate) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	overhang := ui.Offset{
		Dx: d.anchor.Dx + childSize.Width - size.Width,
		Dy: d.anchor.Dy + childSize.Height - size.Height,
	}
	pos := d.anchor
	if overhang.Dx > 0 {
		pos.Dx = d.anchor.Dx - overhang.Dx
	}
	if overhang.Dy > 0 {
		pos.Dy = d.anchor.Dy - overhang.Dy
	}
	return pos
}

// ShouldRelayout is whether the anchor changed.
//
// Flutter: widgets/desktop_text_selection_toolbar_layout_delegate.dart DesktopTextSelectionToolbarLayoutDelegate.shouldRelayout
func (d *DesktopTextSelectionToolbarLayoutDelegate) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	return d.anchor != old.(*DesktopTextSelectionToolbarLayoutDelegate).anchor
}
