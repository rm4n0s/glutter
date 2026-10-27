package svg

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func key(name string) cacheKey { return cacheKey{source: stringKey{name}} }

func sized(n int) *entry { return &entry{bytes: n} }

func TestCacheEvictsLeastRecentlyUsedBySize(t *testing.T) {
	type step struct {
		op   string // put, get
		name string
		size int
	}
	tests := []struct {
		name      string
		limit     int
		steps     []step
		wantKeys  []string // in the cache at the end
		wantBytes int
	}{
		{"fits", 100, []step{{"put", "a", 40}, {"put", "b", 40}}, []string{"a", "b"}, 80},
		{"the oldest goes", 100, []step{{"put", "a", 40}, {"put", "b", 40}, {"put", "c", 40}}, []string{"b", "c"}, 80},
		{"a read makes an entry recent", 100, []step{{"put", "a", 40}, {"put", "b", 40}, {"get", "a", 0}, {"put", "c", 40}}, []string{"a", "c"}, 80},
		{"a big entry evicts several", 100, []step{{"put", "a", 30}, {"put", "b", 30}, {"put", "c", 30}, {"put", "d", 90}}, []string{"d"}, 90},
		{"an entry over the limit is not kept", 100, []step{{"put", "a", 40}, {"put", "huge", 101}}, []string{"a"}, 40},
		{"the same key is replaced", 100, []step{{"put", "a", 40}, {"put", "a", 60}}, []string{"a"}, 60},
		{"a limit of zero keeps nothing", 0, []step{{"put", "a", 1}}, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCache(CacheOpts{MaximumSizeBytes: some(tt.limit)})
			for _, s := range tt.steps {
				switch s.op {
				case "put":
					c.put(key(s.name), sized(s.size))
				case "get":
					c.get(key(s.name))
				}
			}
			var got []string
			for _, n := range []string{"a", "b", "c", "d", "huge"} {
				if _, ok := c.get(key(n)); ok {
					got = append(got, n)
				}
			}
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("keys = %v, want %v", got, tt.wantKeys)
			}
			for i := range got {
				if got[i] != tt.wantKeys[i] {
					t.Fatalf("keys = %v, want %v", got, tt.wantKeys)
				}
			}
			if c.SizeBytes() != tt.wantBytes || c.Count() != len(tt.wantKeys) {
				t.Errorf("size = %d, count = %d; want %d, %d", c.SizeBytes(), c.Count(), tt.wantBytes, len(tt.wantKeys))
			}
		})
	}
}

func TestCacheLimitCanChange(t *testing.T) {
	c := NewCache()
	if c.MaximumSizeBytes() != DefaultCacheMaximumSizeBytes {
		t.Fatalf("default limit = %d", c.MaximumSizeBytes())
	}
	for _, n := range []string{"a", "b", "c"} {
		c.put(key(n), sized(10))
	}
	c.SetMaximumSizeBytes(25)
	if c.Count() != 2 {
		t.Errorf("count after shrinking = %d, want 2", c.Count())
	}
	if _, ok := c.get(key("a")); ok {
		t.Error("the oldest entry is still there")
	}
	c.SetMaximumSizeBytes(0)
	if c.Count() != 0 || c.SizeBytes() != 0 {
		t.Errorf("count = %d, size = %d after a limit of zero", c.Count(), c.SizeBytes())
	}
	c.SetMaximumSizeBytes(-5)
	if c.MaximumSizeBytes() != 0 {
		t.Errorf("a negative limit is %d", c.MaximumSizeBytes())
	}
	c.SetMaximumSizeBytes(100)
	c.put(key("a"), sized(10))
	c.Clear()
	if c.Count() != 0 || c.SizeBytes() != 0 {
		t.Errorf("count = %d, size = %d after Clear", c.Count(), c.SizeBytes())
	}
}

func TestCacheLoadSharesOneLoad(t *testing.T) {
	c := NewCache()
	var loads atomic.Int32
	release := make(chan struct{})
	started := make(chan struct{}, 8)
	fn := func(context.Context) (*entry, error) {
		loads.Add(1)
		started <- struct{}{}
		<-release
		return sized(10), nil
	}
	var wg sync.WaitGroup
	results := make([]*entry, 5)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e, err := c.load(context.Background(), key("a"), fn)
			if err != nil {
				t.Error(err)
			}
			results[i] = e
		}()
	}
	<-started
	close(release)
	wg.Wait()
	if loads.Load() != 1 {
		t.Errorf("%d loads, want 1", loads.Load())
	}
	for i, e := range results {
		if e == nil || e != results[0] {
			t.Errorf("result %d = %p, want %p", i, e, results[0])
		}
	}
	// A later load is a hit.
	if _, err := c.load(context.Background(), key("a"), fn); err != nil || loads.Load() != 1 {
		t.Errorf("loads = %d, err = %v after a hit", loads.Load(), err)
	}
}

const tagTest = "TestFailure"

func TestCacheLoadFailureIsSharedAndNotCached(t *testing.T) {
	c := NewCache()
	var loads atomic.Int32
	fail := func(context.Context) (*entry, error) {
		loads.Add(1)
		return nil, errors.New(tagTest, "no")
	}
	for range 2 {
		_, err := c.load(context.Background(), key("a"), fail)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagTest {
			t.Fatalf("err = %v", err)
		}
	}
	if loads.Load() != 2 {
		t.Errorf("%d loads: a failure must not be cached", loads.Load())
	}
	if c.Count() != 0 {
		t.Errorf("count = %d", c.Count())
	}
}

func TestCacheLoadHandsOverWhenTheLoaderEnds(t *testing.T) {
	c := NewCache()
	first, cancelFirst := context.WithCancel(context.Background())
	inFirst := make(chan struct{})
	var calls atomic.Int32
	fn := func(ctx context.Context) (*entry, error) {
		if calls.Add(1) == 1 {
			close(inFirst)
			<-ctx.Done()
			return nil, errors.New(tagTest, "canceled")
		}
		return sized(10), nil
	}
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		e, err := c.load(first, key("a"), fn)
		if e != nil || err != nil {
			t.Errorf("the canceled loader got (%v, %v), want (nil, nil)", e, err)
		}
	}()
	<-inFirst
	secondDone := make(chan *entry)
	go func() {
		e, err := c.load(context.Background(), key("a"), fn)
		if err != nil {
			t.Error(err)
		}
		secondDone <- e
	}()
	// Wait until the second caller is waiting on the first.
	for {
		c.mu.Lock()
		f := c.pending[key("a")]
		c.mu.Unlock()
		if f != nil {
			break
		}
	}
	cancelFirst()
	<-firstDone
	if e := <-secondDone; e == nil {
		t.Fatal("the waiting caller got no entry")
	}
	if calls.Load() != 2 {
		t.Errorf("%d loads, want 2", calls.Load())
	}
	if c.Count() != 1 {
		t.Errorf("count = %d", c.Count())
	}
}

func TestCacheLoadStopsWaitingWhenItsContextEnds(t *testing.T) {
	c := NewCache()
	release := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_, _ = c.load(context.Background(), key("a"), func(context.Context) (*entry, error) {
			close(started)
			<-release
			return sized(1), nil
		})
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e, err := c.load(ctx, key("a"), nil); e != nil || err != nil {
		t.Errorf("load = (%v, %v), want (nil, nil)", e, err)
	}
	close(release)
}

func TestSvgPicturesShareOneDocument(t *testing.T) {
	bundle := newCountingBundle(map[string]string{"a.svg": red20x10, "b.svg": leftRed})
	asset := func(name string) widgets.Widget {
		return NewSvgPictureAsset(name, SvgPictureAssetOpts{Bundle: bundle})
	}
	ts := newTester(t)
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{asset("a.svg"), asset("a.svg"), asset("b.svg")})))
	states := func() []*svgPictureState {
		var out []*svgPictureState
		for _, e := range ts.Elements(gluttertest.ByType[*SvgPicture]()) {
			out = append(out, e.(*widgets.StatefulElement).State().(*svgPictureState))
		}
		return out
	}
	waitFor(t, ts, func() bool {
		for _, s := range states() {
			if s.entry == nil {
				return false
			}
		}
		return true
	})
	s := states()
	if s[0].entry != s[1].entry || s[0].entry == s[2].entry {
		t.Errorf("entries: %p %p %p", s[0].entry, s[1].entry, s[2].entry)
	}
	if bundle.count("a.svg") != 1 || bundle.count("b.svg") != 1 {
		t.Errorf("loads: a = %d, b = %d; want one each", bundle.count("a.svg"), bundle.count("b.svg"))
	}
	if CacheOf(s[0].Context()).Count() != 2 {
		t.Errorf("the cache holds %d documents", CacheOf(s[0].Context()).Count())
	}

	// A widget built later finds the document in the cache: it is drawn in its first frame.
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{asset("a.svg"), asset("a.svg"), asset("b.svg"), asset("a.svg")})))
	for i, s := range states() {
		if s.entry == nil {
			t.Errorf("widget %d has no document in its first frame", i)
		}
	}
	if bundle.count("a.svg") != 1 {
		t.Errorf("a.svg was loaded %d times", bundle.count("a.svg"))
	}
}

func TestSvgPicturesLoadingTheSameAssetAtOnceLoadItOnce(t *testing.T) {
	gate := newGateBundle(red20x10)
	ts := newTester(t)
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
		NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate}),
		NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate}),
	})))
	<-gate.started
	// Give the second goroutine the time to find the load of the first.
	for range 3 {
		ts.Pump()
	}
	gate.release()
	waitFor(t, ts, func() bool {
		for _, e := range ts.Elements(gluttertest.ByType[*SvgPicture]()) {
			if e.(*widgets.StatefulElement).State().(*svgPictureState).entry == nil {
				return false
			}
		}
		return true
	})
	if gate.loads.Load() != 1 {
		t.Errorf("%d loads, want 1", gate.loads.Load())
	}
}

func TestSvgPictureTakesOverALoadThatItsPeerEnded(t *testing.T) {
	gate := newGateBundle(red20x10)
	k1, k2 := foundation.NewValueKey("one"), foundation.NewValueKey("two")
	both := func() widgets.Widget {
		return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
			NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate, SvgPictureOpts: SvgPictureOpts{Key: k1}}),
			NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate, SvgPictureOpts: SvgPictureOpts{Key: k2}}),
		}))
	}
	ts := newTester(t)
	ts.PumpWidget(both())
	<-gate.started
	for range 3 {
		ts.Pump()
	}
	// The first widget leaves: its load is canceled, and the second one takes the load over.
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
		NewSvgPictureAsset("a.svg", SvgPictureAssetOpts{Bundle: gate, SvgPictureOpts: SvgPictureOpts{Key: k2}}),
	})))
	<-gate.started
	gate.release()
	waitFor(t, ts, loaded(ts))
	if gate.loads.Load() != 2 || gate.canceled.Load() != 1 {
		t.Errorf("loads = %d, canceled = %d; want 2 and 1", gate.loads.Load(), gate.canceled.Load())
	}
	ts.Pump()
	if got := pixel(ts, 50, 5); got != redC {
		t.Errorf("pixel = %v", got)
	}
}

func TestSvgPictureEvictsBySize(t *testing.T) {
	// A cache that holds one small document: showing a second one evicts the first.
	small := newCountingBundle(map[string]string{"a.svg": red20x10, "b.svg": leftRed})
	one := mustParseDoc(t, red20x10).d.ApproximateBytes()
	cache := NewCache(CacheOpts{MaximumSizeBytes: some(one + one/2)})
	ts := newTester(t)
	show := func(name string) {
		ts.PumpWidget(NewSvgCacheScope(cache, widgets.NewDirectionality(ui.TextDirectionLTR,
			NewSvgPictureAsset(name, SvgPictureAssetOpts{Bundle: small}))))
		waitFor(t, ts, loaded(ts))
	}
	show("a.svg")
	show("b.svg")
	if cache.Count() != 1 {
		t.Fatalf("the cache holds %d documents, want 1", cache.Count())
	}
	// Back to the first: it was evicted, so it loads again.
	show("a.svg")
	if small.count("a.svg") != 2 || small.count("b.svg") != 1 {
		t.Errorf("loads: a = %d, b = %d; want 2 and 1", small.count("a.svg"), small.count("b.svg"))
	}
}

func TestCacheOfIsPerBindingUnlessAScopeGivesOne(t *testing.T) {
	cacheOf := func(ts *gluttertest.Tester, scope *Cache) *Cache {
		var got *Cache
		var w widgets.Widget = widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			got = CacheOf(bc)
			return widgets.NewSizedBox(nil)
		})
		if scope != nil {
			w = NewSvgCacheScope(scope, w)
		}
		ts.PumpWidget(w)
		return got
	}
	a, b := newTester(t), newTester(t)
	ca1, ca2, cb := cacheOf(a, nil), cacheOf(a, nil), cacheOf(b, nil)
	if ca1 == nil || ca1 != ca2 {
		t.Error("two contexts of one binding got different caches")
	}
	if ca1 == cb {
		t.Error("two bindings share a cache")
	}
	own := NewCache()
	if got := cacheOf(a, own); got != own {
		t.Error("the scope's cache was not used")
	}
	if got := cacheOf(a, nil); got != ca1 {
		t.Error("the cache of the binding was lost")
	}
}

func TestSvgCacheScopeChangesTheCache(t *testing.T) {
	first, second := NewCache(), NewCache()
	ts := newTester(t)
	show := func(c *Cache) {
		ts.PumpWidget(NewSvgCacheScope(c, widgets.NewDirectionality(ui.TextDirectionLTR,
			widgets.NewAlign(NewSvgPictureString(red20x10), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft}))))
	}
	show(first)
	show(second)
	if first.Count() != 1 || second.Count() != 1 {
		t.Errorf("first holds %d, second %d; want 1 each", first.Count(), second.Count())
	}
	if got := pixel(ts, 5, 5); got != redC {
		t.Errorf("pixel = %v", got)
	}
}

func TestEqualMemoryBytesShareADocument(t *testing.T) {
	ts := newTester(t)
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
		NewSvgPictureMemory([]byte(red20x10)),
		NewSvgPictureMemory([]byte(red20x10)),
		NewSvgPictureString(red20x10),
	})))
	s := gluttertest.StateOf[*svgPictureState](ts, gluttertest.First(gluttertest.ByType[*SvgPicture]()))
	// A string and the same bytes are two sources (as in flutter_svg), so two documents.
	if n := CacheOf(s.Context()).Count(); n != 2 {
		t.Errorf("the cache holds %d documents, want 2", n)
	}
}
