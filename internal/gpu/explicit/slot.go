package explicit

import (
	"encoding/binary"
	"math"
)

// minDynamicBuffer is the size a frame's instance and uniform buffers start with.
const minDynamicBuffer = 64 << 10

// frameUniformSize is the size of the Frame uniform of the shaders: the size of the target and
// its origin, four floats. A Draw uniform is the clip mask's rectangle, four floats, and for
// the vertices pipeline also the paint, the clip and the blend, sixteen.
const (
	frameUniformSize = 16
	maxDrawUniform   = 64
)

// frameSlot is the dynamic buffers of one frame in flight: the instance data of the frame's
// shapes, and the ring of uniforms addressed with dynamic offsets (SPEC §9.3 "Frames").
// The buffers grow when a frame needs more and are never shrunk.
type frameSlot struct {
	instances Buffer
	instCap   int

	uniforms Buffer
	uniCap   int
	// ring is the uniform data of the frame being built; each entry starts at a multiple of
	// the device's offset alignment.
	ring []byte
}

// ensure makes the buffer big enough for need bytes, replacing it (the GPU finishes the
// frames that use the old one first).
func (r *Renderer) ensure(b *Buffer, capacity *int, need int, usage BufferUsage) error {
	if *b != nil && *capacity >= need {
		return nil
	}
	n := max(need, minDynamicBuffer, 2**capacity)
	if *b != nil {
		(*b).Destroy()
		*b, *capacity = nil, 0
	}
	nb, err := r.dev.NewBuffer(n, usage)
	if err != nil {
		return err
	}
	*b, *capacity = nb, n
	return nil
}

// push adds a uniform to the ring and returns its dynamic offset.
func (s *frameSlot) push(align int, data []byte) uint32 {
	for len(s.ring)%align != 0 {
		s.ring = append(s.ring, 0)
	}
	off := len(s.ring)
	s.ring = append(s.ring, data...)
	return uint32(off)
}

// frameUniform encodes the Frame uniform of a target of w x h pixels whose top left corner
// is the pixel (ox, oy) of the surface.
func frameUniform(w, h, ox, oy int) []byte {
	return floatBytes([]float32{float32(w), float32(h), float32(ox), float32(oy)})
}

// upload writes the frame's primitive data into the slot's instance buffer, one pipeline's
// data after the other, and the ring into its uniform buffer. It returns where the data of each
// pipeline starts in the buffer, in bytes.
func (r *Renderer) upload(s *frameSlot, data *[numPipes][]float32) (base [numPipes]int, err error) {
	var all []byte
	for p := range data {
		base[p] = len(all)
		all = append(all, floatBytes(data[p])...)
	}
	if len(all) > 0 {
		if err := r.ensure(&s.instances, &s.instCap, len(all), BufferVertex); err != nil {
			return base, err
		}
		r.dev.WriteBuffer(s.instances, 0, all)
	}
	if len(s.ring) > 0 {
		// The last window of the ring must fit the buffer: a binding is at most
		// maxDrawUniform bytes at the dynamic offset.
		if err := r.ensure(&s.uniforms, &s.uniCap, len(s.ring)+maxDrawUniform, BufferUniform); err != nil {
			return base, err
		}
		r.dev.WriteBuffer(s.uniforms, 0, s.ring)
	}
	return base, nil
}

// destroy releases the slot's buffers.
func (s *frameSlot) destroy() {
	if s.instances != nil {
		s.instances.Destroy()
	}
	if s.uniforms != nil {
		s.uniforms.Destroy()
	}
	*s = frameSlot{}
}

// floatBytes encodes floats as the little-endian bytes the shaders read.
func floatBytes(f []float32) []byte {
	b := make([]byte, 4*len(f))
	for i, v := range f {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v))
	}
	return b
}
