//go:build linux && !android

package wayland

import (
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// connSource plugs a Wayland connection into the event loop. It follows
// libwayland's pattern for an external loop: before the wait, dispatch what is
// queued, announce the read and flush; after the wait, read or cancel, and
// dispatch (docs/SPEC.md §8.4).
type connSource struct {
	conn *wl.Conn
	// blocked is whether the last flush could not send everything, so the loop
	// must also wait for the socket to become writable.
	blocked bool
	// reading is whether BeforeWait announced a read that AfterWait owes an end.
	reading bool
}

var _ linux.Source = (*connSource)(nil)

func (c *connSource) FD() int          { return c.conn.FD() }
func (c *connSource) WantsWrite() bool { return c.blocked }

func (c *connSource) BeforeWait() error {
	for !c.conn.PrepareRead() {
		// Events are already queued; handle them first.
		if _, err := c.conn.DispatchPending(); err != nil {
			return err
		}
	}
	c.reading = true
	blocked, err := c.conn.Flush()
	if err != nil {
		c.conn.CancelRead()
		c.reading = false
		return err
	}
	c.blocked = blocked
	return nil
}

func (c *connSource) AfterWait(ready linux.Events) error {
	if c.reading {
		c.reading = false
		if ready&linux.Readable != 0 {
			if err := c.conn.ReadEvents(); err != nil {
				return err
			}
		} else {
			c.conn.CancelRead()
		}
	}
	if ready&linux.Writable != 0 {
		blocked, err := c.conn.Flush()
		if err != nil {
			return err
		}
		c.blocked = blocked
	}
	_, err := c.conn.DispatchPending()
	return err
}
