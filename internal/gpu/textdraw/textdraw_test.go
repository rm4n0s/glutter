package textdraw

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// painter is a paragraph that emits the given pieces.
type painter struct {
	ui.ParagraphLayout
	draws []ui.ParagraphDraw
}

func (p painter) PaintParagraph(emit func(ui.ParagraphDraw)) {
	for _, d := range p.draws {
		emit(d)
	}
}

// plain is a paragraph that cannot draw itself.
type plain struct{ ui.ParagraphLayout }

type drawn struct {
	op ui.DisplayOp
	m  [16]float32
}

func expand(t *testing.T, para ui.ParagraphLayout, offset ui.Offset, ctm *ui.Matrix4) ([]drawn, bool) {
	t.Helper()
	var out []drawn
	ok := Expand(ui.OpDrawParagraph{Paragraph: ui.NewParagraph(para), Offset: offset}, ctm, func(op ui.DisplayOp, m *ui.Matrix4) {
		out = append(out, drawn{op, *m.Storage()})
	})
	return out, ok
}

func TestExpandPlacesEachPiece(t *testing.T) {
	rect := ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, 1, 1)}
	scale2 := ui.NewMatrix4Identity()
	scale2.Scale(2, 2, 1)
	rot := ui.NewMatrix4Identity()
	rot.RotateZ(math.Pi / 6)

	tests := []struct {
		name         string
		ctm          *ui.Matrix4
		offset       ui.Offset
		draw         ui.ParagraphDraw
		wantX, wantY float32
	}{
		{"offsets add up", ui.NewMatrix4Identity(), ui.Offset{Dx: 10, Dy: 20}, ui.ParagraphDraw{Op: rect, Offset: ui.Offset{Dx: 1, Dy: 2}}, 11, 22},
		{"not snapped without Snap", ui.NewMatrix4Identity(), ui.Offset{Dx: 0.3, Dy: 0.3}, ui.ParagraphDraw{Op: rect}, 0.3, 0.3},
		{"x to a quarter and y to a pixel", ui.NewMatrix4Identity(), ui.Offset{Dx: 0.3, Dy: 0.6}, ui.ParagraphDraw{Op: rect, Snap: true}, 0.25, 1},
		{"x rounds up to the next quarter", ui.NewMatrix4Identity(), ui.Offset{Dx: 0.4, Dy: 0.4}, ui.ParagraphDraw{Op: rect, Snap: true}, 0.5, 0},
		{"snapped in pixels, after the scale", scale2, ui.Offset{Dx: 0.3, Dy: 0.3}, ui.ParagraphDraw{Op: rect, Snap: true}, 0.5, 1},
		{"a rotation keeps the position", rot, ui.Offset{Dx: 0.3, Dy: 0.6}, ui.ParagraphDraw{Op: rect, Snap: true}, 0.3*float32(math.Cos(math.Pi/6)) - 0.6*float32(math.Sin(math.Pi/6)), 0.3*float32(math.Sin(math.Pi/6)) + 0.6*float32(math.Cos(math.Pi/6))},
	}
	for _, tt := range tests {
		got, ok := expand(t, painter{draws: []ui.ParagraphDraw{tt.draw}}, tt.offset, tt.ctm)
		if !ok || len(got) != 1 {
			t.Fatalf("%s: ok=%v, %d pieces", tt.name, ok, len(got))
		}
		if got[0].op != tt.draw.Op {
			t.Errorf("%s: op = %v", tt.name, got[0].op)
		}
		if x, y := got[0].m[12], got[0].m[13]; math.Abs(float64(x-tt.wantX)) > 1e-5 || math.Abs(float64(y-tt.wantY)) > 1e-5 {
			t.Errorf("%s: origin = (%v, %v), want (%v, %v)", tt.name, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestExpandAppliesThePiecesMatrixBeforeItsOffset(t *testing.T) {
	m := ui.NewMatrix4Identity()
	m.Scale(3, 3, 1)
	got, ok := expand(t, painter{draws: []ui.ParagraphDraw{{Op: ui.OpDrawRect{}, Offset: ui.Offset{Dx: 5, Dy: 5}, Matrix: m}}}, ui.Offset{}, ui.NewMatrix4Identity())
	if !ok || len(got) != 1 {
		t.Fatalf("ok=%v, %d pieces", ok, len(got))
	}
	if got[0].m[0] != 3 || got[0].m[5] != 3 || got[0].m[12] != 5 || got[0].m[13] != 5 {
		t.Errorf("transform = %v", got[0].m)
	}
}

func TestExpandKeepsTheOrderAndDoesNotChangeTheTransform(t *testing.T) {
	ctm := ui.NewMatrix4Identity()
	before := *ctm.Storage()
	a, b := ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, 1, 1)}, ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, 2, 2)}
	got, ok := expand(t, painter{draws: []ui.ParagraphDraw{{Op: a, Snap: true}, {Op: b}}}, ui.Offset{Dx: 0.3}, ctm)
	if !ok || len(got) != 2 || got[0].op != a || got[1].op != b {
		t.Fatalf("ok=%v, pieces %v", ok, got)
	}
	if *ctm.Storage() != before {
		t.Error("Expand changed the transform it was given")
	}
}

func TestExpandWithoutAPainter(t *testing.T) {
	got, ok := expand(t, plain{}, ui.Offset{}, ui.NewMatrix4Identity())
	if ok || len(got) != 0 {
		t.Errorf("a paragraph that cannot draw itself: ok=%v, %d pieces", ok, len(got))
	}
	if !Expand(ui.OpDrawParagraph{}, ui.NewMatrix4Identity(), func(ui.DisplayOp, *ui.Matrix4) { t.Error("emitted") }) {
		t.Error("no paragraph is nothing to draw, not a failure")
	}
}

func TestIsAxisAligned(t *testing.T) {
	flip := ui.NewMatrix4Identity()
	flip.Scale(1, -1, 1)
	rot := ui.NewMatrix4Identity()
	rot.RotateZ(0.1)
	persp := ui.NewMatrix4Identity()
	persp.SetEntry(3, 0, 0.001)
	tests := []struct {
		name string
		m    *ui.Matrix4
		want bool
	}{
		{"identity", ui.NewMatrix4Identity(), true},
		{"flip", flip, true},
		{"rotation", rot, false},
		{"perspective", persp, false},
	}
	for _, tt := range tests {
		if got := isAxisAligned(tt.m); got != tt.want {
			t.Errorf("%s: %v", tt.name, got)
		}
	}
}

// rowPainter is a paragraph that paints the rows it is asked for and notes them.
type rowPainter struct {
	painter
	rows  *[2]float64
	asked *int
}

func (p rowPainter) PaintParagraphRows(top, bottom float64, emit func(ui.ParagraphDraw)) {
	*p.rows = [2]float64{top, bottom}
	*p.asked++
	p.PaintParagraph(emit)
}

func TestExpandVisibleAsksForTheRowsOfTheVisibleRectangle(t *testing.T) {
	translate := func(dx, dy float32) *ui.Matrix4 { m := ui.NewMatrix4Identity(); m.Translate(dx, dy, 0); return m }
	scale := func(s float32) *ui.Matrix4 { m := ui.NewMatrix4Identity(); m.Scale(s, s, 1); return m }
	rot := ui.NewMatrix4Identity()
	rot.RotateZ(math.Pi / 2)
	perspective := ui.NewMatrix4Identity()
	perspective.SetEntry(3, 0, 0.01)
	flipped := ui.NewMatrix4Identity()
	flipped.Scale(1, -1, 1)
	visible := ui.NewRectFromLTWH(10, 100, 50, 40)

	tests := []struct {
		name     string
		ctm      *ui.Matrix4
		offset   ui.Offset
		visible  ui.Rect
		asked    bool
		wantRows [2]float64
	}{
		{"the identity", ui.NewMatrix4Identity(), ui.Offset{}, visible, true, [2]float64{100, 140}},
		{"the paragraph's offset is taken off", ui.NewMatrix4Identity(), ui.Offset{Dx: 3, Dy: 25}, visible, true, [2]float64{75, 115}},
		{"a translation", translate(0, -300), ui.Offset{}, visible, true, [2]float64{400, 440}},
		{"a scale", scale(2), ui.Offset{}, visible, true, [2]float64{50, 70}},
		{"a flip", flipped, ui.Offset{}, visible, true, [2]float64{-140, -100}},
		{"a quarter turn covers the rows of the rotated rectangle", rot, ui.Offset{}, visible, true, [2]float64{-60, -10}},
		{"a perspective cannot be inverted for rows", perspective, ui.Offset{}, visible, false, [2]float64{}},
		{"a rectangle that is not finite", ui.NewMatrix4Identity(), ui.Offset{}, ui.Rect{Left: 0, Top: math.Inf(-1), Right: 10, Bottom: math.Inf(1)}, false, [2]float64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rows [2]float64
			asked := 0
			para := rowPainter{painter: painter{draws: []ui.ParagraphDraw{{Op: ui.OpDrawRect{}}, {Op: ui.OpDrawRect{}}}}, rows: &rows, asked: &asked}
			var got int
			ok := ExpandVisible(ui.OpDrawParagraph{Paragraph: ui.NewParagraph(para), Offset: tt.offset}, tt.ctm, tt.visible, func(ui.DisplayOp, *ui.Matrix4) { got++ })
			if !ok || got != 2 {
				t.Fatalf("ok = %v, %d pieces, want true and 2", ok, got)
			}
			if (asked == 1) != tt.asked {
				t.Fatalf("the rows were asked %d times, want asked = %v", asked, tt.asked)
			}
			if tt.asked && (math.Abs(rows[0]-tt.wantRows[0]) > 1e-4 || math.Abs(rows[1]-tt.wantRows[1]) > 1e-4) {
				t.Errorf("rows = %v, want %v", rows, tt.wantRows)
			}
		})
	}
}

func TestExpandVisibleOfAParagraphWithoutRowsPaintsItAll(t *testing.T) {
	got := 0
	ok := ExpandVisible(ui.OpDrawParagraph{Paragraph: ui.NewParagraph(painter{draws: []ui.ParagraphDraw{{Op: ui.OpDrawRect{}}, {Op: ui.OpDrawRect{}}, {Op: ui.OpDrawRect{}}}})},
		ui.NewMatrix4Identity(), ui.NewRectFromLTWH(0, 0, 1, 1), func(ui.DisplayOp, *ui.Matrix4) { got++ })
	if !ok || got != 3 {
		t.Errorf("ok = %v, %d pieces, want true and 3", ok, got)
	}
	if ExpandVisible(ui.OpDrawParagraph{Paragraph: ui.NewParagraph(plain{})}, ui.NewMatrix4Identity(), ui.Rect{}, func(ui.DisplayOp, *ui.Matrix4) {}) {
		t.Error("a paragraph that cannot draw itself was expanded")
	}
}

// boxed is a paragraph that tells the box of what it draws.
type boxed struct {
	painter
	box ui.Rect
}

func (b boxed) PaintBounds() ui.Rect { return b.box }

func TestBoundsOfAParagraph(t *testing.T) {
	tests := []struct {
		name string
		op   ui.OpDrawParagraph
		want ui.Rect
		ok   bool
	}{
		{"a paragraph that knows its box", ui.OpDrawParagraph{Paragraph: ui.NewParagraph(boxed{box: ui.NewRectFromLTWH(-2, -2, 104, 54)}), Offset: ui.Offset{Dx: 10, Dy: 20}}, ui.NewRectFromLTWH(8, 18, 104, 54), true},
		{"a paragraph that does not", ui.OpDrawParagraph{Paragraph: ui.NewParagraph(painter{})}, ui.Rect{}, false},
		{"no paragraph", ui.OpDrawParagraph{}, ui.Rect{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Bounds(tt.op)
			if ok != tt.ok || got != tt.want {
				t.Errorf("Bounds = %v, %v, want %v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExpandHandsOutOneMatrixAtATimeAndAllocatesNone(t *testing.T) {
	draws := make([]ui.ParagraphDraw, 100)
	for i := range draws {
		draws[i] = ui.ParagraphDraw{Op: ui.OpDrawRect{}, Offset: ui.Offset{Dx: float64(i)}, Snap: true}
	}
	op := ui.OpDrawParagraph{Paragraph: ui.NewParagraph(painter{draws: draws})}
	ctm := ui.NewMatrix4Identity()
	var xs []float32
	Expand(op, ctm, func(_ ui.DisplayOp, m *ui.Matrix4) { xs = append(xs, m.Storage()[12]) })
	for i, x := range xs {
		if x != float32(i) {
			t.Fatalf("piece %d is at x = %v", i, x)
		}
	}
	if n := testing.AllocsPerRun(20, func() { Expand(op, ctm, func(ui.DisplayOp, *ui.Matrix4) {}) }); n > 3 {
		t.Errorf("Expand of 100 pieces allocates %v times, want a few at most (not one for each piece)", n)
	}
}
