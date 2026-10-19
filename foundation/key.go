package foundation

import (
	"reflect"

	"github.com/rm4n0s/errors"
)

// Key is an identifier for widgets, elements and semantic nodes. A widget with
// a key replaces a widget of the same type with an equal key instead of
// updating the one at its position.
//
// Two keys are the same key when they are equal under Go's == operator, so
// keys work as map keys. A key type of your own embeds KeyBase (or
// LocalKeyBase) and must be comparable.
//
// Flutter: foundation/key.dart Key
type Key interface {
	// String describes the key for debug output, such as "[<'title'>]".
	String() string
	keyMarker()
}

// KeyBase marks a type as a Key. Embed it in a key type.
//
// Flutter: foundation/key.dart Key.empty
type KeyBase struct{}

func (KeyBase) keyMarker() {}

// LocalKey is a Key that is not a GlobalKey: it is unique only among the
// children of one parent.
//
// Flutter: foundation/key.dart LocalKey
type LocalKey interface {
	Key
	localKeyMarker()
}

// LocalKeyBase marks a type as a LocalKey. Embed it in a local key type.
//
// Flutter: foundation/key.dart LocalKey
type LocalKeyBase struct{ KeyBase }

func (LocalKeyBase) localKeyMarker() {}

// ValueKey is a key that uses a value of a given type to identify itself. Two
// ValueKeys are equal when their values are equal.
//
// Deviation: it is a plain value, not a pointer, so that == and map lookups
// compare the values (SPEC §12.10).
//
// Flutter: foundation/key.dart ValueKey
type ValueKey[T comparable] struct {
	LocalKeyBase
	value T
}

// NewValueKey creates a key from a value.
//
// Flutter: foundation/key.dart ValueKey.new
func NewValueKey[T comparable](value T) ValueKey[T] { return ValueKey[T]{value: value} }

// NewKey creates a key from a string. It is Flutter's `Key('value')`, a
// ValueKey of a string.
//
// Flutter: foundation/key.dart Key.new
func NewKey(value string) ValueKey[string] { return NewValueKey(value) }

// Value is the value the key was made from.
//
// Flutter: foundation/key.dart ValueKey.value
func (k ValueKey[T]) Value() T { return k.value }

// String describes the key, such as "[<'title'>]" for a string and "[<3>]"
// for other values.
//
// Flutter: foundation/key.dart ValueKey.toString
func (k ValueKey[T]) String() string {
	if s, ok := any(k.value).(string); ok {
		return "[<'" + s + "'>]"
	}
	return "[<" + describeValue(k.value) + ">]"
}

// ObjectKey is a key that takes its identity from the object it was made
// from: two ObjectKeys are equal when they hold the same object (the same
// pointer, or equal values of a type that is not a pointer).
//
// Flutter: widgets/framework.dart ObjectKey
type ObjectKey struct {
	LocalKeyBase
	value any
}

// NewObjectKey creates a key from an object. The object's type must be
// comparable (not a slice, map or function), or it panics with the tag
// ObjectKeyNotComparable.
//
// Flutter: widgets/framework.dart ObjectKey.new
func NewObjectKey(value any) ObjectKey {
	if value != nil && !reflect.TypeOf(value).Comparable() {
		panic(errors.New("ObjectKeyNotComparable", "the value of an ObjectKey must be comparable",
			"type", RuntimeTypeName(value)))
	}
	return ObjectKey{value: value}
}

// Value is the object the key was made from.
//
// Flutter: widgets/framework.dart ObjectKey.value
func (k ObjectKey) Value() any { return k.value }

// String describes the key, such as "[Foo#1a2b3]".
//
// Flutter: widgets/framework.dart ObjectKey.toString
func (k ObjectKey) String() string { return "[" + DescribeIdentity(k.value) + "]" }

// UniqueKey is a key that is equal only to itself.
//
// Flutter: foundation/key.dart UniqueKey
type UniqueKey struct {
	LocalKeyBase
	// id makes the struct non-empty, so that every UniqueKey is a different
	// allocation with its own address.
	id byte
}

// NewUniqueKey creates a key that is equal only to itself. Compare keys by
// pointer; it is a *UniqueKey.
//
// Flutter: foundation/key.dart UniqueKey.new
func NewUniqueKey() *UniqueKey { return &UniqueKey{id: 1} }

// String describes the key, such as "[#1a2b3]".
//
// Flutter: foundation/key.dart UniqueKey.toString
func (k *UniqueKey) String() string { return "[#" + ShortHash(k) + "]" }
