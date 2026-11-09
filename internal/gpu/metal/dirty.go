package metal

import "sort"

// span is a range of bytes of a buffer: the first byte and the number of bytes.
type span struct{ off, n int }

// end returns the offset after the span.
func (s span) end() int { return s.off + s.n }

// dirtyRanges collects the ranges of a managed buffer that the CPU wrote, so that one didModifyRange: tells
// Metal about each merged range instead of one call for each write (SPEC §29.5, the managed-memory path of
// Macs with a discrete GPU). Ranges that overlap or touch are merged.
type dirtyRanges struct{ spans []span }

// add records a write of n bytes at off; an empty or negative range is ignored.
func (d *dirtyRanges) add(off, n int) {
	if n <= 0 || off < 0 {
		return
	}
	d.spans = append(d.spans, span{off, n})
}

// take returns the merged ranges in order of their offsets and forgets them.
func (d *dirtyRanges) take() []span {
	if len(d.spans) == 0 {
		return nil
	}
	sort.Slice(d.spans, func(i, j int) bool { return d.spans[i].off < d.spans[j].off })
	merged := []span{d.spans[0]}
	for _, s := range d.spans[1:] {
		last := &merged[len(merged)-1]
		if s.off <= last.end() {
			if e := s.end(); e > last.end() {
				last.n = e - last.off
			}
			continue
		}
		merged = append(merged, s)
	}
	d.spans = d.spans[:0]
	return merged
}

// empty says that nothing is recorded.
func (d *dirtyRanges) empty() bool { return len(d.spans) == 0 }
