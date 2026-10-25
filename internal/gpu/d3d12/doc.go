// Package d3d12 is the Direct3D 12 backend of the explicit renderer (D25, SPEC §9.8): the third
// explicit.Device, after vulkan and webgpu, for Windows. Every call into Windows is made with the
// standard library's syscall, through internal/platform/windows/win32.
//
// The shaders (step 28.7): the HLSL that cmd/glutter-gen shaders made from the WGSL
// (explicit.Shader.HLSL) is compiled by D3DCompile from d3dcompiler_47.dll into DXBC, shader model
// 5.1 (vs_5_1 and ps_5_1), when a pipeline first needs it, and the bytecode is kept in the shader
// cache (SPEC §9.7). D3DReflect reads the registers back from the bytecode, which is the contract
// with the root signatures of step 28.9: binding B of group G is at register(<class>B, spaceG).
//
//   - compiler.go and hlslcache.go are plain Go: the compile flags, the cache key and the file
//     check, and the Compiler that asks the cache first. They are tested on Linux with a stand-in
//     for D3DCompile.
//   - compile_windows.go (CompilerDLL) calls d3dcompiler_47.dll; reflect_windows.go reads the
//     registers. Their Go mirrors of the structures are in types.go with no OS tag, and
//     layout_test.go checks them against the mingw-w64 headers.
//
// The device (SPEC §9.8, "Decided in step 28.8") is Open, which returns a Device, an explicit.Device
// offscreen: the choice of the adapter, the device at feature level 11_0, the queue and the fence, the
// two frame slots, committed buffers, textures and samplers, uploads through a staging ring, the read
// back of textures, and the state tracker that batches the barriers. Pipelines, bind groups and draws
// come with step 28.9, the swap chain with 28.10.
//
//   - adapter.go, errors.go, check.go, states.go, memory.go, mapping.go and debug.go are plain Go,
//     tested on Linux: the adapter choice, the table of HRESULTs, the state tracker, the staging
//     ring, the layout of a copy, the descriptor numbers, the mapping of formats and samplers, and the
//     reading of a message of the debug layer. types.go and vtables.go have the Go mirrors of the
//     structures of DXGI and Direct3D 12, the positions of the methods of their interfaces and their
//     IDs, all checked against the mingw-w64 headers by layout_test.go.
//   - dll_windows.go, device_windows.go, memory_windows.go, resources_windows.go, fence_windows.go,
//     frames_windows.go and descriptors_windows.go call Windows through syscall, and are tested under
//     Wine (device_wine_test.go, the glutter_wine tag).
//
// Flutter: no counterpart. Flutter's Windows embedder draws through ANGLE.
package d3d12
