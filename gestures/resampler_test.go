package gestures

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/gestures/resampler_test.dart.

func createSimulatedPointerAddedEvent(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerAddedEvent(PointerEventOpts{TimeStamp: us(timeStampUs), Position: ui.Offset{Dx: x, Dy: y}})
}

func createSimulatedPointerRemovedEvent(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerRemovedEvent(PointerEventOpts{TimeStamp: us(timeStampUs), Position: ui.Offset{Dx: x, Dy: y}})
}

func createSimulatedPointerDownEvent(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerDownEvent(PointerEventOpts{TimeStamp: us(timeStampUs), Position: ui.Offset{Dx: x, Dy: y}})
}

func createSimulatedPointerMoveEvent(timeStampUs int, x, y, deltaX, deltaY float64) PointerEvent {
	return NewPointerMoveEvent(PointerEventOpts{
		TimeStamp: us(timeStampUs), Position: ui.Offset{Dx: x, Dy: y}, Delta: ui.Offset{Dx: deltaX, Dy: deltaY},
	})
}

func createSimulatedPointerHoverEvent(timeStampUs int, x, y, deltaX, deltaY float64) PointerEvent {
	return NewPointerHoverEvent(PointerEventOpts{
		TimeStamp: us(timeStampUs), Position: ui.Offset{Dx: x, Dy: y}, Delta: ui.Offset{Dx: deltaX, Dy: deltaY},
	})
}

func createSimulatedPointerUpEvent(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerUpEvent(PointerEventOpts{TimeStamp: us(timeStampUs), Position: ui.Offset{Dx: x, Dy: y}})
}

func wantLen(t *testing.T, result []PointerEvent, n int) {
	t.Helper()
	if len(result) != n {
		t.Fatalf("got %d events, want %d: %v", len(result), n, typeNames(result))
	}
}

func wantTime(t *testing.T, result []PointerEvent, i int, want time.Duration) {
	t.Helper()
	if got := result[i].TimeStamp(); got != want {
		t.Errorf("result[%d].TimeStamp() = %v, want %v", i, got, want)
	}
}

func wantType(t *testing.T, result []PointerEvent, i int, want string) {
	t.Helper()
	if got := foundation.RuntimeTypeName(result[i]); got != want {
		t.Errorf("result[%d] is %s, want %s", i, got, want)
	}
}

func wantFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func TestResamplerBasic(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 50.0)
	event1 := createSimulatedPointerHoverEvent(2000, 10.0, 40.0, 10.0, -10.0)
	event2 := createSimulatedPointerDownEvent(2000, 10.0, 40.0)
	event3 := createSimulatedPointerMoveEvent(3000, 20.0, 30.0, 10.0, -10.0)
	event4 := createSimulatedPointerMoveEvent(4000, 30.0, 20.0, 10.0, -10.0)
	event5 := createSimulatedPointerUpEvent(4000, 30.0, 20.0)
	event6 := createSimulatedPointerHoverEvent(5000, 40.0, 10.0, 10.0, -10.0)
	event7 := createSimulatedPointerHoverEvent(6000, 50.0, 0.0, 10.0, -10.0)
	event8 := createSimulatedPointerRemovedEvent(6000, 50.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)
	resampler.AddEvent(event6)
	resampler.AddEvent(event7)
	resampler.AddEvent(event8)

	resampler.Sample(us(500), 0, add)

	// No pointer event should have been returned yet.
	wantLen(t, result, 0)

	resampler.Sample(us(1500), 0, add)

	// Add pointer event should have been returned.
	wantLen(t, result, 1)
	wantTime(t, result, 0, us(1500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 5.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 45.0)

	resampler.Sample(us(2500), 0, add)

	// Hover and down pointer events should have been returned.
	wantLen(t, result, 3)
	wantTime(t, result, 1, us(2500))
	wantType(t, result, 1, "PointerHoverEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 15.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 35.0)
	wantFloat(t, "result[1].delta.dx", result[1].Delta().Dx, 10.0)
	wantFloat(t, "result[1].delta.dy", result[1].Delta().Dy, -10.0)
	wantTime(t, result, 2, us(2500))
	wantType(t, result, 2, "PointerDownEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 15.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 35.0)

	resampler.Sample(us(3500), 0, add)

	// Move pointer event should have been returned.
	wantLen(t, result, 4)
	wantTime(t, result, 3, us(3500))
	wantType(t, result, 3, "PointerMoveEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 25.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 25.0)
	wantFloat(t, "result[3].delta.dx", result[3].Delta().Dx, 10.0)
	wantFloat(t, "result[3].delta.dy", result[3].Delta().Dy, -10.0)

	resampler.Sample(us(4500), 0, add)

	// Move and up pointer events should have been returned.
	wantLen(t, result, 6)
	wantTime(t, result, 4, us(4500))
	wantType(t, result, 4, "PointerMoveEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 35.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 15.0)
	wantFloat(t, "result[4].delta.dx", result[4].Delta().Dx, 10.0)
	wantFloat(t, "result[4].delta.dy", result[4].Delta().Dy, -10.0)
	// buttons field needs to be a valid value
	if result[4].Buttons() != PrimaryButton {
		t.Errorf("result[4].Buttons() = %d, want PrimaryButton", result[4].Buttons())
	}
	wantTime(t, result, 5, us(4500))
	wantType(t, result, 5, "PointerUpEvent")
	wantFloat(t, "result[5].position.dx", result[5].Position().Dx, 35.0)
	wantFloat(t, "result[5].position.dy", result[5].Position().Dy, 15.0)

	resampler.Sample(us(5500), 0, add)

	// Hover pointer event should have been returned.
	wantLen(t, result, 7)
	wantTime(t, result, 6, us(5500))
	wantType(t, result, 6, "PointerHoverEvent")
	wantFloat(t, "result[6].position.dx", result[6].Position().Dx, 45.0)
	wantFloat(t, "result[6].position.dy", result[6].Position().Dy, 5.0)
	wantFloat(t, "result[6].delta.dx", result[6].Delta().Dx, 10.0)
	wantFloat(t, "result[6].delta.dy", result[6].Delta().Dy, -10.0)

	resampler.Sample(us(6500), 0, add)

	// Hover and removed pointer events should have been returned.
	wantLen(t, result, 9)
	wantTime(t, result, 7, us(6500))
	wantType(t, result, 7, "PointerHoverEvent")
	wantFloat(t, "result[7].position.dx", result[7].Position().Dx, 50.0)
	wantFloat(t, "result[7].position.dy", result[7].Position().Dy, 0.0)
	wantFloat(t, "result[7].delta.dx", result[7].Delta().Dx, 5.0)
	wantFloat(t, "result[7].delta.dy", result[7].Delta().Dy, -5.0)
	wantTime(t, result, 8, us(6500))
	wantType(t, result, 8, "PointerRemovedEvent")
	wantFloat(t, "result[8].position.dx", result[8].Position().Dx, 50.0)
	wantFloat(t, "result[8].position.dy", result[8].Position().Dy, 0.0)

	resampler.Sample(us(7500), 0, add)

	// No pointer event should have been returned.
	wantLen(t, result, 9)
}

func TestResamplerStream(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 50.0)
	event1 := createSimulatedPointerHoverEvent(2000, 10.0, 40.0, 10.0, -10.0)
	event2 := createSimulatedPointerDownEvent(2000, 10.0, 40.0)
	event3 := createSimulatedPointerMoveEvent(3000, 20.0, 30.0, 10.0, -10.0)
	event4 := createSimulatedPointerMoveEvent(4000, 30.0, 20.0, 10.0, -10.0)
	event5 := createSimulatedPointerUpEvent(4000, 30.0, 20.0)
	event6 := createSimulatedPointerHoverEvent(5000, 40.0, 10.0, 10.0, -10.0)
	event7 := createSimulatedPointerHoverEvent(6000, 50.0, 0.0, 10.0, -10.0)
	event8 := createSimulatedPointerRemovedEvent(6000, 50.0, 0.0)

	resampler.AddEvent(event0)

	//
	// Initial sample time a 0.5 ms.
	//

	resampler.Sample(us(500), 0, add)

	// No pointer event should have been returned yet.
	wantLen(t, result, 0)

	resampler.AddEvent(event1)
	resampler.AddEvent(event2)

	resampler.Sample(us(500), 0, add)

	// No pointer event should have been returned yet.
	wantLen(t, result, 0)

	//
	// Advance sample time to 1.5 ms.
	//

	resampler.Sample(us(1500), 0, add)

	// Added pointer event should have been returned.
	wantLen(t, result, 1)
	wantTime(t, result, 0, us(1500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 5.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 45.0)

	resampler.AddEvent(event3)

	resampler.Sample(us(1500), 0, add)

	// No more pointer events should have been returned.
	wantLen(t, result, 1)

	//
	// Advance sample time to 2.5 ms.
	//

	resampler.Sample(us(2500), 0, add)

	// Hover and down pointer events should have been returned.
	wantLen(t, result, 3)
	wantTime(t, result, 1, us(2500))
	wantType(t, result, 1, "PointerHoverEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 15.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 35.0)
	wantFloat(t, "result[1].delta.dx", result[1].Delta().Dx, 10.0)
	wantFloat(t, "result[1].delta.dy", result[1].Delta().Dy, -10.0)
	wantTime(t, result, 2, us(2500))
	wantType(t, result, 2, "PointerDownEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 15.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 35.0)

	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	resampler.Sample(us(2500), 0, add)

	// No more pointer events should have been returned.
	wantLen(t, result, 3)

	//
	// Advance sample time to 3.5 ms.
	//

	resampler.Sample(us(3500), 0, add)

	// Move pointer event should have been returned.
	wantLen(t, result, 4)
	wantTime(t, result, 3, us(3500))
	wantType(t, result, 3, "PointerMoveEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 25.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 25.0)
	wantFloat(t, "result[3].delta.dx", result[3].Delta().Dx, 10.0)
	wantFloat(t, "result[3].delta.dy", result[3].Delta().Dy, -10.0)

	resampler.AddEvent(event6)

	resampler.Sample(us(3500), 0, add)

	// No more pointer events should have been returned.
	wantLen(t, result, 4)

	//
	// Advance sample time to 4.5 ms.
	//

	resampler.Sample(us(4500), 0, add)

	// Move and up pointer events should have been returned.
	wantLen(t, result, 6)
	wantTime(t, result, 4, us(4500))
	wantType(t, result, 4, "PointerMoveEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 35.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 15.0)
	wantFloat(t, "result[4].delta.dx", result[4].Delta().Dx, 10.0)
	wantFloat(t, "result[4].delta.dy", result[4].Delta().Dy, -10.0)
	wantTime(t, result, 5, us(4500))
	wantType(t, result, 5, "PointerUpEvent")
	wantFloat(t, "result[5].position.dx", result[5].Position().Dx, 35.0)
	wantFloat(t, "result[5].position.dy", result[5].Position().Dy, 15.0)

	resampler.AddEvent(event7)
	resampler.AddEvent(event8)

	resampler.Sample(us(4500), 0, add)

	// No more pointer events should have been returned.
	wantLen(t, result, 6)

	//
	// Advance sample time to 5.5 ms.
	//

	resampler.Sample(us(5500), 0, add)

	// Hover pointer event should have been returned.
	wantLen(t, result, 7)
	wantTime(t, result, 6, us(5500))
	wantType(t, result, 6, "PointerHoverEvent")
	wantFloat(t, "result[6].position.dx", result[6].Position().Dx, 45.0)
	wantFloat(t, "result[6].position.dy", result[6].Position().Dy, 5.0)
	wantFloat(t, "result[6].delta.dx", result[6].Delta().Dx, 10.0)
	wantFloat(t, "result[6].delta.dy", result[6].Delta().Dy, -10.0)

	//
	// Advance sample time to 6.5 ms.
	//

	resampler.Sample(us(6500), 0, add)

	// Hover and removed pointer event should have been returned.
	wantLen(t, result, 9)
	wantTime(t, result, 7, us(6500))
	wantType(t, result, 7, "PointerHoverEvent")
	wantFloat(t, "result[7].position.dx", result[7].Position().Dx, 50.0)
	wantFloat(t, "result[7].position.dy", result[7].Position().Dy, 0.0)
	wantFloat(t, "result[7].delta.dx", result[7].Delta().Dx, 5.0)
	wantFloat(t, "result[7].delta.dy", result[7].Delta().Dy, -5.0)
	wantTime(t, result, 8, us(6500))
	wantType(t, result, 8, "PointerRemovedEvent")
	wantFloat(t, "result[8].position.dx", result[8].Position().Dx, 50.0)
	wantFloat(t, "result[8].position.dy", result[8].Position().Dy, 0.0)

	//
	// Advance sample time to 7.5 ms.
	//

	resampler.Sample(us(7500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 9)
}

func TestResamplerQuickTap(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(1000, 0.0, 0.0)
	event2 := createSimulatedPointerUpEvent(1000, 0.0, 0.0)
	event3 := createSimulatedPointerRemovedEvent(1000, 0.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)

	resampler.Sample(us(1500), 0, add)

	// All pointer events should have been returned.
	wantLen(t, result, 4)
	wantTime(t, result, 0, us(1500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 0.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(1500))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 0.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)
	wantTime(t, result, 2, us(1500))
	wantType(t, result, 2, "PointerUpEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 0.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantTime(t, result, 3, us(1500))
	wantType(t, result, 3, "PointerRemovedEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 0.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)
}

func TestResamplerAdvanceSlowly(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(1000, 0.0, 0.0)
	event2 := createSimulatedPointerMoveEvent(2000, 10.0, 0.0, 10.0, 0.0)
	event3 := createSimulatedPointerMoveEvent(3000, 20.0, 0.0, 10.0, 0.0)
	event4 := createSimulatedPointerUpEvent(3000, 20.0, 0.0)
	event5 := createSimulatedPointerRemovedEvent(3000, 20.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	resampler.Sample(us(1500), 0, add)

	// Added and down pointer events should have been returned.
	wantLen(t, result, 2)
	wantTime(t, result, 0, us(1500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 5.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(1500))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 5.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)

	resampler.Sample(us(1500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 2)

	resampler.Sample(us(1750), 0, add)

	// Move pointer event should have been returned.
	wantLen(t, result, 3)
	wantTime(t, result, 2, us(1750))
	wantType(t, result, 2, "PointerMoveEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 7.5)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantFloat(t, "result[2].delta.dx", result[2].Delta().Dx, 2.5)
	wantFloat(t, "result[2].delta.dy", result[2].Delta().Dy, 0.0)

	resampler.Sample(us(2000), 0, add)

	// Another move pointer event should have been returned.
	wantLen(t, result, 4)
	wantTime(t, result, 3, us(2000))
	wantType(t, result, 3, "PointerMoveEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 10.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)
	wantFloat(t, "result[3].delta.dx", result[3].Delta().Dx, 2.5)
	wantFloat(t, "result[3].delta.dy", result[3].Delta().Dy, 0.0)

	resampler.Sample(us(3000), 0, add)

	// Move, up and removed pointer events should have been returned.
	wantLen(t, result, 7)
	wantTime(t, result, 4, us(3000))
	wantType(t, result, 4, "PointerMoveEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 20.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 0.0)
	wantFloat(t, "result[4].delta.dx", result[4].Delta().Dx, 10.0)
	wantFloat(t, "result[4].delta.dy", result[4].Delta().Dy, 0.0)
	wantTime(t, result, 5, us(3000))
	wantType(t, result, 5, "PointerUpEvent")
	wantFloat(t, "result[5].position.dx", result[5].Position().Dx, 20.0)
	wantFloat(t, "result[5].position.dy", result[5].Position().Dy, 0.0)
	wantTime(t, result, 6, us(3000))
	wantType(t, result, 6, "PointerRemovedEvent")
	wantFloat(t, "result[6].position.dx", result[6].Position().Dx, 20.0)
	wantFloat(t, "result[6].position.dy", result[6].Position().Dy, 0.0)
}

func TestResamplerAdvanceFast(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(1000, 0.0, 0.0)
	event2 := createSimulatedPointerMoveEvent(2000, 5.0, 0.0, 5.0, 0.0)
	event3 := createSimulatedPointerMoveEvent(3000, 20.0, 0.0, 15.0, 0.0)
	event4 := createSimulatedPointerUpEvent(4000, 30.0, 0.0)
	event5 := createSimulatedPointerRemovedEvent(4000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	resampler.Sample(us(2500), 0, add)

	// Added and down pointer events should have been returned.
	wantLen(t, result, 2)
	wantTime(t, result, 0, us(2500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 12.5)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(2500))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 12.5)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)

	resampler.Sample(us(5500), 0, add)

	// Move, up and removed pointer events should have been returned.
	wantLen(t, result, 5)
	wantTime(t, result, 2, us(5500))
	wantType(t, result, 2, "PointerMoveEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 30.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantFloat(t, "result[2].delta.dx", result[2].Delta().Dx, 17.5)
	wantFloat(t, "result[2].delta.dy", result[2].Delta().Dy, 0.0)
	wantTime(t, result, 3, us(5500))
	wantType(t, result, 3, "PointerUpEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 30.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)
	wantTime(t, result, 4, us(5500))
	wantType(t, result, 4, "PointerRemovedEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 30.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 0.0)

	resampler.Sample(us(6500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 5)
}

func TestResamplerSkip(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(1000, 0.0, 0.0)
	event2 := createSimulatedPointerMoveEvent(2000, 10.0, 0.0, 10.0, 0.0)
	event3 := createSimulatedPointerUpEvent(3000, 10.0, 0.0)
	event4 := createSimulatedPointerHoverEvent(4000, 20.0, 0.0, 10.0, 0.0)
	event5 := createSimulatedPointerDownEvent(4000, 20.0, 0.0)
	event6 := createSimulatedPointerMoveEvent(5000, 30.0, 0.0, 10.0, 0.0)
	event7 := createSimulatedPointerUpEvent(5000, 30.0, 0.0)
	event8 := createSimulatedPointerRemovedEvent(5000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)
	resampler.AddEvent(event6)
	resampler.AddEvent(event7)
	resampler.AddEvent(event8)

	resampler.Sample(us(1500), 0, add)

	// Added and down pointer events should have been returned.
	wantLen(t, result, 2)
	wantTime(t, result, 0, us(1500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 5.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(1500))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 5.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)

	resampler.Sample(us(5500), 0, add)

	// All remaining pointer events should have been returned.
	wantLen(t, result, 7)
	wantTime(t, result, 2, us(5500))
	wantType(t, result, 2, "PointerMoveEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 30.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantFloat(t, "result[2].delta.dx", result[2].Delta().Dx, 25.0)
	wantFloat(t, "result[2].delta.dy", result[2].Delta().Dy, 0.0)
	wantTime(t, result, 3, us(5500))
	wantType(t, result, 3, "PointerUpEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 30.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)
	wantTime(t, result, 4, us(5500))
	wantType(t, result, 4, "PointerDownEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 30.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 0.0)
	wantTime(t, result, 5, us(5500))
	wantType(t, result, 5, "PointerUpEvent")
	wantFloat(t, "result[5].position.dx", result[5].Position().Dx, 30.0)
	wantFloat(t, "result[5].position.dy", result[5].Position().Dy, 0.0)
	wantTime(t, result, 6, us(5500))
	wantType(t, result, 6, "PointerRemovedEvent")
	wantFloat(t, "result[6].position.dx", result[6].Position().Dx, 30.0)
	wantFloat(t, "result[6].position.dy", result[6].Position().Dy, 0.0)

	resampler.Sample(us(6500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 7)
}

func TestResamplerSkipAll(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(1000, 0.0, 0.0)
	event2 := createSimulatedPointerMoveEvent(4000, 30.0, 0.0, 30.0, 0.0)
	event3 := createSimulatedPointerUpEvent(4000, 30.0, 0.0)
	event4 := createSimulatedPointerRemovedEvent(4000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)

	resampler.Sample(us(500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 0)

	resampler.Sample(us(5500), 0, add)

	// All remaining pointer events should have been returned.
	wantLen(t, result, 4)
	wantTime(t, result, 0, us(5500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 30.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(5500))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 30.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)
	wantTime(t, result, 2, us(5500))
	wantType(t, result, 2, "PointerUpEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 30.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantTime(t, result, 3, us(5500))
	wantType(t, result, 3, "PointerRemovedEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 30.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)

	resampler.Sample(us(6500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 4)
}

func TestResamplerStop(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(2000, 0.0, 0.0)
	event2 := createSimulatedPointerMoveEvent(3000, 10.0, 0.0, 10.0, 0.0)
	event3 := createSimulatedPointerMoveEvent(4000, 20.0, 0.0, 10.0, 0.0)
	event4 := createSimulatedPointerUpEvent(4000, 20.0, 0.0)
	event5 := createSimulatedPointerRemovedEvent(5000, 20.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	resampler.Sample(us(500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 0)

	resampler.Stop(add)

	// All pointer events should have been returned with original
	// time stamps and positions.
	wantLen(t, result, 6)
	wantTime(t, result, 0, us(1000))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 0.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(2000))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 0.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)
	wantTime(t, result, 2, us(3000))
	wantType(t, result, 2, "PointerMoveEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 10.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantFloat(t, "result[2].delta.dx", result[2].Delta().Dx, 10.0)
	wantFloat(t, "result[2].delta.dy", result[2].Delta().Dy, 0.0)
	wantTime(t, result, 3, us(4000))
	wantType(t, result, 3, "PointerMoveEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 20.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)
	wantFloat(t, "result[3].delta.dx", result[3].Delta().Dx, 10.0)
	wantFloat(t, "result[3].delta.dy", result[3].Delta().Dy, 0.0)
	wantTime(t, result, 4, us(4000))
	wantType(t, result, 4, "PointerUpEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 20.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 0.0)
	wantTime(t, result, 5, us(5000))
	wantType(t, result, 5, "PointerRemovedEvent")
	wantFloat(t, "result[5].position.dx", result[5].Position().Dx, 20.0)
	wantFloat(t, "result[5].position.dy", result[5].Position().Dy, 0.0)

	resampler.Sample(us(10000), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 6)
}

func TestResamplerSyntheticMove(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(2000, 0.0, 0.0)
	event2 := createSimulatedPointerMoveEvent(3000, 10.0, 0.0, 10.0, 0.0)
	event3 := createSimulatedPointerUpEvent(4000, 10.0, 0.0)
	event4 := createSimulatedPointerRemovedEvent(5000, 10.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)

	resampler.Sample(us(500), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 0)

	resampler.Sample(us(2000), 0, add)

	// Added and down pointer events should have been returned.
	wantLen(t, result, 2)
	wantTime(t, result, 0, us(2000))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 0.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(2000))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 0.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)

	resampler.Sample(us(5000), 0, add)

	// All remaining pointer events and a synthetic move event should
	// have been returned.
	wantLen(t, result, 5)
	wantTime(t, result, 2, us(5000))
	wantType(t, result, 2, "PointerMoveEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 10.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantFloat(t, "result[2].delta.dx", result[2].Delta().Dx, 10.0)
	wantFloat(t, "result[2].delta.dy", result[2].Delta().Dy, 0.0)
	wantTime(t, result, 3, us(5000))
	wantType(t, result, 3, "PointerUpEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 10.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)
	wantTime(t, result, 4, us(5000))
	wantType(t, result, 4, "PointerRemovedEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 10.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 0.0)

	resampler.Sample(us(10000), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 5)
}

func TestResamplerNextSampleTime(t *testing.T) {
	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }
	_ = add
	resampler := NewPointerEventResampler()
	event0 := createSimulatedPointerAddedEvent(1000, 0.0, 0.0)
	event1 := createSimulatedPointerDownEvent(1000, 0.0, 0.0)
	event2 := createSimulatedPointerMoveEvent(2000, 10.0, 0.0, 10.0, 0.0)
	event3 := createSimulatedPointerMoveEvent(3000, 20.0, 0.0, 10.0, 0.0)
	event4 := createSimulatedPointerUpEvent(3000, 20.0, 0.0)
	event5 := createSimulatedPointerHoverEvent(4000, 30.0, 0.0, 10.0, 0.0)
	event6 := createSimulatedPointerRemovedEvent(4000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)
	resampler.AddEvent(event6)

	sampleTime := us(500)
	nextSampleTime := us(1500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// No pointer events should have been returned.
	wantLen(t, result, 0)

	sampleTime = nextSampleTime
	nextSampleTime = us(2500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// Added and down pointer events should have been returned.
	wantLen(t, result, 2)
	wantTime(t, result, 0, us(1500))
	wantType(t, result, 0, "PointerAddedEvent")
	wantFloat(t, "result[0].position.dx", result[0].Position().Dx, 5.0)
	wantFloat(t, "result[0].position.dy", result[0].Position().Dy, 0.0)
	wantTime(t, result, 1, us(1500))
	wantType(t, result, 1, "PointerDownEvent")
	wantFloat(t, "result[1].position.dx", result[1].Position().Dx, 5.0)
	wantFloat(t, "result[1].position.dy", result[1].Position().Dy, 0.0)

	sampleTime = nextSampleTime
	nextSampleTime = us(3500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// Move and up pointer events should have been returned.
	wantLen(t, result, 4)
	wantTime(t, result, 2, us(2500))
	wantType(t, result, 2, "PointerMoveEvent")
	wantFloat(t, "result[2].position.dx", result[2].Position().Dx, 15.0)
	wantFloat(t, "result[2].position.dy", result[2].Position().Dy, 0.0)
	wantFloat(t, "result[2].delta.dx", result[2].Delta().Dx, 10.0)
	wantFloat(t, "result[2].delta.dy", result[2].Delta().Dy, 0.0)
	wantTime(t, result, 3, us(2500))
	wantType(t, result, 3, "PointerUpEvent")
	wantFloat(t, "result[3].position.dx", result[3].Position().Dx, 15.0)
	wantFloat(t, "result[3].position.dy", result[3].Position().Dy, 0.0)

	sampleTime = nextSampleTime
	nextSampleTime = us(4500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// All remaining pointer events should have been returned.
	wantLen(t, result, 6)
	wantTime(t, result, 4, us(3500))
	wantType(t, result, 4, "PointerHoverEvent")
	wantFloat(t, "result[4].position.dx", result[4].Position().Dx, 25.0)
	wantFloat(t, "result[4].position.dy", result[4].Position().Dy, 0.0)
	wantFloat(t, "result[4].delta.dx", result[4].Delta().Dx, 10.0)
	wantFloat(t, "result[4].delta.dy", result[4].Delta().Dy, 0.0)
	wantTime(t, result, 5, us(3500))
	wantType(t, result, 5, "PointerRemovedEvent")
	wantFloat(t, "result[5].position.dx", result[5].Position().Dx, 25.0)
	wantFloat(t, "result[5].position.dy", result[5].Position().Dy, 0.0)

	resampler.Sample(us(10000), 0, add)

	// No pointer events should have been returned.
	wantLen(t, result, 6)
}
