package fakedevice

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// stateText describes the parts of a pipeline that the 3D API adds, in the text of the log; it is empty
// for a pipeline of the 2D renderer, so that its log does not change.
func stateText(desc explicit.PipelineDesc) string {
	var b strings.Builder
	if len(desc.ColorTargets) > 0 {
		fmt.Fprintf(&b, " targets=%+v", desc.ColorTargets)
	}
	if desc.Topology != explicit.TopologyTriangleList {
		fmt.Fprintf(&b, " topology=%d", desc.Topology)
	}
	if desc.CullMode != explicit.CullNone {
		fmt.Fprintf(&b, " cull=%d", desc.CullMode)
	}
	if desc.FrontFace != explicit.FrontFaceCCW {
		fmt.Fprintf(&b, " frontface=%d", desc.FrontFace)
	}
	if desc.DepthStencil != (explicit.DepthStencilState{}) {
		fmt.Fprintf(&b, " depthstencil=%+v", desc.DepthStencil)
	}
	if desc.SampleCount > 1 {
		fmt.Fprintf(&b, " samples=%d", desc.SampleCount)
	}
	return b.String()
}

// passText describes the attachments of a pass in the text of the log.
func passText(desc explicit.PassDesc) string {
	var b strings.Builder
	for i, c := range desc.Color {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "color%d={%v load=%d store=%d clear=%v", i, c.Target.Desc().Format, c.Load, c.Store, c.Clear)
		if c.Resolve != nil {
			b.WriteString(" resolve")
		}
		b.WriteString("}")
	}
	if d := desc.Depth; d != nil {
		fmt.Fprintf(&b, " depth={%v load=%d store=%d clear=%v stencil load=%d store=%d clear=%d}", d.Target.Desc().Format,
			d.DepthLoad, d.DepthStore, d.ClearDepth, d.StencilLoad, d.StencilStore, d.ClearStencil)
	}
	return b.String()
}

// fillBytes sets every pixel to the bytes of one pixel.
func (t *texture) fillBytes(px []byte) {
	if t.pix == nil || len(px) != t.bpp {
		return
	}
	for i := 0; i+t.bpp <= len(t.pix); i += t.bpp {
		copy(t.pix[i:], px)
	}
}

// encodeColor returns the bytes of one pixel of the format that hold the color, as a clearing pass
// writes them: unorm formats round, float formats keep the value, Uint formats take the whole
// numbers.
func encodeColor(f explicit.TextureFormat, c [4]float64) []byte {
	u8 := func(v float64) byte { return unorm8(float32(v)) }
	switch f {
	case explicit.TextureFormatRGBA8Unorm:
		return []byte{u8(c[0]), u8(c[1]), u8(c[2]), u8(c[3])}
	case explicit.TextureFormatBGRA8Unorm:
		return []byte{u8(c[2]), u8(c[1]), u8(c[0]), u8(c[3])}
	case explicit.TextureFormatR8Unorm:
		return []byte{u8(c[3])} // R8 holds the alpha of a clear color, as coverage (see texture.fill)
	case explicit.TextureFormatRGBA16Float:
		out := make([]byte, 8)
		for i := 0; i < 4; i++ {
			binary.LittleEndian.PutUint16(out[i*2:], explicit.Float16Bits(float32(c[i])))
		}
		return out
	case explicit.TextureFormatRGBA32Float:
		out := make([]byte, 16)
		for i := 0; i < 4; i++ {
			binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(float32(c[i])))
		}
		return out
	case explicit.TextureFormatR32Float:
		return binary.LittleEndian.AppendUint32(nil, math.Float32bits(float32(c[0])))
	case explicit.TextureFormatR32Uint:
		return binary.LittleEndian.AppendUint32(nil, uint32(c[0]))
	case explicit.TextureFormatRG32Uint:
		return binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(nil, uint32(c[0])), uint32(c[1]))
	}
	return nil
}
