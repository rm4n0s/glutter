package wl

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// epROTO is Linux's EPROTO, the error that libwayland records for a protocol
// error.
const epROTO = 71

// Conn is a connection to a Wayland compositor. Use it only on the UI thread.
//
// Flutter: no counterpart
type Conn struct {
	api     *clientAPI
	display uintptr
	disp    *WlDisplay
	// live holds the handles of the objects of this connection, so Close can
	// forget them.
	live map[uintptr]struct{}
	// panicValue is the first panic raised by an event handler during the
	// current call into libwayland. See rethrow.
	panicValue *panicBox
}

// Connect connects to the compositor with the given socket name. An empty name
// uses $WAYLAND_DISPLAY, then "wayland-0", as libwayland does. It fails with
// WaylandConnectFailed when the compositor cannot be reached, and with
// LibraryLoadFailed when libwayland-client is not installed.
//
// Flutter: no counterpart
func Connect(name string) (*Conn, error) {
	api, err := loadClient()
	if err != nil {
		return nil, err
	}
	var namePtr uintptr
	if name != "" {
		namePtr = dl.CString(name)
		defer dl.Free(namePtr)
	}
	display := api.displayConnect(namePtr)
	if display == 0 {
		return nil, errors.New("WaylandConnectFailed", "could not connect to the Wayland compositor", "display", name)
	}
	return newConn(api, display), nil
}

// ConnectFD connects over an already connected socket, which the Conn takes
// ownership of. It exists for tests that run a fake compositor on the other end
// of a socket pair. A failure is WaylandConnectFailed.
//
// Flutter: no counterpart
func ConnectFD(fd int) (*Conn, error) {
	api, err := loadClient()
	if err != nil {
		return nil, err
	}
	display := api.displayConnectToFD(int32(fd))
	if display == 0 {
		return nil, errors.New("WaylandConnectFailed", "could not use the socket as a Wayland connection", "fd", fd)
	}
	return newConn(api, display), nil
}

func newConn(api *clientAPI, display uintptr) *Conn {
	c := &Conn{api: api, display: display, live: map[uintptr]struct{}{}}
	c.disp = newWlDisplay(c, display)
	return c
}

// Display returns the wl_display object, from which to get the registry.
//
// Flutter: no counterpart
func (c *Conn) Display() *WlDisplay { return c.disp }

// FD returns the file descriptor of the connection, to poll for readability.
//
// Flutter: no counterpart
func (c *Conn) FD() int { return int(c.api.displayGetFD(c.display)) }

// Close disconnects. Every object of the connection becomes unusable.
//
// Flutter: no counterpart
func (c *Conn) Close() {
	if c.display == 0 {
		return
	}
	for h := range c.live {
		removeHandle(h)
	}
	c.live = map[uintptr]struct{}{}
	c.api.displayDisconnect(c.display)
	c.display = 0
}

// Flush sends the buffered requests. blocked is true when the socket is full:
// wait for it to become writable and call Flush again. An error means the
// connection is lost.
//
// Flutter: no counterpart
func (c *Conn) Flush() (blocked bool, err error) {
	if c.api.displayFlush(c.display) >= 0 {
		return false, nil
	}
	// libwayland records every error except EAGAIN as the display error.
	if err := c.lastError("Flush"); err != nil {
		return false, err
	}
	return true, nil
}

// PrepareRead announces that the caller will read from the socket. It returns
// false when events are already queued: call DispatchPending, then try again.
// After a true result, call exactly one of ReadEvents or CancelRead.
//
// Flutter: no counterpart
func (c *Conn) PrepareRead() bool { return c.api.displayPrepareRead(c.display) == 0 }

// ReadEvents reads from the socket, once it is readable, and queues the events.
//
// Flutter: no counterpart
func (c *Conn) ReadEvents() error {
	if c.api.displayReadEvents(c.display) < 0 {
		return c.lastErrorOr("ReadEvents")
	}
	return nil
}

// CancelRead ends a PrepareRead without reading.
//
// Flutter: no counterpart
func (c *Conn) CancelRead() { c.api.displayCancelRead(c.display) }

// DispatchPending calls the handlers of the queued events and returns how many
// there were. A panic in a handler is raised again here.
//
// Flutter: no counterpart
func (c *Conn) DispatchPending() (int, error) {
	n := c.api.displayDispatchPending(c.display)
	c.rethrow()
	if n < 0 {
		return 0, c.lastErrorOr("DispatchPending")
	}
	return int(n), nil
}

// Roundtrip sends the requests, waits until the compositor has handled them,
// and dispatches the events that arrive meanwhile. A panic in a handler is
// raised again here.
//
// Flutter: no counterpart
func (c *Conn) Roundtrip() error {
	n := c.api.displayRoundtrip(c.display)
	c.rethrow()
	if n < 0 {
		return c.lastErrorOr("Roundtrip")
	}
	return nil
}

// rethrow raises again a panic that an event handler caused inside a call into
// libwayland, now that the C frames are gone.
func (c *Conn) rethrow() {
	if box := c.panicValue; box != nil {
		c.panicValue = nil
		panic(box.value)
	}
}

// lastError returns the error that libwayland recorded, or nil if there is none.
// A protocol error is WaylandProtocolError (metadata "interface", "id", "code");
// any other is WaylandConnectionLost (metadata "operation", "errno").
func (c *Conn) lastError(operation string) error {
	errno := c.api.displayGetError(c.display)
	if errno == 0 {
		return nil
	}
	if errno == epROTO {
		ifacePtr := dl.Calloc(1, 8)
		idPtr := dl.Calloc(1, 4)
		defer dl.Free(ifacePtr)
		defer dl.Free(idPtr)
		code := c.api.displayGetProtocolError(c.display, ifacePtr, idPtr)
		name := ""
		if iface := *(*uintptr)(dl.Pointer(ifacePtr)); iface != 0 {
			name = dl.GoString((*cInterface)(dl.Pointer(iface)).Name)
		}
		return errors.New("WaylandProtocolError", "the compositor reported a protocol error",
			"operation", operation, "interface", name, "id", *(*uint32)(dl.Pointer(idPtr)), "code", code)
	}
	return errors.New("WaylandConnectionLost", "the connection to the compositor failed",
		"operation", operation, "errno", errno)
}

// lastErrorOr is lastError for a call that reported failure: when libwayland
// recorded nothing, the failure is still reported.
func (c *Conn) lastErrorOr(operation string) error {
	if err := c.lastError(operation); err != nil {
		return err
	}
	return errors.New("WaylandConnectionLost", "the connection to the compositor failed",
		"operation", operation, "errno", 0)
}
