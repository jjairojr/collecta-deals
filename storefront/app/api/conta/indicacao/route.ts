import { errorResponse, NO_STORE } from "@/lib/account-api";
import { referralCheck } from "@/lib/collecta";

const CODE_RE = /^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{6}$/;

export async function GET(req: Request): Promise<Response> {
  const code = (new URL(req.url).searchParams.get("code") ?? "").trim().toUpperCase();
  if (!CODE_RE.test(code)) {
    return Response.json({ ok: false, reason: "Código inválido" }, { headers: NO_STORE });
  }
  try {
    return Response.json(await referralCheck(code), { headers: NO_STORE });
  } catch (err) {
    return errorResponse(err);
  }
}
