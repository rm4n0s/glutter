//go:build !windows

package d3d12

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32/layouttest"
)

// TestLayoutAgainstTheHeaders checks the Go mirrors of the shader reflection, the positions of the
// methods of the interfaces the backend calls, the compile flags and the values of the enumerations
// against the mingw-w64 headers (SPEC §25.2). It runs on the Linux host and is skipped, with the
// install command, when the compiler is missing.
func TestLayoutAgainstTheHeaders(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"windows.h", "d3dcommon.h", "d3dcompiler.h", "d3d12shader.h"},
		Structs: []layouttest.Struct{
			layouttest.StructOf("D3D12_SHADER_DESC", ShaderDesc{}),
			layouttest.StructOf("D3D12_SHADER_INPUT_BIND_DESC", ShaderInputBindDesc{}),
			layouttest.StructOf("D3D12_SIGNATURE_PARAMETER_DESC", SignatureParameterDesc{}),
		},
		VTables: []layouttest.VTable{
			{CName: "ID3D10BlobVtbl", Methods: []layouttest.Method{
				{Name: "GetBufferPointer", Index: blobGetBufferPointer},
				{Name: "GetBufferSize", Index: blobGetBufferSize},
			}},
			{CName: "ID3D12ShaderReflectionVtbl", Methods: []layouttest.Method{
				{Name: "GetDesc", Index: reflectGetDesc},
				{Name: "GetResourceBindingDesc", Index: reflectGetResourceBindingDesc},
				{Name: "GetInputParameterDesc", Index: reflectGetInputParameterDesc},
			}},
		},
		Consts: []layouttest.Const{
			{Name: "D3DCOMPILE_DEBUG", Value: CompileDebug},
			{Name: "D3DCOMPILE_SKIP_OPTIMIZATION", Value: CompileSkipOptimization},
			{Name: "D3DCOMPILE_OPTIMIZATION_LEVEL3", Value: CompileOptimizationLevel3},
			{Name: "D3D_SIT_CBUFFER", Value: SITConstantBuffer},
			{Name: "D3D_SIT_TEXTURE", Value: SITTexture},
			{Name: "D3D_SIT_SAMPLER", Value: SITSampler},
		},
	})
}
