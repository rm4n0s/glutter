//go:build linux

package vulkan

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func TestChooseSurfaceFormat(t *testing.T) {
	const srgb = colorSpaceSRGBNonlinear
	tests := []struct {
		name    string
		formats []surfaceFormat
		wantVk  uint32
		wantTex explicit.TextureFormat
		ok      bool
	}{
		{"BGRA first", []surfaceFormat{{formatRGBA8Unorm, srgb}, {formatBGRA8Unorm, srgb}}, formatBGRA8Unorm, explicit.TextureFormatBGRA8Unorm, true},
		{"RGBA when there is no BGRA", []surfaceFormat{{formatRGBA8Unorm, srgb}}, formatRGBA8Unorm, explicit.TextureFormatRGBA8Unorm, true},
		{"never an SRGB format", []surfaceFormat{{50, srgb}, {43, srgb}}, 0, 0, false}, // VK_FORMAT_B8G8R8A8_SRGB is 50, R8G8B8A8_SRGB is 43
		{"another color space is skipped", []surfaceFormat{{formatBGRA8Unorm, 1000104001}, {formatRGBA8Unorm, srgb}}, formatRGBA8Unorm, explicit.TextureFormatRGBA8Unorm, true},
		{"no formats", nil, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vk, tex, ok := chooseSurfaceFormat(tt.formats)
			if ok != tt.ok || vk != tt.wantVk || tex != tt.wantTex {
				t.Errorf("chooseSurfaceFormat = %d, %d, %v; want %d, %d, %v", vk, tex, ok, tt.wantVk, tt.wantTex, tt.ok)
			}
		})
	}
}

func TestChoosePresentMode(t *testing.T) {
	tests := []struct {
		name  string
		modes []uint32
		want  uint32
	}{
		{"mailbox first", []uint32{presentModeFIFO, presentModeImmediate, presentModeMailbox}, presentModeMailbox},
		{"immediate when there is no mailbox", []uint32{presentModeFIFO, presentModeImmediate}, presentModeImmediate},
		{"fifo only when the driver has neither", []uint32{presentModeFIFO}, presentModeFIFO},
		{"fifo when nothing is listed", nil, presentModeFIFO},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := choosePresentMode(tt.modes); got != tt.want {
				t.Errorf("choosePresentMode(%v) = %d, want %d", tt.modes, got, tt.want)
			}
		})
	}
}

func TestSurfaceExtensions(t *testing.T) {
	got, err := surfaceExtensions(gpu.SurfaceKindWayland)
	if err != nil || !reflect.DeepEqual(got, []string{"VK_KHR_surface", "VK_KHR_wayland_surface"}) {
		t.Errorf("Wayland: %v, %v", got, err)
	}
	got, err = surfaceExtensions(gpu.SurfaceKindX11)
	if err != nil || !reflect.DeepEqual(got, []string{"VK_KHR_surface", "VK_KHR_xlib_surface"}) {
		t.Errorf("X11: %v, %v", got, err)
	}
	_, err = surfaceExtensions(gpu.SurfaceKindFake)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagVulkanSurfaceFailed || !appErr.HasRoute("surfaceExtensions.VulkanSurfaceFailed") {
		t.Errorf("a fake surface: got %v, want VulkanSurfaceFailed via surfaceExtensions", err)
	}
}

func TestSurfaceAndSwapchainErrors(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		tag   string
		route string
	}{
		{"surface lost", surfaceError("vkCreateSurface", resultErrorSurfaceLost), "VulkanSurfaceFailed", "surfaceError.VulkanSurfaceFailed"},
		{"window in use", surfaceError("vkCreateSurface", resultErrorWindowInUse), "VulkanSurfaceFailed", "surfaceError.VulkanSurfaceFailed"},
		{"surface: out of memory", surfaceError("vkCreateSurface", resultErrorOutOfHost), "GPUOutOfMemory", "vkError.GPUOutOfMemory"},
		{"swapchain: surface lost", swapchainError("vkCreateSwapchainKHR", resultErrorSurfaceLost), "VulkanSwapchainFailed", "swapchainError.VulkanSwapchainFailed"},
		{"swapchain: initialization failed", swapchainError("vkCreateSwapchainKHR", resultErrorInitFailed), "VulkanSwapchainFailed", "swapchainError.VulkanSwapchainFailed"},
		{"swapchain: out of device memory", swapchainError("vkCreateSwapchainKHR", resultErrorOutOfDevice), "GPUOutOfMemory", "vkError.GPUOutOfMemory"},
		{"swapchain: device lost", swapchainError("vkQueuePresentKHR", resultErrorDeviceLost), "VulkanDeviceLost", "vkError.VulkanDeviceLost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.err)
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("got %v, want %s via %s", tt.err, tt.tag, tt.route)
			}
		})
	}
}

func TestAcquireReportsAnOutOfDateSwapchain(t *testing.T) {
	tests := []struct {
		name   string
		handle swapchainH
		result int32
	}{
		{"a swapchain that has no handle", 0, resultSuccess},
		{"the surface changed under the swapchain", 1, resultErrorOutOfDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Device{
				f:     &vkFuncs{acquireNextImage: func(uintptr, uint64, uint64, uint64, uint64, uintptr) int32 { return tt.result }},
				arena: &arena{},
				open:  &encoder{},
			}
			sc := &swapchain{d: d, handle: tt.handle}
			_, err := sc.Acquire()
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != explicit.SwapchainOutOfDate || !appErr.HasRoute("swapchain.Acquire."+explicit.SwapchainOutOfDate) {
				t.Fatalf("got %v, want %s via swapchain.Acquire", err, explicit.SwapchainOutOfDate)
			}
			if appErr.Metadata["call"] != "vkAcquireNextImageKHR" {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
			if !explicit.IsSwapchainOutOfDate(err) {
				t.Error("the renderer cannot tell that the swapchain is out of date")
			}
		})
	}
}

func TestCreateSurfaceNeedsTheNativeWindow(t *testing.T) {
	inst := &instance{f: &vkFuncs{}} // no surface functions: the instance lacks the extensions
	tests := []struct {
		name string
		h    gpu.SurfaceHandle
	}{
		{"a Wayland window on an instance without the extension", gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Display: 1, Surface: 2}},
		{"an X11 window on an instance without the extension", gpu.SurfaceHandle{Kind: gpu.SurfaceKindX11, Display: 1, Surface: 2}},
		{"a fake window", gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake}},
		{"no window", gpu.SurfaceHandle{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := inst.createSurface(tt.h)
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagVulkanSurfaceFailed || !appErr.HasRoute("instance.createSurface.VulkanSurfaceFailed") {
				t.Errorf("got %v, want VulkanSurfaceFailed via instance.createSurface", err)
			}
		})
	}
}

func TestChooseDeviceNeedsTheSwapchainForWindows(t *testing.T) {
	d := testGPU(0, "No Swapchain", deviceTypeDiscrete)
	_, _, err := chooseDevice([]physicalInfo{d}, chooseOpts{RequireSwapchain: true})
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagVulkanUnsupported {
		t.Fatalf("got %v, want VulkanUnsupported", err)
	}
	d.Extensions = map[string]bool{extSwapchain: true}
	if _, _, err := chooseDevice([]physicalInfo{d}, chooseOpts{RequireSwapchain: true}); err != nil {
		t.Errorf("a device with the extension: %v", err)
	}
	// Without windows the extension is not needed.
	if _, _, err := chooseDevice([]physicalInfo{testGPU(0, "Offscreen", deviceTypeDiscrete)}, chooseOpts{}); err != nil {
		t.Errorf("offscreen: %v", err)
	}
}
