import {
  INK, PAGE, ROYAL, ROYAL_LIGHT, BRAND, BRAND_SOFT, ON_SOFT, TILE,
} from "./lib.mjs";

/* Every piece is laid out at its final pixel size — the Liga admin validates
   each upload against an exact width/height, so nothing here scales. */

export const artCSS = `
.logo,.topbg,.mini,.fav,.mkt,.avatar,.strip,.hero,.nhero,.trust,.tri{position:relative;overflow:hidden}

/* ── logo 200x85, transparent ─────────────────────────────────── */
.logo{
  width:200px;height:85px;display:flex;flex-direction:column;
  align-items:center;justify-content:center;gap:9px;
}
.logo .wm{
  font-size:40px;
  text-shadow:3px 3px 0 ${BRAND},6px 6px 0 ${INK};
}
.logo .tag{font-size:7px;color:${BRAND_SOFT};letter-spacing:.6px}

/* ── header background 1920x110, tiles horizontally ───────────── */
.topbg{width:1920px;height:110px;background:${ROYAL}}
.topbg .ticks{
  background-image:repeating-linear-gradient(90deg,${BRAND} 0 8px,transparent 8px ${TILE}px);
  background-repeat:repeat-x;background-size:${TILE}px 4px;
  height:4px;top:0;bottom:auto;opacity:.9;
}
.topbg .shade{
  background:linear-gradient(180deg,rgba(11,11,12,.34) 0%,rgba(11,11,12,0) 44%,rgba(11,11,12,.36) 100%);
}
.topbg .edge{top:auto;height:8px;background:${INK}}

/* ── square marks: miniatura 300x300 + favicons ───────────────── */
.mini{
  width:300px;height:300px;background:${ROYAL};
  display:flex;flex-direction:column;align-items:center;justify-content:center;gap:16px;
}
.mini .disc{width:152px;height:152px;border-width:7px}
.mini .wm{font-size:42px;text-shadow:3px 3px 0 ${BRAND},6px 6px 0 ${INK}}
.mini .frame{border:8px solid ${INK}}

.fav{background:${BRAND};display:flex;align-items:center;justify-content:center}
.fav img{
  width:126%;height:126%;min-width:126%;
  object-fit:cover;object-position:50% 40%;display:block;
}
/* At 16–48px the mascot's ink linework dissolves into a smudge, so the tab-bar
   sizes get the typographic mark instead and only 192/512 carry the face. */
.fav .letter{
  font-family:"Baloo 2",sans-serif;font-weight:800;color:#fff;line-height:1;
  display:block;
}

/* ── marketplace lockups ──────────────────────────────────────── */
.mkt{
  width:101px;height:30px;background:${ROYAL};
  display:flex;align-items:center;justify-content:center;gap:3px;
}
.mkt .mgrid{background-image:repeating-linear-gradient(90deg,rgba(255,255,255,.075) 0 1px,transparent 1px 12px)}
.mkt .ticks{
  top:0;bottom:auto;height:2px;opacity:.95;
  background-image:repeating-linear-gradient(90deg,${BRAND} 0 4px,transparent 4px 12px);
}
.mkt .mdisc{
  width:24px;height:24px;border-radius:999px;overflow:hidden;flex:none;
  background:#fff;border:2px solid ${INK};margin-top:-1px;position:relative;
}
.mkt .mdisc img{
  position:absolute;display:block;width:155%;left:-26%;top:-42%;
  filter:contrast(1.55) saturate(1.25);
}
.mkt .wm{font-size:13.5px;text-shadow:1px 1px 0 ${BRAND},2px 2px 0 ${INK}}

.avatar{width:55px;height:55px;background:${BRAND};border:3px solid ${INK};overflow:hidden}
.avatar img{width:100%;height:100%;object-fit:cover;object-position:50% 38%;display:block}

/* ── announcement strips ──────────────────────────────────────── */
.strip{background:${ROYAL};display:flex;align-items:center;justify-content:center}
.strip .rule{background:${INK};left:0;right:0;height:4px;top:auto}
.strip .rule-top{top:0;bottom:auto}
.strip .notes{display:flex;align-items:center;gap:22px;color:${BRAND_SOFT}}
.strip .dot{
  width:8px;height:8px;border-radius:999px;background:${BRAND};flex:none;
}
.strip .sep{color:${BRAND};font-size:11px}

/* ── hero 1170x275 ────────────────────────────────────────────── */
.hero{width:1170px;height:275px;background:${ROYAL};display:flex;align-items:center}
.hero .glow{
  background:radial-gradient(circle at 76% 46%,rgba(246,85,155,.55) 0%,rgba(246,85,155,0) 58%);
}
.hero .frame{border:5px solid ${INK}}
.hero .left{padding-left:58px;display:flex;flex-direction:column;align-items:flex-start;gap:14px;width:690px}
.hero .chip{
  background:${BRAND_SOFT};color:${INK};font-size:10px;padding:9px 13px;
  border:3px solid ${INK};border-radius:7px;box-shadow:3px 3px 0 ${INK};
}
.hero .wm{font-size:74px;text-shadow:4px 4px 0 ${BRAND},8px 8px 0 ${INK}}
.hero .sub{font-size:17px;line-height:1.35;color:${BRAND_SOFT};max-width:520px}
.hero .sub b{color:#fff;font-weight:700}
.hero .ctas{display:flex;gap:16px;margin-top:2px}
.hero .cta{font-size:11px;padding:14px 20px;border-radius:9px}
.hero .cta-a{background:${BRAND};color:#fff}
.hero .cta-b{background:${BRAND_SOFT};color:${INK}}
.hero .right{position:relative;margin-left:auto;margin-right:66px;width:226px;height:226px}
.hero .shadowdisc{
  position:absolute;left:18px;top:18px;width:208px;height:208px;
  border-radius:999px;background:${BRAND};
}
.hero .disc{position:absolute;left:0;top:0;width:208px;height:208px;background:${ROYAL_LIGHT}}
.hero .tagsticker{
  position:absolute;bottom:-2px;right:-14px;transform:rotate(-4deg);
  background:#fff;color:${INK};font-size:10px;padding:10px 12px;
  border:4px solid ${INK};border-radius:8px;box-shadow:4px 4px 0 ${INK};
}

/* ── news hero 1170x360 (the slider slot at the top of the home) ─ */
.nhero{width:1170px;height:360px;background:${ROYAL};display:flex;align-items:center}
.nhero .glow{background:radial-gradient(circle at 76% 48%,rgba(246,85,155,.55) 0%,rgba(246,85,155,0) 58%)}
.nhero .frame{border:5px solid ${INK}}
.nhero .left{padding-left:64px;display:flex;flex-direction:column;align-items:flex-start;gap:17px;width:720px}
.nhero .chip{
  background:${BRAND_SOFT};color:${INK};font-size:11px;padding:11px 15px;
  border:3px solid ${INK};border-radius:7px;box-shadow:3px 3px 0 ${INK};
}
.nhero .wm{font-size:94px;text-shadow:5px 5px 0 ${BRAND},10px 10px 0 ${INK}}
.nhero .sub{font-size:19px;line-height:1.4;color:${BRAND_SOFT};max-width:560px}
.nhero .sub b{color:#fff;font-weight:700}
.nhero .ctas{display:flex;gap:18px;margin-top:4px}
.nhero .cta{font-size:12px;padding:16px 22px;border-radius:9px}
.nhero .cta-a{background:${BRAND};color:#fff}
.nhero .cta-b{background:${BRAND_SOFT};color:${INK}}
.nhero .right{position:relative;margin-left:auto;margin-right:74px;width:290px;height:290px}
.nhero .shadowdisc{position:absolute;left:22px;top:22px;width:268px;height:268px;border-radius:999px;background:${BRAND}}
.nhero .disc{position:absolute;left:0;top:0;width:268px;height:268px;background:${ROYAL_LIGHT}}
.nhero .tagsticker{
  position:absolute;bottom:2px;right:-10px;transform:rotate(-4deg);
  background:#fff;color:${INK};font-size:11px;padding:12px 14px;
  border:4px solid ${INK};border-radius:8px;box-shadow:4px 4px 0 ${INK};
}

/* ── trust strip 1170x275 (the Banner 100% slot, below the cards) ─ */
.trust{width:1170px;height:275px;background:${PAGE};display:flex;align-items:stretch}
.trust .glow{background:radial-gradient(circle at 50% 120%,rgba(19,85,179,.55) 0%,rgba(19,85,179,0) 62%)}
.trust .frame{border:5px solid ${INK};border-top-width:7px;border-top-color:${BRAND}}
/* Fixed padding-top, not centering: the columns have different line counts and
   centering each one independently made the three headlines sit at three
   different heights. */
/* Liga has no mobile variant for the 100% slot, so the whole strip gets scaled
   to ~37% on a phone. Type is sized so the headlines survive that and each
   support line stays on one line. */
.trust .col{
  flex:1;display:flex;flex-direction:column;justify-content:flex-start;
  padding:70px 32px 0;position:relative;z-index:2;
}
.trust .col + .col{box-shadow:inset 2px 0 0 rgba(255,255,255,.10)}
.trust .num{
  width:36px;height:36px;background:${BRAND};color:#fff;border:3px solid ${INK};
  border-radius:8px;display:flex;align-items:center;justify-content:center;
  font-size:11px;box-shadow:3px 3px 0 ${INK};
}
.trust .t{font-size:33px;line-height:1.05;margin-top:17px;color:#fff;text-shadow:3px 3px 0 ${INK}}
.trust .d{font-size:16px;line-height:1.4;color:#c4c4cc;margin-top:11px;white-space:nowrap}
.trust .d b{color:${BRAND_SOFT};font-weight:700}

/* ── triple banners 400x275 ───────────────────────────────────── */
.tri{width:400px;height:275px;display:flex;flex-direction:column;justify-content:flex-end;padding:26px}
.tri .frame{border:5px solid ${INK}}
.tri .kicker{font-size:9px;letter-spacing:1px}
.tri .title{font-size:40px;line-height:.95;margin-top:11px}
.tri .line{font-size:13px;line-height:1.35;margin-top:9px;max-width:236px}
.tri .go{
  display:inline-block;margin-top:17px;font-size:9px;padding:11px 14px;border-radius:8px;
  border:3px solid ${INK};box-shadow:4px 4px 0 ${INK};
}
.tri.pink{background:${BRAND}}
.tri.pink .kicker{color:rgba(255,255,255,.82)}
.tri.pink .title{text-shadow:3px 3px 0 ${INK}}
.tri.pink .line{color:#fff}
.tri.pink .go{background:#fff;color:${INK}}
.tri.blue{background:${ROYAL}}
.tri.blue .kicker{color:${BRAND_SOFT}}
.tri.blue .title{text-shadow:3px 3px 0 ${BRAND},6px 6px 0 ${INK}}
.tri.blue .line{color:${BRAND_SOFT}}
.tri.blue .go{background:${BRAND};color:#fff}
.tri.soft{background:${BRAND_SOFT}}
.tri.soft .kicker{color:${ON_SOFT}}
.tri.soft .title{color:${INK};text-shadow:3px 3px 0 ${BRAND}}
.tri.soft .line{color:${ON_SOFT}}
.tri.soft .go{background:${INK};color:#fff}

/* Flat art props, boxed into the top-right corner so they never reach the
   copy block that grows up from the bottom edge. */
.props{position:absolute;right:20px;top:18px;width:110px;height:110px}
.props div{position:absolute;border:4px solid ${INK};border-radius:6px}
`;

export function logo() {
  return `<div class="logo">
  <div class="wordmark wm">COLLECTA</div>
  <div class="pixel tag">CARDS &amp; COLECIONAVEIS</div>
</div>`;
}

export function headerBG() {
  return `<div class="topbg">
  <div class="fill grid"></div>
  <div class="fill shade"></div>
  <div class="fill scan"></div>
  <div class="fill ticks"></div>
  <div class="fill edge"></div>
</div>`;
}

export function miniatura({ mascot }) {
  return `<div class="mini">
  <div class="fill grid"></div>
  <div class="fill scan"></div>
  <div class="fill frame"></div>
  <div class="disc z"><img src="${mascot}" style="object-position:50% 40%"></div>
  <div class="z" style="display:flex;flex-direction:column;align-items:center;gap:10px">
    <div class="wordmark wm">COLLECTA</div>
    <div class="pixel" style="font-size:8px;letter-spacing:.8px;color:${BRAND_SOFT}">CARDS &amp; COLECIONAVEIS</div>
  </div>
</div>`;
}

export function favicon({ mascot, size }) {
  if (size >= 192) {
    const border = Math.round(size * 0.055);
    return `<div class="fav" style="width:${size}px;height:${size}px;border:${border}px solid ${INK}">
  <img src="${mascot}">
</div>`;
  }
  return `<div class="fav" style="width:${size}px;height:${size}px">
  <span class="letter" style="font-size:${Math.round(size * 1.12)}px;margin-top:${(size * 0.06).toFixed(1)}px">C</span>
</div>`;
}

export function marketplaceLogo({ mascot }) {
  return `<div class="mkt">
  <div class="fill mgrid"></div>
  <div class="fill scan"></div>
  <div class="fill ticks"></div>
  <div class="mdisc z"><img src="${mascot}"></div>
  <span class="wordmark wm z">COLLECTA</span>
</div>`;
}

export function marketplaceAvatar({ mascot }) {
  return `<div class="avatar"><img src="${mascot}"></div>`;
}

const NOTES = [
  "A LOJA DE CARTAS MAIS RAPIDA DO BRASIL",
  "CARTA CONFERIDA UMA A UMA",
  "ENVIO EM 24H",
];

function noteRow(fontSize) {
  return NOTES.map(
    (n, i) =>
      (i ? `<span class="pixel sep">&#9670;</span>` : `<span class="dot"></span>`) +
      `<span class="pixel" style="font-size:${fontSize}px;letter-spacing:.5px">${n}</span>`,
  ).join("");
}

export function stripDesktop() {
  return `<div class="strip" style="width:1170px;height:60px">
  <div class="fill grid"></div>
  <div class="fill scan"></div>
  <div class="fill rule rule-top"></div>
  <div class="fill rule"></div>
  <div class="notes z">${noteRow(11)}</div>
</div>`;
}

export function stripMobile() {
  return `<div class="strip" style="width:400px;height:80px;flex-direction:column;gap:11px">
  <div class="fill grid"></div>
  <div class="fill scan"></div>
  <div class="fill rule rule-top"></div>
  <div class="fill rule"></div>
  <div class="notes z" style="gap:12px">
    <span class="dot"></span>
    <span class="pixel" style="font-size:8px;letter-spacing:.5px">${NOTES[0]}</span>
  </div>
  <div class="notes z" style="gap:12px">
    <span class="pixel" style="font-size:8px;letter-spacing:.5px">${NOTES[1]}</span>
    <span class="pixel sep" style="font-size:8px">&#9670;</span>
    <span class="pixel" style="font-size:8px;letter-spacing:.5px">${NOTES[2]}</span>
  </div>
</div>`;
}

export function hero({ mascot }) {
  return `<div class="hero">
  <div class="fill grid"></div>
  <div class="fill glow"></div>
  <div class="fill scan"></div>
  <div class="fill frame"></div>
  <div class="left z">
    <div class="pixel chip">PLAYER 1 &middot; READY</div>
    <div class="wordmark wm">COLLECTA</div>
    <div class="sub">Singles avulsas e produto selado, <b>conferidos carta por carta</b>.</div>
    <div class="ctas">
      <div class="pixel sticker cta cta-a">VER SINGLES</div>
      <div class="pixel sticker cta cta-b">VER SELADOS</div>
    </div>
  </div>
  <div class="right z">
    <div class="shadowdisc"></div>
    <div class="disc"><img src="${mascot}" style="object-position:50% 42%"></div>
    <div class="pixel tagsticker">PEGA ESSA!</div>
  </div>
</div>`;
}

/* A fanned hand — three cards splayed from a common pivot. */
export function newsHero({ mascot }) {
  return `<div class="nhero">
  <div class="fill grid"></div>
  <div class="fill glow"></div>
  <div class="fill scan"></div>
  <div class="fill frame"></div>
  <div class="left z">
    <div class="pixel chip">PLAYER 1 &middot; READY</div>
    <div class="wordmark wm">COLLECTA</div>
    <div class="sub">Singles avulsas e produto selado, <b>conferidos carta por carta</b>. Pedido rápido, envio pro Brasil todo.</div>
    <div class="ctas">
      <div class="pixel sticker cta cta-a">VER SINGLES</div>
      <div class="pixel sticker cta cta-b">VER SELADOS</div>
    </div>
  </div>
  <div class="right z">
    <div class="shadowdisc"></div>
    <div class="disc"><img src="${mascot}" style="object-position:50% 42%"></div>
    <div class="pixel tagsticker">PEGA ESSA!</div>
  </div>
</div>`;
}

/* Only claims the brand already makes on the storefront: conferida uma a uma,
   envio em 24h, atendimento no WhatsApp. */
const TRUST = [
  { n: "01", t: "CARTA CONFERIDA", d: "Checada <b>uma a uma</b> antes do envio." },
  { n: "02", t: "ENVIO EM 24H", d: "A loja <b>mais rápida</b> do Brasil." },
  { n: "03", t: "FALE NO ZAP", d: "Atendimento no <b>(62) 99339-9980</b>." },
];

export function trustStrip() {
  return `<div class="trust">
  <div class="fill grid"></div>
  <div class="fill glow"></div>
  <div class="fill scan"></div>
  <div class="fill frame"></div>
  ${TRUST.map(
    (c) => `<div class="col">
    <div class="pixel num">${c.n}</div>
    <div class="wordmark t">${c.t}</div>
    <div class="d">${c.d}</div>
  </div>`,
  ).join("")}
</div>`;
}

const HAND = `<div class="props">
  <div style="background:${ROYAL};left:0;top:24px;width:44px;height:62px;transform:rotate(-19deg)"></div>
  <div style="background:${BRAND_SOFT};left:33px;top:16px;width:44px;height:62px"></div>
  <div style="background:#fff;left:66px;top:24px;width:44px;height:62px;transform:rotate(19deg)"></div>
</div>`;

/* A display box — tall body with a lighter lid band, plus one pack tipped out. */
const SEALED = `<div class="props">
  <div style="background:${BRAND};left:6px;top:14px;width:66px;height:84px"></div>
  <div style="background:${BRAND_SOFT};left:6px;top:14px;width:66px;height:22px;border-bottom-left-radius:0;border-bottom-right-radius:0"></div>
  <div style="background:#fff;left:62px;top:44px;width:34px;height:50px;transform:rotate(13deg)"></div>
</div>`;

/* A sleeved card sliding out of an ink sleeve. */
const SLEEVES = `<div class="props">
  <div style="background:${INK};left:4px;top:22px;width:52px;height:74px"></div>
  <div style="background:${ROYAL};left:30px;top:12px;width:48px;height:70px;transform:rotate(9deg)"></div>
  <div style="background:#fff;left:60px;top:26px;width:48px;height:70px;transform:rotate(-6deg)"></div>
</div>`;

const TRIPLE = [
  {
    name: "singles",
    tone: "pink",
    stripes: "stripes-45",
    kicker: "01 &middot; AVULSAS",
    title: "SINGLES",
    line: "Carta por carta, com foto e qualidade conferida.",
    go: "MONTAR DECK",
    props: HAND,
  },
  {
    name: "selados",
    tone: "blue",
    stripes: "stripes-90",
    kicker: "02 &middot; LACRADO",
    title: "SELADOS",
    line: "Boosters, boxes e displays ainda fechados.",
    go: "VER SELADOS",
    props: SEALED,
  },
  {
    name: "acessorios",
    tone: "soft",
    stripes: "stripes-135",
    kicker: "03 &middot; PROTECAO",
    title: "ACESSÓRIOS",
    line: "Sleeves, toploaders e cases pra guardar a coleção.",
    go: "PROTEGER",
    props: SLEEVES,
  },
];

export function triples() {
  return TRIPLE.map((t) => ({
    name: `banner-triplo-${t.name}-400x275`,
    body: `<div class="tri ${t.tone}">
  <div class="fill ${t.stripes}"></div>
  ${t.props}
  <div class="fill frame"></div>
  <div class="z">
    <div class="pixel kicker">${t.kicker}</div>
    <div class="wordmark title">${t.title}</div>
    <div class="line">${t.line}</div>
    <div class="pixel go">${t.go}</div>
  </div>
</div>`,
  }));
}
