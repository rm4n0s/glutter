// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The fraction of the screen the bottom sheet has to cover for the scrim to
// show and the floating action button to hide.
//
// Flutter: material/scaffold.dart _kBottomSheetDominatesPercentage
const bottomSheetDominatesPercentage = 0.3

// Flutter: material/scaffold.dart _kMinBottomSheetScrimOpacity
const minBottomSheetScrimOpacity = 0.1

// Flutter: material/scaffold.dart _kMaxBottomSheetScrimOpacity
const maxBottomSheetScrimOpacity = 0.6

// defaultBottomSheetScrimBuilder darkens the body more as the sheet covers more
// of the screen.
//
// Flutter: material/scaffold.dart Scaffold._defaultBottomSheetScrimBuilder
func defaultBottomSheetScrimBuilder(_ widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
	return widgets.NewAnimatedBuilder(a, func(widgets.BuildContext, widgets.Widget) widgets.Widget {
		extentRemaining := bottomSheetDominatesPercentage * (1.0 - a.Value())
		floatingButtonVisibilityValue := extentRemaining * bottomSheetDominatesPercentage * 10
		opacity := math.Max(minBottomSheetScrimOpacity, maxBottomSheetScrimOpacity-floatingButtonVisibilityValue)
		return widgets.NewModalBarrier(widgets.ModalBarrierOpts{
			Dismissible: foundation.Some(false), Color: foundation.Some(Colors.Black.WithOpacity(opacity))})
	}, nil)
}

// standardBottomSheet is the widget of a persistent bottom sheet and of one
// shown by ShowBottomSheet: the BottomSheet with the animation that makes it
// grow from the bottom.
//
// Flutter: material/scaffold.dart _StandardBottomSheet
type standardBottomSheet struct {
	widgets.StatefulWidget
	// animationController is controlled by the sheet but disposed by whoever
	// created it.
	animationController *animation.AnimationController
	enableDrag          bool
	showDragHandle      foundation.Opt[bool]
	onClosing           func()
	onDismissed         func()
	onDispose           func()
	builder             widgets.WidgetBuilder
	isPersistent        bool
	backgroundColor     foundation.Opt[ui.Color]
	elevation           foundation.Opt[float64]
	shape               painting.ShapeBorder
	clipBehavior        ui.Clip
	constraints         foundation.Opt[rendering.BoxConstraints]
}

func (w *standardBottomSheet) CreateState() widgets.State { return &standardBottomSheetState{} }

type standardBottomSheetState struct {
	widgets.StateOf[*standardBottomSheet]
	animationCurve animation.Curve
	statusHandle   animation.StatusListenerHandle
}

func (s *standardBottomSheetState) InitState() {
	s.animationCurve = animation.Curves.FastOutSlowIn
	s.statusHandle = s.Widget().animationController.AddStatusListener(s.handleStatusChange)
}

func (s *standardBottomSheetState) Dispose() {
	w := s.Widget()
	w.animationController.RemoveStatusListener(s.statusHandle)
	if w.onDispose != nil {
		w.onDispose()
	}
}

func (s *standardBottomSheetState) close() {
	w := s.Widget()
	w.animationController.Reverse()
	if w.onClosing != nil {
		w.onClosing()
	}
}

func (s *standardBottomSheetState) handleDragStart(gestures.DragStartDetails) {
	// Allow the bottom sheet to track the user's finger accurately.
	s.animationCurve = animation.Curves.Linear
}

func (s *standardBottomSheetState) handleDragEnd(gestures.DragEndDetails, bool) {
	// Allow the bottom sheet to animate smoothly from its current position.
	s.animationCurve = animation.NewSplit(s.Widget().animationController.Value(), animation.SplitOpts{EndCurve: animation.Curves.FastOutSlowIn})
}

func (s *standardBottomSheetState) handleStatusChange(status animation.AnimationStatus) {
	if status.IsDismissed() && s.Widget().onDismissed != nil {
		s.Widget().onDismissed()
	}
}

func (s *standardBottomSheetState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var onDismiss func()
	if !w.isPersistent {
		onDismiss = s.close
	}
	return widgets.NewAnimatedBuilder(w.animationController, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		return widgets.NewAlign(child, widgets.AlignOpts{
			Alignment:    painting.AlignmentDirectionalTopStart,
			HeightFactor: foundation.Some(s.animationCurve.Transform(w.animationController.Value()))})
	}, widgets.NewSemantics(
		// Deviation: Flutter's NotificationListener<DraggableScrollableNotification>
		// that hides the floating action button and shows the body scrim is not
		// ported with DraggableScrollableSheet (SPEC §12.10).
		NewBottomSheet(w.onClosing, w.builder, BottomSheetOpts{
			AnimationController: w.animationController, NoEnableDrag: !w.enableDrag, ShowDragHandle: w.showDragHandle,
			OnDragStart: s.handleDragStart, OnDragEnd: s.handleDragEnd, BackgroundColor: w.backgroundColor,
			Elevation: w.elevation, Shape: w.shape, ClipBehavior: w.clipBehavior, Constraints: w.constraints}),
		widgets.SemanticsOpts{Container: true, OnDismiss: onDismiss}))
}

// PersistentBottomSheetController controls a bottom sheet shown by
// ScaffoldState.ShowBottomSheet or built from Scaffold.BottomSheet.
//
// Flutter: material/scaffold.dart PersistentBottomSheetController
type PersistentBottomSheetController struct {
	*ScaffoldFeatureController[*standardBottomSheet, struct{}]
	isLocalHistoryEntry bool
}

// buildBottomSheetOpts are the parameters of ScaffoldState.buildBottomSheet.
type buildBottomSheetOpts struct {
	isPersistent                 bool
	animationController          *animation.AnimationController
	backgroundColor              foundation.Opt[ui.Color]
	elevation                    foundation.Opt[float64]
	shape                        painting.ShapeBorder
	clipBehavior                 ui.Clip
	constraints                  foundation.Opt[rendering.BoxConstraints]
	enableDrag                   foundation.Opt[bool]
	showDragHandle               foundation.Opt[bool]
	noDisposeAnimationController bool
}

func (s *ScaffoldState) maybeBuildPersistentBottomSheet() {
	if s.Widget().o.BottomSheet == nil || s.currentBottomSheet != nil {
		return
	}
	// The new currentBottomSheet is not a local history entry so a "back" button
	// will not be added to the Scaffold's app bar and the bottom sheet will not
	// support drag or swipe to dismiss.
	controller := BottomSheetCreateAnimationController(s, nil)
	controller.SetValue(1.0)

	// Stop the animation and unmount the dismissed sheets from the tree
	// immediately, otherwise may cause a duplicate GlobalKey assertion if the
	// sheet sub-tree contains GlobalKey widgets.
	for _, sheet := range append([]*standardBottomSheet(nil), s.dismissedBottomSheets...) {
		sheet.animationController.Reset()
	}

	// Deviation: the DraggableScrollableNotification that adds a local history
	// entry to the persistent sheet is not ported (SPEC §12.10).
	s.currentBottomSheet = s.buildBottomSheet(func(widgets.BuildContext) widgets.Widget {
		return widgets.NewStatefulBuilder(func(widgets.BuildContext, widgets.StateSetter) widgets.Widget {
			if b := s.Widget().o.BottomSheet; b != nil {
				return b
			}
			return widgets.NewSizedBoxShrink(nil)
		}, widgets.WidgetOpts{Key: s.currentBottomSheetKey})
	}, buildBottomSheetOpts{isPersistent: true, animationController: controller})
}

// showFloatingActionButton shows the floating action button again, which a bottom
// sheet that fills the screen hides.
//
// Flutter: material/scaffold.dart ScaffoldState._showFloatingActionButton
func (s *ScaffoldState) showFloatingActionButton() *scheduler.TickerFuture {
	return s.floatingActionButtonVisibilityController.Forward()
}

func (s *ScaffoldState) closeCurrentBottomSheet() {
	if s.currentBottomSheet != nil && !s.currentBottomSheet.isLocalHistoryEntry {
		s.currentBottomSheet.Close()
	}
}

func (s *ScaffoldState) updatePersistentBottomSheet() {
	if st, ok := s.currentBottomSheetKey.CurrentState(); ok {
		st.(interface{ SetState(func()) }).SetState(func() {})
	}
}

func (s *ScaffoldState) buildBottomSheet(builder widgets.WidgetBuilder, o buildBottomSheetOpts) *PersistentBottomSheetController {
	completer := newCompleter[struct{}]()
	bottomSheetKey := widgets.NewGlobalKey[*standardBottomSheetState]()
	var bottomSheet *standardBottomSheet
	removedEntry := false
	doingDispose := false
	animationController := o.animationController

	var entry *widgets.LocalHistoryEntry

	removeCurrentBottomSheet := func() {
		removedEntry = true
		if s.currentBottomSheet == nil {
			return
		}
		s.showFloatingActionButton()
		if st, ok := bottomSheetKey.CurrentState(); ok {
			st.close()
		}
		s.SetState(func() {
			s.showBodyScrim = false
			s.bottomSheetScrimController.SetValue(0.0)
			s.currentBottomSheet = nil
		})
		if !animationController.Status().IsDismissed() {
			s.dismissedBottomSheets = append(s.dismissedBottomSheets, bottomSheet)
		}
		completer.complete(struct{}{})
	}

	if !o.isPersistent {
		entry = widgets.NewLocalHistoryEntry(widgets.LocalHistoryEntryOpts{OnRemove: func() {
			if !removedEntry && s.currentBottomSheet != nil && s.currentBottomSheet.Widget() == bottomSheet && !doingDispose {
				removeCurrentBottomSheet()
			}
		}})
	}

	removeEntryIfNeeded := func() {
		if !o.isPersistent && !removedEntry {
			entry.Remove()
			removedEntry = true
		}
	}

	bottomSheet = &standardBottomSheet{
		StatefulWidget:      widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: bottomSheetKey}),
		animationController: animationController,
		enableDrag:          o.enableDrag.OrElse(!o.isPersistent),
		showDragHandle:      o.showDragHandle,
		onClosing: func() {
			if s.currentBottomSheet == nil {
				return
			}
			removeEntryIfNeeded()
		},
		onDismissed: func() {
			for i, sheet := range s.dismissedBottomSheets {
				if sheet == bottomSheet {
					s.SetState(func() {
						s.dismissedBottomSheets = append(s.dismissedBottomSheets[:i:i], s.dismissedBottomSheets[i+1:]...)
					})
					break
				}
			}
		},
		onDispose: func() {
			doingDispose = true
			removeEntryIfNeeded()
			if !o.noDisposeAnimationController {
				animationController.Dispose()
			}
		},
		builder:         builder,
		isPersistent:    o.isPersistent,
		backgroundColor: o.backgroundColor,
		elevation:       o.elevation,
		shape:           o.shape,
		clipBehavior:    o.clipBehavior,
		constraints:     o.constraints,
	}

	if !o.isPersistent {
		if route := widgets.ModalRouteOf(s.Context()); route != nil {
			route.AddLocalHistoryEntry(entry)
		}
	}

	closeFn := removeCurrentBottomSheet
	if entry != nil {
		closeFn = entry.Remove
	}
	return &PersistentBottomSheetController{
		ScaffoldFeatureController: &ScaffoldFeatureController[*standardBottomSheet, struct{}]{
			widget:    bottomSheet,
			completer: completer,
			close:     closeFn,
			setState: func(fn func()) {
				if st, ok := bottomSheetKey.CurrentState(); ok {
					st.SetState(fn)
				}
			},
		},
		isLocalHistoryEntry: !o.isPersistent,
	}
}

// ShowBottomSheetOpts are the optional parameters of ShowBottomSheet and
// ScaffoldState.ShowBottomSheet.
//
// Flutter: material/scaffold.dart ScaffoldState.showBottomSheet
type ShowBottomSheetOpts struct {
	// BackgroundColor is the color of the sheet.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the sheet.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the sheet.
	Shape painting.ShapeBorder
	// ClipBehavior is the clip of the sheet.
	ClipBehavior ui.Clip
	// Constraints are the size constraints of the sheet.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// EnableDrag says whether the sheet can be dragged down to close it; the
	// default is true.
	EnableDrag foundation.Opt[bool]
	// ShowDragHandle says whether the sheet shows a drag handle.
	ShowDragHandle foundation.Opt[bool]
	// TransitionAnimationController controls the animation of the sheet; the
	// caller disposes it.
	TransitionAnimationController *animation.AnimationController
	// SheetAnimationStyle is the duration of the animation.
	SheetAnimationStyle *animation.AnimationStyle
}

// ShowBottomSheet shows a persistent bottom sheet in the closest Scaffold. The
// sheet stays until Close is called on the controller, the user drags it away,
// or the route is popped.
//
// Flutter: material/scaffold.dart ScaffoldState.showBottomSheet
func (s *ScaffoldState) ShowBottomSheet(builder widgets.WidgetBuilder, opts ...ShowBottomSheetOpts) *PersistentBottomSheetController {
	o := firstOpts(opts)
	if foundation.DebugMode && s.Widget().o.BottomSheet != nil {
		panic(errors.New("ScaffoldBottomSheetShowConflict", "Scaffold.bottomSheet cannot be specified while a bottom sheet displayed with showBottomSheet() is still visible. Rebuild the Scaffold with a null bottomSheet before calling showBottomSheet()."))
	}
	widgets.DebugCheckHasMediaQuery(s.Context())

	s.closeCurrentBottomSheet()
	controller := o.TransitionAnimationController
	if controller == nil {
		controller = BottomSheetCreateAnimationController(s, o.SheetAnimationStyle)
	}
	controller.Forward()
	s.SetState(func() {
		s.currentBottomSheet = s.buildBottomSheet(builder, buildBottomSheetOpts{
			animationController: controller, backgroundColor: o.BackgroundColor, elevation: o.Elevation, shape: o.Shape,
			clipBehavior: o.ClipBehavior, constraints: o.Constraints, enableDrag: o.EnableDrag, showDragHandle: o.ShowDragHandle,
			noDisposeAnimationController: o.TransitionAnimationController != nil})
	})
	return s.currentBottomSheet
}

// ShowBottomSheet shows a Material Design bottom sheet in the Scaffold that most
// tightly encloses the context.
//
// Flutter: material/bottom_sheet.dart showBottomSheet
func ShowBottomSheet(bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...ShowBottomSheetOpts) *PersistentBottomSheetController {
	return ScaffoldOf(bc).ShowBottomSheet(builder, opts...)
}
