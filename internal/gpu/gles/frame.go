package gles

// The frame is what a draw list becomes before any GL call is made: the data of
// each pipeline's instances or vertices, and the draw calls that use them, in
// order. Building it first lets the renderer fail on an unsupported feature
// before it draws anything, and upload each pipeline's data once.

// pipeline is one of the shader programs that draw a kind of primitive.
type pipeline int

const (
	pipeShape    pipeline = iota // analytic shapes: instanced quads (shaders/shape.*)
	pipeMask                     // coverage atlas tiles: instanced quads (shaders/mask.*)
	pipeImage                    // image rectangles: instanced quads (shaders/image.*)
	pipeVertices                 // triangles from vertices (shaders/vertices.*)
	numPipelines
)

// The size of an item of each pipeline in floats: an instance, or for the vertices
// pipeline one vertex (x, y, then r, g, b, a not premultiplied).
var itemFloats = [numPipelines]int{pipeShape: 48, pipeMask: 16, pipeImage: 32, pipeVertices: 6}

// program is the name of the shader program of each pipeline.
var programNames = [numPipelines]string{pipeShape: "shape", pipeMask: "mask", pipeImage: "image", pipeVertices: "vertices"}

// callKey is everything that two draws must share to be one draw call: the
// pipeline and its fixed-function and texture state.
type callKey struct {
	pipe    pipeline
	blend   blendFunc
	scissor [4]int // x, y, width, height from the top left of the surface
	// tex is the texture on unit 0: an atlas page or an image; nil when the
	// pipeline needs none.
	tex *Texture
	// clip is the coverage mask on unit 1, or nil.
	clip *clipMask
	// The vertices pipeline's uniforms.
	paint     [4]float32 // premultiplied
	mode      int32      // blend index of the paint with the vertex colors
	hasColors bool
	clipRect  [4]float32
}

// drawCall is a run of items drawn with one state.
type drawCall struct {
	callKey
	first, count int // the items, in the pipeline's data
}

// frame is the instances, vertices and draw calls of one frame.
type frame struct {
	data  [numPipelines][]float32
	calls []drawCall
}

// items returns the number of items of the pipeline.
func (f *frame) items(p pipeline) int { return len(f.data[p]) / itemFloats[p] }

// add appends count items, whose floats are given, as part of the last draw call
// when it has the same state, else as a new one.
func (f *frame) add(key callKey, floats []float32, count int) {
	first := f.items(key.pipe)
	if n := len(f.calls); n > 0 && f.calls[n-1].callKey == key {
		f.calls[n-1].count += count
	} else {
		f.calls = append(f.calls, drawCall{callKey: key, first: first, count: count})
	}
	f.data[key.pipe] = append(f.data[key.pipe], floats...)
}

// callsOf counts the draw calls of a pipeline.
func (f *frame) callsOf(p pipeline) int {
	n := 0
	for _, c := range f.calls {
		if c.pipe == p {
			n++
		}
	}
	return n
}
