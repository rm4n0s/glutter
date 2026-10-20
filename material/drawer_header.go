// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/widgets"
)

// Flutter: material/drawer_header.dart _kDrawerHeaderHeight
const drawerHeaderHeight = 160.0 + 1.0 // bottom edge

// DrawerHeaderOpts are the optional parameters of NewDrawerHeader.
//
// Flutter: material/drawer_header.dart DrawerHeader.new
type DrawerHeaderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Decoration is the decoration of the header, animated when it changes.
	Decoration painting.Decoration
	// Margin is the margin around the header; the default is a bottom margin of 8.
	// Pass painting.EdgeInsetsZero for none (Flutter's margin: null).
	Margin painting.EdgeInsetsGeometry
	// Padding is the padding inside the header; the default is 16 around, 8 at the
	// bottom.
	Padding painting.EdgeInsetsGeometry
	// Duration is the duration of the animation of the decoration; the default is
	// 250 ms.
	Duration time.Duration
	// Curve is the curve of the animation; the default is Curves.FastOutSlowIn.
	Curve animation.Curve
	// Child is the content of the header, in the body text style.
	Child widgets.Widget
}

// DrawerHeader is the top-most region of a Material Design drawer. The header
// stays in place when the rest of the drawer scrolls.
//
// Flutter: material/drawer_header.dart DrawerHeader
type DrawerHeader struct {
	widgets.StatelessWidget
	o DrawerHeaderOpts
}

// NewDrawerHeader creates a header.
//
// Flutter: material/drawer_header.dart DrawerHeader.new
func NewDrawerHeader(opts ...DrawerHeaderOpts) *DrawerHeader {
	o := firstOpts(opts)
	if o.Margin == nil {
		o.Margin = painting.NewEdgeInsetsFromLTRB(0, 0, 0, 8)
	}
	if o.Padding == nil {
		o.Padding = painting.NewEdgeInsetsFromLTRB(16, 16, 16, 8)
	}
	if o.Duration == 0 {
		o.Duration = 250 * time.Millisecond
	}
	if o.Curve == nil {
		o.Curve = animation.Curves.FastOutSlowIn
	}
	return &DrawerHeader{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Child is the content of the header.
//
// Flutter: material/drawer_header.dart DrawerHeader.child
func (w *DrawerHeader) Child() widgets.Widget { return w.o.Child }

// Build builds the header.
//
// Flutter: material/drawer_header.dart DrawerHeader.build
func (w *DrawerHeader) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	widgets.DebugCheckHasMediaQuery(bc)
	theme := ThemeOf(bc)
	statusBarHeight := widgets.MediaQueryPaddingOf(bc).Top
	var child widgets.Widget
	if w.o.Child != nil {
		child = widgets.NewDefaultTextStyle(theme.TextTheme.BodyLarge,
			widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Top: true}, w.o.Child))
	}
	return widgets.NewContainer(
		widgets.NewAnimatedContainer(w.o.Duration, child, widgets.AnimatedContainerOpts{
			Padding:    w.o.Padding.Add(painting.NewEdgeInsetsFromLTRB(0, statusBarHeight, 0, 0)),
			Decoration: w.o.Decoration,
			Curve:      w.o.Curve,
		}),
		widgets.ContainerOpts{
			Height: foundation.Some(statusBarHeight + drawerHeaderHeight),
			Margin: w.o.Margin,
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
				Border: painting.Border{Bottom: DividerCreateBorderSide(bc, DividerBorderSideOpts{Width: foundation.Some(0.0)})}}),
		})
}
