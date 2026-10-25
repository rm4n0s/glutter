//go:build windows && glutter_wine

package windows

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// DPIChildEnv makes the test binary run TestDPIChild: the monitors of Wine are made when its server
// starts, so the DPI is changed by the host side (dpi_host_wine_test.go), which writes the registry,
// stops the server and starts this program again.
const dpiChildEnv = "GLUTTER_DPI_CHILD"

// TestDPIChild is the program that the host-side test starts: it opens a window of 320x240 logical
// pixels and prints the scale and the sizes it got.
func TestDPIChild(t *testing.T) {
	if os.Getenv(dpiChildEnv) != "1" {
		t.Skip("started by TestScaleFollowsTheDPIOfTheDisplay on the host")
	}
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		rc, _ := win32.GetClientRect(w.hwnd)
		m := r.metrics[0]
		fmt.Printf("RESULT scale=%v client=%dx%d metrics=%vx%v dpr=%v displays=%v\n", w.scale(), rc.Right, rc.Bottom,
			m.PhysicalSize.Width, m.PhysicalSize.Height, m.DevicePixelRatio, p.Displays()[0].DevicePixelRatio)
	})
}
