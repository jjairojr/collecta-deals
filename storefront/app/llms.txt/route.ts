import { loadCatalog } from "@/lib/catalog";
import { llmsLines } from "@/lib/llms";

// Curto para refletir logo o limite do frete grátis que vem do servidor.
export const revalidate = 300;

export async function GET(): Promise<Response> {
  const lines = llmsLines(await loadCatalog());
  lines.push("", `Atualizado em ${new Date().toISOString().slice(0, 10)}.`, "");
  return new Response(lines.join("\n"), {
    headers: { "Content-Type": "text/markdown; charset=utf-8" },
  });
}
