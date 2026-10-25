package d3d12

import (
	"fmt"

	"github.com/rm4n0s/errors"
)

// The flags of D3DCompile that the backend uses (d3dcompiler.h; the layout test checks them).
const (
	// CompileDebug is D3DCOMPILE_DEBUG: debugging information in the bytecode.
	//
	// Flutter: no counterpart
	CompileDebug uint32 = 1 << 0
	// CompileSkipOptimization is D3DCOMPILE_SKIP_OPTIMIZATION.
	//
	// Flutter: no counterpart
	CompileSkipOptimization uint32 = 1 << 2
	// CompileOptimizationLevel3 is D3DCOMPILE_OPTIMIZATION_LEVEL3: the highest optimization.
	//
	// Flutter: no counterpart
	CompileOptimizationLevel3 uint32 = 1 << 15
)

// DebugEnvVar is the environment variable that turns on the Direct3D debug layer (28.8) and the
// debug flags of the compiler when it is "1" (SPEC §9.8).
//
// Flutter: no counterpart
const DebugEnvVar = "GLUTTER_D3D12_DEBUG"

// CompileFlags returns the flags for D3DCompile (SPEC §9.8): optimization level 3 in a release
// build, and debugging information with no optimization when GLUTTER_D3D12_DEBUG is "1".
//
// Flutter: no counterpart
func CompileFlags(getenv func(string) string) uint32 {
	if getenv(DebugEnvVar) == "1" {
		return CompileDebug | CompileSkipOptimization
	}
	return CompileOptimizationLevel3
}

// The compile targets of the backend: shader model 5.1, vertex and pixel (SPEC §9.8).
const (
	// TargetVertex is the target of a vertex entry point.
	//
	// Flutter: no counterpart
	TargetVertex = "vs_5_1"
	// TargetPixel is the target of a fragment entry point.
	//
	// Flutter: no counterpart
	TargetPixel = "ps_5_1"
)

// Backend turns HLSL into DXBC: D3DCompile of d3dcompiler_47.dll (CompilerDLL), or a stand-in in a test.
//
// Flutter: no counterpart
type Backend interface {
	// Version is the file version of the compiler ("10.0.19041.1"), part of the cache key; empty
	// when it cannot be known, and then nothing is cached.
	Version() string
	// Compile compiles the entry point of the HLSL for the target. It returns the bytecode, the
	// compiler's messages (warnings on success, errors and warnings on failure) and the HRESULT
	// of D3DCompile, negative when the compile failed.
	Compile(hlsl, entry, target string, flags uint32) (dxbc []byte, messages string, hresult int32)
}

// CompileResult is what Compiler.Compile makes.
//
// Flutter: no counterpart
type CompileResult struct {
	// DXBC is the bytecode.
	DXBC []byte
	// Messages are the compiler's warnings; empty when the bytecode came from the cache.
	Messages string
	// FromCache says that the bytecode was read from the cache and D3DCompile did not run.
	FromCache bool
}

// CompilerStats counts what a Compiler did, for the startup report (SPEC §9.7) and for tests.
//
// Flutter: no counterpart
type CompilerStats struct {
	// Compiled is the number of shaders D3DCompile ran for, failures included.
	Compiled int
	// FromCache is the number of shaders read from the cache.
	FromCache int
	// Refused is the number of cache files that did not have a DXBC header and were dropped.
	Refused int
}

// Compiler compiles entry points of the generated HLSL to DXBC, reading the cache first and storing
// what it compiled. It is used from one goroutine, the renderer's.
//
// Flutter: no counterpart
type Compiler struct {
	backend Backend
	cache   BlobCache
	version string
	stats   CompilerStats
}

// NewCompiler makes a Compiler over the backend. cache may be nil: there is no cache directory, which
// is not an error. A backend that does not know its version caches nothing.
//
// Flutter: no counterpart
func NewCompiler(backend Backend, cache BlobCache) *Compiler {
	c := &Compiler{backend: backend, version: backend.Version()}
	if c.version != "" {
		c.cache = cache
	}
	return c
}

// Stats returns the counts so far.
//
// Flutter: no counterpart
func (c *Compiler) Stats() CompilerStats { return c.stats }

// Compile returns the bytecode of the entry point: from the cache when the file is there and has a
// DXBC header, else from D3DCompile, and then it is stored. A compile that fails is
// D3D12ShaderCompileFailed, with the compiler's messages (an internal bug: the HLSL comes from the
// generator), and nothing is stored.
//
// Flutter: no counterpart
func (c *Compiler) Compile(hlsl, entry, target string, flags uint32) (CompileResult, error) {
	var key string
	if c.cache != nil {
		key = CacheKey(hlsl, entry, target, flags, c.version)
		if data, ok := c.cache.Load(key); ok {
			if ValidDXBC(data) {
				c.stats.FromCache++
				return CompileResult{DXBC: data, FromCache: true}, nil
			}
			c.stats.Refused++ // dropped: the compile below replaces the file
		}
	}
	c.stats.Compiled++
	dxbc, messages, hr := c.backend.Compile(hlsl, entry, target, flags)
	if hr < 0 || len(dxbc) == 0 {
		return CompileResult{}, errors.New(TagD3D12ShaderCompileFailed, "D3DCompile failed on a generated shader",
			"entry", entry, "target", target, "hresult", hresultText(hr), "messages", messages)
	}
	if c.cache != nil {
		c.cache.Store(key, dxbc)
	}
	return CompileResult{DXBC: dxbc, Messages: messages}, nil
}

// hresultText writes an HRESULT as the headers do, "0x80004005".
func hresultText(hr int32) string { return fmt.Sprintf("0x%08X", uint32(hr)) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagD3D12ShaderCompileFailed is the tag of the error for HLSL that D3DCompile refuses; the
	// metadata are "entry", "target", "hresult" and "messages" (the compiler's text, with the line
	// and column of each error). It is an internal bug: the HLSL is generated (SPEC §9.8).
	TagD3D12ShaderCompileFailed = "D3D12ShaderCompileFailed"
)
