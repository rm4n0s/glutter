package docs_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// header turns a fragment of declarations of the guide into a program: the packages that the
// guide says it leaves out, and a use of each of them so that the fragment need not use all.
const header = `package main

import (
	"context"
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

var (
	_ = context.Background
	_ = strconv.Itoa
	_ = time.Second
	_ animation.Animation[float64]
	_ foundation.Key
	_ painting.EdgeInsets
	_ rendering.Box
	_ ui.Color
)

func main() {}

`

var goBlock = regexp.MustCompile("(?s)```go\n(.*?)```")

// TestGoBlocksCompile type-checks every Go block of the README and of the guide: a block that
// starts with "package" is a program of its own, any other is wrapped by header.
func TestGoBlocksCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go vet on the examples of the documentation")
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(root, "docs", "_guidecheck")
	if err := os.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Error(err)
		}
	})
	var dirs []string
	for _, doc := range []string{"README.md", filepath.Join("docs", "GUIDE.md")} {
		data, err := os.ReadFile(filepath.Join(root, doc))
		if err != nil {
			t.Fatal(err)
		}
		for i, m := range goBlock.FindAllStringSubmatch(string(data), -1) {
			src := m[1]
			if strings.HasPrefix(src, "import ") {
				continue // the import line that the text shows is not a program
			}
			if !strings.HasPrefix(src, "package ") {
				src = header + src
			}
			name := fmt.Sprintf("%s_%d", strings.TrimSuffix(strings.ToLower(filepath.Base(doc)), ".md"), i+1)
			dir := filepath.Join(tmp, name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			dirs = append(dirs, "./docs/_guidecheck/"+name)
		}
	}
	if len(dirs) < 10 {
		t.Fatalf("found %d Go blocks, want the 10 of the guide and the quick start", len(dirs))
	}
	cmd := exec.Command("go", append([]string{"vet"}, dirs...)...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("go vet on the documentation's Go blocks failed: %v\n%s", err, out)
	}
}
