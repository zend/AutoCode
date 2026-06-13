#!/usr/bin/env python3
"""从神奇宝贝百科抓取关都图鉴宝可梦完整资料并生成 Markdown。"""

from __future__ import annotations

import json
import re
import sys
import time
from pathlib import Path

from bs4 import BeautifulSoup, NavigableString, Tag
from playwright.sync_api import sync_playwright

BASE_URL = "https://wiki.52poke.com"
LIST_URL = (
    f"{BASE_URL}/wiki/"
    "%E5%AE%9D%E5%8F%AF%E6%A2%A6%E5%88%97%E8%A1%A8%EF%BC%88%E6%8C%89%E5%85%B3%E9%83%BD%E5%9B%BE%E9%89%B4%E7%BC%96%E5%8F%B7%EF%BC%89"
)
OUTPUT_PATH = Path(__file__).resolve().parent.parent / "docs" / "关都宝可梦完整资料.md"
CACHE_PATH = Path(__file__).resolve().parent.parent / "docs" / ".kanto_pokemon_cache.json"

STAT_NAMES = ["HP", "攻击", "防御", "特攻", "特防", "速度"]
STAT_KEYS = ["hp", "attack", "defense", "sp_attack", "sp_defense", "speed"]


def clean_text(text: str) -> str:
    text = re.sub(r"\[\[（属性）\|]]", "", text)
    text = re.sub(r"\s+", " ", text)
    return text.strip()


def iter_section_nodes(start: Tag) -> list[Tag]:
    nodes: list[Tag] = []
    for sib in start.next_siblings:
        if isinstance(sib, Tag) and sib.name == "h2":
            break
        if isinstance(sib, Tag):
            nodes.append(sib)
    return nodes


def iter_tables_in_section(start: Tag) -> list[Tag]:
    tables: list[Tag] = []
    seen: set[int] = set()
    for node in iter_section_nodes(start):
        candidates = [node] if node.name == "table" else node.find_all("table", recursive=True)
        for table in candidates:
            tid = id(table)
            if tid not in seen:
                seen.add(tid)
                tables.append(table)
    return tables


def section_content(start: Tag) -> str:
    parts: list[str] = []
    for sib in start.next_siblings:
        if isinstance(sib, Tag) and sib.name == "h2":
            break
        if isinstance(sib, Tag):
            txt = sib.get_text("\n", strip=True)
            if txt:
                parts.append(txt)
        elif isinstance(sib, NavigableString):
            txt = str(sib).strip()
            if txt:
                parts.append(txt)
    return "\n".join(parts)


def section_tables(start: Tag) -> list[Tag]:
    tables: list[Tag] = []
    for sib in start.next_siblings:
        if isinstance(sib, Tag) and sib.name == "h2":
            break
        if isinstance(sib, Tag) and sib.name == "table":
            tables.append(sib)
    return tables


def find_h2(soup: BeautifulSoup, title: str) -> Tag | None:
    for h2 in soup.select("h2 .mw-headline"):
        if h2.get_text(strip=True) == title:
            return h2.find_parent("h2")
    return None


def find_h3(soup: BeautifulSoup, title: str) -> Tag | None:
    for h3 in soup.select("h3 .mw-headline"):
        if h3.get_text(strip=True) == title:
            return h3.find_parent("h3")
    return None


def parse_infobox(soup: BeautifulSoup) -> dict:
    info: dict = {}
    main = soup.select_one("#mw-content-text")
    if not main:
        return info

    for table in main.select("table.roundy, table"):
        text = table.get_text("\n", strip=True)
        if "身高" not in text or "体重" not in text or "分类" not in text:
            continue
        if len(text) > 12000:
            continue

        lines = [ln.strip() for ln in text.split("\n") if ln.strip()]

        for i, line in enumerate(lines[:15]):
            if re.match(r"^[\u4e00-\u9fff]+$", line) and line not in {"官方绘图", "属性", "分类"}:
                jp = ""
                en = ""
                if i + 1 < len(lines) and re.search(r"[\u3040-\u30ff]", lines[i + 1]):
                    jp = lines[i + 1]
                if i + 2 < len(lines) and re.match(r"^[A-Z][a-z]+", lines[i + 2]):
                    en = lines[i + 2]
                info["names"] = {"zh": line, "jp": jp, "en": en}
                break

        for i, line in enumerate(lines):
            if line == "属性" and i + 1 < len(lines):
                types: list[str] = []
                j = i + 1
                while j < len(lines) and lines[j] not in {"分类", "特性", "身高", "体重"}:
                    if lines[j] not in types:
                        types.append(lines[j])
                    j += 1
                info["types"] = types
            elif line == "分类" and i + 1 < len(lines):
                info["category"] = lines[i + 1]
            elif line == "特性":
                chunk: list[str] = []
                j = i + 1
                while j < len(lines) and lines[j] not in {"100级时经验值", "地区图鉴编号"}:
                    chunk.append(lines[j])
                    j += 1
                if "隐藏特性" in chunk:
                    idx = chunk.index("隐藏特性")
                    abilities = [x for x in chunk[:idx] if x != "隐藏特性"]
                    hidden = abilities.pop() if abilities else ""
                    info["abilities"] = abilities
                    if hidden:
                        info["hidden_ability"] = hidden
                else:
                    info["abilities"] = [x for x in chunk if x != "特性"]
            elif line == "身高" and i + 1 < len(lines):
                info["height"] = lines[i + 1]
            elif line == "体重" and i + 1 < len(lines):
                info["weight"] = lines[i + 1]
            elif line == "图鉴颜色" and i + 1 < len(lines):
                info["color"] = lines[i + 1]
            elif line == "捕获率" and i + 1 < len(lines):
                info["catch_rate"] = lines[i + 1]
            elif line == "性别比例":
                parts: list[str] = []
                j = i + 1
                while j < len(lines) and lines[j] not in {"培育", "取得基础点数"}:
                    parts.append(lines[j])
                    j += 1
                info["gender_ratio"] = " ".join(parts)
            elif line == "100级时经验值" and i + 1 < len(lines):
                info["experience"] = lines[i + 1]
            elif line == "培育":
                parts = []
                j = i + 1
                while j < len(lines) and lines[j] not in {"取得基础点数", "基础经验值"}:
                    parts.append(lines[j])
                    j += 1
                info["breeding"] = " ".join(parts)

        m = re.search(r"#(\d{3,4})", text)
        if m:
            info["national_dex"] = m.group(1).zfill(4)
        m = re.search(r"基础经验值[：:]\s*(\d+)", text)
        if m:
            info["base_exp"] = m.group(1)

        ev_yield: dict[str, str] = {}
        for label in ["ＨＰ", "HP", "攻击", "防御", "特攻", "特防", "速度"]:
            em = re.search(rf"{label}\s*(\d+)", text)
            if em:
                key = "HP" if "Ｈ" in label or label == "HP" else label
                ev_yield[key] = em.group(1)
        if ev_yield:
            info["ev_yield"] = ev_yield

        regions = [
            "关都", "城都", "丰缘", "神奥", "合众", "卡洛斯", "阿罗拉",
            "伽勒尔", "洗翠", "帕底亚", "北上", "蓝莓", "密阿雷",
        ]
        regional: dict[str, str] = {}
        for i, line in enumerate(lines):
            if line in regions and i + 1 < len(lines):
                nxt = lines[i + 1]
                if nxt.startswith("#"):
                    num = nxt.lstrip("#")
                    if num not in {"000", "0", "—", "-"}:
                        regional[line] = num
        if regional:
            info["regional_dex"] = regional

        break

    return info


def extract_stats_from_table(table: Tag) -> dict[str, int]:
    stats: dict[str, int] = {}
    for row in table.select("tr"):
        th = row.select_one("th")
        if not th:
            continue
        label_link = th.select_one("a")
        value_span = th.select_one("span[style*='float:right']")
        if not label_link or not value_span:
            continue
        label = label_link.get_text(strip=True).replace("ＨＰ", "HP")
        try:
            value = int(value_span.get_text(strip=True))
        except ValueError:
            continue
        for stat_name, key in zip(STAT_NAMES, STAT_KEYS):
            if stat_name in label or label in stat_name:
                stats[key] = value
                break
    if stats:
        stats["total"] = sum(stats.get(k, 0) for k in STAT_KEYS)
    return stats


def parse_stats(soup: BeautifulSoup) -> dict[str, int]:
    h3 = find_h3(soup, "种族值")
    if not h3:
        return {}

    for table in iter_tables_in_section(h3):
        classes = " ".join(table.get("class", []))
        if "bg-" not in classes:
            continue
        stats = extract_stats_from_table(table)
        if stats:
            return stats
    return {}


def parse_evolution(soup: BeautifulSoup) -> str:
    h2 = find_h2(soup, "进化")
    if not h2:
        return ""
    return section_content(h2)


def parse_pokedex(soup: BeautifulSoup) -> list[dict[str, str]]:
    h3 = find_h3(soup, "图鉴介绍")
    if not h3:
        return []

    entries: list[dict[str, str]] = []
    seen: set[tuple[str, str]] = set()
    for table in iter_tables_in_section(h3):
        for row in table.select("tr"):
            cells = [c.get_text(" ", strip=True) for c in row.select("th, td")]
            if len(cells) < 2:
                continue
            game = cells[0]
            desc = cells[-1]
            if not desc or len(desc) < 10:
                continue
            if "{{{" in desc or "}}" in desc:
                continue
            if not game or "世代" in game or "活动" in game or "第九世代" in game:
                continue
            if game in {"一", "二", "三", "四", "五", "六", "七", "八", "九"}:
                continue
            key = (game, desc)
            if key in seen:
                continue
            seen.add(key)
            entries.append({"game": game, "description": desc})
    return entries


def parse_names(soup: BeautifulSoup) -> list[dict[str, str]]:
    h2 = find_h2(soup, "名字")
    if not h2:
        return []

    names: list[dict[str, str]] = []
    for table in iter_tables_in_section(h2):
        rows = table.select("tr")
        if not rows:
            continue
        headers = [c.get_text(strip=True) for c in rows[0].select("th, td")]
        if "语言" not in headers[0]:
            continue
        for row in rows[1:]:
            cells = [c.get_text(" ", strip=True) for c in row.select("th, td")]
            if len(cells) < 2:
                continue
            language = cells[0]
            if len(cells) >= 4:
                region = cells[1]
                name = cells[2].split()[0]
                label = f"{language}（{region}）" if region else language
            else:
                label = language
                name = cells[1].split()[0]
            if name and name not in {"任天堂", "大陆", "台湾", "香港"}:
                names.append({"language": label, "name": name})
    return names


def parse_overview(soup: BeautifulSoup) -> str:
    h2 = find_h2(soup, "概述")
    if not h2:
        return ""
    return section_content(h2)


def parse_list_page(html: str) -> list[dict]:
    soup = BeautifulSoup(html, "lxml")
    pokemon_list: list[dict] = []
    for table in soup.select("table"):
        for row in table.select("tr"):
            cells = row.select("td")
            if len(cells) < 4:
                continue
            kanto = cells[0].get_text(strip=True)
            if not re.match(r"#\d+", kanto):
                continue
            national = cells[1].get_text(strip=True)
            link = cells[2].select_one("a")
            href = link.get("href", "") if link else ""
            name = cells[3].get_text(strip=True)
            types = []
            for c in cells[4:]:
                t = clean_text(c.get_text(strip=True))
                if t and "属性" not in t:
                    types.append(t)
            if not name and types:
                name = types[0]
                types = types[1:]
            types = [t for t in types if t and not t.startswith("[[")]
            pokemon_list.append(
                {
                    "kanto": kanto,
                    "national": national,
                    "name": name,
                    "href": href,
                    "types": types,
                }
            )
    return pokemon_list


def parse_detail_page(html: str, basic: dict) -> dict:
    soup = BeautifulSoup(html, "lxml")
    info = parse_infobox(soup)
    stats = parse_stats(soup)
    overview = parse_overview(soup)
    evolution = parse_evolution(soup)
    pokedex = parse_pokedex(soup)
    localized_names = parse_names(soup)

    names = info.get("names", {})
    if not names.get("zh") or names.get("zh", "").startswith("#"):
        names["zh"] = basic["name"]

    return {
        **basic,
        "name": names.get("zh") or basic["name"],
        "names": names,
        "types": info.get("types") or basic.get("types", []),
        "category": info.get("category", ""),
        "abilities": info.get("abilities", []),
        "hidden_ability": info.get("hidden_ability", ""),
        "height": info.get("height", ""),
        "weight": info.get("weight", ""),
        "gender_ratio": info.get("gender_ratio", ""),
        "catch_rate": info.get("catch_rate", ""),
        "color": info.get("color", ""),
        "experience": info.get("experience", ""),
        "breeding": info.get("breeding", ""),
        "base_exp": info.get("base_exp", ""),
        "ev_yield": info.get("ev_yield", {}),
        "regional_dex": info.get("regional_dex", {}),
        "stats": stats,
        "overview": overview,
        "evolution": evolution,
        "pokedex": pokedex,
        "other_names": localized_names,
    }


def display_name(p: dict) -> str:
    names = p.get("names", {})
    zh = names.get("zh") or p.get("name", "")
    if zh.startswith("#"):
        zh = p.get("name", zh)
    return zh


def render_pokemon(p: dict) -> str:
    lines: list[str] = []
    zh = display_name(p)
    names = p.get("names", {})
    jp = names.get("jp", "")
    en = names.get("en", "")

    lines.append(f"### {p['kanto']} {zh}")
    lines.append("")
    lines.append(f"- **全国图鉴编号**: {p['national'].lstrip('#')}")
    if jp or en:
        lines.append(f"- **日文名**: {jp or '—'}")
        lines.append(f"- **英文名**: {en or '—'}")
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
        stat_labels = [
            ("HP", "hp"),
            ("攻击", "attack"),
            ("防御", "defense"),
            ("特攻", "sp_attack"),
            ("特防", "sp_defense"),
            ("速度", "speed"),
        ]
        for label, key in stat_labels:
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

    if p.get("other_names") and isinstance(p["other_names"], list):
        lines.append("")
        lines.append("#### 各地名称")
        lines.append("")
        lines.append("| 语言 | 名称 |")
        lines.append("| --- | --- |")
        for n in p["other_names"]:
            if isinstance(n, dict) and n.get("language") and n.get("name"):
                lines.append(f"| {n['language']} | {n['name']} |")

    wiki_path = p.get("href", "")
    if wiki_path:
        lines.append("")
        lines.append(f"- **百科链接**: [{zh}]({BASE_URL}{wiki_path})")

    lines.append("")
    return "\n".join(lines)


def render_markdown(pokemon: list[dict]) -> str:
    lines = [
        "# 关都地区宝可梦完整资料",
        "",
        "> 资料来源：[神奇宝贝百科 - 宝可梦列表（按关都图鉴编号）](https://wiki.52poke.com/wiki/%E5%AE%9D%E5%8F%AF%E6%A2%A6%E5%88%97%E8%A1%A8%EF%BC%88%E6%8C%89%E5%85%B3%E9%83%BD%E5%9B%BE%E9%89%B4%E7%BC%96%E5%8F%B7%EF%BC%89)",
        "",
        "关都图鉴是宝可梦系列最早出现的图鉴。在《红／绿／蓝／皮卡丘》及《火红／叶绿》中包含 **151** 只宝可梦；"
        "在《Let's Go! 皮卡丘／Let's Go! 伊布》中扩展至 **153** 只（新增美录坦、美录梅塔）。",
        "",
        "## 目录",
        "",
    ]

    for p in pokemon:
        kanto = p["kanto"]
        name = display_name(p)
        anchor = f"{kanto.lstrip('#')}-{name}"
        lines.append(f"- [{kanto} {name}](#{anchor})")

    lines.append("")
    lines.append("---")
    lines.append("")

    # 汇总表
    lines.append("## 关都图鉴一览表")
    lines.append("")
    lines.append("| 关都编号 | 全国编号 | 宝可梦 | 属性 | 分类 | 种族值合计 |")
    lines.append("| ---: | ---: | --- | --- | --- | ---: |")
    for p in pokemon:
        types = " / ".join(p.get("types", []))
        total = p.get("stats", {}).get("total", "—")
        name = display_name(p)
        lines.append(
            f"| {p['kanto']} | {p['national']} | {name} | {types} | {p.get('category', '—')} | {total} |"
        )

    lines.append("")
    lines.append("---")
    lines.append("")
    lines.append("## 详细信息")
    lines.append("")

    for p in pokemon:
        lines.append(render_pokemon(p))

    lines.append("---")
    lines.append("")
    lines.append(
        f"*本文档由脚本自动生成，共收录 {len(pokemon)} 只关都图鉴宝可梦。*"
    )
    lines.append("")
    return "\n".join(lines)


def scrape(start: int = 0, limit: int | None = None, force: bool = False) -> list[dict]:
    results: list[dict] = []
    if CACHE_PATH.exists() and not force:
        results = json.loads(CACHE_PATH.read_text(encoding="utf-8"))
        print(f"Loaded {len(results)} cached entries")
    elif CACHE_PATH.exists() and force:
        results = []
        print("Force refresh: ignoring cache")
    else:
        results = []

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=False,
            channel="chrome",
            args=["--disable-blink-features=AutomationControlled"],
        )
        context = browser.new_context(locale="zh-CN", viewport={"width": 1280, "height": 900})
        page = context.new_page()
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

        if not results:
            print("Fetching list page...")
            html = fetch(LIST_URL)
            basics = parse_list_page(html)
            print(f"Found {len(basics)} pokemon in list")
        else:
            basics = [
                {
                    "kanto": r["kanto"],
                    "national": r["national"],
                    "name": r.get("names", {}).get("zh") or r["name"],
                    "href": r.get("href", ""),
                    "types": r.get("types", []),
                }
                for r in results
            ]
            # refill missing from cache length mismatch
            if len(basics) < 153:
                html = fetch(LIST_URL)
                basics = parse_list_page(html)

        cached_nums = {r["kanto"] for r in results}
        end = len(basics) if limit is None else min(start + limit, len(basics))
        targets = basics[start:end]

        for i, basic in enumerate(targets):
            if basic["kanto"] in cached_nums and not force:
                print(f"Skip cached {basic['kanto']} {basic['name']}")
                continue
            url = BASE_URL + basic["href"]
            print(f"[{start + i + 1}/{len(basics)}] Fetching {basic['kanto']} {basic['name']}...")
            try:
                html = fetch(url)
                detail = parse_detail_page(html, basic)
                # update or append
                existing = next((j for j, r in enumerate(results) if r["kanto"] == basic["kanto"]), None)
                if existing is not None:
                    results[existing] = detail
                else:
                    results.append(detail)
                CACHE_PATH.parent.mkdir(parents=True, exist_ok=True)
                CACHE_PATH.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
            except Exception as exc:
                print(f"  ERROR: {exc}")

        browser.close()

    results.sort(key=lambda x: int(x["kanto"].lstrip("#")))
    return results


def main() -> None:
    start = int(sys.argv[1]) if len(sys.argv) > 1 and sys.argv[1].isdigit() else 0
    limit = int(sys.argv[2]) if len(sys.argv) > 2 and sys.argv[2].isdigit() else None
    force = "--force" in sys.argv
    render_only = "--render-only" in sys.argv

    if render_only:
        pokemon = json.loads(CACHE_PATH.read_text(encoding="utf-8"))
        pokemon.sort(key=lambda x: int(x["kanto"].lstrip("#")))
    else:
        pokemon = scrape(start=start, limit=limit, force=force)
    if len(pokemon) < 153:
        print(f"Warning: only {len(pokemon)}/153 pokemon scraped. Re-run to continue.")

    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    md = render_markdown(pokemon)
    OUTPUT_PATH.write_text(md, encoding="utf-8")
    print(f"Written {OUTPUT_PATH} ({len(md):,} chars, {len(pokemon)} pokemon)")


if __name__ == "__main__":
    main()
