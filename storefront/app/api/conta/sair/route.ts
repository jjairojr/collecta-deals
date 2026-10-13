import { NO_STORE } from "@/lib/account-api";
import { logout } from "@/lib/collecta";
import { clearSessionCookie, sessionToken } from "@/lib/session";

export async function POST(): Promise<Response> {
  const token = await sessionToken();
  if (token) {
    await logout(token).catch(() => null);
  }
  await clearSessionCookie();
  return Response.json({ ok: true }, { headers: NO_STORE });
}
