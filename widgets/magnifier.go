package widgets

import (
	"reflect"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// MagnifierBuilder builds a widget with a MagnifierController. It returns nil
// (Dart's null) when it builds no magnifier.
//
// The builder is called exactly once per magnifier.
//
// If the AnimationController of the controller is set (by the builder) to an
// AnimationController, the MagnifierController drives the animation during
// entry and exit.
//
// The magnifierInfo is updated with new MagnifierInfo values during the
// lifetime of the built magnifier, e.g. as the user moves their finger around
// the text field.
//
// Flutter: widgets/magnifier.dart MagnifierBuilder
type MagnifierBuilder func(bc BuildContext, controller *MagnifierController, magnifierInfo *foundation.ValueNotifier[MagnifierInfo]) Widget

// MagnifierInfo is a data class that contains the geometry information of text
// layouts and selection gestures, used to position magnifiers. Its fields are
// all the parameters of Flutter's constructor (use NewMagnifierInfo or a
// literal).
//
// Flutter: widgets/magnifier.dart MagnifierInfo
type MagnifierInfo struct {
	// GlobalGesturePosition is the offset of the gesture position that the
	// magnifier should be shown at.
	//
	// Flutter: widgets/magnifier.dart MagnifierInfo.globalGesturePosition
	GlobalGesturePosition ui.Offset
	// CurrentLineBoundaries is the rect of the current line the magnifier
	// should be shown at, without taking into account any padding of the field;
	// only the position of the first and last character.
	//
	// Flutter: widgets/magnifier.dart MagnifierInfo.currentLineBoundaries
	CurrentLineBoundaries ui.Rect
	// CaretRect is the rect of the handle that the magnifier should follow.
	//
	// Flutter: widgets/magnifier.dart MagnifierInfo.caretRect
	CaretRect ui.Rect
	// FieldBounds are the bounds of the entire text field that the magnifier is
	// bound to.
	//
	// Flutter: widgets/magnifier.dart MagnifierInfo.fieldBounds
	FieldBounds ui.Rect
}

// NewMagnifierInfo constructs a MagnifierInfo from provided geometry values.
//
// Flutter: widgets/magnifier.dart MagnifierInfo.new
func NewMagnifierInfo(globalGesturePosition ui.Offset, caretRect, fieldBounds, currentLineBoundaries ui.Rect) MagnifierInfo {
	return MagnifierInfo{GlobalGesturePosition: globalGesturePosition, CaretRect: caretRect, FieldBounds: fieldBounds, CurrentLineBoundaries: currentLineBoundaries}
}

// MagnifierInfoEmpty is the MagnifierInfo with all values set to 0. It is a
// read-only value.
//
// Flutter: widgets/magnifier.dart MagnifierInfo.empty
var MagnifierInfoEmpty = MagnifierInfo{}

// String describes the info, such as "MagnifierInfo(position: Offset(0.0, 0.0),
// line: ..., caret: ..., field: ...)".
//
// Flutter: widgets/magnifier.dart MagnifierInfo.toString
func (m MagnifierInfo) String() string {
	return "MagnifierInfo(position: " + m.GlobalGesturePosition.String() + ", line: " + m.CurrentLineBoundaries.String() +
		", caret: " + m.CaretRect.String() + ", field: " + m.FieldBounds.String() + ")"
}

// noMagnifier is the builder of a TextMagnifierConfiguration without a
// magnifier builder: it builds nothing.
//
// Flutter: widgets/magnifier.dart TextMagnifierConfiguration._none
func noMagnifier(BuildContext, *MagnifierController, *foundation.ValueNotifier[MagnifierInfo]) Widget {
	return nil
}

// TextMagnifierConfigurationOpts are the optional parameters of
// NewTextMagnifierConfiguration.
//
// Flutter: widgets/magnifier.dart TextMagnifierConfiguration.new
type TextMagnifierConfigurationOpts struct {
	// MagnifierBuilder is the builder callback that creates the widget that
	// renders the magnifier. When it is nil, a default builder that does not
	// build a magnifier is used.
	MagnifierBuilder MagnifierBuilder
	// HideHandlesInMagnifier stops a magnifier from showing the text editing
	// handles (Flutter's shouldDisplayHandlesInMagnifier is true by default).
	HideHandlesInMagnifier bool
}

// TextMagnifierConfiguration is a configuration object for a magnifier (e.g. in
// a text field).
//
// In general, most features of the magnifier can be configured by controlling
// the widgets built by the magnifier builder.
//
// A configuration is compared by identity (Dart has no == for it): the same
// *TextMagnifierConfiguration is the same configuration.
//
// Flutter: widgets/magnifier.dart TextMagnifierConfiguration
type TextMagnifierConfiguration struct {
	magnifierBuilder                MagnifierBuilder
	shouldDisplayHandlesInMagnifier bool
}

// NewTextMagnifierConfiguration constructs a TextMagnifierConfiguration from
// parts.
//
// If the magnifier builder is nil, a default MagnifierBuilder will be used that
// does not build a magnifier.
//
// Flutter: widgets/magnifier.dart TextMagnifierConfiguration.new
func NewTextMagnifierConfiguration(opts ...TextMagnifierConfigurationOpts) *TextMagnifierConfiguration {
	o := firstOpts(opts)
	return &TextMagnifierConfiguration{magnifierBuilder: o.MagnifierBuilder, shouldDisplayHandlesInMagnifier: !o.HideHandlesInMagnifier}
}

// TextMagnifierConfigurationDisabled is a TextMagnifierConfiguration that is
// disabled, meaning it never builds anything, regardless of platform. It is a
// read-only value.
//
// Flutter: widgets/magnifier.dart TextMagnifierConfiguration.disabled
var TextMagnifierConfigurationDisabled = NewTextMagnifierConfiguration()

// MagnifierBuilder is the builder callback that creates the widget that renders
// the magnifier: the default one builds nothing.
//
// Flutter: widgets/magnifier.dart TextMagnifierConfiguration.magnifierBuilder
func (c *TextMagnifierConfiguration) MagnifierBuilder() MagnifierBuilder {
	if c.magnifierBuilder == nil {
		return noMagnifier
	}
	return c.magnifierBuilder
}

// ShouldDisplayHandlesInMagnifier is whether a magnifier should show the text
// editing handles or not.
//
// This flag is used by SelectionOverlay.ShowMagnifier to control the order of
// layers in the rendering; specifically, whether to place the layer containing
// the handles above or below the layer containing the magnifier in the Overlay.
//
// Flutter: widgets/magnifier.dart TextMagnifierConfiguration.shouldDisplayHandlesInMagnifier
func (c *TextMagnifierConfiguration) ShouldDisplayHandlesInMagnifier() bool {
	return c.shouldDisplayHandlesInMagnifier
}

// MagnifierControllerOpts are the optional parameters of NewMagnifierController.
//
// Flutter: widgets/magnifier.dart MagnifierController.new
type MagnifierControllerOpts struct {
	// AnimationController is the controller that will be driven in / out when
	// show / hide is triggered, respectively. If there is no in / out animation
	// for the magnifier, leave it nil.
	AnimationController *animation.AnimationController
}

// MagnifierController is a controller for a magnifier.
//
// MagnifierController's main benefit over holding a raw OverlayEntry is that
// MagnifierController will handle logic around waiting for a magnifier to
// animate in or out.
//
// If a magnifier chooses to have an entry / exit animation, it should provide
// the animation controller to SetAnimationController. MagnifierController will
// then drive the AnimationController and wait for it to be complete before
// removing it from the Overlay.
//
// To check the status of the magnifier, see Shown.
//
// Flutter: widgets/magnifier.dart MagnifierController
type MagnifierController struct {
	animationController *animation.AnimationController
	overlayEntry        *OverlayEntry
}

// NewMagnifierController creates a controller. If there is no in / out
// animation for the magnifier, the animation controller should be left nil.
//
// Flutter: widgets/magnifier.dart MagnifierController.new
func NewMagnifierController(opts ...MagnifierControllerOpts) *MagnifierController {
	c := &MagnifierController{animationController: firstOpts(opts).AnimationController}
	if c.animationController != nil {
		c.animationController.SetValue(0)
	}
	return c
}

// AnimationController is the controller that will be driven in / out when show
// / hide is triggered, respectively.
//
// Flutter: widgets/magnifier.dart MagnifierController.animationController
func (c *MagnifierController) AnimationController() *animation.AnimationController {
	return c.animationController
}

// SetAnimationController sets the controller that will be driven in / out when
// show / hide is triggered, respectively.
//
// Flutter: widgets/magnifier.dart MagnifierController.animationController
func (c *MagnifierController) SetAnimationController(controller *animation.AnimationController) {
	c.animationController = controller
}

// OverlayEntry is the magnifier's OverlayEntry, if currently in the overlay,
// else nil.
//
// This is exposed so that other overlay entries can be positioned above or
// below this overlay entry. Anything in the paint order after the RawMagnifier
// in this OverlayEntry will not be displayed in the magnifier; if it is desired
// for an overlay entry to be displayed in the magnifier, it must be positioned
// below the magnifier.
//
// To check if a magnifier is in the overlay, use Shown. The overlay entry may
// be non-nil even when the magnifier is not visible.
//
// Flutter: widgets/magnifier.dart MagnifierController.overlayEntry
func (c *MagnifierController) OverlayEntry() *OverlayEntry { return c.overlayEntry }

// Shown is whether the magnifier is currently being shown.
//
// This is false when nothing is in the overlay, when the animation controller
// is in the dismissed state, or when the animation controller is animating out
// (i.e. in the reverse state).
//
// It is true in the opposite cases, i.e. when the overlay is not empty, and
// either the animation controller is nil, in the completed state, or in the
// forward state.
//
// Flutter: widgets/magnifier.dart MagnifierController.shown
func (c *MagnifierController) Shown() bool {
	return c.overlayEntry != nil && (c.animationController == nil || c.animationController.Status().IsForwardOrCompleted())
}

// MagnifierControllerShowOpts are the optional parameters of
// MagnifierController.Show.
//
// Flutter: widgets/magnifier.dart MagnifierController.show
type MagnifierControllerShowOpts struct {
	// DebugRequiredFor is the widget that requires the overlay, named in the
	// error when there is none.
	DebugRequiredFor Widget
	// Below is the entry the magnifier is inserted below, to control what
	// overlays are shown in the magnifier. See OverlayEntry.
	Below *OverlayEntry
}

// Show displays the magnifier.
//
// It returns a future that completes when the magnifier is fully shown, i.e.
// done with its entry animation (a completed one when there is no animation).
//
// To control what overlays are shown in the magnifier, use Below. See
// OverlayEntry for more details on how to utilize Below.
//
// If the magnifier already exists (i.e. OverlayEntry is not nil), then Show
// will replace the old overlay without playing an exit animation. Consider
// hiding it first, to animate from the old magnifier to the new one.
//
// Flutter: widgets/magnifier.dart MagnifierController.show
func (c *MagnifierController) Show(bc BuildContext, builder WidgetBuilder, opts ...MagnifierControllerShowOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	if c.overlayEntry != nil {
		c.overlayEntry.Remove()
		c.overlayEntry.Dispose()
	}

	overlayState := OverlayOf(bc, OverlayOfOpts{RootOverlay: true, DebugRequiredFor: o.DebugRequiredFor})

	var navigatorContext BuildContext
	if navigator := NavigatorMaybeOf(bc); navigator != nil {
		navigatorContext = navigator.Context()
	}
	capturedThemes := CaptureThemes(bc, navigatorContext)

	c.overlayEntry = NewOverlayEntry(func(bc BuildContext) Widget { return capturedThemes.Wrap(builder(bc)) })
	overlayState.Insert(c.overlayEntry, OverlayInsertOpts{Below: o.Below})

	if c.animationController != nil {
		return c.animationController.Forward()
	}
	return scheduler.NewTickerFutureComplete()
}

// MagnifierControllerHideOpts are the optional parameters of
// MagnifierController.Hide.
//
// Flutter: widgets/magnifier.dart MagnifierController.hide
type MagnifierControllerHideOpts struct {
	// KeepInOverlay keeps the magnifier in the overlay once it is hidden, for a
	// magnifier that needs to preserve states between shows / hides (Flutter's
	// removeFromOverlay is true by default).
	KeepInOverlay bool
}

// Hide schedules a hide of the magnifier.
//
// If this MagnifierController has an AnimationController, then Hide reverses
// the animation controller and waits for the animation to complete. Then,
// unless KeepInOverlay is set, it removes the magnifier from the overlay. The
// returned future completes when that is done (it never completes if the
// reverse animation is canceled, as Dart's TickerFuture does not).
//
// In general, the magnifier should be removed from the overlay, unless the
// magnifier needs to preserve states between shows / hides.
//
// Flutter: widgets/magnifier.dart MagnifierController.hide
func (c *MagnifierController) Hide(opts ...MagnifierControllerHideOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	future, complete := scheduler.NewTickerFuturePending()
	if c.overlayEntry == nil {
		complete()
		return future
	}
	finish := func() {
		if !o.KeepInOverlay {
			c.RemoveFromOverlay()
		}
		complete()
	}
	if c.animationController != nil {
		c.animationController.Reverse().Then(finish)
	} else {
		finish()
	}
	return future
}

// RemoveFromOverlay removes the OverlayEntry from the Overlay.
//
// This method removes the OverlayEntry synchronously, regardless of exit
// animation: this leads to abrupt removals of OverlayEntries with animations.
//
// To allow the OverlayEntry to play its exit animation, consider calling Hide
// instead.
//
// Flutter: widgets/magnifier.dart MagnifierController.removeFromOverlay
func (c *MagnifierController) RemoveFromOverlay() {
	if c.overlayEntry != nil {
		c.overlayEntry.Remove()
		c.overlayEntry.Dispose()
	}
	c.overlayEntry = nil
}

// MagnifierControllerShiftWithinBounds is a utility for calculating a new Rect
// from rect such that rect is fully constrained within bounds.
//
// Any point in the output rect is guaranteed to also be a point contained in
// bounds.
//
// It is a runtime error for rect's width to be greater than bounds' width, and
// it is also an error for rect's height to be greater than bounds' height
// (a panic with the tag MagnifierRectLargerThanBounds in debug builds).
//
// This algorithm translates rect the shortest distance such that it is entirely
// within bounds.
//
// If rect is already within bounds, no shift will be applied to rect and rect
// will be returned as-is.
//
// It is perfectly valid for the output rect to have a point along the edge of
// the bounds. If the desired output rect requires that no edges are parallel to
// edges of bounds, deflate the bounds by 1 to achieve this effect.
//
// Flutter: widgets/magnifier.dart MagnifierController.shiftWithinBounds
func MagnifierControllerShiftWithinBounds(rect, bounds ui.Rect) ui.Rect {
	if foundation.DebugMode {
		if rect.Width() > bounds.Width() {
			panic(errors.New(tagMagnifierRectLargerThanBounds, "attempted to shift a rect within bounds, but the rect has a greater width",
				"rect", rect.String(), "bounds", bounds.String()))
		}
		if rect.Height() > bounds.Height() {
			panic(errors.New(tagMagnifierRectLargerThanBounds, "attempted to shift a rect within bounds, but the rect has a greater height",
				"rect", rect.String(), "bounds", bounds.String()))
		}
	}

	rectShift := ui.OffsetZero
	if rect.Left < bounds.Left {
		rectShift = rectShift.Add(ui.Offset{Dx: bounds.Left - rect.Left})
	} else if rect.Right > bounds.Right {
		rectShift = rectShift.Add(ui.Offset{Dx: bounds.Right - rect.Right})
	}

	if rect.Top < bounds.Top {
		rectShift = rectShift.Add(ui.Offset{Dy: bounds.Top - rect.Top})
	} else if rect.Bottom > bounds.Bottom {
		rectShift = rectShift.Add(ui.Offset{Dy: bounds.Bottom - rect.Bottom})
	}

	return rect.Shift(rectShift)
}

// MagnifierDecorationOpts are the optional parameters of NewMagnifierDecoration.
//
// Flutter: widgets/magnifier.dart MagnifierDecoration.new
type MagnifierDecorationOpts struct {
	// Opacity is the opacity of the magnifier and decorations around the
	// magnifier; the default is 1.
	Opacity foundation.Opt[float64]
	// Shadows is a list of shadows cast by the shape.
	Shadows []painting.BoxShadow
	// Shape is the shape of the magnifier and the outline (border) around it;
	// the default is a RoundedRectangleBorder.
	Shape painting.ShapeBorder
}

// MagnifierDecoration is the decorations to put around the loupe in a
// RawMagnifier.
//
// Flutter: widgets/magnifier.dart MagnifierDecoration
type MagnifierDecoration struct {
	opacity float64
	shadows []painting.BoxShadow
	shape   painting.ShapeBorder
}

// NewMagnifierDecoration constructs a MagnifierDecoration.
//
// By default, MagnifierDecoration is a rectangular magnifier with no shadows,
// and fully opaque.
//
// Flutter: widgets/magnifier.dart MagnifierDecoration.new
func NewMagnifierDecoration(opts ...MagnifierDecorationOpts) *MagnifierDecoration {
	o := firstOpts(opts)
	d := &MagnifierDecoration{opacity: o.Opacity.OrElse(1.0), shadows: o.Shadows, shape: o.Shape}
	if d.shape == nil {
		d.shape = painting.NewRoundedRectangleBorder()
	}
	return d
}

// Opacity is the opacity of the magnifier and decorations around the magnifier.
//
// When this is 1.0, the magnified image shows in the shape of the magnifier.
// When this is less than 1.0, the magnified image is transparent and shows
// through the unmagnified background.
//
// Generally this is only useful for animating the magnifier in and out, as a
// transparent magnifier looks quite confusing.
//
// Flutter: widgets/magnifier.dart MagnifierDecoration.opacity
func (d *MagnifierDecoration) Opacity() float64 { return d.opacity }

// Shadows is a list of shadows cast by the shape (nil for none).
//
// If the shadows are offset, consider setting RawMagnifier's clip behavior to
// ClipHardEdge (or similar) to ensure the shadow does not occlude the magnifier
// (the shadow is drawn above the magnifier).
//
// If the shadows are not offset, consider using the outer blur style in the
// shadows instead, to avoid having to introduce a clip.
//
// In the event that the shape consists of a stack of borders, the shadow is
// drawn using the bounds of the last one.
//
// Flutter: widgets/magnifier.dart MagnifierDecoration.shadows
func (d *MagnifierDecoration) Shadows() []painting.BoxShadow { return d.shadows }

// Shape is the shape of the magnifier and the outline (border) around it.
//
// Shapes can be stacked. In that case, the magnifier and shadow are drawn
// according to the outside edge of the last shape, with the borders painted on
// top.
//
// Flutter: widgets/magnifier.dart MagnifierDecoration.shape
func (d *MagnifierDecoration) Shape() painting.ShapeBorder { return d.shape }

// Equal reports whether the other decoration has the same opacity, shadows and
// shape (Dart's ==).
//
// Flutter: widgets/magnifier.dart MagnifierDecoration.==
func (d *MagnifierDecoration) Equal(other *MagnifierDecoration) bool {
	if d == other {
		return true
	}
	if d == nil || other == nil {
		return false
	}
	return d.opacity == other.opacity && slices.Equal(d.shadows, other.shadows) && shapeBordersEqual(d.shape, other.shape)
}

// RawMagnifierOpts are the optional parameters of NewRawMagnifier.
//
// Flutter: widgets/magnifier.dart RawMagnifier.new
type RawMagnifierOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Child is an optional widget to position inside the lens of the
	// RawMagnifier.
	Child Widget
	// Decoration is this magnifier's decoration; the default draws nothing.
	Decoration *MagnifierDecoration
	// ClipBehavior is whether and how to clip the parts of the decoration that
	// render inside the loupe; the default is ClipNone.
	ClipBehavior ui.Clip
	// FocalPointOffset is the offset of the magnifier from the RawMagnifier's
	// center; the default is zero.
	FocalPointOffset ui.Offset
	// MagnificationScale is how "zoomed in" the magnification subject is in the
	// lens; the default is 1, which is no magnification.
	MagnificationScale foundation.Opt[float64]
}

// RawMagnifier is a common base class for magnifiers.
//
// This magnifying glass is useful for scenarios on mobile devices where the
// user's finger may be covering part of the screen where a granular action is
// being performed, such as navigating a small cursor with a drag gesture, on an
// image or text.
//
// A magnifier can be conveniently managed by MagnifierController, which handles
// showing and hiding the magnifier, with an optional entry / exit animation.
//
// Flutter: widgets/magnifier.dart RawMagnifier
type RawMagnifier struct {
	StatelessWidget
	child              Widget
	decoration         *MagnifierDecoration
	clipBehavior       ui.Clip
	focalPointOffset   ui.Offset
	magnificationScale float64
	size               ui.Size
}

// NewRawMagnifier constructs a RawMagnifier.
//
// By default, this magnifier uses the default MagnifierDecoration (which draws
// nothing), the focal point is directly under the magnifier, and there is no
// magnification; this means that a default magnifier will be entirely invisible
// to the naked eye, painting exactly what is under it, exactly where it was
// painted originally.
//
// In debug builds a magnification scale of 0 panics with the tag
// RawMagnifierScaleZero.
//
// Flutter: widgets/magnifier.dart RawMagnifier.new
func NewRawMagnifier(size ui.Size, opts ...RawMagnifierOpts) *RawMagnifier {
	o := firstOpts(opts)
	w := &RawMagnifier{
		StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child: o.Child, decoration: o.Decoration, clipBehavior: o.ClipBehavior,
		focalPointOffset: o.FocalPointOffset, magnificationScale: o.MagnificationScale.OrElse(1), size: size,
	}
	if w.decoration == nil {
		w.decoration = NewMagnifierDecoration()
	}
	if w.clipBehavior == ui.ClipUnspecified {
		w.clipBehavior = ui.ClipNone
	}
	if foundation.DebugMode && w.magnificationScale == 0 {
		panic(errors.New(tagRawMagnifierScaleZero, "magnification scale of 0 results in undefined behavior"))
	}
	return w
}

// Child is an optional widget to position inside the lens of the RawMagnifier.
//
// This is positioned over the RawMagnifier - it may be useful for tinting the
// RawMagnifier, or drawing a crosshair-like UI.
//
// Flutter: widgets/magnifier.dart RawMagnifier.child
func (w *RawMagnifier) Child() Widget { return w.child }

// Decoration is this magnifier's decoration.
//
// This sets the shape of the loupe, plus any borders and shadows that it casts.
// The default has no border and no shadow; combined with the default
// magnification scale of 1.0, this results in the magnifier having no visible
// effect.
//
// Flutter: widgets/magnifier.dart RawMagnifier.decoration
func (w *RawMagnifier) Decoration() *MagnifierDecoration { return w.decoration }

// ClipBehavior is whether and how to clip the parts of the decoration that
// render inside the loupe.
//
// Flutter: widgets/magnifier.dart RawMagnifier.clipBehavior
func (w *RawMagnifier) ClipBehavior() ui.Clip { return w.clipBehavior }

// FocalPointOffset is the offset of the magnifier from the RawMagnifier's
// center.
//
// For example, if the RawMagnifier is globally positioned at Offset(100, 100),
// and the focal point offset is Offset(-20, -20), then the RawMagnifier will
// see the content at global offset (80, 80).
//
// If left as zero, the RawMagnifier will show the content that is directly
// below it.
//
// Flutter: widgets/magnifier.dart RawMagnifier.focalPointOffset
func (w *RawMagnifier) FocalPointOffset() ui.Offset { return w.focalPointOffset }

// MagnificationScale is how "zoomed in" the magnification subject is in the
// lens. The default is 1.0, which is no magnification.
//
// Flutter: widgets/magnifier.dart RawMagnifier.magnificationScale
func (w *RawMagnifier) MagnificationScale() float64 { return w.magnificationScale }

// Size is the size of the magnifier. This does not include the border from the
// decoration; it only includes the size of the magnifier.
//
// Flutter: widgets/magnifier.dart RawMagnifier.size
func (w *RawMagnifier) Size() ui.Size { return w.size }

// Build builds the lens and, above it, the decoration.
//
// Flutter: widgets/magnifier.dart RawMagnifier.build
func (w *RawMagnifier) Build(bc BuildContext) Widget {
	return NewStack([]Widget{
		// The magnified image is clipped to the outer path of the shape.
		NewClipPathShape(w.decoration.shape,
			NewOpacity(w.decoration.opacity,
				newMagnifierRenderWidget(w.focalPointOffset, w.magnificationScale, NewSizedBoxFromSize(w.size, w.child)))),
		// Because BackdropFilter will filter any widgets before it, we apply
		// these styles after (i.e. in a younger sibling) to avoid the magnifier
		// from seeing its own styling.
		NewIgnorePointer(
			NewOpacity(w.decoration.opacity,
				NewClipPath(
					NewDecoratedBox(painting.NewShapeDecoration(w.decoration.shape, painting.ShapeDecorationOpts{Shadows: w.decoration.shadows}),
						NewSizedBoxFromSize(w.size, nil)),
					ClipPathOpts{ClipBehavior: w.clipBehavior, Clipper: &negativeClip{shape: w.decoration.shape}}))),
	}, StackOpts{Alignment: painting.AlignmentCenter, ClipBehavior: ui.ClipNone})
}

// negativeClip is a clip that renders everything except the inside of a shape.
//
// Flutter: widgets/magnifier.dart _NegativeClip
type negativeClip struct {
	rendering.CustomClipperBase[*ui.Path]
	shape painting.ShapeBorder
}

func (c *negativeClip) GetClip(size ui.Size) *ui.Path {
	path := ui.NewPath()
	path.SetFillType(ui.PathFillTypeEvenOdd)
	path.AddRect(ui.RectLargest)
	path.AddPath(c.shape.GetInnerPath(ui.OffsetZero.And(size), ui.TextDirectionUnspecified), ui.OffsetZero)
	return path
}

func (c *negativeClip) ShouldReclip(oldClipper rendering.CustomClipper[*ui.Path]) bool {
	old, ok := oldClipper.(*negativeClip)
	return !ok || !shapeBordersEqual(old.shape, c.shape)
}

// shapeBordersEqual compares two shape borders by value (Dart's ==).
func shapeBordersEqual(a, b painting.ShapeBorder) bool { return reflect.DeepEqual(a, b) }

// magnifierRenderWidget is Flutter's _Magnifier: the widget of the render
// object that magnifies what is painted behind it.
//
// Flutter: widgets/magnifier.dart _Magnifier
type magnifierRenderWidget struct {
	SingleChildRenderObjectWidget
	// focalPointOffset is the Offset that the center of the _Magnifier points
	// to, relative to the center of the magnifier.
	focalPointOffset ui.Offset
	// magnificationScale is the enlarge multiplier of the magnification. If
	// equal to 1.0, the content in the magnifier is true to its real size. If
	// greater than 1.0, the content appears bigger in the magnifier.
	magnificationScale float64
}

func newMagnifierRenderWidget(focalPointOffset ui.Offset, magnificationScale float64, child Widget) *magnifierRenderWidget {
	return &magnifierRenderWidget{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child), focalPointOffset: focalPointOffset, magnificationScale: magnificationScale}
}

func (w *magnifierRenderWidget) CreateRenderObject(BuildContext) rendering.RenderObject {
	return newRenderMagnification(w.focalPointOffset, w.magnificationScale)
}

func (w *magnifierRenderWidget) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderMagnification)
	r.setFocalPointOffset(w.focalPointOffset)
	r.setMagnificationScale(w.magnificationScale)
}

// renderMagnification is Flutter's _RenderMagnification: a proxy box that
// pushes a backdrop filter layer whose matrix magnifies the backdrop around the
// focal point.
//
// Flutter: widgets/magnifier.dart _RenderMagnification
type renderMagnification struct {
	rendering.RenderProxyBox
	focalPointOffset   ui.Offset
	magnificationScale float64
}

func newRenderMagnification(focalPointOffset ui.Offset, magnificationScale float64) *renderMagnification {
	r := &renderMagnification{focalPointOffset: focalPointOffset, magnificationScale: magnificationScale}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (r *renderMagnification) setFocalPointOffset(value ui.Offset) {
	if r.focalPointOffset == value {
		return
	}
	r.focalPointOffset = value
	r.MarkNeedsPaint()
}

func (r *renderMagnification) setMagnificationScale(value float64) {
	if r.magnificationScale == value {
		return
	}
	r.magnificationScale = value
	r.MarkNeedsPaint()
}

func (r *renderMagnification) AlwaysNeedsCompositing() bool { return true }

func (r *renderMagnification) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	thisCenter := painting.AlignmentCenter.AlongSize(r.Size()).Add(offset)
	matrix := ui.NewMatrix4Identity()
	matrix.Translate(
		float32(r.magnificationScale*((r.focalPointOffset.Dx*-1)-thisCenter.Dx)+thisCenter.Dx),
		float32(r.magnificationScale*((r.focalPointOffset.Dy*-1)-thisCenter.Dy)+thisCenter.Dy),
		0)
	matrix.Scale(float32(r.magnificationScale), float32(r.magnificationScale), float32(r.magnificationScale))
	filter := ui.NewImageFilterMatrix(matrix, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityHigh})

	layer, _ := r.Layer().(*rendering.BackdropFilterLayer)
	if layer == nil {
		layer = rendering.NewBackdropFilterLayer(rendering.BackdropFilterLayerOpts{Filter: filter})
		r.SetLayer(layer)
	} else {
		layer.SetFilter(filter)
	}

	context.PushLayer(layer, r.RenderProxyBox.Paint, offset, foundation.None[ui.Rect]())
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMagnifierRectLargerThanBounds = "MagnifierRectLargerThanBounds"
	tagRawMagnifierScaleZero         = "RawMagnifierScaleZero"
)
