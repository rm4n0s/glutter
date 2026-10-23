// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const expansionPanelHeaderCollapsedHeight = MinInteractiveDimension

// ExpansionPanelCallback is called when a panel is expanded or collapsed.
//
// Flutter: material/expansion_panel.dart ExpansionPanelCallback
type ExpansionPanelCallback func(panelIndex int, isExpanded bool)

// ExpansionPanelHeaderBuilder builds the header of a panel.
//
// Flutter: material/expansion_panel.dart ExpansionPanelHeaderBuilder
type ExpansionPanelHeaderBuilder func(bc widgets.BuildContext, isExpanded bool) widgets.Widget

// ExpansionPanelOpts are the optional parameters of NewExpansionPanel.
//
// Flutter: material/expansion_panel.dart ExpansionPanel.new
type ExpansionPanelOpts struct {
	// IsExpanded is whether the panel is expanded.
	IsExpanded bool
	// CanTapOnHeader makes a tap on the header expand the panel, and not only a
	// tap on the icon.
	CanTapOnHeader bool
	// BackgroundColor is the color of the panel.
	BackgroundColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the header's tap.
	SplashColor foundation.Opt[ui.Color]
	// HighlightColor is the highlight color of the header's tap.
	HighlightColor foundation.Opt[ui.Color]
}

// ExpansionPanel is a panel of an ExpansionPanelList: a header and a body that
// shows when the panel is expanded.
//
// Flutter: material/expansion_panel.dart ExpansionPanel
type ExpansionPanel struct {
	// HeaderBuilder builds the header.
	HeaderBuilder ExpansionPanelHeaderBuilder
	// Body is the content that shows when the panel is expanded.
	Body widgets.Widget
	// Opts are the optional parameters.
	Opts ExpansionPanelOpts
}

// NewExpansionPanel creates a panel.
//
// Flutter: material/expansion_panel.dart ExpansionPanel.new
func NewExpansionPanel(headerBuilder ExpansionPanelHeaderBuilder, body widgets.Widget, opts ...ExpansionPanelOpts) *ExpansionPanel {
	return &ExpansionPanel{HeaderBuilder: headerBuilder, Body: body, Opts: firstOpts(opts)}
}

// ExpansionPanelRadio is a panel of an ExpansionPanelList that allows one open
// panel only, that has a value to tell it from the others.
//
// Flutter: material/expansion_panel.dart ExpansionPanelRadio
type ExpansionPanelRadio struct {
	ExpansionPanel
	// Value identifies the panel; it must be unique in the list and comparable.
	Value any
}

// NewExpansionPanelRadio creates a panel with a value.
//
// Flutter: material/expansion_panel.dart ExpansionPanelRadio.new
func NewExpansionPanelRadio(value any, headerBuilder ExpansionPanelHeaderBuilder, body widgets.Widget, opts ...ExpansionPanelOpts) *ExpansionPanelRadio {
	return &ExpansionPanelRadio{ExpansionPanel: *NewExpansionPanel(headerBuilder, body, opts...), Value: value}
}

// ExpansionPanelListOpts are the optional parameters of NewExpansionPanelList
// and NewExpansionPanelListRadio.
//
// Flutter: material/expansion_panel.dart ExpansionPanelList.new
type ExpansionPanelListOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ExpansionCallback is called when a panel expands or collapses.
	ExpansionCallback ExpansionPanelCallback
	// AnimationDuration is how long the panels take to expand and collapse; the
	// default is 200 milliseconds.
	AnimationDuration foundation.Opt[time.Duration]
	// InitialOpenPanelValue is the value of the panel that is open at the start
	// of a radio list.
	InitialOpenPanelValue any
	// ExpandedHeaderPadding is the padding around an expanded header; the
	// default is 16 above and below.
	ExpandedHeaderPadding foundation.Opt[painting.EdgeInsets]
	// DividerColor is the color of the dividers.
	DividerColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the panels; it is one of the keys of
	// ElevationToShadow; the default is 2.
	Elevation foundation.Opt[float64]
	// ExpandIconColor is the color of the expand icon.
	ExpandIconColor foundation.Opt[ui.Color]
	// MaterialGapSize is the size of the gap between expanded panels; the default
	// is 16.
	MaterialGapSize foundation.Opt[float64]
}

// ExpansionPanelList is a Material expansion panel list: a list of panels that
// expand and collapse.
//
// Flutter: material/expansion_panel.dart ExpansionPanelList
type ExpansionPanelList struct {
	widgets.StatefulWidget
	children              []*ExpansionPanel
	radios                []*ExpansionPanelRadio
	allowOnlyOnePanelOpen bool
	o                     ExpansionPanelListOpts
}

// NewExpansionPanelList creates a list of panels that open independently.
//
// Flutter: material/expansion_panel.dart ExpansionPanelList.new
func NewExpansionPanelList(children []*ExpansionPanel, opts ...ExpansionPanelListOpts) *ExpansionPanelList {
	o := firstOpts(opts)
	return &ExpansionPanelList{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), children: children, o: o}
}

// NewExpansionPanelListRadio creates a list of panels of which one is open at
// most. The tag of its failure is ExpansionPanelListDuplicateValues.
//
// Flutter: material/expansion_panel.dart ExpansionPanelList.radio
func NewExpansionPanelListRadio(children []*ExpansionPanelRadio, opts ...ExpansionPanelListOpts) *ExpansionPanelList {
	o := firstOpts(opts)
	panels := make([]*ExpansionPanel, len(children))
	for i, c := range children {
		panels[i] = &c.ExpansionPanel
	}
	return &ExpansionPanelList{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), children: panels, radios: children,
		allowOnlyOnePanelOpen: true, o: o}
}

// CreateState creates the state.
//
// Flutter: material/expansion_panel.dart ExpansionPanelList.createState
func (w *ExpansionPanelList) CreateState() widgets.State { return &expansionPanelListState{} }

type expansionPanelListState struct {
	widgets.StateOf[*ExpansionPanelList]
	currentOpenPanel *ExpansionPanelRadio
}

func (s *expansionPanelListState) InitState() {
	w := s.Widget()
	if w.allowOnlyOnePanelOpen {
		s.debugAllIdentifiersUnique()
		if w.o.InitialOpenPanelValue != nil {
			s.currentOpenPanel = searchPanelByValue(w.radios, w.o.InitialOpenPanelValue)
		}
	}
}

func (s *expansionPanelListState) DidUpdateWidget(old *ExpansionPanelList) {
	w := s.Widget()
	if w.allowOnlyOnePanelOpen {
		s.debugAllIdentifiersUnique()
		if !old.allowOnlyOnePanelOpen {
			s.currentOpenPanel = searchPanelByValue(w.radios, w.o.InitialOpenPanelValue)
		}
	} else {
		s.currentOpenPanel = nil
	}
}

func (s *expansionPanelListState) debugAllIdentifiersUnique() {
	if !foundation.DebugMode {
		return
	}
	seen := map[any]bool{}
	for _, c := range s.Widget().radios {
		seen[c.Value] = true
	}
	if len(seen) != len(s.Widget().radios) {
		panic(errors.New(tagExpansionPanelListDuplicateValues, "all ExpansionPanelRadio identifier values must be unique"))
	}
}

func searchPanelByValue(panels []*ExpansionPanelRadio, value any) *ExpansionPanelRadio {
	for _, p := range panels {
		if p.Value == value {
			return p
		}
	}
	return nil
}

func (s *expansionPanelListState) isChildExpanded(index int) bool {
	w := s.Widget()
	if w.allowOnlyOnePanelOpen {
		return s.currentOpenPanel != nil && s.currentOpenPanel.Value == w.radios[index].Value
	}
	return w.children[index].Opts.IsExpanded
}

func (s *expansionPanelListState) handlePressed(isExpanded bool, index int) {
	w := s.Widget()
	if w.allowOnlyOnePanelOpen {
		pressed := w.radios[index]
		for childIndex, child := range w.radios {
			if w.o.ExpansionCallback != nil && childIndex != index && s.currentOpenPanel != nil && child.Value == s.currentOpenPanel.Value {
				w.o.ExpansionCallback(childIndex, false)
			}
		}
		s.SetState(func() {
			if isExpanded {
				s.currentOpenPanel = nil
			} else {
				s.currentOpenPanel = pressed
			}
		})
	}
	if w.o.ExpansionCallback != nil {
		w.o.ExpansionCallback(index, !isExpanded)
	}
}

// expansionPanelSaltedKey is the key of an item of the mergeable material: the
// context and the index of the item.
//
// Flutter: material/expansion_panel.dart _SaltedKey
type expansionPanelSaltedKey struct {
	salt  widgets.BuildContext
	value int
}

func saltedKey(bc widgets.BuildContext, index int) foundation.LocalKey {
	return foundation.NewValueKey(expansionPanelSaltedKey{salt: bc, value: index})
}

// Build builds the panels in a MergeableMaterial.
//
// Flutter: material/expansion_panel.dart _ExpansionPanelListState.build
func (s *expansionPanelListState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	elevation := o.Elevation.OrElse(2)
	if foundation.DebugMode {
		if _, ok := ElevationToShadow[int(elevation)]; !ok || elevation != float64(int(elevation)) {
			panic(errors.New(tagExpansionPanelListInvalidElevation, "invalid value for elevation: it must be one of the keys of ElevationToShadow", "elevation", elevation))
		}
	}
	gapSize := o.MaterialGapSize.OrElse(16)
	duration := o.AnimationDuration.OrElse(ThemeAnimationDuration)
	expandedHeaderPadding := o.ExpandedHeaderPadding.OrElse(painting.NewEdgeInsetsSymmetric(64-expansionPanelHeaderCollapsedHeight, 0))
	var items []MergeableMaterialItem
	for index, child := range w.children {
		expanded := s.isChildExpanded(index)
		if expanded && index != 0 && !s.isChildExpanded(index-1) {
			items = append(items, NewMaterialGap(saltedKey(bc, index*2-1), gapSize))
		}
		headerWidget := child.HeaderBuilder(bc, expanded)
		idx := index
		var expandIconPadded widgets.Widget = widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(0, 0, 8, 0),
			widgets.NewIgnorePointer(NewExpandIcon(func(isExpanded bool) { s.handlePressed(isExpanded, idx) }, ExpandIconOpts{
				Color: o.ExpandIconColor, IsExpanded: expanded, Padding: painting.NewEdgeInsetsAll(12),
				SplashColor: child.Opts.SplashColor, HighlightColor: child.Opts.HighlightColor,
			}), widgets.IgnorePointerOpts{Ignoring: foundation.Some(child.Opts.CanTapOnHeader)}))
		if !child.Opts.CanTapOnHeader {
			localizations := MaterialLocalizationsOf(bc)
			label := localizations.CollapsedIconTapHint()
			if expanded {
				label = localizations.ExpandedIconTapHint()
			}
			expandIconPadded = widgets.NewSemantics(expandIconPadded, widgets.SemanticsOpts{Label: foundation.Some(label), Container: true})
		}
		margin := painting.EdgeInsets{}
		if expanded {
			margin = expandedHeaderPadding
		}
		var header widgets.Widget = widgets.NewRow([]widgets.Widget{
			widgets.NewExpanded(widgets.NewAnimatedContainer(duration, widgets.NewConstrainedBox(
				rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(expansionPanelHeaderCollapsedHeight)}), headerWidget),
				widgets.AnimatedContainerOpts{Curve: animation.Curves.FastOutSlowIn, Margin: margin})),
			expandIconPadded,
		})
		if child.Opts.CanTapOnHeader {
			header = widgets.NewMergeSemantics(NewInkWell(header, InkWellOpts{
				SplashColor: child.Opts.SplashColor, HighlightColor: child.Opts.HighlightColor,
				OnTap: func() { s.handlePressed(s.isChildExpanded(idx), idx) },
			}))
		}
		items = append(items, NewMaterialSlice(saltedKey(bc, index*2), widgets.NewColumn([]widgets.Widget{
			header,
			widgets.NewAnimatedCrossFade(
				widgets.NewLimitedBox(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1)), Height: foundation.Some(0.0)}),
					widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0)}),
				child.Body,
				crossFadeState(expanded), duration, widgets.AnimatedCrossFadeOpts{
					FirstCurve:  animation.NewInterval(0, 0.6, animation.Curves.FastOutSlowIn),
					SecondCurve: animation.NewInterval(0.4, 1, animation.Curves.FastOutSlowIn),
					SizeCurve:   animation.Curves.FastOutSlowIn,
				}),
		}), child.Opts.BackgroundColor))
		if expanded && index != len(w.children)-1 {
			items = append(items, NewMaterialGap(saltedKey(bc, index*2+1), gapSize))
		}
	}
	return NewMergeableMaterial(items, MergeableMaterialOpts{HasDividers: true, DividerColor: o.DividerColor, Elevation: foundation.Some(elevation)})
}

func crossFadeState(expanded bool) widgets.CrossFadeState {
	if expanded {
		return widgets.CrossFadeStateShowSecond
	}
	return widgets.CrossFadeStateShowFirst
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagExpansionPanelListDuplicateValues  = "ExpansionPanelListDuplicateValues"
	tagExpansionPanelListInvalidElevation = "ExpansionPanelListInvalidElevation"
)
