import {
  INK, ROYAL, ROYAL_LIGHT, BRAND, BRAND_SOFT, ON_SOFT,
  SITE, INSTAGRAM, COUPON, WHATSAPP,
} from "../lib.mjs";

export const flyerCSS = `
.flyer{width:calc(105mm + 2 * var(--bleed));height:calc(148mm + 2 * var(--bleed))}
.flyer .band{
  background:${INK};color:${BRAND_SOFT};
  font-family:"Press Start 2P",monospace;font-size:1.5mm;line-height:1;letter-spacing:.02em;
  padding:calc(1.9mm + var(--bleed)) 2mm 1.9mm;
  margin:calc(-1 * var(--bleed)) calc(-1 * var(--bleed)) 0;
  display:flex;align-items:center;justify-content:center;gap:2mm;
}
.flyer .band .dot{width:1.1mm;height:1.1mm;border-radius:999px;background:${BRAND}}
.flyer .body{flex:1;display:flex;flex-direction:column;padding:5.5mm 7mm 0}
.flyer .brandrow{display:flex;align-items:center;gap:2.2mm}
.flyer .badge{
  width:9.6mm;height:9.6mm;flex:none;border-width:0.7mm;
  box-shadow:1mm 1mm 0 ${BRAND};
}
.flyer .wordmark{font-size:6.6mm}
.flyer .sep{
  margin-left:auto;text-align:right;
  font-family:"Press Start 2P",monospace;font-size:1.35mm;line-height:1.5;color:${BRAND_SOFT};
}
.flyer .tag{
  align-self:flex-start;margin-top:4mm;
  background:${BRAND_SOFT};color:${INK};
  font-family:"Press Start 2P",monospace;font-size:1.6mm;line-height:1;
  padding:1.5mm 1.9mm;border:0.55mm solid ${INK};border-radius:1.1mm;
  box-shadow:0.85mm 0.85mm 0 ${INK};
}
.flyer .hero{
  font-family:"Baloo 2",sans-serif;font-weight:800;
  font-size:13.4mm;line-height:1;letter-spacing:-.02em;margin-top:2mm;
  text-shadow:0.85mm 0.85mm 0 ${BRAND}, 1.6mm 1.6mm 0 ${INK};
}
.flyer .lead{margin-top:3mm;max-width:82mm;font-size:2.6mm;line-height:1.5;color:${BRAND_SOFT};font-weight:500}
.flyer .lead b{color:#ffffff;font-weight:700}
.flyer .coupon{
  position:relative;margin-top:4mm;background:${BRAND};
  border:0.8mm solid ${INK};border-radius:1.8mm;box-shadow:1.6mm 1.6mm 0 ${INK};
  padding:2.8mm 3mm 3mm;transform:rotate(-1.2deg);
  display:flex;align-items:center;gap:3.2mm;
}
.flyer .coupon-left{flex:none}
.flyer .coupon-kicker{font-family:"Press Start 2P",monospace;font-size:1.35mm;line-height:1;color:${INK}}
.flyer .off{
  font-family:"Baloo 2",sans-serif;font-weight:800;font-size:11mm;line-height:1;
  margin-top:1.2mm;letter-spacing:-.03em;text-shadow:0.75mm 0.75mm 0 ${INK};
}
.flyer .off span{font-size:6.2mm;letter-spacing:-.01em}
.flyer .code{
  flex:1;background:#ffffff;color:${INK};
  border:0.5mm dashed ${INK};border-radius:1mm;padding:2mm 1.4mm;text-align:center;
  font-family:"Press Start 2P",monospace;font-size:3.15mm;line-height:1;letter-spacing:.01em;
}
.flyer .code em{
  display:block;font-style:normal;font-family:"DM Sans",sans-serif;font-weight:700;
  font-size:1.8mm;letter-spacing:.16em;margin-bottom:1.4mm;color:${ON_SOFT};
}
.flyer .mid{display:flex;align-items:center;gap:3mm;margin-top:auto;padding-top:2.5mm}
.flyer .perks{list-style:none;display:flex;flex-direction:column;gap:2.2mm;flex:1}
.flyer .perks li{display:flex;align-items:center;gap:1.8mm;font-size:2.45mm;line-height:1.15;font-weight:700}
.flyer .perks li::before{
  content:"";width:1.8mm;height:1.8mm;flex:none;
  background:${BRAND};border:0.35mm solid ${INK};border-radius:0.35mm;
}
.flyer .mascot{position:relative;flex:none;width:27mm;height:27mm}
.flyer .mascot .disc{width:27mm;height:27mm;box-shadow:1.5mm 1.5mm 0 ${BRAND}}
.flyer .mascot .sticker{
  position:absolute;left:-3.6mm;bottom:-2.8mm;transform:rotate(-6deg);
  background:#ffffff;color:${INK};
  font-family:"Press Start 2P",monospace;font-size:1.5mm;line-height:1;
  padding:1.4mm 1.6mm;border:0.5mm solid ${INK};border-radius:1mm;box-shadow:0.8mm 0.8mm 0 ${INK};
}
.flyer .qrrow{display:flex;align-items:center;gap:3.2mm;padding:3mm 0 3.5mm}
.flyer .qrbox{
  width:20mm;height:20mm;flex:none;background:#ffffff;
  border:0.7mm solid ${INK};border-radius:1.1mm;box-shadow:1.2mm 1.2mm 0 ${INK};padding:1.3mm;
}
.flyer .qrbox svg{display:block}
.flyer .qrtext .url{
  font-family:"Baloo 2",sans-serif;font-weight:800;font-size:4.6mm;line-height:1.05;
  text-shadow:0.45mm 0.45mm 0 ${INK};
}
.flyer .qrtext p{margin-top:1.2mm;font-size:2.3mm;line-height:1.35;color:${BRAND_SOFT};font-weight:500}
.flyer .steps{
  background:${INK};margin:0 calc(-1 * var(--bleed)) calc(-1 * var(--bleed));
  padding:2.6mm 4mm calc(2.6mm + var(--bleed));
  display:flex;flex-direction:column;align-items:center;gap:1.7mm;
}
.flyer .steps .row{
  display:flex;align-items:center;justify-content:center;gap:2.4mm;
  font-family:"Press Start 2P",monospace;font-size:1.4mm;line-height:1.3;color:${BRAND_SOFT};
}
.flyer .steps .n{color:${BRAND}}
.flyer .steps .fine{font-family:"DM Sans",sans-serif;font-size:1.95mm;color:#9a9aa2;font-weight:500}
.flyer .steps .fine b{color:#ffffff;font-weight:700}
`;

export function flyer({ mascot, qr, bleed }) {
  return `<div class="art grain flyer" style="--bleed:${bleed}">
  <div class="trim">
    <div class="band"><span class="dot"></span>OBRIGADO POR JOGAR COM A GENTE<span class="dot"></span></div>
    <div class="body">
      <div class="brandrow">
        <div class="disc badge"><img src="${mascot}" alt=""></div>
        <div class="wordmark">COLLECTA</div>
        <div class="sep">SINGLES<br>&amp; SELADOS</div>
      </div>
      <div class="tag">STAGE 1 CLEAR!</div>
      <div class="hero">CONTINUE?</div>
      <p class="lead">Valeu pela compra! Da próxima vez, monte seu pedido <b>direto no site</b> e leve <b>5% de desconto</b>.</p>
      <div class="coupon">
        <div class="coupon-left">
          <div class="coupon-kicker">CUPOM</div>
          <div class="off">5%<span> OFF</span></div>
        </div>
        <div class="code"><em>DIGITE</em>${COUPON}</div>
      </div>
      <div class="mid">
        <ul class="perks">
          <li>Envio em 24h</li>
          <li>Carta conferida uma a uma</li>
          <li>Pokémon, One Piece, Riftbound e mais</li>
        </ul>
        <div class="mascot">
          <div class="disc"><img src="${mascot}" alt=""></div>
          <span class="sticker">VOLTA LOGO!</span>
        </div>
      </div>
      <div class="qrrow">
        <div class="qrbox">${qr}</div>
        <div class="qrtext">
          <div class="url">${SITE}</div>
          <p>Aponte a câmera do celular<br>e caia direto na loja.</p>
        </div>
      </div>
    </div>
    <div class="steps">
      <div class="row"><span><span class="n">1</span> MONTE O CARRINHO</span><span><span class="n">2</span> FINALIZE NO ZAP</span><span><span class="n">3</span> MANDE O CUPOM</span></div>
      <div class="fine">WhatsApp <b>${WHATSAPP}</b> &middot; Instagram <b>${INSTAGRAM}</b> &middot; cupom válido em todo pedido pelo site.</div>
    </div>
  </div>
</div>`;
}
