import hashlib, json, os, sys, time
from playwright.sync_api import sync_playwright

ENTRIES = json.load(open("/workspace/.scrape/entries.json", encoding="utf-8"))
OUT = "/workspace/.scrape/cries"
os.makedirs(OUT, exist_ok=True)
UA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
LOG = open("/workspace/.scrape/scrape.log", "a", encoding="utf-8")

def log(*a):
    msg = " ".join(str(x) for x in a)
    print(msg, flush=True)
    LOG.write(msg + "\n"); LOG.flush()

def orig_url(fn):
    h = hashlib.md5(fn.encode()).hexdigest()
    return f"https://media.52poke.com/wiki/{h[0]}/{h[0:2]}/{fn}"

def wait_cf(page):
    for _ in range(25):
        t = page.title()
        if "Just a moment" not in t and t.strip():
            return True
        page.wait_for_timeout(2000)
    return False

manifest = {}
mpath = "/workspace/.scrape/manifest.json"
if os.path.exists(mpath):
    manifest = json.load(open(mpath, encoding="utf-8"))

with sync_playwright() as p:
    browser = p.chromium.launch(channel="chrome", headless=False,
        args=["--no-sandbox", "--disable-blink-features=AutomationControlled"])
    ctx = browser.new_context(locale="zh-CN", user_agent=UA)
    # speed: block images/fonts (keep scripts + media + xhr)
    def route(r):
        if r.request.resource_type in ("image", "font"):
            return r.abort()
        return r.continue_()
    ctx.route("**/*", route)

    page = ctx.new_page()
    dl = ctx.new_page()  # dedicated page for media download

    # prime Cloudflare
    page.goto(ENTRIES[0]["url"], wait_until="domcontentloaded", timeout=60000)
    if not wait_cf(page):
        log("FATAL: cloudflare not cleared"); browser.close(); sys.exit(1)
    log("Cloudflare cleared. Starting.", time.strftime("%H:%M:%S"))

    ok = fail = skip = 0
    for e in ENTRIES:
        kanto = e["kanto"]; name = e["name"]
        dest = os.path.join(OUT, f"{kanto}.opus")
        if os.path.exists(dest) and os.path.getsize(dest) > 200:
            skip += 1; continue
        try:
            page.goto(e["url"], wait_until="domcontentloaded", timeout=60000)
            if "Just a moment" in page.title():
                wait_cf(page)
            # find the cry audio element
            el = page.query_selector('span[data-audio="cry"] audio[data-mwtitle]') \
                 or page.query_selector('audio[data-mwtitle]')
            if not el:
                log(f"[{kanto}] {name}: NO AUDIO ELEMENT"); fail += 1; continue
            fn = el.get_attribute("data-mwtitle")

            # honor "click play": capture the media URL the player loads
            captured = {}
            def on_resp(r, cap=captured):
                u = r.url
                if "media.52poke.com" in u and ("cry" in u or u.endswith((".webm",".opus",".ogg"))):
                    cap.setdefault("url", u)
            page.on("response", on_resp)
            trig = page.query_selector('[data-audio-trigger="cry"]') or page.query_selector('a.mw-tmh-play')
            if trig:
                try: trig.click(timeout=3000)
                except Exception: pass
            page.wait_for_timeout(2500)
            page.remove_listener("response", on_resp)

            # derive original filename: prefer data-mwtitle (authoritative file name)
            url = orig_url(fn)

            # download full bytes via navigation (no CORS); capture body from response
            body_holder = {}
            def on_dl(r, h=body_holder, target=url):
                if r.url == target:
                    try: h["body"] = r.body()
                    except Exception: pass
            dl.on("response", on_dl)
            resp = dl.goto(url, wait_until="load", timeout=30000)
            dl.wait_for_timeout(500)
            dl.remove_listener("response", on_dl)
            body = body_holder.get("body")
            if body is None and resp is not None:
                body = resp.body()
            if not body or len(body) < 200:
                log(f"[{kanto}] {name}: EMPTY BODY for {fn}"); fail += 1; continue
            open(dest, "wb").write(body)
            manifest[kanto] = {"name": name, "cry_file": fn, "bytes": len(body),
                               "played_url": captured.get("url"), "src": url}
            ok += 1
            if ok % 10 == 0:
                json.dump(manifest, open(mpath,"w",encoding="utf-8"), ensure_ascii=False, indent=2)
            log(f"[{kanto}] {name}: OK {fn} {len(body)}B played={captured.get('url') is not None}")
            time.sleep(0.4)
        except Exception as ex:
            log(f"[{kanto}] {name}: EXC {repr(ex)[:160]}"); fail += 1

    json.dump(manifest, open(mpath,"w",encoding="utf-8"), ensure_ascii=False, indent=2)
    log(f"DONE ok={ok} fail={fail} skip={skip}", time.strftime("%H:%M:%S"))
    browser.close()
