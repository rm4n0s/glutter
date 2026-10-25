package d3d12

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestPresentRects(t *testing.T) {
	r := func(l, t, rt, b int32) d3d12Rect { return d3d12Rect{Left: l, Top: t, Right: rt, Bottom: b} }
	tests := []struct {
		name          string
		damage        []image.Rectangle
		width, height int
		want          []d3d12Rect
	}{
		{"nil is the whole window: no rectangles", nil, 100, 50, nil},
		{"one rectangle, from the top left", []image.Rectangle{image.Rect(10, 5, 30, 20)}, 100, 50, []d3d12Rect{r(10, 5, 30, 20)}},
		{"two rectangles keep their order", []image.Rectangle{image.Rect(0, 0, 5, 5), image.Rect(50, 20, 99, 49)}, 100, 50, []d3d12Rect{r(0, 0, 5, 5), r(50, 20, 99, 49)}},
		{"cut to the window", []image.Rectangle{image.Rect(-10, -10, 20, 20), image.Rect(90, 40, 200, 200)}, 100, 50, []d3d12Rect{r(0, 0, 20, 20), r(90, 40, 100, 50)}},
		{"the whole window is a rectangle", []image.Rectangle{image.Rect(0, 0, 100, 50)}, 100, 50, []d3d12Rect{r(0, 0, 100, 50)}},
		{"an empty rectangle is dropped", []image.Rectangle{image.Rect(5, 5, 5, 20), image.Rect(1, 1, 3, 3)}, 100, 50, []d3d12Rect{r(1, 1, 3, 3)}},
		{"outside the window is dropped", []image.Rectangle{image.Rect(100, 0, 120, 10), image.Rect(0, 50, 10, 60), image.Rect(2, 2, 4, 4)}, 100, 50, []d3d12Rect{r(2, 2, 4, 4)}},
		{"an upside-down rectangle is put right", []image.Rectangle{{Min: image.Pt(30, 20), Max: image.Pt(10, 5)}}, 100, 50, []d3d12Rect{r(10, 5, 30, 20)}},
		{"nothing changed is a pixel", []image.Rectangle{}, 100, 50, []d3d12Rect{r(0, 0, 1, 1)}},
		{"the renderer's pixel stays a pixel", []image.Rectangle{image.Rect(0, 0, 1, 1)}, 100, 50, []d3d12Rect{r(0, 0, 1, 1)}},
		{"everything outside is a pixel", []image.Rectangle{image.Rect(200, 200, 300, 300)}, 100, 50, []d3d12Rect{r(0, 0, 1, 1)}},
		{"a window with no area has none", []image.Rectangle{image.Rect(0, 0, 5, 5)}, 0, 50, nil},
		{"a window with no height has none", []image.Rectangle{image.Rect(0, 0, 5, 5)}, 100, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := presentRects(tt.damage, tt.width, tt.height); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("presentRects = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChooseSwapFormat(t *testing.T) {
	refuse := win32.EInvalidArg
	tests := []struct {
		name      string
		answers   map[uint32]win32.HRESULT // by DXGI format; a missing one is S_OK
		want      string
		wantTried []string // the formats attempt was called with
		wantOK    bool
		wantLast  win32.HRESULT
		refused   int
	}{
		{"BGRA when it is accepted", nil, "B8G8R8A8_UNORM", []string{"B8G8R8A8_UNORM"}, true, win32.SOK, 0},
		{"RGBA when BGRA is refused", map[uint32]win32.HRESULT{dxgiFormatBGRA8: refuse}, "R8G8B8A8_UNORM", []string{"B8G8R8A8_UNORM", "R8G8B8A8_UNORM"}, true, win32.SOK, 1},
		{"neither", map[uint32]win32.HRESULT{dxgiFormatBGRA8: refuse, dxgiFormatRGBA8: win32.EFail}, "", []string{"B8G8R8A8_UNORM", "R8G8B8A8_UNORM"}, false, win32.EFail, 2},
		{"a lost device stops the search", map[uint32]win32.HRESULT{dxgiFormatBGRA8: win32.DXGIErrorDeviceRemoved}, "", []string{"B8G8R8A8_UNORM"}, false, win32.DXGIErrorDeviceRemoved, 1},
		{"out of memory stops the search", map[uint32]win32.HRESULT{dxgiFormatBGRA8: win32.EOutOfMemory}, "", []string{"B8G8R8A8_UNORM"}, false, win32.EOutOfMemory, 1},
		{"a success status is a success", map[uint32]win32.HRESULT{dxgiFormatBGRA8: win32.HRESULT(1)}, "B8G8R8A8_UNORM", []string{"B8G8R8A8_UNORM"}, true, win32.HRESULT(1), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tried []string
			got, refused, last, ok := chooseSwapFormat(swapFormats, func(f swapFormat) win32.HRESULT {
				tried = append(tried, f.Name)
				return tt.answers[f.DXGI]
			})
			if ok != tt.wantOK || got.Name != tt.want || last != tt.wantLast || len(refused) != tt.refused {
				t.Errorf("chooseSwapFormat = %q, %v, %v, %v; want %q, %v, %v, %d refused", got.Name, refused, last, ok, tt.want, tt.wantOK, tt.wantLast, tt.refused)
			}
			if !reflect.DeepEqual(tried, tt.wantTried) {
				t.Errorf("tried %v, want %v", tried, tt.wantTried)
			}
			for _, line := range refused {
				if !strings.Contains(line, ": ") {
					t.Errorf("a refused entry says no answer: %q", line)
				}
			}
		})
	}
}

func TestChooseSwapFormatWithNoFormatsAcceptsNothing(t *testing.T) {
	_, refused, last, ok := chooseSwapFormat(nil, func(swapFormat) win32.HRESULT { t.Error("attempt was called"); return win32.SOK })
	if ok || len(refused) != 0 || last != win32.SOK {
		t.Errorf("chooseSwapFormat(nil) = %v, %v, %v", refused, last, ok)
	}
}

// The formats are in the order of SPEC §9.8, are never sRGB, and map to the texture formats the blit
// pipeline is made for.
func TestSwapFormatsAreTheSpecifiedOnes(t *testing.T) {
	if len(swapFormats) != 2 || swapFormats[0].DXGI != dxgiFormatBGRA8 || swapFormats[1].DXGI != dxgiFormatRGBA8 {
		t.Fatalf("swapFormats = %+v, want B8G8R8A8_UNORM then R8G8B8A8_UNORM", swapFormats)
	}
	for _, f := range swapFormats {
		if dxgiFormat(f.Texture) != f.DXGI {
			t.Errorf("%s: the texture format maps to %d, not %d", f.Name, dxgiFormat(f.Texture), f.DXGI)
		}
	}
	if swapFormats[0].Texture != explicit.TextureFormatBGRA8Unorm || swapFormats[1].Texture != explicit.TextureFormatRGBA8Unorm {
		t.Errorf("the texture formats are %v and %v", swapFormats[0].Texture, swapFormats[1].Texture)
	}
}

func TestIsDeviceFailure(t *testing.T) {
	tests := []struct {
		hr   win32.HRESULT
		want bool
	}{
		{win32.EOutOfMemory, true}, {win32.DXGIErrorDeviceRemoved, true}, {win32.DXGIErrorDeviceReset, true},
		{win32.DXGIErrorDeviceHung, true}, {win32.HRESULT(int32(-0x7785FFE0)), true},
		{win32.EInvalidArg, false}, {win32.EFail, false}, {win32.SOK, false},
	}
	for _, tt := range tests {
		if got := isDeviceFailure(tt.hr); got != tt.want {
			t.Errorf("isDeviceFailure(%v) = %v, want %v", tt.hr, got, tt.want)
		}
	}
}
