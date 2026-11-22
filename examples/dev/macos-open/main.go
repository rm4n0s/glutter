// Command macos-open shows what the system asks the app to open (step 56.1): the links of a URL scheme the app
// handles and the files of a type it opens, which macOS sends to the app as events and not as arguments
// (ui.OpenRequest, PlatformDispatcher.AddOpenRequestListener). The window lists them, with the arguments of the
// program, which is where the other systems put them.
//
// The app must be a bundle whose Info.plist names the scheme and the file type, so build it with glutter-macos:
//
//	go run ./cmd/glutter-macos build -o /tmp/Open.app -name Open -url-scheme glutter-open \
//	    -doc-type "Glutter open test=glutteropen" ./examples/dev/macos-open
//	open /tmp/Open.app                       # registers the bundle with Launch Services and starts it
//	open "glutter-open://hello?x=1"          # the link reaches the running app
//	touch /tmp/a.glutteropen && open /tmp/a.glutteropen   # so does the file, also when the app is not running
//
// Environment, in debug builds (-debug): GLUTTER_DEV_OPEN_LOG=<file> appends a line to the file for the start
// ("ready <pid>"), for each URL ("url <url>") and for each file ("file <path>"), which the test of the step reads.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	err := m.RunApp(NewOpenApp(), m.RunAppOpts{
		AppID:  "dev.glutter.macos-open",
		Window: m.WindowConfig{Title: "Open requests", Size: m.Size{Width: 640, Height: 400}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// OpenApp is the app.
type OpenApp struct{ m.StatelessWidget }

// NewOpenApp creates the app.
func NewOpenApp() *OpenApp { return &OpenApp{} }

// Build builds the app.
func (a *OpenApp) Build(m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Open requests",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewOpenList(),
	})
}

// OpenList lists what the app was asked to open.
type OpenList struct{ m.StatefulWidget }

// NewOpenList creates the list.
func NewOpenList() *OpenList { return &OpenList{StatefulWidget: m.NewStatefulWidgetBase()} }

// CreateState creates the state.
func (*OpenList) CreateState() m.State { return &openListState{} }

type openListState struct {
	m.StateOf[*OpenList]
	lines  []string
	remove func()
}

// InitState listens for the open requests. The ones that started the app waited for this listener.
func (s *openListState) InitState() {
	devLog(fmt.Sprintf("ready %d", os.Getpid()))
	d := m.WidgetsBindingOf(s.Context()).PlatformDispatcher()
	s.remove = d.AddOpenRequestListener(func(r ui.OpenRequest) {
		s.SetState(func() {
			for _, u := range r.URLs {
				s.lines = append(s.lines, "url "+u)
				devLog("url " + u)
			}
			for _, f := range r.Files {
				s.lines = append(s.lines, "file "+f)
				devLog("file " + f)
			}
		})
	})
}

// Dispose stops listening.
func (s *openListState) Dispose() { s.remove() }

func (s *openListState) Build(m.BuildContext) m.Widget {
	rows := []m.Widget{
		m.NewText(fmt.Sprintf("Arguments: %s", strings.Join(os.Args[1:], " "))),
		m.NewText(fmt.Sprintf("Asked to open: %d", len(s.lines))),
	}
	for _, l := range s.lines {
		rows = append(rows, m.NewText(l))
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Open requests")}),
		Body:   m.NewPadding(m.NewEdgeInsetsAll(16), m.NewColumn(rows, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})),
	})
}

// devLog appends a line to the file of GLUTTER_DEV_OPEN_LOG, in a debug build.
func devLog(line string) {
	path := os.Getenv("GLUTTER_DEV_OPEN_LOG")
	if !foundation.DebugMode || path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Print(err)
		return
	}
	if _, err := fmt.Fprintln(f, line); err != nil {
		log.Print(err)
	}
	if err := f.Close(); err != nil {
		log.Print(err)
	}
}
