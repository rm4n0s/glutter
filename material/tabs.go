// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

// TabBarIndicatorSize defines how the bounds of the selected tab indicator are
// computed.
//
// Flutter: material/tabs.dart TabBarIndicatorSize
type TabBarIndicatorSize int

const (
	// TabBarIndicatorSizeUnspecified is the zero value: the size of the theme, or
	// tab.
	TabBarIndicatorSizeUnspecified TabBarIndicatorSize = iota
	// TabBarIndicatorSizeTab makes the indicator as wide as the tab.
	//
	// Flutter: material/tabs.dart TabBarIndicatorSize.tab
	TabBarIndicatorSizeTab
	// TabBarIndicatorSizeLabel makes the indicator as wide as the label of the tab.
	//
	// Flutter: material/tabs.dart TabBarIndicatorSize.label
	TabBarIndicatorSizeLabel
)

// Name is the name of the value, as Dart's Enum.name.
func (s TabBarIndicatorSize) Name() string { return enumName(int(s), "tab", "label") }

// String is "TabBarIndicatorSize." and the name.
func (s TabBarIndicatorSize) String() string { return "TabBarIndicatorSize." + s.Name() }

// TabAlignment specifies the horizontal alignment of the tabs of a TabBar.
//
// Flutter: material/tabs.dart TabAlignment
type TabAlignment int

const (
	// TabAlignmentUnspecified is the zero value: the alignment of the theme, or
	// the one that suits the bar.
	TabAlignmentUnspecified TabAlignment = iota
	// TabAlignmentStart aligns the tabs at the start (only for scrollable bars).
	//
	// Flutter: material/tabs.dart TabAlignment.start
	TabAlignmentStart
	// TabAlignmentStartOffset aligns the tabs at the start, after an offset of 52
	// (only for scrollable bars).
	//
	// Flutter: material/tabs.dart TabAlignment.startOffset
	TabAlignmentStartOffset
	// TabAlignmentFill makes the tabs fill the bar (only for bars that are not
	// scrollable).
	//
	// Flutter: material/tabs.dart TabAlignment.fill
	TabAlignmentFill
	// TabAlignmentCenter centers the tabs.
	//
	// Flutter: material/tabs.dart TabAlignment.center
	TabAlignmentCenter
)

// Name is the name of the value, as Dart's Enum.name.
func (a TabAlignment) Name() string {
	return enumName(int(a), "start", "startOffset", "fill", "center")
}

// String is "TabAlignment." and the name.
func (a TabAlignment) String() string { return "TabAlignment." + a.Name() }

// TabIndicatorAnimation defines how the tab indicator animates when the selected
// tab changes.
//
// Flutter: material/tabs.dart TabIndicatorAnimation
type TabIndicatorAnimation int

const (
	// TabIndicatorAnimationUnspecified is the zero value: the animation of the
	// theme, or linear.
	TabIndicatorAnimationUnspecified TabIndicatorAnimation = iota
	// TabIndicatorAnimationLinear moves and resizes the indicator linearly.
	//
	// Flutter: material/tabs.dart TabIndicatorAnimation.linear
	TabIndicatorAnimationLinear
	// TabIndicatorAnimationElastic stretches the indicator while it moves.
	//
	// Flutter: material/tabs.dart TabIndicatorAnimation.elastic
	TabIndicatorAnimationElastic
)

// Name is the name of the value, as Dart's Enum.name.
func (a TabIndicatorAnimation) Name() string { return enumName(int(a), "linear", "elastic") }

// String is "TabIndicatorAnimation." and the name.
func (a TabIndicatorAnimation) String() string { return "TabIndicatorAnimation." + a.Name() }
