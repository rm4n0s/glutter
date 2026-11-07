//go:build linux && !android

package pulse

import (
	"log/slog"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The C→Go callbacks of libpulse are made once for the process (SPEC §8.1 rule 2) and tell the Go object that an event is for by
// the user data, which is the handle of the object in the table below (precedent: the proxies of wayland/wl). libpulse calls them
// on the thread of its main loop with the lock of the loop held, so what they touch is guarded by that lock.

// handleTable maps the integer user data of a callback to its Go object (a *Service or a *device). Nothing else is process-wide.
var handleTable = struct {
	sync.Mutex
	next uintptr
	m    map[uintptr]any
}{next: 1, m: map[uintptr]any{}}

func addHandle(o any) uintptr {
	handleTable.Lock()
	defer handleTable.Unlock()
	h := handleTable.next
	handleTable.next++
	handleTable.m[h] = o
	return h
}

func getHandle(h uintptr) any {
	handleTable.Lock()
	defer handleTable.Unlock()
	return handleTable.m[h]
}

func removeHandle(h uintptr) {
	handleTable.Lock()
	defer handleTable.Unlock()
	delete(handleTable.m, h)
}

// handleCount returns how many objects are registered, for tests.
func handleCount() int {
	handleTable.Lock()
	defer handleTable.Unlock()
	return len(handleTable.m)
}

// callbackSet holds the C function pointers of the callbacks.
type callbackSet struct {
	contextState  uintptr // void (pa_context *c, void *userdata)
	subscribe     uintptr // void (pa_context *c, pa_subscription_event_type_t t, uint32_t idx, void *userdata)
	serverInfo    uintptr // void (pa_context *c, const pa_server_info *i, void *userdata)
	sinkInfo      uintptr // void (pa_context *c, const pa_sink_info *i, int eol, void *userdata)
	sourceInfo    uintptr // void (pa_context *c, const pa_source_info *i, int eol, void *userdata)
	streamState   uintptr // void (pa_stream *s, void *userdata)
	streamRequest uintptr // void (pa_stream *s, size_t nbytes, void *userdata): both the write and the read callback
	underflow     uintptr // void (pa_stream *s, void *userdata): the same shape as the state callback
}

// callbacks makes the set on first use, and returns the same one afterwards.
var callbacks = sync.OnceValue(func() callbackSet {
	return callbackSet{
		contextState:  dl.Callback(onContextState),
		subscribe:     dl.Callback(onSubscribe),
		serverInfo:    dl.Callback(onServerInfo),
		sinkInfo:      dl.Callback(onSinkInfo),
		sourceInfo:    dl.Callback(onSourceInfo),
		streamState:   dl.Callback(onStreamState),
		streamRequest: dl.Callback(onStreamRequest),
		underflow:     dl.Callback(onStreamUnderflow),
	}
})

// guard stops a panic of a handler from unwinding through the C frames of libpulse's thread: it is logged instead.
func guard(what string) {
	if r := recover(); r != nil {
		slog.Error("audio: a callback panicked", "callback", what, "panic", r)
	}
}

func onContextState(c, userdata uintptr) {
	defer guard("context state")
	if s, ok := getHandle(userdata).(*Service); ok {
		s.contextStateChanged(c)
	}
}

func onSubscribe(c, event, index, userdata uintptr) {
	defer guard("subscribe")
	if s, ok := getHandle(userdata).(*Service); ok {
		s.subscribed(uint32(event))
	}
}

func onServerInfo(c, info, userdata uintptr) {
	defer guard("server info")
	if s, ok := getHandle(userdata).(*Service); ok && info != 0 {
		s.serverInfo(info)
	}
}

func onSinkInfo(c, info, eol, userdata uintptr) {
	defer guard("sink info")
	if s, ok := getHandle(userdata).(*Service); ok {
		s.deviceInfo(info, int32(eol), false)
	}
}

func onSourceInfo(c, info, eol, userdata uintptr) {
	defer guard("source info")
	if s, ok := getHandle(userdata).(*Service); ok {
		s.deviceInfo(info, int32(eol), true)
	}
}

func onStreamState(stream, userdata uintptr) {
	defer guard("stream state")
	if d, ok := getHandle(userdata).(*device); ok {
		d.stateChanged()
	}
}

func onStreamRequest(stream, nbytes, userdata uintptr) {
	defer guard("stream request")
	if d, ok := getHandle(userdata).(*device); ok {
		d.service()
	}
}

func onStreamUnderflow(stream, userdata uintptr) {
	defer guard("stream underflow")
	if d, ok := getHandle(userdata).(*device); ok {
		d.underruns++
	}
}
