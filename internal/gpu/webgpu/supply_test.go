package webgpu

import "testing"

func TestSupplyTakeStartsAndDelivers(t *testing.T) {
	requests := 0
	s := &supply{request: func() { requests++ }}

	if st, f := s.peek(); st != supplyEmpty || f != nil {
		t.Fatalf("a new supply is %v, %v", st, f)
	}
	// An empty supply asks the browser for a device, and says it is coming.
	if d, st, _ := s.take(); d != nil || st != supplyPending || requests != 1 {
		t.Fatalf("first take = %v, %v with %d requests; want nothing, pending, 1", d, st, requests)
	}
	// It is not asked twice.
	s.start()
	if d, st, _ := s.take(); d != nil || st != supplyPending || requests != 1 {
		t.Fatalf("second take = %v, %v with %d requests; want nothing, pending, 1", d, st, requests)
	}
	s.deliver("device", nil)
	if st, _ := s.peek(); st != supplyReady {
		t.Errorf("after deliver the supply is %v, want ready", st)
	}
	s.start() // a device is waiting: nothing is asked
	if requests != 1 {
		t.Errorf("start asked for a device while one was waiting")
	}
	d, st, _ := s.take()
	if d != "device" || st != supplyReady {
		t.Fatalf("take = %v, %v; want the device", d, st)
	}
	// The device is given once; the next take asks for another.
	if d, st, _ := s.take(); d != nil || st != supplyPending || requests != 2 {
		t.Fatalf("take after the device was taken = %v, %v with %d requests; want nothing, pending, 2", d, st, requests)
	}
}

func TestSupplyFailureIsKeptUntilAskedAgain(t *testing.T) {
	requests := 0
	s := &supply{request: func() { requests++ }}
	s.start()
	s.deliver(nil, &failure{tag: "unavailable", message: "no adapter"})
	if st, f := s.peek(); st != supplyFailed || f == nil || f.message != "no adapter" {
		t.Fatalf("peek = %v, %v", st, f)
	}
	// Taking reports the failure every time, without asking again.
	for range 2 {
		if d, st, f := s.take(); d != nil || st != supplyFailed || f == nil || f.tag != "unavailable" {
			t.Fatalf("take = %v, %v, %v; want the failure", d, st, f)
		}
	}
	if requests != 1 {
		t.Errorf("%d requests, want 1", requests)
	}
	// A new request (after a loss) clears it.
	s.start()
	if requests != 2 {
		t.Errorf("start after a failure asked %d times, want 2", requests)
	}
	if st, _ := s.peek(); st != supplyPending {
		t.Errorf("the supply is %v while the new request is out, want pending", st)
	}
}

func TestSupplyReleaseTakesTheDeviceNobodyOpened(t *testing.T) {
	s := &supply{request: func() {}}
	if s.release() != nil {
		t.Fatal("release of an empty supply gave a device")
	}
	s.start()
	s.deliver("device", nil)
	if d := s.release(); d != "device" {
		t.Fatalf("release = %v, want the device", d)
	}
	if st, _ := s.peek(); st != supplyEmpty {
		t.Errorf("after release the supply is %v, want empty", st)
	}
}
