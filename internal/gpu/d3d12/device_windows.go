//go:build windows

package d3d12

import (
	"fmt"
	"log/slog"
	"os"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// framesInFlight is how many frames the GPU can work on at once (SPEC §9.3).
const framesInFlight = explicit.FramesInFlight

// featureLevels are the feature levels asked for in CheckFeatureSupport, to report the highest one
// the adapter has (SPEC §9.8); the device is made at 11_0 whatever it says. It is read-only: the
// request points into it.
var featureLevels = [...]uint32{featureLevel12_2, featureLevel12_1, featureLevel12_0, featureLevel11_1, featureLevel11_0}

// Options are what Open takes.
//
// Flutter: no counterpart
type Options struct {
	// Logger gets the log of the backend, and the messages of the debug layer; slog.Default() when nil.
	Logger *slog.Logger
	// Getenv reads the environment (GLUTTER_D3D12_ADAPTER, GLUTTER_D3D12_DEBUG); os.Getenv when nil.
	Getenv func(string) string
	// AllowWarp lets WARP, Windows' own software adapter, be the last candidate when the adapter is
	// not named: the engine sets it for `auto` (SPEC §9.6). An adapter asked for by name, "warp"
	// included, is used whatever this says.
	AllowWarp bool
	// Debug turns on the debug layer, as GLUTTER_D3D12_DEBUG=1 does.
	Debug bool
	// ShaderCache keeps the bytecode that D3DCompile makes, and the serialized pipeline library
	// (SPEC §9.7); *shadercache.Cache is one. nil is no cache.
	ShaderCache BlobCache

	// viewBlock and samplerBlock are the sizes of the blocks of the shader-visible heaps, in
	// descriptors; a test makes them small to see a block added. noLibrary makes no pipeline library, as
	// a driver without one does.
	viewBlock, samplerBlock int
	noLibrary               bool
	makeLibrary             func(blob []byte) (win32.Object, win32.HRESULT)
}

// adapterEntry is an adapter DXGI listed, with the object.
type adapterEntry struct {
	obj  win32.Object
	info AdapterInfo
}

// Device is a Direct3D 12 device and its queue: an explicit.Device. It is used on the UI thread
// only, and all calls of one device are made from one OS thread, as the tests (runtime.LockOSThread)
// and the platform do.
//
// Flutter: no counterpart
type Device struct {
	log    *slog.Logger
	getenv func(string) string
	libs   *libraries
	limits explicit.Limits

	factory, adapter, dev, queue, infoQueue win32.Object
	info                                    AdapterInfo
	adapters                                []AdapterInfo
	warp                                    bool
	debugLayer                              bool
	featureLevel, bindingTier               uint32
	opened                                  time.Duration
	closed, lost                            bool
	leaked                                  int

	// objects is the number of COM objects made and not yet released, for the test that Close leaves
	// nothing.
	objects int
	counts  counters

	fence      win32.Object
	fenceEvent uintptr
	fenceValue uint64
	// removed asks the device why it was removed (GetDeviceRemovedReason); a test replaces it to make a
	// device that is lost on demand.
	removed func() win32.HRESULT

	slots [framesInFlight]frameSlot
	next  int      // the slot of the next frame
	last  int      // the slot most recently submitted, -1 before the first
	open  *encoder // the frame being recorded
	// toPresent are the swap chain buffers that were acquired for the frame being recorded: Submit puts
	// them back in the state COMMON (PRESENT) before it closes the list.
	toPresent []*texture
	// swapFormats replaces the formats of the swap chain, for the test of a device that accepts none.
	swapFormats []swapFormat
	pending     []func(list win32.Object)
	staging     staging
	tracker     *stateTracker[*texture]
	live        map[any]func() // what to destroy at Close: every resource that is not destroyed yet

	cpu         [3]*cpuHeaps // render target views, shader resource views and samplers
	debugErrors []string

	// viewBlock and samplerBlock are the sizes of the blocks of the slots' shader-visible heaps, and
	// viewIncr and samplerIncr the sizes of their descriptors.
	viewBlock, samplerBlock int
	viewIncr, samplerIncr   uintptr

	// roots are the root signatures, by the shape of their bindings; the pipelines share them.
	roots map[string]*rootSig
	// cache keeps the bytecode and the pipeline library; compiler is made when the first pipeline needs
	// it, and dxbc remembers the bytecode of each entry point for the pipelines that share it.
	cache    BlobCache
	compiler *Compiler
	dxbc     map[dxbcKey][]byte
	library  pipelineLibrary
	pipes    pipelineStats
	noLib    bool
	// makeLibrary replaces ID3D12Device1::CreatePipelineLibrary; a test gives a library of its own,
	// since the one of the real device is not there under Wine.
	makeLibrary func(blob []byte) (win32.Object, win32.HRESULT)
}

var _ explicit.Device = (*Device)(nil)

// counters are the numbers Stats reports.
type counters struct {
	buffers, textures, samplers int
	bytes                       int64
	oneOff                      int
}

// Stats is the accounting of a device, for the tests that frames do not grow memory and that Close
// leaves nothing.
//
// Flutter: no counterpart
type Stats struct {
	// Objects is the number of COM objects the device made and has not released.
	Objects int
	// Buffers, Textures and Samplers are the resources that were made and not destroyed.
	Buffers, Textures, Samplers int
	// CommittedBytes is the size of the committed resources alive: buffers, textures (their mip levels
	// included), the staging ring and the readback buffers.
	CommittedBytes int64
	// OneOffUploads is the number of upload buffers of their own (for data larger than the staging
	// ring's half) that frames still hold.
	OneOffUploads int
	// Descriptors is the number of CPU descriptors in use.
	Descriptors int
	// StagingUsed is how many bytes of the staging ring each frame slot has taken.
	StagingUsed [framesInFlight]int
	// Pipelines are the pipelines that were made and not destroyed, and RootSignatures the root
	// signatures there are (one for each shape of bindings).
	Pipelines, RootSignatures int
	// ViewBlocks and SamplerBlocks are the blocks of shader-visible heaps the frame slots have made
	// (they are reused), and ViewDescriptors and SamplerDescriptors the descriptors taken in them now.
	ViewBlocks, SamplerBlocks, ViewDescriptors, SamplerDescriptors int
}

// Open loads Direct3D 12, chooses the adapter (SPEC §9.8) and makes the device at feature level 11_0,
// its queue and its frame slots. It fails with:
//
//   - D3D12LoadFailed when d3d12.dll or dxgi.dll is not in System32
//   - D3D12NoAdapter when there is no adapter to try: no hardware adapter, or none that
//     GLUTTER_D3D12_ADAPTER names
//   - D3D12Unsupported when the adapters there are cannot make a device at feature level 11_0
//   - GPUOutOfMemory and GPUDeviceLost, and D3D12CallFailed for any other failed HRESULT
//
// Flutter: no counterpart
func Open(o Options) (*Device, error) {
	return openWith(o, defaultEntryNames(), featureLevel11_0)
}

// openWith is Open with the names of the libraries and the feature level of the device, which a test
// changes to see the errors.
func openWith(o Options, names entryNames, level uint32) (*Device, error) {
	start := time.Now()
	getenv := o.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	log := o.Logger
	if log == nil {
		log = slog.Default()
	}
	libs, err := loadLibraries(names)
	if err != nil {
		return nil, err
	}
	d := &Device{
		log: log, getenv: getenv, libs: libs, limits: explicit.DefaultLimits(), last: -1,
		tracker: newStateTracker[*texture](), live: map[any]func(){}, staging: staging{ring: newStagingRing(stagingSize)},
		viewBlock: viewBlockSize, samplerBlock: samplerBlockSize, roots: map[string]*rootSig{}, dxbc: map[dxbcKey][]byte{},
		cache: o.ShaderCache, noLib: o.noLibrary, makeLibrary: o.makeLibrary,
	}
	if o.viewBlock > 0 {
		d.viewBlock = o.viewBlock
	}
	if o.samplerBlock > 0 {
		d.samplerBlock = o.samplerBlock
	}
	d.removed = d.deviceRemovedReason
	d.cpu = [3]*cpuHeaps{newCPUHeaps(descriptorHeapRTV), newCPUHeaps(descriptorHeapSRV), newCPUHeaps(descriptorHeapSampler)}
	if err := d.openDevice(o, level); err != nil {
		if cerr := d.Close(); cerr != nil {
			log.Error("closing the device that failed to open", "err", cerr)
		}
		return nil, err
	}
	d.opened = time.Since(start)
	d.report()
	return d, nil
}

// openDevice makes the factory, chooses the adapter and makes the device, the queue, the fence and
// the frame slots. A failure leaves what was made in d, and the caller closes it.
func (d *Device) openDevice(o Options, level uint32) error {
	if o.Debug || d.getenv(DebugEnvVar) == "1" {
		d.enableDebugLayer()
	}
	flags := uint32(0)
	if d.debugLayer {
		flags = dxgiCreateFactoryDebug
	}
	var factory uintptr
	r, _ := d.libs.createFactory.Call(uintptr(flags), uintptr(unsafe.Pointer(&iidDXGIFactory1)), uintptr(unsafe.Pointer(&factory)))
	if err := check("CreateDXGIFactory2", win32.FromResult(r)); err != nil {
		return err
	}
	d.factory = d.own(win32.ObjectFromPtr(factory))

	entries, err := d.enumerateAdapters()
	if err != nil {
		return err
	}
	defer func() {
		for _, e := range entries {
			d.disown(e.obj)
		}
	}()
	infos := make([]AdapterInfo, len(entries))
	for i, e := range entries {
		infos[i] = e.info
	}
	d.adapters = infos
	cands, err := chooseAdapters(infos, d.getenv(AdapterEnvVar), o.AllowWarp)
	if err != nil {
		return err
	}

	var tried []string
	reachedDevice := false
	for _, c := range cands {
		var adapter win32.Object
		var info AdapterInfo
		if c.Warp {
			var err error
			if adapter, info, err = d.warpAdapter(); err != nil {
				tried = append(tried, "warp: "+err.Error())
				continue
			}
		} else {
			adapter, info = d.own(entries[c.Index].obj), entries[c.Index].info
			adapter.AddRef()
		}
		var dev uintptr
		r, _ := d.libs.createDevice.Call(adapter.Ptr(), uintptr(level), uintptr(unsafe.Pointer(&iidDevice)), uintptr(unsafe.Pointer(&dev)))
		reachedDevice = true
		if hr := win32.FromResult(r); hr.Failed() {
			tried = append(tried, fmt.Sprintf("%s: D3D12CreateDevice failed with %s", info.Name, hr))
			d.disown(adapter)
			continue
		}
		d.adapter, d.info, d.warp = adapter, info, c.Warp
		d.dev = d.own(win32.ObjectFromPtr(dev))
		break
	}
	if d.dev.IsNil() {
		if !reachedDevice {
			return errors.New(TagD3D12NoAdapter, "no adapter could be opened", "tried", tried, "selector", d.getenv(AdapterEnvVar))
		}
		return errors.New(TagD3D12Unsupported, "no adapter can make a Direct3D 12 device at feature level 11_0",
			"feature_level", featureLevelName(level), "tried", tried)
	}
	d.queryFeatures()
	d.viewIncr = d.dev.Call(deviceGetDescriptorIncrementSize, uintptr(descriptorHeapSRV))
	d.samplerIncr = d.dev.Call(deviceGetDescriptorIncrementSize, uintptr(descriptorHeapSampler))
	if d.debugLayer {
		d.openInfoQueue()
	}
	if err := d.createQueue(); err != nil {
		return err
	}
	if err := d.createFence(); err != nil {
		return err
	}
	return d.createSlots()
}

// enableDebugLayer calls D3D12GetDebugInterface and EnableDebugLayer before the device is made. The
// layer is part of the optional Windows feature "Graphics Tools"; without it the call fails
// (DXGI_ERROR_SDK_COMPONENT_MISSING, also under Wine), which is logged once at Warn level and
// ignored.
func (d *Device) enableDebugLayer() {
	var dbg uintptr
	r, _ := d.libs.getDebugInterface.Call(uintptr(unsafe.Pointer(&iidDebug)), uintptr(unsafe.Pointer(&dbg)))
	if hr := win32.FromResult(r); hr.Failed() || dbg == 0 {
		d.log.Warn("the Direct3D debug layer is not installed; going on without it",
			"hresult", win32.FromResult(r).String(), "missing", win32.FromResult(r).Code() == dxgiErrorSDKComponentMissing)
		return
	}
	layer := win32.ObjectFromPtr(dbg)
	layer.Call(debugEnableDebugLayer)
	layer.Release()
	d.debugLayer = true
}

// enumerateAdapters lists the adapters of DXGI: with DXGI 1.6 in the order of
// EnumAdapterByGpuPreference (high performance first), else EnumAdapters1 in order (SPEC §9.8).
func (d *Device) enumerateAdapters() ([]adapterEntry, error) {
	f6, err := d.factory.QueryInterface(&iidDXGIFactory6)
	haveF6 := err == nil // an older DXGI lacks the interface: EnumAdapters1 then
	if haveF6 {
		d.own(f6)
		defer d.disown(f6)
	}
	var list []adapterEntry
	for i := 0; ; i++ {
		var out uintptr
		var r uintptr
		method := "IDXGIFactory1::EnumAdapters1"
		if haveF6 {
			method = "IDXGIFactory6::EnumAdapterByGpuPreference"
			r = f6.Call(factoryEnumAdapterByGPUPreference, uintptr(i), uintptr(dxgiGPUPreferenceHighPerf), uintptr(unsafe.Pointer(&iidDXGIAdapter1)), uintptr(unsafe.Pointer(&out)))
		} else {
			r = d.factory.Call(factoryEnumAdapters1, uintptr(i), uintptr(unsafe.Pointer(&out)))
		}
		hr := win32.FromResult(r)
		if hr.Code() == dxgiErrorNotFound {
			return list, nil
		}
		if err := check(method, hr); err != nil {
			for _, e := range list {
				d.disown(e.obj)
			}
			return nil, err
		}
		obj := d.own(win32.ObjectFromPtr(out))
		info, err := d.describeAdapter(obj, i)
		if err != nil {
			d.disown(obj)
			for _, e := range list {
				d.disown(e.obj)
			}
			return nil, err
		}
		list = append(list, adapterEntry{obj: obj, info: info})
	}
}

// describeAdapter reads the description of an adapter.
func (d *Device) describeAdapter(a win32.Object, index int) (AdapterInfo, error) {
	var desc dxgiAdapterDesc1
	if err := callHR(a, "IDXGIAdapter1::GetDesc1", adapterGetDesc1, uintptr(unsafe.Pointer(&desc))); err != nil {
		return AdapterInfo{}, err
	}
	return AdapterInfo{
		Index: index, Name: win32.UTF16ToString(desc.Description[:]),
		VendorID: desc.VendorID, DeviceID: desc.DeviceID, Revision: desc.Revision,
		Software:             desc.Flags&dxgiAdapterSoftware != 0,
		DedicatedVideoMemory: uint64(desc.DedicatedVideoMemory),
	}, nil
}

// warpAdapter is IDXGIFactory4::EnumWarpAdapter. It fails when DXGI lacks the interface, and under
// Wine, which has no WARP.
func (d *Device) warpAdapter() (win32.Object, AdapterInfo, error) {
	f4, err := d.factory.QueryInterface(&iidDXGIFactory4)
	if err != nil {
		return win32.Object{}, AdapterInfo{}, err
	}
	d.own(f4)
	defer d.disown(f4)
	var out uintptr
	if err := callHR(f4, "IDXGIFactory4::EnumWarpAdapter", factoryEnumWarpAdapter, uintptr(unsafe.Pointer(&iidDXGIAdapter1)), uintptr(unsafe.Pointer(&out))); err != nil {
		return win32.Object{}, AdapterInfo{}, err
	}
	obj := d.own(win32.ObjectFromPtr(out))
	info, err := d.describeAdapter(obj, -1)
	if err != nil {
		d.disown(obj)
		return win32.Object{}, AdapterInfo{}, err
	}
	info.Software = true
	return obj, info, nil
}

// queryFeatures reads the highest feature level of the adapter and the resource binding tier, for
// the report. The device exists at 11_0, so a failure of either question only leaves the report
// short.
func (d *Device) queryFeatures() {
	d.featureLevel = featureLevel11_0
	fl := d3d12FeatureLevels{NumFeatureLevels: uint32(len(featureLevels)), FeatureLevelsRequested: unsafe.Pointer(&featureLevels[0])}
	if hr := win32.FromResult(d.dev.Call(deviceCheckFeatureSupport, uintptr(featureFeatureLevels), uintptr(unsafe.Pointer(&fl)), unsafe.Sizeof(fl))); hr.Succeeded() {
		d.featureLevel = fl.MaxSupportedFeatureLevel
	} else {
		d.log.Debug("the highest feature level of the adapter is not known", "hresult", hr.String())
	}
	var opts d3d12Options
	if hr := win32.FromResult(d.dev.Call(deviceCheckFeatureSupport, uintptr(featureD3D12Options), uintptr(unsafe.Pointer(&opts)), unsafe.Sizeof(opts))); hr.Succeeded() {
		d.bindingTier = opts.ResourceBindingTier
		if d.bindingTier < resourceBindingTier1 {
			d.log.Warn("the adapter reports no resource binding tier; the device needs tier 1", "tier", d.bindingTier)
		}
	} else {
		d.log.Debug("the resource binding tier of the adapter is not known", "hresult", hr.String())
	}
}

// createQueue makes the direct command queue.
func (d *Device) createQueue() error {
	desc := d3d12CommandQueueDesc{Type: commandListDirect}
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateCommandQueue", deviceCreateCommandQueue,
		uintptr(unsafe.Pointer(&desc)), uintptr(unsafe.Pointer(&iidCommandQueue)), uintptr(unsafe.Pointer(&out))); err != nil {
		return err
	}
	d.queue = d.own(win32.ObjectFromPtr(out))
	return nil
}

// own counts a COM object that the device made, and returns it.
func (d *Device) own(o win32.Object) win32.Object {
	d.objects++
	return o
}

// disown releases a COM object that own counted.
func (d *Device) disown(o win32.Object) {
	if o.IsNil() {
		return
	}
	d.objects--
	o.Release()
}

// hr calls a method that returns an HRESULT and turns a failure into the tagged error (check). A lost
// device is told so with the reason in the metadata "reason".
//
//go:uintptrescapes
func (d *Device) hr(obj win32.Object, method string, index int, args ...uintptr) error {
	return d.annotate(callHR(obj, method, index, args...))
}

// callHR calls a method that returns an HRESULT, for the code that has no device yet.
//
//go:uintptrescapes
func callHR(obj win32.Object, method string, index int, args ...uintptr) error {
	return check(method, win32.FromResult(obj.Call(index, args...)))
}

// annotate adds the reason of the removal (GetDeviceRemovedReason) to the error of a device that was
// lost, and remembers the loss.
func (d *Device) annotate(err error) error {
	if err == nil {
		return nil
	}
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == tagGPUDeviceLost {
		d.lost = true
		_ = appErr.SetMetadata("reason", d.removed().String()) // adds to the error in place
	}
	return err
}

// deviceRemovedReason is ID3D12Device::GetDeviceRemovedReason.
func (d *Device) deviceRemovedReason() win32.HRESULT {
	if d.dev.IsNil() {
		return win32.SOK
	}
	return win32.FromResult(d.dev.Call(deviceGetDeviceRemovedReason))
}

// checkRemoved asks the device whether it was removed and fails with GPUDeviceLost, with the reason in
// the metadata, if so (SPEC §9.8 "Device removal"). The rule of §9.3 ("Decided in step 20.7") then
// recreates everything that belongs to the device.
func (d *Device) checkRemoved() error {
	reason := d.removed()
	if reason == win32.SOK {
		return nil
	}
	d.lost = true
	return errors.New(tagGPUDeviceLost, "the Direct3D 12 device was removed", "method", "ID3D12Device::GetDeviceRemovedReason", "hresult", reason.String(), "reason", reason.String())
}

// Description names the backend and the adapter: "d3d12: NVIDIA GeForce GTX 470 (10DE:1D01)", and
// "d3d12: warp (Microsoft Basic Render Driver)" for WARP (SPEC §9.8).
//
// Flutter: no counterpart
func (d *Device) Description() string {
	if d.warp {
		return "d3d12: warp (" + d.info.Name + ")"
	}
	return fmt.Sprintf("d3d12: %s (%04X:%04X)", d.info.Name, d.info.VendorID, d.info.DeviceID)
}

// Adapter is the adapter the device was made on; Index is -1 for WARP.
//
// Flutter: no counterpart
func (d *Device) Adapter() AdapterInfo { return d.info }

// Adapters are the adapters that DXGI listed when the device was made, in its order (WARP is not
// among them, unless the list has it).
//
// Flutter: no counterpart
func (d *Device) Adapters() []AdapterInfo { return append([]AdapterInfo(nil), d.adapters...) }

// FeatureLevel is the highest feature level the adapter has, such as "12_1" (the device is made at
// 11_0).
//
// Flutter: no counterpart
func (d *Device) FeatureLevel() string { return featureLevelName(d.featureLevel) }

// BindingTier is the resource binding tier of the adapter (1 to 3; 0 when it is not known).
//
// Flutter: no counterpart
func (d *Device) BindingTier() int { return int(d.bindingTier) }

// DebugLayer says whether the debug layer is on.
//
// Flutter: no counterpart
func (d *Device) DebugLayer() bool { return d.debugLayer }

// StartupReport implements gpu.StartupReporter (SPEC §9.7): the time it took to open the device, the
// time of the shaders and pipelines made so far, how many shaders were compiled, taken from the cache
// or refused (a cache file without a DXBC header), and the size of the pipeline library that was
// loaded.
//
// Flutter: no counterpart
func (d *Device) StartupReport() gpu.StartupReport {
	rep := gpu.StartupReport{Context: d.opened, Shaders: d.pipes.took, CacheBytes: d.library.loaded}
	if d.compiler != nil {
		st := d.compiler.Stats()
		rep.Built, rep.Cached, rep.Refused = st.Compiled, st.FromCache, st.Refused
	}
	return rep
}

var _ gpu.StartupReporter = (*Device)(nil)

// report logs the lines of the startup report that the device knows (SPEC §9.7, §9.8): the adapter,
// the feature level and the resource binding tier, and the time it took.
func (d *Device) report() {
	d.log.Info("Direct3D 12 device", "device", d.Description(), "feature_level", "11_0", "max_feature_level", d.FeatureLevel(),
		"binding_tier", d.bindingTier, "debug_layer", d.debugLayer, "opened", d.opened.String())
}

// Limits implements explicit.Device: the design limits of SPEC §9.3, which are inside those of
// feature level 11_0 (SPEC §9.8).
func (d *Device) Limits() explicit.Limits { return d.limits }

// Stats counts what the device has made and not released.
//
// Flutter: no counterpart
func (d *Device) Stats() Stats {
	s := Stats{
		Objects: d.objects, Buffers: d.counts.buffers, Textures: d.counts.textures, Samplers: d.counts.samplers,
		CommittedBytes: d.counts.bytes, OneOffUploads: d.counts.oneOff,
	}
	for _, h := range d.cpu {
		s.Descriptors += h.pool.live
	}
	for i := range s.StagingUsed {
		s.StagingUsed[i] = d.staging.ring.used(i)
	}
	s.Pipelines, s.RootSignatures = d.pipes.live, len(d.roots)
	for i := range d.slots {
		if v := d.slots[i].views; v != nil {
			s.ViewBlocks += len(v.heaps)
			s.SamplerBlocks += len(d.slots[i].samplers.heaps)
			s.ViewDescriptors += v.alloc.used()
			s.SamplerDescriptors += d.slots[i].samplers.alloc.used()
		}
	}
	return s
}

// DebugErrors returns the errors that the debug layer reported so far, when it is on.
//
// Flutter: no counterpart
func (d *Device) DebugErrors() []string { return append([]string(nil), d.debugErrors...) }

// Leaks is what Close found alive: the COM objects the device had not released, and the references
// that the device itself still had when it was released. It is 0 after a Close that left nothing.
//
// Flutter: no counterpart
func (d *Device) Leaks() int { return d.leaked }

// Close waits for the GPU, releases everything that was made with the device, the device and the
// adapter, and counts what it could not (Leaks). It can be called twice.
//
// Flutter: no counterpart
func (d *Device) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	if !d.dev.IsNil() {
		if err := d.waitIdle(); err != nil {
			d.log.Warn("waiting for the GPU before closing the device failed", "err", err)
		}
		for i := range d.slots {
			d.slots[i].runDeferred()
		}
		d.SavePipelineCache()
		for _, destroy := range d.live {
			destroy()
		}
		d.live = nil
		d.library.destroy(d)
		for _, rs := range d.roots {
			d.disown(rs.obj)
		}
		d.roots = nil
		d.staging.destroy(d)
		for _, h := range d.cpu {
			h.destroy(d)
		}
		for i := range d.slots {
			if d.slots[i].views != nil {
				d.slots[i].views.destroy(d)
				d.slots[i].samplers.destroy(d)
			}
			d.disown(d.slots[i].list)
			d.disown(d.slots[i].alloc)
			d.slots[i] = frameSlot{}
		}
		d.disown(d.fence)
		if d.fenceEvent != 0 {
			if err := win32.CloseHandle(d.fenceEvent); err != nil {
				d.log.Warn("closing the event of the fence failed", "err", err)
			}
			d.fenceEvent = 0
		}
		d.disown(d.queue)
		d.disown(d.infoQueue)
		d.disown(d.adapter)
		d.disown(d.factory)
		d.objects-- // the device
		rest := int(d.dev.Release())
		d.dev = win32.Object{}
		d.leaked = d.objects + rest
	} else {
		d.disown(d.adapter)
		d.disown(d.factory)
		d.leaked = d.objects
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagD3D12Unsupported is the tag of the error for adapters that cannot make a Direct3D 12 device at
	// feature level 11_0; the metadata are "feature_level" and "tried" (each adapter and why).
	TagD3D12Unsupported = "D3D12Unsupported"
)
