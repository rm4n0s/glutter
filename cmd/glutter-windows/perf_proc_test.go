package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeProc writes a process into a fake /proc directory.
func fakeProc(t *testing.T, proc string, pid, comm, cmd, env, stat, status, rollup string) {
	t.Helper()
	dir := filepath.Join(proc, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"comm": comm + "\n", "cmdline": cmd, "environ": env, "stat": stat, "status": status, "smaps_rollup": rollup} {
		if content == "\n" || (name == "smaps_rollup" && content == "") {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProcFSSample(t *testing.T) {
	proc := t.TempDir()
	const prefix = "/home/u/glutter/.wine-test/prefix"
	mine := "HOME=/home/u\x00WINEPREFIX=" + prefix + "\x00PATH=/bin\x00"
	other := "WINEPREFIX=/home/u/other\x00"
	stat := func(comm, state string, utime, stime int) string {
		return "1 (" + comm + ") " + state + " 1 1 1 0 -1 0 0 0 0 0 " + strconv.Itoa(utime) + " " + strconv.Itoa(stime) + " 0 0 20 0 1 0 1 0 0\n"
	}
	status := "Name:\tx\nVmHWM:\t 2000 kB\nVmRSS:\t 1500 kB\n"
	fakeProc(t, proc, "100", "counter.exe", "/tmp/counter.exe\x00", mine, stat("counter.exe", "S", 150, 30), status, "Pss:   1200 kB\n")
	fakeProc(t, proc, "101", "wineserver", "/usr/bin/wineserver\x00", mine, stat("wineserver", "S", 20, 5), status, "")
	fakeProc(t, proc, "102", "Xvfb", "Xvfb\x00-displayfd\x003\x00", mine, stat("Xvfb", "S", 9, 9), status, "")
	fakeProc(t, proc, "103", "bash", "bash\x00x.sh\x00", mine, stat("bash", "S", 1, 1), status, "")
	fakeProc(t, proc, "104", "firefox", "firefox\x00", other, stat("firefox", "S", 500, 500), status, "")
	fakeProc(t, proc, "105", "gone.exe", "gone.exe\x00", mine, stat("gone.exe", "Z", 1, 1), status, "")
	fakeProc(t, proc, "106", "kernel", "", mine, stat("kernel", "S", 1, 1), "Name:\tkthread\n", "")
	if err := os.MkdirAll(filepath.Join(proc, "self"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := procFS{root: proc, marker: []byte("WINEPREFIX=" + prefix + "\x00"), self: 9999}
	got, err := p.Sample()
	if err != nil {
		t.Fatal(err)
	}
	want := []procSample{
		{PID: 100, Name: "counter.exe", Cmd: "/tmp/counter.exe", Ticks: 180, RSSKB: 1500, HWMKB: 2000, PSSKB: 1200, HasPSS: true},
		{PID: 101, Name: "wineserver", Cmd: "/usr/bin/wineserver", Ticks: 25, RSSKB: 1500, HWMKB: 2000},
	}
	byPID := map[int]procSample{}
	for _, s := range got {
		byPID[s.PID] = s
	}
	if len(got) != len(want) {
		t.Fatalf("Sample = %+v, want the processes %v", got, want)
	}
	for _, w := range want {
		if byPID[w.PID] != w {
			t.Errorf("process %d = %+v, want %+v", w.PID, byPID[w.PID], w)
		}
	}
}

func TestProcFSSampleSkipsItself(t *testing.T) {
	proc := t.TempDir()
	mine := "WINEPREFIX=/p\x00"
	fakeProc(t, proc, "7", "glutter-windows", "x\x00", mine, "1 (x) S 1 1 1 0 -1 0 0 0 0 0 1 1 0 0 20 0 1 0 1 0 0\n", "VmRSS:\t1 kB\n", "")
	got, err := procFS{root: proc, marker: []byte(mine), self: 7}.Sample()
	if err != nil || len(got) != 0 {
		t.Errorf("Sample = %v, %v, want nothing", got, err)
	}
}

func TestProcFSSampleOfAMissingDirectory(t *testing.T) {
	_, err := procFS{root: filepath.Join(t.TempDir(), "nothing")}.Sample()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("procFS.Sample." + tagWindowsPerfProcRead) {
		t.Errorf("route %s, want procFS.Sample.%s", appErr.Route(), tagWindowsPerfProcRead)
	}
}

func TestRealProcSeesOnlyThePrefix(t *testing.T) {
	// A prefix that no process has: the real /proc gives no process (and no error), on Linux.
	if _, err := os.Stat("/proc/self/environ"); err != nil {
		t.Skip("there is no /proc here")
	}
	got, err := newProcFS("/no/such/prefix/" + t.Name()).Sample()
	if err != nil || len(got) != 0 {
		t.Errorf("Sample = %v, %v, want nothing", got, err)
	}
}

func TestSleepContext(t *testing.T) {
	if err := sleepContext(context.Background(), 0); err != nil {
		t.Errorf("sleepContext(0) = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sleepContext(ctx, 1<<40)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("sleepContext."+tagWindowsPerfInterrupted) || !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want WindowsPerfInterrupted from sleepContext wrapping context.Canceled", err)
	}
}
