//go:build linux

package linux

import (
	"testing"
	"time"
)

func TestProcessStartFrom(t *testing.T) {
	// The command name holds spaces and parentheses; field 22 is 123456 ticks after boot.
	stat := "4242 (my (odd) cmd) S 1 4242 4242 0 -1 4194560 1000 0 0 0 5 3 0 0 20 0 8 0 123456 1000000 2000 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0\n"
	uptime := "2000.50 15000.25\n"
	now := time.Unix(1700000000, 250_000_000)
	got, ok := processStartFrom([]byte(stat), []byte(uptime), now)
	// Boot was 2000.5 s before now, and the process started 1234.56 s after it.
	want := now.Add(-2000500*time.Millisecond + 1234560*time.Millisecond)
	if !ok || !got.Equal(want) {
		t.Errorf("start = %v, %v; want %v", got, ok, want)
	}
	for name, in := range map[string][2]string{
		"no parenthesis": {"4242 S 1", uptime},
		"short":          {"4242 (x) S 1 2", uptime},
		"no uptime":      {stat, ""},
		"bad uptime":     {stat, "abc 1\n"},
		"bad ticks":      {"4242 (x) S 1 4242 4242 0 -1 4194560 1000 0 0 0 5 3 0 0 20 0 8 0 abc 1", uptime},
	} {
		if _, ok := processStartFrom([]byte(in[0]), []byte(in[1]), now); ok {
			t.Errorf("%s: parsed", name)
		}
	}
}

func TestProcessStartIsBeforeNowOnLinux(t *testing.T) {
	start, ok := ProcessStart()
	if !ok {
		t.Skip("/proc cannot be read here")
	}
	if age := time.Since(start); age < 0 || age > time.Hour {
		t.Errorf("the test process started %v ago", age)
	}
}
