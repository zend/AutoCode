#!/usr/bin/env python3
"""Build structured National Pokédex knowledge base from PokeAPI + Chinese wiki data."""

from __future__ import annotations

import hashlib
import json
import re
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = ROOT / "data" / "pokedex"
ENTRIES_DIR = DATA_DIR / "entries"
CACHE_DIR = DATA_DIR / ".build_cache"
WIKI_MD = ROOT / "docs" / "全国宝可梦完整资料.md"

API_BASE = "https://pokeapi.co/api/v2"
REQUEST_DELAY = 0.15

GENERATION_RANGES = [
    (1, 151),
    (2, 251),
    (3, 386),
    (4, 493),
    (5, 649),
    (6, 721),
    (7, 809),
    (8, 905),
    (9, 1025),
]

TYPE_ZH = {
    "normal": "一般", "fire": "火", "water": "水", "electric": "电",
    "grass": "草", "ice": "冰", "fighting": "格斗", "poison": "毒",
    "ground": "地面", "flying": "飞行", "psychic": "超能力", "bug": "虫",
    "rock": "岩石", "ghost": "幽灵", "dragon": "龙", "dark": "恶",
    "steel": "钢", "fairy": "妖精", "stellar": "太晶",
}

COLOR_ZH = {
    "black": "黑色", "blue": "蓝色", "brown": "褐色", "gray": "灰色",
    "green": "绿色", "pink": "粉红色", "purple": "紫色", "red": "红色",
    "white": "白色", "yellow": "黄色",
}

EGG_GROUP_ZH = {
    "monster": "怪兽", "water1": "水中1", "bug": "虫", "flying": "飞行",
    "ground": "陆上", "fairy": "妖精", "plant": "植物", "humanshape": "人型",
    "water3": "水中3", "mineral": "矿物", "indeterminate": "不定形",
    "water2": "水中2", "ditto": "百变怪", "dragon": "龙", "no-eggs": "未发现",
}

STAT_MAP = {
    "hp": "hp",
    "attack": "attack",
    "defense": "defense",
    "special-attack": "special_attack",
    "special-defense": "special_defense",
    "speed": "speed",
}

EV_STAT_MAP = {
    "hp": "hp", "attack": "attack", "defense": "defense",
    "special-attack": "special_attack", "special-defense": "special_defense",
    "speed": "speed",
}

REGIONAL_FORM_PATTERNS = [
    (r"（阿罗拉的样子）", "alola"),
    (r"（伽勒尔的样子）", "galar"),
    (r"（洗翠的样子）", "hisui"),
    (r"（帕底亚的样子）", "paldea"),
]


def generation_for(national_id: int) -> int:
    for gen, end in GENERATION_RANGES:
        if national_id <= end:
            return gen
    return 9


def api_get(path: str) -> dict:
    cache_path = CACHE_DIR / hashlib.sha256(path.encode()).hexdigest()
    if cache_path.exists():
        return json.loads(cache_path.read_text(encoding="utf-8"))

    url = f"{API_BASE}/{path.lstrip('/')}"
    for attempt in range(5):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "AutoCode-PokedexKB/1.0"})
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = json.loads(resp.read().decode("utf-8"))
            CACHE_DIR.mkdir(parents=True, exist_ok=True)
            cache_path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
            time.sleep(REQUEST_DELAY)
            return data
        except urllib.error.HTTPError as exc:
            if exc.code == 404:
                raise
            time.sleep(2 ** attempt)
        except Exception:
            time.sleep(2 ** attempt)
    raise RuntimeError(f"failed to fetch {url}")


def localized_name(names: list[dict], lang: str) -> str:
    for item in names:
        if item.get("language", {}).get("name") == lang:
            return item["name"]
    return ""


def parse_wiki_markdown(path: Path) -> dict[str, dict]:
    if not path.exists():
        print(f"warning: wiki markdown not found at {path}, Chinese names will be missing", file=sys.stderr)
        return {}

    text = path.read_text(encoding="utf-8")
    summary: dict[str, dict] = {}

    for line in text.splitlines():
        m = re.match(
            r"^\| (#\d{4}) \| ([^|]+) \| ([^|]+) \| ([^|]+) \| ([^|]+) \| (\d+) \|",
            line,
        )
        if not m:
            continue
        national_str, zh, types, category, gen_label, total = m.groups()
        national = int(national_str.lstrip("#"))
        zh = zh.strip()
        form = "default"
        for pattern, key in REGIONAL_FORM_PATTERNS:
            if re.search(pattern, zh):
                form = key
                break
        entry_id = f"{national:04d}" if form == "default" else f"{national:04d}-{form}"
        summary[entry_id] = {
            "national_id": national,
            "form": form,
            "names_zh": zh,
            "types_zh": [t.strip() for t in re.split(r"\s*/\s*", types.strip()) if t.strip()],
            "category_zh": category.strip(),
            "generation_label": gen_label.strip(),
            "base_stat_total": int(total),
        }

    details: dict[str, dict] = {}
    sections = re.split(r"\n### (#\d{4}[^\n]*)\n", text)
    for i in range(1, len(sections), 2):
        header = sections[i].strip()
        body = sections[i + 1] if i + 1 < len(sections) else ""
        num_m = re.match(r"#(\d{4})\s*(.*)", header)
        if not num_m:
            continue
        national = int(num_m.group(1))
        form_suffix = num_m.group(2).strip()
        form = "default"
        for pattern, key in REGIONAL_FORM_PATTERNS:
            if re.search(pattern, form_suffix):
                form = key
                break
        entry_id = f"{national:04d}" if form == "default" else f"{national:04d}-{form}"

        detail: dict = {}
        for key, pattern in [
            ("name_ja", r"\*\*日文名\*\*:\s*(.+)"),
            ("name_en", r"\*\*英文名\*\*:\s*(.+)"),
            ("category_zh", r"\*\*分类\*\*:\s*(.+)"),
            ("abilities_zh", r"\*\*特性\*\*:\s*(.+)"),
            ("hidden_ability_zh", r"\*\*隐藏特性\*\*:\s*(.+)"),
            ("height_m", r"\*\*身高\*\*:\s*([\d.]+)m"),
            ("weight_kg", r"\*\*体重\*\*:\s*([\d.]+)kg"),
            ("capture_rate", r"\*\*捕获率\*\*:\s*(\d+)"),
            ("color_zh", r"\*\*图鉴颜色\*\*:\s*(.+)"),
            ("base_experience", r"\*\*基础经验值\*\*:\s*([\d,]+)"),
            ("regional_dex_raw", r"\*\*地区图鉴编号\*\*:\s*(.+)"),
        ]:
            m = re.search(pattern, body)
            if m:
                detail[key] = m.group(1).strip()

        stats: dict[str, int] = {}
        for stat_line in re.finditer(r"\| (HP|攻击|防御|特攻|特防|速度) \| (\d+) \|", body):
            stat_zh, val = stat_line.groups()
            stat_key = {
                "HP": "hp", "攻击": "attack", "防御": "defense",
                "特攻": "special_attack", "特防": "special_defense", "速度": "speed",
            }[stat_zh]
            stats[stat_key] = int(val)

        overview_m = re.search(r"#### 概述\n\n(.+?)(?:\n\n#### |\Z)", body, re.S)
        if overview_m:
            detail["flavor_text_zh"] = re.sub(r"\s+", " ", overview_m.group(1).strip())[:2000]

        if stats:
            detail["base_stats"] = stats
            detail["base_stats"]["total"] = sum(stats.values())

        if "regional_dex_raw" in detail:
            regional: dict[str, int] = {}
            for part in re.split(r",\s*", detail["regional_dex_raw"]):
                m = re.match(r"(\S+)\s+#(\d+)", part)
                if m:
                    regional[m.group(1)] = int(m.group(2))
            detail["regional_dex"] = regional

        details[entry_id] = detail

    merged: dict[str, dict] = {}
    for entry_id, base in summary.items():
        merged[entry_id] = {**base, **details.get(entry_id, {})}
    for entry_id, detail in details.items():
        if entry_id not in merged:
            merged[entry_id] = detail
    return merged


@dataclass
class EvolutionCache:
    chains: dict[int, dict] = field(default_factory=dict)

    def get_chain(self, chain_id: int) -> dict:
        if chain_id not in self.chains:
            self.chains[chain_id] = api_get(f"evolution-chain/{chain_id}")
        return self.chains[chain_id]


def build_evolution(species: dict, evo_cache: EvolutionCache, species_zh: dict[int, str]) -> dict:
    from_list = []
    to_list = []

    evolves_from = species.get("evolves_from_species")
    if evolves_from:
        parent = api_get(evolves_from["url"].replace(API_BASE + "/", ""))
        from_list.append({
            "national_id": parent["id"],
            "name_en": parent["name"],
            "name_zh": species_zh.get(parent["id"], ""),
            "trigger": "",
        })

    chain_url = species.get("evolution_chain", {}).get("url", "")
    if chain_url:
        chain_id = int(chain_url.rstrip("/").split("/")[-1])
        chain = evo_cache.get_chain(chain_id)
        national_id = species["order"]

        def find_node(node: dict, target_id: int) -> dict | None:
            sp = api_get(node["species"]["url"].replace(API_BASE + "/", ""))
            if sp["id"] == target_id:
                return node
            for child in node.get("evolves_to", []):
                found = find_node(child, target_id)
                if found:
                    return found
            return None

        node = find_node(chain["chain"], species["id"])
        if node:
            for child in node.get("evolves_to", []):
                child_species = api_get(child["species"]["url"].replace(API_BASE + "/", ""))
                trigger = ""
                details = child.get("evolution_details") or []
                if details:
                    trigger = details[0].get("trigger", {}).get("name", "")
                to_list.append({
                    "national_id": child_species["id"],
                    "name_en": child_species["name"],
                    "name_zh": species_zh.get(child_species["id"], ""),
                    "trigger": trigger,
                })

    if not from_list and not to_list:
        return {}
    return {"from": from_list, "to": to_list}


def gender_ratio_from_species(species: dict) -> dict:
    rate = species.get("gender_rate", -1)
    if rate == -1:
        return {"genderless": True}
    female = rate / 8 * 100
    male = 100 - female
    return {"male_percent": male, "female_percent": female, "genderless": False}


def form_key_from_name(pokemon_name: str, species_name: str) -> str:
    if pokemon_name == species_name:
        return "default"
    suffix = pokemon_name.removeprefix(species_name).lstrip("-")
    return suffix or "default"


def build_entry(
    pokemon: dict,
    species: dict,
    national_id: int,
    form: str,
    wiki: dict,
    evo_cache: EvolutionCache,
    species_zh: dict[int, str],
) -> dict:
    entry_id = f"{national_id:04d}" if form == "default" else f"{national_id:04d}-{form}"
    wiki_entry = wiki.get(entry_id, {})

    names = {
        "en": localized_name(species["names"], "en") or pokemon["name"],
        "ja": localized_name(species["names"], "ja"),
        "zh": wiki_entry.get("names_zh", ""),
    }

    types = [t["type"]["name"] for t in pokemon["types"]]
    types_zh = wiki_entry.get("types_zh") or [TYPE_ZH.get(t, t) for t in types]

    abilities = []
    for item in pokemon["abilities"]:
        ab_name = item["ability"]["name"]
        abilities.append({
            "name_en": ab_name,
            "name_zh": "",
            "hidden": item["is_hidden"],
        })

    base_stats = {"hp": 0, "attack": 0, "defense": 0, "special_attack": 0, "special_defense": 0, "speed": 0}
    for stat in pokemon["stats"]:
        key = STAT_MAP.get(stat["stat"]["name"])
        if key:
            base_stats[key] = stat["base_stat"]
    base_stats["total"] = sum(v for k, v in base_stats.items() if k != "total")

    if wiki_entry.get("base_stats"):
        base_stats = wiki_entry["base_stats"]

    ev_yield = {}
    for stat in pokemon["stats"]:
        key = EV_STAT_MAP.get(stat["stat"]["name"])
        if key and stat["effort"]:
            ev_yield[key] = stat["effort"]

    gen_match = re.search(r"generation-([ivx]+)", species["generation"]["name"])
    roman = {"i": 1, "ii": 2, "iii": 3, "iv": 4, "v": 5, "vi": 6, "vii": 7, "viii": 8, "ix": 9}
    gen_num = roman.get(gen_match.group(1), generation_for(national_id)) if gen_match else generation_for(national_id)

    flavor_en = ""
    for entry in reversed(species.get("flavor_text_entries", [])):
        if entry.get("language", {}).get("name") == "en":
            flavor_en = re.sub(r"[\f\n\r]+", " ", entry["flavor_text"]).strip()
            break

    color = species.get("color", {}).get("name", "")
    entry = {
        "entry_id": entry_id,
        "national_id": national_id,
        "form": form,
        "names": names,
        "generation": gen_num,
        "types": types,
        "types_zh": types_zh,
        "category": species.get("genera", [{}])[0].get("genus", "") if species.get("genera") else "",
        "category_zh": wiki_entry.get("category_zh", ""),
        "abilities": abilities,
        "height_m": pokemon["height"] / 10,
        "weight_kg": pokemon["weight"] / 10,
        "gender_ratio": gender_ratio_from_species(species),
        "capture_rate": species.get("capture_rate", 0),
        "color": color,
        "color_zh": wiki_entry.get("color_zh") or COLOR_ZH.get(color, ""),
        "base_experience": pokemon.get("base_experience") or wiki_entry.get("base_experience"),
        "growth_rate": species.get("growth_rate", {}).get("name", ""),
        "egg_groups": [g["name"] for g in species.get("egg_groups", [])],
        "egg_groups_zh": [EGG_GROUP_ZH.get(g["name"], g["name"]) for g in species.get("egg_groups", [])],
        "base_stats": base_stats,
        "ev_yield": ev_yield,
        "evolution": build_evolution(species, evo_cache, species_zh),
        "regional_dex": wiki_entry.get("regional_dex", {}),
        "flavor_text_zh": wiki_entry.get("flavor_text_zh", ""),
        "flavor_text_en": flavor_en,
        "is_default_form": form == "default",
        "source": {
            "pokeapi_id": pokemon["id"],
            "pokeapi_species_id": species["id"],
            "pokeapi_url": f"{API_BASE}/pokemon/{pokemon['id']}",
        },
    }

    if wiki_entry.get("abilities_zh"):
        for i, ab in enumerate(entry["abilities"]):
            if not ab["hidden"]:
                ab["name_zh"] = wiki_entry["abilities_zh"]
                break
    if wiki_entry.get("hidden_ability_zh"):
        for ab in entry["abilities"]:
            if ab["hidden"]:
                ab["name_zh"] = wiki_entry["hidden_ability_zh"]

    return entry


def main() -> int:
    print("loading wiki markdown for Chinese names...")
    wiki = parse_wiki_markdown(WIKI_MD)

    print("fetching national dex list...")
    dex = api_get("pokedex/national")
    species_list = dex["pokemon_entries"]

    species_zh: dict[int, str] = {}
    for entry_id, data in wiki.items():
        if data.get("form", "default") == "default" and data.get("names_zh"):
            species_zh[data["national_id"]] = data["names_zh"]

    evo_cache = EvolutionCache()
    entries: list[dict] = []
    seen_ids: set[str] = set()

    print(f"building {len(species_list)} base species entries...")
    for item in species_list:
        national_id = item["entry_number"]
        species_name = item["pokemon_species"]["name"]
        species = api_get(f"pokemon-species/{species_name}")

        default_variety = None
        for variety in species.get("varieties", []):
            if variety.get("is_default"):
                default_variety = variety
                break
        if default_variety is None and species.get("varieties"):
            default_variety = species["varieties"][0]

        pokemon = api_get(f"pokemon/{default_variety['pokemon']['name']}")
        entry = build_entry(pokemon, species, national_id, "default", wiki, evo_cache, species_zh)
        entry_id = entry["entry_id"]
        if entry_id not in seen_ids:
            seen_ids.add(entry_id)
            entries.append(entry)
            print(f"  [{len(entries):4d}] {entry_id} {entry['names'].get('zh') or entry['names']['en']}")

    regional_forms = sorted(
        (entry_id, data)
        for entry_id, data in wiki.items()
        if data.get("form", "default") != "default" and entry_id not in seen_ids
    )
    if regional_forms:
        print(f"adding {len(regional_forms)} regional form entries...")
        for entry_id, wiki_entry in regional_forms:
            national_id = wiki_entry["national_id"]
            form = wiki_entry["form"]
            species_name = dex["pokemon_entries"][national_id - 1]["pokemon_species"]["name"]
            species = api_get(f"pokemon-species/{species_name}")
            pokemon_name = f"{species_name}-{form}"
            try:
                pokemon = api_get(f"pokemon/{pokemon_name}")
            except Exception:
                print(f"  skip {entry_id}: pokemon {pokemon_name} not in PokeAPI", file=sys.stderr)
                continue
            entry = build_entry(pokemon, species, national_id, form, wiki, evo_cache, species_zh)
            if entry["entry_id"] in seen_ids:
                continue
            seen_ids.add(entry["entry_id"])
            entries.append(entry)
            print(f"  [{len(entries):4d}] {entry['entry_id']} {entry['names'].get('zh') or entry['names']['en']}")

    entries.sort(key=lambda e: (e["national_id"], e["form"]))

    ENTRIES_DIR.mkdir(parents=True, exist_ok=True)
    for old in ENTRIES_DIR.glob("*.json"):
        old.unlink()

    index = []
    for entry in entries:
        path = ENTRIES_DIR / f"{entry['entry_id']}.json"
        path.write_text(json.dumps(entry, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        index.append({
            "entry_id": entry["entry_id"],
            "national_id": entry["national_id"],
            "form": entry["form"],
            "names": entry["names"],
            "types": entry["types"],
            "types_zh": entry.get("types_zh", []),
            "generation": entry["generation"],
            "base_stat_total": entry["base_stats"]["total"],
        })

    index_path = DATA_DIR / "index.json"
    index_path.write_text(json.dumps(index, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    content_hash = hashlib.sha256(index_path.read_bytes()).hexdigest()
    for entry in entries:
        content_hash = hashlib.sha256(
            (content_hash + (ENTRIES_DIR / f"{entry['entry_id']}.json").read_bytes().hex()).encode()
        ).hexdigest()

    manifest = {
        "schema_version": "1.0.0",
        "name": "national-pokedex",
        "description": "全国图鉴结构化知识库，供大模型 grounded 查询，禁止编造库外数据",
        "entry_count": len(entries),
        "species_count": len({e["national_id"] for e in entries}),
        "sources": [
            {"name": "PokeAPI", "url": "https://pokeapi.co", "license": "fair use"},
            {"name": "神奇宝贝百科", "url": "https://wiki.52poke.com", "note": "中文名称与描述"},
        ],
        "content_sha256": content_hash,
        "built_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "anti_hallucination": {
            "policy": "所有事实必须来自本知识库 JSON 文件；查询工具仅返回已存储字段",
            "not_found_message": "知识库中未找到该条目，请勿编造",
        },
    }
    (DATA_DIR / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    print(f"\ndone: {len(entries)} entries written to {ENTRIES_DIR}")
    print(f"index: {index_path}")
    print(f"manifest sha256: {content_hash[:16]}...")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
