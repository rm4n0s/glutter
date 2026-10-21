package explicit

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
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
			{Name: "vs_main", Stage: ShaderStageVertex, Bindings: []ShaderBinding{}},
			{Name: "fs_main", Stage: ShaderStageFragment, Bindings: []ShaderBinding{
				{Name: "frameTexture", Group: 0, Binding: 0, Kind: BindingSampledTexture},
				{Name: "frameSampler", Group: 0, Binding: 1, Kind: BindingSampler},
			}},
		}},
		{ShaderSolid, "solid", []EntryPoint{
			{Name: "vs_main", Stage: ShaderStageVertex, Bindings: []ShaderBinding{
				{Name: "frame", Group: 0, Binding: 0, Kind: BindingUniformBuffer},
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
