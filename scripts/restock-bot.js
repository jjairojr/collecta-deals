#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");

const COPAG = "https://www.copagloja.com.br";
const KEYWORDS = /30 anos|reinado delta|me06|mega evolu|pok[eé]mon/i;
const HOT = /30 anos/i;
const COPAG_RANGE = [2500, 2800];
const MIN_QTY = 5;
const STATE_FILE = path.join(__dirname, "..", "data", "restock-state.json");

function parseArgs(argv) {
  const opts = { interval: 60, once: false, test: false };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === "-i" || a === "--interval") opts.interval = Number(argv[++i]);
    else if (a === "--once") opts.once = true;
    else if (a === "--test") opts.test = true;
    else {
      console.error(`argumento desconhecido: ${a}`);
      process.exit(1);
    }
  }
  return opts;
}

function loadState() {
  try {
    return JSON.parse(fs.readFileSync(STATE_FILE, "utf8"));
  } catch {
    return {};
  }
}

function saveState(state) {
  fs.mkdirSync(path.dirname(STATE_FILE), { recursive: true });
  fs.writeFileSync(STATE_FILE, JSON.stringify(state, null, 2));
}

function normalize(text) {
  return text.normalize("NFD").replace(/[̀-ͯ]/g, "").replace(/[–-]/g, " ").replace(/\s+/g, " ").trim().toLowerCase();
}

function cleanName(name) {
  const words = name.replace(/^CADASTRAR\s*-\s*/i, "").trim().split(/\s+/);
  for (let n = 1; n < words.length; n++) {
    const head = words.slice(0, n).join(" ");
    if (normalize(head) === normalize(words.slice(n).join(" "))) return head;
  }
  return words.join(" ");
}

async function notify(topic, { title, body, click, priority, tags }) {
  const params = new URLSearchParams({ title, priority: priority || "default" });
  if (click) params.set("click", click);
  if (tags) params.set("tags", tags);
  const res = await fetch(`https://ntfy.sh/${topic}?${params}`, { method: "POST", body });
  if (!res.ok) console.error(`ntfy falhou: HTTP ${res.status}`);
}

let copagOrderFormId = null;

async function checkCopag() {
  if (copagOrderFormId) {
    const cleared = await fetch(`${COPAG}/api/checkout/pub/orderForm/${copagOrderFormId}/items/removeAll`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    });
    if (!cleared.ok) copagOrderFormId = null;
  }
  if (!copagOrderFormId) {
    const orderForm = await fetch(`${COPAG}/api/checkout/pub/orderForm`).then((r) => r.json());
    copagOrderFormId = orderForm.orderFormId;
  }
  const ids = [];
  for (let id = COPAG_RANGE[0]; id <= COPAG_RANGE[1]; id++) ids.push(String(id));
  const res = await fetch(`${COPAG}/api/checkout/pub/orderForm/${copagOrderFormId}/items`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ orderItems: ids.map((id) => ({ id, quantity: 6, seller: "1" })) }),
  });
  if (!res.ok) {
    copagOrderFormId = null;
    throw new Error(`copag HTTP ${res.status}`);
  }
  const data = await res.json();
  const out = {};
  for (const item of data.items || []) {
    if (!KEYWORDS.test(item.name)) continue;
    out[`copag:${item.id}`] = {
      name: cleanName(item.name),
      status: item.availability === "available" ? "available" : "nostock",
      qty: item.availability === "available" ? item.quantity : 0,
      price: item.sellingPrice / 100,
      click: `${COPAG}/checkout/cart/add?sku=${item.id}&qty=${item.availability === "available" ? item.quantity : 1}&seller=1&sc=1`,
    };
  }
  for (const msg of data.messages || []) {
    const m = msg.text.match(/^Item (.+) não encontrado ou indisponível$/);
    if (!m || !KEYWORDS.test(m[1])) continue;
    const id = msg.fields?.id;
    if (!id) continue;
    out[`copag:${id}`] = { name: cleanName(m[1]), status: "inactive", qty: 0, price: null, click: null };
  }
  return out;
}

function stocked(item) {
  return item?.status === "available" && item.qty >= MIN_QTY;
}

async function tick(topic, state) {
  const current = {};
  try {
    Object.assign(current, await checkCopag());
  } catch (err) {
    console.error(`[${new Date().toLocaleTimeString()}] copag: ${err.message}`);
  }
  for (const [key, value] of Object.entries(state)) if (!current[key]) current[key] = value;

  for (const [key, now] of Object.entries(current)) {
    if (!HOT.test(now.name) || !stocked(now) || stocked(state[key])) continue;
    await notify(topic, {
      title: `Copag: ${now.name}`,
      body: `EM ESTOQUE — ${now.qty}${now.qty >= 6 ? "+" : ""} disponíveis — R$ ${now.price.toFixed(2).replace(".", ",")}. Toque para pôr no carrinho.`,
      click: now.click,
      priority: "urgent",
      tags: "rotating_light",
    });
  }

  const available = Object.entries(current).filter(([, v]) => stocked(v)).length;
  console.log(`[${new Date().toLocaleTimeString()}] ${Object.keys(current).length} produtos vigiados, ${available} com ${MIN_QTY}+ em estoque`);
  return current;
}

async function main() {
  const opts = parseArgs(process.argv);
  const topic = process.env.NTFY_TOPIC;
  if (!topic) {
    console.error("defina NTFY_TOPIC");
    process.exit(1);
  }
  if (opts.test) {
    await notify(topic, { title: "Restock bot", body: "Teste: notificações funcionando.", priority: "high", tags: "white_check_mark" });
    return;
  }
  let state = loadState();
  for (;;) {
    state = await tick(topic, state);
    saveState(state);
    if (opts.once) return;
    await new Promise((r) => setTimeout(r, opts.interval * 1000));
  }
}

main();
