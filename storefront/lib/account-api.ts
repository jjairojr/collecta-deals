import "server-only";

import { CollectaError, type SessionUser } from "@/lib/collecta";
import type { AccountUser } from "@/lib/account";

export function publicUser(user: SessionUser): AccountUser {
  return {
    id: user.id,
    username: user.username,
    displayName: user.displayName,
    email: user.email,
    whatsapp: user.whatsapp,
  };
}

export function errorResponse(err: unknown): Response {
  if (err instanceof CollectaError) {
    return Response.json({ error: err.message }, { status: err.status >= 400 && err.status < 600 ? err.status : 502 });
  }
  return Response.json({ error: "conta indisponível no momento" }, { status: 502 });
}

export async function readJSON(req: Request): Promise<Record<string, unknown> | null> {
  try {
    const body: unknown = await req.json();
    if (body && typeof body === "object" && !Array.isArray(body)) {
      return body as Record<string, unknown>;
    }
  } catch {
    return null;
  }
  return null;
}

export function str(body: Record<string, unknown>, key: string): string {
  const v = body[key];
  return typeof v === "string" ? v.trim() : "";
}

export const NO_STORE = { "Cache-Control": "no-store" };
