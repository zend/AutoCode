package pokedex

import (
	"os"
	"path/filepath"
	"testing"
)

func testDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	manifest := `{
  "schema_version": "1.0.0",
  "name": "test-pokedex",
  "entry_count": 1,
  "species_count": 1,
  "anti_hallucination": {
    "not_found_message": "未找到，请勿编造"
  }
}`
	index := `[
  {
    "entry_id": "0025",
    "national_id": 25,
    "form": "default",
    "names": {"zh": "皮卡丘", "en": "Pikachu", "ja": "ピカチュウ"},
    "types": ["electric"],
    "types_zh": ["电"],
    "generation": 1,
    "base_stat_total": 320
  }
]`
	entry := `{
  "entry_id": "0025",
  "national_id": 25,
  "form": "default",
  "names": {"zh": "皮卡丘", "en": "Pikachu", "ja": "ピカチュウ"},
  "generation": 1,
  "types": ["electric"],
  "types_zh": ["电"],
  "category_zh": "鼠宝可梦",
  "abilities": [
    {"name_en": "static", "name_zh": "静电", "hidden": false},
    {"name_en": "lightning-rod", "name_zh": "避雷针", "hidden": true}
  ],
  "base_stats": {
    "hp": 35, "attack": 55, "defense": 40,
    "special_attack": 50, "special_defense": 50, "speed": 90, "total": 320
  },
  "is_default_form": true,
  "source": {
    "pokeapi_id": 25,
    "pokeapi_species_id": 25,
    "pokeapi_url": "https://pokeapi.co/api/v2/pokemon/25"
  }
}`

	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(index), 0644); err != nil {
		t.Fatal(err)
	}
	entriesDir := filepath.Join(dir, "entries")
	if err := os.Mkdir(entriesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entriesDir, "0025.json"), []byte(entry), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoad(t *testing.T) {
	kb, err := Load(testDataDir(t))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if kb.EntryCount() != 1 {
		t.Errorf("expected 1 entry, got %d", kb.EntryCount())
	}
}

func TestGetByNationalID(t *testing.T) {
	kb, err := Load(testDataDir(t))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	result, err := kb.GetByNationalID(25, "")
	if err != nil {
		t.Fatalf("GetByNationalID failed: %v", err)
	}
	if !result.Found || result.Count != 1 {
		t.Fatalf("expected found=1, got %+v", result)
	}
	if result.Entries[0].Names.Zh != "皮卡丘" {
		t.Errorf("unexpected name: %s", result.Entries[0].Names.Zh)
	}
}

func TestQueryByName(t *testing.T) {
	kb, err := Load(testDataDir(t))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	result, err := kb.Query(QueryParams{Name: "皮卡丘"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !result.Found {
		t.Fatal("expected to find Pikachu")
	}
}

func TestQueryByType(t *testing.T) {
	kb, err := Load(testDataDir(t))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	result, err := kb.Query(QueryParams{Type: "electric"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !result.Found {
		t.Fatal("expected to find electric type")
	}
}

func TestNotFound(t *testing.T) {
	kb, err := Load(testDataDir(t))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	result, err := kb.GetByNationalID(9999, "")
	if err != nil {
		t.Fatalf("GetByNationalID failed: %v", err)
	}
	if result.Found {
		t.Fatal("expected not found")
	}
	if result.Message == "" {
		t.Error("expected anti-hallucination message")
	}
}

func TestFormatEntry(t *testing.T) {
	kb, err := Load(testDataDir(t))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	result, err := kb.GetByNationalID(25, "")
	if err != nil {
		t.Fatalf("GetByNationalID failed: %v", err)
	}

	formatted := FormatEntry(result.Entries[0])
	if !contains(formatted, "皮卡丘") || !contains(formatted, "320") {
		t.Errorf("unexpected format: %s", formatted)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func TestParseNationalID(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"25", 25},
		{"#0025", 25},
		{"0025", 25},
	}
	for _, tt := range tests {
		got, err := ParseNationalID(tt.input)
		if err != nil {
			t.Errorf("ParseNationalID(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseNationalID(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
