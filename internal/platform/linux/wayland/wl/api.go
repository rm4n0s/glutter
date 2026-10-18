package wl

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// clientLibrary is the versioned name of libwayland-client; the unversioned
// symlink exists only when the -devel package is installed (SPEC §8.1).
const clientLibrary = "libwayland-client.so.0"

// flagDestroy is WL_MARSHAL_FLAG_DESTROY: libwayland destroys the proxy after
// sending the request.
const flagDestroy = 1 << 0

// clientAPI is the set of libwayland-client functions this package calls. It is
// built once and never changes. None of them is variadic (SPEC §8.1 rule 1):
// requests go through wl_proxy_marshal_array_flags.
type clientAPI struct {
	displayConnect          func(name uintptr) uintptr
	displayConnectToFD      func(fd int32) uintptr
	displayDisconnect       func(display uintptr)
	displayGetFD            func(display uintptr) int32
	displayFlush            func(display uintptr) int32
	displayPrepareRead      func(display uintptr) int32
	displayReadEvents       func(display uintptr) int32
	displayCancelRead       func(display uintptr)
	displayDispatchPending  func(display uintptr) int32
	displayRoundtrip        func(display uintptr) int32
	displayGetError         func(display uintptr) int32
	displayGetProtocolError func(display, iface, id uintptr) uint32

	proxyMarshalArrayFlags func(proxy uintptr, opcode uint32, iface uintptr, version, flags uint32, args uintptr) uintptr
	proxyAddDispatcher     func(proxy, dispatcher, implementation, data uintptr) int32
	proxyDestroy           func(proxy uintptr)
	proxySetUserData       func(proxy, data uintptr)
	proxyGetUserData       func(proxy uintptr) uintptr
	proxyGetVersion        func(proxy uintptr) uint32
	proxyGetID             func(proxy uintptr) uint32
}

// loadClient opens libwayland-client once and binds the functions. A failure
// is returned on every call (LibraryLoadFailed or SymbolMissing).
var loadClient = sync.OnceValues(func() (*clientAPI, error) {
	lib, err := dl.Open(clientLibrary)
	if err != nil {
		return nil, err
	}
	api := new(clientAPI)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&api.displayConnect, "wl_display_connect"},
		{&api.displayConnectToFD, "wl_display_connect_to_fd"},
		{&api.displayDisconnect, "wl_display_disconnect"},
		{&api.displayGetFD, "wl_display_get_fd"},
		{&api.displayFlush, "wl_display_flush"},
		{&api.displayPrepareRead, "wl_display_prepare_read"},
		{&api.displayReadEvents, "wl_display_read_events"},
		{&api.displayCancelRead, "wl_display_cancel_read"},
		{&api.displayDispatchPending, "wl_display_dispatch_pending"},
		{&api.displayRoundtrip, "wl_display_roundtrip"},
		{&api.displayGetError, "wl_display_get_error"},
		{&api.displayGetProtocolError, "wl_display_get_protocol_error"},
		{&api.proxyMarshalArrayFlags, "wl_proxy_marshal_array_flags"},
		{&api.proxyAddDispatcher, "wl_proxy_add_dispatcher"},
		{&api.proxyDestroy, "wl_proxy_destroy"},
		{&api.proxySetUserData, "wl_proxy_set_user_data"},
		{&api.proxyGetUserData, "wl_proxy_get_user_data"},
		{&api.proxyGetVersion, "wl_proxy_get_version"},
		{&api.proxyGetID, "wl_proxy_get_id"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return api, nil
})

// dispatcherPointer is the C function pointer of the one dispatcher that every
// proxy uses (SPEC §8.1 rule 2). It is created on first use.
var dispatcherPointer = sync.OnceValue(func() uintptr {
	return dl.Callback(dispatchEvent)
})
