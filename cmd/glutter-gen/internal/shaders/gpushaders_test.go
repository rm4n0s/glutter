package shaders

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

var update = flag.Bool("update", false, "write the expected output of the gpushaders testdata")

// The directory with the shaders of the testdata and their expected output (shaders_gen.go and the .spv
// files are what gpushaders writes for the .wgsl files beside them, with naga 30.0.1).
const appTestdata = "../../testdata/gpushaders/shaders"

// appNaga returns the installed naga as the app generator uses it, or skips the test.
func appNaga(t *testing.T) Naga {
	t.Helper()
	n := Naga{App: true}
	v, err := n.Version()
	if err != nil {
		t.Skipf("naga is not installed (%s): %v", nagaInstall, err)
	}
	if v != "30.0.1" {
		t.Skipf("naga %s is installed, and the expected output was made with naga 30.0.1", v)
	}
	return n
}

func wantAppRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func TestLayoutSizes(t *testing.T) {
	tests := []struct {
		name string
		decl string // the declarations; the binding is `var<uniform> x: T` for the type of the test
		typ  string
		want int
	}{
		{"f32", "", "f32", 4},
		{"u32", "", "u32", 4},
		{"vec2", "", "vec2<f32>", 8},
		{"vec3 is 12 bytes", "", "vec3<f32>", 12},
		{"vec4", "", "vec4<i32>", 16},
		{"the short names", "", "vec3f", 12},
		{"mat4x4", "", "mat4x4<f32>", 64},
		{"mat3x3 has columns of 16", "", "mat3x3<f32>", 48},
		{"mat2x2", "", "mat2x2<f32>", 16},
		{"mat4x3 has four columns of 16", "", "mat4x3f", 64},
		{"an array of vec4", "", "array<vec4<f32>, 8>", 128},
		{"an array of f32 with a stride of 4", "", "array<f32, 5>", 20},
		{"an array whose length is a constant", "const N = 6;", "array<vec2<f32>, N>", 48},
		{"a vec3 then an f32 fill 16 bytes", "struct S { a: vec3<f32>, b: f32 }", "S", 16},
		{"an f32 then a vec3 leave a gap", "struct S { a: f32, b: vec3<f32> }", "S", 32},
		{"a struct is rounded to its alignment", "struct S { a: vec4<f32>, b: f32 }", "S", 32},
		{"@size and @align", "struct S { @size(16) a: f32, @align(16) b: f32 }", "S", 32},
		{"a struct in a struct", "struct A { x: f32 } struct S { a: A, b: vec2<f32> }", "S", 16},
		{"an array of structs", "struct P { a: vec2<f32>, b: f32 } ", "array<P, 3>", 48},
		{"an array without a length counts one element", "struct S { n: u32, items: array<vec2<f32>> }", "S", 16},
		{"an alias", "alias Vec = vec4<f32>;", "Vec", 16},
		{"an atomic", "", "atomic<u32>", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := tt.decl + "\n@group(0) @binding(0) var<uniform> x: " + tt.typ + ";\n@vertex fn vs() -> @builtin(position) vec4<f32> { _ = x; return vec4<f32>(0.0); }"
			m, err := parse(src, true, true)
			if err != nil {
				t.Fatal(err)
			}
			if got := m.Bindings[0].Size; got != tt.want {
				t.Errorf("size = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLayoutErrors(t *testing.T) {
	tests := []struct {
		name string
		decl string
		typ  string
	}{
		{"a type it does not know", "", "vec4<f16>"},
		{"a struct that holds itself", "struct S { a: S }", "S"},
		{"an array with a length it cannot read", "", "array<f32, count>"},
		{"@align below the alignment of the type", "struct S { @align(2) a: vec4<f32> }", "S"},
		{"@size below the size of the type", "struct S { @size(2) a: f32 }", "S"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := tt.decl + "\n@group(0) @binding(0) var<uniform> x: " + tt.typ + ";\n@vertex fn vs() -> @builtin(position) vec4<f32> { _ = x; return vec4<f32>(0.0); }"
			_, err := parse(src, true, true)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			// The route of a struct that holds itself is cut at the depth that the errors package keeps.
			if appErr.Tag != "ShadersLayoutUnsupported" || (tt.name != "a struct that holds itself" && !appErr.HasRoute("parse->layouts.bindingSize->layouts.sizeOf")) {
				t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestVertexInputs(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Input
	}{
		{"parameters", "@vertex fn vs(@location(0) a: vec2<f32>, @location(1) b: u32, @builtin(vertex_index) v: u32) -> @builtin(position) vec4<f32> { return vec4<f32>(0.0); }",
			[]Input{{"a", 0, "Float32x2"}, {"b", 1, "Uint32"}}},
		{"a struct and a parameter", "struct I { @location(2) c: vec4<i32>, @location(3) @interpolate(flat) d: i32 }\n@vertex fn vs(i: I, @location(0) e: vec3f) -> @builtin(position) vec4<f32> { return vec4<f32>(0.0); }",
			[]Input{{"c", 2, "Sint32x4"}, {"d", 3, "Sint32"}, {"e", 0, "Float32x3"}}},
		{"no inputs", "@vertex fn vs() -> @builtin(position) vec4<f32> { return vec4<f32>(0.0); }", nil},
		{"a fragment entry point has none", "@fragment fn fs(@location(0) a: vec4<f32>) -> @location(0) vec4<f32> { return a; }", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := parse(tt.src, true, true)
			if err != nil {
				t.Fatal(err)
			}
			if got := m.Entries[0].Inputs; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inputs = %v, want %v", got, tt.want)
			}
		})
	}
	_, err := parse("@vertex fn vs(@location(0) a: vec4<f16>) -> @builtin(position) vec4<f32> { return vec4<f32>(0.0); }", true, true)
	wantAppRoute(t, err, "ShadersLayoutUnsupported", "parse->layouts.inputs->layouts.input.ShadersLayoutUnsupported")
}

func TestAppBindingsThatAreNotAccepted(t *testing.T) {
	tests := []struct{ name, decl string }{
		{"a depth texture", "var t: texture_depth_2d;"},
		{"a multisampled texture", "var t: texture_multisampled_2d<f32>;"},
		{"a comparison sampler", "var t: sampler_comparison;"},
		{"a texture of integers with signs", "var t: texture_2d<i32>;"},
		{"a storage texture of another format", "var t: texture_storage_2d<rgba32float, write>;"},
		{"a storage texture that is read", "var t: texture_storage_2d<rgba8unorm, read>;"},
		{"a private variable with a binding", "var<private> t: f32;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "@group(0) @binding(0) " + tt.decl + "\n@vertex fn vs() -> @builtin(position) vec4<f32> { _ = t; return vec4<f32>(0.0); }"
			_, err := parse(src, true, true)
			wantAppRoute(t, err, "GPUShadersBindingUnsupported", "parse.GPUShadersBindingUnsupported")
		})
	}
}

func TestPackageName(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		dir   string
		want  string
	}{
		{"the package clause of a Go file", map[string]string{"a.go": "package scene\n"}, "shaders", "scene"},
		{"shaders_gen.go and tests are not read", map[string]string{"shaders_gen.go": "package old\n", "a_test.go": "package scene_test\n"}, "my-shaders", "myshaders"},
		{"the name of the directory", nil, "Shaders", "shaders"},
		{"a name that is not an identifier", nil, "3d", "shaders"},
		{"a name with nothing in it", nil, "-", "shaders"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), tt.dir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, src := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := PackageName(dir); got != tt.want {
				t.Errorf("PackageName = %q, want %q", got, tt.want)
			}
		})
	}
}

// copyWGSL copies the .wgsl files of the testdata to a new directory called shaders.
func copyWGSL(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "shaders")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(appTestdata, "*.wgsl"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no testdata: %v", err)
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(f)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestGenerateApp generates the shaders of the testdata and compares the files with the expected ones: every
// kind of binding, a vertex shader with several input types, a compute shader with a workgroup size and a
// shader that reads instance_index.
func TestGenerateApp(t *testing.T) {
	naga := appNaga(t)
	dir := copyWGSL(t)
	if err := GenerateApp(dir, dir, naga, AppOptions{ValidateSPIRV: validateWithSpirvVal}); err != nil {
		t.Fatal(err)
	}
	made, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range made {
		base := filepath.Base(f)
		if strings.HasSuffix(base, ".wgsl") {
			continue
		}
		got, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(appTestdata, base)
		if *update {
			if err := os.WriteFile(want, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		expected, err := os.ReadFile(want)
		if err != nil {
			t.Errorf("%s: %v (run the test with -update)", base, err)
			continue
		}
		if string(got) != string(expected) {
			t.Errorf("%s differs from %s (run the test with -update and look at the change)", base, want)
		}
	}
	if !*update {
		for _, want := range []string{"shaders_gen.go", "kinds.spv", "vertex.spv", "instance.spv"} {
			if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
				t.Errorf("%s was not made: %v", want, err)
			}
		}
	}
}

// TestGeneratedAppCodeCompiles runs go vet on the expected output, which is a package of the module:
// the generated code builds against package gpu.
func TestGeneratedAppCodeCompiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("the go command is not installed")
	}
	cmd := exec.Command("go", "vet", "./cmd/glutter-gen/testdata/gpushaders/shaders")
	cmd.Dir = "../../../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go vet: %v\n%s", err, out)
	}
}

// TestGeneratedAppReflection checks the reflection that the expected output holds, through the generated
// text, for the values an app depends on.
func TestGeneratedAppReflection(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(appTestdata, generated))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`{Name: "frame", Group: 0, Binding: 0, Kind: gpu.BindingUniformBuffer, Size: 96,`,
		`{Name: "lights", Group: 0, Binding: 1, Kind: gpu.BindingUniformBuffer, Size: 64,`,
		`{Name: "input", Group: 1, Binding: 0, Kind: gpu.BindingStorageBufferReadOnly, Size: 24,`,
		`{Name: "output", Group: 1, Binding: 1, Kind: gpu.BindingStorageBuffer, Size: 24,`,
		`Kind: gpu.BindingSampledTexture, Dimension: gpu.TextureDim2DArray,`,
		`Kind: gpu.BindingSampledTexture, Dimension: gpu.TextureDimCube,`,
		`Kind: gpu.BindingSampledTexture, Dimension: gpu.TextureDim3D,`,
		`Kind: gpu.BindingSampledTexture, Sample: gpu.SampleTypeUint,`,
		`Kind: gpu.BindingSampler,`,
		`Kind: gpu.BindingStorageTexture, StorageFormat: "rgba8unorm",`,
		`Kind: gpu.BindingStorageTexture, Dimension: gpu.TextureDim2DArray, StorageFormat: "rgba16float",`,
		`Kind: gpu.BindingStorageTexture, Dimension: gpu.TextureDim3D, StorageFormat: "r32float",`,
		`Stage: gpu.ShaderStageCompute, WorkgroupSize: [3]int{8, 4, 1}`,
		`Stage: gpu.ShaderStageVertex, ReadsInstanceIndex: true, ReadsVertexIndex: true`,
		`{Name: "position", Location: 0, Type: gpu.ShaderTypeFloat32x2}`,
		`{Name: "color", Location: 5, Type: gpu.ShaderTypeUint32x4}`,
		`{Name: "shift", Location: 7, Type: gpu.ShaderTypeSint32x2}`,
		`Register: gpu.ShaderRegister{Class: "u", Space: 1, Index: 1}`,
		"// Code generated by glutter-gen gpushaders; DO NOT EDIT.\n// Compiled from *.wgsl with naga 30.0.1",
		"package shaders",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the generated file lacks %q", want)
		}
	}
}

func TestCheckApp(t *testing.T) {
	naga := appNaga(t)
	fresh := func(t *testing.T) string {
		t.Helper()
		dir := copyWGSL(t)
		if err := GenerateApp(dir, dir, naga, AppOptions{}); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	t.Run("fresh output passes", func(t *testing.T) {
		if err := CheckApp(fresh(t), naga, AppOptions{}); err != nil {
			t.Fatal(err)
		}
	})
	tests := []struct {
		name  string
		edit  func(t *testing.T, dir string)
		files string
	}{
		{"an edited shader", func(t *testing.T, dir string) {
			f := filepath.Join(dir, "instance.wgsl")
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(f, []byte(strings.Replace(string(data), "0.0, 1.0", "0.5, 1.0", 1)), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "instance.spv shaders_gen.go"},
		{"an edited generated file", func(t *testing.T, dir string) {
			f := filepath.Join(dir, generated)
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(f, append(data, "// hand edit\n"...), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "shaders_gen.go"},
		{"a missing .spv", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "kinds.spv")); err != nil {
				t.Fatal(err)
			}
		}, "kinds.spv"},
		{"an .spv without a shader", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "old.spv"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "old.spv (has no .wgsl)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := fresh(t)
			tt.edit(t, dir)
			err := CheckApp(dir, naga, AppOptions{})
			wantAppRoute(t, err, "GPUShadersOutOfDate", "CheckApp.GPUShadersOutOfDate")
			appErr, _ := errors.FromError(err)
			files, _ := appErr.Metadata["files"].([]string)
			if got := strings.Join(files, " "); got != tt.files {
				t.Errorf("files = %q, want %q", got, tt.files)
			}
		})
	}
}

func TestRunGPUWritesAndRemovesStale(t *testing.T) {
	appNaga(t)
	dir := copyWGSL(t)
	stale := filepath.Join(dir, "gone.spv")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunGPU([]string{"-wine", "off", dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("gone.spv was not removed: %v", err)
	}
	if err := RunGPU([]string{"-check", "-wine", "off", dir}); err != nil {
		t.Errorf("-check fails on fresh output: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "instance.wgsl"), []byte(strings.Replace(mustRead(t, filepath.Join(dir, "instance.wgsl")), "1.0)", "2.0)", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	wantAppRoute(t, RunGPU([]string{"-check", "-wine", "off", dir}), "GPUShadersOutOfDate", "RunGPU->CheckApp.GPUShadersOutOfDate")
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRunGPUFlags(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		route string
	}{
		{"an unknown flag", []string{"-nope"}, "RunGPU.ShadersBadFlags"},
		{"no directory", nil, "RunGPU.ShadersBadFlags"},
		{"two directories", []string{"a", "b"}, "RunGPU.ShadersBadFlags"},
		{"a -wine that is not known", []string{"-wine", "maybe", "a"}, "RunGPU.ShadersBadFlags"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantAppRoute(t, RunGPU(tt.args), "ShadersBadFlags", tt.route)
		})
	}
}

// appTinyShader is tinyShader with the struct of its uniform.
const appTinyShader = "struct U { x: f32 }\n" + tinyShader

func TestGenerateAppErrors(t *testing.T) {
	t.Run("naga is not installed", func(t *testing.T) {
		dir := copyWGSL(t)
		err := GenerateApp(dir, dir, Naga{Path: "/nonexistent/naga", App: true}, AppOptions{})
		wantAppRoute(t, err, "GPUShadersNagaMissing", "GenerateApp->Naga.Version.GPUShadersNagaMissing")
		appErr, _ := errors.FromError(err)
		if got, _ := appErr.Metadata["install"].(string); got != nagaInstall || !strings.Contains(got, "30.0.1") {
			t.Errorf("install = %q", got)
		}
	})
	t.Run("no shaders", func(t *testing.T) {
		wantAppRoute(t, GenerateApp(t.TempDir(), t.TempDir(), &fakeNaga{version: "30.0.1"}, AppOptions{}), "ShadersNoShaders", "GenerateApp.ShadersNoShaders")
	})
	t.Run("two files with one name in Go", func(t *testing.T) {
		dir := t.TempDir()
		for _, n := range []string{"blur_pass.wgsl", "blurPass.wgsl"} {
			if err := os.WriteFile(filepath.Join(dir, n), []byte(appTinyShader), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		wantAppRoute(t, GenerateApp(dir, dir, &fakeNaga{version: "30.0.1", t: t}, AppOptions{}), "ShadersNameTwice", "GenerateApp.ShadersNameTwice")
	})
	t.Run("a shader that naga refuses", func(t *testing.T) {
		naga := appNaga(t)
		dir := t.TempDir()
		src := "@vertex fn vs() -> @builtin(position) vec4<f32> { return vec4<f32>(undefined_name); }\n"
		if err := os.WriteFile(filepath.Join(dir, "bad.wgsl"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		err := GenerateApp(dir, dir, naga, AppOptions{})
		wantAppRoute(t, err, "GPUShadersCompileFailed", "GenerateApp->compile->Naga.Compile.GPUShadersCompileFailed")
		appErr, _ := errors.FromError(err)
		if out, _ := appErr.Metadata["output"].(string); !strings.Contains(out, "undefined_name") {
			t.Errorf("the messages of naga are not in the error: %q", out)
		}
	})
	t.Run("a binding that is not accepted", func(t *testing.T) {
		dir := t.TempDir()
		src := "@group(0) @binding(0) var t: texture_depth_2d;\n@vertex fn vs() -> @builtin(position) vec4<f32> { _ = t; return vec4<f32>(0.0); }\n"
		if err := os.WriteFile(filepath.Join(dir, "depth.wgsl"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		err := GenerateApp(dir, dir, &fakeNaga{version: "30.0.1", t: t}, AppOptions{})
		wantAppRoute(t, err, "GPUShadersBindingUnsupported", "GenerateApp->compile->parse.GPUShadersBindingUnsupported")
		appErr, _ := errors.FromError(err)
		if appErr.Metadata["file"] == nil || appErr.Metadata["name"] != "t" {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
	})
	t.Run("a wgsl file that holds a backquote", func(t *testing.T) {
		dir := t.TempDir()
		src := appTinyShader + "// a `quoted` word is in the text that is written as a raw string\n"
		if err := os.WriteFile(filepath.Join(dir, "tick.wgsl"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		wantAppRoute(t, GenerateApp(dir, dir, &fakeNaga{version: "30.0.1", t: t}, AppOptions{}), "ShadersBacktick", "GenerateApp->renderApp.ShadersBacktick")
	})
}

func TestSPIRVValidation(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\necho \"error: id 7 is not defined\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, spirvVal), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.wgsl"), []byte(appTinyShader), 0o644); err != nil {
		t.Fatal(err)
	}
	err := GenerateApp(dir, dir, &fakeNaga{version: "30.0.1", t: t}, AppOptions{ValidateSPIRV: validateWithSpirvVal})
	wantAppRoute(t, err, "GPUShadersSPIRVInvalid", "GenerateApp->validateWithSpirvVal.GPUShadersSPIRVInvalid")
	appErr, _ := errors.FromError(err)
	if out, _ := appErr.Metadata["output"].(string); !strings.Contains(out, "id 7 is not defined") {
		t.Errorf("the messages of spirv-val are not in the error: %q", out)
	}
	t.Run("not installed is nothing to check", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		if err := validateWithSpirvVal("x.spv"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSpirvValAcceptsWhatNagaWrites(t *testing.T) {
	if _, err := exec.LookPath(spirvVal); err != nil {
		t.Skipf("%s is not installed (it is part of the Vulkan SDK and of spirv-tools)", spirvVal)
	}
	naga := appNaga(t)
	dir := copyWGSL(t)
	if err := GenerateApp(dir, dir, naga, AppOptions{ValidateSPIRV: validateWithSpirvVal}); err != nil {
		t.Fatal(err)
	}
}

// fakeWine makes a Glutter module in a temporary directory whose scripts/wine-exec.sh and go command are
// scripts, so that runWineCheck can be tested without Wine: go "builds" the program by writing the file,
// and the runner prints output and exits with status.
func fakeWine(t *testing.T, output string, status int) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+modulePath+"\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := "#!/bin/sh\ncat <<'EOF'\n" + output + "\nEOF\nexit " + itoa(status) + "\n"
	if err := os.WriteFile(filepath.Join(root, wineScript), []byte(runner), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	goScript := "#!/bin/sh\n# go build -o <exe> <package>\n: > \"$3\"\n"
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return root
}

func TestWineCheckReadsTheOutputOfTheChecker(t *testing.T) {
	shaders := []HLSLShader{{Name: "a", HLSL: "x", Entries: []HLSLEntry{{Name: "vs_main", Target: "vs_5_1"}}}}
	tests := []struct {
		name    string
		output  string
		status  int
		force   bool
		wantTag string
		route   string
	}{
		{"every entry compiles", "HLSLCHECK DONE 1", 0, true, "", ""},
		{"an entry that does not compile", "HLSLCHECK FAIL a vs_main vs_5_1: shader.hlsl(3,5): error X3004: undeclared identifier\nHLSLCHECK DONE 0", 1, false, "GPUShadersHLSLInvalid", "runWineCheck.GPUShadersHLSLInvalid"},
		{"Wine is not installed", "error: Wine is not installed.", 69, true, "GPUShadersWineUnavailable", "runWineCheck.GPUShadersWineUnavailable"},
		{"Wine is not installed and not asked for", "error: Wine is not installed.", 69, false, "", ""},
		{"Wine does not run the program", "wine: cannot find the program", 2, true, "GPUShadersWineUnavailable", "runWineCheck.GPUShadersWineUnavailable"},
		{"Wine does not run the program and is not asked for", "wine: cannot find the program", 2, false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := fakeWine(t, tt.output, tt.status)
			err := runWineCheck(root, shaders, tt.force)
			if tt.wantTag == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			wantAppRoute(t, err, tt.wantTag, tt.route)
			if tt.wantTag == "GPUShadersHLSLInvalid" {
				appErr, _ := errors.FromError(err)
				fails, _ := appErr.Metadata["failures"].([]string)
				if len(fails) != 1 || !strings.HasPrefix(fails[0], "a vs_main vs_5_1: ") || !strings.Contains(fails[0], "X3004") {
					t.Errorf("failures = %v", fails)
				}
			}
		})
	}
}

func TestWineCheckNeedsTheModule(t *testing.T) {
	outside := t.TempDir()
	check, err := newWineHLSLCheck(outside, false)
	if err != nil || check != nil {
		t.Errorf("auto outside the module: check=%v err=%v, want neither", check != nil, err)
	}
	_, err = newWineHLSLCheck(outside, true)
	wantAppRoute(t, err, "GPUShadersWineUnavailable", "newWineHLSLCheck.GPUShadersWineUnavailable")
	root := fakeWine(t, "HLSLCHECK DONE 0", 0)
	check, err = newWineHLSLCheck(filepath.Join(root, "app", "shaders"), false)
	if err != nil || check == nil {
		t.Fatalf("auto in the module: check=%v err=%v, want a check", check != nil, err)
	}
	if err := check(nil); err != nil {
		t.Fatal(err)
	}
}

// TestWineCompilesTheHLSL runs the real D3DCompile under Wine on the HLSL of the testdata shaders, and on HLSL
// that does not compile. It is skipped when Wine is not installed.
func TestWineCompilesTheHLSL(t *testing.T) {
	root := moduleRoot(".")
	if root == "" {
		t.Skip("not in the Glutter module")
	}
	if _, err := exec.LookPath("wine"); err != nil {
		if _, err := exec.LookPath("wine64"); err != nil {
			t.Skip("Wine is not installed (sudo dnf install wine-core)")
		}
	}
	naga := appNaga(t)
	dir := copyWGSL(t)
	check := func(shaders []HLSLShader) error { return runWineCheck(root, shaders, true) }
	if err := GenerateApp(dir, dir, naga, AppOptions{CheckHLSL: check}); err != nil {
		t.Fatal(err)
	}
	bad := []HLSLShader{{Name: "bad", HLSL: "float4 vs_main() : SV_Position { return undefined_name; }\n", Entries: []HLSLEntry{{Name: "vs_main", Target: "vs_5_1"}}}}
	err := runWineCheck(root, bad, true)
	wantAppRoute(t, err, "GPUShadersHLSLInvalid", "runWineCheck.GPUShadersHLSLInvalid")
	appErr, _ := errors.FromError(err)
	fails, _ := appErr.Metadata["failures"].([]string)
	if len(fails) != 1 || !strings.Contains(fails[0], "undefined_name") {
		t.Errorf("failures = %v", fails)
	}
}
