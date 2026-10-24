package android

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The constants of the Java tests of AndroidTouchProcessor.
const (
	// tpTime is the time of the mocked events, 2 days in milliseconds.
	tpTime int64 = 172800000
	// tpPressure is the pressure of the mocked events.
	tpPressure = 0.8
	// tpEngineStylusButton is the framework's primary stylus button.
	tpEngineStylusButton = 0x02

	// The InputDevice.SOURCE_* values the tests use.
	tpSourceTouchscreen = 0x00001002
	tpSourceStylus      = 0x00004002
	tpSourceKeyboard    = 0x00000101

	// MotionEvent.ACTION_BUTTON_PRESS, which the touch processor does not handle.
	tpActionButtonPress = 11
	// MotionEvent.BUTTON_STYLUS_PRIMARY.
	tpButtonStylusPrimary = 32

	// The platform data of a move of one pointer, and of two.
	tpMove1 = pointerDataFlagMultiple | 1<<pointerDataMultiplePointerCountShift
	tpMove2 = pointerDataFlagMultiple | 2<<pointerDataMultiplePointerCountShift
)

func tpPtr(id, tool int, x, y float64) motionPointer {
	return motionPointer{ID: id, ToolType: tool, X: x, Y: y, Pressure: tpPressure}
}

func tpEv(action, source, buttons int, ptrs ...motionPointer) motionEvent {
	return motionEvent{ActionMasked: action, Source: source, ButtonState: buttons, EventTime: tpTime, Pointers: ptrs}
}

func (e motionEvent) index(i int) motionEvent { e.ActionIndex = i; return e }

// tpData makes the data the processor must make for a pointer that has the pressure of the mocked
// events and the default pressure range, and the time of the mocked events.
func tpData(change ui.PointerChange, kind ui.PointerDeviceKind, device int, x, y float64, mods ...func(*ui.PointerData)) ui.PointerData {
	d := ui.PointerData{
		ViewID: ui.ImplicitViewID, TimeStamp: time.Duration(tpTime) * time.Millisecond,
		Change: change, Kind: kind, SignalKind: ui.PointerSignalKindNone, Device: device,
		PhysicalX: x, PhysicalY: y, Pressure: tpPressure, PressureMax: 1, Scale: 1,
	}
	for _, m := range mods {
		m(&d)
	}
	return d
}

func tpAt(ms int64) func(*ui.PointerData) {
	return func(d *ui.PointerData) { d.TimeStamp = time.Duration(ms) * time.Millisecond }
}

func tpFlags(n int) func(*ui.PointerData) {
	return func(d *ui.PointerData) { d.PlatformData = n }
}

func tpButtons(n int) func(*ui.PointerData) {
	return func(d *ui.PointerData) { d.Buttons = n }
}

func tpPan(x, y float64) func(*ui.PointerData) {
	return func(d *ui.PointerData) { d.PanX, d.PanY = x, y }
}

func tpScroll(x, y float64) func(*ui.PointerData) {
	return func(d *ui.PointerData) {
		d.SignalKind = ui.PointerSignalKindScroll
		d.ScrollDeltaX, d.ScrollDeltaY = x, y
	}
}

// tpStep is one call of the processor: OnTouchEvent, or OnGenericMotionEvent with the factors.
type tpStep struct {
	ev         motionEvent
	generic    bool
	hFactor    float64
	vFactor    float64
	notHandled bool // for a generic event: it must not be handled
	want       []ui.PointerData
}

func TestTouchProcessor(t *testing.T) {
	finger, stylus, mouse, eraser := toolTypeFinger, toolTypeStylus, toolTypeMouse, toolTypeEraser
	touch, kStylus, kMouse := ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus, ui.PointerDeviceKindMouse
	sizes := func(size, major, minor float64) func(*ui.PointerData) {
		return func(d *ui.PointerData) { d.Size, d.RadiusMajor, d.RadiusMinor = size, major, minor }
	}
	stylusPtr := func(x, y, distance, tilt float64) motionPointer {
		p := tpPtr(0, stylus, x, y)
		p.Distance, p.Tilt = distance, tilt
		return p
	}

	tests := []struct {
		name  string
		steps []tpStep
	}{
		{
			// normalTouch
			name: "NormalTouch",
			steps: []tpStep{
				{
					ev:   tpEv(actionDown, tpSourceTouchscreen, 0, tpPtr(0, finger, 0, 0)),
					want: []ui.PointerData{tpData(ui.PointerChangeDown, touch, 1, 0, 0)},
				},
				{
					ev:   tpEv(actionMove, tpSourceTouchscreen, 0, tpPtr(0, finger, 10, 5)),
					want: []ui.PointerData{tpData(ui.PointerChangeMove, touch, 1, 10, 5, tpFlags(tpMove1))},
				},
				{
					// Two pointer changes: the original UP and the synthesized REMOVE.
					ev: tpEv(actionUp, tpSourceTouchscreen, 0, tpPtr(0, finger, 10, 5)),
					want: []ui.PointerData{
						tpData(ui.PointerChangeUp, touch, 1, 10, 5),
						tpData(ui.PointerChangeRemove, touch, 1, 10, 5),
					},
				},
			},
		},
		{
			// trackpadGesture
			name: "TrackpadGesture",
			steps: []tpStep{
				{
					ev:   tpEv(actionDown, sourceMouse, 0, tpPtr(1, mouse, 0, 0)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomStart, ui.PointerDeviceKindTrackpad, 11, 0, 0)},
				},
				{
					// The position stays the one of the start; the pan is the distance from it. The
					// pan delta is always zero and the scale one.
					ev:   tpEv(actionMove, sourceMouse, 0, tpPtr(1, mouse, 10, 5)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomUpdate, ui.PointerDeviceKindTrackpad, 11, 0, 0, tpPan(10, 5), tpFlags(tpMove1))},
				},
				{
					ev:   tpEv(actionUp, sourceMouse, 0, tpPtr(1, mouse, 10, 5)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomEnd, ui.PointerDeviceKindTrackpad, 11, 0, 0, tpPan(10, 5))},
				},
				{
					// The pan has ended: the next move is a mouse move again.
					ev:   tpEv(actionMove, sourceMouse, 0, tpPtr(1, mouse, 20, 20)),
					want: []ui.PointerData{tpData(ui.PointerChangeMove, kMouse, 11, 20, 20, tpFlags(tpMove1))},
				},
			},
		},
		{
			name: "TrackpadGestureCancelEndsThePan",
			steps: []tpStep{
				{
					ev:   tpEv(actionDown, sourceMouse, 0, tpPtr(1, mouse, 3, 4)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomStart, ui.PointerDeviceKindTrackpad, 11, 3, 4)},
				},
				{
					ev:   tpEv(actionCancel, sourceMouse, 0, tpPtr(1, mouse, 5, 9)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomEnd, ui.PointerDeviceKindTrackpad, 11, 3, 4, tpPan(2, 5), tpFlags(tpMove1))},
				},
				{
					ev:   tpEv(actionCancel, sourceMouse, 0, tpPtr(1, mouse, 5, 9)),
					want: []ui.PointerData{tpData(ui.PointerChangeCancel, kMouse, 11, 5, 9, tpFlags(tpMove1))},
				},
			},
		},
		{
			// mouse: no synthesized remove, and the mouse buttons
			name: "Mouse",
			steps: []tpStep{
				{
					ev:   tpEv(actionDown, sourceMouse, 1, tpPtr(2, mouse, 0, 0)),
					want: []ui.PointerData{tpData(ui.PointerChangeDown, kMouse, 19, 0, 0, tpButtons(1))},
				},
				{
					ev:   tpEv(actionMove, sourceMouse, 1, tpPtr(2, mouse, 10, 5)),
					want: []ui.PointerData{tpData(ui.PointerChangeMove, kMouse, 19, 10, 5, tpButtons(1), tpFlags(tpMove1))},
				},
				{
					// Ensure that no remove is sent when using a mouse.
					ev:   tpEv(actionUp, sourceMouse, 1, tpPtr(2, mouse, 10, 5)),
					want: []ui.PointerData{tpData(ui.PointerChangeUp, kMouse, 19, 10, 5, tpButtons(1))},
				},
			},
		},
		{
			name: "MouseButtonsAreTheLowFiveBits",
			steps: []tpStep{{
				ev:   tpEv(actionMove, sourceClassPointer, 0x3F, tpPtr(0, mouse, 1, 2)),
				want: []ui.PointerData{tpData(ui.PointerChangeMove, kMouse, 3, 1, 2, tpButtons(0x1F), tpFlags(tpMove1))},
			}},
		},
		{
			// unexpectedMaskedAction: ACTION_BUTTON_PRESS is not handled, nothing is dispatched.
			name: "UnexpectedMaskedAction",
			steps: []tpStep{{
				ev:   tpEv(tpActionButtonPress, tpSourceStylus, 0, tpPtr(1, stylus, 0, 0)),
				want: nil,
			}},
		},
		{
			name: "NoPointersAndABadActionIndexGiveNothing",
			steps: []tpStep{
				{ev: tpEv(actionDown, tpSourceTouchscreen, 0), want: nil},
				{ev: tpEv(actionDown, tpSourceTouchscreen, 0, tpPtr(0, finger, 0, 0)).index(1), want: nil},
				{ev: tpEv(actionHoverMove, sourceClassPointer, 0), generic: true, want: nil},
			},
		},
		{
			// scrollWheelAbove26: the factors come from the view configuration
			name: "ScrollWheel",
			steps: []tpStep{{
				ev: tpEv(actionScroll, sourceClassPointer, 1, func() motionPointer {
					p := tpPtr(0, mouse, 0, 0)
					p.HScroll, p.VScroll = -1, 0.5
					return p
				}()),
				generic: true, hFactor: 24, vFactor: 32,
				want: []ui.PointerData{tpData(ui.PointerChangeHover, kMouse, 3, 0, 0, tpButtons(1), tpScroll(24, -16))},
			}},
		},
		{
			// scrollWheelBelow26, the default factors: onTouchEvent has no view configuration
			name: "ScrollWheelWithTheDefaultFactors",
			steps: []tpStep{{
				ev: tpEv(actionScroll, sourceClassPointer, 1, func() motionPointer {
					p := tpPtr(0, mouse, 0, 0)
					p.HScroll, p.VScroll = -1, 0.5
					return p
				}()),
				want: []ui.PointerData{tpData(ui.PointerChangeHover, kMouse, 3, 0, 0, tpButtons(1), tpScroll(48, -24), tpFlags(tpMove1))},
			}},
		},
		{
			// timeStamp: milliseconds become a time stamp, and a scroll through onTouchEvent has
			// the default factors
			name: "TimeStamp",
			steps: []tpStep{{
				ev: tpEv(actionScroll, sourceClassPointer, 1, func() motionPointer {
					p := tpPtr(0, mouse, 1, 1)
					p.HScroll, p.VScroll = 1, 1
					return p
				}()),
				want: []ui.PointerData{tpData(ui.PointerChangeHover, kMouse, 3, 1, 1, tpButtons(1), tpScroll(-48, -48), tpFlags(tpMove1), tpAt(172800000))},
			}},
		},
		{
			// samePointerIdWithDifferentToolTypes_mapsToUniqueDevice
			name: "SamePointerIDWithDifferentToolTypesMapsToUniqueDevices",
			steps: []tpStep{
				{
					ev:   tpEv(actionHoverMove, sourceClassPointer, 1, tpPtr(2, mouse, 1, 1)),
					want: []ui.PointerData{tpData(ui.PointerChangeHover, kMouse, 2<<3|3, 1, 1, tpButtons(1), tpFlags(tpMove1))},
				},
				{
					ev:   tpEv(actionDown, sourceClassPointer, 1, tpPtr(2, finger, 1, 1)),
					want: []ui.PointerData{tpData(ui.PointerChangeDown, touch, 2<<3|1, 1, 1)},
				},
			},
		},
		{
			// physicalXPhysicalY, obscured, synthesized (always false) and pressure (the default
			// range, as the device is unknown)
			name: "PositionObscuredSynthesizedAndPressure",
			steps: []tpStep{{
				ev:   tpEv(actionDown, sourceClassPointer, 0, tpPtr(1, mouse, 10, 20)),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, kMouse, 11, 10, 20)},
			}},
		},
		{
			name: "PressureRangeOfTheDevice",
			steps: []tpStep{{
				ev: func() motionEvent {
					e := tpEv(actionDown, tpSourceTouchscreen, 0, tpPtr(0, finger, 1, 1))
					e.PressureMin, e.PressureMax = 0.25, 2
					return e
				}(),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, touch, 1, 1, 1, func(d *ui.PointerData) {
					d.PressureMin, d.PressureMax = 0.25, 2
				})},
			}},
		},
		{
			// stylusDistance: distance, tilt and the stylus button in the framework's numbering
			name: "StylusDistance",
			steps: []tpStep{{
				ev: tpEv(actionDown, tpSourceStylus, tpButtonStylusPrimary, stylusPtr(0, 0, 10, 20)),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, kStylus, 2, 0, 0, tpButtons(tpEngineStylusButton), func(d *ui.PointerData) {
					d.Distance, d.Tilt = 10, 20
				})},
			}},
		},
		{
			name: "StylusButtonsAreTheNextFourBits",
			steps: []tpStep{{
				ev:   tpEv(actionMove, tpSourceStylus, 0x60|0x1, stylusPtr(0, 0, 0, 0)),
				want: []ui.PointerData{tpData(ui.PointerChangeMove, kStylus, 2, 0, 0, tpButtons(6), tpFlags(tpMove1))},
			}},
		},
		{
			// A touch has neither distance nor tilt nor buttons, whatever the event says.
			name: "OnlyAStylusHasDistanceAndTilt",
			steps: []tpStep{{
				ev: tpEv(actionDown, tpSourceTouchscreen, 0xFF, func() motionPointer {
					p := tpPtr(0, finger, 0, 0)
					p.Distance, p.Tilt, p.Orientation = 3, 4, 0.5
					return p
				}()),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, touch, 1, 0, 0, func(d *ui.PointerData) { d.Orientation = 0.5 })},
			}},
		},
		{
			// sizeAndRadius
			name: "SizeAndRadius",
			steps: []tpStep{{
				ev: tpEv(actionDown, tpSourceStylus, 0, func() motionPointer {
					p := tpPtr(0, stylus, 0, 0)
					p.Size, p.ToolMajor, p.ToolMinor = 10, 20, 30
					return p
				}()),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, kStylus, 2, 0, 0, sizes(10, 20, 30))},
			}},
		},
		{
			// unexpectedPointerChange: a scroll during a trackpad pan makes nothing
			name: "UnexpectedPointerChange",
			steps: []tpStep{
				{
					ev:   tpEv(actionDown, sourceMouse, 0, tpPtr(0, mouse, 0, 0)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomStart, ui.PointerDeviceKindTrackpad, 3, 0, 0)},
				},
				{
					ev:   tpEv(actionMove, sourceMouse, 0, tpPtr(0, mouse, 10, 5)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomUpdate, ui.PointerDeviceKindTrackpad, 3, 0, 0, tpPan(10, 5), tpFlags(tpMove1))},
				},
				{
					ev:      tpEv(actionScroll, sourceMouse, 0, tpPtr(0, mouse, 0, 0)),
					generic: true, hFactor: 48, vFactor: 48,
					want: nil,
				},
				{
					ev:   tpEv(actionUp, sourceMouse, 0, tpPtr(0, mouse, 10, 5)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomEnd, ui.PointerDeviceKindTrackpad, 3, 0, 0, tpPan(10, 5))},
				},
			},
		},
		{
			name: "AMouseDownWithButtonsFromAnotherSourceIsNoPan",
			steps: []tpStep{{
				ev:   tpEv(actionDown, tpSourceTouchscreen, 0, tpPtr(0, mouse, 0, 0)),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, kMouse, 3, 0, 0)},
			}},
		},
		{
			name: "AnEraserIsAnInvertedStylusAndHasNoRemove",
			steps: []tpStep{
				{
					ev:   tpEv(actionDown, tpSourceStylus, 0x20, tpPtr(3, eraser, 1, 2)),
					want: []ui.PointerData{tpData(ui.PointerChangeDown, ui.PointerDeviceKindInvertedStylus, 3<<3|4, 1, 2)},
				},
				{
					ev:   tpEv(actionUp, tpSourceStylus, 0x20, tpPtr(3, eraser, 1, 2)),
					want: []ui.PointerData{tpData(ui.PointerChangeUp, ui.PointerDeviceKindInvertedStylus, 3<<3|4, 1, 2)},
				},
			},
		},
		{
			name: "AnUnknownToolTypeIsAnUnknownDevice",
			steps: []tpStep{{
				ev:   tpEv(actionDown, sourceClassPointer, 0, tpPtr(5, 0, 1, 2)),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, ui.PointerDeviceKindUnknown, 5<<3, 1, 2)},
			}},
		},
		{
			name: "CancelOfATouchHasNoRemove",
			steps: []tpStep{{
				ev:   tpEv(actionCancel, tpSourceTouchscreen, 0, tpPtr(0, finger, 4, 4)),
				want: []ui.PointerData{tpData(ui.PointerChangeCancel, touch, 1, 4, 4, tpFlags(tpMove1))},
			}},
		},
		{
			// The pointer that goes down is the one at the action index only.
			name: "PointerDownIsForTheActionIndexOnly",
			steps: []tpStep{{
				ev:   tpEv(actionPointerDown, tpSourceTouchscreen, 0, tpPtr(0, finger, 1, 1), tpPtr(4, finger, 7, 8)).index(1),
				want: []ui.PointerData{tpData(ui.PointerChangeDown, touch, 4<<3|1, 7, 8)},
			}},
		},
		{
			// The up of a pointer carries the other fingers as batched moves, then the up, then
			// the remove; a pointer that is not a finger is not carried.
			name: "PointerUpCarriesTheOtherFingers",
			steps: []tpStep{{
				ev: tpEv(actionPointerUp, tpSourceTouchscreen, 0,
					tpPtr(0, finger, 1, 1), tpPtr(4, finger, 7, 8), tpPtr(6, stylus, 9, 9), tpPtr(2, finger, 3, 3)).index(1),
				want: []ui.PointerData{
					tpData(ui.PointerChangeMove, touch, 0<<3|1, 1, 1, tpFlags(pointerDataFlagBatched)),
					tpData(ui.PointerChangeMove, touch, 2<<3|1, 3, 3, tpFlags(pointerDataFlagBatched)),
					tpData(ui.PointerChangeUp, touch, 4<<3|1, 7, 8),
					tpData(ui.PointerChangeRemove, touch, 4<<3|1, 7, 8),
				},
			}},
		},
		{
			name: "MoveCarriesAllPointers",
			steps: []tpStep{{
				ev: tpEv(actionMove, tpSourceTouchscreen, 0, tpPtr(0, finger, 1, 1), tpPtr(1, finger, 2, 2)),
				want: []ui.PointerData{
					tpData(ui.PointerChangeMove, touch, 1, 1, 1, tpFlags(tpMove2)),
					tpData(ui.PointerChangeMove, touch, 1<<3|1, 2, 2, tpFlags(tpMove2)),
				},
			}},
		},
		{
			// Deviation from Flutter: the historical samples of a move come first, oldest first, a
			// move per pointer with the time of the sample.
			name: "MoveHistoryComesBeforeTheCurrentSample",
			steps: []tpStep{{
				ev: func() motionEvent {
					e := tpEv(actionMove, tpSourceTouchscreen, 0, tpPtr(0, finger, 4, 4), tpPtr(1, finger, 14, 14))
					older := func(x float64) motionPointer { p := tpPtr(0, finger, x, x); p.Size = 0.5; return p }
					other := func(x float64) motionPointer { return tpPtr(1, finger, x+10, x+10) }
					e.History = []motionSample{
						{Time: tpTime - 16, Pointers: []motionPointer{older(2), other(2)}},
						{Time: tpTime - 8, Pointers: []motionPointer{older(3), other(3)}},
					}
					return e
				}(),
				want: []ui.PointerData{
					tpData(ui.PointerChangeMove, touch, 1, 2, 2, tpFlags(tpMove2), tpAt(tpTime-16), func(d *ui.PointerData) { d.Size = 0.5 }),
					tpData(ui.PointerChangeMove, touch, 1<<3|1, 12, 12, tpFlags(tpMove2), tpAt(tpTime-16)),
					tpData(ui.PointerChangeMove, touch, 1, 3, 3, tpFlags(tpMove2), tpAt(tpTime-8), func(d *ui.PointerData) { d.Size = 0.5 }),
					tpData(ui.PointerChangeMove, touch, 1<<3|1, 13, 13, tpFlags(tpMove2), tpAt(tpTime-8)),
					tpData(ui.PointerChangeMove, touch, 1, 4, 4, tpFlags(tpMove2)),
					tpData(ui.PointerChangeMove, touch, 1<<3|1, 14, 14, tpFlags(tpMove2)),
				},
			}},
		},
		{
			// A history sample has the identity of the pointer of the event at its index, and only
			// a move has history.
			name: "HistoryOfAMoveTakesTheIdentityOfTheEvent",
			steps: []tpStep{
				{
					ev: func() motionEvent {
						e := tpEv(actionMove, tpSourceTouchscreen, 0, tpPtr(3, finger, 4, 4))
						e.History = []motionSample{{Time: tpTime - 5, Pointers: []motionPointer{{X: 1, Y: 1}}}}
						return e
					}(),
					want: []ui.PointerData{
						tpData(ui.PointerChangeMove, touch, 3<<3|1, 1, 1, tpFlags(tpMove1), tpAt(tpTime-5), func(d *ui.PointerData) { d.Pressure = 0 }),
						tpData(ui.PointerChangeMove, touch, 3<<3|1, 4, 4, tpFlags(tpMove1)),
					},
				},
				{
					ev: func() motionEvent {
						e := tpEv(actionCancel, tpSourceTouchscreen, 0, tpPtr(3, finger, 4, 4))
						e.History = []motionSample{{Time: tpTime - 5, Pointers: []motionPointer{tpPtr(3, finger, 1, 1)}}}
						return e
					}(),
					want: []ui.PointerData{tpData(ui.PointerChangeCancel, touch, 3<<3|1, 4, 4, tpFlags(tpMove1))},
				},
			},
		},
		{
			name: "HistoryOfATrackpadPanIsPanUpdates",
			steps: []tpStep{
				{
					ev:   tpEv(actionDown, sourceMouse, 0, tpPtr(0, mouse, 10, 10)),
					want: []ui.PointerData{tpData(ui.PointerChangePanZoomStart, ui.PointerDeviceKindTrackpad, 3, 10, 10)},
				},
				{
					ev: func() motionEvent {
						e := tpEv(actionMove, sourceMouse, 0, tpPtr(0, mouse, 13, 14))
						e.History = []motionSample{{Time: tpTime - 4, Pointers: []motionPointer{tpPtr(0, mouse, 11, 12)}}}
						return e
					}(),
					want: []ui.PointerData{
						tpData(ui.PointerChangePanZoomUpdate, ui.PointerDeviceKindTrackpad, 3, 10, 10, tpFlags(tpMove1), tpAt(tpTime-4), tpPan(1, 2)),
						tpData(ui.PointerChangePanZoomUpdate, ui.PointerDeviceKindTrackpad, 3, 10, 10, tpFlags(tpMove1), tpPan(3, 4)),
					},
				},
			},
		},
		{
			name: "HoverMove",
			steps: []tpStep{{
				ev:      tpEv(actionHoverMove, sourceClassPointer, 0, tpPtr(0, mouse, 6, 7)),
				generic: true,
				want:    []ui.PointerData{tpData(ui.PointerChangeHover, kMouse, 3, 6, 7)},
			}},
		},
		{
			name: "AGenericEventThatIsNotFromAPointerIsNotHandled",
			steps: []tpStep{{
				ev:      tpEv(actionHoverMove, tpSourceKeyboard, 0, tpPtr(0, mouse, 6, 7)),
				generic: true, notHandled: true,
				want: nil,
			}},
		},
		{
			name: "AGenericEventThatIsNeitherHoverNorScrollIsNotHandled",
			steps: []tpStep{{
				ev:      tpEv(actionMove, sourceClassPointer, 0, tpPtr(0, mouse, 6, 7)),
				generic: true, notHandled: true,
				want: nil,
			}},
		},
		{
			// A touch screen event is from a pointer source too.
			name: "AGenericEventOfATouchScreenIsHandled",
			steps: []tpStep{{
				ev:      tpEv(actionHoverMove, tpSourceTouchscreen, 0, tpPtr(0, finger, 6, 7)),
				generic: true,
				want:    []ui.PointerData{tpData(ui.PointerChangeHover, touch, 1, 6, 7)},
			}},
		},
		{
			// Only the pointer at the action index of a generic event is converted.
			name: "AGenericEventConvertsTheActionIndexOnly",
			steps: []tpStep{{
				ev:      tpEv(actionHoverMove, sourceClassPointer, 0, tpPtr(0, mouse, 6, 7), tpPtr(1, mouse, 8, 9)).index(1),
				generic: true,
				want:    []ui.PointerData{tpData(ui.PointerChangeHover, kMouse, 1<<3|3, 8, 9)},
			}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newTouchProcessor()
			for i, s := range tc.steps {
				var got []ui.PointerData
				if s.generic {
					var handled bool
					got, handled = p.OnGenericMotionEvent(s.ev, s.hFactor, s.vFactor)
					if handled == s.notHandled {
						t.Errorf("step %d: handled = %v, want %v", i, handled, !s.notHandled)
					}
				} else {
					got = p.OnTouchEvent(s.ev)
				}
				if len(got) != len(s.want) {
					t.Fatalf("step %d: got %d data, want %d:\n got  %+v\n want %+v", i, len(got), len(s.want), got, s.want)
				}
				for j := range s.want {
					if !reflect.DeepEqual(got[j], s.want[j]) {
						t.Errorf("step %d, data %d:\n got  %+v\n want %+v", i, j, got[j], s.want[j])
					}
				}
			}
		})
	}
}

// TestTouchProcessorPansAreKeptByDevice checks that the pans of two mouse devices are kept apart,
// and that a pan that was never ended can start again.
func TestTouchProcessorPansAreKeptByDevice(t *testing.T) {
	p := newTouchProcessor()
	p.OnTouchEvent(tpEv(actionDown, sourceMouse, 0, tpPtr(1, toolTypeMouse, 1, 1)))
	if got := p.ongoingPans[1<<3|toolTypeMouse]; got != [2]float64{1, 1} {
		t.Fatalf("pan of mouse 1 = %v, want [1 1]", got)
	}
	// Another mouse pointer with buttons is not a pan.
	got := p.OnTouchEvent(tpEv(actionDown, sourceMouse, 1, tpPtr(2, toolTypeMouse, 5, 5)))
	if len(got) != 1 || got[0].Change != ui.PointerChangeDown || got[0].Kind != ui.PointerDeviceKindMouse {
		t.Fatalf("down of mouse 2 = %+v, want a mouse down", got)
	}
	// The first pointer goes down again without having ended: the pan starts again, at the new place.
	got = p.OnTouchEvent(tpEv(actionDown, sourceMouse, 0, tpPtr(1, toolTypeMouse, 9, 9)))
	if len(got) != 1 || got[0].Change != ui.PointerChangePanZoomStart || got[0].PhysicalX != 9 {
		t.Fatalf("second down of mouse 1 = %+v, want a pan start at 9", got)
	}
	p.OnTouchEvent(tpEv(actionUp, sourceMouse, 0, tpPtr(1, toolTypeMouse, 9, 9)))
	if len(p.ongoingPans) != 0 {
		t.Fatalf("ongoingPans = %v, want none after the end", p.ongoingPans)
	}
}

// TestTouchProcessorThroughTheConverter runs what the processor makes through the converter, as
// the platform does: a touch gets its identifier, the remove after the up, and an add before the
// next contact.
func TestTouchProcessorThroughTheConverter(t *testing.T) {
	p, c := newTouchProcessor(), newPointerConverter()
	var out []ui.PointerData
	feed := func(ev motionEvent) { out = append(out, c.Convert(p.OnTouchEvent(ev))...) }
	feed(tpEv(actionDown, tpSourceTouchscreen, 0, tpPtr(0, toolTypeFinger, 5, 6)))
	feed(tpEv(actionMove, tpSourceTouchscreen, 0, tpPtr(0, toolTypeFinger, 8, 6)))
	feed(tpEv(actionUp, tpSourceTouchscreen, 0, tpPtr(0, toolTypeFinger, 8, 6)))
	feed(tpEv(actionDown, tpSourceTouchscreen, 0, tpPtr(0, toolTypeFinger, 1, 1)))
	w := func(change ui.PointerChange, pid int, synth bool, x, y, dx, dy float64) pcWant {
		return pcw(change, 1, pid, synth, x, y, dx, dy, 0).withPan(0, 0, 0, 0, 1)
	}
	pcCheck(t, out, []pcWant{
		w(pcAdd, 0, true, 5, 6, 0, 0),
		w(pcDown, 1, false, 5, 6, 0, 0),
		w(pcMove, 1, false, 8, 6, 3, 0),
		w(pcUp, 1, false, 8, 6, 0, 0),
		w(pcRemove, 0, false, 8, 6, 0, 0),
		w(pcAdd, 0, true, 1, 1, 0, 0),
		w(pcDown, 2, false, 1, 1, 0, 0),
	})
}
