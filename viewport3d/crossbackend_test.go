//go:build (linux && glutter_display) || (darwin && !ios && glutter_display) || (windows && glutter_wine)

package viewport3d_test

import (
	"time"

	"github.com/rm4n0s/glutter/internal/gpuhost"
)

// waitFor makes the wait function of scenetest.Render from the app: it runs the posted work until the read is done, which
// takes one call on a device that finishes at once and many on a device that the GPU works for.
func waitFor(app *gpuhost.App) func(done func() bool) {
	return func(done func() bool) {
		for i := 0; i < 2000 && !done(); i++ {
			app.RunPosted()
			if !done() {
				time.Sleep(time.Millisecond)
			}
		}
	}
}
