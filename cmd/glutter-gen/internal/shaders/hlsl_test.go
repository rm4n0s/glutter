package shaders

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// fixtureWGSL has a binding of every kind in two groups: a uniform, a texture and a sampler in group 0 and
// a texture, a sampler, a depth texture and a comparison sampler in group 1, with the same binding number
// (5) in the two samplers. The comparison sampler is not a binding the generator accepts (SPEC §9.3), so
// this is the input of the rewrite only.
const fixtureWGSL = `struct U { m: vec4<f32> };
@group(0) @binding(2) var<uniform> u: U;
@group(0) @binding(0) var t0: texture_2d<f32>;
@group(0) @binding(5) var s0: sampler;
@group(1) @binding(3) var t1: texture_2d<f32>;
@group(1) @binding(5) var s1: sampler;
@group(1) @binding(6) var dt: texture_depth_2d;
@group(1) @binding(7) var cs: sampler_comparison;
struct VO { @builtin(position) p: vec4<f32>, @location(0) uv: vec2<f32> };
@vertex fn vs(@builtin(vertex_index) vi: u32) -> VO {
  var o: VO; o.p = u.m * f32(vi); o.uv = vec2<f32>(0.0); return o;
}
@fragment fn fs(i: VO) -> @location(0) vec4<f32> {
  return textureSample(t0, s0, i.uv) + textureSample(t1, s1, i.uv) + vec4<f32>(textureSampleCompare(dt, cs, i.uv, 0.5));
}
`

// fixtureHLSL is what naga 30.0.1 writes for fixtureWGSL with --shader-model 51 (TestFixtureIsWhatNagaWrites
// checks it when that naga is installed).
const fixtureHLSL = `struct U {
    float4 m;
};

struct VO {
    float4 p : SV_Position;
    float2 uv : LOC0;
};

cbuffer u : register(b2) { U u; }
Texture2D<float4> t0_ : register(t0);
SamplerState nagaSamplerHeap[2048]: register(s0, space0);
SamplerComparisonState nagaComparisonSamplerHeap[2048]: register(s0, space1);
StructuredBuffer<uint> nagaGroup0SamplerIndexArray : register(t0, space255);
static const SamplerState s0_ = nagaSamplerHeap[nagaGroup0SamplerIndexArray[5]];
Texture2D<float4> t1_ : register(t3, space1);
StructuredBuffer<uint> nagaGroup1SamplerIndexArray : register(t1, space255);
static const SamplerState s1_ = nagaSamplerHeap[nagaGroup1SamplerIndexArray[5]];
Texture2D<float> dt : register(t6, space1);
static const SamplerComparisonState cs = nagaComparisonSamplerHeap[nagaGroup1SamplerIndexArray[7]];

struct VertexOutput_vs {
    float2 uv : LOC0;
    float4 p : SV_Position;
};

struct FragmentInput_fs {
    float2 uv_1 : LOC0;
    float4 p_1 : SV_Position;
};

VertexOutput_vs vs(uint vi : SV_VertexID)
{
    VO o = (VO)0;

    float4 _e5 = u.m;
    o.p = (_e5 * float(vi));
    o.uv = (0.0).xx;
    VO _e11 = o;
    const VO vo = _e11;
    const VertexOutput_vs vo_1 = { vo.uv, vo.p };
    return vo_1;
}

float4 fs(FragmentInput_fs fragmentinput_fs) : SV_Target0
{
    VO i = { fragmentinput_fs.p_1, fragmentinput_fs.uv_1 };
    float4 _e4 = t0_.Sample(s0_, i.uv);
    float4 _e8 = t1_.Sample(s1_, i.uv);
    float _e14 = dt.SampleCmp(cs, i.uv, 0.5);
    return ((_e4 + _e8) + (_e14).xxxx);
}
`

func TestRewriteHLSL(t *testing.T) {
	got, err := rewriteHLSL(fixtureHLSL)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"cbuffer u : register(b2, space0) { U u; }",
		"Texture2D<float4> t0_ : register(t0, space0);",
		"SamplerState s0_ : register(s5, space0);",
		"Texture2D<float4> t1_ : register(t3, space1);",
		"SamplerState s1_ : register(s5, space1);",
		"Texture2D<float> dt : register(t6, space1);",
		"SamplerComparisonState cs : register(s7, space1);",
		"float4 fs(FragmentInput_fs fragmentinput_fs) : SV_Target0", // the rest of the text is naga's
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the HLSL lacks %q:\n%s", want, got)
		}
	}
	for _, gone := range leftoverFragments {
		if strings.Contains(got, gone) {
			t.Errorf("the HLSL still has %q", gone)
		}
	}
	// Only the lines of the sampler machinery changed: the text is otherwise naga's.
	if a, b := strings.Count(fixtureHLSL, "\n"), strings.Count(got, "\n"); a-b != 4 {
		t.Errorf("%d lines became %d, want the 4 lines of the two heaps and the two index buffers gone\n%s", a, b, got)
	}
}

func TestRewriteHLSLIsIdempotentOnItsOutput(t *testing.T) {
	once, err := rewriteHLSL(fixtureHLSL)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := rewriteHLSL(once)
	if err != nil {
		t.Fatal(err)
	}
	if once != twice {
		t.Error("a second rewrite changed the text")
	}
}

func TestRewriteHLSLErrors(t *testing.T) {
	const heap = "SamplerState nagaSamplerHeap[2048]: register(s0, space0);"
	tests := []struct {
		name string
		edit func(string) string
		tag  string
	}{
		{"a heap of another size", func(s string) string { return strings.Replace(s, "nagaSamplerHeap[2048]", "nagaSamplerHeap[4096]", 1) }, tagShadersHLSLUnexpected},
		{"a heap in another register", func(s string) string {
			return strings.Replace(s, heap, "SamplerState nagaSamplerHeap[2048]: register(s1, space0);", 1)
		}, tagShadersHLSLUnexpected},
		{"a sampler line of another shape", func(s string) string {
			return strings.Replace(s, "static const SamplerState s0_ = nagaSamplerHeap[nagaGroup0SamplerIndexArray[5]];", "static const SamplerState s0_ = nagaSamplerHeap[nagaGroup0SamplerIndexArray[5] + 1];", 1)
		}, tagShadersHLSLUnexpected},
		{"a sampler that reads the other heap", func(s string) string {
			return strings.Replace(s, "SamplerComparisonState cs = nagaComparisonSamplerHeap", "SamplerComparisonState cs = nagaSamplerHeap", 1)
		}, tagShadersHLSLUnexpected},
		{"a use of the heap in a function", func(s string) string { return s + "\nfloat4 f() { return nagaSamplerHeap[0].x; }\n" }, tagShadersHLSLUnexpected},
		{"an index array that naga did not declare", func(s string) string {
			return strings.Replace(s, "StructuredBuffer<uint> nagaGroup1SamplerIndexArray : register(t1, space255);\n", "", 1)
		}, tagShadersHLSLUnexpected},
		{"a register with a space that is not a number", func(s string) string { return strings.Replace(s, "register(b2)", "register(b2, spaceX)", 1) }, tagShadersHLSLUnexpected},
		{"a register of another shape", func(s string) string { return strings.Replace(s, "register(b2)", "register(b2, space0, x)", 1) }, tagShadersHLSLUnexpected},
		{"a carriage return", func(s string) string { return strings.Replace(s, "\n", "\r\n", 1) }, tagShadersHLSLUnexpected},
		{"space255 left behind", func(s string) string { return s + "\nStructuredBuffer<uint> other : register(t9, space255);\n" }, tagShadersHLSLUnexpected},
		{"SV_InstanceID", func(s string) string {
			return strings.Replace(s, "VertexOutput_vs vs(uint vi : SV_VertexID)", "VertexOutput_vs vs(uint vi : SV_VertexID, uint ii : SV_InstanceID)", 1)
		}, tagShadersHLSLInstanceID},
	}
	if !strings.Contains(fixtureHLSL, "VertexOutput_vs vs(uint vi : SV_VertexID)") {
		t.Fatal("the fixture changed: fix the SV_InstanceID row")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.edit(fixtureHLSL)
			if in == fixtureHLSL {
				t.Fatal("the edit changed nothing")
			}
			_, err := rewriteHLSL(in)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute("rewriteHLSL."+tt.tag) {
				t.Errorf("got %s via %s, want %s via rewriteHLSL", appErr.Tag, appErr.Route(), tt.tag)
			}
		})
	}
}

func TestBindingRegister(t *testing.T) {
	tests := []struct {
		b     Binding
		class string
		space int
		index int
	}{
		{Binding{"u", 0, 2, KindUniform}, "b", 0, 2},
		{Binding{"t", 1, 3, KindTexture}, "t", 1, 3},
		{Binding{"s", 2, 0, KindSampler}, "s", 2, 0},
	}
	for _, tt := range tests {
		class, space, index := tt.b.Register()
		if class != tt.class || space != tt.space || index != tt.index {
			t.Errorf("%+v: register(%s%d, space%d), want register(%s%d, space%d)", tt.b, class, index, space, tt.class, tt.index, tt.space)
		}
	}
}

func TestCheckHLSL(t *testing.T) {
	const wgsl = `@group(0) @binding(2) var<uniform> u: U;
@group(1) @binding(3) var t: texture_2d<f32>;
@group(1) @binding(5) var s: sampler;
@vertex fn vs() -> @builtin(position) vec4<f32> { return vec4<f32>(u.x); }
@fragment fn fs() -> @location(0) vec4<f32> { return textureSample(t, s, vec2<f32>(0.0)); }`
	m, err := Parse(wgsl)
	if err != nil {
		t.Fatal(err)
	}
	const good = `cbuffer u : register(b2, space0) { U u; }
Texture2D<float4> t : register(t3, space1);
SamplerState s : register(s5, space1);
float4 vs() : SV_Position { return 0; }
float4 fs() : SV_Target0 { return 0; }
`
	if err := checkHLSL(good, m); err != nil {
		t.Fatalf("the good HLSL: %v", err)
	}
	tests := []struct{ name, hlsl string }{
		{"a binding that is not declared", strings.Replace(good, "SamplerState s : register(s5, space1);\n", "", 1)},
		{"a register in another space", strings.Replace(good, "register(t3, space1)", "register(t3, space0)", 1)},
		{"a register of another class", strings.Replace(good, "register(s5, space1)", "register(t5, space1)", 1)},
		{"a register that no binding has", good + "Texture2D<float4> extra : register(t9, space0);\n"},
		{"the same register twice", good + "Texture2D<float4> again : register(t3, space1);\n"},
		{"an entry point that is gone", strings.Replace(good, "float4 vs()", "float4 vs_renamed()", 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkHLSL(tt.hlsl, m)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagShadersHLSLMismatch || !appErr.HasRoute("checkHLSL.ShadersHLSLMismatch") {
				t.Errorf("got %v, want ShadersHLSLMismatch via checkHLSL", err)
			}
		})
	}
}

// fixtureShader is a WGSL module with a uniform in group 0, a texture and a sampler in group 1, and
// a vertex entry that reads the instance index; the uniform and the pair are read by the two stages.
const (
	samplerShader = `struct U { m: vec4<f32> }
@group(0) @binding(2) var<uniform> u: U;
@group(0) @binding(0) var t0: texture_2d<f32>;
@group(0) @binding(5) var s0: sampler;
@group(1) @binding(3) var t1: texture_2d<f32>;
@group(1) @binding(5) var s1: sampler;
struct VO { @builtin(position) p: vec4<f32>, @location(0) uv: vec2<f32> }
@vertex fn vs_main(@builtin(vertex_index) vi: u32) -> VO {
	var o: VO;
	o.p = u.m * f32(vi);
	o.uv = vec2<f32>(0.0);
	return o;
}
@fragment fn fs_main(i: VO) -> @location(0) vec4<f32> {
	return textureSample(t0, s0, i.uv) + textureSample(t1, s1, i.uv);
}
`
	instanceShader = `struct U { m: vec4<f32> }
@group(0) @binding(0) var<uniform> u: U;
struct VO { @builtin(position) p: vec4<f32> }
@vertex fn vs_main(@builtin(instance_index) ii: u32) -> VO {
	var o: VO;
	o.p = u.m * f32(ii);
	return o;
}
@fragment fn fs_main(i: VO) -> @location(0) vec4<f32> { return i.p; }
`
)

// withNaga returns the installed naga, or skips the test.
func withNaga(t *testing.T) Naga {
	t.Helper()
	n := Naga{}
	if _, err := n.Version(); err != nil {
		t.Skipf("naga is not installed (cargo install naga-cli --locked): %v", err)
	}
	return n
}

// TestFixtureIsWhatNagaWrites keeps fixtureHLSL honest: it is naga's output for fixtureWGSL when naga
// is the version the fixture was written with.
func TestFixtureIsWhatNagaWrites(t *testing.T) {
	n := withNaga(t)
	if v, _ := n.Version(); v != "30.0.1" {
		t.Skipf("naga %s is installed, and the fixture was written with naga 30.0.1", v)
	}
	dir := t.TempDir()
	in, out := filepath.Join(dir, "f.wgsl"), filepath.Join(dir, "f.hlsl")
	if err := os.WriteFile(in, []byte(fixtureWGSL), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := n.CompileHLSL(in, out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != fixtureHLSL {
		t.Errorf("naga writes another text than the fixture:\n%s", got)
	}
}

// TestGenerateHLSLWithNaga runs the real naga over a shader with two groups, samplers in both and every
// kind of binding, and checks the HLSL in the generated file and the mapping recorded for each binding.
func TestGenerateHLSLWithNaga(t *testing.T) {
	n := withNaga(t)
	out := t.TempDir()
	if err := Generate(writeShaders(t, map[string]string{"pair.wgsl": samplerShader}), out, n); err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join(out, generated))
	if err != nil {
		t.Fatal(err)
	}
	text := string(code)
	for _, want := range []string{
		"cbuffer u : register(b2, space0) { U u; }",
		"Texture2D<float4> t0_ : register(t0, space0);",
		"SamplerState s0_ : register(s5, space0);",
		"Texture2D<float4> t1_ : register(t3, space1);",
		"SamplerState s1_ : register(s5, space1);",
		"VertexOutput_vs_main vs_main(uint vi : SV_VertexID)",
		"HLSL: `",
		`{Name: "u", Group: 0, Binding: 2, Kind: BindingUniformBuffer, Register: ShaderRegister{Class: "b", Space: 0, Index: 2}}`,
		`{Name: "t1", Group: 1, Binding: 3, Kind: BindingSampledTexture, Register: ShaderRegister{Class: "t", Space: 1, Index: 3}}`,
		`{Name: "s1", Group: 1, Binding: 5, Kind: BindingSampler, Register: ShaderRegister{Class: "s", Space: 1, Index: 5}}`,
		"HLSL for shader model 5.1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the generated file lacks %q", want)
		}
	}
	for _, gone := range leftoverFragments {
		if strings.Contains(text, gone) {
			t.Errorf("the generated file has %q", gone)
		}
	}
}

// A shader that reads the instance index is refused: SV_InstanceID leaves out the first instance of
// the draw (SPEC §9.8), so the value is a uniform.
func TestGenerateRefusesTheInstanceIndex(t *testing.T) {
	n := withNaga(t)
	err := Generate(writeShaders(t, map[string]string{"inst.wgsl": instanceShader}), t.TempDir(), n)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagShadersHLSLInstanceID || !appErr.HasRoute("Generate->rewriteHLSL.ShadersHLSLInstanceID") {
		t.Fatalf("got %v, want ShadersHLSLInstanceID via Generate->rewriteHLSL", err)
	}
	if appErr.Metadata["file"] == nil {
		t.Error("the error does not name the file")
	}
}

// The errors of the HLSL pass reach the caller of Generate with the shader's file in the metadata.
func TestGenerateHLSLErrors(t *testing.T) {
	tests := []struct {
		name  string
		hlsl  func(string) string
		tag   string
		route string
	}{
		{"naga's sampler heap in a form that is not known", func(string) string {
			return "SamplerState nagaSamplerHeap[2048]: register(s0, space0);\nstatic const SamplerState s = nagaSamplerHeap[nagaGroup0SamplerIndexArray[0]+1];\n"
		}, tagShadersHLSLUnexpected, "Generate->rewriteHLSL.ShadersHLSLUnexpected"},
		{"a register that the WGSL does not have", func(s string) string { return s + "Texture2D<float4> extra : register(t9, space0);\n" }, tagShadersHLSLMismatch, "Generate->checkHLSL.ShadersHLSLMismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			naga := &fakeNaga{t: t, version: "1", hlslEdit: tt.hlsl}
			err := Generate(writeShaders(t, map[string]string{"a.wgsl": tinyShader}), t.TempDir(), naga)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Fatalf("got %v, want %s via %s", err, tt.tag, tt.route)
			}
			if appErr.Metadata["file"] == nil {
				t.Error("the error does not name the file")
			}
		})
	}
}

// nagaStyleHLSL writes, for a WGSL module the parser accepts, the HLSL in the shape naga 30 gives it:
// group 0 without a space, the other groups with one, and the sampler heap for the samplers.
func nagaStyleHLSL(t *testing.T, src string) string {
	t.Helper()
	m, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	space := func(class string, g, n int) string {
		if g == 0 {
			return "register(" + class + itoa(n) + ")"
		}
		return "register(" + class + itoa(n) + ", space" + itoa(g) + ")"
	}
	var samplers []Binding
	for _, bd := range m.Bindings {
		switch bd.Kind {
		case KindUniform:
			b.WriteString("cbuffer " + bd.Name + " : " + space("b", bd.Group, bd.Binding) + " { T " + bd.Name + "; }\n")
		case KindTexture:
			b.WriteString("Texture2D<float4> " + bd.Name + " : " + space("t", bd.Group, bd.Binding) + ";\n")
		case KindSampler:
			samplers = append(samplers, bd)
		}
	}
	if len(samplers) > 0 {
		b.WriteString("SamplerState nagaSamplerHeap[2048]: register(s0, space0);\n")
		b.WriteString("SamplerComparisonState nagaComparisonSamplerHeap[2048]: register(s0, space1);\n")
		done := map[int]bool{}
		for _, s := range samplers {
			if !done[s.Group] {
				done[s.Group] = true
				b.WriteString("StructuredBuffer<uint> nagaGroup" + itoa(s.Group) + "SamplerIndexArray : register(t" + itoa(s.Group) + ", space255);\n")
			}
			b.WriteString("static const SamplerState " + s.Name + " = nagaSamplerHeap[nagaGroup" + itoa(s.Group) + "SamplerIndexArray[" + itoa(s.Binding) + "]];\n")
		}
	}
	for _, e := range m.Entries {
		b.WriteString("float4 " + e.Name + "()\n{\n    return 0;\n}\n")
	}
	return b.String()
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestNagaCompileHLSLFails(t *testing.T) {
	n := withNaga(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "bad.wgsl")
	if err := os.WriteFile(in, []byte("this is not WGSL"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := n.CompileHLSL(in, filepath.Join(dir, "bad.hlsl"))
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagShadersNagaFailed || !appErr.HasRoute("Naga.CompileHLSL.ShadersNagaFailed") {
		t.Fatalf("got %v, want ShadersNagaFailed via Naga.CompileHLSL", err)
	}
	if appErr.Metadata["output"] == nil {
		t.Error("the error does not carry naga's message")
	}
}

func TestDropConstStructs(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"a struct local", "    const VertexOutput_vs_main vo = { a, b };", "    VertexOutput_vs_main vo = { a, b };"},
		{"a struct copy", "\tconst VertexOutput v = _e35;", "\tVertexOutput v = _e35;"},
		{"a float4 stays", "    const float4 _e32 = toClip(x);", "    const float4 _e32 = toClip(x);"},
		{"a matrix stays", "    const float4x4 m = n;", "    const float4x4 m = n;"},
		{"an int stays", "    const int i = 0;", "    const int i = 0;"},
		{"a uint2 stays", "    const uint2 i = 0;", "    const uint2 i = 0;"},
		{"a global static const stays", "static const int MAX = 4;", "static const int MAX = 4;"},
		{"a global struct const stays", "static const Params p = { 1 };", "static const Params p = { 1 };"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dropConstStructs(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
