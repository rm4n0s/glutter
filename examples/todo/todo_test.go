package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// useLinux makes the framework behave as on the Linux desktop, which the example is for: the
// tests default to Android.
func useLinux(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the platform override is a debug-only feature")
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func newTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 560, Height: 520}})
	ts.PumpWidget(NewTodoApp())
	return ts
}

func pageState(ts *gluttertest.Tester) *todoPageState {
	return gluttertest.StateOf[*todoPageState](ts, gluttertest.ByType[*TodoPage]())
}

// titles are the titles of the tasks, in the order of the rows.
func titles(ts *gluttertest.Tester) []string {
	var out []string
	for _, task := range pageState(ts).tasks {
		out = append(out, task.Title)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// settle pumps frames until the goroutine that waits for the result of the dialog has handed
// it to the UI thread.
func settle(t *testing.T, ts *gluttertest.Tester, done func() bool) {
	t.Helper()
	for range 200 {
		ts.Pump(10 * time.Millisecond)
		if done() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the result of the dialog did not arrive")
}

func TestEachRowHasEditAndDeleteButtons(t *testing.T) {
	ts := newTester(t)
	for _, tt := range []struct {
		name string
		got  int
		want int
	}{
		{"rows", ts.Count(gluttertest.ByType[*m.DataTable]()), 1},
		{"edit buttons", ts.Count(gluttertest.ByTooltip("Edit")), 3},
		{"delete buttons", ts.Count(gluttertest.ByTooltip("Delete")), 3},
		{"checkboxes", ts.Count(gluttertest.ByType[*m.Checkbox]()), 3},
	} {
		if tt.got != tt.want {
			t.Errorf("%s: %d, want %d", tt.name, tt.got, tt.want)
		}
	}
	for _, text := range []string{"Done", "Task", "Actions", "Buy milk", "Write the report", "Call the bank"} {
		if n := ts.Count(gluttertest.ByText(text)); n != 1 {
			t.Errorf("the text %q is shown %d times, want 1", text, n)
		}
	}
}

func TestAddAndDelete(t *testing.T) {
	tests := []struct {
		name  string
		act   func(ts *gluttertest.Tester)
		want  []string
		title string
	}{
		{"add with the button", func(ts *gluttertest.Tester) {
			ts.EnterText(gluttertest.ByType[*m.TextField](), "Pay rent")
			ts.Tap(gluttertest.ByText("Add"))
		}, []string{"Buy milk", "Write the report", "Call the bank", "Pay rent"}, "To-do: 4 left"},
		{"a blank task is not added", func(ts *gluttertest.Tester) {
			ts.EnterText(gluttertest.ByType[*m.TextField](), "   ")
			ts.Tap(gluttertest.ByText("Add"))
		}, []string{"Buy milk", "Write the report", "Call the bank"}, "To-do: 3 left"},
		{"delete the first row", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.First(gluttertest.ByTooltip("Delete")))
		}, []string{"Write the report", "Call the bank"}, "To-do: 2 left"},
		{"delete the last row", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.Last(gluttertest.ByTooltip("Delete")))
		}, []string{"Buy milk", "Write the report"}, "To-do: 2 left"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			tt.act(ts)
			ts.Pump(50 * time.Millisecond)
			if got := titles(ts); !equal(got, tt.want) {
				t.Errorf("tasks = %v, want %v", got, tt.want)
			}
			if n := ts.Count(gluttertest.ByTooltip("Edit")); n != len(tt.want) {
				t.Errorf("%d edit buttons, want %d", n, len(tt.want))
			}
			if n := ts.Count(gluttertest.ByText(tt.title)); n != 1 {
				t.Errorf("the app bar %q is shown %d times, want 1", tt.title, n)
			}
		})
	}
}

func TestDoneChangesTheCount(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByType[*m.Checkbox]()))
	ts.Pump(50 * time.Millisecond)
	if !pageState(ts).tasks[0].Done {
		t.Error("the first task is not done")
	}
	if n := ts.Count(gluttertest.ByText("To-do: 2 left")); n != 1 {
		t.Errorf("the app bar shows %d times 'To-do: 2 left', want 1", n)
	}
}

func TestEdit(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		button string
		want   string
	}{
		{"save a new title", "Buy oat milk", "Save", "Buy oat milk"},
		{"cancel keeps the title", "Something else", "Cancel", "Buy milk"},
		{"a blank title keeps the title", "  ", "Save", "Buy milk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			ts.Tap(gluttertest.First(gluttertest.ByTooltip("Edit")))
			ts.PumpAndSettle()
			if n := ts.Count(gluttertest.ByText("Edit task")); n != 1 {
				t.Fatalf("the dialog is shown %d times, want 1", n)
			}
			field := gluttertest.Descendant(gluttertest.ByType[*m.AlertDialog](), gluttertest.ByType[*m.TextField]())
			ts.EnterText(field, tt.answer)
			ts.Tap(gluttertest.ByText(tt.button))
			ts.PumpAndSettle()
			settle(t, ts, func() bool { return titles(ts)[0] == tt.want })
			if n := ts.Count(gluttertest.ByText("Edit task")); n != 0 {
				t.Errorf("the dialog is still shown")
			}
			if got := titles(ts)[0]; got != tt.want {
				t.Errorf("title = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditDialogStartsWithTheTitle(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.Last(gluttertest.ByTooltip("Edit")))
	ts.PumpAndSettle()
	d := gluttertest.StateOf[*editDialogState](ts, gluttertest.ByType[*EditDialog]())
	if got := d.field.Text(); got != "Call the bank" {
		t.Errorf("the field starts with %q, want the title of the task", got)
	}
}

func TestGolden(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.Last(gluttertest.ByType[*m.Checkbox]()))
	ts.PumpAndSettle()
	ts.MatchesGolden("todo")
}
