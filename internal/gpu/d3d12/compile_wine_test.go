//go:build windows && glutter_wine

package d3d12

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
)

// These tests run the real d3dcompiler_47.dll: Wine's own, built on vkd3d-shader (SPEC §25.13). It is
// not Microsoft's compiler, so what they check is what holds in both: the bytecode is a DXBC
// container, the registers of the reflection are the mapping of the generator, and a broken shader
// gives the compiler's text with the position of the error. How fast it is, what it warns about and
// the exact words of its messages differ.

// loadCompiler loads the compiler of the prefix.
func loadCompiler(t *testing.T) *CompilerDLL {
	t.Helper()
	d, err := LoadCompilerDLL()
	if err != nil {
		t.Fatalf("LoadCompilerDLL: %v", err)
	}
	return d
}

// targetOf is the compile target of a stage.
func targetOf(s explicit.ShaderStage) string {
	if s == explicit.ShaderStageVertex {
		return TargetVertex
	}
	return TargetPixel
}

// classOf is the D3D_SHADER_INPUT_TYPE of a binding kind.
func classOf(k explicit.BindingKind) uint32 {
	switch k {
	case explicit.BindingUniformBuffer:
		return SITConstantBuffer
	case explicit.BindingSampledTexture:
		return SITTexture
	}
	return SITSampler
}

func TestWineCompilerHasAVersion(t *testing.T) {
	d := loadCompiler(t)
	if d.Version() == "" {
		t.Fatal("the compiler has no file version: nothing would be cached")
	}
	t.Logf("d3dcompiler_47.dll %s", d.Version())
}

// Every entry point of every shader compiles to vs_5_1 or ps_5_1 and starts with DXBC.
func TestWineEveryEntryPointCompiles(t *testing.T) {
	d := loadCompiler(t)
	flags := CompileFlags(func(string) string { return "" })
	var total time.Duration
	count := 0
	for _, s := range explicit.Shaders() {
		for _, e := range s.Entries {
			t.Run(s.Name+"/"+e.Name, func(t *testing.T) {
				start := time.Now()
				dxbc, messages, hr := d.Compile(s.HLSL, e.Name, targetOf(e.Stage), flags)
				total += time.Since(start)
				count++
				if hr < 0 {
					t.Fatalf("D3DCompile failed with %s:\n%s", hresultText(hr), messages)
				}
				if len(dxbc) < 4 || string(dxbc[:4]) != "DXBC" {
					t.Fatalf("the bytecode does not start with DXBC: % x", dxbc[:min(len(dxbc), 8)])
				}
				if !ValidDXBC(dxbc) {
					t.Errorf("the header of the bytecode is not valid: %d bytes", len(dxbc))
				}
				if messages != "" {
					t.Logf("messages: %s", messages)
				}
			})
		}
	}
	t.Logf("%d entry points compiled in %v", count, total)
	if count < 2*len(explicit.Shaders()) {
		t.Errorf("only %d entry points: every shader has a vertex and a fragment entry", count)
	}
}

// The debug flags give bytecode too (GLUTTER_D3D12_DEBUG=1).
func TestWineDebugFlagsCompile(t *testing.T) {
	d := loadCompiler(t)
	flags := CompileFlags(func(k string) string {
		if k == DebugEnvVar {
			return "1"
		}
		return ""
	})
	s := explicit.ShaderBlit
	for _, e := range s.Entries {
		dxbc, messages, hr := d.Compile(s.HLSL, e.Name, targetOf(e.Stage), flags)
		if hr < 0 || !ValidDXBC(dxbc) {
			t.Errorf("%s: %s %s", e.Name, hresultText(hr), messages)
		}
	}
}

// countingBackend counts the calls of D3DCompile and can report another version.
type countingBackend struct {
	Backend
	version string
	calls   int
}

func (c *countingBackend) Version() string { return c.version }

func (c *countingBackend) Compile(hlsl, entry, target string, flags uint32) ([]byte, string, int32) {
	c.calls++
	return c.Backend.Compile(hlsl, entry, target, flags)
}

// A second Compile with the cache is a hit and does not call D3DCompile; another version string of
// the compiler is a miss.
func TestWineCacheHitsAndMisses(t *testing.T) {
	d := loadCompiler(t)
	dir := t.TempDir()
	s := explicit.ShaderBlit
	flags := CompileFlags(func(string) string { return "" })

	be := &countingBackend{Backend: d, version: d.Version()}
	c := NewCompiler(be, shadercache.New(dir, nil))
	first, err := c.Compile(s.HLSL, "vs_main", TargetVertex, flags)
	if err != nil || first.FromCache || be.calls != 1 {
		t.Fatalf("first: %+v, %v, %d calls; want a compile", first.FromCache, err, be.calls)
	}
	second, err := c.Compile(s.HLSL, "vs_main", TargetVertex, flags)
	if err != nil || !second.FromCache || be.calls != 1 {
		t.Fatalf("second: from cache %v, %v, %d calls; want a hit and D3DCompile not called again", second.FromCache, err, be.calls)
	}
	if string(first.DXBC) != string(second.DXBC) {
		t.Error("the cached bytecode is not what was compiled")
	}

	// A new process: a new Compiler over the same directory also hits.
	again := &countingBackend{Backend: d, version: d.Version()}
	res, err := NewCompiler(again, shadercache.New(dir, nil)).Compile(s.HLSL, "vs_main", TargetVertex, flags)
	if err != nil || !res.FromCache || again.calls != 0 {
		t.Errorf("a second start: from cache %v, %v, %d calls; want a hit", res.FromCache, err, again.calls)
	}

	// Another entry point, another target and other flags are misses.
	for _, tt := range []struct {
		name          string
		entry, target string
		flags         uint32
	}{
		{"another entry point", "fs_main", TargetPixel, flags},
		{"other flags", "vs_main", TargetVertex, CompileDebug | CompileSkipOptimization},
	} {
		be.calls = 0
		res, err := c.Compile(s.HLSL, tt.entry, tt.target, tt.flags)
		if err != nil || res.FromCache || be.calls != 1 {
			t.Errorf("%s: from cache %v, %v, %d calls; want a miss", tt.name, res.FromCache, err, be.calls)
		}
	}

	// A changed version string of the compiler is a miss.
	newer := &countingBackend{Backend: d, version: d.Version() + ".1"}
	res, err = NewCompiler(newer, shadercache.New(dir, nil)).Compile(s.HLSL, "vs_main", TargetVertex, flags)
	if err != nil || res.FromCache || newer.calls != 1 {
		t.Errorf("a new compiler version: from cache %v, %v, %d calls; want a miss", res.FromCache, err, newer.calls)
	}
}

// A file of the cache that is not bytecode is dropped and compiled again.
func TestWineBadCacheFileIsReplaced(t *testing.T) {
	d := loadCompiler(t)
	cache := shadercache.New(t.TempDir(), nil)
	s := explicit.ShaderBlit
	flags := CompileFlags(func(string) string { return "" })
	cache.Store(CacheKey(s.HLSL, "vs_main", TargetVertex, flags, d.Version()), []byte("DXBC but cut short"))
	be := &countingBackend{Backend: d, version: d.Version()}
	res, err := NewCompiler(be, cache).Compile(s.HLSL, "vs_main", TargetVertex, flags)
	if err != nil || res.FromCache || be.calls != 1 || !ValidDXBC(res.DXBC) {
		t.Fatalf("from cache %v, %v, %d calls, valid %v; want a compile", res.FromCache, err, be.calls, ValidDXBC(res.DXBC))
	}
}

// The reflection of each compiled entry point reports registers and spaces that are the mapping the
// generator recorded: binding B of group G at register(<class>B, spaceG), with the class of its kind.
func TestWineReflectionMatchesTheMapping(t *testing.T) {
	d := loadCompiler(t)
	flags := CompileFlags(func(string) string { return "" })
	type slot struct{ class, bind, space uint32 }
	for _, s := range explicit.Shaders() {
		for _, e := range s.Entries {
			t.Run(s.Name+"/"+e.Name, func(t *testing.T) {
				dxbc, messages, hr := d.Compile(s.HLSL, e.Name, targetOf(e.Stage), flags)
				if hr < 0 {
					t.Fatalf("D3DCompile failed with %s:\n%s", hresultText(hr), messages)
				}
				refl, err := d.Reflect(dxbc)
				if err != nil {
					t.Fatal(err)
				}
				want := map[slot]string{}
				for _, b := range e.Bindings {
					want[slot{classOf(b.Kind), uint32(b.Register.Index), uint32(b.Register.Space)}] = b.Name
				}
				got := map[slot]string{}
				for _, r := range refl.Resources {
					if r.BindCount != 1 {
						t.Errorf("%s: takes %d registers", r.Name, r.BindCount)
					}
					got[slot{r.Type, r.BindPoint, r.Space}] = r.Name
				}
				for sl, name := range got {
					if _, ok := want[sl]; !ok {
						t.Errorf("the bytecode binds %s at type %d register %d space %d, which no binding of the entry point has", name, sl.class, sl.bind, sl.space)
					}
				}
				for sl, name := range want {
					if _, ok := got[sl]; !ok {
						t.Errorf("binding %s (type %d register %d space %d) is not in the bytecode", name, sl.class, sl.bind, sl.space)
					}
				}
			})
		}
	}
}

// The vertex stage reads its attributes through the semantics LOC0, LOC1, ... that the generator's
// naga wrote for the locations (the input layout of step 28.9 uses them).
func TestWineVertexInputsAreTheLocations(t *testing.T) {
	d := loadCompiler(t)
	flags := CompileFlags(func(string) string { return "" })
	s := explicit.ShaderSolid
	dxbc, messages, hr := d.Compile(s.HLSL, "vs_main", TargetVertex, flags)
	if hr < 0 {
		t.Fatalf("D3DCompile failed with %s:\n%s", hresultText(hr), messages)
	}
	refl, err := d.Reflect(dxbc)
	if err != nil {
		t.Fatal(err)
	}
	var loc []uint32
	vertexID := false
	for _, in := range refl.Inputs {
		switch in.SemanticName {
		case "LOC":
			loc = append(loc, in.SemanticIndex)
		case "SV_VertexID":
			vertexID = true
		}
	}
	if !vertexID || len(loc) != 2 || loc[0] != 0 || loc[1] != 1 {
		t.Errorf("inputs %+v: want SV_VertexID and LOC0, LOC1", refl.Inputs)
	}
}

// A deliberately broken HLSL gives D3D12ShaderCompileFailed with the compiler's text and the line.
func TestWineBrokenHLSL(t *testing.T) {
	d := loadCompiler(t)
	const broken = "float4 ok() { return 0; }\n\nfloat4 broken() { return undeclared_name; }\n"
	c := NewCompiler(d, nil)
	_, err := c.Compile(broken, "broken", TargetPixel, CompileFlags(func(string) string { return "" }))
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != TagD3D12ShaderCompileFailed || !appErr.HasRoute("Compiler.Compile.D3D12ShaderCompileFailed") {
		t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
	}
	messages, _ := appErr.Metadata["messages"].(string)
	t.Logf("the compiler says: %s", strings.TrimSpace(messages))
	// Microsoft's compiler writes "shader.hlsl(3,26-41): error X3004: undeclared identifier"; Wine's
	// writes "shader.hlsl:3:26: E5000: ...". Both name the source, the line 3 and the identifier.
	if !strings.Contains(messages, "shader.hlsl") || !strings.Contains(messages, "undeclared_name") || !strings.ContainsAny(messages, "3") {
		t.Errorf("the messages %q lack the source name, the line or the identifier", messages)
	}
	if !strings.Contains(messages, "(3,") && !strings.Contains(messages, ":3:") {
		t.Errorf("the messages %q do not give the line 3", messages)
	}
	// An entry point that is not there fails too.
	if _, _, hr := d.Compile(broken, "nothing", TargetPixel, 0); hr >= 0 {
		t.Error("an entry point that is not in the HLSL compiled")
	}
	// A target that does not exist fails.
	if _, _, hr := d.Compile(broken, "ok", "xx_9_9", 0); hr >= 0 {
		t.Error("a target that does not exist compiled")
	}
}
