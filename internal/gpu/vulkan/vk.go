//go:build linux

package vulkan

import (
	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// vkFuncs is the table of the Vulkan functions that the backend calls (SPEC §9.4), as
// gl.go has for GLES. Each field is bound by name: the global and instance functions
// through vkGetInstanceProcAddr, the device functions through vkGetDeviceProcAddr. No
// function takes or returns a struct by value (TestNoFunctionTakesAStructByValue):
// structs go by pointer, to C memory (SPEC §8.1). Handles that are not dispatchable
// are uint64, so only 64-bit targets are supported.
type vkFuncs struct {
	// Loader and global functions.
	getInstanceProcAddr             func(instance uintptr, name uintptr) uintptr
	enumerateInstanceVersion        func(pVersion uintptr) int32
	enumerateInstanceExtensionProps func(layer, pCount, pProps uintptr) int32
	enumerateInstanceLayerProps     func(pCount, pProps uintptr) int32
	createInstance                  func(pInfo, pAllocator, pInstance uintptr) int32

	// Instance functions.
	destroyInstance                   func(instance, pAllocator uintptr)
	enumeratePhysicalDevices          func(instance, pCount, pDevices uintptr) int32
	getPhysicalDeviceProperties       func(phys, pProps uintptr)
	getPhysicalDeviceQueueFamilyProps func(phys, pCount, pProps uintptr)
	getPhysicalDeviceMemoryProps      func(phys, pProps uintptr)
	enumerateDeviceExtensionProps     func(phys, layer, pCount, pProps uintptr) int32
	getPhysicalDeviceFeatures2        func(phys, pFeatures uintptr)
	createDevice                      func(phys, pInfo, pAllocator, pDevice uintptr) int32
	getDeviceProcAddr                 func(device uintptr, name uintptr) uintptr
	createDebugUtilsMessenger         func(instance, pInfo, pAllocator, pMessenger uintptr) int32
	destroyDebugUtilsMessenger        func(instance uintptr, messenger uint64, pAllocator uintptr)
	destroySurface                    func(instance uintptr, surface uint64, pAllocator uintptr)
	createWaylandSurface              func(instance, pInfo, pAllocator, pSurface uintptr) int32
	createXlibSurface                 func(instance, pInfo, pAllocator, pSurface uintptr) int32
	createAndroidSurface              func(instance, pInfo, pAllocator, pSurface uintptr) int32
	getPhysicalDeviceSurfaceSupport   func(phys uintptr, family uint32, surface uint64, pSupported uintptr) int32
	getPhysicalDeviceSurfaceCaps      func(phys uintptr, surface uint64, pCaps uintptr) int32
	getPhysicalDeviceSurfaceFormats   func(phys uintptr, surface uint64, pCount, pFormats uintptr) int32
	getPhysicalDeviceSurfaceModes     func(phys uintptr, surface uint64, pCount, pModes uintptr) int32

	// Device functions.
	destroyDevice               func(device, pAllocator uintptr)
	getDeviceQueue              func(device uintptr, family, index uint32, pQueue uintptr)
	queueSubmit                 func(queue uintptr, count uint32, pSubmits uintptr, fence uint64) int32
	deviceWaitIdle              func(device uintptr) int32
	createBuffer                func(device, pInfo, pAllocator, pBuffer uintptr) int32
	destroyBuffer               func(device uintptr, buffer uint64, pAllocator uintptr)
	getBufferMemoryRequirements func(device uintptr, buffer uint64, pReq uintptr)
	bindBufferMemory            func(device uintptr, buffer, mem, offset uint64) int32
	allocateMemory              func(device, pInfo, pAllocator, pMemory uintptr) int32
	freeMemory                  func(device uintptr, mem uint64, pAllocator uintptr)
	mapMemory                   func(device uintptr, mem, offset, size uint64, flags uint32, ppData uintptr) int32
	unmapMemory                 func(device uintptr, mem uint64)
	createImage                 func(device, pInfo, pAllocator, pImage uintptr) int32
	destroyImage                func(device uintptr, image uint64, pAllocator uintptr)
	getImageMemoryRequirements  func(device uintptr, image uint64, pReq uintptr)
	bindImageMemory             func(device uintptr, image, mem, offset uint64) int32
	createImageView             func(device, pInfo, pAllocator, pView uintptr) int32
	destroyImageView            func(device uintptr, view uint64, pAllocator uintptr)
	createSampler               func(device, pInfo, pAllocator, pSampler uintptr) int32
	destroySampler              func(device uintptr, sampler uint64, pAllocator uintptr)
	createShaderModule          func(device, pInfo, pAllocator, pModule uintptr) int32
	destroyShaderModule         func(device uintptr, module uint64, pAllocator uintptr)
	createDescriptorSetLayout   func(device, pInfo, pAllocator, pLayout uintptr) int32
	destroyDescriptorSetLayout  func(device uintptr, layout uint64, pAllocator uintptr)
	createPipelineLayout        func(device, pInfo, pAllocator, pLayout uintptr) int32
	destroyPipelineLayout       func(device uintptr, layout uint64, pAllocator uintptr)
	createGraphicsPipelines     func(device uintptr, cache uint64, count uint32, pInfos, pAllocator, pPipelines uintptr) int32
	createPipelineCache         func(device, pInfo, pAllocator, pCache uintptr) int32
	destroyPipelineCache        func(device uintptr, cache uint64, pAllocator uintptr)
	getPipelineCacheData        func(device uintptr, cache uint64, pDataSize, pData uintptr) int32
	destroyPipeline             func(device uintptr, pipeline uint64, pAllocator uintptr)
	createDescriptorPool        func(device, pInfo, pAllocator, pPool uintptr) int32
	destroyDescriptorPool       func(device uintptr, pool uint64, pAllocator uintptr)
	resetDescriptorPool         func(device uintptr, pool uint64, flags uint32) int32
	allocateDescriptorSets      func(device, pInfo, pSets uintptr) int32
	updateDescriptorSets        func(device uintptr, writeCount uint32, pWrites uintptr, copyCount uint32, pCopies uintptr)
	createCommandPool           func(device, pInfo, pAllocator, pPool uintptr) int32
	destroyCommandPool          func(device uintptr, pool uint64, pAllocator uintptr)
	resetCommandPool            func(device uintptr, pool uint64, flags uint32) int32
	allocateCommandBuffers      func(device, pInfo, pBuffers uintptr) int32
	beginCommandBuffer          func(cb, pInfo uintptr) int32
	endCommandBuffer            func(cb uintptr) int32
	cmdBeginRendering           func(cb, pInfo uintptr)
	cmdEndRendering             func(cb uintptr)
	cmdBindPipeline             func(cb uintptr, bindPoint uint32, pipeline uint64)
	cmdBindDescriptorSets       func(cb uintptr, bindPoint uint32, layout uint64, first, count uint32, pSets uintptr, dynCount uint32, pDyn uintptr)
	cmdBindVertexBuffers        func(cb uintptr, first, count uint32, pBuffers, pOffsets uintptr)
	cmdBindIndexBuffer          func(cb uintptr, buffer, offset uint64, indexType uint32)
	cmdSetViewport              func(cb uintptr, first, count uint32, pViewports uintptr)
	cmdSetScissor               func(cb uintptr, first, count uint32, pScissors uintptr)
	cmdDraw                     func(cb uintptr, vertices, instances, firstVertex, firstInstance uint32)
	cmdDrawIndexed              func(cb uintptr, indices, instances, firstIndex uint32, vertexOffset int32, firstInstance uint32)
	cmdCopyImage                func(cb uintptr, src uint64, srcLayout uint32, dst uint64, dstLayout uint32, count uint32, pRegions uintptr)
	cmdCopyBufferToImage        func(cb uintptr, buffer, image uint64, layout, count uint32, pRegions uintptr)
	cmdCopyImageToBuffer        func(cb uintptr, image uint64, layout uint32, buffer uint64, count uint32, pRegions uintptr)
	cmdPipelineBarrier2         func(cb, pDependency uintptr)
	createFence                 func(device, pInfo, pAllocator, pFence uintptr) int32
	destroyFence                func(device uintptr, fence uint64, pAllocator uintptr)
	resetFences                 func(device uintptr, count uint32, pFences uintptr) int32
	waitForFences               func(device uintptr, count uint32, pFences uintptr, waitAll uint32, timeout uint64) int32
	createSemaphore             func(device, pInfo, pAllocator, pSemaphore uintptr) int32
	destroySemaphore            func(device uintptr, semaphore uint64, pAllocator uintptr)
	createSwapchain             func(device, pInfo, pAllocator, pSwapchain uintptr) int32
	destroySwapchain            func(device uintptr, swapchain uint64, pAllocator uintptr)
	getSwapchainImages          func(device uintptr, swapchain uint64, pCount, pImages uintptr) int32
	acquireNextImage            func(device uintptr, swapchain, timeout, semaphore, fence uint64, pIndex uintptr) int32
	queuePresent                func(queue, pInfo uintptr) int32
}

// level is the object that a function is looked up through.
type level int

const (
	levelGlobal level = iota
	levelInstance
	levelDevice
)

// symbol binds one function of the table: the names to try in order (the core name,
// then the extension's), and whether a missing function is an error.
type symbol struct {
	level    level
	names    []string
	fptr     any
	optional bool
}

// symbols lists the functions of the table with the level each is looked up at.
func (f *vkFuncs) symbols() []symbol {
	g := func(name string, fptr any, optional ...bool) symbol {
		return symbol{levelGlobal, []string{name}, fptr, len(optional) > 0 && optional[0]}
	}
	i := func(fptr any, names ...string) symbol { return symbol{levelInstance, names, fptr, false} }
	d := func(fptr any, names ...string) symbol { return symbol{levelDevice, names, fptr, false} }
	return []symbol{
		g("vkEnumerateInstanceVersion", &f.enumerateInstanceVersion, true), // missing in a Vulkan 1.0 loader
		g("vkEnumerateInstanceExtensionProperties", &f.enumerateInstanceExtensionProps),
		g("vkEnumerateInstanceLayerProperties", &f.enumerateInstanceLayerProps),
		g("vkCreateInstance", &f.createInstance),

		i(&f.destroyInstance, "vkDestroyInstance"),
		i(&f.enumeratePhysicalDevices, "vkEnumeratePhysicalDevices"),
		i(&f.getPhysicalDeviceProperties, "vkGetPhysicalDeviceProperties"),
		i(&f.getPhysicalDeviceQueueFamilyProps, "vkGetPhysicalDeviceQueueFamilyProperties"),
		i(&f.getPhysicalDeviceMemoryProps, "vkGetPhysicalDeviceMemoryProperties"),
		i(&f.enumerateDeviceExtensionProps, "vkEnumerateDeviceExtensionProperties"),
		i(&f.getPhysicalDeviceFeatures2, "vkGetPhysicalDeviceFeatures2", "vkGetPhysicalDeviceFeatures2KHR"),
		i(&f.createDevice, "vkCreateDevice"),
		i(&f.getDeviceProcAddr, "vkGetDeviceProcAddr"),
		{levelInstance, []string{"vkCreateDebugUtilsMessengerEXT"}, &f.createDebugUtilsMessenger, true},
		{levelInstance, []string{"vkDestroyDebugUtilsMessengerEXT"}, &f.destroyDebugUtilsMessenger, true},
		// The window system functions exist only when the instance has the extensions.
		{levelInstance, []string{"vkDestroySurfaceKHR"}, &f.destroySurface, true},
		{levelInstance, []string{"vkCreateWaylandSurfaceKHR"}, &f.createWaylandSurface, true},
		{levelInstance, []string{"vkCreateXlibSurfaceKHR"}, &f.createXlibSurface, true},
		{levelInstance, []string{"vkCreateAndroidSurfaceKHR"}, &f.createAndroidSurface, true},
		{levelInstance, []string{"vkGetPhysicalDeviceSurfaceSupportKHR"}, &f.getPhysicalDeviceSurfaceSupport, true},
		{levelInstance, []string{"vkGetPhysicalDeviceSurfaceCapabilitiesKHR"}, &f.getPhysicalDeviceSurfaceCaps, true},
		{levelInstance, []string{"vkGetPhysicalDeviceSurfaceFormatsKHR"}, &f.getPhysicalDeviceSurfaceFormats, true},
		{levelInstance, []string{"vkGetPhysicalDeviceSurfacePresentModesKHR"}, &f.getPhysicalDeviceSurfaceModes, true},

		d(&f.destroyDevice, "vkDestroyDevice"),
		d(&f.getDeviceQueue, "vkGetDeviceQueue"),
		d(&f.queueSubmit, "vkQueueSubmit"),
		d(&f.deviceWaitIdle, "vkDeviceWaitIdle"),
		d(&f.createBuffer, "vkCreateBuffer"),
		d(&f.destroyBuffer, "vkDestroyBuffer"),
		d(&f.getBufferMemoryRequirements, "vkGetBufferMemoryRequirements"),
		d(&f.bindBufferMemory, "vkBindBufferMemory"),
		d(&f.allocateMemory, "vkAllocateMemory"),
		d(&f.freeMemory, "vkFreeMemory"),
		d(&f.mapMemory, "vkMapMemory"),
		d(&f.unmapMemory, "vkUnmapMemory"),
		d(&f.createImage, "vkCreateImage"),
		d(&f.destroyImage, "vkDestroyImage"),
		d(&f.getImageMemoryRequirements, "vkGetImageMemoryRequirements"),
		d(&f.bindImageMemory, "vkBindImageMemory"),
		d(&f.createImageView, "vkCreateImageView"),
		d(&f.destroyImageView, "vkDestroyImageView"),
		d(&f.createSampler, "vkCreateSampler"),
		d(&f.destroySampler, "vkDestroySampler"),
		d(&f.createShaderModule, "vkCreateShaderModule"),
		d(&f.destroyShaderModule, "vkDestroyShaderModule"),
		d(&f.createDescriptorSetLayout, "vkCreateDescriptorSetLayout"),
		d(&f.destroyDescriptorSetLayout, "vkDestroyDescriptorSetLayout"),
		d(&f.createPipelineLayout, "vkCreatePipelineLayout"),
		d(&f.destroyPipelineLayout, "vkDestroyPipelineLayout"),
		d(&f.createGraphicsPipelines, "vkCreateGraphicsPipelines"),
		d(&f.createPipelineCache, "vkCreatePipelineCache"),
		d(&f.destroyPipelineCache, "vkDestroyPipelineCache"),
		d(&f.getPipelineCacheData, "vkGetPipelineCacheData"),
		d(&f.destroyPipeline, "vkDestroyPipeline"),
		d(&f.createDescriptorPool, "vkCreateDescriptorPool"),
		d(&f.destroyDescriptorPool, "vkDestroyDescriptorPool"),
		d(&f.resetDescriptorPool, "vkResetDescriptorPool"),
		d(&f.allocateDescriptorSets, "vkAllocateDescriptorSets"),
		d(&f.updateDescriptorSets, "vkUpdateDescriptorSets"),
		d(&f.createCommandPool, "vkCreateCommandPool"),
		d(&f.destroyCommandPool, "vkDestroyCommandPool"),
		d(&f.resetCommandPool, "vkResetCommandPool"),
		d(&f.allocateCommandBuffers, "vkAllocateCommandBuffers"),
		d(&f.beginCommandBuffer, "vkBeginCommandBuffer"),
		d(&f.endCommandBuffer, "vkEndCommandBuffer"),
		d(&f.cmdBeginRendering, "vkCmdBeginRendering", "vkCmdBeginRenderingKHR"),
		d(&f.cmdEndRendering, "vkCmdEndRendering", "vkCmdEndRenderingKHR"),
		d(&f.cmdBindPipeline, "vkCmdBindPipeline"),
		d(&f.cmdBindDescriptorSets, "vkCmdBindDescriptorSets"),
		d(&f.cmdBindVertexBuffers, "vkCmdBindVertexBuffers"),
		d(&f.cmdBindIndexBuffer, "vkCmdBindIndexBuffer"),
		d(&f.cmdSetViewport, "vkCmdSetViewport"),
		d(&f.cmdSetScissor, "vkCmdSetScissor"),
		d(&f.cmdDraw, "vkCmdDraw"),
		d(&f.cmdDrawIndexed, "vkCmdDrawIndexed"),
		d(&f.cmdCopyImage, "vkCmdCopyImage"),
		d(&f.cmdCopyBufferToImage, "vkCmdCopyBufferToImage"),
		d(&f.cmdCopyImageToBuffer, "vkCmdCopyImageToBuffer"),
		d(&f.cmdPipelineBarrier2, "vkCmdPipelineBarrier2", "vkCmdPipelineBarrier2KHR"),
		d(&f.createFence, "vkCreateFence"),
		d(&f.destroyFence, "vkDestroyFence"),
		d(&f.resetFences, "vkResetFences"),
		d(&f.waitForFences, "vkWaitForFences"),
		d(&f.createSemaphore, "vkCreateSemaphore"),
		d(&f.destroySemaphore, "vkDestroySemaphore"),
		{levelDevice, []string{"vkCreateSwapchainKHR"}, &f.createSwapchain, true},
		{levelDevice, []string{"vkDestroySwapchainKHR"}, &f.destroySwapchain, true},
		{levelDevice, []string{"vkGetSwapchainImagesKHR"}, &f.getSwapchainImages, true},
		{levelDevice, []string{"vkAcquireNextImageKHR"}, &f.acquireNextImage, true},
		{levelDevice, []string{"vkQueuePresentKHR"}, &f.queuePresent, true},
	}
}

// resolver finds the address of a function by name at a level, or 0.
type resolver func(lv level, name string) uintptr

// bind binds the functions of one level of the table, trying the names of each in
// order. A function that is missing is SymbolMissing, unless it is optional, in which
// case its field stays nil.
func (f *vkFuncs) bind(lv level, resolve resolver) error {
	for _, s := range f.symbols() {
		if s.level != lv {
			continue
		}
		var addr uintptr
		for _, name := range s.names {
			if addr = resolve(lv, name); addr != 0 {
				break
			}
		}
		if addr == 0 {
			if s.optional {
				continue
			}
			return errors.New(tagSymbolMissing, "a Vulkan function is not in the driver",
				"symbol", s.names[0], "library", "libvulkan")
		}
		bindFunc(s.fptr, addr)
	}
	return nil
}

// bindFunc makes the function pointer fptr (a pointer to a func field) call the C
// function at addr.
func bindFunc(fptr any, addr uintptr) { purego.RegisterFunc(fptr, addr) }

// loadLibrary opens libvulkan.so.1 (GLUTTER_VULKAN_LIBRARY overrides it, SPEC §9.4),
// binds vkGetInstanceProcAddr and the global functions. A library that cannot be loaded
// is VulkanLoadFailed; a global function that is missing is SymbolMissing.
func loadLibrary(getenv func(string) string) (*vkFuncs, error) {
	var (
		lib *dl.Lib
		err error
	)
	if path := getenv(libraryEnvVar); path != "" {
		lib, err = dl.Open(path)
	} else {
		lib, err = dl.Open("libvulkan.so.1", "libvulkan.so")
	}
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(tagVulkanLoadFailed, "could not load libvulkan.so.1: is a Vulkan driver installed?",
			"names", md["names"], "dlerror", md["dlerror"])
	}
	f := &vkFuncs{}
	if err := lib.Bind(&f.getInstanceProcAddr, "vkGetInstanceProcAddr"); err != nil {
		return nil, err
	}
	err = f.bind(levelGlobal, func(_ level, name string) uintptr { return f.procAddr(0, name) })
	if err != nil {
		return nil, err
	}
	return f, nil
}

// procAddr asks vkGetInstanceProcAddr for the function of the instance (0 for the
// global ones).
func (f *vkFuncs) procAddr(instance uintptr, name string) uintptr {
	c := dl.CString(name)
	defer dl.Free(c)
	return f.getInstanceProcAddr(instance, c)
}

// devProcAddr asks vkGetDeviceProcAddr for the function of the device.
func (f *vkFuncs) devProcAddr(device uintptr, name string) uintptr {
	c := dl.CString(name)
	defer dl.Free(c)
	return f.getDeviceProcAddr(device, c)
}

// bindInstance binds the instance functions.
func (f *vkFuncs) bindInstance(instance uintptr) error {
	return f.bind(levelInstance, func(_ level, name string) uintptr { return f.procAddr(instance, name) })
}

// bindDevice binds the device functions, which are called without going through the
// instance's dispatch.
func (f *vkFuncs) bindDevice(device uintptr) error {
	return f.bind(levelDevice, func(_ level, name string) uintptr { return f.devProcAddr(device, name) })
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSymbolMissing    = "SymbolMissing"
	tagVulkanLoadFailed = "VulkanLoadFailed"
)
