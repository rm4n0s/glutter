package gles

// handleTable gives small integer names to values that the renderer cannot hold as numbers. The
// renderer's GL interface names every object (shader, buffer, texture) by a uint32, as OpenGL
// ES does; WebGL's objects are JavaScript values, so the WebGL binding keeps each one in a table
// and hands out its index. Name 0 is never given: it is OpenGL's "no object", and WebGL's null.
//
// The table is plain Go over any type, so that its rules run in the normal tests; gl_js.go
// instantiates it with js.Value.
type handleTable[T any] struct {
	values []T
	live   []bool
	free   []uint32
}

// add stores the value and returns its name, reusing the name of a deleted one.
func (t *handleTable[T]) add(v T) uint32 {
	if n := len(t.free); n > 0 {
		id := t.free[n-1]
		t.free = t.free[:n-1]
		t.values[id-1], t.live[id-1] = v, true
		return id
	}
	t.values = append(t.values, v)
	t.live = append(t.live, true)
	return uint32(len(t.values))
}

// get returns the value of the name, and false for 0, a name never given and a deleted one.
func (t *handleTable[T]) get(id uint32) (T, bool) {
	if id == 0 || int(id) > len(t.values) || !t.live[id-1] {
		var zero T
		return zero, false
	}
	return t.values[id-1], true
}

// remove forgets the name, so that a later add may give it again, and returns its value.
// Removing 0, a name never given or one that is already removed returns false.
func (t *handleTable[T]) remove(id uint32) (T, bool) {
	v, ok := t.get(id)
	if !ok {
		return v, false
	}
	var zero T
	t.values[id-1], t.live[id-1] = zero, false
	t.free = append(t.free, id)
	return v, true
}

// count is how many names are in use.
func (t *handleTable[T]) count() int { return len(t.values) - len(t.free) }
