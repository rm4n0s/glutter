package ole

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// fakeData is a DataSource that counts how often it was closed.
type fakeData struct {
	name   string
	closed int
	log    *[]string
}

func (f *fakeData) Formats() ([]uint32, error)  { return nil, nil }
func (f *fakeData) Read(uint32) ([]byte, error) { return nil, nil }
func (f *fakeData) Close() {
	f.closed++
	*f.log = append(*f.log, "close "+f.name)
}

// fakeHandler records what the Target calls and answers the effects it is given.
type fakeHandler struct {
	log    *[]string
	answer uint32
}

func (h *fakeHandler) DragEnter(d DataSource, _ uint32, _ win32.Point, _ uint32) uint32 {
	*h.log = append(*h.log, "enter "+d.(*fakeData).name)
	return h.answer
}
func (h *fakeHandler) DragOver(uint32, win32.Point, uint32) uint32 {
	*h.log = append(*h.log, "over")
	return h.answer
}
func (h *fakeHandler) DragLeave() { *h.log = append(*h.log, "leave") }
func (h *fakeHandler) Drop(d DataSource, _ uint32, _ win32.Point, _ uint32) uint32 {
	*h.log = append(*h.log, "drop "+d.(*fakeData).name)
	return h.answer
}

func TestTargetKeepsAndReleasesTheDataInOrder(t *testing.T) {
	tests := []struct {
		name string
		run  func(tg *Target, mk func(string) *fakeData)
		want []string
	}{
		{"enter, over, leave", func(tg *Target, mk func(string) *fakeData) {
			tg.Enter(mk("a"), 0, win32.Point{}, 0)
			tg.Over(0, win32.Point{}, 0)
			tg.Leave()
		}, []string{"enter a", "over", "leave", "close a"}},
		{"enter, over, drop releases both references", func(tg *Target, mk func(string) *fakeData) {
			tg.Enter(mk("a"), 0, win32.Point{}, 0)
			tg.Over(0, win32.Point{}, 0)
			tg.Drop(mk("b"), 0, win32.Point{}, 0)
		}, []string{"enter a", "over", "drop b", "close a", "close b"}},
		{"over and leave with no enter do nothing", func(tg *Target, mk func(string) *fakeData) {
			if got := tg.Over(0, win32.Point{}, win32.DropEffectCopy); got != win32.DropEffectNone {
				t.Errorf("Over with no enter = %#x", got)
			}
			tg.Leave()
		}, nil},
		{"a second enter releases the first drag", func(tg *Target, mk func(string) *fakeData) {
			tg.Enter(mk("a"), 0, win32.Point{}, 0)
			tg.Enter(mk("b"), 0, win32.Point{}, 0)
			tg.Leave()
		}, []string{"enter a", "close a", "enter b", "leave", "close b"}},
		{"over after the drag ended does nothing", func(tg *Target, mk func(string) *fakeData) {
			tg.Enter(mk("a"), 0, win32.Point{}, 0)
			tg.Leave()
			tg.Over(0, win32.Point{}, 0)
		}, []string{"enter a", "leave", "close a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var log []string
			tg := &Target{H: &fakeHandler{log: &log, answer: win32.DropEffectCopy}}
			var all []*fakeData
			mk := func(name string) *fakeData {
				d := &fakeData{name: name, log: &log}
				all = append(all, d)
				return d
			}
			tt.run(tg, mk)
			if !slices.Equal(log, tt.want) {
				t.Errorf("calls = %v, want %v", log, tt.want)
			}
			for _, d := range all {
				if d.closed > 1 {
					t.Errorf("data %s was closed %d times", d.name, d.closed)
				}
			}
		})
	}
}

func TestTargetReturnsTheHandlersEffect(t *testing.T) {
	var log []string
	tg := &Target{H: &fakeHandler{log: &log, answer: win32.DropEffectMove}}
	if got := tg.Enter(&fakeData{log: &log}, 0, win32.Point{}, 0); got != win32.DropEffectMove {
		t.Errorf("Enter = %#x", got)
	}
	if got := tg.Over(0, win32.Point{}, 0); got != win32.DropEffectMove {
		t.Errorf("Over = %#x", got)
	}
	if got := tg.Drop(&fakeData{log: &log}, 0, win32.Point{}, 0); got != win32.DropEffectMove {
		t.Errorf("Drop = %#x", got)
	}
}
