import { NO_STORE } from "@/lib/account-api";
import { takeHandoffCookie } from "@/lib/handoff";

export async function GET(): Promise<Response> {
  const payload = await takeHandoffCookie();
  return Response.json(payload ?? { lines: [], missing: 0 }, { headers: NO_STORE });
}
