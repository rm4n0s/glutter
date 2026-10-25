package d3d12

// blockAlloc hands out ranges of descriptors from the blocks of one kind of shader-visible heap of a
// frame slot (SPEC §9.8 "Descriptors"). A block is one heap of size descriptors; when the range asked
// for does not fit in the block being filled, the next block is used, and one is added when there is
// none. reset makes every block empty again, keeping the blocks, when the frame that used them is
// done. It is plain Go: the device makes a heap for each block alloc says is new.
//
// The memory a heap takes does not depend on its size (Wine's vkd3d takes about 120 MB for each one,
// whatever it holds), so the blocks are big, a frame makes few, and the number of blocks of a slot is
// capped at maxBlocks: a frame that needs more fails with an error instead of using the machine's
// memory. A table that a frame has placed already (same descriptors) is shared by the bind groups that
// need it (lookup and remember), which keeps a frame to the number of different tables it uses.
type blockAlloc struct {
	size      int
	maxBlocks int
	heads     []int // the descriptors taken in each block
	cur       int   // the block being filled
	tables    map[string]tableLoc
}

// tableLoc is where a table of descriptors was placed.
type tableLoc struct{ block, offset int }

func newBlockAlloc(size, maxBlocks int) blockAlloc {
	return blockAlloc{size: size, maxBlocks: maxBlocks, tables: map[string]tableLoc{}}
}

// lookup finds the table that was placed with the key in the block being filled: a table in an earlier
// block is not reused, since the groups are moved into the latest block (settleBlocks).
func (b *blockAlloc) lookup(key string) (block, offset int, ok bool) {
	t, ok := b.tables[key]
	if !ok || len(b.heads) == 0 || t.block != b.cur {
		return 0, 0, false
	}
	return t.block, t.offset, true
}

// remember keeps where a table was placed.
func (b *blockAlloc) remember(key string, block, offset int) {
	b.tables[key] = tableLoc{block, offset}
}

// alloc takes n descriptors and returns the block and the first descriptor in it. grew says that the
// block is new: the caller makes its heap. ok is false when n does not fit in a block at all, or when it
// needs a block after the last that maxBlocks allows (maxBlocks 0 is no limit).
func (b *blockAlloc) alloc(n int) (block, offset int, grew, ok bool) {
	if n > b.size {
		return 0, 0, false, false
	}
	if len(b.heads) == 0 {
		b.heads = append(b.heads, 0)
		grew = true
	}
	if b.heads[b.cur]+n > b.size {
		if b.maxBlocks > 0 && b.cur+1 >= b.maxBlocks {
			return 0, 0, false, false
		}
		b.cur++
		if b.cur == len(b.heads) {
			b.heads = append(b.heads, 0)
			grew = true
		}
	}
	offset = b.heads[b.cur]
	b.heads[b.cur] += n
	return b.cur, offset, grew, true
}

// reset empties every block, and forgets the tables.
func (b *blockAlloc) reset() {
	for i := range b.heads {
		b.heads[i] = 0
	}
	b.cur = 0
	clear(b.tables)
}

// blocks is the number of blocks made so far.
func (b *blockAlloc) blocks() int { return len(b.heads) }

// used is the number of descriptors taken in all blocks.
func (b *blockAlloc) used() int {
	n := 0
	for _, h := range b.heads {
		n += h
	}
	return n
}

// rollback takes back the block that alloc just added, when the device could not make its heap.
func (b *blockAlloc) rollback() {
	b.heads = b.heads[:len(b.heads)-1]
	b.cur = max(len(b.heads)-1, 0)
}

// settleBlocks brings the groups that are bound to the latest blocks of the heaps, since only one heap of
// each kind can be set: home says where group i is (a block of each heap, -1 for a heap it has no table
// in; has is false for a group that is not bound), latest the blocks being filled, and rehome copies group i
// into them (which may add a block, and then the others are moved again). It returns which groups were
// moved, and ok is false when it did not settle in rounds rounds (the groups bound together do not fit one
// block). A frame that needs one block, the normal case, moves nothing.
func settleBlocks(n int, home func(i int) (v, s int, has bool), latest func() (v, s int), rehome func(i int) error, rounds int) (moved []bool, ok bool, err error) {
	moved = make([]bool, n)
	for round := 0; round < rounds; round++ {
		lv, ls := latest()
		for i := 0; i < n; i++ {
			v, s, has := home(i)
			if !has || (v < 0 && s < 0) {
				continue
			}
			if (v >= 0 && v != lv) || (s >= 0 && s != ls) {
				if err := rehome(i); err != nil {
					return moved, false, err
				}
				moved[i] = true
			}
		}
		if nv, ns := latest(); nv == lv && ns == ls {
			return moved, true, nil
		}
	}
	return moved, false, nil
}

// tableKey names a table by the CPU handles its descriptors are copied from (0 is the null descriptor or
// the default sampler): two groups with the same key have the same table.
func tableKey(srcs []uintptr) string {
	b := make([]byte, 0, 8*len(srcs))
	for _, s := range srcs {
		v := uint64(s)
		b = append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
	}
	return string(b)
}
