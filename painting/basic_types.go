// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// Axis is a direction along which objects are laid out. The zero value is
// AxisUnspecified, the default of the widget.
//
// Flutter: painting/basic_types.dart Axis
type Axis int

// The values of Axis follow Flutter's declaration order.
const (
	// AxisUnspecified is the zero value: the widget's default.
	//
	// Flutter: no counterpart
	AxisUnspecified Axis = iota
	// AxisHorizontal lays objects out in the horizontal direction.
	//
	// Flutter: painting/basic_types.dart Axis.horizontal
	AxisHorizontal
	// AxisVertical lays objects out in the vertical direction.
	//
	// Flutter: painting/basic_types.dart Axis.vertical
	AxisVertical
)

// String returns the name Dart prints, such as "Axis.horizontal".
//
// Flutter: painting/basic_types.dart Axis.toString
func (a Axis) String() string {
	switch a {
	case AxisHorizontal:
		return "Axis.horizontal"
	case AxisVertical:
		return "Axis.vertical"
	}
	return "Axis.unspecified"
}

// FlipAxis returns the other axis.
//
// Flutter: painting/basic_types.dart flipAxis
func FlipAxis(direction Axis) Axis {
	switch direction {
	case AxisHorizontal:
		return AxisVertical
	case AxisVertical:
		return AxisHorizontal
	}
	return direction
}

// VerticalDirection is a direction in which boxes flow vertically.
//
// Flutter: painting/basic_types.dart VerticalDirection
type VerticalDirection int

// The values of VerticalDirection follow Flutter's declaration order.
const (
	// VerticalDirectionUnspecified is the zero value: the widget's default
	// (down).
	//
	// Flutter: no counterpart
	VerticalDirectionUnspecified VerticalDirection = iota
	// VerticalDirectionUp is from bottom to top.
	//
	// Flutter: painting/basic_types.dart VerticalDirection.up
	VerticalDirectionUp
	// VerticalDirectionDown is from top to bottom.
	//
	// Flutter: painting/basic_types.dart VerticalDirection.down
	VerticalDirectionDown
)

// String returns the name Dart prints, such as "VerticalDirection.up".
//
// Flutter: painting/basic_types.dart VerticalDirection.toString
func (d VerticalDirection) String() string {
	switch d {
	case VerticalDirectionUp:
		return "VerticalDirection.up"
	case VerticalDirectionDown:
		return "VerticalDirection.down"
	}
	return "VerticalDirection.unspecified"
}

// AxisDirection is a direction along either the horizontal or the vertical axis.
//
// Flutter: painting/basic_types.dart AxisDirection
type AxisDirection int

// The values of AxisDirection follow Flutter's declaration order.
const (
	// AxisDirectionUnspecified is the zero value: the widget's default.
	//
	// Flutter: no counterpart
	AxisDirectionUnspecified AxisDirection = iota
	// AxisDirectionUp is towards the top.
	//
	// Flutter: painting/basic_types.dart AxisDirection.up
	AxisDirectionUp
	// AxisDirectionRight is towards the right.
	//
	// Flutter: painting/basic_types.dart AxisDirection.right
	AxisDirectionRight
	// AxisDirectionDown is towards the bottom.
	//
	// Flutter: painting/basic_types.dart AxisDirection.down
	AxisDirectionDown
	// AxisDirectionLeft is towards the left.
	//
	// Flutter: painting/basic_types.dart AxisDirection.left
	AxisDirectionLeft
)

// String returns the name Dart prints, such as "AxisDirection.up".
//
// Flutter: painting/basic_types.dart AxisDirection.toString
func (d AxisDirection) String() string {
	switch d {
	case AxisDirectionUp:
		return "AxisDirection.up"
	case AxisDirectionRight:
		return "AxisDirection.right"
	case AxisDirectionDown:
		return "AxisDirection.down"
	case AxisDirectionLeft:
		return "AxisDirection.left"
	}
	return "AxisDirection.unspecified"
}

// AxisDirectionToAxis returns the axis of a direction.
//
// Flutter: painting/basic_types.dart axisDirectionToAxis
func AxisDirectionToAxis(d AxisDirection) Axis {
	switch d {
	case AxisDirectionUp, AxisDirectionDown:
		return AxisVertical
	case AxisDirectionLeft, AxisDirectionRight:
		return AxisHorizontal
	}
	return AxisUnspecified
}

// TextDirectionToAxisDirection returns the axis direction in which text flows.
//
// Flutter: painting/basic_types.dart textDirectionToAxisDirection
func TextDirectionToAxisDirection(d ui.TextDirection) AxisDirection {
	if d == ui.TextDirectionRTL {
		return AxisDirectionLeft
	}
	return AxisDirectionRight
}

// FlipAxisDirection returns the opposite direction.
//
// Flutter: painting/basic_types.dart flipAxisDirection
func FlipAxisDirection(d AxisDirection) AxisDirection {
	switch d {
	case AxisDirectionUp:
		return AxisDirectionDown
	case AxisDirectionRight:
		return AxisDirectionLeft
	case AxisDirectionDown:
		return AxisDirectionUp
	case AxisDirectionLeft:
		return AxisDirectionRight
	}
	return d
}

// AxisDirectionIsReversed reports whether the direction runs against the
// reading order (up and left).
//
// Flutter: painting/basic_types.dart axisDirectionIsReversed
func AxisDirectionIsReversed(d AxisDirection) bool {
	return d == AxisDirectionUp || d == AxisDirectionLeft
}

// Name returns the Dart enum name, such as "horizontal".
//
// Flutter: painting/basic_types.dart Axis.name
func (a Axis) Name() string { return strings.TrimPrefix(a.String(), "Axis.") }

// Name returns the Dart enum name, such as "up".
//
// Flutter: painting/basic_types.dart VerticalDirection.name
func (d VerticalDirection) Name() string { return strings.TrimPrefix(d.String(), "VerticalDirection.") }

// Name returns the Dart enum name, such as "up".
//
// Flutter: painting/basic_types.dart AxisDirection.name
func (d AxisDirection) Name() string { return strings.TrimPrefix(d.String(), "AxisDirection.") }

// RenderComparison is the description of the difference between two objects, in
// terms of how much work is needed to apply it: nothing, the metadata, a repaint
// or a new layout.
//
// Flutter: painting/basic_types.dart RenderComparison
type RenderComparison int

// The values of RenderComparison follow Flutter's declaration order.
const (
	// RenderComparisonIdentical means the objects are the same: no work is needed.
	//
	// Flutter: painting/basic_types.dart RenderComparison.identical
	RenderComparisonIdentical RenderComparison = iota
	// RenderComparisonMetadata means that only the metadata, such as the
	// semantics, changed.
	//
	// Flutter: painting/basic_types.dart RenderComparison.metadata
	RenderComparisonMetadata
	// RenderComparisonPaint means that the object needs to be repainted.
	//
	// Flutter: painting/basic_types.dart RenderComparison.paint
	RenderComparisonPaint
	// RenderComparisonLayout means that the object needs a new layout.
	//
	// Flutter: painting/basic_types.dart RenderComparison.layout
	RenderComparisonLayout
)

// Name returns the Dart enum name, such as "layout".
func (c RenderComparison) Name() string {
	switch c {
	case RenderComparisonMetadata:
		return "metadata"
	case RenderComparisonPaint:
		return "paint"
	case RenderComparisonLayout:
		return "layout"
	}
	return "identical"
}

// String returns the name Dart prints, such as "RenderComparison.layout".
func (c RenderComparison) String() string { return "RenderComparison." + c.Name() }
