#!/usr/bin/env node

const { execFile } = require("node:child_process");

const API = "https://cartinha.com.br/api/v2/front";
const HEADERS = { "app-token": "wapstore", Accept: "application/json" };
const DEFAULT_URL =
  "https://cartinha.com.br/unitario-one-piece-tcg-the-worlds-strongest-warriors-booster-display-op-17.html";

function parseArgs(argv) {
  const opts = { url: DEFAULT_URL, qty: 1, watch: false, interval: 5, open: true };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === "-q" || a === "--qty") opts.qty = Number(argv[++i]);
    else if (a === "-w" || a === "--watch") opts.watch = true;
    else if (a === "-i" || a === "--interval") opts.interval = Number(argv[++i]);
    else if (a === "--no-open") opts.open = false;
    else if (a.startsWith("http")) opts.url = a;
    else {
      console.error(`argumento desconhecido: ${a}`);
      process.exit(1);
    }
  }
  if (!Number.isInteger(opts.qty) || opts.qty < 1) {
    console.error("quantidade inválida");
    process.exit(1);
  }
  return opts;
}

async function fetchProduct(productUrl) {
  const path = new URL(productUrl).pathname;
  const res = await fetch(`${API}/url/product/detail?url=${encodeURIComponent(path)}`, {
    headers: HEADERS,
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const data = await res.json();
  const p = data.conteudo;
  if (!p || !p.id) throw new Error("produto não encontrado na resposta");
  return p;
}

function buildAddUrl(product, qty) {
  const [id, attr, unit, wh] = product.carrinho.hash.split("-");
  return `https://cartinha.com.br/checkout/carrinho?add=${id}-${attr}-${unit}-${wh}-${qty}`;
}

function sleep(s) {
  return new Promise((r) => setTimeout(r, s * 1000));
}

function openInBrowser(url) {
  execFile("open", [url], (err) => {
    if (err) console.error("falha ao abrir o navegador:", err.message);
  });
}

async function main() {
  const opts = parseArgs(process.argv);
  let attempt = 0;
  for (;;) {
    attempt++;
    let product;
    try {
      product = await fetchProduct(opts.url);
    } catch (err) {
      console.log(`[${new Date().toLocaleTimeString()}] tentativa ${attempt}: erro (${err.message})`);
      if (!opts.watch) process.exit(1);
      await sleep(opts.interval);
      continue;
    }

    const available = product.status === "disponivel" && product.estoque > 0;
    console.log(
      `[${new Date().toLocaleTimeString()}] ${product.nome}\n` +
        `  status: ${product.status} | estoque: ${product.estoque} | preço: R$ ${product.precos.por}`
    );

    if (!available) {
      if (!opts.watch) {
        console.log("produto indisponível. use -w para monitorar até liberar.");
        process.exit(1);
      }
      await sleep(opts.interval);
      continue;
    }

    const qty = Math.min(opts.qty, product.estoque);
    if (qty < opts.qty) console.log(`  estoque menor que o pedido, ajustando para ${qty}`);
    const addUrl = buildAddUrl(product, qty);
    console.log(`\n✅ adicionando ${qty}x ao carrinho:\n  ${addUrl}\n`);
    if (opts.open) {
      openInBrowser(addUrl);
      console.log("aberto no navegador — o item entra no carrinho da SUA sessão. finalize o checkout por lá.");
    } else {
      console.log("abra essa URL no seu navegador para o item cair no seu carrinho.");
    }
    return;
  }
}

main();
