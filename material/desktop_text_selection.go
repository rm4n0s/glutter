// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// DesktopTextSelectionControls are the selection controls of the desktop
// platforms: there are no selection handles, and the context menu is not a part
// of the controls (it is the contextMenuBuilder of the text field), so Flutter's
// DesktopTextSelectionHandleControls is the same type.
//
// Deviation: the deprecated buildToolbar, canSelectAll and handleSelectAll of
// the controls are not ported (SPEC §12.10).
//
// Flutter: material/desktop_text_selection.dart DesktopTextSelectionControls
type DesktopTextSelectionControls struct{}

// DesktopTextSelectionHandleControls is the controls of the desktop platforms,
// to give to a text field, a SelectableRegion or an EditableText. It is
// read-only.
//
// Flutter: material/desktop_text_selection.dart desktopTextSelectionHandleControls
var DesktopTextSelectionHandleControls widgets.TextSelectionControls = DesktopTextSelectionControls{}

// GetHandleSize is the zero size: there are no handles.
//
// Flutter: material/desktop_text_selection.dart DesktopTextSelectionControls.getHandleSize
func (DesktopTextSelectionControls) GetHandleSize(float64) ui.Size { return ui.SizeZero }

// BuildHandle builds nothing.
//
// Flutter: material/desktop_text_selection.dart DesktopTextSelectionControls.buildHandle
func (DesktopTextSelectionControls) BuildHandle(widgets.BuildContext, rendering.TextSelectionHandleType, float64, foundation.VoidCallback) widgets.Widget {
	return widgets.NewSizedBoxShrink(nil)
}

// GetHandleAnchor is the zero offset.
//
// Flutter: material/desktop_text_selection.dart DesktopTextSelectionControls.getHandleAnchor
func (DesktopTextSelectionControls) GetHandleAnchor(rendering.TextSelectionHandleType, float64) ui.Offset {
	return ui.OffsetZero
}
