package issues

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// The issue "A string built on every layout and paint" of Fishball's list of 2026-11-20, on Glutter
// 1.17.6, fixed in step 52.1 (Glutter 1.17.9): the recovery point of a render object's layout,
// resize and paint built the context of a report ("during PerformLayout()") on every call, also
// when nothing panicked, and a CPU profile of a scroll frame of 150 pictures spent 27% of the frame
// in the concatenation. The dispatch of a pointer event had the same fault for each target of the
// hit test, and worse: it described the target (its type's name, through regular expressions) for
// every event. The reproduction lays out and paints a row of leaves under a parent, through the
// pipeline of a binding with an error reporter as an app's, and dispatches a pointer event to a
// row of targets; it counts the allocations of each child or target, and checks that a panic in
// each is still reported with the context that names it.

// GuardAllocsName is the name of the issue: what an app of a device test is asked for, and the
// prefix of the lines of its report.
const GuardAllocsName = "guard-allocs"

// The cases of the reproduction, in the order of the report.
const (
	// GuardAllocsLayout lays out leaves whose PerformLayout sets their size.
	GuardAllocsLayout = "layout"
	// GuardAllocsResize lays out leaves sized by their parent: PerformResize and PerformLayout.
	GuardAllocsResize = "resize"
	// GuardAllocsPaint paints leaves that draw nothing.
	GuardAllocsPaint = "paint"
	// GuardAllocsDispatch dispatches a pointer event to targets that do nothing with it.
	GuardAllocsDispatch = "dispatch"
)

// guardAllocsCases is what each case may allocate for each child or target, in a release build
// and in a debug build, and the context that the report of a panic there starts with.
var guardAllocsCases = []struct {
	name            string
	most, debugMost float64
	context         string
}{
	// A debug build checks the constraints that each Layout is given, as Flutter's assert does,
	// with a function that names the object (two allocations).
	{GuardAllocsLayout, 0, 2, "during PerformLayout()"},
	{GuardAllocsResize, 0, 2, "during PerformResize()"},
	{GuardAllocsPaint, 0, 0, "during paint()"},
	// The event in the coordinates of each target is a copy (Flutter's PointerEvent.transformed).
	{GuardAllocsDispatch, 1, 1, "while dispatching a pointer event to guardTarget#"},
}

// guardAllocsSlack is how far over its most a case may be, for allocations of other goroutines
// of an app on a device while it counts (a few in thousands of calls); the issue was one or more
// for each call.
const guardAllocsSlack = 0.1

// The size of the measure: the children of the parent (or the targets of the hit test), the
// frames of a run, and the runs, of which the least is reported.
const (
	guardAllocsChildren = 200
	guardAllocsFrames   = 10
	guardAllocsRuns     = 5
)

// GuardAllocsCase is what came of one case.
type GuardAllocsCase struct {
	Name string
	// Allocs is the allocations of the frame for each child (each target for the dispatch): the
	// difference between a frame with the children and one without, the least of several runs.
	Allocs float64
	// Context is the context of the report of a panic in one child, or "" when the panic was not
	// reported.
	Context string
}

// GuardAllocsReport is what came of the reproduction.
type GuardAllocsReport struct {
	// Debug says whether the build that made the report is a debug build (foundation.DebugMode),
	// whose checks allocate more.
	Debug bool
	Cases []GuardAllocsCase
}

// RunGuardAllocs runs the reproduction in the calling goroutine. It changes GOMAXPROCS to one while
// it counts, as testing.AllocsPerRun does, and puts it back.
func RunGuardAllocs() GuardAllocsReport {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	rep := GuardAllocsReport{Debug: foundation.DebugMode}
	for _, c := range guardAllocsCases {
		measure := measureGuardTree
		if c.name == GuardAllocsDispatch {
			measure = measureDispatch
		}
		rep.Cases = append(rep.Cases, measure(c.name))
	}
	return rep
}

// measureGuardTree runs the case of layout, resize or paint: the allocations of a frame of a parent
// with guardAllocsChildren leaves less those of one with none, and the context of the report of a
// panic in the first leaf.
func measureGuardTree(name string) GuardAllocsCase {
	perFrame := func(children int, panicIn string) (float64, string) {
		var reported []error
		s := renderhost.New(renderhost.Opts{ReportError: func(err error) { reported = append(reported, err) }})
		s.Host.DiscardScenes = true
		leaves := make([]rendering.Box, children)
		for i := range leaves {
			leaves[i] = newGuardLeaf(name == GuardAllocsResize)
		}
		parent := newGuardParent(leaves)
		s.SetRoot(parent)
		s.Pump()
		frame := func() {
			if name == GuardAllocsPaint {
				parent.MarkNeedsPaint()
				s.Owner.FlushPaint()
				return
			}
			// Other constraints in each frame, so that every leaf is laid out again.
			parent.wide = !parent.wide
			parent.MarkNeedsLayout()
			s.Owner.FlushLayout()
		}
		frame()
		if panicIn == "" {
			return leastAllocs(frame), ""
		}
		leaves[0].(*guardLeaf).panicIn = panicIn
		context := ""
		func() {
			defer func() { _ = recover() }() // a panic that is not reported: Context stays ""
			frame()
		}()
		// The first report: in a debug build a leaf whose resize panicked has no size, and its
		// layout's check of the constraints is reported after it.
		if len(reported) > 0 {
			if appErr, ok := errors.FromError(reported[0]); ok {
				context, _ = appErr.Metadata["context"].(string)
			}
		}
		return 0, context
	}
	with, _ := perFrame(guardAllocsChildren, "")
	without, _ := perFrame(0, "")
	_, context := perFrame(1, name)
	return GuardAllocsCase{Name: name, Allocs: (with - without) / guardAllocsChildren, Context: context}
}

// measureDispatch runs the case of the dispatch: the allocations of the dispatch of a pointer event
// to guardAllocsChildren targets less those of one to none, and the context of the report of a
// panic in a target.
func measureDispatch(name string) GuardAllocsCase {
	perEvent := func(targets int, panics bool) (float64, string) {
		var reported []error
		s := renderhost.New()
		b := gestures.NewGestureBinding(s.Dispatcher, s.Scheduler, gestures.GestureBindingOpts{
			ErrorReporter: func(err error) { reported = append(reported, err) }})
		result := gestures.NewHitTestResult()
		for range targets {
			result.Add(gestures.NewHitTestEntry(&guardTarget{panics: panics}))
		}
		event := gestures.NewPointerMoveEvent(gestures.PointerEventOpts{Position: ui.Offset{Dx: 4, Dy: 4}})
		dispatch := func() { b.DispatchEvent(event, result) }
		dispatch()
		if !panics {
			return leastAllocs(dispatch), ""
		}
		context := ""
		// The first report: in a debug build a leaf whose resize panicked has no size, and its
		// layout's check of the constraints is reported after it.
		if len(reported) > 0 {
			if appErr, ok := errors.FromError(reported[0]); ok {
				context, _ = appErr.Metadata["context"].(string)
			}
		}
		return 0, context
	}
	with, _ := perEvent(guardAllocsChildren, false)
	without, _ := perEvent(0, false)
	_, context := perEvent(1, true)
	return GuardAllocsCase{Name: name, Allocs: (with - without) / guardAllocsChildren, Context: context}
}

// leastAllocs returns the allocations of one call of fn, the least over guardAllocsRuns runs of
// guardAllocsFrames calls.
func leastAllocs(fn func()) float64 {
	least := math.Inf(1)
	var m runtime.MemStats
	for range guardAllocsRuns {
		runtime.ReadMemStats(&m)
		before := m.Mallocs
		for range guardAllocsFrames {
			fn()
		}
		runtime.ReadMemStats(&m)
		least = math.Min(least, float64(m.Mallocs-before)/guardAllocsFrames)
	}
	return least
}

// Problems returns what is wrong with the report: a case that allocates for each child or target,
// or a panic that is not reported with the context that names where it was. None means the issue
// is fixed on the platform that made the report.
func (r GuardAllocsReport) Problems() []string {
	var out []string
	if len(r.Cases) != len(guardAllocsCases) {
		return []string{fmt.Sprintf("the report has %d cases, want %d", len(r.Cases), len(guardAllocsCases))}
	}
	for i, c := range r.Cases {
		want := guardAllocsCases[i]
		if c.Name != want.name {
			out = append(out, fmt.Sprintf("case %d is %q, want %q", i, c.Name, want.name))
			continue
		}
		most := want.most
		if r.Debug {
			most = want.debugMost
		}
		if c.Allocs > most+guardAllocsSlack {
			out = append(out, fmt.Sprintf("%s: %.3f allocations for each child, want at most %g", c.Name, c.Allocs, most))
		}
		if !strings.HasPrefix(c.Context, want.context) {
			out = append(out, fmt.Sprintf("%s: a panic was reported with the context %q, want %q", c.Name, c.Context, want.context))
		}
	}
	return out
}

// guardAllocsLine and guardAllocsBuildLine are the formats of the lines of the report: their prefix
// says what the line is, so that the lines are found in the output of an app among others.
const (
	guardAllocsLine      = GuardAllocsName + ": case %q allocs %f context %q"
	guardAllocsBuildLine = GuardAllocsName + ": debug %t"
)

// Lines returns the report as text, a line of the build and one for each case, for an app on a
// device to write into a file or its output.
func (r GuardAllocsReport) Lines() []string {
	out := make([]string, 0, 1+len(r.Cases))
	out = append(out, fmt.Sprintf(guardAllocsBuildLine, r.Debug))
	for _, c := range r.Cases {
		out = append(out, fmt.Sprintf(guardAllocsLine, c.Name, c.Allocs, c.Context))
	}
	return out
}

// ParseGuardAllocs reads a report from the lines of an app's file or output; lines that are not of
// the report are passed over. It fails with IssueReportInvalid for a line of the report that does
// not read.
func ParseGuardAllocs(lines []string) (GuardAllocsReport, error) {
	var rep GuardAllocsReport
	for _, line := range lines {
		if at := strings.Index(line, GuardAllocsName+": debug "); at >= 0 {
			// Not %t: fmt reads a word that is not a bool as false, with no error.
			switch strings.TrimSpace(line[at+len(GuardAllocsName+": debug "):]) {
			case "true":
				rep.Debug = true
			case "false":
				rep.Debug = false
			default:
				return GuardAllocsReport{}, errors.New(TagIssueReportInvalid, "the line of the build is not debug true or debug false", "line", line)
			}
			continue
		}
		at := strings.Index(line, GuardAllocsName+": case ")
		if at < 0 {
			continue
		}
		var c GuardAllocsCase
		if _, err := fmt.Sscanf(line[at:], guardAllocsLine, &c.Name, &c.Allocs, &c.Context); err != nil {
			return GuardAllocsReport{}, errors.NewErr(TagIssueReportInvalid, err).SetMetadata("line", line)
		}
		rep.Cases = append(rep.Cases, c)
	}
	return rep, nil
}

// guardParent lays out its children with tight constraints, of one width or another, and paints
// them at its origin. The constraints and the options of Layout are made once: a call through the
// Box interface puts a new value of them on the heap, which every parent pays and which is not what
// the reproduction counts.
type guardParent struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	wide           bool
	narrowC, wideC rendering.Constraints
	layoutOpts     []rendering.LayoutOpts
}

type guardParentData struct {
	rendering.ContainerBoxParentData[rendering.Box]
}

func newGuardParent(children []rendering.Box) *guardParent {
	p := &guardParent{
		narrowC:    rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10}),
		wideC:      rendering.NewBoxConstraintsTight(ui.Size{Width: 11, Height: 10}),
		layoutOpts: []rendering.LayoutOpts{{ParentUsesSize: true}},
	}
	rendering.InitRenderBox(&p.RenderBox, p)
	rendering.InitContainerRenderObject(&p.ContainerRenderObject, p)
	p.AddAll(children)
	return p
}

func (p *guardParent) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*guardParentData); !ok {
		child.AsRenderObject().SetParentData(&guardParentData{})
	}
}

func (p *guardParent) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }

func (p *guardParent) PerformLayout() {
	c := p.narrowC
	if p.wide {
		c = p.wideC
	}
	for ch := p.FirstChild(); ch != nil; ch = ch.ParentData().(*guardParentData).NextSibling {
		ch.Layout(c, p.layoutOpts...)
	}
	p.SetSize(p.BoxConstraints().Biggest())
}

func (p *guardParent) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	for ch := p.FirstChild(); ch != nil; ch = ch.ParentData().(*guardParentData).NextSibling {
		ctx.PaintChild(ch, offset)
	}
}

// guardLeaf is a leaf that takes the smallest size its constraints allow, in PerformResize when it
// is sized by its parent, and draws nothing. It panics in the method of panicIn ("layout",
// "resize" or "paint"), when that is set.
type guardLeaf struct {
	rendering.RenderBox
	sized   bool
	panicIn string
}

func newGuardLeaf(sized bool) *guardLeaf {
	l := &guardLeaf{sized: sized}
	rendering.InitRenderBox(&l.RenderBox, l)
	return l
}

func (l *guardLeaf) SizedByParent() bool { return l.sized }

func (l *guardLeaf) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Smallest() }

func (l *guardLeaf) PerformResize() {
	l.panicWhere(GuardAllocsResize)
	l.SetSize(l.BoxConstraints().Smallest())
}

func (l *guardLeaf) PerformLayout() {
	l.panicWhere(GuardAllocsLayout)
	if !l.sized {
		l.SetSize(l.BoxConstraints().Smallest())
	}
}

func (l *guardLeaf) Paint(*rendering.PaintingContext, ui.Offset) { l.panicWhere(GuardAllocsPaint) }

func (l *guardLeaf) panicWhere(method string) {
	if l.panicIn == method {
		panic(errors.New(tagGuardAllocsPanicked, "the leaf panics where the reproduction asks it to", "method", method))
	}
}

// guardTarget is a hit test target that does nothing with an event, or panics.
type guardTarget struct {
	panics bool
}

func (t *guardTarget) HandleEvent(gestures.PointerEvent, gestures.HitTestEntry) {
	if t.panics {
		panic(errors.New(tagGuardAllocsPanicked, "the target panics where the reproduction asks it to", "method", "HandleEvent"))
	}
}

// The tags of the errors that this file makes.
const tagGuardAllocsPanicked = "GuardAllocsPanicked"
