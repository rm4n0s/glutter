package explicit

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestGeneratedShaders(t *testing.T) {
	tests := []struct {
		shader  *Shader
		file    string
		entries []EntryPoint
	}{
		{ShaderBlit, "blit", []EntryPoint{
			{Name: "vs_main", Stage: ShaderStageVertex, Bindings: []ShaderBinding{
				{Name: "blit", Group: 0, Binding: 2, Kind: BindingUniformBuffer, Register: ShaderRegister{Class: "b", Space: 0, Index: 2}},
			}},
			{Name: "fs_main", Stage: ShaderStageFragment, Bindings: []ShaderBinding{
				{Name: "frameTexture", Group: 0, Binding: 0, Kind: BindingSampledTexture, Register: ShaderRegister{Class: "t", Space: 0, Index: 0}},
				{Name: "frameSampler", Group: 0, Binding: 1, Kind: BindingSampler, Register: ShaderRegister{Class: "s", Space: 0, Index: 1}},
			}},
		}},
		{ShaderSolid, "solid", []EntryPoint{
			{Name: "vs_main", Stage: ShaderStageVertex, Bindings: []ShaderBinding{
				{Name: "frame", Group: 0, Binding: 0, Kind: BindingUniformBuffer, Register: ShaderRegister{Class: "b", Space: 0, Index: 0}},
			}},
			{Name: "fs_main", Stage: ShaderStageFragment, Bindings: []ShaderBinding{}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			s := tt.shader
			if s.Name != tt.file {
				t.Errorf("Name = %q", s.Name)
			}
			src, err := os.ReadFile(filepath.Join("shaders", tt.file+".wgsl"))
			if err != nil {
				t.Fatal(err)
			}
			if s.WGSL != string(src) {
				t.Error("the WGSL of the generated file is not the file's text")
			}
			// A SPIR-V module is words, and starts with the magic number 0x07230203.
			if len(s.SPIRV) < 20 || len(s.SPIRV)%4 != 0 || binary.LittleEndian.Uint32(s.SPIRV) != 0x07230203 {
				t.Errorf("SPIRV is not a SPIR-V module: %d bytes", len(s.SPIRV))
			}
			if !reflect.DeepEqual(s.Entries, tt.entries) {
				t.Errorf("entries\n got %+v\nwant %+v", s.Entries, tt.entries)
			}
			for _, e := range s.Entries {
				if got, ok := s.Entry(e.Name); !ok || got.Name != e.Name {
					t.Errorf("Entry(%q) = %v, %v", e.Name, got, ok)
				}
			}
			if _, ok := s.Entry("nothing"); ok {
				t.Error("Entry found an entry point that is not there")
			}
		})
	}
}

// TestGeneratedShadersFitTheLimits builds a pipeline description of each shader and
// checks it against the design limits.
func TestGeneratedShadersFitTheLimits(t *testing.T) {
	for _, s := range []*Shader{ShaderBlit, ShaderSolid} {
		DefaultLimits().CheckPipeline(PipelineDesc{Shader: s, VertexEntry: "vs_main", FragmentEntry: "fs_main"})
	}
}

func TestAlignedRowPitch(t *testing.T) {
	l := DefaultLimits()
	tests := []struct{ width, bpp, want int }{
		{1, 4, 256}, {64, 4, 256}, {65, 4, 512}, {256, 1, 256}, {257, 1, 512}, {0, 4, 0},
	}
	for _, tt := range tests {
		if got := l.AlignedRowPitch(tt.width, tt.bpp); got != tt.want {
			t.Errorf("AlignedRowPitch(%d, %d) = %d, want %d", tt.width, tt.bpp, got, tt.want)
		}
	}
}

func TestFormatsAndNames(t *testing.T) {
	for f, want := range map[TextureFormat]int{TextureFormatUnspecified: 0, TextureFormatRGBA8Unorm: 4, TextureFormatBGRA8Unorm: 4, TextureFormatR8Unorm: 1} {
		if got := f.BytesPerPixel(); got != want {
			t.Errorf("format %d has %d bytes per pixel, want %d", f, got, want)
		}
	}
	if ShaderStageVertex.String() != "vertex" || ShaderStageFragment.String() != "fragment" || ShaderStageUnspecified.String() != "unspecified" {
		t.Error("ShaderStage.String")
	}
	if BindingUniformBuffer.String() != "uniform" || BindingSampledTexture.String() != "texture" || BindingSampler.String() != "sampler" || BindingUnspecified.String() != "unspecified" {
		t.Error("BindingKind.String")
	}
}

// TestShadersListsEveryGeneratedShader: Shaders is the list that the WebGPU backend compiles at
// startup, so a shader file without an entry would reach the browser late.
func TestShadersListsEveryGeneratedShader(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("shaders", "*.wgsl"))
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, s := range Shaders() {
		listed[s.Name] = true
	}
	if len(files) == 0 || len(listed) != len(files) || len(listed) != len(Shaders()) {
		t.Fatalf("Shaders lists %d distinct shaders of %d, there are %d .wgsl files", len(listed), len(Shaders()), len(files))
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".wgsl")
		if !listed[name] {
			t.Errorf("shaders/%s.wgsl is not in Shaders()", name)
		}
	}
}

// TestGeneratedHLSL checks the HLSL of every shader against the contract of step 28.7 (D25, SPEC §9.8):
// each binding is at register(<class>B, spaceG), the class is the kind's, nothing of naga's sampler heap
// is left, and the entry points are functions of the text under their WGSL names.
func TestGeneratedHLSL(t *testing.T) {
	classes := map[BindingKind]string{BindingUniformBuffer: "b", BindingSampledTexture: "t", BindingSampler: "s"}
	for _, s := range Shaders() {
		t.Run(s.Name, func(t *testing.T) {
			if s.HLSL == "" {
				t.Fatal("no HLSL")
			}
			for _, bad := range []string{"nagaSamplerHeap", "nagaComparisonSamplerHeap", "SamplerIndexArray", "space255", "SV_InstanceID"} {
				if strings.Contains(s.HLSL, bad) {
					t.Errorf("the HLSL has %q", bad)
				}
			}
			for _, d := range regexp.MustCompile(`register\(([^)]*)\)`).FindAllStringSubmatch(s.HLSL, -1) {
				if !regexp.MustCompile(`^[bts]\d+, space\d+$`).MatchString(d[1]) {
					t.Errorf("register(%s) is not register(<class>B, spaceG)", d[1])
				}
			}
			for _, e := range s.Entries {
				if !regexp.MustCompile(`(?m)^\w+ ` + e.Name + `\(`).MatchString(s.HLSL) {
					t.Errorf("the HLSL has no function %s", e.Name)
				}
				for _, b := range e.Bindings {
					want := ShaderRegister{Class: classes[b.Kind], Space: b.Group, Index: b.Binding}
					if b.Register != want {
						t.Errorf("%s.%s: register %v, want %v", e.Name, b.Name, b.Register, want)
					}
					if !strings.Contains(s.HLSL, "register("+want.String()+")") {
						t.Errorf("%s.%s: the HLSL does not declare register(%s)", e.Name, b.Name, want)
					}
				}
			}
		})
	}
}

func TestShaderRegisterString(t *testing.T) {
	if got := (ShaderRegister{Class: "t", Space: 1, Index: 3}).String(); got != "t3, space1" {
		t.Errorf("String = %q", got)
	}
}
