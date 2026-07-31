import { mkdir } from "node:fs/promises";
import {
  out, loadAssets, qrSVG, target, baseCSS, sheetCSS, page, emit, gridSheet,
} from "./lib.mjs";
import { flyerCSS, flyer } from "./art/flyer.mjs";
import { cardCSS, cardFront, cardBack } from "./art/card.mjs";
import { stickersCSS, stickers } from "./art/stickers.mjs";

const px = (mm) => Math.round(mm * 3.7795275591);

async function main() {
  await mkdir(out, { recursive: true });

  const { fonts, mascot } = await loadAssets();
  const [qrFlyer, qrCard, qrSticker] = await Promise.all([
    qrSVG(target("print")),
    qrSVG(target("cartao")),
    qrSVG(target("adesivo")),
  ]);

  const css = [fonts, baseCSS, sheetCSS, flyerCSS, cardCSS, stickersCSS].join("\n");
  const proofCSS = `
body.proof{width:96mm;display:flex;flex-direction:column;align-items:center;gap:6mm;padding:0}
body.proof .art{--bleed:3mm}
`;

  await emit({
    name: "flyer-a6-sangria",
    html: page({
      title: "Collecta — flyer A6 (sangria 3mm)",
      css,
      body: flyer({ mascot, qr: qrFlyer, bleed: "3mm" }),
      pageCSS: "@page{size:111mm 154mm;margin:0}body{width:111mm;height:154mm}",
    }),
    pdf: true,
    png: { w: px(111), h: px(154) },
  });

  await emit({
    name: "flyer-a4-4up",
    html: page({
      title: "Collecta — flyer A6 4-up em A4",
      css,
      bodyClass: "sheet",
      body: gridSheet({
        cols: 2, rows: 2, w: 105, h: 148,
        cell: flyer({ mascot, qr: qrFlyer, bleed: "0mm" }),
      }),
      pageCSS: "@page{size:A4;margin:0}",
    }),
    pdf: true,
  });

  await emit({
    name: "cartao-frente-verso-sangria",
    html: page({
      title: "Collecta — cartão de visita 90x50mm (sangria 3mm)",
      css,
      body:
        `<div class="pg">${cardFront({ mascot, bleed: "3mm" })}</div>` +
        `<div>${cardBack({ qr: qrCard, bleed: "3mm" })}</div>`,
      pageCSS:
        "@page{size:96mm 56mm;margin:0}body{width:96mm}.pg{break-after:page}",
    }),
    pdf: true,
  });

  await emit({
    name: "cartao-preview",
    html: page({
      title: "Collecta — cartão de visita (frente e verso)",
      css,
      bodyClass: "proof",
      body: cardFront({ mascot, bleed: "3mm" }) + cardBack({ qr: qrCard, bleed: "3mm" }),
      pageCSS: proofCSS,
    }),
    png: { w: px(96), h: px(56 * 2 + 6) },
  });

  await emit({
    name: "cartao-a4-10up",
    html: page({
      title: "Collecta — cartão 10-up em A4 (frente + verso)",
      css,
      bodyClass: "sheet",
      body:
        `<div class="pg">${gridSheet({
          cols: 2, rows: 5, w: 90, h: 50,
          cell: cardFront({ mascot, bleed: "0mm" }),
        })}</div>` +
        gridSheet({
          cols: 2, rows: 5, w: 90, h: 50,
          cell: cardBack({ qr: qrCard, bleed: "0mm" }),
        }),
      pageCSS:
        "@page{size:A4;margin:0}body.sheet{display:block}.pg{break-after:page}" +
        "body.sheet > *{width:210mm;height:297mm;display:flex;align-items:center;justify-content:center}",
    }),
    pdf: true,
  });

  await emit({
    name: "adesivos-a6-sangria",
    html: page({
      title: "Collecta — cartela de adesivos A6 (sangria 3mm)",
      css,
      body: stickers({ mascot, qr: qrSticker, bleed: "3mm" }),
      pageCSS: "@page{size:111mm 154mm;margin:0}body{width:111mm;height:154mm}",
    }),
    pdf: true,
    png: { w: px(111), h: px(154) },
  });

  await emit({
    name: "adesivos-a4-4up",
    html: page({
      title: "Collecta — cartela de adesivos 4-up em A4",
      css,
      bodyClass: "sheet",
      body: gridSheet({
        cols: 2, rows: 2, w: 105, h: 148,
        cell: stickers({ mascot, qr: qrSticker, bleed: "0mm" }),
      }),
      pageCSS: "@page{size:A4;margin:0}",
    }),
    pdf: true,
  });

  console.log("ok");
}

main();
