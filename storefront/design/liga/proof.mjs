import { readFile } from "node:fs/promises";
import { join } from "node:path";
import {
  out, loadAssets, baseCSS, page, shot,
  INK, PAGE, SURFACE, ROYAL, BRAND, BRAND_SOFT,
} from "./lib.mjs";
import { artCSS } from "./art.mjs";

const W = 1260;
const H = 3010;

async function uri(name) {
  const buf = await readFile(join(out, name));
  return `data:image/png;base64,${buf.toString("base64")}`;
}

const NAV = [
  "One Piece", "Magic", "Pokemon", "Yugioh", "Selados",
  "Acessórios", "Sets / Playsets", "Colecionáveis",
];

const proofCSS = `
body{background:#e8e8ec;font-family:"DM Sans",sans-serif;overflow:visible}
.sheet{width:${W}px;padding:26px 30px 40px;display:flex;flex-direction:column;gap:26px}
.h{font-family:"Baloo 2",sans-serif;font-weight:800;font-size:19px;color:#1a1a1e}
.h span{font-family:"DM Sans";font-weight:500;font-size:12px;color:#6f6f77;margin-left:9px}
.card{background:#fff;border:1px solid #cfcfd6;border-radius:10px;overflow:hidden}
.pad{padding:18px}

/* Cabeçalho simulado: bg em repeat + logo + barra de menu, como a Liga monta. */
.simtop{height:110px;position:relative}
.simtop .lg{position:absolute;left:78px;top:19px;width:170px;height:72px}
.simtop .lg img{width:100%;height:100%;object-fit:contain;display:block}
.simtop .acct{
  position:absolute;right:70px;top:38px;display:flex;align-items:center;gap:26px;
  color:#fff;font-size:13px;font-weight:700;
}
.simtop .search{
  position:absolute;left:50%;transform:translateX(-50%);top:36px;
  width:420px;height:36px;background:#fff;border-radius:3px;
  display:flex;align-items:center;padding:0 12px;color:#8d8d95;font-size:13px;
}
.simmenu{height:34px;background:${PAGE};display:flex;align-items:center;gap:26px;padding:0 78px}
.simmenu a{color:#fff;font-size:12.5px;font-weight:700;text-decoration:none}
.simmenu a.on{color:${BRAND}}
.simsub{
  margin-left:78px;width:210px;background:${SURFACE};border-top:0;
}
.simsub div{padding:9px 14px;color:#fff;font-size:12.5px;font-weight:600}
.simsub div.on{background:${BRAND}}
.simbody{background:#fff;padding:22px 60px 26px;display:flex;flex-direction:column;gap:18px}
.simfoot{height:74px;background:${ROYAL};display:flex;align-items:center;padding:0 78px;gap:24px}
.simfoot .fw{font-family:"Baloo 2";font-weight:800;font-size:26px;color:#fff;text-shadow:2px 2px 0 ${BRAND},4px 4px 0 ${INK}}
.simfoot a{color:#fff;font-size:13px;font-weight:700;text-decoration:none}
.simfoot2{height:30px;background:${INK};display:flex;align-items:center;justify-content:center}
.simfoot2 span{font-family:"Press Start 2P";font-size:8px;color:#5c5c66}

.row{display:flex;gap:18px;align-items:flex-start}
.shot{display:block}
.tag{font-family:"Press Start 2P";font-size:8px;color:#6f6f77;margin-top:7px}
.swatch{width:74px;height:52px;border-radius:7px;border:2px solid ${INK}}
.sw{display:flex;flex-direction:column;gap:5px;align-items:center}
.sw code{font-size:10px;color:#4a4a52;font-family:ui-monospace,monospace}
.onroyal{background:${ROYAL};padding:14px;border-radius:8px;display:flex;gap:16px;align-items:center}
.onwhite{background:#fff;padding:14px;border-radius:8px;border:1px solid #dcdce2;display:flex;gap:16px;align-items:center}
.pixels{image-rendering:pixelated}
`;

async function main() {
  const { fonts } = await loadAssets();
  const [
    top, logoU, superior, superiorM, heroU, nheroU, trustU, tSing, tSel, tAce,
    mini, mkt, avatar, f16, f32, f48, f192,
  ] = await Promise.all([
    uri("header-bg-1920x110.png"),
    uri("logo-200x85.png"),
    uri("banner-superior-1170x60.png"),
    uri("banner-superior-mobile-400x80.png"),
    uri("banner-hero-1170x275.png"),
    uri("noticia-hero-1170x360.png"),
    uri("banner-full-confianca-1170x275.png"),
    uri("banner-triplo-singles-400x275.png"),
    uri("banner-triplo-selados-400x275.png"),
    uri("banner-triplo-acessorios-400x275.png"),
    uri("miniatura-300x300.png"),
    uri("marketplace-logo-101x30.png"),
    uri("marketplace-avatar-55x55.png"),
    uri("favicon-16.png"),
    uri("favicon-32.png"),
    uri("favicon-48.png"),
    uri("favicon-192.png"),
  ]);

  const swatches = [
    ["#f6559b", "brand"],
    ["#fdc4e5", "brand-soft"],
    ["#1355b3", "royal"],
    ["#141416", "ink/page"],
    ["#0b0b0c", "outline"],
    ["#1f1f22", "surface"],
  ]
    .map(
      ([hex, label]) =>
        `<div class="sw"><div class="swatch" style="background:${hex}"></div><code>${hex}</code><code>${label}</code></div>`,
    )
    .join("");

  const body = `<div class="sheet">
  <div class="h">Loja virtual — como vai ficar <span>cabeçalho 1920×110 em repeat · logo 200×85 · menu #141416 · hover rosa</span></div>
  <div class="card">
    <div class="simtop" style="background-image:url(${top});background-repeat:repeat">
      <div class="lg"><img src="${logoU}"></div>
      <div class="search">Faça sua busca</div>
      <div class="acct"><span>Olá, visitante</span><span>Carrinho 0</span></div>
    </div>
    <div class="simmenu">${NAV.map((n, i) => `<a href="#" class="${i === 4 ? "on" : ""}">${n}</a>`).join("")}</div>
    <div class="simsub"><div>Boosters</div><div class="on">Booster Box</div><div>Elite Trainer Box</div></div>
    <div class="simbody">
      <img class="shot" src="${nheroU}" width="1140" height="350">
      <div class="row" style="gap:10px">
        <img class="shot" src="${tSing}" width="373" height="257">
        <img class="shot" src="${tSel}" width="373" height="257">
        <img class="shot" src="${tAce}" width="373" height="257">
      </div>
      <img class="shot" src="${trustU}" width="1140" height="268">
    </div>
    <div class="simfoot">
      <span class="fw">COLLECTA</span>
      <a href="#">Fale Conosco</a><a href="#">Perguntas Frequentes</a><a href="#">Trocas e Devoluções</a>
    </div>
    <div class="simfoot2"><span>© 2026 COLLECTA</span></div>
  </div>

  <div class="h">Peças no tamanho exato <span>notícia 1170×360 · superior 1170×60 (+mobile 400×80) · 100% 1170×275 · triplo 400×275 ×3</span></div>
  <div class="card pad" style="display:flex;flex-direction:column;gap:14px">
    <div><img class="shot" src="${nheroU}"><div class="tag">NOTICIA / HERO 1170x360 &mdash; topo da home</div></div>
    <div><img class="shot" src="${superior}"><div class="tag">BANNER SUPERIOR 1170x60 &mdash; paginas internas</div></div>
    <div><img class="shot" src="${superiorM}"><div class="tag">BANNER SUPERIOR MOBILE 400x80</div></div>
    <div><img class="shot" src="${trustU}"><div class="tag">BANNER 100% 1170x275 &mdash; faixa de confianca</div></div>
    <div><img class="shot" src="${heroU}"><div class="tag">BANNER 100% 1170x275 &mdash; hero alternativo (reserva, nao publicado)</div></div>
    <div class="row">
      <div><img class="shot" src="${tSing}"><div class="tag">TRIPLO 1</div></div>
      <div><img class="shot" src="${tSel}"><div class="tag">TRIPLO 2</div></div>
      <div><img class="shot" src="${tAce}"><div class="tag">TRIPLO 3</div></div>
    </div>
  </div>

  <div class="h">Marcas <span>logo 200×85 · marketplace 101×30 · avatar 55×55 · miniatura 300×300 · favicons</span></div>
  <div class="card pad row" style="align-items:flex-start;gap:26px">
    <div>
      <div class="onroyal"><img src="${logoU}" width="200" height="85"></div>
      <div class="tag">LOGO 200x85 SOBRE O AZUL</div>
    </div>
    <div>
      <div class="onwhite"><img src="${mkt}" width="101" height="30"><img src="${avatar}" width="55" height="55"></div>
      <div class="tag">MARKETPLACE 101x30 + AVATAR 55x55</div>
    </div>
    <div>
      <img class="shot" src="${mini}" width="150" height="150">
      <div class="tag">MINIATURA 300x300 (a 50%)</div>
    </div>
    <div>
      <div class="onwhite" style="align-items:flex-end">
        <img class="pixels" src="${f16}" width="16" height="16">
        <img class="pixels" src="${f32}" width="32" height="32">
        <img class="pixels" src="${f48}" width="48" height="48">
        <img src="${f192}" width="64" height="64">
      </div>
      <div class="tag">FAVICON 16 / 32 / 48 / 192</div>
    </div>
  </div>

  <div class="h">Paleta aplicada</div>
  <div class="card pad row" style="gap:20px">${swatches}</div>
</div>`;

  await shot({
    name: "PROVA-collecta-liga",
    w: W,
    h: H,
    html: page({
      w: W,
      h: H,
      css: [fonts, baseCSS, artCSS, proofCSS].join("\n"),
      body,
    }),
  });
  console.log("dist/PROVA-collecta-liga.png");
}

main();
