(async () => {
  const sets = [
    "M6a", "M6", "M5", "M4", "M3", "M2a", "M2", "M1S", "M1L",
    "SV11W", "SV11B", "SV10", "SV9a", "SV9", "SV8a", "SV8", "SV7a", "SV7", "SV6a", "SV6",
    "SV5a", "SV5M", "SV5K", "SV4a", "SV4M", "SV4K", "SV3a", "SV3", "SV2a", "SV2P", "SV2D", "SV1a", "SV1V", "SV1S",
  ];
  const pause = (ms) => new Promise((r) => setTimeout(r, ms));
  const get = async (url) => {
    for (let attempt = 0; attempt < 4; attempt++) {
      const ac = new AbortController();
      const timer = setTimeout(() => ac.abort(), 60000);
      try {
        const res = await fetch(url, { signal: ac.signal });
        const html = await res.text();
        if (res.ok) return html;
      } catch {
      } finally {
        clearTimeout(timer);
      }
      window.__cardrushLog.push(`retry ${url} ${attempt}`);
      await pause(5000);
    }
    window.__cardrushFail.push(url);
    return "";
  };
  const best = new Map();
  window.__cardrushLog = [];
  window.__cardrushFail = [];
  for (const set of sets) {
    for (let page = 1; page < 20; page++) {
      const html = await get(`/product-list?keyword=${encodeURIComponent(`AR ${set}`)}&num=100&page=${page}`);
      const doc = new DOMParser().parseFromString(html, "text/html");
      const items = [...doc.querySelectorAll("li.list_item_cell")];
      for (const li of items) {
        const raw = li.querySelector(".goods_name")?.textContent.trim() || "";
        if (raw.startsWith("〔")) continue;
        const model = li.querySelector(".model_number_value")?.textContent.trim() || "";
        if (model.toLowerCase() !== set.toLowerCase()) continue;
        const m = raw.match(/^(?:☆SALE☆)?(.*?)【(AR|SAR)】\{(\d+)\/[^}]*\}/);
        if (!m) continue;
        const text = li.textContent.replace(/\s+/g, " ");
        const yen = +((text.match(/([\d,]+)円/) || [])[1] || "0").replace(/,/g, "");
        const stock = +((text.match(/在庫数\s*(\d+)/) || [])[1] || 0);
        const id = (li.querySelector("a")?.getAttribute("href") || "").split("/").pop();
        const item = { set: model, num: +m[3], rarity: m[2], yen, stock, id };
        const key = `${model}|${item.num}|${item.rarity}`;
        const prev = best.get(key);
        const better = !prev || (stock > 0 && prev.stock === 0) || ((stock > 0) === (prev.stock > 0) && yen < prev.yen);
        if (yen > 0 && better) best.set(key, item);
      }
      window.__cardrushLog.push(`${set} p${page} ${items.length}`);
      await pause(2500);
      if (items.length < 100) break;
    }
  }
  const lines = [...best.values()].map((i) => [i.set, i.num, i.rarity, i.yen, i.stock, i.id].join(","));
  document.body.innerHTML = "";
  const pre = document.createElement("pre");
  pre.textContent = `${new Date().toISOString()}\n${lines.join(";")}`;
  document.body.appendChild(pre);
  window.__cardrushDone = lines.length;
})();
