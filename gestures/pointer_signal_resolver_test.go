package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/gestures/pointer_signal_resolver_test.dart.

type testPointerSignalListener struct {
	t           *testing.T
	event       PointerSignalEvent
	callbackRan bool
}

func (l *testPointerSignalListener) callback(event PointerSignalEvent) {
	l.t.Helper()
	if event != l.event {
		l.t.Errorf("callback got %v, want %v", event, l.event)
	}
	if l.callbackRan {
		l.t.Error("the callback ran twice")
	}
	l.callbackRan = true
}

type pointerSignalTester struct {
	t        *testing.T
	resolver *PointerSignalResolver
	event    PointerSignalEvent
}

func newPointerSignalTester(t *testing.T) *pointerSignalTester {
	return &pointerSignalTester{t: t, resolver: NewPointerSignalResolver(), event: NewPointerScrollEvent()}
}

func (p *pointerSignalTester) addListener() *testPointerSignalListener {
	l := &testPointerSignalListener{t: p.t, event: p.event}
	p.resolver.Register(p.event, l.callback)
	return l
}

// resolve simulates a new event dispatch cycle by resolving the current event
// and setting a new event to use for future calls.
func (p *pointerSignalTester) resolve() {
	p.resolver.Resolve(p.event)
	p.event = NewPointerScrollEvent()
}

func TestPointerSignalResolver(t *testing.T) {
	t.Run("Resolving with no entries should be a no-op", func(t *testing.T) {
		p := newPointerSignalTester(t)
		p.resolver.Resolve(p.event)
	})
	t.Run("Resolving with no entries should notify engine of no-op", func(t *testing.T) {
		allowedPlatformDefault := false
		p := newPointerSignalTester(t)
		p.event = NewPointerScrollEvent(PointerEventOpts{OnRespond: func(allow bool) { allowedPlatformDefault = allow }})
		p.resolver.Resolve(p.event)
		if !allowedPlatformDefault {
			t.Error("Respond was not called with allowPlatformDefault: true")
		}
	})
	t.Run("First entry should always win", func(t *testing.T) {
		p := newPointerSignalTester(t)
		first, second := p.addListener(), p.addListener()
		p.resolve()
		if !first.callbackRan || second.callbackRan {
			t.Errorf("first ran %v, second ran %v", first.callbackRan, second.callbackRan)
		}
	})
	t.Run("Re-use after resolve should work", func(t *testing.T) {
		p := newPointerSignalTester(t)
		first, second := p.addListener(), p.addListener()
		p.resolve()
		newEventListener := p.addListener()
		p.resolve()
		if !newEventListener.callbackRan {
			t.Error("the listener of the new event did not run")
		}
		if !first.callbackRan || second.callbackRan {
			t.Errorf("first ran %v, second ran %v", first.callbackRan, second.callbackRan)
		}
	})
	t.Run("works with transformed events", func(t *testing.T) {
		resolver := NewPointerSignalResolver()
		originalEvent := NewPointerScrollEvent()
		transformed := originalEvent.Transformed(ui.NewMatrix4TranslationValues(10, 20, 0)).(PointerSignalEvent)
		another := originalEvent.Transformed(ui.NewMatrix4TranslationValues(30, 50, 0)).(PointerSignalEvent)
		if PointerEvent(originalEvent) == PointerEvent(transformed) || transformed.Original() != PointerEvent(originalEvent) {
			t.Error("the transformed event is not a new event made from the original")
		}
		if PointerEvent(originalEvent) == PointerEvent(another) || another.Original() != PointerEvent(originalEvent) {
			t.Error("the other transformed event is not a new event made from the original")
		}
		var events []PointerSignalEvent
		resolver.Register(transformed, func(e PointerSignalEvent) { events = append(events, e) })
		// Registering a second transformed event must not panic.
		resolver.Register(another, func(e PointerSignalEvent) { events = append(events, e) })
		resolver.Resolve(originalEvent)
		if len(events) != 1 || events[0] != transformed {
			t.Errorf("events = %v, want the first transformed event only", events)
		}
	})
}
