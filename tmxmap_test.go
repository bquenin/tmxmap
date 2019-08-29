package tmxmap

import (
	"testing"
)

func TestExternal(t *testing.T) {
	tmx, err := Load("assets/external/track1_bg.tmx")
	if err != nil {
		t.Error(err)
	}
	if tmx.TileSets[0].Image.Image == nil {
		t.Errorf("tileset image should not be null")
	}
}

func TestEmbedded(t *testing.T) {
	tmx, err := Load("assets/embedded/overworld.tmx")
	if err != nil {
		t.Error(err)
	}
	if tmx.TileSets[0].Image.Image == nil {
		t.Errorf("tileset image should not be null")
	}
}
