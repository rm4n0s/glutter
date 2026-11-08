package gles

import "strings"

// softwareRenderers are the parts of GL_RENDERER (lower case) that name a rasterizer on the CPU: Mesa's
// llvmpipe, softpipe and swrast, "Software Rasterizer" (Mesa's fallback name and Apple's), and Google's
// SwiftShader. The check is a table so that a driver that is not on it is added with one line (SPEC
// §9.6, step 32.3). Hardware drivers do not have these words in their names: "AMD Radeon ... (radeonsi
// ...)", "Mesa Intel(R) ...", "NVIDIA ...", "virgl" (a GPU on the host of a virtual machine) and
// "Adreno" are not software.
var softwareRenderers = []string{"llvmpipe", "softpipe", "swrast", "software rasterizer", "swiftshader"}

// Software says whether a GL_RENDERER string names a rasterizer that runs on the CPU, such as llvmpipe,
// which is what OpenGL is in a virtual machine with no GPU. The software renderer of Glutter (package
// shmwindow) redraws only what changed, while OpenGL on a CPU rasterizes the whole window for every
// frame, so the engine prefers it there (RendererOpts.RejectSoftware).
//
// Flutter: no counterpart
func Software(renderer string) bool {
	name := strings.ToLower(renderer)
	for _, s := range softwareRenderers {
		if strings.Contains(name, s) {
			return true
		}
	}
	return false
}
