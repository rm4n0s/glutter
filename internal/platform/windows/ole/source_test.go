package ole

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestSourceMachineQuery(t *testing.T) {
	left := uint32(win32.MKLButton)
	tests := []struct {
		name      string
		buttons   uint32
		cancelled bool
		escape    bool
		key       uint32
		want      Decision
		hr        win32.HRESULT
	}{
		{"the button is still down", left, false, false, win32.MKLButton, Continue, win32.SOK},
		{"the button is down with a modifier", left, false, false, win32.MKLButton | win32.MKShift, Continue, win32.SOK},
		{"the button was released", left, false, false, 0, Drop, win32.DragDropSDrop},
		{"another button is not the one", left, false, false, win32.MKRButton, Drop, win32.DragDropSDrop},
		{"escape cancels", left, false, true, win32.MKLButton, Cancel, win32.DragDropSCancel},
		{"escape wins over the release", left, false, true, 0, Cancel, win32.DragDropSCancel},
		{"the platform cancelled", left, true, false, win32.MKLButton, Cancel, win32.DragDropSCancel},
		{"a right-button drag goes on with the right button", win32.MKRButton, false, false, win32.MKRButton, Continue, win32.SOK},
		{"a right-button drag drops on its release", win32.MKRButton, false, false, win32.MKLButton, Drop, win32.DragDropSDrop},
		{"two buttons: one released, the other still holds", left | win32.MKRButton, false, false, win32.MKRButton, Continue, win32.SOK},
		{"two buttons: both released", left | win32.MKRButton, false, false, 0, Drop, win32.DragDropSDrop},
		{"no button at the start drops at once", 0, false, false, 0, Drop, win32.DragDropSDrop},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SourceMachine{Buttons: tt.buttons, Cancelled: tt.cancelled}
			got := m.Query(tt.escape, tt.key)
			if got != tt.want || got.HRESULT() != tt.hr {
				t.Errorf("Query = %v (%v), want %v (%v)", got, got.HRESULT(), tt.want, tt.hr)
			}
		})
	}
}

func TestSourceMachineFeedbackRecordsTheEffect(t *testing.T) {
	m := &SourceMachine{}
	if hr := m.Feedback(win32.DropEffectMove); hr != win32.DragDropSUseDefaultCursors || m.LastEffect != win32.DropEffectMove {
		t.Errorf("Feedback = %v, effect %#x", hr, m.LastEffect)
	}
}

func TestButtonKeys(t *testing.T) {
	tests := []struct {
		buttons int
		want    uint32
	}{
		{0, 0}, {1, win32.MKLButton}, {2, win32.MKRButton}, {4, win32.MKMButton},
		{1 | 2, win32.MKLButton | win32.MKRButton}, {8, 0}, {1 | 8, win32.MKLButton},
	}
	for _, tt := range tests {
		if got := ButtonKeys(tt.buttons); got != tt.want {
			t.Errorf("ButtonKeys(%d) = %#x, want %#x", tt.buttons, got, tt.want)
		}
	}
}
