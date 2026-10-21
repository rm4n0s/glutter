package gles

import "github.com/rm4n0s/glutter/internal/gpu"

// The raster cache (SPEC §9.1) is the policy of gpu/layercache.go; here its textures are
// framebuffers, which the cache keeps, not returns to the pool, until it evicts the entry.

type (
	layerCache = gpu.LayerCache[*Framebuffer]
	layerEntry = gpu.LayerEntry[*Framebuffer]
	layerKey   = gpu.LayerKey
)

func newLayerCache(after int) *layerCache { return gpu.NewLayerCache[*Framebuffer](after) }

// keyOf makes the key of a layer's group, as gpu.LayerKeyOf does.
var keyOf = gpu.LayerKeyOf
