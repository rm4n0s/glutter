package svg

import (
	"container/list"
	"context"
	"sync"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/widgets"
)

// DefaultCacheMaximumSizeBytes is the limit of a Cache that was given none: 16 MiB, about a thousand icons.
//
// flutter_svg: cache.dart Cache.maximumSize
const DefaultCacheMaximumSizeBytes = 16 << 20

// CacheOpts are the optional parameters of NewCache.
//
// flutter_svg: cache.dart Cache.new
type CacheOpts struct {
	// MaximumSizeBytes is the most memory that the documents in the cache may hold, as Document.ApproximateBytes
	// estimates it; the default is DefaultCacheMaximumSizeBytes. Zero means that nothing is kept.
	MaximumSizeBytes foundation.Opt[int]
}

// Cache keeps the parsed documents of the SvgPicture widgets, by source and theme, so that the same icon in a list is
// parsed once. It is a least recently used cache with a limit in bytes. It is safe to use from several goroutines.
//
// There is no global cache. An app has one for each binding (CacheOf), or gives its own to a subtree with
// NewSvgCacheScope.
//
// Deviation: flutter_svg's cache is the global svg.cache and counts entries (100 by default); this one belongs to the
// binding and counts bytes, as SPEC §26.1 says (SPEC §12.10).
//
// flutter_svg: cache.dart Cache
type Cache struct {
	mu      sync.Mutex
	max     int
	size    int
	lru     *list.List // of *cacheItem, the most recently used first
	items   map[cacheKey]*list.Element
	pending map[cacheKey]*flight
}

// cacheKey is the key of an entry: the source (a comparable value that says which bytes) and the theme the document
// was parsed with.
type cacheKey struct {
	source any
	theme  SvgTheme
}

type cacheItem struct {
	key   cacheKey
	entry *entry
}

// flight is a load that is going on: the widgets that want the same document wait for it.
type flight struct {
	done     chan struct{}
	entry    *entry
	err      error
	canceled bool
}

// NewCache creates an empty cache.
//
// flutter_svg: cache.dart Cache.new
func NewCache(o ...CacheOpts) *Cache {
	co := opts.First(o)
	return &Cache{
		max:     co.MaximumSizeBytes.OrElse(DefaultCacheMaximumSizeBytes),
		lru:     list.New(),
		items:   map[cacheKey]*list.Element{},
		pending: map[cacheKey]*flight{},
	}
}

// MaximumSizeBytes is the most memory that the cache keeps.
//
// flutter_svg: cache.dart Cache.maximumSize
func (c *Cache) MaximumSizeBytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.max
}

// SetMaximumSizeBytes changes the limit. If the documents in the cache are larger than the new limit, the least
// recently used ones are evicted at once; zero clears the cache.
//
// flutter_svg: cache.dart Cache.maximumSize
func (c *Cache) SetMaximumSizeBytes(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.max = max(n, 0)
	c.trim()
}

// Count is the number of documents in the cache.
//
// flutter_svg: cache.dart Cache.count
func (c *Cache) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}

// SizeBytes is the memory that the documents in the cache hold, as Document.ApproximateBytes estimates it.
//
// flutter_svg: no counterpart
func (c *Cache) SizeBytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}

// Clear evicts every document. The widgets that show one keep it; they load it again when they are rebuilt with
// another source or theme.
//
// flutter_svg: cache.dart Cache.clear
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Init()
	c.items = map[cacheKey]*list.Element{}
	c.size = 0
}

// get returns the entry of the key and makes it the most recently used.
func (c *Cache) get(key cacheKey) (*entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.lru.MoveToFront(el)
		return el.Value.(*cacheItem).entry, true
	}
	return nil, false
}

// put adds an entry (replacing the one with the same key) and evicts what no longer fits. An entry larger than the
// limit is not kept.
func (c *Cache) put(key cacheKey, e *entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.putLocked(key, e)
}

func (c *Cache) putLocked(key cacheKey, e *entry) {
	if el, ok := c.items[key]; ok {
		c.size -= el.Value.(*cacheItem).entry.bytes
		c.lru.Remove(el)
		delete(c.items, key)
	}
	if e.bytes > c.max {
		return
	}
	c.items[key] = c.lru.PushFront(&cacheItem{key: key, entry: e})
	c.size += e.bytes
	c.trim()
}

// trim evicts the least recently used entries until the cache fits its limit.
func (c *Cache) trim() {
	for c.size > c.max {
		el := c.lru.Back()
		if el == nil {
			c.size = 0
			return
		}
		item := el.Value.(*cacheItem)
		c.lru.Remove(el)
		delete(c.items, item.key)
		c.size -= item.entry.bytes
	}
}

// load returns the entry of the key: the cached one, the one that another caller is loading (it waits for it), or
// the one that fn returns, which is then cached. fn runs on the calling goroutine, without the lock. It returns
// (nil, nil) when ctx ended before there was an entry. If the caller that was loading the entry ended first, a waiter
// takes the load over.
func (c *Cache) load(ctx context.Context, key cacheKey, fn func(ctx context.Context) (*entry, error)) (*entry, error) {
	for {
		c.mu.Lock()
		if el, ok := c.items[key]; ok {
			c.lru.MoveToFront(el)
			e := el.Value.(*cacheItem).entry
			c.mu.Unlock()
			return e, nil
		}
		if f, ok := c.pending[key]; ok {
			c.mu.Unlock()
			select {
			case <-f.done:
			case <-ctx.Done():
				return nil, nil
			}
			if f.canceled {
				continue
			}
			return f.entry, f.err
		}
		f := &flight{done: make(chan struct{})}
		c.pending[key] = f
		c.mu.Unlock()

		e, err := fn(ctx)

		c.mu.Lock()
		delete(c.pending, key)
		switch {
		case ctx.Err() != nil:
			f.canceled = true
		case err != nil:
			f.err = err
		default:
			f.entry = e
			c.putLocked(key, e)
		}
		c.mu.Unlock()
		close(f.done)
		switch {
		case f.canceled:
			return nil, nil
		case f.err != nil:
			return nil, f.err
		}
		return e, nil
	}
}

// SvgCacheScope gives the SvgPicture widgets below it a Cache of their own, in place of the cache of the binding.
//
// flutter_svg: no counterpart
type SvgCacheScope struct {
	widgets.InheritedWidget
	cache *Cache
}

// NewSvgCacheScope creates a scope whose descendants cache their documents in the cache.
//
// flutter_svg: no counterpart
func NewSvgCacheScope(cache *Cache, child widgets.Widget, o ...widgets.WidgetOpts) *SvgCacheScope {
	return &SvgCacheScope{InheritedWidget: widgets.NewInheritedWidgetBase(child, o...), cache: cache}
}

// Cache is the cache of the scope.
//
// flutter_svg: no counterpart
func (s *SvgCacheScope) Cache() *Cache { return s.cache }

// UpdateShouldNotify says that the descendants load their documents again when the cache is another one.
//
// flutter_svg: no counterpart
func (s *SvgCacheScope) UpdateShouldNotify(old widgets.Widget) bool {
	return old.(*SvgCacheScope).cache != s.cache
}

// bindingCacheKey is the key of the default cache in the widgets binding (WidgetsBinding.Extension).
type bindingCacheKey struct{}

// CacheOf is the cache that the widgets in the context use: the one of the closest SvgCacheScope, else the cache of
// the binding, made on first use with the default limit. The context depends on the scope.
//
// flutter_svg: svg.dart Svg.cache
func CacheOf(bc widgets.BuildContext) *Cache {
	if s, ok := widgets.DependOn[*SvgCacheScope](bc); ok && s.cache != nil {
		return s.cache
	}
	return widgets.WidgetsBindingOf(bc).Extension(bindingCacheKey{}, func() any { return NewCache() }).(*Cache)
}
