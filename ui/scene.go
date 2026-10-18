package ui

import "github.com/rm4n0s/errors"

// Layer is a node of the tree of a Scene. A scene is a tree of layers: container
// layers that apply an effect (a transform, a clip, an opacity, a filter) to the
// layers under them, and leaf layers that draw a Picture. The renderers walk the
// tree and pick the concrete layer with a type switch. A layer is immutable once
// the scene is built.
//
// Flutter: no counterpart (flutter::Layer)
type Layer interface {
	// Children returns the layers under this one, in painting order. The slice
	// is the layer's own: callers must not change it.
	Children() []Layer
	// OldLayer returns the layer of the previous frame this one replaces, when
	// the app said so with an oldLayer argument, or nil. Renderers may reuse
	// what they cached for it.
	OldLayer() Layer

	layer() *layerBase
}

// layerBase holds what every layer has.
type layerBase struct {
	children []Layer
	old      Layer
}

func (b *layerBase) layer() *layerBase { return b }

// Children returns the layers under this one.
//
// Flutter: no counterpart
func (b *layerBase) Children() []Layer { return b.children }

// OldLayer returns the layer of the previous frame this one replaces.
//
// Flutter: no counterpart
func (b *layerBase) OldLayer() Layer { return b.old }

// ContainerLayer groups layers and applies nothing. The root of a scene is one.
//
// Flutter: flow/layers/container_layer.h ContainerLayer
type ContainerLayer struct{ layerBase }

// TransformLayer applies a transform to the layers under it. PushOffset makes
// one too, with a translation.
//
// Flutter: flow/layers/transform_layer.h TransformLayer
type TransformLayer struct {
	layerBase
	// Matrix is the transform applied to the children.
	Matrix Matrix4
}

// ClipRectLayer clips the layers under it to a rectangle.
//
// Flutter: flow/layers/clip_rect_layer.h ClipRectLayer
type ClipRectLayer struct {
	layerBase
	// Rect is the clip shape.
	Rect Rect
	// ClipBehavior says how the edge is clipped; never ClipUnspecified.
	ClipBehavior Clip
}

// ClipRRectLayer clips the layers under it to a rounded rectangle.
//
// Flutter: flow/layers/clip_rrect_layer.h ClipRRectLayer
type ClipRRectLayer struct {
	layerBase
	// RRect is the clip shape.
	RRect RRect
	// ClipBehavior says how the edge is clipped; never ClipUnspecified.
	ClipBehavior Clip
}

// ClipRSuperellipseLayer clips the layers under it to a rounded superellipse.
//
// Flutter: flow/layers/clip_rsuperellipse_layer.h ClipRSuperellipseLayer
type ClipRSuperellipseLayer struct {
	layerBase
	// RSuperellipse is the clip shape.
	RSuperellipse RSuperellipse
	// ClipBehavior says how the edge is clipped; never ClipUnspecified.
	ClipBehavior Clip
}

// ClipPathLayer clips the layers under it to a path.
//
// Flutter: flow/layers/clip_path_layer.h ClipPathLayer
type ClipPathLayer struct {
	layerBase
	// Path is the clip shape: a copy of the path that was given.
	Path *Path
	// ClipBehavior says how the edge is clipped; never ClipUnspecified.
	ClipBehavior Clip
}

// OpacityLayer draws the layers under it into a group and composites the group
// with an alpha.
//
// Flutter: flow/layers/opacity_layer.h OpacityLayer
type OpacityLayer struct {
	layerBase
	// Alpha is the opacity, from 0 (transparent) to 255 (opaque).
	Alpha int
	// Offset moves the children, before the alpha is applied.
	Offset Offset
}

// ColorFilterLayer applies a color filter to the layers under it.
//
// Flutter: flow/layers/color_filter_layer.h ColorFilterLayer
type ColorFilterLayer struct {
	layerBase
	// Filter is the color filter.
	Filter *ColorFilter
}

// ImageFilterLayer applies an image filter to the layers under it.
//
// Flutter: flow/layers/image_filter_layer.h ImageFilterLayer
type ImageFilterLayer struct {
	layerBase
	// Filter is the image filter.
	Filter ImageFilter
	// Offset moves the children, before the filter is applied.
	Offset Offset
}

// BackdropFilterLayer applies an image filter to what was drawn before it, within
// its clip, and then draws the layers under it.
//
// Flutter: flow/layers/backdrop_filter_layer.h BackdropFilterLayer
type BackdropFilterLayer struct {
	layerBase
	// Filter is the image filter applied to the backdrop.
	Filter ImageFilter
	// BlendMode is how the filtered backdrop combines with what is there.
	BlendMode BlendMode
	// BackdropID, when HasBackdropID is true, names a backdrop shared by several
	// backdrop filters, which are then computed once.
	BackdropID    int
	HasBackdropID bool
}

// ShaderMaskLayer draws the layers under it into a group and then paints a
// shader over the group with a blend mode, within a rectangle.
//
// Flutter: flow/layers/shader_mask_layer.h ShaderMaskLayer
type ShaderMaskLayer struct {
	layerBase
	// Shader is the shader that makes the mask.
	Shader Shader
	// MaskRect is the area the shader is painted over.
	MaskRect Rect
	// BlendMode is how the shader combines with the group.
	BlendMode BlendMode
	// FilterQuality is the quality of sampling an image shader; never
	// FilterQualityUnspecified.
	FilterQuality FilterQuality
}

// PictureLayer draws a picture at an offset.
//
// Flutter: flow/layers/display_list_layer.h DisplayListLayer
type PictureLayer struct {
	layerBase
	// Offset is where the origin of the picture goes.
	Offset Offset
	// Picture is the picture to draw.
	Picture *Picture
	// IsComplexHint says that the picture is expensive to draw, so a renderer is
	// advised to cache it.
	IsComplexHint bool
	// WillChangeHint says that the picture is likely to change, so a renderer is
	// advised not to cache it.
	WillChangeHint bool
}

// Scene is a composited scene: a tree of layers built by a SceneBuilder and
// ready to be rendered. It is immutable.
//
// Flutter: ui/compositing.dart Scene
type Scene struct {
	root *ContainerLayer
}

// Root returns the layer at the top of the tree.
//
// Flutter: no counterpart
func (s *Scene) Root() *ContainerLayer { return s.root }

// ToImage draws the scene into an image of the given size in pixels.
//
// TODO(step 2.10): until then it returns the error NotImplemented.
//
// Deviation: Flutter's returns a Future; this one is synchronous.
//
// Flutter: ui/compositing.dart Scene.toImage
func (s *Scene) ToImage(width, height int) (*Image, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("ImageSizeInvalid", "Invalid image dimensions.", "width", width, "height", height)
	}
	return nil, errors.New("NotImplemented", "Scene.ToImage needs the software renderer's images (step 2.10)")
}

// ToImageSync is ToImage.
//
// Flutter: ui/compositing.dart Scene.toImageSync
func (s *Scene) ToImageSync(width, height int) (*Image, error) { return s.ToImage(width, height) }

// Dispose releases the scene. It does nothing in this implementation, which
// holds no native resources.
//
// Flutter: ui/compositing.dart Scene.dispose
func (s *Scene) Dispose() {}

// String describes the scene, as Dart's toString does.
//
// Flutter: ui/compositing.dart Scene.toString
func (s *Scene) String() string { return "Scene" }

// EngineLayer is a handle to a layer a SceneBuilder pushed. Keep it, and pass it
// as the OldLayer of the same kind of push in the next frame, to say that the new
// layer replaces it; or pass it to AddRetained to put the same layer in another
// scene without building it again.
//
// Flutter: ui/painting.dart EngineLayer
type EngineLayer interface {
	// Dispose releases the handle. A disposed layer cannot be added with
	// AddRetained.
	Dispose()

	handle() *engineLayerHandle
}

// engineLayerHandle is what the handle types share.
type engineLayerHandle struct {
	layer    Layer
	disposed bool
}

func (h *engineLayerHandle) handle() *engineLayerHandle { return h }

// Dispose releases the handle.
//
// Flutter: ui/compositing.dart _EngineLayerWrapper.dispose
func (h *engineLayerHandle) Dispose() { h.disposed = true }

// Layer returns the layer the handle stands for.
//
// Flutter: no counterpart
func (h *engineLayerHandle) Layer() Layer { return h.layer }

// The handle types, one for each kind of push, so that a layer of one kind cannot
// be passed as the old layer of another.

// TransformEngineLayer is the handle PushTransform returns.
//
// Flutter: ui/compositing.dart TransformEngineLayer
type TransformEngineLayer struct{ engineLayerHandle }

// OffsetEngineLayer is the handle PushOffset returns.
//
// Flutter: ui/compositing.dart OffsetEngineLayer
type OffsetEngineLayer struct{ engineLayerHandle }

// ClipRectEngineLayer is the handle PushClipRect returns.
//
// Flutter: ui/compositing.dart ClipRectEngineLayer
type ClipRectEngineLayer struct{ engineLayerHandle }

// ClipRRectEngineLayer is the handle PushClipRRect returns.
//
// Flutter: ui/compositing.dart ClipRRectEngineLayer
type ClipRRectEngineLayer struct{ engineLayerHandle }

// ClipRSuperellipseEngineLayer is the handle PushClipRSuperellipse returns.
//
// Flutter: ui/compositing.dart ClipRSuperellipseEngineLayer
type ClipRSuperellipseEngineLayer struct{ engineLayerHandle }

// ClipPathEngineLayer is the handle PushClipPath returns.
//
// Flutter: ui/compositing.dart ClipPathEngineLayer
type ClipPathEngineLayer struct{ engineLayerHandle }

// OpacityEngineLayer is the handle PushOpacity returns.
//
// Flutter: ui/compositing.dart OpacityEngineLayer
type OpacityEngineLayer struct{ engineLayerHandle }

// ColorFilterEngineLayer is the handle PushColorFilter returns.
//
// Flutter: ui/compositing.dart ColorFilterEngineLayer
type ColorFilterEngineLayer struct{ engineLayerHandle }

// ImageFilterEngineLayer is the handle PushImageFilter returns.
//
// Flutter: ui/compositing.dart ImageFilterEngineLayer
type ImageFilterEngineLayer struct{ engineLayerHandle }

// BackdropFilterEngineLayer is the handle PushBackdropFilter returns.
//
// Flutter: ui/compositing.dart BackdropFilterEngineLayer
type BackdropFilterEngineLayer struct{ engineLayerHandle }

// ShaderMaskEngineLayer is the handle PushShaderMask returns.
//
// Flutter: ui/compositing.dart ShaderMaskEngineLayer
type ShaderMaskEngineLayer struct{ engineLayerHandle }

// SceneBuilder builds a Scene: a tree of layers. Push methods start a layer that
// applies an effect to everything added until the matching Pop, and AddPicture
// adds a picture. Build ends the building and returns the scene.
//
// Flutter: ui/compositing.dart SceneBuilder
type SceneBuilder struct {
	stack []Layer
	built bool
}

// NewSceneBuilder creates a builder with an empty scene, which has a container
// layer as its root.
//
// Flutter: ui/compositing.dart SceneBuilder
func NewSceneBuilder() *SceneBuilder {
	return &SceneBuilder{stack: []Layer{&ContainerLayer{}}}
}

// add puts a layer under the current one.
func (b *SceneBuilder) add(l Layer) {
	if b.built {
		panic(errors.New("SceneBuilderBuilt", "The SceneBuilder was already used to build a Scene."))
	}
	top := b.stack[len(b.stack)-1].layer()
	top.children = append(top.children, l)
}

// push puts a layer under the current one and makes it the current one.
func (b *SceneBuilder) push(l Layer, old EngineLayer) {
	if old != nil {
		l.layer().old = old.handle().layer
	}
	b.add(l)
	b.stack = append(b.stack, l)
}

// oldOf returns the handle of an optional old layer as the interface, nil if the
// pointer is nil.
func oldOf[T interface {
	EngineLayer
	comparable
}](old T) EngineLayer {
	var zero T
	if old == zero {
		return nil
	}
	return old
}

// clipOrDefault turns the unspecified clip behavior into Flutter's default, antiAlias.
func clipOrDefault(c Clip) Clip {
	if c == ClipUnspecified {
		return ClipAntiAlias
	}
	return c
}

// PushTransformOpts are the optional arguments of PushTransform.
//
// Flutter: ui/compositing.dart SceneBuilder.pushTransform
type PushTransformOpts struct {
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *TransformEngineLayer
}

// PushTransform pushes a transform operation onto the operation stack. The
// matrix is copied. Everything added until the matching Pop is transformed.
//
// Flutter: ui/compositing.dart SceneBuilder.pushTransform
func (b *SceneBuilder) PushTransform(matrix *Matrix4, opts ...PushTransformOpts) *TransformEngineLayer {
	o := oneOpts(opts)
	l := &TransformLayer{Matrix: *matrix}
	b.push(l, oldOf(o.OldLayer))
	return &TransformEngineLayer{engineLayerHandle{layer: l}}
}

// PushOffsetOpts are the optional arguments of PushOffset.
//
// Flutter: ui/compositing.dart SceneBuilder.pushOffset
type PushOffsetOpts struct {
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *OffsetEngineLayer
}

// PushOffset pushes a translation onto the operation stack.
//
// Flutter: ui/compositing.dart SceneBuilder.pushOffset
func (b *SceneBuilder) PushOffset(dx, dy float64, opts ...PushOffsetOpts) *OffsetEngineLayer {
	o := oneOpts(opts)
	l := &TransformLayer{Matrix: *NewMatrix4TranslationValues(float32(dx), float32(dy), 0)}
	b.push(l, oldOf(o.OldLayer))
	return &OffsetEngineLayer{engineLayerHandle{layer: l}}
}

// PushClipRectOpts are the optional arguments of PushClipRect.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipRect
type PushClipRectOpts struct {
	// ClipBehavior says how the edge is clipped; the zero value is
	// ClipAntiAlias, Flutter's default.
	ClipBehavior Clip
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *ClipRectEngineLayer
}

// PushClipRect pushes a rectangular clip operation onto the operation stack.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipRect
func (b *SceneBuilder) PushClipRect(rect Rect, opts ...PushClipRectOpts) *ClipRectEngineLayer {
	o := oneOpts(opts)
	l := &ClipRectLayer{Rect: rect, ClipBehavior: clipOrDefault(o.ClipBehavior)}
	b.push(l, oldOf(o.OldLayer))
	return &ClipRectEngineLayer{engineLayerHandle{layer: l}}
}

// PushClipRRectOpts are the optional arguments of PushClipRRect.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipRRect
type PushClipRRectOpts struct {
	// ClipBehavior says how the edge is clipped; the zero value is
	// ClipAntiAlias, Flutter's default.
	ClipBehavior Clip
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *ClipRRectEngineLayer
}

// PushClipRRect pushes a rounded-rectangular clip operation onto the operation
// stack.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipRRect
func (b *SceneBuilder) PushClipRRect(rrect RRect, opts ...PushClipRRectOpts) *ClipRRectEngineLayer {
	o := oneOpts(opts)
	l := &ClipRRectLayer{RRect: rrect, ClipBehavior: clipOrDefault(o.ClipBehavior)}
	b.push(l, oldOf(o.OldLayer))
	return &ClipRRectEngineLayer{engineLayerHandle{layer: l}}
}

// PushClipRSuperellipseOpts are the optional arguments of PushClipRSuperellipse.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipRSuperellipse
type PushClipRSuperellipseOpts struct {
	// ClipBehavior says how the edge is clipped; the zero value is
	// ClipAntiAlias, Flutter's default.
	ClipBehavior Clip
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *ClipRSuperellipseEngineLayer
}

// PushClipRSuperellipse pushes a rounded-superellipse clip operation onto the
// operation stack.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipRSuperellipse
func (b *SceneBuilder) PushClipRSuperellipse(rs RSuperellipse, opts ...PushClipRSuperellipseOpts) *ClipRSuperellipseEngineLayer {
	o := oneOpts(opts)
	l := &ClipRSuperellipseLayer{RSuperellipse: rs, ClipBehavior: clipOrDefault(o.ClipBehavior)}
	b.push(l, oldOf(o.OldLayer))
	return &ClipRSuperellipseEngineLayer{engineLayerHandle{layer: l}}
}

// PushClipPathOpts are the optional arguments of PushClipPath.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipPath
type PushClipPathOpts struct {
	// ClipBehavior says how the edge is clipped; the zero value is
	// ClipAntiAlias, Flutter's default.
	ClipBehavior Clip
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *ClipPathEngineLayer
}

// PushClipPath pushes a path clip operation onto the operation stack. The path is
// copied.
//
// Flutter: ui/compositing.dart SceneBuilder.pushClipPath
func (b *SceneBuilder) PushClipPath(path *Path, opts ...PushClipPathOpts) *ClipPathEngineLayer {
	o := oneOpts(opts)
	l := &ClipPathLayer{Path: NewPathFrom(path), ClipBehavior: clipOrDefault(o.ClipBehavior)}
	b.push(l, oldOf(o.OldLayer))
	return &ClipPathEngineLayer{engineLayerHandle{layer: l}}
}

// PushOpacityOpts are the optional arguments of PushOpacity.
//
// Flutter: ui/compositing.dart SceneBuilder.pushOpacity
type PushOpacityOpts struct {
	// Offset moves the children before the alpha is applied.
	Offset Offset
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *OpacityEngineLayer
}

// PushOpacity pushes an opacity operation onto the operation stack. The alpha is
// from 0 (transparent) to 255 (opaque).
//
// Flutter: ui/compositing.dart SceneBuilder.pushOpacity
func (b *SceneBuilder) PushOpacity(alpha int, opts ...PushOpacityOpts) *OpacityEngineLayer {
	o := oneOpts(opts)
	l := &OpacityLayer{Alpha: alpha, Offset: o.Offset}
	b.push(l, oldOf(o.OldLayer))
	return &OpacityEngineLayer{engineLayerHandle{layer: l}}
}

// PushColorFilterOpts are the optional arguments of PushColorFilter.
//
// Flutter: ui/compositing.dart SceneBuilder.pushColorFilter
type PushColorFilterOpts struct {
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *ColorFilterEngineLayer
}

// PushColorFilter pushes a color filter operation onto the operation stack.
//
// Flutter: ui/compositing.dart SceneBuilder.pushColorFilter
func (b *SceneBuilder) PushColorFilter(filter *ColorFilter, opts ...PushColorFilterOpts) *ColorFilterEngineLayer {
	o := oneOpts(opts)
	l := &ColorFilterLayer{Filter: filter}
	b.push(l, oldOf(o.OldLayer))
	return &ColorFilterEngineLayer{engineLayerHandle{layer: l}}
}

// PushImageFilterOpts are the optional arguments of PushImageFilter.
//
// Flutter: ui/compositing.dart SceneBuilder.pushImageFilter
type PushImageFilterOpts struct {
	// Offset moves the children before the filter is applied.
	Offset Offset
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *ImageFilterEngineLayer
}

// PushImageFilter pushes an image filter operation onto the operation stack.
//
// Flutter: ui/compositing.dart SceneBuilder.pushImageFilter
func (b *SceneBuilder) PushImageFilter(filter ImageFilter, opts ...PushImageFilterOpts) *ImageFilterEngineLayer {
	o := oneOpts(opts)
	l := &ImageFilterLayer{Filter: filter, Offset: o.Offset}
	b.push(l, oldOf(o.OldLayer))
	return &ImageFilterEngineLayer{engineLayerHandle{layer: l}}
}

// PushBackdropFilterOpts are the optional arguments of PushBackdropFilter.
//
// Flutter: ui/compositing.dart SceneBuilder.pushBackdropFilter
type PushBackdropFilterOpts struct {
	// BlendMode is how the filtered backdrop combines with what is there; the
	// zero value is BlendModeSrcOver, Flutter's default.
	BlendMode BlendMode
	// BackdropID, when not nil, names a backdrop that several filters share.
	BackdropID *int
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *BackdropFilterEngineLayer
}

// PushBackdropFilter pushes a backdrop filter operation onto the operation stack:
// the filter is applied to what was drawn before, inside the current clip, and
// then the children are drawn on top.
//
// Flutter: ui/compositing.dart SceneBuilder.pushBackdropFilter
func (b *SceneBuilder) PushBackdropFilter(filter ImageFilter, opts ...PushBackdropFilterOpts) *BackdropFilterEngineLayer {
	o := oneOpts(opts)
	mode := o.BlendMode
	if mode == BlendModeUnspecified {
		mode = BlendModeSrcOver
	}
	l := &BackdropFilterLayer{Filter: filter, BlendMode: mode}
	if o.BackdropID != nil {
		l.BackdropID, l.HasBackdropID = *o.BackdropID, true
	}
	b.push(l, oldOf(o.OldLayer))
	return &BackdropFilterEngineLayer{engineLayerHandle{layer: l}}
}

// PushShaderMaskOpts are the optional arguments of PushShaderMask.
//
// Flutter: ui/compositing.dart SceneBuilder.pushShaderMask
type PushShaderMaskOpts struct {
	// FilterQuality is the quality of sampling an image shader; the zero value is
	// FilterQualityLow, Flutter's default.
	FilterQuality FilterQuality
	// OldLayer is the layer of the previous frame this one replaces.
	OldLayer *ShaderMaskEngineLayer
}

// PushShaderMask pushes a shader mask operation onto the operation stack: the
// children are drawn into a group, and the shader is painted over the group with
// the blend mode, inside maskRect.
//
// Flutter: ui/compositing.dart SceneBuilder.pushShaderMask
func (b *SceneBuilder) PushShaderMask(shader Shader, maskRect Rect, blendMode BlendMode, opts ...PushShaderMaskOpts) *ShaderMaskEngineLayer {
	o := oneOpts(opts)
	q := o.FilterQuality
	if q == FilterQualityUnspecified {
		q = FilterQualityLow
	}
	l := &ShaderMaskLayer{Shader: shader, MaskRect: maskRect, BlendMode: blendMode, FilterQuality: q}
	b.push(l, oldOf(o.OldLayer))
	return &ShaderMaskEngineLayer{engineLayerHandle{layer: l}}
}

// Pop ends the effect of the last push operation. Nothing happens when only the
// root is left.
//
// Flutter: ui/compositing.dart SceneBuilder.pop
func (b *SceneBuilder) Pop() {
	if b.built {
		panic(errors.New("SceneBuilderBuilt", "The SceneBuilder was already used to build a Scene."))
	}
	if len(b.stack) > 1 {
		b.stack = b.stack[:len(b.stack)-1]
	}
}

// AddRetained adds a retained layer to the scene: the same layer a previous
// push returned, with everything under it, without building it again. The layer
// must not have been disposed; it panics with the tag EngineLayerDisposed if it
// was.
//
// Flutter: ui/compositing.dart SceneBuilder.addRetained
func (b *SceneBuilder) AddRetained(retained EngineLayer) {
	h := retained.handle()
	if h.disposed {
		panic(errors.New("EngineLayerDisposed", "Object disposed"))
	}
	b.add(h.layer)
}

// AddPictureOpts are the optional arguments of AddPicture.
//
// Flutter: ui/compositing.dart SceneBuilder.addPicture
type AddPictureOpts struct {
	// IsComplexHint says that the picture is expensive to draw, so it may be
	// worth caching.
	IsComplexHint bool
	// WillChangeHint says that the picture is likely to change, so it should not
	// be cached.
	WillChangeHint bool
}

// AddPicture adds a picture to the scene, with its origin at the offset.
//
// Flutter: ui/compositing.dart SceneBuilder.addPicture
func (b *SceneBuilder) AddPicture(offset Offset, picture *Picture, opts ...AddPictureOpts) {
	o := oneOpts(opts)
	if picture == nil {
		return
	}
	b.add(&PictureLayer{Offset: offset, Picture: picture, IsComplexHint: o.IsComplexHint, WillChangeHint: o.WillChangeHint})
}

// Build finishes the scene and returns it. The layers that are still open are
// closed. The builder cannot be used again: its methods panic with the tag
// SceneBuilderBuilt.
//
// Flutter: ui/compositing.dart SceneBuilder.build
func (b *SceneBuilder) Build() *Scene {
	if b.built {
		panic(errors.New("SceneBuilderBuilt", "The SceneBuilder was already used to build a Scene."))
	}
	b.built = true
	root := b.stack[0].(*ContainerLayer)
	b.stack = nil
	return &Scene{root: root}
}

// String describes the builder, as Dart's toString does.
//
// Flutter: ui/compositing.dart SceneBuilder.toString
func (b *SceneBuilder) String() string { return "SceneBuilder" }
