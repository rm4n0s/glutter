package services

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
)

// recordedCursor is a MouseCursor that logs its sessions.
type recordedCursor struct {
	name string
	log  *[]string
}

func (c recordedCursor) CreateSession(device int) MouseCursorSession {
	return &recordedSession{c: c, device: device}
}
func (c recordedCursor) DebugDescription() string { return c.name }

type recordedSession struct {
	c      recordedCursor
	device int
}

func (s *recordedSession) Cursor() MouseCursor { return s.c }
func (s *recordedSession) Device() int         { return s.device }
func (s *recordedSession) Activate()           { *s.c.log = append(*s.c.log, "activate "+s.c.name) }
func (s *recordedSession) Dispose()            { *s.c.log = append(*s.c.log, "dispose "+s.c.name) }

func TestMouseCursorManager(t *testing.T) {
	var log []string
	basic := recordedCursor{"basic", &log}
	click := recordedCursor{"click", &log}
	text := recordedCursor{"text", &log}
	m := NewMouseCursorManager(basic)

	check := func(want ...string) {
		t.Helper()
		if !reflect.DeepEqual(log, want) {
			t.Errorf("log = %q, want %q", log, want)
		}
		log = nil
	}
	added := gestures.NewPointerAddedEvent()
	hover := gestures.NewPointerHoverEvent()

	m.HandleDeviceCursorUpdate(0, added, nil)
	check("activate basic") // all candidates defer: the fallback
	m.HandleDeviceCursorUpdate(0, hover, []MouseCursor{MouseCursorDefer, click, text})
	check("dispose basic", "activate click") // the first that does not defer
	m.HandleDeviceCursorUpdate(0, hover, []MouseCursor{click})
	check() // unchanged
	if c, ok := m.DebugDeviceActiveCursor(0); foundation.DebugMode && (!ok || c != click) {
		t.Errorf("active cursor = %v, %v", c, ok)
	}
	m.HandleDeviceCursorUpdate(1, added, []MouseCursor{text})
	check("activate text") // another device has its own session
	m.HandleDeviceCursorUpdate(0, gestures.NewPointerRemovedEvent(), nil)
	check() // the removed device is forgotten without a dispose
	if _, ok := m.DebugDeviceActiveCursor(0); ok {
		t.Error("the removed device has a cursor")
	}
	m.HandleDeviceCursorUpdate(0, added, []MouseCursor{MouseCursorUncontrolled})
	if len(log) != 0 {
		t.Errorf("log = %q, want the uncontrolled cursor to do nothing", log)
	}
}

func TestMouseCursorFirstNonDeferred(t *testing.T) {
	if _, ok := MouseCursorFirstNonDeferred([]MouseCursor{MouseCursorDefer}); ok {
		t.Error("found a cursor among deferring ones")
	}
	if c, ok := MouseCursorFirstNonDeferred([]MouseCursor{MouseCursorDefer, MouseCursorUncontrolled}); !ok || c != MouseCursorUncontrolled {
		t.Errorf("got %v, %v", c, ok)
	}
}

func TestMouseCursorDescriptions(t *testing.T) {
	if MouseCursorDefer.DebugDescription() != "defer" || MouseCursorUncontrolled.DebugDescription() != "uncontrolled" {
		t.Error("wrong descriptions")
	}
}

func TestMouseCursorErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fn    func()
		tag   string
		debug bool // the panic is a debug-mode check
	}{
		{"a deferring cursor creates no session", func() { MouseCursorDefer.CreateSession(0) }, "MouseCursorDeferCreatedSession", false},
		{"the fallback can not defer", func() { NewMouseCursorManager(MouseCursorDefer) }, "MouseCursorFallbackDefers", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.debug && !foundation.DebugMode {
				t.Skip("the check is for debug builds")
			}
			defer func() {
				err, _ := recover().(error)
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tc.tag {
					t.Errorf("recovered %v, want tag %s", err, tc.tag)
				}
			}()
			tc.fn()
		})
	}
}
