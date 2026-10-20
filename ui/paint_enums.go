// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

// dartEnumName returns the name of an enum value the way Dart's toString does:
// "Type.name". An out of range value gives "Type(v)".
func dartEnumName(names []string, v int, typeName string) string {
	if v >= 0 && v < len(names) {
		return typeName + "." + names[v]
	}
	return enumName(nil, v, typeName)
}

// BlendMode describes the algorithm used to combine a source and a destination image when drawing; the order is Skia's. The default is BlendModeSrcOver.
//
// Flutter: ui/painting.dart BlendMode
type BlendMode int

// The values of BlendMode follow Flutter's declaration order.
const (
	// BlendModeUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	BlendModeUnspecified BlendMode = iota
	// BlendModeClear is Flutter's BlendMode.clear. Drop both the source and destination images, leaving nothing.
	//
	// Flutter: ui/painting.dart BlendMode.clear
	BlendModeClear
	// BlendModeSrc is Flutter's BlendMode.src. Drop the destination image, only paint the source image.
	//
	// Flutter: ui/painting.dart BlendMode.src
	BlendModeSrc
	// BlendModeDst is Flutter's BlendMode.dst. Drop the source image, only paint the destination image.
	//
	// Flutter: ui/painting.dart BlendMode.dst
	BlendModeDst
	// BlendModeSrcOver is Flutter's BlendMode.srcOver. Composite the source image over the destination image. This is the default.
	//
	// Flutter: ui/painting.dart BlendMode.srcOver
	BlendModeSrcOver
	// BlendModeDstOver is Flutter's BlendMode.dstOver. Composite the source image under the destination image.
	//
	// Flutter: ui/painting.dart BlendMode.dstOver
	BlendModeDstOver
	// BlendModeSrcIn is Flutter's BlendMode.srcIn. Show the source image, but only where the two images overlap. The destination is treated merely as a mask.
	//
	// Flutter: ui/painting.dart BlendMode.srcIn
	BlendModeSrcIn
	// BlendModeDstIn is Flutter's BlendMode.dstIn. Show the destination image, but only where the two images overlap. The source is treated merely as a mask.
	//
	// Flutter: ui/painting.dart BlendMode.dstIn
	BlendModeDstIn
	// BlendModeSrcOut is Flutter's BlendMode.srcOut. Show the source image, but only where the two images do not overlap.
	//
	// Flutter: ui/painting.dart BlendMode.srcOut
	BlendModeSrcOut
	// BlendModeDstOut is Flutter's BlendMode.dstOut. Show the destination image, but only where the two images do not overlap.
	//
	// Flutter: ui/painting.dart BlendMode.dstOut
	BlendModeDstOut
	// BlendModeSrcATop is Flutter's BlendMode.srcATop. Composite the source image over the destination image, but only where it overlaps the destination.
	//
	// Flutter: ui/painting.dart BlendMode.srcATop
	BlendModeSrcATop
	// BlendModeDstATop is Flutter's BlendMode.dstATop. Composite the destination image over the source image, but only where it overlaps the source.
	//
	// Flutter: ui/painting.dart BlendMode.dstATop
	BlendModeDstATop
	// BlendModeXor is Flutter's BlendMode.xor. Apply a bitwise xor operator to the source and destination images, which leaves transparency where they overlap.
	//
	// Flutter: ui/painting.dart BlendMode.xor
	BlendModeXor
	// BlendModePlus is Flutter's BlendMode.plus. Sum the components of the source and destination images.
	//
	// Flutter: ui/painting.dart BlendMode.plus
	BlendModePlus
	// BlendModeModulate is Flutter's BlendMode.modulate. Multiply the color components of the source and destination images.
	//
	// Flutter: ui/painting.dart BlendMode.modulate
	BlendModeModulate
	// BlendModeScreen is Flutter's BlendMode.screen. Multiply the inverse of the components of the source and destination images, and inverse the result.
	//
	// Flutter: ui/painting.dart BlendMode.screen
	BlendModeScreen
	// BlendModeOverlay is Flutter's BlendMode.overlay. Multiply the components of the source and destination images after adjusting them to favor the destination.
	//
	// Flutter: ui/painting.dart BlendMode.overlay
	BlendModeOverlay
	// BlendModeDarken is Flutter's BlendMode.darken. Composite the source and destination image by choosing the lowest value from each color channel.
	//
	// Flutter: ui/painting.dart BlendMode.darken
	BlendModeDarken
	// BlendModeLighten is Flutter's BlendMode.lighten. Composite the source and destination image by choosing the highest value from each color channel.
	//
	// Flutter: ui/painting.dart BlendMode.lighten
	BlendModeLighten
	// BlendModeColorDodge is Flutter's BlendMode.colorDodge. Divide the destination by the inverse of the source.
	//
	// Flutter: ui/painting.dart BlendMode.colorDodge
	BlendModeColorDodge
	// BlendModeColorBurn is Flutter's BlendMode.colorBurn. Divide the inverse of the destination by the source, and inverse the result.
	//
	// Flutter: ui/painting.dart BlendMode.colorBurn
	BlendModeColorBurn
	// BlendModeHardLight is Flutter's BlendMode.hardLight. Multiply the components of the source and destination images after adjusting them to favor the source.
	//
	// Flutter: ui/painting.dart BlendMode.hardLight
	BlendModeHardLight
	// BlendModeSoftLight is Flutter's BlendMode.softLight. Use colorDodge for source values below 0.5 and colorBurn for source values above 0.5.
	//
	// Flutter: ui/painting.dart BlendMode.softLight
	BlendModeSoftLight
	// BlendModeDifference is Flutter's BlendMode.difference. Subtract the smaller value from the bigger value for each channel.
	//
	// Flutter: ui/painting.dart BlendMode.difference
	BlendModeDifference
	// BlendModeExclusion is Flutter's BlendMode.exclusion. Subtract double the product of the two images from the sum of the two images.
	//
	// Flutter: ui/painting.dart BlendMode.exclusion
	BlendModeExclusion
	// BlendModeMultiply is Flutter's BlendMode.multiply. Multiply the components of the source and destination images, including the alpha channel.
	//
	// Flutter: ui/painting.dart BlendMode.multiply
	BlendModeMultiply
	// BlendModeHue is Flutter's BlendMode.hue. Take the hue of the source image, and the saturation and luminosity of the destination image.
	//
	// Flutter: ui/painting.dart BlendMode.hue
	BlendModeHue
	// BlendModeSaturation is Flutter's BlendMode.saturation. Take the saturation of the source image, and the hue and luminosity of the destination image.
	//
	// Flutter: ui/painting.dart BlendMode.saturation
	BlendModeSaturation
	// BlendModeColor is Flutter's BlendMode.color. Take the hue and saturation of the source image, and the luminosity of the destination image.
	//
	// Flutter: ui/painting.dart BlendMode.color
	BlendModeColor
	// BlendModeLuminosity is Flutter's BlendMode.luminosity. Take the luminosity of the source image, and the hue and saturation of the destination image.
	//
	// Flutter: ui/painting.dart BlendMode.luminosity
	BlendModeLuminosity
)

var blendModeNames = [...]string{"unspecified", "clear", "src", "dst", "srcOver", "dstOver", "srcIn", "dstIn", "srcOut", "dstOut", "srcATop", "dstATop", "xor", "plus", "modulate", "screen", "overlay", "darken", "lighten", "colorDodge", "colorBurn", "hardLight", "softLight", "difference", "exclusion", "multiply", "hue", "saturation", "color", "luminosity"}

// String returns the name Dart prints for the value, such as "BlendMode.clear".
func (e BlendMode) String() string { return dartEnumName(blendModeNames[:], int(e), "BlendMode") }

// FilterQuality is the quality of filtering to use when an image is scaled. The default for a Paint is FilterQualityNone.
//
// Flutter: ui/painting.dart FilterQuality
type FilterQuality int

// The values of FilterQuality follow Flutter's declaration order.
const (
	// FilterQualityUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	FilterQualityUnspecified FilterQuality = iota
	// FilterQualityNone is Flutter's FilterQuality.none. Fastest possible filtering, albeit also the lowest quality: nearest-neighbor sampling.
	//
	// Flutter: ui/painting.dart FilterQuality.none
	FilterQualityNone
	// FilterQualityLow is Flutter's FilterQuality.low. Better quality than none, faster than medium: bilinear interpolation.
	//
	// Flutter: ui/painting.dart FilterQuality.low
	FilterQualityLow
	// FilterQualityMedium is Flutter's FilterQuality.medium. Better quality than low, faster than high: bilinear interpolation with mipmaps.
	//
	// Flutter: ui/painting.dart FilterQuality.medium
	FilterQualityMedium
	// FilterQualityHigh is Flutter's FilterQuality.high. Best possible quality when scaling up images, scaling down images (slow).
	//
	// Flutter: ui/painting.dart FilterQuality.high
	FilterQualityHigh
)

var filterQualityNames = [...]string{"unspecified", "none", "low", "medium", "high"}

// String returns the name Dart prints for the value, such as "FilterQuality.none".
func (e FilterQuality) String() string {
	return dartEnumName(filterQualityNames[:], int(e), "FilterQuality")
}

// StrokeCap is the shape drawn at the ends of a stroked open path. The default is StrokeCapButt.
//
// Flutter: ui/painting.dart StrokeCap
type StrokeCap int

// The values of StrokeCap follow Flutter's declaration order.
const (
	// StrokeCapUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	StrokeCapUnspecified StrokeCap = iota
	// StrokeCapButt is Flutter's StrokeCap.butt. Begin and end contours with a flat edge and no extension.
	//
	// Flutter: ui/painting.dart StrokeCap.butt
	StrokeCapButt
	// StrokeCapRound is Flutter's StrokeCap.round. Begin and end contours with a semi-circle extension.
	//
	// Flutter: ui/painting.dart StrokeCap.round
	StrokeCapRound
	// StrokeCapSquare is Flutter's StrokeCap.square. Begin and end contours with a half square extension.
	//
	// Flutter: ui/painting.dart StrokeCap.square
	StrokeCapSquare
)

var strokeCapNames = [...]string{"unspecified", "butt", "round", "square"}

// String returns the name Dart prints for the value, such as "StrokeCap.butt".
func (e StrokeCap) String() string { return dartEnumName(strokeCapNames[:], int(e), "StrokeCap") }

// StrokeJoin is the shape drawn where two segments of a stroked path meet. The default is StrokeJoinMiter.
//
// Flutter: ui/painting.dart StrokeJoin
type StrokeJoin int

// The values of StrokeJoin follow Flutter's declaration order.
const (
	// StrokeJoinUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	StrokeJoinUnspecified StrokeJoin = iota
	// StrokeJoinMiter is Flutter's StrokeJoin.miter. Joins between line segments form sharp corners, up to the miter limit.
	//
	// Flutter: ui/painting.dart StrokeJoin.miter
	StrokeJoinMiter
	// StrokeJoinRound is Flutter's StrokeJoin.round. Joins between line segments form circular arcs.
	//
	// Flutter: ui/painting.dart StrokeJoin.round
	StrokeJoinRound
	// StrokeJoinBevel is Flutter's StrokeJoin.bevel. Joins between line segments are cut off with a straight line.
	//
	// Flutter: ui/painting.dart StrokeJoin.bevel
	StrokeJoinBevel
)

var strokeJoinNames = [...]string{"unspecified", "miter", "round", "bevel"}

// String returns the name Dart prints for the value, such as "StrokeJoin.miter".
func (e StrokeJoin) String() string { return dartEnumName(strokeJoinNames[:], int(e), "StrokeJoin") }

// PaintingStyle is whether to paint inside shapes, the edges of shapes, or both. The default is PaintingStyleFill.
//
// Flutter: ui/painting.dart PaintingStyle
type PaintingStyle int

// The values of PaintingStyle follow Flutter's declaration order.
const (
	// PaintingStyleUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	PaintingStyleUnspecified PaintingStyle = iota
	// PaintingStyleFill is Flutter's PaintingStyle.fill. Apply the Paint to the inside of the shape.
	//
	// Flutter: ui/painting.dart PaintingStyle.fill
	PaintingStyleFill
	// PaintingStyleStroke is Flutter's PaintingStyle.stroke. Apply the Paint to the edge of the shape.
	//
	// Flutter: ui/painting.dart PaintingStyle.stroke
	PaintingStyleStroke
)

var paintingStyleNames = [...]string{"unspecified", "fill", "stroke"}

// String returns the name Dart prints for the value, such as "PaintingStyle.fill".
func (e PaintingStyle) String() string {
	return dartEnumName(paintingStyleNames[:], int(e), "PaintingStyle")
}

// Clip lists the different ways to clip a widget's content.
//
// Flutter: ui/painting.dart Clip
type Clip int

// The values of Clip follow Flutter's declaration order.
const (
	// ClipUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	ClipUnspecified Clip = iota
	// ClipNone is Flutter's Clip.none. No clip at all.
	//
	// Flutter: ui/painting.dart Clip.none
	ClipNone
	// ClipHardEdge is Flutter's Clip.hardEdge. Clip, but do not apply anti-aliasing.
	//
	// Flutter: ui/painting.dart Clip.hardEdge
	ClipHardEdge
	// ClipAntiAlias is Flutter's Clip.antiAlias. Clip with anti-aliasing.
	//
	// Flutter: ui/painting.dart Clip.antiAlias
	ClipAntiAlias
	// ClipAntiAliasWithSaveLayer is Flutter's Clip.antiAliasWithSaveLayer. Clip with anti-aliasing and saveLayer immediately following the clip.
	//
	// Flutter: ui/painting.dart Clip.antiAliasWithSaveLayer
	ClipAntiAliasWithSaveLayer
)

var clipNames = [...]string{"unspecified", "none", "hardEdge", "antiAlias", "antiAliasWithSaveLayer"}

// String returns the name Dart prints for the value, such as "Clip.none".
func (e Clip) String() string { return dartEnumName(clipNames[:], int(e), "Clip") }

// TileMode is what the shader does outside the bounds it was defined for. The default is TileModeClamp.
//
// Flutter: ui/painting.dart TileMode
type TileMode int

// The values of TileMode follow Flutter's declaration order.
const (
	// TileModeUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	TileModeUnspecified TileMode = iota
	// TileModeClamp is Flutter's TileMode.clamp. Edge is extended indefinitely.
	//
	// Flutter: ui/painting.dart TileMode.clamp
	TileModeClamp
	// TileModeRepeated is Flutter's TileMode.repeated. Edge is repeated from first to last.
	//
	// Flutter: ui/painting.dart TileMode.repeated
	TileModeRepeated
	// TileModeMirror is Flutter's TileMode.mirror. Edge is mirrored from last to first to last.
	//
	// Flutter: ui/painting.dart TileMode.mirror
	TileModeMirror
	// TileModeDecal is Flutter's TileMode.decal. Render the shader's image pixels only within its original bounds; outside is transparent.
	//
	// Flutter: ui/painting.dart TileMode.decal
	TileModeDecal
)

var tileModeNames = [...]string{"unspecified", "clamp", "repeated", "mirror", "decal"}

// String returns the name Dart prints for the value, such as "TileMode.clamp".
func (e TileMode) String() string { return dartEnumName(tileModeNames[:], int(e), "TileMode") }

// BlurStyle lists the styles to use for blurs in MaskFilter objects.
//
// Flutter: ui/painting.dart BlurStyle
type BlurStyle int

// The values of BlurStyle follow Flutter's declaration order.
const (
	// BlurStyleUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	BlurStyleUnspecified BlurStyle = iota
	// BlurStyleNormal is Flutter's BlurStyle.normal. Fuzzy inside and outside. This is useful for painting shadows that are offset from the shape that ostensibly is casting the shadow.
	//
	// Flutter: ui/painting.dart BlurStyle.normal
	BlurStyleNormal
	// BlurStyleSolid is Flutter's BlurStyle.solid. Solid inside, fuzzy outside. This corresponds to drawing the shape, and additionally drawing the blur.
	//
	// Flutter: ui/painting.dart BlurStyle.solid
	BlurStyleSolid
	// BlurStyleOuter is Flutter's BlurStyle.outer. Nothing inside, fuzzy outside. This is useful for painting shadows for partially transparent shapes.
	//
	// Flutter: ui/painting.dart BlurStyle.outer
	BlurStyleOuter
	// BlurStyleInner is Flutter's BlurStyle.inner. Fuzzy inside, nothing outside. This can make shapes appear to be lit from within.
	//
	// Flutter: ui/painting.dart BlurStyle.inner
	BlurStyleInner
)

var blurStyleNames = [...]string{"unspecified", "normal", "solid", "outer", "inner"}

// String returns the name Dart prints for the value, such as "BlurStyle.normal".
func (e BlurStyle) String() string { return dartEnumName(blurStyleNames[:], int(e), "BlurStyle") }

// ColorSpace lists the color spaces a Color can be in. Unlike the other enums of this package its zero value is ColorSpaceSRGB, so that a Color written as a literal is in sRGB.
//
// Flutter: ui/painting.dart ColorSpace
type ColorSpace int

// The values of ColorSpace follow Flutter's declaration order.
const (
	// ColorSpaceSRGB is Flutter's ColorSpace.sRGB. The sRGB color space.
	//
	// Flutter: ui/painting.dart ColorSpace.sRGB
	ColorSpaceSRGB ColorSpace = iota
	// ColorSpaceExtendedSRGB is Flutter's ColorSpace.extendedSRGB. The extended sRGB color space: sRGB with component values outside 0 to 1.
	//
	// Flutter: ui/painting.dart ColorSpace.extendedSRGB
	ColorSpaceExtendedSRGB
	// ColorSpaceDisplayP3 is Flutter's ColorSpace.displayP3. The Display P3 color space, a wider gamut than sRGB.
	//
	// Flutter: ui/painting.dart ColorSpace.displayP3
	ColorSpaceDisplayP3
)

var colorSpaceNames = [...]string{"sRGB", "extendedSRGB", "displayP3"}

// String returns the name Dart prints for the value, such as "ColorSpace.sRGB".
func (e ColorSpace) String() string { return dartEnumName(colorSpaceNames[:], int(e), "ColorSpace") }
