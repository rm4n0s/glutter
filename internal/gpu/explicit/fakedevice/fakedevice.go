package fakedevice

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// Call is one call the renderer made, in order. Op names the method and Args are
// its arguments in a readable form.
//
// Flutter: no counterpart
type Call struct {
	// Op is the name of the method: "NewBuffer", "BeginPass", "Draw", ...
	Op string
	// Args are the arguments, as text.
	Args string
}

func (c Call) String() string { return c.Op + "(" + c.Args + ")" }

// Device is the fake device. The zero value is not usable; use New.
//
// Flutter: no counterpart
type Device struct {
	limits explicit.Limits
	calls  []Call
	reads  []pendingRead
	closed bool
	// Err, when set, is what the next call that can fail returns, once.
	Err error
	// AcquireErrs are the errors that the next calls of Swapchain.Acquire return, one
	// each, before it succeeds again.
	AcquireErrs []error
	// Rotation is the pre-rotation that the swapchains report after Configure, as Android's
	// currentTransform does: with a quarter turn their images have the width and height swapped.
	Rotation explicit.Rotation
	// StaleOnce makes the next Present leave its swapchain stale, as Android's surface turning
	// does: Stale is true until Configure, which then reports RotationAfter.
	StaleOnce     bool
	RotationAfter explicit.Rotation
	// CompleteReadsOnSubmit makes Submit run the done functions of the reads of the frame,
	// as a device that waits for the GPU does.
	CompleteReadsOnSubmit bool
	// Name is what Description says.
	Name string
	// Writes are the calls of WriteBuffer, with a copy of their data.
	Writes []BufferWrite
}

// BufferWrite is one call of WriteBuffer.
//
// Flutter: no counterpart
type BufferWrite struct {
	// Offset is where the data went in the buffer.
	Offset int
	// Data is a copy of the bytes written.
	Data []byte
	// Usage is what the buffer was made for.
	Usage explicit.BufferUsage
}

var _ explicit.Device = (*Device)(nil)

// New returns a device with the design limits.
//
// Flutter: no counterpart
func New() *Device { return &Device{limits: explicit.DefaultLimits()} }

// Calls returns the calls made so far.
//
// Flutter: no counterpart
func (d *Device) Calls() []Call { return slices.Clone(d.calls) }

// Reset forgets the calls and the buffer writes made so far, for a benchmark that draws many
// frames.
//
// Flutter: no counterpart
func (d *Device) Reset() {
	d.calls = d.calls[:0]
	d.Writes = d.Writes[:0]
}

// Ops returns the names of the calls made so far.
//
// Flutter: no counterpart
func (d *Device) Ops() []string {
	ops := make([]string, len(d.calls))
	for i, c := range d.calls {
		ops[i] = c.Op
	}
	return ops
}

// Description names the device, for the renderer's log.
//
// Flutter: no counterpart
func (d *Device) Description() string { return "fake: " + d.Name }

// Closed is whether Close was called.
//
// Flutter: no counterpart
func (d *Device) Closed() bool { return d.closed }

func (d *Device) record(op, format string, args ...any) {
	d.calls = append(d.calls, Call{Op: op, Args: fmt.Sprintf(format, args...)})
}

// fail returns the error set in Err and clears it.
func (d *Device) fail() error {
	err := d.Err
	d.Err = nil
	return err
}

// Limits implements explicit.Device.
func (d *Device) Limits() explicit.Limits { return d.limits }

// SetLimits changes the limits the device checks, to test code that reads them.
//
// Flutter: no counterpart
func (d *Device) SetLimits(l explicit.Limits) { d.limits = l }

// NewBuffer implements explicit.Device.
func (d *Device) NewBuffer(size int, usage explicit.BufferUsage) (explicit.Buffer, error) {
	d.record("NewBuffer", "size=%d usage=%d", size, usage)
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &buffer{data: make([]byte, size), usage: usage}, nil
}

// NewTexture implements explicit.Device.
func (d *Device) NewTexture(desc explicit.TextureDesc) (explicit.Texture, error) {
	d.record("NewTexture", "%dx%d format=%d levels=%d usage=%d", desc.Width, desc.Height, desc.Format, desc.MipLevels, desc.Usage)
	d.limits.CheckTexture(desc)
	if err := d.fail(); err != nil {
		return nil, err
	}
	if desc.MipLevels < 1 {
		desc.MipLevels = 1
	}
	bpp := desc.Format.BytesPerPixel()
	t := &texture{desc: desc, bpp: bpp}
	if bpp > 0 && desc.Width > 0 && desc.Height > 0 {
		t.pix = make([]byte, desc.Width*desc.Height*bpp)
	}
	return t, nil
}

// NewSampler implements explicit.Device.
func (d *Device) NewSampler(desc explicit.SamplerDesc) (explicit.Sampler, error) {
	d.record("NewSampler", "%+v", desc)
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &sampler{}, nil
}

// NewPipeline implements explicit.Device.
func (d *Device) NewPipeline(desc explicit.PipelineDesc) (explicit.Pipeline, error) {
	name := "<nil>"
	if desc.Shader != nil {
		name = desc.Shader.Name
	}
	d.record("NewPipeline", "shader=%s vertex=%s fragment=%s buffers=%d target=%d", name, desc.VertexEntry, desc.FragmentEntry, len(desc.VertexBuffers), desc.Target)
	d.limits.CheckPipeline(desc)
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &pipeline{desc: desc}, nil
}

// NewBindGroup implements explicit.Device.
func (d *Device) NewBindGroup(p explicit.Pipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.record("NewBindGroup", "group=%d entries=%d", group, len(entries))
	d.limits.CheckBindGroup(group, entries)
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &bindGroup{}, nil
}

// WriteBuffer implements explicit.Device.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	d.record("WriteBuffer", "offset=%d bytes=%d", offset, len(data))
	var usage explicit.BufferUsage
	if fb, ok := b.(*buffer); ok {
		usage = fb.usage
	}
	d.Writes = append(d.Writes, BufferWrite{Offset: offset, Data: slices.Clone(data), Usage: usage})
	if fb, ok := b.(*buffer); ok && offset >= 0 && offset+len(data) <= len(fb.data) {
		copy(fb.data[offset:], data)
	}
}

// WriteTexture implements explicit.Device. It keeps the pixels of every level.
func (d *Device) WriteTexture(t explicit.Texture, level int, r image.Rectangle, data []byte, stride int) {
	d.record("WriteTexture", "level=%d rect=%v bytes=%d stride=%d", level, r, len(data), stride)
	ft, ok := t.(*texture)
	if !ok {
		return
	}
	if level != 0 {
		if level >= ft.desc.MipLevels {
			return
		}
		ft = ft.Level(level).(*texture)
	}
	if ft.pix == nil {
		return
	}
	r = r.Intersect(image.Rect(0, 0, ft.desc.Width, ft.desc.Height))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		src := data[(y-r.Min.Y)*stride:]
		n := r.Dx() * ft.bpp
		if len(src) < n {
			return
		}
		copy(ft.pix[(y*ft.desc.Width+r.Min.X)*ft.bpp:], src[:n])
	}
}

// NewSwapchain implements explicit.Device.
func (d *Device) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	d.record("NewSwapchain", "kind=%s", h.Kind)
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &swapchain{d: d}, nil
}

// Begin implements explicit.Device.
func (d *Device) Begin() (explicit.Encoder, error) {
	d.record("Begin", "")
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &encoder{d: d}, nil
}

// Close implements explicit.Device.
func (d *Device) Close() error {
	d.record("Close", "")
	d.closed = true
	return d.fail()
}

// pendingRead is a ReadTexture whose done has not run.
type pendingRead struct {
	src  *texture
	r    image.Rectangle
	done func(*image.RGBA, error)
}

// CompleteReads runs the done function of every ReadTexture that is waiting, as the
// GPU finishing the frame would. The real backends never call it inside the frame
// that asked, and neither does this: the test calls it after Submit.
//
// Flutter: no counterpart
func (d *Device) CompleteReads() {
	reads := d.reads
	d.reads = nil
	for _, rd := range reads {
		rd.done(rd.src.read(rd.r), nil)
	}
}

type buffer struct {
	data  []byte
	usage explicit.BufferUsage
}

func (b *buffer) Size() int { return len(b.data) }
func (b *buffer) Destroy()  {}

// Bytes returns what was written to the buffer, if b is a fake buffer.
//
// Flutter: no counterpart
func Bytes(b explicit.Buffer) []byte {
	if fb, ok := b.(*buffer); ok {
		return fb.data
	}
	return nil
}

type texture struct {
	desc   explicit.TextureDesc
	bpp    int
	pix    []byte     // level 0
	levels []*texture // the textures of Level, indexed by level
	parent *texture
}

func (t *texture) Desc() explicit.TextureDesc { return t.desc }
func (t *texture) Destroy()                   {}

// Level implements explicit.Texture: a texture of its own, of the size of the level.
func (t *texture) Level(i int) explicit.Texture {
	if i < 0 || i >= t.desc.MipLevels {
		panic(errors.New(tagFakeLevelInvalid, "the texture has no such mip level", "level", i, "levels", t.desc.MipLevels))
	}
	if t.desc.MipLevels == 1 || t.parent != nil {
		return t
	}
	if t.levels == nil {
		t.levels = make([]*texture, t.desc.MipLevels)
	}
	if t.levels[i] == nil {
		desc := t.desc
		desc.Width, desc.Height, desc.MipLevels = max(desc.Width>>i, 1), max(desc.Height>>i, 1), 1
		lv := &texture{desc: desc, bpp: t.bpp, parent: t}
		if i == 0 {
			lv = t // level 0 is the texture's own pixels
		} else if t.bpp > 0 {
			lv.pix = make([]byte, desc.Width*desc.Height*t.bpp)
		}
		t.levels[i] = lv
	}
	return t.levels[i]
}

// fill sets every pixel to the color, as a clearing pass does.
func (t *texture) fill(c ui.Color) {
	if t.pix == nil {
		return
	}
	r, g, b, a := unorm8(c.R), unorm8(c.G), unorm8(c.B), unorm8(c.A)
	px := []byte{r, g, b, a}
	switch {
	case t.bpp == 1:
		px = px[3:] // R8 holds the alpha, as coverage
	case t.desc.Format == explicit.TextureFormatBGRA8Unorm:
		px = []byte{b, g, r, a}
	}
	for i := 0; i+t.bpp <= len(t.pix); i += t.bpp {
		copy(t.pix[i:], px)
	}
}

// unorm8 converts a channel from 0 to 1 to a byte, rounding as the GPU does.
func unorm8(v float32) byte {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return byte(v*255 + 0.5)
}

// read returns the rectangle as RGBA; an R8 texture reads as gray.
func (t *texture) read(r image.Rectangle) *image.RGBA {
	r = r.Intersect(image.Rect(0, 0, t.desc.Width, t.desc.Height))
	img := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := (y*t.desc.Width + x) * t.bpp
			var c color.RGBA
			switch {
			case t.pix == nil:
			case t.bpp == 1:
				c = color.RGBA{t.pix[i], t.pix[i], t.pix[i], 255}
			case t.desc.Format == explicit.TextureFormatBGRA8Unorm:
				c = color.RGBA{t.pix[i+2], t.pix[i+1], t.pix[i], t.pix[i+3]}
			default:
				c = color.RGBA{t.pix[i], t.pix[i+1], t.pix[i+2], t.pix[i+3]}
			}
			img.SetRGBA(x-r.Min.X, y-r.Min.Y, c)
		}
	}
	return img
}

type sampler struct{}

func (*sampler) Destroy() {}

type pipeline struct{ desc explicit.PipelineDesc }

func (p *pipeline) Desc() explicit.PipelineDesc { return p.desc }
func (*pipeline) Destroy()                      {}

type bindGroup struct{}

func (*bindGroup) Destroy() {}

type swapchain struct {
	d     *Device
	w, h  int
	tex   *texture
	stale bool
}

func (s *swapchain) Configure(width, height int) error {
	s.d.record("Configure", "%dx%d", width, height)
	if err := s.d.fail(); err != nil {
		return err
	}
	s.w, s.h, s.tex = width, height, nil
	if s.stale {
		s.stale = false
		s.d.Rotation = s.d.RotationAfter
	}
	if s.d.Rotation == explicit.Rotation90 || s.d.Rotation == explicit.Rotation270 {
		s.w, s.h = height, width
	}
	return nil
}

func (s *swapchain) PreRotation() explicit.Rotation { return s.d.Rotation }

func (s *swapchain) Stale() bool { return s.stale }

func (s *swapchain) Acquire() (explicit.Texture, error) {
	s.d.record("Acquire", "")
	if len(s.d.AcquireErrs) > 0 {
		err := s.d.AcquireErrs[0]
		s.d.AcquireErrs = s.d.AcquireErrs[1:]
		return nil, err
	}
	if err := s.d.fail(); err != nil {
		return nil, err
	}
	if s.tex == nil {
		desc := explicit.TextureDesc{Width: s.w, Height: s.h, Format: explicit.TextureFormatBGRA8Unorm, MipLevels: 1, Usage: explicit.TextureRenderTarget}
		s.d.limits.CheckTexture(desc)
		s.tex = &texture{desc: desc, bpp: 4, pix: make([]byte, s.w*s.h*4)}
	}
	return s.tex, nil
}

func (s *swapchain) Present(damage []image.Rectangle) error {
	s.d.record("Present", "damage=%v", damage)
	if s.d.StaleOnce {
		s.d.StaleOnce, s.stale = false, true
	}
	return s.d.fail()
}

func (s *swapchain) Destroy() { s.d.record("DestroySwapchain", "") }

type encoder struct {
	d         *Device
	pass      *pass
	submitted bool
}

func (e *encoder) BeginPass(target explicit.Texture, load explicit.LoadOp, clear ui.Color) explicit.Pass {
	e.d.record("BeginPass", "target=%v load=%d clear=%v", target.Desc().Format, load, clear)
	if e.pass != nil && !e.pass.ended {
		panic(errors.New(tagFakePassNotEnded, "BeginPass was called before the previous pass ended"))
	}
	if t, ok := target.(*texture); ok && (load == explicit.LoadClear || load == explicit.LoadDontCare) {
		t.fill(clear)
	}
	e.pass = &pass{d: e.d}
	return e.pass
}

func (e *encoder) CopyTexture(src explicit.Texture, r image.Rectangle, dst explicit.Texture, at image.Point) {
	e.d.record("CopyTexture", "rect=%v at=%v", r, at)
	s, ok1 := src.(*texture)
	t, ok2 := dst.(*texture)
	if !ok1 || !ok2 || s.pix == nil || t.pix == nil || s.bpp != t.bpp {
		return
	}
	img := s.read(r)
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			dx, dy := at.X+x, at.Y+y
			if dx < 0 || dy < 0 || dx >= t.desc.Width || dy >= t.desc.Height {
				continue
			}
			c := img.RGBAAt(x, y)
			i := (dy*t.desc.Width + dx) * t.bpp
			switch {
			case t.bpp == 1:
				t.pix[i] = c.R
			case t.desc.Format == explicit.TextureFormatBGRA8Unorm:
				t.pix[i], t.pix[i+1], t.pix[i+2], t.pix[i+3] = c.B, c.G, c.R, c.A
			default:
				t.pix[i], t.pix[i+1], t.pix[i+2], t.pix[i+3] = c.R, c.G, c.B, c.A
			}
		}
	}
}

func (e *encoder) ReadTexture(src explicit.Texture, r image.Rectangle, done func(*image.RGBA, error)) {
	e.d.record("ReadTexture", "rect=%v", r)
	if t, ok := src.(*texture); ok {
		e.d.reads = append(e.d.reads, pendingRead{src: t, r: r, done: done})
	}
}

func (e *encoder) Submit() error {
	e.d.record("Submit", "")
	e.submitted = true
	err := e.d.fail()
	if e.d.CompleteReadsOnSubmit {
		e.d.CompleteReads()
	}
	return err
}

type pass struct {
	d     *Device
	ended bool
}

func (p *pass) SetPipeline(pl explicit.Pipeline) {
	name := "<nil>"
	if s := pl.Desc().Shader; s != nil {
		name = s.Name
	}
	p.d.record("SetPipeline", "shader=%s", name)
}

func (p *pass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	p.d.record("SetBindGroup", "group=%d offsets=%v", group, dynamicOffsets)
	p.d.limits.CheckSetBindGroup(group, dynamicOffsets)
}

func (p *pass) SetVertexBuffer(slot int, b explicit.Buffer, offset int) {
	p.d.record("SetVertexBuffer", "slot=%d offset=%d", slot, offset)
	p.d.limits.CheckSetVertexBuffer(slot)
}

func (p *pass) SetIndexBuffer(b explicit.Buffer, offset int) {
	p.d.record("SetIndexBuffer", "offset=%d", offset)
}

func (p *pass) SetScissor(r image.Rectangle) { p.d.record("SetScissor", "%v", r) }

func (p *pass) Draw(vertices, instances, firstVertex, firstInstance int) {
	p.d.record("Draw", "vertices=%d instances=%d firstVertex=%d firstInstance=%d", vertices, instances, firstVertex, firstInstance)
}

func (p *pass) DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int) {
	p.d.record("DrawIndexed", "indices=%d instances=%d firstIndex=%d baseVertex=%d firstInstance=%d", indices, instances, firstIndex, baseVertex, firstInstance)
}

func (p *pass) End() {
	p.d.record("EndPass", "")
	p.ended = true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFakeLevelInvalid = "FakeLevelInvalid"
	tagFakePassNotEnded = "FakePassNotEnded"
)
