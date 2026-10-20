package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Tests the layout of NavigationToolbar (Flutter has no test file for it):
// leading, middle and trailing in a 300 x 56 toolbar.
func TestNavigationToolbarLayout(t *testing.T) {
	tests := []struct {
		name string
		o    func() NavigationToolbarOpts
		dir  ui.TextDirection
		want []ui.Rect
	}{
		{
			name: "centered middle",
			o: func() NavigationToolbarOpts {
				return NavigationToolbarOpts{Leading: newBox(40, 10), Middle: newBox(100, 20), Trailing: newBox(30, 30)}
			},
			dir:  ui.TextDirectionLTR,
			want: []ui.Rect{rect(0, 0, 40, 56), rect(100, 18, 100, 20), rect(270, 13, 30, 30)},
		},
		{
			name: "start-justified middle",
			o: func() NavigationToolbarOpts {
				return NavigationToolbarOpts{Leading: newBox(40, 10), Middle: newBox(100, 20), Trailing: newBox(30, 30), NoCenterMiddle: true}
			},
			dir:  ui.TextDirectionLTR,
			want: []ui.Rect{rect(0, 0, 40, 56), rect(56, 18, 100, 20), rect(270, 13, 30, 30)},
		},
		{
			name: "centered middle pushed by the trailing widget",
			o: func() NavigationToolbarOpts {
				return NavigationToolbarOpts{Leading: newBox(40, 10), Middle: newBox(100, 20), Trailing: newBox(150, 30)}
			},
			dir:  ui.TextDirectionLTR,
			want: []ui.Rect{rect(0, 0, 40, 56), rect(56, 18, 78, 20), rect(150, 13, 150, 30)},
		},
		{
			name: "right to left",
			o: func() NavigationToolbarOpts {
				return NavigationToolbarOpts{Leading: newBox(40, 10), Middle: newBox(100, 20), Trailing: newBox(30, 30), NoCenterMiddle: true, MiddleSpacing: foundation.Some(8.0)}
			},
			dir:  ui.TextDirectionRTL,
			want: []ui.Rect{rect(260, 0, 40, 56), rect(152, 18, 100, 20), rect(0, 13, 30, 30)},
		},
		{
			name: "middle only",
			o:    func() NavigationToolbarOpts { return NavigationToolbarOpts{Middle: newBox(100, 20)} },
			dir:  ui.TextDirectionLTR,
			want: []ui.Rect{rect(100, 18, 100, 20)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewSizedBox(NewNavigationToolbar(tc.o()), SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(56.0)}), tc.dir)
			if got := rectsOf(ts); !rectsEqual(got, tc.want) {
				t.Errorf("boxes = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNavigationToolbarAccessors(t *testing.T) {
	w := NewNavigationToolbar()
	if !w.CenterMiddle() || w.MiddleSpacing() != NavigationToolbarMiddleSpacing || w.Leading() != nil || w.Middle() != nil || w.Trailing() != nil {
		t.Errorf("defaults = %v %v", w.CenterMiddle(), w.MiddleSpacing())
	}
}
