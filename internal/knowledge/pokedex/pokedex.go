package pokedex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Manifest describes the knowledge base metadata and anti-hallucination policy.
type Manifest struct {
	SchemaVersion     string `json:"schema_version"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	EntryCount        int    `json:"entry_count"`
	SpeciesCount      int    `json:"species_count"`
	ContentSHA256     string `json:"content_sha256"`
	BuiltAt           string `json:"built_at"`
	AntiHallucination struct {
		Policy          string `json:"policy"`
		NotFoundMessage string `json:"not_found_message"`
	} `json:"anti_hallucination"`
}

// Names holds localized Pokémon names.
type Names struct {
	Zh string `json:"zh,omitempty"`
	En string `json:"en"`
	Ja string `json:"ja,omitempty"`
}

// Ability describes a Pokémon ability.
type Ability struct {
	NameEn string `json:"name_en"`
	NameZh string `json:"name_zh,omitempty"`
	Hidden bool   `json:"hidden"`
}

// BaseStats holds base stat values.
type BaseStats struct {
	HP             int `json:"hp"`
	Attack         int `json:"attack"`
	Defense        int `json:"defense"`
	SpecialAttack  int `json:"special_attack"`
	SpecialDefense int `json:"special_defense"`
	Speed          int `json:"speed"`
	Total          int `json:"total"`
}

// GenderRatio describes gender distribution.
type GenderRatio struct {
	MalePercent   float64 `json:"male_percent,omitempty"`
	FemalePercent float64 `json:"female_percent,omitempty"`
	Genderless    bool    `json:"genderless,omitempty"`
}

// EvolutionLink is one step in an evolution chain.
type EvolutionLink struct {
	NationalID int    `json:"national_id"`
	NameEn     string `json:"name_en"`
	NameZh     string `json:"name_zh,omitempty"`
	Trigger    string `json:"trigger,omitempty"`
}

// Evolution holds evolution relationships.
type Evolution struct {
	From []EvolutionLink `json:"from,omitempty"`
	To   []EvolutionLink `json:"to,omitempty"`
}

// Source records the authoritative data origin.
type Source struct {
	PokeAPIID        int    `json:"pokeapi_id"`
	PokeAPISpeciesID int    `json:"pokeapi_species_id"`
	PokeAPIURL       string `json:"pokeapi_url,omitempty"`
}

// Entry is a complete Pokémon knowledge base record.
type Entry struct {
	EntryID        string         `json:"entry_id"`
	NationalID     int            `json:"national_id"`
	Form           string         `json:"form,omitempty"`
	Names          Names          `json:"names"`
	Generation     int            `json:"generation"`
	Types          []string       `json:"types"`
	TypesZh        []string       `json:"types_zh,omitempty"`
	Category       string         `json:"category,omitempty"`
	CategoryZh     string         `json:"category_zh,omitempty"`
	Abilities      []Ability      `json:"abilities,omitempty"`
	HeightM        float64        `json:"height_m,omitempty"`
	WeightKg       float64        `json:"weight_kg,omitempty"`
	GenderRatio    GenderRatio    `json:"gender_ratio,omitempty"`
	CaptureRate    int            `json:"capture_rate,omitempty"`
	Color          string         `json:"color,omitempty"`
	ColorZh        string         `json:"color_zh,omitempty"`
	BaseExperience int            `json:"base_experience,omitempty"`
	GrowthRate     string         `json:"growth_rate,omitempty"`
	EggGroups      []string       `json:"egg_groups,omitempty"`
	EggGroupsZh    []string       `json:"egg_groups_zh,omitempty"`
	BaseStats      BaseStats      `json:"base_stats"`
	EVYield        map[string]int `json:"ev_yield,omitempty"`
	Evolution      Evolution      `json:"evolution,omitempty"`
	RegionalDex    map[string]int `json:"regional_dex,omitempty"`
	FlavorTextZh   string         `json:"flavor_text_zh,omitempty"`
	FlavorTextEn   string         `json:"flavor_text_en,omitempty"`
	IsDefaultForm  bool           `json:"is_default_form,omitempty"`
	Source         Source         `json:"source"`
}

// IndexEntry is a compact summary for listing and filtering.
type IndexEntry struct {
	EntryID       string   `json:"entry_id"`
	NationalID    int      `json:"national_id"`
	Form          string   `json:"form,omitempty"`
	Names         Names    `json:"names"`
	Types         []string `json:"types"`
	TypesZh       []string `json:"types_zh,omitempty"`
	Generation    int      `json:"generation"`
	BaseStatTotal int      `json:"base_stat_total"`
}

// QueryParams defines structured lookup filters.
type QueryParams struct {
	NationalID int
	EntryID    string
	Name       string
	Type       string
	Generation int
	Form       string
	Limit      int
}

// QueryResult is returned by lookups with explicit grounding metadata.
type QueryResult struct {
	Found   bool    `json:"found"`
	Count   int     `json:"count"`
	Entries []Entry `json:"entries,omitempty"`
	Message string  `json:"message,omitempty"`
	Source  string  `json:"source"`
}

// KB is the loaded National Pokédex knowledge base.
type KB struct {
	dir      string
	manifest Manifest
	index    []IndexEntry
	byID     map[string]*IndexEntry
}

// Load reads the knowledge base from dataDir (expects manifest.json, index.json, entries/).
func Load(dataDir string) (*KB, error) {
	manifestPath := filepath.Join(dataDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	indexPath := filepath.Join(dataDir, "index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	var index []IndexEntry
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}

	byID := make(map[string]*IndexEntry, len(index))
	for i := range index {
		byID[index[i].EntryID] = &index[i]
	}

	return &KB{
		dir:      dataDir,
		manifest: manifest,
		index:    index,
		byID:     byID,
	}, nil
}

// Manifest returns knowledge base metadata.
func (kb *KB) Manifest() Manifest {
	return kb.manifest
}

// EntryCount returns total indexed entries.
func (kb *KB) EntryCount() int {
	return len(kb.index)
}

// NotFoundMessage returns the anti-hallucination message for missing entries.
func (kb *KB) NotFoundMessage() string {
	if kb.manifest.AntiHallucination.NotFoundMessage != "" {
		return kb.manifest.AntiHallucination.NotFoundMessage
	}
	return "知识库中未找到该条目，请勿编造"
}

func (kb *KB) loadEntry(entryID string) (*Entry, error) {
	path := filepath.Join(kb.dir, "entries", entryID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read entry %s: %w", entryID, err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parse entry %s: %w", entryID, err)
	}
	return &entry, nil
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func entryIDFromNationalID(id int, form string) string {
	if form == "" || form == "default" {
		return fmt.Sprintf("%04d", id)
	}
	return fmt.Sprintf("%04d-%s", id, form)
}

// GetByEntryID returns a single entry by its unique key.
func (kb *KB) GetByEntryID(entryID string) (*QueryResult, error) {
	entryID = strings.ToLower(strings.TrimSpace(entryID))
	if _, ok := kb.byID[entryID]; !ok {
		return &QueryResult{
			Found:   false,
			Count:   0,
			Message: kb.NotFoundMessage(),
			Source:  kb.manifest.Name,
		}, nil
	}

	entry, err := kb.loadEntry(entryID)
	if err != nil {
		return nil, err
	}

	return &QueryResult{
		Found:   true,
		Count:   1,
		Entries: []Entry{*entry},
		Source:  kb.manifest.Name,
	}, nil
}

// GetByNationalID returns entries for a national dex number, optionally filtered by form.
func (kb *KB) GetByNationalID(id int, form string) (*QueryResult, error) {
	if form != "" {
		return kb.GetByEntryID(entryIDFromNationalID(id, form))
	}

	var matches []Entry
	for _, idx := range kb.index {
		if idx.NationalID != id {
			continue
		}
		entry, err := kb.loadEntry(idx.EntryID)
		if err != nil {
			return nil, err
		}
		matches = append(matches, *entry)
	}

	if len(matches) == 0 {
		return &QueryResult{
			Found:   false,
			Count:   0,
			Message: kb.NotFoundMessage(),
			Source:  kb.manifest.Name,
		}, nil
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].IsDefaultForm != matches[j].IsDefaultForm {
			return matches[i].IsDefaultForm
		}
		return matches[i].Form < matches[j].Form
	})

	return &QueryResult{
		Found:   true,
		Count:   len(matches),
		Entries: matches,
		Source:  kb.manifest.Name,
	}, nil
}

// Query performs structured search; only returns data present in the KB.
func (kb *KB) Query(params QueryParams) (*QueryResult, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 50 {
		params.Limit = 50
	}

	if params.EntryID != "" {
		return kb.GetByEntryID(params.EntryID)
	}
	if params.NationalID > 0 {
		return kb.GetByNationalID(params.NationalID, params.Form)
	}

	nameQuery := normalizeName(params.Name)
	typeQuery := normalizeName(params.Type)

	var matched []IndexEntry
	for _, idx := range kb.index {
		if params.Generation > 0 && idx.Generation != params.Generation {
			continue
		}
		if params.Form != "" && normalizeName(idx.Form) != normalizeName(params.Form) {
			continue
		}
		if typeQuery != "" && !hasType(idx, typeQuery) {
			continue
		}
		if nameQuery != "" && !matchesName(idx, nameQuery) {
			continue
		}
		matched = append(matched, idx)
	}

	if len(matched) == 0 {
		return &QueryResult{
			Found:   false,
			Count:   0,
			Message: kb.NotFoundMessage(),
			Source:  kb.manifest.Name,
		}, nil
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].NationalID != matched[j].NationalID {
			return matched[i].NationalID < matched[j].NationalID
		}
		return matched[i].EntryID < matched[j].EntryID
	})

	if len(matched) > params.Limit {
		matched = matched[:params.Limit]
	}

	entries := make([]Entry, 0, len(matched))
	for _, idx := range matched {
		entry, err := kb.loadEntry(idx.EntryID)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}

	return &QueryResult{
		Found:   true,
		Count:   len(entries),
		Entries: entries,
		Source:  kb.manifest.Name,
	}, nil
}

// ListIndex returns compact index entries, optionally limited.
func (kb *KB) ListIndex(limit int) []IndexEntry {
	if limit <= 0 || limit >= len(kb.index) {
		result := make([]IndexEntry, len(kb.index))
		copy(result, kb.index)
		return result
	}
	result := make([]IndexEntry, limit)
	copy(result, kb.index[:limit])
	return result
}

// ParseNationalID parses "#25", "25", or "0025" into an integer ID.
func ParseNationalID(s string) (int, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	return 0, fmt.Errorf("invalid national id: %s", s)
}

func hasType(idx IndexEntry, typeQuery string) bool {
	for _, t := range idx.Types {
		if normalizeName(t) == typeQuery {
			return true
		}
	}
	for _, t := range idx.TypesZh {
		if normalizeName(t) == typeQuery {
			return true
		}
	}
	return false
}

func matchesName(idx IndexEntry, nameQuery string) bool {
	candidates := []string{
		idx.Names.En,
		idx.Names.Zh,
		idx.Names.Ja,
		idx.EntryID,
		fmt.Sprintf("%04d", idx.NationalID),
		strconv.Itoa(idx.NationalID),
	}
	for _, c := range candidates {
		if normalizeName(c) == nameQuery || strings.Contains(normalizeName(c), nameQuery) {
			return true
		}
	}
	return false
}

// FormatEntry renders an entry as human-readable grounded text for LLM observation.
func FormatEntry(e Entry) string {
	var b strings.Builder
	displayName := e.Names.Zh
	if displayName == "" {
		displayName = e.Names.En
	}

	fmt.Fprintf(&b, "#%04d %s", e.NationalID, displayName)
	if e.Form != "" && e.Form != "default" {
		fmt.Fprintf(&b, " [%s]", e.Form)
	}
	b.WriteString("\n")

	if e.Names.En != "" {
		fmt.Fprintf(&b, "英文名: %s\n", e.Names.En)
	}
	if e.Names.Ja != "" {
		fmt.Fprintf(&b, "日文名: %s\n", e.Names.Ja)
	}
	fmt.Fprintf(&b, "世代: %d\n", e.Generation)

	types := e.TypesZh
	if len(types) == 0 {
		types = e.Types
	}
	fmt.Fprintf(&b, "属性: %s\n", strings.Join(types, "/"))

	if e.CategoryZh != "" {
		fmt.Fprintf(&b, "分类: %s\n", e.CategoryZh)
	} else if e.Category != "" {
		fmt.Fprintf(&b, "分类: %s\n", e.Category)
	}

	for _, ab := range e.Abilities {
		label := "特性"
		if ab.Hidden {
			label = "隐藏特性"
		}
		name := ab.NameZh
		if name == "" {
			name = ab.NameEn
		}
		fmt.Fprintf(&b, "%s: %s\n", label, name)
	}

	fmt.Fprintf(&b, "种族值: HP %d / 攻 %d / 防 %d / 特攻 %d / 特防 %d / 速 %d / 合计 %d\n",
		e.BaseStats.HP, e.BaseStats.Attack, e.BaseStats.Defense,
		e.BaseStats.SpecialAttack, e.BaseStats.SpecialDefense, e.BaseStats.Speed,
		e.BaseStats.Total)

	if len(e.Evolution.To) > 0 || len(e.Evolution.From) > 0 {
		b.WriteString("进化: ")
		if len(e.Evolution.From) > 0 {
			from := e.Evolution.From[0]
			name := from.NameZh
			if name == "" {
				name = from.NameEn
			}
			fmt.Fprintf(&b, "由 #%04d %s", from.NationalID, name)
		}
		if len(e.Evolution.To) > 0 {
			to := e.Evolution.To[0]
			name := to.NameZh
			if name == "" {
				name = to.NameEn
			}
			if len(e.Evolution.From) > 0 {
				b.WriteString(" → ")
			}
			fmt.Fprintf(&b, "→ #%04d %s", to.NationalID, name)
		}
		b.WriteString("\n")
	}

	if e.FlavorTextZh != "" {
		text := e.FlavorTextZh
		if len(text) > 300 {
			text = text[:300] + "…"
		}
		fmt.Fprintf(&b, "概述: %s\n", text)
	}

	fmt.Fprintf(&b, "数据来源: %s (pokeapi_id=%d)\n", e.Source.PokeAPIURL, e.Source.PokeAPIID)
	return b.String()
}
