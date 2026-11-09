//go:build darwin

package cf

import (
	"sync"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// Ref is any Core Foundation object (CFTypeRef), by its address.
//
// Flutter: no counterpart
type Ref uintptr

// kCFStringEncodingUTF8 is Core Foundation's UTF-8 string encoding.
const kCFStringEncodingUTF8 = 0x08000100

// funcs is the table of the Core Foundation functions of this package, bound once.
type funcs struct {
	retain  func(Ref) Ref
	release func(Ref)

	stringCreateWithBytes func(alloc Ref, bytes unsafe.Pointer, n int64, encoding uint32, isExternal bool) Ref
	stringGetLength       func(Ref) int64
	stringGetBytes        func(s Ref, r Range, encoding uint32, lossByte uint8, isExternal bool, buf unsafe.Pointer, maxBuf int64, used *int64) int64

	numberCreate func(alloc Ref, typ int64, value unsafe.Pointer) Ref

	dictionaryCreate   func(alloc Ref, keys, values unsafe.Pointer, n int64, keyCB, valueCB uintptr) Ref
	dictionaryGetValue func(d, key Ref) Ref
	dictionaryGetCount func(d Ref) int64

	runLoopGetMain         func() Ref
	runLoopGetCurrent      func() Ref
	runLoopWakeUp          func(Ref)
	runLoopRunInMode       func(mode Ref, seconds float64, returnAfterSourceHandled bool) int32
	runLoopStop            func(Ref)
	runLoopAddSource       func(rl, source, mode Ref)
	runLoopRemoveSource    func(rl, source, mode Ref)
	runLoopAddTimer        func(rl, timer, mode Ref)
	runLoopRemoveTimer     func(rl, timer, mode Ref)
	runLoopSourceCreate    func(alloc Ref, order int64, ctx *SourceContext) Ref
	runLoopSourceSignal    func(Ref)
	runLoopSourceInvalid   func(Ref)
	runLoopTimerCreate     func(alloc Ref, fireDate, interval float64, flags uint64, order int64, callout uintptr, ctx *TimerContext) Ref
	runLoopTimerSetNext    func(timer Ref, at float64)
	runLoopTimerInvalidate func(Ref)
	absoluteTimeGetCurrent func() float64

	commonModes, defaultMode Ref // the CFStringRef constants kCFRunLoopCommonModes and kCFRunLoopDefaultMode
	typeKeyCB, typeValueCB   uintptr
}

var table = sync.OnceValue(func() *funcs {
	lib, err := dl.OpenFramework("CoreFoundation")
	if err != nil {
		panic(err)
	}
	f := new(funcs)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&f.retain, "CFRetain"}, {&f.release, "CFRelease"},
		{&f.stringCreateWithBytes, "CFStringCreateWithBytes"}, {&f.stringGetLength, "CFStringGetLength"},
		{&f.stringGetBytes, "CFStringGetBytes"},
		{&f.numberCreate, "CFNumberCreate"},
		{&f.dictionaryCreate, "CFDictionaryCreate"}, {&f.dictionaryGetValue, "CFDictionaryGetValue"},
		{&f.dictionaryGetCount, "CFDictionaryGetCount"},
		{&f.runLoopGetMain, "CFRunLoopGetMain"}, {&f.runLoopGetCurrent, "CFRunLoopGetCurrent"},
		{&f.runLoopWakeUp, "CFRunLoopWakeUp"}, {&f.runLoopRunInMode, "CFRunLoopRunInMode"},
		{&f.runLoopStop, "CFRunLoopStop"},
		{&f.runLoopAddSource, "CFRunLoopAddSource"}, {&f.runLoopRemoveSource, "CFRunLoopRemoveSource"},
		{&f.runLoopAddTimer, "CFRunLoopAddTimer"}, {&f.runLoopRemoveTimer, "CFRunLoopRemoveTimer"},
		{&f.runLoopSourceCreate, "CFRunLoopSourceCreate"}, {&f.runLoopSourceSignal, "CFRunLoopSourceSignal"},
		{&f.runLoopSourceInvalid, "CFRunLoopSourceInvalidate"},
		{&f.runLoopTimerCreate, "CFRunLoopTimerCreate"}, {&f.runLoopTimerSetNext, "CFRunLoopTimerSetNextFireDate"},
		{&f.runLoopTimerInvalidate, "CFRunLoopTimerInvalidate"},
		{&f.absoluteTimeGetCurrent, "CFAbsoluteTimeGetCurrent"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			panic(err)
		}
	}
	// Data symbols: Dlsym gives the address of the variable that holds the constant.
	for _, c := range []struct {
		dst    *Ref
		symbol string
	}{{&f.commonModes, "kCFRunLoopCommonModes"}, {&f.defaultMode, "kCFRunLoopDefaultMode"}} {
		addr, err := lib.Symbol(c.symbol)
		if err != nil {
			panic(err)
		}
		*c.dst = *(*Ref)(dl.Pointer(addr))
	}
	for _, c := range []struct {
		dst    *uintptr
		symbol string
	}{{&f.typeKeyCB, "kCFTypeDictionaryKeyCallBacks"}, {&f.typeValueCB, "kCFTypeDictionaryValueCallBacks"}} {
		addr, err := lib.Symbol(c.symbol)
		if err != nil {
			panic(err)
		}
		*c.dst = addr // the callbacks structures themselves: CFDictionaryCreate takes their addresses
	}
	return f
})

// kCFNumberSInt64Type is Core Foundation's 64-bit signed integer number type.
const kCFNumberSInt64Type = 4

// NewNumber makes a CFNumber of the integer (+1: release it).
//
// Flutter: no counterpart
func NewNumber(v int64) Ref {
	return table().numberCreate(0, kCFNumberSInt64Type, unsafe.Pointer(&v))
}

// Retain adds a reference to the object and returns it. 0 stays 0.
//
// Flutter: no counterpart
func Retain(r Ref) Ref {
	if r == 0 {
		return 0
	}
	return table().retain(r)
}

// Release gives up a reference that the caller owns. 0 does nothing.
//
// Flutter: no counterpart
func Release(r Ref) {
	if r != 0 {
		table().release(r)
	}
}

// NewString makes a CFString from a Go string (+1: release it). An embedded NUL is kept. The bytes must be
// valid UTF-8, or the result is 0.
//
// Flutter: no counterpart
func NewString(s string) Ref {
	var data unsafe.Pointer
	if len(s) > 0 {
		data = unsafe.Pointer(unsafe.StringData(s))
	}
	return table().stringCreateWithBytes(0, data, int64(len(s)), kCFStringEncodingUTF8, false)
}

// String copies a CFString into a Go string. 0 gives "". An embedded NUL is kept.
//
// Flutter: no counterpart
func String(r Ref) string {
	if r == 0 {
		return ""
	}
	t := table()
	length := t.stringGetLength(r)
	if length == 0 {
		return ""
	}
	whole := Range{Location: 0, Length: length}
	var used int64
	// First call: how many bytes the UTF-8 form takes. Second: the bytes.
	if t.stringGetBytes(r, whole, kCFStringEncodingUTF8, 0, false, nil, 0, &used) == 0 {
		return ""
	}
	buf := make([]byte, used)
	t.stringGetBytes(r, whole, kCFStringEncodingUTF8, 0, false, unsafe.Pointer(&buf[0]), used, &used)
	return string(buf[:used])
}

// NewDictionary makes a CFDictionary of the keys and values, which it retains (+1: release it). keys and
// values must have the same length, or it panics (a mistake of the caller).
//
// Flutter: no counterpart
func NewDictionary(keys, values []Ref) Ref {
	if len(keys) != len(values) {
		panic("cf: NewDictionary needs as many values as keys")
	}
	t := table()
	var k, v unsafe.Pointer
	if len(keys) > 0 {
		k = unsafe.Pointer(&keys[0])
		v = unsafe.Pointer(&values[0])
	}
	return t.dictionaryCreate(0, k, v, int64(len(keys)), t.typeKeyCB, t.typeValueCB)
}

// DictionaryGet is the value of the key, not owned by the caller, or 0 when the dictionary has none.
//
// Flutter: no counterpart
func DictionaryGet(dict, key Ref) Ref { return table().dictionaryGetValue(dict, key) }

// DictionaryCount is the number of entries of the dictionary.
//
// Flutter: no counterpart
func DictionaryCount(dict Ref) int { return int(table().dictionaryGetCount(dict)) }

// MainRunLoop is the run loop of the main thread (not owned by the caller).
//
// Flutter: no counterpart
func MainRunLoop() Ref { return table().runLoopGetMain() }

// CurrentRunLoop is the run loop of the calling thread (not owned by the caller).
//
// Flutter: no counterpart
func CurrentRunLoop() Ref { return table().runLoopGetCurrent() }

// CommonModes is kCFRunLoopCommonModes: the pseudo mode that stands for every mode that is marked
// common (the default one, the modes of event tracking and of a modal panel). Sources and timers that
// must run while a menu is open or a window is being resized are added in it (SPEC §29.3).
//
// Flutter: no counterpart
func CommonModes() Ref { return table().commonModes }

// DefaultMode is kCFRunLoopDefaultMode.
//
// Flutter: no counterpart
func DefaultMode() Ref { return table().defaultMode }

// WakeUp wakes the run loop if it is asleep. Any goroutine may call it.
//
// Flutter: no counterpart
func WakeUp(rl Ref) { table().runLoopWakeUp(rl) }

// The results of RunInMode (CFRunLoopRunResult).
const (
	// RunFinished says that the mode had no sources or timers.
	RunFinished = 1
	// RunStopped says that Stop was called.
	RunStopped = 2
	// RunTimedOut says that the time given was up.
	RunTimedOut = 3
	// RunHandledSource says that a source was handled and the call asked to return then.
	RunHandledSource = 4
)

// RunInMode runs the calling thread's run loop in the mode for at most that long, and returns why it
// came back (the Run constants). With returnAfterSourceHandled it returns after one source.
//
// Flutter: no counterpart
func RunInMode(mode Ref, d time.Duration, returnAfterSourceHandled bool) int32 {
	return table().runLoopRunInMode(mode, d.Seconds(), returnAfterSourceHandled)
}

// Stop asks the run loop to stop; RunInMode then returns RunStopped.
//
// Flutter: no counterpart
func Stop(rl Ref) { table().runLoopStop(rl) }

// referenceDateUnix is the Unix time of Core Foundation's reference date, 2001-01-01 00:00:00 UTC.
const referenceDateUnix = 978307200

// AbsoluteTime converts a time to a CFAbsoluteTime: seconds since the reference date.
//
// Flutter: no counterpart
func AbsoluteTime(t time.Time) float64 {
	return float64(t.Unix()-referenceDateUnix) + float64(t.Nanosecond())/1e9
}

// Now is the current CFAbsoluteTime.
//
// Flutter: no counterpart
func Now() float64 { return table().absoluteTimeGetCurrent() }

// callbacks maps the id that a source or timer carries as its info to its function, because Core
// Foundation hands the info back and Go must find the function (rule 2 of SPEC §8.1): the two
// callbacks below are made once for the process. An entry is deleted by Close, so the map holds only
// what is alive.
var callbacks struct {
	mu   sync.RWMutex
	next uintptr
	fns  map[uintptr]func()
}

func addCallback(fn func()) uintptr {
	callbacks.mu.Lock()
	defer callbacks.mu.Unlock()
	if callbacks.fns == nil {
		callbacks.fns = map[uintptr]func(){}
	}
	callbacks.next++
	callbacks.fns[callbacks.next] = fn
	return callbacks.next
}

func dropCallback(id uintptr) {
	callbacks.mu.Lock()
	delete(callbacks.fns, id)
	callbacks.mu.Unlock()
}

func fire(id uintptr) {
	callbacks.mu.RLock()
	fn := callbacks.fns[id]
	callbacks.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

// performCallback is the perform function of every source, and timerCallback the callout of every timer:
// made once, on first use, and counted by package dl.
var (
	performCallback = sync.OnceValue(func() uintptr { return dl.Callback(func(info uintptr) { fire(info) }) })
	timerCallback   = sync.OnceValue(func() uintptr { return dl.Callback(func(_ Ref, info uintptr) { fire(info) }) })
)

// Source is a version-0 run loop source: Signal marks it ready from any goroutine, and the run loop it
// was added to calls fn on its own thread when it next wakes (that is how Post reaches the UI thread).
//
// Flutter: no counterpart
type Source struct {
	ref Ref
	id  uintptr
}

// NewSource makes a source that calls fn on the thread of the run loop that handles it. Add it to a run
// loop with AddTo, and give it back with Close.
//
// Flutter: no counterpart
func NewSource(fn func()) *Source {
	id := addCallback(fn)
	ctx := &SourceContext{Info: id, Perform: performCallback()}
	return &Source{ref: table().runLoopSourceCreate(0, 0, ctx), id: id}
}

// AddTo adds the source to the run loop in the mode (CommonModes, to run while a menu is open).
//
// Flutter: no counterpart
func (s *Source) AddTo(rl, mode Ref) {
	if s.ref != 0 {
		table().runLoopAddSource(rl, s.ref, mode)
	}
}

// Signal marks the source ready to run. Any goroutine may call it; the run loop still has to be woken
// (WakeUp) if it is asleep.
//
// Flutter: no counterpart
func (s *Source) Signal() {
	if s.ref != 0 {
		table().runLoopSourceSignal(s.ref)
	}
}

// Close invalidates the source, which takes it out of every run loop, and gives back its reference. It
// does nothing when called again.
//
// Flutter: no counterpart
func (s *Source) Close() {
	if s.ref == 0 {
		return
	}
	table().runLoopSourceInvalid(s.ref)
	Release(s.ref)
	dropCallback(s.id)
	s.ref = 0
}

// Timer is a run loop timer that fires once each time it is set to a time, so that one timer serves the
// timer heap of the UI thread (SPEC §29.3): the heap sets it to its next deadline each time it changes.
// Core Foundation invalidates a timer with no interval after its first fire, so this one has an interval of a
// hundred years: after it fires it is as good as disarmed, and SetNextFire arms it again.
//
// Flutter: no counterpart
type Timer struct {
	ref Ref
	id  uintptr
}

// farFuture is how far away the fire date of a timer that is not armed is, and the interval of every
// timer: about 100 years, in seconds.
const farFuture = 100 * 365 * 24 * 3600

// NewTimer makes a timer that is not armed (its fire date is a hundred years away) and calls fn on
// the thread of its run loop each time it fires. Add it with AddTo, arm it with SetNextFire, and give
// it back with Close.
//
// Flutter: no counterpart
func NewTimer(fn func()) *Timer {
	id := addCallback(fn)
	ctx := &TimerContext{Info: id}
	t := table()
	ref := t.runLoopTimerCreate(0, t.absoluteTimeGetCurrent()+farFuture, farFuture, 0, 0, timerCallback(), ctx)
	return &Timer{ref: ref, id: id}
}

// AddTo adds the timer to the run loop in the mode.
//
// Flutter: no counterpart
func (t *Timer) AddTo(rl, mode Ref) {
	if t.ref != 0 {
		table().runLoopAddTimer(rl, t.ref, mode)
	}
}

// SetNextFire arms the timer to fire once at that time (a time in the past fires at once).
//
// Flutter: no counterpart
func (t *Timer) SetNextFire(at time.Time) {
	if t.ref != 0 {
		table().runLoopTimerSetNext(t.ref, AbsoluteTime(at))
	}
}

// Disarm moves the next fire to the far future, so that the timer does not fire.
//
// Flutter: no counterpart
func (t *Timer) Disarm() {
	if t.ref != 0 {
		table().runLoopTimerSetNext(t.ref, table().absoluteTimeGetCurrent()+farFuture)
	}
}

// Close invalidates the timer and gives back its reference. It does nothing when called again.
//
// Flutter: no counterpart
func (t *Timer) Close() {
	if t.ref == 0 {
		return
	}
	table().runLoopTimerInvalidate(t.ref)
	Release(t.ref)
	dropCallback(t.id)
	t.ref = 0
}
