package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// LayerLink links a LeaderLayer to the FollowerLayers that follow it. Create
// one and give it to both.
//
// Deviation: the check that a previous leader is gone by the end of the frame
// (a post-frame callback in Flutter) is not ported (SPEC §12.10).
//
// Flutter: rendering/layer.dart LayerLink
type LayerLink struct {
	leader          *LeaderLayer
	previousLeaders map[*LeaderLayer]struct{}
	// LeaderSize is the size of the leader's render object, set when it paints;
	// the follower can use it to place itself.
	//
	// Flutter: rendering/layer.dart LayerLink.leaderSize
	LeaderSize foundation.Opt[ui.Size]
}

// NewLayerLink creates a link without a leader.
//
// Flutter: rendering/layer.dart LayerLink.new
func NewLayerLink() *LayerLink { return &LayerLink{} }

// Leader is the leader layer that is attached now, or nil.
//
// Flutter: rendering/layer.dart LayerLink.leader
func (k *LayerLink) Leader() *LeaderLayer { return k.leader }

func (k *LayerLink) registerLeader(leader *LeaderLayer) {
	if foundation.DebugMode {
		if k.leader != nil {
			if k.previousLeaders == nil {
				k.previousLeaders = map[*LeaderLayer]struct{}{}
			}
			k.previousLeaders[k.leader] = struct{}{}
		}
	}
	k.leader = leader
}

func (k *LayerLink) unregisterLeader(leader *LeaderLayer) {
	if k.leader == leader {
		k.leader = nil
		return
	}
	if foundation.DebugMode {
		if _, ok := k.previousLeaders[leader]; !ok {
			panic(errors.New("LayerLinkLeaderUnknown", "the leader was not registered with the link"))
		}
		delete(k.previousLeaders, leader)
	}
}

// DebugPreviousLeaderCount is the number of leaders that were replaced and are
// still attached (debug builds only); it is 0 when the app is correct.
//
// Flutter: rendering/layer.dart LayerLink._debugPreviousLeaders
func (k *LayerLink) DebugPreviousLeaderCount() int { return len(k.previousLeaders) }

// String describes the link.
//
// Flutter: rendering/layer.dart LayerLink.toString
func (k *LayerLink) String() string {
	state := "<dangling>"
	if k.leader != nil {
		state = "<linked>"
	}
	return foundation.DescribeIdentity(k) + "(" + state + ")"
}

// LeaderLayer is the layer that FollowerLayers follow.
//
// Flutter: rendering/layer.dart LeaderLayer
type LeaderLayer struct {
	ContainerLayer
	link   *LayerLink
	offset ui.Offset
}

// LeaderLayerOpts are the optional parameters of NewLeaderLayer.
//
// Flutter: rendering/layer.dart LeaderLayer.new
type LeaderLayerOpts struct {
	// Offset is the position of the leader in its parent.
	Offset ui.Offset
}

// NewLeaderLayer creates a leader for the link.
//
// Flutter: rendering/layer.dart LeaderLayer.new
func NewLeaderLayer(link *LayerLink, opts ...LeaderLayerOpts) *LeaderLayer {
	l := &LeaderLayer{link: link, offset: firstOpts(opts).Offset}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// Link is the link the leader belongs to.
//
// Flutter: rendering/layer.dart LeaderLayer.link
func (l *LeaderLayer) Link() *LayerLink { return l.link }

// SetLink changes the link, moving the registration if attached.
//
// Flutter: rendering/layer.dart LeaderLayer.link
func (l *LeaderLayer) SetLink(v *LayerLink) {
	if l.link == v {
		return
	}
	if l.Attached() {
		l.link.unregisterLeader(l)
		v.registerLeader(l)
	}
	l.link = v
}

// Offset is the position of the leader in its parent.
//
// Flutter: rendering/layer.dart LeaderLayer.offset
func (l *LeaderLayer) Offset() ui.Offset { return l.offset }

// SetOffset moves the leader.
//
// Flutter: rendering/layer.dart LeaderLayer.offset
func (l *LeaderLayer) SetOffset(v ui.Offset) {
	if v == l.offset {
		return
	}
	l.offset = v
	if !l.AlwaysNeedsAddToScene() {
		l.markNeedsAddToScene()
	}
}

// Attach registers the leader with its link.
//
// Flutter: rendering/layer.dart LeaderLayer.attach
func (l *LeaderLayer) Attach(owner any) {
	l.ContainerLayer.Attach(owner)
	l.link.registerLeader(l)
}

// Detach unregisters the leader from its link.
//
// Flutter: rendering/layer.dart LeaderLayer.detach
func (l *LeaderLayer) Detach() {
	l.link.unregisterLeader(l)
	l.ContainerLayer.Detach()
}

// FindAnnotations searches the children at the position minus the offset.
//
// Flutter: rendering/layer.dart LeaderLayer.findAnnotations
func (l *LeaderLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	return l.ContainerLayer.FindAnnotations(result, localPosition.Sub(l.offset), onlyFirst)
}

// AddToScene pushes the offset as a transform, if there is one, and adds the
// children.
//
// Flutter: rendering/layer.dart LeaderLayer.addToScene
func (l *LeaderLayer) AddToScene(builder *ui.SceneBuilder) {
	if l.offset != ui.OffsetZero {
		old, _ := l.engineLayer.(*ui.TransformEngineLayer)
		l.SetEngineLayer(builder.PushTransform(ui.NewMatrix4TranslationValues(float32(l.offset.Dx), float32(l.offset.Dy), 0),
			ui.PushTransformOpts{OldLayer: old}))
	} else {
		l.SetEngineLayer(nil)
	}
	l.AddChildrenToScene(builder)
	if l.offset != ui.OffsetZero {
		builder.Pop()
	}
}

// ApplyTransform translates the transform by the offset.
//
// Flutter: rendering/layer.dart LeaderLayer.applyTransform
func (l *LeaderLayer) ApplyTransform(child Layer, transform *ui.Matrix4) {
	if l.offset != ui.OffsetZero {
		transform.Translate(float32(l.offset.Dx), float32(l.offset.Dy), 0)
	}
}

// DebugFillProperties adds the offset and the link.
//
// Flutter: rendering/layer.dart LeaderLayer.debugFillProperties
func (l *LeaderLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("offset", l.offset))
	p.Add(foundation.NewDiagnosticsProperty("link", l.link))
}

// FollowerLayer is a layer that follows a LeaderLayer: it applies the transform
// from the leader's coordinates to its own.
//
// Flutter: rendering/layer.dart FollowerLayer
type FollowerLayer struct {
	ContainerLayer
	// Link is the link to the leader.
	//
	// Flutter: rendering/layer.dart FollowerLayer.link
	Link *LayerLink
	// ShowWhenUnlinked says whether the children are shown when the link has no
	// leader.
	//
	// Flutter: rendering/layer.dart FollowerLayer.showWhenUnlinked
	ShowWhenUnlinked bool
	// UnlinkedOffset is where the children are painted when there is no leader.
	//
	// Flutter: rendering/layer.dart FollowerLayer.unlinkedOffset
	UnlinkedOffset ui.Offset
	// LinkedOffset is the offset from the leader's origin when there is one.
	//
	// Flutter: rendering/layer.dart FollowerLayer.linkedOffset
	LinkedOffset ui.Offset

	lastOffset        foundation.Opt[ui.Offset]
	lastTransform     *ui.Matrix4
	invertedTransform *ui.Matrix4
	inverseDirty      bool
}

// FollowerLayerOpts are the optional parameters of NewFollowerLayer.
//
// Flutter: rendering/layer.dart FollowerLayer.new
type FollowerLayerOpts struct {
	// HideWhenUnlinked hides the children when the link has no leader; the
	// default is to show them (Flutter's showWhenUnlinked = true).
	HideWhenUnlinked bool
	// UnlinkedOffset is where the children are painted when there is no leader.
	UnlinkedOffset ui.Offset
	// LinkedOffset is the offset from the leader's origin when there is one.
	LinkedOffset ui.Offset
}

// NewFollowerLayer creates a follower of the link.
//
// Flutter: rendering/layer.dart FollowerLayer.new
func NewFollowerLayer(link *LayerLink, opts ...FollowerLayerOpts) *FollowerLayer {
	o := firstOpts(opts)
	l := &FollowerLayer{Link: link, ShowWhenUnlinked: !o.HideWhenUnlinked, UnlinkedOffset: o.UnlinkedOffset,
		LinkedOffset: o.LinkedOffset, inverseDirty: true}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

func (l *FollowerLayer) transformOffset(localPosition ui.Offset) (ui.Offset, bool) {
	if l.inverseDirty {
		if m := l.LastTransform(); m != nil {
			l.invertedTransform = tryInvert(m)
		} else {
			l.invertedTransform = nil
		}
		l.inverseDirty = false
	}
	if l.invertedTransform == nil {
		return ui.Offset{}, false
	}
	r := l.invertedTransform.Transform(ui.Vector4{X: float32(localPosition.Dx), Y: float32(localPosition.Dy), Z: 0, W: 1})
	return ui.Offset{Dx: float64(r.X) - l.LinkedOffset.Dx, Dy: float64(r.Y) - l.LinkedOffset.Dy}, true
}

// FindAnnotations searches the children at the position in the leader's
// coordinates, or at the unlinked offset if there is no leader.
//
// Flutter: rendering/layer.dart FollowerLayer.findAnnotations
func (l *FollowerLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	if l.Link.leader == nil {
		if l.ShowWhenUnlinked {
			return l.ContainerLayer.FindAnnotations(result, localPosition.Sub(l.UnlinkedOffset), onlyFirst)
		}
		return false
	}
	p, ok := l.transformOffset(localPosition)
	if !ok {
		return false
	}
	return l.ContainerLayer.FindAnnotations(result, p, onlyFirst)
}

// LastTransform is the transform from the follower's parent's coordinates to
// the leader's, used by the last AddToScene, or nil.
//
// Flutter: rendering/layer.dart FollowerLayer.getLastTransform
func (l *FollowerLayer) LastTransform() *ui.Matrix4 {
	if l.lastTransform == nil {
		return nil
	}
	off, _ := l.lastOffset.Get()
	result := ui.NewMatrix4TranslationValues(float32(-off.Dx), float32(-off.Dy), 0)
	result.MultiplyBy(l.lastTransform)
	return result
}

func collectTransformForLayerChain(layers []ContainerLayerLike) *ui.Matrix4 {
	result := ui.NewMatrix4Identity()
	for index := len(layers) - 1; index > 0; index-- {
		if layers[index] != nil {
			var child Layer
			if layers[index-1] != nil {
				child = layers[index-1]
			}
			layers[index].ApplyTransform(child, result)
		}
	}
	return result
}

func parentLike(l Layer) ContainerLayerLike {
	p := l.AsLayer().parent
	if p == nil {
		return nil
	}
	return p.self.(ContainerLayerLike)
}

func pathsToCommonAncestor(a, b Layer, ancestorsA, ancestorsB *[]ContainerLayerLike) Layer {
	for {
		if a == nil || b == nil {
			return nil
		}
		if a == b {
			return a
		}
		ab, bb := a.AsLayer(), b.AsLayer()
		switch {
		case ab.depth < bb.depth:
			*ancestorsB = append(*ancestorsB, parentLike(b))
			b = layerOrNil(parentLike(b))
		case ab.depth > bb.depth:
			*ancestorsA = append(*ancestorsA, parentLike(a))
			a = layerOrNil(parentLike(a))
		default:
			*ancestorsA = append(*ancestorsA, parentLike(a))
			*ancestorsB = append(*ancestorsB, parentLike(b))
			a, b = layerOrNil(parentLike(a)), layerOrNil(parentLike(b))
		}
	}
}

func layerOrNil(c ContainerLayerLike) Layer {
	if c == nil {
		return nil
	}
	return c
}

func debugCheckLeaderBeforeFollower(leaderToCommon, followerToCommon []ContainerLayerLike) bool {
	if len(followerToCommon) <= 1 {
		return false
	}
	if len(leaderToCommon) <= 1 {
		return true
	}
	leaderBelow := leaderToCommon[len(leaderToCommon)-2]
	followerBelow := followerToCommon[len(followerToCommon)-2]
	for sibling := Layer(leaderBelow); sibling != nil; sibling = sibling.AsLayer().nextSibling {
		if sibling == Layer(followerBelow) {
			return true
		}
	}
	return false
}

func (l *FollowerLayer) establishTransform() {
	l.lastTransform = nil
	leader := l.Link.leader
	if leader == nil {
		return
	}
	if foundation.DebugMode && leader.owner != l.owner {
		panic(errors.New("FollowerLayerLeaderOwner", "the linked LeaderLayer anchor is not in the same layer tree as the FollowerLayer"))
	}
	forward := []ContainerLayerLike{leader}
	inverse := []ContainerLayerLike{l.self.(ContainerLayerLike)}
	ancestor := pathsToCommonAncestor(leader, l.self, &forward, &inverse)
	if foundation.DebugMode {
		if ancestor == nil {
			panic(errors.New("FollowerLayerNoCommonAncestor", "the LeaderLayer and the FollowerLayer do not have a common ancestor"))
		}
		if !debugCheckLeaderBeforeFollower(forward, inverse) {
			panic(errors.New("FollowerLayerLeaderAfterFollower",
				"the LeaderLayer anchor must come before the FollowerLayer in paint order, but the reverse was true"))
		}
	}
	forwardTransform := collectTransformForLayerChain(forward)
	leader.ApplyTransform(nil, forwardTransform)
	forwardTransform.Translate(float32(l.LinkedOffset.Dx), float32(l.LinkedOffset.Dy), 0)
	inverseTransform := collectTransformForLayerChain(inverse)
	if inverseTransform.Invert() == 0 {
		return
	}
	inverseTransform.MultiplyBy(forwardTransform)
	l.lastTransform = inverseTransform
	l.inverseDirty = true
}

// AlwaysNeedsAddToScene is true: the transform depends on the leader, which
// can move without this layer knowing.
//
// Flutter: rendering/layer.dart FollowerLayer.alwaysNeedsAddToScene
func (l *FollowerLayer) AlwaysNeedsAddToScene() bool { return true }

// AddToScene finds the transform to the leader, pushes it, adds the children
// and pops.
//
// Flutter: rendering/layer.dart FollowerLayer.addToScene
func (l *FollowerLayer) AddToScene(builder *ui.SceneBuilder) {
	if l.Link.leader == nil && !l.ShowWhenUnlinked {
		l.lastTransform = nil
		l.lastOffset = foundation.None[ui.Offset]()
		l.inverseDirty = true
		l.SetEngineLayer(nil)
		return
	}
	l.establishTransform()
	old, _ := l.engineLayer.(*ui.TransformEngineLayer)
	if l.lastTransform != nil {
		l.lastOffset = foundation.Some(l.UnlinkedOffset)
		l.SetEngineLayer(builder.PushTransform(l.lastTransform, ui.PushTransformOpts{OldLayer: old}))
	} else {
		l.lastOffset = foundation.None[ui.Offset]()
		m := ui.NewMatrix4TranslationValues(float32(l.UnlinkedOffset.Dx), float32(l.UnlinkedOffset.Dy), 0)
		l.SetEngineLayer(builder.PushTransform(m, ui.PushTransformOpts{OldLayer: old}))
	}
	l.AddChildrenToScene(builder)
	builder.Pop()
	l.inverseDirty = true
}

// ApplyTransform multiplies the transform by the follower's.
//
// Flutter: rendering/layer.dart FollowerLayer.applyTransform
func (l *FollowerLayer) ApplyTransform(child Layer, transform *ui.Matrix4) {
	if l.lastTransform != nil {
		transform.MultiplyBy(l.lastTransform)
	} else {
		transform.MultiplyBy(ui.NewMatrix4TranslationValues(float32(l.UnlinkedOffset.Dx), float32(l.UnlinkedOffset.Dy), 0))
	}
}

// DebugFillProperties adds the link and the transform.
//
// Flutter: rendering/layer.dart FollowerLayer.debugFillProperties
func (l *FollowerLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("link", l.Link))
	p.Add(painting.NewTransformProperty("transform", l.LastTransform(), foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}
