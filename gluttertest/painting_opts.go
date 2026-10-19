package gluttertest

import (
	"time"

	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/painting"
)

// widgetsPaintingOpts makes the painting binding's timers run on the fake clock.
func widgetsPaintingOpts(c *fakeclock.Fake) painting.PaintingBindingOpts {
	return painting.PaintingBindingOpts{
		AfterFunc: func(d time.Duration, fn func()) func() {
			timer := c.AfterFunc(d, fn)
			return func() { timer.Stop() }
		},
	}
}
