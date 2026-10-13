import "server-only";

export interface SessionUser {
  id: string;
  username: string;
  displayName: string;
  isGuest: boolean;
  email: string | null;
  city: string | null;
  state: string | null;
  whatsapp: string | null;
  mainGameId: number | null;
}

export type CouponKind = "REFERRER" | "REFERRED";

export interface Coupon {
  id: string;
  code: string;
  kind: CouponKind;
  valueCents: number;
  minOrderCents: number;
  unlockedAt: string | null;
  usedAt: string | null;
  referredUsername: string | null;
}

export interface ReferralOverview {
  code: string;
  link: string;
  referrals: { id: string; username: string; createdAt: string }[];
  coupons: Coupon[];
  rewardCents: number;
  minOrderCents: number;
  max: number;
  activationCards: number;
  catalogItems: number;
}

export type CodeCheck =
  | { ok: true; ownerId: string; ownerUsername: string }
  | { ok: false; reason: string };

interface AuthResult {
  token: string;
  user: SessionUser;
}

export class CollectaError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

interface ErrorEnvelope {
  error?: { json?: { message?: string; data?: { httpStatus?: number } } };
}

interface ResultEnvelope<T> {
  result?: { data?: { json?: T } };
}

function messageOf(raw: string): string {
  try {
    const parsed: unknown = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      const first: unknown = parsed[0];
      if (first && typeof first === "object" && "message" in first && typeof first.message === "string") {
        return first.message;
      }
    }
  } catch {
    return raw;
  }
  return raw;
}

function base(): string {
  const url = process.env.COLLECTA_API;
  if (!url) {
    throw new CollectaError("conta indisponível", 503);
  }
  return url.replace(/\/+$/, "");
}

async function call<T>(path: string, init: RequestInit, token?: string): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  let res: Response;
  try {
    res = await fetch(`${base()}/trpc/${path}`, { ...init, headers, cache: "no-store" });
  } catch {
    throw new CollectaError("conta indisponível no momento", 502);
  }
  const body = (await res.json()) as ResultEnvelope<T> & ErrorEnvelope;
  if (body.error) {
    const err = body.error.json;
    throw new CollectaError(messageOf(err?.message ?? "erro"), err?.data?.httpStatus ?? res.status);
  }
  if (!res.ok || !body.result?.data) {
    throw new CollectaError("conta indisponível no momento", res.status || 502);
  }
  return body.result.data.json as T;
}

function query<T>(path: string, input?: unknown, token?: string): Promise<T> {
  const qs = input === undefined ? "" : `?input=${encodeURIComponent(JSON.stringify({ json: input }))}`;
  return call<T>(`${path}${qs}`, { method: "GET" }, token);
}

function mutation<T>(path: string, input: unknown, token?: string): Promise<T> {
  return call<T>(
    path,
    { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ json: input }) },
    token,
  );
}

export function me(token: string): Promise<SessionUser | null> {
  return query<SessionUser | null>("auth.me", undefined, token);
}

export function login(email: string, password: string): Promise<AuthResult> {
  return mutation<AuthResult>("auth.login", { email, password });
}

export function register(input: {
  email: string;
  password: string;
  displayName?: string;
  referralCode?: string;
}): Promise<AuthResult> {
  return mutation<AuthResult>("auth.register", input);
}

export function logout(token: string): Promise<{ ok: boolean }> {
  return mutation<{ ok: boolean }>("auth.logout", undefined, token);
}

export function referralMe(token: string): Promise<ReferralOverview> {
  return query<ReferralOverview>("referral.me", undefined, token);
}

export function referralCheck(code: string): Promise<CodeCheck> {
  return query<CodeCheck>("referral.check", { code });
}

export interface HandoffLine {
  sku: string;
  qty: number;
}

export interface HandoffResult {
  token: string | null;
  lines: HandoffLine[];
}

export async function exchangeHandoff(code: string): Promise<HandoffResult> {
  let res: Response;
  try {
    res = await fetch(`${base()}/store/handoff/exchange`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ code }),
      cache: "no-store",
    });
  } catch {
    throw new CollectaError("conta indisponível no momento", 502);
  }
  const body = (await res.json()) as Partial<HandoffResult> & { message?: string };
  if (!res.ok) {
    throw new CollectaError(body.message ?? "link inválido", res.status);
  }
  return { token: body.token ?? null, lines: Array.isArray(body.lines) ? body.lines : [] };
}
