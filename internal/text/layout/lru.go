package layout

// lruCache is a cache of at most max entries that, when it is full, drops the one that was
// used least recently. A get or a put makes an entry the most recently used. It is not safe for
// concurrent use: an Engine is for the UI thread only.
type lruCache[K comparable, V any] struct {
	max        int
	entries    map[K]*lruNode[K, V]
	head, tail *lruNode[K, V] // the most and the least recently used

	// A cache made with newWeightedLRU is also bounded by the total weight of its entries.
	maxWeight int
	weight    int
	weigh     func(K, V) int
}

// lruNode is an entry of a lruCache, linked from the most to the least recently used.
type lruNode[K comparable, V any] struct {
	key        K
	val        V
	weight     int
	prev, next *lruNode[K, V]
}

// newLRU makes a cache that holds at most size entries (at least 1).
func newLRU[K comparable, V any](size int) *lruCache[K, V] {
	return &lruCache[K, V]{max: max(size, 1), entries: map[K]*lruNode[K, V]{}}
}

// newWeightedLRU makes a cache that holds at most size entries and at most maxWeight in all, the
// sum of what weigh says each entry weighs (their size in bytes, for instance). An entry that
// weighs more than maxWeight by itself is not stored.
func newWeightedLRU[K comparable, V any](size, maxWeight int, weigh func(K, V) int) *lruCache[K, V] {
	c := newLRU[K, V](size)
	c.maxWeight, c.weigh = max(maxWeight, 1), weigh
	return c
}

// get returns the value of key and makes it the most recently used.
func (c *lruCache[K, V]) get(key K) (V, bool) {
	n, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	c.moveToFront(n)
	return n.val, true
}

// put stores the value of key as the most recently used, and drops the least recently used
// entry when the cache is over its size.
func (c *lruCache[K, V]) put(key K, val V) {
	w := 0
	if c.weigh != nil {
		w = c.weigh(key, val)
		if w > c.maxWeight {
			c.remove(key)
			return
		}
	}
	if n, ok := c.entries[key]; ok {
		c.weight += w - n.weight
		n.val, n.weight = val, w
		c.moveToFront(n)
	} else {
		n := &lruNode[K, V]{key: key, val: val, weight: w}
		c.entries[key] = n
		c.pushFront(n)
		c.weight += w
	}
	for len(c.entries) > c.max || (c.maxWeight > 0 && c.weight > c.maxWeight) {
		c.evict(c.tail)
	}
}

// remove drops the entry of key, if there is one.
func (c *lruCache[K, V]) remove(key K) {
	if n, ok := c.entries[key]; ok {
		c.evict(n)
	}
}

func (c *lruCache[K, V]) evict(n *lruNode[K, V]) {
	c.unlink(n)
	delete(c.entries, n.key)
	c.weight -= n.weight
}

// clear drops every entry.
func (c *lruCache[K, V]) clear() {
	clear(c.entries)
	c.head, c.tail, c.weight = nil, nil, 0
}

// len is the number of entries.
func (c *lruCache[K, V]) len() int { return len(c.entries) }

// lruGetBytes is get for a key that is in a byte buffer: the conversion in the index
// expression does not allocate.
func lruGetBytes[V any](c *lruCache[string, V], key []byte) (V, bool) {
	n, ok := c.entries[string(key)]
	if !ok {
		var zero V
		return zero, false
	}
	c.moveToFront(n)
	return n.val, true
}

func (c *lruCache[K, V]) moveToFront(n *lruNode[K, V]) {
	if c.head == n {
		return
	}
	c.unlink(n)
	c.pushFront(n)
}

func (c *lruCache[K, V]) pushFront(n *lruNode[K, V]) {
	n.prev, n.next = nil, c.head
	if c.head != nil {
		c.head.prev = n
	}
	c.head = n
	if c.tail == nil {
		c.tail = n
	}
}

func (c *lruCache[K, V]) unlink(n *lruNode[K, V]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		c.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		c.tail = n.prev
	}
	n.prev, n.next = nil, nil
}
