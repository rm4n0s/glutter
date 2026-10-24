package widgets

import (
	"math"
	"reflect"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

const schedulerPhasePersistentCallbacks = scheduler.SchedulerPhasePersistentCallbacks

// SelectionOverlayFadeDuration controls the fade-in and fade-out animations
// for the toolbar and handles.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.fadeDuration
const SelectionOverlayFadeDuration = 150 * time.Millisecond

// SelectionOverlayOpts are the optional parameters of NewSelectionOverlay.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.new
type SelectionOverlayOpts struct {
	// DebugRequiredFor is the widget that requires the overlay, named in the
	// error when there is none.
	DebugRequiredFor Widget
	// StartHandlesVisible is whether the start handle is visible; nil for
	// always visible.
	StartHandlesVisible foundation.ValueListenable[bool]
	// OnStartHandleDragStart is called when the user starts dragging the start handle.
	OnStartHandleDragStart func(gestures.DragStartDetails)
	// OnStartHandleDragUpdate is called when the user drags the start handle.
	OnStartHandleDragUpdate func(gestures.DragUpdateDetails)
	// OnStartHandleDragEnd is called when the user lifts their finger after
	// dragging the start handle.
	OnStartHandleDragEnd func(gestures.DragEndDetails)
	// EndHandlesVisible is whether the end handle is visible; nil for always visible.
	EndHandlesVisible foundation.ValueListenable[bool]
	// OnEndHandleDragStart is called when the user starts dragging the end handle.
	OnEndHandleDragStart func(gestures.DragStartDetails)
	// OnEndHandleDragUpdate is called when the user drags the end handle.
	OnEndHandleDragUpdate func(gestures.DragUpdateDetails)
	// OnEndHandleDragEnd is called when the user lifts their finger after
	// dragging the end handle.
	OnEndHandleDragEnd func(gestures.DragEndDetails)
	// ToolbarVisible is whether the toolbar is visible; nil for always visible.
	ToolbarVisible foundation.ValueListenable[bool]
	// DragStartBehavior is where a handle drag starts; the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// OnSelectionHandleTapped is called when a handle is tapped.
	OnSelectionHandleTapped foundation.VoidCallback
	// MagnifierConfiguration is the configuration for the magnifier. By default
	// (nil), SelectionOverlay's TextMagnifierConfiguration is disabled.
	MagnifierConfiguration *TextMagnifierConfiguration
}

type selectionHandles struct{ start, end *OverlayEntry }

// SelectionOverlay builds text editing handles and the toolbar of a text
// selection in the Overlay.
//
// Deviation: the spell check toolbar and the deprecated buildToolbar path
// (toolbarLocation, selectionDelegate and clipboardStatus) are not ported
// (SPEC §12.10).
//
// Flutter: widgets/text_selection.dart SelectionOverlay
type SelectionOverlay struct {
	context BuildContext
	o       SelectionOverlayOpts

	startHandleType   rendering.TextSelectionHandleType
	lineHeightAtStart float64
	endHandleType     rendering.TextSelectionHandleType
	lineHeightAtEnd   float64
	selectionEndpoint []rendering.TextSelectionPoint

	selectionControls TextSelectionControls
	startLink         *rendering.LayerLink
	endLink           *rendering.LayerLink
	toolbarLink       *rendering.LayerLink

	// startHandleDragInProgress is whether a drag is in progress on the start
	// handle. This differs from isDraggingStartHandle in that it is not blocked
	// by canDragStartHandle.
	startHandleDragInProgress bool
	isDraggingStart           bool
	endHandleDragInProgress   bool
	isDraggingEnd             bool

	handles        *selectionHandles
	contextMenu    *ContextMenuController
	buildScheduled bool
	disposed       bool

	// magnifierInfo is the info the magnifier follows; showMagnifier and
	// updateMagnifier set it.
	magnifierInfo *foundation.ValueNotifier[MagnifierInfo]
	// magnifierController shows and hides the magnifier. It should not be used
	// directly, except from ShowMagnifier and HideMagnifier: the magnifier needs
	// to orchestrate with other properties in SelectionOverlay.
	magnifierController *MagnifierController
	magnifierConfig     *TextMagnifierConfiguration
}

// NewSelectionOverlay creates an object that manages overlay entries for
// selection handles. The context must have an Overlay as an ancestor.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.new
func NewSelectionOverlay(bc BuildContext, startHandleType rendering.TextSelectionHandleType, lineHeightAtStart float64,
	endHandleType rendering.TextSelectionHandleType, lineHeightAtEnd float64, selectionEndpoints []rendering.TextSelectionPoint,
	selectionControls TextSelectionControls, startHandleLayerLink, endHandleLayerLink, toolbarLayerLink *rendering.LayerLink,
	opts ...SelectionOverlayOpts) *SelectionOverlay {
	o := &SelectionOverlay{
		context: bc, o: firstOpts(opts), startHandleType: startHandleType, lineHeightAtStart: lineHeightAtStart,
		endHandleType: endHandleType, lineHeightAtEnd: lineHeightAtEnd, selectionEndpoint: selectionEndpoints,
		selectionControls: selectionControls, startLink: startHandleLayerLink, endLink: endHandleLayerLink,
		toolbarLink: toolbarLayerLink, contextMenu: NewContextMenuController(),
		magnifierInfo:       foundation.NewValueNotifier(MagnifierInfoEmpty),
		magnifierController: NewMagnifierController(),
		magnifierConfig:     TextMagnifierConfigurationDisabled,
	}
	if o.o.MagnifierConfiguration != nil {
		o.magnifierConfig = o.o.MagnifierConfiguration
	}
	return o
}

// MagnifierConfiguration is the configuration for the magnifier.
//
// By default, SelectionOverlay's TextMagnifierConfiguration is disabled.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.magnifierConfiguration
func (o *SelectionOverlay) MagnifierConfiguration() *TextMagnifierConfiguration {
	return o.magnifierConfig
}

// MagnifierIsVisible is whether the magnifier is currently visible.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.magnifierIsVisible
func (o *SelectionOverlay) MagnifierIsVisible() bool { return o.magnifierController.Shown() }

// MagnifierExists is whether the magnifier currently exists.
//
// This differs from MagnifierIsVisible in that the magnifier may exist in the
// overlay, but not be shown.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.magnifierExists
func (o *SelectionOverlay) MagnifierExists() bool { return o.magnifierController.OverlayEntry() != nil }

// ShowMagnifier shows the magnifier, and hides the toolbar if it was showing
// when ShowMagnifier was called. This is safe to call on platforms not mobile,
// since a magnifier builder will not be provided, or the magnifier builder will
// return nil on platforms not mobile.
//
// This is NOT the source of truth for if the magnifier is up or not, since
// magnifiers may hide themselves. If this info is needed, check
// MagnifierIsVisible.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.showMagnifier
func (o *SelectionOverlay) ShowMagnifier(initialMagnifierInfo MagnifierInfo) {
	// Do not show the magnifier if one already exists.
	if o.magnifierController.OverlayEntry() != nil {
		return
	}
	if o.ToolbarIsVisible() {
		o.HideToolbar()
	}

	// Start from empty, so we don't utilize any remnant values.
	o.magnifierInfo.SetValue(initialMagnifierInfo)

	// Pre-build the magnifiers so we can tell if we've built something or not.
	// If we don't build a magnifiers, then we should not insert anything in the
	// overlay.
	builtMagnifier := o.magnifierConfig.MagnifierBuilder()(o.context, o.magnifierController, o.magnifierInfo)
	if builtMagnifier == nil {
		return
	}

	var below *OverlayEntry
	if !o.magnifierConfig.ShouldDisplayHandlesInMagnifier() && o.handles != nil {
		below = o.handles.start
	}
	o.magnifierController.Show(o.context, func(BuildContext) Widget { return builtMagnifier }, MagnifierControllerShowOpts{Below: below})
}

// HideMagnifier hides the current magnifier. This does nothing if there is no
// magnifier.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.hideMagnifier
func (o *SelectionOverlay) HideMagnifier() {
	// This cannot be a check on MagnifierController.Shown, since it's possible
	// that the magnifier is still in the overlay, but not shown in cases where
	// the magnifier hides itself.
	if o.magnifierController.OverlayEntry() == nil {
		return
	}
	o.magnifierController.Hide()
}

// UpdateMagnifier updates the current magnifier with new selection data, so the
// magnifier can respond accordingly.
//
// If the magnifier is not shown, this still updates the magnifier position
// because the magnifier may have hidden itself and is looking for a cue to
// reshow itself.
//
// If there is no magnifier in the overlay, this does nothing.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.updateMagnifier
func (o *SelectionOverlay) UpdateMagnifier(magnifierInfo MagnifierInfo) {
	if o.magnifierController.OverlayEntry() == nil {
		return
	}
	o.magnifierInfo.SetValue(magnifierInfo)
}

// Context is the context in which the selection handles should appear.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.context
func (o *SelectionOverlay) Context() BuildContext { return o.context }

// ToolbarIsVisible is whether the toolbar is currently visible.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.toolbarIsVisible
func (o *SelectionOverlay) ToolbarIsVisible() bool { return o.contextMenu.IsShown() }

// StartHandleType is the type of the start selection handle.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.startHandleType
func (o *SelectionOverlay) StartHandleType() rendering.TextSelectionHandleType {
	return o.startHandleType
}

// SetStartHandleType sets the type of the start selection handle. Changing the
// value while the handles are visible causes them to rebuild.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.startHandleType
func (o *SelectionOverlay) SetStartHandleType(value rendering.TextSelectionHandleType) {
	if o.startHandleType == value {
		return
	}
	o.startHandleType = value
	o.MarkNeedsBuild()
}

// LineHeightAtStart is the line height at the selection start.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.lineHeightAtStart
func (o *SelectionOverlay) LineHeightAtStart() float64 { return o.lineHeightAtStart }

// SetLineHeightAtStart sets the line height at the selection start, which is
// used for the size of the start handle. Changing the value while the handles
// are visible causes them to rebuild.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.lineHeightAtStart
func (o *SelectionOverlay) SetLineHeightAtStart(value float64) {
	if o.lineHeightAtStart == value {
		return
	}
	o.lineHeightAtStart = value
	o.MarkNeedsBuild()
}

// IsDraggingStartHandle is whether the selection start handle is currently
// being dragged.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.isDraggingStartHandle
func (o *SelectionOverlay) IsDraggingStartHandle() bool {
	return o.isDraggingStart || o.startHandleDragInProgress
}

// EndHandleType is the type of the end selection handle.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.endHandleType
func (o *SelectionOverlay) EndHandleType() rendering.TextSelectionHandleType { return o.endHandleType }

// SetEndHandleType sets the type of the end selection handle. Changing the
// value while the handles are visible causes them to rebuild.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.endHandleType
func (o *SelectionOverlay) SetEndHandleType(value rendering.TextSelectionHandleType) {
	if o.endHandleType == value {
		return
	}
	o.endHandleType = value
	o.MarkNeedsBuild()
}

// LineHeightAtEnd is the line height at the selection end.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.lineHeightAtEnd
func (o *SelectionOverlay) LineHeightAtEnd() float64 { return o.lineHeightAtEnd }

// SetLineHeightAtEnd sets the line height at the selection end, which is used
// for the size of the end handle. Changing the value while the handles are
// visible causes them to rebuild.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.lineHeightAtEnd
func (o *SelectionOverlay) SetLineHeightAtEnd(value float64) {
	if o.lineHeightAtEnd == value {
		return
	}
	o.lineHeightAtEnd = value
	o.MarkNeedsBuild()
}

// IsDraggingEndHandle is whether the selection end handle is currently being
// dragged.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.isDraggingEndHandle
func (o *SelectionOverlay) IsDraggingEndHandle() bool {
	return o.isDraggingEnd || o.endHandleDragInProgress
}

// onlyOneHandleDragsAtATime is whether only one selection handle can be dragged
// at a time: on Apple platforms.
func onlyOneHandleDragsAtATime() bool {
	p := foundation.DefaultTargetPlatform()
	return p == foundation.TargetPlatformIOS || p == foundation.TargetPlatformMacOS
}

func (o *SelectionOverlay) canDragStartHandle() bool {
	return !o.isDraggingEnd || !onlyOneHandleDragsAtATime()
}

func (o *SelectionOverlay) canDragEndHandle() bool {
	return !o.isDraggingStart || !onlyOneHandleDragsAtATime()
}

func isTouch(kind foundation.Opt[ui.PointerDeviceKind]) bool {
	k, ok := kind.Get()
	return ok && k == ui.PointerDeviceKindTouch
}

func dragStartFromUpdate(d gestures.DragUpdateDetails) gestures.DragStartDetails {
	return gestures.DragStartDetails{GlobalPosition: d.GlobalPosition, LocalPosition: d.LocalPosition, SourceTimeStamp: d.SourceTimeStamp, Kind: d.Kind}
}

func (o *SelectionOverlay) handleStartHandleDragStart(details gestures.DragStartDetails) {
	// Calling OverlayEntry.Remove may not happen until the following frame, so
	// it's possible for the handles to receive a gesture after calling remove.
	if o.handles == nil {
		o.isDraggingStart = false
		return
	}
	o.startHandleDragInProgress = true
	if !o.canDragStartHandle() {
		return
	}
	o.isDraggingStart = isTouch(details.Kind)
	if o.o.OnStartHandleDragStart != nil {
		o.o.OnStartHandleDragStart(details)
	}
}

func (o *SelectionOverlay) handleStartHandleDragUpdate(details gestures.DragUpdateDetails) {
	if o.handles == nil {
		o.isDraggingStart = false
		return
	}
	if !o.canDragStartHandle() {
		return
	}
	// The handle drag may have been blocked before on Apple platforms while the
	// opposite handle was being dragged. Ensure that any logic that was meant to
	// be run in OnStartHandleDragStart is still run.
	if !o.isDraggingStart {
		o.isDraggingStart = isTouch(details.Kind)
		if o.o.OnStartHandleDragStart != nil {
			o.o.OnStartHandleDragStart(dragStartFromUpdate(details))
		}
	}
	if o.o.OnStartHandleDragUpdate != nil {
		o.o.OnStartHandleDragUpdate(details)
	}
}

func (o *SelectionOverlay) handleStartHandleDragEnd(details gestures.DragEndDetails) {
	o.isDraggingStart = false
	if o.handles == nil {
		return
	}
	o.startHandleDragInProgress = false
	if !o.canDragStartHandle() {
		return
	}
	if o.o.OnStartHandleDragEnd != nil {
		o.o.OnStartHandleDragEnd(details)
	}
}

func (o *SelectionOverlay) handleEndHandleDragStart(details gestures.DragStartDetails) {
	if o.handles == nil {
		o.isDraggingEnd = false
		return
	}
	o.endHandleDragInProgress = true
	if !o.canDragEndHandle() {
		return
	}
	o.isDraggingEnd = isTouch(details.Kind)
	if o.o.OnEndHandleDragStart != nil {
		o.o.OnEndHandleDragStart(details)
	}
}

func (o *SelectionOverlay) handleEndHandleDragUpdate(details gestures.DragUpdateDetails) {
	if o.handles == nil {
		o.isDraggingEnd = false
		return
	}
	if !o.canDragEndHandle() {
		return
	}
	if !o.isDraggingEnd {
		o.isDraggingEnd = isTouch(details.Kind)
		if o.o.OnEndHandleDragStart != nil {
			o.o.OnEndHandleDragStart(dragStartFromUpdate(details))
		}
	}
	if o.o.OnEndHandleDragUpdate != nil {
		o.o.OnEndHandleDragUpdate(details)
	}
}

func (o *SelectionOverlay) handleEndHandleDragEnd(details gestures.DragEndDetails) {
	o.isDraggingEnd = false
	if o.handles == nil {
		return
	}
	o.endHandleDragInProgress = false
	if !o.canDragEndHandle() {
		return
	}
	if o.o.OnEndHandleDragEnd != nil {
		o.o.OnEndHandleDragEnd(details)
	}
}

// SelectionEndpoints are the text selection positions of selection start and end.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.selectionEndpoints
func (o *SelectionOverlay) SelectionEndpoints() []rendering.TextSelectionPoint {
	return o.selectionEndpoint
}

// SetSelectionEndpoints sets the text selection positions of selection start
// and end, and rebuilds the overlay if they changed.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.selectionEndpoints
func (o *SelectionOverlay) SetSelectionEndpoints(value []rendering.TextSelectionPoint) {
	if !reflect.DeepEqual(o.selectionEndpoint, value) {
		o.MarkNeedsBuild()
		if (o.isDraggingEnd || o.isDraggingStart) && foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid {
			WidgetsBindingOf(o.context).HapticFeedback().SelectionClick()
		}
	}
	o.selectionEndpoint = value
}

// ShowHandles builds the handles by inserting them into the overlay of the
// context.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.showHandles
func (o *SelectionOverlay) ShowHandles() {
	if o.handles != nil {
		return
	}
	overlay := OverlayOf(o.context, OverlayOfOpts{RootOverlay: true, DebugRequiredFor: o.o.DebugRequiredFor})
	capturedThemes := CaptureThemes(o.context, overlay.Context())
	o.handles = &selectionHandles{
		start: NewOverlayEntry(func(bc BuildContext) Widget { return capturedThemes.Wrap(o.buildStartHandle(bc)) }),
		end:   NewOverlayEntry(func(bc BuildContext) Widget { return capturedThemes.Wrap(o.buildEndHandle(bc)) }),
	}
	overlay.InsertAll([]*OverlayEntry{o.handles.start, o.handles.end})
}

// HideHandles destroys the handles by removing them from the overlay.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.hideHandles
func (o *SelectionOverlay) HideHandles() {
	if o.handles == nil {
		return
	}
	o.handles.start.Remove()
	o.handles.start.Dispose()
	o.handles.end.Remove()
	o.handles.end.Dispose()
	o.handles = nil
}

// ShowToolbar shows the toolbar by inserting it into the overlay of the context.
// A nil contextMenuBuilder does nothing.
//
// Deviation: Flutter also builds the deprecated toolbar of the selection
// controls when the builder is null; that path is not ported (SPEC §12.10).
//
// Flutter: widgets/text_selection.dart SelectionOverlay.showToolbar
func (o *SelectionOverlay) ShowToolbar(bc BuildContext, contextMenuBuilder WidgetBuilder) {
	if contextMenuBuilder == nil || bc == nil {
		return
	}
	renderBox, _ := bc.FindRenderObject().(rendering.Box)
	o.contextMenu.Show(bc, func(bc BuildContext) Widget {
		// The offset is read at each build, as Flutter's closure does: the field may
		// have moved since the toolbar was shown (the keyboard came up, a scroll).
		offset := ui.Offset{}
		if renderBox != nil {
			offset = renderBox.LocalToGlobal(ui.Offset{}, nil).Neg()
		}
		return newSelectionToolbarWrapper(o.o.ToolbarVisible, o.toolbarLink, offset, contextMenuBuilder(bc))
	}, ContextMenuShowOpts{DebugRequiredFor: o.o.DebugRequiredFor})
}

// MarkNeedsBuild rebuilds the selection toolbar or handles if they are present.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.markNeedsBuild
func (o *SelectionOverlay) MarkNeedsBuild() {
	if o.handles == nil && !o.contextMenu.IsShown() {
		return
	}
	// If we are in build state, it will be too late to update visibility. We
	// will need to schedule the build in next frame.
	sched := WidgetsBindingOf(o.context).Scheduler()
	if sched.SchedulerPhase() == schedulerPhasePersistentCallbacks {
		if o.buildScheduled {
			return
		}
		o.buildScheduled = true
		sched.AddPostFrameCallback(func(time.Duration) {
			o.buildScheduled = false
			o.rebuild()
		})
		return
	}
	o.rebuild()
}

func (o *SelectionOverlay) rebuild() {
	if o.handles != nil {
		o.handles.start.MarkNeedsBuild()
		o.handles.end.MarkNeedsBuild()
	}
	if o.contextMenu.IsShown() {
		o.contextMenu.MarkNeedsBuild()
	}
}

// Hide hides the entire overlay including the toolbar and the handles.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.hide
func (o *SelectionOverlay) Hide() {
	o.magnifierController.Hide()
	o.HideHandles()
	if o.contextMenu.IsShown() {
		o.HideToolbar()
	}
}

// HideToolbar hides the toolbar part of the overlay. To hide the whole overlay,
// see Hide.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.hideToolbar
func (o *SelectionOverlay) HideToolbar() { o.contextMenu.Remove() }

// Dispose disposes this object and releases resources.
//
// Flutter: widgets/text_selection.dart SelectionOverlay.dispose
func (o *SelectionOverlay) Dispose() {
	o.disposed = true
	o.Hide()
	o.magnifierInfo.Dispose()
}

func (o *SelectionOverlay) buildHandle(handleType rendering.TextSelectionHandleType, hidden bool, link *rendering.LayerLink,
	lineHeight float64, visibility foundation.ValueListenable[bool], onStart func(gestures.DragStartDetails),
	onUpdate func(gestures.DragUpdateDetails), onEnd func(gestures.DragEndDetails)) Widget {
	var handle Widget
	if o.selectionControls == nil || hidden {
		handle = NewSizedBoxShrink(nil)
	} else {
		handle = newSelectionHandleOverlay(selectionHandleOverlayArgs{
			handleType: handleType, handleLayerLink: link, onSelectionHandleTapped: o.o.OnSelectionHandleTapped,
			onDragStart: onStart, onDragUpdate: onUpdate, onDragEnd: onEnd, selectionControls: o.selectionControls,
			visibility: visibility, preferredLineHeight: lineHeight, dragStartBehavior: o.o.DragStartBehavior,
		})
	}
	return NewTapRegion(NewTextFieldTapRegion(NewExcludeSemantics(handle)), TapRegionOpts{GroupID: SelectableRegionTapRegionGroupID{}})
}

func (o *SelectionOverlay) buildStartHandle(BuildContext) Widget {
	// Hide the start handle when dragging the end handle and collapsing the
	// selection.
	hidden := o.startHandleType == rendering.TextSelectionHandleTypeCollapsed && o.isDraggingEnd
	return o.buildHandle(o.startHandleType, hidden, o.startLink, o.lineHeightAtStart, o.o.StartHandlesVisible,
		o.handleStartHandleDragStart, o.handleStartHandleDragUpdate, o.handleStartHandleDragEnd)
}

func (o *SelectionOverlay) buildEndHandle(BuildContext) Widget {
	// Hide the end handle when dragging the start handle and collapsing the
	// selection or when the selection is collapsed and no handle is being dragged.
	hidden := o.endHandleType == rendering.TextSelectionHandleTypeCollapsed && (o.isDraggingStart || !o.isDraggingEnd)
	return o.buildHandle(o.endHandleType, hidden, o.endLink, o.lineHeightAtEnd, o.o.EndHandlesVisible,
		o.handleEndHandleDragStart, o.handleEndHandleDragUpdate, o.handleEndHandleDragEnd)
}

// selectionToolbarWrapper is Flutter's _SelectionToolbarWrapper: it wraps the
// child in the widgets common to every toolbar.
type selectionToolbarWrapper struct {
	StatefulWidget
	child      Widget
	offset     ui.Offset
	layerLink  *rendering.LayerLink
	visibility foundation.ValueListenable[bool]
}

func newSelectionToolbarWrapper(visibility foundation.ValueListenable[bool], link *rendering.LayerLink, offset ui.Offset, child Widget) *selectionToolbarWrapper {
	return &selectionToolbarWrapper{NewStatefulWidgetBase(), child, offset, link, visibility}
}

func (w *selectionToolbarWrapper) CreateState() State { return &selectionToolbarWrapperState{} }

type selectionToolbarWrapperState struct {
	StateOf[*selectionToolbarWrapper]
	SingleTickerProvider
	controller *animation.AnimationController
	handle     foundation.ListenerHandle
}

func (s *selectionToolbarWrapperState) InitState() {
	s.controller = animation.NewAnimationController(&s.SingleTickerProvider, animation.AnimationControllerOpts{Duration: foundation.Some(SelectionOverlayFadeDuration)})
	s.toolbarVisibilityChanged()
	s.listen()
}

func (s *selectionToolbarWrapperState) listen() {
	if v := s.Widget().visibility; v != nil {
		s.handle = v.AddListener(s.toolbarVisibilityChanged)
	}
}

func (s *selectionToolbarWrapperState) unlisten(w *selectionToolbarWrapper) {
	if w.visibility != nil {
		w.visibility.RemoveListener(s.handle)
	}
}

func (s *selectionToolbarWrapperState) DidUpdateWidget(old *selectionToolbarWrapper) {
	if old.visibility == s.Widget().visibility {
		return
	}
	s.unlisten(old)
	s.toolbarVisibilityChanged()
	s.listen()
}

func (s *selectionToolbarWrapperState) Dispose() {
	s.unlisten(s.Widget())
	s.controller.Dispose()
}

func (s *selectionToolbarWrapperState) toolbarVisibilityChanged() {
	if v := s.Widget().visibility; v == nil || v.Value() {
		s.controller.Forward()
	} else {
		s.controller.Reverse()
	}
}

func (s *selectionToolbarWrapperState) Build(bc BuildContext) Widget {
	w := s.Widget()
	return NewTapRegion(NewTextFieldTapRegion(NewDirectionality(DirectionalityOf(s.Context()),
		NewFadeTransition(s.controller.View(),
			NewCompositedTransformFollower(w.layerLink, w.child, CompositedTransformFollowerOpts{HideWhenUnlinked: true, Offset: w.offset})))),
		TapRegionOpts{GroupID: SelectableRegionTapRegionGroupID{}})
}

type selectionHandleOverlayArgs struct {
	handleType              rendering.TextSelectionHandleType
	handleLayerLink         *rendering.LayerLink
	onSelectionHandleTapped foundation.VoidCallback
	onDragStart             func(gestures.DragStartDetails)
	onDragUpdate            func(gestures.DragUpdateDetails)
	onDragEnd               func(gestures.DragEndDetails)
	selectionControls       TextSelectionControls
	visibility              foundation.ValueListenable[bool]
	preferredLineHeight     float64
	dragStartBehavior       gestures.DragStartBehavior
}

// selectionHandleOverlay is Flutter's _SelectionHandleOverlay: a single
// draggable selection handle.
type selectionHandleOverlay struct {
	StatefulWidget
	a selectionHandleOverlayArgs
}

func newSelectionHandleOverlay(a selectionHandleOverlayArgs) *selectionHandleOverlay {
	return &selectionHandleOverlay{NewStatefulWidgetBase(), a}
}

func (w *selectionHandleOverlay) CreateState() State { return &selectionHandleOverlayState{} }

type selectionHandleOverlayState struct {
	StateOf[*selectionHandleOverlay]
	SingleTickerProvider
	controller *animation.AnimationController
	handle     foundation.ListenerHandle
}

func (s *selectionHandleOverlayState) InitState() {
	s.controller = animation.NewAnimationController(&s.SingleTickerProvider, animation.AnimationControllerOpts{Duration: foundation.Some(SelectionOverlayFadeDuration)})
	s.handleVisibilityChanged()
	if v := s.Widget().a.visibility; v != nil {
		s.handle = v.AddListener(s.handleVisibilityChanged)
	}
}

func (s *selectionHandleOverlayState) handleVisibilityChanged() {
	if v := s.Widget().a.visibility; v == nil || v.Value() {
		s.controller.Forward()
	} else {
		s.controller.Reverse()
	}
}

func (s *selectionHandleOverlayState) DidUpdateWidget(old *selectionHandleOverlay) {
	if v := old.a.visibility; v != nil {
		v.RemoveListener(s.handle)
	}
	s.handleVisibilityChanged()
	if v := s.Widget().a.visibility; v != nil {
		s.handle = v.AddListener(s.handleVisibilityChanged)
	}
}

func (s *selectionHandleOverlayState) Dispose() {
	if v := s.Widget().a.visibility; v != nil {
		v.RemoveListener(s.handle)
	}
	s.controller.Dispose()
}

func (s *selectionHandleOverlayState) Build(bc BuildContext) Widget {
	a := s.Widget().a
	handleSize := a.selectionControls.GetHandleSize(a.preferredLineHeight)
	// The bounding rect of the text selection handle in local coordinates.
	handleRect := ui.NewRectFromLTWH(0, 0, handleSize.Width, handleSize.Height)

	// Make sure the GestureDetector is big enough to be easily interactive if
	// the handleRect is not empty.
	interactiveRect := handleRect
	if !handleRect.IsEmpty() {
		interactiveRect = handleRect.ExpandToInclude(ui.NewRectFromCircle(handleRect.Center(), MinInteractiveDimension/2))
	}
	padding := rendering.RelativeRectFill
	if !interactiveRect.IsEmpty() {
		px := math.Max((interactiveRect.Width()-handleRect.Width())/2, 0)
		py := math.Max((interactiveRect.Height()-handleRect.Height())/2, 0)
		padding = rendering.NewRelativeRectFromLTRB(px, py, px, py)
	}
	handleAnchor := a.selectionControls.GetHandleAnchor(a.handleType, a.preferredLineHeight)

	// Make sure a drag is eagerly accepted. This is used on iOS to match the
	// behavior where a drag directly on a collapse handle will always win
	// against other drag gestures.
	eagerlyAcceptDragWhenCollapsed := a.handleType == rendering.TextSelectionHandleTypeCollapsed && foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS

	binding := WidgetsBindingOf(bc).Gestures()
	factory := NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.PanGestureRecognizer {
			return gestures.NewPanGestureRecognizer(binding, gestures.DragGestureRecognizerOpts{
				DebugOwner: s,
				// Mouse events select the text and do not drag the cursor.
				SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus, ui.PointerDeviceKindUnknown},
			})
		},
		func(r *gestures.PanGestureRecognizer) {
			a := s.Widget().a
			r.DragStartBehavior = a.dragStartBehavior
			r.GestureSettings = foundation.None[gestures.DeviceGestureSettings]()
			if eagerlyAcceptDragWhenCollapsed {
				r.GestureSettings = foundation.Some(gestures.DeviceGestureSettings{TouchSlop: foundation.Some(1.0)})
			}
			r.OnStart, r.OnUpdate, r.OnEnd = a.onDragStart, a.onDragUpdate, a.onDragEnd
		})

	return NewCompositedTransformFollower(a.handleLayerLink,
		NewFadeTransition(s.controller.View(),
			NewSizedBox(NewAlign(NewRawGestureDetector(
				NewPadding(painting.NewEdgeInsetsFromLTRB(padding.Left, padding.Top, padding.Right, padding.Bottom),
					a.selectionControls.BuildHandle(bc, a.handleType, a.preferredLineHeight, a.onSelectionHandleTapped)),
				RawGestureDetectorOpts{
					Behavior: rendering.HitTestBehaviorTranslucent,
					Gestures: map[reflect.Type]GestureRecognizerFactory{factory.RecognizerType(): factory},
				}), AlignOpts{Alignment: painting.AlignmentTopLeft}),
				SizedBoxOpts{Width: foundation.Some(interactiveRect.Width()), Height: foundation.Some(interactiveRect.Height())})),
		// Put the handle's anchor point on the leader's anchor point.
		CompositedTransformFollowerOpts{HideWhenUnlinked: true, Offset: handleAnchor.Neg().Sub(ui.Offset{Dx: padding.Left, Dy: padding.Top})})
}
