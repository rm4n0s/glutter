package d3d12

import "github.com/rm4n0s/errors"

// resState is the state of a resource or of one of its subresources: the bits of
// D3D12_RESOURCE_STATES that the device uses (SPEC §9.8 "States"). Direct3D 12 has no GENERAL
// layout, so the device tracks the state of every texture and of every mip level that
// Texture.Level made a view of, and the encoder batches the transitions into one ResourceBarrier.
type resState uint32

// The states of a texture.
const (
	stateCommon              resState = resState(resourceStateCommon)
	stateRenderTarget        resState = resState(resourceStateRenderTarget)
	statePixelShaderResource resState = resState(resourceStatePixelShaderRS)
	stateCopyDest            resState = resState(resourceStateCopyDest)
	stateCopySource          resState = resState(resourceStateCopySource)
)

// String names a state for the messages of an error.
func (s resState) String() string {
	switch s {
	case stateCommon:
		return "COMMON"
	case stateRenderTarget:
		return "RENDER_TARGET"
	case statePixelShaderResource:
		return "PIXEL_SHADER_RESOURCE"
	case stateCopyDest:
		return "COPY_DEST"
	case stateCopySource:
		return "COPY_SOURCE"
	}
	return "unknown state"
}

// allLevels is the level that stands for every subresource of a resource.
const allLevels = -1

// transition is one change of state that a barrier makes: the resource, its subresource (a mip
// level), and the states before and after. A transition of every level at once is listed level by
// level.
type transition[K comparable] struct {
	Res           K
	Level         int
	Before, After resState
}

// lock is the state a subresource was put in during the operation that is open: the same subresource
// in two states in one operation is a conflict.
type lock struct {
	epoch uint64
	state resState
}

// stateTracker follows the state of every subresource of the resources of a device. A resource is
// added with its number of mip levels in the state COMMON. need says that the next operation wants a
// subresource in a state; the transitions that this takes are collected, and flush returns them for
// one ResourceBarrier call. It is used from one goroutine, the UI thread's.
type stateTracker[K comparable] struct {
	levels map[K][]resState
	locks  map[K][]lock
	batch  []transition[K]
	epoch  uint64
	active bool
}

func newStateTracker[K comparable]() *stateTracker[K] {
	return &stateTracker[K]{levels: map[K][]resState{}, locks: map[K][]lock{}}
}

// add starts tracking a resource with the number of levels, all in the state COMMON.
func (t *stateTracker[K]) add(res K, levels int) {
	t.levels[res] = make([]resState, max(levels, 1))
	t.locks[res] = make([]lock, max(levels, 1))
}

// forget stops tracking a resource; the transitions waiting in the batch for it stay.
func (t *stateTracker[K]) forget(res K) {
	delete(t.levels, res)
	delete(t.locks, res)
}

// state is the state of a level; ok is false for a resource that is not tracked.
func (t *stateTracker[K]) state(res K, level int) (resState, bool) {
	ls, ok := t.levels[res]
	if !ok || level < 0 || level >= len(ls) {
		return 0, false
	}
	return ls[level], true
}

// begin starts an operation, a pass or a copy: a subresource that is given a state in it keeps that
// state until end, and another state for it is a conflict.
func (t *stateTracker[K]) begin() {
	t.epoch++
	t.active = true
}

// end ends the operation that begin started.
func (t *stateTracker[K]) end() { t.active = false }

// need says that the operation after it wants the level of the resource (allLevels for every level)
// in the state. A level in another state gets a transition in the batch. It fails with
// D3D12StateConflict when the operation that is open (begin) has put the same subresource in another
// state already (a texture that is drawn into and sampled in one pass, or copied onto itself), and with D3D12ForeignResource for a resource that is not
// tracked. A level that does not exist is D3D12LevelInvalid.
func (t *stateTracker[K]) need(res K, level int, want resState) error {
	ls, ok := t.levels[res]
	if !ok {
		return errors.New(tagD3D12ForeignResource, "the resource is not tracked by this device", "resource", "texture")
	}
	if level != allLevels && (level < 0 || level >= len(ls)) {
		return errors.New(tagD3D12LevelInvalid, "the texture has no such mip level", "level", level, "levels", len(ls))
	}
	from, to := level, level+1
	if level == allLevels {
		from, to = 0, len(ls)
	}
	// Check every level first, so that a conflict leaves the batch as it was.
	if t.active {
		for i := from; i < to; i++ {
			if l := t.locks[res][i]; l.epoch == t.epoch && l.state != want {
				return errors.New(tagD3D12StateConflict, "a texture cannot be in two states in one pass",
					"level", i, "state", l.state.String(), "wanted", want.String())
			}
		}
	}
	for i := from; i < to; i++ {
		if ls[i] != want {
			t.batch = append(t.batch, transition[K]{Res: res, Level: i, Before: ls[i], After: want})
			ls[i] = want
		}
		if t.active {
			t.locks[res][i] = lock{epoch: t.epoch, state: want}
		}
	}
	return nil
}

// flush returns the transitions collected since the last flush, in the order they were asked for,
// and clears the batch: one ResourceBarrier call.
func (t *stateTracker[K]) flush() []transition[K] {
	b := t.batch
	t.batch = nil
	return b
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagD3D12StateConflict is the tag of the programmer error of a texture that is wanted in two
	// states in one pass (SPEC §9.8 "States"); the metadata are "level", "state" and "wanted".
	tagD3D12StateConflict = "D3D12StateConflict"
	// tagD3D12ForeignResource is the tag of the programmer error of a resource that another
	// device made (SPEC §9.8); the metadata name the "resource".
	tagD3D12ForeignResource = "D3D12ForeignResource"
	// tagD3D12LevelInvalid is the tag of the programmer error of a mip level that the texture does not
	// have; the metadata are "level" and "levels".
	tagD3D12LevelInvalid = "D3D12LevelInvalid"
)
