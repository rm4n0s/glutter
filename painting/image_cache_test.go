package painting

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// loadedCompleter is a completer that already has an image of w x h pixels.
func loadedCompleter(pt *paintingTester, w, h int) *ImageStreamCompleter {
	c := NewImageStreamCompleter(pt.PaintingBinding)
	c.SetImage(NewImageInfo(testImage(w, h)))
	return c
}

func put(c *ImageCache, key any, comp *ImageStreamCompleter) *ImageStreamCompleter {
	return c.PutIfAbsent(key, func() *ImageStreamCompleter { return comp }, nil)
}

func TestImageCacheMaximumSize(t *testing.T) {
	tests := []struct {
		name     string
		max      int
		puts     []string
		touch    string
		wantKept []string
		wantGone []string
	}{
		{"evicts the oldest", 2, []string{"a", "b", "c"}, "", []string{"b", "c"}, []string{"a"}},
		{"a hit refreshes the entry", 2, []string{"a", "b"}, "a", []string{"a", "c"}, []string{"b"}},
		{"zero caches nothing", 0, []string{"a"}, "", nil, []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt := newPaintingTester(t)
			cache := pt.ImageCache()
			cache.SetMaximumSize(tt.max)
			for _, k := range tt.puts {
				put(cache, k, loadedCompleter(pt, 2, 2))
			}
			if tt.touch != "" {
				put(cache, tt.touch, nil)
				put(cache, "c", loadedCompleter(pt, 2, 2))
			}
			for _, k := range tt.wantKept {
				if !cache.ContainsKey(k) {
					t.Errorf("%q should be cached", k)
				}
			}
			for _, k := range tt.wantGone {
				if cache.ContainsKey(k) {
					t.Errorf("%q should be evicted", k)
				}
			}
		})
	}
}

func TestImageCacheMaximumSizeBytes(t *testing.T) {
	pt := newPaintingTester(t)
	cache := pt.ImageCache()
	cache.SetMaximumSizeBytes(250)
	put(cache, "a", loadedCompleter(pt, 5, 5)) // 100 bytes each
	put(cache, "b", loadedCompleter(pt, 5, 5))
	put(cache, "c", loadedCompleter(pt, 5, 5))
	if cache.CurrentSizeBytes() != 200 || cache.CurrentSize() != 2 || cache.ContainsKey("a") {
		t.Errorf("size %d bytes %d, a cached %v", cache.CurrentSize(), cache.CurrentSizeBytes(), cache.ContainsKey("a"))
	}
	// An image bigger than the whole cache is not cached.
	put(cache, "huge", loadedCompleter(pt, 10, 10))
	if cache.ContainsKey("huge") {
		t.Error("an image larger than the limit must not be cached")
	}
	// Shrinking the limit evicts.
	cache.SetMaximumSizeBytes(100)
	if cache.CurrentSize() != 1 || cache.CurrentSizeBytes() != 100 {
		t.Errorf("after shrinking: %d images, %d bytes", cache.CurrentSize(), cache.CurrentSizeBytes())
	}
	cache.SetMaximumSizeBytes(0)
	if cache.CurrentSize() != 0 || cache.CurrentSizeBytes() != 0 {
		t.Error("a zero limit clears the cache")
	}
	if cache.MaximumSize() != 1000 || NewImageCache(pt.Scheduler()).MaximumSizeBytes() != 100<<20 {
		t.Error("the defaults are 1000 images and 100 MiB")
	}
}

func TestImageCacheClearAndEvict(t *testing.T) {
	pt := newPaintingTester(t)
	cache := pt.ImageCache()
	put(cache, "a", loadedCompleter(pt, 2, 2))
	put(cache, "b", loadedCompleter(pt, 2, 2))
	if !cache.Evict("a") || cache.Evict("a") {
		t.Error("Evict reports whether something was removed")
	}
	if cache.CurrentSizeBytes() != 16 {
		t.Errorf("bytes = %d", cache.CurrentSizeBytes())
	}
	cache.Clear()
	if cache.CurrentSize() != 0 || cache.CurrentSizeBytes() != 0 || cache.ContainsKey("b") {
		t.Error("Clear empties the cache")
	}
}

func TestImageCachePendingAndLive(t *testing.T) {
	pt := newPaintingTester(t)
	cache := pt.ImageCache()
	pending := NewImageStreamCompleter(pt.PaintingBinding)
	pending.AddListener((&imageRecorder{}).listener()) // a widget that waits for the image
	got := put(cache, "p", pending)
	if got != pending {
		t.Fatal("PutIfAbsent returns the completer of the loader")
	}
	if s := cache.StatusForKey("p"); !s.Pending || s.KeepAlive || !s.Live {
		t.Errorf("status = %v", s)
	}
	if cache.PendingImageCount() != 1 || cache.LiveImageCount() != 1 {
		t.Errorf("pending %d live %d", cache.PendingImageCount(), cache.LiveImageCount())
	}
	// A second request while it loads gets the same completer without calling the loader.
	again := cache.PutIfAbsent("p", func() *ImageStreamCompleter { t.Fatal("loader called"); return nil }, nil)
	if again != pending {
		t.Error("a pending image is shared")
	}
	pending.SetImage(NewImageInfo(testImage(2, 2)))
	if s := cache.StatusForKey("p"); s.Pending || !s.KeepAlive || !s.Live {
		t.Errorf("after loading: %v", s)
	}
	if s := cache.StatusForKey("p"); s.String() != "ImageCacheStatus(pending: false, live: true, keepAlive: true)" {
		t.Errorf("String = %s", s)
	}
	if !cache.StatusForKey("p").Tracked() || !cache.StatusForKey("nothing").Untracked() {
		t.Error("tracked and untracked")
	}
}

func TestImageCacheLiveImagesOutliveEviction(t *testing.T) {
	pt := newPaintingTester(t)
	cache := pt.ImageCache()
	cache.SetMaximumSize(1)
	a := loadedCompleter(pt, 2, 2)
	l := (&imageRecorder{}).listener()
	a.AddListener(l)
	put(cache, "a", a)
	put(cache, "b", loadedCompleter(pt, 2, 2)) // evicts a from the cache
	if cache.ContainsKey("a") {
		t.Fatal("a is evicted")
	}
	if s := cache.StatusForKey("a"); !s.Live || s.KeepAlive {
		t.Errorf("a is live but not kept: %v", s)
	}
	// Asking for it again finds the live image and puts it back.
	got := cache.PutIfAbsent("a", func() *ImageStreamCompleter { t.Fatal("loader called"); return nil }, nil)
	if got != a || !cache.ContainsKey("a") {
		t.Error("a live image is found again")
	}
	// The completer stays alive until a frame has passed after the cache let go.
	cache.Clear()
	if a.Disposed() {
		t.Fatal("the listener still holds the completer")
	}
	a.RemoveListener(l)
	pt.frame(0)
	if !a.Disposed() {
		t.Error("the completer is disposed when the last holder is gone")
	}
	if cache.LiveImageCount() != 0 {
		t.Errorf("live images = %d", cache.LiveImageCount())
	}
}

func TestImageCacheClearLiveImages(t *testing.T) {
	pt := newPaintingTester(t)
	cache := pt.ImageCache()
	a := loadedCompleter(pt, 2, 2)
	a.AddListener((&imageRecorder{}).listener())
	put(cache, "a", a)
	if cache.LiveImageCount() != 1 {
		t.Fatalf("live images = %d", cache.LiveImageCount())
	}
	cache.ClearLiveImages()
	if cache.LiveImageCount() != 0 {
		t.Error("ClearLiveImages forgets the live images")
	}
	if !cache.Evict("a", false) {
		t.Error("evicting without live images still removes the cached one")
	}
}

func TestImageCacheLoaderPanic(t *testing.T) {
	pt := newPaintingTester(t)
	cache := pt.ImageCache()
	boom := errors.New("Boom", "boom")
	var got error
	res := cache.PutIfAbsent("x", func() *ImageStreamCompleter { panic(boom) }, func(err error) { got = err })
	if res != nil || got != boom {
		t.Errorf("result %v error %v", res, got)
	}
	if cache.ContainsKey("x") || cache.LiveImageCount() != 0 {
		t.Error("a failed load leaves nothing in the cache")
	}
	defer func() {
		if recover() != boom {
			t.Error("without an error handler the panic propagates")
		}
	}()
	cache.PutIfAbsent("x", func() *ImageStreamCompleter { panic(boom) }, nil)
}

func TestImageCacheRemovesPendingWhenCleared(t *testing.T) {
	pt := newPaintingTester(t)
	cache := pt.ImageCache()
	pending := NewImageStreamCompleter(pt.PaintingBinding)
	put(cache, "p", pending)
	cache.Clear()
	if cache.PendingImageCount() != 0 || pending.HasListeners() {
		t.Error("clearing removes the pending image and its listener")
	}
	pending.SetImage(NewImageInfo(testImage(2, 2))) // nobody listens now
	if cache.ContainsKey("p") {
		t.Error("a cleared pending image must not be cached when it arrives")
	}
}
