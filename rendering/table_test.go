package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func tableSizedBox(w, h float64) rendering.Box {
	return rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: w, Height: h}), nil)
}

func newLTRTable(opts ...rendering.RenderTableOpts) *rendering.RenderTable {
	return rendering.NewRenderTable(ui.TextDirectionLTR, opts...)
}

func positioned() rendering.Box { return rendering.NewRenderPositionedBox(nil) }

func TestTableControlTight(t *testing.T) {
	table := newLTRTable()
	layout(t, newHarness(t), table)
	if table.Size() != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("size = %v, want 800x600", table.Size())
	}
	skipUnlessDebug(t)
	// Flutter's dump also has a "semantic boundary" line after the constraints;
	// that is added with the semantics of step 6.7.
	want := "RenderTable#00000 NEEDS-PAINT\n" +
		" │ parentData: <none>\n" +
		" │ constraints: BoxConstraints(w=800.0, h=600.0)\n" +
		" │ size: Size(800.0, 600.0)\n" +
		" │ default column width: FlexColumnWidth(1.0)\n" +
		" │ table size: 0×0\n" +
		" │ column offsets: unknown\n" +
		" │ row offsets: []\n" +
		" │\n" +
		" └─table is empty\n"
	if got := deepInfo(table); got != want {
		t.Errorf("toStringDeep:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableControlLoose(t *testing.T) {
	table := newLTRTable()
	layout(t, newHarness(t), rendering.NewRenderPositionedBox(table))
	if table.Size() != (ui.Size{}) {
		t.Errorf("size = %v, want zero", table.Size())
	}
}

func TestTableConstrainedFlexColumns(t *testing.T) {
	table := newLTRTable()
	var children []rendering.Box
	for i := 0; i < 6; i++ {
		children = append(children, positioned())
	}
	table.SetFlatChildren(6, children)
	layout(t, newHarness(t), table, withConstraints(tightFor(some(100), foundation.None[float64]())))
	for i, c := range children {
		if got := c.Size().Width; got < 100.0/6-1e-9 || got > 100.0/6+1e-9 {
			t.Errorf("child %d width = %v, want %v", i, got, 100.0/6)
		}
	}
}

func TestTableCombinations(t *testing.T) {
	table := newLTRTable(rendering.RenderTableOpts{
		Columns: foundation.Some(5), Rows: foundation.Some(5), DefaultColumnWidth: rendering.NewIntrinsicColumnWidth(),
		DefaultVerticalAlignment: rendering.TableCellVerticalAlignmentBaseline, TextBaseline: ui.TextBaselineAlphabetic,
	})
	h := newHarness(t)
	layout(t, h, rendering.NewRenderPositionedBox(table))
	if table.Size() != (ui.Size{}) {
		t.Fatalf("empty size = %v", table.Size())
	}
	table.SetChild(2, 4, tableSizedBox(100, 200))
	pumpFrame(h, phaseLayout)
	if table.Size() != (ui.Size{Width: 100, Height: 200}) {
		t.Fatalf("size = %v, want 100x200", table.Size())
	}
	table.SetChild(0, 0, tableSizedBox(10, 30))
	table.SetChild(1, 0, tableSizedBox(20, 20))
	table.SetChild(2, 0, tableSizedBox(30, 10))
	pumpFrame(h, phaseLayout)
	if table.Size() != (ui.Size{Width: 130, Height: 230}) {
		t.Fatalf("size = %v, want 130x230", table.Size())
	}
	skipUnlessDebug(t)
	got := deepInfo(table)
	for _, want := range []string{
		"RenderTable#00000 relayoutBoundary=up1 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE\n",
		" │ default column width: IntrinsicColumnWidth(flex: null)\n",
		" │ table size: 5×5\n",
		" │ column offsets: 0.0, 10.0, 30.0, 130.0, 130.0\n",
		" │ row offsets: 0.0, 30.0, 30.0, 30.0, 30.0, 230.0\n",
		" ├─child (0, 0): RenderConstrainedBox#00000 relayoutBoundary=up2 NEEDS-PAINT\n",
		" │   constraints: BoxConstraints(w=100.0, 0.0<=h<=Infinity)\n",
		" ├─child (3, 0) is null\n",
		" └─child (4, 4) is null\n",
	} {
		if !containsStr(got, want) {
			t.Errorf("the dump does not contain %q:\n%s", want, got)
		}
	}
}

func TestTableRemovingCells(t *testing.T) {
	table := newLTRTable(rendering.RenderTableOpts{Columns: foundation.Some(5), Rows: foundation.Some(5)})
	child := tableSizedBox(10, 10)
	table.SetChild(4, 4, child)
	layout(t, newHarness(t), table)
	if !child.AsRenderObject().Attached() {
		t.Fatal("the child is not attached")
	}
	table.SetRows(4)
	if child.AsRenderObject().Attached() {
		t.Error("the child is still attached after its row was removed")
	}
}

func TestTableReplacingCells(t *testing.T) {
	child1, child2, child3 := positioned(), positioned(), positioned()
	table := newLTRTable()
	table.SetFlatChildren(3, []rendering.Box{child1, positioned(), child2, positioned(), child3, positioned()})
	if table.Rows() != 2 {
		t.Fatalf("rows = %d, want 2", table.Rows())
	}
	h := newHarness(t)
	layout(t, h, table)
	for i := 0; i < 2; i++ {
		table.SetFlatChildren(3, []rendering.Box{positioned(), child1, positioned(), child2, positioned(), child3})
		pumpFrame(h, phaseLayout)
	}
	if table.Columns() != 3 || table.Rows() != 2 {
		t.Errorf("table is %dx%d, want 3x2", table.Columns(), table.Rows())
	}
	for i, c := range []rendering.Box{child1, child2, child3} {
		if !c.AsRenderObject().Attached() {
			t.Errorf("child %d lost", i+1)
		}
	}
}

func TestTableBorderPainting(t *testing.T) {
	border := rendering.NewTableBorderAll()
	table := rendering.NewRenderTable(ui.TextDirectionRTL, rendering.RenderTableOpts{Border: &border})
	h := newHarness(t)
	layout(t, h, table)
	steps := []struct {
		columns int
		cells   int
		paths   int
	}{{1, 0, 4}, {1, 1, 4}, {1, 2, 5}, {2, 2, 5}, {2, 4, 6}, {3, 6, 6}}
	for _, s := range steps {
		cells := make([]rendering.Box, s.cells)
		for i := range cells {
			cells[i] = positioned()
		}
		table.SetFlatChildren(s.columns, cells)
		pumpFrame(h, phaseLayout)
		if got := countOps[ui.OpDrawPath](recordOps(table)); got != s.paths {
			t.Errorf("%d columns, %d cells: %d paths, want %d", s.columns, s.cells, got, s.paths)
		}
	}
}

func TestTableFlexSizing(t *testing.T) {
	var cells []rendering.Box
	widths := map[int]rendering.TableColumnWidth{0: rendering.NewFlexColumnWidth()}
	for i := 0; i < 7; i++ {
		cells = append(cells, rendering.NewRenderConstrainedBox(square100, nil))
		if i > 0 {
			widths[i] = rendering.NewFlexColumnWidth(rendering.FlexColumnWidthOpts{Value: some(0.123)})
		}
	}
	table := rendering.NewRenderTable(ui.TextDirectionRTL, rendering.RenderTableOpts{Children: [][]rendering.Box{cells}, ColumnWidths: widths})
	layout(t, newHarness(t), table, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 800, Height: 600})))
	if !table.HasSize() {
		t.Error("the table has no size")
	}
}

func TestTablePaintsBorderRadius(t *testing.T) {
	border := rendering.NewTableBorderAll(rendering.TableBorderAllOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(8))})
	table := newLTRTable(rendering.RenderTableOpts{Border: &border})
	h := newHarness(t)
	layout(t, h, table)
	table.SetFlatChildren(2, []rendering.Box{positioned(), positioned(), positioned(), positioned()})
	pumpFrame(h, phaseLayout)
	var drrect *ui.OpDrawDRRect
	for _, op := range recordOps(table) {
		if o, ok := op.(ui.OpDrawDRRect); ok {
			drrect = &o
		}
	}
	if drrect == nil {
		t.Fatal("no DRRect was drawn")
	}
	if want := ui.NewRRectFromLTRBR(0, 0, 800, 0, ui.NewRadiusCircular(8)); drrect.Outer != want {
		t.Errorf("outer = %v, want %v", drrect.Outer, want)
	}
	if want := ui.NewRRectFromLTRBR(1, 1, 799, -1, ui.NewRadiusCircular(7)); drrect.Inner != want {
		t.Errorf("inner = %v, want %v", drrect.Inner, want)
	}
}

func TestTableColumnWidths(t *testing.T) {
	fixed, flex := rendering.NewFixedColumnWidth(100), rendering.NewFlexColumnWidth()
	for name, w := range map[string]rendering.TableColumnWidth{
		"max, fixed first": rendering.NewMaxColumnWidth(fixed, flex), "max, flex first": rendering.NewMaxColumnWidth(flex, fixed),
		"min, fixed first": rendering.NewMinColumnWidth(fixed, flex), "min, flex first": rendering.NewMinColumnWidth(flex, fixed),
	} {
		if got := w.Flex(nil); got != foundation.Some(1.0) {
			t.Errorf("%s: flex = %v, want 1", name, got)
		}
	}
	if got := rendering.NewMaxColumnWidth(rendering.NewFlexColumnWidth(rendering.FlexColumnWidthOpts{Value: some(3)}), flex).Flex(nil); got != foundation.Some(3.0) {
		t.Errorf("max of flexes = %v, want 3", got)
	}
	if got := rendering.NewMinColumnWidth(rendering.NewFlexColumnWidth(rendering.FlexColumnWidthOpts{Value: some(3)}), flex).Flex(nil); got != foundation.Some(1.0) {
		t.Errorf("min of flexes = %v, want 1", got)
	}
	tests := []struct {
		w          rendering.TableColumnWidth
		container  float64
		min, max   float64
		wantString string
	}{
		{rendering.NewFixedColumnWidth(5), 200, 5, 5, "FixedColumnWidth(5.0)"},
		{rendering.NewFractionColumnWidth(0.25), 200, 50, 50, "FractionColumnWidth(0.25)"},
		{rendering.NewFractionColumnWidth(0.25), inf, 0, 0, "FractionColumnWidth(0.25)"},
		{rendering.NewFlexColumnWidth(), 200, 0, 0, "FlexColumnWidth(1.0)"},
		{rendering.NewIntrinsicColumnWidth(rendering.IntrinsicColumnWidthOpts{Flex: some(2)}), 200, 0, 0, "IntrinsicColumnWidth(flex: 2.0)"},
		{rendering.NewMaxColumnWidth(rendering.NewFixedColumnWidth(5), rendering.NewFixedColumnWidth(9)), 200, 9, 9, "MaxColumnWidth(FixedColumnWidth(5.0), FixedColumnWidth(9.0))"},
		{rendering.NewMinColumnWidth(rendering.NewFixedColumnWidth(5), rendering.NewFixedColumnWidth(9)), 200, 5, 5, "MinColumnWidth(FixedColumnWidth(5.0), FixedColumnWidth(9.0))"},
	}
	for _, tc := range tests {
		t.Run(tc.wantString, func(t *testing.T) {
			if got := tc.w.MinIntrinsicWidth(nil, tc.container); got != tc.min {
				t.Errorf("min = %v, want %v", got, tc.min)
			}
			if got := tc.w.MaxIntrinsicWidth(nil, tc.container); got != tc.max {
				t.Errorf("max = %v, want %v", got, tc.max)
			}
			if got := tc.w.String(); got != tc.wantString {
				t.Errorf("String() = %q, want %q", got, tc.wantString)
			}
		})
	}
	t.Run("intrinsic widths of cells", func(t *testing.T) {
		cells := []rendering.Box{tableSizedBox(10, 1), tableSizedBox(30, 1)}
		w := rendering.NewIntrinsicColumnWidth()
		if w.MinIntrinsicWidth(cells, 0) != 30 || w.MaxIntrinsicWidth(cells, 0) != 30 {
			t.Error("intrinsic column width is not the widest cell")
		}
	})
}

func TestTableIntrinsicHeightAlignment(t *testing.T) {
	table := rendering.NewRenderTable(ui.TextDirectionRTL, rendering.RenderTableOpts{
		DefaultVerticalAlignment: rendering.TableCellVerticalAlignmentIntrinsicHeight,
		Children: [][]rendering.Box{{
			rendering.NewRenderConstrainedBox(square100, nil),
			rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 200, Height: 200}), nil),
		}},
		ColumnWidths: map[int]rendering.TableColumnWidth{0: rendering.NewFlexColumnWidth(), 1: rendering.NewFlexColumnWidth()},
	})
	size := ui.Size{Width: 300, Height: 300}
	layout(t, newHarness(t), table, withConstraints(rendering.NewBoxConstraintsTight(size)))
	if table.Size() != size {
		t.Errorf("size = %v, want %v", table.Size(), size)
	}
	if table.DefaultVerticalAlignment() != rendering.TableCellVerticalAlignmentIntrinsicHeight {
		t.Errorf("default alignment = %v", table.DefaultVerticalAlignment())
	}
}

func TestTableEmptyIntrinsicDimensions(t *testing.T) {
	table := newLTRTable()
	if table.Rows() != 0 || table.Columns() != 0 {
		t.Fatalf("table is %dx%d", table.Columns(), table.Rows())
	}
	for _, extent := range []float64{100, inf} {
		if table.GetMinIntrinsicWidth(extent) != 0 || table.GetMaxIntrinsicWidth(extent) != 0 ||
			table.GetMinIntrinsicHeight(extent) != 0 || table.GetMaxIntrinsicHeight(extent) != 0 {
			t.Errorf("an intrinsic dimension of an empty table is not 0 for %v", extent)
		}
	}
}

func TestTableCellVerticalAlignments(t *testing.T) {
	tests := []struct {
		alignment rendering.TableCellVerticalAlignment
		wantDy    float64
		wantH     float64
	}{
		{rendering.TableCellVerticalAlignmentTop, 0, 20},
		{rendering.TableCellVerticalAlignmentMiddle, 20, 20},
		{rendering.TableCellVerticalAlignmentBottom, 40, 20},
		{rendering.TableCellVerticalAlignmentIntrinsicHeight, 0, 60},
	}
	for _, tc := range tests {
		t.Run(tc.alignment.String(), func(t *testing.T) {
			// The tall cell sizes the row; the short one is aligned in it.
			tall, short := tableSizedBox(50, 60), tableSizedBox(50, 20)
			table := newLTRTable(rendering.RenderTableOpts{Children: [][]rendering.Box{{tall, short}},
				DefaultColumnWidth: rendering.NewFixedColumnWidth(50)})
			flexData := short.ParentData().(*rendering.TableCellParentData)
			flexData.VerticalAlignment = tc.alignment
			layout(t, newHarness(t), rendering.NewRenderPositionedBox(table))
			if got := offsetOf(short).Dy; got != tc.wantDy {
				t.Errorf("short cell dy = %v, want %v", got, tc.wantDy)
			}
			if got := short.Size().Height; got != tc.wantH {
				t.Errorf("short cell height = %v, want %v", got, tc.wantH)
			}
			if got := flexData.String(); !containsStr(got, tc.alignment.String()) {
				t.Errorf("parent data %q does not name the alignment", got)
			}
		})
	}
	t.Run("fill", func(t *testing.T) {
		// A filling cell does not size the row: the other cells do.
		filler, other := tableSizedBox(50, 5), tableSizedBox(50, 40)
		table := newLTRTable(rendering.RenderTableOpts{Children: [][]rendering.Box{{filler, other}},
			DefaultColumnWidth: rendering.NewFixedColumnWidth(50)})
		filler.ParentData().(*rendering.TableCellParentData).VerticalAlignment = rendering.TableCellVerticalAlignmentFill
		layout(t, newHarness(t), rendering.NewRenderPositionedBox(table))
		if filler.Size() != (ui.Size{Width: 50, Height: 40}) {
			t.Errorf("filling cell size = %v, want 50x40", filler.Size())
		}
	})
}

func TestTableRTLColumnOrder(t *testing.T) {
	a, b := tableSizedBox(30, 10), tableSizedBox(50, 10)
	table := rendering.NewRenderTable(ui.TextDirectionRTL, rendering.RenderTableOpts{
		Children: [][]rendering.Box{{a, b}}, DefaultColumnWidth: rendering.NewIntrinsicColumnWidth()})
	layout(t, newHarness(t), rendering.NewRenderPositionedBox(table))
	if offsetOf(a).Dx != 50 || offsetOf(b).Dx != 0 {
		t.Errorf("cell offsets = %v and %v, want x 50 and 0", offsetOf(a), offsetOf(b))
	}
	if got := table.GetRowBox(0); got != ui.NewRectFromLTRB(0, 0, 80, 10) {
		t.Errorf("row box = %v", got)
	}
}

func TestTableErrors(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("row with the wrong number of cells", func(t *testing.T) {
		table := newLTRTable(rendering.RenderTableOpts{Columns: foundation.Some(2)})
		requireTag(t, "TableCellCountMismatch", func() { table.AddRow([]rendering.Box{positioned()}) })
	})
	t.Run("negative columns", func(t *testing.T) {
		requireTag(t, "TableNegativeCount", func() { newLTRTable().SetColumns(-1) })
	})
	t.Run("rows and children", func(t *testing.T) {
		requireTag(t, "TableRowsAndChildren", func() {
			newLTRTable(rendering.RenderTableOpts{Rows: foundation.Some(1), Children: [][]rendering.Box{{positioned()}}})
		})
	})
	t.Run("cell out of range", func(t *testing.T) {
		requireTag(t, "TableCellOutOfRange", func() { newLTRTable().SetChild(0, 0, positioned()) })
	})
	t.Run("baseline without text baseline", func(t *testing.T) {
		table := newLTRTable(rendering.RenderTableOpts{Children: [][]rendering.Box{{tableSizedBox(1, 1)}},
			DefaultVerticalAlignment: rendering.TableCellVerticalAlignmentBaseline})
		requireTag(t, "TableBaselineMissing", func() { layout(t, newHarness(t), table) })
	})
	t.Run("invalid column width", func(t *testing.T) {
		table := newLTRTable(rendering.RenderTableOpts{Children: [][]rendering.Box{{tableSizedBox(1, 1)}},
			DefaultColumnWidth: rendering.NewFixedColumnWidth(-1)})
		requireTag(t, "TableColumnWidthInvalid", func() { layout(t, newHarness(t), table) })
	})
	t.Run("border out of range", func(t *testing.T) {
		b := rendering.NewTableBorderAll()
		requireTag(t, "TableBorderOutOfRange", func() {
			b.Paint(ui.NewCanvas(ui.NewPictureRecorder()), ui.NewRectFromLTWH(0, 0, 10, 10), []float64{20}, nil)
		})
	})
}

func TestTableBaselineAlignment(t *testing.T) {
	a := newBaselineGridBox(1, atTen)
	b := newBaselineGridBox(1, func(float64) foundation.Opt[float64] { return foundation.Some(5.0) })
	table := newLTRTable(rendering.RenderTableOpts{Children: [][]rendering.Box{{a, b}},
		DefaultVerticalAlignment: rendering.TableCellVerticalAlignmentBaseline, TextBaseline: ui.TextBaselineAlphabetic,
		DefaultColumnWidth: rendering.NewFixedColumnWidth(10)})
	layout(t, newHarness(t), rendering.NewRenderPositionedBox(table))
	if offsetOf(a).Dy != 0 || offsetOf(b).Dy != 5 {
		t.Errorf("offsets = %v and %v, want dy 0 and 5", offsetOf(a), offsetOf(b))
	}
	if got := table.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic); got != foundation.Some(10.0) {
		t.Errorf("baseline = %v, want 10", got)
	}
	skipUnlessDebug(t)
	if got := table.GetDryBaseline(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}, ui.TextBaselineAlphabetic); got != foundation.Some(10.0) {
		t.Errorf("dry baseline = %v, want 10", got)
	}
}

func TestTableRowBoxNeedsLayout(t *testing.T) {
	skipUnlessDebug(t)
	table := newLTRTable(rendering.RenderTableOpts{Children: [][]rendering.Box{{tableSizedBox(1, 1)}}})
	requireTag(t, "TableNeedsLayout", func() { table.GetRowBox(0) })
	requireTag(t, "BaselineNeedsLayout", func() { table.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic) })
}
