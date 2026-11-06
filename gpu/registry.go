package gpu

import "github.com/rm4n0s/glutter/internal/gpu/explicit"

// RegisterTexture lets the windows show the texture: it returns the id that a Texture widget, a TextureBox or
// SceneBuilder.AddTexture names (SPEC §27.4; Flutter's embedder registers textures the same way). The renderer samples
// the texture where the widget is, scaled to its rectangle and premultiplied, with the commands that wrote it
// submitted before the frame's composite; nothing is copied. The texture is 2D with one sample and
// TextureUsageTextureBinding, and its format is RGBA8Unorm or BGRA8Unorm (sRGB-encoded values) or RGBA16Float
// (linear values, which the composite encodes to sRGB). A lost device registers nothing and returns 0; so does a
// texture that the renderer cannot show, in debug builds after a panic with GPUEncodingInvalid.
//
// Flutter: no counterpart (FlutterEngineRegisterExternalTexture)
func (d *Device) RegisterTexture(t *Texture) int64 {
	const call = "Device.RegisterTexture"
	d.assertThread(call)
	if d.lostErr != nil {
		return 0
	}
	misuse(t != nil && t.d == d, call, "the texture was made by this device")
	t.usable(call)
	misuse(t.desc.Kind == explicit.TextureKind2D && t.desc.SampleCount == 1, call, "the texture is a 2D texture of one sample")
	misuse(t.desc.Usage&explicit.TextureSampled != 0, call, "the texture has TextureUsageTextureBinding")
	switch t.desc.Format {
	case explicit.TextureFormatRGBA8Unorm, explicit.TextureFormatBGRA8Unorm, explicit.TextureFormatRGBA16Float:
	default:
		misuse(false, call, "the format of the texture is RGBA8Unorm, BGRA8Unorm or RGBA16Float", "format", int(t.desc.Format))
	}
	id := d.textures().Register(d.dev, t.exp)
	if d.registered == nil {
		d.registered = map[int64]*Texture{}
	}
	d.registered[id] = t
	return id
}

// UnregisterTexture stops the windows from showing the texture with the id; a layer that names it draws nothing
// then. An id that is not registered is ignored. Releasing the texture unregisters it too.
//
// Flutter: no counterpart (FlutterEngineUnregisterExternalTexture)
func (d *Device) UnregisterTexture(id int64) {
	d.assertThread("Device.UnregisterTexture")
	delete(d.registered, id)
	if d.lostErr != nil {
		return
	}
	d.textures().Unregister(id)
}

// MarkTextureFrameAvailable says that the app wrote the texture with the id since the last frame: it schedules a
// frame, and the layers that show the texture are damaged, so that the compositor redraws their rectangles. Call it
// after the commands that wrote the texture were submitted. An id that is not registered is ignored.
//
// Flutter: no counterpart (FlutterEngineMarkExternalTextureFrameAvailable)
func (d *Device) MarkTextureFrameAvailable(id int64) {
	d.assertThread("Device.MarkTextureFrameAvailable")
	if d.lostErr != nil {
		return
	}
	if d.textures().MarkFrameAvailable(id) {
		d.st.b.Scheduler().ScheduleFrame()
	}
}

// textures is the registry the host gives.
func (d *Device) textures() *explicit.TextureRegistry { return d.st.host.GPUTextures() }

// unregisterAll forgets the registrations of a texture that is released.
func (d *Device) unregisterAll(t *Texture) {
	for id, r := range d.registered {
		if r == t {
			delete(d.registered, id)
			if d.lostErr == nil {
				d.textures().Unregister(id)
			}
		}
	}
}
