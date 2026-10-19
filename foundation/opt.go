package foundation

// Opt is an optional value: either a value (Some) or nothing (None, the zero
// value). It stands in for a nullable Flutter parameter whose null behaves
// differently from Go's zero value (SPEC §12.3).
//
// Flutter: no counterpart (Dart's nullable types)
type Opt[T any] struct {
	value T
	some  bool
}

// Some returns an Opt that holds v.
//
// Flutter: no counterpart (Dart's nullable types)
func Some[T any](v T) Opt[T] { return Opt[T]{value: v, some: true} }

// None returns an Opt that holds nothing, which is also the zero value.
//
// Flutter: no counterpart (Dart's null)
func None[T any]() Opt[T] { return Opt[T]{} }

// IsSome reports whether o holds a value.
func (o Opt[T]) IsSome() bool { return o.some }

// IsNone reports whether o holds nothing.
func (o Opt[T]) IsNone() bool { return !o.some }

// Get returns the value and whether there is one. Without a value it returns
// T's zero value and false.
func (o Opt[T]) Get() (T, bool) { return o.value, o.some }

// Value returns the value, or T's zero value when there is none.
func (o Opt[T]) Value() T { return o.value }

// OrElse returns the value, or def when there is none. It is Dart's `??`.
func (o Opt[T]) OrElse(def T) T {
	if o.some {
		return o.value
	}
	return def
}
