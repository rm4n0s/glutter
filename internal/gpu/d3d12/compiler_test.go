package d3d12

import (
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeBackend is a Backend that counts its calls and returns what it is given.
type fakeBackend struct {
	version  string
	dxbc     []byte
	messages string
	hresult  int32
	calls    int
	last     struct {
		hlsl, entry, target string
		flags               uint32
	}
}

func (f *fakeBackend) Version() string { return f.version }

func (f *fakeBackend) Compile(hlsl, entry, target string, flags uint32) ([]byte, string, int32) {
	f.calls++
	f.last.hlsl, f.last.entry, f.last.target, f.last.flags = hlsl, entry, target, flags
	return f.dxbc, f.messages, f.hresult
}

// memCache is a BlobCache in memory.
type memCache struct{ files map[string][]byte }

func (m *memCache) Load(key string) ([]byte, bool) { d, ok := m.files[key]; return d, ok }
func (m *memCache) Store(key string, data []byte)  { m.files[key] = data }

func TestCompilePassesItsArgumentsToTheBackend(t *testing.T) {
	be := &fakeBackend{version: "1.0", dxbc: dxbcOf(64, 64), messages: "a warning"}
	res, err := NewCompiler(be, nil).Compile("float4 f() { return 0; }", "f", TargetVertex, CompileDebug)
	if err != nil {
		t.Fatal(err)
	}
	if be.last.hlsl != "float4 f() { return 0; }" || be.last.entry != "f" || be.last.target != TargetVertex || be.last.flags != CompileDebug {
		t.Errorf("the backend got %+v", be.last)
	}
	if res.Messages != "a warning" {
		t.Errorf("messages %q, want the compiler's warning", res.Messages)
	}
}

func TestCompileFailure(t *testing.T) {
	const message = "shader.hlsl(3,5): error X3004: undeclared identifier 'x'"
	tests := []struct {
		name string
		be   *fakeBackend
	}{
		{"a failed HRESULT", &fakeBackend{version: "1.0", messages: message, hresult: -2005270523}},
		{"success with no bytecode", &fakeBackend{version: "1.0", messages: message}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &memCache{files: map[string][]byte{}}
			_, err := NewCompiler(tt.be, cache).Compile("broken", "f", TargetPixel, 1)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != TagD3D12ShaderCompileFailed || !appErr.HasRoute("Compiler.Compile.D3D12ShaderCompileFailed") {
				t.Errorf("got %s via %s, want D3D12ShaderCompileFailed via Compiler.Compile", appErr.Tag, appErr.Route())
			}
			if got, _ := appErr.Metadata["messages"].(string); !strings.Contains(got, "(3,5)") {
				t.Errorf("messages %q do not carry the compiler's line and column", got)
			}
			if appErr.Metadata["entry"] != "f" || appErr.Metadata["target"] != TargetPixel {
				t.Errorf("metadata %v lack the entry point and the target", appErr.Metadata)
			}
			if len(cache.files) != 0 {
				t.Error("a failed compile was cached")
			}
		})
	}
}

func TestHresultText(t *testing.T) {
	if got := hresultText(-2147467259); got != "0x80004005" {
		t.Errorf("hresultText = %q", got)
	}
}
