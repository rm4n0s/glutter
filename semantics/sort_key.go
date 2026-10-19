package semantics

import (
	"math"
	"reflect"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// SemanticsSortKey orders the nodes of a parent for accessibility traversal.
// Keys can only be compared to keys of the same type. Embed
// SemanticsSortKeyBase for the name.
//
// Flutter: semantics/semantics.dart SemanticsSortKey
type SemanticsSortKey interface {
	foundation.Diagnosticable
	// Name is the name of the key; keys with different names are ordered by
	// name, a key without a name comes first.
	//
	// Flutter: semantics/semantics.dart SemanticsSortKey.name
	Name() foundation.Opt[string]
	// DoCompare compares to a key of the same type that has the same name.
	//
	// Flutter: semantics/semantics.dart SemanticsSortKey.doCompare
	DoCompare(other SemanticsSortKey) int
}

// SemanticsSortKeyBase holds the name of a sort key.
//
// Flutter: semantics/semantics.dart SemanticsSortKey
type SemanticsSortKeyBase struct{ name foundation.Opt[string] }

// NewSemanticsSortKeyBase creates the base with a name, which may be None.
//
// Flutter: semantics/semantics.dart SemanticsSortKey.new
func NewSemanticsSortKeyBase(name foundation.Opt[string]) SemanticsSortKeyBase {
	return SemanticsSortKeyBase{name}
}

// Name is the name of the key.
func (b SemanticsSortKeyBase) Name() foundation.Opt[string] { return b.name }

// DebugFillBase adds the name.
//
// Flutter: semantics/semantics.dart SemanticsSortKey.debugFillProperties
func (b SemanticsSortKeyBase) DebugFillBase(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("name", b.name, foundation.StringPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// SemanticsSortKeyCompare compares two keys of the same type. In debug builds
// a key of another type panics with the tag SemanticsSortKeyTypeMismatch.
//
// Flutter: semantics/semantics.dart SemanticsSortKey.compareTo
func SemanticsSortKeyCompare(a, b SemanticsSortKey) int {
	if foundation.DebugMode && reflect.TypeOf(a) != reflect.TypeOf(b) {
		panic(errors.New("SemanticsSortKeyTypeMismatch", "Semantics sort keys can only be compared to other sort keys of the same type.",
			"a", reflect.TypeOf(a).String(), "b", reflect.TypeOf(b).String()))
	}
	an, bn := a.Name(), b.Name()
	if optEqual(an, bn) {
		return a.DoCompare(b)
	}
	if an.IsNone() && bn.IsSome() {
		return -1
	}
	if an.IsSome() && bn.IsNone() {
		return 1
	}
	return strings.Compare(an.Value(), bn.Value())
}

// OrdinalSortKey is a sort key that orders by a number.
//
// Flutter: semantics/semantics.dart OrdinalSortKey
type OrdinalSortKey struct {
	SemanticsSortKeyBase
	order float64
}

// NewOrdinalSortKey creates a key. In debug builds an order that is infinite
// or NaN panics with the tag OrdinalSortKeyOrderInvalid.
//
// Flutter: semantics/semantics.dart OrdinalSortKey.new
func NewOrdinalSortKey(order float64, name ...string) *OrdinalSortKey {
	if foundation.DebugMode && (math.IsInf(order, 0) || math.IsNaN(order)) {
		panic(errors.New("OrdinalSortKeyOrderInvalid", "the order of an OrdinalSortKey must be finite", "order", order))
	}
	k := &OrdinalSortKey{order: order}
	if len(name) > 0 {
		k.name = foundation.Some(name[0])
	}
	return k
}

// Order is the number that the key orders by.
//
// Flutter: semantics/semantics.dart OrdinalSortKey.order
func (k *OrdinalSortKey) Order() float64 { return k.order }

// DoCompare compares the orders.
//
// Flutter: semantics/semantics.dart OrdinalSortKey.doCompare
func (k *OrdinalSortKey) DoCompare(other SemanticsSortKey) int {
	o := other.(*OrdinalSortKey)
	switch {
	case o.order == k.order:
		return 0
	case k.order < o.order:
		return -1
	}
	return 1
}

// DebugFillProperties adds the name and the order.
//
// Flutter: semantics/semantics.dart OrdinalSortKey.debugFillProperties
func (k *OrdinalSortKey) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	k.DebugFillBase(p)
	p.Add(foundation.NewDoubleProperty("order", foundation.Some(k.order), foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// String describes the key, such as "OrdinalSortKey#1a2b3(order: 1.0)".
//
// Flutter: semantics/semantics.dart OrdinalSortKey.toString
func (k *OrdinalSortKey) String() string { return foundation.DiagnosticableString(k) }
