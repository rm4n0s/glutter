package explicit

import "sync"

// TextureRegistry is the table of the textures that an app shows in its windows with a TextureLayer (SPEC §27.4):
// package gpu registers a texture of the renderer's device and gets an id, the scene names the id, and the renderer
// looks the texture up when it draws the layer. A texture has a version that MarkFrameAvailable raises; the damage
// tracker hashes it, so that a frame in which the app wrote the texture damages the layer's rectangle. It is used
// on the UI thread (the lock is for the tests that read it from another goroutine).
//
// Flutter: no counterpart (the embedder's texture registry, FlutterEngineRegisterExternalTexture)
type TextureRegistry struct {
	mu      sync.Mutex
	next    int64
	entries map[int64]*registered
}

type registered struct {
	dev     Device
	tex     Texture
	version uint64
}

// NewTextureRegistry returns an empty registry.
//
// Flutter: no counterpart
func NewTextureRegistry() *TextureRegistry { return &TextureRegistry{entries: map[int64]*registered{}} }

// Register adds a texture of the device and returns its id, from 1 up. The id is not used again.
//
// Flutter: no counterpart (FlutterEngineRegisterExternalTexture)
func (r *TextureRegistry) Register(dev Device, tex Texture) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	r.entries[r.next] = &registered{dev: dev, tex: tex, version: 1}
	return r.next
}

// Unregister removes the texture and reports whether the id was registered.
//
// Flutter: no counterpart (FlutterEngineUnregisterExternalTexture)
func (r *TextureRegistry) Unregister(id int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.entries[id]
	delete(r.entries, id)
	return ok
}

// Lookup returns the texture with the id if it is registered and belongs to the device: a texture of a device that was
// lost is not drawn by the one that replaced it.
//
// Flutter: no counterpart
func (r *TextureRegistry) Lookup(dev Device, id int64) (Texture, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[id]
	if !ok || e.dev != dev {
		return nil, false
	}
	return e.tex, true
}

// MarkFrameAvailable says that the texture has a new frame, and reports whether the id is registered.
//
// Flutter: no counterpart (FlutterEngineMarkExternalTextureFrameAvailable)
func (r *TextureRegistry) MarkFrameAvailable(id int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[id]
	if ok {
		e.version++
	}
	return ok
}

// Version is a number that changes when the texture has a new frame; 0 for an id that is not registered.
//
// Flutter: no counterpart
func (r *TextureRegistry) Version(id int64) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[id]; ok {
		return e.version
	}
	return 0
}

// DropDevice forgets the textures of the device, which is gone.
//
// Flutter: no counterpart
func (r *TextureRegistry) DropDevice(dev Device) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, e := range r.entries {
		if e.dev == dev {
			delete(r.entries, id)
		}
	}
}

// Len is the number of registered textures.
//
// Flutter: no counterpart
func (r *TextureRegistry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}
