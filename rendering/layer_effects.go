// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

func (b *LayerBase) debugOptions() *DebugOptions {
	if o, ok := b.owner.(interface{ DebugOptions() *DebugOptions }); ok {
		if d := o.DebugOptions(); d != nil {
			return d
		}
	}
	return &DebugOptions{}
}

// ColorFilterLayer is a layer that applies a color filter to its children.
//
// Flutter: rendering/layer.dart ColorFilterLayer
type ColorFilterLayer struct {
	ContainerLayer
	colorFilter *ui.ColorFilter
}

// NewColorFilterLayer creates a layer that applies the filter (nil leaves it
// to SetColorFilter).
//
// Flutter: rendering/layer.dart ColorFilterLayer.new
func NewColorFilterLayer(colorFilter *ui.ColorFilter) *ColorFilterLayer {
	l := &ColorFilterLayer{colorFilter: colorFilter}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// ColorFilter is the filter.
//
// Flutter: rendering/layer.dart ColorFilterLayer.colorFilter
func (l *ColorFilterLayer) ColorFilter() *ui.ColorFilter { return l.colorFilter }

// SetColorFilter changes the filter, which must not be nil.
//
// Flutter: rendering/layer.dart ColorFilterLayer.colorFilter
func (l *ColorFilterLayer) SetColorFilter(v *ui.ColorFilter) {
	if foundation.DebugMode && v == nil {
		panic(errors.New("LayerPropertyMissing", "the color filter of a layer must not be nil"))
	}
	if !v.Equal(l.colorFilter) {
		l.colorFilter = v
		l.markNeedsAddToScene()
	}
}

// AddToScene pushes the filter, adds the children and pops.
//
// Flutter: rendering/layer.dart ColorFilterLayer.addToScene
func (l *ColorFilterLayer) AddToScene(builder *ui.SceneBuilder) {
	if l.colorFilter == nil {
		panic(errors.New("LayerPropertyMissing", "a ColorFilterLayer without a filter was added to a scene"))
	}
	old, _ := l.engineLayer.(*ui.ColorFilterEngineLayer)
	l.SetEngineLayer(builder.PushColorFilter(l.colorFilter, ui.PushColorFilterOpts{OldLayer: old}))
	l.AddChildrenToScene(builder)
	builder.Pop()
}

// DebugFillProperties adds the filter.
//
// Flutter: rendering/layer.dart ColorFilterLayer.debugFillProperties
func (l *ColorFilterLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("colorFilter", l.colorFilter))
}

// ImageFilterLayer is a layer that applies an image filter to its children.
//
// Flutter: rendering/layer.dart ImageFilterLayer
type ImageFilterLayer struct {
	OffsetLayer
	imageFilter ui.ImageFilter
}

// ImageFilterLayerOpts are the optional parameters of NewImageFilterLayer.
//
// Flutter: rendering/layer.dart ImageFilterLayer.new
type ImageFilterLayerOpts struct {
	// ImageFilter is the filter; it must be set before the layer is added to a
	// scene.
	ImageFilter ui.ImageFilter
	// Offset is how far the children are moved.
	Offset ui.Offset
}

// NewImageFilterLayer creates an image filter layer.
//
// Flutter: rendering/layer.dart ImageFilterLayer.new
func NewImageFilterLayer(opts ...ImageFilterLayerOpts) *ImageFilterLayer {
	o := firstOpts(opts)
	l := &ImageFilterLayer{imageFilter: o.ImageFilter}
	InitOffsetLayer(&l.OffsetLayer, l, OffsetLayerOpts{Offset: o.Offset})
	return l
}

// ImageFilter is the filter.
//
// Flutter: rendering/layer.dart ImageFilterLayer.imageFilter
func (l *ImageFilterLayer) ImageFilter() ui.ImageFilter { return l.imageFilter }

// SetImageFilter changes the filter, which must not be nil.
//
// Flutter: rendering/layer.dart ImageFilterLayer.imageFilter
func (l *ImageFilterLayer) SetImageFilter(v ui.ImageFilter) {
	if foundation.DebugMode && v == nil {
		panic(errors.New("LayerPropertyMissing", "the image filter of a layer must not be nil"))
	}
	if l.imageFilter == nil || !v.Equal(l.imageFilter) {
		l.imageFilter = v
		l.markNeedsAddToScene()
	}
}

// AddToScene pushes the filter, adds the children and pops.
//
// Flutter: rendering/layer.dart ImageFilterLayer.addToScene
func (l *ImageFilterLayer) AddToScene(builder *ui.SceneBuilder) {
	if l.imageFilter == nil {
		panic(errors.New("LayerPropertyMissing", "an ImageFilterLayer without a filter was added to a scene"))
	}
	old, _ := l.engineLayer.(*ui.ImageFilterEngineLayer)
	l.SetEngineLayer(builder.PushImageFilter(l.imageFilter, ui.PushImageFilterOpts{Offset: l.offset, OldLayer: old}))
	l.AddChildrenToScene(builder)
	builder.Pop()
}

// DebugFillProperties adds the filter.
//
// Flutter: rendering/layer.dart ImageFilterLayer.debugFillProperties
func (l *ImageFilterLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.OffsetLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("imageFilter", l.imageFilter))
}

// TransformLayer is a layer that transforms its children by a matrix.
//
// Flutter: rendering/layer.dart TransformLayer
type TransformLayer struct {
	OffsetLayer
	transform         *ui.Matrix4
	lastEffective     *ui.Matrix4
	invertedTransform *ui.Matrix4
	inverseDirty      bool
}

// TransformLayerOpts are the optional parameters of NewTransformLayer.
//
// Flutter: rendering/layer.dart TransformLayer.new
type TransformLayerOpts struct {
	// Transform is the matrix; it must be set before the layer is added to a
	// scene. The layer keeps the pointer.
	Transform *ui.Matrix4
	// Offset is how far the children are moved, before the transform.
	Offset ui.Offset
}

// NewTransformLayer creates a transform layer.
//
// Flutter: rendering/layer.dart TransformLayer.new
func NewTransformLayer(opts ...TransformLayerOpts) *TransformLayer {
	o := firstOpts(opts)
	l := &TransformLayer{transform: o.Transform, inverseDirty: true}
	InitOffsetLayer(&l.OffsetLayer, l, OffsetLayerOpts{Offset: o.Offset})
	return l
}

// Transform is the matrix that transforms the children.
//
// Flutter: rendering/layer.dart TransformLayer.transform
func (l *TransformLayer) Transform() *ui.Matrix4 { return l.transform }

// SetTransform changes the matrix, which must be non-nil and finite.
//
// Flutter: rendering/layer.dart TransformLayer.transform
func (l *TransformLayer) SetTransform(v *ui.Matrix4) {
	if foundation.DebugMode {
		if v == nil {
			panic(errors.New("LayerPropertyMissing", "the transform of a layer must not be nil"))
		}
		for _, c := range v.Storage() {
			if math.IsInf(float64(c), 0) || math.IsNaN(float64(c)) {
				panic(errors.New("TransformLayerNotFinite", "the transform of a layer must be finite", "transform", v))
			}
		}
	}
	if v == l.transform || (l.transform != nil && v.Equals(l.transform)) {
		return
	}
	l.transform = v
	l.inverseDirty = true
	l.markNeedsAddToScene()
}

// AddToScene pushes the (offset) transform, adds the children and pops.
//
// Flutter: rendering/layer.dart TransformLayer.addToScene
func (l *TransformLayer) AddToScene(builder *ui.SceneBuilder) {
	if l.transform == nil {
		panic(errors.New("LayerPropertyMissing", "a TransformLayer without a transform was added to a scene"))
	}
	l.lastEffective = l.transform
	if l.offset != ui.OffsetZero {
		l.lastEffective = ui.NewMatrix4TranslationValues(float32(l.offset.Dx), float32(l.offset.Dy), 0).Multiply(l.lastEffective)
	}
	old, _ := l.engineLayer.(*ui.TransformEngineLayer)
	l.SetEngineLayer(builder.PushTransform(l.lastEffective, ui.PushTransformOpts{OldLayer: old}))
	l.AddChildrenToScene(builder)
	builder.Pop()
}

func (l *TransformLayer) transformOffset(localPosition ui.Offset) (ui.Offset, bool) {
	if l.inverseDirty {
		l.invertedTransform = tryInvert(gestures.RemovePerspectiveTransform(l.transform))
		l.inverseDirty = false
	}
	if l.invertedTransform == nil {
		return ui.Offset{}, false
	}
	return l.invertedTransform.TransformPoint(localPosition), true
}

// tryInvert returns the inverse of the matrix, or nil if it has none.
//
// Flutter: package:vector_math Matrix4.tryInvert
func tryInvert(m *ui.Matrix4) *ui.Matrix4 {
	inv := m.Clone()
	if inv.Invert() == 0 {
		return nil
	}
	return inv
}

// FindAnnotations searches the children at the inverse-transformed position.
//
// Flutter: rendering/layer.dart TransformLayer.findAnnotations
func (l *TransformLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	p, ok := l.transformOffset(localPosition)
	if !ok {
		return false
	}
	return l.ContainerLayer.FindAnnotations(result, p, onlyFirst)
}

// ApplyTransform multiplies the transform by the effective transform.
//
// Flutter: rendering/layer.dart TransformLayer.applyTransform
func (l *TransformLayer) ApplyTransform(child Layer, transform *ui.Matrix4) {
	if l.lastEffective == nil {
		transform.MultiplyBy(l.transform)
	} else {
		transform.MultiplyBy(l.lastEffective)
	}
}

// DebugFillProperties adds the transform.
//
// Flutter: rendering/layer.dart TransformLayer.debugFillProperties
func (l *TransformLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.OffsetLayer.DebugFillProperties(p)
	p.Add(painting.NewTransformProperty("transform", l.transform))
}

// OpacityLayer is a layer that makes its children partly transparent.
//
// Flutter: rendering/layer.dart OpacityLayer
type OpacityLayer struct {
	OffsetLayer
	alpha foundation.Opt[int]
}

// OpacityLayerOpts are the optional parameters of NewOpacityLayer.
//
// Flutter: rendering/layer.dart OpacityLayer.new
type OpacityLayerOpts struct {
	// Alpha is the opacity, 0 to 255; it must be set before the layer is added
	// to a scene.
	Alpha foundation.Opt[int]
	// Offset is how far the children are moved.
	Offset ui.Offset
}

// NewOpacityLayer creates an opacity layer.
//
// Flutter: rendering/layer.dart OpacityLayer.new
func NewOpacityLayer(opts ...OpacityLayerOpts) *OpacityLayer {
	o := firstOpts(opts)
	l := &OpacityLayer{alpha: o.Alpha}
	InitOffsetLayer(&l.OffsetLayer, l, OffsetLayerOpts{Offset: o.Offset})
	return l
}

// Alpha is the opacity, 0 to 255.
//
// Flutter: rendering/layer.dart OpacityLayer.alpha
func (l *OpacityLayer) Alpha() foundation.Opt[int] { return l.alpha }

// SetAlpha changes the opacity. The engine layer is dropped when the alpha
// becomes or stops being 255, as the kind of layer pushed changes.
//
// Flutter: rendering/layer.dart OpacityLayer.alpha
func (l *OpacityLayer) SetAlpha(v int) {
	if cur, ok := l.alpha.Get(); !ok || cur != v {
		if v == 255 || (ok && cur == 255) {
			l.SetEngineLayer(nil)
		}
		l.alpha = foundation.Some(v)
		l.markNeedsAddToScene()
	}
}

// AddToScene pushes an opacity (or only an offset when it is opaque), adds the
// children and pops. A layer without children adds nothing.
//
// Flutter: rendering/layer.dart OpacityLayer.addToScene
func (l *OpacityLayer) AddToScene(builder *ui.SceneBuilder) {
	alpha, ok := l.alpha.Get()
	if !ok {
		panic(errors.New("LayerPropertyMissing", "an OpacityLayer without an alpha was added to a scene"))
	}
	enabled := l.firstChild != nil
	if !enabled {
		l.SetEngineLayer(nil)
		return
	}
	if foundation.DebugMode && l.debugOptions().DisableOpacityLayers {
		enabled = false
	}
	if enabled && alpha < 255 {
		old, _ := l.engineLayer.(*ui.OpacityEngineLayer)
		l.SetEngineLayer(builder.PushOpacity(alpha, ui.PushOpacityOpts{Offset: l.offset, OldLayer: old}))
	} else {
		old, _ := l.engineLayer.(*ui.OffsetEngineLayer)
		l.SetEngineLayer(builder.PushOffset(l.offset.Dx, l.offset.Dy, ui.PushOffsetOpts{OldLayer: old}))
	}
	l.AddChildrenToScene(builder)
	builder.Pop()
}

// DebugFillProperties adds the alpha.
//
// Flutter: rendering/layer.dart OpacityLayer.debugFillProperties
func (l *OpacityLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.OffsetLayer.DebugFillProperties(p)
	p.Add(foundation.NewIntProperty("alpha", l.alpha))
}

// ShaderMaskLayer is a layer that applies a shader as a mask to its children.
//
// Flutter: rendering/layer.dart ShaderMaskLayer
type ShaderMaskLayer struct {
	ContainerLayer
	shader    ui.Shader
	maskRect  foundation.Opt[ui.Rect]
	blendMode foundation.Opt[ui.BlendMode]
}

// ShaderMaskLayerOpts are the optional parameters of NewShaderMaskLayer.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.new
type ShaderMaskLayerOpts struct {
	// Shader is the mask; it must be set before the layer is added to a scene.
	Shader ui.Shader
	// MaskRect is the rectangle the shader covers.
	MaskRect foundation.Opt[ui.Rect]
	// BlendMode combines the shader with the children.
	BlendMode foundation.Opt[ui.BlendMode]
}

// NewShaderMaskLayer creates a shader mask layer.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.new
func NewShaderMaskLayer(opts ...ShaderMaskLayerOpts) *ShaderMaskLayer {
	o := firstOpts(opts)
	l := &ShaderMaskLayer{shader: o.Shader, maskRect: o.MaskRect, blendMode: o.BlendMode}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// Shader is the mask shader.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.shader
func (l *ShaderMaskLayer) Shader() ui.Shader { return l.shader }

// SetShader changes the shader.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.shader
func (l *ShaderMaskLayer) SetShader(v ui.Shader) {
	if v != l.shader {
		l.shader = v
		l.markNeedsAddToScene()
	}
}

// MaskRect is the rectangle the shader covers.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.maskRect
func (l *ShaderMaskLayer) MaskRect() foundation.Opt[ui.Rect] { return l.maskRect }

// SetMaskRect changes the rectangle.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.maskRect
func (l *ShaderMaskLayer) SetMaskRect(v ui.Rect) {
	if cur, ok := l.maskRect.Get(); !ok || cur != v {
		l.maskRect = foundation.Some(v)
		l.markNeedsAddToScene()
	}
}

// BlendMode combines the shader with the children.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.blendMode
func (l *ShaderMaskLayer) BlendMode() foundation.Opt[ui.BlendMode] { return l.blendMode }

// SetBlendMode changes the blend mode.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.blendMode
func (l *ShaderMaskLayer) SetBlendMode(v ui.BlendMode) {
	if cur, ok := l.blendMode.Get(); !ok || cur != v {
		l.blendMode = foundation.Some(v)
		l.markNeedsAddToScene()
	}
}

// AddToScene pushes the mask, adds the children and pops.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.addToScene
func (l *ShaderMaskLayer) AddToScene(builder *ui.SceneBuilder) {
	rect, okR := l.maskRect.Get()
	mode, okM := l.blendMode.Get()
	if l.shader == nil || !okR || !okM {
		panic(errors.New("LayerPropertyMissing", "a ShaderMaskLayer needs a shader, a mask rectangle and a blend mode"))
	}
	old, _ := l.engineLayer.(*ui.ShaderMaskEngineLayer)
	l.SetEngineLayer(builder.PushShaderMask(l.shader, rect, mode, ui.PushShaderMaskOpts{OldLayer: old}))
	l.AddChildrenToScene(builder)
	builder.Pop()
}

// DebugFillProperties adds the shader, the rectangle and the blend mode.
//
// Flutter: rendering/layer.dart ShaderMaskLayer.debugFillProperties
func (l *ShaderMaskLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("shader", l.shader))
	p.Add(foundation.NewDiagnosticsPropertyOpt("maskRect", l.maskRect))
	p.Add(foundation.NewEnumProperty("blendMode", l.blendMode))
}

// BackdropFilterLayer is a layer that applies an image filter to what is
// painted behind it.
//
// Flutter: rendering/layer.dart BackdropFilterLayer
type BackdropFilterLayer struct {
	ContainerLayer
	filter      ui.ImageFilter
	blendMode   ui.BlendMode
	backdropKey foundation.Opt[BackdropKey]
}

// BackdropFilterLayerOpts are the optional parameters of
// NewBackdropFilterLayer.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.new
type BackdropFilterLayerOpts struct {
	// Filter is the filter; it must be set before the layer is added to a scene.
	Filter ui.ImageFilter
	// BlendMode defaults to BlendModeSrcOver.
	BlendMode ui.BlendMode
}

// NewBackdropFilterLayer creates a backdrop filter layer.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.new
func NewBackdropFilterLayer(opts ...BackdropFilterLayerOpts) *BackdropFilterLayer {
	o := firstOpts(opts)
	if o.BlendMode == ui.BlendModeUnspecified {
		o.BlendMode = ui.BlendModeSrcOver
	}
	l := &BackdropFilterLayer{filter: o.Filter, blendMode: o.BlendMode}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// Filter is the filter.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.filter
func (l *BackdropFilterLayer) Filter() ui.ImageFilter { return l.filter }

// SetFilter changes the filter.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.filter
func (l *BackdropFilterLayer) SetFilter(v ui.ImageFilter) {
	if (v == nil) != (l.filter == nil) || (v != nil && !v.Equal(l.filter)) {
		l.filter = v
		l.markNeedsAddToScene()
	}
}

// BlendMode is how the filtered backdrop combines with what is there.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.blendMode
func (l *BackdropFilterLayer) BlendMode() ui.BlendMode { return l.blendMode }

// SetBlendMode changes the blend mode.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.blendMode
func (l *BackdropFilterLayer) SetBlendMode(v ui.BlendMode) {
	if v != l.blendMode {
		l.blendMode = v
		l.markNeedsAddToScene()
	}
}

// BackdropKey is the key that shares the backdrop with other filters.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.backdropKey
func (l *BackdropFilterLayer) BackdropKey() foundation.Opt[BackdropKey] { return l.backdropKey }

// SetBackdropKey changes the key.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.backdropKey
func (l *BackdropFilterLayer) SetBackdropKey(v foundation.Opt[BackdropKey]) {
	if v != l.backdropKey {
		l.backdropKey = v
		l.markNeedsAddToScene()
	}
}

// AddToScene pushes the filter, adds the children and pops.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.addToScene
func (l *BackdropFilterLayer) AddToScene(builder *ui.SceneBuilder) {
	if l.filter == nil {
		panic(errors.New("LayerPropertyMissing", "a BackdropFilterLayer without a filter was added to a scene"))
	}
	old, _ := l.engineLayer.(*ui.BackdropFilterEngineLayer)
	o := ui.PushBackdropFilterOpts{BlendMode: l.blendMode, OldLayer: old}
	if k, ok := l.backdropKey.Get(); ok {
		id := k.key
		o.BackdropID = &id
	}
	l.SetEngineLayer(builder.PushBackdropFilter(l.filter, o))
	l.AddChildrenToScene(builder)
	builder.Pop()
}

// DebugFillProperties adds the filter, the blend mode and the key.
//
// Flutter: rendering/layer.dart BackdropFilterLayer.debugFillProperties
func (l *BackdropFilterLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("filter", l.filter))
	p.Add(foundation.NewEnumProperty("blendMode", foundation.Some(l.blendMode)))
	var key foundation.Opt[int]
	if k, ok := l.backdropKey.Get(); ok {
		key = foundation.Some(k.key)
	}
	p.Add(foundation.NewIntProperty("backdropKey", key))
}

// AnnotatedRegionLayer is a layer that marks a region of the layer tree with a
// value that Layer.find can look up.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer
type AnnotatedRegionLayer struct {
	ContainerLayer
	value  any
	size   foundation.Opt[ui.Size]
	offset ui.Offset
	opaque bool
}

// AnnotatedRegionLayerOpts are the optional parameters of
// NewAnnotatedRegionLayer.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.new
type AnnotatedRegionLayerOpts struct {
	// Size limits the region to a rectangle of this size at Offset; unset
	// means everything.
	Size foundation.Opt[ui.Size]
	// Offset is the top left of the region.
	Offset ui.Offset
	// Opaque makes the region absorb the search: layers under it are not
	// searched.
	Opaque bool
}

// NewAnnotatedRegionLayer creates a layer that annotates its region with the
// value.
//
// Deviation: Flutter's is generic (AnnotatedRegionLayer<T>) and matches
// annotations whose type is exactly the searched one; this one holds an any
// and matches the types the search accepts (SPEC §12.10).
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.new
func NewAnnotatedRegionLayer(value any, opts ...AnnotatedRegionLayerOpts) *AnnotatedRegionLayer {
	o := firstOpts(opts)
	l := &AnnotatedRegionLayer{value: value, size: o.Size, offset: o.Offset, opaque: o.Opaque}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// Value is the annotation.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.value
func (l *AnnotatedRegionLayer) Value() any { return l.value }

// Size is the size of the region, if limited.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.size
func (l *AnnotatedRegionLayer) Size() foundation.Opt[ui.Size] { return l.size }

// Offset is the top left of the region.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.offset
func (l *AnnotatedRegionLayer) Offset() ui.Offset { return l.offset }

// Opaque reports whether the region absorbs the search.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.opaque
func (l *AnnotatedRegionLayer) Opaque() bool { return l.opaque }

// FindAnnotations searches the children, then adds its own value if the
// position is inside the region and the result accepts the value.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.findAnnotations
func (l *AnnotatedRegionLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	isAbsorbed := l.ContainerLayer.FindAnnotations(result, localPosition, onlyFirst)
	if len(result.entries) > 0 && onlyFirst {
		return isAbsorbed
	}
	if size, ok := l.size.Get(); ok && !l.offset.And(size).Contains(localPosition) {
		return isAbsorbed
	}
	if result.Accepts(l.value) {
		isAbsorbed = isAbsorbed || l.opaque
		result.Add(AnnotationEntry{Annotation: l.value, LocalPosition: localPosition.Sub(l.offset)})
	}
	return isAbsorbed
}

// DebugFillProperties adds the value, the size, the offset and opaque.
//
// Flutter: rendering/layer.dart AnnotatedRegionLayer.debugFillProperties
func (l *AnnotatedRegionLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("value", l.value))
	p.Add(foundation.NewDiagnosticsPropertyOpt("size", l.size, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewDiagnosticsProperty("offset", l.offset, foundation.DiagnosticsPropertyOpts{DefaultValue: ui.OffsetZero}))
	p.Add(foundation.NewDiagnosticsProperty("opaque", l.opaque, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
}
