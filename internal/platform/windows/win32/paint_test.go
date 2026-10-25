package win32

import "testing"

func TestPaintHandlerTable(t *testing.T) {
	if PaintHandler(0x10) != nil {
		t.Fatal("a window with no handler has one")
	}
	var painted []uintptr
	SetPaintHandler(0x10, func(h uintptr) { painted = append(painted, h) })
	SetPaintHandler(0x20, func(h uintptr) { painted = append(painted, h+1) })
	PaintHandler(0x10)(0x10)
	PaintHandler(0x20)(0x20)
	if len(painted) != 2 || painted[0] != 0x10 || painted[1] != 0x21 {
		t.Errorf("painted = %v", painted)
	}
	SetPaintHandler(0x10, nil)
	if PaintHandler(0x10) != nil {
		t.Error("the handler was not removed")
	}
	if PaintHandler(0x20) == nil {
		t.Error("removing one handler removed another")
	}
	SetPaintHandler(0x20, nil)
}
