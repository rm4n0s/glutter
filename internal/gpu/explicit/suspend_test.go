package explicit

import "testing"

// idleDevice is a device that counts WaitIdle.
type idleDevice struct {
	Device
	waited int
}

func (d *idleDevice) WaitIdle() { d.waited++ }

func TestRendererSuspendWaitsForTheDeviceThatCan(t *testing.T) {
	r := &Renderer{}
	if r.Suspend() {
		t.Error("Suspend waited with no device")
	}
	r.dev = &idleDevice{}
	if !r.Suspend() || r.dev.(*idleDevice).waited != 1 {
		t.Error("Suspend did not wait for the device that can")
	}
	r.dev = struct{ Device }{}
	if r.Suspend() {
		t.Error("Suspend waited for a device that cannot")
	}
}
