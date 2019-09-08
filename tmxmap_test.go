package tmxmap

import (
	"os"
	"testing"
)

func TestExternal(t *testing.T) {
	tmx, err := Load("assets/external/track1_bg.tmx")
	if err != nil {
		t.Error(err)
	}
	if tmx.TileSets[0].Image.Image == nil {
		t.Errorf("tileset Image.Image should not be null")
	}
}

func TestEmbedded(t *testing.T) {
	tmx, err := Load("assets/embedded/overworld.tmx")
	if err != nil {
		t.Error(err)
	}
	if tmx.TileSets[0].Image.Image == nil {
		t.Errorf("tileset Image.Image should not be null")
	}
}

func TestDecodeExternal(t *testing.T) {
	external, err := os.Open("assets/external/track1_bg.tmx")
	if err != nil {
		t.Error(err)
	}
	defer external.Close()

	tmx, err := Decode(external)
	if err != nil {
		t.Error(err)
	}
	if tmx.TileSets[0].Image != nil {
		t.Errorf("tileset Image should be null")
	}
}

func TestDecodeEmbedded(t *testing.T) {
	embedded, err := os.Open("assets/embedded/overworld.tmx")
	if err != nil {
		t.Error(err)
	}
	defer embedded.Close()

	tmx, err := Decode(embedded)
	if err != nil {
		t.Error(err)
	}
	if tmx.TileSets[0].Image == nil {
		t.Errorf("tileset Image should not be null")
	}
	if tmx.TileSets[0].Image.Image != nil {
		t.Errorf("tileset Image.Image should be null")
	}
}
