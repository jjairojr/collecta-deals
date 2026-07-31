import { readFile, writeFile } from "node:fs/promises";
import { execFile } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";
import QRCode from "qrcode";

const run = promisify(execFile);

export const here = dirname(fileURLToPath(import.meta.url));
export const storefront = join(here, "..", "..");
export const og = join(storefront, "app", "og");
export const out = join(here, "dist");

const CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";

export const SITE = "collectatcg.com.br";
export const INSTAGRAM = "@tcgcollecta";
export const COUPON = "CONTINUE5";
export const WHATSAPP = "(62) 99339-9980";

export const INK = "#0b0b0c";
export const ROYAL = "#1355b3";
export const ROYAL_LIGHT = "#2f74d8";
export const BRAND = "#f6559b";
export const BRAND_SOFT = "#fdc4e5";
export const ON_SOFT = "#8a3f68";

export function target(medium) {
  return `https://www.${SITE}/?utm_source=flyer&utm_medium=${medium}`;
}

async function dataURI(path, mime) {
  const buf = await readFile(path);
  return `data:${mime};base64,${buf.toString("base64")}`;
}

async function fontFace(family, file, weight) {
  const uri = await dataURI(join(og, file), "font/ttf");
  return `@font-face{font-family:"${family}";font-weight:${weight};font-style:normal;font-display:block;src:url(${uri}) format("truetype")}`;
}

export async function loadAssets() {
  const [baloo, pixel, dmMedium, dmBold, mascot] = await Promise.all([
    fontFace("Baloo 2", "baloo2-extrabold.ttf", 800),
    fontFace("Press Start 2P", "press-start-2p.ttf", 400),
    fontFace("DM Sans", "dm-sans-medium.ttf", 500),
    fontFace("DM Sans", "dm-sans-bold.ttf", 700),
    dataURI(join(storefront, "public", "mascot.png"), "image/png"),
  ]);
  return { fonts: [baloo, pixel, dmMedium, dmBold].join("\n"), mascot };
}

export async function qrSVG(url) {
  const raw = await QRCode.toString(url, {
    type: "svg",
    margin: 0,
    errorCorrectionLevel: "M",
  });
  return raw
    .replace(/<\?xml[^>]*\?>/, "")
    .replace("<svg", '<svg preserveAspectRatio="xMidYMid meet" width="100%" height="100%"')
    .replace(/#000000/g, INK);
}

export const ICON_WHATSAPP = `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/></svg>`;

export const ICON_INSTAGRAM = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.1"><rect x="2.2" y="2.2" width="19.6" height="19.6" rx="5.6"/><circle cx="12" cy="12" r="4.9"/><circle cx="17.6" cy="6.4" r="1.35" fill="currentColor" stroke="none"/></svg>`;

export const baseCSS = `
*{margin:0;padding:0;box-sizing:border-box}
html,body{background:#3a3a40}
body.sheet{background:#ffffff}
.art{
  position:relative;overflow:hidden;
  background-color:${ROYAL};color:#ffffff;
  font-family:"DM Sans",sans-serif;-webkit-font-smoothing:antialiased;
  padding:var(--bleed);
}
.art.grain::before{
  content:"";position:absolute;inset:0;
  background-image:
    radial-gradient(circle at 88% 16%, rgba(246,85,155,.70) 0%, rgba(246,85,155,0) 46%),
    radial-gradient(circle at 4% 88%, rgba(47,116,216,.85) 0%, rgba(47,116,216,0) 40%),
    repeating-linear-gradient(90deg, rgba(255,255,255,.075) 0 0.45mm, transparent 0.45mm 8mm);
}
.art.grain::after{
  content:"";position:absolute;inset:0;
  background-image:repeating-linear-gradient(180deg, rgba(11,11,12,.10) 0 0.18mm, transparent 0.18mm 0.62mm);
}
.trim{position:relative;z-index:2;display:flex;flex-direction:column;height:100%}
.wordmark{
  font-family:"Baloo 2",sans-serif;font-weight:800;line-height:.9;letter-spacing:-.01em;
  text-shadow:0.5mm 0.5mm 0 ${ROYAL}, 0.95mm 0.95mm 0 ${INK};
}
.pixel{font-family:"Press Start 2P",monospace;font-weight:400;line-height:1}
.disc{
  border-radius:999px;overflow:hidden;
  border:0.8mm solid ${INK};background:${ROYAL_LIGHT};
}
.disc img{width:100%;height:100%;object-fit:cover;display:block}
`;

export function page({ title, css, bodyClass, body, pageCSS }) {
  return `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8">
<title>${title}</title>
<style>${css}${pageCSS ?? ""}</style>
</head><body class="${bodyClass ?? ""}">${body}</body></html>`;
}

async function chrome(args) {
  await run(CHROME, [
    "--headless=new",
    "--disable-gpu",
    "--no-sandbox",
    "--hide-scrollbars",
    "--force-color-profile=srgb",
    ...args,
  ]);
}

export async function emit({ name, html, pdf, png }) {
  const htmlPath = join(out, `${name}.html`);
  await writeFile(htmlPath, html);
  if (pdf) {
    await chrome([
      `--print-to-pdf=${join(out, `${name}.pdf`)}`,
      "--no-pdf-header-footer",
      `file://${htmlPath}`,
    ]);
  }
  if (png) {
    await chrome([
      `--screenshot=${join(out, `${name}.png`)}`,
      `--window-size=${png.w},${png.h}`,
      `--force-device-scale-factor=${png.scale ?? 4}`,
      `file://${htmlPath}`,
    ]);
  }
  return htmlPath;
}

export const MM = 3.7795275591;

export function gridSheet({ cols, rows, w, h, cell }) {
  return `<div class="grid" style="display:grid;grid-template-columns:repeat(${cols},${w}mm);grid-template-rows:repeat(${rows},${h}mm)">${Array.from(
    { length: cols * rows },
    () => cell,
  ).join("")}</div>`;
}

export const sheetCSS = `
.sheet{width:210mm;height:297mm;display:flex;flex-direction:column;align-items:center;justify-content:center}
.sheet .art{--bleed:0mm}
.grid{outline:0.2mm solid #cfcfcf}
.cutgap{gap:0}
`;
