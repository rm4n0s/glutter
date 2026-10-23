package engine

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
)

var _ platform.SurfaceHandler = (*handler)(nil)

// TestSurfaceLostReleasesTheRenderersSurface checks that a platform whose native surface goes away
// (an Android activity) makes the renderer let go of it (platform.SurfaceHandler).
func TestSurfaceLostReleasesTheRenderersSurface(t *testing.T) {
	lost := gpu.SurfaceHandle{Kind: gpu.SurfaceKindAndroid, Window: 0x1000}
	r := &fakeRenderer{}
	h := &handler{r: r}
	h.OnSurfaceLost(1, lost)
	if len(r.released) != 1 || r.released[0] != lost {
		t.Errorf("released %v, want %v", r.released, lost)
	}
	// Before the first frame there is no renderer: nothing to release, nothing to fail.
	(&handler{}).OnSurfaceLost(1, lost)
}
