// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/widgets"
)

// SelectionAreaOpts are the optional parameters of NewSelectionArea.
//
// Flutter: material/selection_area.dart SelectionArea.new
type SelectionAreaOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FocusNode is the focus node of the area; the default is one that the area
	// creates.
	FocusNode *widgets.FocusNode
	// SelectionControls build the selection handles; the default is the controls
	// of the platform.
	SelectionControls widgets.TextSelectionControls
	// ContextMenuBuilder builds the context menu; the default is an
	// AdaptiveTextSelectionToolbar.
	ContextMenuBuilder widgets.SelectableRegionContextMenuBuilder
	// NoContextMenu shows no context menu (Flutter's contextMenuBuilder: null).
	NoContextMenu bool
	// OnSelectionChanged is called when the selected content changes.
	OnSelectionChanged func(content *rendering.SelectedContent)
}

// SelectionArea is a widget that covers its subtree with a SelectableRegion that
// has the Material look: the user can select the text of the Text widgets below
// it with the pointer, and copy it from the context menu.
//
// Deviation: the magnifier is not ported (SPEC §12.10).
//
// Flutter: material/selection_area.dart SelectionArea
type SelectionArea struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     SelectionAreaOpts
}

// NewSelectionArea creates an area that makes the widgets below it selectable.
//
// Flutter: material/selection_area.dart SelectionArea.new
func NewSelectionArea(child widgets.Widget, opts ...SelectionAreaOpts) *SelectionArea {
	o := firstOpts(opts)
	return &SelectionArea{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Child is the widget below the area in the tree.
//
// Flutter: material/selection_area.dart SelectionArea.child
func (w *SelectionArea) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: material/selection_area.dart SelectionArea.createState
func (w *SelectionArea) CreateState() widgets.State {
	return &SelectionAreaState{selectableRegionKey: widgets.NewGlobalKey[*widgets.SelectableRegionState]()}
}

// defaultSelectionAreaContextMenuBuilder builds the toolbar of the platform.
//
// Flutter: material/selection_area.dart SelectionArea._defaultContextMenuBuilder
func defaultSelectionAreaContextMenuBuilder(_ widgets.BuildContext, state *widgets.SelectableRegionState) widgets.Widget {
	return NewAdaptiveTextSelectionToolbarSelectableRegion(state)
}

// SelectionAreaState is the state of a SelectionArea.
//
// Flutter: material/selection_area.dart SelectionAreaState
type SelectionAreaState struct {
	widgets.StateOf[*SelectionArea]
	selectableRegionKey *widgets.GlobalKey[*widgets.SelectableRegionState]
}

// SelectableRegion is the state of the SelectableRegion below the area, which
// can select all, clear the selection and so on.
//
// Flutter: material/selection_area.dart SelectionAreaState.selectableRegion
func (s *SelectionAreaState) SelectableRegion() *widgets.SelectableRegionState {
	region, _ := s.selectableRegionKey.CurrentState()
	return region
}

// Build builds the SelectableRegion with the controls of the platform.
//
// Flutter: material/selection_area.dart SelectionAreaState.build
func (s *SelectionAreaState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	w := s.Widget()
	controls := w.o.SelectionControls
	if controls == nil {
		switch ThemeOf(bc).Platform {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS:
			controls = MaterialTextSelectionHandleControls
		default:
			controls = DesktopTextSelectionHandleControls
		}
	}
	contextMenuBuilder := w.o.ContextMenuBuilder
	switch {
	case w.o.NoContextMenu:
		contextMenuBuilder = nil
	case contextMenuBuilder == nil:
		contextMenuBuilder = defaultSelectionAreaContextMenuBuilder
	}
	return widgets.NewSelectableRegion(controls, w.child, widgets.SelectableRegionOpts{
		Key:                s.selectableRegionKey,
		FocusNode:          w.o.FocusNode,
		ContextMenuBuilder: contextMenuBuilder,
		OnSelectionChanged: w.o.OnSelectionChanged,
	})
}
