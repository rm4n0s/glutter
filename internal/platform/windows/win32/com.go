package win32

// The positions of the methods in the vtables of the COM interfaces that this package uses. The
// vtable of a COM interface is an array of function pointers; the base interfaces come first, so
// every interface starts with the three methods of IUnknown. The layout test of this package checks
// each index against the mingw-w64 headers (SPEC §25.2).
const (
	// IUnknownQueryInterface is the index of IUnknown::QueryInterface.
	//
	// Flutter: no counterpart
	IUnknownQueryInterface = 0
	// IUnknownAddRef is the index of IUnknown::AddRef.
	//
	// Flutter: no counterpart
	IUnknownAddRef = 1
	// IUnknownRelease is the index of IUnknown::Release.
	//
	// Flutter: no counterpart
	IUnknownRelease = 2
)

// CallbackLimit is the number of callbacks (syscall.NewCallback) that a process of Glutter may make:
// the runtime allows about 2,000 and never frees one, so a test fails above this number (SPEC
// §6.3 rule 11, §25.2). Callbacks are made once at startup, one per window procedure and one per
// method of each COM interface type that Go implements, never one per object.
//
// Flutter: no counterpart
const CallbackLimit = 1500
