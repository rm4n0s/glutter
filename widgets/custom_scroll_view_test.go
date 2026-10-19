package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/custom_scroll_view_test.dart.

func keyedBox(key string) *SliverToBoxAdapter {
	return NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(100.0)}), SliverToBoxAdapterOpts{Key: foundation.NewValueKey(key)})
}

func boxRect(ts *gluttertest.Tester, key string, offstage bool) (topLeft, bottomRight ui.Offset) {
	f := gluttertest.Descendant(gluttertest.ByKey(foundation.NewValueKey(key), gluttertest.FindOpts{IncludeOffstage: offstage}),
		gluttertest.ByType[*SizedBox](gluttertest.FindOpts{IncludeOffstage: offstage}))
	return ts.TopLeft(f), ts.BottomRight(f)
}

func TestCustomScrollViewCenterUpdate1(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/96024.
	ts := gluttertest.New(t, gluttertest.Opts{})
	centerKey := foundation.NewUniqueKey()
	var setState StateSetter
	hasKey := false
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{
		keyedBox("a"),
		NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			if hasKey {
				return keyedBox("b")
			}
			return NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(100.0)}))
		}, WidgetOpts{Key: centerKey}),
	}, CustomScrollViewOpts{Center: centerKey})))
	ts.PumpAndSettle()
	// Changing the center key removes the old render object and inserts a new one.
	setState(func() { hasKey = true })
	ts.PumpAndSettle()
}

func TestCustomScrollViewCenterUpdate2(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	frame := func(center string, keys ...string) Widget {
		slivers := make([]Widget, len(keys))
		for i, k := range keys {
			slivers[i] = keyedBox(k)
		}
		return ltr(NewCustomScrollView(slivers, CustomScrollViewOpts{Center: foundation.NewValueKey(center)}))
	}
	ts.PumpWidget(frame("b", "a", "b", "c"))
	ts.PumpAndSettle()
	ts.PumpWidget(frame("d", "c", "d", "a"))
	ts.PumpAndSettle()
}

func TestCustomScrollViewCenter(t *testing.T) {
	for _, c := range []struct {
		name, center                 string
		aTop, aBottom, bTop, bBottom float64
		offstage                     bool
	}{
		{"a", "a", 0, 100, 100, 200, false},
		{"b", "b", -100, 0, 0, 100, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(ltr(NewCustomScrollView([]Widget{keyedBox("a"), keyedBox("b")}, CustomScrollViewOpts{Center: foundation.NewValueKey(c.center)})))
			ts.PumpAndSettle()
			tl, br := boxRect(ts, "a", c.offstage)
			if tl != (ui.Offset{Dy: c.aTop}) || br != (ui.Offset{Dx: 800, Dy: c.aBottom}) {
				t.Errorf("a = %v..%v", tl, br)
			}
			tl, br = boxRect(ts, "b", c.offstage)
			if tl != (ui.Offset{Dy: c.bTop}) || br != (ui.Offset{Dx: 800, Dy: c.bBottom}) {
				t.Errorf("b = %v..%v", tl, br)
			}
		})
	}
}

func TestCustomScrollViewAnchor(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{keyedBox("a"), keyedBox("b")}, CustomScrollViewOpts{Center: foundation.NewValueKey("b"), Anchor: 1})))
	ts.PumpAndSettle()
	tl, br := boxRect(ts, "a", false)
	if tl != (ui.Offset{Dy: 500}) || br != (ui.Offset{Dx: 800, Dy: 600}) {
		t.Errorf("a = %v..%v", tl, br)
	}
	tl, br = boxRect(ts, "b", true)
	if tl != (ui.Offset{Dy: 600}) || br != (ui.Offset{Dx: 800, Dy: 700}) {
		t.Errorf("b = %v..%v", tl, br)
	}
}
