#!/usr/bin/env python3
"""从神奇宝贝百科抓取全国图鉴宝可梦完整资料并生成 Markdown。"""

from __future__ import annotations

import json
import re
import sys
import time
from pathlib import Path

from bs4 import BeautifulSoup
from playwright.sync_api import sync_playwright

from scrape_kanto_pokemon import BASE_URL, clean_text, parse_detail_page

LIST_URL = (
    f"{BASE_URL}/wiki/"
    "%E5%AE%9D%E5%8F%AF%E6%A2%A6%E5%88%97%E8%A1%A8%EF%BC%88%E6%8C%89%E5%85%A8%E5%9B%BD%E5%9B%BE%E9%89%B4%E7%BC%96%E5%8F%B7%EF%BC%89"
)
DOCS_DIR = Path(__file__).resolve().parent.parent / "docs"
OUTPUT_PATH = DOCS_DIR / "全国宝可梦完整资料.md"
LIST_CACHE_PATH = DOCS_DIR / ".national_pokemon_list.json"
DETAIL_CACHE_PATH = DOCS_DIR / ".national_pokemon_details.json"
KANTO_CACHE_PATH = DOCS_DIR / ".kanto_pokemon_cache.json"

GENERATION_RANGES = [
    (1, 1, 151, "第一世代"),
    (2, 152, 251, "第二世代"),
    (3, 252, 386, "第三世代"),
    (4, 387, 493, "第四世代"),
    (5, 494, 649, "第五世代"),
    (6, 650, 721, "第六世代"),
    (7, 722, 809, "第七世代"),
    (8, 810, 905, "第八世代"),
    (9, 906, 1025, "第九世代"),
]


def generation_label(national_num: int) -> str:
    for _, start, end, label in GENERATION_RANGES:
        if start <= national_num <= end:
            return label
    return "未知世代"


def format_name(name: str) -> str:
    lines = [ln.strip() for ln in name.split("\n") if ln.strip() and ln.strip() != "*"]
    if not lines:
        return name.replace("*", "").strip()
    if len(lines) == 1:
        return lines[0].replace("*", "")
    base = lines[0].replace("*", "")
    form = " ".join(lines[1:]).replace("*", "")
    if "的样子" in form and not form.endswith("）"):
        form = form if form.endswith("的样子") else form
        return f"{base}（{form}）"
    return f"{base}（{form}）"


def entry_key(entry: dict) -> str:
    return f"{entry['national']}:{entry['name']}"


def parse_national_list(html: str) -> list[dict]:
    soup = BeautifulSoup(html, "lxml")
    main = soup.select_one("#mw-content-text")
    if not main:
        return []

    entries: list[dict] = []
    for table in main.select("table"):
        for row in table.select("tr"):
            cells = row.select("td")
            if len(cells) < 7:
                continue
            national = cells[0].get_text(strip=True)
            if not re.match(r"#\d{4}", national):
                continue

            href = ""
            for cell in cells[1:4]:
                link = cell.select_one("a[href*='/wiki/']")
                if link:
                    href = link.get("href", "")
                    break

            zh = cells[3].get_text("\n", strip=True)
            jp = cells[4].get_text(strip=True) if len(cells) > 4 else ""
            en = cells[5].get_text(strip=True) if len(cells) > 5 else ""
            types = [
                clean_text(c.get_text(strip=True))
                for c in cells[6:]
                if clean_text(c.get_text(strip=True)) and not c.get_text(strip=True).startswith("[[")
            ]

            num = int(national.lstrip("#"))
            gen_label = generation_label(num)
            entries.append(
                {
                    "national": national,
                    "national_num": num,
                    "name": zh,
                    "display_name": format_name(zh),
                    "jp": jp,
                    "en": en,
                    "types": types,
                    "href": href,
                    "generation": f"{gen_label}宝可梦",
                }
            )
    return entries


def load_detail_cache() -> dict[str, dict]:
    cache: dict[str, dict] = {}
    if DETAIL_CACHE_PATH.exists():
        cache = json.loads(DETAIL_CACHE_PATH.read_text(encoding="utf-8"))
    if KANTO_CACHE_PATH.exists():
        for item in json.loads(KANTO_CACHE_PATH.read_text(encoding="utf-8")):
            href = item.get("href", "")
            if href and href not in cache:
                cache[href] = item
    return cache


def save_detail_cache(cache: dict[str, dict]) -> None:
    DETAIL_CACHE_PATH.parent.mkdir(parents=True, exist_ok=True)
    DETAIL_CACHE_PATH.write_text(json.dumps(cache, ensure_ascii=False, indent=2), encoding="utf-8")


def merge_entry(basic: dict, detail: dict) -> dict:
    names = detail.get("names", {})
    if basic.get("jp"):
        names["jp"] = basic["jp"]
    if basic.get("en"):
        names["en"] = basic["en"]
    names["zh"] = basic.get("display_name") or basic.get("name", "")

    types = basic.get("types") or detail.get("types", [])
    types = [t for t in types if t and "属性" not in t and not t.startswith("[[")]

    merged = {
        **basic,
        "name": names["zh"],
        "names": names,
        "types": types or detail.get("types", []),
        "category": detail.get("category", ""),
        "abilities": detail.get("abilities", []),
        "hidden_ability": detail.get("hidden_ability", ""),
        "height": detail.get("height", ""),
        "weight": detail.get("weight", ""),
        "gender_ratio": detail.get("gender_ratio", ""),
        "catch_rate": detail.get("catch_rate", ""),
        "color": detail.get("color", ""),
        "experience": detail.get("experience", ""),
        "breeding": detail.get("breeding", ""),
        "base_exp": detail.get("base_exp", ""),
        "ev_yield": detail.get("ev_yield", {}),
        "regional_dex": detail.get("regional_dex", {}),
        "stats": detail.get("stats", {}),
        "overview": detail.get("overview", ""),
        "evolution": detail.get("evolution", ""),
        "pokedex": detail.get("pokedex", []),
        "other_names": detail.get("other_names", []),
    }
    return merged


def render_entry(p: dict) -> str:
    lines: list[str] = []
    zh = p.get("display_name") or p.get("name", "")
    names = p.get("names", {})
    jp = names.get("jp") or p.get("jp", "")
    en = names.get("en") or p.get("en", "")

    lines.append(f"### {p['national']} {zh}")
    lines.append("")
    if jp or en:
        lines.append(f"- **日文名**: {jp or '—'}")
        lines.append(f"- **英文名**: {en or '—'}")
    lines.append(f"- **世代**: {p.get('generation', '—')}")
    types = " / ".join(p.get("types", []))
    lines.append(f"- **属性**: {types or '—'}")
    if p.get("category"):
        lines.append(f"- **分类**: {p['category']}")
    if p.get("abilities"):
        lines.append(f"- **特性**: {' / '.join(p['abilities'])}")
    if p.get("hidden_ability"):
        lines.append(f"- **隐藏特性**: {p['hidden_ability']}")
    if p.get("height"):
        lines.append(f"- **身高**: {p['height']}")
    if p.get("weight"):
        lines.append(f"- **体重**: {p['weight']}")
    if p.get("gender_ratio"):
        lines.append(f"- **性别比例**: {p['gender_ratio']}")
    if p.get("catch_rate"):
        lines.append(f"- **捕获率**: {p['catch_rate']}")
    if p.get("color"):
        lines.append(f"- **图鉴颜色**: {p['color']}")
    if p.get("experience"):
        lines.append(f"- **经验值**: {p['experience']}")
    if p.get("base_exp"):
        lines.append(f"- **基础经验值**: {p['base_exp']}")
    if p.get("breeding"):
        lines.append(f"- **培育**: {p['breeding']}")
    if p.get("ev_yield"):
        ev = ", ".join(f"{k} +{v}" for k, v in p["ev_yield"].items() if v != "0")
        if ev:
            lines.append(f"- **击败后可获得努力值**: {ev}")
    if p.get("regional_dex"):
        regional = ", ".join(
            f"{k} #{v}" for k, v in p["regional_dex"].items() if v and v not in {"000", "0", "—", "-"}
        )
        if regional:
            lines.append(f"- **地区图鉴编号**: {regional}")

    stats = p.get("stats", {})
    if stats:
        lines.append("")
        lines.append("#### 种族值")
        lines.append("")
        lines.append("| 能力 | 数值 |")
        lines.append("| --- | ---: |")
        for label, key in [
            ("HP", "hp"),
            ("攻击", "attack"),
            ("防御", "defense"),
            ("特攻", "sp_attack"),
            ("特防", "sp_defense"),
            ("速度", "speed"),
        ]:
            if key in stats:
                lines.append(f"| {label} | {stats[key]} |")
        if "total" in stats:
            lines.append(f"| **合计** | **{stats['total']}** |")

    if p.get("overview"):
        lines.append("")
        lines.append("#### 概述")
        lines.append("")
        lines.append(p["overview"])

    if p.get("evolution"):
        lines.append("")
        lines.append("#### 进化")
        lines.append("")
        lines.append(p["evolution"])

    if p.get("pokedex"):
        lines.append("")
        lines.append("#### 图鉴介绍")
        lines.append("")
        for entry in p["pokedex"]:
            lines.append(f"- **{entry['game']}**: {entry['description']}")

    wiki_path = p.get("href", "")
    if wiki_path:
        lines.append("")
        lines.append(f"- **百科链接**: [{zh}]({BASE_URL}{wiki_path})")

    lines.append("")
    return "\n".join(lines)


def render_markdown(entries: list[dict]) -> str:
    unique_species = len({e["national_num"] for e in entries})
    lines = [
        "# 全国图鉴宝可梦完整资料",
        "",
        "> 资料来源：[神奇宝贝百科 - 宝可梦列表（按全国图鉴编号）](https://wiki.52poke.com/wiki/%E5%AE%9D%E5%8F%AF%E6%A2%A6%E5%88%97%E8%A1%A8%EF%BC%88%E6%8C%89%E5%85%A8%E5%9B%BD%E5%9B%BE%E9%89%B4%E7%BC%96%E5%8F%B7%EF%BC%89)",
        "",
        f"本文档收录全国图鉴 **{unique_species}** 种、共 **{len(entries)}** 条记录"
        "（含阿罗拉、伽勒尔、洗翠、帕底亚等地区形态）。",
        "",
        "## 目录",
        "",
    ]

    for _, _, _, gen_label in GENERATION_RANGES:
        gen_entries = [e for e in entries if e.get("generation") == f"{gen_label}宝可梦" or e.get("generation", "").startswith(gen_label)]
        if not gen_entries:
            gen_entries = [e for e in entries if generation_label(e["national_num"]) == gen_label]
        if not gen_entries:
            continue
        lines.append(f"### {gen_label}")
        lines.append("")
        for p in gen_entries:
            name = p.get("display_name") or p.get("name", "")
            anchor = f"{p['national'].lstrip('#')}-{name}"
            lines.append(f"- [{p['national']} {name}](#{anchor})")
        lines.append("")

    lines.extend(["---", "", "## 全国图鉴一览表", ""])
    lines.append("| 全国编号 | 宝可梦 | 属性 | 分类 | 世代 | 种族值合计 |")
    lines.append("| ---: | --- | --- | --- | --- | ---: |")
    for p in entries:
        types = " / ".join(p.get("types", []))
        total = p.get("stats", {}).get("total", "—")
        gen = p.get("generation", generation_label(p["national_num"])).replace("宝可梦", "")
        name = p.get("display_name") or p.get("name", "")
        lines.append(
            f"| {p['national']} | {name} | {types} | {p.get('category', '—')} | {gen} | {total} |"
        )

    lines.extend(["", "---", "", "## 详细信息", ""])
    current_gen = ""
    for p in entries:
        gen = p.get("generation", generation_label(p["national_num"]))
        if gen != current_gen:
            current_gen = gen
            lines.append(f"## {gen}")
            lines.append("")
        lines.append(render_entry(p))

    lines.extend(
        [
            "---",
            "",
            f"*本文档由脚本自动生成，共收录 {len(entries)} 条全国图鉴记录。*",
            "",
        ]
    )
    return "\n".join(lines)


def scrape(force: bool = False) -> list[dict]:
    detail_cache = {} if force else load_detail_cache()
    if force:
        print("Force refresh: clearing detail cache")

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=False,
            channel="chrome",
            args=["--disable-blink-features=AutomationControlled"],
        )
        page = browser.new_context(locale="zh-CN", viewport={"width": 1280, "height": 900}).new_page()
        page.add_init_script(
            "Object.defineProperty(navigator, 'webdriver', {get: () => undefined})"
        )

        def fetch(url: str) -> str:
            for attempt in range(3):
                try:
                    page.goto(url, wait_until="domcontentloaded", timeout=120000)
                    time.sleep(2)
                    if "请稍候" in page.title() or "Just a moment" in page.title():
                        time.sleep(5)
                        continue
                    main = page.query_selector("#mw-content-text")
                    if main and "目前没有内容" not in main.inner_text()[:200]:
                        return page.content()
                except Exception as exc:
                    print(f"  retry {attempt + 1}: {exc}")
                    time.sleep(3)
            raise RuntimeError(f"failed to fetch {url}")

        if LIST_CACHE_PATH.exists() and not force:
            basics = json.loads(LIST_CACHE_PATH.read_text(encoding="utf-8"))
            print(f"Loaded {len(basics)} list entries from cache")
        else:
            print("Fetching national list page...")
            html = fetch(LIST_URL)
            basics = parse_national_list(html)
            LIST_CACHE_PATH.write_text(json.dumps(basics, ensure_ascii=False, indent=2), encoding="utf-8")
            print(f"Found {len(basics)} list entries")

        hrefs = sorted({b["href"] for b in basics if b.get("href")})
        print(f"Unique wiki pages to fetch: {len(hrefs)} (cached: {len(detail_cache)})")

        for i, href in enumerate(hrefs):
            if href in detail_cache and not force:
                continue
            sample = next(b for b in basics if b["href"] == href)
            url = BASE_URL + href
            print(f"[{i + 1}/{len(hrefs)}] Fetching {sample['national']} {sample.get('display_name', sample['name'])}...")
            try:
                html = fetch(url)
                detail = parse_detail_page(
                    html,
                    {
                        "national": sample["national"],
                        "name": sample.get("display_name", sample["name"]),
                        "href": href,
                        "types": sample.get("types", []),
                        "jp": sample.get("jp", ""),
                        "en": sample.get("en", ""),
                    },
                )
                detail_cache[href] = detail
                save_detail_cache(detail_cache)
            except Exception as exc:
                print(f"  ERROR: {exc}")

        browser.close()

    results = []
    for basic in basics:
        href = basic.get("href", "")
        detail = detail_cache.get(href, {})
        results.append(merge_entry(basic, detail))

    results.sort(key=lambda x: (x["national_num"], x.get("display_name", x.get("name", ""))))
    return results


def main() -> None:
    force = "--force" in sys.argv
    render_only = "--render-only" in sys.argv

    if render_only:
        if not LIST_CACHE_PATH.exists():
            print("List cache missing. Run without --render-only first.")
            sys.exit(1)
        basics = json.loads(LIST_CACHE_PATH.read_text(encoding="utf-8"))
        detail_cache = load_detail_cache()
        entries = [merge_entry(b, detail_cache.get(b.get("href", ""), {})) for b in basics]
        entries.sort(key=lambda x: (x["national_num"], x.get("display_name", x.get("name", ""))))
    else:
        entries = scrape(force=force)

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    md = render_markdown(entries)
    OUTPUT_PATH.write_text(md, encoding="utf-8")
    print(f"Written {OUTPUT_PATH} ({len(md):,} chars, {len(entries)} entries)")


if __name__ == "__main__":
    main()
