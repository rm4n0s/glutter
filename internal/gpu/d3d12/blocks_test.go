package d3d12

import "testing"

func TestBlockAllocator(t *testing.T) {
	type step struct {
		n                    int
		block, offset        int
		grew, ok             bool
		resetBefore          bool
		wantBlocks, wantUsed int
	}
	tests := []struct {
		name  string
		size  int
		steps []step
	}{
		{"the first range makes the first block", 8, []step{
			{n: 2, block: 0, offset: 0, grew: true, ok: true, wantBlocks: 1, wantUsed: 2},
			{n: 3, block: 0, offset: 2, ok: true, wantBlocks: 1, wantUsed: 5},
		}},
		{"a range that does not fit goes to a new block", 8, []step{
			{n: 6, block: 0, offset: 0, grew: true, ok: true, wantBlocks: 1, wantUsed: 6},
			{n: 4, block: 1, offset: 0, grew: true, ok: true, wantBlocks: 2, wantUsed: 10},
			{n: 4, block: 1, offset: 4, ok: true, wantBlocks: 2, wantUsed: 14},
			{n: 1, block: 2, offset: 0, grew: true, ok: true, wantBlocks: 3, wantUsed: 15},
		}},
		{"a range of exactly the block fits", 4, []step{
			{n: 4, block: 0, offset: 0, grew: true, ok: true, wantBlocks: 1, wantUsed: 4},
			{n: 4, block: 1, offset: 0, grew: true, ok: true, wantBlocks: 2, wantUsed: 8},
		}},
		{"a range larger than a block is refused", 4, []step{
			{n: 5, ok: false, wantBlocks: 0, wantUsed: 0},
			{n: 1, block: 0, offset: 0, grew: true, ok: true, wantBlocks: 1, wantUsed: 1},
		}},
		{"reset reuses the blocks without making any", 4, []step{
			{n: 3, block: 0, offset: 0, grew: true, ok: true, wantBlocks: 1, wantUsed: 3},
			{n: 3, block: 1, offset: 0, grew: true, ok: true, wantBlocks: 2, wantUsed: 6},
			{n: 3, resetBefore: true, block: 0, offset: 0, ok: true, wantBlocks: 2, wantUsed: 3},
			{n: 3, block: 1, offset: 0, ok: true, wantBlocks: 2, wantUsed: 6},
			{n: 3, block: 2, offset: 0, grew: true, ok: true, wantBlocks: 3, wantUsed: 9},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBlockAlloc(tt.size, 0)
			for i, s := range tt.steps {
				if s.resetBefore {
					b.reset()
				}
				block, offset, grew, ok := b.alloc(s.n)
				if ok != s.ok || (ok && (block != s.block || offset != s.offset || grew != s.grew)) {
					t.Fatalf("step %d: alloc(%d) = block %d offset %d grew %v ok %v; want %d %d %v %v", i, s.n, block, offset, grew, ok, s.block, s.offset, s.grew, s.ok)
				}
				if b.blocks() != s.wantBlocks || b.used() != s.wantUsed {
					t.Fatalf("step %d: %d blocks, %d used; want %d, %d", i, b.blocks(), b.used(), s.wantBlocks, s.wantUsed)
				}
			}
		})
	}
}

func TestBlockRollback(t *testing.T) {
	b := newBlockAlloc(4, 0)
	b.alloc(3)
	if _, _, grew, _ := b.alloc(3); !grew {
		t.Fatal("the second block should be new")
	}
	b.rollback() // the device could not make its heap
	if b.blocks() != 1 || b.cur != 0 {
		t.Fatalf("after rollback: %d blocks, current %d", b.blocks(), b.cur)
	}
	if block, off, grew, _ := b.alloc(3); block != 1 || off != 0 || !grew {
		t.Errorf("asking again: block %d offset %d grew %v", block, off, grew)
	}
}

func TestBlockAllocatorCapAndSharedTables(t *testing.T) {
	b := newBlockAlloc(4, 3)
	for i := 0; i < 3; i++ {
		if _, _, _, ok := b.alloc(4); !ok {
			t.Fatalf("block %d refused", i)
		}
	}
	if _, _, _, ok := b.alloc(1); ok {
		t.Error("a fourth block was made, the cap is 3")
	}
	if b.blocks() != 3 {
		t.Errorf("%d blocks after the refusal", b.blocks())
	}
	b.reset()
	if _, _, grew, ok := b.alloc(4); !ok || grew {
		t.Errorf("after reset the first block is reused: grew %v ok %v", grew, ok)
	}

	// A table is found only in the block being filled, and reset forgets it.
	c := newBlockAlloc(4, 0)
	if _, _, ok := c.lookup("a"); ok {
		t.Error("a table before any block")
	}
	block, off, _, _ := c.alloc(2)
	c.remember("a", block, off)
	if bl, of, ok := c.lookup("a"); !ok || bl != 0 || of != 0 {
		t.Errorf("lookup = %d %d %v", bl, of, ok)
	}
	c.alloc(3) // goes to block 1
	if _, _, ok := c.lookup("a"); ok {
		t.Error("a table of an earlier block was found")
	}
	c.reset()
	if _, _, ok := c.lookup("a"); ok {
		t.Error("a table survived reset")
	}
}

func TestTableKey(t *testing.T) {
	if tableKey([]uintptr{1, 2}) == tableKey([]uintptr{2, 1}) || tableKey([]uintptr{1}) == tableKey([]uintptr{1, 0}) || tableKey([]uintptr{0x100}) == tableKey([]uintptr{1}) {
		t.Error("different tables have one key")
	}
	same := []uintptr{5, 6}
	if tableKey([]uintptr{5, 6}) != tableKey(same) {
		t.Error("one table has two keys")
	}
}

// fakeGroup is a bound group of the model: the tables it needs and where they are.
type fakeGroup struct {
	views, samplers int
	vKey, sKey      string
	vBlock, sBlock  int
}

// slotModel is the heaps of one frame slot, as the device and the encoder use them.
type slotModel struct {
	v, s  blockAlloc
	grown int // blocks made
}

func (m *slotModel) place(g *fakeGroup) error {
	g.vBlock, g.sBlock = -1, -1
	if g.views > 0 {
		block, _, ok := m.v.lookup(g.vKey)
		if !ok {
			var off int
			var grew bool
			block, off, grew, ok = m.v.alloc(g.views)
			if !ok {
				return errNoRoom
			}
			if grew {
				m.grown++
			}
			m.v.remember(g.vKey, block, off)
		}
		g.vBlock = block
	}
	if g.samplers > 0 {
		block, _, ok := m.s.lookup(g.sKey)
		if !ok {
			var off int
			var grew bool
			block, off, grew, ok = m.s.alloc(g.samplers)
			if !ok {
				return errNoRoom
			}
			if grew {
				m.grown++
			}
			m.s.remember(g.sKey, block, off)
		}
		g.sBlock = block
	}
	return nil
}

type noRoom struct{}

func (noRoom) Error() string { return "no room" }

var errNoRoom error = noRoom{}

// frame makes n groups of the model, one after the other, binding each into one of four group slots as
// the renderer does and settling the blocks after each, and returns the number of blocks the slot made.
func (m *slotModel) frame(t *testing.T, n int) (rehomed int) {
	t.Helper()
	var bound [4]*fakeGroup
	for i := 0; i < n; i++ {
		// Groups of 1, 2 and 2 views and one sampler, with 7 different tables between them.
		g := &fakeGroup{views: 1 + (i%3+1)/2, samplers: i % 2, vKey: string(rune('a' + i%7)), sKey: string(rune('s' + i%2))}
		if err := m.place(g); err != nil {
			t.Fatalf("group %d: %v", i, err)
		}
		bound[i%4] = g
		home := func(k int) (int, int, bool) {
			if bound[k] == nil {
				return 0, 0, false
			}
			return bound[k].vBlock, bound[k].sBlock, true
		}
		latest := func() (int, int) { return m.v.cur, m.s.cur }
		moved, ok, err := settleBlocks(4, home, latest, func(k int) error { rehomed++; return m.place(bound[k]) }, 5)
		if err != nil || !ok {
			t.Fatalf("group %d: settled %v, %v", i, ok, err)
		}
		_ = moved
	}
	return rehomed
}

// TestTinyBlocksMakeABoundedNumberOfBlocksAndAreReused: a frame with many groups over tiny blocks makes
// a number of blocks that the frame determines, the other slot makes the same for the same frame, and a
// third frame in the first slot, after its reset, makes none (SPEC §9.8 "Decided in step 28.9").
func TestTinyBlocksMakeABoundedNumberOfBlocksAndAreReused(t *testing.T) {
	const groups = 600
	// A block holds the groups that are bound at once (8 descriptors at most here) many times over:
	// a block as small as that group would hold nothing else, and every new group would make a block.
	newSlot := func() *slotModel { return &slotModel{v: newBlockAlloc(64, 40), s: newBlockAlloc(8, 40)} }
	a, b := newSlot(), newSlot()
	a.frame(t, groups)
	made := a.grown
	t.Logf("%d groups over blocks of 64 and 8 descriptors made %d blocks", groups, made)
	if made == 0 || made > 2*groups {
		t.Fatalf("slot A made %d blocks", made)
	}
	// With 7 view tables and 2 sampler tables the groups that are in use at once are few, so the
	// blocks are far fewer than the groups.
	if made > groups/10 {
		t.Errorf("%d blocks for %d groups: the tables are not shared", made, groups)
	}
	b.frame(t, groups)
	if b.grown != made {
		t.Errorf("the other slot made %d blocks for the same frame, the first made %d", b.grown, made)
	}
	a.v.reset()
	a.s.reset()
	a.frame(t, groups)
	if a.grown != made {
		t.Errorf("a third frame in the first slot made %d more blocks", a.grown-made)
	}
	// And a cap of blocks is a failure, not growth: a frame that needs more than the cap is refused.
	c := &slotModel{v: newBlockAlloc(6, 2), s: newBlockAlloc(2, 2)}
	var bound *fakeGroup
	failed := false
	for i := 0; i < 50 && !failed; i++ {
		bound = &fakeGroup{views: 5, vKey: string(rune('A' + i))}
		failed = c.place(bound) != nil
	}
	if !failed || c.v.blocks() != 2 {
		t.Errorf("the cap of 2 blocks: failed %v with %d blocks", failed, c.v.blocks())
	}
}

func TestSettleBlocks(t *testing.T) {
	type g struct{ v, s int }
	tests := []struct {
		name       string
		groups     []*g // nil is not bound
		latest     [2]int
		wantMoved  []bool
		wantSettle bool
		growBy     int // each rehome adds this many blocks (the model of a full block)
	}{
		{"everything in the latest blocks moves nothing", []*g{{0, -1}, {0, 0}, nil, {-1, 0}}, [2]int{0, 0}, []bool{false, false, false, false}, true, 0},
		{"a group in an earlier block is moved", []*g{{0, -1}, {1, 0}, nil, {1, -1}}, [2]int{1, 0}, []bool{true, false, false, false}, true, 0},
		{"a group with no table is never moved", []*g{{-1, -1}, {2, -1}}, [2]int{2, 0}, []bool{false, false}, true, 0},
		{"the sampler block is checked too", []*g{{1, 0}, {1, 1}}, [2]int{1, 1}, []bool{true, false}, true, 0},
		{"a rehome that adds a block moves the others again and does not settle when it always does", []*g{{0, -1}, {0, -1}}, [2]int{1, 0}, []bool{true, true}, false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest := tt.latest
			rehomed := 0
			moved, ok, err := settleBlocks(len(tt.groups),
				func(i int) (int, int, bool) {
					if tt.groups[i] == nil {
						return 0, 0, false
					}
					return tt.groups[i].v, tt.groups[i].s, true
				},
				func() (int, int) { return latest[0], latest[1] },
				func(i int) error {
					rehomed++
					latest[0] += tt.growBy
					tt.groups[i].v, tt.groups[i].s = latest[0], latest[1]
					return nil
				}, 5)
			if err != nil || ok != tt.wantSettle {
				t.Fatalf("ok %v, err %v; want ok %v", ok, err, tt.wantSettle)
			}
			for i := range tt.wantMoved {
				if moved[i] != tt.wantMoved[i] {
					t.Errorf("moved = %v, want %v", moved, tt.wantMoved)
					break
				}
			}
			if !tt.wantSettle && rehomed > 5*len(tt.groups) {
				t.Errorf("%d rehomes: the rounds are not bounded", rehomed)
			}
		})
	}
}
