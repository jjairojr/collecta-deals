import {
  INK, ROYAL, ROYAL_LIGHT, BRAND, BRAND_SOFT, ON_SOFT,
  SITE, INSTAGRAM, WHATSAPP, ICON_WHATSAPP, ICON_INSTAGRAM,
} from "../lib.mjs";

export const cardCSS = `
.card{
  width:calc(90mm + 2 * var(--bleed));height:calc(50mm + 2 * var(--bleed));
  background-color:${INK};padding:calc(var(--bleed) + 2.6mm);
}
.card .face{
  position:relative;z-index:2;height:100%;overflow:hidden;
  border:0.7mm solid ${INK};border-radius:2.2mm;
  background-color:${ROYAL};
  display:flex;flex-direction:column;
}
.card .face::before{
  content:"";position:absolute;inset:0;
  background-image:
    radial-gradient(circle at 86% 14%, rgba(246,85,155,.62) 0%, rgba(246,85,155,0) 54%),
    repeating-linear-gradient(90deg, rgba(255,255,255,.08) 0 0.4mm, transparent 0.4mm 6mm);
}
.card .plate{position:relative;z-index:2;display:flex;flex-direction:column;height:100%;padding:2.4mm}

.card .head{display:flex;align-items:center;gap:2mm}
.card .head .name{
  font-family:"Baloo 2",sans-serif;font-weight:800;font-size:5.4mm;line-height:1;
  letter-spacing:-.01em;text-shadow:0.4mm 0.4mm 0 ${ROYAL}, 0.8mm 0.8mm 0 ${INK};
}
.card .head .pips{margin-left:auto;display:flex;gap:1.1mm}
.card .head .pips i{
  width:2.2mm;height:2.2mm;transform:rotate(45deg);
  background:${BRAND};border:0.35mm solid ${INK};border-radius:0.3mm;
}

.card .main{flex:1;display:flex;gap:2.4mm;margin-top:2mm;min-height:0}
.card .window{
  width:27mm;flex:none;border:0.6mm solid ${INK};border-radius:1.2mm;
  background:${ROYAL_LIGHT};overflow:hidden;box-shadow:0.9mm 0.9mm 0 rgba(11,11,12,.45);
}
.card .window img{width:100%;height:100%;object-fit:cover;display:block}
.card .col{flex:1;display:flex;flex-direction:column;gap:1.5mm;min-width:0}
.card .type{
  background:${INK};color:${BRAND_SOFT};border-radius:0.9mm;
  font-family:"Press Start 2P",monospace;font-size:1.25mm;line-height:1;
  padding:1.5mm 1.6mm;letter-spacing:.02em;
}
.card .box{
  flex:1;background:${BRAND_SOFT};color:${INK};
  border:0.55mm solid ${INK};border-radius:1.2mm;
  padding:1.9mm;display:flex;flex-direction:column;justify-content:center;gap:1.9mm;
}
.card .box .ab{
  display:flex;align-items:center;gap:1.5mm;
  font-family:"Press Start 2P",monospace;font-size:1.5mm;line-height:1.35;
}
.card .box .ab::before{
  content:"";width:1.5mm;height:1.5mm;flex:none;transform:rotate(45deg);
  background:${BRAND};border:0.3mm solid ${INK};
}
.card .box .rule{height:0.3mm;background:rgba(11,11,12,.28);border-radius:999px}
.card .box .flavor{font-size:2mm;line-height:1.25;font-style:italic;color:${ON_SOFT}}

.card .foot{
  display:flex;align-items:center;gap:2mm;margin-top:1.8mm;
  font-family:"Press Start 2P",monospace;font-size:1.2mm;line-height:1;color:#ffffff;
}
.card .foot .set{color:${BRAND_SOFT}}
.card .foot .site{margin-left:auto}

.card.back .face{background-color:${ROYAL}}
.card.back .face::before{
  background-image:
    radial-gradient(circle at 50% 50%, rgba(246,85,155,.55) 0%, rgba(246,85,155,0) 62%),
    repeating-linear-gradient(45deg, rgba(11,11,12,.16) 0 1.6mm, transparent 1.6mm 3.2mm);
}
.card.back .ring{
  position:absolute;z-index:2;inset:1.6mm;border:0.5mm solid rgba(255,255,255,.34);border-radius:1.4mm;
}
.card.back .ring::before{
  content:"";position:absolute;inset:0.9mm;border:0.35mm dashed rgba(255,255,255,.24);border-radius:1mm;
}
.card.back .plate{align-items:center;justify-content:center;padding:3mm}
.card .label{
  position:relative;z-index:3;background:#ffffff;color:${INK};
  border:0.7mm solid ${INK};border-radius:1.4mm;box-shadow:1.1mm 1.1mm 0 ${BRAND};
  transform:rotate(-1.4deg);padding:2.4mm 2.8mm;
  display:flex;align-items:center;gap:2.8mm;
}
.card .label .qrbox{width:16mm;height:16mm;flex:none}
.card .label .qrbox svg{display:block}
.card .contact{display:flex;flex-direction:column;gap:1.8mm;min-width:0}
.card .contact .url{
  font-family:"Press Start 2P",monospace;font-size:2.2mm;line-height:1;white-space:nowrap;
  color:${INK};
}
.card .contact .line{
  display:flex;align-items:center;gap:1.5mm;white-space:nowrap;
  font-family:"Press Start 2P",monospace;font-size:1.8mm;line-height:1;
}
.card .contact .line .ic{
  width:3.9mm;height:3.9mm;flex:none;border-radius:999px;background:${INK};color:#ffffff;
  display:flex;align-items:center;justify-content:center;
}
.card .contact .line .ic svg{width:2.5mm;height:2.5mm;display:block}
.card .contact .tail{
  font-family:"Press Start 2P",monospace;font-size:1.15mm;line-height:1;color:${ON_SOFT};white-space:nowrap;
}
`;

export function cardFront({ mascot, bleed }) {
  return `<div class="art card front" style="--bleed:${bleed}">
  <div class="face">
    <div class="plate">
      <div class="head">
        <div class="name">COLLECTA</div>
        <div class="pips"><i></i><i></i><i></i></div>
      </div>
      <div class="main">
        <div class="window"><img src="${mascot}" alt=""></div>
        <div class="col">
          <div class="type">LOJA DE CARTAS &middot; SINGLES &amp; SELADOS</div>
          <div class="box">
            <div class="ab">Envio em 24h</div>
            <div class="ab">Carta conferida uma a uma</div>
            <div class="rule"></div>
            <div class="flavor">&ldquo;Sua coleção começa aqui.&rdquo;</div>
          </div>
        </div>
      </div>
      <div class="foot">
        <span class="set">COL-001</span>
        <span>POKEMON &middot; ONE PIECE &middot; RIFTBOUND</span>
        <span class="site">${SITE.toUpperCase()}</span>
      </div>
    </div>
  </div>
</div>`;
}

export function cardBack({ qr, bleed }) {
  return `<div class="art card back" style="--bleed:${bleed}">
  <div class="face">
    <div class="ring"></div>
    <div class="plate">
      <div class="label">
        <div class="qrbox">${qr}</div>
        <div class="contact">
          <div class="url">${SITE}</div>
          <div class="line"><span class="ic">${ICON_WHATSAPP}</span>${WHATSAPP}</div>
          <div class="line"><span class="ic">${ICON_INSTAGRAM}</span>${INSTAGRAM}</div>
          <div class="tail">ENVIO EM 24H</div>
        </div>
      </div>
    </div>
  </div>
</div>`;
}
