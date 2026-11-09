//go:build darwin && !ios

package metal

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/darwin/layouttest"
)

// TestStructMirrorsMatchTheSDK checks the Go mirrors of the structures that the device passes by value against the
// headers of the SDK (skipped, with the install command, without the Command Line Tools).
func TestStructMirrorsMatchTheSDK(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"Metal/Metal.h", "QuartzCore/QuartzCore.h"},
		Structs: []layouttest.Struct{
			layouttest.StructOf("MTLOrigin", mtlOrigin{}),
			layouttest.StructOf("MTLSize", mtlSize{}),
			layouttest.StructOf("MTLRegion", mtlRegion{}),
			layouttest.StructOf("MTLClearColor", mtlClearColor{}),
			layouttest.StructOf("MTLScissorRect", mtlScissorRect{}),
		},
	})
}

// TestConstantsMatchTheSDK checks the numbers of Metal's enumerations that mapping.go and the device use against
// the headers.
func TestConstantsMatchTheSDK(t *testing.T) {
	c := func(name string, v uint64) layouttest.Const { return layouttest.Const{Name: name, Value: int64(v)} }
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"Metal/Metal.h", "QuartzCore/QuartzCore.h"},
		Consts: []layouttest.Const{
			c("MTLPixelFormatR8Unorm", pixelFormatR8Unorm), c("MTLPixelFormatR32Uint", pixelFormatR32Uint),
			c("MTLPixelFormatR32Float", pixelFormatR32Float), c("MTLPixelFormatRGBA8Unorm", pixelFormatRGBA8Unorm),
			c("MTLPixelFormatBGRA8Unorm", pixelFormatBGRA8Unorm), c("MTLPixelFormatRG32Uint", pixelFormatRG32Uint),
			c("MTLPixelFormatRGBA16Float", pixelFormatRGBA16Float), c("MTLPixelFormatRGBA32Float", pixelFormatRGBA32Float),
			c("MTLPixelFormatDepth32Float", pixelFormatDepth32Float), c("MTLPixelFormatDepth32Float_Stencil8", pixelFormatDepth32FloatStenc),
			c("MTLTextureType2D", textureType2D), c("MTLTextureType2DArray", textureType2DArray),
			c("MTLTextureType2DMultisample", textureType2DMultisample), c("MTLTextureTypeCube", textureTypeCube), c("MTLTextureType3D", textureType3D),
			c("MTLTextureUsageShaderRead", textureUsageShaderRead), c("MTLTextureUsageShaderWrite", textureUsageShaderWrite),
			c("MTLTextureUsageRenderTarget", textureUsageRenderTarget), c("MTLTextureUsagePixelFormatView", textureUsagePixelFormatView),
			c("MTLStorageModeShared", storageShared), c("MTLStorageModeManaged", storageManaged), c("MTLStorageModePrivate", storagePrivate),
			c("MTLResourceCPUCacheModeWriteCombined", resourceCPUCacheWriteCombined),
			c("MTLResourceStorageModeShift", resourceStorageShift),
			c("MTLSamplerMinMagFilterNearest", filterNearest), c("MTLSamplerMinMagFilterLinear", filterLinear),
			c("MTLSamplerMipFilterNotMipmapped", mipNotMipmapped), c("MTLSamplerMipFilterNearest", mipNearest), c("MTLSamplerMipFilterLinear", mipLinear),
			c("MTLSamplerAddressModeClampToEdge", addressClampToEdge), c("MTLSamplerAddressModeRepeat", addressRepeat),
			c("MTLSamplerAddressModeMirrorRepeat", addressMirrorRepeat),
			c("MTLBlendFactorZero", blendZero), c("MTLBlendFactorOne", blendOne), c("MTLBlendFactorSourceAlpha", blendSourceAlpha),
			c("MTLBlendFactorOneMinusSourceAlpha", blendOneMinusSourceAlpha), c("MTLBlendFactorDestinationAlpha", blendDestAlpha),
			c("MTLBlendFactorOneMinusDestinationAlpha", blendOneMinusDestAlpha), c("MTLBlendOperationAdd", blendOpAdd),
			c("MTLColorWriteMaskRed", colorWriteRed), c("MTLColorWriteMaskGreen", colorWriteGreen),
			c("MTLColorWriteMaskBlue", colorWriteBlue), c("MTLColorWriteMaskAlpha", colorWriteAlpha),
			c("MTLPrimitiveTypePoint", primitivePoint), c("MTLPrimitiveTypeLine", primitiveLine), c("MTLPrimitiveTypeLineStrip", primitiveLineStrip),
			c("MTLPrimitiveTypeTriangle", primitiveTriangle), c("MTLPrimitiveTypeTriangleStrip", primitiveTriangleStrip),
			c("MTLPrimitiveTopologyClassPoint", topologyClassPoint), c("MTLPrimitiveTopologyClassLine", topologyClassLine),
			c("MTLPrimitiveTopologyClassTriangle", topologyClassTriangle),
			c("MTLCullModeNone", cullNone), c("MTLCullModeFront", cullFront), c("MTLCullModeBack", cullBack),
			c("MTLWindingClockwise", windingClockwise), c("MTLWindingCounterClockwise", windingCounterClockwise),
			c("MTLCompareFunctionNever", compareNever), c("MTLCompareFunctionLess", compareLess), c("MTLCompareFunctionEqual", compareEqual),
			c("MTLCompareFunctionLessEqual", compareLessEqual), c("MTLCompareFunctionGreater", compareGreater),
			c("MTLCompareFunctionNotEqual", compareNotEqual), c("MTLCompareFunctionGreaterEqual", compareGreaterEqual), c("MTLCompareFunctionAlways", compareAlways),
			c("MTLStencilOperationKeep", stencilKeep), c("MTLStencilOperationZero", stencilZero), c("MTLStencilOperationReplace", stencilReplace),
			c("MTLStencilOperationIncrementClamp", stencilIncrementClamp), c("MTLStencilOperationDecrementClamp", stencilDecrementClamp),
			c("MTLStencilOperationInvert", stencilInvert), c("MTLStencilOperationIncrementWrap", stencilIncrementWrap),
			c("MTLStencilOperationDecrementWrap", stencilDecrementWrap),
			c("MTLLoadActionDontCare", loadDontCare), c("MTLLoadActionLoad", loadLoad), c("MTLLoadActionClear", loadClear),
			c("MTLStoreActionDontCare", storeDontCare), c("MTLStoreActionStore", storeStore),
			c("MTLStoreActionMultisampleResolve", storeMultisampleResolve), c("MTLStoreActionStoreAndMultisampleResolve", storeStoreAndMultisampleResolve),
			c("MTLIndexTypeUInt16", indexUInt16), c("MTLIndexTypeUInt32", indexUInt32),
			c("MTLVertexStepFunctionPerVertex", stepPerVertex), c("MTLVertexStepFunctionPerInstance", stepPerInstance),
			c("MTLLanguageVersion2_4", languageVersion24), c("MTLPipelineOptionFailOnBinaryArchiveMiss", pipelineFailOnArchiveMiss),
			c("MTLCommandBufferStatusCompleted", statusCompleted), c("MTLCommandBufferStatusError", statusError),
			c("MTLVertexFormatUChar2", vfUChar2), c("MTLVertexFormatUChar4", vfUChar4), c("MTLVertexFormatChar2", vfChar2),
			c("MTLVertexFormatChar4", vfChar4), c("MTLVertexFormatUChar2Normalized", vfUChar2Normalized),
			c("MTLVertexFormatUChar4Normalized", vfUChar4Normalized), c("MTLVertexFormatChar2Normalized", vfChar2Normalized),
			c("MTLVertexFormatChar4Normalized", vfChar4Normalized), c("MTLVertexFormatUShort2", vfUShort2), c("MTLVertexFormatUShort4", vfUShort4),
			c("MTLVertexFormatShort2", vfShort2), c("MTLVertexFormatShort4", vfShort4), c("MTLVertexFormatUShort2Normalized", vfUShort2Normalized),
			c("MTLVertexFormatUShort4Normalized", vfUShort4Normalized), c("MTLVertexFormatShort2Normalized", vfShort2Normalized),
			c("MTLVertexFormatShort4Normalized", vfShort4Normalized), c("MTLVertexFormatHalf2", vfHalf2), c("MTLVertexFormatHalf4", vfHalf4),
			c("MTLVertexFormatFloat", vfFloat), c("MTLVertexFormatFloat2", vfFloat2), c("MTLVertexFormatFloat3", vfFloat3), c("MTLVertexFormatFloat4", vfFloat4),
			c("MTLVertexFormatInt", vfInt), c("MTLVertexFormatInt2", vfInt2), c("MTLVertexFormatInt3", vfInt3), c("MTLVertexFormatInt4", vfInt4),
			c("MTLVertexFormatUInt", vfUInt), c("MTLVertexFormatUInt2", vfUInt2), c("MTLVertexFormatUInt3", vfUInt3), c("MTLVertexFormatUInt4", vfUInt4),
			c("MTLArgumentTypeBuffer", argBuffer), c("MTLArgumentTypeTexture", argTexture), c("MTLArgumentTypeSampler", argSampler),
			c("MTLFunctionTypeVertex", funcVertex), c("MTLFunctionTypeFragment", funcFragment), c("MTLFunctionTypeKernel", funcKernel),
			{Name: "kCALayerWidthSizable|kCALayerHeightSizable", Value: int64(layerSizable)},
		},
	})
}
