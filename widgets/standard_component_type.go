// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import "github.com/rm4n0s/glutter/foundation"

// StandardComponentType identifies a standard user interface component, such as
// a back button, so that tests and tools can find it by its key.
//
// Flutter: widgets/standard_component_type.dart StandardComponentType
type StandardComponentType int

const (
	// StandardComponentTypeUnspecified is the zero value: no standard component.
	//
	// Flutter: no counterpart (the zero value of the Go enum)
	StandardComponentTypeUnspecified StandardComponentType = iota
	// StandardComponentTypeBackButton indicates that the associated widget is a
	// standard back button, typically used to navigate back to the previous
	// screen.
	//
	// Flutter: widgets/standard_component_type.dart StandardComponentType.backButton
	StandardComponentTypeBackButton
	// StandardComponentTypeCloseButton indicates that the associated widget is a
	// close button, typically used to dismiss a dialog or modal sheet.
	//
	// Flutter: widgets/standard_component_type.dart StandardComponentType.closeButton
	StandardComponentTypeCloseButton
	// StandardComponentTypeMoreButton indicates that the associated widget is a
	// "more" button, typically used to display a menu of additional options.
	//
	// Flutter: widgets/standard_component_type.dart StandardComponentType.moreButton
	StandardComponentTypeMoreButton
	// StandardComponentTypeDrawerButton indicates that the associated widget is
	// a drawer button, typically used to open a drawer.
	//
	// Flutter: widgets/standard_component_type.dart StandardComponentType.drawerButton
	StandardComponentTypeDrawerButton
)

// Key returns a ValueKey for this type: attach it to a widget to indicate that
// it is a standard UI component.
//
// Flutter: widgets/standard_component_type.dart StandardComponentType.key
func (t StandardComponentType) Key() foundation.Key {
	return foundation.NewValueKey(t)
}
