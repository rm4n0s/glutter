//go:build linux

package vulkan

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// instance is a VkInstance with the debug messenger of the validation layer, when
// that is on (SPEC §9.4).
type instance struct {
	f       *vkFuncs
	handle  instanceH
	version uint32 // the API version the instance was made with
	log     *slog.Logger

	messenger debugMessengerH
	targetID  uintptr // the user data of the messenger, which finds this instance
	mu        sync.Mutex
	errors    []string // the validation errors seen
}

// instanceOptions are what an instance is made with.
type instanceOptions struct {
	// Validation turns on VK_LAYER_KHRONOS_validation and the debug messenger.
	Validation bool
	// Extensions are instance extensions that must be present and are enabled (the
	// surface extensions of the window system).
	Extensions []string
	// Logger gets the validation messages.
	Logger *slog.Logger
}

// debugTargets maps the user data of a debug messenger to its instance. A C callback
// has no Go closure, so there is one callback for the process, created once, and it
// finds the instance by this number (SPEC §8.1 rule 2).
var debugTargets = struct {
	sync.Mutex
	next uintptr
	m    map[uintptr]*instance
}{m: map[uintptr]*instance{}}

var (
	debugCallbackOnce sync.Once
	debugCallbackAddr uintptr
)

// debugCallback returns the C function of the messenger, made once.
func debugCallback() uintptr {
	debugCallbackOnce.Do(func() { debugCallbackAddr = dl.Callback(onDebugMessage) })
	return debugCallbackAddr
}

// onDebugMessage is the callback of VK_EXT_debug_utils. It runs on the thread that
// made the Vulkan call, inside it.
func onDebugMessage(severity, _ uint32, data, user uintptr) uint32 {
	debugTargets.Lock()
	inst := debugTargets.m[user]
	debugTargets.Unlock()
	if inst == nil || data == 0 {
		return 0
	}
	cb := (*debugUtilsMessengerCallbackData)(dl.Pointer(data))
	msg := dl.GoString(cb.pMessage)
	switch {
	case severity&debugSeverityError != 0:
		inst.mu.Lock()
		inst.errors = append(inst.errors, msg)
		inst.mu.Unlock()
		inst.log.Error("Vulkan validation", "message", msg)
	case severity&debugSeverityWarning != 0:
		inst.log.Warn("Vulkan validation", "message", msg)
	}
	return 0 // VK_FALSE: do not abort the call
}

// validationErrors returns the validation errors seen so far.
func (i *instance) validationErrors() []string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]string(nil), i.errors...)
}

// names returns the names of an array of extension or layer properties.
func cName(b []byte) string {
	if n := strings.IndexByte(string(b), 0); n >= 0 {
		return string(b[:n])
	}
	return string(b)
}

// instanceExtensions lists the instance extensions that the loader and the layers
// offer.
func instanceExtensions(f *vkFuncs, a *arena) ([]string, error) {
	defer a.rewind(a.mark())
	count := a.out()
	if err := check("vkEnumerateInstanceExtensionProperties", f.enumerateInstanceExtensionProps(0, count, 0)); err != nil {
		return nil, err
	}
	n := int(u32(count))
	props := a.alloc(uintptr(n)*sizeofExtensionProperties, 8)
	if err := check("vkEnumerateInstanceExtensionProperties", f.enumerateInstanceExtensionProps(0, count, props)); err != nil && n > 0 {
		return nil, err
	}
	var names []string
	for _, p := range readSlice[extensionProperties](props, int(u32(count))) {
		names = append(names, cName(p.extensionName[:]))
	}
	return names, nil
}

// hasLayer says whether the loader has the layer.
func hasLayer(f *vkFuncs, a *arena, name string) (bool, error) {
	defer a.rewind(a.mark())
	count := a.out()
	if err := check("vkEnumerateInstanceLayerProperties", f.enumerateInstanceLayerProps(count, 0)); err != nil {
		return false, err
	}
	n := int(u32(count))
	props := a.alloc(uintptr(n)*sizeofLayerProperties, 8)
	if err := check("vkEnumerateInstanceLayerProperties", f.enumerateInstanceLayerProps(count, props)); err != nil && n > 0 {
		return false, err
	}
	for _, p := range readSlice[layerProperties](props, int(u32(count))) {
		if cName(p.layerName[:]) == name {
			return true, nil
		}
	}
	return false, nil
}

// newInstance makes the VkInstance: Vulkan 1.1 or later is needed (SPEC §9.4), the
// version it asks for is the loader's, at most 1.3. With validation it enables the
// layer and the debug messenger. It fails with VulkanUnsupported when the loader has
// only Vulkan 1.0, or lacks the validation layer or an extension asked for.
func newInstance(f *vkFuncs, o instanceOptions) (*instance, error) {
	a := &arena{}
	defer a.free()
	version := uint32(0x00400000) // Vulkan 1.0, when the loader has no vkEnumerateInstanceVersion
	if f.enumerateInstanceVersion != nil {
		out := a.out()
		if err := check("vkEnumerateInstanceVersion", f.enumerateInstanceVersion(out)); err != nil {
			return nil, err
		}
		version = u32(out)
	}
	if version < apiVersion11 {
		return nil, errors.New("VulkanUnsupported", "the Vulkan loader supports Vulkan 1.0 only; Glutter needs 1.1 or later",
			"version", int(version))
	}
	version = min(version, apiVersion13)

	extensions := append([]string(nil), o.Extensions...)
	var layers []string
	available, err := instanceExtensions(f, a)
	if err != nil {
		return nil, err
	}
	if o.Validation {
		ok, err := hasLayer(f, a, validationLayerName)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("VulkanUnsupported", "the validation layer is not installed (sudo dnf install vulkan-validation-layers)",
				"layer", validationLayerName)
		}
		layers = append(layers, validationLayerName)
		extensions = append(extensions, debugUtilsExtension)
	}
	for _, e := range extensions {
		if !contains(available, e) && (e != debugUtilsExtension || !o.Validation) {
			return nil, errors.New("VulkanUnsupported", "the Vulkan instance lacks an extension that Glutter needs", "extension", e)
		}
	}

	app := put(a, applicationInfo{sType: sTypeApplicationInfo, pApplicationName: a.cstring("Glutter"), pEngineName: a.cstring("Glutter"), apiVersion: version})
	info := put(a, instanceCreateInfo{
		sType: sTypeInstanceCreateInfo, pApplicationInfo: app,
		enabledLayerCount: uint32(len(layers)), ppEnabledLayerNames: a.cstrings(layers),
		enabledExtensionCount: uint32(len(extensions)), ppEnabledExtensionNames: a.cstrings(extensions),
	})
	out := a.out()
	if err := check("vkCreateInstance", f.createInstance(info, 0, out)); err != nil {
		return nil, err
	}
	log := o.Logger
	if log == nil {
		log = slog.Default()
	}
	inst := &instance{f: f, handle: ptr(out), version: version, log: log}
	if err := f.bindInstance(inst.handle); err != nil {
		f.destroyInstance(inst.handle, 0)
		return nil, err
	}
	if o.Validation && f.createDebugUtilsMessenger != nil {
		if err := inst.createMessenger(a); err != nil {
			f.destroyInstance(inst.handle, 0)
			return nil, err
		}
	}
	return inst, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// createMessenger registers the debug messenger for warnings and errors.
func (i *instance) createMessenger(a *arena) error {
	debugTargets.Lock()
	debugTargets.next++
	i.targetID = debugTargets.next
	debugTargets.m[i.targetID] = i
	debugTargets.Unlock()
	info := put(a, debugUtilsMessengerCreateInfo{
		sType:           sTypeDebugUtilsMessengerCreateInfo,
		messageSeverity: debugSeverityWarning | debugSeverityError,
		messageType:     debugTypeGeneral | debugTypeValidation | debugTypePerformance,
		pfnUserCallback: debugCallback(), pUserData: i.targetID,
	})
	out := a.out()
	if err := check("vkCreateDebugUtilsMessengerEXT", i.f.createDebugUtilsMessenger(i.handle, info, 0, out)); err != nil {
		debugTargets.Lock()
		delete(debugTargets.m, i.targetID)
		debugTargets.Unlock()
		return err
	}
	i.messenger = u64(out)
	return nil
}

// destroy destroys the messenger and the instance.
func (i *instance) destroy() {
	if i.handle == 0 {
		return
	}
	if i.messenger != 0 {
		i.f.destroyDebugUtilsMessenger(i.handle, i.messenger, 0)
		debugTargets.Lock()
		delete(debugTargets.m, i.targetID)
		debugTargets.Unlock()
		i.messenger = 0
	}
	i.f.destroyInstance(i.handle, 0)
	i.handle = 0
}
