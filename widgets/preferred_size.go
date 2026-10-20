// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// PreferredSizeWidget is a widget with a preferred size, which an ancestor such
// as a Scaffold uses to size it without laying it out. Embed the widget in the
// interface values that carry it: a Scaffold finds the size by asserting the
// widget to this interface.
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

// PreferredSizeOpts are the optional parameters of NewPreferredSize.
//
// Flutter: widgets/preferred_size.dart PreferredSize.new
type PreferredSizeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// PreferredSize gives a preferred size to any child, for the places that take a
// PreferredSizeWidget, such as AppBar.bottom. It does not constrain the child;
// the parent decides the size that the child has.
//
// Flutter: widgets/preferred_size.dart PreferredSize
type PreferredSize struct {
	StatelessWidget
	preferredSize ui.Size
	child         Widget
}

// NewPreferredSize creates the widget.
//
// Flutter: widgets/preferred_size.dart PreferredSize.new
func NewPreferredSize(preferredSize ui.Size, child Widget, opts ...PreferredSizeOpts) *PreferredSize {
	o := firstOpts(opts)
	return &PreferredSize{StatelessWidget: NewStatelessWidgetBase(WidgetOpts(o)), preferredSize: preferredSize, child: child}
}

// PreferredSize is the size that the widget would prefer.
//
// Flutter: widgets/preferred_size.dart PreferredSize.preferredSize
func (w *PreferredSize) PreferredSize() ui.Size { return w.preferredSize }

// Child is the widget below this widget in the tree.
//
// Flutter: widgets/preferred_size.dart PreferredSize.child
func (w *PreferredSize) Child() Widget { return w.child }

// Build returns the child.
//
// Flutter: widgets/preferred_size.dart PreferredSize.build
func (w *PreferredSize) Build(BuildContext) Widget { return w.child }
