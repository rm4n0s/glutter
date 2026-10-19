package widgets

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/scheduler"
)

// ScrollControllerCallback is called with the position that a controller
// attached or detached.
//
// Flutter: widgets/scroll_controller.dart ScrollControllerCallback
type ScrollControllerCallback func(position ScrollPosition)

// ScrollController controls one or more scrollables: it reads and changes their
// scroll position, and notifies its listeners when a position changes. Pass it
// to a scrollable; it creates the position (CreateScrollPosition) and attaches
// it.
//
// Implement it by embedding ScrollControllerBase (InitScrollControllerBase) to
// change how the positions are created or attached.
//
// Flutter: widgets/scroll_controller.dart ScrollController
type ScrollController interface {
	foundation.Listenable
	// InitialScrollOffset is the initial scroll offset that the controller gives
	// the positions it creates.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.initialScrollOffset
	InitialScrollOffset() float64
	// KeepScrollOffset is whether the positions save their offset in the
	// PageStorage.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.keepScrollOffset
	KeepScrollOffset() bool
	// DebugLabel names the controller in debug output.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.debugLabel
	DebugLabel() string
	// Positions are the scroll positions attached to the controller.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.positions
	Positions() []ScrollPosition
	// HasClients is whether any position is attached.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.hasClients
	HasClients() bool
	// Position is the position attached; it panics with the tag
	// ScrollControllerNotAttached in debug builds when there is none, and with
	// ScrollControllerMultipleClients when there are several.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.position
	Position() ScrollPosition
	// Offset is the scroll offset of the attached position.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.offset
	Offset() float64
	// AnimateTo animates the positions to the offset.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.animateTo
	AnimateTo(offset float64, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture
	// JumpTo changes the positions to the offset.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.jumpTo
	JumpTo(value float64)
	// Attach registers the position to the controller.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.attach
	Attach(position ScrollPosition)
	// Detach unregisters the position from the controller.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.detach
	Detach(position ScrollPosition)
	// CreateScrollPosition creates the position for the scrollable.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.createScrollPosition
	CreateScrollPosition(physics ScrollPhysics, context ScrollContext, oldPosition ScrollPosition) ScrollPosition
	// Dispose releases the controller.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.dispose
	Dispose()
	// DebugFillDescription appends the parts of the description.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.debugFillDescription
	DebugFillDescription(description []string) []string
	// String describes the controller.
	//
	// Flutter: widgets/scroll_controller.dart ScrollController.toString
	String() string
}

// ScrollControllerOpts are the optional parameters of NewScrollController.
//
// Flutter: widgets/scroll_controller.dart ScrollController.new
type ScrollControllerOpts struct {
	// InitialScrollOffset is the initial offset of the positions; the default is
	// 0.
	InitialScrollOffset float64
	// KeepScrollOffset is whether the positions save their offset in the
	// PageStorage; the default is true.
	KeepScrollOffset foundation.Opt[bool]
	// DebugLabel names the controller in debug output.
	DebugLabel string
	// OnAttach is called when a position is attached.
	OnAttach ScrollControllerCallback
	// OnDetach is called when a position is detached.
	OnDetach ScrollControllerCallback
}

// ScrollControllerBase implements ScrollController. It is what NewScrollController
// returns; embed it to extend the controller.
//
// Flutter: widgets/scroll_controller.dart ScrollController
type ScrollControllerBase struct {
	foundation.ChangeNotifier
	self                ScrollController
	initialScrollOffset float64
	keepScrollOffset    bool
	onAttach, onDetach  ScrollControllerCallback
	debugLabel          string
	positions           []ScrollPosition
	notifyHandle        map[ScrollPosition]foundation.ListenerHandle
}

// NewScrollController creates a controller.
//
// Flutter: widgets/scroll_controller.dart ScrollController.new
func NewScrollController(opts ...ScrollControllerOpts) *ScrollControllerBase {
	c := &ScrollControllerBase{}
	InitScrollControllerBase(c, c, firstOpts(opts))
	return c
}

// InitScrollControllerBase sets the outer value and the options.
//
// Flutter: widgets/scroll_controller.dart ScrollController.new
func InitScrollControllerBase(c *ScrollControllerBase, self ScrollController, o ScrollControllerOpts) {
	c.self = self
	c.initialScrollOffset = o.InitialScrollOffset
	c.keepScrollOffset = o.KeepScrollOffset.OrElse(true)
	c.debugLabel = o.DebugLabel
	c.onAttach, c.onDetach = o.OnAttach, o.OnDetach
	c.notifyHandle = map[ScrollPosition]foundation.ListenerHandle{}
}

// InitialScrollOffset is the initial offset of the positions.
//
// Flutter: widgets/scroll_controller.dart ScrollController.initialScrollOffset
func (c *ScrollControllerBase) InitialScrollOffset() float64 { return c.initialScrollOffset }

// KeepScrollOffset is whether the positions save their offset in the
// PageStorage.
//
// Flutter: widgets/scroll_controller.dart ScrollController.keepScrollOffset
func (c *ScrollControllerBase) KeepScrollOffset() bool { return c.keepScrollOffset }

// DebugLabel names the controller in debug output.
//
// Flutter: widgets/scroll_controller.dart ScrollController.debugLabel
func (c *ScrollControllerBase) DebugLabel() string { return c.debugLabel }

// Positions are the scroll positions attached to the controller.
//
// Flutter: widgets/scroll_controller.dart ScrollController.positions
func (c *ScrollControllerBase) Positions() []ScrollPosition { return slices.Clone(c.positions) }

// HasClients is whether any position is attached.
//
// Flutter: widgets/scroll_controller.dart ScrollController.hasClients
func (c *ScrollControllerBase) HasClients() bool { return len(c.positions) > 0 }

// Position is the position attached to the controller.
//
// Flutter: widgets/scroll_controller.dart ScrollController.position
func (c *ScrollControllerBase) Position() ScrollPosition {
	if foundation.DebugMode {
		if len(c.positions) == 0 {
			panic(errors.New("ScrollControllerNotAttached", "ScrollController not attached to any scroll views"))
		}
		if len(c.positions) != 1 {
			panic(errors.New("ScrollControllerMultipleClients", "ScrollController attached to multiple scroll views", "clients", len(c.positions)))
		}
	}
	return c.positions[0]
}

// Offset is the scroll offset of the attached position.
//
// Flutter: widgets/scroll_controller.dart ScrollController.offset
func (c *ScrollControllerBase) Offset() float64 { return c.self.Position().Pixels() }

func (c *ScrollControllerBase) assertAttached() {
	if foundation.DebugMode && len(c.positions) == 0 {
		panic(errors.New("ScrollControllerNotAttached", "ScrollController not attached to any scroll views"))
	}
}

// AnimateTo animates every attached position to the offset. The future
// completes when all the animations have. In debug builds it panics with the tag
// ScrollControllerNotAttached when no position is attached.
//
// Flutter: widgets/scroll_controller.dart ScrollController.animateTo
func (c *ScrollControllerBase) AnimateTo(offset float64, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	c.assertAttached()
	futures := make([]*scheduler.TickerFuture, 0, len(c.positions))
	for _, p := range slices.Clone(c.positions) {
		futures = append(futures, p.AnimateTo(offset, duration, curve))
	}
	return scheduler.WaitTickerFutures(futures...)
}

// JumpTo changes every attached position to the value. In debug builds it panics
// with the tag ScrollControllerNotAttached when no position is attached.
//
// Flutter: widgets/scroll_controller.dart ScrollController.jumpTo
func (c *ScrollControllerBase) JumpTo(value float64) {
	c.assertAttached()
	for _, p := range slices.Clone(c.positions) {
		p.JumpTo(value)
	}
}

// Attach registers the position, and listens to it. In debug builds it panics
// with the tag ScrollControllerPositionAttached when it is attached already.
//
// Flutter: widgets/scroll_controller.dart ScrollController.attach
func (c *ScrollControllerBase) Attach(position ScrollPosition) {
	if foundation.DebugMode && slices.Contains(c.positions, position) {
		panic(errors.New("ScrollControllerPositionAttached", "the position is attached to the controller already"))
	}
	c.positions = append(c.positions, position)
	c.notifyHandle[position] = position.AddListener(c.NotifyListeners)
	if c.onAttach != nil {
		c.onAttach(position)
	}
}

// Detach unregisters the position. In debug builds it panics with the tag
// ScrollControllerPositionNotAttached when it is not attached.
//
// Flutter: widgets/scroll_controller.dart ScrollController.detach
func (c *ScrollControllerBase) Detach(position ScrollPosition) {
	i := slices.Index(c.positions, position)
	if i < 0 {
		if foundation.DebugMode {
			panic(errors.New("ScrollControllerPositionNotAttached", "the position is not attached to the controller"))
		}
		return
	}
	if c.onDetach != nil {
		c.onDetach(position)
	}
	position.RemoveListener(c.notifyHandle[position])
	delete(c.notifyHandle, position)
	c.positions = slices.Delete(c.positions, i, i+1)
}

// Dispose stops listening to the positions.
//
// Flutter: widgets/scroll_controller.dart ScrollController.dispose
func (c *ScrollControllerBase) Dispose() {
	for _, p := range c.positions {
		p.RemoveListener(c.notifyHandle[p])
	}
	c.ChangeNotifier.Dispose()
}

// CreateScrollPosition creates a ScrollPositionWithSingleContext with the
// initial offset of the controller.
//
// Flutter: widgets/scroll_controller.dart ScrollController.createScrollPosition
func (c *ScrollControllerBase) CreateScrollPosition(physics ScrollPhysics, context ScrollContext, oldPosition ScrollPosition) ScrollPosition {
	return NewScrollPositionWithSingleContext(physics, context, ScrollPositionWithSingleContextOpts{
		InitialPixels: foundation.Some(c.self.InitialScrollOffset()), KeepScrollOffset: foundation.Some(c.keepScrollOffset),
		OldPosition: oldPosition, DebugLabel: c.debugLabel})
}

// DebugFillDescription appends the label, the initial offset and the clients.
//
// Flutter: widgets/scroll_controller.dart ScrollController.debugFillDescription
func (c *ScrollControllerBase) DebugFillDescription(description []string) []string {
	if c.debugLabel != "" {
		description = append(description, c.debugLabel)
	}
	if c.self.InitialScrollOffset() != 0 {
		description = append(description, "initialScrollOffset: "+dartfmt.ToStringAsFixed(c.self.InitialScrollOffset(), 1)+", ")
	}
	switch len(c.positions) {
	case 0:
		description = append(description, "no clients")
	case 1:
		// Don't actually list the client, since it's more typically a
		// _ScrollPosition for which toString() is more complicated.
		description = append(description, "one client, offset "+dartfmt.ToStringAsFixed(c.self.Offset(), 1))
	default:
		description = append(description, strconv.Itoa(len(c.positions))+" clients")
	}
	return description
}

// String is the identity of the controller and its description.
//
// Flutter: widgets/scroll_controller.dart ScrollController.toString
func (c *ScrollControllerBase) String() string {
	return foundation.DescribeIdentity(c.self) + "(" + strings.Join(c.self.DebugFillDescription(nil), ", ") + ")"
}

// TrackingScrollController is a controller that remembers the last position that
// was updated, and starts new positions at its offset. Several scrollables can
// share it.
//
// Flutter: widgets/scroll_controller.dart TrackingScrollController
type TrackingScrollController struct {
	ScrollControllerBase
	positionToListener map[ScrollPosition]foundation.ListenerHandle
	lastUpdated        ScrollPosition
	lastUpdatedOffset  foundation.Opt[float64]
}

// NewTrackingScrollController creates the controller.
//
// Flutter: widgets/scroll_controller.dart TrackingScrollController.new
func NewTrackingScrollController(opts ...ScrollControllerOpts) *TrackingScrollController {
	c := &TrackingScrollController{positionToListener: map[ScrollPosition]foundation.ListenerHandle{}}
	InitScrollControllerBase(&c.ScrollControllerBase, c, firstOpts(opts))
	return c
}

// MostRecentlyUpdatedPosition is the position that notified last, or nil.
//
// Flutter: widgets/scroll_controller.dart TrackingScrollController.mostRecentlyUpdatedPosition
func (c *TrackingScrollController) MostRecentlyUpdatedPosition() ScrollPosition { return c.lastUpdated }

// InitialScrollOffset is the offset of the last position that updated, or the
// initial offset.
//
// Flutter: widgets/scroll_controller.dart TrackingScrollController.initialScrollOffset
func (c *TrackingScrollController) InitialScrollOffset() float64 {
	return c.lastUpdatedOffset.OrElse(c.ScrollControllerBase.InitialScrollOffset())
}

// Attach registers the position and tracks its updates.
//
// Flutter: widgets/scroll_controller.dart TrackingScrollController.attach
func (c *TrackingScrollController) Attach(position ScrollPosition) {
	c.ScrollControllerBase.Attach(position)
	if foundation.DebugMode {
		if _, ok := c.positionToListener[position]; ok {
			panic(errors.New("ScrollControllerPositionAttached", "the position is tracked by the controller already"))
		}
	}
	c.positionToListener[position] = position.AddListener(func() {
		c.lastUpdated = position
		c.lastUpdatedOffset = foundation.Some(position.Pixels())
	})
}

// Detach unregisters the position and stops tracking it.
//
// Flutter: widgets/scroll_controller.dart TrackingScrollController.detach
func (c *TrackingScrollController) Detach(position ScrollPosition) {
	c.ScrollControllerBase.Detach(position)
	if foundation.DebugMode {
		if _, ok := c.positionToListener[position]; !ok {
			panic(errors.New("ScrollControllerPositionNotAttached", "the position is not tracked by the controller"))
		}
	}
	position.RemoveListener(c.positionToListener[position])
	delete(c.positionToListener, position)
	if c.lastUpdated == position {
		c.lastUpdated = nil
	}
	if len(c.positionToListener) == 0 {
		c.lastUpdatedOffset = foundation.Opt[float64]{}
	}
}

// Dispose stops tracking the positions.
//
// Flutter: widgets/scroll_controller.dart TrackingScrollController.dispose
func (c *TrackingScrollController) Dispose() {
	for _, p := range c.Positions() {
		if h, ok := c.positionToListener[p]; ok {
			p.RemoveListener(h)
		}
	}
	c.ScrollControllerBase.Dispose()
}
