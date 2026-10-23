package rendering

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
)

// BaselineOffset is the distance from the top of a box to a baseline, or no
// baseline.
//
// Flutter: rendering/box.dart BaselineOffset
type BaselineOffset struct {
	// Offset is the distance; it is None for no baseline.
	Offset foundation.Opt[float64]
}

// NoBaseline is a BaselineOffset without a baseline.
//
// Flutter: rendering/box.dart BaselineOffset.noBaseline
var NoBaseline = BaselineOffset{}

// NewBaselineOffset makes a baseline offset of the distance (None for none).
//
// Flutter: rendering/box.dart BaselineOffset.new
func NewBaselineOffset(offset foundation.Opt[float64]) BaselineOffset { return BaselineOffset{offset} }

// Add moves the baseline down by the offset; no baseline stays none.
//
// Flutter: rendering/box.dart BaselineOffset.operator+
func (b BaselineOffset) Add(offset float64) BaselineOffset {
	if v, ok := b.Offset.Get(); ok {
		return BaselineOffset{foundation.Some(v + offset)}
	}
	return b
}

// MinOf is the baseline that is higher up; a baseline wins over none.
//
// Flutter: rendering/box.dart BaselineOffset.minOf
func (b BaselineOffset) MinOf(other BaselineOffset) BaselineOffset {
	lhs, okL := b.Offset.Get()
	rhs, okR := other.Offset.Get()
	switch {
	case okL && okR:
		if lhs >= rhs {
			return other
		}
		return b
	case okL:
		return b
	}
	return other
}

type intrinsicDimension int

const (
	intrinsicMinWidth intrinsicDimension = iota
	intrinsicMaxWidth
	intrinsicMinHeight
	intrinsicMaxHeight
)

type intrinsicKey struct {
	dim intrinsicDimension
	arg float64
}

// layoutCacheStorage caches the results of the intrinsic size, dry layout and
// dry baseline computations until the box needs layout.
//
// Flutter: rendering/box.dart _LayoutCacheStorage
type layoutCacheStorage struct {
	intrinsics map[intrinsicKey]float64
	dryLayout  map[BoxConstraints]ui.Size
	alphabetic map[BoxConstraints]BaselineOffset
	ideo       map[BoxConstraints]BaselineOffset
}

func (s *layoutCacheStorage) clear() bool {
	has := len(s.dryLayout) > 0 || len(s.intrinsics) > 0 || len(s.alphabetic) > 0 || len(s.ideo) > 0
	if has {
		clear(s.dryLayout)
		clear(s.intrinsics)
		clear(s.alphabetic)
		clear(s.ideo)
	}
	return has
}

// Box is a render object that uses the box layout protocol: every type that
// embeds RenderBox. The methods are those of RenderBox that are overridden or
// called by other boxes.
//
// Flutter: rendering/box.dart RenderBox
type Box interface {
	RenderObject
	// AsRenderBox returns the RenderBox that the object embeds.
	AsRenderBox() *RenderBox

	// Size is the size of the box, which layout set. It panics with
	// BoxNotLaidOut before layout.
	//
	// Flutter: rendering/box.dart RenderBox.size
	Size() ui.Size
	// HasSize reports whether the box has a size.
	//
	// Flutter: rendering/box.dart RenderBox.hasSize
	HasSize() bool
	// BoxConstraints are the constraints of the last layout.
	//
	// Flutter: rendering/box.dart RenderBox.constraints
	BoxConstraints() BoxConstraints

	// ComputeMinIntrinsicWidth is the smallest width at which the box can paint
	// without clipping, for the height.
	//
	// Flutter: rendering/box.dart RenderBox.computeMinIntrinsicWidth
	ComputeMinIntrinsicWidth(height float64) float64
	// ComputeMaxIntrinsicWidth is the width beyond which more width does not
	// help, for the height.
	//
	// Flutter: rendering/box.dart RenderBox.computeMaxIntrinsicWidth
	ComputeMaxIntrinsicWidth(height float64) float64
	// ComputeMinIntrinsicHeight is the smallest height at which the box can paint
	// without clipping, for the width.
	//
	// Flutter: rendering/box.dart RenderBox.computeMinIntrinsicHeight
	ComputeMinIntrinsicHeight(width float64) float64
	// ComputeMaxIntrinsicHeight is the height beyond which more height does not
	// help, for the width.
	//
	// Flutter: rendering/box.dart RenderBox.computeMaxIntrinsicHeight
	ComputeMaxIntrinsicHeight(width float64) float64
	// ComputeDryLayout is the size the box would have for the constraints,
	// without laying out.
	//
	// Flutter: rendering/box.dart RenderBox.computeDryLayout
	ComputeDryLayout(constraints BoxConstraints) ui.Size
	// ComputeDryBaseline is the baseline the box would have for the constraints,
	// without laying out.
	//
	// Flutter: rendering/box.dart RenderBox.computeDryBaseline
	ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64]
	// ComputeDistanceToActualBaseline is the distance to the baseline, or None
	// if the box has none.
	//
	// Flutter: rendering/box.dart RenderBox.computeDistanceToActualBaseline
	ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64]
	// HitTestSelf reports whether the box itself is hit at the position.
	//
	// Flutter: rendering/box.dart RenderBox.hitTestSelf
	HitTestSelf(position ui.Offset) bool
	// HitTestChildren hit tests the children at the position.
	//
	// Flutter: rendering/box.dart RenderBox.hitTestChildren
	HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool

	// GetMinIntrinsicWidth is the cached ComputeMinIntrinsicWidth.
	//
	// Flutter: rendering/box.dart RenderBox.getMinIntrinsicWidth
	GetMinIntrinsicWidth(height float64) float64
	// GetMaxIntrinsicWidth is the cached ComputeMaxIntrinsicWidth.
	//
	// Flutter: rendering/box.dart RenderBox.getMaxIntrinsicWidth
	GetMaxIntrinsicWidth(height float64) float64
	// GetMinIntrinsicHeight is the cached ComputeMinIntrinsicHeight.
	//
	// Flutter: rendering/box.dart RenderBox.getMinIntrinsicHeight
	GetMinIntrinsicHeight(width float64) float64
	// GetMaxIntrinsicHeight is the cached ComputeMaxIntrinsicHeight.
	//
	// Flutter: rendering/box.dart RenderBox.getMaxIntrinsicHeight
	GetMaxIntrinsicHeight(width float64) float64
	// GetDryLayout is the cached ComputeDryLayout.
	//
	// Flutter: rendering/box.dart RenderBox.getDryLayout
	GetDryLayout(constraints BoxConstraints) ui.Size
	// GetDryBaseline is the cached ComputeDryBaseline.
	//
	// Flutter: rendering/box.dart RenderBox.getDryBaseline
	GetDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64]
	// GetDistanceToBaseline is the distance to the baseline of the laid out box;
	// without OnlyReal it falls back to the height.
	//
	// Flutter: rendering/box.dart RenderBox.getDistanceToBaseline
	GetDistanceToBaseline(baseline ui.TextBaseline, opts ...BaselineOpts) foundation.Opt[float64]
	// GetDistanceToActualBaseline is the cached ComputeDistanceToActualBaseline.
	//
	// Flutter: rendering/box.dart RenderBox.getDistanceToActualBaseline
	GetDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64]
	// HitTest adds the box and its descendants under the position to the result.
	//
	// Flutter: rendering/box.dart RenderBox.hitTest
	HitTest(result *BoxHitTestResult, position ui.Offset) bool
	// LocalToGlobal converts a point from the box's coordinates to those of the
	// ancestor (the root if nil).
	//
	// Flutter: rendering/box.dart RenderBox.localToGlobal
	LocalToGlobal(point ui.Offset, ancestor RenderObject) ui.Offset
	// GlobalToLocal converts a point from the ancestor's coordinates (the root's
	// if nil) to the box's.
	//
	// Flutter: rendering/box.dart RenderBox.globalToLocal
	GlobalToLocal(point ui.Offset, ancestor RenderObject) ui.Offset
}

// BaselineOpts are the optional parameters of GetDistanceToBaseline.
//
// Flutter: rendering/box.dart RenderBox.getDistanceToBaseline
type BaselineOpts struct {
	// OnlyReal returns None when the box has no baseline, instead of its height.
	OnlyReal bool
}

// RenderBox is a render object that has a rectangular size and is laid out with
// BoxConstraints. A box type embeds it and calls InitRenderBox(&r.RenderBox, r).
//
// Flutter: rendering/box.dart RenderBox
type RenderBox struct {
	RenderObjectBase

	size                     foundation.Opt[ui.Size]
	debugSizeUsableByParent  bool
	layoutCache              layoutCacheStorage
	computingThisDryLayout   bool
	computingThisDryBaseline bool
	debugActivePointers      int
}

// InitRenderBox sets the outer object of a render box.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitRenderBox(b *RenderBox, self Box) { InitRenderObject(&b.RenderObjectBase, self) }

// AsRenderBox returns the box itself.
func (r *RenderBox) AsRenderBox() *RenderBox { return r }

// SetupParentData gives the child a BoxParentData if it has none of that kind.
//
// Flutter: rendering/box.dart RenderBox.setupParentData
func (r *RenderBox) SetupParentData(child RenderObject) {
	if _, ok := child.AsRenderObject().parentData.(boxParentDataHolder); !ok {
		child.AsRenderObject().parentData = &BoxParentData{}
	}
}

func (r *RenderBox) debugSelf() Box { return r.self.(Box) }

func (r *RenderBox) cachedIntrinsic(dim intrinsicDimension, input float64, computer func(float64) float64) float64 {
	if foundation.DebugMode {
		if !r.dbg().checkingIntrinsics && r.debugDoingThisResize {
			panic(errors.New(tagIntrinsicsDuringResize, "PerformResize should not depend on anything except the incoming constraints",
				"object", foundation.DescribeIdentity(r.self)))
		}
		if r.dbg().checkingIntrinsics {
			return computer(input)
		}
	}
	key := intrinsicKey{dim, input}
	if v, ok := r.layoutCache.intrinsics[key]; ok {
		return v
	}
	v := computer(input)
	if r.layoutCache.intrinsics == nil {
		r.layoutCache.intrinsics = map[intrinsicKey]float64{}
	}
	r.layoutCache.intrinsics[key] = v
	return v
}

func (r *RenderBox) debugCheckIntrinsicArgument(name string, argName string, value float64) {
	if foundation.DebugMode && value < 0 {
		panic(errors.New(tagIntrinsicArgumentNegative, "The "+argName+" argument to "+name+" was negative. "+
			"The argument must not be negative; if you perform computations on another "+argName+" before passing it, "+
			"force the value into the valid range with math.Max or a clamp.",
			"object", foundation.DescribeIdentity(r.self), "argument", value))
	}
}

// GetMinIntrinsicWidth calls ComputeMinIntrinsicWidth and caches the result. A
// negative height panics with IntrinsicArgumentNegative in debug builds.
//
// Flutter: rendering/box.dart RenderBox.getMinIntrinsicWidth
func (r *RenderBox) GetMinIntrinsicWidth(height float64) float64 {
	r.debugCheckIntrinsicArgument("GetMinIntrinsicWidth", "height", height)
	return r.cachedIntrinsic(intrinsicMinWidth, height, r.debugSelf().ComputeMinIntrinsicWidth)
}

// ComputeMinIntrinsicWidth is 0.
//
// Flutter: rendering/box.dart RenderBox.computeMinIntrinsicWidth
func (r *RenderBox) ComputeMinIntrinsicWidth(float64) float64 { return 0 }

// GetMaxIntrinsicWidth calls ComputeMaxIntrinsicWidth and caches the result. A
// negative height panics with IntrinsicArgumentNegative in debug builds.
//
// Flutter: rendering/box.dart RenderBox.getMaxIntrinsicWidth
func (r *RenderBox) GetMaxIntrinsicWidth(height float64) float64 {
	r.debugCheckIntrinsicArgument("GetMaxIntrinsicWidth", "height", height)
	return r.cachedIntrinsic(intrinsicMaxWidth, height, r.debugSelf().ComputeMaxIntrinsicWidth)
}

// ComputeMaxIntrinsicWidth is 0.
//
// Flutter: rendering/box.dart RenderBox.computeMaxIntrinsicWidth
func (r *RenderBox) ComputeMaxIntrinsicWidth(float64) float64 { return 0 }

// GetMinIntrinsicHeight calls ComputeMinIntrinsicHeight and caches the result. A
// negative width panics with IntrinsicArgumentNegative in debug builds.
//
// Flutter: rendering/box.dart RenderBox.getMinIntrinsicHeight
func (r *RenderBox) GetMinIntrinsicHeight(width float64) float64 {
	r.debugCheckIntrinsicArgument("GetMinIntrinsicHeight", "width", width)
	return r.cachedIntrinsic(intrinsicMinHeight, width, r.debugSelf().ComputeMinIntrinsicHeight)
}

// ComputeMinIntrinsicHeight is 0.
//
// Flutter: rendering/box.dart RenderBox.computeMinIntrinsicHeight
func (r *RenderBox) ComputeMinIntrinsicHeight(float64) float64 { return 0 }

// GetMaxIntrinsicHeight calls ComputeMaxIntrinsicHeight and caches the result. A
// negative width panics with IntrinsicArgumentNegative in debug builds.
//
// Flutter: rendering/box.dart RenderBox.getMaxIntrinsicHeight
func (r *RenderBox) GetMaxIntrinsicHeight(width float64) float64 {
	r.debugCheckIntrinsicArgument("GetMaxIntrinsicHeight", "width", width)
	return r.cachedIntrinsic(intrinsicMaxHeight, width, r.debugSelf().ComputeMaxIntrinsicHeight)
}

// ComputeMaxIntrinsicHeight is 0.
//
// Flutter: rendering/box.dart RenderBox.computeMaxIntrinsicHeight
func (r *RenderBox) ComputeMaxIntrinsicHeight(float64) float64 { return 0 }

// GetDryLayout calls ComputeDryLayout and caches the result.
//
// Flutter: rendering/box.dart RenderBox.getDryLayout
func (r *RenderBox) GetDryLayout(constraints BoxConstraints) ui.Size {
	if foundation.DebugMode && !r.dbg().checkingIntrinsics && r.debugDoingThisResize {
		panic(errors.New(tagIntrinsicsDuringResize, "PerformResize should not depend on anything except the incoming constraints",
			"object", foundation.DescribeIdentity(r.self)))
	}
	if foundation.DebugMode && r.dbg().checkingIntrinsics {
		return r.computeDryLayout(constraints)
	}
	if s, ok := r.layoutCache.dryLayout[constraints]; ok {
		return s
	}
	s := r.computeDryLayout(constraints)
	if r.layoutCache.dryLayout == nil {
		r.layoutCache.dryLayout = map[BoxConstraints]ui.Size{}
	}
	r.layoutCache.dryLayout[constraints] = s
	return s
}

func (r *RenderBox) computeDryLayout(constraints BoxConstraints) ui.Size {
	if foundation.DebugMode {
		if r.computingThisDryLayout {
			panic(errors.New(tagDryLayoutReentrant, "ComputeDryLayout was called while it was running",
				"object", foundation.DescribeIdentity(r.self)))
		}
		r.computingThisDryLayout = true
	}
	result := r.debugSelf().ComputeDryLayout(constraints)
	if foundation.DebugMode {
		r.computingThisDryLayout = false
	}
	return result
}

// ComputeDryLayout reports in debug builds, with the tag DryLayoutUnsupported,
// that the box does not implement it, and returns the zero size.
//
// Flutter: rendering/box.dart RenderBox.computeDryLayout
func (r *RenderBox) ComputeDryLayout(BoxConstraints) ui.Size {
	r.DebugCannotComputeDryLayout("the " + foundation.RuntimeTypeName(r.self) + ` class does not implement "ComputeDryLayout"`)
	return ui.SizeZero
}

// GetDryBaseline calls ComputeDryBaseline and caches the result.
//
// Flutter: rendering/box.dart RenderBox.getDryBaseline
func (r *RenderBox) GetDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	var cache *map[BoxConstraints]BaselineOffset
	if baseline == ui.TextBaselineIdeographic {
		cache = &r.layoutCache.ideo
	} else {
		cache = &r.layoutCache.alphabetic
	}
	var result foundation.Opt[float64]
	if foundation.DebugMode && r.dbg().checkingIntrinsics {
		result = r.computeDryBaseline(constraints, baseline).Offset
	} else if v, ok := (*cache)[constraints]; ok {
		result = v.Offset
	} else {
		result = r.computeDryBaseline(constraints, baseline).Offset
		if *cache == nil {
			*cache = map[BoxConstraints]BaselineOffset{}
		}
		(*cache)[constraints] = BaselineOffset{result}
	}
	return result
}

func (r *RenderBox) computeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) BaselineOffset {
	if foundation.DebugMode {
		if r.computingThisDryBaseline {
			panic(errors.New(tagDryLayoutReentrant, "ComputeDryBaseline was called while it was running",
				"object", foundation.DescribeIdentity(r.self)))
		}
		r.computingThisDryBaseline = true
	}
	result := BaselineOffset{r.debugSelf().ComputeDryBaseline(constraints, baseline)}
	if foundation.DebugMode {
		r.computingThisDryBaseline = false
	}
	return result
}

// ComputeDryBaseline reports in debug builds, with the tag DryLayoutUnsupported,
// that the box does not implement it, and returns None.
//
// Flutter: rendering/box.dart RenderBox.computeDryBaseline
func (r *RenderBox) ComputeDryBaseline(BoxConstraints, ui.TextBaseline) foundation.Opt[float64] {
	r.DebugCannotComputeDryLayout("the " + foundation.RuntimeTypeName(r.self) + ` class does not implement "ComputeDryBaseline"`)
	return foundation.None[float64]()
}

// DebugCannotComputeDryLayout is called by a box that can not compute its dry
// layout. Outside the intrinsic checks it panics with DryLayoutUnsupported in
// debug builds; during them it marks the dry layout as not comparable.
//
// Flutter: rendering/box.dart RenderBox.debugCannotComputeDryLayout
func (r *RenderBox) DebugCannotComputeDryLayout(reason string) bool {
	if foundation.DebugMode {
		if !r.dbg().checkingIntrinsics {
			panic(errors.New(tagDryLayoutUnsupported, "The "+foundation.RuntimeTypeName(r.self)+" class does not support dry layout. "+reason,
				"object", foundation.DescribeIdentity(r.self)))
		}
		r.dbg().dryLayoutCalculationValid = false
	}
	return true
}

// HasSize reports whether the box has a size.
//
// Flutter: rendering/box.dart RenderBox.hasSize
func (r *RenderBox) HasSize() bool { return r.size.IsSome() }

// Size is the size of the box. In debug builds it panics with BoxNotLaidOut if the
// box has no size, and with BoxSizeAccessInvalid if the size is read at a time
// the layout protocol does not allow: only the box itself, or its parent when the
// box was laid out with ParentUsesSize, may read it during layout.
//
// Flutter: rendering/box.dart RenderBox.size
func (r *RenderBox) Size() ui.Size {
	if foundation.DebugMode {
		if !r.HasSize() {
			panic(errors.New(tagBoxNotLaidOut, "RenderBox was not laid out: "+foundation.DescribeIdentity(r.self),
				"object", foundation.DescribeIdentity(r.self)))
		}
		dbg := r.dbg()
		doingRegularLayout := false
		if al := dbg.activeLayout; al != nil {
			doingRegularLayout = !al.DebugDoingThisLayoutWithCallback()
		}
		allowed := !doingRegularLayout || r.debugDoingThisResize || r.debugDoingThisLayout || dbg.doingBaseline ||
			(dbg.activeLayout != nil && dbg.activeLayout == r.parent && r.debugSizeUsableByParent)
		if !allowed {
			panic(errors.New(tagBoxSizeAccessInvalid,
				"RenderBox.size accessed beyond the scope of resize, layout, or permitted parent access. "+
					"RenderBox can always access its own size, otherwise, the only object that is allowed to read RenderBox.size "+
					`is its parent, if they have said they will. If you hit this assert trying to access a child's size, pass `+
					`"ParentUsesSize: true" to that child's Layout() in `+foundation.RuntimeTypeName(r.self)+".PerformLayout.",
				"object", foundation.DescribeIdentity(r.self)))
		}
		var dryBox Box
		if r.computingThisDryLayout {
			dryBox = r.debugSelf()
		} else if p, ok := r.parent.(Box); ok && p.AsRenderBox().computingThisDryLayout {
			dryBox = p
		}
		if dryBox != nil {
			panic(errors.New(tagBoxSizeAccessedInDryLayout, "RenderBox.size accessed in "+foundation.RuntimeTypeName(dryBox)+
				".ComputeDryLayout. The ComputeDryLayout method must not access the RenderBox's own size, or the size of its child, "+
				"because it's established in PerformLayout or PerformResize using different BoxConstraints.",
				"object", foundation.DescribeIdentity(dryBox)))
		}
		var dryBaselineBox Box
		if r.computingThisDryBaseline {
			dryBaselineBox = r.debugSelf()
		} else if p, ok := r.parent.(Box); ok && p.AsRenderBox().computingThisDryBaseline {
			dryBaselineBox = p
		}
		if dryBaselineBox != nil {
			panic(errors.New(tagBoxSizeAccessedInDryLayout, "RenderBox.size accessed in "+foundation.RuntimeTypeName(dryBaselineBox)+
				".ComputeDryBaseline. The ComputeDryBaseline method must not access the RenderBox's own size, or the size of its child, "+
				"because it's established in PerformLayout or PerformResize using different BoxConstraints.",
				"object", foundation.DescribeIdentity(dryBaselineBox)))
		}
	}
	s, ok := r.size.Get()
	if !ok {
		panic(errors.New(tagBoxNotLaidOut, "RenderBox was not laid out: "+foundation.DescribeIdentity(r.self),
			"object", foundation.DescribeIdentity(r.self)))
	}
	return s
}

// SetSize sets the size of the box. Only the box itself may call it, from
// PerformResize if it is SizedByParent and from PerformLayout otherwise; a debug
// build panics with BoxSizeSetterInvalid anywhere else.
//
// Flutter: rendering/box.dart RenderBox.size
func (r *RenderBox) SetSize(value ui.Size) {
	if foundation.DebugMode {
		sized := r.self.SizedByParent()
		if r.debugDoingThisResize && r.debugDoingThisLayout {
			panic(errors.New(tagBoxSizeSetterInvalid, "the box is resizing and laying out at the same time",
				"object", foundation.DescribeIdentity(r.self)))
		}
		if (!sized || !r.debugDoingThisResize) && (sized || !r.debugDoingThisLayout) {
			msg := "RenderBox size setter called incorrectly. "
			if r.debugDoingThisLayout {
				msg += "It appears that the size setter was called from PerformLayout(). "
			} else {
				msg += "The size setter was called from outside layout (neither PerformResize() nor PerformLayout() were being run for this object). "
				if r.owner != nil && r.owner.debugDoingLayout {
					msg += "Only the object itself can set its size. It is a contract violation for other objects to set it. "
				}
			}
			if sized {
				msg += "Because this RenderBox has SizedByParent set to true, it must set its size in PerformResize()."
			} else {
				msg += "Because this RenderBox has SizedByParent set to false, it must set its size in PerformLayout()."
			}
			panic(errors.New(tagBoxSizeSetterInvalid, msg, "object", foundation.DescribeIdentity(r.self)))
		}
		r.debugSizeUsableByParent = r.debugCanParentUseSize.OrElse(false)
	}
	r.size = foundation.Some(value)
	if foundation.DebugMode {
		r.self.DebugAssertDoesMeetConstraints()
	}
}

// DebugAdoptSize returns the size unchanged.
//
// Deviation: Flutter wraps the size to remember which box it came from, to check
// that a parent only uses sizes it asked for; sizes are plain values here (SPEC
// §12.10).
//
// Flutter: rendering/box.dart RenderBox.debugAdoptSize
func (r *RenderBox) DebugAdoptSize(value ui.Size) ui.Size { return value }

// SemanticBounds are the bounds of the box: its size at the origin.
//
// Flutter: rendering/box.dart RenderBox.semanticBounds
func (r *RenderBox) SemanticBounds() ui.Rect { return ui.OffsetZero.And(r.Size()) }

// PaintBounds are the bounds of the box: its size at the origin.
//
// Flutter: rendering/box.dart RenderBox.paintBounds
func (r *RenderBox) PaintBounds() ui.Rect { return ui.OffsetZero.And(r.Size()) }

// DebugResetSize sets the size to itself, which checks it against the
// constraints.
//
// Flutter: rendering/box.dart RenderBox.debugResetSize
func (r *RenderBox) DebugResetSize() { r.SetSize(r.Size()) }

// GetDistanceToBaseline is the distance from the top of the box to the baseline;
// without OnlyReal it is the height when the box has no baseline. It can only be
// called by the parent, during its layout or paint, with the box laid out.
//
// Flutter: rendering/box.dart RenderBox.getDistanceToBaseline
func (r *RenderBox) GetDistanceToBaseline(baseline ui.TextBaseline, opts ...BaselineOpts) foundation.Opt[float64] {
	o := firstOpts(opts)
	dbg := r.dbg()
	if foundation.DebugMode {
		if dbg.doingBaseline {
			panic(errors.New(tagBaselineReentrant, "GetDistanceToBaseline was called while a baseline was being computed; "+
				"see the documentation of ComputeDistanceToActualBaseline for the calling conventions",
				"object", foundation.DescribeIdentity(r.self)))
		}
		if r.DebugNeedsLayout() && !dbg.checkingIntrinsics {
			panic(errors.New(tagBaselineNeedsLayout, "GetDistanceToBaseline was called on a box that needs layout",
				"object", foundation.DescribeIdentity(r.self)))
		}
		if !dbg.checkingIntrinsics && r.owner != nil {
			ok := false
			switch {
			case r.owner.debugDoingLayout:
				ok = dbg.activeLayout == r.parent && r.parent != nil && r.parent.AsRenderObject().debugDoingThisLayout
			case r.owner.debugDoingPaint:
				ok = (dbg.activePaint == r.parent && r.parent != nil && r.parent.AsRenderObject().debugDoingThisPaint) ||
					(dbg.activePaint == r.self && r.debugDoingThisPaint)
			}
			if !ok {
				panic(errors.New(tagBaselineWrongPhase, "GetDistanceToBaseline can only be called by the parent during its layout or paint, or by the box during its paint",
					"object", foundation.DescribeIdentity(r.self)))
			}
		}
		dbg.doingBaseline = true
	}
	var result foundation.Opt[float64]
	func() {
		defer func() {
			if foundation.DebugMode {
				dbg.doingBaseline = false
			}
		}()
		result = r.debugSelf().GetDistanceToActualBaseline(baseline)
	}()
	if !result.IsSome() && !o.OnlyReal {
		return foundation.Some(r.Size().Height)
	}
	return result
}

// GetDistanceToActualBaseline calls ComputeDistanceToActualBaseline and caches
// the result. Only GetDistanceToBaseline and the parent's helpers call it.
//
// Flutter: rendering/box.dart RenderBox.getDistanceToActualBaseline
func (r *RenderBox) GetDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	if foundation.DebugMode && !r.dbg().doingBaseline {
		panic(errors.New(tagBaselineWrongCall, "GetDistanceToActualBaseline was called outside GetDistanceToBaseline; "+
			"see the documentation of ComputeDistanceToActualBaseline for the calling conventions",
			"object", foundation.DescribeIdentity(r.self)))
	}
	c := r.BoxConstraints()
	cache := &r.layoutCache.alphabetic
	if baseline == ui.TextBaselineIdeographic {
		cache = &r.layoutCache.ideo
	}
	if foundation.DebugMode && r.dbg().checkingIntrinsics {
		return r.debugSelf().ComputeDistanceToActualBaseline(baseline)
	}
	if v, ok := (*cache)[c]; ok {
		return v.Offset
	}
	v := r.debugSelf().ComputeDistanceToActualBaseline(baseline)
	if *cache == nil {
		*cache = map[BoxConstraints]BaselineOffset{}
	}
	(*cache)[c] = BaselineOffset{v}
	return v
}

// ComputeDistanceToActualBaseline is None: the box has no baseline.
//
// Flutter: rendering/box.dart RenderBox.computeDistanceToActualBaseline
func (r *RenderBox) ComputeDistanceToActualBaseline(ui.TextBaseline) foundation.Opt[float64] {
	if foundation.DebugMode && !r.dbg().doingBaseline {
		panic(errors.New(tagBaselineWrongCall, "ComputeDistanceToActualBaseline was called outside GetDistanceToBaseline",
			"object", foundation.DescribeIdentity(r.self)))
	}
	return foundation.None[float64]()
}

// BoxConstraints are the constraints that the parent gave at the last layout.
//
// Flutter: rendering/box.dart RenderBox.constraints
func (r *RenderBox) BoxConstraints() BoxConstraints {
	return r.Constraints().(BoxConstraints)
}

// DebugAssertDoesMeetConstraints panics in debug builds, with BoxSizeNotSet,
// BoxSizeInfinite or BoxSizeViolatesConstraints, if the size is missing,
// infinite or outside the constraints. With DebugOptions.CheckIntrinsicSizes it
// also checks the intrinsic sizes and the dry layout (BoxIntrinsicsInvalid,
// BoxDryLayoutMismatch).
//
// Flutter: rendering/box.dart RenderBox.debugAssertDoesMeetConstraints
func (r *RenderBox) DebugAssertDoesMeetConstraints() {
	if !foundation.DebugMode {
		return
	}
	self := r.debugSelf()
	size, ok := r.size.Get()
	if !ok {
		contract := "Because this RenderBox has SizedByParent set to false, it must set its size in PerformLayout()."
		if r.self.SizedByParent() {
			contract = "Because this RenderBox has SizedByParent set to true, it must set its size in PerformResize()."
		}
		panic(errors.New(tagBoxSizeNotSet, "RenderBox did not set its size during layout. "+contract+
			" It appears that this did not happen; layout completed, but the size property is still null.",
			"object", foundation.DescribeIdentity(r.self)))
	}
	c := r.BoxConstraints()
	if !size.IsFinite() {
		msg := foundation.RuntimeTypeName(r.self) + " object was given an infinite size during layout. " +
			"This probably means that it is a render object that tries to be as big as possible, but it was put inside " +
			"another render object that allows its children to pick their own size."
		meta := []any{"object", foundation.DescribeIdentity(r.self), "constraints", c, "size", size}
		if !c.HasBoundedWidth() {
			node := self
			for !node.BoxConstraints().HasBoundedWidth() {
				p, ok := node.Parent().(Box)
				if !ok {
					break
				}
				node = p
			}
			meta = append(meta, "nearestUnboundedWidth", foundation.DescribeIdentity(node))
		}
		if !c.HasBoundedHeight() {
			node := self
			for !node.BoxConstraints().HasBoundedHeight() {
				p, ok := node.Parent().(Box)
				if !ok {
					break
				}
				node = p
			}
			meta = append(meta, "nearestUnboundedHeight", foundation.DescribeIdentity(node))
		}
		panic(errors.New(tagBoxSizeInfinite, msg+" See https://flutter.dev/to/unbounded-constraints for more information.", meta...))
	}
	if !c.IsSatisfiedBy(size) {
		panic(errors.New(tagBoxSizeViolatesConstraints, foundation.RuntimeTypeName(r.self)+" does not meet its constraints. "+
			"Constraints: "+c.String()+", size: "+size.String(),
			"object", foundation.DescribeIdentity(r.self), "constraints", c, "size", size))
	}
	if r.DebugOptions().CheckIntrinsicSizes {
		r.debugCheckIntrinsics(self, c, size)
	}
}

func (r *RenderBox) debugCheckIntrinsics(self Box, c BoxConstraints, size ui.Size) {
	dbg := r.dbg()
	dbg.checkingIntrinsics = true
	var failures []string
	test := func(f func(float64) float64, name string, constraint float64) float64 {
		v := f(constraint)
		if v < 0 {
			failures = append(failures, " * "+name+"("+fixedG(constraint)+") returned a negative value: "+fixedG(v))
		}
		if math.IsInf(v, 0) || math.IsNaN(v) {
			failures = append(failures, " * "+name+"("+fixedG(constraint)+") returned a non-finite value: "+fixedG(v))
		}
		return v
	}
	testBoth := func(getMin, getMax func(float64) float64, name string, constraint float64) {
		mn := test(getMin, "getMinIntrinsic"+name, constraint)
		mx := test(getMax, "getMaxIntrinsic"+name, constraint)
		if mn > mx {
			failures = append(failures, " * getMinIntrinsic"+name+"("+fixedG(constraint)+") returned a larger value ("+fixedG(mn)+
				") than getMaxIntrinsic"+name+"("+fixedG(constraint)+") ("+fixedG(mx)+")")
		}
	}
	func() {
		defer func() { dbg.checkingIntrinsics = false }()
		testBoth(self.GetMinIntrinsicWidth, self.GetMaxIntrinsicWidth, "Width", math.Inf(1))
		testBoth(self.GetMinIntrinsicHeight, self.GetMaxIntrinsicHeight, "Height", math.Inf(1))
		if c.HasBoundedWidth() {
			testBoth(self.GetMinIntrinsicWidth, self.GetMaxIntrinsicWidth, "Width", c.MaxHeight)
		}
		if c.HasBoundedHeight() {
			testBoth(self.GetMinIntrinsicHeight, self.GetMaxIntrinsicHeight, "Height", c.MaxWidth)
		}
	}()
	if len(failures) > 0 {
		panic(errors.New(tagBoxIntrinsicsInvalid, "The intrinsic dimension methods of the "+foundation.RuntimeTypeName(r.self)+
			" class returned values that violate the intrinsic protocol contract.",
			"object", foundation.DescribeIdentity(r.self), "failures", failures))
	}
	dbg.dryLayoutCalculationValid = true
	dbg.checkingIntrinsics = true
	var dry ui.Size
	func() {
		defer func() { dbg.checkingIntrinsics = false }()
		dry = self.GetDryLayout(c)
	}()
	if dbg.dryLayoutCalculationValid && dry != size {
		panic(errors.New(tagBoxDryLayoutMismatch, "The size given to the "+foundation.RuntimeTypeName(r.self)+
			" class differs from the size computed by ComputeDryLayout. The size computed is "+size.String()+
			", which is different from "+dry.String()+", which was computed by ComputeDryLayout. The constraints used were "+c.String()+".",
			"object", foundation.DescribeIdentity(r.self)))
	}
}

func fixedG(v float64) string { return fixed1(v) }

func (r *RenderBox) debugVerifyDryBaselines() {
	self := r.debugSelf()
	dbg := r.dbg()
	c := r.BoxConstraints()
	for _, baseline := range []ui.TextBaseline{ui.TextBaselineAlphabetic, ui.TextBaselineIdeographic} {
		dbg.checkingIntrinsics = true
		dbg.dryLayoutCalculationValid = true
		var dryB, realB foundation.Opt[float64]
		func() {
			defer func() { dbg.checkingIntrinsics = false }()
			dryB = self.GetDryBaseline(c, baseline)
			realB = self.GetDistanceToBaseline(baseline, BaselineOpts{OnlyReal: true})
		}()
		if !dbg.dryLayoutCalculationValid || dryB == realB {
			continue
		}
		if dryB.IsSome() != realB.IsSome() {
			nullMethod, nonNull := "ComputeDryBaseline", "ComputeDistanceToActualBaseline"
			if dryB.IsSome() {
				nullMethod, nonNull = nonNull, nullMethod
			}
			panic(errors.New(tagBoxDryBaselineMismatch, "The "+baseline.String()+" location returned by "+
				foundation.RuntimeTypeName(r.self)+".ComputeDistanceToActualBaseline differs from the baseline location computed by ComputeDryBaseline. "+
				"The "+nullMethod+" method returned null while the "+nonNull+" returned a non-null baseline. "+
				"Did you forget to implement "+nullMethod+" for "+foundation.RuntimeTypeName(r.self)+"? The constraints used were "+c.String()+".",
				"object", foundation.DescribeIdentity(r.self)))
		}
		panic(errors.New(tagBoxDryBaselineMismatch, "The "+baseline.String()+" location returned by "+
			foundation.RuntimeTypeName(r.self)+".ComputeDistanceToActualBaseline differs from the baseline location computed by ComputeDryBaseline. "+
			"The ComputeDryBaseline method returned "+fixedG(dryB.Value())+", while the ComputeDistanceToActualBaseline method returned "+
			fixedG(realB.Value())+". The constraints used were "+c.String()+".",
			"object", foundation.DescribeIdentity(r.self)))
	}
}

// MarkNeedsLayout clears the caches of the intrinsic sizes, dry layouts and
// baselines. If there was anything cached the parent may have used it, so the
// parent needs layout too.
//
// Flutter: rendering/box.dart RenderBox.markNeedsLayout
func (r *RenderBox) MarkNeedsLayout() {
	if r.layoutCache.clear() && r.parent != nil {
		r.MarkParentNeedsLayout()
		return
	}
	r.RenderObjectBase.MarkNeedsLayout()
}

// PerformResize sets the size to the dry layout for the constraints.
//
// Flutter: rendering/box.dart RenderBox.performResize
func (r *RenderBox) PerformResize() {
	r.SetSize(r.debugSelf().ComputeDryLayout(r.BoxConstraints()))
	if foundation.DebugMode && !r.size.Value().IsFinite() {
		panic(errors.New(tagBoxSizeInfinite, "PerformResize set an infinite size", "object", foundation.DescribeIdentity(r.self)))
	}
}

// PerformLayout panics in debug builds, with BoxPerformLayoutMissing, if the box
// is not SizedByParent: it must override PerformLayout then.
//
// Flutter: rendering/box.dart RenderBox.performLayout
func (r *RenderBox) PerformLayout() {
	if foundation.DebugMode && !r.self.SizedByParent() {
		panic(errors.New(tagBoxPerformLayoutMissing, foundation.RuntimeTypeName(r.self)+" did not implement PerformLayout(). "+
			"RenderBox subclasses need to either override PerformLayout() to set a size and lay out any children, or, "+
			"set SizedByParent to true so that PerformResize() sizes the render object.",
			"object", foundation.DescribeIdentity(r.self)))
	}
}

// HitTest adds the box to the result if the position is inside its size and it
// or a child is hit. Hit testing a box that has no size panics with
// BoxHitTestNoSize in debug builds.
//
// Flutter: rendering/box.dart RenderBox.hitTest
func (r *RenderBox) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if foundation.DebugMode && !r.HasSize() {
		msg := "Cannot hit test a render box with no size. Although this node is not marked as needing layout, its size is not set."
		if r.DebugNeedsLayout() {
			msg = "Cannot hit test a render box that has never been laid out. Unfortunately, this object's geometry is not known " +
				"at this time, probably because it has never been laid out. This means it cannot be accurately hit-tested. " +
				"If you are trying to perform a hit test during the layout phase itself, make sure you only hit test nodes that " +
				"have completed layout (e.g. the node's children, after their Layout() method has been called)."
		}
		panic(errors.New(tagBoxHitTestNoSize, msg, "object", foundation.DescribeIdentity(r.self)))
	}
	self := r.debugSelf()
	if r.size.Value().Contains(position) {
		if self.HitTestChildren(result, position) || self.HitTestSelf(position) {
			result.Add(NewBoxHitTestEntry(self, position))
			return true
		}
	}
	return false
}

// HitTestSelf is false: the box is not hit by itself.
//
// Flutter: rendering/box.dart RenderBox.hitTestSelf
func (r *RenderBox) HitTestSelf(ui.Offset) bool { return false }

// HitTestChildren is false: no child is hit.
//
// Flutter: rendering/box.dart RenderBox.hitTestChildren
func (r *RenderBox) HitTestChildren(*BoxHitTestResult, ui.Offset) bool { return false }

// ApplyPaintTransform translates the transform by the offset in the child's
// BoxParentData. A child with other parent data panics in debug builds with
// ApplyPaintTransformParentData.
//
// Flutter: rendering/box.dart RenderBox.applyPaintTransform
func (r *RenderBox) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if foundation.DebugMode {
		if child.AsRenderObject().parent != r.self {
			panic(errors.New(tagRenderObjectNotChild, "ApplyPaintTransform was called with a render object that is not a child",
				"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
		}
		if _, ok := child.AsRenderObject().parentData.(boxParentDataHolder); !ok {
			panic(errors.New(tagApplyPaintTransformParentData, foundation.RuntimeTypeName(r.self)+
				" does not implement ApplyPaintTransform: the child did not use a BoxParentData for its parent data. "+
				"The default ApplyPaintTransform of RenderBox assumes that the children all use BoxParentData objects; "+
				"a class that uses other parent data must provide its own.",
				"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
		}
	}
	offset := child.AsRenderObject().parentData.(boxParentDataHolder).boxParentData().Offset
	transform.Translate(float32(offset.Dx), float32(offset.Dy), 0)
}

// GlobalToLocal converts a point from the global coordinates (those of the
// ancestor, or of the root if nil) to the box's.
//
// Flutter: rendering/box.dart RenderBox.globalToLocal
func (r *RenderBox) GlobalToLocal(point ui.Offset, ancestor RenderObject) ui.Offset {
	transform := r.GetTransformTo(ancestor)
	if transform.Invert() == 0 {
		return ui.OffsetZero
	}
	localScreenOrigin := transform.PerspectiveTransform(ui.Vector3{})
	localViewDirection := transform.PerspectiveTransform(ui.Vector3{Z: 1})
	localViewDirection = ui.Vector3{X: localViewDirection.X - localScreenOrigin.X, Y: localViewDirection.Y - localScreenOrigin.Y,
		Z: localViewDirection.Z - localScreenOrigin.Z}
	if localViewDirection.Z == 0 {
		return ui.OffsetZero
	}
	p := transform.PerspectiveTransform(ui.Vector3{X: float32(point.Dx), Y: float32(point.Dy)})
	k := p.Z / localViewDirection.Z
	return ui.Offset{Dx: float64(p.X - localViewDirection.X*k), Dy: float64(p.Y - localViewDirection.Y*k)}
}

// LocalToGlobal converts a point from the box's coordinates to the global ones
// (those of the ancestor, or of the root if nil).
//
// Flutter: rendering/box.dart RenderBox.localToGlobal
func (r *RenderBox) LocalToGlobal(point ui.Offset, ancestor RenderObject) ui.Offset {
	return r.GetTransformTo(ancestor).TransformPoint(point)
}

// HandleEvent does nothing here; DebugHandleEvent is for the pointer paint
// debugging.
//
// Flutter: rendering/box.dart RenderBox.handleEvent
func (r *RenderBox) HandleEvent(gestures.PointerEvent, gestures.HitTestEntry) {}

// DebugHandleEvent counts the pointers that are down, to paint them when
// DebugOptions.PaintPointersEnabled is set. It returns true.
//
// Flutter: rendering/box.dart RenderBox.debugHandleEvent
func (r *RenderBox) DebugHandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) bool {
	if foundation.DebugMode && r.DebugOptions().PaintPointersEnabled {
		switch event.(type) {
		case *gestures.PointerDownEvent:
			r.debugActivePointers++
		case *gestures.PointerUpEvent, *gestures.PointerCancelEvent:
			r.debugActivePointers--
		}
		r.MarkNeedsPaint()
	}
	return true
}

// DebugPaint paints the debug overlays that DebugOptions asks for.
//
// Flutter: rendering/box.dart RenderBox.debugPaint
func (r *RenderBox) DebugPaint(context *PaintingContext, offset ui.Offset) {
	if !foundation.DebugMode {
		return
	}
	o := r.DebugOptions()
	if o.CheckIntrinsicSizes {
		r.debugVerifyDryBaselines()
	}
	if o.PaintSizeEnabled {
		r.self.(interface {
			DebugPaintSize(*PaintingContext, ui.Offset)
		}).DebugPaintSize(context, offset)
	}
	if o.PaintBaselinesEnabled {
		r.DebugPaintBaselines(context, offset)
	}
	if o.PaintPointersEnabled {
		r.DebugPaintPointers(context, offset)
	}
}

// DebugPaintSize paints a cyan outline of the box.
//
// Flutter: rendering/box.dart RenderBox.debugPaintSize
func (r *RenderBox) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	if !foundation.DebugMode {
		return
	}
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(1)
	paint.SetColor(ui.NewColorFromARGB(0xFF, 0x00, 0xFF, 0xFF))
	context.Canvas().DrawRect(offset.And(r.Size()).Deflate(0.5), paint)
}

// DebugPaintBaselines paints the baselines: ideographic in yellow, alphabetic in
// green.
//
// Flutter: rendering/box.dart RenderBox.debugPaintBaselines
func (r *RenderBox) DebugPaintBaselines(context *PaintingContext, offset ui.Offset) {
	if !foundation.DebugMode {
		return
	}
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(0.25)
	line := func(y float64, c ui.Color) {
		paint.SetColor(c)
		path := ui.NewPath()
		path.MoveTo(offset.Dx, offset.Dy+y)
		path.LineTo(offset.Dx+r.Size().Width, offset.Dy+y)
		context.Canvas().DrawPath(path, paint)
	}
	self := r.debugSelf()
	if b, ok := self.GetDistanceToBaseline(ui.TextBaselineIdeographic, BaselineOpts{OnlyReal: true}).Get(); ok {
		line(b, ui.NewColorFromARGB(0xFF, 0xFF, 0xD0, 0x00))
	}
	if b, ok := self.GetDistanceToBaseline(ui.TextBaselineAlphabetic, BaselineOpts{OnlyReal: true}).Get(); ok {
		line(b, ui.NewColorFromARGB(0xFF, 0x00, 0xFF, 0x00))
	}
}

// DebugPaintPointers paints a color over the box while pointers are down on it.
//
// Flutter: rendering/box.dart RenderBox.debugPaintPointers
func (r *RenderBox) DebugPaintPointers(context *PaintingContext, offset ui.Offset) {
	if !foundation.DebugMode || r.debugActivePointers <= 0 {
		return
	}
	alpha := uint32((0x04000000 * r.depth) & 0xFF000000)
	paint := ui.NewPaint()
	paint.SetColor(ui.NewColor(0x00BBBB | alpha))
	context.Canvas().DrawRect(offset.And(r.Size()), paint)
}

// DebugFillProperties adds the size.
//
// Flutter: rendering/box.dart RenderBox.debugFillProperties
func (r *RenderBox) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	r.RenderObjectBase.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsPropertyOpt("size", r.size, foundation.DiagnosticsPropertyOpts{MissingIfNull: true}))
}

// ToStringShort is the description of the object; render boxes use the base one.
func (r *RenderBox) ToStringShort() string { return r.RenderObjectBase.ToStringShort() }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagApplyPaintTransformParentData = "ApplyPaintTransformParentData"
	tagBaselineNeedsLayout           = "BaselineNeedsLayout"
	tagBaselineReentrant             = "BaselineReentrant"
	tagBaselineWrongCall             = "BaselineWrongCall"
	tagBaselineWrongPhase            = "BaselineWrongPhase"
	tagBoxDryBaselineMismatch        = "BoxDryBaselineMismatch"
	tagBoxDryLayoutMismatch          = "BoxDryLayoutMismatch"
	tagBoxHitTestNoSize              = "BoxHitTestNoSize"
	tagBoxIntrinsicsInvalid          = "BoxIntrinsicsInvalid"
	tagBoxNotLaidOut                 = "BoxNotLaidOut"
	tagBoxPerformLayoutMissing       = "BoxPerformLayoutMissing"
	tagBoxSizeAccessInvalid          = "BoxSizeAccessInvalid"
	tagBoxSizeAccessedInDryLayout    = "BoxSizeAccessedInDryLayout"
	tagBoxSizeInfinite               = "BoxSizeInfinite"
	tagBoxSizeNotSet                 = "BoxSizeNotSet"
	tagBoxSizeSetterInvalid          = "BoxSizeSetterInvalid"
	tagBoxSizeViolatesConstraints    = "BoxSizeViolatesConstraints"
	tagDryLayoutReentrant            = "DryLayoutReentrant"
	tagDryLayoutUnsupported          = "DryLayoutUnsupported"
	tagIntrinsicArgumentNegative     = "IntrinsicArgumentNegative"
	tagIntrinsicsDuringResize        = "IntrinsicsDuringResize"
	tagRenderObjectNotChild          = "RenderObjectNotChild"
)
