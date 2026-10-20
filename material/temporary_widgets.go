// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The functions in this file stand in for Divider and Tooltip, which come with
// step 13.9. Each use is marked, and the real widgets replace them when they
// arrive.

// tempHairlineDivider stands in for Divider(height: 0, color: color): a hairline
// along the bottom of a zero height box.
//
// TODO(step 13.9): Divider.
func tempHairlineDivider(color foundation.Opt[ui.Color]) widgets.Widget {
	side := painting.NewBorderSide(painting.BorderSideOpts{Color: color, Width: foundation.Some(0.0)})
	return widgets.NewSizedBox(
		widgets.NewCenter(widgets.NewContainer(nil, widgets.ContainerOpts{
			Height:     foundation.Some(0.0),
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: painting.Border{Bottom: side}}),
		})),
		widgets.SizedBoxOpts{Height: foundation.Some(0.0)})
}

// tempTooltip stands in for a Tooltip: it only gives the child the tooltip
// semantics.
//
// TODO(step 13.9): Tooltip.
func tempTooltip(message string, child widgets.Widget) widgets.Widget {
	return widgets.NewSemantics(child, widgets.SemanticsOpts{Tooltip: foundation.Some(message)})
}
