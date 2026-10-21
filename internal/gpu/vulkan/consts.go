//go:build linux

package vulkan

// Vulkan constants that the backend uses. The comment on each is its name in
// vulkan_core.h; TestConstantsMatchHeader compiles a program against the header
// when it is installed and compares every one. Each value is a literal, so that the
// test can read it from this file.

// VkResult.
const (
	resultSuccess             = 0           // VK_SUCCESS
	resultNotReady            = 1           // VK_NOT_READY
	resultTimeout             = 2           // VK_TIMEOUT
	resultIncomplete          = 5           // VK_INCOMPLETE
	resultErrorOutOfHost      = -1          // VK_ERROR_OUT_OF_HOST_MEMORY
	resultErrorOutOfDevice    = -2          // VK_ERROR_OUT_OF_DEVICE_MEMORY
	resultErrorInitFailed     = -3          // VK_ERROR_INITIALIZATION_FAILED
	resultErrorDeviceLost     = -4          // VK_ERROR_DEVICE_LOST
	resultErrorLayerNotPres   = -6          // VK_ERROR_LAYER_NOT_PRESENT
	resultErrorExtNotPres     = -7          // VK_ERROR_EXTENSION_NOT_PRESENT
	resultErrorFeatureNotPres = -8          // VK_ERROR_FEATURE_NOT_PRESENT
	resultErrorIncompatible   = -9          // VK_ERROR_INCOMPATIBLE_DRIVER
	resultErrorFragmentedPool = -12         // VK_ERROR_FRAGMENTED_POOL
	resultErrorOutOfPool      = -1000069000 // VK_ERROR_OUT_OF_POOL_MEMORY
)

// VkStructureType.
const (
	sTypeApplicationInfo                  = 0          // VK_STRUCTURE_TYPE_APPLICATION_INFO
	sTypeInstanceCreateInfo               = 1          // VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO
	sTypeDeviceQueueCreateInfo            = 2          // VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO
	sTypeDeviceCreateInfo                 = 3          // VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO
	sTypeSubmitInfo                       = 4          // VK_STRUCTURE_TYPE_SUBMIT_INFO
	sTypeMemoryAllocateInfo               = 5          // VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO
	sTypeFenceCreateInfo                  = 8          // VK_STRUCTURE_TYPE_FENCE_CREATE_INFO
	sTypeBufferCreateInfo                 = 12         // VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO
	sTypeImageCreateInfo                  = 14         // VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO
	sTypeImageViewCreateInfo              = 15         // VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO
	sTypeShaderModuleCreateInfo           = 16         // VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO
	sTypePipelineShaderStageCreateInfo    = 18         // VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO
	sTypePipelineVertexInputStateCreate   = 19         // VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO
	sTypePipelineInputAssemblyStateCreate = 20         // VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO
	sTypePipelineViewportStateCreate      = 22         // VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO
	sTypePipelineRasterizationStateCreate = 23         // VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO
	sTypePipelineMultisampleStateCreate   = 24         // VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO
	sTypePipelineColorBlendStateCreate    = 26         // VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO
	sTypePipelineDynamicStateCreate       = 27         // VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO
	sTypeGraphicsPipelineCreateInfo       = 28         // VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO
	sTypePipelineLayoutCreateInfo         = 30         // VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO
	sTypeSamplerCreateInfo                = 31         // VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO
	sTypeDescriptorSetLayoutCreateInfo    = 32         // VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO
	sTypeDescriptorPoolCreateInfo         = 33         // VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO
	sTypeDescriptorSetAllocateInfo        = 34         // VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO
	sTypeWriteDescriptorSet               = 35         // VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET
	sTypeCommandPoolCreateInfo            = 39         // VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO
	sTypeCommandBufferAllocateInfo        = 40         // VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO
	sTypeCommandBufferBeginInfo           = 42         // VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO
	sTypePhysicalDeviceFeatures2          = 1000059000 // VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2
	sTypeDebugUtilsMessengerCreateInfo    = 1000128004 // VK_STRUCTURE_TYPE_DEBUG_UTILS_MESSENGER_CREATE_INFO_EXT
	sTypePipelineRenderingCreateInfo      = 1000044002 // VK_STRUCTURE_TYPE_PIPELINE_RENDERING_CREATE_INFO
	sTypeRenderingInfo                    = 1000044000 // VK_STRUCTURE_TYPE_RENDERING_INFO
	sTypeRenderingAttachmentInfo          = 1000044001 // VK_STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO
	sTypePhysicalDeviceDynamicRendering   = 1000044003 // VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_DYNAMIC_RENDERING_FEATURES
	sTypePhysicalDeviceSynchronization2   = 1000314007 // VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SYNCHRONIZATION_2_FEATURES
	sTypeMemoryBarrier2                   = 1000314000 // VK_STRUCTURE_TYPE_MEMORY_BARRIER_2
	sTypeImageMemoryBarrier2              = 1000314002 // VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER_2
	sTypeDependencyInfo                   = 1000314003 // VK_STRUCTURE_TYPE_DEPENDENCY_INFO
)

// Versions and flags of instances and devices.
const (
	apiVersion11 = 0x00401000 // VK_API_VERSION_1_1
	apiVersion13 = 0x00403000 // VK_API_VERSION_1_3

	queueGraphics = 0x00000001 // VK_QUEUE_GRAPHICS_BIT

	deviceTypeOther      = 0 // VK_PHYSICAL_DEVICE_TYPE_OTHER
	deviceTypeIntegrated = 1 // VK_PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU
	deviceTypeDiscrete   = 2 // VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU
	deviceTypeVirtual    = 3 // VK_PHYSICAL_DEVICE_TYPE_VIRTUAL_GPU
	deviceTypeCPU        = 4 // VK_PHYSICAL_DEVICE_TYPE_CPU

	memoryDeviceLocal  = 0x00000001 // VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT
	memoryHostVisible  = 0x00000002 // VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT
	memoryHostCoherent = 0x00000004 // VK_MEMORY_PROPERTY_HOST_COHERENT_BIT

	debugSeverityVerbose = 0x00000001 // VK_DEBUG_UTILS_MESSAGE_SEVERITY_VERBOSE_BIT_EXT
	debugSeverityInfo    = 0x00000010 // VK_DEBUG_UTILS_MESSAGE_SEVERITY_INFO_BIT_EXT
	debugSeverityWarning = 0x00000100 // VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT
	debugSeverityError   = 0x00001000 // VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT
	debugTypeGeneral     = 0x00000001 // VK_DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT
	debugTypeValidation  = 0x00000002 // VK_DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT
	debugTypePerformance = 0x00000004 // VK_DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT
)

// Buffers, images, samplers.
const (
	bufferUsageTransferSrc = 0x00000001 // VK_BUFFER_USAGE_TRANSFER_SRC_BIT
	bufferUsageTransferDst = 0x00000002 // VK_BUFFER_USAGE_TRANSFER_DST_BIT
	bufferUsageUniform     = 0x00000010 // VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT
	bufferUsageIndex       = 0x00000040 // VK_BUFFER_USAGE_INDEX_BUFFER_BIT
	bufferUsageVertex      = 0x00000080 // VK_BUFFER_USAGE_VERTEX_BUFFER_BIT

	sharingExclusive = 0 // VK_SHARING_MODE_EXCLUSIVE

	imageType2D           = 1          // VK_IMAGE_TYPE_2D
	imageViewType2D       = 1          // VK_IMAGE_VIEW_TYPE_2D
	imageTilingOptimal    = 0          // VK_IMAGE_TILING_OPTIMAL
	imageUsageTransferSrc = 0x00000001 // VK_IMAGE_USAGE_TRANSFER_SRC_BIT
	imageUsageTransferDst = 0x00000002 // VK_IMAGE_USAGE_TRANSFER_DST_BIT
	imageUsageSampled     = 0x00000004 // VK_IMAGE_USAGE_SAMPLED_BIT
	imageUsageColor       = 0x00000010 // VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT
	sampleCount1          = 0x00000001 // VK_SAMPLE_COUNT_1_BIT
	imageAspectColor      = 0x00000001 // VK_IMAGE_ASPECT_COLOR_BIT

	layoutUndefined = 0 // VK_IMAGE_LAYOUT_UNDEFINED
	layoutGeneral   = 1 // VK_IMAGE_LAYOUT_GENERAL

	formatR8Unorm      = 9   // VK_FORMAT_R8_UNORM
	formatRGBA8Unorm   = 37  // VK_FORMAT_R8G8B8A8_UNORM
	formatBGRA8Unorm   = 44  // VK_FORMAT_B8G8R8A8_UNORM
	formatR32Sfloat    = 100 // VK_FORMAT_R32_SFLOAT
	formatRG32Sfloat   = 103 // VK_FORMAT_R32G32_SFLOAT
	formatRGB32Sfloat  = 106 // VK_FORMAT_R32G32B32_SFLOAT
	formatRGBA32Sfloat = 109 // VK_FORMAT_R32G32B32A32_SFLOAT

	filterNearest          = 0 // VK_FILTER_NEAREST
	filterLinear           = 1 // VK_FILTER_LINEAR
	mipmapNearest          = 0 // VK_SAMPLER_MIPMAP_MODE_NEAREST
	mipmapLinear           = 1 // VK_SAMPLER_MIPMAP_MODE_LINEAR
	addressRepeat          = 0 // VK_SAMPLER_ADDRESS_MODE_REPEAT
	addressMirror          = 1 // VK_SAMPLER_ADDRESS_MODE_MIRRORED_REPEAT
	addressClamp           = 2 // VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE
	borderTransparentBlack = 0 // VK_BORDER_COLOR_FLOAT_TRANSPARENT_BLACK
)

// Descriptors and pipelines.
const (
	descriptorSampler        = 0 // VK_DESCRIPTOR_TYPE_SAMPLER
	descriptorSampledImage   = 2 // VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE
	descriptorUniformDynamic = 8 // VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC

	stageVertex   = 0x00000001 // VK_SHADER_STAGE_VERTEX_BIT
	stageFragment = 0x00000010 // VK_SHADER_STAGE_FRAGMENT_BIT

	vertexInputRateVertex   = 0   // VK_VERTEX_INPUT_RATE_VERTEX
	vertexInputRateInstance = 1   // VK_VERTEX_INPUT_RATE_INSTANCE
	topologyTriangleList    = 3   // VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST
	polygonFill             = 0   // VK_POLYGON_MODE_FILL
	cullNone                = 0   // VK_CULL_MODE_NONE
	frontFaceCCW            = 0   // VK_FRONT_FACE_COUNTER_CLOCKWISE
	dynamicViewport         = 0   // VK_DYNAMIC_STATE_VIEWPORT
	dynamicScissor          = 1   // VK_DYNAMIC_STATE_SCISSOR
	colorComponentsAll      = 0xF // all four color components: the R, G, B and A bits

	blendZero             = 0 // VK_BLEND_FACTOR_ZERO
	blendOne              = 1 // VK_BLEND_FACTOR_ONE
	blendSrcAlpha         = 6 // VK_BLEND_FACTOR_SRC_ALPHA
	blendOneMinusSrcAlpha = 7 // VK_BLEND_FACTOR_ONE_MINUS_SRC_ALPHA
	blendDstAlpha         = 8 // VK_BLEND_FACTOR_DST_ALPHA
	blendOneMinusDstAlpha = 9 // VK_BLEND_FACTOR_ONE_MINUS_DST_ALPHA
	blendOpAdd            = 0 // VK_BLEND_OP_ADD

	descriptorPoolFreeSet = 0x00000001 // VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT
	indexTypeUint16       = 0          // VK_INDEX_TYPE_UINT16
	pipelineBindGraphics  = 0          // VK_PIPELINE_BIND_POINT_GRAPHICS
)

// Commands.
const (
	commandPoolTransient = 0x00000001 // VK_COMMAND_POOL_CREATE_TRANSIENT_BIT
	commandBufferPrimary = 0          // VK_COMMAND_BUFFER_LEVEL_PRIMARY
	commandBufferOneTime = 0x00000001 // VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT
	fenceSignaled        = 0x00000001 // VK_FENCE_CREATE_SIGNALED_BIT
	loadOpLoad           = 0          // VK_ATTACHMENT_LOAD_OP_LOAD
	loadOpClear          = 1          // VK_ATTACHMENT_LOAD_OP_CLEAR
	storeOpStore         = 0          // VK_ATTACHMENT_STORE_OP_STORE
	stage2AllCommands    = 0x00010000 // VK_PIPELINE_STAGE_2_ALL_COMMANDS_BIT
	stage2Host           = 0x00004000 // VK_PIPELINE_STAGE_2_HOST_BIT
	access2MemoryRead    = 0x00008000 // VK_ACCESS_2_MEMORY_READ_BIT
	access2MemoryWrite   = 0x00010000 // VK_ACCESS_2_MEMORY_WRITE_BIT
	access2HostRead      = 0x00002000 // VK_ACCESS_2_HOST_READ_BIT
	queueFamilyIgnored   = 0xFFFFFFFF // VK_QUEUE_FAMILY_IGNORED
)

// Sizes of fixed arrays and special values.
const (
	maxMemoryTypes = 32                 // VK_MAX_MEMORY_TYPES
	maxMemoryHeaps = 16                 // VK_MAX_MEMORY_HEAPS
	maxNameSize    = 256                // VK_MAX_PHYSICAL_DEVICE_NAME_SIZE
	uuidSize       = 16                 // VK_UUID_SIZE
	wholeSize      = 0xFFFFFFFFFFFFFFFF // VK_WHOLE_SIZE
)

// Window system integration (VK_KHR_surface, VK_KHR_swapchain and the surface
// extensions of Wayland and Xlib).
const (
	resultSuboptimal          = 1000001003  // VK_SUBOPTIMAL_KHR
	resultErrorSurfaceLost    = -1000000000 // VK_ERROR_SURFACE_LOST_KHR
	resultErrorWindowInUse    = -1000000001 // VK_ERROR_NATIVE_WINDOW_IN_USE_KHR
	resultErrorOutOfDate      = -1000001004 // VK_ERROR_OUT_OF_DATE_KHR
	sTypeSemaphoreCreateInfo  = 9           // VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO
	sTypeSwapchainCreateInfo  = 1000001000  // VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR
	sTypePresentInfo          = 1000001001  // VK_STRUCTURE_TYPE_PRESENT_INFO_KHR
	sTypeXlibSurfaceCreate    = 1000004000  // VK_STRUCTURE_TYPE_XLIB_SURFACE_CREATE_INFO_KHR
	sTypeWaylandSurfaceCreate = 1000006000  // VK_STRUCTURE_TYPE_WAYLAND_SURFACE_CREATE_INFO_KHR
	sTypePresentRegions       = 1000084000  // VK_STRUCTURE_TYPE_PRESENT_REGIONS_KHR

	presentModeImmediate = 0 // VK_PRESENT_MODE_IMMEDIATE_KHR
	presentModeMailbox   = 1 // VK_PRESENT_MODE_MAILBOX_KHR
	presentModeFIFO      = 2 // VK_PRESENT_MODE_FIFO_KHR

	colorSpaceSRGBNonlinear  = 0          // VK_COLOR_SPACE_SRGB_NONLINEAR_KHR
	compositeAlphaOpaque     = 0x00000001 // VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR
	surfaceTransformIdentity = 0x00000001 // VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR
	layoutPresentSrc         = 1000001002 // VK_IMAGE_LAYOUT_PRESENT_SRC_KHR
	stageAllCommands         = 0x00010000 // VK_PIPELINE_STAGE_ALL_COMMANDS_BIT
	currentExtentUndefined   = 0xFFFFFFFF // the "window decides" value of VkSurfaceCapabilitiesKHR::currentExtent
)

// Names of the extensions that windows need.
const (
	extSurface            = "VK_KHR_surface"
	extWaylandSurface     = "VK_KHR_wayland_surface"
	extXlibSurface        = "VK_KHR_xlib_surface"
	extSwapchain          = "VK_KHR_swapchain"
	extIncrementalPresent = "VK_KHR_incremental_present"
)
