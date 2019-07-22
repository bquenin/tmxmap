package tmxmap

import (
	"testing"
)

func TestLoad(t *testing.T) {
	_, err := Load("assets/track1_bg.tmx")
	if err != nil {
		t.Fatal(err)
	}
}
