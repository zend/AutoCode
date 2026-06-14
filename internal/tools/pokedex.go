package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zend/AutoCode/internal/knowledge/pokedex"
)

// PokedexInput defines structured query parameters for the pokedex tool.
type PokedexInput struct {
	NationalID    int    `json:"national_id,omitempty"`
	EntryID       string `json:"entry_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Type          string `json:"type,omitempty"`
	Generation    int    `json:"generation,omitempty"`
	Form          string `json:"form,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	List          bool   `json:"list,omitempty"`
	IncludeSprite bool   `json:"include_sprite,omitempty"`
	SpriteOnly    bool   `json:"sprite_only,omitempty"`
	Format        string `json:"format,omitempty"` // "text" (default) or "json"
}

// PokedexTool provides grounded National Pokédex lookups.
type PokedexTool struct {
	kb *pokedex.KB
}

// NewPokedexTool creates a pokedex tool backed by the knowledge base at dataDir.
func NewPokedexTool(dataDir string) (*PokedexTool, error) {
	kb, err := pokedex.Load(dataDir)
	if err != nil {
		return nil, fmt.Errorf("load pokedex kb: %w", err)
	}
	return &PokedexTool{kb: kb}, nil
}

func (t *PokedexTool) Name() string {
	return "pokedex"
}

func (t *PokedexTool) Description() string {
	return `Query the National Pokédex knowledge base (grounded facts only, no hallucination).
Parameters (JSON):
- national_id: National dex number (e.g. 25)
- entry_id: Unique entry key (e.g. "0025" or "0026-alola")
- name: Search by Chinese/English/Japanese name
- type: Filter by type (e.g. "electric", "电")
- generation: Filter by generation (1-9)
- form: Regional form (alola, galar, hisui, paldea)
- limit: Max results for list queries (default 20, max 50)
- list: If true, return compact index listing
- include_sprite: If true, append spritesheet coordinates to results
- sprite_only: If true, return only spritesheet coordinates (requires entry_id or national_id)
- format: "text" (default) or "json"
Only returns data stored in the knowledge base. If not found, explicitly says so.`
}

func (t *PokedexTool) Execute(ctx context.Context, input string) (string, error) {
	_ = ctx

	var params PokedexInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	if params.List {
		return t.listResult(params)
	}

	if params.SpriteOnly {
		return t.spriteResult(params)
	}

	result, err := t.kb.Query(pokedex.QueryParams{
		NationalID: params.NationalID,
		EntryID:    params.EntryID,
		Name:       params.Name,
		Type:       params.Type,
		Generation: params.Generation,
		Form:       params.Form,
		Limit:      params.Limit,
	})
	if err != nil {
		return "", err
	}

	if params.Format == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal result: %w", err)
		}
		return string(data), nil
	}

	return t.formatTextResult(result, params.IncludeSprite), nil
}

func (t *PokedexTool) spriteResult(params PokedexInput) (string, error) {
	atlas := t.kb.SpriteAtlas()
	if atlas == nil {
		return "雪碧图元数据未加载", nil
	}

	entryID := params.EntryID
	if entryID == "" && params.NationalID > 0 {
		entryID = fmt.Sprintf("%04d", params.NationalID)
		if params.Form != "" {
			entryID = fmt.Sprintf("%04d-%s", params.NationalID, params.Form)
		}
	}
	if entryID == "" {
		return "sprite_only 需要 entry_id 或 national_id", nil
	}

	rect, ok := t.kb.GetSprite(entryID)
	if !ok {
		return fmt.Sprintf("雪碧图中未找到条目 %s", entryID), nil
	}

	if params.Format == "json" {
		data, err := json.MarshalIndent(rect, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal sprite: %w", err)
		}
		return string(data), nil
	}

	return pokedex.FormatSpriteRect(*rect, atlas.ImagePath(t.kb.DataDir())), nil
}

func (t *PokedexTool) listResult(params PokedexInput) (string, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	index := t.kb.ListIndex(limit)
	manifest := t.kb.Manifest()

	var b strings.Builder
	fmt.Fprintf(&b, "全国图鉴知识库索引 (共 %d 条, 显示 %d 条)\n", t.kb.EntryCount(), len(index))
	fmt.Fprintf(&b, "版本: %s | 构建: %s\n", manifest.SchemaVersion, manifest.BuiltAt)
	fmt.Fprintf(&b, "防幻觉策略: %s\n\n", manifest.AntiHallucination.Policy)

	for _, item := range index {
		name := item.Names.Zh
		if name == "" {
			name = item.Names.En
		}
		types := item.TypesZh
		if len(types) == 0 {
			types = item.Types
		}
		fmt.Fprintf(&b, "#%04d %s | %s | 世代%d | BST %d",
			item.NationalID, name, strings.Join(types, "/"), item.Generation, item.BaseStatTotal)
		if item.Form != "" && item.Form != "default" {
			fmt.Fprintf(&b, " [%s]", item.Form)
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

func (t *PokedexTool) formatTextResult(result *pokedex.QueryResult, includeSprite bool) string {
	if !result.Found {
		return fmt.Sprintf("未找到匹配条目。\n%s\n来源: %s", result.Message, result.Source)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "找到 %d 条记录 (来源: %s, 仅返回知识库已存储数据)\n\n", result.Count, result.Source)
	for i, entry := range result.Entries {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		b.WriteString(pokedex.FormatEntry(entry))
		if includeSprite {
			if rect, ok := t.kb.GetSprite(entry.EntryID); ok {
				b.WriteString("\n")
				b.WriteString(pokedex.FormatSpriteRect(*rect, t.kb.SpriteAtlas().ImagePath(t.kb.DataDir())))
			}
		}
	}
	return b.String()
}
