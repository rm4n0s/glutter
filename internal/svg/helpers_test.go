package svg

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

const svgOpen = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100">`

// mustParse parses a document and fails the test if that gives an error.
func mustParse(t *testing.T, src string, opts ...ParseOpts) *Document {
	t.Helper()
	d, err := Parse([]byte(src), opts...)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

// findNode returns the first node with the id, in document order, or nil.
func findNode(n *Node, id string) *Node {
	if n.ID == id {
		return n
	}
	for _, c := range n.Children {
		if f := findNode(c, id); f != nil {
			return f
		}
	}
	return nil
}

// mustFind is findNode that fails the test when there is no such node.
func mustFind(t *testing.T, d *Document, id string) *Node {
	t.Helper()
	n := findNode(d.Root, id)
	if n == nil {
		t.Fatalf("no node with id %q", id)
	}
	return n
}

func col(s string) ui.Color {
	c, _, ok := ParseColor(s)
	if !ok {
		panic("bad test color " + s)
	}
	return c
}

func countNodes(n *Node) int {
	c := 1
	for _, ch := range n.Children {
		c += countNodes(ch)
	}
	return c
}
