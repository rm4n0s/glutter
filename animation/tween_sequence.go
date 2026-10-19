package animation

import (
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// TweenSequenceItem is one animatable of a TweenSequence and its weight.
//
// Flutter: animation/tween_sequence.dart TweenSequenceItem
type TweenSequenceItem[T any] struct {
	// Tween is the animatable of the item.
	//
	// Flutter: animation/tween_sequence.dart TweenSequenceItem.tween
	Tween Animatable[T]
	// Weight is the item's share of the sequence; it must be greater than zero.
	//
	// Flutter: animation/tween_sequence.dart TweenSequenceItem.weight
	Weight float64
}

type tweenInterval struct{ start, end float64 }

func (i tweenInterval) contains(t float64) bool { return t >= i.start && t < i.end }
func (i tweenInterval) value(t float64) float64 { return (t - i.start) / (i.end - i.start) }

// TweenSequence lets a sequence of animatables run one after the other, each for
// a share of the time that follows its weight.
//
// Flutter: animation/tween_sequence.dart TweenSequence
type TweenSequence[T any] struct {
	AnimatableBase[T]
	items     []TweenSequenceItem[T]
	intervals []tweenInterval
}

// NewTweenSequence creates a sequence of the items. It panics with the tag
// TweenSequenceInvalid when there are no items or a weight is not greater than
// zero.
//
// Flutter: animation/tween_sequence.dart TweenSequence.new
func NewTweenSequence[T any](items []TweenSequenceItem[T]) *TweenSequence[T] {
	return newTweenSequence(items)
}

func newTweenSequence[T any](items []TweenSequenceItem[T]) *TweenSequence[T] {
	if len(items) == 0 {
		panic(errors.New("TweenSequenceInvalid", "a TweenSequence needs at least one item"))
	}
	s := &TweenSequence[T]{items: append([]TweenSequenceItem[T](nil), items...)}
	total := 0.0
	for i, item := range s.items {
		if !(item.Weight > 0) {
			panic(errors.New("TweenSequenceInvalid", "the weight of a TweenSequenceItem must be greater than zero", "index", i, "weight", item.Weight))
		}
		total += item.Weight
	}
	start := 0.0
	for i, item := range s.items {
		end := 1.0
		if i != len(s.items)-1 {
			end = start + item.Weight/total
		}
		s.intervals = append(s.intervals, tweenInterval{start, end})
		start = end
	}
	InitAnimatableBase(&s.AnimatableBase, Animatable[T](s))
	return s
}

func (s *TweenSequence[T]) evaluateAt(t float64, index int) T {
	return s.items[index].Tween.Transform(s.intervals[index].value(t))
}

// Transform returns the value of the item that t falls in.
//
// Flutter: animation/tween_sequence.dart TweenSequence.transform
func (s *TweenSequence[T]) Transform(t float64) T {
	if foundation.DebugMode && (t < 0 || t > 1) {
		panic(errors.New("TweenSequenceOutOfRange", fmt.Sprintf("t %v is outside of the range [0, 1]", t), "t", t))
	}
	if t == 1 {
		return s.evaluateAt(t, len(s.items)-1)
	}
	for i := range s.items {
		if s.intervals[i].contains(t) {
			return s.evaluateAt(t, i)
		}
	}
	if foundation.DebugMode {
		panic(errors.New("TweenSequenceNoInterval", fmt.Sprintf("TweenSequence.Transform could not find an interval for %v", t), "t", t))
	}
	return s.evaluateAt(t, len(s.items)-1)
}

// String describes the sequence.
//
// Flutter: animation/tween_sequence.dart TweenSequence.toString
func (s *TweenSequence[T]) String() string {
	return fmt.Sprintf("TweenSequence(%d items)", len(s.items))
}

// FlippedTweenSequence is a TweenSequence of float64 that runs backwards: it
// has 1 - sequence(1 - t).
//
// Flutter: animation/tween_sequence.dart FlippedTweenSequence
type FlippedTweenSequence struct{ *TweenSequence[float64] }

// NewFlippedTweenSequence creates the flipped sequence of the items.
//
// Flutter: animation/tween_sequence.dart FlippedTweenSequence.new
func NewFlippedTweenSequence(items []TweenSequenceItem[float64]) *FlippedTweenSequence {
	f := &FlippedTweenSequence{newTweenSequence(items)}
	InitAnimatableBase(&f.AnimatableBase, Animatable[float64](f))
	return f
}

// Transform returns 1 - the sequence at 1 - t.
//
// Flutter: animation/tween_sequence.dart FlippedTweenSequence.transform
func (f *FlippedTweenSequence) Transform(t float64) float64 {
	return 1 - f.TweenSequence.Transform(1-t)
}
