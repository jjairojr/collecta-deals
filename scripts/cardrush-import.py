import json
import sys

raw = open(sys.argv[1]).read().strip().split("\n", 1)
items = []
for row in raw[1].split(";"):
    s, num, rarity, yen, stock, pid = row.split(",")
    items.append({
        "set": s,
        "num": int(num),
        "rarity": rarity,
        "yen": int(yen),
        "stock": int(stock),
        "url": f"https://www.cardrush-pokemon.jp/product/{pid}",
    })
json.dump({"capturedAt": raw[0], "items": items}, open(sys.argv[2], "w"), ensure_ascii=False)
print(len(items), "items")
