//go:build (linux && glutter_display) || (darwin && !ios && glutter_display) || (windows && glutter_wine)

package viewport3d_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/scenetest"
)

// TestPicksOnTheDevice makes the picks of internal/scenetest (SPEC §27.8, "Picking") on the device of the test: lavapipe on Linux, Direct3D 12
// under Wine. The cases run on the test's goroutine, which package gpu requires; each is reported alone.
func TestPicksOnTheDevice(t *testing.T) {
	app, d := deviceApp(t)
	for _, c := range scenetest.PickChecks() {
		msg, err := scenetest.RunPickCheck(d, waitFor(app), c)
		switch {
		case err != nil:
			t.Errorf("%s: %v", c.Name, err)
		case msg != "":
			t.Errorf("%s: %s", c.Name, msg)
		}
	}
}
