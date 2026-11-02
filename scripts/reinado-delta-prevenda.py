import json
import os
import sys
import urllib.request
import uuid

BASE = sys.argv[1].rstrip("/")
TOKEN = os.environ.get("ADMIN_TOKEN", "")
IMG = "/Users/jjairo/Documents/www/collecta-app/reinado-delta/WhatsApp Image 2026-11-02 at 13.08.51{}.jpeg"
SHIPS = "2026-11-06"

PRODUCTS = [
    ("(PT-BR) Blister Triplo Megaevolução 6 - Reinado Delta - Spritzee", 39.80, 41.90, " (1)"),
    ("(PT-BR) Blister Quádruplo Megaevolução 6 - Reinado Delta - Seel", 51.90, 54.90, ""),
    ("(PT-BR) Combo de Pacotes de Booster - Megaevolução 6 - Reinado Delta", 237.90, 249.90, " (4)"),
    ("(PT-BR) Caixa de Booster - Megaevolução 6 - Reinado Delta", 389.50, 410.00, " (3)"),
    ("(PT-BR) Coleção Treinador Avançado (ETB) - Megaevolução 6 - Reinado Delta", 379.90, 399.90, " (2)"),
]


def request(method, path, body=None, headers=None):
    req = urllib.request.Request(BASE + path, data=body, method=method)
    if TOKEN:
        req.add_header("X-Admin-Token", TOKEN)
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    with urllib.request.urlopen(req) as res:
        return json.loads(res.read())


def post_json(path, payload):
    return request("POST", path, json.dumps(payload).encode(), {"Content-Type": "application/json"})


def upload(trade_id, path):
    boundary = uuid.uuid4().hex
    with open(path, "rb") as f:
        data = f.read()
    body = (
        f"--{boundary}\r\nContent-Disposition: form-data; name=\"photo\"; filename=\"{trade_id}.jpg\"\r\n"
        f"Content-Type: image/jpeg\r\n\r\n"
    ).encode() + data + f"\r\n--{boundary}--\r\n".encode()
    return request("POST", f"/api/trades/{trade_id}/photo?game=pokemon", body, {"Content-Type": f"multipart/form-data; boundary={boundary}"})


existing = {t["name"] for t in request("GET", "/api/trades?game=pokemon")["trades"] if t.get("status") == "holding"}
for name, pix, card, suffix in PRODUCTS:
    if name in existing:
        print("skip, already there:", name)
        continue
    created = post_json("/api/trades?game=pokemon", {
        "kind": "sealed",
        "name": name,
        "set": "SEALED",
        "condition": "Lacrado",
        "qty": 99,
        "buyBRL": 0,
        "buyDate": "2026-11-02",
        "manualBRL": pix,
        "askBRL": pix,
        "cardBRL": card,
        "shipsFrom": SHIPS,
        "listed": True,
        "status": "holding",
        "notes": "Pré-venda Reinado Delta, entrega a partir de 06/11; qty e custo provisórios",
    })
    photo = upload(created["id"], IMG.format(suffix))
    print(created["id"], name, photo.get("imageURL"), photo.get("askBRL"), photo.get("cardBRL"), photo.get("shipsFrom"))
