import json
import sys
import urllib.request

B = "https://collecta-deals-production.up.railway.app"
MAPPING = sys.argv[1] if len(sys.argv) > 1 else "tasks/storefront-images-2026-10-09/product-images.json"


def get(path):
    return json.load(urllib.request.urlopen(f"{B}{path}", timeout=60))


def post(path, body):
    req = urllib.request.Request(
        f"{B}{path}",
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    return json.load(urllib.request.urlopen(req, timeout=120))


images = json.load(open(MAPPING, encoding="utf-8"))
rows = get("/api/purchases")
everything = rows["unassigned"] + [i for p in rows["purchases"] for i in p["items"]]
by_game = {}
matched = set()
for t in everything:
    if not t.get("kind") or t.get("status") != "holding" or t.get("imageURL"):
        continue
    url = images.get(t["name"].strip())
    if not url:
        continue
    matched.add(t["name"].strip())
    game = t["game"] if t["game"] != "acessorios" else "onepiece"
    by_game.setdefault(game, []).append({
        "id": t["id"],
        "askBRL": t.get("askBRL", 0),
        "listed": t.get("listed", False),
        "imageURL": url,
        "featured": t.get("featured", False),
    })

total = 0
for game, items in by_game.items():
    res = post(f"/api/trades/listings?game={game}", {"items": items})
    print(game, "enviados", len(items), "->", res)
    total += res.get("updated", 0)
print("TOTAL atualizados", total)
for name in images:
    if name not in matched:
        print("sem trade correspondente (ou já tinha imagem):", name)
