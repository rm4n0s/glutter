package stl

import (
	"context"
	"encoding/binary"
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
)

// checkEvery is how many triangles or lines are read between two looks at the context.
const checkEvery = 1 << 15

// defaultColor is the color of a triangle without one in a file where some triangles have it: the default color of a material.
var defaultColor = packColor(
	byte(model.DefaultColor[0]*255+0.5), byte(model.DefaultColor[1]*255+0.5), byte(model.DefaultColor[2]*255+0.5))

func packColor(r, g, b byte) uint32 { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 | 0xFF<<24 }

// attributeColor reads the two attribute bytes of a triangle by the VisCAM and SolidView convention: bit 15 says that the color is
// valid, and bits 10 to 14, 5 to 9 and 0 to 4 are red, green and blue, five bits each. It returns 0 for a triangle without color.
func attributeColor(attr uint16) uint32 {
	if attr&0x8000 == 0 {
		return 0
	}
	scale := func(v uint16) byte { return byte((uint32(v&31)*255 + 15) / 31) }
	return packColor(scale(attr>>10), scale(attr>>5), scale(attr))
}

func decodeBinary(ctx context.Context, data []byte, o Options) (*model.Model, error) {
	count, ok := binaryCount(data)
	if !ok {
		return nil, errors.New(TagSTLInvalid, "the file is neither ASCII nor binary STL: it is shorter than the header of a binary file",
			"offset", len(data), "size", len(data), "reason", "short")
	}
	need := binaryHeader + binaryRecord*count
	if need > uint64(len(data)) {
		return nil, errors.New(TagSTLInvalid, "the binary file has fewer triangles than its count says",
			"offset", len(data), "triangles", count, "size", len(data), "expected", need, "reason", "truncated")
	}
	if count == 0 {
		return nil, errors.New(TagSTLInvalid, "the binary file has no triangles", "offset", 80, "reason", "empty")
	}
	n := int(count)
	s := &soup[float32]{pos: make([]float32, 9*n), norm: make([]float32, 3*n)}
	colors := make([]uint32, n)
	workers := workersFor(n)
	bad := make([]int, workers) // the first triangle of each worker with a coordinate that is not a number, or -1
	colored := make([]bool, workers)
	le := binary.LittleEndian
	rec := data[binaryHeader:]
	parallel(n, workers, func(worker, lo, hi int) {
		bad[worker] = -1
		for t := lo; t < hi; t++ {
			if t%checkEvery == 0 && ctx.Err() != nil {
				return
			}
			r := rec[t*binaryRecord : t*binaryRecord+binaryRecord : t*binaryRecord+binaryRecord]
			var sum float64
			for i := 0; i < 12; i++ {
				v := math.Float32frombits(le.Uint32(r[4*i:]))
				if i < 3 {
					// The stored normal is only a hint: one that is not a number is as good as none.
					if v-v != 0 {
						v = 0
					}
					s.norm[3*t+i] = v
					continue
				}
				s.pos[9*t+i-3] = v
				sum += float64(v)
			}
			if sum-sum != 0 && bad[worker] < 0 {
				bad[worker] = t
			}
			if attr := le.Uint16(r[48:]); attr != 0 {
				if c := attributeColor(attr); c != 0 {
					colors[t] = c
					colored[worker] = true
				}
			}
		}
	})
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("offset", binaryHeader, "phase", "parse")
	}
	for _, t := range bad {
		if t >= 0 {
			return nil, errors.New(TagSTLInvalid, "a coordinate is not a finite number",
				"offset", binaryHeader+binaryRecord*t+12, "triangle", t, "reason", "not finite")
		}
	}
	if !slices.Contains(colored, true) {
		colors = nil
	}
	if colors != nil {
		for i, c := range colors {
			if c == 0 {
				colors[i] = defaultColor
			}
		}
		s.color = colors
	}
	mesh, err := build(ctx, s, o)
	if err != nil {
		return nil, err
	}
	m := newModel(o.Name)
	mesh.Name = o.Name
	addMesh(m, mesh)
	return m, nil
}
