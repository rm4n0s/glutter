//go:build glutter_android

package androidtest

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

// a11yApp installs examples/dev/android-a11y, which logs the button, the list and the text field, and
// starts it with the renderer soft, the accessibility bridge forced on (GLUTTER_A11Y; the images have no
// TalkBack) and the dev events script. It returns once the app has presented its first frame.
func a11yApp(ctx context.Context, t *testing.T, e *Emulator, script string) string {
	t.Helper()
	appID := installApp(ctx, t, e, "./examples/dev/android-a11y")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	extras := map[string]string{"GLUTTER_RENDERER": "soft", "GLUTTER_A11Y": "1"}
	if script != "" {
		extras["GLUTTER_DEV_EVENTS"] = script
	}
	if err := e.Launch(ctx, appID, extras); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

// uiNode is a node of the dump of uiautomator.
type uiNode struct {
	Class      string   `xml:"class,attr"`
	Text       string   `xml:"text,attr"`
	Desc       string   `xml:"content-desc,attr"`
	Bounds     string   `xml:"bounds,attr"`
	Clickable  string   `xml:"clickable,attr"`
	Scrollable string   `xml:"scrollable,attr"`
	Enabled    string   `xml:"enabled,attr"`
	Children   []uiNode `xml:"node"`
}

// flat lists the node and all below it.
func (n uiNode) flat() []uiNode {
	out := []uiNode{n}
	for _, c := range n.Children {
		out = append(out, c.flat()...)
	}
	return out
}

// uiDump runs uiautomator dump on the device until the app's nodes are in it, and returns them.
func uiDump(ctx context.Context, t *testing.T, e *Emulator, want string) []uiNode {
	t.Helper()
	var last string
	for end := time.Now().Add(90 * time.Second); ; time.Sleep(2 * time.Second) {
		out, err := e.ADB.Output(ctx, "shell", "uiautomator", "dump", "/sdcard/glutter-ui.xml")
		if err == nil {
			var b string
			b, err = e.ADB.Output(ctx, "shell", "cat", "/sdcard/glutter-ui.xml")
			if err == nil {
				last = b
				var root struct {
					Nodes []uiNode `xml:"node"`
				}
				if xml.Unmarshal([]byte(b), &root) == nil {
					var all []uiNode
					for _, n := range root.Nodes {
						all = append(all, n.flat()...)
					}
					for _, n := range all {
						if strings.Contains(n.Text, want) || strings.Contains(n.Desc, want) {
							return all
						}
					}
				}
			}
		}
		_ = out
		if time.Now().After(end) {
			t.Fatalf("the dump of uiautomator never had %q; the last was:\n%s", want, last)
		}
	}
}

// find is the first node whose text or description contains label.
func find(nodes []uiNode, label string) (uiNode, bool) {
	for _, n := range nodes {
		if strings.Contains(n.Text, label) || strings.Contains(n.Desc, label) {
			return n, true
		}
	}
	return uiNode{}, false
}

// TestTheSystemSeesTheNodesOfTheApp has uiautomator, which reads the screen through the accessibility
// service as TalkBack does, list the button, the live text, the text field (an EditText; uiautomator does not dump its hint) and
// the list with their labels and bounds.
func TestTheSystemSeesTheNodesOfTheApp(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		_ = a11yApp(ctx, t, e, "")
		nodes := uiDump(ctx, t, e, "Increment")
		for _, want := range []string{"Increment", "Count: 0", "Items", "Item 0"} {
			n, ok := find(nodes, want)
			if !ok {
				t.Errorf("the dump has no node with %q", want)
				continue
			}
			if n.Bounds == "" || n.Bounds == "[0,0][0,0]" {
				t.Errorf("%q has the bounds %q", want, n.Bounds)
			}
		}
		if n, ok := find(nodes, "Increment"); ok && (n.Clickable != "true" || n.Enabled != "true") {
			t.Errorf("the button is not clickable and enabled: %+v", n)
		}
		// The list is the scrollable node below the labeled container.
		list := false
		for _, n := range nodes {
			if strings.Contains(n.Class, "ScrollView") && n.Scrollable == "true" {
				list = true
			}
		}
		if !list {
			t.Errorf("the dump has no scrollable list")
		}
		field := false
		for _, n := range nodes {
			if strings.Contains(n.Class, "EditText") {
				field = true
			}
		}
		if !field {
			t.Errorf("the dump has no text field (EditText): %+v", nodes)
		}
	})
}

// TestAClickActionPressesTheButton has the dev events do what TalkBack does on a double tap: the click
// action of the accessibility node of the button.
func TestAClickActionPressesTheButton(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := a11yApp(ctx, t, e, "wait:4s;a11y:click,Increment")
		if _, err := e.WaitLog(ctx, appID, "button pressed count=1", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
	})
}

// TestAScrollActionScrollsTheList has the accessibility scroll forward action run on the list.
func TestAScrollActionScrollsTheList(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := a11yApp(ctx, t, e, "wait:4s;a11y:scrollforward,Items")
		if _, err := e.WaitLog(ctx, appID, "list scrolled offset=", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
	})
}

// TestAChangeOfALiveRegionIsAnnounced has the button pressed by its accessibility action, which changes
// the text of the live region "Count: N": the bridge sends the accessibility event of the live region,
// which the system's services speak.
func TestAChangeOfALiveRegionIsAnnounced(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := a11yApp(ctx, t, e, "wait:4s;a11y:click,Increment")
		if _, err := e.WaitLog(ctx, appID, "button pressed count=1", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		for end := time.Now().Add(30 * time.Second); ; time.Sleep(500 * time.Millisecond) {
			log, err := e.FullLog(ctx)
			if err != nil {
				t.Fatal(Describe(err))
			}
			for _, l := range lines(log, "a11y event") {
				if strings.Contains(l, "Count: 1") {
					return
				}
			}
			if time.Now().After(end) {
				t.Fatalf("no accessibility event for the live region:\n%s", strings.Join(lines(log, "a11y "), "\n"))
			}
		}
	})
}
