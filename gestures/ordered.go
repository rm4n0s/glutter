package gestures

import "slices"

// orderedMap is a map from pointers that remembers the order of insertion, as
// Dart's maps do.
type orderedMap[V any] struct {
	keys []int
	m    map[int]V
}

func (o *orderedMap[V]) set(k int, v V) {
	if o.m == nil {
		o.m = map[int]V{}
	}
	if _, ok := o.m[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}

func (o *orderedMap[V]) remove(k int) {
	if _, ok := o.m[k]; !ok {
		return
	}
	delete(o.m, k)
	if i := slices.Index(o.keys, k); i >= 0 {
		o.keys = slices.Delete(o.keys, i, i+1)
	}
}

func (o *orderedMap[V]) len() int { return len(o.keys) }

func (o *orderedMap[V]) values() []V {
	vs := make([]V, len(o.keys))
	for i, k := range o.keys {
		vs[i] = o.m[k]
	}
	return vs
}
