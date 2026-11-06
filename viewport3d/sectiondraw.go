package viewport3d

import (
	"log/slog"
	"slices"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// sectionDraw is a triangle item of the frame that the section planes cut: the caps and the outlines are drawn from it.
type sectionDraw struct {
	mesh *Mesh
	g    *meshGPU
	mv   [16]float32
	// capped is whether the item gets a cap (a closed mesh that is opaque), in the color cap.
	capped bool
	cap    ui.Color
}

// record returns the offset in the uniform scratch of record n, after the frame's data, making the scratch big enough.
func (r *Renderer) record(n int) int {
	need := itemBase + (n+1)*uniformAlign
	if len(r.ubytes) < need {
		r.ubytes = slices.Grow(r.ubytes[:len(r.ubytes)], need-len(r.ubytes))[:need]
	}
	return itemBase + n*uniformAlign
}

// writeSectionRecords writes the records of the sections of the frame, from record first: one for each plane and each item that
// the planes cut, plane by plane. The record has the matrix of the item, the color of the cap, the color of the outline, and the
// index of the plane, the width of the outline and its pull towards the camera (section.wgsl).
func (r *Renderer) writeSectionRecords(first int, style *resolvedStyle) {
	planes := len(r.scene.planes)
	outline := style.section.outlineColor
	for i := 0; i < planes; i++ {
		for j := range r.sect {
			sd := &r.sect[j]
			off := r.record(first + i*len(r.sect) + j)
			putMat(r.ubytes, off, &sd.mv)
			putVec4(r.ubytes, off+64, float32(sd.cap.R), float32(sd.cap.G), float32(sd.cap.B), 1)
			putVec4(r.ubytes, off+80, float32(i), float32(style.section.outlineWidth), edgePull, 0)
			putVec4(r.ubytes, off+96, float32(outline.R), float32(outline.G), float32(outline.B), float32(outline.A))
		}
	}
}

// cornersOf makes the buffer of the corners of the triangles of a mesh, once. It reports whether the mesh has one.
func (r *Renderer) cornersOf(m *Mesh, g *meshGPU) bool {
	if g.corners != nil {
		return true
	}
	n := m.IndexCount()
	pos := make([]float32, 0, 3*n)
	for i := 0; i < n; i++ {
		v := m.index(i)
		pos = append(pos, m.data.Positions[3*v], m.data.Positions[3*v+1], m.data.Positions[3*v+2])
	}
	b, err := r.newBuffer("viewport3d corners", floatBytes(nil, pos), gpu.BufferUsageVertex)
	if err != nil {
		slog.Error("viewport3d: the corners of a mesh could not be uploaded", "err", err)
		return false
	}
	g.corners, g.triangles = b, n/3
	return true
}

// drawSections draws, in the pass of the frame, for each plane and each item the planes cut: the cap (when the item has one),
// then the outlines of the cuts. first is the first record of the sections.
func (r *Renderer) drawSections(pass *gpu.RenderPass, base pipeKey, first, planes int) {
	key := func(k pipeKind) *gpu.RenderPipeline {
		pk := base
		pk.kind = k
		return r.pipeline(pk)
	}
	count, capDraw, clear, outline := key(pipeCapCount), key(pipeCapDraw), key(pipeCapClear), key(pipeOutline)
	if capDraw != nil && count != nil && clear != nil {
		pass.SetStencilReference(0)
		for i := 0; i < planes; i++ {
			for j := range r.sect {
				sd := &r.sect[j]
				if !sd.capped {
					continue
				}
				rec := uint32(itemBase + (first+i*len(r.sect)+j)*uniformAlign)
				pass.SetPipeline(count)
				pass.SetBindGroup(1, r.sectionGroup, rec)
				pass.SetVertexBuffer(0, sd.g.pos, 0)
				if sd.g.index != nil {
					r.setIndices(pass, sd.g)
					pass.DrawIndexed(sd.g.count, 1, 0, 0, 0)
				} else {
					pass.Draw(sd.g.count, 1, 0, 0)
				}
				pass.SetPipeline(capDraw)
				pass.SetBindGroup(1, r.sectionGroup, rec)
				pass.Draw(3, 1, 0, 0)
				pass.SetPipeline(clear)
				pass.SetBindGroup(1, r.sectionGroup, rec)
				pass.Draw(3, 1, 0, 0)
				r.stats.CapDraws++
			}
		}
	}
	if outline == nil || !r.scene.rs.section.outline {
		return
	}
	pass.SetPipeline(outline)
	for i := 0; i < planes; i++ {
		for j := range r.sect {
			sd := &r.sect[j]
			if !r.cornersOf(sd.mesh, sd.g) || sd.g.triangles == 0 {
				continue
			}
			pass.SetBindGroup(1, r.sectionGroup, uint32(itemBase+(first+i*len(r.sect)+j)*uniformAlign))
			pass.SetVertexBuffer(0, sd.g.corners, 0)
			pass.SetVertexBuffer(1, r.corners, 0)
			pass.Draw(6, sd.g.triangles, 0, 0)
			r.stats.OutlineDraws++
		}
	}
}
