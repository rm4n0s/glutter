package gles

import (
	"fmt"

	"github.com/rm4n0s/errors"
)

// TextureFormat is the pixel format of a texture.
//
// Flutter: no counterpart
type TextureFormat int

// The texture formats.
const (
	// TextureFormatUnspecified is the zero value: TextureFormatRGBA8.
	//
	// Flutter: no counterpart
	TextureFormatUnspecified TextureFormat = iota
	// TextureFormatRGBA8 is 8 bits per channel, premultiplied alpha.
	//
	// Flutter: no counterpart
	TextureFormatRGBA8
	// TextureFormatR8 is one 8-bit channel: coverage masks.
	//
	// Flutter: no counterpart
	TextureFormatR8
)

// glFormats returns the internal format, the format and the type of pixel data
// for a texture format, and the bytes per pixel.
func (f TextureFormat) glFormats() (internal int32, format, kind uint32, bpp int) {
	if f == TextureFormatR8 {
		return glR8, glRed, glUnsignedByte, 1
	}
	return glRgba8, glRgba, glUnsignedByte, 4
}

type textureKey struct {
	w, h   int
	format TextureFormat
}

// Texture is a 2D texture. Get one from Device.AcquireTexture and give it back
// with Device.ReleaseTexture.
//
// Flutter: no counterpart
type Texture struct {
	id       uint32
	w, h     int
	format   TextureFormat
	dev      *Device
	inUse    bool
	lastUsed uint64
	// The sampling parameters last set, to skip redundant calls.
	minFilter, magFilter, wrapS, wrapT int32
	attached                           bool // the color attachment of a pooled framebuffer
}

// Width is the width in pixels.
//
// Flutter: no counterpart
func (t *Texture) Width() int { return t.w }

// Height is the height in pixels.
//
// Flutter: no counterpart
func (t *Texture) Height() int { return t.h }

// Format is the pixel format.
//
// Flutter: no counterpart
func (t *Texture) Format() TextureFormat { return t.format }

// ID is the GL name of the texture.
//
// Flutter: no counterpart
func (t *Texture) ID() uint32 { return t.id }

// AcquireTexture returns a texture of the size and format, reusing a free one of
// the same size and format if there is one. Its contents are undefined. Its
// sampling is linear with clamping to the edge. It fails with TextureSizeInvalid
// for a size that is not positive or larger than the driver allows, and GLError
// when the driver cannot allocate it.
//
// Flutter: no counterpart
func (d *Device) AcquireTexture(w, h int, format TextureFormat) (*Texture, error) {
	if format == TextureFormatUnspecified {
		format = TextureFormatRGBA8
	}
	t, err := d.newTexture(w, h, format)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (d *Device) newTexture(w, h int, format TextureFormat) (*Texture, error) {
	if w <= 0 || h <= 0 || w > d.caps.MaxTextureSize || h > d.caps.MaxTextureSize {
		return nil, errors.New("TextureSizeInvalid", "the texture size is not valid for this driver",
			"width", w, "height", h, "max", d.caps.MaxTextureSize)
	}
	key := textureKey{w, h, format}
	if free := d.textures[key]; len(free) > 0 {
		t := free[len(free)-1]
		d.textures[key] = free[:len(free)-1]
		t.inUse = true
		return t, nil
	}
	internal, glFormat, kind, _ := format.glFormats()
	t := &Texture{id: d.gl.genTexture(), w: w, h: h, format: format, dev: d, inUse: true}
	d.bindTextureID(0, t.id)
	d.gl.texImage2D(glTexture2D, 0, internal, int32(w), int32(h), glFormat, kind, nil)
	if err := d.checkError("texImage2D"); err != nil {
		d.bindTextureID(0, 0)
		d.gl.deleteTexture(t.id)
		return nil, err
	}
	d.SetTextureParams(t, glLinear, glLinear, glClampToEdge, glClampToEdge)
	d.live.textures++
	return t, nil
}

// ReleaseTexture gives a texture back to the pool. Using it afterwards is a
// programmer error; releasing it twice panics with the tag TextureReleased.
//
// Flutter: no counterpart
func (d *Device) ReleaseTexture(t *Texture) {
	if !t.inUse {
		panic(errors.New("TextureReleased", "the texture was already released", "id", t.id))
	}
	t.inUse = false
	t.lastUsed = d.frame
	key := textureKey{t.w, t.h, t.format}
	d.textures[key] = append(d.textures[key], t)
}

// deleteTexture frees the GL texture. Any binding of it is cleared in the cache.
func (d *Device) deleteTexture(t *Texture) {
	for i := range d.state.texture {
		if d.state.texture[i] == t.id {
			d.state.texture[i] = 0
		}
	}
	d.gl.deleteTexture(t.id)
	d.live.textures--
}

// SetTextureParams sets the filters and the wrapping of a texture, skipping what
// is already set. The min filter may be one of the mipmap filters when the
// texture has mipmaps.
//
// Flutter: no counterpart
func (d *Device) SetTextureParams(t *Texture, minFilter, magFilter, wrapS, wrapT int32) {
	d.bindTextureID(0, t.id)
	if t.minFilter != minFilter {
		t.minFilter = minFilter
		d.gl.texParameteri(glTexture2D, glTextureMinFilter, minFilter)
	}
	if t.magFilter != magFilter {
		t.magFilter = magFilter
		d.gl.texParameteri(glTexture2D, glTextureMagFilter, magFilter)
	}
	if t.wrapS != wrapS {
		t.wrapS = wrapS
		d.gl.texParameteri(glTexture2D, glTextureWrapS, wrapS)
	}
	if t.wrapT != wrapT {
		t.wrapT = wrapT
		d.gl.texParameteri(glTexture2D, glTextureWrapT, wrapT)
	}
}

// UploadTexture writes the w x h pixels at x, y of the texture. The pixels are
// tightly packed, in the texture's format. It fails with TextureUploadInvalid
// when the rectangle is outside the texture or the data has the wrong length.
//
// Flutter: no counterpart
func (d *Device) UploadTexture(t *Texture, x, y, w, h int, pixels []byte) error {
	_, glFormat, kind, bpp := t.format.glFormats()
	if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > t.w || y+h > t.h || len(pixels) != w*h*bpp {
		return errors.New("TextureUploadInvalid", "the rectangle or the data does not fit the texture",
			"x", x, "y", y, "width", w, "height", h, "texture", fmt.Sprintf("%dx%d", t.w, t.h), "length", len(pixels))
	}
	d.bindTextureID(0, t.id)
	d.gl.texSubImage2D(glTexture2D, 0, int32(x), int32(y), int32(w), int32(h), glFormat, kind, pixels)
	return d.checkError("texSubImage2D")
}

// CopyFramebuffer copies the w x h pixels at x, y (counted from the bottom left)
// of the bound framebuffer into the corner of the texture where its first row
// starts, after binding the texture to the unit. The texture must be at least that
// big and RGBA8.
//
// Flutter: no counterpart
func (d *Device) CopyFramebuffer(unit int, t *Texture, x, y, w, h int) error {
	d.bindTextureID(unit, t.id)
	if d.state.activeUnit != uint32(unit) {
		d.state.activeUnit = uint32(unit)
		d.gl.activeTexture(glTexture0 + uint32(unit))
	}
	d.gl.copyTexSubImage2D(glTexture2D, 0, 0, 0, int32(x), int32(y), int32(w), int32(h))
	return d.checkError("copyTexSubImage2D")
}

// GenerateMipmaps builds the mipmap chain of a texture from its level 0.
//
// Flutter: no counterpart
func (d *Device) GenerateMipmaps(t *Texture) {
	d.bindTextureID(0, t.id)
	d.gl.generateMipmap(glTexture2D)
}

type bufferKey struct {
	target uint32
	size   int
}

// Buffer is a GL buffer whose capacity is a power of two, so that buffers of
// similar sizes are interchangeable. Get one from Device.AcquireBuffer.
//
// Flutter: no counterpart
type Buffer struct {
	id       uint32
	target   uint32
	capacity int
	dev      *Device
	inUse    bool
	lastUsed uint64
}

// Capacity is the number of bytes the buffer can hold.
//
// Flutter: no counterpart
func (b *Buffer) Capacity() int { return b.capacity }

// ID is the GL name of the buffer.
//
// Flutter: no counterpart
func (b *Buffer) ID() uint32 { return b.id }

// minBufferSize is the smallest buffer capacity.
const minBufferSize = 256

// AcquireBuffer returns a buffer for target (glArrayBuffer or
// glElementArrayBuffer) of at least size bytes, reusing a free one of the same
// capacity class if there is one.
//
// Flutter: no counterpart
func (d *Device) AcquireBuffer(target uint32, size int) (*Buffer, error) {
	capacity := minBufferSize
	for capacity < size {
		capacity *= 2
	}
	key := bufferKey{target, capacity}
	if free := d.buffers[key]; len(free) > 0 {
		b := free[len(free)-1]
		d.buffers[key] = free[:len(free)-1]
		b.inUse = true
		return b, nil
	}
	b := &Buffer{id: d.gl.genBuffer(), target: target, capacity: capacity, dev: d, inUse: true}
	d.bindBufferTarget(target, b.id)
	d.gl.bufferData(target, capacity, nil, glStreamDraw)
	if err := d.checkError("bufferData"); err != nil {
		d.gl.deleteBuffer(b.id)
		return nil, err
	}
	d.live.buffers++
	return b, nil
}

// bindBufferTarget binds a buffer to a target, with the cache for the array
// buffer; the element buffer is part of the vertex array's state and is always
// bound.
func (d *Device) bindBufferTarget(target, id uint32) {
	if target == glArrayBuffer {
		if d.state.arrayBuffer == id {
			return
		}
		d.state.arrayBuffer = id
	}
	d.gl.bindBuffer(target, id)
}

// ReleaseBuffer gives a buffer back to the pool; releasing it twice panics with
// the tag BufferReleased.
//
// Flutter: no counterpart
func (d *Device) ReleaseBuffer(b *Buffer) {
	if !b.inUse {
		panic(errors.New("BufferReleased", "the buffer was already released", "id", b.id))
	}
	b.inUse = false
	b.lastUsed = d.frame
	key := bufferKey{b.target, b.capacity}
	d.buffers[key] = append(d.buffers[key], b)
}

func (d *Device) deleteBuffer(b *Buffer) {
	if d.state.arrayBuffer == b.id {
		d.state.arrayBuffer = 0
	}
	d.gl.deleteBuffer(b.id)
	d.live.buffers--
}

// UploadBuffer writes data at the start of the buffer. The old contents are
// orphaned first, so that the driver need not wait for draws that still read them.
// It fails with BufferTooSmall when data is longer than the capacity.
//
// Flutter: no counterpart
func (d *Device) UploadBuffer(b *Buffer, data []byte) error {
	if len(data) > b.capacity {
		return errors.New("BufferTooSmall", "the data does not fit the buffer", "length", len(data), "capacity", b.capacity)
	}
	d.bindBufferTarget(b.target, b.id)
	d.gl.bufferData(b.target, b.capacity, nil, glStreamDraw)
	if len(data) > 0 {
		d.gl.bufferSubData(b.target, 0, data)
	}
	return d.checkError("bufferSubData")
}

type framebufferKey struct {
	w, h    int
	stencil bool
}

// Framebuffer is an offscreen render target: an RGBA8 texture, and optionally a
// stencil buffer. Get one from Device.AcquireFramebuffer.
//
// Flutter: no counterpart
type Framebuffer struct {
	id        uint32
	w, h      int
	stencil   bool
	stencilRB uint32
	tex       *Texture
	dev       *Device
	inUse     bool
	lastUsed  uint64
}

// Texture is the color attachment, which can be sampled once drawing is done.
//
// Flutter: no counterpart
func (f *Framebuffer) Texture() *Texture { return f.tex }

// Width is the width in pixels.
//
// Flutter: no counterpart
func (f *Framebuffer) Width() int { return f.w }

// Height is the height in pixels.
//
// Flutter: no counterpart
func (f *Framebuffer) Height() int { return f.h }

// ID is the GL name of the framebuffer.
//
// Flutter: no counterpart
func (f *Framebuffer) ID() uint32 { return f.id }

// AcquireFramebuffer returns an offscreen framebuffer of the size, with a stencil
// buffer if asked, reusing a free one of the same size. It fails with
// TextureSizeInvalid for a bad size and FramebufferIncomplete when the driver
// cannot make it.
//
// Flutter: no counterpart
func (d *Device) AcquireFramebuffer(w, h int, stencil bool) (*Framebuffer, error) {
	key := framebufferKey{w, h, stencil}
	if free := d.frames[key]; len(free) > 0 {
		f := free[len(free)-1]
		d.frames[key] = free[:len(free)-1]
		f.inUse = true
		return f, nil
	}
	tex, err := d.newTexture(w, h, TextureFormatRGBA8)
	if err != nil {
		return nil, err
	}
	tex.attached = true
	f := &Framebuffer{id: d.gl.genFramebuffer(), w: w, h: h, stencil: stencil, tex: tex, dev: d, inUse: true}
	prev := d.state.framebuffer
	d.bindFramebufferID(f.id)
	d.gl.framebufferTexture2D(glFramebuffer, glColorAttachment0, glTexture2D, tex.id, 0)
	if stencil {
		f.stencilRB = d.gl.genRenderbuffer()
		d.gl.bindRenderbuffer(glRenderbuffer, f.stencilRB)
		d.gl.renderbufferStorage(glRenderbuffer, glStencilIndex8, int32(w), int32(h))
		d.gl.framebufferRenderbuffer(glFramebuffer, glStencilAttachment, glRenderbuffer, f.stencilRB)
		d.gl.bindRenderbuffer(glRenderbuffer, 0)
		d.live.renderbuffers++
	}
	status := d.gl.checkFramebufferStatus(glFramebuffer)
	d.bindFramebufferID(prev)
	if status != glFramebufferComplete {
		d.deleteFramebuffer(f)
		return nil, errors.New("FramebufferIncomplete", "the driver could not make the framebuffer complete",
			"width", w, "height", h, "stencil", stencil, "status", fmt.Sprintf("0x%x", status))
	}
	d.live.framebuffers++
	return f, nil
}

// ReleaseFramebuffer gives a framebuffer back to the pool; releasing it twice
// panics with the tag FramebufferReleased.
//
// Flutter: no counterpart
func (d *Device) ReleaseFramebuffer(f *Framebuffer) {
	if !f.inUse {
		panic(errors.New("FramebufferReleased", "the framebuffer was already released", "id", f.id))
	}
	f.inUse = false
	f.lastUsed = d.frame
	key := framebufferKey{f.w, f.h, f.stencil}
	d.frames[key] = append(d.frames[key], f)
}

// deleteFramebuffer frees the framebuffer, its stencil buffer and its texture.
func (d *Device) deleteFramebuffer(f *Framebuffer) {
	if d.state.framebuffer == f.id {
		d.state.framebuffer = 0
		d.gl.bindFramebuffer(glFramebuffer, 0)
	}
	d.gl.deleteFramebuffer(f.id)
	if f.stencilRB != 0 {
		d.gl.deleteRenderbuffer(f.stencilRB)
		d.live.renderbuffers--
	}
	d.deleteTexture(f.tex)
	if f.id != 0 && d.live.framebuffers > 0 {
		d.live.framebuffers--
	}
}
