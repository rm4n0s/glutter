package layout

// lruCache is a cache of at most max entries that, when it is full, drops the one that was
// used least recently. A get or a put makes an entry the most recently used. It is not safe for
// concurrent use: an Engine is for the UI thread only.
type lruCache[K comparable, V any] struct {
	max        int
	entries    map[K]*lruNode[K, V]
	head, tail *lruNode[K, V] // the most and the least recently used
}

// lruNode is an entry of a lruCache, linked from the most to the least recently used.
type lruNode[K comparable, V any] struct {
	key        K
	val        V
	prev, next *lruNode[K, V]
}

// newLRU makes a cache that holds at most size entries (at least 1).
func newLRU[K comparable, V any](size int) *lruCache[K, V] {
	return &lruCache[K, V]{max: max(size, 1), entries: map[K]*lruNode[K, V]{}}
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
	if n, ok := c.entries[key]; ok {
		n.val = val
		c.moveToFront(n)
		return
	}
	n := &lruNode[K, V]{key: key, val: val}
	c.entries[key] = n
	c.pushFront(n)
	if len(c.entries) > c.max {
		oldest := c.tail
		c.unlink(oldest)
		delete(c.entries, oldest.key)
	}
}

// len is the number of entries.
func (c *lruCache[K, V]) len() int { return len(c.entries) }

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
