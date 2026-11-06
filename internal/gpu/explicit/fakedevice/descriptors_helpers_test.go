package fakedevice

import (
	"image"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

var image2x2 = image.Rect(0, 0, 2, 2)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func reflectEqual(a, b any) bool { return reflect.DeepEqual(a, b) }

// colorZero is a clear color.
var colorZero = ui.Color{}

// mustTexture makes a render target.
func mustTexture(t *testing.T, d *Device) explicit.Texture {
	t.Helper()
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	return tex
}

// mustPipeline makes a pipeline of computeShader() with the vertex entry point.
func mustPipeline(t *testing.T, d *Device, vertex string) explicit.Pipeline {
	t.Helper()
	p, err := d.NewPipeline(explicit.PipelineDesc{Shader: computeShader(), VertexEntry: vertex, FragmentEntry: "fs", Target: explicit.TextureFormatRGBA8Unorm})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func containsOp(ops []string, op string) bool { return slices.Contains(ops, op) }

// expectTag checks that err is a tagged error with the route.
func expectTag(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Fatalf("got %v, want %s via %s", err, tag, route)
	}
}
