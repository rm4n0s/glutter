package widgets

import (
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// selectableVerticalComparingThreshold: in practice some selectables like
// widgetspan shift several pixels. So when the vertical position diff is within
// the threshold, compare the horizontal position to make the compare function
// more robust.
//
// Flutter: widgets/selectable_region.dart _kSelectableVerticalComparingThreshold
const selectableVerticalComparingThreshold = 3.0

// selectionHandleDrawableAreaPadding is the number of additional pixels added
// to the selection handle drawable area. Selection handles that are outside of
// the drawable area will be hidden. That logic prevents handles that get
// scrolled off the viewport from being drawn on the screen.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate._kSelectionHandleDrawableAreaPadding
const selectionHandleDrawableAreaPadding = 5.0

// MultiSelectableDelegate is what MultiSelectableSelectionContainerDelegate
// calls on the delegate that embeds it, so the delegate can override the
// behaviors that Flutter's subclasses override (SPEC §12.5, the self pattern).
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate
type MultiSelectableDelegate interface {
	SelectionContainerDelegate
	// NotifyListeners notifies the listeners of the delegate.
	//
	// Flutter: foundation/change_notifier.dart ChangeNotifier.notifyListeners
	NotifyListeners()
	// CompareOrder is the compare function this delegate uses for determining
	// the selection order of the selectables; it defaults to the screen order.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.compareOrder
	CompareOrder(a, b rendering.Selectable) int
	// DidChangeSelectables is called when this delegate finishes updating the
	// selectables.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.didChangeSelectables
	DidChangeSelectables()
	// EnsureChildUpdated ensures the selectable child has received up to date
	// selection events. It is called when a new selectable is added to the
	// delegate, and its screen location falls into the previous selection.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.ensureChildUpdated
	EnsureChildUpdated(selectable rendering.Selectable)
	// DispatchSelectionEventToChild dispatches a selection event to a specific
	// selectable.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.dispatchSelectionEventToChild
	DispatchSelectionEventToChild(selectable rendering.Selectable, event rendering.SelectionEvent) rendering.SelectionResult
	// HandleSelectAll selects all contents of all selectables.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectAll
	HandleSelectAll(event rendering.SelectAllSelectionEvent) rendering.SelectionResult
	// HandleSelectWord selects a word in a selectable at the event's location.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectWord
	HandleSelectWord(event rendering.SelectWordSelectionEvent) rendering.SelectionResult
	// HandleSelectParagraph selects a paragraph in a selectable at the event's
	// location.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectParagraph
	HandleSelectParagraph(event rendering.SelectParagraphSelectionEvent) rendering.SelectionResult
	// HandleClearSelection removes the selection of all selectables.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleClearSelection
	HandleClearSelection(event rendering.ClearSelectionEvent) rendering.SelectionResult
	// HandleGranularlyExtendSelection extends the current selection in a
	// certain granularity.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleGranularlyExtendSelection
	HandleGranularlyExtendSelection(event rendering.GranularlyExtendSelectionEvent) rendering.SelectionResult
	// HandleDirectionallyExtendSelection extends the current selection in a
	// direction.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleDirectionallyExtendSelection
	HandleDirectionallyExtendSelection(event rendering.DirectionallyExtendSelectionEvent) rendering.SelectionResult
	// HandleSelectionEdgeUpdate updates the selection edges.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectionEdgeUpdate
	HandleSelectionEdgeUpdate(event rendering.SelectionEdgeUpdateEvent) rendering.SelectionResult
}

// MultiSelectableSelectionContainerDelegate is a delegate that handles events
// and updates for multiple Selectable children.
//
// Updates are optimized by tracking which Selectables reside on the edges of a
// selection. Delegates that embed it implement EnsureChildUpdated to describe
// how a Selectable should behave when added to a selection, and call
// InitMultiSelectableSelectionContainerDelegate with themselves.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate
type MultiSelectableSelectionContainerDelegate struct {
	SelectionContainerDelegateBase
	foundation.ChangeNotifier
	self MultiSelectableDelegate

	// Selectables is the list of selectables this delegate is managing.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.selectables
	Selectables []rendering.Selectable
	// CurrentSelectionEndIndex is the current selectable that contains the
	// selection end edge, or -1.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.currentSelectionEndIndex
	CurrentSelectionEndIndex int
	// CurrentSelectionStartIndex is the current selectable that contains the
	// selection start edge, or -1.
	//
	// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.currentSelectionStartIndex
	CurrentSelectionStartIndex int

	startHandleLayer      *rendering.LayerLink
	startHandleLayerOwner rendering.Selectable
	endHandleLayer        *rendering.LayerLink
	endHandleLayerOwner   rendering.Selectable

	isHandlingSelectionEvent  bool
	scheduledSelectableUpdate bool
	selectionInProgress       bool
	additions                 []rendering.Selectable
	extendSelectionInProgress bool
	geometryHandles           map[rendering.Selectable]foundation.ListenerHandle
	selectionGeometry         rendering.SelectionGeometry
}

// InitMultiSelectableSelectionContainerDelegate sets the delegate that
// embeds the base, so that the base calls its overrides.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitMultiSelectableSelectionContainerDelegate(m *MultiSelectableSelectionContainerDelegate, self MultiSelectableDelegate) {
	m.self = self
	m.CurrentSelectionEndIndex = -1
	m.CurrentSelectionStartIndex = -1
	m.geometryHandles = map[rendering.Selectable]foundation.ListenerHandle{}
	m.selectionGeometry = rendering.SelectionGeometry{Status: rendering.SelectionStatusNone}
}

// Add adds the selectable; it takes part in the selection after the next frame.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.add
func (m *MultiSelectableSelectionContainerDelegate) Add(selectable rendering.Selectable) {
	if foundation.DebugMode && slices.Contains(m.Selectables, selectable) {
		panic(errors.New("SelectableAlreadyAdded", "the selectable is already in this registrar"))
	}
	m.additions = append(m.additions, selectable)
	m.scheduleSelectableUpdate()
}

// Remove removes the selectable.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.remove
func (m *MultiSelectableSelectionContainerDelegate) Remove(selectable rendering.Selectable) {
	if i := slices.Index(m.additions, selectable); i >= 0 {
		// The same selectable was added in the same frame and is not yet
		// incorporated into the selectables. Removing such selectable doesn't
		// require selection geometry update.
		m.additions = slices.Delete(m.additions, i, i+1)
		return
	}
	m.removeSelectable(selectable)
	m.scheduleSelectableUpdate()
}

// LayoutDidChange notifies this delegate that layout of the container has
// changed.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.layoutDidChange
func (m *MultiSelectableSelectionContainerDelegate) LayoutDidChange() { m.updateSelectionGeometry() }

func (m *MultiSelectableSelectionContainerDelegate) binding() *WidgetsBinding {
	return m.bindingOfContext
}

func (m *MultiSelectableSelectionContainerDelegate) scheduleSelectableUpdate() {
	if m.scheduledSelectableUpdate {
		return
	}
	m.scheduledSelectableUpdate = true
	run := func() {
		if !m.scheduledSelectableUpdate {
			return
		}
		m.scheduledSelectableUpdate = false
		m.updateSelectables()
	}
	b := m.binding()
	if b.Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePostFrameCallbacks {
		// A new task can be scheduled as a result of running the scheduled task
		// from another delegate. This can happen if nesting two
		// SelectionContainers. The selectable can be safely updated in the same
		// frame in this case.
		b.Post(run)
	} else {
		b.RunPostFrame(func(time.Duration) { run() })
	}
}

func (m *MultiSelectableSelectionContainerDelegate) updateSelectables() {
	if len(m.additions) > 0 {
		m.flushAdditions()
	}
	m.self.DidChangeSelectables()
}

func (m *MultiSelectableSelectionContainerDelegate) flushAdditions() {
	merging := slices.Clone(m.additions)
	slices.SortStableFunc(merging, m.self.CompareOrder)
	existing := m.Selectables
	m.Selectables = nil
	mergingIndex, existingIndex := 0, 0
	selectionStartIndex, selectionEndIndex := m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex
	// Merge two sorted lists.
	for mergingIndex < len(merging) || existingIndex < len(existing) {
		if mergingIndex >= len(merging) || (existingIndex < len(existing) && m.self.CompareOrder(existing[existingIndex], merging[mergingIndex]) < 0) {
			if existingIndex == m.CurrentSelectionStartIndex {
				selectionStartIndex = len(m.Selectables)
			}
			if existingIndex == m.CurrentSelectionEndIndex {
				selectionEndIndex = len(m.Selectables)
			}
			m.Selectables = append(m.Selectables, existing[existingIndex])
			existingIndex++
			continue
		}
		// If the merging selectable falls in the selection range, their
		// selection needs to be updated.
		s := merging[mergingIndex]
		if existingIndex < max(m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex) &&
			existingIndex > min(m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex) {
			m.self.EnsureChildUpdated(s)
		}
		m.geometryHandles[s] = s.AddListener(m.handleSelectableGeometryChange)
		m.Selectables = append(m.Selectables, s)
		mergingIndex++
	}
	m.CurrentSelectionEndIndex = selectionEndIndex
	m.CurrentSelectionStartIndex = selectionStartIndex
	m.additions = nil
}

func (m *MultiSelectableSelectionContainerDelegate) removeSelectable(selectable rendering.Selectable) {
	index := slices.Index(m.Selectables, selectable)
	if index < 0 {
		if foundation.DebugMode {
			panic(errors.New("SelectableNotInRegistrar", "the selectable is not in this registrar"))
		}
		return
	}
	m.Selectables = slices.Delete(m.Selectables, index, index+1)
	if index <= m.CurrentSelectionEndIndex {
		m.CurrentSelectionEndIndex--
	}
	if index <= m.CurrentSelectionStartIndex {
		m.CurrentSelectionStartIndex--
	}
	selectable.RemoveListener(m.geometryHandles[selectable])
	delete(m.geometryHandles, selectable)
}

// DidChangeSelectables is called when this delegate finishes updating the
// selectables.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.didChangeSelectables
func (m *MultiSelectableSelectionContainerDelegate) DidChangeSelectables() {
	m.updateSelectionGeometry()
}

// Value is the geometry of the selection of all the selectables.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.value
func (m *MultiSelectableSelectionContainerDelegate) Value() rendering.SelectionGeometry {
	return m.selectionGeometry
}

// updateSelectionGeometry updates the value in this class and notifies
// listeners if necessary.
func (m *MultiSelectableSelectionContainerDelegate) updateSelectionGeometry() {
	newValue := m.GetSelectionGeometry()
	if !m.selectionGeometry.Equal(newValue) {
		m.selectionGeometry = newValue
		m.self.NotifyListeners()
	}
	m.updateHandleLayersAndOwners()
}

func selectableBoundingBox(s rendering.Selectable) ui.Rect {
	boxes := s.BoundingBoxes()
	result := boxes[0]
	for _, b := range boxes[1:] {
		result = result.ExpandToInclude(b)
	}
	return result
}

// CompareOrder compares the selectables in the screen order.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.compareOrder
func (m *MultiSelectableSelectionContainerDelegate) CompareOrder(a, b rendering.Selectable) int {
	rectA := a.GetTransformTo(nil).TransformRect(selectableBoundingBox(a))
	rectB := b.GetTransformTo(nil).TransformRect(selectableBoundingBox(b))
	if result := compareRectsVertically(rectA, rectB); result != 0 {
		return result
	}
	return compareRectsHorizontally(rectA, rectB)
}

// compareRectsVertically compares two rectangles in the screen order solely by
// their vertical positions. It returns positive if a is lower, negative if a is
// higher, 0 if their order can't be determined solely by their vertical position.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate._compareVertically
func compareRectsVertically(a, b ui.Rect) int {
	// The rectangles overlap so defer to horizontal comparison.
	if (a.Top-b.Top < selectableVerticalComparingThreshold && a.Bottom-b.Bottom > -selectableVerticalComparingThreshold) ||
		(b.Top-a.Top < selectableVerticalComparingThreshold && b.Bottom-a.Bottom > -selectableVerticalComparingThreshold) {
		return 0
	}
	if math.Abs(a.Top-b.Top) > selectableVerticalComparingThreshold {
		if a.Top > b.Top {
			return 1
		}
		return -1
	}
	if a.Bottom > b.Bottom {
		return 1
	}
	return -1
}

// compareRectsHorizontally compares two rectangles in the screen order by
// their horizontal positions assuming one of the rectangles enclose the other
// rect vertically.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate._compareHorizontally
func compareRectsHorizontally(a, b ui.Rect) int {
	const tol = foundation.PrecisionErrorTolerance
	// a encloses b.
	if a.Left-b.Left < tol && a.Right-b.Right > -tol {
		return -1
	}
	// b encloses a.
	if b.Left-a.Left < tol && b.Right-a.Right > -tol {
		return 1
	}
	if math.Abs(a.Left-b.Left) > tol {
		if a.Left > b.Left {
			return 1
		}
		return -1
	}
	if a.Right > b.Right {
		return 1
	}
	return -1
}

func (m *MultiSelectableSelectionContainerDelegate) handleSelectableGeometryChange() {
	// Geometries of selectable children may change multiple times when handling
	// selection events. Ignore these updates since the selection geometry of
	// this delegate will be updated after handling the selection events.
	if m.isHandlingSelectionEvent {
		return
	}
	m.updateSelectionGeometry()
}

// GetSelectionGeometry gets the combined SelectionGeometry for child selectables.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.getSelectionGeometry
func (m *MultiSelectableSelectionContainerDelegate) GetSelectionGeometry() rendering.SelectionGeometry {
	if m.CurrentSelectionEndIndex == -1 || m.CurrentSelectionStartIndex == -1 || len(m.Selectables) == 0 {
		// There is no valid selection.
		return rendering.SelectionGeometry{Status: rendering.SelectionStatusNone, HasContent: len(m.Selectables) > 0}
	}
	if !m.extendSelectionInProgress {
		m.CurrentSelectionStartIndex = m.adjustSelectionIndexBasedOnSelectionGeometry(m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex)
		m.CurrentSelectionEndIndex = m.adjustSelectionIndexBasedOnSelectionGeometry(m.CurrentSelectionEndIndex, m.CurrentSelectionStartIndex)
	}
	// Need to find the non-null start selection point.
	startGeometry := m.Selectables[m.CurrentSelectionStartIndex].Value()
	forwardSelection := m.CurrentSelectionEndIndex >= m.CurrentSelectionStartIndex
	startIndexWalker := m.CurrentSelectionStartIndex
	for startIndexWalker != m.CurrentSelectionEndIndex && startGeometry.StartSelectionPoint == nil {
		if forwardSelection {
			startIndexWalker++
		} else {
			startIndexWalker--
		}
		startGeometry = m.Selectables[startIndexWalker].Value()
	}
	var startPoint *rendering.SelectionPoint
	if p := startGeometry.StartSelectionPoint; p != nil {
		start := m.GetTransformFrom(m.Selectables[startIndexWalker]).TransformPoint(p.LocalPosition)
		// It can be NaN if it is detached or off-screen.
		if start.IsFinite() {
			startPoint = &rendering.SelectionPoint{LocalPosition: start, LineHeight: p.LineHeight, HandleType: p.HandleType}
		}
	}
	// Need to find the non-null end selection point.
	endGeometry := m.Selectables[m.CurrentSelectionEndIndex].Value()
	endIndexWalker := m.CurrentSelectionEndIndex
	for endIndexWalker != m.CurrentSelectionStartIndex && endGeometry.EndSelectionPoint == nil {
		if forwardSelection {
			endIndexWalker--
		} else {
			endIndexWalker++
		}
		endGeometry = m.Selectables[endIndexWalker].Value()
	}
	var endPoint *rendering.SelectionPoint
	if p := endGeometry.EndSelectionPoint; p != nil {
		end := m.GetTransformFrom(m.Selectables[endIndexWalker]).TransformPoint(p.LocalPosition)
		if end.IsFinite() {
			endPoint = &rendering.SelectionPoint{LocalPosition: end, LineHeight: p.LineHeight, HandleType: p.HandleType}
		}
	}
	// Need to collect selection rects from selectables ranging from the
	// CurrentSelectionStartIndex to the CurrentSelectionEndIndex.
	var selectionRects []ui.Rect
	var drawableArea *ui.Rect
	if m.HasSize() {
		r := ui.NewRectFromLTWH(0, 0, m.ContainerSize().Width, m.ContainerSize().Height)
		drawableArea = &r
	}
	for index := m.CurrentSelectionStartIndex; index <= m.CurrentSelectionEndIndex; index++ {
		for _, selectionRect := range m.Selectables[index].Value().SelectionRects {
			localRect := m.GetTransformFrom(m.Selectables[index]).TransformRect(selectionRect)
			if drawableArea != nil {
				localRect = drawableArea.Intersect(localRect)
			}
			if localRect.IsFinite() && !localRect.IsEmpty() {
				selectionRects = append(selectionRects, localRect)
			}
		}
	}
	status := startGeometry.Status
	if !startGeometry.Equal(endGeometry) {
		status = rendering.SelectionStatusUncollapsed
	}
	return rendering.SelectionGeometry{
		StartSelectionPoint: startPoint, EndSelectionPoint: endPoint, SelectionRects: selectionRects, Status: status,
		// Would have at least one selectable child.
		HasContent: true,
	}
}

// adjustSelectionIndexBasedOnSelectionGeometry: the CurrentSelectionStartIndex
// or CurrentSelectionEndIndex may not be the current index that contains
// selection edges. This can happen if the selection edge is in between two
// selectables. One of the selectable will have its selection collapsed at the
// index 0 or contentLength depends on whether the selection is reversed or not.
// The current selection index can be point to either one. This method adjusts
// the index to point to selectable with valid selection.
func (m *MultiSelectableSelectionContainerDelegate) adjustSelectionIndexBasedOnSelectionGeometry(currentIndex, towardIndex int) int {
	forward := towardIndex > currentIndex
	for currentIndex != towardIndex && m.Selectables[currentIndex].Value().Status != rendering.SelectionStatusUncollapsed {
		if forward {
			currentIndex++
		} else {
			currentIndex--
		}
	}
	return currentIndex
}

// PushHandleLayers gives the layer links that the selection handles follow.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.pushHandleLayers
func (m *MultiSelectableSelectionContainerDelegate) PushHandleLayers(startHandle, endHandle *rendering.LayerLink) {
	if m.startHandleLayer == startHandle && m.endHandleLayer == endHandle {
		return
	}
	m.startHandleLayer = startHandle
	m.endHandleLayer = endHandle
	m.updateHandleLayersAndOwners()
}

// updateHandleLayersAndOwners pushes both handle layers to the selectables that
// contain selection edges. It needs to be called every time the selectables
// that contain the selection edges change, otherwise the handle may be painted
// in the wrong place.
func (m *MultiSelectableSelectionContainerDelegate) updateHandleLayersAndOwners() {
	effectiveStart, effectiveEnd := m.startHandleLayer, m.endHandleLayer
	if effectiveStart != nil || effectiveEnd != nil {
		var drawableArea *ui.Rect
		if m.HasSize() {
			r := ui.NewRectFromLTWH(0, 0, m.ContainerSize().Width, m.ContainerSize().Height).Inflate(selectionHandleDrawableAreaPadding)
			drawableArea = &r
		}
		v := m.Value()
		hideStart := v.StartSelectionPoint == nil || drawableArea == nil || !drawableArea.Contains(v.StartSelectionPoint.LocalPosition)
		hideEnd := v.EndSelectionPoint == nil || drawableArea == nil || !drawableArea.Contains(v.EndSelectionPoint.LocalPosition)
		if hideStart {
			effectiveStart = nil
		}
		if hideEnd {
			effectiveEnd = nil
		}
	}
	if m.CurrentSelectionStartIndex == -1 || m.CurrentSelectionEndIndex == -1 {
		// No valid selection.
		if m.startHandleLayerOwner != nil {
			m.startHandleLayerOwner.PushHandleLayers(nil, nil)
			m.startHandleLayerOwner = nil
		}
		if m.endHandleLayerOwner != nil {
			m.endHandleLayerOwner.PushHandleLayers(nil, nil)
			m.endHandleLayerOwner = nil
		}
		return
	}
	if m.Selectables[m.CurrentSelectionStartIndex] != m.startHandleLayerOwner && m.startHandleLayerOwner != nil {
		m.startHandleLayerOwner.PushHandleLayers(nil, nil)
	}
	if m.Selectables[m.CurrentSelectionEndIndex] != m.endHandleLayerOwner && m.endHandleLayerOwner != nil {
		m.endHandleLayerOwner.PushHandleLayers(nil, nil)
	}
	m.startHandleLayerOwner = m.Selectables[m.CurrentSelectionStartIndex]
	if m.CurrentSelectionStartIndex == m.CurrentSelectionEndIndex {
		// Selection edges is on the same selectable.
		m.endHandleLayerOwner = m.startHandleLayerOwner
		m.startHandleLayerOwner.PushHandleLayers(effectiveStart, effectiveEnd)
		return
	}
	m.startHandleLayerOwner.PushHandleLayers(effectiveStart, nil)
	m.endHandleLayerOwner = m.Selectables[m.CurrentSelectionEndIndex]
	m.endHandleLayerOwner.PushHandleLayers(nil, effectiveEnd)
}

// GetSelectedContent copies the selected contents of all selectables.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.getSelectedContent
func (m *MultiSelectableSelectionContainerDelegate) GetSelectedContent() *rendering.SelectedContent {
	var text []byte
	hasContent := false
	for _, s := range m.Selectables {
		if c := s.GetSelectedContent(); c != nil {
			text = append(text, c.PlainText...)
			hasContent = true
		}
	}
	if !hasContent {
		return nil
	}
	return &rendering.SelectedContent{PlainText: string(text)}
}

// ContentLength is the total length of the content under this delegate: the sum
// of the content length of each selectable it manages.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.contentLength
func (m *MultiSelectableSelectionContainerDelegate) ContentLength() int {
	sum := 0
	for _, s := range m.Selectables {
		sum += s.ContentLength()
	}
	return sum
}

type selectionInfo struct {
	contentLength int
	rng           *rendering.SelectedContentRange
}

// calculateLocalRange calculates a local SelectedContentRange based on the
// list of selections that are accumulated from the selectable children under
// this delegate. This calculation takes into account the accumulated content
// length before the active selection, and returns nil when either selection
// edge has not been set.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate._calculateLocalRange
func (m *MultiSelectableSelectionContainerDelegate) calculateLocalRange(selections []selectionInfo) *rendering.SelectedContentRange {
	if m.CurrentSelectionStartIndex == -1 || m.CurrentSelectionEndIndex == -1 {
		return nil
	}
	startOffset, endOffset := 0, 0
	foundStart := false
	forwardSelection := m.CurrentSelectionEndIndex >= m.CurrentSelectionStartIndex
	if m.CurrentSelectionEndIndex == m.CurrentSelectionStartIndex {
		// Determining selection direction is inaccurate if the indices are equal.
		// Use the range from the selectable within the selection as the source of
		// truth for selection direction.
		if r := m.Selectables[m.CurrentSelectionStartIndex].GetSelection(); r != nil {
			forwardSelection = r.EndOffset >= r.StartOffset
		}
	}
	result := func() *rendering.SelectedContentRange {
		if forwardSelection {
			return &rendering.SelectedContentRange{StartOffset: startOffset, EndOffset: endOffset}
		}
		return &rendering.SelectedContentRange{StartOffset: endOffset, EndOffset: startOffset}
	}
	for _, selection := range selections {
		if selection.rng == nil {
			if foundStart {
				return result()
			}
			startOffset += selection.contentLength
			endOffset = startOffset
			continue
		}
		selectionStartNormalized := min(selection.rng.StartOffset, selection.rng.EndOffset)
		selectionEndNormalized := max(selection.rng.StartOffset, selection.rng.EndOffset)
		if !foundStart {
			startOffset += selectionStartNormalized
			endOffset = startOffset + absInt(selectionEndNormalized-selectionStartNormalized)
			foundStart = true
		} else {
			endOffset += absInt(selectionEndNormalized - selectionStartNormalized)
		}
	}
	return result()
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// GetSelection returns a SelectedContentRange considering the range from each
// selectable child managed under this delegate. When nothing is selected or
// either selection edge has not been set, it returns nil.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.getSelection
func (m *MultiSelectableSelectionContainerDelegate) GetSelection() *rendering.SelectedContentRange {
	selections := make([]selectionInfo, 0, len(m.Selectables))
	for _, s := range m.Selectables {
		selections = append(selections, selectionInfo{contentLength: s.ContentLength(), rng: s.GetSelection()})
	}
	return m.calculateLocalRange(selections)
}

// flushInactiveSelections clears the selection on all selectables not in the
// range of CurrentSelectionStartIndex..CurrentSelectionEndIndex. If one of the
// edges does not exist, then it clears the selection in all selectables except
// the existing edge. If neither of the edges exist it returns immediately.
func (m *MultiSelectableSelectionContainerDelegate) flushInactiveSelections() {
	if m.CurrentSelectionStartIndex == -1 && m.CurrentSelectionEndIndex == -1 {
		return
	}
	if m.CurrentSelectionStartIndex == -1 || m.CurrentSelectionEndIndex == -1 {
		skip := m.CurrentSelectionStartIndex
		if skip == -1 {
			skip = m.CurrentSelectionEndIndex
		}
		m.clearSelectables(skip)
		return
	}
	skipStart := min(m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex)
	skipEnd := max(m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex)
	for index := range m.Selectables {
		if index >= skipStart && index <= skipEnd {
			continue
		}
		m.self.DispatchSelectionEventToChild(m.Selectables[index], rendering.ClearSelectionEvent{})
	}
}

// HandleSelectAll selects all contents of all selectables.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectAll
func (m *MultiSelectableSelectionContainerDelegate) HandleSelectAll(event rendering.SelectAllSelectionEvent) rendering.SelectionResult {
	for _, s := range m.Selectables {
		m.self.DispatchSelectionEventToChild(s, event)
	}
	m.CurrentSelectionStartIndex = 0
	m.CurrentSelectionEndIndex = len(m.Selectables) - 1
	return rendering.SelectionResultNone
}

// clearSelectables clears the selection in all selectables, skipping the one at
// the given index (-1 skips none).
func (m *MultiSelectableSelectionContainerDelegate) clearSelectables(skipIndex int) {
	for i := range m.Selectables {
		if i == skipIndex {
			continue
		}
		m.self.DispatchSelectionEventToChild(m.Selectables[i], rendering.ClearSelectionEvent{})
	}
}

func (m *MultiSelectableSelectionContainerDelegate) handleSelectBoundary(event rendering.SelectionEvent, effectiveGlobalPosition ui.Offset) rendering.SelectionResult {
	var lastResult foundation.Opt[rendering.SelectionResult]
	// For tracking the nearest index if no bounding box contains the position.
	minDistanceSquared := math.Inf(1)
	nearestIndex := 0
	for index := 0; index < len(m.Selectables); index++ {
		globalRectsContainPosition := false
		transform := m.Selectables[index].GetTransformTo(nil)
		for _, rect := range m.Selectables[index].BoundingBoxes() {
			globalRect := transform.TransformRect(rect)
			if globalRect.Contains(effectiveGlobalPosition) {
				globalRectsContainPosition = true
				break
			}
			dx := effectiveGlobalPosition.Dx - ui.ClampDouble(effectiveGlobalPosition.Dx, globalRect.Left, globalRect.Right)
			dy := effectiveGlobalPosition.Dy - ui.ClampDouble(effectiveGlobalPosition.Dy, globalRect.Top, globalRect.Bottom)
			if d := dx*dx + dy*dy; d < minDistanceSquared {
				minDistanceSquared = d
				nearestIndex = index
			}
		}
		if globalRectsContainPosition {
			existing := m.Selectables[index].Value()
			result := m.self.DispatchSelectionEventToChild(m.Selectables[index], event)
			lastResult = foundation.Some(result)
			if index == len(m.Selectables)-1 && result == rendering.SelectionResultNext {
				return rendering.SelectionResultNext
			}
			if result == rendering.SelectionResultNext {
				continue
			}
			if index == 0 && result == rendering.SelectionResultPrevious {
				return rendering.SelectionResultPrevious
			}
			if !m.Selectables[index].Value().Equal(existing) {
				// Geometry has changed as a result of select word, need to clear the
				// selection of other selectables to keep selection in sync.
				m.clearSelectables(index)
				m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex = index, index
			}
			return rendering.SelectionResultEnd
		}
		if r, ok := lastResult.Get(); ok && r == rendering.SelectionResultNext {
			m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex = index-1, index-1
			return rendering.SelectionResultEnd
		}
	}
	// No selectable's bounding box contained the position. Clamp to the nearest
	// selectable so that the boundary selection event always produces a valid
	// selection.
	if len(m.Selectables) > 0 {
		existing := m.Selectables[nearestIndex].Value()
		m.self.DispatchSelectionEventToChild(m.Selectables[nearestIndex], event)
		if !m.Selectables[nearestIndex].Value().Equal(existing) {
			m.clearSelectables(nearestIndex)
			m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex = nearestIndex, nearestIndex
		}
	}
	return rendering.SelectionResultEnd
}

// HandleSelectWord selects a word in a selectable at the location of the event.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectWord
func (m *MultiSelectableSelectionContainerDelegate) HandleSelectWord(event rendering.SelectWordSelectionEvent) rendering.SelectionResult {
	return m.handleSelectBoundary(event, event.GlobalPosition)
}

// HandleSelectParagraph selects a paragraph in a selectable at the location of
// the event.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectParagraph
func (m *MultiSelectableSelectionContainerDelegate) HandleSelectParagraph(event rendering.SelectParagraphSelectionEvent) rendering.SelectionResult {
	return m.handleSelectBoundary(event, event.GlobalPosition)
}

// HandleClearSelection removes the selection of all selectables this delegate
// manages.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleClearSelection
func (m *MultiSelectableSelectionContainerDelegate) HandleClearSelection(event rendering.ClearSelectionEvent) rendering.SelectionResult {
	for _, s := range m.Selectables {
		m.self.DispatchSelectionEventToChild(s, event)
	}
	m.CurrentSelectionEndIndex = -1
	m.CurrentSelectionStartIndex = -1
	return rendering.SelectionResultNone
}

// HandleGranularlyExtendSelection extends the current selection in a certain
// granularity.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleGranularlyExtendSelection
func (m *MultiSelectableSelectionContainerDelegate) HandleGranularlyExtendSelection(event rendering.GranularlyExtendSelectionEvent) rendering.SelectionResult {
	if m.CurrentSelectionStartIndex == -1 {
		if event.Forward {
			m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex = 0, 0
		} else {
			m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex = len(m.Selectables)-1, len(m.Selectables)-1
		}
	}
	targetIndex := m.CurrentSelectionStartIndex
	if event.IsEnd {
		targetIndex = m.CurrentSelectionEndIndex
	}
	result := m.self.DispatchSelectionEventToChild(m.Selectables[targetIndex], event)
	if event.Forward {
		for targetIndex < len(m.Selectables)-1 && result == rendering.SelectionResultNext {
			targetIndex++
			result = m.self.DispatchSelectionEventToChild(m.Selectables[targetIndex], event)
		}
	} else {
		for targetIndex > 0 && result == rendering.SelectionResultPrevious {
			targetIndex--
			result = m.self.DispatchSelectionEventToChild(m.Selectables[targetIndex], event)
		}
	}
	if event.IsEnd {
		m.CurrentSelectionEndIndex = targetIndex
	} else {
		m.CurrentSelectionStartIndex = targetIndex
	}
	return result
}

// HandleDirectionallyExtendSelection extends the current selection in a
// direction.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleDirectionallyExtendSelection
func (m *MultiSelectableSelectionContainerDelegate) HandleDirectionallyExtendSelection(event rendering.DirectionallyExtendSelectionEvent) rendering.SelectionResult {
	if m.CurrentSelectionStartIndex == -1 {
		switch event.Direction {
		case rendering.SelectionExtendDirectionPreviousLine, rendering.SelectionExtendDirectionBackward:
			m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex = len(m.Selectables)-1, len(m.Selectables)-1
		default:
			m.CurrentSelectionStartIndex, m.CurrentSelectionEndIndex = 0, 0
		}
	}
	targetIndex := m.CurrentSelectionStartIndex
	if event.IsEnd {
		targetIndex = m.CurrentSelectionEndIndex
	}
	result := m.self.DispatchSelectionEventToChild(m.Selectables[targetIndex], event)
	switch event.Direction {
	case rendering.SelectionExtendDirectionPreviousLine:
		if result == rendering.SelectionResultPrevious && targetIndex > 0 {
			targetIndex--
			result = m.self.DispatchSelectionEventToChild(m.Selectables[targetIndex],
				event.CopyWith(rendering.DirectionallyExtendSelectionEventCopy{Direction: rendering.SelectionExtendDirectionBackward}))
		}
	case rendering.SelectionExtendDirectionNextLine:
		if result == rendering.SelectionResultNext && targetIndex < len(m.Selectables)-1 {
			targetIndex++
			result = m.self.DispatchSelectionEventToChild(m.Selectables[targetIndex],
				event.CopyWith(rendering.DirectionallyExtendSelectionEventCopy{Direction: rendering.SelectionExtendDirectionForward}))
		}
	}
	if event.IsEnd {
		m.CurrentSelectionEndIndex = targetIndex
	} else {
		m.CurrentSelectionStartIndex = targetIndex
	}
	return result
}

// HandleSelectionEdgeUpdate updates the selection edges.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.handleSelectionEdgeUpdate
func (m *MultiSelectableSelectionContainerDelegate) HandleSelectionEdgeUpdate(event rendering.SelectionEdgeUpdateEvent) rendering.SelectionResult {
	if event.Type() == rendering.SelectionEventTypeEndEdgeUpdate {
		if m.CurrentSelectionEndIndex == -1 {
			return m.initSelection(event, true)
		}
		return m.adjustSelection(event, true)
	}
	if m.CurrentSelectionStartIndex == -1 {
		return m.initSelection(event, false)
	}
	return m.adjustSelection(event, false)
}

// DispatchSelectionEvent handles the event by dispatching it to the selectables.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.dispatchSelectionEvent
func (m *MultiSelectableSelectionContainerDelegate) DispatchSelectionEvent(event rendering.SelectionEvent) rendering.SelectionResult {
	_, isClear := event.(rendering.ClearSelectionEvent)
	selectionWillBeInProgress := !isClear
	if !m.selectionInProgress && selectionWillBeInProgress {
		// Sort the selectable every time a selection start.
		slices.SortStableFunc(m.Selectables, m.self.CompareOrder)
	}
	m.selectionInProgress = selectionWillBeInProgress
	m.isHandlingSelectionEvent = true
	var result rendering.SelectionResult
	switch e := event.(type) {
	case rendering.SelectionEdgeUpdateEvent:
		m.extendSelectionInProgress = false
		result = m.self.HandleSelectionEdgeUpdate(e)
	case rendering.ClearSelectionEvent:
		m.extendSelectionInProgress = false
		result = m.self.HandleClearSelection(e)
	case rendering.SelectAllSelectionEvent:
		m.extendSelectionInProgress = false
		result = m.self.HandleSelectAll(e)
	case rendering.SelectWordSelectionEvent:
		m.extendSelectionInProgress = false
		result = m.self.HandleSelectWord(e)
	case rendering.SelectParagraphSelectionEvent:
		m.extendSelectionInProgress = false
		result = m.self.HandleSelectParagraph(e)
	case rendering.GranularlyExtendSelectionEvent:
		m.extendSelectionInProgress = true
		result = m.self.HandleGranularlyExtendSelection(e)
	case rendering.DirectionallyExtendSelectionEvent:
		m.extendSelectionInProgress = true
		result = m.self.HandleDirectionallyExtendSelection(e)
	}
	m.isHandlingSelectionEvent = false
	m.updateSelectionGeometry()
	return result
}

// Dispose releases the selectables and the listeners.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.dispose
func (m *MultiSelectableSelectionContainerDelegate) Dispose() {
	for s, h := range m.geometryHandles {
		s.RemoveListener(h)
	}
	m.geometryHandles = map[rendering.Selectable]foundation.ListenerHandle{}
	m.Selectables = nil
	m.scheduledSelectableUpdate = false
	m.ChangeNotifier.Dispose()
}

// DispatchSelectionEventToChild dispatches a selection event to a specific
// selectable. Override it if the delegate needs to generate additional events or
// treatments prior to sending the event.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate.dispatchSelectionEventToChild
func (m *MultiSelectableSelectionContainerDelegate) DispatchSelectionEventToChild(selectable rendering.Selectable, event rendering.SelectionEvent) rendering.SelectionResult {
	return selectable.DispatchSelectionEvent(event)
}

// initSelection initializes the selection of the selectable children. The goal
// is to find the selectable child that contains the selection edge. It returns
// end if the selection edge ends on any of the children. Otherwise, it returns
// previous if the selection does not reach any of its children, and next if the
// selection reaches the end of its children.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate._initSelection
func (m *MultiSelectableSelectionContainerDelegate) initSelection(event rendering.SelectionEdgeUpdateEvent, isEnd bool) rendering.SelectionResult {
	newIndex := -1
	hasFoundEdgeIndex := false
	var result foundation.Opt[rendering.SelectionResult]
	forward := foundation.None[bool]()
	// If we already have an opposite edge initialized, start our sweep from
	// there to ensure all items between the two edges are properly visited.
	oppositeEdgeIndex := m.CurrentSelectionEndIndex
	if isEnd {
		oppositeEdgeIndex = m.CurrentSelectionStartIndex
	}
	index := max(oppositeEdgeIndex, 0)
	for index >= 0 && index < len(m.Selectables) {
		child := m.Selectables[index]
		childResult := m.self.DispatchSelectionEventToChild(child, event)
		switch childResult {
		case rendering.SelectionResultNext:
			if f, ok := forward.Get(); ok && !f {
				hasFoundEdgeIndex = true
				result = foundation.Some(rendering.SelectionResultEnd)
			} else {
				forward = foundation.Some(true)
				newIndex = index
			}
		case rendering.SelectionResultNone:
			newIndex = index
		case rendering.SelectionResultEnd:
			newIndex = index
			result = foundation.Some(rendering.SelectionResultEnd)
			hasFoundEdgeIndex = true
		case rendering.SelectionResultPrevious:
			if index == 0 {
				hasFoundEdgeIndex = true
				newIndex = 0
				result = foundation.Some(rendering.SelectionResultPrevious)
			} else if forward.OrElse(false) {
				hasFoundEdgeIndex = true
				result = foundation.Some(rendering.SelectionResultEnd)
			} else {
				forward = foundation.Some(false)
				newIndex = index
			}
		case rendering.SelectionResultPending:
			newIndex = index
			result = foundation.Some(rendering.SelectionResultPending)
			hasFoundEdgeIndex = true
		}
		if hasFoundEdgeIndex {
			break
		}
		if forward.OrElse(true) {
			index++
		} else {
			index--
		}
	}
	if newIndex == -1 {
		return rendering.SelectionResultNone
	}
	if isEnd {
		m.CurrentSelectionEndIndex = newIndex
	} else {
		m.CurrentSelectionStartIndex = newIndex
	}
	m.flushInactiveSelections()
	// The result can only be none if the loop went through the entire list
	// without any of the selection returned end or previous. In this case, the
	// caller of this method needs to find the next selectable in their list.
	return result.OrElse(rendering.SelectionResultNext)
}

// adjustSelection adjusts the selection based on the drag selection update
// event if there is already a selectable child that contains the selection
// edge. It starts by sending the selection event to the current selectable that
// contains the selection edge, and finds forward or backward if that selectable
// no longer contains the selection edge.
//
// Flutter: widgets/selectable_region.dart MultiSelectableSelectionContainerDelegate._adjustSelection
func (m *MultiSelectableSelectionContainerDelegate) adjustSelection(event rendering.SelectionEdgeUpdateEvent, isEnd bool) rendering.SelectionResult {
	var finalResult foundation.Opt[rendering.SelectionResult]
	// Determines if the edge being adjusted is within the current viewport. If
	// so, the search for the new selection edge position begins at the
	// current index. If not, it is located starting from the opposite end. If
	// neither edge is in the current viewport, the search begins at 0. This can
	// happen when there is a scrollable child and the edge being adjusted has
	// been scrolled out of view.
	currentEdgeWithin := m.selectionGeometry.StartSelectionPoint != nil
	oppositeEdgeWithin := m.selectionGeometry.EndSelectionPoint != nil
	if isEnd {
		currentEdgeWithin, oppositeEdgeWithin = oppositeEdgeWithin, currentEdgeWithin
	}
	var newIndex int
	switch {
	case isEnd && currentEdgeWithin:
		newIndex = m.CurrentSelectionEndIndex
	case isEnd && oppositeEdgeWithin:
		newIndex = m.CurrentSelectionStartIndex
	case isEnd:
		newIndex = 0
	case currentEdgeWithin:
		newIndex = m.CurrentSelectionStartIndex
	case oppositeEdgeWithin:
		newIndex = m.CurrentSelectionEndIndex
	default:
		newIndex = 0
	}
	forward := foundation.None[bool]()
	// This loop sends the selection event to one of the current edge's
	// selectable, the opposite edge's selectable or index 0, to determine the
	// direction of the search. If the result is next, this loop looks backward.
	// Otherwise, it looks forward.
	for newIndex < len(m.Selectables) && newIndex >= 0 && finalResult.IsNone() {
		current := m.self.DispatchSelectionEventToChild(m.Selectables[newIndex], event)
		switch current {
		case rendering.SelectionResultEnd, rendering.SelectionResultPending, rendering.SelectionResultNone:
			finalResult = foundation.Some(current)
		case rendering.SelectionResultNext:
			if f, ok := forward.Get(); ok && !f {
				newIndex++
				finalResult = foundation.Some(rendering.SelectionResultEnd)
			} else if newIndex == len(m.Selectables)-1 {
				finalResult = foundation.Some(current)
			} else {
				forward = foundation.Some(true)
				newIndex++
			}
		case rendering.SelectionResultPrevious:
			if forward.OrElse(false) {
				newIndex--
				finalResult = foundation.Some(rendering.SelectionResultEnd)
			} else if newIndex == 0 {
				finalResult = foundation.Some(current)
			} else {
				forward = foundation.Some(false)
				newIndex--
			}
		}
	}
	if isEnd {
		m.CurrentSelectionEndIndex = newIndex
	} else {
		m.CurrentSelectionStartIndex = newIndex
	}
	m.flushInactiveSelections()
	return finalResult.OrElse(rendering.SelectionResultNone)
}
