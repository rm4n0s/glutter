package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
)

// undoThrottleDuration is how long the changes of the value are coalesced into
// one entry of the undo stack.
//
// Flutter: widgets/undo_history.dart UndoHistoryState._kThrottleDuration
const undoThrottleDuration = 500 * time.Millisecond

// UndoHistoryOpts are the optional parameters of NewUndoHistory.
//
// Flutter: widgets/undo_history.dart UndoHistory.new
type UndoHistoryOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ShouldChangeUndoStack says whether a new value goes on the undo stack,
	// given the last value (hasOld is false when there is none); the default
	// accepts every value.
	ShouldChangeUndoStack func(oldValue T, hasOld bool, newValue T) bool
	// UndoStackModifier changes a value before it goes on the stack; the default
	// keeps it as it is.
	UndoStackModifier func(value T) T
	// Controller can undo and redo from outside the widget, and tells whether
	// they are possible.
	Controller *UndoHistoryController
}

// UndoHistory keeps the history of the values of a Listenable value and undoes
// and redoes them for the UndoTextIntent and RedoTextIntent below it. The
// changes in quick succession (500 ms) count as one.
//
// Deviation: Flutter's iOS UndoManager integration doesn't exist (SPEC §12.10).
//
// Flutter: widgets/undo_history.dart UndoHistory
type UndoHistory[T comparable] struct {
	StatefulWidget
	value       foundation.ValueListenable[T]
	onTriggered func(value T)
	focusNode   *FocusNode
	child       Widget
	o           UndoHistoryOpts[T]
}

// NewUndoHistory creates the widget: onTriggered receives the value to go back
// to or forward to, and focusNode is the focus of the field.
//
// Flutter: widgets/undo_history.dart UndoHistory.new
func NewUndoHistory[T comparable](value foundation.ValueListenable[T], onTriggered func(value T), focusNode *FocusNode, child Widget, opts ...UndoHistoryOpts[T]) *UndoHistory[T] {
	o := firstOpts(opts)
	return &UndoHistory[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), value: value, onTriggered: onTriggered, focusNode: focusNode, child: child, o: o}
}

// Value is the value that is tracked.
//
// Flutter: widgets/undo_history.dart UndoHistory.value
func (w *UndoHistory[T]) Value() foundation.ValueListenable[T] { return w.value }

// Child is the widget below this one in the tree.
//
// Flutter: widgets/undo_history.dart UndoHistory.child
func (w *UndoHistory[T]) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/undo_history.dart UndoHistory.createState
func (w *UndoHistory[T]) CreateState() State { return &UndoHistoryState[T]{} }

// UndoHistoryState is the state of an UndoHistory.
//
// Flutter: widgets/undo_history.dart UndoHistoryState
type UndoHistoryState[T comparable] struct {
	StateOf[*UndoHistory[T]]

	stack         undoStack[T]
	throttleTimer *gestures.Timer
	throttleArg   T
	duringTrigger bool
	lastValue     T
	hasLast       bool
	controller    *UndoHistoryController

	pushHandle   foundation.ListenerHandle
	focusHandle  foundation.ListenerHandle
	undoHandle   foundation.ListenerHandle
	redoHandle   foundation.ListenerHandle
	listeningCtl *UndoHistoryController
}

func (s *UndoHistoryState[T]) effectiveController() *UndoHistoryController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	if s.controller == nil {
		s.controller = NewUndoHistoryController()
	}
	return s.controller
}

// Undo goes back one value.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.undo
func (s *UndoHistoryState[T]) Undo() {
	cur, ok := s.stack.currentValue()
	if !ok {
		return
	}
	if s.throttleTimer != nil && s.throttleTimer.IsActive() {
		s.throttleTimer.Cancel() // Cancel the push that is waiting.
		s.update(cur, true)
	} else {
		v, has := s.stack.undo()
		s.update(v, has)
	}
	s.updateState()
}

// Redo goes forward one value.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.redo
func (s *UndoHistoryState[T]) Redo() {
	v, has := s.stack.redo()
	s.update(v, has)
	s.updateState()
}

// CanUndo is whether there is a value to go back to.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.canUndo
func (s *UndoHistoryState[T]) CanUndo() bool { return s.stack.canUndo() }

// CanRedo is whether there is a value to go forward to.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.canRedo
func (s *UndoHistoryState[T]) CanRedo() bool { return s.stack.canRedo() }

func (s *UndoHistoryState[T]) updateState() {
	s.effectiveController().SetValue(UndoHistoryValue{CanUndo: s.CanUndo(), CanRedo: s.CanRedo()})
}

func (s *UndoHistoryState[T]) update(next T, has bool) {
	if !has {
		return
	}
	if s.hasLast && next == s.lastValue {
		return
	}
	s.lastValue, s.hasLast = next, true
	s.duringTrigger = true
	defer func() { s.duringTrigger = false }()
	s.Widget().onTriggered(next)
	if foundation.DebugMode && s.Widget().value.Value() != next {
		panic(errors.New(tagUndoHistoryValueNotSet, "onTriggered must set the value of the UndoHistory to the value it is given",
			"triggered", next, "value", s.Widget().value.Value()))
	}
}

func (s *UndoHistoryState[T]) push() {
	w := s.Widget()
	cur := w.value.Value()
	if s.hasLast && cur == s.lastValue {
		return
	}
	if s.duringTrigger {
		return
	}
	if f := w.o.ShouldChangeUndoStack; f != nil && !f(s.lastValue, s.hasLast, cur) {
		return
	}
	next := cur
	if w.o.UndoStackModifier != nil {
		next = w.o.UndoStackModifier(cur)
	}
	if s.hasLast && next == s.lastValue {
		return
	}
	s.lastValue, s.hasLast = next, true
	s.throttledPush(next)
}

// throttledPush pushes the value after the throttle duration; a value that
// arrives while the timer runs replaces the one it will push.
//
// Flutter: widgets/undo_history.dart _throttle
func (s *UndoHistoryState[T]) throttledPush(arg T) {
	s.throttleArg = arg
	if s.throttleTimer != nil && s.throttleTimer.IsActive() {
		return
	}
	s.throttleTimer = WidgetsBindingOf(s.Context()).Gestures().NewTimer(undoThrottleDuration, func() {
		s.stack.push(s.throttleArg)
		s.updateState()
	})
}

// InitState starts to listen to the value, the focus and the controller.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.initState
func (s *UndoHistoryState[T]) InitState() {
	s.push()
	s.listen()
}

func (s *UndoHistoryState[T]) listen() {
	w := s.Widget()
	s.pushHandle = w.value.AddListener(s.push)
	s.focusHandle = w.focusNode.AddListener(s.handleFocus)
	s.handleFocus()
	s.listeningCtl = s.effectiveController()
	s.undoHandle = s.listeningCtl.OnUndo.AddListener(s.Undo)
	s.redoHandle = s.listeningCtl.OnRedo.AddListener(s.Redo)
}

func (s *UndoHistoryState[T]) handleFocus() {
	if s.Widget().focusNode.HasFocus() {
		s.updateState()
	}
}

// DidUpdateWidget moves the listeners to the new value, focus node and
// controller; a new value clears the stack.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.didUpdateWidget
func (s *UndoHistoryState[T]) DidUpdateWidget(old *UndoHistory[T]) {
	w := s.Widget()
	if w.value != old.value {
		s.stack.clear()
		old.value.RemoveListener(s.pushHandle)
		s.pushHandle = w.value.AddListener(s.push)
	}
	if w.focusNode != old.focusNode {
		old.focusNode.RemoveListener(s.focusHandle)
		s.focusHandle = w.focusNode.AddListener(s.handleFocus)
	}
	if w.o.Controller != old.o.Controller {
		s.listeningCtl.OnUndo.RemoveListener(s.undoHandle)
		s.listeningCtl.OnRedo.RemoveListener(s.redoHandle)
		if s.controller != nil {
			s.controller.Dispose()
			s.controller = nil
		}
		s.listeningCtl = s.effectiveController()
		s.undoHandle = s.listeningCtl.OnUndo.AddListener(s.Undo)
		s.redoHandle = s.listeningCtl.OnRedo.AddListener(s.Redo)
	}
}

// Dispose stops listening and cancels the push that is waiting.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.dispose
func (s *UndoHistoryState[T]) Dispose() {
	w := s.Widget()
	w.value.RemoveListener(s.pushHandle)
	w.focusNode.RemoveListener(s.focusHandle)
	s.listeningCtl.OnUndo.RemoveListener(s.undoHandle)
	s.listeningCtl.OnRedo.RemoveListener(s.redoHandle)
	if s.controller != nil {
		s.controller.Dispose()
	}
	if s.throttleTimer != nil {
		s.throttleTimer.Cancel()
	}
}

// Build binds the actions for the undo and redo intents.
//
// Flutter: widgets/undo_history.dart UndoHistoryState.build
func (s *UndoHistoryState[T]) Build(bc BuildContext) Widget {
	return NewActions(ActionMap{
		IntentTypeOf[UndoTextIntent](): NewOverridableAction(NewCallbackAction(func(UndoTextIntent) any { s.Undo(); return nil }), bc),
		IntentTypeOf[RedoTextIntent](): NewOverridableAction(NewCallbackAction(func(RedoTextIntent) any { s.Redo(); return nil }), bc),
	}, s.Widget().child)
}

// UndoHistoryValue is the state of an UndoHistoryController: whether it can undo
// and redo.
//
// Flutter: widgets/undo_history.dart UndoHistoryValue
type UndoHistoryValue struct {
	// CanUndo is whether there is a value to go back to.
	CanUndo bool
	// CanRedo is whether there is a value to go forward to.
	CanRedo bool
}

// UndoHistoryValueEmpty is the value that can neither undo nor redo.
//
// Flutter: widgets/undo_history.dart UndoHistoryValue.empty
var UndoHistoryValueEmpty = UndoHistoryValue{}

// String describes the value.
func (v UndoHistoryValue) String() string {
	return "UndoHistoryValue(canUndo: " + boolString(v.CanUndo) + ", canRedo: " + boolString(v.CanRedo) + ")"
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// UndoHistoryController controls an UndoHistory from outside, and tells whether
// it can undo and redo.
//
// Flutter: widgets/undo_history.dart UndoHistoryController
type UndoHistoryController struct {
	foundation.ValueNotifier[UndoHistoryValue]
	// OnUndo notifies the UndoHistory to undo.
	//
	// Flutter: widgets/undo_history.dart UndoHistoryController.onUndo
	OnUndo *foundation.ChangeNotifier
	// OnRedo notifies the UndoHistory to redo.
	//
	// Flutter: widgets/undo_history.dart UndoHistoryController.onRedo
	OnRedo *foundation.ChangeNotifier
}

// NewUndoHistoryController creates a controller that can neither undo nor redo.
//
// Flutter: widgets/undo_history.dart UndoHistoryController.new
func NewUndoHistoryController() *UndoHistoryController {
	return &UndoHistoryController{
		ValueNotifier: *foundation.NewValueNotifier(UndoHistoryValueEmpty),
		OnUndo:        foundation.NewChangeNotifier(), OnRedo: foundation.NewChangeNotifier(),
	}
}

// Undo asks the UndoHistory to go back, when it can.
//
// Flutter: widgets/undo_history.dart UndoHistoryController.undo
func (c *UndoHistoryController) Undo() {
	if !c.Value().CanUndo {
		return
	}
	c.OnUndo.NotifyListeners()
}

// Redo asks the UndoHistory to go forward, when it can.
//
// Flutter: widgets/undo_history.dart UndoHistoryController.redo
func (c *UndoHistoryController) Redo() {
	if !c.Value().CanRedo {
		return
	}
	c.OnRedo.NotifyListeners()
}

// Dispose releases the controller.
//
// Flutter: widgets/undo_history.dart UndoHistoryController.dispose
func (c *UndoHistoryController) Dispose() {
	c.OnUndo.Dispose()
	c.OnRedo.Dispose()
	c.ValueNotifier.Dispose()
}

// undoStack is the stack of the values that can be undone and redone.
//
// Flutter: widgets/undo_history.dart _UndoStack
//
// Deviation: Flutter's _UndoStack keeps every value. Here the stack keeps at most
// maxUndoEntries values and maxUndoBytes bytes of text, and drops the oldest
// first: in Go each entry of a text field holds a copy of the whole text, so an
// unbounded stack grows by the size of the text at each edit (SPEC §12.10).
type undoStack[T comparable] struct {
	list  []T
	index int
	// bytes is the size of the values in list, as undoSize counts it.
	bytes int
}

// maxUndoEntries and maxUndoBytes bound the undo stack (see undoStack). A short
// field keeps 1,000 steps; a field with a large text keeps the steps that fit in
// 16 MiB. The current value and the one before it are always kept.
const (
	maxUndoEntries = 1000
	maxUndoBytes   = 16 << 20
)

// undoSize is the memory of a value that the stack counts: the text of a
// TextEditingValue, and nothing for other values.
func undoSize[T comparable](v T) int {
	if tv, ok := any(v).(services.TextEditingValue); ok {
		return len(tv.Text)
	}
	return 0
}

func (u *undoStack[T]) currentValue() (T, bool) {
	var zero T
	if len(u.list) == 0 {
		return zero, false
	}
	return u.list[u.index], true
}

func (u *undoStack[T]) canUndo() bool { return len(u.list) > 0 && u.index > 0 }
func (u *undoStack[T]) canRedo() bool { return len(u.list) > 0 && u.index < len(u.list)-1 }

func (u *undoStack[T]) push(value T) {
	if len(u.list) == 0 {
		u.index = 0
		u.list = append(u.list, value)
		u.bytes = undoSize(value)
		return
	}
	if cur, _ := u.currentValue(); value == cur {
		return
	}
	if u.index != len(u.list)-1 {
		for _, v := range u.list[u.index+1:] {
			u.bytes -= undoSize(v)
		}
		clear(u.list[u.index+1:]) // the redo entries that are dropped must not stay in the array
		u.list = u.list[:u.index+1]
	}
	u.list = append(u.list, value)
	u.bytes += undoSize(value)
	u.index = len(u.list) - 1
	u.trim()
}

// trim drops the oldest values while the stack holds more than maxUndoEntries
// values or maxUndoBytes bytes, and keeps the last two. It is called after a push,
// when the current value is the last one.
func (u *undoStack[T]) trim() {
	drop := 0
	for len(u.list)-drop > 2 && (len(u.list)-drop > maxUndoEntries || u.bytes > maxUndoBytes) {
		u.bytes -= undoSize(u.list[drop])
		drop++
	}
	if drop == 0 {
		return
	}
	n := copy(u.list, u.list[drop:])
	clear(u.list[n:]) // the dropped values must not stay in the array
	u.list = u.list[:n]
	u.index -= drop
}

func (u *undoStack[T]) undo() (T, bool) {
	if len(u.list) == 0 {
		var zero T
		return zero, false
	}
	if u.index != 0 {
		u.index--
	}
	return u.currentValue()
}

func (u *undoStack[T]) redo() (T, bool) {
	if len(u.list) == 0 {
		var zero T
		return zero, false
	}
	if u.index < len(u.list)-1 {
		u.index++
	}
	return u.currentValue()
}

func (u *undoStack[T]) clear() {
	u.list = nil
	u.index = -1
	u.bytes = 0
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagUndoHistoryValueNotSet = "UndoHistoryValueNotSet"
)
