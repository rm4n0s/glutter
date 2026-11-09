//go:build darwin && !ios

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// This file loads the frameworks that the backend calls and has the small helpers that the other files
// share: selectors, classes, geometry values and the C functions that are not methods.

var sel = objcx.Sel

// class is the Objective-C class of the name as an ID, for the class methods (+alloc, +sharedApplication).
func class(name string) objc.ID { return objc.ID(objcx.MustClass(name)) }

// The values of AppKit's enumerations that the backend uses.
const (
	activationPolicyRegular   int64 = 0
	activationPolicyAccessory int64 = 1

	backingStoreBuffered uint64 = 2

	// NSEventTypeApplicationDefined is the type of the event that wakes the loop to end it.
	eventTypeApplicationDefined uint64 = 15

	titleVisible int64 = 0
	titleHidden  int64 = 1

	collectionBehaviorFullScreenPrimary uint64 = 1 << 7
	collectionBehaviorFullScreenNone    uint64 = 1 << 9
	// collectionBehaviorMoveToActiveSpace is not used: the window stays in its space.

	viewLayerContentsRedrawNever int64 = 0

	// windowLevelModal is NSModalPanelWindowLevel.
	windowLevelNormal int64 = 0

	terminateCancel uint64 = 0
	terminateNow    uint64 = 1
)

// frameworks are the frameworks that the backend loads before it looks for their classes, so that
// objc_getClass finds them. They are in the dyld shared cache, by absolute path (SPEC §29.2).
var frameworks = []string{"AppKit", "QuartzCore", "CoreGraphics", "IOKit", "CoreVideo"}

// system is what the process shares: the loaded frameworks, the classes that Go implements and the C
// functions of the backend.
type system struct {
	appDelegate, window, windowDelegate, view, settingsObserver, menuTarget, menuDelegate, dragSource *objcx.RegisteredClass

	// pthreadMainNP is pthread_main_np: nonzero on the main thread.
	pthreadMainNP func() int32
	// mediaTime is CACurrentMediaTime: seconds on the clock that CADisplayLink timestamps use.
	mediaTime func() float64
	// IOPMAssertion functions for KeepScreenOn.
	pmCreate  func(kind uintptr, level uint32, name uintptr, outID *uint32) int32
	pmRelease func(id uint32) int32
	// The CVDisplayLink functions, which macOS 12 and 13 use for the frames (SPEC §29.3).
	cv cvFuncs
	// kb are the functions of the keyboard layout (keyboard_darwin.go).
	kb kbFuncs
}

var sys = sync.OnceValues(func() (*system, error) {
	if err := objcx.Init(); err != nil {
		return nil, err
	}
	s := new(system)
	for _, name := range frameworks {
		if _, err := dl.OpenFramework(name); err != nil {
			return nil, err
		}
	}
	libc, err := dl.Open("/usr/lib/libSystem.B.dylib")
	if err != nil {
		return nil, err
	}
	if err := libc.Bind(&s.pthreadMainNP, "pthread_main_np"); err != nil {
		return nil, err
	}
	qc, err := dl.OpenFramework("QuartzCore")
	if err != nil {
		return nil, err
	}
	if err := qc.Bind(&s.mediaTime, "CACurrentMediaTime"); err != nil {
		return nil, err
	}
	iok, err := dl.OpenFramework("IOKit")
	if err != nil {
		return nil, err
	}
	if err := iok.Bind(&s.pmCreate, "IOPMAssertionCreateWithName"); err != nil {
		return nil, err
	}
	if err := iok.Bind(&s.pmRelease, "IOPMAssertionRelease"); err != nil {
		return nil, err
	}
	if err := s.cv.bind(); err != nil {
		return nil, err
	}
	if err := s.kb.bind(); err != nil {
		return nil, err
	}
	if err := s.registerClasses(); err != nil {
		return nil, err
	}
	return s, nil
})

// nsPoint, nsSize and nsRect are the geometry values of AppKit; the mirrors are in package darwin.
type (
	nsPoint = darwin.CGPoint
	nsSize  = darwin.CGSize
	nsRect  = darwin.CGRect
)

func rectOf(x, y, w, h float64) nsRect {
	return nsRect{Origin: nsPoint{X: x, Y: y}, Size: nsSize{Width: w, Height: h}}
}

// send is Send for a call whose result is a value of type T.
func send[T any](id objc.ID, name string, args ...any) T { return objc.Send[T](id, sel(name), args...) }

var _ = errors.New
