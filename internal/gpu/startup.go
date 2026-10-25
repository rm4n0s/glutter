package gpu

import "time"

// StartupReport is what a renderer measured while it made its first frame: the parts of the cold start that
// GLUTTER_FRAME_STATS=1 logs (SPEC §9.7). A renderer that has one implements StartupReporter.
//
// Flutter: no counterpart
type StartupReport struct {
	// Context is how long it took to open the GPU: the GL context, or the Vulkan instance and device.
	Context time.Duration
	// Shaders is how long the programs or pipelines that were made so far took, whether compiled or loaded
	// from the cache.
	Shaders time.Duration
	// Built are the programs or pipelines made from source, Cached those made from the cache, and Refused the
	// cached entries that the driver did not take (they were built).
	Built, Cached, Refused int
	// CacheBytes is the size of the pipeline cache that was loaded from disk, for the renderer that has one
	// (Vulkan); 0 when there was none.
	CacheBytes int
}

// StartupReporter is implemented by a renderer that can say how its start went.
//
// Flutter: no counterpart
type StartupReporter interface {
	StartupReport() StartupReport
}
