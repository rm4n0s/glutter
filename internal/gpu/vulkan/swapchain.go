//go:build linux

package vulkan

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// acquireTimeout is how long Acquire waits for an image, in nanoseconds. The present
// modes that are used never block for long (SPEC §8.11), so a wait this long means the
// window is not being shown.
const acquireTimeout = 1_000_000_000

// swapchain is the images of one window (SPEC §9.4): B8G8R8A8_UNORM or R8G8B8A8_UNORM,
// opaque, MAILBOX, else IMMEDIATE, else FIFO, with one image more than the driver's
// minimum.
type swapchain struct {
	d      *Device
	surf   *surface
	handle swapchainH
	format explicit.TextureFormat
	width  int
	height int

	images     []*texture   // the images of the swapchain, which it owns
	renderSems []semaphoreH // signaled when the frame that drew into the image is done, one per image
	acquireSem [framesInFlight]semaphoreH
	semIdx     int // the acquire semaphore to use next

	index     uint32 // the image that was acquired, valid while acquired
	acquired  bool
	recreate  bool // out of date or suboptimal: the next Configure makes a new swapchain
	destroyed bool
}

var _ explicit.Swapchain = (*swapchain)(nil)

// acquiredImage is a swapchain image that the frame draws into: its commands wait for
// the acquire semaphore, end with the image in the present layout, and signal the
// semaphore that the present waits for.
type acquiredImage struct {
	sc   *swapchain
	idx  uint32
	wait semaphoreH
}

// NewSwapchain implements explicit.Device: it makes the surface of the window and the
// swapchain object; the images come with Configure. It fails with VulkanSurfaceFailed
// when the window cannot be drawn into with Vulkan.
func (d *Device) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	if !d.windows || d.f.createSwapchain == nil {
		return nil, errors.New(tagVulkanSurfaceFailed, "this Vulkan device was opened without a window: it lacks VK_KHR_swapchain")
	}
	s, err := d.surfaceFor(h)
	if err != nil {
		return nil, err
	}
	sc := &swapchain{d: d, surf: s}
	d.live[sc] = func() { sc.destroyed = true; sc.release(false) }
	return sc, nil
}

// Configure implements explicit.Swapchain: it makes the swapchain, or a new one when the
// size changed or the last one was out of date or suboptimal. The size is in device
// pixels; a surface that decides its own size (X11) has the last word, and the images
// then have that size. It fails with VulkanSwapchainFailed.
func (sc *swapchain) Configure(width, height int) error {
	if sc.handle != 0 && !sc.recreate && width == sc.width && height == sc.height {
		return nil
	}
	d := sc.d
	f := d.f
	a := d.arena
	defer a.rewind(a.mark())

	capsPtr := a.alloc(unsafeSizeof[surfaceCapabilities](), 8)
	if r := f.getPhysicalDeviceSurfaceCaps(d.phys, sc.surf.handle, capsPtr); r != resultSuccess {
		return swapchainError("vkGetPhysicalDeviceSurfaceCapabilitiesKHR", r)
	}
	caps := *(*surfaceCapabilities)(dl.Pointer(capsPtr))
	formats, err := sc.formats(a)
	if err != nil {
		return err
	}
	modes, err := sc.presentModes(a)
	if err != nil {
		return err
	}
	vkFormat, texFormat, ok := chooseSurfaceFormat(formats)
	if !ok {
		return errors.New(tagVulkanSwapchainFailed, "the surface has no B8G8R8A8_UNORM or R8G8B8A8_UNORM format", "formats", len(formats))
	}
	extent := extent2D{uint32(width), uint32(height)}
	if caps.currentExtent.width != currentExtentUndefined {
		extent = caps.currentExtent
	} else {
		extent.width = min(max(extent.width, caps.minImageExtent.width), caps.maxImageExtent.width)
		extent.height = min(max(extent.height, caps.minImageExtent.height), caps.maxImageExtent.height)
	}
	if extent.width == 0 || extent.height == 0 {
		return errors.New(tagVulkanSwapchainFailed, "the surface has no size", "width", int(extent.width), "height", int(extent.height))
	}
	const usage = imageUsageColor | imageUsageTransferDst
	if caps.supportedUsageFlags&usage != usage {
		return errors.New(tagVulkanSwapchainFailed, "the surface does not allow color attachment and transfer destination images", "usage", int(caps.supportedUsageFlags))
	}
	count := caps.minImageCount + 1
	if caps.maxImageCount > 0 {
		count = min(count, caps.maxImageCount)
	}
	alpha := uint32(compositeAlphaOpaque)
	if caps.supportedCompositeAlpha&alpha == 0 {
		alpha = caps.supportedCompositeAlpha & -caps.supportedCompositeAlpha // the lowest bit that is set
	}
	transform := uint32(surfaceTransformIdentity)
	if caps.supportedTransforms&transform == 0 {
		transform = caps.currentTransform
	}
	info := put(a, swapchainCreateInfo{
		sType: sTypeSwapchainCreateInfo, surface: sc.surf.handle, minImageCount: count,
		imageFormat: vkFormat, imageColorSpace: colorSpaceSRGBNonlinear, imageExtent: extent, imageArrayLayers: 1,
		imageUsage: usage, imageSharingMode: sharingExclusive, preTransform: transform, compositeAlpha: alpha,
		presentMode: choosePresentMode(modes), clipped: 1, oldSwapchain: sc.handle,
	})
	out := a.out()
	if r := f.createSwapchain(d.dev, info, 0, out); r != resultSuccess {
		return swapchainError("vkCreateSwapchainKHR", r)
	}
	old := sc.takeObjects()
	sc.handle = u64(out)
	sc.format, sc.width, sc.height = texFormat, int(extent.width), int(extent.height)
	sc.recreate, sc.acquired = false, false
	// A present that is still waiting for a semaphore is not a frame whose fence we can
	// wait for, so the old swapchain goes when the device is idle: a resize is rare.
	_ = d.f.deviceWaitIdle(d.dev)
	old()
	return sc.makeImages(a, vkFormat)
}

// formats lists the formats of the surface.
func (sc *swapchain) formats(a *arena) ([]surfaceFormat, error) {
	f := sc.d.f
	count := a.out()
	if r := f.getPhysicalDeviceSurfaceFormats(sc.d.phys, sc.surf.handle, count, 0); r != resultSuccess {
		return nil, swapchainError("vkGetPhysicalDeviceSurfaceFormatsKHR", r)
	}
	n := int(u32(count))
	p := a.alloc(uintptr(n)*unsafeSizeof[surfaceFormat](), 8)
	if r := f.getPhysicalDeviceSurfaceFormats(sc.d.phys, sc.surf.handle, count, p); r != resultSuccess && r != resultIncomplete {
		return nil, swapchainError("vkGetPhysicalDeviceSurfaceFormatsKHR", r)
	}
	return readSlice[surfaceFormat](p, int(u32(count))), nil
}

// presentModes lists the present modes of the surface.
func (sc *swapchain) presentModes(a *arena) ([]uint32, error) {
	f := sc.d.f
	count := a.out()
	if r := f.getPhysicalDeviceSurfaceModes(sc.d.phys, sc.surf.handle, count, 0); r != resultSuccess {
		return nil, swapchainError("vkGetPhysicalDeviceSurfacePresentModesKHR", r)
	}
	n := int(u32(count))
	p := a.alloc(uintptr(n)*4, 4)
	if r := f.getPhysicalDeviceSurfaceModes(sc.d.phys, sc.surf.handle, count, p); r != resultSuccess && r != resultIncomplete {
		return nil, swapchainError("vkGetPhysicalDeviceSurfacePresentModesKHR", r)
	}
	return readSlice[uint32](p, int(u32(count))), nil
}

// chooseSurfaceFormat picks B8G8R8A8_UNORM, else R8G8B8A8_UNORM, in the sRGB color space
// and never an SRGB format (blending is done in non-linear sRGB, SPEC §9.1, §9.4).
func chooseSurfaceFormat(formats []surfaceFormat) (vk uint32, tex explicit.TextureFormat, ok bool) {
	for _, want := range []struct {
		vk  uint32
		tex explicit.TextureFormat
	}{{formatBGRA8Unorm, explicit.TextureFormatBGRA8Unorm}, {formatRGBA8Unorm, explicit.TextureFormatRGBA8Unorm}} {
		for _, f := range formats {
			if f.format == want.vk && f.colorSpace == colorSpaceSRGBNonlinear {
				return want.vk, want.tex, true
			}
		}
	}
	return 0, 0, false
}

// choosePresentMode picks MAILBOX, else IMMEDIATE, else FIFO: the first two never
// block, so frames are paced by the compositor's frame callbacks (SPEC §8.11).
func choosePresentMode(modes []uint32) uint32 {
	for _, want := range []uint32{presentModeMailbox, presentModeImmediate} {
		for _, m := range modes {
			if m == want {
				return want
			}
		}
	}
	return presentModeFIFO
}

// swapchainError is vkError with the tag of the swapchain for the results that mean the
// window cannot be drawn into.
func swapchainError(call string, result int32) error {
	switch result {
	case resultErrorSurfaceLost, resultErrorWindowInUse, resultErrorInitFailed:
		return errors.New(tagVulkanSwapchainFailed, "the swapchain of the window cannot be made or used", "call", call, "result", int(result), "name", resultNames[result])
	}
	return vkError(call, result)
}

// makeImages wraps the images of the new swapchain and makes their views and the
// semaphores.
func (sc *swapchain) makeImages(a *arena, vkFormat uint32) error {
	d := sc.d
	count := a.out()
	if r := d.f.getSwapchainImages(d.dev, sc.handle, count, 0); r != resultSuccess {
		return swapchainError("vkGetSwapchainImagesKHR", r)
	}
	n := int(u32(count))
	p := a.alloc(uintptr(n)*8, 8)
	if r := d.f.getSwapchainImages(d.dev, sc.handle, count, p); r != resultSuccess && r != resultIncomplete {
		return swapchainError("vkGetSwapchainImagesKHR", r)
	}
	for _, img := range readSlice[imageH](p, int(u32(count))) {
		view, err := d.newView(img, vkFormat, 1)
		if err != nil {
			return err
		}
		sc.images = append(sc.images, &texture{
			d: d, image: img, view: view, external: true,
			desc: explicit.TextureDesc{Width: sc.width, Height: sc.height, Format: sc.format, MipLevels: 1,
				Usage: explicit.TextureRenderTarget | explicit.TextureCopySource},
		})
		sem, err := d.newSemaphore()
		if err != nil {
			return err
		}
		sc.renderSems = append(sc.renderSems, sem)
	}
	for i := range sc.acquireSem {
		sem, err := d.newSemaphore()
		if err != nil {
			return err
		}
		sc.acquireSem[i] = sem
	}
	return nil
}

// newSemaphore makes a binary semaphore.
func (d *Device) newSemaphore() (semaphoreH, error) {
	a := d.arena
	defer a.rewind(a.mark())
	out := a.out()
	info := put(a, semaphoreCreateInfo{sType: sTypeSemaphoreCreateInfo})
	if err := check("vkCreateSemaphore", d.f.createSemaphore(d.dev, info, 0, out)); err != nil {
		return 0, err
	}
	return u64(out), nil
}

// takeObjects hands the swapchain, the views and the semaphores over as a function
// that destroys them, and forgets them: the GPU may still use them.
func (sc *swapchain) takeObjects() func() {
	d := sc.d
	handle := sc.handle
	var views []imageViewH
	for _, t := range sc.images {
		views = append(views, t.view)
	}
	sems := append(append([]semaphoreH(nil), sc.renderSems...), sc.acquireSem[:]...)
	sc.images, sc.renderSems = nil, nil
	sc.acquireSem = [framesInFlight]semaphoreH{}
	return func() {
		// The views, then the swapchain, then the semaphores: the validation layer counts a
		// semaphore that a present waited for as in use until the swapchain is gone.
		for _, v := range views {
			d.f.destroyImageView(d.dev, v, 0)
		}
		if handle != 0 {
			d.f.destroySwapchain(d.dev, handle, 0)
		}
		for _, s := range sems {
			if s != 0 {
				d.f.destroySemaphore(d.dev, s, 0)
			}
		}
	}
}

// Acquire implements explicit.Swapchain. It must be called after Begin: the image is
// moved to the layout the frame draws in, and the frame's submit waits for the image
// and ends with it ready to present. When the swapchain is out of date it returns the
// error that explicit.IsSwapchainOutOfDate recognizes: Configure again and retry once.
func (sc *swapchain) Acquire() (explicit.Texture, error) {
	d := sc.d
	if d.open == nil {
		return nil, errors.New(tagVulkanFrameNotOpen, "Acquire needs the frame that draws into the image: call Begin first")
	}
	if sc.handle == 0 || sc.recreate {
		return nil, errors.New(explicit.SwapchainOutOfDate, "the swapchain no longer matches the surface: configure it again",
			"call", "vkAcquireNextImageKHR", "result", int(resultErrorOutOfDate))
	}
	if sc.acquired {
		return nil, errors.New(tagVulkanSwapchainFailed, "the image acquired before was not presented")
	}
	a := d.arena
	defer a.rewind(a.mark())
	sem := sc.acquireSem[sc.semIdx]
	out := a.out()
	switch r := d.f.acquireNextImage(d.dev, sc.handle, acquireTimeout, sem, 0, out); r {
	case resultSuccess:
	case resultSuboptimal:
		sc.recreate = true // the image is usable; the next frame makes a new swapchain
	case resultErrorOutOfDate:
		sc.recreate = true
		return nil, errors.New(explicit.SwapchainOutOfDate, "the swapchain no longer matches the surface: configure it again",
			"call", "vkAcquireNextImageKHR", "result", int(r))
	case resultTimeout, resultNotReady:
		return nil, errors.New(tagVulkanSwapchainFailed, "no swapchain image came: the window is not being shown", "result", int(r))
	default:
		return nil, swapchainError("vkAcquireNextImageKHR", r)
	}
	sc.index = u32(out)
	sc.acquired = true
	sc.semIdx = (sc.semIdx + 1) % framesInFlight
	img := sc.images[sc.index]
	d.open.acquired = append(d.open.acquired, acquiredImage{sc: sc, idx: sc.index, wait: sem})
	d.imageBarrier(d.open.cb, img.image, layoutUndefined, layoutGeneral)
	return img, nil
}

// Present implements explicit.Swapchain: it shows the image that was acquired, after the
// frame that drew into it, and tells the compositor which rectangles changed (from the
// top left; nil means all) when the device has VK_KHR_incremental_present. A swapchain
// that is out of date or suboptimal is made again by the next Configure.
func (sc *swapchain) Present(damage []image.Rectangle) error {
	d := sc.d
	if !sc.acquired {
		return errors.New(tagVulkanSwapchainFailed, "Present was called without an image acquired")
	}
	sc.acquired = false
	a := d.arena
	defer a.rewind(a.mark())
	pi := presentInfo{
		sType: sTypePresentInfo, waitSemaphoreCount: 1, pWaitSemaphores: putSlice(a, []semaphoreH{sc.renderSems[sc.index]}),
		swapchainCount: 1, pSwapchains: putSlice(a, []swapchainH{sc.handle}), pImageIndices: put(a, sc.index),
	}
	if d.incremental && len(damage) > 0 {
		rects := make([]rectLayer, 0, len(damage))
		for _, r := range damage {
			r = r.Intersect(image.Rect(0, 0, sc.width, sc.height))
			if r.Empty() {
				continue
			}
			rects = append(rects, rectLayer{offset: offset2D{int32(r.Min.X), int32(r.Min.Y)}, extent: extent2D{uint32(r.Dx()), uint32(r.Dy())}})
		}
		if len(rects) > 0 {
			region := put(a, presentRegion{rectangleCount: uint32(len(rects)), pRectangles: putSlice(a, rects)})
			pi.pNext = put(a, presentRegions{sType: sTypePresentRegions, swapchainCount: 1, pRegions: region})
		}
	}
	switch r := d.f.queuePresent(d.queue, put(a, pi)); r {
	case resultSuccess:
		return nil
	case resultSuboptimal, resultErrorOutOfDate:
		sc.recreate = true
		return nil
	default:
		return swapchainError("vkQueuePresentKHR", r)
	}
}

// Destroy implements explicit.Swapchain: the device waits until it is idle, then the
// swapchain, its images and semaphores, and the surface, are destroyed.
func (sc *swapchain) Destroy() {
	if sc.destroyed {
		return
	}
	sc.destroyed = true
	delete(sc.d.live, sc)
	sc.release(true)
}

// release destroys what the swapchain made. When wait is set the device is idle first:
// the present of the last frame may still wait for a semaphore, which a fence of the
// frame does not say.
func (sc *swapchain) release(wait bool) {
	d := sc.d
	destroy := sc.takeObjects()
	surf := sc.surf
	sc.handle = 0
	if wait {
		_ = d.f.deviceWaitIdle(d.dev)
	}
	destroy()
	d.destroySurface(surf)
}

// imageBarrier moves the image between layouts; every stage and access, as the other
// barriers are.
func (d *Device) imageBarrier(cb commandBufferH, img imageH, from, to uint32) {
	a := d.arena
	defer a.rewind(a.mark())
	barrier := put(a, imageMemoryBarrier2{
		sType: sTypeImageMemoryBarrier2, srcStageMask: stage2AllCommands, srcAccessMask: access2MemoryWrite,
		dstStageMask: stage2AllCommands, dstAccessMask: access2MemoryRead | access2MemoryWrite,
		oldLayout: from, newLayout: to,
		srcQueueFamilyIndex: queueFamilyIgnored, dstQueueFamilyIndex: queueFamilyIgnored,
		image:            img,
		subresourceRange: imageSubresourceRange{aspectMask: imageAspectColor, levelCount: 1, layerCount: 1},
	})
	dep := put(a, dependencyInfo{sType: sTypeDependencyInfo, imageMemoryBarrierCount: 1, pImageMemoryBarriers: barrier})
	d.f.cmdPipelineBarrier2(cb, dep)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagVulkanSwapchainFailed = "VulkanSwapchainFailed"
)
