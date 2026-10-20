package decor

import (
	"testing"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// TestLayouts checks the Go mirrors of the two interface structs of libdecor.h
// (libdecor 0.2): a pointer for each function, and reserved pointers after them.
func TestLayouts(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the layouts assume 64-bit pointers")
	}
	dl.AssertLayout(t, "libdecor_interface", 11*8, map[string]uintptr{"errorFn": 0, "reserved": 8}, interfaceStruct{})
	dl.AssertLayout(t, "libdecor_frame_interface", 13*8,
		map[string]uintptr{"configure": 0, "closeFn": 8, "commit": 16, "dismissPopup": 24, "reserved": 32}, frameInterfaceStruct{})
}

func TestLibraryOverrideRoute(t *testing.T) {
	t.Setenv(LibraryEnv, "relative/libdecor.so")
	// loadAPI is cached for the process, so test the opener it uses.
	_, err := dl.OpenOverride(LibraryEnv, libraryName)
	if err == nil {
		t.Fatal("a relative override should fail")
	}
}

// routeOf asserts that err is a tagged error whose route contains route.
func routeOf(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

// fakeContext is a Context over functions that fail, since libdecor itself cannot
// be made to fail on demand.
func fakeContext() *Context {
	return &Context{api: &api{
		dispatch: func(uintptr, int32) int32 { return -1 },
		decorate: func(_, _, _, _ uintptr) uintptr { return 0 },
	}, ptr: 1}
}

func TestDispatchFailureRoute(t *testing.T) {
	routeOf(t, fakeContext().Dispatch(), "Context.Dispatch.LibdecorDispatchFailed")
}

func TestDecorateFailureRoute(t *testing.T) {
	_, err := fakeContext().Decorate(1, nil)
	routeOf(t, err, "Context.Decorate.LibdecorDecorateFailed")
}

// TestInitFailureRoute makes libdecor_new return NULL, as it does when it cannot
// start (a real libdecor can not be made to fail without crashing).
func TestInitFailureRoute(t *testing.T) {
	_, err := newContext(&api{new: func(_, _ uintptr) uintptr { return 0 }}, 1, nil)
	routeOf(t, err, "newContext.LibdecorInitFailed")
}

type noHandler struct{}

func (noHandler) Configure(Configuration) {}
func (noHandler) Close()                  {}
func (noHandler) Commit()                 {}
func (noHandler) DismissPopup(string)     {}

var _ Handler = noHandler{}
