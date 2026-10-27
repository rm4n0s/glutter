package widgets

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
	"weak"

	"github.com/rm4n0s/glutter/services"
)

// TestUndoStackDropsTheRedoEntriesItReplaces checks that an entry that a push drops from the redo
// part of the stack does not stay in the array behind the slice (step 29.21).
func TestUndoStackDropsTheRedoEntriesItReplaces(t *testing.T) {
	var st undoStack[*[1 << 10]byte]
	var dropped []weak.Pointer[[1 << 10]byte]
	for i := 0; i < 4; i++ {
		v := new([1 << 10]byte)
		v[0] = byte(i)
		if i >= 2 {
			dropped = append(dropped, weak.Make(v))
		}
		st.push(v)
	}
	st.undo()
	st.undo()
	st.undo() // back at the first entry
	st.push(new([1 << 10]byte))
	runtime.GC()
	runtime.GC()
	for i, w := range dropped {
		if w.Value() != nil {
			t.Errorf("the redo entry %d is still alive after a push dropped it", i)
		}
	}
	if len(st.list) != 2 {
		t.Errorf("the stack has %d entries, want 2", len(st.list))
	}
}

// TestUndoStackIsBounded checks the bound of the undo stack (step 29.21): at most maxUndoEntries
// values and maxUndoBytes bytes of text, the oldest dropped first, and the last two always kept.
func TestUndoStackIsBounded(t *testing.T) {
	text := func(i, size int) services.TextEditingValue {
		b := make([]byte, size)
		for j := range b {
			b[j] = 'a'
		}
		return services.TextEditingValue{Text: strconv.Itoa(i) + string(b)}
	}
	tests := []struct {
		name        string
		pushes      int
		size        int // bytes of text of each value (0: ints are pushed)
		wantLen     int
		wantOldest  int // the number of the oldest value left
		wantMaxSize int
	}{
		{"fewer values than the bound", 10, 0, 10, 0, 0},
		{"more values than the bound", maxUndoEntries + 500, 0, maxUndoEntries, 500, 0},
		{"short texts, bound by count", maxUndoEntries + 10, 100, maxUndoEntries, 10, maxUndoBytes},
		{"large texts, bound by bytes", 40, 1 << 20, 15, 25, maxUndoBytes},
		{"one text larger than the bound", 5, maxUndoBytes + 1, 2, 3, 3 * (maxUndoBytes + 2)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.size == 0 {
				var st undoStack[int]
				for i := range tt.pushes {
					st.push(i)
				}
				if len(st.list) != tt.wantLen || st.list[0] != tt.wantOldest || st.index != len(st.list)-1 {
					t.Fatalf("len %d, oldest %d, index %d; want len %d, oldest %d, index at the end", len(st.list), st.list[0], st.index, tt.wantLen, tt.wantOldest)
				}
				for range tt.pushes {
					st.undo()
				}
				if v, _ := st.currentValue(); v != tt.wantOldest {
					t.Errorf("undo to the start reached %d, want %d", v, tt.wantOldest)
				}
				return
			}
			var st undoStack[services.TextEditingValue]
			for i := range tt.pushes {
				st.push(text(i, tt.size))
			}
			oldest, _ := strconv.Atoi(strings.TrimRight(st.list[0].Text, "a"))
			if len(st.list) != tt.wantLen || oldest != tt.wantOldest {
				t.Errorf("len %d, oldest %d; want len %d, oldest %d", len(st.list), oldest, tt.wantLen, tt.wantOldest)
			}
			sum := 0
			for _, v := range st.list {
				sum += len(v.Text)
			}
			if sum != st.bytes || st.bytes > tt.wantMaxSize {
				t.Errorf("bytes %d (counted %d), want at most %d", st.bytes, sum, tt.wantMaxSize)
			}
		})
	}
}

// TestUndoStackCountsTheBytesOfDroppedRedoEntries checks that a push that drops the redo part
// takes its bytes off the count.
func TestUndoStackCountsTheBytesOfDroppedRedoEntries(t *testing.T) {
	var st undoStack[services.TextEditingValue]
	for _, s := range []string{"a", "bb", "ccc", "dddd"} {
		st.push(services.TextEditingValue{Text: s})
	}
	st.undo()
	st.undo()
	st.push(services.TextEditingValue{Text: "eeeee"})
	if want := len("a") + len("bb") + len("eeeee"); st.bytes != want || len(st.list) != 3 {
		t.Errorf("bytes %d and %d values, want %d and 3", st.bytes, len(st.list), want)
	}
	st.clear()
	if st.bytes != 0 {
		t.Errorf("bytes %d after clear, want 0", st.bytes)
	}
}
