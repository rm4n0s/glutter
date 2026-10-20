package gles

import "math"

// A program remembers the value of every uniform it has set, so that setting the same
// value again costs no GL call (SPEC §9.1, "GPU efficiency"). The values live in the
// program object of the driver, so they stay across frames; every uniform of a program
// is set through these methods, or the cache would be wrong.

// uniformKind is the type of a uniform's value.
type uniformKind uint8

const (
	uniformInt uniformKind = iota + 1
	uniformFloat
	uniformVec2
	uniformVec4
	uniformFloatArray
	uniformVec4Array
)

// uniformValue is the last value set to a location.
type uniformValue struct {
	kind uniformKind
	v    [4]uint32 // the bits of up to four components
	arr  []float32 // the values of an array
}

// current returns the cache entry of the location, made if needed.
func (p *Program) entry(loc int32) *uniformValue {
	e := p.values[loc]
	if e == nil {
		e = &uniformValue{}
		p.values[loc] = e
	}
	return e
}

// scalar says whether the location already has the value, and remembers it if not.
func (p *Program) same(loc int32, kind uniformKind, v [4]uint32) bool {
	e := p.entry(loc)
	if e.kind == kind && e.v == v {
		p.dev.counts.uniformsSkipped++
		return true
	}
	e.kind, e.v, e.arr = kind, v, nil
	p.dev.counts.uniformsSet++
	return false
}

// SetInt sets an int (or sampler, or bool) uniform. A name the program does not have is
// ignored, as GL ignores location -1.
//
// Flutter: no counterpart
func (p *Program) SetInt(name string, v int32) {
	loc := p.Uniform(name)
	if loc < 0 || p.same(loc, uniformInt, [4]uint32{uint32(v)}) {
		return
	}
	p.dev.gl.uniform1i(loc, v)
}

// SetFloat sets a float uniform.
//
// Flutter: no counterpart
func (p *Program) SetFloat(name string, v float32) {
	loc := p.Uniform(name)
	if loc < 0 || p.same(loc, uniformFloat, [4]uint32{math.Float32bits(v)}) {
		return
	}
	p.dev.gl.uniform1f(loc, v)
}

// SetVec2 sets a vec2 uniform.
//
// Flutter: no counterpart
func (p *Program) SetVec2(name string, x, y float32) {
	loc := p.Uniform(name)
	if loc < 0 || p.same(loc, uniformVec2, [4]uint32{math.Float32bits(x), math.Float32bits(y)}) {
		return
	}
	p.dev.gl.uniform2f(loc, x, y)
}

// SetVec4 sets a vec4 uniform.
//
// Flutter: no counterpart
func (p *Program) SetVec4(name string, x, y, z, w float32) {
	loc := p.Uniform(name)
	if loc < 0 || p.same(loc, uniformVec4, [4]uint32{math.Float32bits(x), math.Float32bits(y), math.Float32bits(z), math.Float32bits(w)}) {
		return
	}
	p.dev.gl.uniform4f(loc, x, y, z, w)
}

// setArray is SetFloats and SetVec4s: it sets an array uniform unless it has these
// values.
func (p *Program) setArray(name string, kind uniformKind, values []float32) bool {
	loc := p.Uniform(name)
	if loc < 0 {
		return false
	}
	e := p.entry(loc)
	if e.kind == kind && len(e.arr) == len(values) {
		same := true
		for i, v := range values {
			if math.Float32bits(e.arr[i]) != math.Float32bits(v) {
				same = false
				break
			}
		}
		if same {
			p.dev.counts.uniformsSkipped++
			return false
		}
	}
	e.kind, e.v = kind, [4]uint32{}
	e.arr = append(e.arr[:0], values...)
	p.dev.counts.uniformsSet++
	return true
}

// SetFloats sets a float[] uniform.
//
// Flutter: no counterpart
func (p *Program) SetFloats(name string, values []float32) {
	if p.setArray(name, uniformFloatArray, values) {
		p.dev.gl.uniform1fv(p.Uniform(name), values)
	}
}

// SetVec4s sets a vec4[] uniform; values has four floats for each element.
//
// Flutter: no counterpart
func (p *Program) SetVec4s(name string, values []float32) {
	if p.setArray(name, uniformVec4Array, values) {
		p.dev.gl.uniform4fv(p.Uniform(name), values)
	}
}

// forgetUniforms drops the remembered values, so that every uniform is set again.
func (p *Program) forgetUniforms() { clear(p.values) }
