package pokedex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpriteAtlas(t *testing.T) {
	dir := testDataDir(t)

	atlasData := `{
  "schema_version": "1.0.0",
  "image": "assets/spritesheet.png",
  "image_width": 256,
  "image_height": 128,
  "cell_width": 128,
  "cell_height": 128,
  "columns": 2,
  "rows": 1,
  "entry_count": 2,
  "sprite_style": "home",
  "by_entry_id": {
    "0025": {
      "entry_id": "0025",
      "national_id": 25,
      "names": {"zh": "皮卡丘", "en": "Pikachu"},
      "index": 1,
      "col": 1,
      "row": 0,
      "x": 128,
      "y": 0,
      "width": 128,
      "height": 128,
      "pokeapi_id": 25,
      "source_url": "https://example.com/25.png",
      "sprite_style": "home"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "spritesheet.json"), []byte(atlasData), 0644); err != nil {
		t.Fatal(err)
	}

	atlas, err := LoadSpriteAtlas(dir)
	if err != nil {
		t.Fatalf("LoadSpriteAtlas failed: %v", err)
	}
	if atlas.EntryCount != 2 {
		t.Errorf("expected entry_count 2, got %d", atlas.EntryCount)
	}

	rect, ok := atlas.GetSprite("0025")
	if !ok {
		t.Fatal("expected to find sprite 0025")
	}
	if rect.X != 128 || rect.Y != 0 {
		t.Errorf("unexpected position: x=%d y=%d", rect.X, rect.Y)
	}
}

func TestKBWithSpriteAtlas(t *testing.T) {
	dir := testDataDir(t)

	atlasData := `{
  "schema_version": "1.0.0",
  "image": "assets/spritesheet.png",
  "entry_count": 1,
  "by_entry_id": {
    "0025": {
      "entry_id": "0025",
      "national_id": 25,
      "names": {"zh": "皮卡丘", "en": "Pikachu"},
      "x": 0, "y": 0, "width": 128, "height": 128,
      "col": 0, "row": 0, "index": 0,
      "pokeapi_id": 25,
      "source_url": "https://example.com/25.png",
      "sprite_style": "home"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "spritesheet.json"), []byte(atlasData), 0644); err != nil {
		t.Fatal(err)
	}

	kb, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if kb.SpriteAtlas() == nil {
		t.Fatal("expected sprite atlas to be loaded")
	}

	rect, ok := kb.GetSprite("0025")
	if !ok || rect.Names.Zh != "皮卡丘" {
		t.Fatalf("unexpected sprite: %+v ok=%v", rect, ok)
	}
}

func TestFormatSpriteRect(t *testing.T) {
	rect := SpriteRect{
		EntryID:    "0025",
		NationalID: 25,
		Names:      Names{Zh: "皮卡丘", En: "Pikachu"},
		X:          128,
		Y:          0,
		Width:      128,
		Height:     128,
		Col:        1,
		Row:        0,
		Index:      1,
		SourceURL:  "https://example.com/25.png",
	}
	out := FormatSpriteRect(rect, "/data/pokedex/assets/spritesheet.png")
	if !contains(out, "x=128") || !contains(out, "皮卡丘") {
		t.Errorf("unexpected output: %s", out)
	}
}
