package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TextMagnifierJumpBetweenLinesAnimationDuration is the duration that the
// position is animated if a TextMagnifier just switched between lines.
//
// Flutter: material/magnifier.dart TextMagnifier.jumpBetweenLinesAnimationDuration
const TextMagnifierJumpBetweenLinesAnimationDuration = 70 * time.Millisecond

// The two TextMagnifierConfigurations of TextMagnifierAdaptiveMagnifierConfiguration:
// they differ in whether the magnifier shows the editing handles, which only
// iOS does. They are read-only values, so that a text field does not recreate
// its selection overlay when it is rebuilt with the same configuration.
var (
	adaptiveMagnifierConfigurationWithHandles = widgets.NewTextMagnifierConfiguration(widgets.TextMagnifierConfigurationOpts{
		MagnifierBuilder: adaptiveMagnifierBuilder,
	})
	adaptiveMagnifierConfigurationWithoutHandles = widgets.NewTextMagnifierConfiguration(widgets.TextMagnifierConfigurationOpts{
		MagnifierBuilder: adaptiveMagnifierBuilder, HideHandlesInMagnifier: true,
	})
)

// adaptiveMagnifierBuilder builds the TextMagnifier of Android and nothing on
// the other platforms.
//
// Deviation: Flutter builds a CupertinoTextMagnifier on iOS; there is no
// Cupertino library, so iOS builds nothing (SPEC §12.10).
//
// Flutter: material/magnifier.dart TextMagnifier.adaptiveMagnifierConfiguration
func adaptiveMagnifierBuilder(_ widgets.BuildContext, _ *widgets.MagnifierController, magnifierInfo *foundation.ValueNotifier[widgets.MagnifierInfo]) widgets.Widget {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid:
		return NewTextMagnifier(magnifierInfo)
	case foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia, foundation.TargetPlatformLinux,
		foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
		return nil
	}
	return nil
}

// TextMagnifierAdaptiveMagnifierConfiguration is a TextMagnifierConfiguration
// that returns a TextMagnifier on Android and nil on all other platforms, and
// shows the editing handles only on iOS (where it builds nothing, see the
// deviation of the builder). The same value is returned for the same platform.
//
// Deviation: Flutter's is a static field, whose shouldDisplayHandlesInMagnifier
// is set when it is first read; this is a function that reads the platform each
// time, and there is no CupertinoTextMagnifier for iOS (SPEC §12.10).
//
// Flutter: material/magnifier.dart TextMagnifier.adaptiveMagnifierConfiguration
func TextMagnifierAdaptiveMagnifierConfiguration() *widgets.TextMagnifierConfiguration {
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS {
		return adaptiveMagnifierConfigurationWithHandles
	}
	return adaptiveMagnifierConfigurationWithoutHandles
}

// TextMagnifier is a Magnifier positioned by rules dictated by the native
// Android magnifier.
//
// The positioning rules are based on the magnifier info, as follows:
//
//   - The loupe tracks the gesture's x coordinate, clamping to the beginning and
//     end of the currently editing line.
//
//   - The focal point never contains anything out of the bounds of the text
//     field or other widget being magnified (the MagnifierInfo's FieldBounds).
//
//   - The focal point always remains aligned with the y coordinate of the touch.
//
//   - The loupe always remains on the screen.
//
//   - When the line targeted by the touch's y coordinate changes, the position is
//     animated over TextMagnifierJumpBetweenLinesAnimationDuration.
//
// This behavior was based on the Android 12 source code, where possible, and on
// eyeballing a Pixel 6 running Android 12 otherwise.
//
// Flutter: material/magnifier.dart TextMagnifier
type TextMagnifier struct {
	widgets.StatefulWidget
	magnifierInfo *foundation.ValueNotifier[widgets.MagnifierInfo]
}

// TextMagnifierOpts are the optional parameters of NewTextMagnifier.
//
// Flutter: material/magnifier.dart TextMagnifier.new
type TextMagnifierOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// NewTextMagnifier creates a TextMagnifier.
//
// The magnifierInfo must be provided, and must be updated with new values as the
// user's touch changes.
//
// Flutter: material/magnifier.dart TextMagnifier.new
func NewTextMagnifier(magnifierInfo *foundation.ValueNotifier[widgets.MagnifierInfo], opts ...TextMagnifierOpts) *TextMagnifier {
	o := firstOpts(opts)
	return &TextMagnifier{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), magnifierInfo: magnifierInfo}
}

// MagnifierInfo is the current status of the user's touch.
//
// As the value of the info changes, the position of the loupe is adjusted
// automatically, according to the rules described in the TextMagnifier class
// description.
//
// Flutter: material/magnifier.dart TextMagnifier.magnifierInfo
func (w *TextMagnifier) MagnifierInfo() *foundation.ValueNotifier[widgets.MagnifierInfo] {
	return w.magnifierInfo
}

// CreateState creates the state.
//
// Flutter: material/magnifier.dart TextMagnifier.createState
func (w *TextMagnifier) CreateState() widgets.State { return &textMagnifierState{} }

type textMagnifierState struct {
	widgets.StateOf[*TextMagnifier]
	// magnifierPosition should only be none on construction. This is because of
	// the animation logic.
	//
	// Animations are added when last_build_y != current_build_y. This condition
	// is true on the initial render, which would mean that the initial build
	// would be animated - this is undesired. Thus, this is none for the first
	// frame and the condition becomes magnifierPosition != none && last_build_y
	// != this_build_y.
	magnifierPosition foundation.Opt[ui.Offset]

	// positionShouldBeAnimatedTimer is a timer that unsets itself after an
	// animation duration. If the timer exists, then the magnifier animates its
	// position - if this timer does not exist, the magnifier tracks the gesture
	// (with respect to the positioning rules) directly.
	positionShouldBeAnimatedTimer *gestures.Timer

	extraFocalPointOffset ui.Offset

	infoHandle foundation.ListenerHandle
}

func (s *textMagnifierState) positionShouldBeAnimated() bool {
	return s.positionShouldBeAnimatedTimer != nil
}

func (s *textMagnifierState) InitState() {
	s.infoHandle = s.Widget().magnifierInfo.AddListener(s.determineMagnifierPositionAndFocalPoint)
}

func (s *textMagnifierState) Dispose() {
	s.Widget().magnifierInfo.RemoveListener(s.infoHandle)
	if s.positionShouldBeAnimatedTimer != nil {
		s.positionShouldBeAnimatedTimer.Cancel()
	}
}

func (s *textMagnifierState) DidChangeDependencies() {
	s.determineMagnifierPositionAndFocalPoint()
}

func (s *textMagnifierState) DidUpdateWidget(oldWidget *TextMagnifier) {
	if oldWidget.magnifierInfo != s.Widget().magnifierInfo {
		oldWidget.magnifierInfo.RemoveListener(s.infoHandle)
		s.infoHandle = s.Widget().magnifierInfo.AddListener(s.determineMagnifierPositionAndFocalPoint)
	}
}

func (s *textMagnifierState) determineMagnifierPositionAndFocalPoint() {
	selectionInfo := s.Widget().magnifierInfo.Value()
	screenRect := ui.OffsetZero.And(widgets.MediaQuerySizeOf(s.Context()))

	// Since by default we draw at the top left corner, this offset shifts the
	// magnifier so we draw at the center, and then also includes the "above
	// touch point" shift.
	basicMagnifierOffset := ui.Offset{
		Dx: MagnifierDefaultMagnifierSize.Width / 2,
		Dy: MagnifierDefaultMagnifierSize.Height + MagnifierStandardVerticalFocalPointShift,
	}

	// Since the magnifier should not go past the edges of the line, but must
	// track the gesture otherwise, constrain the X of the magnifier to always
	// stay between line start and end.
	magnifierX := ui.ClampDouble(
		selectionInfo.GlobalGesturePosition.Dx,
		selectionInfo.CurrentLineBoundaries.Left,
		selectionInfo.CurrentLineBoundaries.Right,
	)

	// Place the magnifier at the previously calculated X, and the Y should be
	// exactly at the center of the handle.
	unadjustedMagnifierRect := ui.Offset{Dx: magnifierX, Dy: selectionInfo.CaretRect.Center().Dy}.Sub(basicMagnifierOffset).And(MagnifierDefaultMagnifierSize)

	// Shift the magnifier so that, if we are ever out of the screen, we become in
	// bounds. This probably won't have much of an effect on the X, since it is
	// already bound to the currentLineBoundaries, but will shift vertically if
	// the magnifier is out of bounds.
	screenBoundsAdjustedMagnifierRect := widgets.MagnifierControllerShiftWithinBounds(unadjustedMagnifierRect, screenRect)

	// Done with the magnifier position!
	finalMagnifierPosition := screenBoundsAdjustedMagnifierRect.TopLeft()

	// The insets, from either edge, that the focal point should not point past
	// lest the magnifier displays something out of bounds.
	horizontalMaxFocalPointEdgeInsets := (MagnifierDefaultMagnifierSize.Width / 2) / magnifierMagnification

	// Adjust the focal point horizontally such that none of the magnifier ever
	// points to anything out of bounds.
	var newGlobalFocalPointX float64

	// If the text field is so narrow that we must show out of bounds, then settle
	// for pointing to the center all the time.
	if selectionInfo.FieldBounds.Width() < horizontalMaxFocalPointEdgeInsets*2 {
		newGlobalFocalPointX = selectionInfo.FieldBounds.Center().Dx
	} else {
		// Otherwise, we can clamp the focal point to always point in bounds.
		newGlobalFocalPointX = ui.ClampDouble(
			screenBoundsAdjustedMagnifierRect.Center().Dx,
			selectionInfo.FieldBounds.Left+horizontalMaxFocalPointEdgeInsets,
			selectionInfo.FieldBounds.Right-horizontalMaxFocalPointEdgeInsets,
		)
	}

	// Since the previous value is now a global offset (i.e. newGlobalFocalPoint
	// is now a global offset), we must subtract the magnifier's global offset to
	// obtain the relative shift in the focal point.
	newRelativeFocalPointX := newGlobalFocalPointX - screenBoundsAdjustedMagnifierRect.Center().Dx

	// The Y component means that if we are pressed up against the top of the
	// screen, then we should adjust the focal point such that it now points to
	// how far we moved the magnifier. screenBoundsAdjustedMagnifierRect.top ==
	// unadjustedMagnifierRect.top for most cases, but when pressed up against the
	// top of the screen, we adjust the focal point by the amount that we shifted
	// from our "natural" position.
	focalPointAdjustmentForScreenBoundsAdjustment := ui.Offset{
		Dx: newRelativeFocalPointX,
		Dy: unadjustedMagnifierRect.Top - screenBoundsAdjustedMagnifierRect.Top,
	}

	positionShouldBeAnimated := s.positionShouldBeAnimatedTimer

	if position, ok := s.magnifierPosition.Get(); ok && finalMagnifierPosition.Dy != position.Dy {
		if s.positionShouldBeAnimatedTimer != nil && s.positionShouldBeAnimatedTimer.IsActive() {
			s.positionShouldBeAnimatedTimer.Cancel()
		}

		// Create a timer that deletes itself when the timer is complete. This is
		// mounted safe, since the timer is canceled in Dispose.
		positionShouldBeAnimated = widgets.WidgetsBindingOf(s.Context()).Gestures().NewTimer(
			TextMagnifierJumpBetweenLinesAnimationDuration,
			func() {
				s.SetState(func() { s.positionShouldBeAnimatedTimer = nil })
			})
	}

	s.SetState(func() {
		s.magnifierPosition = foundation.Some(finalMagnifierPosition)
		s.positionShouldBeAnimatedTimer = positionShouldBeAnimated
		s.extraFocalPointOffset = focalPointAdjustmentForScreenBoundsAdjustment
	})
}

func (s *textMagnifierState) Build(widgets.BuildContext) widgets.Widget {
	position, ok := s.magnifierPosition.Get()
	if foundation.DebugMode && !ok {
		panic(errors.New(tagTextMagnifierPositionNotSet, "the magnifier position should only be unset before the first build"))
	}

	duration := time.Duration(0)
	if s.positionShouldBeAnimated() {
		// Material magnifier typically does not animate, unless we jump between
		// lines, in which case we animate between lines.
		duration = TextMagnifierJumpBetweenLinesAnimationDuration
	}
	return widgets.NewAnimatedPositioned(duration,
		NewMagnifier(MagnifierOpts{AdditionalFocalPointOffset: s.extraFocalPointOffset}),
		widgets.AnimatedPositionedOpts{Top: foundation.Some(position.Dy), Left: foundation.Some(position.Dx)})
}

const (
	// magnifierBorderRadius is the radius of the corners of the magnifier.
	//
	// Flutter: material/magnifier.dart Magnifier._borderRadius
	magnifierBorderRadius = 40.0
	// magnifierMagnification is how much the magnifier zooms in.
	//
	// Flutter: material/magnifier.dart Magnifier._magnification
	magnifierMagnification = 1.25
)

// MagnifierDefaultMagnifierSize is the default size of a Magnifier.
//
// The size of the magnifier may be modified through the constructor; it is
// extracted from the default parameter of Magnifier's constructor so that
// positioners may depend on it. It is a read-only value.
//
// Flutter: material/magnifier.dart Magnifier.kDefaultMagnifierSize
var MagnifierDefaultMagnifierSize = ui.Size{Width: 77.37, Height: 37.9}

// MagnifierStandardVerticalFocalPointShift is the vertical distance that the
// magnifier should be above the focal point.
//
// The value is a constant so that positioning of a Magnifier can be done with a
// guaranteed size, as opposed to an estimate.
//
// Flutter: material/magnifier.dart Magnifier.kStandardVerticalFocalPointShift
const MagnifierStandardVerticalFocalPointShift = 22.0

// MagnifierOpts are the optional parameters of NewMagnifier.
//
// Flutter: material/magnifier.dart Magnifier.new
type MagnifierOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AdditionalFocalPointOffset is any additional offset the focal point
	// requires to "point" to the correct place.
	//
	// This value is added to MagnifierStandardVerticalFocalPointShift to obtain
	// the actual offset.
	//
	// This is useful for instances where the magnifier is not pointing to
	// something directly below it.
	AdditionalFocalPointOffset ui.Offset
	// BorderRadius is the border radius for this magnifier; the default is a
	// circular radius of 40.
	//
	// The magnifier's shape is a RoundedRectangleBorder with this radius.
	BorderRadius painting.BorderRadiusGeometry
	// FilmColor is the color to tint the image in this Magnifier; the default is
	// an almost transparent gray.
	//
	// On native Android, there is a almost transparent gray tint to the
	// magnifier, in order to better distinguish the contents of the lens from the
	// background.
	FilmColor foundation.Opt[ui.Color]
	// Shadows is a list of shadows cast by the Magnifier; nil is the default
	// shadow, and an empty slice is no shadow.
	//
	// If the shadows use a blur style that paints inside the shape, or if they
	// are offset, then a clip behavior that enables clipping (such as the default
	// ClipHardEdge) is recommended, otherwise the shadow will occlude the
	// magnifier (the shadow is drawn above the magnifier so as to not be included
	// in the magnified image).
	//
	// By default, the shadows are offset vertically by two logical pixels, so
	// clipping is recommended.
	//
	// A shadow that uses the outer blur style and is not offset does not need
	// clipping; in that case, consider setting the clip behavior to ClipNone.
	Shadows []painting.BoxShadow
	// ClipBehavior is whether and how to clip the shadows that render inside the
	// loupe; the default is ClipHardEdge.
	//
	// A value of ClipNone can be used if the shadow will not paint where the
	// magnified image appears, or if doing so is intentional (e.g. to blur the
	// edges of the magnified image).
	ClipBehavior ui.Clip
	// Size is the size of the Magnifier; the default is
	// MagnifierDefaultMagnifierSize. The shadows are drawn outside of the size.
	Size foundation.Opt[ui.Size]
}

// Magnifier is a Material-styled magnifying glass.
//
// This magnifying glass is useful for scenarios on mobile devices where the
// user's finger may be covering part of the screen where a granular action is
// being performed, such as navigating a small cursor with a drag gesture, on an
// image or text.
//
// This widget focuses on mimicking the style of the magnifier on material. For a
// widget that is focused on mimicking the behavior of a material magnifier, see
// TextMagnifier, which uses Magnifier.
//
// The styles implemented in this widget were based on the Android 12 source
// code, where possible, and on eyeballing a Pixel 6 running Android 12
// otherwise.
//
// Flutter: material/magnifier.dart Magnifier
type Magnifier struct {
	widgets.StatelessWidget
	additionalFocalPointOffset ui.Offset
	borderRadius               painting.BorderRadiusGeometry
	filmColor                  ui.Color
	shadows                    []painting.BoxShadow
	clipBehavior               ui.Clip
	size                       ui.Size
}

// NewMagnifier creates a RawMagnifier in the Material style.
//
// Flutter: material/magnifier.dart Magnifier.new
func NewMagnifier(opts ...MagnifierOpts) *Magnifier {
	o := firstOpts(opts)
	m := &Magnifier{
		StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), additionalFocalPointOffset: o.AdditionalFocalPointOffset,
		borderRadius: o.BorderRadius, filmColor: o.FilmColor.OrElse(ui.NewColorFromARGB(8, 158, 158, 158)), shadows: o.Shadows,
		clipBehavior: o.ClipBehavior, size: o.Size.OrElse(MagnifierDefaultMagnifierSize),
	}
	if m.borderRadius == nil {
		m.borderRadius = painting.NewBorderRadiusCircular(magnifierBorderRadius)
	}
	if m.shadows == nil {
		m.shadows = []painting.BoxShadow{painting.NewBoxShadow(painting.BoxShadowOpts{
			BlurRadius: 1.5, Offset: ui.Offset{Dy: 2}, SpreadRadius: 0.75, Color: foundation.Some(ui.NewColorFromARGB(25, 0, 0, 0)),
		})}
	}
	if m.clipBehavior == ui.ClipUnspecified {
		m.clipBehavior = ui.ClipHardEdge
	}
	return m
}

// AdditionalFocalPointOffset is any additional offset the focal point requires
// to "point" to the correct place.
//
// Flutter: material/magnifier.dart Magnifier.additionalFocalPointOffset
func (m *Magnifier) AdditionalFocalPointOffset() ui.Offset { return m.additionalFocalPointOffset }

// BorderRadius is the border radius for this magnifier.
//
// Flutter: material/magnifier.dart Magnifier.borderRadius
func (m *Magnifier) BorderRadius() painting.BorderRadiusGeometry { return m.borderRadius }

// FilmColor is the color to tint the image in this Magnifier.
//
// Flutter: material/magnifier.dart Magnifier.filmColor
func (m *Magnifier) FilmColor() ui.Color { return m.filmColor }

// Shadows is a list of shadows cast by the Magnifier.
//
// Flutter: material/magnifier.dart Magnifier.shadows
func (m *Magnifier) Shadows() []painting.BoxShadow { return m.shadows }

// ClipBehavior is whether and how to clip the shadows that render inside the
// loupe.
//
// Flutter: material/magnifier.dart Magnifier.clipBehavior
func (m *Magnifier) ClipBehavior() ui.Clip { return m.clipBehavior }

// Size is the size of this Magnifier.
//
// Flutter: material/magnifier.dart Magnifier.size
func (m *Magnifier) Size() ui.Size { return m.size }

// Build builds a RawMagnifier in the Material style.
//
// Flutter: material/magnifier.dart Magnifier.build
func (m *Magnifier) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewRawMagnifier(m.size, widgets.RawMagnifierOpts{
		Decoration: widgets.NewMagnifierDecoration(widgets.MagnifierDecorationOpts{
			Shape:   painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: m.borderRadius}),
			Shadows: m.shadows,
		}),
		ClipBehavior:       m.clipBehavior,
		MagnificationScale: foundation.Some(magnifierMagnification),
		FocalPointOffset: m.additionalFocalPointOffset.Add(
			ui.Offset{Dy: MagnifierStandardVerticalFocalPointShift + MagnifierDefaultMagnifierSize.Height/2}),
		Child: widgets.NewColoredBox(
			// This couldn't be part of the decoration (even if the
			// MagnifierDecoration supported specifying a color) because the
			// decoration's shadows are offset and therefore we set a clipBehavior
			// that clips the inner part of the decoration to avoid occluding the
			// magnified image with the shadow.
			m.filmColor, nil),
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTextMagnifierPositionNotSet = "TextMagnifierPositionNotSet"
)
