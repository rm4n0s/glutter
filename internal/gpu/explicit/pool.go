package explicit

import "github.com/rm4n0s/glutter/internal/gpu"

// The textures of layers are pooled by size (rounded up to a granule): a layer takes one that is
// free, and gives it back when it is composited, so that the layers of a frame, and of the frames
// after it, reuse the same few textures. A texture that no layer has taken for idleFrames is
// destroyed.

// pooledTexture is a free texture of a layer.
type pooledTexture struct {
	tex      Texture
	w, h     int
	lastUsed uint64
}

// layerTexture returns a texture of w x h pixels to draw a layer into, from the pool or new.
func (r *Renderer) layerTexture(w, h int) (Texture, error) {
	for i, p := range r.layerPool {
		if p.w == w && p.h == h {
			r.layerPool = append(r.layerPool[:i], r.layerPool[i+1:]...)
			return p.tex, nil
		}
	}
	return r.dev.NewTexture(TextureDesc{Width: w, Height: h, Format: TextureFormatRGBA8Unorm, MipLevels: 1,
		Usage: TextureRenderTarget | TextureSampled | TextureCopySource})
}

// releaseLayer gives the texture of a layer back to the pool.
func (r *Renderer) releaseLayer(t Texture) {
	d := t.Desc()
	var frame uint64
	if r.res != nil {
		frame = r.res.frame
	}
	r.layerPool = append(r.layerPool, &pooledTexture{tex: t, w: d.Width, h: d.Height, lastUsed: frame})
}

// sweepLayers destroys the textures that no layer has taken for a long time.
func (r *Renderer) sweepLayers() {
	if r.res == nil {
		return
	}
	kept := r.layerPool[:0]
	for _, p := range r.layerPool {
		if r.res.frame-p.lastUsed > gpu.IdleFrames/2 {
			p.tex.Destroy()
			continue
		}
		kept = append(kept, p)
	}
	r.layerPool = kept
}

// dstTexture returns the texture that holds a copy of the pixels under a draw that blends in the
// shader: one texture, of at least w x h pixels, made bigger when a draw needs more.
func (r *Renderer) dstTexture(w, h int) (Texture, error) {
	if r.dst != nil {
		if d := r.dst.Desc(); d.Width >= w && d.Height >= h {
			return r.dst, nil
		}
		w, h = max(w, r.dst.Desc().Width), max(h, r.dst.Desc().Height)
		r.dst.Destroy()
		r.dst = nil
	}
	t, err := r.dev.NewTexture(TextureDesc{Width: w, Height: h, Format: TextureFormatRGBA8Unorm, MipLevels: 1, Usage: TextureSampled})
	if err != nil {
		return nil, err
	}
	r.dst = t
	return t, nil
}

// destroyPools destroys the pooled textures.
func (r *Renderer) destroyPools() {
	for _, p := range r.layerPool {
		p.tex.Destroy()
	}
	r.layerPool = nil
	if r.dst != nil {
		r.dst.Destroy()
		r.dst = nil
	}
}
