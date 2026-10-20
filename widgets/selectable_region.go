package widgets

import (
	"context"
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// SelectableRegionContextMenuBuilder builds a context menu for the given
// SelectableRegionState.
//
// Flutter: widgets/selectable_region.dart SelectableRegionContextMenuBuilder
type SelectableRegionContextMenuBuilder func(bc BuildContext, selectableRegionState *SelectableRegionState) Widget

// SelectableRegionOpts are the optional parameters of NewSelectableRegion.
//
// Flutter: widgets/selectable_region.dart SelectableRegion.new
type SelectableRegionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ContextMenuBuilder builds the context menu.
	ContextMenuBuilder SelectableRegionContextMenuBuilder
	// FocusNode is the focus node of the region; the default is one that the
	// state creates.
	FocusNode *FocusNode
	// OnSelectionChanged is called when the selected content changes.
	OnSelectionChanged func(content *rendering.SelectedContent)
}

// SelectableRegion is a widget that introduces an area for user selections.
//
// Flutter widgets are not selectable by default. Wrapping a widget subtree with
// a SelectableRegion enables selection for widgets in the subtree that
// participate in selection. For example, Text widgets automatically look for
// selectable regions to enable selection, while widgets such as RichText must
// be configured with a SelectionRegistrar and a selection color. The wrapped
// subtree can be selected by users using mouse or touch gestures, e.g. users
// can select widgets by holding the mouse left-click and dragging across
// widgets, or they can use long press gestures to select words on touch devices.
//
// Every Selectable under the SelectableRegion can be selected. They form a
// selection tree structure to handle the selection. The SelectableRegion is a
// wrapper over SelectionContainer. It listens to user gestures and sends
// corresponding SelectionEvents to the SelectionContainer it creates. The
// SelectionContainer created by SelectableRegion is the root node of a selection
// tree. Each non-leaf node in the tree is a SelectionContainer, and the leaf
// node is a leaf widget whose render object implements Selectable.
//
// Deviation: the magnifier, the web context menu, the text processing actions
// and the share button are not ported (SPEC §12.10).
//
// Flutter: widgets/selectable_region.dart SelectableRegion
type SelectableRegion struct {
	StatefulWidget
	selectionControls TextSelectionControls
	child             Widget
	o                 SelectableRegionOpts
}

// NewSelectableRegion creates a new selectable region. The selectionControls
// are used for building the selection handles for mobile devices.
//
// Flutter: widgets/selectable_region.dart SelectableRegion.new
func NewSelectableRegion(selectionControls TextSelectionControls, child Widget, opts ...SelectableRegionOpts) *SelectableRegion {
	o := firstOpts(opts)
	return &SelectableRegion{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), selectionControls: selectionControls, child: child, o: o}
}

// Child is the widget this selection area applies to.
//
// Flutter: widgets/selectable_region.dart SelectableRegion.child
func (w *SelectableRegion) Child() Widget { return w.child }

// SelectionControls build the selection handles for mobile devices.
//
// Flutter: widgets/selectable_region.dart SelectableRegion.selectionControls
func (w *SelectableRegion) SelectionControls() TextSelectionControls { return w.selectionControls }

// SelectableButtonItemsCallbacks are the callbacks of the buttons that
// GetSelectableButtonItems returns; OnShare may be nil.
//
// Flutter: widgets/selectable_region.dart SelectableRegion.getSelectableButtonItems
type SelectableButtonItemsCallbacks struct {
	OnCopy, OnSelectAll, OnShare foundation.VoidCallback
}

// GetSelectableButtonItems returns the items representing the buttons in this
// platform's default selection menu. For example, SelectableRegion uses this to
// generate the default buttons for its context menu.
//
// Flutter: widgets/selectable_region.dart SelectableRegion.getSelectableButtonItems
func GetSelectableButtonItems(selectionGeometry rendering.SelectionGeometry, c SelectableButtonItemsCallbacks) []ContextMenuButtonItem {
	canCopy := selectionGeometry.Status == rendering.SelectionStatusUncollapsed
	canSelectAll := selectionGeometry.HasContent
	platformCanShare := foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid && selectionGeometry.Status == rendering.SelectionStatusUncollapsed
	canShare := c.OnShare != nil && platformCanShare
	// On Android, the share button is before the select all button.
	showShareBeforeSelectAll := foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid
	var items []ContextMenuButtonItem
	if canCopy {
		items = append(items, ContextMenuButtonItem{OnPressed: c.OnCopy, Type: ContextMenuButtonTypeCopy})
	}
	if canShare && showShareBeforeSelectAll {
		items = append(items, ContextMenuButtonItem{OnPressed: c.OnShare, Type: ContextMenuButtonTypeShare})
	}
	if canSelectAll {
		items = append(items, ContextMenuButtonItem{OnPressed: c.OnSelectAll, Type: ContextMenuButtonTypeSelectAll})
	}
	if canShare && !showShareBeforeSelectAll {
		items = append(items, ContextMenuButtonItem{OnPressed: c.OnShare, Type: ContextMenuButtonTypeShare})
	}
	return items
}

// CreateState creates the state.
//
// Flutter: widgets/selectable_region.dart SelectableRegion.createState
func (w *SelectableRegion) CreateState() State {
	s := &SelectableRegionState{
		startHandleLink:         rendering.NewLayerLink(),
		endHandleLink:           rendering.NewLayerLink(),
		toolbarLink:             rendering.NewLayerLink(),
		selectionDelegate:       NewStaticSelectionContainerDelegate(),
		selectionStatusNotifier: &selectionStatusNotifier{value: SelectableRegionSelectionStatusFinalized},
		gestureRecognizers:      map[reflect.Type]GestureRecognizerFactory{},
	}
	return s
}

// SelectableRegionSelectionStatus is the status of the selection under a
// SelectableRegion.
//
// This value under a SelectableRegion is updated frequently during selection
// gestures such as clicks and taps to select and keyboard shortcuts.
//
// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatus
type SelectableRegionSelectionStatus int

const (
	// SelectableRegionSelectionStatusUnspecified is the zero value; the
	// notifier never has it.
	SelectableRegionSelectionStatusUnspecified SelectableRegionSelectionStatus = iota
	// SelectableRegionSelectionStatusChanging indicates that the selection under
	// a SelectableRegion is changing: it is being updated by the user through
	// selection gestures and keyboard shortcuts.
	//
	// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatus.changing
	SelectableRegionSelectionStatusChanging
	// SelectableRegionSelectionStatusFinalized indicates that the selection
	// under a SelectableRegion is finalized: it is no longer being updated by
	// the user.
	//
	// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatus.finalized
	SelectableRegionSelectionStatusFinalized
)

// Name is the name of the status.
//
// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatus.name
func (s SelectableRegionSelectionStatus) Name() string {
	switch s {
	case SelectableRegionSelectionStatusChanging:
		return "changing"
	case SelectableRegionSelectionStatusFinalized:
		return "finalized"
	}
	return "unspecified"
}

// String is "SelectableRegionSelectionStatus." and the name.
//
// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatus.toString
func (s SelectableRegionSelectionStatus) String() string {
	return "SelectableRegionSelectionStatus." + s.Name()
}

// selectionStatusNotifier notifies its listeners when the SelectableRegion that
// created it is changing or finalizes its selection.
//
// Flutter: widgets/selectable_region.dart _SelectableRegionSelectionStatusNotifier
type selectionStatusNotifier struct {
	foundation.ChangeNotifier
	value SelectableRegionSelectionStatus
}

func (n *selectionStatusNotifier) Value() SelectableRegionSelectionStatus { return n.value }

// set sets the status, and notifies the listeners even if it did not change.
func (n *selectionStatusNotifier) set(status SelectableRegionSelectionStatus) {
	if foundation.DebugMode && status != SelectableRegionSelectionStatusChanging &&
		(status != SelectableRegionSelectionStatusFinalized || n.value != SelectableRegionSelectionStatusChanging) {
		panic(errors.New("SelectableRegionAlreadyFinalized", "attempting to finalize the selection when it is already finalized"))
	}
	n.value = status
	n.NotifyListeners()
}

// SelectableRegionSelectionStatusScope notifies its listeners when the
// selection under a SelectableRegion or SelectionArea is being changed or
// finalized. Its listeners will be called even when the value of the status does
// not change.
//
// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatusScope
type SelectableRegionSelectionStatusScope struct {
	InheritedWidget
	selectionStatusNotifier foundation.ValueListenable[SelectableRegionSelectionStatus]
}

// SelectableRegionSelectionStatusScopeMaybeOf is the notifier of the status of
// the closest SelectableRegion above the context, or nil if there is none. It
// creates a dependency on the closest scope.
//
// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatusScope.maybeOf
func SelectableRegionSelectionStatusScopeMaybeOf(bc BuildContext) foundation.ValueListenable[SelectableRegionSelectionStatus] {
	if scope, ok := DependOn[*SelectableRegionSelectionStatusScope](bc); ok {
		return scope.selectionStatusNotifier
	}
	return nil
}

// UpdateShouldNotify is whether the notifier is another one.
//
// Flutter: widgets/selectable_region.dart SelectableRegionSelectionStatusScope.updateShouldNotify
func (w *SelectableRegionSelectionStatusScope) UpdateShouldNotify(old Widget) bool {
	return w.selectionStatusNotifier != old.(*SelectableRegionSelectionStatusScope).selectionStatusNotifier
}

// selectableRegionLongPressSelectionDevices are the devices that long press to
// select.
//
// Flutter: widgets/selectable_region.dart _kLongPressSelectionDevices
var selectableRegionLongPressSelectionDevices = []ui.PointerDeviceKind{
	ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus,
}

// SelectableRegionState is the state of a SelectableRegion. It is the
// registrar of the selection container it creates, and handles the gestures
// and the actions of the region.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState
type SelectableRegionState struct {
	StateOf[*SelectableRegion]

	actions            ActionMap
	gestureRecognizers map[reflect.Type]GestureRecognizerFactory
	selectionOverlay   *SelectionOverlay
	startHandleLink    *rendering.LayerLink
	endHandleLink      *rendering.LayerLink
	toolbarLink        *rendering.LayerLink
	selectionDelegate  *StaticSelectionContainerDelegate
	// selectable is the only selectable, which is the SelectionContainer.
	selectable       rendering.Selectable
	selectableHandle foundation.ListenerHandle

	lastOrientation     foundation.Opt[Orientation]
	lastSelectedContent *rendering.SelectedContent

	localFocusNode *FocusNode
	focusHandle    foundation.ListenerHandle
	focusListening *FocusNode

	selectionStatusNotifier *selectionStatusNotifier

	// isShiftPressed is whether the Shift key was pressed when the most recent
	// pointer down event was tracked by the tap and drag recognizer.
	isShiftPressed bool
	// lastSecondaryTapDownPosition is the position of the most recent secondary
	// tap down event on this region.
	lastSecondaryTapDownPosition foundation.Opt[ui.Offset]
	// lastPointerDeviceKind is the device kind for the pointer of the most
	// recent tap down event on this region.
	lastPointerDeviceKind foundation.Opt[ui.PointerDeviceKind]

	selectionEndPosition              foundation.Opt[ui.Offset]
	scheduledSelectionEndEdgeUpdate   bool
	selectionStartPosition            foundation.Opt[ui.Offset]
	scheduledSelectionStartEdgeUpdate bool
	selectionStartHandleDragPosition  ui.Offset
	selectionEndHandleDragPosition    ui.Offset
	adjustingSelectionEnd             foundation.Opt[bool]
	directionalHorizontalBaseline     foundation.Opt[float64]
}

func (s *SelectableRegionState) binding() *WidgetsBinding { return WidgetsBindingOf(s.Context()) }

func (s *SelectableRegionState) hasSelectionOverlayGeometry() bool {
	v := s.selectionDelegate.Value()
	return v.StartSelectionPoint != nil || v.EndSelectionPoint != nil
}

// SelectionOverlay is the overlay that is currently visible on the screen, or
// nil.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.selectionOverlay
func (s *SelectableRegionState) SelectionOverlay() *SelectionOverlay { return s.selectionOverlay }

func (s *SelectableRegionState) focusNode() *FocusNode {
	if f := s.Widget().o.FocusNode; f != nil {
		return f
	}
	if s.localFocusNode == nil {
		s.localFocusNode = NewFocusNode(FocusNodeOpts{DebugLabel: "SelectableRegion"})
	}
	return s.localFocusNode
}

func (s *SelectableRegionState) listenToFocus() {
	n := s.focusNode()
	s.focusListening = n
	s.focusHandle = n.AddListener(s.handleFocusChanged)
}

func (s *SelectableRegionState) stopListeningToFocus() {
	if s.focusListening != nil {
		s.focusListening.RemoveListener(s.focusHandle)
		s.focusListening = nil
	}
}

// InitState creates the actions and the gesture recognizers.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.initState
func (s *SelectableRegionState) InitState() {
	s.actions = s.buildActions()
	s.listenToFocus()
	s.initMouseGestureRecognizer()
	s.initTouchGestureRecognizer()
	// Right clicks.
	s.gestureRecognizers[reflect.TypeFor[*gestures.TapGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.TapGestureRecognizer {
			return gestures.NewTapGestureRecognizer(s.binding().Gestures(), gestures.TapGestureRecognizerOpts{DebugOwner: s})
		},
		func(r *gestures.TapGestureRecognizer) { r.OnSecondaryTapDown = s.handleRightClickDown })
}

// DidChangeDependencies hides the toolbar on mobile when the orientation changes.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.didChangeDependencies
func (s *SelectableRegionState) DidChangeDependencies() {
	p := foundation.DefaultTargetPlatform()
	if p != foundation.TargetPlatformAndroid && p != foundation.TargetPlatformIOS {
		return
	}
	orientation := MediaQueryOrientationOf(s.Context())
	if s.lastOrientation.IsNone() {
		s.lastOrientation = foundation.Some(orientation)
		return
	}
	if orientation != s.lastOrientation.Value() {
		s.lastOrientation = foundation.Some(orientation)
		s.HideToolbar(p == foundation.TargetPlatformAndroid)
	}
}

// DidUpdateWidget moves the listener to the new focus node.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.didUpdateWidget
func (s *SelectableRegionState) DidUpdateWidget(old *SelectableRegion) {
	w := s.Widget()
	if w.o.FocusNode == old.o.FocusNode {
		return
	}
	s.stopListeningToFocus()
	if old.o.FocusNode == nil && w.o.FocusNode != nil && s.localFocusNode != nil {
		s.localFocusNode.Dispose()
		s.localFocusNode = nil
	}
	s.listenToFocus()
	if old.o.FocusNode == nil || s.focusNode().HasFocus() != old.o.FocusNode.HasFocus() {
		s.handleFocusChanged()
	}
}

func (s *SelectableRegionState) makeOverridable(a Action) Action {
	return NewOverridableAction(a, s.Context())
}

func (s *SelectableRegionState) handleFocusChanged() {
	if s.focusNode().HasFocus() {
		return
	}
	if s.binding().Scheduler().LifecycleState() == ui.AppLifecycleStateResumed {
		// We should only clear the selection when this SelectableRegion loses
		// focus while the application is currently running. It is possible that
		// the application is not currently running, for example on desktop
		// platforms, clicking on a different window switches the focus to the new
		// window causing the Flutter application to go inactive. In this case we
		// want to retain the selection so it remains when we return to the
		// Flutter application.
		s.ClearSelection()
		s.selectionStatusNotifier.set(SelectableRegionSelectionStatusChanging)
		s.finalizeSelectableRegionStatus()
	}
}

func (s *SelectableRegionState) updateSelectionStatus() {
	if s.hasSelectionOverlayGeometry() {
		s.updateSelectionOverlay()
	} else if s.selectionOverlay != nil {
		s.selectionOverlay.Dispose()
		s.selectionOverlay = nil
	}
}

func isPrecisePointerDevice(kind ui.PointerDeviceKind) bool { return kind == ui.PointerDeviceKindMouse }

func (s *SelectableRegionState) finalizeSelectableRegionStatus() {
	if s.selectionStatusNotifier.value != SelectableRegionSelectionStatusChanging {
		// Don't finalize the selection again if it is not currently changing.
		return
	}
	s.selectionStatusNotifier.set(SelectableRegionSelectionStatusFinalized)
	s.setPrimarySelection()
}

// setPrimarySelection puts the selected content on the primary selection, so
// that a middle click pastes it on Linux desktops; the other platforms have none.
//
// Flutter: no counterpart
func (s *SelectableRegionState) setPrimarySelection() {
	if s.selectable == nil || foundation.DefaultTargetPlatform() != foundation.TargetPlatformLinux {
		return
	}
	content := s.selectable.GetSelectedContent()
	if content == nil || content.PlainText == "" {
		return
	}
	// A platform without a primary selection fails the call; there is nothing to
	// do about it.
	s.binding().Clipboard().SetPrimaryData(s.Lifetime(), services.NewClipboardData(content.PlainText), nil)
}

func (s *SelectableRegionState) setChanging() {
	s.selectionStatusNotifier.set(SelectableRegionSelectionStatusChanging)
}

// effectiveConsecutiveTapCount converts the consecutive tap count of a tap and
// drag details, which can grow to be infinitely large, to a value between 1 and
// the supported max consecutive tap count. The value that the raw count is
// converted to is based on the default observed behavior on the native platforms.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._getEffectiveConsecutiveTapCount
func (s *SelectableRegionState) effectiveConsecutiveTapCount(rawCount int) int {
	maxConsecutiveTap := 3
	wrap := func() int {
		// From observation, these platforms reset their tap count to 0 when the
		// number of consecutive taps exceeds the max consecutive tap supported.
		// For example on Debian Linux with GTK, when going past a triple click, on
		// the fourth click the selection is moved to the precise click position,
		// on the fifth click the word at the position is selected, and on the
		// sixth click the paragraph at the position is selected.
		if rawCount <= maxConsecutiveTap {
			return rawCount
		}
		if rawCount%maxConsecutiveTap == 0 {
			return maxConsecutiveTap
		}
		return rawCount % maxConsecutiveTap
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		if k, ok := s.lastPointerDeviceKind.Get(); ok && k != ui.PointerDeviceKindMouse {
			// When the pointer device kind is not precise like a mouse, native
			// Android resets the tap count at 2. For example, this is so the
			// selection can collapse on the third tap.
			maxConsecutiveTap = 2
		}
		return wrap()
	case foundation.TargetPlatformLinux:
		return wrap()
	}
	// From observation, these platforms hold their tap count at the max
	// consecutive tap supported. For example on macOS, when going past a triple
	// click, the selection should be retained at the paragraph that was first
	// selected on triple click.
	return min(rawCount, maxConsecutiveTap)
}

func (s *SelectableRegionState) initMouseGestureRecognizer() {
	s.gestureRecognizers[reflect.TypeFor[*gestures.TapAndPanGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.TapAndPanGestureRecognizer {
			return gestures.NewTapAndPanGestureRecognizer(s.binding().Gestures(),
				gestures.TapAndDragGestureRecognizerOpts{DebugOwner: s, SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindMouse}})
		},
		func(r *gestures.TapAndPanGestureRecognizer) {
			r.OnTapTrackStart = s.onTapTrackStart
			r.OnTapTrackReset = s.onTapTrackReset
			r.OnTapDown = s.startNewMouseSelectionGesture
			r.OnTapUp = s.handleMouseTapUp
			r.OnDragStart = s.handleMouseDragStart
			r.OnDragUpdate = s.handleMouseDragUpdate
			r.OnDragEnd = s.handleMouseDragEnd
			r.OnCancel = s.ClearSelection
			r.DragStartBehavior = gestures.DragStartBehaviorDown
		})
}

func (s *SelectableRegionState) onTapTrackStart() { s.isShiftPressed = hasShift(s.binding()) }

func (s *SelectableRegionState) onTapTrackReset() { s.isShiftPressed = false }

func (s *SelectableRegionState) initTouchGestureRecognizer() {
	// A tap and horizontal drag recognizer is used on non-precise pointer
	// devices like touch so the gestures do not conflict with ancestor
	// Scrollable gestures in common scenarios like a vertically scrolling list.
	nonMouse := []ui.PointerDeviceKind{ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus,
		ui.PointerDeviceKindTrackpad, ui.PointerDeviceKindUnknown}
	s.gestureRecognizers[reflect.TypeFor[*gestures.TapAndHorizontalDragGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.TapAndHorizontalDragGestureRecognizer {
			return gestures.NewTapAndHorizontalDragGestureRecognizer(s.binding().Gestures(),
				gestures.TapAndDragGestureRecognizerOpts{DebugOwner: s, SupportedDevices: nonMouse})
		},
		func(r *gestures.TapAndHorizontalDragGestureRecognizer) {
			// iOS does not provide a device specific touch slop unlike Android
			// (~8.0), so the touch slop for a Scrollable always default to 18.0.
			// When the region is the child of a horizontal scrollable that means it
			// will always win the gesture arena when competing with the ancestor
			// scrollable because they both have the same touch slop threshold and
			// the child receives the event first. To avoid this conflict and ensure
			// a smooth scrolling experience, on iOS the recognizer will wait for all
			// other gestures to lose before declaring victory.
			r.EagerVictoryOnDrag = foundation.DefaultTargetPlatform() != foundation.TargetPlatformIOS
			r.OnTapDown = s.startNewMouseSelectionGesture
			r.OnTapUp = s.handleMouseTapUp
			r.OnDragStart = s.handleMouseDragStart
			r.OnDragUpdate = s.handleMouseDragUpdate
			r.OnDragEnd = s.handleMouseDragEnd
			r.OnCancel = s.ClearSelection
			r.DragStartBehavior = gestures.DragStartBehaviorDown
		})
	s.gestureRecognizers[reflect.TypeFor[*gestures.LongPressGestureRecognizer]()] = NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.LongPressGestureRecognizer {
			return gestures.NewLongPressGestureRecognizer(s.binding().Gestures(),
				gestures.LongPressGestureRecognizerOpts{DebugOwner: s, SupportedDevices: selectableRegionLongPressSelectionDevices})
		},
		func(r *gestures.LongPressGestureRecognizer) {
			r.OnLongPressStart = s.handleTouchLongPressStart
			r.OnLongPressMoveUpdate = s.handleTouchLongPressMoveUpdate
			r.OnLongPressEnd = s.handleTouchLongPressEnd
		})
}

func (s *SelectableRegionState) startNewMouseSelectionGesture(details gestures.TapDragDownDetails) {
	s.lastPointerDeviceKind = details.Kind
	platform := foundation.DefaultTargetPlatform()
	mobile := platform == foundation.TargetPlatformAndroid || platform == foundation.TargetPlatformFuchsia || platform == foundation.TargetPlatformIOS
	precise := details.Kind.IsSome() && isPrecisePointerDevice(details.Kind.Value())
	switch s.effectiveConsecutiveTapCount(details.ConsecutiveTapCount) {
	case 1:
		s.focusNode().RequestFocus()
		if mobile {
			// On mobile platforms the selection is set on tap up for the first tap.
			break
		}
		s.HideToolbar(true)
		// It is impossible to extend the selection when the shift key is pressed
		// and the start of the selection has not been initialized. In this case we
		// fallback on collapsing the selection to first initialize the selection.
		if s.isShiftPressed && s.selectionDelegate.Value().StartSelectionPoint != nil {
			s.selectEndTo(details.GlobalPosition, false, rendering.TextGranularityUnspecified)
			s.setChanging()
			break
		}
		s.ClearSelection()
		s.collapseSelectionAt(details.GlobalPosition)
		s.setChanging()
	case 2:
		s.selectWordAt(details.GlobalPosition)
		s.setChanging()
		if platform == foundation.TargetPlatformIOS && details.Kind.IsSome() && !precise {
			s.showHandles()
		}
	case 3:
		if mobile && !precise {
			// Triple tap on static text is only supported on mobile platforms using
			// a precise pointer device.
			break
		}
		s.selectParagraphAt(details.GlobalPosition)
		s.setChanging()
	}
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) handleMouseDragStart(details gestures.TapDragStartDetails) {
	if s.effectiveConsecutiveTapCount(details.ConsecutiveTapCount) == 1 {
		if k, ok := details.Kind.Get(); ok && !isPrecisePointerDevice(k) {
			// Drag to select is only enabled with a precise pointer device.
			return
		}
		s.selectStartTo(details.GlobalPosition, false, rendering.TextGranularityUnspecified)
		s.setChanging()
	}
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) handleMouseDragUpdate(details gestures.TapDragUpdateDetails) {
	platform := foundation.DefaultTargetPlatform()
	mobile := platform == foundation.TargetPlatformAndroid || platform == foundation.TargetPlatformFuchsia || platform == foundation.TargetPlatformIOS
	kind, hasKind := details.Kind.Get()
	switch s.effectiveConsecutiveTapCount(details.ConsecutiveTapCount) {
	case 1:
		if hasKind && !isPrecisePointerDevice(kind) {
			// Drag to select is only enabled with a precise pointer device.
			return
		}
		s.selectEndTo(details.GlobalPosition, true, rendering.TextGranularityUnspecified)
		s.setChanging()
	case 2:
		s.selectEndTo(details.GlobalPosition, true, rendering.TextGranularityWord)
		s.setChanging()
		if platform == foundation.TargetPlatformIOS && hasKind && !isPrecisePointerDevice(kind) {
			s.showHandles()
		}
	case 3:
		// Triple tap + drag is only supported on mobile devices when using a
		// precise pointer device.
		if mobile && (!hasKind || !isPrecisePointerDevice(kind)) {
			break
		}
		s.selectEndTo(details.GlobalPosition, true, rendering.TextGranularityParagraph)
		s.setChanging()
	}
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) handleMouseDragEnd(gestures.TapDragEndDetails) {
	isPointerPrecise := s.lastPointerDeviceKind.IsSome() && isPrecisePointerDevice(s.lastPointerDeviceKind.Value())
	// On mobile platforms like android, fuchsia, and iOS, a drag gesture will
	// only show the selection overlay when the drag has finished and the pointer
	// device kind is not precise, for example at the end of a double tap + drag
	// to select on native iOS.
	shouldShowSelectionOverlayOnMobile := !isPointerPrecise
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		if shouldShowSelectionOverlayOnMobile {
			s.showHandles()
			s.showToolbar(foundation.None[ui.Offset]())
		}
	case foundation.TargetPlatformIOS:
		if shouldShowSelectionOverlayOnMobile {
			s.showToolbar(foundation.None[ui.Offset]())
		}
	}
	// The selection overlay is not shown on desktop platforms after a drag.
	s.finalizeSelection()
	s.updateSelectedContentIfNeeded()
	s.finalizeSelectableRegionStatus()
}

func (s *SelectableRegionState) handleMouseTapUp(details gestures.TapDragUpDetails) {
	platform := foundation.DefaultTargetPlatform()
	if platform == foundation.TargetPlatformIOS && s.positionIsOnActiveSelection(details.GlobalPosition) {
		// On iOS when the tap occurs on the previous selection, instead of moving
		// the selection, the context menu will be toggled.
		if s.selectionOverlay != nil && s.selectionOverlay.ToolbarIsVisible() {
			s.HideToolbar(false)
		} else {
			s.showToolbar(foundation.None[ui.Offset]())
		}
		return
	}
	switch s.effectiveConsecutiveTapCount(details.ConsecutiveTapCount) {
	case 1:
		switch platform {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS:
			s.HideToolbar(true)
			s.collapseSelectionAt(details.GlobalPosition)
			s.setChanging()
		}
		// On desktop platforms the selection is set on tap down.
	case 2:
		isPointerPrecise := isPrecisePointerDevice(details.Kind)
		switch platform {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
			if !isPointerPrecise {
				// On Android, a double tap will only show the selection overlay after
				// the following tap up when the pointer device kind is not precise.
				s.showHandles()
				s.showToolbar(foundation.None[ui.Offset]())
			}
		case foundation.TargetPlatformIOS:
			if !isPointerPrecise {
				// On iOS, a double tap will only show the selection toolbar after the
				// following tap up when the pointer device kind is not precise.
				s.showToolbar(foundation.None[ui.Offset]())
			}
		}
		// The selection overlay is not shown on desktop platforms on a double click.
	}
	s.finalizeSelectableRegionStatus()
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) updateSelectedContentIfNeeded() {
	w := s.Widget()
	if w.o.OnSelectionChanged == nil || s.selectable == nil {
		return
	}
	content := s.selectable.GetSelectedContent()
	var last, current string
	if s.lastSelectedContent != nil {
		last = s.lastSelectedContent.PlainText
	}
	if content != nil {
		current = content.PlainText
	}
	if (s.lastSelectedContent == nil) != (content == nil) || last != current {
		s.lastSelectedContent = content
		w.o.OnSelectionChanged(content)
	}
}

func (s *SelectableRegionState) handleTouchLongPressStart(details gestures.LongPressStartDetails) {
	s.focusNode().RequestFocus()
	s.selectWordAt(details.GlobalPosition)
	s.setChanging()
	// Platforms besides Android will show the text selection handles when the
	// long press is initiated. Android shows the text selection handles when the
	// long press has ended, usually after a pointer up event is received.
	if foundation.DefaultTargetPlatform() != foundation.TargetPlatformAndroid {
		s.showHandles()
	}
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) handleTouchLongPressMoveUpdate(details gestures.LongPressMoveUpdateDetails) {
	s.selectEndTo(details.GlobalPosition, false, rendering.TextGranularityWord)
	s.setChanging()
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) handleTouchLongPressEnd(gestures.LongPressEndDetails) {
	s.finalizeSelection()
	s.updateSelectedContentIfNeeded()
	s.finalizeSelectableRegionStatus()
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid {
		s.showHandles()
	}
	s.showToolbar(foundation.None[ui.Offset]())
}

func (s *SelectableRegionState) positionIsOnActiveSelection(globalPosition ui.Offset) bool {
	if s.selectable == nil {
		return false
	}
	for _, selectionRect := range s.selectionDelegate.Value().SelectionRects {
		globalRect := s.selectable.GetTransformTo(nil).TransformRect(selectionRect)
		if globalRect.Contains(globalPosition) {
			return true
		}
	}
	return false
}

func (s *SelectableRegionState) handleRightClickDown(details gestures.TapDownDetails) {
	previousSecondaryTapDownPosition := s.lastSecondaryTapDownPosition
	toolbarIsVisible := s.selectionOverlay != nil && s.selectionOverlay.ToolbarIsVisible()
	s.lastSecondaryTapDownPosition = foundation.Some(details.GlobalPosition)
	s.focusNode().RequestFocus()
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformWindows:
		// If the position is within the current selection then keep the current
		// selection, if not then collapse it.
		if s.positionIsOnActiveSelection(details.GlobalPosition) {
			// Restore the position since it may be cleared if a user accesses
			// ContextMenuAnchors.
			s.lastSecondaryTapDownPosition = foundation.Some(details.GlobalPosition)
			s.showHandles()
			s.showToolbar(s.lastSecondaryTapDownPosition)
			s.updateSelectedContentIfNeeded()
			return
		}
		s.collapseSelectionAt(details.GlobalPosition)
	case foundation.TargetPlatformIOS:
		s.selectWordAt(details.GlobalPosition)
	case foundation.TargetPlatformMacOS:
		if previousSecondaryTapDownPosition == s.lastSecondaryTapDownPosition && toolbarIsVisible {
			s.HideToolbar(true)
			return
		}
		s.selectWordAt(details.GlobalPosition)
	case foundation.TargetPlatformLinux:
		if toolbarIsVisible {
			s.HideToolbar(true)
			return
		}
		// If the position is within the current selection then keep the current
		// selection, if not then collapse it.
		if !s.positionIsOnActiveSelection(details.GlobalPosition) {
			s.collapseSelectionAt(details.GlobalPosition)
		}
	}
	s.setChanging()
	s.finalizeSelectableRegionStatus()
	// Restore the position since it may be cleared if a user accesses
	// ContextMenuAnchors.
	s.lastSecondaryTapDownPosition = foundation.Some(details.GlobalPosition)
	s.showHandles()
	s.showToolbar(s.lastSecondaryTapDownPosition)
	s.updateSelectedContentIfNeeded()
}

// triggerSelectionEndEdgeUpdate sends an end SelectionEdgeUpdateEvent to the
// selectable subtree. If the selectable subtree returns pending, it continues
// to send the events every frame until the result is not pending or users end
// their gestures.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._triggerSelectionEndEdgeUpdate
func (s *SelectableRegionState) triggerSelectionEndEdgeUpdate(granularity rendering.TextGranularity) {
	// This method can be called when the drag is not in progress. This can
	// happen if the child scrollable returns pending, and the selection area
	// scheduled a selection update for the next frame, but the drag is lifted
	// before the scheduled selection update is run.
	if s.scheduledSelectionEndEdgeUpdate || s.selectionEndPosition.IsNone() || s.selectable == nil {
		return
	}
	if s.selectable.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForEnd(s.selectionEndPosition.Value(), granularity)) == rendering.SelectionResultPending {
		s.scheduledSelectionEndEdgeUpdate = true
		s.binding().RunPostFrame(func(time.Duration) {
			if !s.scheduledSelectionEndEdgeUpdate {
				return
			}
			s.scheduledSelectionEndEdgeUpdate = false
			s.triggerSelectionEndEdgeUpdate(granularity)
		})
	}
}

func (s *SelectableRegionState) onAnyDragEnd(gestures.DragEndDetails) {
	draggingHandles := s.selectionOverlay != nil && (s.selectionOverlay.IsDraggingStartHandle() || s.selectionOverlay.IsDraggingEndHandle())
	if !draggingHandles {
		s.showToolbar(foundation.None[ui.Offset]())
	}
	s.finalizeSelection()
	s.updateSelectedContentIfNeeded()
	s.finalizeSelectableRegionStatus()
}

func (s *SelectableRegionState) stopSelectionEndEdgeUpdate() {
	s.scheduledSelectionEndEdgeUpdate = false
	s.selectionEndPosition = foundation.None[ui.Offset]()
}

// triggerSelectionStartEdgeUpdate sends a start SelectionEdgeUpdateEvent to the
// selectable subtree. See triggerSelectionEndEdgeUpdate.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._triggerSelectionStartEdgeUpdate
func (s *SelectableRegionState) triggerSelectionStartEdgeUpdate(granularity rendering.TextGranularity) {
	if s.scheduledSelectionStartEdgeUpdate || s.selectionStartPosition.IsNone() || s.selectable == nil {
		return
	}
	if s.selectable.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForStart(s.selectionStartPosition.Value(), granularity)) == rendering.SelectionResultPending {
		s.scheduledSelectionStartEdgeUpdate = true
		s.binding().RunPostFrame(func(time.Duration) {
			if !s.scheduledSelectionStartEdgeUpdate {
				return
			}
			s.scheduledSelectionStartEdgeUpdate = false
			s.triggerSelectionStartEdgeUpdate(granularity)
		})
	}
}

func (s *SelectableRegionState) stopSelectionStartEdgeUpdate() {
	s.scheduledSelectionStartEdgeUpdate = false
	// Flutter clears the end position here (it may be meant to be the start one);
	// the port keeps its behavior.
	s.selectionEndPosition = foundation.None[ui.Offset]()
}

func (s *SelectableRegionState) handleSelectionStartHandleDragStart(details gestures.DragStartDetails) {
	local := s.selectionDelegate.Value().StartSelectionPoint.LocalPosition
	s.selectionStartHandleDragPosition = s.selectable.GetTransformTo(nil).TransformPoint(local)
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) handleSelectionStartHandleDragUpdate(details gestures.DragUpdateDetails) {
	s.selectionStartHandleDragPosition = s.selectionStartHandleDragPosition.Add(details.Delta)
	// The value corresponds to the paint origin of the selection handle. Offset
	// it to the center of the line to make it feel more natural.
	s.selectionStartPosition = foundation.Some(s.selectionStartHandleDragPosition.Sub(ui.Offset{Dy: s.selectionDelegate.Value().StartSelectionPoint.LineHeight / 2}))
	s.triggerSelectionStartEdgeUpdate(rendering.TextGranularityUnspecified)
	s.updateSelectedContentIfNeeded()
	s.setChanging()
}

func (s *SelectableRegionState) handleSelectionEndHandleDragStart(details gestures.DragStartDetails) {
	local := s.selectionDelegate.Value().EndSelectionPoint.LocalPosition
	s.selectionEndHandleDragPosition = s.selectable.GetTransformTo(nil).TransformPoint(local)
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) handleSelectionEndHandleDragUpdate(details gestures.DragUpdateDetails) {
	s.selectionEndHandleDragPosition = s.selectionEndHandleDragPosition.Add(details.Delta)
	s.selectionEndPosition = foundation.Some(s.selectionEndHandleDragPosition.Sub(ui.Offset{Dy: s.selectionDelegate.Value().EndSelectionPoint.LineHeight / 2}))
	s.triggerSelectionEndEdgeUpdate(rendering.TextGranularityUnspecified)
	s.updateSelectedContentIfNeeded()
	s.setChanging()
}

func (s *SelectableRegionState) createSelectionOverlay() {
	if s.selectionOverlay != nil {
		return
	}
	v := s.selectionDelegate.Value()
	startType, endType := rendering.TextSelectionHandleTypeCollapsed, rendering.TextSelectionHandleTypeCollapsed
	var startHeight, endHeight float64
	if v.StartSelectionPoint != nil {
		startType, startHeight = v.StartSelectionPoint.HandleType, v.StartSelectionPoint.LineHeight
	} else {
		startHeight = v.EndSelectionPoint.LineHeight
	}
	if v.EndSelectionPoint != nil {
		endType, endHeight = v.EndSelectionPoint.HandleType, v.EndSelectionPoint.LineHeight
	} else {
		endHeight = v.StartSelectionPoint.LineHeight
	}
	s.selectionOverlay = NewSelectionOverlay(s.Context(), startType, startHeight, endType, endHeight, s.SelectionEndpoints(),
		s.Widget().selectionControls, s.startHandleLink, s.endHandleLink, s.toolbarLink, SelectionOverlayOpts{
			DebugRequiredFor:        s.Widget(),
			OnStartHandleDragStart:  s.handleSelectionStartHandleDragStart,
			OnStartHandleDragUpdate: s.handleSelectionStartHandleDragUpdate,
			OnStartHandleDragEnd:    s.onAnyDragEnd,
			OnEndHandleDragStart:    s.handleSelectionEndHandleDragStart,
			OnEndHandleDragUpdate:   s.handleSelectionEndHandleDragUpdate,
			OnEndHandleDragEnd:      s.onAnyDragEnd,
		})
}

func (s *SelectableRegionState) updateSelectionOverlay() {
	if s.selectionOverlay == nil {
		return
	}
	v := s.selectionDelegate.Value()
	startType, endType := rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight
	var startHeight, endHeight float64
	if v.StartSelectionPoint != nil {
		startType, startHeight = v.StartSelectionPoint.HandleType, v.StartSelectionPoint.LineHeight
	} else {
		startHeight = v.EndSelectionPoint.LineHeight
	}
	if v.EndSelectionPoint != nil {
		endType, endHeight = v.EndSelectionPoint.HandleType, v.EndSelectionPoint.LineHeight
	} else {
		endHeight = v.StartSelectionPoint.LineHeight
	}
	o := s.selectionOverlay
	o.SetStartHandleType(startType)
	o.SetLineHeightAtStart(startHeight)
	o.SetEndHandleType(endType)
	o.SetLineHeightAtEnd(endHeight)
	o.SetSelectionEndpoints(s.SelectionEndpoints())
}

// showHandles shows the selection handles, and says whether it did.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._showHandles
func (s *SelectableRegionState) showHandles() bool {
	if s.selectionOverlay != nil {
		s.selectionOverlay.ShowHandles()
		return true
	}
	if !s.hasSelectionOverlayGeometry() {
		return false
	}
	s.createSelectionOverlay()
	s.selectionOverlay.ShowHandles()
	return true
}

// showToolbar shows the text selection toolbar, and says whether it did. If the
// location is given, the toolbar is shown there; otherwise, the toolbar
// location is calculated based on the handles' locations.
//
// Deviation: the toolbar location is the secondary tap position that
// ContextMenuAnchors uses; there is no separate toolbarLocation (SPEC §12.10).
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._showToolbar
func (s *SelectableRegionState) showToolbar(location foundation.Opt[ui.Offset]) bool {
	if !s.hasSelectionOverlayGeometry() && s.selectionOverlay == nil {
		return false
	}
	builder := s.Widget().o.ContextMenuBuilder
	if builder == nil {
		return false
	}
	if s.selectionOverlay == nil {
		s.createSelectionOverlay()
	}
	s.selectionOverlay.HideToolbar()
	s.selectionOverlay.ShowToolbar(s.Context(), func(bc BuildContext) Widget { return builder(bc, s) })
	return true
}

// selectEndTo sets or updates selection end edge to the offset location.
//
// A selection always contains a select start edge and selection end edge. They
// can be created by calling both selectStartTo and selectEndTo, or use other
// selection APIs, such as selectWordAt or SelectAll. This method sets or updates
// the selection end edge by sending SelectionEdgeUpdateEvents to the child
// Selectables.
//
// If continuous is set to true and the update causes scrolling, the method will
// continue sending the same events to the child Selectables every frame until
// the scrolling finishes or finalizeSelection is called. The offset is in
// global coordinates.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._selectEndTo
func (s *SelectableRegionState) selectEndTo(offset ui.Offset, continuous bool, granularity rendering.TextGranularity) {
	if !continuous {
		if s.selectable != nil {
			s.selectable.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForEnd(offset, granularity))
		}
		return
	}
	if s.selectionEndPosition != foundation.Some(offset) {
		s.selectionEndPosition = foundation.Some(offset)
		s.triggerSelectionEndEdgeUpdate(granularity)
	}
}

// selectStartTo sets or updates selection start edge to the offset location.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._selectStartTo
func (s *SelectableRegionState) selectStartTo(offset ui.Offset, continuous bool, granularity rendering.TextGranularity) {
	if !continuous {
		if s.selectable != nil {
			s.selectable.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForStart(offset, granularity))
		}
		return
	}
	if s.selectionStartPosition != foundation.Some(offset) {
		s.selectionStartPosition = foundation.Some(offset)
		s.triggerSelectionStartEdgeUpdate(granularity)
	}
}

// collapseSelectionAt collapses the selection at the given offset location.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._collapseSelectionAt
func (s *SelectableRegionState) collapseSelectionAt(offset ui.Offset) {
	// There may be other selection ongoing.
	s.finalizeSelection()
	s.selectStartTo(offset, false, rendering.TextGranularityUnspecified)
	s.selectEndTo(offset, false, rendering.TextGranularityUnspecified)
}

// selectWordAt selects a whole word at the offset location. If the whole word
// is already in the current selection, selection won't change.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._selectWordAt
func (s *SelectableRegionState) selectWordAt(offset ui.Offset) {
	// There may be other selection ongoing.
	s.finalizeSelection()
	if s.selectable != nil {
		s.selectable.DispatchSelectionEvent(rendering.SelectWordSelectionEvent{GlobalPosition: offset})
	}
}

// selectParagraphAt selects the entire paragraph at the offset location.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._selectParagraphAt
func (s *SelectableRegionState) selectParagraphAt(offset ui.Offset) {
	// There may be other selection ongoing.
	s.finalizeSelection()
	if s.selectable != nil {
		s.selectable.DispatchSelectionEvent(rendering.SelectParagraphSelectionEvent{GlobalPosition: offset})
	}
}

// finalizeSelection stops any ongoing selection updates. It is different from
// ClearSelection in that it does not remove the current selection. It only stops
// the continuous updates.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState._finalizeSelection
func (s *SelectableRegionState) finalizeSelection() {
	s.stopSelectionEndEdgeUpdate()
	s.stopSelectionStartEdgeUpdate()
}

// ClearSelection removes the ongoing selection for this region.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.clearSelection
func (s *SelectableRegionState) ClearSelection() {
	s.finalizeSelection()
	s.directionalHorizontalBaseline = foundation.None[float64]()
	s.adjustingSelectionEnd = foundation.None[bool]()
	if s.selectable != nil {
		s.selectable.DispatchSelectionEvent(rendering.ClearSelectionEvent{})
	}
	s.updateSelectedContentIfNeeded()
}

func (s *SelectableRegionState) copy() {
	if s.selectable == nil {
		return
	}
	data := s.selectable.GetSelectedContent()
	if data == nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.Lifetime(), 5*time.Second)
	s.binding().Clipboard().SetData(ctx, services.NewClipboardData(data.PlainText), func(err error) {
		cancel()
		if err != nil {
			s.binding().ErrorReporter()(errors.NewErr("SelectableRegionCopyFailed", err))
		}
	})
}

// ContextMenuAnchors returns the anchor points for the default context menu.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.contextMenuAnchors
func (s *SelectableRegionState) ContextMenuAnchors() TextSelectionToolbarAnchors {
	if p, ok := s.lastSecondaryTapDownPosition.Get(); ok {
		// Clear the state of the position after use since a user may access
		// ContextMenuAnchors and receive invalid anchors for their context menu.
		s.lastSecondaryTapDownPosition = foundation.None[ui.Offset]()
		return TextSelectionToolbarAnchors{PrimaryAnchor: p}
	}
	return NewTextSelectionToolbarAnchorsFromSelection(s.Context().FindRenderObject().(rendering.Box),
		s.StartGlyphHeight(), s.EndGlyphHeight(), s.SelectionEndpoints())
}

func (s *SelectableRegionState) determineIsAdjustingSelectionEnd(forward bool) bool {
	if a, ok := s.adjustingSelectionEnd.Get(); ok {
		return a
	}
	v := s.selectionDelegate.Value()
	start, end := v.StartSelectionPoint, v.EndSelectionPoint
	var isReversed bool
	switch {
	case start.LocalPosition.Dy > end.LocalPosition.Dy:
		isReversed = true
	case start.LocalPosition.Dy < end.LocalPosition.Dy:
		isReversed = false
	default:
		isReversed = start.LocalPosition.Dx > end.LocalPosition.Dx
	}
	// Always move the selection edge that increases the selection range.
	result := forward != isReversed
	s.adjustingSelectionEnd = foundation.Some(result)
	return result
}

func (s *SelectableRegionState) granularlyExtendSelection(granularity rendering.TextGranularity, forward bool) {
	s.directionalHorizontalBaseline = foundation.None[float64]()
	if !s.selectionDelegate.Value().HasSelection() {
		return
	}
	s.selectable.DispatchSelectionEvent(rendering.GranularlyExtendSelectionEvent{
		Forward: forward, IsEnd: s.determineIsAdjustingSelectionEnd(forward), Granularity: granularity,
	})
	s.updateSelectedContentIfNeeded()
	s.setChanging()
	s.finalizeSelectableRegionStatus()
}

func (s *SelectableRegionState) directionallyExtendSelection(forward bool) {
	v := s.selectionDelegate.Value()
	if !v.HasSelection() {
		return
	}
	adjustingSelectionExtend := s.determineIsAdjustingSelectionEnd(forward)
	baseLinePoint := v.StartSelectionPoint
	if adjustingSelectionExtend {
		baseLinePoint = v.EndSelectionPoint
	}
	if s.directionalHorizontalBaseline.IsNone() {
		s.directionalHorizontalBaseline = foundation.Some(baseLinePoint.LocalPosition.Dx)
	}
	box := s.Context().FindRenderObject()
	global := box.GetTransformTo(nil).TransformPoint(ui.Offset{Dx: s.directionalHorizontalBaseline.Value()})
	direction := rendering.SelectionExtendDirectionPreviousLine
	if forward {
		direction = rendering.SelectionExtendDirectionNextLine
	}
	s.selectable.DispatchSelectionEvent(rendering.DirectionallyExtendSelectionEvent{
		IsEnd: s.adjustingSelectionEnd.Value(), Direction: direction, Dx: global.Dx,
	})
	s.updateSelectedContentIfNeeded()
	s.setChanging()
	s.finalizeSelectableRegionStatus()
}

// ContextMenuButtonItems returns the items representing the buttons in this
// platform's default selection menu.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.contextMenuButtonItems
func (s *SelectableRegionState) ContextMenuButtonItems() []ContextMenuButtonItem {
	clearOrHide := func() {
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
			// On Android copy should clear the selection.
			s.ClearSelection()
			s.setChanging()
			s.finalizeSelectableRegionStatus()
		case foundation.TargetPlatformIOS:
			s.HideToolbar(false)
		default:
			s.HideToolbar(true)
		}
	}
	return GetSelectableButtonItems(s.selectionDelegate.Value(), SelectableButtonItemsCallbacks{
		OnCopy: func() {
			s.copy()
			clearOrHide()
		},
		OnSelectAll: func() {
			switch foundation.DefaultTargetPlatform() {
			case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia:
				s.SelectAll(services.SelectionChangedCauseToolbar)
			default:
				s.SelectAll()
				s.HideToolbar(true)
			}
		},
	})
}

// StartGlyphHeight is the line height at the start of the current selection.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.startGlyphHeight
func (s *SelectableRegionState) StartGlyphHeight() float64 {
	return s.selectionDelegate.Value().StartSelectionPoint.LineHeight
}

// EndGlyphHeight is the line height at the end of the current selection.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.endGlyphHeight
func (s *SelectableRegionState) EndGlyphHeight() float64 {
	return s.selectionDelegate.Value().EndSelectionPoint.LineHeight
}

// SelectionEndpoints returns the local coordinates of the endpoints of the
// current selection.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.selectionEndpoints
func (s *SelectableRegionState) SelectionEndpoints() []rendering.TextSelectionPoint {
	v := s.selectionDelegate.Value()
	var startLocal, endLocal ui.Offset
	switch {
	case v.StartSelectionPoint != nil:
		startLocal = v.StartSelectionPoint.LocalPosition
	case v.EndSelectionPoint != nil:
		startLocal = v.EndSelectionPoint.LocalPosition
	}
	if v.EndSelectionPoint != nil {
		endLocal = v.EndSelectionPoint.LocalPosition
	} else {
		endLocal = startLocal
	}
	ltr := foundation.Some(ui.TextDirectionLTR)
	if startLocal.Dy > endLocal.Dy {
		return []rendering.TextSelectionPoint{{Point: endLocal, Direction: ltr}, {Point: startLocal, Direction: ltr}}
	}
	return []rendering.TextSelectionPoint{{Point: startLocal, Direction: ltr}, {Point: endLocal, Direction: ltr}}
}

// HideToolbar hides the toolbar, and the handles when hideHandles is set.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.hideToolbar
func (s *SelectableRegionState) HideToolbar(hideHandles bool) {
	if s.selectionOverlay == nil {
		return
	}
	s.selectionOverlay.HideToolbar()
	if hideHandles {
		s.selectionOverlay.HideHandles()
	}
}

func (s *SelectableRegionState) hideToolbarIfVisible(i DismissIntent) any {
	if s.selectionOverlay != nil && s.selectionOverlay.ToolbarIsVisible() {
		s.HideToolbar(false)
		return nil
	}
	return ActionsInvoke(s.Context(), i)
}

// SelectAll selects all the content of the region. The cause is the toolbar when
// the handles and the toolbar are shown after it.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.selectAll
func (s *SelectableRegionState) SelectAll(cause ...services.SelectionChangedCause) {
	s.ClearSelection()
	if s.selectable != nil {
		s.selectable.DispatchSelectionEvent(rendering.SelectAllSelectionEvent{})
	}
	if len(cause) > 0 && cause[0] == services.SelectionChangedCauseToolbar {
		s.showHandles()
		s.showToolbar(foundation.None[ui.Offset]())
	}
	s.updateSelectedContentIfNeeded()
	s.setChanging()
	s.finalizeSelectableRegionStatus()
}

// CopySelection copies the selected content and clears the selection.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.copySelection
func (s *SelectableRegionState) CopySelection() {
	s.copy()
	s.ClearSelection()
	s.setChanging()
	s.finalizeSelectableRegionStatus()
}

// Add registers the selectable, which is the SelectionContainer of the region.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.add
func (s *SelectableRegionState) Add(selectable rendering.Selectable) {
	if foundation.DebugMode && s.selectable != nil {
		panic(errors.New("SelectableRegionMultipleSelectables", "a SelectableRegion has one selectable: the SelectionContainer it creates"))
	}
	s.selectable = selectable
	s.selectableHandle = selectable.AddListener(s.updateSelectionStatus)
	selectable.PushHandleLayers(s.startHandleLink, s.endHandleLink)
}

// Remove unregisters the selectable.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.remove
func (s *SelectableRegionState) Remove(selectable rendering.Selectable) {
	if foundation.DebugMode && s.selectable != selectable {
		panic(errors.New("SelectableRegionUnknownSelectable", "the selectable is not the one of this SelectableRegion"))
	}
	s.selectable.RemoveListener(s.selectableHandle)
	s.selectable.PushHandleLayers(nil, nil)
	s.selectable = nil
}

// Dispose releases the delegate, the overlay and the focus node.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.dispose
func (s *SelectableRegionState) Dispose() {
	if s.selectable != nil {
		s.selectable.RemoveListener(s.selectableHandle)
		s.selectable.PushHandleLayers(nil, nil)
	}
	s.selectionDelegate.Dispose()
	s.selectionStatusNotifier.Dispose()
	if s.selectionOverlay != nil {
		s.selectionOverlay.Dispose()
		s.selectionOverlay = nil
	}
	s.stopListeningToFocus()
	if s.localFocusNode != nil {
		s.localFocusNode.Dispose()
	}
}

// Build builds the selection container with the gestures and the actions.
//
// Flutter: widgets/selectable_region.dart SelectableRegionState.build
func (s *SelectableRegionState) Build(bc BuildContext) Widget {
	w := s.Widget()
	result := &SelectableRegionSelectionStatusScope{
		InheritedWidget:         NewInheritedWidgetBase(NewSelectionContainer(s.selectionDelegate, w.child, SelectionContainerOpts{Registrar: s})),
		selectionStatusNotifier: s.selectionStatusNotifier,
	}
	return NewTapRegion(
		NewCompositedTransformTarget(s.toolbarLink,
			NewRawGestureDetector(
				NewActions(s.actions, NewFocusWithExternalFocusNode(s.focusNode(), result, FocusWithExternalFocusNodeOpts{ExcludeSemantics: true})),
				RawGestureDetectorOpts{Gestures: s.gestureRecognizers, Behavior: rendering.HitTestBehaviorTranslucent, ExcludeFromSemantics: true})),
		TapRegionOpts{GroupID: SelectableRegionTapRegionGroupID{}})
}

// nonOverrideAction is an action that does not override any overridable action
// in the subtree. If this action is invoked by an overridable action, it will
// immediately invoke the overridable action and do nothing else. Otherwise, it
// will call invokeAction.
//
// Flutter: widgets/selectable_region.dart _NonOverrideAction
type nonOverrideAction[T Intent] struct {
	ContextActionBase
	invokeAction func(intent T)
}

func newNonOverrideAction[T Intent](invokeAction func(intent T)) *nonOverrideAction[T] {
	a := &nonOverrideAction[T]{invokeAction: invokeAction}
	InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *nonOverrideAction[T]) IntentType() reflect.Type { return reflect.TypeFor[T]() }

func (a *nonOverrideAction[T]) InvokeWithContext(intent Intent, _ BuildContext) any {
	if calling := a.CallingAction(); calling != nil {
		return calling.Invoke(intent)
	}
	a.invokeAction(intent.(T))
	return nil
}

func (s *SelectableRegionState) buildActions() ActionMap {
	m := ActionMap{}
	add := func(a Action) { m[a.IntentType()] = s.makeOverridable(a) }
	add(newNonOverrideAction(func(SelectAllTextIntent) { s.SelectAll(services.SelectionChangedCauseKeyboard) }))
	add(newNonOverrideAction(func(CopySelectionTextIntent) { s.copy() }))
	add(newNonOverrideAction(func(i ExtendSelectionToNextWordBoundaryOrCaretLocationIntent) {
		s.granularlyExtendSelection(rendering.TextGranularityWord, i.IsForward())
	}))
	add(newNonOverrideAction(func(i ExpandSelectionToDocumentBoundaryIntent) {
		s.granularlyExtendSelection(rendering.TextGranularityDocument, i.IsForward())
	}))
	add(newNonOverrideAction(func(i ExpandSelectionToLineBreakIntent) {
		s.granularlyExtendSelection(rendering.TextGranularityLine, i.IsForward())
	}))
	caret := func(granularity rendering.TextGranularity, i DirectionalCaretMovementIntent) {
		if i.CollapseSelection() {
			// Selectable region never collapses selection.
			return
		}
		s.granularlyExtendSelection(granularity, i.IsForward())
	}
	add(newNonOverrideAction(func(i ExtendSelectionByCharacterIntent) { caret(rendering.TextGranularityCharacter, i) }))
	add(newNonOverrideAction(func(i ExtendSelectionToNextWordBoundaryIntent) { caret(rendering.TextGranularityWord, i) }))
	add(newNonOverrideAction(func(i ExtendSelectionToLineBreakIntent) { caret(rendering.TextGranularityLine, i) }))
	add(newNonOverrideAction(func(i ExtendSelectionVerticallyToAdjacentLineIntent) {
		if i.CollapseSelection() {
			return
		}
		s.directionallyExtendSelection(i.IsForward())
	}))
	add(newNonOverrideAction(func(i ExtendSelectionToDocumentBoundaryIntent) { caret(rendering.TextGranularityDocument, i) }))
	d := NewCallbackAction(s.hideToolbarIfVisible)
	m[d.IntentType()] = d
	return m
}
