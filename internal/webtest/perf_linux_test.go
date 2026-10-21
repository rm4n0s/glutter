//go:build linux

package webtest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestParseStat(t *testing.T) {
	tests := []struct {
		name      string
		stat      string
		pgrp, cpu int
		ok        bool
	}{
		{"a plain process", "123 (chrome) S 1 456 456 0 -1 4194560 100 0 0 0 70 30 0 0 20 0 1 0 1000 1000 100 18446744073709551615", 456, 100, true},
		{"a name with blanks and parentheses", "9 (Web Content (x) y) R 1 777 777 0 -1 4194560 100 0 0 0 5 6 0 0 20 0 1 0 1000", 777, 11, true},
		{"too few fields", "9 (x) S 1 2", 0, 0, false},
		{"no name", "9 x S 1 2 3", 0, 0, false},
		{"a field that is not a number", "9 (x) S 1 pg 3 0 -1 0 0 0 0 0 a b 0", 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pgrp, cpu, ok := parseStat(tc.stat)
			if ok != tc.ok || pgrp != tc.pgrp || cpu != tc.cpu {
				t.Errorf("parseStat = %d, %d, %v; want %d, %d, %v", pgrp, cpu, ok, tc.pgrp, tc.cpu, tc.ok)
			}
		})
	}
}

// fakeProc writes a directory laid out like /proc with the given stat files.
func fakeProc(t *testing.T, stats map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for pid, stat := range stats {
		if err := os.MkdirAll(filepath.Join(dir, pid), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, pid, "stat"), []byte(stat), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Entries that are not processes, and a process that ended between the listing and the read.
	if err := os.WriteFile(filepath.Join(dir, "version"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "999"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestGroupCPUTime(t *testing.T) {
	dir := fakeProc(t, map[string]string{
		"10": "10 (timeout) S 1 10 10 0 -1 0 0 0 0 0 100 50 0 0",  // 150 ticks, group 10
		"11": "11 (chrome) S 10 10 10 0 -1 0 0 0 0 0 200 100 0 0", // 300 ticks, group 10
		"20": "20 (Xvfb) S 1 20 20 0 -1 0 0 0 0 0 10 5 0 0",       // 15 ticks, group 20
		"30": "30 (other) S 1 30 30 0 -1 0 0 0 0 0 1000 1000 0 0", // not ours
		"31": "31 (broken) S 1",                                   // unreadable
	})
	tests := []struct {
		name   string
		groups []int
		want   time.Duration
	}{
		{"one group", []int{10}, 4500 * time.Millisecond},
		{"the browser and its X server", []int{10, 20}, 4650 * time.Millisecond},
		{"a group with no process", []int{77}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := groupCPUTime(dir, tc.groups)
			if err != nil || got != tc.want {
				t.Errorf("groupCPUTime = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
	_, err := groupCPUTime(filepath.Join(dir, "missing"), []int{10})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "BrowserCPUUnavailable" || !appErr.HasRoute("groupCPUTime.BrowserCPUUnavailable") {
		t.Errorf("a missing /proc gave %v", err)
	}
}
