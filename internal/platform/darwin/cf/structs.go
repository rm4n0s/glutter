package cf

// The Go mirrors of the Core Foundation structures that this package passes to the system. They are in a
// file with no OS tag, so that their layout tests run wherever the SDK's headers are (SPEC §29.2). Only
// 64-bit targets build, so CFIndex is int64.

// Range is Core Foundation's CFRange: a location and a length.
//
// Flutter: no counterpart
type Range struct {
	Location int64 `c:"location"`
	Length   int64 `c:"length"`
}

// SourceContext is CFRunLoopSourceContext as it is for a version-0 source: the info that Core
// Foundation hands back, and the perform function. The function pointers that this package does not
// use are zero.
//
// Flutter: no counterpart
type SourceContext struct {
	Version         int64   `c:"version"`
	Info            uintptr `c:"info"`
	Retain          uintptr `c:"retain"`
	Release         uintptr `c:"release"`
	CopyDescription uintptr `c:"copyDescription"`
	Equal           uintptr `c:"equal"`
	Hash            uintptr `c:"hash"`
	Schedule        uintptr `c:"schedule"`
	Cancel          uintptr `c:"cancel"`
	Perform         uintptr `c:"perform"`
}

// TimerContext is CFRunLoopTimerContext.
//
// Flutter: no counterpart
type TimerContext struct {
	Version         int64   `c:"version"`
	Info            uintptr `c:"info"`
	Retain          uintptr `c:"retain"`
	Release         uintptr `c:"release"`
	CopyDescription uintptr `c:"copyDescription"`
}
