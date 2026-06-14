package pokedex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SpriteRect describes a sprite region inside the spritesheet.
type SpriteRect struct {
	EntryID     string `json:"entry_id"`
	NationalID  int    `json:"national_id"`
	Form        string `json:"form,omitempty"`
	Names       Names  `json:"names"`
	Index       int    `json:"index"`
	Col         int    `json:"col"`
	Row         int    `json:"row"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	PokeAPIID   int    `json:"pokeapi_id"`
	SourceURL   string `json:"source_url"`
	SpriteStyle string `json:"sprite_style"`
}

// SpriteAtlas holds spritesheet image metadata and per-entry coordinates.
type SpriteAtlas struct {
	SchemaVersion string                `json:"schema_version"`
	Image         string                `json:"image"`
	ImageWidth    int                   `json:"image_width"`
	ImageHeight   int                   `json:"image_height"`
	CellWidth     int                   `json:"cell_width"`
	CellHeight    int                   `json:"cell_height"`
	Columns       int                   `json:"columns"`
	Rows          int                   `json:"rows"`
	EntryCount    int                   `json:"entry_count"`
	SpriteStyle   string                `json:"sprite_style"`
	ContentSHA256 string                `json:"content_sha256"`
	BuiltAt       string                `json:"built_at"`
	ByEntryID     map[string]SpriteRect `json:"by_entry_id"`
}

// LoadSpriteAtlas reads spritesheet.json from dataDir.
func LoadSpriteAtlas(dataDir string) (*SpriteAtlas, error) {
	path := filepath.Join(dataDir, "spritesheet.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spritesheet: %w", err)
	}

	var atlas SpriteAtlas
	if err := json.Unmarshal(data, &atlas); err != nil {
		return nil, fmt.Errorf("parse spritesheet: %w", err)
	}
	if atlas.ByEntryID == nil {
		atlas.ByEntryID = make(map[string]SpriteRect)
	}
	return &atlas, nil
}

// GetSprite returns sprite coordinates for an entry ID.
func (a *SpriteAtlas) GetSprite(entryID string) (*SpriteRect, bool) {
	if a == nil {
		return nil, false
	}
	rect, ok := a.ByEntryID[entryID]
	if !ok {
		return nil, false
	}
	return &rect, true
}

// ImagePath returns the absolute path to the spritesheet image.
func (a *SpriteAtlas) ImagePath(dataDir string) string {
	if a == nil {
		return ""
	}
	return filepath.Join(dataDir, a.Image)
}

// FormatSpriteRect renders sprite coordinates for LLM observation.
func FormatSpriteRect(rect SpriteRect, imagePath string) string {
	name := rect.Names.Zh
	if name == "" {
		name = rect.Names.En
	}
	return fmt.Sprintf(
		"雪碧图: %s\n条目: %s #%04d %s\n位置: x=%d y=%d width=%d height=%d (col=%d row=%d index=%d)\n原图: %s",
		imagePath,
		rect.EntryID,
		rect.NationalID,
		name,
		rect.X,
		rect.Y,
		rect.Width,
		rect.Height,
		rect.Col,
		rect.Row,
		rect.Index,
		rect.SourceURL,
	)
}
