package painting

import (
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
)

const (
	defaultCacheSize      = 1000
	defaultCacheSizeBytes = 100 << 20 // 100 MiB
)

// orderedMap is a map that remembers the order its keys were added in, like
// Dart's LinkedHashMap.
type orderedMap[V any] struct {
	items map[any]V
	keys  []any
}

func newOrderedMap[V any]() *orderedMap[V] { return &orderedMap[V]{items: map[any]V{}} }

func (m *orderedMap[V]) get(k any) (V, bool) { v, ok := m.items[k]; return v, ok }

func (m *orderedMap[V]) set(k any, v V) {
	if _, ok := m.items[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.items[k] = v
}

func (m *orderedMap[V]) remove(k any) (V, bool) {
	v, ok := m.items[k]
	if !ok {
		return v, false
	}
	delete(m.items, k)
	for i, key := range m.keys {
		if key == k {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
	return v, true
}

func (m *orderedMap[V]) len() int { return len(m.keys) }

func (m *orderedMap[V]) clear() { m.items, m.keys = map[any]V{}, nil }

// values returns the values in insertion order.
func (m *orderedMap[V]) values() []V {
	out := make([]V, len(m.keys))
	for i, k := range m.keys {
		out[i] = m.items[k]
	}
	return out
}

// ImageCache class holds a finite number of images in memory, keyed by the
// object an ImageProvider gives for the image (which must be comparable).
//
// Flutter: painting/image_cache.dart ImageCache
type ImageCache struct {
	scheduler *scheduler.SchedulerBinding

	pendingImages *orderedMap[*pendingImage]
	cache         *orderedMap[*cachedImage]
	liveImages    *orderedMap[*cachedImage]

	maximumSize      int
	maximumSizeBytes int
	currentSizeBytes int
}

// NewImageCache creates a cache of 1000 images and 100 MiB. The scheduler binding
// disposes the images after a frame.
//
// Flutter: painting/image_cache.dart ImageCache.new
func NewImageCache(sched *scheduler.SchedulerBinding) *ImageCache {
	return &ImageCache{
		scheduler: sched, pendingImages: newOrderedMap[*pendingImage](), cache: newOrderedMap[*cachedImage](),
		liveImages: newOrderedMap[*cachedImage](), maximumSize: defaultCacheSize, maximumSizeBytes: defaultCacheSizeBytes,
	}
}

// MaximumSize is the maximum number of images to cache.
//
// Flutter: painting/image_cache.dart ImageCache.maximumSize
func (c *ImageCache) MaximumSize() int { return c.maximumSize }

// SetMaximumSize sets the maximum number of images. Setting it to 0 clears the
// cache; a smaller value evicts the oldest images. In debug builds a negative
// value panics with the tag ImageCacheSizeInvalid.
//
// Flutter: painting/image_cache.dart ImageCache.maximumSize
func (c *ImageCache) SetMaximumSize(value int) {
	if foundation.DebugMode && value < 0 {
		panic(errors.New("ImageCacheSizeInvalid", "the maximum size of the image cache must not be negative", "value", value))
	}
	if value == c.maximumSize {
		return
	}
	c.maximumSize = value
	if c.maximumSize == 0 {
		c.Clear()
	} else {
		c.checkCacheSize()
	}
}

// CurrentSize is the current number of cached images.
//
// Flutter: painting/image_cache.dart ImageCache.currentSize
func (c *ImageCache) CurrentSize() int { return c.cache.len() }

// MaximumSizeBytes is the maximum size of the cache in bytes.
//
// Flutter: painting/image_cache.dart ImageCache.maximumSizeBytes
func (c *ImageCache) MaximumSizeBytes() int { return c.maximumSizeBytes }

// SetMaximumSizeBytes sets the maximum size in bytes. Setting it to 0 clears the
// cache. In debug builds a negative value panics with the tag
// ImageCacheSizeInvalid.
//
// Flutter: painting/image_cache.dart ImageCache.maximumSizeBytes
func (c *ImageCache) SetMaximumSizeBytes(value int) {
	if foundation.DebugMode && value < 0 {
		panic(errors.New("ImageCacheSizeInvalid", "the maximum size in bytes of the image cache must not be negative", "value", value))
	}
	if value == c.maximumSizeBytes {
		return
	}
	c.maximumSizeBytes = value
	if c.maximumSizeBytes == 0 {
		c.Clear()
	} else {
		c.checkCacheSize()
	}
}

// CurrentSizeBytes is the current size of the cache in bytes.
//
// Flutter: painting/image_cache.dart ImageCache.currentSizeBytes
func (c *ImageCache) CurrentSizeBytes() int { return c.currentSizeBytes }

// Clear evicts all pending and keepAlive entries from the cache. Live images
// that listeners still use are not cleared.
//
// Flutter: painting/image_cache.dart ImageCache.clear
func (c *ImageCache) Clear() {
	for _, image := range c.cache.values() {
		c.disposeCached(image)
	}
	c.cache.clear()
	for _, p := range c.pendingImages.values() {
		p.removeListener()
	}
	c.pendingImages.clear()
	c.currentSizeBytes = 0
}

// Evict removes the image with the key from the cache, as a pending, keepAlive
// and, unless includeLive is false, live image. It reports whether an entry
// that was pending or kept alive was removed.
//
// Flutter: painting/image_cache.dart ImageCache.evict
func (c *ImageCache) Evict(key any, includeLive ...bool) bool {
	live := true
	if len(includeLive) > 0 {
		live = includeLive[0]
	}
	if live {
		if image, ok := c.liveImages.remove(key); ok {
			c.disposeCached(image)
		}
	}
	if p, ok := c.pendingImages.remove(key); ok {
		p.removeListener()
		return true
	}
	if image, ok := c.cache.remove(key); ok {
		c.currentSizeBytes -= image.sizeBytes
		c.disposeCached(image)
		return true
	}
	return false
}

func (c *ImageCache) touch(key any, image *cachedImage) {
	if image.hasSize && image.sizeBytes <= c.maximumSizeBytes && c.maximumSize > 0 {
		c.currentSizeBytes += image.sizeBytes
		c.cache.set(key, image)
		c.checkCacheSize()
	} else {
		c.disposeCached(image)
	}
}

func (c *ImageCache) trackLiveImage(key any, completer *ImageStreamCompleter, sizeBytes int, hasSize bool) {
	live, ok := c.liveImages.get(key)
	if !ok {
		live = c.newLiveImage(completer, func() { c.liveImages.remove(key) })
		c.liveImages.set(key, live)
	}
	if !live.hasSize && hasSize {
		live.sizeBytes, live.hasSize = sizeBytes, true
	}
}

// PutIfAbsent returns the completer cached for the key, or calls the loader to
// make one and caches it. If the loader panics with an error and onError is not
// nil, onError gets it and PutIfAbsent returns nil.
//
// Flutter: painting/image_cache.dart ImageCache.putIfAbsent
func (c *ImageCache) PutIfAbsent(key any, loader func() *ImageStreamCompleter, onError ImageErrorListener) *ImageStreamCompleter {
	if p, ok := c.pendingImages.get(key); ok {
		return p.completer
	}
	if image, ok := c.cache.remove(key); ok {
		c.trackLiveImage(key, image.completer, image.sizeBytes, image.hasSize)
		c.cache.set(key, image)
		return image.completer
	}
	if live, ok := c.liveImages.get(key); ok {
		c.touch(key, c.newCachedImage(live.completer, live.sizeBytes, live.hasSize))
		return live.completer
	}
	var result *ImageStreamCompleter
	func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok && onError != nil {
					onError(err)
					result = nil
					return
				}
				panic(r)
			}
		}()
		result = loader()
		c.trackLiveImage(key, result, 0, false)
	}()
	if result == nil {
		return nil
	}
	listenedOnce := false
	trackPending := c.maximumSize > 0 && c.maximumSizeBytes > 0
	var pending *pendingImage
	listener := NewImageStreamListener(func(info *ImageInfo, syncCall bool) {
		sizeBytes, hasSize := 0, false
		if info != nil {
			sizeBytes, hasSize = info.SizeBytes(), true
			info.Dispose()
		}
		image := c.newCachedImage(result, sizeBytes, hasSize)
		c.trackLiveImage(key, result, sizeBytes, hasSize)
		if trackPending {
			c.touch(key, image)
		} else {
			c.disposeCached(image)
		}
		c.pendingImages.remove(key)
		if !listenedOnce {
			pending.removeListener()
		}
		listenedOnce = true
	})
	pending = &pendingImage{completer: result, listener: listener}
	if trackPending {
		c.pendingImages.set(key, pending)
	}
	result.AddListener(listener)
	return result
}

// ImageCacheStatus is information about how the cache handles an image key.
//
// Flutter: painting/image_cache.dart ImageCacheStatus
type ImageCacheStatus struct {
	// Pending says the image is being loaded and cached when done.
	Pending bool
	// KeepAlive says the cache holds the image.
	KeepAlive bool
	// Live says a listener holds the image.
	Live bool
}

// Tracked reports whether the image is pending, kept alive or live.
//
// Flutter: painting/image_cache.dart ImageCacheStatus.tracked
func (s ImageCacheStatus) Tracked() bool { return s.Pending || s.KeepAlive || s.Live }

// Untracked reports whether the cache knows nothing of the image.
//
// Flutter: painting/image_cache.dart ImageCacheStatus.untracked
func (s ImageCacheStatus) Untracked() bool { return !s.Pending && !s.KeepAlive && !s.Live }

// String describes the status.
//
// Flutter: painting/image_cache.dart ImageCacheStatus.toString
func (s ImageCacheStatus) String() string {
	b := func(v bool) string {
		if v {
			return "true"
		}
		return "false"
	}
	return "ImageCacheStatus(pending: " + b(s.Pending) + ", live: " + b(s.Live) + ", keepAlive: " + b(s.KeepAlive) + ")"
}

// StatusForKey returns the status of the image with the key.
//
// Flutter: painting/image_cache.dart ImageCache.statusForKey
func (c *ImageCache) StatusForKey(key any) ImageCacheStatus {
	_, pending := c.pendingImages.get(key)
	_, keepAlive := c.cache.get(key)
	_, live := c.liveImages.get(key)
	return ImageCacheStatus{Pending: pending, KeepAlive: keepAlive, Live: live}
}

// ContainsKey reports whether the key is pending or kept alive.
//
// Flutter: painting/image_cache.dart ImageCache.containsKey
func (c *ImageCache) ContainsKey(key any) bool {
	_, pending := c.pendingImages.get(key)
	_, keepAlive := c.cache.get(key)
	return pending || keepAlive
}

// LiveImageCount is the number of live images being held.
//
// Flutter: painting/image_cache.dart ImageCache.liveImageCount
func (c *ImageCache) LiveImageCount() int { return c.liveImages.len() }

// PendingImageCount is the number of images being loaded.
//
// Flutter: painting/image_cache.dart ImageCache.pendingImageCount
func (c *ImageCache) PendingImageCount() int { return c.pendingImages.len() }

// ClearLiveImages clears the live images, so the cache no longer tracks them.
//
// Flutter: painting/image_cache.dart ImageCache.clearLiveImages
func (c *ImageCache) ClearLiveImages() {
	for _, image := range c.liveImages.values() {
		c.disposeCached(image)
	}
	c.liveImages.clear()
}

// checkCacheSize evicts the oldest images until the cache fits its limits.
func (c *ImageCache) checkCacheSize() {
	for c.currentSizeBytes > c.maximumSizeBytes || c.cache.len() > c.maximumSize {
		key := c.cache.keys[0]
		image, _ := c.cache.get(key)
		c.currentSizeBytes -= image.sizeBytes
		c.disposeCached(image)
		c.cache.remove(key)
	}
}

// cachedImage is Flutter's _CachedImage and _LiveImage: a completer the cache
// keeps alive with a handle until a frame after it is disposed.
type cachedImage struct {
	completer *ImageStreamCompleter
	sizeBytes int
	hasSize   bool
	handle    *ImageStreamCompleterHandle

	handleRemove func() // live images only
	removeHook   func()
}

func (c *ImageCache) newCachedImage(completer *ImageStreamCompleter, sizeBytes int, hasSize bool) *cachedImage {
	return &cachedImage{completer: completer, sizeBytes: sizeBytes, hasSize: hasSize, handle: completer.KeepAlive()}
}

func (c *ImageCache) newLiveImage(completer *ImageStreamCompleter, handleRemove func()) *cachedImage {
	img := &cachedImage{completer: completer, handle: completer.KeepAlive()}
	self := func() {
		handleRemove()
		c.disposeCached(img)
	}
	img.handleRemove = self
	img.removeHook = completer.AddOnLastListenerRemovedCallback(self)
	return img
}

// disposeCached releases the handle of the cached image after the next frame.
//
// Flutter: painting/image_cache.dart _CachedImageBase.dispose
func (c *ImageCache) disposeCached(image *cachedImage) {
	if image.removeHook != nil {
		image.removeHook()
		image.removeHook = nil
	}
	handle := image.handle
	if handle == nil {
		return
	}
	image.handle = nil
	c.scheduler.AddPostFrameCallback(func(_ time.Duration) { handle.Dispose() })
}

// pendingImage is Flutter's _PendingImage.
type pendingImage struct {
	completer *ImageStreamCompleter
	listener  *ImageStreamListener
}

func (p *pendingImage) removeListener() { p.completer.RemoveListener(p.listener) }
