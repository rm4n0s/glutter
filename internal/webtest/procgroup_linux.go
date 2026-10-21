//go:build linux

package webtest

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rm4n0s/errors"
)

// setProcessGroup puts the command in a process group of its own, so that the browser and
// every process it starts can be killed together (CLAUDE.md "Don't leave processes
// running").
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup kills the process group that setProcessGroup made, whose ID is the command's.
func killGroup(pid int) {
	// The group may be gone already: there is nothing to report if it is.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

// privateDisplay is an X server (Xvfb) that Launch starts for a browser that cannot get a GPU
// context without one.
type privateDisplay struct {
	cmd  *exec.Cmd
	name string // the value of DISPLAY
	done chan struct{}
}

// startPrivateDisplay starts an Xvfb on a display number of its own (-displayfd, so that it never
// meets the user's display or another test's), under `timeout -k` and in a process group of its
// own. It returns nil, with no error, when Xvfb is not installed: the browser then runs with no
// display, and a page that asks for WebGL gets none. It fails with BrowserLaunchFailed when the
// server does not start or does not say which display it took.
func startPrivateDisplay(lifetime time.Duration) (*privateDisplay, error) {
	xvfb, err := exec.LookPath("Xvfb")
	if err != nil {
		return nil, nil
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, errors.NewErr("BrowserLaunchFailed", err).SetMetadata("what", "the pipe of the private display")
	}
	defer func() { _ = r.Close() }() // the read end is only for the display's number
	secs := strconv.Itoa(max(1, int(lifetime.Seconds())))
	cmd := exec.Command("timeout", "-k", "5s", secs+"s", xvfb, "-displayfd", "3", "-screen", "0", "1280x1024x24", "-nolisten", "tcp")
	cmd.ExtraFiles = []*os.File{w} // file descriptor 3
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	setProcessGroup(cmd)
	err = cmd.Start()
	_ = w.Close() // the server has its copy
	if err != nil {
		return nil, errors.NewErr("BrowserLaunchFailed", err).SetMetadata("what", "Xvfb", "path", xvfb)
	}
	d := &privateDisplay{cmd: cmd, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait() // it ends when killed or when its time is up
		close(d.done)
	}()
	line := make(chan string, 1)
	go func() {
		s, _ := bufio.NewReader(r).ReadString('\n') // empty when the server ends without saying
		line <- strings.TrimSpace(s)
	}()
	select {
	case n := <-line:
		if n != "" {
			d.name = ":" + n
			return d, nil
		}
	case <-time.After(10 * time.Second):
	}
	d.stop()
	return nil, errors.New("BrowserLaunchFailed", "Xvfb did not say which display it took", "what", "Xvfb", "path", xvfb)
}

// stop kills the server and its group and waits for it.
func (d *privateDisplay) stop() {
	killGroup(d.cmd.Process.Pid)
	<-d.done
}
