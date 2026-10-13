import "server-only";

import { cookies } from "next/headers";
import { me, type SessionUser } from "@/lib/collecta";

export const SESSION_COOKIE = "collecta_session";

const MAX_AGE = 90 * 24 * 60 * 60;

export function cookieOptions() {
  return {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax" as const,
    path: "/",
    maxAge: MAX_AGE,
  };
}

export async function sessionToken(): Promise<string | null> {
  const jar = await cookies();
  return jar.get(SESSION_COOKIE)?.value ?? null;
}

export async function sessionUser(): Promise<{ token: string; user: SessionUser } | null> {
  const token = await sessionToken();
  if (!token) {
    return null;
  }
  try {
    const user = await me(token);
    if (!user || user.isGuest) {
      return null;
    }
    return { token, user };
  } catch {
    return null;
  }
}

export async function setSessionCookie(token: string): Promise<void> {
  const jar = await cookies();
  jar.set(SESSION_COOKIE, token, cookieOptions());
}

export async function clearSessionCookie(): Promise<void> {
  const jar = await cookies();
  jar.set(SESSION_COOKIE, "", { ...cookieOptions(), maxAge: 0 });
}
