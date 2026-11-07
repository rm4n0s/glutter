// Command genassets writes the assets of examples/game3d: the character and the level as glTF binary files (.glb) and the
// music and the sound effects as WAV files. Everything is made by this program, from numbers, so the files are public domain
// (CC0 1.0): the same bytes come out each time. Run it from examples/game3d (go generate does): go run ./internal/genassets assets
package main

import (
	"log"
	"os"
	"path/filepath"
)

func main() {
	dir := "assets"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	files := map[string][]byte{
		"character.glb": makeCharacter(),
		"level.glb":     makeLevel(),
		"coin.wav":      makeCoin(),
		"jump.wav":      makeJump(),
		"hit.wav":       makeHit(),
		"music.wav":     makeMusic(),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
