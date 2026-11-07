package main

import (
	"fmt"
	"math"
)

// The roles of the nodes of the level are in their names, which the game reads: Ground, Platform, Wall and Pillar are solid,
// Hazard hurts, Coin is picked up and Spawn is where the character starts.

type block struct {
	name       string
	x, z       float64 // the middle of the block on the ground
	w, d       float64 // its width (x) and depth (z)
	bottom, hi float64 // its lowest and highest y
	material   int
}

// prism adds a regular prism with the axis along z, centred at the origin, as the mesh of a coin.
func (b *geometry) prism(sides int, radius, half float64) {
	for _, z := range []float64{half, -half} {
		centre := uint16(len(b.pos) / 3)
		b.pos = append(b.pos, 0, 0, float32(z))
		b.nrm = append(b.nrm, 0, 0, float32(math.Copysign(1, z)))
		for i := 0; i < sides; i++ {
			a := 2 * math.Pi * float64(i) / float64(sides)
			b.pos = append(b.pos, float32(radius*math.Cos(a)), float32(radius*math.Sin(a)), float32(z))
			b.nrm = append(b.nrm, 0, 0, float32(math.Copysign(1, z)))
		}
		for i := 0; i < sides; i++ {
			next := (i + 1) % sides
			if z > 0 {
				b.idx = append(b.idx, centre, centre+1+uint16(i), centre+1+uint16(next))
			} else {
				b.idx = append(b.idx, centre, centre+1+uint16(next), centre+1+uint16(i))
			}
		}
	}
	for i := 0; i < sides; i++ {
		a0, a1 := 2*math.Pi*float64(i)/float64(sides), 2*math.Pi*float64(i+1)/float64(sides)
		am := (a0 + a1) / 2
		nx, ny := float32(math.Cos(am)), float32(math.Sin(am))
		base := uint16(len(b.pos) / 3)
		for _, p := range [4][3]float64{
			{radius * math.Cos(a0), radius * math.Sin(a0), half}, {radius * math.Cos(a0), radius * math.Sin(a0), -half},
			{radius * math.Cos(a1), radius * math.Sin(a1), -half}, {radius * math.Cos(a1), radius * math.Sin(a1), half},
		} {
			b.pos = append(b.pos, float32(p[0]), float32(p[1]), float32(p[2]))
			b.nrm = append(b.nrm, nx, ny, 0)
		}
		b.idx = append(b.idx, base, base+1, base+2, base, base+2, base+3)
	}
}

// makeLevel writes the level: a ground, platforms to climb, walls, pillars, tiles of lava and coins, each a node of its own.
func makeLevel() []byte {
	g := &glb{}
	ground := g.material("grass", [4]float64{0.14, 0.32, 0.12, 1}, 0, 0.9, [3]float64{})
	stone := g.material("stone", [4]float64{0.38, 0.34, 0.30, 1}, 0, 0.8, [3]float64{})
	wall := g.material("wall", [4]float64{0.26, 0.24, 0.34, 1}, 0, 0.85, [3]float64{})
	lava := g.material("lava", [4]float64{0.75, 0.12, 0.03, 1}, 0, 0.5, [3]float64{0.55, 0.10, 0.02})
	gold := g.material("gold", [4]float64{1.0, 0.76, 0.18, 1}, 0.6, 0.35, [3]float64{0.35, 0.24, 0.02})

	var blocks []block
	add := func(name string, x, z, w, d, bottom, top float64, m int) {
		blocks = append(blocks, block{name: name, x: x, z: z, w: w, d: d, bottom: bottom, hi: top, material: m})
	}
	add("Ground", 0, 0, 42, 42, -1, 0, ground)
	// The walls around the ground.
	add("Wall.N", 0, 21.5, 44, 1, 0, 3, wall)
	add("Wall.S", 0, -21.5, 44, 1, 0, 3, wall)
	add("Wall.E", 21.5, 0, 1, 44, 0, 3, wall)
	add("Wall.W", -21.5, 0, 1, 44, 0, 3, wall)
	// Stepping stones to a platform in the left half; each is 0.7 higher than the last.
	add("Platform.1", -3, -10, 3, 3, 0, 0.7, stone)
	add("Platform.2", -7, -7, 3, 3, 0, 1.4, stone)
	add("Platform.3", -11, -3, 3, 3, 0, 2.1, stone)
	add("Platform.4", -7, 2, 5, 5, 0, 2.8, stone)
	// A bridge over the lava of the right half.
	add("Platform.5", 8, 5, 3, 3, 0, 0.9, stone)
	add("Platform.6", 12, 9, 3, 3, 0, 1.6, stone)
	add("Platform.7", 7, 13, 4, 4, 0, 2.2, stone)
	// A low wall to jump over, and pillars to run around.
	add("Platform.8", 0, 4, 6, 1, 0, 1.0, stone)
	add("Pillar.1", 13, -2, 1.4, 1.4, 0, 4, wall)
	add("Pillar.2", -14, 10, 1.4, 1.4, 0, 4, wall)
	add("Pillar.3", 15, -12, 1.4, 1.4, 0, 4, wall)
	add("Pillar.4", -2, 12, 1.4, 1.4, 0, 4, wall)
	// The lava: it is only 0.2 high, and it hurts to stand on it or to run into it.
	add("Hazard.1", 6, -8, 7, 2, 0, 0.2, lava)
	add("Hazard.2", 11, 0, 2, 8, 0, 0.2, lava)
	add("Hazard.3", -4, 15, 9, 2, 0, 0.2, lava)
	add("Hazard.4", 2, -1, 2, 2, 0, 0.2, lava)

	mesh := 0
	var nodes []int
	node := func(name string, translation [3]float64, meshIndex int) {
		n := map[string]any{"name": name, "translation": translation}
		if meshIndex >= 0 {
			n["mesh"] = meshIndex
		}
		g.Nodes = append(g.Nodes, n)
		nodes = append(nodes, len(g.Nodes)-1)
	}
	for _, b := range blocks {
		// The mesh holds the size, and the node the place: no node is scaled, so the normals stay what they are.
		var geo geometry
		geo.box(0, (b.hi-b.bottom)/2, 0, b.w/2, (b.hi-b.bottom)/2, b.d/2, 0, false)
		g.Meshes = append(g.Meshes, map[string]any{"name": b.name, "primitives": []any{g.primitive(&geo, b.material)}})
		node(b.name, [3]float64{b.x, b.bottom, b.z}, mesh)
		mesh++
	}
	var coin geometry
	coin.prism(10, 0.30, 0.04)
	g.Meshes = append(g.Meshes, map[string]any{"name": "Coin", "primitives": []any{g.primitive(&coin, gold)}})
	coinMesh := mesh
	// The coins: along the way, on the platforms, and over the lava.
	coins := [][3]float64{
		{0, 0.8, -12}, {3, 0.8, -6}, {-3, 1.5, -10}, {-7, 2.2, -7}, {-11, 2.9, -3}, {-8, 3.6, 1.5}, {-6, 3.6, 3},
		{8, 1.7, 5}, {12, 2.4, 9}, {7, 3.0, 13}, {6, 1.0, -8}, {-4, 1.0, 15},
	}
	for i, c := range coins {
		node(fmt.Sprintf("Coin.%d", i+1), c, coinMesh)
	}
	node("Spawn", [3]float64{0, 0, -16}, -1)
	return g.bytes(nodes)
}
