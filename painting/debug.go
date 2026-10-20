// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// debugCheckCanResolveTextDirection panics in debug builds if the text direction
// is unspecified, naming the object that needs it.
//
// Flutter: painting/debug.dart debugCheckCanResolveTextDirection
func debugCheckCanResolveTextDirection(direction ui.TextDirection, target string) {
	if foundation.DebugMode && direction == ui.TextDirectionUnspecified {
		panic(errors.New("TextDirectionRequired",
			"a TextDirection is required to resolve a directional value; a directional "+target+" cannot be resolved without one",
			"target", target))
	}
}
