package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoDesktopTextSelectionControls are the controls of the text selection of macOS: there are no handles, and
// the context menu is not part of the controls (it is the context menu builder of the text field), so Flutter's
// CupertinoDesktopTextSelectionHandleControls is the same type.
//
// Deviation: the deprecated buildToolbar and handleSelectAll of the controls are not ported (SPEC §12.10).
//
// Flutter: cupertino/desktop_text_selection.dart CupertinoDesktopTextSelectionControls
type CupertinoDesktopTextSelectionControls struct{}

// CupertinoDesktopTextSelectionHandleControls are the controls of the macOS selection, to give to a text field, a
// SelectableRegion or an EditableText. It is read-only.
//
// Flutter: cupertino/desktop_text_selection.dart cupertinoDesktopTextSelectionHandleControls
var CupertinoDesktopTextSelectionHandleControls widgets.TextSelectionControls = CupertinoDesktopTextSelectionControls{}

// GetHandleSize is the zero size: there are no handles.
//
// Flutter: cupertino/desktop_text_selection.dart CupertinoDesktopTextSelectionControls.getHandleSize
func (CupertinoDesktopTextSelectionControls) GetHandleSize(float64) ui.Size { return ui.SizeZero }

// BuildHandle builds nothing.
//
// Flutter: cupertino/desktop_text_selection.dart CupertinoDesktopTextSelectionControls.buildHandle
func (CupertinoDesktopTextSelectionControls) BuildHandle(widgets.BuildContext, rendering.TextSelectionHandleType, float64, foundation.VoidCallback) widgets.Widget {
	return widgets.NewSizedBoxShrink(nil)
}

// GetHandleAnchor is the zero offset.
//
// Flutter: cupertino/desktop_text_selection.dart CupertinoDesktopTextSelectionControls.getHandleAnchor
func (CupertinoDesktopTextSelectionControls) GetHandleAnchor(rendering.TextSelectionHandleType, float64) ui.Offset {
	return ui.OffsetZero
}
