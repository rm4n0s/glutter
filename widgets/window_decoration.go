package widgets

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// WindowDecorationOpts are the optional parameters of NewWindowDecoration.
//
// Flutter: no counterpart (the title bar of the GTK embedder)
type WindowDecorationOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Title is the widget in the title bar; the default is the title of the window
	// in the text style of the bar.
	Title Widget
	// Leading are widgets at the start of the bar, before the title. They are
	// not part of the drag area: a drag, a double tap and a right click on them
	// are theirs alone. They are laid out as one row of minimum width after the
	// Actions, with the room that is left after the gap of 12 pixels and the
	// Actions as the maximum width: a widget without a width of its own gets that
	// room as its limit, a Flexible takes what it needs up to it, an Expanded
	// takes all of it, and the title gets the rest (which may be nothing).
	Leading []Widget
	// Actions are widgets in the bar before the buttons of the window. Like
	// Leading, they are not part of the drag area.
	Actions []Widget
	// BackgroundColor is the color of the bar; the default is a light or a dark
	// surface, by the brightness of the platform.
	BackgroundColor foundation.Opt[ui.Color]
	// ForegroundColor is the color of the title and the buttons.
	ForegroundColor foundation.Opt[ui.Color]
	// Height is the height of the bar; the default is 40.
	Height foundation.Opt[float64]
	// BorderColor is the color of the line around the window; the default is a
	// gray that suits the brightness.
	BorderColor foundation.Opt[ui.Color]
	// ResizeBorder is the width of the strip along each edge where the pointer
	// starts a resize; the default is 6.
	ResizeBorder foundation.Opt[float64]
}

// WindowDecoration is the title bar and the borders that Glutter draws for a
// window when the compositor and libdecor do not (SPEC §8.2). It shows the
// title and the buttons that the compositor allows (minimize, maximize or
// restore, close); a drag of the title or of the empty space around it moves the
// window, a double tap there maximizes or restores it, a right click there opens
// the window menu (the widgets in Leading and Actions have their own gestures
// only), and a press on the border starts a resize. When another decoration
// draws the title bar (the window is decorated by the compositor or by libdecor,
// or has none) it shows the child alone, so an app can place it without checking.
//
// The default RunApp wraps the app in one when the window needs it; an app that
// sets WindowConfig.Decorations to DecorationsApp places its own.
//
// Deviation: the window has no drop shadow, so no area around the content is
// excluded with xdg_surface.set_window_geometry (SPEC §8.2, §12.10).
//
// Flutter: no counterpart (the title bar of the GTK embedder)
type WindowDecoration struct {
	StatefulWidget
	child Widget
	o     WindowDecorationOpts
}

// NewWindowDecoration creates the decoration around the child.
//
// Flutter: no counterpart
func NewWindowDecoration(child Widget, opts ...WindowDecorationOpts) *WindowDecoration {
	o := firstOpts(opts)
	return &WindowDecoration{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, o: o}
}

// CreateState creates the state.
//
// Flutter: no counterpart
func (w *WindowDecoration) CreateState() State { return &windowDecorationState{} }

type windowDecorationState struct {
	StateOf[*WindowDecoration]
	WidgetsBindingObserverBase
	// childKey keeps the state of the child when the decoration appears or goes
	// away, because the compositor tells the mode only after the first frame.
	childKey *GlobalKey[State]
}

func (s *windowDecorationState) InitState() {
	WidgetsBindingOf(s.Context()).AddObserver(s)
}

func (s *windowDecorationState) Dispose() {
	if b := MaybeWidgetsBindingOf(s.Context()); b != nil {
		b.RemoveObserver(s)
	}
}

// DidChangeMetrics rebuilds when the window changes: the platform reports the
// maximized state and the focus together with the size.
func (s *windowDecorationState) DidChangeMetrics() { s.SetState(func() {}) }

const (
	windowDecorationBarHeight = 40.0
	windowDecorationResize    = 6.0
	windowButtonWidth         = 46.0
	windowDecorationGap       = 12.0
)

// windowDecorationColors are the defaults for a brightness.
func windowDecorationColors(dark bool) (background, foreground, border ui.Color) {
	if dark {
		return ui.NewColor(0xFF2B2930), ui.NewColor(0xFFE6E0E9), ui.NewColor(0xFF49454F)
	}
	return ui.NewColor(0xFFECE6F0), ui.NewColor(0xFF1D1B20), ui.NewColor(0xFFCAC4D0)
}

// Build builds the child, with the bar and the borders when the app draws them.
//
// Flutter: no counterpart
func (s *windowDecorationState) Build(bc BuildContext) Widget {
	w := s.Widget()
	view := MaybeViewOf(bc)
	child := NewKeyedSubtree(w.child, WidgetOpts{Key: s.key()})
	if view == nil || view.Window().Decoration != ui.DecorationApp {
		return child
	}
	info := view.Window()
	dark := MediaQueryPlatformBrightnessOf(bc) == ui.BrightnessDark
	bg, fg, border := windowDecorationColors(dark)
	bg, fg, border = w.o.BackgroundColor.OrElse(bg), w.o.ForegroundColor.OrElse(fg), w.o.BorderColor.OrElse(border)
	if !info.Activated {
		fg = fg.WithOpacity(0.6)
	}
	height := w.o.Height.OrElse(windowDecorationBarHeight)
	edge := w.o.ResizeBorder.OrElse(windowDecorationResize)
	resizable := !info.Maximized && !info.Fullscreen && !info.Tiled

	var body Widget = child
	if !info.Fullscreen {
		// The bar is above the app, so it has the direction and the text style of its own.
		bar := NewDirectionality(ui.TextDirectionLTR, NewDefaultTextStyle(painting.NewTextStyle(painting.TextStyleOpts{
			Color: foundation.Some(fg), FontSize: foundation.Some(14.0), Decoration: foundation.Some(ui.TextDecorationNone), FontWeight: ui.FontWeightW500}),
			newWindowTitleBar(view, info, w.o, bg, fg, height)))
		body = NewColumn([]Widget{bar, NewExpanded(child)}, ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
	}
	stack := []Widget{NewPositionedFill(body)}
	if !info.Maximized && !info.Fullscreen {
		stack = append(stack, NewPositionedFill(NewIgnorePointer(NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(border)})}),
			NewSizedBoxExpand(nil)))))
	}
	if resizable {
		stack = append(stack, resizeHandles(view, edge)...)
	}
	return NewStack(stack, StackOpts{Fit: rendering.StackFitExpand, TextDirection: ui.TextDirectionLTR})
}

func (s *windowDecorationState) key() foundation.Key {
	if s.childKey == nil {
		s.childKey = NewGlobalKey[State]()
	}
	return s.childKey
}

// resizeHandles are the strips along the edges and the corners where a press
// starts a resize.
func resizeHandles(view *ui.View, edge float64) []Widget {
	corner := edge * 2
	handle := func(e ui.ResizeEdge, cursor services.MouseCursor, o PositionedOpts) Widget {
		return NewPositioned(NewMouseRegion(NewListener(NewSizedBoxExpand(nil), ListenerOpts{
			Behavior: rendering.HitTestBehaviorOpaque,
			OnPointerDown: func(*gestures.PointerDownEvent) {
				view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionResize, Edge: e})
			},
		}), MouseRegionOpts{Cursor: cursor, Opaque: foundation.Some(true)}), o)
	}
	some := foundation.Some[float64]
	cur := services.SystemMouseCursors
	return []Widget{
		handle(ui.ResizeEdgeTop, cur.ResizeUpDown, PositionedOpts{Left: some(corner), Right: some(corner), Top: some(0), Height: some(edge)}),
		handle(ui.ResizeEdgeBottom, cur.ResizeUpDown, PositionedOpts{Left: some(corner), Right: some(corner), Bottom: some(0), Height: some(edge)}),
		handle(ui.ResizeEdgeLeft, cur.ResizeLeftRight, PositionedOpts{Left: some(0), Top: some(corner), Bottom: some(corner), Width: some(edge)}),
		handle(ui.ResizeEdgeRight, cur.ResizeLeftRight, PositionedOpts{Right: some(0), Top: some(corner), Bottom: some(corner), Width: some(edge)}),
		handle(ui.ResizeEdgeTopLeft, cur.ResizeUpLeftDownRight, PositionedOpts{Left: some(0), Top: some(0), Width: some(corner), Height: some(corner)}),
		handle(ui.ResizeEdgeTopRight, cur.ResizeUpRightDownLeft, PositionedOpts{Right: some(0), Top: some(0), Width: some(corner), Height: some(corner)}),
		handle(ui.ResizeEdgeBottomLeft, cur.ResizeUpRightDownLeft, PositionedOpts{Left: some(0), Bottom: some(0), Width: some(corner), Height: some(corner)}),
		handle(ui.ResizeEdgeBottomRight, cur.ResizeUpLeftDownRight, PositionedOpts{Right: some(0), Bottom: some(0), Width: some(corner), Height: some(corner)}),
	}
}

// newWindowTitleBar is the bar: the title in the middle of a draggable area, and
// the buttons at the end.
func newWindowTitleBar(view *ui.View, info ui.WindowInfo, o WindowDecorationOpts, bg, fg ui.Color, height float64) Widget {
	title := o.Title
	if title == nil {
		title = NewText(info.Title, TextOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis})
	}
	buttons := []Widget{}
	if info.Capabilities.Minimize {
		buttons = append(buttons, newWindowButton(windowGlyphMinimize, fg, ui.NewColor(0x1F000000), height, "Minimize", func() {
			view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMinimize})
		}))
	}
	if info.Capabilities.Maximize {
		glyph, label, kind := windowGlyphMaximize, "Maximize", ui.WindowActionMaximize
		if info.Maximized {
			glyph, label, kind = windowGlyphRestore, "Restore", ui.WindowActionRestore
		}
		buttons = append(buttons, newWindowButton(glyph, fg, ui.NewColor(0x1F000000), height, label, func() {
			view.PerformWindowAction(ui.WindowAction{Kind: kind})
		}))
	}
	buttons = append(buttons, newWindowButton(windowGlyphClose, fg, ui.NewColor(0xFFC42B1C), height, "Close", func() {
		view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionClose})
	}))

	// Only the draggable area has the gestures of the window: the gap before the
	// Leading widgets and the title with the space around it. Leading and Actions
	// widgets are outside it, so a drag, a double tap or a right click on them
	// is theirs alone, and no double-tap recognizer of the bar waits in their arena.
	drag := func(child Widget) Widget {
		return NewGestureDetector(child, GestureDetectorOpts{
			Behavior: rendering.HitTestBehaviorOpaque,
			OnPanStart: func(details gestures.DragStartDetails) {
				view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMove})
			},
			OnDoubleTap: func() {
				kind := ui.WindowActionMaximize
				if info.Maximized {
					kind = ui.WindowActionRestore
				}
				if info.Capabilities.Maximize {
					view.PerformWindowAction(ui.WindowAction{Kind: kind})
				}
			},
			OnSecondaryTapDown: func(details gestures.TapDownDetails) {
				view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionShowMenu, Position: details.GlobalPosition})
			},
		})
	}
	gap := drag(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(windowDecorationGap), Height: foundation.Some(height)}))
	draggable := drag(NewAlign(title, AlignOpts{Alignment: painting.AlignmentCenterLeft}))
	row := NewContainer(newWindowBar(gap, draggable, o.Leading, o.Actions, height), ContainerOpts{Height: foundation.Some(height)})
	return NewSizedBox(NewColoredBox(bg, NewRow([]Widget{NewExpanded(row), NewRow(buttons)})), SizedBoxOpts{Height: foundation.Some(height)})
}

// windowBar lays out the part of the bar before the window buttons, as Flutter's
// NavigationToolbar does: the Actions first, as one group; then the Leading
// widgets, as one group that gets the room that is left after the gap and the
// Actions; and the draggable area last, with exactly the rest (SPEC §8.2).
type windowBar struct {
	StatelessWidget
	gap, title       Widget
	leading, actions []Widget
	height           float64
}

// newWindowBar creates the bar from the gap, the draggable area (the title with
// its gestures), the Leading and the Actions widgets.
func newWindowBar(gap, title Widget, leading, actions []Widget, height float64) Widget {
	return &windowBar{StatelessWidget: NewStatelessWidgetBase(), gap: gap, title: title, leading: leading, actions: actions, height: height}
}

type windowBarSlot int

const (
	windowBarSlotGap windowBarSlot = iota
	windowBarSlotLeading
	windowBarSlotTitle
	windowBarSlotActions
)

func (w *windowBar) Build(bc BuildContext) Widget {
	d := &windowBarLayout{
		MultiChildLayoutDelegateBase: rendering.NewMultiChildLayoutDelegateBase(nil),
		textDirection:                DirectionalityOf(bc),
		gap:                          windowDecorationGap,
	}
	children := []Widget{NewLayoutId(windowBarSlotGap, w.gap)}
	if len(w.leading) > 0 {
		leading := w.leading
		// The widgets are built for the room the delegate gives the group, so
		// that a widget without a limit of its own gets the room as its maximum.
		children = append(children, NewLayoutId(windowBarSlotLeading, NewLayoutBuilder(func(_ BuildContext, c rendering.BoxConstraints) Widget {
			items := make([]Widget, len(leading))
			for i, item := range leading {
				items[i] = limitWindowBarItem(item, c.MaxWidth)
			}
			return NewRow(items, RowOpts{MainAxisSize: rendering.MainAxisSizeMin})
		})))
	}
	children = append(children, NewLayoutId(windowBarSlotTitle, w.title))
	if len(w.actions) > 0 {
		children = append(children, NewLayoutId(windowBarSlotActions, NewRow(w.actions, RowOpts{MainAxisSize: rendering.MainAxisSizeMin})))
	}
	return NewCustomMultiChildLayout(d, children)
}

// limitWindowBarItem gives a widget of the Leading group the room as its maximum
// width. A Row passes an unlimited width to a child without a flex factor, so
// a widget that fills its width (a Row of maximum size with an Expanded in it)
// would have no limit. Flexible, Expanded and Spacer widgets get the room from
// the Row already. The wrapper has a key made from the widget's key when that
// is a local key (not the same key, so a finder by key finds the widget alone),
// so that reordering keyed widgets keeps their state.
func limitWindowBarItem(item Widget, maxWidth float64) Widget {
	switch item.(type) {
	case *Flexible, *Expanded, *Spacer:
		return item
	}
	var key foundation.Key
	if k, ok := item.Key().(foundation.LocalKey); ok {
		key = windowBarItemKey{inner: k}
	}
	return NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(maxWidth)}), item, ConstrainedBoxOpts{Key: key})
}

// windowBarItemKey is the key of the wrapper of a keyed Leading widget.
type windowBarItemKey struct {
	foundation.LocalKeyBase
	inner foundation.Key
}

func (k windowBarItemKey) String() string { return "[windowBar " + k.inner.String() + "]" }

// windowBarLayout is the delegate of windowBar.
type windowBarLayout struct {
	rendering.MultiChildLayoutDelegateBase
	textDirection ui.TextDirection
	gap           float64
}

func (d *windowBarLayout) PerformLayout(size ui.Size) {
	rtl := d.textDirection == ui.TextDirectionRTL
	place := func(slot windowBarSlot, x, width float64) {
		if rtl {
			x = size.Width - x - width
		}
		d.PositionChild(slot, ui.Offset{Dx: x, Dy: 0})
	}
	// Every child has the height of the bar; a widget smaller than that is
	// centered by its group.
	actionsWidth := 0.0
	var actionsSize ui.Size
	if d.HasChild(windowBarSlotActions) {
		actionsSize = d.LayoutChild(windowBarSlotActions, rendering.BoxConstraints{MaxWidth: size.Width, MinHeight: size.Height, MaxHeight: size.Height})
		actionsWidth = actionsSize.Width
	}
	gapWidth := math.Min(d.gap, size.Width)
	d.LayoutChild(windowBarSlotGap, rendering.BoxConstraints{MaxWidth: gapWidth, MinHeight: size.Height, MaxHeight: size.Height})
	place(windowBarSlotGap, 0, gapWidth)

	leadingWidth := 0.0
	if d.HasChild(windowBarSlotLeading) {
		room := math.Max(0, size.Width-gapWidth-actionsWidth)
		leadingWidth = d.LayoutChild(windowBarSlotLeading, rendering.BoxConstraints{MaxWidth: room, MinHeight: size.Height, MaxHeight: size.Height}).Width
		place(windowBarSlotLeading, gapWidth, leadingWidth)
	}

	titleX := gapWidth + leadingWidth
	titleWidth := math.Max(0, size.Width-titleX-actionsWidth)
	d.LayoutChild(windowBarSlotTitle, rendering.NewBoxConstraintsTight(ui.Size{Width: titleWidth, Height: size.Height}))
	place(windowBarSlotTitle, titleX, titleWidth)

	if d.HasChild(windowBarSlotActions) {
		place(windowBarSlotActions, size.Width-actionsWidth, actionsWidth)
	}
}

func (d *windowBarLayout) ShouldRelayout(old rendering.MultiChildLayoutDelegate) bool {
	o := old.(*windowBarLayout)
	return o.textDirection != d.textDirection || o.gap != d.gap
}

type windowGlyph int

const (
	windowGlyphMinimize windowGlyph = iota
	windowGlyphMaximize
	windowGlyphRestore
	windowGlyphClose
)

// windowButton is a button of the title bar: a glyph that shows a hover color.
type windowButton struct {
	StatefulWidget
	glyph        windowGlyph
	color, hover ui.Color
	height       float64
	label        string
	onTap        func()
}

func newWindowButton(g windowGlyph, color, hover ui.Color, height float64, label string, onTap func()) Widget {
	return &windowButton{StatefulWidget: NewStatefulWidgetBase(), glyph: g, color: color, hover: hover, height: height, label: label, onTap: onTap}
}

func (w *windowButton) CreateState() State { return &windowButtonState{} }

type windowButtonState struct {
	StateOf[*windowButton]
	hovering bool
}

func (s *windowButtonState) Build(BuildContext) Widget {
	w := s.Widget()
	bg := ui.NewColor(0x00000000)
	fg := w.color
	if s.hovering {
		bg = w.hover
		if w.glyph == windowGlyphClose {
			fg = ui.NewColor(0xFFFFFFFF)
		}
	}
	return NewSemantics(NewMouseRegion(NewGestureDetector(NewColoredBox(bg, NewSizedBox(NewCustomPaint(nil, CustomPaintOpts{Painter: &windowGlyphPainter{glyph: w.glyph, color: fg}}),
		SizedBoxOpts{Width: foundation.Some(windowButtonWidth), Height: foundation.Some(w.height)})), GestureDetectorOpts{OnTap: w.onTap, Behavior: rendering.HitTestBehaviorOpaque}),
		MouseRegionOpts{
			Cursor:  services.SystemMouseCursors.Basic,
			OnEnter: func(*gestures.PointerEnterEvent) { s.SetState(func() { s.hovering = true }) },
			OnExit:  func(*gestures.PointerExitEvent) { s.SetState(func() { s.hovering = false }) },
		}), SemanticsOpts{Label: foundation.Some(w.label), Button: foundation.Some(true), OnTap: w.onTap})
}

// windowGlyphPainter draws the symbol of a button of the title bar, 10 pixels wide
// in the middle of its box.
type windowGlyphPainter struct {
	rendering.CustomPainterBase
	glyph windowGlyph
	color ui.Color
}

func (p *windowGlyphPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	paint.SetStrokeWidth(1)
	paint.SetStyle(ui.PaintingStyleStroke)
	cx, cy := size.Width/2, size.Height/2
	const h = 5.0
	switch p.glyph {
	case windowGlyphMinimize:
		canvas.DrawLine(ui.Offset{Dx: cx - h, Dy: cy}, ui.Offset{Dx: cx + h, Dy: cy}, paint)
	case windowGlyphMaximize:
		canvas.DrawRect(ui.NewRectFromLTRB(cx-h, cy-h, cx+h, cy+h), paint)
	case windowGlyphRestore:
		canvas.DrawRect(ui.NewRectFromLTRB(cx-h, cy-h+2, cx+h-2, cy+h), paint)
		canvas.DrawLine(ui.Offset{Dx: cx - h + 2, Dy: cy - h + 2}, ui.Offset{Dx: cx - h + 2, Dy: cy - h}, paint)
		canvas.DrawLine(ui.Offset{Dx: cx - h + 2, Dy: cy - h}, ui.Offset{Dx: cx + h, Dy: cy - h}, paint)
		canvas.DrawLine(ui.Offset{Dx: cx + h, Dy: cy - h}, ui.Offset{Dx: cx + h, Dy: cy + h - 2}, paint)
		canvas.DrawLine(ui.Offset{Dx: cx + h, Dy: cy + h - 2}, ui.Offset{Dx: cx + h - 2, Dy: cy + h - 2}, paint)
	case windowGlyphClose:
		canvas.DrawLine(ui.Offset{Dx: cx - h, Dy: cy - h}, ui.Offset{Dx: cx + h, Dy: cy + h}, paint)
		canvas.DrawLine(ui.Offset{Dx: cx - h, Dy: cy + h}, ui.Offset{Dx: cx + h, Dy: cy - h}, paint)
	}
}

func (p *windowGlyphPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*windowGlyphPainter)
	return !ok || o.glyph != p.glyph || o.color != p.color
}
