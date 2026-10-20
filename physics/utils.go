// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package physics

import "github.com/rm4n0s/glutter/foundation"

// NearEqual reports whether a and b are within epsilon of each other. Equal
// infinities are near equal. A negative epsilon panics with the tag
// EpsilonNegative in debug builds.
//
// Flutter: physics/utils.dart nearEqual
func NearEqual(a, b, epsilon float64) bool {
	if foundation.DebugMode && epsilon < 0 {
		panicEpsilon(epsilon)
	}
	return (a > (b-epsilon) && a < (b+epsilon)) || a == b
}

// NearEqualOpt is NearEqual for values that may be missing: two missing values
// are near equal, and a missing value is never near a present one.
//
// Flutter: physics/utils.dart nearEqual
func NearEqualOpt(a, b foundation.Opt[float64], epsilon float64) bool {
	av, aok := a.Get()
	bv, bok := b.Get()
	if !aok || !bok {
		return aok == bok
	}
	return NearEqual(av, bv, epsilon)
}

// NearZero reports whether a is within epsilon of zero.
//
// Flutter: physics/utils.dart nearZero
func NearZero(a, epsilon float64) bool { return NearEqual(a, 0, epsilon) }
