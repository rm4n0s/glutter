package wl

// WrapBuffer returns a *WlBuffer for a wl_buffer that another library created on
// this connection, such as libwayland-cursor for a cursor image, so that it can
// be attached to a surface. The wrapper has no event handler and is not
// registered: the library that made the buffer owns it, and the Go side must not
// destroy it.
//
// Flutter: no counterpart
func (c *Conn) WrapBuffer(ptr uintptr) *WlBuffer {
	b := new(WlBuffer)
	b.conn = c
	b.ptr = ptr
	return b
}
