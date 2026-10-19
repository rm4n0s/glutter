package semantics

import (
	"math"
	"reflect"
	"slices"
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// nodeSet is a set of nodes that remembers the order of insertion, like Dart's
// Set.
type nodeSet struct{ list []*SemanticsNode }

func (s *nodeSet) has(n *SemanticsNode) bool { return slices.Contains(s.list, n) }
func (s *nodeSet) add(n *SemanticsNode) {
	if !s.has(n) {
		s.list = append(s.list, n)
	}
}
func (s *nodeSet) remove(n *SemanticsNode) {
	s.list = slices.DeleteFunc(s.list, func(x *SemanticsNode) bool { return x == n })
}
func (s *nodeSet) len() int { return len(s.list) }

func sameKeyType(a, b SemanticsSortKey) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

type traversalSortNode struct {
	node     *SemanticsNode
	sortKey  SemanticsSortKey
	position int
}

func (a traversalSortNode) compare(b traversalSortNode) int {
	if a.sortKey == nil || b.sortKey == nil {
		return a.position - b.position
	}
	return SemanticsSortKeyCompare(a.sortKey, b.sortKey)
}

func sortTraversalNodes(nodes []traversalSortNode) {
	slices.SortStableFunc(nodes, func(a, b traversalSortNode) int { return a.compare(b) })
}

// boxEdge is a leading or trailing edge of the rectangle of a node.
type boxEdge struct {
	isLeadingEdge bool
	offset        float64
	node          *SemanticsNode
}

func sortEdges(edges []boxEdge) {
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].offset < edges[j].offset })
}

// semanticsSortGroup is a group of nodes that are a row or a column.
type semanticsSortGroup struct {
	startOffset   float64
	textDirection ui.TextDirection
	nodes         []*SemanticsNode
}

// sortedWithinVerticalGroup orders the nodes of a band from side to side.
func (g *semanticsSortGroup) sortedWithinVerticalGroup() []*SemanticsNode {
	var edges []boxEdge
	for _, child := range g.nodes {
		r := child.rect.Deflate(0.1)
		edges = append(edges,
			boxEdge{true, pointInParentCoordinates(child, r.TopLeft()).Dx, child},
			boxEdge{false, pointInParentCoordinates(child, r.BottomRight()).Dx, child})
	}
	sortEdges(edges)
	var horizontal []*semanticsSortGroup
	var group *semanticsSortGroup
	depth := 0
	for _, e := range edges {
		if e.isLeadingEdge {
			depth++
			if group == nil {
				group = &semanticsSortGroup{startOffset: e.offset, textDirection: g.textDirection}
			}
			group.nodes = append(group.nodes, e.node)
		} else {
			depth--
		}
		if depth == 0 {
			horizontal = append(horizontal, group)
			group = nil
		}
	}
	sort.SliceStable(horizontal, func(i, j int) bool { return horizontal[i].startOffset < horizontal[j].startOffset })
	if g.textDirection == ui.TextDirectionRTL {
		slices.Reverse(horizontal)
	}
	var out []*SemanticsNode
	for _, h := range horizontal {
		out = append(out, h.sortedWithinKnot()...)
	}
	return out
}

// sortedWithinKnot orders the nodes that overlap each other by their centers.
func (g *semanticsSortGroup) sortedWithinKnot() []*SemanticsNode {
	if len(g.nodes) <= 1 {
		return g.nodes
	}
	nodeMap := map[int]*SemanticsNode{}
	edges := map[int]int{}
	for _, node := range g.nodes {
		nodeMap[node.id] = node
		center := pointInParentCoordinates(node, node.rect.Center())
		for _, next := range g.nodes {
			if node == next {
				continue
			}
			if to, ok := edges[next.id]; ok && to == node.id {
				continue
			}
			nextCenter := pointInParentCoordinates(next, next.rect.Center())
			direction := nextCenter.Sub(center).Direction()
			ltrForward := g.textDirection == ui.TextDirectionLTR && -math.Pi/4 < direction && direction < 3*math.Pi/4
			rtlForward := g.textDirection == ui.TextDirectionRTL && (direction < -3*math.Pi/4 || direction > 3*math.Pi/4)
			if ltrForward || rtlForward {
				edges[node.id] = next.id
			}
		}
	}
	var sortedIDs []int
	visited := map[int]bool{}
	start := slices.Clone(g.nodes)
	sort.SliceStable(start, func(i, j int) bool {
		a, b := pointInParentCoordinates(start[i], start[i].rect.TopLeft()), pointInParentCoordinates(start[j], start[j].rect.TopLeft())
		if a.Dy != b.Dy {
			return a.Dy > b.Dy
		}
		return a.Dx > b.Dx
	})
	var search func(id int)
	search = func(id int) {
		if visited[id] {
			return
		}
		visited[id] = true
		if to, ok := edges[id]; ok {
			search(to)
		}
		sortedIDs = append(sortedIDs, id)
	}
	for _, n := range start {
		search(n.id)
	}
	out := make([]*SemanticsNode, len(sortedIDs))
	for i, id := range sortedIDs {
		out[len(sortedIDs)-1-i] = nodeMap[id]
	}
	return out
}

// pointInParentCoordinates transforms a point of the node to the coordinates of
// its parent for traversal.
func pointInParentCoordinates(node *SemanticsNode, point ui.Offset) ui.Offset {
	t := node.traversalTransform()
	if t == nil {
		return point
	}
	v := t.Transform3(ui.Vector3{X: float32(point.Dx), Y: float32(point.Dy)})
	return ui.Offset{Dx: float64(v.X), Dy: float64(v.Y)}
}

// childrenInDefaultOrder orders the children into rows from top to bottom, and
// each row from side to side.
func childrenInDefaultOrder(children []*SemanticsNode, textDirection ui.TextDirection) []*SemanticsNode {
	var edges []boxEdge
	for _, child := range children {
		r := child.rect.Deflate(0.1)
		edges = append(edges,
			boxEdge{true, pointInParentCoordinates(child, r.TopLeft()).Dy, child},
			boxEdge{false, pointInParentCoordinates(child, r.BottomRight()).Dy, child})
	}
	sortEdges(edges)
	var vertical []*semanticsSortGroup
	var group *semanticsSortGroup
	depth := 0
	for _, e := range edges {
		if e.isLeadingEdge {
			depth++
			if group == nil {
				group = &semanticsSortGroup{startOffset: e.offset, textDirection: textDirection}
			}
			group.nodes = append(group.nodes, e.node)
		} else {
			depth--
		}
		if depth == 0 {
			vertical = append(vertical, group)
			group = nil
		}
	}
	sort.SliceStable(vertical, func(i, j int) bool { return vertical[i].startOffset < vertical[j].startOffset })
	var out []*SemanticsNode
	for _, g := range vertical {
		out = append(out, g.sortedWithinVerticalGroup()...)
	}
	return out
}
