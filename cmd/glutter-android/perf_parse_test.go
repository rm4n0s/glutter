package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// The samples are the real formats of the tools: `am start -W` of Android 8 and of Android 14,
// `dumpsys meminfo` of Android 8, 11 and 14, /proc/<pid>/stat and /proc/<pid>/status.

const amStartAndroid14 = `Starting: Intent { cmp=org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity }
Status: ok
LaunchState: COLD
Activity: org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity
TotalTime: 523
WaitTime: 541
Complete
`

const amStartAndroid8 = `Starting: Intent { cmp=org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity }
Status: ok
Activity: org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity
ThisTime: 748
TotalTime: 748
WaitTime: 783
Complete
`

const amStartMissingActivity = `Starting: Intent { cmp=org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity }
Error type 3
Error: Activity class {org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity} does not exist.
`

const amStartTimeout = `Starting: Intent { cmp=org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity }
Status: timeout
LaunchState: UNKNOWN (0)
Activity: org.glutter.counter/io.github.rm4n0s.glutter.GlutterActivity
TotalTime: 0
WaitTime: 10010
Complete
`

func TestParseAmStart(t *testing.T) {
	ms := time.Millisecond
	tests := []struct {
		name  string
		out   string
		want  amStart
		route string // empty when it parses
	}{
		{name: "Android 14", out: amStartAndroid14, want: amStart{Status: "ok", LaunchState: "COLD", Total: 523 * ms, Wait: 541 * ms}},
		{name: "Android 8 has ThisTime and no LaunchState", out: amStartAndroid8, want: amStart{Status: "ok", This: 748 * ms, Total: 748 * ms, Wait: 783 * ms}},
		{name: "Windows line ends", out: strings.ReplaceAll(amStartAndroid14, "\n", "\r\n"), want: amStart{Status: "ok", LaunchState: "COLD", Total: 523 * ms, Wait: 541 * ms}},
		{name: "an activity that does not exist", out: amStartMissingActivity, route: "parseAmStart." + tagAndroidPerfStartRejected},
		{name: "a timeout", out: amStartTimeout, route: "parseAmStart." + tagAndroidPerfStartRejected},
		{name: "no times", out: "Starting: Intent { cmp=a/b }\nStatus: ok\nComplete\n", route: "parseAmStart." + tagAndroidPerfAmStartParse},
		{name: "nothing", out: "", route: "parseAmStart." + tagAndroidPerfAmStartParse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmStart(tt.out)
			if tt.route != "" {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(tt.route) {
					t.Fatalf("got %v, want route %s", err, tt.route)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("parseAmStart = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func TestParseProcStat(t *testing.T) {
	tests := []struct {
		name  string
		stat  string
		want  uint64
		route string
	}{
		{name: "an app", stat: "4242 (org.glutter.counter) S 1234 1234 0 0 -1 1077936448 25000 0 0 0 1523 311 0 0 20 0 28 0 123456 6412345344 51234 18446744073709551615 1 1 0 0 0 0 4612 1 1073775864 0 0 0 17 3 0 0 0 0 0\n", want: 1834},
		{name: "a name with blanks and parentheses", stat: "77 (Binder:77_3) (x) R 1 1 0 0 -1 4194560 100 0 0 0 7 5 0 0 20 0 1 0 1 1 1 1 1 1\n", want: 12},
		{name: "idle", stat: "9 (a) S 1 1 0 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 1 1 1\n", want: 0},
		{name: "no parenthesis", stat: "garbage", route: "parseProcStat." + tagAndroidPerfProcStatParse},
		{name: "too few fields", stat: "1 (a) S 1 1 0", route: "parseProcStat." + tagAndroidPerfProcStatParse},
		{name: "not numbers", stat: "1 (a) S 1 1 0 0 -1 0 0 0 0 0 x y 0 0 20 0 1 0 1", route: "parseProcStat." + tagAndroidPerfProcStatParse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProcStat(tt.stat)
			if tt.route != "" {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(tt.route) {
					t.Fatalf("got %v, want route %s", err, tt.route)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("parseProcStat = %d, %v; want %d", got, err, tt.want)
			}
		})
	}
}

func TestParseProcStatus(t *testing.T) {
	status := "Name:\torg.glutter.counter\nUmask:\t0077\nState:\tS (sleeping)\nTgid:\t4242\nVmPeak:\t 6511220 kB\nVmSize:\t 6412344 kB\nVmHWM:\t  151204 kB\nVmRSS:\t  143360 kB\nRssAnon:\t   61440 kB\nThreads:\t28\n"
	tests := []struct {
		name   string
		status string
		want   procStatus
		route  string
	}{
		{name: "a running app", status: status, want: procStatus{VmRSSKB: 143360, VmHWMKB: 151204}},
		{name: "no VmHWM", status: "VmRSS:\t  100 kB\n", want: procStatus{VmRSSKB: 100}},
		{name: "a process that ended", status: "cat: /proc/4242/status: No such file or directory\n", route: "parseProcStatus." + tagAndroidPerfProcStatusParse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProcStatus(tt.status)
			if tt.route != "" {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(tt.route) {
					t.Fatalf("got %v, want route %s", err, tt.route)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("parseProcStatus = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

// meminfoAndroid14 is `dumpsys meminfo org.glutter.counter` of Android 14 (API 34): an Rss column in
// the table and "TOTAL PSS: n TOTAL RSS: n" in the App Summary.
const meminfoAndroid14 = `Applications Memory Usage (in Kilobytes):
Uptime: 1203998 Realtime: 1203998

** MEMINFO in pid 4242 [org.glutter.counter] **
                   Pss  Private  Private  SwapPss      Rss     Heap     Heap     Heap
                 Total    Dirty    Clean    Dirty    Total     Size    Alloc     Free
                ------   ------   ------   ------   ------   ------   ------   ------
  Native Heap    21346    21296        0        0    22640    32768    26112     6655
  Dalvik Heap     1856     1820        0        0     3500     4116     1524     2592
 Dalvik Other      612      612        0        0      896
        Stack      124      124        0        0      132
       Ashmem        2        0        0        0       76
    Other dev       20        0       20        0      304
     .so mmap    11422      444     5832        0    28180
    .jar mmap      340        0        0        0     3268
    .apk mmap    16410        0    16408        0    20488
    .dex mmap      104        0       32        0      160
    .oat mmap      120        0        0        0     1776
    .art mmap     1350      900        0        0     8300
   Other mmap       56        8       12        0      528
      GL mtrack    8292     8292        0        0     8292
      Unknown     4382     4382        0        0     4976
        TOTAL    66436    38878    22304        0   103516    36884    27636     9247

 App Summary
                       Pss(KB)                        Rss(KB)
                        ------                         ------
           Java Heap:     2720                           8664
         Native Heap:    21296                          22640
                Code:    22616                          55004
               Stack:      124                            132
            Graphics:     8292                           8292
       Private Other:     5030
              System:     6358
             Unknown:                                    8784

           TOTAL PSS:    66436            TOTAL RSS:   103516       TOTAL SWAP PSS:        0

 Objects
               Views:        0         ViewRootImpl:        0
         AppContexts:        3           Activities:        1
              Assets:        7        AssetManagers:        0
       Local Binders:       12        Proxy Binders:       24
       Parcel memory:        4         Parcel count:       16
    Death Recipients:        0      OpenSSL Sockets:        0
            WebViews:        0
`

// meminfoAndroid8 is the same of Android 8.0 (API 26): no Rss column, and "TOTAL:" in the App Summary.
const meminfoAndroid8 = `Applications Memory Usage (in kB):
Uptime: 397841 Realtime: 397841

** MEMINFO in pid 3187 [org.glutter.counter] **
                   Pss  Private  Private  SwapPss     Heap     Heap     Heap
                 Total    Dirty    Clean    Dirty     Size    Alloc     Free
                ------   ------   ------   ------   ------   ------   ------
  Native Heap    14912    14860        0        0    22528    18204     4323
  Dalvik Heap     1427     1400        0        0     3072     1388     1684
 Dalvik Other      408      408        0        0
        Stack      100      100        0        0
       Ashmem        3        0        0        0
    Other dev        8        0        8        0
     .so mmap     6190      332     3236        0
    .apk mmap    11054        0    11052        0
    .dex mmap       16        0        0        0
    .oat mmap      912        0        0        0
    .art mmap     1098      872        0        0
   Other mmap       43        4       16        0
      GL mtrack     5764     5764        0        0
      Unknown     1630     1630        0        0
        TOTAL    43565    25370    14312        0    25600    19592     6007

 App Summary
                       Pss(KB)
                        ------
           Java Heap:     2272
         Native Heap:    14860
                Code:    14796
               Stack:      100
            Graphics:     5764
       Private Other:     2062
              System:     3711

               TOTAL:    43565       TOTAL SWAP PSS:        0

 Objects
               Views:        0         ViewRootImpl:        0
`

// meminfoAndroid11 is Android 11 (API 30): an Rss column in the table, "TOTAL:" in the App Summary.
const meminfoAndroid11 = `Applications Memory Usage (in Kilobytes):
Uptime: 2210347 Realtime: 2210347

** MEMINFO in pid 8120 [org.glutter.counter] **
                   Pss  Private  Private  SwapPss      Rss     Heap     Heap     Heap
                 Total    Dirty    Clean    Dirty    Total     Size    Alloc     Free
                ------   ------   ------   ------   ------   ------   ------   ------
  Native Heap    17236    17192        0        0    18480    26624    20496     6127
  Dalvik Heap     1688     1664        0        0     3232     3764     1496     2268
        TOTAL    58880    32204    19680        0    98240    30388    21992     8395

 App Summary
                       Pss(KB)                        Rss(KB)
                        ------                         ------
           Java Heap:     2412                           6820
         Native Heap:    17192                          18480

               TOTAL:    58880       TOTAL SWAP PSS:        0
`

func TestParseMeminfo(t *testing.T) {
	tests := []struct {
		name  string
		out   string
		want  meminfo
		route string
	}{
		{name: "Android 14: TOTAL PSS and TOTAL RSS", out: meminfoAndroid14, want: meminfo{PSSKB: 66436, RSSKB: 103516, HasRSS: true}},
		{name: "Android 11: the Rss column of the table", out: meminfoAndroid11, want: meminfo{PSSKB: 58880, RSSKB: 98240, HasRSS: true}},
		{name: "Android 8: no RSS", out: meminfoAndroid8, want: meminfo{PSSKB: 43565}},
		{name: "only the summary line", out: "           TOTAL PSS:    1234            TOTAL RSS:     5678       TOTAL SWAP PSS:        0\n", want: meminfo{PSSKB: 1234, RSSKB: 5678, HasRSS: true}},
		{name: "a process that does not run", out: "No process found for: org.glutter.counter\n", route: "parseMeminfo." + tagAndroidPerfMeminfoParse},
		{name: "nothing", out: "", route: "parseMeminfo." + tagAndroidPerfMeminfoParse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMeminfo(tt.out)
			if tt.route != "" {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(tt.route) {
					t.Fatalf("got %v, want route %s", err, tt.route)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("parseMeminfo = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func TestParseFirstFrame(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want firstFrame
		ok   bool
	}{
		{name: "a logcat brief line", ok: true,
			log: "I/org.glutter.counter( 4242): 2026/10/24 10:11:11 INFO startup stage=\"x\" took=3ms\n" +
				"I/org.glutter.counter( 4242): 2026/10/24 10:11:12 INFO first frame presented since_run=212ms renderer=\"vulkan: Adreno (TM) 740\" go_heap_mb=9 go_sys_mb=22\n",
			want: firstFrame{SinceRun: 212 * time.Millisecond, Renderer: "vulkan: Adreno (TM) 740"}},
		{name: "with the time since the process started", ok: true,
			log:  "I/org.glutter.counter(  100): 2026/10/24 10:11:12 INFO first frame presented since_run=1.25s renderer=gles since_process_start=1.4s\n",
			want: firstFrame{SinceRun: 1250 * time.Millisecond, Renderer: "gles", SinceProcess: 1400 * time.Millisecond}},
		{name: "not yet", log: "I/org.glutter.counter( 4242): 2026/10/24 10:11:11 INFO startup stage=x\n"},
		{name: "a value that is not a duration", log: "INFO first frame presented since_run=soon\n"},
		{name: "an empty log", log: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseFirstFrame(tt.log)
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseFirstFrame = %+v, %v; want %+v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseFrameLines(t *testing.T) {
	log := `I/org.glutter.gallery(  500): 2026/10/24 10:11:13 INFO first frame presented since_run=300ms renderer="gles: Mali-G78"
I/org.glutter.gallery(  500): 2026/10/24 10:11:14 INFO slow frame build=18.52ms raster=1.2ms
I/org.glutter.gallery(  500): 2026/10/24 10:11:14 INFO frames renderer="gles: Mali-G78" fps=58 build_avg=3.1ms build_max=18.52ms raster_avg=1.9ms raster_max=4.4ms over_16ms=1 go_heap_mb=14 go_sys_mb=38
I/org.glutter.gallery(  500): 2026/10/24 10:11:15 INFO frames renderer="gles: Mali-G78" fps=60 build_avg=2.9ms build_max=4ms raster_avg=1.8ms raster_max=3ms over_16ms=0 go_heap_mb=15 go_sys_mb=38
I/org.glutter.gallery(  500): 2026/10/24 10:11:16 INFO frames renderer=x fps=many
D/something( 1): unrelated frames of the movie
`
	got := parseFrameLines(log)
	if len(got) != 2 {
		t.Fatalf("parseFrameLines found %d lines, want 2: %+v", len(got), got)
	}
	if got[0].FPS != 58 || got[0].Over16 != 1 || got[0].BuildMax != 18520*time.Microsecond || got[0].Renderer != "gles: Mali-G78" || got[1].GoHeapMB != 15 {
		t.Errorf("parseFrameLines = %+v", got)
	}
	if len(parseFrameLines("")) != 0 {
		t.Error("an empty log has frames")
	}
}

func TestParsePid(t *testing.T) {
	tests := []struct {
		out  string
		want int
	}{{"4242\n", 4242}, {"  17\r\n", 17}, {"", 0}, {"pidof: bad", 0}, {"-3", 0}}
	for _, tt := range tests {
		if got := parsePid(tt.out); got != tt.want {
			t.Errorf("parsePid(%q) = %d, want %d", tt.out, got, tt.want)
		}
	}
}

func TestPickABI(t *testing.T) {
	arm, x86 := androidtool.ABIs[0], androidtool.ABIs[1]
	tests := []struct {
		name    string
		abilist string
		built   []androidtool.ABI
		want    androidtool.ABI
		route   string
	}{
		{name: "an emulator", abilist: "x86_64,arm64-v8a\n", built: androidtool.ABIs, want: x86},
		{name: "a phone", abilist: "arm64-v8a,armeabi-v7a,armeabi", built: androidtool.ABIs, want: arm},
		{name: "the device prefers x86_64 but only arm64 was built", abilist: "x86_64,arm64-v8a", built: []androidtool.ABI{arm}, want: arm},
		{name: "none", abilist: "armeabi-v7a", built: androidtool.ABIs, route: "pickABI." + tagAndroidPerfNoABI},
		{name: "built for the other ABI", abilist: "x86_64", built: []androidtool.ABI{arm}, route: "pickABI." + tagAndroidPerfNoABI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pickABI(tt.abilist, tt.built)
			if tt.route != "" {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(tt.route) {
					t.Fatalf("got %v, want route %s", err, tt.route)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("pickABI = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func TestParseABIs(t *testing.T) {
	tests := []struct {
		in    string
		want  string // the short names
		route string
	}{
		{in: "arm64,amd64", want: "arm64,amd64"},
		{in: " amd64 ", want: "amd64"},
		{in: "x86_64,arm64-v8a", want: "amd64,arm64"},
		{in: "armv7", route: "parseABIs." + tagAndroidPerfBadABI},
		{in: "", route: "parseABIs." + tagAndroidPerfBadABI},
		{in: ",", route: "parseABIs." + tagAndroidPerfBadABI},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseABIs(tt.in)
			if tt.route != "" {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(tt.route) {
					t.Fatalf("got %v, want route %s", err, tt.route)
				}
				return
			}
			var shorts []string
			for _, a := range got {
				shorts = append(shorts, a.Short)
			}
			if err != nil || strings.Join(shorts, ",") != tt.want {
				t.Errorf("parseABIs = %v, %v; want %s", shorts, err, tt.want)
			}
		})
	}
}

// writeZip makes a zip with the given file names and sizes.
func writeZip(t *testing.T, path string, files map[string]int, method uint16) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, size := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(make([]byte, size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReadAPK(t *testing.T) {
	dir := t.TempDir()
	both := filepath.Join(dir, "both.apk")
	writeZip(t, both, map[string]int{
		"AndroidManifest.xml":            400,
		"classes.dex":                    2000,
		"lib/arm64-v8a/libapp.so":        9000,
		"lib/x86_64/libapp.so":           10000,
		"lib/x86_64/libother.so":         77,
		"assets/lib/arm64-v8a/libapp.so": 5, // not at the top
	}, zip.Store)
	deflated := filepath.Join(dir, "deflated.apk")
	writeZip(t, deflated, map[string]int{"lib/arm64-v8a/libapp.so": 50000}, zip.Deflate)
	notZip := filepath.Join(dir, "x.apk")
	if err := os.WriteFile(notZip, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		path  string
		libs  map[string]int64
		route string
	}{
		{name: "both ABIs", path: both, libs: map[string]int64{"arm64-v8a": 9000, "x86_64": 10000}},
		{name: "the size is the uncompressed one", path: deflated, libs: map[string]int64{"arm64-v8a": 50000}},
		{name: "not a zip", path: notZip, route: "readAPK." + tagAndroidPerfAPKRead},
		{name: "missing", path: filepath.Join(dir, "none.apk"), route: "readAPK." + tagAndroidPerfAPKRead},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readAPK(tt.path)
			if tt.route != "" {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(tt.route) {
					t.Fatalf("got %v, want route %s", err, tt.route)
				}
				return
			}
			if err != nil || len(got.LibBytes) != len(tt.libs) {
				t.Fatalf("readAPK = %+v, %v; want %v", got, err, tt.libs)
			}
			for k, v := range tt.libs {
				if got.LibBytes[k] != v {
					t.Errorf("libapp.so of %s = %d, want %d", k, got.LibBytes[k], v)
				}
			}
		})
	}
}
