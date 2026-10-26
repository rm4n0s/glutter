//go:build windows && glutter_wine

package d3d12

import (
	"bytes"
	"encoding/binary"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// blitDesc is the pipeline of the blit into a BGRA image.
func blitDesc() explicit.PipelineDesc {
	return explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatBGRA8Unorm}
}

// solidDesc is the pipeline of the solid rectangles: one instance buffer with the rectangle and the
// color of each.
func solidDesc(blend explicit.BlendState) explicit.PipelineDesc {
	return explicit.PipelineDesc{
		Shader: explicit.ShaderSolid, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatRGBA8Unorm, Blend: blend,
		VertexBuffers: []explicit.VertexBufferLayout{{Stride: 32, PerInstance: true, Attributes: []explicit.VertexAttribute{
			{Location: 0, Format: explicit.VertexFloat32x4, Offset: 0}, {Location: 1, Format: explicit.VertexFloat32x4, Offset: 16},
		}}},
	}
}

// TestWineRootSignatureCoversWhatTheCompilerSees compiles both entry points of every pipeline of the
// renderer, reflects them, and checks that the root signature planned for the pipeline has a parameter
// for every resource the bytecode binds, at the register and the space it binds it (SPEC §9.8, the
// contract of step 28.7).
func TestWineRootSignatureCoversWhatTheCompilerSees(t *testing.T) {
	d := loadCompiler(t)
	flags := CompileFlags(func(string) string { return "" })
	for _, sh := range explicit.Shaders() {
		t.Run(sh.Name, func(t *testing.T) {
			vs, fs := entryOf(t, sh, "vs_main"), entryOf(t, sh, "fs_main")
			plan, err := planRoot(vs, fs, explicit.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range []explicit.EntryPoint{vs, fs} {
				dxbc, messages, hr := d.Compile(sh.HLSL, e.Name, targetOf(e.Stage), flags)
				if hr < 0 {
					t.Fatalf("D3DCompile %s failed with %s:\n%s", e.Name, hresultText(hr), messages)
				}
				refl, err := d.Reflect(dxbc)
				if err != nil {
					t.Fatal(err)
				}
				for _, r := range refl.Resources {
					found := false
					for _, p := range plan.Params {
						if p.Group != int(r.Space) {
							continue
						}
						switch {
						case r.Type == SITConstantBuffer && p.Kind == rootCBV && p.Binding == int(r.BindPoint):
							found = true
						case r.Type == SITTexture && p.Kind == rootSRVTable && containsInt(p.Bindings, int(r.BindPoint)):
							found = true
						case r.Type == SITSampler && p.Kind == rootSamplerTable && containsInt(p.Bindings, int(r.BindPoint)):
							found = true
						}
					}
					if !found {
						t.Errorf("%s binds %s (type %d, register %d, space %d), which no parameter of the root signature covers: %+v", e.Name, r.Name, r.Type, r.BindPoint, r.Space, plan.Params)
					}
				}
			}
		})
	}
}

func containsInt(list []int, n int) bool {
	for _, v := range list {
		if v == n {
			return true
		}
	}
	return false
}

// TestWineEveryPipelineOfTheRendererIsMade makes a pipeline of every shader, the blend variants of
// one shader share a root signature, and the pipelines are released with their device (Stats).
func TestWineEveryPipelineOfTheRendererIsMade(t *testing.T) {
	cache := shadercache.New(t.TempDir(), func(err error) { t.Errorf("the shader cache: %v", err) })
	d, _ := openTestDevice(t, Options{ShaderCache: cache})
	var made []explicit.Pipeline
	for _, sh := range explicit.Shaders() {
		desc := explicit.PipelineDesc{Shader: sh, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatRGBA8Unorm}
		switch sh.Name {
		case "solid":
			desc = solidDesc(explicit.BlendState{})
		case "shape", "mask", "image", "layer", "vertices":
			// The renderer's pipelines read the instances (or the vertices) from a buffer; the layout
			// does not matter for making the pipeline, only that its locations exist in the shader.
			desc.VertexBuffers = []explicit.VertexBufferLayout{{Stride: 16, PerInstance: true, Attributes: []explicit.VertexAttribute{{Location: 0, Format: explicit.VertexFloat32x4}}}}
		}
		p, err := d.NewPipeline(desc)
		if err != nil {
			t.Fatalf("%s: %+v", sh.Name, err)
		}
		if p.Desc().Shader != sh {
			t.Errorf("%s: Desc has another shader", sh.Name)
		}
		made = append(made, p)
	}
	st := d.Stats()
	if st.Pipelines != len(explicit.Shaders()) {
		t.Errorf("Pipelines = %d, want %d", st.Pipelines, len(explicit.Shaders()))
	}
	roots := st.RootSignatures
	// Four blends of the shape shader: no new root signature.
	for _, b := range []explicit.BlendState{
		{Enabled: true, SrcFactor: explicit.BlendOne, DstFactor: explicit.BlendOneMinusSrcAlpha},
		{Enabled: true, SrcFactor: explicit.BlendOneMinusDstAlpha, DstFactor: explicit.BlendOne},
		{Enabled: true, SrcFactor: explicit.BlendZero, DstFactor: explicit.BlendOneMinusSrcAlpha},
		{Enabled: true, SrcFactor: explicit.BlendSrcAlpha, DstFactor: explicit.BlendDstAlpha},
	} {
		desc := solidDesc(b)
		p, err := d.NewPipeline(desc)
		if err != nil {
			t.Fatalf("blend %+v: %+v", b, err)
		}
		made = append(made, p)
	}
	if got := d.Stats().RootSignatures; got != roots {
		t.Errorf("four blends of one shader made %d root signatures", got-roots)
	}
	t.Logf("%d pipelines, %d root signatures, shaders %+v, pipelines %+v", d.Stats().Pipelines, roots, d.CompilerStats(), d.PipelineStats())
	if cs := d.CompilerStats(); cs.Compiled != 2*len(explicit.Shaders()) {
		t.Errorf("compiled %d entry points for %d shaders: %+v", cs.Compiled, len(explicit.Shaders()), cs)
	}
	for _, p := range made {
		p.Destroy()
		p.Destroy() // twice is fine
	}
	if got := d.Stats().Pipelines; got != 0 {
		t.Errorf("%d pipelines after Destroy", got)
	}
	rep := d.StartupReport()
	if rep.Built != 2*len(explicit.Shaders()) || rep.Shaders <= 0 {
		t.Errorf("startup report %+v", rep)
	}
}

// TestWinePipelineErrors: the pipelines that cannot be made, with their routes.
func TestWinePipelineErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	u := explicit.ShaderBinding{Name: "u", Group: 0, Binding: 0, Kind: explicit.BindingUniformBuffer}
	tx := explicit.ShaderBinding{Name: "u", Group: 0, Binding: 0, Kind: explicit.BindingSampledTexture}
	sh := func(vs, fs []explicit.ShaderBinding, hlsl string) *explicit.Shader {
		return &explicit.Shader{Name: "made-up", HLSL: hlsl, Entries: []explicit.EntryPoint{
			{Name: "vs_main", Stage: explicit.ShaderStageVertex, Bindings: vs}, {Name: "fs_main", Stage: explicit.ShaderStageFragment, Bindings: fs}}}
	}
	rgba := explicit.TextureFormatRGBA8Unorm
	tests := []struct {
		name  string
		desc  explicit.PipelineDesc
		tag   string
		route string
	}{
		{"no shader", explicit.PipelineDesc{VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: rgba}, tagD3D12PipelineInvalid, "Device.NewPipeline.D3D12PipelineInvalid"},
		{"a shader without HLSL", explicit.PipelineDesc{Shader: sh(nil, nil, ""), VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: rgba}, tagD3D12PipelineInvalid, "Device.NewPipeline.D3D12PipelineInvalid"},
		{"an entry point the shader lacks", explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_other", Target: rgba}, tagD3D12PipelineInvalid, "Device.NewPipeline.D3D12PipelineInvalid"},
		{"no target format", explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_main"}, tagD3D12PipelineInvalid, "Device.NewPipeline.D3D12PipelineInvalid"},
		{"a binding of two kinds", explicit.PipelineDesc{Shader: sh([]explicit.ShaderBinding{u}, []explicit.ShaderBinding{tx}, "x"), VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: rgba}, tagD3D12PipelineInvalid, "Device.NewPipeline->planRoot.D3D12PipelineInvalid"},
		{"HLSL that does not compile", explicit.PipelineDesc{Shader: sh(nil, nil, "this is not HLSL"), VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: rgba}, TagD3D12ShaderCompileFailed, "Device.NewPipeline->Device.bytecode->Compiler.Compile.D3D12ShaderCompileFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := d.NewPipeline(tt.desc)
			if p != nil {
				t.Errorf("a pipeline was made: %v", p)
			}
			_ = wantRoute(t, err, tt.tag, tt.route)
		})
	}
	if got := d.Stats().Pipelines; got != 0 {
		t.Errorf("%d pipelines after the failures", got)
	}
}

// TestWinePipelineLibrary says what the device does with ID3D12PipelineLibrary on this Direct3D 12:
// where CreatePipelineLibrary works (Windows) the device has a library; where it does not (vkd3d) the
// device has none and still makes its pipelines (SPEC §9.8 "Pipelines").
func TestWinePipelineLibrary(t *testing.T) {
	cache := shadercache.New(t.TempDir(), func(err error) { t.Errorf("the shader cache: %v", err) })
	d, rec := openTestDevice(t, Options{ShaderCache: cache})
	p, err := d.NewPipeline(blitDesc())
	if err != nil {
		t.Fatalf("NewPipeline: %+v", err)
	}
	defer p.Destroy()
	have := !d.library.IsNil()
	t.Logf("CreatePipelineLibrary gave a library: %v", have)
	for _, r := range rec.records {
		if strings.Contains(r.Message, "pipeline library") {
			var attrs []string
			r.Attrs(func(a slog.Attr) bool { attrs = append(attrs, a.String()); return true })
			t.Logf("log: %s %v", r.Message, attrs)
		}
	}
	if have {
		if d.PipelineStats().Created != 1 {
			t.Errorf("made %+v", d.PipelineStats())
		}
		return
	}
	if v, ok := rec.find("there is no pipeline library on this device"); !ok {
		t.Error("the device did not say that it has no pipeline library")
	} else if h, _ := attr(v, "hresult"); h.String() == "" {
		t.Error("the hresult of CreatePipelineLibrary is not logged")
	}
	if d.PipelineStats().Created != 1 {
		t.Errorf("a device without a library made %+v, want 1 pipeline", d.PipelineStats())
	}
	d.SavePipelineCache() // nothing to save, and no panic
}

// fakeLibrary is an ID3D12PipelineLibrary made in Go, for the code that stores pipelines into the
// library and saves its blob, which the library of vkd3d (none) cannot run. It keeps the names that
// were stored and serializes them as bytes; loading never finds a pipeline.
type fakeLibrary struct {
	mu       sync.Mutex
	stores   int
	given    []byte // the blob it was made from
	serials  int
	loadsTry int
}

var (
	libHostOnce sync.Once
	libHost     *win32.Host
	libType     *win32.InterfaceType
	libHostErr  error
)

// newFakeLibrary makes the COM object; the Host and the interface type are made once for the process
// (every callback is made once, SPEC §6.3 rule 11).
func newFakeLibrary(t *testing.T, blob []byte) (win32.Object, *fakeLibrary) {
	t.Helper()
	libHostOnce.Do(func() {
		libHost, libHostErr = win32.NewHost(nil)
		if libHostErr != nil {
			return
		}
		filler := win32.MethodDef{Name: "unused", Args: 1, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.SOK) }}
		libType, libHostErr = libHost.NewInterface(win32.InterfaceDef{Name: "ID3D12PipelineLibrary", IID: iidPipelineLibrary, Methods: []win32.MethodDef{
			filler, filler, filler, filler, filler, // the methods of ID3D12Object (4) and ID3D12DeviceChild::GetDevice
			{Name: "StorePipeline", Args: 2, Fn: func(impl any, _ []uintptr) uintptr {
				l := impl.(*fakeLibrary)
				l.mu.Lock()
				defer l.mu.Unlock()
				l.stores++
				return win32.Result(win32.SOK)
			}},
			{Name: "LoadGraphicsPipeline", Args: 4, Fn: func(impl any, _ []uintptr) uintptr {
				l := impl.(*fakeLibrary)
				l.mu.Lock()
				defer l.mu.Unlock()
				l.loadsTry++
				return win32.Result(win32.EInvalidArg) // not found, as a library without the name says
			}},
			{Name: "LoadComputePipeline", Args: 4, Fn: func(any, []uintptr) uintptr { return win32.Result(win32.EInvalidArg) }},
			{Name: "GetSerializedSize", Args: 0, Fn: func(any, []uintptr) uintptr { return 16 }},
			{Name: "Serialize", Args: 2, Fn: func(impl any, a []uintptr) uintptr {
				l := impl.(*fakeLibrary)
				l.mu.Lock()
				defer l.mu.Unlock()
				l.serials++
				out := win32.Bytes(a[0], int(a[1]))
				binary.LittleEndian.PutUint64(out, 0xD3D1200D3D120001)
				binary.LittleEndian.PutUint64(out[8:], uint64(l.stores))
				return win32.Result(win32.SOK)
			}},
		}})
	})
	if libHostErr != nil {
		t.Fatal(libHostErr)
	}
	l := &fakeLibrary{given: append([]byte(nil), blob...)}
	obj, err := libHost.NewObject(l, libType)
	if err != nil {
		t.Fatal(err)
	}
	return obj, l
}

// libRig is a device whose pipeline library is a fakeLibrary, made at the first pipeline.
type libRig struct {
	d     *Device
	lib   *fakeLibrary
	calls int
}

func openWithFakeLibrary(t *testing.T, cache BlobCache, refuseBlobs bool) *libRig {
	t.Helper()
	g := &libRig{}
	g.d, _ = openTestDevice(t, Options{ShaderCache: cache, makeLibrary: func(blob []byte) (win32.Object, win32.HRESULT) {
		g.calls++
		if refuseBlobs && len(blob) > 0 {
			return win32.Object{}, win32.HRESULT(int32(-0x7781FFFE)) // D3D12_ERROR_DRIVER_VERSION_MISMATCH
		}
		var obj win32.Object
		obj, g.lib = newFakeLibrary(t, blob)
		return obj, win32.SOK
	}})
	return g
}

// TestWinePipelineLibraryStoresAndSaves uses a library of its own: a pipeline that it does not have is
// made and stored, the blob is saved into the shader cache under the key of the adapter when the device
// asks (and again only when it grew), and a second device reads that blob and gives it to
// CreatePipelineLibrary; a blob the library refuses is dropped and the library starts empty.
func TestWinePipelineLibraryStoresAndSaves(t *testing.T) {
	cache := shadercache.New(t.TempDir(), func(err error) { t.Errorf("the shader cache: %v", err) })
	g1 := openWithFakeLibrary(t, cache, false)
	d1 := g1.d
	p, err := d1.NewPipeline(blitDesc())
	if err != nil {
		t.Fatal(err)
	}
	if d1.library.IsNil() || g1.lib == nil {
		t.Fatal("the device has no library")
	}
	if g1.lib.loadsTry != 1 || g1.lib.stores != 1 || !d1.library.dirty {
		t.Errorf("the pipeline was looked up %d times and stored %d times; dirty %v", g1.lib.loadsTry, g1.lib.stores, d1.library.dirty)
	}
	if d1.PipelineStats().Created != 1 || d1.PipelineStats().FromLibrary != 0 {
		t.Errorf("pipelines %+v", d1.PipelineStats())
	}
	key := d1.library.key
	if key != LibraryKey(d1.info.VendorID, d1.info.DeviceID, d1.info.Revision, d1.driverVersion()) {
		t.Errorf("library key %q", key)
	}
	d1.SavePipelineCache()
	blob, ok := cache.Load(key)
	if !ok || len(blob) != 16 || binary.LittleEndian.Uint64(blob) != 0xD3D1200D3D120001 || binary.LittleEndian.Uint64(blob[8:]) != 1 {
		t.Fatalf("the cache has %x under %q", blob, key)
	}
	if d1.library.dirty {
		t.Error("the library is still dirty after the save")
	}
	d1.SavePipelineCache()
	if g1.lib.serials != 1 {
		t.Errorf("serialized %d times, want once: a save with nothing new does nothing", g1.lib.serials)
	}
	p.Destroy()
	// A second device reads the blob and gives it to the library.
	g2 := openWithFakeLibrary(t, cache, false)
	if _, err := g2.d.NewPipeline(blitDesc()); err != nil {
		t.Fatal(err)
	}
	if g2.lib == nil || string(g2.lib.given) != string(blob) {
		t.Errorf("the second library was made from %x, want %x", g2.lib.given, blob)
	}
	if g2.d.StartupReport().CacheBytes != len(blob) || g2.calls != 1 {
		t.Errorf("report %+v, %d calls", g2.d.StartupReport(), g2.calls)
	}
	// A library that refuses the blob: it is dropped and the library starts empty.
	g3 := openWithFakeLibrary(t, cache, true)
	if _, err := g3.d.NewPipeline(blitDesc()); err != nil {
		t.Fatal(err)
	}
	if g3.d.library.IsNil() || g3.calls != 2 || g3.d.StartupReport().CacheBytes != 0 || len(g3.lib.given) != 0 {
		t.Errorf("after a refused blob: library nil %v, %d calls, report %+v", g3.d.library.IsNil(), g3.calls, g3.d.StartupReport())
	}
}

// TestWineLibraryBlobIsCopiedToTheHeap checks the memory that CreatePipelineLibrary keeps: the library
// reads its pipelines from the blob it was made with for as long as it lives, so the blob is a copy in
// process heap memory that holds the same bytes, and is given back by freeBlob
// (SPEC §8.1, "data C keeps lives in C memory").
func TestWineLibraryBlobIsCopiedToTheHeap(t *testing.T) {
	tests := []struct {
		name string
		blob []byte
	}{
		{name: "a few bytes", blob: []byte{1, 2, 3, 4, 5}},
		{name: "a page and more", blob: func() []byte {
			b := make([]byte, 5000)
			for i := range b {
				b[i] = byte(i * 7)
			}
			return b
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Device{log: slog.New(slog.DiscardHandler)}
			mem, err := copyBlob(tt.blob)
			if err != nil {
				t.Fatal(err)
			}
			d.library.blobMem = mem
			defer d.library.freeBlob(d)
			if mem == 0 {
				t.Fatal("the copy is at 0")
			}
			if got := win32.Bytes(mem, len(tt.blob)); !bytes.Equal(got, tt.blob) {
				t.Errorf("the copy differs from the blob")
			}
			d.library.freeBlob(d)
			if d.library.blobMem != 0 {
				t.Errorf("blobMem = %#x after freeBlob, want 0", d.library.blobMem)
			}
		})
	}
}
