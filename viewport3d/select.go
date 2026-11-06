package viewport3d

import (
	"image"
	"log/slog"
	"math"
	"math/bits"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// SelectionStyle is how the selected items look (SceneStyle.Selection).
//
// Flutter: no counterpart
type SelectionStyle struct {
	// Color is the highlight color; the default is amber.
	Color foundation.Opt[ui.Color]
	// Tint is how much of the highlight color replaces the color of the material of a selected item, from 0 to 1; the default is
	// 0.35. Lines and points of a selected item are drawn in the highlight color.
	Tint foundation.Opt[float64]
	// OutlineWidth is the width in logical pixels of the outline around the selected items that have faces; the default is 2.
	OutlineWidth float64
	// NoOutline leaves out the outline.
	NoOutline bool
}

// maxOutlineRadius is the largest outline, in pixels of the target: the shader looks at the pixels around each one.
const maxOutlineRadius = 8

var defaultHighlight = ui.NewColor(0xFFFFA000)

// resolvedSelection is a SelectionStyle with its defaults filled in.
type resolvedSelection struct {
	color   ui.Color
	tint    float32
	width   float64
	outline bool
}

// resolve fills in the defaults of the style.
func (s SelectionStyle) resolve() resolvedSelection {
	r := resolvedSelection{color: s.Color.OrElse(defaultHighlight), tint: float32(s.Tint.OrElse(0.35)), width: s.OutlineWidth, outline: !s.NoOutline}
	if r.width <= 0 {
		r.width = 2
	}
	r.tint = min(max(r.tint, 0), 1)
	return r
}

// Selected says whether the item is selected: drawn with the highlight color and an outline (SceneStyle.Selection).
//
// Flutter: no counterpart
func (it *Item) Selected() bool { return it.selected }

// SetSelected selects or deselects the item.
//
// Flutter: no counterpart
func (it *Item) SetSelected(on bool) {
	if it.selected != on {
		it.selected = on
		it.changed(false)
	}
}

// Selection returns the selected items of the scene, in the order of the scene.
//
// Flutter: no counterpart
func (s *Scene) Selection() []*Item {
	var out []*Item
	for _, it := range s.items {
		if it.selected {
			out = append(out, it)
		}
	}
	return out
}

// Select makes the items the selection: they are selected and every other item of the scene is not. An item that is not in the scene
// is left out.
//
// Flutter: no counterpart
func (s *Scene) Select(items ...*Item) {
	want := make(map[*Item]bool, len(items))
	for _, it := range items {
		if it != nil && it.scene == s {
			want[it] = true
		}
	}
	changed := false
	for _, it := range s.items {
		if it.selected != want[it] {
			it.selected = want[it]
			changed = true
		}
	}
	if changed {
		s.NotifyListeners()
	}
}

// ClearSelection deselects every item.
//
// Flutter: no counterpart
func (s *Scene) ClearSelection() { s.Select() }

// selectionRect is the rectangle of pixels that the selected items cover, and the outline around them, or the whole frame when
// an item reaches behind the camera.
func (r *Renderer) selectionRect(cam *Camera, items []*Item, radius, w, h int, dpr float64) image.Rectangle {
	full := image.Rect(0, 0, w, h)
	rect := image.Rectangle{}
	for _, it := range items {
		b := it.Bounds()
		if b.IsEmpty() {
			continue
		}
		minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for i := 0; i < 8; i++ {
			o, ok := cam.Project(b.Corner(i))
			if !ok {
				return full
			}
			minX, maxX = math.Min(minX, o.Dx), math.Max(maxX, o.Dx)
			minY, maxY = math.Min(minY, o.Dy), math.Max(maxY, o.Dy)
		}
		pad := radius + 2
		r := image.Rect(int(math.Floor(minX*dpr))-pad, int(math.Floor(minY*dpr))-pad, int(math.Ceil(maxX*dpr))+pad, int(math.Ceil(maxY*dpr))+pad)
		rect = rect.Union(r)
	}
	return rect.Intersect(full)
}

// ensureMask makes the mask texture of the size, and the group that the outline reads it through, when the size changed.
func (r *Renderer) ensureMask(w, h int, selectPipe *gpu.RenderPipeline) bool {
	if r.mask != nil && r.maskW == w && r.maskH == h && r.maskGroup != nil {
		return true
	}
	r.releaseMask()
	tex, err := r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d selection mask", Size: gpu.Extent3D{Width: w, Height: h, DepthOrArrayLayers: 1},
		Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageTextureBinding})
	if err != nil {
		slog.Error("viewport3d: the selection mask could not be made", "err", err)
		return false
	}
	group, err := r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d selection mask", Layout: selectPipe.BindGroupLayout(1), Entries: []gpu.BindGroupEntry{
		{Binding: 0, Buffer: r.uniforms, Size: itemUniformSize},
		{Binding: 1, TextureView: tex.View()},
	}})
	if err != nil {
		tex.Release()
		slog.Error("viewport3d: the selection mask could not be made", "err", err)
		return false
	}
	r.mask, r.maskGroup, r.maskW, r.maskH = tex, group, w, h
	return true
}

// releaseMask frees the mask texture and its group.
func (r *Renderer) releaseMask() {
	if r.maskGroup != nil {
		r.maskGroup.Release()
	}
	if r.mask != nil {
		r.mask.Release()
	}
	r.mask, r.maskGroup, r.maskW, r.maskH = nil, nil, 0, 0
}

// drawMask draws the selected items white into the mask, before the frame's pass.
func (r *Renderer) drawMask(f *gpu.ViewFrame) {
	pass := f.Encoder.BeginRenderPass(gpu.RenderPassDesc{Label: "viewport3d selection mask", ColorAttachments: []gpu.RenderPassColorAttachment{{View: r.mask.View()}}})
	pass.SetBindGroup(0, r.frameGroup, 0)
	for i := range r.items {
		d := &r.items[i]
		if !d.selected {
			continue
		}
		k := d.pk
		k.kind, k.blend, k.samples, k.color, k.depth = pipeMask, false, 1, gpu.TextureFormatRGBA8Unorm, gpu.TextureFormatUnspecified
		pipe := r.pipeline(k)
		if pipe == nil {
			continue
		}
		pass.SetPipeline(pipe)
		r.bindMesh(pass, d.mesh)
		pass.SetVertexBuffer(1+bits.OnesCount8(d.mesh.variant), r.instances, int(d.instance)*instanceStride)
		if d.mesh.index != nil {
			pass.DrawIndexed(d.mesh.count, 1, 0, 0, 0)
		} else {
			pass.Draw(d.mesh.count, 1, 0, 0)
		}
	}
	pass.End()
}

// tintOf is the tint of the instance or the record of an item: the highlight color and how much of it, or nothing.
func tintOf(sel *resolvedSelection) [4]float32 {
	c := sel.color
	return [4]float32{float32(c.R), float32(c.G), float32(c.B), sel.tint}
}
