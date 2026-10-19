package widgets_test

import (
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestDirectionality(t *testing.T) {
	ts := gluttertest.New(t)
	var seen []ui.TextDirection
	var buildCount int
	child := builderOf(func(bc BuildContext) Widget {
		buildCount++
		seen = append(seen, DirectionalityOf(bc))
		return newLeaf()
	})
	ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, child))
	ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, child))
	if buildCount != 1 {
		t.Errorf("built %d times, want 1 when the direction did not change", buildCount)
	}
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, child))
	if buildCount != 2 || seen[0] != ui.TextDirectionRTL || seen[1] != ui.TextDirectionLTR {
		t.Errorf("built %d times, directions %v; want rtl then ltr", buildCount, seen)
	}
}

func TestDirectionalityMaybeOfAndMissing(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	var found bool
	ts.PumpWidget(builderOf(func(bc BuildContext) Widget {
		_, found = MaybeDirectionalityOf(bc)
		DirectionalityOf(bc)
		return newLeaf()
	}))
	if found {
		t.Error("MaybeDirectionalityOf found a Directionality in a tree that has none")
	}
	_ = requireErrorTag(t, takeFirst(t, ts), "DirectionalityMissing")
}

func TestDirectionalityAccessors(t *testing.T) {
	d := NewDirectionality(ui.TextDirectionRTL, newLeaf())
	if d.TextDirection() != ui.TextDirectionRTL {
		t.Errorf("TextDirection = %v", d.TextDirection())
	}
	ts := gluttertest.New(t)
	var ok bool
	var got ui.TextDirection
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, builderOf(func(bc BuildContext) Widget {
		got, ok = MaybeDirectionalityOf(bc)
		return newLeaf()
	})))
	if !ok || got != ui.TextDirectionLTR {
		t.Errorf("MaybeDirectionalityOf = %v, %v", got, ok)
	}
}
