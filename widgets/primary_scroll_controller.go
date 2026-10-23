package widgets

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
)

// PrimaryScrollControllerOpts are the optional parameters of
// NewPrimaryScrollController.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.new
type PrimaryScrollControllerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AutomaticallyInheritForPlatforms are the platforms on which the scroll
	// views below inherit the controller by default; the default is the mobile
	// platforms (Android, iOS, Fuchsia).
	AutomaticallyInheritForPlatforms []foundation.TargetPlatform
	// ScrollDirection is the axis of the scroll views that inherit the controller
	// by default; the default is the vertical axis.
	ScrollDirection foundation.Opt[painting.Axis]
}

// PrimaryScrollController associates a ScrollController with a subtree. The
// scroll views below it that do not have a controller of their own use it on the
// platforms and axis it names, and widgets such as a scrollbar find the controller
// of the primary scroll view with it.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController
type PrimaryScrollController struct {
	InheritedWidget
	controller                       ScrollController
	scrollDirection                  foundation.Opt[painting.Axis]
	automaticallyInheritForPlatforms []foundation.TargetPlatform
}

var mobilePlatforms = []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia}

// NewPrimaryScrollController creates the widget.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.new
func NewPrimaryScrollController(controller ScrollController, child Widget, opts ...PrimaryScrollControllerOpts) *PrimaryScrollController {
	o := firstOpts(opts)
	platforms := o.AutomaticallyInheritForPlatforms
	if platforms == nil {
		platforms = mobilePlatforms
	}
	direction := o.ScrollDirection
	if !direction.IsSome() {
		direction = foundation.Some(painting.AxisVertical)
	}
	return &PrimaryScrollController{NewInheritedWidgetBase(child, WidgetOpts{Key: o.Key}), controller, direction, platforms}
}

// NewPrimaryScrollControllerNone creates a widget that blocks the primary scroll
// controller of the widgets above it: there is none below.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.none
func NewPrimaryScrollControllerNone(child Widget, opts ...WidgetOpts) *PrimaryScrollController {
	return &PrimaryScrollController{InheritedWidget: NewInheritedWidgetBase(child, opts...), automaticallyInheritForPlatforms: []foundation.TargetPlatform{}}
}

// Controller is the controller, or nil for the widget made by
// NewPrimaryScrollControllerNone.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.controller
func (w *PrimaryScrollController) Controller() ScrollController { return w.controller }

// ScrollDirection is the axis of the scroll views that inherit the controller by
// default; none for NewPrimaryScrollControllerNone.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.scrollDirection
func (w *PrimaryScrollController) ScrollDirection() foundation.Opt[painting.Axis] {
	return w.scrollDirection
}

// AutomaticallyInheritForPlatforms are the platforms on which the scroll views
// below inherit the controller by default.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.automaticallyInheritForPlatforms
func (w *PrimaryScrollController) AutomaticallyInheritForPlatforms() []foundation.TargetPlatform {
	return slices.Clone(w.automaticallyInheritForPlatforms)
}

// PrimaryScrollControllerShouldInherit is whether a scroll view along the axis
// that does not have a controller uses the closest PrimaryScrollController above
// the context: on the platforms and along the axis it names.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.shouldInherit
func PrimaryScrollControllerShouldInherit(bc BuildContext, scrollDirection painting.Axis) bool {
	result, ok := FindAncestorWidgetOfExactType[*PrimaryScrollController](bc)
	if !ok {
		return false
	}
	platform := ScrollConfigurationOf(bc).GetPlatform(bc)
	if slices.Contains(result.automaticallyInheritForPlatforms, platform) {
		return result.scrollDirection.IsSome() && result.scrollDirection.Value() == scrollDirection
	}
	return false
}

// PrimaryScrollControllerMaybeOf is the controller of the closest
// PrimaryScrollController above the context, or nil. The context depends on it.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.maybeOf
func PrimaryScrollControllerMaybeOf(bc BuildContext) ScrollController {
	if w, ok := DependOn[*PrimaryScrollController](bc); ok {
		return w.controller
	}
	return nil
}

// PrimaryScrollControllerOf is the controller of the closest
// PrimaryScrollController above the context. In debug builds it panics with the
// tag PrimaryScrollControllerMissing when there is none.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.of
func PrimaryScrollControllerOf(bc BuildContext) ScrollController {
	c := PrimaryScrollControllerMaybeOf(bc)
	if isNil(c) && foundation.DebugMode {
		panic(errors.New(tagPrimaryScrollControllerMissing,
			"PrimaryScrollControllerOf() was called with a context that does not contain a PrimaryScrollController widget; no PrimaryScrollController widget ancestor could be found starting from the context that was passed to PrimaryScrollControllerOf(); this can happen because you are using a widget that looks for a PrimaryScrollController ancestor, but no such ancestor exists",
			"context", bc.(Element).String()))
	}
	return c
}

// UpdateShouldNotify is whether the controller changed.
//
// Flutter: widgets/primary_scroll_controller.dart PrimaryScrollController.updateShouldNotify
func (w *PrimaryScrollController) UpdateShouldNotify(old Widget) bool {
	return w.controller != old.(*PrimaryScrollController).controller
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPrimaryScrollControllerMissing = "PrimaryScrollControllerMissing"
)
