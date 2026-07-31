import { mkdir, stat } from "node:fs/promises";
import { basename } from "node:path";
import { out, loadAssets, baseCSS, page, shot, toJpeg } from "./lib.mjs";
import {
  artCSS, logo, headerBG, miniatura, favicon, marketplaceLogo,
  marketplaceAvatar, stripDesktop, stripMobile, hero, newsHero,
  trustStrip, triples,
} from "./art.mjs";

const FAVICONS = [16, 32, 48, 192, 512];

async function main() {
  await mkdir(out, { recursive: true });
  const { fonts, mascot } = await loadAssets();
  const css = [fonts, baseCSS, artCSS].join("\n");

  const pieces = [
    { name: "logo-200x85", w: 200, h: 85, body: logo(), transparent: true },
    { name: "header-bg-1920x110", w: 1920, h: 110, body: headerBG() },
    { name: "miniatura-300x300", w: 300, h: 300, body: miniatura({ mascot }), jpeg: 90 },
    { name: "marketplace-logo-101x30", w: 101, h: 30, body: marketplaceLogo({ mascot }), jpeg: 95 },
    { name: "marketplace-avatar-55x55", w: 55, h: 55, body: marketplaceAvatar({ mascot }), jpeg: 88 },
    { name: "banner-superior-1170x60", w: 1170, h: 60, body: stripDesktop() },
    { name: "banner-superior-mobile-400x80", w: 400, h: 80, body: stripMobile() },
    { name: "banner-hero-1170x275", w: 1170, h: 275, body: hero({ mascot }) },
    { name: "noticia-hero-1170x360", w: 1170, h: 360, body: newsHero({ mascot }) },
    { name: "banner-full-confianca-1170x275", w: 1170, h: 275, body: trustStrip() },
    ...triples().map((t) => ({ name: t.name, w: 400, h: 275, body: t.body })),
    ...FAVICONS.map((size) => ({
      name: `favicon-${size}`,
      w: size,
      h: size,
      body: favicon({ mascot, size }),
    })),
  ];

  const report = [];
  for (const p of pieces) {
    const png = await shot({
      name: p.name,
      w: p.w,
      h: p.h,
      html: page({ w: p.w, h: p.h, css, body: p.body }),
      transparent: p.transparent,
    });
    report.push(png);
    if (p.jpeg) report.push(await toJpeg({ png, name: p.name, quality: p.jpeg }));
  }

  for (const file of report) {
    const { size } = await stat(file);
    console.log(`${(size / 1024).toFixed(1).padStart(7)} kb  ${basename(file)}`);
  }
}

main();
