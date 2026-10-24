package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from
// packages/flutter/test/material/predictive_back_page_transitions_builder_test.dart.
// flutter_test sends the back gesture to the channel flutter/backgesture; the
// tester plays the same calls on the platform dispatcher (Tester.StartBackGesture
// and the rest). The cases that Flutter runs for TargetPlatformVariant.all() run
// on Android and on Linux, where the other platforms behave as Linux; "x" and
// "y" in the update message of the Dart tests are not the key of the touch offset,
// so those updates have no touch offset.

// predictiveBackBuilderCase is a builder under test with the names of the widget
// of its gesture transition and of its fallback.
type predictiveBackBuilderCase struct {
	name        string
	builder     widgets.PageTransitionsBuilder
	predictive  string
	fallback    string
	withFallbck func(c ui.Color) widgets.PageTransitionsBuilder
}

func predictiveBackBuilderCases() []predictiveBackBuilderCase {
	return []predictiveBackBuilderCase{
		{"PredictiveBackPageTransitionsBuilder", NewPredictiveBackPageTransitionsBuilder(),
			"predictiveBackSharedElementPageTransition", "fadeForwardsPageTransition",
			func(c ui.Color) widgets.PageTransitionsBuilder {
				return NewPredictiveBackPageTransitionsBuilder(PredictiveBackPageTransitionsBuilderOpts{FallbackColor: foundation.Some(c)})
			}},
		{"PredictiveBackFullscreenPageTransitionsBuilder", NewPredictiveBackFullscreenPageTransitionsBuilder(),
			"predictiveBackFullscreenPageTransition", "zoomPageTransition",
			func(c ui.Color) widgets.PageTransitionsBuilder {
				return NewPredictiveBackFullscreenPageTransitionsBuilder(PredictiveBackFullscreenPageTransitionsBuilderOpts{FallbackColor: foundation.Some(c)})
			}},
	}
}

// allPlatformsTheme is a theme whose page transitions are the builder on every
// platform.
func allPlatformsTheme(builder widgets.PageTransitionsBuilder) *ThemeData {
	builders := map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{}
	for p := foundation.TargetPlatformAndroid; p <= foundation.TargetPlatformWindows; p++ {
		builders[p] = builder
	}
	return NewThemeData(ThemeDataOpts{PageTransitionsTheme: NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: builders})})
}

// pushBRoutes are the routes '/' with a "push" button and '/b' with the text
// "page b".
func pushBRoutes() map[string]widgets.WidgetBuilder {
	return map[string]widgets.WidgetBuilder{
		"/": func(bc widgets.BuildContext) widgets.Widget {
			return NewMaterial(NewTextButton(func() { widgets.NavigatorPushNamed(bc, "/b") }, widgets.NewText("push")))
		},
		"/b": func(widgets.BuildContext) widgets.Widget { return widgets.NewText("page b") },
	}
}

var (
	backStart = gluttertest.BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 5, Dy: 300})}
	backAt100 = gluttertest.BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 100, Dy: 300})}
)

// forAndroidAndLinux runs the test on the two platforms that stand for
// TargetPlatformVariant.all().
func forAndroidAndLinux(t *testing.T, fn func(t *testing.T, platform foundation.TargetPlatform)) {
	t.Helper()
	for _, p := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformLinux} {
		t.Run(p.String(), func(t *testing.T) {
			withPlatform(t, p)
			fn(t, p)
		})
	}
}

// expectCounts checks the number of widgets of the gesture transition and of the
// fallback transition.
func expectCounts(t *testing.T, ts *gluttertest.Tester, c predictiveBackBuilderCase, step string, predictive, fallback int) {
	t.Helper()
	if got := ts.Count(transitionWidget(c.predictive)); got != predictive {
		t.Errorf("%s: %d %s, want %d", step, got, c.predictive, predictive)
	}
	if got := ts.Count(transitionWidget(c.fallback)); got != fallback {
		t.Errorf("%s: %d %s, want %d", step, got, c.fallback, fallback)
	}
}

func expectShown(t *testing.T, ts *gluttertest.Tester, step string, texts map[string]bool) {
	t.Helper()
	for text, want := range texts {
		if got := ts.Exists(gluttertest.ByText(text)); got != want {
			t.Errorf("%s: %q shown = %v, want %v", step, text, got, want)
		}
	}
}

func TestPredictiveBackPageTransitionsBuilder(t *testing.T) {
	for _, c := range predictiveBackBuilderCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Run("supports predictive back on Android", func(t *testing.T) {
				forAndroidAndLinux(t, func(t *testing.T, platform foundation.TargetPlatform) {
					ts := gluttertest.New(t)
					ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(c.builder), Routes: pushBRoutes()}))

					expectShown(t, ts, "start", map[string]bool{"push": true, "page b": false})
					expectCounts(t, ts, c, "start", 0, 1)

					ts.Tap(gluttertest.ByText("push"))
					ts.PumpAndSettle()

					expectShown(t, ts, "pushed", map[string]bool{"push": false, "page b": true})
					expectCounts(t, ts, c, "pushed", 0, 1)

					// Only Android supports backGesture channel methods. Other
					// platforms will do nothing.
					if platform != foundation.TargetPlatformAndroid {
						return
					}

					// Start a system pop gesture, which will switch to using the
					// gesture transition for the page transition.
					if !ts.StartBackGesture(0, backStart) {
						t.Fatal("the framework should take the gesture")
					}
					ts.Pump()

					expectCounts(t, ts, c, "gesture started", 1, 0)
					startPageBOffset := ts.TopLeft(gluttertest.ByText("page b"))
					if startPageBOffset.Dx != 0.0 {
						t.Errorf("page b starts at dx %v, want 0", startPageBOffset.Dx)
					}

					// Drag the system back gesture far enough to commit.
					ts.UpdateBackGestureProgress(0.35)
					ts.PumpAndSettle()

					expectCounts(t, ts, c, "gesture updated", 2, 0)
					if updatePageBOffset := ts.TopLeft(gluttertest.ByText("page b")); !(updatePageBOffset.Dx > startPageBOffset.Dx) {
						t.Errorf("page b moved to dx %v, want more than %v", updatePageBOffset.Dx, startPageBOffset.Dx)
					}

					// Commit the system back gesture.
					ts.CommitBackGesture()
					ts.PumpAndSettle()

					expectCounts(t, ts, c, "committed", 0, 1)
					expectShown(t, ts, "committed", map[string]bool{"push": true, "page b": false})
				})
			})

			t.Run("supports canceling a predictive back gesture", func(t *testing.T) {
				forAndroidAndLinux(t, func(t *testing.T, platform foundation.TargetPlatform) {
					ts := gluttertest.New(t)
					ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(c.builder), Routes: pushBRoutes()}))

					expectShown(t, ts, "start", map[string]bool{"push": true, "page b": false})
					expectCounts(t, ts, c, "start", 0, 1)

					ts.Tap(gluttertest.ByText("push"))
					ts.PumpAndSettle()

					expectShown(t, ts, "pushed", map[string]bool{"push": false, "page b": true})
					expectCounts(t, ts, c, "pushed", 0, 1)

					if platform != foundation.TargetPlatformAndroid {
						return
					}

					if !ts.StartBackGesture(0, backStart) {
						t.Fatal("the framework should take the gesture")
					}
					ts.Pump()

					expectCounts(t, ts, c, "gesture started", 1, 0)
					startPageBOffset := ts.TopLeft(gluttertest.ByText("page b"))
					if startPageBOffset.Dx != 0.0 {
						t.Errorf("page b starts at dx %v, want 0", startPageBOffset.Dx)
					}

					// Drag the system back gesture.
					ts.UpdateBackGestureProgress(0.35, backAt100)
					ts.PumpAndSettle()

					expectCounts(t, ts, c, "gesture updated", 2, 0)
					if updatePageBOffset := ts.TopLeft(gluttertest.ByText("page b")); !(updatePageBOffset.Dx > startPageBOffset.Dx) {
						t.Errorf("page b moved to dx %v, want more than %v", updatePageBOffset.Dx, startPageBOffset.Dx)
					}

					// Cancel the system back gesture.
					ts.CancelBackGesture()
					ts.PumpAndSettle()

					expectShown(t, ts, "canceled", map[string]bool{"push": false, "page b": true})
					expectCounts(t, ts, c, "canceled", 0, 1)
				})
			})

			t.Run("if there are multiple observers, only one gets called for a given back gesture", func(t *testing.T) {
				forAndroidAndLinux(t, func(t *testing.T, platform foundation.TargetPlatform) {
					includingNestedNavigator := false
					var setState widgets.StateSetter
					routes := map[string]widgets.WidgetBuilder{
						"/": func(bc widgets.BuildContext) widgets.Widget {
							return NewMaterial(NewTextButton(func() { widgets.NavigatorPushNamed(bc, "/b") }, widgets.NewText("push")))
						},
						"/b": func(widgets.BuildContext) widgets.Widget {
							return widgets.NewColumn([]widgets.Widget{
								widgets.NewText("page b"),
								widgets.NewStatefulBuilder(func(_ widgets.BuildContext, localSetState widgets.StateSetter) widgets.Widget {
									setState = localSetState
									if !includingNestedNavigator {
										return widgets.NewSizedBoxShrink(nil)
									}
									return widgets.NewNavigator(widgets.NavigatorOpts{
										InitialRoute: "b/nested",
										OnGenerateRoute: func(settings widgets.RouteSettings) widgets.Route {
											if settings.Name() != "b/nested" {
												t.Errorf("invalid route: %s", settings.Name())
												return nil
											}
											builder := func(widgets.BuildContext) widgets.Widget {
												return NewMaterial(NewTheme(allPlatformsTheme(c.builder),
													widgets.NewColumn([]widgets.Widget{widgets.NewText("Nested route inside of page b")})))
											}
											return NewMaterialPageRoute(builder, MaterialPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}})
										},
									})
								}),
							}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
						},
					}
					ts := gluttertest.New(t)
					ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(c.builder), Routes: routes}))

					expectShown(t, ts, "start", map[string]bool{"push": true, "page b": false, "Nested route inside of page b": false})
					expectCounts(t, ts, c, "start", 0, 1)

					ts.Tap(gluttertest.ByText("push"))
					ts.PumpAndSettle()

					expectShown(t, ts, "pushed", map[string]bool{"push": false, "page b": true, "Nested route inside of page b": false})
					expectCounts(t, ts, c, "pushed", 0, 1)

					if platform != foundation.TargetPlatformAndroid {
						return
					}

					if !ts.StartBackGesture(0, backStart) {
						t.Fatal("the framework should take the gesture")
					}
					ts.Pump()

					expectCounts(t, ts, c, "gesture started", 1, 0)
					startPageBOffset := ts.TopLeft(gluttertest.ByText("page b"))
					if startPageBOffset.Dx != 0.0 {
						t.Errorf("page b starts at dx %v, want 0", startPageBOffset.Dx)
					}

					// Drag the system back gesture.
					ts.UpdateBackGestureProgress(0.3, backAt100)
					ts.PumpAndSettle()

					expectCounts(t, ts, c, "gesture updated", 2, 0)
					updatePageBOffset := ts.TopLeft(gluttertest.ByText("page b"))
					if !(updatePageBOffset.Dx > startPageBOffset.Dx) {
						t.Errorf("page b moved to dx %v, want more than %v", updatePageBOffset.Dx, startPageBOffset.Dx)
					}

					// In the middle of the system back gesture here, add a nested
					// Navigator that includes a new predictive back gesture
					// observer.
					setState(func() { includingNestedNavigator = true })
					ts.PumpAndSettle()
					expectShown(t, ts, "nested added", map[string]bool{"push": true, "page b": true, "Nested route inside of page b": true})

					// Send another drag gesture, and ensure that the original
					// observer still gets it.
					ts.UpdateBackGestureProgress(0.35, gluttertest.BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 110, Dy: 300})})
					ts.PumpAndSettle()

					if got := ts.Count(transitionWidget(c.predictive)); got != 2 {
						t.Errorf("after the second update: %d %s, want 2", got, c.predictive)
					}
					// Despite using a PredictiveBackPageTransitions, the new route
					// has not received a start event, so it is still using the
					// fallback transition.
					if got := ts.Count(transitionWidget(c.fallback)); got != 1 {
						t.Errorf("after the second update: %d %s, want 1", got, c.fallback)
					}
					if update2PageBOffset := ts.TopLeft(gluttertest.ByText("page b")); !(update2PageBOffset.Dx > updatePageBOffset.Dx) {
						t.Errorf("page b moved to dx %v, want more than %v", update2PageBOffset.Dx, updatePageBOffset.Dx)
					}

					// Commit the system back gesture, and the original observer is
					// able to handle the back without interference.
					ts.CommitBackGesture()
					ts.PumpAndSettle()

					expectCounts(t, ts, c, "committed", 0, 1)
					expectShown(t, ts, "committed", map[string]bool{"push": true, "page b": false, "Nested route inside of page b": false})
				})
			})

			t.Run("two back gestures back to back dismiss two routes", func(t *testing.T) {
				forAndroidAndLinux(t, func(t *testing.T, platform foundation.TargetPlatform) {
					routes := map[string]widgets.WidgetBuilder{
						"/": func(bc widgets.BuildContext) widgets.Widget {
							return NewMaterial(NewTextButton(func() { widgets.NavigatorPushNamed(bc, "/b") }, widgets.NewText("push b")))
						},
						"/b": func(bc widgets.BuildContext) widgets.Widget {
							return NewMaterial(NewTextButton(func() { widgets.NavigatorPushNamed(bc, "/c") }, widgets.NewText("push c")))
						},
						"/c": func(widgets.BuildContext) widgets.Widget { return widgets.NewText("page c") },
					}
					ts := gluttertest.New(t)
					ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(c.builder), Routes: routes}))

					expectShown(t, ts, "start", map[string]bool{"push b": true, "push c": false, "page c": false})
					expectCounts(t, ts, c, "start", 0, 1)

					ts.Tap(gluttertest.ByText("push b"))
					ts.PumpAndSettle()

					expectShown(t, ts, "b pushed", map[string]bool{"push": false, "push c": true, "page c": false})
					expectCounts(t, ts, c, "b pushed", 0, 1)

					ts.Tap(gluttertest.ByText("push c"))
					ts.PumpAndSettle()

					expectShown(t, ts, "c pushed", map[string]bool{"push": false, "push c": false, "page c": true})
					expectCounts(t, ts, c, "c pushed", 0, 1)

					if platform != foundation.TargetPlatformAndroid {
						return
					}

					// Start a system pop gesture, which will switch to using the
					// gesture transition for the page transition.
					ts.StartBackGesture(0, backStart)
					ts.Pump()

					expectCounts(t, ts, c, "gesture started", 1, 0)

					// Drag the system back gesture far enough to commit.
					ts.UpdateBackGestureProgress(0.35)
					ts.Pump()

					expectCounts(t, ts, c, "gesture updated", 2, 0)

					// Commit the system back gesture.
					ts.CommitBackGesture()
					ts.Pump()

					// The predictive back page transitions still exist because the
					// outgoing animation has not yet finished.
					expectCounts(t, ts, c, "first committed", 2, 0)
					expectShown(t, ts, "first committed", map[string]bool{"push": false, "push c": true, "page c": true})

					// Start another system pop gesture, before the first has
					// finished animating out.
					ts.StartBackGesture(0, backStart)
					ts.Pump()

					expectCounts(t, ts, c, "second started", 2, 0)

					// Drag the system back gesture far enough to commit.
					ts.UpdateBackGestureProgress(0.35)
					ts.Pump()

					expectCounts(t, ts, c, "second updated", 3, 0)

					// Commit the system back gesture.
					ts.CommitBackGesture()
					ts.PumpAndSettle()

					expectCounts(t, ts, c, "second committed", 0, 1)
					expectShown(t, ts, "second committed", map[string]bool{"push b": true, "push c": false, "page c": false})
				})
			})

			t.Run("uses display corner radii when available", func(t *testing.T) {
				// flutter_test's default target platform is Android.
				withPlatform(t, foundation.TargetPlatformAndroid)
				displayCornerRadii := painting.NewBorderRadiusCircular(33.3)
				clipRRects := func(ts *gluttertest.Tester) int {
					return ts.Count(gluttertest.ByType[*widgets.ClipRRect]())
				}
				withRadii := func(ts *gluttertest.Tester) int {
					return ts.Count(gluttertest.ByPredicate("ClipRRect with the display corner radii", func(w widgets.Widget) bool {
						r, ok := w.(*widgets.ClipRRect)
						return ok && r.BorderRadius() == painting.BorderRadiusGeometry(displayCornerRadii)
					}))
				}

				ts := gluttertest.New(t)
				ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
					Theme:  allPlatformsTheme(c.builder),
					Routes: pushBRoutes(),
					Builder: func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
						return widgets.NewMediaQuery(widgets.MediaQueryOf(bc).ApplyDisplayCornerRadii(foundation.Some(displayCornerRadii)), child)
					},
				}))

				expectCounts(t, ts, c, "start", 0, 1)
				if n := clipRRects(ts); n != 0 {
					t.Errorf("start: %d ClipRRects, want 0", n)
				}

				ts.Tap(gluttertest.ByText("push"))
				ts.PumpAndSettle()

				expectCounts(t, ts, c, "pushed", 0, 1)
				if n := clipRRects(ts); n != 0 {
					t.Errorf("pushed: %d ClipRRects, want 0", n)
				}

				// Start a system pop gesture, which will switch to using the
				// gesture transition for the page transition.
				ts.StartBackGesture(0, backStart)
				ts.Pump()

				expectCounts(t, ts, c, "gesture started", 1, 0)
				if n := withRadii(ts); n != 1 {
					t.Errorf("gesture started: %d ClipRRects with the display corner radii, want 1", n)
				}
			})
		})
	}

	t.Run("uses the fallback radius of 32 when the display corner radii are unknown", func(t *testing.T) {
		for _, c := range predictiveBackBuilderCases() {
			t.Run(c.name, func(t *testing.T) {
				withPlatform(t, foundation.TargetPlatformAndroid)
				ts := gluttertest.New(t)
				ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(c.builder), Routes: pushBRoutes()}))
				ts.Tap(gluttertest.ByText("push"))
				ts.PumpAndSettle()
				ts.StartBackGesture(0, backStart)
				ts.Pump()
				// The shared element transition grows the radius with the gesture
				// from 0 to 32; the full screen transition has 32 all along.
				ts.UpdateBackGestureProgress(1, backAt100)
				ts.Pump()
				want := painting.BorderRadiusGeometry(painting.NewBorderRadiusCircular(32))
				if n := ts.Count(gluttertest.ByPredicate("ClipRRect of 32", func(w widgets.Widget) bool {
					r, ok := w.(*widgets.ClipRRect)
					return ok && r.BorderRadius() == want
				})); n == 0 {
					t.Error("no ClipRRect with the radius 32")
				}
			})
		}
	})

	t.Run("PredictiveBackPageTransitionsBuilder uses fallbackColor", func(t *testing.T) {
		forAndroidAndLinux(t, func(t *testing.T, _ foundation.TargetPlatform) {
			c := predictiveBackBuilderCases()[0]
			builder := c.withFallbck(Colors.Black)
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(builder), Routes: pushBRoutes()}))

			expectShown(t, ts, "start", map[string]bool{"push": true, "page b": false})
			expectCounts(t, ts, c, "start", 0, 1)

			ts.Tap(gluttertest.ByText("push"))
			ts.Pump()
			ts.Pump(400 * time.Millisecond)

			// Flutter finds the ColoredBox of the page below with .last, as the
			// overlay lists the route on top first; the elements here are visited
			// from the bottom, so the test looks for the color among all of them.
			if colors := boxColors(ts); !hasColor(colors, Colors.Black) {
				t.Errorf("the ColoredBoxes have the colors %v, want black among them", colors)
			}

			ts.PumpAndSettle()
		})
	})

	t.Run("PredictiveBackFullscreenPageTransitionsBuilder uses fallbackColor", func(t *testing.T) {
		forAndroidAndLinux(t, func(t *testing.T, _ foundation.TargetPlatform) {
			c := predictiveBackBuilderCases()[1]
			builder := c.withFallbck(Colors.Black)
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(builder), Routes: pushBRoutes()}))

			expectShown(t, ts, "start", map[string]bool{"push": true, "page b": false})
			expectCounts(t, ts, c, "start", 0, 1)

			// Pump till animation is half-way through.
			ts.Tap(gluttertest.ByText("push"))
			ts.Pump()
			ts.Pump(125 * time.Millisecond)

			// The scrim is painting the right color: exactly 59.6% opacity at
			// this time.
			if want := Colors.Black.WithOpacity(0.596); !hasColor(boxColors(ts), want) {
				t.Errorf("the scrim colors are %v, want one of %v", boxColors(ts), want)
			}

			ts.PumpAndSettle()
		})
	})
}

// TestPredictiveBackBuilderValues covers the parts of the builders that the
// Dart tests do not name: the durations, the fallback color and the equality of
// the value structs.
func TestPredictiveBackBuilderValues(t *testing.T) {
	tests := []struct {
		name             string
		builder          widgets.PageTransitionsBuilder
		duration, rev    time.Duration
		hasDelegatedPath bool
	}{
		{"predictive back: 450 ms", NewPredictiveBackPageTransitionsBuilder(), 450 * time.Millisecond, 450 * time.Millisecond, false},
		{"full screen: 300 ms", NewPredictiveBackFullscreenPageTransitionsBuilder(), 300 * time.Millisecond, 300 * time.Millisecond, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.builder.TransitionDuration(); got != tt.duration {
				t.Errorf("TransitionDuration = %v, want %v", got, tt.duration)
			}
			if got := tt.builder.ReverseTransitionDuration(); got != tt.rev {
				t.Errorf("ReverseTransitionDuration = %v, want %v", got, tt.rev)
			}
			if (tt.builder.DelegatedTransition() != nil) != tt.hasDelegatedPath {
				t.Errorf("DelegatedTransition is nil: %v", tt.builder.DelegatedTransition() == nil)
			}
		})
	}

	black := foundation.Some(Colors.Black)
	if got := NewPredictiveBackPageTransitionsBuilder(PredictiveBackPageTransitionsBuilderOpts{FallbackColor: black}).FallbackColor(); got != black {
		t.Errorf("FallbackColor = %v", got)
	}
	if got := NewPredictiveBackFullscreenPageTransitionsBuilder(PredictiveBackFullscreenPageTransitionsBuilderOpts{FallbackColor: black}).FallbackColor(); got != black {
		t.Errorf("FallbackColor = %v", got)
	}
	if !foundation.ValuesEqual(NewPredictiveBackPageTransitionsBuilder(), NewPredictiveBackPageTransitionsBuilder()) ||
		foundation.ValuesEqual(NewPredictiveBackPageTransitionsBuilder(), NewPredictiveBackPageTransitionsBuilder(PredictiveBackPageTransitionsBuilderOpts{FallbackColor: black})) {
		t.Error("builders with the same fallback color should be equal, and the others not")
	}
}

// TestPredictiveBackSharedElementMotion checks the motion of the shared element
// transition against the numbers of the motion spec in the Dart file: the page
// shrinks to 0.9 and moves sideways by a twentieth of the width minus 8 as the
// gesture goes on, follows the finger vertically (eased, and at most a twentieth
// of the height minus 8), and goes to the other side for a swipe from the right
// edge. The top left of the text of the page is where the scale (around the
// center of the page) and the translation put the point (0, 0).
func TestPredictiveBackSharedElementMotion(t *testing.T) {
	const width, height = 800.0, 600.0
	xShift := width/20 - 8
	yShiftMax := height/20 - 8
	at := func(dx, dy float64, edge ui.BackSwipeEdge) gluttertest.BackGestureOpts {
		return gluttertest.BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: dx, Dy: dy}), SwipeEdge: edge}
	}
	tests := []struct {
		name     string
		start    gluttertest.BackGestureOpts
		progress float64
		touch    gluttertest.BackGestureOpts
		scale    float64
		dx, dy   float64
	}{
		{"the start of the gesture: no movement", backStart, 0, backStart, 1, 0, 0},
		{"halfway from the left edge: smaller and to the right", backStart, 0.5, at(50, 300, ui.BackSwipeEdgeLeft), 0.95, xShift / 2, 0},
		{"all the way from the left edge: scale 0.9 and the full shift", backStart, 1, at(100, 300, ui.BackSwipeEdgeLeft), 0.9, xShift, 0},
		{"a finger below the start moves the page down, eased", backStart, 1, at(100, 600, ui.BackSwipeEdgeLeft), 0.9, xShift, yShiftMax * animation.Curves.EaseOut.Transform(0.5)},
		{"a finger far above the start is clamped", backStart, 1, at(100, -3000, ui.BackSwipeEdgeLeft), 0.9, xShift, -yShiftMax * animation.Curves.EaseOut.Transform(1)},
		{"from the right edge the page moves left", at(795, 300, ui.BackSwipeEdgeRight), 1, at(700, 300, ui.BackSwipeEdgeRight), 0.9, -xShift, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withPlatform(t, foundation.TargetPlatformAndroid)
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: allPlatformsTheme(NewPredictiveBackPageTransitionsBuilder()), Routes: pushBRoutes()}))
			ts.Tap(gluttertest.ByText("push"))
			ts.PumpAndSettle()
			if !ts.StartBackGesture(0, tt.start) {
				t.Fatal("the framework should take the gesture")
			}
			ts.Pump()
			ts.UpdateBackGestureProgress(tt.progress, tt.touch)
			ts.Pump()

			got := ts.TopLeft(gluttertest.ByText("page b"))
			wantX := width/2 + tt.scale*(tt.dx-width/2)
			wantY := height/2 + tt.scale*(tt.dy-height/2)
			if !closeTo(got.Dx, wantX) || !closeTo(got.Dy, wantY) {
				t.Errorf("page b is at %v, want (%v, %v)", got, wantX, wantY)
			}
		})
	}
}

func closeTo(a, b float64) bool {
	d := a - b
	return d < 1e-3 && d > -1e-3
}
