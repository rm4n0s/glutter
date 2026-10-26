package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

// procSample is one process of the Wine prefix, as /proc shows it.
type procSample struct {
	PID int
	// Name is the process's name (comm): "counter.exe", "wineserver", "explorer.exe".
	Name string
	// Cmd is its command line, the arguments joined with blanks.
	Cmd string
	// Ticks is utime+stime in clock ticks.
	Ticks uint64
	// RSSKB and HWMKB are VmRSS and VmHWM; PSSKB is Pss from smaps_rollup, when HasPSS.
	RSSKB, HWMKB, PSSKB int
	HasPSS              bool
}

// procSource lists the Wine processes of the prefix; procFS is the real one, a test gives a fake.
type procSource interface {
	Sample() ([]procSample, error)
}

// notWine are the names of the processes that carry the prefix in their environment without being
// Wine: the headless X server and the shells and timeouts of the scripts that start the app. They are
// left out, so that the numbers are those of Wine and the app.
var notWine = map[string]bool{"Xvfb": true, "bash": true, "sh": true, "timeout": true, "xkbcomp": true, "xauth": true}

// procFS reads the Wine processes of a prefix from a /proc directory: the processes whose environment
// has WINEPREFIX=<prefix> (the marker internal/winetest's Leftovers uses), except zombies, this
// process and the notWine ones.
type procFS struct {
	root   string
	marker []byte
	self   int
}

// newProcFS makes the reader of the real /proc for a Wine prefix.
func newProcFS(prefix string) procFS {
	return procFS{root: "/proc", marker: []byte("WINEPREFIX=" + prefix + "\x00"), self: os.Getpid()}
}

// Sample implements procSource. It fails with WindowsPerfProcRead when the /proc directory cannot be
// read (this is not Linux). A process that ends while it is read is left out.
func (p procFS) Sample() ([]procSample, error) {
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return nil, errors.NewErr(tagWindowsPerfProcRead, err).SetMetadata("dir", p.root)
	}
	var out []procSample
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == p.self {
			continue
		}
		dir := filepath.Join(p.root, e.Name())
		env, err := os.ReadFile(filepath.Join(dir, "environ"))
		if err != nil || !bytes.Contains(env, p.marker) {
			continue
		}
		stat, err := os.ReadFile(filepath.Join(dir, "stat"))
		if err != nil || bytes.Contains(stat, []byte(") Z ")) {
			continue
		}
		ticks, ok := perfstats.ProcTicks(string(stat))
		if !ok {
			continue
		}
		comm, _ := os.ReadFile(filepath.Join(dir, "comm")) // gone already: an empty name
		name := strings.TrimSpace(string(comm))
		if notWine[name] {
			continue
		}
		s := procSample{PID: pid, Name: name, Ticks: ticks}
		cmd, _ := os.ReadFile(filepath.Join(dir, "cmdline")) // the same
		s.Cmd = strings.TrimSpace(strings.ReplaceAll(strings.TrimRight(string(cmd), "\x00"), "\x00", " "))
		status, err := os.ReadFile(filepath.Join(dir, "status"))
		if err != nil {
			continue
		}
		rss, hwm, ok := perfstats.ProcMemory(string(status))
		if !ok {
			continue
		}
		s.RSSKB, s.HWMKB = rss, hwm
		if roll, err := os.ReadFile(filepath.Join(dir, "smaps_rollup")); err == nil {
			s.PSSKB, s.HasPSS = perfstats.RollupPSS(string(roll))
		}
		out = append(out, s)
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagWindowsPerfProcRead is the tag of the error for a /proc directory that cannot be read.
	tagWindowsPerfProcRead = "WindowsPerfProcRead"
)
