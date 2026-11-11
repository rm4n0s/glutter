package cupertino

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	expansionTileAnimationDuration = 250 * time.Millisecond
	expansionTileIconFontSize      = 15.0
	expansionTileHeaderHeight      = 44.0
)

// ExpansionTileTransitionMode is how the body of a CupertinoExpansionTile appears while it expands.
//
// Flutter: cupertino/expansion_tile.dart ExpansionTileTransitionMode
type ExpansionTileTransitionMode int

const (
	// ExpansionTileTransitionModeUnspecified is the zero value: the default, fade.
	//
	// Flutter: no counterpart
	ExpansionTileTransitionModeUnspecified ExpansionTileTransitionMode = iota
	// ExpansionTileTransitionModeFade fades the body in over the content that moves down.
	//
	// Flutter: cupertino/expansion_tile.dart ExpansionTileTransitionMode.fade
	ExpansionTileTransitionModeFade
	// ExpansionTileTransitionModeScroll grows the body with its size, like a scroll.
	//
	// Flutter: cupertino/expansion_tile.dart ExpansionTileTransitionMode.scroll
	ExpansionTileTransitionModeScroll
)

// CupertinoExpansionTileOpts are the optional parameters of NewCupertinoExpansionTile.
//
// Flutter: cupertino/expansion_tile.dart CupertinoExpansionTile.new
type CupertinoExpansionTileOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller expands and collapses the tile; the default is one of the tile's own.
	Controller *widgets.ExpansibleController
	// TransitionMode is how the body appears; the default is fade.
	TransitionMode ExpansionTileTransitionMode
}

// CupertinoExpansionTile is a tile with a title that shows a body when it is tapped, with a chevron that turns as it
// expands.
//
// Flutter: cupertino/expansion_tile.dart CupertinoExpansionTile
type CupertinoExpansionTile struct {
	widgets.StatefulWidget
	title, child widgets.Widget
	o            CupertinoExpansionTileOpts
}

// NewCupertinoExpansionTile creates the tile.
//
// Flutter: cupertino/expansion_tile.dart CupertinoExpansionTile.new
func NewCupertinoExpansionTile(title, child widgets.Widget, opts ...CupertinoExpansionTileOpts) *CupertinoExpansionTile {
	o := firstOf(opts)
	if o.TransitionMode == ExpansionTileTransitionModeUnspecified {
		o.TransitionMode = ExpansionTileTransitionModeFade
	}
	return &CupertinoExpansionTile{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), title: title, child: child, o: o}
}

// Title is the title in the header.
//
// Flutter: cupertino/expansion_tile.dart CupertinoExpansionTile.title
func (w *CupertinoExpansionTile) Title() widgets.Widget { return w.title }

// Child is the body.
//
// Flutter: cupertino/expansion_tile.dart CupertinoExpansionTile.child
func (w *CupertinoExpansionTile) Child() widgets.Widget { return w.child }

// TransitionMode is how the body appears.
//
// Flutter: cupertino/expansion_tile.dart CupertinoExpansionTile.transitionMode
func (w *CupertinoExpansionTile) TransitionMode() ExpansionTileTransitionMode {
	return w.o.TransitionMode
}

// CreateState creates the state.
//
// Flutter: cupertino/expansion_tile.dart CupertinoExpansionTile.createState
func (w *CupertinoExpansionTile) CreateState() widgets.State { return &cupertinoExpansionTileState{} }

type cupertinoExpansionTileState struct {
	widgets.StateOf[*CupertinoExpansionTile]
	headerKey      *widgets.GlobalKey[widgets.State]
	fadeController *widgets.OverlayPortalController
	tileController *widgets.ExpansibleController
	quarterTween   animation.Animatable[float64]
	curveTween     animation.Animatable[float64]
}

func (s *cupertinoExpansionTileState) InitState() {
	s.headerKey = widgets.NewGlobalKey[widgets.State]()
	s.fadeController = widgets.NewOverlayPortalController()
	s.quarterTween = animation.NewTween(0.0, 0.25)
	s.curveTween = animation.NewCurveTween(animation.Curves.EaseInOut)
	s.adoptController()
}

func (s *cupertinoExpansionTileState) adoptController() {
	s.tileController = s.Widget().o.Controller
	if s.tileController == nil {
		s.tileController = widgets.NewExpansibleController()
	}
}

func (s *cupertinoExpansionTileState) DidUpdateWidget(old *CupertinoExpansionTile) {
	if old.o.Controller != s.Widget().o.Controller {
		if old.o.Controller == nil {
			s.tileController.Dispose()
		}
		s.adoptController()
	}
}

func (s *cupertinoExpansionTileState) Dispose() {
	if s.Widget().o.Controller == nil {
		s.tileController.Dispose()
	}
}

func (s *cupertinoExpansionTileState) buildIcon(bc widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
	turns := animation.Drive[float64](animation.Drive[float64](a, s.curveTween), s.quarterTween)
	fontSize := CupertinoThemeOf(bc).TextTheme().TextStyle().FontSize
	return widgets.NewRotationTransition(turns, widgets.NewSizedBoxSquare(fontSize, widgets.NewCenter(
		widgets.NewIcon(CupertinoIcons.RightChevron(), widgets.IconOpts{
			Color:      foundation.Some(CupertinoDynamicColorResolve(CupertinoColors.ActiveBlue, bc)),
			Size:       foundation.Some(expansionTileIconFontSize),
			FontWeight: ui.FontWeightW900,
		}))))
}

func (s *cupertinoExpansionTileState) onHeaderTap() {
	if s.tileController.IsExpanded() {
		s.tileController.Collapse()
	} else {
		s.tileController.Expand()
	}
	s.fadeController.Show()
}

func (s *cupertinoExpansionTileState) buildHeader(bc widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
	l := CupertinoLocalizationsOf(bc)
	expanded := s.tileController.IsExpanded()
	onTapHint := l.ExpansionTileCollapsedTapHint()
	if expanded {
		onTapHint = l.ExpansionTileExpandedTapHint()
	}
	var hint foundation.Opt[string]
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		if expanded {
			hint = foundation.Some(l.CollapsedHint() + "\n " + l.ExpansionTileExpandedHint())
		} else {
			hint = foundation.Some(l.ExpandedHint() + "\n " + l.ExpansionTileCollapsedHint())
		}
	}
	return widgets.NewSemantics(NewCupertinoListTile(s.Widget().title, CupertinoListTileOpts{
		Key: s.headerKey, OnTap: s.onHeaderTap, Trailing: s.buildIcon(bc, a), BackgroundColorActivated: CupertinoColors.Transparent,
	}), widgets.SemanticsOpts{Hint: hint, HintOverrides: &semantics.SemanticsHintOverrides{OnTapHint: onTapHint}})
}

func (s *cupertinoExpansionTileState) buildExpansible(bc widgets.BuildContext, header, body widgets.Widget, a animation.Animation[float64]) widgets.Widget {
	w := s.Widget()
	second := body
	if animation.IsAnimating(a) && w.o.TransitionMode == ExpansionTileTransitionModeFade {
		second = widgets.NewOpacity(0.0, body)
	}
	child := widgets.NewColumn([]widgets.Widget{header, second}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})
	if w.o.TransitionMode == ExpansionTileTransitionModeScroll {
		return child
	}
	return widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
		return widgets.NewOverlayPortal(s.fadeController, func(bc widgets.BuildContext) widgets.Widget {
			headerContext := s.headerKey.CurrentContext()
			overlay := widgets.OverlayOf(headerContext).Context().FindRenderObject().(rendering.Box)
			headerBox := headerContext.FindRenderObject().(rendering.Box)
			headerOffset := headerBox.LocalToGlobal(ui.OffsetZero, overlay)
			return widgets.NewPositioned(widgets.NewConstrainedBox(constraints, widgets.NewVisibility(
				widgets.NewFadeTransition(a, w.child), widgets.VisibilityOpts{Visible: foundation.Some(animation.IsAnimating(a))})),
				widgets.PositionedOpts{Top: foundation.Some(headerOffset.Dy + expansionTileHeaderHeight), Left: foundation.Some(headerOffset.Dx)})
		}, child)
	})
}

func (s *cupertinoExpansionTileState) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewExpansible(s.tileController, s.buildHeader,
		func(bc widgets.BuildContext, a animation.Animation[float64]) widgets.Widget { return s.Widget().child },
		widgets.ExpansibleOpts{Duration: foundation.Some(expansionTileAnimationDuration), Curve: animation.Curves.EaseInOut,
			ExpansibleBuilder: s.buildExpansible})
}
