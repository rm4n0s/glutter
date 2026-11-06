package gpu

import "testing"

func TestShaderEntry(t *testing.T) {
	s := &Shader{Name: "blur", Entries: []EntryPoint{
		{Name: "vs_main", Stage: ShaderStageVertex},
		{Name: "cs_main", Stage: ShaderStageCompute, WorkgroupSize: [3]int{8, 8, 1}},
	}}
	tests := []struct {
		name   string
		entry  string
		wantOK bool
		stage  ShaderStage
	}{
		{"a vertex entry point", "vs_main", true, ShaderStageVertex},
		{"a compute entry point", "cs_main", true, ShaderStageCompute},
		{"an entry point the shader lacks", "fs_main", false, ShaderStageUnspecified},
		{"no name", "", false, ShaderStageUnspecified},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, ok := s.Entry(tt.entry)
			if ok != tt.wantOK || e.Stage != tt.stage {
				t.Errorf("Entry(%q) = %+v, %v; want stage %v, %v", tt.entry, e, ok, tt.stage, tt.wantOK)
			}
		})
	}
}

func TestNames(t *testing.T) {
	stages := []struct {
		s    ShaderStage
		want string
	}{{ShaderStageUnspecified, "unspecified"}, {ShaderStageVertex, "vertex"}, {ShaderStageFragment, "fragment"}, {ShaderStageCompute, "compute"}}
	for _, tt := range stages {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("ShaderStage(%d) = %q, want %q", tt.s, got, tt.want)
		}
	}
	kinds := []struct {
		k    BindingKind
		want string
	}{
		{BindingUnspecified, "unspecified"}, {BindingUniformBuffer, "uniform"}, {BindingSampledTexture, "texture"}, {BindingSampler, "sampler"},
		{BindingStorageBufferReadOnly, "storage-read"}, {BindingStorageBuffer, "storage"}, {BindingStorageTexture, "storage-texture"},
	}
	for _, tt := range kinds {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("BindingKind(%d) = %q, want %q", tt.k, got, tt.want)
		}
	}
}

// The zero values mean what the generated code leaves out: a 2D texture of floats.
func TestZeroValues(t *testing.T) {
	var b ShaderBinding
	if b.Dimension != TextureDim2D || b.Sample != SampleTypeFloat || b.Kind != BindingUnspecified {
		t.Errorf("zero ShaderBinding = %+v", b)
	}
}
