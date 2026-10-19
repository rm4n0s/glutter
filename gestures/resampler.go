package gestures

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// HandleEventCallback receives the events a resampler makes.
//
// Flutter: gestures/resampler.dart HandleEventCallback
type HandleEventCallback func(event PointerEvent)

// PointerEventResampler resamples the events of one touch device to the times
// the app wants to see them (the frame times): it interpolates the position
// between the events, and holds up and removed events until their time comes.
//
// Flutter: gestures/resampler.dart PointerEventResampler
type PointerEventResampler struct {
	queuedEvents      []PointerEvent
	last              PointerEvent
	next              PointerEvent
	position          ui.Offset
	isTracked         bool
	isDown            bool
	pointerIdentifier int
	hasButtons        int
}

// NewPointerEventResampler creates a resampler without events.
//
// Flutter: gestures/resampler.dart PointerEventResampler.new
func NewPointerEventResampler() *PointerEventResampler { return &PointerEventResampler{} }

func toHoverEvent(event PointerEvent, position, delta ui.Offset, timeStamp time.Duration) PointerEvent {
	return NewPointerHoverEvent(PointerEventOpts{
		ViewID:      event.ViewID(),
		TimeStamp:   timeStamp,
		Kind:        event.Kind(),
		Device:      event.Device(),
		Position:    position,
		Delta:       delta,
		Buttons:     foundation.Some(event.Buttons()),
		Obscured:    event.Obscured(),
		PressureMin: foundation.Some(event.PressureMin()),
		PressureMax: foundation.Some(event.PressureMax()),
		Distance:    event.Distance(),
		DistanceMax: event.DistanceMax(),
		Size:        event.Size(),
		RadiusMajor: event.RadiusMajor(),
		RadiusMinor: event.RadiusMinor(),
		RadiusMin:   event.RadiusMin(),
		RadiusMax:   event.RadiusMax(),
		Orientation: event.Orientation(),
		Tilt:        event.Tilt(),
		Synthesized: event.Synthesized(),
		EmbedderID:  event.EmbedderID(),
	})
}

func toMoveEvent(event PointerEvent, position, delta ui.Offset, pointerIdentifier int, timeStamp time.Duration, buttons int) PointerEvent {
	return NewPointerMoveEvent(PointerEventOpts{
		ViewID:       event.ViewID(),
		TimeStamp:    timeStamp,
		Pointer:      pointerIdentifier,
		Kind:         event.Kind(),
		Device:       event.Device(),
		Position:     position,
		Delta:        delta,
		Buttons:      foundation.Some(buttons),
		Obscured:     event.Obscured(),
		Pressure:     foundation.Some(event.Pressure()),
		PressureMin:  foundation.Some(event.PressureMin()),
		PressureMax:  foundation.Some(event.PressureMax()),
		DistanceMax:  event.DistanceMax(),
		Size:         event.Size(),
		RadiusMajor:  event.RadiusMajor(),
		RadiusMinor:  event.RadiusMinor(),
		RadiusMin:    event.RadiusMin(),
		RadiusMax:    event.RadiusMax(),
		Orientation:  event.Orientation(),
		Tilt:         event.Tilt(),
		PlatformData: event.PlatformData(),
		Synthesized:  event.Synthesized(),
		EmbedderID:   event.EmbedderID(),
	})
}

func toMoveOrHoverEvent(event PointerEvent, position, delta ui.Offset, pointerIdentifier int, timeStamp time.Duration, isDown bool, buttons int) PointerEvent {
	if isDown {
		return toMoveEvent(event, position, delta, pointerIdentifier, timeStamp, buttons)
	}
	return toHoverEvent(event, position, delta, timeStamp)
}

func (r *PointerEventResampler) positionAt(sampleTime time.Duration) ui.Offset {
	var x, y float64
	var nextTimeStamp, lastTimeStamp time.Duration
	if r.next != nil {
		x, y = r.next.Position().Dx, r.next.Position().Dy
		nextTimeStamp = r.next.TimeStamp()
	}
	if r.last != nil {
		lastTimeStamp = r.last.TimeStamp()
	}
	if nextTimeStamp > sampleTime && nextTimeStamp > lastTimeStamp {
		interval := float64((nextTimeStamp - lastTimeStamp).Microseconds())
		scalar := float64((sampleTime - lastTimeStamp).Microseconds()) / interval
		var lastX, lastY float64
		if r.last != nil {
			lastX, lastY = r.last.Position().Dx, r.last.Position().Dy
		}
		x = lastX + (x-lastX)*scalar
		y = lastY + (y-lastY)*scalar
	}
	return ui.Offset{Dx: x, Dy: y}
}

func (r *PointerEventResampler) processPointerEvents(sampleTime time.Duration) {
	for _, event := range r.queuedEvents {
		if event.TimeStamp() <= sampleTime || r.last == nil {
			r.last = event
			r.next = event
			continue
		}
		var nextTimeStamp time.Duration
		if r.next != nil {
			nextTimeStamp = r.next.TimeStamp()
		}
		if nextTimeStamp < sampleTime {
			r.next = event
			break
		}
	}
}

func (r *PointerEventResampler) dequeueAndSampleNonHoverOrMovePointerEventsUntil(sampleTime, nextSampleTime time.Duration, callback HandleEventCallback) {
	endTime := sampleTime
	for _, event := range r.queuedEvents {
		if event.TimeStamp() > sampleTime {
			// Scan the queue for events that can be handled at the sample time,
			// up to the next sample time: an up or removed event ends the scan
			// window, and a move or hover event continues it.
			if event.TimeStamp() >= nextSampleTime {
				break
			}
			switch event.(type) {
			case *PointerUpEvent, *PointerRemovedEvent:
				endTime = event.TimeStamp()
				continue
			case *PointerMoveEvent, *PointerHoverEvent:
			default:
				// Any other event ends the scan.
				goto scanned
			}
		}
	}
scanned:
	for len(r.queuedEvents) > 0 {
		event := r.queuedEvents[0]
		if event.TimeStamp() > endTime {
			break
		}
		wasTracked := r.isTracked
		wasDown := r.isDown
		hadButtons := r.hasButtons
		_, removed := event.(*PointerRemovedEvent)
		r.isTracked = !removed
		r.isDown = event.Down()
		r.hasButtons = event.Buttons()
		position := r.positionAt(sampleTime)
		if r.isTracked && !wasTracked {
			r.position = position
		}
		pointerIdentifier := event.Pointer()
		r.pointerIdentifier = pointerIdentifier

		switch event.(type) {
		case *PointerMoveEvent, *PointerHoverEvent:
		default:
			if position != r.position {
				delta := position.Sub(r.position)
				callback(toMoveOrHoverEvent(event, position, delta, r.pointerIdentifier, sampleTime, wasDown, hadButtons))
				r.position = position
			}
			callback(event.CopyWith(PointerEventCopyOpts{
				Position:  foundation.Some(position),
				Delta:     foundation.Some(ui.Offset{}),
				Pointer:   foundation.Some(pointerIdentifier),
				TimeStamp: foundation.Some(sampleTime),
			}))
		}
		r.queuedEvents = r.queuedEvents[1:]
	}
}

func (r *PointerEventResampler) samplePointerPosition(sampleTime time.Duration, callback HandleEventCallback) {
	position := r.positionAt(sampleTime)
	if position != r.position && r.next != nil {
		delta := position.Sub(r.position)
		callback(toMoveOrHoverEvent(r.next, position, delta, r.pointerIdentifier, sampleTime, r.isDown, r.hasButtons))
		r.position = position
	}
}

// AddEvent queues an event to be resampled.
//
// Flutter: gestures/resampler.dart PointerEventResampler.addEvent
func (r *PointerEventResampler) AddEvent(event PointerEvent) {
	r.queuedEvents = append(r.queuedEvents, event)
}

// Sample calls callback with the events at sampleTime. Events that fall
// between sampleTime and nextSampleTime that can't be interpolated, such as an
// up event, are handled together with it.
//
// Flutter: gestures/resampler.dart PointerEventResampler.sample
func (r *PointerEventResampler) Sample(sampleTime, nextSampleTime time.Duration, callback HandleEventCallback) {
	r.processPointerEvents(sampleTime)
	r.dequeueAndSampleNonHoverOrMovePointerEventsUntil(sampleTime, nextSampleTime, callback)
	// Get the position for the sample time, if it is tracked.
	if r.isTracked {
		r.samplePointerPosition(sampleTime, callback)
	}
}

// Stop calls callback with all the queued events, as they are, and resets the
// resampler.
//
// Flutter: gestures/resampler.dart PointerEventResampler.stop
func (r *PointerEventResampler) Stop(callback HandleEventCallback) {
	for len(r.queuedEvents) > 0 {
		event := r.queuedEvents[0]
		r.queuedEvents = r.queuedEvents[1:]
		callback(event)
	}
	r.pointerIdentifier = 0
	r.isDown = false
	r.isTracked = false
	r.position = ui.Offset{}
	r.next = nil
	r.last = nil
}

// HasPendingEvents reports whether events are queued.
//
// Flutter: gestures/resampler.dart PointerEventResampler.hasPendingEvents
func (r *PointerEventResampler) HasPendingEvents() bool { return len(r.queuedEvents) > 0 }

// IsTracked reports whether the pointer is tracked: added and not removed.
//
// Flutter: gestures/resampler.dart PointerEventResampler.isTracked
func (r *PointerEventResampler) IsTracked() bool { return r.isTracked }

// IsDown reports whether the pointer is down.
//
// Flutter: gestures/resampler.dart PointerEventResampler.isDown
func (r *PointerEventResampler) IsDown() bool { return r.isDown }
