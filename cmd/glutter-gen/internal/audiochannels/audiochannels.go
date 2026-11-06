// Package audiochannels implements the `glutter-gen audiochannels` subcommand, which turns the channel converters of SDL 3
// (src/audio/SDL_audio_channel_converters.h, which SDL generates itself) into the matrices that internal/audiocvt applies
// (SPEC §28.1, §28.3).
//
// Flutter: no counterpart.
package audiochannels

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

const (
	defaultReference = "reference/sdl"
	defaultOutDir    = "internal/audiocvt"
	convertersHeader = "src/audio/SDL_audio_channel_converters.h"
	maxChannels      = 8
)

// layoutChannels is the number of channels of each name that SDL's converters use in their function names.
var layoutChannels = map[string]int{"Mono": 1, "Stereo": 2, "21": 3, "Quad": 4, "41": 5, "51": 6, "61": 7, "71": 8}

// Run executes `glutter-gen audiochannels [-reference dir] [-out dir]`. It reads SDL 3's SDL_audio_channel_converters.h under the
// reference directory and writes channels_gen.go to the output directory.
//
// Flutter: no counterpart
func Run(args []string) error {
	flags := flag.NewFlagSet("audiochannels", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reference := flags.String("reference", defaultReference, "directory with the SDL 3 sources (see scripts/fetch-reference.sh)")
	outDir := flags.String("out", defaultOutDir, "directory to write channels_gen.go to")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagAudioChannelsBadFlags, err).SetMetadata("args", args)
	}
	path := filepath.Join(*reference, convertersHeader)
	text, err := os.ReadFile(path)
	if err != nil {
		return errors.NewErr(tagAudioChannelsReadFailed, err).SetMetadata("file", path)
	}
	matrices, err := parse(string(text))
	if err != nil {
		return err
	}
	code, err := generate(matrices)
	if err != nil {
		return err
	}
	out := filepath.Join(*outDir, "channels_gen.go")
	if old, err := os.ReadFile(out); err == nil && bytes.Equal(old, code) {
		return nil
	}
	if err := os.WriteFile(out, code, 0o644); err != nil {
		return errors.NewErr(tagAudioChannelsWriteFailed, err).SetMetadata("file", out)
	}
	return nil
}

// matrix is a converter as a matrix of dst rows and src columns.
type matrix struct {
	src, dst int
	rows     [][]float32
}

var (
	funcRE  = regexp.MustCompile(`^static void SDL_Convert(Mono|Stereo|21|Quad|41|51|61|71)To(Mono|Stereo|21|Quad|41|51|61|71)\(`)
	varRE   = regexp.MustCompile(`^\s*const float (src[A-Za-z0-9]+) = src\[(\d+)\];`)
	dstRE   = regexp.MustCompile(`^\s*dst\[(\d+)\]\s*/\*[^*]*\*/\s*=\s*(.*);\s*$`)
	termRE  = regexp.MustCompile(`\(\s*(src(?:[A-Za-z0-9]+|\[\d+\]))\s*\*\s*([0-9]*\.[0-9]+)f\s*\)`)
	bareRE  = regexp.MustCompile(`^\s*(src(?:[A-Za-z0-9]+|\[\d+\]))\s*$`)
	indexRE = regexp.MustCompile(`^src\[(\d+)\]$`)
)

// parse reads the converters of the header and returns them by source and destination channels.
func parse(text string) (map[[2]int]*matrix, error) {
	out := map[[2]int]*matrix{}
	var cur *matrix
	vars := map[string]int{}
	for n, line := range strings.Split(text, "\n") {
		if m := funcRE.FindStringSubmatch(line); m != nil {
			cur = &matrix{src: layoutChannels[m[1]], dst: layoutChannels[m[2]]}
			cur.rows = make([][]float32, cur.dst)
			vars = map[string]int{}
			out[[2]int{cur.src, cur.dst}] = cur
			continue
		}
		if cur == nil {
			continue
		}
		if m := varRE.FindStringSubmatch(line); m != nil {
			idx, err := strconv.Atoi(m[2])
			if err != nil {
				return nil, errors.NewErr(tagAudioChannelsSyntax, err).SetMetadata("line", n+1)
			}
			vars[m[1]] = idx
			continue
		}
		m := dstRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		k, err := strconv.Atoi(m[1])
		if err != nil || k >= cur.dst {
			return nil, errors.New(tagAudioChannelsSyntax, "a destination channel is out of range", "line", n+1, "channel", m[1])
		}
		row := make([]float32, cur.src)
		expr := m[2]
		for _, t := range termRE.FindAllStringSubmatch(expr, -1) {
			idx, err := channelIndex(t[1], vars)
			if err != nil {
				return nil, err
			}
			if idx >= cur.src {
				return nil, errors.New(tagAudioChannelsSyntax, "a source channel is out of range", "line", n+1, "name", t[1])
			}
			c, err := strconv.ParseFloat(t[2], 32)
			if err != nil {
				return nil, errors.NewErr(tagAudioChannelsSyntax, err).SetMetadata("line", n+1)
			}
			row[idx] += float32(c)
		}
		rest := strings.TrimSpace(termRE.ReplaceAllString(expr, ""))
		rest = strings.Trim(strings.ReplaceAll(rest, "+", ""), " ")
		switch {
		case rest == "":
		case rest == "0.0f":
		case bareRE.MatchString(rest):
			idx, err := channelIndex(rest, vars)
			if err != nil {
				return nil, err
			}
			if idx >= cur.src {
				return nil, errors.New(tagAudioChannelsSyntax, "a source channel is out of range", "line", n+1, "name", rest)
			}
			row[idx] += 1
		default:
			return nil, errors.New(tagAudioChannelsSyntax, "an expression of a converter is not a sum of products", "line", n+1, "expression", expr)
		}
		cur.rows[k] = row
	}
	if len(out) != maxChannels*(maxChannels-1) {
		return nil, errors.New(tagAudioChannelsSyntax, "the header does not have a converter for each pair of layouts", "found", len(out))
	}
	for key, m := range out {
		for k, row := range m.rows {
			if row == nil {
				return nil, errors.New(tagAudioChannelsSyntax, "a converter does not set a channel", "src", key[0], "dst", key[1], "channel", k)
			}
		}
	}
	return out, nil
}

// channelIndex is the index of a source channel written as src[N] or as one of the names that the converter gave to a channel.
func channelIndex(name string, vars map[string]int) (int, error) {
	if m := indexRE.FindStringSubmatch(name); m != nil {
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, errors.NewErr(tagAudioChannelsSyntax, err).SetMetadata("name", name)
		}
		return idx, nil
	}
	if idx, ok := vars[name]; ok {
		return idx, nil
	}
	return 0, errors.New(tagAudioChannelsSyntax, "a source channel has no name", "name", name)
}

const generatedNotice = `// Code generated by glutter-gen audiochannels; DO NOT EDIT.
//
// This file is made from %s of SDL 3 (https://libsdl.org, zlib license), translated to Go and changed. SDL's notice is kept in
// third_party/sdl/LICENSE.

`

func generate(matrices map[[2]int]*matrix) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, generatedNotice, convertersHeader)
	b.WriteString("package audiocvt\n\n")
	b.WriteString("// channelMatrices[src-1][dst-1] is the conversion from src channels to dst channels (SDL's layouts: mono, stereo, 2.1, quad, 4.1, 5.1,\n// 6.1 and 7.1) as a matrix of dst rows and src columns, row after row; the entry for equal counts is nil.\nvar channelMatrices = [8][8][]float32{\n")
	for src := 1; src <= maxChannels; src++ {
		fmt.Fprintf(&b, "\t%d: {\n", src-1)
		for dst := 1; dst <= maxChannels; dst++ {
			if src == dst {
				continue
			}
			m := matrices[[2]int{src, dst}]
			fmt.Fprintf(&b, "\t\t%d: {", dst-1)
			var flat []string
			for _, row := range m.rows {
				for _, c := range row {
					flat = append(flat, strconv.FormatFloat(float64(c), 'g', -1, 32))
				}
			}
			b.WriteString(strings.Join(flat, ", "))
			b.WriteString("},\n")
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")
	return formatCode(b.Bytes())
}

// formatCode formats the generated source; it fails only when the generator made source that is not Go.
func formatCode(src []byte) ([]byte, error) {
	out, err := format.Source(src)
	if err != nil {
		return nil, errors.NewErr(tagAudioChannelsFormatFailed, err)
	}
	return out, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAudioChannelsBadFlags     = "AudioChannelsBadFlags"
	tagAudioChannelsReadFailed   = "AudioChannelsReadFailed"
	tagAudioChannelsWriteFailed  = "AudioChannelsWriteFailed"
	tagAudioChannelsSyntax       = "AudioChannelsSyntax"
	tagAudioChannelsFormatFailed = "AudioChannelsFormatFailed"
)
