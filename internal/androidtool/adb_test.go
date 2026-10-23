package androidtool

import (
	"context"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

type scriptRunner struct {
	out  map[string]string // the output by the arguments joined with spaces
	cmds []Command
}

func (s *scriptRunner) Run(_ context.Context, c Command) (string, error) {
	s.cmds = append(s.cmds, c)
	key := strings.Join(c.Args, " ")
	for k, v := range s.out {
		if strings.Contains(key, k) {
			return v, nil
		}
	}
	return "", nil
}

func TestParseDevices(t *testing.T) {
	out := "List of devices attached\nemulator-5554\tdevice\nR58M\toffline\n* daemon started *\n\n"
	got := ParseDevices(out)
	if len(got) != 2 || got[0] != (Device{"R58M", "offline"}) || got[1] != (Device{"emulator-5554", "device"}) {
		t.Errorf("devices %+v", got)
	}
}

func TestPickDevice(t *testing.T) {
	one := []Device{{"a", "device"}}
	two := []Device{{"a", "device"}, {"b", "device"}}
	tests := []struct {
		name    string
		devs    []Device
		want    string
		wantTag string
		serial  string
	}{
		{name: "the only one", devs: one, serial: "a"},
		{name: "named", devs: two, want: "b", serial: "b"},
		{name: "none", wantTag: TagAndroidNoDevice},
		{name: "only an offline one", devs: []Device{{"a", "offline"}}, wantTag: TagAndroidNoDevice},
		{name: "two", devs: two, wantTag: TagAndroidManyDevices},
		{name: "named but absent", devs: one, want: "z", wantTag: TagAndroidDeviceNotFound},
		{name: "named but unauthorized", devs: []Device{{"a", "unauthorized"}}, want: "a", wantTag: TagAndroidDeviceNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := PickDevice(tt.devs, tt.want)
			if tt.wantTag == "" {
				if err != nil || d.Serial != tt.serial {
					t.Fatalf("got %v %v", d, err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("PickDevice."+tt.wantTag) {
				t.Fatalf("got %v", err)
			}
			if tt.wantTag == TagAndroidManyDevices && !strings.Contains(appErr.Metadata["devices"].(string), "a (device), b (device)") {
				t.Errorf("devices metadata %v", appErr.Metadata)
			}
		})
	}
}

func TestADBCommands(t *testing.T) {
	r := &scriptRunner{out: map[string]string{"abilist": "x86_64,arm64-v8a\n", "pidof": "1234\n"}}
	a := ADB{Path: "/p/adb", Serial: "emu", Env: []string{"ANDROID_ADB_SERVER_PORT=1"}, Run: r}
	abi, err := a.ABI(context.Background())
	if err != nil || abi.Name != "x86_64" {
		t.Fatalf("ABI %v %v", abi, err)
	}
	if err := a.Install(context.Background(), "app.apk"); err != nil {
		t.Fatal(err)
	}
	if err := a.Start(context.Background(), "org.x.y", map[string]string{"GLUTTER_B": "it's", "GLUTTER_A": "a b"}); err != nil {
		t.Fatal(err)
	}
	pid, err := a.Pid(context.Background(), "org.x.y")
	if err != nil || pid != 1234 {
		t.Fatalf("pid %d %v", pid, err)
	}
	var lines []string
	for _, c := range r.cmds {
		if c.Name != "/p/adb" || len(c.Env) != 1 {
			t.Errorf("command %+v", c)
		}
		lines = append(lines, strings.Join(c.Args, " "))
	}
	want := []string{
		"-s emu shell getprop ro.product.cpu.abilist",
		"-s emu install -r -t app.apk",
		"-s emu shell am force-stop org.x.y",
		"-s emu shell am start -W -n org.x.y/io.github.rm4n0s.glutter.GlutterActivity --es 'GLUTTER_A' 'a b' --es 'GLUTTER_B' 'it'\\''s'",
		"-s emu shell pidof -s org.x.y",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("commands:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestADBFailures(t *testing.T) {
	r := &scriptRunner{out: map[string]string{"abilist": "armeabi-v7a\n", "am start": "Error: Activity class does not exist.\n"}}
	a := ADB{Run: r}
	_, err := a.ABI(context.Background())
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("ADB.ABI."+TagAndroidABIUnsupported) {
		t.Errorf("ABI: %v", err)
	}
	err = a.Start(context.Background(), "org.x.y", nil)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("ADB.Start."+TagAndroidStartFailed) {
		t.Errorf("Start: %v", err)
	}
}

func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{"a": "'a'", "a b": "'a b'", "it's": `'it'\''s'`, "": "''"} {
		if got := ShellQuote(in); got != want {
			t.Errorf("ShellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

// failingRunner fails every command.
type failingRunner struct{}

func (failingRunner) Run(_ context.Context, c Command) (string, error) {
	return "", errors.New(TagAndroidCommandFailed, "failed", "command", c.String())
}

func TestPullAndScreencap(t *testing.T) {
	ctx := context.Background()
	const pngHeader = "\x89PNG\r\n\x1a\nrest"
	a := ADB{Serial: "emu", Run: &scriptRunner{out: map[string]string{"cat files/a.png": pngHeader, "screencap": pngHeader}}}
	b, err := a.PullAppFile(ctx, "org.x.y", "files/a.png")
	if err != nil || string(b) != pngHeader {
		t.Errorf("PullAppFile = %q, %v", b, err)
	}
	if b, err = a.Screencap(ctx); err != nil || string(b) != pngHeader {
		t.Errorf("Screencap = %q, %v", b, err)
	}
	if err := a.RemoveAppFile(ctx, "org.x.y", "files/a.png"); err != nil {
		t.Errorf("RemoveAppFile: %v", err)
	}

	// exec-out passes the output of a failing cat on as if it were the file.
	missing := ADB{Run: &scriptRunner{out: map[string]string{"cat": "cat: files/a.png: No such file or directory\n"}}}
	if _, err := missing.PullAppFile(ctx, "org.x.y", "files/a.png"); !hasRoute(err, "ADB.PullAppFile."+TagAndroidPullFailed) {
		t.Errorf("a missing file: %v", err)
	}
	if _, err := (ADB{Run: failingRunner{}}).PullAppFile(ctx, "org.x.y", "files/a.png"); !hasRoute(err, "ADB.PullAppFile->ADB.Output->failingRunner.Run."+TagAndroidCommandFailed) {
		t.Errorf("a failing adb: %v", err)
	}
	if err := (ADB{Run: failingRunner{}}).RemoveAppFile(ctx, "org.x.y", "files/a.png"); err != nil {
		t.Errorf("RemoveAppFile must not fail for a missing directory: %v", err)
	}
	notPNG := ADB{Run: &scriptRunner{out: map[string]string{"screencap": "error: no display\n"}}}
	if _, err := notPNG.Screencap(ctx); !hasRoute(err, "ADB.Screencap."+TagAndroidPullFailed) {
		t.Errorf("a screencap that is not a PNG: %v", err)
	}
	if _, err := (ADB{Run: failingRunner{}}).Screencap(ctx); !hasRoute(err, "ADB.Output->failingRunner.Run."+TagAndroidCommandFailed) {
		t.Errorf("a failing screencap: %v", err)
	}
}

func hasRoute(err error, route string) bool {
	appErr, ok := errors.FromError(err)
	return ok && appErr.HasRoute(route)
}
