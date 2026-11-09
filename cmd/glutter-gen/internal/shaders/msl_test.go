package shaders

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// nagaStyleMSL writes, for a WGSL module the parser accepts, the MSL in the shape naga 30 gives it: every
// entry point lists the resources that it reads as parameters named like the variables, all with
// [[user(fake0)]], and the lengths of runtime-sized arrays as a last parameter.
func nagaStyleMSL(t *testing.T, src string) string {
	t.Helper()
	m, err := Parse(src, true)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("// language: metal2.4\n#include <metal_stdlib>\n\nstruct _mslBufferSizes {\n    uint size0;\n};\n\n")
	for _, e := range m.Entries {
		stage, ret := "vertex", "vs_Output"
		switch e.Stage {
		case "Fragment":
			stage = "fragment"
		case "Compute":
			stage = "kernel"
			ret = "void"
		}
		b.WriteString(stage + " " + ret + " " + e.Name + "(\n  uint index [[vertex_id]]\n")
		for _, bd := range e.Bindings {
			switch bd.MetalClass() {
			case metalBuffer:
				b.WriteString(", constant T& " + bd.Name + " [[user(fake0)]]\n")
			case metalTexture:
				b.WriteString(", metal::texture2d<float, metal::access::sample> " + bd.Name + " [[user(fake0)]]\n")
			default:
				b.WriteString(", metal::sampler " + bd.Name + " [[user(fake0)]]\n")
			}
		}
		b.WriteString(", constant _mslBufferSizes& _buffer_sizes [[user(fake0)]]\n) {\n    return 0;\n}\n\n")
	}
	return b.String()
}

const mslModule = `@group(0) @binding(0) var<uniform> frame: F;
@group(0) @binding(1) var smp: sampler;
@group(0) @binding(2) var tex: texture_2d<f32>;
@group(1) @binding(0) var<uniform> extra: F;
@group(1) @binding(3) var tex2: texture_2d<f32>;
@vertex fn vs_main() -> @builtin(position) vec4<f32> { return vec4<f32>(frame.x + extra.x); }
@fragment fn fs_main() -> @location(0) vec4<f32> { return textureSample(tex2, smp, vec2<f32>(0.0)) + textureSample(tex, smp, vec2<f32>(0.0)); }
`

func TestMetalIndexesFollowTheOrderOfGroupAndBinding(t *testing.T) {
	m, err := Parse(mslModule, false)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		class string
		index int
	}{
		{"frame", metalBuffer, 0},
		{"extra", metalBuffer, 1},
		{"smp", metalSampler, 0},
		{"tex", metalTexture, 0},
		{"tex2", metalTexture, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, b := range m.Bindings {
				if b.Name == tt.name {
					if b.MetalClass() != tt.class || b.MetalIndex != tt.index {
						t.Errorf("%s is %s(%d), want %s(%d)", tt.name, b.MetalClass(), b.MetalIndex, tt.class, tt.index)
					}
					return
				}
			}
			t.Fatalf("no binding %s", tt.name)
		})
	}
	// An entry point's bindings carry the module's indexes, not their own count.
	for _, e := range m.Entries {
		if e.Name != "fs_main" {
			continue
		}
		for _, b := range e.Bindings {
			if b.Name == "tex2" && b.MetalIndex != 1 {
				t.Errorf("tex2 in fs_main has index %d, want the module's 1", b.MetalIndex)
			}
		}
	}
}

func TestRewriteMSLPutsTheResourcesAtTheirIndexes(t *testing.T) {
	m, err := Parse(mslModule, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := rewriteMSL(nagaStyleMSL(t, mslModule), m)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"constant T& frame [[buffer(0)]]",
		"constant T& extra [[buffer(1)]]",
		"metal::sampler smp [[sampler(0)]]",
		"metal::texture2d<float, metal::access::sample> tex2 [[texture(1)]]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the MSL lacks %q:\n%s", want, got)
		}
	}
	for _, gone := range []string{"fake", "_buffer_sizes", "_mslBufferSizes"} {
		if strings.Contains(got, gone) {
			t.Errorf("the MSL still has %q", gone)
		}
	}
	if err := checkMSL(got, m); err != nil {
		t.Errorf("checkMSL: %v", err)
	}
}

func TestRewriteMSLFailures(t *testing.T) {
	m, err := Parse(mslModule, false)
	if err != nil {
		t.Fatal(err)
	}
	good := nagaStyleMSL(t, mslModule)
	tests := []struct {
		name  string
		edit  func(string) string
		route string
	}{
		{"a carriage return", func(s string) string { return s + "\r\n" }, "rewriteMSL.ShadersMSLUnexpected"},
		{"an unknown resource", func(s string) string { return strings.Replace(s, "frame [[user", "mystery [[user", 1) }, "rewriteMSL.ShadersMSLUnexpected"},
		{"a read of the buffer sizes", func(s string) string { return strings.Replace(s, "return 0;", "return _buffer_sizes.size0;", 1) }, "rewriteMSL.ShadersMSLUnexpected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := rewriteMSL(tt.edit(good), m)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagShadersMSLUnexpected || !appErr.HasRoute(tt.route) {
				t.Fatalf("got %v, want %s", err, tt.route)
			}
		})
	}
}

func TestRewriteMSLFindsTheVariablesThatNagaRenamed(t *testing.T) {
	m, err := Parse(mslModule, false)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct{ name, renamed string }{
		{"a hidden name", "frame_1"},
		{"a name that ends in a digit", "tex2_"},
		{"both", "tex2_3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := "frame"
			if strings.HasPrefix(tt.renamed, "tex2") {
				old = "tex2"
			}
			src := strings.Replace(nagaStyleMSL(t, mslModule), old+" [[user", tt.renamed+" [[user", 1)
			got, err := rewriteMSL(src, m)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(got, "fake") {
				t.Errorf("a resource is left unbound:\n%s", got)
			}
		})
	}
}

func TestRewriteMSLRefusesMoreBuffersThanTheTableHolds(t *testing.T) {
	var b strings.Builder
	for i := 0; i <= maxMetalBuffers; i++ {
		b.WriteString("@group(0) @binding(" + itoa(i) + ") var<uniform> u" + itoa(i) + ": F;\n")
	}
	b.WriteString("@vertex fn vs_main() -> @builtin(position) vec4<f32> { return vec4<f32>(u0.x); }\n")
	m, err := Parse(b.String(), false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rewriteMSL("// msl\n", m)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagShadersMSLUnexpected || !appErr.HasRoute("rewriteMSL.ShadersMSLUnexpected") {
		t.Fatalf("got %v, want ShadersMSLUnexpected from rewriteMSL", err)
	}
}

func TestCheckMSLFailures(t *testing.T) {
	m, err := Parse(mslModule, false)
	if err != nil {
		t.Fatal(err)
	}
	good, err := rewriteMSL(nagaStyleMSL(t, mslModule), m)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		edit func(string) string
	}{
		{"an entry point that is missing", func(s string) string { return strings.Replace(s, "fs_main(", "fs_other(", 1) }},
		{"a resource that is missing", func(s string) string { return strings.Replace(s, "[[sampler(0)]]", "[[sampler(5)]]", 1) }},
		{"two resources at one index", func(s string) string { return strings.Replace(s, "[[buffer(1)]]", "[[buffer(0)]]", 1) }},
		{"a resource that is not read", func(s string) string {
			return strings.Replace(s, "\n) {\n    return 0;\n}\n\nfragment", "\n, constant T& more [[buffer(9)]]\n) {\n    return 0;\n}\n\nfragment", 1)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edited := tt.edit(good)
			if edited == good {
				t.Fatal("the edit changed nothing")
			}
			err := checkMSL(edited, m)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagShadersMSLMismatch || !appErr.HasRoute("checkMSL.ShadersMSLMismatch") {
				t.Fatalf("got %v, want ShadersMSLMismatch from checkMSL", err)
			}
		})
	}
}

func TestNagaCompileMSLFails(t *testing.T) {
	n := withNaga(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "bad.wgsl")
	if err := os.WriteFile(in, []byte("this is not WGSL"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := n.CompileMSL(in, filepath.Join(dir, "bad.metal"))
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagShadersNagaFailed || !appErr.HasRoute("Naga.CompileMSL.ShadersNagaFailed") {
		t.Fatalf("got %v, want ShadersNagaFailed via Naga.CompileMSL", err)
	}
	if appErr.Metadata["output"] == nil {
		t.Error("the error does not carry naga's message")
	}
}

// TestRealNagaMSLOfTheExplicitShaders runs naga on the shaders of the explicit renderer and of its device
// tests, and checks that the generator can rewrite and check what it writes.
func TestRealNagaMSLOfTheExplicitShaders(t *testing.T) {
	n := withNaga(t)
	for _, dir := range []string{"../../../../internal/gpu/explicit/shaders", "../../../../internal/gpu/explicit/devicetest/shaders"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.wgsl"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no shaders in %s: %v", dir, err)
		}
		for _, f := range files {
			t.Run(filepath.Base(dir)+"/"+filepath.Base(f), func(t *testing.T) {
				c, err := compile(f, t.TempDir(), filepath.Join(t.TempDir(), "x.spv"), n, strings.Contains(dir, "devicetest"), false)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(c.msl, "#include <metal_stdlib>") {
					t.Error("the MSL is not naga's")
				}
			})
		}
	}
}
