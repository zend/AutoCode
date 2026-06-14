#!/usr/bin/env python3
"""Download Pokémon images and assemble a spritesheet with position metadata."""

from __future__ import annotations

import hashlib
import json
import math
import sys
import time
import urllib.error
import urllib.request
from io import BytesIO
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = ROOT / "data" / "pokedex"
ENTRIES_DIR = DATA_DIR / "entries"
INDEX_PATH = DATA_DIR / "index.json"
ASSETS_DIR = DATA_DIR / "assets"
CACHE_DIR = ASSETS_DIR / ".sprite_cache"
SPRITESHEET_PATH = ASSETS_DIR / "spritesheet.png"
METADATA_PATH = DATA_DIR / "spritesheet.json"

CELL_SIZE = 128
SPRITE_STYLE = "home"  # home | official-artwork | front_default
USER_AGENT = "AutoCode-PokedexSpritesheet/1.0"

URL_TEMPLATES = {
    "home": "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/home/{id}.png",
    "official-artwork": "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/official-artwork/{id}.png",
    "front_default": "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/{id}.png",
}


def sprite_url(pokeapi_id: int, style: str = SPRITE_STYLE) -> str:
    template = URL_TEMPLATES.get(style, URL_TEMPLATES["home"])
    return template.format(id=pokeapi_id)


def download_image(url: str, cache_path: Path) -> Image.Image | None:
    if cache_path.exists():
        try:
            return Image.open(cache_path).convert("RGBA")
        except Exception:
            cache_path.unlink(missing_ok=True)

    for attempt in range(4):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = resp.read()
            cache_path.parent.mkdir(parents=True, exist_ok=True)
            cache_path.write_bytes(data)
            return Image.open(BytesIO(data)).convert("RGBA")
        except urllib.error.HTTPError as exc:
            if exc.code == 404:
                return None
            time.sleep(0.5 * (attempt + 1))
        except Exception:
            time.sleep(0.5 * (attempt + 1))
    return None


def fit_image(img: Image.Image, cell: int) -> Image.Image:
    canvas = Image.new("RGBA", (cell, cell), (0, 0, 0, 0))
    img.thumbnail((cell, cell), Image.Resampling.LANCZOS)
    offset = ((cell - img.width) // 2, (cell - img.height) // 2)
    canvas.paste(img, offset, img)
    return canvas


def load_entry_source(entry_id: str) -> dict:
    entry_path = ENTRIES_DIR / f"{entry_id}.json"
    entry = json.loads(entry_path.read_text(encoding="utf-8"))
    return entry.get("source", {})


def build_spritesheet(style: str = SPRITE_STYLE, cell_size: int = CELL_SIZE) -> dict:
    index = json.loads(INDEX_PATH.read_text(encoding="utf-8"))
    entries = sorted(index, key=lambda item: item["entry_id"])

    count = len(entries)
    columns = math.ceil(math.sqrt(count))
    rows = math.ceil(count / columns)
    sheet_w = columns * cell_size
    sheet_h = rows * cell_size
    sheet = Image.new("RGBA", (sheet_w, sheet_h), (32, 32, 32, 255))

    sprites: list[dict] = []
    by_entry_id: dict[str, dict] = {}
    missing: list[str] = []

    print(f"building spritesheet: {count} entries, {columns}x{rows} grid, cell={cell_size}px, style={style}")

    for idx, item in enumerate(entries):
        entry_id = item["entry_id"]
        col = idx % columns
        row = idx // columns
        x = col * cell_size
        y = row * cell_size

        source = load_entry_source(entry_id)
        pokeapi_id = source.get("pokeapi_id", 0)
        url = sprite_url(pokeapi_id, style) if pokeapi_id else ""

        cache_path = CACHE_DIR / style / f"{pokeapi_id}.png"
        img = None
        used_style = style
        if pokeapi_id:
            img = download_image(url, cache_path)
            if img is None:
                for fallback in ("official-artwork", "front_default"):
                    if fallback == style:
                        continue
                    fallback_url = sprite_url(pokeapi_id, fallback)
                    fallback_cache = CACHE_DIR / fallback / f"{pokeapi_id}.png"
                    img = download_image(fallback_url, fallback_cache)
                    if img is not None:
                        used_style = fallback
                        url = fallback_url
                        break

        if img is None:
            missing.append(entry_id)
            img = Image.new("RGBA", (cell_size, cell_size), (64, 64, 64, 255))

        cell_img = fit_image(img, cell_size)
        sheet.paste(cell_img, (x, y), cell_img)

        record = {
            "entry_id": entry_id,
            "national_id": item["national_id"],
            "form": item.get("form", "default"),
            "names": item.get("names", {}),
            "index": idx,
            "col": col,
            "row": row,
            "x": x,
            "y": y,
            "width": cell_size,
            "height": cell_size,
            "pokeapi_id": pokeapi_id,
            "source_url": url,
            "sprite_style": used_style,
        }
        sprites.append(record)
        by_entry_id[entry_id] = record

        if (idx + 1) % 50 == 0 or idx + 1 == count:
            print(f"  [{idx + 1:4d}/{count}] {entry_id} {item.get('names', {}).get('zh') or item.get('names', {}).get('en', '')}")

    ASSETS_DIR.mkdir(parents=True, exist_ok=True)
    sheet.save(SPRITESHEET_PATH, optimize=True)

    content_hash = hashlib.sha256(SPRITESHEET_PATH.read_bytes()).hexdigest()
    metadata = {
        "schema_version": "1.0.0",
        "image": "assets/spritesheet.png",
        "image_width": sheet_w,
        "image_height": sheet_h,
        "cell_width": cell_size,
        "cell_height": cell_size,
        "columns": columns,
        "rows": rows,
        "entry_count": count,
        "sprite_style": style,
        "content_sha256": content_hash,
        "built_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "missing_entries": missing,
        "sprites": sprites,
        "by_entry_id": by_entry_id,
    }
    METADATA_PATH.write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    print(f"\ndone: {SPRITESHEET_PATH} ({sheet_w}x{sheet_h})")
    print(f"metadata: {METADATA_PATH}")
    print(f"sha256: {content_hash[:16]}...")
    if missing:
        print(f"warning: {len(missing)} entries missing images: {missing[:10]}...", file=sys.stderr)
    return metadata


def main() -> int:
    style = SPRITE_STYLE
    if len(sys.argv) > 1:
        style = sys.argv[1]
    if style not in URL_TEMPLATES:
        print(f"unknown style: {style}, choose from {list(URL_TEMPLATES)}", file=sys.stderr)
        return 1
    build_spritesheet(style=style)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
