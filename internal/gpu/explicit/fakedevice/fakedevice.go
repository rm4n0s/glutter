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
	// completions are the OnComplete functions of the submitted encoders that have not run.
	completions []func()
	closed      bool
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
	// Supported are the optional features that Features reports.
	Supported explicit.Features
	// Quiet stops the device from keeping the log of its calls and the copies of the buffer
	// writes, so that a benchmark of the renderer counts what the renderer allocates and not
	// what the log does. A quiet device also uses one encoder and one pass again and again, so
	// that its own calls allocate nothing (a frame that is checked for allocations is not
	// counted for what the fake needs to run it).
	Quiet bool

	quietEncoder encoder
	quietPass    pass
	spare        []func()

	live Live
}

// Live is how many resources of each kind the device made that were not destroyed yet. A
// renderer that keeps making resources and never destroys them (a texture for each frame) shows
// here as a count that grows. Bind groups are not counted: the real devices take them from the
// pool of a frame, which is reset when the frame is done, and their Destroy does nothing.
//
// Flutter: no counterpart
type Live struct {
	Buffers, Textures, Samplers, Pipelines int
}

// Live returns how many resources of each kind are alive.
//
// Flutter: no counterpart
func (d *Device) Live() Live { return d.live }

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
	if d.Quiet {
		return
	}
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

// Features implements explicit.Device: the features set in Supported.
func (d *Device) Features() explicit.Features { return d.Supported }

// SetLimits changes the limits the device checks, to test code that reads them.
//
// Flutter: no counterpart
func (d *Device) SetLimits(l explicit.Limits) { d.limits = l }

// NewBuffer implements explicit.Device.
func (d *Device) NewBuffer(size int, usage explicit.BufferUsage) (explicit.Buffer, error) {
	if !d.Quiet {
		d.record("NewBuffer", "size=%d usage=%d", size, usage)
	}
	d.limits.CheckBuffer(size, usage)
	if err := d.fail(); err != nil {
		return nil, err
	}
	d.live.Buffers++
	return &buffer{d: d, data: make([]byte, size), usage: usage}, nil
}

// NewTexture implements explicit.Device. A 2D texture keeps its pixels; a texture of another kind
// keeps the pixels of its layers, which Layer and WriteTextureRegion give access to.
func (d *Device) NewTexture(desc explicit.TextureDesc) (explicit.Texture, error) {
	if desc.Kind == explicit.TextureKindUnspecified && desc.DepthOrLayers == 0 && desc.SampleCount == 0 {
		if !d.Quiet {
			d.record("NewTexture", "%dx%d format=%d levels=%d usage=%d", desc.Width, desc.Height, desc.Format, desc.MipLevels, desc.Usage)
		}
	} else {
		if !d.Quiet {
			d.record("NewTexture", "%dx%d format=%d levels=%d usage=%d kind=%d depth_or_layers=%d samples=%d", desc.Width, desc.Height, desc.Format, desc.MipLevels, desc.Usage, desc.Kind, desc.DepthOrLayers, desc.SampleCount)
		}
	}
	d.limits.CheckTexture(desc)
	if err := d.fail(); err != nil {
		return nil, err
	}
	desc = desc.Normalized()
	bpp := desc.Format.BytesPerPixel()
	d.live.Textures++
	t := &texture{d: d, desc: desc, bpp: bpp}
	if bpp > 0 && desc.Width > 0 && desc.Height > 0 && desc.Kind == explicit.TextureKind2D && desc.SampleCount == 1 {
		t.pix = make([]byte, desc.Width*desc.Height*bpp)
	}
	return t, nil
}

// NewSampler implements explicit.Device.
func (d *Device) NewSampler(desc explicit.SamplerDesc) (explicit.Sampler, error) {
	if !d.Quiet {
		d.record("NewSampler", "%+v", desc)
	}
	if err := d.fail(); err != nil {
		return nil, err
	}
	d.live.Samplers++
	return &sampler{d: d}, nil
}

// NewPipeline implements explicit.Device.
func (d *Device) NewPipeline(desc explicit.PipelineDesc) (explicit.Pipeline, error) {
	name := "<nil>"
	if desc.Shader != nil {
		name = desc.Shader.Name
	}
	if !d.Quiet {
		d.record("NewPipeline", "shader=%s vertex=%s fragment=%s buffers=%d target=%d%s", name, desc.VertexEntry, desc.FragmentEntry, len(desc.VertexBuffers), desc.Target, stateText(desc))
	}
	d.limits.CheckPipeline(desc)
	if err := d.fail(); err != nil {
		return nil, err
	}
	d.live.Pipelines++
	return &pipeline{d: d, desc: desc}, nil
}

// NewBindGroup implements explicit.Device.
func (d *Device) NewBindGroup(p explicit.Pipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	if !d.Quiet {
		d.record("NewBindGroup", "group=%d entries=%d", group, len(entries))
	}
	d.limits.CheckBindGroup(group, entries)
	if fp, ok := p.(*pipeline); ok && fp.desc.Shader != nil {
		d.checkBindings(group, entries, fp.desc.Shader, fp.desc.VertexEntry, fp.desc.FragmentEntry)
	}
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &bindGroup{}, nil
}

// WriteBuffer implements explicit.Device.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	if !d.Quiet {
		d.record("WriteBuffer", "offset=%d bytes=%d", offset, len(data))
	}
	var usage explicit.BufferUsage
	if fb, ok := b.(*buffer); ok {
		usage = fb.usage
	}
	if !d.Quiet {
		d.Writes = append(d.Writes, BufferWrite{Offset: offset, Data: slices.Clone(data), Usage: usage})
	}
	if fb, ok := b.(*buffer); ok && offset >= 0 && offset+len(data) <= len(fb.data) {
		copy(fb.data[offset:], data)
	}
}

// WriteTexture implements explicit.Device. It keeps the pixels of every level.
func (d *Device) WriteTexture(t explicit.Texture, level int, r image.Rectangle, data []byte, stride int) {
	if !d.Quiet {
		d.record("WriteTexture", "level=%d rect=%v bytes=%d stride=%d", level, r, len(data), stride)
	}
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
	d.writeRows(ft, r, data, stride)
}

// WriteTextureRegion implements explicit.Device. It writes the pixels of a 2D array or a cube
// into their layers, and of a 3D texture into its slices (kept as one image of Width x Height x
// Depth rows).
func (d *Device) WriteTextureRegion(t explicit.Texture, region explicit.TextureRegion, data []byte, bytesPerRow, rowsPerImage int) {
	if !d.Quiet {
		d.record("WriteTextureRegion", "region=%+v bytes=%d bytesPerRow=%d rowsPerImage=%d", region, len(data), bytesPerRow, rowsPerImage)
	}
	ft, ok := t.(*texture)
	if !ok || ft.bpp == 0 {
		return
	}
	for z := 0; z < region.Size[2]; z++ {
		img := data[z*rowsPerImage*bytesPerRow:]
		var dst *texture
		switch ft.desc.Kind {
		case explicit.TextureKind3D:
			dst = ft.slice(region.Level, region.Origin[2]+z)
		default:
			dst = ft.Layer(region.Level, region.Origin[2]+z).(*texture)
		}
		r := image.Rect(region.Origin[0], region.Origin[1], region.Origin[0]+region.Size[0], region.Origin[1]+region.Size[1])
		d.writeRows(dst, r, img, bytesPerRow)
	}
}

// writeRows copies rows of data into the pixels of a 2D texture.
func (d *Device) writeRows(ft *texture, r image.Rectangle, data []byte, stride int) {
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
	if !d.Quiet {
		d.record("NewSwapchain", "kind=%s", h.Kind)
	}
	if err := d.fail(); err != nil {
		return nil, err
	}
	return &swapchain{d: d}, nil
}

// Begin implements explicit.Device.
func (d *Device) Begin() (explicit.Encoder, error) {
	if !d.Quiet {
		d.record("Begin", "")
	}
	if err := d.fail(); err != nil {
		return nil, err
	}
	if d.Quiet {
		e := &d.quietEncoder
		e.d, e.pass, e.cpass, e.submitted = d, nil, nil, false
		clear(e.complete)
		e.complete = e.complete[:0]
		return e, nil
	}
	return &encoder{d: d}, nil
}

// Close implements explicit.Device.
func (d *Device) Close() error {
	if !d.Quiet {
		d.record("Close", "")
	}
	d.closed = true
	return d.fail()
}

// pendingRead is a ReadTexture whose done has not run.
type pendingRead struct {
	src       *texture
	r         image.Rectangle
	done      func(*image.RGBA, error)
	doneBytes func([]byte, int, error)
	// buf, off and size are a ReadBuffer, with doneBuf.
	buf       *buffer
	off, size int
	doneBuf   func([]byte, error)
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
		if rd.doneBuf != nil {
			rd.doneBuf(append([]byte(nil), rd.buf.data[rd.off:rd.off+rd.size]...), nil)
			continue
		}
		if rd.doneBytes != nil {
			data, rowBytes := rd.src.readBytes(rd.r)
			rd.doneBytes(data, rowBytes, nil)
			continue
		}
		rd.done(rd.src.read(rd.r), nil)
	}
	done := d.completions
	d.completions = d.spare[:0]
	for _, fn := range done {
		fn()
	}
	// The list that was run is the next one's storage, so that a device that is checked for allocations makes none here.
	clear(done)
	d.spare = done[:0]
}

type buffer struct {
	d         *Device
	data      []byte
	usage     explicit.BufferUsage
	destroyed bool
}

func (b *buffer) Size() int                   { return len(b.data) }
func (b *buffer) Usage() explicit.BufferUsage { return b.usage }
func (b *buffer) Destroy() {
	if !b.destroyed {
		b.destroyed = true
		b.d.live.Buffers--
	}
}

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
	d         *Device // nil for the textures of the swapchain and of the mip levels
	destroyed bool
	desc      explicit.TextureDesc
	bpp       int
	pix       []byte     // level 0
	levels    []*texture // the textures of Level, indexed by level
	parent    *texture
	layers    map[[2]int]*texture // the textures of Layer and the slices of a 3D texture, by level and layer
}

func (t *texture) Desc() explicit.TextureDesc { return t.desc }
func (t *texture) Destroy() {
	if t.d != nil && !t.destroyed {
		t.destroyed = true
		t.d.live.Textures--
	}
}

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

// Layer implements explicit.Texture: a 2D texture of its own, of the size of the level.
func (t *texture) Layer(level, layer int) explicit.Texture {
	limits := explicit.DefaultLimits()
	if t.d != nil {
		limits = t.d.limits
	}
	limits.CheckLayer(t.desc, level, layer)
	if t.desc.Kind == explicit.TextureKind2D {
		return t.Level(level)
	}
	return t.slice(level, layer)
}

// slice returns the 2D texture of a layer of an array or a cube, or of a slice of a 3D texture, at a
// level.
func (t *texture) slice(level, layer int) *texture {
	key := [2]int{level, layer}
	if st, ok := t.layers[key]; ok {
		return st
	}
	if t.layers == nil {
		t.layers = map[[2]int]*texture{}
	}
	desc := t.desc
	desc.Kind, desc.DepthOrLayers = explicit.TextureKind2D, 1
	desc.Width, desc.Height, desc.MipLevels = max(desc.Width>>level, 1), max(desc.Height>>level, 1), 1
	st := &texture{desc: desc, bpp: t.bpp, parent: t}
	if t.bpp > 0 {
		st.pix = make([]byte, desc.Width*desc.Height*t.bpp)
	}
	t.layers[key] = st
	return st
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

// readBytes returns the rows of the rectangle as the bytes of the format.
func (t *texture) readBytes(r image.Rectangle) ([]byte, int) {
	r = r.Intersect(image.Rect(0, 0, t.desc.Width, t.desc.Height))
	row := r.Dx() * t.bpp
	out := make([]byte, row*r.Dy())
	if t.pix == nil {
		return out, row
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		copy(out[(y-r.Min.Y)*row:], t.pix[(y*t.desc.Width+r.Min.X)*t.bpp:][:row])
	}
	return out, row
}

type sampler struct {
	d         *Device
	destroyed bool
}

func (s *sampler) Destroy() {
	if !s.destroyed {
		s.destroyed = true
		s.d.live.Samplers--
	}
}

type pipeline struct {
	d         *Device
	desc      explicit.PipelineDesc
	destroyed bool
}

func (p *pipeline) Desc() explicit.PipelineDesc { return p.desc }
func (p *pipeline) Destroy() {
	if !p.destroyed {
		p.destroyed = true
		p.d.live.Pipelines--
	}
}

type bindGroup struct{}

func (*bindGroup) Destroy() {}

type swapchain struct {
	d     *Device
	w, h  int
	tex   *texture
	stale bool
}

func (s *swapchain) Configure(width, height int) error {
	if !s.d.Quiet {
		s.d.record("Configure", "%dx%d", width, height)
	}
	if err := s.d.fail(); err != nil {
		return err
	}
	// The images of the swapchain are kept while the size stays the same, as a device does.
	if s.w != width || s.h != height || s.stale {
		s.tex = nil
	}
	s.w, s.h = width, height
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
	if !s.d.Quiet {
		s.d.record("Acquire", "")
	}
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
	if !s.d.Quiet {
		s.d.record("Present", "damage=%v", damage)
	}
	if s.d.StaleOnce {
		s.d.StaleOnce, s.stale = false, true
	}
	return s.d.fail()
}

func (s *swapchain) Destroy() { s.d.record("DestroySwapchain", "") }

type encoder struct {
	d         *Device
	pass      *pass
	cpass     *computePass
	submitted bool
	complete  []func()
}

func (e *encoder) BeginPass(target explicit.Texture, load explicit.LoadOp, clear ui.Color) explicit.Pass {
	if !e.d.Quiet {
		e.d.record("BeginPass", "target=%v load=%d clear=%v", target.Desc().Format, load, clear)
	}
	if e.pass != nil && !e.pass.ended {
		panic(errors.New(tagFakePassNotEnded, "BeginPass was called before the previous pass ended"))
	}
	if t, ok := target.(*texture); ok && (load == explicit.LoadClear || load == explicit.LoadDontCare) {
		t.fill(clear)
	}
	e.pass = e.newPass()
	return e.pass
}

func (e *encoder) BeginRenderPass(desc explicit.PassDesc) explicit.Pass {
	if !e.d.Quiet {
		e.d.record("BeginRenderPass", "%s", passText(desc))
	}
	e.d.limits.CheckPass(desc)
	e.d.limits.CheckTimestamps(e.d.Supported, desc.Timestamps)
	if e.pass != nil && !e.pass.ended {
		panic(errors.New(tagFakePassNotEnded, "BeginRenderPass was called before the previous pass ended"))
	}
	for _, c := range desc.Color {
		// A quiet device keeps no pixels of a clear: building the bytes of one is an allocation.
		if t, ok := c.Target.(*texture); ok && !e.d.Quiet && (c.Load == explicit.LoadClear || c.Load == explicit.LoadDontCare) {
			t.fillBytes(encodeColor(t.desc.Format, c.Clear))
		}
	}
	e.pass = e.newPass()
	return e.pass
}

// newPass is the pass of a new BeginRenderPass: a quiet device keeps one.
func (e *encoder) newPass() *pass {
	if e.d.Quiet {
		e.d.quietPass = pass{d: e.d}
		return &e.d.quietPass
	}
	return &pass{d: e.d}
}

func (e *encoder) CopyTexture(src explicit.Texture, r image.Rectangle, dst explicit.Texture, at image.Point) {
	if !e.d.Quiet {
		e.d.record("CopyTexture", "rect=%v at=%v", r, at)
	}
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
	if !e.d.Quiet {
		e.d.record("ReadTexture", "rect=%v", r)
	}
	if t, ok := src.(*texture); ok {
		e.d.limits.CheckRead(t.desc, false)
		e.d.reads = append(e.d.reads, pendingRead{src: t, r: r, done: done})
	}
}

func (e *encoder) ReadTextureBytes(src explicit.Texture, r image.Rectangle, done func([]byte, int, error)) {
	if !e.d.Quiet {
		e.d.record("ReadTextureBytes", "rect=%v", r)
	}
	if t, ok := src.(*texture); ok {
		e.d.limits.CheckRead(t.desc, true)
		e.d.reads = append(e.d.reads, pendingRead{src: t, r: r, doneBytes: done})
	}
}

func (e *encoder) Submit() error {
	if !e.d.Quiet {
		e.d.record("Submit", "")
	}
	e.submitted = true
	e.d.completions = append(e.d.completions, e.complete...)
	err := e.d.fail()
	if e.d.CompleteReadsOnSubmit {
		e.d.CompleteReads()
	}
	return err
}

type pass struct {
	d     *Device
	ended bool
	pipe  explicit.Pipeline
}

func (p *pass) SetPipeline(pl explicit.Pipeline) {
	name := "<nil>"
	if s := pl.Desc().Shader; s != nil {
		name = s.Name
	}
	if !p.d.Quiet {
		p.d.record("SetPipeline", "shader=%s", name)
	}
	p.pipe = pl
}

// vertexEntry returns the vertex entry point of the pipeline that is set, if there is one.
func (p *pass) vertexEntry() (explicit.EntryPoint, bool) {
	if p.pipe == nil {
		return explicit.EntryPoint{}, false
	}
	d := p.pipe.Desc()
	if d.Shader == nil {
		return explicit.EntryPoint{}, false
	}
	return d.Shader.Entry(d.VertexEntry)
}

func (p *pass) SetBindGroup(group int, g explicit.BindGroup, dynamicOffsets ...uint32) {
	if !p.d.Quiet {
		p.d.record("SetBindGroup", "group=%d offsets=%v", group, dynamicOffsets)
	}
	p.d.limits.CheckSetBindGroup(group, dynamicOffsets)
}

func (p *pass) SetVertexBuffer(slot int, b explicit.Buffer, offset int) {
	if !p.d.Quiet {
		p.d.record("SetVertexBuffer", "slot=%d offset=%d", slot, offset)
	}
	p.d.limits.CheckSetVertexBuffer(slot)
}

func (p *pass) SetIndexBuffer(b explicit.Buffer, offset int) {
	if !p.d.Quiet {
		p.d.record("SetIndexBuffer", "offset=%d", offset)
	}
}

func (p *pass) SetIndexBuffer32(b explicit.Buffer, offset int) {
	if !p.d.Quiet {
		p.d.record("SetIndexBuffer32", "offset=%d", offset)
	}
}

func (p *pass) SetScissor(r image.Rectangle) {
	if !p.d.Quiet {
		p.d.record("SetScissor", "%v", r)
	}
}

func (p *pass) SetStencilReference(ref uint32) {
	if !p.d.Quiet {
		p.d.record("SetStencilReference", "%d", ref)
	}
}

func (p *pass) Draw(vertices, instances, firstVertex, firstInstance int) {
	if !p.d.Quiet {
		p.d.record("Draw", "vertices=%d instances=%d firstVertex=%d firstInstance=%d", vertices, instances, firstVertex, firstInstance)
	}
	if e, ok := p.vertexEntry(); ok {
		explicit.CheckFirstIndices(e, firstVertex, 0, firstInstance)
	}
}

func (p *pass) DrawIndexed(indices, instances, firstIndex, baseVertex, firstInstance int) {
	if !p.d.Quiet {
		p.d.record("DrawIndexed", "indices=%d instances=%d firstIndex=%d baseVertex=%d firstInstance=%d", indices, instances, firstIndex, baseVertex, firstInstance)
	}
	if e, ok := p.vertexEntry(); ok {
		explicit.CheckFirstIndices(e, 0, baseVertex, firstInstance)
	}
}

func (p *pass) DrawIndirect(b explicit.Buffer, offset int) {
	if !p.d.Quiet {
		p.d.record("DrawIndirect", "offset=%d", offset)
	}
	p.d.limits.CheckIndirect(b, offset, 16)
}

func (p *pass) DrawIndexedIndirect(b explicit.Buffer, offset int) {
	if !p.d.Quiet {
		p.d.record("DrawIndexedIndirect", "offset=%d", offset)
	}
	p.d.limits.CheckIndirect(b, offset, 20)
}

func (p *pass) End() {
	if !p.d.Quiet {
		p.d.record("EndPass", "")
	}
	p.ended = true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFakeLevelInvalid = "FakeLevelInvalid"
	tagFakePassNotEnded = "FakePassNotEnded"
)
