//go:build linux

package vulkan

import (
	"encoding/binary"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
)

// instructions lists the (opcode, words) of a SPIR-V module.
func instructions(t *testing.T, spirv []byte) [][]uint32 {
	t.Helper()
	words := make([]uint32, len(spirv)/4)
	for i := range words {
		words[i] = binary.LittleEndian.Uint32(spirv[i*4:])
	}
	var out [][]uint32
	for at := spvHeaderWords; at < len(words); {
		n := int(words[at] >> 16)
		if n == 0 || at+n > len(words) {
			t.Fatalf("a broken instruction at word %d", at)
		}
		out = append(out, words[at:at+n])
		at += n
	}
	return out
}

func TestWithPointSize(t *testing.T) {
	for _, name := range []string{"tri", "fill", "sample", "vformats"} {
		shader := map[string][]byte{"tri": devicetest.ShaderTri.SPIRV, "fill": devicetest.ShaderFill.SPIRV,
			"sample": devicetest.ShaderSample.SPIRV, "vformats": devicetest.ShaderVformats.SPIRV}[name]
		t.Run(name, func(t *testing.T) {
			patched, err := withPointSize(shader)
			if err != nil {
				t.Fatal(err)
			}
			before, after := instructions(t, shader), instructions(t, patched)
			var variable uint32
			var decorated, stores, returns, vertexEntries, interfaces int
			for _, in := range after {
				op := in[0] & 0xffff
				switch {
				case op == spvOpDecorate && in[2] == spvDecorationBuiltIn && in[3] == spvBuiltInPointSize:
					variable = in[1]
					decorated++
				}
			}
			if decorated != 1 {
				t.Fatalf("%d PointSize decorations, want 1", decorated)
			}
			for _, in := range after {
				op := in[0] & 0xffff
				switch {
				case op == spvOpStore && in[1] == variable:
					stores++
				case op == spvOpEntryPoint && in[1] == spvModelVertex:
					vertexEntries++
					if in[len(in)-1] == variable {
						interfaces++
					}
				}
			}
			for _, in := range before {
				if in[0]&0xffff == spvOpEntryPoint && in[1] == spvModelVertex {
					returns++
				}
			}
			if vertexEntries != returns || interfaces != vertexEntries {
				t.Errorf("%d vertex entry points before, %d after, %d with the variable in their interface", returns, vertexEntries, interfaces)
			}
			if stores < vertexEntries {
				t.Errorf("%d stores of the PointSize for %d vertex entry points", stores, vertexEntries)
			}
			if bound := binary.LittleEndian.Uint32(patched[12:]); bound <= binary.LittleEndian.Uint32(shader[12:]) {
				t.Errorf("the bound is %d, it must be larger than %d", bound, binary.LittleEndian.Uint32(shader[12:]))
			}
			if len(after) <= len(before) {
				t.Errorf("the module has %d instructions after, %d before", len(after), len(before))
			}
		})
	}
}

func TestWithPointSizeWithoutVertexEntryPoints(t *testing.T) {
	// The fragment-only module is the shader with the model of every entry point changed to fragment.
	spirv := append([]byte(nil), devicetest.ShaderTri.SPIRV...)
	for at := spvHeaderWords * 4; at < len(spirv); {
		w := binary.LittleEndian.Uint32(spirv[at:])
		if w&0xffff == spvOpEntryPoint {
			binary.LittleEndian.PutUint32(spirv[at+4:], 4) // fragment
		}
		at += int(w>>16) * 4
	}
	got, err := withPointSize(spirv)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(spirv) {
		t.Error("a module without a vertex entry point changed")
	}
}

func TestWithPointSizeErrors(t *testing.T) {
	good := devicetest.ShaderTri.SPIRV
	tests := []struct {
		name  string
		spirv []byte
	}{
		{"not a whole number of words", good[:len(good)-1]},
		{"too short", good[:8]},
		{"the wrong magic", append([]byte{1, 2, 3, 4}, good[4:]...)},
		{"an instruction past the end", func() []byte {
			b := append([]byte(nil), good...)
			binary.LittleEndian.PutUint32(b[spvHeaderWords*4:], 0xffff<<16|binary.LittleEndian.Uint32(b[spvHeaderWords*4:])&0xffff)
			return b
		}()},
		{"an instruction of no words", func() []byte {
			b := append([]byte(nil), good...)
			binary.LittleEndian.PutUint32(b[spvHeaderWords*4:], 0)
			return b
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := withPointSize(tt.spirv)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %v", err)
			}
			if !appErr.HasRoute("withPointSize.VulkanSPIRVInvalid") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
