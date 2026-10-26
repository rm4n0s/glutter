package svg

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzParse checks that no input makes Parse panic or hang; a document is either parsed or an error.
func FuzzParse(f *testing.F) {
	files, _ := filepath.Glob("testdata/*.svg")
	for _, name := range files {
		if data, err := os.ReadFile(name); err == nil {
			f.Add(data)
		}
	}
	f.Add([]byte(`<svg><use href="#a"/><g id="a"><use href="#a"/></g></svg>`))
	f.Add([]byte(`<svg viewBox="0 0 1 1"><style>*{fill:red}</style><rect width="50%" height="1em" style="transform:rotate(1deg)"/></svg>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := Parse(data)
		if (d == nil) == (err == nil) {
			t.Fatalf("document %v and error %v", d != nil, err)
		}
	})
}

// FuzzPathData checks that no path data makes the parser panic.
func FuzzPathData(f *testing.F) {
	for _, s := range []string{"M0 0 L1 1 z", "M1.5.5-1-2A1 1 0 011 1", "m1 1c1 2 3 4 5 6s1 2 3 4q1 2 3 4t1 2", "M0 0 A0 0 0 0 0 0 0", "M1e999 0"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, _ := ParsePathData(s)
		if p == nil {
			t.Fatal("nil path")
		}
	})
}
