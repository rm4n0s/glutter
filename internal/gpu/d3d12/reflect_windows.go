//go:build windows

package d3d12

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// iidShaderReflection is IID_ID3D12ShaderReflection.
var iidShaderReflection = win32.MustParseGUID("{5a58797d-a72c-478d-8ba2-efc6b0efe88e}")

// BoundResource is one resource that the bytecode of a shader binds, as D3DReflect reports it.
//
// Flutter: no counterpart
type BoundResource struct {
	// Name is the name of the variable in the HLSL.
	Name string
	// Type is the D3D_SHADER_INPUT_TYPE: SITConstantBuffer, SITTexture or SITSampler.
	Type uint32
	// BindPoint is the register number and Space the register space: register(<class>BindPoint, spaceSpace).
	BindPoint, Space uint32
	// BindCount is the number of registers it takes (1 for all of ours).
	BindCount uint32
}

// InputParameter is one input of the first stage of a shader, as D3DReflect reports it: a vertex
// attribute with its semantic.
//
// Flutter: no counterpart
type InputParameter struct {
	// SemanticName and SemanticIndex are the semantic: "LOC" and 3 for "LOC3", or "SV_VertexID".
	SemanticName  string
	SemanticIndex uint32
	// Register is the input register the semantic is in.
	Register uint32
}

// Reflection is what D3DReflect says about the bytecode of one entry point.
//
// Flutter: no counterpart
type Reflection struct {
	// Resources are the resources it binds, in the order of the registers the compiler gave them.
	Resources []BoundResource
	// Inputs are its inputs.
	Inputs []InputParameter
}

// Reflect reads the resources and the inputs out of the bytecode (D3DReflect, ID3D12ShaderReflection).
// It fails with COMCallFailed and the other errors of win32.HResultError.
//
// Flutter: no counterpart
func (d *CompilerDLL) Reflect(dxbc []byte) (Reflection, error) {
	if len(dxbc) == 0 {
		return Reflection{}, win32.HResultError("D3DReflect", win32.EInvalidArg)
	}
	var out uintptr
	r, _ := d.reflect.Call(uintptr(unsafe.Pointer(&dxbc[0])), uintptr(len(dxbc)),
		uintptr(unsafe.Pointer(&iidShaderReflection)), uintptr(unsafe.Pointer(&out)))
	if err := win32.HResultError("D3DReflect", win32.FromResult(r)); err != nil {
		return Reflection{}, err
	}
	refl := win32.ObjectFromPtr(out)
	defer refl.Release()
	var desc ShaderDesc
	if err := refl.CallHR("ID3D12ShaderReflection::GetDesc", reflectGetDesc, uintptr(unsafe.Pointer(&desc))); err != nil {
		return Reflection{}, err
	}
	var res Reflection
	for i := uint32(0); i < desc.BoundResources; i++ {
		var b ShaderInputBindDesc
		if err := refl.CallHR("ID3D12ShaderReflection::GetResourceBindingDesc", reflectGetResourceBindingDesc, uintptr(i), uintptr(unsafe.Pointer(&b))); err != nil {
			return Reflection{}, err
		}
		res.Resources = append(res.Resources, BoundResource{Name: win32.CString(b.Name), Type: b.Type, BindPoint: b.BindPoint, Space: b.Space, BindCount: b.BindCount})
	}
	for i := uint32(0); i < desc.InputParameters; i++ {
		var p SignatureParameterDesc
		if err := refl.CallHR("ID3D12ShaderReflection::GetInputParameterDesc", reflectGetInputParameterDesc, uintptr(i), uintptr(unsafe.Pointer(&p))); err != nil {
			return Reflection{}, err
		}
		res.Inputs = append(res.Inputs, InputParameter{SemanticName: win32.CString(p.SemanticName), SemanticIndex: p.SemanticIndex, Register: p.Register})
	}
	return res, nil
}
