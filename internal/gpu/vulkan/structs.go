//go:build linux

package vulkan

// Go mirrors of the Vulkan structs that the backend builds or reads. Pointers are
// uintptr: a struct that points to other data lives in C memory (the arena), never in
// Go memory (SPEC §8.1, §9.4). Field names are the C names; the layout test
// (structs_layout_test.go) compiles a program against vulkan_core.h and compares the
// size and the offset of every field. Non-dispatchable handles are uint64 (only 64-bit
// targets are supported), dispatchable ones are uintptr.

type (
	instanceH       = uintptr
	physicalDeviceH = uintptr
	deviceH         = uintptr
	queueH          = uintptr
	commandBufferH  = uintptr

	bufferH              = uint64
	imageH               = uint64
	imageViewH           = uint64
	samplerH             = uint64
	deviceMemoryH        = uint64
	shaderModuleH        = uint64
	descriptorSetLayoutH = uint64
	descriptorPoolH      = uint64
	descriptorSetH       = uint64
	pipelineLayoutH      = uint64
	pipelineH            = uint64
	commandPoolH         = uint64
	fenceH               = uint64
	debugMessengerH      = uint64
	surfaceH             = uint64
	swapchainH           = uint64
	semaphoreH           = uint64
)

type applicationInfo struct {
	sType              uint32
	pNext              uintptr
	pApplicationName   uintptr
	applicationVersion uint32
	pEngineName        uintptr
	engineVersion      uint32
	apiVersion         uint32
}

type instanceCreateInfo struct {
	sType                   uint32
	pNext                   uintptr
	flags                   uint32
	pApplicationInfo        uintptr
	enabledLayerCount       uint32
	ppEnabledLayerNames     uintptr
	enabledExtensionCount   uint32
	ppEnabledExtensionNames uintptr
}

type extensionProperties struct {
	extensionName [maxNameSize]byte
	specVersion   uint32
}

type layerProperties struct {
	layerName             [maxNameSize]byte
	specVersion           uint32
	implementationVersion uint32
	description           [maxNameSize]byte
}

// physicalDeviceLimits holds the two limits that the backend reads; the rest of
// VkPhysicalDeviceLimits is padding.
type physicalDeviceLimits struct {
	maxImageDimension1D uint32
	maxImageDimension2D uint32
	_                   [62]uint64 // 8-byte aligned, as the struct is: it has VkDeviceSize members
}

type physicalDeviceProperties struct {
	apiVersion        uint32
	driverVersion     uint32
	vendorID          uint32
	deviceID          uint32
	deviceType        uint32
	deviceName        [maxNameSize]byte
	pipelineCacheUUID [uuidSize]byte
	limits            physicalDeviceLimits
	_                 [20]byte // sparseProperties
}

type extent3D struct {
	width, height, depth uint32
}

type extent2D struct {
	width, height uint32
}

type offset2D struct {
	x, y int32
}

type offset3D struct {
	x, y, z int32
}

type rect2D struct {
	offset offset2D
	extent extent2D
}

type viewport struct {
	x, y, width, height, minDepth, maxDepth float32
}

type queueFamilyProperties struct {
	queueFlags                  uint32
	queueCount                  uint32
	timestampValidBits          uint32
	minImageTransferGranularity extent3D
}

type memoryType struct {
	propertyFlags uint32
	heapIndex     uint32
}

type memoryHeap struct {
	size  uint64
	flags uint32
	_     uint32
}

type physicalDeviceMemoryProperties struct {
	memoryTypeCount uint32
	memoryTypes     [maxMemoryTypes]memoryType
	memoryHeapCount uint32
	memoryHeaps     [maxMemoryHeaps]memoryHeap
}

type physicalDeviceFeatures2 struct {
	sType    uint32
	pNext    uintptr
	features [55 * 4]byte // VkPhysicalDeviceFeatures: 55 VkBool32
}

type physicalDeviceDynamicRenderingFeatures struct {
	sType            uint32
	pNext            uintptr
	dynamicRendering uint32
	_                uint32
}

type physicalDeviceSynchronization2Features struct {
	sType            uint32
	pNext            uintptr
	synchronization2 uint32
	_                uint32
}

type deviceQueueCreateInfo struct {
	sType            uint32
	pNext            uintptr
	flags            uint32
	queueFamilyIndex uint32
	queueCount       uint32
	_                uint32
	pQueuePriorities uintptr
}

type deviceCreateInfo struct {
	sType                   uint32
	pNext                   uintptr
	flags                   uint32
	queueCreateInfoCount    uint32
	pQueueCreateInfos       uintptr
	enabledLayerCount       uint32
	_                       uint32
	ppEnabledLayerNames     uintptr
	enabledExtensionCount   uint32
	_                       uint32
	ppEnabledExtensionNames uintptr
	pEnabledFeatures        uintptr
}

type bufferCreateInfo struct {
	sType                 uint32
	pNext                 uintptr
	flags                 uint32
	_                     uint32
	size                  uint64
	usage                 uint32
	sharingMode           uint32
	queueFamilyIndexCount uint32
	_                     uint32
	pQueueFamilyIndices   uintptr
}

type memoryRequirements struct {
	size           uint64
	alignment      uint64
	memoryTypeBits uint32
	_              uint32
}

type memoryAllocateInfo struct {
	sType           uint32
	pNext           uintptr
	allocationSize  uint64
	memoryTypeIndex uint32
	_               uint32
}

type imageCreateInfo struct {
	sType                 uint32
	pNext                 uintptr
	flags                 uint32
	imageType             uint32
	format                uint32
	extent                extent3D
	mipLevels             uint32
	arrayLayers           uint32
	samples               uint32
	tiling                uint32
	usage                 uint32
	sharingMode           uint32
	queueFamilyIndexCount uint32
	_                     uint32
	pQueueFamilyIndices   uintptr
	initialLayout         uint32
	_                     uint32
}

type componentMapping struct {
	r, g, b, a uint32
}

type imageSubresourceRange struct {
	aspectMask     uint32
	baseMipLevel   uint32
	levelCount     uint32
	baseArrayLayer uint32
	layerCount     uint32
}

type imageSubresourceLayers struct {
	aspectMask     uint32
	mipLevel       uint32
	baseArrayLayer uint32
	layerCount     uint32
}

type imageViewCreateInfo struct {
	sType            uint32
	pNext            uintptr
	flags            uint32
	_                uint32
	image            uint64
	viewType         uint32
	format           uint32
	components       componentMapping
	subresourceRange imageSubresourceRange
}

type samplerCreateInfo struct {
	sType                   uint32
	pNext                   uintptr
	flags                   uint32
	magFilter               uint32
	minFilter               uint32
	mipmapMode              uint32
	addressModeU            uint32
	addressModeV            uint32
	addressModeW            uint32
	mipLodBias              float32
	anisotropyEnable        uint32
	maxAnisotropy           float32
	compareEnable           uint32
	compareOp               uint32
	minLod                  float32
	maxLod                  float32
	borderColor             uint32
	unnormalizedCoordinates uint32
}

type shaderModuleCreateInfo struct {
	sType    uint32
	pNext    uintptr
	flags    uint32
	_        uint32
	codeSize uintptr
	pCode    uintptr
}

type descriptorSetLayoutBinding struct {
	binding            uint32
	descriptorType     uint32
	descriptorCount    uint32
	stageFlags         uint32
	pImmutableSamplers uintptr
}

type descriptorSetLayoutCreateInfo struct {
	sType        uint32
	pNext        uintptr
	flags        uint32
	bindingCount uint32
	pBindings    uintptr
}

type pipelineLayoutCreateInfo struct {
	sType                  uint32
	pNext                  uintptr
	flags                  uint32
	setLayoutCount         uint32
	pSetLayouts            uintptr
	pushConstantRangeCount uint32
	_                      uint32
	pPushConstantRanges    uintptr
}

type pipelineShaderStageCreateInfo struct {
	sType               uint32
	pNext               uintptr
	flags               uint32
	stage               uint32
	module              uint64
	pName               uintptr
	pSpecializationInfo uintptr
}

type vertexInputBindingDescription struct {
	binding   uint32
	stride    uint32
	inputRate uint32
}

type vertexInputAttributeDescription struct {
	location uint32
	binding  uint32
	format   uint32
	offset   uint32
}

type pipelineVertexInputStateCreateInfo struct {
	sType                           uint32
	pNext                           uintptr
	flags                           uint32
	vertexBindingDescriptionCount   uint32
	pVertexBindingDescriptions      uintptr
	vertexAttributeDescriptionCount uint32
	_                               uint32
	pVertexAttributeDescriptions    uintptr
}

type pipelineInputAssemblyStateCreateInfo struct {
	sType                  uint32
	pNext                  uintptr
	flags                  uint32
	topology               uint32
	primitiveRestartEnable uint32
	_                      uint32
}

type pipelineViewportStateCreateInfo struct {
	sType         uint32
	pNext         uintptr
	flags         uint32
	viewportCount uint32
	pViewports    uintptr
	scissorCount  uint32
	_             uint32
	pScissors     uintptr
}

type pipelineRasterizationStateCreateInfo struct {
	sType                   uint32
	pNext                   uintptr
	flags                   uint32
	depthClampEnable        uint32
	rasterizerDiscardEnable uint32
	polygonMode             uint32
	cullMode                uint32
	frontFace               uint32
	depthBiasEnable         uint32
	depthBiasConstantFactor float32
	depthBiasClamp          float32
	depthBiasSlopeFactor    float32
	lineWidth               float32
	_                       uint32
}

type pipelineMultisampleStateCreateInfo struct {
	sType                 uint32
	pNext                 uintptr
	flags                 uint32
	rasterizationSamples  uint32
	sampleShadingEnable   uint32
	minSampleShading      float32
	pSampleMask           uintptr
	alphaToCoverageEnable uint32
	alphaToOneEnable      uint32
}

type pipelineColorBlendAttachmentState struct {
	blendEnable         uint32
	srcColorBlendFactor uint32
	dstColorBlendFactor uint32
	colorBlendOp        uint32
	srcAlphaBlendFactor uint32
	dstAlphaBlendFactor uint32
	alphaBlendOp        uint32
	colorWriteMask      uint32
}

type pipelineColorBlendStateCreateInfo struct {
	sType           uint32
	pNext           uintptr
	flags           uint32
	logicOpEnable   uint32
	logicOp         uint32
	attachmentCount uint32
	pAttachments    uintptr
	blendConstants  [4]float32
}

type pipelineDynamicStateCreateInfo struct {
	sType             uint32
	pNext             uintptr
	flags             uint32
	dynamicStateCount uint32
	pDynamicStates    uintptr
}

type pipelineRenderingCreateInfo struct {
	sType                   uint32
	pNext                   uintptr
	viewMask                uint32
	colorAttachmentCount    uint32
	pColorAttachmentFormats uintptr
	depthAttachmentFormat   uint32
	stencilAttachmentFormat uint32
}

type graphicsPipelineCreateInfo struct {
	sType               uint32
	pNext               uintptr
	flags               uint32
	stageCount          uint32
	pStages             uintptr
	pVertexInputState   uintptr
	pInputAssemblyState uintptr
	pTessellationState  uintptr
	pViewportState      uintptr
	pRasterizationState uintptr
	pMultisampleState   uintptr
	pDepthStencilState  uintptr
	pColorBlendState    uintptr
	pDynamicState       uintptr
	layout              uint64
	renderPass          uint64
	subpass             uint32
	_                   uint32
	basePipelineHandle  uint64
	basePipelineIndex   int32
	_                   uint32
}

type descriptorPoolSize struct {
	typ             uint32
	descriptorCount uint32
}

type descriptorPoolCreateInfo struct {
	sType         uint32
	pNext         uintptr
	flags         uint32
	maxSets       uint32
	poolSizeCount uint32
	_             uint32
	pPoolSizes    uintptr
}

type descriptorSetAllocateInfo struct {
	sType              uint32
	pNext              uintptr
	descriptorPool     uint64
	descriptorSetCount uint32
	_                  uint32
	pSetLayouts        uintptr
}

type descriptorBufferInfo struct {
	buffer uint64
	offset uint64
	rng    uint64 // range
}

type descriptorImageInfo struct {
	sampler     uint64
	imageView   uint64
	imageLayout uint32
	_           uint32
}

type writeDescriptorSet struct {
	sType            uint32
	pNext            uintptr
	dstSet           uint64
	dstBinding       uint32
	dstArrayElement  uint32
	descriptorCount  uint32
	descriptorType   uint32
	pImageInfo       uintptr
	pBufferInfo      uintptr
	pTexelBufferView uintptr
}

type commandPoolCreateInfo struct {
	sType            uint32
	pNext            uintptr
	flags            uint32
	queueFamilyIndex uint32
}

type commandBufferAllocateInfo struct {
	sType              uint32
	pNext              uintptr
	commandPool        uint64
	level              uint32
	commandBufferCount uint32
}

type commandBufferBeginInfo struct {
	sType            uint32
	pNext            uintptr
	flags            uint32
	_                uint32
	pInheritanceInfo uintptr
}

type fenceCreateInfo struct {
	sType uint32
	pNext uintptr
	flags uint32
	_     uint32
}

type submitInfo struct {
	sType                uint32
	pNext                uintptr
	waitSemaphoreCount   uint32
	_                    uint32
	pWaitSemaphores      uintptr
	pWaitDstStageMask    uintptr
	commandBufferCount   uint32
	_                    uint32
	pCommandBuffers      uintptr
	signalSemaphoreCount uint32
	_                    uint32
	pSignalSemaphores    uintptr
}

// clearValue is VkClearValue as the color member of its union: four floats, then
// padding to the 16 bytes of the union.
type clearValue struct {
	color [4]float32
}

type renderingAttachmentInfo struct {
	sType              uint32
	pNext              uintptr
	imageView          uint64
	imageLayout        uint32
	resolveMode        uint32
	resolveImageView   uint64
	resolveImageLayout uint32
	loadOp             uint32
	storeOp            uint32
	clearValue         clearValue
}

type renderingInfo struct {
	sType                uint32
	pNext                uintptr
	flags                uint32
	renderArea           rect2D
	layerCount           uint32
	viewMask             uint32
	colorAttachmentCount uint32
	pColorAttachments    uintptr
	pDepthAttachment     uintptr
	pStencilAttachment   uintptr
}

type memoryBarrier2 struct {
	sType         uint32
	pNext         uintptr
	srcStageMask  uint64
	srcAccessMask uint64
	dstStageMask  uint64
	dstAccessMask uint64
}

type imageMemoryBarrier2 struct {
	sType               uint32
	pNext               uintptr
	srcStageMask        uint64
	srcAccessMask       uint64
	dstStageMask        uint64
	dstAccessMask       uint64
	oldLayout           uint32
	newLayout           uint32
	srcQueueFamilyIndex uint32
	dstQueueFamilyIndex uint32
	image               uint64
	subresourceRange    imageSubresourceRange
	_                   uint32
}

type dependencyInfo struct {
	sType                    uint32
	pNext                    uintptr
	dependencyFlags          uint32
	memoryBarrierCount       uint32
	pMemoryBarriers          uintptr
	bufferMemoryBarrierCount uint32
	_                        uint32
	pBufferMemoryBarriers    uintptr
	imageMemoryBarrierCount  uint32
	_                        uint32
	pImageMemoryBarriers     uintptr
}

type bufferImageCopy struct {
	bufferOffset      uint64
	bufferRowLength   uint32
	bufferImageHeight uint32
	imageSubresource  imageSubresourceLayers
	imageOffset       offset3D
	imageExtent       extent3D
}

type imageCopy struct {
	srcSubresource imageSubresourceLayers
	srcOffset      offset3D
	dstSubresource imageSubresourceLayers
	dstOffset      offset3D
	extent         extent3D
}

type debugUtilsMessengerCreateInfo struct {
	sType           uint32
	pNext           uintptr
	flags           uint32
	messageSeverity uint32
	messageType     uint32
	_               uint32
	pfnUserCallback uintptr
	pUserData       uintptr
}

// debugUtilsMessengerCallbackData is VkDebugUtilsMessengerCallbackDataEXT up to the
// message; the rest is not read.
type debugUtilsMessengerCallbackData struct {
	sType            uint32
	pNext            uintptr
	flags            uint32
	_                uint32
	pMessageIdName   uintptr
	messageIdNumber  int32
	_                uint32
	pMessage         uintptr
	queueLabelCount  uint32
	_                uint32
	pQueueLabels     uintptr
	cmdBufLabelCount uint32
	_                uint32
	pCmdBufLabels    uintptr
	objectCount      uint32
	_                uint32
	pObjects         uintptr
}

type waylandSurfaceCreateInfo struct {
	sType   uint32
	pNext   uintptr
	flags   uint32
	_       uint32
	display uintptr
	surface uintptr
}

// androidSurfaceCreateInfo is VkAndroidSurfaceCreateInfoKHR (vulkan_android.h, which needs the
// NDK's headers, so its layout test is written by hand).
type androidSurfaceCreateInfo struct {
	sType  uint32
	pNext  uintptr
	flags  uint32
	_      uint32
	window uintptr
}

type xlibSurfaceCreateInfo struct {
	sType  uint32
	pNext  uintptr
	flags  uint32
	_      uint32
	dpy    uintptr
	window uintptr
}

type surfaceCapabilities struct {
	minImageCount           uint32
	maxImageCount           uint32
	currentExtent           extent2D
	minImageExtent          extent2D
	maxImageExtent          extent2D
	maxImageArrayLayers     uint32
	supportedTransforms     uint32
	currentTransform        uint32
	supportedCompositeAlpha uint32
	supportedUsageFlags     uint32
}

type surfaceFormat struct {
	format     uint32
	colorSpace uint32
}

type swapchainCreateInfo struct {
	sType                 uint32
	pNext                 uintptr
	flags                 uint32
	_                     uint32
	surface               uint64
	minImageCount         uint32
	imageFormat           uint32
	imageColorSpace       uint32
	imageExtent           extent2D
	imageArrayLayers      uint32
	imageUsage            uint32
	imageSharingMode      uint32
	queueFamilyIndexCount uint32
	_                     uint32
	pQueueFamilyIndices   uintptr
	preTransform          uint32
	compositeAlpha        uint32
	presentMode           uint32
	clipped               uint32
	oldSwapchain          uint64
}

type semaphoreCreateInfo struct {
	sType uint32
	pNext uintptr
	flags uint32
	_     uint32
}

type presentInfo struct {
	sType              uint32
	pNext              uintptr
	waitSemaphoreCount uint32
	_                  uint32
	pWaitSemaphores    uintptr
	swapchainCount     uint32
	_                  uint32
	pSwapchains        uintptr
	pImageIndices      uintptr
	pResults           uintptr
}

type rectLayer struct {
	offset offset2D
	extent extent2D
	layer  uint32
}

type presentRegion struct {
	rectangleCount uint32
	_              uint32
	pRectangles    uintptr
}

type presentRegions struct {
	sType          uint32
	pNext          uintptr
	swapchainCount uint32
	_              uint32
	pRegions       uintptr
}
