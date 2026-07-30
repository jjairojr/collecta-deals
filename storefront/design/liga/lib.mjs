import { readFile, writeFile } from "node:fs/promises";
import { execFile } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

const run = promisify(execFile);

export const here = dirname(fileURLToPath(import.meta.url));
export const storefront = join(here, "..", "..");
export const og = join(storefront, "app", "og");
export const out = join(here, "dist");

const CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const SIPS = "/usr/bin/sips";

export const INK = "#0b0b0c";
export const PAGE = "#141416";
export const SURFACE = "#1f1f22";
export const ROYAL = "#1355b3";
export const ROYAL_LIGHT = "#2f74d8";
export const BRAND = "#f6559b";
export const BRAND_DARK = "#e23d85";
export const BRAND_SOFT = "#fdc4e5";
export const ON_SOFT = "#8a3f68";
export const MUTED = "#9a9aa2";

/* Header background tiles horizontally: the store renders it with
   background-repeat:repeat at intrinsic size, so every horizontal period has to
   divide 1920 and nothing may depend on the x position. */
export const TILE = 96;

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
    dataURI(join(og, "mascot.png"), "image/png"),
  ]);
  return { fonts: [baloo, pixel, dmMedium, dmBold].join("\n"), mascot };
}

export const baseCSS = `
*{margin:0;padding:0;box-sizing:border-box}
html,body{background:transparent}
body{
  font-family:"DM Sans",sans-serif;-webkit-font-smoothing:antialiased;
  overflow:hidden;position:relative;
}
.pixel{font-family:"Press Start 2P",monospace;font-weight:400;line-height:1}
.wordmark{
  font-family:"Baloo 2",sans-serif;font-weight:800;line-height:1;
  letter-spacing:-.01em;color:#fff;
}
.fill{position:absolute;inset:0}
.grid{background-image:repeating-linear-gradient(90deg,rgba(255,255,255,.075) 0 2px,transparent 2px ${TILE}px)}
.scan{background-image:repeating-linear-gradient(180deg,rgba(11,11,12,.11) 0 1px,transparent 1px 3px)}
.stripes-45{background-image:repeating-linear-gradient(45deg,rgba(11,11,12,.13) 0 12px,transparent 12px 24px)}
.stripes-90{background-image:repeating-linear-gradient(90deg,rgba(11,11,12,.13) 0 12px,transparent 12px 24px)}
.stripes-135{background-image:repeating-linear-gradient(135deg,rgba(11,11,12,.13) 0 12px,transparent 12px 24px)}
.disc{
  border-radius:999px;overflow:hidden;flex:none;
  border:6px solid ${INK};background:${BRAND};
}
.disc img{width:100%;height:100%;object-fit:cover;display:block}
.sticker{border:4px solid ${INK};box-shadow:5px 5px 0 ${INK};border-radius:8px}
.z{position:relative;z-index:2}
`;

export function page({ w, h, css, body, bodyStyle }) {
  return `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8">
<style>${css}
html,body{width:${w}px;height:${h}px}
body{${bodyStyle ?? ""}}
</style></head><body>${body}</body></html>`;
}

async function chrome(args) {
  await run(CHROME, [
    "--headless=new",
    "--disable-gpu",
    "--no-sandbox",
    "--hide-scrollbars",
    "--force-color-profile=srgb",
    "--force-device-scale-factor=1",
    ...args,
  ]);
}

export async function shot({ name, w, h, html, transparent }) {
  const htmlPath = join(out, `${name}.html`);
  const pngPath = join(out, `${name}.png`);
  await writeFile(htmlPath, html);
  const args = [
    `--screenshot=${pngPath}`,
    `--window-size=${w},${h}`,
    ...(transparent ? ["--default-background-color=00000000"] : []),
    `file://${htmlPath}`,
  ];
  await chrome(args);
  return pngPath;
}

export async function toJpeg({ png, name, quality = 88 }) {
  const jpg = join(out, `${name}.jpg`);
  await run(SIPS, [
    "-s", "format", "jpeg",
    "-s", "formatOptions", String(quality),
    png, "--out", jpg,
  ]);
  return jpg;
}
