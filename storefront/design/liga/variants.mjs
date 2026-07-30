import { readFile } from "node:fs/promises";
import { join } from "node:path";
import {
  out, loadAssets, baseCSS, page, shot, INK, BRAND, BRAND_SOFT, ROYAL,
} from "./lib.mjs";
import { artCSS, logo } from "./art.mjs";

/* Candidates for the 200x85 store logo. The header renders it at 170x72
   (0.85x), so every candidate is judged at that size over the real header
   background — not at 1:1. */

const variantCSS = `
.lg2,.lg3,.lg4{
  width:200px;height:85px;position:relative;overflow:hidden;
  display:flex;align-items:center;justify-content:center;
}
.lg2 .disc,.lg3 .disc,.lg4 .disc{border-width:4px;box-shadow:3px 3px 0 ${INK}}
/* mascot.png carries its own brand-sheet backdrop. At 46-78px those pink/blue
   shards read as confetti behind the head, so the small discs crop in until the
   face fills the circle. */
.lg2 .disc img,.lg3 .disc img,.lg4 .disc img{
  width:138%;height:138%;min-width:138%;object-position:50% 37%;
}

/* B — disc + wordmark on one line */
.lg2{gap:9px}
.lg2 .disc{width:64px;height:64px}
.lg2 .wm{font-size:25px;text-shadow:2px 2px 0 ${BRAND},4px 4px 0 ${INK}}

/* C — disc + wordmark split on two lines, like the brand-sheet app icons */
.lg3{gap:11px}
.lg3 .disc{width:78px;height:78px;box-shadow:4px 4px 0 ${INK}}
.lg3 .stack{display:flex;flex-direction:column;line-height:.86}
.lg3 .wm{font-size:36px;text-shadow:3px 3px 0 ${BRAND},5px 5px 0 ${INK}}

/* D — wordmark leading, small mascot trailing */
.lg4{gap:5px}
.lg4 .disc{width:46px;height:46px;border-width:3px}
.lg4 .wm{font-size:31px;text-shadow:2px 2px 0 ${BRAND},4px 4px 0 ${INK}}
`;

const disc = (m, size) =>
  `<div class="disc" style="width:${size}px;height:${size}px"><img src="${m}" style="object-position:50% 40%"></div>`;

const CANDIDATES = [
  { name: "logo-A-atual", label: "A · WORDMARK + LINHA (ATUAL)", body: () => logo() },
  {
    name: "logo-B-disco-linha",
    label: "B · MASCOTE + WORDMARK",
    body: (m) => `<div class="lg2">${disc(m, 64)}<div class="wordmark wm">COLLECTA</div></div>`,
  },
  {
    name: "logo-C-disco-empilhado",
    label: "C · MASCOTE + COLL/ECTA",
    body: (m) =>
      `<div class="lg3">${disc(m, 78)}<div class="stack"><div class="wordmark wm">COLL</div><div class="wordmark wm">ECTA</div></div></div>`,
  },
  {
    name: "logo-D-wordmark-mascote",
    label: "D · WORDMARK + MASCOTE PEQUENO",
    body: (m) => `<div class="lg4"><div class="wordmark wm">COLLECTA</div>${disc(m, 46)}</div>`,
  },
];

async function main() {
  const { fonts, mascot } = await loadAssets();
  const css = [fonts, baseCSS, artCSS, variantCSS].join("\n");

  for (const c of CANDIDATES) {
    if (c.name === "logo-A-atual") continue;
    await shot({
      name: c.name,
      w: 200,
      h: 85,
      html: page({ w: 200, h: 85, css, body: c.body(mascot) }),
      transparent: true,
    });
  }

  const top = `data:image/png;base64,${(await readFile(join(out, "header-bg-1920x110.png"))).toString("base64")}`;
  const shots = await Promise.all(
    CANDIDATES.map(async (c) => {
      const file = c.name === "logo-A-atual" ? "logo-200x85.png" : `${c.name}.png`;
      const buf = await readFile(join(out, file));
      return { ...c, uri: `data:image/png;base64,${buf.toString("base64")}` };
    }),
  );

  const sheetCSS = `
body{background:#e8e8ec;font-family:"DM Sans",sans-serif;overflow:visible}
.sheet{width:1180px;padding:24px 28px 34px;display:flex;flex-direction:column;gap:20px}
.h{font-family:"Baloo 2",sans-serif;font-weight:800;font-size:18px;color:#1a1a1e}
.h span{font-family:"DM Sans";font-weight:500;font-size:12px;color:#6f6f77;margin-left:8px}
.card{background:#fff;border:1px solid #cfcfd6;border-radius:10px;overflow:hidden}
.strip{height:110px;position:relative;background-repeat:repeat}
.strip .slot{position:absolute;left:56px;top:19px;width:170px;height:72px}
.strip .slot img{width:100%;height:100%;object-fit:contain;display:block}
.strip .mob{position:absolute;left:300px;top:8px;width:255px;height:108px}
.strip .mob img{width:100%;height:100%;object-fit:contain;display:block}
.strip .note{
  position:absolute;right:26px;top:44px;font-family:"Press Start 2P";
  font-size:9px;color:${BRAND_SOFT};
}
.cap{padding:10px 14px;font-family:"Press Start 2P";font-size:9px;color:#4a4a52}
.raw{background:${ROYAL};padding:12px;display:flex;gap:14px;align-items:center}
.raw img{display:block}
`;

  const body = `<div class="sheet">
  <div class="h">Logo com o mascote — 4 candidatos <span>cada faixa é o cabeçalho real: slot desktop 170×72 à esquerda, mobile ~255×108 ao centro</span></div>
  ${shots
    .map(
      (s) => `<div class="card">
    <div class="cap">${s.label}</div>
    <div class="strip" style="background-image:url(${top})">
      <div class="slot"><img src="${s.uri}"></div>
      <div class="mob"><img src="${s.uri}"></div>
      <div class="note">&larr; DESKTOP &nbsp;&nbsp; MOBILE &rarr;</div>
    </div>
    <div class="raw"><img src="${s.uri}" width="200" height="85"><span style="font-family:'Press Start 2P';font-size:8px;color:${BRAND_SOFT}">1:1 &mdash; 200x85</span></div>
  </div>`,
    )
    .join("")}
</div>`;

  await shot({
    name: "PROVA-logos",
    w: 1180,
    h: 1310,
    html: page({ w: 1180, h: 1310, css: [fonts, baseCSS, artCSS, variantCSS, sheetCSS].join("\n"), body }),
  });
  console.log("dist/PROVA-logos.png");
}

main();
