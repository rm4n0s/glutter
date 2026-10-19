package widgets

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// AnyPageStorageKey is a PageStorageKey of any value type.
//
// Flutter: widgets/page_storage.dart PageStorageKey
type AnyPageStorageKey interface {
	foundation.LocalKey
	// StorageValue is the value of the key.
	//
	// Flutter: foundation/key.dart ValueKey.value
	StorageValue() any
}

// PageStorageKey is a ValueKey that identifies the widget whose state a
// PageStorageBucket keeps: the keys of a widget and of its ancestors up to the
// PageStorage form the identifier of the entry.
//
// Flutter: widgets/page_storage.dart PageStorageKey
type PageStorageKey[T comparable] struct {
	foundation.ValueKey[T]
}

// NewPageStorageKey creates a key from the value.
//
// Flutter: widgets/page_storage.dart PageStorageKey.new
func NewPageStorageKey[T comparable](value T) PageStorageKey[T] {
	return PageStorageKey[T]{foundation.NewValueKey(value)}
}

// StorageValue is the value of the key.
//
// Flutter: foundation/key.dart ValueKey.value
func (k PageStorageKey[T]) StorageValue() any { return k.Value() }

// String describes the key, such as "[<'list'>]".
//
// Flutter: widgets/page_storage.dart PageStorageKey
func (k PageStorageKey[T]) String() string { return k.ValueKey.String() }

// storageEntryIdentifier is the list of keys that identify an entry, as a
// comparable value.
type storageEntryIdentifier struct {
	keys  string
	count int
}

func (i storageEntryIdentifier) isNotEmpty() bool { return i.count > 0 }

// String describes the identifier, such as "StorageEntryIdentifier([<'a'>]:[<'b'>])".
//
// Flutter: widgets/page_storage.dart _StorageEntryIdentifier.toString
func (i storageEntryIdentifier) String() string { return "StorageEntryIdentifier(" + i.keys + ")" }

// PageStorageBucket is a storage of state that a PageStorage widget offers to
// the widgets below it, so that they keep it when they are rebuilt from scratch,
// for instance when a route is rebuilt.
//
// Flutter: widgets/page_storage.dart PageStorageBucket
type PageStorageBucket struct {
	storage map[any]any
}

// NewPageStorageBucket creates an empty bucket.
//
// Flutter: widgets/page_storage.dart PageStorageBucket.new
func NewPageStorageBucket() *PageStorageBucket { return &PageStorageBucket{} }

// maybeAddKey adds the key of the context's widget, if it is a page storage key,
// and returns whether to go on to the parent (false at a PageStorage widget).
func maybeAddKey(bc BuildContext, keys *[]string) bool {
	w := bc.Widget()
	if k, ok := w.Key().(AnyPageStorageKey); ok {
		*keys = append(*keys, fmt.Sprintf("%T(%v)", k, k.StorageValue()))
	}
	_, isStorage := w.(*PageStorage)
	return !isStorage
}

func (b *PageStorageBucket) computeIdentifier(bc BuildContext) storageEntryIdentifier {
	var keys []string
	if maybeAddKey(bc, &keys) {
		bc.VisitAncestorElements(func(e Element) bool { return maybeAddKey(e, &keys) })
	}
	return storageEntryIdentifier{keys: strings.Join(keys, ":"), count: len(keys)}
}

// PageStorageOpts are the optional parameters of the state functions of a
// PageStorageBucket.
//
// Flutter: widgets/page_storage.dart PageStorageBucket.writeState
type PageStorageOpts struct {
	// Identifier is the key of the entry, instead of the one computed from the
	// keys of the context and its ancestors.
	Identifier any
}

// WriteState saves the data for the context. Without an identifier, the entry is
// the keys of the context's widget and its ancestors; nothing is saved if there
// is none.
//
// Flutter: widgets/page_storage.dart PageStorageBucket.writeState
func (b *PageStorageBucket) WriteState(bc BuildContext, data any, opts ...PageStorageOpts) {
	if b.storage == nil {
		b.storage = map[any]any{}
	}
	if id := firstOpts(opts).Identifier; id != nil {
		b.storage[id] = data
		return
	}
	if id := b.computeIdentifier(bc); id.isNotEmpty() {
		b.storage[id] = data
	}
}

// ReadState returns the data saved for the context, or nil.
//
// Flutter: widgets/page_storage.dart PageStorageBucket.readState
func (b *PageStorageBucket) ReadState(bc BuildContext, opts ...PageStorageOpts) any {
	if b.storage == nil {
		return nil
	}
	if id := firstOpts(opts).Identifier; id != nil {
		return b.storage[id]
	}
	if id := b.computeIdentifier(bc); id.isNotEmpty() {
		return b.storage[id]
	}
	return nil
}

// PageStorage is a widget that gives the widgets below it a PageStorageBucket to
// save their state in.
//
// Flutter: widgets/page_storage.dart PageStorage
type PageStorage struct {
	StatelessWidget
	child  Widget
	bucket *PageStorageBucket
}

// NewPageStorage creates the widget.
//
// Flutter: widgets/page_storage.dart PageStorage.new
func NewPageStorage(bucket *PageStorageBucket, child Widget, opts ...WidgetOpts) *PageStorage {
	return &PageStorage{StatelessWidget: NewStatelessWidgetBase(opts...), child: child, bucket: bucket}
}

// Child is the widget below this one.
//
// Flutter: widgets/page_storage.dart PageStorage.child
func (w *PageStorage) Child() Widget { return w.child }

// Bucket is the bucket the widgets below use.
//
// Flutter: widgets/page_storage.dart PageStorage.bucket
func (w *PageStorage) Bucket() *PageStorageBucket { return w.bucket }

// Build returns the child.
//
// Flutter: widgets/page_storage.dart PageStorage.build
func (w *PageStorage) Build(BuildContext) Widget { return w.child }

// MaybePageStorageOf is the bucket of the closest PageStorage above the context,
// or nil.
//
// Flutter: widgets/page_storage.dart PageStorage.maybeOf
func MaybePageStorageOf(bc BuildContext) *PageStorageBucket {
	if w, ok := FindAncestorWidgetOfExactType[*PageStorage](bc); ok {
		return w.bucket
	}
	return nil
}

// PageStorageOf is the bucket of the closest PageStorage above the context. It
// panics with the tag PageStorageMissing in debug builds when there is none.
//
// Flutter: widgets/page_storage.dart PageStorage.of
func PageStorageOf(bc BuildContext) *PageStorageBucket {
	b := MaybePageStorageOf(bc)
	if b == nil && foundation.DebugMode {
		panic(errors.New("PageStorageMissing",
			"PageStorageOf was called with a context that does not contain a PageStorage widget; no PageStorage widget ancestor could be found starting from the context that was passed; this can happen because you are using a widget that looks for a PageStorage ancestor, but no such ancestor exists",
			"context", fmt.Sprint(bc)))
	}
	return b
}
