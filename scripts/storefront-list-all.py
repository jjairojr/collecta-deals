import json
import os
import urllib.request

B = "https://collecta-deals-production.up.railway.app"


def admin_token():
    if os.environ.get("ADMIN_TOKEN"):
        return os.environ["ADMIN_TOKEN"]
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "deploy.env")
    for line in open(path, encoding="utf-8"):
        if line.startswith("ADMIN_TOKEN="):
            return line.split("=", 1)[1].strip()
    raise SystemExit("ADMIN_TOKEN ausente em scripts/deploy.env")


AUTH = {"X-Admin-Token": admin_token()}


def get(path):
    return json.load(urllib.request.urlopen(urllib.request.Request(f"{B}{path}", headers=AUTH), timeout=60))


def post(path, body):
    req = urllib.request.Request(
        f"{B}{path}",
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json", **AUTH},
        method="POST",
    )
    return json.load(urllib.request.urlopen(req, timeout=120))


def listing(t):
    return {
        "id": t["id"],
        "askBRL": t["askBRL"],
        "listed": True,
        "imageURL": t.get("imageURL", ""),
        "featured": t.get("featured", False),
    }


def wants_listing(t):
    return t.get("status") == "holding" and t.get("askBRL", 0) > 0 and not t.get("listed")


total = 0
for game in ["onepiece", "pokemon", "riftbound", "lorcana", "gundam"]:
    items = [listing(t) for t in get(f"/api/trades?game={game}")["trades"] if wants_listing(t)]
    if not items:
        print(game, "nada a listar")
        continue
    res = post(f"/api/trades/listings?game={game}", {"items": items})
    print(game, "enviados", len(items), "->", res)
    total += res.get("updated", 0)

rows = get("/api/purchases")
everything = rows["unassigned"] + [i for p in rows["purchases"] for i in p["items"]]
items = [listing(t) for t in everything if t["game"] == "acessorios" and wants_listing(t)]
if items:
    res = post("/api/trades/listings?game=onepiece", {"items": items})
    print("acessorios enviados", len(items), "->", res)
    total += res.get("updated", 0)
else:
    print("acessorios nada a listar")

print("TOTAL atualizados", total)
print("storefront count", get("/api/storefront")["count"])
