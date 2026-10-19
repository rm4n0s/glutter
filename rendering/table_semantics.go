package rendering

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// tableIndex is the position of a cell in the table.
//
// Flutter: rendering/table.dart _Index
type tableIndex struct{ y, x int }

// DescribeSemanticsConfiguration makes the table a semantic boundary whose
// children form nodes of their own.
//
// Flutter: rendering/table.dart RenderTable.describeSemanticsConfiguration
func (r *RenderTable) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderBox.DescribeSemanticsConfiguration(config)
	config.SetRole(ui.SemanticsRoleTable)
	config.SetIsSemanticBoundary(true)
	config.ExplicitChildNodes = true
}

// ClearSemantics forgets the nodes of the rows and cells too.
//
// Flutter: rendering/table.dart RenderTable.clearSemantics
func (r *RenderTable) ClearSemantics() {
	r.RenderBox.ClearSemantics()
	clear(r.cachedRows)
	clear(r.cachedCells)
}

func semanticsRectWithOffset(node *semantics.SemanticsNode) ui.Rect {
	offset := ui.OffsetZero
	if t := node.Transform(); t != nil {
		if o, ok := t.AsTranslation(); ok {
			offset = o
		}
	}
	return node.Rect().Shift(offset)
}

// AssembleSemanticsNode gives the table the nodes of its rows, which wrap the
// nodes of the cells. The rows are not render objects, so their nodes are made
// here; a cell with several nodes or without the role of a cell gets a wrapper
// node.
//
// Flutter: rendering/table.dart RenderTable.assembleSemanticsNode
func (r *RenderTable) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	if r.cachedRows == nil {
		r.cachedRows = map[int]*semantics.SemanticsNode{}
		r.cachedCells = map[tableIndex]*semantics.SemanticsNode{}
		r.idToIndexMap = map[int]tableIndex{}
	}
	var rows []*semantics.SemanticsNode
	rawCells := make([][][]*semantics.SemanticsNode, r.rows)
	for y := range rawCells {
		rawCells[y] = make([][]*semantics.SemanticsNode, r.columns)
	}

	findRowIndex := func(top float64) int {
		// Iterate over rows rather than the length of rowTops, which has one
		// more entry that marks the bottom of the last row.
		for i := r.rows - 1; i >= 0; i-- {
			if r.rowTops[i] <= top {
				return i
			}
		}
		return -1
	}
	findColumnIndex := func(left float64) int {
		if r.columnLefts == nil {
			return -1
		}
		for i := len(r.columnLefts) - 1; i >= 0; i-- {
			if r.columnLefts[i] <= left {
				return i
			}
		}
		return -1
	}
	shiftTransform := func(node *semantics.SemanticsNode, dx, dy float64) {
		offset := ui.OffsetZero
		if t := node.Transform(); t != nil {
			if o, ok := t.AsTranslation(); ok {
				offset = o
			}
		}
		node.SetTransform(ui.NewMatrix4TranslationValues(float32(offset.Dx+dx), float32(offset.Dy+dy), 0))
	}

	for _, child := range children {
		if index, ok := r.idToIndexMap[child.ID()]; ok {
			if index.y < r.rows && index.x < r.columns {
				rawCells[index.y][index.x] = append(rawCells[index.y][index.x], child)
			}
			continue
		}
		rect := semanticsRectWithOffset(child)
		y, x := findRowIndex(rect.Top), findColumnIndex(rect.Left)
		if y != -1 && x != -1 {
			rawCells[y][x] = append(rawCells[y][x], child)
		}
	}

	for y := 0; y < r.rows; y++ {
		rowBox := r.GetRowBox(y)
		// Skip an empty row.
		if rowBox.Height() == 0 {
			continue
		}
		newRow := r.cachedRows[y]
		if newRow == nil {
			newRow = semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{ShowOnScreen: func() {
				r.ShowOnScreen(ShowOnScreenOpts{Descendant: r.self, Rect: foundation.Some(rowBox)})
			}})
			r.cachedRows[y] = newRow
		}

		var cells []*semantics.SemanticsNode
		for x := 0; x < r.columns; x++ {
			rawChildren := rawCells[y][x]
			if len(rawChildren) == 0 {
				continue
			}
			// A cell with several children, or whose only child is not a cell or
			// a column header (it has another role, or the user does not use
			// TableCell), gets a new node with the role cell.
			addCellWrapper := len(rawChildren) > 1 ||
				(rawChildren[0].Role() != ui.SemanticsRoleCell && rawChildren[0].Role() != ui.SemanticsRoleColumnHeader)

			var cell *semantics.SemanticsNode
			if !addCellWrapper {
				cell = rawChildren[0]
			} else {
				index := tableIndex{y, x}
				cell = r.cachedCells[index]
				if cell == nil {
					cell = semantics.NewSemanticsNode()
					r.cachedCells[index] = cell
				}
				cellConfig := semantics.NewSemanticsConfiguration()
				cellConfig.SetRole(ui.SemanticsRoleCell)
				cell.UpdateWith(cellConfig, rawChildren)
			}

			var cellWidth float64
			if x == r.columns-1 {
				cellWidth = rowBox.Width() - r.columnLefts[x]
			} else {
				cellWidth = r.columnLefts[x+1] - r.columnLefts[x]
			}
			// Skip an invisible cell.
			if cellWidth <= 0 {
				continue
			}
			if addCellWrapper {
				cell.SetTransform(ui.NewMatrix4TranslationValues(float32(r.columnLefts[x]), 0, 0))
				cell.SetRect(ui.NewRectFromLTWH(0, 0, cellWidth, rowBox.Height()))
			}
			for _, child := range rawChildren {
				r.idToIndexMap[child.ID()] = tableIndex{y, x}

				// The rect should satisfy 0 <= top < bottom <= rowBox.height.
				localRect := semanticsRectWithOffset(child)
				dy := 0.0
				if localRect.Bottom > rowBox.Height()+foundation.PrecisionErrorTolerance {
					dy = -r.rowTops[y]
				}
				// With a wrapper the rect is relative to the cell and should
				// satisfy 0 <= left < right <= cellWidth; without one it is
				// relative to the row and should satisfy
				// columnLefts[x] <= left < right <= columnLefts[x+1].
				dx := 0.0
				if addCellWrapper {
					if localRect.Left >= cellWidth {
						dx = -r.columnLefts[x]
					}
				} else if localRect.Right <= r.columnLefts[x] {
					dx = r.columnLefts[x]
				}
				if dx != 0 || dy != 0 {
					shiftTransform(child, dx, dy)
				}
			}
			cell.IndexInParent = foundation.Some(x)
			cells = append(cells, cell)
		}

		rowConfig := semantics.NewSemanticsConfiguration()
		rowConfig.SetIndexInParent(foundation.Some(y))
		rowConfig.SetRole(ui.SemanticsRoleRow)
		newRow.UpdateWith(rowConfig, cells)
		newRow.SetTransform(ui.NewMatrix4TranslationValues(float32(rowBox.Left), float32(rowBox.Top), 0))
		newRow.SetRect(ui.NewRectFromLTWH(0, 0, rowBox.Width(), rowBox.Height()))
		rows = append(rows, newRow)
	}
	node.UpdateWith(config, rows)
}
