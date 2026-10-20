// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ChildLayouter lays out a child with the constraints and returns its size.
//
// Flutter: rendering/layout_helper.dart ChildLayouter
type ChildLayouter func(child Box, constraints BoxConstraints) ui.Size

// ChildBaselineGetter returns the baseline of a child for the constraints.
//
// Flutter: rendering/layout_helper.dart ChildBaselineGetter
type ChildBaselineGetter func(child Box, constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64]

// ChildLayoutHelper has the functions that a render box uses to lay out a child
// and measure it, the real way and the dry way. The pairs can be passed as the
// ChildLayouter and ChildBaselineGetter of a layout algorithm that has a dry and
// a real mode.
//
// Flutter: rendering/layout_helper.dart ChildLayoutHelper
var ChildLayoutHelper = childLayoutHelper{}

type childLayoutHelper struct{}

// DryLayoutChild returns the size the child would have for the constraints,
// without laying it out.
//
// Flutter: rendering/layout_helper.dart ChildLayoutHelper.dryLayoutChild
func (childLayoutHelper) DryLayoutChild(child Box, constraints BoxConstraints) ui.Size {
	return child.GetDryLayout(constraints)
}

// LayoutChild lays out the child with the constraints, telling it that the
// parent uses its size, and returns the size.
//
// Flutter: rendering/layout_helper.dart ChildLayoutHelper.layoutChild
func (childLayoutHelper) LayoutChild(child Box, constraints BoxConstraints) ui.Size {
	child.Layout(constraints, LayoutOpts{ParentUsesSize: true})
	return child.Size()
}

// GetDryBaseline returns the baseline the child would have for the constraints.
//
// Flutter: rendering/layout_helper.dart ChildLayoutHelper.getDryBaseline
func (childLayoutHelper) GetDryBaseline(child Box, constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	return child.GetDryBaseline(constraints, baseline)
}

// GetBaseline returns the real baseline of a child that was laid out with the
// constraints. A debug build panics with ChildBaselineInvalid if the child needs
// layout or has other constraints.
//
// Flutter: rendering/layout_helper.dart ChildLayoutHelper.getBaseline
func (childLayoutHelper) GetBaseline(child Box, constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if foundation.DebugMode && (child.DebugNeedsLayout() || child.BoxConstraints() != constraints) {
		panic(errors.New("ChildBaselineInvalid", "the baseline of a child can only be read after it was laid out with the same constraints",
			"child", foundation.DescribeIdentity(child)))
	}
	return child.GetDistanceToBaseline(baseline, BaselineOpts{OnlyReal: true})
}
