package layout

import "testing"

func TestLRUCache(t *testing.T) {
	type step struct {
		op   string // "put", "get"
		key  int
		val  string
		want string // for get: the value, or "" for a miss
	}
	tests := []struct {
		name  string
		size  int
		steps []step
		// keys are the keys left, from the most to the least recently used.
		keys []int
	}{
		{"fills up to its size", 3, []step{{"put", 1, "a", ""}, {"put", 2, "b", ""}, {"put", 3, "c", ""}}, []int{3, 2, 1}},
		{"drops the oldest when full, and does not empty the cache", 3, []step{
			{"put", 1, "a", ""}, {"put", 2, "b", ""}, {"put", 3, "c", ""}, {"put", 4, "d", ""},
			{"get", 1, "", ""}, {"get", 2, "", "b"}, {"get", 4, "", "d"},
		}, []int{4, 2, 3}},
		{"a hit protects an entry", 3, []step{
			{"put", 1, "a", ""}, {"put", 2, "b", ""}, {"put", 3, "c", ""},
			{"get", 1, "", "a"}, {"put", 4, "d", ""}, {"get", 2, "", ""}, {"get", 1, "", "a"},
		}, []int{1, 4, 3}},
		{"a put of a key it has replaces the value and protects it", 2, []step{
			{"put", 1, "a", ""}, {"put", 2, "b", ""}, {"put", 1, "A", ""}, {"put", 3, "c", ""},
			{"get", 1, "", "A"}, {"get", 2, "", ""},
		}, []int{1, 3}},
		{"a size of one", 1, []step{{"put", 1, "a", ""}, {"put", 2, "b", ""}, {"get", 1, "", ""}, {"get", 2, "", "b"}}, []int{2}},
		{"a size below one is one", 0, []step{{"put", 1, "a", ""}, {"put", 2, "b", ""}}, []int{2}},
		{"a miss changes nothing", 2, []step{{"put", 1, "a", ""}, {"get", 9, "", ""}}, []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newLRU[int, string](tt.size)
			for i, s := range tt.steps {
				switch s.op {
				case "put":
					c.put(s.key, s.val)
				case "get":
					got, ok := c.get(s.key)
					if got != s.want || ok != (s.want != "") {
						t.Errorf("step %d: get(%d) = %q, %v; want %q", i, s.key, got, ok, s.want)
					}
				}
			}
			var keys []int
			for n := c.head; n != nil; n = n.next {
				keys = append(keys, n.key)
			}
			if len(keys) != len(tt.keys) || c.len() != len(tt.keys) {
				t.Fatalf("keys from the most recent = %v (len %d), want %v", keys, c.len(), tt.keys)
			}
			for i := range keys {
				if keys[i] != tt.keys[i] {
					t.Errorf("keys = %v, want %v", keys, tt.keys)
					break
				}
			}
			// The links go both ways.
			var back []int
			for n := c.tail; n != nil; n = n.prev {
				back = append(back, n.key)
			}
			for i := range back {
				if back[i] != tt.keys[len(tt.keys)-1-i] {
					t.Errorf("keys from the least recent = %v, want the reverse of %v", back, tt.keys)
					break
				}
			}
		})
	}
}

func TestLRUCacheHoldsNilValues(t *testing.T) {
	// A glyph without an outline is cached as a nil path: a hit, not a miss.
	c := newLRU[int, *int](2)
	c.put(1, nil)
	if v, ok := c.get(1); !ok || v != nil {
		t.Errorf("get = %v, %v; want a hit with nil", v, ok)
	}
}
