package widgets

import (
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// TableRow defines the configuration of a row in a Table: an optional key, an
// optional decoration and the cells.
//
// Flutter: widgets/table.dart TableRow
type TableRow struct {
	// Key is the key of the row, which lets the table keep the cells of a row
	// when the rows are reordered.
	Key foundation.Key
	// Decoration is the decoration to paint behind the row.
	Decoration painting.Decoration
	// Children are the widgets that fill the cells of the row.
	Children []Widget
}

// NewTableRow creates a row from the cells.
//
// Flutter: widgets/table.dart TableRow.new
func NewTableRow(children []Widget, opts ...TableRowOpts) TableRow {
	o := firstOpts(opts)
	return TableRow{Key: o.Key, Decoration: o.Decoration, Children: children}
}

// TableRowOpts are the optional parameters of NewTableRow.
//
// Flutter: widgets/table.dart TableRow.new
type TableRowOpts struct {
	// Key is the key of the row.
	Key foundation.Key
	// Decoration is the decoration to paint behind the row.
	Decoration painting.Decoration
}

// String describes the row.
//
// Flutter: widgets/table.dart TableRow.toString
func (r TableRow) String() string {
	s := "TableRow("
	if r.Key != nil {
		s += fmt.Sprint(r.Key) + ", "
	}
	if r.Decoration != nil {
		s += fmt.Sprint(r.Decoration) + ", "
	}
	if len(r.Children) == 0 {
		return s + "no children)"
	}
	return s + fmt.Sprint(r.Children) + ")"
}

// TableOpts are the optional parameters of NewTable.
//
// Flutter: widgets/table.dart Table.new
type TableOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ColumnWidths says how wide the columns are, by index; the other columns use
	// DefaultColumnWidth.
	ColumnWidths map[int]rendering.TableColumnWidth
	// DefaultColumnWidth is how wide a column is when ColumnWidths has nothing for
	// it; the default is FlexColumnWidth.
	DefaultColumnWidth rendering.TableColumnWidth
	// TextDirection is the direction in which the columns are laid out; the
	// default is the one of the closest Directionality.
	TextDirection ui.TextDirection
	// Border is the style to use when painting the boundary and interior divisions
	// of the table.
	Border *rendering.TableBorder
	// DefaultVerticalAlignment is how the cells are aligned vertically by default;
	// the default is TableCellVerticalAlignmentTop.
	DefaultVerticalAlignment rendering.TableCellVerticalAlignment
	// TextBaseline is the baseline to use when the vertical alignment is baseline.
	TextBaseline ui.TextBaseline
}

// Table implements the table layout algorithm: a grid of widgets, with rows of
// the same number of cells.
//
// Flutter: widgets/table.dart Table
type Table struct {
	RenderObjectWidget
	children                 []TableRow
	columnWidths             map[int]rendering.TableColumnWidth
	defaultColumnWidth       rendering.TableColumnWidth
	textDirection            ui.TextDirection
	border                   *rendering.TableBorder
	defaultVerticalAlignment rendering.TableCellVerticalAlignment
	textBaseline             ui.TextBaseline
	rowDecorations           []painting.Decoration
}

// NewTable creates a table. In debug builds it panics with the tags
// TableBaselineMissing (the baseline alignment without a baseline),
// TableRowKeysDuplicate, TableRowsIrregular, TableRowEmpty and
// ChildrenHaveDuplicateKeys (for the cells).
//
// Flutter: widgets/table.dart Table.new
func NewTable(children []TableRow, opts ...TableOpts) *Table {
	o := firstOpts(opts)
	if o.DefaultColumnWidth == nil {
		o.DefaultColumnWidth = rendering.NewFlexColumnWidth()
	}
	if o.DefaultVerticalAlignment == rendering.TableCellVerticalAlignmentUnspecified {
		o.DefaultVerticalAlignment = rendering.TableCellVerticalAlignmentTop
	}
	w := &Table{RenderObjectWidget: NewRenderObjectWidgetBase(WidgetOpts{Key: o.Key}), children: children, columnWidths: o.ColumnWidths,
		defaultColumnWidth: o.DefaultColumnWidth, textDirection: o.TextDirection, border: o.Border,
		defaultVerticalAlignment: o.DefaultVerticalAlignment, textBaseline: o.TextBaseline}
	anyDecoration := false
	for _, r := range children {
		if r.Decoration != nil {
			anyDecoration = true
		}
	}
	if anyDecoration {
		w.rowDecorations = make([]painting.Decoration, len(children))
		for i, r := range children {
			w.rowDecorations[i] = r.Decoration
		}
	}
	if foundation.DebugMode {
		w.debugCheck()
	}
	return w
}

func (w *Table) debugCheck() {
	if w.defaultVerticalAlignment == rendering.TableCellVerticalAlignmentBaseline && w.textBaseline == ui.TextBaselineUnspecified {
		panic(errors.New("TableBaselineMissing", "textBaseline is required if you specify the defaultVerticalAlignment with TableCellVerticalAlignmentBaseline"))
	}
	for i, r1 := range w.children {
		if r1.Key == nil {
			continue
		}
		for j, r2 := range w.children {
			if i != j && r2.Key != nil && r1.Key == r2.Key {
				panic(errors.New("TableRowKeysDuplicate", "two or more TableRow children of this Table had the same key; all the keyed TableRow children of a Table must have different keys"))
			}
		}
	}
	if len(w.children) > 0 {
		cellCount := len(w.children[0].Children)
		for _, r := range w.children {
			if len(r.Children) != cellCount {
				panic(errors.New("TableRowsIrregular", "table contains irregular row lengths; every TableRow in a Table must have the same number of children, so that every cell is filled",
					"expected", cellCount, "found", len(r.Children)))
			}
		}
		for _, r := range w.children {
			if len(r.Children) == 0 {
				panic(errors.New("TableRowEmpty", "one or more TableRow have no children; every TableRow in a Table must have at least one child, so there is no empty row"))
			}
		}
	}
	var flat []Widget
	for _, r := range w.children {
		flat = append(flat, r.Children...)
	}
	DebugChildrenHaveDuplicateKeys(w, flat)
}

// Children are the rows of the table.
//
// Flutter: widgets/table.dart Table.children
func (w *Table) Children() []TableRow { return w.children }

// ColumnWidths are the widths of the columns that have their own.
//
// Flutter: widgets/table.dart Table.columnWidths
func (w *Table) ColumnWidths() map[int]rendering.TableColumnWidth { return w.columnWidths }

// DefaultColumnWidth is the width of the columns that have no entry in
// ColumnWidths.
//
// Flutter: widgets/table.dart Table.defaultColumnWidth
func (w *Table) DefaultColumnWidth() rendering.TableColumnWidth { return w.defaultColumnWidth }

// TextDirection is the direction in which the columns are laid out.
//
// Flutter: widgets/table.dart Table.textDirection
func (w *Table) TextDirection() ui.TextDirection { return w.textDirection }

// Border is the style of the boundary and the divisions of the table.
//
// Flutter: widgets/table.dart Table.border
func (w *Table) Border() *rendering.TableBorder { return w.border }

// DefaultVerticalAlignment is how the cells are aligned vertically by default.
//
// Flutter: widgets/table.dart Table.defaultVerticalAlignment
func (w *Table) DefaultVerticalAlignment() rendering.TableCellVerticalAlignment {
	return w.defaultVerticalAlignment
}

// TextBaseline is the baseline used when the vertical alignment is baseline.
//
// Flutter: widgets/table.dart Table.textBaseline
func (w *Table) TextBaseline() ui.TextBaseline { return w.textBaseline }

func (w *Table) columns() int {
	if len(w.children) == 0 {
		return 0
	}
	return len(w.children[0].Children)
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/table.dart Table.createRenderObject
func (w *Table) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	DebugCheckHasDirectionality(bc)
	return rendering.NewRenderTable(firstDirection(w.textDirection, bc), rendering.RenderTableOpts{
		Columns: foundation.Some(w.columns()), Rows: foundation.Some(len(w.children)), ColumnWidths: w.columnWidths,
		DefaultColumnWidth: w.defaultColumnWidth, Border: w.border, RowDecorations: w.rowDecorations,
		Configuration:            CreateLocalImageConfiguration(bc, foundation.None[ui.Size]()),
		DefaultVerticalAlignment: w.defaultVerticalAlignment, TextBaseline: w.textBaseline})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/table.dart Table.updateRenderObject
func (w *Table) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	DebugCheckHasDirectionality(bc)
	r := renderObject.(*rendering.RenderTable)
	r.SetColumnWidths(w.columnWidths)
	r.SetDefaultColumnWidth(w.defaultColumnWidth)
	r.SetTextDirection(firstDirection(w.textDirection, bc))
	r.SetBorder(w.border)
	r.SetRowDecorations(w.rowDecorations)
	r.SetConfiguration(CreateLocalImageConfiguration(bc, foundation.None[ui.Size]()))
	r.SetDefaultVerticalAlignment(w.defaultVerticalAlignment)
	r.SetTextBaseline(w.textBaseline)
}

// CreateElement creates the element, which keeps the cells in rows.
//
// Flutter: widgets/table.dart Table.createElement
func (w *Table) CreateElement() Element {
	e := &tableElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// tableSlot is the place of a cell in the table.
//
// Flutter: widgets/table.dart _TableSlot
type tableSlot struct{ column, row int }

// tableElementRow is a row of elements.
//
// Flutter: widgets/table.dart _TableElementRow
type tableElementRow struct {
	key      foundation.Key
	children []Element
}

// tableElement is the element of a Table.
//
// Flutter: widgets/table.dart _TableElement
type tableElement struct {
	RenderObjectElement
	children           []tableElementRow
	doingMountOrUpdate bool
	forgottenChildren  map[Element]struct{}
}

func (e *tableElement) renderTable() *rendering.RenderTable {
	return e.renderObject.(*rendering.RenderTable)
}

// Mount creates the render object and inflates the cells.
func (e *tableElement) Mount(parent Element, newSlot any) {
	e.doingMountOrUpdate = true
	e.RenderObjectElement.Mount(parent, newSlot)
	rows := e.widget.(*Table).children
	e.children = make([]tableElementRow, len(rows))
	for rowIndex, row := range rows {
		cells := make([]Element, len(row.Children))
		for columnIndex, child := range row.Children {
			cells[columnIndex] = e.InflateWidget(child, tableSlot{columnIndex, rowIndex})
		}
		e.children[rowIndex] = tableElementRow{key: row.Key, children: cells}
	}
	e.updateRenderObjectChildren()
	e.doingMountOrUpdate = false
}

// InsertRenderObjectChild sets up the parent data of the cell; the cells are
// set all at once when the mount or update is done.
func (e *tableElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	e.renderTable().SetupParentData(child)
	if !e.doingMountOrUpdate {
		s := slot.(tableSlot)
		e.renderTable().SetChild(s.column, s.row, child.(rendering.Box))
	}
}

// MoveRenderObjectChild does nothing: the cell moves at the end of Update.
func (e *tableElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {}

// RemoveRenderObjectChild empties the cell.
func (e *tableElement) RemoveRenderObjectChild(_ rendering.RenderObject, slot any) {
	s := slot.(tableSlot)
	e.renderTable().SetChild(s.column, s.row, nil)
}

// Update updates the rows, matching the keyed ones by key.
func (e *tableElement) Update(newWidget Widget) {
	e.doingMountOrUpdate = true
	if e.forgottenChildren == nil {
		e.forgottenChildren = map[Element]struct{}{}
	}
	nw := newWidget.(*Table)
	oldKeyed := map[foundation.Key][]Element{}
	var oldUnkeyed []tableElementRow
	for _, row := range e.children {
		if row.key != nil {
			oldKeyed[row.key] = row.children
		} else {
			oldUnkeyed = append(oldUnkeyed, row)
		}
	}
	taken := map[foundation.Key]bool{}
	var newChildren []tableElementRow
	for rowIndex, row := range nw.children {
		var oldChildren []Element
		if k := row.Key; k != nil {
			if old, ok := oldKeyed[k]; ok {
				oldChildren = old
				taken[k] = true
			}
		} else if len(oldUnkeyed) > 0 {
			oldChildren = oldUnkeyed[0].children
			oldUnkeyed = oldUnkeyed[1:]
		}
		slots := make([]any, len(row.Children))
		for c := range slots {
			slots[c] = tableSlot{c, rowIndex}
		}
		newChildren = append(newChildren, tableElementRow{key: row.Key,
			children: e.UpdateChildren(oldChildren, row.Children, UpdateChildrenOpts{ForgottenChildren: e.forgottenChildren, Slots: slots})})
	}
	for _, row := range oldUnkeyed {
		e.UpdateChildren(row.children, nil, UpdateChildrenOpts{ForgottenChildren: e.forgottenChildren})
	}
	for k, old := range oldKeyed {
		if !taken[k] {
			e.UpdateChildren(old, nil, UpdateChildrenOpts{ForgottenChildren: e.forgottenChildren})
		}
	}
	e.children = newChildren
	e.updateRenderObjectChildren()
	clear(e.forgottenChildren)
	e.RenderObjectElement.Update(newWidget)
	e.doingMountOrUpdate = false
}

func (e *tableElement) updateRenderObjectChildren() {
	columns := 0
	if len(e.children) > 0 {
		columns = len(e.children[0].children)
	}
	var cells []rendering.Box
	for _, row := range e.children {
		for _, c := range row.children {
			cells = append(cells, c.RenderObject().(rendering.Box))
		}
	}
	e.renderTable().SetFlatChildren(columns, cells)
}

// VisitChildren visits the cells that were not forgotten, row by row.
func (e *tableElement) VisitChildren(visitor ElementVisitor) {
	for _, row := range e.children {
		for _, c := range row.children {
			if _, ok := e.forgottenChildren[c]; !ok {
				visitor(c)
			}
		}
	}
}

// ForgetChild remembers that the cell went to another parent.
func (e *tableElement) ForgetChild(child Element) {
	if e.forgottenChildren == nil {
		e.forgottenChildren = map[Element]struct{}{}
	}
	e.forgottenChildren[child] = struct{}{}
	e.RenderObjectElement.ForgetChild(child)
}

// TableCellOpts are the optional parameters of NewTableCell.
//
// Flutter: widgets/table.dart TableCell.new
type TableCellOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// VerticalAlignment is how this cell is aligned vertically; unspecified uses
	// the default of the table.
	VerticalAlignment rendering.TableCellVerticalAlignment
}

// TableCell controls how a cell of a Table is aligned vertically.
//
// Deviation: the cell is not wrapped in a Semantics with the cell role yet
// (Semantics comes in step 7.7) (SPEC §12.10).
//
// Flutter: widgets/table.dart TableCell
type TableCell struct {
	StatelessWidget
	verticalAlignment rendering.TableCellVerticalAlignment
	child             Widget
}

// NewTableCell creates a cell of a table.
//
// Flutter: widgets/table.dart TableCell.new
func NewTableCell(child Widget, opts ...TableCellOpts) *TableCell {
	o := firstOpts(opts)
	return &TableCell{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), o.VerticalAlignment, child}
}

// VerticalAlignment is how this cell is aligned vertically.
//
// Flutter: widgets/table.dart TableCell.verticalAlignment
func (w *TableCell) VerticalAlignment() rendering.TableCellVerticalAlignment {
	return w.verticalAlignment
}

// Child is the widget in the cell.
//
// Flutter: widgets/table.dart TableCell.child
func (w *TableCell) Child() Widget { return w.child }

// Build builds the parent data widget around the child.
//
// Flutter: widgets/table.dart TableCell.build
func (w *TableCell) Build(BuildContext) Widget {
	return &tableCellData{NewParentDataWidgetBase[*rendering.TableCellParentData](w.child), w.verticalAlignment}
}

// tableCellData writes the vertical alignment of a cell.
//
// Flutter: widgets/table.dart _TableCell
type tableCellData struct {
	ParentDataWidgetOf[*rendering.TableCellParentData]
	verticalAlignment rendering.TableCellVerticalAlignment
}

func (w *tableCellData) ApplyParentData(renderObject rendering.RenderObject) {
	pd := renderObject.ParentData().(*rendering.TableCellParentData)
	if pd.VerticalAlignment != w.verticalAlignment {
		pd.VerticalAlignment = w.verticalAlignment
		if p := renderObject.AsRenderObject().Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

func (*tableCellData) DebugTypicalAncestorWidgetClass() string { return "Table" }
