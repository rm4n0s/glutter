package layout

import (
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// Cluster is a cluster of a laid-out paragraph with the glyphs that draw it, for a caller that places the glyphs
// itself (SVG text, which positions each character).
//
// Flutter: no counterpart
type Cluster struct {
	// Start and End are the byte range of the text that the cluster covers.
	Start, End int
	// X0 and X1 are the horizontal extent of the cluster's advance, in paragraph coordinates, and Baseline is the y of the
	// baseline of its line.
	X0, X1, Baseline float64
	// RTL says that the cluster is in a right-to-left run.
	RTL bool
	// Size is the font size of the cluster, and Ascent and Descent the font's metrics at that size (both positive).
	Size, Ascent, Descent float64
	// Underline, Strike and Overline are the distances of the centers of the decoration lines below the baseline
	// (negative is above it), and UnderlineThickness and StrikeThickness their thicknesses.
	Underline, Strike, Overline         float64
	UnderlineThickness, StrikeThickness float64
	// Glyphs are what draws the cluster.
	Glyphs []ClusterGlyph
}

// ClusterGlyph is one glyph of a Cluster.
//
// Flutter: no counterpart
type ClusterGlyph struct {
	// Path is the outline, with its origin at the pen position and y growing down; nil for a glyph with no outline or a
	// color glyph.
	Path *ui.Path
	// Picture is a color glyph (an emoji) drawn with Matrix, or nil.
	Picture *ui.Picture
	Matrix  *ui.Matrix4
	// Origin is where the origin of Path is, in paragraph coordinates.
	Origin ui.Offset
}

// Clusterer is implemented by the paragraphs that this package lays out: the clusters of the text in logical order.
//
// Flutter: no counterpart
type Clusterer interface {
	// Clusters returns the clusters of the laid-out paragraph in the order of the text.
	Clusters() []Cluster
}

var _ Clusterer = (*paragraph)(nil)

// Clusters implements Clusterer.
//
// Flutter: no counterpart
func (p *paragraph) Clusters() []Cluster {
	p.ensureLaidOut()
	var out []Cluster
	for _, l := range p.lines {
		for ri := range l.runs {
			pr := &l.runs[ri]
			if pr.ellipsis || pr.run.ph != nil || pr.run.face == nil {
				continue
			}
			face := pr.run.face
			st := p.styles[pr.run.style]
			for i := 0; i < len(pr.glyphs); {
				g0 := pr.glyphs[i]
				x0, x1 := g0.pen, g0.pen+g0.width
				j := i + 1
				for j < len(pr.glyphs) && pr.glyphs[j].cluster == g0.cluster {
					x0, x1 = min(x0, pr.glyphs[j].pen), max(x1, pr.glyphs[j].pen+pr.glyphs[j].width)
					j++
				}
				count := max(1, p.clusterRunes(pr.run, g0.cluster))
				count = min(count, len(p.runes)-g0.cluster)
				c := Cluster{
					Start: p.byteOf(g0.cluster), End: p.byteOf(g0.cluster + count),
					X0: l.left + pr.x + x0, X1: l.left + pr.x + x1, Baseline: l.baseline, RTL: pr.run.rtl, Size: pr.run.size,
					Ascent: pr.run.ascent, Descent: pr.run.descent,
				}
				c.Underline, c.Strike, c.Overline, c.UnderlineThickness, c.StrikeThickness = decorationMetrics(face, pr.run.size, pr.run.ascent)
				for _, g := range pr.glyphs[i:j] {
					if p.drawsNothing(pr, &g) {
						continue
					}
					origin := ui.Offset{Dx: l.left + pr.x + g.x, Dy: l.baseline + g.y}
					if pic := p.eng.colorGlyph(face, g.id, st.Color, pr.run.size); pic != nil {
						s := pr.run.size / float64(face.Upem())
						c.Glyphs = append(c.Glyphs, ClusterGlyph{Picture: pic, Matrix: ui.NewMatrix4Diagonal3Values(float32(s), float32(-s), 1), Origin: origin})
						continue
					}
					if path := p.eng.glyphPath(face, g.id, pr.run.size); path != nil {
						c.Glyphs = append(c.Glyphs, ClusterGlyph{Path: path, Origin: origin})
					}
				}
				out = append(out, c)
				i = j
			}
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Start < out[b].Start })
	return out
}
