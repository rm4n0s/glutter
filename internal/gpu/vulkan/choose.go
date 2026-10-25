//go:build linux

package vulkan

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// Device extensions of the versions below 1.3: dynamic rendering and
// synchronization 2 come as extensions of Vulkan 1.1 and 1.2 (SPEC §9.4).
const (
	extDynamicRendering = "VK_KHR_dynamic_rendering"
	extSynchronization2 = "VK_KHR_synchronization2"
	minImageDimension2D = 8192 // the design limit of SPEC §9.3
	deviceEnvVar        = "GLUTTER_VULKAN_DEVICE"
	validationEnvVar    = "GLUTTER_VULKAN_VALIDATION"
	libraryEnvVar       = "GLUTTER_VULKAN_LIBRARY"
	validationLayerName = "VK_LAYER_KHRONOS_validation"
	debugUtilsExtension = "VK_EXT_debug_utils"
)

// extensionDependencies are the device extensions that VK_KHR_dynamic_rendering
// needs and that Vulkan 1.2 and 1.3 have in the core; below 1.2 they are enabled with it.
var extensionDependencies = []string{
	"VK_KHR_depth_stencil_resolve", "VK_KHR_create_renderpass2", "VK_KHR_multiview", "VK_KHR_maintenance2",
}

// physicalInfo is what the choice of a physical device needs to know about it.
type physicalInfo struct {
	// Index is the position in the loader's order.
	Index int
	// Name is deviceName.
	Name string
	// Type is VkPhysicalDeviceType.
	Type uint32
	// VendorID, DeviceID and DriverVersion are the fields of those names.
	VendorID, DeviceID, DriverVersion uint32
	// PipelineCacheUUID is pipelineCacheUUID: it identifies the driver build that wrote a pipeline cache.
	PipelineCacheUUID [uuidSize]byte
	// APIVersion is the version the device supports.
	APIVersion uint32
	// Extensions are the device extensions.
	Extensions map[string]bool
	// DynamicRendering and Synchronization2 are the features of those names.
	DynamicRendering, Synchronization2 bool
	// QueueFlags are the flags of each queue family, by index.
	QueueFlags []uint32
	// MaxImageDimension2D is the largest texture.
	MaxImageDimension2D uint32
}

// chooseOpts are the rules of the choice.
type chooseOpts struct {
	// Selector is the value of GLUTTER_VULKAN_DEVICE: an index or part of a device name
	// (any case). Empty takes the first device that works.
	Selector string
	// AllowCPU lets a device of type CPU (lavapipe) be chosen: only when Vulkan is
	// asked for by name (SPEC §9.4).
	AllowCPU bool
	// DenyKnownBad makes a device with a driver on the deny list (DriverInfo.IsKnownBadDriver)
	// unusable: openDevice checks it after the choice, and fails with VulkanDriverDenied.
	DenyKnownBad bool
	// RequireSwapchain makes a device that lacks VK_KHR_swapchain unsupported: the
	// device is for windows.
	RequireSwapchain bool
	// CanPresent, when not nil, says whether the queue family of the device can
	// present to the window's surface; the family must also have graphics.
	CanPresent func(device, family int) bool
}

// supports says why the device cannot be used, or "" when it can: Vulkan 1.3, or
// 1.1 or later with dynamic rendering and synchronization 2 (extensions and
// features), and textures of 8192 pixels.
func (p physicalInfo) supports(requireSwapchain bool) string {
	switch {
	case p.APIVersion < apiVersion11:
		return "it has no Vulkan 1.1"
	case p.APIVersion < apiVersion13 && (!p.Extensions[extDynamicRendering] || !p.Extensions[extSynchronization2]):
		return "below Vulkan 1.3 it needs VK_KHR_dynamic_rendering and VK_KHR_synchronization2"
	case !p.DynamicRendering:
		return "it does not support the dynamicRendering feature"
	case !p.Synchronization2:
		return "it does not support the synchronization2 feature"
	case requireSwapchain && !p.Extensions[extSwapchain]:
		return "it lacks VK_KHR_swapchain"
	case p.MaxImageDimension2D < minImageDimension2D:
		return fmt.Sprintf("its largest texture is %d pixels, and the design limit is %d", p.MaxImageDimension2D, minImageDimension2D)
	}
	return ""
}

// graphicsFamily returns the first queue family with graphics (and present) support.
func (p physicalInfo) graphicsFamily(o chooseOpts) (int, bool) {
	for i, flags := range p.QueueFlags {
		if flags&queueGraphics == 0 {
			continue
		}
		if o.CanPresent != nil && !o.CanPresent(p.Index, i) {
			continue
		}
		return i, true
	}
	return 0, false
}

// matches says whether the selector names the device: by index, or by part of its
// name.
func (p physicalInfo) matches(selector string) bool {
	if n, err := strconv.Atoi(selector); err == nil {
		return n == p.Index
	}
	return strings.Contains(strings.ToLower(p.Name), strings.ToLower(selector))
}

// chooseDevice picks the physical device and its queue family (SPEC §9.4): the first
// in the loader's order that supports what the backend needs and has a queue family
// with graphics and present support; a device of type CPU only when allowed; or the
// one that GLUTTER_VULKAN_DEVICE names, which must pass the same tests. It fails
// with VulkanNoDevice when there is no such device, and with VulkanUnsupported when
// the devices there are lack the version, extensions or features.
func chooseDevice(list []physicalInfo, o chooseOpts) (device, family int, err error) {
	if len(list) == 0 {
		return 0, 0, errors.New(tagVulkanNoDevice, "Vulkan has no physical device")
	}
	var rejected []string
	unsupported := false
	for _, p := range list {
		if o.Selector != "" && !p.matches(o.Selector) {
			rejected = append(rejected, fmt.Sprintf("%d %s: not the device that %s names", p.Index, p.Name, deviceEnvVar))
			continue
		}
		if why := p.supports(o.RequireSwapchain); why != "" {
			unsupported = true
			rejected = append(rejected, fmt.Sprintf("%d %s: %s", p.Index, p.Name, why))
			continue
		}
		if p.Type == deviceTypeCPU && !o.AllowCPU {
			rejected = append(rejected, fmt.Sprintf("%d %s: a CPU device is used only when Vulkan is asked for", p.Index, p.Name))
			continue
		}
		f, ok := p.graphicsFamily(o)
		if !ok {
			rejected = append(rejected, fmt.Sprintf("%d %s: no queue family with graphics and present support", p.Index, p.Name))
			continue
		}
		return p.Index, f, nil
	}
	if unsupported && (o.Selector == "" || allRejectedAsUnsupported(list, o)) {
		return 0, 0, errors.New(tagVulkanUnsupported, "no Vulkan device has the version, extensions and features that Glutter needs",
			"rejected", rejected)
	}
	return 0, 0, errors.New(tagVulkanNoDevice, "no Vulkan device can be used", "rejected", rejected, "selector", o.Selector)
}

// allRejectedAsUnsupported says whether every device that the selector names is
// rejected for what it supports.
func allRejectedAsUnsupported(list []physicalInfo, o chooseOpts) bool {
	named := 0
	for _, p := range list {
		if p.matches(o.Selector) {
			named++
			if p.supports(o.RequireSwapchain) == "" {
				return false
			}
		}
	}
	return named > 0
}

// checkDriver fails with VulkanDriverDenied when the chosen device's driver is on the deny list.
func checkDriver(info physicalInfo) error {
	driver := DriverInfo{Name: info.Name, VendorID: info.VendorID, DeviceID: info.DeviceID, DriverVersion: info.DriverVersion, APIVersion: info.APIVersion}
	if bad, why := driver.IsKnownBadDriver(); bad {
		return errors.New(tagVulkanDriverDenied, "Vulkan is not used with this driver: "+why,
			"device", info.Name, "vendor_id", int(info.VendorID), "device_id", int(info.DeviceID), "driver_version", int(info.DriverVersion))
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagVulkanNoDevice     = "VulkanNoDevice"
	tagVulkanUnsupported  = "VulkanUnsupported"
	tagVulkanDriverDenied = "VulkanDriverDenied"
)
