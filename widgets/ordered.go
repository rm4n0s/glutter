package widgets

// orderedSet is a set that keeps the order of insertion, so that the messages
// and the processing order of the framework do not depend on map iteration.
type orderedSet[T comparable] struct {
	items []T
	set   map[T]struct{}
}

func (s *orderedSet[T]) add(v T) bool {
	if _, ok := s.set[v]; ok {
		return false
	}
	if s.set == nil {
		s.set = map[T]struct{}{}
	}
	s.set[v] = struct{}{}
	s.items = append(s.items, v)
	return true
}

func (s *orderedSet[T]) has(v T) bool {
	_, ok := s.set[v]
	return ok
}

func (s *orderedSet[T]) len() int { return len(s.items) }

func (s *orderedSet[T]) clear() {
	s.items = nil
	s.set = nil
}

// orderedMap is a map that keeps the order in which keys were first put.
type orderedMap[K comparable, V any] struct {
	keys []K
	m    map[K]V
}

func (m *orderedMap[K, V]) get(k K) (V, bool) {
	v, ok := m.m[k]
	return v, ok
}

func (m *orderedMap[K, V]) put(k K, v V) {
	if m.m == nil {
		m.m = map[K]V{}
	}
	if _, ok := m.m[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.m[k] = v
}

func (m *orderedMap[K, V]) remove(k K) {
	if _, ok := m.m[k]; !ok {
		return
	}
	delete(m.m, k)
	for i, x := range m.keys {
		if x == k {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}

func (m *orderedMap[K, V]) len() int { return len(m.keys) }

func (m *orderedMap[K, V]) clear() {
	m.keys = nil
	m.m = nil
}
