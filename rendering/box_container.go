package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// The functions below are the defaults that RenderBoxContainerDefaultsMixin gives
// a render box with a list of children whose parent data embeds
// ContainerBoxParentData. The box passes its ContainerRenderObject.

func boxChildOffset[C Box](child C) ui.Offset {
	pd, ok := child.ParentData().(boxParentDataHolder)
	if !ok {
		panic(errors.New(tagContainerBoxParentDataMissing, "a child's parent data does not embed BoxParentData",
			"child", foundation.DescribeIdentity(child)))
	}
	return pd.boxParentData().Offset
}

// DefaultComputeDistanceToFirstActualBaseline is the distance to the baseline of
// the first child that has one, plus the child's vertical offset.
//
// Flutter: rendering/box.dart RenderBoxContainerDefaultsMixin.defaultComputeDistanceToFirstActualBaseline
func DefaultComputeDistanceToFirstActualBaseline[C Box](c *ContainerRenderObject[C], baseline ui.TextBaseline) foundation.Opt[float64] {
	if foundation.DebugMode && c.host.DebugNeedsLayout() {
		panic(errors.New(tagBaselineNeedsLayout, "the baseline was read from a box that needs layout",
			"object", foundation.DescribeIdentity(c.host)))
	}
	for child := c.firstChild; !isNil(child); child = c.data(child).NextSibling {
		if v, ok := child.GetDistanceToActualBaseline(baseline).Get(); ok {
			return foundation.Some(v + boxChildOffset(child).Dy)
		}
	}
	return foundation.None[float64]()
}

// DefaultComputeDistanceToHighestActualBaseline is the smallest distance to a
// baseline among the children, plus the children's vertical offsets.
//
// Flutter: rendering/box.dart RenderBoxContainerDefaultsMixin.defaultComputeDistanceToHighestActualBaseline
func DefaultComputeDistanceToHighestActualBaseline[C Box](c *ContainerRenderObject[C], baseline ui.TextBaseline) foundation.Opt[float64] {
	if foundation.DebugMode && c.host.DebugNeedsLayout() {
		panic(errors.New(tagBaselineNeedsLayout, "the baseline was read from a box that needs layout",
			"object", foundation.DescribeIdentity(c.host)))
	}
	min := NoBaseline
	for child := c.firstChild; !isNil(child); child = c.data(child).NextSibling {
		candidate := NewBaselineOffset(child.GetDistanceToActualBaseline(baseline)).Add(boxChildOffset(child).Dy)
		min = min.MinOf(candidate)
	}
	return min.Offset
}

// DefaultHitTestChildren hit tests the children from the last (front) to the
// first, each at its offset, and stops at the first hit.
//
// Flutter: rendering/box.dart RenderBoxContainerDefaultsMixin.defaultHitTestChildren
func DefaultHitTestChildren[C Box](c *ContainerRenderObject[C], result *BoxHitTestResult, position ui.Offset) bool {
	for child := c.lastChild; !isNil(child); child = c.data(child).PreviousSibling {
		child := child
		offset := boxChildOffset(child)
		isHit := result.AddWithPaintOffset(&offset, position, func(result *BoxHitTestResult, transformed ui.Offset) bool {
			return child.HitTest(result, transformed)
		})
		if isHit {
			return true
		}
	}
	return false
}

// DefaultPaint paints the children in order, each at its offset.
//
// Flutter: rendering/box.dart RenderBoxContainerDefaultsMixin.defaultPaint
func DefaultPaint[C Box](c *ContainerRenderObject[C], context *PaintingContext, offset ui.Offset) {
	for child := c.firstChild; !isNil(child); child = c.data(child).NextSibling {
		context.PaintChild(child, boxChildOffset(child).Add(offset))
	}
}

// GetChildrenAsList returns the children in order.
//
// Flutter: rendering/box.dart RenderBoxContainerDefaultsMixin.getChildrenAsList
func GetChildrenAsList[C Box](c *ContainerRenderObject[C]) []C {
	var out []C
	for child := c.firstChild; !isNil(child); child = c.data(child).NextSibling {
		out = append(out, child)
	}
	return out
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagContainerBoxParentDataMissing = "ContainerBoxParentDataMissing"
)
