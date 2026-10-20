// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// MergeableMaterialItem is a child of a MergeableMaterial: a MaterialSlice or a
// MaterialGap.
//
// Flutter: material/mergeable_material.dart MergeableMaterialItem
type MergeableMaterialItem interface {
	// ItemKey is the key of the item.
	//
	// Flutter: material/mergeable_material.dart MergeableMaterialItem.key
	ItemKey() foundation.LocalKey
}

// MaterialSlice is a piece of material in a MergeableMaterial.
//
// Flutter: material/mergeable_material.dart MaterialSlice
type MaterialSlice struct {
	key   foundation.LocalKey
	child widgets.Widget
	color foundation.Opt[ui.Color]
}

// NewMaterialSlice creates a slice with the key and the child; the color is
// the card color of the theme when it is not set.
//
// Flutter: material/mergeable_material.dart MaterialSlice.new
func NewMaterialSlice(key foundation.LocalKey, child widgets.Widget, color ...foundation.Opt[ui.Color]) *MaterialSlice {
	return &MaterialSlice{key: key, child: child, color: firstOpt(color...)}
}

// ItemKey is the key of the slice.
//
// Flutter: material/mergeable_material.dart MergeableMaterialItem.key
func (s *MaterialSlice) ItemKey() foundation.LocalKey { return s.key }

// Child is the content of the slice.
//
// Flutter: material/mergeable_material.dart MaterialSlice.child
func (s *MaterialSlice) Child() widgets.Widget { return s.child }

// MaterialGap is a gap between the slices of a MergeableMaterial.
//
// Flutter: material/mergeable_material.dart MaterialGap
type MaterialGap struct {
	key  foundation.LocalKey
	size float64
}

// NewMaterialGap creates a gap with the key and the size (default 16).
//
// Flutter: material/mergeable_material.dart MaterialGap.new
func NewMaterialGap(key foundation.LocalKey, size ...float64) *MaterialGap {
	s := 16.0
	if len(size) > 0 {
		s = size[0]
	}
	return &MaterialGap{key: key, size: s}
}

// ItemKey is the key of the gap.
//
// Flutter: material/mergeable_material.dart MergeableMaterialItem.key
func (g *MaterialGap) ItemKey() foundation.LocalKey { return g.key }

// Size is the main axis extent of the gap.
//
// Flutter: material/mergeable_material.dart MaterialGap.size
func (g *MaterialGap) Size() float64 { return g.size }

// MergeableMaterialOpts are the optional parameters of NewMergeableMaterial.
//
// Flutter: material/mergeable_material.dart MergeableMaterial.new
type MergeableMaterialOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MainAxis is the direction to use as the main axis; the default is vertical.
	MainAxis painting.Axis
	// Elevation is the z-coordinate at which to place the slices; the default is 2.
	Elevation foundation.Opt[float64]
	// HasDividers draws dividers between the slices.
	HasDividers bool
	// DividerColor is the color of the dividers.
	DividerColor foundation.Opt[ui.Color]
}

// MergeableMaterial is a vertical or horizontal list of slices of material that
// merge, and that split apart with a gap that animates in and out.
//
// Flutter: material/mergeable_material.dart MergeableMaterial
type MergeableMaterial struct {
	widgets.StatefulWidget
	children []MergeableMaterialItem
	o        MergeableMaterialOpts
}

// NewMergeableMaterial creates the widget. The tag of its failure is
// MergeableMaterialInvalidGaps (a gap at the start or the end, or two gaps in a
// row).
//
// Flutter: material/mergeable_material.dart MergeableMaterial.new
func NewMergeableMaterial(children []MergeableMaterialItem, opts ...MergeableMaterialOpts) *MergeableMaterial {
	o := firstOpts(opts)
	if o.MainAxis == painting.AxisUnspecified {
		o.MainAxis = painting.AxisVertical
	}
	return &MergeableMaterial{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), children: children, o: o}
}

// CreateState creates the state.
//
// Flutter: material/mergeable_material.dart MergeableMaterial.createState
func (w *MergeableMaterial) CreateState() widgets.State { return &mergeableMaterialState{} }

// mergeableAnimationTuple is the animations of a gap.
//
// Flutter: material/mergeable_material.dart _AnimationTuple
type mergeableAnimationTuple struct {
	controller     *animation.AnimationController
	startAnimation *animation.CurvedAnimation
	endAnimation   *animation.CurvedAnimation
	gapAnimation   *animation.CurvedAnimation
	gapStart       float64
}

func (t *mergeableAnimationTuple) dispose() {
	t.controller.Dispose()
	t.startAnimation.Dispose()
	t.endAnimation.Dispose()
	t.gapAnimation.Dispose()
}

type mergeableMaterialState struct {
	widgets.StateOf[*MergeableMaterial]
	widgets.TickerProvider
	children        []MergeableMaterialItem
	animationTuples map[foundation.LocalKey]*mergeableAnimationTuple
	sliceKeys       map[foundation.LocalKey]*widgets.GlobalKey[widgets.State]
}

func (s *mergeableMaterialState) InitState() {
	w := s.Widget()
	s.animationTuples = map[foundation.LocalKey]*mergeableAnimationTuple{}
	s.sliceKeys = map[foundation.LocalKey]*widgets.GlobalKey[widgets.State]{}
	s.children = append([]MergeableMaterialItem(nil), w.children...)
	for _, child := range s.children {
		if gap, ok := child.(*MaterialGap); ok {
			s.initGap(gap)
			s.animationTuples[gap.key].controller.SetValue(1) // Gaps are initially full-sized.
		}
	}
	s.debugGapsAreValid(s.children)
}

func (s *mergeableMaterialState) initGap(gap *MaterialGap) {
	controller := animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(ThemeAnimationDuration)})
	t := &mergeableAnimationTuple{
		controller:     controller,
		startAnimation: animation.NewCurvedAnimation(controller, animation.Curves.FastOutSlowIn),
		endAnimation:   animation.NewCurvedAnimation(controller, animation.Curves.FastOutSlowIn),
		gapAnimation:   animation.NewCurvedAnimation(controller, animation.Curves.FastOutSlowIn),
	}
	controller.AddListener(func() { s.SetState(func() {}) })
	s.animationTuples[gap.key] = t
}

func (s *mergeableMaterialState) Dispose() {
	for _, child := range s.children {
		if gap, ok := child.(*MaterialGap); ok {
			if t := s.animationTuples[gap.key]; t != nil {
				t.dispose()
			}
		}
	}
}

// debugGapsAreValid panics with the tag MergeableMaterialInvalidGaps when there
// are consecutive gaps or a gap at the start or the end of the children.
//
// Flutter: material/mergeable_material.dart _MergeableMaterialState._debugGapsAreValid
func (s *mergeableMaterialState) debugGapsAreValid(children []MergeableMaterialItem) {
	if !foundation.DebugMode {
		return
	}
	isGap := func(i int) bool { _, ok := children[i].(*MaterialGap); return ok }
	invalid := false
	for i := 0; i < len(children)-1; i++ {
		if isGap(i) && isGap(i+1) {
			invalid = true
		}
	}
	if len(children) > 0 && (isGap(0) || isGap(len(children)-1)) {
		invalid = true
	}
	if invalid {
		panic(errors.New("MergeableMaterialInvalidGaps", "a MergeableMaterial can't have a gap at its start or its end, or two gaps in a row"))
	}
}

func (s *mergeableMaterialState) insertChild(index int, child MergeableMaterialItem) {
	s.children = append(s.children, nil)
	copy(s.children[index+1:], s.children[index:])
	s.children[index] = child
	if gap, ok := child.(*MaterialGap); ok {
		s.initGap(gap)
	}
}

func (s *mergeableMaterialState) removeChild(index int) {
	child := s.children[index]
	s.children = append(s.children[:index], s.children[index+1:]...)
	if gap, ok := child.(*MaterialGap); ok {
		s.animationTuples[gap.key].dispose()
		delete(s.animationTuples, gap.key)
	}
}

func (s *mergeableMaterialState) isClosingGap(index int) bool {
	if index < len(s.children)-1 {
		if _, ok := s.children[index].(*MaterialGap); ok {
			return s.animationTuples[s.children[index].ItemKey()].controller.Status() == animation.AnimationStatusReverse
		}
	}
	return false
}

func (s *mergeableMaterialState) removeEmptyGaps() {
	for j := len(s.children) - 1; j >= 0; j-- {
		if _, ok := s.children[j].(*MaterialGap); ok && s.animationTuples[s.children[j].ItemKey()].controller.Status() == animation.AnimationStatusDismissed {
			s.removeChild(j)
		}
	}
}

func isMaterialGap(item MergeableMaterialItem) bool { _, ok := item.(*MaterialGap); return ok }

// DidUpdateWidget works out how the children changed, and animates the gaps.
//
// Flutter: material/mergeable_material.dart _MergeableMaterialState.didUpdateWidget
func (s *mergeableMaterialState) DidUpdateWidget(old *MergeableMaterial) {
	w := s.Widget()
	oldKeys, newKeys := map[foundation.LocalKey]bool{}, map[foundation.LocalKey]bool{}
	for _, c := range old.children {
		oldKeys[c.ItemKey()] = true
	}
	for _, c := range w.children {
		newKeys[c.ItemKey()] = true
	}
	newOnly := func(k foundation.LocalKey) bool { return newKeys[k] && !oldKeys[k] }
	oldOnly := func(k foundation.LocalKey) bool { return oldKeys[k] && !newKeys[k] }
	newChildren := w.children
	i, j := 0, 0
	s.debugGapsAreValid(newChildren)
	s.removeEmptyGaps()
	for i < len(newChildren) && j < len(s.children) {
		if newOnly(newChildren[i].ItemKey()) || oldOnly(s.children[j].ItemKey()) {
			startNew, startOld := i, j
			for i < len(newChildren) && newOnly(newChildren[i].ItemKey()) {
				i++
			}
			for j < len(s.children) && (oldOnly(s.children[j].ItemKey()) || s.isClosingGap(j)) {
				j++
			}
			newLength, oldLength := i-startNew, j-startOld
			_, startOldIsSlice := s.children[startOld].(*MaterialSlice)
			if newLength > 0 {
				switch {
				case oldLength > 1 || oldLength == 1 && startOldIsSlice:
					if newLength == 1 && isMaterialGap(newChildren[startNew]) {
						gapSizeSum := 0.0
						for startOld < j {
							if gap, ok := s.children[startOld].(*MaterialGap); ok {
								gapSizeSum += gap.size
							}
							s.removeChild(startOld)
							j--
						}
						s.insertChild(startOld, newChildren[startNew])
						t := s.animationTuples[newChildren[startNew].ItemKey()]
						t.gapStart = gapSizeSum
						t.controller.Forward()
						j++
					} else {
						for k := 0; k < oldLength; k++ {
							s.removeChild(startOld)
						}
						for k := 0; k < newLength; k++ {
							s.insertChild(startOld+k, newChildren[startNew+k])
						}
						j += newLength - oldLength
					}
				case oldLength == 1:
					if newLength == 1 && isMaterialGap(newChildren[startNew]) && s.children[startOld].ItemKey() == newChildren[startNew].ItemKey() {
						s.animationTuples[newChildren[startNew].ItemKey()].controller.Forward()
					} else {
						gapSize := s.getGapSize(startOld)
						s.removeChild(startOld)
						for k := 0; k < newLength; k++ {
							s.insertChild(startOld+k, newChildren[startNew+k])
						}
						j += newLength - 1
						gapSizeSum := 0.0
						for k := startNew; k < i; k++ {
							if gap, ok := newChildren[k].(*MaterialGap); ok {
								gapSizeSum += gap.size
							}
						}
						for k := startNew; k < i; k++ {
							if gap, ok := newChildren[k].(*MaterialGap); ok {
								t := s.animationTuples[gap.key]
								t.gapStart = gapSize * gap.size / gapSizeSum
								t.controller.SetValue(0)
								t.controller.Forward()
							}
						}
					}
				default:
					for k := 0; k < newLength; k++ {
						newChild := newChildren[startNew+k]
						s.insertChild(startOld+k, newChild)
						if gap, ok := newChild.(*MaterialGap); ok {
							s.animationTuples[gap.key].controller.Forward()
						}
					}
					j += newLength
				}
			} else {
				switch {
				case oldLength > 1 || oldLength == 1 && startOldIsSlice:
					gapSizeSum := 0.0
					for startOld < j {
						if gap, ok := s.children[startOld].(*MaterialGap); ok {
							gapSizeSum += gap.size
						}
						s.removeChild(startOld)
						j--
					}
					if gapSizeSum != 0 {
						gap := NewMaterialGap(foundation.NewUniqueKey(), gapSizeSum)
						s.insertChild(startOld, gap)
						t := s.animationTuples[gap.key]
						t.gapStart = 0
						t.controller.SetValue(1)
						t.controller.Reverse()
						j++
					}
				case oldLength == 1:
					gap := s.children[startOld].(*MaterialGap)
					t := s.animationTuples[gap.key]
					t.gapStart = 0
					t.controller.Reverse()
				}
			}
		} else {
			if isMaterialGap(s.children[j]) == isMaterialGap(newChildren[i]) {
				s.children[j] = newChildren[i]
				i++
				j++
			} else {
				// A closing gap is skipped.
				j++
			}
		}
	}
	for j < len(s.children) {
		s.removeChild(j)
	}
	for i < len(newChildren) {
		newChild := newChildren[i]
		s.insertChild(j, newChild)
		if gap, ok := newChild.(*MaterialGap); ok {
			s.animationTuples[gap.key].controller.Forward()
		}
		i++
		j++
	}
}

func (s *mergeableMaterialState) borderRadius(index int, start, end bool) painting.BorderRadius {
	cardRadius := MaterialEdges[MaterialTypeCard].TopLeft
	startRadius, endRadius := ui.RadiusZero, ui.RadiusZero
	if index > 0 && isMaterialGap(s.children[index-1]) {
		zero := ui.RadiusZero
		startRadius = *ui.RadiusLerp(&zero, &cardRadius, s.animationTuples[s.children[index-1].ItemKey()].startAnimation.Value())
	}
	if index < len(s.children)-2 && isMaterialGap(s.children[index+1]) {
		zero := ui.RadiusZero
		endRadius = *ui.RadiusLerp(&zero, &cardRadius, s.animationTuples[s.children[index+1].ItemKey()].endAnimation.Value())
	}
	first, last := startRadius, endRadius
	if start {
		first = cardRadius
	}
	if end {
		last = cardRadius
	}
	if s.Widget().o.MainAxis == painting.AxisVertical {
		return painting.NewBorderRadiusVertical(first, last)
	}
	return painting.NewBorderRadiusHorizontal(first, last)
}

func (s *mergeableMaterialState) getGapSize(index int) float64 {
	gap := s.children[index].(*MaterialGap)
	t := s.animationTuples[gap.key]
	return t.gapStart + (gap.size-t.gapStart)*t.gapAnimation.Value()
}

func (s *mergeableMaterialState) willNeedDivider(index int) bool {
	if index < 0 || index >= len(s.children) {
		return false
	}
	_, isSlice := s.children[index].(*MaterialSlice)
	return isSlice || s.isClosingGap(index)
}

func (s *mergeableMaterialState) sliceKey(k foundation.LocalKey) *widgets.GlobalKey[widgets.State] {
	if g, ok := s.sliceKeys[k]; ok {
		return g
	}
	g := widgets.NewGlobalKey[widgets.State]()
	s.sliceKeys[k] = g
	return g
}

// Build builds the slices, grouped between the gaps.
//
// Flutter: material/mergeable_material.dart _MergeableMaterialState.build
func (s *mergeableMaterialState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	s.removeEmptyGaps()
	var items, slices []widgets.Widget
	for i, child := range s.children {
		if _, ok := child.(*MaterialGap); ok {
			items = append(items, widgets.NewListBody(slices, widgets.ListBodyOpts{MainAxis: o.MainAxis}))
			slices = nil
			if o.MainAxis == painting.AxisHorizontal {
				items = append(items, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(s.getGapSize(i))}))
			} else {
				items = append(items, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(s.getGapSize(i))}))
			}
			continue
		}
		slice := child.(*MaterialSlice)
		content := slice.child
		if o.HasDividers {
			hasTopDivider, hasBottomDivider := s.willNeedDivider(i-1), s.willNeedDivider(i+1)
			divider := DividerCreateBorderSide(bc, DividerBorderSideOpts{Width: foundation.Some(0.5), Color: o.DividerColor})
			var top, bottom painting.BorderSide
			if hasTopDivider && i != 0 {
				top = divider
			}
			if hasBottomDivider && i != len(s.children)-1 {
				bottom = divider
			}
			content = widgets.NewAnimatedContainer(ThemeAnimationDuration, content, widgets.AnimatedContainerOpts{
				Key:        s.sliceKey(child.ItemKey()),
				Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: painting.Border{Top: top, Bottom: bottom}}),
				Curve:      animation.Curves.FastOutSlowIn,
			})
		}
		slices = append(slices, widgets.NewContainer(NewMaterial(content, MaterialOpts{Type: MaterialTypeTransparency}), widgets.ContainerOpts{
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
				Color:        firstOpt(slice.color, foundation.Some(ThemeOf(bc).CardColor)),
				BorderRadius: s.borderRadius(i, i == 0, i == len(s.children)-1),
			}),
		}))
	}
	if len(slices) > 0 {
		items = append(items, widgets.NewListBody(slices, widgets.ListBodyOpts{MainAxis: o.MainAxis}))
	}
	return &mergeableMaterialListBody{
		MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(items),
		mainAxis:                     o.MainAxis, elevation: o.Elevation.OrElse(2),
	}
}

// mergeableMaterialListBody lays out the groups of slices and the gaps between
// them, and paints the shadows of the groups.
//
// Flutter: material/mergeable_material.dart _MergeableMaterialListBody
type mergeableMaterialListBody struct {
	widgets.MultiChildRenderObjectWidget
	mainAxis  painting.Axis
	elevation float64
}

func (w *mergeableMaterialListBody) direction(bc widgets.BuildContext) painting.AxisDirection {
	return widgets.AxisDirectionFromAxisReverseAndDirectionality(bc, w.mainAxis, false)
}

func (w *mergeableMaterialListBody) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderMergeableMaterialListBody{elevation: w.elevation}
	rendering.InitRenderListBody(&r.RenderListBody, r, rendering.RenderListBodyOpts{AxisDirection: w.direction(bc)})
	return r
}

func (w *mergeableMaterialListBody) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderMergeableMaterialListBody)
	r.SetAxisDirection(w.direction(bc))
	if r.elevation != w.elevation {
		r.elevation = w.elevation
		r.MarkNeedsPaint()
	}
}

// renderMergeableMaterialListBody is the render object of
// mergeableMaterialListBody.
//
// Flutter: material/mergeable_material.dart _RenderMergeableMaterialListBody
type renderMergeableMaterialListBody struct {
	rendering.RenderListBody
	elevation float64
}

func (r *renderMergeableMaterialListBody) paintShadows(canvas *ui.Canvas, rect ui.Rect) {
	if r.elevation != 0 {
		path := ui.NewPath()
		path.AddRRect(MaterialEdges[MaterialTypeCard].ToRRect(rect))
		// The occluding object is not (necessarily) opaque.
		canvas.DrawShadow(path, Colors.Black, r.elevation, true)
	}
}

// Paint paints the shadows of the groups, then the children.
//
// Flutter: material/mergeable_material.dart _RenderMergeableMaterialListBody.paint
func (r *renderMergeableMaterialListBody) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	index := 0
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		o := child.ParentData().(*rendering.ListBodyParentData).Offset.Add(offset)
		rect := ui.NewRectFromLTWH(o.Dx, o.Dy, child.Size().Width, child.Size().Height)
		if index%2 == 0 {
			r.paintShadows(context.Canvas(), rect)
		}
		index++
	}
	r.RenderListBody.Paint(context, offset)
}
