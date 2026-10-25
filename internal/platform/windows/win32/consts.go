package win32

import "time"

// The flags and results of the kernel32 functions that this package binds (kernel32_windows.go). They
// carry no OS tag, so that the layout test checks them against the headers.
const (
	// HeapZeroMemory is HEAP_ZERO_MEMORY: HeapAlloc clears the block.
	//
	// Flutter: no counterpart
	HeapZeroMemory = 0x00000008
	// GMemMoveable is GMEM_MOVEABLE, the flag the clipboard needs for GlobalAlloc.
	//
	// Flutter: no counterpart
	GMemMoveable = 0x0002
	// GMemZeroInit is GMEM_ZEROINIT: GlobalAlloc clears the block.
	//
	// Flutter: no counterpart
	GMemZeroInit = 0x0040
	// WaitInfinite is the timeout that never ends (INFINITE).
	//
	// Flutter: no counterpart
	WaitInfinite time.Duration = -1
)

// WaitResult is what a wait function returns.
//
// Flutter: no counterpart
type WaitResult uint32

// The results of WaitForSingleObject and WaitForMultipleObjects. For several objects the result
// WaitObject0 + i says that object i is signaled (WaitAbandoned0 + i, that it was abandoned).
const (
	// WaitObject0 is WAIT_OBJECT_0: the object is signaled.
	//
	// Flutter: no counterpart
	WaitObject0 WaitResult = 0x00000000
	// WaitAbandoned0 is WAIT_ABANDONED_0.
	//
	// Flutter: no counterpart
	WaitAbandoned0 WaitResult = 0x00000080
	// WaitTimeout is WAIT_TIMEOUT: the time ran out.
	//
	// Flutter: no counterpart
	WaitTimeout WaitResult = 0x00000102
	waitFailed  WaitResult = 0xFFFFFFFF
)
