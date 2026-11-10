package cupertino

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	listTileLeadingSize                      = 28.0
	listTileNotchedLeadingSize               = 30.0
	listTileMinHeight                        = listTileLeadingSize + 2*8.0
	listTileMinHeightWithSubtitle            = listTileLeadingSize + 2*10.0
	listTileNotchedMinHeight                 = listTileNotchedLeadingSize + 2*12.0
	listTileNotchedMinHeightWithoutLead      = listTileNotchedLeadingSize + 2*10.0
	listTileLeadingToTitle                   = 16.0
	listTileNotchedLeadingToTitle            = 12.0
	listTileNotchedTitleToSubtitle           = 3.0
	listTileAdditionalInfoToTrailing         = 6.0
	listTileNotchedTitleWithSubtitleFontSize = 16.0
	listTileSubtitleFontSize                 = 12.0
	listTileNotchedSubtitleFontSize          = 14.0
)

var (
	listTilePadding                   = painting.EdgeInsetsDirectional{Start: 20, End: 14}
	listTileNotchedPadding            = painting.EdgeInsets{Left: 14, Right: 14}
	listTileNotchedPaddingWithoutLead = painting.NewEdgeInsetsDirectionalFromSTEB(28, 10, 14, 10)
)

type listTileType int

const (
	listTileBase listTileType = iota
	listTileNotched
)

// CupertinoListTileOpts are the optional parameters of NewCupertinoListTile and NewCupertinoListTileNotched.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile.new
type CupertinoListTileOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Subtitle is shown under the title.
	Subtitle widgets.Widget
	// AdditionalInfo is shown at the end, before the trailing widget.
	AdditionalInfo widgets.Widget
	// Leading is shown at the start.
	Leading widgets.Widget
	// Trailing is shown at the end.
	Trailing widgets.Widget
	// OnTap is called when the tile is tapped. Nil makes the tile not respond to taps.
	OnTap func()
	// BackgroundColor is the background of the tile; the default is transparent.
	BackgroundColor ui.ColorValue
	// BackgroundColorActivated is the background while the tile is pressed; the default is CupertinoColors.SystemGrey4.
	BackgroundColorActivated ui.ColorValue
	// Padding is the padding of the content; the default is that of the type of the tile.
	Padding painting.EdgeInsetsGeometry
	// LeadingSize is the width and height of the leading widget; the default is 28, or 30 for a notched tile.
	LeadingSize foundation.Opt[float64]
	// LeadingToTitle is the space between the leading widget and the title; the default is 16, or 12 for a notched tile.
	LeadingToTitle foundation.Opt[float64]
}

// CupertinoListTile is an iOS style list tile: a row with a title, an optional subtitle, a leading and a trailing widget and additional info.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile
type CupertinoListTile struct {
	widgets.StatefulWidget
	tileType    listTileType
	title       widgets.Widget
	leadingSize float64
	toTitle     float64
	o           CupertinoListTileOpts
}

// NewCupertinoListTile creates a list tile.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile.new
func NewCupertinoListTile(title widgets.Widget, opts ...CupertinoListTileOpts) *CupertinoListTile {
	o := firstOf(opts)
	return &CupertinoListTile{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), tileType: listTileBase, title: title, o: o,
		leadingSize: o.LeadingSize.OrElse(listTileLeadingSize), toTitle: o.LeadingToTitle.OrElse(listTileLeadingToTitle)}
}

// NewCupertinoListTileNotched creates a list tile in the style of the inset grouped lists of iOS: taller, with a bigger leading widget.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile.notched
func NewCupertinoListTileNotched(title widgets.Widget, opts ...CupertinoListTileOpts) *CupertinoListTile {
	o := firstOf(opts)
	return &CupertinoListTile{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), tileType: listTileNotched, title: title, o: o,
		leadingSize: o.LeadingSize.OrElse(listTileNotchedLeadingSize), toTitle: o.LeadingToTitle.OrElse(listTileNotchedLeadingToTitle)}
}

// Title is the title.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile.title
func (w *CupertinoListTile) Title() widgets.Widget { return w.title }

// LeadingSize is the width and height of the leading widget.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile.leadingSize
func (w *CupertinoListTile) LeadingSize() float64 { return w.leadingSize }

// LeadingToTitle is the space between the leading widget and the title.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile.leadingToTitle
func (w *CupertinoListTile) LeadingToTitle() float64 { return w.toTitle }

// CreateState creates the state.
//
// Flutter: cupertino/list_tile.dart CupertinoListTile.createState
func (w *CupertinoListTile) CreateState() widgets.State { return &cupertinoListTileState{} }

type cupertinoListTileState struct {
	widgets.StateOf[*CupertinoListTile]
	tapped bool
}

func (s *cupertinoListTileState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	textStyle := CupertinoThemeOf(bc).TextTheme().TextStyle()
	coloredStyle := textStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(CupertinoDynamicColorResolve(CupertinoColors.SecondaryLabel, bc))})
	base := w.tileType == listTileBase
	titleStyle := textStyle
	if !base && w.o.Subtitle != nil {
		copy := painting.TextStyleCopy{FontWeight: ui.FontWeightW600}
		if w.o.Leading == nil {
			copy.FontSize = foundation.Some(listTileNotchedTitleWithSubtitleFontSize)
		}
		titleStyle = textStyle.CopyWith(copy)
	}
	title := widgets.NewDefaultTextStyle(titleStyle, w.title, widgets.DefaultTextStyleOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis})

	padding := w.o.Padding
	if padding == nil {
		switch {
		case base && w.o.Subtitle != nil:
			padding = listTilePadding
		case !base && w.o.Leading != nil:
			padding = listTileNotchedPadding
		case base:
			padding = listTilePadding
		default:
			padding = listTileNotchedPaddingWithoutLead
		}
	}
	var background ui.ColorValue = CupertinoColors.Transparent
	if w.o.BackgroundColor != nil {
		background = w.o.BackgroundColor
	}
	if s.tapped {
		if w.o.BackgroundColorActivated != nil {
			background = w.o.BackgroundColorActivated
		} else {
			background = CupertinoColors.SystemGrey4
		}
	}
	var minHeight float64
	switch {
	case base && w.o.Subtitle != nil:
		minHeight = listTileMinHeightWithSubtitle
	case !base && w.o.Leading != nil:
		minHeight = listTileNotchedMinHeight
	case base:
		minHeight = listTileMinHeight
	default:
		minHeight = listTileNotchedMinHeightWithoutLead
	}

	var row []widgets.Widget
	if w.o.Leading != nil {
		row = append(row, widgets.NewSizedBox(widgets.NewCenter(w.o.Leading), widgets.SizedBoxOpts{Width: foundation.Some(w.leadingSize), Height: foundation.Some(w.leadingSize)}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(w.toTitle)}))
	} else {
		row = append(row, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(w.leadingSize)}))
	}
	column := []widgets.Widget{title}
	if w.o.Subtitle != nil {
		subtitleSize := listTileNotchedSubtitleFontSize
		if base {
			subtitleSize = listTileSubtitleFontSize
		}
		column = append(column, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(listTileNotchedTitleToSubtitle)}),
			widgets.NewDefaultTextStyle(coloredStyle.CopyWith(painting.TextStyleCopy{FontSize: foundation.Some(subtitleSize)}), w.o.Subtitle,
				widgets.DefaultTextStyleOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis}))
	}
	row = append(row, widgets.NewExpanded(widgets.NewColumn(column, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween, CrossAxisAlignment: rendering.CrossAxisAlignmentStart})))
	if w.o.AdditionalInfo != nil {
		row = append(row, widgets.NewDefaultTextStyle(coloredStyle, w.o.AdditionalInfo, widgets.DefaultTextStyleOpts{MaxLines: foundation.Some(1)}))
		if w.o.Trailing != nil {
			row = append(row, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(listTileAdditionalInfoToTrailing)}))
		}
	}
	if w.o.Trailing != nil {
		row = append(row, w.o.Trailing)
	}
	child := widgets.NewConstrainedBox(rendering.BoxConstraints{MinWidth: math.Inf(1), MaxWidth: math.Inf(1), MinHeight: minHeight, MaxHeight: math.Inf(1)},
		widgets.NewColoredBox(CupertinoDynamicColorResolve(background, bc), widgets.NewPadding(padding, widgets.NewRow(row))))
	if w.o.OnTap == nil {
		return child
	}
	return widgets.NewGestureDetector(child, widgets.GestureDetectorOpts{
		OnTapDown:   func(gestures.TapDownDetails) { s.SetState(func() { s.tapped = true }) },
		OnTapCancel: func() { s.SetState(func() { s.tapped = false }) },
		OnTap: func() {
			w.o.OnTap()
			if s.Mounted() {
				s.SetState(func() { s.tapped = false })
			}
		},
		Behavior: rendering.HitTestBehaviorOpaque,
	})
}

// CupertinoListTileChevron is the chevron that trails a tile that leads somewhere.
//
// Flutter: cupertino/list_tile.dart CupertinoListTileChevron
type CupertinoListTileChevron struct{ widgets.StatelessWidget }

// NewCupertinoListTileChevron creates the chevron.
//
// Flutter: cupertino/list_tile.dart CupertinoListTileChevron.new
func NewCupertinoListTileChevron(opts ...widgets.WidgetOpts) *CupertinoListTileChevron {
	return &CupertinoListTileChevron{StatelessWidget: widgets.NewStatelessWidgetBase(firstOf(opts))}
}

// Build builds the icon.
//
// Flutter: cupertino/list_tile.dart CupertinoListTileChevron.build
func (w *CupertinoListTileChevron) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewIcon(CupertinoIcons.RightChevron(), widgets.IconOpts{Size: CupertinoThemeOf(bc).TextTheme().TextStyle().FontSize,
		Color: foundation.Some(CupertinoDynamicColorResolve(CupertinoColors.SystemGrey2, bc))})
}
