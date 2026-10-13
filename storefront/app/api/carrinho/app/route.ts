import { redirect } from "next/navigation";
import { sealedToCartLine, singleToCartLine } from "@/lib/cart-lines";
import { findBySku, loadCatalog } from "@/lib/catalog";
import { exchangeHandoff, type HandoffLine } from "@/lib/collecta";
import { setHandoffCookie } from "@/lib/handoff";
import { setSessionCookie } from "@/lib/session";
import type { CartLine } from "@/lib/types";

const MAX_LINES = 20;

async function resolveLines(lines: HandoffLine[]): Promise<{ lines: CartLine[]; missing: number }> {
  const catalog = await loadCatalog();
  const resolved: CartLine[] = [];
  let missing = 0;
  for (const line of lines.slice(0, MAX_LINES)) {
    const product = findBySku(catalog, line.sku);
    if (!product || product.qty <= 0) {
      missing += 1;
      continue;
    }
    const base = product.kind === "single" ? singleToCartLine(product) : sealedToCartLine(product, product.qty);
    resolved.push({ ...base, qty: Math.min(line.qty, product.qty) });
  }
  return { lines: resolved, missing };
}

export async function GET(req: Request): Promise<Response> {
  const code = new URL(req.url).searchParams.get("code") ?? "";
  let result;
  try {
    result = await exchangeHandoff(code);
  } catch {
    redirect("/carrinho?origem=app&erro=1");
  }
  if (result.token) {
    await setSessionCookie(result.token);
  }
  await setHandoffCookie(await resolveLines(result.lines));
  redirect("/carrinho?origem=app");
}
