package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// toolbarShowData is the value and the selection bounds that were there when
// the toolbar was hidden for a scroll.
type toolbarShowData struct {
	value           services.TextEditingValue
	selectionBounds ui.Rect
}

// SelectionOverlay is the overlay of the selection handles and toolbar of the
// field, or nil when there is none.
//
// Flutter: widgets/editable_text.dart EditableTextState.selectionOverlay
func (s *EditableTextState) SelectionOverlay() *TextSelectionOverlay { return s.selectionOverlay }

// contextMenuBuilder dispatches to the current ContextMenuBuilder of the widget,
// so that the overlay doesn't have to be recreated when only the builder
// changes.
//
// Flutter: widgets/editable_text.dart EditableTextState._contextMenuBuilder
func (s *EditableTextState) contextMenuBuilder(bc BuildContext) Widget {
	return s.Widget().o.ContextMenuBuilder(bc, s)
}

// createSelectionOverlay creates the overlay of the selection handles and toolbar.
//
// Flutter: widgets/editable_text.dart EditableTextState._createSelectionOverlay
func (s *EditableTextState) createSelectionOverlay() *TextSelectionOverlay {
	w := s.Widget()
	var builder WidgetBuilder
	if w.o.ContextMenuBuilder != nil {
		builder = s.contextMenuBuilder
	}
	return NewTextSelectionOverlay(s.value(), s.Context(), s.toolbarLayerLink, s.startHandleLink, s.endHandleLink, s.RenderEditable(), s,
		TextSelectionOverlayOpts{
			DebugRequiredFor: w, SelectionControls: w.o.SelectionControls, DragStartBehavior: w.o.DragStartBehavior,
			OnSelectionHandleTapped: w.o.OnSelectionHandleTapped, ContextMenuBuilder: builder,
		})
}

// updateOrDisposeSelectionOverlayIfNeeded updates the overlay while the field
// has the focus and disposes it when the focus is lost.
//
// Flutter: widgets/editable_text.dart EditableTextState._updateOrDisposeSelectionOverlayIfNeeded
func (s *EditableTextState) updateOrDisposeSelectionOverlayIfNeeded() {
	if s.selectionOverlay == nil {
		return
	}
	if s.hasFocus() {
		s.selectionOverlay.Update(s.value())
	} else {
		s.selectionOverlay.Dispose()
		s.selectionOverlay = nil
	}
}

// updateSelectionOverlayAfterWidgetChange rebuilds or updates the overlay when
// the widget changed.
//
// Flutter: widgets/editable_text.dart EditableTextState.didUpdateWidget
func (s *EditableTextState) updateSelectionOverlayAfterWidgetChange(old *EditableText) {
	w := s.Widget()
	// If only the identity of the context menu builder changed, the overlay does
	// not need to be recreated: just rebuild the toolbar that is shown, so its
	// overlay entry picks up the new builder.
	if o := s.selectionOverlay; o != nil && o.ToolbarIsVisible() && (w.o.ContextMenuBuilder == nil) == (old.o.ContextMenuBuilder == nil) {
		// Deferred to the next frame because ShowToolbar needs a fully laid-out
		// render tree, and DidUpdateWidget is called before layout.
		s.binding().RunPostFrame(func(time.Duration) {
			if s.Mounted() && s.selectionOverlay != nil && s.selectionOverlay.ToolbarIsVisible() {
				s.selectionOverlay.ShowToolbar()
			}
		})
	}
	if s.selectionOverlay != nil &&
		((w.o.ContextMenuBuilder == nil) != (old.o.ContextMenuBuilder == nil) || w.o.SelectionControls != old.o.SelectionControls ||
			w.o.DragStartBehavior != old.o.DragStartBehavior) {
		shouldShowToolbar := s.selectionOverlay.ToolbarIsVisible()
		shouldShowHandles := s.selectionOverlay.HandlesVisible()
		s.selectionOverlay.Dispose()
		s.selectionOverlay = s.createSelectionOverlay()
		if shouldShowToolbar || shouldShowHandles {
			s.binding().RunPostFrame(func(time.Duration) {
				if !s.Mounted() || s.selectionOverlay == nil {
					return
				}
				if shouldShowToolbar {
					s.selectionOverlay.ShowToolbar()
				}
				if shouldShowHandles {
					s.selectionOverlay.ShowHandles()
				}
			})
		}
	} else if w.controller.Selection() != old.controller.Selection() && s.selectionOverlay != nil {
		s.selectionOverlay.Update(s.value())
	}
	if s.selectionOverlay != nil {
		s.selectionOverlay.SetHandlesVisible(w.o.ShowSelectionHandles)
	}
}

// updateSelectionOverlayAfterSelectionChange creates or updates the overlay and
// shows its handles, or disposes it when the field has no way to show one.
//
// Flutter: widgets/editable_text.dart EditableTextState._handleSelectionChanged
func (s *EditableTextState) updateSelectionOverlayAfterSelectionChange() {
	w := s.Widget()
	if w.o.SelectionControls == nil && w.o.ContextMenuBuilder == nil {
		if s.selectionOverlay != nil {
			s.selectionOverlay.Dispose()
			s.selectionOverlay = nil
		}
		return
	}
	if s.selectionOverlay == nil {
		s.selectionOverlay = s.createSelectionOverlay()
	} else {
		s.selectionOverlay.Update(s.value())
	}
	s.selectionOverlay.SetHandlesVisible(w.o.ShowSelectionHandles)
	s.selectionOverlay.ShowHandles()
}

// GetGlyphHeights returns the heights of the glyphs at the start and at the
// end of the selection.
//
// Flutter: widgets/editable_text.dart EditableTextState.getGlyphHeights
func (s *EditableTextState) GetGlyphHeights() (startGlyphHeight, endGlyphHeight float64) {
	r := s.RenderEditable()
	selection := s.value().Selection
	// Only calculate the rects if the text in the previous frame is the same as
	// in the current one: the render object has the text of the previous frame.
	// Otherwise getRectForComposingRange might fail, so fall back to the
	// preferred line height.
	currText := s.value().Text
	if r.PlainText() != currText || !selection.IsValid() || selection.IsCollapsed() {
		return r.PreferredLineHeight(), r.PreferredLineHeight()
	}
	selected := selection.Range().TextInside(currText)
	boundaries := foundation.GraphemeBoundaries(selected)
	first := foundation.GraphemeOffset(selected, 1)
	last := 0
	if len(boundaries) >= 2 {
		last = boundaries[len(boundaries)-1] - boundaries[len(boundaries)-2]
	}
	startGlyphHeight, endGlyphHeight = r.PreferredLineHeight(), r.PreferredLineHeight()
	if rect, ok := r.GetRectForComposingRange(ui.TextRange{Start: selection.Start(), End: selection.Start() + first}); ok {
		startGlyphHeight = rect.Height()
	}
	if rect, ok := r.GetRectForComposingRange(ui.TextRange{Start: selection.End() - last, End: selection.End()}); ok {
		endGlyphHeight = rect.Height()
	}
	return startGlyphHeight, endGlyphHeight
}

// ContextMenuAnchors returns the anchor points for the default context menu.
//
// Flutter: widgets/editable_text.dart EditableTextState.contextMenuAnchors
func (s *EditableTextState) ContextMenuAnchors() TextSelectionToolbarAnchors {
	r := s.RenderEditable()
	if p, ok := r.LastSecondaryTapDownPosition().Get(); ok {
		return TextSelectionToolbarAnchors{PrimaryAnchor: p}
	}
	startGlyphHeight, endGlyphHeight := s.GetGlyphHeights()
	points := r.GetEndpointsForSelection(s.value().Selection)
	return NewTextSelectionToolbarAnchorsFromSelection(r, startGlyphHeight, endGlyphHeight, points)
}

// ContextMenuButtonItems returns the items representing the buttons in this
// platform's default selection menu for the field.
//
// Deviation: the look up, search web, share and Live Text buttons and the
// buttons of the text processing actions are not ported (SPEC §12.10).
//
// Flutter: widgets/editable_text.dart EditableTextState.contextMenuButtonItems
func (s *EditableTextState) ContextMenuButtonItems() []ContextMenuButtonItem {
	var onCopy, onCut, onPaste, onSelectAll foundation.VoidCallback
	if s.CopyEnabled() {
		onCopy = func() { s.CopySelection(services.SelectionChangedCauseToolbar) }
	}
	if s.CutEnabled() {
		onCut = func() { s.CutSelection(services.SelectionChangedCauseToolbar) }
	}
	if s.PasteEnabled() {
		onPaste = func() { s.PasteText(services.SelectionChangedCauseToolbar) }
	}
	if s.SelectAllEnabled() {
		onSelectAll = func() { s.SelectAll(services.SelectionChangedCauseToolbar) }
	}
	return GetEditableButtonItems(s.clipboardStatus.Value(), EditableButtonItemsCallbacks{
		OnCopy: onCopy, OnCut: onCut, OnPaste: onPaste, OnSelectAll: onSelectAll,
	})
}

// EditableButtonItemsCallbacks are the callbacks of the buttons that
// GetEditableButtonItems returns; a nil one leaves the button out.
//
// Flutter: widgets/editable_text.dart EditableText.getEditableButtonItems
type EditableButtonItemsCallbacks struct {
	OnCopy, OnCut, OnPaste, OnSelectAll, OnLookUp, OnSearchWeb, OnShare, OnLiveTextInput foundation.VoidCallback
}

// GetEditableButtonItems returns the items representing the buttons in this
// platform's default selection menu for an editable field. For example,
// EditableText uses this to generate the default buttons for its context menu.
//
// Flutter: widgets/editable_text.dart EditableText.getEditableButtonItems
func GetEditableButtonItems(clipboardStatus ClipboardStatus, c EditableButtonItemsCallbacks) []ContextMenuButtonItem {
	var items []ContextMenuButtonItem
	add := func(onPressed foundation.VoidCallback, t ContextMenuButtonType) {
		if onPressed != nil {
			items = append(items, ContextMenuButtonItem{OnPressed: onPressed, Type: t})
		}
	}
	// If the paste button is enabled, don't render anything until the state of
	// the clipboard is known, since it's used to determine if paste is shown.
	if c.OnPaste == nil || clipboardStatus != ClipboardStatusUnknown {
		// On Android, the share button is before the select all button.
		showShareBeforeSelectAll := foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid
		add(c.OnCut, ContextMenuButtonTypeCut)
		add(c.OnCopy, ContextMenuButtonTypeCopy)
		add(c.OnPaste, ContextMenuButtonTypePaste)
		if showShareBeforeSelectAll {
			add(c.OnShare, ContextMenuButtonTypeShare)
		}
		add(c.OnSelectAll, ContextMenuButtonTypeSelectAll)
		add(c.OnLookUp, ContextMenuButtonTypeLookUp)
		add(c.OnSearchWeb, ContextMenuButtonTypeSearchWeb)
		if !showShareBeforeSelectAll {
			add(c.OnShare, ContextMenuButtonTypeShare)
		}
	}
	add(c.OnLiveTextInput, ContextMenuButtonTypeLiveTextInput)
	return items
}

func platformSupportsFadeOnScroll() bool {
	p := foundation.DefaultTargetPlatform()
	return p == foundation.TargetPlatformAndroid || p == foundation.TargetPlatformIOS
}

func (s *EditableTextState) disposeScrollNotificationObserver() {
	s.listeningToScrollObserver = false
	if s.scrollObserver != nil {
		s.scrollObserver.RemoveListener(s.scrollObserverHandle)
		s.scrollObserver = nil
	}
}

func (s *EditableTextState) isInternalScrollableNotification(nc BuildContext) bool {
	if nc == nil {
		return s.scrollableKey.CurrentContext() == nil
	}
	st, ok := FindAncestorStateOfType[*ScrollableState](nc)
	var ctx BuildContext
	if ok {
		ctx = st.Context()
	}
	return s.scrollableKey.CurrentContext() == ctx
}

func (s *EditableTextState) scrollableNotificationIsFromSameSubtree(nc BuildContext) bool {
	if nc == nil {
		return false
	}
	target, ok := FindAncestorStateOfType[*ScrollableState](nc)
	if !ok {
		return false
	}
	current := s.Context()
	for current != nil {
		st, ok := FindAncestorStateOfType[*ScrollableState](current)
		if !ok {
			return false
		}
		if st == target {
			return true
		}
		current = st.Context()
	}
	return false
}

func (s *EditableTextState) handleContextMenuOnParentScroll(n ScrollNotification) {
	// Do some preliminary checks to avoid expensive subtree traversal.
	_, start := n.(*ScrollStartNotification)
	_, end := n.(*ScrollEndNotification)
	if !start && !end {
		return
	}
	switch {
	case start && s.dataWhenToolbarShow != nil, end && s.dataWhenToolbarShow == nil:
	case end && s.dataWhenToolbarShow.value != s.value():
		s.dataWhenToolbarShow = nil
		s.disposeScrollNotificationObserver()
	case !s.isInternalScrollableNotification(n.Context()) && s.scrollableNotificationIsFromSameSubtree(n.Context()):
		s.handleContextMenuOnScroll(n)
	}
}

func (s *EditableTextState) calculateDeviceRect() ui.Rect {
	size, _ := MediaQueryMaybeSizeOf(s.Context())
	padding, _ := MediaQueryMaybeViewPaddingOf(s.Context())
	insets, _ := MediaQueryMaybeViewInsetsOf(s.Context())
	obscuredVertical := padding.Top + padding.Bottom + insets.Bottom
	obscuredHorizontal := padding.Left + padding.Right
	return ui.NewRectFromLTWH(padding.Left, padding.Top, size.Width-obscuredHorizontal, size.Height-obscuredVertical)
}

func (s *EditableTextState) selectionInViewport(selectionBounds ui.Rect) bool {
	r := s.RenderEditable()
	closest := rendering.RenderAbstractViewportMaybeOf(r)
	for closest != nil {
		local := r.GetTransformTo(closest).TransformRect(selectionBounds)
		paint := closest.PaintBounds()
		if local.HasNaN() || paint.HasNaN() || !paint.Overlaps(local) {
			return false
		}
		closest = rendering.RenderAbstractViewportMaybeOf(closest.Parent())
	}
	return true
}

// handleContextMenuOnScroll updates the overlay when the field scrolls, and on
// mobile platforms hides the toolbar until the scrolling ends.
//
// Deviation: the web context menu is not applicable (SPEC §12.10).
//
// Flutter: widgets/editable_text.dart EditableTextState._handleContextMenuOnScroll
func (s *EditableTextState) handleContextMenuOnScroll(n ScrollNotification) {
	if !platformSupportsFadeOnScroll() {
		if s.selectionOverlay != nil {
			s.selectionOverlay.UpdateForScroll()
		}
		return
	}
	// When the scroll begins and the toolbar is visible, hide it until scrolling
	// ends. The selection and the render object need to be visible within the
	// current viewport for the toolbar to show when scrolling ends. If they are
	// not then the toolbar is shown when they are scrolled back into view,
	// unless invalidated by a change in the value.
	switch n.(type) {
	case *ScrollStartNotification:
		if s.dataWhenToolbarShow != nil || s.selectionOverlay == nil || !s.selectionOverlay.ToolbarIsVisible() {
			return
		}
		r := s.RenderEditable()
		v := s.value()
		boxes := r.GetBoxesForSelection(v.Selection)
		var bounds ui.Rect
		if v.Selection.IsCollapsed() || len(boxes) == 0 {
			bounds = r.GetLocalRectForCaret(v.Selection.Extent())
		} else {
			bounds = boxes[0].ToRect()
			for _, b := range boxes[1:] {
				bounds = bounds.ExpandToInclude(b.ToRect())
			}
		}
		s.dataWhenToolbarShow = &toolbarShowData{value: v, selectionBounds: bounds}
		s.selectionOverlay.HideToolbar()
	case *ScrollEndNotification:
		if s.dataWhenToolbarShow == nil {
			return
		}
		if s.dataWhenToolbarShow.value != s.value() {
			// The value has changed so we should invalidate any toolbar scheduling.
			s.dataWhenToolbarShow = nil
			s.disposeScrollNotificationObserver()
			return
		}
		if s.showToolbarOnScreenScheduled {
			return
		}
		s.showToolbarOnScreenScheduled = true
		schedule := func(time.Duration) {
			s.showToolbarOnScreenScheduled = false
			if !s.Mounted() || s.dataWhenToolbarShow == nil {
				return
			}
			if s.dataWhenToolbarShow.value != s.value() {
				s.dataWhenToolbarShow = nil
				s.disposeScrollNotificationObserver()
				return
			}
			r := s.RenderEditable()
			deviceRect := s.calculateDeviceRect()
			visibleInEditable := r.SelectionStartInViewport().Value() || r.SelectionEndInViewport().Value()
			bounds := r.GetTransformTo(nil).TransformRect(s.dataWhenToolbarShow.selectionBounds)
			overlaps := !bounds.HasNaN() && deviceRect.Overlaps(bounds)
			if visibleInEditable && overlaps && s.selectionInViewport(s.dataWhenToolbarShow.selectionBounds) {
				s.TryShowToolbar()
				s.dataWhenToolbarShow = nil
			}
		}
		sched := s.binding().Scheduler()
		switch sched.SchedulerPhase() {
		case scheduler.SchedulerPhaseIdle, scheduler.SchedulerPhasePostFrameCallbacks:
			// During these scheduler phases we cannot guarantee there will be a
			// frame after, so we schedule a frame callback.
			sched.ScheduleFrameCallback(schedule)
		default:
			// During an active frame we can still schedule a post-frame callback
			// to be run after the current frame.
			sched.AddPostFrameCallback(schedule)
		}
	}
}

// ensureSelectionOverlay creates the overlay if there is none.
func (s *EditableTextState) ensureSelectionOverlay() *TextSelectionOverlay {
	if s.selectionOverlay == nil {
		s.selectionOverlay = s.createSelectionOverlay()
	}
	return s.selectionOverlay
}

// hideToolbarIfVisible hides the toolbar if it is visible, and otherwise lets
// the dismiss intent through.
//
// Flutter: widgets/editable_text.dart EditableTextState._hideToolbarIfVisible
func (s *EditableTextState) hideToolbarIfVisible(i DismissIntent) any {
	if s.selectionOverlay != nil && s.selectionOverlay.ToolbarIsVisible() {
		s.HideToolbar(false)
		return nil
	}
	return ActionsInvoke(s.Context(), i)
}

// scrollPaddingWithHandles is the padding the field enlarges the caret rect by
// to keep the caret and its handle from the edge.
func (s *EditableTextState) caretPadding(lineHeight float64) painting.EdgeInsets {
	padding := s.Widget().o.ScrollPadding.OrElse(defaultEditableScrollPadding)
	bottom := padding.Bottom
	if s.selectionOverlay != nil && s.selectionOverlay.SelectionControls() != nil {
		c := s.selectionOverlay.SelectionControls()
		handleHeight := c.GetHandleSize(lineHeight).Height
		interactive := max(handleHeight, MinInteractiveDimension)
		anchor := c.GetHandleAnchor(rendering.TextSelectionHandleTypeCollapsed, lineHeight)
		handleCenter := handleHeight/2 - anchor.Dy
		bottom = max(handleCenter+interactive/2, bottom)
	}
	padding.Bottom = bottom
	return padding
}
