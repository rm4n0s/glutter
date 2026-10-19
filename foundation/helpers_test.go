package foundation

import (
	"regexp"
	"testing"
)

// TestTree is the DiagnosticableTree of Flutter's tests.
type TestTree struct {
	name       string
	style      DiagnosticsTreeStyle
	children   []*TestTree
	properties []DiagnosticsNode
}

func (t *TestTree) DebugDescribeChildren() []DiagnosticsNode {
	var nodes []DiagnosticsNode
	for _, child := range t.children {
		nodes = append(nodes, ToDiagnosticsNode(child, DiagnosticsNodeOpts{Name: Some("child " + child.name), Style: child.style}))
	}
	return nodes
}

func (t *TestTree) DebugFillProperties(properties *DiagnosticPropertiesBuilder) {
	if t.style != DiagnosticsTreeStyleUnspecified {
		properties.DefaultDiagnosticsTreeStyle = t.style
	}
	for _, p := range t.properties {
		properties.Add(p)
	}
}

var hashCodes = regexp.MustCompile(`#[0-9a-fA-F]{5}`)

// ignoringHashCodes is Flutter's equalsIgnoringHashCodes normalization.
func ignoringHashCodes(s string) string { return hashCodes.ReplaceAllString(s, "#00000") }

type exampleEnum int

const (
	exampleHello exampleEnum = iota
	exampleWorld
	exampleDeferToChild
)

func (e exampleEnum) Name() string { return [...]string{"hello", "world", "deferToChild"}[e] }

// recoverError runs fn and returns the error it panicked with.
func recoverError(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic")
		}
		e, ok := r.(error)
		if !ok {
			t.Fatalf("expected an error panic, got %T (%v)", r, r)
		}
		err = e
	}()
	fn()
	return nil
}
