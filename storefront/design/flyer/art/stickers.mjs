import {
  INK, ROYAL_LIGHT, BRAND, BRAND_SOFT, SITE, INSTAGRAM, ICON_INSTAGRAM,
} from "../lib.mjs";

export const STICKER_COUNT = 12;

export const stickersCSS = `
.pack{width:calc(105mm + 2 * var(--bleed));height:calc(148mm + 2 * var(--bleed))}
.pack .band{
  background:${INK};color:${BRAND_SOFT};
  font-family:"Press Start 2P",monospace;font-size:1.5mm;line-height:1;letter-spacing:.02em;
  padding:calc(1.9mm + var(--bleed)) 2mm 1.9mm;
  margin:calc(-1 * var(--bleed)) calc(-1 * var(--bleed)) 0;
  display:flex;align-items:center;justify-content:center;gap:2mm;
}
.pack .band .dot{width:1.1mm;height:1.1mm;border-radius:999px;background:${BRAND}}
.pack .foot{
  background:${INK};margin:0 calc(-1 * var(--bleed)) calc(-1 * var(--bleed));
  padding:1.9mm 3mm calc(1.9mm + var(--bleed));text-align:center;
  font-family:"Press Start 2P",monospace;font-size:1.3mm;line-height:1;color:#ffffff;letter-spacing:.02em;
}
.pack .foot span{color:${BRAND}}
.pack .body{
  flex:1;display:flex;flex-direction:column;justify-content:space-between;
  padding:5mm 6mm;
}
.pack .row{display:flex;align-items:center;gap:4mm}
.pack .stack{display:flex;flex-direction:column;gap:3mm;flex:1}

.st{background:#ffffff;padding:1.1mm;display:flex;align-self:flex-start}
.st > *{
  border:0.7mm solid ${INK};display:flex;align-items:center;justify-content:center;
}
.st.block{width:100%}
.st.block > *{width:100%}
.st.round,.st.round > *{border-radius:999px}
.st.pill{border-radius:3.2mm}
.st.pill > *{border-radius:2.4mm}
.st.plate{border-radius:2.6mm}
.st.plate > *{border-radius:1.8mm}

.s-word > *{
  background:${BRAND};flex-direction:column;gap:1.4mm;padding:3.4mm 2mm;
}
.s-word .mark{
  font-family:"Baloo 2",sans-serif;font-weight:800;font-size:9.6mm;line-height:.9;
  color:#ffffff;letter-spacing:-.01em;text-shadow:0.8mm 0.8mm 0 ${INK};
}
.s-word .sub{font-family:"Press Start 2P",monospace;font-size:1.5mm;line-height:1;color:${INK}}

.s-mascot{flex:none}
.s-mascot > *{width:30mm;height:30mm;background:${ROYAL_LIGHT};overflow:hidden;padding:0}
.s-mascot img{width:100%;height:100%;object-fit:cover;display:block}

.s-tag > *{
  font-family:"Press Start 2P",monospace;font-size:2.1mm;line-height:1;padding:2.6mm 3mm;white-space:nowrap;
}
.s-tag.white > *{background:#ffffff;color:${INK}}
.s-tag.soft > *{background:${BRAND_SOFT};color:${INK}}
.s-tag.royal > *{background:${ROYAL_LIGHT};color:#ffffff}
.s-tag.pink > *{background:${BRAND};color:#ffffff}

.s-qr{flex:none}
.s-qr > *{width:24mm;height:24mm;background:#ffffff;padding:1.6mm}
.s-qr svg{display:block}
.s-ig svg{width:4.2mm;height:4.2mm;display:block;flex:none}

.s-url > *{
  background:${INK};color:#ffffff;padding:2.4mm 3mm;
  font-family:"Baloo 2",sans-serif;font-weight:800;font-size:4.4mm;line-height:1;
  text-shadow:0.45mm 0.45mm 0 ${BRAND};
}
.s-ig > *{
  background:${BRAND};color:#ffffff;padding:2.4mm 3mm;gap:1.8mm;
  font-family:"Baloo 2",sans-serif;font-weight:800;font-size:4.2mm;line-height:1;
  text-shadow:0.45mm 0.45mm 0 ${INK};
}
.s-coin{flex:none}
.s-coin > *{
  width:13.5mm;height:13.5mm;font-family:"Press Start 2P",monospace;font-size:2.3mm;line-height:1;
}
.s-coin.pink > *{background:${BRAND};color:#ffffff}
.s-coin.soft > *{background:${BRAND_SOFT};color:${INK}}
.s-coin.white > *{background:#ffffff;color:${INK}}
`;

export function stickers({ mascot, qr, bleed }) {
  return `<div class="art grain pack" style="--bleed:${bleed}">
  <div class="trim">
    <div class="band"><span class="dot"></span>COLLECTA STICKER PACK<span class="dot"></span></div>
    <div class="body">
      <div class="st plate s-word block"><div>
        <div class="mark">COLLECTA</div>
        <div class="sub">SINGLES &amp; SELADOS</div>
      </div></div>

      <div class="row">
        <div class="st round s-mascot"><div><img src="${mascot}" alt=""></div></div>
        <div class="stack">
          <div class="st pill s-tag white"><div>PEGA ESSA!</div></div>
          <div class="st pill s-tag soft"><div>VOLTA LOGO!</div></div>
          <div class="st pill s-tag royal"><div>INSERT COIN</div></div>
        </div>
      </div>

      <div class="row">
        <div class="st plate s-qr"><div>${qr}</div></div>
        <div class="stack">
          <div class="st plate s-url"><div>${SITE}</div></div>
          <div class="st plate s-ig"><div>${ICON_INSTAGRAM}<span>${INSTAGRAM}</span></div></div>
        </div>
      </div>

      <div class="row">
        <div class="st round s-coin pink"><div>P1</div></div>
        <div class="st round s-coin soft"><div>&lt;3</div></div>
        <div class="st round s-coin white"><div>GG!</div></div>
        <div class="st pill s-tag pink"><div>STAGE CLEAR</div></div>
      </div>
    </div>
    <div class="foot">${STICKER_COUNT} ADESIVOS <span>&middot;</span> ${SITE.toUpperCase()} <span>&middot;</span> ${INSTAGRAM.toUpperCase()}</div>
  </div>
</div>`;
}
