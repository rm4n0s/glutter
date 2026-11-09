package metal

import "github.com/rm4n0s/glutter/internal/platform/darwin"

// The Go mirrors of the structures of Metal that the device passes by value (SPEC §29.2, §29.5). They are in a
// file with no OS tag so that their layout test runs wherever the SDK's headers are; NSUInteger is uint64
// and the doubles are float64 on the 64-bit targets.

// mtlOrigin is MTLOrigin: a position in a texture.
type mtlOrigin struct {
	X uint64 `c:"x"`
	Y uint64 `c:"y"`
	Z uint64 `c:"z"`
}

// mtlSize is MTLSize: the extent of a box of a texture or of a grid of threadgroups.
type mtlSize struct {
	Width  uint64 `c:"width"`
	Height uint64 `c:"height"`
	Depth  uint64 `c:"depth"`
}

// mtlRegion is MTLRegion: a box of a texture.
type mtlRegion struct {
	Origin mtlOrigin `c:"origin"`
	Size   mtlSize   `c:"size"`
}

// mtlClearColor is MTLClearColor: the color that a pass clears an attachment to.
type mtlClearColor struct {
	Red   float64 `c:"red"`
	Green float64 `c:"green"`
	Blue  float64 `c:"blue"`
	Alpha float64 `c:"alpha"`
}

// mtlScissorRect is MTLScissorRect: a scissor rectangle in pixels.
type mtlScissorRect struct {
	X      uint64 `c:"x"`
	Y      uint64 `c:"y"`
	Width  uint64 `c:"width"`
	Height uint64 `c:"height"`
}

// The structures of Core Graphics and Foundation come from package darwin, which has their layout test.
type (
	// nsRange is NSRange: the levels and slices of a texture view.
	nsRange = darwin.NSRange
	// cgSize is CGSize: the drawable size of a layer.
	cgSize = darwin.CGSize
	// cgRect is CGRect: the frame of a layer.
	cgRect = darwin.CGRect
)
