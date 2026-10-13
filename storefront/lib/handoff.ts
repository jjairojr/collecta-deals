import "server-only";

import { cookies } from "next/headers";
import type { CartLine } from "@/lib/types";

export const HANDOFF_COOKIE = "collecta_handoff";
const MAX_AGE = 5 * 60;

export interface HandoffPayload {
  lines: CartLine[];
  missing: number;
}

export async function setHandoffCookie(payload: HandoffPayload): Promise<void> {
  const jar = await cookies();
  jar.set(HANDOFF_COOKIE, JSON.stringify(payload), {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: MAX_AGE,
  });
}

export async function takeHandoffCookie(): Promise<HandoffPayload | null> {
  const jar = await cookies();
  const raw = jar.get(HANDOFF_COOKIE)?.value;
  if (!raw) {
    return null;
  }
  jar.set(HANDOFF_COOKIE, "", { path: "/", maxAge: 0 });
  try {
    const parsed = JSON.parse(raw) as Partial<HandoffPayload>;
    return {
      lines: Array.isArray(parsed.lines) ? parsed.lines : [],
      missing: typeof parsed.missing === "number" ? parsed.missing : 0,
    };
  } catch {
    return null;
  }
}
