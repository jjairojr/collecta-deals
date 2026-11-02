import { redirect } from "next/navigation";
import { resolveCartLines } from "@/lib/catalog";
import { exchangeHandoff } from "@/lib/collecta";
import { setHandoffCookie } from "@/lib/handoff";
import { setSessionCookie } from "@/lib/session";

const MAX_LINES = 20;

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
  await setHandoffCookie(await resolveCartLines(result.lines.slice(0, MAX_LINES)));
  redirect("/carrinho?origem=app");
}
