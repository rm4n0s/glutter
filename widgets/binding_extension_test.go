package widgets_test

import (
	"sync"
	"testing"

	. "github.com/rm4n0s/glutter/widgets"
)

type extensionKey struct{}

func TestBindingExtensionIsMadeOncePerBinding(t *testing.T) {
	_, bc := windowApp(t)
	b := WidgetsBindingOf(bc)
	made := 0
	create := func() any { made++; return &struct{ n int }{n: made} }
	first := b.Extension(extensionKey{}, create)
	second := b.Extension(extensionKey{}, create)
	if first != second || made != 1 {
		t.Errorf("made %d values, first %p second %p", made, first, second)
	}
	// Another key has a value of its own, and another binding does not share it.
	if other := b.Extension("other", create); other == first || made != 2 {
		t.Errorf("another key shared the value (made %d)", made)
	}
	_, bc2 := windowApp(t)
	if third := WidgetsBindingOf(bc2).Extension(extensionKey{}, create); third == first {
		t.Error("two bindings share an extension")
	}
}

func TestBindingExtensionIsSafeFromManyGoroutines(t *testing.T) {
	_, bc := windowApp(t)
	b := WidgetsBindingOf(bc)
	var mu sync.Mutex
	made := 0
	var wg sync.WaitGroup
	results := make([]any, 16)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = b.Extension(extensionKey{}, func() any { mu.Lock(); made++; mu.Unlock(); return new(int) })
		}()
	}
	wg.Wait()
	for _, r := range results {
		if r != results[0] {
			t.Fatal("the goroutines got different values")
		}
	}
	if made != 1 {
		t.Errorf("created %d values", made)
	}
}
