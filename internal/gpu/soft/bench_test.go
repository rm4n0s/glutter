package soft_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
)

// BenchmarkRenderScene is the time of the software renderer for each scene of the proof
// registry: the goldens' renderer, and the reference of the GPU's parity tests. SPEC §15
// has its numbers for the framework's frames; this one tracks the rasterizer, which
// also makes the coverage tiles of paths and glyphs for the GPU renderer. Run it with
//
//	go test ./internal/gpu/soft -run xxx -bench RenderScene -benchmem
func BenchmarkRenderScene(b *testing.B) {
	for _, name := range proof.Names() {
		sc, err := proof.Lookup(name)
		if err != nil {
			b.Fatal(err)
		}
		scene := sc.Build()
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(sc.Width * sc.Height * 4))
			for i := 0; i < b.N; i++ {
				if _, err := soft.Render(scene, sc.Width, sc.Height); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
