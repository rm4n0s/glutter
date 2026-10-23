package shaders

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []Entry
	}{
		{
			name: "a texture and a sampler read by the fragment stage only",
			src: `@group(0) @binding(0) var t: texture_2d<f32>;
@group(0) @binding(1) var s: sampler;
@vertex fn vs() -> @builtin(position) vec4<f32> { return vec4<f32>(0.0); }
@fragment fn fs() -> @location(0) vec4<f32> { return textureSample(t, s, vec2<f32>(0.0)); }`,
			want: []Entry{
				{Name: "vs", Stage: "Vertex"},
				{Name: "fs", Stage: "Fragment", Bindings: []Binding{
					{"t", 0, 0, KindTexture}, {"s", 0, 1, KindSampler}}},
			},
		},
		{
			name: "attributes in the other order, a space, and spaces in the type",
			src: `@binding(3) @group(1) var<uniform> u : Uniforms ;
@binding(0) @group(2) var   img: texture_2d < f32 >;
@fragment fn fs() -> @location(0) vec4<f32> { return vec4<f32>(u.x) + textureLoad(img, vec2<i32>(0), 0); }`,
			want: []Entry{{Name: "fs", Stage: "Fragment", Bindings: []Binding{
				{"u", 1, 3, KindUniform}, {"img", 2, 0, KindTexture}}}},
		},
		{
			name: "a binding read through a function that the entry point calls",
			src: `@group(0) @binding(0) var<uniform> frame: Frame;
@group(0) @binding(1) var<uniform> other: Frame;
fn inner() -> f32 { return frame.x; }
fn middle() -> f32 { return inner(); }
@vertex fn vs() -> @builtin(position) vec4<f32> { return vec4<f32>(middle()); }
@fragment fn fs() -> @location(0) vec4<f32> { return vec4<f32>(1.0); }`,
			want: []Entry{
				{Name: "vs", Stage: "Vertex", Bindings: []Binding{{"frame", 0, 0, KindUniform}}},
				{Name: "fs", Stage: "Fragment"},
			},
		},
		{
			name: "a name in a comment is not a use, and a longer name is not a match",
			src: `@group(0) @binding(0) var<uniform> frame: Frame;
// frame is not used here: /* frame */
@vertex fn vs() -> @builtin(position) vec4<f32> { let frames = 1.0; return vec4<f32>(frames); }`,
			want: []Entry{{Name: "vs", Stage: "Vertex"}},
		},
		{
			name: "nested braces in the body",
			src: `@group(0) @binding(0) var<uniform> u: U;
@fragment fn fs() -> @location(0) vec4<f32> { if (true) { for (var i = 0; i < 2; i++) { let a = u.x; } } return vec4<f32>(0.0); }`,
			want: []Entry{{Name: "fs", Stage: "Fragment", Bindings: []Binding{{"u", 0, 0, KindUniform}}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := Parse(tt.src)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(m.Entries, tt.want) {
				t.Errorf("entries\n got %+v\nwant %+v", m.Entries, tt.want)
			}
		})
	}
}

func TestParseSortsBindings(t *testing.T) {
	m, err := Parse(`@group(1) @binding(0) var a: sampler;
@group(0) @binding(2) var b: sampler;
@group(0) @binding(1) var c: sampler;
@fragment fn fs() { _ = a; _ = b; _ = c; }`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, b := range m.Entries[0].Bindings {
		got = append(got, b.Name)
	}
	if !reflect.DeepEqual(got, []string{"c", "b", "a"}) {
		t.Errorf("order %v, want by group then binding: c b a", got)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		tag   string
		route string
	}{
		{"a storage buffer", "@group(0) @binding(0) var<storage, read> b: array<f32>;\n@vertex fn vs() { _ = b; }", "ShadersUnsupportedBinding", "Parse.ShadersUnsupportedBinding"},
		{"a texture of another type", "@group(0) @binding(0) var t: texture_cube<f32>;\n@vertex fn vs() { _ = t; }", "ShadersUnsupportedBinding", "Parse.ShadersUnsupportedBinding"},
		{"a comparison sampler", "@group(0) @binding(0) var s: sampler_comparison;\n@vertex fn vs() { _ = s; }", "ShadersUnsupportedBinding", "Parse.ShadersUnsupportedBinding"},
		{"a compute entry point", "@compute @workgroup_size(1) fn cs() {}", "ShadersComputeEntry", "Parse.ShadersComputeEntry"},
		{"no entry point", "fn helper() {}", "ShadersNoEntry", "Parse.ShadersNoEntry"},
		{"two variables in one slot", "@group(0) @binding(0) var a: sampler;\n@group(0) @binding(0) var b: sampler;\n@vertex fn vs() {}", "ShadersBindingTwice", "Parse.ShadersBindingTwice"},
		{"a body that is not closed", "@vertex fn vs() { if (true) {", "ShadersUnbalancedBraces", "Parse->functionBodies.ShadersUnbalancedBraces"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.src)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || (tt.route != "" && !appErr.HasRoute(tt.route)) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
		})
	}
}

// fakeNaga is a Compiler that writes the text of the WGSL file as its "SPIR-V".
type fakeNaga struct {
	version string
	failOn  string
	calls   []string
}

func (f *fakeNaga) Version() (string, error) { return f.version, nil }

func (f *fakeNaga) Compile(wgsl, spirv string) error {
	f.calls = append(f.calls, filepath.Base(wgsl))
	if f.failOn != "" && strings.HasSuffix(wgsl, f.failOn) {
		return errors.New(tagShadersNagaFailed, "invalid shader", "file", wgsl)
	}
	src, err := os.ReadFile(wgsl)
	if err != nil {
		return err
	}
	return os.WriteFile(spirv, append([]byte("SPV:"), src...), 0o644)
}

func writeShaders(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, shadersDir), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, shadersDir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const tinyShader = `@group(0) @binding(0) var<uniform> u: U;
@vertex fn vs_main() -> @builtin(position) vec4<f32> { return vec4<f32>(u.x); }
@fragment fn fs_main() -> @location(0) vec4<f32> { return vec4<f32>(1.0); }
`

func TestGenerate(t *testing.T) {
	src := writeShaders(t, map[string]string{"blur_pass.wgsl": tinyShader, "solid.wgsl": tinyShader, "notes.txt": "ignored"})
	out := t.TempDir()
	naga := &fakeNaga{version: "9.8.7"}
	if err := Generate(src, out, naga); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(naga.calls, []string{"blur_pass.wgsl", "solid.wgsl"}) {
		t.Errorf("naga ran on %v, want each shader once, in order", naga.calls)
	}
	for _, name := range []string{"blur_pass", "solid"} {
		spv, err := os.ReadFile(filepath.Join(out, shadersDir, name+".spv"))
		if err != nil || !bytes.HasPrefix(spv, []byte("SPV:")) {
			t.Errorf("%s.spv: %v, %q", name, err, spv)
		}
	}
	code, err := os.ReadFile(filepath.Join(out, generated))
	if err != nil {
		t.Fatal(err)
	}
	text := string(code)
	if _, err := parser.ParseFile(token.NewFileSet(), "shaders_gen.go", code, parser.ParseComments); err != nil {
		t.Fatalf("the generated file does not parse: %v\n%s", err, text)
	}
	for _, want := range []string{
		"// Code generated by glutter-gen shaders; DO NOT EDIT.",
		"naga 9.8.7",
		"//go:embed shaders/blur_pass.spv",
		"var ShaderBlurPass = &Shader{",
		"var ShaderSolid = &Shader{",
		`{Name: "u", Group: 0, Binding: 0, Kind: BindingUniformBuffer}`,
		`{Name: "fs_main", Stage: ShaderStageFragment, Bindings: []ShaderBinding{}}`,
		"// Flutter: no counterpart",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the generated file lacks %q", want)
		}
	}
	// The same input gives the same files.
	out2 := t.TempDir()
	if err := Generate(src, out2, &fakeNaga{version: "9.8.7"}); err != nil {
		t.Fatal(err)
	}
	code2, _ := os.ReadFile(filepath.Join(out2, generated))
	if !bytes.Equal(code, code2) {
		t.Error("two runs made different files")
	}
}

func TestGenerateErrors(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		naga  *fakeNaga
		tag   string
		route string
	}{
		{"no shaders", map[string]string{}, &fakeNaga{version: "1"}, "ShadersNoShaders", "Generate.ShadersNoShaders"},
		{"naga rejects a shader", map[string]string{"bad.wgsl": tinyShader}, &fakeNaga{version: "1", failOn: "bad.wgsl"}, "ShadersNagaFailed", "ShadersNagaFailed"},
		{"a shader the parser rejects", map[string]string{"bad.wgsl": "fn helper() {}"}, &fakeNaga{version: "1"}, "ShadersNoEntry", "Generate->Parse.ShadersNoEntry"},
		{"a backquote in the WGSL", map[string]string{"bad.wgsl": tinyShader + "// a `quote`\n"}, &fakeNaga{version: "1"}, "ShadersBacktick", "Generate->render.ShadersBacktick"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Generate(writeShaders(t, tt.files), t.TempDir(), tt.naga)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
			if tt.name == "a shader the parser rejects" && appErr.Metadata["file"] == nil {
				t.Error("the error does not name the file")
			}
		})
	}
}

func TestNagaMissing(t *testing.T) {
	_, err := Naga{Path: "/nonexistent-glutter-dir/naga"}.Version()
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagShadersNagaMissing || !appErr.HasRoute("Naga.Version.ShadersNagaMissing") {
		t.Fatalf("got %v, want ShadersNagaMissing via Naga.Version", err)
	}
	if appErr.Metadata["install"] != "cargo install naga-cli --locked" {
		t.Errorf("the error does not say how to install naga: %v", appErr.Metadata)
	}
}

func TestRunBadFlags(t *testing.T) {
	err := Run([]string{"-nope"})
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagShadersBadFlags || !appErr.HasRoute("Run.ShadersBadFlags") {
		t.Errorf("got %v, want ShadersBadFlags via Run", err)
	}
}

func TestRemoveStale(t *testing.T) {
	dir := writeShaders(t, map[string]string{"keep.wgsl": tinyShader, "keep.spv": "x", "gone.spv": "x"})
	if err := removeStale(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, shadersDir, "gone.spv")); err == nil {
		t.Error("the .spv of a shader that is gone was kept")
	}
	if _, err := os.Stat(filepath.Join(dir, shadersDir, "keep.spv")); err != nil {
		t.Error("the .spv of a shader that exists was removed")
	}
}

func TestGoName(t *testing.T) {
	for in, want := range map[string]string{"blit": "Blit", "blur_pass": "BlurPass", "a-b.c": "ABC"} {
		if got := goName(in); got != want {
			t.Errorf("goName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCheckedInFilesAreUpToDate runs naga on the shaders of internal/gpu/explicit
// into a temporary directory and compares what it makes with the checked-in files.
// It skips without naga, and when naga is not the version that made them, because the
// SPIR-V of another version can differ; builds always use the checked-in files.
func TestCheckedInFilesAreUpToDate(t *testing.T) {
	src := filepath.Join("..", "..", "..", "..", "internal", "gpu", "explicit")
	naga := Naga{}
	version, err := naga.Version()
	if err != nil {
		t.Skipf("naga is not installed (cargo install naga-cli --locked): %v", err)
	}
	checkedIn, err := os.ReadFile(filepath.Join(src, generated))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`with naga (\S+) `).FindSubmatch(checkedIn)
	if m == nil {
		t.Fatal("the header of shaders_gen.go does not name the version of naga")
	}
	if string(m[1]) != version {
		t.Skipf("naga %s is installed, and the files were made with naga %s", version, m[1])
	}
	out := t.TempDir()
	if err := Generate(src, out, naga); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(out, "*", "*.spv"))
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, filepath.Join(out, generated))
	for _, got := range files {
		rel, _ := filepath.Rel(out, got)
		want, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Errorf("%s is not checked in: %v", rel, err)
			continue
		}
		have, err := os.ReadFile(got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(have, want) {
			t.Errorf("%s is out of date: run go generate ./internal/gpu/explicit", rel)
		}
	}
}

func TestIncludesAreExpanded(t *testing.T) {
	src := writeShaders(t, map[string]string{
		"main.wgsl":       "//#include \"common.wgsl.inc\"\n" + tinyShader,
		"common.wgsl.inc": "// shared\nstruct U { x: f32 }\n//#include \"deeper.wgsl.inc\"\n",
		"deeper.wgsl.inc": "const K: f32 = 2.0;\n",
	})
	out := t.TempDir()
	naga := &fakeNaga{version: "1.0.0"}
	if err := Generate(src, out, naga); err != nil {
		t.Fatal(err)
	}
	// naga compiled the expanded text, once, for the shader and not for the includes.
	spv, err := os.ReadFile(filepath.Join(out, shadersDir, "main.spv"))
	if err != nil || !bytes.Contains(spv, []byte("struct U { x: f32 }\nconst K: f32 = 2.0;\n@group")) || bytes.Contains(spv, []byte("#include")) {
		t.Errorf("the compiled text: %v\n%s", err, spv)
	}
	if !reflect.DeepEqual(naga.calls, []string{"main.wgsl"}) {
		t.Errorf("naga ran on %v", naga.calls)
	}
	code, _ := os.ReadFile(filepath.Join(out, generated))
	if !strings.Contains(string(code), "const K: f32 = 2.0;") || strings.Contains(string(code), "#include") {
		t.Errorf("the generated file does not hold the expanded text:\n%s", code)
	}
	// The bindings of an included file count for the entry points that use them.
	if !strings.Contains(string(code), `{Name: "u", Group: 0, Binding: 0, Kind: BindingUniformBuffer}`) {
		t.Error("the binding is missing")
	}
}

func TestIncludeErrors(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		tag   string
	}{
		{"a file that is not there", map[string]string{"a.wgsl": "//#include \"missing.inc\"\n" + tinyShader}, "ShadersIncludeMissing"},
		{"a file that includes itself", map[string]string{"a.wgsl": "//#include \"b.inc\"\n" + tinyShader, "b.inc": "//#include \"b.inc\"\n"}, "ShadersIncludeCycle"},
		{"two files that include each other", map[string]string{"a.wgsl": "//#include \"b.inc\"\n" + tinyShader, "b.inc": "//#include \"c.inc\"\n", "c.inc": "//#include \"b.inc\"\n"}, "ShadersIncludeCycle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Generate(writeShaders(t, tt.files), t.TempDir(), &fakeNaga{version: "1"})
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute("expand."+tt.tag) {
				t.Errorf("got %v, want %s through expand", err, tt.tag)
			}
		})
	}
}
