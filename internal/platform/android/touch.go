package android

import (
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The MotionEvent.ACTION_* values (the masked action) that the touch input reads.
const (
	actionDown        = 0
	actionUp          = 1
	actionMove        = 2
	actionCancel      = 3
	actionPointerDown = 5
	actionPointerUp   = 6
	actionHoverMove   = 7
	actionScroll      = 8
)

// The MotionEvent.TOOL_TYPE_* values.
//
// TOOL_TYPE_UNKNOWN is 0 and reaches the default of pointerDeviceKindForToolType.
const (
	toolTypeFinger = 1
	toolTypeStylus = 2
	toolTypeMouse  = 3
	toolTypeEraser = 4
)

// The InputDevice.SOURCE_* values that the touch input reads. A source is a class (the low byte)
// and the bits of the device type.
const (
	sourceClassPointer = 0x00000002
	sourceMouse        = 0x00002002
)

const (
	// toolTypeBits is the number of bits of the tool type in a device ID: 3 bits are enough for
	// the 6 kinds of pointer device (Flutter says to check it when the PointerDeviceKind enum
	// reaches 8 values).
	toolTypeBits = 3
	// toolTypeMask cuts a tool type to toolTypeBits bits.
	toolTypeMask = 1<<toolTypeBits - 1

	// pointerDataFlagBatched marks a move that stands for a position update of a pointer other
	// than the one of an ACTION_UP or ACTION_POINTER_UP.
	pointerDataFlagBatched = 1
	// pointerDataFlagMultiple marks data that is part of a group about several pointers; the
	// count of pointers is in the bits from pointerDataMultiplePointerCountShift.
	pointerDataFlagMultiple              = 2
	pointerDataMultiplePointerCountShift = 8

	// defaultHorizontalScrollFactor and defaultVerticalScrollFactor are the scroll factors of a
	// scroll that has no view configuration to ask, chosen by Flutter for a reasonable speed.
	defaultHorizontalScrollFactor = 48.0
	defaultVerticalScrollFactor   = 48.0
)

// motionPointer holds the values that the Java side copies out of one pointer of a MotionEvent.
//
// Flutter: no counterpart
type motionPointer struct {
	// ID is MotionEvent.getPointerId and ToolType is MotionEvent.getToolType (TOOL_TYPE_FINGER 1,
	// STYLUS 2, MOUSE 3, ERASER 4, UNKNOWN 0).
	ID, ToolType int
	// X and Y are MotionEvent.getX and getY, Pressure getPressure, Size getSize, ToolMajor and
	// ToolMinor getToolMajor and getToolMinor, Orientation, Tilt, Distance, HScroll and VScroll
	// getAxisValue of AXIS_ORIENTATION, AXIS_TILT, AXIS_DISTANCE, AXIS_HSCROLL and AXIS_VSCROLL.
	X, Y, Pressure, Size, ToolMajor, ToolMinor, Orientation, Tilt, Distance, HScroll, VScroll float64
}

// motionSample is an older sample of the pointers of a MotionEvent (getHistoricalX and the
// others).
//
// Flutter: no counterpart
type motionSample struct {
	// Time is MotionEvent.getHistoricalEventTime, in milliseconds.
	Time int64
	// Pointers are the pointers of the sample, in the order of the pointers of the event.
	Pointers []motionPointer
}

// motionEvent is a MotionEvent, copied out by the Java side so that the conversion is plain Go.
//
// Flutter: no counterpart
type motionEvent struct {
	// ActionMasked is MotionEvent.getActionMasked (ACTION_DOWN 0, UP 1, MOVE 2, CANCEL 3,
	// POINTER_DOWN 5, POINTER_UP 6, HOVER_MOVE 7, SCROLL 8) and ActionIndex getActionIndex.
	ActionMasked, ActionIndex int
	// Source is MotionEvent.getSource (the InputDevice.SOURCE_* bits) and ButtonState
	// getButtonState.
	Source, ButtonState int
	// EventTime is MotionEvent.getEventTime, in milliseconds.
	EventTime int64
	// PressureMin and PressureMax are the range of the AXIS_PRESSURE of the device of the event
	// (InputDevice.getMotionRange): 0 and 1 when the device is not known. Both zero stand for
	// "not known" too.
	PressureMin, PressureMax float64
	// Pointers are all the pointers of the event.
	Pointers []motionPointer
	// History are the samples that an ACTION_MOVE batches before its current one, oldest first.
	History []motionSample
}

// touchProcessor turns the MotionEvents of the view into the raw pointer data of the framework
// (before the pointerConverter gives it identifiers and deltas). It keeps the pointers that a
// Chromebook reports as a mouse and that are really a trackpad's pan.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor
type touchProcessor struct {
	// ongoingPans are the start positions of the trackpad pans, by device.
	ongoingPans map[int][2]float64
}

// newTouchProcessor makes a processor that has no pan under way.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.AndroidTouchProcessor
func newTouchProcessor() *touchProcessor {
	return &touchProcessor{ongoingPans: map[int][2]float64{}}
}

// OnTouchEvent converts a MotionEvent of onTouchEvent. A touch that goes up is followed by a
// synthesized remove, and the up of a pointer carries the moves of the other fingers; a move
// carries all the pointers, with the samples that Android batched into it first.
//
// Deviation: Flutter's Java ignores the historical samples of an ACTION_MOVE; here each pointer of
// each of them is a move of its own, with the time and the position of the sample, before the data
// of the current sample (SPEC §24.5).
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.onTouchEvent
func (p *touchProcessor) OnTouchEvent(ev motionEvent) []ui.PointerData {
	if ev.ActionIndex < 0 || ev.ActionIndex >= len(ev.Pointers) {
		return nil
	}
	maskedAction := ev.ActionMasked
	pointerChange := pointerChangeForAction(maskedAction)
	updateForSinglePointer := maskedAction == actionDown || maskedAction == actionPointerDown
	updateForMultiplePointers := !updateForSinglePointer && (maskedAction == actionUp || maskedAction == actionPointerUp)

	deviceType := pointerDeviceKindForToolType(ev.Pointers[ev.ActionIndex].ToolType)
	shouldRemovePointer := updateForMultiplePointers && deviceType == ui.PointerDeviceKindTouch
	originalPointerCount := len(ev.Pointers)

	var out []ui.PointerData
	switch {
	case updateForSinglePointer:
		// ACTION_DOWN and ACTION_POINTER_DOWN always apply to a single pointer only.
		out = p.addPointerForIndex(out, ev, ev.Pointers[ev.ActionIndex], pointerChange, 0, defaultHorizontalScrollFactor, defaultVerticalScrollFactor)
	case updateForMultiplePointers:
		// ACTION_UP and ACTION_POINTER_UP may contain position updates for other pointers. They
		// become move events, to preserve this data, marked with a flag so that the framework can
		// reassemble the original Android event.
		for i, ptr := range ev.Pointers {
			if i != ev.ActionIndex && ptr.ToolType == toolTypeFinger {
				out = p.addPointerForIndex(out, ev, ptr, ui.PointerChangeMove, pointerDataFlagBatched, defaultHorizontalScrollFactor, defaultVerticalScrollFactor)
			}
		}
		// The UP is sent last.
		out = p.addPointerForIndex(out, ev, ev.Pointers[ev.ActionIndex], pointerChange, 0, defaultHorizontalScrollFactor, defaultVerticalScrollFactor)
		if shouldRemovePointer {
			// A remove right after the up of a touch divides the touches into segments that begin
			// with a down and end with an up, and keeps the framework from hovering between them:
			// a touch screen cannot hover. (The converter synthesizes the add before the next
			// contact.)
			out = p.addPointerForIndex(out, ev, ev.Pointers[ev.ActionIndex], ui.PointerChangeRemove, 0, defaultHorizontalScrollFactor, defaultVerticalScrollFactor)
		}
	default:
		// ACTION_MOVE may not mean that all pointers have moved, but a later part of the system
		// ignores the zero deltas if it wants to.
		flags := pointerDataFlagMultiple | originalPointerCount<<pointerDataMultiplePointerCountShift
		if maskedAction == actionMove {
			for _, sample := range ev.History {
				for i, ptr := range sample.Pointers {
					if i >= len(ev.Pointers) {
						break
					}
					// The sample has the identity of the pointer of the event at the same index.
					ptr.ID, ptr.ToolType = ev.Pointers[i].ID, ev.Pointers[i].ToolType
					old := ev
					old.EventTime = sample.Time
					out = p.addPointerForIndex(out, old, ptr, pointerChange, flags, defaultHorizontalScrollFactor, defaultVerticalScrollFactor)
				}
			}
		}
		for _, ptr := range ev.Pointers {
			out = p.addPointerForIndex(out, ev, ptr, pointerChange, flags, defaultHorizontalScrollFactor, defaultVerticalScrollFactor)
		}
	}
	return out
}

// OnGenericMotionEvent converts a generic MotionEvent (mouse hover, scroll wheel, trackpad): the
// result is empty and false for an event that is not from a pointer or is neither a hover move nor
// a scroll. hScrollFactor and vScrollFactor are the pixels of one unit of scroll (the view
// configuration's scaled scroll factors); a scroll with them zero is a scroll of no distance.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.onGenericMotionEvent
func (p *touchProcessor) OnGenericMotionEvent(ev motionEvent, hScrollFactor, vScrollFactor float64) ([]ui.PointerData, bool) {
	isPointerEvent := ev.Source&sourceClassPointer == sourceClassPointer
	isMovementEvent := ev.ActionMasked == actionHoverMove || ev.ActionMasked == actionScroll
	if !isPointerEvent || !isMovementEvent {
		return nil, false
	}
	if ev.ActionIndex < 0 || ev.ActionIndex >= len(ev.Pointers) {
		return nil, true
	}
	// ACTION_HOVER_MOVE and ACTION_SCROLL always apply to a single pointer only.
	return p.addPointerForIndex(nil, ev, ev.Pointers[ev.ActionIndex], pointerChangeForAction(ev.ActionMasked), 0, hScrollFactor, vScrollFactor), true
}

// uniquePointerIDByType gives each pair of pointer ID and tool type its own ID: some screen
// mirroring tools report the same ID with different tool types in different events, which breaks
// the framework's handling of pointers.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.uniquePointerIdByType
func uniquePointerIDByType(ptr motionPointer) int {
	return ptr.ID<<toolTypeBits | ptr.ToolType&toolTypeMask
}

// addPointerForIndex appends the data of one pointer of the event, if its change has data.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.addPointerForIndex
func (p *touchProcessor) addPointerForIndex(out []ui.PointerData, ev motionEvent, ptr motionPointer, pointerChange ui.PointerChange, pointerData int, hScrollFactor, vScrollFactor float64) []ui.PointerData {
	if pointerChange == ui.PointerChangeUnspecified {
		return out
	}
	pointerID := uniquePointerIDByType(ptr)
	pointerKind := pointerDeviceKindForToolType(ptr.ToolType)

	var buttons int
	switch pointerKind {
	case ui.PointerDeviceKindMouse:
		buttons = ev.ButtonState & 0x1F
		if buttons == 0 && ev.Source == sourceMouse && pointerChange == ui.PointerChangeDown {
			// Some implementations translate trackpad scrolling into a mouse down-move-up
			// sequence with buttons 0, such as ARC on a Chromebook.
			p.ongoingPans[pointerID] = [2]float64{ptr.X, ptr.Y}
		}
	case ui.PointerDeviceKindStylus:
		// The Android button state in the framework's normalized state.
		buttons = (ev.ButtonState >> 4) & 0xF
	}

	panStart, isTrackpadPan := p.ongoingPans[pointerID]
	panZoomType := ui.PointerChangeUnspecified
	if isTrackpadPan {
		panZoomType = pointerChangeForPanZoom(pointerChange)
		if panZoomType == ui.PointerChangeUnspecified {
			return out
		}
	}

	signalKind := ui.PointerSignalKindNone
	if ev.ActionMasked == actionScroll {
		signalKind = ui.PointerSignalKindScroll
	}

	pressureMin, pressureMax := ev.PressureMin, ev.PressureMax
	if pressureMin == 0 && pressureMax == 0 {
		pressureMin, pressureMax = 0, 1
	}

	d := ui.PointerData{
		ViewID:            ui.ImplicitViewID,
		TimeStamp:         time.Duration(ev.EventTime) * time.Millisecond,
		Change:            pointerChange,
		Kind:              pointerKind,
		SignalKind:        signalKind,
		Device:            pointerID,
		PointerIdentifier: 0, // generated by the pointerConverter
		PhysicalX:         ptr.X,
		PhysicalY:         ptr.Y,
		Buttons:           buttons,
		Pressure:          ptr.Pressure,
		PressureMin:       pressureMin,
		PressureMax:       pressureMax,
		Size:              ptr.Size,
		RadiusMajor:       ptr.ToolMajor,
		RadiusMinor:       ptr.ToolMinor,
		Orientation:       ptr.Orientation,
		PlatformData:      pointerData,
		Scale:             1,
	}
	if isTrackpadPan {
		d.Change = panZoomType
		d.Kind = ui.PointerDeviceKindTrackpad
		d.PhysicalX, d.PhysicalY = panStart[0], panStart[1]
		d.PanX, d.PanY = ptr.X-panStart[0], ptr.Y-panStart[1]
	}
	if pointerKind == ui.PointerDeviceKindStylus {
		d.Distance = ptr.Distance
		d.Tilt = ptr.Tilt
	}
	if signalKind == ui.PointerSignalKindScroll {
		// The sign of the scroll value is flipped because it aligns the pixel value with the
		// scroll direction in native Android.
		d.ScrollDeltaX = hScrollFactor * -ptr.HScroll
		d.ScrollDeltaY = vScrollFactor * -ptr.VScroll
	}
	if isTrackpadPan && panZoomType == ui.PointerChangePanZoomEnd {
		delete(p.ongoingPans, pointerID)
	}
	return append(out, d)
}

// pointerChangeForAction is the change of a MotionEvent action; PointerChangeUnspecified when
// the action has none.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.getPointerChangeForAction
func pointerChangeForAction(maskedAction int) ui.PointerChange {
	switch maskedAction {
	case actionDown, actionPointerDown:
		return ui.PointerChangeDown
	case actionUp, actionPointerUp:
		return ui.PointerChangeUp
	case actionMove:
		return ui.PointerChangeMove
	case actionHoverMove, actionScroll:
		return ui.PointerChangeHover
	case actionCancel:
		return ui.PointerChangeCancel
	}
	return ui.PointerChangeUnspecified
}

// pointerChangeForPanZoom is the change of a trackpad pan for the change of the mouse event that
// stands for it; PointerChangeUnspecified when the pan has none (a hover).
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.getPointerChangeForPanZoom
func pointerChangeForPanZoom(pointerChange ui.PointerChange) ui.PointerChange {
	switch pointerChange {
	case ui.PointerChangeDown:
		return ui.PointerChangePanZoomStart
	case ui.PointerChangeMove:
		return ui.PointerChangePanZoomUpdate
	case ui.PointerChangeUp, ui.PointerChangeCancel:
		return ui.PointerChangePanZoomEnd
	}
	return ui.PointerChangeUnspecified
}

// pointerDeviceKindForToolType is the kind of device of a MotionEvent tool type.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java AndroidTouchProcessor.getPointerDeviceTypeForToolType
func pointerDeviceKindForToolType(toolType int) ui.PointerDeviceKind {
	switch toolType {
	case toolTypeFinger:
		return ui.PointerDeviceKindTouch
	case toolTypeStylus:
		return ui.PointerDeviceKindStylus
	case toolTypeMouse:
		return ui.PointerDeviceKindMouse
	case toolTypeEraser:
		return ui.PointerDeviceKindInvertedStylus
	}
	// toolTypeUnknown reaches here.
	return ui.PointerDeviceKindUnknown
}
