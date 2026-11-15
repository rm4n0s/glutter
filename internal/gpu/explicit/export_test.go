package explicit

import "github.com/rm4n0s/glutter/internal/gpu"

// SetMaskAtlasForTest gives the renderer a coverage atlas of pages of pageSize pixels that
// keeps kept pages and that a frame may make grow to frame pages, so that a test fills it with
// a few shapes. The renderer must have drawn a frame (its caches are made by the first) and no
// path yet.
//
// Flutter: no counterpart
func SetMaskAtlasForTest(r *Renderer, pageSize, kept, frame int) {
	r.res.masks.SetAtlas(gpu.NewAtlas[gpu.MaskKey](pageSize, kept).SetFrameLimit(frame))
}

// MaskAtlasStatsForTest says what the coverage atlas of the renderer holds.
//
// Flutter: no counterpart
func MaskAtlasStatsForTest(r *Renderer) gpu.AtlasStats { return r.res.masks.Atlas().Stats() }
