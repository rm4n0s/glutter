// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import "github.com/rm4n0s/glutter/ui"

// PreferredSizeWidget is a widget with a preferred size, which an ancestor such
// as a Scaffold uses to size it without laying it out. Embed the widget in the
// interface values that carry it: a Scaffold finds the size by asserting the
// widget to this interface.
//
// TODO: the PreferredSize widget, which gives a preferred size to any child.
//
// Flutter: widgets/preferred_size.dart PreferredSizeWidget
type PreferredSizeWidget interface {
	Widget
	// PreferredSize is the size this widget would prefer if it were otherwise
	// unconstrained. In many cases it is only the height, with the width
	// infinite.
	//
	// Flutter: widgets/preferred_size.dart PreferredSizeWidget.preferredSize
	PreferredSize() ui.Size
}
