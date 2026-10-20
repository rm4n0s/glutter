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

// The helpers of the component themes: Dart's nullable lerps and equalities on
// the types that Glutter spells as foundation.Opt, a nil pointer or a nil
// interface.

// lerpOptDouble is lerpDouble(a, b, t) for nullable values.
func lerpOptDouble(a, b foundation.Opt[float64], t float64) foundation.Opt[float64] {
	var pa, pb *float64
	if v, ok := a.Get(); ok {
		pa = &v
	}
	if v, ok := b.Get(); ok {
		pb = &v
	}
	if r := ui.LerpDoubleOpt(pa, pb, t); r != nil {
		return foundation.Some(*r)
	}
	return foundation.None[float64]()
}

// lerpOptColor is Color.lerp(a, b, t) for nullable colors.
func lerpOptColor(a, b foundation.Opt[ui.Color], t float64) foundation.Opt[ui.Color] {
	var pa, pb *ui.Color
	if v, ok := a.Get(); ok {
		pa = &v
	}
	if v, ok := b.Get(); ok {
		pb = &v
	}
	if r := ui.ColorLerp(pa, pb, t); r != nil {
		return foundation.Some(*r)
	}
	return foundation.None[ui.Color]()
}

// lerpOptIconTheme is IconThemeData.lerp(a, b, t) for nullable icon themes.
func lerpOptIconTheme(a, b foundation.Opt[widgets.IconThemeData], t float64) foundation.Opt[widgets.IconThemeData] {
	if a.IsNone() && b.IsNone() {
		return foundation.None[widgets.IconThemeData]()
	}
	var pa, pb *widgets.IconThemeData
	if v, ok := a.Get(); ok {
		pa = &v
	}
	if v, ok := b.Get(); ok {
		pb = &v
	}
	return foundation.Some(widgets.IconThemeDataLerp(pa, pb, t))
}

// lerpFirstOrSecond is `t < 0.5 ? a : b`, which Flutter uses for what cannot be
// interpolated.
func lerpFirstOrSecond[T any](a, b T, t float64) T {
	if t < 0.5 {
		return a
	}
	return b
}

// iconThemeOptEqual is Dart's == on nullable icon themes.
func iconThemeOptEqual(a, b foundation.Opt[widgets.IconThemeData]) bool {
	if a.IsNone() || b.IsNone() {
		return a.IsNone() == b.IsNone()
	}
	return a.Value().Equal(b.Value())
}

// colorProp is ColorProperty(name, value, defaultValue: null).
func colorProp(name string, v foundation.Opt[ui.Color]) *foundation.DiagnosticsProperty {
	return painting.NewColorProperty(name, v, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull})
}

// doubleProp is DoubleProperty(name, value, defaultValue: null).
func doubleProp(name string, v foundation.Opt[float64]) *foundation.DiagnosticsProperty {
	return foundation.NewDoubleProperty(name, v, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull})
}

// objectProp is DiagnosticsProperty<T>(name, value, defaultValue: null) for a
// nil-able value.
func objectProp(name string, v any) *foundation.DiagnosticsProperty {
	return foundation.NewDiagnosticsProperty(name, v, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull})
}
