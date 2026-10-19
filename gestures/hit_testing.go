package gestures

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// HitTestable is an object that can be hit tested: asked which of the objects
// in it are under a position.
//
// Deviation: the deprecated hitTest, which tested the implicit view, is not
// ported (SPEC §12.10).
//
// Flutter: gestures/hit_test.dart HitTestable
type HitTestable interface {
	// HitTestInView adds to result the objects under the position in the
	// view, front first.
	//
	// Flutter: gestures/hit_test.dart HitTestable.hitTestInView
	HitTestInView(result *HitTestResult, position ui.Offset, viewID ui.ViewID)
}

// HitTestDispatcher is an object that can dispatch events to the targets of a
// hit test.
//
// Flutter: gestures/hit_test.dart HitTestDispatcher
type HitTestDispatcher interface {
	// DispatchEvent sends the event to the targets in the result.
	//
	// Flutter: gestures/hit_test.dart HitTestDispatcher.dispatchEvent
	DispatchEvent(event PointerEvent, result *HitTestResult)
}

// HitTestTarget is an object that can receive events.
//
// Flutter: gestures/hit_test.dart HitTestTarget
type HitTestTarget interface {
	// HandleEvent is called with an event that happened on the target. The
	// event is transformed to the coordinates of the entry.
	//
	// Flutter: gestures/hit_test.dart HitTestTarget.handleEvent
	HandleEvent(event PointerEvent, entry HitTestEntry)
}

// NativeHitTestTarget marks a HitTestTarget that stands for a native view:
// events over it belong to the platform, not to the framework. Implement
// IsNativeHitTestTarget to mark a target.
//
// Flutter: gestures/hit_test.dart NativeHitTestTarget
type NativeHitTestTarget interface {
	IsNativeHitTestTarget()
}

// HitTestEntry is the data about a target that a hit test found: the target and
// the transform from the global coordinates to its own. A type that adds data to
// an entry (such as rendering's BoxHitTestEntry) embeds HitTestEntryBase.
//
// Deviation: Flutter's HitTestEntry is a generic class that subclasses extend;
// Go's is an interface with a struct to embed (SPEC §12.10).
//
// Flutter: gestures/hit_test.dart HitTestEntry
type HitTestEntry interface {
	// Target is the object the hit test found.
	//
	// Flutter: gestures/hit_test.dart HitTestEntry.target
	Target() HitTestTarget
	// Transform is the matrix that takes the global coordinates to those of
	// the target. It is nil until the entry is added to a HitTestResult.
	//
	// Flutter: gestures/hit_test.dart HitTestEntry.transform
	Transform() *ui.Matrix4

	setTransform(m *ui.Matrix4)
}

// HitTestEntryBase is the implementation of HitTestEntry to create or embed.
//
// Flutter: gestures/hit_test.dart HitTestEntry
type HitTestEntryBase struct {
	target    HitTestTarget
	transform *ui.Matrix4
}

// NewHitTestEntry creates an entry for the target.
//
// Flutter: gestures/hit_test.dart HitTestEntry.new
func NewHitTestEntry(target HitTestTarget) *HitTestEntryBase {
	return &HitTestEntryBase{target: target}
}

// Target is the object the hit test found.
//
// Flutter: gestures/hit_test.dart HitTestEntry.target
func (e *HitTestEntryBase) Target() HitTestTarget { return e.target }

// Transform is the matrix that takes the global coordinates to those of the
// target. It is nil until the entry is added to a HitTestResult.
//
// Flutter: gestures/hit_test.dart HitTestEntry.transform
func (e *HitTestEntryBase) Transform() *ui.Matrix4 { return e.transform }

func (e *HitTestEntryBase) setTransform(m *ui.Matrix4) { e.transform = m }

// String describes the entry by its target.
//
// Flutter: gestures/hit_test.dart HitTestEntry.toString
func (e *HitTestEntryBase) String() string {
	return foundation.DescribeIdentity(e) + "(" + describeTarget(e.target) + ")"
}

func describeTarget(target HitTestTarget) string {
	if s, ok := target.(interface{ String() string }); ok {
		return s.String()
	}
	return foundation.ToStringShort(target)
}

// transformPart is one step of the transform from the parent's coordinates to
// the child's, not yet multiplied with those before it.
type transformPart interface {
	// multiply returns the part applied to rhs from the left, as if the part
	// were a matrix.
	multiply(rhs *ui.Matrix4) *ui.Matrix4
}

type matrixTransformPart struct{ matrix *ui.Matrix4 }

func (p matrixTransformPart) multiply(rhs *ui.Matrix4) *ui.Matrix4 { return p.matrix.Multiply(rhs) }

type offsetTransformPart struct{ offset ui.Offset }

func (p offsetTransformPart) multiply(rhs *ui.Matrix4) *ui.Matrix4 {
	return ui.NewMatrix4TranslationValues(float32(p.offset.Dx), float32(p.offset.Dy), 0).Multiply(rhs)
}

// hitTestState is what a HitTestResult and the results wrapped around it share.
type hitTestState struct {
	path []HitTestEntry
	// transforms are matrices that include their ancestors', so that they are
	// relative to the global coordinates.
	transforms []*ui.Matrix4
	// localTransforms are the parts pushed since the last global matrix was
	// made, relative to their parent. They are made global when first used.
	localTransforms []transformPart
}

// HitTestResult is the list of targets a hit test found, front first, with the
// transform to each.
//
// Flutter: gestures/hit_test.dart HitTestResult
type HitTestResult struct {
	s *hitTestState
}

// NewHitTestResult creates an empty result.
//
// Flutter: gestures/hit_test.dart HitTestResult.new
func NewHitTestResult() *HitTestResult {
	return &HitTestResult{&hitTestState{transforms: []*ui.Matrix4{ui.NewMatrix4Identity()}}}
}

// NewHitTestResultWrap creates a result that shares its path and transforms
// with result. A subtype that adds data to a result (rendering's
// BoxHitTestResult) wraps one.
//
// Flutter: gestures/hit_test.dart HitTestResult.wrap
func NewHitTestResultWrap(result *HitTestResult) *HitTestResult {
	return &HitTestResult{result.s}
}

// Path is the targets the hit test found, front first.
//
// Flutter: gestures/hit_test.dart HitTestResult.path
func (r *HitTestResult) Path() []HitTestEntry { return r.s.path }

func (r *HitTestResult) globalizeTransforms() {
	s := r.s
	if len(s.localTransforms) == 0 {
		return
	}
	last := s.transforms[len(s.transforms)-1]
	for _, part := range s.localTransforms {
		last = part.multiply(last)
		s.transforms = append(s.transforms, last)
	}
	s.localTransforms = s.localTransforms[:0]
}

func (r *HitTestResult) lastTransform() *ui.Matrix4 {
	r.globalizeTransforms()
	return r.s.transforms[len(r.s.transforms)-1]
}

// Add adds an entry to the path. The entry's transform is the one that is on
// top of the transform stack now.
//
// Flutter: gestures/hit_test.dart HitTestResult.add
func (r *HitTestResult) Add(entry HitTestEntry) {
	if foundation.DebugMode && entry.Transform() != nil {
		panic(errors.New("HitTestEntryAddedTwice", "a hit test entry was added to a result twice",
			"entry", entry))
	}
	entry.setTransform(r.lastTransform())
	r.s.path = append(r.s.path, entry)
}

// PushTransform pushes a transform from the parent's coordinates to the
// child's. Call PopTransform when the child is done. Its third row and column
// must be the identity's (see RemovePerspectiveTransform), or a debug build
// panics with the tag HitTestTransformPerspective.
//
// Flutter: gestures/hit_test.dart HitTestResult.pushTransform
func (r *HitTestResult) PushTransform(transform *ui.Matrix4) {
	if foundation.DebugMode && !debugHasNoPerspective(transform) {
		panic(errors.New("HitTestTransformPerspective",
			"the third row and third column of a transform matrix for pointer events must be (0, 0, 1, 0) "+
				"to ensure that a transformed point is directly under the pointing device; "+
				"run the paint matrix through RemovePerspectiveTransform",
			"transform", transform))
	}
	r.s.localTransforms = append(r.s.localTransforms, matrixTransformPart{transform})
}

// PushOffset pushes a translation from the parent's coordinates to the
// child's. Call PopTransform when the child is done.
//
// Flutter: gestures/hit_test.dart HitTestResult.pushOffset
func (r *HitTestResult) PushOffset(offset ui.Offset) {
	r.s.localTransforms = append(r.s.localTransforms, offsetTransformPart{offset})
}

// PopTransform removes the transform pushed last.
//
// Flutter: gestures/hit_test.dart HitTestResult.popTransform
func (r *HitTestResult) PopTransform() {
	s := r.s
	if n := len(s.localTransforms); n > 0 {
		s.localTransforms = s.localTransforms[:n-1]
	} else {
		s.transforms = s.transforms[:len(s.transforms)-1]
	}
	if foundation.DebugMode && len(s.transforms) == 0 {
		panic(errors.New("HitTestTransformStackEmpty", "PopTransform was called more often than PushTransform"))
	}
}

func debugHasNoPerspective(m *ui.Matrix4) bool {
	want := [4]float32{0, 0, 1, 0}
	for i, w := range want {
		if math.Abs(float64(m.Entry(2, i)-w)) >= foundation.PrecisionErrorTolerance {
			return false
		}
		if math.Abs(float64(m.Entry(i, 2)-w)) >= foundation.PrecisionErrorTolerance {
			return false
		}
	}
	return true
}

// String lists the entries of the path.
//
// Flutter: gestures/hit_test.dart HitTestResult.toString
func (r *HitTestResult) String() string {
	if len(r.s.path) == 0 {
		return "HitTestResult(<empty path>)"
	}
	s := "HitTestResult("
	for i, e := range r.s.path {
		if i > 0 {
			s += ", "
		}
		s += describeEntry(e)
	}
	return s + ")"
}

func describeEntry(e HitTestEntry) string {
	if s, ok := e.(interface{ String() string }); ok {
		return s.String()
	}
	return foundation.DescribeIdentity(e) + "(" + describeTarget(e.Target()) + ")"
}
