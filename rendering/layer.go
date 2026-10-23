package rendering

import (
	"fmt"
	"sync/atomic"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// AnnotationEntry is an annotation found by Layer.FindAnnotations and the
// position it was found at, in the coordinates of the annotation's layer.
//
// Flutter: rendering/layer.dart AnnotationEntry
type AnnotationEntry struct {
	// Annotation is the annotation of the layer.
	//
	// Flutter: rendering/layer.dart AnnotationEntry.annotation
	Annotation any
	// LocalPosition is the position that was searched for, in the coordinates
	// of the layer the annotation belongs to.
	//
	// Flutter: rendering/layer.dart AnnotationEntry.localPosition
	LocalPosition ui.Offset
}

// String describes the entry.
//
// Flutter: rendering/layer.dart AnnotationEntry.toString
func (e AnnotationEntry) String() string {
	return "AnnotationEntry(annotation: " + fmt.Sprint(e.Annotation) + ", localPosition: " + e.LocalPosition.String() + ")"
}

// AnnotationResult is the list of annotations of one type that
// Layer.FindAnnotations found, front first.
//
// Deviation: Flutter's AnnotationResult<T> is generic, and its layers compare
// the type T of their annotation with S at run time; here the result holds a
// type check made by NewAnnotationResult (SPEC §12.10).
//
// Flutter: rendering/layer.dart AnnotationResult
type AnnotationResult struct {
	entries []AnnotationEntry
	accepts func(any) bool
}

// NewAnnotationResult creates an empty result that collects annotations that
// are of type S.
//
// Flutter: rendering/layer.dart AnnotationResult.new
func NewAnnotationResult[S any]() *AnnotationResult {
	return &AnnotationResult{accepts: func(v any) bool { _, ok := v.(S); return ok }}
}

// Accepts reports whether the result collects annotations like this value.
// Layers call it before they Add.
//
// Flutter: rendering/layer.dart AnnotationResult (the type S)
func (r *AnnotationResult) Accepts(annotation any) bool { return r.accepts(annotation) }

// Add adds the entry to the end of the result.
//
// Flutter: rendering/layer.dart AnnotationResult.add
func (r *AnnotationResult) Add(entry AnnotationEntry) { r.entries = append(r.entries, entry) }

// Entries are the annotations found with their positions, front first.
//
// Flutter: rendering/layer.dart AnnotationResult.entries
func (r *AnnotationResult) Entries() []AnnotationEntry { return r.entries }

// Annotations are the annotations found, front first.
//
// Flutter: rendering/layer.dart AnnotationResult.annotations
func (r *AnnotationResult) Annotations() []any {
	out := make([]any, len(r.entries))
	for i, e := range r.entries {
		out[i] = e.Annotation
	}
	return out
}

// CompositionCallback is called when the layer is composited into a scene (or
// removed), with the layer.
//
// Flutter: rendering/layer.dart CompositionCallback
type CompositionCallback func(layer Layer)

// Layer is a composited layer: a node of the tree that the pipeline builds
// from the painting and turns into a scene. A layer type embeds one of the
// structs of this package (LayerBase, ContainerLayer, OffsetLayer, ...) and
// overrides the methods it needs; the embedded struct reaches the outer
// object through the self pointer that the constructor set.
//
// Flutter: rendering/layer.dart Layer
type Layer interface {
	foundation.DiagnosticableTree
	// AddToScene adds the layer and its children to the scene builder. It is
	// the method to override; the pipeline calls it through the retained
	// rendering path.
	//
	// Flutter: rendering/layer.dart Layer.addToScene
	AddToScene(builder *ui.SceneBuilder)
	// FindAnnotations adds the annotations at the position (in the layer's
	// coordinates) to the result, and reports whether the layer absorbs hits.
	//
	// Flutter: rendering/layer.dart Layer.findAnnotations
	FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool
	// AlwaysNeedsAddToScene reports whether the layer must be added to every
	// scene, even when it did not change.
	//
	// Flutter: rendering/layer.dart Layer.alwaysNeedsAddToScene
	AlwaysNeedsAddToScene() bool
	// SupportsRasterization reports whether the layer can be rasterized (an
	// embedded platform view cannot).
	//
	// Flutter: rendering/layer.dart Layer.supportsRasterization
	SupportsRasterization() bool
	// DescribeClipBounds are the bounds of the clip that the layer applies, if
	// it is a clip.
	//
	// Flutter: rendering/layer.dart Layer.describeClipBounds
	DescribeClipBounds() foundation.Opt[ui.Rect]
	// Attach attaches the layer to the owner.
	//
	// Flutter: rendering/layer.dart Layer.attach
	Attach(owner any)
	// Detach detaches the layer from its owner.
	//
	// Flutter: rendering/layer.dart Layer.detach
	Detach()
	// Dispose releases the layer's resources. Use handles; do not call it
	// while a handle holds the layer.
	//
	// Flutter: rendering/layer.dart Layer.dispose
	Dispose()
	// UpdateSubtreeNeedsAddToScene marks the layer as needing AddToScene if
	// it or a descendant needs it.
	//
	// Flutter: rendering/layer.dart Layer.updateSubtreeNeedsAddToScene
	UpdateSubtreeNeedsAddToScene()
	// RedepthChildren updates the depth of the children.
	//
	// Flutter: rendering/layer.dart Layer.redepthChildren
	RedepthChildren()
	// AsLayer returns the LayerBase every layer embeds.
	AsLayer() *LayerBase

	fireCompositionCallbacks(includeChildren bool)
}

type compositionCallbackEntry struct {
	id int
	fn func()
}

// LayerBase is the state and the default behavior that every Layer embeds.
//
// Flutter: rendering/layer.dart Layer
type LayerBase struct {
	self Layer

	callbacks          []compositionCallbackEntry
	nextCallbackID     int
	compositionCount   int
	debugMutationsLock bool
	debugDisposed      bool
	parentHandle       LayerHandle[Layer]
	refCount           int
	parent             *ContainerLayer
	needsAddToScene    bool
	engineLayer        ui.EngineLayer
	owner              any
	depth              int
	nextSibling        Layer
	previousSibling    Layer

	// DebugCreator is the object that created the layer, for debug output.
	//
	// Flutter: rendering/layer.dart Layer.debugCreator
	DebugCreator any
}

// InitLayer sets the outer object of a layer. The constructor of a type that
// embeds LayerBase calls it.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitLayer(base *LayerBase, self Layer) {
	base.self = self
	base.needsAddToScene = true
}

// AsLayer returns the base itself.
func (b *LayerBase) AsLayer() *LayerBase { return b }

// SubtreeHasCompositionCallbacks reports whether this layer or a descendant has
// composition callbacks.
//
// Flutter: rendering/layer.dart Layer.subtreeHasCompositionCallbacks
func (b *LayerBase) SubtreeHasCompositionCallbacks() bool { return b.compositionCount > 0 }

func (b *LayerBase) updateSubtreeCompositionObserverCount(delta int) {
	b.compositionCount += delta
	if b.parent != nil {
		b.parent.updateSubtreeCompositionObserverCount(delta)
	}
}

func (b *LayerBase) fireCompositionCallbacks(includeChildren bool) {
	if len(b.callbacks) == 0 {
		return
	}
	for _, c := range append([]compositionCallbackEntry(nil), b.callbacks...) {
		c.fn()
	}
}

func (b *LayerBase) debugCheckMutable(what string) {
	if foundation.DebugMode && b.debugMutationsLock {
		panic(errors.New(tagLayerMutationLocked, "a layer was changed while its composition callbacks ran",
			"operation", what, "layer", foundation.DescribeIdentity(b.self)))
	}
}

// SupportsRasterization reports whether the layer can be rasterized.
//
// Flutter: rendering/layer.dart Layer.supportsRasterization
func (b *LayerBase) SupportsRasterization() bool { return true }

// DescribeClipBounds is nothing for a layer that does not clip.
//
// Flutter: rendering/layer.dart Layer.describeClipBounds
func (b *LayerBase) DescribeClipBounds() foundation.Opt[ui.Rect] { return foundation.None[ui.Rect]() }

// AddCompositionCallback registers a callback that is called when the layer is
// composited into a scene and when it is detached. It returns the function
// that removes the callback.
//
// Flutter: rendering/layer.dart Layer.addCompositionCallback
func (b *LayerBase) AddCompositionCallback(callback CompositionCallback) func() {
	b.updateSubtreeCompositionObserverCount(1)
	b.nextCallbackID++
	id := b.nextCallbackID
	b.callbacks = append(b.callbacks, compositionCallbackEntry{id, func() {
		b.debugMutationsLock = foundation.DebugMode
		callback(b.self)
		b.debugMutationsLock = false
	}})
	return func() {
		found := false
		for i, c := range b.callbacks {
			if c.id == id {
				b.callbacks = append(b.callbacks[:i], b.callbacks[i+1:]...)
				found = true
				break
			}
		}
		if foundation.DebugMode && !found && !b.debugDisposed {
			panic(errors.New(tagLayerCompositionCallbackRemoved, "the composition callback was removed twice",
				"layer", foundation.DescribeIdentity(b.self)))
		}
		b.updateSubtreeCompositionObserverCount(-1)
	}
}

// DebugDisposed reports whether the layer was disposed (always false in
// release builds).
//
// Flutter: rendering/layer.dart Layer.debugDisposed
func (b *LayerBase) DebugDisposed() bool { return b.debugDisposed }

func (b *LayerBase) unref() {
	b.debugCheckMutable("unref")
	b.refCount--
	if b.refCount == 0 {
		b.self.Dispose()
	}
}

// DebugHandleCount is the number of handles that hold the layer.
//
// Flutter: rendering/layer.dart Layer.debugHandleCount
func (b *LayerBase) DebugHandleCount() int { return b.refCount }

// Dispose releases the engine layer. Layers are disposed when the last handle
// lets go of them; a layer must be disposed once. A debug build panics with
// LayerDisposedTwice or LayerDisposedWithHandles when it is not.
//
// Flutter: rendering/layer.dart Layer.dispose
func (b *LayerBase) Dispose() {
	b.debugCheckMutable("dispose")
	if foundation.DebugMode {
		if b.debugDisposed {
			panic(errors.New(tagLayerDisposedTwice, "layers must only be disposed once; this is handled by LayerHandle",
				"layer", foundation.DescribeIdentity(b.self)))
		}
		if b.refCount != 0 {
			panic(errors.New(tagLayerDisposedWithHandles, "do not call Dispose on a layer that handles hold; release the handles",
				"layer", foundation.DescribeIdentity(b.self), "handles", b.refCount))
		}
		b.debugDisposed = true
	}
	if b.engineLayer != nil {
		b.engineLayer.Dispose()
	}
	b.engineLayer = nil
}

// Parent is the layer that holds this one as a child.
//
// Flutter: rendering/layer.dart Layer.parent
func (b *LayerBase) Parent() *ContainerLayer { return b.parent }

func (b *LayerBase) markNeedsAddToScene() {
	b.debugCheckMutable("markNeedsAddToScene")
	if foundation.DebugMode {
		if b.self.AlwaysNeedsAddToScene() {
			panic(errors.New(tagLayerAlwaysNeedsAddToScene,
				"a layer with AlwaysNeedsAddToScene called MarkNeedsAddToScene; it is always added",
				"layer", foundation.DescribeIdentity(b.self)))
		}
		if b.debugDisposed {
			panic(errors.New(tagLayerDisposedUsed, "a disposed layer was marked as needing to be added to the scene",
				"layer", foundation.DescribeIdentity(b.self)))
		}
	}
	b.needsAddToScene = true
}

// MarkNeedsAddToScene makes the layer be added to the next scene again, not
// reused from the previous one.
//
// Flutter: rendering/layer.dart Layer.markNeedsAddToScene
func (b *LayerBase) MarkNeedsAddToScene() { b.markNeedsAddToScene() }

// DebugMarkClean marks the layer as not needing AddToScene (debug builds
// only), for tests.
//
// Flutter: rendering/layer.dart Layer.debugMarkClean
func (b *LayerBase) DebugMarkClean() {
	b.debugCheckMutable("debugMarkClean")
	if foundation.DebugMode {
		b.needsAddToScene = false
	}
}

// AlwaysNeedsAddToScene is false: a layer is reused when it did not change.
//
// Flutter: rendering/layer.dart Layer.alwaysNeedsAddToScene
func (b *LayerBase) AlwaysNeedsAddToScene() bool { return false }

// DebugSubtreeNeedsAddToScene reports whether the layer is marked as needing
// to be added to the scene (debug builds only; release builds give false).
//
// Flutter: rendering/layer.dart Layer.debugSubtreeNeedsAddToScene
func (b *LayerBase) DebugSubtreeNeedsAddToScene() bool {
	return foundation.DebugMode && b.needsAddToScene
}

// EngineLayer is the engine layer of the last time the layer was added to a
// scene, or nil.
//
// Flutter: rendering/layer.dart Layer.engineLayer
func (b *LayerBase) EngineLayer() ui.EngineLayer { return b.engineLayer }

// SetEngineLayer replaces the engine layer, disposing the old one, and marks
// the parent as needing AddToScene.
//
// Flutter: rendering/layer.dart Layer.engineLayer
func (b *LayerBase) SetEngineLayer(value ui.EngineLayer) {
	b.debugCheckMutable("engineLayer")
	if foundation.DebugMode && b.debugDisposed {
		panic(errors.New(tagLayerDisposedUsed, "the engine layer of a disposed layer was set",
			"layer", foundation.DescribeIdentity(b.self)))
	}
	if b.engineLayer != nil {
		b.engineLayer.Dispose()
	}
	b.engineLayer = value
	if !b.self.AlwaysNeedsAddToScene() {
		if b.parent != nil && !b.parent.self.AlwaysNeedsAddToScene() {
			b.parent.markNeedsAddToScene()
		}
	}
}

// UpdateSubtreeNeedsAddToScene marks the layer as needing AddToScene if it
// always does.
//
// Flutter: rendering/layer.dart Layer.updateSubtreeNeedsAddToScene
func (b *LayerBase) UpdateSubtreeNeedsAddToScene() {
	b.debugCheckMutable("updateSubtreeNeedsAddToScene")
	b.needsAddToScene = b.needsAddToScene || b.self.AlwaysNeedsAddToScene()
}

// Owner is the object that the layer is attached to, or nil.
//
// Flutter: rendering/layer.dart Layer.owner
func (b *LayerBase) Owner() any { return b.owner }

// Attached reports whether the layer has an owner.
//
// Flutter: rendering/layer.dart Layer.attached
func (b *LayerBase) Attached() bool { return b.owner != nil }

// Attach sets the owner.
//
// Flutter: rendering/layer.dart Layer.attach
func (b *LayerBase) Attach(owner any) {
	if foundation.DebugMode && b.owner != nil {
		panic(errors.New(tagLayerAlreadyAttached, "the layer is attached to an owner already",
			"layer", foundation.DescribeIdentity(b.self)))
	}
	b.owner = owner
}

// Detach clears the owner.
//
// Flutter: rendering/layer.dart Layer.detach
func (b *LayerBase) Detach() {
	if foundation.DebugMode && b.owner == nil {
		panic(errors.New(tagLayerNotAttached, "a layer that is not attached was detached",
			"layer", foundation.DescribeIdentity(b.self)))
	}
	b.owner = nil
}

// Depth is the distance of the layer from the root, set when it is attached.
//
// Flutter: rendering/layer.dart Layer.depth
func (b *LayerBase) Depth() int { return b.depth }

// RedepthChildren does nothing: a leaf has no children.
//
// Flutter: rendering/layer.dart Layer.redepthChildren
func (b *LayerBase) RedepthChildren() {}

// NextSibling is the layer after this one in the parent's list.
//
// Flutter: rendering/layer.dart Layer.nextSibling
func (b *LayerBase) NextSibling() Layer { return b.nextSibling }

// PreviousSibling is the layer before this one in the parent's list.
//
// Flutter: rendering/layer.dart Layer.previousSibling
func (b *LayerBase) PreviousSibling() Layer { return b.previousSibling }

// Remove removes the layer from its parent.
//
// Flutter: rendering/layer.dart Layer.remove
func (b *LayerBase) Remove() {
	b.debugCheckMutable("remove")
	if b.parent != nil {
		b.parent.removeChild(b.self)
	}
}

// FindAnnotations finds nothing: a layer has no annotations by default.
//
// Flutter: rendering/layer.dart Layer.findAnnotations
func (b *LayerBase) FindAnnotations(*AnnotationResult, ui.Offset, bool) bool { return false }

// AddToScene panics: a layer type must override it.
//
// Flutter: rendering/layer.dart Layer.addToScene
func (b *LayerBase) AddToScene(*ui.SceneBuilder) {
	panic(errors.New(tagLayerAddToSceneMissing, "the layer type does not implement AddToScene",
		"layer", foundation.DescribeIdentity(b.self)))
}

func (b *LayerBase) addToSceneWithRetainedRendering(builder *ui.SceneBuilder) {
	b.debugCheckMutable("addToScene")
	if !b.needsAddToScene && b.engineLayer != nil {
		builder.AddRetained(b.engineLayer)
		return
	}
	b.self.AddToScene(builder)
	b.needsAddToScene = false
}

// LayerFind returns the first annotation of type S at the position, searching
// the layer front to back. The position is in the layer's coordinates.
//
// Flutter: rendering/layer.dart Layer.find
func LayerFind[S any](l Layer, localPosition ui.Offset) (S, bool) {
	result := NewAnnotationResult[S]()
	l.FindAnnotations(result, localPosition, true)
	if len(result.entries) == 0 {
		var zero S
		return zero, false
	}
	return result.entries[0].Annotation.(S), true
}

// LayerFindAllAnnotations returns all annotations of type S at the position,
// front first.
//
// Flutter: rendering/layer.dart Layer.findAllAnnotations
func LayerFindAllAnnotations[S any](l Layer, localPosition ui.Offset) *AnnotationResult {
	result := NewAnnotationResult[S]()
	l.FindAnnotations(result, localPosition, false)
	return result
}

// ToStringShort adds " DETACHED" to a layer that has no owner.
//
// Flutter: rendering/layer.dart Layer.toStringShort
func (b *LayerBase) ToStringShort() string {
	s := foundation.DescribeIdentity(b.self)
	if b.owner == nil {
		s += " DETACHED"
	}
	return s
}

// String describes the layer.
func (b *LayerBase) String() string { return foundation.DiagnosticableString(b.self) }

// DebugDescribeChildren is empty for a leaf layer.
//
// Flutter: rendering/layer.dart Layer.debugDescribeChildren
func (b *LayerBase) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

// DebugFillProperties adds the owner, the creator, the engine layer and the
// handle count.
//
// Flutter: rendering/layer.dart Layer.debugFillProperties
func (b *LayerBase) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	level := foundation.DiagnosticLevelInfo
	if b.parent != nil {
		level = foundation.DiagnosticLevelHidden
	}
	p.Add(foundation.NewDiagnosticsProperty("owner", b.owner,
		foundation.DiagnosticsPropertyOpts{Level: level, DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewDiagnosticsProperty("creator", b.DebugCreator,
		foundation.DiagnosticsPropertyOpts{Level: foundation.DiagnosticLevelDebug, DefaultValue: foundation.DefaultNull}))
	if b.engineLayer != nil {
		p.Add(foundation.NewDiagnosticsProperty("engine layer", foundation.DescribeIdentity(b.engineLayer)))
	}
	p.Add(foundation.NewDiagnosticsProperty("handles", b.refCount))
}

// LayerHandle holds a reference to a layer and disposes the layer when the
// last handle lets go. A render object holds its layer in one.
//
// Flutter: rendering/layer.dart LayerHandle
type LayerHandle[T Layer] struct {
	layer T
}

// NewLayerHandle creates a handle holding the layer (nil for none).
//
// Flutter: rendering/layer.dart LayerHandle.new
func NewLayerHandle[T Layer](layer T) *LayerHandle[T] {
	h := &LayerHandle[T]{}
	h.SetLayer(layer)
	return h
}

func isNilLayer[T Layer](l T) bool {
	var zero T
	return any(l) == any(zero)
}

// Layer is the held layer, or the zero value of T (nil).
//
// Flutter: rendering/layer.dart LayerHandle.layer
func (h *LayerHandle[T]) Layer() T { return h.layer }

// SetLayer replaces the held layer, releasing the old one. Pass the zero value
// (nil) to let go. Setting a disposed layer panics with LayerHandleDisposed in
// debug builds.
//
// Flutter: rendering/layer.dart LayerHandle.layer
func (h *LayerHandle[T]) SetLayer(layer T) {
	if !isNilLayer(layer) && foundation.DebugMode && layer.AsLayer().debugDisposed {
		panic(errors.New(tagLayerHandleDisposed, "a handle was created for a layer that was disposed already",
			"layer", foundation.DescribeIdentity(layer)))
	}
	if any(layer) == any(h.layer) {
		return
	}
	if !isNilLayer(h.layer) {
		h.layer.AsLayer().unref()
	}
	h.layer = layer
	if !isNilLayer(h.layer) {
		h.layer.AsLayer().refCount++
	}
}

// String describes the handle.
//
// Flutter: rendering/layer.dart LayerHandle.toString
func (h *LayerHandle[T]) String() string {
	if isNilLayer(h.layer) {
		return "LayerHandle(DISPOSED)"
	}
	return "LayerHandle(" + foundation.DescribeIdentity(h.layer) + ")"
}

// PictureLayer is a layer that holds a picture the canvas recorded.
//
// Flutter: rendering/layer.dart PictureLayer
type PictureLayer struct {
	LayerBase
	canvasBounds   ui.Rect
	picture        *ui.Picture
	isComplexHint  bool
	willChangeHint bool
}

// NewPictureLayer creates a layer for a picture that is recorded into the
// bounds.
//
// Flutter: rendering/layer.dart PictureLayer.new
func NewPictureLayer(canvasBounds ui.Rect) *PictureLayer {
	l := &PictureLayer{canvasBounds: canvasBounds}
	InitLayer(&l.LayerBase, l)
	return l
}

// CanvasBounds are the bounds the picture is drawn in, for debugging; the
// picture is not clipped to them.
//
// Flutter: rendering/layer.dart PictureLayer.canvasBounds
func (l *PictureLayer) CanvasBounds() ui.Rect { return l.canvasBounds }

// Picture is the picture of the layer.
//
// Flutter: rendering/layer.dart PictureLayer.picture
func (l *PictureLayer) Picture() *ui.Picture { return l.picture }

// SetPicture replaces the picture, disposing the old one.
//
// Flutter: rendering/layer.dart PictureLayer.picture
func (l *PictureLayer) SetPicture(p *ui.Picture) {
	if foundation.DebugMode && l.debugDisposed {
		panic(errors.New(tagLayerDisposedUsed, "the picture of a disposed layer was set", "layer", foundation.DescribeIdentity(l)))
	}
	l.markNeedsAddToScene()
	if l.picture != nil {
		l.picture.Dispose()
	}
	l.picture = p
}

// IsComplexHint tells the engine that the picture is expensive to draw.
//
// Flutter: rendering/layer.dart PictureLayer.isComplexHint
func (l *PictureLayer) IsComplexHint() bool { return l.isComplexHint }

// SetIsComplexHint sets the hint.
//
// Flutter: rendering/layer.dart PictureLayer.isComplexHint
func (l *PictureLayer) SetIsComplexHint(v bool) {
	if v != l.isComplexHint {
		l.isComplexHint = v
		l.markNeedsAddToScene()
	}
}

// WillChangeHint tells the engine that the picture is likely to change.
//
// Flutter: rendering/layer.dart PictureLayer.willChangeHint
func (l *PictureLayer) WillChangeHint() bool { return l.willChangeHint }

// SetWillChangeHint sets the hint.
//
// Flutter: rendering/layer.dart PictureLayer.willChangeHint
func (l *PictureLayer) SetWillChangeHint(v bool) {
	if v != l.willChangeHint {
		l.willChangeHint = v
		l.markNeedsAddToScene()
	}
}

// Dispose disposes the picture and the layer.
//
// Flutter: rendering/layer.dart PictureLayer.dispose
func (l *PictureLayer) Dispose() {
	l.SetPicture(nil)
	l.LayerBase.Dispose()
}

// AddToScene adds the picture to the scene.
//
// Flutter: rendering/layer.dart PictureLayer.addToScene
func (l *PictureLayer) AddToScene(builder *ui.SceneBuilder) {
	if foundation.DebugMode && l.picture == nil {
		panic(errors.New(tagLayerPropertyMissing, "a picture layer without a picture was added to a scene"))
	}
	builder.AddPicture(ui.OffsetZero, l.picture, ui.AddPictureOpts{IsComplexHint: l.isComplexHint, WillChangeHint: l.willChangeHint})
}

// DebugFillProperties adds the paint bounds, the picture and the hints.
//
// Flutter: rendering/layer.dart PictureLayer.debugFillProperties
func (l *PictureLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.LayerBase.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("paint bounds", l.canvasBounds))
	p.Add(foundation.NewDiagnosticsProperty("picture", foundation.DescribeIdentity(l.picture)))
	p.Add(foundation.NewDiagnosticsProperty("raster cache hints",
		"isComplex = "+boolString(l.isComplexHint)+", willChange = "+boolString(l.willChangeHint)))
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

var backdropKeyCounter atomic.Int64

// BackdropKey identifies a backdrop that several BackdropFilterLayers share.
//
// Deviation: the key is made from a counter that all keys share, as Flutter's
// static counter is (SPEC §12.10).
//
// Flutter: rendering/layer.dart BackdropKey
type BackdropKey struct{ key int }

// NewBackdropKey creates a key that differs from all the others.
//
// Flutter: rendering/layer.dart BackdropKey.new
func NewBackdropKey() BackdropKey { return BackdropKey{key: int(backdropKeyCounter.Add(1) - 1)} }

// Key is the number that identifies the backdrop.
func (k BackdropKey) Key() int { return k.key }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagLayerAddToSceneMissing          = "LayerAddToSceneMissing"
	tagLayerAlreadyAttached            = "LayerAlreadyAttached"
	tagLayerAlwaysNeedsAddToScene      = "LayerAlwaysNeedsAddToScene"
	tagLayerCompositionCallbackRemoved = "LayerCompositionCallbackRemoved"
	tagLayerDisposedTwice              = "LayerDisposedTwice"
	tagLayerDisposedUsed               = "LayerDisposedUsed"
	tagLayerDisposedWithHandles        = "LayerDisposedWithHandles"
	tagLayerHandleDisposed             = "LayerHandleDisposed"
	tagLayerMutationLocked             = "LayerMutationLocked"
	tagLayerNotAttached                = "LayerNotAttached"
	tagLayerPropertyMissing            = "LayerPropertyMissing"
)
