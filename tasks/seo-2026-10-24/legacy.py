import json, re, sys, unicodedata
S = sys.argv[1]
LABELS = {"pokemon": "Pokémon", "onepiece": "One Piece", "riftbound": "Riftbound", "lorcana": "Disney Lorcana", "gundam": "Gundam Card Game"}
def slugify(s):
    s = unicodedata.normalize("NFD", s)
    s = "".join(c for c in s if unicodedata.category(c) != "Mn").lower()
    return re.sub(r"[^a-z0-9]+", "-", s).strip("-")
def slug(game, t):
    if t.get("slug"):
        return slugify(game + "-" + t["slug"])
    return slugify("-".join(p for p in [game, t.get("name") or "", t.get("number") or "", t.get("variant") or "", t.get("condition") or ""] if p))
def setcode(s):
    s = (s or "").strip()
    m = re.search(r"\(([A-Za-z0-9-]+)\)\s*$", s)
    return (m.group(1) if m else s).upper()
def parse(t):
    name = (t.get("name") or "").strip()
    num = (t.get("number") or "").strip()
    m = re.search(r"\(#?([A-Za-z]*\d+[a-zA-Z]?(?:/\w+)?)\)", name)
    if not num and m:
        num = m.group(1)
    base = re.sub(r"\s*\([^)]*\)", "", name).strip() if (t.get("kind") or "") == "" else name
    return base, num
def display(t):
    name = (t.get("name") or "").strip()
    if (t.get("kind") or "") != "":
        return name, False
    shown = re.sub(r"\s*\(#?[A-Za-z]*\d+[a-zA-Z]?(?:/\w+)?\)", "", name).strip()
    marked = "(" in shown or (t.get("variant") or "").upper() not in ("", "EN", "JP", "PT", "PT-BR", "ING", "JAP")
    return shown, marked
def key(t):
    base, num = parse(t)
    short = re.sub(r"/.*$", "", num).lstrip("0") or num
    return (slugify(base), short.upper(), setcode(t.get("set")), (t.get("condition") or "").upper(), slugify(t.get("variant") or ""), t.get("kind") or "")
old_files = {"pokemon": "trades-pkm.json", "onepiece": "trades.json", "riftbound": "trades-rft.json", "lorcana": "trades-lor.json"}
redirects, sold = {}, {}
for game, f in old_files.items():
    old = json.load(open(f"data/.sync-bak/prewipe-20261007-123024/{f}"))
    try:
        cur = json.load(open(f"{S}/cur-{game}.json"))["trades"]
    except FileNotFoundError:
        cur = []
    idx = {}
    for t in cur:
        if t.get("listed") and (t.get("askBRL") or 0) > 0:
            idx.setdefault(key(t), []).append(t)
    cur_slugs = {slug(game, t) for t in cur if t.get("listed") and (t.get("askBRL") or 0) > 0}
    for t in old:
        if not (t.get("listed") and (t.get("askBRL") or 0) > 0):
            continue
        o = slug(game, t)
        if o in cur_slugs or o in redirects:
            continue
        cands = idx.get(key(t), [])
        if cands:
            cands.sort(key=lambda c: (c["status"] != "holding", c.get("createdAt", "")))
            redirects[o] = slug(game, cands[0])
            sold.pop(o, None)
            continue
        if o in sold and t["status"] != "holding":
            continue
        base, num = parse(t)
        shown, marked = display(t)
        item = {"slug": o, "game": game, "gameLabel": LABELS[game], "name": shown, "number": num,
                "set": setcode(t.get("set")), "askBRL": t["askBRL"], "qty": 0}
        for k in ("kind", "variant", "condition", "imageURL"):
            if t.get(k):
                item[k] = t[k]
        if t.get("createdAt"):
            item["addedAt"] = t["createdAt"]
        if marked and not t.get("imageURL"):
            item["noArt"] = True
        sold[o] = item
json.dump(dict(sorted(redirects.items())), open("storefront/lib/legacy-slugs.json", "w"), indent=2, ensure_ascii=False)
json.dump(sorted(sold.values(), key=lambda x: x["slug"]), open("storefront/lib/legacy-sold.json", "w"), indent=1, ensure_ascii=False)
print("redirects", len(redirects), "legacy sold", len(sold))
for line in open(f"{S}/gsc404.txt").read().split():
    s = line.split("/", 1)[1]
    print("REDIR" if s in redirects else "SOLD " if s in sold else "MISS ", s, redirects.get(s, ""))
