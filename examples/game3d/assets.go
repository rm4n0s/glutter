package main

//go:generate go run ./internal/genassets assets

import (
	"context"
	"embed"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/audio/wav"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/gltf"
)

// assetFS holds the character, the level, the music and the sound effects. internal/genassets makes them (go generate), from
// numbers: they are public domain (assets/LICENSE.txt).
//
//go:embed assets/*.glb assets/*.wav
var assetFS embed.FS

// assets are the files of the game, read.
type assets struct {
	character, level         *model.Model
	music, coin, jump, hitFX *wav.Audio
}

// loadAssets reads every asset of the game: the level and the character, which the world and the scene are made of, and the sounds.
func loadAssets(ctx context.Context) (*assets, error) {
	var a assets
	var err error
	if a.character, err = gltf.Load(ctx, assetFS, "assets/character.glb"); err != nil {
		return nil, err
	}
	if a.level, err = gltf.Load(ctx, assetFS, "assets/level.glb"); err != nil {
		return nil, err
	}
	for name, dst := range map[string]**wav.Audio{"music": &a.music, "coin": &a.coin, "jump": &a.jump, "hit": &a.hitFX} {
		data, err := readAsset("assets/" + name + ".wav")
		if err != nil {
			return nil, err
		}
		if *dst, err = wav.DecodeBytes(data); err != nil {
			return nil, err
		}
	}
	return &a, nil
}

// readAsset reads a file that is built into the program.
func readAsset(name string) ([]byte, error) {
	data, err := assetFS.ReadFile(name)
	if err != nil {
		return nil, errors.NewErr(tagAssetMissing, err).SetMetadata("file", name)
	}
	return data, nil
}

// tagAssetMissing is the tag of the error of a file that is not built into the program.
const tagAssetMissing = "AssetMissing"
