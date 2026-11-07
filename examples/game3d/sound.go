package main

import (
	"context"
	"log/slog"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/audio"
	"github.com/rm4n0s/glutter/audio/wav"
)

// sounds plays the music and the effects of the game on the default playback device. A program with no audio has no sounds (nil):
// every method is then a no-op.
type sounds struct {
	dev   *audio.Device
	music *audio.Stream
	fx    map[string]*effect

	musicOn bool
	paused  bool
}

// effect is a short sound and the stream that plays it.
type effect struct {
	stream  *audio.Stream
	samples []float32
}

// Volumes of the music, while the game is played and while it is paused, and of the effects.
const (
	musicVolume  = 0.30
	pausedVolume = 0.10
	effectVolume = 0.8
)

// newSounds opens the device and starts the music, which loops. The effects are played with play.
func newSounds(ctx context.Context, sys *audio.System, a *assets) (*sounds, error) {
	if !sys.Available() {
		return nil, errors.New(tagSoundUnavailable, "the app has no audio")
	}
	dev, err := sys.OpenPlayback(ctx)
	if err != nil {
		return nil, err
	}
	s := &sounds{dev: dev, fx: map[string]*effect{}, musicOn: true}
	for name, w := range map[string]*wav.Audio{"coin": a.coin, "jump": a.jump, "hit": a.hitFX} {
		st := dev.NewStream(w.Format)
		st.SetGain(effectVolume)
		s.fx[name] = &effect{stream: st, samples: w.Samples}
	}
	// The music is a stream of a callback that the device's thread calls: it copies the loop, over and over, and touches nothing else.
	music := a.music
	pos := 0
	s.music = dev.NewCallbackStream(music.Format, func(out []float32) {
		for i := range out {
			out[i] = music.Samples[pos]
			if pos++; pos == len(music.Samples) {
				pos = 0
			}
		}
	})
	s.applyVolume()
	return s, nil
}

// play plays an effect from its start ("coin", "jump" or "hit"), over whatever of it is still playing.
func (s *sounds) play(name string) {
	if s == nil {
		return
	}
	e := s.fx[name]
	if e == nil {
		return
	}
	e.stream.Clear()
	// The samples are float32 of the effect's own format, which the stream converts: Put fails only for a stream that is closed.
	if err := e.stream.Put(e.samples); err != nil {
		slog.Debug("an effect did not play", "effect", name, "err", err)
	}
}

// setMusic turns the music on or off.
func (s *sounds) setMusic(on bool) {
	if s == nil {
		return
	}
	s.musicOn = on
	s.applyVolume()
}

// setPaused makes the music quieter while the game is paused.
func (s *sounds) setPaused(paused bool) {
	if s == nil {
		return
	}
	s.paused = paused
	s.applyVolume()
}

func (s *sounds) applyVolume() {
	switch {
	case !s.musicOn:
		s.music.SetGain(0)
	case s.paused:
		s.music.SetGain(pausedVolume)
	default:
		s.music.SetGain(musicVolume)
	}
}

// close stops the sounds and closes the device.
func (s *sounds) close() error {
	if s == nil {
		return nil
	}
	return s.dev.Close()
}

// tagSoundUnavailable is the tag of the error of a program with no audio.
const tagSoundUnavailable = "SoundUnavailable"
