package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestDirectionality(t *testing.T) {
	tr := newTree(t)
	var seen []ui.TextDirection
	var buildCount int
	child := builderOf(func(bc BuildContext) Widget {
		buildCount++
		seen = append(seen, DirectionalityOf(bc))
		return newLeaf()
	})
	tr.pump(NewDirectionality(ui.TextDirectionRTL, child))
	tr.pump(NewDirectionality(ui.TextDirectionRTL, child))
	if buildCount != 1 {
		t.Errorf("built %d times, want 1 when the direction did not change", buildCount)
	}
	tr.pump(NewDirectionality(ui.TextDirectionLTR, child))
	if buildCount != 2 || seen[0] != ui.TextDirectionRTL || seen[1] != ui.TextDirectionLTR {
		t.Errorf("built %d times, directions %v; want rtl then ltr", buildCount, seen)
	}
}

func TestDirectionalityMaybeOfAndMissing(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.collectPanics = true
	var found bool
	tr.pump(builderOf(func(bc BuildContext) Widget {
		_, found = MaybeDirectionalityOf(bc)
		DirectionalityOf(bc)
		return newLeaf()
	}))
	if found {
		t.Error("MaybeDirectionalityOf found a Directionality in a tree that has none")
	}
	_ = requireErrorTag(t, tr.errs[0], "DirectionalityMissing")
}

func TestDirectionalityAccessors(t *testing.T) {
	d := NewDirectionality(ui.TextDirectionRTL, newLeaf())
	if d.TextDirection() != ui.TextDirectionRTL {
		t.Errorf("TextDirection = %v", d.TextDirection())
	}
	tr := newTree(t)
	var ok bool
	var got ui.TextDirection
	tr.pump(NewDirectionality(ui.TextDirectionLTR, builderOf(func(bc BuildContext) Widget {
		got, ok = MaybeDirectionalityOf(bc)
		return newLeaf()
	})))
	if !ok || got != ui.TextDirectionLTR {
		t.Errorf("MaybeDirectionalityOf = %v, %v", got, ok)
	}
}
