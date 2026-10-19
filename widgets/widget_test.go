package widgets

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

type otherLeaf struct{ WidgetBase }

func (w *otherLeaf) CreateElement() Element {
	e := &leafElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

func TestCanUpdate(t *testing.T) {
	u1, u2 := foundation.NewUniqueKey(), foundation.NewUniqueKey()
	tests := []struct {
		name     string
		old, new Widget
		want     bool
	}{
		{"same type, no keys", newLeaf(), newLeaf(), true},
		{"different types", newLeaf(), &otherLeaf{}, false},
		{"equal value keys", newLeaf(WidgetOpts{Key: foundation.NewKey("a")}), newLeaf(WidgetOpts{Key: foundation.NewKey("a")}), true},
		{"different value keys", newLeaf(WidgetOpts{Key: foundation.NewKey("a")}), newLeaf(WidgetOpts{Key: foundation.NewKey("b")}), false},
		{"key against no key", newLeaf(WidgetOpts{Key: foundation.NewKey("a")}), newLeaf(), false},
		{"same unique key", newLeaf(WidgetOpts{Key: u1}), newLeaf(WidgetOpts{Key: u1}), true},
		{"different unique keys", newLeaf(WidgetOpts{Key: u1}), newLeaf(WidgetOpts{Key: u2}), false},
		{"key types differ", newLeaf(WidgetOpts{Key: foundation.NewValueKey(3)}), newLeaf(WidgetOpts{Key: foundation.NewValueKey("3")}), false},
		{"stateless against stateful", newWrapper(newLeaf()), newBuilder(nil), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanUpdate(tc.old, tc.new); got != tc.want {
				t.Errorf("CanUpdate = %v, want %v", got, tc.want)
			}
		})
	}
}

// badWidget has a base but implements none of the element interfaces.
type badWidget struct{ WidgetBase }

// statelessLookalike has Build but not the StatelessWidget marker.
type statelessLookalike struct{ WidgetBase }

func (*statelessLookalike) Build(BuildContext) Widget { return nil }

// customElementWidget is a stateless widget that also makes its own element.
type customElementWidget struct {
	StatelessWidget
	created *bool
}

func (w *customElementWidget) Build(BuildContext) Widget { return newLeaf() }

func (w *customElementWidget) CreateElement() Element {
	*w.created = true
	e := &leafElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

func TestCreateElementKind(t *testing.T) {
	created := false
	tests := []struct {
		name   string
		widget Widget
		check  func(Element) bool
	}{
		{"stateless", newWrapper(newLeaf()), func(e Element) bool { _, ok := e.(*StatelessElement); return ok }},
		{"stateful", newBuilder(nil), func(e Element) bool { _, ok := e.(*StatefulElement); return ok }},
		{"own element wins", &customElementWidget{NewStatelessWidgetBase(), &created}, func(e Element) bool {
			_, ok := e.(*leafElement)
			return ok && created
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if e := CreateElement(tc.widget); !tc.check(e) {
				t.Errorf("CreateElement gave %T", e)
			}
		})
	}
}

func TestCreateElementUnknownKind(t *testing.T) {
	for name, w := range map[string]Widget{
		"no element interface": &badWidget{},
		"Build without marker": &statelessLookalike{},
	} {
		t.Run(name, func(t *testing.T) {
			requireTag(t, "CreateElement.WidgetKindUnknown", func() { CreateElement(w) })
		})
	}
}

func TestWidgetToStringShort(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name   string
		widget Widget
		want   string
	}{
		{"no key", newLeaf(), "leaf#00000"},
		{"value key", newLeaf(WidgetOpts{Key: foundation.NewKey("k")}), "leaf#00000-[<'k'>]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeHashes(WidgetToStringShort(tc.widget)); got != tc.want {
				t.Errorf("WidgetToStringShort = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDescribeWidget(t *testing.T) {
	skipUnlessDebug(t)
	got := normalizeHashes(foundation.ToStringDeep(&widgetDescriptionTree{newLeaf(WidgetOpts{Key: foundation.NewKey("k")})}))
	if !strings.HasPrefix(got, "leaf#00000-[<'k'>]") {
		t.Errorf("description = %q", got)
	}
	if n := DescribeWidget(newLeaf()); n == nil {
		t.Error("DescribeWidget returned nil")
	}
}
