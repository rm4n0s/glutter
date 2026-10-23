//go:build linux

package vulkan

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// surfaceExtensions are the instance extensions that windows of the kind need
// (SPEC §9.4): VK_KHR_surface and the one of the window system.
func surfaceExtensions(kind gpu.SurfaceKind) ([]string, error) {
	switch kind {
	case gpu.SurfaceKindWayland:
		return []string{extSurface, extWaylandSurface}, nil
	case gpu.SurfaceKindX11:
		return []string{extSurface, extXlibSurface}, nil
	}
	return nil, errors.New(tagVulkanSurfaceFailed, "there is no Vulkan surface for this kind of window", "kind", string(kind))
}

// surface is a VkSurfaceKHR of a window. A device has one for each window that has a
// swapchain.
type surface struct {
	handle surfaceH
	key    gpu.SurfaceHandle
}

// createSurface makes the VkSurfaceKHR of the native window of h: on Wayland from the
// wl_display and the wl_surface, on X11 from the Xlib display and the window ID. It
// fails with VulkanSurfaceFailed when the kind has no surface, the instance lacks the
// extension, or the driver cannot make the surface (a window that already has a
// renderer, for one).
func (i *instance) createSurface(h gpu.SurfaceHandle) (surfaceH, error) {
	a := &arena{}
	defer a.free()
	out := a.out()
	var result int32
	switch h.Kind {
	case gpu.SurfaceKindWayland:
		if i.f.createWaylandSurface == nil || h.Display == 0 || h.Surface == 0 {
			return 0, errors.New(tagVulkanSurfaceFailed, "there is no Wayland surface to draw into, or the instance lacks VK_KHR_wayland_surface",
				"display", h.Display != 0, "surface", h.Surface != 0)
		}
		info := put(a, waylandSurfaceCreateInfo{sType: sTypeWaylandSurfaceCreate, display: h.Display, surface: h.Surface})
		result = i.f.createWaylandSurface(i.handle, info, 0, out)
	case gpu.SurfaceKindX11:
		if i.f.createXlibSurface == nil || h.Display == 0 || h.Surface == 0 {
			return 0, errors.New(tagVulkanSurfaceFailed, "there is no X11 window to draw into, or the instance lacks VK_KHR_xlib_surface",
				"display", h.Display != 0, "window", h.Surface != 0)
		}
		info := put(a, xlibSurfaceCreateInfo{sType: sTypeXlibSurfaceCreate, dpy: h.Display, window: h.Surface})
		result = i.f.createXlibSurface(i.handle, info, 0, out)
	default:
		return 0, errors.New(tagVulkanSurfaceFailed, "there is no Vulkan surface for this kind of window", "kind", string(h.Kind))
	}
	if result != resultSuccess {
		return 0, surfaceError("vkCreateSurface", result)
	}
	return u64(out), nil
}

// surfaceError is vkError with the tag of the surface for the results that mean the
// window cannot be used.
func surfaceError(call string, result int32) error {
	switch result {
	case resultErrorSurfaceLost, resultErrorWindowInUse, resultErrorIncompatible, resultErrorExtNotPres:
		return errors.New(tagVulkanSurfaceFailed, "the window cannot be drawn into with Vulkan", "call", call, "result", int(result), "name", resultNames[result])
	}
	return vkError(call, result)
}

// surfaceFor returns the surface of the window, making it on first use.
func (d *Device) surfaceFor(h gpu.SurfaceHandle) (*surface, error) {
	if s, ok := d.surfaces[h]; ok {
		return s, nil
	}
	handle, err := d.inst.createSurface(h)
	if err != nil {
		return nil, err
	}
	s := &surface{handle: handle, key: h}
	d.surfaces[h] = s
	return s, nil
}

// destroySurface destroys the surface and forgets it.
func (d *Device) destroySurface(s *surface) {
	if s.handle == 0 {
		return
	}
	d.inst.f.destroySurface(d.inst.handle, s.handle, 0)
	s.handle = 0
	delete(d.surfaces, s.key)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagVulkanSurfaceFailed = "VulkanSurfaceFailed"
)
