package rendering

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// RenderEditablePainter paints on the canvas of a RenderEditable, below or above
// its text. It is a Listenable: the editable paints again when it notifies.
//
// Flutter: rendering/editable.dart RenderEditablePainter
type RenderEditablePainter interface {
	foundation.Listenable
	// ShouldRepaint says whether this painter paints differently from the old
	// one (nil when there was none), so that the editable has to paint again.
	//
	// Flutter: rendering/editable.dart RenderEditablePainter.shouldRepaint
	ShouldRepaint(oldDelegate RenderEditablePainter) bool
	// Paint paints on the canvas, in the coordinate space of the viewport of
	// the editable, which has the given size.
	//
	// Flutter: rendering/editable.dart RenderEditablePainter.paint
	Paint(canvas *ui.Canvas, size ui.Size, renderEditable *RenderEditable)
}

// textHighlightPainter paints a highlight behind a range of the text.
//
// Flutter: rendering/editable.dart _TextHighlightPainter
type textHighlightPainter struct {
	foundation.ChangeNotifier
	highlightColor       foundation.Opt[ui.Color]
	highlightedRange     foundation.Opt[ui.TextRange]
	selectionHeightStyle ui.BoxHeightStyle
	selectionWidthStyle  ui.BoxWidthStyle
}

func newTextHighlightPainter() *textHighlightPainter {
	return &textHighlightPainter{selectionHeightStyle: ui.BoxHeightStyleTight, selectionWidthStyle: ui.BoxWidthStyleTight}
}

func (p *textHighlightPainter) setHighlightColor(v foundation.Opt[ui.Color]) {
	if v == p.highlightColor {
		return
	}
	p.highlightColor = v
	p.NotifyListeners()
}

func (p *textHighlightPainter) setHighlightedRange(v foundation.Opt[ui.TextRange]) {
	if v == p.highlightedRange {
		return
	}
	p.highlightedRange = v
	p.NotifyListeners()
}

func (p *textHighlightPainter) setSelectionHeightStyle(v ui.BoxHeightStyle) {
	if p.selectionHeightStyle == v {
		return
	}
	p.selectionHeightStyle = v
	p.NotifyListeners()
}

func (p *textHighlightPainter) setSelectionWidthStyle(v ui.BoxWidthStyle) {
	if p.selectionWidthStyle == v {
		return
	}
	p.selectionWidthStyle = v
	p.NotifyListeners()
}

// Paint fills the boxes of the range with the highlight color.
func (p *textHighlightPainter) Paint(canvas *ui.Canvas, _ ui.Size, renderEditable *RenderEditable) {
	rng, okRange := p.highlightedRange.Get()
	color, okColor := p.highlightColor.Get()
	if !okRange || !okColor || rng.IsCollapsed() {
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(color)
	textPainter := renderEditable.textPainter
	boxes := textPainter.GetBoxesForSelection(rng, ui.BoxesForRangeOpts{HeightStyle: p.selectionHeightStyle, WidthStyle: p.selectionWidthStyle})
	seen := map[ui.TextBox]bool{}
	bounds := ui.NewRectFromLTWH(0, 0, textPainter.Width(), textPainter.Height())
	for _, box := range boxes {
		if seen[box] {
			continue
		}
		seen[box] = true
		canvas.DrawRect(box.ToRect().Shift(renderEditable.paintOffset()).Intersect(bounds), paint)
	}
}

// ShouldRepaint is true if the color, the range or the styles differ.
func (p *textHighlightPainter) ShouldRepaint(oldDelegate RenderEditablePainter) bool {
	if oldDelegate == RenderEditablePainter(p) {
		return false
	}
	if oldDelegate == nil {
		return p.highlightColor.IsSome() && p.highlightedRange.IsSome()
	}
	old, ok := oldDelegate.(*textHighlightPainter)
	return !ok || old.highlightColor != p.highlightColor || old.highlightedRange != p.highlightedRange ||
		old.selectionHeightStyle != p.selectionHeightStyle || old.selectionWidthStyle != p.selectionWidthStyle
}

// caretPainter paints the caret of a collapsed selection.
//
// Deviation: Flutter's also paints the floating cursor of iOS; it is not ported
// (SPEC §12.10).
//
// Flutter: rendering/editable.dart _CaretPainter
type caretPainter struct {
	foundation.ChangeNotifier
	shouldPaint           bool
	caretColor            foundation.Opt[ui.Color]
	cursorRadius          foundation.Opt[ui.Radius]
	cursorOffset          ui.Offset
	backgroundCursorColor foundation.Opt[ui.Color]
}

func newCaretPainter() *caretPainter { return &caretPainter{shouldPaint: true} }

func (p *caretPainter) setShouldPaint(v bool) {
	if p.shouldPaint == v {
		return
	}
	p.shouldPaint = v
	p.NotifyListeners()
}

func (p *caretPainter) setCaretColor(v foundation.Opt[ui.Color]) {
	if p.caretColor == v {
		return
	}
	p.caretColor = v
	p.NotifyListeners()
}

func (p *caretPainter) setCursorRadius(v foundation.Opt[ui.Radius]) {
	if p.cursorRadius == v {
		return
	}
	p.cursorRadius = v
	p.NotifyListeners()
}

func (p *caretPainter) setCursorOffset(v ui.Offset) {
	if p.cursorOffset == v {
		return
	}
	p.cursorOffset = v
	p.NotifyListeners()
}

func (p *caretPainter) setBackgroundCursorColor(v foundation.Opt[ui.Color]) {
	p.backgroundCursorColor = v
}

// Paint computes the rectangle of the caret (even when it is not painted) and
// paints it.
func (p *caretPainter) Paint(canvas *ui.Canvas, _ ui.Size, renderEditable *RenderEditable) {
	sel, ok := renderEditable.selection.Get()
	if !ok || !sel.IsCollapsed() || !sel.IsValid() {
		return
	}
	color, ok := p.caretColor.Get()
	if !ok {
		return
	}
	integralRect := renderEditable.GetLocalRectForCaret(sel.Extent())
	if !p.shouldPaint {
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(color)
	if radius, ok := p.cursorRadius.Get(); ok {
		canvas.DrawRRect(ui.NewRRectFromRectAndRadius(integralRect, radius), paint)
	} else {
		canvas.DrawRect(integralRect, paint)
	}
}

// ShouldRepaint is true if what the caret looks like changed.
func (p *caretPainter) ShouldRepaint(oldDelegate RenderEditablePainter) bool {
	if oldDelegate == RenderEditablePainter(p) {
		return false
	}
	if oldDelegate == nil {
		return p.shouldPaint
	}
	old, ok := oldDelegate.(*caretPainter)
	return !ok || old.shouldPaint != p.shouldPaint || old.caretColor != p.caretColor || old.cursorRadius != p.cursorRadius ||
		old.cursorOffset != p.cursorOffset || old.backgroundCursorColor != p.backgroundCursorColor
}

// compositeRenderEditablePainter paints with several painters, in order.
//
// Flutter: rendering/editable.dart _CompositeRenderEditablePainter
type compositeRenderEditablePainter struct {
	foundation.Listenable
	painters []RenderEditablePainter
}

func newCompositeRenderEditablePainter(painters ...RenderEditablePainter) *compositeRenderEditablePainter {
	listenables := make([]foundation.Listenable, len(painters))
	for i, p := range painters {
		listenables[i] = p
	}
	return &compositeRenderEditablePainter{Listenable: foundation.MergeListenables(listenables...), painters: painters}
}

// Paint calls the painters in order.
func (p *compositeRenderEditablePainter) Paint(canvas *ui.Canvas, size ui.Size, renderEditable *RenderEditable) {
	for _, painter := range p.painters {
		painter.Paint(canvas, size, renderEditable)
	}
}

// ShouldRepaint is true if any painter should repaint.
func (p *compositeRenderEditablePainter) ShouldRepaint(oldDelegate RenderEditablePainter) bool {
	if oldDelegate == RenderEditablePainter(p) {
		return false
	}
	old, ok := oldDelegate.(*compositeRenderEditablePainter)
	if !ok || len(old.painters) != len(p.painters) {
		return true
	}
	for i := range p.painters {
		if p.painters[i].ShouldRepaint(old.painters[i]) {
			return true
		}
	}
	return false
}

// renderEditableCustomPaint is the render object that runs the painters of a
// RenderEditable, in a repaint boundary of its own.
//
// Flutter: rendering/editable.dart _RenderEditableCustomPaint
type renderEditableCustomPaint struct {
	RenderBox
	painter       RenderEditablePainter
	painterHandle foundation.ListenerHandle
}

func newRenderEditableCustomPaint(painter RenderEditablePainter) *renderEditableCustomPaint {
	r := &renderEditableCustomPaint{painter: painter}
	InitRenderBox(&r.RenderBox, r)
	return r
}

// IsRepaintBoundary is true.
func (r *renderEditableCustomPaint) IsRepaintBoundary() bool { return true }

// SizedByParent is true.
func (r *renderEditableCustomPaint) SizedByParent() bool { return true }

// ComputeDryLayout is the biggest size.
func (r *renderEditableCustomPaint) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return constraints.Biggest()
}

func (r *renderEditableCustomPaint) setPainter(newValue RenderEditablePainter) {
	if newValue == r.painter {
		return
	}
	old := r.painter
	r.painter = newValue
	if newValue == nil || newValue.ShouldRepaint(old) {
		r.MarkNeedsPaint()
	}
	if r.Attached() {
		if old != nil {
			old.RemoveListener(r.painterHandle)
		}
		if newValue != nil {
			r.painterHandle = newValue.AddListener(r.MarkNeedsPaint)
		}
	}
}

// Paint runs the painter, after the text of the parent is laid out.
func (r *renderEditableCustomPaint) Paint(context *PaintingContext, _ ui.Offset) {
	parent, _ := r.Parent().(*RenderEditable)
	if r.painter != nil && parent != nil {
		parent.computeTextMetricsIfNeeded()
		r.painter.Paint(context.Canvas(), r.Size(), parent)
	}
}

// Attach listens to the painter.
func (r *renderEditableCustomPaint) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	if r.painter != nil {
		r.painterHandle = r.painter.AddListener(r.MarkNeedsPaint)
	}
}

// Detach stops listening to the painter.
func (r *renderEditableCustomPaint) Detach() {
	if r.painter != nil {
		r.painter.RemoveListener(r.painterHandle)
	}
	r.RenderBox.Detach()
}
