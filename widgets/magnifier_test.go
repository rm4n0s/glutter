package widgets_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/magnifier_test.dart.
// Dart's FakeAsync is the fake clock of the tester here (Pump advances it).
// The _MockAnimationController of the Dart test is a subclass that counts the
// calls of forward and reverse, which a Go AnimationController cannot be; and
// in the Dart test it is not given to the MagnifierController either, so the
// port gives it one and looks at the status instead.

// magnifierContext pumps a TestWidgetsApp with a Placeholder and returns the
// context of the Placeholder.
func magnifierContext(ts *gluttertest.Tester, home Widget) BuildContext {
	var bc BuildContext
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewBuilder(func(b BuildContext) Widget {
		bc = b
		return home
	})}))
	return bc
}

func TestRawMagnifier(t *testing.T) {
	// The Dart test has one golden, with an opacity of 0.5. The opacity wraps the
	// magnifier in a layer, and a backdrop filter reads the layer it is in, as in
	// Skia and in every renderer here, so the lens of that golden shows what is
	// under it unmagnified (see the golden). Without an opacity layer (the
	// opacity of a real magnifier is 1) the lens is magnified: the second golden.
	for _, tt := range []struct {
		name    string
		opacity float64
		golden  string
	}{
		{"should render with correct focal point and decoration", 0.5, "widgets.magnifier.styled"},
		{"should render the magnified image without an opacity layer", 1, "widgets.magnifier.styled-opaque"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			const magnificationScale = 2.0
			magnifierSize := ui.Size{Width: 100, Height: 100}
			magnifierFocalPoint := ui.Offset{Dx: 50, Dy: 50}
			magnifierPosition := ui.Offset{Dx: 200, Dy: 200}

			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewContainer(
				NewStack([]Widget{
					NewPositioned(
						// Positioned so that it is right in the center of the magnifier
						// focal point.
						NewContainer(nil, ContainerOpts{
							Color: foundation.Some(ui.NewColor(0xFF000000)), // black
							// Since it is the size of the magnifier but over its
							// magnificationScale, it should take up the whole magnifier.
							Width:  foundation.Some(magnifierSize.Width * 1.5 / magnificationScale),
							Height: foundation.Some(magnifierSize.Height * 1.5 / magnificationScale),
						}),
						PositionedOpts{Left: foundation.Some(magnifierPosition.Dx + magnifierFocalPoint.Dx), Top: foundation.Some(magnifierPosition.Dy + magnifierFocalPoint.Dy)}),
					NewPositioned(
						NewRawMagnifier(magnifierSize, RawMagnifierOpts{
							FocalPointOffset:   magnifierFocalPoint,
							MagnificationScale: foundation.Some(magnificationScale),
							ClipBehavior:       ui.ClipHardEdge,
							Decoration: NewMagnifierDecoration(MagnifierDecorationOpts{
								Shadows: []painting.BoxShadow{painting.NewBoxShadow(painting.BoxShadowOpts{
									SpreadRadius: 10, BlurRadius: 10, Color: foundation.Some(ui.NewColor(0xFFFFFF00)), // yellow
									Offset: ui.Offset{Dx: 5, Dy: 5},
								})},
								Opacity: foundation.Some(tt.opacity),
							}),
						}),
						PositionedOpts{Left: foundation.Some(magnifierPosition.Dx), Top: foundation.Some(magnifierPosition.Dy)}),
				}),
				ContainerOpts{Color: foundation.Some(ui.NewColor(0xFF0000FF)), Width: foundation.Some(math.Inf(1)), Height: foundation.Some(math.Inf(1))}), // blue
			}))

			// Should look like a blue screen, with two black boxes. The larger black
			// box is in the magnifier, is outlined in yellow, and is doubled in size
			// (from magnification). The magnifier should be slightly transparent.
			ts.MatchesGolden(tt.golden)
		})
	}

	t.Run("a scale of 0 is rejected", func(t *testing.T) {
		skipUnlessDebug(t)
		v := panicOf(func() {
			NewRawMagnifier(ui.Size{Width: 1, Height: 1}, RawMagnifierOpts{MagnificationScale: foundation.Some(0.0)})
		})
		err, ok := v.(error)
		if !ok {
			t.Fatalf("panic value is %T (%v), want an error", v, v)
		}
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T", err)
		}
		if !appErr.HasRoute("NewRawMagnifier.RawMagnifierScaleZero") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	})

	t.Run("defaults", func(t *testing.T) {
		w := NewRawMagnifier(ui.Size{Width: 10, Height: 20})
		if w.MagnificationScale() != 1 || w.ClipBehavior() != ui.ClipNone || w.FocalPointOffset() != (ui.Offset{}) ||
			w.Decoration().Opacity() != 1 || w.Decoration().Shadows() != nil || w.Size() != (ui.Size{Width: 10, Height: 20}) {
			t.Errorf("defaults = scale %v, clip %v, focal %v, decoration %v", w.MagnificationScale(), w.ClipBehavior(), w.FocalPointOffset(), w.Decoration())
		}
	})

	t.Run("transition states", func(t *testing.T) {
		t.Run("should immediately remove from overlay on no animation controller", func(t *testing.T) {
			ts := gluttertest.New(t)
			magnifierController := NewMagnifierController()
			t.Cleanup(magnifierController.RemoveFromOverlay)
			testMagnifier := NewRawMagnifier(ui.Size{Width: 100, Height: 100})

			bc := magnifierContext(ts, NewPlaceholder(nil))

			magnifierController.Show(bc, func(BuildContext) Widget { return testMagnifier })

			ts.Pump()

			if magnifierController.OverlayEntry() == nil {
				t.Fatal("overlayEntry is nil after show")
			}

			magnifierController.Hide()
			ts.Pump()

			if magnifierController.OverlayEntry() != nil {
				t.Error("overlayEntry is not nil after hide")
			}
		})

		t.Run("should update shown based on animation status", func(t *testing.T) {
			ts := gluttertest.New(t)
			animationController := animation.NewAnimationController(vsyncOf{ts.Binding.Scheduler()}, animation.AnimationControllerOpts{Duration: foundation.Some(2 * time.Minute)})
			t.Cleanup(animationController.Dispose)
			magnifierController := NewMagnifierController(MagnifierControllerOpts{AnimationController: animationController})
			t.Cleanup(magnifierController.RemoveFromOverlay)
			testMagnifier := NewRawMagnifier(ui.Size{Width: 100, Height: 100})

			bc := magnifierContext(ts, NewPlaceholder(nil))

			magnifierController.Show(bc, func(BuildContext) Widget { return testMagnifier })

			ts.Pump()

			// No time has passed, so the animation controller has not completed.
			if got := magnifierController.AnimationController().Status(); got != animation.AnimationStatusForward {
				t.Errorf("status = %v, want forward", got)
			}
			if !magnifierController.Shown() {
				t.Error("shown = false, want true")
			}

			ts.Pump(animationController.Duration.Value())
			ts.PumpAndSettle()

			if got := magnifierController.AnimationController().Status(); got != animation.AnimationStatusCompleted {
				t.Errorf("status = %v, want completed", got)
			}
			if !magnifierController.Shown() {
				t.Error("shown = false, want true")
			}

			magnifierController.Hide()
			ts.Pump()

			if got := magnifierController.AnimationController().Status(); got != animation.AnimationStatusReverse {
				t.Errorf("status = %v, want reverse", got)
			}
			if magnifierController.Shown() {
				t.Error("shown = true, want false")
			}

			ts.Pump(animationController.Duration.Value())
			ts.PumpAndSettle()

			if got := magnifierController.AnimationController().Status(); got != animation.AnimationStatusDismissed {
				t.Errorf("status = %v, want dismissed", got)
			}
			if magnifierController.Shown() {
				t.Error("shown = true, want false")
			}
			// The exit animation is done, so the magnifier left the overlay.
			if magnifierController.OverlayEntry() != nil {
				t.Error("overlayEntry is not nil after the exit animation")
			}
		})
	})
}

func TestMagnifierController(t *testing.T) {
	t.Run("show", func(t *testing.T) {
		t.Run("should insert below below widget", func(t *testing.T) {
			ts := gluttertest.New(t)
			magnifierController := NewMagnifierController()
			t.Cleanup(magnifierController.RemoveFromOverlay)
			bc := magnifierContext(ts, NewText("text"))

			fakeMagnifier := NewPlaceholder(nil, PlaceholderOpts{Key: foundation.NewUniqueKey()})
			fakeBefore := NewPlaceholder(nil, PlaceholderOpts{Key: foundation.NewUniqueKey()})

			fakeBeforeOverlayEntry := NewOverlayEntry(func(BuildContext) Widget { return fakeBefore })
			t.Cleanup(func() {
				fakeBeforeOverlayEntry.Remove()
				fakeBeforeOverlayEntry.Dispose()
			})

			OverlayOf(bc).Insert(fakeBeforeOverlayEntry)
			magnifierController.Show(bc, func(BuildContext) Widget { return fakeMagnifier }, MagnifierControllerShowOpts{Below: fakeBeforeOverlayEntry})

			ts.PumpAndSettle()

			allOverlayChildren := ts.Elements(gluttertest.Descendant(gluttertest.ByType[*Overlay](), gluttertest.ByType[*Placeholder]()))
			if len(allOverlayChildren) != 2 {
				t.Fatalf("placeholders in the overlay = %d, want 2", len(allOverlayChildren))
			}

			// Expect the magnifier to be the first child, even though it was
			// inserted after the fakeBefore.
			if allOverlayChildren[len(allOverlayChildren)-1].Widget().Key() != fakeBefore.Key() {
				t.Error("the last child of the overlay is not fakeBefore")
			}
			if allOverlayChildren[0].Widget().Key() != fakeMagnifier.Key() {
				t.Error("the first child of the overlay is not the magnifier")
			}
		})

		t.Run("should insert newly built widget without animating out if overlay != null", func(t *testing.T) {
			ts := gluttertest.New(t)
			animationController := animation.NewAnimationController(vsyncOf{ts.Binding.Scheduler()}, animation.AnimationControllerOpts{Duration: foundation.Some(time.Minute)})
			t.Cleanup(animationController.Dispose)
			magnifierController := NewMagnifierController(MagnifierControllerOpts{AnimationController: animationController})
			t.Cleanup(magnifierController.RemoveFromOverlay)

			testMagnifier := NewRawMagnifier(ui.Size{Width: 100, Height: 100})
			testMagnifier2 := NewRawMagnifier(ui.Size{Width: 100, Height: 100})

			bc := magnifierContext(ts, NewPlaceholder(nil))

			magnifierController.Show(bc, func(BuildContext) Widget { return testMagnifier })

			ts.Pump()

			ts.Pump(animationController.Duration.Value())
			ts.PumpAndSettle()

			magnifierController.Show(bc, func(BuildContext) Widget { return testMagnifier2 })

			ts.Pump()

			// The old magnifier is replaced without playing an exit animation.
			if got := animationController.Status(); got == animation.AnimationStatusReverse {
				t.Errorf("status = %v: should not have called reverse on animation controller due to force remove", got)
			}

			if ts.Count(gluttertest.ByWidget(testMagnifier2)) != 1 {
				t.Error("the new magnifier is not in the tree")
			}
			if ts.Count(gluttertest.ByWidget(testMagnifier)) != 0 {
				t.Error("the old magnifier is still in the tree")
			}
		})
	})

	t.Run("hide waits for the exit animation unless it is kept in the overlay", func(t *testing.T) {
		tests := []struct {
			name          string
			keepInOverlay bool
			wantEntry     bool
		}{
			{"removed", false, false},
			{"kept", true, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ts := gluttertest.New(t)
				animationController := animation.NewAnimationController(vsyncOf{ts.Binding.Scheduler()}, animation.AnimationControllerOpts{Duration: foundation.Some(time.Second)})
				t.Cleanup(animationController.Dispose)
				magnifierController := NewMagnifierController(MagnifierControllerOpts{AnimationController: animationController})
				t.Cleanup(magnifierController.RemoveFromOverlay)
				bc := magnifierContext(ts, NewPlaceholder(nil))

				shown := magnifierController.Show(bc, func(BuildContext) Widget { return NewRawMagnifier(ui.Size{Width: 10, Height: 10}) })
				ts.PumpAndSettle()
				if completed, ended := shown.Completed(); !completed || !ended {
					t.Errorf("the future of show completed = %v, ended = %v, want both true", completed, ended)
				}

				hidden := magnifierController.Hide(MagnifierControllerHideOpts{KeepInOverlay: tt.keepInOverlay})
				if magnifierController.OverlayEntry() == nil {
					t.Fatal("the magnifier left the overlay before its exit animation ended")
				}
				if _, ended := hidden.Completed(); ended {
					t.Error("the future of hide ended before the animation")
				}
				ts.PumpAndSettle()
				if got := magnifierController.OverlayEntry() != nil; got != tt.wantEntry {
					t.Errorf("overlayEntry != nil = %v, want %v", got, tt.wantEntry)
				}
				if completed, ended := hidden.Completed(); !completed || !ended {
					t.Errorf("the future of hide completed = %v, ended = %v, want both true", completed, ended)
				}
			})
		}
	})

	t.Run("hide without a magnifier does nothing", func(t *testing.T) {
		magnifierController := NewMagnifierController()
		future := magnifierController.Hide()
		if completed, ended := future.Completed(); !completed || !ended {
			t.Errorf("completed = %v, ended = %v, want both true", completed, ended)
		}
		if magnifierController.Shown() {
			t.Error("shown = true with no magnifier")
		}
	})

	t.Run("shift within bounds", func(t *testing.T) {
		tests := []struct {
			bounds, input, want ui.Rect
		}{
			{ui.NewRectFromLTRB(0, 0, 100, 100), ui.NewRectFromLTRB(-100, -100, -80, -80), ui.NewRectFromLTRB(0, 0, 20, 20)},
			{ui.NewRectFromLTRB(0, 0, 100, 100), ui.NewRectFromLTRB(0, 0, 20, 20), ui.NewRectFromLTRB(0, 0, 20, 20)},
			{ui.NewRectFromLTRB(0, 0, 100, 100), ui.NewRectFromLTRB(110, 0, 120, 10), ui.NewRectFromLTRB(90, 0, 100, 10)},
			{ui.NewRectFromLTRB(0, 0, 100, 100), ui.NewRectFromLTRB(110, 110, 120, 120), ui.NewRectFromLTRB(90, 90, 100, 100)},
		}
		for _, tt := range tests {
			t.Run("should shift "+tt.input.String()+" to "+tt.want.String()+" for bounds "+tt.bounds.String(), func(t *testing.T) {
				if got := MagnifierControllerShiftWithinBounds(tt.input, tt.bounds); got != tt.want {
					t.Errorf("shift = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("shifting a rect larger than the bounds is rejected", func(t *testing.T) {
		skipUnlessDebug(t)
		tests := []struct {
			name         string
			rect, bounds ui.Rect
		}{
			{"wider", ui.NewRectFromLTRB(0, 0, 200, 10), ui.NewRectFromLTRB(0, 0, 100, 100)},
			{"taller", ui.NewRectFromLTRB(0, 0, 10, 200), ui.NewRectFromLTRB(0, 0, 100, 100)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				v := panicOf(func() { MagnifierControllerShiftWithinBounds(tt.rect, tt.bounds) })
				err, ok := v.(error)
				if !ok {
					t.Fatalf("panic value is %T (%v), want an error", v, v)
				}
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T", err)
				}
				if !appErr.HasRoute("MagnifierControllerShiftWithinBounds.MagnifierRectLargerThanBounds") {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			})
		}
	})
}

func TestMagnifierInfo(t *testing.T) {
	t.Run("MagnifierInfo.toString", func(t *testing.T) {
		want := "MagnifierInfo(position: Offset(0.0, 0.0), line: Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), " +
			"caret: Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), field: Rect.fromLTRB(0.0, 0.0, 0.0, 0.0))"
		if got := MagnifierInfoEmpty.String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("NewMagnifierInfo keeps the fields in order", func(t *testing.T) {
		pos, caret, field, line := ui.Offset{Dx: 1, Dy: 2}, ui.NewRectFromLTRB(1, 2, 3, 4), ui.NewRectFromLTRB(5, 6, 7, 8), ui.NewRectFromLTRB(9, 10, 11, 12)
		info := NewMagnifierInfo(pos, caret, field, line)
		if info.GlobalGesturePosition != pos || info.CaretRect != caret || info.FieldBounds != field || info.CurrentLineBoundaries != line {
			t.Errorf("info = %v", info)
		}
		if info == MagnifierInfoEmpty {
			t.Error("a filled info equals the empty one")
		}
	})
}

func TestTextMagnifierConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      *TextMagnifierConfiguration
		wantBuilds  bool
		wantHandles bool
	}{
		{"disabled", TextMagnifierConfigurationDisabled, false, true},
		{"without a builder", NewTextMagnifierConfiguration(), false, true},
		{"with a builder", NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{MagnifierBuilder: func(BuildContext, *MagnifierController, *foundation.ValueNotifier[MagnifierInfo]) Widget {
			return NewPlaceholder(nil)
		}}), true, true},
		{"without handles", NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{HideHandlesInMagnifier: true}), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := foundation.NewValueNotifier(MagnifierInfoEmpty)
			defer info.Dispose()
			built := tt.config.MagnifierBuilder()(nil, NewMagnifierController(), info)
			if (built != nil) != tt.wantBuilds {
				t.Errorf("built = %v, want a widget: %v", built, tt.wantBuilds)
			}
			if tt.config.ShouldDisplayHandlesInMagnifier() != tt.wantHandles {
				t.Errorf("ShouldDisplayHandlesInMagnifier = %v, want %v", tt.config.ShouldDisplayHandlesInMagnifier(), tt.wantHandles)
			}
		})
	}
}

func TestMagnifierDecorationEqual(t *testing.T) {
	shadow := painting.NewBoxShadow(painting.BoxShadowOpts{BlurRadius: 2})
	tests := []struct {
		name string
		a, b *MagnifierDecoration
		want bool
	}{
		{"defaults", NewMagnifierDecoration(), NewMagnifierDecoration(), true},
		{"same shadows", NewMagnifierDecoration(MagnifierDecorationOpts{Shadows: []painting.BoxShadow{shadow}}),
			NewMagnifierDecoration(MagnifierDecorationOpts{Shadows: []painting.BoxShadow{shadow}}), true},
		{"other opacity", NewMagnifierDecoration(), NewMagnifierDecoration(MagnifierDecorationOpts{Opacity: foundation.Some(0.5)}), false},
		{"other shadows", NewMagnifierDecoration(), NewMagnifierDecoration(MagnifierDecorationOpts{Shadows: []painting.BoxShadow{shadow}}), false},
		{"other shape", NewMagnifierDecoration(),
			NewMagnifierDecoration(MagnifierDecorationOpts{Shape: painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusCircular(4)})}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal = %v, want %v", got, tt.want)
			}
		})
	}
}
