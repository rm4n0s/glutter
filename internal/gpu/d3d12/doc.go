// Package d3d12 is the Direct3D 12 backend of the explicit renderer (D25, SPEC §9.8): the third
// explicit.Device, after vulkan and webgpu, for Windows. Every call into Windows is made with the
// standard library's syscall, through internal/platform/windows/win32.
//
// This is the part that step 28.7 wrote: the shaders. The HLSL that cmd/glutter-gen shaders made
// from the WGSL (explicit.Shader.HLSL) is compiled by D3DCompile from d3dcompiler_47.dll into DXBC,
// shader model 5.1 (vs_5_1 and ps_5_1), when a pipeline first needs it, and the bytecode is kept in
// the shader cache (SPEC §9.7). D3DReflect reads the registers back from the bytecode, which is the
// contract with the root signatures of step 28.9: binding B of group G is at
// register(<class>B, spaceG).
//
//   - compiler.go and hlslcache.go are plain Go: the compile flags, the cache key and the file
//     check, and the Compiler that asks the cache first. They are tested on Linux with a stand-in
//     for D3DCompile.
//   - compile_windows.go (CompilerDLL) calls d3dcompiler_47.dll; reflect_windows.go reads the
//     registers. Their Go mirrors of the structures are in types.go with no OS tag, and
//     layout_test.go checks them against the mingw-w64 headers.
//
// Flutter: no counterpart. Flutter's Windows embedder draws through ANGLE.
package d3d12
