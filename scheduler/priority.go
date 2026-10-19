package scheduler

// Priority is the priority of a task that SchedulerBinding runs between frames.
// A larger value is more important. The zero value is PriorityIdle.
//
// Flutter: scheduler/priority.dart Priority
type Priority int

// The priorities Flutter defines. Use Add and Sub to get one in between.
const (
	// PriorityIdle is the priority of tasks that can wait: nothing is waiting
	// for them.
	//
	// Flutter: scheduler/priority.dart Priority.idle
	PriorityIdle Priority = 0
	// PriorityAnimation is the priority of work that must finish before the
	// next frame, so that animations do not stutter.
	//
	// Flutter: scheduler/priority.dart Priority.animation
	PriorityAnimation Priority = 100000
	// PriorityTouch is the priority of work that responds to user input.
	//
	// Flutter: scheduler/priority.dart Priority.touch
	PriorityTouch Priority = 200000
)

// PriorityMaxOffset is the largest offset Add and Sub use; a larger one is
// cut to it, so that a priority does not reach the next level.
//
// Flutter: scheduler/priority.dart Priority.kMaxOffset
const PriorityMaxOffset = 10000

// Value is the priority as a number.
//
// Flutter: scheduler/priority.dart Priority.value
func (p Priority) Value() int { return int(p) }

// Add returns the priority offset more important (or less, for a negative
// offset). The offset is cut to PriorityMaxOffset.
//
// Flutter: scheduler/priority.dart Priority.+
func (p Priority) Add(offset int) Priority {
	offset = max(-PriorityMaxOffset, min(PriorityMaxOffset, offset))
	return p + Priority(offset)
}

// Sub returns the priority offset less important. It is Add(-offset).
//
// Flutter: scheduler/priority.dart Priority.-
func (p Priority) Sub(offset int) Priority { return p.Add(-offset) }
