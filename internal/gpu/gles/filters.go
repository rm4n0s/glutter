package gles

import (
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Image filters (SPEC §9.1) run as passes over a layer's pixels: each pass draws a
// rectangle over a pooled framebuffer and reads the framebuffer or texture before it
// (shaders/filter.*). The passes are planned in gpu/filters.go, which the explicit
// renderer shares.

type filterPass = gpu.FilterPass

const (
	minSigma        = gpu.MinSigma
	maxLowSigma     = gpu.MaxLowSigma
	maxKernelRadius = gpu.MaxKernelRadius
)

func filterPasses(f ui.ImageFilter, ctm affine2D, tile ui.TileMode, w, h int) ([]filterPass, bool) {
	return gpu.FilterPasses(f, ctm, tile, w, h)
}

func blurPasses(sx, sy float64, tile ui.TileMode, w, h int) []filterPass {
	return gpu.BlurPasses(sx, sy, tile, w, h)
}

// runPasses runs the passes over the w x h pixels of src, whose top left corner is
// the pixel origin of the surface, and returns the framebuffer that holds the result.
// With dst not nil the last pass draws into it; else the result is a pooled
// framebuffer that the caller releases. Nothing but the passes is drawn.
func (dr *drawer) runPasses(src *Texture, w, h int, origin [2]int, passes []filterPass, dst *Framebuffer) (*Framebuffer, error) {
	d := dr.r.ctx.dev
	gl := dr.r.ctx.gl
	prog, err := d.Program("filter")
	if err != nil {
		return nil, err
	}
	d.BindVertexArray(d.VertexArray("filter", func(uint32) {}))
	d.UseProgram(prog)
	dr.current = -1
	d.SetScissor(false, 0, 0, 0, 0)
	d.BlendFunc(glOne, glZero, glOne, glZero)
	prog.SetInt("uSrc", 0)
	if len(passes) == 0 {
		passes = []filterPass{{Kind: gpu.PassCopy, OutW: w, OutH: h}}
	}
	cur, curW, curH := src, w, h
	var owned, result *Framebuffer
	for i := range passes {
		ps := &passes[i]
		var out *Framebuffer
		if i == len(passes)-1 && dst != nil {
			out = dst
		} else if out, err = d.AcquireFramebuffer(ps.OutW, ps.OutH, false); err != nil {
			if owned != nil {
				d.ReleaseFramebuffer(owned)
			}
			return nil, err
		}
		d.BindFramebuffer(out)
		d.Viewport(0, 0, ps.OutW, ps.OutH)
		d.BindTexture(0, cur)
		prog.SetInt("uPass", int32(ps.Kind))
		prog.SetVec2("uSrcSize", float32(curW), float32(curH))
		prog.SetVec2("uOutSize", float32(ps.OutW), float32(ps.OutH))
		prog.SetInt("uAxis", ps.Axis)
		prog.SetInt("uTile", ps.Tile)
		switch ps.Kind {
		case gpu.PassBlur:
			prog.SetFloat("uSigma", ps.Sigma)
			prog.SetInt("uRadius", ps.Radius)
			prog.SetFloat("uNorm", ps.Norm)
		case gpu.PassMorph:
			prog.SetInt("uRadius", ps.Radius)
			prog.SetInt("uDilate", b2i(ps.Dilate))
		case gpu.PassDown, gpu.PassUp:
			prog.SetVec2("uFactor", float32(ps.Factor[0]), float32(ps.Factor[1]))
		case gpu.PassMatrix:
			prog.SetVec4("uMat0", ps.Inv[0], ps.Inv[1], ps.Inv[2], ps.Inv[3])
			prog.SetVec2("uMat1", ps.Inv[4], ps.Inv[5])
			prog.SetVec2("uRectOrigin", float32(origin[0]), float32(origin[1]))
			prog.SetInt("uQuality", ps.Quality)
		case gpu.PassColor:
			applySpec(prog, ps.Spec)
		}
		gl.drawArrays(glTriangleStrip, 0, 4)
		if owned != nil {
			d.ReleaseFramebuffer(owned)
			owned = nil
		}
		if out != dst {
			owned = out
		}
		cur, curW, curH = out.Texture(), ps.OutW, ps.OutH
		result = out
	}
	return result, nil
}
