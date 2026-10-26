// Package semtree builds semantics updates for the tests of UI Automation (SPEC §25.11): the nodes the way
// the engine fills them in, so that a test states only what it cares about.
//
// Flutter: no counterpart
package semtree

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Identity is the identity transform of a node, a 4x4 matrix in column-major order.
//
// Flutter: no counterpart
var Identity = [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}

// Scale is a transform that scales x and y.
//
// Flutter: no counterpart
func Scale(s float32) [16]float32 { return [16]float32{s, 0, 0, 0, 0, s, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }

// N makes the update of a node the way the engine fills one in: the fields that the framework leaves
// unset are -1 or NaN, and the transform is the identity. The children are the IDs in traversal order.
//
// Flutter: no counterpart
func N(id int, label string, x, y, w, h float64, children ...int) ui.SemanticsNodeUpdate {
	return ui.SemanticsNodeUpdate{
		ID: id, Label: label, Rect: ui.NewRectFromLTWH(x, y, w, h), Transform: Identity, HitTestTransform: Identity,
		TextSelectionBase: -1, TextSelectionExtent: -1, PlatformViewID: -1, MaxValueLength: -1, CurrentValueLength: -1,
		ScrollPosition: math.NaN(), ScrollExtentMax: math.NaN(), ScrollExtentMin: math.NaN(), TraversalParent: -1,
		ChildrenInTraversalOrder: children, ChildrenInHitTestOrder: children,
	}
}

// Update makes a semantics update of the nodes.
//
// Flutter: no counterpart
func Update(nodes ...ui.SemanticsNodeUpdate) *ui.SemanticsUpdate {
	return &ui.SemanticsUpdate{Nodes: nodes}
}
