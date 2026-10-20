// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The functions in this file stand in for the Scrollbar, which comes with step
// 13.10. Each use is marked, and the real widgets replace them when they
// arrive.

// tempScrollbar stands in for the Material Scrollbar that wraps the scrollable
// of a menu: a RawScrollbar with the thickness, the radius, the margin, the
// fading and the colors of the Material 3 scrollbar at rest, without the
// colors for the states of hover and drag.
//
// TODO(step 13.10): Scrollbar.
func tempScrollbar(bc widgets.BuildContext, thumbVisibility bool, child widgets.Widget) widgets.Widget {
	theme := ThemeOf(bc)
	thumb := theme.ColorScheme.OnSurface().WithOpacity(0.1)
	if theme.Brightness() == ui.BrightnessDark {
		thumb = theme.HighlightColor.WithOpacity(1.0)
	}
	return widgets.NewRawScrollbar(child, widgets.RawScrollbarOpts{
		ThumbVisibility: foundation.Some(thumbVisibility),
		Thickness:       foundation.Some(8.0),
		Radius:          foundation.Some(ui.NewRadiusCircular(8.0)),
		CrossAxisMargin: 2.0,
		MinThumbLength:  foundation.Some(48.0),
		FadeDuration:    foundation.Some(300 * time.Millisecond),
		TimeToFade:      foundation.Some(600 * time.Millisecond),
		ThumbColor:      foundation.Some(thumb),
	})
}
