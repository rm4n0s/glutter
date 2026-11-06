package audiochannels

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

var names = []string{"Mono", "Stereo", "21", "Quad", "41", "51", "61", "71"}

// header makes a header with a converter for each pair of layouts, written the way SDL's generator writes them: the first source
// channel as src[0], the second as a named variable and a destination that is a bare variable when it has only one source.
func header() string {
	var b strings.Builder
	for s := 1; s <= 8; s++ {
		for d := 1; d <= 8; d++ {
			if s == d {
				continue
			}
			fmt.Fprintf(&b, "static void SDL_Convert%sTo%s(float *dst, const float *src, int num_frames)\n{\n", names[s-1], names[d-1])
			if s > 1 {
				b.WriteString("    const float srcSecond = src[1];\n")
			}
			for k := 0; k < d; k++ {
				switch {
				case k == 0 && s > 1:
					fmt.Fprintf(&b, "    dst[%d] /* FL */ = (src[0] * 0.500000000f) + (srcSecond * 0.250000000f);\n", k)
				case k == 1 && s > 1:
					fmt.Fprintf(&b, "    dst[%d] /* FR */ = srcSecond;\n", k)
				case k == 0:
					fmt.Fprintf(&b, "    dst[%d] /* FC */ = src[0];\n", k)
				default:
					fmt.Fprintf(&b, "    dst[%d] /* X */ = 0.0f;\n", k)
				}
			}
			b.WriteString("}\n\n")
		}
	}
	return b.String()
}

func TestParseReadsTheTermsOfAConverter(t *testing.T) {
	m, err := parse(header())
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 56 {
		t.Fatalf("%d converters", len(m))
	}
	tests := []struct {
		src, dst int
		want     [][]float32
	}{
		{1, 2, [][]float32{{1}, {0}}},
		{2, 1, [][]float32{{0.5, 0.25}}},
		{3, 3 + 1, [][]float32{{0.5, 0.25, 0}, {0, 1, 0}, {0, 0, 0}, {0, 0, 0}}},
	}
	for _, tt := range tests {
		got := m[[2]int{tt.src, tt.dst}]
		for k, row := range tt.want {
			for j, c := range row {
				if got.rows[k][j] != c {
					t.Errorf("%d->%d [%d][%d] = %v, want %v", tt.src, tt.dst, k, j, got.rows[k][j], c)
				}
			}
		}
	}
}

func TestParseRealHeader(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "..", "..", defaultReference, convertersHeader))
	if err != nil {
		t.Skipf("no SDL sources (scripts/fetch-reference.sh): %v", err)
	}
	m, err := parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	if got := m[[2]int{2, 1}].rows[0]; got[0] != 0.5 || got[1] != 0.5 {
		t.Errorf("stereo to mono = %v", got)
	}
	// 5.1 to stereo (SDL's own numbers): the front left reaches the left with 0.294545442.
	if got := m[[2]int{6, 2}].rows[0][0]; got != 0.294545442 {
		t.Errorf("5.1 to stereo FL = %v", got)
	}
}

func TestParseErrors(t *testing.T) {
	good := header()
	tests := []struct {
		name  string
		text  string
		route string
	}{
		{"too few converters", "static void SDL_ConvertMonoToStereo(float *dst, const float *src, int num_frames)\n{\n dst[0] /* FL */ = src[0];\n dst[1] /* FR */ = src[0];\n}\n", "parse.AudioChannelsSyntax"},
		{"a channel out of range", strings.Replace(good, "dst[1] /* FR */ = srcSecond;", "dst[9] /* FR */ = srcSecond;", 1), "parse.AudioChannelsSyntax"},
		{"a source that has no name", strings.Replace(good, "(srcSecond * 0.250000000f)", "(srcNobody * 0.250000000f)", 1), "parse->channelIndex.AudioChannelsSyntax"},
		{"an expression that is not a sum", strings.Replace(good, "(src[0] * 0.500000000f) + (srcSecond * 0.250000000f)", "src[0] / srcSecond", 1), "parse.AudioChannelsSyntax"},
		{"a bare source that has no name", strings.Replace(good, "dst[1] /* FR */ = srcSecond;", "dst[1] /* FR */ = srcNobody;", 1), "parse->channelIndex.AudioChannelsSyntax"},
		{"a source channel that is out of range", strings.Replace(good, "(srcSecond * 0.250000000f)", "(src[7] * 0.250000000f)", 1), "parse.AudioChannelsSyntax"},
		{"a source number that is too long", strings.Replace(good, "(srcSecond * 0.250000000f)", "(src[99999999999999999999] * 0.250000000f)", 1), "parse->channelIndex.AudioChannelsSyntax"},
		{"a bare source that is out of range", strings.Replace(good, "dst[1] /* FR */ = srcSecond;", "dst[1] /* FR */ = src[6];", 1), "parse.AudioChannelsSyntax"},
		{"a channel that is not set", strings.Replace(good, "dst[1] /* FR */ = srcSecond;\n", "", 1), "parse.AudioChannelsSyntax"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse(tt.text)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tagAudioChannelsSyntax || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestRunWritesTheTable(t *testing.T) {
	ref, out := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(ref, "src", "audio"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ref, convertersHeader), []byte(header()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"-reference", ref, "-out", out}); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(out, "channels_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "var channelMatrices = [8][8][]float32{") || !strings.Contains(string(first), "0: {0.5, 0.25}") {
		t.Errorf("the table is not what was asked for:\n%.400s", first)
	}
	// A second run changes nothing, not even the time of the file.
	info, err := os.Stat(filepath.Join(out, "channels_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"-reference", ref, "-out", out}); err != nil {
		t.Fatal(err)
	}
	again, err := os.Stat(filepath.Join(out, "channels_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !again.ModTime().Equal(info.ModTime()) {
		t.Error("the file was written again")
	}
}

func TestRunErrors(t *testing.T) {
	ref := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ref, "src", "audio"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ref, convertersHeader), []byte(header()), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		args  []string
		tag   string
		route string
	}{
		{"bad flags", []string{"-nope"}, tagAudioChannelsBadFlags, "Run.AudioChannelsBadFlags"},
		{"no header", []string{"-reference", t.TempDir()}, tagAudioChannelsReadFailed, "Run.AudioChannelsReadFailed"},
		{"an output directory that is not there", []string{"-reference", ref, "-out", filepath.Join(t.TempDir(), "nothing")}, tagAudioChannelsWriteFailed, "Run.AudioChannelsWriteFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(tt.args)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestFormatCodeFailsOnSourceThatIsNotGo(t *testing.T) {
	_, err := formatCode([]byte("package x\nfunc ("))
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tagAudioChannelsFormatFailed || !appErr.HasRoute("formatCode.AudioChannelsFormatFailed") {
		t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
	}
}
