//go:build darwin && !ios

package metal

import (
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// The MSL that glutter-gen writes is compiled by Metal's own compiler here (SPEC §29.5: a shader that Metal
// refuses is a test failure), and what Metal reflects of each entry point is compared with the binding map
// that the generator wrote: the same names at the same indexes.

// unit is a shader, reduced to what the check needs. The shaders of viewport3d (gpu.Shader) are checked in step 33.9,
// in a package that can import gpu: this package is imported by the engine, which gpu imports.
type unit struct {
	name, msl string
	entries   []unitEntry
}

type unitEntry struct {
	name   string
	stage  string // vertex, fragment or kernel
	binds  []unitBind
	inputs int
}

type unitBind struct {
	name  string
	class argClass
	index int
}

func stageName(s explicit.ShaderStage) string {
	switch s {
	case explicit.ShaderStageVertex:
		return "vertex"
	case explicit.ShaderStageFragment:
		return "fragment"
	}
	return "kernel"
}

func unitOfExplicit(sh *explicit.Shader) unit {
	u := unit{name: sh.Name, msl: sh.MSL}
	for _, e := range sh.Entries {
		ue := unitEntry{name: e.Name, stage: stageName(e.Stage)}
		for _, b := range e.Bindings {
			ue.binds = append(ue.binds, unitBind{b.Name, classOf(b.Kind), b.MetalIndex})
		}
		u.entries = append(u.entries, ue)
	}
	return u
}

func allUnits() []unit {
	var units []unit
	for _, sh := range explicit.Shaders() {
		units = append(units, unitOfExplicit(sh))
	}
	for _, sh := range []*explicit.Shader{devicetest.ShaderCompute, devicetest.ShaderFill, devicetest.ShaderInstance, devicetest.ShaderSample, devicetest.ShaderTri, devicetest.ShaderVformats} {
		units = append(units, unitOfExplicit(sh))
	}
	return units
}

// The types of Metal's reflection (MTLArgumentType, MTLFunctionType, MTLDataType).
const (
	argBuffer  = 0
	argTexture = 2
	argSampler = 3

	funcVertex   = 1
	funcFragment = 2
	funcKernel   = 3
)

// reflected is what Metal says of the arguments of an entry point: the name and index of each resource.
type reflected struct {
	class argClass
	index int
}

func reflectArguments(args objc.ID) map[string]reflected {
	out := map[string]reflected{}
	for i := 0; i < objcx.ArrayCount(args); i++ {
		a := objcx.ArrayAt(args, i)
		var class argClass
		switch send[uint64](a, "type") {
		case argBuffer:
			class = classBuffer
		case argTexture:
			class = classTexture
		case argSampler:
			class = classSampler
		default:
			continue
		}
		index := int(send[uint64](a, "index"))
		if class == classBuffer && index >= explicit.MetalVertexBufferBase {
			continue // the vertex buffers of the vertex descriptor
		}
		out[objcx.String(call(a, "name"))] = reflected{class, index}
	}
	return out
}

// vertexFormatOfDataType maps the type of a stage input (MTLDataType) to a vertex format of the same shape.
func vertexFormatOfDataType(t uint64) uint64 {
	switch t {
	case 3, 4, 5, 6: // float .. float4
		return vfFloat + (t - 3)
	case 29, 30, 31, 32: // int .. int4
		return vfInt + (t - 29)
	case 33, 34, 35, 36: // uint .. uint4
		return vfUInt + (t - 33)
	}
	return vfFloat4
}

func TestEveryMSLCompilesAndIsBoundAtItsIndexes(t *testing.T) {
	d := openTestDevice(t, Options{})
	compared := 0
	t.Cleanup(func() {
		if compared == 0 && !t.Failed() {
			t.Error("no resource was compared with the map")
		}
		t.Logf("%d resources compared with Metal's reflection", compared)
	})
	for _, u := range allUnits() {
		t.Run(u.name, func(t *testing.T) {
			if u.msl == "" {
				t.Fatal("the shader has no MSL")
			}
			lib, err := d.library(&explicit.Shader{Name: u.name, MSL: u.msl})
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range u.entries {
				fn, err := lib.function(e.name)
				if err != nil {
					t.Errorf("%s: %v", e.name, err)
					continue
				}
				wantType := map[string]uint64{"vertex": funcVertex, "fragment": funcFragment, "kernel": funcKernel}[e.stage]
				if got := send[uint64](fn, "functionType"); got != wantType {
					t.Errorf("%s is a function of type %d, want %d", e.name, got, wantType)
					continue
				}
				var got map[string]reflected
				switch e.stage {
				case "kernel":
					got = d.reflectCompute(t, fn)
				case "vertex":
					got = d.reflectVertex(t, fn)
				default:
					continue // a fragment function needs a vertex function to make a pipeline; the text check covers it
				}
				if got == nil {
					continue
				}
				for _, b := range e.binds {
					r, ok := got[b.name]
					if !ok {
						// Metal leaves out an argument that the compiled function does not use.
						continue
					}
					compared++
					if r.class != b.class || r.index != b.index {
						t.Errorf("%s: Metal has %s at class %d index %d, the map says class %d index %d", e.name, b.name, r.class, r.index, b.class, b.index)
					}
				}
				for name := range got {
					found := false
					for _, b := range e.binds {
						found = found || b.name == name
					}
					if !found {
						t.Errorf("%s: Metal has a resource %q that the map does not", e.name, name)
					}
				}
			}
		})
	}
}

// reflectCompute makes a compute pipeline of the function with argument reflection and returns its arguments.
func (d *Device) reflectCompute(t *testing.T, fn objc.ID) map[string]reflected {
	t.Helper()
	var out map[string]reflected
	objcx.WithPool(func() {
		cd := newObject("MTLComputePipelineDescriptor")
		defer release(cd)
		call(cd, "setComputeFunction:", fn)
		var refl, e objc.ID
		st := call(d.dev, "newComputePipelineStateWithDescriptor:options:reflection:error:", cd, uint64(1), ptr(&refl), ptr(&e))
		if st == 0 {
			t.Errorf("the compute pipeline: %s", description(e))
			return
		}
		defer release(st)
		out = reflectArguments(call(refl, "arguments"))
	})
	return out
}

// reflectVertex makes a pipeline of the vertex function with a fragment function that does nothing, with argument reflection, and
// returns the arguments of the vertex stage. The vertex descriptor is made from the inputs the function declares.
func (d *Device) reflectVertex(t *testing.T, fn objc.ID) map[string]reflected {
	t.Helper()
	var out map[string]reflected
	objcx.WithPool(func() {
		rd := newObject("MTLRenderPipelineDescriptor")
		defer release(rd)
		call(rd, "setVertexFunction:", fn)
		call(rd, "setFragmentFunction:", d.noopFragment(t))
		call(call(call(rd, "colorAttachments"), "objectAtIndexedSubscript:", uint64(0)), "setPixelFormat:", pixelFormatBGRA8Unorm)
		if inputs := call(fn, "stageInputAttributes"); inputs != 0 && objcx.ArrayCount(inputs) > 0 {
			vd := call(class("MTLVertexDescriptor"), "vertexDescriptor")
			attrs, layouts := call(vd, "attributes"), call(vd, "layouts")
			offset := 0
			for i := 0; i < objcx.ArrayCount(inputs); i++ {
				a := objcx.ArrayAt(inputs, i)
				f := vertexFormatOfDataType(send[uint64](a, "attributeType"))
				at := call(attrs, "objectAtIndexedSubscript:", send[uint64](a, "attributeIndex"))
				call(at, "setFormat:", f)
				call(at, "setOffset:", uint64(offset))
				call(at, "setBufferIndex:", vertexBufferIndex(0))
				offset += 16
			}
			l := call(layouts, "objectAtIndexedSubscript:", vertexBufferIndex(0))
			call(l, "setStride:", uint64(offset))
			call(rd, "setVertexDescriptor:", vd)
		}
		var refl, e objc.ID
		st := call(d.dev, "newRenderPipelineStateWithDescriptor:options:reflection:error:", rd, uint64(1), ptr(&refl), ptr(&e))
		if st == 0 {
			t.Errorf("the vertex pipeline: %s", description(e))
			return
		}
		defer release(st)
		out = reflectArguments(call(refl, "vertexArguments"))
	})
	return out
}

// noopFragment is a fragment function that reads and writes nothing, which a vertex function is paired with
// to make a pipeline.
func (d *Device) noopFragment(t *testing.T) objc.ID {
	t.Helper()
	lib, err := d.library(&explicit.Shader{Name: "noop", MSL: "#include <metal_stdlib>\nfragment metal::float4 noop() { return metal::float4(0.0); }\n"})
	if err != nil {
		t.Fatal(err)
	}
	fn, err := lib.function("noop")
	if err != nil {
		t.Fatal(err)
	}
	return fn
}
