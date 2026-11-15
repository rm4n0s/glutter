package gpu

import (
	"container/list"

	"github.com/rm4n0s/errors"
)

// atlasClasses are the sizes of the square slots a page is cut into. A tile goes
// in the smallest slot that holds it.
var atlasClasses = []int{32, 64, 128, 256}

// TagAtlasFull is the tag of the error that Atlas.Put returns when every slot of the atlas is in
// use in the frame and the atlas is as large as a frame may make it. A renderer compares with
// it to leave the shape out and draw the rest of the frame.
//
// Flutter: no counterpart
const TagAtlasFull = "AtlasFull"

// FeatureAtlasFull is what the frame statistics of a renderer list among the reasons of
// commands that were left out when the coverage atlas had no room for a shape.
//
// Flutter: no counterpart
const FeatureAtlasFull = "coverage atlas full"

// MaxAtlasTile is the largest width or height of a tile of an Atlas; a bigger
// mask is cut into tiles.
//
// Flutter: no counterpart
const MaxAtlasTile = 256

// AtlasEntry is a tile of an Atlas: where it is, and how much of its slot it
// uses. An entry stays valid until the atlas evicts it, which only happens for
// entries that were not used in the current frame.
//
// Flutter: no counterpart
type AtlasEntry[K comparable] struct {
	// Key is what the tile was stored under.
	Key K
	// Page is the index of the page the tile is on.
	Page int
	// X and Y are the top left corner of the tile's slot in the page, in pixels.
	X, Y int
	// W and H are the size of the tile's content, at the slot's corner.
	W, H int
	// Empty says that the tile holds nothing: it has no slot, and W and H are 0.
	// Storing the empty tiles keeps a renderer from rasterizing them again.
	Empty bool
	// OX and OY are free for the renderer to keep a tile's offset in; the atlas
	// does not use them.
	OX, OY int

	class    int // the slot size
	slot     int // the index of the slot on its page
	lastUsed uint64
	elem     *list.Element
}

// atlasPage is one page: a square cut into slots of one class.
type atlasPage struct {
	class int
	side  int   // slots per row
	free  []int // free slot indices
	used  int
	// lastUsed is the last frame that used a tile of the page, or made the page.
	lastUsed uint64
}

// AtlasStats says what an Atlas holds.
//
// Flutter: no counterpart
type AtlasStats struct {
	// Pages is the number of pages the atlas has now, Entries the number of tiles
	// stored (including empty ones), and Evicted the number of tiles evicted so far.
	Pages, Entries, Evicted int
	// PagesOverBudget is how many of the pages are more than the atlas keeps: a
	// frame needed them, and they are freed when a frame no longer does.
	PagesOverBudget int
	// PagesFreed is the number of pages freed so far.
	PagesFreed int
}

// Atlas keeps track of where the 8-bit coverage tiles of a renderer's coverage
// atlas are (SPEC §9.1, "General paths"). The pages are squares of pageSize pixels
// that the renderer makes into textures. A page is cut into square slots of one
// size, so a freed slot is always reusable, and tiles are cached by key: the
// least recently used are evicted when the atlas is full, but never one that was
// used in the current frame, because the frame's draws still read it.
//
// The atlas keeps maxPages pages. A frame whose tiles do not fit them makes it
// grow, up to the frame limit (SetFrameLimit; without one it does not grow), so
// that a frame of many large shapes is drawn whole; EndFrame frees the pages
// over maxPages that the frame did not use. A freed page leaves its index empty
// until a new page takes it, so the index of a page never changes.
//
// It does not touch the pixels; the renderer uploads a tile into its slot after
// Put. It is not safe for concurrent use.
//
// Flutter: no counterpart (Skia's GrDrawOpAtlas)
type Atlas[K comparable] struct {
	pageSize   int
	maxPages   int          // the pages it keeps
	frameLimit int          // the pages a frame may make it grow to; at least maxPages
	pages      []*atlasPage // by index; nil where a page was freed
	live       int          // the pages that are not nil
	entries    map[K]*AtlasEntry[K]
	lru        *list.List // front is the most recently used
	frame      uint64
	evicted    int
	freed      int
}

// NewAtlas makes an atlas that keeps at most maxPages pages of pageSize pixels
// each. pageSize should be a multiple of MaxAtlasTile.
//
// Flutter: no counterpart
func NewAtlas[K comparable](pageSize, maxPages int) *Atlas[K] {
	return &Atlas[K]{pageSize: pageSize, maxPages: maxPages, frameLimit: maxPages, entries: map[K]*AtlasEntry[K]{}, lru: list.New()}
}

// SetFrameLimit sets the number of pages that the tiles of one frame may make
// the atlas grow to, and returns the atlas. A limit below the pages it keeps is
// those pages.
//
// Flutter: no counterpart
func (a *Atlas[K]) SetFrameLimit(pages int) *Atlas[K] {
	a.frameLimit = max(pages, a.maxPages)
	return a
}

// BeginFrame starts a frame: the tiles used from now on are protected from
// eviction until the next call.
//
// Flutter: no counterpart
func (a *Atlas[K]) BeginFrame() { a.frame++ }

// EndFrame ends a frame. While the atlas has more pages than it keeps, it frees
// the pages that no tile of the frame is on, with their tiles, and returns their
// indices, for the renderer to free their textures. A page that the frame used
// stays, however many there are.
//
// Flutter: no counterpart
func (a *Atlas[K]) EndFrame() []int {
	if a.live <= a.maxPages {
		return nil
	}
	var out []int
	// From the last index, so that the indices in use stay low.
	for i := len(a.pages) - 1; i >= 0 && len(out) < a.live-a.maxPages; i-- {
		if p := a.pages[i]; p != nil && p.lastUsed != a.frame {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		return nil
	}
	gone := make(map[int]bool, len(out))
	for _, i := range out {
		gone[i] = true
		a.pages[i] = nil
		a.live--
		a.freed++
	}
	for _, e := range a.entries {
		if !e.Empty && gone[e.Page] {
			a.lru.Remove(e.elem)
			delete(a.entries, e.Key)
			a.evicted++
		}
	}
	for len(a.pages) > 0 && a.pages[len(a.pages)-1] == nil {
		a.pages = a.pages[:len(a.pages)-1]
	}
	return out
}

// Get returns the tile stored under key, or nil, and marks it used in this frame.
//
// Flutter: no counterpart
func (a *Atlas[K]) Get(key K) *AtlasEntry[K] {
	e := a.entries[key]
	if e != nil {
		a.touch(e)
	}
	return e
}

func (a *Atlas[K]) touch(e *AtlasEntry[K]) {
	e.lastUsed = a.frame
	if !e.Empty {
		a.pages[e.Page].lastUsed = a.frame
	}
	a.lru.MoveToFront(e.elem)
}

// Put stores a tile of w x h pixels under key and returns it; the caller uploads
// the pixels at (X, Y) of the page. A tile of no size is stored as an empty entry
// with no slot. The key must not be stored already. It fails with
// AtlasTileInvalid for a size that is negative or above MaxAtlasTile, and
// AtlasFull when every slot is in use in this frame and the atlas has as many
// pages as a frame may make.
//
// Flutter: no counterpart
func (a *Atlas[K]) Put(key K, w, h int) (*AtlasEntry[K], error) {
	if w < 0 || h < 0 || w > MaxAtlasTile || h > MaxAtlasTile {
		return nil, errors.New(tagAtlasTileInvalid, "the tile size is not valid for the atlas", "width", w, "height", h, "max", MaxAtlasTile)
	}
	if _, dup := a.entries[key]; dup {
		panic(errors.New(tagAtlasKeyStored, "the key is already in the atlas"))
	}
	e := &AtlasEntry[K]{Key: key, W: w, H: h, lastUsed: a.frame}
	if w == 0 || h == 0 {
		e.Empty, e.W, e.H = true, 0, 0
	} else {
		class := atlasClasses[len(atlasClasses)-1]
		for _, c := range atlasClasses {
			if max(w, h) <= c {
				class = c
				break
			}
		}
		if err := a.allocate(e, class); err != nil {
			return nil, err
		}
	}
	e.elem = a.lru.PushFront(e)
	a.entries[key] = e
	return e, nil
}

// allocate gives the entry a slot of the class.
func (a *Atlas[K]) allocate(e *AtlasEntry[K], class int) error {
	e.class = class
	// A free slot on a page of the class.
	for i, p := range a.pages {
		if p != nil && p.class == class && len(p.free) > 0 {
			a.take(e, i, p)
			return nil
		}
	}
	// A new page, while the atlas has fewer than it keeps.
	if a.live < a.maxPages {
		i := a.addPage(class)
		a.take(e, i, a.pages[i])
		return nil
	}
	// The least recently used tile of the class that no draw of this frame needs.
	for el := a.lru.Back(); el != nil; el = el.Prev() {
		old := el.Value.(*AtlasEntry[K])
		if old.class == class && !old.Empty && old.lastUsed != a.frame {
			a.evict(old)
			a.take(e, old.Page, a.pages[old.Page])
			return nil
		}
	}
	// A whole page of another class, if none of its tiles is in use.
	for i, p := range a.pages {
		if p != nil && p.class != class && p.lastUsed != a.frame {
			for _, old := range a.entriesOn(i) {
				a.evict(old)
			}
			a.pages[i] = a.newPage(class)
			a.take(e, i, a.pages[i])
			return nil
		}
	}
	// Every slot is read by a draw of this frame: a page more than the atlas keeps, which
	// EndFrame frees when a frame does not use it.
	if a.live < a.frameLimit {
		i := a.addPage(class)
		a.take(e, i, a.pages[i])
		return nil
	}
	return errors.New(TagAtlasFull, "every slot of the atlas is in use in this frame", "class", class, "pages", a.live)
}

// addPage makes a page of the class at the first empty index and returns the index.
func (a *Atlas[K]) addPage(class int) int {
	a.live++
	for i, p := range a.pages {
		if p == nil {
			a.pages[i] = a.newPage(class)
			return i
		}
	}
	a.pages = append(a.pages, a.newPage(class))
	return len(a.pages) - 1
}

func (a *Atlas[K]) newPage(class int) *atlasPage {
	side := a.pageSize / class
	p := &atlasPage{class: class, side: side, free: make([]int, 0, side*side), lastUsed: a.frame}
	for s := side*side - 1; s >= 0; s-- {
		p.free = append(p.free, s)
	}
	return p
}

// take gives the entry the first free slot of the page.
func (a *Atlas[K]) take(e *AtlasEntry[K], page int, p *atlasPage) {
	slot := p.free[len(p.free)-1]
	p.free = p.free[:len(p.free)-1]
	p.used++
	p.lastUsed = a.frame
	e.Page, e.slot = page, slot
	e.X, e.Y = (slot%p.side)*p.class, (slot/p.side)*p.class
}

// evict removes a tile and frees its slot.
func (a *Atlas[K]) evict(e *AtlasEntry[K]) {
	if !e.Empty {
		p := a.pages[e.Page]
		if p.class == e.class {
			p.free = append(p.free, e.slot)
			p.used--
		}
	}
	a.lru.Remove(e.elem)
	delete(a.entries, e.Key)
	a.evicted++
}

// entriesOn returns the tiles that have a slot on the page.
func (a *Atlas[K]) entriesOn(page int) []*AtlasEntry[K] {
	var out []*AtlasEntry[K]
	for _, e := range a.entries {
		if !e.Empty && e.Page == page {
			out = append(out, e)
		}
	}
	return out
}

// Pages returns the number of page indices in use: the highest index of a page plus one. The
// renderer keeps a texture for each page; an index whose page was freed has none.
//
// Flutter: no counterpart
func (a *Atlas[K]) Pages() int { return len(a.pages) }

// KeptPages returns the number of pages the atlas keeps from one frame to the next.
//
// Flutter: no counterpart
func (a *Atlas[K]) KeptPages() int { return a.maxPages }

// FrameLimit returns the number of pages a frame may make the atlas grow to.
//
// Flutter: no counterpart
func (a *Atlas[K]) FrameLimit() int { return a.frameLimit }

// PageSize returns the size of a page in pixels.
//
// Flutter: no counterpart
func (a *Atlas[K]) PageSize() int { return a.pageSize }

// Stats says what the atlas holds.
//
// Flutter: no counterpart
func (a *Atlas[K]) Stats() AtlasStats {
	return AtlasStats{
		Pages: a.live, Entries: len(a.entries), Evicted: a.evicted,
		PagesOverBudget: max(a.live-a.maxPages, 0), PagesFreed: a.freed,
	}
}

// Clear forgets every tile and every page.
//
// Flutter: no counterpart
func (a *Atlas[K]) Clear() {
	a.pages = nil
	a.live = 0
	a.entries = map[K]*AtlasEntry[K]{}
	a.lru.Init()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAtlasKeyStored   = "AtlasKeyStored"
	tagAtlasTileInvalid = "AtlasTileInvalid"
)
