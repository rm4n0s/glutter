package widgets

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// NavigationToolbarMiddleSpacing is the default spacing around the middle
// widget of a NavigationToolbar.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.kMiddleSpacing
const NavigationToolbarMiddleSpacing = 16.0

// NavigationToolbarOpts are the optional parameters of NewNavigationToolbar.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.new
type NavigationToolbarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Leading is the widget at the start of the toolbar; it is as high as the
	// toolbar.
	Leading Widget
	// Middle is the widget between the leading and the trailing widgets.
	Middle Widget
	// Trailing is the widget at the end of the toolbar.
	Trailing Widget
	// NoCenterMiddle says that the middle widget is start-justified within the
	// space between the leading and trailing widgets, instead of centered within
	// the toolbar (the default, Flutter's centerMiddle).
	NoCenterMiddle bool
	// MiddleSpacing is the spacing around the middle widget on the sides that
	// have a leading or trailing widget; the default is
	// NavigationToolbarMiddleSpacing.
	MiddleSpacing foundation.Opt[float64]
}

// NavigationToolbar is a horizontal layout of a leading widget, a middle widget
// and a trailing widget, as in the toolbar of an AppBar.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar
type NavigationToolbar struct {
	StatelessWidget
	o NavigationToolbarOpts
}

// NewNavigationToolbar creates the toolbar.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.new
func NewNavigationToolbar(opts ...NavigationToolbarOpts) *NavigationToolbar {
	o := firstOpts(opts)
	return &NavigationToolbar{StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), o: o}
}

// Leading is the widget at the start of the toolbar.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.leading
func (w *NavigationToolbar) Leading() Widget { return w.o.Leading }

// Middle is the widget between the leading and the trailing widgets.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.middle
func (w *NavigationToolbar) Middle() Widget { return w.o.Middle }

// Trailing is the widget at the end of the toolbar.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.trailing
func (w *NavigationToolbar) Trailing() Widget { return w.o.Trailing }

// CenterMiddle says whether the middle widget is centered within the toolbar.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.centerMiddle
func (w *NavigationToolbar) CenterMiddle() bool { return !w.o.NoCenterMiddle }

// MiddleSpacing is the spacing around the middle widget.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.middleSpacing
func (w *NavigationToolbar) MiddleSpacing() float64 {
	return w.o.MiddleSpacing.OrElse(NavigationToolbarMiddleSpacing)
}

// Build lays the three widgets out with a CustomMultiChildLayout.
//
// Flutter: widgets/navigation_toolbar.dart NavigationToolbar.build
func (w *NavigationToolbar) Build(bc BuildContext) Widget {
	DebugCheckHasDirectionality(bc)
	d := &toolbarLayout{
		MultiChildLayoutDelegateBase: rendering.NewMultiChildLayoutDelegateBase(nil),
		centerMiddle:                 w.CenterMiddle(),
		middleSpacing:                w.MiddleSpacing(),
		textDirection:                DirectionalityOf(bc),
	}
	var children []Widget
	if w.o.Leading != nil {
		children = append(children, NewLayoutId(toolbarSlotLeading, w.o.Leading))
	}
	if w.o.Middle != nil {
		children = append(children, NewLayoutId(toolbarSlotMiddle, w.o.Middle))
	}
	if w.o.Trailing != nil {
		children = append(children, NewLayoutId(toolbarSlotTrailing, w.o.Trailing))
	}
	return NewCustomMultiChildLayout(d, children)
}

type toolbarSlot int

const (
	toolbarSlotLeading toolbarSlot = iota
	toolbarSlotMiddle
	toolbarSlotTrailing
)

type toolbarLayout struct {
	rendering.MultiChildLayoutDelegateBase
	centerMiddle  bool
	middleSpacing float64
	textDirection ui.TextDirection
}

func (d *toolbarLayout) PerformLayout(size ui.Size) {
	leadingWidth, trailingWidth := 0.0, 0.0
	rtl := d.textDirection == ui.TextDirectionRTL

	if d.HasChild(toolbarSlotLeading) {
		// The height should be exactly the height of the bar.
		c := rendering.BoxConstraints{MaxWidth: size.Width, MinHeight: size.Height, MaxHeight: size.Height}
		leadingWidth = d.LayoutChild(toolbarSlotLeading, c).Width
		leadingX := 0.0
		if rtl {
			leadingX = size.Width - leadingWidth
		}
		d.PositionChild(toolbarSlotLeading, ui.Offset{Dx: leadingX, Dy: 0})
	}

	if d.HasChild(toolbarSlotTrailing) {
		trailingSize := d.LayoutChild(toolbarSlotTrailing, rendering.NewBoxConstraintsLoose(size))
		trailingX := size.Width - trailingSize.Width
		if rtl {
			trailingX = 0
		}
		trailingY := (size.Height - trailingSize.Height) / 2.0
		trailingWidth = trailingSize.Width
		d.PositionChild(toolbarSlotTrailing, ui.Offset{Dx: trailingX, Dy: trailingY})
	}

	if d.HasChild(toolbarSlotMiddle) {
		maxWidth := math.Max(size.Width-leadingWidth-trailingWidth-d.middleSpacing*2.0, 0.0)
		c := rendering.NewBoxConstraintsLoose(size).CopyWith(rendering.BoxConstraintsCopy{MaxWidth: foundation.Some(maxWidth)})
		middleSize := d.LayoutChild(toolbarSlotMiddle, c)

		middleStartMargin := leadingWidth + d.middleSpacing
		middleStart := middleStartMargin
		middleY := (size.Height - middleSize.Height) / 2.0
		// If the centered middle will not fit between the leading and trailing
		// widgets, then align its left or right edge with the adjacent boundary.
		if d.centerMiddle {
			middleStart = (size.Width - middleSize.Width) / 2.0
			if middleStart+middleSize.Width > size.Width-trailingWidth {
				middleStart = size.Width - trailingWidth - middleSize.Width - d.middleSpacing
			} else if middleStart < middleStartMargin {
				middleStart = middleStartMargin
			}
		}
		middleX := middleStart
		if rtl {
			middleX = size.Width - middleSize.Width - middleStart
		}
		d.PositionChild(toolbarSlotMiddle, ui.Offset{Dx: middleX, Dy: middleY})
	}
}

func (d *toolbarLayout) ShouldRelayout(old rendering.MultiChildLayoutDelegate) bool {
	o := old.(*toolbarLayout)
	return o.centerMiddle != d.centerMiddle || o.middleSpacing != d.middleSpacing || o.textDirection != d.textDirection
}
