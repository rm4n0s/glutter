// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
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

// nilIfZero is nil for the zero value of an enum (its Unspecified, Dart's null)
// and the value otherwise, for the diagnostics of a nullable enum.
func nilIfZero[T comparable](v T) any {
	var zero T
	if v == zero {
		return nil
	}
	return v
}

// lerpStateWith interpolates two properties like the lerp function does the
// values they resolve to: the property is nil when both are, and a nil property
// is Dart's null for the lerp function. A value that is None resolves to the zero
// value of T.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.lerp
func lerpStateWith[T any](a, b widgets.WidgetStateProperty[T], t float64,
	lerp func(a, b foundation.Opt[T], t float64) foundation.Opt[T]) widgets.WidgetStateProperty[T] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) T {
		var ca, cb foundation.Opt[T]
		if a != nil {
			ca = foundation.Some(a.Resolve(states))
		}
		if b != nil {
			cb = foundation.Some(b.Resolve(states))
		}
		v, _ := lerp(ca, cb, t).Get()
		return v
	})
}

// lerpOptTextStyle is TextStyle.lerp on nullable styles.
func lerpOptTextStyle(a, b foundation.Opt[*painting.TextStyle], t float64) foundation.Opt[*painting.TextStyle] {
	pa, _ := a.Get()
	pb, _ := b.Get()
	r := painting.TextStyleLerp(pa, pb, t)
	if r == nil {
		return foundation.None[*painting.TextStyle]()
	}
	return foundation.Some(r)
}

// lerpOptSize is Size.lerp on nullable sizes.
func lerpOptSize(a, b foundation.Opt[ui.Size], t float64) foundation.Opt[ui.Size] {
	var pa, pb *ui.Size
	if v, ok := a.Get(); ok {
		pa = &v
	}
	if v, ok := b.Get(); ok {
		pb = &v
	}
	if r := ui.SizeLerp(pa, pb, t); r != nil {
		return foundation.Some(*r)
	}
	return foundation.None[ui.Size]()
}

// lerpOptConstraints is BoxConstraints.lerp on nullable constraints.
func lerpOptConstraints(a, b foundation.Opt[rendering.BoxConstraints], t float64) foundation.Opt[rendering.BoxConstraints] {
	var pa, pb *rendering.BoxConstraints
	if v, ok := a.Get(); ok {
		pa = &v
	}
	if v, ok := b.Get(); ok {
		pb = &v
	}
	if r := rendering.BoxConstraintsLerp(pa, pb, t); r != nil {
		return foundation.Some(*r)
	}
	return foundation.None[rendering.BoxConstraints]()
}

// lerpOptBorderRadius is BorderRadius.lerp(a, b, t) for nullable radii.
func lerpOptBorderRadius(a, b foundation.Opt[painting.BorderRadius], t float64) foundation.Opt[painting.BorderRadius] {
	if a.IsNone() && b.IsNone() {
		return foundation.None[painting.BorderRadius]()
	}
	var pa, pb *painting.BorderRadius
	if v, ok := a.Get(); ok {
		pa = &v
	}
	if v, ok := b.Get(); ok {
		pb = &v
	}
	if r := painting.BorderRadiusLerp(pa, pb, t); r != nil {
		return foundation.Some(*r)
	}
	return foundation.None[painting.BorderRadius]()
}

// lerpStateOptColor is WidgetStateProperty.lerp<Color?>(a, b, t, Color.lerp).
func lerpStateOptColor(a, b widgets.WidgetStateProperty[foundation.Opt[ui.Color]], t float64) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		var ca, cb foundation.Opt[ui.Color]
		if a != nil {
			ca = a.Resolve(states)
		}
		if b != nil {
			cb = b.Resolve(states)
		}
		return lerpOptColor(ca, cb, t)
	})
}

// lerpStateNil is WidgetStateProperty.lerp(a, b, t, lerp) for the properties whose
// value type has a nil: nil is Dart's null for the property and for its value.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.lerp
func lerpStateNil[T any](a, b widgets.WidgetStateProperty[T], t float64, lerp func(a, b T, t float64) T) widgets.WidgetStateProperty[T] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) T {
		var va, vb T
		if a != nil {
			va = a.Resolve(states)
		}
		if b != nil {
			vb = b.Resolve(states)
		}
		return lerp(va, vb, t)
	})
}

// lerpStateOpt is WidgetStateProperty.lerp(a, b, t, lerp) for the properties whose
// value is nullable and an Opt.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.lerp
func lerpStateOpt[T any](a, b widgets.WidgetStateProperty[foundation.Opt[T]], t float64,
	lerp func(a, b foundation.Opt[T], t float64) foundation.Opt[T]) widgets.WidgetStateProperty[foundation.Opt[T]] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[T] {
		var va, vb foundation.Opt[T]
		if a != nil {
			va = a.Resolve(states)
		}
		if b != nil {
			vb = b.Resolve(states)
		}
		return lerp(va, vb, t)
	})
}

// lerpOptBorderSide is how WidgetStateBorderSide.lerp interpolates two sides: a
// missing one is a side of no width and the transparent color of the other.
//
// Flutter: painting/borders.dart WidgetStateBorderSide.lerp
func lerpOptBorderSide(a, b foundation.Opt[painting.BorderSide], t float64) foundation.Opt[painting.BorderSide] {
	sa, okA := a.Get()
	sb, okB := b.Get()
	switch {
	case !okA && !okB:
		return foundation.None[painting.BorderSide]()
	case !okA:
		return foundation.Some(painting.BorderSideLerp(painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(0.0), Color: foundation.Some(sb.Color.WithAlpha(0))}), sb, t))
	case !okB:
		return foundation.Some(painting.BorderSideLerp(sa, painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(0.0), Color: foundation.Some(sa.Color.WithAlpha(0))}), t))
	}
	return foundation.Some(painting.BorderSideLerp(sa, sb, t))
}

// lerpWidgetSide is how the themes of Checkbox and Radio interpolate their sides:
// a side that depends on the states is resolved for no state first.
//
// Flutter: material/checkbox_theme.dart CheckboxThemeData._lerpSides
func lerpWidgetSide(a foundation.Opt[painting.BorderSide], aStates widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]],
	b foundation.Opt[painting.BorderSide], bStates widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]], t float64) foundation.Opt[painting.BorderSide] {
	if aStates != nil {
		a = aStates.Resolve(0)
	}
	if bStates != nil {
		b = bStates.Resolve(0)
	}
	return lerpOptBorderSide(a, b, t)
}

// lerpStateOptDouble is WidgetStateProperty.lerp<double?>(a, b, t, lerpDouble).
func lerpStateOptDouble(a, b widgets.WidgetStateProperty[foundation.Opt[float64]], t float64) widgets.WidgetStateProperty[foundation.Opt[float64]] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[float64] {
		var ca, cb foundation.Opt[float64]
		if a != nil {
			ca = a.Resolve(states)
		}
		if b != nil {
			cb = b.Resolve(states)
		}
		return lerpOptDouble(ca, cb, t)
	})
}

// lerpStateOptSide is WidgetStateBorderSide.lerp(a, b, t): the sides of the two
// properties, in each state, interpolate.
//
// Flutter: widgets/widget_state.dart WidgetStateBorderSide.lerp
func lerpStateOptSide(a, b widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]], t float64) widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
		var ca, cb foundation.Opt[painting.BorderSide]
		if a != nil {
			ca = a.Resolve(states)
		}
		if b != nil {
			cb = b.Resolve(states)
		}
		return lerpOptBorderSide(ca, cb, t)
	})
}

// lerpStateOutlined is WidgetStateProperty.lerp<OutlinedBorder?>(a, b, t,
// OutlinedBorder.lerp): a nil border is null.
func lerpStateOutlined(a, b widgets.WidgetStateProperty[painting.OutlinedBorder], t float64) widgets.WidgetStateProperty[painting.OutlinedBorder] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) painting.OutlinedBorder {
		var ca, cb painting.OutlinedBorder
		if a != nil {
			ca = a.Resolve(states)
		}
		if b != nil {
			cb = b.Resolve(states)
		}
		return painting.OutlinedBorderLerp(ca, cb, t)
	})
}

// lerpOptInsets is EdgeInsetsGeometry.lerp on nullable insets.
func lerpOptInsets(a, b foundation.Opt[painting.EdgeInsetsGeometry], t float64) foundation.Opt[painting.EdgeInsetsGeometry] {
	pa, _ := a.Get()
	pb, _ := b.Get()
	if r := painting.EdgeInsetsGeometryLerp(pa, pb, t); r != nil {
		return foundation.Some(r)
	}
	return foundation.None[painting.EdgeInsetsGeometry]()
}
