//go:build linux

package vulkan

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Options are what Open takes.
//
// Flutter: no counterpart
type Options struct {
	// Logger gets the log of the backend, and the messages of the validation layer;
	// slog.Default() when nil.
	Logger *slog.Logger
	// Getenv reads the environment (GLUTTER_VULKAN_LIBRARY, GLUTTER_VULKAN_DEVICE,
	// GLUTTER_VULKAN_VALIDATION); os.Getenv when nil.
	Getenv func(string) string
	// AllowCPU lets a device of type CPU (lavapipe) be used: the engine sets it when
	// Vulkan is asked for by name (SPEC §9.4).
	AllowCPU bool
	// Validation turns on the validation layer, as GLUTTER_VULKAN_VALIDATION=1 does.
	Validation bool
	// InstanceExtensions are the instance extensions to enable, such as the surface
	// extensions of the window system (step 20.2).
	InstanceExtensions []string
}

// Device is a Vulkan device and its queue: an explicit.Device. It is used on the UI
// thread only.
//
// Flutter: no counterpart
type Device struct {
	inst   *instance
	f      *vkFuncs
	log    *slog.Logger
	phys   physicalDeviceH
	dev    deviceH
	queue  queueH
	family uint32
	info   physicalInfo
	limits explicit.Limits
	alloc  *allocator
	arena  *arena
	closed bool

	slots   [framesInFlight]frameSlot
	next    int      // the slot of the next frame
	last    int      // the slot most recently submitted, -1 before the first
	open    *encoder // the frame being recorded
	pending []func(cb commandBufferH)
	staging stagingRing
	modules map[*explicit.Shader]shaderModuleH
	live    map[any]func() // what to destroy at Close: every resource that is not destroyed yet
}

var _ explicit.Device = (*Device)(nil)

// memory is the memoryBackend of the allocator, over the device.
type memory struct{ d *Device }

func (m memory) allocate(size uint64, typeIndex uint32) (deviceMemoryH, error) {
	a := m.d.arena
	defer a.rewind(a.mark())
	info := put(a, memoryAllocateInfo{sType: sTypeMemoryAllocateInfo, allocationSize: size, memoryTypeIndex: typeIndex})
	out := a.out()
	if err := check("vkAllocateMemory", m.d.f.allocateMemory(m.d.dev, info, 0, out)); err != nil {
		return 0, err
	}
	return u64(out), nil
}

func (m memory) mapAll(mem deviceMemoryH, size uint64) (uintptr, error) {
	a := m.d.arena
	defer a.rewind(a.mark())
	out := a.out()
	if err := check("vkMapMemory", m.d.f.mapMemory(m.d.dev, mem, 0, size, 0, out)); err != nil {
		return 0, err
	}
	return ptr(out), nil
}

func (m memory) free(mem deviceMemoryH) { m.d.f.freeMemory(m.d.dev, mem, 0) }

// Open loads Vulkan, makes the instance, chooses the physical device and makes the
// logical device and its queue (SPEC §9.4). It fails with:
//
//   - VulkanLoadFailed when libvulkan.so.1 cannot be loaded
//   - VulkanUnsupported when the loader, or every device, lacks the version, the
//     extensions or the features that Glutter needs
//   - VulkanNoDevice when there is no device that can be used
//   - GPUOutOfMemory when the driver is out of memory
//
// Flutter: no counterpart
func Open(o Options) (*Device, error) {
	getenv := o.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	log := o.Logger
	if log == nil {
		log = slog.Default()
	}
	f, err := loadLibrary(getenv)
	if err != nil {
		return nil, err
	}
	inst, err := newInstance(f, instanceOptions{
		Validation: o.Validation || getenv(validationEnvVar) == "1",
		Extensions: o.InstanceExtensions, Logger: log,
	})
	if err != nil {
		return nil, err
	}
	d, err := openDevice(inst, chooseOpts{Selector: getenv(deviceEnvVar), AllowCPU: o.AllowCPU}, log)
	if err != nil {
		inst.destroy()
		return nil, err
	}
	return d, nil
}

// openDevice chooses a physical device and makes the logical one.
func openDevice(inst *instance, opts chooseOpts, log *slog.Logger) (*Device, error) {
	a := &arena{}
	defer a.free()
	handles, infos, err := inst.physicalDevices(a)
	if err != nil {
		return nil, err
	}
	index, family, err := chooseDevice(infos, opts)
	if err != nil {
		return nil, err
	}
	info := infos[index]
	f := inst.f
	d := &Device{inst: inst, f: f, log: log, phys: handles[index], family: uint32(family), info: info,
		limits: explicit.DefaultLimits(), arena: &arena{}, last: -1,
		modules: map[*explicit.Shader]shaderModuleH{}, live: map[any]func(){}}
	if err := d.createDevice(a); err != nil {
		return nil, err
	}
	out := a.out()
	f.getDeviceQueue(d.dev, d.family, 0, out)
	d.queue = ptr(out)

	var props physicalDeviceMemoryProperties
	mem := a.alloc(uintptr(sizeofMemoryProperties), 8)
	f.getPhysicalDeviceMemoryProps(d.phys, mem)
	props = *(*physicalDeviceMemoryProperties)(dl.Pointer(mem))
	d.alloc = newAllocator(memory{d}, newMemoryProps(props))

	if err := d.initFrames(); err != nil {
		d.Close() //nolint:errcheck // the error that matters is the one being returned
		return nil, err
	}
	log.Info("Vulkan device", "device", d.Description(), "api", versionString(info.APIVersion), "queue_family", family)
	return d, nil
}

// physicalDevices lists the physical devices with what the choice needs to know.
func (i *instance) physicalDevices(a *arena) ([]physicalDeviceH, []physicalInfo, error) {
	f := i.f
	count := a.out()
	if err := check("vkEnumeratePhysicalDevices", f.enumeratePhysicalDevices(i.handle, count, 0)); err != nil {
		return nil, nil, err
	}
	n := int(u32(count))
	list := a.alloc(uintptr(n)*8, 8)
	if r := f.enumeratePhysicalDevices(i.handle, count, list); r != resultSuccess && r != resultIncomplete {
		return nil, nil, vkError("vkEnumeratePhysicalDevices", r)
	}
	handles := readSlice[physicalDeviceH](list, int(u32(count)))
	infos := make([]physicalInfo, len(handles))
	for idx, h := range handles {
		infos[idx], _ = i.describe(a, idx, h)
	}
	return handles, infos, nil
}

// describe asks a physical device for its properties, queues, extensions and the two
// features. A device that cannot be asked gets the zero info for what failed, which
// the choice rejects.
func (i *instance) describe(a *arena, index int, h physicalDeviceH) (physicalInfo, error) {
	defer a.rewind(a.mark())
	f := i.f
	pp := a.alloc(unsafeSizeof[physicalDeviceProperties](), 8)
	f.getPhysicalDeviceProperties(h, pp)
	props := *(*physicalDeviceProperties)(dl.Pointer(pp))
	info := physicalInfo{
		Index: index, Name: cName(props.deviceName[:]), Type: props.deviceType,
		APIVersion:          min(props.apiVersion, i.version),
		MaxImageDimension2D: props.limits.maxImageDimension2D,
		Extensions:          map[string]bool{},
	}

	count := a.out()
	f.getPhysicalDeviceQueueFamilyProps(h, count, 0)
	qn := int(u32(count))
	qp := a.alloc(uintptr(qn)*unsafeSizeof[queueFamilyProperties](), 8)
	f.getPhysicalDeviceQueueFamilyProps(h, count, qp)
	for _, q := range readSlice[queueFamilyProperties](qp, int(u32(count))) {
		info.QueueFlags = append(info.QueueFlags, q.queueFlags)
	}

	if r := f.enumerateDeviceExtensionProps(h, 0, count, 0); r != resultSuccess {
		return info, vkError("vkEnumerateDeviceExtensionProperties", r)
	}
	en := int(u32(count))
	ep := a.alloc(uintptr(en)*sizeofExtensionProperties, 8)
	if r := f.enumerateDeviceExtensionProps(h, 0, count, ep); r != resultSuccess && r != resultIncomplete {
		return info, vkError("vkEnumerateDeviceExtensionProperties", r)
	}
	for _, e := range readSlice[extensionProperties](ep, int(u32(count))) {
		info.Extensions[cName(e.extensionName[:])] = true
	}

	// The feature structs may be chained only when the version or the extension has
	// them.
	hasDR := info.APIVersion >= apiVersion13 || info.Extensions[extDynamicRendering]
	hasS2 := info.APIVersion >= apiVersion13 || info.Extensions[extSynchronization2]
	var dr, s2 uintptr
	var chain uintptr
	if hasS2 {
		s2 = put(a, physicalDeviceSynchronization2Features{sType: sTypePhysicalDeviceSynchronization2, pNext: chain})
		chain = s2
	}
	if hasDR {
		dr = put(a, physicalDeviceDynamicRenderingFeatures{sType: sTypePhysicalDeviceDynamicRendering, pNext: chain})
		chain = dr
	}
	f2 := put(a, physicalDeviceFeatures2{sType: sTypePhysicalDeviceFeatures2, pNext: chain})
	f.getPhysicalDeviceFeatures2(h, f2)
	if hasDR {
		info.DynamicRendering = (*physicalDeviceDynamicRenderingFeatures)(dl.Pointer(dr)).dynamicRendering != 0
	}
	if hasS2 {
		info.Synchronization2 = (*physicalDeviceSynchronization2Features)(dl.Pointer(s2)).synchronization2 != 0
	}
	return info, nil
}

// createDevice makes the VkDevice with one queue, dynamic rendering and
// synchronization 2. Below Vulkan 1.3 the two come as extensions, with what
// VK_KHR_dynamic_rendering depends on.
func (d *Device) createDevice(a *arena) error {
	var extensions []string
	if d.info.APIVersion < apiVersion13 {
		extensions = append(extensions, extDynamicRendering, extSynchronization2)
		if d.info.APIVersion < 0x00402000 { // before Vulkan 1.2 the dependencies are extensions too
			for _, dep := range extensionDependencies {
				if d.info.Extensions[dep] {
					extensions = append(extensions, dep)
				}
			}
		}
	}
	s2 := put(a, physicalDeviceSynchronization2Features{sType: sTypePhysicalDeviceSynchronization2, synchronization2: 1})
	dr := put(a, physicalDeviceDynamicRenderingFeatures{sType: sTypePhysicalDeviceDynamicRendering, pNext: s2, dynamicRendering: 1})
	priority := put(a, float32(1))
	queue := put(a, deviceQueueCreateInfo{sType: sTypeDeviceQueueCreateInfo, queueFamilyIndex: d.family, queueCount: 1, pQueuePriorities: priority})
	info := put(a, deviceCreateInfo{
		sType: sTypeDeviceCreateInfo, pNext: dr,
		queueCreateInfoCount: 1, pQueueCreateInfos: queue,
		enabledExtensionCount: uint32(len(extensions)), ppEnabledExtensionNames: a.cstrings(extensions),
	})
	out := a.out()
	if err := check("vkCreateDevice", d.f.createDevice(d.phys, info, 0, out)); err != nil {
		return err
	}
	d.dev = ptr(out)
	if err := d.f.bindDevice(d.dev); err != nil {
		d.f.destroyDevice(d.dev, 0)
		d.dev = 0
		return err
	}
	return nil
}

// Description names the backend and the device: "vulkan: Intel(R) Graphics (ADL GT2)".
//
// Flutter: no counterpart
func (d *Device) Description() string { return "vulkan: " + d.info.Name }

// DeviceName is the name of the physical device.
//
// Flutter: no counterpart
func (d *Device) DeviceName() string { return d.info.Name }

// ValidationErrors returns the errors that the validation layer reported so far, when
// validation is on.
//
// Flutter: no counterpart
func (d *Device) ValidationErrors() []string { return d.inst.validationErrors() }

// Limits implements explicit.Device.
func (d *Device) Limits() explicit.Limits { return d.limits }

// NewSwapchain implements explicit.Device. Windows come in step 20.2.
func (d *Device) NewSwapchain(gpu.SurfaceHandle) (explicit.Swapchain, error) {
	// TODO(step 20.2): swapchains from VK_KHR_wayland_surface and VK_KHR_xlib_surface.
	return nil, errors.New("VulkanSurfaceFailed", "this device draws offscreen only until step 20.2")
}

// Close waits for the GPU, destroys everything that was made with the device and the
// instance. It can be called twice.
//
// Flutter: no counterpart
func (d *Device) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	if d.dev != 0 {
		_ = d.f.deviceWaitIdle(d.dev)
		for i := range d.slots {
			d.slots[i].runDeferred()
		}
		for _, destroy := range d.live {
			destroy()
		}
		d.live = nil
		d.destroyFrames()
		for _, m := range d.modules {
			d.f.destroyShaderModule(d.dev, m, 0)
		}
		d.staging.destroy(d)
		d.alloc.close()
		d.f.destroyDevice(d.dev, 0)
		d.dev = 0
	}
	d.arena.free()
	d.inst.destroy()
	return nil
}

// versionString formats a Vulkan version as "1.3.0".
func versionString(v uint32) string {
	return fmt.Sprintf("%d.%d.%d", v>>22, (v>>12)&0x3FF, v&0xFFF)
}
