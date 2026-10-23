package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// MinInteractiveDimension is the minimum dimension of any interactive region
// according to Material guidelines: the smallest size a tap target can have.
//
// Flutter: widgets/constants.dart kMinInteractiveDimension
const MinInteractiveDimension = 48.0

// SelectableRegionTapRegionGroupID is the group of the tap regions of a
// SelectableRegion, its toolbar and its handles: a tap in any of them is a tap
// inside all of them.
//
// Flutter: widgets/text_selection.dart SelectionOverlay._buildStartHandle (groupId: SelectableRegion)
type SelectableRegionTapRegionGroupID struct{}

// ToolbarBuilder builds a toolbar around the child.
//
// Flutter: widgets/text_selection.dart ToolbarBuilder
type ToolbarBuilder func(bc BuildContext, child Widget) Widget

// ToolbarItemsParentData is the parent data of the items of a text selection
// toolbar: the offset and whether the item is painted.
//
// Flutter: widgets/text_selection.dart ToolbarItemsParentData
type ToolbarItemsParentData struct {
	rendering.ContainerBoxParentData[rendering.Box]
	// ShouldPaint is whether the item is painted.
	//
	// Flutter: widgets/text_selection.dart ToolbarItemsParentData.shouldPaint
	ShouldPaint bool
}

// String describes the parent data.
//
// Flutter: widgets/text_selection.dart ToolbarItemsParentData.toString
func (d *ToolbarItemsParentData) String() string {
	return d.ContainerBoxParentData.String() + "; shouldPaint=" + boolString(d.ShouldPaint)
}

// TextSelectionControls is the interface that builds the selection handles of
// a text field.
//
// Deviation: Flutter's class also builds a toolbar (buildToolbar) and tests
// whether a selection can be cut, copied, pasted or all selected (canCut ...),
// which are deprecated in favor of the context menu builder; they are not
// ported, so every text selection controls is Flutter's TextSelectionHandleControls
// (SPEC §12.10).
//
// Flutter: widgets/text_selection.dart TextSelectionControls
type TextSelectionControls interface {
	// BuildHandle builds a selection handle of the given type. The handle
	// should call onTap, if given, when it is tapped.
	//
	// Flutter: widgets/text_selection.dart TextSelectionControls.buildHandle
	BuildHandle(bc BuildContext, handleType rendering.TextSelectionHandleType, textLineHeight float64, onTap foundation.VoidCallback) Widget
	// GetHandleAnchor gets the position of the text selection handles' anchor
	// relative to the handle's top left corner.
	//
	// Flutter: widgets/text_selection.dart TextSelectionControls.getHandleAnchor
	GetHandleAnchor(handleType rendering.TextSelectionHandleType, textLineHeight float64) ui.Offset
	// GetHandleSize gets the size of a selection handle given the line height.
	//
	// Flutter: widgets/text_selection.dart TextSelectionControls.getHandleSize
	GetHandleSize(textLineHeight float64) ui.Size
}

// EmptyTextSelectionControls is a TextSelectionControls that does not build
// anything.
//
// Flutter: widgets/text_selection.dart EmptyTextSelectionControls
type EmptyTextSelectionControls struct{}

// NewEmptyTextSelectionControls creates the controls that build nothing.
//
// Flutter: widgets/text_selection.dart emptyTextSelectionControls
func NewEmptyTextSelectionControls() EmptyTextSelectionControls { return EmptyTextSelectionControls{} }

// GetHandleSize is the zero size.
//
// Flutter: widgets/text_selection.dart EmptyTextSelectionControls.getHandleSize
func (EmptyTextSelectionControls) GetHandleSize(float64) ui.Size { return ui.SizeZero }

// BuildHandle builds an empty box.
//
// Flutter: widgets/text_selection.dart EmptyTextSelectionControls.buildHandle
func (EmptyTextSelectionControls) BuildHandle(BuildContext, rendering.TextSelectionHandleType, float64, foundation.VoidCallback) Widget {
	return NewSizedBoxShrink(nil)
}

// GetHandleAnchor is the zero offset.
//
// Flutter: widgets/text_selection.dart EmptyTextSelectionControls.getHandleAnchor
func (EmptyTextSelectionControls) GetHandleAnchor(rendering.TextSelectionHandleType, float64) ui.Offset {
	return ui.OffsetZero
}

// TextSelectionOverlayOpts are the optional parameters of NewTextSelectionOverlay.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.new
type TextSelectionOverlayOpts struct {
	// DebugRequiredFor is the widget that requires the overlay, named in the
	// error when there is none.
	DebugRequiredFor Widget
	// SelectionControls build the selection handles; none for no handles.
	SelectionControls TextSelectionControls
	// HandlesVisible is whether the handles are visible; the default is false.
	HandlesVisible bool
	// DragStartBehavior is when a handle drag starts; the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// OnSelectionHandleTapped is called when a handle is tapped.
	OnSelectionHandleTapped foundation.VoidCallback
	// ContextMenuBuilder builds the context menu of the toolbar.
	ContextMenuBuilder WidgetBuilder
}

// TextSelectionOverlay manages a pair of text selection handles, and the
// toolbar, of a RenderEditable. The selection handles are displayed in the
// Overlay that most closely encloses the given BuildContext.
//
// Deviation: the magnifier and the spell check suggestions toolbar are not
// ported (SPEC §12.10).
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay
type TextSelectionOverlay struct {
	context            BuildContext
	renderObject       *rendering.RenderEditable
	selectionControls  TextSelectionControls
	selectionDelegate  services.TextSelectionDelegate
	selectionOverlay   *SelectionOverlay
	contextMenuBuilder WidgetBuilder

	value          services.TextEditingValue
	handlesVisible bool

	effectiveStartHandleVisibility *foundation.ValueNotifier[bool]
	effectiveEndHandleVisibility   *foundation.ValueNotifier[bool]
	effectiveToolbarVisibility     *foundation.ValueNotifier[bool]
	startInViewportHandle          foundation.ListenerHandle
	endInViewportHandle            foundation.ListenerHandle

	// endHandleDragPosition is the contact position of the gesture at the
	// current end handle location, in global coordinates.
	endHandleDragPosition float64
	// endHandleDragTarget is the distance from endHandleDragPosition to the
	// center of the line that it corresponds to, in global coordinates.
	endHandleDragTarget float64
	// dragStartSelection is the initial selection when a selection handle drag
	// has started. It is used on Apple platforms to preserve a collapsed
	// selection, and to anchor the non-dragged end of a non-collapsed one.
	dragStartSelection foundation.Opt[services.TextSelection]

	startHandleDragPosition float64
	startHandleDragTarget   float64
}

// NewTextSelectionOverlay creates an object that manages overlay entries for
// selection handles.
//
// The context must have an Overlay as an ancestor because this object will
// display the text selection handles in that Overlay.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.new
func NewTextSelectionOverlay(value services.TextEditingValue, bc BuildContext, toolbarLayerLink, startHandleLayerLink, endHandleLayerLink *rendering.LayerLink,
	renderObject *rendering.RenderEditable, selectionDelegate services.TextSelectionDelegate, opts ...TextSelectionOverlayOpts) *TextSelectionOverlay {
	o := firstOpts(opts)
	t := &TextSelectionOverlay{
		context: bc, renderObject: renderObject, selectionControls: o.SelectionControls, selectionDelegate: selectionDelegate,
		contextMenuBuilder: o.ContextMenuBuilder, value: value, handlesVisible: o.HandlesVisible,
		effectiveStartHandleVisibility: foundation.NewValueNotifier(false),
		effectiveEndHandleVisibility:   foundation.NewValueNotifier(false),
		effectiveToolbarVisibility:     foundation.NewValueNotifier(false),
	}
	t.startInViewportHandle = renderObject.SelectionStartInViewport().AddListener(t.updateTextSelectionOverlayVisibilities)
	t.endInViewportHandle = renderObject.SelectionEndInViewport().AddListener(t.updateTextSelectionOverlayVisibilities)
	t.updateTextSelectionOverlayVisibilities()
	t.selectionOverlay = NewSelectionOverlay(bc, rendering.TextSelectionHandleTypeCollapsed, 0, rendering.TextSelectionHandleTypeCollapsed, 0, nil,
		o.SelectionControls, startHandleLayerLink, endHandleLayerLink, toolbarLayerLink, SelectionOverlayOpts{
			DebugRequiredFor:        o.DebugRequiredFor,
			StartHandlesVisible:     t.effectiveStartHandleVisibility,
			OnStartHandleDragStart:  t.handleSelectionStartHandleDragStart,
			OnStartHandleDragUpdate: t.handleSelectionStartHandleDragUpdate,
			OnStartHandleDragEnd:    t.handleAnyDragEnd,
			EndHandlesVisible:       t.effectiveEndHandleVisibility,
			OnEndHandleDragStart:    t.handleSelectionEndHandleDragStart,
			OnEndHandleDragUpdate:   t.handleSelectionEndHandleDragUpdate,
			OnEndHandleDragEnd:      t.handleAnyDragEnd,
			ToolbarVisible:          t.effectiveToolbarVisibility,
			OnSelectionHandleTapped: o.OnSelectionHandleTapped,
			DragStartBehavior:       o.DragStartBehavior,
		})
	return t
}

// Context is the context in which the selection handles should appear.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.context
func (t *TextSelectionOverlay) Context() BuildContext { return t.context }

// RenderObject is the editable text in which the selection handles are
// anchored.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.renderObject
func (t *TextSelectionOverlay) RenderObject() *rendering.RenderEditable { return t.renderObject }

// SelectionControls builds the text selection handles.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.selectionControls
func (t *TextSelectionOverlay) SelectionControls() TextSelectionControls { return t.selectionControls }

// Value is the text editing value the handles show the selection of.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.value
func (t *TextSelectionOverlay) Value() services.TextEditingValue { return t.value }

func (t *TextSelectionOverlay) selection() services.TextSelection { return t.value.Selection }

func (t *TextSelectionOverlay) updateTextSelectionOverlayVisibilities() {
	t.effectiveStartHandleVisibility.SetValue(t.handlesVisible && t.renderObject.SelectionStartInViewport().Value())
	t.effectiveEndHandleVisibility.SetValue(t.handlesVisible && t.renderObject.SelectionEndInViewport().Value())
	t.effectiveToolbarVisibility.SetValue(t.renderObject.SelectionStartInViewport().Value() || t.renderObject.SelectionEndInViewport().Value())
}

// HandlesVisible is whether selection handles are visible.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.handlesVisible
func (t *TextSelectionOverlay) HandlesVisible() bool { return t.handlesVisible }

// SetHandlesVisible sets whether selection handles are visible. Set to false
// if you want to hide the handles. Use this property to show or hide the handle
// without rebuilding them.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.handlesVisible
func (t *TextSelectionOverlay) SetHandlesVisible(visible bool) {
	if t.handlesVisible == visible {
		return
	}
	t.handlesVisible = visible
	t.updateTextSelectionOverlayVisibilities()
}

// ShowHandles builds the handles by inserting them into the Overlay.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.showHandles
func (t *TextSelectionOverlay) ShowHandles() {
	t.updateSelectionOverlay()
	t.selectionOverlay.ShowHandles()
}

// HideHandles destroys the handles by removing them from the Overlay.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.hideHandles
func (t *TextSelectionOverlay) HideHandles() { t.selectionOverlay.HideHandles() }

// ShowToolbar shows the toolbar by inserting it into the Overlay. It must not
// be called during the build or layout phase.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.showToolbar
func (t *TextSelectionOverlay) ShowToolbar() {
	if foundation.DebugMode && WidgetsBindingOf(t.context).Scheduler().SchedulerPhase() == schedulerPhasePersistentCallbacks {
		panic(errors.New(tagShowToolbarDuringBuild, "ShowToolbar must not be called during the build or layout phase"))
	}
	t.updateSelectionOverlay()
	if t.contextMenuBuilder == nil {
		return
	}
	t.selectionOverlay.ShowToolbar(t.context, t.contextMenuBuilder)
}

// Update updates the text selection to the new value.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.update
func (t *TextSelectionOverlay) Update(newValue services.TextEditingValue) {
	if t.value.Equal(newValue) {
		return
	}
	t.value = newValue
	t.updateSelectionOverlay()
	// updateSelectionOverlay may not rebuild the selection overlay if the text
	// metrics and selection doesn't change even if the text has changed. This
	// rebuild is needed for the toolbar to update based on the latest text value.
	t.selectionOverlay.MarkNeedsBuild()
}

func (t *TextSelectionOverlay) updateSelectionOverlay() {
	o := t.selectionOverlay
	// Update selection handle metrics.
	o.SetStartHandleType(t.chooseType(t.renderObject.TextDirection(), rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight))
	o.SetLineHeightAtStart(t.getStartGlyphHeight())
	o.SetEndHandleType(t.chooseType(t.renderObject.TextDirection(), rendering.TextSelectionHandleTypeRight, rendering.TextSelectionHandleTypeLeft))
	o.SetLineHeightAtEnd(t.getEndGlyphHeight())
	// Update selection toolbar metrics.
	o.SetSelectionEndpoints(t.renderObject.GetEndpointsForSelection(t.selection()))
}

// UpdateForScroll causes the overlay to update its rendering. This is intended
// to be called when the RenderEditable scrolls.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.updateForScroll
func (t *TextSelectionOverlay) UpdateForScroll() {
	t.updateSelectionOverlay()
	// This method may be called due to windows metrics changes. In that case,
	// none of the properties in the selection overlay will change, but a rebuild
	// is still needed.
	t.selectionOverlay.MarkNeedsBuild()
}

// HandlesAreVisible is whether the handles are currently visible.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.handlesAreVisible
func (t *TextSelectionOverlay) HandlesAreVisible() bool {
	return t.selectionOverlay.handles != nil && t.HandlesVisible()
}

// ToolbarIsVisible is whether the toolbar is currently visible.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.toolbarIsVisible
func (t *TextSelectionOverlay) ToolbarIsVisible() bool { return t.selectionOverlay.ToolbarIsVisible() }

// Hide hides the entire overlay including the toolbar and the handles.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.hide
func (t *TextSelectionOverlay) Hide() { t.selectionOverlay.Hide() }

// HideToolbar hides the toolbar part of the overlay.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.hideToolbar
func (t *TextSelectionOverlay) HideToolbar() { t.selectionOverlay.HideToolbar() }

// Dispose disposes this object and releases resources.
//
// Flutter: widgets/text_selection.dart TextSelectionOverlay.dispose
func (t *TextSelectionOverlay) Dispose() {
	t.selectionOverlay.Dispose()
	t.renderObject.SelectionStartInViewport().RemoveListener(t.startInViewportHandle)
	t.renderObject.SelectionEndInViewport().RemoveListener(t.endInViewportHandle)
	t.effectiveToolbarVisibility.Dispose()
	t.effectiveStartHandleVisibility.Dispose()
	t.effectiveEndHandleVisibility.Dispose()
	t.HideToolbar()
}

func (t *TextSelectionOverlay) glyphHeight(selectedGrapheme func(selected string) (start, end int)) float64 {
	currText := t.selectionDelegate.TextEditingValue().Text
	sel := t.selection()
	// Only calculate handle rects if the text in the previous frame is the same
	// as the text in the current frame: the render object is the one of the
	// previous frame. In cases where the current frame is different from the
	// previous we fall back to the preferred line height.
	if t.renderObject.PlainText() == currText && sel.IsValid() && !sel.IsCollapsed() {
		selectedGraphemes := sel.Range().TextInside(currText)
		start, end := selectedGrapheme(selectedGraphemes)
		if rect, ok := t.renderObject.GetRectForComposingRange(ui.TextRange{Start: start, End: end}); ok {
			return rect.Height()
		}
	}
	return t.renderObject.PreferredLineHeight()
}

func (t *TextSelectionOverlay) getStartGlyphHeight() float64 {
	return t.glyphHeight(func(selected string) (int, int) {
		first := foundation.GraphemeOffset(selected, 1)
		return t.selection().Start(), t.selection().Start() + first
	})
}

func (t *TextSelectionOverlay) getEndGlyphHeight() float64 {
	return t.glyphHeight(func(selected string) (int, int) {
		boundaries := foundation.GraphemeBoundaries(selected)
		last := 0
		if len(boundaries) >= 2 {
			last = boundaries[len(boundaries)-1] - boundaries[len(boundaries)-2]
		}
		return t.selection().End() - last, t.selection().End()
	})
}

func (t *TextSelectionOverlay) isApple() bool {
	p := foundation.DefaultTargetPlatform()
	return p == foundation.TargetPlatformIOS || p == foundation.TargetPlatformMacOS
}

func (t *TextSelectionOverlay) handleDy(dragDy, handleDy float64) (float64, bool) {
	preferredLineHeight := t.renderObject.PreferredLineHeight()
	if preferredLineHeight <= 0 || math.IsInf(dragDy, 0) || math.IsNaN(dragDy) || math.IsInf(handleDy, 0) || math.IsNaN(handleDy) {
		return 0, false
	}
	distanceDragged := dragDy - handleDy
	dragDirection := 1.0
	if distanceDragged < 0 {
		dragDirection = -1
	}
	linesDragged := dragDirection * math.Floor(math.Abs(distanceDragged)/preferredLineHeight)
	return handleDy + linesDragged*preferredLineHeight, true
}

func (t *TextSelectionOverlay) handleSelectionEndHandleDragStart(details gestures.DragStartDetails) {
	if !t.renderObject.Attached() {
		return
	}
	t.endHandleDragPosition = details.GlobalPosition.Dy
	// Use local coordinates when dealing with line height, because in case of a
	// scale transformation, the line height will also be scaled.
	endpoints := t.selectionOverlay.SelectionEndpoints()
	centerOfLineLocal := endpoints[len(endpoints)-1].Point.Dy - t.renderObject.PreferredLineHeight()/2
	centerOfLineGlobal := t.renderObject.LocalToGlobal(ui.Offset{Dy: centerOfLineLocal}, nil).Dy
	t.endHandleDragTarget = centerOfLineGlobal - details.GlobalPosition.Dy
	// The drag start selection is only utilized on Apple platforms.
	if t.isApple() && t.dragStartSelection.IsNone() {
		t.dragStartSelection = foundation.Some(t.selection())
	}
}

func (t *TextSelectionOverlay) handleSelectionEndHandleDragUpdate(details gestures.DragUpdateDetails) {
	if !t.renderObject.Attached() {
		return
	}
	// This is NOT the same as details.localPosition. That is relative to the
	// selection handle, whereas this is relative to the RenderEditable.
	localPosition := t.renderObject.GlobalToLocal(details.GlobalPosition, nil)
	next, ok := t.handleDy(localPosition.Dy, t.renderObject.GlobalToLocal(ui.Offset{Dy: t.endHandleDragPosition}, nil).Dy)
	if !ok {
		return
	}
	t.endHandleDragPosition = t.renderObject.LocalToGlobal(ui.Offset{Dy: next}, nil).Dy
	handleTargetGlobal := ui.Offset{Dx: details.GlobalPosition.Dx, Dy: t.endHandleDragPosition + t.endHandleDragTarget}
	position := t.renderObject.GetPositionForPoint(handleTargetGlobal)

	var newSelection services.TextSelection
	if t.isApple() {
		// On Apple platforms, dragging the base handle makes it the extent.
		start := t.dragStartSelection.OrElse(t.selection())
		if start.IsCollapsed() {
			t.handleSelectionHandleChanged(services.NewTextSelectionCollapsed(position.Offset, services.TextSelectionOpts{Affinity: position.Affinity}))
			return
		}
		// Use this instead of dragStartSelection.isNormalized because
		// TextRange.isNormalized always returns true for a TextSelection.
		base := start.ExtentOffset
		if start.ExtentOffset >= start.BaseOffset {
			base = start.BaseOffset
		}
		newSelection = services.NewTextSelection(base, position.Offset)
	} else {
		if t.selection().IsCollapsed() {
			t.handleSelectionHandleChanged(services.NewTextSelectionCollapsed(position.Offset, services.TextSelectionOpts{Affinity: position.Affinity}))
			return
		}
		newSelection = services.NewTextSelection(t.selection().BaseOffset, position.Offset)
		if newSelection.BaseOffset >= newSelection.ExtentOffset {
			return // Don't allow order swapping.
		}
	}
	t.handleSelectionHandleChanged(newSelection)
}

func (t *TextSelectionOverlay) handleSelectionStartHandleDragStart(details gestures.DragStartDetails) {
	if !t.renderObject.Attached() {
		return
	}
	t.startHandleDragPosition = details.GlobalPosition.Dy
	endpoints := t.selectionOverlay.SelectionEndpoints()
	centerOfLineLocal := endpoints[0].Point.Dy - t.renderObject.PreferredLineHeight()/2
	centerOfLineGlobal := t.renderObject.LocalToGlobal(ui.Offset{Dy: centerOfLineLocal}, nil).Dy
	t.startHandleDragTarget = centerOfLineGlobal - details.GlobalPosition.Dy
	// The drag start selection is only utilized on Apple platforms.
	if t.isApple() && t.dragStartSelection.IsNone() {
		t.dragStartSelection = foundation.Some(t.selection())
	}
}

func (t *TextSelectionOverlay) handleSelectionStartHandleDragUpdate(details gestures.DragUpdateDetails) {
	if !t.renderObject.Attached() {
		return
	}
	localPosition := t.renderObject.GlobalToLocal(details.GlobalPosition, nil)
	next, ok := t.handleDy(localPosition.Dy, t.renderObject.GlobalToLocal(ui.Offset{Dy: t.startHandleDragPosition}, nil).Dy)
	if !ok {
		return
	}
	t.startHandleDragPosition = t.renderObject.LocalToGlobal(ui.Offset{Dy: next}, nil).Dy
	handleTargetGlobal := ui.Offset{Dx: details.GlobalPosition.Dx, Dy: t.startHandleDragPosition + t.startHandleDragTarget}
	position := t.renderObject.GetPositionForPoint(handleTargetGlobal)

	var newSelection services.TextSelection
	if t.isApple() {
		start := t.dragStartSelection.OrElse(t.selection())
		if start.IsCollapsed() {
			t.handleSelectionHandleChanged(services.NewTextSelectionCollapsed(position.Offset, services.TextSelectionOpts{Affinity: position.Affinity}))
			return
		}
		base := start.BaseOffset
		if start.ExtentOffset >= start.BaseOffset {
			base = start.ExtentOffset
		}
		newSelection = services.NewTextSelection(base, position.Offset)
	} else {
		if t.selection().IsCollapsed() {
			t.handleSelectionHandleChanged(services.NewTextSelectionCollapsed(position.Offset, services.TextSelectionOpts{Affinity: position.Affinity}))
			return
		}
		newSelection = services.NewTextSelection(position.Offset, t.selection().ExtentOffset)
		if newSelection.BaseOffset >= newSelection.ExtentOffset {
			return // Don't allow order swapping.
		}
	}
	t.handleSelectionHandleChanged(newSelection)
}

func (t *TextSelectionOverlay) handleAnyDragEnd(gestures.DragEndDetails) {
	if !t.context.Mounted() {
		return
	}
	t.dragStartSelection = foundation.None[services.TextSelection]()
	draggingHandles := t.selectionOverlay.IsDraggingStartHandle() || t.selectionOverlay.IsDraggingEndHandle()
	if !draggingHandles && !t.selection().IsCollapsed() {
		t.selectionOverlay.ShowToolbar(t.context, t.contextMenuBuilder)
	}
}

func (t *TextSelectionOverlay) handleSelectionHandleChanged(newSelection services.TextSelection) {
	t.selectionDelegate.UserUpdateTextEditingValue(t.value.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(newSelection)}), services.SelectionChangedCauseDrag)
}

func (t *TextSelectionOverlay) chooseType(textDirection ui.TextDirection, ltrType, rtlType rendering.TextSelectionHandleType) rendering.TextSelectionHandleType {
	if t.selection().IsCollapsed() {
		return rendering.TextSelectionHandleTypeCollapsed
	}
	if textDirection == ui.TextDirectionRTL {
		return rtlType
	}
	return ltrType
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShowToolbarDuringBuild = "ShowToolbarDuringBuild"
)
