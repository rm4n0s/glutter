//go:build windows && glutter_wine

package windows

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests use the real shell calls of Wine for the desktop services (SPEC §25.9, §25.13). The
// tray needs a shell: Wine's explorer on a virtual desktop of 800 by 600 pixels, named glutter
// (explorer /desktop=glutter,800x600), which the first test that needs it starts and the thread of
// each test joins (SetThreadDesktop). Wine's X driver of a process cannot have threads on two desktops
// (a window made later on the first one fails with BadWindow and ends the process), so the tests that
// join the shell run in a process of their own: TestWineShellServices starts this test binary again
// with GLUTTER_SHELL_CHILD set, which makes the tests named TestWineShell... run, and checks that each
// of them passed. The child's output reaches the parent, which the program started through explorer
// would not do. The shell stops when the child ends (TestMain).

// shellChildEnv is set in the process that runs the TestWineShell... tests.
const shellChildEnv = "GLUTTER_SHELL_CHILD"

var (
	shellOnce sync.Once
	shellErr  error
	shellCmd  *exec.Cmd
)

// startShell starts Wine's explorer on the virtual desktop once.
func startShell() error {
	shellOnce.Do(func() {
		cmd := exec.Command(`C:\windows\explorer.exe`, "/desktop=glutter,800x600")
		if err := cmd.Start(); err != nil {
			shellErr = errors.NewErr(tagExplorerStartFailed, err)
			return
		}
		shellCmd = cmd
	})
	return shellErr
}

// TestMain stops the shell that a test started, when the tests of the process are done.
func TestMain(m *testing.M) {
	code := m.Run()
	if shellCmd != nil && shellCmd.Process != nil {
		_ = shellCmd.Process.Kill() // the shell of a test; nothing is left to tell
	}
	os.Exit(code)
}

// inShellChild skips the test unless it runs in the process that TestWineShellServices starts.
func inShellChild(t *testing.T) {
	t.Helper()
	if os.Getenv(shellChildEnv) == "" {
		t.Skip("run by TestWineShellServices, in a process of its own")
	}
}

// shellTests are the tests that join the shell's desktop.
var shellTests = []string{"TestWineShellTrayIconAddModifyDeleteAndClick", "TestWineShellTaskbarCreatedAddsTheIconAgain", "TestWineShellBalloonIsAcceptedAndItsClickCallsTheAction"}

func TestWineShellServices(t *testing.T) {
	if os.Getenv(shellChildEnv) != "" {
		t.Skip("this is the child process")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^("+strings.Join(shellTests, "|")+")$", "-test.v", "-test.timeout=120s")
	cmd.Env = append(os.Environ(), shellChildEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the process of the shell tests failed: %v\n%s", err, out)
	}
	for _, name := range shellTests {
		if !strings.Contains(string(out), "--- PASS: "+name+" ") {
			t.Errorf("%s did not pass in the child process:\n%s", name, out)
		}
	}
	t.Logf("the shell tests passed in a process of their own:\n%s", out)
}

// joinShell makes the calling thread, which must have no window yet, a thread of the shell's desktop and
// waits for the task bar to exist there. A test that needs the tray calls it before it makes the platform.
func joinShell(t testing.TB) {
	t.Helper()
	if !winetestUnderWine() {
		t.Skip("the shell tests need Wine")
	}
	if err := startShell(); err != nil {
		t.Fatal(err)
	}
	user, err := win32.LoadSystemDLL("user32.dll")
	if err != nil {
		t.Fatal(err)
	}
	openDesktop, err := user.Proc("OpenDesktopW")
	if err != nil {
		t.Fatal(err)
	}
	setThreadDesktop, err := user.Proc("SetThreadDesktop")
	if err != nil {
		t.Fatal(err)
	}
	name, _ := win32.UTF16PtrFromString("glutter")
	deadline := time.Now().Add(20 * time.Second)
	for {
		hdesk, _ := openDesktop.Call(uintptr(unsafe.Pointer(name)), 0, 0, 0x1FF)
		if hdesk != 0 {
			if r, _ := setThreadDesktop.Call(hdesk); r != 0 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("the desktop glutter of Wine's explorer did not appear")
		}
		time.Sleep(100 * time.Millisecond)
	}
	for win32.FindWindow("Shell_TrayWnd", "") == 0 {
		if time.Now().After(deadline) {
			t.Fatal("Wine's explorer made no Shell_TrayWnd")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// winetestUnderWine says whether the program runs under Wine: ntdll.dll of Wine exports wine_get_version.
func winetestUnderWine() bool {
	lib, err := win32.LoadSystemDLL("ntdll.dll")
	if err != nil {
		return false
	}
	_, ok := lib.OptionalProc("wine_get_version")
	return ok
}

// newShellPlatform makes the platform on a thread of the shell's desktop. The thread is never unlocked:
// a goroutine that ends while it is locked ends its thread too, so no other test gets a thread that
// belongs to the virtual desktop.
func newShellPlatform(t *testing.T) (*Platform, *rec) {
	t.Helper()
	runtime.LockOSThread()
	joinShell(t)
	p, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	return p, &rec{p: p, closeAnswer: true}
}

// recordingShell is the real shellAPI, with a record of the notify calls.
type recordingShell struct {
	winShell
	mu       sync.Mutex
	commands []uint32
	results  []error
	executed []string
}

func (r *recordingShell) Notify(command uint32, d iconData) error {
	err := r.winShell.Notify(command, d)
	r.mu.Lock()
	r.commands = append(r.commands, command)
	r.results = append(r.results, err)
	r.mu.Unlock()
	return err
}

// lastResult is the result of the last call of the command, and whether there was one.
func (r *recordingShell) lastResult(command uint32) (error, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.commands) - 1; i >= 0; i-- {
		if r.commands[i] == command {
			return r.results[i], true
		}
	}
	return nil, false
}

func (r *recordingShell) ShellExecute(uri string) error {
	r.mu.Lock()
	r.executed = append(r.executed, uri)
	r.mu.Unlock()
	return r.winShell.ShellExecute(uri)
}

func (r *recordingShell) count(command uint32) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.commands {
		if c == command {
			n++
		}
	}
	return n
}

// useRecording puts the recording shell under the platform's Desktop and returns it.
func useRecording(p *Platform) (*Desktop, *recordingShell) {
	rs := &recordingShell{winShell: winShell{p}}
	taskbar, _ := win32.RegisterWindowMessage("TaskbarCreated")
	p.deskOnce.Do(func() { p.desk.Store(newDesktop(rs, p.log, taskbar)) })
	return p.desk.Load(), rs
}

// shellSends posts the callback message of an icon to the service window, as the shell does with
// NOTIFYICON_VERSION_4: the event in the low word of lParam, the ID of the icon in the high word, and the
// anchor point in wParam.
func shellSends(t testing.TB, p *Platform, id, event uint32, at point) {
	t.Helper()
	wparam := uintptr(uint16(at.X)) | uintptr(uint16(at.Y))<<16
	if err := win32.PostMessage(p.service.Load(), msgTray, wparam, uintptr(event)|uintptr(id)<<16); err != nil {
		t.Fatal(err)
	}
}

func TestWineShellTrayIconAddModifyDeleteAndClick(t *testing.T) {
	inShellChild(t)
	p, r := newShellPlatform(t)
	d, rs := useRecording(p)
	var activated, secondary int
	spec := platform.TraySpec{
		Tooltip: "Glutter tray", Icon: rgba(16, 200),
		Menu:       []platform.TrayMenuItem{{Label: "Open"}},
		OnActivate: func() { activated++ }, OnSecondaryActivate: func() { secondary++ },
	}
	runFor(t, p, r, 40*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		tr, err := d.NewTray(spec)
		if err != nil {
			t.Fatalf("NewTray: %v", err)
		}
		if rs.count(win32.NIMAdd) != 1 || rs.count(win32.NIMSetVersion) != 1 {
			t.Errorf("the shell calls were %v: one NIM_ADD and one NIM_SETVERSION", rs.commands)
		}
		// NIM_MODIFY for the tooltip and for the icon: the real shell accepts both.
		changed := spec
		changed.Tooltip = "Glutter tray, changed"
		tr.Update(changed)
		changed.Icon = rgba(16, 100)
		tr.Update(changed)
		if rs.count(win32.NIMModify) != 2 {
			t.Errorf("NIM_MODIFY was sent %d times, want 2 (%v)", rs.count(win32.NIMModify), rs.commands)
		}
		// The clicks the shell would send: NIN_SELECT and the middle button, with version 4.
		shellSends(t, p, trayIconID, win32.NINSelect, point{10, 20})
		shellSends(t, p, trayIconID, win32.WMMButtonUp, point{10, 20})
		deadline := time.Now().Add(10 * time.Second)
		for activated == 0 || secondary == 0 {
			if time.Now().After(deadline) {
				t.Fatalf("the clicks did not reach the callbacks: activate %d, secondary %d", activated, secondary)
			}
			p.turn()
			time.Sleep(5 * time.Millisecond)
		}
		tr.Close()
		if rs.count(win32.NIMDelete) != 1 {
			t.Errorf("NIM_DELETE was sent %d times", rs.count(win32.NIMDelete))
		}
		// The shell has forgotten the icon: it refuses to change it.
		if err := (winShell{p}).Notify(win32.NIMModify, iconData{ID: trayIconID, Flags: win32.NIFTip, Tip: "x"}); err == nil {
			t.Log("Wine's shell accepts NIM_MODIFY for an icon that was deleted")
		}
	})
}

func TestWineShellTaskbarCreatedAddsTheIconAgain(t *testing.T) {
	inShellChild(t)
	p, r := newShellPlatform(t)
	d, rs := useRecording(p)
	runFor(t, p, r, 40*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		tr, err := d.NewTray(platform.TraySpec{Icon: rgba(16, 50)})
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		// Explorer restarted: its new task bar has no icon. The real shell refuses a NIM_ADD for an icon it
		// still has, so the icon is taken away first, behind the platform's back.
		if err := (winShell{p}).Notify(win32.NIMDelete, iconData{ID: trayIconID}); err != nil {
			t.Fatalf("NIM_DELETE: %v", err)
		}
		adds := rs.count(win32.NIMAdd)
		msg, err := win32.RegisterWindowMessage("TaskbarCreated")
		if err != nil {
			t.Fatal(err)
		}
		if err := win32.PostMessage(p.service.Load(), msg, 0, 0); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for rs.count(win32.NIMAdd) == adds {
			if time.Now().After(deadline) {
				t.Fatal("TaskbarCreated did not add the icon again")
			}
			p.turn()
			time.Sleep(5 * time.Millisecond)
		}
		if err, _ := rs.lastResult(win32.NIMAdd); err != nil {
			t.Errorf("the shell refused the icon that TaskbarCreated added again: %v", err)
		}
		// The icon works: the shell takes a change of it.
		tr.Update(platform.TraySpec{Icon: rgba(16, 50), Tooltip: "back"})
		if err, _ := rs.lastResult(win32.NIMModify); err != nil {
			t.Errorf("the shell refused a change of the icon that came back: %v", err)
		}
	})
}

// menuItemOf reads the item at a position of a menu with GetMenuItemInfoW.
type readMenuItem struct {
	Text      string
	Separator bool
	Disabled  bool
	Checked   bool
	Radio     bool
	ID        uint32
	Sub       []readMenuItem
}

func readMenu(t testing.TB, menu uintptr) []readMenuItem {
	t.Helper()
	n := win32.GetMenuItemCount(menu)
	if n < 0 {
		t.Fatalf("GetMenuItemCount(%#x) = %d", menu, n)
	}
	var out []readMenuItem
	for i := range n {
		buf := make([]uint16, 256)
		info := win32.MenuItemInfo{Mask: win32.MIIMFType | win32.MIIMID | win32.MIIMState | win32.MIIMSubMenu | win32.MIIMString,
			TypeData: unsafe.Pointer(&buf[0]), CCH: uint32(len(buf))}
		if err := win32.GetMenuItemInfo(menu, uint32(i), &info); err != nil {
			t.Fatalf("GetMenuItemInfo(%d): %v", i, err)
		}
		it := readMenuItem{
			Text: win32.UTF16ToString(buf[:info.CCH]), Separator: info.Type&win32.MFTSeparator != 0, Disabled: info.State&win32.MFSGrayed != 0,
			Checked: info.State&win32.MFSChecked != 0, Radio: info.Type&win32.MFTRadioCheck != 0, ID: info.ID,
		}
		if info.SubMenu != 0 {
			it.Sub = readMenu(t, info.SubMenu)
		}
		out = append(out, it)
	}
	return out
}

func TestWineMenuHasItsItemsInOrder(t *testing.T) {
	items := []platform.TrayMenuItem{
		{Label: "Show window"},
		{Separator: true},
		{Label: "Save & go", Disabled: true},
		{Label: "Dark mode", Checkable: true, Checked: true},
		{Label: "Mode A", Radio: true, Checked: true},
		{Label: "Hidden", Hidden: true},
		{Label: "More", Children: []platform.TrayMenuItem{{Label: "One"}, {Separator: true}, {Label: "Two", Checkable: true}}},
		{Label: "Quit"},
	}
	plan := planMenu(items)
	menu, err := buildMenu(plan.Entries)
	if err != nil {
		t.Fatal(err)
	}
	defer win32.DestroyMenu(menu)
	got := readMenu(t, menu)
	want := []readMenuItem{
		{Text: "Show window", ID: 1},
		{Separator: true, ID: 2},
		{Text: "Save && go", Disabled: true, ID: 3}, // the label has its & doubled, which Windows shows as one
		{Text: "Dark mode", Checked: true, ID: 4},
		{Text: "Mode A", Checked: true, Radio: true, ID: 5},
		{Text: "More", ID: 6, Sub: []readMenuItem{{Text: "One", ID: 7}, {Separator: true, ID: 8}, {Text: "Two", ID: 9}}},
		{Text: "Quit", ID: 10},
	}
	// The text of a separator is not read.
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the menu =\n%+v\nwant\n%+v", got, want)
	}
}

func TestWineShellBalloonIsAcceptedAndItsClickCallsTheAction(t *testing.T) {
	inShellChild(t)
	p, r := newShellPlatform(t)
	d, rs := useRecording(p)
	var mu sync.Mutex
	var actions []string
	var closed []mobile.CloseReason
	var firstID uint32
	ctx := context.Background()
	runFor(t, p, r, 40*time.Second, func(cancel context.CancelFunc) {
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			id, err := d.ShowNotification(ctx, mobile.NotificationSpec{
				Title: "Build finished", Body: "All 12 tests passed", Urgency: mobile.UrgencyLow,
				Actions:  []mobile.Action{{ID: mobile.DefaultAction, Label: "Open"}},
				OnAction: func(a string) { mu.Lock(); actions = append(actions, a); mu.Unlock() },
				OnClose:  func(c mobile.CloseReason) { mu.Lock(); closed = append(closed, c); mu.Unlock() },
			})
			if err != nil {
				t.Errorf("ShowNotification: %v", err)
				return
			}
			firstID = id
		}()
		// Wait for the call, running the loop's turns on this (the UI) thread.
		deadline := time.Now().Add(15 * time.Second)
		for {
			p.turn()
			select {
			case <-finished:
			default:
				if time.Now().After(deadline) {
					t.Error("ShowNotification did not return")
					cancel()
					return
				}
				time.Sleep(5 * time.Millisecond)
				continue
			}
			break
		}
		if firstID == 0 {
			cancel()
			return
		}
		if rs.count(win32.NIMModify) == 0 || rs.count(win32.NIMAdd) == 0 {
			t.Errorf("the shell calls were %v: the hidden icon and the balloon are NIM_ADD and NIM_MODIFY", rs.commands)
		}
		shellSends(t, p, balloonIconID, win32.NINBalloonUserClick, point{})
		deadline = time.Now().Add(10 * time.Second)
		for {
			p.turn()
			mu.Lock()
			n := len(actions)
			mu.Unlock()
			if n > 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Error("the click did not call OnAction")
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	})
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(actions, []string{"default"}) || !reflect.DeepEqual(closed, []mobile.CloseReason{mobile.CloseDismissed}) {
		t.Errorf("actions %v, closes %v", actions, closed)
	}
}

// waitForDialog finds the dialog with the title among the top-level windows of the desktop and returns
// its handle.
func waitForDialog(t testing.TB, p *Platform, title string, limit time.Duration) uintptr {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		p.turn()
		if h := win32.FindWindow("#32770", title); h != 0 {
			return h
		}
		if time.Now().After(deadline) {
			t.Fatalf("no dialog titled %q appeared", title)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

type dialogOutcome struct {
	res platform.FileDialogResult
	err error
}

// The dialogs, the worker threads and the programs of a scheme are on the process's own desktop, not on
// the shell's virtual one that joinShell gives the UI thread, so these tests use newTestPlatform.

func TestWineFileDialogsAreShownOnAWorkerAndClosedByTheTest(t *testing.T) {
	tests := []struct {
		name string
		spec platform.FileDialogSpec
	}{
		{"open one file with filters", platform.FileDialogSpec{Kind: platform.FileDialogOpen,
			Filters: []platform.FileFilter{{Name: "Images", Patterns: []string{"*.png", "*.jpg"}}, {Name: "All files"}}}},
		{"open several files", platform.FileDialogSpec{Kind: platform.FileDialogOpen, Multiple: true, AcceptLabel: "Choose"}},
		{"open a folder", platform.FileDialogSpec{Kind: platform.FileDialogOpen, Directory: true, Folder: `C:\windows`}},
		{"save a file", platform.FileDialogSpec{Kind: platform.FileDialogSave, FileName: "report.png",
			Filters: []platform.FileFilter{{Name: "PNG", Patterns: []string{"*.png"}}}}},
		{"save several files", platform.FileDialogSpec{Kind: platform.FileDialogSaveMany, Names: []string{"a.txt", "b.txt"}}},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, r := newTestPlatform(t)
			d := p.Desktop().(*Desktop)
			spec := tt.spec
			spec.Title = "Glutter dialog " + string(rune('A'+i))
			runFor(t, p, r, 60*time.Second, func(cancel context.CancelFunc) {
				defer cancel()
				done := make(chan dialogOutcome, 1)
				go func() {
					res, err := d.ShowFileDialog(context.Background(), spec)
					done <- dialogOutcome{res, err}
				}()
				hwnd := waitForDialog(t, p, spec.Title, 30*time.Second)
				if err := win32.PostMessage(hwnd, win32.WMCommand, win32.IDCancel, 0); err != nil {
					t.Fatal(err)
				}
				deadline := time.After(20 * time.Second)
				for {
					p.turn()
					select {
					case out := <-done:
						if out.err != nil {
							t.Fatalf("ShowFileDialog: %v", out.err)
						}
						if !out.res.Cancelled || len(out.res.Paths) != 0 {
							t.Errorf("result = %+v: IDCANCEL is a Cancelled result", out.res)
						}
						return
					case <-deadline:
						t.Fatal("the dialog did not end after IDCANCEL")
					default:
						time.Sleep(10 * time.Millisecond)
					}
				}
			})
		})
	}
}

func TestWineFileDialogClosesWithItsContext(t *testing.T) {
	p, r := newTestPlatform(t)
	d := p.Desktop().(*Desktop)
	runFor(t, p, r, 60*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		ctx, stop := context.WithCancel(context.Background())
		defer stop()
		done := make(chan dialogOutcome, 1)
		go func() {
			res, err := d.ShowFileDialog(ctx, platform.FileDialogSpec{Title: "Glutter dialog ctx"})
			done <- dialogOutcome{res, err}
		}()
		waitForDialog(t, p, "Glutter dialog ctx", 30*time.Second)
		stop()
		deadline := time.After(20 * time.Second)
		for {
			p.turn()
			select {
			case out := <-done:
				assertTagRoute(t, out.err, tagFileChooserFailed, "Desktop.ShowFileDialog."+tagFileChooserFailed)
				return
			case <-deadline:
				t.Fatal("the dialog did not close with its context")
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	})
}

// The custom scheme of the open-URI test and the program that handles it: this test binary again, which
// writes the URI it was given to a file whose name the URI carries.
const openURIScheme = "glutter-test"

func uriFile(nonce string) string {
	return filepath.Join(os.TempDir(), "glutter-openuri-"+nonce+".txt")
}

// TestWineOpenURIHelper is the program that Windows starts for the custom scheme: it writes the URI it
// got to the file named by the URI. It does nothing without such an argument.
func TestWineOpenURIHelper(t *testing.T) {
	var uri string
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, openURIScheme+":") {
			uri = a
		}
	}
	if uri == "" {
		t.Skip("only the program that Windows starts for the custom scheme runs this")
	}
	nonce := uri[strings.LastIndex(uri, "=")+1:]
	if err := os.WriteFile(uriFile(nonce), []byte(uri), 0o600); err != nil {
		t.Fatal(err)
	}
}

// regSetString writes a string value of HKEY_CURRENT_USER (name "" is the default value of the key).
func regSetString(t testing.TB, root uintptr, subKey, name, value string) {
	t.Helper()
	lib, err := win32.LoadSystemDLL("advapi32.dll")
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lib.Proc("RegSetKeyValueW")
	if err != nil {
		t.Fatal(err)
	}
	ps, _ := win32.UTF16PtrFromString(subKey)
	var pn *uint16
	if name != "" {
		pn, _ = win32.UTF16PtrFromString(name)
	}
	data, _ := win32.UTF16FromString(value)
	const regSZ = 1
	if r, _ := proc.Call(root, uintptr(unsafe.Pointer(ps)), uintptr(unsafe.Pointer(pn)), regSZ,
		uintptr(unsafe.Pointer(&data[0])), uintptr(2*len(data))); r != 0 {
		t.Fatalf("RegSetKeyValueW(%s, %s) = %d", subKey, name, r)
	}
}

func regDeleteTree(t testing.TB, root uintptr, subKey string) {
	t.Helper()
	lib, err := win32.LoadSystemDLL("advapi32.dll")
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lib.Proc("RegDeleteTreeW")
	if err != nil {
		t.Fatal(err)
	}
	ps, _ := win32.UTF16PtrFromString(subKey)
	_, _ = proc.Call(root, uintptr(unsafe.Pointer(ps))) // the key may not be there
}

func TestWineOpenURIStartsTheProgramOfACustomScheme(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	classKey := `Software\Classes\` + openURIScheme
	root := uintptr(win32.HKeyLocalMachine)
	regDeleteTree(t, root, classKey)
	t.Cleanup(func() { regDeleteTree(t, root, classKey) })
	regSetString(t, root, classKey, "", "URL:Glutter test")
	regSetString(t, root, classKey, "URL Protocol", "")
	regSetString(t, root, classKey+`\shell\open\command`, "", `"`+exe+`" -test.run=^TestWineOpenURIHelper$ "%1"`)

	p, r := newTestPlatform(t)
	_, rs := useRecording(p)
	d := p.Desktop().(*Desktop)
	const nonce = "n1"
	_ = os.Remove(uriFile(nonce))
	t.Cleanup(func() { _ = os.Remove(uriFile(nonce)) })
	runFor(t, p, r, 60*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		type outcome struct{ err error }
		done := make(chan outcome, 1)
		uri := openURIScheme + "://write?nonce=" + nonce
		go func() { done <- outcome{d.OpenURI(context.Background(), uri)} }()
		deadline := time.Now().Add(40 * time.Second)
		for {
			p.turn()
			if data, err := os.ReadFile(uriFile(nonce)); err == nil {
				if string(data) != uri {
					t.Errorf("the program got %q, want %q", data, uri)
				}
				break
			}
			if time.Now().After(deadline) {
				t.Error("the program of the custom scheme did not write its file")
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		select {
		case o := <-done:
			if o.err != nil {
				meta := map[string]any(nil)
				if ae, ok := errors.FromError(o.err); ok {
					meta = ae.Metadata
				}
				t.Errorf("OpenURI: %v (%v)", o.err, meta)
			}
		case <-time.After(10 * time.Second):
			t.Error("OpenURI did not return")
		}
		if len(rs.executed) != 1 {
			t.Errorf("ShellExecuteExW was called %d times", len(rs.executed))
		}

		// The blocked schemes are refused before ShellExecuteExW is called.
		before := len(rs.executed)
		for _, blocked := range []string{`C:\windows\notepad.exe`, "javascript:alert(1)", "vbscript:x", "file:///C:/windows/notepad.exe", "file:///C:/x/a.bat"} {
			go func() { done <- outcome{d.OpenURI(context.Background(), blocked)} }()
			select {
			case o := <-done:
				assertTagRoute(t, o.err, tagOpenURISchemeBlocked, "Desktop.OpenURI->checkURI."+tagOpenURISchemeBlocked)
			case <-time.After(10 * time.Second):
				t.Errorf("OpenURI(%q) did not return", blocked)
			}
		}
		if len(rs.executed) != before {
			t.Errorf("ShellExecuteExW was called for a blocked URI: %v", rs.executed[before:])
		}

		// A scheme nobody handles is OpenURIUnavailable (or opened by Wine's own handler, which is noted).
		go func() { done <- outcome{d.OpenURI(context.Background(), "glutter-nobody://x")} }()
		select {
		case o := <-done:
			if o.err == nil {
				t.Log("Wine opened a URI that has no handler")
			} else {
				assertTagRoute(t, o.err, tagOpenURIUnavailable, "winShell.ShellExecute."+tagOpenURIUnavailable)
			}
		case <-time.After(20 * time.Second):
			t.Error("OpenURI of an unhandled scheme did not return")
		}
	})
}

// TestWineShellItemPath checks the shell item calls that the dialogs use against Wine: the item of a
// folder (IIDShellItem) gives back the path (IShellItem::GetDisplayName with SIGDN_FILESYSPATH), and a
// folder that does not exist is a failed call.
func TestWineShellItemPath(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := win32.CoInitializeEx(win32.COInitApartmentThreaded); err != nil {
		t.Fatal(err)
	}
	defer win32.CoUninitialize()
	item, err := win32.SHCreateItemFromParsingName(`C:\windows`, &win32.IIDShellItem)
	if err != nil {
		t.Fatal(err)
	}
	path, err := itemPath(item)
	if err != nil || !strings.EqualFold(path, `C:\windows`) {
		t.Errorf("the path of the item = %q, %v", path, err)
	}
	if _, err := win32.SHCreateItemFromParsingName(`C:\no\such\folder\anywhere`, &win32.IIDShellItem); err == nil {
		t.Error("the item of a folder that does not exist was made")
	}
}

func TestWineMakeIcon(t *testing.T) {
	p, _ := newTestPlatform(t)
	s := winShell{p}
	h, owned, err := s.MakeIcon(rgba(32, 90), false)
	if err != nil || h == 0 || !owned {
		t.Fatalf("MakeIcon = %#x, %v, %v", h, owned, err)
	}
	s.FreeIcon(h)
	h, owned, err = s.MakeIcon(ui.WindowIcon{}, false)
	if err != nil || h == 0 || owned {
		t.Errorf("the icon of the program = %#x, owned %v, %v", h, owned, err)
	}
	if _, _, err := s.MakeIcon(ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 16, Scale: 1, RGBA: make([]byte, 3)}}}, false); err == nil {
		t.Error("an icon with a wrong picture was made")
	}
}

// The tag of the error that this file makes (rm4n0s/errors, R5).
const tagExplorerStartFailed = "ExplorerStartFailed"
