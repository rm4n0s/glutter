package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// testFeature is an ink feature that paints nothing.
type testFeature struct{ InkFeatureBase }

func (*testFeature) PaintFeature(*ui.Canvas, *ui.Matrix4) {}

func newTestFeature(controller MaterialInkController, box rendering.Box) *testFeature {
	f := &testFeature{}
	InitInkFeature(&f.InkFeatureBase, f, controller, box)
	return f
}

// hasError reports whether one of the errors has the route.
func hasError(errs []error, route string) bool {
	for _, err := range errs {
		if e, ok := asAppError(err); ok && e.HasRoute(route) {
			return true
		}
	}
	return false
}

func TestMaterialColorMissing(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(materialApp(nil, NewMaterial(nil, MaterialOpts{Type: MaterialTypeButton})))
	if errs := ts.TakeErrors(); !hasError(errs, "MaterialColorMissing") {
		t.Errorf("errors = %v, want MaterialColorMissing", errs)
	}
	// A transparency material needs no color.
	ts.PumpWidget(materialApp(nil, NewMaterial(nil, MaterialOpts{Type: MaterialTypeTransparency})))
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func TestMaterialOfHiddenByLookupBoundary(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var err error
	var maybeFound bool
	ts.PumpWidget(materialApp(nil, NewMaterial(widgets.NewLookupBoundary(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		_, maybeFound = MaterialMaybeOf(bc)
		err = recoverError(func() { MaterialOf(bc) })
		return widgets.NewSizedBox(nil)
	})))))
	if maybeFound {
		t.Error("MaybeOf looked through the LookupBoundary")
	}
	wantRoute(t, err, "MaterialOf.MaterialHiddenByLookupBoundary")
}

func TestDebugCheckHasMaterialHiddenByLookupBoundary(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var err error
	ts.PumpWidget(materialApp(nil, NewMaterial(widgets.NewLookupBoundary(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		err = recoverError(func() { DebugCheckHasMaterial(bc) })
		return widgets.NewSizedBox(nil)
	})))))
	wantRoute(t, err, "DebugCheckHasMaterial.MaterialMissing")
}

func TestInkFeatureLifecycleErrors(t *testing.T) {
	skipUnlessDebug(t)
	setup := func(t *testing.T) (controller MaterialInkController, other MaterialInkController, box rendering.Box) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(materialApp(nil, widgets.NewColumn([]widgets.Widget{
			NewMaterial(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				controller = MaterialOf(bc)
				return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(10.0)})
			})),
			NewMaterial(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				other = MaterialOf(bc)
				return widgets.NewSizedBox(nil)
			})),
		})))
		box = ts.RenderObject(gluttertest.First(gluttertest.ByType[*widgets.SizedBox]())).(rendering.Box)
		return controller, other, box
	}
	tests := []struct {
		name  string
		run   func(controller, other MaterialInkController, box rendering.Box)
		route string
	}{
		{"added twice", func(c, _ MaterialInkController, b rendering.Box) {
			f := newTestFeature(c, b)
			c.AddInkFeature(f)
			c.AddInkFeature(f)
		}, "AddInkFeature.InkFeatureAddedTwice"},
		{"added to the wrong controller", func(c, o MaterialInkController, b rendering.Box) {
			o.AddInkFeature(newTestFeature(c, b))
		}, "AddInkFeature.InkFeatureWrongController"},
		{"added after being disposed", func(c, _ MaterialInkController, b rendering.Box) {
			f := newTestFeature(c, b)
			c.AddInkFeature(f)
			f.Dispose()
			c.AddInkFeature(f)
		}, "AddInkFeature.InkFeatureDisposed"},
		{"disposed twice", func(c, _ MaterialInkController, b rendering.Box) {
			f := newTestFeature(c, b)
			c.AddInkFeature(f)
			f.Dispose()
			f.Dispose()
		}, "Dispose.InkFeatureDisposed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, o, b := setup(t)
			wantRoute(t, recoverError(func() { tt.run(c, o, b) }), tt.route)
		})
	}
}

func TestInkFeaturePaintedDetached(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var controller MaterialInkController
	var box rendering.Box
	child := func(show bool) widgets.Widget {
		return materialApp(nil, NewMaterial(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			controller = MaterialOf(bc)
			if show {
				return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("gone"), Width: foundation.Some(10.0), Height: foundation.Some(10.0)})
			}
			return widgets.NewSizedBox(nil)
		})))
	}
	ts.PumpWidget(child(true))
	box = ts.RenderObject(gluttertest.ByKey(foundation.NewValueKey("gone"))).(rendering.Box)
	controller.AddInkFeature(newTestFeature(controller, box))
	ts.PumpWidget(child(false))
	controller.MarkNeedsPaint()
	err := recoverError(func() { ts.Pump() })
	if err == nil {
		err = firstOrNil(ts.TakeErrors())
	}
	wantRoute(t, err, "InkFeatureDetached")
}

func firstOrNil(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return errs[0]
}

func TestMaterialLocalizationsErrors(t *testing.T) {
	skipUnlessDebug(t)
	l := DefaultMaterialLocalizations{}
	tests := []struct {
		name  string
		run   func()
		route string
	}{
		{"tab index 0", func() { l.TabLabel(0, 3) }, "TabLabel.MaterialLocalizationsTabLabel"},
		{"tab count 0", func() { l.TabLabel(1, 0) }, "TabLabel.MaterialLocalizationsTabLabel"},
		{"hour out of range", func() { l.FormatHour(TimeOfDay{Hour: 100}, true) }, "formatTwoDigitZeroPad.MaterialLocalizationsNumberRange"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantRoute(t, recoverError(tt.run), tt.route)
		})
	}
}

func TestInkResponseNeedsASizeToSplashFromContext(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var err error
	// The layout builder runs while its render box, the one of the ink well, has no size yet.
	ts.PumpWidget(inkApp(nil, NewInkWell(widgets.NewLayoutBuilder(func(bc widgets.BuildContext, _ rendering.BoxConstraints) widgets.Widget {
		if err == nil {
			err = recoverError(func() { widgets.ActionsInvoke(bc, widgets.ActivateIntent{}) })
		}
		return widgets.NewSizedBox(nil)
	}), InkWellOpts{OnTap: func() {}})))
	wantRoute(t, err, "inkResponseState.startNewSplash.InkResponseNoSize")
}
