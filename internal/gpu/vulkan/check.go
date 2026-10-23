//go:build linux

package vulkan

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// resultNames are the names of the VkResult values the backend can meet, for the
// message of an error.
var resultNames = map[int32]string{
	resultNotReady:            "VK_NOT_READY",
	resultTimeout:             "VK_TIMEOUT",
	resultIncomplete:          "VK_INCOMPLETE",
	resultErrorOutOfHost:      "VK_ERROR_OUT_OF_HOST_MEMORY",
	resultErrorOutOfDevice:    "VK_ERROR_OUT_OF_DEVICE_MEMORY",
	resultErrorInitFailed:     "VK_ERROR_INITIALIZATION_FAILED",
	resultErrorDeviceLost:     "VK_ERROR_DEVICE_LOST",
	resultErrorLayerNotPres:   "VK_ERROR_LAYER_NOT_PRESENT",
	resultErrorExtNotPres:     "VK_ERROR_EXTENSION_NOT_PRESENT",
	resultErrorFeatureNotPres: "VK_ERROR_FEATURE_NOT_PRESENT",
	resultErrorIncompatible:   "VK_ERROR_INCOMPATIBLE_DRIVER",
	resultErrorFragmentedPool: "VK_ERROR_FRAGMENTED_POOL",
	resultErrorOutOfPool:      "VK_ERROR_OUT_OF_POOL_MEMORY",
}

// vkError turns a VkResult that is not VK_SUCCESS into the tagged error of SPEC §9.4:
// GPUOutOfMemory for the two out-of-memory results, VulkanDeviceLost for a lost
// device, and VulkanCallFailed for the others. call names the Vulkan function.
func vkError(call string, result int32) error {
	name, ok := resultNames[result]
	if !ok {
		name = "unknown result"
	}
	switch result {
	case resultErrorOutOfHost, resultErrorOutOfDevice:
		return errors.New(tagGPUOutOfMemory, "the GPU or the system is out of memory", "call", call, "result", int(result), "name", name)
	case resultErrorDeviceLost:
		return errors.New(explicit.TagVulkanDeviceLost, "the Vulkan device was lost", "call", call, "result", int(result), "name", name)
	}
	return errors.New(tagVulkanCallFailed, "a Vulkan call failed", "call", call, "result", int(result), "name", name)
}

// check returns nil for VK_SUCCESS and vkError otherwise.
func check(call string, result int32) error {
	if result == resultSuccess {
		return nil
	}
	return vkError(call, result)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGPUOutOfMemory   = "GPUOutOfMemory"
	tagVulkanCallFailed = "VulkanCallFailed"
)
